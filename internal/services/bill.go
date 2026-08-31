package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

var driveURLRegex = regexp.MustCompile(`^https?://(drive\.google\.com|docs\.google\.com)/.+`)

// BillService contiene la lógica de negocio para facturas.
type BillService struct {
	storage  *storage.BillStorage
	services *storage.ServiceStorage
}

// NewBillService crea un nuevo BillService.
func NewBillService(st *storage.BillStorage, services *storage.ServiceStorage) *BillService {
	return &BillService{storage: st, services: services}
}

// ListByService devuelve las facturas de un servicio.
func (s *BillService) ListByService(ctx context.Context, serviceID int64) ([]models.Bill, error) {
	return s.storage.ListByService(ctx, serviceID)
}

// GetByID busca una factura por ID.
func (s *BillService) GetByID(ctx context.Context, id int64) (*models.Bill, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea una nueva factura.
func (s *BillService) Create(ctx context.Context, bill *models.Bill) (*models.Bill, error) {
	if err := s.validate(ctx, bill, true); err != nil {
		return nil, err
	}
	return s.storage.Create(ctx, bill)
}

// Update actualiza una factura existente.
func (s *BillService) Update(ctx context.Context, bill *models.Bill) (*models.Bill, error) {
	if bill.ID == 0 {
		return nil, fmt.Errorf("id de factura requerido")
	}
	if err := s.validate(ctx, bill, false); err != nil {
		return nil, err
	}
	return s.storage.Update(ctx, bill)
}

// Delete elimina lógicamente una factura.
func (s *BillService) Delete(ctx context.Context, id int64) error {
	return s.storage.SoftDelete(ctx, id)
}

// PayBill marca una factura como pagada (SPEC-043). Persiste la fecha de pago
// (obligatoria) y, opcionalmente, el link de Google Drive del comprobante y una
// referencia interna del pago.
func (s *BillService) PayBill(ctx context.Context, id int64, paidAt time.Time, driveURL, paymentReference string) (*models.Bill, error) {
	if paidAt.IsZero() {
		return nil, fmt.Errorf("la fecha de pago es obligatoria")
	}
	if !paidAt.Before(time.Now().Add(24 * time.Hour)) {
		return nil, fmt.Errorf("la fecha de pago no puede ser futura")
	}

	bill, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("obtener factura: %w", err)
	}
	if bill == nil {
		return nil, fmt.Errorf("la factura no existe")
	}
	if bill.Status == "paid" {
		return nil, fmt.Errorf("la factura ya está pagada")
	}

	driveURL = strings.TrimSpace(driveURL)
	if driveURL != "" && !driveURLRegex.MatchString(driveURL) {
		return nil, fmt.Errorf("el enlace de Google Drive no es válido")
	}

	return s.storage.Pay(ctx, id, paidAt, driveURL, strings.TrimSpace(paymentReference))
}

func (s *BillService) validate(ctx context.Context, bill *models.Bill, isNew bool) error {
	bill.InvoiceNumber = strings.TrimSpace(bill.InvoiceNumber)
	bill.DriveURL = strings.TrimSpace(bill.DriveURL)
	bill.Status = strings.ToLower(strings.TrimSpace(bill.Status))

	if bill.ServiceID == 0 {
		return fmt.Errorf("el servicio es requerido")
	}
	if bill.Year < 1900 || bill.Year > 2100 {
		return fmt.Errorf("año inválido")
	}

	svc, err := s.services.GetByID(ctx, bill.ServiceID)
	if err != nil {
		return fmt.Errorf("validar servicio: %w", err)
	}
	if svc == nil {
		return fmt.Errorf("el servicio no existe")
	}

	if svc.Frequency == FrequencyYearly {
		if bill.Month != 0 {
			return fmt.Errorf("los servicios anuales no requieren mes")
		}
	} else {
		if bill.Month < 1 || bill.Month > 12 {
			return fmt.Errorf("mes inválido")
		}
	}

	if bill.Amount < 0 {
		return fmt.Errorf("el monto no puede ser negativo")
	}
	if bill.Status != "pending" && bill.Status != "paid" {
		return fmt.Errorf("el estado debe ser pendiente o pagada")
	}

	if bill.Status == "paid" {
		if bill.DriveURL != "" && !driveURLRegex.MatchString(bill.DriveURL) {
			return fmt.Errorf("el enlace de Google Drive no es válido")
		}
	}

	if isNew {
		existing, err := s.storage.FindByServicePeriod(ctx, bill.ServiceID, bill.Year, bill.Month)
		if err != nil {
			return fmt.Errorf("verificar factura existente: %w", err)
		}
		if existing != nil && existing.ID != bill.ID {
			return fmt.Errorf("ya existe una factura para ese periodo")
		}
	}

	return nil
}
