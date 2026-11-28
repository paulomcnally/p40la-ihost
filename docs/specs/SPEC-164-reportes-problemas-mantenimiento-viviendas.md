---
title: "Reportes de problemas y mantenimiento en viviendas"
id: "SPEC-164"
status: "pending_release"
author: "opencode"
created: "2026-11-27"
updated: "2026-11-27"
github_issue: 166
---

# Reportes de problemas y mantenimiento en viviendas

**ID**: SPEC-164
**Estado**: pending_release
**Autor**: opencode
**Creado**: 2026-11-27
**Actualizado**: 2026-11-27

---

## 1. Resumen Ejecutivo

Hoy el usuario puede registrar **reparaciones ya realizadas** en cada vivienda (`home.maintenance`, SPEC-141): un histórico de hechos consumados con fecha de realización, insumos y mano de obra. Lo que no existe es el paso previo: **reportar un problema o un trabajo deseado** en una casa para que quede registrado como pendiente, con evidencia de la falla, antes de decidir si se convierte en una reparación.

El usuario necesita un sistema para reportar en las viviendas tanto **fallas** (daño en cielo raso, filtración de agua por lluvia en el techo, reparar una llave de agua) como **trabajos de mejora o extensión** (extender el cielo raso, agregar una pared, agregar canaleta para lluvia). Cada reporte arranca en un estado **«reportado»/pendiente, sin insumos ni mano de obra asignados**, y debe poder adjuntar **fotos y/o videos como evidencia** de la falla. Más adelante, un reporte puede **convertirse en una reparación/mantenimiento** reutilizando el módulo existente.

El resultado esperado es un módulo nuevo `home.workrequest` que:
- registra reportes por vivienda con título, descripción, categoría (falla / mejora / mantenimiento / otro) y fecha de reporte;
- mantiene el estado inicial `reported` (pendiente) sin costos;
- adjunta evidencia en fotos y videos, servida desde disco con metadatos en SQLite;
- se enlaza opcionalmente a una reparación (`home_maintenances`) cuando el usuario la convierte.

**Decisión central de modelado**: no se reutiliza `home_maintenances` para los reportes. Se crea una entidad separada (`home_work_requests`) porque una solicitud es una etapa distinta del ciclo de vida (sin fecha de realización, sin costos, con evidencia de la falla) y `home_maintenances` es un **histórico de reparaciones** con `performed_at NOT NULL` y líneas de costo. Reutilizarlo exigiría hacer nullable la fecha de realización y revertir la decisión de SPEC-141 (ADR-002: «histórico, no planificador»). La justificación completa está en §4.3 (ADR-001).

