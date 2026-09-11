---
title: "Editar seguros de auto (vencidos o no) con menú de 3 puntos en la vista del auto"
id: "SPEC-067"
status: "released"
author: "paulomcnally"
created: "2026-09-11"
updated: "2026-09-11"
github_issue: 69
---

# Editar seguros de auto (vencidos o no) con menú de 3 puntos en la vista del auto

**ID**: SPEC-067  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-11  
**Actualizado**: 2026-09-11

---

## 1. Resumen Ejecutivo

Actualmente los seguros de auto (pólizas asociadas a un auto) solo se pueden **crear y eliminar**. La vista del auto (`AutoShowPage`) muestra cada seguro con un botón directo de ícono trash para eliminarlo, pero **no existe ninguna forma de editar** un seguro, ya sea vencido o no. Esto obliga al usuario a eliminar y volver a crear la póliza ante cualquier error de carga (número de póliza, tipo de cobertura, certificado, aseguradora), perdiendo datos y agregando fricción.

Esta spec agrega la funcionalidad de **edición de seguros de auto** (vigentes o vencidos) y cambia la UI del listado de seguros en la vista del auto: se reemplaza el ícono trash directo por el menú estándar de **3 puntos** (`CardMenu`) que despliega un dropdown con las opciones **"Editar"** y **"Eliminar"**, siguiendo el patrón ya usado en `AutosPage`, `ServicesPage`, `DeudasPage`, etc. El requerimiento del usuario es explícito: *no pedir otro ícono*, sino el menú de 3 puntos que ya es el estándar del proyecto.

Consideraciones iHost: el cambio no agrega dependencias nuevas ni procesos; solo un endpoint PUT nuevo (SQL UPDATE simple sobre `auto_services`) y un modal de edición reutilizando el componente existente `AddInsuranceModal` (convertido a create/edit). Impacto en memoria y SQLite mínimo. Para el frontend, `app.edit`/`app.delete` ya existen en i18n y se reutilizan.

Consideraciones de UI obligatorias: el modal de edición es un formulario (modal) que DEBE usar los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y verificarse en darkmode. No aplica `BACK_ROUTES`/flecha atrás porque no se crea una página de detalle nueva ni un formulario en página propia: la edición se hace en modal desde la vista existente del auto (que ya tiene su flecha atrás registrada).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Poder **editar** un seguro de auto desde la vista del auto, tanto si está vencido como vigente.
2. **REQ-002**: En el listado de seguros de la vista del auto, reemplazar el botón directo de ícono trash por el **menú de 3 puntos** (`CardMenu`) que despliega un dropdown con las opciones **"Editar"** y **"Eliminar"**.
3. **REQ-003**: La opción **"Eliminar"** del dropdown debe conservar el flujo actual de eliminación (confirmación con `DeleteModal` + `DELETE /api/autos/{id}/services/{service_id}`).
4. **REQ-004**: La opción **"Editar"** debe abrir el modal de edición con los datos actuales del seguro precargados y permitir modificar los campos editables: `coverage_type`, `policy_number`, `certificate`, `insurer_number`.
5. **REQ-005**: Agregar endpoint backend de actualización de seguro de auto (PUT/PATCH) y el método `Update` en las capas de service y storage, con las mismas validaciones que el create (`coverage_type`, `policy_number`, `insurer_number` requeridos y trim).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Reutilizar el modal existente `AddInsuranceModal.tsx` convirtiéndolo en un modal compartido **crear/editar** (o crear un modal de edición equivalente) para evitar duplicar código de formulario.
2. **REQ-007**: Exponer en `frontend/src/api/index.ts` el método `updateService` para seguros de auto.
3. **REQ-008**: La edición debe ser visible tanto para seguros vigentes como vencidos (no filtrar por estado).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Toast de éxito/error al editar un seguro (siguiendo el patrón de las demás operaciones del frontend).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Una consulta SQL UPDATE por operación; sin impacto perceptible en iHost.
- **Seguridad**: Endpoint protegido con `authMiddleware` (igual que los existentes de auto_services). Validación de pertenencia del seguro al auto (`auto_id` y `service_id`).
- **Almacenamiento**: Sin cambios de esquema; la tabla `auto_services` ya tiene todos los campos editables.
- **Disponibilidad**: `GET /health` inalterado; sin servicios nuevos.
- **iHost**: Sin dependencias nuevas; memoria/CPU despreciables (operación puntual).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se relevó el código actual en el worktree (`feature/SPEC-067`):

