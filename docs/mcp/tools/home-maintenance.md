---
title: "Tools de mantenimiento del hogar"
sidebar_position: 10
description: "Referencia de home_maintenance_list, home_maintenance_labor_register y home_maintenance_supplies_register"
---

# Tools de mantenimiento del hogar

Las tres tools operan sobre las **reparaciones y mantenimientos** registrados en una vivienda (`home.maintenance`): `home_maintenance_list` consulta el histórico y las dos de registro agregan líneas de **mano de obra** (`labor`) e **insumos** (`supplies`). El `home_id` es el mismo de `/homes/{id}` en la UI.

Las tools de registro resuelven el **mantenimiento del día**, deduplican antes de escribir, respetan `dry_run` (`true` por defecto) y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). `home_maintenance_list` sigue siendo de **solo lectura**: no escribe, no depende de la compuerta y no requiere permisos adicionales.

El MCP no accede a SQLite: cada tool llama a la API HTTP existente (`/api/homes/{id}/maintenances`). Las tools validan la **forma** de la entrada; las **reglas de dominio** las valida el backend. Las escrituras se envían con origen `mcp` (`X-P40LA-Source`).

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `home_maintenance_list` | Lectura | 1 | Histórico de reparaciones de una vivienda, con filtros de fecha, tipo y texto, y totales por moneda |
| `home_maintenance_labor_register` | Escritura | 1–2 | Registra una línea de mano de obra; reutiliza o crea el mantenimiento del día |
| `home_maintenance_supplies_register` | Escritura | 1–2 | Registra una línea de insumos con cantidad y unidad; reutiliza o crea el mantenimiento del día |

## `home_maintenance_list`

Lista los mantenimientos de una vivienda, con filtros opcionales, y devuelve cada registro con su desglose de mano de obra e insumos. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `year` | integer | No | Filtra por año del mantenimiento; rango `1900..9999`. Mutuamente excluyente con `date` |
| `date` | string `YYYY-MM-DD` | No | Filtra por fecha exacta. Mutuamente excluyente con `year` |
| `kind` | string | No | `corrective` (correctivo) o `preventive` (preventivo) |
| `item_type` | string | No | `labor` o `supplies`: solo los mantenimientos con al menos una línea de ese tipo |
| `query` | string | No | Texto en la descripción del mantenimiento o de sus líneas; máximo 100 caracteres |
| `include_items` | boolean | No | Incluye las líneas de cada mantenimiento en la respuesta |

Los filtros se combinan con AND. Sin filtros se devuelve el histórico completo de la vivienda.

### Ejemplo

```json
{
  "home_id": 3,
  "date": "2026-03-10",
  "item_type": "labor",
  "query": "fuga",
  "include_items": true
}
```

### Salida

```json
{
  "home_id": 3,
  "count": 1,
  "truncated": false,
  "maintenances": [
    {
      "id": 1,
      "home_id": 3,
      "performed_at": "2026-03-10",
      "kind": "corrective",
      "area": "plumbing",
      "institution_id": 7,
      "institution_name": "Taller Demo",
      "description": "Reparación demo de fuga",
      "totals_by_currency": [
        {
          "currency_id": 1,
          "currency_code": "NIO",
          "labor_cost": 500.00,
          "supplies_cost": 600.00,
          "total": 1100.00
        }
      ],
      "items": [
        {
          "id": 101,
          "maintenance_id": 1,
          "item_type": "labor",
          "description": "Mano de obra demo",
          "quantity": 1,
          "unit": "unit",
          "amount": 500.00,
          "currency_id": 1,
          "currency_code": "NIO",
          "institution_id": 7,
          "institution_name": "Taller Demo"
        },
        {
          "id": 102,
          "maintenance_id": 1,
          "item_type": "supplies",
          "description": "Insumos demo",
          "quantity": 2,
          "unit": "unit",
          "amount": 600.00,
          "currency_id": 1,
          "currency_code": "NIO",
          "institution_id": 7,
          "institution_name": "Taller Demo"
        }
      ],
      "created_at": "2026-03-10T14:00:00Z",
      "updated_at": "2026-03-10T14:00:00Z"
    }
  ],
  "totals_by_currency": [
    {
      "currency_id": 1,
      "currency_code": "NIO",
      "labor_cost": 500.00,
      "supplies_cost": 600.00,
      "total": 1100.00
    }
  ]
}
```

