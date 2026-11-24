package entities

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// BanksCreateInput es la entrada de banks_create.
type BanksCreateInput struct {
	Name      string `json:"name" jsonschema:"Nombre del banco (obligatorio)"`
	ShortCode string `json:"short_code,omitempty" jsonschema:"Código corto (opcional)"`
	Country   string `json:"country,omitempty" jsonschema:"País ISO-2, por ejemplo NI (opcional)"`
	Notes     string `json:"notes,omitempty" jsonschema:"Notas (opcional)"`
	Active    *bool  `json:"active,omitempty" jsonschema:"Activo; default true"`
}

func (t *Toolset) banksCreate(ctx context.Context, _ *mcp.CallToolRequest, in BanksCreateInput) (*mcp.CallToolResult, any, error) {
	if result, output, err := t.checkWrites(); result != nil {
		return result, output, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return toolkit.ToolError("invalid_request", "name es obligatorio.", "")
	}
	country := strings.ToUpper(strings.TrimSpace(in.Country))
	if country != "" {
		if len(country) != 2 || country[0] < 'A' || country[0] > 'Z' || country[1] < 'A' || country[1] > 'Z' {
			return toolkit.ToolError("invalid_value", "country debe ser un código ISO-2 de dos letras.",
				"Ejemplo: NI, PA, US.")
		}
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}

	banks, err := t.client.ListBanks(ctx)
	if err != nil {
		return toolkit.APIError(err)
	}
	if matches := findByName(banks, func(b api.Bank) string { return b.Name }, name); len(matches) > 0 {
		existing := matches[0]
		return toolkit.ToolError("already_exists", "Ya existe un banco con ese nombre.",
			fmt.Sprintf("Usá el existente: id=%d, name=%q", existing.ID, existing.Name))
	}

	created, err := t.client.CreateBank(ctx, api.BankCreateRequest{
		Name:      name,
		ShortCode: strings.TrimSpace(in.ShortCode),
		Country:   country,
		Notes:     strings.TrimSpace(in.Notes),
		Active:    &active,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(created)
}
