---
title: "Configuración"
sidebar_position: 3
description: "Variables de entorno y configuración de clientes MCP"
---

# Configuración

## Variables de entorno

| Variable | Obligatoria | Default | Descripción |
|---|---|---|---|
| `P40LA_BASE_URL` | No | `http://ihost.local:8088` | URL base del servidor; admite un path (ej. detrás de proxy) |
| `P40LA_EMAIL` | Sí* | — | Email del usuario |
| `P40LA_PASSWORD` | Sí* | — | Contraseña del usuario |
| `P40LA_SESSION_COOKIE` | Alternativa | — | Cookie `session` ya emitida; tiene prioridad sobre email/clave |
| `P40LA_MCP_ALLOW_WRITES` | No | `false` | Habilita `bills_generate` y `bills_pay` con `dry_run=false` |
| `P40LA_TIMEOUT` | No | `15s` | Timeout HTTP por llamada (1s a 60s) |
| `P40LA_TIMEZONE` | No | `America/Managua` | Zona para “mes actual” y “hoy” |

\* Se requiere email y contraseña, o bien la cookie de sesión.

## Seguridad de las credenciales

- Usá variables de entorno, un archivo `0600` cargado por el cliente MCP o un gestor de secretos.
- **Nunca** pases la contraseña por argumentos de línea de comandos: queda visible en el listado de procesos.
- El MCP no escribe la cookie en disco; vive en memoria mientras el proceso está vivo.
- No commitees archivos `.env` con credenciales reales.

## OpenCode

Ejemplo sintético (`docs/mcp/examples/opencode.json`):

```json
{
  "mcp": {
    "p40la": {
      "type": "local",
      "command": ["/home/demo/bin/p40la-mcp"],
      "environment": {
        "P40LA_BASE_URL": "http://ihost.local:8088",
        "P40LA_EMAIL": "demo@example.com",
        "P40LA_PASSWORD": "{env:P40LA_PASSWORD}",
        "P40LA_MCP_ALLOW_WRITES": "false",
        "P40LA_TIMEZONE": "America/Managua"
      },
      "enabled": true
    }
  }
}
```

Si tu versión de OpenCode no interpola `{env:VAR}`, exportá la variable en la shell que inicia el cliente y poné el valor literal solo en tu configuración local (nunca en el repo).

## Claude Desktop

Ejemplo sintético (`docs/mcp/examples/claude-desktop.json`):

```json
{
  "mcpServers": {
    "p40la": {
      "command": "/home/demo/bin/p40la-mcp",
      "env": {
        "P40LA_BASE_URL": "http://ihost.local:8088",
        "P40LA_EMAIL": "demo@example.com",
        "P40LA_PASSWORD": "demo-password",
        "P40LA_MCP_ALLOW_WRITES": "false"
      }
    }
  }
}
```

## dry_run y compuerta de escritura

- `bills_generate` y `bills_pay` tienen `dry_run=true` por defecto en el MCP.
- Un `dry_run=true` funciona aunque las escrituras estén deshabilitadas.
- Para ejecutar cambios se necesitan **dos** condiciones: `dry_run=false` en la llamada y `P40LA_MCP_ALLOW_WRITES=true` en el entorno.
- Sin la compuerta, la tool devuelve:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true o repetí con dry_run=true para previsualizar."
}
```

## Zona horaria

`P40LA_TIMEZONE` define el “mes actual” cuando `bills_generate` no recibe `to` y la fecha por defecto de `bills_pay`. El servidor usa su propia zona para validar; para evitar ambigüedad, el MCP siempre envía `to` y `paid_at` explícitos calculados en esta zona.

## Verificación

```bash
curl -s http://ihost.local:8088/health
# {"status":"ok","timestamp":"..."}
```

Si `/health` responde pero las tools fallan con `auth_failed`, revisá las credenciales o la cookie.
