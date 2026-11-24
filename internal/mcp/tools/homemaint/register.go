package homemaint

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

const (
	itemTypeLabor    = "labor"
	itemTypeSupplies = "supplies"
	unitDefault      = "unit"
	maxQueryLen      = 100
	maxItemDescLen   = 200
	maxExistingToday = 50
)

var validAreas = []string{
	"plumbing", "electrical", "roof", "paint", "carpentry",
	"appliances", "garden", "cleaning", "other",
}

var validUnits = []string{
	"unit", "m", "m2", "kg", "lb", "l", "gal", "sack", "box",
}

// writeRequest es la entrada normalizada de las tools de registro.
type writeRequest struct {
	HomeID         int64
	MaintenanceID  *int64
	PerformedAt    string
	Area           string
	Description    string
	Amount         float64
	CurrencyID     int64
	InstitutionID  *int64
	DryRun         *bool
	AllowDuplicate bool
	ItemType       string
	Quantity       float64
	Unit           string
}

type invalidInputError struct {
	code    string
	message string
}

func (e *invalidInputError) Error() string { return e.message }

func invalidInput(format string, args ...any) error {
	return &invalidInputError{code: "invalid_request", message: fmt.Sprintf(format, args...)}
}

func invalidDate(format string, args ...any) error {
	return &invalidInputError{code: "invalid_date", message: fmt.Sprintf(format, args...)}
}

type resolution struct {
	Maintenance *api.HomeMaintenance
	Day         *api.HomeMaintenanceList
	Date        string
	Reused      bool
}

type ambiguousError struct {
	Candidates []api.HomeMaintenance
}

type duplicateMatch struct {
	Maintenance api.HomeMaintenance
	Item        api.HomeMaintenanceItem
}

