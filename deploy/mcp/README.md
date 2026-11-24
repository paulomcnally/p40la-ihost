# MCP en el iHost con nodo Tailscale dedicado

> SPEC-148. El MCP remoto corre 24/7 en el iHost y se publica con un **nodo
> Tailscale dedicado**. El nodo actual del dashboard **no se toca**.

## Por qué dos nodos

El iHost ya publica el dashboard con **Tailscale Serve** en 443:

```
ihost.<tailnet>.ts.net  ── Serve 443 ──► 127.0.0.1:8088   (solo tailnet)
```

Tailscale **no permite Serve y Funnel en el mismo puerto**: si se activa Funnel
en ese 443, todo el puerto se vuelve público, incluido el dashboard. Por eso el
MCP usa otro nodo:

```
p40la-mcp.<tailnet>.ts.net ── Funnel 443 ──► 127.0.0.1:8090  (MCP, público con OAuth)
```

- Nodo actual: no se modifica, no se reinicia, no se le cambia el `serve.json`.
- Nodo nuevo: contenedor `tailscale-mcp`, estado propio, hostname propio, Funnel propio.
- Si el segundo Tailscale choca con el 443 del host, se detiene y se evalúa un
  proxy en el host; el bridge simple no alcanza porque el MCP solo escucha
  loopback. Ver `docs/mcp/ihost.md` → “Fallback si el 443 choca”.

## Prerrequisitos

1. App `paulomcnally/p40la-ihost:0.5.17` o superior desplegada en el iHost
   (incluye los endpoints de API key de SPEC-145).
2. Imágenes publicadas:
   - `paulomcnally/p40la-mcp:0.1.4`
   - `paulomcnally/p40la-mcp-tailscale:0.1.0`
3. **Auth key de Tailscale** para el nodo nuevo. Recomendado: tag `tag:p40la-mcp`.
4. En la tailnet: **MagicDNS** y **HTTPS Certificates** habilitados, y el atributo
   `funnel` para el tag/nodo en Access Controls.
5. Acceso a la UI de Docker del iHost (`http://ihost.local`).
6. Una contraseña de administración para OAuth (distinta de la del dashboard).

## 1. API key del MCP

Con la app 0.5.17 ya desplegada, generá una API key global:

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" \
BASE_URL="http://ihost.local:8088" \
KEY_FILE="/tmp/p40la-mcp-api-key" \
./scripts/create-api-key.sh
```

- La key se muestra una sola vez.
- Guardala en el iHost en `/etc/p40la-mcp.env` (permisos `0600`).
- No la pegues en issues, specs ni capturas.

Alternativa: desde la UI de la app, sección API key.

## 2. Hash de la contraseña OAuth

En una máquina con el binario del MCP:

```bash
read -rs -p "Contraseña OAuth: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD
printf 'P40LA_MCP_OAUTH_PASSWORD_HASH=%s\n' "$HASH"
```

Guardá solo el hash en `/etc/p40la-mcp.env`. La contraseña en claro no se guarda
en el iHost.

## 3. Archivo de entorno

Usá `mcp.env.example` como base y completá:

- `P40LA_API_KEY`: la key real.
- `P40LA_MCP_PUBLIC_URL`: `https://p40la-mcp.<tailnet>.ts.net` (sin `/mcp`).
- `P40LA_MCP_OAUTH_PASSWORD_HASH`: el hash bcrypt.
- `P40LA_MCP_ALLOW_WRITES=true` y `P40LA_MCP_REMOTE_WRITES=true` (escrituras).
- `P40LA_BASE_URL=http://127.0.0.1:8088` (el MCP comparte la red host).

Permisos: `chmod 600 /etc/p40la-mcp.env`.

## 4. Volúmenes

En la UI de Docker del iHost, creá:

| Volumen | Uso |
|---|---|
| `p40la-mcp-data` | `/data` del MCP: `oauth.json`, `audit.jsonl` |
| `tailscale-mcp-state` | `/var/lib/tailscale` del nodo dedicado |

## 5. Contenedores

`docker-compose.yml` es la referencia. En la UI del iHost, creá dos contenedores:

### `p40la-mcp`

| Campo | Valor |
|---|---|
| Imagen | `paulomcnally/p40la-mcp:0.1.4` |
| Red | `host` |
| Restart | `unless-stopped` |
| Volumen | `p40la-mcp-data` → `/data` |
| Variables | las de `/etc/p40la-mcp.env` |

### `tailscale-mcp`

