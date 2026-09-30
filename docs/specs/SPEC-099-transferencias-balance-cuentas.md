---
title: "Transferencias entre cuentas y balance por cuenta en el presupuesto"
id: "SPEC-099"
status: "released"
author: "opencode"
created: "2026-09-30"
updated: "2026-09-30"
github_issue: 102
---

# Transferencias entre cuentas y balance por cuenta en el presupuesto

**ID**: SPEC-099  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-30  
**Actualizado**: 2026-09-30

---

## 1. Resumen Ejecutivo

El módulo de presupuesto (SPEC-093) modela cuentas (`accounts`, tipos `checking|savings|credit_card|cash`) y movimientos (`transactions` con `outflow`/`inflow`), pero **no existe el concepto de transferencia entre cuentas ni el balance calculado por cuenta**. Hoy el `GET /api/budget/accounts` solo devuelve `starting_balance`; mover dinero de una cuenta a otra no tiene representación limpia: registrar una salida en una cuenta y una entrada en otra como dos transacciones contaminaría el presupuesto (la entrada se sumaría como ingreso en `IncomeByMonthGrouped` y la salida como actividad en `ActivityByCategoryGrouped`).

Este problema bloquea además el soporte de **tarjetas de crédito** (tipo ya existente en `accounts`): una tarjeta de crédito es una cuenta de pasivo cuyo saldo crece al gastar y se "paga" con una transferencia desde la cuenta corriente. Sin transferencias no se puede representar el pago de la tarjeta sin duplicar el gasto en el presupuesto.

Esta spec agrega: (1) una tabla `transfers` que modela movimientos entre dos cuentas (origen → destino) sin afectar la actividad ni los ingresos del presupuesto, y (2) el **balance calculado por cuenta** (`starting_balance` + neto de transacciones + neto de transferencias), expuesto en la API y mostrado en la UI. Para tarjetas de crédito, el balance se muestra como deuda (negativo) conforme al tipo de cuenta.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Tabla `transfers` en SQLite (migración `0038`) con `from_account_id`, `to_account_id`, `currency_id`, `date`, `payee`, `memo`, `amount`, `cleared`, timestamps. Validación: cuentas origen y destino distintas, ambas activas, `amount > 0`, misma moneda (P0; monedas distintas con tipo de cambio es P1).
2. **REQ-002**: Las transferencias **NO** deben afectar la actividad por categoría (`ActivityByCategoryGrouped`) ni los ingresos por moneda (`IncomeByMonthGrouped`) del presupuesto. Son movimientos neutros entre cuentas.
3. **REQ-003**: Las transferencias **SÍ** deben afectar el balance de ambas cuentas: restar del origen y sumar al destino.
4. **REQ-004**: `GET /api/budget/accounts` devuelve cada cuenta con su `balance` calculado: `starting_balance` + Σ(inflow − outflow) de `transactions` activas + neto de `transfers`. Para cuentas `credit_card`, el balance representa la deuda (crece al gastar, se reduce al transferir el pago).
5. **REQ-005**: CRUD de transferencias en backend: `GET /api/budget/transfers` (con filtros `year`/`month`), `POST /api/budget/transfers`, `PUT /api/budget/transfers/{id}`, `DELETE /api/budget/transfers/{id}` (soft delete).
6. **REQ-006**: UI de transferencias en el frontend: modal/formulario para crear y editar transferencias (cuenta origen, cuenta destino, monto, moneda, fecha, payee, memo, cleared) y listado del mes dentro de la página de transacciones (`BudgetTransactionsPage`), siguiendo el patrón de cards/tabla existente.
7. **REQ-007**: El modal de cuentas (`AccountsModal` en `BudgetTransactionsPage`) muestra el `balance` calculado por cuenta (formateado con `formatMoney`), en lugar de solo el tipo/moneda. Para `credit_card`, el balance se muestra como deuda.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: `POST /api/budget/transfers` permite monedas distintas entre origen y destino con `exchange_rate` (se registra el monto convertido en la moneda de destino o un monto por cuenta). Requiere decisión de modelado (ver ADR-003).
2. **REQ-009**: El dashboard de presupuesto (`BudgetPage`) puede mostrar un resumen de balances por cuenta (cards con balance por cuenta y moneda).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Bot de Telegram: comando para registrar una transferencia (espejo de `/budget_transaction`, SPEC-097).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El cálculo de balance debe ser una query agregada eficiente (una por `GET /api/budget/accounts`); el volumen de datos es bajo (SQLite local).
- **Seguridad**: Endpoints autenticados (`authMiddleware`), consistente con el resto de rutas del presupuesto.
- **Almacenamiento**: Una tabla nueva (~100 bytes/fila); impacto nulo en iHost.
- **Disponibilidad**: Sin procesos extra; misma arquitectura de storage/services/api.
- **iHost**: SQLite con WAL existente; migración aditiva sin reescritura de tablas. Bajo consumo de RAM.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Modelo actual (SPEC-093)**: `transactions` (account_id, category_id, currency_id, date, payee, memo, outflow, inflow, cleared, source_bill_id). `accounts` con tipo `credit_card` ya soportado (migración `0035`).
- **Actividad e ingresos**: `ActivityByCategoryGrouped` (storage/transaction.go:142) suma `SUM(outflow − inflow)` agrupado por `category_id`; `IncomeByMonthGrouped` (storage/transaction.go:175) suma `SUM(inflow)`. Cualquier entrada con `inflow` contamina el total de ingresos; cualquier salida con `outflow` contamina la actividad si tiene categoría.
- **Tarjetas de crédito**: el análisis previo (no especificado) concluyó que la tarjeta debe vivir dentro del presupuesto como cuenta de pasivo, con dos momentos: (1) gasto = transacción con outflow categorizada desde la cuenta tarjeta; (2) pago = **transferencia** cuenta corriente → tarjeta, que NO es gasto ni ingreso.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **A: Tabla `transfers` separada** | Registro único y atómico de la transferencia; no toca `transactions` ni contamina actividad/ingresos; balance por cuenta trivial (restar/sumar según lado); soft delete simple | Tabla nueva; el listado del mes debe unir `transactions` + `transfers` en la vista | ✅ Seleccionada |
| B: Dos `transactions` enlazadas con `transfer_id` | Reutiliza la tabla existente | Contamina `IncomeByMonthGrouped`/`ActivityByCategoryGrouped` salvo filtrar por `transfer_id` en cada query (más superficie de cambio); riesgo de doble conteo en vistas existentes; complejidad de mantenimiento | ❌ Rechazada |
| C: Columnas `from_account_id`/`to_account_id` en `transactions` | Sin tabla nueva | Rompe el modelo outflow/inflow existente y todas las validaciones; migración compleja sobre tabla en producción | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Tabla `transfers` separada, no dos transacciones**
- **Contexto**: Las transferencias son movimientos neutros (no categorizables) que NO deben aparecer como ingreso ni como actividad del presupuesto.
- **Decisión**: Crear tabla `transfers` con `from_account_id`, `to_account_id`, `amount`, `currency_id`, `date`, `payee`, `memo`, `cleared`. No se usa categoría (siempre neutra).
- **Consecuencias**: El balance por cuenta se calcula como `starting_balance + transacciones + neto_transferencias`. El listado mensual de la página de transacciones une ambas fuentes para mostrar todo junto, pero las consultas agregadas del presupuesto (actividad/ingresos) solo leen `transactions`, por lo que quedan intactas sin cambios.

