package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type homeRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// HomeHandlers agrupa los handlers de hogares.
type HomeHandlers struct {
	service *services.HomeService
}

// NewHomeHandlers crea un nuevo HomeHandlers.
func NewHomeHandlers(service *services.HomeService) *HomeHandlers {
	return &HomeHandlers{service: service}
}

// ListHomes responde con todos los hogares.
func (h *HomeHandlers) ListHomes(w http.ResponseWriter, r *http.Request) {
	homes, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, homes)
}

// GetHome responde con un hogar por ID.
func (h *HomeHandlers) GetHome(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	home, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if home == nil {
		respondError(w, http.StatusNotFound, "not_found", "Hogar no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, home)
}

// CreateHome crea un nuevo hogar.
func (h *HomeHandlers) CreateHome(w http.ResponseWriter, r *http.Request) {
	var req homeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	home, err := h.service.Create(r.Context(), req.Name, req.Address)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, home)
}

// UpdateHome actualiza un hogar existente.
func (h *HomeHandlers) UpdateHome(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req homeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	home, err := h.service.Update(r.Context(), id, req.Name, req.Address)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if home == nil {
		respondError(w, http.StatusNotFound, "not_found", "Hogar no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, home)
}

// DeleteHome elimina lógicamente un hogar.
func (h *HomeHandlers) DeleteHome(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Hogar eliminado"})
}

// HomeCountResponse expone la cantidad de hogares.
type HomeCountResponse struct {
	Count int64 `json:"count"`
}

// CountHomes responde con la cantidad de hogares activos.
func (h *HomeHandlers) CountHomes(w http.ResponseWriter, r *http.Request) {
	count, err := h.service.Count(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, HomeCountResponse{Count: count})
}
