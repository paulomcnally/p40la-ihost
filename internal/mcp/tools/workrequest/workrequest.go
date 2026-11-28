// Package workrequest implementa las tools MCP de reportes de problemas y
// mantenimiento por vivienda: consulta, alta, edición, conversión a reparación
// y descarte/reactivación.
package workrequest

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

const (
	dateLayout        = "2006-01-02"
	maxTitleLen       = 200
	maxDescriptionLen = 2000
)

var validStatuses = []string{"reported", "converted", "dismissed"}

var validCategories = []string{"repair", "improvement", "maintenance", "other"}

var validPriorities = []string{"low", "normal", "high"}

var validKinds = []string{"corrective", "preventive"}

var validAreas = []string{
	"plumbing", "electrical", "roof", "paint", "carpentry",
	"appliances", "garden", "cleaning", "other",
}

// Toolset agrupa las dependencias de las tools de reportes.
type Toolset struct {
	client api.HomeWorkRequestAPI
	cfg    config.Config
	now    func() time.Time
}

// New crea un Toolset con el reloj real.
func New(client api.HomeWorkRequestAPI, cfg config.Config) *Toolset {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	return &Toolset{client: client, cfg: cfg, now: time.Now}
}

// Register monta las tools de reportes en el servidor MCP.
func Register(server *mcp.Server, client api.HomeWorkRequestAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	t.registerList(server)
	t.registerGet(server)
	t.registerCreate(server)
	t.registerUpdate(server)
	t.registerConvert(server)
	t.registerStatus(server)
}

// SetClock permite fijar el reloj en tests.
func (t *Toolset) SetClock(now func() time.Time) { t.now = now }

func (t *Toolset) today() string {
	return t.now().In(t.cfg.Location).Format(dateLayout)
}

func (t *Toolset) validateDate(value string) (string, error) {
	parsed, err := time.Parse(dateLayout, strings.TrimSpace(value))
	if err != nil {
		return "", invalidInput("reported_at debe tener formato YYYY-MM-DD")
	}
	today, err := time.Parse(dateLayout, t.today())
	if err != nil {
		return "", invalidInput("no se pudo resolver la fecha actual")
	}
	if parsed.After(today) {
		return "", invalidInput("reported_at no puede ser futura")
	}
	return parsed.Format(dateLayout), nil
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

type invalidInputError struct {
	code    string
	message string
}

func (e *invalidInputError) Error() string { return e.message }

func invalidInput(format string, args ...any) error {
	return &invalidInputError{code: "invalid_request", message: fmt.Sprintf(format, args...)}
}

func invalidToolError(err error) (*mcp.CallToolResult, any, error) {
	var invalid *invalidInputError
	if errors.As(err, &invalid) {
		return toolkit.ToolError(invalid.code, invalid.Error(), "")
	}
	return toolkit.ToolError("invalid_request", err.Error(), "")
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func workRequestFrom(current *api.HomeWorkRequest) api.HomeWorkRequestRequest {
	if current == nil {
		return api.HomeWorkRequestRequest{}
	}
	return api.HomeWorkRequestRequest{
		Title:       current.Title,
		Description: current.Description,
		Category:    current.Category,
		Priority:    current.Priority,
		ReportedAt:  current.ReportedAt,
	}
}

func requestSnapshot(request api.HomeWorkRequestRequest) map[string]any {
	return map[string]any{
		"title":       request.Title,
		"description": request.Description,
		"category":    request.Category,
		"priority":    request.Priority,
		"reported_at": request.ReportedAt,
	}
}

// changedFields devuelve los campos editables que difieren, en orden estable.
func changedFields(current, proposed api.HomeWorkRequestRequest) []string {
	changed := make([]string, 0)
	if current.Title != proposed.Title {
		changed = append(changed, "title")
	}
	if current.Description != proposed.Description {
		changed = append(changed, "description")
	}
	if current.Category != proposed.Category {
		changed = append(changed, "category")
	}
	if current.Priority != proposed.Priority {
		changed = append(changed, "priority")
	}
	if current.ReportedAt != proposed.ReportedAt {
		changed = append(changed, "reported_at")
	}
	return changed
}
