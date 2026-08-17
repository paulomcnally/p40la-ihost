---
title: "Estado Configurado y botón Reconfigurar para SMTP (mirror UI de Voice Monkey)"
id: "SPEC-034"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16 (released)"
github_issue: 34
---

# Estado Configurado y botón Reconfigurar para SMTP (mirror UI de Voice Monkey)

**ID**: SPEC-034  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16 (released)

---

## 1. Resumen Ejecutivo

En SPEC-033 se cambió la UI de la sección **Voice Monkey (Alexa)** en Settings: cuando el usuario ya configuró token y device (`voicemonkey_configured=true`), la sección deja de mostrar los inputs y muestra un estado **"Configurado"** con un botón **"Reconfigurar"** que limpia las credenciales y vuelve a mostrar el formulario (REQ-020 / CA-016 de SPEC-033).

La sección **SMTP** hoy muestra siempre el formulario completo (dentro de un acordeón), y cuando hay credenciales guardadas solo se muestra un badge "Configurado" junto a los inputs, con placeholders "dejar vacío para mantener el valor actual". El usuario pide aplicar el **mismo patrón de Voice Monkey al SMTP**: cuando ya está configurado, mostrar un estado "Configurado" (sin inputs) con un botón **"Reconfigurar"** que limpie las credenciales SMTP y permita volver a configurar desde cero.

El cambio es **100% frontend + un endpoint pequeño de limpieza en backend** (espejo del patrón `DeleteVoiceMonkey` → `DELETE /api/system-settings/voicemonkey`). No toca el `EmailService`, ni los schedulers, ni la lógica de envío: solo la forma en que se presenta y se limpa la configuración SMTP.