// registerLine valida, resuelve la reparación, deduplica y crea o agrega la
// línea según dry_run y la compuerta de escritura.
func (t *Toolset) registerLine(ctx context.Context, req writeRequest) (*mcp.CallToolResult, any, error) {
	if err := validateWriteRequest(&req); err != nil {
		return invalidToolError(err)
	}
	dryRun, disabled, output := t.writeMode(req.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	resolved, ambiguous, err := t.resolve(ctx, req)
	if err != nil {
		var invalid *invalidInputError
		if errors.As(err, &invalid) {
			return toolkit.ToolError(invalid.code, invalid.Error(), "")
		}
		return toolkit.APIError(err)
	}
	if ambiguous != nil {
		return ambiguousResult(ambiguous, dryRun)
	}

	duplicate := findDuplicate(resolved.Day, req)
	if duplicate != nil && !req.AllowDuplicate {
		return toolkit.JSONResult(map[string]any{
			"status":   "duplicate",
			"dry_run":  dryRun,
			"existing": duplicateMap(duplicate),
			"hint":     "Ya existe una línea equivalente el mismo día. Usá allow_duplicate=true solo si es un registro distinto.",
		})
	}

	if dryRun {
		return toolkit.JSONResult(map[string]any{
			"status":             "planned",
			"dry_run":            true,
			"reused_maintenance": resolved.Maintenance != nil,
			"planned":            plannedMap(req, resolved),
			"existing_today":     existingToday(resolved.Day, maxExistingToday),
			"duplicate":          duplicateMap(duplicate),
		})
	}

	warnings := make([]string, 0)
	if duplicate != nil {
		warnings = append(warnings, "Se creó un duplicado porque allow_duplicate=true.")
	}
	itemReq := api.HomeMaintenanceItemRequest{
		ItemType:      req.ItemType,
		Description:   req.Description,
		Quantity:      req.Quantity,
		Unit:          req.Unit,
		Amount:        req.Amount,
		CurrencyID:    req.CurrencyID,
		InstitutionID: req.InstitutionID,
	}

	if resolved.Maintenance != nil {
		created, err := t.client.CreateHomeMaintenanceItem(ctx, req.HomeID, resolved.Maintenance.ID, itemReq)
		if err != nil {
			return toolkit.APIError(err)
		}
		result := map[string]any{
			"status":             "created",
			"dry_run":            false,
			"reused_maintenance": true,
			"maintenance":        resolved.Maintenance,
			"item":               created,
		}
		if len(warnings) > 0 {
			result["warnings"] = warnings
		}
		return toolkit.JSONResult(result)
	}

	area := req.Area
	if area == "" {
		area = "other"
	}
	created, err := t.client.CreateHomeMaintenance(ctx, req.HomeID, api.HomeMaintenanceRequest{
		PerformedAt:   resolved.Date,
		Kind:          "corrective",
		Area:          area,
		InstitutionID: req.InstitutionID,
		Description:   req.Description,
		Item:          &itemReq,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	result := map[string]any{
		"status":             "created",
		"dry_run":            false,
		"reused_maintenance": false,
		"maintenance":        created,
	}
	if len(created.Items) > 0 {
		result["item"] = created.Items[len(created.Items)-1]
	} else {
		warnings = append(warnings, "El mantenimiento se creó, pero la respuesta no incluyó la línea.")
	}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}
	return toolkit.JSONResult(result)
}

func (t *Toolset) resolve(ctx context.Context, req writeRequest) (*resolution, *ambiguousError, error) {
	if req.MaintenanceID != nil {
		maintenance, err := t.client.GetHomeMaintenance(ctx, req.HomeID, *req.MaintenanceID)
		if err != nil {
			return nil, nil, err
		}
		date := maintenance.PerformedAt
		if req.PerformedAt != "" {
			validated, err := t.validateDate(req.PerformedAt)
			if err != nil {
				return nil, nil, err
			}
			if validated != date {
				return nil, nil, invalidInput("performed_at no coincide con la fecha del mantenimiento")
			}
		}
		day, err := t.dayList(ctx, req.HomeID, date)
		if err != nil {
			day = &api.HomeMaintenanceList{Items: []api.HomeMaintenance{*maintenance}}
		} else if !containsMaintenance(day.Items, maintenance.ID) {
			day.Items = append(day.Items, *maintenance)
		}
		return &resolution{Maintenance: maintenance, Day: day, Date: date, Reused: true}, nil, nil
	}

	if req.PerformedAt == "" {
		return nil, nil, invalidInput("performed_at es obligatoria cuando no hay maintenance_id")
	}
	date, err := t.validateDate(req.PerformedAt)
	if err != nil {
		return nil, nil, err
	}
	day, err := t.dayList(ctx, req.HomeID, date)
	if err != nil {
		return nil, nil, err
	}
	candidates := filterByArea(day.Items, req.Area)
	if len(candidates) > 1 {
		return nil, &ambiguousError{Candidates: candidates}, nil
	}
	if len(candidates) == 1 {
		maintenance := candidates[0]
		return &resolution{Maintenance: &maintenance, Day: day, Date: date, Reused: true}, nil, nil
	}
	return &resolution{Day: day, Date: date}, nil, nil
}

func (t *Toolset) dayList(ctx context.Context, homeID int64, date string) (*api.HomeMaintenanceList, error) {
	return t.client.ListHomeMaintenances(ctx, homeID, api.HomeMaintenanceFilter{
		Date:         date,
		IncludeItems: true,
	})
}

func (t *Toolset) validateDate(value string) (string, error) {
	parsed, err := time.Parse(dateLayout, strings.TrimSpace(value))
	if err != nil {
		return "", invalidDate("la fecha debe tener formato YYYY-MM-DD")
	}
	today, err := time.Parse(dateLayout, t.today())
	if err != nil {
		return "", invalidDate("no se pudo resolver la fecha actual")
	}
	if parsed.After(today) {
		return "", invalidDate("la fecha no puede ser futura")
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

func invalidToolError(err error) (*mcp.CallToolResult, any, error) {
	var invalid *invalidInputError
	if errors.As(err, &invalid) {
		return toolkit.ToolError(invalid.code, invalid.Error(), "")
	}
	return toolkit.ToolError("invalid_request", err.Error(), "")
}

func validateWriteRequest(req *writeRequest) error {
	if req.HomeID <= 0 {
		return invalidInput("home_id debe ser mayor que cero")
	}
	if req.MaintenanceID != nil && *req.MaintenanceID <= 0 {
		req.MaintenanceID = nil
	}
	if req.InstitutionID != nil && *req.InstitutionID <= 0 {
		req.InstitutionID = nil
	}
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		return invalidInput("description es obligatoria")
	}
	if utf8.RuneCountInString(req.Description) > maxItemDescLen {
		return invalidInput("description no puede superar los %d caracteres", maxItemDescLen)
	}
	if math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) {
		return invalidInput("amount no es válido")
	}
	if req.Amount < 0 {
		return invalidInput("amount no puede ser negativo")
	}
	req.Amount = round2(req.Amount)
	if req.CurrencyID <= 0 {
		return invalidInput("currency_id es obligatorio")
	}
	req.Area = strings.TrimSpace(req.Area)
	if req.Area != "" && !contains(validAreas, req.Area) {
		return invalidInput("area debe ser plumbing, electrical, roof, paint, carpentry, appliances, garden, cleaning u other")
	}
	req.PerformedAt = strings.TrimSpace(req.PerformedAt)

	switch req.ItemType {
	case itemTypeLabor:
		req.Quantity = 1
		req.Unit = unitDefault
	case itemTypeSupplies:
		if req.Quantity == 0 {
			req.Quantity = 1
		}
		if math.IsNaN(req.Quantity) || math.IsInf(req.Quantity, 0) || req.Quantity < 0 {
			return invalidInput("quantity debe ser mayor que cero")
		}
		req.Unit = strings.TrimSpace(req.Unit)
		if req.Unit == "" {
			req.Unit = unitDefault
		}
		if !contains(validUnits, req.Unit) {
			return invalidInput("unit debe ser unit, m, m2, kg, lb, l, gal, sack o box")
		}
	default:
		return invalidInput("item_type debe ser labor o supplies")
	}
	return nil
}

func filterByArea(items []api.HomeMaintenance, area string) []api.HomeMaintenance {
	if area == "" {
		return items
	}
	out := make([]api.HomeMaintenance, 0, len(items))
	for _, item := range items {
		if item.Area == area {
			out = append(out, item)
		}
	}
	return out
}

func containsMaintenance(items []api.HomeMaintenance, id int64) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func findDuplicate(day *api.HomeMaintenanceList, req writeRequest) *duplicateMatch {
	if day == nil {
		return nil
	}
	for _, maintenance := range day.Items {
		for _, item := range maintenance.Items {
			if itemMatches(item, req) {
				return &duplicateMatch{Maintenance: maintenance, Item: item}
			}
		}
	}
	return nil
}

func itemMatches(item api.HomeMaintenanceItem, req writeRequest) bool {
	if item.ItemType != req.ItemType {
		return false
	}
	if normalize(item.Description) != normalize(req.Description) {
		return false
	}
	if round2(item.Amount) != round2(req.Amount) {
		return false
	}
	if item.CurrencyID != req.CurrencyID {
		return false
	}
	if !sameID(item.InstitutionID, req.InstitutionID) {
		return false
	}
	if req.ItemType == itemTypeSupplies {
		if round2(item.Quantity) != round2(req.Quantity) {
			return false
		}
		if item.Unit != req.Unit {
			return false
		}
	}
	return true
}

func sameID(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func normalize(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func duplicateMap(duplicate *duplicateMatch) any {
	if duplicate == nil {
		return nil
	}
	return map[string]any{
		"maintenance_id": duplicate.Maintenance.ID,
		"performed_at":   duplicate.Maintenance.PerformedAt,
		"area":           duplicate.Maintenance.Area,
		"item":           duplicate.Item,
	}
}

func ambiguousResult(ambiguous *ambiguousError, dryRun bool) (*mcp.CallToolResult, any, error) {
	candidates := make([]map[string]any, 0, len(ambiguous.Candidates))
	for _, candidate := range ambiguous.Candidates {
		candidates = append(candidates, map[string]any{
			"maintenance_id":   candidate.ID,
			"performed_at":     candidate.PerformedAt,
			"area":             candidate.Area,
			"description":      candidate.Description,
			"institution_name": candidate.InstitutionName,
		})
	}
	return toolkit.JSONResult(map[string]any{
		"status":     "ambiguous_maintenance",
		"dry_run":    dryRun,
		"candidates": candidates,
		"hint":       "Hay más de una reparación para esa fecha y área. Pasá maintenance_id para elegir una.",
	})
}

func plannedMap(req writeRequest, resolved *resolution) map[string]any {
	planned := map[string]any{
		"performed_at": resolved.Date,
		"item_type":    req.ItemType,
		"description":  req.Description,
		"quantity":     req.Quantity,
		"unit":         req.Unit,
		"amount":       req.Amount,
		"currency_id":  req.CurrencyID,
	}
	if req.InstitutionID != nil {
		planned["institution_id"] = *req.InstitutionID
	}
	if resolved.Maintenance != nil {
		planned["action"] = "append_item"
		planned["maintenance_id"] = resolved.Maintenance.ID
	} else {
		planned["action"] = "create_maintenance_with_item"
		area := req.Area
		if area == "" {
			area = "other"
		}
		planned["area"] = area
		planned["kind"] = "corrective"
		planned["maintenance_description"] = req.Description
	}
	return planned
}

func existingToday(day *api.HomeMaintenanceList, limit int) []map[string]any {
	if day == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0)
	for _, maintenance := range day.Items {
		for _, item := range maintenance.Items {
			if len(out) >= limit {
				return out
			}
			row := map[string]any{
				"maintenance_id":          maintenance.ID,
				"performed_at":            maintenance.PerformedAt,
				"area":                    maintenance.Area,
				"maintenance_description": maintenance.Description,
				"item_id":                 item.ID,
				"item_type":               item.ItemType,
				"description":             item.Description,
				"quantity":                item.Quantity,
				"unit":                    item.Unit,
				"amount":                  item.Amount,
				"currency_id":             item.CurrencyID,
				"currency_code":           item.CurrencyCode,
			}
			if item.InstitutionID != nil {
				row["institution_id"] = *item.InstitutionID
			}
			if item.InstitutionName != "" {
				row["institution_name"] = item.InstitutionName
			}
			out = append(out, row)
		}
	}
	return out
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func quantityValue(value *float64) float64 {
	if value == nil {
		return 1
	}
	return *value
}
