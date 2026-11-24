---
title: "Servicios: casa filtrada al crear y alta rápida de institución"
id: "SPEC-144"
status: "released"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 147
---

# Servicios: casa filtrada al crear y alta rápida de institución

**ID**: SPEC-144  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

Cuando el listado de servicios (`/services`) está filtrado por una casa, el flujo de creación (`/services/new`) no conserva ese contexto: el formulario inicializa la casa con la primera de la lista (`homes[0]`). El usuario puede crear el servicio en la casa equivocada sin advertirlo. Esta spec hace que el formulario preseleccione la casa que se está filtrando; si no hay filtro activo, mantiene el comportamiento actual y permite elegir cualquier casa.

En el mismo formulario, el dropdown de institución obliga a abandonar el flujo para crear una institución nueva. Se agrega al final del dropdown la opción canónica «Agregar» (`app.add`), que abre un modal de alta rápida con nombre y categoría opcional; al guardar, la institución queda creada y seleccionada sin perder los datos del servicio.

El cambio es **exclusivamente de frontend**: reutiliza `GET /api/institutions`, `GET /api/institution-categories` y `POST /api/institutions` existentes, no agrega dependencias, no crea migraciones y no modifica datos históricos. Impacto en iHost: nulo en memoria y almacenamiento.

> **Las reglas no se copian en la spec.** Se citan por ID desde `docs/rules/` y se declaran en la sección 2.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Sin dependencias nuevas; React + Tailwind + Go + SQLite existentes | `CA-ARCH-STACK` |
| `ARCH-LAYERS` | N/A: solo cambia la capa de presentación; no se agrega lógica de negocio ni SQL | — |
| `ARCH-MODULES` | N/A: se modifica una página legacy (`internal/api` no cambia); no se crea un módulo nuevo | — |
| `ARCH-IHOST` | Cambio de UI sin migración, sin proceso nuevo y sin dependencias; impacto nulo en RAM/almacenamiento | `CA-ARCH-IHOST` |
| `UI-SELECT` | El selector de casa y el de institución siguen usando `Select`; la categoría del alta rápida también usa `Select` searchable | `CA-UI-SELECT` |
| `UI-SELECT-ADD` | La opción de crear institución se rotula `app.add` («Agregar»), lleva ícono `plus`, usa `addOption` y va al final del dropdown | `CA-UI-SELECT-ADD` |
| `UI-NUMBER-INPUT` | N/A: la spec no agrega ni modifica campos numéricos; `suggested_amount` y `billing_day` quedan fuera de alcance | — |
| `UI-INPUT-TOKENS` | El input de nombre del alta rápida usa `bg-card`, `text-text`, `border-border` y placeholder `text-text-secondary` | `CA-UI-INPUT-TOKENS` |
| `UI-DARKMODE` | El dropdown y el modal de alta rápida se verifican en darkmode (texto, placeholder y bordes legibles) | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | N/A: no se agregan booleanos de entidad ni de configuración | — |
| `UI-LIST-PATTERN` | N/A: no se modifica el listado de servicios; solo cambian los destinos de creación | — |
| `UI-CREATE-MENU` | El botón de crear del header sigue siendo `CreateMenu`; solo se le agrega el parámetro `home` al destino | `CA-UI-CREATE-MENU` |
| `UI-EDIT-LABEL` | N/A: no se agregan ni cambian acciones de edición | — |
| `UI-I18N` | Se reutilizan claves existentes (`app.add`, `institutions.*`, `app.saving`, `app.save`, `app.cancel`); no se agregan claves nuevas. `check:i18n` pasa | `CA-UI-I18N` |
| `UX-BACK` | N/A: no se agregan rutas; `/services/new` ya está registrada en `BACK_ROUTES` | — |
| `UX-MODAL` | El alta rápida es un modal anidado `z-[70]`: no cierra por click fuera, tiene X y Cancelar visibles, overlay bloquea el fondo y usa la escala de capas | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | N/A: no hay confirmación por palabra | — |
| `UX-DETAIL-ACTIONS` | N/A: no se modifican páginas de detalle | — |
| `UX-DETAIL-READONLY` | N/A: no se modifican páginas de detalle | — |
| `UX-PREREQ` | El backend ya exige al menos una institución (`412 NO_INSTITUTIONS`); el frontend mantiene `DependencyWarning` cuando no hay ninguna y permite crear una desde el dropdown cuando el formulario está disponible | `CA-UX-PREREQ` |
| `UX-EMPTY` | El estado vacío del listado de servicios se mantiene; su botón de crear ahora conserva el filtro activo | `CA-UX-EMPTY` |
| `UX-LOADING` | El botón Guardar del alta rápida se deshabilita mientras crea; el formulario conserva su estado de carga | `CA-UX-LOADING` |
| `UX-FEEDBACK` | Alta exitosa muestra toast `institutions.created`; nombre vacío y errores de API se muestran junto al campo | `CA-UX-FEEDBACK` |
| `UX-FORMS` | El servicio conserva Guardar/Cancelar; el alta rápida tiene Cancelar/Crear como acciones secundaria/primaria | `CA-UX-FORMS` |
| `DATA-MIGRATIONS` | N/A: no se crean tablas ni columnas; `institutions` y `services.institution_id` ya existen | — |
| `DATA-NO-DROP` | No se toca `data/app.db`; las pruebas usan fixtures sintéticos y `/tmp` si hiciera falta | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | N/A: el comportamiento no depende del contenido real de la DB; se valida con datos sintéticos | — |
| `DATA-PROTECTED-SCHEMA` | N/A: no se modifica `debts` / `debt_bills` | — |
| `SEC-NO-PII` | Spec, issue, pruebas y capturas usan solo datos sintéticos (`Casa Demo`, `Proveedor Demo`, `demo@example.com`) | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con datos `demo-*` | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` | `git diff --cached` revisado antes del commit | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `check:ui`, `check:i18n`, `check:numbers` y build del frontend | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | `CA-TEST-SERVER` |
| `TEST-KILLALL` | Solo `killall`, nunca `pkill` | `CA-TEST-KILLALL` |
| `TEST-USER-FLOW` | Se pregunta al usuario si quiere probar en local antes de cerrar la spec | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo exclusivo en `../p40la-ihost-spec-144` (`feature/SPEC-144`) | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita del usuario | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Mensaje `SPEC-144: ...` | `CA-DELIV-COMMIT` |
| `DELIV-RELEASE` | Release con label `spec/released`, cierre del issue y limpieza del worktree | `CA-DELIV-RELEASE` |
| `DELIV-HOOKS` | Hooks instalados, sin `--no-verify` | `CA-DELIV-HOOKS` |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En `ServicesPage`, todos los accesos de creación de servicio (menú de crear del header, estado vacío sin servicios y estado vacío de una vista filtrada) deben propagar el filtro de casa activo como `?home=<id>` al navegar a `/services/new`.
2. **REQ-002**: `ServiceFormPage` debe leer el parámetro `home` de la URL. En modo creación, si el valor corresponde a una casa existente, debe inicializar `home_id` con esa casa. Si el parámetro falta o no corresponde a una casa existente, debe mantener el comportamiento actual (primera casa) y permitir elegir cualquier casa.
3. **REQ-003**: En modo edición, el parámetro `home` debe ignorarse; la casa del formulario es la del servicio que se edita.
4. **REQ-004**: El dropdown de institución del formulario de servicio debe incluir al final la opción `addOption(t('app.add'))` («Agregar», ícono `plus`, valor centinela `ADD_OPTION_VALUE`). Seleccionarla no debe cambiar la institución seleccionada actualmente.
5. **REQ-005**: Al elegir «Agregar» se debe abrir un modal de alta rápida de institución con: nombre (obligatorio, se recorta el espacio), categoría (opcional, `Select` searchable con «Sin categoría»), X de cierre, Cancelar y Crear. El modal no se cierra por click fuera, usa `z-[70]`, enfoca el nombre al abrirse y muestra errores junto al campo.
6. **REQ-006**: Al guardar el alta rápida se debe llamar a `POST /api/institutions` con `{ name, category_id }`, refrescar la lista de instituciones, seleccionar la institución creada en el formulario de servicio, reiniciar `institution_analyzer_id`, mostrar el toast `institutions.created` y cerrar el modal. Los datos ya cargados del servicio no se pierden.
7. **REQ-007**: Si no hay ninguna institución, se mantiene el bloqueo actual con `DependencyWarning` y navegación a `/institutions/new`; el backend ya rechaza la creación de servicios con `412 NO_INSTITUTIONS`. El alta rápida aplica cuando el formulario está disponible (al menos una institución existente).

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: Al guardar o cancelar un servicio creado desde un listado filtrado, volver a `/services?home=<id>` para conservar el filtro de casa.
2. **REQ-009**: El alta rápida debe preseleccionar «Sin categoría» y no obligar a elegir una categoría.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Extraer el alta rápida de institución a un componente compartido para reutilizarlo en otros dropdowns (reparaciones del hogar, rentas, automatizaciones). **Fuera de alcance de esta spec.**

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: una llamada extra a `GET /api/institution-categories` al montar el formulario y una recarga de `GET /api/institutions` después del alta; sin impacto perceptible.
- **Seguridad**: endpoints autenticados existentes; sin permisos nuevos.
- **Almacenamiento**: cero migraciones, cero tablas nuevas.
- **iHost**: sin dependencias nuevas; RAM/CPU sin cambios significativos.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `frontend/src/pages/ServicesPage.tsx`: el filtro de casa vive en `?home=<id>`; `CreateMenu` y los dos estados vacíos navegan a `/services/new` sin propagarlo.
- `frontend/src/pages/ServiceFormPage.tsx`: en creación inicializa `home_id: homes[0].id`; ignora la URL. El dropdown de institución se arma con `institutions.map(...)` sin opción de alta.
- `frontend/src/pages/LeaseFormPage.tsx`: precedente de `?home=` en la URL (`queryHomeID`) y de alta rápida de institución con `addOption` + modal + `api.institutions.create` + recarga.
- `frontend/src/components/home/HomeMaintenanceItemModal.tsx`: precedente de `addOption(t('app.add'))`, `ADD_OPTION_VALUE` y modal de alta rápida con nombre/categoría.
- `internal/api/service_handlers.go`: `CreateService` devuelve `412 NO_INSTITUTIONS` cuando no hay instituciones; `internal/services/institution.go` exige nombre no vacío.
- La pantalla de creación actual es una página (`/services/new`), no un modal. La spec aplica el mismo comportamiento si en el futuro se presenta como modal.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Preseleccionar la casa filtrada y permitir cambiarla | Conserva el contexto sin bloquear correcciones | El usuario podría cambiarla por error | ✅ Seleccionada |
| Bloquear la casa cuando hay filtro (como `LeaseFormPage`) | Evita errores de casa | Impide corregir sin volver atrás | ❌ Rechazada |
| Navegar a `/institutions/new` y volver | Cero código nuevo | Pierde el formulario de servicio y su estado | ❌ Rechazada |
| Modal de alta rápida local a `ServiceFormPage` | Bajo acoplamiento, sigue el patrón de SPEC-141 | Duplica ~50 líneas de UI | ✅ Seleccionada |
| Extraer un componente compartido ahora | Menos duplicación futura | Amplía el alcance y toca otros módulos | ❌ Rechazada (P2) |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: La casa filtrada se **preselecciona**, no se bloquea.
- **Contexto**: el listado filtrado comunica intención, pero el formulario debe permitir corregirla.
- **Decisión**: inicializar `home_id` con `?home=` válido y dejar el `Select` editable.
- **Consecuencias**: menos fricción; el usuario conserva el control.

**ADR-002**: El alta rápida es local a `ServiceFormPage`.
- **Contexto**: `UI-SELECT-ADD` exige «Agregar» en el dropdown y `UX-PREREQ` exige resolver el prerrequisito sin callejones sin salida.
- **Decisión**: modal anidado `z-[70]` con nombre/categoría, crear → recargar lista → seleccionar → cerrar.
- **Consecuencias**: no se pierde lo escrito; se evita acoplar la página legacy a la UI de rentas.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
[ServicesPage ?home=<id>] --(navega con ?home)--> [ServiceFormPage]
                                                        |
                                                        +--> GET /api/institutions
                                                        +--> GET /api/institution-categories
                                                        |
                                                        +--> [Modal alta rápida]
                                                               |
                                                               +--> POST /api/institutions
                                                               +--> GET /api/institutions (refresh)
                                                        |
                                                        +--> POST /api/services
```

