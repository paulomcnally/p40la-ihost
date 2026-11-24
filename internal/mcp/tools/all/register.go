// Package all registra todas las tools MCP del proyecto. Es el único punto de
// entrada que conoce los paquetes de cada dominio.
package all

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/registry"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/autos"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/bills"
)

// Register agrega los conjuntos de tools al registry.
func Register(r *registry.Registry, client api.FullAPI, cfg config.Config) {
	r.Add(func(server *mcp.Server) {
		bills.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		autos.Register(server, client, cfg)
	})
}
