package services

import (
	"context"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// MonthClosingService contiene la lógica de cierre y reapertura de meses.
type MonthClosingService struct {
	storage *storage.MonthClosingStorage
}

// NewMonthClosingService crea un nuevo MonthClosingService.
func NewMonthClosingService(st *storage.MonthClosingStorage) *MonthClosingService {
	return &MonthClosingService{storage: st}
}

// IsClosed indica si un mes está cerrado.
func (s *MonthClosingService) IsClosed(ctx context.Context, year, month int) (bool, error) {
	return s.storage.IsClosed(ctx, year, month)
}

// Status devuelve el estado de cierre de un mes.
func (s *MonthClosingService) Status(ctx context.Context, year, month int) (closed bool, closedAt any, err error) {
	c, err := s.storage.Find(ctx, year, month)
	if err != nil {
		return false, nil, err
	}
	if c == nil {
		return false, nil, nil
	}
	return true, c.ClosedAt, nil
}

// Close cierra un mes. Error si ya está cerrado o el período es inválido.
func (s *MonthClosingService) Close(ctx context.Context, year, month int) (*models.MonthClosing, error) {
	if err := validatePeriod(year, month); err != nil {
		return nil, err
	}
	closed, err := s.storage.IsClosed(ctx, year, month)
	if err != nil {
		return nil, err
	}
	if closed {
		return nil, fmt.Errorf("el mes %d/%d ya está cerrado", month, year)
	}
	return s.storage.Close(ctx, year, month)
}

// Reopen reabre un mes. Error si no está cerrado.
func (s *MonthClosingService) Reopen(ctx context.Context, year, month int) error {
	closed, err := s.storage.IsClosed(ctx, year, month)
	if err != nil {
		return err
	}
	if !closed {
		return fmt.Errorf("el mes %d/%d no está cerrado", month, year)
	}
	return s.storage.Reopen(ctx, year, month)
}

func validatePeriod(year, month int) error {
	if year < 2000 || year > 2100 {
		return fmt.Errorf("el año no es válido")
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("el mes debe estar entre 1 y 12")
	}
	return nil
}
