package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// MonthClosingHandlers agrupa los handlers de cierre de mes.
type MonthClosingHandlers struct {
	service      *services.MonthClosingService
	notification *services.PensionNotificationService
}

// NewMonthClosingHandlers crea un nuevo MonthClosingHandlers.
func NewMonthClosingHandlers(service *services.MonthClosingService, notification *services.PensionNotificationService) *MonthClosingHandlers {
	return &MonthClosingHandlers{service: service, notification: notification}
}

// GetClosingStatus responde si un mes está cerrado.
func (h *MonthClosingHandlers) GetClosingStatus(w http.ResponseWriter, r *http.Request) {
	year, month, ok := periodFromPath(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		return
	}
	closed, closedAt, err := h.service.Status(r.Context(), year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"closed":    closed,
		"closed_at": closedAt,
	})
}

// CloseMonth cierra un mes.
func (h *MonthClosingHandlers) CloseMonth(w http.ResponseWriter, r *http.Request) {
	year, month, ok := periodFromPath(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		return
	}
	closing, err := h.service.Close(r.Context(), year, month)
	if err != nil {
		respondError(w, http.StatusConflict, "conflict", err.Error())
		return
	}
	go h.notification.SendMonthClosing(context.Background(), year, month)
	respondJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"closed_at": closing.ClosedAt,
	})
}

// ReopenMonth reabre un mes.
func (h *MonthClosingHandlers) ReopenMonth(w http.ResponseWriter, r *http.Request) {
	year, month, ok := periodFromPath(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		return
	}
	if err := h.service.Reopen(r.Context(), year, month); err != nil {
		respondError(w, http.StatusConflict, "conflict", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// periodFromPath extrae year/month de la ruta (formato /pension/closing/{year}/{month}).
func periodFromPath(r *http.Request) (year, month int, ok bool) {
	y, err1 := strconv.Atoi(r.PathValue("year"))
	m, err2 := strconv.Atoi(r.PathValue("month"))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return y, m, true
}
