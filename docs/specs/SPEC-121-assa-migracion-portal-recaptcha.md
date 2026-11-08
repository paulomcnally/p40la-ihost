---
title: "ASSA: recuperar la automatización tras la migración del portal Mi ASSANET (reCAPTCHA)"
id: "SPEC-121"
status: "released"
author: "opencode"
created: "2026-11-03"
updated: "2026-11-08"
github_issue: 124
---

# ASSA: recuperar la automatización tras la migración del portal Mi ASSANET (reCAPTCHA)

**ID**: SPEC-121  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-03  
**Actualizado**: 2026-11-08

---

## 1. Resumen Ejecutivo

La automatización de ASSA (`assa.nicaragua`) dejó de funcionar el **2026-10-17**. La última ejecución exitosa fue el **2026-10-16** y, desde entonces, cada intento falla con:

```
la API del proveedor cambió su formato: login sin token var
```

La causa raíz **no** es un cambio de nombre del token `var`: ASSA reemplazó el login del portal legado "Mi ASSANET" (ASP.NET WebForms + Ext.NET) por una aplicación MVC nueva y dejó el endpoint viejo como redirect. El endpoint que usaba el plugin (`/ni/miassanet-NI/login_v.aspx`) hoy responde `302` hacia `/ni/MiAssan-NI/login`, sin `var` y sin `.ASPXAUTH`. El login nuevo exige antiforgery token y **reCAPTCHA v3** (con escalado a v2), por lo que un cliente HTTP puro en Go no puede autenticarse. El plugin v1.1.0, última versión publicada, no tiene forma de resolver el login legacy.

El objetivo de esta spec es doble:

1. **P0 — Dejar de fallar a ciegas:** detectar explícitamente la migración y el captcha, clasificarlos con errores distintos y mostrar un mensaje accionable, sin falsos positivos ni reintentos agresivos.
2. **P1 — Recuperar los datos:** validar y ejecutar una ruta de reemplazo. La opción recomendada es una **sesión asistida** (el usuario inicia sesión manualmente una vez y el plugin reutiliza la cookie de sesión), con la **API oficial de ASSA** como solución definitiva si está disponible. Un navegador headless dentro del iHost se descarta por consumo de recursos, score de reCAPTCHA y términos de uso.

Consideraciones de iHost: no se agregan dependencias pesadas ni un navegador; no se modifica el esquema SQLite; el trabajo nuevo es un tag del módulo privado de plugins, un bump de `go.mod` y una imagen multi-arch. La cookie de sesión se guarda en el campo `credentials` existente, se enmascara en la UI y nunca se loguea.

Consideraciones de UI: el formulario de cuenta ya usa los tokens del tema y el componente `Select`; si se agrega el campo de cookie, debe verse bien en darkmode y mantener el patrón de credenciales existente. No se crean páginas de detalle nuevas, por lo que no aplica `BACK_ROUTES` salvo que el diseño cambie.

### 1.1 Alcance liberado (2026-11-08)

Se liberó el **MVP P0** (Fase 1 y milestone 1): detección explícita de la migración y del CAPTCHA, errores centinela nuevos, mensaje accionable en backend/UI, tests de regresión y plugin `v1.1.1` publicado.

La **Fase 2 (spike con sesión real) y la Fase 3 (sesión asistida + parsers del portal nuevo)** quedaron diferidas a **SPEC-129** por una dependencia externa: requieren que el usuario inicie sesión manualmente y aporte una cookie de sesión válida para mapear las rutas autenticadas. Sin esa evidencia no se escribe el scraping, tal como exigen REQ-005, REQ-008 y CA-008.