- **Frontend — vista del auto**: `frontend/src/pages/AutoShowPage.tsx`. El listado de seguros (líneas 98-167) renderiza cada seguro como fila con un botón trash directo (líneas 155-160) que setea `deleteTarget` y confirma con `DeleteModal` (líneas 178-185) llamando `api.autos.removeService(id, serviceId)`.
- **Frontend — patrón de menú**: existe el componente `frontend/src/components/CardMenu.tsx` (dropdown de 3 puntos, portal a `document.body`, opción `danger`). Se usa en 10+ páginas con el patrón exacto `{ label: t('app.edit'), icon: 'edit', onClick }` + `{ label: t('app.delete'), icon: 'delete', danger: true, onClick }` (ej: `AutosPage.tsx:71-76`, `ServicesPage.tsx:123-126`, `DeudasPage.tsx:165-168`).
- **Frontend — formulario de seguros**: `frontend/src/components/AddInsuranceModal.tsx` (177 líneas) solo crea seguros (carga `availableServices`, campos `coverage_type`, `policy_number`, `certificate`, `insurer_number`, submit `api.autos.addService`). No es reutilizable para editar porque no recibe un id/seguro existente.
- **Frontend — api**: `frontend/src/api/index.ts:157-167` — objeto `autos` solo expone `listServices`, `addService`, `removeService`, `availableServices`. **No existe `updateService`**.
- **Backend — handlers**: `internal/api/auto_service_handlers.go` — `ListAutoServices` (GET), `CreateAutoService` (POST), `DeleteAutoService` (DELETE), `ListAvailableServices` (GET). **No existe handler de update**.
- **Backend — rutas**: `internal/api/routes.go:105-109` — registra GET/POST/DELETE/GET de auto_services, todos con `authMiddleware`. **No hay PUT/PATCH**.
- **Backend — service**: `internal/services/auto_service.go` — `ListByAuto`, `Create` (valida `coverage_type`, `policy_number`, `insurer_number` requeridos + trim), `Delete`, `ListAvailableServices`. **No hay `Update`**.
- **Backend — storage**: `internal/storage/auto_service.go` — `ListByAuto` (JOIN auto_services+services+institutions), `Create`, `Delete` (por `auto_id` + `service_id`), `ListAvailableServices`. **No hay `Update`**.
- **Modelo de datos**: tabla `auto_services` (migración 0010) con `id, auto_id, service_id, coverage_type, policy_number, certificate, insurer_number, created_at` y `UNIQUE(auto_id, service_id)` (campos de póliza agregados en migración 0012). La clave natural de operación es `(auto_id, service_id)` — el DELETE actual ya usa esa clave.
- **i18n**: fuente de verdad `frontend/public/i18n/{es,en}.json`. Claves `app.edit` ("Editar"/"Edit") y `app.delete` ("Eliminar"/"Delete") ya existen. `AutoShowPage` y `AddInsuranceModal` usan strings hardcodeados en español (no i18n); no se introduce traducción obligatoria en esta spec, pero las acciones del dropdown reutilizan `app.edit`/`app.delete`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Endpoint `PUT /api/autos/{id}/services/{service_id}` + `CardMenu` | Consistente con el DELETE existente (misma clave), patrón de UI estándar del proyecto | Ninguna relevante | ✅ Seleccionada |
| Página dedicada de edición (`/autos/:id/insurances/:serviceId/edit`) | Más espacio para el formulario | Requiere nueva ruta, `BACK_ROUTES`, más archivos y fricción; el formulario actual es corto (4 campos) | ❌ Rechazada |
| Modal de edición separado (`EditInsuranceModal`) | Aislamiento | Duplica el formulario de `AddInsuranceModal`; no se reutiliza código | ❌ Rechazada (se prefiere modal compartido) |
| Endpoint `PATCH` parcial | Menos payload | Mayor complejidad de validación; los 4 campos editables se envían completos igual | ❌ Rechazada (PUT completo es suficiente) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Actualización por clave natural `(auto_id, service_id)` con `PUT /api/autos/{id}/services/{service_id}`
- **Contexto**: El DELETE existente identifica el seguro por `(auto_id, service_id)` (la tabla tiene `UNIQUE(auto_id, service_id)`). El `id` de `auto_services` existe pero no se expone como clave de operación en la API actual.
- **Decisión**: El endpoint de actualización usa la misma ruta/forma que el DELETE: `PUT /api/autos/{id}/services/{service_id}`, donde `service_id` es el id del servicio base (`services.id`).
- **Consecuencias**: Consistencia total con el patrón existente; `Update` en storage usa `UPDATE auto_services SET ... WHERE auto_id = ? AND service_id = ?`. El `service_id` no es editable (identifica al servicio base de la póliza); solo se editan `coverage_type`, `policy_number`, `certificate`, `insurer_number`.

