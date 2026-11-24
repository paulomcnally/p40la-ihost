---
title: "Tools de alertas"
sidebar_position: 12
description: "Referencia de alerts_list"
---

# Tools de alertas

Tool de **solo lectura** sobre el catálogo de alertas del sistema (SPEC-154). No habilita canales, no envía notificaciones y no dispara schedulers.

## `alerts_list`

Sin parámetros.

```json
{
  "count": 2,
  "alerts": [
    { "key": "bills_due", "title": "Facturas por vencer", "mail_enabled": true, "voice_enabled": false, "telegram_enabled": true },
    { "key": "debts_due", "title": "Cuotas por vencer", "mail_enabled": true, "voice_enabled": true, "telegram_enabled": false }
  ]
}
```

## Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida o ausente |
| `upstream_error` | La API no responde |

## Privacidad

Se omite el campo `speech` porque contiene un nombre personal hardcodeado en la base (`SEC-NO-PII`). Si necesitás el texto de voz, consultalo desde Configuración → Alertas en la UI.
