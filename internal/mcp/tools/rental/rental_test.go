package rental

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestRentalTools(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/rental/leases?home_id=2&status=active": {Body: []map[string]any{
			{"id": 5, "status": "active", "tenant_email": "demo@example.com"},
		}},
		"/api/rental/leases/5":         {Body: map[string]any{"id": 5, "status": "active", "email": "demo@example.com"}},
		"/api/rental/leases/5/balance": {Body: map[string]any{"charged": 100.0, "paid": 40.0}},
		"/api/rental/homes/3": {Body: map[string]any{
			"home_id": 3, "state": "rented",
			"reminders": []any{map[string]any{"id": 1, "extra_emails": "demo@example.com"}},
		}},
		"/api/rental/portfolio/summary?month=11&year=2026": {Body: map[string]any{
			"income": 1000.0, "expenses": 200.0,
		}},
	}}

	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	names := toolkitest.ToolNames(t, session)
	for _, want := range []string{
		"rental_leases_list", "rental_lease_get", "rental_home_status", "rental_portfolio_summary",
	} {
		if !names[want] {
			t.Fatalf("falta la tool %q", want)
		}
	}

	leases := toolkitest.Call(t, session, "rental_leases_list", map[string]any{
		"status": "active", "home_id": 2,
	})
	if !strings.Contains(leases, `"count": 1`) {
		t.Fatalf("listado inesperado: %s", leases)
	}
	if strings.Contains(leases, "tenant_email") {
		t.Fatalf("no debe exponer emails: %s", leases)
	}

	lease := toolkitest.Call(t, session, "rental_lease_get", map[string]any{"lease_id": 5})
	for _, want := range []string{`"charged": 100`, `"paid": 40`} {
		if !strings.Contains(lease, want) {
			t.Fatalf("detalle sin %q: %s", want, lease)
		}
	}

	status := toolkitest.Call(t, session, "rental_home_status", map[string]any{"home_id": 3})
	if !strings.Contains(status, `"state": "rented"`) {
		t.Fatalf("estado inesperado: %s", status)
	}
	if strings.Contains(status, "extra_emails") {
		t.Fatalf("no debe exponer extra_emails: %s", status)
	}

	portfolio := toolkitest.Call(t, session, "rental_portfolio_summary", map[string]any{"period": "2026-11"})
	if !strings.Contains(portfolio, `"income": 1000`) {
		t.Fatalf("portfolio inesperado: %s", portfolio)
	}
}

func TestRentalRejectsBadInput(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	out := toolkitest.Call(t, session, "rental_lease_get", map[string]any{"lease_id": 0})
	if !strings.Contains(out, `"error":"invalid_request"`) {
		t.Fatalf("esperaba invalid_request: %s", out)
	}
}