### 5.2 Componentes

#### 5.2.1 `frontend/src/pages/ServicesPage.tsx`
- **Responsabilidad**: construir el destino de creación con el filtro activo y usarlo en `CreateMenu` y en ambos estados vacíos.
- **Cambio**: helper `createServicePath = homeId ? '/services/new?home=' + homeId : '/services/new'`.
- **Ubicación**: `frontend/src/pages/ServicesPage.tsx`.

#### 5.2.2 `frontend/src/pages/ServiceFormPage.tsx`
- **Responsabilidad**: inicializar la casa desde la URL en creación; agregar la opción «Agregar» al dropdown de institución y el modal de alta rápida.
- **Estado nuevo**: `quickOpen`, `quickName`, `quickCategoryId`, `quickSaving`, `quickError`, `categories`.
- **Ubicación**: `frontend/src/pages/ServiceFormPage.tsx`.
- **Dependencias**: `useSearchParams`, `addOption`, `ADD_OPTION_VALUE`, `api.institutions`, `api.institutionCategories`, `Select`, `Toast`.

#### 5.2.3 `frontend/public/i18n/{es,en}.json`
- **Responsabilidad**: N/A. Se reutilizan claves existentes. Si durante la implementación se necesita un título específico, usar `institutions.create` («Nueva institución»), que ya existe en ambos idiomas.

