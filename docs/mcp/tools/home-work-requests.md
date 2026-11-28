---
title: "Tools de reportes de vivienda"
sidebar_position: 15
description: "Referencia de home_work_request_list, home_work_request_get, home_work_request_create, home_work_request_update, home_work_request_convert y home_work_request_status"
---

# Tools de reportes de vivienda

Las seis tools —dos de lectura y cuatro de escritura— operan sobre los **reportes de problemas y trabajos deseados** de una vivienda (`home.workrequest`, SPEC-164). Un reporte nace en estado `reported`, sin costos ni líneas de reparación, y puede convertirse en una **reparación vacía** de `home.maintenance`, descartarse o reactivarse. `home_work_request_list` consulta los reportes de una vivienda y `home_work_request_get` devuelve el detalle con los metadatos de su evidencia; `home_work_request_create` da de alta un reporte; `home_work_request_update` edita su cabecera con merge parcial; `home_work_request_convert` lo convierte en reparación; y `home_work_request_status` lo descarta o reactiva. El `home_id` es el mismo de `/homes/{id}` en la UI.

Las cuatro tools de escritura respetan `dry_run` (`true` por defecto) y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). Las herramientas de lectura son de **solo lectura**: no escriben, no dependen de la compuerta y no requieren permisos adicionales.

El MCP no accede a SQLite: cada tool llama a la API HTTP existente (`/api/homes/{id}/workrequests`, `/api/homes/{id}/workrequests/{work_request_id}`, `.../convert` y `.../status`). Las tools validan la **forma** de la entrada; las **reglas de dominio** las valida el backend. Las escrituras se envían con origen `mcp` (`X-P40LA-Source`).

**La evidencia se consulta, no se carga.** `home_work_request_list` devuelve el conteo de adjuntos y `home_work_request_get` los metadatos de cada foto o video; el MCP nunca devuelve el binario. La carga, la descarga y el borrado de evidencia se hacen desde la web. El MCP tampoco borra reportes.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `home_work_request_list` | Lectura | 1 `GET` | Lista los reportes de una vivienda con filtros de estado y categoría; devuelve `pending_count`, `truncated` y el conteo de adjuntos |
| `home_work_request_get` | Lectura | 1 `GET` | Detalle de un reporte con el vínculo a la reparación y los metadatos de su evidencia, sin binarios |
| `home_work_request_create` | Escritura | 0 en `dry_run`; 1 `POST` al escribir | Crea un reporte en estado `reported`, sin costos ni líneas |
| `home_work_request_update` | Escritura | 1 `GET` + 1 `PUT` si hay cambios | Edita la cabecera con merge parcial; no cambia el estado |
| `home_work_request_convert` | Escritura | 1 `GET` + 1 `POST` | Convierte un reporte `reported` en una reparación vacía y enlaza ambas entidades |
| `home_work_request_status` | Escritura | 1 `GET` + 1 `POST` | Descarta (`dismissed`) o reactiva (`reported`) un reporte |

`home_work_request_create` no hace ninguna llamada en `dry_run` porque todos los datos vienen en la entrada. `home_work_request_update`, `home_work_request_convert` y `home_work_request_status` hacen siempre un `GET` previo para leer el estado actual; la escritura agrega el `PUT`/`POST` únicamente cuando `dry_run=false` y hay un cambio efectivo.

## Estados del reporte

| Estado | Significado | Acciones MCP disponibles |
|---|---|---|
| `reported` | Pendiente: el problema o trabajo deseado está registrado y sin convertir | Editar, convertir a reparación, descartar |
| `converted` | Ya convertido: tiene `maintenance_id` y `converted_at` | Consultar y editar; no se vuelve a convertir ni cambia de estado |
| `dismissed` | Descartado: se conserva el registro, pero no está pendiente | Consultar, editar y reactivar (`status=reported`) |

`home_work_request_update` no restringe por estado: un reporte `reported`, `dismissed` o `converted` conserva su estado y solo cambian los campos enviados. Los cambios de estado son acciones de negocio separadas (`convert` y `status`) y no se pueden hacer con `update`.

## Valores por defecto