**ADR-002**: Reutilizar `AddInsuranceModal` como modal compartido crear/editar
- **Contexto**: El formulario de seguros tiene solo 4 campos y `AddInsuranceModal` ya contiene la UI completa de creación.
- **Decisión**: Extender `AddInsuranceModal` para aceptar un prop opcional `insurance` (o `initialData`) y `onSubmit` que detecte si es edición o creación; en modo edición no recarga `availableServices` (el servicio ya está asociado) y precarga los valores.
- **Consecuencias**: Un solo componente de formulario (menos código, darkmode verificado una vez). El título del modal cambia entre "Agregar Seguro" y "Editar Seguro" según el modo.

**ADR-003**: Reemplazar el ícono trash por `CardMenu` en `AutoShowPage`
- **Contexto**: El requerimiento pide el menú de 3 puntos en vez de otro ícono. `CardMenu` es el estándar del proyecto.
- **Decisión**: En cada fila de seguro, reemplazar el `<button>` trash por `<CardMenu options={[{label: Editar}, {label: Eliminar, danger}]}>`. La fila/card debe tener `relative` para el posicionamiento absoluto del menú.
- **Consecuencias**: UI consistente con el resto del proyecto; la eliminación conserva su flujo (`DeleteModal`).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[AutoShowPage (lista seguros)]
      |  CardMenu (3 puntos): Editar / Eliminar
      ├─ Editar → AddInsuranceModal (modo edición)
      │     └─ PUT /api/autos/{id}/services/{service_id} → handler → service.Update → storage.Update (SQLite)
      └─ Eliminar → DeleteModal → DELETE /api/autos/{id}/services/{service_id} (existente)
