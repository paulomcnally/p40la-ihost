---
title: "CRUD de Categorías en módulo Pensión Alimenticia"
id: "SPEC-048"
status: "in_progress"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-02"
github_issue: 48
---

# CRUD de Categorías en módulo Pensión Alimenticia

**ID**: SPEC-048  
**Estado**: in_progress  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-02

---

## 1. Resumen Ejecutivo

El módulo de **Pensión Alimenticia** (base estructural creada en SPEC-044) tiene la sección **Categorías** navegando actualmente a una página placeholder. Esta spec convierte esa sección en un **CRUD completo de categorías de gastos**: registrar, listar, editar y eliminar cada categoría que agrupa los conceptos de la pensión alimenticia (ej: Alimentación, Educación, Salud, Vivienda).

Cada categoría tendrá: **nombre** (requerido), **descripción** (opcional) y un **booleano "Generación automática"** con sub-label *"Si se activa, se generarán registros automáticamente cada mes"*. Este flag es el insumo para la futura generación automática de registros mensuales (sección "Registros mensuales" de SPEC-044); en esta spec solo se persiste y se muestra en la UI, sin implementar el scheduler de generación (depende de una spec futura del módulo de registros mensuales).

El CRUD replica el patrón ya establecido en el proyecto y en el mismo módulo (Hijos, Salarios): página de cards con `CreateMenu` (3 puntos en el header), menú de acciones de 3 puntos por card (Editar/Eliminar), modal de confirmación para eliminar y `EmptyCard` cuando no hay registros. Respeto de iHost: una tabla SQLite pequeña, sin dependencias nuevas. Para no colisionar con las categorías de instituciones existentes (tabla `institution_categories`, API `/api/institution-categories`), se usa una tabla dedicada `pension_categories` con API `/api/pension-categories`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Tabla SQLite `pension_categories` con campos: id, name (TEXT NOT NULL), description (TEXT NULL), auto_generate (INTEGER NOT NULL DEFAULT 0, booleano), created_at, updated_at.
2. **REQ-002**: API REST CRUD protegida por auth: GET /api/pension-categories, GET /api/pension-categories/:id, POST /api/pension-categories, PUT /api/pension-categories/:id, DELETE /api/pension-categories/:id.
3. **REQ-003**: La sección Categorías (`/pension/categorias`) deja de ser placeholder y muestra la página de listado de categorías con cards en grid (patrón `AutosPage`/`HijosPage`).
4. **REQ-004**: Formulario de creación/edición (`CategoryFormPage`) con campos: **Nombre** (requerido, texto), **Descripción** (opcional, textarea) y **Generación automática** (switch/toggle booleano, default false) con sub-label "Si se activa, se generarán registros automáticamente cada mes".
5. **REQ-005**: En cada card del listado se muestra el **nombre** como título, la **descripción** como subtítulo (si existe) y un indicador visual cuando **Generación automática** está activa.
6. **REQ-006**: Cada card tiene menú de acciones de 3 puntos (CardMenu) con opciones **Editar** y **Eliminar**.
7. **REQ-007**: Eliminar usa el modal de confirmación existente (DeleteModal).
8. **REQ-008**: `EmptyCard` cuando no hay categorías registradas, con botón para crear la primera.
9. **REQ-009**: Header de la página con título "Categorías" y `CreateMenu` (ícono 3 puntos) con opción de crear categoría.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: Validación de formulario: nombre no vacío. Toast de error si falla la validación.
2. **REQ-011**: El flag **auto_generate** (default false) se persiste y se muestra visualmente en la card.
3. **REQ-012**: Toast de éxito/error al crear, editar y eliminar.
4. **REQ-013**: Loading spinner mientras se cargan los datos.
5. **REQ-014**: Etiquetas traducibles vía i18n es/en en `frontend/public/i18n/` (fuente de verdad), incluyendo el namespace `categories` y actualizando `pension.categories_empty` si es necesario.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Ordenar las categorías por fecha de creación (más recientes primero) o alfabéticamente en el listado.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: CRUD ligero con queries simples sobre SQLite; sin joins ni computación pesada.
- **Seguridad**: Misma autenticación existente (`authMiddleware` en todas las rutas de la API).
- **Almacenamiento**: Una tabla SQLite nueva (`pension_categories`), registros de ~100-200 bytes cada uno.
- **Disponibilidad**: Sin cambios en health checks ni schedulers existentes. El flag `auto_generate` NO dispara generación en esta spec (se documenta como futuro).
- **iHost**: Sin dependencias nuevas; solo archivos estáticos React pre-build (Vite) servidos por el backend Go; sin Node.js en runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron los patrones CRUD existentes para replicarlos exactamente:

- **SPEC-045 (CRUD de Hijos)**: referencia de CRUD en el mismo módulo — migración SQLite, `storage` + `service` + `handlers` en Go, rutas en `internal/api/routes.go`, página de listado y formulario.
- **SPEC-047 (CRUD de Salarios)**: referencia de CRUD con flag booleano y switch en el formulario.
- **SPEC-044 (Menú Pensión Alimenticia)**: ya definió la sección **Categorías** navegando a `/pension/categorias` con página placeholder (`PensionPage.tsx`). Esta spec reemplaza el placeholder por el CRUD real.
- **SPEC-026 (Categorías de Instituciones)**: existe `institution_categories` con API `/api/institution-categories`; la nueva tabla y rutas usan prefijo `pension-` para evitar colisiones.
- **`internal/services/billing_scheduler.go`**: patrón de scheduler de generación automática de facturas (SPEC-008) que servirá de referencia futura para la generación automática de registros mensuales; NO se implementa en esta spec.
- **`internal/api/routes.go`**: patrón de registro de rutas con `authMiddleware`.
- **`migrations/`**: última migración es `0016` (children, SPEC-045). SPEC-046 y SPEC-047 son drafts y usarán `0017`/`0018`; esta spec usa la siguiente disponible (`0018` o `0019` según colisión).
- **i18n**: fuente de verdad en `frontend/public/i18n/{es,en}.json` (se sobrescriben en el build). Ya existen `pension.categories` y `pension.categories_empty`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| CRUD completo (DB + API + listado + formulario) | Funcionalidad completa, consistente con Hijos/Salarios | Más archivos a crear | ✅ Seleccionada |
| Solo frontend con datos en memoria/JSON | Rápido de implementar | No persiste, no escala | ❌ Rechazada |
| Reutilizar `institution_categories` | Evita tabla nueva | Dominios distintos (instituciones vs pensiones); mezcla conceptos | ❌ Rechazada |
| Mantener placeholder e implementar más adelante | Cero trabajo | No cumple el requerimiento | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Tabla dedicada `pension_categories` con API `/api/pension-categories`
- **Contexto**: Ya existe `institution_categories` (SPEC-026) para categorías de instituciones. Las categorías de pensión alimenticia son un dominio distinto.
- **Decisión**: Crear tabla `pension_categories` y endpoints con prefijo `pension-categories` para no colisionar con `/api/institution-categories`.
- **Consecuencias**: Cero mezcla de dominios; rutas y modelos claramente separados.

**ADR-002**: Solo se persiste `auto_generate` en esta spec; la generación automática es futura
- **Contexto**: El requerimiento pide el booleano "Generación automática" con sub-label que indica que se generarán registros mensuales automáticamente. La sección "Registros mensuales" aún es placeholder (SPEC-044).
- **Decisión**: Esta spec implementa el CRUD y persiste `auto_generate` (default 0). La lógica de generación mensual se implementará en una spec futura del módulo de registros mensuales, tomando como referencia `billing_scheduler.go`.
- **Consecuencias**: El flag queda listo y visible en la UI; no hay scheduler nuevo en esta spec, manteniendo el cambio acotado y de bajo riesgo en iHost.

**ADR-003**: `name` como único campo requerido
- **Contexto**: El requerimiento pide que solo el nombre sea obligatorio; la descripción es opcional.
- **Decisión**: Columna `name TEXT NOT NULL` y `description TEXT NULL`. Validación solo de nombre no vacío.
- **Consecuencias**: Formulario simple; descripción libremente omitible.

