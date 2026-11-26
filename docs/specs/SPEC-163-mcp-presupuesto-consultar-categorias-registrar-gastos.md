---
title: "MCP: consultar categorías del presupuesto y registrar gastos (transacciones)"
id: "SPEC-163"
status: "pending_release"
author: "opencode"
created: "2026-11-25"
updated: "2026-11-25"
github_issue: 165
---

# MCP: consultar categorías del presupuesto y registrar gastos (transacciones)

**ID**: SPEC-163  
**Estado**: pending_release
**Autor**: opencode  
**Creado**: 2026-11-25  
**Actualizado**: 2026-11-25

---

## 1. Resumen Ejecutivo

El usuario registra muchos gastos del día a día (compras por pedido) y quiere que un **LLM externo** (por ejemplo Claude Code) pueda anotarlos por él. El flujo que describió: le envía al LLM un PDF de factura, el LLM extrae los datos y llama a una tool para registrar el movimiento en la categoría correspondiente (ej. «comida»). Para que el LLM no pregunte al usuario toda la información cada vez, necesita **consultar las categorías del presupuesto** y hacer la asociación.

El módulo de presupuesto estilo YNAB (SPEC-093) ya persiste categorías, cuentas, meses y transacciones, y ya expone los endpoints HTTP necesarios (`/api/budget/...`). El servidor MCP, en cambio, **no expone ninguna tool de presupuesto** hoy: el catálogo tiene 48 tools (25 lectura + 23 escritura) y ninguna toca `category_groups`, `categories`, `accounts`, `budget_months` ni `transactions`.

Esta spec agrega un **toolset MCP `budget`** con:

- `budget_categories_list` (lectura) — grupos y categorías activas del presupuesto, para que el LLM mapee un gasto a su categoría.
- `budget_accounts_list` (lectura) — cuentas activas del presupuesto con su moneda (necesarias porque toda transacción exige una cuenta).
- `budget_transactions_create` (escritura) — registra un gasto (outflow) con **deduplicación**, `dry_run` (default `true`) y la compuerta `P40LA_MCP_ALLOW_WRITES`.
- P1: `budget_transactions_list` (lectura) y `budget_month_view` (lectura) para que el LLM vea lo registrado y el estado mensual.

Las tres tools P0 reutilizan los endpoints existentes; **no hay migraciones, dependencias ni rutas nuevas**. El cambio es solo la capa de cliente MCP (`internal/mcp/api/budget.go`), el toolset (`internal/mcp/tools/budget/`), su registro, el manifiesto/contrato y la documentación.

**Decisión clave documentada (no es requisito nuevo)**: el usuario no definió si la categoría la identifica el LLM de forma autónoma o la indica él. La spec **no** convierte esa incertidumbre en requisito: la app expone la superficie (listar categorías) y exige una categoría resuelta para crear el gasto; **quién** decide la categoría es el LLM. La tool aporta controles de ambigüedad (resolución por `id` o por `name` exacto con `ambiguous_category`) y de confirmación (`dry_run` + `status=duplicate`), no una política de categorización.

**El PDF y su extracción los hace el LLM externo** según el ejemplo del usuario; la aplicación no recibe ni procesa PDFs.

