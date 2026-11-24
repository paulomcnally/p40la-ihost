package banking

import (
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

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
		hint := ""
		switch apiErr.Code {
		case "auth_failed":
			hint = "Revisá la API key del MCP (Configuración → MCP en p40la-ihost)."
		case "bank_not_found":
			hint = "Verificá el banco con bank_accounts_search o credit_cards_search, o resolvelo por nombre."
		case "account_not_found":
			hint = "Verificá el id con bank_accounts_search."
		case "credit_card_not_found":
			hint = "Verificá el id con credit_cards_search."
		case "duplicate_account", "duplicate_card":
			hint = "La entidad ya existe: buscala con la tool de búsqueda correspondiente."
		case "writes_disabled":
			hint = "Definí P40LA_MCP_ALLOW_WRITES=true para ejecutar cambios."
		}
		code := apiErr.Code
		switch {
		case apiErr.Status == 404:
			code = "not_found"
		case apiErr.Status == 409:
			code = "already_exists"
		case apiErr.Status == 401 || apiErr.Status == 403:
			code = "auth_failed"
		case apiErr.Status >= 400 && apiErr.Status < 500:
			code = "invalid_value"
		}
		return toolError(code, apiErr.Message, hint)
	}
	return toolError("upstream_error", "No se pudo contactar la API: "+err.Error(),
		"Verificá P40LA_BASE_URL y que GET /health responda.")
}
