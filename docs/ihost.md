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

> **Si ya tenés Tailscale corriendo en el iHost:** fijate si el contenedor es **`tailscale/tailscale`** (el que usa `TS_STATE_DIR`/`TS_SOCKET`) o el **add-on de eWeLink** con el botón **Go to TailScale**.
>
> - Si es **`tailscale/tailscale`**, **sí** acepta `TS_SERVE_CONFIG`: reutilizá su volumen de estado (por ejemplo `tailscale` → `/data/tailscalState`), agregá un volumen `tailscale-config` → `/config` con `serve.json` y la variable `TS_SERVE_CONFIG=/config/serve.json`. La URL queda en el nodo existente: `https://ihost.<tailnet>.ts.net`.
> - Si es el **add-on de eWeLink** y no permite agregar variables/volúmenes, usá un contenedor `tailscale/tailscale` **aparte** con `TS_HOSTNAME=p40la-ihost`; la URL será `https://p40la-ihost.<tailnet>.ts.net`.

### ¿Add-on oficial o contenedor `tailscale/tailscale`?

| | Add-on de eWeLink (`tailscale/TailScale` con Go to TailScale) | Contenedor `tailscale/tailscale` |
|---|---|---|
| Seña particular | Botón **Go to TailScale**; UI propia de eWeLink | Variables `TS_STATE_DIR`, `TS_SOCKET`, `TS_SERVE_CONFIG`; imagen oficial de Tailscale |
| ¿Publica la app en `:8088`? | **No por defecto** (solo la UI del iHost) | **Sí**, con `TS_SERVE_CONFIG` + volumen `/config` |
| ¿Acepta `serve.json`? | **No documentado**; se puede intentar si la UI permite agregar `TS_SERVE_CONFIG` y un volumen `/config` | **Sí** |
| Nodo/URL | `ihost.<tailnet>.ts.net` | El nodo existente (`ihost...`) o uno nuevo (`p40la-ihost...`) |

Si tenés el add-on oficial, no lo borres: sirve para acceder a la UI del iHost. Si en tu add-on ves un volumen como `tailscale` → `/data/tailscalState` y `TS_STATE_DIR=/data/tailscalState`, **no cambies ese volumen ni esa variable**: agregá un volumen nuevo para `/config` y la variable `TS_SERVE_CONFIG`. Si el add-on no aplica la configuración, usá el contenedor aparte. Si ya tenés el contenedor `tailscale/tailscale`, solo falta agregarle el volumen `/config` con `serve.json` y la variable `TS_SERVE_CONFIG=/config/serve.json`.

#### Si tu add-on oficial ya usa `tailscale` → `/data/tailscalState`

No cambies el volumen de estado ni `TS_STATE_DIR`. Para intentar publicar la app:

1. Creá un volumen nuevo para la configuración, por ejemplo `tailscale-config`.
2. Con Filebrowser, montá `tailscale-config` → `/srv/tailscale-config` y creá ahí `serve.json`.
3. En el add-on Tailscale, agregá:
   - **Volumen**: `tailscale-config` → `/config`
   - **Variable**: `TS_SERVE_CONFIG=/config/serve.json`
   - **Network**: `host`
4. Reiniciá el add-on usando **el mismo volumen de estado** `tailscale` → `/data/tailscalState`.
5. Probá `https://ihost.<tailnet>.ts.net`.

Si el add-on no aplica la configuración (los logs no muestran Serve o la URL sigue mostrando la UI del iHost), no insistas: instalá el contenedor `tailscale/tailscale` aparte con su propio nodo.

### HTTPS con Tailscale Serve (todo desde la UI del iHost)

Tailscale publica la app en `https://<nodo>.<tailnet>.ts.net` con certificado Let's Encrypt automático, sin abrir puertos a Internet. En el iHost **no hace falta `docker exec` ni una consola**: la UI de Docker permite definir variables de entorno y volúmenes antes de ejecutar cada add-on, y el `serve.json` se edita con el add-on **Filebrowser**. Tailscale aplica la configuración de Serve de forma declarativa con `TS_SERVE_CONFIG`.

#### 1. Preparar la tailnet

1. En el admin de Tailscale, habilita **MagicDNS** y **HTTPS Certificates** para la tailnet.
2. Crea un **auth key** (Settings → Keys). Guardalo como secreto; no lo commitees.

#### 2. Crear o identificar los volúmenes en el iHost

> **Los nombres son ejemplos, no son obligatorios.** Si ya tenés Tailscale configurado, **no crees un volumen de estado nuevo**: usá el que ya tiene el contenedor. Para `serve.json` podés crear un volumen nuevo o reutilizar uno existente; lo importante es que Filebrowser y Tailscale monten **el mismo volumen** y que `TS_SERVE_CONFIG` apunte al archivo.

En `http://ihost.local` → **Docker → Volume**:

- **Estado**: usá el volumen que ya tiene tu Tailscale (por ejemplo, el que está montado en `/var/lib/tailscale` o `/data/tailscalState`). Si estás instalando desde cero, creá uno, por ejemplo `p40la-tailscale-state`.
- **Configuración de Serve**: creá un volumen nuevo, por ejemplo `p40la-tailscale-config`, o usá uno existente que puedas montar en `/config`. Ese volumen contendrá `serve.json`.

#### 3. Crear `serve.json` con Filebrowser

