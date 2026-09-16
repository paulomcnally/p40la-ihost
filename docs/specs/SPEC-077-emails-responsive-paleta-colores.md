---
title: "Refactor Responsive de Emails (Mobile) + Paleta de Colores Configurable"
id: "SPEC-077"
status: "released"
author: "paulomcnally"
created: "2026-09-15"
updated: "2026-09-15"
github_issue: 80
---

# SPEC-077: Refactor Responsive de Emails (Mobile) + Paleta de Colores Configurable

**ID**: SPEC-077
**Estado**: released
**Autor**: paulomcnally (spec redactada con asistencia de Claude)
**Creado**: 2026-09-15

---

## 1. Resumen Ejecutivo

Los emails que envía p40la-ihost (resumen diario de facturas pendientes, cuotas de deudas que vencen hoy, factura nueva generada, notificaciones de pensión alimenticia) se ven mal en clientes de correo móviles. El caso más grave es el **resumen diario de facturas** (`renderBillSummaryContent`, `internal/services/bill_summary_email.go`) y el de **cuotas de deudas** (`renderDebtDueContent`, `internal/services/debt_due_email.go`): ambos arman una tabla HTML de 5 columnas (`Institución / Servicio / Período / Monto / Antigüedad`) con `cellpadding="8"` fija, sin ningún tratamiento responsive. En un iPhone esa tabla se ve exactamente como en la captura que motivó esta spec: columnas comprimidas, texto y badges de antigüedad partidos en varias líneas, muy difícil de leer.

La causa raíz es doble:

1. La plantilla base (`emailTemplateHTML` en `internal/services/email_service.go`) usa una tabla contenedora de **ancho fijo `width="600"`** sin `max-width` fluido ni ninguna regla `@media`.
2. Las tablas de datos (`renderBillSummaryContent`, `renderDebtDueContent`) son tablas de N columnas construidas a mano, pensadas para desktop, sin ninguna estrategia de apilado en pantallas angostas.

Esta spec propone:

- **(A)** Refactorizar la plantilla base y las tablas de datos para que sean responsive de verdad en mobile (fluidas + apiladas en pantallas angostas), sin romper la compatibilidad con Gmail/Outlook/Apple Mail en desktop.
- **(B)** Agregar una sección nueva en Settings donde el usuario pueda configurar la **paleta de colores** que se aplica a todos los emails (color principal/header, fondo, texto, etc.), persistida en `system_settings` igual que el resto de la configuración del sistema.

Es un cambio de **frontend de emails + Settings**, sin migraciones de base de datos (se reutiliza `system_settings`, igual que SPEC-029/SPEC-058) y sin dependencias nuevas (sigue usando `net/smtp` de stdlib + HTML/CSS inline).

---

## 2. Requerimientos

### 2.1 Responsive (P0 — Obligatorios)

1. **REQ-001**: La tabla contenedora de la plantilla base debe ser fluida: `width="100%"` con `max-width:600px` (en vez de `width="600"` fijo), para que se ajuste al ancho real del viewport en mobile.
2. **REQ-002**: Se debe agregar un bloque `<style>` en el `<head>` de la plantilla con una media query `@media screen and (max-width: 480px)` que reduzca paddings y tamaños de fuente del header, título y contenido para pantallas angostas.
3. **REQ-003**: Los datos multi-columna (resumen diario de facturas, cuotas de deudas del día, y cualquier lista similar futura) deben renderizarse como **cards verticales** con pares `etiqueta: valor` (estilos inline, tablas anidadas), visibles en cualquier cliente y ancho — **sin depender de media queries** (Gmail recorta `<style>`). Ver ADR-001.
4. **REQ-004**: En clientes de email que no soportan `@media` (ej. Outlook desktop) y en los que recortan `<style>` (Gmail app/web), las cards se ven idénticas porque el layout es intrínsecamente vertical (estilos inline). La degradación aceptable aplica solo a los paddings del header/título en pantallas angostas.
5. **REQ-005**: Se debe extraer la lógica de renderizado de cards a un helper compartido (`internal/services/email_components.go`, `RenderEmailCard`) para que **todos** los emails de datos (actuales y futuros) usen el mismo patrón, en vez de HTML duplicado por archivo.
6. **REQ-006**: El refactor no debe cambiar el contenido/información mostrada en ningún email (mismos campos, mismos datos), solo el markup/CSS.
7. **REQ-007**: Actualizar `bill_summary_email.go` y `debt_due_email.go` para usar el nuevo helper de tabla responsive. Los emails de texto simple (`bill_email.go`, `pension_email.go`, que usan `<p>`/`<ul>` en vez de tablas) no requieren cambio estructural, solo heredan la nueva plantilla base y paleta de colores.

