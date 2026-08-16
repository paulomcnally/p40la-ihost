---
title: "Sobrescribir bill existente cuando el analizador extrae datos"
id: "SPEC-021"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 21
---

# Sobrescribir bill existente cuando el analizador extrae datos

**ID**: SPEC-021  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

Actualmente, cuando un usuario sube un documento al analizador (Claro, DISNORTE-DISSUR, etc.) y ya existe una factura para ese servicio en el mismo periodo (año/mes), el sistema retorna un error: `"ya existe una factura para este problema"`. Esto bloquea al usuario y le obliga a eliminar manualmente la factura existente antes de poder re-analizar.

El problema es que las facturas pueden ser generadas automáticamente por el `BillingScheduler` con montos estimados (`SuggestedAmount`). Cuando el usuario sube el documento real, el analizador extrae datos verificados (monto exacto, número de factura, fecha de vencimiento). **Los datos del analizador son más confiables que los auto-generados**, por lo que deberían sobrescribir la factura existente en lugar de fallar.

Este cambio afecta solo al flujo del analizador (`DocumentService.CreateBillFromExtracted`). Los flujos de creación manual y auto-generación no se modifican.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Cuando el analizador extrae datos y ya existe una factura para el mismo servicio+periodo, el sistema DEBE actualizar la factura existente con los datos extraídos (monto, número de factura, fecha de vencimiento)
2. **REQ-002**: El endpoint `POST /api/services/{service_id}/bills/from-extracted` debe retornar HTTP 200 (no 409) cuando actualiza una factura existente, indicando que fue una actualización
3. **REQ-003**: Solo los campos provistos por el analizador deben actualizarse: `amount`, `invoice_number`, `due_date`. Los campos `status`, `drive_url` y `service_id` NO deben modificarse

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: El frontend debe mostrar un indicador de que la factura fue actualizada (toast o mensaje)
2. **REQ-005**: El endpoint debe retornar el bill actualizado (no solo un mensaje de éxito) para que el frontend pueda actualizar la UI

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-006**: Ninguno adicional

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Una query UPDATE en lugar de INSERT + DELETE previo. Sin impacto.
- **Seguridad**: Sin cambios
- **Almacenamiento**: Sin cambios (SQLite, misma tabla)
- **Disponibilidad**: Sin cambios
- **iHost**: Sin impacto en recursos

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron las 3 rutas de creación de bills:

| Ruta | Archivo | Comportamiento actual en duplicado |
|------|---------|-------------------------------------|
| Analizador | `document.go:100-121` | Retorna error `"ya existe una factura para este periodo"` |
| Manual | `bill.go:106-107` | Retorna error `"ya existe una factura para ese periodo"` |
| Auto-gen | `billing_scheduler.go:140` | Silently skip (return nil) |

La función clave es `BillStorage.FindByServicePeriod()` en `storage/bill.go:50` que busca por `service_id + year + month` y excluye soft-deleted.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| UPDATE existente en `CreateBillFromExtracted` | Soluciona el problema directamente | Cambia contrato del endpoint | ✅ Seleccionada |
| DELETE + INSERT | Más simple | Pierde `created_at`, rompe referencias | ❌ Rechazada |
| Nuevo endpoint `PUT from-extracted` | REST limpio | Más código, más rutas | ❌ Rechazada |
| Mantener error y guiar al usuario | Sin cambios backend | Mala UX, paso innecesario | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Sobrescribir en `CreateBillFromExtracted` manteniendo el mismo endpoint
- **Contexto**: El analizador trae datos verificados que siempre son más confiables que datos auto-generados o manuales parciales
- **Decisión**: Si `FindByServicePeriod` retorna un bill existente, hacer `UPDATE` en lugar de retornar error
- **Consecuencias**: El endpoint ahora tiene doble propósito (create/update). El response debe indicar cuál acción se realizó.

**ADR-002**: No modificar flujos de creación manual ni auto-generación
- **Contexto**: La creación manual tiene su propia validación y el usuario controla explícitamente los datos. La auto-generación usa montos estimados y ya hace skip silencioso.
- **Decisión**: Solo modificar `DocumentService.CreateBillFromExtracted`
- **Consecuencias**: Consistencia en cada flujo según su naturaleza

## 4. Diseño Técnico

### 4.1 Flujo actual

```
POST /api/services/{id}/bills/from-extracted
  → DocumentService.CreateBillFromExtracted()
    → billStorage.FindByServicePeriod()
    → Si existe: RETURN ERROR 409
    → Si no existe: billStorage.Create()
```

### 4.2 Flujo propuesto

```
POST /api/services/{id}/bills/from-extracted
  → DocumentService.CreateBillFromExtracted()
    → billStorage.FindByServicePeriod()
    → Si existe: billStorage.UpdateFields() con datos extraídos
    → Si no existe: billStorage.Create()
    → RETURN 200 con bill (create o update)
```

### 4.3 Cambios en `internal/storage/bill.go`

Nueva función `UpdateFromExtracted()`:

```go
func (s *BillStorage) UpdateFromExtracted(ctx context.Context, billID int64, amount float64, invoiceNumber string, dueDate *time.Time) error {
    _, err := s.db.ExecContext(ctx, `
        UPDATE bills 
        SET amount = ?, invoice_number = ?, due_date = ?, updated_at = datetime('now')
        WHERE id = ? AND deleted_at IS NULL
    `, amount, invoiceNumber, dueDate, billID)
    return err
}
```

### 4.4 Cambios en `internal/services/document.go`

Modificar `CreateBillFromExtracted()` (línea ~100-121):

**Antes:**
```go
existing, _ := billStorage.FindByServicePeriod(ctx, serviceID, extracted.Year, extracted.Month)
if existing != nil {
    return nil, fmt.Errorf("ya existe una factura para este periodo")
}
// ... create new bill
```

**Después:**
```go
existing, _ := billStorage.FindByServicePeriod(ctx, serviceID, extracted.Year, extracted.Month)
if existing != nil {
    // Actualizar bill existente con datos del analizador
    err := billStorage.UpdateFromExtracted(ctx, existing.ID, extracted.Amount, extracted.InvoiceNumber, extracted.DueDate)
    if err != nil {
        return nil, fmt.Errorf("error actualizando factura: %w", err)
    }
    existing.Amount = extracted.Amount
    existing.InvoiceNumber = extracted.InvoiceNumber
    existing.DueDate = extracted.DueDate
    return existing, nil
}
// ... create new bill (sin cambios)
```

### 4.5 Cambios en response del endpoint

El handler `CreateBillFromExtracted` en `document_handlers.go` ya retorna el bill como JSON. No necesita cambios significativos. Se agrega un campo `updated: true/false` para que el frontend sepa si fue creación o actualización.

**Response nuevo:**
```json
{
  "id": 123,
  "service_id": 5,
  "year": 2026,
  "month": 8,
  "amount": 1500.00,
  "invoice_number": "A123456",
  "status": "pending",
  "updated": true
}
```

### 4.6 Cambios en frontend (`UploadBillModal.tsx`)

En `handleSave()` (línea ~111), el toast de éxito debe diferenciar:
- Si fue creación: "Factura creada correctamente"
- Si fue actualización: "Factura actualizada con datos del analizador"

### 4.7 Dependencias

- **Internas**: `BillStorage`, `DocumentService`, `DocumentHandlers`, `UploadBillModal`
- **Externas**: Ninguna

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un servicio con factura auto-generada para agosto 2026, cuando el usuario sube una factura de Claro para agosto 2026, entonces la factura existente se actualiza con el monto y número de factura reales
- [ ] CA-002: Dado un servicio sin factura para el periodo, cuando el usuario sube una factura, entonces se crea una nueva factura (comportamiento original)
- [ ] CA-003: El endpoint retorna HTTP 200 con el bill actualizado cuando sobrescribe
- [ ] CA-004: El frontend muestra "Factura actualizada con datos del analizador" cuando sobrescribe
- [ ] CA-005: El campo `status` de la factura existente NO se modifica al sobrescribir
- [ ] CA-006: El campo `drive_url` de la factura existente NO se modifica al sobrescribir

### 5.2 No funcionales

- [ ] CA-NF-001: La creación manual de bills sigue retornando error en duplicado (sin cambios)

### 5.3 Testing

- **Unit tests**: Probar `CreateBillFromExtracted` con bill existente (update) y sin bill (create)
- **Integration tests**: Subir documento cuando ya hay factura → verificar actualización
- **Manual**: Subir factura de Claro cuando ya existe una auto-generada para ese mes

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar `UpdateFromExtracted` en `BillStorage` | 10 min | Ninguna |
| 2 | Modificar `CreateBillFromExtracted` en `DocumentService` | 10 min | Fase 1 |
| 3 | Actualizar response del handler con campo `updated` | 5 min | Fase 2 |
| 4 | Actualizar frontend `UploadBillModal` para mostrar toast diferenciado | 5 min | Fase 3 |
| 5 | Verificar en local: subir factura existente + factura nueva | 10 min | Fase 4 |

### 6.2 Milestones

1. **Backend**: Update funcional, endpoint retorna 200 con bill actualizado
2. **Frontend**: Toast diferenciado para create vs update
3. **Verificación**: App corriendo en local para validación manual

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Perder datos al sobrescribir | Baja | Alto | Solo sobrescribir campos del analizador, mantener status/drive_url |
| Breaking API para clientes existentes | Baja | Medio | Response sigue siendo un Bill, se agrega campo opcional `updated` |
| Conflicto con concurrent uploads | Muy baja | Bajo | SQLite WAL maneja concurrencia básica |

## 8. Notas y Referencias

- **Spec relacionada**: SPEC-008 (Sistema de facturación automática), SPEC-009 (Analizadores de documentos)
- **Archivos a modificar**:
  - `internal/storage/bill.go` - Agregar `UpdateFromExtracted()`
  - `internal/services/document.go` - Modificar `CreateBillFromExtracted()`
  - `internal/api/document_handlers.go` - Agregar campo `updated` al response
  - `frontend/src/components/UploadBillModal.tsx` - Toast diferenciado
- **Endpoint afectado**: `POST /api/services/{service_id}/bills/from-extracted`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
