// Package catalog implementa tools MCP de solo lectura de entidades
// existentes que no tienen detalle en las tools de alta (SPEC-154, Ola 1).
package catalog

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de catálogo.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las tools de catálogo en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "services_get",
		Description: "Devuelve el detalle completo de un servicio: facturación, día de cobro, recurrencia, " +
			"vigencia, ciclo, contrato y último estado de factura. No crea facturas. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.serviceGet)
}

type serviceGetInput struct {
	ServiceID int64 `json:"service_id" jsonschema:"ID del servicio (obligatorio)"`
}

func (t *Toolset) serviceGet(ctx context.Context, _ *mcp.CallToolRequest, input serviceGetInput) (*mcp.CallToolResult, any, error) {
	if input.ServiceID <= 0 {
		return toolkit.ToolError("invalid_request", "service_id debe ser mayor que cero", "")
	}
	var service map[string]any
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/services/%d", input.ServiceID), &service); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(toolkit.OmitDeep(service, "webhook_uuid"))
}
