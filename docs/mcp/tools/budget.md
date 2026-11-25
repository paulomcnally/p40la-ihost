---
title: "Tools de presupuesto"
sidebar_position: 14
description: "Referencia de budget_categories_list, budget_accounts_list, budget_transactions_create, budget_transactions_list y budget_month_view"
---

# Tools de presupuesto

Las cinco tools —cuatro de lectura y una de escritura— operan sobre el **presupuesto estilo YNAB** del sistema (`SPEC-093`): `budget_categories_list` y `budget_accounts_list` dan el contexto necesario para clasificar un gasto; `budget_transactions_create` registra el gasto, resuelve la categoría y la cuenta, deduce el mes y la moneda y deduplica antes de escribir; `budget_transactions_list` y `budget_month_view` permiten revisar lo registrado y el estado mensual.

Todas las tools usan el prefijo `budget_` para evitar la colisión de nombres con las **cuentas bancarias** (`bank_accounts_*`). Son conceptos distintos y viven en tablas distintas: las cuentas del presupuesto (`accounts`) pertenecen al módulo de presupuesto y son el origen de cada transacción; las cuentas bancarias (`bank_accounts`) pertenecen al módulo de banca. Una tool de presupuesto nunca devuelve cuentas bancarias y una tool de banca nunca devuelve cuentas del presupuesto.

El MCP no accede a SQLite: cada tool llama a la API HTTP existente (`/api/budget/category-groups`, `/api/budget/accounts`, `/api/budget/transactions` y `/api/budget/months/{year}/{month}`). Las tools validan la **forma** de la entrada; las **reglas de dominio** las valida el backend. La escritura se envía con origen `mcp` (`X-P40LA-Source`).

El flujo previsto es el del usuario que reenvía una factura en PDF a un LLM externo (Claude Code u otro cliente MCP). **El PDF lo extrae el LLM**: la aplicación no recibe ni procesa PDFs, no hace OCR y no interpreta imágenes. El LLM convierte la factura en `date`, `amount`, `payee` y una categoría, y usa estas tools para consultar el catálogo y registrar el gasto.

`budget_transactions_create` registra **solo gastos** (`outflow`). Los ingresos (`inflow`) no forman parte de este toolset.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `budget_categories_list` | Lectura | 1 | Grupos de categorías con sus categorías activas, ícono y objetivo opcional |
| `budget_accounts_list` | Lectura | 1 | Cuentas activas del presupuesto con su moneda y saldo calculado |
| `budget_transactions_list` | Lectura | 1 | Transacciones de un mes del presupuesto |
| `budget_month_view` | Lectura | 1 | Vista mensual por grupo y categoría: asignado, actividad y disponible por moneda |
| `budget_transactions_create` | Escritura | 3–4 | Registra un gasto: resuelve categoría y cuenta, deduplica y respeta `dry_run` |

`budget_transactions_create` lee categorías, cuentas y las transacciones del mes para resolver y deduplicar, y suma el `POST` cuando escribe: tres lecturas en `dry_run` y cuatro llamadas en el alta.

Las cuatro tools de lectura son de **solo lectura**: no escriben, no dependen de la compuerta y no requieren permisos adicionales.

## Compuerta de escritura

Las lecturas no dependen de la compuerta. `budget_transactions_create` sí, pero solo cuando `dry_run=false`:

| Transporte | Requisito |
|---|---|
| stdio | `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

La compuerta se evalúa **antes** de validar la fecha y el monto: con `dry_run=false` y las escrituras apagadas, la tool responde `writes_disabled` sin llamar a la API, aunque la entrada tenga otros errores.

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true (y P40LA_MCP_REMOTE_WRITES=true en modo HTTP), o repetí con dry_run=true para previsualizar."
}
```

`dry_run=true` no pasa por la compuerta: la previsualización funciona siempre. Ver [http.md](../http.md) y [security.md](../security.md).

## Cuentas del presupuesto y cuentas bancarias

