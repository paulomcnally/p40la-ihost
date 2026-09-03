package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type pensionCategoryRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	AutoGenerate bool   `json:"auto_generate"`
}

// PensionCategoryHandlers agrupa los handlers de categorías de pensión.
type PensionCategoryHandlers struct {
	service *services.PensionCategoryService
}

// NewPensionCategoryHandlers crea un nuevo PensionCategoryHandlers.
func NewPensionCategoryHandlers(service *services.PensionCategoryService) *PensionCategoryHandlers {
	return &PensionCategoryHandlers{service: service}
}

// ListCategories responde con todas las categorías de pensión.
func (h *PensionCategoryHandlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, categories)
}

// GetCategory responde con una categoría de pensión por ID.
func (h *PensionCategoryHandlers) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	category, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if category == nil {
		respondError(w, http.StatusNotFound, "not_found", "Categoría no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, category)
}

// CreateCategory crea una nueva categoría de pensión.
func (h *PensionCategoryHandlers) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req pensionCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	category, err := h.service.Create(r.Context(), req.Name, req.Description, req.AutoGenerate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, category)
}

// UpdateCategory actualiza una categoría de pensión existente.
func (h *PensionCategoryHandlers) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req pensionCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	category, err := h.service.Update(r.Context(), id, req.Name, req.Description, req.AutoGenerate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if category == nil {
		respondError(w, http.StatusNotFound, "not_found", "Categoría no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, category)
}

// DeleteCategory elimina una categoría de pensión.
func (h *PensionCategoryHandlers) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Categoría eliminada"})
}