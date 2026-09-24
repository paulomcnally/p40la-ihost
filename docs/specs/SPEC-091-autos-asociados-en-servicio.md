---
title: "Pólizas y ciclos de servicio: ver, agregar y renovar en el detalle"
id: "SPEC-091"
status: "pending_release"
author: "opencode"
created: "2026-09-24"
updated: "2026-09-24"
github_issue: 94
---

# Pólizas y ciclos de servicio: ver, agregar y renovar en el detalle

**ID**: SPEC-091  
**Estado**: pending_release  
**Autor**: opencode  
**Creado**: 2026-09-24  
**Actualizado**: 2026-09-24

---

## 1. Resumen Ejecutivo

La asociación auto ↔ servicio (póliza de seguro) vive en `auto_services` y solo es visible desde el lado del auto (`AutoShowPage`). Desde el detalle del servicio (`/services/bills/:id`, `BillsPage`) no hay forma de ver ni gestionar qué autos tienen el servicio como seguro, y no existe concepto de "ciclo" de vigencia renovable: el servicio acumula facturas indefinidamente sin distinguir a qué período de la póliza pertenecen.

Este spec unifica la gestión de pólizas y de ciclos en el detalle del servicio:
1. **Pestaña "Pólizas"** (solo para servicios de instituciones de seguro): lista los autos asociados con sus datos de póliza, y permite **agregar, editar y eliminar** la asociación desde el servicio (hoy solo se hace desde el auto).
2. **Ciclos de servicio**: nuevo modelo `service_cycles` + `bills.cycle_id`. Cada renovación crea un ciclo nuevo (sequence 1, 2, 3…) y las facturas quedan **asociadas a su ciclo**, dejando rastro de a qué período pertenecen (no más "facturas infinitas" sin contexto).
3. **Botón "Renovar"** (aplica a todos los servicios): un formulario que edita lo necesario (fechas del nuevo período, monto sugerido) y crea el ciclo nuevo, conservando `webhook_uuid` e historial.

