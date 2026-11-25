---
title: "Tools de mantenimiento del hogar"
sidebar_position: 10
description: "Referencia de home_maintenance_list, home_maintenance_create, home_maintenance_labor_register, home_maintenance_supplies_register y home_maintenance_item_move"
---

# Tools de mantenimiento del hogar

Las cinco tools —una de lectura y cuatro de escritura— operan sobre las **reparaciones y mantenimientos** registrados en una vivienda (`home.maintenance`): `home_maintenance_list` consulta el histórico, `home_maintenance_create` da de alta la reparación sin líneas, las dos de registro agregan líneas de **mano de obra** (`labor`) e **insumos** (`supplies`) y `home_maintenance_item_move` mueve líneas de una reparación a otra del mismo hogar. El `home_id` es el mismo de `/homes/{id}` en la UI.

Las cuatro tools de escritura respetan `dry_run` (`true` por defecto) y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). Las de registro resuelven el **mantenimiento del día** y deduplican antes de escribir; `home_maintenance_create` solo crea la cabecera, y `home_maintenance_item_move` crea el destino y mueve las líneas en una sola transacción. `home_maintenance_list` sigue siendo de **solo lectura**: no escribe, no depende de la compuerta y no requiere permisos adicionales.

El MCP no accede a SQLite: cada tool llama a la API HTTP existente (`/api/homes/{id}/maintenances` y `/api/homes/{id}/maintenances/{maintenance_id}/items/move`). Las tools validan la **forma** de la entrada; las **reglas de dominio** las valida el backend. Las escrituras se envían con origen `mcp` (`X-P40LA-Source`).

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `home_maintenance_list` | Lectura | 1 | Histórico de reparaciones de una vivienda, con filtros de fecha, tipo y texto, y totales por moneda |
| `home_maintenance_create` | Escritura | 1 | Crea la reparación vacía: 0 insumos y 0 mano de obra |
| `home_maintenance_labor_register` | Escritura | 1–2 | Registra una línea de mano de obra; reutiliza o crea el mantenimiento del día |
| `home_maintenance_supplies_register` | Escritura | 1–2 | Registra una línea de insumos con cantidad y unidad; reutiliza o crea el mantenimiento del día |
| `home_maintenance_item_move` | Escritura | 2–3 | Mueve líneas entre reparaciones del mismo hogar; el destino puede existir o crearse en la misma llamada |

`home_maintenance_item_move` hace `GET` del origen, `GET` del destino si ya existe y `POST .../items/move`; el `POST` crea el destino en la misma transacción cuando se envía `target`. `home_maintenance_create` en `dry_run` no hace llamadas HTTP.

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
- Las reparaciones sin líneas (creadas con `home_maintenance_create`) se devuelven con `items: []` y `totals_by_currency: []`, y `count` las incluye; el filtro `item_type` no las devuelve.
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

## `home_maintenance_create`

Crea una **reparación vacía**: solo la cabecera, con `0` insumos y `0` mano de obra. Sirve para preparar el destino de un movimiento o para registrar el hecho antes de cargar las líneas con `home_maintenance_labor_register` o `home_maintenance_supplies_register`.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda destino; debe ser mayor que `0` |
| `performed_at` | string `YYYY-MM-DD` | Sí | Fecha de la reparación; no puede ser futura |
| `kind` | string | No | `corrective` (correctivo) o `preventive` (preventivo); default `corrective` |
| `area` | string | No | Sector de la vivienda (ver [Reglas](#reglas-1)); default `other` |
| `institution_id` | integer | No | Proveedor o contratista; debe existir |
| `description` | string | No | Descripción libre; máximo 500 caracteres |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "home_id": 3,
  "performed_at": "2026-03-12",
  "kind": "preventive",
  "area": "roof",
  "description": "Revisión demo de techo",
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "action": "create_maintenance",
    "performed_at": "2026-03-12",
    "kind": "preventive",
    "area": "roof",
    "description": "Revisión demo de techo",
    "item_count": 0
  }
}
```

`dry_run=true` no llama a la API. Si se envía `institution_id`, el plan lo incluye.

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "item_count": 0,
  "maintenance": {
    "id": 12,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "preventive",
    "area": "roof",
    "institution_id": null,
    "institution_name": "",
    "description": "Revisión demo de techo",
    "totals_by_currency": [],
    "items": [],
    "created_at": "2026-03-12T09:00:00Z",
    "updated_at": "2026-03-12T09:00:00Z"
  }
}
```