| Concepto | Namespace MCP | Tabla | Endpoint |
|---|---|---|---|
| Cuenta del presupuesto | `budget_accounts_list`, `budget_transactions_create` | `accounts` | `/api/budget/accounts` |
| Cuenta bancaria | `bank_accounts_search`, `bank_accounts_create`, `bank_accounts_update` | `bank_accounts` | `/api/banks/{id}/accounts` |

La cuenta del presupuesto es el origen de cada transacción y define su moneda. La cuenta bancaria es un dato del módulo de banca y no se usa en las tools de presupuesto. Si el usuario pide «ver sus cuentas» en un contexto de gastos, la tool correcta es `budget_accounts_list`.

## Flujo recomendado para un agente

1. **Contexto, una vez por sesión.** Llamar `budget_categories_list` y `budget_accounts_list` al inicio, guardar los IDs y mapear el concepto del gasto a una categoría existente. No repetir estos listados en cada gasto salvo que el catálogo haya cambiado.
2. **Previsualizar.** Llamar `budget_transactions_create` con los datos extraídos del documento y `dry_run=true` (el valor por defecto). Mostrar al usuario la cuenta, la categoría, la fecha, el monto y la moneda resueltos, y si se detectó un duplicado.
3. **Confirmar.** Si el plan es correcto, el usuario confirma y se repite la llamada con `dry_run=false`. La compuerta `P40LA_MCP_ALLOW_WRITES=true` debe estar activa; si el transporte es HTTP, también `P40LA_MCP_REMOTE_WRITES=true`.
4. **Verificar.** Reportar el `id` de la transacción creada o el registro existente. Para revisar el mes, usar `budget_transactions_list`; para el estado del presupuesto, `budget_month_view`.

Si la extracción del documento es dudosa, el paso 2 es el momento de corregir: el monto, la fecha y la categoría se muestran resueltos antes de escribir. La tool no decide la categoría por su cuenta; la asociación la propone el LLM y la confirma el usuario.

## `budget_categories_list`

Devuelve los grupos de categorías del presupuesto con sus **categorías activas**. Es la superficie que el LLM necesita para traducir un concepto («comida», «transporte») a un `category_id` o a un nombre exacto. Solo lectura.

### Entrada

Sin parámetros.

### Ejemplo

```json
{}
```

### Salida

```json
{
  "groups": [
    {
      "id": 2,
      "name": "Necesidades",
      "icon": "cart",
      "categories": [
        {
          "id": 5,
          "category_group_id": 2,
          "name": "Comida",
          "icon": "food",
          "sort_order": 1,
          "target_amount": 5000,
          "target_type": "monthly",
          "account_id": 3
        }
      ]
    }
  ],
  "count": 1
}
```

### Reglas

- **Solo activas**: las categorías eliminadas o inactivas no se devuelven. Un `category_id` que no aparece en la respuesta no es utilizable en `budget_transactions_create`.
- **`count`** es la cantidad de grupos devueltos, no la de categorías.
- **Campos opcionales**: `target_amount`, `target_type` y `target_date` aparecen solo si la categoría tiene objetivo definido. `account_id` aparece solo si la categoría tiene una cuenta vinculada (`categories.account_id`, `SPEC-093`); cuando está presente, `budget_transactions_create` la usa como cuenta por defecto si no se envía `account_id`. `service_ids` y `service_names` aparecen solo si la categoría está vinculada a servicios (`SPEC-094`/`SPEC-096`), y sirven para entender el criterio de clasificación sin cambiar la resolución del gasto.
- **`icon`** es un identificador del tema visual, no una ruta ni un archivo.
- Los grupos se devuelven ordenados por su `sort_order` y las categorías dentro de cada grupo también; el LLM no necesita reordenarlos.
- La respuesta no incluye montos asignados ni gastados: eso se consulta con `budget_month_view`.

### Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `budget_accounts_list`

Devuelve las **cuentas activas del presupuesto**, con su moneda y su saldo calculado. Es el segundo dato que el LLM necesita: toda transacción exige una cuenta y la moneda se deriva de ella. Solo lectura.

### Entrada