Consecuencia operativa del MVP: la cuenta ASSA deja de reportar un error engañoso ("la API cambió su formato") y pasa a reportar `portal migrado` o `CAPTCHA requerido`; el usuario puede desactivar el schedule sin perder configuración ni facturas históricas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Detectar la migración del portal.** Cuando el `POST` legacy responda `302` hacia la ruta nueva de login (`/ni/MiAssan-NI/login` o variante de mayúsculas), el plugin debe devolver un error centinela nuevo (`ErrPortalMigrated`) con un mensaje accionable, en vez de `ErrAPISchemaChanged: login sin token var`.
2. **REQ-002 — Detectar el captcha del portal nuevo.** Si la respuesta es `200` con el formulario nuevo (`txt_usuario`, `txt_contrasena`, `recaptchaTokenV3`, `__RequestVerificationToken`) o con el mensaje de CAPTCHA, el plugin debe devolver `ErrCaptchaRequired` (o equivalente) y no reintentar el login dentro de la misma ejecución.
3. **REQ-003 — Prohibido el falso positivo.** No se debe "arreglar" el error agregando `?var=...` a la URL legacy ni siguiendo redirecciones a ciegas. El servidor legacy preserva query strings y devolvería un `Location` con `var` sin sesión válida; eso produciría un falso login. Debe existir un test que lo impida.
4. **REQ-004 — Seguridad operativa.** El error persistido y el log deben incluir solo status HTTP y path sanitizado; nunca cookies, tokens, `Set-Cookie`, credenciales ni query strings sensibles. La UI debe mostrar un mensaje claro ("ASSA migró su portal; requiere sesión manual" / "CAPTCHA requerido") y permitir desactivar el schedule de la cuenta sin borrar datos.
5. **REQ-005 — Definir la ruta de reemplazo antes de implementarla.** La spec debe cerrar una decisión validada entre: (a) sesión asistida por cookie, (b) API oficial de ASSA, (c) carga manual/PDF. No se acepta implementar scraping del portal nuevo sin haber validado que la sesión y las rutas de datos funcionan con una cuenta real.
6. **REQ-006 — No filtrar secretos.** La cookie de sesión, si se implementa, se guarda en `automation_accounts.credentials` (mismo mecanismo que la contraseña), se enmascara en la UI, no se devuelve en listados y no aparece en logs, errores, tests ni commits. El `reveal` sigue exigiendo contraseña de la app.
7. **REQ-007 — Sesión asistida (si se elige).** Agregar al `CredentialSchema` de ASSA un campo opcional `session_cookie` de tipo `password`/`Secret`. El plugin debe: validar la cookie contra una ruta autenticada, usarla para leer los datos y devolver `ErrAuthExpired` cuando el portal redirija al login. La UI debe explicar cómo renovarla y permitir pegar una nueva sin recrear la cuenta.
8. **REQ-008 — Mapear el portal nuevo.** Identificar las rutas autenticadas de pólizas, polifacturas pendientes y recibos pagados en la aplicación MVC nueva. Construir parsers con fixtures sintéticos y sanitizados. Si el mapeo no es viable o la sesión dura demasiado poco, detener la implementación y volver a la opción manual/oficial.
9. **REQ-009 — Tests de regresión.** Agregar tests para: `302` legacy → `/login` sin `var`; `200` con formulario nuevo/captcha; cookie válida; cookie expirada; y ausencia de falsos positivos por `var` agregado. Los tests deben usar datos sintéticos (`demo-user`, `demo-pass`, `demo-cookie`, `demo-policy-001`).
10. **REQ-010 — Versionado y release.** Cualquier cambio del plugin requiere un tag nuevo del módulo privado: `v1.1.1` si solo agrega detección/diagnóstico, `v1.2.0` si agrega sesión asistida y nuevos parsers. Actualizar `go.mod`/`go.sum`, reconstruir la imagen multi-arch (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) y verificar que `automation_plugins.version` y `automation_bills.plugin_version` reflejen la versión nueva.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

> **Estado**: los requerimientos que dependen de la sesión real (REQ-007, REQ-008) se trasladaron a **SPEC-129**. REQ-011 a REQ-015 quedan como seguimiento de la operación del MVP.

