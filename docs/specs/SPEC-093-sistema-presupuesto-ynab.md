---
title: "Sistema de Presupuesto estilo YNAB: grupos/categorías, asignación mensual, recurrencia y transacciones"
id: "SPEC-093"
status: "released"
author: "opencode"
created: "2026-09-27"
updated: "2026-09-27"
github_issue: 96
---

# Sistema de Presupuesto estilo YNAB: grupos/categorías, asignación mensual, recurrencia y transacciones

**ID**: SPEC-093  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-27  
**Actualizado**: 2026-09-27

---

## 1. Resumen Ejecutivo

Construir un módulo de **presupuesto personal estilo YNAB** dentro de `p40la-ihost`. El usuario configura una vez sus **grupos y categorías** (ej. "Necesidades", "Gustos"), cada mes **asigna montos** a cada categoría (de forma puntual o recurrente), y luego registra **transacciones** (gastos/ingresos) contra esas categorías. La pantalla principal es la **vista mensual** con las columnas `Categoría | Asignado | Actividad | Disponible`, agrupada por `CategoryGroup`, con navegación entre meses y un encabezado de "plata sin asignar". Se agregan también **cuentas** (banco, tarjeta, efectivo, ahorro) con su balance y una **grilla de transacciones** estilo YNAB.

Decisión del usuario (relevada al crear la spec): ítem de menú **"Presupuesto"** en el sidebar como **submenú colapsable** (estilo Pensión alimenticia) con dos subpáginas en v1: **Vista mensual** (`/budget`) y **Transacciones** (`/budget/transacciones`). La **edición de grupos/categorías va dentro de la vista mensual** (sección de configuración accesible desde esa página), no como submenú separado. Prefijo de ruta `/budget`, clave i18n `menu.budget`. Alcance v1: **Fases 1 a 4** del requerimiento (fundaciones, presupuesto mensual, recurrencia y transacciones). **Sin carryover**: el `available` se calcula como `assigned - activity` del mes, se resetea cada mes y la UI lo indica explícitamente. El módulo es **multi-moneda** (reutiliza las monedas existentes de `currencies`, como Deudas) y las **cuentas son independientes** de instituciones (no se vinculan).

**Consideraciones iHost**: módulo backend + frontend sin dependencias externas nuevas (Go stdlib, SQLite). Tablas nuevas: `accounts`, `category_groups`, `categories`, `recurring_rules`, `budget_months`, `assignments`, `transactions` (migraciones `0035`+). Memoria/CPU despreciables: CRUDs simples + consultas de agregación por mes/categoría. Se siguen los patrones de capas existentes (`models`, `storage`, `services`, `api`) y el patrón de UI de cards + `CreateMenu` + `EmptyCard` + `CardMenu` de 3 puntos.

