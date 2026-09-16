package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// renderBillSummaryContent construye el HTML del cuerpo del email de resumen
// diario de facturas pendientes, agrupado por casa (SPEC-031). Cada factura
// se renderiza como una card vertical (etiqueta: valor) con estilos inline,
// legible en cualquier cliente de email (ADR-001, SPEC-077).
func renderBillSummaryContent(pending []models.PendingBillDetail, format CurrencyFormat, palette EmailPalette) string {
	if len(pending) == 0 {
		return "<p>No hay facturas pendientes.</p>"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<p>Tenés <strong>%d facturas pendientes</strong> de pago. Este es el resumen:</p>", len(pending)))

	homes := groupByHome(pending)
	for _, home := range homes {
		b.WriteString(fmt.Sprintf(`<h3 style="margin:24px 0 8px;color:%s;font-size:16px;">%s</h3>`, palette.Text, esc(home.Name)))

		for _, d := range home.Bills {
			institution := d.Institution
			if institution == "" {
				institution = "—"
			}
			b.WriteString(RenderEmailCard([]EmailField{
				{Label: "Institución", Value: institution},
				{Label: "Servicio", Value: d.ServiceName},
				{Label: "Período", Value: formatPeriod(d.Month, d.Year)},
				{Label: "Monto", Value: formatAmount(d.Amount, d.CurrencySymbol, format)},
				{Label: "Antigüedad", Value: ageBadge(daysSince(d.CreatedAt)), HTML: true},
			}, palette))
		}
	}

	b.WriteString(`<p style="margin-top:24px;color:#8e8e93;font-size:13px;">`)
	b.WriteString(`Ingresá a <a href="http://ihost:8088/bills" style="color:#007aff;">P40LA</a> para gestionar tus facturas pendientes.</p>`)

	return b.String()
}

// homeGroup agrupa facturas pendientes bajo una misma casa.
type homeGroup struct {
	Name  string
	Bills []models.PendingBillDetail
}

// groupByHome agrupa las facturas pendientes por casa, en orden de nombre.
func groupByHome(pending []models.PendingBillDetail) []homeGroup {
	index := make(map[string]int)
	var groups []homeGroup

	for _, d := range pending {
		name := d.HomeName
		if name == "" {
			name = "Otras"
		}
		i, ok := index[name]
		if !ok {
			index[name] = len(groups)
			groups = append(groups, homeGroup{Name: name})
			i = len(groups) - 1
		}
		groups[i].Bills = append(groups[i].Bills, d)
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups
}

// daysSince devuelve los días calendario transcurridos desde una fecha.
func daysSince(t time.Time) int {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	days := int(now.Sub(day).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// ageBadge renderiza la antigüedad de una factura con color según urgencia.
func ageBadge(days int) string {
	var label, color string
	switch {
	case days == 0:
		label, color = "Hoy", "#8e8e93"
	case days == 1:
		label, color = "Ayer", "#8e8e93"
	case days <= 15:
		label, color = fmt.Sprintf("Hace %d días", days), "#ff9500"
	default:
		label, color = fmt.Sprintf("Hace %d días", days), "#ff3b30"
	}
	return fmt.Sprintf(`<span style="background-color:%s;color:#ffffff;border-radius:8px;padding:2px 8px;font-size:12px;">%s</span>`, color, label)
}

// esc escapa HTML para evitar inyección en los emails.
func esc(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}
