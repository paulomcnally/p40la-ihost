---
title: "Fechas de emisión y vencimiento en facturas de servicios y deudas + webhooks"
id: "SPEC-081"
status: "in_progress"
author: "paulomcnally"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 84
---

# Fechas de emisión y vencimiento en facturas de servicios y deudas + webhooks

**ID**: SPEC-081  
**Estado**: in_progress  
**Autor**: paulomcnally  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

Las facturas de servicios (`bills`) y las cuotas de deudas (`debt_bills`) carecen de una fecha de **emisión** (cuándo el proveedor generó la factura) y de una fecha de **vencimiento** explícita en `bills` (cuándo hay que pagarla antes de que suspendan el servicio). Hoy solo existe `created_at` (cuándo se registró localmente), que no refleja la fecha real del proveedor. Al conectarse directamente a los sistemas externos (vía webhooks), estos podrán enviar ambas fechas.

Se agregan las columnas `issue_date` (fecha de emisión) y `due_date` (fecha de vencimiento) a ambas tablas, **opcionales y con NULL por defecto**: no todos los sistemas externos las proveen, por lo que ningún flujo existente debe romperse. En `debt_bills` ya existe `due_date` (fecha de pago de la cuota calculada desde `payment_day`), que cubre el vencimiento; se agrega únicamente `issue_date`. El webhook de facturas por servicio (`POST /webhooks/{uuid}`, SPEC-069) aceptará `issue_date` y `due_date` opcionales y las persistirá al crear o actualizar facturas.

**En la UI** (páginas que muestran facturas y cuotas) se mostrará la fecha de vencimiento como un texto relativo tipo "En N días" con un **label semáforo**: verde si faltan más de 10 días, amarillo entre 10 y 1 día (inclusive), rojo cuando vence hoy o ya venció. El cálculo de "hoy" debe usar la **zona horaria configurada del sistema** (setting `timezone` de SPEC-078, nombre IANA) con fallback a la zona del navegador, para que la información sea real y no dependa del UTC.

**Nota de excepción a la política vigente**: AGENTS.md prohíbe modificar el esquema de `debts`/`debt_bills`. El usuario (dueño de la política) solicita explícitamente agregar `issue_date` a `debt_bills` en este spec; es una autorización puntual y documentada, sin cambios de semántica en las columnas existentes.

