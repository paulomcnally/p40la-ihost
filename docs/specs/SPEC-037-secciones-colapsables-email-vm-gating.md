---
title: "Secciones colapsables de Email/Voice Monkey y gating de toggles de email"
id: "SPEC-037"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 37
---

# Secciones colapsables de Email/Voice Monkey y gating de toggles de email

**ID**: SPEC-037  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

En la sección **Alertas** de Configuraciones hay dos problemas de UX:

1. **Bug de gating**: los toggles de canal **Mail** de cada alerta (`SettingsPage.tsx:340`) se pueden activar aunque no haya SMTP configurado ni destinatarios cargados, a diferencia del canal **Alexa** (`SettingsPage.tsx:344`) que se deshabilita cuando Voice Monkey no está activo (`vmActive`). El backend tampoco valida (`alerts_handlers.go:54`). Para activar los toggles de email se debe exigir **SMTP configurado Y al menos un destinatario**.
2. **UX de colapso**: la sección **Alertas por Email** muestra los destinatarios siempre visibles y fuera del acordeón SMTP, cuando en realidad toda la sección (SMTP + destinatarios + test) es una misma feature. **Voice Monkey** cuando está habilitado se ve completamente abierto siempre.

Se propone unificar el patrón de ambas features: un **toggle maestro de activación** (email y Voice Monkey, espejo el uno del otro) y una **sección colapsable** que se auto-abre solo si falta configurar algo. Se agrega el setting `email_alerts_enabled` (espejo de `voicemonkey_enabled`) y el gating de los toggles de email en frontend Y backend.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar setting `email_alerts_enabled` (bool, default false) espejo de `voicemonkey_enabled`, con toggle maestro en la sección "Alertas por Email".
2. **REQ-002**: **Gating de toggles de email**: los toggles de canal Mail de cada alerta solo se pueden activar si `email_alerts_enabled=true` **Y** `smtp_configured=true` **Y** existe al menos 1 destinatario (`alertEmails.length > 0`). Si no se cumplen, el toggle aparece deshabilitado con hint explicativo.
3. **REQ-003**: **Gating en backend**: `UpdateAlert` debe rechazar (400/422) intentos de activar `mail_enabled=true` si no hay SMTP configurado o no hay destinatarios, y `voice_enabled=true` si Voice Monkey no está activo (espejo del frontend).
4. **REQ-004**: **Sección colapsable "Alertas por Email"**: todo el contenido (SMTP, destinatarios, botón test) vive dentro de la sección colapsable. Colapsada por defecto cuando está configurada; auto-abierta si la feature está activa pero falta configurar SMTP o destinatarios.
5. **REQ-005**: **Sección colapsable Voice Monkey**: mismo comportamiento — colapsada si configurada, auto-abierta si activa pero falta configurar.
6. **REQ-006**: Los **destinatarios** pasan a estar dentro de la sección colapsable de email (hoy están fuera del acordeón).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Cuando una feature (email o Voice Monkey) está activa pero con falta de configuración, mostrar hint visual indicando qué falta (ej: "Falta configurar SMTP y/o agregar destinatarios").
2. **REQ-008**: Internacionalizar todas las cadenas nuevas en `frontend/public/i18n/{es,en}.json` (fuente de verdad, AGENTS.md) y correr `npm run build`.
3. **REQ-009**: Mantener el estado de colapso en estado local del componente (no persistir) para simplificar.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Al desactivar `email_alerts_enabled` o desconectar SMTP, los toggles de mail de las alertas permanecen guardados pero deshabilitados en UI (no borrar preferencias).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto (UI estática, una validación extra en backend).
- **Seguridad**: La validación de gating en backend evita activar alertas de mail sin configuración válida.
- **Almacenamiento**: 1 key nueva en `system_settings` (`email_alerts_enabled`), sin migración de schema (key-value).
- **Disponibilidad**: Endpoints existentes; `PUT /api/alerts/{key}` gana validación.
- **iHost**: Cero dependencias nuevas; solo un campo bool en handler + key en storage.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `frontend/src/pages/SettingsPage.tsx:340` — toggle Mail sin gating (bug).
- `frontend/src/pages/SettingsPage.tsx:343-346` — toggle Alexa con gating `vmActive` (patrón a imitar).
- `frontend/src/pages/SettingsPage.tsx:275` — `vmActive = vmEnabled && vmConfigured && vmSendAlerts`.
- `frontend/src/pages/SettingsPage.tsx:435-465` — destinatarios fuera del acordeón (chips + modal, SPEC-035).
- `frontend/src/pages/SettingsPage.tsx:363-433` — acordeón SMTP (`smtpOpen`).
- `frontend/src/pages/SettingsPage.tsx:502-569` — Voice Monkey siempre expandido cuando habilitado.
- `internal/api/alerts_handlers.go:54` — `UpdateAlert` sin validación de gating.
- `internal/services/system_settings.go:186-197` — Get/SetAlertEmails; patrón key-value existente para `voicemonkey_enabled`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Toggle explícito `email_alerts_enabled` | Espejo exacto de Voice Monkey, clara semántica de activación | Requiere campo nuevo en handler | ✅ Seleccionada (decisión usuario) |
| Derivar de `smtp_configured` | Sin setting nuevo | Sin toggle maestro; no permite desactivar sin borrar config | ❌ Rechazada |
| Gating solo frontend | Menos código backend | API pública permite activar toggles sin config; frágil | ❌ Rechazada |
| Gating frontend + backend | Consistente, robusto | Campo nuevo en `updateAlertRequest` | ✅ Seleccionada |
| Siempre colapsado al cargar | Simple | No guía al usuario que le falta configurar | ❌ Rechazada |
| Auto-abrir si falta configurar | Guía al usuario, no molesta si está ok | Lógica de estado inicial levemente mayor | ✅ Seleccionada (decisión usuario) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: `email_alerts_enabled` como key de system_settings
- **Contexto**: Se necesita un toggle maestro de email equivalente al de Voice Monkey.
- **Decisión**: Nueva key `email_alerts_enabled` (bool) en `system_settings`, expuesta como `email_alerts_enabled` en `GET/PUT /api/system-settings`. Default `false`. Sin migración (key-value).
- **Consecuencias**: Frontend y backend leen/escriben un bool; comportamiento simétrico a `voicemonkey_enabled`.

