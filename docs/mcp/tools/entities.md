---
title: "Tools de entidades"
sidebar_label: "Tools de entidades"
sidebar_position: 3
description: "Referencia de entities_search, services_get, homes_create, homes_update, institutions_create, banks_create y services_create"
---

# Tools de entidades

Las cuatro tools de alta crean entidades base del sistema: hogares, instituciones, bancos y servicios. Todas **escriben**, comprueban duplicados antes del POST y no tienen `dry_run`: se ejecutan en una sola llamada y dependen de la compuerta de escritura.

`homes_update` completa el paquete con la **edición** de un hogar existente. Resuelve el hogar por `home_id` o por nombre, hace edición parcial, rechaza un nombre que ya pertenece a otro hogar y reutiliza `PUT /api/homes/{id}`; tampoco tiene `dry_run` y depende de la misma compuerta.

`entities_search` es la contraparte de **solo lectura**: busca esas mismas entidades por nombre y devuelve sus IDs para usarlos en las tools de alta, en `homes_update` o en las de facturas.

`services_get` completa esa lectura con el **detalle** de un servicio: configuración de facturación, vigencia, contrato y último estado de factura, sin escribir nada.

## `entities_search`

Busca servicios, hogares, bancos e instituciones por nombre y devuelve los candidatos con su `id` y el contexto necesario para desambiguarlos. **No distingue mayúsculas ni acentos** y ordena los resultados por calidad de coincidencia.

Es una tool de **solo lectura**: no requiere `P40LA_MCP_ALLOW_WRITES` ni `P40LA_MCP_REMOTE_WRITES`, no crea ni modifica nada y está disponible en stdio y HTTP.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `entity` | string | Sí | `service`, `home`, `bank` o `institution` |
| `query` | string | No | Texto a buscar por nombre, hasta 100 caracteres. Vacío lista las primeras entidades |
| `home` | string | No | Solo para `service`: nombre del hogar para filtrar |
| `home_id` | integer | No | Solo para `service`: ID del hogar; gana sobre `home` |
| `limit` | integer | No | Máximo de resultados, `1..50`; default `10` |

`home` y `home_id` solo aplican a `service` y se ignoran en las demás entidades. Si vienen ambos, gana `home_id`. `home` se resuelve por nombre exacto normalizado; si no existe devuelve `not_found` y si hay más de un hogar con ese nombre devuelve `ambiguous_reference` con los candidatos en el `hint`.

### Coincidencias y orden

- La comparación normaliza mayúsculas, acentos (`á` → `a`, `ñ` → `n`) y espacios al inicio y al final.
- Puntaje: **exacto** (`0`) → **prefijo** (`1`) → **contenido** (`2`). Sin `query`, todas las entidades se consideran con puntaje `3`.
- Se ordena por puntaje ascendente; en empate, por nombre normalizado y luego por `id`.
- Los bancos también coinciden por `short_code`; se conserva el mejor puntaje entre nombre y código.

### Salida

```json
{
  "entity": "home",
  "query": "demo",
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 7,
      "name": "Hogar Demo",
      "address": "Calle Demo 123",
      "usage": "rental"
    }
  ]
}
```

| Campo | Descripción |
|---|---|
| `entity` | Entidad buscada, normalizada a minúsculas |
| `query` | Texto buscado, ya recortado (`""` si se listó sin filtro) |
| `count` | Cantidad de items devueltos |
| `total` | Total de coincidencias encontradas |
| `truncated` | `true` cuando `total` supera `limit` |
| `items` | Candidatos, con los campos de la entidad |
| `hint` | Solo aparece cuando `total` es `0`; explica cómo seguir |

Campos por entidad (los opcionales se omiten cuando están vacíos):

| `entity` | Campos de cada item |
|---|---|
| `home` | `id`, `name`, `address`, `usage` |
| `service` | `id`, `name`, `home_id`, `institution`, `currency_id`, `frequency`, `suggested_amount`, `active` |
| `bank` | `id`, `name`, `short_code`, `country`, `active` |
| `institution` | `id`, `name`, `category_id` |

Sin coincidencias **no es un error**: la tool devuelve `count: 0`, `items: []` y un `hint`, con `isError=false`.

La tool nunca elige una entidad por su cuenta: devuelve todos los candidatos con contexto (`home_id`, `short_code`) para que el agente los muestre y el usuario desambigüe.

### Ejemplo: servicios de un hogar

```json
{
  "entity": "service",
  "query": "internet",
  "home": "Hogar Demo",
  "limit": 5
}
```

