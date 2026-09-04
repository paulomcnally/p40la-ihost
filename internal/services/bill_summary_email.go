package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// renderBillSummaryContent construye el HTML del cuerpo del email de resumen
// diario de facturas pendientes, agrupado por casa (SPEC-031).
func renderBillSummaryContent(pending []models.PendingBillDetail, format CurrencyFormat) string {
	if len(pending) == 0 {
		return "<p>No hay facturas pendientes.</p>"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<p>Tenés <strong>%d facturas pendientes</strong> de pago. Este es el resumen:</p>", len(pending)))

	homes := groupByHome(pending)
	for _, home := range homes {
		b.WriteString(fmt.Sprintf(`<h3 style="margin:24px 0 8px;color:#1d1d1f;font-size:16px;">%s</h3>`, esc(home.Name)))
		b.WriteString(`<table width="100%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin-top:8px;">`)
		b.WriteString(`<tr style="background-color:#f5f5f7;">`)
		b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Institución</th>`)
		b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Servicio</th>`)
		b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Período</th>`)
		b.WriteString(`<th style="text-align:right;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Monto</th>`)
		b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Antigüedad</th>`)
		b.WriteString(`</tr>`)

		for _, d := range home.Bills {
			institution := d.Institution
			if institution == "" {
				institution = "—"
			}
			badge := ageBadge(daysSince(d.CreatedAt))

			b.WriteString("<tr>")
			b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, esc(institution)))
			b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, esc(d.ServiceName)))
			b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, esc(formatPeriod(d.Month, d.Year))))
			b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;text-align:right;">%s</td>`, esc(formatAmount(d.Amount, d.CurrencySymbol, format))))
			b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, badge))
			b.WriteString("</tr>")
		}

		b.WriteString("</table>")
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
