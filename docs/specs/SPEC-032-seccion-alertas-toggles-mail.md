---
title: "Sección Alertas en Configuraciones con toggles por funcionalidad de mail"
id: "SPEC-032"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-09-04 (released)"
github_issue: 32
---

# Sección Alertas en Configuraciones con toggles por funcionalidad de mail

**ID**: SPEC-032  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-09-04 (released)

---

## 1. Resumen Ejecutivo

El sistema p40la-ihost tiene actualmente **tres funcionalidades que envían emails** automáticamente: SPEC-029 (alertas de seguros de autos vencidos), SPEC-030 (email informativo al generar factura automática) y SPEC-031 (resumen diario de facturas pendientes). Hoy todos los schedulers envían sus emails siempre que SMTP y destinatarios estén configurados, **sin que el usuario pueda desactivar** una funcionalidad puntual ni todas a la vez. Esto genera emails no deseados (por ejemplo, el resumen diario puede ser ruido si el usuario prefiere revisar la app manualmente).

Esta spec agrega una sección **"Alertas" en Configuraciones** donde el usuario puede **encender o apagar cada funcionalidad de mail** mediante un toggle individual. Cada funcionalidad tiene un **título y una descripción** clara. **Por defecto todas las alertas están APAGADAS** (opt-in): solo se envían emails de una funcionalidad si su toggle está encendido. Además, la **configuración SMTP** (hoy mostrada inline en la sección de email) pasa a ser un **acordeón colapsable que arranca cerrado por defecto**, para mantener la página limpia.

**Consideraciones de iHost**: Se reutiliza la tabla `system_settings` (SQLite key-value) con claves booleanas nuevas (`1`/`0`). No se crean tablas nuevas. Los toggles se leen desde cada scheduler antes de enviar el email — cambio mínimo (una condición por scheduler). **Cero dependencias nuevas.** Consumo de RAM/CPU despreciable.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Debe existir una sección **"Alertas"** en la página de Configuraciones (Settings) que liste **cada funcionalidad de mail** con su **título**, **descripción** y un **toggle** de encendido/apagado.
2. **REQ-002**: Cada funcionalidad de mail existente debe tener su propio toggle:
   - **REQ-002a**: Alertas de seguros de autos vencidos (SPEC-029 / `AlertScheduler`).
   - **REQ-002b**: Email informativo al generar factura automática (SPEC-030 / `BillingScheduler`).
   - **REQ-002c**: Resumen diario de facturas pendientes (SPEC-031 / `BillSummaryScheduler`).
