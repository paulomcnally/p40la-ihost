---
title: "Resumen diario de facturas pendientes por email"
id: "SPEC-031"
status: "released"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16 (v3: released)"
github_issue: 31
---

# Resumen diario de facturas pendientes por email

**ID**: SPEC-031  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16 (v3: released)

---

## 1. Resumen Ejecutivo

El sistema p40la-ihost genera facturas automáticamente (SPEC-008) y envía un email por cada factura nueva (SPEC-030), pero no existe una vista consolidada que permita al usuario saber de un vistazo **qué facturas siguen pendientes** y **desde hace cuántos días**. Revisar la app casa por casa es tedioso y se pierden de vista los pagos que se van acumulando.

Esta spec agrega un **resumen diario por email** que lista todas las facturas **pendientes** (`status: pending`) del sistema, agrupadas por **Casa**, **Institución** y **Servicio**. Cada factura muestra su **monto** y —dato clave— **cuántos días transcurrieron desde que se generó**, para que el usuario identifique rápidamente los pagos más antiguos y urgentes. El email se envía **una vez por día** a la hora configurada (`alert_check_hour`, ya existente de SPEC-029). **Si no hay facturas pendientes, no se envía ningún email.**

**Consideraciones de iHost**: Se reutiliza íntegramente el `EmailService` existente (SPEC-029) — `net/smtp` de stdlib, **cero dependencias nuevas**. No se crean tablas nuevas. El scheduler sigue el mismo patrón de `AlertScheduler` (ticker horario + deduplicación diaria con clave en `system_settings`). La query de pendientes es una sola lectura sobre `bills` con joins a `services`, `homes` e `institutions`. Consumo adicional de RAM despreciable.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Debe existir un **BillSummaryScheduler** que ejecute una vez por día a la hora configurada en `alert_check_hour` (clave existente de SPEC-029; fallback a `billing_generation_hour`).
2. **REQ-002**: El scheduler debe consultar todas las facturas con `status = pending` y `deleted_at IS NULL`, con sus datos de **Casa** (home), **Institución** y **Servicio** asociados.
3. **REQ-003**: Para cada factura pendiente, el email debe indicar **cuántos días pasaron desde que se generó** (diferencia entre hoy y `created_at` de la factura).
4. **REQ-004**: Si hay al menos una factura pendiente, debe enviarse **un único email resumen** a los destinatarios configurados en `alert_emails` (clave existente de SPEC-029), con la lista completa agrupada por casa/institución/servicio.
5. **REQ-005**: **Si no hay facturas pendientes, no se envía ningún email** (deduplicación diaria igual: se marca el check del día pero no se envía).
6. **REQ-006**: El envío usa el `EmailService` existente (`internal/services/email_service.go`) y su plantilla HTML única, sin modificar su interfaz.
7. **REQ-007**: El scheduler debe usar deduplicación por fecha (clave `last_bill_summary_check` en `system_settings`) para enviar el resumen **una sola vez por día**.
8. **REQ-008**: Si la config SMTP no está completa o no hay destinatarios configurados, el scheduler loguea un warning y no envía (sin afectar otras operaciones).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: El email debe estar organizado por **Casa**, mostrando dentro de cada casa los **servicios con facturas pendientes** y sus **instituciones**, con el monto de cada factura pendiente.
2. **REQ-010**: El email debe resaltar visualmente las facturas más antiguas (ej: badge "Hace N días" con color según antigüedad: >30 días rojo, >15 días naranja, resto gris).
3. **REQ-011**: El asunto (subject) debe ser descriptivo, por ejemplo: `"P40LA — Resumen de facturas pendientes (N pendientes)"`.
4. **REQ-012**: El email debe incluir el monto formateado con la moneda del servicio (reusar lógica existente de `currency` si aplica).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013**: Incluir en el email un enlace directo a la app (ej: `http://<ihost-ip>:8088/bills`) para ver y gestionar las facturas.
2. **REQ-014**: Permitir configurar desde Settings una hora independiente para el resumen (`bill_summary_hour`), separada de `alert_check_hour`.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El envío del email no debe bloquear el servidor HTTP. El scheduler corre en una goroutine separada (mismo patrón que AlertScheduler/BillingScheduler).
- **Seguridad — Protección de información sensible**: Las mismas reglas de SPEC-029 aplican: el password SMTP **nunca** se expone en responses de API, logs ni mensajes de error. El email no incluye credenciales ni datos sensibles (solo datos de facturas/servicios del propio usuario).
- **Almacenamiento**: Sin tablas nuevas. Se reutiliza `system_settings` (clave `last_bill_summary_check`).
- **Disponibilidad**: Si el servidor SMTP no responde, el scheduler loguea el error (sin credenciales). La deduplicación diaria se marca igual para no reintentar el mismo día; se reintenta al día siguiente.
- **iHost**: Cero dependencias nuevas. `net/smtp` está en stdlib. Una query SQL por día sobre `bills` (volumen bajo, < 1000 filas).

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-029** implementó `EmailService` (`internal/services/email_service.go`) con `Send`, `RenderTemplate`, `SendTest`, config SMTP y `alert_emails` en `system_settings`, protegidos.
- **SPEC-029** también implementó `AlertScheduler` (`internal/services/alert_scheduler.go`) — patrón de referencia: ticker horario, `GetAlertCheckHour` (fallback a `billing_generation_hour`), deduplicación diaria con clave, render de contenido HTML en función pura (`renderAlertsContent`).
- **SPEC-030** agregó el envío de email por factura automática (reutilizando el mismo `EmailService`).
- **Modelo `Bill`** (`internal/models/bill.go`): `ServiceID`, `Year`, `Month`, `Amount`, `Status`, `CreatedAt` (fecha de generación, clave para calcular antigüedad).
- **Modelo `Service`** (`internal/models/service.go`): `HomeID`, `Name`, `Institution`, `InstitutionID`, `Frequency`, `SuggestedAmount`, `BillingType`.
- **Modelo `Home`** (`internal/models/home.go`): `Name`.
- **Modelo `Institution`** (`internal/models/institution.go`): `Name`.
- **`BillStorage`** (`internal/storage/bill.go`) no tiene hoy una query de "listar pendientes con joins". Se debe agregar.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A: Nuevo `BillSummaryScheduler` siguiendo patrón de `AlertScheduler`** | Independiente, mismo patrón probado, deduplicación diaria propia, no toca la generación de facturas | Un scheduler más (mínimo overhead) | ✅ Seleccionada |
| **B: Integrar el resumen dentro del `BillingScheduler` existente** | Menos goroutines | Mezcla generación con notificación de pendientes; la hora de generación no es necesariamente la mejor para el resumen; acopla dos responsabilidades | ❌ Rechazada |
| **C: Query de pendientes en `BillStorage` con joins a services/homes/institutions** | SQL directo, una sola query, eficiente en SQLite | Requiere un struct de proyección nuevo | ✅ Seleccionada (para la query) |
| **D: Navegar services → bills por casa en Go** | Evita SQL complejo | Múltiples queries N+1, más memoria, menos eficiente | ❌ Rechazada |
| **E: Enviar siempre el resumen aunque no haya pendientes** | Confirmación diaria | Ruido innecesario; el usuario pidió no enviar si no hay pendientes | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Nuevo `BillSummaryScheduler` con el patrón de `AlertScheduler`**
- **Contexto**: SPEC-029 estableció un patrón claro y probado de scheduler diario (ticker horario + hora configurable + deduplicación por fecha). El resumen de pendientes es una responsabilidad distinta de la generación de facturas y de las alertas de seguros.
- **Decisión**: Crear `internal/services/bill_summary_scheduler.go` siguiendo el mismo patrón. Inyectar `BillStorage`, `ServiceStorage`/`HomeStorage`/`InstitutionStorage` (o la query combinada), `EmailService` y `SystemSettingsService`. Usar clave `last_bill_summary_check` para deduplicación.
- **Consecuencias**:
  - ✅ Independencia y testabilidad
  - ✅ Sin cambios en el flujo de generación de facturas (SPEC-008/030)
  - ⚠️ Una goroutine adicional (despreciable, ya hay BillingScheduler + AlertScheduler)

