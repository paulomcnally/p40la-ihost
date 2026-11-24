package entities

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// ServicesCreateInput es la entrada de services_create.
type ServicesCreateInput struct {
	Name            string   `json:"name" jsonschema:"Nombre del servicio (obligatorio)"`
	Home            string   `json:"home,omitempty" jsonschema:"Nombre del hogar (uno de home o home_id)"`
	HomeID          *int64   `json:"home_id,omitempty" jsonschema:"ID del hogar (uno de home o home_id)"`
	Currency        string   `json:"currency,omitempty" jsonschema:"Código o nombre de la moneda (uno de currency o currency_id)"`
	CurrencyID      *int64   `json:"currency_id,omitempty" jsonschema:"ID de la moneda (uno de currency o currency_id)"`
	Institution     string   `json:"institution,omitempty" jsonschema:"Nombre de la institución (opcional)"`
	InstitutionID   *int64   `json:"institution_id,omitempty" jsonschema:"ID de la institución (opcional; gana sobre institution)"`
	Frequency       string   `json:"frequency,omitempty" jsonschema:"monthly (default) o yearly"`
	SuggestedAmount *float64 `json:"suggested_amount,omitempty" jsonschema:"Monto sugerido; default 0"`
	IconKey         string   `json:"icon_key,omitempty" jsonschema:"Clave de ícono; default other"`
	BillingType     string   `json:"billing_type,omitempty" jsonschema:"fixed o variable (default)"`
	BillingDay      *int     `json:"billing_day,omitempty" jsonschema:"Día de facturación 1..31 (opcional)"`
	Active          *bool    `json:"active,omitempty" jsonschema:"Activo; default true. Con true se genera la factura del período actual"`
	AutoGenerate    bool     `json:"auto_generate,omitempty" jsonschema:"Generación automática (default false)"`
	IsRecurring     bool     `json:"is_recurring,omitempty" jsonschema:"Recurrente (default false)"`
	StartDate       *string  `json:"start_date,omitempty" jsonschema:"Fecha de inicio YYYY-MM-DD (opcional)"`
	EndDate         *string  `json:"end_date,omitempty" jsonschema:"Fecha de fin YYYY-MM-DD (opcional)"`
}

// ServicesCreateOutput es el resultado de services_create.
type ServicesCreateOutput struct {
	api.Service
	CurrentBillGenerated bool `json:"current_bill_generated"`
}

var (
	serviceFrequencies = map[string]bool{"monthly": true, "yearly": true}
	billingTypes       = map[string]bool{"fixed": true, "variable": true}
)

func (t *Toolset) servicesCreate(ctx context.Context, _ *mcp.CallToolRequest, in ServicesCreateInput) (*mcp.CallToolResult, any, error) {
	if result, output, err := t.checkWrites(); result != nil {
		return result, output, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return toolkit.ToolError("invalid_request", "name es obligatorio.", "")
	}

	hasHomeID := in.HomeID != nil && *in.HomeID > 0
	hasHomeName := strings.TrimSpace(in.Home) != ""
	if hasHomeID == hasHomeName {
		return toolkit.ToolError("invalid_request", "Indicá exactamente uno: home o home_id.", "")
	}
	hasCurrencyID := in.CurrencyID != nil && *in.CurrencyID > 0
	hasCurrencyName := strings.TrimSpace(in.Currency) != ""
	if hasCurrencyID == hasCurrencyName {
		return toolkit.ToolError("invalid_request", "Indicá exactamente uno: currency o currency_id.", "")
	}

	frequency := strings.ToLower(strings.TrimSpace(in.Frequency))
	if frequency == "" {
		frequency = "monthly"
	}
	if !serviceFrequencies[frequency] {
		return toolkit.ToolError("invalid_value", "frequency debe ser monthly o yearly.",
			"Valores válidos: monthly, yearly.")
	}
	billingType := strings.ToLower(strings.TrimSpace(in.BillingType))
	if billingType == "" {
		billingType = "variable"
	}
	if !billingTypes[billingType] {
		return toolkit.ToolError("invalid_value", "billing_type debe ser fixed o variable.",
			"Valores válidos: fixed, variable.")
	}
	if in.BillingDay != nil && (*in.BillingDay < 1 || *in.BillingDay > 31) {
		return toolkit.ToolError("invalid_value", "billing_day debe estar entre 1 y 31.", "")
	}
	amount := 0.0
	if in.SuggestedAmount != nil {
		amount = *in.SuggestedAmount
	}
	if amount < 0 {
		return toolkit.ToolError("invalid_value", "suggested_amount no puede ser negativo.", "")
	}
	if err := validateDate("start_date", in.StartDate); err != nil {
		return toolkit.ToolError("invalid_value", err.Error(), "Formato: YYYY-MM-DD.")
	}
	if err := validateDate("end_date", in.EndDate); err != nil {
		return toolkit.ToolError("invalid_value", err.Error(), "Formato: YYYY-MM-DD.")
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	iconKey := strings.TrimSpace(in.IconKey)
	if iconKey == "" {
		iconKey = "other"
	}

	home, err := t.resolveHome(ctx, in.HomeID, in.Home)
	if err != nil {
		return toResult(err)
	}
	currency, err := t.resolveCurrency(ctx, in.CurrencyID, in.Currency)
	if err != nil {
		return toResult(err)
	}
	var institution *api.Institution
	if in.InstitutionID != nil {
		institution, err = t.resolveInstitution(ctx, in.InstitutionID, "")
		if err != nil {
			return toResult(err)
		}
	} else if strings.TrimSpace(in.Institution) != "" {
		institution, err = t.resolveInstitution(ctx, nil, in.Institution)
		if err != nil {
			return toResult(err)
		}
	}

	services, err := t.client.ListServices(ctx, &home.ID)
	if err != nil {
		return toolkit.APIError(err)
	}
	if matches := findByName(services, func(s api.Service) string { return s.Name }, name); len(matches) > 0 {
		existing := matches[0]
		return toolkit.ToolError("already_exists", "Ya existe un servicio con ese nombre en el hogar.",
			fmt.Sprintf("Usá el existente: id=%d, name=%q, home_id=%d", existing.ID, existing.Name, home.ID))
	}

	req := api.ServiceCreateRequest{
		HomeID:          home.ID,
		Name:            name,
		CurrencyID:      currency.ID,
		Frequency:       frequency,
		SuggestedAmount: amount,
		Active:          active,
		IconKey:         iconKey,
		BillingType:     billingType,
		BillingDay:      in.BillingDay,
		AutoGenerate:    in.AutoGenerate,
		IsRecurring:     in.IsRecurring,
		StartDate:       trimPtr(in.StartDate),
		EndDate:         trimPtr(in.EndDate),
	}
	if institution != nil {
		req.InstitutionID = &institution.ID
		req.Institution = institution.Name
	}

	created, err := t.client.CreateService(ctx, req)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(ServicesCreateOutput{
		Service:              *created,
		CurrentBillGenerated: created.Active,
	})
}

func validateDate(field string, raw *string) error {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%s debe tener formato YYYY-MM-DD", field)
	}
	return nil
}

func trimPtr(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}
