---
title: "Emails y Generación Mensual de Registros de Pensión Alimenticia"
id: "SPEC-051"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-03"
github_issue: 51
---

# Emails y Generación Mensual de Registros de Pensión Alimenticia

**ID**: SPEC-051  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-03

---

## 1. Resumen Ejecutivo

Esta spec es la **tercera y última** del grupo que replica la página `child-support/records` de P4OLA en p40la-ihost (SPEC-049 backend → SPEC-050 frontend → SPEC-051 emails + generación). Cubre dos capacidades que P4OLA tiene integradas a la página de registros:

1. **Generación mensual**: P4OLA crea automáticamente los `salary_payments` (uno por salario activo del mes) y los `support_records` (por cada config activa hijo/categoría con `autoGenerate`, solo si hay al menos un pago de salario en el mes). Se acordó con el usuario implementar la **generación por botón** (endpoint `POST /api/pension/generate`) en vez del cron diario automático; el cron queda documentado como trabajo futuro.
2. **Notificaciones por email**: al generar registros, al cambiar estado (pagado/recibido), al rechazar un registro y al cerrar el mes. Se usa el **`EmailService` existente** del proyecto (plantilla HTML única, destinatarios desde la tabla `notifications` y toggles de alerta vía `AlertService`, patrón de `bill_email.go`/`billing_scheduler.go`). En P4OLA los destinatarios son `notification_destinations` con flag `isPrimary`; en p40la-ihost la tabla `notifications` ya cumple ese rol (name, email, active).

Para la generación se necesita la tabla **`child_support_configs`** (monto/moneda/activo/auto-generación por par hijo-categoría), que no existe aún. P4OLA la usa como fuente de montos automáticos. Se replica con el mismo modelo, adaptado a la convención de p40la-ihost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Migración SQLite `0023_create_child_support_configs` (up/down): id, child_id (FK→children.id), pension_category_id (FK→pension_categories.id), amount (REAL), currency (TEXT 3, default NIO), is_active (INTEGER default 1), auto_generate (INTEGER default 0), created_at, updated_at. **UNIQUE (child_id, pension_category_id)**.
2. **REQ-002**: Modelo Go `ChildSupportConfig` + storage `child_support_config.go` + service `child_support_config.go` + handlers CRUD (protegidos):
   - `GET/POST /api/pension/configs`
   - `PUT/DELETE /api/pension/configs/{id}`
3. **REQ-003**: Endpoint `POST /api/pension/generate?year&month` que:
   - Crea `salary_payments` pendientes para cada salario **activo** del mes (si no existe para salary+year+month).
   - Crea `support_records` pendientes para cada config activa con `auto_generate=1` (si existe al menos un salary_payment en el mes y no existe ya el registro child+category+year+month).
   - Devuelve `{ ok: true, created_salary_payments: N, created_support_records: N }`.
