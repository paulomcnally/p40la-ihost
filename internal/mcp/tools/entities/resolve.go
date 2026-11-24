package entities

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// refError es un error accionable de resolución o duplicado.
type refError struct {
	Code    string
	Message string
	Hint    string
}

func (e *refError) Error() string { return e.Message }

func (e *refError) result() (*mcp.CallToolResult, any, error) {
	return toolkit.ToolError(e.Code, e.Message, e.Hint)
}

func invalidRequest(message string) *refError {
	return &refError{Code: "invalid_request", Message: message}
}

func invalidValue(message string) *refError {
	return &refError{Code: "invalid_value", Message: message}
}

func toResult(err error) (*mcp.CallToolResult, any, error) {
	if ref, ok := err.(*refError); ok {
		return ref.result()
	}
	return toolkit.APIError(err)
}

// normalize compara nombres sin distinguir mayúsculas, acentos ni espacios.
func normalize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	)
	return replacer.Replace(s)
}

func sameName(a, b string) bool {
	return normalize(a) == normalize(b)
}

// hintOptions devuelve hasta 10 opciones "id=name" para corregir una referencia.
func hintOptions[T any](items []T, id func(T) int64, name func(T) string) string {
	limit := len(items)
	if limit > 10 {
		limit = 10
	}
	parts := make([]string, 0, limit)
	for _, item := range items[:limit] {
		parts = append(parts, fmt.Sprintf("%d=%s", id(item), name(item)))
	}
	if len(parts) == 0 {
		return "No hay opciones disponibles."
	}
	return "Opciones: " + strings.Join(parts, ", ")
}

func findHomeByName(items []api.Home, name string) []api.Home {
	var found []api.Home
	for _, item := range items {
		if sameName(item.Name, name) {
			found = append(found, item)
		}
	}
	return found
}

func findCurrency(items []api.Currency, query string) []api.Currency {
	var found []api.Currency
	for _, item := range items {
		if sameName(item.Code, query) || sameName(item.Name, query) {
			found = append(found, item)
		}
	}
	return found
}

func findCategory(items []api.InstitutionCategory, query string) []api.InstitutionCategory {
	var found []api.InstitutionCategory
	for _, item := range items {
		if sameName(item.Key, query) || sameName(item.Name, query) {
			found = append(found, item)
		}
	}
	return found
}

func findInstitution(items []api.Institution, query string) []api.Institution {
	var found []api.Institution
	for _, item := range items {
		if sameName(item.Name, query) {
			found = append(found, item)
		}
	}
	return found
}

func findByName[T any](items []T, name func(T) string, query string) []T {
	var found []T
	for _, item := range items {
		if sameName(name(item), query) {
			found = append(found, item)
		}
	}
	return found
}

func (t *Toolset) resolveHome(ctx context.Context, id *int64, name string) (*api.Home, error) {
	homes, err := t.client.ListHomes(ctx)
	if err != nil {
		return nil, err
	}
	if id != nil {
		for i := range homes {
			if homes[i].ID == *id {
				return &homes[i], nil
			}
		}
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe un hogar con id=%d.", *id),
			Hint:    hintOptions(homes, func(h api.Home) int64 { return h.ID }, func(h api.Home) string { return h.Name }),
		}
	}
	query := strings.TrimSpace(name)
	if query == "" {
		return nil, invalidRequest("Indicá home o home_id.")
	}
	matches := findHomeByName(homes, query)
	switch len(matches) {
	case 0:
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe un hogar llamado %q.", query),
			Hint:    hintOptions(homes, func(h api.Home) int64 { return h.ID }, func(h api.Home) string { return h.Name }),
		}
	case 1:
		return &matches[0], nil
	default:
		return nil, &refError{
			Code:    "ambiguous_reference",
			Message: fmt.Sprintf("Hay %d hogares que coinciden con %q.", len(matches), query),
			Hint:    "Usá home_id. " + hintOptions(matches, func(h api.Home) int64 { return h.ID }, func(h api.Home) string { return h.Name }),
		}
	}
}

