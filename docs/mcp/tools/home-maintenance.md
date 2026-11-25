---
title: "Tools de mantenimiento del hogar"
sidebar_position: 10
description: "Referencia de home_maintenance_list, home_maintenance_get, home_maintenance_create, home_maintenance_labor_register, home_maintenance_supplies_register, home_maintenance_update, home_maintenance_item_update, home_maintenance_item_delete y home_maintenance_item_move"
---

# Tools de mantenimiento del hogar

Las nueve tools —dos de lectura y siete de escritura— operan sobre las **reparaciones y mantenimientos** registrados en una vivienda (`home.maintenance`): `home_maintenance_list` consulta el histórico y `home_maintenance_get` devuelve el detalle de una reparación; `home_maintenance_create` da de alta la reparación sin líneas; las dos de registro agregan líneas de **mano de obra** (`labor`) e **insumos** (`supplies`); `home_maintenance_update` edita la cabecera e `home_maintenance_item_update` edita una línea; `home_maintenance_item_delete` borra una línea y `home_maintenance_item_move` mueve líneas de una reparación a otra del mismo hogar. El `home_id` es el mismo de `/homes/{id}` en la UI.

Las siete tools de escritura respetan `dry_run` (`true` por defecto) y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). Las de registro resuelven el **mantenimiento del día** y deduplican antes de escribir; `home_maintenance_create` solo crea la cabecera; `home_maintenance_update` e `home_maintenance_item_update` hacen edición **parcial** (leen el detalle y envían el cuerpo completo); `home_maintenance_item_delete` borra una línea pero **nunca** la cabecera; y `home_maintenance_item_move` crea el destino y mueve las líneas en una sola transacción. `home_maintenance_list` y `home_maintenance_get` son de **solo lectura**: no escriben, no dependen de la compuerta y no requieren permisos adicionales.

El MCP no accede a SQLite: cada tool llama a la API HTTP existente (`/api/homes/{id}/maintenances`, `/api/homes/{id}/maintenances/{maintenance_id}`, `.../items/{item_id}` y `.../items/move`). Las tools validan la **forma** de la entrada; las **reglas de dominio** las valida el backend. Las escrituras se envían con origen `mcp` (`X-P40LA-Source`).

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `home_maintenance_list` | Lectura | 1 | Histórico de reparaciones de una vivienda, con filtros de fecha, tipo y texto, y totales por moneda |
| `home_maintenance_get` | Lectura | 1 | Detalle de una reparación con sus líneas y totales por moneda, sin los topes del listado |
| `home_maintenance_create` | Escritura | 1 | Crea la reparación vacía: 0 insumos y 0 mano de obra |
| `home_maintenance_update` | Escritura | 1–2 | Edita la cabecera con merge parcial; no toca las líneas |
| `home_maintenance_labor_register` | Escritura | 1–2 | Registra una línea de mano de obra; reutiliza o crea el mantenimiento del día |
| `home_maintenance_supplies_register` | Escritura | 1–2 | Registra una línea de insumos con cantidad y unidad; reutiliza o crea el mantenimiento del día |
| `home_maintenance_item_update` | Escritura | 1–3 | Edita una línea con merge parcial y devuelve los totales recalculados |
| `home_maintenance_item_delete` | Escritura | 1–3 | Borra una línea; no borra la cabecera y avisa si la reparación queda vacía |
| `home_maintenance_item_move` | Escritura | 2–3 | Mueve líneas entre reparaciones del mismo hogar; el destino puede existir o crearse en la misma llamada |

`home_maintenance_item_move` hace `GET` del origen, `GET` del destino si ya existe y `POST .../items/move`; el `POST` crea el destino en la misma transacción cuando se envía `target`. `home_maintenance_create` en `dry_run` no hace llamadas HTTP. `home_maintenance_update`, `home_maintenance_item_update` y `home_maintenance_item_delete` hacen siempre un `GET` previo para leer el estado y comparar; la escritura agrega el `PUT`/`DELETE` y, en las líneas, un `GET` final para recalcular `totals_by_currency`.

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