**Consideraciones de iHost**: Cero dependencias nuevas. Sin migraciones ni tablas nuevas (solo `system_settings`, ya existente). Un handler nuevo de ~20 líneas y cambios de UI/i18n. Consumo de RAM/CPU despreciable.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Cuando SMTP esté configurado (`smtp_configured=true`), la sección SMTP en Settings debe mostrar un estado **"Configurado"** (sin los inputs del formulario) con un botón **"Reconfigurar"** — espejo del patrón de Voice Monkey.
2. **REQ-002**: Al pulsar **"Reconfigurar"**, se deben **limpiar las credenciales SMTP** (`smtp_host`, `smtp_port`, `smtp_user`, `smtp_password`, `smtp_from_email`, `smtp_from_name`), `smtp_configured` pasa a `false` y se vuelve a mostrar el formulario de configuración.
3. **REQ-003**: Debe existir un endpoint de limpieza SMTP en backend (p. ej. `DELETE /api/system-settings/smtp`) y un método `ClearSMTP` en `SystemSettingsService`, espejo de `ClearVoiceMonkey` / `DeleteVoiceMonkey`.
4. **REQ-004**: La limpieza de SMTP NO debe afectar otras configuraciones: Voice Monkey, destinatarios (`alert_emails`), toggles de alertas (SPEC-032/033), hora de facturación, etc.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Antes de limpiar, debe mostrarse una **confirmación** al usuario (patrón `window.confirm` de Voice Monkey).
2. **REQ-006**: El botón **"Enviar email de prueba"** debe estar **deshabilitado** (gris) mientras SMTP no esté configurado, espejo del botón "Probar aviso" de Voice Monkey.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Mostrar en el **header del acordeón SMTP** el estado ("Configurado" / "No configurado") para visibilidad sin necesidad de abrirlo.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Cambios mínimos; un endpoint más. Impacto despreciable.
- **Seguridad — Protección de información sensible**: Se mantienen las reglas de SPEC-029/033: `smtp_user` y `smtp_password` **nunca** se devuelven en `GET /api/system-settings` ni se loguean. El endpoint de limpieza solo borra claves; no expone valores.
- **Almacenamiento**: Sin cambios de esquema. Solo claves existentes en `system_settings` (se vacían con `""` al reconfigurar, patrón `ClearVoiceMonkey`).
- **Disponibilidad**: Si el usuario reconfigura SMTP, el envío de emails se deshabilita hasta volver a configurar (comportamiento esperado y explícito por la confirmación).
- **iHost**: Cero dependencias nuevas. `DELETE /api/system-settings/smtp` con stdlib.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-033 (REQ-020 / CA-016)**: patrón "Configurado + Reconfigurar" ya implementado para Voice Monkey. Frontend: `handleReconfigureVoiceMonkey` (confirm + `api.systemSettings.disconnectVoiceMonkey()` + reset de estado). Backend: `ClearVoiceMonkey` en `internal/services/system_settings.go` y handler `DeleteVoiceMonkey` en `internal/api/system_settings_handlers.go`, ruta `DELETE /api/system-settings/voicemonkey` en `internal/api/routes.go:34`.
- **UI SMTP actual** (`frontend/src/pages/SettingsPage.tsx:322-388`): acordeón colapsable; inputs de host/port/user/password/from_email/from_name con placeholders "dejar vacío para mantener el valor actual" y un badge `Configurado`/`No configurado` inline (`toggleCls(smtpConfigured)`). No existe estado sin-inputs ni botón Reconfigurar.
- **`SystemSettingsService`** (`internal/services/system_settings.go`): `GetSMTPConfig`, `SetSMTPConfig` (patrón `setIfNonEmpty`), `IsSMTPConfigured`. `ClearVoiceMonkey` es el modelo para `ClearSMTP`.
- **Client API** (`frontend/src/api/index.ts`): `systemSettings.disconnectVoiceMonkey` (`DELETE /api/system-settings/voicemonkey`) es el modelo para `disconnectSMTP`.
- **i18n**: fuente de verdad en `frontend/public/i18n/{es,en}.json` bajo `settings.email_alerts.*`. Voice Monkey ya tiene `configured_status`, `reconfigure`, `reconfigure_confirm`, `reconfigured` — se replican para SMTP.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A: Estado "Configurado" + botón "Reconfigurar" que limpia credenciales (mirror Voice Monkey)** | Consistencia total con la UI que el usuario ya aprobó (SPEC-033); UX clara; sin ambigüedad sobre el valor guardado | Requiere un endpoint de limpieza nuevo (pequeño) | ✅ Seleccionada |
| **B: Mantener form con badge y placeholders "dejar vacío"** | Cero cambios | No cumple el requerimiento explícito del usuario (quiere el mismo comportamiento que Voice Monkey) | ❌ Rechazada |
| **C: Solo ocultar user/password (mostrar el resto)** | Menos código | Half-baked; no refleja el patrón Voice Monkey; sigue exponiendo inputs de host/from | ❌ Rechazada |
| **D: Botón "Limpiar credenciales" sin estado Configurado** | Simple | No es el patrón pedido; pierde el feedback visual de "ya configurado" | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Replicar el patrón `ClearVoiceMonkey`/`DeleteVoiceMonkey` para SMTP**
- **Contexto**: Se necesita una acción idempotente de "quitar la configuración SMTP" invocable desde la UI con confirmación.
- **Decisión**: Nuevo método `ClearSMTP(ctx)` en `SystemSettingsService` que vacía las 6 claves SMTP con `storage.Set(key, "")`, y nuevo handler `DeleteSMTP` expuesto en `DELETE /api/system-settings/smtp` con auth. Espejo exacto de `ClearVoiceMonkey` (que además resetea toggles) — en SMTP no hay toggles propios que resetear, solo credenciales.
- **Consecuencias**:
  - ✅ Consistencia de código y de API con SPEC-033
  - ✅ Idempotente y simple (un loop de `Set("")`)
  - ⚠️ Un endpoint nuevo más (mínimo, ~20 líneas)

**ADR-002: `smtp_configured` se calcula desde las claves; al limpiar pasa automáticamente a `false`**
- **Contexto**: `smtp_configured` no se persiste; se deriva en `isConfigured(cfg)` (user+password+host no vacíos).
- **Decisión**: No tocar la lógica de `isConfigured`. Al vaciar las claves, el GET devuelve `smtp_configured: false` sin cambios adicionales. La UI re-consulta el GET tras reconfigurar (patrón `handleSaveEmail` actual).
- **Consecuencias**:
  - ✅ Cero cambios en el cálculo de configuración
  - ✅ Sin estado inconsistente (no hay clave que pueda quedar desincronizada)

**ADR-003: La confirmación y el estado "Configurado" viven en el frontend**
- **Contexto**: El backend solo necesita limpiar; el flujo de UI (confirm → llamar DELETE → resetear estado local → toast) es del frontend.
- **Decisión**: `handleReconfigureSMTP` en `SettingsPage.tsx` replicando `handleReconfigureVoiceMonkey`. Claves i18n nuevas bajo `settings.email_alerts.*`.
- **Consecuencias**:
  - ✅ Frontend autocontenido, sin lógica nueva en backend más allá de `ClearSMTP`
  - ✅ Reutiliza `Toggle`, cards iOS, `showToast` existentes

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌────────────────────────────────────────────────────────────┐
│            SettingsPage.tsx (frontend) MODIFICADO          │
│  Acordeón "Configuración SMTP"                             │
│  ├─ Si smtpConfigured=false → formulario (inputs actuales) │
│  └─ Si smtpConfigured=true  → estado "Configurado"         │
│        + botón [Reconfigurar] (confirm)                    │
│  ─ Reconfigurar → DELETE /api/system-settings/smtp        │
│        → reset estado local → volver a formulario          │
└───────────────┬────────────────────────────────────────────┘
                │ DELETE /api/system-settings/smtp
                ▼
