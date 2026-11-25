// Package budget implementa las tools MCP del presupuesto (SPEC-163): consulta
// de categorías y cuentas, registro de gastos con deduplicación y consultas del
// mes. Las tools hablan con /api/budget/*; el MCP no accede a SQLite.
package budget

import (
	"math"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

const dateLayout = "2006-01-02"

// Toolset agrupa las dependencias de las tools de presupuesto.
type Toolset struct {
	client api.BudgetAPI
	cfg    config.Config
	now    func() time.Time
}

// New crea un Toolset con el reloj real.
func New(client api.BudgetAPI, cfg config.Config) *Toolset {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	return &Toolset{client: client, cfg: cfg, now: time.Now}
}

// Register monta las tools de presupuesto en el servidor MCP.
func Register(server *mcp.Server, client api.BudgetAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	t.registerLists(server)
	t.registerCreate(server)
}

func (t *Toolset) today() string {
	return t.now().In(t.cfg.Location).Format(dateLayout)
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
			"Las escrituras están deshabilitadas en este servidor MCP.", hint)
		return false, result, output
	}
	return dryRun, nil, nil
}

// normalizeName compara nombres de categoría sin distinguir mayúsculas,
// acentos ni espacios repetidos. No aplica coincidencias parciales.
func normalizeName(value string) string {
	folded := strings.ToLower(strings.Join(strings.Fields(value), " "))
	return accentReplacer.Replace(folded)
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "ä", "a", "â", "a",
	"é", "e", "è", "e", "ë", "e", "ê", "e",
	"í", "i", "ì", "i", "ï", "i", "î", "i",
	"ó", "o", "ò", "o", "ö", "o", "ô", "o",
	"ú", "u", "ù", "u", "ü", "u", "û", "u",
	"ñ", "n", "ç", "c",
)

// round2 redondea a dos decimales para comparar montos.
func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// categoryIndex aplana los grupos en categorías activas con el nombre del grupo.
func categoryIndex(groups []api.CategoryGroup) []api.Category {
	categories := make([]api.Category, 0)
	for _, group := range groups {
		categories = append(categories, group.Categories...)
	}
	return categories
}

func groupName(groups []api.CategoryGroup, categoryID int64) string {
	for _, group := range groups {
		for _, category := range group.Categories {
			if category.ID == categoryID {
				return group.Name
			}
		}
	}
	return ""
}

// dateOnly recorta una fecha a YYYY-MM-DD por si el backend la devuelve con
// hora.
func dateOnly(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= len(dateLayout) {
		return trimmed[:len(dateLayout)]
	}
	return trimmed
}