**iHost**: cero dependencias nuevas. Una migración aditiva de dos tablas, consultas indexadas por `home_id`, archivos de evidencia en el volumen montado (con límites de tamaño y cantidad) y UI con los componentes obligatorios (`Select`, `CardMenu`, i18n es/en, darkmode, detalle de solo lectura).

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`). Las reglas no se copian: se citan por ID.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `net/http` + SQLite; React/Tailwind en build. Sin librerías nuevas en backend ni frontend. | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Módulo nuevo en `internal/modules/home/workrequest/`; frontend en `frontend/src/pages/` y `components/home/`. | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Dentro del módulo: `handlers.go` (parseo HTTP), `service.go` (validación/negocio), `storage.go` (SQL), `model.go` (structs). | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | Módulo `home.workrequest` con los 5 archivos obligatorios + tests, registrado en `internal/modules/all/register.go`; rutas montadas por el propio módulo. | `CA-ARCH-MODULES` |
| `ARCH-GO` | `gofmt`, nombres de código en inglés, errores explícitos. | `CA-ARCH-GO` |
| `ARCH-IHOST` | Migración aditiva, índices por `home_id`, límites de tamaño/cantidad de adjuntos, sin cron ni proceso extra, sin llamadas de red. | `CA-ARCH-IHOST` |
| `UI-SELECT` | Categoría y prioridad (si aplica) usan `Select`; no hay `<select>` nativo. | `CA-UI-SELECT` |
| `UI-SELECT-ADD` | `N/A`: la spec no crea entidades desde un dropdown. | — |
| `UI-NUMBER-INPUT` | `N/A`: la spec no agrega campos numéricos editables. | — |
| `UI-INPUT-TOKENS` | Inputs y textarea usan `bg-card`, `text-text`, `text-text-secondary`, `border-border`. | `CA-UI-TOKENS` |
| `UI-DARKMODE` | Formulario, detalle, galería de evidencia y modales se verifican en darkmode. | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | `N/A`: la spec no agrega booleanos de entidad ni de configuración. | — |
| `UI-LIST-PATTERN` | Listado en cards con menú de 3 puntos; sin registros usa `EmptyCard` con acción. | `CA-UI-LIST-PATTERN` |
| `UI-CREATE-MENU` | `N/A`: el alta vive dentro del tab de negocio, no en un header de listado raíz. | — |
| `UI-EDIT-LABEL` | Las acciones de edición usan `app.edit`. | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | Claves nuevas solo en `frontend/public/i18n/{es,en}.json`, con paridad y sin apuntar a objetos padre. | `CA-UI-I18N` |
| `UI-STYLE` | Cards `rounded-ios shadow-ios`, sidebar existente, menú de acciones, confirmación de borrado. | `CA-UI-STYLE` |
| `UX-BACK` | Rutas de detalle/formulario con lista padre registradas en `BACK_ROUTES` con retorno a `/home/:id?tab=reports`. | `CA-UX-BACK` |
| `UX-MODAL` | `DeleteModal` a `z-[80]`, sin cierre por click fuera, con cierre visible. | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | El borrado de reporte usa `DeleteModal` (palabra `confirmo`). | `CA-UX-MODAL-CONFIRM` |
| `UX-DETAIL-ACTIONS` | En el detalle, Editar/Eliminar y Convertir viven en `CardMenu`, sin íconos sueltos. | `CA-UX-DETAIL-ACTIONS` |
| `UX-DETAIL-READONLY` | El detalle muestra los datos sin formularios; editar navega a `/edit`. | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | Vivienda (y reparación al convertir) se validan en backend y frontend; vivienda inexistente devuelve 404. | `CA-UX-PREREQ` |
| `UX-EMPTY` | Listado vacío muestra `EmptyCard` explicando qué es y con acción de alta. | `CA-UX-EMPTY` |
| `UX-LOADING` | Spinner/skeleton en listado y detalle; botones deshabilitados mientras guardan. | `CA-UX-LOADING` |
| `UX-FEEDBACK` | Toast de éxito/error al crear, editar, borrar, adjuntar y convertir; errores junto al campo. | `CA-UX-FEEDBACK` |
| `UX-FORMS` | Un Guardar y un Cancelar; validación bloquea el guardado; obligatorios marcados. | `CA-UX-FORMS` |
| `DATA-SQLITE` | Tablas `home_work_requests` y `home_work_request_attachments` con FKs, CHECK e índices; SQL crudo sin ORM. | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | `0062_home_work_requests.up.sql` y `.down.sql` reversibles; probadas con `go test`. | `CA-DATA-MIGRATIONS` |
| `DATA-PRODCOPY` | `N/A`: la migración es aditiva y no transforma datos; no hay matching contra datos reales. | — |
| `DATA-NO-DROP` / `DATA-PROTECTED-SCHEMA` | `N/A`: no se toca `data/app.db`, ni `debts`/`debt_bills`, ni el esquema de `home_maintenances`. | — |
| `SEC-NO-PII` | Spec, tests y fixtures usan solo datos sintéticos; no se commitean fotos, videos ni capturas reales. | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Casa Demo`, `Proveedor Demo`, descripciones y nombres de archivo ficticios. | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` | `git diff --cached` revisado: sin fotos, videos, DBs ni exports reales. | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...` y los guardrails de frontend antes de commit. | `CA-TEST-LOCAL` |
| `TEST-SERVER` | Se valida con `./scripts/check-server.sh` antes de avisar al usuario. | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | Se pregunta 🤖 antes de probar y antes de cerrar la spec. | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-164`, rama `feature/SPEC-164`. | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita del usuario. | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa con validación de subagente. | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; veredicto citado en §6. | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-HOOKS` | Commits `SPEC-164: …` con hooks instalados, sin `--no-verify`. | `CA-DELIV-HOOKS` |
| `DELIV-DOCKER` / `DELIV-DEPLOY` | `N/A`: la spec no cambia Dockerfile ni despliega; el deploy lo aprueba el usuario. | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Reporte por vivienda.** Crear un reporte de problema/trabajo asociado a una vivienda, con `title` (resumen corto, obligatorio), `description` (detalle, opcional), `category` (`repair` falla / `improvement` mejora o extensión / `maintenance` preventivo / `other`), `priority` (`low` / `normal` / `high`, default `normal`) y `reported_at` (fecha del reporte, no futura). El estado inicial es **`reported`** (reportado/pendiente), sin insumos ni mano de obra.
2. **REQ-002 — Estado inicial pendiente.** Todo reporte nuevo nace con `status = 'reported'`. El reporte **no** tiene líneas de insumo ni mano de obra: esas pertenecen a `home_maintenances` cuando el reporte se convierta (REQ-009). Un reporte en `reported` no genera ningún costo.
3. **REQ-003 — Evidencia en fotos y videos.** Adjuntar al reporte archivos de evidencia de tipo imagen (`jpeg`, `png`, `webp`, `heic`) y video corto (`mp4`, `webm`, `mov`), con límites de tamaño y cantidad (ver §3.4). Los archivos se guardan en disco (`DATA_DIR`) y la base solo conserva metadatos.
4. **REQ-004 — Listado por vivienda.** Listar los reportes de una vivienda ordenados por `reported_at DESC, id DESC`, con badge de estado y de categoría, cantidad de adjuntos y vínculo a la reparación cuando exista. Filtro por `status` y `category` (P1, REQ-011).
5. **REQ-005 — Detalle de solo lectura.** Página de detalle que muestra los datos del reporte, la evidencia (galería de imágenes con thumbnail y reproducción de video) y las acciones en el menú de 3 puntos.
6. **REQ-006 — Edición.** Editar título, descripción, categoría, prioridad y fecha de reporte de un reporte existente (no se edita el estado directamente: el estado cambia por las acciones «convertir» y «descartar»).
7. **REQ-007 — Eliminación.** Eliminar un reporte y, en cascada, sus adjuntos (metadatos y archivos en disco). Requiere confirmación (`DeleteModal`).
8. **REQ-008 — CRUD de evidencia.** Subir, servir y eliminar adjuntos de un reporte (`POST`/`GET`/`DELETE`), validando pertenencia vivienda → reporte → adjunto.
9. **REQ-009 — Conversión a reparación.** Convertir un reporte en una reparación del módulo `home.maintenance`: crea un `home_maintenance` vacío (sin líneas), copia el título/descripción y mapea la categoría a `kind`/`area`, y enlaza el reporte (`maintenance_id`, `status = 'converted'`, `converted_at`). Es **P1** (REQ-010); el modelo lo soporta desde la primera migración mediante la FK nullable `maintenance_id`.
10. **REQ-010 — Persistencia de la evidencia tras convertir.** Al convertir, la evidencia queda asociada al reporte (no se mueve a la reparación). El vínculo `reporte → reparación` permite navegar del reporte al detalle de la reparación.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011 — Filtros.** Filtro por `status` (`reported` / `converted` / `dismissed`) y por `category` en el listado, con contador por estado.
2. **REQ-012 — Descartar un reporte.** Acción «Descartar» que pasa un reporte `reported` a `status = 'dismissed'` sin crear reparación ni borrar la evidencia. Un reporte descartado se puede reactivar a `reported`.
3. **REQ-013 — Conversión con detalle.** La acción «Convertir a reparación» permite elegir `kind` (correctivo/preventivo) y `area` (defaults: `corrective` y `other`), y reutiliza `POST /api/homes/{id}/maintenances` vía contrato para crear la reparación vacía. Tras convertir, el detalle del reporte muestra un enlace a la reparación.
4. **REQ-014 — Resumen en el tab de la vivienda.** El tab «Reportes» muestra el conteo de reportes pendientes (`reported`) para que el usuario vea el volumen pendiente de un vistazo.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015 — Evidencia en la reparación.** Evaluar en una spec futura copiar o mover la evidencia del reporte al adjuntar comprobantes a la reparación (SPEC-141 REQ-021 quedó como P2 y no se implementó).
2. **REQ-016 — Recordatorio.** Programar un recordatorio o vencimiento para un reporte pendiente.
3. **REQ-017 — Historial de cambios.** Auditar los cambios de estado de un reporte (reported → converted/dismissed) con fecha y autor.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: listar 500 reportes de una vivienda en < 150 ms; el listado no carga adjuntos uno por uno (una sola consulta + conteo agregado de adjuntos).
- **Memoria**: el listado se pagina o limita; no se mantienen los adjuntos de todos los reportes en memoria.
- **Almacenamiento**: dos tablas nuevas + índices; los archivos de evidencia viven en `DATA_DIR` con límites (ver abajo).
- **Adjuntos**: máximo **10 adjuntos por reporte**; máximo **25 MB por archivo**; imágenes `jpeg/png/webp/heic` y videos `mp4/webm/mov`. Sin transcodificación ni generación de thumbnails en el servidor: las imágenes se muestran con `<img>` (thumbnail por CSS) y los videos con `<video controls preload="metadata">`.
- **Seguridad**: autenticación obligatoria; validación de pertenencia vivienda → reporte → adjunto; `Content-Type` correcto al servir y `X-Content-Type-Options: nosniff`; no se aceptan archivos que no sean imagen/video.
- **iHost**: sin dependencias nuevas, sin cron, sin llamadas de red, sin FFmpeg.
- **i18n**: paridad es/en y claves planas.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- El módulo `internal/modules/home/maintenance/` (SPEC-141) modela reparaciones **realizadas** (`home_maintenances`) con `performed_at TEXT NOT NULL`, `kind` (`corrective`/`preventive`), `area` y líneas de insumo/mano de obra (`home_maintenance_items`). **No tiene concepto de estado ni de fecha de reporte.** Migración `0057_home_maintenance`.
- SPEC-141 ADR-002 decidió explícitamente «histórico, no planificador»: sin tareas pendientes. Esta spec introduce la etapa de *reporte pendiente* que aquella decisión dejó fuera, como entidad separada.
- `homes` usa borrado lógico (`deleted_at`); `currencies` e `institutions` también se referencian por FK con soft-delete. El módulo nuevo sigue ese patrón (`HomeExists`, `CurrencyExists`, `InstitutionExists` en `storage.go` de `home.maintenance`).
- El patrón de adjuntos existente es `auto_attachment` (SPEC-124), **solo PDF** (valida `%PDF-`) y atado a autos (`internal/models/auto_attachment.go`, `internal/services/auto_attachment.go`, `internal/api/auto_attachment_handlers.go`). Para **imágenes** ya hay un precedente limpio en `realestate.rental` (`internal/modules/realestate/rental/billing.go`): `readUpload` + `storeRentalFileInDir` + `detectFileKind` con magic bytes para `pdf`/`png`/`jpeg`, y la pensión acepta además `webp`. Para **video no existe nada** en el repo: hay que agregar detección de `mp4`/`webm`/`quicktime` y su `Content-Type`. La spec define un adjunto multimedia propio siguiendo el patrón de `realestate.rental`, ampliado a video.
- El frontend de vivienda (`HomeShowPage.tsx`) ya usa tabs por query string (`?tab=summary|rental|finances|maintenance`) y el patrón de páginas `HomeMaintenanceFormPage` / `HomeMaintenanceShowPage` (SPEC-141).
- Investigación externa (subagente `general`): el patrón estándar de la industria separa **solicitud** (sin costo, con evidencia, estado pendiente) de **orden/reparación** (donde viven costos y trabajo). MaintainX separa *Work Request Management* de *Work Order Management*; Property Meld describe «from request to resolution» con el costo asignado en la fase de reparación. La evidencia visual es un adjunto de primera clase (fotos anotadas, thumbnails); para video en web ligera se recomienda `<video preload="metadata">` sin FFmpeg y con soporte de HTTP Range.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Reutilizar `home_maintenances` agregando una columna `status` | Una sola tabla; literalmente «mismo modelo con estado» | Exige hacer nullable `performed_at` (hoy NOT NULL) y reintroducir «tareas pendientes» que SPEC-141 ADR-002 descartó; contamina el histórico y el reporte financiero (`realestate.portfolio` suma por `performed_at`); rompe el contrato de las 9 tools MCP `home_maintenance_*` | ❌ Rechazada |
| Entidad separada `home_work_requests` enlazada a `home_maintenances` | Sin tocar el histórico ni el reporte financiero; evidencia y estado propios; mapea al patrón estándar | Una tabla y un módulo nuevos | ✅ Seleccionada |
| Modelo polimórfico `work_requests(owner_type, owner_id)` | Una tabla para casas y autos | Sin FKs reales, validaciones manuales | ❌ Rechazada |
| Adjuntos de evidencia reutilizando `auto_attachment` | Menos código | Es solo PDF y está atado a autos | ❌ Rechazada |
| Adjuntos de evidencia con tabla propia multimedia | Acepta imágenes y video con límites | Una tabla y lógica de subida/servido nuevas | ✅ Seleccionada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Reporte y reparación son entidades separadas.**
- **Contexto**: el usuario pidió «un estado tipo reportado sin insumo y sin mano de obra, como pendiente» y que luego «se convierta en reparación/mantenimiento». `home_maintenances` es un histórico con `performed_at NOT NULL` y líneas de costo.
- **Decisión**: nueva entidad `home_work_requests` con su propio ciclo (`reported` → `converted`/`dismissed`) y FK nullable `maintenance_id → home_maintenances(id)`. No se modifica el esquema ni la semántica de `home_maintenances`.
- **Consecuencias**: el reporte puede existir sin fecha de realización ni costo; el histórico y el reporte financiero no cambian. La conversión es un paso explícito que crea una `home_maintenance` vacía y enlaza.

