package maintenance

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type handlers struct {
	service *Service
}

// RegisterRoutes monta los endpoints del módulo (ARCH-MODULES).
func (h *handlers) register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/homes/{id}/maintenances", auth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/homes/{id}/maintenances", auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/homes/{id}/maintenances/{maintenance_id}", auth(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/homes/{id}/maintenances/{maintenance_id}", auth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/homes/{id}/maintenances/{maintenance_id}", auth(http.HandlerFunc(h.delete)))
	mux.Handle("POST /api/homes/{id}/maintenances/{maintenance_id}/items", auth(http.HandlerFunc(h.createItem)))
	mux.Handle("POST /api/homes/{id}/maintenances/{maintenance_id}/items/move", auth(http.HandlerFunc(h.moveItems)))
	mux.Handle("PUT /api/homes/{id}/maintenances/{maintenance_id}/items/{item_id}", auth(http.HandlerFunc(h.updateItem)))
	mux.Handle("DELETE /api/homes/{id}/maintenances/{maintenance_id}/items/{item_id}", auth(http.HandlerFunc(h.deleteItem)))
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	homeID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	filter := Filter{
		Kind:     r.URL.Query().Get("kind"),
		Date:     r.URL.Query().Get("date"),
		ItemType: r.URL.Query().Get("item_type"),
		Query:    r.URL.Query().Get("q"),
	}
	if raw := r.URL.Query().Get("year"); raw != "" {
		year, err := strconv.Atoi(raw)
		if err != nil || year < 1900 || year > 9999 {
			respondError(w, http.StatusBadRequest, "invalid_year", "Año inválido")
			return
		}
		filter.Year = year
	}
	if raw := r.URL.Query().Get("include_items"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_filter", "include_items debe ser true o false")
			return
		}
		filter.IncludeItems = value
	}
	result, err := h.service.List(r.Context(), homeID, filter)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	homeID, maintenanceID, ok := h.ids(w, r)
	if !ok {
		return
	}
	item, err := h.service.Get(r.Context(), homeID, maintenanceID)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) create(w http.ResponseWriter, r *http.Request) {
	homeID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req CreateInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.CreateWithItem(r.Context(), homeID, req.Input, req.Item)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) update(w http.ResponseWriter, r *http.Request) {
	homeID, maintenanceID, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req Input
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.Update(r.Context(), homeID, maintenanceID, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) delete(w http.ResponseWriter, r *http.Request) {
	homeID, maintenanceID, ok := h.ids(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), homeID, maintenanceID); err != nil {
		respondModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) createItem(w http.ResponseWriter, r *http.Request) {
	homeID, maintenanceID, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req ItemInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.CreateItem(r.Context(), homeID, maintenanceID, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) moveItems(w http.ResponseWriter, r *http.Request) {
	homeID, maintenanceID, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req MoveInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	result, err := h.service.MoveItems(r.Context(), homeID, maintenanceID, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *handlers) updateItem(w http.ResponseWriter, r *http.Request) {
	homeID, maintenanceID, ok := h.ids(w, r)
	if !ok {
		return
	}
	itemID, err := pathID(r, "item_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req ItemInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.UpdateItem(r.Context(), homeID, maintenanceID, itemID, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) deleteItem(w http.ResponseWriter, r *http.Request) {
	homeID, maintenanceID, ok := h.ids(w, r)
	if !ok {
		return
	}
	itemID, err := pathID(r, "item_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteItem(r.Context(), homeID, maintenanceID, itemID); err != nil {
		respondModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) ids(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	homeID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return 0, 0, false
	}
	maintenanceID, err := pathID(r, "maintenance_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return 0, 0, false
	}
	return homeID, maintenanceID, true
}

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, errorBody{Error: code, Message: message})
}

func respondModuleError(w http.ResponseWriter, err error) {
	var validation *ValidationError
	if errors.As(err, &validation) {
		respondError(w, http.StatusBadRequest, validation.Code, validation.Message)
		return
	}
	switch {
	case errors.Is(err, ErrHomeNotFound):
		respondError(w, http.StatusNotFound, "home_not_found", "Vivienda no encontrada")
	case errors.Is(err, ErrMaintenanceNotFound):
		respondError(w, http.StatusNotFound, "maintenance_not_found", "Mantenimiento no encontrado")
	case errors.Is(err, ErrItemNotFound):
		respondError(w, http.StatusNotFound, "item_not_found", "Línea no encontrada")
	case errors.Is(err, ErrCurrencyNotFound):
		respondError(w, http.StatusNotFound, "currency_not_found", "Moneda no encontrada")
	case errors.Is(err, ErrInstitutionNotFound):
		respondError(w, http.StatusNotFound, "institution_not_found", "Institución no encontrada")
	case errors.Is(err, ErrMoveConflict):
		respondError(w, http.StatusConflict, "move_conflict", "Las líneas cambiaron durante el movimiento; reintentá")
	default:
		respondError(w, http.StatusInternalServerError, "internal_error", "Error interno")
	}
}