```json
{
  "entity": "service",
  "query": "internet",
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 12,
      "name": "Internet Demo",
      "home_id": 7,
      "institution": "Proveedor Demo",
      "currency_id": 1,
      "frequency": "monthly",
      "suggested_amount": 1000.00,
      "active": true
    }
  ]
}
```

En el ejemplo, `currency_id: 1` corresponde a la moneda `NIO`. Sin `query` y con `home_id: 7`, la misma búsqueda listaría los primeros servicios del hogar, por ejemplo `Internet Demo` y `Servicio Demo`.

### Ejemplo: banco por código corto

```json
{
  "entity": "bank",
  "query": "DEMO"
}
```

```json
{
  "entity": "bank",
  "query": "DEMO",
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 2,
      "name": "Banco Demo",
      "short_code": "DEMO",
      "country": "NI",
      "active": true
    }
  ]
}
```

### Ejemplo: sin coincidencias

```json
{
  "entity": "home",
  "query": "zzz"
}
```

```json
{
  "entity": "home",
  "query": "zzz",
  "count": 0,
  "total": 0,
  "truncated": false,
  "items": [],
  "hint": "No se encontraron coincidencias. Probá con otro término o sin query para listar."
}
```

### Errores

| Código | Cuándo |
|---|---|
| `invalid_value` | `entity` fuera de `service`, `home`, `bank` o `institution`, o `limit` fuera de `1..50` |
| `invalid_request` | `query` supera los 100 caracteres |
| `not_found` | El filtro `home` no coincide con ningún hogar; el `hint` lista hasta 10 opciones `id=nombre` |
| `ambiguous_reference` | El filtro `home` coincide con más de un hogar; el `hint` trae los candidatos y hay que usar `home_id` |
| `api_error` | La API rechazó la consulta con un error de negocio propio |
| `upstream_error` | Error de red o respuesta del servidor no interpretable |

`entities_search` nunca devuelve `writes_disabled`: no escribe y no consulta la compuerta.

### Flujo recomendado: buscar → elegir ID → operar

1. Llamar `entities_search` con `entity` y `query`; para servicios, agregar `home` o `home_id` si se conoce el hogar.
2. Si `count` es `0`, probar otro término o repetir sin `query` para listar. No es un error: usar el `hint`.
3. Si hay varios candidatos, mostrarlos al usuario con su contexto (`home_id`, `short_code`) y pedir que elija.
4. Usar el `id` elegido en `bills_list`, `bills_generate` o `bills_pay`, en `homes_update` o en las tools de alta (`services_create`, `institutions_create`, etc.).
5. Si `truncated` es `true`, subir `limit` o afinar `query` antes de concluir que no existe la entidad.

## Requisitos comunes

Estos requisitos aplican a las cuatro tools de alta (`homes_create`, `institutions_create`, `banks_create`, `services_create`) y a `homes_update`. `entities_search` y `services_get` son de solo lectura y no dependen de ellos.

### Compuerta de escritura

La compuerta se evalúa **antes** de validar la entrada. Sin ella, la tool devuelve `writes_disabled` y no llama a la API.