┌────────────────────────────────────────────────────────────┐
│  system_settings_handlers.go (EXTENDIDO)                   │
│  DeleteSMTP → settings.ClearSMTP(ctx)                      │
└───────────────┬────────────────────────────────────────────┘
                │
                ▼
┌────────────────────────────────────────────────────────────┐
│  SystemSettingsService (EXTENDIDO)                         │
│  ClearSMTP(ctx): vacía smtp_host/port/user/password/       │
│                  from_email/from_name con Set("")          │
└────────────────────────────────────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 `internal/services/system_settings.go` (EXTENDIDO)

Nuevo método (espejo de `ClearVoiceMonkey`, línea 286):
```go
// ClearSMTP limpia toda la configuración SMTP (host, port, user,
// password, from_email, from_name) — botón "Reconfigurar".
func (s *SystemSettingsService) ClearSMTP(ctx context.Context) error {
    for _, key := range []string{
        "smtp_host", "smtp_port", "smtp_user",
        "smtp_password", "smtp_from_email", "smtp_from_name",
    } {
        if err := s.storage.Set(ctx, key, ""); err != nil {
            return err
        }
    }
    return nil
}
```

#### 4.2.2 `internal/api/system_settings_handlers.go` (EXTENDIDO)

Nuevo handler (espejo de `DeleteVoiceMonkey`, línea 277):
```go
// DeleteSMTP limpia la configuración SMTP (botón "Reconfigurar").
func (h *SystemSettingsHandlers) DeleteSMTP(w http.ResponseWriter, r *http.Request) {
    if err := h.settings.ClearSMTP(r.Context()); err != nil {
        respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
        return
    }
    smtp, err := h.settings.GetSMTPConfigPublic(r.Context())
    if err != nil {
        respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
        return
    }
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "smtp_configured": smtp.Configured,
        "message":         "Configuración SMTP eliminada",
    })
}
```

#### 4.2.3 `internal/api/routes.go` (EXTENDIDO)

Nueva ruta con auth:
```go
mux.Handle("DELETE /api/system-settings/smtp", authMiddleware(http.HandlerFunc(handler.systemSettings.DeleteSMTP)))
```

#### 4.2.4 `frontend/src/pages/SettingsPage.tsx` (MODIFICADO)

- Nuevo estado local `smtpReconfiguring` (opcional, para deshabilitar el botón durante la llamada).
- `handleReconfigureSMTP`: confirm → `await api.systemSettings.disconnectSMTP()` → resetear `smtpHost`, `smtpPort=587`, `smtpUser`, `smtpPassword`, `smtpFromEmail`, `smtpFromName`, `smtpConfigured=false` → toast `settings.email_alerts.reconfigured`. Espejo de `handleReconfigureVoiceMonkey` (línea 212).
- Dentro del acordeón abierto: si `smtpConfigured` → fila con badge "Configurado" (verde) + botón "Reconfigurar", **sin inputs**. Si no → formulario actual (inputs + badge inline).
- Botón "Enviar email de prueba" con `disabled={sendingTest || !smtpConfigured}` (REQ-006).

#### 4.2.5 `frontend/src/api/index.ts` (EXTENDIDO)

Nuevo método en `systemSettings`:
```ts
disconnectSMTP: () => del<{ smtp_configured: boolean }>('/api/system-settings/smtp'),
```

#### 4.2.6 `frontend/public/i18n/{es,en}.json` (EXTENDIDO)

Claves nuevas bajo `settings.email_alerts.*` (espejo de `settings.voicemonkey.*`):
- `configured_status`: "Configuración SMTP configurada" / "SMTP configured"
- `reconfigure`: "Reconfigurar" / "Reconfigure"
- `reconfigure_confirm`: "¿Querés quitar la configuración actual de SMTP? Tendrás que volver a ingresar host, usuario y contraseña." / "Do you want to remove the current SMTP configuration? You will need to re-enter host, user and password."
- `reconfigured`: "Configuración SMTP eliminada" / "SMTP configuration removed"

