---
title: "Fix error 500 en GET /api/autos/{id}/available-services (scan 21 vs 23)"
id: "SPEC-076"
status: "released"
author: "paulomcnally"
created: "2026-09-14"
updated: "2026-09-14"
github_issue: 79
---

# Fix error 500 en GET /api/autos/{id}/available-services (scan 21 vs 23)

**ID**: SPEC-076  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-14  
**Actualizado**: 2026-09-14

---

## 1. Resumen Ejecutivo

El endpoint `GET /api/autos/{id}/available-services` (usado en la página de detalle de un auto para listar seguros disponibles) devuelve un error 500 en el iHost:

```
{"error": "internal_error", "message": "escanear servicio: sql: expected 21 destination arguments in Scan, not 23"}
```

La causa raíz: la query de `ListAvailableServices` en `internal/storage/auto_service.go` nunca se actualizó con las columnas `webhook_uuid` y `last_webhook_request` que SPEC-069 y SPEC-074 agregaron a la tabla `services`. El escáner compartido `scanServices` (en `internal/storage/service.go`) espera 23 columnas (la constante `serviceColumns` incluye ambas), pero la query devolvía 21, rompiendo el `Scan`. El bug solo se manifiesta en producción porque el iHost corre la versión con las columnas nuevas.

La solución es agregar las 2 columnas faltantes a la query, en el mismo orden que `serviceColumns`, para alinear el SELECT con el Scan. No se toca SQLite, el esquema ni el frontend; es un cambio de una query en backend.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Corregir la query de `ListAvailableServices` para que `GET /api/autos/{id}/available-services` responda 200 con la lista de servicios de categoría `insurance` no asociados al auto, incluyendo `webhook_uuid` y `last_webhook_request` en cada servicio.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-002**: Mantener el orden de columnas exacto de `scanServices`/`serviceColumns` para evitar futuros desalineamientos (webhook_uuid y last_webhook_request entre `is_recurring` y `latest_bill_status`).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-003**: Verificar que no existan otras queries de `services` con el mismo desalineamiento (auditoría de SELECTs que alimentan `scanServices`/`scanService`).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto. La query ya existía; solo se agregan 2 columnas al SELECT.
- **Seguridad**: Sin impacto.
- **Almacenamiento**: Sin impacto.
- **Disponibilidad**: Restaura el funcionamiento del endpoint en producción (iHost).
- **iHost**: Sin dependencias nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `scanServices` (service.go:194) y `scanService` (service.go:145) esperan 23 columnas: `id, home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, billing_day, auto_generate, institution_id, institution_analyzer_id, start_date, end_date, is_recurring, webhook_uuid, last_webhook_request, latest_bill_status, deleted_at, created_at, updated_at`.
- La constante `serviceColumns` (service.go:20) incluye `webhook_uuid` (SPEC-069) y `last_webhook_request` (SPEC-074).
- `ListAvailableServices` (auto_service.go:130) seleccionaba solo 21 columnas: le faltaban `s.webhook_uuid` y `s.last_webhook_request`.
- El error `sql: expected 21 destination arguments in Scan, not 23` confirma que el SELECT devuelve menos columnas que el destino del Scan.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Agregar las 2 columnas faltantes a la query de `ListAvailableServices` | Fix mínimo, alinea con `scanServices` y `serviceColumns` | Ninguna | ✅ Seleccionada |
| Crear un escáner separado para esta query | Evita depender de columnas globales | Duplica lógica; el modelo es el mismo `models.Service` | ❌ Rechazada |
| Usar `serviceColumns` (con alias `s.`) en la query | Unifica definición | Requiere refactor mayor con riesgo de romper otras queries | ❌ Rechazada (fuera de alcance del hotfix) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Alinear la query con el escáner compartido
- **Contexto**: `scanServices` es la única forma de escanear `[]models.Service`; toda query que lo use debe devolver exactamente el mismo orden de columnas que `serviceColumns`.
- **Decisión**: Agregar `s.webhook_uuid, s.last_webhook_request` en el orden exacto (después de `s.is_recurring`, antes del subquery `latest_bill_status`).
- **Consecuencias**: El endpoint vuelve a funcionar; la lista de servicios disponibles ahora incluye los campos de webhook como el resto de listados.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
GET /api/autos/{id}/available-services
  → AutoServiceHandlers.ListAvailableServices
  → AutoServiceService.ListAvailableServices
  → AutoServiceStorage.ListAvailableServices (SELECT 23 columnas)
  → scanServices (23 destinos) ✅ alineado
