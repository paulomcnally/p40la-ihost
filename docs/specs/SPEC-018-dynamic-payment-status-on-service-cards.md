---
title: "Estado de pago dinámico en cards de servicios"
id: "SPEC-018"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-16"
github_issue: 18
---

# Estado de pago dinámico en cards de servicios

**ID**: SPEC-018  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-16

---

## 1. Resumen Ejecutivo

Las cards de servicios actualmente muestran un label "Pagada" en verde de forma estática, basado en el campo booleano `Service.Active`. Este campo es un toggle de activación del servicio, no refleja el estado real de facturación. Esto genera confusión: un servicio puede estar "activo" (mostrando "Pagada") pero tener todas sus facturas pendientes de pago.

El objetivo es que el label en la card muestre el estado de pago real de la factura más reciente del servicio. Si no hay facturas para el servicio, mostrar "Sin facturas". La factura más relevante se determina por el año/mes más reciente, en orden descendente desde el mes actual hacia atrás.

**Restricción iHost**: La query debe ser ligera dado que se ejecuta en el `ListServices` que ya corre `ReconcileBills`. No se deben agregar queries N+1.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El label en la card de servicio debe reflejar el estado de la factura más reciente (según year/month más alto), no el campo `Service.Active`.
2. **REQ-002**: Estados posibles del label: "Pagada" (verde), "Pendiente" (amarillo), "Sin facturas" (gris).
3. **REQ-003**: La factura más reciente se determina por `year DESC, month DESC` (mes actual hacia atrás).
4. **REQ-004**: El campo `Service.Active` sigue existiendo como toggle de activación, pero ya NO determina el label de pago.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: El endpoint `GET /api/services` debe incluir un campo `latest_bill_status` con el estado de la factura más reciente del servicio (`"paid"`, `"pending"`, o `null` si no hay facturas).
2. **REQ-006**: El frontend debe usar `latest_bill_status` en vez de `active` para renderizar el label.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Mostrar el período (mes/año) de la factura más reciente en el label como tooltip o texto secundario.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: La query de facturas más recientes debe ser una sola query con subquery o JOIN, no N queries por servicio. Impacto mínimo en `ListServices`.
- **Almacenamiento**: Sin cambios en esquema SQLite.
- **iHost**: Sin dependencias nuevas. Query SQLite ligera con índice existente.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Estado actual**: `ServicesPage.tsx:118-122` renderiza el label basándose en `svc.active`.
- **Backend**: `ListServices` en `service_handlers.go:71` ejecuta `ReconcileBills()` y devuelve servicios con campos básicos.
- **Bill model**: Tiene campo `status` (`"pending"` o `"paid"`) y `year`/`month` para período.
- **No existe** un endpoint que devuelva el estado de la factura más reciente por servicio.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Agregar campo `latest_bill_status` en el response de `ListServices` | Simple, sin cambio de schema, una sola query | Añade lógica en backend | ✅ Seleccionada |
| B. Hacer un endpoint separado por servicio | Reutilizable | N+1 queries, más complejo | ❌ Rechazada |
| C. Cargar todas las facturas en el frontend y calcular allí | Sin cambio backend | Mucho data transfer, lógica duplicada | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Enriquecer el response de `ListServices` con estado de factura más reciente
- **Contexto**: El label de la card necesita el estado de la factura más reciente, pero no queremos N+1 queries.
- **Decisión**: Agregar una subquery en `ListServices` que traiga `bill.status` de la factura con `year DESC, month DESC` por cada servicio. Devolver como campo `latest_bill_status` en el JSON.
- **Consecuencias**: Response de `ListServices` crece ligeramente. La query se ejecuta una sola vez. Sin cambios en la UI de otros módulos.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[GET /api/services]
       |
       v
[ListServices handler] --> [ReconcileBills] --> [SQLite: subquery bills]
       |
       v
[Response JSON + latest_bill_status]
       |
       v
