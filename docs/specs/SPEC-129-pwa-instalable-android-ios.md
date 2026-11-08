---
title: "PWA instalable en Android e iOS: HTTPS, manifest, íconos y service worker"
id: "SPEC-129"
status: "in_progress"
author: "opencode"
created: "2026-11-08"
updated: "2026-11-08"
github_issue: 133
---

# PWA instalable en Android e iOS: HTTPS, manifest, íconos y service worker

**ID**: SPEC-129  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-08  
**Actualizado**: 2026-11-08

---

## 1. Resumen Ejecutivo

La app se abre hoy desde Android en `http://ihost.local:8088` (o por IP). Chrome muestra el menú **"Install and create shortcut"**, pero la acción **Install** aparece deshabilitada: solo queda **Create shortcut**. La causa no es un detalle del manifest ni del ícono: el origen `http://` en una IP/hostname de LAN **no es un *secure context***, y Chrome Android exige HTTPS con un certificado confiable para instalar una PWA (WebAPK). A esto se suma que el proyecto **no tiene manifest, ni íconos 192/512, ni service worker, ni meta tags de iOS**.

La auditoría de `main` confirma el estado: el backend sirve HTTP plano (`cmd/server/main.go:244-246`), no hay TLS ni reverse proxy, el frontend no declara `<link rel="manifest">` (`frontend/index.html`), en `frontend/public/` solo existen `favicon.svg` y un `apple-touch-icon.png` de 180×180, y no hay registro de service worker en `frontend/src/`. El fallback SPA (`internal/api/routes.go:312-322`) devuelve `index.html` con 200 para archivos faltantes, lo que puede ocultar errores de manifest/SW con MIME incorrecto.

En iOS el panorama es distinto pero complementario: **no existe "Install app"**: la instalación es manual con **Compartir → Añadir a pantalla de inicio** (Safari, y también Chrome/Edge desde iOS 16.4). iOS 26 abre por defecto cualquier sitio agregado como web app, incluso sin manifest ni service worker. Sin embargo, **service worker, caché offline y push sí requieren secure context**, así que HTTPS también es necesario para una PWA de calidad en iOS. La solución de fondo es la misma para ambas plataformas: **un origen HTTPS estable, manifest público, íconos correctos y un service worker liviano**.

Esta spec es de **investigación y definición**: documenta qué falta y propone un plan de implementación. El camino de HTTPS ya fue decidido: **Tailscale Serve**, con certificado automático `*.ts.net`, acceso solo por tailnet y sin abrir puertos a Internet. No agrega dependencias de frontend (manifest y SW hand-written), no toca base de datos ni migraciones, y el impacto en el iHost se limita al contenedor de Tailscale (~50-80 MB) más el estado persistente de la tailnet.