**ADR-002: Hora de envío = `alert_check_hour`**
- **Contexto**: El usuario eligió reutilizar la hora de las alertas de seguros en lugar de agregar configuración nueva.
- **Decisión**: El `BillSummaryScheduler` usa `GetAlertCheckHour(ctx)` (fallback a `billing_generation_hour`). Si se necesita hora propia en el futuro, se agrega `bill_summary_hour` (P2/REQ-014).
- **Consecuencias**:
  - ✅ Cero configuración nueva
  - ⚠️ Alertas de seguros y resumen de facturas se envían a la misma hora (aceptable)

**ADR-003: Query única en `BillStorage` con joins**
- **Contexto**: Se necesita facturas pendientes + nombre de casa, servicio e institución en una sola lectura.
- **Decisión**: Agregar método `ListPendingWithDetails(ctx)` en `BillStorage` que haga `SELECT` sobre `bills` con `LEFT JOIN services`, `LEFT JOIN homes`, `LEFT JOIN institutions` (y `LEFT JOIN currencies` si aplica), filtrando `bills.status = 'pending' AND bills.deleted_at IS NULL AND services.deleted_at IS NULL`. Devuelve un slice de `models.PendingBillDetail`.
- **Consecuencias**:
  - ✅ Eficiente (una query)
  - ✅ Datos completos para renderizar el email
  - ⚠️ Nuevo struct de proyección en `models`

