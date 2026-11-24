package portfolio

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// handlers expone el resumen por HTTP.
type handlers struct {
	service *Service
}

// register monta los endpoints del módulo.
func (h *handlers) register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/rental/portfolio/summary", auth(http.HandlerFunc(h.summary)))
	mux.Handle("GET /api/rental/portfolio/homes/{id}/finances", auth(http.HandlerFunc(h.homeFinances)))
}

func (h *handlers) summary(w http.ResponseWriter, r *http.Request) {
	year, err := optionalInt(r.URL.Query().Get("year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		return
	}
	month, err := optionalInt(r.URL.Query().Get("month"))
	if err != nil || (year == 0) != (month == 0) {
		respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		return
	}
	data, err := h.service.Summary(r.Context(), year, month)
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (h *handlers) homeFinances(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	year, err := optionalInt(r.URL.Query().Get("year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		return
	}
	month, err := optionalInt(r.URL.Query().Get("month"))
	if err != nil || (year == 0) != (month == 0) {
		respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		return
	}
	data, err := h.service.HomeFinances(r.Context(), id, year, month)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPeriod):
			respondError(w, http.StatusBadRequest, "invalid_period", "Período inválido")
		case errors.Is(err, ErrHomeNotFound):
			respondError(w, http.StatusNotFound, "home_not_found", "Vivienda no encontrada")
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}
	respondJSON(w, http.StatusOK, data)
}

// optionalInt devuelve 0 cuando el valor viene vacío.
func optionalInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, errorBody{Error: code, Message: message})
}