**ADR-002 — Categoría del reporte, no `kind` de la reparación.**
- **Contexto**: los ejemplos incluyen fallas (reparar) y mejoras/extensiones (agregar pared, extender cielo raso, agregar canaleta), que no encajan en `corrective`/`preventive`.
- **Decisión**: el reporte tiene `category ∈ {repair, improvement, maintenance, other}`. Al convertir, se mapea a `kind` (default `corrective`) y `area` (default `other`), sin ampliar el enum de `home_maintenances.kind` en esta spec.
- **Consecuencias**: la clasificación rica vive en el reporte; la reparación mantiene su modelo intacto.

**ADR-003 — Evidencia multimedia propia, sin transcodificación.**
- **Contexto**: el iHost no tiene FFmpeg ni debe procesar video. `auto_attachment` es PDF-only.
- **Decisión**: tabla `home_work_request_attachments` con `media_type ∈ {image, video}`, archivos en `DATA_DIR/workrequests/{work_request_id}/`, validación por firma de archivo contra una allowlist. Las imágenes se muestran con `<img>` y los videos con `<video preload="metadata">` (el navegador muestra el primer frame). El servidor emite `Content-Type` correcto y soporta Range.
- **Consecuencias**: cero dependencias; evidencia ligera y suficiente. La generación de thumbnails de video queda fuera de alcance.

