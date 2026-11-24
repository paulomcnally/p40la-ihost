package debts

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestDebtBillsList(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/debts/4/bills": {Body: []map[string]any{
			{"id": 10, "status": "pending", "amount": 100.0, "invoice_file": "demo.pdf"},
		}},
		"/api/debt-bills?month=11&year=2026": {Body: []map[string]any{
			{"id": 11, "status": "paid", "amount": 50.0},
		}},
	}}

	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	names := toolkitest.ToolNames(t, session)
	if !names["debt_bills_list"] {
		t.Fatal("falta debt_bills_list")
	}

	byDebt := toolkitest.Call(t, session, "debt_bills_list", map[string]any{"debt_id": 4})
	for _, want := range []string{`"mode": "debt"`, `"count": 1`} {
		if !strings.Contains(byDebt, want) {
			t.Fatalf("modo deuda sin %q: %s", want, byDebt)
		}
	}
	if strings.Contains(byDebt, "invoice_file") {
		t.Fatalf("no debe exponer archivos: %s", byDebt)
	}

	byMonth := toolkitest.Call(t, session, "debt_bills_list", map[string]any{"period": "2026-11"})
	for _, want := range []string{`"mode": "month"`, `"period": "2026-11"`} {
		if !strings.Contains(byMonth, want) {
			t.Fatalf("modo mes sin %q: %s", want, byMonth)
		}
	}
}

func TestDebtBillsRequiresExactlyOneMode(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	out := toolkitest.Call(t, session, "debt_bills_list", map[string]any{})
	if !strings.Contains(out, `"error":"invalid_request"`) {
		t.Fatalf("esperaba invalid_request: %s", out)
	}
}