### 2.2 Paleta de colores configurable (P0 — Obligatorios)

8. **REQ-008**: Debe existir una sección nueva en Settings, **"Apariencia de Emails"**, donde el usuario pueda configurar:
   - Color principal (header + links + acentos) — default `#007aff`
   - Color de fondo de página del email — default `#f5f5f7`
   - Color de fondo de la tarjeta/contenido — default `#ffffff`
   - Color de texto principal — default `#1d1d1f`
   - Color de texto secundario/footer — default `#8e8e93`
   - Color de bordes/separadores — default `#e5e5ea`
9. **REQ-009**: Los valores de color se guardan en `system_settings` (claves nuevas `email_color_primary`, `email_color_background`, `email_color_card`, `email_color_text`, `email_color_muted`, `email_color_border`), siguiendo el mismo patrón key-value que `smtp_*` (SPEC-029) y `currency_*` (SPEC-058). **Sin tablas ni migraciones nuevas.**
10. **REQ-010**: Cada color se valida con formato hex estricto (`^#[0-9A-Fa-f]{6}$`) antes de persistir. Un valor inválido se rechaza (no se guarda) y la API devuelve 400.
11. **REQ-011**: `EmailService.RenderTemplate` debe leer la paleta configurada (o los defaults si no hay nada configurado) e inyectarla en la plantilla en el momento de renderizar cada email. Si el usuario no configuró nada, el email se ve **exactamente igual que hoy** (mismos defaults).
12. **REQ-012**: Los colores semánticos de los badges de antigüedad de factura (`ageBadge`: gris/naranja/rojo según urgencia) **NO son parte de la paleta configurable** — se mantienen fijos porque codifican significado (urgencia), no marca. Ver ADR-002.
13. **REQ-013**: La UI de Settings debe mostrar, para cada color, un `<input type="color">` sincronizado con un input de texto hex, y un botón **"Restablecer a valores por defecto"** que reponga los 6 defaults sin necesidad de recordarlos.

### 2.3 Deseables (P1)

14. **REQ-014**: Endpoint `POST /api/system-settings/preview-email` que reciba una paleta candidata (aún no guardada) y devuelva el HTML renderizado de un email de ejemplo, para que la UI de Settings muestre una preview en vivo (ej. en un `<iframe sandbox>`) antes de guardar.
15. **REQ-015**: Agregar la nueva sección de Settings como subpágina dedicada (`/settings/email-appearance`), siguiendo el patrón índice + subpáginas introducido en SPEC-075, en vez de seguir agregando secciones a una página larga.

### 2.4 No objetivos

- No se rediseña el contenido/copy de los emails, solo el layout y la paleta.
- No se agrega un editor de plantillas libre (drag & drop, HTML custom del usuario) — la paleta cubre colores, no estructura.
- No se cambia el proveedor de envío (sigue siendo `net/smtp` vía SMTP configurado por el usuario).

### 2.5 Requerimientos No Funcionales

- **Compatibilidad**: Verificar visualmente en Apple Mail (iOS, el cliente de la captura), Gmail app (iOS/Android) y Gmail web. Outlook desktop debe seguir siendo legible (degradación aceptable, no rota).
- **iHost**: Cero dependencias nuevas. Los helpers de tabla y el CSS van embebidos como strings Go, igual que la plantilla actual.
- **Compatibilidad hacia atrás**: Sin paleta configurada, el resultado visual debe ser pixel-equivalente al actual.

