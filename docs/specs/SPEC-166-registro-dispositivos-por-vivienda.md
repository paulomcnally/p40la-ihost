---
title: "Registro y módulos de dispositivos por vivienda"
id: "SPEC-166"
status: "draft"
author: "opencode"
created: "2026-11-27"
updated: "2026-11-27"
github_issue: 168
---

# Registro y módulos de dispositivos por vivienda

**ID**: SPEC-166
**Estado**: draft
**Autor**: opencode
**Creado**: 2026-11-27
**Actualizado**: 2026-11-27

---

## 1. Resumen Ejecutivo

Hoy una vivienda se puede describir, alquilar, mantener y reportar, pero **no existe un inventario de sus dispositivos**. El usuario no puede registrar que en la casa hay un televisor, una cafetera, un apagador, una luz SonOff, un Echo, un refrigerador, un microondas o una lavadora, ni conservar sus datos de marca, modelo, año y especificaciones técnicas. La solicitud es explícita: registrar dispositivos **desde un dropdown con la opción «Agregar»**, sin exigir un catálogo cerrado, con una lista inicial de categorías comunes y con espacio para valores propios.

Esta spec separa dos etapas con una frontera deliberada:

1. **Etapa 1 (esta spec, P0/P1): registro y administración.** CRUD de dispositivos por vivienda, catálogo abierto de tipos (semillas + tipos creados por el usuario), nombre identificable, marca, modelo, año, datos técnicos flexibles, capacidades **declaradas** y metadatos de integración **no secretos**. Nada de control, credenciales, descubrimiento ni llamadas de red a dispositivos.
2. **Etapa 2 (spec futura, solo diseñada acá): módulos de control.** Un contrato versionado `device-control/v1` con manifiesto, drivers, capabilities, salud y un panel administrativo de módulos. Se documenta la arquitectura objetivo para no cerrar puertas, pero **no se implementa**.

La decisión de producto más importante: **el registro no sabe controlar nada**. El dispositivo es una entidad de inventario; el control es un módulo enchufable posterior. Esto evita el riesgo más caro (credenciales en claro, SSRF, ejecución de código de terceros en el iHost) mientras entrega valor inmediato y deja el modelo preparado.

