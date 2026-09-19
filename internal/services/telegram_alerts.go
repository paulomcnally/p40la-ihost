package services

import (
	"fmt"
	"strings"
	"time"

	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
)

// Formatters de texto plano para las alertas push de Telegram (SPEC-088).
// Espejo de los builders HTML de los emails, con el mismo estilo visual de
// los comandos del bot (negritas, emojis, montos formateados).

// formatInsuranceAlerts construye el mensaje de la alerta de seguros de autos
// (AlertScheduler), espejo en texto plano de renderAlertsContent.
func formatInsuranceAlerts(alerts []appmodels.AutoAlert) []string {
	if len(alerts) == 0 {
		return []string{"✅ No hay autos en condición de alerta."}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "🚗 *Alertas de seguros de autos* (%d vehículo%s)\n\n", len(alerts), plural(len(alerts)))
	for _, a := range alerts {
		vehicle := strings.TrimSpace(a.Brand + " " + a.Model)
		if a.Year > 0 {
			vehicle = fmt.Sprintf("%s %d", vehicle, a.Year)
		}
		if vehicle == "" {
			vehicle = fmt.Sprintf("Auto #%d", a.AutoID)
		}
		var reason string
		switch a.AlertType {
		case appmodels.AlertTypeNoInsurance:
			reason = "🔴 Sin seguro asociado"
		case appmodels.AlertTypeExpired:
			reason = fmt.Sprintf("🟠 Seguro vencido el %s", formatDate(a.EndDate))
		default:
			reason = a.AlertType
		}
		fmt.Fprintf(&b, "  • %s — %s\n    %s\n\n", vehicle, a.Placa, reason)
	}
	b.WriteString("Ingresá a P40LA para revisar y asociar un nuevo seguro.")
	return []string{strings.TrimRight(b.String(), "\n")}
}

// formatBillCreatedAlert construye el aviso de facturas generadas
// automáticamente (BillingScheduler, AlertKeyBillCreated). Un mensaje por
// corrida (mismo criterio que la voz, SPEC-088).
func formatBillCreatedAlert(generated int) []string {
	if generated == 1 {
		return []string{"⚡ Se generó *1 factura automática*."}
	}
	return []string{fmt.Sprintf("⚡ Se generaron *%d facturas automáticas*.", generated)}
}