**ADR-004 — La evidencia queda en el reporte tras convertir.**
- **Contexto**: la evidencia documenta la falla que motivó el reporte, no la reparación.
- **Decisión**: convertir no mueve ni copia adjuntos; el reporte conserva su evidencia y queda enlazado a la reparación.
- **Consecuencias**: no hay duplicación de archivos; el historial visual se consulta desde el reporte. Copiar evidencia a la reparación se evalúa como P2 (REQ-015).

**ADR-005 — Conversión reutilizando `home.maintenance`, sin tocar su código.**
- **Contexto**: `ARCH-MODULES` prohíbe importar `storage`/`handlers` de otro módulo; la comunicación es por contratos. El módulo `home.maintenance` ya expone `Module.Service() *Service` público y su `Service.Create(ctx, homeID, Input)` es reutilizable. Existe un patrón cross-módulo establecido: `realestate.portfolio` consume `realestate.rental` inyectándolo por constructor en `internal/modules/all/register.go`.
- **Decisión**: `home.workrequest` recibe el `*modules.Registry` (o el módulo `home.maintenance` por constructor) y, al convertir, llama a `home.maintenance.Service().Create(...)` para crear la reparación vacía y enlazarla. **No se edita ningún archivo de `internal/modules/home/maintenance/`** ni se agregan métodos nuevos.
- **Consecuencias**: conversión atómica a nivel de servicio, sin acoplar módulos y **sin impacto en las tools MCP** `home_maintenance_*` (solo se modifica `internal/modules/all/register.go`, fuera de la superficie MCP — ver §6).

