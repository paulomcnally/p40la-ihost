---
title: "Tools de alertas"
sidebar_position: 12
description: "Referencia de alerts_list"
---

# Tools de alertas

La tool es de **solo lectura** sobre el catálogo de alertas del sistema. No habilita ni deshabilita canales, no envía notificaciones, no dispara schedulers y no depende de la compuerta de escritura.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `alerts_list` | Lectura | 1 | Lista las alertas con los canales habilitados de cada una |

## `alerts_list`

Lista las alertas del sistema con sus canales habilitados (email, voz y Telegram). Solo lectura. No recibe parámetros.

### Entrada

La tool no recibe campos: se llama con un objeto vacío.

### Ejemplo

```json
{}
```

### Salida

```json
{
  "count": 2,
  "alerts": [
    {
      "id": 3,
      "key": "bill_summary",
      "title": "Resumen diario de facturas",
      "description": "Resumen diario de todas las facturas pendientes.",
      "mail_enabled": true,
      "voice_enabled": false,
      "telegram_enabled": true
    },
    {
      "id": 4,
      "key": "debt_due",
      "title": "Cuotas de deudas que vencen hoy",
      "description": "Email diario que agrupa todas las cuotas de deudas que vencen hoy, con el total del día.",
      "mail_enabled": true,
      "voice_enabled": true,
      "telegram_enabled": false
    }
  ]
}
```

### Claves estables

| `key` | Alerta |
|---|---|
| `insurance` | Seguros de autos vencidos |
| `bill_created` | Nueva factura generada |
| `bill_summary` | Resumen diario de facturas |
| `debt_due` | Cuotas de deudas que vencen hoy |
| `pension_records_created` | Pensión: registros del mes creados |
| `pension_record_paid` | Pensión: pago de manutención registrado |
| `pension_salary_received` | Pensión: salario recibido |
| `pension_record_rejected` | Pensión: registro rechazado |
| `pension_month_closing` | Pensión: cierre de mes |

Las claves son estables y las usa el scheduler; el `title` y la `description` pueden cambiar.

### Reglas

- `mail_enabled`, `voice_enabled` y `telegram_enabled` indican si el canal está habilitado para esa alerta.
- Que un canal esté habilitado **no** garantiza que esté configurado: el estado de SMTP, voz y Telegram se consulta con `settings_status`.
- `count` es la cantidad de alertas del catálogo.
- Cada alerta incluye también `id`, `created_at` y `updated_at`; no aportan al diagnóstico.
- No modifica flags ni envía alertas: para editarlas o probarlas se usa Configuración → Alertas en la UI.

### Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## Privacidad y datos ocultos

La tool omite el campo `speech` (`SEC-NO-PII`): es el texto que se envía al canal de voz y contiene un nombre personal fijo en la base de datos. El texto de voz no se devuelve por ninguna vía del MCP; se consulta en Configuración → Alertas de la UI.

## Flujo recomendado para un agente

1. Llamar `alerts_list` para saber qué alertas existen y qué canales tienen habilitados.
2. Si el usuario pregunta si las notificaciones van a llegar, combinar con `settings_status` (`smtp_configured`, `voicemonkey_configured`, `telegram_bot_configured`).
3. Aclarar que esta tool no habilita canales ni envía pruebas: esas acciones se hacen desde la UI.
