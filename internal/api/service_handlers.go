package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ServiceHandlers agrupa los handlers de servicios.
type ServiceHandlers struct {
	service      *services.ServiceService
	homes        *services.HomeService
	institutions *storage.InstitutionStorage
	automation   *services.AutomationClient
}

// NewServiceHandlers crea un nuevo ServiceHandlers.
func NewServiceHandlers(service *services.ServiceService, homes *services.HomeService, institutions *storage.InstitutionStorage, automation *services.AutomationClient) *ServiceHandlers {
	return &ServiceHandlers{service: service, homes: homes, institutions: institutions, automation: automation}
}

type serviceRequest struct {
	HomeID                int64   `json:"home_id"`
	Name                  string  `json:"name"`
	Institution           string  `json:"institution"`
	CurrencyID            int64   `json:"currency_id"`
	Frequency             string  `json:"frequency"`
	SuggestedAmount       float64 `json:"suggested_amount"`
	Active                bool    `json:"active"`
	IconKey               string  `json:"icon_key"`
	BillingType           string  `json:"billing_type"`
	BillingDay            *int    `json:"billing_day"`
	AutoGenerate          bool    `json:"auto_generate"`
	InstitutionID         *int64  `json:"institution_id,omitempty"`
	InstitutionAnalyzerID *int64  `json:"institution_analyzer_id,omitempty"`
	StartDate             *string `json:"start_date,omitempty"`
	EndDate               *string `json:"end_date,omitempty"`
	IsRecurring           bool    `json:"is_recurring"`
	AutomationAccountID   *int64  `json:"automation_account_id,omitempty"`
}

func (h *ServiceHandlers) toModel(req serviceRequest) *models.Service {
	return &models.Service{
		HomeID:                req.HomeID,
		Name:                  req.Name,
		Institution:           req.Institution,
		CurrencyID:            req.CurrencyID,
		Frequency:             req.Frequency,
		SuggestedAmount:       req.SuggestedAmount,
		Active:                req.Active,
		IconKey:               req.IconKey,
		BillingType:           req.BillingType,
		BillingDay:            req.BillingDay,
		AutoGenerate:          req.AutoGenerate,
		InstitutionID:         req.InstitutionID,
		InstitutionAnalyzerID: req.InstitutionAnalyzerID,
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		IsRecurring:           req.IsRecurring,
		AutomationAccountID:   req.AutomationAccountID,
	}
}

// ListServices responde con los servicios, opcionalmente filtrados por home_id.
func (h *ServiceHandlers) ListServices(w http.ResponseWriter, r *http.Request) {
	var homeID *int64
	if raw := r.URL.Query().Get("home_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_home_id", "home_id inválido")
			return
		}
		homeID = &id
	}

	if err := h.service.ReconcileBills(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	svcList, err := h.service.List(r.Context(), homeID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, svcList)
}

// GetService responde con un servicio por ID.
func (h *ServiceHandlers) GetService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	svc, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if svc == nil {
		respondError(w, http.StatusNotFound, "not_found", "Servicio no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, svc)
}

// CreateService crea un nuevo servicio.
func (h *ServiceHandlers) CreateService(w http.ResponseWriter, r *http.Request) {
	count, err := h.homes.Count(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if count == 0 {
		respondError(w, http.StatusPreconditionFailed, "NO_HOMES", "Debe crear al menos un Home/Casa antes de registrar servicios.")
		return
	}

	instCount, err := h.institutions.Count(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if instCount == 0 {
		respondError(w, http.StatusPreconditionFailed, "NO_INSTITUTIONS", "Debe crear al menos una Institución antes de registrar servicios.")
		return
	}

	var req serviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	svc, err := h.service.Create(r.Context(), h.toModel(req))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, svc)
}

// UpdateService actualiza un servicio existente.
func (h *ServiceHandlers) UpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}

	var req serviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	model := h.toModel(req)
	model.ID = id
	svc, err := h.service.Update(r.Context(), model)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if svc == nil {
		respondError(w, http.StatusNotFound, "not_found", "Servicio no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, svc)
}

// DeleteService elimina lógicamente un servicio.
func (h *ServiceHandlers) DeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Servicio eliminado"})
}

// SyncService dispara la sincronización manual del servicio (SPEC-092): ejecuta
// los plugins de p40la-ihost-automation (llamado externo) y el envío al webhook
// del servicio vía el endpoint webhook:run de automation. Requiere que el
// servicio tenga automation_account_id y que automation esté configurado.
func (h *ServiceHandlers) SyncService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}

	svc, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if svc == nil {
		respondError(w, http.StatusNotFound, "not_found", "Servicio no encontrado")
		return
	}
	if svc.AutomationAccountID == nil {
		respondError(w, http.StatusBadRequest, "no_automation_account", "Este servicio no tiene vinculada una cuenta de automation. Editá el servicio y configurá el ID de cuenta en Automation.")
		return
	}

	configured, err := h.automation.IsConfigured(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if !configured {
		respondError(w, http.StatusBadRequest, "automation_not_configured", "Configurá Automation en Settings antes de sincronizar.")
		return
	}

	delivered, failed, err := h.automation.SyncAccount(r.Context(), *svc.AutomationAccountID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAutomationNotConfigured):
			respondError(w, http.StatusBadRequest, "automation_not_configured", err.Error())
		case errors.Is(err, services.ErrAutomationUnauthorized):
			respondError(w, http.StatusUnauthorized, "automation_unauthorized", err.Error())
		case errors.Is(err, services.ErrAutomationUnreachable):
			respondError(w, http.StatusBadGateway, "automation_unreachable", err.Error())
		default:
			respondError(w, http.StatusBadGateway, "automation_error", err.Error())
		}
		return
	}
	respondJSON(w, http.StatusOK, map[string]int{"delivered": delivered, "failed": failed})
}