**Consideraciones iHost**: sin estado ni caché en el MCP; una consulta de listado y un POST corto; sin dependencias nuevas; sin cambios de esquema; respuesta acotada (el cliente MCP ya corta en 1 MiB).

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec toca backend MCP y reutiliza SQLite; **no toca frontend, rutas de UI, i18n ni migraciones**. Las reglas de UI/UX se declaran `N/A` con justificación.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go stdlib + SQLite; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Cliente en `internal/mcp/api/`; tools en `internal/mcp/tools/budget/`; el dominio de presupuesto sigue en `internal/{api,services,storage,models}` (legacy, sin migrar) | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Tool MCP → cliente HTTP → handler → service → storage. **Prohibido** SQL desde el MCP | `CA-ARCH-LAYERS` |
| `ARCH-MCP-TOOLS-MAP` | Las 5 tools se registran con nombre literal (`mcp.AddTool`) y entrada en `docs/mcp/tools/impact.json`; el contrato se regenera de forma intencional | `CA-ARCH-MCP-TOOLS-MAP` |
| `ARCH-IHOST` | Sin caché/estado en el MCP; consultas con rango de fechas (usa `idx_transactions_date`); respuesta acotada | `CA-ARCH-IHOST` |
| `ARCH-GO` | `gofmt`, nombres en inglés, errores explícitos, `log/slog` sin cambios | `CA-ARCH-GO` |
| `DATA-SQLITE` | Reutiliza `category_groups`, `categories`, `accounts`, `transactions` (migración 0035); SQL crudo con `database/sql` ya existente | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | **N/A**: no hay cambios de esquema | — |
| `DATA-PROTECTED-SCHEMA` | **N/A**: no se toca `debts` / `debt_bills` | — |
| `DATA-NO-DROP` | **N/A**: no se ejecuta nada sobre `data/app.db` | — |
| `DATA-PRODCOPY` | **N/A**: la validación usa DB temporal y fixtures sintéticos; no depende de datos reales | — |
| `DATA-PROD-WRITE` | **N/A**: no se escribe en producción | — |
| `SEC-NO-PII` | Las tools devuelven nombres de categorías/cuentas y montos del propio usuario; no agregan emails, teléfonos ni identificadores externos. Los ejemplos son sintéticos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Cuenta Demo`, `Proveedor Demo`, montos ficticios y fechas `2026-03-10` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Los logs del MCP no agregan argumentos ni montos; la auditoría existente no cambia | `CA-SEC-LOGS` |
| `SEC-COMMIT` | `git diff --cached` se revisa; sin DBs, PDFs, exports ni secretos | `CA-SEC-COMMIT` |
| `UI-*` (todas) | **N/A**: no toca frontend ni i18n | — |
| `UX-*` (todas) | **N/A**: no hay pantallas, rutas ni modales; el feedback se expresa con `status`, `warnings` y errores estables | — |
| `UX-PREREQ` | Aplica al backend MCP: `account_id`, `category_id`, `date` y `amount` se validan antes de escribir; el agente resuelve la categoría con `budget_categories_list` y la cuenta con `budget_accounts_list` | `CA-UX-PREREQ` |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` antes del commit; pruebas con DB temporal | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | Se actualiza `docs/mcp/tools/contracts.json` de forma intencional y se corren `TestToolContracts` + `TestMCPBackendContract` | `CA-TEST-MCP-CONTRACT` |
| `TEST-TESTUSER` | **N/A**: las pruebas usan httptest y DB temporal, no el server local con usuario | — |
| `DELIV-WORKTREE` | La spec vive en `../p40la-ihost-spec-163` sobre `feature/SPEC-163`; `./scripts/check-worktree.sh` pasa | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push con autorización explícita del usuario y sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa; `./scripts/check-mcp-impact.sh --files ...` y `--staged --gate` antes del commit | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; veredicto citado en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-MCP-DOCS` | `docs/mcp/tools/budget.md` (nuevo) y `docs/mcp/tools/README.md` se actualizan en el mismo commit | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente en paralelo a la implementación | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-COMMIT` | Commits `SPEC-163: ...` | `CA-DELIV-COMMIT` |
| `DELIV-LABELS` | Issue con `spec/draft`, transiciones a `spec/in-progress`, `spec/pending-release` y `spec/released` | `CA-DELIV-LABELS` |
| `DELIV-RELEASE` | Merge, push, tracker, label, cierre de issue y limpieza del worktree en el mismo ciclo | `CA-DELIV-RELEASE` |
| `DELIV-DOCKER` | **N/A**: no se publica imagen ni cambia el Dockerfile en esta spec | — |
| `DELIV-DEPLOY` | **N/A**: no se despliega al iHost en esta spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Exponer `budget_categories_list` (lectura) que devuelve los grupos de categorías con sus categorías **activas** (id, nombre, grupo, ícono, target opcional y `account_id` si existe), reutilizando `GET /api/budget/category-groups`.
2. **REQ-002**: Exponer `budget_accounts_list` (lectura) que devuelve las cuentas **activas** del presupuesto con su moneda (`id`, `name`, `type`, `currency_id`, `currency_code`, `starting_balance`, `balance`), reutilizando `GET /api/budget/accounts`.
3. **REQ-003**: Exponer `budget_transactions_create` (escritura) que registra un **gasto** (outflow) asociado a una cuenta, una categoría y una fecha; la fecha determina el mes del presupuesto (no hay parámetro de mes separado). Reutiliza `POST /api/budget/transactions`.
4. **REQ-004**: `budget_transactions_create` debe resolver la **categoría** por `category_id` o, alternativamente, por `category` (nombre exacto sobre las categorías activas). Si el nombre no coincide con ninguna, devolver `category_not_found`; si coincide con más de una, devolver `ambiguous_category` con las candidatas y no escribir.
5. **REQ-005**: `budget_transactions_create` debe resolver la **cuenta**. Si recibe `account_id`, lo usa. Si no lo recibe, usa la cuenta vinculada a la categoría (`categories.account_id`, SPEC-093) si existe; si no, usa la primera cuenta activa. Si no hay ninguna cuenta activa, devuelve `account_not_found` con un hint para crear una. La **moneda** de la transacción se deriva de la cuenta elegida (`accounts.currency_id`), patrón SPEC-097 ADR-002.
6. **REQ-006**: **Deduplicación**: antes de escribir, `budget_transactions_create` consulta las transacciones del mes destino (`GET /api/budget/transactions?year=&month=`, con el año/mes derivados de `date`) y busca una existente de la misma categoría con la misma `date`, el mismo `outflow` (≈ `amount`) y el mismo `payee` normalizado (vacío en ambos lados equivale a coincidencia). Si la encuentra y `allow_duplicate` no es `true`, devuelve `status=duplicate` con el registro existente y **no** escribe.
7. **REQ-007**: `dry_run` viene en `true` por defecto. En modo preview la tool devuelve `status=planned` (o `status=duplicate` si detecta el duplicado) con el plan resuelto (cuenta, categoría, fecha, monto, moneda) y sin escribir.
8. **REQ-008**: Respetar la compuerta de escritura existente: `P40LA_MCP_ALLOW_WRITES=true` (y `P40LA_MCP_REMOTE_WRITES=true` en modo HTTP). Sin la compuerta, `dry_run=false` devuelve `writes_disabled`.
9. **REQ-009**: Validar entrada: `date` en formato `YYYY-MM-DD` y no futura respecto a hoy (en la zona horaria configurada); `amount` > 0; `category_id`/`category` y `account_id` coherentes. Errores estables: `invalid_request`, `invalid_date`, `invalid_amount`, `category_not_found`, `ambiguous_category`, `account_not_found`.
10. **REQ-010**: Marcar las escrituras con `X-P40LA-Source: mcp` mediante `sourceHeaders()` del cliente existente (`internal/mcp/api/autos.go`).
11. **REQ-011**: Actualizar manifiesto, contrato congelado, stubs de `TestMCPBackendContract`, documentación MCP y registro del toolset en el mismo commit.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: Exponer `budget_transactions_list` (lectura) que lista las transacciones de un mes (`year`, `month` obligatorios), reutilizando `GET /api/budget/transactions?year=&month=`, para que el LLM verifique lo ya registrado.
2. **REQ-013**: Exponer `budget_month_view` (lectura) que devuelve la vista mensual completa (por grupo y categoría: `assigned`, `activity`, `available` por moneda, más `currency_totals`), reutilizando `GET /api/budget/months/{year}/{month}`.
3. **REQ-014**: Devolver `warnings: ["possible_duplicate"]` cuando se crea con `allow_duplicate=true` y había un duplicado detectado, para que el LLM pueda explicarlo.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Evaluar en una spec futura una clave de idempotencia persistida (`dedup_key` + índice único) para eliminar la carrera entre dos escrituras concurrentes. En esta spec la deduplicación es de aplicación, igual que el control de duplicados de `bills_create` y el de las líneas de reparación (SPEC-155).

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: la deduplicación es una consulta por mes sobre `transactions` con rango de fechas (usa `idx_transactions_date`); el alta es un `INSERT` único.
- **Memoria**: sin caché ni estado en el MCP; respuestas acotadas por el cliente (1 MiB).
- **Seguridad**: endpoints autenticados por API key/sesión; las tools no agregan secretos ni PII; escrituras con `X-P40LA-Source: mcp`.
- **Almacenamiento**: sin migraciones ni archivos de datos nuevos.
- **Disponibilidad**: la creación es atómica (un `INSERT`); un fallo deja la DB sin cambios.
- **iHost**: sin dependencias nuevas; dos consultas por escritura como máximo (listar categorías/cuentas ya se hace una vez por sesión del LLM); sin procesos en background.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- El módulo de presupuesto (SPEC-093) define las tablas `category_groups`, `categories`, `accounts`, `budget_months`, `recurring_rules`, `assignments`, `transactions` (migración `0035_create_budget_core.up.sql`). `transactions` es la tabla de movimientos (outflow/inflow) con `account_id` (NOT NULL), `category_id` (nullable), `currency_id` (NOT NULL) y `date` (TEXT `YYYY-MM-DD`). No tiene columna `source` (la auditoría `source=mcp` de facturas no aplica acá).
- Los endpoints ya existen y están montados en `internal/api/routes.go` (líneas 315–340): `GET /api/budget/category-groups`, `GET /api/budget/accounts`, `GET /api/budget/transactions?year=&month=`, `GET /api/budget/categories/{category_id}/transactions?year=&month=`, `POST /api/budget/transactions`, `GET /api/budget/months/{year}/{month}`, entre otros.
- `internal/api/budget_handlers.go` implementa `ListCategoryGroups` (grupos con categorías activas vía `CategoryGroupService.List`), `ListAccounts` (cuentas activas vía `AccountService.List`), `ListTransactions` (filtro `year`/`month` por rango `date >= YYYY-MM-01 AND date <= YYYY-MM-31`), `ListCategoryTransactions` y `CreateTransaction` (valida cuenta/categoría/moneda/fecha/montos en `BudgetTransactionService.validate`, `internal/services/transaction.go:76`).
- El **mes** no es un campo de la transacción: se deriva de `date` (ADR-001).
- `Category` (SPEC-093) tiene `AccountID *int64` opcional (cuenta vinculada); `Account` tiene `CurrencyID`. El frontend deriva la moneda de la cuenta (SPEC-097 ADR-002).
- El servidor MCP es un proceso separado que habla con la app por HTTP autenticado (`internal/mcp/api/client.go`). Las tools de lectura usan `api.ReadAPI`; las de escritura usan `api.API` y la compuerta `config.WritesEnabled()` (`internal/mcp/config/config.go:203`, envs `P40LA_MCP_ALLOW_WRITES` / `P40LA_MCP_REMOTE_WRITES`). El header de escritura se envía con `sourceHeaders()` (`internal/mcp/api/autos.go:302-304`).
- `FullAPI` (`internal/mcp/api/autos.go:26`) hoy embebe `API`, `AutosAPI`, `ReadAPI`, `HomeMaintenanceAPI`. Para agregar presupuesto se define `BudgetAPI` y se suma a `FullAPI`.
- El precedente directo de tool de escritura con `dry_run`, dedup y compuerta es SPEC-155 (`home_maintenance_labor_register` / `supplies_register`); el precedente de flujo de negocio es SPEC-097 (`/budget_transaction` del bot de Telegram, que ya crea transacciones del presupuesto).
- **Colisión de nombres a evitar**: las cuentas del **presupuesto** (tabla `accounts`, SPEC-093) son distintas de las **cuentas bancarias** (tabla `bank_accounts`, SPEC-133/152). El toolset usa el namespace `budget_` y el endpoint `/api/budget/accounts` para no confundirlas.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Solo `budget_transactions_create` sin tools de lectura | Catálogo mínimo | El LLM no puede mapear categorías ni cuentas sin preguntar todo al usuario; contradice la solicitud | ❌ Rechazada |
| B. `budget_categories_list` + `budget_accounts_list` + `budget_transactions_create` | Cubre exactamente los 3 criterios de éxito del usuario; reusa endpoints existentes | Tres tools nuevas | ✅ Seleccionada |
| C. Añadir también `budget_month_view` + `budget_transactions_list` como P0 | Contexto mensual y transparencia inmediatas | Más superficie; el mes y el listado son complementos, no bloqueantes | ⚠️ Diferido a P1 (REQ-012/013) |
| D. Recibir el PDF y extraerlo en la app | El usuario no extraería fuera | Rompe el alcance acordado (el PDF lo procesa el LLM externo); requiere OCR/deps | ❌ Rechazada (fuera de alcance) |
| E. La app decide la categoría por matching automático | «Cero preguntas» | El usuario no definió la política; inventar reglas de categorización viola el alcance | ❌ Rechazada (ver ADR-004) |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: el mes se deriva de la fecha, no es un parámetro.**
- **Contexto**: la tabla `transactions` no tiene mes; `ListByMonth` filtra por rango `date`. La app nunca pregunta por mes al crear una transacción.
- **Decisión**: `budget_transactions_create` recibe `date` (`YYYY-MM-DD`) y el mes queda implícito. No existe `month`/`year` en la entrada de creación.
- **Consecuencias**: sin ambigüedad de mes; la factura aporta la fecha y el mes sale de ella. Las tools de lectura `budget_transactions_list`/`budget_month_view` sí reciben `year`/`month` porque son consultas.