Sin parámetros.

### Ejemplo

```json
{}
```

### Salida

```json
{
  "accounts": [
    {
      "id": 3,
      "name": "Cuenta Demo",
      "type": "checking",
      "currency_id": 1,
      "currency_code": "NIO",
      "starting_balance": 1000,
      "balance": 850
    }
  ],
  "count": 1
}
```

### Reglas

- **Solo activas**: las cuentas eliminadas o inactivas no se devuelven y no pueden usarse como `account_id`.
- **`type`** es `checking`, `savings`, `credit_card` o `cash`.
- **`balance`** es un valor calculado: `starting_balance` más las transacciones (`inflow` y `outflow`) más las transferencias entre cuentas. No es un saldo editable ni un campo de la tabla.
- **`starting_balance`** es el saldo inicial registrado al crear la cuenta.
- **`count`** es la cantidad de cuentas devueltas.
- La moneda de la cuenta (`currency_id`, `currency_code`) es la que hereda toda transacción creada sobre ella: no hay parámetro de moneda en `budget_transactions_create`.
- Esta tool **no** devuelve cuentas bancarias (`bank_accounts`). Para esas, usar `bank_accounts_search`.

### Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `budget_transactions_list`

Lista las transacciones de un mes del presupuesto. Es la vía para verificar lo registrado, detectar duplicados después de una escritura y dar contexto al usuario. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `year` | integer | Sí | Año del presupuesto; rango `1900..3000` |
| `month` | integer | Sí | Mes del presupuesto; rango `1..12` |

### Ejemplo

```json
{
  "year": 2026,
  "month": 3
}
```

### Salida

```json
{
  "year": 2026,
  "month": 3,
  "count": 1,
  "transactions": [
    {
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
  ]
}
```

### Reglas

- **Rango de fechas**: el backend filtra por el mes completo (`date` mayor o igual al día 1 y menor o igual al último día del mes). No se filtra por cuenta ni por categoría en esta tool.
- **`outflow` e `inflow`**: un gasto llega como `outflow` mayor que `0` e `inflow` en `0`; un ingreso, al revés. Las tools de presupuesto solo crean `outflow`.
- **`cleared`** indica si la transacción fue conciliada con el estado de cuenta en la UI.
- **`account_name`** y **`category_name`** son datos derivados que la API resuelve para la respuesta; `category_id` puede ser `null` o ausente en transacciones sin categoría.
- **`source_bill_id`** aparece solo cuando la transacción se generó desde una factura de servicio; un gasto cargado por el MCP no lo trae.
- **`count`** es la cantidad de transacciones del mes devueltas. No hay paginación: el volumen por mes es acotado.
- Un mes sin transacciones devuelve `count: 0` y `transactions: []`; no es un error.
- `year` fuera de `1900..3000` o `month` fuera de `1..12` devuelve `invalid_request` sin llamar a la API.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `year` fuera de `1900..3000` o `month` fuera de `1..12`; no llama a la API |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `budget_month_view`

Devuelve la vista mensual del presupuesto: totales por moneda y, por grupo y categoría, cuánto se asignó, cuánto se movió y cuánto queda disponible. Es la foto que el usuario ve en la grilla del presupuesto. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `year` | integer | Sí | Año del presupuesto; rango `1900..3000` |
| `month` | integer | Sí | Mes del presupuesto; rango `1..12` |

### Ejemplo

```json
{
  "year": 2026,
  "month": 3
}
```

### Salida

```json
{
  "year": 2026,
  "month": 3,
  "currency_totals": [
    {
      "currency_id": 1,
      "code": "NIO",
      "symbol": "C$",
      "total_income": 0,
      "total_assigned": 5000,
      "unassigned": -5000
    }
  ],
  "groups": [
    {
      "id": 2,
      "name": "Necesidades",
      "icon": "cart",
      "categories": [
        {
          "id": 5,
          "name": "Comida",
          "assigned": { "1": 5000 },
          "activity": { "1": 1200 },
          "available": { "1": 3800 }
        }
      ]
    }
  ]
}
```

