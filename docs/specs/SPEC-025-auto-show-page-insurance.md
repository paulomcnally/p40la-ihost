---
title: "Página Show de Autos con Seguros + Vigencia y Estado en Servicios"
id: "SPEC-025"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 25
---

# Página Show de Autos con Seguros + Vigencia y Estado en Servicios

**ID**: SPEC-025
**Estado**: released
**Autor**: p40la-ihost-team
**Creado**: 2026-08-16
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

Crear una página de detalle (show) para cada auto registrado, que permita visualizar su información completa y gestionar seguros asociados. Un seguro es en realidad un **servicio existente** (que ya tiene institución, monto, frecuencia) vinculado al auto con un **tipo de cobertura**: "Daños a terceros" o "Full Cover".

Además, los servicios necesitan soportar **fechas de vigencia**: un seguro puede durar solo un año, un servicio de internet puede tener inicio y fin. Se agrega un toggle "Recurrente" que al activarse muestra campos de fecha de inicio y fin, permitiendo definir si un servicio es de duración definida o indefinida.

El modelo de datos existente ya tiene la infraestructura: los servicios existen, las instituciones existen, y los autos existen. Se necesita:
1. Agregar campos `start_date`, `end_date` y `is_recurring` a la tabla `services`
2. Crear tabla pivote `auto_services` para vincular autos con servicios (seguros)
3. Página de detalle con info del auto + sección de seguros agrupados por institución

Impacto en iHost: mínimo — ALTER TABLE ligero + tabla pivote nueva, sin dependencias nuevas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

**Servicios - Fechas de vigencia y estado:**
1. **REQ-001**: Agregar campos a tabla services: `start_date TEXT` (ISO date), `end_date TEXT` (ISO date), `is_recurring BOOLEAN DEFAULT 0`
2. **REQ-002**: Toggle "Recurrente" en ServiceFormPage — cuando se activa, muestra campos "Fecha de inicio" y "Fecha de fin"
3. **REQ-003**: Cuando `is_recurring = 0` (no recurrente), el servicio es de pago único, se ocultan los campos de fecha
4. **REQ-004**: Cuando `is_recurring = 1`, el servicio tiene vigencia definida por `start_date` y `end_date` (fin es opcional = vigencia indefinida)
5. **REQ-005**: Display de fechas de vigencia en ServicesPage (badge o texto debajo del nombre del servicio)
6. **REQ-006**: Servicios con `end_date` en el pasado se muestran como "Vencido" (badge rojo)
7. **REQ-007**: Campo `active` ya existe en services (DEFAULT 1). Servicios inactivos (`active = 0`) se muestran con card de fondo gris (`bg-gray-100`) y texto atenuado (`opacity-60`) en ServicesPage
8. **REQ-008**: Toggle "Activo" en ServiceFormPage (default: activo). El usuario puede desactivar un servicio sin eliminarlo
9. **REQ-009**: En AutoShowPage, los seguros asociados a servicios inactivos se muestran atenuados (card gris + badge "Inactivo")

