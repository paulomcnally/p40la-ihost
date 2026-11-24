package cards

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestCardStatements(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/credit-cards/7/statements": {Body: []map[string]any{
			{"id": 1, "status": "unpaid", "new_balance": 500.0, "transaction_count": 3},
		}},
		"/api/credit-card-statements/9": {Body: map[string]any{
			"id": 9, "status": "unpaid", "reconciliation": map[string]any{"balanced": true},
		}},
	}}

	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	names := toolkitest.ToolNames(t, session)
	for _, want := range []string{"credit_card_statements_list", "credit_card_statements_get"} {
		if !names[want] {
			t.Fatalf("falta la tool %q", want)
		}
	}

	list := toolkitest.Call(t, session, "credit_card_statements_list", map[string]any{"card_id": 7})
	if !strings.Contains(list, `"count": 1`) {
		t.Fatalf("listado inesperado: %s", list)
	}

	statement := toolkitest.Call(t, session, "credit_card_statements_get", map[string]any{"statement_id": 9})
	if !strings.Contains(statement, `"balanced": true`) {
		t.Fatalf("detalle inesperado: %s", statement)
	}
}