// formatDebtsDueToday filtra las cuotas pendientes que vencen HOY y las
// formatea con el layout de /deudas_pendientes (DebtDueScheduler, SPEC-088).
func formatDebtsDueToday(pending []appmodels.PendingDebtDetail, format CurrencyFormat, now time.Time, sepLen, showMonths int) []string {
	filtered := make([]appmodels.PendingDebtDetail, 0, len(pending))
	today := now.Format("2006-01-02")
	for _, p := range pending {
		if p.DueDate == today {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return []string{"✅ No hay cuotas que vencen hoy."}
	}
	return formatDeudasPendientes(filtered, format, now, sepLen, showMonths)
}

// formatPensionRecordsCreated construye el aviso de registros/salarios
// generados al crear un mes de pensión (SPEC-088).
func formatPensionRecordsCreated(salaryPayments []appmodels.SalaryPayment, records []appmodels.SupportRecord, year, month int, format CurrencyFormat) []string {
	period := pensionPeriod(month, year)
	total := len(salaryPayments) + len(records)
	var b strings.Builder
	fmt.Fprintf(&b, "📋 *Pensión — %d registro(s) creado(s) — %s*\n\n", total, period)

	if len(salaryPayments) > 0 {
		b.WriteString("*Salarios generados:*\n")
		for _, sp := range salaryPayments {
			fmt.Fprintf(&b, "  • %s — %s\n", sp.Employer, pensionAmount(sp.Amount, sp.Currency, format))
		}
		b.WriteString("\n")
	}
	if len(records) > 0 {
		b.WriteString("*Registros de manutención generados:*\n")
		for _, r := range records {
			fmt.Fprintf(&b, "  • %s — %s\n", pensionChildLabel(&r), pensionAmount(r.Amount, r.Currency, format))
		}
	}
	return []string{strings.TrimRight(b.String(), "\n")}
}

// formatPensionRecordPaid construye el aviso de pago de manutención (SPEC-088).
func formatPensionRecordPaid(record *appmodels.SupportRecord, format CurrencyFormat) []string {
	text := fmt.Sprintf("💰 *Pensión — Pago registrado — %s*\n\n", pensionPeriod(record.Month, record.Year)) +
		fmt.Sprintf("  *Hijo / Categoría*: %s\n", pensionChildLabel(record)) +
		fmt.Sprintf("  *Monto*: %s\n", pensionAmount(record.Amount, record.Currency, format)) +
		fmt.Sprintf("  *Fecha de pago*: %s\n", formatTimePtr(record.PaidAt)) +
		fmt.Sprintf("  *Método*: %s\n", strOrDash(record.PaymentMethod)) +
		fmt.Sprintf("  *Referencia*: %s", strOrDash(record.PaymentReference))
	return []string{text}
}

// formatPensionSalaryReceived construye el aviso de salario recibido (SPEC-088).
func formatPensionSalaryReceived(payment *appmodels.SalaryPayment, format CurrencyFormat) []string {
	text := fmt.Sprintf("💵 *Pensión — Salario recibido — %s*\n\n", pensionPeriod(payment.Month, payment.Year)) +
		fmt.Sprintf("  *Empleador / Fuente*: %s\n", payment.Employer) +
		fmt.Sprintf("  *Monto esperado*: %s\n", pensionAmount(payment.Amount, payment.Currency, format)) +
		fmt.Sprintf("  *Monto recibido*: %s\n", pensionAmount(floatOr(payment.ReceivedAmount, payment.Amount), payment.Currency, format)) +
		fmt.Sprintf("  *Fecha de recepción*: %s\n", formatTimePtr(payment.ReceivedAt)) +
		fmt.Sprintf("  *Notas*: %s", strOrDash(payment.Notes))
	return []string{text}
}

// formatPensionRecordRejected construye el aviso de registro rechazado (SPEC-088).
func formatPensionRecordRejected(record *appmodels.SupportRecord, reason string, format CurrencyFormat) []string {
	text := fmt.Sprintf("⛔ *Pensión — Registro rechazado — %s*\n\n", pensionPeriod(record.Month, record.Year)) +
		fmt.Sprintf("  *Hijo / Categoría*: %s\n", pensionChildLabel(record)) +
		fmt.Sprintf("  *Monto*: %s\n", pensionAmount(record.Amount, record.Currency, format)) +
		fmt.Sprintf("  *Motivo*: %s", reason)
	return []string{text}
}

// formatPensionMonthClosing construye el resumen de cierre de mes (SPEC-088).
func formatPensionMonthClosing(records []appmodels.SupportRecord, salaryPayments []appmodels.SalaryPayment, year, month int, format CurrencyFormat) []string {
	period := pensionPeriod(month, year)

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

	var b strings.Builder
	fmt.Fprintf(&b, "📊 *Pensión — Cierre de mes — %s*\n\n", period)

	if len(salaryPayments) > 0 {
		b.WriteString("*Salarios del mes:*\n")
		for _, sp := range salaryPayments {
			status := "pendiente"
			if sp.Status == "received" {
				status = "recibido"
			}
			fmt.Fprintf(&b, "  • %s — %s (%s)\n", sp.Employer, pensionAmount(sp.Amount, sp.Currency, format), status)
		}
		b.WriteString("\n")
	}

	b.WriteString("*Registros de manutención:*\n")
	fmt.Fprintf(&b, "  • *Pagados*: %d — %s\n", paid, pensionAmount(paidAmount, "NIO", format))
	fmt.Fprintf(&b, "  • *Pendientes*: %d — %s\n", pending, pensionAmount(pendingAmount, "NIO", format))
	fmt.Fprintf(&b, "  • *Rechazados*: %d — %s", rejected, pensionAmount(rejectedAmount, "NIO", format))
	return []string{b.String()}
}