## `home_maintenance_get`

Devuelve una reparación con **todas** sus líneas de mano de obra e insumos y los totales por moneda. Es el detalle completo de una sola reparación y no aplica los topes de 100 reparaciones y 1.000 líneas de `home_maintenance_list`. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `maintenance_id` | integer | Sí | ID de la reparación; debe ser mayor que `0` |

### Ejemplo

```json
{
  "home_id": 3,
  "maintenance_id": 12
}
```

### Salida

```json
{
  "home_id": 3,
  "item_count": 2,
  "maintenance": {
    "id": 12,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "paint",
    "institution_id": 7,
    "institution_name": "Proveedor Demo",
    "description": "Reparación demo de pintura",
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
        "maintenance_id": 12,
        "item_type": "labor",
        "description": "Mano de obra demo",
        "quantity": 1,
        "unit": "unit",
        "amount": 500.00,
        "currency_id": 1,
        "currency_code": "NIO",
        "institution_id": 7,
        "institution_name": "Proveedor Demo",
        "created_at": "2026-03-12T09:00:00Z",
        "updated_at": "2026-03-12T09:00:00Z"
      },
      {
        "id": 102,
        "maintenance_id": 12,
        "item_type": "supplies",
        "description": "Insumos demo",
        "quantity": 2,
        "unit": "unit",
        "amount": 600.00,
        "currency_id": 1,
        "currency_code": "NIO",
        "institution_id": 7,
        "institution_name": "Proveedor Demo",
        "created_at": "2026-03-12T09:10:00Z",
        "updated_at": "2026-03-12T09:10:00Z"
      }
    ],
    "created_at": "2026-03-12T09:00:00Z",
    "updated_at": "2026-03-12T09:10:00Z"
  }
}
```

### Reglas

- **Solo lectura**: no depende de la compuerta ni de `dry_run`; funciona siempre, con o sin `P40LA_MCP_ALLOW_WRITES`.
- **Sin topes de listado**: devuelve la reparación completa. `home_maintenance_list` acota a 100 reparaciones y 1.000 líneas y marca `truncated=true`; este detalle no trunca.
- **`item_count`** es la cantidad de líneas devueltas. Una reparación sin líneas devuelve `item_count: 0`, `items: []` y `totals_by_currency: []`.
- **Sin allowlist**: el objeto `maintenance` se devuelve tal como lo entrega la API; no se agregan ni se ocultan campos.
- **Pertenencia**: una reparación de otra vivienda devuelve `maintenance_not_found`.
- `totals_by_currency` se deriva de las líneas y nunca suma monedas distintas: la moneda vive en cada línea.
- Es la vía recomendada para obtener los `item_ids` antes de `home_maintenance_item_update`, `home_maintenance_item_delete` o `home_maintenance_item_move`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` o `maintenance_id` no positivos; no llama a la API |
| `home_not_found` | La vivienda no existe |
| `maintenance_not_found` | La reparación no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_maintenance_update`

Edita la **cabecera** de una reparación con merge parcial: solo cambian los campos enviados y lo omitido conserva su valor. No toca las líneas. El backend usa semántica de **reemplazo**, así que la tool lee el detalle actual y envía un `PUT` con los cinco campos completos.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `maintenance_id` | integer | Sí | ID de la reparación; debe ser mayor que `0` |
| `performed_at` | string `YYYY-MM-DD` | No | Fecha de la reparación; no puede ser futura. Omitir conserva |
| `kind` | string | No | `corrective` (correctivo) o `preventive` (preventivo). Omitir conserva |
| `area` | string | No | Sector de la vivienda. Omitir conserva |
| `institution_id` | integer | No | Proveedor o contratista; `0` limpia la institución. Omitir conserva |
| `description` | string | No | Descripción libre hasta 500 caracteres; vacío limpia. Omitir conserva |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

Debe venir **al menos un campo editable**; si no, `invalid_request`. `dry_run` no cuenta como campo editable.

### Ejemplo