| Campo | Tool | Default |
|---|---|---|
| `category` | `home_work_request_create` | `repair` |
| `priority` | `home_work_request_create` | `normal` |
| `reported_at` | `home_work_request_create` | Hoy, según `P40LA_TIMEZONE` |
| `kind` | `home_work_request_convert` | `corrective` |
| `area` | `home_work_request_convert` | `other` |
| `dry_run` | Las cuatro tools de escritura | `true` |

## Compuerta de escritura

Las lecturas no dependen de la compuerta. Las cuatro tools de escritura sí, pero solo cuando `dry_run=false`:

| Transporte | Requisito |
|---|---|
| stdio | `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

La compuerta se evalúa **antes** de llamar a la API (y antes del `GET` previo de `update`, `convert` y `status`), aunque después de las validaciones locales de forma. Con `dry_run=false` y las escrituras apagadas, la tool responde `writes_disabled` sin leer ni escribir:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true, o repetí con dry_run=true para previsualizar."
}
```

En stdio el `hint` solo menciona `P40LA_MCP_ALLOW_WRITES`. `dry_run=true` no pasa por la compuerta: la previsualización funciona siempre. Ver [http.md](../http.md) y [security.md](../security.md).

## `home_work_request_list`

Lista los reportes de una vivienda, con filtros opcionales de estado y categoría. Devuelve el conteo de pendientes, si la respuesta quedó truncada y cada reporte con su conteo de adjuntos. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `status` | string | No | Filtra por estado: `reported`, `converted` o `dismissed` |
| `category` | string | No | Filtra por categoría: `repair`, `improvement`, `maintenance` u `other` |

Los filtros se combinan con AND. Un valor vacío o solo con espacios se ignora.

### Ejemplo

```json
{
  "home_id": 3,
  "status": "reported",
  "category": "repair"
}
```

### Salida

```json
{
  "home_id": 3,
  "count": 2,
  "pending_count": 2,
  "truncated": false,
  "work_requests": [
    {
      "id": 10,
      "home_id": 3,
      "title": "Filtro demo con fuga",
      "description": "Gotea debajo del lavabo de la cocina",
      "category": "repair",
      "priority": "high",
      "status": "reported",
      "reported_at": "2026-11-26",
      "maintenance_id": null,
      "converted_at": null,
      "attachment_count": 2,
      "created_at": "2026-11-26T14:00:00Z",
      "updated_at": "2026-11-26T14:00:00Z"
    },
    {
      "id": 9,
      "home_id": 3,
      "title": "Pintura demo del portal",
      "description": "La pared del portal está descascarada",
      "category": "repair",
      "priority": "normal",
      "status": "reported",
      "reported_at": "2026-11-20",
      "maintenance_id": null,
      "converted_at": null,
      "attachment_count": 0,
      "created_at": "2026-11-20T09:00:00Z",
      "updated_at": "2026-11-20T09:00:00Z"
    }
  ]
}
```

### Reglas

- **Solo lectura**: no depende de la compuerta ni de `dry_run`; funciona siempre, con o sin `P40LA_MCP_ALLOW_WRITES`.
- **`pending_count`** cuenta **todos** los reportes en estado `reported` de la vivienda, sin aplicar los filtros de la llamada. Por eso puede ser mayor que `count`, por ejemplo al filtrar `status=converted`.
- **`count`** es la cantidad de reportes devueltos, no el total del sistema.
- **`truncated`**: el listado devuelve hasta **500 reportes**. Si hay más, `truncated=true` y la respuesta sigue siendo válida: es un subconjunto, no un error.
- **Sin `attachments`**: el listado solo incluye `attachment_count`. Los metadatos de cada foto o video se consultan con `home_work_request_get`.
- **Orden**: `reported_at` descendente y, en empate, `id` descendente.
- **`maintenance_id` y `converted_at`** son `null` mientras el reporte no se haya convertido; después, apuntan a la reparación creada y al momento de la conversión.
- Una vivienda inexistente devuelve `home_not_found`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` es `0` o negativo, `status` distinto de `reported`/`converted`/`dismissed` o `category` fuera de `repair`/`improvement`/`maintenance`/`other`; no llama a la API |
| `home_not_found` | La vivienda no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_get`

