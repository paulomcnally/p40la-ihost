---
title: "Tools de banca"
sidebar_label: "Tools de banca"
sidebar_position: 4
description: "Referencia de bank_accounts_search, bank_accounts_create, bank_accounts_update, credit_cards_search, credit_cards_create y credit_cards_update"
---

# Tools de banca

Las seis tools operan sobre las **cuentas bancarias** y las **tarjetas de crédito** de los bancos registrados. El MCP no accede a SQLite: usa los endpoints HTTP existentes. Como no hay un listado global de cuentas ni de tarjetas, las búsquedas recorren los bancos (`/api/banks`) y consultan los listados por banco (`/api/banks/{id}/accounts` y `/api/banks/{id}/credit-cards`).

Las tools aceptan el **banco** y la **moneda** por nombre, código o ID, resuelven la referencia y llaman a la API con los IDs. Las búsquedas son de solo lectura y funcionan siempre; las altas y ediciones escriben y quedan detrás de la compuerta de escritura.

## Resumen

| Tool | Tipo | Qué hace | Idempotente |
|---|---|---|---|
| `bank_accounts_search` | Lectura | Busca cuentas por alias, número, beneficiario o banco | — |
| `bank_accounts_create` | Escritura | Crea una cuenta bancaria | No (repetir devuelve `already_exists`) |
| `bank_accounts_update` | Escritura | Edita una cuenta sin cambiar el banco | Sí |
| `credit_cards_search` | Lectura | Busca tarjetas por alias, últimos cuatro, titular o banco | — |
| `credit_cards_create` | Escritura | Crea una tarjeta de crédito con los defaults del sistema | No (repetir devuelve `already_exists`) |
| `credit_cards_update` | Escritura | Edita una tarjeta sin cambiar el banco | Sí |

## Compuerta de escritura

Las búsquedas no dependen de la compuerta. Las altas y ediciones sí:

| Transporte | Requisito |
|---|---|
| stdio | `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

La compuerta se evalúa antes de resolver el banco y la moneda: si está apagada, la tool responde `writes_disabled` y no llama a la API.

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true (y P40LA_MCP_REMOTE_WRITES=true en modo HTTP) para ejecutar cambios."
}
```

Ver [http.md](../http.md) y [security.md](../security.md).

## Resolución de banco y moneda

### Banco

`bank` acepta el `name`, el `short_code` o el ID como texto; `bank_id` acepta el ID numérico. Si vienen ambos, gana `bank_id`. La comparación ignora mayúsculas, acentos y espacios sobrantes, y prueba en orden **exacto → prefijo → contiene**.

| Situación | Resultado |
|---|---|
| Una sola coincidencia | Se usa ese banco |
| Ninguna | `not_found` con los bancos disponibles en el `hint` |
| Varias con la misma prioridad | `ambiguous_reference` con los candidatos |

```json
{
  "error": "ambiguous_reference",
  "message": "Hay más de un banco que coincide con «Demo».",
  "hint": "Enviá bank_id para desambiguar. Candidatos: Banco Demo, Banco Demo Dos."
}
```

### Moneda

`currency` acepta el código (`NIO`, `USD`), el nombre o el ID como texto; `currency_id` acepta el ID. Si vienen ambos, gana `currency_id`. Los valores válidos son los que devuelve `GET /api/currencies`, y la comparación sigue el mismo orden **exacto → prefijo → contiene**.

Si la moneda no existe, la tool responde `not_found` con las opciones disponibles:

```json
{
  "error": "not_found",
  "message": "No existe una moneda que coincida con «XXX».",
  "hint": "Opciones disponibles: NIO, USD."
}
```

Si varias coinciden con la misma prioridad, responde `ambiguous_reference` con los códigos para desambiguar.

## Búsqueda

`bank_accounts_search` y `credit_cards_search` comparten el contrato. `query` es opcional y se compara contra estos campos:

| Campo | Cuentas | Tarjetas |
|---|---|---|
| Alias | `alias` | `alias` |
| Número | `account_number` | `last4` (últimos cuatro dígitos) |
| Persona | `beneficiary` | `holder` (titular) |
| Marca | — | `network` |
| Banco | `bank_name` | `bank_name` |

Reglas:

- Sin `query` se listan las primeras `limit` entidades que pasen el filtro.
- La comparación ignora mayúsculas y acentos, y recorta espacios al inicio y al final.
- Orden: coincidencia exacta, luego prefijo, luego contiene. Dentro de cada grupo, por alias y `id` ascendente.
- `active`: `true` solo activas, `false` solo inactivas, omitido devuelve ambas.
- `limit`: default 10, rango 1..50. Un valor fuera de rango devuelve `invalid_value`.
- `bank`/`bank_id` limitan la búsqueda a un solo banco.

### Salida

```json
{
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": []
}
```

- `count`: entidades devueltas.
- `total`: coincidencias antes de aplicar `limit`.
- `truncated`: `true` cuando hay más coincidencias que `count`.

Sin resultados **no es un error**: devuelve `count: 0`, `items: []` y un `hint`.

```json
{
  "count": 0,
  "total": 0,
  "truncated": false,
  "items": [],
  "hint": "No hay cuentas que coincidan. Probá con otro texto, revisá active o creá una con bank_accounts_create."
}
```

### Rendimiento

Sin `bank`/`bank_id`, la búsqueda hace una llamada a `/api/banks` y una por cada banco: **1 + B llamadas HTTP**, donde B es la cantidad de bancos. Con `bank` o `bank_id` se resuelve el banco y se consulta solo su listado: dos llamadas. Si conocés el banco, mandalo.

## Duplicados

- **Cuenta**: mismo `bank_id` + `account_number`, sin distinguir mayúsculas y sin espacios sobrantes.
- **Tarjeta**: mismo `bank_id` + `alias`, sin distinguir mayúsculas ni acentos.

La verificación corre antes de crear y antes de editar (excluyendo la propia entidad). Si hay coincidencia, la tool responde `already_exists` con el `id` existente y **no escribe**.

```json
{
  "error": "already_exists",
  "message": "Ya existe una tarjeta con ese alias en Banco Demo.",
  "hint": "Usá credit_cards_update con id=1 o cambiá alias."
}
```

## Edición parcial

`bank_accounts_update` y `credit_cards_update` leen la entidad, mezclan **solo los campos enviados** y mandan el `PUT` completo que exige la API (que reemplaza el recurso). Así, omitir un campo nunca lo pone en cero.

- Campo ausente = sin cambios.
- Cadena vacía = limpia el campo (`notes: ""`, `holder: ""`).
- `secondary_currency: ""` (o `secondary_currency_id: 0`) limpia la moneda secundaria de una tarjeta.
- Los numéricos aceptan `0` explícito cuando el rango lo permite (`bonus_due_day: 0`, `credit_limit: 0`).
- No envíes `null` para limpiar: usá `""`.
- El **banco no se cambia**: `bank` y `bank_id` se rechazan con `invalid_value`.
- Cada edición lee la entidad y el listado del banco antes del `PUT`, y es idempotente: repetir el mismo payload deja el mismo estado.

## `bank_accounts_search`

Busca cuentas bancarias. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `query` | string | No | Texto libre sobre alias, número de cuenta, beneficiario y banco |
| `bank` | string | No | Nombre, `short_code` o ID como texto del banco |
| `bank_id` | integer | No | ID del banco; si viene con `bank`, gana `bank_id` |
| `active` | boolean | No | Filtra por estado; omitido devuelve activas e inactivas |
| `limit` | integer | No | Máximo de resultados, 1..50; default 10 |

### Ejemplo

```json
{
  "query": "demo",
  "bank": "Banco Demo",
  "active": true,
  "limit": 10
}
```

### Salida

```json
{
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 1,
      "bank_id": 1,
      "bank_name": "Banco Demo",
      "alias": "Cuenta Demo",
      "account_number": "demo-account-001",
      "account_type": "savings",
      "currency_id": 1,
      "currency_code": "NIO",
      "beneficiary": "Titular Demo",
      "active": true
    }
  ]
}
```

## `bank_accounts_create`

Crea una cuenta bancaria. Requiere la compuerta de escritura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `bank` | string | Sí* | Nombre, `short_code` o ID como texto del banco |
| `bank_id` | integer | Sí* | ID del banco; si viene con `bank`, gana `bank_id` |
| `alias` | string | Sí | Nombre corto de la cuenta (máx. 100 caracteres) |
| `account_number` | string | Sí | Número de cuenta (máx. 50 caracteres) |
| `currency` | string | Sí* | Código o nombre de la moneda, o ID como texto |
| `currency_id` | integer | Sí* | ID de la moneda; si viene con `currency`, gana `currency_id` |
| `beneficiary` | string | Sí | Titular o beneficiario (máx. 150 caracteres) |
| `account_type` | string | No | `savings` (default) o `checking` |
| `notes` | string | No | Notas (máx. 500 caracteres) |
| `active` | boolean | No | Default `true` |