```json
{
  "home_id": 3,
  "maintenance_id": 12,
  "area": "paint",
  "description": "Reparación demo de pintura",
  "dry_run": true
}
```

### Salida con `dry_run`

`dry_run=true` (default) hace solo el `GET` del detalle y devuelve los campos que cambiarían, el estado actual y el propuesto.

```json
{
  "status": "planned",
  "dry_run": true,
  "maintenance_id": 12,
  "changed_fields": ["area", "description"],
  "current": {
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "other",
    "institution_id": 7,
    "description": "Reparación demo"
  },
  "proposed": {
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "paint",
    "institution_id": 7,
    "description": "Reparación demo de pintura"
  }
}
```

### Salida con escritura

```json
{
  "status": "updated",
  "dry_run": false,
  "maintenance_id": 12,
  "changed_fields": ["area", "description"],
  "maintenance": {
    "id": 12,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "paint",
    "institution_id": 7,
    "institution_name": "Proveedor Demo",
    "description": "Reparación demo de pintura",
    "totals_by_currency": [
      {
        "currency_id": 1,
        "currency_code": "NIO",
        "labor_cost": 500.00,
        "supplies_cost": 0,
        "total": 500.00
      }
    ],
    "items": [
      {
        "id": 101,
        "maintenance_id": 12,
        "item_type": "labor",
        "description": "Mano de obra demo",
        "quantity": 1,
        "unit": "unit",
        "amount": 500.00,
        "currency_id": 1,
        "currency_code": "NIO",
        "institution_id": 7,
        "institution_name": "Proveedor Demo",
        "created_at": "2026-03-12T09:00:00Z",
        "updated_at": "2026-03-12T09:00:00Z"
      }
    ],
    "created_at": "2026-03-12T09:00:00Z",
    "updated_at": "2026-03-12T10:30:00Z"
  }
}
```

### Salida sin cambios

```json
{
  "status": "unchanged",
  "dry_run": false,
  "maintenance_id": 12,
  "changed_fields": [],
  "maintenance": {
    "id": 12,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "paint",
    "institution_id": 7,
    "institution_name": "Proveedor Demo",
    "description": "Reparación demo de pintura",
    "totals_by_currency": [
      {
        "currency_id": 1,
        "currency_code": "NIO",
        "labor_cost": 500.00,
        "supplies_cost": 0,
        "total": 500.00
      }
    ],
    "items": [
      {
        "id": 101,
        "maintenance_id": 12,
        "item_type": "labor",
        "description": "Mano de obra demo",
        "quantity": 1,
        "unit": "unit",
        "amount": 500.00,
        "currency_id": 1,
        "currency_code": "NIO",
        "institution_id": 7,
        "institution_name": "Proveedor Demo",
        "created_at": "2026-03-12T09:00:00Z",
        "updated_at": "2026-03-12T09:00:00Z"
      }
    ],
    "created_at": "2026-03-12T09:00:00Z",
    "updated_at": "2026-03-12T10:30:00Z"
  }
}
```

`status=unchanged` **no es un error**: la tool responde con `isError=false`, el detalle actual y `changed_fields: []`, y **no ejecuta el `PUT`**, ni siquiera con `dry_run=false`.

### Reglas

