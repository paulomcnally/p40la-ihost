package services

import (
	"errors"
	"fmt"
	"time"
)

// Errores centinela del módulo de mantenimientos e inspecciones (SPEC-124).
var (
	ErrAutoNotFound            = errors.New("el auto no existe")
	ErrMaintenanceNotFound     = errors.New("el mantenimiento no existe")
	ErrMaintenanceItemNotFound = errors.New("el concepto del mantenimiento no existe")
	ErrInspectionNotFound      = errors.New("la inspección no existe")
	ErrInstitutionCategory     = errors.New("la institución no pertenece a la categoría requerida")
	ErrInstitutionNotFound     = errors.New("la institución no existe")
	ErrCurrencyNotFound        = errors.New("la moneda no existe")
	ErrAttachmentNotFound      = errors.New("el adjunto no existe")
	ErrAttachmentInvalid       = errors.New("archivo inválido")
	ErrAttachmentTooLarge      = errors.New("el archivo supera el máximo de 10 MB")
	ErrAttachmentLimit         = errors.New("se alcanzó el máximo de adjuntos por registro")
	ErrAttachmentOwnerAbsent   = errors.New("el registro asociado no existe")
)

// parseDateOnly valida una fecha YYYY-MM-DD.
func parseDateOnly(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("la fecha debe tener formato YYYY-MM-DD")
	}
	return parsed, nil
}

// dateOnly normaliza una fecha a medianoche UTC para comparar solo el día.
func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

// isFutureDate indica si la fecha es posterior al día de hoy.
func isFutureDate(value, now time.Time) bool {
	return dateOnly(value).After(dateOnly(now))
}
