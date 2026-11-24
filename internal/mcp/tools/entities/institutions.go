package entities

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// InstitutionsCreateInput es la entrada de institutions_create.
type InstitutionsCreateInput struct {
	Name       string `json:"name" jsonschema:"Nombre de la institución (obligatorio)"`
	CategoryID *int64 `json:"category_id,omitempty" jsonschema:"ID de la categoría (opcional; gana sobre category)"`
	Category   string `json:"category,omitempty" jsonschema:"Clave o nombre de la categoría (opcional)"`
}

func (t *Toolset) institutionsCreate(ctx context.Context, _ *mcp.CallToolRequest, in InstitutionsCreateInput) (*mcp.CallToolResult, any, error) {
	if result, output, err := t.checkWrites(); result != nil {
		return result, output, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return toolkit.ToolError("invalid_request", "name es obligatorio.", "")
	}

	var categoryID *int64
	if in.CategoryID != nil {
		category, err := t.resolveCategory(ctx, in.CategoryID, "")
		if err != nil {
			return toResult(err)
		}
		categoryID = &category.ID
	} else if strings.TrimSpace(in.Category) != "" {
		category, err := t.resolveCategory(ctx, nil, in.Category)
		if err != nil {
			return toResult(err)
		}
		categoryID = &category.ID
	}

	institutions, err := t.client.ListInstitutions(ctx)
	if err != nil {
		return toolkit.APIError(err)
	}
	if matches := findByName(institutions, func(i api.Institution) string { return i.Name }, name); len(matches) > 0 {
		existing := matches[0]
		return toolkit.ToolError("already_exists", "Ya existe una institución con ese nombre.",
			fmt.Sprintf("Usá la existente: id=%d, name=%q", existing.ID, existing.Name))
	}

	created, err := t.client.CreateInstitution(ctx, api.InstitutionCreateRequest{
		Name:       name,
		CategoryID: categoryID,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(created)
}
