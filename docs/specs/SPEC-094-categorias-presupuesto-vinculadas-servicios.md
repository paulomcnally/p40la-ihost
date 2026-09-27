---
title: "Vincular categorías de presupuesto con servicios: transacción automática al pagar facturas"
id: "SPEC-094"
status: "released"
author: "opencode"
created: "2026-09-27"
updated: "2026-09-27"
github_issue: 97
---

# Vincular categorías de presupuesto con servicios: transacción automática al pagar facturas

**ID**: SPEC-094  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-27  
**Actualizado**: 2026-09-27

---

## 1. Resumen Ejecutivo

El módulo de Presupuesto (SPEC-093) permite registrar transacciones manualmente contra categorías, pero el usuario debe crearlas a mano cada vez que paga una factura de un servicio existente en el sistema. Este requerimiento pide **vincular una categoría de presupuesto con un servicio** del sistema de servicios, de modo que **cuando una factura de ese servicio se marque como pagada, se cree automáticamente una transacción en la categoría vinculada** (outflow por el monto de la factura, en la moneda del servicio, con la fecha de pago).

La asociación puede hacerse **durante la creación de la categoría** (campo "Servicio" en el formulario de categoría) o **después de la creación** (editando la categoría para asignarle o cambiarle el servicio). Esto elimina el registro manual duplicado de gastos: pagar una factura de Claro, por ejemplo, refleja automáticamente el gasto en la categoría "Internet" del presupuesto.

**Consideraciones iHost**: no se agregan dependencias nuevas. Se agrega una migración (`0036`) que añade columnas `service_id` y `account_id` a `categories` y `source_bill_id` a `transactions` (para idempotencia y trazabilidad). La lógica es un hook en el flujo de pago de facturas existente (`BillService.PayBill`) más un servicio de enlace liviano. Memoria/CPU despreciables (una query por pago). Se siguen los patrones de capas existentes y el patrón de `SetXxxStorage`/setter opcional ya usado en `BillService.SetBillHistoryStorage`.