- **Merge parcial**: se compara el valor efectivo campo por campo. `changed_fields` lista solo los que difieren, en el orden `performed_at`, `kind`, `area`, `institution_id`, `description`.
- **Cuerpo completo**: el `PUT` siempre lleva los cinco campos, porque el backend reemplaza la cabecera. Los omitidos salen con el valor que ya tenían.
- **`institution_id`**: `0` limpia la institución; un valor negativo devuelve `invalid_request`; si no se envía, conserva la actual. Un proveedor inexistente devuelve `institution_not_found` al escribir.
- **`description`**: se recorta con `TrimSpace`; vacía limpia la descripción; más de 500 caracteres devuelve `invalid_request`.
- **`performed_at`**: formato `YYYY-MM-DD` y no futura según `P40LA_TIMEZONE`. Una fecha mal formada o futura devuelve `invalid_date` sin llamar al `PUT`.
- **`kind` y `area`**: mismos valores del dominio (`corrective`/`preventive`; `plumbing`, `electrical`, `roof`, `paint`, `carpentry`, `appliances`, `garden`, `cleaning`, `other`). Un valor inválido devuelve `invalid_request`.
- **No toca las líneas**: montos, monedas y proveedores de las líneas quedan intactos y `totals_by_currency` no cambia. Para editar una línea usá `home_maintenance_item_update`.
- **`dry_run`**: `true` por defecto. No escribe: hace solo el `GET` y devuelve `planned` con `current` y `proposed`. La compuerta se evalúa antes de leer cuando `dry_run=false`.
- **Idempotente**: repetir la misma edición devuelve `status=unchanged` sin `PUT`; el `updated_at` solo cambia cuando hay una escritura real.
- **Llamadas HTTP**: `GET` siempre; `PUT` solo si hay cambios y `dry_run=false`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` o `maintenance_id` no positivos; ningún campo editable; `kind` fuera de `corrective`/`preventive`; `area` inválida; `institution_id` negativo; `description` de más de 500 caracteres |
| `invalid_date` | `performed_at` sin formato `YYYY-MM-DD` o futura |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `home_not_found` | La vivienda no existe |
| `maintenance_not_found` | La reparación no existe o no pertenece al `home_id` |
| `institution_not_found` | La institución no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

El backend también devuelve códigos granulares (`invalid_kind`, `invalid_area`, `future_date`, `invalid_date`); la tool los valida en el preflight y, si llegan por un cambio de estado entre la lectura y la escritura, se devuelven tal cual.

## `home_maintenance_item_update`

Edita una **línea** de mano de obra o insumos con merge parcial sobre el detalle de la reparación. El `PUT` envía los siete campos completos de la línea y, al escribir, devuelve la línea actualizada y los `totals_by_currency` recalculados.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `maintenance_id` | integer | Sí | ID de la reparación que contiene la línea; debe ser mayor que `0` |
| `item_id` | integer | Sí | ID de la línea a editar; debe ser mayor que `0` |
| `item_type` | string | No | `labor` o `supplies`. Omitir conserva |
| `description` | string | No | Descripción de la línea hasta 200 caracteres; vacía es inválida. Omitir conserva |
| `quantity` | number | No | Cantidad de insumos, mayor que `0`; en `labor` se fuerza `1`. Omitir conserva |
| `unit` | string | No | Unidad de insumos; en `labor` se fuerza `unit`. Omitir conserva |
| `amount` | number | No | Monto mayor o igual a `0`. Omitir conserva |
| `currency_id` | integer | No | Moneda de la línea; debe existir. Omitir conserva |
| `institution_id` | integer | No | Proveedor o contratista; `0` limpia. Omitir conserva |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

Debe venir **al menos un campo editable**; si no, `invalid_request`. `dry_run` no cuenta como campo editable.

### Ejemplo

```json
{
  "home_id": 3,
  "maintenance_id": 12,
  "item_id": 101,
  "amount": 650.00,
  "currency_id": 2,
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "maintenance_id": 12,
  "item_id": 101,
  "changed_fields": ["amount", "currency_id"],
  "current": {
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 500.00,
    "currency_id": 1,
    "institution_id": 7
  },
  "proposed": {
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 650.00,
    "currency_id": 2,
    "institution_id": 7
  }
}
```

### Salida con escritura

```json
{
  "status": "updated",
  "dry_run": false,
  "maintenance_id": 12,
  "item_id": 101,
  "changed_fields": ["amount", "currency_id"],
  "item": {
    "id": 101,
    "maintenance_id": 12,
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 650.00,
    "currency_id": 2,
    "currency_code": "USD",
    "institution_id": 7,
    "institution_name": "Proveedor Demo",
    "created_at": "2026-03-12T09:00:00Z",
    "updated_at": "2026-03-12T10:30:00Z"
  },
  "totals_by_currency": [
    {
      "currency_id": 2,
      "currency_code": "USD",
      "labor_cost": 650.00,
      "supplies_cost": 0,
      "total": 650.00
    }
  ]
}
```

### Salida sin cambios

```json
{
  "status": "unchanged",
  "dry_run": false,
  "maintenance_id": 12,
  "item_id": 101,
  "changed_fields": [],
  "item": {
    "id": 101,
    "maintenance_id": 12,
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 650.00,
    "currency_id": 2,
    "currency_code": "USD",
    "institution_id": 7,
    "institution_name": "Proveedor Demo",
    "created_at": "2026-03-12T09:00:00Z",
    "updated_at": "2026-03-12T10:30:00Z"
  }
}
```

Con `changed_fields: []` la tool **no ejecuta el `PUT`**, ni siquiera con `dry_run=false`.

### Reglas

- **Merge parcial**: la tool lee el detalle, busca la línea por `item_id` y compara campo por campo en el orden `item_type`, `description`, `quantity`, `unit`, `amount`, `currency_id`, `institution_id`.
- **Pertenencia**: si la línea no existe en esa reparación devuelve `item_not_found` con un hint para obtener los IDs con `home_maintenance_get` o `home_maintenance_list` con `include_items=true`.
- **Mano de obra**: si el `item_type` efectivo es `labor`, la tool fuerza `quantity=1` y `unit=unit` aunque se envíen otros valores. `supplies → labor` descarta cantidad y unidad; `labor → supplies` conserva `quantity=1` y `unit=unit` del merge y permite ajustarlos.
- **Insumos**: `quantity` debe ser mayor que `0` y `unit` debe ser `unit`, `m`, `m2`, `kg`, `lb`, `l`, `gal`, `sack` o `box`.
- **`description`**: obligatoria en la línea; vacía devuelve `invalid_request` y más de 200 caracteres también. A diferencia de la cabecera, **no** se puede limpiar.
- **`amount`**: finito y mayor o igual a `0`; se redondea a dos decimales. La precisión final la define la moneda en el backend.
- **`currency_id`**: debe ser mayor que `0`; el backend valida que exista y devuelve `currency_not_found`.
- **`institution_id`**: `0` limpia el proveedor; un valor negativo devuelve `invalid_request`.
- **Totales recalculados**: al escribir, la tool hace un `GET` final y devuelve `totals_by_currency` de la reparación. Si ese `GET` falla, la línea ya quedó actualizada: la respuesta trae `warnings` y **no** incluye `totals_by_currency`.
- **`dry_run`**: `true` por defecto. Hace solo el `GET` y devuelve `planned` con `current` y `proposed`; la compuerta se evalúa antes de leer cuando `dry_run=false`.
- **Idempotente**: repetir la misma edición devuelve `status=unchanged` sin `PUT`.
- **Llamadas HTTP**: `GET` inicial siempre; `PUT` si hay cambios y `dry_run=false`; `GET` final de totales solo después de escribir.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id`, `maintenance_id` o `item_id` no positivos; ningún campo editable; `item_type` fuera de `labor`/`supplies`; `description` vacía o de más de 200 caracteres; `amount` no finito o negativo; `currency_id` no positivo; `quantity` no positiva; `unit` fuera de la lista; `institution_id` negativo |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `home_not_found` | La vivienda no existe |
| `maintenance_not_found` | La reparación no existe o no pertenece al `home_id` |
| `item_not_found` | La línea no existe o no pertenece a la reparación indicada; incluye el hint con `home_maintenance_get` |
| `currency_not_found` | La moneda no existe |
| `institution_not_found` | La institución no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