**Autos - Página de detalle:**
10. **REQ-010**: Página de detalle (`AutoShowPage`) accesible desde el click en una card de AutosPage
11. **REQ-011**: Sección superior con información del auto: ícono, marca, modelo, año, color
12. **REQ-012**: Tabla `auto_services` con: id, auto_id (FK), service_id (FK), coverage_type ('daños_a_terceros' | 'full_cover'), created_at
13. **REQ-013**: API REST para gestión de seguros: GET /api/autos/:id/services, POST /api/autos/:id/services, DELETE /api/autos/:id/services/:service_id
14. **REQ-014**: Sección de seguros mostrando lista de servicios asociados al auto
15. **REQ-015**: Cada seguro muestra: nombre del servicio, institución (nombre), tipo de cobertura (badge), monto, frecuencia, fechas de vigencia
16. **REQ-016**: Seguros agrupados visualmente por institución
17. **REQ-017**: Botón "Agregar Seguro" que abre modal para seleccionar servicio y definir cobertura
18. **REQ-018**: Modal de selección: lista de servicios disponibles con búsqueda por nombre
19. **REQ-019**: Opción de eliminar seguro asociado (con confirmación)
20. **REQ-020**: Badge visual para cobertura: "Daños a terceros" (amber) / "Full Cover" (green)
21. **REQ-021**: Badge "Vencido" (rojo) para seguros cuyo servicio tiene `end_date` en el pasado
22. **REQ-022**: EmptyState cuando no hay seguros asociados

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-023**: Botón "Volver" para regresar al listado de autos
2. **REQ-024**: Toast de éxito/error al agregar/eliminar seguro
3. **REQ-025**: Loading spinner mientras se carga la información

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-026**: Resumen de total de seguros y monto total mensual/anual del auto
2. **REQ-027**: Indicador visual de cantidad de seguros en la card de AutosPage

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: ALTER TABLE + queries con JOINs ligeros, sin impacto
- **Seguridad**: Misma autenticación existente
- **Almacenamiento**: Campos adicionales ~20 bytes por servicio, tabla pivote ~50 bytes por registro
- **iHost**: Sin dependencias nuevas

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron los modelos existentes:
- **Service**: Tiene `institution_id`, `name`, `suggested_amount`, `frequency`, `icon_key`. Falta soporte de fechas de vigencia.
- **Institution**: Tiene `name`. Instituciones de seguros ya podrían existir.
- **Auto**: Entidad independiente, necesita tabla pivote para servicios.

El patrón de tabla pivote ya se usa en `institution_analyzers`. Las fechas se almacenan como TEXT en ISO format (SQLite no tiene tipo DATE nativo).

### 3.2 Terminología de vigencia

| Término | Descripción | is_recurring | start_date | end_date |
|---------|-------------|--------------|------------|----------|
| **Recurrente con vigencia definida** | Se renueva automáticamente pero tiene fecha de fin (ej: seguro anual) | 1 | Requerido | Opcional |
| **Recurrente indefinido** | Se renueva sin fecha de fin (ej: servicio de internet mensual) | 1 | Opcional | null |
| **De pago único** | Se paga una sola vez, sin recurrencia | 0 | null | null |

### 3.3 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Campos start_date + end_date + is_recurring en services | Simple, un solo registro, queries directas | ALTER TABLE en tabla existente | ✅ Seleccionada |
| Modelo Separate con vigencias | Más flexible | Más tablas, más complejidad, overkill para iHost | ❌ Rechazada |
| Campo JSON con vigencias | Todo en un campo | Sin integridad, queries difíciles | ❌ Rechazada |

### 3.4 Decisiones arquitectónicas (ADRs)

**ADR-001**: Fechas como campos directos en services
- **Contexto**: Los servicios necesitan saber cuándo empiezan y terminan
- **Decisión**: Agregar `start_date TEXT`, `end_date TEXT`, `is_recurring BOOLEAN` a la tabla services via ALTER TABLE
- **Consecuencias**: Migración simple, compatible con registros existentes (valores null por defecto)

**ADR-002**: Reutilizar servicios existentes como seguros
- **Contexto**: Los seguros son servicios que se pagan (asociados a institución)
- **Decisión**: No crear tabla de seguros separada. Un seguro = servicio vinculado a auto + tipo de cobertura
- **Consecuencias**: El usuario crea el servicio primero, luego lo asocia al auto

**ADR-003**: Tipo de cobertura en tabla pivote
- **Contexto**: Cada seguro tiene un tipo (daños a terceros o full cover)
- **Decisión**: Campo `coverage_type` en `auto_services`
- **Consecuencias**: Permite definir cobertura por auto, no por servicio