**ADR-002**: Gating en backend (UpdateAlert)
- **Contexto**: El gating solo en UI no impide activar `mail_enabled` vía API.
- **Decisión**: En `AlertsHandlers.UpdateAlert`, si se intenta poner `mail_enabled=true` y no hay SMTP configurado ni destinatarios, responder `422 validation_error`. Ídem para `voice_enabled=true` sin VM activo.
- **Consecuencias**: Endpoint más seguro; frontend muestra toast de error si por algún motivo llega a fallar.

**ADR-003**: Colapso con auto-apertura por falta de configuración
- **Contexto**: El usuario pidió que las secciones no se vean abiertas siempre, pero guíen al que falta configurar.
- **Decisión**: Estado de colapso local (`emailOpen`/`vmOpen`). Se inicializa abierto si la feature está activa y (no configurada o sin destinatarios, según el caso). Colapsado si está completa.
- **Refinamiento UX (validación manual)**: La sección **solo se puede desplegar si el toggle maestro está activo**. Con el master apagado no se muestra chevron y el contenido (SMTP, destinatarios, test) permanece oculto. El chevron aparece al activar el toggle y la auto-apertura guía a completar SMTP/destinatarios.
- **Consecuencias**: UX guiada sin persistir estado.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[SettingsPage]
   │
   ├─ Alertas por Email
   │    ├─ Toggle maestro email_alerts_enabled ──► PUT /api/system-settings
   │    ├─ Sección colapsable (emailOpen)
   │    │    ├─ SMTP (acordeón existente smtpOpen)
   │    │    ├─ Destinatarios (chips + modal, movidos adentro)
   │    │    └─ Botón test email
   │    └─ gates de alerta: mail toggle habilitado ⇔ email_enabled ∧ smtp_configured ∧ recipients>0
   │
   ├─ Alertas (catálogo)
   │    └─ toggle mail → PUT /api/alerts/{key}  [backend valida gating]
   │
   └─ Voice Monkey (Alexa)
        ├─ Toggle maestro voicemonkey_enabled (existente)
        └─ Sección colapsable (vmOpen): config + send_alerts + test