Consideraciones iHost: cambio de esquema mínimo (2 columnas nullable por tabla), sin dependencias nuevas, sin impacto relevante en RAM/almacenamiento. El label semáforo se implementa sin librerías nuevas (helper propio + `Intl`), respetando los tokens del tema (`bg-success`, `bg-warning`, `bg-danger` + `text-text`).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar columna `issue_date` (fecha de emisión de la factura) nullable a la tabla `bills`. No es `created_at`: la provee el sistema externo.
2. **REQ-002**: Agregar columna `due_date` (fecha de vencimiento) nullable a la tabla `bills`. Permite saber cuántos días quedan antes de que suspendan el servicio.
3. **REQ-003**: Agregar columna `issue_date` nullable a la tabla `debt_bills`. El vencimiento de las cuotas ya está cubierto por la columna existente `due_date` (TEXT, no nullable, calculada desde `payment_day`).
4. **REQ-004**: Exponer `issue_date` y `due_date` (opcionales, no requeridas) en el payload del webhook `POST /webhooks/{uuid}` (SPEC-069): aceptarlas, validarlas, normalizarlas a `YYYY-MM-DD` y persistirlas al crear y al actualizar facturas.
5. **REQ-005**: Exponer `issue_date` y `due_date` en las respuestas JSON de las APIs de facturas y cuotas (via modelos `Bill` y `DebtBill`), con valor `null`/omitido cuando no existan.
6. **REQ-008**: Mostrar en las UI que listan facturas (`BillsPage.tsx`) y cuotas (`DebtBillsPage.tsx`) un texto relativo tipo "En N días" basado en `due_date`, con un **label semáforo**: verde si faltan más de 10 días, amarillo entre 10 y 1 día (inclusive), rojo si vence hoy o ya venció. El cálculo de "hoy" DEBE usar la zona horaria configurada en el sistema (setting `timezone`, SPEC-078) con fallback a la zona del navegador, nunca UTC a ciegas. Si `due_date` es NULL o la factura está pagada, no se muestra el semáforo.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Al actualizar una factura existente vía webhook (upsert), los nuevos valores de `issue_date`/`due_date` sobrescriben los previos; si el campo viene vacío/ausente, **no** borra el valor existente (semántica additiva del upsert actual).
2. **REQ-007**: Persistir `issue_date` en la generación automática de cuotas de deuda (solo si el flujo de generación lo provee; si no, queda NULL). Sin cambios de comportamiento en `due_date`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Mostrar en las UI de facturas y cuotas la fecha de emisión (`issue_date`) junto a la fecha de vencimiento, además del label semáforo de vencimiento.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto medible; solo columnas adicionales en SELECT/INSERT existentes.
- **Seguridad**: Sin cambios en autenticación; el webhook sigue validando la api_key global y el UUID del servicio.
- **Almacenamiento**: 2 columnas TEXT nullable por fila; incremento despreciable en el tamaño de la DB.
- **Disponibilidad**: Migración 0030 aditiva (`ADD COLUMN`), aplicable en caliente; sin reescritura de tabla en `.up.sql`.
- **iHost**: Sin dependencias nuevas, sin procesos extra, consumo de RAM marginal.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Tabla `bills`** (`migrations/0005_create_bills.up.sql` + `0015`): columnas actuales `service_id, year, month, amount, invoice_number, status, drive_url, file_hash, paid_at, payment_reference, deleted_at, created_at, updated_at`. Sin fecha de emisión ni vencimiento.
- **Tabla `debt_bills`** (`migrations/0024_create_debts_and_debt_bills.up.sql`): ya tiene `due_date TEXT NOT NULL` (día de pago de la cuota) + `UNIQUE(debt_id, installment_number)`.
- **Webhook** (SPEC-069/071/074): `POST /webhooks/{uuid}` → `WebhookService.UpsertBill` con payload `WebhookBillPayload{year, month, amount, invoice_number, status, paid_at, payment_reference, drive_url}`; `year/month/amount` obligatorios, resto opcional. No existe webhook de deudas.
- **Migraciones**: runner en `internal/db/db.go` (lee `NNNN_*.up.sql` en orden). Precedente de `ADD COLUMN` aditivo en `0027`, `0028`, `0029`; precedente de `DROP COLUMN` en el `.down.sql` de `0029`.
- **Convención de fechas**: `paid_at` se parsea aceptando RFC3339, `2006-01-02T15:04:05`, `2006-01-02 15:04:05` y `2006-01-02`; `debt_bills.due_date` es `TEXT` en formato `YYYY-MM-DD`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| TEXT nullable `YYYY-MM-DD` | Consistente con `debt_bills.due_date`; simple para sistemas externos; sin timezone | Pierde precisión horaria | ✅ Seleccionada |
| DATETIME nullable (como `paid_at`) | Consistente con `paid_at`; permite hora exacta | Sobredimensionado para "fecha de vencimiento"; inconsistente con `due_date` de deudas | ❌ Rechazada |
| Agregar `due_date` duplicada a `debt_bills` | Simetría con `bills` | Colisión de nombre con columna existente; confusión de semántica | ❌ Rechazada |
| Backfill de `issue_date` desde `created_at` en migración | Datos históricos poblados | Crea datos falsos que no son la fecha real del proveedor | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Almacenar fechas como `TEXT` nullable en formato `YYYY-MM-DD`
- **Contexto**: El webhook puede recibir las fechas en distintos formatos y los sistemas externos trabajan con fechas día.
- **Decisión**: Columnas `issue_date`/`due_date` como `TEXT` (nullable, default NULL) en `bills`; `issue_date` igual en `debt_bills`. El webhook acepta RFC3339 o `YYYY-MM-DD` y normaliza a `YYYY-MM-DD`.
- **Consecuencias**: Consistencia con `debt_bills.due_date`; parsing simple sin zonas horarias; el frontend puede formatearlas como fechas cortas. Se pierde la hora exacta (no necesaria).

**ADR-002**: Excepción documentada a la política de no tocar el esquema de `debt_bills`
- **Contexto**: AGENTS.md prohíbe modificar el esquema de `debts`/`debt_bills` (política del usuario).
- **Decisión**: El usuario solicita explícitamente esta columna en este spec. Se documenta como autorización puntual; solo se agrega `issue_date`, no se altera ninguna columna ni constraint existente.
- **Consecuencias**: La generación de cuotas sigue funcionando idéntico; `due_date` existente queda como fecha de vencimiento de las cuotas.

