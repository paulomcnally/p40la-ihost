// Package bills implementa las tools MCP de facturas de p40la-ihost.
package bills

import (
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de facturas.
type Toolset struct {
	client api.API
	cfg    config.Config
	now    func() time.Time
}

// New crea un Toolset con el reloj real.
func New(client api.API, cfg config.Config) *Toolset {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	return &Toolset{client: client, cfg: cfg, now: time.Now}
}

// Register monta las tres tools de facturas en el servidor MCP.
func Register(server *mcp.Server, client api.API, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	destructive := false
	openWorld := false

	mcp.AddTool(server, &mcp.Tool{
		Name:        "bills_list",
		Description: "Lista las facturas activas de un servicio con filtros opcionales de período (YYYY-MM) y estado. Solo lectura; no modifica datos.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.list)

	mcp.AddTool(server, &mcp.Tool{
		Name: "bills_generate",
		Description: "Genera facturas de un servicio para un rango inclusivo de meses (máximo 36, sin períodos futuros). " +
			"Omite períodos que ya tienen factura y reporta los slots con factura eliminada. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true. " +
			"Mostrá el resumen al usuario y pedí confirmación antes de ejecutar con dry_run=false.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: true, OpenWorldHint: &openWorld},
	}, t.generate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "bills_pay",
		Description: "Paga facturas de un servicio con exactamente un selector: bill_ids, month (YYYY-MM), from+to (YYYY-MM) o all_pending_up_to (YYYY-MM). " +
			"Omite las facturas ya pagadas y los períodos futuros. paid_at por defecto es hoy. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true. " +
			"Mostrá el resumen al usuario y pedí confirmación antes de ejecutar con dry_run=false.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: true, OpenWorldHint: &openWorld},
	}, t.pay)
}

// writeMode resuelve dry_run (default true) y aplica la compuerta de escritura.
func (t *Toolset) writeMode(requested *bool) (bool, *mcp.CallToolResult, any) {
	dryRun := true
	if requested != nil {
		dryRun = *requested
	}
	if !dryRun && !t.cfg.WritesEnabled() {
		hint := "Definí P40LA_MCP_ALLOW_WRITES=true o repetí con dry_run=true para previsualizar."
		if t.cfg.IsHTTP() {
			hint = "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true, o repetí con dry_run=true para previsualizar."
		}
		result, output, _ := toolkit.ToolError("writes_disabled",
			"Las escrituras están deshabilitadas en este servidor MCP.",
			hint)
		return false, result, output
	}
	return dryRun, nil, nil
}

func (t *Toolset) currentPeriod() period {
	return currentPeriod(t.now().In(t.cfg.Location))
}

func (t *Toolset) today() string {
	return t.now().In(t.cfg.Location).Format("2006-01-02")
}
