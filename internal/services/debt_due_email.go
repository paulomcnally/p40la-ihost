package services

import (
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// renderDebtDueContent construye el HTML del email diario que agrupa todas las
// cuotas de deudas que vencen hoy, con el total del día (SPEC-054). Cada cuota
// se renderiza como una card vertical (etiqueta: valor) con estilos inline,
// legible en cualquier cliente de email (ADR-001, SPEC-077).
func renderDebtDueContent(due []models.DebtBill, format CurrencyFormat, palette EmailPalette) string {
	if len(due) == 0 {
		return "<p>No hay cuotas de deudas que venzan hoy.</p>"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<p>Hoy vencen <strong>%d cuota%s</strong> de tus deudas. Este es el detalle:</p>", len(due), plural(len(due))))

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

		b.WriteString(RenderEmailCard([]EmailField{
			{Label: "Acreedor", Value: institution},
			{Label: "Deuda", Value: description},
			{Label: "Cuota", Value: fmt.Sprintf("Cuota #%d", bill.InstallmentNumber)},
			{Label: "Vence", Value: bill.DueDate},
			{Label: "Monto", Value: formatAmount(bill.Amount, bill.CurrencyCode, format)},
		}, palette))

		total += bill.Amount
	}

	b.WriteString(RenderEmailCard([]EmailField{
		{Label: "Total del día", Value: formatAmount(total, firstCurrency(due), format), Bold: true},
	}, palette))

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