**ADR-004: No enviar email si no hay pendientes, pero marcar el check diario**
- **Contexto**: El usuario pidió no enviar resumen si no hay facturas pendientes.
- **Decisión**: Si `len(pending) == 0`, se loguea "no hay facturas pendientes" y se marca `last_bill_summary_check = hoy` igual (para no re-evaluar el mismo día). Si hay pendientes, se envía el email y luego se marca.
- **Consecuencias**:
  - ✅ Sin ruido diario
  - ✅ Deduplicación consistente con el patrón existente

**ADR-005: Antigüedad calculada desde `created_at` de la factura**
- **Contexto**: El dato clave pedido por el usuario es "cuántos días desde que se generó la factura".
- **Decisión**: `daysSince := int(time.Since(bill.CreatedAt).Hours() / 24)` (o diferencia de fechas calendario truncando a días). Se muestra como "Hace N días" (N = 0 → "Hoy", N = 1 → "Ayer", N > 1 → "Hace N días").
- **Consecuencias**:
  - ✅ Implementación simple con `time.Time`
  - ⚠️ Uso de días calendario (no horas), consistente con percepción del usuario

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────────────────────────────────┐
│                   cmd/server/main.go                     │
│  Arranca BillSummaryScheduler + EmailService (goroutine) │
└───────────┬─────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────┐
│               BillSummaryScheduler (NUEVO)               │
│  (ticker horario, patrón AlertScheduler)                 │
│                                                          │
│  1. Lee alert_check_hour                                 │
│  2. Deduplicación last_bill_summary_check                │
│  3. BillStorage.ListPendingWithDetails()                 │
│  4. Si len>0: render + EmailService.Send                 │
└───────────┬───────────────────────────────┬─────────────┘
            │                               │
            ▼                               ▼
┌──────────────────────────┐   ┌──────────────────────────┐
│  BillStorage (EXTENDIDO) │   │  EmailService (existente)│
│  ListPendingWithDetails  │   │  Send/RenderTemplate     │
│  (JOIN services/homes/   │   │  (net/smtp stdlib)       │
│   institutions/currencies)│   └──────────┬───────────────┘
└──────────────────────────┘              │
            │                              ▼
            ▼                    ┌──────────────────────────┐