**Consideraciones de UI obligatorias**: todos los `input`/`select`/`textarea` DEBEN usar tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), verificarse en darkmode (SPEC-060). El ítem "Presupuesto" en el sidebar usa submenú colapsable (patrón de `Sidebar.tsx` con `pensionItems`). La edición de grupos/categorías desde la vista mensual abre un modal/formulario; no crea páginas de detalle con lista padre nuevas, por lo que solo aplica `BACK_ROUTES` si se implementa como sub-ruta (ver CA-BACK). **Los dropdowns NUNCA usan `<select>` nativo del sistema: todos los selectores usan el componente custom `Select`** (`frontend/src/components/Select.tsx`, patrón ya documentado en SPEC-004 REQ-024 y aplicado en ServiceFormPage/BillFormPage). Esto aplica a moneda, cuenta, categoría y tipo de cuenta en los modales de presupuesto.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Menú **"Presupuesto"** en el sidebar como **submenú colapsable** con dos subpáginas: **Vista mensual** (`/budget`) y **Transacciones** (`/budget/transacciones`). Ícono propio (ej. `wallet`), clave i18n `menu.budget` → "Presupuesto" en `frontend/public/i18n/{es,en}.json`.
2. **REQ-002**: Modelo de datos y migraciones (`0035`+): tablas `accounts`, `category_groups`, `categories`, `recurring_rules`, `budget_months`, `assignments`, `transactions` (esquema en sección 4.3).
3. **REQ-003**: CRUD de **grupos de categorías** (`category_groups`): nombre, ícono, orden. Solo edición/borrado de grupos sin categorías (o con confirmación; archivar en vez de borrado duro si tiene categorías).
4. **REQ-004**: CRUD de **categorías** (`categories`): nombre, ícono, orden, grupo padre, y `target` opcional (meta mensual). **Eliminar una categoría con transacciones/assignments asociados → archivar** (soft delete, se oculta de la vista mensual pero conserva historial). Nunca borrado duro en ese caso.
5. **REQ-005**: CRUD de **cuentas** (`accounts`): nombre, tipo (`checking`/`savings`/`credit_card`/`cash`), moneda (`currency_id`), balance inicial. Independientes de instituciones.
6. **REQ-006**: **Vista mensual** (`/budget?year=YYYY&month=MM`): tabla `Categoría | Asignado | Actividad | Disponible` agrupada por `CategoryGroup`, con selector de mes (◀ ▶), total "plata sin asignar" por moneda, y empty state cuando no hay categorías (botón para crear la primera).
7. **REQ-007**: **Asignación** por categoría/mes (puntual `one_time`): input de monto desde la vista mensual (modal/panel lateral al hacer click en la categoría). Única por `(budget_month_id, category_id)` — asignar dos veces en el mismo mes suma o edita, no duplica.
8. **REQ-008**: Cálculo por categoría y mes: `assigned` (suma de assignments), `activity` (suma `outflow - inflow` de transacciones del mes en esa categoría), `available = assigned - activity` (**sin carryover**, se resetea cada mes; la UI muestra el aviso "El saldo se resetea cada mes").
9. **REQ-009**: **Reglas recurrentes** (`recurring_rules`): crear desde la asignación de una categoría ("hacer recurrente desde este mes"), monto, `frequency='monthly'`, `start_month`, `end_month` (nullable), `active`. Pausar/reanudar y editar monto (solo afecta desde el mes actual en adelante, nunca meses pasados). Al abrir un mes nuevo, generar automáticamente los `assignments` derivados de reglas activas.
10. **REQ-010**: **CRUD de transacciones** (`transactions`): cuenta (`account_id`), categoría (`category_id`, nullable = "Sin categorizar"), fecha, payee, memo, `outflow`/`inflow`, `cleared` (bool). Alta rápida inline desde `/budget/transacciones`. Al guardar, impacta automáticamente `activity`/`available` de la categoría en el mes correspondiente.
11. **REQ-011**: **Multi-moneda**: asignaciones, transacciones y cuentas usan `currency_id` de la tabla existente `currencies`. Los cálculos de `assigned/activity/available` y los totales se agrupan **por moneda** (un presupuesto puede tener categorías en C$ y USD; la vista muestra totales por moneda). Deudas/Servicios ya siguen este patrón.
12. **REQ-012**: **Transacciones sin categoría** se muestran en el bucket "Sin categorizar" sin romper los cálculos del resto de categorías.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: Edición de **grupos/categorías desde la vista mensual**: sección/acción de "Configurar" dentro de `/budget` que abre la gestión (crear/editar/eliminar/reordenar) sin submenú separado en el sidebar. Reordenar grupos y categorías con drag & drop (patrón existente de `HomesPage` + `SortableGrid`, SPEC-061) o con controles de orden (P2 si complejiza).
2. **REQ-014**: Modal/panel de asignación con **historial simple de actividad**: lista de transacciones del mes en esa categoría.
3. **REQ-015**: **Avísos YNAB**: no bloquear si `assigned > available total` (overfunded) — permitir y mostrar aviso ("Overfunded"). Mostrar aviso si la categoría está "sin usar" (`assigned > 0` y `activity == 0`).
4. **REQ-016**: **Índices y consultas eficientes** en SQLite para los cálculos mensuales (índices en `transactions(date, category_id)`, `assignments(budget_month_id, category_id)`, etc.) para mantener la vista mensual rápida en iHost.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-017**: Meta (`target`) por categoría con aviso "$X más necesario antes del día 30" (patrón Rent/Mortgage de YNAB).
2. **REQ-018**: Flag manual rápido "ya lo gasté" (`is_planned_spent`) en categoría/mes para registro rápido sin transacción completa.
3. **REQ-019**: Carryover del `available` no gastado al mes siguiente (excluido de v1 por decisión del usuario; dejar el diseño listo para Fase 5).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: La vista mensual debe cargar en < 300ms con datos típicos (~50 categorías, ~500 transacciones/mes) en iHost. Usar una query de agregación por mes+categoría (un solo `GROUP BY`), no N+1.
- **Seguridad**: Todos los endpoints bajo `authMiddleware` (sesión). Validación de IDs y montos (no negativos para outflow/assignment). Sin datos sensibles nuevos.
- **Almacenamiento**: 7 tablas nuevas, tamaño estimado despreciable (< 1 MB/año de uso personal). Montos como `REAL` (mismo patrón que `debts`/`bills`).
- **Disponibilidad**: CRUDs simples; sin schedulers nuevos (la generación de recurrentes se hace on-demand al abrir un mes o al crear el mes). Health check existente.
- **iHost**: Sin dependencias nuevas (Go stdlib + SQLite existente). Migraciones `0035`+ en el mecanismo existente. Frontend build estático con Vite (fuera del iHost).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Requerimiento fuente**: `/home/paulomcnally/Downloads/budget-system-spec.md` (estilo YNAB). Se relevó con el usuario: menú "Presupuesto" (submenú), rutas `/budget`, i18n `menu.budget`, fases 1-4 en v1, sin carryover, multi-moneda, cuentas independientes.
- **Patrón de sidebar**: `frontend/src/components/Sidebar.tsx` ya implementa submenú colapsable para **Pensión alimenticia** (`pensionItems`, estado `pensionOpen`). Se replica ese patrón para Presupuesto.
- **Patrón de rutas frontend**: `frontend/src/App.tsx` registra rutas anidadas bajo `DashboardLayout`. `activeBase = location.pathname.split('/')[1]` (DashboardLayout.tsx:40) → `/budget/...` da `activeBase='budget'`. Se debe contemplar `t('budget.title')` como fallback del header o usar `usePageTitleStore`.
- **Patrón backend**: capas `models` (structs con tags json), `storage` (SQL crudo), `services` (lógica), `api` (handlers + `routes.go` con `authMiddleware`). Se replica para cada entidad nueva.
- **Patrón multi-moneda**: `debts` (SPEC-054) usa `currency_id` FK a `currencies` y expone `CurrencyCode`; los totales se agrupan por moneda (análisis de deudas por mes). El módulo de presupuesto sigue el mismo enfoque.
- **Migraciones**: última migración es `0034` (SPEC-092). Las nuevas serán `0035`+ (`NNNN_descripcion.up.sql`/`.down.sql`).
- **Patrón de UI**: cards + `CardMenu` (3 puntos) + `CreateMenu` en header + `EmptyCard` cuando no hay registros (`HomesPage.tsx` es referencia). Formularios con tokens del tema (SPEC-060). Reordenar con `SortableGrid` (SPEC-061).
- **YNAB modelo**: `available = assigned - activity`; sin carryover en v1 se simplifica a `assigned - activity` por mes, con aviso explícito en UI.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Submenú colapsable** en sidebar (como Pensión) | Consistente con patrón existente, espacio para Vista mensual + Transacciones + futuro | Un nivel más de navegación | ✅ Seleccionada |
| Ítem directo "Presupuesto" con tabs internas | Un click a la vista | Tabs internas no usadas en el proyecto para secciones grandes | ❌ Rechazada |
| Edición de categorías como submenú separado | Acceso directo | El usuario pidió que vaya dentro de la vista mensual | ❌ Rechazada (decisión usuario) |
| Carryover YNAB en v1 | Fiel a YNAB | Complejidad de cálculo de saldo arrastrado; el usuario pidió simplificado | ❌ Rechazada en v1 (Fase 5) |
| Moneda única default | Más simple | Deudas ya soporta multi-moneda; incoherente | ❌ Rechazada (multi-moneda) |
| Cuentas vinculadas a instituciones | Reutiliza datos | Usuario pidió independiente | ❌ Rechazada (decisión usuario) |
| `type` fijo por categoría (fijo/variable) | Simple de entender | Rígido; el requerimiento pide flexibilidad | ❌ Rechazada (el tipo vive en Assignment/RecurringRule) |
| `budget_months` + `assignments` (tabla de meses y asignaciones) | Desnormaliza y simplifica cálculos, indexable | Tabla extra | ✅ Seleccionada (fiel al requerimiento) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El tipo de presupuesto (fijo/variable) vive en el `Assignment`/`RecurringRule`, no en la categoría.
- **Contexto**: El requerimiento original plantea "fixed" vs "variable" por grupo/categoría, pero la forma más flexible (y pedida por el usuario) es que una categoría pueda tener asignaciones recurrentes y puntuales según el mes.
- **Decisión**: `assignments.source` ∈ `('recurring'|'one_time')` y `recurring_rules` cubren lo "fijo". Sin campo booleano en `categories`.
- **Consecuencias**: Modelo más simple, sin decisión temprana de tipo por categoría. Las reglas recurrentes generan assignments automáticos al materializar un mes.