3. **REQ-003**: **Por defecto todas las alertas están APAGADAS** (opt-in). En una instalación nueva o sin configurar, los toggles están en off y **no se envía ningún email automático** hasta que el usuario los active.
4. **REQ-004**: Cada scheduler DEBE verificar el estado de su toggle antes de enviar emails. Si está apagado, **no envía** ningún email.
5. **REQ-005**: La configuración SMTP (host, port, user, password, from_email, from_name) debe mostrarse en un **acordeón colapsable** dentro de la sección de email, **cerrado por defecto**.
6. **REQ-006**: La lista de destinatarios (`alert_emails`) debe seguir siendo configurable y visible en la misma sección (no requiere toggle).
7. **REQ-007**: El botón **"Enviar email de prueba"** debe seguir existiendo y funcionando independientemente del estado de los toggles (permite validar SMTP siempre).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: Los toggles deben persistirse en `system_settings` con claves booleanas (`1`/`0`) y sobrevivir a reinicios.
2. **REQ-009**: La API `GET /api/system-settings` debe exponer el estado de cada toggle (booleano) y la UI debe reflejarlo al cargar.
3. **REQ-010**: La API `PUT /api/system-settings` debe aceptar el estado de cada toggle y persistirlo.
4. **REQ-011**: Cambiar un toggle desde la UI debe guardarse inmediatamente (sin botón guardar separado) o al guardar la sección, según el patrón existente de Settings.
5. **REQ-012**: Si el toggle de una funcionalidad está apagado, su scheduler debe loguear que está deshabilitada (nivel debug/info) y retornar sin enviar.
6. **REQ-013**: La funcionalidad "Enviar email de prueba" no debe verse afectada por los toggles (es una validación manual de SMTP).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Toggle maestro "Todas las alertas" que active/desactive todas a la vez (con confirmación si se apaga todo).
2. **REQ-015**: Mostrar un aviso en Settings si SMTP no está configurado pero hay toggles encendidos (para guiar al usuario).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Leer el toggle es una consulta extra a `system_settings` por ciclo del scheduler (una por día o por email). Impacto despreciable.
- **Seguridad — Protección de información sensible**: Se mantienen las reglas de SPEC-029: password SMTP nunca se expone en API ni logs. Los toggles no son información sensible.
- **Almacenamiento**: Solo 3 claves nuevas en `system_settings` (o 4 con el toggle maestro). Sin tablas nuevas.
- **Disponibilidad**: Si una clave de toggle no existe, se interpreta como `false` (apagado) — comportamiento seguro por defecto.
- **iHost**: Cero dependencias nuevas. Solo lectura/escritura en `system_settings`.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-029** implementó `AlertScheduler` (seguros de autos) y `EmailService` con `system_settings` para SMTP y `alert_emails`.
- **SPEC-030** integró el envío de email en `BillingScheduler` (`sendBillCreatedEmail`) tras generar cada factura automática.
- **SPEC-031** implementó `BillSummaryScheduler` (resumen diario de facturas pendientes).
- **`SystemSettingsService`** (`internal/services/system_settings.go`) ya provee `GetSetting`, `Set`, `GetAlertEmails`, `GetAlertCheckHour`, etc. — patrón reutilizable para toggles booleanos.
- **`SystemSettingsStorage`** (`internal/storage/system_settings.go`) provee `Get`/`Set`/`GetSetting` (UPSERT).
- **Handlers** (`internal/api/system_settings_handlers.go`): `GetSystemSettings` y `UpdateSystemSettings` con `settingsRequest` (patrón de campos opcionales con punteros).
- **Frontend** (`frontend/src/pages/SettingsPage.tsx`): sección "Alertas por Email" con SMTP inline; usa `api.systemSettings.get/update`. El i18n está en `public/i18n/{es,en}.json` bajo `settings.email_alerts`.
- **Cliente API** (`frontend/src/api/index.ts`): `systemSettings.get/update/testEmail`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A: Toggles booleanos en `system_settings` leídos por cada scheduler** | Mínimo cambio, sin migraciones, consistente con el patrón existente | Cada scheduler agrega una consulta extra | ✅ Seleccionada |
| **B: Campo `enabled` en cada entidad (auto/servicio)** | Granularidad por entidad | Mucho más complejo, sobre-ingeniería para uso personal | ❌ Rechazada |
| **C: Flag global único "mails activados"** | Muy simple | No permite desactivar una funcionalidad puntual (requerimiento explícito) | ❌ Rechazada |
| **D: Toggle solo en frontend (sin backend)** | Solo UI | Los schedulers seguirían enviando; no cumple el requerimiento | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Claves booleanas en `system_settings` con valores `1`/`0`**
- **Contexto**: Se necesita persistir el estado de cada toggle sin migraciones ni tablas nuevas.
- **Decisión**: Usar `system_settings` con claves:
  - `alert_insurance_enabled` (SPEC-029)
  - `alert_bill_created_enabled` (SPEC-030)
  - `alert_bill_summary_enabled` (SPEC-031)
- **Consecuencias**:
  - ✅ Sin migraciones, cero RAM extra
  - ✅ `GetSetting` devuelve `nil` si no existe → interpretar como `false` (apagado por defecto)
  - ⚠️ Valores son strings en SQLite; parsear `"1"` como `true`

