package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// TransferService contiene la lógica de transferencias del presupuesto
// (SPEC-099). Las transferencias son movimientos neutros entre cuentas: no
// afectan la actividad por categoría ni los ingresos, solo los balances.
type TransferService struct {
	transfers  *storage.TransferStorage
	accounts   *storage.AccountStorage
	currencies *storage.CurrencyStorage
}

// NewTransferService crea un nuevo TransferService.
func NewTransferService(
	transfers *storage.TransferStorage,
	accounts *storage.AccountStorage,
	currencies *storage.CurrencyStorage,
) *TransferService {
	return &TransferService{
		transfers:  transfers,
		accounts:   accounts,
		currencies: currencies,
	}
}

// ListByMonth devuelve las transferencias de un mes.
func (s *TransferService) ListByMonth(ctx context.Context, year, month int) ([]models.Transfer, error) {
	return s.transfers.ListByMonth(ctx, year, month)
}

// List devuelve todas las transferencias.
func (s *TransferService) List(ctx context.Context) ([]models.Transfer, error) {
	return s.transfers.List(ctx)
}

// GetByID busca una transferencia.
func (s *TransferService) GetByID(ctx context.Context, id int64) (*models.Transfer, error) {
	return s.transfers.GetByID(ctx, id)
}

// Create crea una transferencia validando cuentas, moneda y monto.
func (s *TransferService) Create(ctx context.Context, t *models.Transfer) (*models.Transfer, error) {
	if err := s.validate(ctx, t); err != nil {
		return nil, err
	}
	return s.transfers.Create(ctx, t)
}

// Update actualiza una transferencia existente.
func (s *TransferService) Update(ctx context.Context, t *models.Transfer) (*models.Transfer, error) {
	if t.ID == 0 {
		return nil, fmt.Errorf("id de transferencia requerido")
	}
	if err := s.validate(ctx, t); err != nil {
		return nil, err
	}
	return s.transfers.Update(ctx, t)
}

// Delete elimina lógicamente una transferencia.
func (s *TransferService) Delete(ctx context.Context, id int64) error {
	return s.transfers.SoftDelete(ctx, id)
}

func (s *TransferService) validate(ctx context.Context, t *models.Transfer) error {
	t.Payee = strings.TrimSpace(t.Payee)
	t.Memo = strings.TrimSpace(t.Memo)
	if t.FromAccountID == 0 {
		return fmt.Errorf("debe seleccionar la cuenta de origen")
	}
	if t.ToAccountID == 0 {
		return fmt.Errorf("debe seleccionar la cuenta de destino")
	}
	if t.FromAccountID == t.ToAccountID {
		return fmt.Errorf("la cuenta de origen y destino deben ser distintas")
	}
	if t.CurrencyID == 0 {
		return fmt.Errorf("debe seleccionar una moneda")
	}
	if t.Amount <= 0 {
		return fmt.Errorf("el monto debe ser mayor a cero")
	}
	if _, err := time.Parse("2006-01-02", t.Date); err != nil {
		return fmt.Errorf("la fecha es requerida (formato YYYY-MM-DD)")
	}

	if s.accounts != nil {
		from, err := s.accounts.GetByID(ctx, t.FromAccountID)
		if err != nil {
			return fmt.Errorf("validar cuenta de origen: %w", err)
		}
		if from == nil {
			return fmt.Errorf("la cuenta de origen no existe")
		}
		to, err := s.accounts.GetByID(ctx, t.ToAccountID)
		if err != nil {
			return fmt.Errorf("validar cuenta de destino: %w", err)
		}
		if to == nil {
			return fmt.Errorf("la cuenta de destino no existe")
		}
		if t.CurrencyID != from.CurrencyID || t.CurrencyID != to.CurrencyID {
			return fmt.Errorf("las cuentas deben usar la misma moneda para la transferencia")
		}
	}
	if s.currencies != nil {
		currency, err := s.currencies.GetByID(ctx, t.CurrencyID)
		if err != nil {
			return fmt.Errorf("validar moneda: %w", err)
		}
		if currency == nil {
			return fmt.Errorf("la moneda seleccionada no existe")
		}
	}
	return nil
}
