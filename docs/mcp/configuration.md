---
title: "Configuración"
sidebar_position: 3
description: "Variables de entorno y configuración de clientes MCP"
---

# Configuración

## Variables de entorno

### Comunes y modo local (stdio)

| Variable | Obligatoria | Default | Descripción |
|---|---|---|---|
| `P40LA_BASE_URL` | No | `http://ihost.local:8088` | URL base del servidor; admite un path (ej. detrás de proxy) |
| `P40LA_API_KEY` | Sí | — | API key global `p40la_...` (ver [api-key.md](api-key.md)) |
| `P40LA_MCP_TRANSPORT` | No | `stdio` | `stdio` o `http` |
| `P40LA_MCP_ALLOW_WRITES` | No | `false` | Compuerta principal de escritura: aplica a toda tool que escribe (`bills_*`, `autos_*`, `bank_accounts_*`, `credit_cards_*`, `homes_update` y las `*_create` de entidades) |
| `P40LA_TIMEOUT` | No | `15s` | Timeout HTTP por llamada (1s a 60s) |
| `P40LA_TIMEZONE` | No | `America/Managua` | Zona para “mes actual” y “hoy” |

### Solo modo HTTP

| Variable | Obligatoria | Default | Descripción |
|---|---|---|---|
| `P40LA_MCP_LISTEN` | No | `127.0.0.1:8090` | Listener público; debe ser loopback y distinto del admin |
| `P40LA_MCP_ADMIN_LISTEN` | No | `127.0.0.1:8091` | Listener administrativo (`/healthz`, revocación); loopback |
| `P40LA_MCP_PUBLIC_URL` | Sí | — | Origen público exacto, `https://demo-node.example.ts.net`, sin `/mcp` |
| `P40LA_MCP_OAUTH_PASSWORD_HASH` | Sí | — | Hash bcrypt generado con `p40la-mcp -hash-password` |
| `P40LA_MCP_OAUTH_SUBJECT` | No | `owner` | Identificador opaco del usuario en la auditoría |
| `P40LA_MCP_STATE_DIR` | No | `/data` | Directorio de `oauth.json` y `audit.jsonl` |
| `P40LA_MCP_REMOTE_WRITES` | No | `false` | Segunda compuerta para toda tool de escritura por HTTP |
| `P40LA_MCP_RATE_LIMIT` | No | `60` | Requests por minuto por token (1 a 600) |
| `P40LA_MCP_AUDIT_MAX_MB` | No | `5` | Tamaño de rotación de `audit.jsonl` (1 a 100) |

Las variables de la segunda tabla se ignoran en modo stdio. En modo HTTP, `P40LA_API_KEY` sigue siendo obligatoria: el binario la usa para llamar a la API del sistema.

### Validaciones del modo HTTP

El binario **no arranca** si:

- `P40LA_MCP_LISTEN` o `P40LA_MCP_ADMIN_LISTEN` no son loopback (`127.0.0.1` o `::1`) o son iguales.
- `P40LA_MCP_PUBLIC_URL` está vacía, tiene path/query/fragment o usa `http://` con un host que no es loopback.
- `P40LA_MCP_OAUTH_PASSWORD_HASH` está vacía o no empieza con `$2a$`, `$2b$` o `$2y$`.
- `P40LA_MCP_RATE_LIMIT` está fuera de 1–600 o `P40LA_MCP_AUDIT_MAX_MB` fuera de 1–100.
- `P40LA_BASE_URL` no es una URL absoluta `http`/`https`.
- `P40LA_API_KEY` no empieza con `p40la_`.

La URL pública es el **origen**, no el endpoint: `https://demo-node.example.ts.net` es correcto; `https://demo-node.example.ts.net/mcp` se rechaza.

El MCP **no** usa `P40LA_EMAIL`, `P40LA_PASSWORD` ni cookies. La contraseña del dashboard solo se usa una vez para crear o rotar la key.

## Provisión de la key

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" ./scripts/create-api-key.sh
export P40LA_API_KEY="p40la_..."
```

Detalle de creación, rotación y revocación en [api-key.md](api-key.md).

## Ejemplo de `.env` para modo HTTP

Archivo sintético; reemplazá los valores por los de tu instalación y dejalo con permisos `0600`. No lo commitees.

```dotenv
# API de p40la-ihost
P40LA_BASE_URL=http://ihost.local:8088
# Placeholder: la key real empieza con p40la_
P40LA_API_KEY=demo-api-key
P40LA_TIMEOUT=15s
P40LA_TIMEZONE=America/Managua

# Transporte
P40LA_MCP_TRANSPORT=http
P40LA_MCP_LISTEN=127.0.0.1:8090
P40LA_MCP_ADMIN_LISTEN=127.0.0.1:8091
P40LA_MCP_PUBLIC_URL=https://demo-node.example.ts.net

# OAuth (reemplazar por el hash real; el placeholder no es un bcrypt válido)
P40LA_MCP_OAUTH_PASSWORD_HASH=CAMBIAR_POR_HASH_BCRYPT
P40LA_MCP_OAUTH_SUBJECT=owner

# Estado y límites
P40LA_MCP_STATE_DIR=/data
P40LA_MCP_RATE_LIMIT=60
P40LA_MCP_AUDIT_MAX_MB=5