Devuelve el detalle de un reporte: título, descripción, categoría, prioridad, estado, fecha, vínculo a la reparación cuando fue convertido y los metadatos de su evidencia (fotos y videos, sin binarios). Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `work_request_id` | integer | Sí | ID del reporte; debe ser mayor que `0` |

### Ejemplo

```json
{
  "home_id": 3,
  "work_request_id": 10
}
```

### Salida

```json
{
  "home_id": 3,
  "attachment_count": 2,
  "work_request": {
    "id": 10,
    "home_id": 3,
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "status": "reported",
    "reported_at": "2026-11-26",
    "maintenance_id": null,
    "converted_at": null,
    "attachment_count": 2,
    "attachments": [
      {
        "id": 1,
        "work_request_id": 10,
        "media_type": "image",
        "file_name": "foto-demo.jpg",
        "mime_type": "image/jpeg",
        "size_bytes": 1024,
        "created_at": "2026-11-26T14:05:00Z"
      },
      {
        "id": 2,
        "work_request_id": 10,
        "media_type": "video",
        "file_name": "video-demo.mp4",
        "mime_type": "video/mp4",
        "size_bytes": 2048,
        "created_at": "2026-11-26T14:06:00Z"
      }
    ],
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-26T14:00:00Z"
  }
}
```

### Reglas

- **Solo lectura**: no depende de la compuerta ni de `dry_run`.
- **Evidencia como metadatos**: `attachments[]` describe cada archivo (`media_type`, `file_name`, `mime_type`, `size_bytes`, `created_at`) pero no incluye el contenido ni una URL de descarga. El binario se ve, se descarga y se borra desde la web.
- **Sin evidencia**: si el reporte no tiene adjuntos, `attachment_count` es `0` y `attachments` se omite. El backend admite hasta **10 adjuntos** por reporte y **25 MB** por archivo; esos límites aplican a la carga desde la web, no a esta tool.
- **`attachment_count`** aparece en la parte superior y dentro de `work_request`; ambos valores son la cantidad de adjuntos del reporte.
- **Vínculo a la reparación**: `maintenance_id` y `converted_at` son `null` mientras el reporte no se haya convertido. Después de `home_work_request_convert` apuntan a la reparación de `home.maintenance` y al momento de la conversión.
- Es la vía recomendada para revisar la evidencia antes de convertir o descartar un reporte.
- Un reporte de otra vivienda devuelve `work_request_not_found`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` o `work_request_id` no positivos; no llama a la API |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_create`

