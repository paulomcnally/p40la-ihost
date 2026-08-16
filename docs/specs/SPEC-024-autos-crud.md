---
title: "Módulo CRUD de Autos"
id: "SPEC-024"
status: "pending_release"
author: "p40la-ihost-team"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 24
---

# Módulo CRUD de Autos

**ID**: SPEC-024
**Estado**: draft
**Autor**: p40la-ihost-team
**Creado**: 2026-08-16
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

Agregar un nuevo módulo "Autos" al sidebar del dashboard, con un CRUD completo para gestionar vehículos. El formulario permite registrar: Año, Modelo, Marca y Color. Cada vehículo se asocia con un ícono representativo (Van, Camioneta, Vehículo o Moto) que se selecciona al momento de crear o editar.

El módulo sigue exactamente los patrones CRUD existentes (Homes, Institutions, Services) para minimizar la deuda técnica y mantener consistencia en la UX. Impacto en recursos del iHost: mínimo — una tabla SQLite adicional, sin dependencias nuevas, sin cambios en la infraestructura Docker.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar item "Autos" al sidebar con ícono representativo (Van/Moto)
2. **REQ-002**: Tabla SQLite `autos` con campos: id, year, model, brand, color, icon (texto), created_at, updated_at
3. **REQ-003**: API REST CRUD: GET /api/autos, GET /api/autos/:id, POST /api/autos, PUT /api/autos/:id, DELETE /api/autos/:id
4. **REQ-004**: Página de listado (`AutosPage`) con cards en grid, usando CreateMenu y CardMenu (3 puntos)
5. **REQ-005**: Página de formulario (`AutoFormPage`) con campos: Año, Modelo, Marca, Color, selector de ícono
6. **REQ-006**: Selector de íconos: Van, Camioneta, Vehículo, Moto (4 opciones visuales)
7. **REQ-007**: Modal de confirmación para eliminar (DeleteModal existente)
8. **REQ-008**: EmptyCard cuando no hay autos registrados

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Validación de formulario: año (número, rango razonable), modelo (no vacío), marca (no vacío), color (no vacío)
2. **REQ-010**: Toast de éxito/error al crear, editar, eliminar
3. **REQ-011**: Loading spinner mientras se cargan los datos

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Búsqueda/filtro por marca o modelo en la página de listado
2. **REQ-013**: Paginación si hay muchos registros

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto significativo — CRUD ligero, queries simples sobre SQLite
- **Seguridad**: Misma autenticación existente (authMiddleware en todas las rutas)
- **Almacenamiento**: Tabla SQLite adicional, registros de ~100 bytes cada uno
- **Disponibilidad**: Sin cambios en health check
- **iHost**: Sin dependencias nuevas, sin increase de RAM significativo

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron los CRUDs existentes en el proyecto para replicar el patrón exacto:
- **HomesPage / HomeFormPage**: CRUD base con cards, CreateMenu, CardMenu, DeleteModal
- **InstitutionsPage / InstitutionFormPage**: CRUD simple sin dependencias
- **ServicesPage / ServiceFormPage**: CRUD con dependencias (home, institution) — más complejo
- **Icons.tsx**: Catálogo de íconos SVG inline disponibles

Los íconos Van, Camioneta, Vehículo y Moto no existen actualmente en `Icons.tsx` — se deben agregar.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| CRUD completo (list+form+api+db) | Funcionalidad completa, consistente | Más archivos a crear | ✅ Seleccionada |
| Solo frontend con JSON local | Rápido de implementar | No persiste, no escala | ❌ Rechazada |
| Reutilizar módulo existente con variantes | Menos código nuevo | Confunde patrones, acoplamiento | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Iconos del CRUD de autos
- **Contexto**: El usuario pide iconos de Van, Camioneta, Vehículo, Moto para el selector
- **Decisión**: Agregar 4 iconos SVG nuevos a `Icons.tsx` siguiendo el patrón existente de icons inline
- **Consecuencias**: +4 iconos en el bundle (mínimo impacto, SVGs inline son ligeros)

