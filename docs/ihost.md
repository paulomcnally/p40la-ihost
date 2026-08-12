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
   - Host: `8000`
   - Container: `8000`
7. Crea un volumen para persistir datos:
   - Host path: elige o crea una carpeta, por ejemplo `/userdata/p40la-ihost/data`.
   - Container path: `/app/data`
8. (Opcional) Cambia **Network** a `host` si necesitas que el add-on alcance otros servicios locales sin configurar IPs.
9. Pulsa **Done** para iniciar el contenedor.
10. Abre un navegador en `http://<ip-del-ihost>:8000` y completa el wizard.

## Actualización

1. Detén y elimina el contenedor actual.
2. Descarga la nueva imagen `paulomcnally/p40la-ihost:latest`.
3. Vuelve a crear el contenedor usando el **mismo volumen** en `/app/data` para conservar el usuario y la configuración.

## Variables de entorno

| Variable | Valor por defecto | Descripción |
|----------|-------------------|-------------|
| `PORT` | `8000` | Puerto HTTP del servidor |
| `DATA_DIR` | `/app/data` | Directorio de datos persistentes |
| `LOG_LEVEL` | `info` | Nivel de log (`debug`, `info`, `warn`, `error`) |
| `BCRYPT_COST` | `10` | Coste de bcrypt para hash de contraseñas |
| `SESSION_DURATION` | `24h` | Duración de la sesión |
| `SECURE_COOKIE` | `false` | `true` para marcar cookies como Secure (requiere HTTPS) |

## Solución de problemas

- Si el puerto `8000` está ocupado, cambia el mapeo de puertos en el host.
- Si los datos no persisten, verifica que el volumen esté montado en `/app/data`.
- Para debug, usa la variable `LOG_LEVEL=debug`.
