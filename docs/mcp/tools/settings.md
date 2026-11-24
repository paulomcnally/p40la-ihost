---
title: "Tools de configuración"
sidebar_position: 13
description: "Referencia de settings_status"
---

# Tools de configuración

La tool es de **solo lectura** sobre el estado de configuración e integraciones del sistema. No modifica settings, no prueba SMTP ni voz, no rota keys y no depende de la compuerta de escritura.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `settings_status` | Lectura | 1 | Estado de configuración, integraciones y formato de moneda, sin secretos |

## `settings_status`

Devuelve el estado de configuración e integraciones con una **allowlist de campos**: horas de proceso, zona horaria, país, estado de SMTP, voz, Telegram, webhook y formato de moneda. Solo lectura. No recibe parámetros.

### Entrada

La tool no recibe campos: se llama con un objeto vacío.

### Ejemplo

```json
{}
```

### Salida

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
  "voicemonkey_send_alerts": false,
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

### Allowlist de campos

Solo se devuelven las claves del ejemplo. Quedan **excluidos**:

| Campo excluido | Motivo |
|---|---|
| `smtp_from_email`, `alert_emails` | Emails (`SEC-NO-PII`) |
| `smtp_user`, `smtp_password` | Credenciales |
| `telegram_bot_token`, `telegram_bot_chat_ids` | Secreto y chat IDs; solo se expone `telegram_bot_chat_ids_count` |
| `voicemonkey_token` | Secreto |
| `automation_base_url` | Configuración interna |
| `email_color_*` | Paleta de emails, sin valor operativo para el agente |

La allowlist se aplica en el servidor MCP, no en la UI: aunque la API devuelva más campos, la tool nunca los propaga.

### Reglas

- Las claves ausentes se omiten de la respuesta: si una integración no está configurada, sus campos pueden no aparecer.
- `smtp_configured`, `voicemonkey_configured`, `telegram_bot_configured` y `automation_configured` indican si la integración está lista para usarse.
- `telegram_bot_chat_ids_count` es la cantidad de chats configurados, no sus IDs.
- `currency_thousands_separator`, `currency_decimal_separator` y `currency_decimal_digits` describen el formato de moneda del sistema.
- No modifica nada: no hay parámetros de escritura.

### Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## Privacidad y datos ocultos

La tool nunca devuelve secretos, emails ni chat IDs (`SEC-NO-PII`). Si el usuario necesita ver o cambiar esos valores, la acción se hace en Configuración de la UI con sesión autenticada.

No inventes valores: si un campo no está en la respuesta, la integración correspondiente no está configurada o quedó fuera de la allowlist.

## Flujo recomendado para un agente

1. Llamar `settings_status` antes de diagnosticar notificaciones o programaciones.
2. Cruzar `*_configured` con los canales habilitados que devuelve `alerts_list`.
3. Reportar solo estado (`true`/`false`), horas y zona horaria; no intentes inferir credenciales ni emails.