### Reglas

- **Solo cabecera**: la tool nunca envía `item`; la reparación queda con `items: []` y `totals_by_currency: []`. No requiere `currency_id`: la moneda vive en cada línea.
- **`description` opcional** de hasta 500 caracteres. Es independiente del máximo de 200 de las líneas.
- **`kind` y `area`** usan los mismos valores que el resto del dominio; sin valor se aplican `corrective` y `other`.
- **`institution_id`** es opcional; si viene, debe existir. Un proveedor inexistente devuelve `institution_not_found`.
- **`performed_at`** es obligatoria, con formato `YYYY-MM-DD` y no futura según `P40LA_TIMEZONE`. Una fecha mal formada o futura devuelve `invalid_date` sin llamar a la API.
- **`dry_run`** viene activado por defecto y no escribe. El preflight valida la forma y la fecha; `home_not_found` e `institution_not_found` los valida el backend al escribir.
- **Sin deduplicación**: repetir el alta crea otra reparación, aunque coincidan fecha, área y descripción. Usá `home_maintenance_list` con `date` para revisar lo existente antes de crear.
- **Reparación vacía visible**: `home_maintenance_list` no descarta las reparaciones sin líneas; aparecen con `items: []`, `totals_by_currency: []` y `count` las incluye. En cambio, el filtro `item_type` **no** las devuelve, porque exige al menos una línea de ese tipo.
- La reparación vacía es un destino válido de `home_maintenance_item_move`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` no positivo, `performed_at` ausente, `kind` inválido, `area` inválida o `description` de más de 500 caracteres |
| `invalid_date` | `performed_at` sin formato `YYYY-MM-DD` o futura |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `home_not_found` | El `home_id` no existe |
| `institution_not_found` | La institución no existe |
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
| `area` | string | No | Sector de la vivienda (ver [Reglas](#reglas-2)); default `other` |
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

## `home_maintenance_item_move`

Mueve entre **1 y 100 líneas** de mano de obra y/o insumos de una reparación **origen** a una reparación **destino** del mismo hogar. El destino puede existir (`target_maintenance_id`) o crearse en la misma llamada (`target`); en ese caso el backend crea la reparación, mueve las líneas y, si se pide, elimina el origen vacío en **una sola transacción**.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda de ambas reparaciones; debe ser mayor que `0` |
| `source_maintenance_id` | integer | Sí | Reparación que contiene las líneas a mover |
| `item_ids` | integer[] | Sí | Entre 1 y 100 líneas, sin repetir y con IDs positivos |
| `item_type` | string | No | Salvaguarda opcional: `labor` o `supplies`; rechaza líneas de otro tipo |
| `target_maintenance_id` | integer | Condicional | Reparación destino existente; excluyente con `target` |
| `target` | object | Condicional | Datos para crear el destino; excluyente con `target_maintenance_id` |
| `delete_source_if_empty` | boolean | No | `false` por defecto: borra el origen solo si queda con 0 líneas |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

Exactamente uno de `target_maintenance_id` o `target` es obligatorio. `target` acepta:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `performed_at` | string `YYYY-MM-DD` | Sí | Fecha del destino nuevo; no puede ser futura |
| `kind` | string | No | `corrective` o `preventive`; default `corrective` |
| `area` | string | No | Sector de la vivienda; default `other` |
| `institution_id` | integer | No | Proveedor o contratista del destino; debe existir |
| `description` | string | No | Descripción del destino; máximo 500 caracteres |

### Ejemplo

```json
{
  "home_id": 3,
  "source_maintenance_id": 10,
  "item_ids": [101],
  "item_type": "labor",
  "target": {
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "plumbing",
    "description": "Reparación demo destino"
  },
  "delete_source_if_empty": true,
  "dry_run": true
}
```

### Salida con `dry_run`

`dry_run=true` (default) hace solo lecturas: trae el origen y, si el destino existe, también el destino. Devuelve las líneas que se moverían, las que quedarían y el estado proyectado del origen.

```json
{
  "status": "planned",
  "dry_run": true,
  "source": {
    "maintenance_id": 10,
    "performed_at": "2026-03-10",
    "kind": "corrective",
    "area": "plumbing",
    "description": "Reparación demo de fuga",
    "item_count": 3
  },
  "target": {
    "action": "create_maintenance",
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "plumbing",
    "description": "Reparación demo destino",
    "item_count": 0
  },
  "moved_items": [
    {
      "id": 101,
      "maintenance_id": 10,
      "item_type": "labor",
      "description": "Mano de obra demo",
      "quantity": 1,
      "unit": "unit",
      "amount": 500.00,
      "currency_id": 1,
      "currency_code": "NIO",
      "institution_id": 7,
      "institution_name": "Taller Demo",
      "created_at": "2026-03-10T14:00:00Z",
      "updated_at": "2026-03-10T14:00:00Z"
    }
  ],
  "remaining_items": [
    {
      "id": 102,
      "maintenance_id": 10,
      "item_type": "supplies",
      "description": "Insumos demo",
      "quantity": 2,
      "unit": "unit",
      "amount": 600.00,
      "currency_id": 1,
      "currency_code": "NIO",
      "institution_id": 7,
      "institution_name": "Taller Demo",
      "created_at": "2026-03-10T14:00:00Z",
      "updated_at": "2026-03-10T14:00:00Z"
    },
    {
      "id": 103,
      "maintenance_id": 10,
      "item_type": "labor",
      "description": "Ajuste demo",
      "quantity": 1,
      "unit": "unit",
      "amount": 250.00,
      "currency_id": 1,
      "currency_code": "NIO",
      "institution_id": null,
      "institution_name": "",
      "created_at": "2026-03-10T15:00:00Z",
      "updated_at": "2026-03-10T15:00:00Z"
    }
  ],
  "source_item_count_after": 2,
  "would_delete_source": false,
  "warnings": [
    "El origen conserva 2 líneas; no se borrará."
  ]
}
```

Con `target_maintenance_id`, `target.action` es `use_existing` y el objeto trae el resumen del destino existente (`maintenance_id`, `performed_at`, `kind`, `area`, `description`, `item_count`). `would_delete_source` es `true` solo si `delete_source_if_empty=true` y no queda ninguna línea.

### Salida con escritura

```json
{
  "status": "moved",
  "dry_run": false,
  "source_maintenance_id": 10,
  "target_maintenance_id": 11,
  "target_created": true,
  "moved_count": 1,
  "moved_items": [
    {
      "id": 101,
      "maintenance_id": 11,
      "item_type": "labor",
      "description": "Mano de obra demo",
      "quantity": 1,
      "unit": "unit",
      "amount": 500.00,
      "currency_id": 1,
      "currency_code": "NIO",
      "institution_id": 7,
      "institution_name": "Taller Demo",
      "created_at": "2026-03-10T14:00:00Z",
      "updated_at": "2026-03-12T09:00:00Z"
    }
  ],
  "source_item_count": 2,
  "source_deleted": false,
  "target": {
    "id": 11,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "plumbing",
    "institution_id": null,
    "institution_name": "",
    "description": "Reparación demo destino",
    "totals_by_currency": [],
    "items": [],
    "created_at": "2026-03-12T09:00:00Z",
    "updated_at": "2026-03-12T09:00:00Z"
  }
}
```

`target_created=true` indica que el destino se creó en esta llamada; con `target_maintenance_id` es `false`. `source_item_count` es lo que queda en el origen después del movimiento y `source_deleted=true` confirma que el origen se eliminó. `target` trae la cabecera del destino, sin líneas ni totales recalculados: las líneas movidas están en `moved_items`. La respuesta agrega `warnings` cuando `delete_source_if_empty=true` y el origen conserva líneas.

### Reglas

- **Un solo destino**: exactamente uno de `target_maintenance_id` o `target`. Ninguno devuelve `target_required`; ambos, `target_conflict`.
- **Destino distinto**: `target_maintenance_id` no puede ser igual a `source_maintenance_id`; devuelve `same_maintenance`.
- **Mismo hogar**: origen y destino se resuelven con el mismo `home_id`. Un ID de otra vivienda devuelve `maintenance_not_found`.
- **Selección**: `item_ids` acepta entre 1 y 100 IDs positivos, sin repetir. Un `item_id` que no existe o no pertenece al origen devuelve `item_not_found` antes de escribir.
- **`item_type` como salvaguarda**: si se envía, todas las líneas deben ser de ese tipo. Una línea de otro tipo devuelve `invalid_request` en el preflight; el backend responde `item_type_mismatch` si el estado cambió.
- **Validación del destino nuevo**: `target.performed_at` es obligatoria, con formato `YYYY-MM-DD` y no futura; `target.kind` y `target.area` usan los mismos valores del dominio; `target.description` admite hasta 500 caracteres.
- **Transacción única**: con `target`, el backend crea la reparación, mueve las líneas y, si corresponde, borra el origen en una sola transacción. Si el `UPDATE` no afecta exactamente las líneas pedidas, devuelve `move_conflict`, revierte todo y no deja un destino vacío.
- **`delete_source_if_empty`**: `false` por defecto. Con `true`, el origen se borra **solo** si queda con 0 líneas; si conserva alguna, no se borra y la respuesta agrega `warnings` con el conteo final. En `dry_run` el aviso es «no se borrará»; al escribir, «no se borró».
- **`dry_run`**: `true` por defecto. No escribe: hace 1 lectura del origen y, si el destino existe, 1 del destino. El preflight valida forma, fecha, pertenencia y tipo; los prerrequisitos del destino los valida el backend al escribir.
- **No idempotente**: repetir la misma llamada después de un movimiento exitoso devuelve `item_not_found`, porque las líneas ya no pertenecen al origen. Para verificar, usá `home_maintenance_list` con `include_items=true`.
- **Totales derivados**: las líneas conservan su moneda, proveedor y montos. Los `totals_by_currency` no se mueven como dato: se recalculan desde las líneas, así que bajan en el origen y suben en el destino.
- **Destructiva con el flag**: `home_maintenance_item_move` declara `destructiveHint=true` porque `delete_source_if_empty=true` puede borrar la reparación de origen. Sin el flag solo cambia la relación de las líneas.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id`, `source_maintenance_id` o `target_maintenance_id` no positivos; `item_ids` vacío, con más de 100, repetido o con IDs no positivos; `item_type` fuera de `labor`/`supplies`; una línea elegida no es de `item_type`; `target.performed_at` ausente; `target.kind` o `target.area` inválidos; `target.description` de más de 500 caracteres |
| `invalid_date` | `target.performed_at` sin formato `YYYY-MM-DD` o futura |
| `target_required` | No se envió `target_maintenance_id` ni `target` |
| `target_conflict` | Se enviaron `target_maintenance_id` y `target` a la vez |
| `same_maintenance` | El destino es la misma reparación de origen |
| `item_not_found` | Un `item_id` no existe o no pertenece al origen |
| `maintenance_not_found` | El origen o el destino no existe o no pertenece al `home_id` |
| `institution_not_found` | El proveedor del destino nuevo no existe |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `home_not_found` | El `home_id` no existe |
| `move_conflict` | El movimiento no afectó todas las líneas pedidas; el backend revierte todo |
| `item_not_in_source` | El backend detectó que una línea dejó de pertenecer al origen entre la lectura y la escritura |
| `item_type_mismatch` | El backend detectó una línea de otro tipo cuando se envió `item_type` |
| `invalid_item_ids` | El backend rechazó `item_ids` por cantidad, repetición o IDs no positivos |
| `invalid_item_type` / `invalid_kind` / `invalid_area` / `future_date` | Validaciones del backend si el estado o los datos cambian entre el preflight y la escritura |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

