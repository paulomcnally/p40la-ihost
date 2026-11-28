---
title: "Tools de reportes de vivienda"
sidebar_position: 15
description: "Referencia de home_work_request_list, home_work_request_get, home_work_request_create, home_work_request_update, home_work_request_convert, home_work_request_status y las tools de evidencia"
---

# Tools de reportes de vivienda

Las diez tools —tres de lectura y siete de escritura— operan sobre los **reportes de problemas y trabajos deseados** de una vivienda (`home.workrequest`, SPEC-164). Un reporte nace en estado `reported`, sin costos ni líneas de reparación, y puede convertirse en una **reparación vacía** de `home.maintenance`, descartarse o reactivarse. `home_work_request_list` consulta los reportes de una vivienda y `home_work_request_get` devuelve el detalle con los metadatos de su evidencia; `home_work_request_create` da de alta un reporte; `home_work_request_update` edita su cabecera con merge parcial; `home_work_request_convert` lo convierte en reparación; y `home_work_request_status` lo descarta o reactiva. Las tools de evidencia listan, enlazan la carga, agregan y borran fotos o videos. El `home_id` es el mismo de `/homes/{id}` en la UI.

Las siete tools de escritura respetan `dry_run` (`true` por defecto) y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). Las tres de lectura son de **solo lectura**: no escriben, no dependen de la compuerta y no requieren permisos adicionales.

El MCP no accede a SQLite: cada tool llama a la API HTTP existente (`/api/homes/{id}/workrequests`, `/api/homes/{id}/workrequests/{work_request_id}`, `.../convert`, `.../status`, `.../upload-link`, `.../attachments/evidence` y `.../attachments/{attachment_id}`). La carga por link usa además el endpoint público `POST /api/work-request-uploads`, autenticado con el token efímero y no con la API key. Las tools validan la **forma** de la entrada; las **reglas de dominio** las valida el backend. Las escrituras se envían con origen `mcp` (`X-P40LA-Source`).

**La evidencia se consulta y también se carga por MCP, con límites.** `home_work_request_list` devuelve el conteo de adjuntos; `home_work_request_get` y `home_work_request_attachment_list` devuelven los metadatos de cada foto o video, nunca el binario. Para archivos grandes o varios a la vez, `home_work_request_attachment_upload_link` emite un token HMAC de corta vida y el agente sube los archivos con `curl` contra `POST /api/work-request-uploads`; para archivos pequeños, `home_work_request_attachment_add` acepta hasta **256 KiB** en base64. `home_work_request_attachment_delete` borra un adjunto. La web conserva su flujo de carga y su comportamiento; el video se almacena como evidencia, pero el agente **no lo analiza ni lo describe**. El MCP tampoco borra reportes.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `home_work_request_list` | Lectura | 1 `GET` | Lista los reportes de una vivienda con filtros de estado y categoría; devuelve `pending_count`, `truncated` y el conteo de adjuntos |
| `home_work_request_get` | Lectura | 1 `GET` | Detalle de un reporte con el vínculo a la reparación y los metadatos de su evidencia, sin binarios |
| `home_work_request_create` | Escritura | 0 en `dry_run`; 1 `POST` al escribir | Crea un reporte en estado `reported`, sin costos ni líneas |
| `home_work_request_update` | Escritura | 1 `GET` + 1 `PUT` si hay cambios | Edita la cabecera con merge parcial; no cambia el estado |
| `home_work_request_convert` | Escritura | 1 `GET` + 1 `POST` | Convierte un reporte `reported` en una reparación vacía y enlaza ambas entidades |
| `home_work_request_status` | Escritura | 1 `GET` + 1 `POST` | Descarta (`dismissed`) o reactiva (`reported`) un reporte |
| `home_work_request_attachment_list` | Lectura | 1 `GET` | Lista la evidencia con metadatos, cupo usado y límites; nunca devuelve el binario |
| `home_work_request_attachment_upload_link` | Escritura | 1 `GET` en `dry_run`; 1 `GET` + 1 `POST` al emitir | Emite un link y un token HMAC de 1..10 archivos y 1..30 minutos para cargar por HTTP |
| `home_work_request_attachment_add` | Escritura | 1 `GET` en `dry_run`; 1 `GET` + 1 `POST` multipart al escribir | Carga inline un archivo de hasta 256 KiB; deduce el tipo por firma y deduplica por `sha256` |
| `home_work_request_attachment_delete` | Escritura | 1 `GET`; +1 `DELETE` si borra | Borra un adjunto y su archivo; `dry_run` por defecto y doble borrado devuelve `unchanged` |

`home_work_request_create` no hace ninguna llamada en `dry_run` porque todos los datos vienen en la entrada. `home_work_request_update`, `home_work_request_convert` y `home_work_request_status` hacen siempre un `GET` previo para leer el estado actual; la escritura agrega el `PUT`/`POST` únicamente cuando `dry_run=false` y hay un cambio efectivo. `home_work_request_attachment_upload_link` y `home_work_request_attachment_add` hacen siempre un `GET` previo para validar estado y cupo; el `POST` solo ocurre con `dry_run=false`. `home_work_request_attachment_delete` hace el `GET` y, si el adjunto existe y `dry_run=false`, el `DELETE`. `home_work_request_attachment_list` es una única lectura.

