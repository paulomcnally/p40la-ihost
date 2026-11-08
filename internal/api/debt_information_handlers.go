package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// DebtInformationHandlers agrupa los endpoints de plantillas de información
// y forma de pago por deuda (SPEC-131).
type DebtInformationHandlers struct {
	service *services.DebtInformationService
}

// NewDebtInformationHandlers crea un nuevo DebtInformationHandlers.
func NewDebtInformationHandlers(service *services.DebtInformationService) *DebtInformationHandlers {
	return &DebtInformationHandlers{service: service}
}

type debtInformationTemplateRequest struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Fields      []models.DebtInformationField `json:"fields"`
}

type debtPaymentInformationRequest struct {
	TemplateID int64          `json:"template_id"`
	Values     map[string]any `json:"values"`
}

// ListTemplates responde con las plantillas de información.
func (h *DebtInformationHandlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.service.ListTemplates(r.Context())
	if err != nil {
		respondDebtInformationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, templates)
}

// GetTemplate responde con una plantilla y sus campos.
func (h *DebtInformationHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	template, err := h.service.GetTemplate(r.Context(), id)
	if err != nil {
		respondDebtInformationError(w, err)
		return
	}
	if template == nil {
		respondError(w, http.StatusNotFound, "not_found", "Plantilla de información no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, template)
}

// CreateTemplate crea una plantilla de información.
func (h *DebtInformationHandlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req debtInformationTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	template, err := h.service.CreateTemplate(r.Context(), &models.DebtInformationTemplate{
		Name:        req.Name,
		Description: req.Description,
		Fields:      req.Fields,
	})
	if err != nil {
		respondDebtInformationError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, template)
}

// UpdateTemplate actualiza una plantilla de información.
func (h *DebtInformationHandlers) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req debtInformationTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	template, err := h.service.UpdateTemplate(r.Context(), &models.DebtInformationTemplate{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Fields:      req.Fields,
	})
	if err != nil {
		respondDebtInformationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, template)
}

// DeleteTemplate elimina una plantilla si no está asignada.
func (h *DebtInformationHandlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteTemplate(r.Context(), id); err != nil {
		respondDebtInformationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListTokens responde con el catálogo de tokens para campos calculados.
func (h *DebtInformationHandlers) ListTokens(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, h.service.Tokens())
}

// GetDebtPaymentInformation responde con la forma de pago de una deuda.
func (h *DebtInformationHandlers) GetDebtPaymentInformation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	info, err := h.service.GetForDebt(r.Context(), id)
	if err != nil {
		respondDebtInformationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, info)
}

// PutDebtPaymentInformation guarda la forma de pago de una deuda.
func (h *DebtInformationHandlers) PutDebtPaymentInformation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req debtPaymentInformationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	info, err := h.service.SaveForDebt(r.Context(), id, req.TemplateID, req.Values)
	if err != nil {
		respondDebtInformationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, info)
}

// DeleteDebtPaymentInformation quita la forma de pago de una deuda.
func (h *DebtInformationHandlers) DeleteDebtPaymentInformation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.ClearForDebt(r.Context(), id); err != nil {
		respondDebtInformationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func respondDebtInformationError(w http.ResponseWriter, err error) {
	var validation *services.DebtInformationValidationError
	if errors.As(err, &validation) {
		code := validation.Code
		if code == "" {
			code = "validation_error"
		}
		respondError(w, http.StatusBadRequest, code, validation.Message)
		return
	}

	switch {
	case errors.Is(err, services.ErrDebtInformationTemplateNameExists):
		respondError(w, http.StatusConflict, "template_name_exists", "Ya existe una plantilla con ese nombre")
	case errors.Is(err, services.ErrDebtInformationTemplateInUse):
		respondError(w, http.StatusConflict, "template_in_use", "La plantilla está asignada a una o más deudas")
	case errors.Is(err, services.ErrDebtInformationTemplateNotFound):
		respondError(w, http.StatusNotFound, "not_found", "Plantilla de información no encontrada")
	case errors.Is(err, services.ErrDebtInformationDebtNotFound):
		respondError(w, http.StatusNotFound, "not_found", "Deuda no encontrada")
	default:
		respondError(w, http.StatusInternalServerError, "internal_error", "Error interno")
	}
}
