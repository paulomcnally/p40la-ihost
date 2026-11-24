package autos

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

func TestProtocolToolsListAndCall(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "p40la-mcp-test", Version: "0.1.0"}, nil)
	Register(server, &fakeAPI{autos: []api.Auto{{ID: 1, Brand: "Marca Demo", Model: "Modelo Demo", Color: "white"}}}, config.Config{
		Timezone: "UTC",
		Location: time.UTC,
	})

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
	for _, want := range []string{
		"autos_list",
		"autos_create",
		"autos_update",
		"autos_maintenance_register",
		"autos_inspection_register",
		"autos_insurance_register",
	} {
		if !names[want] {
			t.Fatalf("falta la tool %q en %v", want, names)
		}
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "autos_list",
		Arguments: map[string]any{"query": "demo"},
	})
	if err != nil {
		t.Fatalf("tools/call list: %v", err)
	}
	if result.IsError || result.StructuredContent == nil {
		t.Fatalf("list devolvió error: %#v", result.Content)
	}

	createResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "autos_create",
		Arguments: map[string]any{
			"year": 2024, "brand": "Marca Demo", "model": "Modelo Demo", "color": "white",
		},
	})
	if err != nil {
		t.Fatalf("tools/call create: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create dry_run devolvió error: %#v", createResult.Content)
	}

	errorResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "autos_create",
		Arguments: map[string]any{"year": 2024, "model": "Modelo Demo", "color": "white"},
	})
	if err != nil {
		t.Fatalf("tools/call create inválido: %v", err)
	}
	if !errorResult.IsError {
		t.Fatal("quiero isError=true sin marca")
	}
	text, ok := errorResult.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "brand") {
		t.Fatalf("contenido = %#v", errorResult.Content[0])
	}
}
