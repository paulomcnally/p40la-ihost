---
title: "Tools de autos"
sidebar_position: 3
description: "Referencia de autos_list, autos_create, autos_update y el registro de mantenimientos, inspecciones y seguros"
---

# Tools de autos

Las seis tools operan sobre el parque vehicular y sus registros asociados. El `auto_id` es el mismo que aparece en la URL de la UI, por ejemplo `/autos/1` → `auto_id: 1`; el `service_id` de un seguro es el servicio que se asocia al auto.

El MCP no accede a SQLite: cada tool llama a la API HTTP existente (`/api/autos`, `/api/autos/{id}/maintenances`, `/api/autos/{id}/inspections`, `/api/autos/{id}/services`). Las tools validan la **forma** de la entrada; las **reglas de dominio** las valida el backend. Ver [Validaciones de dominio](#validaciones-de-dominio).

## Resumen

| Tool | Tipo | Qué hace | Idempotente |
|---|---|---|---|
| `autos_list` | Lectura | Lista y filtra autos; permite pedir uno por ID | — |
| `autos_create` | Escritura | Crea un auto | No |
| `autos_update` | Escritura | Reemplaza los datos de un auto | Sí |
| `autos_maintenance_register` | Escritura | Registra un mantenimiento y sus líneas de mano de obra e insumos | No |
| `autos_inspection_register` | Escritura | Registra una inspección mecánica y devuelve el cronograma | No |
| `autos_insurance_register` | Escritura | Asocia un servicio de seguro al auto | No (repetir devuelve `already_exists`) |

## `autos_list`

Lista los autos registrados, ordenados por marca, modelo e ID. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `query` | string | No | Filtra por marca, modelo, placa o VIN; ignora mayúsculas y acentos |
| `auto_id` | integer | No | Devuelve solo ese auto; se puede combinar con `query` |

### Ejemplo

```json
{
  "query": "demo"
}
```

### Salida

```json
{
  "query": "demo",
  "count": 1,
  "autos": [
    {
      "id": 1,
      "year": 2024,
      "brand": "Marca Demo",
      "model": "Modelo Demo",
      "color": "white",
      "icon": "vehicle",
      "vin": "VIN0000000000000001",
      "placa": "DEMO-001"
    }
  ]
}
```

### Reglas

- `query` se compara contra marca, modelo, placa y VIN; `sedan` encuentra `Sedán`.
- Si se envían `query` y `auto_id`, se aplican ambos filtros.
- `query` aparece en la salida solo cuando tiene texto.
- `motor`, `chasis`, `vin` y `placa` aparecen solo cuando tienen valor.
- `count` es la cantidad de autos devueltos, no el total del sistema.
- No modifica datos.

## `autos_create`

Crea un auto. No es idempotente: repetir la llamada con los mismos datos crea otro auto.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `year` | integer | Sí | Año del auto, entre 1900 y 2100 |
| `brand` | string | Sí | Marca |
| `model` | string | Sí | Modelo |
| `color` | string | Sí | Código canónico de la paleta |
| `icon` | string | No | Ícono; default `vehicle` |
| `motor` | string | No | Número de motor |
| `chasis` | string | No | Número de chasis |
| `vin` | string | No | VIN |
| `placa` | string | No | Placa |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

La paleta canónica de `color` es: `white`, `black`, `gray`, `silver`, `red`, `blue`, `green`, `yellow`, `orange`, `brown`, `beige`, `gold`, `burgundy`, `light_blue`, `pink`, `purple`.

### Ejemplo

```json
{
  "year": 2024,
  "brand": "Marca Demo",
  "model": "Modelo Demo",
  "color": "white",
  "vin": "VIN0000000000000001",
  "placa": "DEMO-001",
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "id": 0,
    "year": 2024,
    "brand": "Marca Demo",
    "model": "Modelo Demo",
    "color": "white",
    "icon": "vehicle",
    "vin": "VIN0000000000000001",
    "placa": "DEMO-001"
  }
}
```

`id: 0` es un marcador: el auto todavía no existe. La salida refleja los valores normalizados (espacios recortados, `color` en minúsculas e `icon` con su default).

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "auto": {
    "id": 1,
    "year": 2024,
    "brand": "Marca Demo",
    "model": "Modelo Demo",
    "color": "white",
    "icon": "vehicle",
    "vin": "VIN0000000000000001",
    "placa": "DEMO-001"
  }
}
```

### Reglas

- `dry_run=true` (default) no llama a `POST /api/autos`.
- Con `dry_run=false` y la compuerta activa crea el auto y devuelve `status: created`.
- **No es idempotente**: no deduplica por placa ni VIN. Antes de crear, usá `autos_list` para verificar que no exista.
- Un `year` fuera de rango, marca/modelo/color vacíos o un color fuera de la paleta devuelven `invalid_request`.
- El preflight de `dry_run` valida forma y rango de año, pero **no** la paleta de color: eso lo hace el backend al escribir.

## `autos_update`

Actualiza un auto existente. El endpoint es un `PUT`: reemplaza el registro completo.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `auto_id` | integer | Sí | ID del auto a actualizar |
| `year` | integer | Sí | Año del auto, entre 1900 y 2100 |
| `brand` | string | Sí | Marca |
| `model` | string | Sí | Modelo |
| `color` | string | Sí | Código canónico de la paleta |
| `icon` | string | No | Ícono; default `vehicle` |
| `motor` | string | No | Número de motor; vacío lo limpia |
| `chasis` | string | No | Número de chasis; vacío lo limpia |
| `vin` | string | No | VIN; vacío lo limpia |
| `placa` | string | No | Placa; vacía la limpia |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "auto_id": 1,
  "year": 2024,
  "brand": "Marca Demo",
  "model": "Modelo Demo Actualizado",
  "color": "black",
  "icon": "vehicle",
  "vin": "VIN0000000000000001",
  "placa": "DEMO-001",
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "id": 1,
    "year": 2024,
    "brand": "Marca Demo",
    "model": "Modelo Demo Actualizado",
    "color": "black",
    "icon": "vehicle",
    "vin": "VIN0000000000000001",
    "placa": "DEMO-001"
  }
}
```