## Estados del reporte

| Estado | Significado | Acciones MCP disponibles |
|---|---|---|
| `reported` | Pendiente: el problema o trabajo deseado está registrado y sin convertir | Editar, convertir a reparación, descartar, emitir link, cargar y borrar evidencia |
| `converted` | Ya convertido: tiene `maintenance_id` y `converted_at` | Consultar y editar; no se vuelve a convertir ni cambia de estado. La evidencia se consulta, pero no se carga ni se borra por MCP |
| `dismissed` | Descartado: se conserva el registro, pero no está pendiente | Consultar, editar, reactivar (`status=reported`) y borrar evidencia. Para cargar evidencia nueva hay que reactivarlo primero |

`home_work_request_update` no restringe por estado: un reporte `reported`, `dismissed` o `converted` conserva su estado y solo cambian los campos enviados. Los cambios de estado son acciones de negocio separadas (`convert` y `status`) y no se pueden hacer con `update`.

Las reglas de evidencia son distintas: **solo un reporte `reported` acepta un link o una carga nueva**. Un `dismissed` devuelve `invalid_status` con el hint para reactivarlo; un `converted` también rechaza la carga. El borrado se permite en `reported` y `dismissed`, y se rechaza en `converted`. La web conserva su comportamiento.

## Valores por defecto

| Campo | Tool | Default |
|---|---|---|
| `category` | `home_work_request_create` | `repair` |
| `priority` | `home_work_request_create` | `normal` |
| `reported_at` | `home_work_request_create` | Hoy, según `P40LA_TIMEZONE` |
| `kind` | `home_work_request_convert` | `corrective` |
| `area` | `home_work_request_convert` | `other` |
| `count` | `home_work_request_attachment_upload_link` | `10`, acotado al cupo libre del reporte |
| `ttl_minutes` | `home_work_request_attachment_upload_link` | `10` |
| `media_type` | `home_work_request_attachment_add` | Se deriva de la firma real del archivo (`image` o `video`) |
| `dry_run` | Las siete tools de escritura | `true` |

## Compuerta de escritura

Las lecturas no dependen de la compuerta. Las siete tools de escritura sí, pero solo cuando `dry_run=false`:

| Transporte | Requisito |
|---|---|
| stdio | `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

La compuerta se evalúa **antes** de llamar a la API (y antes del `GET` previo de `update`, `convert`, `status`, `attachment_upload_link`, `attachment_add` y `attachment_delete`), aunque después de las validaciones locales de forma. Con `dry_run=false` y las escrituras apagadas, la tool responde `writes_disabled` sin leer ni escribir:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true, o repetí con dry_run=true para previsualizar."
}
```

