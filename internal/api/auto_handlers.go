package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type autoRequest struct {
	Year   int64  `json:"year"`
	Model  string `json:"model"`
	Brand  string `json:"brand"`
	Color  string `json:"color"`
	Icon   string `json:"icon"`
	Motor  string `json:"motor"`
	Chasis string `json:"chasis"`
	VIN    string `json:"vin"`
	Placa  string `json:"placa"`
}

// AutoHandlers agrupa los handlers de autos.
type AutoHandlers struct {
	service *services.AutoService
}

// NewAutoHandlers crea un nuevo AutoHandlers.
func NewAutoHandlers(service *services.AutoService) *AutoHandlers {
	return &AutoHandlers{service: service}
}

// ListAutos responde con todos los autos.
func (h *AutoHandlers) ListAutos(w http.ResponseWriter, r *http.Request) {
	autos, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, autos)
}

// GetAuto responde con un auto por ID.
func (h *AutoHandlers) GetAuto(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	auto, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if auto == nil {
		respondError(w, http.StatusNotFound, "not_found", "Auto no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, auto)
}

// CreateAuto crea un nuevo auto.
func (h *AutoHandlers) CreateAuto(w http.ResponseWriter, r *http.Request) {
	var req autoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	auto, err := h.service.Create(r.Context(), req.Year, req.Model, req.Brand, req.Color, req.Icon, req.Motor, req.Chasis, req.VIN, req.Placa)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, auto)
}

// UpdateAuto actualiza un auto existente.
func (h *AutoHandlers) UpdateAuto(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req autoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	auto, err := h.service.Update(r.Context(), id, req.Year, req.Model, req.Brand, req.Color, req.Icon, req.Motor, req.Chasis, req.VIN, req.Placa)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if auto == nil {
		respondError(w, http.StatusNotFound, "not_found", "Auto no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, auto)
}

// DeleteAuto elimina un auto.
func (h *AutoHandlers) DeleteAuto(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Auto eliminado"})
}