**ADR-003**: `due_date` en `debt_bills` no se duplica
- **Contexto**: El usuario pidió fecha de vencimiento en facturas de deudas y servicios.
- **Decisión**: En `debt_bills`, la columna existente `due_date` cumple ese rol; se agrega solo `issue_date`.
- **Consecuencias**: No hay cambios de comportamiento en el cálculo de cuotas; el requisito de vencimiento queda satisfecho sin duplicación.

**ADR-004**: Semáforo de vencimiento calculado con la zona horaria configurada del sistema
- **Contexto**: `due_date` es una fecha plana (`YYYY-MM-DD`) sin zona horaria. El usuario exige que el "En N días" refleje la realidad según la configuración del sistema (SPEC-078), no UTC.
- **Decisión**: El frontend obtiene la zona IANA del setting `timezone` vía API de settings (fallback: `getBrowserTimeZone()`), calcula el "hoy" con `Intl.DateTimeFormat` en esa zona (sin librerías nuevas), y deriva `daysRemaining = due_date - today`. Umbrales: verde si `daysRemaining > 10`; amarillo si `1 <= daysRemaining <= 10`; rojo si `daysRemaining <= 0` (vence hoy o ya venció). El label usa los tokens del tema (`bg-success/20`, `bg-warning/20`, `bg-danger/20`) con `text-text`.
- **Consecuencias**: Información correcta aunque el browser del usuario esté en otra zona; sin dependencias nuevas (solo `Intl`); requiere que el setting `timezone` esté disponible en el frontend (ya lo usa `SettingsBillingPage`).

**ADR-005**: La fecha de emisión (`issue_date`) no genera semáforo
- **Contexto**: El usuario pidió el semáforo específicamente para el vencimiento ("Si es la fecha de cuando vence").
- **Decisión**: El label semáforo y el texto "En N días" se basan solo en `due_date`. `issue_date` se muestra como dato informativo (P2, REQ-009).
- **Consecuencias**: Semántica clara: el semáforo alerta sobre suspensión del servicio, no sobre emisión.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Sistema externo] --POST /webhooks/{uuid} (issue_date, due_date opcionales)--> [WebhookService.UpsertBill]
                                                                                   |
                                                                                   v
                                                          [BillStorage: Create / UpdateWebhookFields]
                                                                                   |
                                                                                   v
                                        [SQLite: bills(issue_date, due_date) + debt_bills(issue_date)]
                                                                                   |
                                                                                   v
                                     [APIs REST: JSON de Bill y DebtBill incluye los nuevos campos]
