# Configuración de Tailscale Serve + PWA en el iHost

> Runbook de la configuración aplicada en el iHost para publicar `p40la-ihost` como PWA en `https://ihost.<tailnet>.ts.net`, sin `docker exec`.
>
> **Importante**: este runbook configura TLS y el proxy. La imagen Docker del iHost debe incluir los cambios de manifest/service worker de la SPEC-130; una imagen vieja no se vuelve instalable solo por configurar Tailscale.

## 1. Resultado esperado

| Elemento | Valor |
|---|---|
| Nodo Tailscale | el nodo existente (`ihost` en este iHost) |
| Origen PWA | `https://ihost.<tailnet>.ts.net` |
| Volumen de estado | `tailscale` → `/data/tailscalState` (se conserva) |
| Volumen de configuración | `tailscale-config` → `/config` |
| Archivo | `/config/serve.json` dentro del contenedor |
| Variable | `TS_SERVE_CONFIG=/config/serve.json` |
| Red | `host` |
| Filebrowser | puerto `8083`, solo para editar `serve.json`; se detiene después |

## 2. Prerrequisitos

1. **MagicDNS** y **HTTPS Certificates** habilitados en la tailnet:
   `https://login.tailscale.com/admin/dns`.
2. Acceso a la UI de Docker del iHost (`http://ihost.local`).
3. Un **token de sesión** de la UI del iHost para usar la API (ver §3).
4. No se necesita `docker exec`, SSH ni consola dentro de los contenedores.

## 3. Autenticación contra la API del iHost

La UI usa la API `http://ihost.local/api/v1/rest` con `Authorization: Bearer <token>`.

Para obtener el token:

1. Abrí `http://ihost.local` e iniciá sesión.
2. DevTools → **Network** → filtrá por `api/v1/rest`.
3. Abrí cualquier request y copiá el valor de `Authorization` (sin `Bearer `).
4. Guardalo **solo en una variable de entorno**:
   ```bash
   export IHOST_TOKEN='...'
   ```
   No lo commitees ni lo guardes en archivos. Es un token de sesión: caduca y hay que renovarlo.

> No existe un script de login con password en este repo: el login de la UI usa nonce y cifrado del lado del navegador. El token se obtiene de la sesión ya autenticada.

## 4. Descubrir los IDs de los add-ons

Los IDs numéricos de la API **no** son los hashes de `addon_info`:

```bash
curl -s -H "Authorization: Bearer $IHOST_TOKEN" \
  http://ihost.local/api/v1/rest/docker/addons | jq '.data[] | {id,name}'
```

En este iHost:

| Add-on | `addon_id` |
|---|---|
| `filebrowser/filebrowser` | `9` |
| `tailscale/tailscale` | `21` |

En otro iHost pueden ser distintos: usá el ID que devuelve `/docker/addons`.

## 5. Crear el volumen de configuración

```bash
curl -s -X POST -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"volume_name":"tailscale-config"}' \
  http://ihost.local/api/v1/rest/docker/volumes
```

Si ya existe, la API devuelve error; se puede ignorar.

## 6. Crear `serve.json`

Contenido (también en `deploy/tailscale/serve.json`):

```json
{
  "TCP": {
    "443": {
      "HTTPS": true
    }
  },
  "Web": {
    "${TS_CERT_DOMAIN}:443": {
      "Handlers": {
        "/": {
          "Proxy": "http://127.0.0.1:8088"
        }
      }
    }
  },
  "AllowFunnel": {
    "${TS_CERT_DOMAIN}:443": false
  }
}
```

`${TS_CERT_DOMAIN}` lo sustituye Tailscale por el nombre del nodo (`ihost.<tailnet>.ts.net`). `AllowFunnel: false` lo mantiene privado.

Para escribirlo se usa Filebrowser (ver §7) o el script `scripts/ihost-pwa-setup.mjs`, que lo crea por la API de Filebrowser.

## 7. Recrear Filebrowser con el volumen correcto

En este iHost el Filebrowser original tenía el volumen de estado `tailscale` montado en `/srv` (causaba `Forbidden`) y el puerto `8081` estaba ocupado por `presence-ihost`. La configuración correcta es:

| Volume | Container path |
|---|---|
| `tailscale-config` | `/srv` |
| `filebrowser-db` | `/database` |
| `filebrowser-config` | `/config` |

- Red: `bridge`
- Puerto: host `8083` → container `80`
- `UID=0`, `GID=0` (temporal, para poder escribir en el volumen)
- `PATH` se conserva del contenedor original

Pasos por API:

```bash
# 1. Detener y eliminar el contenedor viejo (NO borra volúmenes)
curl -s -X PUT -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' -d '{"state":"stop"}' \
  http://ihost.local/api/v1/rest/docker/containers/<FILEBROWSER_CONTAINER_ID>
curl -s -X PUT -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' -d '{"state":"delete"}' \
  http://ihost.local/api/v1/rest/docker/containers/<FILEBROWSER_CONTAINER_ID>

# 2. Crear el contenedor nuevo
curl -s -X POST -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "addon_id": 9,
    "dev_snd": false,
    "usb_devices": [],
    "port_list": [{"host_port":"8083","container_port":"80"}],
    "network": "bridge",
    "volume_list": [
      {"volume_name":"tailscale-config","mountpoint":"/srv"},
      {"volume_name":"filebrowser-db","mountpoint":"/database"},
      {"volume_name":"filebrowser-config","mountpoint":"/config"}
    ],
    "environments": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
      "UID=0",
      "GID=0"
    ]
  }' \
  http://ihost.local/api/v1/rest/docker/containers
```

