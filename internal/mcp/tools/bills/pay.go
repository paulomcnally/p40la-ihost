package bills

import (
	"context"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// PayInput es la entrada de bills_pay.
type PayInput struct {
	ServiceID        int64   `json:"service_id" jsonschema:"ID del servicio (obligatorio)"`
	BillIDs          []int64 `json:"bill_ids,omitempty" jsonschema:"Selector: IDs de facturas a pagar"`
	Month            *string `json:"month,omitempty" jsonschema:"Selector: mes YYYY-MM a pagar"`
	From             *string `json:"from,omitempty" jsonschema:"Selector de rango: período inicial YYYY-MM"`
	To               *string `json:"to,omitempty" jsonschema:"Selector de rango: período final YYYY-MM"`
	AllPendingUpTo   *string `json:"all_pending_up_to,omitempty" jsonschema:"Selector: pagar todas las pendientes hasta YYYY-MM (inclusive)"`
	PaidAt           *string `json:"paid_at,omitempty" jsonschema:"Fecha de pago YYYY-MM-DD (default hoy)"`
	PaymentReference *string `json:"payment_reference,omitempty" jsonschema:"Referencia interna del pago"`
	DriveURL         *string `json:"drive_url,omitempty" jsonschema:"URL de Google Drive del comprobante"`
	DryRun           *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) pay(ctx context.Context, _ *mcp.CallToolRequest, input PayInput) (*mcp.CallToolResult, any, error) {
	if input.ServiceID <= 0 {
		return toolError("invalid_request", "service_id debe ser mayor que cero", "")
	}
	req := api.PayRequest{
		BillIDs: input.BillIDs,
		PaidAt:  t.today(),
	}
	if input.PaymentReference != nil {
		req.PaymentReference = strings.TrimSpace(*input.PaymentReference)
	}
	if input.DriveURL != nil {
		req.DriveURL = strings.TrimSpace(*input.DriveURL)
	}

	selectors := 0
	if len(input.BillIDs) > 0 {
		req.BillIDs = input.BillIDs
		selectors++
	}
	if input.Month != nil && strings.TrimSpace(*input.Month) != "" {
		parsed, err := parsePeriod(*input.Month)
		if err != nil {
			return toolError("invalid_selector", err.Error(), "")
		}
		req.Month = parsed.String()
		selectors++
	}
	hasFrom := input.From != nil && strings.TrimSpace(*input.From) != ""
	hasTo := input.To != nil && strings.TrimSpace(*input.To) != ""
	if hasFrom || hasTo {
		if !hasFrom || !hasTo {
			return toolError("invalid_selector", "el selector de rango requiere from y to", "")
		}
		from, err := parsePeriod(*input.From)
		if err != nil {
			return toolError("invalid_selector", err.Error(), "")
		}
		to, err := parsePeriod(*input.To)
		if err != nil {
			return toolError("invalid_selector", err.Error(), "")
		}
		if to.before(from) {
			return toolError("invalid_selector", "el período final no puede ser anterior al inicial", "")
		}
		req.From = from.String()
		req.To = to.String()
		selectors++
	}
	if input.AllPendingUpTo != nil && strings.TrimSpace(*input.AllPendingUpTo) != "" {
		parsed, err := parsePeriod(*input.AllPendingUpTo)
		if err != nil {
			return toolError("invalid_selector", err.Error(), "")
		}
		req.AllPendingUpTo = parsed.String()
		selectors++
	}
	if selectors != 1 {
		return toolError("invalid_selector",
			"Indicá exactamente un selector: bill_ids, month, from+to o all_pending_up_to.", "")
	}

	if input.PaidAt != nil && strings.TrimSpace(*input.PaidAt) != "" {
		raw := strings.TrimSpace(*input.PaidAt)
		paidAt, err := time.ParseInLocation("2006-01-02", raw, t.cfg.Location)
		if err != nil {
			return toolError("invalid_paid_at", "paid_at debe tener formato YYYY-MM-DD", "")
		}
		today, err := time.ParseInLocation("2006-01-02", t.today(), t.cfg.Location)
		if err != nil {
			return toolError("invalid_paid_at", "no se pudo resolver la fecha de hoy", "")
		}
		if paidAt.After(today) {
			return toolError("invalid_paid_at", "paid_at no puede ser una fecha futura", "")
		}
		req.PaidAt = raw
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}
	req.DryRun = dryRun

	result, err := t.client.PayBills(ctx, input.ServiceID, req)
	if err != nil {
		return apiError(err)
	}
	return jsonResult(result)
}