### Reglas

- **`assigned`, `activity` y `available` son mapas `currency_id → monto`**, con el ID de la moneda como clave de texto. Una categoría puede tener más de una moneda si sus transacciones usan cuentas con monedas distintas. Nunca se suman monedas distintas en un solo número.
- Cada categoría de la vista incluye además sus campos de catálogo (`category_group_id`, `sort_order`, `target_amount`, `target_type`, `target_date`, `account_id`, `service_ids` y `service_names` cuando existen) y `recurring_rule` cuando la categoría tiene una regla recurrente asociada.
- **`activity`** es el neto del mes calculado como `outflow − inflow`: un gasto suma en **positivo** y un ingreso resta. Un gasto de `1200` sobre una categoría asignada con `5000` deja `activity` en `1200` y `available` en `3800`.
- **`available`** es `assigned − activity`. Puede ser negativo cuando la actividad supera lo asignado. La vista no arrastra saldo de meses anteriores: cada mes se calcula con sus propias asignaciones y transacciones.
- **`currency_totals`** resume el mes por moneda: `total_income` (ingresos), `total_assigned` (asignado) y `unassigned` (ingresos menos asignado, que puede ser negativo). El `symbol` es el símbolo de presentación de la moneda. Solo aparecen las monedas con ingresos o con montos asignados en el mes.
- Los grupos y categorías siguen el orden del catálogo. Las categorías sin movimientos se devuelven igual, con sus mapas vacíos o en cero, porque la grilla las muestra.
- Un mes sin movimientos devuelve la estructura con los totales en cero; no es un error.
- `year` fuera de `1900..3000` o `month` fuera de `1..12` devuelve `invalid_request` sin llamar a la API.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `year` fuera de `1900..3000` o `month` fuera de `1..12`; no llama a la API |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `budget_transactions_create`

Registra un **gasto** (`outflow`) en una categoría, con una cuenta y una fecha. Resuelve la categoría y la cuenta, deduce el mes de la fecha y la moneda de la cuenta, deduplica contra lo ya registrado y respeta `dry_run` y la compuerta de escritura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `date` | string `YYYY-MM-DD` | Sí | Fecha del gasto; no puede ser futura. Determina el mes del presupuesto |
| `amount` | number | Sí | Monto del gasto (`outflow`); debe ser mayor que `0` |
| `category_id` | integer | Condicional | Categoría destino por ID; alternativa a `category` |
| `category` | string | Condicional | Nombre exacto de una categoría activa; alternativa a `category_id` |
| `account_id` | integer | No | Cuenta del presupuesto; si falta, se usa la de la categoría o la primera activa |
| `payee` | string | No | Comercio o beneficiario; se usa para deduplicar |
| `memo` | string | No | Nota libre del movimiento |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |
| `allow_duplicate` | boolean | No | `false` por defecto: permite crear un duplicado de forma explícita |

Es obligatorio resolver la categoría: se envía `category_id` **o** `category`. Si faltan ambos, no hay nada que asociar y la tool responde `invalid_request`.

### Ejemplo

```json
{
  "date": "2026-03-10",
  "amount": 1200,
  "category": "Comida",
  "payee": "Proveedor Demo",
  "dry_run": true
}
```

### Salida con `dry_run`

`dry_run=true` es el valor por defecto: no escribe ni llama al `POST`. Devuelve el plan resuelto para que el agente lo muestre antes de confirmar.

```json
{
  "status": "planned",
  "dry_run": true,
  "plan": {
    "account_id": 3,
    "account_name": "Cuenta Demo",
    "category_id": 5,
    "category_name": "Comida",
    "currency_id": 1,
    "currency_code": "NIO",
    "date": "2026-03-10",
    "amount": 1200,
    "payee": "Proveedor Demo"
  }
}
```

Si en la previsualización se detecta un duplicado, la respuesta es `status=duplicate` (ver más abajo) y tampoco escribe.

### Salida con escritura

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