**ADR-002**: `available = assigned - activity` por mes, **sin carryover** en v1.
- **Contexto**: YNAB arrastra el saldo no gastado al mes siguiente. El usuario eligió simplificado para v1.
- **Decisión**: El `available` de cada categoría se resetea cada mes. La UI muestra el aviso "El saldo se resetea cada mes". Diseño preparado para agregar carryover en Fase 5 (columna de arrastre en `budget_months` o cálculo incremental).
- **Consecuencias**: Cálculos más simples y predecibles. Puede sorprender a usuarios de YNAB → mitigado con el aviso en UI.

**ADR-003**: Multi-moneda por `currency_id` (patrón de Deudas), con cálculos agrupados por moneda.
- **Contexto**: El proyecto ya maneja varias monedas en `currencies`; Deudas (SPEC-054) las usa. El presupuesto puede tener categorías en distintas monedas.
- **Decisión**: `accounts.currency_id`, `assignments.currency_id`, `transactions.currency_id`. Los totales ("plata sin asignar", columnas) se calculan y muestran agrupados por moneda.
- **Consecuencias**: La vista mensual muestra totales por moneda. Complejidad adicional menor en las queries de agregación (GROUP BY moneda).

**ADR-004**: Cuentas independientes (sin FK a instituciones).
- **Contexto**: El usuario decidió que las cuentas del presupuesto son propias (banco, tarjeta, efectivo) y no se vinculan a instituciones.
- **Decisión**: `accounts` solo con `currency_id`; sin `institution_id`.
- **Consecuencias**: Menos acoplamiento; el módulo es autocontenido.

