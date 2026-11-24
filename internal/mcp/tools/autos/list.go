package autos

import (
	"context"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListInput es la entrada de autos_list.
type ListInput struct {
	Query  *string `json:"query,omitempty" jsonschema:"Texto opcional para filtrar por marca, modelo, placa o VIN"`
	AutoID *int64  `json:"auto_id,omitempty" jsonschema:"Si se indica, devuelve solo ese auto"`
}

// ListOutput es el resultado de autos_list.
type ListOutput struct {
	Query string       `json:"query,omitempty"`
	Count int          `json:"count"`
	Autos []AutoOutput `json:"autos"`
}

var accentFolder = strings.NewReplacer(
	"á", "a", "à", "a", "ä", "a", "â", "a",
	"é", "e", "è", "e", "ë", "e", "ê", "e",
	"í", "i", "ì", "i", "ï", "i", "î", "i",
	"ó", "o", "ò", "o", "ö", "o", "ô", "o",
	"ú", "u", "ù", "u", "ü", "u", "û", "u",
	"ñ", "n", "ç", "c",
)

func normalizeText(value string) string {
	return accentFolder.Replace(strings.ToLower(strings.TrimSpace(value)))
}

func (t *Toolset) list(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, any, error) {
	autos, err := t.client.ListAutos(ctx)
	if err != nil {
		return apiError(err)
	}

	query := ""
	if input.Query != nil {
		query = strings.TrimSpace(*input.Query)
	}
	needle := normalizeText(query)

	output := ListOutput{Query: query, Autos: make([]AutoOutput, 0, len(autos))}
	for _, auto := range autos {
		if input.AutoID != nil && auto.ID != *input.AutoID {
			continue
		}
		if needle != "" {
			haystack := normalizeText(strings.Join([]string{auto.Brand, auto.Model, auto.Placa, auto.VIN}, " "))
			if !strings.Contains(haystack, needle) {
				continue
			}
		}
		output.Autos = append(output.Autos, *autoOutput(&auto))
	}

	sort.SliceStable(output.Autos, func(i, j int) bool {
		a, b := output.Autos[i], output.Autos[j]
		if a.Brand != b.Brand {
			return a.Brand < b.Brand
		}
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		return a.ID < b.ID
	})
	output.Count = len(output.Autos)
	return jsonResult(output)
}
