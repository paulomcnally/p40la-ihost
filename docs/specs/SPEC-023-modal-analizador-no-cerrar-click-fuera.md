---
title: "Modal del analizador: deshabilitar cierre al hacer click fuera"
id: "SPEC-023"
status: "pending_release"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 23
---

# Modal del analizador: deshabilitar cierre al hacer click fuera

**ID**: SPEC-023  
**Estado**: pending_release  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

Actualmente, los modales del analizador de facturas (`UploadBillModal` y `AnalyzerPickerModal`) se cierran tanto al hacer click fuera del modal como al presionar Escape. El usuario ha solicitado que **solo el botón Cancelar** pueda cerrar el modal.

Esto es importante porque durante el proceso de análisis de facturas, un click accidental fuera del modal puede interrumpir el flujo de trabajo y perder datos. El usuario debe tener control explícito sobre cuándo cancelar la operación.

El cambio es puramente de frontend (React), sin impacto en backend, base de datos o consumo de recursos del iHost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En `UploadBillModal`, el modal NO debe cerrarse al hacer click fuera del área del modal (backdrop click).
2. **REQ-002**: En `AnalyzerPickerModal`, el modal NO debe cerrarse al hacer click fuera del área del modal (backdrop click).
3. **REQ-003**: En ambos modales, el modal NO debe cerrarse al presionar la tecla Escape.
4. **REQ-004**: En ambos modales, SOLO el botón "Cancelar" debe cerrar el modal.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Mantener el backdrop visual (overlay oscuro) para indicar que el modal está activo.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-006**: Mantener el comportamiento de cierre por click fuera en otros modales del sistema (`IconPickerModal`, `DeleteModal`) que no son parte de este cambio.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin cambio. Remoción de event listeners (mejora leve).
- **Seguridad**: Sin cambio.
- **Almacenamiento**: Sin cambio.
- **iHost**: Sin impacto. Cambio mínimo de código frontend.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se identificaron los siguientes modales en el frontend:

| Modal | Archivo | Cierra en backdrop | Cierra en Escape | Cierra en botón |
|-------|---------|-------------------|------------------|-----------------|
| `UploadBillModal` | `frontend/src/components/UploadBillModal.tsx` | Sí (L59-60) | Sí (L50) | Sí (L150) |
| `AnalyzerPickerModal` | `frontend/src/components/AnalyzerPickerModal.tsx` | Sí (L33-34) | Sí (L44) | Sí (L133) |
| `IconPickerModal` | `frontend/src/components/IconPickerModal.tsx` | Sí (L32-33) | Sí (L43) | Sí |
| `DeleteModal` | `frontend/src/components/DeleteModal.tsx` | No | No | Sí (onConfirm/onCancel) |

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Quitar solo backdrop click | Cumple requerimiento del usuario | Escape sigue funcionando | ❌ Incompleta |
| Quitar backdrop click + Escape | Cumple requerimiento completo | User no puede cerrar con teclado | ✅ Seleccionada |
| Agregar prop `closeOnBackdrop` | Reutilizable para otros modales | Complejidad innecesaria para este caso | ❌ Over-engineering |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Deshabilitar cierre por backdrop click y Escape en modales de analizador
- **Contexto**: El usuario accidentalmente cierra el modal durante análisis de facturas
- **Decisión**: Eliminar el handler de `mousedown` en backdrop y el listener de `Escape` en `UploadBillModal` y `AnalyzerPickerModal`
- **Consecuencias**: El usuario solo puede cerrar el modal con el botón Cancelar. Esto es más seguro para procesos destructivos/análisis.

## 4. Diseño Técnico

### 4.1 Cambios por archivo

#### `frontend/src/components/UploadBillModal.tsx`
- Eliminar `useEffect` que detecta click fuera (líneas 56-64)
- Eliminar `useEffect` que detecta Escape (líneas 48-53)
- Mantener backdrop visual (div overlay)
- Mantener botón Cancelar (línea 150)

#### `frontend/src/components/AnalyzerPickerModal.tsx`
- Eliminar `useEffect` que detecta click fuera (líneas 28-38)
- Eliminar `useEffect` que detecta Escape (líneas 42-48)
- Mantener backdrop visual (div overlay)
- Mantener botón Cancelar (línea 133)

### 4.2 No se modifican

- `IconPickerModal.tsx` — mantiene comportamiento actual
- `DeleteModal.tsx` — ya no cierra con backdrop
- Ningún archivo backend

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado que `UploadBillModal` está abierto, cuando el usuario hace click fuera del modal, entonces el modal permanece abierto.
- [ ] CA-002: Dado que `UploadBillModal` está abierto, cuando el usuario presiona Escape, entonces el modal permanece abierto.
- [ ] CA-003: Dado que `UploadBillModal` está abierto, cuando el usuario hace click en "Cancelar", entonces el modal se cierra.
- [ ] CA-004: Dado que `AnalyzerPickerModal` está abierto, cuando el usuario hace click fuera del modal, entonces el modal permanece abierto.
- [ ] CA-005: Dado que `AnalyzerPickerModal` está abierto, cuando el usuario presiona Escape, entonces el modal permanece abierto.
- [ ] CA-006: Dado que `AnalyzerPickerModal` está abierto, cuando el usuario hace click en "Cancelar", entonces el modal se cierra.
- [ ] CA-007: El backdrop visual (overlay oscuro) sigue visible en ambos modales.

### 5.2 No funcionales

- [ ] CA-NF-001: No hay increase en bundle size (solo eliminación de código).

### 5.3 Testing

- **Manual**: Abrir cada modal, click fuera → debe permanecer abierto. Presionar Escape → debe permanecer abierto. Click en Cancelar → debe cerrar.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modificar `UploadBillModal.tsx` | 5 min | Ninguna |
| 2 | Modificar `AnalyzerPickerModal.tsx` | 5 min | Ninguna |
| 3 | Testing manual en local | 5 min | Fase 1-2 |

### 6.2 Milestones

1. **MVP**: Ambos modales solo cierran con botón Cancelar

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Usuario no pueda cerrar modal si no ve botón | Baja | Medio | Botón Cancelar siempre visible |

## 8. Notas y Referencias

- `frontend/src/components/UploadBillModal.tsx` — Modal de subida/análisis de facturas
- `frontend/src/components/AnalyzerPickerModal.tsx` — Modal de selección de analizador
- Relacionado con SPEC-009 (Módulo de Instituciones y Analizadores)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