### 5.3 Modelo de datos

Sin cambios de esquema.

```
institutions
- id: INTEGER PK
- name: TEXT NOT NULL
- category_id: INTEGER NULL → institution_categories(id)

services
- home_id: INTEGER NOT NULL → homes(id)
- institution_id: INTEGER NULL → institutions(id)
```

### 5.4 APIs / Contratos

#### `GET /api/institutions`
Lista para el dropdown y refresh posterior al alta.

#### `GET /api/institution-categories`
Lista para el `Select` de categoría del alta rápida.

#### `POST /api/institutions`

**Request**:
```json
{
  "name": "Proveedor Demo",
  "category_id": null
}
```

**Response 201**:
```json
{
  "id": 1,
  "name": "Proveedor Demo",
  "category_id": null,
  "created_at": "2026-11-23T00:00:00Z",
  "updated_at": "2026-11-23T00:00:00Z"
}
```

**Response 400**:
```json
{
  "error": "invalid_request",
  "message": "el nombre de la institución es requerido"
}
```

#### `POST /api/services`
Sin cambios. Sigue devolviendo `412 NO_INSTITUTIONS` si no existe ninguna institución.

### 5.5 Dependencias

- **Internas**: `api.institutions`, `api.institutionCategories`, `Select` (`addOption`), `Toast`, i18n.
- **Externas**: ninguna nueva.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [ ] `CA-REQ-001`: Con `/services?home=1` activo, el menú de crear y ambos botones de estado vacío navegan a `/services/new?home=1`.
- [ ] `CA-REQ-002`: Al abrir `/services/new?home=1` con una casa existente, el `Select` de casa muestra esa casa seleccionada.
- [ ] `CA-REQ-003`: Al abrir `/services/new` sin `home` o con un id inexistente, el formulario mantiene el comportamiento actual y permite elegir cualquier casa.
- [ ] `CA-REQ-004`: Al editar un servicio, la casa mostrada es la del servicio aunque la URL tenga otro `home`.
- [ ] `CA-UI-SELECT-ADD`: El dropdown de institución muestra «Agregar» con ícono `plus` al final; al elegirla no cambia la institución seleccionada y se abre el modal.
- [ ] `CA-REQ-005`: El modal de alta rápida no cierra por click fuera; tiene X y Cancelar visibles; usa `z-[70]`; enfoca el nombre; el nombre vacío muestra error y no guarda.
- [ ] `CA-REQ-006`: Al crear una institución, el dropdown se refresca, la nueva institución queda seleccionada, `institution_analyzer_id` se reinicia, aparece el toast y el modal se cierra sin perder los datos del servicio.
- [ ] `CA-UX-PREREQ`: Con cero instituciones se mantiene la advertencia y el backend responde `412 NO_INSTITUTIONS`; el alta rápida no reemplaza esa validación.
- [ ] `CA-UX-MODAL`: El modal respeta la escala de capas (modal `z-[70]`, `Select` `z-[90]`, toasts `z-[100]`) y el overlay bloquea el fondo.
- [ ] `CA-UX-FEEDBACK`: Éxito con toast `institutions.created`; error de API visible dentro del modal.
- [ ] `CA-UX-LOADING`: Mientras se guarda, el botón Crear se deshabilita y muestra `app.saving`.
- [ ] `CA-UX-FORMS`: El formulario de servicio conserva Guardar/Cancelar; el modal tiene Cancelar/Crear.
- [ ] `CA-UI-SELECT`: Todos los selectores usan `Select`; no hay `<select>` nativo.
- [ ] `CA-UI-INPUT-TOKENS`: El input de nombre usa tokens del tema y placeholder legible.
- [ ] `CA-UI-DARKMODE`: Dropdown y modal verificados en darkmode.
- [ ] `CA-UI-I18N`: `check:i18n` pasa; no hay claves nuevas ni paridad rota.
- [ ] `CA-UI-CREATE-MENU`: El header de servicios sigue usando `CreateMenu`, sin botón directo de crear.
- [ ] `CA-UX-EMPTY`: Los estados vacíos de servicios se mantienen y conservan el filtro.
- [ ] `CA-ARCH-STACK`: Sin dependencias nuevas.
- [ ] `CA-ARCH-IHOST`: Sin migraciones, procesos ni dependencias nuevas.
- [ ] `CA-DATA-NO-DROP`: No se toca `data/app.db`.
- [ ] `CA-SEC-NO-PII`: Sin datos reales en código, spec ni issue.
- [ ] `CA-SEC-SYNTHETIC`: Ejemplos `demo-*`.
- [ ] `CA-SEC-COMMIT`: `git diff --cached` limpio.

