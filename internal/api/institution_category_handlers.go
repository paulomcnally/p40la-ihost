package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type InstitutionCategoryHandlers struct {
	svc *services.InstitutionCategoryService
}

func NewInstitutionCategoryHandlers(svc *services.InstitutionCategoryService) *InstitutionCategoryHandlers {
	return &InstitutionCategoryHandlers{svc: svc}
}

func (h *InstitutionCategoryHandlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, list)
}

func (h *InstitutionCategoryHandlers) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	cat, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if cat == nil {
		respondError(w, http.StatusNotFound, "not_found", "Categoría no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, cat)
}

func (h *InstitutionCategoryHandlers) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IconKey     string `json:"icon_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	cat, err := h.svc.Create(r.Context(), &models.InstitutionCategory{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		IconKey:     req.IconKey,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, cat)
}

func (h *InstitutionCategoryHandlers) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IconKey     string `json:"icon_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	cat, err := h.svc.Update(r.Context(), &models.InstitutionCategory{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		IconKey:     req.IconKey,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if cat == nil {
		respondError(w, http.StatusNotFound, "not_found", "Categoría no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, cat)
}

func (h *InstitutionCategoryHandlers) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Categoría eliminada"})
}
