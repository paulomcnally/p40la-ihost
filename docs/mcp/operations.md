---
title: "Operación"
sidebar_position: 6
description: "Build, release, ejecución, API key, diagnóstico y respaldo del servidor MCP"
---

# Operación

## Build

```bash
# Binario local
go build -o bin/p40la-mcp ./cmd/mcp-server

# Versión embebida
go build -ldflags="-X main.version=0.1.0" -o bin/p40la-mcp ./cmd/mcp-server

# Multi-plataforma (ejemplo)
GOOS=linux GOARCH=arm64 go build -o bin/p40la-mcp-linux-arm64 ./cmd/mcp-server
GOOS=darwin GOARCH=arm64 go build -o bin/p40la-mcp-darwin-arm64 ./cmd/mcp-server
```

### Imagen Docker del MCP

`Dockerfile.mcp` es independiente de la imagen del add-on web. La imagen final es distroless `nonroot` (UID/GID 65532), expone 8090 y 8091, y guarda el estado en `/data`.

El binario MCP **no** se incluye en la imagen Docker del servidor web ni se inicia con el add-on principal. Puede correr en la máquina del cliente (stdio), como contenedor aparte con OAuth (HTTP) o en el iHost con red host. Esto evita sumar procesos al add-on web y mantiene separadas las credenciales.

## Release y publicación

Cada artefacto —app web, MCP y wrapper Tailscale— se versiona y publica de forma **independiente**. `release.sh` compara el contenido de la allowlist de cada artefacto contra el commit del último tag de ese artefacto (`git diff --name-only <sha>..HEAD -- <rutas>`) y solo crea tag y dispara el workflow si hay cambios relevantes.

| Artefacto | Tag git | Allowlist | Workflow | Imagen en Docker Hub |
|---|---|---|---|---|
| App web | `vX.Y.Z` | `cmd/server/`, `internal/` (sin `internal/mcp/`), `frontend/`, `public/`, `migrations/`, `go.mod`, `go.sum`, `Dockerfile`, `.github/workflows/docker-publish.yml` | `Build & Push Docker Image` (`docker-publish.yml`) | `paulomcnally/p40la-ihost` |
| MCP | `mcp-vX.Y.Z` | `cmd/mcp-server/`, `internal/mcp/`, `Dockerfile.mcp`, `.github/workflows/mcp-publish.yml`, `go.mod`, `go.sum` | `Build & Push MCP Image` (`mcp-publish.yml`) | `paulomcnally/p40la-mcp` |
| Tailscale | `tailscale-vX.Y.Z` | `deploy/mcp/tailscale/` | `Build & Push Tailscale Image` (`tailscale-publish.yml`) | `paulomcnally/p40la-mcp-tailscale` |

La allowlist de la app excluye `internal/mcp/`: el binario de la app no importa ese paquete y los grafos de build están separados, así que un cambio solo del MCP no versiona la app.

```bash
./scripts/release.sh                             # libera solo lo que cambió (bump patch por artefacto)
./scripts/release.sh 0.5.22                      # fuerza la app; MCP y Tailscale automáticos
./scripts/release.sh 0.5.22 0.7.0                # fuerza app y MCP
./scripts/release.sh 0.5.22 0.7.0 0.2.0          # fuerza app, MCP y wrapper Tailscale
APP_VERSION=0.5.22 MCP_VERSION=0.7.0 TAILSCALE_VERSION=0.2.0 ./scripts/release.sh
./scripts/release.sh --no-mcp                    # omite el MCP (o MCP_VERSION=skip)
./scripts/release.sh --no-tailscale              # omite el wrapper (o TAILSCALE_VERSION=skip)
./scripts/release.sh --check                     # plan read-only
./scripts/release.sh --list-tags                 # último tag y cambios pendientes por artefacto
```