**ADR-002: Cada scheduler lee su toggle antes de enviar**
- **Contexto**: Los tres schedulers ya tienen acceso a `SystemSettingsService`.
- **Decisión**: Cada scheduler consulta su clave antes de enviar. Si `false`, loguea (debug) y retorna.
- **Consecuencias**:
  - ✅ Mínimo cambio por scheduler (una condición)
  - ✅ Sin tocar la lógica de generación de facturas ni de detección de seguros

**ADR-003: Apagado no bloquea la generación de facturas ni la deduplicación diaria**
- **Contexto**: Los toggles solo controlan el ENVÍO de emails, no la lógica de negocio.
- **Decisión**: `BillingScheduler` sigue generando facturas aunque el toggle de email esté apagado (solo salta el envío). Para los schedulers diarios, si el toggle está apagado se retorna antes del check de hora/deduplicación para no marcar nada (o se evalúa igual — decisión: retornar antes, simple y sin efectos).
- **Consecuencias**:
  - ✅ No se pierden facturas ni se rompe la deduplicación
  - ⚠️ Si el toggle está apagado, los schedulers diarios no registran su check (irrelevante porque no envían)

**ADR-004: SMTP como acordeón colapsable cerrado por defecto**
- **Contexto**: El usuario pidió que SMTP sea un acordeón colapsado por defecto para limpiar la página.
- **Decisión**: En `SettingsPage`, envolver los campos SMTP en un componente `Accordion` (o `<details>`/estado local) con `defaultOpen: false`. Título "Configuración SMTP", subtítulo "Servidor de correo saliente".
- **Consecuencias**:
  - ✅ UI más limpia
  - ✅ Destinatarios y toggles quedan visibles sin scroll
  - ⚠️ Requiere estado local en React (componente acordeón simple)

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────────────────────────────────────┐
│                   SettingsPage.tsx (frontend)                │
│  Sección "Alertas"                                          │
│  ├─ Toggle: Seguros de autos vencidos (title+desc)          │
│  ├─ Toggle: Email al generar factura (title+desc)           │
│  ├─ Toggle: Resumen diario de facturas (title+desc)         │
│  └─ Sección Email:                                          │
│     ├─ Destinatarios (visible)                              │
│     ├─ [Acordeón cerrado] Configuración SMTP                │
│     └─ Botón "Enviar email de prueba"                       │
└───────────────┬─────────────────────────────────────────────┘
                │ GET/PUT /api/system-settings
                ▼
┌─────────────────────────────────────────────────────────────┐
│  system_settings_handlers.go (EXTENDIDO)                    │
│  settingsRequest += alert_*_enabled (*bool)                 │
│  GetSystemSettings += alert_*_enabled (bool)                │
└───────────────┬─────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│  SystemSettingsService (EXTENDIDO)                          │
│  IsAlertEnabled(ctx, key) / SetAlertEnabled(ctx, key, bool) │
└───────────────┬─────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│  Schedulers (cada uno consulta su clave antes de enviar)    │
│  ├─ AlertScheduler        → alert_insurance_enabled         │
│  ├─ BillingScheduler      → alert_bill_created_enabled      │
│  └─ BillSummaryScheduler  → alert_bill_summary_enabled      │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 `internal/services/system_settings.go` (EXTENDIDO)
- **Responsabilidad**: Getters/setters booleanos para toggles de alertas.
- **Nuevos métodos**:
  ```go
  // IsAlertEnabled devuelve true si la clave existe y vale "1".
  func (s *SystemSettingsService) IsAlertEnabled(ctx context.Context, key string) (bool, error)
  // SetAlertEnabled guarda "1"/"0".
  func (s *SystemSettingsService) SetAlertEnabled(ctx context.Context, key string, enabled bool) error
  ```
- **Constantes**:
  ```go
  const (
      AlertInsuranceEnabledKey   = "alert_insurance_enabled"
      AlertBillCreatedEnabledKey = "alert_bill_created_enabled"
      AlertBillSummaryEnabledKey = "alert_bill_summary_enabled"
  )
  ```

