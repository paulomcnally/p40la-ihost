---
title: "Extensión de campos para autos y pólizas de seguro"
id: "SPEC-028"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 28
---

# Extensión de campos para autos y pólizas de seguro

**ID**: SPEC-028  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

El módulo de autos actual permite registrar vehículos con año, marca, modelo, color e ícono, y asociarles servicios de seguros con un tipo de cobertura. Para completar la ficha técnica y legal de cada vehículo, es necesario agregar los campos de identificación del auto: motor, chasis, VIN y placa. Estos datos son obligatorios para trámites de seguro, importación y registro vehicular.

Además, la asociación entre auto y seguro requiere información de la póliza: número de póliza, certificado (opcional) y número de aseguradora. Esto permite identificar unívocamente cada contrato de seguro y facilitar la conciliación con facturas futuras.

El cambio es puramente una extensión del esquema de datos, API y formularios existentes. No se agregan dependencias externas ni servicios nuevos, manteniendo el impacto mínimo en memoria y almacenamiento del iHost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar a la entidad `Auto` los campos `motor`, `chasis`, `vin` y `placa` como cadenas de texto.
2. **REQ-002**: Todos los campos nuevos del auto son obligatorios al crear/editar un auto.
3. **REQ-003**: Agregar a la entidad `AutoService` los campos `policy_number` (número de póliza), `certificate` (certificado, nullable) e `insurer_number` (número de aseguradora).
4. **REQ-004**: `policy_number` e `insurer_number` son obligatorios al asociar un seguro a un auto. `certificate` es opcional.
5. **REQ-005**: Actualizar el formulario de auto (`AutoFormPage`) para capturar motor, chasis, VIN y placa.
6. **REQ-006**: Actualizar el modal `AddInsuranceModal` para capturar número de póliza, certificado y número de aseguradora.
7. **REQ-007**: Mostrar los nuevos datos del auto en la página de detalle (`AutoShowPage`) y en la tarjeta de listado (`AutosPage`).
8. **REQ-008**: Mostrar los nuevos datos de la póliza en el listado de seguros del auto (`AutoShowPage`).
9. **REQ-009**: Crear migraciones SQLite `up`/`down` para las nuevas columnas sin perder datos existentes.
10. **REQ-010**: Validar en backend y frontend los campos obligatorios, mostrando mensajes claros al usuario.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Aplicar formato/visualización consistente para placas (mayúsculas) y VIN (mayúsculas sin espacios) en el frontend.
2. **REQ-012**: Permitir búsqueda/filter básico por placa en el listado de autos.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013**: Validar que el VIN tenga 17 caracteres (estándar internacional).
2. **REQ-014**: Mostrar un indicador visual si falta algún dato obligatorio del auto o de la póliza.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Las consultas a SQLite deben mantenerse con índices existentes; no se requieren índices adicionales para campos de texto pequeños.
- **Seguridad**: Los datos de identificación vehicular se almacenan en la base local; no se transmiten a servicios externos.
- **Almacenamiento**: Campos de texto cortos (`TEXT`), sin impacto significativo en el tamaño de la DB.
- **Disponibilidad**: El health check `/health` no se ve afectado.
- **iHost**: Sin dependencias nuevas; cambios limitados a Go stdlib, React y Tailwind ya presentes.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se revisó el estado actual del módulo de autos:

- Tabla `autos`: `id`, `year`, `model`, `brand`, `color`, `icon`, `created_at`, `updated_at`.
- Tabla `auto_services`: `id`, `auto_id`, `service_id`, `coverage_type`, `created_at`.
- Modelos Go en `internal/models/auto.go` y `internal/models/auto_service.go`.
- Storage en `internal/storage/auto.go` y `internal/storage/auto_service.go`.
- Handlers en `internal/api/auto_handlers.go` y `internal/api/auto_service_handlers.go`.
- Servicios en `internal/services/auto.go` y `internal/services/auto_service.go`.
- Frontend: `AutoFormPage.tsx`, `AutoShowPage.tsx`, `AutosPage.tsx`, `AddInsuranceModal.tsx`, `types/index.ts`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Agregar columnas a tablas existentes | Simple, mantiene relaciones, migración con `ALTER TABLE` | Requiere actualizar todos los scans/handlers | ✅ Seleccionada |
| Crear tablas auxiliares para datos técnicos/legales | Normalización, permite historial | Más joins, más complejo, overkill para 4-7 campos | ❌ Rechazada |
| Usar JSON en una columna `metadata` | Flexible | Pierde tipado, validación y búsqueda simple | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Extender tablas existentes con columnas `TEXT`.
- **Contexto**: Los datos son atributos directos del auto y de la póliza; no requieren historial ni relaciones adicionales.
- **Decisión**: Usar `ALTER TABLE ... ADD COLUMN` en migraciones nuevas (`0012_*.sql`).
- **Consecuencias**: Todos los scans, modelos, servicios, handlers y formularios deben actualizarse. La migración preserva datos existentes.

