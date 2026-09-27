package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// AccountService contiene la lógica de negocio para cuentas (SPEC-093).
type AccountService struct {
	accounts   *storage.AccountStorage
	currencies *storage.CurrencyStorage
}

// NewAccountService crea un nuevo AccountService.
func NewAccountService(accounts *storage.AccountStorage, currencies *storage.CurrencyStorage) *AccountService {
	return &AccountService{accounts: accounts, currencies: currencies}
}

// List devuelve todas las cuentas activas.
func (s *AccountService) List(ctx context.Context) ([]models.Account, error) {
	return s.accounts.List(ctx)
}

// GetByID busca una cuenta por ID.
func (s *AccountService) GetByID(ctx context.Context, id int64) (*models.Account, error) {
	return s.accounts.GetByID(ctx, id)
}

// Create crea una nueva cuenta.
func (s *AccountService) Create(ctx context.Context, account *models.Account) (*models.Account, error) {
	if err := s.validate(ctx, account); err != nil {
		return nil, err
	}
	return s.accounts.Create(ctx, account)
}

// Update actualiza una cuenta existente.
func (s *AccountService) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	if account.ID == 0 {
		return nil, fmt.Errorf("id de cuenta requerido")
	}
	if err := s.validate(ctx, account); err != nil {
		return nil, err
	}
	return s.accounts.Update(ctx, account)
}

// Delete elimina lógicamente una cuenta.
func (s *AccountService) Delete(ctx context.Context, id int64) error {
	return s.accounts.SoftDelete(ctx, id)
}

func (s *AccountService) validate(ctx context.Context, account *models.Account) error {
	account.Name = strings.TrimSpace(account.Name)
	if account.Name == "" {
		return fmt.Errorf("el nombre de la cuenta es requerido")
	}
	switch account.Type {
	case "checking", "savings", "credit_card", "cash":
	default:
		return fmt.Errorf("el tipo de cuenta debe ser checking, savings, credit_card o cash")
	}
	if account.CurrencyID == 0 {
		return fmt.Errorf("debe seleccionar una moneda")
	}
	if account.StartingBalance < 0 {
		return fmt.Errorf("el balance inicial no puede ser negativo")
	}
	currency, err := s.currencies.GetByID(ctx, account.CurrencyID)
	if err != nil {
		return fmt.Errorf("validar moneda: %w", err)
	}
	if currency == nil {
		return fmt.Errorf("la moneda seleccionada no existe")
	}
	return nil
}