// Package registry agrupa el montaje de tools del servidor MCP.
package registry

import "github.com/modelcontextprotocol/go-sdk/mcp"

// RegisterFunc monta un conjunto de tools en un servidor MCP.
type RegisterFunc func(*mcp.Server)

// Registry mantiene los conjuntos de tools para que cmd/mcp-server no conozca
// los paquetes de cada dominio. Agregar un dominio nuevo es un Add más.
type Registry struct {
	funcs []RegisterFunc
}

// Add registra un conjunto de tools.
func (r *Registry) Add(fn RegisterFunc) {
	if fn == nil {
		return
	}
	r.funcs = append(r.funcs, fn)
}

// Mount monta todos los conjuntos registrados.
func (r *Registry) Mount(server *mcp.Server) {
	for _, fn := range r.funcs {
		fn(server)
	}
}