```

### 4.2 Componentes

#### 4.2.1 SettingsPage (secciones Email y Voice Monkey)
- **Responsabilidad**: Toggle maestro email, secciones colapsables, gating visual de toggles de mail, estado inicial de colapso.
- **Interfaz**: `api.systemSettings.get/update`, `api.alerts.update`, `useToast`, i18n.
- **Dependencias**: `EmailRecipientsModal` (SPEC-035), `Toggle`, `Icon`.
- **Ubicación**: `frontend/src/pages/SettingsPage.tsx`

#### 4.2.2 Backend — system settings
- **Responsabilidad**: Exponer/persistir `email_alerts_enabled`.
- **Interfaz**: campo `EmailAlertsEnabled *bool` en `settingsRequest`; lectura en `GetSystemSettings`.
- **Ubicación**: `internal/api/system_settings_handlers.go`, `internal/services/system_settings.go`

#### 4.2.3 Backend — alerts handlers
- **Responsabilidad**: Validar gating al activar `mail_enabled` / `voice_enabled`.
- **Interfaz**: `PUT /api/alerts/{key}` con respuesta `422 validation_error` si no aplica.
- **Ubicación**: `internal/api/alerts_handlers.go`

### 4.3 Modelo de datos

```
system_settings (key-value, sin migración)
- email_alerts_enabled: "true"|"false"   (default "false")
- alert_emails: string comma-separated   (existente)

alerts (tabla existente, sin cambios)
- mail_enabled / voice_enabled: bool
```

### 4.4 APIs / Contratos

#### Endpoint: `PUT /api/system-settings` (extendido)

**Request**:
```json
{
  "email_alerts_enabled": true
}
```

**Response 200**: incluye `"email_alerts_enabled": true`.

#### Endpoint: `PUT /api/alerts/{key}` (con validación)

**Request** (activación de mail sin prerequisitos):
```json
{
  "mail_enabled": true
}
```

**Response Error** (si SMTP no configurado o sin destinatarios):
```json
{
  "error": "validation_error",
  "message": "Para activar alertas por email se necesita SMTP configurado y al menos un destinatario."
}
```

### 4.5 Dependencias

- **Internas**: `internal/api/system_settings_handlers.go`, `internal/api/alerts_handlers.go`, `internal/services/system_settings.go`, `frontend/src/pages/SettingsPage.tsx`, `frontend/src/api/index.ts` (tipos), `frontend/public/i18n/{es,en}.json`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Con SMTP no configurado o sin destinatarios, el toggle Mail de cada alerta está deshabilitado con hint. Activarlo vía API devuelve `422 validation_error`.
- [x] CA-002: Con SMTP configurado Y ≥1 destinatario Y `email_alerts_enabled=true`, el toggle Mail se habilita y se puede activar (frontend y API).
- [x] CA-003: Existe el toggle maestro "Alertas por Email" que persiste `email_alerts_enabled`; al desactivarlo se oculta/colapsa la sección y se deshabilitan los toggles de mail.
- [x] CA-004: Los destinatarios están dentro de la sección colapsable de email (no visibles cuando la sección está cerrada).
- [x] CA-005: La sección de email se auto-abre si la feature está activa y falta configurar SMTP o destinatarios; queda colapsada si está completa.
- [x] CA-006: La sección Voice Monkey se colapsa cuando está configurada y se auto-abre si está activa pero falta token/device.
- [x] CA-007: Al desactivar `email_alerts_enabled` no se borran los `mail_enabled` guardados (P2, vuelven habilitados al reactivar si hay config válida).
- [x] CA-008: Con el toggle maestro apagado la sección no es desplegable (sin chevron); al activarlo aparece el chevron y se puede expandir/colapsar.
- [ ] CA-006: La sección Voice Monkey se colapsa cuando está configurada y se auto-abre si está activa pero falta token/device.
- [ ] CA-007: Al desactivar `email_alerts_enabled` no se borran los `mail_enabled` guardados (P2, vuelven habilitados al reactivar si hay config válida).

### 5.2 No funcionales

- [x] CA-NF-001: `npm run build` en `frontend/` sin errores e i18n servido por `http://localhost:8088/i18n/es.json`.
- [x] CA-NF-002: `go build ./...` y `go test ./...` pasan; `go vet` sin errores.

