package toolkit

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestWithQuery(t *testing.T) {
	if got := WithQuery("/api/x", url.Values{}); got != "/api/x" {
		t.Fatalf("sin valores = %q", got)
	}
	query := url.Values{}
	query.Set("year", "2026")
	query.Set("month", "11")
	if got := WithQuery("/api/x", query); got != "/api/x?month=11&year=2026" {
		t.Fatalf("con valores = %q", got)
	}
}

func TestPeriod(t *testing.T) {
	year, month, err := Period("2026-11")
	if err != nil || year != 2026 || month != 11 {
		t.Fatalf("period = %d-%d err=%v", year, month, err)
	}
	for _, invalid := range []string{"2026", "2026-13", "26-11", "2026-1", "abc-de"} {
		if _, _, err := Period(invalid); err == nil {
			t.Fatalf("%q debería fallar", invalid)
		}
	}
}

func TestOmitDeep(t *testing.T) {
	input := map[string]any{
		"id":    1,
		"email": "demo@example.com",
		"parties": []any{
			map[string]any{"name": "Demo", "email": "demo@example.com"},
			map[string]any{"name": "Test", "phone": "+1 555 0100"},
		},
	}
	redacted := OmitDeep(input, "email", "phone")
	encoded, err := json.Marshal(redacted)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"email", "phone", "demo@example.com", "555"} {
		if contains(text, forbidden) {
			t.Fatalf("no debe contener %q: %s", forbidden, text)
		}
	}
	if !contains(text, `"name":"Demo"`) {
		t.Fatalf("debe conservar el nombre: %s", text)
	}
}

func TestTotalsByCurrency(t *testing.T) {
	items := []map[string]any{
		{"currency": "NIO", "amount": 100.0},
		{"currency": "NIO", "amount": 50.25},
		{"currency": "USD", "amount": 10.0},
		{"amount": 99.0},
	}
	totals := TotalsByCurrency(items, "currency", "amount")
	if len(totals) != 2 {
		t.Fatalf("totales = %v", totals)
	}
	if totals[0].Currency != "NIO" || totals[0].Amount != 150.25 || totals[0].Count != 2 {
		t.Fatalf("NIO = %+v", totals[0])
	}
	if totals[1].Currency != "USD" || totals[1].Amount != 10 {
		t.Fatalf("USD = %+v", totals[1])
	}
}

func contains(text, part string) bool {
	for i := 0; i+len(part) <= len(text); i++ {
		if text[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