1. **REQ-011 — Taxonomía de errores en la app.** Mapear `ErrPortalMigrated`, `ErrCaptchaRequired` y `ErrAuthExpired` a mensajes/HTTP claros en `GET`/`POST` de automation y en el sync de servicios, sin romper el contrato existente (`502 automation_error`/`plugin_error`).
2. **REQ-012 — Alertas de fallo recurrente.** Cuando una cuenta acumule N ejecuciones fallidas por migración/captcha/sesión, mostrar un aviso visible en la página de automation (y opcionalmente Telegram/email) con la acción recomendada.
3. **REQ-013 — Runbook de renovación.** Documentar en la spec y en la UI el procedimiento para renovar la cookie de sesión (login manual, extracción segura, pegado, validación) sin exponer valores reales.
4. **REQ-014 — Semántica de facturas.** Revisar el hallazgo colateral: el plugin devuelve polifacturas pendientes y recibos pagados, y el upsert de p40la es por `(service_id, year, month)`. Esto puede colapsar varias facturas del mismo mes o marcar una pendiente como pagada. No es la causa del fallo actual, pero debe evaluarse antes de reactivar ASSA; si requiere cambios, abrir una spec separada.
5. **REQ-015 — Observabilidad.** Registrar en `automation_webhook_logs`/`automation_bills` el tipo de fallo y la versión del plugin, sin secretos, para poder distinguir migración, captcha y sesión expirada.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016 — API oficial.** Iniciar el contacto con ASSA para obtener acceso a API/partner o a un canal de servicio que no dependa de captcha. Si existe, priorizarlo sobre cualquier scraping.
2. **REQ-017 — Renovación automática de sesión.** Si el portal ofrece "recordarme" o una sesión de larga duración, evaluar si la cookie puede renovarse sin interacción humana. No implementar bypass de captcha.
3. **REQ-018 — Fallback manual/PDF.** Mientras no haya automatización, permitir cargar la factura de ASSA manualmente o vía el analizador de PDF existente (SPEC-040) y dejar claro en la UI que el dato no vino del portal.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: una ejecución diaria por cuenta; timeout de 15 s por request; sin reintentos dentro del mismo job; sin navegador headless.
- **Seguridad**: cookies y contraseñas nunca en logs, errores, tests, issues ni commits; `credentials` fuera de los listados; `reveal` con contraseña; tests con datos sintéticos.
- **Almacenamiento**: sin cambios de esquema; reutilizar `automation_accounts.credentials`, `automation_bills` y `automation_webhook_logs`. No guardar cookies en `raw`.
- **Disponibilidad**: si ASSA falla, el resto de la app y los demás plugins siguen operativos; el scheduler no debe bloquearse.
- **iHost**: sin Chromium/Playwright; sin dependencias Go nuevas para detección; memoria y CPU adicionales despreciables.
- **UI**: inputs con `bg-card`/`text-text`; darkmode verificado; sin `<select>` nativo; si se agrega un selector, usar `Select`.
- **Compatibilidad**: mantener el contrato `plugins.Plugin`; no romper los otros cuatro plugins.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

**Línea de tiempo**

| Fecha | Evento |
|---|---|
| 2026-09-14 | Última verificación documentada del login legacy en el plugin (302 con `var` y `.ASPXAUTH`). |
| 2026-10-16 | Última ejecución exitosa de ASSA en producción. |
| 2026-10-17 | Primer fallo: `login sin token var`. |
| 2026-11-03 | Verificación read-only del portal: el login legacy redirige al login nuevo; el plugin v1.1.0 sigue fallando. |

**Evidencia del portal (2026-11-03, sin credenciales reales)**

| Prueba | Resultado |
|---|---|
| `GET /ni/miassanet-NI/login_v.aspx` | `302` → `https://miassanet.assanet.com/ni/MiAssan-NI/login`, sin query y sin `Set-Cookie`. |
| `POST /ni/miassanet-NI/login_v.aspx` con el body legacy | Mismo `302` al login nuevo; el body se ignora. |
| `GET /ni/MiAssan-NI/login` | `200`, formulario MVC nuevo con `txt_usuario`, `txt_contrasena`, `recaptchaTokenV3`, `requireRecaptchaV2` y `__RequestVerificationToken`; cookie de antiforgery. |
| `POST /NI/MiAssan-NI/` con datos sintéticos y sin token de captcha | `200` con "Por favor complete el CAPTCHA"; sin cookie de autenticación. |
| `POST /NI/MiAssan-NI/` con token de captcha sintético | Mismo `200`; el token se valida server-side. |
| `GET /ni/miassanet-NI/Default_p.aspx?var=TESTVAR` | `302` → `/ni/MiAssan-NI/Home?var=TESTVAR` → `/NI/MiAssan-NI/Login/LogOut`; el `var` ya no autentica. |
| `GET /ni/sca/Consulta_Polizas.aspx` sin sesión | `302` al login legacy con `error=1`; las páginas viejas siguen existiendo pero requieren la sesión legacy. |

