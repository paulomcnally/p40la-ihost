---
title: "Despliegue 24/7 en el iHost"
sidebar_position: 11
description: "Runbook para correr el MCP remoto en el iHost con un nodo Tailscale dedicado y Funnel"
---

# Despliegue 24/7 en el iHost

Esta guía deja el MCP remoto corriendo **24/7 en el iHost** y describe cómo publicarlo con **Tailscale Funnel** en un **nodo dedicado**. El nodo que ya sirve el dashboard (`ihost.example.ts.net`) **no se modifica**: conserva su Serve en 443, su estado y su configuración.

El detalle de los assets de despliegue (compose de referencia, archivo de entorno y `serve.json`) está en [`deploy/mcp/README.md`](https://github.com/paulomcnally/p40la-ihost/blob/main/deploy/mcp/README.md). El comportamiento y los límites de Funnel, en [funnel.md](funnel.md).

> **Este repositorio es público.** Todos los ejemplos usan placeholders: `p40la-mcp.example.ts.net`, `ihost.example.ts.net`, `192.0.2.10`, `demo-api-key` y hashes `demo`. Nunca pegues el nombre real de tu tailnet, direcciones del rango CGNAT de Tailscale o rangos privados, auth keys, tokens, API keys ni capturas con datos reales.

## Por qué dos nodos Tailscale

El iHost ya tiene un nodo Tailscale que publica el dashboard con **Serve** en 443, solo dentro de la tailnet:

```
ihost.example.ts.net  ── Serve 443 ──► 127.0.0.1:8088   (dashboard, solo tailnet)
```

Tailscale **no permite Serve y Funnel en el mismo puerto**. Si se activa Funnel en ese 443, todo el puerto se vuelve público —handlers incluidos— y el dashboard quedaría expuesto en Internet. Repartir `/mcp`, `/.well-known/*`, `/register`, `/authorize` y `/token` como handlers por path tampoco es buena idea: es frágil y mezcla el ciclo de vida del dashboard con el del MCP.

Por eso el MCP usa un **segundo nodo**, con identidad, estado y Funnel propios:

```
p40la-mcp.example.ts.net ── Funnel 443 ──► 127.0.0.1:8090  (MCP, público con OAuth)
```

| Nodo | Contenedor | Publicación | Alcance |
|---|---|---|---|
| `ihost.example.ts.net` | Tailscale actual del add-on | Serve 443 → `127.0.0.1:8088` | Solo tailnet. **No se toca.** |
| `p40la-mcp.example.ts.net` | `tailscale-mcp` | Funnel 443 → `127.0.0.1:8090` | Internet, protegido por OAuth |

Reglas del despliegue:

- **No** ejecutes `tailscale funnel` en el nodo actual ni edites su `serve.json`, sus volúmenes o su contenedor.
- **No** apuntes Funnel a `8088` (dashboard) ni a `8091` (listener administrativo del MCP).
- Habilitar Funnel es una **decisión explícita del usuario** y un paso de este runbook; el repositorio no lo activa solo.
- El nodo dedicado usa `TS_USERSPACE=true` porque el iHost no tiene `/dev/net/tun`. En modo userspace no debería tomar el 443 del host; verificalo antes de seguir y, si choca, usá el [fallback](#fallback-si-el-443-choca).

## Diagrama

```
claude.ai / Claude móvil
        │ HTTPS 443 + OAuth 2.1
        ▼
┌──────────────────────────────────────────────┐
│ tailscale-mcp  (imagen wrapper, red host)    │
│ p40la-mcp.example.ts.net                     │
│ TS_USERSPACE=true · estado en volumen propio │
│ Funnel 443 ──► http://127.0.0.1:8090         │
└───────────────────┬──────────────────────────┘
                    │ loopback
                    ▼
┌──────────────────────────────────────────────┐
│ p40la-mcp  (red host, restart unless-stopped)│
│ :8090 /mcp + OAuth · :8091 health/admin      │
│ /data/oauth.json · /data/audit.jsonl         │
└───────────────────┬──────────────────────────┘
                    │ HTTP + Authorization: Bearer <API key>
                    ▼
┌──────────────────────────────────────────────┐
│ app p40la-ihost :8088  (contenedor actual)   │
└──────────────────────────────────────────────┘

Dispositivo en la tailnet
        │ Serve 443 (sin Funnel)
        ▼
ihost.example.ts.net ──► 127.0.0.1:8088   (nodo actual, intacto)
```

## Prerrequisitos

1. **App `0.5.17` o superior** desplegada en el iHost. Esa versión incluye los endpoints de API key de SPEC-145; sin ella no se puede provisionar la key.
2. **Imágenes publicadas en Docker Hub y multi-arch** (`linux/amd64`, `linux/arm/v7`, `linux/arm64`):
   - `paulomcnally/p40la-mcp:0.1.0` — servidor MCP.
   - `paulomcnally/p40la-mcp-tailscale:0.1.0` — nodo Tailscale dedicado con `serve.json` incluido.
   El iHost es `armv7l`; verificá que el manifiesto `arm/v7` exista antes de crear los contenedores.
3. **Auth key de Tailscale** para el nodo nuevo. Recomendado: one-off, con tag `tag:p40la-mcp`. Si la key no trae tag, anunciá el tag con `TS_EXTRA_ARGS` (ver [contenedores](#52-tailscale-mcp)).
4. **Tailnet preparada**: MagicDNS y *HTTPS Certificates* habilitados, y el atributo `funnel` para el tag o el nodo en *Access controls*. Sin eso, Funnel falla por permisos.
5. **Acceso a la UI de Docker del iHost** (`http://ihost.local`) o a su API (`http://ihost.local/api/v1/rest`). El iHost no expone Docker CLI por SSH.
6. **Contraseña de administración OAuth** distinta de la del dashboard. Solo se guarda su hash bcrypt.
7. **Puertos `8090` y `8091` libres** en el loopback del iHost.
8. **Espacio y RAM** para dos contenedores livianos. Medí el RSS después de estabilizar; el objetivo es mantenerse por debajo de 120 MB combinados.

## 1. API key global

Con la app `0.5.17` ya desplegada, generá una API key desde una máquina de desarrollo:

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" \
BASE_URL="http://ihost.local:8088" \
KEY_FILE="/tmp/p40la-mcp-api-key" \
./scripts/create-api-key.sh
```

- La key se muestra **una sola vez**.
- Guardala en el iHost en `/etc/p40la-mcp.env` con permisos `0600`.
- No la pegues en issues, specs, capturas ni mensajes.

También se puede crear o rotar desde la UI de la app, sección **API key**. El detalle está en [api-key.md](api-key.md).

## 2. Hash de la contraseña OAuth

En una máquina con el binario del MCP (o con la imagen):

```bash
read -rs -p "Contraseña OAuth: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD
printf 'P40LA_MCP_OAUTH_PASSWORD_HASH=%s\n' "$HASH"
```

También sirve un contenedor descartable:

```bash
docker run --rm -it paulomcnally/p40la-mcp:0.1.0 -hash-password
```

Guardá **solo el hash** en `/etc/p40la-mcp.env`. La contraseña en claro no se guarda en el iHost y debe ser única, larga y distinta de la del dashboard.

## 3. Archivo de entorno

El archivo vive en `/etc/p40la-mcp.env`, con permisos `0600`. Usá `deploy/mcp/mcp.env.example` como base:

```dotenv
# API de p40la-ihost (loopback: el MCP comparte la red host)
P40LA_BASE_URL=http://127.0.0.1:8088
# Placeholder: la key real empieza con p40la_
P40LA_API_KEY=demo-api-key
P40LA_TIMEOUT=15s
P40LA_TIMEZONE=America/Managua

# Transporte HTTP
P40LA_MCP_TRANSPORT=http
P40LA_MCP_LISTEN=127.0.0.1:8090
P40LA_MCP_ADMIN_LISTEN=127.0.0.1:8091
P40LA_MCP_PUBLIC_URL=https://p40la-mcp.example.ts.net

# OAuth (reemplazar por el hash real)
P40LA_MCP_OAUTH_PASSWORD_HASH=$2a$12$demo-demo-demo-demo-demo-demo-demo-demo-demo-demo-demo
P40LA_MCP_OAUTH_SUBJECT=owner

# Estado y límites
P40LA_MCP_STATE_DIR=/data
P40LA_MCP_RATE_LIMIT=60
P40LA_MCP_AUDIT_MAX_MB=5

# Escrituras: ambas en true para permitir escrituras por HTTP
P40LA_MCP_ALLOW_WRITES=true
P40LA_MCP_REMOTE_WRITES=true
```

Puntos clave:

- `P40LA_BASE_URL` usa **loopback** porque el MCP comparte la red host con la app. No uses el hostname del nodo del dashboard ni dependas de MagicDNS desde el contenedor.
- `P40LA_MCP_PUBLIC_URL` es el hostname del **nodo dedicado** (sin `/mcp`), no `ihost.example.ts.net`. Debe coincidir exactamente con el FQDN que publica Funnel.
- Las dos compuertas de escritura se habilitan solo con aprobación del usuario. El flujo seguro (en facturas, `dry_run` primero; en banca, resumen y confirmación) está en [README.md](README.md#flujo-seguro-de-escritura).
- El hash bcrypt contiene `$`: si editás el archivo desde una shell, citá con comillas simples. `docker --env-file` lo lee tal cual.
- `P40LA_MCP_STATE_DIR=/data` apunta al volumen persistente.

```bash
sudo chmod 600 /etc/p40la-mcp.env
```

## 4. Volúmenes

En la UI de Docker del iHost, creá dos volúmenes persistentes:

| Volumen | Montaje | Contenido |
|---|---|---|
| `p40la-mcp-data` | `/data` del MCP | `oauth.json` (clientes y tokens) y `audit.jsonl` |
| `tailscale-mcp-state` | `/var/lib/tailscale` del nodo dedicado | Identidad, claves y configuración de Tailscale |

No hace falta un volumen de configuración para Tailscale: el `serve.json` viene **dentro de la imagen wrapper**.

Antes de actualizar cualquiera de las dos imágenes, respaldá los volúmenes (`p40la-mcp-data` y `tailscale-mcp-state`) desde la UI o copiando sus archivos. No los commitees ni los subas al repositorio. Conservar el volumen de Tailscale evita tener que reautenticar el nodo dedicado.

El proceso del MCP corre como `nonroot` (UID/GID `65532`). Docker crea los volúmenes nombrados propiedad de `root`, así que el primer arranque puede fallar con `oauth.json: permission denied`. Si la UI no permite inicializar el volumen con ese dueño, usá un bind mount preparado por SSH (el iHost sí permite escribir archivos, aunque no tenga Docker CLI):

```bash
sudo install -d -o 65532 -g 65532 -m 0700 /opt/p40la-mcp/data
```

y montá `/opt/p40la-mcp/data` → `/data` en la UI. El síntoma y la corrección están en [Troubleshooting](#troubleshooting).

## 5. Alta de contenedores en la UI del iHost

El iHost no tiene Docker CLI por SSH: usá la UI (`http://ihost.local`) o la API (`http://ihost.local/api/v1/rest`). El `docker-compose.yml` de `deploy/mcp/` es una **referencia** de imágenes, variables, volúmenes y red; no se ejecuta en el iHost.

Orden recomendado:

1. Crear los volúmenes.
2. Crear y arrancar `p40la-mcp`.
3. Verificar que el health local responde.
4. Crear y arrancar `tailscale-mcp`.
5. Verificar que el dashboard sigue intacto **antes** de validar la publicación.
6. Verificar Funnel, DNS, TLS y el conector.

### 5.1 `p40la-mcp`

| Campo | Valor |
|---|---|
| Nombre | `p40la-mcp` |
| Imagen | `paulomcnally/p40la-mcp:0.1.0` |
| Red | `host` |
| Restart | `unless-stopped` |
| Volumen | `p40la-mcp-data` → `/data` (o bind mount con dueño `65532`) |
| Variables | las de `/etc/p40la-mcp.env` |
| Puertos | ninguno publicado; los listeners quedan en `127.0.0.1:8090` y `127.0.0.1:8091` |

Notas:

- Con red host no se configuran `ports:`; el MCP escucha solo en loopback.
- La imagen es distroless: no tiene shell ni `curl`. El health se consulta desde el host o desde el contenedor de Tailscale.
- Si el contenedor reinicia en loop, mirá los logs en la UI: suelen ser una variable faltante, un hash inválido o `P40LA_MCP_PUBLIC_URL` con path.

### 5.2 `tailscale-mcp`

| Campo | Valor |
|---|---|
| Nombre | `tailscale-mcp` |
| Imagen | `paulomcnally/p40la-mcp-tailscale:0.1.0` |
| Red | `host` (primario) |
| Restart | `unless-stopped` |
| Volumen | `tailscale-mcp-state` → `/var/lib/tailscale` |
| `TS_AUTHKEY` | auth key del nodo nuevo (idealmente con `tag:p40la-mcp`) |
| `TS_HOSTNAME` | `p40la-mcp` |
| `TS_USERSPACE` | `true` |
| `TS_STATE_DIR` | `/var/lib/tailscale` |
| `TS_SERVE_CONFIG` | `/config/serve.json` (incluido en la imagen) |
| `TS_EXTRA_ARGS` | opcional: `--advertise-tags=tag:p40la-mcp` si la key no trae tag |

La imagen `p40la-mcp-tailscale` está basada en `tailscale/tailscale` e incluye un `serve.json` con:

```json
{
  "TCP": { "443": { "HTTPS": true } },
  "Web": {
    "${TS_CERT_DOMAIN}:443": {
      "Handlers": {
        "/": { "Proxy": "http://127.0.0.1:8090" }
      }
    }
  },
  "AllowFunnel": { "${TS_CERT_DOMAIN}:443": true }
}
```

- `${TS_CERT_DOMAIN}` se sustituye por el FQDN del nodo dedicado.
- No edites archivos dentro del volumen de estado para cambiar el proxy: el `serve.json` se mantiene en la imagen.
- No reutilices el volumen ni el estado del Tailscale actual: cada nodo necesita su propia identidad y su propio directorio.
- `TS_HOSTNAME=p40la-mcp` produce `p40la-mcp.example.ts.net` en tu tailnet.
- La auth key se usa para registrar el nodo. El estado persiste en el volumen; si el volumen se pierde, hace falta una key nueva.
- Arrancá `p40la-mcp` **antes** que `tailscale-mcp`: si el proxy de Funnel no encuentra el listener, el primer request devuelve error.

### 5.3 Verificación temprana del dashboard

Apenas arranque `tailscale-mcp`, comprobá desde un dispositivo dentro de la tailnet:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' https://ihost.example.ts.net
# 200
```

Si el dashboard no responde, detené `tailscale-mcp` de inmediato: el segundo Tailscale habría tomado el 443 del host. No sigas con la publicación hasta resolverlo ([fallback](#fallback-si-el-443-choca)). El nodo del dashboard no se reinicia ni se reconfigura en ningún caso.

## 6. Verificación

### 6.1 El dashboard sigue igual

Desde un dispositivo en la tailnet:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' https://ihost.example.ts.net
# 200
```

Desde **fuera** de la tailnet (datos móviles), el dashboard **no** debe responder: si aparece, el Funnel se activó en el nodo equivocado. El nodo actual no debe tener Funnel habilitado; si su consola muestra un handler de Funnel, revertir.

### 6.2 Health del MCP en el iHost

Desde la consola del contenedor `tailscale-mcp` (red host; su imagen basada en Alpine incluye `wget`) o desde el add-on SSH del iHost:

```bash
wget -qO- http://127.0.0.1:8091/healthz
# {"status":"ok","version":"0.1.0","uptime_s":42,"writes_local":true,"writes_remote":true}
```

El listener `8091` es administrativo y solo acepta loopback. No se publica y no debe ser alcanzable desde Internet.

### 6.3 DNS público con A IPv4

Desde fuera de la tailnet, con un resolutor público:

```bash
dig +short @1.1.1.1 p40la-mcp.example.ts.net A
# 192.0.2.10  (ejemplo de IPv4 pública; la real la asigna Funnel)
```

- Claude es **IPv4-only**: si el nombre solo devuelve `AAAA`, el conector no va a conectar.
- La propagación puede tardar hasta 10 minutos. Si no aparece el registro A, revisá que Funnel esté activo y que el nodo esté online.

### 6.4 TLS, discovery y 401

Siempre desde fuera de la tailnet:

```bash
curl -sS https://p40la-mcp.example.ts.net/.well-known/oauth-protected-resource
# {"resource":"https://p40la-mcp.example.ts.net/mcp","authorization_servers":["https://p40la-mcp.example.ts.net"],...}

curl -sS -o /dev/null -w '%{http_code}\n' https://p40la-mcp.example.ts.net/mcp
# 401

curl -sSI https://p40la-mcp.example.ts.net/mcp | grep -i '^www-authenticate'
# WWW-Authenticate: Bearer resource_metadata="https://p40la-mcp.example.ts.net/.well-known/oauth-protected-resource"
```

El certificado debe ser válido y corresponder al nodo dedicado. Si `curl` falla por TLS, revisá *HTTPS Certificates* y MagicDNS en la tailnet.

### 6.5 Superficie que no debe ser pública

```bash
# Listener administrativo y revocación: 404 desde Internet
curl -sS -o /dev/null -w '%{http_code}\n' https://p40la-mcp.example.ts.net/healthz
# 404
curl -sS -o /dev/null -w '%{http_code}\n' https://p40la-mcp.example.ts.net/admin/revoke
# 404

# Dashboard y 8088: no publicados
curl -sS -o /dev/null -w '%{http_code}\n' https://ihost.example.ts.net
# sin respuesta fuera de la tailnet
```

Checklist:

- [ ] El dashboard responde `200` dentro de la tailnet y no fuera.
- [ ] `tailscale-mcp` está online y su `funnel status` muestra 443 → `127.0.0.1:8090`.
- [ ] `/healthz` del MCP responde en `127.0.0.1:8091`.
- [ ] El FQDN dedicado resuelve con **A IPv4** pública.
- [ ] Discovery responde `200` y `resource` apunta a `/mcp`.
- [ ] `/mcp` sin token responde `401` con `WWW-Authenticate`.
- [ ] `/healthz`, `/admin/revoke`, el dashboard y el `8088` no son públicos.
- [ ] Tras reiniciar el iHost, ambos contenedores vuelven a arrancar solos.

## 7. Conector en claude.ai

1. En claude.ai, **Configuración → Conectores → Agregar conector personalizado**.
2. URL del servidor: `https://p40la-mcp.example.ts.net/mcp`.
3. Autenticación: **OAuth** (registro dinámico de cliente). Claude descubre los endpoints desde `/.well-known/*`.
4. Iniciá sesión con la **contraseña de administración** (la que hasheaste). No uses la del dashboard.
5. Autorizá el conector y verificá que aparezcan las tools.
6. Probá una lectura (`bills_list` o `bank_accounts_search`). En facturas, para una escritura primero usá `dry_run=true`, mostrá el resumen y pedí confirmación; recién después `dry_run=false`. En banca no hay `dry_run`: mostrá el banco y la moneda resueltos, el cuerpo y los duplicados, pedí confirmación y ejecutá.
7. Revisá `audit.jsonl` en el volumen para confirmar que la llamada quedó registrada.

Si el conector queda en `401` después de autorizar, revisá [oauth.md](oauth.md) y la sección de revocación de [operations.md](operations.md#rotación-y-revocación).

## 8. Rollback

El dashboard no se ve afectado en ningún paso.

1. Detener `tailscale-mcp` desde la UI. El nodo dedicado queda offline y el endpoint público deja de responder.
2. Detener `p40la-mcp` si no se usará más.
3. Si querés despublicar formalmente el nodo dedicado, ejecutá `tailscale funnel reset` en su consola (si la UI permite comandos) o quitá el atributo `funnel` del tag en *Access controls*.
4. Si el nodo no se reutilizará, eliminalo en la consola de administración de Tailscale. Esa acción invalida su identidad: para volver a levantarlo necesitarás una auth key nueva.
5. Si el endpoint estuvo expuesto, revocá los tokens OAuth:

   ```bash
   curl -s -X POST http://127.0.0.1:8091/admin/revoke
   ```

6. Para volver a levantar todo, arrancá primero `p40la-mcp` y después `tailscale-mcp`. Los tokens OAuth persisten en `p40la-mcp-data` y la identidad de Tailscale en `tailscale-mcp-state`.
7. Verificá de nuevo el dashboard ([6.1](#61-el-dashboard-sigue-igual)).

## Fallback si el 443 choca

El modo userspace **no debería** bindear el 443 del host, pero si al arrancar `tailscale-mcp` el dashboard deja de responder, el segundo Tailscale tomó ese puerto. El fallback documentado es aislarlo en una red bridge propia:

1. Detené `tailscale-mcp` y confirmá que el dashboard volvió.
2. Cambiá `tailscale-mcp` a una red bridge con `host.docker.internal:host-gateway`.
3. Usá un `serve.json` que apunte a `http://host.docker.internal:8090` en lugar de `127.0.0.1:8090`.
4. Volvé a arrancar y verificá el 443 del host antes de continuar.

> **Límite del fallback:** el MCP valida que su listener sea loopback (`127.0.0.1:8090`), así que desde una red bridge `host.docker.internal:8090` no llega al proceso por sí solo. El bridge necesita un proxy en el host hacia `127.0.0.1:8090` (por ejemplo, un forward local) o el MCP no responderá. Si no podés montarlo, dejá `tailscale-mcp` detenido y volvé a la topología anterior (MCP en una laptop) hasta resolverlo. Nunca actives Funnel en el nodo del dashboard como atajo.

## Troubleshooting

| Síntoma | Causa probable | Acción |
|---|---|---|
| `p40la-mcp` reinicia en loop | Variable faltante, hash inválido o `PUBLIC_URL` con path | Revisar logs en la UI y [configuration.md](configuration.md#validaciones-del-modo-http) |
| `oauth.json: permission denied` | `/data` no escribible por UID 65532 | Bind mount con `install -d -o 65532 -g 65532 -m 0700` o inicializar el volumen con ese dueño |
| `tailscale-mcp` no arranca | Auth key inválida o vencida, o falta el atributo `funnel` | Generar otra key y revisar *Access controls* |
| El dashboard deja de responder | El segundo Tailscale tomó el 443 del host | Detener `tailscale-mcp`; ver [fallback](#fallback-si-el-443-choca) |
| Funnel activo pero Claude no llega | Sin A IPv4, DNS sin propagar o MCP caído | Esperar hasta 10 min; verificar con `1.1.1.1`; probar `/healthz` |
| `502` desde Internet | Funnel proxya a un puerto cerrado | Revisar `P40LA_MCP_LISTEN` y el estado de `p40la-mcp` |
| `401` en `/mcp` | Access token vencido o revocado | Reautorizar el conector; revisar `/admin/revoke` |
| `writes_disabled` | Falta una de las dos compuertas | `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |
| El login OAuth falla | Hash mal copiado, contraseña equivocada o reloj desfasado | Regenerar el hash y reiniciar; revisar la hora del iHost |
| RAM alta | MCP + Tailscale + app en un nodo pequeño | Medir RSS; limitar contenedores; si no hay margen, volver a la laptop |
| El nodo no aparece en la tailnet | Auth key sin tag o expirada, o `TS_HOSTNAME` en conflicto | Revisar logs; usar una key nueva y un hostname único |

## Seguridad

- El endpoint es **público**: OAuth 2.1 es obligatorio. No hay “URL secreta”.
- `/healthz` y `/admin/revoke` viven en `127.0.0.1:8091`, fuera de Funnel.
- El dashboard, su nodo Tailscale y el puerto `8088` **no** se publican por Funnel.
- Los secretos viven solo en `/etc/p40la-mcp.env` (`0600`) y en el volumen de Tailscale. El repo no contiene auth keys, API keys ni hashes reales.
- La API key es global y tiene todos los permisos: rotala si se filtra (`SEC-INCIDENT`).
- La auth key del nodo dedicado debe ser de un solo uso y con tag; revocala si se filtra.
- La auditoría registra `tools/call` sin argumentos, resultados, montos ni tokens. Revisá `audit.jsonl` después de las primeras llamadas.
- El rate limit por defecto es 60 req/min por token y el body máximo es 1 MiB.
- No adjuntes capturas del iHost con nombres de tailnet, IPs o claves. Si hay un incidente, rotá, purgá y documentá.

## Referencias

- [`deploy/mcp/README.md`](https://github.com/paulomcnally/p40la-ihost/blob/main/deploy/mcp/README.md) — assets, compose de referencia y verificación resumida.
- [`deploy/mcp/docker-compose.yml`](https://github.com/paulomcnally/p40la-ihost/blob/main/deploy/mcp/docker-compose.yml) y [`deploy/mcp/mcp.env.example`](https://github.com/paulomcnally/p40la-ihost/blob/main/deploy/mcp/mcp.env.example).
- [`deploy/mcp/tailscale/serve.json`](https://github.com/paulomcnally/p40la-ihost/blob/main/deploy/mcp/tailscale/serve.json) — configuración incluida en la imagen wrapper.
- [Tailscale Funnel](funnel.md) · [Servidor HTTP](http.md) · [OAuth 2.1](oauth.md) · [Seguridad](security.md) · [Operación](operations.md).
- Documentación de Tailscale: *Funnel*, *Serve*, *HTTPS certificates*, *Userspace networking* y *Auth keys*.
