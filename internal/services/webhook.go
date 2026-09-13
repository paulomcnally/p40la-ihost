package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// webhookAPIKeySetting es la clave en system_settings que guarda la api_key
// global usada para autenticar los webhooks de facturas (SPEC-069).
const webhookAPIKeySetting = "webhook_api_key"

// WebhookService encapsula la lógica de negocio de los webhooks por servicio.
type WebhookService struct {
	systemSettings *storage.SystemSettingsStorage
	settings       *SystemSettingsService
	services       *storage.ServiceStorage
	bills          *storage.BillStorage
	history        *storage.BillHistoryStorage
}

// NewWebhookService crea un nuevo WebhookService.
func NewWebhookService(
	systemSettings *storage.SystemSettingsStorage,
	settings *SystemSettingsService,
	services *storage.ServiceStorage,
	bills *storage.BillStorage,
) *WebhookService {
	return &WebhookService{
		systemSettings: systemSettings,
		settings:       settings,
		services:       services,
		bills:          bills,
	}
}

// SetBillHistoryStorage habilita el registro de auditoría de facturas
// (SPEC-070). Si no se configura, los flujos existentes no registran historial.
func (s *WebhookService) SetBillHistoryStorage(h *storage.BillHistoryStorage) {
	s.history = h
}

// IsEnabled indica si la feature de webhooks está habilitada (Settings).
func (s *WebhookService) IsEnabled(ctx context.Context) (bool, error) {
	return s.settings.GetWebhookEnabled(ctx)
}

// WebhookURL construye la URL completa del webhook de un servicio usando la
// base URL configurada en Settings (default http://ihost.local:8088). SPEC-069.
func (s *WebhookService) WebhookURL(ctx context.Context, uuid string) (string, error) {
	base, err := s.settings.GetWebhookBaseURL(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base, "/") + "/webhooks/" + uuid, nil
}

// GetOrCreateWebhookKey devuelve la api_key global de webhooks, generándola y
// persistiéndola la primera vez (REQ-011).
func (s *WebhookService) GetOrCreateWebhookKey(ctx context.Context) (string, error) {
	key, err := s.systemSettings.Get(ctx, webhookAPIKeySetting)
	if err != nil {
		return "", err
	}
	if key != "" {
		return key, nil
	}

	key, err = randomHex(32)
	if err != nil {
		return "", err
	}
	if err := s.systemSettings.Set(ctx, webhookAPIKeySetting, key); err != nil {
		return "", err
	}
	return key, nil
}

// RegenerateWebhookKey genera una nueva api_key global (invalida la anterior).
func (s *WebhookService) RegenerateWebhookKey(ctx context.Context) (string, error) {
	key, err := randomHex(32)
	if err != nil {
		return "", err
	}
	if err := s.systemSettings.Set(ctx, webhookAPIKeySetting, key); err != nil {
		return "", err
	}
	return key, nil
}