| Transporte | Requisito |
|---|---|
| stdio | `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

### Resolución por nombre o ID

Las tools de alta y `homes_update` aceptan referencias por nombre o por ID donde aplica. La comparación ignora mayúsculas, acentos y espacios sobrantes.

- **Hogar y moneda** (`services_create`): exactamente uno de `home`/`home_id` y uno de `currency`/`currency_id`.
- **Hogar** (`homes_update`): `home` o `home_id`; si vienen ambos, gana el ID. Sin ninguno de los dos devuelve `invalid_request`.
- **Institución** (`services_create`): `institution` o `institution_id`; si vienen ambos, gana el ID.
- **Categoría** (`institutions_create`): `category` acepta clave o nombre; si viene `category_id`, gana el ID.
- Si la referencia por nombre no existe: `not_found`, con un `hint` de hasta 10 opciones en formato `id=nombre` (moneda usa `id=código`, categoría `id=clave`).
- Si más de una entidad coincide: `ambiguous_reference`, con los candidatos en el `hint`. La solución es repetir con el `*_id`.

### Edición parcial

`homes_update` completa `name`, `address` y `usage` con el valor actual antes de enviar el `PUT`, así que el agente solo envía lo que cambia. No hay `dry_run`: la tool resuelve el hogar, valida la entrada y escribe en una sola llamada.

### Duplicados

Las tools de alta comparan siempre el nombre normalizado. `homes_update` solo compara cuando el `name` cambia: si el nombre nuevo pertenece a otro hogar, devuelve `already_exists` con el `id` y `name` del existente y **no** ejecuta el PUT. Cambiar solo mayúsculas o acentos del mismo nombre no se considera duplicado.

Cada tool de alta hace un GET de la lista correspondiente y compara el nombre normalizado:

- Hogares, instituciones y bancos: nombre único en toda la tabla.
- Servicios: nombre único **dentro del hogar**.

En un alta, si el nombre ya existe, la tool devuelve `already_exists` con el `id` existente y **no** ejecuta el POST.

Una tool de alta no es idempotente en el sentido estricto: repetir la misma llamada devuelve `already_exists` en lugar de un éxito nuevo. Si una llamada falla por red después de crear el registro, reintentar y recibir `already_exists` con el `id` es el resultado correcto: tratá ese `id` como el alta ya realizada.

## `homes_create`

Crea un hogar.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `name` | string | Sí | Nombre del hogar |
| `address` | string | No | Dirección |
| `usage` | string | No | `rental` (default), `inhabited` o `unused` |

### Ejemplo

```json
{
  "name": "Hogar Demo",
  "address": "Calle Demo 123",
  "usage": "rental"
}
```

### Salida

```json
{
  "id": 7,
  "name": "Hogar Demo",
  "address": "Calle Demo 123",
  "usage": "rental"
}
```

## `homes_update`

Edita un hogar existente con **edición parcial**: lo que se omite conserva su valor actual. Requiere la compuerta de escritura y **no tiene `dry_run`**.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Selector | ID del hogar a editar |
| `home` | string | Selector | Nombre del hogar a editar |
| `name` | string | No | Nombre nuevo; si se omite, conserva el actual |
| `address` | string | No | Dirección nueva; si se omite, conserva la actual y `""` la limpia |
| `usage` | string | No | `rental`, `inhabited` o `unused`; si se omite, conserva el actual |

Hace falta `home_id` o `home`; si vienen ambos, gana `home_id`. Sin ninguno de los dos, la tool devuelve `invalid_request`.

### Ejemplo

```json
{
  "home": "Hogar Demo",
  "usage": "inhabited"
}
```

### Salida

```json
{
  "id": 7,
  "name": "Hogar Demo",
  "address": "Calle Demo 123",
  "usage": "inhabited"
}
```

### Reglas

- `usage` se normaliza a minúsculas; un valor fuera de `rental`, `inhabited` o `unused` devuelve `invalid_value`.
- `name` vacío o solo espacios devuelve `invalid_request`; para conservar el nombre, omitilo.
- Si `name` cambia a un nombre que ya pertenece a otro hogar, devuelve `already_exists` con el `id` y `name` del existente y **no** ejecuta el PUT. Cambiar solo mayúsculas o acentos del mismo nombre no se considera duplicado.
- `address` vacío limpia la dirección; para conservarla, omitila.
- Si el hogar tiene un contrato o reserva activa, no se puede cambiar el uso a `inhabited` o `unused`: la API responde `409 home_has_active_rental` y la tool lo devuelve con un hint accionable. `name` y `address` sí se pueden editar.
- Es idempotente: repetir la misma edición deja el hogar en el mismo estado.

### Errores propios

| Código | Cuándo |
|---|---|
| `already_exists` | El `name` nuevo ya pertenece a otro hogar (comparación normalizada); el `hint` trae el `id` y el `name` del existente y no se ejecuta el PUT |
| `home_has_active_rental` | El hogar tiene un contrato o reserva activa y se intentó cambiar el uso fuera de `rental`; el `hint` pide cerrar el contrato o la reserva y aclara que `name`/`address` se pueden editar |

Los errores de resolución (`not_found`, `ambiguous_reference`), de validación (`invalid_request`, `invalid_value`) y de compuerta (`writes_disabled`) están en la tabla general.

## `institutions_create`

Crea una institución o proveedor.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `name` | string | Sí | Nombre de la institución |
| `category_id` | integer | No | ID de la categoría; gana sobre `category` |
| `category` | string | No | Clave o nombre de la categoría |

`category` se resuelve contra `GET /api/institution-categories`. Acepta la clave (`servicios`) o el nombre (`Servicios`). Si se envían `category_id` y `category`, se usa `category_id`. Si no se envía ninguno, no se manda `category_id` al backend.

### Ejemplo

```json
{
  "name": "Proveedor Demo",
  "category": "servicios"
}
```

### Salida

```json
{
  "id": 4,
  "name": "Proveedor Demo",
  "category_id": 3
}
```

## `banks_create`

Crea un banco.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `name` | string | Sí | Nombre del banco |
| `short_code` | string | No | Código corto |
| `country` | string ISO-2 | No | Dos letras; se normaliza a mayúsculas (`NI`) |
| `notes` | string | No | Notas |
| `active` | boolean | No | Default `true` |

`country` debe tener exactamente dos letras A-Z. Cualquier otro valor devuelve `invalid_value`.

### Ejemplo

```json
{
  "name": "Banco Demo",
  "short_code": "BDEMO",
  "country": "NI",
  "notes": "Sucursal demo",
  "active": true
}
```

### Salida

```json
{
  "id": 2,
  "name": "Banco Demo",
  "short_code": "BDEMO",
  "country": "NI",
  "notes": "Sucursal demo",
  "active": true
}
```

## `services_create`

Crea un servicio asociado a un hogar y una moneda.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `name` | string | Sí | Nombre del servicio |
| `home` | string | Selector | Nombre del hogar |
| `home_id` | integer | Selector | ID del hogar |
| `currency` | string | Selector | Código o nombre de la moneda |
| `currency_id` | integer | Selector | ID de la moneda |
| `institution` | string | No | Nombre de la institución |
| `institution_id` | integer | No | ID de la institución; gana sobre `institution` |
| `frequency` | string | No | `monthly` (default) o `yearly` |
| `suggested_amount` | number | No | Monto sugerido; default `0`; no puede ser negativo |
| `icon_key` | string | No | Clave de ícono; default `other` |
| `billing_type` | string | No | `fixed` o `variable` (default) |
| `billing_day` | integer | No | Día de facturación, `1..31` |
| `active` | boolean | No | Default `true`; con `true` se genera la factura del período actual |
| `auto_generate` | boolean | No | Generación automática; default `false` |
| `is_recurring` | boolean | No | Recurrente; default `false` |
| `start_date` | string `YYYY-MM-DD` | No | Fecha de inicio |
| `end_date` | string `YYYY-MM-DD` | No | Fecha de fin |

Se debe indicar **exactamente uno** de `home`/`home_id` y **exactamente uno** de `currency`/`currency_id`. Enviar ambos o ninguno devuelve `invalid_request`.

### Reglas

- `frequency` y `billing_type` se normalizan a minúsculas; los valores fuera de la lista devuelven `invalid_value`.
- `suggested_amount` negativo, `billing_day` fuera de `1..31` o fechas con formato inválido devuelven `invalid_value`.
- El nombre es único dentro del hogar: si ya existe un servicio con ese nombre, la tool devuelve `already_exists` y no crea nada.
- Con `active=true` (default), el backend genera la factura del período actual y la salida informa `current_bill_generated: true`.
- Con `active=false`, no se genera factura y `current_bill_generated` es `false`.

### Ejemplo

```json
{
  "name": "Internet Demo",
  "home": "Hogar Demo",
  "currency": "NIO",
  "institution": "Proveedor Demo",
  "frequency": "monthly",
  "suggested_amount": 1000.00,
  "billing_type": "fixed",
  "billing_day": 5,
  "active": true
}
```

### Salida

```json
{
  "id": 12,
  "name": "Internet Demo",
  "frequency": "monthly",
  "suggested_amount": 1000.00,
  "active": true,
  "currency_id": 1,
  "home_id": 7,
  "institution": "Proveedor Demo",
  "icon_key": "other",
  "billing_type": "fixed",
  "billing_day": 5,
  "current_bill_generated": true
}
```

## Errores

Todos los errores devuelven `isError=true` y un objeto `{error, message, hint?}`.

| Código | Cuándo |
|---|---|
| `invalid_request` | Falta `name` en un alta, falta el hogar o la moneda, se enviaron ambos selectores o `homes_update` recibió `name` vacío o ningún selector de hogar |
| `invalid_value` | `usage` (en alta o edición), `country`, `frequency`, `billing_type`, `billing_day`, `suggested_amount` o fechas fuera de rango |
| `writes_disabled` | La compuerta de escritura está apagada; se evalúa antes de validar |
| `not_found` | La referencia por nombre o ID no existe; el `hint` lista hasta 10 opciones `id=nombre`. En `homes_update` aplica tanto a `home_id` como a `home` |
| `ambiguous_reference` | Más de una entidad coincide con el nombre; el `hint` trae los candidatos. Repetir con `home_id`, `currency_id`, `institution_id` o `category_id` según la tool |
| `home_has_active_rental` | El hogar tiene un contrato o reserva activa y se intentó cambiar el uso a `inhabited` o `unused` (409); el `hint` aclara que `name` y `address` sí se pueden editar |
| `already_exists` | Ya existe una entidad con ese nombre; el `hint` trae el `id` existente y su nombre. En `homes_update` aplica cuando el `name` nuevo pertenece a otro hogar |
| `api_error` | La API rechazó la operación con un error de negocio propio |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red o respuesta del servidor no interpretable |

### Ejemplo: referencia ambigua

```json
{
  "error": "ambiguous_reference",
  "message": "Hay 2 hogares que coinciden con \"Hogar Demo\".",
  "hint": "Usá home_id. Opciones: 7=Hogar Demo, 8=Hogar Demo"
}
```

### Ejemplo: duplicado

```json
{
  "error": "already_exists",
  "message": "Ya existe un hogar con ese nombre.",
  "hint": "Usá el existente: id=7, name=\"Hogar Demo\""
}
```

## `services_get`

Devuelve el detalle completo de un servicio: facturación, día de cobro, recurrencia, vigencia, vínculo con automatización, contrato y último estado de factura. Es la contraparte de detalle de `entities_search` y **no crea facturas**.

Es de **solo lectura**: no requiere la compuerta de escritura. A diferencia de `GET /api/services` (listado, que reconcilia facturas del período), este endpoint no escribe.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `service_id` | integer | Sí | ID del servicio; debe ser mayor que `0` |

### Ejemplo

```json
{
  "service_id": 12
}
```

### Salida

```json
{
  "id": 12,
  "home_id": 3,
  "name": "Servicio Demo",
  "institution": "Proveedor Demo",
  "institution_id": 7,
  "currency_id": 1,
  "frequency": "monthly",
  "suggested_amount": 1000.00,
  "active": true,
  "icon_key": "internet",
  "billing_type": "fixed",
  "billing_day": 5,
  "auto_generate": false,
  "is_recurring": true,
  "is_insurance": false,
  "start_date": "2026-01-01",
  "latest_bill_status": "pending",
  "has_contract": true,
  "created_at": "2026-01-01T12:00:00Z",
  "updated_at": "2026-11-01T12:00:00Z"
}
```

### Reglas

- `frequency` es `monthly` o `yearly`; `billing_type` es `fixed` o `variable`.
- `billing_day` puede ser `null` si el servicio no define día de facturación.
- `latest_bill_status` es `pending`, `paid` o `null` si el servicio no tiene facturas.
- `has_contract` indica si existe un PDF de contrato en el volumen; no lo descarga.
- `start_date` y `end_date` aparecen solo si están definidas; `end_date` marca el fin de vigencia.
- `automation_account_id` aparece cuando el servicio está vinculado a una cuenta de automatización.
- Un `service_id` inexistente devuelve `not_found` (la API responde `404`).
- Un `service_id` menor o igual a `0` devuelve `invalid_request` sin llamar a la API.

### Privacidad y datos ocultos

La tool **no aplica allowlist**: devuelve el registro del servicio tal como lo entrega la API. Si el servicio tiene webhook propio, la respuesta puede incluir `webhook_uuid`, que es un secreto (`SEC-NO-PII`). No lo reproduzcas en respuestas, logs ni capturas, y no lo uses para construir URLs.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `service_id` es `0` o negativo |
| `not_found` | El servicio no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red o respuesta del servidor no interpretable |

## Flujo recomendado para un agente

1. Reunir los datos mínimos con el usuario: nombre y, para servicios, hogar y moneda; para editar un hogar, los campos que cambian.
2. **Buscar antes de crear o editar**: usar `entities_search` para reutilizar hogares, instituciones, bancos o servicios existentes y evitar duplicados. Con el `home_id` devuelto, `homes_update` edita el hogar sin reenviar el resto de los campos. Con el `id` del servicio, `services_get` da el detalle antes de crear facturas o cambiarlo.
3. Respetar las dependencias: hogar → institución o banco → servicio.
4. Mostrar al usuario lo que se va a crear o editar y pedir confirmación explícita; no hay `dry_run`.
5. Ejecutar una llamada por entidad.
6. Si llega `not_found` o `ambiguous_reference`, usar el `hint` o `entities_search` para elegir el `*_id` correcto o preguntar al usuario.
7. Si llega `already_exists` en un alta o en un cambio de nombre, no reintentar con otro nombre a ciegas: usar el `id` del `hint` o pedir otro nombre al usuario.
8. Reportar el `id` creado o el hogar actualizado y, en servicios, si se generó la factura del período actual.
9. Si `homes_update` devuelve `home_has_active_rental`, no reintentar con otro uso: cerrar el contrato o la reserva, o editar solo `name` y `address`.

No reintentes una escritura que falló por red sin revisar el resultado: el alta pudo completarse y el reintento devolverá `already_exists` con el `id`.