- **Sin cambios relevantes:** si desde el último tag solo cambiaron docs u otros archivos fuera de la allowlist de la app, la app no se versiona ni se publica. Si ningún artefacto cambió y no hay versiones explícitas, el script imprime «Nada para liberar» y termina sin commits, tags ni workflows.
- **Primera release:** el MCP y el wrapper Tailscale arrancan en `0.1.0` si no tienen tag previo. Con tag previo, el bump es de patch.
- **Versión explícita:** publica aunque no haya cambios y avisa con `AVISO: ... no se detectaron cambios ... por versión explícita`.
- **Wrapper Tailscale:** tiene tag y versión propios y ya **no** hereda la versión del MCP. `mcp-publish.yml` ya no incluye el job `build-tailscale`; el wrapper se publica con `tailscale-publish.yml`, que construye `linux/amd64`, `linux/arm/v7` y `linux/arm64` y publica `X.Y.Z` y `latest`.

`release.sh` exige estar en `main` con el árbol limpio (con `--check` o `--list-tags` solo avisa si estás en otra rama), crea el commit de bump de la app (`docker-compose.yml`, `frontend/package.json`) solo si la app se publica, crea los tags y espera la conclusión de cada workflow disparado. Antes de crear cualquier tag valida que no exista ni local ni remotamente y aborta si existe. En el release real hace `git fetch --tags origin` antes de detectar cambios.

`--check` es un plan read-only: imprime por artefacto el último tag, la versión a crear, los archivos cambiados y la decisión, sin crear commits ni tags ni hacer push, y sin ejecutar `git fetch --tags` (resuelve los tags remotos con `git ls-remote --tags origin`). Requiere que los commits de los tags base estén presentes localmente. `--list-tags` muestra, por artefacto, el último tag y si hay cambios pendientes, sin armar plan.

**Propagación de fallos:** si un workflow disparado no aparece o su conclusión no es `success`, el script sale con error y deja los tags creados para reintentar el workflow desde GitHub Actions. La app y el MCP publican una imagen por plataforma y un job de manifest (`merge`) que depende de **todos** los jobs de build de su workflow; el wrapper usa un único job multi-arch. Si falta una plataforma en el manifest, el job `merge` no pudo completarse y hay que reintentar el workflow.

### Verificar la publicación

```bash
docker buildx imagetools inspect paulomcnally/p40la-mcp:latest
docker buildx imagetools inspect paulomcnally/p40la-mcp:0.7.0
docker buildx imagetools inspect paulomcnally/p40la-mcp-tailscale:latest
docker buildx imagetools inspect paulomcnally/p40la-mcp-tailscale:0.2.0
# Platform: linux/amd64, linux/arm/v7, linux/arm64
```

El tag `latest` y el de la versión deben listar las tres plataformas en cada imagen; si falta alguna, el manifest no se pudo crear y hay que reintentar el workflow.

### Publicación manual

Si hiciste el release de un artefacto sin otro, o necesitás republicar una imagen, dispará `workflow_dispatch` del workflow correspondiente con la versión (`X.Y.Z`) como input:

- `Build & Push MCP Image` publica `paulomcnally/p40la-mcp`.
- `Build & Push Tailscale Image` publica `paulomcnally/p40la-mcp-tailscale` con esa versión y `latest`.

Ninguno crea ni mueve tags git; tené en cuenta que `latest` queda apuntando a la versión indicada.

El token de GitHub se pasa como secreto de BuildKit (`id=gh_token`) en CI, porque el módulo de plugins es privado; no queda en la imagen.

## Provisionar la API key

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" ./scripts/create-api-key.sh
```

La key se imprime una sola vez. Guardala en un gestor de secretos o en un archivo `0600`:

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" \
KEY_FILE="$HOME/.config/p40la/api-key" \
./scripts/create-api-key.sh
```

La contraseña se usa solo para esta operación; el MCP no la necesita. Detalle en [api-key.md](api-key.md).

## Ejecución (stdio)