[ServicesPage.tsx: renderiza label según latest_bill_status]
```

### 4.2 Componentes

#### 4.2.1 Backend: `ListServices`
- **Responsabilidad**: Devolver servicios con el estado de su factura más reciente
- **Cambio**: Agregar subquery al SELECT existente
- **Ubicación**: `internal/api/service_handlers.go` + `internal/services/service.go`

#### 4.2.2 Frontend: Service Card
- **Responsabilidad**: Renderizar label basado en `latest_bill_status`
- **Cambio**: Reemplazar `svc.active` por `svc.latest_bill_status` en el label
- **Ubicación**: `frontend/src/pages/ServicesPage.tsx`

### 4.3 Modelo de datos

Sin cambios en el esquema. Se agrega campo virtual al response:

```
Service response (extendido):
- ...campos existentes...
- latest_bill_status: string | null  ("paid", "pending", o null)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/services` (modificado)

**Response 200** (cambio):
```json
{
  "services": [
    {
      "id": 1,
      "name": "Internet Fibra",
      "active": true,
      "latest_bill_status": "paid",
      ...
    }
  ]
}
```

### 4.5 Dependencias

- **Internas**: `ListServices` (handler + service), `ServicesPage.tsx`
- **Externas**: Ninguna

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un servicio con factura del mes actual en estado "paid", la card muestra "Pagada" en verde.
- [x] CA-002: Dado un servicio con factura del mes actual en estado "pending", la card muestra "Pendiente" en amarillo.
- [x] CA-003: Dado un servicio sin facturas, la card muestra "Sin facturas" en gris.
- [x] CA-004: Dado un servicio con factura de un mes pasado pero ninguna del mes actual, se muestra el estado de la factura más reciente (mes pasado).
- [x] CA-005: El toggle "Activo" en el formulario de servicio sigue funcionando independientemente del label de pago.
- [x] CA-006: La respuesta de `GET /api/services` incluye el campo `latest_bill_status`.

### 5.2 No funcionales

- [x] CA-NF-001: `ListServices` con 50 servicios tarda menos de 200ms con la subquery agregada.

### 5.3 Testing

- **Unit tests**: Validar que la subquery retorna el bill correcto (más reciente por year/month).
- **Integration tests**: Verificar que `ListServices` devuelve `latest_bill_status` correctamente.
- **E2E tests**: Verificar que la card muestra el label correcto según el estado de la factura.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modificar `ListServices` para incluir subquery de factura más reciente | 30 min | Ninguna |
| 2 | Actualizar tipo TypeScript `Service` con `latest_bill_status` | 10 min | Fase 1 |
| 3 | Modificar `ServicesPage.tsx` para usar `latest_bill_status` | 20 min | Fase 2 |
| 4 | Agregar traducción "Sin facturas" en i18n | 5 min | Fase 3 |
| 5 | Pruebas locales y validación | 15 min | Fase 4 |

### 6.2 Milestones

1. **MVP**: Backend devuelve `latest_bill_status`, frontend lo muestra (Fases 1-4)
2. **V1.1**: Tooltip con período de la factura (Fase P2, opcional)

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Subquery afecta performance en iHost con muchos servicios | Baja | Medio | Usar índice en bills(service_id, year, month). Monitorear en iHost. |
| Servicios sin facturas muestran "null" en vez de "Sin facturas" | Media | Bajo | Manejar null en frontend con valor por defecto |

## 8. Notas y Referencias

- Archivo actual de cards: `frontend/src/pages/ServicesPage.tsx:118-122`
- Handler ListServices: `internal/api/service_handlers.go:71`
- Service model: `internal/models/service.go`
- Bill model: `internal/models/bill.go`
- i18n: `frontend/src/i18n/es.json`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
| 2026-08-16 | paulomcnally | Implementación completa. Subquery con filtro de facturas reales. Label movido a bottom-right. |
| 2026-08-16 | paulomcnally | Release: verificado en iHost. Commit `4d1a554`, versión v0.4.7. Issue #18 cerrado. |
