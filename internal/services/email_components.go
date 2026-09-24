package services

import (
	"fmt"
	"strings"
)

// EmailField es un par etiqueta:valor de una card de email.
type EmailField struct {
	Label string
	Value string // ya formateado por el caller
	// HTML indica que Value es HTML propio (ej. el badge de antigüedad) y
	// no debe escaparse.
	HTML bool
	// Bold renderiza el valor en negrita (ej. el total del día).
	Bold bool
}

// RenderEmailCard arma una card vertical con pares etiqueta: valor usando
// tablas anidadas y estilos 100% inline. NO depende de <style> ni media
// queries: se ve igual en Gmail app/web (que recortan <style>), Apple Mail,
// Outlook y cualquier ancho de pantalla (ADR-001). Reemplaza el HTML de
// tablas de datos duplicado en bill_summary_email.go y debt_due_email.go.
func RenderEmailCard(fields []EmailField, palette EmailPalette) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		`<table width="100%%" cellpadding="0" cellspacing="0" style="margin:0 0 12px;border:1px solid %s;border-radius:12px;background-color:%s;">`,
		palette.Border, palette.Card,
	))
	b.WriteString(`<tr><td style="padding:6px 16px;">`)
	b.WriteString(`<table width="100%" cellpadding="0" cellspacing="0">`)
	for _, f := range fields {
		value := esc(f.Value)
		if f.HTML {
			value = f.Value
		}
		weight := "400"
		if f.Bold {
			weight = "700"
		}
		b.WriteString(`<tr>`)
		b.WriteString(fmt.Sprintf(
			`<td style="padding:5px 0;font-size:11px;color:%s;text-transform:uppercase;letter-spacing:0.4px;vertical-align:top;white-space:nowrap;">%s</td>`,
			palette.Muted, esc(f.Label),
		))
		b.WriteString(fmt.Sprintf(
			`<td style="padding:5px 0 5px 12px;font-size:14px;color:%s;font-weight:%s;text-align:right;word-break:break-word;">%s</td>`,
			palette.Text, weight, value,
		))
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</table>`)
	b.WriteString(`</td></tr></table>`)
	return b.String()
}
