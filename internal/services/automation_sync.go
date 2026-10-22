package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost-automation-plugins/plugins"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// Errores de dominio del sync de automation (SPEC-110 fase 3).
var (
	ErrAutomationNoPlugin         = errors.New("la cuenta no tiene plugin asociado")
	ErrAutomationNotLinked        = errors.New("la cuenta no tiene servicio vinculado")
	ErrAutomationAmbiguousService = errors.New("la cuenta tiene más de un servicio vinculado")
	ErrAutomationWebhooksDisabled = errors.New("los webhooks están deshabilitados")
)

// AutomationSyncer es el contrato que consumen los endpoints de sync y el bot.
// Lo implementan AutomationService (nuevo backend) y AutomationClient (legacy).
type AutomationSyncer interface {
	SyncAccount(ctx context.Context, accountID int64) (delivered, failed int, err error)
}

// SyncPluginCatalog registra/actualiza en SQLite los plugins compilados.
func (s *AutomationService) SyncPluginCatalog(ctx context.Context) error {
	if s.plugins == nil {
		return nil
	}
	for _, p := range s.plugins.List() {
		if err := s.storage.UpsertPlugin(ctx, p.Name(), p.Version(), p.Source(), p.Description(), "active"); err != nil {
			return err
		}
	}
	return nil
}

// PluginSchema devuelve los campos de credenciales de un plugin compilado.
func (s *AutomationService) PluginSchema(ctx context.Context, name string) ([]plugins.CredentialField, error) {
	if s.plugins == nil {
		return nil, errors.New("registry de plugins no configurado")
	}
	p, ok := s.plugins.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin %q no registrado", name)
	}
	return p.CredentialSchema(), nil
}

// FetchBills consulta el plugin de la cuenta y persiste el resultado en
// automation_bills. Devuelve las facturas obtenidas.
func (s *AutomationService) FetchBills(ctx context.Context, accountID int64) ([]plugins.Bill, error) {
	account, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.PluginName == nil || strings.TrimSpace(*account.PluginName) == "" {
		return nil, ErrAutomationNoPlugin
	}
	if s.plugins == nil {
		return nil, errors.New("registry de plugins no configurado")
	}
	plugin, ok := s.plugins.Get(*account.PluginName)
	if !ok {
		return nil, fmt.Errorf("plugin %q no registrado", *account.PluginName)
	}

	credentialsJSON, err := s.storage.CredentialsForAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	var credentials map[string]string
	if credentialsJSON != "" {
		if err := json.Unmarshal([]byte(credentialsJSON), &credentials); err != nil {
			return nil, fmt.Errorf("credenciales JSON inválidas: %w", err)
		}
	}

	bills, err := plugin.FetchBills(ctx, credentials, account.Identifier)
	if err != nil {
		s.persistFetchedBill(ctx, account, plugin, "error", err.Error(), "{}")
		return nil, err
	}
	for _, bill := range bills {
		raw, _ := json.Marshal(bill.Raw)
		s.persistFetchedBill(ctx, account, plugin, "ok", "", string(raw))
	}
	return bills, nil
}

func (s *AutomationService) persistFetchedBill(ctx context.Context, account *models.AutomationAccount, plugin plugins.Plugin, status, errText, raw string) {
	bill := &models.AutomationBill{
		AccountID:     account.ID,
		PluginName:    plugin.Name(),
		PluginVersion: plugin.Version(),
		Source:        plugin.Source(),
		Status:        status,
		Raw:           raw,
	}
	if errText != "" {
		bill.Error = &errText
	}
	if err := s.storage.CreateBill(ctx, bill); err != nil {
		slog.Error("automation: persistir factura consultada", "account_id", account.ID, "error", err)
	}
}

// SyncAccount consulta el plugin de la cuenta y entrega cada factura al
// servicio vinculado, en proceso (sin POST HTTP interno).
func (s *AutomationService) SyncAccount(ctx context.Context, accountID int64) (delivered, failed int, err error) {
	account, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return 0, 0, err
	}
	if s.webhook == nil || s.serviceStorage == nil || s.plugins == nil {
		return 0, 0, errors.New("sync de automation no configurado")
	}

	enabled, err := s.webhook.IsEnabled(ctx)
	if err != nil {
		return 0, 0, err
	}
	if !enabled {
		return 0, 1, ErrAutomationWebhooksDisabled
	}

	service, err := s.resolveServiceForAccount(ctx, account)
	if err != nil {
		return 0, 0, err
	}

	bills, err := s.FetchBills(ctx, accountID)
	if err != nil {
		s.logDelivery(ctx, account, "error", "{}", "error consultando facturas: "+err.Error())
		return 0, 1, err
	}

	for _, bill := range bills {
		payload, perr := buildAutomationPayload(bill)
		if perr != nil {
			failed++
			s.logDelivery(ctx, account, "error", "{}", perr.Error())
			continue
		}
		payloadJSON, _ := json.Marshal(payload)
		if _, uerr := s.webhook.UpsertBill(ctx, service, payload); uerr != nil {
			failed++
			s.logDelivery(ctx, account, "error", string(payloadJSON), uerr.Error())
			continue
		}
		delivered++
		s.logDelivery(ctx, account, "ok", string(payloadJSON), "upsert en proceso")
	}

	if err := s.storage.SetLastRun(ctx, accountID); err != nil {
		slog.Error("automation: actualizar last_run_at", "account_id", accountID, "error", err)
	}
	return delivered, failed, nil
}