```bash
P40LA_BASE_URL="http://ihost.local:8088" \
P40LA_API_KEY="p40la_..." \
./bin/p40la-mcp
```

El proceso espera mensajes JSON-RPC por stdin. Todo log va a stderr para no romper el protocolo.

## Arranque en modo HTTP

```bash
# Hash de la contraseña de administración (una vez)
read -rs -p "Contraseña: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | ./bin/p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD

# Estado local (en el contenedor se usa /data)
mkdir -p /tmp/p40la-mcp-state && chmod 700 /tmp/p40la-mcp-state

# Variables mínimas
export P40LA_BASE_URL="http://ihost.local:8088"
export P40LA_API_KEY="demo-api-key"
export P40LA_MCP_TRANSPORT=http
export P40LA_MCP_PUBLIC_URL="https://demo-node.example.ts.net"
export P40LA_MCP_OAUTH_PASSWORD_HASH="$HASH"
export P40LA_MCP_STATE_DIR="/tmp/p40la-mcp-state"
export P40LA_MCP_ALLOW_WRITES=false
export P40LA_MCP_REMOTE_WRITES=false

./bin/p40la-mcp
```

El proceso levanta dos listeners loopback:

- `127.0.0.1:8090`: `/mcp` y OAuth.
- `127.0.0.1:8091`: `/healthz` y `/admin/revoke`.

No arranca si alguno deja de ser loopback, si los puertos son iguales o si la URL pública tiene path. El detalle de configuración y límites está en [http.md](http.md).

## Health y revocación

```bash
# Estado del proceso (solo loopback)
curl -s http://127.0.0.1:8091/healthz
# {"status":"ok","version":"0.1.0","uptime_s":42,"writes_local":false,"writes_remote":false}

# Revocar todos los tokens y códigos OAuth
curl -s -X POST http://127.0.0.1:8091/admin/revoke
# {"status":"revoked"}
```

La revocación conserva los clientes registrados. Para un reinicio total, detener el proceso y borrar o mover `oauth.json` antes de volver a arrancar.

## Auditoría

En modo HTTP, cada `tools/call` se registra en `P40LA_MCP_STATE_DIR/audit.jsonl`:

```json
{"ts":"2026-11-23T15:04:05Z","request_id":"9f2c1a7b3d4e5f60","subject":"owner","tool":"bills_list","result":"ok","duration_ms":42}
```

- Campos: `ts`, `request_id`, `subject`, `tool`, `result`, `duration_ms`. **Sin** argumentos, resultados, montos ni tokens.
- `result` puede ser `ok`, `error` o `tool_error`.
- Permisos `0600`; el directorio, `0700`.
- Rotación al superar `P40LA_MCP_AUDIT_MAX_MB` (default 5 MB): `audit.jsonl` → `audit.1.jsonl` → `audit.2.jsonl` → `audit.3.jsonl`. El espacio máximo aproximado es `4 × P40LA_MCP_AUDIT_MAX_MB`.
- Los archivos viven en el volumen de estado y **nunca** se commitean ni se adjuntan sin sanitizar.

Para revisar actividad reciente:

```bash
sudo tail -n 20 /opt/p40la-mcp/data/audit.jsonl
```

## Despliegue en iHost

El runbook completo —por qué dos nodos Tailscale, API key, hash OAuth, volúmenes, alta por UI, verificación, rollback y troubleshooting— está en [ihost.md](ihost.md). Esta sección resume los recursos que se crean en el iHost.

| Recurso | Valor |
|---|---|
| Imagen del MCP | `paulomcnally/p40la-mcp:0.5.0` |
| Imagen del nodo dedicado | `paulomcnally/p40la-mcp-tailscale:latest` |
| Contenedor MCP | `p40la-mcp`, red `host`, `restart: unless-stopped` |
| Contenedor Tailscale | `tailscale-mcp`, red `host`, `restart: unless-stopped` |
| Volumen MCP | `p40la-mcp-data` → `/data` |
| Volumen Tailscale | `tailscale-mcp-state` → `/var/lib/tailscale` |
| Entorno MCP | `/etc/p40la-mcp.env` (`0600`) |
| Listeners MCP | `127.0.0.1:8090` (público vía Funnel) y `127.0.0.1:8091` (admin) |