# Escrituras: ambas deben ser true para escribir por HTTP
P40LA_MCP_ALLOW_WRITES=false
P40LA_MCP_REMOTE_WRITES=false
```

Generación del hash:

```bash
read -rs -p "Contraseña: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | ./bin/p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD
printf 'P40LA_MCP_OAUTH_PASSWORD_HASH=%s\n' "$HASH"
```

`docker --env-file` lee el archivo tal cual; si lo abrís en una shell, los hashes bcrypt contienen `$` y conviene citarlos con comillas simples. Nunca guardes la contraseña en claro en el archivo: solo el hash.

## Modo remoto

Para conectores remotos, el binario se arranca con `P40LA_MCP_TRANSPORT=http` y se publica con Tailscale Funnel. La configuración necesita, además de la API key:

- una contraseña de administración hasheada (`P40LA_MCP_OAUTH_PASSWORD_HASH`),
- la URL pública exacta (`P40LA_MCP_PUBLIC_URL`),
- un directorio de estado escribible (`P40LA_MCP_STATE_DIR`).

El flujo completo, el formato del hash y los límites están en [http.md](http.md) y [oauth.md](oauth.md). La publicación en 443 se detalla en [funnel.md](funnel.md).

### Despliegue en el iHost

Cuando el MCP corre 24/7 en el iHost, se usa un **nodo Tailscale dedicado** para no tocar el nodo que publica el dashboard. Los valores cambian respecto de una laptop:

| Variable | Valor en el iHost | Por qué |
|---|---|---|
| `P40LA_BASE_URL` | `http://127.0.0.1:8088` | El MCP comparte la red host con la app; no depende de MagicDNS ni del FQDN del dashboard |
| `P40LA_MCP_PUBLIC_URL` | `https://p40la-mcp.example.ts.net` | Hostname del nodo dedicado, sin `/mcp`; debe coincidir con el FQDN de Funnel |
| `P40LA_MCP_STATE_DIR` | `/data` | Volumen `p40la-mcp-data`, escribible por UID 65532 |
| `P40LA_MCP_ALLOW_WRITES` | `true` solo con aprobación | Primera compuerta de escritura |
| `P40LA_MCP_REMOTE_WRITES` | `true` solo con aprobación | Segunda compuerta para el transporte HTTP |

Imágenes usadas en el iHost:

- `paulomcnally/p40la-mcp:0.5.0` — servidor MCP, red host, estado en `/data`.
- `paulomcnally/p40la-mcp-tailscale:latest` — nodo Tailscale dedicado, basado en `tailscale/tailscale`, con `serve.json` incluido y estado en `/var/lib/tailscale`.

No uses `ihost.example.ts.net` como `P40LA_MCP_PUBLIC_URL`: ese es el nodo del dashboard y **no se modifica**. El runbook completo está en [ihost.md](ihost.md).

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
        "P40LA_API_KEY": "demo-api-key",
        "P40LA_MCP_ALLOW_WRITES": "false"
      }
    }
  }
}
```

Reemplazá el valor por tu key real solo en tu configuración local (nunca en el repo).

## dry_run y compuerta de escritura

`P40LA_MCP_ALLOW_WRITES` es la compuerta principal y aplica a **toda** tool de escritura: `bills_*`, `autos_*`, `bank_accounts_*`, `credit_cards_*`, `homes_update` y las `*_create` de entidades. En modo HTTP se suma `P40LA_MCP_REMOTE_WRITES=true`; **ambas** deben estar encendidas.

`dry_run` es propio de facturas y autos: `bills_generate`, `bills_pay`, `autos_create`, `autos_update`, `autos_maintenance_register`, `autos_inspection_register` y `autos_insurance_register`.

- Vienen con `dry_run=true` por defecto en el MCP.
- Un `dry_run=true` funciona aunque las escrituras estén deshabilitadas.
- En **stdio**, para ejecutar cambios se necesitan **dos** condiciones: `dry_run=false` en la llamada y `P40LA_MCP_ALLOW_WRITES=true` en el entorno.
- En **HTTP**, se agrega una tercera: `P40LA_MCP_REMOTE_WRITES=true`.

Las tools de banca (`bank_accounts_create`, `bank_accounts_update`, `credit_cards_create` y `credit_cards_update`) **no tienen `dry_run`**: con la compuerta activa escriben directamente. Antes de llamarlas, la tool resuelve el banco y la moneda y verifica duplicados; el agente debe mostrar el resumen al usuario y pedir confirmación explícita antes de ejecutar.

Las altas de entidades y `homes_update` tampoco tienen `dry_run`: con la compuerta activa escriben directamente. El agente muestra la entidad y el cuerpo del alta, o el hogar resuelto y los campos que cambian, y pide confirmación explícita antes de ejecutar.

Sin la compuerta, `bills_generate` y `bills_pay` devuelven:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true o repetí con dry_run=true para previsualizar."
}
```

Las tools de banca y las de alta de entidades devuelven el mismo error con un `hint` sin `dry_run`:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true para ejecutar cambios."
}
```

En modo HTTP ambos `hint` mencionan las dos variables (`P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true`).

## Zona horaria

`P40LA_TIMEZONE` define el “mes actual” cuando `bills_generate` no recibe `to`, la fecha por defecto de `bills_pay` y el “hoy” con el que se valida que `performed_at` de mantenimientos e inspecciones no sea futura. El servidor usa su propia zona para validar; para evitar ambigüedad, el MCP envía `to` y `paid_at` explícitos calculados en esta zona y valida las fechas de autos con el mismo criterio.

## Verificación

```bash
curl -s http://ihost.local:8088/health
# {"status":"ok","timestamp":"..."}

# Modo HTTP: health del listener administrativo
curl -s http://127.0.0.1:8091/healthz
# {"status":"ok","version":"0.1.0","uptime_s":42,"writes_local":false,"writes_remote":false}
```

Si `/health` responde pero las tools fallan con `auth_failed`, revisá que `P40LA_API_KEY` sea la key vigente (no una rotada o revocada). Si `/mcp` responde `401 invalid_token`, el access token OAuth venció o fue revocado: ver [oauth.md](oauth.md#troubleshooting).
