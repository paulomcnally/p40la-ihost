// Package all registra todas las tools MCP del proyecto. Es el único punto de
// entrada que conoce los paquetes de cada dominio.
package all

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/registry"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/alerts"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/automation"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/autos"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/banking"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/bills"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/billviews"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/cards"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/catalog"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/debts"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/entities"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/fleet"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/homemaint"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/pension"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/rental"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/settings"
)

// Register agrega los conjuntos de tools al registry.
func Register(r *registry.Registry, client api.FullAPI, cfg config.Config) {
	r.Add(func(server *mcp.Server) {
		bills.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		autos.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		entities.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		banking.Register(server, client, cfg)
	})

	// SPEC-154 — Ola 1: tools de solo lectura sobre funcionalidades existentes.
	r.Add(func(server *mcp.Server) {
		pension.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		rental.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		debts.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		cards.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		homemaint.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		fleet.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		billviews.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		catalog.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		automation.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		alerts.Register(server, client, cfg)
	})
	r.Add(func(server *mcp.Server) {
		settings.Register(server, client, cfg)
	})
}