Crea un reporte de problema o trabajo deseado en una vivienda. El reporte nace en estado `reported`, sin costos, sin insumos ni mano de obra. La evidencia se adjunta después desde la web.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `title` | string | Sí | Título del reporte; se recorta y máximo 200 caracteres |
| `description` | string | No | Detalle del problema o trabajo; máximo 2000 caracteres |
| `category` | string | No | `repair`, `improvement`, `maintenance` u `other`; default `repair` |
| `priority` | string | No | `low`, `normal` o `high`; default `normal` |
| `reported_at` | string `YYYY-MM-DD` | No | Fecha del reporte; no puede ser futura; default hoy |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "home_id": 3,
  "title": "Filtro demo con fuga",
  "description": "Gotea debajo del lavabo de la cocina",
  "category": "repair",
  "priority": "high",
  "reported_at": "2026-11-26",
  "dry_run": true
}
```

### Salida con `dry_run`

`dry_run=true` es el valor por defecto: no llama a la API. Devuelve el reporte que se crearía con los defaults ya aplicados.

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "reported_at": "2026-11-26"
  }
}
```

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "work_request": {
    "id": 10,
    "home_id": 3,
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "status": "reported",
    "reported_at": "2026-11-26",
    "maintenance_id": null,
    "converted_at": null,
    "attachment_count": 0,
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-26T14:00:00Z"
  }
}
```

### Reglas

- **`title`** es obligatorio, se recorta con `TrimSpace` y admite hasta **200 caracteres**. Vacío o de más de 200 devuelve `invalid_request` sin llamar a la API.
- **`description`** es opcional, se recorta y admite hasta **2000 caracteres**.
- **Defaults**: sin `category` se usa `repair`; sin `priority`, `normal`; sin `reported_at`, la fecha de hoy según `P40LA_TIMEZONE`.
- **`reported_at`** debe tener formato `YYYY-MM-DD` y no puede ser futura. Una fecha futura o mal formada devuelve `invalid_request` sin llamar a la API.
- **Estado inicial**: lo fija el backend en `reported`. El reporte no tiene costos, líneas ni evidencia.
- **Sin deduplicación**: repetir el alta crea otro reporte, aunque coincidan título, fecha y categoría. Revisá lo existente con `home_work_request_list` antes de crear.
- **`dry_run`**: `true` por defecto. No escribe y devuelve `status=planned` con el snapshot. La compuerta se evalúa después de las validaciones locales y antes del `POST`.
- Para adjuntar fotos o videos, crear el reporte y luego cargar la evidencia desde la web.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` no positivo, `title` vacío o de más de 200 caracteres, `description` de más de 2000 caracteres, `category` o `priority` inválidas, o `reported_at` mal formada o futura |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `home_not_found` | La vivienda no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_update`

Edita la **cabecera** de un reporte con merge parcial: solo cambian los campos enviados y lo omitido conserva su valor. El estado no se modifica. El backend usa semántica de **reemplazo**, así que la tool lee el reporte actual y envía un `PUT` con los cinco campos editables completos.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `work_request_id` | integer | Sí | ID del reporte; debe ser mayor que `0` |
| `title` | string | No | Título hasta 200 caracteres; vacío es inválido. Omitir conserva |
| `description` | string | No | Descripción hasta 2000 caracteres; vacío limpia. Omitir conserva |
| `category` | string | No | `repair`, `improvement`, `maintenance` u `other`. Omitir conserva |
| `priority` | string | No | `low`, `normal` o `high`. Omitir conserva |
| `reported_at` | string `YYYY-MM-DD` | No | Fecha no futura. Omitir conserva |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

Debe venir **al menos un campo editable**; si no, `invalid_request`. `dry_run` no cuenta como campo editable.

### Ejemplo

```json
{
  "home_id": 3,
  "work_request_id": 10,
  "title": "Filtro demo revisado",
  "priority": "low",
  "dry_run": true
}
```

### Salida con `dry_run`

`dry_run=true` (default) hace solo el `GET` del reporte y devuelve los campos que cambiarían, el estado actual y el propuesto.

```json
{
  "status": "planned",
  "dry_run": true,
  "work_request_id": 10,
  "changed_fields": ["title", "priority"],
  "current": {
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "reported_at": "2026-11-26"
  },
  "proposed": {
    "title": "Filtro demo revisado",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "low",
    "reported_at": "2026-11-26"
  }
}
```

### Salida con escritura

```json
{
  "status": "updated",
  "dry_run": false,
  "work_request_id": 10,
  "changed_fields": ["title", "priority"],
  "work_request": {
    "id": 10,
    "home_id": 3,
    "title": "Filtro demo revisado",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "low",
    "status": "reported",
    "reported_at": "2026-11-26",
    "maintenance_id": null,
    "converted_at": null,
    "attachment_count": 2,
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-27T09:30:00Z"
  }
}
```

### Salida sin cambios

```json
{
  "status": "unchanged",
  "dry_run": false,
  "work_request_id": 10,
  "changed_fields": [],
  "work_request": {
    "id": 10,
    "home_id": 3,
    "title": "Filtro demo revisado",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "low",
    "status": "reported",
    "reported_at": "2026-11-26",
    "maintenance_id": null,
    "converted_at": null,
    "attachment_count": 2,
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-27T09:30:00Z"
  }
}
```

`status=unchanged` **no es un error**: la tool responde con `isError=false`, el registro actual y `changed_fields: []`, y **no ejecuta el `PUT`**, ni siquiera con `dry_run=false`.

### Reglas

- **Merge parcial**: se compara el valor efectivo campo por campo. `changed_fields` lista solo los que difieren, en el orden `title`, `description`, `category`, `priority`, `reported_at`.
- **Cuerpo completo**: el `PUT` siempre lleva los cinco campos editables, porque el backend reemplaza la cabecera. Los omitidos salen con el valor que ya tenían.
- **`title`**: se recorta; vacío devuelve `invalid_request` y más de 200 caracteres también. A diferencia de `description`, no se puede limpiar.
- **`description`**: se recorta; vacía limpia la descripción; más de 2000 caracteres devuelve `invalid_request`.
- **`category` y `priority`**: si se envían, deben ser valores válidos y no vacíos; un valor inválido devuelve `invalid_request`.
- **`reported_at`**: formato `YYYY-MM-DD` y no futura según `P40LA_TIMEZONE`. Una fecha mal formada o futura devuelve `invalid_request` sin llamar al `PUT`.
- **Estado intacto**: la tool no envía `status`; el reporte conserva `reported`, `converted` o `dismissed`. Para descartar o reactivar usá `home_work_request_status`; para convertir, `home_work_request_convert`.
- **No toca la evidencia**: `attachments` y `attachment_count` quedan intactos.
- **`dry_run`**: `true` por defecto. Hace solo el `GET` y devuelve `planned` con `current` y `proposed`. La compuerta se evalúa antes del `GET` cuando `dry_run=false`.
- **Idempotente**: repetir la misma edición devuelve `status=unchanged` sin `PUT`; `updated_at` solo cambia cuando hay una escritura real.
- **Llamadas HTTP**: `GET` siempre; `PUT` solo si hay cambios y `dry_run=false`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` o `work_request_id` no positivos; ningún campo editable; `title` vacío o de más de 200 caracteres; `description` de más de 2000 caracteres; `category` o `priority` inválidas; `reported_at` mal formada o futura |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_convert`

Convierte un reporte pendiente (`status=reported`) en una **reparación vacía** de `home.maintenance`, copia su título y descripción, enlaza `maintenance_id` y deja el reporte en `converted`. La reparación nace sin líneas ni costos; los insumos y la mano de obra se agregan después con las tools `home_maintenance_*`.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `work_request_id` | integer | Sí | ID del reporte; debe ser mayor que `0` |
| `kind` | string | No | Tipo de reparación: `corrective` o `preventive`; default `corrective` |
| `area` | string | No | `plumbing`, `electrical`, `roof`, `paint`, `carpentry`, `appliances`, `garden`, `cleaning` u `other`; default `other` |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "home_id": 3,
  "work_request_id": 10,
  "kind": "corrective",
  "area": "plumbing",
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "action": "convert_work_request",
    "work_request_id": 10,
    "title": "Filtro demo con fuga",
    "kind": "corrective",
    "area": "plumbing",
    "maintenance_id": null
  }
}
```