En stdio el `hint` solo menciona `P40LA_MCP_ALLOW_WRITES`. `dry_run=true` no pasa por la compuerta: la previsualización funciona siempre. Para emitir un link con `dry_run=false`, el backend además necesita `UPLOAD_TOKEN_SECRET` (mínimo 32 caracteres); si falta, `home_work_request_attachment_upload_link` devuelve `upload_link_disabled`. Ver [http.md](../http.md) y [security.md](../security.md).

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
      "client_request_id": "demo-msg-1",
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
- **Sin `attachments`**: el listado solo incluye `attachment_count`. Los metadatos de cada foto o video se consultan con `home_work_request_get` o `home_work_request_attachment_list`; el listado no devuelve `sha256` ni `deduplicated`.
- **`client_request_id`**: aparece en el reporte solo cuando se creó con esa clave; es la clave de idempotencia del alta, no un identificador de negocio.
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
    "client_request_id": "demo-msg-1",
    "attachments": [
      {
        "id": 1,
        "work_request_id": 10,
        "media_type": "image",
        "file_name": "foto-demo.jpg",
        "mime_type": "image/jpeg",
        "size_bytes": 1024,
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "created_at": "2026-11-26T14:05:00Z"
      },
      {
        "id": 2,
        "work_request_id": 10,
        "media_type": "video",
        "file_name": "video-demo.mp4",
        "mime_type": "video/mp4",
        "size_bytes": 2048,
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
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
- **Evidencia como metadatos**: `attachments[]` describe cada archivo (`media_type`, `file_name`, `mime_type`, `size_bytes`, `sha256`, `created_at`) pero no incluye el contenido ni una URL de descarga. El binario se ve y se descarga desde la web; se puede borrar desde la web o con `home_work_request_attachment_delete`, pero el MCP nunca devuelve el binario.
- **`sha256`**: identifica el contenido de la evidencia cargada por las vías MCP. Queda vacío (`""`) en los adjuntos cargados desde la web y en los anteriores a la migración `0063`; esos adjuntos no participan de la deduplicación. Ver [Deduplicación e idempotencia](#deduplicación-e-idempotencia).
- **`deduplicated`**: no aparece en el detalle; solo lo agregan las respuestas de alta o carga cuando el contenido ya existía. `client_request_id` aparece únicamente cuando el reporte se creó con esa clave.
- **Sin evidencia**: si el reporte no tiene adjuntos, `attachment_count` es `0` y `attachments` se omite. El backend admite hasta **10 adjuntos** por reporte y **25 MB** por archivo; esos límites aplican a la web y también a las tools de evidencia.
- **`attachment_count`** aparece en la parte superior y dentro de `work_request`; ambos valores son la cantidad de adjuntos del reporte.
- **Vínculo a la reparación**: `maintenance_id` y `converted_at` son `null` mientras el reporte no se haya convertido. Después de `home_work_request_convert` apuntan a la reparación de `home.maintenance` y al momento de la conversión.
- Es la vía recomendada para revisar la evidencia antes de convertir o descartar un reporte, junto con `home_work_request_attachment_list`.
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

Crea un reporte de problema o trabajo deseado en una vivienda. El reporte nace en estado `reported`, sin costos, sin insumos ni mano de obra. La evidencia se adjunta después con las tools de evidencia o desde la web.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `title` | string | Sí | Título del reporte; se recorta y máximo 200 caracteres |
| `description` | string | No | Detalle del problema o trabajo; máximo 2000 caracteres |
| `category` | string | No | `repair`, `improvement`, `maintenance` u `other`; default `repair` |
| `priority` | string | No | `low`, `normal` o `high`; default `normal` |
| `reported_at` | string `YYYY-MM-DD` | No | Fecha del reporte; no puede ser futura; default hoy |
| `client_request_id` | string | No | Clave de idempotencia de 1..64 caracteres `[A-Za-z0-9._:-]`; repetir el alta devuelve el mismo reporte |
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
  "client_request_id": "demo-msg-1",
  "dry_run": true
}
```

### Salida con `dry_run`

`dry_run=true` es el valor por defecto: no llama a la API y no consulta duplicados. Devuelve el reporte que se crearía con los defaults ya aplicados.

```json
{
  "status": "planned",
  "dry_run": true,
  "planned": {
    "title": "Filtro demo con fuga",
    "description": "Gotea debajo del lavabo de la cocina",
    "category": "repair",
    "priority": "high",
    "reported_at": "2026-11-26",
    "client_request_id": "demo-msg-1"
  }
}
```

### Salida con escritura

```json
{
  "status": "created",
  "dry_run": false,
  "deduplicated": false,
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
    "client_request_id": "demo-msg-1",
    "created_at": "2026-11-26T14:00:00Z",
    "updated_at": "2026-11-26T14:00:00Z"
  }
}
```

Si el `client_request_id` ya existía en la misma vivienda, la respuesta tiene la misma forma pero con `status=existing` y `deduplicated=true`, y el `work_request` es el registro existente (mismo `id`, sin crear otro reporte ni otros adjuntos). Ese `status=existing` solo se conoce al escribir: `dry_run` no consulta duplicados.

### Reglas

- **`title`** es obligatorio, se recorta con `TrimSpace` y admite hasta **200 caracteres**. Vacío o de más de 200 devuelve `invalid_request` sin llamar a la API.
- **`description`** es opcional, se recorta y admite hasta **2000 caracteres**.
- **Defaults**: sin `category` se usa `repair`; sin `priority`, `normal`; sin `reported_at`, la fecha de hoy según `P40LA_TIMEZONE`.
- **`reported_at`** debe tener formato `YYYY-MM-DD` y no puede ser futura. Una fecha futura o mal formada devuelve `invalid_request` sin llamar a la API.
- **Estado inicial**: lo fija el backend en `reported`. El reporte no tiene costos, líneas ni evidencia.
- **Idempotencia opcional**: con `client_request_id`, repetir el alta en la misma vivienda devuelve `status=existing`, `deduplicated=true` y el mismo `id`. La clave es única por vivienda y el backend resuelve la carrera con un índice único, así que dos llamadas concurrentes no crean dos reportes. Si la clave ya existe, se devuelve el reporte existente sin aplicar los demás campos de la nueva llamada.
- **Sin `client_request_id`**: no hay deduplicación y repetir el alta crea otro reporte, aunque coincidan título, fecha y categoría. Si necesitás reintentos seguros, enviá una clave estable. La tool sigue declarando `idempotentHint=false` porque sin la clave el alta no es idempotente.
- **Formato de la clave**: 1..64 caracteres, solo `[A-Za-z0-9._:-]`. Un valor fuera de esa forma devuelve `invalid_client_request_id` sin llamar a la API. Un valor vacío se trata como omitido.
- **`dry_run`**: `true` por defecto. No escribe y devuelve `status=planned` con el snapshot; si hay `client_request_id`, se incluye en `planned`. La compuerta se evalúa después de las validaciones locales y antes del `POST`.
- Para adjuntar fotos o videos, crear el reporte y luego usar `home_work_request_attachment_upload_link` (principal), `home_work_request_attachment_add` (archivos pequeños) o la web.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` no positivo, `title` vacío o de más de 200 caracteres, `description` de más de 2000 caracteres, `category` o `priority` inválidas, o `reported_at` mal formada o futura |
| `invalid_client_request_id` | `client_request_id` fuera de 1..64 caracteres o con caracteres distintos de `[A-Za-z0-9._:-]`; no llama a la API |
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

## Evidencia: alcance y límites

Las cuatro tools de evidencia operan sobre el mismo reporte y comparten estas reglas:

| Regla | Valor |
|---|---|
| Adjuntos por reporte | 10 |
| Tamaño por archivo | 25 MB (`26214400` bytes) |
| Archivos por link | 1..10 (`count`), acotado al cupo libre del reporte |
| Tamaño total del request de carga | 25 MB × `max_files` + 1 MiB |
| Vía inline | 256 KiB reales por llamada (base64 ≈ 341 KiB) |
| Imágenes aceptadas | JPEG, PNG, WebP y HEIC |
| Videos aceptados | MP4, QuickTime y WebM |
| Detección | Firma real (magic bytes); se ignoran la extensión y el `Content-Type` del cliente |
| Nombre | Se sanitiza; sin rutas, sin `..` y máximo 180 caracteres |
| Duplicado | `sha256` idéntico en el mismo reporte devuelve `duplicate` en las vías MCP |
| Video | Se almacena como evidencia; el agente **no** lo analiza ni lo describe |
| Metadatos | Los bytes se guardan tal cual: no se eliminan EXIF/GPS. Revisá las fotos antes de subirlas |

- **Solo `reported` acepta evidencia nueva.** El link de carga se emite solo para un reporte `reported`; la carga inline también exige ese estado. Un `dismissed` devuelve `invalid_status` con el hint para reactivarlo; un `converted` también rechaza la carga.
- **Borrado**: se permite en `reported` y `dismissed`; se rechaza en `converted`. La web conserva su comportamiento.
- **La web no deduplica**: sus cargas no calculan `sha256` ni comprueban duplicados, y dejan `sha256=""`. Los adjuntos previos a la migración `0063` también quedan con `sha256=""`; no se backfillean. Por eso la deduplicación solo aplica entre cargas hechas por las vías MCP.
- **La vía por link no pasa los binarios por el MCP**: `attachment_list` y `get` devuelven metadatos; la descarga sigue siendo de la web. Con `attachment_upload_link`, el agente sube los bytes directamente al backend con `curl`. En la vía inline, los bytes sí viajan como `data_base64` en los argumentos de la tool.

## `home_work_request_attachment_list`

Lista la evidencia de un reporte con sus metadatos, el cupo usado y los límites. Es la vía canónica para verificar que una carga terminó. Solo lectura.

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
  "work_request_id": 10,
  "attachment_count": 2,
  "max_attachments": 10,
  "max_bytes": 26214400,
  "remaining_slots": 8,
  "attachments": [
    {
      "id": 1,
      "work_request_id": 10,
      "media_type": "image",
      "file_name": "foto-demo.jpg",
      "mime_type": "image/jpeg",
      "size_bytes": 1024,
      "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "created_at": "2026-11-26T14:05:00Z"
    },
    {
      "id": 2,
      "work_request_id": 10,
      "media_type": "video",
      "file_name": "video-demo.mp4",
      "mime_type": "video/mp4",
      "size_bytes": 2048,
      "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
      "created_at": "2026-11-26T14:06:00Z"
    }
  ]
}
```

### Reglas

- **Solo lectura**: no depende de la compuerta ni de `dry_run`; funciona en reportes `reported`, `converted` y `dismissed`.
- **`max_attachments`** es siempre `10` y **`max_bytes`** siempre `26214400` (25 MB).
- **`remaining_slots`** es `max(0, 10 - attachment_count)`.
- **`attachments`** siempre es un array; si no hay evidencia, es `[]` y `attachment_count` es `0`.
- **`sha256`** está presente en todas las filas, pero queda vacío en las cargas de la web y en los adjuntos previos a `0063`.
- Los metadatos no incluyen el binario ni una URL de descarga. Para descargar, usá la web.
- Un reporte de otra vivienda devuelve `work_request_not_found`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` o `work_request_id` no positivos; no llama a la API |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_attachment_upload_link`

Emite un token HMAC de corta vida para subir de 1 a 10 fotos o videos a un reporte en estado `reported`. El agente ejecuta la carga con `curl` contra `upload_url`, usando el token en el header `Authorization: Bearer`. `dry_run=true` (default) previsualiza sin emitir token.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `work_request_id` | integer | Sí | ID del reporte; debe ser mayor que `0` |
| `count` | integer | No | Archivos que admitirá el link, `1..10`; default `10` y se acota al cupo libre |
| `ttl_minutes` | integer | No | Vigencia del link en minutos, `1..30`; default `10` |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "home_id": 3,
  "work_request_id": 10,
  "count": 4,
  "ttl_minutes": 10,
  "dry_run": true
}
```