El movimiento no crea ni edita montos: las líneas conservan su moneda y su proveedor, por eso `currency_not_found` no aplica a esta tool.

## Compuerta de escritura

Las cuatro tools de escritura del dominio (`home_maintenance_create`, `home_maintenance_labor_register`, `home_maintenance_supplies_register` y `home_maintenance_item_move`) comparten la misma compuerta. `dry_run` viene activado por defecto y no requiere permisos de escritura; `home_maintenance_list` es de lectura y funciona siempre.

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
| `invalid_request` | Campos faltantes o inconsistentes: `home_id` no positivo, `description` de línea vacía o de más de 200 caracteres, `description` de reparación de más de 500, `amount` negativo, `quantity` no positiva, `unit` fuera de la lista, `area` o `kind` inválidos, `item_ids` vacío, con más de 100, repetido o con IDs no positivos, `item_type` fuera de `labor`/`supplies`, línea de otro tipo cuando se envía `item_type`, `performed_at` ausente en el alta, `target.performed_at` ausente, `performed_at` que no coincide con la fecha del `maintenance_id` o `date` junto con `year` en el listado |
| `invalid_date` | `performed_at` (o `target.performed_at`) no tiene formato `YYYY-MM-DD`, o la fecha es futura |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `home_not_found` | El `home_id` no existe |
| `maintenance_not_found` | El `maintenance_id` no existe o no pertenece al `home_id`; incluye el origen o el destino de un movimiento |
| `item_not_found` | Una línea de `item_ids` no existe o no pertenece al origen del movimiento |
| `currency_not_found` | La moneda no existe (registro de líneas) |
| `institution_not_found` | La institución no existe (alta, registro o destino del movimiento) |
| `target_required` | El movimiento no recibió `target_maintenance_id` ni `target` |
| `target_conflict` | El movimiento recibió `target_maintenance_id` y `target` a la vez |
| `same_maintenance` | El destino del movimiento es la misma reparación de origen |
| `item_not_in_source` | El backend detectó que una línea dejó de pertenecer al origen entre la lectura y la escritura |
| `item_type_mismatch` | El backend detectó una línea de otro tipo cuando se envió `item_type` |
| `move_conflict` | El movimiento no afectó todas las líneas pedidas; el backend revierte todo y no modifica nada |
| `duplicate` | No es un error de transporte: `status=duplicate` con la línea equivalente del día, sin escribir |
| `ambiguous_maintenance` | No es un error de transporte: `status=ambiguous_maintenance` con los candidatos del día, sin escribir |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