**iHost**: cero dependencias nuevas, cero llamadas de red, cero procesos extra. Una migración aditiva de cuatro tablas pequeñas con seeds idempotentes, índices por `home_id` y UI con los componentes obligatorios (`Select` con `addOption`, `NumberInput`, `CardMenu`, `DeleteModal`, i18n es/en, darkmode, detalle de solo lectura).

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables (índice: `docs/rules/00-index.md`). Las reglas no se copian: se citan por ID.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `net/http` + SQLite; React/Tailwind en build. Sin librerías nuevas. | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Módulo nuevo en `internal/modules/home/device/`; frontend en `pages/` y `components/home/`. | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | `handlers.go` solo parsea HTTP, `service.go` valida/decide, `storage.go` único con SQL, `model.go` structs. | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | Módulo `home.device` con los 5 archivos obligatorios + tests, registrado en `internal/modules/all/register.go`; rutas montadas por el propio módulo. | `CA-ARCH-MODULES` |
| `ARCH-GO` | `gofmt`, nombres de código en inglés, errores explícitos. | `CA-ARCH-GO` |
| `ARCH-IHOST` | Migración aditiva, índices por `home_id`, sin cron, sin red, sin proceso extra. | `CA-ARCH-IHOST` |
| `UI-SELECT` | Tipo, protocolo, filtros y opción «Agregar» usan `Select` (searchable donde hay muchas opciones); no hay `<select>` nativo. | `CA-UI-SELECT` |
| `UI-SELECT-ADD` | La opción de crear tipo dentro del dropdown se rotula `app.add` («Agregar») con ícono `plus`, al final; el modal sí se titula «Nuevo tipo de dispositivo». | `CA-UI-SELECT-ADD` |
| `UI-NUMBER-INPUT` | El año usa `NumberInput`; no hay `<input type="number">`. | `CA-UI-NUMBER-INPUT` |
| `UI-INPUT-TOKENS` | Inputs, textarea y Select usan `bg-card`/`text-text`/`text-text-secondary`; sin colores hardcodeados. | `CA-UI-TOKENS` |
| `UI-DARKMODE` | Listado, formulario, detalle y modal de tipo se verifican en darkmode. | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | Las capabilities declaradas se activan con `Toggle` por fila; no se usan checkboxes nativos. | `CA-UI-TOGGLE` |
| `UI-LIST-PATTERN` | Listado en cards con `CardMenu`; sin dispositivos muestra `EmptyCard` con acción. | `CA-UI-LIST-PATTERN` |
| `UI-CREATE-MENU` | `N/A`: el alta vive dentro del tab «Dispositivos» de la vivienda, no en el header de un listado raíz (precedente SPEC-164). | — |
| `UI-EDIT-LABEL` | La acción de edición usa `app.edit` («Editar»). | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | Claves nuevas solo en `frontend/public/i18n/{es,en}.json`, con paridad; los tipos del sistema se resuelven con un mapa estático de claves, no con objetos padre. | `CA-UI-I18N` |
| `UI-STYLE` | Cards `rounded-ios shadow-ios`, sidebar existente, menú de acciones, confirmación de borrado. | `CA-UI-STYLE` |
| `UX-BACK` | Rutas de formulario/detalle registradas en `BACK_ROUTES` con retorno a `/home/:id?tab=devices`. | `CA-UX-BACK` |
| `UX-MODAL` | Modal de tipo a `z-[60]` (no hay modal padre) y `DeleteModal` a `z-[80]`, sin cierre por click fuera, con cierre visible. | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | El borrado de dispositivo usa `DeleteModal` con palabra `confirmo` y Enter. | `CA-UX-MODAL-CONFIRM` |
| `UX-DETAIL-ACTIONS` | En el detalle, Editar/Archivar/Eliminar viven en `CardMenu`, sin íconos sueltos. | `CA-UX-DETAIL-ACTIONS` |
| `UX-DETAIL-READONLY` | El detalle muestra los datos sin formularios; editar navega a `/edit`. | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | Vivienda y tipo se validan en backend y frontend; vivienda inexistente devuelve 404. | `CA-UX-PREREQ` |
| `UX-EMPTY` | Listado vacío con `EmptyCard` explicando qué es un dispositivo y con acción de alta. | `CA-UX-EMPTY` |
| `UX-LOADING` | Spinner/skeleton en listado, detalle y guardado; botones deshabilitados mientras guardan. | `CA-UX-LOADING` |
| `UX-FEEDBACK` | Toast de éxito/error al crear, editar, archivar y borrar; errores junto al campo. | `CA-UX-FEEDBACK` |
| `UX-FORMS` | Un Guardar y un Cancelar; validación bloquea el guardado; obligatorios marcados. | `CA-UX-FORMS` |
| `DATA-SQLITE` | Cuatro tablas con FKs, CHECK e índices; SQL crudo sin ORM. | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | `0063_home_devices.up.sql` y `.down.sql` reversibles, con seeds idempotentes; test dedicado. | `CA-DATA-MIGRATIONS` |
| `DATA-PRODCOPY` | `N/A`: la migración es aditiva y no transforma datos existentes; la validación contra copia aplica recién antes de `pending_release`, no a esta spec draft. | — |
| `DATA-NO-DROP` / `DATA-PROTECTED-SCHEMA` | `N/A`: no se toca `data/app.db`, ni `debts`/`debt_bills`, ni tablas existentes. | — |
| `SEC-NO-PII` | Spec, tests y fixtures usan solo datos sintéticos; no se guardan secretos ni se commitean DBs. | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Casa Demo`, `Luz Sala Demo`, `Proveedor Demo`, IP `192.0.2.10` y MAC `02:00:00:00:00:01`. | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` | `git diff --cached` revisado: sin PII, secretos, DBs ni exports. | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...` y guardrails de frontend antes del commit. | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario (cuando se implemente). | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | La implementación pregunta 🤖 antes de pruebas manuales y antes de cerrar. | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-166`, rama `feature/SPEC-166`. | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita; esta tarea solo pushea la rama de la spec. | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa con veredicto y validación de subagente. | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; sesión y veredicto citados en §6. | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-HOOKS` | Commits `SPEC-166: …` con hooks instalados, sin `--no-verify`. | `CA-DELIV-HOOKS` |
| `DELIV-DOCKER` / `DELIV-DEPLOY` | `N/A`: la spec no cambia Dockerfile ni despliega. | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Tab «Dispositivos» por vivienda.** Agregar un tab `devices` a `HomeShowPage` (`/home/:id?tab=devices`) que liste los dispositivos de la vivienda con nombre, tipo, marca/modelo/año, estado (`active`/`archived`) y acciones. La navegación del tab se agrega al arreglo existente de tabs y a `BACK_ROUTES`.
2. **REQ-002 — Alta desde dropdown con «Agregar».** En el formulario de alta, el campo **Tipo** usa `Select` con las categorías iniciales y, al final, la opción `addOption(t('app.add'))`. Al elegirla se abre un modal para crear un tipo propio sin salir del formulario; el tipo creado queda seleccionado y disponible para futuros dispositivos de cualquier vivienda.
3. **REQ-003 — Catálogo abierto con semillas.** La migración siembra categorías comunes (`light`, `switch`, `plug`, `tv`, `speaker`, `coffee_maker`, `refrigerator`, `microwave`, `washing_machine`, `air_conditioner`, `curtain`, `camera`, `sensor`, `lock`, `network`, `other`) con `is_system = 1`. El usuario puede agregar tipos propios (`is_system = 0`) con nombre libre; no se exige un catálogo cerrado ni se permite duplicar por nombre/slug.
4. **REQ-004 — Identificación del dispositivo.** Campos obligatorios: `name` (nombre identificable en la vivienda, ej. «Luz Sala») y `type_id`. Campos opcionales: `brand`, `model`, `year`, `location` (ambiente/ubicación libre), `notes`, `serial_number`. El nombre es único por vivienda entre dispositivos no archivados (comparación sin distinguir mayúsculas ni acentos).
5. **REQ-005 — Datos técnicos flexibles.** El dispositivo puede tener N atributos técnicos definidos por el usuario (`label`, `value`, `unit`), por ejemplo `Voltaje=110 V`, `Potencia=1200 W`, `Capacidad=1.5 L`, `Conectividad=WiFi`. Límites: 30 atributos, `label` ≤ 60, `value` ≤ 200, `unit` ≤ 20 caracteres. Se editan como filas con agregar/quitar; no hay esquema fijo por tipo.
6. **REQ-006 — Edición.** Editar todos los campos de identificación, técnicos y de integración, incluidos tipo, atributos y capabilities. El nombre sigue siendo obligatorio y único; el año, si se informa, debe ser un entero entre 1900 y el año actual + 1.
7. **REQ-007 — Archivado y restauración.** Un dispositivo `active` se puede archivar (`archived`) sin perder datos y sin borrar sus atributos; se puede restaurar a `active`. El listado filtra por estado y por defecto muestra activos y archivados con badges distintos.
8. **REQ-008 — Eliminación.** Eliminar un dispositivo lo borra en cascada con sus atributos y capabilities, previa confirmación por palabra (`DeleteModal`). La UI ofrece Archivar como alternativa no destructiva.
9. **REQ-009 — Detalle de solo lectura.** Página de detalle con secciones: identificación (nombre, tipo, marca, modelo, año, ubicación, estado), datos técnicos, capabilities declaradas y metadatos de integración. Sin formularios; Editar/Archivar/Eliminar en el `CardMenu`.
10. **REQ-010 — Búsqueda y filtros.** El listado permite buscar por nombre, marca, modelo o número de serie, y filtrar por tipo y por estado. Devuelve contadores `total`, `active_count` y `archived_count`.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011 — Capabilities declaradas.** Cada dispositivo guarda una lista de capabilities de un vocabulario fijo (`power.onoff`, `power.toggle`, `brightness`, `colorTemperature`, `colorRGB`, `percentage`, `position`, `volume`, `mute`, `input.source`, `media.playback`, `media.app`, `temperature`, `humidity`, `battery`, `rssi`, `energy.power`, `sensor.contact`, `sensor.motion`, `sensor.leak`, `sensor.smoke`, `lock`). Cada fila tiene `source` (`template` | `manual`); la etapa 1 **nunca** usa `observed`. Un tipo tiene una plantilla por defecto; el usuario puede activar/desactivar capabilities con `Toggle`.
2. **REQ-012 — Metadatos de integración no secretos.** Campos opcionales `protocol` (`manual`, `ewelink-ihost`, `sony-bravia`, `sonoff-diy`, `matter`, `other`), `host` (IP o hostname), `mac` y `integration_ref` (referencia opaca, ej. número de serie del hub). **No se guardan tokens, PSK, contraseñas ni claves.** Estos campos son solo metadata: la etapa 1 no resuelve DNS, no abre conexiones y no descubre dispositivos.
3. **REQ-013 — Unicidad de binding.** Cuando se informan, `serial_number` y `mac` son únicos por vivienda entre dispositivos activos; un duplicado devuelve `409` con código `serial_taken` o `mac_taken`. El `name` duplicado devuelve `409 device_name_taken`.
4. **REQ-014 — Resumen en el tab.** El tab muestra el total de dispositivos y, si hay archivados, un acceso para verlos; el badge de estado diferencia `active` de `archived`.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015 — Página global de dispositivos.** Un listado transversal `/devices` con filtro por vivienda, tipo y estado, reutilizando el mismo módulo. No es necesario para el MVP.
2. **REQ-016 — Administración del catálogo.** Página en `Settings` para renombrar/eliminar tipos propios no usados y reordenar la lista; los tipos del sistema no se eliminan.
3. **REQ-017 — Import/export.** Exportar e importar el inventario de una vivienda en JSON/CSV con validación y vista previa.
4. **REQ-018 — Foto del dispositivo.** Adjuntar una imagen de referencia (no evidencia de falla) siguiendo el patrón de adjuntos de SPEC-164.

### 3.4 Fuera de alcance explícito de esta spec (etapa 2)

- Ejecutar comandos de control (`on`/`off`, volumen, entrada HDMI, etc.) o cualquier `Execute`.
- Drivers, interfaces `Driver`/`Device`, registry de módulos, manifiestos instalables o panel de plugins.
- Descubrimiento SSDP/mDNS/DHCP, pairing Zigbee, commissioning Matter, `actRegister`/PIN de Bravia o importación automática de `GET /devices` del iHost.
- Guardar PSK, tokens de iHost/eWeLink, `appSecret`, `devicekey` o cualquier secreto.
- Polling, SSE/WebSocket, colas de eventos, histórico de estados, health checks activos o «probar conexión».
- Automatizaciones, escenas, horarios, control remoto fuera de la LAN, Alexa/Google, métricas de energía.
- Convertir la app en hub Matter ni reemplazar al iHost.

La arquitectura objetivo de la etapa 2 está diseñada en §5.6 para que el modelo de datos y el vocabulario no se rompan después.

### 3.5 Requerimientos No Funcionales

- **Rendimiento**: listar 200 dispositivos de una vivienda en < 100 ms; el listado no carga atributos ni capabilities uno por uno (solo el detalle los trae).
- **Memoria**: el listado se limita (máx. 500) y no mantiene capabilities de todos los dispositivos en memoria.
- **Almacenamiento**: cuatro tablas nuevas + índices; los seeds son 16 filas; sin archivos binarios.
- **Seguridad**: autenticación obligatoria en todas las rutas; toda consulta filtra por `home_id`; sin secretos; sin red saliente; validación de formato de año, MAC y host sin resolver DNS.
- **iHost**: sin dependencias nuevas, sin cron, sin llamadas de red, sin FFmpeg ni procesos auxiliares.
- **i18n**: paridad es/en; etiquetas de tipos del sistema por claves estáticas; tipos propios muestran su nombre.
- **Accesibilidad**: controles con `min-h-[44px]`, foco visible, navegación por teclado en `Select` y modales.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

**Repo (subagente `explore`, informe integrado)**:
- `internal/modules/home/maintenance/` (SPEC-141) y `internal/modules/home/workrequest/` (SPEC-164) son el patrón canónico: módulo namespaced con `module.go`, `model.go`, `storage.go`, `service.go`, `handlers.go`, rutas montadas por `RegisterRoutes`, migración propia y UI en tab de la vivienda.
- `homes` usa `id INTEGER AUTOINCREMENT` y borrado lógico `deleted_at` (`migrations/0003_create_homes.up.sql`). Las entidades hijas usan FK `home_id ... ON DELETE CASCADE` e índices por `home_id` (ver `0062_home_work_requests.up.sql`).
- El frontend de vivienda ya usa tabs por query string (`?tab=summary|rental|finances|maintenance|reports`), `Select`, `CardMenu`, `DeleteModal`, `EmptyCard`, `NumberInput`, `Field`/`TextInput`/`TextArea` de `components/rental/RentalUI.tsx`, y el patrón de formulario/detalle de `HomeWorkRequestFormPage`/`HomeWorkRequestShowPage`.
- `frontend/src/components/DashboardLayout.tsx` centraliza `BACK_ROUTES`; `App.tsx` declara las rutas; `frontend/src/api/index.ts` agrupa APIs por dominio; `frontend/src/types/` contiene los tipos.
- No existe hoy ninguna entidad, tabla, componente ni tool MCP de dispositivos, plugins o integraciones. `grep` de `device|dispositivo|ewelink|bravia|plugin|integration` no devuelve código de producto.
- **Autenticación y alcance**: hay un único usuario administrador (`internal/models/user.go`); `homes` no tiene `user_id` y no hay roles ni multi-tenant. Todas las viviendas son visibles para el administrador. La feature no introduce ownership por usuario.
- **Borrado lógico de viviendas**: `HomeStorage.SoftDelete` solo setea `deleted_at`; la FK `ON DELETE CASCADE` no se dispara al “eliminar” una vivienda. Los dispositivos quedarían huérfanos en la DB (ocultos por `requireHome`, que filtra `deleted_at IS NULL`). Se acepta el mismo comportamiento que `home.workrequest` y se documenta como riesgo; no se toca el servicio legacy para no ampliar el impacto MCP.
- **Componentes reutilizables**: `Select`, `NumberInput`, `CardMenu`, `DeleteModal`, `Toggle`, `Field`, `Modal` y `SectionCard` son compartidos; **`EmptyCard` no lo es** (está definido localmente en `HomesPage.tsx`). El tab de dispositivos replica ese patrón local; no se refactoriza `HomesPage` en esta spec.
- **Helpers HTTP por módulo**: `pathID`, `decodeJSON`, `respondJSON` y `respondError` se duplican en cada módulo (`workrequest`, `maintenance`). El módulo `home.device` los replica en su propio `handlers.go`; no se crea un helper compartido para no tocar código existente.
- **Precedente de catálogo**: `institution_categories` usa `key UNIQUE`, `name`, `icon_key`, `is_system`, `sort_order`; es el modelo más cercano para el catálogo abierto de tipos. `automation_plugins` (catálogo compilado) es el precedente para el panel de módulos de la etapa 2.

**Investigación externa (subagente `general`, verificado)**:
- **iHost Open API V2**: API local documentada (`http://<ihost>/open-api/v2/rest/...`) con `GET /devices` y `PUT /devices/{serial}`; autenticación por `GET /open-api/v2/rest/bridge/access_token?app_name=...` + clic físico en «Done», token Bearer de larga duración; push por SSE. Modelo de `capabilities` con bitmap de permisos (`read`/`write`/`report`/`control`). Documentación: <https://ewelink.cc/ewelink-cube/introduce-open-api/document> y <https://help.sonoff.tech/docs/iHost-API>.
- **Taxonomía del iHost insuficiente**: `display_category` no incluye TV, cafetera, refrigerador, microondas, lavadora ni parlante; sólo `plug`, `switch`, `light`, sensores, clima, cortina, cámara, etc. Por eso la app necesita **su propia** taxonomía y el iHost es un *binding* futuro, no el modelo.
- **`ewelink-mcp`** (<https://github.com/paulomcnally/ewelink-mcp>): prototipo Python de 1 commit, sin licencia ni tests; expone 2 tools (`list_devices`, `control_device`) por stdio; configuración global por `IHOST_URL`/`IHOST_TOKEN`. Concepto reutilizable: cliente por protocolo aislado y mapeo `capability → ruta JSON de estado`. No reutilizar código (sin licencia) ni el bus MCP como bus interno.
- **`sony-bravia-mcp`** (<https://github.com/paulomcnally/sony-bravia-mcp>): prototipo Python de 1 commit, sin licencia ni tests; 15 tools sobre JSON-RPC `/sony/{service}` + IRCC SOAP, auth `X-Auth-PSK`, sin descubrimiento. Concepto reutilizable: driver con métodos por capability y tabla declarativa de comandos. No reutilizar código ni hardcodear versiones de API: hay que negociar por modelo.
- **Patrones de industria**: Home Assistant separa manifiesto declarativo, *config entry* y *coordinator* con `unavailable`/backoff/reautenticación; Matter modela `Node → Endpoint → Cluster → Attribute/Command` con device types que sí cubren refrigerador, microondas, lavadora y parlante; eWeLink CUBE define acceso de dispositivos de terceros con `DiscoveryRequest`/`UpdateDeviceStates`/`DeviceStatesChangeReport` y un `service_address` webhook. La spec toma el vocabulario de capabilities de iHost, la idea de manifiesto de HA y el aislamiento por proceso de CUBE, sin implementarlos ahora.
- **Seguridad**: los riesgos propios de la etapa 2 son credenciales en claro, HTTP plano en LAN, token en query string (SSE), SSRF por `service_address`/IP editable, RCE por plugins y autorización horizontal entre viviendas. Se documentan en §9 con mitigaciones; la etapa 1 los evita por diseño.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Catálogo cerrado de tipos (CHECK/enum) | Simple, consistente | El usuario pidió valores propios; cada tipo nuevo exige migración | ❌ Rechazada |
| Tipo como texto libre sin catálogo | Máxima flexibilidad | Duplicados («TV», «Tv», «Televisor»), filtros sucios, sin reutilización | ❌ Rechazada |
| Catálogo en tabla con seeds + tipos propios (`home_device_types`) | Lista inicial útil, valores propios, filtros limpios, reutilizable entre viviendas | Una tabla y un endpoint extra | ✅ Seleccionada |
| Control de dispositivos en la misma etapa | Valor completo de inmediato | Credenciales, red, drivers, SSRF, RCE y enorme superficie; el usuario lo planteó como segunda etapa | ❌ Rechazada |
| Registro ahora + contrato de módulos diseñado para después | Entrega valor sin riesgo; no cierra puertas | Hay que documentar la etapa 2 y evitar prometer control | ✅ Seleccionada |
| Datos técnicos en JSON (`attributes_json`) | Menos tablas | Difícil filtrar/validar; parseo en listados | ❌ Rechazada |
| Datos técnicos normalizados (`home_device_attributes`) | Validación por fila, orden, editable, consultable | Una tabla hija | ✅ Seleccionada |
| Capabilities como JSON | Menos tablas | Sin unicidad ni consulta por capability | ❌ Rechazada |
| Capabilities normalizadas (`home_device_capabilities`) | Unicidad por dispositivo, consulta para etapa 2, `source` explícito | Una tabla hija | ✅ Seleccionada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Registro y control son capas separadas.**
- **Contexto**: el usuario pidió registrar dispositivos ahora y, como segunda etapa, lógica propia para controlarlos. Mezclar ambas etapas obliga a guardar secretos y abrir red desde el día uno.
- **Decisión**: la etapa 1 es CRUD de inventario sin secretos, sin descubrimiento y sin I/O de red. La etapa 2 se especifica aparte con un contrato `device-control/v1`.
- **Consecuencias**: el registro es seguro y útil de inmediato; el modelo guarda `protocol`, `host`, `mac`, `serial_number`, `integration_ref` y capabilities declaradas para que la etapa 2 solo agregue runtime, no migre el inventario.

**ADR-002 — Catálogo abierto con semillas y tipos del sistema inmutables.**
- **Contexto**: el usuario no quiere una lista predefinida obligatoria, pero sí una lista inicial útil.
- **Decisión**: tabla `home_device_types` con 16 seeds `is_system = 1` y tipos propios `is_system = 0`. Los tipos del sistema se traducen por i18n y no se eliminan; los propios se crean desde el dropdown con «Agregar».
- **Consecuencias**: el dropdown arranca poblado y crece sin migraciones. Los duplicados se evitan por `slug` único y comparación normalizada de nombre.

**ADR-003 — Taxonomía propia, no la del iHost.**
- **Contexto**: `display_category` del iHost no modela TV ni electrodomésticos; adoptarla dejaría fuera la mayoría de los ejemplos del usuario.
- **Decisión**: taxonomía propia orientada a vivienda. El mapeo a iHost/Bravia/Matter será parte del manifiesto de cada módulo en la etapa 2.
- **Consecuencias**: la app puede representar todos los ejemplos; la interoperabilidad futura requiere una tabla de mapeo por driver, que es explícita y versionable.

**ADR-004 — Datos técnicos y capabilities normalizados.**
- **Contexto**: «los datos técnicos que correspondan» varían por dispositivo; las capabilities son un contrato que la etapa 2 va a consultar.
- **Decisión**: `home_device_attributes` (label/value/unit/sort_order) y `home_device_capabilities` (capability/source, PK compuesta). Sin JSON en la etapa 1.
- **Consecuencias**: validación y edición por fila; la etapa 2 puede consultar «todos los dispositivos con `power.onoff`» con SQL. El costo es una transacción de reemplazo al editar.

**ADR-005 — Metadatos de integración no secretos en la etapa 1.**
- **Contexto**: para vincular «Luz Sala» con su serial de iHost hace falta guardar identificadores, pero un token/PSK no puede vivir en una columna de texto.
- **Decisión**: se guardan `protocol`, `host`, `mac`, `serial_number` e `integration_ref`; **ningún secreto**. La etapa 2 define un almacén de secretos cifrado y separado (`device_secrets`, AEAD, clave fuera de la DB), con rotación y redacción en logs.
- **Consecuencias**: la etapa 1 no puede autenticarse contra un dispositivo, que es exactamente el objetivo. Si en el futuro se guarda un secreto, será en una spec con revisión de seguridad.

**ADR-006 — Contrato de módulos versionado e isomorfo a iHost.**
- **Contexto**: hay que dejar planteada la arquitectura de plugins sin implementarla.
- **Decisión**: la etapa 2 usará un manifiesto declarativo (`Manifest` con `SchemaVersion`, `ID`, `Version`, `Contract: "device-control/v1"`, `Capabilities`, `Limits`, `Permissions`) y un runtime mínimo (`Driver`/`Device` con `Read`, `Execute`, `Subscribe`, `Health`). Las capabilities usan el vocabulario de iHost; el `service_address` y los comandos son isomorfos al protocolo de terceros de CUBE.
- **Consecuencias**: los drivers first-party pueden ser in-process y los de terceros sidecar; el registry rechaza un `contract` mayor incompatible. No se construye el runtime hasta que exista el segundo driver.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura (etapa 1)

```
HomeShowPage (?tab=devices)
        │
        ▼
HomeDeviceTab ──► api.homeDevice ──► GET/POST/PUT/DELETE /api/homes/{id}/devices
        │                                   │
        ├──► HomeDeviceFormPage             ▼
        │      └── Select + addOption ──► /api/device-types
        │                                   │
        └──► HomeDeviceShowPage             ▼
                                      home.device module
                                   handlers → service → storage
                                             │
                                             ▼
                       home_devices / home_device_attributes /
                       home_device_capabilities / home_device_types
```

### 5.2 Componentes

#### 5.2.1 Backend `internal/modules/home/device/`
- **Responsabilidad**: CRUD de dispositivos por vivienda y catálogo de tipos.
- **Interfaz**: `Module.Info()` (`ID: "home.device"`, `Version: "1.0.0"`, `Kind: "core"`, `Migrations: ["0063_home_devices"]`), `Module.RegisterRoutes(mux, auth)`, `Module.Service()`; implementa `modules.Router`.
- **Dependencias**: `*sql.DB`, `modules.Info`, `net/http`; sin servicios de red.
- **Ubicación**: `internal/modules/home/device/` con `module.go`, `model.go`, `storage.go`, `service.go`, `handlers.go` y tests.
- **Registro**: `internal/modules/all/register.go` agrega `registry.Register(device.New(db))` (no toca `cmd/server/main.go`).
- **Helpers HTTP**: `pathID`, `decodeJSON`, `respondJSON`, `respondError` se replican localmente (patrón de `workrequest`); no se crea un helper compartido.

#### 5.2.2 Frontend
- `frontend/src/components/home/HomeDeviceTab.tsx`: listado, búsqueda, filtros, acciones y estado vacío (replica el patrón local de `EmptyCard` de `HomesPage.tsx`; no se extrae un componente compartido).
- `frontend/src/pages/HomeDeviceFormPage.tsx`: alta/edición con secciones y `Select` con «Agregar».
- `frontend/src/pages/HomeDeviceShowPage.tsx`: detalle de solo lectura.
- `frontend/src/components/home/DeviceTypeModal.tsx`: alta de tipo propio desde el dropdown.
- `frontend/src/api/homeDevice.ts`, `frontend/src/types/homeDevice.ts`: contrato tipado.
- `frontend/src/App.tsx`, `frontend/src/components/DashboardLayout.tsx`, `frontend/src/pages/HomeShowPage.tsx`, `frontend/public/i18n/{es,en}.json`.

### 5.3 Modelo de datos

Migración `migrations/0063_home_devices.up.sql` (aditiva; `.down.sql` elimina en orden inverso).

```sql
CREATE TABLE home_device_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT 'other',
    is_system INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE home_devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    type_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    name_normalized TEXT NOT NULL,
    brand TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    year INTEGER,
    serial_number TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    protocol TEXT NOT NULL DEFAULT 'manual'
        CHECK (protocol IN ('manual','ewelink-ihost','sony-bravia','sonoff-diy','matter','other')),
    host TEXT NOT NULL DEFAULT '',
    mac TEXT NOT NULL DEFAULT '',
    integration_ref TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','archived')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES home_device_types(id)
);

CREATE INDEX idx_home_devices_home_status ON home_devices(home_id, status, name);
CREATE INDEX idx_home_devices_type ON home_devices(home_id, type_id);
CREATE UNIQUE INDEX ux_home_devices_name
    ON home_devices(home_id, name_normalized)
    WHERE status = 'active';
CREATE UNIQUE INDEX ux_home_devices_serial
    ON home_devices(home_id, serial_number)
    WHERE serial_number <> '' AND status = 'active';
CREATE UNIQUE INDEX ux_home_devices_mac
    ON home_devices(home_id, mac)
    WHERE mac <> '' AND status = 'active';

CREATE TABLE home_device_attributes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    label TEXT NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    unit TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (device_id) REFERENCES home_devices(id) ON DELETE CASCADE
);
CREATE INDEX idx_home_device_attributes_owner
    ON home_device_attributes(device_id, sort_order, id);

CREATE TABLE home_device_capabilities (
    device_id INTEGER NOT NULL,
    capability TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'template'
        CHECK (source IN ('template','manual','observed')),
    PRIMARY KEY (device_id, capability),
    FOREIGN KEY (device_id) REFERENCES home_devices(id) ON DELETE CASCADE
);
```

**Seeds idempotentes** (`INSERT OR IGNORE`) para los 16 tipos del sistema, con `slug`, `name` en español como fallback, `icon` de `Icons.tsx` (ej. `light→bolt`, `switch→bolt`, `plug→bolt`, `tv→tv`, `speaker→radio`, `coffee_maker→coffee`, `refrigerator→other`, `microwave→flame`, `washing_machine→laundry`, `air_conditioner→signal`, `curtain→window`, `camera→camera`, `sensor→alarm`, `lock→lock`, `network→wifi`, `other→other`) y `sort_order` 10..160.

**Entidades**:
- `DeviceType`: `id`, `slug`, `name`, `icon`, `is_system`, `sort_order`.
- `Device`: `id`, `home_id`, `type_id`, `type` (embebido en detalle), `name`, `name_normalized` (interno, no se expone en la API), `brand`, `model`, `year` (`*int`), `serial_number`, `location`, `notes`, `protocol`, `host`, `mac`, `integration_ref`, `status`, `attributes[]`, `capabilities[]`, `created_at`, `updated_at`. El listado ordena `status ASC, name ASC` (activos primero).
- `Attribute`: `id`, `device_id`, `label`, `value`, `unit`, `sort_order`.
- `Capability`: `capability`, `source`.

**Nota de integridad**: `home_devices` no tiene `deleted_at`; el archived es explícito y el DELETE es físico con cascada. La unicidad de nombre/serial/MAC sólo aplica a `status='active'` para permitir reemplazar un equipo archivado; restaurar valida conflictos y devuelve 409. **Nota de borrado de vivienda**: como `homes` usa soft delete, la cascada de la FK no se dispara al eliminar lógicamente una vivienda; los dispositivos quedan huérfanos pero inaccesibles porque `requireHome` filtra `deleted_at IS NULL`. Se acepta y documenta (mismo comportamiento que `home.workrequest`).

### 5.4 APIs / Contratos

Todas las rutas requieren auth (middleware existente) y filtran por vivienda.

#### `GET /api/device-types`
**Response 200**:
```json
{"items":[{"id":1,"slug":"light","name":"Luz","icon":"bolt","is_system":true,"sort_order":10}]}
```

#### `POST /api/device-types`
**Request**:
```json
{"name":"Purificador de aire","icon":"other"}
```
**Response 201**:
```json
{"id":17,"slug":"purificador-de-aire","name":"Purificador de aire","icon":"other","is_system":false,"sort_order":0}
```
**Errores**: `400 invalid_request`; `409 type_exists` (nombre/slug normalizado duplicado).

#### `GET /api/homes/{id}/devices?status=&type_id=&q=&limit=`
**Response 200**:
```json
{
  "items":[{
    "id":10,"home_id":3,"type_id":1,
    "type":{"id":1,"slug":"light","name":"Luz","icon":"bolt","is_system":true,"sort_order":10},
    "name":"Luz Sala Demo","brand":"Proveedor Demo","model":"Modelo Demo",
    "year":2024,"serial_number":"demo-serial-001","location":"Sala",
    "notes":"","protocol":"ewelink-ihost","host":"192.0.2.10",
    "mac":"02:00:00:00:00:01","integration_ref":"demo-hub-001",
    "status":"active","attribute_count":2,"capability_count":3,
    "created_at":"2026-11-27T00:00:00Z","updated_at":"2026-11-27T00:00:00Z"
  }],
  "total":1,"active_count":1,"archived_count":0
}
```
**Errores**: `400 invalid_id`; `404 home_not_found`.

#### `POST /api/homes/{id}/devices`
**Request**:
```json
{
  "name":"Luz Sala Demo","type_id":1,"brand":"Proveedor Demo","model":"Modelo Demo",
  "year":2024,"serial_number":"demo-serial-001","location":"Sala","notes":"",
  "protocol":"ewelink-ihost","host":"192.0.2.10","mac":"02:00:00:00:00:01",
  "integration_ref":"demo-hub-001",
  "attributes":[
    {"label":"Voltaje","value":"110","unit":"V"},
    {"label":"Potencia","value":"9","unit":"W"}
  ],
  "capabilities":[
    {"capability":"power.onoff","source":"template"},
    {"capability":"brightness","source":"manual"}
  ]
}
```
**Response 201**: el detalle completo con `id`, `type`, `attributes`, `capabilities`.
**Errores**: `400 invalid_request`, `invalid_type`, `invalid_year`, `invalid_mac`, `invalid_host`, `invalid_capability`, `too_many_attributes`; `404 home_not_found`; `409 device_name_taken`, `serial_taken`, `mac_taken`.

#### `GET /api/homes/{id}/devices/{device_id}`
**Response 200**: detalle con `type`, `attributes[]` y `capabilities[]`.
**Errores**: `404 device_not_found` (también si el dispositivo pertenece a otra vivienda).

#### `PUT /api/homes/{id}/devices/{device_id}`
Mismo body que el POST; reemplaza atributos y capabilities en una transacción. **Response 200** con el detalle.
**Errores**: los mismos del POST más `404 device_not_found`.

#### `POST /api/homes/{id}/devices/{device_id}/status`
**Request**: `{"status":"archived"}` o `{"status":"active"}`.
**Response 200**: el detalle actualizado. **Errores**: `400 invalid_status`; `404 device_not_found`; al restaurar, `409 device_name_taken`/`serial_taken`/`mac_taken` si otro dispositivo activo ya usa esos valores (la unicidad solo aplica a `active`).

#### `DELETE /api/homes/{id}/devices/{device_id}`
**Response 204**. Borra atributos y capabilities por cascada. **Errores**: `404 device_not_found`.

**Códigos de error** (formato `{error, message}`): `invalid_id`, `invalid_request`, `home_not_found`, `device_not_found`, `invalid_type`, `type_exists`, `invalid_year`, `invalid_mac`, `invalid_host`, `invalid_capability`, `too_many_attributes`, `device_name_taken`, `serial_taken`, `mac_taken`, `invalid_status`.

### 5.5 Validaciones

- `name`: trim, 1..120; el service calcula `name_normalized` (minúsculas y sin diacríticos) y lo persiste; el índice único parcial `(home_id, name_normalized) WHERE status='active'` garantiza unicidad por vivienda entre no archivados.
- `brand`/`model`/`location`: ≤ 80; `notes` ≤ 1000; `serial_number`/`host`/`mac`/`integration_ref` ≤ 120.
- `year`: `NULL` o entero `1900..añoActual+1`; no se acepta texto.
- `mac`: si no está vacío, `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`.
- `host`: si no está vacío, IPv4 válida o hostname `^[a-zA-Z0-9]([a-zA-Z0-9.-]{0,118}[a-zA-Z0-9])?$`; **sin resolver DNS**.
- `protocol`: uno de los valores del CHECK.
- atributos: máximo 30; `label` obligatorio y ≤ 60; `value` ≤ 200; `unit` ≤ 20; se descartan filas totalmente vacías; `sort_order` por índice.
- capabilities: máximo 22 (el vocabulario), sin duplicados; `source` `template|manual` en la API de escritura (`observed` reservado para etapa 2).
- tipo: debe existir; no se puede asignar un tipo eliminado (no hay borrado de tipos en P0).

### 5.6 Arquitectura objetivo etapa 2 (diseñada, no implementada)

**Contrato `device-control/v1`** (Go, en `internal/modules/devicecontrol/` si se implementa):

```go
type Manifest struct {
    SchemaVersion string   // "1"
    ID            string   // "ewelink-ihost"
    Name          string
    Version       string   // semver del módulo
    Contract      string   // "device-control/v1"
    Vendor        string
    Models        []string
    Transport     string   // ssdp|mdns|http|cloud|zigbee|none
    Capabilities  []CapabilitySpec
    Limits        Limits
    Permissions   []string // ej. "network:192.168.0.0/16"
}

type Driver interface {
    Manifest() Manifest
    Discover(ctx context.Context, o DiscoverOptions) ([]Candidate, error)
    Open(ctx context.Context, b Binding, s SecretReader) (Device, error)
}

type Device interface {
    Read(ctx context.Context, caps []CapabilityID) (map[CapabilityID]Value, error)
    Execute(ctx context.Context, cmd Command) (Result, error)
    Subscribe(ctx context.Context, fn func(Event)) (cancel func(), err error)
    Health(ctx context.Context) Health
    Close() error
}
```

- **Panel administrativo**: `/settings/device-modules` con lista de módulos instalados, versión, contrato, dispositivos vinculados, salud agregada, último error y «probar conexión».
- **Instalación**: validar manifiesto (JSON Schema + `contract` mayor compatible), guardar snapshot + hash + versión, habilitar, resolver driver, `Open()` por dispositivo; desinstalar cierra cada `Device`. Un `contract` mayor incompatible se rechaza.
- **Secretos**: tabla `device_secrets(device_id, kind, ciphertext, key_version, created_at, rotated_at)` con AEAD (AES-GCM), clave fuera de la DB, nunca devuelta por API, redactada en logs/diagnostics.
- **Salud**: `unconfigured → connecting → online ⇄ degraded → offline`, más `disabled` y `failed`; circuit breaker con backoff y jitter; log de transición una vez.
- **Aislamiento**: first-party in-process con recover/timeouts; terceros en sidecar HTTP/Unix socket con límites de CPU/RAM, rootfs read-only, sin privilegios y sin acceso a la DB.
- **SSRF**: allowlist de CIDR privadas, resolver y validar IP post-DNS, bloquear loopback/`169.254.169.254`/`file://`/redirects, timeouts.
- **Mapeo de taxonomía**: cada manifiesto declara a qué `kind`/capabilities mapea (iHost `light/switch/plug`, Bravia `tv`, Matter device types). El inventario no se migra.

Esta sección es un **contrato de intención**; su implementación requiere una spec propia con revisión de seguridad.

### 5.7 Archivos previstos

| Archivo | Acción |
|---|---|
| `migrations/0063_home_devices.up.sql` / `.down.sql` | Crear |
| `internal/db/migrate_063_test.go` | Crear |
| `internal/modules/home/device/{module,model,storage,service,handlers}.go` | Crear |
| `internal/modules/home/device/*_test.go` | Crear |
| `internal/modules/all/register.go` | Editar (una línea) |
| `frontend/src/components/home/HomeDeviceTab.tsx` | Crear |
| `frontend/src/components/home/DeviceTypeModal.tsx` | Crear |
| `frontend/src/pages/HomeDeviceFormPage.tsx` | Crear |
| `frontend/src/pages/HomeDeviceShowPage.tsx` | Crear |
| `frontend/src/api/homeDevice.ts`, `frontend/src/types/homeDevice.ts` | Crear |
| `frontend/src/api/index.ts` | Editar (export) |
| `frontend/src/App.tsx` | Editar (rutas) |
| `frontend/src/components/DashboardLayout.tsx` | Editar (`BACK_ROUTES`) |
| `frontend/src/pages/HomeShowPage.tsx` | Editar (tab `devices`) |
| `frontend/public/i18n/{es,en}.json` | Editar (claves nuevas) |

### 5.8 Dependencias

- **Internas**: `homes` (FK y validación de existencia con `HomeExists` filtrando `deleted_at IS NULL`), `modules.Registry`, middleware de auth (usuario administrador único, sin ownership por usuario), componentes `Select`/`NumberInput`/`CardMenu`/`DeleteModal`/`Toggle`/`Field` y el patrón local de `EmptyCard`.
- **Externas**: ninguna nueva. No se usan SDK de eWeLink, Sony ni MCP.

### 5.9 Supuestos (a validar por el usuario)

- **Un solo administrador.** No hay ownership por usuario; cualquier usuario autenticado ve todas las viviendas. Se mantiene el estado actual del producto.
- **Tipos globales.** Los tipos propios se comparten entre viviendas (catálogo único); no son por vivienda. Es lo que permite reutilizar «Cafetera» en otra casa.
- **Ubicación libre.** No se introduce una entidad «ambiente/habitación»; `location` es texto libre en la etapa 1. Una entidad `rooms` queda para una spec futura.
- **Borrado físico vs archivar.** El usuario puede archivar sin perder datos; el borrado es definitivo y confirmado. No hay papelera.
- **Sin imágenes en la etapa 1.** La foto del dispositivo es P2 y reutilizará el patrón de adjuntos de SPEC-164.
- **Control fuera de alcance.** Cualquier expectativa de encender/apagar «Luz Sala» pertenece a la etapa 2; la UI lo comunica explícitamente.

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). Análisis con `./scripts/check-mcp-impact.sh --files ...` y validación de un subagente (`DELIV-MCP-IMPACT-SUBAGENT`).

### 6.1 Veredicto

**Ninguna tool MCP afectada funcionalmente.** La feature:
- crea un módulo nuevo `internal/modules/home/device/` que no figura en `entrypoints` ni `dependencies` de ninguna tool;
- agrega la migración `0063_home_devices`, que **solo crea tablas nuevas** (`home_device_types`, `home_devices`, `home_device_attributes`, `home_device_capabilities`) y no altera `homes`, `home_maintenances`, `home_work_requests`, `currencies` ni `institutions`;
- solo edita `internal/modules/all/register.go` (fuera de la superficie MCP) además del frontend;
- **no** agrega tools MCP ni cambia contratos. Las tools de dispositivos, si se desean, serán una spec de seguimiento (mismo patrón que SPEC-164 → SPEC-165).

### 6.2 Tools nuevas

Ninguna. La etapa 1 es una feature de la app web. Un `home_device_list/create/get/update` MCP queda para una spec futura.

### 6.3 Tools existentes afectadas

Ninguna con cambio funcional ni de contrato. `docs/mcp/tools/contracts.json` no se modifica; `docs/mcp/` no requiere cambios.

### 6.4 Nota de guardrail: `shared.affects_all` por `migrations/`

El manifiesto declara `migrations/` en `shared.affects_all`. Al agregar `0063_*.sql`, `./scripts/check-mcp-impact.sh --files ...` marca técnicamente **59 tools** por recompilación de la BD compartida. El impacto real es **nulo** porque la migración es aditiva y aislada. Tools marcadas: `alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bank_accounts_create`, `bank_accounts_search`, `bank_accounts_update`, `banks_create`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `budget_accounts_list`, `budget_categories_list`, `budget_month_view`, `budget_transactions_create`, `budget_transactions_list`, `credit_card_statements_get`, `credit_card_statements_list`, `credit_cards_create`, `credit_cards_search`, `credit_cards_update`, `debt_bills_list`, `entities_search`, `home_maintenance_create`, `home_maintenance_get`, `home_maintenance_item_delete`, `home_maintenance_item_move`, `home_maintenance_item_update`, `home_maintenance_labor_register`, `home_maintenance_list`, `home_maintenance_supplies_register`, `home_maintenance_update`, `home_work_request_convert`, `home_work_request_create`, `home_work_request_get`, `home_work_request_list`, `home_work_request_status`, `home_work_request_update`, `homes_create`, `homes_update`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

### 6.5 Condición de diseño para mantener «ninguna afectada»

No tocar `internal/mcp/**`, `internal/api/routes.go`, `internal/api/handlers.go`, `internal/models/`, `internal/services/service.go`, `internal/storage/service.go` ni `internal/modules/home/{maintenance,workrequest}/`. Las rutas del módulo nuevo se registran vía su propio `RegisterRoutes`.

**Validación subagente:** explore — sesión `ses_ddfcb95cdffeCk2ncqAv7BiRVr` — veredicto: **ninguna tool MCP afectada funcionalmente**; el módulo `internal/modules/home/device/` no coincide con `entrypoints`/`dependencies` del manifiesto, `internal/modules/all/register.go` y `frontend/` no están en el manifiesto, y las 59 tools solo se marcan por `shared.affects_all: migrations/` (impacto real nulo con una migración aditiva y aislada). No se actualizan `impact.json`, `contracts.json` ni `internal/mcp/**`.

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

> IDs canónicos (`CA-<PREFIJO>-<NOMBRE>`) para auditoría con grep.

### 7.1 Funcionales

- [ ] `CA-DEV-001`: Desde `/home/:id?tab=devices` se crea un dispositivo con nombre, tipo, marca, modelo, año, ubicación y atributos; queda listado como `active`.
- [ ] `CA-DEV-002`: El campo Tipo muestra las 16 categorías iniciales; la última opción es «Agregar» con ícono `plus`.
- [ ] `CA-DEV-003`: Al elegir «Agregar» se crea un tipo propio; queda seleccionado en el formulario y disponible para otra vivienda.
- [ ] `CA-DEV-004`: Crear sin nombre o con año `1800`/texto devuelve error de validación y no persiste; con 31 atributos devuelve `too_many_attributes`.
- [ ] `CA-DEV-005`: Editar cambia nombre, tipo, marca, modelo, año, atributos y capabilities; el detalle refleja los cambios.
- [ ] `CA-DEV-006`: Archivar un dispositivo lo pasa a `archived` sin borrar atributos; restaurarlo lo devuelve a `active`.
- [ ] `CA-DEV-007`: Eliminar un dispositivo pide confirmación por palabra y borra sus atributos/capabilities en cascada.
- [ ] `CA-DEV-008`: La búsqueda por nombre/marca/modelo/serie y los filtros por tipo/estado devuelven el subconjunto correcto y los contadores.
- [ ] `CA-DEV-009`: El detalle es de solo lectura y muestra identificación, atributos, capabilities y binding; Editar/Archivar/Eliminar están en el `CardMenu`.
- [ ] `CA-DEV-010`: Dos dispositivos activos de la misma vivienda con el mismo nombre, serial o MAC devuelven `409` (`device_name_taken`, `serial_taken`, `mac_taken`).
- [ ] `CA-DEV-011`: Un `home_id` inexistente devuelve `404`; un `device_id` de otra vivienda también devuelve `404` (sin filtrar datos).
- [ ] `CA-DEV-012`: La etapa 1 no hace ninguna llamada de red ni guarda secretos; `protocol`/`host`/`mac` son solo metadata.
- [ ] `CA-DEV-013`: La migración `0063` up/down es reversible y su test crea/elimina las cuatro tablas y los seeds.
- [ ] `CA-DEV-014`: El código no implementa `Execute`, drivers, manifiestos, panel de módulos, descubrimiento ni polling (etapa 2 fuera de alcance).

### 7.2 Reglas de UI/UX/Datos/Seguridad

- [ ] `CA-ARCH-MODULES`: `Info()` devuelve `home.device`, versión semver y `Migrations: ["0063_home_devices"]`; el módulo está registrado en `all/register.go`.
- [ ] `CA-ARCH-LAYERS`: `handlers.go` no contiene SQL; `storage.go` es el único con consultas.
- [ ] `CA-UI-SELECT`: tipo, protocolo y filtros usan `Select`; `npm run check:ui` pasa.
- [ ] `CA-UI-SELECT-ADD`: la opción de agregar usa `addOption(t('app.add'))`, ícono `plus` y va al final.
- [ ] `CA-UI-NUMBER-INPUT`: el año usa `NumberInput`; `npm run check:numbers` pasa.
- [ ] `CA-UI-TOKENS` / `CA-UI-DARKMODE`: formulario, listado, detalle y modal usan tokens y se verifican en darkmode.
- [ ] `CA-UI-TOGGLE`: las capabilities se activan con `Toggle`; no hay checkbox nativo.
- [ ] `CA-UI-I18N`: `npm run check:i18n` pasa; claves nuevas en es/en; tipos del sistema con mapa estático.
- [ ] `CA-UI-LIST-PATTERN` / `CA-UX-EMPTY`: cards con `CardMenu`; vacío con `EmptyCard` y acción.
- [ ] `CA-UX-BACK`: `/home/:id/devices/new`, `/home/:id/devices/:deviceId` y `/edit` están en `BACK_ROUTES` con retorno a `?tab=devices`.
- [ ] `CA-UX-MODAL` / `CA-UX-MODAL-CONFIRM`: sin cierre por click fuera, cierre visible, modal de tipo `z-[60]`, `DeleteModal` `z-[80]`, `confirmo` + Enter.
- [ ] `CA-UX-DETAIL-ACTIONS` / `CA-UX-DETAIL-READONLY`: acciones solo en `CardMenu`; detalle sin formularios.
- [ ] `CA-UX-PREREQ`: vivienda y tipo se validan en backend y frontend.
- [ ] `CA-UX-LOADING` / `CA-UX-FEEDBACK` / `CA-UX-FORMS`: spinner, botones deshabilitados, toasts y errores junto al campo; un Guardar y un Cancelar.
- [ ] `CA-DATA-MIGRATIONS`: migración y test `migrate_063_test.go` verdes.
- [ ] `CA-DATA-SQLITE`: FKs, CHECK e índices; sin ORM.
- [ ] `CA-SEC-NO-PII` / `CA-SEC-SYNTHETIC`: sin secretos ni PII; ejemplos `demo-*`, `Casa Demo`, `192.0.2.10`, `02:00:00:00:00:01`.
- [ ] `CA-TEST-LOCAL`: `go build ./...`, `go test ./...`, `check:ui`, `check:i18n`, `check:numbers` pasan.
- [ ] `CA-DELIV-MCP-IMPACT` / `CA-DELIV-MCP-IMPACT-SUBAGENT`: §6 completa y validación de subagente registrada.

### 7.3 Testing

- **Unit tests backend**: validación de nombre/año/MAC/host; normalización de slug; límites de atributos; capabilities duplicadas; unicidad por vivienda; transición active/archived; cascada.
- **Integration tests**: `service` + `storage` contra SQLite en memoria; pertenencia vivienda→dispositivo (404 cruzado); seeds idempotentes; migración up/down.
- **Handler tests**: códigos 400/404/409 y JSON de error; auth requerida.
- **Frontend**: build + `check:ui`, `check:i18n`, `check:numbers`; prueba manual de alta con tipo nuevo, edición, archivado, borrado y darkmode.
- **Performance**: listado de 200 dispositivos < 100 ms en el entorno local.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Migración `0063` + seeds + test de migración | 0.5 día | Ninguna |
| 2 | Módulo `home.device`: model/storage/service/handlers + tests + registro | 1.5 días | Fase 1 |
| 3 | Frontend: types/api + tab + formulario + detalle + modal de tipo | 2 días | Fase 2 |
| 4 | i18n, darkmode, `BACK_ROUTES`, guardrails y prueba manual | 0.5 día | Fase 3 |
| 5 | Handoff a QA (spec futura; esta tarea solo deja la spec draft) | — | Fase 4 |

### 8.2 Milestones

1. **MVP (P0)**: alta/listado/edición/archivado/borrado con catálogo abierto, atributos y detalle.
2. **V1.0 (P1)**: capabilities declaradas, metadata de integración, unicidad de binding y filtros.
3. **Etapa 2 (spec futura)**: módulos de control, panel administrativo, secretos cifrados, salud y descubrimiento.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| El catálogo abierto se llena de duplicados («TV», «Tv», «Televisor») | Media | Medio | Slug único + comparación normalizada; tipos del sistema cubren los casos comunes; renombrar/merge en P2. |
| El usuario espera control en la primera entrega | Media | Alto | La spec y la UI etiquetan la etapa 1 como «inventario»; el detalle muestra «Control: etapa 2» y no promete botones. |
| Guardar secretos por error en `notes`/`host`/`integration_ref` | Baja | Alto | Validación de formato, textos de ayuda, revisión en QA y regla explícita; la etapa 2 tendrá almacén cifrado. |
| Adoptar la taxonomía del iHost y dejar fuera TV/electrodomésticos | Baja | Alto | ADR-003: taxonomía propia; mapeo por manifiesto en etapa 2. |
| SSRF/RCE al implementar plugins | Media (etapa 2) | Alto | Fuera de alcance ahora; en etapa 2: allowlist CIDR, validación post-DNS, sidecar sin privilegios, firma de manifiesto. |
| Migración sobre producción | Baja | Medio | Aditiva, `down` reversible, backup y copia en `/tmp` antes de release (`DATA-PRODCOPY`). |
| El listado se degrada con muchos dispositivos | Baja | Bajo | Índices `(home_id,status,name)` y `(home_id,type_id)`, límite 500 y contadores agregados. |
| Cambios en frontend rompen i18n o darkmode | Media | Bajo | Guardrails `check:i18n`/`check:ui` y verificación manual en darkmode. |
| Vivienda con soft delete deja dispositivos huérfanos | Media | Bajo | Mismo comportamiento que `home.workrequest`; `requireHome` oculta los datos; limpieza explícita en una spec futura si molesta. |
| No hay multi-usuario ni ownership por vivienda | Alta (estado actual) | Medio | La app es de un administrador único; la feature no promete aislamiento por usuario. Si se agrega multi-usuario, se especifica aparte. |
| Acoplar `ewelink-mcp`/`sony-bravia-mcp` sin licencia | Baja | Medio | Se reutilizan conceptos, no código; los repos no tienen LICENSE. |

---

## 10. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Precedentes internos: SPEC-141 (`home.maintenance`), SPEC-164 (`home.workrequest`), SPEC-165 (tools MCP de reportes), SPEC-140 (`Select` + «Agregar», `NumberInput`).
- eWeLink / iHost Open API V2: <https://ewelink.cc/ewelink-cube/introduce-open-api/document> · <https://help.sonoff.tech/docs/iHost-API>
- eWeLink CUBE third-party devices (patrón de módulo): <https://ewelink.cc/ewelink-cube/introduce-open-api/document>
- Referencias del usuario: <https://github.com/paulomcnally/ewelink-mcp> · <https://github.com/paulomcnally/sony-bravia-mcp>
- Sony Bravia REST/IRCC: <https://pro-bravia.sony.net/develop/integrate/rest-api/spec/> · <https://www.home-assistant.io/integrations/braviatv/>
- Home Assistant integration manifest/coordinator: <https://developers.home-assistant.io/docs/creating_integration_manifest/> · <https://developers.home-assistant.io/docs/integration_fetching_data/>
- Matter device data model: <https://developers.home.google.com/matter/primer/device-data-model>
- OWASP SSRF / Secrets: <https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html> · <https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html>
- Issue raíz: AGE-40 (`9a9138d6-545b-4da3-bda5-d2964c2ea507`); issue de esta spec: AGE-42 (`dfcb484e-62a2-4be1-9adb-ad6c72890a52`).
- Relacionado pero **fuera de alcance**: SPEC-164 / AGE-41 (reportes de problemas y mantenimiento). No se mezcla.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-27 | opencode | Creación inicial de la especificación (draft): registro de dispositivos, catálogo abierto, atributos/capabilities y diseño de la etapa 2 de módulos. Issue GitHub #168. |