// resolveServiceForAccount prefiere el vínculo explícito
// services.automation_account_id y cae al uuid del webhook_url como fallback
// de transición (SPEC-110 ADR-005).
func (s *AutomationService) resolveServiceForAccount(ctx context.Context, account *models.AutomationAccount) (*models.Service, error) {
	services, err := s.serviceStorage.ListByAutomationAccountID(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	switch len(services) {
	case 1:
		return &services[0], nil
	case 0:
		// fallback
	default:
		return nil, fmt.Errorf("%w: cuenta %d, servicios %d", ErrAutomationAmbiguousService, account.ID, len(services))
	}

	if account.WebhookURL == nil || strings.TrimSpace(*account.WebhookURL) == "" {
		return nil, ErrAutomationNotLinked
	}
	parsed, err := url.Parse(*account.WebhookURL)
	if err != nil {
		return nil, ErrAutomationNotLinked
	}
	uuid := path.Base(parsed.Path)
	if uuid == "" || uuid == "/" || uuid == "." {
		return nil, ErrAutomationNotLinked
	}
	service, err := s.serviceStorage.FindByWebhookUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, ErrAutomationNotLinked
	}
	return service, nil
}

func (s *AutomationService) logDelivery(ctx context.Context, account *models.AutomationAccount, status, payload, response string) {
	webhookURL := "internal"
	if account.WebhookURL != nil && *account.WebhookURL != "" {
		webhookURL = *account.WebhookURL
	}
	if err := s.storage.CreateWebhookLog(ctx, &models.AutomationWebhookLog{
		AccountID:  account.ID,
		WebhookURL: webhookURL,
		Status:     status,
		Payload:    payload,
		Response:   response,
	}); err != nil {
		slog.Error("automation: registrar entrega", "account_id", account.ID, "error", err)
	}
}

// buildAutomationPayload adapta una factura del plugin al contrato del
// WebhookService de p40la (SPEC-069).
func buildAutomationPayload(bill plugins.Bill) (*models.WebhookBillPayload, error) {
	year, month, err := parseAutomationPeriod(bill.Period)
	if err != nil {
		return nil, fmt.Errorf("periodo inválido %q: %w", bill.Period, err)
	}
	amount, _ := strconv.ParseFloat(strings.TrimSpace(bill.Amount), 64)

	status := strings.TrimSpace(bill.Status)
	if status == "" {
		status = "pending"
	}

	payload := &models.WebhookBillPayload{
		Year:          year,
		Month:         month,
		Amount:        amount,
		Status:        status,
		InvoiceNumber: extractAutomationInvoice(bill.Raw),
		IssueDate:     isoDateOnly(bill.IssueDate),
		DueDate:       isoDateOnly(bill.DueDate),
	}
	return payload, nil
}

// parseAutomationPeriod acepta "DD-MM-YYYY" (Claro) y "YYYY-MM" (Tigo).
func parseAutomationPeriod(period string) (int, int, error) {
	period = strings.TrimSpace(period)
	if t, err := time.Parse("02-01-2006", period); err == nil {
		return t.Year(), int(t.Month()), nil
	}
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return 0, 0, err
	}
	return t.Year(), int(t.Month()), nil
}

func isoDateOnly(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return ""
	}
	if t.Year() < 1900 || t.Year() > 2100 {
		return ""
	}
	return t.Format("2006-01-02")
}

func extractAutomationInvoice(raw map[string]any) string {
	for _, key := range []string{"invoice_number", "numFactura", "invoiceId"} {
		if value, ok := raw[key]; ok && value != nil {
			if scalar := automationScalar(value); scalar != "" {
				return scalar
			}
		}
	}
	return ""
}

func automationScalar(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case map[string]any:
		if inner, ok := v["value"]; ok && inner != nil {
			if scalar := automationScalar(inner); scalar != "" {
				return scalar
			}
		}
		if formatted, ok := v["formattedValue"]; ok && formatted != nil {
			if scalar := automationScalar(formatted); scalar != "" {
				return scalar
			}
		}
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