\* Es obligatorio `bank` **o** `bank_id`, y `currency` **o** `currency_id`.

### Reglas

- El banco y la moneda se resuelven antes del `POST /api/banks/{bank_id}/accounts`.
- Verifica duplicados por `bank_id` + `account_number`; si existe, responde `already_exists` y no escribe.
- Devuelve la cuenta creada con su `id`.

### Ejemplo

```json
{
  "bank": "Banco Demo",
  "alias": "Cuenta Demo",
  "account_number": "demo-account-001",
  "currency": "NIO",
  "beneficiary": "Titular Demo",
  "account_type": "savings",
  "active": true
}
```

### Salida

```json
{
  "id": 1,
  "bank_id": 1,
  "bank_name": "Banco Demo",
  "alias": "Cuenta Demo",
  "account_number": "demo-account-001",
  "account_type": "savings",
  "currency_id": 1,
  "currency_code": "NIO",
  "beneficiary": "Titular Demo",
  "active": true
}
```

## `bank_accounts_update`

Edita una cuenta con semántica parcial. Requiere la compuerta de escritura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `id` | integer | Sí | ID de la cuenta |
| `alias` | string | No | Nuevo alias |
| `account_number` | string | No | Nuevo número de cuenta |
| `currency` | string | No | Nueva moneda por código, nombre o ID |
| `currency_id` | integer | No | Nueva moneda por ID |
| `beneficiary` | string | No | Nuevo beneficiario |
| `account_type` | string | No | `savings` o `checking` |
| `notes` | string | No | Notas; `""` las limpia |
| `active` | boolean | No | Estado; si se omite, conserva el actual |
| `bank` / `bank_id` | — | — | Rechazados con `invalid_value`: el banco no se cambia |

Enviá solo los campos que cambian. El resto se conserva tal como estaba.

### Ejemplo

```json
{
  "id": 1,
  "alias": "Cuenta Demo Nómina",
  "active": false
}
```

### Salida

```json
{
  "id": 1,
  "bank_id": 1,
  "bank_name": "Banco Demo",
  "alias": "Cuenta Demo Nómina",
  "account_number": "demo-account-001",
  "account_type": "savings",
  "currency_id": 1,
  "currency_code": "NIO",
  "beneficiary": "Titular Demo",
  "active": false
}
```

## `credit_cards_search`

Busca tarjetas de crédito. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `query` | string | No | Texto libre sobre alias, `last4`, titular, marca y banco |
| `bank` | string | No | Nombre, `short_code` o ID como texto del banco |
| `bank_id` | integer | No | ID del banco; si viene con `bank`, gana `bank_id` |
| `active` | boolean | No | Filtra por estado; omitido devuelve activas e inactivas |
| `limit` | integer | No | Máximo de resultados, 1..50; default 10 |

### Ejemplo

```json
{
  "query": "demo",
  "bank": "Banco Demo",
  "active": true,
  "limit": 10
}
```

### Salida

```json
{
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 1,
      "bank_id": 1,
      "bank_name": "Banco Demo",
      "alias": "Tarjeta Demo",
      "network": "visa",
      "last4": "0000",
      "holder": "Titular Demo",
      "currency_id": 1,
      "currency_code": "NIO",
      "credit_limit": 1000,
      "credit_limit_usd": 0,
      "limit_currency": "NIO",
      "statement_day": 1,
      "payment_due_day": 1,
      "active": true
    }
  ]
}
```

## `credit_cards_create`

