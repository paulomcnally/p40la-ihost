package services

import (
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// monthNames en español para formatear el período en los emails.
var monthNames = map[int]string{
	1: "Enero", 2: "Febrero", 3: "Marzo", 4: "Abril",
	5: "Mayo", 6: "Junio", 7: "Julio", 8: "Agosto",
	9: "Septiembre", 10: "Octubre", 11: "Noviembre", 12: "Diciembre",
}

// buildBillCreatedEmail construye el título y contenido del email que se envía
// cuando el sistema genera una factura automáticamente (SPEC-030).
func buildBillCreatedEmail(svc *models.Service, bill *models.Bill, currencySymbol string) (title, contentHTML string) {
	title = fmt.Sprintf("Nueva factura generada — %s", svc.Name)

	period := formatPeriod(bill.Month, bill.Year)
	amount := formatAmount(bill.Amount, currencySymbol)

	amountType := "fijo"
	if svc.BillingType == "variable" {
		amountType = "variable (podés editarlo)"
	}

	institution := svc.Institution
	if institution == "" {
		institution = "—"
	}

	contentHTML = fmt.Sprintf(`
<p>Hola,</p>

<p>
  Se ha generado una nueva factura que debes pagar en el período
  <strong>%s</strong> por el monto de <strong>%s</strong>.
</p>

<p>Este monto es <strong>%s</strong>.</p>

<p>
  <strong>Institución:</strong> %s<br/>
  <strong>Servicio:</strong> %s<br/>
  <strong>Período:</strong> %s<br/>
  <strong>Monto:</strong> %s
</p>

<p style="margin-top:24px;color:#8e8e93;font-size:13px;">
  Esta es una notificación automática del sistema de facturación.
</p>`, period, amount, amountType, institution, svc.Name, period, amount)

	return title, contentHTML
}

// formatPeriod formatea el período de una factura (month=0 es anual).
func formatPeriod(month, year int) string {
	if month == 0 {
		return fmt.Sprintf("Año %d", year)
	}
	name, ok := monthNames[month]
	if !ok {
		return fmt.Sprintf("%d/%d", month, year)
	}
	return fmt.Sprintf("%s %d", name, year)
}

// formatAmount formatea un monto con el símbolo de la moneda.
func formatAmount(amount float64, symbol string) string {
	if symbol == "" {
		return fmt.Sprintf("%.2f", amount)
	}
	return fmt.Sprintf("%s%.2f", symbol, amount)
}