**ADR-002: la moneda se deriva de la cuenta.**
- **Contexto**: `transactions.currency_id` es obligatorio y cada cuenta tiene su moneda (`accounts.currency_id`).
- **Decisión**: no hay parámetro de moneda; se toma de la cuenta resuelta (patrón SPEC-097 ADR-002).
- **Consecuencias**: coherencia cuenta↔moneda garantizada; menos parámetros. Si más adelante se quiere moneda distinta a la cuenta, se agrega como P2.

**ADR-003: deduplicación de aplicación (no índice único), con `dry_run` y `allow_duplicate`.**
- **Contexto**: la clave de negocio (fecha + categoría + monto + payee libre normalizado) no se presta a un índice único sin migración.
- **Decisión**: la tool consulta las transacciones del mes y compara antes de escribir; `dry_run` default `true`; `allow_duplicate=true` permite el alta explícita con `warnings`. La validación no es atómica frente a dos escrituras concurrentes.
- **Consecuencias**: se evita el duplicado accidental típico sin cambiar el esquema; la carrera queda documentada como riesgo y se mitiga con `budget_transactions_list` (P1).

**ADR-004: la categoría se resuelve por `id` o por `name` exacto; la app no decide quién la elige.**
- **Contexto**: el usuario no definió si el LLM identifica la categoría autónomamente o se la indica él.
- **Decisión**: la app expone `budget_categories_list` (superficie) y `budget_transactions_create` exige una categoría resuelta (`category_id` o `category`). Si `category` (nombre) coincide con varias activas, devuelve `ambiguous_category` con candidatas. La decisión de mapear «comida»→`category_id` recae en el LLM, que puede hacerlo solo o preguntando al usuario.
- **Consecuencias**: no se inventa política de categorización; se da un control de ambigüedad concreto (no un matching automático silencioso).