### Salida con escritura

```json
{
  "status": "updated",
  "dry_run": false,
  "auto": {
    "id": 1,
    "year": 2024,
    "brand": "Marca Demo",
    "model": "Modelo Demo Actualizado",
    "color": "black",
    "icon": "vehicle",
    "vin": "VIN0000000000000001",
    "placa": "DEMO-001"
  }
}
```

### Reglas

- Enviá siempre `year`, `brand`, `model` y `color`: el `PUT` reemplaza el registro.
- Si omitís `icon`, se usa `vehicle`.
- Si omitís `motor`, `chasis`, `vin` o `placa`, se envían vacíos y **se limpian**. Para conservar un valor, reenvialo tal como está.
- En `dry_run`, la tool consulta el auto: un `auto_id` inexistente devuelve `not_found` sin escribir.
- Es idempotente: repetir el mismo `PUT` deja el auto en el mismo estado.

## `autos_maintenance_register`

Registra un mantenimiento de un auto y, opcionalmente, sus líneas de mano de obra e insumos.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `auto_id` | integer | Sí | ID del auto |
| `performed_at` | string `YYYY-MM-DD` | Sí | Fecha del mantenimiento; no puede ser futura |
| `mileage` | integer | Sí | Kilometraje; debe ser mayor o igual a `0` |
| `institution_id` | integer | Sí | ID de un taller mecánico |
| `currency_id` | integer | Sí | ID de una moneda existente |
| `description` | string | No | Descripción del mantenimiento |
| `items` | array | No | Hasta 20 líneas de mano de obra o insumos |
| `items[].item_type` | string | Sí | `labor` (mano de obra) o `supplies` (insumos) |
| `items[].description` | string | Sí | Descripción de la línea |
| `items[].amount` | number | Sí | Monto de la línea; no puede ser negativo |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "auto_id": 1,
  "performed_at": "2026-11-20",
  "mileage": 45000,
  "institution_id": 7,
  "currency_id": 1,
  "description": "Cambio de aceite y filtros",
  "items": [
    {
      "item_type": "labor",
      "description": "Mano de obra demo",
      "amount": 500.00
    },
    {
      "item_type": "supplies",
      "description": "Insumos demo",
      "amount": 1200.00
    }
  ],
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "auto_id": 1,
    "maintenance": {
      "performed_at": "2026-11-20",
      "mileage": 45000,
      "institution_id": 7,
      "description": "Cambio de aceite y filtros",
      "currency_id": 1
    },
    "items": [
      {
        "item_type": "labor",
        "description": "Mano de obra demo",
        "amount": 500.00
      },
      {
        "item_type": "supplies",
        "description": "Insumos demo",
        "amount": 1200.00
      }
    ]
  }
}
```

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "maintenance": {
    "id": 7,
    "auto_id": 1,
    "performed_at": "2026-11-20",
    "mileage": 45000,
    "institution_id": 7,
    "institution_name": "Taller Demo",
    "description": "Cambio de aceite y filtros",
    "currency_id": 1,
    "currency_code": "NIO",
    "labor_cost": 500.00,
    "supplies_cost": 1200.00,
    "total_cost": 1700.00,
    "items": [
      {
        "id": 101,
        "maintenance_id": 7,
        "item_type": "labor",
        "description": "Mano de obra demo",
        "amount": 500.00
      },
      {
        "id": 102,
        "maintenance_id": 7,
        "item_type": "supplies",
        "description": "Insumos demo",
        "amount": 1200.00
      }
    ]
  },
  "items": [
    {
      "id": 101,
      "maintenance_id": 7,
      "item_type": "labor",
      "description": "Mano de obra demo",
      "amount": 500.00
    },
    {
      "id": 102,
      "maintenance_id": 7,
      "item_type": "supplies",
      "description": "Insumos demo",
      "amount": 1200.00
    }
  ]
}
```

