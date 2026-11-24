// Package toolkitest provee dobles de prueba para las tools MCP de lectura.
package toolkitest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Response es una respuesta simulada de la API.
type Response struct {
	Body any
	Err  error
}

// Fake implementa api.ReadAPI devolviendo respuestas por ruta.
type Fake struct {
	Responses map[string]Response
	Calls     []string
}

// ReadJSON devuelve la respuesta registrada para la ruta.
func (f *Fake) ReadJSON(_ context.Context, path string, out any) error {
	f.Calls = append(f.Calls, path)
	response, ok := f.Responses[path]
	if !ok {
		return fmt.Errorf("ruta no esperada: %s", path)
	}
	if response.Err != nil {
		return response.Err
	}
	encoded, err := json.Marshal(response.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, out)
}

// NewServer monta las tools de un paquete y conecta un cliente en memoria.
func NewServer(t *testing.T, register func(*mcp.Server)) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "p40la-mcp-test", Version: "0.1.0"}, nil)
	register(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("conectar server: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("conectar cliente: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// ToolNames devuelve los nombres de tools publicados.
func ToolNames(t *testing.T, session *mcp.ClientSession) map[string]bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	return names
}

// Call invoca una tool y devuelve el texto de la respuesta.
func Call(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if len(result.Content) == 0 {
		t.Fatalf("%s sin contenido", name)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("%s: contenido inesperado %T", name, result.Content[0])
	}
	return text.Text
}
