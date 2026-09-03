package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// SalaryService contiene la lógica de negocio para salarios.
type SalaryService struct {
	storage         *storage.SalaryStorage
	currencyStorage *storage.CurrencyStorage
}

// NewSalaryService crea un nuevo SalaryService.
func NewSalaryService(st *storage.SalaryStorage, currencyStorage *storage.CurrencyStorage) *SalaryService {
	return &SalaryService{storage: st, currencyStorage: currencyStorage}
}

// List devuelve todos los salarios.
func (s *SalaryService) List(ctx context.Context) ([]models.Salary, error) {
	return s.storage.List(ctx)
}

// GetByID busca un salario por ID.
func (s *SalaryService) GetByID(ctx context.Context, id int64) (*models.Salary, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea un nuevo salario.
func (s *SalaryService) Create(ctx context.Context, employer string, amount float64, currencyID int64, paymentDay int, active bool, note string) (*models.Salary, error) {
	if err := s.validate(employer, amount, currencyID, paymentDay); err != nil {
		return nil, err
	}
	return s.storage.Create(ctx, strings.TrimSpace(employer), amount, currencyID, paymentDay, active, strings.TrimSpace(note))
}

// Update actualiza un salario existente.
func (s *SalaryService) Update(ctx context.Context, id int64, employer string, amount float64, currencyID int64, paymentDay int, active bool, note string) (*models.Salary, error) {
	if err := s.validate(employer, amount, currencyID, paymentDay); err != nil {
		return nil, err
	}
	return s.storage.Update(ctx, id, strings.TrimSpace(employer), amount, currencyID, paymentDay, active, strings.TrimSpace(note))
}

// Delete elimina un salario.
func (s *SalaryService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}

func (s *SalaryService) validate(employer string, amount float64, currencyID int64, paymentDay int) error {
	if strings.TrimSpace(employer) == "" {
		return fmt.Errorf("el empleador es requerido")
	}
	if amount <= 0 {
		return fmt.Errorf("el monto debe ser mayor a cero")
	}
	if paymentDay < 1 || paymentDay > 31 {
		return fmt.Errorf("el día de pago debe estar entre 1 y 31")
	}
	if currencyID <= 0 {
		return fmt.Errorf("la moneda es requerida")
	}
	currency, err := s.currencyStorage.GetByID(context.Background(), currencyID)
	if err != nil {
		return fmt.Errorf("error al validar la moneda")
	}
	if currency == nil {
		return fmt.Errorf("la moneda no existe")
	}
	return nil
}