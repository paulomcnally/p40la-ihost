package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AutomationHandlers expone el módulo de automation unificado (SPEC-110).
type AutomationHandlers struct {
	automation *services.AutomationService
}

func NewAutomationHandlers(automation *services.AutomationService) *AutomationHandlers {
	return &AutomationHandlers{automation: automation}
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