### Salida con `dry_run`

No emite token; hace un `GET` para validar estado y cupo. Devuelve el link que se emitiría, con `token: null` y sin `expires_at`.

```json
{
  "status": "planned",
  "dry_run": true,
  "home_id": 3,
  "work_request_id": 10,
  "upload_url": "http://ihost.local:8088/api/work-request-uploads",
  "token": null,
  "max_bytes": 26214400,
  "max_request_bytes": 105906176,
  "max_files": 4,
  "accepted_mime": [
    "image/jpeg",
    "image/png",
    "image/webp",
    "image/heic",
    "video/mp4",
    "video/quicktime",
    "video/webm"
  ],
  "attachments_remaining": 8,
  "report_url": "http://ihost.local:8088/home/3/workrequests/10",
  "reachability": "private",
  "curl_hint": "curl -sS -X POST \"<UPLOAD_URL>\" -H \"Authorization: Bearer <TOKEN>\" -F \"file=@/ruta/demo-foto-1.jpg\""
}
```

### Salida con escritura

Con `dry_run=false` y la compuerta activa, el backend emite el token y persiste su `jti`, el cupo y el vencimiento. El ejemplo usa `P40LA_UPLOAD_PUBLIC_URL=https://uploads.example.ts.net`; sin esa variable, `upload_url` y `reachability` serían los del ejemplo planificado.