**ADR-006 — Fecha de reporte, no fecha de realización.**
- **Contexto**: `home_maintenances.performed_at` es la fecha en que se hizo el trabajo; un reporte se registra antes.
- **Decisión**: `home_work_requests.reported_at TEXT NOT NULL` (formato `YYYY-MM-DD`, no futura). `converted_at` se guarda al convertir.
- **Consecuencias**: el listado y los reportes se ordenan por `reported_at`; no hay ambigüedad con la fecha de la reparación.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
[React: HomeShowPage tab "reports"] --> HomeWorkRequestTab / FormPage / ShowPage
              |                                 |
              +------ fetch /api/homes/{id}/workrequests -----+
                                                              |
                                      [módulo home.workrequest]
                                      handlers -> service -> storage
                                              |            |
                              (convertir)     |            v
                              home.maintenance.Service().Create  [SQLite]
                                        |            home_work_requests
                                        v            home_work_request_attachments
                              [home.maintenance] -> home_maintenances (FK)
                                                              |
                                      [DATA_DIR/workrequests/{id}/...]  (archivos)
```

### 5.2 Componentes

#### 5.2.1 Backend — módulo `home.workrequest`
- **Responsabilidad**: CRUD de reportes y de adjuntos, validación, conversión a reparación.
- **Interfaz**: ver §5.4.
- **Dependencias**: `*sql.DB`; `*modules.Registry` (o el módulo `home.maintenance` por constructor, para convertir); helpers de fecha.
- **Ubicación**: `internal/modules/home/workrequest/`
  - `module.go`: `Info()` con ID `home.workrequest`, versión `1.0.0`, `Kind: core`, migración `0062_home_work_requests`; `RegisterRoutes`.
  - `model.go`: structs y constantes (`StatusReported`, `StatusConverted`, `StatusDismissed`; `CategoryRepair`, `CategoryImprovement`, `CategoryMaintenance`, `CategoryOther`; `PriorityLow/Normal/High`; `MediaImage`, `MediaVideo`).
  - `storage.go`: SQL del módulo; listado con conteo de adjuntos, CRUD, `HomeExists`, `MaintenanceExists` (para el link).
  - `service.go`: validaciones, límites de adjuntos, armado de totales/conteos y conversión.
  - `handlers.go`: parseo HTTP y mapeo de errores a 400/404/500.
  - `service_test.go` / `storage_test.go`: tests con DB en `/tmp`.

#### 5.2.2 Frontend
- `frontend/src/types/homeWorkRequest.ts`: tipos `HomeWorkRequest`, `HomeWorkRequestAttachment`, inputs y lista.
- `frontend/src/api/homeWorkRequest.ts`: cliente del módulo (`list`, `get`, `create`, `update`, `delete`, `convert`, `uploadAttachment`, `deleteAttachment`).
- `frontend/src/components/home/HomeWorkRequestTab.tsx`: listado dentro del detalle de la vivienda.
- `frontend/src/components/home/HomeWorkRequestAttachmentField.tsx`: subida y vista previa de evidencia (thumbnails de imagen, `<video>` para video).
- `frontend/src/pages/HomeWorkRequestFormPage.tsx`: formulario nuevo/editar.
- `frontend/src/pages/HomeWorkRequestShowPage.tsx`: detalle de solo lectura con galería de evidencia.
- **Integración**: `HomeShowPage` gana el tab `reports`; `App.tsx` monta las rutas; `DashboardLayout` registra `BACK_ROUTES`.

### 5.3 Modelo de datos

```sql
-- migrations/0062_home_work_requests.up.sql
CREATE TABLE home_work_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'repair'
        CHECK (category IN ('repair', 'improvement', 'maintenance', 'other')),
    priority TEXT NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('low', 'normal', 'high')),
    status TEXT NOT NULL DEFAULT 'reported'
        CHECK (status IN ('reported', 'converted', 'dismissed')),
    reported_at TEXT NOT NULL,
    maintenance_id INTEGER,
    converted_at TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE CASCADE,
    FOREIGN KEY (maintenance_id) REFERENCES home_maintenances(id) ON DELETE SET NULL
);

CREATE INDEX idx_home_work_requests_home_date
    ON home_work_requests(home_id, reported_at DESC, id DESC);
CREATE INDEX idx_home_work_requests_status
    ON home_work_requests(home_id, status);

CREATE TABLE home_work_request_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_request_id INTEGER NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('image', 'video')),
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT '',
    size_bytes INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (work_request_id) REFERENCES home_work_requests(id) ON DELETE CASCADE
);

CREATE INDEX idx_home_work_request_attachments_owner
    ON home_work_request_attachments(work_request_id, id);