4. **REQ-004**: Notificación por email **"registros creados"** tras `POST /api/pension/generate` cuando crea al menos un registro o salario (resumen con hijos/categorías/montos y salarios/empleadores). Usa `EmailService.Send` + destinatarios de `notifications` (solo `active=1`) + toggle de alerta `pension_records_created` si el canal mail está habilitado.
5. **REQ-005**: Notificación por email **"estado cambiado"** al `mark-paid` de un support_record y al `mark-received` de un salary_payment (detalle: qué se pagó/recibió, monto, fecha, diferencia si aplica). Toggle `pension_record_paid` / `pension_salary_received`.
6. **REQ-006**: Notificación por email **"registro rechazado"** al `mark-rejected` (motivo incluido). Toggle `pension_record_rejected`.
7. **REQ-007**: Notificación por email **"cierre de mes"** al `POST /api/pension/closing/:year/:month` (resumen del mes: totales, pagados, pendientes, rechazados). Toggle `pension_month_closing`.
8. **REQ-008**: Nuevas entradas en el catálogo de alertas (`AlertService.Seed`) para los toggles de email de pensión, visibles en la sección Alertas de Settings (patrón SPEC-032).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Los handlers de marca de estado (`support_record_handlers.go`, `salary_payment_handlers.go`) disparan las notificaciones **después** de persistir y de forma **no bloqueante** (si el envío falla, se loguea y no rompe la respuesta HTTP; patrón `billing_scheduler.sendBillCreatedEmail`).
2. **REQ-010**: Respuesta de `POST /api/pension/generate` sin crear duplicados (idempotente por UNIQUE; segunda llamada devuelve 0 creados).
3. **REQ-011**: Los destinatarios se obtienen de `notifications` (active=1); si no hay destinatarios, SMTP no configurado o toggle apagado → no envía y loguea (debug/warn), nunca error.
4. **REQ-012**: Título y contenido de cada email en la plantilla HTML única de `EmailService.RenderTemplate` (mismo look & feel que los emails de facturas).
5. **REQ-013**: Botón "Generar mes" en la página de registros (SPEC-050 REQ-019) conectado a `POST /api/pension/generate`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Cron diario automático de generación (réplica del `ChildSupportCronService` de P4OLA) — documentado como futuro; NO se implementa en esta spec (acordado con el usuario).
2. **REQ-015**: `notification_logs` para historial de emails de pensión (P4OLA lo tiene; p40la-ihost no). Solo si es necesario para diagnóstico.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: La generación de un mes es O(salarios + configs), queries simples; sin bloqueo de envío de emails en la request (best-effort).
- **Seguridad**: Rutas protegidas por auth; destinatarios solo de la tabla `notifications`.
- **Almacenamiento**: 1 tabla nueva (`child_support_configs`) + claves de alerta nuevas.
- **Disponibilidad**: Sin cron nuevo en esta spec; el endpoint de generación es manual.
- **iHost**: Reuso total del `EmailService` existente (net/smtp stdlib); sin dependencias nuevas; sin colas (a diferencia de BullMQ en P4OLA, el envío es síncrono best-effort).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **P4OLA cron**: `child-support-cron.service.ts` genera salary_payments de salarios activos (cuando `currentDay >= payDayOfMonth`) y support_records desde configs activas con auto-generación (solo si hay salary_payments del mes). El usuario eligió **generación por botón** (sin cron).
- **P4OLA emails**: `child-support-notification.service.ts` + `email-queue.service.ts` (BullMQ) + templates handlebars. p40la-ihost no tiene colas; el patrón existente es envío síncrono best-effort en `billing_scheduler.go` y `bill_email.go` (SPEC-029/030/031/032/033).
- **Alertas existentes**: `internal/services/alert_service.go` con `AlertService.Seed` (catálogo) y toggles por canal. Los emails de facturas usan `alertMailEnabled(ctx, alerts, AlertKeyBillCreated)` + `GetAlertEmails(ctx)` (destinatarios de settings). Para pensión, los destinatarios serán la tabla `notifications` (ya existe, SPEC-046).
- **Tabla configs**: P4OLA `child-support-config.entity.ts` (childId, supportCategoryId, amount, currency, isActive, autoGenerate). Se replica como `child_support_configs`.
- **Currencies**: `salaries.currency_id` es FK a `currencies`. Para salary_payments y support_records se persiste el **código** de moneda (TEXT 3) como en P4OLA, resolviéndolo desde `currencies.code` en la generación.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Envío síncrono best-effort (EmailService existente) | Cero dependencias, patrón ya usado | Un fallo SMTP lento puede demorar la request | ✅ Seleccionada |
| Cola de emails (BullMQ/Redis) | Retry robusto | Dependencia externa pesada en iHost | ❌ Rechazada |
| Cron automático diario | 100% automático | Complejidad + riesgo en iHost | ❌ Rechazada (se deja para futuro, REQ-014) |
| Destinatarios desde `notifications` | Tabla ya existe | Sin `isPrimary` (P4OLA lo tiene) | ✅ Seleccionada (todos los activos reciben) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Envío de emails síncrono best-effort
- **Contexto**: P4OLA usa colas BullMQ; p40la-ihost prioriza simplicidad en iHost.
- **Decisión**: Llamar a `EmailService.Send` directamente desde los handlers, después de persistir, en un goroutine o de forma no bloqueante; errores solo se loguean.
- **Consecuencias**: Sin infraestructura de colas; el usuario ya ve la respuesta de la API sin esperar el envío.

**ADR-002**: Generación por botón, no cron
- **Contexto**: El usuario pidió "creación manual + generación por botón"; el cron de P4OLA corre diario.
- **Decisión**: Endpoint `POST /api/pension/generate?year&month` idempotente; el botón en la página lo invoca. El cron diario queda como REQ-014 (futuro).
- **Consecuencias**: Menos riesgo operativo; el usuario controla cuándo generar.

**ADR-003**: Montos automáticos desde `child_support_configs`
- **Contexto**: P4OLA define por config el monto/moneda de cada categoría por hijo.
- **Decisión**: Tabla `child_support_configs` (UNIQUE child+category) con `amount`, `currency`, `is_active`, `auto_generate`. La generación usa `is_active=1` y `auto_generate=1`.
- **Consecuencias**: Necesita una UI de configs (mínima, P1) o seed manual; el CRUD se expone por API y la UI se deja como mejora.

