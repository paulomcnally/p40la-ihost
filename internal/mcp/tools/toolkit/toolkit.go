// Package toolkit agrupa helpers compartidos por las tools MCP.
package toolkit

import (
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// JSONResult serializa un valor como contenido de texto y estructurado.
func JSONResult(value any) (*mcp.CallToolResult, any, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ToolError("internal_error", "no se pudo serializar el resultado", "")
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
		StructuredContent: value,
	}, value, nil
}

// ToolError construye un resultado de error con código estable y hint opcional.
func ToolError(code, message, hint string) (*mcp.CallToolResult, any, error) {
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

// APIError traduce un error del cliente HTTP a un ToolError accionable.
func APIError(err error) (*mcp.CallToolResult, any, error) {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		hint := ""
		switch apiErr.Code {
		case "auth_failed":
			hint = "Revisá la API key del MCP (Configuración → MCP en p40la-ihost)."
		case "service_not_found":
			hint = "Verificá el service_id en la UI o con bills_list."
		case "service_inactive":
			hint = "Activá el servicio antes de generar facturas."
		case "amount_unresolved":
			hint = "Indicá amount o configurá un monto sugerido en el servicio."
		case "writes_disabled":
			hint = "Definí P40LA_MCP_ALLOW_WRITES=true para ejecutar cambios."
		case "attachment_limit":
			hint = "El reporte ya tiene 10 adjuntos; borrá uno o usá otro reporte."
		case "attachment_not_found":
			hint = "Verificá el attachment_id con home_work_request_attachment_list."
		case "request_too_large":
			hint = "El archivo supera el máximo de 25 MB; comprimilo o usá la web."
		case "invalid_file":
			hint = "Solo se aceptan JPEG, PNG, WebP, HEIC, MP4, QuickTime y WebM con firma válida."
		case "upload_link_disabled":
			hint = "Configurá UPLOAD_TOKEN_SECRET (mínimo 32 caracteres) en el backend y reiniciá la app."
		case "upload_token_exhausted":
			hint = "El link ya usó su cupo; pedí un link nuevo con attachment_upload_link."
		case "invalid_token", "expired_token":
			hint = "El token no es válido o expiró; pedí un link nuevo con attachment_upload_link."
		case "invalid_status":
			hint = "Solo los reportes en estado reported aceptan evidencia nueva; reactivalo o usá la web."
		case "invalid_client_request_id":
			hint = "Usá entre 1 y 64 caracteres [A-Za-z0-9._:-]."
		}
		return ToolError(apiErr.Code, apiErr.Message, hint)
	}
	return ToolError("upstream_error", "No se pudo contactar la API: "+err.Error(),
		"Verificá P40LA_BASE_URL y que GET /health responda.")
}