---

## 3. Investigación (código actual)

- `internal/services/email_service.go`: constante `emailTemplateHTML`, plantilla única con placeholders `{{TITLE}}`, `{{CONTENT}}`, `{{DATE}}`. Tabla contenedora `width="600"` fija — **causa raíz #1**.
- `internal/services/bill_summary_email.go` → `renderBillSummaryContent`: arma manualmente una `<table>` de 5 columnas (Institución, Servicio, Período, Monto, Antigüedad) por cada casa (`groupByHome`). Esta es la tabla que aparece en la captura de pantalla que motivó la spec — **causa raíz #2**.
- `internal/services/debt_due_email.go` → `renderDebtDueContent`: mismo patrón de tabla de 5 columnas (Acreedor, Deuda, Cuota, Vence, Monto) + fila de total.
- `internal/services/bill_email.go` y `pension_email.go`: usan `<p>`/`<ul>` sin tablas — no tienen el problema de columnas, pero heredan el `width="600"` fijo de la plantilla base.
- Helper `esc()` (en `bill_summary_email.go`) ya centraliza el escape de HTML — se reutiliza en el nuevo helper compartido.
- `internal/services/system_settings.go` + `internal/storage/system_settings.go`: patrón key-value ya usado para SMTP (SPEC-029) y formato de moneda (SPEC-058) — mismo patrón a seguir para la paleta de colores.
- `internal/api/system_settings_handlers.go`: handlers `GET`/`PUT /api/system-settings` — punto de extensión natural.
- `frontend/src/pages/SettingsEmailAlertsPage.tsx`: página de referencia para el estilo del formulario nuevo (toggles, guardado, `useToast`, `api.systemSettings`).
- `docs/specs/SPEC-075-*.md`: introduce el patrón índice + subpáginas para Settings (ya released) — la nueva sección de apariencia de emails debería seguir ese patrón en vez de crecer una página existente.

---

## 4. Decisiones Técnicas (ADRs)

**ADR-001: Cards de datos con tablas anidadas y estilos inline (NO stacked-table vía media query)**
- **Contexto**: La primera implementación usó el patrón "stacked table" (`data-label` + `display:block` + `::before` en una media query del `<style>` del `<head>`). Resultó **insuficiente**: Gmail (app móvil y web) **recorta el `<style>` del `<head>`**, con lo cual la media query nunca aplica y la tabla de 5 columnas vuelve a comprimirse en mobile (el problema original de la captura). Verificado visualmente por el usuario.
- **Decisión**: En vez de tablas de N columnas, cada registro (factura/cuota) se renderiza como una **card vertical** con pares `etiqueta: valor` construida con tablas anidadas (`<table>` dentro de `<td>`) y **estilos 100% inline**. No depende de `<style>` ni media queries: el layout es inherentemente vertical y se ve bien en cualquier cliente (Gmail app/web, Apple Mail, Outlook) y cualquier ancho.
- **Consecuencias**: ✅ Funciona en clientes que recortan `<style>` (Gmail). ✅ Cero JS, cero dependencias. ✅ El mismo markup sirve para desktop y mobile (diseño mobile-first). ⚠️ En desktop ya no hay "tabla compacta": cada registro es una card apilada (aceptado por el usuario, mejora la legibilidad en ambos). ⚠️ El `<style>` de la plantilla base se mantiene solo para el header/título/contenido (paddings y font-sizes en pantallas angostas), no para los datos.

**ADR-002: Colores semánticos de urgencia (badges) quedan fuera de la paleta configurable**
- **Contexto**: `ageBadge` usa gris/naranja/rojo para indicar antigüedad de una factura pendiente. Si el usuario personaliza toda la paleta, podría sin querer romper la semántica "rojo = urgente".
- **Decisión**: La paleta configurable cubre colores de marca/layout (header, fondo, texto), no los colores de estado. Los badges de antigüedad se mantienen fijos.
- **Consecuencias**: ✅ Se preserva la legibilidad semántica sin importar la paleta elegida. ⚠️ Si en el futuro se pide personalizar también esos colores, se puede agregar como P2 con su propia validación de contraste.

