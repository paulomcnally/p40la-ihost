---
title: "Permitir borrar el campo día de facturación con validación por toast"
id: "SPEC-016"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 16
---

# Permitir borrar el campo día de facturación con validación por toast

**ID**: SPEC-016  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

El campo "día de facturación" (`billing_day`) en el formulario de servicios no permite ser borrado completamente. Cuando el usuario intenta limpiar el campo, el `onChange` del input numérico ejecuta `parseInt(e.target.value) || 1`, lo que convierte cualquier valor vacío/NaN en `1`. Esto genera una experiencia de usuario defensiva e inintuitiva.

El problema afecta tanto al frontend (React) como al backend (Go), ya que el modelo de datos usa `int` en lugar de `*int`, lo que impide representar valores nulos en la UI.

**Resultado esperado**: El usuario puede borrar el campo completamente (quedando vacío). Si el campo es requerido (auto_generate activo + frecuencia mensual) y está vacío al guardar, se muestra un toast de validación. Si no es requerido, se guarda como `null` en la base de datos.

**iHost**: Sin impacto en memoria o rendimiento. Cambio puntual en tipos y validación.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El input de "día de facturación" debe permitir borrar su contenido completamente (valor vacío/null).
2. **REQ-002**: Al guardar el formulario, si `auto_generate` está activo Y `billing_type` es `fixed` Y `billing_day` está vacío, mostrar un toast de error: "El día de facturación es requerido para facturación automática".
3. **REQ-003**: Si las condiciones de REQ-002 no se cumplen (auto_generate apagado, o billing_type variable), guardar `billing_day` como `null` en la base de datos.
4. **REQ-004**: El campo `billing_day` en el modelo Go debe cambiar de `int` a `*int` para soportar nulabilidad.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Al editar un servicio existente con `billing_day: null`, el campo debe mostrarse vacío en el formulario.
2. **REQ-006**: El billing scheduler (`billing_scheduler.go`) debe validar que `billing_day` no sea nil antes de usarlo, y saltar la generación si es nil.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Si `billing_type` es `variable`, ocultar o deshabilitar el campo `billing_day` ya que no aplica.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin cambio.
- **Seguridad**: Sin cambio.
- **Almacenamiento**: La columna `billing_day` en SQLite ya acepta NULL (ver migración 0007). No se requiere migración adicional.
- **iHost**: Sin impacto. Cambio localizado en 4-5 archivos.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Línea problemática** (`ServiceFormPage.tsx:266`): `onChange={(e) => handleChange('billing_day', parseInt(e.target.value) || 1)}`
  - `parseInt("")` → `NaN`
  - `NaN || 1` → `1` (siempre cae en 1)
- **Migración 0007**: `billing_day INTEGER DEFAULT 1 CHECK (billing_day BETWEEN 1 AND 31)` — acepta NULL pero tiene DEFAULT 1.
- **Modelo Go** (`models/service.go:16`): `BillingDay int json:"billing_day"` — no nullable.
- **Scheduler** (`billing_scheduler.go:122`): `targetDay := svc.BillingDay` — si es 0 (default int), el día 0 no existe en el mes, pero el código compara con `now.Day()` que nunca será 0, así que no genera factura. Funciona por accidente pero es frágil.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Cambiar `int` a `*int` en Go + frontend nullable | Solución correcta, soporta null real | Requiere cambiar modelo, storage, handlers, scheduler, frontend types | ✅ Seleccionada |
| Mantener int con valor sentinel (0 o -1) | Menos cambios | Solución hacky, confusa para el usuario, rompe CHECK constraint | ❌ Rechazada |
| Solo frontend: ocultar campo cuando no aplica | Simplifica UI | No resuelve el guardado de null, bug persiste en backend | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Usar `*int` en Go para `billing_day`
- **Contexto**: La columna SQLite acepta NULL, pero el modelo Go no puede representarlo.
- **Decisión**: Cambiar `BillingDay int` a `BillingDay *int` en el modelo, y adaptar storage/handlers/scheduler.
- **Consecuencias**: Se debe verificar nil antes de usar el valor en el scheduler. El JSON serialization de `*int` con `omitempty` no aplica (números null se serializan como `null` en JSON, que es correcto).

## 4. Diseño Técnico

### 4.1 Flujo del cambio

```
Frontend (input vacío)
    │
    ▼
billing_day = null (state)
    │
    ├── auto_generate ON + fixed + empty → Toast error,阻止 submit
    │
    └── auto_generate OFF o variable → POST/PUT con billing_day: null
            │
            ▼
        Backend: BillingDay *int → nil
            │
            ▼
        SQLite: billing_day = NULL
            │
            ▼
        Scheduler: if svc.BillingDay == nil → skip
```

### 4.2 Componentes a modificar