Crea una tarjeta de crédito. Requiere la compuerta de escritura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `bank` | string | Sí* | Nombre, `short_code` o ID como texto del banco |
| `bank_id` | integer | Sí* | ID del banco; si viene con `bank`, gana `bank_id` |
| `alias` | string | Sí | Nombre corto de la tarjeta (máx. 100 caracteres) |
| `network` | string | Sí | `visa`, `mastercard`, `amex`, `diners` u `other` |
| `last4` | string | Sí | Últimos cuatro dígitos; exactamente 4 números |
| `currency` | string | Sí* | Código o nombre de la moneda, o ID como texto |
| `currency_id` | integer | Sí* | ID de la moneda; si viene con `currency`, gana `currency_id` |
| `holder` | string | No | Titular (máx. 150 caracteres) |
| `statement_day` | integer | No | Día de corte, 1..31; default 1 |
| `payment_due_day` | integer | No | Día límite de pago, 1..31; default 1 |
| `min_payment_pct` | number | No | Porcentaje de pago mínimo, 0..100; default 4 |
| `limit_currency` | string | No | `NIO` (default) o `USD` |
| `credit_limit` | number | No | Límite de crédito en `limit_currency`; default 0 |
| `credit_limit_usd` | number | No | Límite de crédito en USD |
| `exchange_rate` | number | No | Tipo de cambio para derivar el límite en la otra moneda |
| `secondary_currency` | string | No | Moneda secundaria por código o ID; debe ser distinta de la principal |
| `secondary_currency_id` | integer | No | ID de la moneda secundaria (alternativa a `secondary_currency`) |
| `category` | string | No | `classic`, `gold`, `platinum`, `black`, `business` u `other`; vacío = no definida |
| `bonus_due_day` | integer | No | Día límite para bonificar, 0..31; `0` = sin día |
| `interest_current_rate` | number | No | Tasa de interés corriente, ≥ 0 |
| `interest_late_rate` | number | No | Tasa de mora, ≥ 0 |
| `interest_current_rate_usd` | number | No | Tasa corriente en USD, ≥ 0 |
| `interest_late_rate_usd` | number | No | Tasa de mora en USD, ≥ 0 |
| `color` | string | No | Color de la tarjeta en la paleta del tema; default `blue` |
| `icon` | string | No | Identificador de ícono; default `credit` |
| `active` | boolean | No | Default `true` |
| `notes` | string | No | Notas (máx. 500 caracteres) |

\* Es obligatorio `bank` **o** `bank_id`, y `currency` **o** `currency_id`.

### Defaults

| Campo | Default |
|---|---|
| `statement_day` | 1 |
| `payment_due_day` | 1 |
| `bonus_due_day` | 0 (sin día) |
| `min_payment_pct` | 4 |
| `limit_currency` | `NIO` |
| `color` | `blue` |
| `icon` | `credit` |
| `active` | `true` |

`category` queda vacía (no definida), `notes` y `holder` quedan vacíos y el límite de crédito queda en `0` si no se informa.

### Validaciones

- `statement_day` y `payment_due_day` entre 1 y 31.
- `bonus_due_day` entre 0 y 31 y distinto de `payment_due_day` cuando es mayor que 0.
- `min_payment_pct` entre 0 y 100; las tasas, el tipo de cambio y los límites no pueden ser negativos.
- La moneda secundaria debe existir y ser distinta de la principal.
- El backend repite estas validaciones y agrega las de longitud y paleta; un valor inválido vuelve como `invalid_value`.

### Ejemplo

```json
{
  "bank": "Banco Demo",
  "alias": "Tarjeta Demo",
  "network": "visa",
  "last4": "0000",
  "currency": "NIO",
  "holder": "Titular Demo",
  "statement_day": 15,
  "payment_due_day": 5,
  "credit_limit": 1000
}
```

### Salida

```json
{
  "id": 1,
  "bank_id": 1,
  "bank_name": "Banco Demo",
  "alias": "Tarjeta Demo",
  "network": "visa",
  "last4": "0000",
  "holder": "Titular Demo",
  "currency_id": 1,
  "currency_code": "NIO",
  "credit_limit": 1000,
  "credit_limit_usd": 0,
  "limit_currency": "NIO",
  "statement_day": 15,
  "payment_due_day": 5,
  "active": true
}
```

La respuesta resume la tarjeta con los mismos campos que la búsqueda; `secondary_currency_code` aparece solo si la tarjeta tiene moneda secundaria.

## `credit_cards_update`

