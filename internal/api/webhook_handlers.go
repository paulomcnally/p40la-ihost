package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

func parseID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

// WebhookHandlers agrupa los handlers de webhooks y de la api_key global.
type WebhookHandlers struct {
	webhooks *services.WebhookService
	services *services.ServiceService
}

// NewWebhookHandlers crea un nuevo WebhookHandlers.
func NewWebhookHandlers(webhooks *services.WebhookService, services *services.ServiceService) *WebhookHandlers {
	return &WebhookHandlers{webhooks: webhooks, services: services}
}

// UpsertBill procesa POST /webhooks/{uuid}: crea o actualiza la factura del
// servicio identificado por el uuid (SPEC-069).
func (h *WebhookHandlers) UpsertBill(w http.ResponseWriter, r *http.Request) {
	uuid := r.PathValue("uuid")
	if uuid == "" {
		respondError(w, http.StatusBadRequest, "invalid_uuid", "uuid requerido")
		return
	}

	enabled, err := h.webhooks.IsEnabled(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if !enabled {
		respondError(w, http.StatusForbidden, "webhook_disabled", "La feature de webhooks está deshabilitada en Configuración")
		return
	}

	svc, err := h.services.FindByWebhookUUID(r.Context(), uuid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if svc == nil {
		respondError(w, http.StatusNotFound, "not_found", "No hay un servicio con ese webhook")
		return
	}

	var payload models.WebhookBillPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", "Cuerpo JSON inválido")
		return
	}

	result, err := h.webhooks.UpsertBill(r.Context(), svc, &payload)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// GetWebhookKey devuelve la api_key global de webhooks (autenticado por sesión).
func (h *WebhookHandlers) GetWebhookKey(w http.ResponseWriter, r *http.Request) {
	key, err := h.webhooks.GetOrCreateWebhookKey(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"api_key": key})
}

// RegenerateWebhookKey genera una nueva api_key global (autenticado por sesión).
func (h *WebhookHandlers) RegenerateWebhookKey(w http.ResponseWriter, r *http.Request) {
	key, err := h.webhooks.RegenerateWebhookKey(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"api_key": key})
}

// RegenerateServiceWebhookUUID reemplaza el webhook_uuid de un servicio.
func (h *WebhookHandlers) RegenerateServiceWebhookUUID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	svc, err := h.services.RegenerateWebhookUUID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if svc == nil {
		respondError(w, http.StatusNotFound, "not_found", "Servicio no encontrado")
		return
	}
	url, err := h.webhooks.WebhookURL(r.Context(), svc.WebhookUUID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"webhook_uuid": svc.WebhookUUID,
		"webhook_url":  url,
	})
}
