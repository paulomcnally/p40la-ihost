package entities

import (
	"context"
	"errors"
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

// HomesUpdateInput es la entrada de homes_update. Todos los campos editables
// son opcionales: lo que se omite conserva el valor actual del hogar.
type HomesUpdateInput struct {
	HomeID  *int64  `json:"home_id,omitempty" jsonschema:"ID del hogar a editar; usá home_id o home"`
	Home    string  `json:"home,omitempty" jsonschema:"Nombre del hogar a editar; usá home_id o home"`
	Name    *string `json:"name,omitempty" jsonschema:"Nombre nuevo; si se omite se conserva el actual"`
	Address *string `json:"address,omitempty" jsonschema:"Dirección nueva; si se omite se conserva y con cadena vacía se limpia"`
	Usage   string  `json:"usage,omitempty" jsonschema:"Uso nuevo: rental, inhabited o unused; si se omite se conserva"`
}

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

func (t *Toolset) homesUpdate(ctx context.Context, _ *mcp.CallToolRequest, in HomesUpdateInput) (*mcp.CallToolResult, any, error) {
	if result, output, err := t.checkWrites(); result != nil {
		return result, output, err
	}

	current, err := t.resolveHome(ctx, in.HomeID, in.Home)
	if err != nil {
		return toResult(err)
	}

	name := current.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if name == "" {
			return toolkit.ToolError("invalid_request", "name no puede estar vacío.",
				"Omití name para conservar el actual.")
		}
		if !sameName(name, current.Name) {
			homes, err := t.client.ListHomes(ctx)
			if err != nil {
				return toolkit.APIError(err)
			}
			if matches := findByName(homes, func(h api.Home) string { return h.Name }, name); len(matches) > 0 {
				existing := matches[0]
				return toolkit.ToolError("already_exists", "Ya existe un hogar con ese nombre.",
					fmt.Sprintf("Usá otro nombre o editá el existente: id=%d, name=%q", existing.ID, existing.Name))
			}
		}
	}

	address := current.Address
	if in.Address != nil {
		address = strings.TrimSpace(*in.Address)
	}

	usage := strings.ToLower(strings.TrimSpace(current.Usage))
	if usage == "" {
		usage = "rental"
	}
	if strings.TrimSpace(in.Usage) != "" {
		usage = strings.ToLower(strings.TrimSpace(in.Usage))
		if !homeUsages[usage] {
			return toolkit.ToolError("invalid_value", "usage debe ser rental, inhabited o unused.",
				"Valores válidos: rental, inhabited, unused.")
		}
	}

	updated, err := t.client.UpdateHome(ctx, current.ID, api.HomeUpdateRequest{
		Name:    name,
		Address: address,
		Usage:   usage,
	})
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "home_has_active_rental" {
			return toolkit.ToolError(apiErr.Code, apiErr.Message,
				"Cerrá el contrato o la reserva activa antes de cambiar el uso. "+
					"Podés editar name/address sin cambiar usage.")
		}
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(updated)
}