> **Nota**: editar SOLO `frontend/public/i18n/{es,en}.json` (fuente de verdad). Después correr `npm run build` en `frontend/`.

### 4.3 Modelo de datos

Sin cambios de esquema. Las 6 claves SMTP existentes en `system_settings` se vacían con `""` (mismo mecanismo que `ClearVoiceMonkey`):

```
Tabla: system_settings (existente, key-value)

Claves vaciadas por ClearSMTP:
- smtp_host        ""
- smtp_port        ""
- smtp_user        ""
- smtp_password    ""
- smtp_from_email  ""
- smtp_from_name   ""
```

### 4.4 APIs / Contratos

#### Endpoint: `DELETE /api/system-settings/smtp` (NUEVO)

**Request**: vacío.

**Response 200**:
```json
{
  "smtp_configured": false,
  "message": "Configuración SMTP eliminada"
}
```

**Response Error (500)**:
```json
{
  "error": "internal_error",
  "message": "..."
}
```

> **Seguridad**: El endpoint no recibe ni devuelve credenciales. Solo vacía claves internas.

### 4.5 UI — Sección SMTP (diseño iOS)

```
┌─ ALERTAS POR EMAIL ────────────────────────────────────────┐
│  [▸] Configuración SMTP      (acordeón, cerrado por defecto)│
│                                                             │
│  (abierto, smtpConfigured = true)                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ (badge verde) Configuración SMTP configurada   [Reconfigurar]│
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  (abierto, smtpConfigured = false)                         │
│  │ Servidor SMTP  [input]                                   │
│  │ Puerto         [input]                                   │
│  │ Usuario        [input]                                   │
│  │ Contraseña     [input]                                   │
│  │ Email remitente [input]                                  │
│  │ Nombre remitente [input]                                 │
│  └──────────────────────────────────────────────────────────┘ │
│                                                             │
│  Destinatarios  [input]                                     │
│  [Guardar]  [Enviar email de prueba]  ← deshabilitado si !smtpConfigured
└─────────────────────────────────────────────────────────────┘
```

### 4.6 Dependencias

- **Internas**:
  - `SystemSettingsService` (extendido: `ClearSMTP`)
  - `system_settings_handlers.go` (extendido: `DeleteSMTP`), `routes.go` (extendido)
  - `SettingsPage.tsx`, `api/index.ts`, `frontend/public/i18n/{es,en}.json` (frontend)
- **Externas**:
  - **NINGUNA nueva** (solo stdlib).

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado SMTP configurado (`smtp_configured=true`), cuando el usuario abre el acordeón SMTP en Settings, entonces ve el estado "Configurado" con un botón "Reconfigurar" y **no** ve los inputs del formulario.
- [ ] CA-002: Dado SMTP no configurado, cuando el usuario abre el acordeón SMTP, entonces ve el formulario completo (comportamiento actual sin cambios).
- [ ] CA-003: Dado el botón "Reconfigurar", cuando el usuario confirma, entonces se limpian las 6 claves SMTP, `smtp_configured` pasa a `false` y se muestra el formulario nuevamente.
- [ ] CA-004: Dado el botón "Reconfigurar", cuando el usuario cancela la confirmación, entonces NO se limpia nada y el estado "Configurado" permanece.
- [ ] CA-005: Dado `DELETE /api/system-settings/smtp`, entonces `GET /api/system-settings` devuelve `smtp_configured: false` y los campos SMTP vacíos.
- [ ] CA-006: Dado que se reconfigura SMTP, entonces Voice Monkey, destinatarios, toggles de alertas y hora de facturación conservan sus valores (no se ven afectados).
- [ ] CA-007: Dado SMTP no configurado, cuando el usuario ve la sección SMTP, entonces el botón "Enviar email de prueba" está deshabilitado.
- [ ] CA-008: Dado SMTP configurado, cuando el usuario hace clic en "Reconfigurar" y completa el formulario de nuevo, entonces puede volver a configurar y "Enviar email de prueba" funciona.

### 5.2 No funcionales

- [ ] CA-NF-001: `go build ./...` compila sin errores.
- [ ] CA-NF-002: `go vet ./...` no reporta errores.
- [ ] CA-NF-003: El build de frontend (`npm run build`) compila sin errores.
- [ ] CA-NF-004: `grep -rn "smtp_password\|smtp_user" internal/ cmd/` no encuentra credenciales hardcoded ni expuestas en responses.
- [ ] CA-NF-005: `DELETE /api/system-settings/smtp` no devuelve ni loguea credenciales.

