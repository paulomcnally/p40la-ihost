package autos

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// AutoOutput es un auto resumido para las respuestas del MCP.
type AutoOutput struct {
	ID     int64  `json:"id"`
	Year   int64  `json:"year"`
	Brand  string `json:"brand"`
	Model  string `json:"model"`
	Color  string `json:"color"`
	Icon   string `json:"icon"`
	Motor  string `json:"motor,omitempty"`
	Chasis string `json:"chasis,omitempty"`
	VIN    string `json:"vin,omitempty"`
	Placa  string `json:"placa,omitempty"`
}

// WriteOutput es la respuesta común de las tools de escritura.
type WriteOutput struct {
	Status      string                  `json:"status"`
	DryRun      bool                    `json:"dry_run"`
	Auto        *AutoOutput             `json:"auto,omitempty"`
	Planned     any                     `json:"planned,omitempty"`
	Maintenance *api.Maintenance        `json:"maintenance,omitempty"`
	Items       []api.MaintenanceItem   `json:"items,omitempty"`
	Partial     bool                    `json:"partial,omitempty"`
	Warnings    []string                `json:"warnings,omitempty"`
	Inspection  *api.Inspection         `json:"inspection,omitempty"`
	Schedule    *api.InspectionSchedule `json:"schedule,omitempty"`
	Insurance   *api.Insurance          `json:"insurance,omitempty"`
}

func autoOutput(auto *api.Auto) *AutoOutput {
	if auto == nil {
		return nil
	}
	return &AutoOutput{
		ID:     auto.ID,
		Year:   auto.Year,
		Brand:  auto.Brand,
		Model:  auto.Model,
		Color:  auto.Color,
		Icon:   auto.Icon,
		Motor:  auto.Motor,
		Chasis: auto.Chasis,
		VIN:    auto.VIN,
		Placa:  auto.Placa,
	}
}

func jsonResult(value any) (*mcp.CallToolResult, any, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return toolError("internal_error", "no se pudo serializar el resultado", "")
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
		StructuredContent: value,
	}, value, nil
}

func toolError(code, message, hint string) (*mcp.CallToolResult, any, error) {
	payload := map[string]any{
		"error":   code,
		"message": message,
	}
	if hint != "" {
		payload["hint"] = hint
	}
	encoded, _ := json.Marshal(payload)
	return &mcp.CallToolResult{
		IsError:           true,
		Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
		StructuredContent: payload,
	}, payload, nil
}

func apiError(err error) (*mcp.CallToolResult, any, error) {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.Code
		hint := hintForCode(code)
		message := apiErr.Message
		if code == "invalid_request" && strings.Contains(strings.ToLower(message), "unique") {
			code = "already_exists"
			hint = "El auto ya tiene asociado ese servicio; usá autos_list para verificarlo."
		}
		return toolError(code, message, hint)
	}
	return toolError("upstream_error", "No se pudo contactar la API: "+err.Error(),
		"Verificá P40LA_BASE_URL y que GET /health responda.")
}

func hintForCode(code string) string {
	switch code {
	case "auth_failed":
		return "Revisá P40LA_API_KEY y que scripts/create-api-key.sh haya provisionado la key."
	case "not_found":
		return "Verificá el auto_id, maintenance_id o service_id en la UI o con autos_list."
	case "institution_category":
		return "El mantenimiento requiere un taller y la inspección un centro de inspección mecánica."
	case "writes_disabled":
		return "Definí P40LA_MCP_ALLOW_WRITES=true para ejecutar cambios."
	default:
		return ""
	}
}
