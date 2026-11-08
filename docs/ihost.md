# Instalación en SONOFF iHost

Esta guía describe cómo instalar el add-on `p40la-ihost` en un SONOFF iHost (eWeLink CUBE) usando Docker.

## Requisitos

- SONOFF iHost con acceso a la UI de Docker.
- Conexión a internet desde el iHost para descargar la imagen.
- Imagen publicada en Docker Hub: `paulomcnally/p40la-ihost`.

## Primer uso

La primera vez que el contenedor arranca sin usuarios registrados, mostrará un wizard de configuración inicial. Debes crear un único usuario administrador con email y contraseña. A partir de ese momento, el acceso requerirá iniciar sesión.

## Instalación paso a paso

1. Abre la aplicación **Docker** en el iHost.
2. Ve a **Container** y pulsa **Add**.
3. En **Image**, escribe: `paulomcnally/p40la-ihost:latest`.
4. En **Container Name**, escribe: `p40la-ihost`.
5. En **Restart Policy**, selecciona `unless-stopped`.
6. Configura el puerto:
   - Host: `8088`
   - Container: `8088`
7. Crea un volumen para persistir datos:
   - Host path: elige o crea una carpeta, por ejemplo `/userdata/p40la-ihost/data`.
   - Container path: `/app/data`
8. (Opcional) Cambia **Network** a `host` si necesitas que el add-on alcance otros servicios locales sin configurar IPs.
9. Pulsa **Done** para iniciar el contenedor.
10. Abre un navegador en `http://<ip-del-ihost>:8088` y completa el wizard.

## Actualización

1. Detén y elimina el contenedor actual.
2. Descarga la nueva imagen `paulomcnally/p40la-ihost:latest`.
3. Vuelve a crear el contenedor usando el **mismo volumen** en `/app/data` para conservar el usuario y la configuración.

## Variables de entorno

| Variable | Valor por defecto | Descripción |
|----------|-------------------|-------------|
| `PORT` | `8088` | Puerto HTTP del servidor |
| `DATA_DIR` | `/app/data` | Directorio de datos persistentes |
| `LOG_LEVEL` | `info` | Nivel de log (`debug`, `info`, `warn`, `error`) |
| `BCRYPT_COST` | `10` | Coste de bcrypt para hash de contraseñas |
| `SESSION_DURATION` | `24h` | Duración de la sesión |
| `SECURE_COOKIE` | `false` | `true` fuerza `Secure` en todos los orígenes (rompe el login por HTTP LAN). Con `false`, la app marca `Secure` solo cuando la request llega por HTTPS/Tailscale. |

## Instalación como PWA (Android e iOS)

La app incluye manifest, íconos y service worker. Para que Chrome Android ofrezca **Instalar app** (y no solo **Crear acceso directo**), la app debe servirse por **HTTPS con certificado confiable**. `http://<ip-del-ihost>:8088` y `http://ihost.local:8088` no son *secure contexts*: el service worker no se registra y la instalación queda deshabilitada. iOS permite **Compartir → Añadir a pantalla de inicio** incluso por HTTP, pero sin HTTPS no hay service worker ni funcionamiento offline.

> **Si ya tenés el add-on oficial `tailscale/TailScale` de iHost:** el nodo `ihost.<tailnet>.ts.net` que ves en el admin de Tailscale es el iHost (UI de eWeLink CUBE), **no** es la app `p40la-ihost`. Ese add-on no publica el puerto `8088` con HTTPS ni permite configurar Serve desde su UI web. Para la PWA se instala **aparte** el contenedor `tailscale/tailscale` con `TS_SERVE_CONFIG` (nodo `p40la-ihost`); no reemplaza al add-on oficial y ambos pueden convivir en el iHost.

### ¿Add-on oficial o contenedor `tailscale/tailscale`?