### 5.3 Testing

- **Unit tests**:
  - `system_settings_test.go`: `ClearSMTP` vacía las 6 claves; `IsSMTPConfigured` devuelve `false` después; otras claves (`voicemonkey_*`, `alert_*`, `alert_emails`, `billing_generation_hour`) intactas.
- **Integration tests**:
  - `DELETE /api/system-settings/smtp` → `GET /api/system-settings` refleja `smtp_configured: false`.
- **E2E tests (manuales)**:
  - Configurar SMTP → guardar → el acordeón muestra "Configurado" + "Reconfigurar".
  - Pulsar "Reconfigurar" → confirmar → vuelve el formulario; "Enviar email de prueba" deshabilitado.
  - Recargar página → el estado persiste (configurado/no configurado).
  - Verificar que Voice Monkey y las alertas siguen funcionando tras reconfigurar SMTP.
- **Carga/Performance**: N/A.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: `ClearSMTP` en `SystemSettingsService` + handler `DeleteSMTP` + ruta `DELETE /api/system-settings/smtp` + unit tests | 0.25 día | Ninguna |
| 2 | Frontend API: `systemSettings.disconnectSMTP` en `api/index.ts` | 0.05 día | Fase 1 |
| 3 | Frontend UI: estado "Configurado" + botón "Reconfigurar" + deshabilitar "Enviar email de prueba" cuando no configurado | 0.5 día | Fase 2 |
| 4 | i18n: claves nuevas en `frontend/public/i18n/{es,en}.json` + `npm run build` | 0.1 día | Fase 3 |
| 5 | Build + vet + tests + validación manual local | 0.5 día | Todas |

**Estimación total**: ~1.4 días

### 6.2 Milestones

1. **MVP**: Backend (endpoint de limpieza) + UI "Configurado/Reconfigurar" para SMTP + i18n.
2. **V1.0**: Tests + validación manual local completa.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El usuario pulsa "Reconfigurar" sin querer y pierde la config SMTP | Media | Bajo | Confirmación explícita (REQ-005) antes de limpiar; reconfigurar es rápido |
| Al limpiar SMTP se deshabilita el envío de emails sin que el usuario lo note | Baja | Bajo | El estado "Configurado"/"No configurado" es visible; el test email queda deshabilitado como señal |
| Romper credenciales de Voice Monkey u otros settings al limpiar SMTP | Baja | Alto | `ClearSMTP` solo toca las 6 claves SMTP (test CA-006 cubre aislamiento) |
| Editar i18n en `public/i18n/` (fuente equivocada) y perder claves en el build | Media | Medio | Regla AGENTS.md: editar SOLO `frontend/public/i18n/`; correr `npm run build` y verificar con `curl` |

---

## 8. Notas y Referencias

- SPEC-033 (REQ-020 / CA-016): patrón "Configurado + Reconfigurar" para Voice Monkey — referencia directa de esta spec.
- SPEC-029: seguridad de credenciales SMTP (base).
- `internal/services/system_settings.go:286` (`ClearVoiceMonkey`) — modelo para `ClearSMTP`.
- `internal/api/system_settings_handlers.go:277` (`DeleteVoiceMonkey`) — modelo para `DeleteSMTP`.
- `internal/api/routes.go:34` (`DELETE /api/system-settings/voicemonkey`) — modelo para la nueva ruta.
- `frontend/src/pages/SettingsPage.tsx:212` (`handleReconfigureVoiceMonkey`) — modelo para `handleReconfigureSMTP`.
- `frontend/src/api/index.ts:100` (`disconnectVoiceMonkey`) — modelo para `disconnectSMTP`.
- `frontend/public/i18n/es.json` (`settings.voicemonkey.reconfigure*`) — claves i18n a replicar.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación. Estado "Configurado" + botón "Reconfigurar" para SMTP, espejo del patrón de Voice Monkey (SPEC-033). Backend: `ClearSMTP` + `DELETE /api/system-settings/smtp`. Frontend: estado sin inputs + confirmación + i18n. |
| 2026-08-16 | paulomcnally | Estado a `pending_execution`. Lista para desarrollo. |
| 2026-08-16 | paulomcnally | Desarrollo completado y validado en local por el usuario. Criterios de aceptación pasan. Estado a `pending_release`. |
| 2026-08-16 | paulomcnally | Released. Commit `b25c618` ("SPEC-034: estado Configurado y botón Reconfigurar para SMTP") mergeado a `main` (fast-forward desde `feature/SPEC-034`). Issue #34 cerrado. |