**Consideraciones de UI obligatorias**: los formularios de categoría (crear/editar) se amplían con selectores de "Servicio" y "Cuenta". Todos los `input`/`select` DEBEN usar tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y el componente custom `Select` (SPEC-004 REQ-024, nunca `<select>` nativo). Verificación en darkmode obligatoria (SPEC-060). No se crean páginas de detalle nuevas (la asociación vive en modales existentes de `BudgetPage.tsx`), por lo que no aplica `BACK_ROUTES` (ver CA-BACK).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Migración `0036` — agregar a `categories` las columnas `service_id INTEGER REFERENCES services(id)` (nullable) y `account_id INTEGER REFERENCES accounts(id)` (nullable), con índice único parcial en `service_id` (solo filas `service_id IS NOT NULL AND deleted_at IS NULL`) para garantizar que un servicio se vincula a una sola categoría activa. Agregar a `transactions` la columna `source_bill_id INTEGER` con índice único parcial (`source_bill_id IS NOT NULL`) para idempotencia de transacciones auto-generadas.
2. **REQ-002**: Modelos — `Category` gana `ServiceID *int64`, `AccountID *int64` y `ServiceName string` (join, solo lectura). `Transaction` gana `SourceBillID *int64`.
3. **REQ-003**: CRUD de categorías (backend) — aceptar `service_id` y `account_id` en `POST/PUT /api/budget/categories[/{id}]`. Validación: si `service_id` viene, `account_id` es obligatorio; si `service_id` viene y ya está en uso por otra categoría activa, rechazar con error claro. `service_id`/`account_id` se pueden limpiar (pasar `null`) al editar. La respuesta incluye `service_name` para la UI.
4. **REQ-004**: UI de categoría (crear y editar, tanto en `BudgetPage.tsx` como en `BudgetConfigModal`) — campos opcionales **"Servicio"** (dropdown custom `Select` con la lista de servicios de `/api/services`) y **"Cuenta"** (dropdown custom `Select` con las cuentas de `/api/budget/accounts`, visible/requerido solo cuando hay servicio seleccionado). Verificación darkmode.
5. **REQ-005**: Hook de pago — al ejecutar `POST /api/bills/{id}/pay` (flujo `BillService.PayBill`), si el servicio de la factura tiene una categoría vinculada activa, crear automáticamente una transacción: `account_id` = `category.account_id`, `category_id` = categoría, `currency_id` = moneda del servicio, `date` = fecha de pago (`paid_at`), `payee` = nombre del servicio, `memo` = referencia de la factura (ej. `Factura #<invoice> — <Mes> <Año>` o periodo), `outflow` = monto de la factura, `inflow` = 0, `cleared` = 1, `source_bill_id` = id de la factura.
6. **REQ-006**: Idempotencia — la transacción auto-generada se registra con `source_bill_id`; el índice único parcial impide duplicados si el mismo pago se procesa dos veces (re-pay, re-import, webhook). El hook verifica si ya existe una transacción con ese `source_bill_id` y no crea otra.
7. **REQ-007**: Indicador visual en la vista mensual — la fila de categoría muestra un badge/ícono con el nombre del servicio vinculado (ej. ícono `webhook`/`service` + nombre truncado) para que el usuario vea la asociación. En el modal de edición de la categoría se muestra el servicio actual y permite cambiar/quitar.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: Flujos secundarios de pago — aplicar el mismo hook cuando una factura se crea o actualiza con `status=paid` (`BillService.Create`/`Update`), cuando se marca pagada vía webhook (`webhookService.UpsertBill`) y vía `documentService.CreateBillFromExtracted`, para que no haya facturas pagadas sin su transacción. La idempotencia por `source_bill_id` cubre la superposición de flujos.
2. **REQ-009**: Desvincular servicio — al desvincular (quitar `service_id`) o archivar la categoría vinculada, las transacciones ya creadas se conservan intactas (historial del presupuesto inalterado). El hook deja de generar nuevas transacciones para ese servicio.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Permitir que un servicio se vincule a varias categorías (quitar la restricción de unicidad) con reparto manual del monto — descartado en v1 por simplicidad y porque la necesidad expresada es 1 servicio → 1 categoría.
2. **REQ-011**: Historial de la transacción en el modal de pago (`PayBillModal`): avisar al usuario que se creará una transacción de presupuesto en la categoría vinculada antes de confirmar el pago.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el hook agrega como máximo 2-3 queries por pago (buscar categoría por servicio + crear transacción). Impacto despreciable en iHost.
- **Seguridad**: los endpoints modificados siguen bajo `authMiddleware` (sesión). Validación de IDs y montos existente se reutiliza. `source_bill_id` no expone datos sensibles.
- **Almacenamiento**: 3 columnas nuevas (1-2 KB total), índices parciales únicos. Sin crecimiento relevante.
- **Disponibilidad**: el hook es síncrono dentro del request de pago; si falla la creación de la transacción, el pago de la factura ya está commiteado y el error se loguea (no bloquea el pago). Ver sección 7.
- **iHost**: sin dependencias nuevas (Go stdlib + SQLite). Migración `0036` en el mecanismo existente. Build multi-arch intacto.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Requerimiento fuente**: solicitud del usuario (sesión 2026-09-27): vincular categoría de presupuesto ↔ servicio del sistema; al pagar una factura del servicio, crear transacción en la categoría. La asociación puede ser al crear la categoría o después.
- **Flujo de pago actual**: `POST /api/bills/{id}/pay` → `api.BillHandlers.PayBill` → `services.BillService.PayBill` (valida `paid_at`, estado, drive URL; persiste pago en `storage.BillStorage.Pay`; registra historial via `recordBillHistory`). Es el punto natural para el hook.
- **Modelo de presupuesto (SPEC-093)**: `categories` (grupo, nombre, ícono, target), `accounts` (con `currency_id`), `transactions` (account, category nullable, currency, date, payee, memo, outflow/inflow, cleared). La transacción auto-generada debe satisfacer este esquema.
- **Moneda**: los servicios tienen `currency_id` (Service.CurrencyID). La factura se paga en la moneda del servicio → la transacción usa `currency_id` del servicio. Las cuentas tienen su propia moneda; se valida de forma blanda que sean coherentes (ver ADR-002).
- **Patrón de inyección opcional**: `BillService.SetBillHistoryStorage` (SPEC-070) es el precedente para inyectar dependencias opcionales en `BillService` sin romper constructores. El hook de presupuesto usará el mismo patrón (`SetBillBudgetLinker`).
- **Patrón de UI**: `BudgetPage.tsx` ya edita categorías con `CategoryFormModal` y `BudgetConfigModal`. El `Select` custom (`frontend/src/components/Select.tsx`) es obligatorio para dropdowns. `api.services.list()` y `api.budget.accounts.list()` ya existen.
- **Migraciones**: última es `0035` (SPEC-093). La nueva es `0036` (`NNNN_descripcion.{up,down}.sql`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Columnas `service_id`+`account_id` en `categories` | Simple, 1 migración, join directo, relación 1:1 clara | Un servicio no puede mapear a varias categorías (limitación aceptada) | ✅ Seleccionada |
| Tabla de enlace `category_service_links` | Permite N:N | Tabla extra, joins adicionales, más complejidad para un caso 1:1 | ❌ Rechazada |
| Hook solo en `PayBill` | Mínimo alcance | Facturas pagadas por webhook/import no generarían transacción | ❌ Rechazada en v1 (se incluye REQ-008 como P1) |
| Transacción sin `source_bill_id` + dedup por (category, memo, date) | Sin columna nueva | Frágil (colisiones legítimas), sin trazabilidad | ❌ Rechazada |
| `source_bill_id` en `transactions` | Idempotencia robusta y trazabilidad | Columna + índice extra | ✅ Seleccionada |
| Cuenta obligatoria al vincular | Evita transacciones sin cuenta | Fricción en el formulario | ✅ Seleccionada (REQ-003 valida `account_id` si hay `service_id`) |
| El hook bloquea el pago si falla la transacción | Consistencia total | Un fallo de presupuesto impediría pagar una factura | ❌ Rechazada (el pago es la fuente de verdad; la transacción se loguea y se puede recrear) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Relación 1:1 servicio → categoría, con columnas directas en `categories`.
- **Contexto**: la necesidad es asociar un servicio con la categoría donde se refleja su gasto. No se pidió reparto entre varias categorías.
- **Decisión**: `categories.service_id` con índice único parcial sobre filas activas. Un servicio activo se vincula a una sola categoría.
- **Consecuencias**: el hook de pago busca una única categoría por `service_id` (sin ambigüedad). Escalable a N:N en el futuro con tabla de enlace (P2).

**ADR-002**: La transacción auto-generada usa la `currency_id` del servicio y la cuenta configurada en el vínculo.
- **Contexto**: la factura se paga en la moneda del servicio; las cuentas pueden tener moneda distinta.
- **Decisión**: `transactions.currency_id` = `services.currency_id`. `account_id` = `categories.account_id`. No se bloquea el vínculo si las monedas difieren (caso borde), pero la UI muestra la moneda de la cuenta para evitar confusiones.
- **Consecuencias**: el monto de la factura se registra en la moneda correcta. El balance de la cuenta puede mezclar monedas solo si el usuario vincula cuentas de moneda distinta (decisión consciente del usuario).

**ADR-003**: Hook de pago como servicio de enlace liviano (`BillBudgetLinker`) inyectado en `BillService` por setter.
- **Contexto**: `BillService` no debe conocer el módulo de presupuesto; `BudgetTransactionService` no debe conocer servicios. Se necesita un punto de integración desacoplado.
- **Decisión**: nueva interfaz en `services` (ej. `BillBudgetLinker`) con método `OnBillPaid(ctx, bill) error`, implementada por un service de enlace (`BudgetBillLinkService`) que orquesta `ServiceStorage` + `CategoryStorage` + `AccountStorage` + `TransactionStorage`. `BillService` la recibe vía `SetBillBudgetLinker` (mismo patrón que `SetBillHistoryStorage`).
- **Consecuencias**: capas limpias; `BillService` queda con un hook opcional que se ignora si no se configura (no rompe tests existentes).

**ADR-004**: El hook no bloquea el pago ante fallo de creación de la transacción.
- **Contexto**: la fuente de verdad del pago es la factura. Un problema en presupuesto no debe impedir registrar el pago.
- **Decisión**: el hook se ejecuta después de persistir el pago y del historial; si falla, se loguea con `slog.Error` y el request responde OK. La idempotencia por `source_bill_id` permite reintentar/recrear la transacción sin duplicados.
- **Consecuencias**: posible desincronización temporal pago→transacción que se resuelve reintentando el pago (idempotente) o creando la transacción manualmente.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Frontend BudgetPage / BudgetConfigModal]
        │  create/update category { service_id, account_id }
        ▼
