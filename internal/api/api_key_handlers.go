package api

import (
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// APIKeyHandlers expone la gestión de la API key global del sistema. Todos los
// endpoints requieren sesión: una key no puede administrar otras keys.
type APIKeyHandlers struct {
	service *services.APIKeyService
}

// NewAPIKeyHandlers crea los handlers de la API key global.
func NewAPIKeyHandlers(service *services.APIKeyService) *APIKeyHandlers {
	return &APIKeyHandlers{service: service}
}

// SetAPIKeyService conecta el servicio de API keys al Handler principal.
func (h *Handler) SetAPIKeyService(service *services.APIKeyService) {
	h.apiKeys = service
	h.apiKey = NewAPIKeyHandlers(service)
}

// GetAPIKey devuelve el estado de la key sin exponer el secreto.
func (h *APIKeyHandlers) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	status, err := h.service.Status(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "no se pudo leer la API key")
		return
	}
	respondJSON(w, http.StatusOK, status)
}

// RegenerateAPIKey rota la key global y devuelve el secreto una sola vez.
func (h *APIKeyHandlers) RegenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	key, status, err := h.service.Regenerate(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "no se pudo generar la API key")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"api_key":    key,
		"prefix":     status.Prefix,
		"created_at": status.CreatedAt,
	})
}

// RevokeAPIKey invalida la key global vigente.
func (h *APIKeyHandlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if err := h.service.Revoke(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "no se pudo revocar la API key")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "API key revocada"})
}
