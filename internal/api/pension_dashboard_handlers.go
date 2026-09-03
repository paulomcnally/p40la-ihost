package api

import (
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// PensionDashboardHandlers agrupa handlers del módulo Pensión (generación).
type PensionDashboardHandlers struct {
	generation *services.PensionGenerationService
}

// NewPensionDashboardHandlers crea un nuevo PensionDashboardHandlers.
func NewPensionDashboardHandlers(generation *services.PensionGenerationService) *PensionDashboardHandlers {
	return &PensionDashboardHandlers{generation: generation}
}

// GenerateMonth genera los pagos de salario y registros de un mes.
func (h *PensionDashboardHandlers) GenerateMonth(w http.ResponseWriter, r *http.Request) {
	year, month := periodFromQuery(r)
	result, err := h.generation.Generate(r.Context(), year, month)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"ok":                      true,
		"year":                    result.Year,
		"month":                   result.Month,
		"created_salary_payments": result.CreatedSalaryPayments,
		"created_support_records": result.CreatedSupportRecords,
	})
}