```

### 4.2 Componentes

#### 4.2.1 Migración `0030_add_bills_debt_bills_dates`
- **Responsabilidad**: Agregar las columnas `issue_date` y `due_date` a `bills`, y `issue_date` a `debt_bills`.
- **Interfaz**: `.up.sql` / `.down.sql` estándar del runner `internal/db/db.go`.
- **Dependencias**: Ninguna.
- **Ubicación**: `migrations/0030_add_bills_debt_bills_dates.up.sql` y `.down.sql`.

#### 4.2.2 Modelos `Bill` y `DebtBill`
- **Responsabilidad**: Exponer los nuevos campos en JSON.
- **Interfaz**: `IssueDate *string json:"issue_date,omitempty"` y `DueDate *string json:"due_date,omitempty"` en `Bill`; `IssueDate *string json:"issue_date,omitempty"` en `DebtBill`.
- **Ubicación**: `internal/models/bill.go`, `internal/models/debt_bill.go`.

#### 4.2.3 Storage `BillStorage` y `DebtBillStorage`
- **Responsabilidad**: Incluir las columnas en SELECTs (columnas base), en `Create`, en `UpdateWebhookFields` y en `Update`.
- **Interfaz**: `UpdateWebhookFields(ctx, id, amount, invoiceNumber, driveURL, issueDate, dueDate *string)`; `Create` persiste `issue_date`/`due_date` cuando vienen seteados.
- **Ubicación**: `internal/storage/bill.go`, `internal/storage/debt_bill.go`.

#### 4.2.4 Servicio `WebhookService`
- **Responsabilidad**: Aceptar y validar `issue_date`/`due_date` opcionales en `WebhookBillPayload`; normalizar a `YYYY-MM-DD`; persistir en creación y actualización.
- **Interfaz**: Nuevos campos en `WebhookBillPayload`; helper `parseWebhookDate(raw string, allowFuture bool)` reutilizando los layouts de `parseWebhookPaidAt` (permitiendo fechas futuras, propio de un vencimiento).
- **Ubicación**: `internal/models/webhook.go`, `internal/services/webhook.go`.

#### 4.2.5 Frontend: helper `getDaysUntilDue` y componente `DueDateBadge`
- **Responsabilidad**: Calcular días restantes hasta `due_date` en la zona horaria configurada y renderizar el label semáforo con texto relativo i18n.
- **Interfaz**: 
  - Helper `getDaysUntilDue(dueDate: string, timezone: string): number` — parsea `YYYY-MM-DD` como fecha local de esa zona (vía `Intl.DateTimeFormat` para obtener "hoy" en la zona) y devuelve `Math.ceil((dueDate - today) / 86400000)` en días de calendario.
  - Componente `DueDateBadge({ dueDate, timezone })` — retorna `null` si `dueDate` es falsy; si la factura está pagada el caller no lo renderiza. Estilos con tokens del tema: `bg-success/20 text-green-800 dark:text-green-400` (verde, `>10`), `bg-warning/20 text-yellow-800 dark:text-yellow-400` (amarillo, `1<days<=10`), `bg-danger/20 text-red-800 dark:text-red-400` (rojo, `days<=1`).
  - Textos i18n (nuevas claves en `frontend/public/i18n/{es,en}.json`): `bills.due_in_days` ("En N días"), `bills.due_today` ("Vence hoy"), `bills.due_overdue` ("Vencida hace N días").
- **Dependencias**: Zona horaria desde settings (`api.systemSettings.get()` → `timezone`, patrón de `SettingsBillingPage.tsx`) con fallback `getBrowserTimeZone()` de `frontend/src/constants/timezones.ts`. Sin librerías nuevas (solo `Intl`).
- **Ubicación**: `frontend/src/components/DueDateBadge.tsx`, `frontend/src/utils/dates.ts` (o dentro del componente si no existe el directorio utils).

### 4.3 Modelo de datos

```
Entidad: bills
- issue_date: TEXT nullable (fecha de emisión YYYY-MM-DD, provista por el sistema externo)
- due_date: TEXT nullable (fecha de vencimiento YYYY-MM-DD)
- Relaciones: services (N:1) — sin cambios

Entidad: debt_bills
- issue_date: TEXT nullable (fecha de emisión YYYY-MM-DD)
- due_date: TEXT NOT NULL (existente, fecha de pago de la cuota — vencimiento)
- Relaciones: debts (N:1) — sin cambios
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /webhooks/{uuid}` (payload extendido, SPEC-069)

**Request** (campos nuevos opcionales):
```json
{
  "year": 2026,
  "month": 9,
  "amount": 1250.0,
  "invoice_number": "INV-2026-09",
  "issue_date": "2026-09-10",
  "due_date": "2026-10-05",
  "status": "pending"
}
```

**Response 200** (factura con los campos nuevos):
```json
{
  "bill": {
    "id": 42,
    "service_id": 7,
    "year": 2026,
    "month": 9,
    "amount": 1250.0,
    "status": "pending",
    "issue_date": "2026-09-10",
    "due_date": "2026-10-05",
    "created_at": "2026-09-19T10:00:00Z"
  },
  "created": true
}
```

**Comportamiento de validación**:
- `issue_date`/`due_date` ausentes o vacíos → se guarda NULL (o se conserva el valor previo en update).
- Formato inválido → error 400 (`{"error":"invalid_date","message":"issue_date debe ser una fecha válida (YYYY-MM-DD o RFC3339)"}`).
- Fechas futuras permitidas (un vencimiento normalmente es futuro).

**Response Error** (sin cambios de contrato):
```json
{
  "error": "invalid_date",
  "message": "issue_date debe ser una fecha válida (YYYY-MM-DD o RFC3339)"
}
```

#### Endpoints REST existentes (facturas y cuotas)

Las respuestas JSON de `GET /api/services/{id}/bills`, `GET /api/debts/{id}/bills`, etc. incluyen `issue_date` y `due_date` (o `debt_bills` con `issue_date`) cuando existen, vía los modelos.

#### UI: label semáforo en listados

En `BillsPage.tsx` (cards móvil y tabla desktop) y `DebtBillsPage.tsx` (cards de cuotas), para facturas **pendientes** con `due_date` definido, junto al badge de estado se renderiza `DueDateBadge`:

```
Vencimiento: [En 12 días]  (verde)   → bg-success/20, text-green-800
             [En 5 días]   (amarillo)→ bg-warning/20, text-yellow-800
             [Vence hoy]   (rojo)    → bg-danger/20, text-red-800
             [Vencida hace 3 días]   → bg-danger/20, text-red-800