El backend también devuelve códigos granulares (`invalid_item_type`, `item_description_required`, `item_description_too_long`, `invalid_amount`, `currency_required`, `invalid_quantity`, `invalid_unit`); la tool los valida en el preflight y, si llegan por un cambio de estado, se devuelven tal cual.

## `home_maintenance_item_delete`

Borra **una línea** de mano de obra o insumos de una reparación. No borra la cabecera: si era la última línea, la reparación queda vacía y la respuesta lo informa con `maintenance_empty=true` y un warning.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `maintenance_id` | integer | Sí | ID de la reparación que contiene la línea; debe ser mayor que `0` |
| `item_id` | integer | Sí | ID de la línea a borrar; debe ser mayor que `0` |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "home_id": 3,
  "maintenance_id": 12,
  "item_id": 101,
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "maintenance_id": 12,
  "item_id": 101,
  "would_delete": {
    "id": 101,
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 650.00,
    "currency_id": 2,
    "currency_code": "USD"
  },
  "remaining_item_count": 0,
  "maintenance_empty": true,
  "totals_by_currency": [
    {
      "currency_id": 2,
      "currency_code": "USD",
      "labor_cost": 650.00,
      "supplies_cost": 0,
      "total": 650.00
    }
  ],
  "warnings": [
    "La reparación quedará sin líneas. home_maintenance_item_delete no borra la cabecera."
  ]
}
```

`totals_by_currency` en `dry_run` son los **actuales** de la reparación, no los proyectados: la tool no recalcula antes de borrar. `remaining_item_count` es el conteo que quedaría (`items` actuales − 1) y `maintenance_empty` avisa si el borrado deja la reparación sin líneas. Si quedan líneas, `warnings` viene vacío y `maintenance_empty=false`.

### Salida con escritura

```json
{
  "status": "deleted",
  "dry_run": false,
  "maintenance_id": 12,
  "item_id": 101,
  "remaining_item_count": 0,
  "maintenance_empty": true,
  "totals_by_currency": [],
  "warnings": [
    "La reparación quedó sin líneas. home_maintenance_item_delete no borra la cabecera; usá home_maintenance_create o mové líneas si querés conservarla."
  ]
}
```

### Reglas

- **No borra la cabecera**: la reparación sigue existiendo aunque quede con 0 líneas. Con `maintenance_empty=true` la respuesta agrega un warning; en `dry_run` avisa que «quedará sin líneas» y al escribir, que «quedó sin líneas».
- **Reparación vacía visible**: `home_maintenance_list` la devuelve con `items: []` y `totals_by_currency: []`. Se puede completar con las tools de registro o usar como destino de un movimiento.
- **Pertenencia**: si la línea no existe en esa reparación devuelve `item_not_found` con un hint para obtener los IDs con `home_maintenance_get` o `home_maintenance_list` con `include_items=true`. Nunca borra líneas de otra reparación ni de otra vivienda.
- **Totales recalculados**: al escribir, la tool hace un `GET` final y devuelve `totals_by_currency` y los conteos reales. Si ese `GET` falla, la línea ya quedó borrada: la respuesta trae `warnings`, `totals_by_currency: []` y los conteos estimados de la lectura previa.
- **`dry_run`**: `true` por defecto. Hace solo el `GET` y devuelve `would_delete` con la línea y el estado proyectado; la compuerta se evalúa antes de leer cuando `dry_run=false`.
- **Repetición**: la tool declara `idempotentHint=true` porque el estado final no cambia, pero un segundo intento sobre la misma línea devuelve `item_not_found` y no modifica nada.
- **Llamadas HTTP**: `GET` inicial siempre; `DELETE` si `dry_run=false`; `GET` final de totales solo después de borrar.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id`, `maintenance_id` o `item_id` no positivos; no llama a la API |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni llama a la API |
| `home_not_found` | La vivienda no existe |
| `maintenance_not_found` | La reparación no existe o no pertenece al `home_id` |
| `item_not_found` | La línea no existe o no pertenece a la reparación indicada; incluye el hint con `home_maintenance_get` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## Compuerta de escritura