**ADR-003: Paleta en `system_settings` (key-value), sin tabla nueva**
- **Contexto**: Mismo razonamiento que SPEC-029 (SMTP) y SPEC-058 (formato de moneda): la app es single-user, local, y `system_settings` ya es el mecanismo estándar de configuración.
- **Decisión**: 6 claves nuevas de texto (hex color), sin tabla ni migración nueva.
- **Consecuencias**: ✅ Cero migraciones. ✅ Consistente con el resto del sistema. ⚠️ Validación de formato debe hacerse en la capa de servicio (igual que `validSeparator` en `system_settings.go`).

---

## 5. Diseño Técnico

### 5.1 Nuevo helper compartido: `internal/services/email_components.go` (NUEVO)

```go
package services

import (
	"fmt"
	"strings"
)

// EmailField es un par etiqueta:valor de una card de email.
type EmailField struct {
	Label string
	Value string // ya formateado/escapado por el caller
	// HTML indica que Value es HTML propio (ej. el badge de antigüedad) y no
	// debe escaparse.
	HTML bool
	// Bold renderiza el valor en negrita (ej. el total).
	Bold bool
}

// RenderEmailCard arma una card vertical con pares etiqueta: valor usando
// tablas anidadas y estilos 100% inline. NO depende de <style> ni media
// queries: se ve igual en Gmail app/web (que recortan <style>), Apple Mail,
// Outlook y cualquier ancho de pantalla (ADR-001). Reemplaza el HTML de
// tablas duplicado en bill_summary_email.go y debt_due_email.go.
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
```

`bill_summary_email.go` y `debt_due_email.go` pasan de construir `<table>` a mano a llamar `RenderEmailCard(fields, palette)` por cada registro (factura/cuota) y una card final para el "Total del día" (con `Bold: true`). La paleta se recibe como parámetro en `renderBillSummaryContent(pending, format, palette)` / `renderDebtDueContent(due, format, palette)`.

### 5.2 Plantilla base actualizada: `internal/services/email_service.go`

Cambios sobre `emailTemplateHTML`:

- Tabla contenedora: `width="600"` → `width="100%" style="max-width:600px;..."`.
- Nuevo `<style>` en `<head>` con:
  - Variables de color inyectadas como placeholders (`{{COLOR_PRIMARY}}`, `{{COLOR_BG}}`, `{{COLOR_CARD}}`, `{{COLOR_TEXT}}`, `{{COLOR_MUTED}}`, `{{COLOR_BORDER}}`).
  - `@media screen and (max-width: 480px)`: reduce paddings/font-size del header, título y contenido.
  - **NO hay reglas de tablas de datos**: las cards de datos usan estilos inline (ADR-001) y no necesitan `<style>`.

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin:0; padding:0; background-color:{{COLOR_BG}}; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif; }
    a { color:{{COLOR_PRIMARY}}; }

    @media screen and (max-width: 480px) {
      .p40la-header { padding:16px 20px !important; }
      .p40la-title { font-size:18px !important; padding:20px 20px 0 20px !important; }
      .p40la-content { padding:12px 20px 20px 20px !important; font-size:14px !important; }
    }
  </style>
