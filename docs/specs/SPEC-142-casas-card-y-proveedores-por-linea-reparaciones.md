---
title: "Casas: card sin link Ver y proveedores por línea en reparaciones"
id: "SPEC-142"
status: "pending_release"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 145
---

# Casas: card sin link Ver y proveedores por línea en reparaciones

**ID**: SPEC-142  
**Estado**: pending_release  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

La pantalla de casas (`/home`) muestra en cada card un enlace visual «Ver» además de que toda la card ya navega al detalle. Ese enlace es redundante y agrega ruido; se elimina y la card completa queda como único punto de navegación.

En el módulo de reparaciones del hogar (SPEC-141), el proveedor/contratista se cargaba a nivel de la reparación y solo los insumos podían indicar dónde se compraron. En la práctica una misma reparación combina trabajos de distintos técnicos y compras en distintos comercios. Por eso el proveedor pasa a definirse **por línea** (mano de obra e insumos), se elimina del formulario de la reparación y cada línea puede crear una institución desde el mismo dropdown sin salir del flujo.

Además, los accesos «Agregar insumo» y «Agregar mano de obra» dejan de ser botones directos y pasan a un menú de 3 puntos con la opción «Agregar», alineado con `UX-DETAIL-ACTIONS` y `UI-SELECT-ADD`.