**ADR-005: namespace `budget_` para evitar colisión con `bank_accounts`.**
- **Contexto**: existen dos conceptos de «cuenta» (presupuesto vs banca).
- **Decisión**: las tools usan `budget_categories_list`, `budget_accounts_list`, `budget_transactions_*`; el cliente usa los endpoints `/api/budget/*`.
- **Consecuencias**: sin colisión semántica con `bank_accounts_search`/`credit_cards_*`; los nombres de dominio son explícitos.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
LLM (Claude Code / MCP client)
   │  budget_categories_list · budget_accounts_list · budget_transactions_create
   ▼
internal/mcp/tools/budget  --(cliente BudgetAPI)-->  API HTTP /api/budget/...
   │                                                      │
   │  GET /category-groups, /accounts,                    v
   │  GET /transactions, GET /months/{y}/{m}                internal/api/budget_handlers.go
   │  POST /transactions (X-P40LA-Source: mcp)              │
   +--------------------------------------------------------+--> services (budget/transaction)
                                                                    │
                                                              storage (SQL crudo)
                                                                    │
                                              SQLite: category_groups, categories,
                                                      accounts, budget_months, transactions
```

### 5.2 Componentes

#### 5.2.1 Cliente MCP `internal/mcp/api/budget.go` (nuevo)

- **Responsabilidad**: transportar y decodificar las llamadas de presupuesto; construir query strings; enviar `X-P40LA-Source: mcp` en el POST.
- **Interfaz** `BudgetAPI` (patrón `AutosAPI`/`HomeMaintenanceAPI`):
  - `ListCategoryGroups(ctx) ([]CategoryGroup, error)` → `GET /api/budget/category-groups`
  - `ListAccounts(ctx) ([]Account, error)` → `GET /api/budget/accounts`
  - `ListTransactions(ctx, year, month int) ([]Transaction, error)` → `GET /api/budget/transactions?year=&month=`
  - `MonthView(ctx, year, month int) (*MonthView, error)` → `GET /api/budget/months/{year}/{month}`
  - `CreateTransaction(ctx, req TransactionRequest) (*Transaction, error)` → `POST /api/budget/transactions` (con `sourceHeaders()`)
- **Dependencias**: `internal/mcp/api/client.go` (`doJSON`, `ReadJSON`), `sourceHeaders()` en `autos.go`.
- **Ubicación**: `internal/mcp/api/budget.go`.

#### 5.2.2 Tools MCP `internal/mcp/tools/budget/` (nuevo)

- **Responsabilidad**: validar entrada, resolver cuenta/categoría, deduplicar, aplicar `dry_run`/compuerta y devolver un resultado accionable.
- **Interfaz**: `budget_categories_list`, `budget_accounts_list`, `budget_transactions_create` (P0); `budget_transactions_list`, `budget_month_view` (P1).
- **Dependencias**: `api.BudgetAPI`, `config.Config`, `toolkit`.
- **Ubicación**: `internal/mcp/tools/budget/` (archivos `tools.go`, `create.go`, `list.go`, `*_test.go`).

#### 5.2.3 Registro `internal/mcp/tools/all/register.go` (modificar)

- Importar `"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/budget"` y agregar `r.Add(func(server *mcp.Server) { budget.Register(server, client, cfg) })`.

### 5.3 Modelo de datos

Sin cambios de esquema. Se reutilizan (migración `0035_create_budget_core`):

```
category_groups: id, name, icon, sort_order, ...
categories:      id, category_group_id, name, icon, sort_order,
                 target_amount, target_type, target_date, account_id (nullable), deleted_at