| Archivo | Cambio |
|---------|--------|
| `frontend/src/types/index.ts` | `billing_day: number` → `billing_day: number \| null` |
| `frontend/src/pages/ServiceFormPage.tsx` | Fix onChange, validación con toast, default null |
| `internal/models/service.go` | `BillingDay int` → `BillingDay *int` |
| `internal/storage/service.go` | Scan nullable int, INSERT/UPDATE con nil |
| `internal/api/service_handlers.go` | `BillingDay int` → `BillingDay *int` en request |
| `internal/services/billing_scheduler.go` | Check nil antes de usar `svc.BillingDay` |

### 4.3 Modelo de datos (sin cambios en DB)

```
billing_day INTEGER NULL DEFAULT NULL
  - NULL: sin día asignado (variable o sin auto_generate)
  - 1-31: día específico de facturación
```

### 4.4 Validación frontend

```typescript
// ServiceFormPage.tsx - submit handler
if (formData.auto_generate && formData.billing_type === 'fixed' && !formData.billing_day) {
  showToast(t('services.billing_day_required'), 'error')
  return
}
```

### 4.5 Backend handler (nullable)

```go
type serviceRequest struct {
    // ...
    BillingDay *int `json:"billing_day"`
}
```

### 4.6 Scheduler (nil check)

```go
func (s *BillingScheduler) generateBillForService(...) error {
    // ...
    if svc.Frequency == "monthly" {
        if svc.BillingDay == nil {
            return nil // skip - no billing day configured
        }
        targetDay := *svc.BillingDay
        // ... rest of logic
    }
}
```

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un formulario de servicio, cuando el usuario borra el contenido del campo "día de facturación", entonces el campo queda vacío (no muestra 1).
- [ ] CA-002: Dado un formulario con `auto_generate` activo, `billing_type` fijo, y `billing_day` vacío, cuando el usuario intenta guardar, entonces se muestra un toast de error y no se guarda.
- [ ] CA-003: Dado un formulario con `auto_generate` inactivo, cuando el usuario guarda con `billing_day` vacío, entonces se guarda `null` en la base de datos.
- [ ] CA-004: Dado un servicio existente con `billing_day: null`, cuando el usuario abre el formulario de edición, entonces el campo "día de facturación" aparece vacío.
- [ ] CA-005: Dado un servicio con `billing_day: null` y `auto_generate: true`, cuando el scheduler corre, entonces no intenta generar factura para ese servicio (salta al siguiente).
- [ ] CA-006: Dado un servicio con `billing_day: 15` y `auto_generate: true`, cuando el scheduler corre el día 15, entonces genera la factura correctamente (regresión check).

### 5.2 No funcionales

- [ ] CA-NF-001: La aplicación compila sin errores (Go + TypeScript).
- [ ] CA-NF-002: No se requiere migración de base de datos (la columna ya acepta NULL).

### 5.3 Testing

- **Unit tests**: Validar que el scheduler salta servicios con `BillingDay == nil`.
- **Integration tests**: Crear servicio con `billing_day: null`, verificar que se persiste y se recupera como null.
- **Manuales**: Probar en formulario: borrar campo, guardar con auto_generate on/off, verificar toast, verificar que el valor se persiste correctamente.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Cambiar modelo Go: `BillingDay int` → `*int`, adaptar storage scan/insert/update | 15 min | Ninguna |
| 2 | Cambiar handler request: `BillingDay int` → `*int` | 5 min | Fase 1 |
| 3 | Adaptar scheduler con nil check | 10 min | Fase 1 |
| 4 | Cambiar frontend types: `number` → `number \| null` | 5 min | Ninguna |
| 5 | Fix ServiceFormPage: onChange, default null, validación toast | 20 min | Fase 4 |
| 6 | Build y testing manual completo | 15 min | Fases 1-5 |

### 6.2 Milestones

1. **MVP**: Backend soporta null, frontend permite borrar y valida con toast.
2. **Completado**: Todos los criterios de aceptación pasan, app compila, regresión verificada.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Scheduler genera factura con BillingDay nil y paniquea | Media | Alto | Validación explícita de nil antes de desreferenciar |
| JSON serialization de `*int` genera `null` y el frontend lo maneja mal | Baja | Medio | Probar round-trip completo create → list → edit |
| Servicios existentes con `billing_day = 1` (DEFAULT) ahora se muestran como 1 en UI, confundiendo al usuario | Baja | Bajo | Comportamiento esperado: el valor 1 es válido y se conserva |

## 8. Notas y Referencias

- Migración original: `migrations/0007_add_billing_automation.up.sql`
- Scheduler: `internal/services/billing_scheduler.go`
- Formulario: `frontend/src/pages/ServiceFormPage.tsx:260-268`
- Modelo: `internal/models/service.go:16`
- i18n: Se debe agregar la clave `services.billing_day_required` a los archivos de traducción.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
| 2026-08-15 | paulomcnally | Implementación completa: modelo Go *int, storage nullable scan, scheduler nil check, frontend types nullable, ServiceFormPage fix onChange + toast validation, i18n keys |
| 2026-08-15 | paulomcnally | Released: pruebas manuales satisfactorias |
