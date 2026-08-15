---
title: "Formulario de facturas: ocultar mes para servicios anuales"
id: "SPEC-007"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 8
---

# Formulario de facturas: ocultar mes para servicios anuales

**ID**: SPEC-007  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

Cuando un servicio tiene frecuencia anual (`yearly`), el formulario de creación/edición de facturas sigue mostrando el selector de mes, lo cual es confuso e innecesario. Un servicio anual se paga una vez por año, no tiene sentido pedir un mes específico.

Este spec propone: para servicios anuales, ocultar el selector de mes en el formulario de facturas y fijar el mes a `0` (o un valor especial) en el backend. La factura anual representa el pago completo del año.

Consideraciones de iHost: cambio puramente de UI + validación mínima en backend. Sin impacto en memoria ni almacenamiento.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En el formulario de facturas (`BillFormPage`), cuando el servicio asociado tiene `frequency: "yearly"`, ocultar el campo de selección de mes
2. **REQ-002**: Para servicios anuales, el mes se fija automáticamente a `0` (valor especial que indica "anual") en el request al backend
3. **REQ-003**: El backend acepta `month: 0` como válido para servicios anuales
4. **REQ-004**: La generación automática de facturas (`generateCurrentBill`) para servicios anuales crea una factura con `month: 0` en lugar de `month: 1`

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: En la lista de facturas, las facturas anuales muestran "Anual" o el año completo en lugar de un mes específico
2. **REQ-006**: En edición de factura anual, el campo mes permanece oculto

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Tooltip o texto informativo que indique "Pago anual" cuando el mes está oculto

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto. Cambio condicional en renderizado.
- **Almacenamiento**: `month: 0` en SQLite para facturas anuales (1 byte adicional por fila, negligible).
- **iHost**: Sin dependencias nuevas.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Frontend actual**: `frontend/src/pages/BillFormPage.tsx` siempre muestra año + mes en grid de 2 columnas (líneas 92-112).
- **Backend actual**: `internal/services/service.go:167-193` — `generateCurrentBill` fuerza `month = 1` para servicios anuales, lo cual es arbitrario y confuso.
- **Modelo**: `internal/models/bill.go` — `Month int` acepta cualquier entero. `0` es válido y no se usa actualmente.
- **Storage**: `internal/storage/bill.go:50-57` — `FindByServicePeriod` busca por `year` y `month`. Con `month: 0` funciona correctamente.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. `month: 0` para anual | Simple, claro, sin schema changes | Requiere validación especial | ✅ Seleccionada |
| B. Campo nullable `month NULL` | Semánticamente correcto | Requiere migración DB, más complejo | ❌ Rechazada |
| C. Tabla separada `annual_bills` | Separación limpia | Overengineering para iHost | ❌ Rechazada |
| D. Mantener `month: 1` pero ocultar UI | Sin cambios backend | Confuso en DB, inconsistente | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: `month: 0` significa factura anual
- **Contexto**: Necesitamos distinguir facturas mensuales de anuales sin cambiar el schema.
- **Decisión**: Usar `month = 0` como convención para facturas anuales.
- **Consecuencias**: Requiere que toda la UI y lógica de negocio manejen `month: 0` correctamente. Documentar esta convención.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[BillFormPage] --(detecta frequency del servicio)--> [oculta/muestra mes]
      |
      v
[API POST /api/bills] --(month: 0 si yearly)--> [BillStorage]
      |
      v
[SQLite: bills.month = 0 para anuales]
```

### 4.2 Componentes

#### 4.2.1 `BillFormPage` (frontend)
- **Cambio**: Cargar el servicio para conocer su `frequency`. Si `yearly`, no renderizar el campo de mes.
- **Ubicación**: `frontend/src/pages/BillFormPage.tsx`

#### 4.2.2 `generateCurrentBill` (backend)
- **Cambio**: Para servicios anuales, usar `month = 0` en lugar de `month = 1`.
- **Ubicación**: `internal/services/service.go:167-193`

#### 4.2.3 Validación de facturas (backend)
- **Cambio**: Aceptar `month: 0` cuando el servicio es anual.
- **Ubicación**: `internal/services/bill.go` (si existe) o en el handler

### 4.3 Modelo de datos

Sin cambios en el schema. Convención: `month = 0` → factura anual.

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/bills`