**ADR-005**: Los grupos/categorías se gestionan desde la vista mensual (sección de configuración), no como submenú.
- **Contexto**: El usuario respondió que la edición de presupuesto va dentro de la vista mensual.
- **Decisión**: Un botón/ícono "Configurar" en `/budget` abre el gestor de grupos/categorías (modal o vista embebida).
- **Consecuencias**: Menos ítems en el sidebar; la configuración queda a un paso de la vista principal.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[React SPA - /budget, /budget/transacciones]
        │  fetch (sesión cookie)
        ▼
[Go net/http - routes.go, authMiddleware]
        │
        ▼
[services: budget.go (cálculos), category_group.go, category.go,
           account.go, recurring_rule.go, transaction.go]
        │
        ▼
[storage: SQL crudo SQLite (modernc.org/sqlite)]
        │
        ▼
[SQLite DB (WAL)]
```

### 4.2 Componentes

#### 4.2.1 Backend

- **`internal/models/`**: `account.go`, `category_group.go`, `category.go`, `recurring_rule.go`, `budget_month.go`, `assignment.go`, `transaction.go` — structs con tags json, sin dependencias de DB/HTTP.
- **`internal/storage/`**: `account.go`, `category_group.go`, `category.go`, `recurring_rule.go`, `budget_month.go`, `assignment.go`, `transaction.go` — CRUDs SQL y las queries de agregación mensual.
- **`internal/services/`**: `budget.go` (lógica de `assigned/activity/available` por mes+categoría+moneda, materialización de meses, generación de recurrentes), más los services CRUD por entidad.
- **`internal/api/`**: handlers nuevos + registro en `routes.go`. Todos bajo `authMiddleware`.

#### 4.2.2 Frontend

- **`frontend/src/pages/BudgetPage.tsx`**: vista mensual (`/budget`). Selector de mes, tabla por grupos, totales por moneda, modal de asignación, sección de configuración de grupos/categorías.
- **`frontend/src/pages/BudgetTransactionsPage.tsx`**: grilla de transacciones (`/budget/transacciones`) + alta rápida inline.
- **`frontend/src/pages/BudgetAccountsPage.tsx`** (si se decide como página) o gestión de cuentas dentro de `/budget/transacciones` (P2; en v1 CRUD de cuentas accesible desde la grilla).
- **`frontend/src/components/BudgetCategoryModal.tsx`**: modal de asignación por categoría/mes (monto, recurrente sí/no, historial de actividad).
- **`frontend/src/components/BudgetGroupsModal.tsx`**: gestor de grupos/categorías (crear/editar/archivar/ordenar).
- **`frontend/src/components/Sidebar.tsx`**: agregar submenú "Presupuesto" colapsable (patrón Pensión).
- **`frontend/src/App.tsx`**: rutas `budget`, `budget/transacciones` (y sub-rutas de formularios si aplica).
- **`frontend/src/api.ts`**: métodos de API para las entidades nuevas.
- **`frontend/public/i18n/{es,en}.json`**: namespace `budget` + `menu.budget`.

### 4.3 Modelo de datos

Migraciones `0035`+ (siguen convención `NNNN_descripcion.{up,down}.sql`). Montos como `REAL` (patrón de `debts`).

```sql
-- 0035_create_budget_core.up.sql

