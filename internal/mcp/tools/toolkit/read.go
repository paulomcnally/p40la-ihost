package toolkit

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// WithQuery agrega a la ruta solo los parámetros con valor.
func WithQuery(path string, values url.Values) string {
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}

// Project devuelve una copia del objeto con las claves presentes en keys.
// Se usa para allowlists de salida (SEC-NO-PII).
func Project(item map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, ok := item[key]; ok {
			out[key] = value
		}
	}
	return out
}

// ProjectAll aplica Project a una lista de objetos.
func ProjectAll(items []map[string]any, keys ...string) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, Project(item, keys...))
	}
	return out
}

// Omit devuelve una copia del objeto sin las claves indicadas.
func Omit(item map[string]any, keys ...string) map[string]any {
	blocked := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		blocked[key] = struct{}{}
	}
	out := make(map[string]any, len(item))
	for key, value := range item {
		if _, skip := blocked[key]; skip {
			continue
		}
		out[key] = value
	}
	return out
}

// OmitAll aplica Omit a una lista de objetos.
func OmitAll(items []map[string]any, keys ...string) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, Omit(item, keys...))
	}
	return out
}

// OmitDeep elimina las claves indicadas en cualquier nivel del valor.
// Necesario para estructuras anidadas (por ejemplo partes de un contrato).
// Normaliza con JSON para cubrir structs y slices tipados.
func OmitDeep(value any, keys ...string) any {
	blocked := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		blocked[key] = struct{}{}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return value
	}
	return omitDeep(decoded, blocked)
}

func omitDeep(value any, blocked map[string]struct{}) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if _, skip := blocked[key]; skip {
				continue
			}
			out[key] = omitDeep(item, blocked)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, omitDeep(item, blocked))
		}
		return out
	default:
		return value
	}
}

// Period valida un período YYYY-MM y devuelve año y mes.
func Period(raw string) (int, int, error) {
	trimmed := strings.TrimSpace(raw)
	parts := strings.Split(trimmed, "-")
	if len(parts) != 2 || len(parts[0]) != 4 || len(parts[1]) != 2 {
		return 0, 0, fmt.Errorf("el período debe tener formato YYYY-MM")
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil || year < 1900 || year > 2100 {
		return 0, 0, fmt.Errorf("el año debe estar entre 1900 y 2100")
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("el mes debe estar entre 1 y 12")
	}
	return year, month, nil
}

// CurrentPeriod devuelve el año y mes actuales en la zona indicada.
func CurrentPeriod(loc *time.Location) (int, int) {
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	return now.Year(), int(now.Month())
}

// CurrencyTotal agrega un monto por moneda.
type CurrencyTotal struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	Count    int     `json:"count"`
}

// TotalsByCurrency suma amountKey agrupando por currencyKey.
// Ignora objetos sin moneda y redondea a dos decimales.
func TotalsByCurrency(items []map[string]any, currencyKey, amountKey string) []CurrencyTotal {
	index := map[string]*CurrencyTotal{}
	order := make([]string, 0, len(items))
	for _, item := range items {
		currency, _ := item[currencyKey].(string)
		if strings.TrimSpace(currency) == "" {
			continue
		}
		total, ok := index[currency]
		if !ok {
			total = &CurrencyTotal{Currency: currency}
			index[currency] = total
			order = append(order, currency)
		}
		amount, _ := item[amountKey].(float64)
		total.Amount += amount
		total.Count++
	}
	out := make([]CurrencyTotal, 0, len(order))
	for _, currency := range order {
		total := index[currency]
		total.Amount = round2(total.Amount)
		out = append(out, *total)
	}
	return out
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
