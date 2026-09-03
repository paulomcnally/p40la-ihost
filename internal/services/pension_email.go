package services

import (
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// monthName devuelve el nombre del mes en español (reusa monthNames de bill_email.go).
func monthName(month int) string {
	if name, ok := monthNames[month]; ok {
		return name
	}
	return fmt.Sprintf("%d", month)
}

func pensionPeriod(month, year int) string {
	return fmt.Sprintf("%s %d", monthName(month), year)
}

func pensionAmount(amount float64, currency string) string {
	return fmt.Sprintf("%s %.2f", currency, amount)
}

func pensionChildLabel(record *models.SupportRecord) string {
	return fmt.Sprintf("%s — %s", record.ChildName, record.CategoryName)
}

// buildPensionRecordsCreatedEmail construye el email de "registros creados"
// al generar un mes (salarios + registros de manutención).
func buildPensionRecordsCreatedEmail(createdSalaryPayments []models.SalaryPayment, createdSupportRecords []models.SupportRecord, year, month int) (title, contentHTML string) {
	period := pensionPeriod(month, year)
	total := len(createdSalaryPayments) + len(createdSupportRecords)
	title = fmt.Sprintf("Pensión — %d registro(s) creado(s) — %s", total, period)

	var b string
	b += fmt.Sprintf("<p>Se generaron los registros de manutención del período <strong>%s</strong>.</p>", period)

	if len(createdSalaryPayments) > 0 {
		b += "<p><strong>Salarios generados:</strong></p><ul>"
		for _, sp := range createdSalaryPayments {
			b += fmt.Sprintf("<li>%s — %s</li>", sp.Employer, pensionAmount(sp.Amount, sp.Currency))
		}
		b += "</ul>"
	}

	if len(createdSupportRecords) > 0 {
		b += "<p><strong>Registros de manutención generados:</strong></p><ul>"
		for _, r := range createdSupportRecords {
			b += fmt.Sprintf("<li>%s — %s</li>", pensionChildLabel(&r), pensionAmount(r.Amount, r.Currency))
		}
		b += "</ul>"
	}

	b += "<p style=\"margin-top:24px;color:#8e8e93;font-size:13px;\">Notificación automática del módulo Pensión Alimenticia.</p>"
	return title, b
}

// buildPensionRecordPaidEmail construye el email al marcar un registro como pagado.
func buildPensionRecordPaidEmail(record *models.SupportRecord) (title, contentHTML string) {
	period := pensionPeriod(record.Month, record.Year)
	title = fmt.Sprintf("Pensión — Pago registrado — %s", period)

	contentHTML = fmt.Sprintf(`
<p>Se registró el pago de un registro de manutención:</p>

<p>
  <strong>Hijo / Categoría:</strong> %s<br/>
  <strong>Monto:</strong> %s<br/>
  <strong>Fecha de pago:</strong> %s<br/>
  <strong>Método:</strong> %s<br/>
  <strong>Referencia:</strong> %s
</p>

<p style="margin-top:24px;color:#8e8e93;font-size:13px;">
  Notificación automática del módulo Pensión Alimenticia.
</p>`,
		pensionChildLabel(record),
		pensionAmount(record.Amount, record.Currency),
		formatTimePtr(record.PaidAt),
		strOrDash(record.PaymentMethod),
		strOrDash(record.PaymentReference),
	)
	return title, contentHTML
}

// buildPensionSalaryReceivedEmail construye el email al marcar un salario como recibido.
func buildPensionSalaryReceivedEmail(payment *models.SalaryPayment) (title, contentHTML string) {
	period := pensionPeriod(payment.Month, payment.Year)
	title = fmt.Sprintf("Pensión — Salario recibido — %s", period)

	contentHTML = fmt.Sprintf(`
<p>Se registró la recepción de un pago de salario:</p>

<p>
  <strong>Empleador / Fuente:</strong> %s<br/>
  <strong>Monto esperado:</strong> %s<br/>
  <strong>Monto recibido:</strong> %s<br/>
  <strong>Fecha de recepción:</strong> %s<br/>
  <strong>Notas:</strong> %s
</p>

<p style="margin-top:24px;color:#8e8e93;font-size:13px;">
  Notificación automática del módulo Pensión Alimenticia.
</p>`,
		payment.Employer,
		pensionAmount(payment.Amount, payment.Currency),
		pensionAmount(floatOr(payment.ReceivedAmount, payment.Amount), payment.Currency),
		formatTimePtr(payment.ReceivedAt),
		strOrDash(payment.Notes),
	)
	return title, contentHTML
}

// buildPensionRecordRejectedEmail construye el email al rechazar un registro.
func buildPensionRecordRejectedEmail(record *models.SupportRecord, reason string) (title, contentHTML string) {
	period := pensionPeriod(record.Month, record.Year)
	title = fmt.Sprintf("Pensión — Registro rechazado — %s", period)

	contentHTML = fmt.Sprintf(`
<p>Se rechazó un registro de manutención:</p>

<p>
  <strong>Hijo / Categoría:</strong> %s<br/>
  <strong>Monto:</strong> %s<br/>
  <strong>Motivo:</strong> %s
</p>

<p style="margin-top:24px;color:#8e8e93;font-size:13px;">
  Notificación automática del módulo Pensión Alimenticia.
</p>`,
		pensionChildLabel(record),
		pensionAmount(record.Amount, record.Currency),
		reason,
	)
	return title, contentHTML
}

// buildPensionMonthClosingEmail construye el resumen al cerrar un mes.
func buildPensionMonthClosingEmail(records []models.SupportRecord, salaryPayments []models.SalaryPayment, year, month int) (title, contentHTML string) {
	period := pensionPeriod(month, year)
	title = fmt.Sprintf("Pensión — Cierre de mes — %s", period)

	var paid, pending, rejected int
	var paidAmount, pendingAmount, rejectedAmount float64
	for _, r := range records {
		switch r.Status {
		case "paid":
			paid++
			paidAmount += r.Amount
		case "rejected":
			rejected++
			rejectedAmount += r.Amount
		default:
			pending++
			pendingAmount += r.Amount
		}
	}

	var b string
	b += fmt.Sprintf("<p>Se cerró el período <strong>%s</strong>. Resumen:</p>", period)

	if len(salaryPayments) > 0 {
		b += "<p><strong>Salarios del mes:</strong></p><ul>"
		for _, sp := range salaryPayments {
			status := "pendiente"
			if sp.Status == "received" {
				status = "recibido"
			}
			b += fmt.Sprintf("<li>%s — %s (%s)</li>", sp.Employer, pensionAmount(sp.Amount, sp.Currency), status)
		}
		b += "</ul>"
	}

	b += "<p><strong>Registros de manutención:</strong></p>"
	b += "<ul>"
	b += fmt.Sprintf("<li><strong>Pagados:</strong> %d — %s</li>", paid, pensionAmount(paidAmount, "NIO"))
	b += fmt.Sprintf("<li><strong>Pendientes:</strong> %d — %s</li>", pending, pensionAmount(pendingAmount, "NIO"))
	b += fmt.Sprintf("<li><strong>Rechazados:</strong> %d — %s</li>", rejected, pensionAmount(rejectedAmount, "NIO"))
	b += "</ul>"

	b += "<p style=\"margin-top:24px;color:#8e8e93;font-size:13px;\">Notificación automática del módulo Pensión Alimenticia.</p>"
	return title, b
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("02/01/2006 15:04")
}

func strOrDash(p *string) string {
	if p == nil || *p == "" {
		return "—"
	}
	return *p
}

func floatOr(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}