La transacción se crea con `inflow: 0` y `cleared: false`. El `id` devuelto es el que la UI muestra y el que permite verificarla con `budget_transactions_list`.

### Salida duplicada

```json
{
  "status": "duplicate",
  "dry_run": false,
  "existing": {
    "id": 9,
    "date": "2026-03-10",
    "category_id": 5,
    "category_name": "Comida",
    "outflow": 1200,
    "payee": "Proveedor Demo"
  },
  "hint": "Ya existe un gasto equivalente el mismo día en esa categoría. Usá allow_duplicate=true solo si es un registro distinto."
}
```

`status=duplicate` **no es un error de transporte**: es el resultado esperado de un alta repetida. La tool responde con el registro existente y **no escribe** una segunda fila.

### Salida con `allow_duplicate`

Con `allow_duplicate=true` la transacción se crea aunque exista una equivalente y la respuesta agrega el aviso para que el agente lo explique:

```json
{
  "status": "created",
  "dry_run": false,
  "warnings": ["possible_duplicate"],
  "transaction": {
    "id": 11,
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

### Salida con categoría ambigua

```json
{
  "error": "ambiguous_category",
  "message": "Más de una categoría activa coincide con el nombre indicado.",
  "status": "ambiguous_category",
  "dry_run": false,
  "candidates": [
    {
      "id": 5,
      "name": "Comida",
      "category_group_id": 2,
      "group_name": "Necesidades"
    },
    {
      "id": 8,
      "name": "Comida",
      "category_group_id": 4,
      "group_name": "Gustos"
    }
  ],
  "hint": "Pasá category_id para elegir una de las candidatas; no se escribió nada.",
  "category_id_hint": "budget_categories_list"
}
```

`ambiguous_category` es un error (`isError=true`), no un resultado de negocio: devuelve las candidatas y **no escribe**. Para desambiguar, se repite la llamada con `category_id`.

### Reglas

- **El mes se deriva de `date`**: no existe un parámetro de mes en la entrada. La fecha del documento determina en qué mes del presupuesto cae el gasto (ADR-001 de `SPEC-163`).
- **La moneda se deriva de la cuenta**: no existe un parámetro de moneda. La transacción hereda `accounts.currency_id` de la cuenta resuelta (ADR-002 de `SPEC-163`).
- **Resolución de la categoría**:
  - por `category_id`, tal cual; un ID inexistente o inactivo devuelve `category_not_found`;
  - por `category`, comparando el nombre **exacto** contra las categorías activas, sin distinguir mayúsculas, acentos ni espacios repetidos;
  - sin coincidencia devuelve `category_not_found`;
  - con más de una coincidencia devuelve `ambiguous_category` con `candidates` y **no escribe**; para desambiguar se repite la llamada con `category_id`;
  - si se envían `category_id` y `category` a la vez, deben apuntar a la misma categoría; si el nombre corresponde a otra, devuelve `invalid_request`.
- **Resolución de la cuenta**, en este orden: `account_id` si se envía; si no, la cuenta vinculada a la categoría (`categories.account_id`); si no, la primera cuenta activa del listado (ordenado por nombre). Un `account_id` que no está entre las cuentas activas devuelve `account_not_found`, igual que un presupuesto sin cuentas.
- **Deduplicación**: antes de escribir, la tool lee las transacciones del mes derivado de `date` y compara `date` + `category_id` + `outflow` + `payee` normalizado. La normalización pasa el `payee` a minúsculas, quita acentos y colapsa los espacios; vacío equivale a vacío. Una coincidencia devuelve `status=duplicate` con el registro `existing` y un `hint`, y no crea una segunda fila.
- **`allow_duplicate`**: `false` por defecto. Con `true` la fila se crea aunque exista una equivalente y la respuesta agrega `warnings: ["possible_duplicate"]`.
- **`dry_run`**: `true` por defecto. Resuelve categoría y cuenta, ejecuta la deduplicación y devuelve `status=planned` con el plan, o `status=duplicate` con el registro existente. No escribe en ningún caso. La compuerta se evalúa solo con `dry_run=false`. El plan incluye cuenta, categoría, moneda, fecha, monto y `payee`, más el `memo` cuando se envió. Con `allow_duplicate=true` y un duplicado detectado, la previsualización no se corta: devuelve `status=planned` con el bloque `duplicate` y `warnings: ["possible_duplicate"]`, para que la decisión quede en el usuario.
- **Solo gastos**: la tool crea `outflow` con `inflow=0`. Los ingresos, las transferencias entre cuentas y la conciliación (`cleared`) quedan fuera del alcance y se hacen desde la UI.
- **`date`**: formato `YYYY-MM-DD` y no futura según `P40LA_TIMEZONE`. Una fecha mal formada o futura devuelve `invalid_date` sin llamar a la API.
- **`amount`**: obligatorio y mayor que `0`. Un valor ausente, cero, negativo o no numérico devuelve `invalid_amount`. Se redondea a dos decimales; la precisión final la define la moneda en el backend.
- **`payee` y `memo`** son texto libre opcional. El `payee` afecta la deduplicación; el `memo` no.
- **Carrera de concurrencia**: la deduplicación es de aplicación, no un índice único del esquema. Dos llamadas simultáneas con el mismo gasto pueden pasar ambas el chequeo y crear dos filas. Es la limitación documentada en ADR-003 de `SPEC-163`; se mitiga usando `budget_transactions_list` para revisar el mes y una clave de idempotencia persistida queda para una spec futura.
- **`X-P40LA-Source: mcp`**: la escritura viaja con el origen `mcp` para la auditoría del backend.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | No se envió `category_id` ni `category`; `category_id` y `category` apuntan a categorías distintas; o la cuenta resuelta no tiene moneda |
| `invalid_date` | `date` sin formato `YYYY-MM-DD` o futura según `P40LA_TIMEZONE`; no llama a la API |
| `invalid_amount` | `amount` ausente, no numérico o menor o igual a `0`; no llama a la API |
| `category_not_found` | El `category_id` no existe entre las categorías activas, o el nombre de `category` no coincide con ninguna |
| `ambiguous_category` | El nombre de `category` coincide con más de una categoría activa; incluye `candidates`, es `isError=true` y no escribe |
| `account_not_found` | No hay ninguna cuenta activa para resolver, o el `account_id` no corresponde a una cuenta existente; incluye un hint |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout, o respuesta con un sobre distinto al esperado |

## Errores comunes

| Código | Cuándo | `hint` |
|---|---|---|
| `writes_disabled` | `dry_run=false` con la compuerta apagada | Definí `P40LA_MCP_ALLOW_WRITES=true`; en modo HTTP también `P40LA_MCP_REMOTE_WRITES=true` |
| `invalid_request` | Entrada con forma inválida, categoría sin identificar o rango de mes fuera de rango | Revisar los campos y el rango de `year`/`month` |
| `invalid_date` | Fecha mal formada o futura | Usar `YYYY-MM-DD` y una fecha no posterior a hoy |
| `invalid_amount` | Monto ausente, no numérico o no positivo | El monto es el `outflow` del gasto y debe ser mayor que `0` |
| `category_not_found` | Categoría inexistente o inactiva | Consultar `budget_categories_list` y usar el `id` devuelto |
| `ambiguous_category` | Varias categorías activas con el mismo nombre | Repetir con `category_id`; vienen las `candidates` |
| `account_not_found` | Sin cuentas activas o `account_id` inválido | Consultar `budget_accounts_list` o crear una cuenta en la UI |
| `auth_failed` | API key inválida, revocada o rotada | Revisar la key y [api-key.md](../api-key.md) |
| `upstream_error` | Fallo de red o respuesta 5xx | Verificar `P40LA_BASE_URL` y `GET /health` |

En las tools de presupuesto no hay `already_exists`: un alta repetida se informa como `status=duplicate`, que es un resultado y no un error. Tampoco hay `ambiguous_reference`: la única ambigüedad por nombre es la de la categoría, y devuelve `ambiguous_category` con las candidatas.