**ADR-004**: `auto_generate` como entero 0/1
- **Contexto**: El flag es un booleano de UI (switch/toggle), consistente con el patrón `active` de salarios (SPEC-047) y `auto_generate` de services (migración 0007).
- **Decisión**: Columna `auto_generate INTEGER NOT NULL DEFAULT 0` (0 = off, 1 = on); el frontend lo mapea a switch y lo serializa como booleano en JSON.
- **Consecuencias**: Persistencia simple y consistente con el resto del sistema.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[CategoryFormPage] --(API REST)--> [PensionCategoryHandlers] --> [PensionCategoryService] --> [PensionCategoryStorage]
      |                                                                                                     |
      v                                                                                                     v
[Frontend: CategoriasPage (cards)]                                                           [SQLite pension_categories]
```

### 4.2 Componentes

#### 4.2.1 Backend - PensionCategoryStorage
- **Responsabilidad**: Queries CRUD contra SQLite.
- **Interfaz**: List, GetByID, Create, Update, Delete.
- **Dependencias**: `database/sql`.
- **Ubicación**: `internal/storage/pension_category.go`.

#### 4.2.2 Backend - PensionCategoryService
- **Responsabilidad**: Validación de negocio (name no vacío; description opcional; auto_generate booleano con default false).
- **Interfaz**: Lista, obtiene, crea, actualiza, elimina.
- **Dependencias**: PensionCategoryStorage.
- **Ubicación**: `internal/services/pension_category.go`.

#### 4.2.3 Backend - PensionCategoryHandlers
- **Responsabilidad**: HTTP handlers para la API REST.
- **Interfaz**: HandleListCategories, HandleGetCategory, HandleCreateCategory, HandleUpdateCategory, HandleDeleteCategory.
- **Dependencias**: PensionCategoryService.
- **Ubicación**: `internal/api/pension_category_handlers.go`.

#### 4.2.4 Frontend - CategoriasPage
- **Responsabilidad**: Listado de categorías en cards con grid, menú de 3 puntos por card (Editar/Eliminar), `CreateMenu` en header, `EmptyCard` cuando no hay registros, indicador visual de auto_generate por card.
- **Interfaz**: Componente React funcional.
- **Dependencias**: api, Icon, CreateMenu, CardMenu, DeleteModal, EmptyCard, LoadingSpinner.
- **Ubicación**: `frontend/src/pages/CategoriasPage.tsx`.

#### 4.2.5 Frontend - CategoryFormPage
- **Responsabilidad**: Formulario de creación/edición con campos Nombre (input), Descripción (textarea) y Generación automática (switch) con sub-label.
- **Interfaz**: Componente React funcional, maneja `:id` param.
- **Dependencias**: api, useToast, useNavigate, Switch/Toggle.
- **Ubicación**: `frontend/src/pages/CategoryFormPage.tsx`.

#### 4.2.6 App / Sidebar
- **Responsabilidad**: Registrar rutas `/pension/categorias` → `CategoriasPage` y `/pension/categorias/new` + `/pension/categorias/edit/:id` → `CategoryFormPage`. El sidebar ya tiene la entrada "Categorías" (SPEC-044).
- **Ubicación**: `frontend/src/App.tsx`.

### 4.3 Modelo de datos

```
Entidad: pension_category
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- name: TEXT NOT NULL (nombre de la categoría, ej: "Alimentación")
- description: TEXT NULL (descripción opcional, ej: "Gastos de supermercado y comidas")
- auto_generate: INTEGER NOT NULL DEFAULT 0 (booleano 0/1; 1 = generación automática mensual)
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP

Relaciones: Ninguna por ahora (a futuro las usará la tabla de registros mensuales del módulo)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/pension-categories`

**Response 200**:
```json
[
  {
    "id": 1,
    "name": "Alimentación",
    "description": "Gastos de supermercado y comidas",
    "auto_generate": true,
    "created_at": "2026-09-02T10:00:00Z",
    "updated_at": "2026-09-02T10:00:00Z"
  }
]
```

#### Endpoint: `GET /api/pension-categories/:id`

**Response 200**: Objeto `pension_category` individual
**Response 404**: `{"error": "not_found", "message": "Categoría no encontrada"}`

#### Endpoint: `POST /api/pension-categories`

**Request**:
```json
{
  "name": "Alimentación",
  "description": "Gastos de supermercado y comidas",
  "auto_generate": true
}
```

**Response 201**: Objeto `pension_category` creado con ID (auto_generate default false si se omite)
**Response 400**: `{"error": "validation_error", "message": "El nombre es requerido"}`

#### Endpoint: `PUT /api/pension-categories/:id`

**Request**: Mismo schema que POST
**Response 200**: Objeto `pension_category` actualizado
**Response 400**: Mismos errores de validación que POST

#### Endpoint: `DELETE /api/pension-categories/:id`

**Response 200**: `{"message": "Categoría eliminada"}`

### 4.5 Dependencias

- **Internas**: `internal/api/routes.go` (registro de rutas), `cmd/server/main.go` (wiring del handler), `frontend/src/App.tsx` (rutas React), `frontend/src/pages/PensionPage.tsx` (se deja de usar para Categorías), componentes UI existentes (CreateMenu, CardMenu, DeleteModal, Toast, Icon, LoadingSpinner, EmptyCard, Switch), archivos i18n.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el sidebar, cuando hago clic en "Pensión alimenticia → Categorías", se navega a `/pension/categorias` y se muestra la página de listado de categorías.
- [ ] CA-002: Dado el listado vacío, cuando no hay categorías, se muestra el EmptyCard con título, descripción y botón para crear la primera.
- [ ] CA-003: Dado el listado con registros, se muestran cards en grid con el nombre como título, la descripción como subtítulo (si existe) y un indicador visual cuando Generación automática está activa.
- [ ] CA-004: Dado una card, cuando hago hover, aparece el menú de 3 puntos con opciones **Editar** y **Eliminar**.
- [ ] CA-005: Dado el formulario de creación, cuando completo el Nombre (Descripción opcional, Generación automática default off) y guardo, se crea la categoría y se redirige al listado.
- [ ] CA-006: Dado el formulario con el nombre vacío, al guardar se muestra toast de error y no se crea el registro.
- [ ] CA-007: Dado el formulario de edición, cuando modifico campos y/o el switch Generación automática y guardo, se actualiza la categoría.
- [ ] CA-008: Dado el formulario, el switch muestra el sub-label "Si se activa, se generarán registros automáticamente cada mes".
- [ ] CA-009: Dado una categoría con auto_generate=true, la card lo muestra visualmente como activa.
- [ ] CA-010: Dado el menú de acciones, cuando selecciono "Eliminar", aparece el modal de confirmación y al confirmar se elimina la categoría.
- [ ] CA-011: Las etiquetas se muestran traducidas en español e inglés según el idioma seleccionado.

### 5.2 No funcionales

- [ ] CA-NF-001: El build de Vite (`npm run build` en `frontend/`) compila sin errores.
- [ ] CA-NF-002: No se agregan dependencias nuevas al proyecto.
- [ ] CA-NF-003: El backend compila (`go build ./...`) y los tests de Go pasan (`go test ./...`).

### 5.3 Testing

- **Unit tests**: Validación del service (nombre vacío, auto_generate booleano default false).
- **Integration tests**: CRUD completo contra SQLite en memoria (patrón de `internal/storage/child.go` y sus tests).
- **E2E tests**: Flujo de usuario: crear categoría → listar con indicador auto_generate → editar (incluido toggle) → eliminar; casos de validación.
- **Carga/Performance**: Validar tiempo de respuesta del listado con 50+ registros (queries simples, sin joins).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración SQLite `0019_create_pension_categories` (up/down) + modelo Go | 15 min | Ninguna |
| 2 | PensionCategoryStorage + PensionCategoryService + PensionCategoryHandlers | 30 min | Fase 1 |
| 3 | Registro de rutas API en `routes.go` + wiring en `main.go` | 15 min | Fase 2 |
| 4 | Tipo `PensionCategory` en `frontend/src/types/index.ts` + métodos en api client | 15 min | Fase 3 |
| 5 | CategoriasPage (listado con cards, menú 3 puntos, indicador auto_generate, EmptyCard) | 30 min | Fase 4 |
| 6 | CategoryFormPage (formulario con validación, switch Generación automática con sub-label) | 30 min | Fase 4 |
| 7 | Rutas React en `App.tsx` (reemplazar placeholder de Categorías) | 15 min | Fase 5, 6 |
| 8 | Claves i18n es/en (`frontend/public/i18n/` + `frontend/src/i18n/`) y build | 15 min | Fase 7 |
| 9 | Pruebas locales (`go test`, `npm run build`, levantar server, validación manual) | 30 min | Fase 8 |
| **Total** | | **~3.25 horas** | |

### 6.2 Milestones

1. **MVP**: CRUD funcional completo de categorías con cards e indicador auto_generate (Fases 1-8).
2. **V1.0**: MVP + pruebas locales + validación manual del usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Colisión de número de migración con SPEC-046/047 (drafts en paralelo) | Media | Medio | Coordinar el número: usar la siguiente migración libre (`0018` o `0019` según lo que ocupen 046/047) |
| Colisión de rutas con `institution-categories` existentes | Baja | Alto | Usar prefijo `pension-categories` en tablas y endpoints (ADR-001) |
| El switch Generación automática sugiere que ya genera registros y aún no existe el scheduler | Media | Medio | Sub-label claro + esta spec solo persiste el flag; el scheduler se documenta como futura spec de Registros mensuales |
| Claves i18n editadas solo en `frontend/src/i18n/` se pierden en el build | Media | Medio | Editar SIEMPRE `frontend/public/i18n/` (fuente de verdad) y correr `npm run build`; verificar con `curl` (ver AGENTS.md) |
| Reemplazar el placeholder de SPEC-044 rompe la ruta `/pension/categorias` | Baja | Medio | Mantener la misma ruta; verificar navegación del resto de secciones de Pensión |

## 8. Notas y Referencias

- Patrón CRUD de referencia: `docs/specs/SPEC-045-crud-hijos-pension.md`, `docs/specs/SPEC-047-crud-salarios-pension.md`, `docs/specs/SPEC-024-autos-crud.md`, `frontend/src/pages/HijosPage.tsx`, `frontend/src/pages/AutosPage.tsx`.
- Backend CRUD: `internal/storage/child.go`, `internal/services/child.go`, `internal/api/child_handlers.go`.
- Categorías de instituciones (referencia a evitar): `docs/specs/SPEC-026-institution-categories.md`, `internal/api/institution_category_handlers.go`, migración `0011`.
- Scheduler de generación automática (referencia futura): `internal/services/billing_scheduler.go` (SPEC-008).
- Base del módulo Pensión Alimenticia: `docs/specs/SPEC-044-pension-alimenticia-sidebar.md` (sección Categorías en `/pension/categorias`).
- Componentes UI: `frontend/src/components/CreateMenu.tsx`, `CardMenu.tsx`, `DeleteModal.tsx`, `EmptyCard` (inline en `HijosPage.tsx`), Switch/Toggle (usado en `SalaryFormPage.tsx` si aplica).
- Migraciones: `migrations/` (última: `0016`; la nueva será `0018` o `0019` según colisión con SPEC-046/047).
- Reglas de i18n: fuente de verdad en `frontend/public/i18n/` (ver AGENTS.md, sección "Reglas críticas").

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-02 | p40la-ihost-team | Estado cambiado de draft a in_progress. Implementación completa: migración 0019, backend CRUD /api/pension-categories, CategoriasPage con indicador auto_generate, CategoryFormPage con switch y sub-label, i18n es/en, tests. Validado en local (go test + npm run build + CRUD API). Pendiente validación manual del usuario |