En el iHost **no hay Docker CLI por SSH**: los contenedores se crean desde la UI de Docker (`http://ihost.local`) o su API (`http://ihost.local/api/v1/rest`). El compose de `deploy/mcp/docker-compose.yml` es la referencia de imágenes, variables, volúmenes y red.

### Contenedor `p40la-mcp`

El contenedor necesita red host para que el proxy de Tailscale alcance `127.0.0.1:8090`, y un volumen escribible por el UID 65532.

```bash
# Bind mount alternativo si el volumen nombrado queda propiedad de root
sudo install -d -o 65532 -g 65532 -m 0700 /opt/p40la-mcp/data

# Archivo de entorno con permisos restringidos
sudo touch /etc/p40la-mcp.env
sudo chmod 600 /etc/p40la-mcp.env
# editar /etc/p40la-mcp.env con los valores reales (ver configuration.md)
```

```yaml
# /opt/p40la-mcp/compose.yaml (referencia; en el iHost se usa la UI)
services:
  p40la-mcp:
    image: paulomcnally/p40la-mcp:0.5.0
    container_name: p40la-mcp
    restart: unless-stopped
    network_mode: host
    env_file:
      - /etc/p40la-mcp.env
    volumes:
      - /opt/p40la-mcp/data:/data   # o el volumen nombrado p40la-mcp-data
```

```bash
# Ilustrativo: el iHost no expone Docker CLI por SSH
docker compose -f /opt/p40la-mcp/compose.yaml up -d
docker logs --tail 20 p40la-mcp
```

### Contenedor `tailscale-mcp`

Es un nodo Tailscale **dedicado al MCP**, separado del nodo que publica el dashboard. La imagen wrapper incluye `/config/serve.json` con Funnel 443 → `http://127.0.0.1:8090`; el estado vive en su propio volumen.

| Variable | Valor |
|---|---|
| `TS_AUTHKEY` | auth key del nodo nuevo, idealmente con `tag:p40la-mcp` |
| `TS_HOSTNAME` | `p40la-mcp` |
| `TS_USERSPACE` | `true` (el iHost no tiene `/dev/net/tun`) |
| `TS_STATE_DIR` | `/var/lib/tailscale` |
| `TS_SERVE_CONFIG` | `/config/serve.json` (incluido en la imagen) |
| `TS_EXTRA_ARGS` | opcional: `--advertise-tags=tag:p40la-mcp` |

Notas:

- Con `network_mode: host`, no se usan `ports:`; los listeners del MCP quedan en el loopback del nodo.
- El proceso corre como `nonroot` (65532). Si el volumen pertenece a otro UID, `oauth.json` no se puede escribir y el arranque falla.
- La imagen del MCP es distroless (sin shell ni `curl`): consultá `/healthz` desde el host o desde la consola de `tailscale-mcp`.
- La imagen de la app web, la del MCP y la del nodo Tailscale se actualizan por separado.
- No modifiques el contenedor Tailscale existente, su `serve.json` ni sus volúmenes: el dashboard debe seguir igual.
- Presupuesto: medir RSS antes de dejarlo permanente en el iHost. Si no hay margen, correr el MCP en una laptop y dejar Funnel allí.

### Verificación

```bash
# 1. El dashboard sigue respondiendo dentro de la tailnet
curl -sS -o /dev/null -w '%{http_code}\n' https://ihost.example.ts.net
# 200

# 2. Health del MCP (loopback administrativo)
wget -qO- http://127.0.0.1:8091/healthz
# {"status":"ok",...}

# 3. DNS público con A IPv4 y TLS
dig +short @1.1.1.1 p40la-mcp.example.ts.net A
curl -sS https://p40la-mcp.example.ts.net/.well-known/oauth-protected-resource

# 4. /mcp sin token
curl -sS -o /dev/null -w '%{http_code}\n' https://p40la-mcp.example.ts.net/mcp
# 401
```

