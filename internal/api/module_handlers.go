package api

import (
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// ListModules devuelve los metadatos de los módulos namespaced registrados.
func (h *Handler) ListModules(w http.ResponseWriter, r *http.Request) {
	if h.moduleRegistry == nil {
		respondJSON(w, http.StatusOK, []modules.Info{})
		return
	}
	respondJSON(w, http.StatusOK, h.moduleRegistry.List())
}