| | Add-on oficial `tailscale/TailScale` | Contenedor `tailscale/tailscale` |
|---|---|---|
| Cómo se instala | Add-on/Docker → buscar `tailscale/TailScale` → Add | Docker → **+** → `tailscale/tailscale` → Add |
| Seña particular | Botón **Go to TailScale**; nodo `ihost` | Variables `TS_AUTHKEY`, `TS_HOSTNAME`, `TS_SERVE_CONFIG`; nodo definido por vos |
| ¿Publica la app en `:8088`? | **No** (solo la UI del iHost) | **Sí**, con `TS_SERVE_CONFIG` + volumen `/config` |
| ¿Acepta `serve.json`? | **No** | **Sí** |

Si tenés el add-on oficial, no lo borres: sirve para acceder a la UI del iHost. Para la PWA usá el contenedor aparte. Si en cambio ya tenés el contenedor `tailscale/tailscale`, solo falta agregarle el volumen `/config` con `serve.json` y la variable `TS_SERVE_CONFIG=/config/serve.json`.

### HTTPS con Tailscale Serve (todo desde la UI del iHost)

Tailscale publica la app en `https://<nodo>.<tailnet>.ts.net` con certificado Let's Encrypt automático, sin abrir puertos a Internet. En el iHost **no hace falta `docker exec` ni una consola**: la UI de Docker permite definir variables de entorno y volúmenes antes de ejecutar cada add-on, y el `serve.json` se edita con el add-on **Filebrowser**. Tailscale aplica la configuración de Serve de forma declarativa con `TS_SERVE_CONFIG`.

#### 1. Preparar la tailnet

1. En el admin de Tailscale, habilita **MagicDNS** y **HTTPS Certificates** para la tailnet.
2. Crea un **auth key** (Settings → Keys). Guardalo como secreto; no lo commitees.

#### 2. Crear los volúmenes en el iHost

En `http://ihost.local` → **Docker → Volume → Add**:

- `p40la-tailscale-state` — estado del nodo (identidad, claves).
- `p40la-tailscale-config` — configuración de Serve (`serve.json`).

#### 3. Crear `serve.json` con Filebrowser

1. Docker → **+** → buscá `filebrowser/filebrowser` → **Add**.
2. **Run** con esta configuración:
   - **Network**: `bridge`
   - **Port**: host `8081` → container `80` (elegí un puerto libre; no uses el 80 del iHost)
   - **Volumes**:
     | Volume | Container path | Para qué |
     |---|---|---|
     | `p40la-tailscale-config` | `/srv/p40la-tailscale-config` | Ver/editar `serve.json` (obligatorio) |
     | `filebrowser-data` (nuevo) | `/database` | Persistir usuarios/config de Filebrowser (opcional) |
   - **Restart Policy**: `unless-stopped`

   > Si la UI del iHost te obliga a mapear un volumen a `/srv`, creá `filebrowser-data` y mapealo a `/srv`; el volumen `p40la-tailscale-config` va igual a `/srv/p40la-tailscale-config`.
3. Abrí `http://ihost.local:8081` (usuario/password por defecto: `admin` / `admin`) y entrá a la carpeta `p40la-tailscale-config`.
4. Creá el archivo `serve.json` con este contenido (también está en `deploy/tailscale/serve.json`):

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

   - `${TS_CERT_DOMAIN}` lo reemplaza Tailscale automáticamente; no pongas el nombre real.
   - `AllowFunnel: false` mantiene la app privada (solo tailnet).
   - El archivo debe quedar dentro de una **carpeta montada**, no como archivo suelto: Tailscale detecta los cambios del directorio por fsnotify.
5. Guardá el archivo. Podés dejar Filebrowser corriendo o detenerlo; no es necesario para que Tailscale funcione.

#### 4. Ejecutar Tailscale en la UI de Docker

1. Docker → **+** → buscá `tailscale/tailscale` → **Add**.
2. **Run** con esta configuración:
   - **Container Name**: `p40la-tailscale`
   - **Restart Policy**: `unless-stopped`
   - **Network**: `host` (necesario para que Serve llegue a `127.0.0.1:8088`)
   - **Environment Variables**:

     | Variable | Valor |
     |---|---|
     | `TS_AUTHKEY` | tu auth key (`tskey-auth-...`) |
     | `TS_HOSTNAME` | `p40la-ihost` |
     | `TS_STATE_DIR` | `/var/lib/tailscale` |
     | `TS_USERSPACE` | `true` |
     | `TS_AUTH_ONCE` | `true` |
     | `TS_SERVE_CONFIG` | `/config/serve.json` |

   - **Volumes**:

     | Volume | Container path |
     |---|---|
     | `p40la-tailscale-state` | `/var/lib/tailscale` |
     | `p40la-tailscale-config` | `/config` |

