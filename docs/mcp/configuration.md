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
| `P40LA_API_KEY` | Sí | — | API key global `p40la_...` (ver [api-key.md](api-key.md)) |
| `P40LA_MCP_ALLOW_WRITES` | No | `false` | Habilita `bills_generate` y `bills_pay` con `dry_run=false` |
| `P40LA_TIMEOUT` | No | `15s` | Timeout HTTP por llamada (1s a 60s) |
| `P40LA_TIMEZONE` | No | `America/Managua` | Zona para “mes actual” y “hoy” |

El MCP **no** usa `P40LA_EMAIL`, `P40LA_PASSWORD` ni cookies. La contraseña solo se usa una vez para crear o rotar la key.

## Provisión de la key

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" ./scripts/create-api-key.sh
export P40LA_API_KEY="p40la_..."
```

Detalle de creación, rotación y revocación en [api-key.md](api-key.md).

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
        "P40LA_API_KEY": "{env:P40LA_API_KEY}",
        "P40LA_MCP_ALLOW_WRITES": "false",
        "P40LA_TIMEZONE": "America/Managua"
      },
      "enabled": true
    }
  }
}
```

Si tu versión de OpenCode no interpola `{env:VAR}`, exportá `P40LA_API_KEY` en la shell que inicia el cliente o usá un archivo local fuera del repo.

## Claude Desktop

Ejemplo sintético (`docs/mcp/examples/claude-desktop.json`):

```json
{
  "mcpServers": {
    "p40la": {
      "command": "/home/demo/bin/p40la-mcp",
      "env": {
        "P40LA_BASE_URL": "http://ihost.local:8088",
        "P40LA_API_KEY": "p40la_demo-demo-demo-demo-demo-demo-demo",
        "P40LA_MCP_ALLOW_WRITES": "false"
      }
    }
  }
}
```

Reemplazá el valor por tu key real solo en tu configuración local (nunca en el repo).

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

Si `/health` responde pero las tools fallan con `auth_failed`, revisá que `P40LA_API_KEY` sea la key vigente (no una rotada o revocada).