Todo es solo-lectura salvo los endpoints de ciclo/póliza, sin dependencias nuevas. El `webhook_uuid` no cambia con la renovación (los recibos del nuevo ciclo siguen cayendo igual). Prioriza simplicidad y bajo consumo (iHost): una tabla nueva, una columna, y lógica liviana.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Pestaña "Pólizas" en `BillsPage` que lista los autos asociados al servicio (ícono, marca/modelo, placa, cobertura, póliza, certificado, aseguradora) con navegación al detalle del auto. *(Ya implementado en el commit 8d7791c.)*
2. **REQ-002**: La pestaña "Pólizas" **solo se muestra para servicios cuya institución pertenece a la categoría `insurance`** (campo computado `is_insurance`). La acción **"Renovar" aplica a TODOS los servicios** (todo servicio tiene un período renovable), independientemente de la categoría de su institución.
3. **REQ-003**: **Agregar una póliza** desde la pestaña "Pólizas": modal que elige un auto no asociado + cobertura + póliza + certificado + aseguradora, reutilizando `POST /api/autos/{id}/services`.
4. **REQ-004**: **Editar y eliminar** una póliza desde la misma pestaña (reutilizando `PUT`/`DELETE /api/autos/{id}/services/{service_id}`).
5. **REQ-005**: **Modelo de ciclos**: tabla `service_cycles` (`service_id`, `sequence`, `start_date`, `end_date`) y columna `bills.cycle_id`. Backfill: ciclo 1 para servicios con vigencia y asociación de sus facturas.
6. **REQ-006**: **Botón "Renovar"** en el detalle del servicio: formulario que edita `start_date`, `end_date` y `suggested_amount` del nuevo período y crea un ciclo nuevo (sequence +1), conservando `webhook_uuid` e historial. Las facturas existentes conservan su `cycle_id` (rastro del ciclo anterior).
7. **REQ-007**: Las facturas creadas por cualquier vía (generación automática, scheduler, webhook, manual) quedan etiquetadas con el ciclo que contiene su período (o el último ciclo activo).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: `GET /api/services/{id}/cycles` para listar los ciclos del servicio (usado por el modal de renovación para mostrar "Ciclo actual").
2. **REQ-009**: `cycle_id` en la respuesta de facturas (`models.Bill`) para exponer la asociación.
3. **REQ-010**: Tests unitarios de storage (ciclos, backfill, tagging de factura, renew) y handler.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-011**: Badge "Ciclo N" visible en las facturas del detalle cuando tengan ciclo.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Tagging de ciclo = 1 consulta extra en la creación de facturas (indexada por `service_id`). Listar pólizas = 1 JOIN. Sin N+1.
- **Seguridad**: Endpoints de ciclo y pólizas autenticados (`authMiddleware`). `POST /api/services/{id}/renew` valida que el servicio sea de seguro.
- **Almacenamiento**: Tabla `service_cycles` + columna `bills.cycle_id` + índices. Sin dependencias externas.
- **Disponibilidad**: La renovación no afecta webhooks ni scheduler (el `webhook_uuid` no cambia).
- **iHost**: Migración SQLite liviana; lógica Go pura sin librerías nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `auto_services` (`migrations/0010`, `0012`): `auto_id`, `service_id`, `coverage_type`, `policy_number`, `certificate`, `insurer_number`. El reverse lookup ya existe (`AutoServiceStorage.ListByService`, SPEC-091 parte 1).
- `bills` tiene `UNIQUE(service_id, year, month)`; no hay noción de ciclo. La creación pasa por `BillStorage.Create` (usado por `generateCurrentBill`, scheduler, webhook, handlers).
- La categoría de institución `insurance` se usa en `ListAvailableServices` (`storage/auto_service.go:129-151`, JOIN `institution_categories ic WHERE ic.key='insurance'`).
- `ServiceStorage.serviceColumns` (una sola const) alimenta list/get/webhook-lookup; se puede extender con un subquery `is_insurance`.
- `ServiceService.Update` valida y regenera la factura actual; la renovación no debe pasar por ahí (es un flujo dedicado).
- El webhook no trae ciclo; la factura se etiqueta en el momento de crear/actualizar.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Tabla `service_cycles` + `bills.cycle_id` | Rastro explícito por período; soporta N renovaciones; consultas simples | Una tabla y una columna nuevas | ✅ Seleccionada |
| Solo columnas `cycle_start`/`cycle_end` en `bills` | Sin tabla nueva | No guarda secuencia ni permite listar ciclos; duplica datos | ❌ Rechazada |
| Reescribir `bills.period` como string de ciclo | Simple de leer | Pierde semántica de mes/año y rompe `UNIQUE`/análisis | ❌ Rechazada |
| Renovar vía `PUT /api/services/{id}` + ciclo en frontend | Sin endpoint nuevo | No es atómico; deja el ciclo sin crear si falla el PUT | ❌ Rechazada |
| Gating por `icon_key` (`insurance`) | Cero backend extra | El ícono no garantiza la categoría real de la institución | ❌ Rechazada |
| Gating por categoría de institución (`ic.key='insurance'`) | Semántica real de negocio | Requiere subquery en la API de servicios | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: `service_cycles` + `bills.cycle_id` como rastro de ciclo**
- **Contexto**: Las facturas deben recordar a qué ciclo de la póliza pertenecen; la renovación abre un ciclo nuevo.
- **Decisión**: Tabla `service_cycles` (sequence creciente por servicio, con fechas) y `bills.cycle_id` (FK nullable). El ciclo de una factura se resuelve por período al crearla.
- **Consecuencias**: Se puede listar el historial de ciclos y saber qué facturas formaron parte de cada uno. Los servicios sin vigencia quedan con `cycle_id` NULL (sin romper el flujo existente).

**ADR-002: `is_insurance` computado en la API de servicios (no en frontend)**
- **Contexto**: El gating de Pólizas/Renovar depende de la categoría real de la institución.
- **Decisión**: `ServiceStorage` agrega un subquery `EXISTS(...ic.key='insurance')` a `serviceColumns` y expone `Service.IsInsurance bool`.
- **Consecuencias**: Una sola fuente de verdad; sin fetches extra en frontend. Costo despreciable en SQLite.