```

Reglas:
- Solo facturas `pending` con `due_date` no nulo.
- "Hoy" se calcula en la zona horaria configurada del sistema (setting `timezone`, fallback browser).
- `daysRemaining > 10` → verde · `1 <= daysRemaining <= 10` → amarillo · `daysRemaining <= 0` → rojo (vence hoy o ya venció).

### 4.5 Dependencias

- **Internas**: `internal/models/bill.go`, `internal/models/debt_bill.go`, `internal/models/webhook.go`, `internal/storage/bill.go`, `internal/storage/debt_bill.go`, `internal/services/webhook.go`, `internal/api/webhook_handlers.go` (sin cambios de firma de handler), tests existentes de webhook/storage.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado una DB con migraciones aplicadas, cuando se corre la migración 0030, entonces `bills` tiene `issue_date` y `due_date` (nullable) y `debt_bills` tiene `issue_date` (nullable), sin datos afectados.
- [ ] CA-002: Dado un webhook con `issue_date` y `due_date` válidos, cuando se crea una factura nueva, entonces ambas fechas quedan persistidas en `YYYY-MM-DD` y aparecen en la respuesta JSON.
- [ ] CA-003: Dado un webhook sin `issue_date`/`due_date`, cuando se crea una factura, entonces las columnas quedan NULL y el flujo no falla.
- [ ] CA-004: Dado un webhook con `issue_date` inválido (ej: `2026-13-40`), cuando se procesa, entonces responde 400 sin crear/actualizar la factura.
- [ ] CA-005: Dado un webhook de actualización con `issue_date`/`due_date` nuevos, cuando se procesa sobre una factura existente, entonces los valores se sobrescriben; si el campo viene ausente, el valor previo se conserva.
- [ ] CA-006: Dado una cuota de deuda existente, cuando se consulta la API, entonces `debt_bills` expone `issue_date` (null si no fue seteado) y `due_date` sigue funcionando como antes.
- [ ] CA-007: Dado el flujo de generación automática de cuotas (SPEC-054), cuando se genera una cuota, entonces `issue_date` queda NULL y `due_date` conserva su cálculo actual (sin cambios de comportamiento).
- [ ] CA-008: Dado el flujo de pagos (SPEC-043) y soft-delete/reactivación (SPEC-071), cuando se paga o reactiva una factura, entonces las nuevas columnas no interfieren y se conservan.
- [ ] CA-009: Dado una factura pendiente con `due_date` dentro de 12 días y la zona horaria configurada, cuando se renderiza la lista, entonces se muestra "En N días" con label verde (más de 10 días).
- [ ] CA-010: Dado una factura pendiente con `due_date` dentro de 5 días, cuando se renderiza la lista, entonces el label es amarillo (entre 10 y 1 día).
- [ ] CA-011: Dado una factura pendiente con `due_date` hoy o ya vencida, cuando se renderiza la lista, entonces el label es rojo ("Vence hoy" / "Vencida hace N días").
- [ ] CA-012: Dado una factura pagada o con `due_date` NULL, cuando se renderiza la lista, entonces NO se muestra el semáforo.
- [ ] CA-013: Dado que el server tiene configurada una zona horaria distinta a la del browser, cuando se calcula "hoy" para el semáforo, entonces se usa la zona del sistema (verificado en el límite de medianoche, ej: hora local del browser 23:00 vs zona del sistema ya en el día siguiente).

### 5.2 No funcionales

- [ ] CA-NF-001: La migración 0030 es aditiva y aplica en caliente sobre una DB existente sin pérdida de datos.
- [ ] CA-NF-002: Los tests existentes de webhook y storage pasan sin modificaciones de semántica.

### 5.3 Testing

- **Unit tests**: parseo/normalización de `issue_date`/`due_date` (formatos válidos, inválidos, futuros, vacíos); `UpdateWebhookFields` con y sin fechas; `BillStorage.Create` con y sin fechas; helper `getDaysUntilDue` con distintos thresholds y zonas horarias (límite de medianoche).
- **Integration tests**: upsert vía webhook (crear y actualizar) con y sin las fechas; 400 con fecha inválida; generación de cuotas sin cambios.
- **E2E tests**: enviar payload con fechas al webhook real y verificar persistencia en DB + respuesta JSON; verificar el semáforo en `BillsPage` y `DebtBillsPage` en darkmode con fechas verde/amarillo/rojo.
- **Carga/Performance**: N/A (cambio aditivo mínimo).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0030` (up/down) para `bills` y `debt_bills` | 0.5 día | Ninguna |
| 2 | Modelos: `Bill`/`DebtBill`/`WebhookBillPayload` con campos nuevos | 0.5 día | Fase 1 |
| 3 | Storage: columnas base, `Create`, `UpdateWebhookFields` en `BillStorage` y `DebtBillStorage` | 1 día | Fase 2 |
| 4 | WebhookService: parseo, validación y persistencia de las fechas | 1 día | Fase 3 |
| 5 | Tests unitarios/integración + validación local (server + DB SQLite) | 1 día | Fases 1-4 |
| 6 | Frontend: helper `getDaysUntilDue` + `DueDateBadge` + integración en `BillsPage` y `DebtBillsPage` + claves i18n | 1 día | Fase 2 |