**Código del plugin v1.1.0**

- `login()` construye el body en orden exacto `txt_Usuario`, `txt_Clave`, `botonLoginAux`, `TipoDispotivivo` y espera `302` + `Location` con `var` (`assa/nicaragua/nicaragua.go:176-204`).
- El error actual se lanza en `:199-202` cuando `u.Query().Get("var")` viene vacío.
- `plugins/types.go:9-16` define `ErrAuthFailed`, `ErrAuthExpired`, `ErrServiceNotFound`, `ErrAPISchemaChanged`, `ErrUpstream`.
- El repo privado solo tiene `v1.0.0` y `v1.1.0`; `main == v1.1.0`, sin ramas ni fixes posteriores. El cambio v1.0.0→v1.1.0 de ASSA fue solo `IssueDate`/versión, no el login.
- Los tests usan un mock del portal viejo (`302` con `.ASPXAUTH` y `var`), por eso `go test` sigue verde aunque producción esté rota.

**Diagnóstico**

La migración es un cambio de contrato completo, no un rename de parámetro:

- Ya no existe `var`.
- Ya no se emite `.ASPXAUTH` en el login legacy.
- El login nuevo exige antiforgery y reCAPTCHA v3; el servidor valida el token contra Google.
- Un `POST` legacy con `?var=` agregado solo produciría un falso positivo: el `Location` incluiría `var` pero no habría sesión.
- El portal nuevo escala a reCAPTCHA v2 tras un intento fallido, lo que aumenta el riesgo de bloqueo si se automatiza a ciegas.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Cambiar el nombre del parámetro `var` | Cambio mínimo | No existe un parámetro equivalente; el portal nuevo no emite token | ❌ Rechazada |
| B. Agregar `?var=` a la URL legacy para "pasar" el chequeo | Una línea de código | Falso positivo: no hay sesión; los datos fallan después; oculta la causa | ❌ Rechazada |
| C. Navegador headless en el iHost (chromedp/Playwright) | Podría ejecutar reCAPTCHA v3 | Chromium consume RAM/disco; el score de v3 puede castigar bots; términos de uso; multi-arch pesado | ❌ Rechazada para iHost |
| D. Sesión asistida por cookie | Sin captcha ni navegador; usa stdlib; el usuario controla el login | Requiere login manual y renovación; cookie bearer; hay que mapear rutas nuevas | ✅ Seleccionada para validación P1 |
| E. API/partner oficial de ASSA | Estable, sin captcha, soporte | Depende de ASSA; puede no existir o tardar | ✅ Preferida a largo plazo |
| F. Carga manual / PDF (SPEC-040) | Disponible ya; sin riesgo | No es automática; solo cubre comprobantes pagados | ✅ Fallback operativo |
| G. Servicio externo con navegador real y webhook | Evita el iHost; puede pasar v3 | Reintroduce un segundo proceso; costo/operación; no unificado | ⚠️ Solo si D/E fallan |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: La causa raíz es la migración del portal, no el token.**
- **Contexto**: el error dice "login sin token var", pero el portal nuevo no tiene `var`.
- **Decisión**: tratar el incidente como migración de portal y no como un cambio de parser.
- **Consecuencias**: no se pierde tiempo en un fix cosmético; se documenta la dependencia externa.

**ADR-002: No se implementa bypass de captcha ni navegador headless en el iHost.**
- **Contexto**: reCAPTCHA v3 requiere ejecutar JS y validarse server-side; el iHost tiene recursos limitados.
- **Decisión**: descartar Chromium/Playwright y cualquier solver de captcha.
- **Consecuencias**: la automatización plena queda condicionada a una API oficial o a una sesión asistida.