### 5.3 Testing

- **Unit tests**: `UpdateAlert` valida gating de mail/voice (nuevos casos en `alerts_handlers_test.go` o services). Get/Set de `email_alerts_enabled`.
- **Integration tests**: Flujo PUT system-settings (email_alerts_enabled) + PUT alerts con/ sin prerequisitos.
- **E2E tests**: Activar/desactivar toggle maestro email; intentar activar mail sin SMTP; agregar destinatario y ver habilitación; colapso/auto-apertura de ambas secciones.
- **Carga/Performance**: Sin métricas nuevas.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: `email_alerts_enabled` (handler + service) y validación gating en `UpdateAlert` | 0.5 día | Ninguna |
| 2 | Frontend: toggle maestro email, mover destinatarios dentro, secciones colapsables (email + VM) con auto-apertura, gating visual de toggles mail | 0.5-1 día | Fase 1 |
| 3 | i18n es/en + `npm run build` + verificación `curl` + tests | 0.25 día | Fase 2 |

### 6.2 Milestones

1. **MVP**: Gating (frontend + backend) del bug de email + secciones colapsables de Email y Voice Monkey con auto-apertura.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Regresión en SPEC-035 (destinatarios modal) al moverlos de lugar | Media | Medio | Mantener `EmailRecipientsModal` intacto; mover solo el contenedor |
| Gating backend rompa activación de voice_enabled legítima | Media | Bajo | Replicar lógica de `vmActive` (enabled ∧ configured ∧ send_alerts) en handler |
| Perder claves i18n editadas en `public/i18n/` | Baja | Medio | Editar SOLO en `frontend/public/i18n/` + build + `curl` |
| Estado de colapso inconsistente tras guardar | Baja | Bajo | Estado local; re-evaluar al recargar desde `smtp_configured`/`vm_configured` |

## 8. Notas y Referencias

- Patrón de gating Alexa existente: `SettingsPage.tsx:343-350`
- Acordeón SMTP existente: `SettingsPage.tsx:363-373`
- Destinatarios con modal (SPEC-035): `SettingsPage.tsx:435-465`, `EmailRecipientsModal.tsx`
- Regla i18n: AGENTS.md (fuente de verdad `frontend/public/i18n/`)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
| 2026-08-16 | paulomcnally | Completada y lista para desarrollo (pending_execution) |
| 2026-08-16 | paulomcnally | Inicio de desarrollo (in_progress) |
| 2026-08-16 | paulomcnally | Refinamiento UX durante validación manual: la sección solo es desplegable con el toggle maestro activo (sin chevron si está apagado) |
| 2026-08-16 | paulomcnally | Validación manual aprobada por el usuario; criterios de aceptación pass (pending_release) |
| 2026-08-16 | paulomcnally | Release: merge feature/SPEC-037 a main (64949af) y push |