```

### 4.2 Componentes

#### 4.2.1 Backend: handler `UpdateAutoService`
- **Responsabilidad**: Parsear body (`coverage_type`, `policy_number`, `certificate`, `insurer_number`), validar, llamar a service.Update y devolver el detalle actualizado.
- **Interfaz**: `PUT /api/autos/{id}/services/{service_id}` (protegido con `authMiddleware`).
- **Dependencias**: `internal/services/auto_service.go`.
- **Ubicación**: `internal/api/auto_service_handlers.go`.

#### 4.2.2 Backend: service `Update`
- **Responsabilidad**: Validación de negocio (mismas reglas que `Create`: `coverage_type` válido, `policy_number` e `insurer_number` requeridos tras trim; `certificate` opcional) y delegar a storage.
- **Dependencias**: `internal/storage/auto_service.go`.
- **Ubicación**: `internal/services/auto_service.go`.

#### 4.2.3 Backend: storage `Update`
- **Responsabilidad**: `UPDATE auto_services SET coverage_type=?, policy_number=?, certificate=?, insurer_number=? WHERE auto_id = ? AND service_id = ?`. Verificar que el seguro exista (rows affected) y devolver el registro actualizado (o reutilizar `ListByAuto`/consulta por id).
- **Ubicación**: `internal/storage/auto_service.go`.

#### 4.2.4 Frontend: `AddInsuranceModal` (modo edición)
- **Responsabilidad**: Si recibe prop de seguro existente, precarga los 4 campos, no carga `availableServices`, y al submit llama `api.autos.updateService(autoId, serviceId, body)` en vez de `addService`.
- **Interfaz**: Props adicionales opcionales (seguro existente + callback de éxito que refresca la lista).
- **Ubicación**: `frontend/src/components/AddInsuranceModal.tsx`.

#### 4.2.5 Frontend: `AutoShowPage` (CardMenu por seguro)
- **Responsabilidad**: Reemplazar el botón trash por `<CardMenu>` con opciones Editar/Eliminar. Mantener `DeleteModal` para eliminar y agregar estado para el seguro en edición.
- **Ubicación**: `frontend/src/pages/AutoShowPage.tsx`.

#### 4.2.6 Frontend: api client
- **Responsabilidad**: Agregar `updateService(id, serviceId, body)` → `put(`/api/autos/${id}/services/${serviceId}`, body)`.
- **Ubicación**: `frontend/src/api/index.ts`.

### 4.3 Modelo de datos

```
Entidad: auto_services (sin cambios de esquema)
- id: INTEGER PK AUTOINCREMENT
- auto_id: INTEGER NOT NULL (FK autos.id)
- service_id: INTEGER NOT NULL (FK services.id)
- coverage_type: TEXT NOT NULL ('daños_a_terceros' | 'full_cover')
- policy_number: TEXT NOT NULL DEFAULT ''
- certificate: TEXT NULL
- insurer_number: TEXT NOT NULL DEFAULT ''
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- UNIQUE(auto_id, service_id)
```

### 4.4 APIs / Contratos

#### Endpoint: `PUT /api/autos/{id}/services/{service_id}`

**Request**:
```json
{
  "coverage_type": "full_cover",
  "policy_number": "POL-2026-0001",
  "certificate": "CERT-001",
  "insurer_number": "INS-42"
}
```

**Response 200**:
```json
{
  "id": 3,
  "auto_id": 1,
  "service_id": 7,
  "coverage_type": "full_cover",
  "policy_number": "POL-2026-0001",
  "certificate": "CERT-001",
  "insurer_number": "INS-42",
  "service_name": "Seguro Todo Riesgo",
  "institution_name": "ASSA",
  "suggested_amount": 120.0,
  "frequency": "annual",
  "icon_key": "insurance",
  "active": true,
  "start_date": "2026-01-01",
  "end_date": "2027-01-01",
  "is_recurring": true,
  "created_at": "2026-01-01T00:00:00Z"
}
```

**Response Error**:
```json
{
  "error": "validation",
  "message": "policy_number is required"
}
```
Errores: 400 validación, 404 si el seguro no existe o no pertenece al auto, 401 sin sesión.

### 4.5 Dependencias

- **Internas**: `internal/api/auto_service_handlers.go`, `internal/api/routes.go`, `internal/services/auto_service.go`, `internal/storage/auto_service.go`, `frontend/src/api/index.ts`, `frontend/src/components/AddInsuranceModal.tsx`, `frontend/src/pages/AutoShowPage.tsx`.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: En la vista del auto, cada seguro (vigente o vencido) muestra el menú de 3 puntos en lugar del ícono trash.
- [x] CA-002: El dropdown del menú muestra las opciones "Editar" y "Eliminar" (Eliminar marcada como `danger`).
- [x] CA-003: "Eliminar" abre el `DeleteModal` de confirmación y al confirmar elimina el seguro (comportamiento actual intacto).
- [x] CA-004: "Editar" abre el modal con los datos del seguro precargados (`coverage_type`, `policy_number`, `certificate`, `insurer_number`).
- [x] CA-005: Al guardar la edición, se ejecuta `PUT /api/autos/{id}/services/{service_id}`, los cambios persisten en SQLite y la lista de seguros se refresca.
- [x] CA-006: Se puede editar un seguro **vencido** (no hay filtro por estado de vigencia).
- [x] CA-007: El endpoint PUT valida igual que el create: `coverage_type` válido, `policy_number` y `insurer_number` requeridos (trim), y devuelve 404 si el seguro no existe o no pertenece al auto.
- [x] CA-DARK: El modal de edición usa tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (texto y placeholders).
- [x] CA-BACK: No aplica (no se crea página de detalle/formulario en página propia; la edición es en modal dentro de la vista existente del auto, que ya tiene su flecha atrás registrada en `BACK_ROUTES`).

### 5.2 No funcionales

- [x] CA-NF-001: Sin dependencias nuevas en Go ni npm; build y tests existentes pasan.
- [x] CA-NF-002: El endpoint PUT está protegido con `authMiddleware` (401 sin sesión).

### 5.3 Testing

- **Unit tests**: validaciones de `Update` en service (campos requeridos, trim, coverage_type inválido) y storage (UPDATE sobre clave `(auto_id, service_id)`; 0 filas afectadas cuando no existe).
- **Integration tests**: PUT actualiza el registro y GET devuelve los valores nuevos; PUT con seguro inexistente → 404.
- **E2E tests**: flujo manual — vista del auto → menú de puntos de un seguro → Editar → modificar póliza → guardar → verificar cambio en la lista; repetir con un seguro vencido.
- **Carga/Performance**: operación puntual sobre SQLite; sin mediciones especiales.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: `Update` en storage + service + handler + ruta `PUT` | 0.5 día | Ninguna |
| 2 | Frontend: `updateService` en api client | 0.1 día | Fase 1 |
| 3 | Frontend: `AddInsuranceModal` modo edición (create/edit) | 0.5 día | Fase 2 |
| 4 | Frontend: `AutoShowPage` con `CardMenu` (Editar/Eliminar) reemplazando el trash | 0.5 día | Fase 3 |
| 5 | Tests, build, verificación en local y pruebas manuales con el usuario | 0.5 día | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: Backend PUT funcional + modal de edición + menú de 3 puntos en la vista del auto.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Romper la eliminación existente al reemplazar el trash | Media | Medio | Conservar `DeleteModal` y el mismo flujo de `deleteTarget`; solo cambia el disparador (CardMenu). |
| Modal de edición sin verificar en darkmode | Media | Bajo | Aplicar regla de tokens de tema (SPEC-060) y verificar darkmode antes de `pending_release`. |
| Confundir `auto_services.id` con `services.id` en la actualización | Media | Alto | Mantener la clave `(auto_id, service_id)` consistente con el DELETE; documentar en la spec. |
| Perder `availableServices` cargado en modo edición | Baja | Bajo | En modo edición no cargar `availableServices`; solo precargar el seguro actual. |

## 8. Notas y Referencias

- Componente de menú estándar: `frontend/src/components/CardMenu.tsx`.
- Modal de creación existente a reutilizar: `frontend/src/components/AddInsuranceModal.tsx`.
- Vista del auto con listado de seguros: `frontend/src/pages/AutoShowPage.tsx`.
- Handlers backend: `internal/api/auto_service_handlers.go`; rutas: `internal/api/routes.go`.
- Migraciones de `auto_services`: `migrations/0010_*` y `migrations/0012_*`.
- Reglas de UI: tokens de tema en inputs (SPEC-060) y flecha atrás del header (SPEC-063).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-11 | paulomcnally | Creación inicial de la especificación |
| 2026-09-11 | paulomcnally | Cambio de estado a `pending_execution` (issue #69) |
| 2026-09-11 | paulomcnally | Cambio de estado a `in_progress` — inicio del desarrollo (issue #69) |
| 2026-09-11 | paulomcnally | Implementación completa: backend PUT, modal edición, CardMenu, moneda/formato. Cierre a `released` (issue #69) |