### Reglas

- `date` y `year` son **mutuamente excluyentes**: enviar ambos devuelve `invalid_request` sin llamar a la API.
- `date` debe tener formato `YYYY-MM-DD`; otro formato devuelve `invalid_request` sin llamar a la API.
- `kind` es `corrective` o `preventive`; cualquier otro valor devuelve `invalid_request` **sin** llamar a la API.
- `year` fuera de `1900..9999` devuelve `invalid_request` sin llamar a la API.
- `item_type` filtra **qué mantenimientos aparecen** (los que tienen al menos una línea de ese tipo). `totals_by_currency` sigue sumando **todas** las líneas de esos mantenimientos con el desglose `labor_cost` / `supplies_cost`: la columna del otro tipo no se anula para no perder el total de la reparación. El `totals_by_currency` de nivel superior aplica la misma regla sobre el conjunto filtrado.
- `query` busca en `description` del mantenimiento y en la de sus líneas; `%`, `_` y `\` se escapan y no se interpretan como patrón. Más de 100 caracteres devuelve `invalid_request`.
- `include_items=true` pide las líneas de cada mantenimiento en la misma llamada. Sin el flag, la respuesta se limita a los mantenimientos y sus totales por moneda.
- La consulta con líneas tiene topes de **100 mantenimientos** y **1.000 líneas**. Si se superan, `truncated=true` y la respuesta sigue siendo válida: los datos devueltos son un subconjunto, no un error.
- `area` describe el sector de la vivienda: `plumbing`, `electrical`, `roof`, `paint`, `carpentry`, `appliances`, `garden`, `cleaning` u `other`.
- `items[].item_type` es `labor` (mano de obra) o `supplies` (insumos). `quantity` y `unit` acompañan a los insumos (`unit`, `m`, `m2`, `kg`, `lb`, `l`, `gal`, `sack`, `box`).
- `totals_by_currency` es por mantenimiento y suma `labor_cost` + `supplies_cost` en `total`, separado por moneda. Nunca se suman monedas distintas.
- `count` es la cantidad de mantenimientos devueltos, no el total del sistema; el `totals_by_currency` de nivel superior suma el conjunto filtrado.
- Los mantenimientos se devuelven ordenados por `performed_at` descendente y, en empate, por `id` descendente.
- Una vivienda inexistente devuelve `home_not_found`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` es `0` o negativo, `year` fuera de `1900..9999`, `kind` distinto de `corrective`/`preventive`, `date` junto con `year`, `date` sin formato `YYYY-MM-DD`, `item_type` inválido o `query` de más de 100 caracteres |
| `home_not_found` | La vivienda no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_maintenance_labor_register`

Registra una línea de **mano de obra** (`item_type=labor`) en una reparación del hogar. Si no recibe `maintenance_id`, resuelve el mantenimiento del mismo día: lo reutiliza si existe uno compatible o lo crea junto con la línea en una sola transacción. Antes de escribir busca líneas equivalentes del mismo día y, si encuentra una, no escribe.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda destino; debe ser mayor que `0` |
| `maintenance_id` | integer | No | Mantenimiento existente; debe pertenecer a `home_id` |
| `performed_at` | string `YYYY-MM-DD` | Condicional | Fecha de la reparación; **obligatoria si no hay `maintenance_id`** y no puede ser futura |
| `area` | string | No | Sector de la vivienda (ver [Reglas](#reglas-1)); default `other` |
| `description` | string | Sí | Descripción de la mano de obra; máximo 200 caracteres |
| `amount` | number | Sí | Monto mayor o igual a `0` en la moneda indicada |
| `currency_id` | integer | Sí | Moneda de la línea |
| `institution_id` | integer | No | Proveedor o contratista que hizo el trabajo |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |
| `allow_duplicate` | boolean | No | `false` por defecto: crear el duplicado de forma explícita |

Con `maintenance_id`, la tool valida que el mantenimiento pertenezca al `home_id` y usa su fecha. Si además enviás `performed_at`, debe coincidir con la del mantenimiento.

### Ejemplo

```json
{
  "home_id": 3,
  "performed_at": "2026-03-10",
  "area": "plumbing",
  "description": "Mano de obra demo",
  "amount": 500.00,
  "currency_id": 1,
  "institution_id": 7,
  "dry_run": true
}
```

### Salida con `dry_run`

`dry_run=true` es el valor por defecto: no escribe ni llama a `POST`. Devuelve el plan y `existing_today` con hasta **50 líneas del mismo día**, para que el agente muestre lo que ya existe antes de confirmar.

```json
{
  "status": "planned",
  "dry_run": true,
  "reused_maintenance": false,
  "planned": {
    "action": "create_maintenance_with_item",
    "performed_at": "2026-03-10",
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 500.00,
    "currency_id": 1,
    "institution_id": 7,
    "area": "plumbing",
    "kind": "corrective",
    "maintenance_description": "Mano de obra demo"
  },
  "existing_today": [
    {
      "maintenance_id": 10,
      "performed_at": "2026-03-10",
      "area": "plumbing",
      "maintenance_description": "Reparación demo",
      "item_id": 101,
      "item_type": "labor",
      "description": "Mano de obra demo",
      "quantity": 1,
      "unit": "unit",
      "amount": 500.00,
      "currency_id": 1,
      "currency_code": "NIO",
      "institution_id": 7,
      "institution_name": "Proveedor Demo"
    }
  ],
  "duplicate": null
}
```

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "reused_maintenance": false,
  "maintenance": {
    "id": 10,
    "home_id": 3,
    "performed_at": "2026-03-10",
    "area": "plumbing"
  },
  "item": {
    "id": 101,
    "maintenance_id": 10,
    "item_type": "labor",
    "description": "Mano de obra demo",
    "amount": 500.00,
    "currency_id": 1
  }
}
```