</head>
<body>
  <table width="100%" cellpadding="0" cellspacing="0" style="background-color:{{COLOR_BG}};padding:24px 0;">
    <tr>
      <td align="center">
        <table width="100%" cellpadding="0" cellspacing="0" style="max-width:600px;background-color:{{COLOR_CARD}};border-radius:16px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
          <tr>
            <td class="p40la-header" style="background-color:{{COLOR_PRIMARY}};padding:24px 32px;">
              <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:700;">P40LA</h1>
            </td>
          </tr>
          <tr>
            <td class="p40la-title" style="padding:32px 32px 0 32px;">
              <h2 style="margin:0;color:{{COLOR_TEXT}};font-size:20px;font-weight:600;">{{TITLE}}</h2>
            </td>
          </tr>
          <tr>
            <td class="p40la-content" style="padding:16px 32px 32px 32px;color:{{COLOR_TEXT}};font-size:15px;line-height:1.6;">
              {{CONTENT}}
            </td>
          </tr>
          <tr>
            <td style="padding:16px 32px;border-top:1px solid {{COLOR_BORDER}};background-color:{{COLOR_BG}};">
              <p style="margin:0;color:{{COLOR_MUTED}};font-size:12px;">
                P40LA iHost — Alerta automática generada el {{DATE}}
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>
```

> Nota: el header mantiene texto blanco fijo (`#ffffff`) sobre `{{COLOR_PRIMARY}}`. Si el usuario elige un primario muy claro, el contraste puede degradarse — ver Riesgos (§7).

### 5.3 `EmailService.RenderTemplate` (cambio de firma)

```go
// RenderTemplate renderiza la plantilla HTML única con título, contenido
// y la paleta de colores configurada (o los defaults si no hay nada
// configurado). Requiere ctx para leer la paleta desde SystemSettingsService.
func (s *EmailService) RenderTemplate(ctx context.Context, title, contentHTML string) string {
	palette, _ := s.settings.GetEmailPalette(ctx) // en error, devuelve defaults (ver 5.4)
	date := time.Now().Format("02/01/2006")

	out := emailTemplateHTML
	out = strings.ReplaceAll(out, "{{TITLE}}", title)
	out = strings.ReplaceAll(out, "{{CONTENT}}", contentHTML)
	out = strings.ReplaceAll(out, "{{DATE}}", date)
	out = strings.ReplaceAll(out, "{{COLOR_PRIMARY}}", palette.Primary)
	out = strings.ReplaceAll(out, "{{COLOR_BG}}", palette.Background)
	out = strings.ReplaceAll(out, "{{COLOR_CARD}}", palette.Card)
	out = strings.ReplaceAll(out, "{{COLOR_TEXT}}", palette.Text)
	out = strings.ReplaceAll(out, "{{COLOR_MUTED}}", palette.Muted)
	out = strings.ReplaceAll(out, "{{COLOR_BORDER}}", palette.Border)
	return out
}
```

**Impacto en call sites**: todos los que llaman `RenderTemplate(title, content)` (scheduler de facturas, scheduler de deudas, handlers de pensión, `SendTest`) ya tienen un `ctx context.Context` disponible en su función — se les agrega el parámetro `ctx` al llamado. Cambio mecánico, sin lógica nueva.

### 5.4 `internal/services/system_settings.go` (EXTENDIDO)

```go
// ---- Paleta de colores de emails (SPEC-077) ----

type EmailPalette struct {
	Primary    string
	Background string
	Card       string
	Text       string
	Muted      string
	Border     string
}

func DefaultEmailPalette() EmailPalette {
	return EmailPalette{
		Primary:    "#007aff",
		Background: "#f5f5f7",
		Card:       "#ffffff",
		Text:       "#1d1d1f",
		Muted:      "#8e8e93",
		Border:     "#e5e5ea",
	}
}

var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ValidHexColor valida el formato estricto #RRGGBB (exportada para
// la preview de la API, REQ-014).
func ValidHexColor(s string) bool { return hexColorRe.MatchString(s) }

// GetEmailPalette devuelve la paleta configurada, rellenando con defaults
// cualquier clave ausente o inválida.
func (s *SystemSettingsService) GetEmailPalette(ctx context.Context) (EmailPalette, error) {
	p := DefaultEmailPalette()
	keys := map[string]*string{
		"email_color_primary":    &p.Primary,
		"email_color_background": &p.Background,
		"email_color_card":       &p.Card,
		"email_color_text":       &p.Text,
		"email_color_muted":      &p.Muted,
		"email_color_border":     &p.Border,
	}
	for key, field := range keys {
		val, err := s.storage.Get(ctx, key)
		if err != nil {
			return p, err
		}
		if validHexColor(val) {
			*field = val
		}
	}
	return p, nil
}

// SetEmailPalette valida y persiste solo los campos no vacíos (permite
// updates parciales, igual que SetSMTPConfig). Devuelve error si algún
// campo enviado es un hex inválido.
func (s *SystemSettingsService) SetEmailPalette(ctx context.Context, p EmailPalette) error {
	fields := map[string]string{
		"email_color_primary":    p.Primary,
		"email_color_background": p.Background,
		"email_color_card":       p.Card,
		"email_color_text":       p.Text,
		"email_color_muted":      p.Muted,
		"email_color_border":     p.Border,
	}
	for key, val := range fields {
		if val == "" {
			continue
		}
		if !validHexColor(val) {
			return fmt.Errorf("color inválido para %s: %q (formato esperado #RRGGBB)", key, val)
		}
		if err := s.storage.Set(ctx, key, val); err != nil {
			return err
		}
	}
	return nil
}

// ResetEmailPalette borra las claves de paleta (vuelve a los defaults).
func (s *SystemSettingsService) ResetEmailPalette(ctx context.Context) error {
	for _, key := range []string{
		"email_color_primary", "email_color_background", "email_color_card",
		"email_color_text", "email_color_muted", "email_color_border",
	} {
		if err := s.storage.Set(ctx, key, ""); err != nil {
			return err
		}
	}
	return nil
}
```