```

- `reported_at` y `converted_at` en `TEXT` ISO `YYYY-MM-DD` (convención del proyecto).
- El `down` de `0062` elimina las dos tablas y sus índices (reversible).
- Al convertir, `maintenance_id` usa `ON DELETE SET NULL`: borrar una reparación no borra el reporte ni su evidencia.
- Los archivos de evidencia viven en `DATA_DIR/workrequests/{work_request_id}/{attachment_id}-{file_name}`; la base solo guarda metadatos y `file_name` se sanitiza (basename, sin caracteres de control). El tipo real se valida por firma (magic bytes) contra la allowlist de §3.4; el `media_type` declarado debe coincidir.

### 5.4 APIs / Contratos

Base: `/api/homes/{homeID}/workrequests`, autenticadas.

#### `GET /api/homes/{homeID}/workrequests?status=reported&category=repair`
**Response 200**:
```json
{
  "items": [{
    "id": 1,
    "home_id": 1,
    "title": "Filtración de agua en el techo",
    "description": "Gotea cuando llueve, sobre el cuarto principal.",
    "category": "repair",
    "priority": "high",
    "status": "reported",
    "reported_at": "2026-11-26",
    "maintenance_id": null,
    "converted_at": null,
    "attachment_count": 2,
    "created_at": "2026-11-26T10:00:00Z",
    "updated_at": "2026-11-26T10:00:00Z"
  }],
  "pending_count": 3,
  "truncated": false
}
```

#### `POST /api/homes/{homeID}/workrequests`
```json
{ "title": "Agregar canaleta para lluvia", "description": "Para evitar que el agua baje por la pared.",
  "category": "improvement", "priority": "normal", "reported_at": "2026-11-26" }