**ADR-002: Balance calculado en tiempo real, no persistido**
- **Contexto**: Persistir un `balance` en `accounts` introduce problemas de consistencia al crear/editar/borrar transacciones y transferencias (hay que recomputar).
- **Decisión**: El balance se calcula en la consulta (`GET /api/budget/accounts`) con un `LEFT JOIN`/subquery agregada: `starting_balance + Σ(inflow−outflow) transactions + Σ(to−from) transfers`, filtrando `deleted_at IS NULL`. Se agrega el campo `balance` al modelo `Account` (solo respuesta, no columna).
- **Consecuencias**: La respuesta de listar cuentas es levemente más costosa (una agregación), pero el volumen es bajo. No hay riesgo de balance desincronizado.

**ADR-003: P0 moneda igual en ambos lados; P1 monedas distintas**
- **Contexto**: Una transferencia puede ser entre cuentas de distinta moneda (ej. C$ → US$), lo que requiere un tipo de cambio.
- **Decisión**: En P0, ambas cuentas deben compartir `currency_id` (validación en service). En P1 se agrega `exchange_rate` y la semántica de "monto en la moneda de origen convertido a la de destino".
- **Consecuencias**: P0 cubre el caso de pago de tarjeta (misma moneda típicamente). El P1 queda como mejora sin romper el modelo.

