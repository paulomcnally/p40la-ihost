// Package entities implementa las tools MCP de alta de hogares, instituciones,
// bancos y servicios (SPEC-146).
package entities

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de entidades.
type Toolset struct {
	client api.API
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.API, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las cuatro tools de alta en el servidor MCP.
func Register(server *mcp.Server, client api.API, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	destructive := true
	openWorld := false
	idempotent := true

	mcp.AddTool(server, &mcp.Tool{
		Name: "homes_create",
		Description: "Crea un hogar. Requiere name; address y usage son opcionales. " +
			"usage: rental (default), inhabited o unused. " +
			"Si ya existe un hogar con el mismo nombre devuelve already_exists con el id existente.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, OpenWorldHint: &openWorld},
	}, t.homesCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "homes_update",
		Description: "Edita un hogar existente. Identificalo con home_id o home (nombre); si vienen ambos gana home_id. " +
			"name, address y usage son opcionales: lo que se omite conserva su valor actual. " +
			"address vacío limpia la dirección. usage acepta rental, inhabited o unused. " +
			"No se puede cambiar el uso fuera de rental si hay un contrato o reserva activa " +
			"(home_has_active_rental). Si el nombre nuevo ya existe en otro hogar devuelve already_exists. " +
			"Devuelve el hogar actualizado.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.homesUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "institutions_create",
		Description: "Crea una institución/proveedor. Requiere name; category_id o category son opcionales. " +
			"category acepta la clave o el nombre de la categoría. " +
			"Si ya existe una institución con el mismo nombre devuelve already_exists.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, OpenWorldHint: &openWorld},
	}, t.institutionsCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "banks_create",
		Description: "Crea un banco. Requiere name; short_code, country ISO-2, notes y active son opcionales. " +
			"Si ya existe un banco con el mismo nombre devuelve already_exists.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, OpenWorldHint: &openWorld},
	}, t.banksCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "entities_search",
		Description: "Busca servicios, hogares, bancos o instituciones por nombre y devuelve sus IDs. " +
			"No distingue mayúsculas ni acentos; ordena exacto, prefijo y contenido. " +
			"entity: service, home, bank o institution. query vacía lista las primeras entidades. " +
			"Para service, home/home_id filtra por hogar. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.entitiesSearch)

	mcp.AddTool(server, &mcp.Tool{
		Name: "services_create",
		Description: "Crea un servicio. Requiere name y un hogar (home o home_id) y una moneda (currency o currency_id). " +
			"institution/institution_id, frequency, suggested_amount, icon_key, billing_type, billing_day, active, " +
			"auto_generate, is_recurring, start_date y end_date son opcionales. " +
			"Con active=true (default) el backend genera la factura del período actual. " +
			"Si ya existe un servicio con el mismo nombre en el hogar devuelve already_exists.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, OpenWorldHint: &openWorld},
	}, t.servicesCreate)
}

// checkWrites aplica la compuerta de escritura antes de cualquier llamada.
func (t *Toolset) checkWrites() (*mcp.CallToolResult, any, error) {
	if t.cfg.WritesEnabled() {
		return nil, nil, nil
	}
	hint := "Definí P40LA_MCP_ALLOW_WRITES=true para ejecutar cambios."
	if t.cfg.IsHTTP() {
		hint = "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true para ejecutar altas."
	}
	return toolkit.ToolError("writes_disabled", "Las escrituras están deshabilitadas en este servidor MCP.", hint)
}
