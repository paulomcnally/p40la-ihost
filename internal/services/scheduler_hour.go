package services

import (
	"context"
	"time"
)

// currentUserNow devuelve el instante actual expresado en la zona horaria
// configurada por el usuario (setting `timezone`, SPEC-078). Si la zona no
// está configurada o es inválida, devuelve la hora en UTC (fallback).
// Todos los schedulers la usan para comparar la hora del día programada:
// el usuario configura una hora LOCAL (ej. 07:00) y el contenedor corre en UTC.
func currentUserNow(ctx context.Context, settings *SystemSettingsService) (time.Time, error) {
	loc, err := settings.GetTimezoneLocation(ctx)
	if err != nil {
		return time.Now().In(time.UTC), err
	}
	return time.Now().In(loc), nil
}