┌──────────────────────────┐    │  SMTP Server (UI config) │
│  SQLite (WAL)            │    └──────────────────────────┘
│  bills, services, homes, │
│  institutions, currencies│
└──────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 `internal/services/bill_summary_scheduler.go` (NUEVO)
- **Responsabilidad**: Scheduler diario que consulta facturas pendientes y envía el resumen por email.
- **Interfaz**:
  ```go
  type BillSummaryScheduler struct { ... }
  func NewBillSummaryScheduler(billStorage *storage.BillStorage, emailService *EmailService, settingsService *SystemSettingsService) *BillSummaryScheduler
  func (s *BillSummaryScheduler) Start()
  func (s *BillSummaryScheduler) Stop()
  func (s *BillSummaryScheduler) CheckNow() // usado por tests y API manual (patrón AlertScheduler)
  ```
- **Patrón**: Idéntico a `AlertScheduler` (ticker horario, `GetAlertCheckHour`, deduplicación con `last_bill_summary_check`).
- **Dependencias**: `BillStorage`, `EmailService`, `SystemSettingsService`
- **Ubicación**: `internal/services/bill_summary_scheduler.go`

#### 4.2.2 `internal/storage/bill.go` (EXTENDIDO)
- **Responsabilidad**: Agregar query de facturas pendientes con detalles.
- **Nuevo método**:
  ```go
  func (s *BillStorage) ListPendingWithDetails(ctx context.Context) ([]models.PendingBillDetail, error)
  ```
- **SQL**:
  ```sql
  SELECT b.id, b.service_id, b.year, b.month, b.amount, b.status, b.created_at,
         svc.name, svc.home_id, svc.institution,
         h.name, inst.name, cur.code, cur.symbol
  FROM bills b
  JOIN services svc ON svc.id = b.service_id AND svc.deleted_at IS NULL
  LEFT JOIN homes h ON h.id = svc.home_id AND h.deleted_at IS NULL
  LEFT JOIN institutions inst ON inst.id = svc.institution_id
  LEFT JOIN currencies cur ON cur.id = svc.currency_id
  WHERE b.status = 'pending' AND b.deleted_at IS NULL
  ORDER BY b.created_at ASC, svc.name
  ```
- **Ubicación**: Extender `internal/storage/bill.go`

#### 4.2.3 `internal/models/bill.go` (EXTENDIDO)
- **Nuevo struct de proyección**:
  ```go
  type PendingBillDetail struct {
      BillID      int64
      ServiceID   int64
      Year        int
      Month       int
      Amount      float64
      CreatedAt   time.Time
      ServiceName string
      HomeID      int64
      HomeName    string
      Institution string  // svc.institution o inst.name
      CurrencyCode   string
      CurrencySymbol string
  }
  ```

#### 4.2.4 `internal/services/bill_summary_email.go` (NUEVO)
- **Responsabilidad**: Construir el HTML del resumen (función pura, testeable).
- **Interfaz**:
  ```go
  func renderBillSummaryContent(pending []models.PendingBillDetail) string
  func daysSince(t time.Time) int
  ```
- **Ubicación**: `internal/services/bill_summary_email.go`

#### 4.2.5 `internal/services/email_service.go` (SIN CAMBIOS)
- Se reutiliza `Send`, `RenderTemplate` y `SendTest`.

#### 4.2.6 `internal/services/system_settings.go` (SIN CAMBIOS)
- Se reutiliza `GetAlertCheckHour`, `GetAlertEmails`.

### 4.3 Modelo de datos

Sin cambios de esquema. Solo lectura sobre tablas existentes:
- `bills` (status, created_at, service_id)
- `services` (name, home_id, institution, institution_id, currency_id)
- `homes` (name)
- `institutions` (name)
- `currencies` (code, symbol)

Clave nueva en `system_settings`:
```
- last_bill_summary_check  TEXT  -- ej: "2026-08-16" (deduplicación diaria)
```

### 4.4 Contenido del Email

**Subject**: `P40LA — Resumen de facturas pendientes (N pendientes)`

**Contenido HTML** (usando la plantilla única de SPEC-029):