`reused_maintenance=true` indica que la línea se agregó a un mantenimiento que ya existía; `false`, que el mantenimiento se creó en la misma transacción.

### Salida duplicada

```json
{
  "status": "duplicate",
  "dry_run": false,
  "existing": {
    "maintenance_id": 10,
    "performed_at": "2026-03-10",
    "area": "plumbing",
    "item": {
      "id": 101,
      "item_type": "labor",
      "description": "Mano de obra demo",
      "amount": 500.00,
      "currency_id": 1
    }
  },
  "hint": "Ya existe una línea equivalente el mismo día. Usá allow_duplicate=true solo si es un registro distinto."
}
```

`status=duplicate` **no es un error de transporte**: la tool responde con el registro existente y **sin escribir**. Con `allow_duplicate=true` crea una línea nueva y la respuesta incluye `warnings` para que el agente explique el duplicado.

### Salida con mantenimientos ambiguos

```json
{
  "status": "ambiguous_maintenance",
  "dry_run": false,
  "candidates": [
    {
      "maintenance_id": 10,
      "performed_at": "2026-03-10",
      "area": "plumbing",
      "description": "Reparación demo de fuga",
      "institution_name": "Proveedor Demo"
    },
    {
      "maintenance_id": 11,
      "performed_at": "2026-03-10",
      "area": "plumbing",
      "description": "Revisión demo de fontanería",
      "institution_name": "Proveedor Demo"
    }
  ],
  "hint": "Hay más de un mantenimiento para esa fecha y área. Repetí con maintenance_id para elegir uno."
}
```

`status=ambiguous_maintenance` **no es un error de transporte**: devuelve los candidatos y **no escribe**. Para desambiguar, repetí la llamada con `maintenance_id`.

### Reglas

