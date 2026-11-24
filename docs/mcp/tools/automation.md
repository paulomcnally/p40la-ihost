---
title: "Tools de automatizaciones"
sidebar_position: 11
description: "Referencia de automation_accounts_list y automation_runs_list"
---

# Tools de automatizaciones

Las dos tools son de **solo lectura** sobre las cuentas de automatización (`automation`) y su historial. No crean cuentas, no ejecutan `sync`, no consultan portales externos, no reenvían webhooks y no revelan credenciales ni URLs de webhook.

No dependen de la compuerta de escritura.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `automation_accounts_list` | Lectura | 1 | Cuentas con institución, plugin, programación y última ejecución |
| `automation_runs_list` | Lectura | 2 | Facturas consultadas y logs de entrega de una cuenta |

## `automation_accounts_list`

Lista las cuentas de automatización con su institución, plugin, programación y última ejecución. Solo lectura. No recibe parámetros.

### Entrada

La tool no recibe campos: se llama con un objeto vacío.

### Ejemplo

```json
{}
```

### Salida

```json
{
  "count": 1,
  "accounts": [
    {
      "id": 1,
      "institution_id": 2,
      "institution_name": "Proveedor Demo",
      "label": "Cuenta demo",
      "plugin_name": "demo.plugin",
      "schedule_enabled": true,
      "schedule_frequency": "daily",
      "schedule_time": "08:00",
      "last_run_at": "2026-11-20T08:00:00Z",
      "created_at": "2026-10-01T12:00:00Z",
      "updated_at": "2026-11-20T08:00:00Z"
    }
  ]
}
```

### Reglas

- `schedule_frequency` es `daily`, `every_n_days`, `weekly` o `monthly`. Con `every_n_days` se usa `schedule_interval`; con `weekly` y `monthly`, `schedule_days`.
- `last_run_at` aparece solo si la cuenta se ejecutó al menos una vez.
- `plugin_name`, `schedule_time`, `schedule_interval` y `schedule_days` se omiten cuando no están definidos.
- `count` es la cantidad de cuentas devueltas.
- La tool elimina `identifier` (identificador de la cuenta en el portal) y `webhook_url` de cada cuenta.
- La respuesta de la API nunca incluye credenciales: el campo `credentials` no se serializa.

### Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## `automation_runs_list`

Devuelve el historial de una cuenta: las facturas consultadas por el plugin y los logs de entrega al webhook. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `account_id` | integer | Sí | ID de la cuenta; debe ser mayor que `0` |
| `limit` | integer | No | Máximo de logs de webhook, `1..200`; default `50` |

`limit` aplica solo a `webhook_logs`; las facturas consultadas se devuelven completas.

### Ejemplo

```json
{
  "account_id": 1,
  "limit": 10
}
```

### Salida

```json
{
  "account_id": 1,
  "bills_count": 1,
  "bills": [
    {
      "id": 5,
      "account_id": 1,
      "plugin_name": "demo.plugin",
      "plugin_version": "1.0.0",
      "source": "portal",
      "status": "ok",
      "fetched_at": "2026-11-20T08:00:30Z",
      "created_at": "2026-11-20T08:00:30Z"
    }
  ],
  "logs_count": 1,
  "webhook_logs": [
    {
      "id": 7,
      "account_id": 1,
      "status": "ok",
      "response": "upsert en proceso",
      "ran_at": "2026-11-20T08:01:00Z"
    }
  ]
}
```

### Reglas

- `bills[].status` es `ok` o `error`. Cuando falla, `error` trae el mensaje y `error_code` clasifica el fallo (`portal_migrated`, `captcha_required`, `auth_expired`).
- `bills[].source` indica el origen del plugin y `plugin_version` su versión.
- `webhook_logs[].status` es `ok` o `error`; `response` es el texto devuelto por el destino y `http_status` aparece cuando la API lo registró.
- La tool elimina `identifier`, `webhook_url`, `payload`, `raw`, `request_body`, `response_body`, `credentials` y `webhook_uuid` de cada objeto.
- Una cuenta inexistente devuelve `not_found`: la API responde `404` en la primera llamada.
- `limit` fuera de `1..200` devuelve `invalid_request` sin llamar a la API.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `account_id` es `0` o negativo, o `limit` está fuera de `1..200` |
| `not_found` | La cuenta de automatización no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## Privacidad y datos ocultos

Las tools aplican una lista de omisión (`SEC-NO-PII`) sobre cada objeto:

- `identifier`: identificador de la cuenta en el portal del proveedor.
- `webhook_url`, `webhook_uuid`: URLs y UUID de webhook (secretos).
- `payload`, `raw`, `request_body`, `response_body`: cuerpos crudos de las consultas y entregas.
- `credentials`: credenciales del portal (la API tampoco las serializa).

El campo `response` de los logs sí se devuelve: es un texto de estado del webhook, no un payload. No lo confundas con datos de la factura.

## Flujo recomendado para un agente

1. Llamar `automation_accounts_list` para ubicar la cuenta y ver si su programación está activa.
2. Llamar `automation_runs_list` con el `id` para revisar la última ejecución: facturas consultadas y resultado de la entrega.
3. Si `status` es `error`, reportar `error_code` y el mensaje, y proponer revisar la cuenta desde Automatizaciones en la UI.
4. No intentar reejecutar el `sync` ni reenviar el webhook desde el MCP: estas tools no escriben.