[api.BudgetHandlers.Create/UpdateCategory] --[validar]--> [services.CategoryService]
        │
        ▼
[SQLite: categories (+service_id, +account_id)]

[Frontend BillsPage -> PayBillModal]
        │  POST /api/bills/{id}/pay { paid_at }
        ▼
[api.BillHandlers.PayBill] --> [services.BillService.PayBill]
        │                      └─ storage.Pay() (marca pagada + historial)
        │                      └─ SetBillBudgetLinker.OnBillPaid(bill)
        ▼
[services.BudgetBillLinkService]
        ├─ storage.ServiceStorage.GetByID(service_id)   (moneda, nombre)
        ├─ storage.CategoryStorage.GetByServiceID(service_id) (categoría activa)
        ├─ verificar idempotencia por source_bill_id
        └─ storage.TransactionStorage.Create({source_bill_id, ...})
```

### 4.2 Componentes

#### 4.2.1 Backend

- **`internal/models/category.go`**: agregar `ServiceID *int64`, `AccountID *int64`, `ServiceName string` (json `service_id,omitempty`, `account_id,omitempty`, `service_name,omitempty`).
- **`internal/models/transaction.go`**: agregar `SourceBillID *int64` (json `source_bill_id,omitempty`).
- **`internal/storage/category.go`**: actualizar `categoryColumns`, `Create`, `Update`, `scanCategory`/`scanCategories` para las 2 columnas nuevas (con `COALESCE`/LEFT JOIN para `service_name`); nuevo método `GetByServiceID(ctx, serviceID)` (devuelve la categoría activa vinculada, `service_id = ? AND deleted_at IS NULL`); nuevo método `ServiceLinkInUse(ctx, serviceID, excludeCategoryID)` para la validación de unicidad.
- **`internal/storage/transaction.go`**: incluir `source_bill_id` en `Create`, `Update`, columnas y scanners; nuevo método `GetBySourceBill(ctx, billID)` (devuelve la transacción con ese `source_bill_id`, si existe).
- **`internal/services/bill.go`**: agregar campo `budgetLinker BillBudgetLinker` y setter `SetBillBudgetLinker`; en `PayBill`, tras persistir el pago y el historial, invocar `s.budgetLinker.OnBillPaid(ctx, paid)` (si el linker está configurado); loguear error sin romper el request.
- **`internal/services/budget_bill_link.go`** (nuevo): interfaz `BillBudgetLinker { OnBillPaid(ctx context.Context, bill *models.Bill) error }` e implementación `BudgetBillLinkService` con los storages necesarios. Lógica: buscar servicio → buscar categoría vinculada → si no hay, no-op → verificar `GetBySourceBill` → si existe, no-op → crear transacción con los valores de REQ-005.
- **`internal/services/category.go`**: `validate` acepta `service_id`/`account_id`; si `service_id != nil` y `account_id == nil` → error; si `service_id != nil` → verificar que el servicio exista (`ServiceStorage` inyectado) y que no esté en uso por otra categoría activa. Requiere inyectar `ServiceStorage` a `CategoryService`.
- **`internal/api/budget_handlers.go`**: `categoryRequest` gana `ServiceID *int64` y `AccountID *int64`; pasarlos al modelo.
- **`internal/api/routes.go`**: sin cambios de rutas.
- **`cmd/server/main.go`**: crear `BudgetBillLinkService` con los storages, inyectarlo en `billService.SetBillBudgetLinker(...)`, e inyectar `ServiceStorage` en `CategoryService`.

#### 4.2.2 Frontend

- **`frontend/src/types/index.ts`**: `BudgetCategory` gana `service_id?: number | null`, `account_id?: number | null`, `service_name?: string`.
- **`frontend/src/api/index.ts`**: `budget.categories.create/update` aceptan `service_id` y `account_id` en el body.
- **`frontend/src/pages/BudgetPage.tsx`**:
  - `CategoryFormModal` (crear y editar): agregar selector **"Servicio"** (custom `Select`, opciones de `api.services.list()`, con opción "Sin servicio" = `null`) y selector **"Cuenta"** (custom `Select`, opciones de `api.budget.accounts.list()`, visible/requerido cuando hay servicio). Estado del form extendido con `serviceId?: number`, `accountId?: number`.
  - `BudgetConfigModal`: mismos campos en su `CategoryFormModal` interno.
  - Fila de categoría en la vista mensual: badge con el nombre del servicio vinculado (`service_name`).
  - Cargar servicios y cuentas una vez (estado local o store) para poblar los selects.
- **`frontend/public/i18n/{es,en}.json`**: claves bajo `budget`: `link_service`, `link_service_hint`, `no_service`, `link_account`, `link_account_required`, `service_badge` (o reutilizar existentes). Luego `npm run build`.

### 4.3 Modelo de datos

Migración `0036_add_category_service_link.{up,down}.sql`:

```sql
-- 0036_add_category_service_link.up.sql
ALTER TABLE categories ADD COLUMN service_id INTEGER REFERENCES services(id);
ALTER TABLE categories ADD COLUMN account_id INTEGER REFERENCES accounts(id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_service_link
  ON categories(service_id) WHERE service_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_categories_account_link
  ON categories(account_id) WHERE account_id IS NOT NULL;

ALTER TABLE transactions ADD COLUMN source_bill_id INTEGER;

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_source_bill
  ON transactions(source_bill_id) WHERE source_bill_id IS NOT NULL;
```

```sql
-- 0036_add_category_service_link.down.sql
DROP INDEX IF EXISTS idx_transactions_source_bill;
ALTER TABLE transactions DROP COLUMN source_bill_id;
DROP INDEX IF EXISTS idx_categories_account_link;
DROP INDEX IF EXISTS idx_categories_service_link;
ALTER TABLE categories DROP COLUMN account_id;
ALTER TABLE categories DROP COLUMN service_id;
```

**Notas**:
- Los índices parciales únicos garantizan: (a) un servicio activo solo se vincula a una categoría; (b) una factura solo genera una transacción.
- `ALTER TABLE ... DROP COLUMN` requiere SQLite ≥ 3.35; `modernc.org/sqlite` incluye versiones recientes (se verifica en local). Si el driver usado no soporta DROP COLUMN, el `.down.sql` se ajusta a recreación de tabla (se documenta en desarrollo).
- `Category.ServiceName` viene de un `LEFT JOIN services s ON s.id = categories.service_id` en las queries de listado/get; nunca se persiste.

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/budget/categories`

**Request** (nuevos campos):
```json
{
  "category_group_id": 1,
  "name": "Internet",
  "icon": "wifi",
  "target_amount": null,
  "service_id": 7,
  "account_id": 3
}
```

**Response 200** (nuevos campos):
```json
{
  "id": 12,
  "category_group_id": 1,
  "name": "Internet",
  "icon": "wifi",
  "sort_order": 0,
  "target_amount": null,
  "service_id": 7,
  "account_id": 3,
  "service_name": "Claro Internet"
}
```

**Response Error**:
```json
{ "error": "invalid_request", "message": "Si vincula un servicio debe seleccionar una cuenta" }
{ "error": "invalid_request", "message": "El servicio ya está vinculado a otra categoría" }
```

#### Endpoint: `PUT /api/budget/categories/{id}`

Igual que POST; `service_id: null` y/o `account_id: null` limpian el vínculo.

#### Endpoint: `POST /api/bills/{id}/pay` (sin cambios de contrato)

El hook se ejecuta de forma transparente. `source_bill_id` y los demás campos de la transacción quedan visibles en `GET /api/budget/transactions`.

### 4.5 Dependencias

- **Internas**: `internal/api/budget_handlers.go`, `internal/api/routes.go` (sin cambios de ruta), `internal/services/bill.go`, `internal/services/category.go`, `internal/services/budget_bill_link.go` (nuevo), `internal/storage/category.go`, `internal/storage/transaction.go`, `cmd/server/main.go`, `frontend/src/pages/BudgetPage.tsx`, `frontend/src/api/index.ts`, `frontend/src/types/index.ts`, `frontend/public/i18n/{es,en}.json`, `frontend/src/components/Select.tsx` (existente).
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Al crear una categoría con un servicio vinculado y su cuenta, la respuesta incluye `service_id`, `account_id` y `service_name`; la categoría aparece en la vista mensual con el badge del servicio.
- [x] CA-002: Al pagar una factura de un servicio vinculado (`POST /api/bills/{id}/pay`), se crea automáticamente una transacción en la categoría: `outflow` = monto de la factura, `currency_id` = moneda del servicio, `date` = `paid_at`, `payee` = nombre del servicio, `cleared` = true, `source_bill_id` = id de la factura.
- [x] CA-003: La transacción auto-generada aparece en la grilla de transacciones del mes (`/budget/transacciones`) y actualiza `activity`/`available` de la categoría en la vista mensual.
- [x] CA-004: Pagar la misma factura dos veces (o re-procesar el pago) no crea transacciones duplicadas (idempotencia por `source_bill_id`).
- [x] CA-005: Al editar una categoría se puede cambiar el servicio/la cuenta o desvincular (pasando `null`); las transacciones previas se conservan.
- [x] CA-006: No se puede vincular el mismo servicio a dos categorías activas (backend rechaza con error claro).
- [x] CA-007: Si se envía `service_id` sin `account_id`, el backend rechaza con error claro.
- [x] CA-008: Un servicio sin categoría vinculada sigue pagándose normalmente (el hook es no-op).
- [x] CA-009: Si el hook falla, el pago de la factura se completa igual (estado `paid` persistido) y el error queda en el log.
- [x] CA-DARK: Los selects y campos nuevos de los formularios de categoría usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (SPEC-060).
- [x] CA-SELECT: Los dropdowns de "Servicio" y "Cuenta" usan el componente custom `Select` de `frontend/src/components/Select.tsx`, nunca un `<select>` nativo (SPEC-004 REQ-024).
- [x] CA-BACK: No aplica (la asociación vive en modales existentes; no hay páginas de detalle/formularios nuevos con lista padre). Verificar que no se introducen links "← Título".

### 5.2 No funcionales

- [x] CA-NF-001: Migración `0036` aplica y revierte limpiamente (`up` y `down`), con índices parciales creados.
- [x] CA-NF-002: Sin dependencias externas nuevas; build multi-arch (`linux/amd64,linux/arm/v7,linux/arm64`) intacto.
- [x] CA-NF-003: i18n `budget.link_*` presente en `es.json` y `en.json` (fuente de verdad `frontend/public/i18n/`) y servido tras `npm run build`.

### 5.3 Testing

- **Unit tests**: `BudgetBillLinkService.OnBillPaid` (con/sin categoría vinculada, idempotencia, cuenta faltante, servicio sin categoría); validación de `CategoryService` (service+account, unicidad, limpiar vínculo); migración `0036` up/down.
- **Integration tests**: flujo `POST /api/bills/{id}/pay` → transacción creada con los valores correctos; re-pay idempotente; editar categoría para desvincular conserva transacciones.
- **E2E tests**: en local: crear categoría con servicio+cuenta → pagar factura del servicio → ver transacción en grilla y `activity` actualizado; desvincular → pagar otra factura → sin transacción nueva.
- **Carga/Performance**: el hook agrega ≤ 3 queries por pago; medir en local con DB de prueba (despreciable).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0036` up/down + modelos (`Category`, `Transaction`) | 0.5 día | Ninguna |
| 2 | Storage: `category.go` (columnas, join service_name, `GetByServiceID`, `ServiceLinkInUse`), `transaction.go` (`source_bill_id`, `GetBySourceBill`) | 1 día | Fase 1 |
| 3 | Services: `budget_bill_link.go` (interfaz + implementación), hook en `BillService.PayBill` (`SetBillBudgetLinker`), validación en `CategoryService` (+ inyección `ServiceStorage`) | 1 día | Fase 2 |
| 4 | API handlers (categoryRequest service/account) + wiring en `main.go` | 0.5 día | Fase 3 |
| 5 | Frontend: tipos, api, `CategoryFormModal`/`BudgetConfigModal` (selects Servicio/Cuenta), badge en vista mensual, i18n, `npm run build` | 1-2 días | Fase 4 |
| 6 | Tests (unit + integration), verificación darkmode y multi-arch, validación manual en local | 1 día | Fases 1-5 |

### 6.2 Milestones

1. **MVP (Fases 1-4)**: backend completo (migración, hook de pago, CRUD con validación). Primer PR ejecutable.
2. **V1.0 (Fases 5-6)**: UI de vinculación + badge + tests + polished.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Fallo del hook deja factura pagada sin transacción | Media | Medio | Pago ya commiteado; error logueado; reintento idempotente por `source_bill_id`; opción de crear transacción manual en UI de transacciones |
| `ALTER TABLE DROP COLUMN` no soportado por el driver SQLite en down | Baja | Bajo | Verificar versión de `modernc.org/sqlite` en local; si no soporta, el `.down.sql` recrea la tabla (documentado) |
| Servicio vinculado a dos categorías por edición concurrente | Baja | Bajo | Índice único parcial + validación en `CategoryService`; SQLite serializa escrituras (WAL) |
| Usuario vincula cuenta en moneda distinta a la del servicio | Media | Bajo | La transacción usa moneda del servicio; la UI muestra la moneda de la cuenta al vincular |
| Selects sin `Select` custom rompen el estándar UI | Baja | Medio | CA-SELECT explícito; revisión en code review |

## 8. Notas y Referencias

- SPEC-093 (módulo Presupuesto, modelo `categories`/`transactions`), SPEC-043 (acción Pagar factura), SPEC-070 (patrón `SetBillHistoryStorage`), SPEC-004 REQ-024 (dropdowns custom `Select`), SPEC-060 (darkmode inputs), SPEC-066 (worktrees).
- Archivos de referencia: `internal/services/bill.go` (PayBill), `internal/storage/category.go`, `internal/storage/transaction.go`, `frontend/src/pages/BudgetPage.tsx`, `frontend/src/api/index.ts`, `migrations/0035_create_budget_core.*`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-27 | opencode | Creación inicial de la especificación (requerimiento relevado con usuario: vincular categoría de presupuesto ↔ servicio del sistema; transacción automática al pagar factura; asociación durante creación o posterior) |