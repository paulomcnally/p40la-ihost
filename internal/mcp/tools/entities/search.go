package entities

import (
	"context"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// EntitiesSearchInput es la entrada de entities_search.
type EntitiesSearchInput struct {
	Entity string `json:"entity" jsonschema:"Entidad a buscar: service, home, bank o institution"`
	Query  string `json:"query,omitempty" jsonschema:"Texto a buscar por nombre (sin distinguir mayúsculas ni acentos). Vacío lista las primeras entidades"`
	Home   string `json:"home,omitempty" jsonschema:"Solo para service: nombre del hogar para filtrar"`
	HomeID *int64 `json:"home_id,omitempty" jsonschema:"Solo para service: ID del hogar para filtrar (gana sobre home)"`
	Limit  *int   `json:"limit,omitempty" jsonschema:"Máximo de resultados (default 10, máximo 50)"`
}

// SearchItem es un candidato con los campos mínimos para desambiguar.
type SearchItem struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Address         string   `json:"address,omitempty"`
	Usage           string   `json:"usage,omitempty"`
	HomeID          *int64   `json:"home_id,omitempty"`
	Institution     string   `json:"institution,omitempty"`
	CurrencyID      *int64   `json:"currency_id,omitempty"`
	Frequency       string   `json:"frequency,omitempty"`
	SuggestedAmount *float64 `json:"suggested_amount,omitempty"`
	Active          *bool    `json:"active,omitempty"`
	ShortCode       string   `json:"short_code,omitempty"`
	Country         string   `json:"country,omitempty"`
	CategoryID      *int64   `json:"category_id,omitempty"`
}

// EntitiesSearchOutput es el resultado de entities_search.
type EntitiesSearchOutput struct {
	Entity    string       `json:"entity"`
	Query     string       `json:"query"`
	Count     int          `json:"count"`
	Total     int          `json:"total"`
	Truncated bool         `json:"truncated"`
	Items     []SearchItem `json:"items"`
	Hint      string       `json:"hint,omitempty"`
}

const (
	searchDefaultLimit = 10
	searchMaxLimit     = 50
	searchMaxQuery     = 100
)

type scoredItem struct {
	score int
	name  string
	item  SearchItem
}

func (t *Toolset) entitiesSearch(ctx context.Context, _ *mcp.CallToolRequest, in EntitiesSearchInput) (*mcp.CallToolResult, any, error) {
	entity := strings.ToLower(strings.TrimSpace(in.Entity))
	switch entity {
	case "service", "home", "bank", "institution":
	default:
		return toolkit.ToolError("invalid_value", "entity debe ser service, home, bank o institution.",
			"Valores válidos: service, home, bank, institution.")
	}
	query := strings.TrimSpace(in.Query)
	if len(query) > searchMaxQuery {
		return toolkit.ToolError("invalid_request", "query supera los 100 caracteres.", "Acortá el término de búsqueda.")
	}
	limit := searchDefaultLimit
	if in.Limit != nil {
		limit = *in.Limit
	}
	if limit < 1 || limit > searchMaxLimit {
		return toolkit.ToolError("invalid_value", "limit debe estar entre 1 y 50.", "")
	}
	normalizedQuery := normalize(query)

	var items []SearchItem
	switch entity {
	case "home":
		homes, err := t.client.ListHomes(ctx)
		if err != nil {
			return toolkit.APIError(err)
		}
		for _, home := range homes {
			items = append(items, SearchItem{
				ID:      home.ID,
				Name:    home.Name,
				Address: home.Address,
				Usage:   home.Usage,
			})
		}
	case "institution":
		institutions, err := t.client.ListInstitutions(ctx)
		if err != nil {
			return toolkit.APIError(err)
		}
		for _, institution := range institutions {
			items = append(items, SearchItem{
				ID:         institution.ID,
				Name:       institution.Name,
				CategoryID: institution.CategoryID,
			})
		}
	case "bank":
		banks, err := t.client.ListBanks(ctx)
		if err != nil {
			return toolkit.APIError(err)
		}
		for _, bank := range banks {
			active := bank.Active
			items = append(items, SearchItem{
				ID:        bank.ID,
				Name:      bank.Name,
				ShortCode: bank.ShortCode,
				Country:   bank.Country,
				Active:    &active,
			})
		}
	case "service":
		var homeID *int64
		if in.HomeID != nil && *in.HomeID > 0 {
			homeID = in.HomeID
		} else if strings.TrimSpace(in.Home) != "" {
			home, err := t.resolveHome(ctx, nil, in.Home)
			if err != nil {
				return toResult(err)
			}
			homeID = &home.ID
		}
		services, err := t.client.ListServices(ctx, homeID)
		if err != nil {
			return toolkit.APIError(err)
		}
		for _, service := range services {
			home := service.HomeID
			currency := service.CurrencyID
			amount := service.SuggestedAmount
			active := service.Active
			items = append(items, SearchItem{
				ID:              service.ID,
				Name:            service.Name,
				HomeID:          &home,
				Institution:     service.Institution,
				CurrencyID:      &currency,
				Frequency:       service.Frequency,
				SuggestedAmount: &amount,
				Active:          &active,
			})
		}
	}

	scored := make([]scoredItem, 0, len(items))
	for _, item := range items {
		score, ok := searchScore(entity, item, normalizedQuery)
		if !ok {
			continue
		}
		scored = append(scored, scoredItem{score: score, name: normalize(item.Name), item: item})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score < scored[j].score
		}
		if scored[i].name != scored[j].name {
			return scored[i].name < scored[j].name
		}
		return scored[i].item.ID < scored[j].item.ID
	})

	total := len(scored)
	truncated := total > limit
	if truncated {
		scored = scored[:limit]
	}
	result := make([]SearchItem, 0, len(scored))
	for _, entry := range scored {
		result = append(result, entry.item)
	}

	output := EntitiesSearchOutput{
		Entity:    entity,
		Query:     query,
		Count:     len(result),
		Total:     total,
		Truncated: truncated,
		Items:     result,
	}
	if total == 0 {
		if query == "" {
			output.Hint = "No hay entidades registradas de ese tipo."
		} else {
			output.Hint = "No se encontraron coincidencias. Probá con otro término o sin query para listar."
		}
	}
	return toolkit.JSONResult(output)
}

// searchScore devuelve 0 exacto, 1 prefijo, 2 contenido y 3 listado sin query.
func searchScore(entity string, item SearchItem, query string) (int, bool) {
	if query == "" {
		return 3, true
	}
	best := -1
	consider := func(value string) {
		if value == "" {
			return
		}
		normalized := normalize(value)
		switch {
		case normalized == query:
			best = 0
		case strings.HasPrefix(normalized, query):
			if best != 0 {
				best = 1
			}
		case strings.Contains(normalized, query):
			if best < 0 || best > 2 {
				best = 2
			}
		}
	}
	consider(item.Name)
	if entity == "bank" {
		consider(item.ShortCode)
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}