Edita una tarjeta con semántica parcial. Requiere la compuerta de escritura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `id` | integer | Sí | ID de la tarjeta |
| `alias` | string | No | Nuevo alias |
| `network` | string | No | Nueva marca de la allowlist |
| `last4` | string | No | Nuevos últimos cuatro dígitos |
| `holder` | string | No | Nuevo titular; `""` lo limpia |
| `currency` | string | No | Nueva moneda principal por código, nombre o ID |
| `currency_id` | integer | No | Nueva moneda principal por ID |
| `secondary_currency` | string | No | Moneda secundaria; `""` la limpia |
| `secondary_currency_id` | integer | No | ID de la moneda secundaria; `0` la limpia |
| `category` | string | No | Categoría de la tarjeta; `""` la deja sin definir |
| `limit_currency` | string | No | `NIO` o `USD` |
| `exchange_rate` | number | No | Tipo de cambio |
| `credit_limit` | number | No | Límite en `limit_currency` |
| `credit_limit_usd` | number | No | Límite en USD |
| `statement_day` | integer | No | Día de corte, 1..31 |
| `payment_due_day` | integer | No | Día límite de pago, 1..31 |
| `bonus_due_day` | integer | No | Día límite para bonificar, 0..31; `0` lo limpia |
| `min_payment_pct` | number | No | Porcentaje de pago mínimo, 0..100 |
| `interest_current_rate` | number | No | Tasa corriente, ≥ 0 |
| `interest_late_rate` | number | No | Tasa de mora, ≥ 0 |
| `interest_current_rate_usd` | number | No | Tasa corriente en USD, ≥ 0 |
| `interest_late_rate_usd` | number | No | Tasa de mora en USD, ≥ 0 |
| `color` | string | No | Color de la tarjeta |
| `icon` | string | No | Identificador de ícono |
| `active` | boolean | No | Estado; si se omite, conserva el actual |
| `notes` | string | No | Notas; `""` las limpia |
| `bank` / `bank_id` | — | — | Rechazados con `invalid_value`: el banco no se cambia |

Enviá solo los campos que cambian. El resto se conserva tal como estaba.

### Ejemplo

```json
{
  "id": 1,
  "alias": "Tarjeta Demo Principal",
  "payment_due_day": 10,
  "secondary_currency": ""
}
```

### Salida

```json
{
  "id": 1,
  "bank_id": 1,
  "bank_name": "Banco Demo",
  "alias": "Tarjeta Demo Principal",
  "network": "visa",
  "last4": "0000",
  "holder": "Titular Demo",
  "currency_id": 1,
  "currency_code": "NIO",
  "credit_limit": 1000,
  "credit_limit_usd": 0,
  "limit_currency": "NIO",
  "statement_day": 15,
  "payment_due_day": 10,
  "active": true
}
```

## Errores comunes

| Código | Cuándo | `hint` |
|---|---|---|
| `writes_disabled` | Alta o edición con la compuerta apagada | Definí `P40LA_MCP_ALLOW_WRITES=true`; en modo HTTP también `P40LA_MCP_REMOTE_WRITES=true` |
| `not_found` | Banco, moneda o entidad inexistente | Lista de opciones o el `id` que no existe |
| `ambiguous_reference` | Varios bancos o monedas coinciden con la referencia | Enviá `bank_id` o el código exacto, con los candidatos |
| `already_exists` | Cuenta o tarjeta duplicada | Devuelve el `id` de la entidad existente |
| `invalid_value` | Campo obligatorio vacío, `limit` fuera de 1..50, allowlist o rango inválido, monto negativo, `id` inválido o intento de cambiar el banco | Mensaje del backend o de la tool |
| `upstream_error` | Fallo de red o respuesta 5xx | Verificá `P40LA_BASE_URL` y `GET /health` |
| `auth_failed` | API key inválida, revocada o rotada | Revisá la key y [api-key.md](../api-key.md) |

## Flujo recomendado para un agente

1. Buscá la entidad con `bank_accounts_search` o `credit_cards_search` para obtener el `id` o confirmar que no existe.
2. Si conocés el banco, pasalo como `bank` o `bank_id`: reduce las llamadas y evita ambigüedad.
3. Antes de una escritura, mostrá al usuario el banco y la moneda resueltos, la entidad o el cuerpo que se enviará y si existe un duplicado.
4. Pedí confirmación explícita y ejecutá con la compuerta activa.
5. Verificá con una búsqueda y reportá el `id` resultante.

Si un alta falla por red después de crear el registro, reintentar devuelve `already_exists` con el `id` existente: tratalo como el alta ya realizada y no insistas con variantes del nombre.