func (t *Toolset) resolveCurrency(ctx context.Context, id *int64, query string) (*api.Currency, error) {
	items, err := t.client.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	if id != nil {
		for i := range items {
			if items[i].ID == *id {
				return &items[i], nil
			}
		}
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe una moneda con id=%d.", *id),
			Hint:    hintOptions(items, func(c api.Currency) int64 { return c.ID }, func(c api.Currency) string { return c.Code }),
		}
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, invalidRequest("Indicá currency o currency_id.")
	}
	matches := findCurrency(items, query)
	switch len(matches) {
	case 0:
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe una moneda %q.", query),
			Hint:    hintOptions(items, func(c api.Currency) int64 { return c.ID }, func(c api.Currency) string { return c.Code }),
		}
	case 1:
		return &matches[0], nil
	default:
		return nil, &refError{
			Code:    "ambiguous_reference",
			Message: fmt.Sprintf("Hay %d monedas que coinciden con %q.", len(matches), query),
			Hint:    "Usá currency_id. " + hintOptions(matches, func(c api.Currency) int64 { return c.ID }, func(c api.Currency) string { return c.Code }),
		}
	}
}

func (t *Toolset) resolveCategory(ctx context.Context, id *int64, query string) (*api.InstitutionCategory, error) {
	items, err := t.client.ListInstitutionCategories(ctx)
	if err != nil {
		return nil, err
	}
	if id != nil {
		for i := range items {
			if items[i].ID == *id {
				return &items[i], nil
			}
		}
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe una categoría con id=%d.", *id),
			Hint:    hintOptions(items, func(c api.InstitutionCategory) int64 { return c.ID }, func(c api.InstitutionCategory) string { return c.Key }),
		}
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, invalidRequest("Indicá category o category_id.")
	}
	matches := findCategory(items, query)
	switch len(matches) {
	case 0:
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe una categoría %q.", query),
			Hint:    hintOptions(items, func(c api.InstitutionCategory) int64 { return c.ID }, func(c api.InstitutionCategory) string { return c.Key }),
		}
	case 1:
		return &matches[0], nil
	default:
		return nil, &refError{
			Code:    "ambiguous_reference",
			Message: fmt.Sprintf("Hay %d categorías que coinciden con %q.", len(matches), query),
			Hint:    "Usá category_id. " + hintOptions(matches, func(c api.InstitutionCategory) int64 { return c.ID }, func(c api.InstitutionCategory) string { return c.Key }),
		}
	}
}

func (t *Toolset) resolveInstitution(ctx context.Context, id *int64, name string) (*api.Institution, error) {
	items, err := t.client.ListInstitutions(ctx)
	if err != nil {
		return nil, err
	}
	if id != nil {
		for i := range items {
			if items[i].ID == *id {
				return &items[i], nil
			}
		}
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe una institución con id=%d.", *id),
			Hint:    hintOptions(items, func(i api.Institution) int64 { return i.ID }, func(i api.Institution) string { return i.Name }),
		}
	}
	query := strings.TrimSpace(name)
	if query == "" {
		return nil, nil
	}
	matches := findInstitution(items, query)
	switch len(matches) {
	case 0:
		return nil, &refError{
			Code:    "not_found",
			Message: fmt.Sprintf("No existe una institución llamada %q.", query),
			Hint:    hintOptions(items, func(i api.Institution) int64 { return i.ID }, func(i api.Institution) string { return i.Name }),
		}
	case 1:
		return &matches[0], nil
	default:
		return nil, &refError{
			Code:    "ambiguous_reference",
			Message: fmt.Sprintf("Hay %d instituciones que coinciden con %q.", len(matches), query),
			Hint:    "Usá institution_id. " + hintOptions(matches, func(i api.Institution) int64 { return i.ID }, func(i api.Institution) string { return i.Name }),
		}
	}
}