Las siete tools de escritura del dominio (`home_maintenance_create`, `home_maintenance_update`, `home_maintenance_labor_register`, `home_maintenance_supplies_register`, `home_maintenance_item_update`, `home_maintenance_item_delete` y `home_maintenance_item_move`) comparten la misma compuerta. `dry_run` viene activado por defecto y no requiere permisos de escritura; `home_maintenance_list` y `home_maintenance_get` son de lectura y funcionan siempre.

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
| `invalid_request` | Campos faltantes o inconsistentes: `home_id`, `maintenance_id` o `item_id` no positivos, ningún campo editable en una edición, `description` de línea vacía o de más de 200 caracteres, `description` de reparación de más de 500, `amount` negativo, `quantity` no positiva, `unit` fuera de la lista, `area` o `kind` inválidos, `institution_id` negativo, `currency_id` no positivo, `item_ids` vacío, con más de 100, repetido o con IDs no positivos, `item_type` fuera de `labor`/`supplies`, línea de otro tipo cuando se envía `item_type`, `performed_at` ausente en el alta, `target.performed_at` ausente, `performed_at` que no coincide con la fecha del `maintenance_id` o `date` junto con `year` en el listado |
| `invalid_date` | `performed_at` (o `target.performed_at`) no tiene formato `YYYY-MM-DD`, o la fecha es futura |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `home_not_found` | El `home_id` no existe |
| `maintenance_not_found` | El `maintenance_id` no existe o no pertenece al `home_id`; incluye el detalle, la edición y el borrado de líneas, y el origen o el destino de un movimiento |
| `item_not_found` | Una línea no existe o no pertenece a la reparación indicada: en `item_update`, `item_delete` y en el origen de un movimiento; incluye un hint con `home_maintenance_get` |
| `currency_not_found` | La moneda no existe (registro de líneas y edición de una línea) |
| `institution_not_found` | La institución no existe (alta, edición de la cabecera, registro, edición de una línea o destino del movimiento) |
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