### Salida con escritura

```json
{
  "status": "converted",
  "dry_run": false,
  "maintenance_id": 42,
  "work_request": {
    "id": 10,
    "home_id": 3,
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "status": "converted",
    "reported_at": "2026-11-26",
    "maintenance_id": 42,
    "converted_at": "2026-11-27T15:00:00Z",
    "attachment_count": 2,
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-27T15:00:00Z"
  }
}
```

### Reglas

- **Solo reportes pendientes**: un reporte `reported` se convierte; uno `converted` devuelve `already_converted`; uno `dismissed` devuelve `invalid_status`. La comprobación ocurre **antes** del `dry_run`, así que `already_converted` también aparece al previsualizar.
- **Reparación vacía**: el backend crea la cabecera en `home.maintenance` con `performed_at` del día, `kind` y `area` indicados, y `0` líneas y `0` costos. El detalle posterior se consulta con `home_maintenance_get`.
- **Descripción copiada**: la reparación recibe el título del reporte y, si hay descripción, el texto separado por una línea en blanco. El total se recorta a **500 caracteres**, el máximo de la descripción de una reparación.
- **Enlace**: el reporte guarda `maintenance_id`, `converted_at` y `status=converted`. La conversión es condicional en el backend: si dos llamadas compiten, solo una enlaza y la otra devuelve `already_converted`.
- **La evidencia no se mueve**: fotos y videos permanecen en el reporte; la reparación nueva no hereda adjuntos.
- **Defaults**: `kind=corrective` y `area=other` cuando se omiten.
- **`dry_run`**: `true` por defecto. Hace solo el `GET` del reporte y devuelve `planned`; no crea la reparación ni modifica el reporte.
- **No idempotente en sentido estricto**: repetir la conversión devuelve `already_converted` (error), no un éxito repetido. Para verificar el resultado usá `home_work_request_get` u `home_maintenance_get`.
- **Después de convertir**: agregá mano de obra e insumos con `home_maintenance_labor_register` y `home_maintenance_supplies_register`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` o `work_request_id` no positivos; `kind` fuera de `corrective`/`preventive`; `area` inválida |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `already_converted` | El reporte ya fue convertido |
| `invalid_status` | El reporte no está `reported` (por ejemplo, `dismissed`) |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_status`