Si una línea falla, el mantenimiento y las líneas anteriores quedan creados. La tool corta en la primera línea fallida y responde con un resumen como este:

```json
{
  "status": "created_partial",
  "dry_run": false,
  "partial": true,
  "maintenance": {
    "id": 7,
    "total_cost": 500.00
  },
  "items": [
    {
      "id": 101,
      "maintenance_id": 7,
      "item_type": "labor",
      "description": "Mano de obra demo",
      "amount": 500.00
    }
  ],
  "warnings": [
    "la línea 2 no se pudo crear: error demo de la API"
  ]
}
```

El `maintenance` de la respuesta real es el detalle recalculado por la API, con `labor_cost`, `supplies_cost`, `total_cost` y sus `items`. El `items` de nivel superior son las líneas creadas en esta llamada; `maintenance.items` es el detalle persistido.

### Reglas

- `dry_run=true` (default) verifica que el auto exista y devuelve `planned`; no crea nada.
- La fecha se valida contra `P40LA_TIMEZONE`: no puede ser futura.
- El preflight de `dry_run` **no** valida la categoría del taller ni la moneda; esas reglas las aplica el backend al escribir.
- Si `items` está presente, se crea primero el mantenimiento y después cada línea, en orden y en peticiones secuenciales (máximo 20).
- Cada línea debe tener `item_type` `labor` o `supplies`, descripción no vacía y `amount >= 0`.
- **No es idempotente**: repetir la llamada crea otro mantenimiento. Si hubo `created_partial`, revisá el detalle antes de reintentar.

## `autos_inspection_register`

Registra una inspección mecánica de un auto y devuelve el cronograma actualizado.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `auto_id` | integer | Sí | ID del auto |
| `performed_at` | string `YYYY-MM-DD` | Sí | Fecha de la inspección; no puede ser futura |
| `inspection_year` | integer | Sí | Año del ciclo que cumple la inspección |
| `institution_id` | integer | Sí | ID de un centro de inspección mecánica |
| `result` | string | Sí | `approved` o `rejected` |
| `notes` | string | No | Notas de la inspección |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "auto_id": 1,
  "performed_at": "2026-11-20",
  "inspection_year": 2026,
  "institution_id": 9,
  "result": "approved",
  "notes": "Inspección demo aprobada",
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "performed_at": "2026-11-20",
    "inspection_year": 2026,
    "institution_id": 9,
    "result": "approved",
    "notes": "Inspección demo aprobada"
  }
}
```

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "inspection": {
    "id": 11,
    "auto_id": 1,
    "performed_at": "2026-11-20",
    "inspection_year": 2026,
    "institution_id": 9,
    "institution_name": "Centro de Inspección Demo",
    "result": "approved",
    "notes": "Inspección demo aprobada"
  },
  "schedule": {
    "first_due_year": 2026,
    "next_due_year": 2027,
    "next_due_date": "2027-01-01",
    "status": "due_soon",
    "days_until_due": 38,
    "approved_years": [
      2026
    ],
    "rejected_years": []
  }
}
```

El ejemplo usa un auto año 2024 y la fecha de referencia 2026-11-24. El cronograma real depende de las inspecciones aprobadas y de la fecha actual. `status` puede ser `ok`, `due_soon` (vence en 60 días o menos) u `overdue`; `next_due_date` es el 1 de enero del primer año sin inspección aprobada.

### Reglas

- `dry_run=true` (default) verifica que el auto exista y devuelve `planned`; no crea nada.
- La fecha no puede ser futura según `P40LA_TIMEZONE`.
- El backend exige que `inspection_year` esté entre `auto.year + 2` y `performed_at.year + 1`; un año fuera de ese rango devuelve `invalid_request`.
- El centro debe pertenecer a la categoría de inspección mecánica; si no, devuelve `institution_category`.
- Después de crear, la tool vuelve a leer las inspecciones para devolver `schedule`. Si esa lectura falla, la inspección queda creada y la respuesta incluye `warnings` sin `schedule`.
- **No es idempotente**: repetir la llamada crea otra inspección para el mismo auto y año.