### 6.2 Milestones

1. **MVP**: Fases 1-5 (backend + webhooks + tests).
2. **V1.0**: Fase 6 (UI con semáforo de vencimiento) — requerida por el usuario (REQ-008), parte del alcance final de este spec.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Romper `UNIQUE(service_id, year, month)` o soft-delete (SPEC-071) al tocar INSERT/UPDATE | Baja | Alto | Cambios aditivos; no se tocan constraints; tests existentes de webhook/storage siguen pasando |
| Sistemas externos envían formatos de fecha no contemplados | Media | Bajo | Aceptar RFC3339 y `YYYY-MM-DD` (mismos layouts que `paid_at`); error 400 claro para formatos raros |
| Backfill accidental de `issue_date` con `created_at` | Baja | Medio | No hacer backfill (ADR); NULL por defecto |
| Violación de la política "no tocar esquema debt_bills" | Baja | Medio | Autorización explícita del usuario documentada en este spec (ADR-002) |
| Semáforo con "hoy" en UTC causa rojo/verde equivocado cerca de medianoche | Media | Medio | Calcular "hoy" con `Intl.DateTimeFormat` en la zona configurada del sistema (ADR-004); CA-013 |
| Umbrales del semáforo malinterpretados (1 día en amarillo vs rojo) | Media | Baja | ADR-004 fija amarillo hasta 1 día inclusive y rojo para `daysRemaining <= 0` (vence hoy o ya venció); corregido por el usuario durante definición |

## 8. Notas y Referencias

- SPEC-054: modelo de deudas y generación de cuotas (`debt_bills.due_date`).
- SPEC-069: webhooks por servicio para facturas (`WebhookBillPayload`).
- SPEC-070: historial de cambios de facturas (auditoría).
- SPEC-071: reactivación de facturas soft-deleted vía webhook.
- Precedentes de migraciones: `0027`-`0029` (ADD COLUMN aditivo), `.down.sql` de `0029` (DROP COLUMN).
- No existe webhook de deudas; si el usuario quiere alimentar `debt_bills.issue_date` desde sistemas externos, se evaluará un webhook de deudas en un spec futuro.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | paulomcnally | Creación inicial de la especificación |
| 2026-09-19 | paulomcnally | Agregado REQ-008 (UI): texto relativo "En N días" + label semáforo (verde >10 días, amarillo 10-1, rojo ≤1 o vencida) calculado con la zona horaria configurada del sistema (ADR-004/005, CA-009..013, fase 6) |
| 2026-09-19 | paulomcnally | Corregidos umbrales del semáforo (solicitud del usuario): amarillo hasta 1 día inclusive (`1 <= daysRemaining <= 10`), rojo solo cuando vence hoy o ya venció (`daysRemaining <= 0`) |