```json
{
  "status": "created",
  "dry_run": false,
  "home_id": 3,
  "work_request_id": 10,
  "upload_url": "https://uploads.example.ts.net/api/work-request-uploads",
  "token": "v1.demo-payload.demo-signature",
  "expires_at": "2026-11-27T15:10:00Z",
  "max_bytes": 26214400,
  "max_request_bytes": 105906176,
  "max_files": 4,
  "accepted_mime": [
    "image/jpeg",
    "image/png",
    "image/webp",
    "image/heic",
    "video/mp4",
    "video/quicktime",
    "video/webm"
  ],
  "attachments_remaining": 8,
  "report_url": "https://ihost.example.ts.net/home/3/workrequests/10",
  "reachability": "public",
  "curl_hint": "curl -sS -X POST \"<UPLOAD_URL>\" -H \"Authorization: Bearer <TOKEN>\" -F \"file=@/ruta/demo-foto-1.jpg\""
}
```

### Reglas

- **`count`**: si se omite, pide `10`; el backend lo acota al cupo libre del reporte. Si el cupo es `0`, la tool devuelve `attachment_limit` antes de previsualizar o emitir. Un `count` explícito fuera de `1..10` devuelve `invalid_request`.
- **`ttl_minutes`**: `1..30`; default `10`. El MCP envía el TTL explícito (10 si se omite), así que el default `UPLOAD_LINK_TTL` del backend no cambia el TTL pedido por esta tool. Un valor fuera de rango devuelve `invalid_request`.
- **`upload_url`**: se compone con `P40LA_UPLOAD_PUBLIC_URL` si está definida; si no, con `P40LA_BASE_URL`. Puede incluir un prefijo de proxy. El token **nunca** va en la URL.
- **`token`**: cadena `v1.<payload>.<firma>` con HMAC-SHA256; solo se devuelve con `dry_run=false`. Va en el header `Authorization: Bearer <token>`, no en query string, logs ni mensajes de error.
- **`reachability`**: `public` si `P40LA_UPLOAD_PUBLIC_URL` está definida; `private` si `upload_url` usa `P40LA_BASE_URL`. Es una clasificación de configuración, no una prueba de conectividad desde el sandbox del agente.
- **`max_bytes`**: 25 MB por archivo. **`max_request_bytes`**: 25 MB × `max_files` + 1 MiB.
- **`attachments_remaining`**: cupo total libre del reporte (no el cupo de este link). **`max_files`** es el cupo efectivo del token.
- **`report_url`**: URL del detalle del reporte en la web, compuesta con `P40LA_BASE_URL`. Sirve para el fallback humano cuando el agente no puede alcanzar `upload_url`.
- **`curl_hint`**: comando de ejemplo con los placeholders `<UPLOAD_URL>` y `<TOKEN>`; no contiene el token real. Para varios archivos, repetí `-F "file=@..."`.
- **Estado**: solo un reporte `reported` acepta el link. `dismissed` o `converted` devuelven `invalid_status`.
- **Secreto**: con `dry_run=false`, el backend necesita `UPLOAD_TOKEN_SECRET` de al menos 32 caracteres. Si falta, devuelve `upload_link_disabled` (HTTP 503). `dry_run=true` no consulta ese secreto.
- **No idempotente**: repetir la emisión crea un token nuevo para el mismo reporte. El token anterior sigue vigente hasta su vencimiento o hasta agotar su cupo.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` o `work_request_id` no positivos; `count` fuera de `1..10`; `ttl_minutes` fuera de `1..30` |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `invalid_status` | El reporte no está `reported` (por ejemplo, `dismissed` o `converted`) |
| `attachment_limit` | El reporte ya tiene 10 adjuntos; no hay cupo para un archivo nuevo |
| `upload_link_disabled` | Falta `UPLOAD_TOKEN_SECRET` en el backend (HTTP 503) |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_attachment_add`

Carga inline una foto o un video pequeño (hasta **256 KiB reales**) en un reporte `reported`. El contenido viaja en base64, el backend valida la firma real y deduplica por `sha256`. Es la vía para clientes que pueden inyectar bytes en los argumentos de la tool; un modelo de chat no puede transcribir un binario, así que para archivos grandes o varios archivos usá `home_work_request_attachment_upload_link`.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `work_request_id` | integer | Sí | ID del reporte; debe ser mayor que `0` |
| `filename` | string | Sí | Nombre con extensión, por ejemplo `foto-demo.jpg`; se sanitiza en el backend |
| `mime` | string | No | MIME declarado: `image/jpeg`, `image/png`, `image/webp`, `image/heic`, `video/mp4`, `video/quicktime` o `video/webm` |
| `media_type` | string | No | `image` o `video`; si se omite, se deriva de la firma real del archivo. Si no coincide con la firma, se rechaza con `invalid_file` |
| `data_base64` | string | Sí | Contenido real en base64; máximo 256 KiB decodificados |
| `sha256` | string | No | SHA-256 hexadecimal de 64 caracteres; si no coincide con el contenido, se rechaza |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Ejemplo

