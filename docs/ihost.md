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

### HTTPS con Tailscale Serve

Tailscale publica la app en `https://<nodo>.<tailnet>.ts.net` con certificado Let's Encrypt automático, sin abrir puertos a Internet. Requiere una cuenta de Tailscale y la app instalada en los dispositivos que vayan a usar la PWA.

1. En el admin de Tailscale, habilita **MagicDNS** y **HTTPS Certificates** para la tailnet.
2. Crea un **auth key** (Settings → Keys). Guardalo como secreto; no lo commitees.
3. Ejecuta el contenedor de Tailscale en el iHost. Ejemplo de Docker Compose:

   ```yaml
   services:
     tailscale:
       image: tailscale/tailscale:stable
       container_name: p40la-tailscale
       restart: unless-stopped
       network_mode: host
       environment:
         - TS_AUTHKEY=${TS_AUTHKEY}
         - TS_HOSTNAME=p40la-ihost
         - TS_STATE_DIR=/var/lib/tailscale
         - TS_USERSPACE=true
         - TS_AUTH_ONCE=true
       volumes:
         - tailscale-state:/var/lib/tailscale

   volumes:
     tailscale-state:
   ```

   - `TS_USERSPACE=true` evita depender de `/dev/net/tun` (si el iHost lo expone, podés usar modo kernel con `cap_add: [NET_ADMIN]` y el dispositivo `/dev/net/tun`).
   - `network_mode: host` permite que Serve llegue a `127.0.0.1:8088` (la app ya publica ese puerto en el host). También podés usar `network_mode: service:p40la-ihost`.
   - El estado del nodo queda en el volumen `tailscale-state`; no borres ese volumen sin querer.

4. Verifica que el nodo esté conectado:

   ```bash
   docker exec p40la-tailscale tailscale status
   ```

5. Configura Serve una vez (con `--bg` se reanuda solo tras reiniciar el contenedor o el iHost):

   ```bash
   docker exec p40la-tailscale tailscale serve --bg --https=443 127.0.0.1:8088
   docker exec p40la-tailscale tailscale serve status
   ```

   Alternativa declarativa: usar la variable `TS_SERVE_CONFIG` con la configuración de Serve.

6. Abre `https://p40la-ihost.<tailnet>.ts.net` y verificá que el certificado sea válido (sin warning).

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

- La PWA queda atada al origen `https://<nodo>.<tailnet>.ts.net`. No la instales desde la IP ni desde `ihost.local`; si renombrás el nodo, hay que reinstalarla.
- Sin Tailscale activo, el nombre `*.ts.net` no resuelve y la PWA no abre. `http://ihost.local:8088` sigue disponible como acceso LAN de respaldo para escritorio.
- El service worker no cachea `/api/*` ni PDFs; solo el shell, los assets con hash, los íconos y el i18n.
- Para verificar el service worker y el manifest: DevTools → Application → Service Workers / Manifest. Lighthouse ya no tiene categoría PWA.
- Si Tailscale no emite el certificado, revisá que MagicDNS y HTTPS Certificates estén habilitados y que el nodo esté conectado.

## Solución de problemas

- Si el puerto `8088` está ocupado, cambia el mapeo de puertos en el host o usa la variable `PORT`.
- Si los datos no persisten, verifica que el volumen esté montado en `/app/data`.
- Para debug, usa la variable `LOG_LEVEL=debug`.