**ADR-003: Endpoint dedicado `POST /api/services/{id}/renew` (atómico)**
- **Contexto**: Renovar = crear ciclo + actualizar fechas/monto del servicio, en una sola operación.
- **Decisión**: Nuevo `ServiceCycleService.Renew` que, en una operación, crea el ciclo (sequence max+1) y actualiza `services.start_date/end_date/suggested_amount`. Handler `ServiceCycleHandlers.RenewService`. Aplica a todos los servicios (el período renovable no es exclusivo de seguros).
- **Consecuencias**: No se toca `ServiceService.Update` (evita regenerar facturas colaterales). El `webhook_uuid` permanece intacto.

**ADR-004: Tagging de ciclo en `BillStorage.Create`**
- **Contexto**: Las facturas se crean desde 4 puntos distintos.
- **Decisión**: `BillStorage.Create` resuelve `cycle_id` con `ServiceCycleStorage.FindForPeriod` (ciclo que contiene el período; fallback al último) antes de insertar.
- **Consecuencias**: Todas las vías (auto-gen, scheduler, webhook, manual) quedan etiquetadas sin duplicar lógica.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[BillsPage]
  ├── ?tab=polizas (solo si service.is_insurance)
  │     GET /api/services/{id}/autos ──────────────► AutoServiceStorage.ListByService
  │     POST/PUT/DELETE /api/autos/{id}/services…  ► AutoServiceService (existente)
  ├── CreateMenu → "Renovar" (solo si is_insurance)
  │     GET  /api/services/{id}/cycles ────────────► ServiceCycleStorage.ListByService
  │     POST /api/services/{id}/renew ─────────────► ServiceCycleService.Renew (tx: ciclo + update service)
  └── facturas
        [BillStorage.Create → resolve cycle_id por período]
```

### 4.2 Componentes

#### 4.2.1 `ServiceCycleStorage` (nuevo, `internal/storage/service_cycle.go`)
- **Responsabilidad**: CRUD de ciclos + resolución por período.
- **Interfaz**:
  - `Create(ctx, cycle *models.ServiceCycle) (*models.ServiceCycle, error)`
  - `NextSequence(ctx, serviceID int64) (int, error)` (max+1, base 1)
  - `ListByService(ctx, serviceID int64) ([]models.ServiceCycle, error)`
  - `FindForPeriod(ctx, serviceID int64, year, month int) (*models.ServiceCycle, error)`
  - `BackfillExistingCycles(ctx) error`
- **Ubicación**: `internal/storage/service_cycle.go`.

#### 4.2.2 `ServiceCycleService` (nuevo, `internal/services/service_cycle.go`)
- **Responsabilidad**: Lógica de renovación.
- **Interfaz**:
  - `Renew(ctx, serviceID int64, startDate, endDate string, suggestedAmount *float64) (*models.Service, *models.ServiceCycle, error)`
  - `ListByService(ctx, serviceID int64) ([]models.ServiceCycle, error)`
- **Validaciones**: servicio existe y `IsInsurance`; `start < end`; monto no negativo.
- **Ubicación**: `internal/services/service_cycle.go`.

#### 4.2.3 `ServiceCycleHandlers` (nuevo, `internal/api/service_cycle_handlers.go`)
- **Responsabilidad**: `RenewService` y `ListServiceCycles`.
- **Rutas**: `POST /api/services/{id}/renew`, `GET /api/services/{id}/cycles` (con `authMiddleware`).

#### 4.2.4 Modificaciones existentes
- `internal/storage/bill.go`: `Create` resuelve `cycle_id`; `models.Bill.CycleID *int64`; scan de `cycle_id`.
- `internal/storage/service.go`: subquery `is_insurance` en `serviceColumns`; `models.Service.IsInsurance bool`.
- `internal/models/`: `ServiceCycle` y `ServiceCycleDetail`.
- `internal/api/handlers.go` + `routes.go`: wiring de `ServiceCycleHandlers`.

#### 4.2.5 Frontend
- `frontend/src/pages/BillsPage.tsx`: gate de pestaña/acción por `is_insurance`; menú "Renovar"; pestaña Pólizas con Agregar/Editar/Eliminar.
- `frontend/src/components/AddPolicyModal.tsx` (nuevo): agregar/editar póliza desde el servicio.
- `frontend/src/components/RenewServiceModal.tsx` (nuevo): formulario de renovación con ciclos.
- `frontend/src/types/index.ts`, `frontend/src/api/index.ts`, i18n `es/en`.

### 4.3 Modelo de datos

```
Entidad: service_cycles (nueva)
- id INTEGER PK AUTOINCREMENT
- service_id INTEGER NOT NULL REFERENCES services(id)
- sequence INTEGER NOT NULL
- start_date TEXT (YYYY-MM-DD)
- end_date TEXT (YYYY-MM-DD)
- created_at DATETIME DEFAULT CURRENT_TIMESTAMP
- UNIQUE(service_id, sequence)
- Índice idx_service_cycles_service_id

