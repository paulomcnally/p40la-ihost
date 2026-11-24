---
title: "Servidor HTTP remoto"
sidebar_position: 8
description: "Modo Streamable HTTP del MCP, listeners, configuración y límites"
---

# Servidor HTTP remoto

`p40la-mcp` tiene dos transportes y el mismo catálogo de tools en ambos:

| Transporte | Cómo se activa | Cliente típico | Exposición |
|---|---|---|---|
| `stdio` (default) | `P40LA_MCP_TRANSPORT` vacío o `stdio` | OpenCode, Claude Code y Claude Desktop local | Ninguna: el cliente lanza el proceso |
| `http` | `P40LA_MCP_TRANSPORT=http` | claude.ai, Claude móvil, Cowork y clientes con Streamable HTTP | HTTPS 443 vía Tailscale Funnel + OAuth 2.1 |

El modo HTTP no reemplaza al modo stdio ni cambia las tools: solo agrega un listener HTTP con autenticación. El proceso sigue siendo un cliente de la API de p40la-ihost y **no** toca SQLite.

## Cuándo usar cada modo

- **stdio** si el binario corre en la misma máquina que el cliente. Es el caso de Claude Desktop y Claude Code locales: el cliente inicia el proceso y habla JSON-RPC por stdin/stdout. No requiere OAuth, URL pública ni Funnel.
- **http** si el cliente corre en otra máquina o en la nube de Anthropic. claude.ai, las apps móviles y Cowork ejecutan el conector desde su infraestructura: necesitan un endpoint HTTPS público en el puerto 443 y autenticación OAuth 2.1. Una IP de Tailscale (rango CGNAT `100.64/10`) no sirve porque Anthropic rechaza rangos CGNAT y direcciones privadas; por eso se publica con Funnel.

> Claude Desktop local que lanza el binario en tu máquina sigue usando **stdio**. El modo HTTP es para conectores remotos, no para “apuntar Desktop a la URL de Funnel” sin más.

## Arranque local

El modo HTTP exige una contraseña de administración hasheada con bcrypt y una URL pública. Para pruebas locales se permite `http://` únicamente si el host es loopback.

```bash
# 1. Compilar
go build -o bin/p40la-mcp ./cmd/mcp-server

# 2. Generar el hash bcrypt (la contraseña no queda en el historial)
read -rs -p "Contraseña: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | ./bin/p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD

# 3. Arrancar en modo HTTP
mkdir -p /tmp/p40la-mcp-state && chmod 700 /tmp/p40la-mcp-state

P40LA_BASE_URL="http://127.0.0.1:8088" \
P40LA_API_KEY="demo-api-key" \
P40LA_MCP_TRANSPORT=http \
P40LA_MCP_PUBLIC_URL="http://127.0.0.1:8090" \
P40LA_MCP_OAUTH_PASSWORD_HASH="$HASH" \
P40LA_MCP_STATE_DIR="/tmp/p40la-mcp-state" \
./bin/p40la-mcp
```

En producción `P40LA_MCP_PUBLIC_URL` debe ser `https://demo-node.example.ts.net` (sin `/mcp`, sin query ni fragment). El binario rechaza HTTP público y cualquier host que no sea loopback para `http://`.

Los logs van a **stderr**; stdout queda reservado para el protocolo en modo stdio.

## Variables del modo HTTP

| Variable | Default | Descripción |
|---|---|---|
| `P40LA_MCP_TRANSPORT` | `stdio` | `stdio` o `http` |
| `P40LA_MCP_LISTEN` | `127.0.0.1:8090` | Listener público; debe ser loopback y distinto del admin |
| `P40LA_MCP_ADMIN_LISTEN` | `127.0.0.1:8091` | Health y revocación; solo loopback, nunca en Funnel |
| `P40LA_MCP_PUBLIC_URL` | — | Origen público exacto, obligatorio en HTTP (`https://demo-node.example.ts.net`) |
| `P40LA_MCP_OAUTH_PASSWORD_HASH` | — | Hash bcrypt de la contraseña de administración (obligatorio) |
| `P40LA_MCP_OAUTH_SUBJECT` | `owner` | Identificador opaco del usuario en la auditoría |
| `P40LA_MCP_STATE_DIR` | `/data` | Directorio de `oauth.json` y `audit.jsonl` |
| `P40LA_MCP_REMOTE_WRITES` | `false` | Segunda compuerta para escrituras por HTTP |
| `P40LA_MCP_RATE_LIMIT` | `60` | Requests por minuto por token (1 a 600) |
| `P40LA_MCP_AUDIT_MAX_MB` | `5` | Tamaño de rotación de la auditoría (1 a 100) |

Las variables de API (`P40LA_BASE_URL`, `P40LA_API_KEY`, `P40LA_TIMEOUT`, `P40LA_TIMEZONE`) y `P40LA_MCP_ALLOW_WRITES` siguen siendo necesarias. Detalle completo en [configuration.md](configuration.md).

## Endpoints

### Listener público (`P40LA_MCP_LISTEN`)

