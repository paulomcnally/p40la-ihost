---
title: "Tools de configuración"
sidebar_position: 13
description: "Referencia de settings_status"
---

# Tools de configuración

Tool de **solo lectura** sobre el estado de configuración e integraciones (SPEC-154). No modifica settings, no prueba SMTP ni voz, y no rota keys.

## `settings_status`

Sin parámetros.

```json
{
  "billing_generation_hour": 6,
  "alert_check_hour": 8,
  "timezone": "America/Managua",
  "country_code": "NI",
  "smtp_host": "smtp.example.com",
  "smtp_port": 587,
  "smtp_from_name": "p40la-ihost",
  "smtp_configured": true,
  "email_alerts_enabled": true,
  "voicemonkey_enabled": false,
  "voicemonkey_configured": false,
  "webhook_enabled": true,
  "webhook_base_url": "https://example.com",
  "automation_configured": false,
  "currency_thousands_separator": ",",
  "currency_decimal_separator": ".",
  "currency_decimal_digits": 2,
  "telegram_bot_enabled": true,
  "telegram_bot_configured": true,
  "telegram_bot_chat_ids_count": 1,
  "telegram_bot_separator_length": 10,
  "telegram_bot_show_months": 3
}
```

## Allowlist de campos

Solo se devuelven las claves de la tabla anterior. Quedan **excluidos**:

- `smtp_from_email` y `alert_emails` (emails).
- `telegram_bot_chat_ids` (solo se expone el conteo).
- `smtp_password`, `telegram_bot_token`, `voicemonkey_token`, `automation_api_key` y cualquier secreto.
- `automation_base_url` (configuración legacy).

## Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida o ausente |
| `upstream_error` | La API no responde |

## Privacidad

La allowlist se aplica en el servidor MCP, no en la UI: aunque la API devuelva más campos, la tool nunca los propaga (`SEC-NO-PII`).
