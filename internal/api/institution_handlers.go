package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type InstitutionHandlers struct {
	inst *services.InstitutionService
	doc  *services.DocumentService
}

func NewInstitutionHandlers(inst *services.InstitutionService, doc *services.DocumentService) *InstitutionHandlers {
	return &InstitutionHandlers{inst: inst, doc: doc}
}

func (h *InstitutionHandlers) ListInstitutions(w http.ResponseWriter, r *http.Request) {
	list, err := h.inst.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, list)
}

func (h *InstitutionHandlers) GetInstitution(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	inst, err := h.inst.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if inst == nil {
		respondError(w, http.StatusNotFound, "not_found", "Institución no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, inst)
}

func (h *InstitutionHandlers) CreateInstitution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	inst, err := h.inst.Create(r.Context(), &models.Institution{Name: req.Name})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, inst)
}

func (h *InstitutionHandlers) UpdateInstitution(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	inst, err := h.inst.Update(r.Context(), &models.Institution{ID: id, Name: req.Name})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if inst == nil {
		respondError(w, http.StatusNotFound, "not_found", "Institución no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, inst)
}

func (h *InstitutionHandlers) DeleteInstitution(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.inst.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Institución eliminada"})
}

func (h *InstitutionHandlers) SetAnalyzers(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req struct {
		AnalyzerIDs []string `json:"analyzer_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	if err := h.inst.SetAnalyzers(r.Context(), id, req.AnalyzerIDs); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Analyzers actualizados"})
}

func (h *InstitutionHandlers) GetAnalyzers(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	list, err := h.inst.GetAnalyzers(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, list)
}

func (h *InstitutionHandlers) ListAnalyzers(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, h.doc.GetAvailableAnalyzers())
}

func (h *InstitutionHandlers) GetAnalyzerOptions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	options, err := h.doc.GetAnalyzerOptions(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, options)
}