## `autos_insurance_register`

Asocia un servicio de seguro a un auto.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `auto_id` | integer | Sí | ID del auto |
| `service_id` | integer | Sí | ID del servicio de seguro a asociar |
| `coverage_type` | string | Sí | `daños_a_terceros` o `full_cover` |
| `policy_number` | string | Sí | Número de póliza |
| `insurer_number` | string | Sí | Número de aseguradora |
| `certificate` | string | No | Número de certificado |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "auto_id": 1,
  "service_id": 12,
  "coverage_type": "full_cover",
  "policy_number": "POL-DEMO-0001",
  "insurer_number": "ASEG-DEMO-001",
  "certificate": "CERT-DEMO-001",
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "auto_id": 1,
    "insurance": {
      "service_id": 12,
      "coverage_type": "full_cover",
      "policy_number": "POL-DEMO-0001",
      "certificate": "CERT-DEMO-001",
      "insurer_number": "ASEG-DEMO-001"
    },
    "service": "Seguro Demo",
    "coverage": "full_cover"
  }
}
```

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "insurance": {
    "id": 21,
    "auto_id": 1,
    "service_id": 12,
    "coverage_type": "full_cover",
    "policy_number": "POL-DEMO-0001",
    "certificate": "CERT-DEMO-001",
    "insurer_number": "ASEG-DEMO-001"
  }
}
```

### Reglas

- Antes de escribir, la tool consulta `GET /api/autos/{id}/available-services`. Si el `service_id` no está disponible para el auto, devuelve `prerequisite_missing`.
- Si el auto ya tiene asociado ese `service_id`, devuelve `already_exists` y el `hint` incluye el ID y la póliza del seguro existente. No se crea un duplicado.
- `coverage_type` se normaliza a minúsculas; solo se aceptan `daños_a_terceros` y `full_cover`.
- `policy_number` e `insurer_number` son obligatorios; `certificate` es opcional.
- El `dry_run` también ejecuta las verificaciones de disponibilidad y duplicado, por lo que puede devolver `prerequisite_missing` o `already_exists` sin escribir.
- Repetir el alta con el mismo `service_id` no duplica el seguro: devuelve `already_exists`.

## Compuerta de escritura

Las cinco tools de escritura comparten la misma compuerta. `dry_run` viene activado por defecto y no requiere permisos de escritura.

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
| `invalid_request` | Campos faltantes o fuera de rango: `year` fuera de 1900–2100, color fuera de la paleta, `mileage` negativo, más de 20 líneas, `item_type` inválido, cobertura inválida, póliza o aseguradora vacías |
| `invalid_date` | `performed_at` no tiene formato `YYYY-MM-DD` o es futura |
| `not_found` | El `auto_id` no existe o el registro referenciado no se encuentra |
| `prerequisite_missing` | El `service_id` no está disponible para asociar al auto |
| `already_exists` | El auto ya tiene ese `service_id` como seguro |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | La API no responde o hay un error de red |
| `institution_category` | La institución no es un taller (mantenimiento) ni un centro de inspección mecánica (inspección) |

## Validaciones de dominio

Las tools son adaptadores delgados: validan forma y hacen el preflight de `dry_run`, pero **la fuente de verdad es el backend**. Un cambio en estas reglas se refleja en el MCP sin modificar las tools.

- **Auto**: marca, modelo y color obligatorios; color de la paleta canónica; año entre 1900 y 2100.
- **Mantenimiento**: fecha no futura, kilometraje no negativo, moneda existente e institución de categoría taller.
- **Inspección**: fecha no futura, resultado `approved`/`rejected`, año entre `auto.year + 2` y `performed_at.year + 1`, institución de categoría inspección mecánica.
- **Seguro**: cobertura `daños_a_terceros`/`full_cover`, póliza y número de aseguradora obligatorios, y servicio disponible para el auto.

## Flujo recomendado para un agente

1. `autos_list` para ubicar el `auto_id` y confirmar que el auto no exista antes de crearlo.
2. Llamar la tool de escritura con `dry_run` (default) y mostrar el plan al usuario.
3. Pedir confirmación explícita.
4. Repetir con `dry_run=false` y la compuerta activa.
5. En mantenimientos, revisar `partial` y `warnings` si alguna línea falló.
6. Verificar el resultado con `autos_list` o volviendo a consultar en la UI.

`autos_create`, `autos_maintenance_register` y `autos_inspection_register` no son idempotentes: no repitas a ciegas una escritura que no sabés si llegó a completarse. `autos_update` sí es idempotente.