Notas de la API:

- `addon_id` es numérico (no el hash).
- `volume_list` usa `mountpoint`, no `addon_mountpoint`.
- `port_list` usa objetos con `host_port` y `container_port`.
- `container_name` **no** se envía: se deriva del add-on.

### Password de Filebrowser

Si la base de Filebrowser es nueva, genera una password aleatoria para `admin` y la imprime en los logs:

```bash
curl -s -H "Authorization: Bearer $IHOST_TOKEN" \
  'http://ihost.local/api/v1/rest/docker/containers/<FILEBROWSER_CONTAINER_ID>/log?tail=200' \
  | jq -r '.data.logs[]' | grep 'randomly generated password'
```

Después del primer login, cambiá la password o detené Filebrowser. No commitees esa password.

### Crear el archivo por la API de Filebrowser

```bash
# Login: devuelve un token JWT
curl -s -X POST http://ihost.local:8083/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<FILEBROWSER_PASSWORD>"}'

# Crear/actualizar serve.json
curl -s -X POST "http://ihost.local:8083/api/resources/serve.json?override=true" \
  -H "X-Auth: <FILEBROWSER_TOKEN>" \
  -H 'Content-Type: text/plain' \
  --data-binary @deploy/tailscale/serve.json

# Verificar
curl -s "http://ihost.local:8083/api/resources/serve.json" \
  -H "X-Auth: <FILEBROWSER_TOKEN>"
```

Al terminar, detener Filebrowser:

```bash
curl -s -X PUT -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' -d '{"state":"stop"}' \
  http://ihost.local/api/v1/rest/docker/containers/<FILEBROWSER_CONTAINER_ID>
```

## 8. Recrear Tailscale con `TS_SERVE_CONFIG`

Se conserva el volumen de estado y se agrega el de configuración:

| Volume | Container path |
|---|---|
| `tailscale` | `/data/tailscalState` |
| `tailscale-config` | `/config` |

- Red: `host`
- Variables: `PATH`, `TS_STATE_DIR=/data/tailscalState`, `TS_SOCKET=/var/run/tailscale/tailscaled.sock`, `TS_SERVE_CONFIG=/config/serve.json`

```bash
# Detener y eliminar el contenedor Tailscale (el volumen de estado persiste)
curl -s -X PUT -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' -d '{"state":"stop"}' \
  http://ihost.local/api/v1/rest/docker/containers/<TAILSCALE_CONTAINER_ID>
curl -s -X PUT -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' -d '{"state":"delete"}' \
  http://ihost.local/api/v1/rest/docker/containers/<TAILSCALE_CONTAINER_ID>

# Crear con el mismo estado + config
curl -s -X POST -H "Authorization: Bearer $IHOST_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "addon_id": 21,
    "dev_snd": false,
    "usb_devices": [],
    "port_list": [],
    "network": "host",
    "volume_list": [
      {"volume_name":"tailscale","mountpoint":"/data/tailscalState"},
      {"volume_name":"tailscale-config","mountpoint":"/config"}
    ],
    "environments": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
      "TS_STATE_DIR=/data/tailscalState",
      "TS_SOCKET=/var/run/tailscale/tailscaled.sock",
      "TS_SERVE_CONFIG=/config/serve.json"
    ]
  }' \
  http://ihost.local/api/v1/rest/docker/containers
```

> **No cambies el volumen de estado ni `TS_STATE_DIR`**: el nodo `ihost` se reutiliza. Si se cambia, Tailscale pierde identidad o crea otro nodo.

## 9. Verificar

1. Logs de Tailscale (`/docker/containers/<id>/log?tail=300`):
   - `serve-config` → configuración aplicada.
   - `Startup complete` → arrancó.
   - `not able to issue TLS certs` → falta **HTTPS Certificates**.
2. Si aparece el error de TLS, habilitar HTTPS Certificates en el admin de Tailscale y reiniciar el contenedor.
3. Abrir `https://ihost.<tailnet>.ts.net` con Tailscale activo en el dispositivo.
4. Verificar `window.isSecureContext === true` y que el manifest responda 200.

## 10. Seguridad

- El token de la API y las passwords **no se guardan en el repo**.
- Usar una sesión temporal de la UI del iHost y cambiarla después.
- No exponer Filebrowser a Internet; detenerlo después de editar.
- No commitear `serve.json` con valores reales si se reemplaza `${TS_CERT_DOMAIN}` por el dominio real.
- No usar `docker exec` para este flujo.

## 11. Rollback

Si algo falla:

1. Detener/eliminar el contenedor Tailscale y recrearlo con el volumen de estado `tailscale` → `/data/tailscalState` y sus variables originales (`TS_STATE_DIR`, `TS_SOCKET`), sin `TS_SERVE_CONFIG`.
2. Detener/eliminar Filebrowser y recrearlo con el volumen de estado/datos que tenía originalmente, o dejarlo detenido.
3. Los volúmenes no se borran al eliminar contenedores; no borres `tailscale` ni `p40la`.

## 12. Referencias

- `docs/ihost.md` — guía de usuario.
- `deploy/tailscale/serve.json` — configuración de Serve.
- `scripts/ihost-pwa-setup.mjs` — automatización con token de sesión.
- SPEC-130 — especificación de la PWA.
- Tailscale Serve: <https://tailscale.com/docs/reference/tailscale-cli/serve>
- Tailscale Docker params: <https://tailscale.com/docs/features/containers/docker/docker-params>
