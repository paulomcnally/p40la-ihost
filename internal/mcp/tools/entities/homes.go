package entities

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// HomesCreateInput es la entrada de homes_create.
type HomesCreateInput struct {
	Name    string `json:"name" jsonschema:"Nombre del hogar (obligatorio)"`
	Address string `json:"address,omitempty" jsonschema:"Dirección (opcional)"`
	Usage   string `json:"usage,omitempty" jsonschema:"Uso: rental (default), inhabited o unused"`
}

var homeUsages = map[string]bool{"rental": true, "inhabited": true, "unused": true}

func (t *Toolset) homesCreate(ctx context.Context, _ *mcp.CallToolRequest, in HomesCreateInput) (*mcp.CallToolResult, any, error) {
	if result, output, err := t.checkWrites(); result != nil {
		return result, output, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return toolkit.ToolError("invalid_request", "name es obligatorio.", "")
	}
	usage := strings.ToLower(strings.TrimSpace(in.Usage))
	if usage == "" {
		usage = "rental"
	}
	if !homeUsages[usage] {
		return toolkit.ToolError("invalid_value", "usage debe ser rental, inhabited o unused.",
			"Valores válidos: rental, inhabited, unused.")
	}

	homes, err := t.client.ListHomes(ctx)
	if err != nil {
		return toolkit.APIError(err)
	}
	if matches := findByName(homes, func(h api.Home) string { return h.Name }, name); len(matches) > 0 {
		existing := matches[0]
		return toolkit.ToolError("already_exists", "Ya existe un hogar con ese nombre.",
			fmt.Sprintf("Usá el existente: id=%d, name=%q", existing.ID, existing.Name))
	}

	created, err := t.client.CreateHome(ctx, api.HomeCreateRequest{
		Name:    name,
		Address: strings.TrimSpace(in.Address),
		Usage:   usage,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(created)
}