#### 4.2.2 `internal/services/alert_scheduler.go` (MODIFICADO)
- En `checkAndAlert()`, antes de recolectar alertas, consultar `IsAlertEnabled(ctx, AlertInsuranceEnabledKey)`. Si `false`, loguear debug y retornar.

#### 4.2.3 `internal/services/billing_scheduler.go` (MODIFICADO)
- En `sendBillCreatedEmail`, primera condición: `IsAlertEnabled(ctx, AlertBillCreatedEnabledKey)`. Si `false`, loguear debug y retornar (sin enviar). La factura ya se generó.

#### 4.2.4 `internal/services/bill_summary_scheduler.go` (MODIFICADO)
- En `checkAndSend()`, antes de consultar facturas, verificar `IsAlertEnabled(ctx, AlertBillSummaryEnabledKey)`. Si `false`, loguear debug y retornar.

#### 4.2.5 `internal/api/system_settings_handlers.go` (EXTENDIDO)
- `settingsRequest` += `AlertInsuranceEnabled *bool`, `AlertBillCreatedEnabled *bool`, `AlertBillSummaryEnabled *bool`.
- `GetSystemSettings` += `"alert_insurance_enabled"`, `"alert_bill_created_enabled"`, `"alert_bill_summary_enabled"` (bool).
- `UpdateSystemSettings` += persistir cada uno si viene en el request.

#### 4.2.6 `frontend/src/pages/SettingsPage.tsx` (MODIFICADO)
- Nueva sección "Alertas" antes de la sección de email.
- 3 filas con título, descripción y toggle (estilo iOS: `<label>` + switch).
- Estado local `alertInsurance`, `alertBillCreated`, `alertBillSummary` cargado desde `GET`.
- Persistencia: al cambiar un toggle, `PUT /api/system-settings` con esa clave.
- SMTP envuelto en acordeón colapsable (cerrado por defecto).

#### 4.2.7 `frontend/src/api/index.ts` (EXTENDIDO)
- Actualizar tipos de `systemSettings.get()` y `update()` con las claves nuevas.

#### 4.2.8 `public/i18n/{es,en}.json` (EXTENDIDO)
- Claves nuevas bajo `settings.alerts.*`: títulos y descripciones de cada alerta, y `settings.smtp_accordeon` si aplica.

### 4.3 Modelo de datos

Sin cambios de esquema. Claves nuevas en `system_settings`:

```
Tabla: system_settings (existente, key-value)

Claves nuevas (valores "1"/"0", ausente = apagado):
- alert_insurance_enabled     -- SPEC-029: alertas de seguros de autos
- alert_bill_created_enabled  -- SPEC-030: email al generar factura automática
- alert_bill_summary_enabled  -- SPEC-031: resumen diario de facturas pendientes
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/system-settings` (EXTENDIDO)

**Response 200** (campos nuevos marcados con ★):
```json
{
  "billing_generation_hour": 8,
  "smtp_host": "smtp.example.com",
  "smtp_port": 587,
  "smtp_user": "",
  "smtp_from_email": "alerts@example.com",
  "smtp_from_name": "P40LA",
  "smtp_configured": true,
  "alert_emails": "paulo@example.com",
  "alert_insurance_enabled": false,        // ★
  "alert_bill_created_enabled": false,     // ★
  "alert_bill_summary_enabled": false      // ★
}
```

#### Endpoint: `PUT /api/system-settings` (EXTENDIDO)

**Request** (campos opcionales):
```json
{
  "alert_insurance_enabled": true,
  "alert_bill_created_enabled": false,
  "alert_bill_summary_enabled": true
}
```

**Response 200**:
```json
{
  "billing_generation_hour": 8,
  "smtp_configured": true,
  "message": "Configuración actualizada"
}
```

### 4.5 UI — Sección Alertas

Diseño tipo iOS (cards), consistente con el resto de Settings:

```
┌─ ALERTAS ──────────────────────────────────────────────┐
│  ┌────────────────────────────────────────────────────┐ │
│  │ Seguros de autos vencidos               [toggle]  │ │
│  │ Recibir un email cuando un seguro de auto       │ │
│  │ vence o un auto queda sin cobertura.             │ │
│  └────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────┐ │
│  │ Nueva factura generada                   [toggle]  │ │
│  │ Email informativo cuando el sistema genera      │ │
│  │ una factura automáticamente.                     │ │
│  └────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────┐ │
│  │ Resumen diario de facturas              [toggle]  │ │
│  │ Resumen diario de todas las facturas            │ │
│  │ pendientes por email.                             │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

El toggle usa el estilo de switch iOS existente en el proyecto (si existe componente `Toggle`/`Switch`, reutilizarlo; si no, crear uno simple siguiendo el patrón de cards).

### 4.6 Dependencias

- **Internas**:
  - `SystemSettingsService` / `SystemSettingsStorage` (extendido)
  - `AlertScheduler`, `BillingScheduler`, `BillSummaryScheduler` (modificados)
  - `system_settings_handlers.go` (extendido)
  - `SettingsPage.tsx`, `api/index.ts`, `public/i18n/*.json` (frontend)
- **Externas**:
  - **NINGUNA nueva**.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un usuario que nunca configuró alertas, cuando abre Settings, entonces los tres toggles están en OFF y no se envía ningún email automático.
- [x] CA-002: Dado el toggle "Seguros de autos vencidos" en OFF, cuando el `AlertScheduler` ejecuta, entonces no envía email.
- [x] CA-003: Dado el toggle "Seguros de autos vencidos" en ON, cuando el `AlertScheduler` detecta alertas, entonces envía email (igual que SPEC-029).
- [x] CA-004: Dado el toggle "Nueva factura generada" en OFF, cuando el `BillingScheduler` genera una factura, entonces la factura se crea pero NO se envía email.
- [x] CA-005: Dado el toggle "Nueva factura generada" en ON, cuando el `BillingScheduler` genera una factura, entonces se envía email informativo (igual que SPEC-030).
- [x] CA-006: Dado el toggle "Resumen diario de facturas" en OFF, cuando el `BillSummaryScheduler` ejecuta, entonces no envía email.
- [x] CA-007: Dado el toggle "Resumen diario de facturas" en ON, cuando el `BillSummaryScheduler` ejecuta y hay pendientes, entonces envía email (igual que SPEC-031).
- [x] CA-008: Dado el usuario que cambia un toggle en la UI, cuando guarda (o inmediatamente), entonces el estado persiste en `system_settings` y sobrevive a un reinicio.
- [x] CA-009: Dado `GET /api/system-settings`, entonces la respuesta incluye `alert_insurance_enabled`, `alert_bill_created_enabled` y `alert_bill_summary_enabled` como booleanos.
- [x] CA-010: Dado el botón "Enviar email de prueba", cuando SMTP está configurado, entonces envía email de prueba incluso con todos los toggles apagados.
- [x] CA-011: Dado la sección SMTP en Settings, entonces se muestra como acordeón colapsable **cerrado por defecto**.
- [x] CA-012: Dado un toggle apagado, cuando el scheduler correspondiente ejecuta, entonces loguea (debug/info) que la alerta está deshabilitada.

### 5.2 No funcionales

- [x] CA-NF-001: El binario compilado no aumenta más de 50KB (sin dependencias nuevas).
- [x] CA-NF-002: `go build ./...` compila sin errores.
- [x] CA-NF-003: `go vet ./...` no reporta errores.
- [x] CA-NF-004: `grep -rn "smtp_password\|smtp_user\|postmaster" internal/ cmd/` no encuentra credenciales hardcoded.
- [x] CA-NF-005: El build de frontend (`npm run build`) compila sin errores.

### 5.3 Testing

- **Unit tests**:
  - `system_settings_test.go`: `IsAlertEnabled` con clave ausente (false), `"0"` (false), `"1"` (true); `SetAlertEnabled` persiste y re-lee.
  - `billing_scheduler_test.go`: toggle OFF → `sendBillCreatedEmail` no llama a `Send`; toggle ON → llama.
  - `bill_summary_scheduler_test.go`: toggle OFF → no consulta facturas ni envía; toggle ON → comportamiento normal.
  - `alert_scheduler_test.go`: toggle OFF → no envía.
- **Integration tests**:
  - `PUT /api/system-settings` con toggles → `GET` los devuelve.
  - Toggle OFF → correr scheduler → no hay envío (mock).
- **E2E tests (manuales)**:
  - Abrir Settings → toggles visibles con título y descripción → activar/desactivar → recargar página → estado persistido.
  - Activar toggle y verificar recepción del email correspondiente.
  - Desactivar toggle y verificar que NO llega email.
  - SMTP como acordeón cerrado; abrirlo, configurar, guardar, enviar test.
- **Carga/Performance**: N/A.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `SystemSettingsService`: `IsAlertEnabled`/`SetAlertEnabled` + constantes + unit tests | 0.25 día | Ninguna |
| 2 | Modificar los 3 schedulers para consultar su toggle antes de enviar + tests | 0.5 día | Fase 1 |
| 3 | Extender handlers `GET/PUT /api/system-settings` con los 3 toggles | 0.25 día | Fase 1 |
| 4 | Frontend: sección Alertas con toggles (título + descripción), persistencia, API types | 0.75 día | Fase 3 |
| 5 | Frontend: SMTP como acordeón colapsable cerrado por defecto + i18n | 0.25 día | Fase 4 |
| 6 | Build + vet + tests + validación manual local | 0.5 día | Todas |

**Estimación total**: ~2.5 días

### 6.2 Milestones

1. **MVP**: Backend (toggles + schedulers + API) — las alertas se pueden prender/apagar por API/curl.
2. **V1.0**: UI completa (sección Alertas + acordeón SMTP + i18n) + tests.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El usuario olvida activar un toggle y no recibe emails | Media | Bajo | Diseño opt-in explícito (REQ-003). Los títulos/descripciones son claros. P2 (REQ-015): aviso si SMTP configurado pero toggles off |
| Toggle apagado impide generación de facturas por error | Baja | Medio | ADR-003: el toggle solo controla el envío, jamás la generación (facturas se crean igual) |
| Confusión entre "alertas" y "SMTP" en la UI | Media | Bajo | Secciones separadas y claras; SMTP en acordeón cerrado |
| Clave de toggle borrada en DB (ausente) | Baja | Bajo | Ausente = apagado (comportamiento seguro por defecto) |
| Toggle maestro (P2) desactiva todo sin querer | Baja | Bajo | Confirmación antes de apagar todo |

---

## 8. Notas y Referencias

- SPEC-029: `AlertScheduler`, `EmailService`, `system_settings` (base).
- SPEC-030: email por factura automática en `BillingScheduler`.
- SPEC-031: `BillSummaryScheduler` y resumen diario.
- `internal/services/system_settings.go` (patrón de settings).
- `internal/api/system_settings_handlers.go` (GET/PUT).
- `frontend/src/pages/SettingsPage.tsx` (UI actual).
- `frontend/src/api/index.ts` (cliente API).
- `public/i18n/{es,en}.json` (traducciones).

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación. Toggles opt-in por funcionalidad de mail (seguros, factura automática, resumen diario), apagados por defecto. SMTP como acordeón colapsado. |
| 2026-08-16 | paulomcnally | Estado a `in_progress`. Desarrollo iniciado en rama `feature/SPEC-032-033-alertas-multicanal`. Implementación integrada con SPEC-033 (tabla `alerts` como almacenamiento unificado de `mail_enabled`/`voice_enabled` — ADR-005 de SPEC-033). |
| 2026-08-16 | paulomcnally | Desarrollo completado y validado en local por el usuario. Criterios de aceptación pasan. Estado a `pending_release`. |
| 2026-09-04 | paulomcnally | Release. Implementada en `main` (commit `824ca16`). Issue #32 ya cerrado en GitHub con label `spec/released`; sync del archivo local y tracker. |