### 6.2 No funcionales

- [ ] `CA-NF-001`: `go build ./...` y `go test ./...` pasan.
- [ ] `CA-TEST-LOCAL`: `check:ui`, `check:i18n`, `check:numbers` y build de Vite pasan.
- [ ] `CA-TEST-SERVER`: `./scripts/check-server.sh` responde `{"status":"ok"}` antes de avisar.
- [ ] `CA-TEST-KILLALL`: Se usa `killall`.
- [ ] `CA-TEST-USER-FLOW`: Se pide confirmación al usuario antes de cerrar.
- [ ] `CA-DELIV-WORKTREE`: Trabajo en `feature/SPEC-144`.
- [ ] `CA-DELIV-CONFIRM`: Merge/push confirmados.
- [ ] `CA-DELIV-COMMIT`: Mensaje `SPEC-144: ...`.
- [ ] `CA-DELIV-RELEASE`: Release completo con label y cierre del issue.
- [ ] `CA-DELIV-HOOKS`: Hooks instalados, sin `--no-verify`.

### 6.3 Testing

- **Unit tests**: N/A para lógica de negocio (no hay cambios Go). Si se extrae un helper de inicialización de casa, cubrirlo con test de frontend si el proyecto lo permite.
- **Integration/manual**: crear servicio desde `/services?home=1` y verificar casa preseleccionada; sin filtro, verificar comportamiento actual; crear institución desde el dropdown y verificar selección + toast; editar servicio y verificar que la URL no pisa la casa.
- **E2E/manual**: darkmode del dropdown y del modal; teclado (Tab/Enter/Escape) y cierre explícito; error de nombre vacío.
- **Carga/Performance**: sin métricas nuevas; verificar que el formulario no hace llamadas en bucle.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | `ServicesPage`: propagar `?home=` en los tres accesos de creación | 0.25 día | Ninguna |
| 2 | `ServiceFormPage`: leer `?home=`, validar y preseleccionar casa | 0.25 día | Fase 1 |
| 3 | `ServiceFormPage`: `addOption` + modal de alta rápida + refresh/selección | 1 día | Fase 2 |
| 4 | i18n (si hace falta), darkmode y guardrails | 0.5 día | Fase 3 |
| 5 | Pruebas manuales con el usuario y release | 0.5 día | Fases 1-4 |