**ADR-002**: Campo icon en la tabla
- **Contexto**: Cada auto tiene un icono representativo
- **Decisión**: Campo `icon TEXT NOT NULL DEFAULT 'vehicle'` en la tabla, almacena el nombre del icono (ej: "van", "truck", "vehicle", "moto")
- **Consecuencias**: Simple, consultable, sin joins

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[AutoFormPage] --(API REST)--> [AutoHandlers] --> [AutoService] --> [AutoStorage]
      |                              |
      v                              v
[SQLite autos]              [Frontend estático]
```

### 4.2 Componentes

#### 4.2.1 Backend - AutoStorage
- **Responsabilidad**: Queries CRUD contra SQLite
- **Interfaz**: List, GetByID, Create, Update, Delete
- **Dependencias**: `database/sql`
- **Ubicación**: `internal/storage/auto.go`

#### 4.2.2 Backend - AutoService
- **Responsabilidad**: Validación de negocio (año razonable, campos no vacíos)
- **Interfaz**: Lista, obtiene, crea, actualiza, elimina
- **Dependencias**: AutoStorage
- **Ubicación**: `internal/services/auto.go`

#### 4.2.3 Backend - AutoHandlers
- **Responsabilidad**: HTTP handlers para la API REST
- **Interfaz**: HandleListAutos, HandleGetAuto, HandleCreateAuto, HandleUpdateAuto, HandleDeleteAuto
- **Dependencias**: AutoService
- **Ubicación**: `internal/api/auto_handlers.go`

#### 4.2.4 Frontend - AutosPage
- **Responsabilidad**: Listado de autos en cards con grid
- **Interfaz**: Componente React funcional
- **Dependencias**: api, Icon, CreateMenu, CardMenu, DeleteModal, EmptyCard
- **Ubicación**: `frontend/src/pages/AutosPage.tsx`

#### 4.2.5 Frontend - AutoFormPage
- **Responsabilidad**: Formulario de creación/edición con selector de ícono
- **Interfaz**: Componente React funcional, maneja :id param
- **Dependencias**: api, Icon, useToast, useNavigate
- **Ubicación**: `frontend/src/pages/AutoFormPage.tsx`

### 4.3 Modelo de datos

```
Entidad: auto
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- year: INTEGER NOT NULL (año del vehículo, ej: 2024)
- model: TEXT NOT NULL (modelo, ej: "Corolla")
- brand: TEXT NOT NULL (marca, ej: "Toyota")
- color: TEXT NOT NULL (color, ej: "Rojo")
- icon: TEXT NOT NULL DEFAULT 'vehicle' (nombre del icono: van, truck, vehicle, moto)
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP

Relaciones: Ninguna (entidad independiente)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/autos`

**Response 200**:
```json
[
  {
    "id": 1,
    "year": 2024,
    "model": "Corolla",
    "brand": "Toyota",
    "color": "Rojo",
    "icon": "vehicle",
    "created_at": "2026-08-16T10:00:00Z",
    "updated_at": "2026-08-16T10:00:00Z"
  }
]
```

#### Endpoint: `GET /api/autos/:id`

**Response 200**: Objeto `auto` individual
**Response 404**: `{"error": "not_found", "message": "Auto no encontrado"}`

#### Endpoint: `POST /api/autos`

**Request**:
```json
{
  "year": 2024,
  "model": "Corolla",
  "brand": "Toyota",
  "color": "Rojo",
  "icon": "vehicle"
}
```

**Response 201**: Objeto `auto` creado con ID

#### Endpoint: `PUT /api/autos/:id`

**Request**: Mismo schema que POST
**Response 200**: Objeto `auto` actualizado

#### Endpoint: `DELETE /api/autos/:id`

**Response 200**: `{"message": "Auto eliminado"}`

### 4.5 Dependencias

- **Internas**: Ninguna nueva. Se reutilizan componentes existentes (CreateMenu, CardMenu, DeleteModal, Toast, Icon, LoadingSpinner, EmptyCard)
- **Externas**: Ninguna nueva

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el sidebar, cuando hago click en "Autos", se navega a la página de listado de autos
- [ ] CA-002: Dado la página de listado vacía, cuando no hay autos, se muestra el EmptyCard con título, descripción y botón "Crear Auto"
- [ ] CA-003: Dado la página de listado, cuando hay autos, se muestran cards en grid con icono, marca, modelo, año, color
- [ ] CA-004: Dado una card de auto, cuando hago hover, aparece el menú de 3 puntos con opciones Editar y Eliminar
- [ ] CA-005: Dado el formulario de creación, cuando completo todos los campos y selecciono un ícono, al guardar se crea el auto y se redirige al listado
- [ ] CA-006: Dado el formulario de edición, cuando modifico campos y guardo, se actualiza el auto
- [ ] CA-007: Dado el menú de acciones, cuando selecciono "Eliminar", aparece el modal de confirmación
- [ ] CA-008: Dado el modal de confirmación, cuando escribo "confirmo" y hago click en eliminar, se elimina el auto
- [ ] CA-009: Dado el selector de íconos, se muestran las 4 opciones: Van, Camioneta, Vehículo, Moto
- [ ] CA-010: Al crear/editar con campos vacíos, se muestra toast de error

### 5.2 No funcionales

- [ ] CA-NF-001: La página carga en < 500ms con 100 autos en la tabla
- [ ] CA-NF-002: El módulo no increase el uso de RAM del contenedor en más de 5MB

### 5.3 Testing

- **Unit tests**: Validación de servicio (campos vacíos, año inválido)
- **Integration tests**: CRUD completo contra SQLite en memoria
- **E2E tests**: Flujos de usuario: crear, listar, editar, eliminar auto
- **Carga/Performance**: Validar tiempo de respuesta con 100+ registros

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración SQLite + modelo Go | 15 min | Ninguna |
| 2 | Storage + Service + Handlers | 30 min | Fase 1 |
| 3 | Rutas API + wiring en main.go | 15 min | Fase 2 |
| 4 | Tipos y API client frontend | 15 min | Fase 3 |
| 5 | Iconos nuevos en Icons.tsx | 15 min | Ninguna |
| 6 | AutosPage (listado) | 30 min | Fase 4, 5 |
| 7 | AutoFormPage (formulario) | 30 min | Fase 4, 5 |
| 8 | Rutas React + Sidebar | 15 min | Fase 6, 7 |
| 9 | Pruebas locales y validación | 20 min | Fase 8 |
| **Total** | | **~3 horas** | |

### 6.2 Milestones

1. **MVP**: CRUD funcional completo (Fases 1-8)
2. **V1.0**: MVP + pruebas + validación en local

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Iconos SVG no quedan estéticos | Baja | Bajo | Usar SVGs simples y consistentes con los existentes |
| Validación de año inválido | Baja | Bajo | Rango 1900-2099 cubre cualquier caso razonable |
| Conflicto de nombres de tablas | Muy Baja | Alto | Nombre `autos` no conflicta con tablas existentes |

## 8. Notas y Referencias

- Patrón CRUD: `frontend/src/pages/HomesPage.tsx`, `frontend/src/pages/HomeFormPage.tsx`
- Backend CRUD: `internal/storage/home.go`, `internal/services/home.go`, `internal/api/home_handlers.go`
- Iconos: `frontend/src/components/Icons.tsx`
- Componentes UI: `frontend/src/components/CreateMenu.tsx`, `CardMenu.tsx`, `DeleteModal.tsx`
- Migraciones: `migrations/` (última: 0008)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-16 | p40la-ihost-team | Implementación completa: backend + frontend, pruebas manuales aprobadas |
