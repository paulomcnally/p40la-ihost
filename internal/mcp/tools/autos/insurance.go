package autos

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// InsuranceInput es la entrada de autos_insurance_register.
type InsuranceInput struct {
	AutoID        int64   `json:"auto_id" jsonschema:"ID del auto (obligatorio)"`
	ServiceID     int64   `json:"service_id" jsonschema:"ID del servicio de seguro a asociar (obligatorio)"`
	CoverageType  string  `json:"coverage_type" jsonschema:"Tipo de cobertura: daños_a_terceros o full_cover (obligatorio)"`
	PolicyNumber  string  `json:"policy_number" jsonschema:"Número de póliza (obligatorio)"`
	InsurerNumber string  `json:"insurer_number" jsonschema:"Número de aseguradora (obligatorio)"`
	Certificate   *string `json:"certificate,omitempty" jsonschema:"Número de certificado (opcional)"`
	DryRun        *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) insuranceRegister(ctx context.Context, _ *mcp.CallToolRequest, input InsuranceInput) (*mcp.CallToolResult, any, error) {
	if input.AutoID <= 0 {
		return toolError("invalid_request", "auto_id debe ser mayor que cero", "")
	}
	if input.ServiceID <= 0 {
		return toolError("invalid_request", "service_id debe ser mayor que cero", "Usá un servicio de seguro existente.")
	}
	coverage := strings.ToLower(strings.TrimSpace(input.CoverageType))
	if coverage != "daños_a_terceros" && coverage != "full_cover" {
		return toolError("invalid_request", "coverage_type debe ser daños_a_terceros o full_cover", "")
	}
	policy := strings.TrimSpace(input.PolicyNumber)
	if policy == "" {
		return toolError("invalid_request", "policy_number es requerido", "")
	}
	insurer := strings.TrimSpace(input.InsurerNumber)
	if insurer == "" {
		return toolError("invalid_request", "insurer_number es requerido", "")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	available, err := t.client.ListAvailableServices(ctx, input.AutoID)
	if err != nil {
		return apiError(err)
	}

	insurances, err := t.client.ListInsurances(ctx, input.AutoID)
	if err != nil {
		return apiError(err)
	}
	for _, existing := range insurances {
		if existing.ServiceID == input.ServiceID {
			return toolError("already_exists",
				"el auto ya tiene asociado ese servicio de seguro",
				fmt.Sprintf("El seguro existente tiene id %d y póliza %s.", existing.ID, existing.PolicyNumber))
		}
	}

	service, ok := findAvailableService(available, input.ServiceID)
	if !ok {
		return toolError("prerequisite_missing",
			"el servicio no está disponible para asociar al auto",
			fmt.Sprintf("Consultá /api/autos/%d/available-services; hay %d servicios disponibles.", input.AutoID, len(available)))
	}

	req := api.InsuranceRequest{
		ServiceID:     input.ServiceID,
		CoverageType:  coverage,
		PolicyNumber:  policy,
		Certificate:   stringValue(input.Certificate, ""),
		InsurerNumber: insurer,
	}

	if dryRun {
		return jsonResult(WriteOutput{
			Status: "planned",
			DryRun: true,
			Planned: map[string]any{
				"auto_id":   input.AutoID,
				"insurance": req,
				"service":   service.Name,
				"coverage":  coverage,
			},
		})
	}

	insurance, err := t.client.CreateInsurance(ctx, input.AutoID, req)
	if err != nil {
		return apiError(err)
	}
	return jsonResult(WriteOutput{
		Status:    "created",
		DryRun:    false,
		Insurance: insurance,
	})
}

func findAvailableService(services []api.AvailableService, id int64) (api.AvailableService, bool) {
	for _, service := range services {
		if service.ID == id {
			return service, true
		}
	}
	return api.AvailableService{}, false
}