**ADR-002**: Campos nuevos obligatorios en backend y frontend.
- **Contexto**: Se requiere integridad de datos desde el inicio para evitar registros incompletos.
- **Decisión**: Backend valida strings no vacíos; frontend aplica `required` y muestra errores por toast.
- **Consecuencias**: Mejor calidad de datos, pero los autos/seguros existentes quedarán sin información nueva hasta que el usuario la complete.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Usuario]
   │
   ▼
[React Frontend] ──(REST)──▶ [Go API /internal/api]
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
              [services]    [services]    [storage]
              /auto.go   /auto_service.go    │
                    │             │          │
                    └─────────────┴──────────┘
                                  ▼
                           [SQLite /data/app.db]
```

### 4.2 Componentes

#### 4.2.1 Migración `migrations/0012_add_auto_and_insurance_fields.up.sql`
- **Responsabilidad**: Agregar columnas `motor`, `chasis`, `vin`, `placa` a `autos`; y `policy_number`, `certificate`, `insurer_number` a `auto_services`.
- **Interfaz**: Script SQL ejecutado por el migrador existente.
- **Dependencias**: Migraciones `0009_create_autos` y `0010_add_service_vigencia_and_auto_services` ya aplicadas.

#### 4.2.2 Modelos Go
- **Responsabilidad**: Representar los nuevos campos en `models.Auto` y `models.AutoService`/`models.AutoServiceDetail`.
- **Ubicación**: `internal/models/auto.go`, `internal/models/auto_service.go`.

#### 4.2.3 Storage
- **Responsabilidad**: Lectura/escritura de las nuevas columnas en SQLite.
- **Ubicación**: `internal/storage/auto.go`, `internal/storage/auto_service.go`.

#### 4.2.4 Servicios
- **Responsabilidad**: Validar datos de negocio (no vacíos, longitudes, etc.).
- **Ubicación**: `internal/services/auto.go`, `internal/services/auto_service.go`.

#### 4.2.5 Handlers
- **Responsabilidad**: Recibir/parsear los nuevos campos en JSON.
- **Ubicación**: `internal/api/auto_handlers.go`, `internal/api/auto_service_handlers.go`.

#### 4.2.6 Frontend
- **Responsabilidad**: Capturar y mostrar los nuevos campos.
- **Ubicación**: `frontend/src/pages/AutoFormPage.tsx`, `frontend/src/pages/AutoShowPage.tsx`, `frontend/src/pages/AutosPage.tsx`, `frontend/src/components/AddInsuranceModal.tsx`, `frontend/src/types/index.ts`.

### 4.3 Modelo de datos

```
Entidad: Auto
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- year: INTEGER NOT NULL
- model: TEXT NOT NULL
- brand: TEXT NOT NULL
- color: TEXT NOT NULL
- icon: TEXT NOT NULL DEFAULT 'vehicle'
- motor: TEXT NOT NULL              # Número de motor
- chasis: TEXT NOT NULL             # Número de chasis
- vin: TEXT NOT NULL                # Vehicle Identification Number
- placa: TEXT NOT NULL              # Placa del vehículo
- created_at: DATETIME
- updated_at: DATETIME