Descarta un reporte pendiente (`status=dismissed`) o lo reactiva (`status=reported`). No admite `converted` ni otros estados y no crea ni borra reparaciones.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `work_request_id` | integer | Sí | ID del reporte; debe ser mayor que `0` |
| `status` | string | Sí | Estado destino: `dismissed` para descartar o `reported` para reactivar |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "home_id": 3,
  "work_request_id": 10,
  "status": "dismissed",
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "work_request_id": 10,
  "from_status": "reported",
  "to_status": "dismissed"
}
```

### Salida con escritura

```json
{
  "status": "updated",
  "dry_run": false,
  "work_request_id": 10,
  "from_status": "reported",
  "to_status": "dismissed",
  "work_request": {
    "id": 10,
    "home_id": 3,
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "status": "dismissed",
    "reported_at": "2026-11-26",
    "maintenance_id": null,
    "converted_at": null,
    "attachment_count": 2,
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-27T09:45:00Z"
  }
}
```

### Salida sin cambios

```json
{
  "status": "unchanged",
  "dry_run": false,
  "work_request_id": 10,
  "work_request": {
    "id": 10,
    "home_id": 3,
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "status": "reported",
    "reported_at": "2026-11-26",
    "maintenance_id": null,
    "converted_at": null,
    "attachment_count": 2,
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-26T14:00:00Z"
  }
}
```

`status=unchanged` **no es un error**: el reporte ya estaba en el estado destino y la tool no ejecuta el `POST`.

### Reglas

- **Estados permitidos**: solo `dismissed` o `reported`. Cualquier otro valor, incluido `converted`, devuelve `invalid_status` sin llamar a la API.
- **Convertido no cambia de estado**: si el reporte ya está `converted`, devuelve `invalid_status`; la reparación enlazada no se toca.
- **Descarte**: `reported → dismissed` conserva el reporte, su evidencia y su historial; solo cambia el estado y `updated_at`.
- **Reactivación**: `dismissed → reported` vuelve a ponerlo pendiente. No crea una reparación ni recupera una conversión anterior.
- **Sin efecto sobre reparaciones**: la tool no crea, edita ni borra registros de `home.maintenance`; solo cambia el estado del reporte.
- **`dry_run`**: `true` por defecto. Hace solo el `GET` para conocer `from_status` y devuelve `planned`; no ejecuta el `POST`.
- **Idempotente**: repetir el mismo estado devuelve `status=unchanged` sin `POST`. El segundo intento no es un error.
- **Llamadas HTTP**: `GET` siempre; `POST .../status` solo si hay cambio y `dry_run=false`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_status` | `status` no es `reported`/`dismissed`, o el reporte está `converted` y no puede cambiar de estado |
| `invalid_request` | `home_id` o `work_request_id` no positivos |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## Errores del backend y validaciones

Las tools validan la forma antes de llamar a la API y devuelven `invalid_request` o `invalid_status` con un mensaje accionable. El backend conserva códigos granulares (`title_required`, `title_too_long`, `description_too_long`, `invalid_category`, `invalid_priority`, `invalid_date`, `future_date`, `invalid_kind`, `invalid_area`); si llegan por una carrera entre la lectura y la escritura, se propagan tal cual. Los errores de negocio responden con `isError=true` y el objeto `{"error", "message", "hint"?}`.

Los ejemplos usan datos sintéticos (`Casa Demo`, `Filtro demo`, `demo-*`).