```html
<p>Hola,</p>

<p>Tenés <strong>N facturas pendientes</strong> de pago. Este es el resumen:</p>

<!-- Por cada casa -->
<h3 style="margin:24px 0 8px;color:#1d1d1f;font-size:16px;">🏠 {Nombre de la Casa}</h3>

<table width="100%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin-top:8px;">
  <tr style="background-color:#f5f5f7;">
    <th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Institución</th>
    <th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Servicio</th>
    <th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Período</th>
    <th style="text-align:right;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Monto</th>
    <th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Antigüedad</th>
  </tr>
  <tr>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">Claro</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">Internet residencial</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">Agosto 2026</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;text-align:right;">$1.500,00</td>
    <td style="padding:12px;border-bottom:1px solid #e5e5ea;">
      <span style="background-color:#ff3b30;color:#fff;border-radius:8px;padding:2px 8px;font-size:12px;">Hace 12 días</span>
    </td>
  </tr>
</table>

<p style="margin-top:24px;color:#8e8e93;font-size:13px;">
  Ingresá a <a href="http://ihost:8088/bills" style="color:#007aff;">P40LA</a> para gestionar tus facturas pendientes.
</p>
```

**Badges de antigüedad**:
- 0 días → "Hoy" (gris)
- 1 día → "Ayer" (gris)
- 2-15 días → "Hace N días" (naranja `#ff9500`)
- >15 días → "Hace N días" (rojo `#ff3b30`)

### 4.5 Dependencias

- **Internas**:
  - `BillStorage` (extendido con `ListPendingWithDetails`)
  - `EmailService` (existente, SPEC-029)
  - `SystemSettingsService` (existente, SPEC-029)
  - Modelos `Bill`, `Service`, `Home`, `Institution`, `Currency` (solo lectura)
  - `AlertScheduler` (patrón a seguir, no se modifica)
- **Externas**:
  - **NINGUNA nueva**. Se usa `net/smtp` de la stdlib (ya en el proyecto).

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado que existen facturas con `status = pending`, cuando el `BillSummaryScheduler` ejecuta, entonces se envía un único email a los destinatarios de `alert_emails` con el listado completo.
- [x] CA-002: Dado el email recibido, entonces cada factura pendiente muestra Casa, Institución, Servicio, Período, Monto y **días desde que se generó**.
- [x] CA-003: Dado que no hay facturas pendientes, cuando el scheduler ejecuta, entonces **no se envía ningún email** y se marca el check del día.
- [x] CA-004: Dado que el scheduler ya ejecutó hoy (`last_bill_summary_check` = hoy), cuando se llama nuevamente, entonces no se envía email duplicado.
- [x] CA-005: Dado que una factura se generó hoy, entonces el email la muestra como "Hoy".
- [x] CA-006: Dado que una factura se generó hace 15+ días, entonces el email la muestra con badge rojo "Hace N días".
- [x] CA-007: Dado que la config SMTP está incompleta o no hay destinatarios, cuando el scheduler intenta enviar, entonces loguea un warning y no envía.
- [x] CA-008: Dado el subject del email, entonces incluye la cantidad de facturas pendientes (ej: "(3 pendientes)").
- [x] CA-009: Dado un servicio sin institución asociada, cuando el email se renderiza, entonces se muestra la institución como el campo `institution` del servicio o "—" si está vacío (sin romper el render).

### 5.2 No funcionales

- [x] CA-NF-001: El binario compilado no aumenta más de 50KB (sin dependencias nuevas).
- [x] CA-NF-002: El scheduler no bloquea el servidor HTTP (corre en goroutine).
- [x] CA-NF-003: La query `ListPendingWithDetails` se ejecuta en < 10ms con 1000 facturas.
- [x] CA-NF-004: El build compila sin errores con `go build ./...`.
- [x] CA-NF-005: `go vet ./...` no reporta errores.
- [x] CA-NF-006: `grep -rn "smtp_password\|smtp_user\|postmaster" internal/ cmd/` no encuentra credenciales hardcoded.
- [x] CA-NF-007: El email no incluye credenciales ni datos sensibles.

### 5.3 Testing