```

### 4.2 Componentes

#### 4.2.1 `internal/storage/auto_service.go` (modificado)
- **Responsabilidad**: Query de servicios disponibles para asociar a un auto.
- **Cambios**: Agregar `s.webhook_uuid, s.last_webhook_request` al SELECT, después de `s.is_recurring` y antes del subquery `latest_bill_status`.
- **Dependencias**: `scanServices` (service.go).

### 4.3 Modelo de datos

Sin cambios en SQLite. La tabla `services` ya tiene `webhook_uuid` (SPEC-069) y `last_webhook_request` (SPEC-074).

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/autos/{id}/available-services`

**Response 200** (antes: 500):
```json
[
  {
    "id": 4,
    "home_id": 2,
    "name": "ASSA",
    "currency_id": 1,
    "frequency": "monthly",
    "suggested_amount": 0,
    "active": true,
    "icon_key": "other",
    "billing_type": "variable",
    "auto_generate": false,
    "institution_id": 3,
    "institution_analyzer_id": 5,
    "is_recurring": false,
    "webhook_uuid": "eba9e18f-4770-4d3b-9b3d-1dd315a31c34",
    "latest_bill_status": "pending",
    "created_at": "2026-08-17T06:20:57Z",
    "updated_at": "2026-09-11T21:07:54Z"
  }
]
```

### 4.5 Dependencias

- **Internas**: `internal/storage/auto_service.go`, `internal/storage/service.go` (scanServices).
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `GET /api/autos/{id}/available-services` responde 200 con la lista de seguros disponibles (antes 500 con `expected 21 destination arguments in Scan, not 23`).
- [ ] CA-002: Cada servicio de la respuesta incluye `webhook_uuid` y `last_webhook_request` (aunque sean null/omitidos).
- [ ] CA-003: Los criterios de filtrado se mantienen: solo servicios de categoría `insurance`, no `deleted`, no asociados al auto.
- [ ] CA-BACK: No aplica (no es página de detalle de UI).

### 5.2 No funcionales

- [ ] CA-NF-001: `go build ./...`, `go vet` y los tests de `internal/storage` pasan sin errores.

### 5.3 Testing

- **Unit tests**: Tests existentes de `internal/storage` pasan.
- **Integration tests**: Servidor local contra copia de la DB real: el endpoint responde 200 con datos.
- **E2E tests**: Página de detalle de auto en el iHost lista seguros disponibles sin error.
- **Carga/Performance**: Sin impacto.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar `s.webhook_uuid, s.last_webhook_request` a la query de `ListAvailableServices` | 0.1 día | Ninguna |
| 2 | Auditoría de otras queries que alimentan `scanServices`/`scanService` | 0.1 día | Fase 1 |
| 3 | Build + tests + prueba local del endpoint contra copia de DB | 0.2 día | Fase 2 |

### 6.2 Milestones

1. **MVP**: Endpoint funciona en local y en iHost.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Otra query con el mismo desalineamiento en otro endpoint | Media | Medio | Auditoría de SELECTs de `services` (REQ-003) antes de cerrar |
| Error de orden de columnas al editar | Baja | Alto | Seguir el orden exacto de `serviceColumns`; validar con build + prueba local |

## 8. Notas y Referencias

- SPEC-069 (webhooks por servicio, agregó `webhook_uuid`).
- SPEC-074 (indicador de tráfico de webhooks, agregó `last_webhook_request`).
- `serviceColumns`: `internal/storage/service.go:20-28`.
- `scanServices`: `internal/storage/service.go:194`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-14 | paulomcnally | Creación inicial de la especificación |
| 2026-09-14 | paulomcnally | Auditoría REQ-003 completada: solo `ListAvailableServices` estaba desalineada; `List`, `GetByID` y `FindByWebhookUUID` usan `serviceColumns` (23 columnas) |
| 2026-09-14 | paulomcnally | Estado `released`. Issue #79 cerrado. Commit de implementación: `50182f1` |