El detalle de cada paso y los valores esperados está en [ihost.md](ihost.md#6-verificación).

## Publicación con Funnel

Publicar el MCP en Internet es una decisión explícita del usuario y requiere que la tailnet tenga Funnel habilitado. El runbook completo está en [funnel.md](funnel.md).

```bash
# 443 → 127.0.0.1:8090
tailscale funnel --bg 8090
tailscale funnel status

# Rollback
tailscale funnel reset
```

Funnel no autentica: `/mcp` sigue exigiendo OAuth. Nunca apuntes Funnel a `8088` (dashboard) ni a `8091` (admin).

En el iHost **no ejecutes estos comandos en el nodo del dashboard**: ese nodo usa Serve en 443 y quedaría público. El MCP se publica desde el nodo dedicado (`tailscale-mcp`), que ya trae su `serve.json` en la imagen. Ver [ihost.md](ihost.md).

## Verificación previa

```bash
curl -s http://ihost.local:8088/health
# {"status":"ok","timestamp":"..."}

# Verificar que la key responde
curl -s -H "Authorization: Bearer $P40LA_API_KEY" \
  http://ihost.local:8088/api/services | head
```

Si el health falla, el MCP no puede operar. Revisá red, URL base y que el contenedor esté corriendo.

## Rotación y revocación

### API key del backend

- Rotar: `POST /api/api-key/regenerate` o el script. La key anterior deja de funcionar de inmediato.
- Revocar: `DELETE /api/api-key`.
- Después de rotar, actualizá `P40LA_API_KEY` en el entorno del MCP y reiniciá el proceso.

### OAuth del MCP remoto

- Revocar tokens: `POST http://127.0.0.1:8091/admin/revoke`. Conserva los clientes registrados.
- Reinicio total: detener el proceso y borrar o mover `P40LA_MCP_STATE_DIR/oauth.json`. Todos los conectores deberán registrarse y autorizar otra vez.
- Rotar la contraseña: generar un hash nuevo, actualizarlo en el entorno, reiniciar y **revocar** (los tokens vigentes no se invalidan solos).
- La revocación de tokens OAuth **no** cambia la API key, y rotar la API key **no** invalida los tokens OAuth.

### Funnel

- Despublicar: `tailscale funnel reset` y verificar con `tailscale funnel status`.
- En el iHost, el reset se hace en `tailscale-mcp`; el nodo del dashboard no se toca.
- Si el nodo también servía el dashboard en 443, restaurar `tailscale serve` después del reset.
- Al dar de baja un conector: revocar tokens, eliminarlo en claude.ai y, si no se usará más, detener el contenedor.

## Procedimiento de backfill

Ejemplo para un servicio mensual, enero a la fecha:

1. `bills_list` con `service_id` para ver qué existe.
2. `bills_generate` con `from=2026-01`, `to=2026-11`, `dry_run=true` (default). Debe reportar 11 planificadas si no hay ninguna.
3. Mostrar el resumen al usuario y pedir confirmación.
4. Respaldar la DB del iHost según `docs/rules/40-data.md`.
5. Repetir con `dry_run=false`.
6. Verificar con `bills_list` que hay 11 facturas y que una muestra `source=mcp` en su historial.

Si el MCP corre por HTTP, además de `P40LA_MCP_ALLOW_WRITES=true` debe estar `P40LA_MCP_REMOTE_WRITES=true`. No generes meses futuros: el backend los rechaza. No repitas a ciegas si hubo un error parcial: la operación es idempotente y el segundo intento reporta lo ya creado como `skipped`.

## Diagnóstico

| Síntoma | Causa probable | Acción |
|---|---|---|
| `auth_failed` | Key inválida, revocada o rotada | Regenerar la key y actualizar `P40LA_API_KEY` |
| `writes_disabled` | Compuerta local o remota apagada | Definir `P40LA_MCP_ALLOW_WRITES=true`; en HTTP, también `P40LA_MCP_REMOTE_WRITES=true` |
| `upstream_error` | URL, red o servidor caído | Probar `/health`, revisar `P40LA_BASE_URL` |
| `service_not_found` | `service_id` incorrecto | Verificar en la UI o con `bills_list` |
| `service_inactive` | Servicio desactivado | Activarlo antes de generar |
| `amount_unresolved` | Sin monto explícito, sin factura previa y sin monto sugerido | Pasar `amount` o configurar el servicio |
| `deleted_slot` | Factura eliminada ocupando el período | Restaurarla o eliminar el registro desde la UI; no se recrea sola |
| Timeout | Red lenta o servidor cargado | Subir `P40LA_TIMEOUT` hasta 60s y revisar el iHost |
| `/mcp` responde `401 invalid_token` | Access token vencido (1 h) o revocado | El cliente debe refrescar; si falla, reautorizar |
| `429 too_many_requests` | Rate limit por token o por IP | Esperar `Retry-After`; subir `P40LA_MCP_RATE_LIMIT` solo si es legítimo |
| `413 payload_too_large` | Body mayor a 1 MiB | Reducir el payload; el MCP normal no envía cuerpos grandes |
| El proceso no arranca: “debe escuchar en loopback” | `P40LA_MCP_LISTEN` o admin no loopback | Corregir a `127.0.0.1` |
| El proceso no arranca: “debe ser el origen, sin path” | `P40LA_MCP_PUBLIC_URL` incluye `/mcp` | Quitar path, query y fragment |
| El proceso no arranca: “debe ser un hash bcrypt” | Hash mal copiado o contraseña en claro | Regenerar con `-hash-password` |
| `oauth.json: permission denied` | `/data` no escribible por UID 65532 | `chown 65532:65532` y `chmod 0700` |
| `/healthz` no responde | Proceso caído o puerto admin distinto | Revisar `docker logs` y `P40LA_MCP_ADMIN_LISTEN` |
| Funnel activo pero Claude no llega | MCP no escucha en 8090 o `PUBLIC_URL` no coincide | Probar discovery en loopback y desde fuera de la tailnet |

## Recursos

- El binario no abre la DB ni mantiene caché en disco.
- En HTTP mantiene en memoria el estado OAuth, los contadores de rate limit y el logger de auditoría; el estado persistente son dos archivos.
- Una generación de 11 facturas hace una sola llamada HTTP y el backend procesa secuencialmente.
- Límite de 36 períodos por generación.
- No hay reintentos automáticos de escrituras.
- La auditoría ocupa como máximo `4 × P40LA_MCP_AUDIT_MAX_MB`.
- Si corre en el iHost, medir RSS antes de dejarlo permanente; el objetivo es mantenerse por debajo de 50 MB.

## Tests

```bash
go build ./...
go test ./internal/services/ ./internal/api/ ./internal/mcp/...
```

Los tests del MCP cubren configuración, OAuth (PKCE, tokens, revocación), listeners (401/429/413, health, revoke) y auditoría. Los tests de integración usan una DB temporal (`t.TempDir()`) y un directorio de estado temporal, nunca `data/app.db`.

## Logs

- Nivel informativo al iniciar: versión, transporte, URL base, listeners, URL pública, estado de las compuertas y rate limit.
- Cada request HTTP registra método, path, estado y duración; **no** registra headers, tokens, cuerpos ni IPs.
- Sin API key, montos, emails ni cuerpos de request/response.
- Los errores HTTP se muestran al agente con código y mensaje, no en logs con payload.