Consideraciones de UI obligatorias:
- Todo `input`/`select`/`textarea` que se agregue (por ejemplo, un formulario de instrucciones o configuración) DEBE usar los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`). Jamás colores hardcodeados (`bg-white`, `dark:bg-[#2c2c2e]`, `text-black`).
- Todo formulario o modal nuevo debe verificarse en darkmode: texto y placeholders legibles.
- Si se agrega un modal de instrucciones de instalación, **no se cierra al hacer click fuera**; el overlay bloquea el fondo, el cierre es solo con **Cancelar/Cerrar/Listo** o la **X** (visible en todos los estados) y se respeta la escala de capas (`z-[60]`/`z-[70]`/`z-[80]`/`z-[90]`/`z-[100]`). Ver SPEC-122, SPEC-126 y `docs/project-rules.md` §4.7.
- Si se agrega una página de detalle o formulario con lista padre, el retorno va en la flecha atrás del header vía `BACK_ROUTES`; nunca links "← Título" dentro del contenido (SPEC-063).
- Todo selector de opciones usa el componente `Select`, nunca `<select>` nativo (SPEC-118); el build falla con `frontend/scripts/check-ui-rules.cjs` si se agrega uno.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Origen seguro (HTTPS)**: la app debe servirse en producción desde un origen **HTTPS con certificado confiable** por Android/iOS. No alcanza un certificado autofirmado sin CA instalada, ni `http://` en IP privada, ni `http://*.local`. `localhost` es la única excepción HTTP y no aplica al teléfono. Este es el bloqueador principal de "Install" en Chrome Android. **Camino decidido: Tailscale Serve**, que publica la app en `https://<nodo>.<tailnet>.ts.net` con certificado Let's Encrypt automático y sin exponer puertos a Internet (ver ADR-001 y §4.2.7).
2. **REQ-002 — Manifest público y válido**: crear `frontend/public/manifest.webmanifest` servido same-origin, sin autenticación, con `id`, `name`, `short_name`, `start_url`, `scope`, `display`, `theme_color`, `background_color`, `prefer_related_applications: false` e `icons`. El manifest debe ser parseable y devolver 200 con `Content-Type: application/manifest+json`.
3. **REQ-003 — Íconos instalables**: generar y declarar PNG reales 192×192 (`purpose: "any"`), 512×512 (`purpose: "any"`) y 512×512 (`purpose: "maskable"`, con safe zone del 40% central). No usar el mismo archivo para `any` y `maskable`. Los íconos deben ser públicos y descargables sin login.
4. **REQ-004 — Íconos y meta tags de iOS**: mantener/generar `apple-touch-icon` PNG 180×180 (y variantes 167/152 para iPad), declarar `apple-mobile-web-app-capable`, `apple-mobile-web-app-title`, `theme-color` y `viewport-fit=cover`. El ícono de iOS tiene prioridad sobre los íconos del manifest.
5. **REQ-005 — `<link rel="manifest">` y metadatos en `index.html`**: el HTML de entrada (`frontend/index.html`, y por lo tanto el build en `public/index.html`) debe declarar el manifest, `theme-color`, los meta tags de iOS y el viewport con `viewport-fit=cover`.
6. **REQ-006 — MIME y rutas explícitas de PWA**: registrar `.webmanifest` como `application/manifest+json` (la tabla MIME de Go y la imagen distroless no lo incluyen) y exponer `/manifest.webmanifest` y `/sw.js` con rutas explícitas para que un archivo faltante devuelva 404 y no `index.html` con 200. El service worker debe servirse como JavaScript válido (`text/javascript`).
7. **REQ-007 — Service worker en scope raíz**: crear `frontend/public/sw.js` (scope `/`) y registrarlo solo en producción desde `frontend/src/main.tsx`. Estrategia: navegación network-first con fallback al app shell cacheado; `/assets/*` cache-first con cache runtime; `/i18n/*` cache-first o stale-while-revalidate; **`/api/*` siempre red y nunca caché**; PDFs y descargas nunca se cachean. Versión de caché explícita y limpieza de cachés viejas en `activate`. El service worker no es requisito para instalar en Chrome 108+ (Android) / 112+ (desktop), pero es necesario para offline, push y una PWA de calidad, y en iOS es la única forma de offline.
8. **REQ-008 — Headers de caché correctos**: `Cache-Control: no-cache` (o `max-age=0, must-revalidate`) para `/sw.js`, `/manifest.webmanifest` e `index.html`; `Cache-Control: public, max-age=31536000, immutable` para `/assets/*` con hash. Sin esto Chrome puede quedar pegado a un SW o manifest viejo.
9. **REQ-009 — UI de instalación en Android**: capturar `beforeinstallprompt` (con `preventDefault()`), mostrar una acción **"Instalar app"** (card en Settings y/o banner discreto en Home), ejecutar `prompt()` y ocultar la acción tras `appinstalled` o si la app ya corre en `display-mode: standalone`. No bloquear la app si el evento no existe (Chrome sin GMS, navegador no compatible, ya instalada).
10. **REQ-010 — Instrucciones de instalación en iOS**: como iOS no expone `beforeinstallprompt`, mostrar instrucciones específicas: **Safari → Compartir → Añadir a pantalla de inicio → "Abrir como app web" activado**. En Chrome/Edge iOS indicar el mismo camino desde su menú Compartir. No prometer un botón de instalación que iOS no tiene.
11. **REQ-011 — Cookies y sesión bajo HTTPS**: la cookie de sesión debe marcarse `Secure` cuando la request llegue por HTTPS (TLS directo o `X-Forwarded-Proto: https` confiable desde Tailscale) y **no** marcarse en HTTP LAN para no romper el acceso de respaldo. Mantener `HttpOnly`, `Path=/` y `Max-Age`/`Expires`; revisar `SameSite=Strict` en modo standalone (hay reportes de pérdida de sesión en WebAPK; si falla, evaluar `Lax`). Si se prefiere `SECURE_COOKIE=true` fijo, documentar que el login por HTTP LAN deja de funcionar. La PWA comparte el perfil de Chrome en Android, pero en iOS el almacenamiento es separado de Safari: el usuario debe poder loguearse dentro de la app instalada.
12. **REQ-012 — Origen canónico estable**: el origen canónico de la PWA es `https://<nodo>.<tailnet>.ts.net`; no instalar desde IP, `ihost.local` ni variantes. La PWA queda atada al origen: cambiar de hostname obliga a reinstalar y pierde cookies/sesiones. `http://ihost.local:8088` queda como acceso LAN de respaldo, sin redirect global; la PWA solo se instala desde HTTPS.
13. **REQ-013 — Documentación de despliegue**: actualizar `docs/ihost.md` con Tailscale Serve (auth key, estado persistente, `serve`, ACL, clientes Android/iOS) y una guía corta de instalación en Android e iOS, incluyendo cómo verificar `chrome://webapks` y el estado standalone.
14. **REQ-014 — Sin regresiones**: el health check `GET /health` debe seguir respondiendo `{"status":"ok"}` por HTTP interno para Docker, el login/logout debe funcionar tanto por HTTP (transición) como por HTTPS, y la app debe seguir funcionando si el service worker no está disponible (navegador viejo, modo privado, HTTP).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-015 — Página offline**: cachear un fallback de app shell (mismo `index.html`) y, opcionalmente, una vista simple "Sin conexión" con reintento. No cachear datos financieros ni respuestas de API.
2. **REQ-016 — Actualización de la PWA**: al detectar un `sw.js` nuevo, aplicar `skipWaiting()` + `clients.claim()` y mostrar un aviso discreto de "Nueva versión disponible" con acción de recarga (sin modal bloqueante).
3. **REQ-017 — Instalación con un click desde Settings**: card "Aplicación" en `SettingsPage` con estado (instalada / instalable / instrucciones iOS), botón "Instalar app" en Android y guía en iOS. Si se usa un modal de instrucciones, cumplir la regla de modales.
4. **REQ-018 — Detección de plataforma**: no mostrar el botón de Android en iOS ni las instrucciones de iOS en Android; detectar `display-mode: standalone` y `navigator.standalone` para no repetir la promoción.
5. **REQ-019 — Manifest enriquecido**: agregar `description`, `lang: "es"`, `dir: "ltr"` y, si se desea, `screenshots` para el diálogo de instalación. No agregar `shortcuts` ni `share_target` en esta etapa.
6. **REQ-020 — Splash de iOS (opcional)**: si se implementa, usar `apple-touch-startup-image` con media queries exactas; documentar que `background_color` es ignorado por iOS. Es costoso en cantidad de archivos, por lo que queda como P1/P2.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-021 — Notificaciones push**: fuera de alcance; iOS requiere app instalada y permiso explícito, Android usa FCM. No implementar en esta spec.
2. **REQ-022 — Background Sync**: iOS no lo soporta; si se necesita, implementar cola en IndexedDB y flush al volver a foreground.
3. **REQ-023 — Badging API y atajos**: evaluar en una spec futura.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el manifest y los íconos son estáticos y no agregan RAM. El service worker no debe agregar latencia perceptible a `/api/*` (siempre red). El bundle no debe crecer por dependencias nuevas: manifest, SW y lógica de instalación se escriben a mano.
- **iHost**: el contenedor de Tailscale debe ser multi-arch (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) y liviano. Presupuesto orientativo: ~50-80 MB de RAM en modo userspace, más un volumen pequeño para el estado del nodo. La app ya corre en 2/4 GB; medir y evitar duplicar procesos. Si el iHost no expone `/dev/net/tun`, usar `TS_USERSPACE=true`.
- **Seguridad**: el `TS_AUTHKEY` de Tailscale es un secreto y va por secreto de Docker/variable de entorno, nunca al repo; rotarlo si se filtra. El certificado lo gestiona Tailscale automáticamente. No habilitar HSTS hasta validar. No cachear `/api/*`, PDFs ni datos financieros. No commitear certificados ni claves privadas.
- **Almacenamiento**: los íconos suman pocos KB/MB; el caché del SW debe acotarse (app shell + assets hasheados + i18n) y limpiarse por versión.
- **Disponibilidad**: `/health` sigue siendo HTTP interno para Docker; el acceso de usuarios pasa por HTTPS. Si se cae el terminador TLS, la app no es accesible desde la PWA (documentar recuperación).
- **Compatibilidad**: Chrome Android actual, Safari iOS 16.4+ (ideal iOS 26), Chrome/Edge iOS. La app debe seguir siendo usable en navegador de escritorio y por HTTP durante la transición.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

**Estado actual del proyecto (auditoría de `main`, 2026-11-08)**

| Área | Hallazgo | Referencia |
|---|---|---|
| Servido | HTTP plano, sin TLS, sin reverse proxy | `cmd/server/main.go:244-246` |
| Estáticos | `http.FileServer` para `/assets` e `/i18n`; SPA fallback para todo lo demás | `internal/api/routes.go:298-322` |
| Fallback | Archivo faltante → `index.html` con 200 (puede enmascarar SW/manifest rotos) | `internal/api/routes.go:312-322` |
| Manifest | No existe | `grep` sin resultados |
| Service worker | No existe ni se registra | `frontend/src/` sin `serviceWorker` |
| Iconos | `favicon.svg` (193 B) y `apple-touch-icon.png` 180×180 16-bit RGBA | `frontend/public/` |
| HTML | Sin `<link rel="manifest">`, sin `theme-color`, sin meta iOS | `frontend/index.html` |
| Cookies | `SECURE_COOKIE=false` por defecto | `internal/config/config.go:44-47` |
| Build | Vite `outDir: ../public`, `emptyOutDir: true` | `frontend/vite.config.ts:7-16` |
| Runtime | distroless static, sin `mime.types`; `.webmanifest` no está en la tabla de Go | `Dockerfile:36` |
| Red | Solo `8088:8088`; sin dominio, DDNS, VPN ni tunnel | `docker-compose.yml:8-9` |
| Docs | Acceso asumido por `http://<ip>:8088` | `docs/ihost.md:30` |

**Requisitos de Chrome Android (2026)**

- **Secure context obligatorio**: HTTPS con certificado válido; solo `localhost`/`127.0.0.1`/`::1` se exceptúan. `http://192.168.x.x`, `http://10.x.x.x` y `http://*.local` **no** califican. El flag `--unsafely-treat-insecure-origin-as-secure` es solo de desarrollo y no es una solución para usuarios.
- **Manifest**: `<link rel="manifest">` presente, parseable, same-origin, `name` o `short_name`, `start_url` válido y same-origin, `display` distinto de `browser`, ícono `any` de al menos 144 px (la documentación recomienda 192 y 512), `prefer_related_applications` ausente o `false`.
- **Service worker**: desde Chrome 108 (Android) / 112 (desktop) **ya no es obligatorio** para instalar desde el menú. Sí es necesario para offline, push y cache. Un SW servido como HTML o con MIME incorrecto es rechazado.
- **"Install" vs "Create shortcut"**: Chrome 150 (2026) muestra **"Install and create shortcut"** con dos acciones. Si la app no es instalable, **Install queda deshabilitado** y solo queda Create shortcut. Install crea un **WebAPK** (aparece en el app drawer y en `chrome://webapks`, ventana standalone, splash, link capturing); Create shortcut crea un bookmark con badge de Chrome. Sin Google Play Services, Chrome puede caer a shortcut aunque el manifest sea válido.
- **Verificación**: DevTools → **Application → Manifest → Installability** muestra los errores exactos (`not-from-secure-origin`, `no-manifest`, `manifest-missing-suitable-icon`, etc.). Lighthouse ya no tiene categoría PWA (Lighthouse 12+); no usarlo como criterio.

**Requisitos de iOS (2026)**

- **No existe "Install app"**: el único mecanismo es **Compartir → Añadir a pantalla de inicio**. No hay `beforeinstallprompt`, `appinstalled` ni API de instalación.
- **Safari**: en iOS 26 el toggle **"Abrir como app web"** viene activado por defecto y **cualquier sitio** agregado abre standalone, aunque no tenga manifest ni SW. Si se desactiva, abre como bookmark en Safari.
- **Chrome/Edge iOS**: desde iOS 16.4 ofrecen "Añadir a pantalla de inicio" en su Share, pero corren sobre WebKit y no muestran el toggle de Safari. Para controlar el modo standalone, la vía recomendada es Safari.
- **Manifest/meta**: iOS lee `name`, `short_name`, `start_url`, `scope`, `id`, `display: standalone` e icons PNG; ignora `background_color`, `orientation`, `maskable`, `shortcuts` y `share_target`. `apple-touch-icon` (PNG, 180×180; 167/152 para iPad) **tiene prioridad** sobre los icons del manifest. `apple-mobile-web-app-capable` sigue siendo necesario para iOS 16.4–18 y para habilitar splash.
- **HTTPS**: "Añadir a pantalla de inicio" **no exige HTTPS** en iOS 26, pero **service worker, Cache API y push sí requieren secure context**. En HTTP la app instalada queda online-only.
- **Cookies**: desde iOS 17.2 se copian las cookies de Safari al crear la web app (incluido el login); después el almacenamiento es **separado** y no se sincroniza. Conviene cookie `HttpOnly` persistente y probar instalar logueado y deslogueado.

**Opciones de HTTPS evaluadas para el iHost**

| Opción | Cert confiable | Expone a Internet | Depende de Internet | RAM extra aprox. | Complejidad | Veredicto |
|---|---|---|---|---|---|---|
| Dominio propio + Let's Encrypt DNS-01 + Caddy | Sí | No (A → IP LAN) | Solo al renovar | 35-60 MB | Media | ⚠️ Alternativa si se quiere LAN sin VPN |
| Tailscale Serve (`*.ts.net`) | Sí | No (solo tailnet) | Sí (control plane/DNS) | 50-80 MB | Media | ✅ **Seleccionada** |
| Cloudflare Tunnel + dominio | Sí | Sí (edge; Access opcional) | Sí, siempre | 30-60 MB | Media | ⚠️ Solo si se quiere acceso remoto |
| TLS nativo en Go con certs provistos | Sí | No | Solo al renovar | 0-10 MB | Media | ⚠️ Alternativa de mínima huella |
| CA local / mkcert instalada en el dispositivo | Sí, con CA instalada | No | No | 0 | Alta | ❌ Frágil para producción |
| Self-signed sin CA instalada | No | No | No | 0 | Baja | ❌ Chrome rechaza |
| Flag de Chrome / ADB | No real | N/A | No | 0 | Alta | ❌ No viable |

**Notas de la comparación**

- **Tailscale Serve (seleccionada)**: certificado Let's Encrypt automático para `https://<nodo>.<tailnet>.ts.net`, sin abrir puertos. Se configura **desde la UI del iHost** con `TS_SERVE_CONFIG=/config/serve.json` y un volumen `/config` (directorio) editado con Filebrowser; **no requiere `docker exec`**. Serve solo puede proxificar a `http://127.0.0.1`, por lo que el contenedor usa `Network: host` y `TS_USERSPACE=true` si no hay `/dev/net/tun`. Requiere Tailscale instalado y activo en cada dispositivo; el nombre del nodo queda en los logs de CT. La imagen publica `arm/v7` y `arm64`.
- **Caddy + DNS-01 (alternativa)**: el registro A puede apuntar a la IP LAN; Let's Encrypt no mira el A para validar por DNS-01. El certificado es público y confiable, se renueva solo y no requiere abrir puertos en el router. Requiere dominio propio y un token de API DNS (secreto). Caddy publica `arm/v7` y `arm64`. Queda como opción si más adelante se quiere acceder sin Tailscale.
- **Cloudflare Tunnel**: cert válido y Access, pero **todo el tráfico depende de Internet** incluso en la LAN y la imagen oficial no publica `arm/v7` (habría que hornear el binario). Descartado para LAN pura.
- **TLS nativo en Go**: mínima RAM y cero contenedores extra, pero hay que montar certificados y renovarlos (o sumar `certmagic` + DNS-01, que agrega dependencia y tamaño de binario). Es una alternativa válida si el usuario prefiere no sumar otro proceso.
- **mkcert/CA local**: funciona si la CA se instala como CA de usuario en Android, pero es frágil (Chrome 99+ exige Certificate Transparency si la CA está en el almacén del sistema; Android 17 endurece CT), hay que repetirlo en cada dispositivo y una actualización puede romperlo. Solo laboratorio.
- **`.local` + mDNS**: una CA pública no emite para `.local`; el autofirmado cae en el caso anterior; la resolución mDNS en Chrome Android no es confiable. Descartado.

### 3.2 Opciones evaluadas (implementación PWA)

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Manifest/SW hand-written en `frontend/public/` | Cero dependencias, control total, mínimo peso | Hay que versionar el cache a mano | ✅ Seleccionada |
| `vite-plugin-pwa` / Workbox | Generación automática de precache y SW | Dependencia nueva, más config y bundle; el iHost pide mínimas dependencias | ❌ Rechazada |
| Instalar solo con manifest (sin SW) | Menos trabajo, alcanza para "Install" en Chrome 108+ | Sin offline, sin push, iOS sin cache; PWA incompleta | ⚠️ MVP mínimo, no objetivo final |
| Botón de instalación con librería (`pwa-install`) | UI lista | Dependencia y estilos ajenos al design system iOS | ❌ Rechazada |
| Modal de instrucciones iOS con componente propio | Reutiliza patrón de modales del repo | Exige cumplir reglas de overlay/cierre | ✅ Seleccionada si se usa modal |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: HTTPS con certificado público es un bloqueador, no una mejora opcional.**

- **Contexto**: Chrome Android no instala PWA sobre HTTP en LAN; el service worker tampoco se registra. iOS permite agregar a inicio sin HTTPS, pero pierde offline/push.
- **Decisión**: resolver primero el origen seguro con **Tailscale Serve**. La app se publica en `https://<nodo>.<tailnet>.ts.net` con certificado Let's Encrypt automático, sin abrir puertos y sin dominio propio. Se descartan self-signed, `.local`, flags y Cloudflare Tunnel; Caddy + DNS-01 queda como alternativa si se quiere LAN sin VPN.
- **Consecuencias**: el origen canónico pasa a ser el nombre `*.ts.net` y la PWA queda atada a él. Cada dispositivo debe tener Tailscale instalado y activo (VPN siempre activa recomendada); si el tailnet o el control plane no está disponible, la PWA no abre. El iHost suma ~50-80 MB de RAM y un volumen de estado. A cambio, Android instala WebAPK real y ambas plataformas tienen SW/offline sin gestionar certificados ni DNS.

**ADR-002: Manifest y service worker hand-written, sin dependencias nuevas.**

- **Contexto**: el proyecto prioriza mínimo consumo y pocas dependencias en el iHost; el bundle ya pesa ~1 MB.
- **Decisión**: escribir `manifest.webmanifest`, `sw.js` y la lógica de instalación a mano. No agregar `vite-plugin-pwa`, Workbox ni librerías de UI.
- **Consecuencias**: menos mantenimiento de dependencias y control del cache; el versionado del cache y la estrategia offline son responsabilidad del equipo.

**ADR-003: Manifest e íconos públicos, same-origin y sin auth.**

- **Contexto**: Chrome pide el manifest y los íconos sin credenciales; si están detrás del login, la instalación falla (`cannot-download-icon`).
- **Decisión**: servir `/manifest.webmanifest` y `/icons/*` desde `public/`, sin middleware de auth. No usar `crossorigin="use-credentials"`.
- **Consecuencias**: metadatos públicos (nombre, íconos), sin datos sensibles. El resto de la app sigue protegido por cookie.

**ADR-004: Origen canónico estable y migración explícita.**

- **Contexto**: la PWA está atada al origen; `ihost.local`, la IP y `*.ts.net` son orígenes distintos, con cookies y almacenamiento distintos.
- **Decisión**: el origen canónico es `https://<nodo>.<tailnet>.ts.net`. Mantener `http://ihost.local:8088` como acceso LAN de respaldo (sin redirect global, para no romper el health check ni el acceso de escritorio) y no habilitar HSTS hasta validar todo. La PWA se instala **solo** desde el origen `https://`.
- **Consecuencias**: los usuarios que ya tengan un shortcut viejo deberán reinstalar desde el nuevo origen. Se documenta en la guía y se aclara que el acceso por IP/`ihost.local` no es instalable.

**ADR-005: Service worker con scope raíz y sin caché de API.**

- **Contexto**: la app maneja datos financieros y PDFs; cachear respuestas de API sería un riesgo de datos viejos o sensibles.
- **Decisión**: `sw.js` en `/`, navegación network-first con fallback al app shell, assets hasheados cache-first, `/api/*` siempre red, PDFs/descargas nunca cachean.
- **Consecuencias**: offline limitado al shell (no a datos); las vistas que requieren API muestran error de red, no datos stale. Hay que versionar el cache en cada release.

**ADR-006: UI de instalación propia y específica por plataforma.**

- **Contexto**: Android tiene `beforeinstallprompt`; iOS no tiene API y requiere instrucciones manuales.
- **Decisión**: card en Settings + banner discreto en Home; en Android botón "Instalar app" con el evento diferido; en iOS instrucciones "Compartir → Añadir a pantalla de inicio". Ocultar cuando ya está instalada (`display-mode: standalone` / `navigator.standalone`).
- **Consecuencias**: no se promete instalación programática en iOS; se evita mostrar UI inútil. Si se usa modal, se cumplen las reglas de SPEC-126.

### 3.4 Decisiones resueltas y abiertas

- **Resuelta — HTTPS**: **Tailscale Serve** (2026-11-08). Origen canónico `https://<nodo>.<tailnet>.ts.net`; ver ADR-001 y §4.2.7.
- **Abierta — Nombre visible**: se propone `p40la` para `name` y `short_name`; confirmar si se prefiere "p40la iHost" u otro.
- **Abierta — Alcance offline**: app shell solamente (propuesto) vs páginas visitadas con datos (riesgo de datos stale; no recomendado).

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
                    LAN / tailnet (WireGuard)
┌──────────────┐        HTTPS              ┌─────────────────────┐
│ Android      │ ────────────────────────► │ Tailscale Serve     │
│ Chrome       │  https://<nodo>.ts.net    │ cert Let's Encrypt  │
│ (WebAPK)     │  tailnet activo           │ sin abrir puertos   │
└──────────────┘                           └──────────┬──────────┘
┌──────────────┐        HTTPS                         │ http://127.0.0.1:8088
│ iOS Safari / │ ────────────────────────►            ▼
│ Chrome iOS   │                           ┌─────────────────────┐
│ (Home Screen)│                           │ Go server           │
└──────────────┘                           │ /api/*  /health     │
                                           │ /manifest.webmanifest│
                                           │ /sw.js  /icons/*    │
                                           │ /assets/*  SPA      │
                                           └──────────┬──────────┘
                                                      │
                                                 SQLite /app/data
```

### 4.2 Componentes y archivos

#### 4.2.1 Manifest

- **Ubicación**: `frontend/public/manifest.webmanifest` (Vite lo copia a `public/`).
- **Responsabilidad**: declarar identidad, modo standalone, colores e íconos.
- **Contenido propuesto**:

```json
{
  "id": "/",
  "name": "p40la",
  "short_name": "p40la",
  "description": "Gestión doméstica p40la",
  "lang": "es",
  "dir": "ltr",
  "start_url": "/",
  "scope": "/",
  "display": "standalone",
  "background_color": "#000000",
  "theme_color": "#000000",
  "prefer_related_applications": false,
  "icons": [
    { "src": "/icons/icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any" },
    { "src": "/icons/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any" },
    { "src": "/icons/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ]
}
```

- **Notas**: `background_color` se usa para el splash de Android; iOS lo ignora. El `theme_color` de `#000000` acompaña el dark mode por defecto; si se quiere acompañar el modo claro, agregar `<meta name="theme-color" media="(prefers-color-scheme: light)" content="#F2F2F7">` y su variante dark.

#### 4.2.2 Íconos

- **Ubicación**: `frontend/public/icons/`.
- **Archivos mínimos**: `icon-192.png`, `icon-512.png`, `icon-maskable-512.png`, `apple-touch-icon-180.png`, `apple-touch-icon-167.png`, `apple-touch-icon-152.png`.
- **Generación**: a partir de `favicon.svg` con una herramienta de escritorio (ImageMagick/rsvg-convert) en el momento de implementar; se commitean los PNG. No se agrega dependencia de runtime ni de build. Verificar 8-bit RGBA (no 16-bit), fondo opaco para `apple-touch-icon` y safe zone del 40% para maskable.
- **Regla**: no reutilizar el mismo archivo como `any` y `maskable`.

#### 4.2.3 `index.html`

- **Ubicación**: `frontend/index.html` (fuente) → `public/index.html` (build).
- **Cambios**:

```html
<meta name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover" />
<link rel="manifest" href="/manifest.webmanifest" />
<meta name="theme-color" content="#000000" />
<meta name="theme-color" media="(prefers-color-scheme: light)" content="#F2F2F7" />
<meta name="apple-mobile-web-app-capable" content="yes" />
<meta name="apple-mobile-web-app-title" content="p40la" />
<link rel="apple-touch-icon" sizes="180x180" href="/icons/apple-touch-icon-180.png" />
<link rel="apple-touch-icon" sizes="167x167" href="/icons/apple-touch-icon-167.png" />
<link rel="apple-touch-icon" sizes="152x152" href="/icons/apple-touch-icon-152.png" />
```

- **Nota iOS 26**: no depender de `apple-mobile-web-app-status-bar-style=black-translucent`; usar `viewport-fit=cover` + `env(safe-area-inset-*)` en el layout.

#### 4.2.4 Service worker

- **Ubicación**: `frontend/public/sw.js`.
- **Registro**: `frontend/src/main.tsx`, solo si `import.meta.env.PROD` y `'serviceWorker' in navigator`; en desarrollo no registrar para no cachear HMR.
- **Estrategia**:

| Ruta | Estrategia | Caché |
|---|---|---|
| Navegación (`request.mode === 'navigate'`) | Network-first; fallback a `/index.html` cacheado | `p40la-shell-vN` |
| `/assets/*` (hash) | Cache-first; si falta, red y cachear | `p40la-assets-vN` |
| `/i18n/*` | Stale-while-revalidate o cache-first | `p40la-i18n-vN` |
| `/icons/*`, `/manifest.webmanifest` | Cache-first | `p40la-shell-vN` |
| `/api/*` | **Network only**, sin caché | — |
| PDFs/descargas (`/api/...pdf`, adjuntos) | Network only | — |
| Otros GET same-origin | Network con fallback opcional | — |

- **Ciclo de vida**: `install` precachea shell mínimo; `activate` borra cachés con prefijo viejo y `clients.claim()`; `skipWaiting()` al recibir mensaje o en instalación inicial. No interceptar requests que no sean GET.
- **Versionado**: constante `CACHE_VERSION` en `sw.js`; cambiarla en cada release que modifique assets. El archivo se sirve con `no-cache` para que el navegador detecte la nueva versión.
- **Límite**: no cachear respuestas con `Set-Cookie`, `Authorization` ni contenido de usuario; no cachear más de ~5 MB.

#### 4.2.5 Backend: rutas, MIME y headers

- **Ubicación**: `internal/api/routes.go` y `cmd/server/main.go`.
- **Cambios**:
  1. Registrar `.webmanifest` con `mime.AddExtensionType(".webmanifest", "application/manifest+json")` al arrancar (o en `init` del paquete api).
  2. Agregar rutas explícitas `GET /manifest.webmanifest` y `GET /sw.js` con `http.ServeFile` (o un handler que devuelva 404 si no existe), **antes** del catch-all SPA, para que no devuelvan `index.html`.
  3. Envolver `/assets/*`, `/manifest.webmanifest`, `/sw.js` e `index.html` con headers de caché (ver REQ-008). Se puede hacer con un middleware `withCacheControl` o wrappers puntuales.
  4. Mantener `/health` sin auth y sin redirect interno.
- **Sin cambios de API de negocio, DB ni migraciones.**

#### 4.2.6 UI de instalación

- **Componente propuesto**: `frontend/src/components/InstallAppCard.tsx` (o sección dentro de `SettingsPage`), más un aviso opcional en `Home`.
- **Lógica**:
  - `useEffect` registra `beforeinstallprompt` → `preventDefault()` y guarda el evento en un ref/estado.
  - `appinstalled` → limpiar estado y ocultar.
  - `matchMedia('(display-mode: standalone)').matches` o `navigator.standalone` → ya instalada, no mostrar acción.
  - Detección de iOS: `navigator.userAgent`/`navigator.platform` (evitar dependencias) para mostrar instrucciones en lugar del botón.
  - Botón Android: `await deferredPrompt.prompt(); await deferredPrompt.userChoice;` y limpiar el evento.
  - iOS: texto/instrucciones con pasos; si se usa modal, cumplir SPEC-126 (sin cierre por click fuera, X/Cerrar visible, `z-[60]`, foco y teclado si aplica).
- **i18n**: agregar claves en `frontend/public/i18n/es.json` y `en.json` (nunca editar `public/i18n/` a mano; el build copia desde `frontend/public/`). Claves propuestas: `settings.app.title`, `settings.app.subtitle`, `settings.app.install`, `settings.app.installed`, `settings.app.ios_step_1..3`, `settings.app.android_unavailable`.

#### 4.2.7 Terminador TLS: Tailscale Serve (seleccionado)

- **Contenedor**: `tailscale/tailscale` (imagen oficial multi-arch; verificar `arm/v7`/`arm64` con `docker buildx imagetools inspect`), con `TS_AUTHKEY` como secreto, `TS_STATE_DIR=/var/lib/tailscale` en un volumen persistente y `TS_USERSPACE=true` si el iHost no expone `/dev/net/tun`.
- **UI del iHost, sin shell**: la configuración se hace desde la UI de Docker del iHost (variables de entorno + volúmenes). **No se usa `docker exec`**. El `serve.json` se crea/edita con el add-on **Filebrowser** (`filebrowser/filebrowser`) montando el volumen de configuración; no hace falta consola ni SSH.
- **Add-on oficial de Tailscale del iHost**: si ya existe el nodo `ihost.<tailnet>.ts.net` (add-on `tailscale/TailScale`), ese nodo publica la UI del iHost y **no** la app en `:8088` por defecto. Si el add-on expone `tailscale` → `/data/tailscalState` y `TS_STATE_DIR=/data/tailscalState`, se conserva ese volumen y se puede **intentar** agregar un volumen `/config` + `TS_SERVE_CONFIG=/config/serve.json` desde la UI; si el add-on no aplica Serve, la PWA usa un contenedor `tailscale/tailscale` **aparte** (`TS_HOSTNAME=p40la-ihost`). Ambos nodos conviven y el add-on oficial no se toca.
- **Red**: `tailscale serve` solo puede proxificar a `http://127.0.0.1`, así que el contenedor debe usar **Network: host** en la UI del iHost (la app ya publica 8088 en el host). La alternativa `network_mode: service:p40la-ihost` requiere compose/SSH y no es el flujo del iHost.
- **Arranque**: el contenedor hace el login solo con `TS_AUTHKEY` (`TS_AUTH_ONCE=true`). La publicación HTTPS se declara con `TS_SERVE_CONFIG=/config/serve.json` y un volumen `/config` montado como **directorio** (no archivo suelto) con el JSON de `deploy/tailscale/serve.json`; `containerboot` sustituye `${TS_CERT_DOMAIN}` y aplica cambios por fsnotify. No se usa `tailscale serve --bg` por CLI.
- **Tailnet**: habilitar **MagicDNS** y **HTTPS Certificates** en el admin de Tailscale; el certificado se emite por DNS-01 y queda en los logs de Certificate Transparency. Aplicar ACLs para que solo los dispositivos/usuarios del hogar accedan al nodo.
- **Origen canónico**: `https://<nodo>.<tailnet>.ts.net` (sin puerto). El nombre del nodo debe ser estable; renombrar o recrear el contenedor cambia el origen y obliga a reinstalar la PWA.
- **Clientes**: instalar Tailscale en Android/iOS, iniciar sesión en la misma tailnet y activar **VPN siempre activa** (Android) / perfil VPN (iOS). Sin Tailscale activo, el nombre `*.ts.net` no resuelve y la PWA no abre.
- **Fallback LAN**: `http://ihost.local:8088` sigue disponible para escritorio y recuperación; no se redirige globalmente. La PWA se instala solo desde `https://`.
- **Cookies**: el backend marca `Secure` cuando `r.TLS != nil` o cuando la request viene de loopback con `X-Forwarded-Proto: https` (Tailscale Serve lo agrega en modo HTTPS; **verificar el header en la fase 4**). Si no llega, usar `SECURE_COOKIE=true` fijo y documentar que el login por HTTP LAN deja de funcionar.
- **Sin HSTS** en la primera iteración. Sin exposición de puertos en el router. Sin token DNS ni certificados en el repo.

#### 4.2.8 Configuración

| Variable | Default | Uso en esta spec |
|---|---|---|
| `PORT` | `8088` | Sin cambios; Tailscale Serve apunta a `127.0.0.1:8088` |
| `SECURE_COOKIE` | `false` | `true` fuerza Secure en todos los orígenes (rompe login HTTP); con `false` se marca Secure por request si llega por HTTPS/`X-Forwarded-Proto` |
| `SESSION_DURATION` | `24h` | Sin cambios |
| `PUBLIC_ORIGIN` (nueva, opcional) | vacío | Para validar/loguear el origen canónico `*.ts.net`, si se implementa |

Variables del contenedor Tailscale (infraestructura, no del binario Go):

| Variable | Default | Uso |
|---|---|---|
| `TS_AUTHKEY` | — | Secreto de autenticación de la tailnet; nunca al repo |
| `TS_STATE_DIR` | `/var/lib/tailscale` | Estado persistente del nodo (volumen) |
| `TS_USERSPACE` | `false` | `true` si el iHost no expone `/dev/net/tun` |
| `TS_SERVE_CONFIG` | vacío | Config declarativa de Serve (opcional) |
| `TS_HOSTNAME` | `p40la-ihost` | Nombre estable del nodo (define el origen `*.ts.net`) |

### 4.3 Modelo de datos

No aplica. Esta spec no crea tablas, columnas ni migraciones. El manifest, los íconos y el service worker son archivos estáticos; la configuración TLS vive en el entorno/contenedor.

### 4.4 APIs / Contratos

No se agregan endpoints de negocio. Nuevos recursos estáticos:

#### Recurso: `GET /manifest.webmanifest`

- **Respuesta 200**: `Content-Type: application/manifest+json`, `Cache-Control: no-cache`, body JSON del §4.2.1.
- **Errores**: 404 si el archivo no existe (no devolver `index.html`).

#### Recurso: `GET /sw.js`

- **Respuesta 200**: `Content-Type: text/javascript; charset=utf-8`, `Cache-Control: no-cache`, `Service-Worker-Allowed: /` (opcional).
- **Errores**: 404 si no existe.

#### Recurso: `GET /icons/*.png`

- **Respuesta 200**: `Content-Type: image/png`, caché larga (hashed o nombre versionado).
- **Errores**: 404.

#### Existente: `GET /health`

- **Sin cambios**: `{"status":"ok"}`, HTTP interno, sin auth. No redirigir a HTTPS.

### 4.5 Dependencias

- **Internas**: `internal/api/routes.go`, `cmd/server/main.go`, `internal/config/config.go`, `frontend/index.html`, `frontend/src/main.tsx`, `frontend/src/pages/SettingsPage.tsx`, `frontend/public/i18n/{es,en}.json`, `docs/ihost.md`.
- **Externas**: ninguna de frontend. **Tailscale** es infraestructura (contenedor Docker multi-arch), no dependencia del binario Go; no se agrega `certmagic` ni librerías ACME al servidor.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: La app se sirve desde `https://<nodo>.<tailnet>.ts.net` con certificado Let's Encrypt confiable y `window.isSecureContext === true` en Chrome Android y Safari iOS, con Tailscale activo en el dispositivo.
- [ ] **CA-002**: `GET /manifest.webmanifest` devuelve 200, `Content-Type: application/manifest+json`, sin redirección al login y sin HTML; el JSON parsea y contiene todos los campos de REQ-002.
- [ ] **CA-003**: Chrome DevTools → Application → Manifest → **Installability** no muestra errores (`not-from-secure-origin`, `no-manifest`, `manifest-missing-suitable-icon`, etc.).
- [ ] **CA-004**: En Chrome Android, el menú **"Install and create shortcut"** muestra **Install habilitado**; al instalarlo se crea un WebAPK visible en `chrome://webapks` y la app abre en ventana standalone con splash e ícono correctos.
- [ ] **CA-005**: Los íconos 192, 512 y maskable 512 existen, son PNG válidos, públicos, con `sizes` reales y `purpose` correcto; el maskable respeta la safe zone. El `apple-touch-icon` 180×180 es PNG y se ve nítido en iOS.
- [ ] **CA-006**: En Safari iOS, **Compartir → Añadir a pantalla de inicio** con **"Abrir como app web"** activo crea un ícono que abre standalone con el ícono correcto; en Chrome/Edge iOS el mismo flujo funciona.
- [ ] **CA-007**: La PWA instalada permite iniciar y cerrar sesión; la cookie persiste al reabrir; sobre HTTPS se marca `Secure` y sobre HTTP LAN no (para no romper el acceso de respaldo); el login funciona en ambos orígenes según §4.2.7.
- [ ] **CA-008**: El service worker se registra en producción, controla la página y **no cachea `/api/*` ni PDFs**. Con la red deshabilitada, una navegación directa muestra el app shell (o la página offline) y no una pantalla de error del navegador.
- [ ] **CA-009**: `/sw.js` y `/manifest.webmanifest` se sirven con `Cache-Control: no-cache`; `/assets/*` con caché inmutable. Una nueva versión de `sw.js` se detecta y actualiza sin quedar pegado a la versión vieja.
- [ ] **CA-010**: Si el manifest o `sw.js` faltan, la respuesta es 404, no `index.html` con 200.
- [ ] **CA-011**: La UI muestra "Instalar app" en Android cuando el evento está disponible, instrucciones en iOS y **no muestra nada** (o muestra "Instalada") cuando ya corre en standalone.
- [ ] **CA-012**: `GET /health` sigue respondiendo `{"status":"ok"}` por HTTP interno y el health check de Docker no se ve afectado por Tailscale.
- [ ] **CA-013**: La app sigue funcionando por `http://ihost.local:8088` como acceso LAN de respaldo y el logout/login no rompe en ninguno de los dos orígenes.
- [ ] **CA-014**: No se agregaron dependencias de frontend ni migraciones; el build de Vite sigue copiando `frontend/public/` a `public/` y `emptyOutDir` no borra manifest/SW/íconos (deben vivir en `frontend/public/`, no en `public/`).
- [ ] **CA-015**: `docs/ihost.md` documenta Tailscale Serve (auth key, estado, `serve`, ACL), el origen canónico `*.ts.net` y cómo verificar la instalación en Android/iOS.
- [ ] **CA-016**: No hay PII, secretos, certificados ni claves privadas en el repo; el `TS_AUTHKEY` y el estado de Tailscale van por secreto/volumen, nunca commiteados.
- [ ] **CA-017**: Si se agrega un modal de instrucciones, no se cierra por click fuera, tiene cierre visible en todos los estados, respeta la escala de capas y se verificó en darkmode. Si se agrega un formulario, los inputs usan tokens del tema y se verificó el darkmode.
- [ ] **CA-018**: Si se agrega una página de detalle con lista padre, el retorno usa la flecha atrás del header vía `BACK_ROUTES`, sin links "← Título" en el contenido.
- [ ] **CA-019**: La imagen `tailscale/tailscale` es multi-arch `linux/amd64`, `linux/arm/v7` y `linux/arm64`, verificada con `docker buildx imagetools inspect`.
- [ ] **CA-020**: El add-on Tailscale se configura **desde la UI del iHost** (variables + volúmenes, sin `docker exec`), arranca sin errores en los logs y el nodo `p40la-ihost` aparece conectado en el admin de Tailscale.
- [ ] **CA-021**: El certificado `*.ts.net` carga sin warning en Chrome Android y Safari iOS; el nodo aparece con HTTPS habilitado en el admin de Tailscale.
- [ ] **CA-022**: El estado de Tailscale persiste en el volumen y la URL HTTPS sigue funcionando tras reiniciar el contenedor/iHost, sin reconfigurar `serve.json`.
- [ ] **CA-023**: `docs/ihost.md` explica que la PWA requiere Tailscale activo en el dispositivo, cómo instalar la app en Android/iOS y qué hacer si el tailnet no está disponible (acceso LAN HTTP de respaldo).

### 5.2 No funcionales

- [ ] **CA-NF-001**: El contenedor Tailscale no supera ~80 MB de RAM en el iHost y la app sigue arrancando sin swaps ni timeouts; el volumen de estado no crece sin control.
- [ ] **CA-NF-002**: El manifest + íconos + SW suman menos de ~1 MB al contenedor y no agregan dependencias de runtime.
- [ ] **CA-NF-003**: La primera carga bajo HTTPS no empeora de forma perceptible respecto de HTTP (sin compresión nueva en esta spec; el bundle de ~1 MB sigue igual).
- [ ] **CA-NF-004**: No se cachean respuestas de API, PDFs ni datos financieros; el logout limpia el estado de la app (el SW no retiene datos de usuario).

### 5.3 Testing

- **Unit tests (backend)**: rutas `/manifest.webmanifest` y `/sw.js` devuelven el MIME y `Cache-Control` esperados; archivo faltante → 404; `/health` no se redirige. Test de que `spaHandler` no captura las rutas PWA.
- **Unit tests (frontend)**: parsing/validación del manifest (campos obligatorios, íconos 192/512/any/maskable); lógica de detección de plataforma/standalone; que el registro del SW no ocurra en dev.
- **Integration tests**: build de Vite copia `manifest.webmanifest`, `sw.js` e íconos a `public/`; `GET /manifest.webmanifest` y `GET /sw.js` responden 200 con contenido; `GET /api/*` no se cachea (inspección del SW o test de estrategia).
- **Infra TLS**: nodo `p40la-ihost` conectado en el admin de Tailscale, URL `https://<nodo>.<tailnet>.ts.net` con certificado válido, logs del add-on sin errores (UI del iHost), persistencia tras reiniciar el add-on/iHost, header `X-Forwarded-Proto: https` en el backend y acceso LAN HTTP de respaldo funcionando.
- **E2E/manual (obligatorio)**:
  - Android real: Tailscale activo; Chrome actual, sin incógnito, app no instalada; menú → Install → WebAPK; `chrome://webapks`; abrir standalone; icono; splash; login; offline.
  - iOS real: Safari → Compartir → Añadir a pantalla de inicio → Open as Web App ON; abrir standalone; icono; login; reabrir; modo avión (shell).
  - Chrome/Edge iOS: mismo flujo desde Share.
  - Desktop: DevTools → Application → Manifest/Service Workers; verificar installability, cache y actualización.
- **Regresión**: `npm run build` (incluye `check:ui`), tests Go, login/logout por HTTP y HTTPS, health check, darkmode, i18n, y que la app siga funcionando sin SW.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 0 | Tailscale: cuenta/tailnet, MagicDNS + HTTPS Certificates, auth key, topología de red (host/userspace) y origen canónico | 0.5 día | Decisión del usuario (Tailscale Serve) |
| 1 | Manifest, íconos 192/512/maskable/apple, meta tags en `index.html`, MIME y rutas explícitas | 1 día | Fase 0 (hostname) |
| 2 | Service worker, registro, headers de caché y fallback offline | 1 día | Fase 1 |
| 3 | UI de instalación (card Settings + banner Home + instrucciones iOS) e i18n | 1 día | Fase 1 |
| 4 | Despliegue Tailscale Serve (contenedor, estado, `serve`, ACL), cookie Secure por request y docs | 1 día | Fase 0 |
| 5 | Validación Android/iOS, tests, darkmode, documentación y release | 1 día | Fases 1-4 |

**Total estimado**: 4-5 días de desarrollo + pruebas manuales en dispositivos reales.

### 6.2 Milestones

1. **MVP instalable en Android**: Tailscale Serve + manifest + íconos + rutas/MIME. Sin SW ni UI. Chrome muestra Install y crea WebAPK.
2. **V1.0 PWA completa**: SW/offline, UI de instalación Android/iOS, headers de caché, docs y validación en ambos dispositivos.
3. **V1.1 (opcional)**: splash iOS, screenshots, aviso de actualización, página offline dedicada.

### 6.3 Orden de commits sugerido

1. `SPEC-129: manifest, íconos y meta tags PWA`
2. `SPEC-129: service worker, registro y headers de caché`
3. `SPEC-129: UI de instalación Android/iOS e i18n`
4. `SPEC-129: despliegue Tailscale Serve y documentación`
5. `SPEC-129: criterios de aceptación y release`

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Tailnet o control plane de Tailscale no disponible / sin Internet | Media | Alto | Mantener HTTP LAN como fallback; documentar que la PWA requiere tailnet activo; monitorear |
| Dispositivo sin Tailscale activo no puede abrir la PWA | Alta | Medio | VPN siempre activa en Android/iOS; guía de instalación; acceso LAN HTTP para escritorio |
| Contenedor Tailscale sin `/dev/net/tun` o sin permisos | Media | Alto | `TS_USERSPACE=true`; `Network: host` en la UI del iHost (sin TUN ni capabilities) |
| `TS_SERVE_CONFIG` no se aplica tras reiniciar | Media | Medio | Volumen `/config` montado como **directorio**, `TS_SERVE_CONFIG=/config/serve.json`, estado persistente; revisar logs y reiniciar el add-on desde la UI |
| Renombrar o recrear el nodo cambia el origen y rompe la PWA | Media | Medio | Fijar `TS_HOSTNAME`; documentar reinstalación; no recrear el contenedor sin necesidad |
| Cambio de origen invalida PWA/shortcuts/cookies existentes | Alta | Medio | Definir origen canónico antes de implementar; mantener HTTP LAN sin redirect global; documentar reinstalación |
| Chrome Android sin Google Play Services no crea WebAPK | Media | Medio | Aceptar fallback a shortcut; documentar; no bloquear la app |
| Certificado self-signed/CA local rechazado o roto por updates | Media | Alto | No usar en producción; exigir cert público/confiable |
| Service worker cachea datos financieros o respuestas viejas | Media | Alto | `/api/*` network-only; no cachear PDFs; versionar y limpiar cachés; tests |
| SW sirve versión vieja por headers de caché | Alta | Medio | `no-cache` en `/sw.js` e `index.html`; `immutable` solo en assets hasheados |
| Manifest/SW servidos como HTML por el fallback SPA | Alta | Alto | Rutas explícitas + 404; test de MIME |
| `.webmanifest` con MIME incorrecto en distroless | Alta | Medio | `mime.AddExtensionType` o `manifest.json`; verificar con `curl -I` |
| iOS no comparte storage con Safari y el usuario no está logueado | Media | Medio | Cookie persistente; login dentro de la app; probar ambos flujos |
| `SameSite=Strict` pierde sesión en standalone | Baja | Medio | Probar; si falla, evaluar `Lax` con validación CSRF existente |
| HSTS mal configurado deja la app inaccesible | Baja | Alto | No habilitar HSTS hasta validar; rollback documentado |
| Tailscale consume demasiada RAM en iHost de 2 GB | Media | Medio | Medir; usar userspace; monitorear y limitar logs; evaluar Caddy si no alcanza |
| Imagen de Tailscale sin `arm/v7` o incompatible con el iHost | Media | Alto | Verificar multi-arch con `docker buildx imagetools inspect`; si falta, usar otra versión o build propio |
| Cambios futuros de criterios de Chrome/iOS | Baja | Medio | Verificar con DevTools/`chrome://webapks` en cada release; no depender de Lighthouse PWA |
| Se commitea `TS_AUTHKEY`, estado de Tailscale o clave privada | Baja | Crítico | Secreto Docker/env; volumen fuera del repo; rotar el auth key si se filtra; revisar `git diff --cached` |

---

## 8. Notas y Referencias

- **Chrome installability**: <https://web.dev/articles/install-criteria>
- **Chrome install criteria update (SW no longer required)**: <https://developer.chrome.com/blog/update-install-criteria>
- **Chrome Help — Install and create shortcut**: <https://support.google.com/chrome/answer/9658361>
- **Secure Contexts (W3C)**: <https://w3c.github.io/webappsec-secure-contexts/>
- **MDN — Making PWAs installable**: <https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Making_PWAs_installable>
- **MDN — Secure Contexts**: <https://developer.mozilla.org/en-US/docs/Web/Security/Defenses/Secure_Contexts>
- **WebKit — Safari 26 features**: <https://webkit.org/blog/17333/webkit-features-in-safari-26-0/>
- **WebKit — Web Push for Web Apps on iOS/iPadOS**: <https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/>
- **WebKit — Cookies 17.2**: <https://webkit.org/blog/14787/webkit-features-in-safari-17-2/>
- **Apple Support — Open as Web App**: <https://support.apple.com/guide/iphone/open-as-web-app-iphea86e5236/ios>
- **Tailscale — HTTPS certificates**: <https://tailscale.com/kb/1153/enabling-https>
- **Tailscale — Serve**: <https://tailscale.com/kb/1242/tailscale-serve>
- **Tailscale — Serve (feature docs)**: <https://tailscale.com/docs/features/tailscale-serve>
- **Tailscale — Docker**: <https://tailscale.com/kb/1282/docker>
- **Tailscale — Userspace networking**: <https://tailscale.com/kb/1112/userspace-networking>
- **Tailscale — Auth keys**: <https://tailscale.com/kb/1085/auth-keys>
- **Tailscale — ACLs**: <https://tailscale.com/kb/1018/acls>
- **Tailscale — MagicDNS**: <https://tailscale.com/kb/1081/magicdns>
- **Caddy — Automatic HTTPS (alternativa)**: <https://caddyserver.com/docs/automatic-https>
- **Let's Encrypt — Challenge types**: <https://letsencrypt.org/docs/challenge-types/>
- **Android — Certificate Transparency**: <https://developer.android.com/privacy-and-security/certificate-transparency-policy>
- **mkcert**: <https://github.com/FiloSottile/mkcert>
- **Specs relacionadas**: SPEC-001 (infraestructura iHost), SPEC-057 (iconos iOS), SPEC-063 (navegación de retorno), SPEC-118 (dropdowns), SPEC-122/SPEC-126 (modales).
- **Documentación interna**: `docs/ihost.md`, `docs/project-rules.md`, `docs/ssh-ihost-access.md`.

---

## 9. Estado de Implementación (2026-11-08)

### 9.1 Implementado

- **Manifest**: `frontend/public/manifest.webmanifest` con `id`, `name`, `short_name`, `start_url`, `scope`, `display: standalone`, colores e íconos 192/512 `any` + 512 `maskable`.
- **Íconos**: `frontend/public/icons/` con 192, 512, maskable 512, apple-touch 180/167/152 y `apple-touch-icon.png` opaco; script reproducible `scripts/generate-pwa-icons.sh`.
- **HTML**: `frontend/index.html` con manifest, `theme-color`, meta iOS, `apple-touch-icon` y `viewport-fit=cover`.
- **Service worker**: `frontend/public/sw.js` (v2) con precache del shell + assets hasheados, navegación network-first, cache-first de assets, stale-while-revalidate de i18n/íconos, y exclusión de `/api/*`, `/webhooks/*` y PDFs. Registro en `frontend/src/main.tsx` solo en producción y contexto seguro.
- **Backend**: `internal/api/pwa.go` (MIME `.webmanifest`, rutas explícitas `/manifest.webmanifest` y `/sw.js`, headers de caché), `internal/api/request.go` (cookie `Secure` por request) y rutas de assets/iconos con caché. `spaHandler` marca `no-cache`.
- **UI**: `frontend/src/components/InstallAppCard.tsx` + `frontend/src/hooks/useInstallPrompt.ts`; card en Settings con botón Android (`beforeinstallprompt`), instrucciones iOS e i18n `settings.app.*` en es/en. Se agregó el ícono `download`.
- **Guardrail**: `frontend/scripts/check-pwa-rules.cjs` valida manifest, campos, íconos PNG, service worker y meta tags; corre en `npm run build` (`check:pwa`).
- **Robustez offline**: `App.tsx` usa `Promise.allSettled` en la inicialización para no quedar en "Loading..." sin conexión.
- **Docs**: `docs/ihost.md` con Tailscale Serve, instalación Android/iOS, fallback LAN y diagnóstico.

### 9.2 Verificación automatizada/local

- `go build ./...` y `go test ./...`: OK (incluye `internal/api/pwa_test.go`).
- `npm run build`: OK (`check:ui`, `check:pwa`, `tsc -b`, Vite).
- Chrome headless (CDP) contra `http://localhost:8090` (localhost es secure context): `Page.getInstallabilityErrors` devuelve **0 errores**; manifest parseado sin errores; service worker registrado en scope `/`.
- Cachés del SW: `p40la-shell-v2` (incluye `index.html`, assets JS/CSS, manifest e íconos), `p40la-assets-v2`, `p40la-i18n-v2`; `/api/me` **no** está cacheado.
- Navegación offline: el SW sirve el shell cacheado; la app sale de "Loading..." y muestra login/shell en lugar de una pantalla de error.
- Cookies: login por HTTP → cookie **sin** `Secure`; login desde loopback con `X-Forwarded-Proto: https` → cookie **con** `Secure`; logout también respeta el flag.
- Headers: manifest `application/manifest+json` + `no-cache`; `sw.js` `text/javascript` + `no-cache` + `Service-Worker-Allowed: /`; assets `immutable`; íconos `image/png` con caché corta.
- UI: con UA Android se muestra la card y el hint de instalación; con UA iOS se muestran las instrucciones de Safari. La card se oculta cuando `display-mode: standalone` o `navigator.standalone`.

### 9.3 Pendiente de validación manual

- Configurar el contenedor `tailscale/tailscale` de la PWA en el iHost real (auth key, `TS_SERVE_CONFIG`, volúmenes, `Network: host`) y verificar `https://p40la-ihost.<tailnet>.ts.net`. El nodo `ihost.<tailnet>.ts.net` del add-on oficial no sirve para la PWA.
- Probar en Android real: Install habilitado, WebAPK en `chrome://webapks`, standalone, splash, login y offline.
- Probar en iOS real: Safari → Compartir → Añadir a pantalla de inicio → Abrir como app web; login y reapertura.
- Verificar consumo de RAM del contenedor Tailscale en el iHost y la persistencia tras reiniciar.
- Confirmar darkmode de la card de instalación y la ausencia de regresiones visuales en Settings.

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-08 | opencode | Creación inicial de la especificación con auditoría de `main`, investigación multi-agente (Chrome Android, iOS/WebKit, opciones HTTPS para iHost), requerimientos, diseño, criterios de aceptación y plan. Estado `draft`; decisión de HTTPS pendiente de confirmación del usuario. |
| 2026-11-08 | opencode | Decisión del usuario: **Tailscale Serve** como camino de HTTPS. Se actualizan ADR-001, REQ-001/REQ-011, tabla comparativa, §4.2.7 (contenedor, red, `serve`, clientes, cookies), configuración, criterios de aceptación, plan y riesgos. La PWA se instala desde `https://<nodo>.<tailnet>.ts.net`; `http://ihost.local:8088` queda como acceso LAN de respaldo. |
| 2026-11-08 | opencode | Implementación completa de la spec: manifest, íconos, service worker v2, rutas/MIME/headers, cookie `Secure` por request, card de instalación Android/iOS, guardrail `check:pwa`, robustez offline y documentación de Tailscale Serve. Verificación local con `go test ./...`, `npm run build` y Chrome headless (`Page.getInstallabilityErrors` sin errores, SW y cachés OK). Pendiente: configurar Tailscale en el iHost y validar en dispositivos reales. |
| 2026-11-08 | opencode | Corrección del despliegue en iHost: **no se usa `docker exec`**. Serve se declara con `TS_SERVE_CONFIG=/config/serve.json` y un volumen `/config` (directorio) editado con el add-on Filebrowser desde la UI del iHost (`deploy/tailscale/serve.json`). Se aclara que `ihost.<tailnet>.ts.net` es el nodo del add-on oficial `tailscale/TailScale` del iHost y no la app; la PWA usa un contenedor `tailscale/tailscale` aparte con `TS_HOSTNAME=p40la-ihost` y URL `https://p40la-ihost.<tailnet>.ts.net`. |