| Campo | Valor |
|---|---|
| Imagen | `paulomcnally/p40la-mcp-tailscale:0.1.0` |
| Red | `host` (primario) |
| Restart | `unless-stopped` |
| Volumen | `tailscale-mcp-state` → `/var/lib/tailscale` |
| `TS_AUTHKEY` | auth key del nodo nuevo |
| `TS_HOSTNAME` | `p40la-mcp` |
| `TS_USERSPACE` | `true` |
| `TS_STATE_DIR` | `/var/lib/tailscale` |
| `TS_SERVE_CONFIG` | `/config/serve.json` (ya viene en la imagen) |

La imagen `p40la-mcp-tailscale` incluye `serve.json` con:

- `443` HTTPS.
- `/` → `http://127.0.0.1:8090`.
- `AllowFunnel: true`.

## 6. Verificación

### 6.1 El dashboard sigue igual

Desde un dispositivo en la tailnet:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' https://ihost.<tailnet>.ts.net
# 200
```

El nodo actual no debe tener Funnel habilitado. Si `tailscale funnel status` en
ese contenedor muestra Funnel, algo se configuró mal: revertir.

### 6.2 El MCP responde en el iHost

Desde la consola del contenedor o la UI:

```bash
wget -qO- http://127.0.0.1:8091/healthz
# {"status":"ok",...}
```

### 6.3 DNS público y TLS

Desde fuera de la tailnet (datos móviles), con un resolutor público:

```bash
dig +short @1.1.1.1 p40la-mcp.<tailnet>.ts.net A
# una IP pública (IPv4); Claude es IPv4-only
```

```bash
curl -sS https://p40la-mcp.<tailnet>.ts.net/.well-known/oauth-protected-resource
curl -sS -o /dev/null -w '%{http_code}\n' https://p40la-mcp.<tailnet>.ts.net/mcp
# 401
curl -sSI https://p40la-mcp.<tailnet>.ts.net/mcp | grep -i '^www-authenticate'
```

### 6.4 Conector en claude.ai

1. Configuración → Conectores → Agregar conector personalizado.
2. URL: `https://p40la-mcp.<tailnet>.ts.net/mcp`.
3. Autenticación: OAuth (DCR). Iniciar sesión con la contraseña de administración.
4. Probar `lists`/lectura y una escritura con `dry_run=false` solo si querés.

## 7. Rollback

1. Detener `p40la-mcp` y `tailscale-mcp` desde la UI.
2. Si hace falta, resetear el Funnel del nodo dedicado:
   `tailscale funnel reset` dentro de `tailscale-mcp` (si la UI permite consola).
3. El dashboard `ihost.<tailnet>.ts.net` no se ve afectado en ningún paso.
4. Para volver a levantarlo, iniciar los contenedores; el estado de Tailscale y
   los tokens OAuth persisten en los volúmenes.

## Troubleshooting

| Síntoma | Causa probable | Acción |
|---|---|---|
| `tailscale-mcp` no arranca | Auth key inválida/expirada o falta atributo funnel | Generar otra key, revisar Access Controls |
| El dashboard deja de responder | El segundo Tailscale tomó el 443 del host | Detener `tailscale-mcp`; ver `docs/mcp/ihost.md` → fallback con proxy |
| Funnel activo pero Claude no llega | Sin A IPv4 público o DNS sin propagar | Esperar hasta 10 min; verificar con 1.1.1.1 |
| `401` en `/mcp` | Token vencido o no autorizado | Reautorizar el conector; revisar `/admin/revoke` |
| `writes_disabled` | Falta uno de los dos flags | `ALLOW_WRITES=true` y `REMOTE_WRITES=true` |
| RAM alta | MCP + Tailscale + app | Medir RSS; si no hay margen, reducir límites o volver a laptop |

## Seguridad

- El endpoint es público: OAuth es obligatorio.
- `/healthz` y `/admin/revoke` viven en `127.0.0.1:8091`, fuera de Funnel.
- El dashboard y el puerto 8088 no se publican por Funnel.
- Los secretos viven solo en `/etc/p40la-mcp.env` y en el volumen de Tailscale.
- El API del iHost no acepta `memory`, `cpu_shares` ni `container_path` (devuelve 500) y exige
  `port_list`/`usb_devices` aunque estén vacíos. El payload exacto está en
  `docs/mcp/ihost.md` → 5.1.1.
- Si se filtra la auth key o la API key: revocar, rotar y documentar (`SEC-INCIDENT`).