El backend también devuelve códigos de validación granular (`invalid_kind`, `invalid_area`, `future_date`, `invalid_item_ids`, `invalid_item_type`). La tool valida esos casos en el preflight y los traduce a `invalid_request` o `invalid_date`; si llegan del backend por un cambio de estado entre la lectura y la escritura, se devuelven tal cual.

`status=duplicate` y `status=ambiguous_maintenance` llegan con `isError=false` y datos: no son fallas de red ni del servidor, y no hay que reintentar la misma llamada. Para resolverlos, usá `allow_duplicate=true` o elegí un `maintenance_id` de los candidatos.

## Privacidad y datos ocultos

Las tools devuelven el registro tal como lo entrega la API; no aplican allowlist ni agregan campos. Los campos con datos financieros (`institution_name`, importes) y las descripciones libres deben reportarse solo en lo necesario: no los incluyas en logs, issues ni capturas. No agregues nombres, teléfonos, emails ni números de cuenta al campo `description`; usá datos sintéticos en ejemplos y pruebas (`Casa Demo`, `Taller Demo`, `2026-03-10`).

## Flujo recomendado para un agente

1. Resolver el `home_id` con `entities_search` (el mismo de `/homes/{id}` en la UI).
2. Llamar `home_maintenance_list` con `date`, `query` o `include_items=true` para encontrar la reparación y los `item_ids` de sus líneas.
3. Si falta el destino, llamar `home_maintenance_create` con `dry_run` (default) para dejarlo vacío, o enviar `target` directamente en el movimiento.
4. Para consolidar o corregir líneas, llamar `home_maintenance_item_move` con `dry_run` y mostrar qué líneas se mueven, cuáles quedan y si el origen se borraría.
5. Para un hecho nuevo, llamar `home_maintenance_labor_register` o `home_maintenance_supplies_register` con `dry_run` y mostrar el plan y `existing_today`.
6. Pedir confirmación explícita al usuario.
7. Repetir con `dry_run=false` y la compuerta activa. Si la respuesta es `duplicate` o `ambiguous_maintenance`, **no** reintentar a ciegas: mostrar el registro existente o los candidatos y preguntar.
8. Verificar con `home_maintenance_list?date=<fecha>&include_items=true`; revisar `truncated` si la respuesta es grande.