3. **Run**. El contenedor hace el login con `TS_AUTHKEY` y aplica `serve.json` al arrancar.

> **Red**: si la UI del iHost no te permite `Network: host`, no uses Tailscale Serve con `TS_SERVE_CONFIG` en bridge: Serve solo puede proxyficar a `127.0.0.1` y no vería la app. La UI del iHost sí ofrece `host` (ver `docs/ihost.md` y la guía de volúmenes de eWeLink CUBE).

#### 5. Verificar

- En el admin de Tailscale → **Machines**, el nodo `p40la-ihost` debe aparecer conectado.
- Abrí `https://p40la-ihost.<tailnet>.ts.net` y verificá que el certificado sea válido (sin warning).
- Si no carga, revisá los **Logs** del add-on `p40la-tailscale` desde la UI de Docker (More info → Logs); no hace falta consola.
- Si cambiás `serve.json`, esperá unos segundos. Si no se aplica, detené y volvé a ejecutar el add-on desde la UI; los volúmenes persisten.

### Instalar en Android

1. Instalá la app de Tailscale, iniciá sesión en la misma tailnet y activá **VPN siempre activa**.
2. Abrí `https://p40la-ihost.<tailnet>.ts.net` en Chrome (no en incógnito).
3. Menú **⋮ → Instalar y crear acceso directo → Instalar**.
4. Verificá que la app abra en ventana standalone y que aparezca en `chrome://webapks`.

Si **Instalar** sigue deshabilitado, revisá en DevTools → Application → Manifest → **Installability** que no haya errores (`not-from-secure-origin`, `no-manifest`, `manifest-missing-suitable-icon`, etc.) y que el manifest y los íconos sean públicos.

### Instalar en iOS

1. Instalá la app de Tailscale, iniciá sesión y activá el perfil VPN.
2. Abrí la URL en **Safari** (recomendado para controlar el modo standalone).
3. Tocá **Compartir → Añadir a pantalla de inicio**.
4. Dejá activado **Abrir como app web** y tocá **Agregar**.

En Chrome o Edge para iOS el camino también está en el menú Compartir, pero no muestran el toggle de Safari. El almacenamiento de la web app en iOS es separado de Safari: si no estabas logueado al instalarla, iniciá sesión dentro de la app.

### Notas y diagnóstico

- La PWA queda atada al origen `https://p40la-ihost.<tailnet>.ts.net`. **No uses `https://ihost.<tailnet>.ts.net`**: ese es el nodo del add-on oficial del iHost, no la app. Tampoco la instales desde la IP ni desde `ihost.local`; si renombrás el nodo, hay que reinstalarla.
- Sin Tailscale activo, el nombre `*.ts.net` no resuelve y la PWA no abre. `http://ihost.local:8088` sigue disponible como acceso LAN de respaldo para escritorio.
- El add-on oficial `tailscale/TailScale` y el contenedor `tailscale/tailscale` pueden convivir: el primero te da acceso remoto a la UI del iHost; el segundo publica la app con Serve.
- El service worker no cachea `/api/*` ni PDFs; solo el shell, los assets con hash, los íconos y el i18n.
- Para verificar el service worker y el manifest: DevTools → Application → Service Workers / Manifest. Lighthouse ya no tiene categoría PWA.
- Si Tailscale no emite el certificado, revisá que MagicDNS y HTTPS Certificates estén habilitados y que el nodo esté conectado.

## Solución de problemas

- Si el puerto `8088` está ocupado, cambia el mapeo de puertos en el host o usa la variable `PORT`.
- Si los datos no persisten, verifica que el volumen esté montado en `/app/data`.
- Para debug, usa la variable `LOG_LEVEL=debug`.
