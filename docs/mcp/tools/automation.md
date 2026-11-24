---
title: "Tools de automatizaciones"
sidebar_position: 11
description: "Referencia de automation_accounts_list y automation_runs_list"
---

# Tools de automatizaciones

Tools de **solo lectura** sobre cuentas de automatización y su historial (SPEC-154). No ejecutan `sync`, no consultan portales externos y no revelan credenciales.

## `automation_accounts_list`

Sin parámetros.

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
      "last_run_at": "2026-11-20T08:00:00Z"
    }
  ]
}
```

Omite `identifier` (número de cuenta) y `webhook_url`.

## `automation_runs_list`

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `account_id` | integer | Sí | ID de la cuenta |
| `limit` | integer | No | Máximo de logs (1–200, default 50) |

```json
{ "account_id": 1, "limit": 10 }
```

```json
{
  "account_id": 1,
  "bills_count": 1,
  "bills": [{ "id": 5, "period": "2026-11", "status": "ok" }],
  "logs_count": 1,
  "webhook_logs": [{ "id": 7, "status_code": 200, "created_at": "2026-11-20T08:01:00Z" }]
}
```

Omite payloads crudos, cuerpos de request/response y URLs de webhook.

## Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `account_id` ≤ 0 o `limit` fuera de 1–200 |
| `not_found` | La cuenta no existe |
| `auth_failed` | API key inválida o ausente |

## Privacidad

`identifier` y `webhook_url` son datos sensibles (`SEC-NO-PII`). Nunca se devuelven. Tampoco se exponen credenciales: la API key del MCP no puede revelarlas.