### 5.5 API — `internal/api/system_settings_handlers.go` (EXTENDIDO)

**`GET /api/system-settings`** — agrega:
```json
{
  "email_color_primary": "#007aff",
  "email_color_background": "#f5f5f7",
  "email_color_card": "#ffffff",
  "email_color_text": "#1d1d1f",
  "email_color_muted": "#8e8e93",
  "email_color_border": "#e5e5ea"
}
```
(siempre presentes; si no hay nada configurado, se devuelven los defaults — no sensibles, se pueden exponer sin restricción).

**`PUT /api/system-settings`** — acepta cualquiera de las 6 claves (parcial). Si algún color enviado no matchea `^#[0-9A-Fa-f]{6}$`, responde **400** con:
```json
{ "error": "invalid_color", "message": "Color inválido para email_color_primary: \"azul\" (formato esperado #RRGGBB)" }
```

**`POST /api/system-settings/email-palette/reset`** (NUEVO) — resetea los 6 colores a default. Response 200 con la paleta default.

**`POST /api/system-settings/preview-email`** (NUEVO, P1) — Request: paleta candidata (mismos 6 campos, todos opcionales, faltantes = default). Response: `{ "html": "<!DOCTYPE html>..." }` con un email de ejemplo (reutiliza `renderBillSummaryContent` con datos de muestra hardcodeados) renderizado con esa paleta, **sin guardarla**.

### 5.6 Frontend (NUEVO — `frontend/src/pages/SettingsEmailAppearancePage.tsx`)

Sigue el patrón de SPEC-075 (subpágina navegable desde el índice de Settings, registrada en `BACK_ROUTES`) y el patrón de estado/guardado de `SettingsEmailAlertsPage.tsx`:

- 6 filas, cada una con `<input type="color">` + `<input type="text">` (hex) sincronizados, label descriptivo ("Color principal (header, links)", "Fondo de página", etc.).
- Botón **"Guardar"** → `PUT /api/system-settings` con los 6 campos.
- Botón **"Restablecer a valores por defecto"** → `POST /api/system-settings/email-palette/reset`, recarga los inputs.
- (P1) Panel de preview: `<iframe sandbox="allow-same-origin">` que carga el HTML devuelto por `POST /api/system-settings/preview-email` cada vez que cambia un color (debounced).
- Entrada en el índice de Settings (`SettingsPage.tsx`, patrón `SettingsNavRow`): "Apariencia de Emails" con subtítulo "Colores personalizados" o "Predeterminado" según si hay paleta configurada.
- i18n: nuevas claves bajo `settings.email_appearance.*` en los archivos de idioma existentes.

### 5.7 Modelo de datos

Sin tablas ni migraciones nuevas. Claves nuevas en `system_settings` (key-value existente):