**ADR-003: La sesión asistida es la ruta interina recomendada.**
- **Contexto**: el usuario ya tiene credenciales y puede iniciar sesión manualmente; la app ya guarda secretos por cuenta.
- **Decisión**: agregar `session_cookie` como credencial secreta opcional y usarla para leer el portal nuevo; detectar expiración con `ErrAuthExpired`.
- **Consecuencias**: no es 100 % desatendida; requiere renovación manual. Se debe validar la duración de la sesión antes de comprometerse.

**ADR-004: Sin cambios de esquema SQLite.**
- **Contexto**: `automation_accounts.credentials` ya es JSON y `automation_bills` ya guarda errores/versión.
- **Decisión**: reutilizar el modelo; no crear tablas ni columnas.
- **Consecuencias**: release más simple y reversible; la limpieza de datos existentes queda fuera de alcance.

**ADR-005: El diagnóstico P0 se puede liberar solo, antes del fix funcional.**
- **Contexto**: la cuenta lleva semanas fallando con un mensaje engañoso.
- **Decisión**: publicar primero `v1.1.1` (detección/clasificación) y luego `v1.2.0` (sesión/parsers).
- **Consecuencias**: el usuario obtiene un mensaje correcto y puede desactivar el schedule mientras se define la solución.

**ADR-006: La API oficial es la única solución plenamente desatendida.**
- **Contexto**: cualquier scraping post-login sigue dependiendo de una sesión y de un portal que puede cambiar.
- **Decisión**: iniciar gestión con ASSA; si no hay API, aceptar sesión asistida o carga manual.
- **Consecuencias**: el objetivo "automático para siempre" no está garantizado por código propio.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Scheduler diario]
       |
       v
[AutomationService.SyncAccount]
       |
       v
[assa.nicaragua plugin]
       |
       +--> POST login_v.aspx
       |        |
       |        +--> 302 a /MiAssan-NI/login  --> ErrPortalMigrated
       |        +--> 200 con CAPTCHA           --> ErrCaptchaRequired
       |        +--> 302 con var (legacy)      --> flujo viejo (solo si aún existe)
       |
       +--> [Sesión asistida: cookie]
                |
                +--> GET ruta autenticada --> 302 login --> ErrAuthExpired
                +--> GET pólizas/facturas/recibos --> parsers nuevos
                        |
                        v
              [WebhookService.UpsertBill]
                        |
                        v
                  [bills / automation_bills]
