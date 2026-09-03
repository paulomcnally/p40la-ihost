package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type childRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	BirthDate string `json:"birth_date"`
	Notes     string `json:"notes"`
}

// ChildHandlers agrupa los handlers de hijos.
type ChildHandlers struct {
	service *services.ChildService
}

// NewChildHandlers crea un nuevo ChildHandlers.
func NewChildHandlers(service *services.ChildService) *ChildHandlers {
	return &ChildHandlers{service: service}
}

// ListChildren responde con todos los hijos.
func (h *ChildHandlers) ListChildren(w http.ResponseWriter, r *http.Request) {
	children, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, children)
}

// GetChild responde con un hijo por ID.
func (h *ChildHandlers) GetChild(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	child, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if child == nil {
		respondError(w, http.StatusNotFound, "not_found", "Hijo no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, child)
}

// CreateChild crea un nuevo hijo.
func (h *ChildHandlers) CreateChild(w http.ResponseWriter, r *http.Request) {
	var req childRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	child, err := h.service.Create(r.Context(), req.FirstName, req.LastName, req.BirthDate, req.Notes)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, child)
}

// UpdateChild actualiza un hijo existente.
func (h *ChildHandlers) UpdateChild(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req childRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	child, err := h.service.Update(r.Context(), id, req.FirstName, req.LastName, req.BirthDate, req.Notes)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if child == nil {
		respondError(w, http.StatusNotFound, "not_found", "Hijo no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, child)
}

// DeleteChild elimina un hijo.
func (h *ChildHandlers) DeleteChild(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Hijo eliminado"})
}