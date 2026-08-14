package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// CurrencyService contiene la lógica de negocio para monedas.
type CurrencyService struct {
	storage *storage.CurrencyStorage
}

// NewCurrencyService crea un nuevo CurrencyService.
func NewCurrencyService(st *storage.CurrencyStorage) *CurrencyService {
	return &CurrencyService{storage: st}
}

// List devuelve todas las monedas activas.
func (s *CurrencyService) List(ctx context.Context) ([]models.Currency, error) {
	return s.storage.List(ctx)
}

// GetByID busca una moneda por ID.
func (s *CurrencyService) GetByID(ctx context.Context, id int64) (*models.Currency, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea una nueva moneda.
func (s *CurrencyService) Create(ctx context.Context, code, name, symbol string) (*models.Currency, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	name = strings.TrimSpace(name)
	symbol = strings.TrimSpace(symbol)

	if code == "" {
		return nil, fmt.Errorf("el código de moneda es requerido")
	}
	if name == "" {
		return nil, fmt.Errorf("el nombre de moneda es requerido")
	}
	if symbol == "" {
		return nil, fmt.Errorf("el símbolo de moneda es requerido")
	}

	return s.storage.Create(ctx, code, name, symbol)
}

// Update actualiza una moneda existente.
func (s *CurrencyService) Update(ctx context.Context, id int64, code, name, symbol string) (*models.Currency, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	name = strings.TrimSpace(name)
	symbol = strings.TrimSpace(symbol)

	if code == "" {
		return nil, fmt.Errorf("el código de moneda es requerido")
	}
	if name == "" {
		return nil, fmt.Errorf("el nombre de moneda es requerido")
	}
	if symbol == "" {
		return nil, fmt.Errorf("el símbolo de moneda es requerido")
	}

	return s.storage.Update(ctx, id, code, name, symbol)
}

// Delete elimina lógicamente una moneda.
func (s *CurrencyService) Delete(ctx context.Context, id int64) error {
	return s.storage.SoftDelete(ctx, id)
}