Entidad: bills (alterada)
- cycle_id INTEGER NULL (REFERENCES service_cycles(id))
- Índice idx_bills_cycle_id

Entidad: services (computado en API)
- is_insurance BOOLEAN (derivado de instituciones → institution_categories.key='insurance')
```

**Migración `0033`**: crear `service_cycles`, `ALTER TABLE bills ADD COLUMN cycle_id`, índices, y backfill (ciclo 1 por servicio con vigencia + asignación de sus facturas). `.down` revierte columna/tabla/índices.

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/services/{id}/autos` *(implementado en commit 8d7791c)*

#### Endpoint: `POST /api/services/{id}/renew`

**Request**:
```json
{
  "start_date": "2026-10-20",
  "end_date": "2027-10-19",
  "suggested_amount": 70.00
}
```
(`suggested_amount` opcional; si se omite se conserva el actual.)

**Response 200**:
```json
{
  "service": { "id": 8, "start_date": "2026-10-20", "end_date": "2027-10-19", "suggested_amount": 70.0, "is_insurance": true, "webhook_uuid": "..." },
  "cycle": { "id": 2, "service_id": 8, "sequence": 2, "start_date": "2026-10-20", "end_date": "2027-10-19" }
}
```

**Response Error**:
```json
{ "error": "invalid_request", "message": "start_date debe ser anterior a end_date" }
{ "error": "not_found", "message": "servicio no encontrado" }
```

#### Endpoint: `GET /api/services/{id}/cycles`

**Response 200**:
```json
[
  { "id": 1, "service_id": 8, "sequence": 1, "start_date": "2025-10-20", "end_date": "2026-10-19", "created_at": "..." }
]
```

### 4.5 Dependencias

- **Internas**: `ServiceStorage` (is_insurance), `BillStorage` (cycle_id), `ServiceCycleStorage`/`Service`/`Handlers` (nuevos), `BillsPage` + modales + i18n.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: `GET /api/services/{id}/autos` devuelve autos con datos de póliza; `[]` si vacío. *(Implementado.)*
- [x] CA-002: La pestaña "Pólizas" solo aparece para servicios con `is_insurance === true`. La acción "Renovar" aparece en **todos** los servicios.
- [x] CA-003: Desde la pestaña "Pólizas" se puede **agregar** una póliza (elige auto no asociado + datos) y aparece en la lista tras guardar.
- [x] CA-004: Desde la pestaña "Pólizas" se puede **editar y eliminar** una póliza existente.
- [x] CA-005: `POST /api/services/{id}/renew` crea un ciclo con `sequence = max+1`, actualiza `start_date`/`end_date`/`suggested_amount` del servicio, y conserva `webhook_uuid`.
- [x] CA-006: `renew` funciona para cualquier servicio (seguro o no); `404` si el servicio no existe.
- [x] CA-007: Las facturas creadas después del backfill (auto-gen, scheduler, webhook, manual) quedan con `cycle_id` del ciclo que contiene su período (o el último).
- [x] CA-008: El backfill crea ciclo 1 para servicios con vigencia y asigna sus facturas existentes; los servicios sin vigencia quedan con `cycle_id` NULL.
- [x] CA-009: `GET /api/services/{id}/cycles` lista los ciclos ordenados por `sequence`.
- [x] CA-DARK: Modales y tarjetas usan tokens del tema; legibles en darkmode.