### 7.2 Milestones

1. **MVP**: casa filtrada preseleccionada y alta rápida de institución funcionando.
2. **V1.0**: mismo alcance con darkmode verificado, guardrails en verde y filtro conservado al volver (P1).

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| `?home=` con un id de casa eliminada | Media | Bajo | Validar contra `homes` cargadas y caer al comportamiento actual |
| El modal queda detrás del `Select` | Baja | Medio | Usar `z-[70]` para el modal y `z-[90]` para `Select` |
| Duplicar instituciones con el mismo nombre | Media | Bajo | El API actual no impone unicidad; se acepta y se documenta. No se agrega validación en esta spec |
| Pérdida de datos del servicio al abrir el modal | Baja | Alto | El modal es secundario; el formulario no se desmonta |
| Divergencia i18n es/en | Baja | Medio | `check:i18n` en el gate |
| El filtro se pierde al guardar/cancelar | Media | Bajo | REQ-008 (P1) conserva `?home=` al volver |

---

## 9. Notas y Referencias

- Reglas: `docs/rules/20-ui.md`, `30-ux.md`, `10-architecture.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Precedentes: SPEC-140 (`UI-SELECT-ADD`), SPEC-141 (alta rápida de institución en reparaciones), SPEC-142 (mismo patrón en líneas de reparación), SPEC-126 (`UX-MODAL`).
- Issue de GitHub: #147.

### Release

- **Estado**: released (2026-11-23)
- **Commit de implementación**: `689c0093`
- **Merge a `main`**: `856b713c`
- **Versión**: 0.5.17
- **Migraciones**: ninguna
- **Issue**: #147 (cerrado)
- **Archivos modificados**:
  - `frontend/src/pages/ServicesPage.tsx`: `createServicePath` con `?home=` propagado al menú de crear y a los dos estados vacíos.
  - `frontend/src/pages/ServiceFormPage.tsx`: lectura de `?home=`, preselección de la casa filtrada, `addOption(t('app.add'))` en el dropdown de institución, modal de alta rápida y retorno al listado filtrado (REQ-008, P1).
- **Backend**: sin cambios.
- **i18n**: sin claves nuevas; se reutilizan `app.add`, `app.create`, `app.saving`, `app.cancel`, `app.close`, `institutions.*` y `errors.generic`.
- **Validación**: `go build ./...`, `go test ./...`, `check:ui`, `check:i18n`, `check:numbers` y build de Vite en verde; smoke test de API en `http://localhost:8090` con base sintética en `/tmp` (alta de institución y servicio). Release aprobado por el usuario.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | opencode | Creación inicial de la especificación |
| 2026-11-23 | opencode | Inicio de implementación; estado `in_progress` y label `spec/in-progress` |
| 2026-11-23 | opencode | Implementación de P0 y P1; guardrails `go build`, `go test`, `check:ui`, `check:i18n`, `check:numbers` y build en verde. Pendiente validación manual del usuario |
| 2026-11-23 | opencode | Release: merge `856b713c` a `main`, estado `released`, versión 0.5.17, issue #147 cerrado y tracker actualizado |