**ADR-004**: Toggles de alerta nuevos en el catálogo existente
- **Contexto**: El sistema ya tiene catálogo de alertas con canal mail/voz (SPEC-032/033).
- **Decisión**: Agregar claves `pension_records_created`, `pension_record_paid`, `pension_salary_received`, `pension_record_rejected`, `pension_month_closing` al `Seed` del `AlertService`, con speech y toggle mail.
- **Consecuencias**: El usuario gestiona qué emails de pensión recibe desde Settings → Alertas, sin cambios de infraestructura.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[POST /api/pension/generate] --> [PensionGenerationService]
   |--> crea salary_payments (salarios activos)
   |--> crea support_records (configs activas auto_generate + hay salario)
   '--> [EmailService.Send] "registros creados" (notifications activos + toggle)

[mark-paid / mark-received / mark-rejected / closing] --> [EmailService.Send] "estado cambiado" / "rechazado" / "cierre de mes"
```

### 4.2 Componentes

#### 4.2.1 Migración + modelo config
- `migrations/0023_create_child_support_configs.*`, `internal/models/child_support_config.go`.

#### 4.2.2 Storage/Services/Handlers
- `internal/storage/child_support_config.go`, `internal/services/child_support_config.go`, `internal/api/child_support_config_handlers.go`.
- `internal/services/pension_generation.go` (lógica de generación mensual).
- `internal/services/pension_email.go` (build de emails de pensión, patrón `bill_email.go`).

#### 4.2.3 Alertas
- `internal/services/alert_service.go` (nuevas claves en `Seed`), `internal/models/alert.go` (constantes de claves).

#### 4.2.4 Wiring
- `cmd/server/main.go`: instanciar config service, generation service, inyectar emailService + alertService + notificationStorage en los handlers de records/salary/closing.

### 4.3 Modelo de datos

```
Entidad: child_support_config
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- child_id: INTEGER NOT NULL (FK → children.id, ON DELETE CASCADE)
- pension_category_id: INTEGER NOT NULL (FK → pension_categories.id, ON DELETE CASCADE)
- amount: REAL NOT NULL
- currency: TEXT(3) NOT NULL DEFAULT 'NIO'
- is_active: INTEGER NOT NULL DEFAULT 1
- auto_generate: INTEGER NOT NULL DEFAULT 0
- created_at / updated_at
- UNIQUE (child_id, pension_category_id)
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/pension/generate?year=2026&month=8`
**Response 200**:
```json
{ "ok": true, "created_salary_payments": 1, "created_support_records": 3 }
```
Sin duplicados: segunda llamada devuelve `0,0`.

#### Endpoint: `POST /api/pension/configs`
**Request**:
```json
{ "child_id": 2, "pension_category_id": 3, "amount": 1500.00, "currency": "NIO", "is_active": true, "auto_generate": true }
```
**Response 201**: config creada | **409**: par child+category ya existe.

#### Emails (formato plantilla única de `EmailService`)

| Tipo | Toggle de alerta | Gatillo | Contenido |
|------|-------------------|---------|-----------|
| registros creados | `pension_records_created` | `POST /api/pension/generate` con >0 creados | Salarios generados (empleador, monto, moneda) + registros (hijo, categoría, monto, moneda) + período |
| pago de registro | `pension_record_paid` | `mark-paid` | Hijo — categoría, monto, fecha de pago, método, referencia |
| salario recibido | `pension_salary_received` | `mark-received` | Empleador, monto, monto recibido, diferencia si aplica, fecha |
| registro rechazado | `pension_record_rejected` | `mark-rejected` | Hijo — categoría, monto, motivo |
| cierre de mes | `pension_month_closing` | `POST closing/:year/:month` | Resumen del mes (totales, pagado, pendiente, rechazado, montos) |

### 4.5 Dependencias

- **Internas**: `EmailService`, `AlertService` (`IsEnabled` + `Seed`), `SystemSettingsService` (SMTP), `NotificationStorage` (destinatarios), storage de records/salary/closing (SPEC-049), storage de salaries/pension_categories/children.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `POST /api/pension/generate` crea salary_payments para salarios activos y support_records para configs activas auto_generate, respetando la condición de "al menos un salario en el mes".
- [ ] CA-002: `POST /api/pension/generate` es idempotente (segunda llamada no duplica).
- [ ] CA-003: Se envía email "registros creados" solo si la alerta correspondiente tiene mail habilitado, hay destinatarios activos y SMTP configurado; sin esas condiciones no envía y loguea.
- [ ] CA-004: `mark-paid` de un registro dispara email "pago registrado"; `mark-received` dispara "salario recibido"; `mark-rejected` dispara "registro rechazado" con motivo; `closing` dispara "cierre de mes" con resumen.
- [ ] CA-005: Las claves de alerta nuevas aparecen en Settings → Alertas con toggle mail funcional.
- [ ] CA-006: Los emails usan la plantilla HTML única de `EmailService` con título y contenido correctos (verificar con email de prueba / logs).
- [ ] CA-007: Si el envío falla, la API no rompe (200 con ok, error solo en log).
- [ ] CA-008: CRUD de configs: crear/editar/eliminar/listar; duplicado de (child, category) devuelve 409.
- [ ] CA-009: `go build ./...` compila y `go test ./...` pasa (incluye tests de generación y de construcción de emails).

### 5.2 No funcionales

- [ ] CA-NF-001: Sin dependencias externas nuevas (ni colas, ni handlebars, ni nodemailer).
- [ ] CA-NF-002: Envío no bloqueante; latencia de la request no depende del SMTP.
- [ ] CA-NF-003: Generación de un mes con 20 salarios + 20 configs se ejecuta en < 1s (queries indexadas).

### 5.3 Testing

- **Unit tests**: `pension_generation` (crea/no crea por condiciones, idempotencia), `pension_email` (build de títulos/contenido por tipo, formateo de montos).
- **Integration tests**: Generación + envío best-effort contra SMTP falso/maqueteado; verificar guardado en logs sin romper el flujo.
- **E2E tests**: Con el usuario: generar mes → ver registros/salarios en la página → pagar/recibir → cerrar mes → verificar emails recibidos en el buzón real.
- **Carga/Performance**: Generación mensual con datos completos; tiempo de respuesta del endpoint.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración 0023 + modelo + storage + CRUD configs | 40 min | SPEC-049 |
| 2 | Claves de alerta nuevas (model + Seed) | 15 min | Ninguna |
| 3 | `pension_generation.go` (lógica de generación) | 45 min | Fase 1 |
| 4 | `pension_email.go` (build de emails por tipo) | 45 min | Fase 2 |
| 5 | Integrar notificaciones en handlers de records/salary/closing (SPEC-049) | 30 min | Fase 4 |
| 6 | Endpoint `POST /api/pension/generate` + wiring main.go | 20 min | Fase 3, 5 |
| 7 | Botón "Generar mes" en RegistrosPage (espejo SPEC-050) | 15 min | SPEC-050 |
| 8 | Tests Go + validación manual (emails reales) | 45 min | Fase 6, 7 |
| **Total** | | **~4.2 horas** | |

### 6.2 Milestones

1. **MVP**: Generación por botón + emails de estado/cierre (Fases 1-6).
2. **V1.0**: MVP + botón en UI + emails "registros creados" + validación manual con buzón real.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| SMTP lento bloquea la request al pagar/cerrar | Media | Medio | Envío en goroutine/best-effort; timeout en `EmailService` (revisar si aplica) |
| Usuario sin configs → generar no crea registros | Media | Medio | UI mínima de configs (P1) o seed; documentar que la generación requiere configs con auto_generate |
| Toggles de alerta nuevos no visibles en Settings | Baja | Medio | Seguir exactamente el patrón `Seed` de `alert_service.go` (SPEC-032) |
| Duplicar registros si el UNIQUE no aplica | Baja | Alto | `Exists` antes de insertar (misma lógica que `support_records.service.ts` de P4OLA) |
| Colisión de migración 0023 con specs paralelas | Baja | Medio | Verificar `migrations/` antes; las specs 043/048 no están en curso (043 in_progress desde 2026-08-31) |

## 8. Notas y Referencias

- P4OLA: `apps/api/src/modules/child-support/{child-support-cron.service,child-support-notification.service,email-queue.service,child-support-configs.service}.ts`, `entities/child-support-config.entity.ts`.
- p40la-ihost: `internal/services/bill_email.go`, `billing_scheduler.go`, `email_service.go`, `alert_service.go`, `internal/models/alert.go`, `internal/storage/notification.go` (destinatarios), SPEC-029/030/031/032/033/034/035/036/037 (sistema de alertas y emails).
- Specs del grupo: SPEC-049 (backend registros), SPEC-050 (frontend registros), SPEC-044 (sidebar).
- Restricciones iHost: sin dependencias nuevas; envío de email con stdlib (net/smtp).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-02 | p40la-ihost-team | Estado cambiado a pending_execution (aprobada para desarrollo) |
| 2026-09-02 | p40la-ihost-team | Estado cambiado a in_progress; inicio de desarrollo en worktree feature/SPEC-051 |
| 2026-09-02 | p40la-ihost-team | Implementación completa: migración 0023 child_support_configs + CRUD /api/pension/configs; POST /api/pension/generate idempotente; notificaciones best-effort (EmailService + destinatarios notifications + toggles de alerta pension_*) integradas en mark-paid/mark-received/mark-rejected/cierre; botón "Generar mes" en RegistrosPage + api client. Tests Go. Validado en local (go build, go test, server con generación y alertas). Fallo preexistente detectado: TestBillPayBill (no relacionado) |
| 2026-09-03 | p40la-ihost-team | Release: merge feature/SPEC-051 a main (commit b3d19b2), validación manual del usuario aprobada, issue #51 cerrado, worktree limpiado |