```

### 4.2 Componentes

#### 4.2.1 Plugin privado `assa.nicaragua`
- **Responsabilidad**: detectar el estado del portal, autenticar por sesión asistida cuando esté configurada, parsear pólizas/facturas/recibos y devolver `plugins.Bill`.
- **Interfaz**: `plugins.Plugin` (`Name`, `Version`, `Source`, `Description`, `CredentialSchema`, `FetchBills`).
- **Dependencias**: stdlib (`net/http`, `net/http/cookiejar`, `encoding/json`, `regexp`); sin navegador.
- **Ubicación**: repo privado `p40la-ihost-automation-plugins`, `assa/nicaragua/nicaragua.go` y tests.

Cambios propuestos:
1. Nuevos centinelas en el módulo (o en el plugin si el contrato no se puede cambiar): `ErrPortalMigrated`, `ErrCaptchaRequired`.
2. `login()`: si el `Location` apunta a `/MiAssan-NI/login` (sin `var`), devolver `ErrPortalMigrated` con status y path sanitizado; si el body es el formulario nuevo, `ErrCaptchaRequired`.
3. `FetchBills()`: si `creds["session_cookie"]` está presente, saltar el login legacy y usar la cookie; validar la sesión contra una ruta autenticada.
4. `CredentialSchema()`: agregar `session_cookie` (label "Cookie de sesión", `Type: "password"`, `Required: false`, `Secret: true`), manteniendo `username`/`password` para cuando el portal legacy vuelva o para la API oficial.
5. `Source()`: mantener el host; no cambiar el `identifier` (póliza).
6. No registrar la cookie; no incluirla en `Raw`; no persistirla en `automation_bills`.

#### 4.2.2 Backend p40la
- **Responsabilidad**: persistir el error clasificado, exponerlo y decidir si se reintenta.
- **Ubicación**: `internal/services/automation_sync.go`, `internal/services/automation_scheduler.go`, `internal/api/automation_handlers.go`, `internal/api/service_handlers.go`.
- **Cambios**:
  - `FetchBills` sigue guardando `err.Error()` en `automation_bills.error`, pero el texto debe ser accionable y sanitizado.
  - `SyncAccount` no debe llamar `SetLastRun` en fallo (comportamiento actual), para que el scheduler no pierda el estado.
  - El scheduler ya ejecuta una vez al día por cuenta; no se agregan reintentos internos.
  - Mapear `ErrPortalMigrated`/`ErrCaptchaRequired`/`ErrAuthExpired` a mensajes de UI; si se mantiene el `502`, incluir un `code` legible en el body (sin cambiar el contrato existente).
  - No se requiere endpoint nuevo: la validación de la cookie ocurre en el fetch/sync.

#### 4.2.3 Frontend
- **Responsabilidad**: mostrar el estado y permitir pegar/renovar la cookie.
- **Ubicación**: `frontend/src/pages/AutomationFormPage.tsx` (credenciales dinámicas) y `AutomationDetailPage.tsx` (errores/estado).
- **Cambios**:
  - El campo `session_cookie` aparece automáticamente desde `GET /api/automation/plugins/{name}/schema`; usar `type="password"` y las clases de tokens existentes.
  - En el detalle, mostrar un banner con el estado "Portal migrado / CAPTCHA / Sesión expirada" y un enlace a las instrucciones de renovación.
  - No mostrar la cookie; el `reveal` existente sigue pidiendo la contraseña de la app.
  - Si se agrega un selector, usar `Select`; no `<select>` nativo.

#### 4.2.4 Base de datos
- Sin migraciones. `automation_accounts.credentials` guarda el JSON (incluida la cookie si se elige la opción D).
- `automation_bills` sigue guardando `plugin_version`, `status`, `error` y `raw` (sin cookie).
- `automation_webhook_logs` sigue igual; no guardar headers ni cookies.

### 4.3 Modelo de datos

No hay entidades nuevas. Ejemplo sintético de credenciales para la opción D:

```json
{
  "username": "demo-user",
  "password": "demo-pass",
  "session_cookie": "demo-cookie"
}
```

`session_cookie` es opcional: si está vacío, el plugin intenta el flujo legacy y devuelve el error clasificado que corresponda.

### 4.4 APIs / Contratos

No se agregan endpoints. Se reutilizan:

- `GET /api/automation/plugins/{name}/schema` → incluye `session_cookie`.
- `PUT /api/automation/accounts/{id}` → acepta `credentials` JSON; si el campo viene vacío, conserva las credenciales actuales (comportamiento existente).
- `POST /api/automation/accounts/{id}/bills:fetch` → devuelve `502 plugin_error` con el mensaje clasificado si falla.
- `POST /api/services/{id}/sync` → devuelve `502 automation_error` si el plugin falla.

**Response de error (ejemplo, sin datos reales)**:

```json
{
  "error": "automation_error",
  "message": "ASSA migró su portal y el acceso automático ya no es válido. Desactivá la automatización para evitar errores repetidos."
}
```

### 4.5 Dependencias

- **Internas**: módulo privado de plugins, `AutomationService`, scheduler, handlers de automation/servicios, UI de automation.
- **Externas**: portal ASSA (sin contrato estable), Google reCAPTCHA (solo si se intenta login, lo cual no se hará en la opción D).
- **Sin nuevas**: no se agrega Chromium, Playwright, selenium, dependencias de captcha ni librerías de scraping.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] **CA-001**: Un `302` legacy hacia `/ni/MiAssan-NI/login` (sin `var`) produce `ErrPortalMigrated` y el mensaje accionable; no produce "login sin token var". Verificado con `TestFetchBills_PortalMigrated`.
- [x] **CA-002**: Un `200` con el formulario nuevo (`txt_usuario`, `recaptchaTokenV3`) o con "Por favor complete el CAPTCHA" produce `ErrCaptchaRequired`. Verificado con `TestFetchBills_CaptchaRequired`.
- [x] **CA-003**: Existe un test que verifica que el plugin **no** agrega `var` a la URL legacy ni acepta un `Location` con `var` como sesión válida si no hay cookie autenticada. Verificado con `TestFetchBills_LegacyVarWithoutCookie` y la aserción de `RawQuery` vacío.
- [x] **CA-004**: La UI de la cuenta muestra el estado de migración/captcha/sesión expirada y permite desactivar el schedule; el error no es un genérico "API cambió su formato". Banner en el detalle con botón Editar; `error_code` verificado por API.
- [ ] **CA-005**: Diferido a SPEC-129: cookie válida, parsers y `ErrAuthExpired`.
- [x] **CA-006**: El MVP no incorpora cookie; los errores persistidos/logueados se sanitizan y los tests verifican que no filtran credenciales ni tokens. El `reveal` no cambió.
- [x] **CA-007**: La cuenta ASSA puede quedar con el schedule desactivado sin perder la configuración ni las facturas históricas (se probó en la DB temporal con la cuenta demo desactivada).
- [ ] **CA-008**: Diferido a SPEC-129: decisión no-go entre sesión asistida, API oficial o carga manual.
- [x] **CA-DARK**: No se agregó un campo de formulario nuevo; el banner usa tokens del tema (`bg-danger/10`, `text-text-secondary`). No aplica la verificación de placeholder.
- [x] **CA-BACK**: No se creó una página nueva; no aplica `BACK_ROUTES`.

### 5.2 No funcionales

- [x] **CA-NF-001**: El plugin no incorpora navegador ni dependencias pesadas; solo agrega detección y centinelas.
- [x] **CA-NF-002**: El scheduler conserva un intento por ejecución y el plugin tiene timeout de 15 s; no se agregaron reintentos.
- [x] **CA-NF-003**: `go test ./...` del repo de la app y del módulo de plugins pasa.
- [x] **CA-NF-004**: La imagen se publica con los tres manifests (`linux/amd64`, `linux/arm/v7`, `linux/arm64`). Verificado con `docker buildx imagetools inspect paulomcnally/p40la-ihost:0.5.5`.
- [x] **CA-NF-005**: No se usaron datos reales: la validación local usó una DB temporal en `/tmp` y `data/app.db` no se tocó.
- [x] **CA-NF-006**: `PRAGMA integrity_check` y `foreign_key_check` quedan OK en la DB temporal de validación.

### 5.3 Testing

- **Unit tests (plugin)**: login legacy con `302` sin `var`; login nuevo con captcha; cookie válida; cookie expirada; parser de pólizas/facturas/recibos con fixtures sintéticos; ausencia de falsos positivos.
- **Unit tests (app)**: persistencia de errores clasificados; mapeo a mensajes; scheduler no reintenta dentro del mismo job; no se guarda `SetLastRun` en fallo.
- **Integration tests**: sync con un plugin fake que devuelve `ErrPortalMigrated` y con uno que devuelve `ErrAuthExpired`; verificar `automation_bills` y logs.
- **E2E manual**: con una cuenta real y una cookie provista por el usuario (nunca commiteada), validar login manual → pegado de cookie → sync → facturas en el servicio. Si la sesión expira, verificar el mensaje y la renovación.
- **Performance**: medir tiempo/RAM de una ejecución diaria en el iHost; no debe superar el presupuesto actual.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 0 | Aprobar la spec y elegir la ruta (D/E/F) con el usuario | 0.5 día | Ninguna |
| 1 | Plugin `v1.1.1`: detección de migración/captcha, errores nuevos, tests; app mapea mensajes | 0.5–1 día | Fase 0 |
| 2 | Spike técnico con el usuario: login manual, cookies, duración de sesión, rutas nuevas y parsers | 0.5–1 día | Fase 1 |
| 3 | Plugin `v1.2.0`: sesión asistida, validación, expiración, parsers nuevos, UI de renovación | 3–5 días | Fase 2 (go) |
| 4 | Release: bump `go.mod`, tag, build multi-arch, deploy, verificación en producción | 0.5 día | Fase 3 |

Si la Fase 2 da "no-go", las fases 3–4 se reemplazan por la opción E (API oficial) o F (manual/PDF), y la spec se actualiza antes de continuar.

### 6.2 Milestones

1. **MVP (P0)**: detección de migración/captcha, mensaje accionable, tests y release `v1.1.1`. La cuenta puede quedar desactivada sin ruido.
2. **V1.0 (P1)**: sesión asistida funcionando con una cuenta real, facturas entregadas y renovación documentada.
3. **V2.0 (P2)**: API oficial o renovación automática, si ASSA la ofrece.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| El portal nuevo cambia otra vez sus rutas/HTML | Alta | Alto | Parsers tolerantes, fixtures, detección de errores, tests; monitoreo y versionado del plugin |
| La sesión dura muy poco y la renovación manual es inviable | Media | Alto | Medir duración en Fase 2; si es < 24 h, optar por API oficial o manual |
| reCAPTCHA v3/v2 bloquea o castiga la automatización | Alta | Alto | No automatizar login; usar sesión asistida u API oficial; no resolver captcha |
| Bloqueo de cuenta por intentos repetidos | Media | Alto | Un intento por día, sin reintentos; no usar credenciales reales en pruebas; avisar al usuario |
| Fuga de cookie/credenciales en logs o commits | Media | Alto | Sanitizar errores, no loguear headers, tests sintéticos, revisar `git diff` antes de commitear |
| El upsert por mes colapsa varias facturas de ASSA | Media | Medio | Evaluar REQ-014 antes de reactivar; si se requiere, spec separada para cardinalidad |
| El módulo privado no se puede bajar en CI | Baja | Alto | Verificar `P40LA_PLUGINS_TOKEN`/`GOPRIVATE` antes del release (SPEC-112) |
| El navegador headless tienta como atajo | Media | Alto | ADR-002: prohibido en iHost; documentar el costo y los términos de uso |
| La UI muestra la cookie o la devuelve en un listado | Baja | Alto | Campo `Secret`, `reveal` con contraseña, tests de no exposición |

## 8. Notas y Referencias

- Portal legacy: `https://miassanet.assanet.com/ni/miassanet-NI/login_v.aspx` (hoy redirige).
- Portal nuevo: `https://miassanet.assanet.com/ni/MiAssan-NI/login`.
- Plugin: `github.com/paulomcnally/p40la-ihost-automation-plugins`, `assa/nicaragua/nicaragua.go` v1.1.0.
- Contrato: `plugins/types.go` (`ErrAuthFailed`, `ErrAuthExpired`, `ErrServiceNotFound`, `ErrAPISchemaChanged`, `ErrUpstream`).
- App: `internal/services/automation_sync.go`, `automation_scheduler.go`, `internal/api/automation_handlers.go`, `frontend/src/pages/AutomationDetailPage.tsx`.
- Specs relacionadas: SPEC-040 (analizador PDF de ASSA), SPEC-092 (sync manual), SPEC-108/109/110/111/112 (unificación automation), SPEC-069/071/081 (webhooks/upsert).
- Repo privado de automation (histórico): SPEC-014 (plugin ASSA) y SPEC-016 (fechas), en `p40la-ihost-automation`.
- Evidencia de la investigación: artefactos HTTP sanitizados en `/tmp/opencode/assanet/` (no se commitean).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-03 | opencode | Creación inicial de la especificación. Se documenta la migración del portal ASSA, la causa raíz del error `login sin token var`, la evidencia del portal nuevo con reCAPTCHA y las opciones de recuperación (sesión asistida, API oficial, manual/PDF). |
| 2026-11-08 | opencode | Estado `released` (MVP P0). Se libera la detección de migración/CAPTCHA, los centinelas `ErrPortalMigrated`/`ErrCaptchaRequired`, el mapeo de mensajes accionables en app/UI, los tests de regresión y el plugin `v1.1.1` (commit `6cb1ebb`, merge `b5b8bf6`, imagen `v0.5.5` con los tres manifests verificados). La Fase 2/3 (sesión asistida, parsers del portal MVC) queda cubierta por SPEC-129. |