**ADR-004**: Visualización de servicios inactivos
- **Contexto**: Los servicios ya tienen campo `active` (boolean). Los usuarios necesitan desactivar servicios sin eliminarlos
- **Decisión**: Servicios inactivos se muestran con fondo gris (`bg-gray-100`), texto atenuado (`opacity-60`) y badge "Inactivo" en ServicesPage y AutoShowPage
- **Consecuencias**: Diferenciación visual clara, sin eliminar datos

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[AutoShowPage] --(API REST)--> [AutoServiceHandlers] --> [AutoServiceStorage] --> [SQLite]
      |                              |
      v                              v
[AutoInfo + Seguros]        [Reutiliza ServiceStorage, InstitutionStorage]

[ServiceFormPage] --(toggle)--> [is_recurring] --> [start_date, end_date fields]
```

### 4.2 Modelo de datos

**Modificación a tabla services (migración 0010):**
```sql
ALTER TABLE services ADD COLUMN start_date TEXT;
ALTER TABLE services ADD COLUMN end_date TEXT;
ALTER TABLE services ADD COLUMN is_recurring BOOLEAN DEFAULT 0;
```

**Nueva tabla auto_services (migración 0010):**
```
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- auto_id: INTEGER NOT NULL (FK → autos.id)
- service_id: INTEGER NOT NULL (FK → services.id)
- coverage_type: TEXT NOT NULL CHECK (coverage_type IN ('daños_a_terceros', 'full_cover'))
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP

UNIQUE(auto_id, service_id)
```

### 4.3 APIs / Contratos

#### `GET /api/autos/:id/services`

**Response 200**:
```json
[
  {
    "id": 1,
    "auto_id": 1,
    "service_id": 5,
    "coverage_type": "full_cover",
    "service_name": "Seguro Full Cover Toyota",
    "institution_name": "Seguros XYZ",
    "institution_id": 3,
    "suggested_amount": 45000,
    "frequency": "monthly",
    "icon_key": "insurance",
    "is_recurring": true,
    "start_date": "2026-01-01",
    "end_date": "2026-12-31",
    "created_at": "2026-08-16T..."
  }
]
```

#### `POST /api/autos/:id/services`

**Request**:
```json
{
  "service_id": 5,
  "coverage_type": "full_cover"
}
```

#### `DELETE /api/autos/:id/services/:service_id`

**Response 200**: `{"message": "Seguro eliminado"}`

### 4.4 Componentes Frontend

#### ServiceFormPage (modificación)
- Nuevo toggle "Recurrente" debajo de Frecuencia
- Al activar: campos "Fecha de inicio" y "Fecha de fin" (fin opcional)
- Al desactivar: limpia las fechas
- Toggle "Activo" (default: activo) — permite desactivar servicio sin eliminarlo

#### ServicesPage (modificación)
- Badge de vigencia en cada servicio (fechas, vencido)
- Servicios inactivos (`active = 0`): card con fondo gris + texto atenuado + badge "Inactivo"

#### AutoShowPage (nuevo)
- Header: botón "Volver" + info del auto
- Sección "Seguros":
  - Header: "Seguros" + badge cantidad + "Agregar Seguro"
  - Si vacío: EmptyState
  - Si tiene datos: Lista agrupada por institución
    - Header grupo: nombre institución + cantidad
    - Cards: servicio, cobertura (badge), monto, frecuencia, vigencia, vencido (badge rojo si aplica), eliminar

#### AddInsuranceModal (nuevo)
- Búsqueda de servicios disponibles
- Lista filtrable
- Selector de cobertura (radio)
- Botón "Asociar"

### 4.5 Dependencias

- **Internas**: ServiceStorage, InstitutionStorage, AutoStorage
- **Externas**: Ninguna

## 5. Criterios de Aceptación

### 5.1 Funcionales

**Servicios - Vigencia y estado:**
- [x] CA-001: ServiceFormPage muestra toggle "Recurrente"
- [x] CA-002: Al activar toggle, aparecen campos "Fecha inicio" y "Fecha fin"
- [x] CA-003: Al desactivar toggle, las fechas se limpian
- [x] CA-004: ServicesPage muestra badge de vigencia en cada servicio
- [x] CA-005: Servicios con end_date pasado muestran badge "Vencido" rojo
- [x] CA-006: ServiceFormPage muestra toggle "Activo" (default: activo)
- [x] CA-007: Servicios inactivos se muestran con card gris y texto atenuado en ServicesPage
- [x] CA-008: Badge "Inactivo" visible en servicios desactivados

**Autos - Detalle:**
- [x] CA-009: Click en card de auto navega a `/autos/:id`
- [x] CA-010: Página muestra info completa del auto
- [x] CA-011: EmptyState cuando no hay seguros
- [x] CA-012: Modal de agregar seguro funciona con búsqueda y selección de cobertura
- [x] CA-013: Seguros aparecen agrupados por institución
- [x] CA-014: Cada seguro muestra nombre, institución, cobertura (badge), monto, frecuencia, fechas
- [x] CA-015: Badge "Daños a terceros" amber, "Full Cover" green
- [x] CA-016: Badge "Vencido" rojo para seguros vencidos
- [x] CA-017: Seguros de servicios inactivos se muestran atenuados + badge "Inactivo"
- [x] CA-018: Eliminar seguro con confirmación funciona

### 5.2 No funcionales

- [x] CA-NF-001: Página carga en < 500ms con 10 seguros

### 5.3 Testing

- **Unit tests**: Validación de coverage_type, fechas de vigencia
- **Integration tests**: CRUD seguros + campos de vigencia contra SQLite
- **E2E tests**: Crear servicio recurrente → asociar a auto → verificar badges → eliminar

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración 0010: ALTER TABLE services + CREATE auto_services | 15 min | Ninguna |
| 2 | Modelo Go actualizado (Service + AutoService) | 10 min | Fase 1 |
| 3 | Storage + Service + Handlers backend | 30 min | Fase 2 |
| 4 | Rutas API | 10 min | Fase 3 |
| 5 | ServiceFormPage: toggle recurrente + fechas + toggle activo | 30 min | Fase 4 |
| 6 | ServicesPage: badge vigencia + estilo inactivo | 20 min | Fase 5 |
| 7 | Tipos y API client frontend (autos services) | 15 min | Fase 4 |
| 8 | AutoShowPage + AddInsuranceModal | 40 min | Fase 7 |
| 9 | AutosPage: click → navigate a show | 10 min | Fase 8 |
| 10 | Pruebas locales | 20 min | Fase 9 |
| **Total** | | **~4 horas** | |

### 6.2 Milestones

1. **MVP**: ServiceFormPage con toggle + fechas + AutoShowPage con CRUD seguros
2. **V1.0**: MVP + badges vigencia/vencido + agrupación por institución

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| ALTER TABLE rompe registros existentes | Muy Baja | Alto | Campos nullable, default null, sin impacto |
| Confusión servicio vs seguro | Media | Bajo | UI clara: "Agregar Seguro" = asociar servicio existente |
| Servicios disponibles muy pocos | Media | Bajo | Usuario crea servicios antes de asociar |

## 8. Notas y Referencias

- Modelo Auto: `internal/models/auto.go`
- Modelo Service: `internal/models/service.go`
- Modelo Institution: `internal/models/institution.go`
- Migraciones: `migrations/` (última: 0009)
- Páginas de referencia: `HomesPage.tsx`, `ServiceFormPage.tsx`
- Componentes UI: CreateMenu, CardMenu, DeleteModal, EmptyCard, LoadingSpinner, Toast

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-16 | p40la-ihost-team | Agregado REQ de fechas de vigencia en servicios (toggle recurrente) |
| 2026-08-16 | p40la-ihost-team | Agregado REQ de estado activo/inactivo con visualización en gris |
| 2026-08-16 | p40la-ihost-team | Release: verificado en iHost. Commit `3073e2c`, versión v0.4.7. Issue #25 cerrado. |