```json
{
  "home_id": 3,
  "work_request_id": 10,
  "filename": "foto-demo.jpg",
  "mime": "image/jpeg",
  "media_type": "image",
  "data_base64": "ZGVtby1ieXRlcw==",
  "dry_run": true
}
```

### Salida con `dry_run`

No hace `POST`. Devuelve el resumen de lo que se subiría; no incluye un objeto `attachment` porque todavía no existe.

```json
{
  "status": "planned",
  "dry_run": true,
  "filename": "foto-demo.jpg",
  "size_bytes": 10,
  "media_type": "image",
  "mime_type": "image/jpeg",
  "attachment_count": 2
}
```

### Salida con escritura

```json
{
  "status": "added",
  "dry_run": false,
  "attachment": {
    "id": 21,
    "work_request_id": 10,
    "media_type": "image",
    "file_name": "foto-demo.jpg",
    "mime_type": "image/jpeg",
    "size_bytes": 1024,
    "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
    "created_at": "2026-11-27T15:00:00Z"
  },
  "attachment_count": 3
}
```

Si el mismo contenido ya existía en el reporte, devuelve `status=duplicate` con el adjunto existente (marcado `deduplicated: true`) y no crea fila ni archivo nuevos.

### Reglas

- **`data_base64`**: se decodifica en el MCP. Base64 inválido o vacío devuelve `invalid_request`; más de **256 KiB** decodificados devuelve `invalid_request` con el hint a `home_work_request_attachment_upload_link`.
- **`filename`**: obligatorio; el backend lo sanitiza (sin rutas, sin `..`, máximo 180 caracteres). Si no tiene extensión, el backend agrega la que corresponde al tipo detectado.
- **`mime` y `media_type`**: `mime`, si se envía, debe pertenecer a la allowlist. `media_type`, si se envía, debe ser `image` o `video` y **coincidir con la firma real** del archivo; una incoherencia devuelve `invalid_file` antes de llamar a la API. El backend vuelve a **detectar el tipo por magic bytes** y guarda ese tipo/MIME; no confía en la extensión ni en el `Content-Type`.
- **`sha256`**: opcional. Si se envía, debe tener 64 caracteres hexadecimales y coincidir con el contenido; una discrepancia devuelve `invalid_request` antes de llamar a la API. Si se omite, el backend calcula el hash igualmente.
- **Idempotencia**: repetir el mismo contenido en el mismo reporte devuelve `status=duplicate` con el adjunto existente, sin crear otro. La unicidad la garantiza un índice parcial `(work_request_id, sha256)` para hashes no vacíos.
- **`attachment_count`**: la respuesta de escritura devuelve el conteo previo más uno (o el previo si fue duplicado). Si necesitás el valor canónico después de una carrera, verificá con `home_work_request_attachment_list`.
- **Estado y cupo**: solo `reported`; un reporte con 10 adjuntos devuelve `attachment_limit`. El `GET` previo permite rechazar `converted`/`dismissed` antes de escribir.
- **Llamadas**: `dry_run=true` hace 1 `GET`; `dry_run=false` hace 1 `GET` + 1 `POST` multipart a `.../attachments/evidence` con la API key.
- **`dry_run`**: `true` por defecto; la compuerta se evalúa antes del `GET` cuando `dry_run=false`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | IDs no positivos; `filename` o `data_base64` vacíos; base64 inválido; más de 256 KiB; `mime` fuera de la allowlist; `media_type` distinto de `image`/`video`; `sha256` mal formado o que no coincide |
| `invalid_file` | El contenido no es una firma permitida o `media_type`/`mime` no coinciden con la firma real |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `invalid_status` | El reporte no está `reported` |
| `attachment_limit` | El reporte ya tiene 10 adjuntos |
| `invalid_file` | El backend no pudo detectar un tipo permitido o el archivo está vacío |
| `request_too_large` | El archivo supera los 25 MB (no debería ocurrir con el límite inline, pero el backend lo valida) |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## `home_work_request_attachment_delete`

Borra un adjunto de un reporte y su archivo. `dry_run=true` (default) solo previsualiza. Se permite en `reported` y `dismissed`; se rechaza en `converted`. Si el adjunto ya no existe, devuelve `status=unchanged` sin error.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `work_request_id` | integer | Sí | ID del reporte; debe ser mayor que `0` |
| `attachment_id` | integer | Sí | ID del adjunto; debe ser mayor que `0` |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin borrar |

### Ejemplo

```json
{
  "home_id": 3,
  "work_request_id": 10,
  "attachment_id": 1,
  "dry_run": true
}
```

### Salida con `dry_run`

```json
{
  "status": "planned",
  "dry_run": true,
  "attachment_id": 1,
  "deleted": false,
  "attachment_count": 2
}
```

### Salida con escritura

```json
{
  "status": "deleted",
  "dry_run": false,
  "attachment_id": 1,
  "deleted": true,
  "attachment_count": 1
}
```

### Salida sin cambios

Si el adjunto no existe (o ya se borró), no es un error:

```json
{
  "status": "unchanged",
  "dry_run": false,
  "attachment_id": 1,
  "deleted": false,
  "attachment_count": 1
}
```

### Reglas