**Request** (servicio anual):
```json
{
  "service_id": 1,
  "year": 2026,
  "month": 0,
  "amount": 240.00,
  "status": "pending"
}
```

**Request** (servicio mensual, sin cambios):
```json
{
  "service_id": 2,
  "year": 2026,
  "month": 8,
  "amount": 20.00,
  "status": "pending"
}
```

### 4.5 Dependencias

- **Internas**: `BillFormPage.tsx`, `service.go` (generateCurrentBill), bill handlers
- **Externas**: Ninguna

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un servicio con `frequency: "yearly"`, cuando abro el formulario de nueva factura, entonces el campo de mes NO se muestra
- [ ] CA-002: Dado un servicio con `frequency: "monthly"`, cuando abro el formulario de nueva factura, entonces el campo de mes SÍ se muestra
- [ ] CA-003: Al crear una factura para servicio anual, el request envía `month: 0`
- [ ] CA-004: El backend acepta y almacena facturas con `month: 0`
- [ ] CA-005**: La generación automática (`generateCurrentBill`) crea facturas anuales con `month: 0`
- [ ] CA-006**: Al editar una factura anual existente, el campo mes permanece oculto

### 5.2 No funcionales

- [ ] CA-NF-001: Sin migraciones de base de datos requeridas
- [ ] CA-NF-002: Sin dependencias nuevas de npm o Go

### 5.3 Testing

- **Unit tests**: Validación de facturas con `month: 0` para servicios anuales
- **Integration tests**: Crear factura anual y verificar `month: 0` en DB

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: `generateCurrentBill` usa `month: 0` para anuales | 0.25 días | Ninguna |
| 2 | Backend: validar `month: 0` para servicios anuales | 0.25 días | Fase 1 |
| 3 | Frontend: BillFormPage oculta mes para servicios anuales | 0.5 días | Fase 2 |
| 4 | Frontend: BillsPage muestra "Anual" para facturas con month=0 | 0.25 días | Fase 3 |
| 5 | Tests y validación | 0.25 días | Fase 4 |

### 6.2 Milestones

1. **MVP**: Backend acepta `month: 0`, frontend oculta mes (Fase 1-3)
2. **V1.0**: UI muestra "Anual" correctamente (Fase 4-5)

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Facturas existentes con month=1 para anuales se confunden | Baja | Medio | No migrar datos existentes; solo afecta nuevas facturas |
| Queries que asumen month 1-12 fallan con month=0 | Media | Alto | Revisar todas las queries que filtran por month |
| i18n no tiene traducción para "Anual" | Baja | Bajo | Agregar clave `bills.annual` al archivo i18n |
| Validación de mes ocurre antes de verificar frecuencia del servicio | **Alta** | **Alto** | **Lección aprendida**: La validación de `month` en `bill.go` debe consultar la frecuencia del servicio ANTES de rechazar `month: 0`. El orden correcto es: 1) validar campos básicos, 2) cargar servicio, 3) validar mes según frecuencia. |
| CHECK constraint de SQLite bloquea month=0 | **Alta** | **Alto** | **Lección aprendida**: Las restricciones CHECK en SQLite son hardcodeadas en el schema. Al permitir nuevos valores (como `month: 0`), siempre verificar y migrar el schema. Se creó migración `0006_relax_month_constraint.up.sql` para cambiar `CHECK (month BETWEEN 1 AND 12)` a `CHECK (month BETWEEN 0 AND 12)`. |
| Errores de validación silenciosos (sin toast) | Alta | Alto | **Lección aprendida**: Todos los formularios deben mostrar toasts con errores de API. Los errores silenciosos dificultan la depuración y frustran al usuario. |

## 8. Notas y Referencias

- Screenshot del problema: formulario muestra "Mes: Agosto" para servicio anual
- `generateCurrentBill` actual fuerza `month = 1` para anuales (arbitrario)
- **Bug encontrado**: `internal/services/bill.go:71` validaba `month < 1` antes de cargar el servicio, rechazando `month: 0` para anuales. Fix: mover la carga del servicio antes de la validación de mes.
- **Lección crítica**: Los errores de API deben mostrarse siempre como toasts visibles. No confiar en que el usuario revise la consola de red.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-15 | p40la-ihost-team | Fix: validación de `month` en `bill.go` ahora verifica frecuencia del servicio antes de rechazar `month: 0`. Agregados toasts de error en formularios. |
