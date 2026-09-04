package services

import (
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// renderDebtDueContent construye el HTML del email diario que agrupa todas las
// cuotas de deudas que vencen hoy, con el total del día (SPEC-054).
func renderDebtDueContent(due []models.DebtBill, format CurrencyFormat) string {
	if len(due) == 0 {
		return "<p>No hay cuotas de deudas que venzan hoy.</p>"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<p>Hoy vencen <strong>%d cuota%s</strong> de tus deudas. Este es el detalle:</p>", len(due), plural(len(due))))

	b.WriteString(`<table width="100%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin-top:8px;">`)
	b.WriteString(`<tr style="background-color:#f5f5f7;">`)
	b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Acreedor</th>`)
	b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Deuda</th>`)
	b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Cuota</th>`)
	b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Vence</th>`)
	b.WriteString(`<th style="text-align:right;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Monto</th>`)
	b.WriteString(`</tr>`)

	total := 0.0
	for _, bill := range due {
		institution := bill.InstitutionName
		if institution == "" {
			institution = "—"
		}
		description := bill.DebtDescription
		if description == "" {
			description = "—"
		}

		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, esc(institution)))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, esc(description)))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">Cuota #%d</td>`, bill.InstallmentNumber))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, esc(bill.DueDate)))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;text-align:right;">%s</td>`, esc(formatAmount(bill.Amount, bill.CurrencyCode, format))))
		b.WriteString("</tr>")

		total += bill.Amount
	}

	b.WriteString(fmt.Sprintf(`<tr><td colspan="4" style="padding:12px;border-bottom:1px solid #e5e5ea;text-align:right;font-weight:700;">Total del día</td><td style="padding:12px;border-bottom:1px solid #e5e5ea;text-align:right;font-weight:700;">%s</td></tr>`, esc(formatAmount(total, firstCurrency(due), format))))
	b.WriteString("</table>")

	b.WriteString(`<p style="margin-top:24px;color:#8e8e93;font-size:13px;">`)
	b.WriteString(`Ingresá a <a href="http://ihost:8088/deudas" style="color:#007aff;">P40LA</a> para gestionar tus deudas.</p>`)

	return b.String()
}

// firstCurrency devuelve el código de moneda de la primera cuota (para el total).
func firstCurrency(due []models.DebtBill) string {
	if len(due) == 0 {
		return ""
	}
	return due[0].CurrencyCode
}