```
email_color_primary     TEXT   -- hex, ej "#007aff"
email_color_background  TEXT   -- hex, ej "#f5f5f7"
email_color_card        TEXT   -- hex, ej "#ffffff"
email_color_text        TEXT   -- hex, ej "#1d1d1f"
email_color_muted       TEXT   -- hex, ej "#8e8e93"
email_color_border      TEXT   -- hex, ej "#e5e5ea"
```

---

## 6. Criterios de Aceptación

### 6.1 Responsive

- [ ] CA-001: El email de resumen diario de facturas muestra **cada factura como una card vertical** con pares `Institución / Servicio / Período / Monto / Antigüedad` (label arriba-izquierda, valor a la derecha), legible en cualquier ancho — incluso en Gmail app/web que recortan `<style>`.
- [ ] CA-002: Lo mismo aplica al email de cuotas de deudas del día (card por cuota + card "Total del día" en negrita).
- [ ] CA-003: En desktop (≥600px) las cards se ven igual de legibles (layout mobile-first, sin regresión: misma información, mejor presentación).
- [ ] CA-004: El badge de antigüedad (`ageBadge`) se sigue viendo como badge de color, no como texto plano.
- [ ] CA-005: Ningún dato/campo se pierde ni cambia respecto al contenido actual (verificado comparando el texto plano extraído, `stripHTML`, antes y después).
- [ ] CA-006: El email se ve legible en Outlook desktop y Gmail (los estilos inline de las cards no dependen de `<style>`).

### 6.2 Paleta de colores

- [ ] CA-007: `GET /api/system-settings` siempre incluye los 6 campos `email_color_*`, con defaults cuando no hay nada configurado.
- [ ] CA-008: `PUT /api/system-settings` con un color inválido (ej. `"email_color_primary": "azul"`) devuelve 400 y no persiste ningún cambio de paleta.
- [ ] CA-009: `PUT /api/system-settings` con colores válidos los persiste, y el siguiente email enviado (de cualquier tipo) usa esos colores.
- [ ] CA-010: Sin ninguna configuración de paleta (instalación nueva), el email enviado es visualmente idéntico al actual (mismos hex por default).
- [ ] CA-011: El botón "Restablecer a valores por defecto" en la UI limpia los 6 colores configurados y el próximo email vuelve a los defaults.
- [ ] CA-012: Los colores de los badges de antigüedad (`ageBadge`) NO cambian sin importar la paleta configurada.
- [ ] CA-013 (P1): El endpoint de preview devuelve HTML renderizado con la paleta candidata sin persistir nada en `system_settings`.

### 6.3 No funcionales

- [ ] CA-NF-001: `go build ./...` y `go vet ./...` sin errores.
- [ ] CA-NF-002: Cero dependencias nuevas en `go.mod`.
- [ ] CA-NF-003: El binario no crece más de ~20KB por el CSS/helper nuevos.

### 6.4 Testing

- **Unit tests**:
  - `email_components_test.go`: `RenderEmailCard` con distintos campos, verificar escape de HTML, `HTML: true` (badge), `Bold: true` (total) y uso de la paleta.
  - `system_settings_email_palette_test.go`: `GetEmailPalette` con claves ausentes → defaults; `SetEmailPalette` con hex inválido → error, sin persistir; reset → vuelve a defaults.
  - `email_service_test.go`: `RenderTemplate` inyecta los 6 placeholders de color correctamente; sin paleta configurada, placeholders quedan en los valores default.