CREATE TABLE IF NOT EXISTS category_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    icon TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_group_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    icon TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    target_amount REAL,
    target_type TEXT CHECK (target_type IN ('monthly','by_date') OR target_type IS NULL),
    target_date TEXT,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_group_id) REFERENCES category_groups(id)
);

CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('checking','savings','credit_card','cash')),
    currency_id INTEGER NOT NULL,
    starting_balance REAL NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE TABLE IF NOT EXISTS budget_months (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
    UNIQUE(year, month)
);

CREATE TABLE IF NOT EXISTS recurring_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL,
    amount REAL NOT NULL DEFAULT 0,
    frequency TEXT NOT NULL DEFAULT 'monthly' CHECK (frequency = 'monthly'),
    start_month TEXT NOT NULL,          -- 'YYYY-MM'
    end_month TEXT,                     -- nullable 'YYYY-MM'
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE TABLE IF NOT EXISTS assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    budget_month_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    currency_id INTEGER NOT NULL,
    amount REAL NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'one_time' CHECK (source IN ('recurring','one_time')),
    recurring_rule_id INTEGER,
    UNIQUE(budget_month_id, category_id, currency_id),
    FOREIGN KEY (budget_month_id) REFERENCES budget_months(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    FOREIGN KEY (recurring_rule_id) REFERENCES recurring_rules(id)
);

CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    category_id INTEGER,               -- NULL = sin categorizar
    currency_id INTEGER NOT NULL,
    date TEXT NOT NULL,                -- 'YYYY-MM-DD'
    payee TEXT,
    memo TEXT,
    outflow REAL NOT NULL DEFAULT 0,
    inflow REAL NOT NULL DEFAULT 0,
    cleared INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE INDEX IF NOT EXISTS idx_categories_group ON categories(category_group_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_category_date ON transactions(category_id, date);
CREATE INDEX IF NOT EXISTS idx_assignments_month_category ON assignments(budget_month_id, category_id);
CREATE INDEX IF NOT EXISTS idx_recurring_rules_category ON recurring_rules(category_id);
```

**Notas**:
- `assignments` es única por `(budget_month_id, category_id, currency_id)` — asignar dos veces en el mismo mes suma o edita, no duplica.
- `activity` de categoría/mes = `SUM(outflow - inflow)` de `transactions` filtrado por `category_id` y rango de fecha del mes (y `deleted_at IS NULL`).
- `available` = `assigned - activity` (sin carryover en v1).
- `budget_months` se crea on-demand al materializar un mes (fecha actual o navegación).
- Las transacciones pueden vincularse a una categoría incluso si luego se archiva (soft delete): el historial se conserva.

### 4.4 APIs / Contratos

Todas bajo `authMiddleware`. Prefijo `/api/budget`.

#### Endpoint: `GET /api/budget/months/{year}/{month}`

Vista mensual completa: grupos con sus categorías y por cada una `assigned`, `activity`, `available` (por moneda), más totales por moneda y "plata sin asignar".

**Response 200**:
```json
{
  "year": 2026,
  "month": 9,
  "currency_totals": [
    { "currency_id": 1, "code": "NIO", "total_assigned": 1000.0, "total_income": 0.0, "unassigned": -200.0 }
  ],
  "groups": [
    {
      "id": 1, "name": "Necesidades", "icon": "home",
      "categories": [
        {
          "id": 1, "name": "Alquiler", "icon": "home",
          "target_amount": 300.0,
          "assigned": { "1": 300.0 },
          "activity": { "1": 300.0 },
          "available": { "1": 0.0 },
          "recurring_rule": { "id": 2, "amount": 300.0, "active": true }
        }
      ]
    }
  ]
}
```

#### Endpoint: `GET /api/budget/category-groups`

**Response 200**: `[{ "id", "name", "icon", "sort_order", "categories": [{ "id", "name", "icon", "sort_order", "target_amount", "target_type", "target_date", "archived" }] }]`

#### Endpoint: `POST/PUT/DELETE /api/budget/category-groups[/{id}]`

CRUD de grupos. `DELETE` archiva/borra según tenga categorías (ver REQ-004).

#### Endpoint: `POST/PUT/DELETE /api/budget/categories[/{id}]`

CRUD de categorías. `DELETE` con transacciones/assignments → archiva (soft delete).

#### Endpoint: `GET/POST/PUT/DELETE /api/budget/accounts[/{id}]`

CRUD de cuentas. **Response** `{ "id", "name", "type", "currency_id", "currency_code", "starting_balance" }`.

#### Endpoint: `PUT /api/budget/months/{year}/{month}/assignments/{category_id}`

Asignar (crea o actualiza el assignment del mes/categoría). Body: `{ "amount": 200.0, "currency_id": 1, "source": "one_time|recurring", "recurring_rule": { "active": true } }`. Si `source=recurring` y no existe regla, la crea desde ese mes.

**Response 200**: `{ "id", "budget_month_id", "category_id", "currency_id", "amount", "source", "recurring_rule_id" }`

#### Endpoint: `POST /api/budget/months/{year}/{month}/materialize`

Genera los `budget_months` + `assignments` derivados de reglas recurrentes activas para el mes indicado (idempotente). Se llama on-demand al abrir un mes.

**Response 200**: `{ "month_id": 12, "created_assignments": 5 }`

#### Endpoint: `GET/POST/PUT/DELETE /api/budget/transactions[/{id}]`

CRUD de transacciones. **POST body**: `{ "account_id", "category_id" (nullable), "currency_id", "date", "payee", "memo", "outflow", "inflow", "cleared" }`.

#### Endpoint: `GET /api/budget/transactions?year=YYYY&month=MM`

Grilla del mes (estilo YNAB): `[{ "id", "account_id", "account_name", "category_id", "category_name", "currency_id", "currency_code", "date", "payee", "memo", "outflow", "inflow", "cleared" }]`.

#### Endpoint: `PUT/DELETE /api/budget/recurring-rules/{id}`

Editar monto (afecta desde el mes actual en adelante) o pausar/reanudar (`active`). **No** toca assignments de meses pasados.

### 4.5 Dependencias

- **Internas**: `internal/api/routes.go` (registro de rutas), `frontend/src/App.tsx` (rutas), `frontend/src/components/Sidebar.tsx` (submenú), `frontend/src/api.ts`, `frontend/public/i18n/{es,en}.json`, `frontend/src/index.css` (tokens ya existentes), mecanismo de migraciones existente.
- **Externas**: ninguna nueva. Go stdlib + `modernc.org/sqlite` (ya en `go.mod`). Frontend: React + Tailwind (ya presentes), `react-router-dom`, `SortableGrid`/dnd existente si se usa drag & drop.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: El sidebar muestra "Presupuesto" como submenú colapsable con "Vista mensual" y "Transacciones"; `menu.budget` está en `es.json` y `en.json` y se sirve tras `npm run build`.
- [ ] CA-002: Se puede crear un grupo de categorías, luego categorías dentro de él (con ícono y opcional meta), y verlas agrupadas en la vista mensual.
- [ ] CA-003: Se puede asignar un monto a una categoría en el mes actual; aparece en la columna "Asignado" y en "Disponible" (`assigned - activity`).
- [ ] CA-004: Al registrar una transacción de `outflow` en una categoría del mes, la columna "Actividad" y "Disponible" se actualizan automáticamente (sin recargar manualmente o tras refresh).
- [ ] CA-005: Se puede crear una regla recurrente desde una asignación; al navegar a otro mes (o materializarlo), el assignment se genera automáticamente. Editar/pausar la regla no modifica meses pasados.
- [ ] CA-006: El available se resetea cada mes (sin carryover) y la UI muestra el aviso "El saldo se resetea cada mes".
- [ ] CA-007: Se pueden crear cuentas de los 4 tipos con su moneda y balance inicial.
- [ ] CA-008: Se puede eliminar (archivar) una categoría con transacciones; desaparece de la vista mensual pero el historial de transacciones se conserva.
- [ ] CA-009: Las transacciones sin categoría aparecen como "Sin categorizar" y no rompen los totales del resto.
- [ ] CA-010: La vista mensual muestra totales "plata sin asignar" por moneda cuando hay categorías en más de una moneda.
- [ ] CA-011: Asignar dos veces a la misma categoría en el mismo mes no duplica filas; suma/edita el assignment.
- [ ] CA-012: La grilla de transacciones permite alta rápida (cuenta, fecha, payee, categoría, outflow/inflow, cleared) y filtra por mes.
- [ ] CA-DARK: Todos los `input`/`select`/`textarea` nuevos usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (SPEC-060).
- [ ] CA-SELECT: Todos los dropdowns (moneda, cuenta, categoría, tipo de cuenta) usan el componente custom `Select` de `frontend/src/components/Select.tsx`, nunca un `<select>` nativo del sistema (SPEC-004 REQ-024).
- [ ] CA-BACK: Las sub-rutas de formularios con lista padre (si se implementan como rutas) se registran en `BACK_ROUTES` con flecha atrás en el header; no hay links "← Título" en contenido (SPEC-063).

### 5.2 No funcionales

- [ ] CA-NF-001: La vista mensual usa agregación por SQL (GROUP BY) — sin N+1 — y responde en < 300ms con datos típicos en iHost.
- [ ] CA-NF-002: Migraciones `0035`+ aplican y revierten limpiamente (`up` y `down`), con índices creados.
- [ ] CA-NF-003: Sin dependencias externas nuevas en Go; build multi-arch (`linux/amd64,linux/arm/v7,linux/arm64`) intacto.

### 5.3 Testing

- **Unit tests**: cálculo de `assigned/activity/available` por mes y categoría (incl. monedas mixtas, sin categoría, categoría archivada); lógica de materialización de meses y generación de recurrentes; regla de "editar recurrente no afecta meses pasados".
- **Integration tests**: CRUD de cada entidad vía handlers; impacto de crear/borrar transacción en `activity`; soft delete de categoría con historial conservado; endpoint de vista mensual con fixture multi-moneda.
- **E2E tests**: flujo completo en local: crear grupo → categoría → asignar → registrar transacción → ver disponible actualizado → crear recurrente → cambiar de mes y ver assignment generado.
- **Carga/Performance**: query de la vista mensual con ~50 categorías y ~500 transacciones en una DB de prueba; medir tiempo en local.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migraciones `0035`+ + modelos + storage CRUD de `category_groups`, `categories`, `accounts` | 1-2 días | Ninguna |
| 2 | Handlers + rutas de CRUD de grupos/categorías/cuentas; UI sidebar + página de vista mensual base (tabla, empty state, selector de mes) | 2-3 días | Fase 1 |
| 3 | `budget_months` + `assignments` (asignación manual, modal, cálculo `assigned/activity/available` por moneda, totales) | 2-3 días | Fase 2 |
| 4 | `recurring_rules` (crear desde asignación, pausar/editar, materialización de meses) | 2 días | Fase 3 |
| 5 | `transactions` (CRUD, alta rápida, impacto en activity, grilla) + CRUD de cuentas en UI | 2-3 días | Fase 2 |
| 6 | Avísos YNAB (overfunded, sin usar), archivar categorías, refinamiento darkmode, tests | 1-2 días | Fases 3-5 |

### 6.2 Milestones

1. **MVP (Fases 1-3)**: categorías + cuentas + vista mensual con asignación manual y cálculos. Primer PR ejecutable.
2. **V1.0 (Fases 4-5)**: recurrencia + transacciones completas.
3. **V1.1 (Fase 6 + P2)**: avisos, archivo de categorías, metas, polished.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Cálculos de `available` inconsistentes con transacciones borradas/editadas | Media | Alto | Recalcular siempre por agregación (nunca almacenar `available`); tests unitarios de los tres estados |
| Mono-moneda vs multi-moneda: totales confusos | Media | Medio | Totales agrupados por moneda (patrón de Deudas); columna moneda visible en la vista |
| Categoría archivada con transacciones: romper joins | Media | Medio | Joins incluyen `deleted_at IS NULL` solo para listado; transacciones conservan `category_id` directo |
| Drag & drop de orden complejiza la vista | Baja | Bajo | P2: alternativa con controles de orden simples (flechas) |
| Generación de recurrentes duplicada al navegar meses | Media | Medio | `materialize` idempotente (UNIQUE en assignments) + `budget_months` creado on-demand |
| Crecimiento de tabla `transactions` en iHost | Baja | Bajo | Índices por `(date)` y `(category_id, date)`; retención personal (≈ cientos/mes) despreciable en disco |

## 8. Notas y Referencias

- Requerimiento fuente: `/home/paulomcnally/Downloads/budget-system-spec.md` (estilo YNAB).
- Patrones de referencia en repo: SPEC-054 (deudas, multi-moneda), SPEC-060 (inputs darkmode), SPEC-061 (drag & drop de homes), SPEC-063 (BACK_ROUTES), SPEC-066 (worktrees), SPEC-092 (estructura de spec reciente).
- `frontend/src/components/Sidebar.tsx` (submenú colapsable Pensión), `frontend/src/App.tsx` (rutas), `internal/api/routes.go` (registro de endpoints), `migrations/` (convención de migraciones).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-27 | opencode | Creación inicial de la especificación (requerimiento relevado con usuario: menú Presupuesto submenú, /budget, fases 1-4, sin carryover, multi-moneda, cuentas independientes) |
| 2026-09-27 | opencode | Feedback usuario durante evaluación: edición directa de grupos/categorías desde la vista mensual (dropdown de 3 puntos por grupo con Agregar categoría/Editar/Eliminar + botón Nuevo grupo) en vez de modal central de configuración; fix z-index del IconPickerModal sobre los formularios; fix scroll horizontal del modal de iconos; **los dropdowns usan el componente custom `Select`, nunca `<select>` nativo (SPEC-004 REQ-024)**. |
| 2026-09-27 | opencode | Fix: `MonthView` devuelve `[]` (no `null`) en `groups`/`currency_totals` para el empty state; defensiva en frontend (`|| []`). Release: merge a main + cierre de issue. |