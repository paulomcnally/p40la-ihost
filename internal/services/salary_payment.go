package services

import (
	"context"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// SalaryPaymentService contiene la lógica de negocio de pagos de salario.
type SalaryPaymentService struct {
	storage  *storage.SalaryPaymentStorage
	closings *storage.MonthClosingStorage
}

// NewSalaryPaymentService crea un nuevo SalaryPaymentService.
func NewSalaryPaymentService(st *storage.SalaryPaymentStorage, closings *storage.MonthClosingStorage) *SalaryPaymentService {
	return &SalaryPaymentService{storage: st, closings: closings}
}

// List devuelve los pagos de salario de un período.
func (s *SalaryPaymentService) List(ctx context.Context, year, month int, salaryID int64) ([]models.SalaryPayment, error) {
	return s.storage.ListByFilters(ctx, year, month, salaryID)
}

// GetByID busca un pago por ID.
func (s *SalaryPaymentService) GetByID(ctx context.Context, id int64) (*models.SalaryPayment, error) {
	return s.storage.GetByID(ctx, id)
}

// MarkReceived marca un pago como recibido. receivedAt es requerido.
func (s *SalaryPaymentService) MarkReceived(ctx context.Context, id int64, receivedAt time.Time, receivedAmount *float64, notes *string) (*models.SalaryPayment, error) {
	if receivedAt.IsZero() {
		return nil, fmt.Errorf("la fecha de recepción es requerida")
	}
	year, month, err := s.storage.GetYearMonth(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("pago de salario no encontrado")
	}
	if err := s.checkNotClosed(ctx, year, month); err != nil {
		return nil, err
	}
	return s.storage.MarkReceived(ctx, id, receivedAt, receivedAmount, notes)
}

// MarkPending devuelve un pago a estado pendiente.
func (s *SalaryPaymentService) MarkPending(ctx context.Context, id int64) (*models.SalaryPayment, error) {
	year, month, err := s.storage.GetYearMonth(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("pago de salario no encontrado")
	}
	if err := s.checkNotClosed(ctx, year, month); err != nil {
		return nil, err
	}
	return s.storage.MarkPending(ctx, id)
}

// Exists verifica si un salario ya tiene pago en el período.
func (s *SalaryPaymentService) Exists(ctx context.Context, salaryID int64, year, month int) (bool, error) {
	return s.storage.Exists(ctx, salaryID, year, month)
}

func (s *SalaryPaymentService) checkNotClosed(ctx context.Context, year, month int) error {
	closed, err := s.closings.IsClosed(ctx, year, month)
	if err != nil {
		return err
	}
	if closed {
		return ErrMonthClosed
	}
	return nil
}