1. Docker → **+** → buscá `filebrowser/filebrowser` → **Add**.
2. **Run** con esta configuración:
   - **Network**: `bridge`
   - **Port**: host `8083` → container `80` (elegí un puerto libre; no uses el 80 del iHost. Si `8081` está ocupado por otro add-on, como `presence-ihost`, usá `8083`.)
   - **Volumes**:
     | Volume | Container path | Para qué |
     |---|---|---|
     | `tailscale-config` (o el nombre real que uses) | `/srv` | Ver/editar `serve.json` (obligatorio; es la raíz de Filebrowser) |
     | `filebrowser-data` (nuevo) | `/database` | Persistir usuarios/config de Filebrowser (opcional) |
   - **Restart Policy**: `unless-stopped`

   > **Importante**: el volumen de configuración va montado en `/srv` (la raíz de Filebrowser), no en un subdirectorio. Así `serve.json` queda en la raíz del volumen y Tailscale lo ve en `/config/serve.json` al montar el mismo volumen en `/config`. No uses el volumen de estado `tailscale` para Filebrowser.
   >
   > Si en la UI ves una fila `/srv` **sin nombre de volumen**, es un volumen anónimo vacío: seleccioná explícitamente `tailscale-config` en esa fila. Si ves `tailscale:/data/tailscalState` dentro de Filebrowser, está mal: ese es el estado de Tailscale, no la configuración.
   >
   > Filebrowser corre como UID/GID `1000`; si el volumen nuevo es propiedad de root y no te deja crear `serve.json`, cambiá temporalmente `UID=0` y `GID=0` en las variables de Filebrowser, o creá el archivo con el add-on SSH.
3. Abrí `http://ihost.local:8083`. Si la base de Filebrowser es nueva, puede generar una **password aleatoria** para `admin`; mirá los **Logs** del add-on. Si ya tenías una base de Filebrowser, usá tu password. Deberías ver la raíz del volumen `tailscale-config` vacía o con `serve.json`.
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

> **Seguridad**: Filebrowser viene con `admin`/`admin`. Cambiá la contraseña desde su configuración o detené el add-on después de editar `serve.json`. No lo expongas a Internet; solo debe ser accesible en la LAN.
>
> **Si Filebrowser no carga, no muestra nada o dice `Forbidden`**:
> - Verificá que el contenedor esté **running** y que el puerto host (`8081`) no esté ocupado por otro add-on.
> - El volumen de configuración debe estar montado en **`/srv`**, no en `/srv/algo`.
> - No montes el volumen de estado `tailscale` (`/data/tailscalState`) en Filebrowser: es de Tailscale y suele ser propiedad de root con permisos restringidos (causa `Forbidden`).
> - Filebrowser corre como UID/GID `1000`; si el volumen nuevo es de root y no te deja escribir, cambiá temporalmente `UID=0` y `GID=0` en las variables de Filebrowser, o creá el archivo con el add-on SSH.
> - Si la **UI del iHost** dice `Forbidden` al guardar los volúmenes, seguramente el contenedor está corriendo: detenelo y volvé a ejecutarlo. Si sigue, eliminá el contenedor Filebrowser (no los volúmenes) y recrealo con el mapeo correcto.
> - Revisá los **Logs** del add-on Filebrowser desde la UI de Docker. Si dice error de base de datos, agregá un volumen `filebrowser-data` → `/database`.
> - Si la página abre pero el login `admin`/`admin` falla, revisá los logs o recreá el volumen `filebrowser-data`.
>
> Si `serve.json` no aparece en Tailscale, revisá que `TS_SERVE_CONFIG=/config/serve.json` esté en el contenedor Tailscale y que el mismo volumen esté montado como directorio en `/config`.

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
     | el volumen de estado que ya usa tu Tailscale (ej. `p40la-tailscale-state`) | `/var/lib/tailscale` |
     | el volumen de configuración con `serve.json` (ej. `p40la-tailscale-config`) | `/config` |

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

- La PWA queda atada al nodo que tenga `TS_SERVE_CONFIG`: `https://ihost.<tailnet>.ts.net` si reutilizás el nodo existente, o `https://p40la-ihost.<tailnet>.ts.net` si usás un contenedor aparte. No la instales desde la IP ni desde `ihost.local`; si renombrás el nodo, hay que reinstalarla.
- Sin Tailscale activo, el nombre `*.ts.net` no resuelve y la PWA no abre. `http://ihost.local:8088` sigue disponible como acceso LAN de respaldo para escritorio.
- El service worker no cachea `/api/*` ni PDFs; solo el shell, los assets con hash, los íconos y el i18n.
- Para verificar el service worker y el manifest: DevTools → Application → Service Workers / Manifest. Lighthouse ya no tiene categoría PWA.
- Si Tailscale no emite el certificado, revisá que MagicDNS y **HTTPS Certificates** estén habilitados y que el nodo esté conectado. En los logs del add-on Tailscale, `serve proxy: ... not able to issue TLS certs` significa que falta habilitar HTTPS Certificates; después de habilitarlo, reiniciá el add-on.

## Solución de problemas

- Si el puerto `8088` está ocupado, cambia el mapeo de puertos en el host o usa la variable `PORT`.
- Si los datos no persisten, verifica que el volumen esté montado en `/app/data`.
- Para debug, usa la variable `LOG_LEVEL=debug`.