El cambio es exclusivamente de frontend y de una validación del módulo Go existente: **no agrega dependencias, no crea migraciones** (las columnas `institution_id` ya existen en ambas tablas) y no altera datos históricos. Impacto en iHost: nulo en memoria y almacenamiento.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables (índice: `docs/rules/00-index.md`).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Sin dependencias nuevas; React + Go + SQLite existentes | `CA-ARCH-STACK` |
| `ARCH-LAYERS` | El ajuste de validación vive en `service.go` del módulo `home.maintenance`; el SQL no cambia | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | Se mantiene `home.maintenance` con `handlers → service → storage` | `CA-ARCH-MODULES` |
| `ARCH-IHOST` | Cambio de UI y validación; sin migración ni proceso nuevo; memoria/almacenamiento sin impacto | `CA-ARCH-IHOST` |
| `UI-SELECT` | Todos los selectores siguen usando `Select` (searchable) y no hay `<select>` nativo | `CA-UI-SELECT` |
| `UI-SELECT-ADD` | La opción de crear institución usa `addOption(t('app.add'))`, al final del dropdown y con ícono `plus` | `CA-UI-SELECT-ADD` |
| `UI-NUMBER-INPUT` | Cantidades y montos siguen con `NumberInput`; no se tocan | `CA-UI-NUMBER-INPUT` |
| `UI-INPUT-TOKENS` | Inputs nuevos (nombre de institución) usan `bg-card`, `text-text`, `border-border` | `CA-UI-INPUT-TOKENS` |
| `UI-DARKMODE` | El modal de alta rápida y el dropdown se verifican en darkmode | `CA-UI-DARKMODE` |
| `UI-TOGGLE` | N/A: la spec no agrega booleanos de entidad | — |
| `UI-LIST-PATTERN` | La card de casas mantiene el patrón de cards y su menú de 3 puntos | `CA-UI-LIST-PATTERN` |
| `UI-CREATE-MENU` | El header de casas no cambia; el menú de acciones del detalle usa el patrón de 3 puntos existente | `CA-UI-CREATE-MENU` |
| `UI-EDIT-LABEL` | Las acciones de edición siguen rotuladas `app.edit` | `CA-UI-EDIT-LABEL` |
| `UI-I18N` | Claves nuevas/eliminadas en `frontend/public/i18n/{es,en}.json` con paridad; `check:i18n` pasa | `CA-UI-I18N` |
| `UX-BACK` | N/A: no se agregan rutas ni páginas nuevas; el detalle conserva `BACK_ROUTES` | — |
| `UX-MODAL` | Modal de alta rápida anidado con `z-[70]`, sin cierre por click fuera, con X y Cancelar visibles | `CA-UX-MODAL` |
| `UX-MODAL-CONFIRM` | N/A: no hay confirmación por palabra | — |
| `UX-DETAIL-ACTIONS` | «Agregar insumo/mano de obra» pasa al menú de 3 puntos de la sección; Editar/Eliminar siguen en el menú de cada línea | `CA-UX-DETAIL-ACTIONS` |
| `UX-DETAIL-READONLY` | El detalle sigue siendo de solo lectura; el alta abre modal | `CA-UX-DETAIL-READONLY` |
| `UX-PREREQ` | La institución se valida en backend (`InstitutionExists`) y se puede crear desde el frontend sin abandonar el modal | `CA-UX-PREREQ` |
| `UX-EMPTY` | El estado vacío de líneas («Sin líneas registradas») se mantiene | `CA-UX-EMPTY` |
| `UX-FEEDBACK` | Alta de institución y de línea muestran toast de éxito; errores de validación junto al campo | `CA-UX-FEEDBACK` |
| `UX-FORMS` | El formulario de reparación conserva una acción primaria (Guardar) y una secundaria (Cancelar) | `CA-UX-FORMS` |
| `DATA-MIGRATIONS` | N/A: `home_maintenances.institution_id` y `home_maintenance_items.institution_id` ya existen (migración 0057); no se agrega migración | — |
| `DATA-NO-DROP` | No se toca `data/app.db`; la validación usa fixtures sintéticos y `/tmp` si hiciera falta | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | N/A: el cambio no depende del contenido de la DB para decidir comportamiento; se valida en local con datos sintéticos | — |
| `DATA-PROTECTED-SCHEMA` | N/A: no se modifica `debts` / `debt_bills` | — |
| `SEC-NO-PII` | Spec, issue, pruebas y capturas usan solo datos sintéticos (`Proveedor Demo`, `demo@example.com`) | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con datos `demo-*` | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` | `git diff --cached` revisado antes del commit | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `npm run check:ui`, `check:i18n`, `check:numbers` y build del frontend | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | `CA-TEST-SERVER` |
| `TEST-KILLALL` | Solo `killall`, nunca `pkill` | `CA-TEST-KILLALL` |
| `TEST-USER-FLOW` | Se pregunta al usuario si quiere probar en local antes de cerrar la spec | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo exclusivo en `../p40la-ihost-spec-142` (`feature/SPEC-142`) | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita del usuario | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Mensaje `SPEC-142: ...` | `CA-DELIV-COMMIT` |
| `DELIV-RELEASE` | Release con label `spec/released`, cierre de issue y limpieza del worktree | `CA-DELIV-RELEASE` |
| `DELIV-HOOKS` | Hooks instalados, sin `--no-verify` | `CA-DELIV-HOOKS` |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En el listado de casas (`HomesPage`), quitar el enlace visual «Ver» (texto + chevron) de cada card. La card completa sigue navegando a `/home/:id` con click y con Enter/Espacio.
2. **REQ-002**: El modal de línea de reparación (insumo y mano de obra) debe mostrar el selector de institución (proveedor/contratista) para **ambos** tipos de línea.
3. **REQ-003**: El dropdown de institución debe incluir al final la opción `app.add` («Agregar»), construida con `addOption`, que abre un modal de alta rápida.
4. **REQ-004**: El alta rápida debe crear la institución vía `POST /api/institutions`, refrescar la lista del dropdown, seleccionar la institución creada y cerrar el modal secundario, sin salir del modal de línea.
5. **REQ-005**: Eliminar el campo «Proveedor / contratista» de «Nueva reparación» y «Editar reparación» (`HomeMaintenanceFormPage`), y dejar de mostrar el proveedor a nivel de reparación en el detalle.
6. **REQ-006**: «Agregar insumo» y «Agregar mano de obra» dejan de ser botones directos: cada sección muestra un menú de 3 puntos que despliega la opción «Agregar» y abre el modal de línea correspondiente.
7. **REQ-007**: El backend debe aceptar `institution_id` en líneas de mano de obra (hoy lo rechaza con `labor_institution`), validando que la institución exista.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: El modal de alta rápida permite elegir categoría (opcional) y preselecciona `hardware` para insumos y `maintenance` para mano de obra cuando esas categorías existen.
2. **REQ-009**: Los registros históricos con proveedor a nivel de reparación se conservan en la DB; la UI simplemente deja de mostrarlos.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Replicar el mismo patrón de proveedor por línea en reparaciones de autos (`AutoMaintenance*`) en una spec futura. **Fuera de alcance de esta spec.**

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: una llamada extra `GET /api/institutions` solo al abrir el modal y después del alta; sin impacto perceptible.
- **Seguridad**: endpoints autenticados existentes; sin nuevos permisos.
- **Almacenamiento**: cero migraciones y cero nuevas tablas.
- **iHost**: sin dependencias nuevas; RAM/CPU sin cambios significativos.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `frontend/src/pages/HomesPage.tsx`: la card ya navega; el span «Ver» es solo affordance visual.
- `frontend/src/components/home/HomeMaintenanceItemModal.tsx`: el selector de institución está dentro de `isSupplies` y ofrece un enlace externo a `/institutions/new` cuando no hay instituciones.
- `frontend/src/pages/HomeMaintenanceFormPage.tsx`: proveedor a nivel de reparación con `institution_id`.
- `internal/modules/home/maintenance/service.go`: `prepareItem` rechaza institución en mano de obra; `storage.go` ya persiste `institution_id` para ambos tipos.
- Patrón de alta rápida ya usado en `LeaseFormPage` (`addOption` + modal + `api.institutions.create` + recarga de lista).

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Alta rápida con modal propio del módulo home | Bajo acoplamiento, z-index correcto, sin tocar rentas | Duplica ~60 líneas de UI | ✅ Seleccionada |
| Reutilizar `Modal` de `components/rental/RentalUI` | Menos código | Acopla hogar a UI de rentas | ❌ Rechazada |
| Navegar a `/institutions/new` y volver | Cero código nuevo | Pierde el formulario de línea y el contexto | ❌ Rechazada |
| Mantener proveedor a nivel de reparación y agregar por línea | Compatibilidad | Dos fuentes de verdad y confusión | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El proveedor/contratista es un atributo de la línea, no de la reparación.
- **Contexto**: una reparación puede tener varios técnicos y comercios.
- **Decisión**: la UI deja de capturar y mostrar `home_maintenances.institution_id`; `home_maintenance_items.institution_id` aplica a `labor` y `supplies`.
- **Consecuencias**: no hay migración ni pérdida de datos; el campo histórico queda conservado pero oculto.

**ADR-002**: El alta rápida de institución es local al modal de línea.
- **Contexto**: `UI-SELECT-ADD` exige «Agregar» en el dropdown y `UX-PREREQ` exige poder resolver el prerrequisito sin callejones sin salida.
- **Decisión**: modal anidado `z-[70]`, crear → recargar lista → seleccionar → cerrar.
- **Consecuencias**: el usuario no pierde lo escrito en la línea.

---

## 5. Diseño Técnico

### 5.1 Componentes

#### 5.1.1 `frontend/src/pages/HomesPage.tsx`
- **Responsabilidad**: quitar el span «Ver» y el chevron; conservar `role="link"`, `tabIndex`, click y teclado.
- **Ubicación**: `frontend/src/pages/HomesPage.tsx`.

#### 5.1.2 `frontend/src/components/home/HomeMaintenanceItemModal.tsx`
- **Responsabilidad**: mostrar proveedor para ambos tipos; `addOption`; modal de alta rápida; enviar `institution_id` siempre.
- **Estado nuevo**: `quickOpen`, `quickName`, `quickCategoryId`, `quickSaving`, `quickError`.
- **Ubicación**: `frontend/src/components/home/HomeMaintenanceItemModal.tsx`.

#### 5.1.3 `frontend/src/pages/HomeMaintenanceShowPage.tsx`
- **Responsabilidad**: reemplazar el botón de agregar por `CreateMenu` con una opción `app.add`; quitar el proveedor del encabezado.
- **Ubicación**: `frontend/src/pages/HomeMaintenanceShowPage.tsx`.

#### 5.1.4 `frontend/src/pages/HomeMaintenanceFormPage.tsx`
- **Responsabilidad**: eliminar estado, fetch y campo de proveedor; enviar `institution_id: null`.
- **Ubicación**: `frontend/src/pages/HomeMaintenanceFormPage.tsx`.

#### 5.1.5 `internal/modules/home/maintenance/service.go`
- **Responsabilidad**: `prepareItem` acepta y valida institución para ambos tipos; mantiene `quantity = 1` y `unit = unit` en mano de obra.
- **Ubicación**: `internal/modules/home/maintenance/service.go`.

### 5.2 Modelo de datos

Sin cambios de esquema. Columnas existentes:

```
home_maintenances.institution_id      INTEGER NULL  (legado, oculto en UI)
home_maintenance_items.institution_id INTEGER NULL  (aplica a labor y supplies)
```

### 5.3 APIs / Contratos

- `GET /api/institutions` → lista para el dropdown.
- `POST /api/institutions` `{ "name": "Proveedor Demo", "category_id": 1 }` → institución creada.
- `POST|PUT /api/homes/:id/maintenances/:mid/items` acepta `institution_id` en `labor` y `supplies`.
- `POST|PUT /api/homes/:id/maintenances` acepta `institution_id: null` (compatibilidad).

### 5.4 Dependencias

- **Internas**: `api.institutions`, `Select` (`addOption`), `CreateMenu`, `Toast`, i18n.
- **Externas**: ninguna nueva.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [ ] `CA-REQ-001`: En `/home`, ninguna card muestra «Ver»; click y Enter navegan a `/home/:id`.
- [ ] `CA-REQ-002`: Al agregar/editar mano de obra aparece el selector de contratista; al agregar/editar insumo aparece el de comercio.
- [ ] `CA-UI-SELECT-ADD`: El dropdown muestra «Agregar» con ícono `plus` al final y crea la institución, refresca la lista y la deja seleccionada.
- [ ] `CA-REQ-005`: «Nueva reparación» y «Editar reparación» no muestran proveedor/contratista; el detalle tampoco lo muestra.
- [ ] `CA-UX-DETAIL-ACTIONS`: Cada sección (mano de obra / insumos) tiene un menú de 3 puntos con la opción «Agregar».
- [ ] `CA-REQ-007`: `POST` de línea `labor` con `institution_id` válido responde 2xx; con institución inexistente responde error de dominio.
- [ ] `CA-UI-SELECT`: Todos los selectores usan `Select`; `npm run check:ui` pasa.
- [ ] `CA-UI-NUMBER-INPUT`: `npm run check:numbers` pasa.
- [ ] `CA-UI-I18N`: `npm run check:i18n` pasa; es/en en paridad.
- [ ] `CA-UI-INPUT-TOKENS`: Inputs con tokens del tema; verificado en darkmode.
- [ ] `CA-UI-DARKMODE`: Modal y dropdown legibles en darkmode.
- [ ] `CA-UI-EDIT-LABEL`: Acciones de edición con `app.edit`.
- [ ] `CA-UI-LIST-PATTERN`: Cards de casas con menú de 3 puntos intacto.
- [ ] `CA-UI-CREATE-MENU`: Header de casas sin botón directo de crear.
- [ ] `CA-UX-MODAL`: El modal de alta rápida no cierra por click fuera, tiene X y Cancelar, overlay bloquea y usa `z-[70]`.
- [ ] `CA-UX-DETAIL-READONLY`: El detalle no edita; el alta abre modal.
- [ ] `CA-UX-PREREQ`: Institución validada en backend y creable desde el frontend.
- [ ] `CA-UX-EMPTY`: El estado vacío de líneas se mantiene.
- [ ] `CA-UX-FEEDBACK`: Toast al crear institución y al guardar línea; error visible en el modal.
- [ ] `CA-UX-FORMS`: Formulario con Guardar/Cancelar.
- [ ] `CA-DATA-NO-DROP`: No se toca `data/app.db`.
- [ ] `CA-SEC-NO-PII`: Sin datos reales en código, spec ni issue.
- [ ] `CA-SEC-SYNTHETIC`: Ejemplos `demo-*`.
- [ ] `CA-SEC-COMMIT`: `git diff --cached` limpio.
- [ ] `CA-ARCH-STACK`: Sin dependencias nuevas.
- [ ] `CA-ARCH-LAYERS`: Lógica en el servicio, SQL intacto.
- [ ] `CA-ARCH-MODULES`: Módulo `home.maintenance` con sus capas.
- [ ] `CA-ARCH-IHOST`: Sin migraciones ni procesos nuevos.

### 6.2 No funcionales

- [ ] `CA-NF-001`: `go build ./...` y `go test ./...` pasan.
- [ ] `CA-TEST-LOCAL`: Guardrails de UI y build del frontend pasan.
- [ ] `CA-TEST-SERVER`: `./scripts/check-server.sh` responde `{"status":"ok"}` antes de avisar.
- [ ] `CA-TEST-KILLALL`: Se usa `killall`.
- [ ] `CA-TEST-USER-FLOW`: Se pide confirmación al usuario antes de cerrar.
- [ ] `CA-DELIV-WORKTREE`: Trabajo en `feature/SPEC-142`.
- [ ] `CA-DELIV-CONFIRM`: Merge/push confirmados.
- [ ] `CA-DELIV-COMMIT`: Mensaje `SPEC-142: ...`.
- [ ] `CA-DELIV-RELEASE`: Release completo.
- [ ] `CA-DELIV-HOOKS`: Hooks instalados, sin `--no-verify`.

### 6.3 Testing

- **Unit tests** (`service_test.go`): línea `labor` con institución válida se guarda; línea `labor` con institución inexistente devuelve `ErrInstitutionNotFound`.
- **Integration/manual**: crear insumo y mano de obra con proveedor distinto; crear institución desde el dropdown y verificar refresco; editar reparación sin campo proveedor.
- **E2E/manual**: card de casa navega sin link «Ver»; menús de 3 puntos despliegan «Agregar».

---

## 7. Plan de Implementación

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Backend: permitir institución en mano de obra + tests | 0.5 día | Ninguna |
| 2 | Modal de línea: proveedor en ambos tipos + alta rápida | 1 día | Fase 1 |
| 3 | Detalle: menú de 3 puntos y quitar proveedor de cabecera | 0.5 día | Fase 2 |
| 4 | Formulario de reparación sin proveedor + i18n | 0.5 día | Fase 2 |
| 5 | Casas sin link «Ver» | 0.25 día | Ninguna |
| 6 | Guardrails, pruebas y release | 0.5 día | Fases 1-5 |

### 7.2 Milestones

1. **MVP**: proveedor por línea (ambos tipos), alta rápida, sin proveedor en reparación, menús de 3 puntos y card sin «Ver».
2. **V1.0**: misma alcance con i18n es/en y darkmode verificado.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Modal anidado con z-index incorrecto | Baja | Medio | Usar `z-[70]` y verificar con `Select` (`z-[90]`) abierto |
| Pérdida de datos de línea al crear institución | Baja | Alto | El alta es un modal secundario; el estado de la línea no se desmonta |
| Registros históricos con proveedor a nivel de reparación | Media | Bajo | Se conservan en DB; solo se ocultan en UI |
| Divergencia i18n es/en | Baja | Medio | `check:i18n` en el gate |
| Alcance se extiende a autos | Media | Bajo | P2 explícitamente fuera de alcance |

---

## 9. Notas y Referencias

- Reglas: `docs/rules/20-ui.md`, `30-ux.md`, `10-architecture.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Precedentes: SPEC-140 (`UI-SELECT-ADD`), SPEC-141 (módulo de reparaciones), SPEC-117/134 (`UX-DETAIL-ACTIONS`).
- Issue de GitHub: pendiente de creación.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | opencode | Creación inicial de la especificación |
| 2026-11-23 | opencode | Implementación completa; guardrails `go build`, `go test`, `check:ui`, `check:i18n`, `check:numbers` y build del frontend en verde. Estado pasa a `pending_release`. |