| Método y ruta | Autenticación | Función |
|---|---|---|
| `POST /mcp` (y `GET`/`DELETE` del Streamable HTTP) | `Authorization: Bearer <access token>` | Streamable HTTP (JSON-RPC MCP, stream SSE y cierre de sesión) |
| `GET /.well-known/oauth-protected-resource` | Ninguna | Descubrimiento del recurso (RFC 9728) |
| `GET /.well-known/oauth-protected-resource/mcp` | Ninguna | Igual que la anterior, con path del recurso |
| `GET /.well-known/oauth-authorization-server` | Ninguna | Metadatos del servidor de autorización (RFC 8414) |
| `POST /register` | Ninguna, con rate limit | Registro dinámico de clientes (RFC 7591) |
| `GET /authorize` | Formulario de contraseña | Inicio del flujo OAuth con PKCE |
| `POST /authorize` | Contraseña bcrypt | Consentimiento y emisión del código |
| `POST /token` | Código + PKCE o refresh token | Emisión y rotación de tokens |

El resto de las rutas responde `404`. No hay CORS: los clientes remotos no llaman a `/mcp` desde un navegador; el flujo OAuth usa redirects del lado del servidor.

### Listener administrativo (`P40LA_MCP_ADMIN_LISTEN`)

| Método y ruta | Autenticación | Función |
|---|---|---|
| `GET /healthz` | Solo loopback | Estado del proceso |
| `POST /admin/revoke` | Solo loopback | Revoca todos los tokens y códigos OAuth |

`/healthz` y la revocación **no** se publican en Funnel. Son operaciones locales del nodo.

## Comportamiento HTTP

- **Sin token o token inválido** en `/mcp`: `401` con
  `WWW-Authenticate: Bearer resource_metadata="<PUBLIC_URL>/.well-known/oauth-protected-resource"`.
- **Rate limit**: `429` con `Retry-After` en segundos. La clave es el hash del bearer token.
- **Body máximo**: 1 MiB. Un `Content-Length` mayor responde `413` sin leer el cuerpo.
- **Headers de seguridad**: `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `X-Frame-Options: DENY`; HSTS solo cuando la URL pública es `https`.
- **Timeouts del listener público**: 5 s para headers, 30 s de lectura, 120 s idle, 64 KiB de headers.
- **Apagado ordenado**: `SIGINT`/`SIGTERM` cierran ambos listeners y la auditoría en un máximo de 5 s.
- **Sin compresión ni listado de directorios**: los archivos estáticos no se sirven desde este proceso.

## Arquitectura de listeners

```
Internet
   │ HTTPS 443 (certificado de Tailscale)
   ▼
Tailscale Funnel ──► 127.0.0.1:8090  (listener público)
                        ├── /mcp
                        └── /.well-known/*, /register, /authorize, /token

127.0.0.1:8091  (listener administrativo, solo loopback)
   ├── GET /healthz
   └── POST /admin/revoke
```

Los dos listeners son procesos `net/http` separados dentro del mismo binario:

- El **público** queda detrás de Funnel y exige OAuth en `/mcp`.
- El **administrativo** nunca debe publicarse: su única protección es el bind a loopback y la verificación de `RemoteAddr`.
- `P40LA_MCP_LISTEN` y `P40LA_MCP_ADMIN_LISTEN` deben ser distintos; el binario no arranca si se repiten o si alguno no es loopback.

## Health check

```bash
curl -s http://127.0.0.1:8091/healthz
# {"status":"ok","version":"0.1.0","uptime_s":42,"writes_local":false,"writes_remote":false}
```

El health no expone tokens, contadores por token ni datos de negocio. Si el proceso está en un contenedor con `network_mode: host`, el puerto 8091 responde en el loopback del nodo.

## Revocación

```bash
curl -s -X POST http://127.0.0.1:8091/admin/revoke
# {"status":"revoked"}
```

Invalida todos los access tokens, refresh tokens y códigos pendientes; los clientes registrados se conservan para que el cliente pueda reintentar la autorización. Detalle en [oauth.md](oauth.md).

## Docker

`Dockerfile.mcp` construye una imagen independiente del add-on web, con usuario `nonroot` (UID/GID 65532) y estado en `/data`:

```bash
GH_TOKEN="github_pat_demo" ./scripts/build-mcp-image.sh
# Imagen multi-arch: linux/amd64, linux/arm/v7, linux/arm64
```

```bash
docker run -d --name p40la-mcp \
  --restart unless-stopped \
  --network host \
  -v /opt/p40la-mcp/data:/data \
  --env-file /etc/p40la-mcp.env \
  paulomcnally/p40la-mcp:0.1.0
```

- `--network host` permite que Tailscale (en el host) alcance `127.0.0.1:8090` y deja el listener administrativo solo en loopback.
- El directorio `/data` debe pertenecer al UID 65532 y tener permisos `0700`.
- La imagen no incluye shell ni utilidades: el health check se ejecuta desde el host con `curl`.

El build y la rotación de la auditoría se detallan en [operations.md](operations.md); el despliegue 24/7 en el iHost, en [ihost.md](ihost.md).

## Límites

| Límite | Valor |
|---|---|
| Body de `/mcp` | 1 MiB |
| Rate limit por token | 60 req/min (configurable) |
| Access token | 1 hora |
| Refresh token | 30 días con rotación |
| Código de autorización | 5 minutos, un solo uso |
| Auditoría | 5 MB por archivo; `audit.jsonl` + 3 rotaciones |
| Timeout de lectura | 30 s |

## Seguridad y operación

- La autenticación del endpoint es OAuth 2.1; la API key es una credencial **distinta**, usada por el MCP para llamar al backend. Ver [oauth.md](oauth.md) y [security.md](security.md).
- Las escrituras requieren `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` en modo HTTP.
- La publicación con Funnel es un paso manual del usuario: [funnel.md](funnel.md).
- Si el endpoint deja de ser necesario, `tailscale funnel reset`, `POST /admin/revoke` y detener el contenedor.
