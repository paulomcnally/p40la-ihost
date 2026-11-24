package bills

import (
	"context"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// ListInput es la entrada de bills_list.
type ListInput struct {
	ServiceID int64   `json:"service_id" jsonschema:"ID del servicio (obligatorio)"`
	From      *string `json:"from,omitempty" jsonschema:"Período inicial YYYY-MM (opcional)"`
	To        *string `json:"to,omitempty" jsonschema:"Período final YYYY-MM (opcional)"`
	Status    *string `json:"status,omitempty" jsonschema:"Filtro de estado: pending, paid o all (default all)"`
}

// BillSummary es una factura resumida para la tool de listado.
type BillSummary struct {
	ID            int64   `json:"id"`
	Year          int     `json:"year"`
	Month         int     `json:"month"`
	Period        string  `json:"period"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	InvoiceNumber string  `json:"invoice_number,omitempty"`
	DueDate       *string `json:"due_date,omitempty"`
}

// ListOutput es el resultado de bills_list.
type ListOutput struct {
	ServiceID   int64         `json:"service_id"`
	ServiceName string        `json:"service_name"`
	From        string        `json:"from,omitempty"`
	To          string        `json:"to,omitempty"`
	Status      string        `json:"status"`
	Count       int           `json:"count"`
	Pending     int           `json:"pending"`
	Paid        int           `json:"paid"`
	TotalAmount float64       `json:"total_amount"`
	Bills       []BillSummary `json:"bills"`
}

func (t *Toolset) list(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, any, error) {
	if input.ServiceID <= 0 {
		return toolkit.ToolError("invalid_request", "service_id debe ser mayor que cero", "")
	}

	status := "all"
	if input.Status != nil && strings.TrimSpace(*input.Status) != "" {
		status = strings.ToLower(strings.TrimSpace(*input.Status))
	}
	if status != "all" && status != "pending" && status != "paid" {
		return toolkit.ToolError("invalid_request", "status debe ser pending, paid o all", "")
	}

	var from, to *period
	if input.From != nil && strings.TrimSpace(*input.From) != "" {
		parsed, err := parsePeriod(*input.From)
		if err != nil {
			return toolkit.ToolError("invalid_range", err.Error(), "")
		}
		from = &parsed
	}
	if input.To != nil && strings.TrimSpace(*input.To) != "" {
		parsed, err := parsePeriod(*input.To)
		if err != nil {
			return toolkit.ToolError("invalid_range", err.Error(), "")
		}
		to = &parsed
	}
	if from != nil && to != nil && to.before(*from) {
		return toolkit.ToolError("invalid_range", "el período final no puede ser anterior al inicial", "")
	}

	service, err := t.client.GetService(ctx, input.ServiceID)
	if err != nil {
		return toolkit.APIError(err)
	}
	bills, err := t.client.ListBills(ctx, input.ServiceID)
	if err != nil {
		return toolkit.APIError(err)
	}

	summary := ListOutput{
		ServiceID:   service.ID,
		ServiceName: service.Name,
		Status:      status,
		Bills:       make([]BillSummary, 0, len(bills)),
	}
	if from != nil {
		summary.From = from.String()
	}
	if to != nil {
		summary.To = to.String()
	}

	for _, bill := range bills {
		billPeriod := billPeriodOf(bill.Year, bill.Month)
		if from != nil && billPeriod.before(*from) {
			continue
		}
		if to != nil && billPeriod.after(*to) {
			continue
		}
		if status != "all" && bill.Status != status {
			continue
		}
		summary.Bills = append(summary.Bills, BillSummary{
			ID:            bill.ID,
			Year:          bill.Year,
			Month:         bill.Month,
			Period:        billPeriod.String(),
			Amount:        bill.Amount,
			Status:        bill.Status,
			InvoiceNumber: bill.InvoiceNumber,
			DueDate:       bill.DueDate,
		})
		summary.TotalAmount += bill.Amount
		if bill.Status == "paid" {
			summary.Paid++
		} else {
			summary.Pending++
		}
	}

	sort.SliceStable(summary.Bills, func(i, j int) bool {
		a := billPeriodOf(summary.Bills[i].Year, summary.Bills[i].Month)
		b := billPeriodOf(summary.Bills[j].Year, summary.Bills[j].Month)
		if a.Year != b.Year {
			return a.Year < b.Year
		}
		if a.Month != b.Month {
			return a.Month < b.Month
		}
		return summary.Bills[i].ID < summary.Bills[j].ID
	})
	summary.Count = len(summary.Bills)
	return toolkit.JSONResult(summary)
}