**ADR-004: UI — cuenta origen sin tarjetas de crédito, balance con signo**
- **Contexto**: En el modal de transferencia, la cuenta origen por defecto excluye las `credit_card` (no se "saca" dinero de una tarjeta hacia otra cuenta; el flujo normal es gasto + transferencia de pago desde una cuenta de fondos).
- **Decisión**: El `Select` de origen lista solo `checking|savings|cash` (con fallback a todas si no hay fondos); el de destino lista todas. El balance de `credit_card` se muestra en rojo cuando es negativo (deuda).
- **Consecuencias**: UX clara para el caso de pago de tarjeta; el destino permite tarjeta (pago) y el origen es siempre una cuenta de fondos.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[BudgetHandlers] --(accounts + balance, transfers CRUD)--> [BudgetService/TransferService]
      |                                                            |
      v                                                            v
[storage.AccountStorage: List con balance]              [storage.TransferStorage: CRUD]
      |                                                            |
      v                                                            v
                          [SQLite: accounts + transactions + transfers]
```

### 4.2 Componentes

#### 4.2.1 `TransferStorage` (nuevo)
- **Responsabilidad**: CRUD de transferencias en SQLite + agregaciones de neto por cuenta para balance.
- **Interfaz**: `ListByMonth(ctx, year, month)`, `GetByID(ctx, id)`, `Create(ctx, t)`, `Update(ctx, t)`, `SoftDelete(ctx, id)`, `NetByAccount(ctx) (map[int64]float64, error)`.
- **Dependencias**: `*sql.DB`.
- **Ubicación**: `internal/storage/transfer.go`.

#### 4.2.2 `TransferService` (nuevo)
- **Responsabilidad**: Validación de negocio: cuentas distintas y activas, monedas iguales (P0), `amount > 0`, fecha válida.
- **Interfaz**: `ListByMonth`, `GetByID`, `Create`, `Update`, `Delete`.
- **Dependencias**: `TransferStorage`, `AccountStorage`, `CurrencyStorage`.
- **Ubicación**: `internal/services/transfer.go`.

#### 4.2.3 Cambios en `AccountStorage`
- **Responsabilidad**: Agregar `balance` calculado al listar/obtener cuentas.
- **Interfaz**: `List(ctx)` y `GetByID(ctx, id)` ahora devuelven `models.Account` con `Balance`.
- **Ubicación**: `internal/storage/account.go`.

#### 4.2.4 Cambios en `BudgetHandlers`
- **Responsabilidad**: Exponer endpoints de transferencias y balance.
- **Rutas nuevas** (internal/api/routes.go):
  - `GET /api/budget/transfers`
  - `POST /api/budget/transfers`
  - `PUT /api/budget/transfers/{id}`
  - `DELETE /api/budget/transfers/{id}`
- **Ubicación**: `internal/api/budget_handlers.go`.

#### 4.2.5 Frontend (`BudgetTransactionsPage.tsx` + componentes)
- **Responsabilidad**: Modal de transferencia (crear/editar) + mostrar balance en `AccountsModal` + listado de transferencias del mes.
- **Componentes nuevos**: `TransferFormModal`; adaptación de `AccountsModal` para mostrar balance.
- **Ubicación**: `frontend/src/pages/BudgetTransactionsPage.tsx`.
- **i18n**: claves nuevas en `frontend/public/i18n/{es,en}.json` (fuente de verdad), luego `npm run build` en `frontend/`.

### 4.3 Modelo de datos

Migración `0038_create_transfers.up.sql`:

```
Entidad: transfers
- id: INTEGER PK AUTOINCREMENT
- from_account_id: INTEGER NOT NULL (FK accounts.id)
- to_account_id: INTEGER NOT NULL (FK accounts.id)
- currency_id: INTEGER NOT NULL (FK currencies.id)
- date: TEXT NOT NULL (YYYY-MM-DD)
- payee: TEXT
- memo: TEXT
- amount: REAL NOT NULL DEFAULT 0 (> 0)
- cleared: INTEGER NOT NULL DEFAULT 0
- deleted_at: DATETIME
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- Relaciones: accounts (N:1) desde/to; currencies (N:1)
```

Migración `0038_create_transfers.down.sql`: `DROP TABLE IF EXISTS transfers;`

Modelo Go `internal/models/transfer.go`:

```
type Transfer struct {
    ID           int64    `json:"id"`
    FromAccountID int64   `json:"from_account_id"`
    FromAccountName string `json:"from_account_name,omitempty"`
    ToAccountID   int64   `json:"to_account_id"`
    ToAccountName string `json:"to_account_name,omitempty"`
    CurrencyID   int64    `json:"currency_id"`
    CurrencyCode string   `json:"currency_code,omitempty"`
    Date         string   `json:"date"`
    Payee        string   `json:"payee"`
    Memo         string   `json:"memo"`
    Amount       float64  `json:"amount"`
    Cleared      bool     `json:"cleared"`
    DeletedAt    *time.Time `json:"deleted_at,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

Cambio en `internal/models/account.go`: agregar `Balance float64 \`json:"balance"\``.

Cálculo de balance (consulta en `AccountStorage.List`):

```sql
SELECT a.id, a.name, a.type, a.currency_id, COALESCE(c.code,''),
       a.starting_balance, a.deleted_at, a.created_at, a.updated_at,
       a.starting_balance
         + COALESCE((SELECT SUM(t.inflow - t.outflow)
                     FROM transactions t
                     WHERE t.account_id = a.id AND t.deleted_at IS NULL), 0)
         + COALESCE((SELECT SUM(CASE WHEN tr.to_account_id = a.id THEN tr.amount
                                     WHEN tr.from_account_id = a.id THEN -tr.amount END)
                     FROM transfers tr
                     WHERE (tr.from_account_id = a.id OR tr.to_account_id = a.id)
                       AND tr.deleted_at IS NULL), 0) AS balance
FROM accounts a
LEFT JOIN currencies c ON c.id = a.currency_id
WHERE a.deleted_at IS NULL
ORDER BY a.name
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/budget/transfers?year=&month=`

**Request**: query `year`/`month` opcionales (si faltan, lista todas).

**Response 200**:
```json
[
  {
    "id": 1,
    "from_account_id": 2,
    "from_account_name": "Banco X",
    "to_account_id": 3,
    "to_account_name": "Tarjeta Y",
    "currency_id": 1,
    "currency_code": "NIO",
    "date": "2026-10-15",
    "payee": "",
    "memo": "Pago tarjeta",
    "amount": 1666.67,
    "cleared": true
  }
]
```

#### Endpoint: `POST /api/budget/transfers`

**Request**:
```json
{
  "from_account_id": 2,
  "to_account_id": 3,
  "currency_id": 1,
  "date": "2026-10-15",
  "payee": "",
  "memo": "Pago tarjeta",
  "amount": 1666.67,
  "cleared": true
}
```

**Response 201**: objeto `Transfer` creado.

**Response Error 400**:
```json
{ "error": "invalid_request", "message": "la cuenta de origen y destino deben ser distintas" }
```

#### Endpoint: `PUT /api/budget/transfers/{id}`

**Request**: mismo body que POST. **Response 200**: objeto `Transfer` actualizado. **Error 404** si no existe.

#### Endpoint: `DELETE /api/budget/transfers/{id}`

**Response 200**: `{ "message": "Transferencia eliminada" }` (soft delete).

#### Endpoint: `GET /api/budget/accounts` (modificado)

**Response 200** (agrega `balance`):
```json
[
  {
    "id": 2,
    "name": "Banco X",
    "type": "checking",
    "currency_id": 1,
    "currency_code": "NIO",
    "starting_balance": 10000,
    "balance": 8333.33
  }
]
```

### 4.5 Dependencias

- **Internas**: `internal/storage/` (account, transaction, transfer), `internal/services/` (account, transfer), `internal/api/budget_handlers.go`, `internal/api/routes.go`, `frontend/src/pages/BudgetTransactionsPage.tsx`, `frontend/src/api/index.ts`, `frontend/src/types/index.ts`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado dos cuentas activas de la misma moneda, cuando se crea una transferencia de $X de A→B vía `POST /api/budget/transfers`, entonces `GET /api/budget/accounts` refleja balance de A reducido en $X y balance de B aumentado en $X, y las agregaciones de actividad e ingresos del presupuesto NO cambian.
- [x] CA-002: Dado un intento de transferencia de una cuenta a sí misma, o con `amount <= 0`, o a/desde una cuenta inexistente o eliminada, entonces la API responde 400 con mensaje claro y no crea el registro.
- [x] CA-003: Dado `GET /api/budget/transfers?year=2026&month=10`, entonces devuelve solo las transferencias de ese mes (filtro por `date`), sin eliminadas.
- [x] CA-004: Dado `PUT /api/budget/transfers/{id}` con monto/cuentas modificados, entonces el balance por cuenta se recalcula correctamente al consultar `GET /api/budget/accounts`.
- [x] CA-005: Dado `DELETE /api/budget/transfers/{id}`, entonces la transferencia queda soft-deleted y deja de afectar el balance y el listado.
- [x] CA-006: Dado una cuenta de tipo `credit_card` con gastos (transacciones outflow) y un pago (transferencia desde checking), entonces su `balance` refleja la deuda restante (negativo si debe más de lo pagado) y el pago NO se cuenta como gasto del presupuesto.
- [x] CA-007: El frontend permite crear, editar y eliminar transferencias desde `BudgetTransactionsPage`, y el `AccountsModal` muestra el balance calculado por cuenta.
- [x] CA-DARK: Los inputs/selects de los nuevos modales usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verifica legibilidad en darkmode (texto y placeholders).

### 5.2 No funcionales

- [x] CA-NF-001: `GET /api/budget/accounts` responde en < 500ms en la DB local de prueba con datos de SPEC-093/094 (decenas de transacciones).

### 5.3 Testing

- **Unit tests**: `TransferService.validate` (cuentas iguales, amount <= 0, moneda distinta P0, fecha inválida); cálculo de balance (account + transaction + transfer) con datos de ejemplo.
- **Integration tests**: flujo completo crear transferencia → listar → balance actualizado → editar → eliminar (sobre `/tmp/test-app.db`, NUNCA `data/app.db`).
- **E2E tests**: alta de transferencia y visualización de balance en la UI (manual local con server en `:8088`).
- **Carga/Performance**: N/A (volumen bajo).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0038` + modelo `Transfer` + `TransferStorage` (CRUD + NetByAccount) | 1 día | Ninguna |
| 2 | `TransferService` con validaciones + cálculo de balance en `AccountStorage` | 1 día | Fase 1 |
| 3 | Handlers y rutas (`GET/POST/PUT/DELETE /api/budget/transfers`, accounts con balance) | 0.5 día | Fase 2 |
| 4 | Frontend: modal de transferencia + balance en AccountsModal + listado del mes + i18n + build | 1.5 días | Fase 3 |
| 5 | Tests (unit + integration), validación manual local con server | 1 día | Fase 4 |

### 6.2 Milestones

1. **MVP**: Backend completo (transfers CRUD + balance) con tests unitarios.
2. **V1.0**: Frontend completo (modal de transferencia + balance visible) y verificación manual en local.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El listado del mes mezcla `transactions` y `transfers` y se vuelve confuso | Media | Medio | Mostrar las transferencias con un indicador visual (ícono/etiqueta "Transferencia") y payee con "A → B" |
| Contaminación accidental de actividad/ingresos si una transferencia se registra como transacción | Media | Alto | Modelo separado (`transfers`) + tests que verifican que las agregaciones no cambian |
| Balance de tarjeta de crédito con signo ambiguo (negativo vs positivo) | Media | Bajo | Definir y documentar: en `credit_card`, balance negativo = deuda; la UI lo muestra con formato explícito |
| Duplicar transferencia por doble submit | Baja | Bajo | Validación en service + toast de error si el POST falla |
| Editar i18n en `public/i18n/` en vez de `frontend/public/i18n/` (pierde en build) | Baja | Alto | Regla AGENTS.md: editar siempre `frontend/public/i18n/` y correr `npm run build` (precedente SPEC-032/033) |

## 8. Notas y Referencias

- SPEC-093: módulo de presupuesto (accounts/transactions/assignments).
- SPEC-094: puente factura → transacción (patrón de idempotencia con `source_bill_id`, a replicar conceptualmente en CRUD).
- SPEC-097: `/budget_transaction` del bot Telegram (referencia para REQ-010).
- `internal/storage/transaction.go`: `ActivityByCategoryGrouped` / `IncomeByMonthGrouped` (queries a NO tocar).
- `frontend/src/pages/BudgetTransactionsPage.tsx`: UI actual de transacciones y cuentas.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-30 | opencode | Creación inicial de la especificación. Transferencias como tabla `transfers` separada (no contamina actividad/ingresos) + balance calculado por cuenta en `GET /api/budget/accounts`. P0: misma moneda; P1: tipo de cambio. Habilitador del soporte de tarjetas de crédito como cuentas de pasivo. |
| 2026-09-30 | opencode | Inicio de desarrollo (estado → in_progress). Backend: migración 0038, modelo `Transfer`, `TransferStorage` (CRUD + `NetByAccount`), `TransferService` con validaciones (cuentas distintas/activas, misma moneda, amount>0, fecha), balance calculado en `AccountStorage`, handlers y rutas `/api/budget/transfers`. Frontend: modal de transferencia, balance en `AccountsModal`, listado del mes, i18n. Tests: API (`TestTransferCRUDAndBalance`, `TestTransferValidation`) y migración (`TestMigration0038UpDown`). ADR-004 (UI: origen sin tarjetas, balance con signo). |
| 2026-09-30 | opencode | Validación manual local satisfactoria (usuario confirmó tras probar con datos mock). Criterios de aceptación marcados como pass. Release: merge `feature/SPEC-099` → `main` (commit `04de62f`), push a `origin/main`. Issue #102 cerrado con label `spec/released`. |