package banking

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// lookupError describe un fallo de resolución que la tool traduce a un código
// estable con hint accionable.
type lookupError struct {
	Code    string
	Message string
	Hint    string
}

func (e *lookupError) Error() string {
	return e.Message
}

// lookupFailure traduce un error de resolución o de la API a un resultado MCP.
func lookupFailure(err error) (*mcp.CallToolResult, any, error) {
	var lookup *lookupError
	if errors.As(err, &lookup) {
		return toolError(lookup.Code, lookup.Message, lookup.Hint)
	}
	return apiError(err)
}

// accentedReplacer normaliza los acentos y la eñe para comparar nombres.
var accentedReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "ä", "a", "â", "a",
	"é", "e", "è", "e", "ë", "e", "ê", "e",
	"í", "i", "ì", "i", "ï", "i", "î", "i",
	"ó", "o", "ò", "o", "ö", "o", "ô", "o",
	"ú", "u", "ù", "u", "ü", "u", "û", "u",
	"ñ", "n", "ç", "c",
)

// fold normaliza un texto para comparar sin distinguir mayúsculas ni acentos.
func fold(value string) string {
	return accentedReplacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}

// matchScore devuelve 0 para coincidencia exacta, 1 para prefijo, 2 para
// contenido y -1 cuando no hay coincidencia. Con consulta vacía todo coincide.
func matchScore(fields []string, query string) int {
	if query == "" {
		return 0
	}
	best := -1
	for _, field := range fields {
		normalized := fold(field)
		switch {
		case normalized == query:
			return 0
		case strings.HasPrefix(normalized, query):
			if best < 0 || best > 1 {
				best = 1
			}
		case strings.Contains(normalized, query):
			if best < 0 || best > 2 {
				best = 2
			}
		}
	}
	return best
}

func candidateHint(values []string, limit int) string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	sort.Strings(unique)
	if len(unique) > limit {
		unique = unique[:limit]
	}
	if len(unique) == 0 {
		return ""
	}
	return "Opciones disponibles: " + strings.Join(unique, ", ") + "."
}

// resolveBank resuelve un banco por id o por nombre/short_code.
func (t *Toolset) resolveBank(ctx context.Context, name string, id int64) (*api.BankRef, error) {
	banks, err := t.client.ListBankRefs(ctx)
	if err != nil {
		return nil, err
	}
	if id > 0 {
		for i := range banks {
			if banks[i].ID == id {
				return &banks[i], nil
			}
		}
		return nil, &lookupError{Code: "not_found", Message: "No existe un banco con ese id.", Hint: bankHint(banks)}
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, &lookupError{Code: "invalid_value", Message: "Indicá bank (nombre o short_code) o bank_id.", Hint: bankHint(banks)}
	}
	if parsed, parseErr := strconv.ParseInt(trimmed, 10, 64); parseErr == nil && parsed > 0 {
		for i := range banks {
			if banks[i].ID == parsed {
				return &banks[i], nil
			}
		}
		return nil, &lookupError{Code: "not_found", Message: "No existe un banco con ese id.", Hint: bankHint(banks)}
	}

	query := fold(trimmed)
	bestScore := 3
	var candidates []api.BankRef
	for _, bank := range banks {
		score := matchScore([]string{bank.Name, bank.ShortCode}, query)
		if score < 0 {
			continue
		}
		switch {
		case score < bestScore:
			bestScore = score
			candidates = []api.BankRef{bank}
		case score == bestScore:
			candidates = append(candidates, bank)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, &lookupError{
			Code:    "not_found",
			Message: "No existe un banco que coincida con " + trimmed + ".",
			Hint:    bankHint(banks),
		}
	case 1:
		return &candidates[0], nil
	default:
		names := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			names = append(names, candidate.Name)
		}
		return nil, &lookupError{
			Code:    "ambiguous_reference",
			Message: "Hay más de un banco que coincide con " + trimmed + ".",
			Hint:    "Enviá bank_id para desambiguar. Candidatos: " + strings.Join(names, ", ") + ".",
		}
	}
}

func bankHint(banks []api.BankRef) string {
	values := make([]string, 0, len(banks))
	for _, bank := range banks {
		if bank.ShortCode != "" {
			values = append(values, bank.Name+" ("+bank.ShortCode+")")
			continue
		}
		values = append(values, bank.Name)
	}
	if hint := candidateHint(values, 10); hint != "" {
		return hint
	}
	return "Todavía no hay bancos cargados; creá uno desde la UI de Bancos o con banks_create si está disponible."
}

// resolveCurrency resuelve una moneda por id o por código/nombre.
func (t *Toolset) resolveCurrency(ctx context.Context, code string, id int64) (*api.CurrencyRef, error) {
	currencies, err := t.client.ListCurrencyRefs(ctx)
	if err != nil {
		return nil, err
	}
	if id > 0 {
		for i := range currencies {
			if currencies[i].ID == id {
				return &currencies[i], nil
			}
		}
		return nil, &lookupError{Code: "not_found", Message: "No existe una moneda con ese id.", Hint: currencyHint(currencies)}
	}

	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return nil, &lookupError{Code: "invalid_value", Message: "Indicá currency (código) o currency_id.", Hint: currencyHint(currencies)}
	}
	if parsed, parseErr := strconv.ParseInt(trimmed, 10, 64); parseErr == nil && parsed > 0 {
		for i := range currencies {
			if currencies[i].ID == parsed {
				return &currencies[i], nil
			}
		}
		return nil, &lookupError{Code: "not_found", Message: "No existe una moneda con ese id.", Hint: currencyHint(currencies)}
	}

	for i := range currencies {
		if strings.EqualFold(currencies[i].Code, trimmed) {
			return &currencies[i], nil
		}
	}
	query := fold(trimmed)
	bestScore := 3
	var candidates []api.CurrencyRef
	for _, currency := range currencies {
		score := matchScore([]string{currency.Code, currency.Name}, query)
		if score < 0 {
			continue
		}
		switch {
		case score < bestScore:
			bestScore = score
			candidates = []api.CurrencyRef{currency}
		case score == bestScore:
			candidates = append(candidates, currency)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, &lookupError{
			Code:    "not_found",
			Message: "No existe una moneda que coincida con " + trimmed + ".",
			Hint:    currencyHint(currencies),
		}
	case 1:
		return &candidates[0], nil
	default:
		codes := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			codes = append(codes, candidate.Code)
		}
		return nil, &lookupError{
			Code:    "ambiguous_reference",
			Message: "Hay más de una moneda que coincide con " + trimmed + ".",
			Hint:    "Usá el código exacto: " + strings.Join(codes, ", ") + ".",
		}
	}
}

func currencyHint(currencies []api.CurrencyRef) string {
	values := make([]string, 0, len(currencies))
	for _, currency := range currencies {
		if currency.Name != "" {
			values = append(values, currency.Code+" ("+currency.Name+")")
			continue
		}
		values = append(values, currency.Code)
	}
	if hint := candidateHint(values, 10); hint != "" {
		return hint
	}
	return "Todavía no hay monedas cargadas."
}