### 5.2 No funcionales

- [x] CA-NF-001: Tagging de ciclo = 1 consulta indexada por `service_id`; sin N+1.
- [x] CA-NF-002: Sin dependencias nuevas.

### 5.3 Testing

- **Unit tests**: `ServiceCycleStorage` (create/sequence/find-for-period/backfill), `BillStorage.Create` tagging, `ServiceCycleService.Renew` (gating, validación de fechas), handler (200/401/not_insurance).
- **Integration tests**: backfill sobre DB de prueba; renew crea ciclo + actualiza servicio + conserva webhook.
- **E2E**: Manual — renovar la Hilux y ver el ciclo nuevo; agregar/editar/eliminar póliza; pestaña oculta en servicios no-seguro.
- **Carga**: Costo de `is_insurance` subquery + tagging irrelevante para el volumen del iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración 0033 (service_cycles + bills.cycle_id + backfill) | 1 h | Ninguna |
| 2 | Backend: modelo ServiceCycle, storage, BillStorage.Create tagging, ServiceStorage.is_insurance | 2 h | Fase 1 |
| 3 | Backend: ServiceCycleService.Renew + handlers + rutas + wiring | 2 h | Fase 2 |
| 4 | Tests backend | 1.5 h | Fase 3 |
| 5 | Frontend: gate is_insurance, AddPolicyModal, RenewServiceModal, pestaña Pólizas full | 3 h | Fase 3 |
| 6 | i18n + build + validación local + pruebas manuales | 1.5 h | Fase 5 |

### 6.2 Milestones

1. **MVP**: is_insurance + pestaña Pólizas con agregar/editar/eliminar + renew (endpoint y modal).
2. **V1.0**: backfill de ciclos, tagging de facturas, GET cycles, badge ciclo (REQ-011).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Backfill mal asociado en servicios sin vigencia | Media | Medio | Backfill solo para servicios con start/end; ciclo NULL para el resto |
| Cambio de `serviceColumns` rompe scan | Media | Alto | Tests de storage (list/get) y revisión del scan |
| El webhook trae facturas de períodos pasados → ciclo equivocado | Media | Bajo | `FindForPeriod` cae al último ciclo como fallback |
| Renovar sin fechas previas (primer ciclo) | Baja | Bajo | El modal permite crear el primer ciclo con prefills por defecto |
| `is_insurance` falso si la institución no tiene categoría | Media | Medio | Pólizas/Renovar se ocultan; el usuario asocia la institución a `insurance` |

## 8. Notas y Referencias

- Vista actual desde el auto: `frontend/src/pages/AutoShowPage.tsx` + `AddInsuranceModal.tsx`.
- Reverse lookup implementado: `internal/storage/auto_service.go` (`ListByService`), pestaña en `BillsPage.tsx` (commit 8d7791c).
- Categorías: `institution_categories.key='insurance'` (usado en `AutoServiceStorage.ListAvailableServices`).
- Política: no modificar esquema de `debts`/`debt_bills` (no afectado).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-24 | opencode | Creación inicial: reverse lookup + pestaña Pólizas (solo lectura) |
| 2026-09-24 | opencode | Ampliación: ciclos de servicio, botón Renovar, agregar/editar/eliminar pólizas desde el servicio, gating por institución de seguro (requerimientos del usuario) |
| 2026-09-24 | opencode | Renovar aplica a todos los servicios (corrección del usuario); CAs validados por el usuario en local → pending_release |