Entidad: AutoService
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- auto_id: INTEGER NOT NULL (FK autos)
- service_id: INTEGER NOT NULL (FK services)
- coverage_type: TEXT NOT NULL CHECK ('daños_a_terceros', 'full_cover')
- policy_number: TEXT NOT NULL      # Número de póliza
- certificate: TEXT                 # Certificado (nullable)
- insurer_number: TEXT NOT NULL     # Número de aseguradora
- created_at: DATETIME
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/autos`

**Request**:
```json
{
  "year": 2020,
  "model": "Corolla",
  "brand": "Toyota",
  "color": "Rojo",
  "icon": "vehicle",
  "motor": "2ZR12345678",
  "chasis": "JTDBU4EE1B9123456",
  "vin": "JTDBU4EE1B9123456",
  "placa": "P123ABC"
}
```

**Response 201**:
```json
{
  "id": 1,
  "year": 2020,
  "model": "Corolla",
  "brand": "Toyota",
  "color": "Rojo",
  "icon": "vehicle",
  "motor": "2ZR12345678",
  "chasis": "JTDBU4EE1B9123456",
  "vin": "JTDBU4EE1B9123456",
  "placa": "P123ABC",
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": "2026-08-16T00:00:00Z"
}
```

**Response Error**:
```json
{
  "error": "invalid_request",
  "message": "el motor es requerido"
}
```

#### Endpoint: `PUT /api/autos/{id}`

Mismo body que `POST /api/autos`. Response 200 con el auto actualizado.

#### Endpoint: `POST /api/autos/{id}/services`

**Request**:
```json
{
  "service_id": 5,
  "coverage_type": "full_cover",
  "policy_number": "POL-2026-001",
  "certificate": "CERT-123456",
  "insurer_number": "ASEG-987654"
}
```

**Response 201**:
```json
{
  "id": 10,
  "auto_id": 1,
  "service_id": 5,
  "coverage_type": "full_cover",
  "policy_number": "POL-2026-001",
  "certificate": "CERT-123456",
  "insurer_number": "ASEG-987654",
  "created_at": "2026-08-16T00:00:00Z"
}
```

**Response Error**:
```json
{
  "error": "invalid_request",
  "message": "el número de póliza es requerido"
}
```

#### Endpoint: `GET /api/autos/{id}/services`

**Response 200** incluye `policy_number`, `certificate`, `insurer_number` en cada item.

### 4.5 Dependencias

- **Internas**: Módulos `models`, `storage`, `services`, `api`, migraciones, frontend React.
- **Externas**: Ninguna. Solo Go stdlib, React y Tailwind ya en el proyecto.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un auto existente, cuando se edita, entonces se pueden guardar motor, chasis, VIN y placa.
- [x] CA-002: Dado un auto nuevo, cuando se crea sin motor/chasis/VIN/placa, entonces el backend rechaza la solicitud con error 400.
- [x] CA-003: Dado un auto, cuando se asocia un seguro, entonces se exige `policy_number` e `insurer_number` y `certificate` es opcional.
- [x] CA-004: Dado un seguro asociado, cuando se lista en la página del auto, entonces se muestran póliza, certificado (si existe) y número de aseguradora.
- [x] CA-005: Dado un auto, cuando se visualiza su detalle, entonces se muestran motor, chasis, VIN y placa.
- [x] CA-006: Dado un auto en el listado, cuando se renderiza la tarjeta, entonces se muestra la placa junto a año y color.

### 5.2 No funcionales

- [x] CA-NF-001: La migración aplica sin pérdida de datos en SQLite con WAL habilitado.
- [x] CA-NF-002: No se agregan librerías ni dependencias nuevas.
- [x] CA-NF-003: El build de Go y el build de Vite continúan funcionando sin warnings.

### 5.3 Testing

- **Manual local**: Crear/editar auto con campos nuevos, asociar seguro con póliza, verificar visualización en listado y detalle.
- **Migración**: Levantar la app con una DB existente y confirmar que `ALTER TABLE` agrega columnas sin errores.
- **Validación**: Enviar requests con campos faltantes y confirmar errores 400.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear migraciones `0012_add_auto_and_insurance_fields.up.sql` y `.down.sql` | 1 h | Ninguna |
| 2 | Actualizar modelos Go (`Auto`, `AutoService`, `AutoServiceDetail`) | 1 h | Fase 1 |
| 3 | Actualizar storage y servicios (scans, queries, validaciones) | 2 h | Fase 2 |
| 4 | Actualizar handlers API (request structs) | 1 h | Fase 3 |
| 5 | Actualizar tipos y frontend (formularios, listado, detalle) | 3 h | Fase 4 |
| 6 | Probar localmente (migración, CRUD, validaciones) | 2 h | Fase 5 |

### 6.2 Milestones

1. **MVP**: Backend funcional con migraciones, modelos, API y validaciones.
2. **V1.0**: Frontend actualizado y validado manualmente en local.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Datos existentes sin los nuevos campos obligatorios | Media | Medio | Los campos se agregan con `DEFAULT ''` temporalmente en migración, o el frontend fuerza edición al abrir autos antiguos. |
| Scans de SQL fallan por columnas nuevas | Baja | Alto | Actualizar todos los `SELECT` y `Scan` en storage antes de probar. |
| Formularios frontend desfasados con backend | Baja | Medio | Usar `Partial<Auto>` y validar que todos los campos se envíen. |
| `gh` CLI no disponible para actualizar issue | Media | Bajo | Continuar con warning y actualizar manualmente si es necesario. |

## 8. Notas y Referencias

- Especificaciones relacionadas:
  - [SPEC-024: Módulo CRUD de Autos](SPEC-024-autos-crud.md)
  - [SPEC-025: Página Show de Autos con Seguros + Vigencia y Estado en Servicios](SPEC-025-auto-show-page-insurance.md)
  - [SPEC-026: Categorías de Instituciones con Seed y Filtro de Seguros](SPEC-026-categorias-institution-insurance.md)
- Archivos a modificar (lista no exhaustiva):
  - `migrations/0012_add_auto_and_insurance_fields.up.sql`
  - `migrations/0012_add_auto_and_insurance_fields.down.sql`
  - `internal/models/auto.go`
  - `internal/models/auto_service.go`
  - `internal/storage/auto.go`
  - `internal/storage/auto_service.go`
  - `internal/services/auto.go`
  - `internal/services/auto_service.go`
  - `internal/api/auto_handlers.go`
  - `internal/api/auto_service_handlers.go`
  - `frontend/src/types/index.ts`
  - `frontend/src/pages/AutoFormPage.tsx`
  - `frontend/src/pages/AutoShowPage.tsx`
  - `frontend/src/pages/AutosPage.tsx`
  - `frontend/src/components/AddInsuranceModal.tsx`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-16 | p40la-ihost-team | Estado cambiado a pending_execution |
| 2026-08-16 | p40la-ihost-team | Estado cambiado a in_progress para comenzar implementación |
| 2026-08-16 | p40la-ihost-team | Implementación completada y verificada en local; estado cambiado a pending_release |
| 2026-08-16 | paulomcnally | Usuario confirmó pruebas satisfactorias; liberado a main (commit `bdc63aa`) |