// ValidateWebhookKey compara la api_key recibida contra la global en tiempo
// constante. Una clave vacía en DB se interpreta como no configurada (se genera).
func (s *WebhookService) ValidateWebhookKey(ctx context.Context, provided string) bool {
	expected, err := s.GetOrCreateWebhookKey(ctx)
	if err != nil || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// RecordWebhookRequest registra la fecha del último request de webhook recibido
// por el servicio (SPEC-074). Se invoca cuando el UUID del webhook matchea un
// servicio, independientemente de la validez del payload: el request es tráfico
// real hacia la URL del webhook.
func (s *WebhookService) RecordWebhookRequest(ctx context.Context, serviceID int64) error {
	return s.services.SetLastWebhookRequest(ctx, serviceID, time.Now())
}

// UpsertBill crea o actualiza la factura del período (service, year, month)
// según el payload del webhook (REQ-003/REQ-004). Devuelve el resultado y si
// la factura fue creada o actualizada.
//
// Resiliencia al soft-delete (SPEC-071): si una factura del período fue borrada
// lógicamente desde la UI (deleted_at seteado), la fila sigue ocupando la clave
// UNIQUE(service_id, year, month) y el INSERT falla con SQLITE_CONSTRAINT_UNIQUE
// (2067). En ese caso se recupera la fila soft-deleted, se reactiva y se
// actualiza con el payload, en lugar de propagar el error 400.
func (s *WebhookService) UpsertBill(ctx context.Context, service *models.Service, p *models.WebhookBillPayload) (*models.WebhookResult, error) {
	if err := validateWebhookPayload(service, p); err != nil {
		return nil, err
	}

	month := p.Month
	if service.Frequency == FrequencyYearly {
		month = 0
	}

	status := strings.ToLower(strings.TrimSpace(p.Status))
	if status == "" {
		status = "pending"
	}

	existing, err := s.bills.FindByServicePeriod(ctx, service.ID, p.Year, month)
	if err != nil {
		return nil, fmt.Errorf("buscar factura existente: %w", err)
	}

	if existing == nil {
		bill := &models.Bill{
			ServiceID:     service.ID,
			Year:          p.Year,
			Month:         month,
			Amount:        p.Amount,
			InvoiceNumber: strings.TrimSpace(p.InvoiceNumber),
			DriveURL:      strings.TrimSpace(p.DriveURL),
			Status:        status,
		}
		if status == "paid" {
			// La columna paid_at se persiste vía Pay; crear primero pendiente.
			bill.Status = "pending"
		}
		created, err := s.bills.Create(ctx, bill)
		if err != nil {
			// (2067) SQLITE_CONSTRAINT_UNIQUE: la clave la ocupa una fila
			// soft-deleted del mismo período. Recuperar y reactivar (SPEC-071).
			if isUniqueConstraintError(err) {
				existing, rerr := s.recoverSoftDeletedBill(ctx, service.ID, p.Year, month)
				if rerr != nil {
					return nil, rerr
				}
				updated, uerr := s.applyWebhookUpdate(ctx, existing, status, p)
				if uerr != nil {
					return nil, uerr
				}
				return &models.WebhookResult{Bill: updated, Created: false}, nil
			}
			return nil, fmt.Errorf("crear factura: %w", err)
		}
		if status == "paid" {
			paidAt, err := parseWebhookPaidAt(p.PaidAt)
			if err != nil {
				return nil, err
			}
			before := *created
			created, err = s.bills.Pay(ctx, created.ID, paidAt, strings.TrimSpace(p.DriveURL), strings.TrimSpace(p.PaymentReference))
			if err != nil {
				return nil, fmt.Errorf("marcar factura como pagada: %w", err)
			}
			if err := recordBillHistory(ctx, s.history, created.ID, models.BillActionPaid, models.BillSourceWebhook, diffBillChanges(&before, created)); err != nil {
				return nil, fmt.Errorf("registrar historial de factura: %w", err)
			}
		}
		if err := recordBillHistory(ctx, s.history, created.ID, models.BillActionCreated, models.BillSourceWebhook, nil); err != nil {
			return nil, fmt.Errorf("registrar historial de factura: %w", err)
		}
		return &models.WebhookResult{Bill: created, Created: true}, nil
	}

	// Actualizar la factura existente. Un status "paid" explícito marca el pago;
	// un status "pending" explícito revierte la factura a pendiente.
	updated, err := s.applyWebhookUpdate(ctx, existing, status, p)
	if err != nil {
		return nil, err
	}
	return &models.WebhookResult{Bill: updated, Created: false}, nil
}

// applyWebhookUpdate aplica el payload del webhook sobre una factura existente
// (o recién reactivada): actualiza campos descriptivos y gestiona el estado
// según el status enviado. Registra el historial de auditoría (SPEC-070).
func (s *WebhookService) applyWebhookUpdate(ctx context.Context, existing *models.Bill, status string, p *models.WebhookBillPayload) (*models.Bill, error) {
	before := *existing
	var err error
	switch status {
	case "paid":
		if existing.Status != "paid" {
			// Actualizar datos descriptivos y luego marcar el pago.
			if err := s.bills.UpdateWebhookFields(ctx, existing.ID, p.Amount, strings.TrimSpace(p.InvoiceNumber), strings.TrimSpace(p.DriveURL)); err != nil {
				return nil, err
			}
			paidAt, err := parseWebhookPaidAt(p.PaidAt)
			if err != nil {
				return nil, err
			}
			existing, err = s.bills.Pay(ctx, existing.ID, paidAt, strings.TrimSpace(p.DriveURL), strings.TrimSpace(p.PaymentReference))
			if err != nil {
				return nil, fmt.Errorf("marcar factura como pagada: %w", err)
			}
			if err := recordBillHistory(ctx, s.history, existing.ID, models.BillActionPaid, models.BillSourceWebhook, diffBillChanges(&before, existing)); err != nil {
				return nil, fmt.Errorf("registrar historial de factura: %w", err)
			}
		} else {
			// Ya estaba pagada: actualizar datos descriptivos sin tocar el pago.
			if err := s.bills.UpdateWebhookFields(ctx, existing.ID, p.Amount, strings.TrimSpace(p.InvoiceNumber), strings.TrimSpace(p.DriveURL)); err != nil {
				return nil, err
			}
			existing, err = s.bills.GetByID(ctx, existing.ID)
			if err != nil {
				return nil, err
			}
			if err := recordBillHistory(ctx, s.history, existing.ID, models.BillActionUpdated, models.BillSourceWebhook, diffBillChanges(&before, existing)); err != nil {
				return nil, fmt.Errorf("registrar historial de factura: %w", err)
			}
		}
	case "pending":
		if err := s.bills.UpdateWebhookFields(ctx, existing.ID, p.Amount, strings.TrimSpace(p.InvoiceNumber), strings.TrimSpace(p.DriveURL)); err != nil {
			return nil, err
		}
		if existing.Status == "paid" {
			if err := s.bills.MarkPending(ctx, existing.ID); err != nil {
				return nil, err
			}
		}
		existing, err = s.bills.GetByID(ctx, existing.ID)
		if err != nil {
			return nil, err
		}
		if err := recordBillHistory(ctx, s.history, existing.ID, models.BillActionUpdated, models.BillSourceWebhook, diffBillChanges(&before, existing)); err != nil {
			return nil, fmt.Errorf("registrar historial de factura: %w", err)
		}
	default:
		// Status omitido: actualizar solo datos descriptivos, no tocar el estado.
		if err := s.bills.UpdateWebhookFields(ctx, existing.ID, p.Amount, strings.TrimSpace(p.InvoiceNumber), strings.TrimSpace(p.DriveURL)); err != nil {
			return nil, err
		}
		existing, err = s.bills.GetByID(ctx, existing.ID)
		if err != nil {
			return nil, err
		}
		if err := recordBillHistory(ctx, s.history, existing.ID, models.BillActionUpdated, models.BillSourceWebhook, diffBillChanges(&before, existing)); err != nil {
			return nil, fmt.Errorf("registrar historial de factura: %w", err)
		}
	}
	return existing, nil
}

// recoverSoftDeletedBill recupera la factura soft-deleted del período y la
// reactiva (SPEC-071). Devuelve error si no existe (race) o falla la query.
func (s *WebhookService) recoverSoftDeletedBill(ctx context.Context, serviceID int64, year, month int) (*models.Bill, error) {
	softDeleted, err := s.bills.FindByServicePeriodIncludingDeleted(ctx, serviceID, year, month)
	if err != nil {
		return nil, fmt.Errorf("buscar factura soft-deleted: %w", err)
	}
	if softDeleted == nil {
		return nil, fmt.Errorf("recuperar factura soft-deleted: no existe fila para el período %d/%d", year, month)
	}
	if err := s.bills.Reactivate(ctx, softDeleted.ID); err != nil {
		return nil, fmt.Errorf("reactivar factura soft-deleted: %w", err)
	}
	return s.bills.GetByID(ctx, softDeleted.ID)
}

// isUniqueConstraintError detecta un conflicto SQLITE_CONSTRAINT_UNIQUE (2067)
// usando el tipo de error de modernc.org/sqlite, con fallback por string por
// robustez ante otros drivers (SPEC-071).
func isUniqueConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return true
	}
	return strings.Contains(err.Error(), "constraint failed")
}