- **Unit tests**:
  - `bill_summary_email_test.go`: `renderBillSummaryContent` con casos: múltiples casas, sin institución, con/sin facturas; `daysSince` con fechas de 0, 1, 15, 30 días.
  - `bill_summary_scheduler_test.go`: deduplicación diaria, no envío si no hay pendientes, warning si SMTP incompleto (mock storage/email).
- **Integration tests**:
  - Flujo completo: insertar factura pendiente con service/home/institution → ejecutar scheduler → verificar email (mock) con datos correctos.
  - Sin pendientes → no se llama `EmailService.Send`.
  - Deduplicación: ejecutar dos veces el mismo día → segundo no envía.
- **E2E tests (manuales)**:
  - Configurar SMTP y `alert_emails` desde UI → dejar facturas pendientes → verificar recepción del resumen diario.
  - Verificar formato/colores en Gmail/Apple Mail.
  - Sin pendientes → no llega email.
- **Carga/Performance**: N/A (volumen bajo, < 1000 facturas).

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modelo `PendingBillDetail` + `BillStorage.ListPendingWithDetails` con joins + unit test de la query | 0.5 día | Ninguna |
| 2 | Helper `renderBillSummaryContent` + `daysSince` + unit tests (fijo/variable, sin institución, antigüedad) | 0.5 día | Fase 1 |
| 3 | `BillSummaryScheduler` (ticker, deduplicación, hora alert_check_hour) + unit tests con mocks | 0.75 día | Fases 1, 2 |
| 4 | Integrar `BillSummaryScheduler` en `main.go` (Start/Stop) | 0.25 día | Fase 3 |
| 5 | Tests de integración + build + vet + validación manual local | 0.5 día | Todas |

**Estimación total**: ~2.5 días

### 6.2 Milestones

1. **MVP**: Query de pendientes + scheduler + email resumen con casa/institución/servicio/monto/antigüedad. Sin UI (config via API/settings existentes).
2. **V1.0**: MVP + enlace a la app (P2) + tests de integración completos.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| SMTP mal configurado: resumen no llega | Media | Medio | Warning en logs sin bloquear; endpoint `test-email` de SPEC-029 permite validar |
| Mismo horario que alertas de seguros (puede saturar bandeja) | Media | Bajo | Aceptable para uso personal. P2 agrega `bill_summary_hour` independiente |
| Volumen de facturas pendientes acumulado | Media | Bajo | Una tabla con todas las facturas, ordenada por antigüedad. Si crece, paginar o resumir (P2) |
| Institución ausente en servicios viejos | Media | Bajo | Mostrar campo `institution` del servicio o "—" |
| Diferencia de zonas horarias en cálculo de días | Baja | Bajo | Usar fechas locales del iHost (`time.Now()`), consistente con el resto del sistema |
| Email HTML con emoji 🏠 se ve mal en algunos clientes | Baja | Bajo | Probar en Gmail/Apple Mail; fallback: texto plano "Casa:" |

---

## 8. Notas y Referencias

- SPEC-029: `EmailService`, `AlertScheduler` (patrón), `alert_emails`, `alert_check_hour`, protección SMTP.
- SPEC-008: `BillingScheduler` y generación automática de facturas (origen de los pendientes).
- SPEC-030: Email por factura automática (relacionado, no se modifica).
- `internal/services/alert_scheduler.go` (patrón de scheduler diario)
- `internal/services/email_service.go` (plantilla única)
- `internal/storage/bill.go` (a extender)
- `internal/models/bill.go`, `internal/models/service.go`, `internal/models/home.go`, `internal/models/institution.go`

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación. Decisiones: no enviar si no hay pendientes, hora = `alert_check_hour`, un email diario agrupado por casa/institución/servicio con antigüedad en días. |
| 2026-08-16 | paulomcnally | Estado a `in_progress`. Desarrollo iniciado. |
| 2026-08-16 | paulomcnally | **Released** junto con SPEC-030/032/033 (commit único del sistema de alertas multicanal). Validado en local por el usuario. |