---
title: "CRUD de Hijos en módulo Pensión Alimenticia"
id: "SPEC-045"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-02"
github_issue: 45
---

# CRUD de Hijos en módulo Pensión Alimenticia

**ID**: SPEC-045  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-02

---

## 1. Resumen Ejecutivo

El módulo de **Pensión Alimenticia** (base estructural creada en SPEC-044) tiene la sección **Hijos** navegando actualmente a una página placeholder. Esta spec convierte esa sección en un **CRUD completo de hijos**: registrar, listar, editar y eliminar los datos personales de cada hijo involucrado en la pensión.

Cada hijo tendrá: **nombres** (requerido), **apellidos** (requerido), **fecha de nacimiento** (requerido) y **notas** (opcional). En la página de listado, las cards muestran además la **edad calculada en base a la fecha actual** a partir de la fecha de nacimiento.

El CRUD replica el patrón ya establecido en el proyecto (Homes, Autos, Services): página de cards con `CreateMenu` (3 puntos en el header), menú de acciones de 3 puntos por card (Editar/Eliminar), modal de confirmación para eliminar y `EmptyCard` cuando no hay registros. Respeto de iHost: una tabla SQLite pequeña, sin dependencias nuevas, cálculo de edad en frontend para no consumir CPU del servidor.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Tabla SQLite `children` con campos: id, first_name, last_name, birth_date, notes (nullable), created_at, updated_at.
2. **REQ-002**: API REST CRUD protegida por auth: GET /api/children, GET /api/children/:id, POST /api/children, PUT /api/children/:id, DELETE /api/children/:id.
3. **REQ-003**: La sección Hijos (`/pension/hijos`) deja de ser placeholder y muestra la página de listado de hijos con cards en grid (patrón `AutosPage`).
4. **REQ-004**: Formulario de creación/edición (`HijoFormPage`) con campos: Nombres (requerido), Apellidos (requerido), Fecha de nacimiento (requerido, input date), Notas (opcional, textarea).
5. **REQ-005**: En cada card del listado se muestra la **edad calculada** a partir de la fecha de nacimiento y la fecha actual (con pluralización correcta: "1 año" / "X años").
6. **REQ-006**: Cada card tiene menú de acciones de 3 puntos (CardMenu) con opciones **Editar** y **Eliminar**.
7. **REQ-007**: Eliminar usa el modal de confirmación existente (DeleteModal).
8. **REQ-008**: `EmptyCard` cuando no hay hijos registrados, con botón para crear el primero.
9. **REQ-009**: Header de la página con título "Hijos" y `CreateMenu` (ícono 3 puntos) con opción de crear hijo.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: Validación de formulario: nombres y apellidos no vacíos, fecha de nacimiento requerida y no futura. Toast de error si falla la validación.
2. **REQ-011**: Toast de éxito/error al crear, editar y eliminar.
3. **REQ-012**: Loading spinner mientras se cargan los datos.
4. **REQ-013**: Etiquetas traducibles vía i18n es/en en `frontend/public/i18n/` (fuente de verdad) y `frontend/src/i18n/`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Ordenar los hijos por edad (de mayor a menor) en el listado.
2. **REQ-015**: Mostrar la fecha de nacimiento formateada junto a la edad en la card.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: CRUD ligero con queries simples sobre SQLite; la edad se calcula en el frontend (sin carga extra de CPU en iHost).
- **Seguridad**: Misma autenticación existente (`authMiddleware` en todas las rutas de la API).
- **Almacenamiento**: Una tabla SQLite nueva (`children`), registros de ~100-200 bytes cada uno.
- **Disponibilidad**: Sin cambios en health checks ni schedulers existentes.
- **iHost**: Sin dependencias nuevas; solo archivos estáticos React pre-build (Vite) servidos por el backend Go; sin Node.js en runtime.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron los patrones CRUD existentes para replicarlos exactamente:

- **SPEC-024 (CRUD de Autos)**: referencia completa de CRUD — migración SQLite, `storage` + `service` + `handlers` en Go, rutas en `internal/api/routes.go`, página de listado (`AutosPage.tsx`) y formulario (`AutoFormPage.tsx`).
- **SPEC-044 (Menú Pensión Alimenticia)**: ya definió la sección **Hijos** navegando a `/pension/hijos` con página placeholder (`PensionPage.tsx`). Esta spec reemplaza el placeholder de Hijos por el CRUD real.
- **`frontend/src/pages/AutosPage.tsx`**: patrón de listado con cards en grid, `CreateMenu` (3 puntos) en el header, `CardMenu` (3 puntos por card con Editar/Eliminar), `DeleteModal` y `EmptyCard`.
- **`internal/api/routes.go`**: patrón de registro de rutas con `authMiddleware`.
- **`migrations/`**: última migración es `0015`; la nueva será `0016`.
- **i18n**: fuente de verdad en `frontend/public/i18n/{es,en}.json` (se sobrescriben en el build), espejo en `frontend/src/i18n/{es,en}.json`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| CRUD completo (DB + API + listado + formulario) | Funcionalidad completa, consistente con Autos/Homes | Más archivos a crear | ✅ Seleccionada |
| Solo frontend con datos en memoria/JSON | Rápido de implementar | No persiste, no escala | ❌ Rechazada |
| Mantener placeholder e implementar más adelante | Cero trabajo | No cumple el requerimiento | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Cálculo de edad en el frontend
- **Contexto**: La card debe mostrar la edad del hijo calculada con la fecha actual. Se evalúa dónde computarla.
- **Decisión**: Calcular la edad en el frontend con una función pura (`calcularEdad(birthDate)`) en el momento de renderizar las cards, usando `new Date()` como fecha de referencia.
- **Consecuencias**: Cero costo de CPU en el iHost, cero campos extra en la API; la edad siempre es "al día de hoy" sin necesidad de regenerar datos.

**ADR-002**: Nombres de columnas en inglés
- **Contexto**: Los labels de UI serán "Nombres", "Apellidos", "Fecha de nacimiento", "Notas" (es) y equivalents en inglés.
- **Decisión**: Columnas `first_name`, `last_name`, `birth_date`, `notes` en la tabla `children`, siguiendo la convención de código en inglés del proyecto (ver `autos`: year/model/brand).
- **Consecuencias**: Consistencia con el resto del código; los labels traducibles se manejan solo en i18n.

**ADR-003**: Tabla dedicada `children` sin FK a usuarios
- **Contexto**: Los hijos pertenecen al contexto global de la pensión alimenticia del usuario.
- **Decisión**: Tabla independiente sin relaciones por ahora; si a futuro se agregan registros mensuales o categorías, se añadirán FKs.
- **Consecuencias**: Simplicidad máxima; evita joins innecesarios; fácil de extender.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[HijoFormPage] --(API REST)--> [ChildHandlers] --> [ChildService] --> [ChildStorage]
      |                                                                       |
      v                                                                       v
[Frontend: HijosPage (cards + edad)]                            [SQLite children]
```

### 4.2 Componentes

#### 4.2.1 Backend - ChildStorage
- **Responsabilidad**: Queries CRUD contra SQLite.
- **Interfaz**: List, GetByID, Create, Update, Delete.
- **Dependencias**: `database/sql`.
- **Ubicación**: `internal/storage/child.go`.

#### 4.2.2 Backend - ChildService
- **Responsabilidad**: Validación de negocio (first_name, last_name y birth_date requeridos; birth_date no futura).
- **Interfaz**: Lista, obtiene, crea, actualiza, elimina.
- **Dependencias**: ChildStorage.
- **Ubicación**: `internal/services/child.go`.

#### 4.2.3 Backend - ChildHandlers
- **Responsabilidad**: HTTP handlers para la API REST.
- **Interfaz**: HandleListChildren, HandleGetChild, HandleCreateChild, HandleUpdateChild, HandleDeleteChild.
- **Dependencias**: ChildService.
- **Ubicación**: `internal/api/child_handlers.go`.

#### 4.2.4 Frontend - HijosPage
- **Responsabilidad**: Listado de hijos en cards con grid, menú de 3 puntos por card (Editar/Eliminar), `CreateMenu` en header, `EmptyCard` cuando no hay registros, cálculo de edad por card.
- **Interfaz**: Componente React funcional.
- **Dependencias**: api, Icon, CreateMenu, CardMenu, DeleteModal, EmptyCard, LoadingSpinner, `calcularEdad`.
- **Ubicación**: `frontend/src/pages/HijosPage.tsx`.

#### 4.2.5 Frontend - HijoFormPage
- **Responsabilidad**: Formulario de creación/edición con campos Nombres, Apellidos, Fecha de nacimiento (input date), Notas (textarea).
- **Interfaz**: Componente React funcional, maneja `:id` param.
- **Dependencias**: api, useToast, useNavigate.
- **Ubicación**: `frontend/src/pages/HijoFormPage.tsx`.

#### 4.2.6 Frontend - util de edad
- **Responsabilidad**: Función pura `calcularEdad(birthDate: string): number` que devuelve años cumplidos según `new Date()`.
- **Ubicación**: `frontend/src/utils/age.ts` (nuevo directorio `utils/` si no existe).

#### 4.2.7 App / Sidebar
- **Responsabilidad**: Registrar rutas `/pension/hijos` → `HijosPage` y `/pension/hijos/new` + `/pension/hijos/edit/:id` → `HijoFormPage`. El sidebar ya tiene la entrada "Hijos" (SPEC-044).
- **Ubicación**: `frontend/src/App.tsx`.

### 4.3 Modelo de datos

```
Entidad: child
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- first_name: TEXT NOT NULL (nombres, ej: "María")
- last_name: TEXT NOT NULL (apellidos, ej: "Pérez")
- birth_date: TEXT NOT NULL (fecha de nacimiento en formato ISO 'YYYY-MM-DD')
- notes: TEXT NULL (notas opcionales)
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP

Relaciones: Ninguna por ahora (entidad independiente dentro del módulo Pensión Alimenticia)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/children`

**Response 200**:
```json
[
  {
    "id": 1,
    "first_name": "María",
    "last_name": "Pérez",
    "birth_date": "2019-05-12",
    "notes": "Alergia al maní",
    "created_at": "2026-09-02T10:00:00Z",
    "updated_at": "2026-09-02T10:00:00Z"
  }
]
```

#### Endpoint: `GET /api/children/:id`

**Response 200**: Objeto `child` individual
**Response 404**: `{"error": "not_found", "message": "Hijo no encontrado"}`

#### Endpoint: `POST /api/children`

**Request**:
```json
{
  "first_name": "María",
  "last_name": "Pérez",
  "birth_date": "2019-05-12",
  "notes": "Alergia al maní"
}
```

**Response 201**: Objeto `child` creado con ID
**Response 400**: `{"error": "validation_error", "message": "Nombres, apellidos y fecha de nacimiento son requeridos"}`

#### Endpoint: `PUT /api/children/:id`

**Request**: Mismo schema que POST
**Response 200**: Objeto `child` actualizado

#### Endpoint: `DELETE /api/children/:id`

**Response 200**: `{"message": "Hijo eliminado"}`

### 4.5 Dependencias

- **Internas**: `internal/api/routes.go` (registro de rutas), `cmd/server/main.go` (wiring del handler), `frontend/src/App.tsx` (rutas React), `frontend/src/pages/PensionPage.tsx` (se deja de usar para Hijos), componentes UI existentes (CreateMenu, CardMenu, DeleteModal, Toast, Icon, LoadingSpinner, EmptyCard), archivos i18n.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el sidebar, cuando hago clic en "Pensión alimenticia → Hijos", se navega a `/pension/hijos` y se muestra la página de listado de hijos.
- [ ] CA-002: Dado el listado vacío, cuando no hay hijos, se muestra el EmptyCard con título, descripción y botón para crear el primer hijo.
- [ ] CA-003: Dado el listado con registros, se muestran cards en grid con nombres, apellidos, fecha de nacimiento y edad calculada según la fecha actual.
- [ ] CA-004: Dado una card de hijo, cuando hago hover, aparece el menú de 3 puntos con opciones **Editar** y **Eliminar**.
- [ ] CA-005: Dado el formulario de creación, cuando completo Nombres, Apellidos y Fecha de nacimiento (Notas opcional) y guardo, se crea el hijo y se redirige al listado.
- [ ] CA-006: Dado el formulario con campos obligatorios vacíos, al guardar se muestra toast de error y no se crea el registro.
- [ ] CA-007: Dado el formulario con fecha de nacimiento futura, al guardar se muestra toast de error.
- [ ] CA-008: Dado el formulario de edición, cuando modifico campos y guardo, se actualiza el hijo.
- [ ] CA-009: Dado el menú de acciones, cuando selecciono "Eliminar", aparece el modal de confirmación y al confirmar se elimina el hijo.
- [ ] CA-010: Dado un hijo con fecha de nacimiento hace exactamente 5 años, la card muestra "5 años" (y "1 año" para 1 año exacto).
- [ ] CA-011: Las etiquetas se muestran traducidas en español e inglés según el idioma seleccionado.

### 5.2 No funcionales

- [ ] CA-NF-001: El build de Vite (`npm run build` en `frontend/`) compila sin errores.
- [ ] CA-NF-002: No se agregan dependencias nuevas al proyecto.
- [ ] CA-NF-003: El backend compila (`go build ./...`) y los tests de Go pasan (`go test ./...`).

### 5.3 Testing

- **Unit tests**: Función `calcularEdad` (casos: cumpleaños hoy, hace N años, año bisiesto, fecha futura), validación del service (campos vacíos, fecha futura).
- **Integration tests**: CRUD completo contra SQLite en memoria (patrón de `internal/storage/auto.go` y sus tests).
- **E2E tests**: Flujo de usuario: crear hijo → listar con edad → editar → eliminar.
- **Carga/Performance**: Validar tiempo de respuesta del listado con 50+ hijos (queries simples, sin joins).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración SQLite `0016_create_children` (up/down) + modelo Go | 15 min | Ninguna |
| 2 | ChildStorage + ChildService + ChildHandlers | 30 min | Fase 1 |
| 3 | Registro de rutas API en `routes.go` + wiring en `main.go` | 15 min | Fase 2 |
| 4 | Tipo `Child` en `frontend/src/types/index.ts` + métodos en api client | 15 min | Fase 3 |
| 5 | Util `calcularEdad` + tests | 15 min | Ninguna |
| 6 | HijosPage (listado con cards, menú 3 puntos, EmptyCard) | 30 min | Fase 4, 5 |
| 7 | HijoFormPage (formulario con validación) | 30 min | Fase 4 |
| 8 | Rutas React en `App.tsx` (reemplazar placeholder de Hijos) | 15 min | Fase 6, 7 |
| 9 | Claves i18n es/en (`frontend/public/i18n/` + `frontend/src/i18n/`) y build | 15 min | Fase 8 |
| 10 | Pruebas locales (`go test`, `npm run build`, levantar server, validación manual) | 30 min | Fase 9 |
| **Total** | | **~3.5 horas** | |

### 6.2 Milestones

1. **MVP**: CRUD funcional completo de hijos con edad calculada en cards (Fases 1-9).
2. **V1.0**: MVP + pruebas locales + validación manual del usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| La edad calculada difiere según zona horaria | Media | Bajo | Calcular con fecha local del cliente (`new Date()`); casos borde documentados en tests |
| Fecha de nacimiento inválida o futura | Media | Medio | Validación en service (backend) + en formulario (frontend) |
| Claves i18n editadas solo en `frontend/src/i18n/` se pierden en el build | Media | Medio | Editar SIEMPRE `frontend/public/i18n/` (fuente de verdad) y correr `npm run build`; verificar con `curl` (ver AGENTS.md) |
| Reemplazar el placeholder de SPEC-044 rompe la ruta `/pension/hijos` | Baja | Medio | Mantener la misma ruta; verificar navegación del resto de secciones de Pensión |

## 8. Notas y Referencias

- Patrón CRUD de referencia: `docs/specs/SPEC-024-autos-crud.md`, `frontend/src/pages/AutosPage.tsx`, `frontend/src/pages/AutoFormPage.tsx`.
- Backend CRUD: `internal/storage/auto.go`, `internal/services/auto.go`, `internal/api/auto_handlers.go`.
- Base del módulo Pensión Alimenticia: `docs/specs/SPEC-044-pension-alimenticia-sidebar.md` (sección Hijos en `/pension/hijos`).
- Componentes UI: `frontend/src/components/CreateMenu.tsx`, `CardMenu.tsx`, `DeleteModal.tsx`, `EmptyCard` (inline en `AutosPage.tsx`).
- Migraciones: `migrations/` (última: `0015`; la nueva será `0016`).
- Reglas de i18n: fuente de verdad en `frontend/public/i18n/` (ver AGENTS.md, sección "Reglas críticas").

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-02 | p40la-ihost-team | Estado cambiado de draft a in_progress (SPEC-044 ya mergeado en main; inicio de desarrollo en worktree) |
| 2026-09-02 | p40la-ihost-team | Implementación completa: migración 0016, backend CRUD /api/children, HijosPage con edad calculada, HijoFormPage, i18n es/en, tests. Validado en local (go test + npm run build + CRUD API). Pendiente validación manual del usuario |
| 2026-09-02 | p40la-ihost-team | Validación manual aprobada por el usuario. Release: merge a main, issue #45 cerrado, worktree limpiado |