```
`201` con el reporte creado. Errores: `400` validación (`title_required`, `invalid_category`, `invalid_priority`, `invalid_date`, `future_date`), `404` vivienda inexistente.

#### `GET|PUT|DELETE /api/homes/{homeID}/workrequests/{work_request_id}`
`PUT` acepta `title`, `description`, `category`, `priority`, `reported_at` (no edita `status`). `DELETE` responde `204` y elimina adjuntos en cascada (metadatos + archivos).

#### `POST /api/homes/{homeID}/workrequests/{work_request_id}/status` (REQ-012)
```json
{ "status": "dismissed" }
```
Pasa un reporte `reported` a `dismissed` o reactiva un `dismissed` a `reported` (no admite `converted`). Responde `200` con el reporte actualizado; `400 invalid_status` si la transición no es válida y `404` si la vivienda o el reporte no existen.

#### `POST /api/homes/{homeID}/workrequests/{work_request_id}/convert` (P1)
```json
{ "kind": "corrective", "area": "other" }
```
Crea un `home_maintenance` vacío vía contrato y responde `200` con `{ "maintenance_id": 42, "work_request": {...} }`. Si el reporte ya está `converted`, responde `409 already_converted`. `404` si la vivienda o el reporte no existen.

#### `POST /api/homes/{homeID}/workrequests/{work_request_id}/attachments`
Multipart `file` + campo `media_type` (`image`|`video`). `201` con los metadatos del adjunto. Errores: `request_too_large` (413), `attachment_limit` (409), `invalid_file` (400, tipo no permitido o archivo vacío).

#### `GET /api/homes/{homeID}/workrequests/{work_request_id}/attachments/{attachment_id}`
Sirve el archivo con `Content-Type` del `mime_type` guardado, `Content-Disposition: inline` (o `attachment` con `?download=1`) y `X-Content-Type-Options: nosniff`. Soporta Range para video.

#### `DELETE /api/homes/{homeID}/workrequests/{work_request_id}/attachments/{attachment_id}`
`204`; elimina metadatos y archivo en disco.

### 5.5 Dependencias

- **Internas**: `homes` (legacy), `home.maintenance` (solo vía `Module.Service().Create` para convertir, sin editar sus archivos), `internal/modules` (registry), middleware de auth, componentes `Select`/`CardMenu`/`DeleteModal`/`Toast`.
- **Externas**: ninguna nueva.

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). Análisis con `./scripts/check-mcp-impact.sh --files ...` y validación de un subagente (`DELIV-MCP-IMPACT-SUBAGENT`).

### 6.1 Veredicto

**Ninguna tool MCP afectada funcionalmente.** La feature:
- crea un módulo nuevo `internal/modules/home/workrequest/` (no está en ninguna dependencia de tool);
- agrega la migración `0062_home_work_requests` que **solo crea tablas nuevas aisladas** (`home_work_requests`, `home_work_request_attachments`) y no altera `home_maintenances`, `home_maintenance_items`, `homes`, `currencies` ni `institutions`;
- la conversión reutiliza `home.maintenance.Service().Create` **sin editar** `internal/modules/home/maintenance/` (solo se modifica `internal/modules/all/register.go`, fuera de la superficie MCP).

### 6.2 Tools nuevas

Ninguna. La spec no agrega tools MCP (es una feature de la app web; las tools MCP de reportes, si se quieren, serán una spec de seguimiento).

### 6.3 Tools existentes afectadas

Ninguna con cambio funcional ni de contrato.

### 6.4 Nota de guardrail: `shared.affects_all` por `migrations/`

El manifiesto `docs/mcp/tools/impact.json` declara `migrations/` en `shared.affects_all`. Por eso, al agregar `0062_*.sql`, `./scripts/check-mcp-impact.sh --files ...` marcará técnicamente **las 53 tools** por recompilación de la BD compartida. El impacto real es **nulo**: la migración es aditiva y aislada. Se listan para que la declaración sea explícita:

`alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bank_accounts_create`, `bank_accounts_search`, `bank_accounts_update`, `banks_create`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `budget_accounts_list`, `budget_categories_list`, `budget_month_view`, `budget_transactions_create`, `budget_transactions_list`, `credit_card_statements_get`, `credit_card_statements_list`, `credit_cards_create`, `credit_cards_search`, `credit_cards_update`, `debt_bills_list`, `entities_search`, `home_maintenance_create`, `home_maintenance_get`, `home_maintenance_item_delete`, `home_maintenance_item_move`, `home_maintenance_item_update`, `home_maintenance_labor_register`, `home_maintenance_list`, `home_maintenance_supplies_register`, `home_maintenance_update`, `homes_create`, `homes_update`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

### 6.5 Condición de diseño para mantener «ninguna afectada»

No tocar `internal/mcp/**`, `internal/api/routes.go`, `internal/api/handlers.go`, `internal/models/`, `internal/services/service.go`, `internal/storage/service.go` ni ningún archivo de `internal/modules/home/maintenance/`. Las rutas del módulo nuevo se registran vía `RegisterRoutes` propio del módulo (como `home.maintenance`), sin editar `routes.go`.

**Validación subagente:** explore — sesión `ses_de0e93074ffePx6QHkTSbclWLY` — veredicto: **ninguna tool MCP afectada funcionalmente**. El módulo nuevo (`internal/modules/home/workrequest/`) no coincide con ninguna dependencia de tool; la conversión reutiliza `home.maintenance.Service().Create` (público) sin editar `internal/modules/home/maintenance/`, por lo que tampoco dispara las 9 tools `home_maintenance_*`. El único disparo del guardrail es `shared.affects_all → migrations/` al agregar `0062_*.sql` (53 tools marcadas por recompilación, impacto real nulo si la migración solo crea tablas nuevas). Se debe evitar modificar `home.maintenance`, `routes.go`, `handlers.go`, `internal/models/`, `services/service.go` y `storage/service.go`.

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

> IDs canónicos por dominio (`CA-<PREFIJO>-<NOMBRE>`) para que el cumplimiento sea auditable con grep.

### 7.1 Funcionales

- [ ] `CA-WR-001`: Desde el tab «Reportes» de una vivienda se crea un reporte con título, descripción, categoría y fecha; queda listado con estado `reported`.
- [ ] `CA-WR-002`: Un reporte nuevo nace `reported` y no tiene insumos ni mano de obra; su detalle no muestra costos.
- [ ] `CA-WR-003`: Se adjunta una foto y un video a un reporte; ambos aparecen en la galería del detalle (imagen con thumbnail, video reproducible).
- [ ] `CA-WR-004`: El listado muestra badge de estado (`reported`/`converted`/`dismissed`) y categoría, y cuenta los adjuntos.
- [ ] `CA-WR-005`: Editar cambia título/descripción/categoría/prioridad/fecha sin alterar el estado; el detalle permanece de solo lectura.
- [ ] `CA-WR-006`: Eliminar un reporte borra sus adjuntos (metadatos y archivos) y requiere confirmación por palabra.
- [ ] `CA-WR-007`: Subir un adjunto de otra vivienda/reporte devuelve `404`; un archivo no imagen/video devuelve `400`.
- [ ] `CA-WR-008`: Superar 10 adjuntos devuelve `attachment_limit`; superar 25 MB devuelve `request_too_large`.
- [ ] `CA-WR-009`: Convertir un reporte crea un `home_maintenance` vacío y enlaza `maintenance_id`; el reporte pasa a `converted`. Convertir dos veces devuelve `409 already_converted`.
- [ ] `CA-WR-010`: Descartar un reporte `reported` lo pasa a `dismissed`; reactivar lo devuelve a `reported`.
- [ ] `CA-WR-011`: Filtro por `status` y `category` devuelve el subconjunto correcto; `pending_count` refleja los `reported`.
- [ ] `CA-ARCH-MODULES`: `Info()` devuelve `home.workrequest`, versión semver y la migración declarada; el módulo está registrado en `all/register.go`.
- [ ] `CA-ARCH-LAYERS`: `handlers.go` no contiene SQL; `storage.go` es el único con consultas.
- [ ] `CA-UI-SELECT`: categoría y prioridad usan `Select`; `npm run check:ui` pasa.
- [ ] `CA-UI-TOKENS`: inputs y textarea usan tokens del tema; verificado en darkmode.
- [ ] `CA-UI-DARKMODE`: formulario, detalle y galería legibles en darkmode.
- [ ] `CA-UI-I18N`: `npm run check:i18n` pasa y no hay claves faltantes en es/en.
- [ ] `CA-UI-EDIT-LABEL`: las acciones de edición dicen «Editar» (`app.edit`).
- [ ] `CA-UI-LIST-PATTERN`: listado en cards con `CardMenu`; vacío con `EmptyCard`.
- [ ] `CA-UX-MODAL` / `CA-UX-MODAL-CONFIRM`: `DeleteModal` no cierra por click fuera y acepta Enter con `confirmo`.
- [ ] `CA-UX-DETAIL-ACTIONS`: editar/eliminar/convertir/descartar están en `CardMenu`, sin íconos sueltos.
- [ ] `CA-UX-DETAIL-READONLY`: el detalle no tiene formularios; editar navega a `/edit`.
- [ ] `CA-UX-BACK`: el retorno va en el header vía `BACK_ROUTES` a `/home/:id?tab=reports`.
- [ ] `CA-UX-PREREQ`: vivienda (y reparación al convertir) se validan en backend y frontend.
- [ ] `CA-UX-EMPTY`: listado vacío muestra `EmptyCard` con acción.
- [ ] `CA-UX-LOADING`: spinner en carga y botones deshabilitados al guardar.
- [ ] `CA-UX-FEEDBACK`: toast de éxito y error en crear, editar, borrar, adjuntar y convertir.
- [ ] `CA-DATA-MIGRATIONS`: migración `0062` up/down reversible y test dedicado.
- [ ] `CA-SEC-SYNTHETIC`: tests y ejemplos usan `Casa Demo` y descripciones/nombres de archivo ficticios.
- [ ] `CA-TEST-LOCAL`: `go build ./...`, `go test ./...`, `check:ui`, `check:i18n` pasan.

### 7.2 No funcionales

- [ ] `CA-NF-001`: listar 500 reportes de una vivienda responde en < 150 ms en el iHost con índices.
- [ ] `CA-NF-002`: el listado no dispara una consulta por reporte para los adjuntos (sin N+1).
- [ ] `CA-NF-003`: no se agregan dependencias a `go.mod` ni a `package.json`.
- [ ] `CA-NF-004`: servir video emite `Content-Type` correcto y soporta Range (seek).
- [ ] `CA-NF-005`: sin FFmpeg ni transcodificación en el servidor.

### 7.3 Testing

- **Unit tests (Go)**: validación de título/categoría/prioridad/fecha; límites de adjuntos; rechazo de tipos no permitidos; conversión y doble-conversión; cascada al borrar.
- **Integration tests**: migración `0062` up/down; CRUD vía handler con DB en `/tmp`; subida/servido/borrado de adjuntos.
- **Frontend**: build + `check:ui` + `check:i18n`; prueba manual de darkmode y móvil; reproducción de video.
- **Datos**: DB sintética; no requiere copia de producción (`DATA-PRODCOPY` N/A).

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Migración `0062` + test up/down | 0.5 día | Ninguna |
| 2 | Módulo Go `home.workrequest` (model/storage/service/handlers) + registro | 1.5 días | Fase 1 |
| 3 | Adjuntos de evidencia (subida, servido con Range, borrado) + tests | 1 día | Fase 2 |
| 4 | Conversión a reparación vía contrato + tests | 0.5 día | Fase 2 |
| 5 | Tipos + API client frontend | 0.5 día | Fase 2 |
| 6 | Tab, formulario, detalle y galería + i18n es/en | 1.5 días | Fase 5 |
| 7 | Guardrails, darkmode, móvil y pruebas del usuario | 0.5 día | Fase 6 |

### 8.2 Milestones

1. **MVP**: P0 completo (reporte + evidencia + listado + detalle + i18n + guardrails).
2. **V1.0**: conversión a reparación, descartar/reactivar y filtros por estado/categoría.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Confundir «reporte» con «reparación» en la UI | Media | Medio | Tab separado «Reportes» y badges de estado claros; texto de ayuda |
| Archivos de evidencia grandes llenan el disco del iHost | Media | Medio | Límite 25 MB/archivo y 10 adjuntos/reporte; borrado en cascada |
| Servir video sin Range rompe el seek | Media | Bajo | `http.ServeFile` (ya soporta Range); test de `Content-Type` |
| La conversión toca `home.maintenance` y dispara impacto MCP | Alta | Bajo | Impacto de recompilación declarado en §6; sin cambio funcional en tools |
| El usuario espera que la evidencia pase a la reparación al convertir | Media | Bajo | Documentado en ADR-004; la evidencia queda en el reporte enlazado |
| Subir archivos que no son imagen/video (seguridad) | Baja | Alto | Validación por firma de archivo + `nosniff`; solo allowlist |

---

## 10. Notas y Referencias

- Reglas aplicadas: `docs/rules/00-index.md`, `10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Precedentes: SPEC-141 (módulo `home.maintenance`), SPEC-143 (tabs de vivienda y `usage`), SPEC-124 (adjuntos de autos, PDF), SPEC-155/157/159 (tools MCP de reparaciones).
- Issue raíz del pipeline: AGE-40. Este issue es AGE-41 (modo `solo spec draft`).

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-27 | opencode | Creación inicial de la especificación (draft) |
| 2026-11-27 | opencode | Investigación con subagentes (`explore` mapeo de código e impacto MCP; `general` ciclo reporte→reparación y evidencia multimedia) integrada en §4/§6; issue #166 creado |
| 2026-11-27 | opencode | Implementación P0/P1 en `feature/SPEC-164`: migración `0062`, módulo `home.workrequest` con evidencia y conversión, API de descarte/reactivación, UI del tab/formulario/detalle y i18n es/en |
| 2026-11-27 | opencode | Verificación local: `go build ./...`, `go test ./...`, `npm run build`, gate MCP y prueba end-to-end con copia de la DB en `/tmp`; preparada para release |