`status=unchanged` de `home_maintenance_update` y `home_maintenance_item_update` tampoco es un error: la tool responde con `isError=false`, `changed_fields: []` y el registro actual, y no ejecuta el `PUT`.

## Privacidad y datos ocultos

Las tools devuelven el registro tal como lo entrega la API; no aplican allowlist ni agregan campos. Los campos con datos financieros (`institution_name`, importes) y las descripciones libres deben reportarse solo en lo necesario: no los incluyas en logs, issues ni capturas. No agregues nombres, teléfonos, emails ni números de cuenta al campo `description`; usá datos sintéticos en ejemplos y pruebas (`Casa Demo`, `Taller Demo`, `2026-03-10`).

## Flujo recomendado para un agente

1. Resolver el `home_id` con `entities_search` (el mismo de `/homes/{id}` en la UI).
2. Consultar: `home_maintenance_list` con `date`, `query` o `include_items=true` para encontrar la reparación y sus `item_ids`; `home_maintenance_get` para el detalle completo de una reparación concreta.
3. Para corregir la cabecera (fecha, tipo, área, institución o descripción), llamar `home_maintenance_update` con `dry_run` y mostrar `changed_fields` con `current` y `proposed`.
4. Para corregir una línea, llamar `home_maintenance_item_update` con `dry_run` y mostrar `changed_fields`; para quitarla, `home_maintenance_item_delete` con `dry_run` y mostrar `would_delete`, `remaining_item_count` y si la reparación queda vacía.
5. Si falta el destino de un movimiento, llamar `home_maintenance_create` con `dry_run` (default) para dejarlo vacío, o enviar `target` directamente en el movimiento.
6. Para consolidar líneas, llamar `home_maintenance_item_move` con `dry_run` y mostrar qué líneas se mueven, cuáles quedan y si el origen se borraría.
7. Para un hecho nuevo, llamar `home_maintenance_labor_register` o `home_maintenance_supplies_register` con `dry_run` y mostrar el plan y `existing_today`.
8. Pedir confirmación explícita al usuario.
9. Repetir con `dry_run=false` y la compuerta activa. Si la respuesta es `duplicate`, `ambiguous_maintenance` o `unchanged`, **no** reintentar a ciegas: mostrar el registro existente, los candidatos o el estado actual y preguntar.
10. Verificar con `home_maintenance_get` o con `home_maintenance_list?date=<fecha>&include_items=true`; revisar `truncated` si la respuesta es grande.
