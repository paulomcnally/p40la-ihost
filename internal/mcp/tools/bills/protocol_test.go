package bills

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestProtocolToolsListAndCall(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "p40la-mcp-test", Version: "0.1.0"}, nil)
	Register(server, &fakeAPI{}, testConfig())

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("conectar server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("conectar cliente: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"bills_list", "bills_generate", "bills_pay"} {
		if !names[want] {
			t.Fatalf("falta la tool %q en %v", want, names)
		}
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "bills_generate",
		Arguments: map[string]any{"service_id": 1, "from": time.Now().Format("2006-01")},
	})
	if err != nil {
		t.Fatalf("tools/call generate: %v", err)
	}
	if result.IsError {
		t.Fatalf("tools/call generate devolvió error: %#v", result.Content)
	}
	if result.StructuredContent == nil {
		t.Fatal("quiero structuredContent en la respuesta")
	}

	errorResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "bills_pay",
		Arguments: map[string]any{"service_id": 1},
	})
	if err != nil {
		t.Fatalf("tools/call pay: %v", err)
	}
	if !errorResult.IsError {
		t.Fatal("quiero isError=true sin selector de pago")
	}
	if len(errorResult.Content) == 0 {
		t.Fatal("quiero contenido de error")
	}
	text, ok := errorResult.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "invalid_selector") {
		t.Fatalf("contenido = %#v", errorResult.Content[0])
	}
}
