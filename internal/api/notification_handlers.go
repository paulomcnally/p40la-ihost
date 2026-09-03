package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type notificationRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Active bool   `json:"active"`
}

// NotificationHandlers agrupa los handlers de notificaciones.
type NotificationHandlers struct {
	service *services.NotificationService
}

// NewNotificationHandlers crea un nuevo NotificationHandlers.
func NewNotificationHandlers(service *services.NotificationService) *NotificationHandlers {
	return &NotificationHandlers{service: service}
}

// ListNotifications responde con todos los registros de notificación.
func (h *NotificationHandlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	notifications, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, notifications)
}

// GetNotification responde con un registro por ID.
func (h *NotificationHandlers) GetNotification(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	notification, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if notification == nil {
		respondError(w, http.StatusNotFound, "not_found", "Registro no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, notification)
}

// CreateNotification crea un nuevo registro de notificación.
func (h *NotificationHandlers) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req notificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	notification, err := h.service.Create(r.Context(), req.Name, req.Email, req.Active)
	if err != nil {
		if err == storage.ErrDuplicateEmail {
			respondError(w, http.StatusConflict, "duplicate_email", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, notification)
}

// UpdateNotification actualiza un registro existente.
func (h *NotificationHandlers) UpdateNotification(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req notificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	notification, err := h.service.Update(r.Context(), id, req.Name, req.Email, req.Active)
	if err != nil {
		if err == storage.ErrDuplicateEmail {
			respondError(w, http.StatusConflict, "duplicate_email", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if notification == nil {
		respondError(w, http.StatusNotFound, "not_found", "Registro no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, notification)
}

// DeleteNotification elimina un registro.
func (h *NotificationHandlers) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Registro eliminado"})
}