- **Resolución del mantenimiento**:
  - con `maintenance_id`, valida que el mantenimiento exista y pertenezca al `home_id`;
  - sin `maintenance_id`, busca los mantenimientos del mismo `home_id` y `performed_at`;
  - si hay exactamente uno compatible con `area`, agrega la línea (`reused_maintenance=true`);
  - si no hay ninguno, crea el mantenimiento y la línea en una sola transacción (`reused_maintenance=false`);
  - si hay más de uno, devuelve `ambiguous_maintenance` sin escribir.
- **Deduplicación por día**: antes de escribir compara las líneas del mismo día por `item_type`, descripción normalizada, monto, moneda, proveedor y, en insumos, cantidad y unidad. Una coincidencia devuelve `status=duplicate` con el registro existente. La comparación es de aplicación, no un índice único: dos escrituras simultáneas podrían pasar el chequeo.
- **`allow_duplicate`**: `false` por defecto. Con `true` la línea se crea aunque exista una equivalente y la respuesta agrega `warnings`.
- **`dry_run`**: `true` por defecto. No escribe: consulta, resuelve y devuelve `planned` + `existing_today` (hasta 50 líneas del día). El preflight valida forma y fecha; los prerrequisitos de moneda e institución los valida el backend al escribir.
- **`performed_at`**: obligatoria si no hay `maintenance_id`; formato `YYYY-MM-DD` y no futura según `P40LA_TIMEZONE`. Una fecha futura o mal formada devuelve `invalid_date`.
- **`description`**: obligatoria, máximo 200 caracteres.
- **`amount`**: obligatorio, mayor o igual a `0`; el redondeo y la precisión los define la moneda en el backend.
- **`area`**: `plumbing`, `electrical`, `roof`, `paint`, `carpentry`, `appliances`, `garden`, `cleaning` u `other`; default `other`.
- **`institution_id`**: opcional; debe existir. Un proveedor inexistente devuelve `institution_not_found`.
- No hay índice único de negocio: la tool **no** garantiza idempotencia ante reintentos concurrentes. Para revisar el estado real usá `home_maintenance_list` con `date` y `include_items=true`.
- Máximo acotado: la búsqueda del día revisa hasta 100 mantenimientos y 1.000 líneas; si se supera, la respuesta incluye `truncated=true` y sigue siendo válida.

## `home_maintenance_supplies_register`

Registra una línea de **insumos** (`item_type=supplies`) con cantidad y unidad. La resolución del mantenimiento del día, la deduplicación, `dry_run`, `allow_duplicate`, los topes y la compuerta son idénticos a `home_maintenance_labor_register`.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda destino; debe ser mayor que `0` |
| `maintenance_id` | integer | No | Mantenimiento existente; debe pertenecer a `home_id` |
| `performed_at` | string `YYYY-MM-DD` | Condicional | Fecha de la reparación; **obligatoria si no hay `maintenance_id`** y no puede ser futura |
| `area` | string | No | Sector de la vivienda; default `other` |
| `description` | string | Sí | Descripción de los insumos; máximo 200 caracteres |
| `quantity` | number | No | Cantidad; debe ser mayor que `0`; default `1` |
| `unit` | string | No | `unit`, `m`, `m2`, `kg`, `lb`, `l`, `gal`, `sack` o `box`; default `unit` |
| `amount` | number | Sí | Monto mayor o igual a `0` en la moneda indicada |
| `currency_id` | integer | Sí | Moneda de la línea |
| `institution_id` | integer | No | Proveedor de los insumos |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |
| `allow_duplicate` | boolean | No | `false` por defecto: crear el duplicado de forma explícita |

### Ejemplo