func validateWebhookPayload(service *models.Service, p *models.WebhookBillPayload) error {
	if p.Year < 1900 || p.Year > 2100 {
		return fmt.Errorf("year inválido")
	}
	if service.Frequency == FrequencyMonthly {
		if p.Month < 1 || p.Month > 12 {
			return fmt.Errorf("month debe estar entre 1 y 12")
		}
	}
	// Para servicios anuales el mes se ignora (la factura se identifica por año).
	if p.Amount < 0 {
		return fmt.Errorf("amount no puede ser negativo")
	}
	if status := strings.ToLower(strings.TrimSpace(p.Status)); status != "" && status != "pending" && status != "paid" {
		return fmt.Errorf("status debe ser 'pending' o 'paid'")
	}
	if drive := strings.TrimSpace(p.DriveURL); drive != "" && !driveURLRegex.MatchString(drive) {
		return fmt.Errorf("drive_url debe ser un enlace válido de Google Drive")
	}
	return nil
}

func parseWebhookPaidAt(raw string) (time.Time, error) {
	if raw == "" {
		return time.Now(), nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			if t.After(time.Now().Add(24 * time.Hour)) {
				return time.Time{}, fmt.Errorf("paid_at no puede ser futura")
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("paid_at debe ser una fecha válida (RFC3339 o YYYY-MM-DD)")
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generar token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// NewWebhookUUID genera un UUID v4 (RFC 4122) para los webhooks de servicios.
func NewWebhookUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generar uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst), nil
}
