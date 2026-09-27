package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// BudgetTransactionService contiene la lógica de transacciones del presupuesto
// (SPEC-093).
type BudgetTransactionService struct {
	transactions *storage.TransactionStorage
	accounts     *storage.AccountStorage
	categories   *storage.CategoryStorage
	currencies   *storage.CurrencyStorage
}

// NewBudgetTransactionService crea un nuevo BudgetTransactionService.
func NewBudgetTransactionService(
	transactions *storage.TransactionStorage,
	accounts *storage.AccountStorage,
	categories *storage.CategoryStorage,
	currencies *storage.CurrencyStorage,
) *BudgetTransactionService {
	return &BudgetTransactionService{
		transactions: transactions,
		accounts:     accounts,
		categories:   categories,
		currencies:   currencies,
	}
}

// ListByMonth devuelve las transacciones de un mes.
func (s *BudgetTransactionService) ListByMonth(ctx context.Context, year, month int) ([]models.Transaction, error) {
	return s.transactions.ListByMonth(ctx, year, month)
}

// ListByCategoryMonth devuelve el historial de una categoría en un mes.
func (s *BudgetTransactionService) ListByCategoryMonth(ctx context.Context, categoryID int64, year, month int) ([]models.Transaction, error) {
	return s.transactions.ListByCategoryMonth(ctx, categoryID, year, month)
}

// GetByID busca una transacción.
func (s *BudgetTransactionService) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	return s.transactions.GetByID(ctx, id)
}

// Create crea una transacción validando cuenta, categoría y moneda.
func (s *BudgetTransactionService) Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	if err := s.validate(ctx, tx); err != nil {
		return nil, err
	}
	return s.transactions.Create(ctx, tx)
}

// Update actualiza una transacción existente.
func (s *BudgetTransactionService) Update(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	if tx.ID == 0 {
		return nil, fmt.Errorf("id de transacción requerido")
	}
	if err := s.validate(ctx, tx); err != nil {
		return nil, err
	}
	return s.transactions.Update(ctx, tx)
}

// Delete elimina lógicamente una transacción.
func (s *BudgetTransactionService) Delete(ctx context.Context, id int64) error {
	return s.transactions.SoftDelete(ctx, id)
}

func (s *BudgetTransactionService) validate(ctx context.Context, tx *models.Transaction) error {
	tx.Payee = strings.TrimSpace(tx.Payee)
	tx.Memo = strings.TrimSpace(tx.Memo)
	if tx.AccountID == 0 {
		return fmt.Errorf("debe seleccionar una cuenta")
	}
	if tx.CurrencyID == 0 {
		return fmt.Errorf("debe seleccionar una moneda")
	}
	if _, err := time.Parse("2006-01-02", tx.Date); err != nil {
		return fmt.Errorf("la fecha es requerida (formato YYYY-MM-DD)")
	}
	if tx.Outflow < 0 || tx.Inflow < 0 {
		return fmt.Errorf("los montos no pueden ser negativos")
	}
	if tx.Outflow == 0 && tx.Inflow == 0 {
		return fmt.Errorf("debe indicar un outflow o inflow mayor a cero")
	}
	if tx.Outflow > 0 && tx.Inflow > 0 {
		return fmt.Errorf("una transacción no puede tener outflow e inflow a la vez")
	}

	if s.accounts != nil {
		account, err := s.accounts.GetByID(ctx, tx.AccountID)
		if err != nil {
			return fmt.Errorf("validar cuenta: %w", err)
		}
		if account == nil {
			return fmt.Errorf("la cuenta seleccionada no existe")
		}
	}
	if s.categories != nil && tx.CategoryID != nil && *tx.CategoryID != 0 {
		cat, err := s.categories.GetByID(ctx, *tx.CategoryID)
		if err != nil {
			return fmt.Errorf("validar categoría: %w", err)
		}
		if cat == nil {
			return fmt.Errorf("la categoría seleccionada no existe")
		}
	}
	if s.currencies != nil {
		currency, err := s.currencies.GetByID(ctx, tx.CurrencyID)
		if err != nil {
			return fmt.Errorf("validar moneda: %w", err)
		}
		if currency == nil {
			return fmt.Errorf("la moneda seleccionada no existe")
		}
	}
	return nil
}