```json
{
  "home_id": 3,
  "performed_at": "2026-03-10",
  "area": "plumbing",
  "description": "Insumos demo",
  "quantity": 2,
  "unit": "unit",
  "amount": 600.00,
  "currency_id": 1,
  "institution_id": 7,
  "dry_run": true
}
```

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "reused_maintenance": true,
  "maintenance": {
    "id": 10,
    "home_id": 3,
    "performed_at": "2026-03-10",
    "area": "plumbing"
  },
  "item": {
    "id": 102,
    "maintenance_id": 10,
    "item_type": "supplies",
    "description": "Insumos demo",
    "quantity": 2,
    "unit": "unit",
    "amount": 600.00,
    "currency_id": 1
  }
}
```

### Reglas

- Aplican las mismas reglas de resolución, deduplicación, `dry_run`, `allow_duplicate` y topes que `home_maintenance_labor_register`.
- `quantity` debe ser mayor que `0`; sin valor se usa `1`.
- `unit` se limita a `unit`, `m`, `m2`, `kg`, `lb`, `l`, `gal`, `sack` y `box`; sin valor se usa `unit`.
- La deduplicación compara también `quantity` y `unit`: dos compras del mismo insumo con distinta cantidad son líneas distintas.

## Compuerta de escritura

Las dos tools de registro comparten la misma compuerta. `dry_run` viene activado por defecto y no requiere permisos de escritura; `home_maintenance_list` es de lectura y funciona siempre.

| Transporte | Lectura y `dry_run=true` | `dry_run=false` |
|---|---|---|
| stdio | Siempre disponible | Requiere `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | Siempre disponible | Requiere `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

Si falta la compuerta, la tool responde sin llamar a la API:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true, o repetí con dry_run=true para previsualizar."
}
```

En stdio el `hint` solo menciona `P40LA_MCP_ALLOW_WRITES`. Ver [configuration.md](../configuration.md) y [security.md](../security.md).

## Errores

Los errores de negocio devuelven `isError=true` y un objeto `{error, message, hint}`. Códigos estables de este dominio:

| Código | Cuándo |
|---|---|
| `invalid_request` | Campos faltantes, `home_id` inválido, `description` vacía o de más de 200 caracteres, `amount` negativo, `quantity` no positiva, `unit` fuera de la lista, `area` inválida, `performed_at` que no coincide con la fecha del `maintenance_id` o `date` junto con `year` en el listado |
| `invalid_date` | `performed_at` no tiene formato `YYYY-MM-DD`, o la fecha es futura |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `home_not_found` | El `home_id` no existe |
| `maintenance_not_found` | El `maintenance_id` no existe o no pertenece al `home_id` |
| `currency_not_found` | La moneda no existe |
| `institution_not_found` | La institución no existe |
| `duplicate` | No es un error de transporte: `status=duplicate` con la línea equivalente del día, sin escribir |
| `ambiguous_maintenance` | No es un error de transporte: `status=ambiguous_maintenance` con los candidatos del día, sin escribir |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

`status=duplicate` y `status=ambiguous_maintenance` llegan con `isError=false` y datos: no son fallas de red ni del servidor, y no hay que reintentar la misma llamada. Para resolverlos, usá `allow_duplicate=true` o elegí un `maintenance_id` de los candidatos.

## Privacidad y datos ocultos

Las tools devuelven el registro tal como lo entrega la API; no aplican allowlist ni agregan campos. Los campos con datos financieros (`institution_name`, importes) y las descripciones libres deben reportarse solo en lo necesario: no los incluyas en logs, issues ni capturas. No agregues nombres, teléfonos, emails ni números de cuenta al campo `description`; usá datos sintéticos en ejemplos y pruebas (`Casa Demo`, `Taller Demo`, `2026-03-10`).

## Flujo recomendado para un agente

1. Resolver el `home_id` con `entities_search` (el mismo de `/homes/{id}` en la UI).
2. Si vas a registrar, llamar primero `home_maintenance_list` con `date` (y `include_items=true`) para ver el mantenimiento y las líneas de ese día.
3. Llamar `home_maintenance_labor_register` o `home_maintenance_supplies_register` con `dry_run` (default) y mostrar el plan y `existing_today`.
4. Pedir confirmación explícita al usuario.
5. Repetir con `dry_run=false` y la compuerta activa. Si la respuesta es `duplicate` o `ambiguous_maintenance`, **no** reintentar a ciegas: mostrar el registro existente o los candidatos y preguntar.
6. Verificar con `home_maintenance_list?date=<fecha>&include_items=true`; revisar `truncated` si la respuesta es grande.