accounts:        id, name, type (checking|savings|credit_card|cash), currency_id, starting_balance, deleted_at
budget_months:   id, year, month, UNIQUE(year, month)
transactions:    id, account_id, category_id (nullable), currency_id, date,
                 payee, memo, outflow, inflow, cleared, source_bill_id (nullable), deleted_at
                 Índices: (date), (category_id, date)
```

### 5.4 APIs / Contratos

#### Tool `budget_categories_list`

**Entrada**: sin parámetros.

**Salida**:
```json
{
  "groups": [
    {
      "id": 2,
      "name": "Necesidades",
      "icon": "cart",
      "categories": [
        {"id": 5, "name": "Comida", "icon": "food", "target_amount": 5000, "target_type": "monthly", "account_id": 3}
      ]
    }
  ],
  "count": 1
}
```

#### Tool `budget_accounts_list`

**Entrada**: sin parámetros.

**Salida**:
```json
{
  "accounts": [
    {"id": 3, "name": "Cuenta Demo", "type": "checking", "currency_id": 1, "currency_code": "NIO", "starting_balance": 1000, "balance": 850}
  ],
  "count": 1
}
```

#### Tool `budget_transactions_create`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `date` | string | Sí | `YYYY-MM-DD`; no futura; determina el mes |
| `amount` | number | Sí | Monto del gasto (outflow), > 0 |
| `category_id` | integer | No | Categoría destino (alternativa a `category`) |
| `category` | string | No | Nombre exacto de la categoría (alternativa a `category_id`) |
| `account_id` | integer | No | Cuenta destino; si falta, cuenta de la categoría o primera activa |
| `payee` | string | No | Beneficiario/comercio (para deduplicar) |
| `memo` | string | No | Nota libre |
| `dry_run` | boolean | No | Default `true` |
| `allow_duplicate` | boolean | No | Default `false` |

**Salida (alta)**:
```json
{
  "status": "created",
  "dry_run": false,
  "transaction": {
    "id": 10,
    "account_id": 3,
    "account_name": "Cuenta Demo",
    "category_id": 5,
    "category_name": "Comida",
    "currency_id": 1,
    "currency_code": "NIO",
    "date": "2026-03-10",
    "payee": "Proveedor Demo",
    "memo": "",
    "outflow": 1200,
    "inflow": 0,
    "cleared": false
  }
}
```

**Salida (preview)**:
```json
{
  "status": "planned",
  "dry_run": true,
  "plan": {
    "account_id": 3,
    "account_name": "Cuenta Demo",
    "category_id": 5,
    "category_name": "Comida",
    "currency_code": "NIO",
    "date": "2026-03-10",
    "amount": 1200,
    "payee": "Proveedor Demo"
  }
}
```

**Salida (duplicado)**:
```json
{
  "status": "duplicate",
  "dry_run": false,
  "existing": {"id": 9, "date": "2026-03-10", "category_id": 5, "category_name": "Comida", "outflow": 1200, "payee": "Proveedor Demo"},
  "hint": "Ya existe un gasto equivalente el mismo día en esa categoría. Usá allow_duplicate=true solo si es un registro distinto."
}
```

**Errores**: `invalid_request`, `invalid_date`, `invalid_amount`, `category_not_found`, `ambiguous_category` (con `candidates`), `account_not_found`, `writes_disabled`.

#### Tool `budget_transactions_list` (P1)

**Entrada**: `year` (int, obligatorio), `month` (int 1–12, obligatorio).

**Salida**: `{ "transactions": [ ... ], "count": N }` con los mismos campos de `Transaction`.

#### Tool `budget_month_view` (P1)

**Entrada**: `year` (int), `month` (int 1–12).

**Salida**: la `BudgetMonthView` del endpoint: `{ year, month, currency_totals: [...], groups: [{ id, name, icon, categories: [{...category, assigned, activity, available, recurring_rule?}] }] }`.

### 5.5 Dependencias

- **Internas**: `internal/mcp/api/budget.go` (nuevo), `internal/mcp/api/autos.go` (`FullAPI`), `internal/mcp/tools/budget/` (nuevo), `internal/mcp/tools/all/register.go`, `internal/mcp/tools/all/contract_test.go`.
- **Externas**: ninguna nueva.

### 5.6 Archivos previstos

| Archivo | Cambio |
|---|---|
| `internal/mcp/api/budget.go` | nuevo; `BudgetAPI`, tipos y llamadas |
| `internal/mcp/api/autos.go` | `FullAPI` embebe `BudgetAPI` |
| `internal/mcp/tools/budget/tools.go` | nuevo; `Toolset`, `Register`, `writeEnabled` |
| `internal/mcp/tools/budget/create.go` | nuevo; `budget_transactions_create` (resolución, dedup, dry_run) |
| `internal/mcp/tools/budget/list.go` | nuevo; `budget_categories_list`, `budget_accounts_list`, `budget_transactions_list`, `budget_month_view` |
| `internal/mcp/tools/budget/*_test.go` | nuevo; tests del toolset |
| `internal/mcp/tools/all/register.go` | registrar `budget.Register` |
| `internal/mcp/tools/all/contract_test.go` | 5 stubs de `BudgetAPI` |
| `internal/api/mcp_backend_contract_test.go` | montar `NewBudgetHandlers` y cubrir la cadena real de `/api/budget/*` |
| `docs/mcp/tools/impact.json` | 5 tools nuevas + dependencias |
| `docs/mcp/tools/contracts.json` | golden regenerado |
| `docs/mcp/tools/budget.md` | nuevo; documentación de las 5 tools |
| `docs/mcp/tools/README.md` | catálogo 48 → 53 (29 lectura + 24 escritura) |
| `docs/mcp/README.md` | conteo y tabla de escritura |
| `docs/mcp/security.md` | tools sujetas a la compuerta |
| `docs/specs/README.md` | tracker (solo en el release) |

**Sin migraciones**: el esquema de `0035_create_budget_core` no cambia.

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). Análisis con `./scripts/check-mcp-impact.sh --files ...` y validación de un subagente (`DELIV-MCP-IMPACT-SUBAGENT`).

### 6.1 Tools nuevas

| Tool | Entrypoint | Endpoints | Contrato |
|---|---|---|---|
| `budget_categories_list` | `internal/mcp/tools/budget/list.go` | `GET /api/budget/category-groups` | Nueva |
| `budget_accounts_list` | `internal/mcp/tools/budget/list.go` | `GET /api/budget/accounts` | Nueva |
| `budget_transactions_create` | `internal/mcp/tools/budget/create.go` | `GET /api/budget/category-groups`, `GET /api/budget/accounts`, `GET /api/budget/transactions`, `POST /api/budget/transactions` | Nueva; `dry_run`, `allow_duplicate` |
| `budget_transactions_list` | `internal/mcp/tools/budget/list.go` | `GET /api/budget/transactions` | Nueva (P1) |
| `budget_month_view` | `internal/mcp/tools/budget/list.go` | `GET /api/budget/months/{year}/{month}` | Nueva (P1) |

### 6.2 Tools existentes afectadas funcionalmente

**Ninguna.** No se modifica el contrato, la ruta ni el comportamiento de ninguna tool existente. Solo se agregan tools nuevas y un paquete de cliente nuevo.

### 6.3 Ruta compartida `internal/mcp/api/` y `internal/mcp/tools/all/`

El manifiesto declara `internal/mcp/api/` y `internal/mcp/tools/all/` en `shared.affects_all`. Agregar `internal/mcp/api/budget.go`, ampliar `FullAPI` (en `internal/mcp/api/autos.go`) y registrar el toolset en `register.go` hace que el analizador reporte **las 48 tools existentes** por compilación del paquete compartido, aunque su comportamiento no cambia. El gate exige declararlas; se listan a continuación:

`alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bank_accounts_create`, `bank_accounts_search`, `bank_accounts_update`, `banks_create`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `credit_card_statements_get`, `credit_card_statements_list`, `credit_cards_create`, `credit_cards_search`, `credit_cards_update`, `debt_bills_list`, `entities_search`, `home_maintenance_create`, `home_maintenance_get`, `home_maintenance_item_delete`, `home_maintenance_item_move`, `home_maintenance_item_update`, `home_maintenance_labor_register`, `home_maintenance_list`, `home_maintenance_supplies_register`, `home_maintenance_update`, `homes_create`, `homes_update`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

**Impacto real**: nulo en el comportamiento de esas 48 tools; solo se recompila el paquete `internal/mcp/api` y `internal/mcp/tools/all`.

### 6.4 Contrato y tests

- `docs/mcp/tools/contracts.json`: se regenera con `go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts`; se revisa el diff.
- `docs/mcp/tools/impact.json`: se agregan las 5 tools nuevas con `entrypoints`, `dependencies`, `endpoints` y `tests`. En `dependencies` incluir la ruta real `internal/services/transaction.go` (el `BudgetTransactionService` vive ahí, no en `budget_transaction.go`) y los handlers/storages de presupuesto, para evitar falsos negativos del `--check-map`.
- `internal/mcp/tools/all/contract_test.go`: `contractAPI` implementa `api.FullAPI` método por método; al ampliar `FullAPI` con `BudgetAPI` hay que agregar **5 stubs** (`ListCategoryGroups`, `ListAccounts`, `ListTransactions`, `MonthView`, `CreateTransaction`) o el paquete de tests no compila.
- `internal/api/mcp_backend_contract_test.go`: hoy `handler.budget` queda `nil` (el literal de `Handler` no asigna presupuesto) mientras `BuildRouter` monta las rutas de presupuesto incondicionalmente. Para cubrir la cadena real hay que construir `NewBudgetHandlers(...)` (patrón de `internal/api/budget_transfer_test.go:20-47`), asignarlo al literal y seedear currency/grupo/categoría/cuenta, cubriendo `GET /api/budget/category-groups`, `GET /api/budget/accounts`, `GET /api/budget/transactions`, `GET /api/budget/months/{y}/{m}` y `POST /api/budget/transactions` (tags `account_id`, `category_id`, `currency_id`, `date`, `payee`, `memo`, `outflow`, `inflow`, `cleared`).
- `docs/mcp/tools/README.md`, `docs/mcp/README.md`, `docs/mcp/security.md`: catálogo 48 → 53 (29 lectura + 24 escritura) y lista de tools sujetas a `P40LA_MCP_ALLOW_WRITES`.
- `docs/mcp/tools/budget.md`: nuevo (delegado a subagente, `DELIV-MCP-DOCS-SUBAGENT`).

Tests:
- `internal/mcp/tools/budget/*_test.go` (dry-run, create, duplicate, allow_duplicate, ambiguous_category, account default, writes_disabled, list filters).
- `internal/mcp/api/budget_test.go` (query string y decode).
- `internal/mcp/tools/all/contract_test.go` (stubs + golden).
- `go test ./internal/mcp/...`
- `go test ./internal/api -run TestMCPBackendContract`
- `./scripts/check-mcp-impact.sh --staged --gate`
- `./scripts/check-mcp-docs.sh --staged`

Validación subagente: explore — sesión `ses_debecb6d5ffeMS3Gjo78ljA1Lk` — veredicto: viable. No existe ningún toolset MCP de presupuesto ni `BudgetAPI` hoy; los endpoints HTTP ya están montados y el modelo de datos ya existe (migración 0035). El patrón a seguir es el de SPEC-155/159. El analizador declarará las 48 tools existentes por `shared.affects_all` (`internal/mcp/api/` y `internal/mcp/tools/all/`) sin cambio funcional. Requiere: 5 stubs en `contract_test.go`, montar `NewBudgetHandlers` en `TestMCPBackendContract` (hoy `handler.budget` queda `nil`), nombres literales en `mcp.AddTool`, y la ruta real `internal/services/transaction.go` en el manifiesto para evitar falsos negativos. Catálogo 48 → 53 (29 lectura + 24 escritura).

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [x] `CA-REQ-001`: `budget_categories_list` devuelve los grupos con sus categorías activas (id, nombre, grupo, ícono, target y `account_id` si existe). Verificado por smoke E2E.
- [x] `CA-REQ-002`: `budget_accounts_list` devuelve las cuentas activas con `currency_id`/`currency_code`. Verificado por smoke E2E.
- [x] `CA-REQ-003`: `budget_transactions_create` con `dry_run=false`, `account_id`, `category_id`, `date` y `amount` crea una transacción `outflow` visible en `/api/budget/transactions` y en la grilla del presupuesto. Verificado por smoke E2E (`status=created` + `budget_month_view`).
- [x] `CA-REQ-004`: con `category` (nombre) que coincide con varias categorías activas, devuelve `ambiguous_category` con `candidates` y no escribe; con un nombre inexistente devuelve `category_not_found`. Cubierto por unit tests de `budget_transactions_create`.
- [x] `CA-REQ-005`: sin `account_id`, la tool usa la cuenta vinculada a la categoría o la primera activa; la moneda de la transacción es la de esa cuenta. Verificado por smoke E2E (sin `account_id`, resolvió la única cuenta activa en NIO).
- [x] `CA-REQ-006`: repetir el mismo gasto (misma `date`, categoría, `amount` y `payee`) devuelve `status=duplicate` y **no** crea una segunda fila. Verificado por smoke E2E (2 llamadas, 1 fila).
- [x] `CA-REQ-006B`: con `allow_duplicate=true`, la segunda transacción se crea y la respuesta incluye `warnings`. Verificado por smoke E2E (`warnings: ["possible_duplicate"]`).
- [x] `CA-REQ-007`: `dry_run` default `true` devuelve `status=planned` con el plan resuelto y sin escribir. Verificado por smoke E2E.
- [x] `CA-REQ-008`: sin `P40LA_MCP_ALLOW_WRITES`, `dry_run=false` devuelve `writes_disabled`. Verificado por smoke E2E.
- [x] `CA-REQ-009`: `date` inválida o futura devuelve `invalid_date`; `amount <= 0` devuelve `invalid_amount`. Cubierto por unit tests.
- [x] `CA-REQ-010`: las escrituras llegan con `X-P40LA-Source: mcp`. Verificado en `internal/mcp/api/budget.go` (`sourceHeaders()` en el `POST`) y en `TestMCPBackendContract`.
- [x] `CA-REQ-011`: `impact.json`, `contracts.json`, `budget.md` y los conteos de `docs/mcp/` actualizados. Verificado en el diff del commit de implementación.
- [x] `CA-REQ-012`: `budget_transactions_list` devuelve las transacciones del `year`/`month` indicados (P1). Verificado por smoke E2E (2 transacciones).
- [x] `CA-REQ-013`: `budget_month_view` devuelve `assigned`/`activity`/`available` por categoría y `currency_totals` (P1). Verificado por smoke E2E (`activity` 24.68 en NIO).
- [x] `CA-UX-PREREQ`: el backend valida los prerrequisitos (cuenta, categoría, moneda, fecha) y la tool los reporta antes de escribir. Cubierto por unit tests y smoke E2E.

### 7.2 No funcionales

- [x] `CA-ARCH-STACK`: sin dependencias nuevas; `go build ./...` y `go test ./...` pasan. Verificado.
- [x] `CA-SEC-SYNTHETIC`: todos los ejemplos y fixtures usan datos sintéticos (`Cuenta Demo`, `Proveedor Demo`, montos ficticios). Verificado en tests y smoke.
- [x] `CA-TEST-MCP-CONTRACT`: `TestToolContracts` y `TestMCPBackendContract` verdes; golden revisado. Verificado.
- [x] `CA-DELIV-MCP-IMPACT`: `./scripts/check-mcp-impact.sh --staged --gate` verde antes del merge. Verificado en el commit de release.
- [x] `CA-DELIV-WORKTREE`: `./scripts/check-worktree.sh` verde en `../p40la-ihost-spec-163`. Verificado (exit 0).

### 7.4 Verificación del release (2026-11-25)

- **Build y tests**: `go build ./...`, `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract` verdes.
- **Smoke E2E** con server local y DB temporal en `/tmp` (nunca `data/app.db`): `budget_categories_list`, `budget_accounts_list`, `budget_transactions_create` (planned, created, duplicate, `allow_duplicate`), `writes_disabled`, `budget_transactions_list` y `budget_month_view`.
- **No aplica** prueba visual: la feature no toca frontend. Release autorizado por el usuario.

### 7.3 Testing

- **Unit tests**: resolución de cuenta/categoría, normalización de `payee`, validación de `date`/`amount`, comparación de duplicados, estados (`planned`/`created`/`duplicate`).
- **Integration tests**: `httptest` del cliente `BudgetAPI` (query strings y decode); deduplicación contra DB temporal con fixtures sintéticos.
- **MCP tool tests**: dry-run, create, duplicate, allow_duplicate, ambiguous_category, account default, writes_disabled.
- **Contrato**: `TestToolContracts` con golden actualizado; `TestMCPBackendContract` sin regresiones.
- **Carga**: no aplica; volumen bajo y hay topes implícitos por rango mensual.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Cliente MCP `BudgetAPI` en `internal/mcp/api/budget.go` + `FullAPI` + 5 stubs de `contract_test.go` | 0.25 día | Ninguna |
| 2 | Toolset `internal/mcp/tools/budget/`: `budget_categories_list`, `budget_accounts_list`, helpers de resolución | 0.25 día | Fase 1 |
| 3 | `budget_transactions_create` con resolución, dedup, `dry_run` y compuerta; `budget_transactions_list` y `budget_month_view` (P1) | 0.5 día | Fase 2 |
| 4 | Manifiesto, contrato, docs MCP (subagente) y README de tools | 0.25 día | Fase 3 |
| 5 | Gate completo, merge, release y limpieza | 0.25 día | Fase 4 |

### 8.2 Milestones

1. **MVP**: cliente + `budget_categories_list` + `budget_accounts_list` + `budget_transactions_create` con dedup y dry-run; tests verdes.
2. **V1.0**: `budget_transactions_list` y `budget_month_view` (P1), documentación, manifiesto y contrato; release.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Dos escrituras concurrentes pasan el chequeo de duplicado | Baja | Medio | Documentado (ADR-003); `budget_transactions_list` permite revisar; spec futura evalúa `dedup_key` único |
| El LLM confunde «cuenta del presupuesto» con «cuenta bancaria» | Media | Medio | Namespace `budget_` explícito y descripción de las tools; documentación `budget.md` |
| El LLM mapea mal la categoría (nombre similar) | Media | Medio | `ambiguous_category` con candidatas; `budget_categories_list` devuelve nombres e ids explícitos; la decisión final queda en el LLM/usuario |
| `account_id` no resuelto deja la tool sin cuenta | Media | Bajo | Default: cuenta de la categoría → primera activa → `account_not_found` con hint |
| El cambio en `internal/mcp/api/` se interpreta como impacto en las 48 tools | Alta (analizador) | Bajo | Declarado en 6.3; sin cambio de comportamiento; tests MCP completos |
| Regenerar el golden sin revisar el diff | Baja | Medio | Diff revisado y declarado; `TestToolContracts` sin `-update` en CI |
| Duplicado legítimo bloqueado | Baja | Bajo | `allow_duplicate=true` con `warnings` |
| `contract_test.go` no compila por falta de stubs | Alta (si se omite) | Alto | Los 5 stubs de `BudgetAPI` se agregan en la misma fase que ampliar `FullAPI` |

---

## 10. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-093 (módulo presupuesto YNAB), SPEC-094/096 (categorías vinculadas a servicios), SPEC-097 (bot Telegram `/budget_transaction`), SPEC-145 (MCP y compuerta de escritura), SPEC-153 (guardrail de impacto), SPEC-154 (catálogo de tools), SPEC-155 (precedente de tool de escritura con dry_run + dedup).
- Documentación: `docs/mcp/tools/budget.md` (a crear), `docs/mcp/tools/README.md`, `docs/mcp/tools/impact.md`.
- Código de referencia: `internal/api/budget_handlers.go`, `internal/services/transaction.go`, `internal/services/budget.go`, `internal/mcp/api/autos.go` (`FullAPI`, `sourceHeaders()`), `internal/mcp/tools/banking/accounts.go`, `internal/mcp/tools/homemaint/`.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-25 | opencode | Creación inicial de la especificación; issue #165 |
| 2026-11-25 | opencode | Estado draft → pending_execution (modo develop); spec lista para implementar |
| 2026-11-25 | Developer | Estado pending_execution → in_progress; toolset `budget` implementado con tests, manifiesto, contrato y documentación MCP |
| 2026-11-25 | DevOps | Estado in_progress → pending_release; QA y Security aprobaron |
| 2026-11-25 | DevOps | CA verificadas con smoke E2E sobre DB temporal (planned/created/duplicate/writes_disabled/month view) y preparación de release |
