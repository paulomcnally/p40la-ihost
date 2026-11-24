// Package autos implementa las tools MCP de autos, mantenimientos,
// inspecciones y seguros de p40la-ihost.
package autos

import (
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

// Toolset agrupa las dependencias de las tools de autos.
type Toolset struct {
	client api.AutosAPI
	cfg    config.Config
	now    func() time.Time
}

// New crea un Toolset con el reloj real.
func New(client api.AutosAPI, cfg config.Config) *Toolset {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	return &Toolset{client: client, cfg: cfg, now: time.Now}
}

// Register monta las tools de autos en el servidor MCP.
func Register(server *mcp.Server, client api.AutosAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	destructive := false
	openWorld := false
	idempotent := true
	notIdempotent := false

	mcp.AddTool(server, &mcp.Tool{
		Name:        "autos_list",
		Description: "Lista los autos registrados. Permite filtrar por texto en marca, modelo, placa o VIN, o pedir un auto por ID. Solo lectura; no modifica datos.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.list)

	mcp.AddTool(server, &mcp.Tool{
		Name: "autos_create",
		Description: "Crea un auto con año, marca, modelo, color, ícono y datos opcionales de motor, chasis, VIN y placa. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true. " +
			"Mostrá el resumen al usuario y pedí confirmación antes de ejecutar con dry_run=false.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.create)

	mcp.AddTool(server, &mcp.Tool{
		Name: "autos_update",
		Description: "Actualiza los datos de un auto existente. PUT reemplaza el registro, por eso se envían todos los campos; " +
			"motor, chasis, VIN, placa e ícono pueden ir vacíos. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.update)

	mcp.AddTool(server, &mcp.Tool{
		Name: "autos_maintenance_register",
		Description: "Registra un mantenimiento de un auto con fecha, kilometraje, taller, moneda y descripción; " +
			"acepta líneas opcionales de mano de obra e insumos. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.maintenanceRegister)

	mcp.AddTool(server, &mcp.Tool{
		Name: "autos_inspection_register",
		Description: "Registra una inspección mecánica de un auto con fecha, año del ciclo, centro de inspección, resultado y notas. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.inspectionRegister)

	mcp.AddTool(server, &mcp.Tool{
		Name: "autos_insurance_register",
		Description: "Asocia un seguro a un auto con servicio, tipo de cobertura, póliza, número de aseguradora y certificado opcional. " +
			"Verifica que el servicio esté disponible para el auto. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.insuranceRegister)
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
		result, output, _ := toolError("writes_disabled",
			"Las escrituras están deshabilitadas en este servidor MCP.",
			hint)
		return false, result, output
	}
	return dryRun, nil, nil
}

// today devuelve la fecha actual en la zona configurada, sin hora.
func (t *Toolset) today() string {
	return t.now().In(t.cfg.Location).Format("2006-01-02")
}
