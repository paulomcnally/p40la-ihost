package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AutomationHandlers expone el módulo de automation unificado (SPEC-110).
type AutomationHandlers struct {
	automation *services.AutomationService
	auth       *services.AuthService
}

func NewAutomationHandlers(automation *services.AutomationService, auth *services.AuthService) *AutomationHandlers {
	return &AutomationHandlers{automation: automation, auth: auth}
}

// ListAccounts responde las cuentas con institución/plugin/schedule.
// Con ?for=service devuelve solo las opciones que consume el selector.
func (h *AutomationHandlers) ListAccounts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("for") == "service" {
		options, err := h.automation.ListAccountOptions(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		respondJSON(w, http.StatusOK, options)
		return
	}
	accounts, err := h.automation.ListAccounts(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, accounts)
}

func (h *AutomationHandlers) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	account, err := h.automation.GetAccount(r.Context(), id)
	if errors.Is(err, services.ErrAutomationAccountNotFound) {
		respondError(w, http.StatusNotFound, "not_found", "Cuenta de automation no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, account)
}

func (h *AutomationHandlers) ListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := h.automation.ListPlugins(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, plugins)
}

func (h *AutomationHandlers) ListBills(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	bills, err := h.automation.ListBills(r.Context(), accountID)
	if errors.Is(err, services.ErrAutomationAccountNotFound) {
		respondError(w, http.StatusNotFound, "not_found", "Cuenta de automation no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, bills)
}

func (h *AutomationHandlers) ListWebhookLogs(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			offset = parsed
		}
	}
	logs, err := h.automation.ListWebhookLogs(r.Context(), accountID, limit, offset)
	if errors.Is(err, services.ErrAutomationAccountNotFound) {
		respondError(w, http.StatusNotFound, "not_found", "Cuenta de automation no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, logs)
}

type automationAccountRequest struct {
	InstitutionID     int64   `json:"institution_id"`
	Identifier        string  `json:"identifier"`
	Label             string  `json:"label"`
	Credentials       string  `json:"credentials"`
	PluginName        *string `json:"plugin_name"`
	WebhookURL        *string `json:"webhook_url"`
	ScheduleTime      *string `json:"schedule_time"`
	ScheduleEnabled   bool    `json:"schedule_enabled"`
	ScheduleFrequency string  `json:"schedule_frequency"`
	ScheduleInterval  *int    `json:"schedule_interval"`
	ScheduleDays      *string `json:"schedule_days"`
	ServiceID         *int64  `json:"service_id"`
}

type automationWebhookRequest struct {
	WebhookURL        *string `json:"webhook_url"`
	ScheduleTime      *string `json:"schedule_time"`
	ScheduleEnabled   bool    `json:"schedule_enabled"`
	ScheduleFrequency string  `json:"schedule_frequency"`
	ScheduleInterval  *int    `json:"schedule_interval"`
	ScheduleDays      *string `json:"schedule_days"`
}

func respondAutomationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrAutomationAccountNotFound):
		respondError(w, http.StatusNotFound, "not_found", "Cuenta de automation no encontrada")
	case errors.Is(err, services.ErrAutomationAccountInUse):
		respondError(w, http.StatusConflict, "account_in_use", err.Error())
	case errors.Is(err, services.ErrAutomationAccountExists):
		respondError(w, http.StatusConflict, "account_exists", err.Error())
	default:
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}

func (h *AutomationHandlers) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req automationAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	account, err := h.automation.CreateAccount(r.Context(), accountInput(req))
	if err != nil {
		respondAutomationError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, account)
}

func (h *AutomationHandlers) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req automationAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	account, err := h.automation.UpdateAccount(r.Context(), id, accountInput(req))
	if err != nil {
		respondAutomationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, account)
}

func (h *AutomationHandlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.automation.DeleteAccount(r.Context(), id); err != nil {
		respondAutomationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Cuenta eliminada"})
}

func (h *AutomationHandlers) RevealCredentials(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	user, ok := userFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "Sesión no válida")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		respondError(w, http.StatusBadRequest, "invalid_request", "Password requerido")
		return
	}
	if !h.auth.ValidatePassword(r.Context(), user.Email, req.Password) {
		respondError(w, http.StatusUnauthorized, "invalid_credentials", "Password incorrecto")
		return
	}
	credentials, err := h.automation.CredentialsForAccount(r.Context(), id)
	if err != nil {
		respondAutomationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"credentials": credentials})
}

func (h *AutomationHandlers) GetAccountWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	account, err := h.automation.GetAccount(r.Context(), id)
	if err != nil {
		respondAutomationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, account)
}

func (h *AutomationHandlers) UpdateAccountWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req automationWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	account, err := h.automation.UpdateAccountWebhook(r.Context(), id, services.AutomationWebhookInput{
		WebhookURL:        req.WebhookURL,
		ScheduleTime:      req.ScheduleTime,
		ScheduleEnabled:   req.ScheduleEnabled,
		ScheduleFrequency: req.ScheduleFrequency,
		ScheduleInterval:  req.ScheduleInterval,
		ScheduleDays:      req.ScheduleDays,
	})
	if err != nil {
		respondAutomationError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, account)
}

func accountInput(req automationAccountRequest) services.AutomationAccountInput {
	return services.AutomationAccountInput{
		InstitutionID:     req.InstitutionID,
		Identifier:        req.Identifier,
		Label:             req.Label,
		Credentials:       req.Credentials,
		PluginName:        req.PluginName,
		WebhookURL:        req.WebhookURL,
		ScheduleTime:      req.ScheduleTime,
		ScheduleEnabled:   req.ScheduleEnabled,
		ScheduleFrequency: req.ScheduleFrequency,
		ScheduleInterval:  req.ScheduleInterval,
		ScheduleDays:      req.ScheduleDays,
		ServiceID:         req.ServiceID,
	}
}

// GetPluginSchema devuelve los campos de credenciales de un plugin compilado.
func (h *AutomationHandlers) GetPluginSchema(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	fields, err := h.automation.PluginSchema(r.Context(), name)
	if err != nil {
		respondError(w, http.StatusNotFound, "plugin_not_found", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, fields)
}

// FetchBills consulta el plugin de una cuenta manualmente y devuelve las
// facturas obtenidas (además de persistirlas en automation_bills).
func (h *AutomationHandlers) FetchBills(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	bills, err := h.automation.FetchBills(r.Context(), id)
	if errors.Is(err, services.ErrAutomationAccountNotFound) {
		respondError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	if errors.Is(err, services.ErrAutomationNoPlugin) {
		respondError(w, http.StatusBadRequest, "no_plugin", err.Error())
		return
	}
	if err != nil {
		respondError(w, http.StatusBadGateway, "plugin_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"bills": bills})
}