- **`dry_run`**: `true` por defecto. Hace el `GET` y, si el adjunto existe, devuelve `planned` sin `DELETE`.
- **Estados**: se permite en `reported` y `dismissed`; un reporte `converted` devuelve `invalid_status` incluso al previsualizar.
- **Idempotente**: borrar dos veces deja el mismo estado final; el segundo intento devuelve `status=unchanged`, `deleted=false`, sin `DELETE`.
- **`attachment_count`**: en `planned` y `unchanged` es el conteo actual; en `deleted` es el conteo menos uno (mínimo `0`). El backend borra la fila y el archivo físico.
- **Solo evidencia**: no borra el reporte, no cambia su estado y no afecta la reparación enlazada.
- **Llamadas**: `GET` siempre; `DELETE .../attachments/{attachment_id}` solo si el adjunto existe y `dry_run=false`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id`, `work_request_id` o `attachment_id` no positivos; no llama a la API |
| `writes_disabled` | `dry_run=false` sin la compuerta de escritura; no lee ni escribe |
| `invalid_status` | El reporte está `converted` y la evidencia histórica no se puede borrar por MCP |
| `home_not_found` | La vivienda no existe |
| `work_request_not_found` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

`attachment_not_found` del backend se traduce a `status=unchanged` en esta tool; no se propaga como error.

## Flujo crear → link → curl → list

El flujo recomendado para cargar evidencia desde un agente es:

1. **Crear el reporte con una clave estable.** Usá `client_request_id` para que un reintento no duplique el alta. Confirmá con `dry_run=false` después de la aprobación del usuario.
2. **Previsualizar el link.** Llamá `home_work_request_attachment_upload_link` con `dry_run=true` y el `count` previsto. Revisá `max_files`, `attachments_remaining`, `reachability` y `report_url`.
3. **Emitir el link.** Repetí con `dry_run=false`. Guardá `upload_url`, `token` y `expires_at` solo en memoria; no los escribas en logs ni en la URL.
4. **Subir los archivos.** Ejecutá `curl` con una parte `file` por archivo, hasta `max_files`:

   ```bash
   curl -sS -X POST "$UPLOAD_URL" \
     -H "Authorization: Bearer $TOKEN" \
     -F "file=@/tmp/demo-foto-1.jpg" \
     -F "file=@/tmp/demo-video-1.mp4"
   ```

5. **Leer el resultado por archivo.** El endpoint devuelve `added`, `duplicates`, `rejected` y un `results[]` con `file_name`, `status` y, cuando aplica, el `attachment` creado o existente. Un archivo rechazado no aborta el resto del lote; si el único archivo del request supera 25 MB, el endpoint responde `413 request_too_large`.
6. **Verificar con `home_work_request_attachment_list`.** Confirmá `attachment_count` y `remaining_slots`; es la vía canónica para cerrar el flujo.
7. **Fallback humano.** Si `reachability=private` y el sandbox no alcanza el backend, o si el `curl` falla, abrí `report_url` en la web y usá el botón «Adjuntar» para cargar los archivos de a uno. El reporte ya creado no se duplica si conservás el mismo `client_request_id`; después verificá con `attachment_list`.

### Endpoint `POST /api/work-request-uploads`

| Aspecto | Comportamiento |
|---|---|
| Autenticación | `Authorization: Bearer <token>`; **no** acepta sesión ni API key global |
| Content-Type | `multipart/form-data` con 1..`max_files` partes llamadas `file` |
| Tipo de archivo | Se infiere por magic bytes; se ignoran extensión y `Content-Type` de la parte |
| Tamaño | 25 MB por archivo; `max_files` × 25 MB + 1 MiB por request |
| Cupo del token | Se consume solo con archivos nuevos; duplicados y rechazos no consumen cupo |
| Rate limit | 30 requests por minuto por hash de token; `429` con `Retry-After` |
| Respuesta | `added`, `duplicates`, `rejected`, `attachment_count`, `remaining_slots` y `results[]` |
| Errores por archivo | `status=rejected` con `error` y `message`; el lote continúa |
| Cabeceras | `Cache-Control: no-store` y `X-Content-Type-Options: nosniff` |

Ejemplo de respuesta:

```json
{
  "home_id": 3,
  "work_request_id": 10,
  "added": 1,
  "duplicates": 1,
  "rejected": 0,
  "attachment_count": 3,
  "remaining_slots": 7,
  "results": [
    {
      "file_name": "demo-foto-1.jpg",
      "status": "added",
      "attachment": {
        "id": 21,
        "work_request_id": 10,
        "media_type": "image",
        "file_name": "demo-foto-1.jpg",
        "mime_type": "image/jpeg",
        "size_bytes": 1024,
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "created_at": "2026-11-27T15:00:00Z"
      }
    },
    {
      "file_name": "demo-foto-1.jpg",
      "status": "duplicate",
      "attachment": {
        "id": 21,
        "work_request_id": 10,
        "media_type": "image",
        "file_name": "demo-foto-1.jpg",
        "mime_type": "image/jpeg",
        "size_bytes": 1024,
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "created_at": "2026-11-27T15:00:00Z",
        "deduplicated": true
      }
    }
  ]
}
```

Si el token ya usó todos sus archivos, el endpoint responde `409 upload_token_exhausted` al iniciar el request; si el cupo se agota durante el lote, los archivos sobrantes se rechazan con ese código y los anteriores quedan guardados. Un token emitido para un reporte que se convierte o se descarta antes de la carga responde `409 invalid_status` sin escribir.

## Deduplicación e idempotencia

| Operación | Clave | Repetición |
|---|---|---|
| `home_work_request_create` con `client_request_id` | `(home_id, client_request_id)` | `status=existing`, `deduplicated=true`, mismo `id`; no crea adjuntos |
| `home_work_request_create` sin `client_request_id` | — | Crea otro reporte |
| `home_work_request_attachment_upload_link` | — | Emite un token nuevo; no es idempotente. El token anterior sigue vigente hasta vencer o agotar su cupo |
| Carga por token | `(work_request_id, sha256)` | `status=duplicate` con el adjunto existente; no crea fila ni archivo ni consume cupo |
| `home_work_request_attachment_add` | `(work_request_id, sha256)` | `status=duplicate` con el adjunto existente; no crea otro |
| `home_work_request_attachment_delete` | `attachment_id` | El segundo intento devuelve `status=unchanged`, `deleted=false` |
| `home_work_request_update` | Campos enviados | `status=unchanged` sin `PUT` si nada cambia |
| `home_work_request_status` | Estado destino | `status=unchanged` sin `POST` si ya está en ese estado |

- La deduplicación de adjuntos es **por reporte**: el mismo contenido en otro reporte se guarda como adjunto nuevo.
- La migración `0063_work_request_evidence_idempotency` agrega `client_request_id` y `sha256`, los índices parciales únicos y la tabla efímera `work_request_upload_tokens(jti, work_request_id, max_files, files_used, expires_at)`. Es aditiva y no borra reportes ni adjuntos existentes.
- La unicidad de `client_request_id` y de `sha256` se garantiza con índices parciales únicos; el backend resuelve carreras y devuelve el registro existente.
- La web no calcula `sha256` ni deduplica; sus adjuntos quedan con `sha256=""` y no bloquean una carga MCP del mismo contenido.
- No hay reintentos automáticos de escrituras: si el `curl` falla a mitad de un lote, consultá `attachment_list` antes de reintentar. Los archivos ya guardados no se duplican.

## Errores de evidencia

| Código | HTTP | Cuándo |
|---|---|---|
| `invalid_request` | — | Validación local del MCP: IDs, `count`, `ttl_minutes`, `filename`, base64, tamaño inline, MIME declarado, `sha256` o `media_type` inválidos |
| `invalid_client_request_id` | — | `client_request_id` con formato inválido en `home_work_request_create` |
| `invalid_status` | `409` | El reporte no está `reported` para cargar, o está `converted` para borrar |
| `attachment_limit` | `409` | El reporte ya tiene 10 adjuntos o no queda cupo |
| `attachment_not_found` | `404` | El adjunto no existe; en `attachment_delete` se traduce a `status=unchanged` |
| `invalid_token` | `401` | Token de carga ausente, mal formado o con firma inválida |
| `expired_token` | `401` | Token de carga vencido |
| `upload_token_exhausted` | `409` | El token ya usó todos sus archivos |
| `invalid_file` | `400` | Archivo vacío, tipo no permitido, multipart ilegible o sin partes `file` |
| `unsupported_media` | `415` | El request de carga no es `multipart/form-data` |
| `request_too_large` | `413` | Un archivo supera 25 MB o el request supera el tope del lote |
| `upload_link_disabled` | `503` | Falta `UPLOAD_TOKEN_SECRET` en el backend |
| `too_many_requests` | `429` | Rate limit del endpoint de carga (30/min por hash de token); incluye `Retry-After` |
| `writes_disabled` | — | `dry_run=false` sin la compuerta de escritura; no llama a la API |
| `home_not_found` | `404` | La vivienda no existe |
| `work_request_not_found` | `404` | El reporte no existe o no pertenece al `home_id` |
| `auth_failed` | — | API key del MCP inválida, revocada o rotada |
| `upstream_error` | — | Error de red, timeout o respuesta con un sobre distinto al esperado |

Los errores de negocio del MCP responden con `isError=true` y `{"error", "message", "hint"?}`. El endpoint de carga devuelve `{"error", "message"}` y el estado HTTP indicado; los errores por archivo viajan dentro de `results[]` sin abortar el lote.

## Errores del backend y validaciones

Las tools validan la forma antes de llamar a la API y devuelven `invalid_request`, `invalid_client_request_id` o `invalid_status` con un mensaje accionable. El backend conserva códigos granulares (`title_required`, `title_too_long`, `description_too_long`, `invalid_category`, `invalid_priority`, `invalid_date`, `future_date`, `invalid_kind`, `invalid_area`, `invalid_file`, `attachment_limit`, `upload_token_exhausted`, `invalid_token`, `expired_token`, `upload_link_disabled`, `request_too_large`, `unsupported_media`, `too_many_requests`); si llegan por una carrera entre la lectura y la escritura, se propagan tal cual. Los errores de negocio responden con `isError=true` y el objeto `{"error", "message", "hint"?}`.

Los ejemplos usan datos sintéticos (`Casa Demo`, `Filtro demo`, `demo-*`, `demo-token`, hashes `0000…`).