- **Integration tests**: `PUT /api/system-settings` con color inválido → 400; con colores válidos → 200 y el siguiente `SendTest` usa la nueva paleta.
- **Manual / visual**: abrir los 4 tipos de email (resumen facturas, deudas del día, factura nueva, pensión) en Apple Mail iOS, Gmail app y Gmail web, en un dispositivo o simulador angosto, antes y después del refactor.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Usuario elige un color primario muy claro → texto blanco del header ilegible | Media | Bajo | Documentar en la UI ("elegí un color oscuro para el header"); P2: calcular automáticamente si usar texto blanco o negro según luminancia del primario |
| Gmail recorta `<style>` del `<head>` | **Alta** | **Alto** | **Cards con estilos inline (ADR-001): el layout de datos no depende de `<style>`; la única pérdida es el padding reducido del header en pantallas angostas, cosmético.** |
| Cambiar la firma de `RenderTemplate` a `(ctx, title, content)` rompe call sites no actualizados | Media | Medio | Buscar todos los usos (`grep -rn "RenderTemplate("`) y actualizarlos en la misma fase; cubierto por `go build` |
| Preview en vivo (P1) agrega complejidad de sandboxing del iframe | Baja | Bajo | Si se posterga, el P0 (guardar + ver el próximo email real) sigue siendo funcional sin preview |

---

## 8. Plan de Implementación

| Fase | Descripción | Estimación |
|------|-------------|------------|
| 1 | `email_components.go`: `RenderEmailCard` + tests | 0.5 día |
| 2 | Refactor plantilla base (`email_service.go`): `width="100%"`, `<style>` + media query, placeholders de color | 0.5 día |
| 3 | Migrar `bill_summary_email.go` y `debt_due_email.go` a `RenderEmailCard` | 0.5 día |
| 4 | `system_settings.go`: `EmailPalette`, `GetEmailPalette`/`SetEmailPalette`/`ResetEmailPalette` + tests | 0.5 día |
| 5 | `RenderTemplate(ctx, ...)`: actualizar firma y todos los call sites | 0.5 día |
| 6 | API: extender GET/PUT `/api/system-settings`, endpoint `email-palette/reset` | 0.5 día |
| 7 | Frontend: `SettingsEmailAppearancePage.tsx` + entrada en índice de Settings + i18n | 1 día |
| 8 | (P1) Endpoint `preview-email` + iframe de preview en frontend | 0.5 día |
| 9 | QA visual en Apple Mail / Gmail app / Gmail web / Outlook desktop + build/vet/tests | 0.5 día |

**Estimación total (P0)**: ~3.5 días. **+0.5 día si se incluye el preview en vivo (P1)**.

---

## 9. Notas y Referencias

- Screenshot que motivó esta spec: email de resumen diario de facturas visto en Apple Mail (iOS), tabla de 5 columnas comprimida.
- ADR-001 revisado tras QA del usuario: el patrón "stacked table" vía media query **no funciona en Gmail** (recorta `<style>`); se reemplazó por cards con tablas anidadas y estilos inline ("bulletproof" para email, técnica estándar documentada en Litmus/Email on Acid).
- Specs relacionadas: SPEC-029 (sistema de emails original), SPEC-030 (email de factura nueva), SPEC-032/037 (toggles de alertas por mail), SPEC-051 (emails de pensión), SPEC-058 (formato de moneda, mismo patrón key-value en `system_settings`), SPEC-075 (patrón índice + subpáginas de Settings a seguir para la nueva sección).

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-15 | Claude (asistido) | Creación inicial de la especificación a partir de una captura de email mal renderizado en mobile. |
| 2026-09-16 | opencode (asistido) | ADR-001 ampliado: `EmailRow.Summary` para la fila "Total del día" de deudas (colspan en desktop, apilada en mobile) + clase CSS `.p40la-table-summary`. |
| 2026-09-16 | opencode (asistido) | **ADR-001 reemplazado** tras QA del usuario: el stacked-table vía media query se ve mal en Gmail (recorta `<style>`). Se adopta el patrón **cards con tablas anidadas y estilos inline** (`RenderEmailCard`), independiente de `<style>`/media queries. Se actualizaron REQ-003/004/005, §5.1, §5.2 (CSS sin reglas de tablas), CA-001/002/003/006 y la tabla de riesgos. |
| 2026-09-16 | opencode (asistido) | Estado `released`. Issue #80 cerrado. Commit de implementación: `3c9ae9f` |
| 2026-09-16 | opencode (asistido) | Commit de release documentado: merge `ad4c177` a `main` (Merge branch 'feature/SPEC-077') |
