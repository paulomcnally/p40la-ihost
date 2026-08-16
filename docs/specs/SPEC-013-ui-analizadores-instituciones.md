---
title: "Mejora de UI para gestión de analizadores en instituciones"
id: "SPEC-013"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 14
---

# Mejora de UI para gestión de analizadores en instituciones

**ID**: SPEC-013  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

La UI actual para asignar analizadores a una institución usa checkboxes inline en el formulario de institución. Esta experiencia no escala: cuando el número de analizadores crezca, el formulario se volverá largo y difícil de usar.

Esta spec propone una UX superior:
- Botón "Agregar analizador" que abre un modal con búsqueda y toggle por analizador.
- Card de "Analizadores asignados" en la edición de institución, con opción de remover.
- Indicador compacto en el listado de instituciones: "N analizador(es)" con icono de PDF.

**Consideraciones de iHost**: 100% frontend. Sin cambios en backend, DB ni API. Zero impacto en memoria o CPU.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En `InstitutionFormPage`, eliminar los checkboxes inline de analizadores.

2. **REQ-002**: Agregar botón "Agregar analizador" en el formulario de institución (a la izquierda o debajo del campo nombre, según convenga visualmente).

3. **REQ-003**: Al presionar "Agregar analizador", abrir un modal (`AnalyzerPickerModal`) que liste todos los analizadores disponibles (`/api/analyzers`).

4. **REQ-004**: El modal debe incluir un campo de búsqueda que filtre analizadores por nombre (case-insensitive).

5. **REQ-005**: Cada analizador en el modal debe tener un toggle (switch) para activarlo/desactivarlo. Al activarlo, se agrega a la lista de asignados; al desactivarlo, se remueve.

6. **REQ-006**: En modo edición, mostrar un card (o sección) "Analizadores asignados" debajo del campo nombre con los analizadores activos. Cada uno debe tener un botón para removerlo.

7. **REQ-007**: En `InstitutionsPage`, cada card de institución debe mostrar el conteo de analizadores asignados: "1 analizador" o "N analizadores", junto con un icono de PDF.

### 2.2 Requerimientos Funcionales (P1 - Importes)

8. **REQ-008**: El modal debe mostrar primero los analizadores ya asignados (pinned) y luego el resto.

### 2.3 Requerimientos No Funcionales

- **Rendimiento**: Zero deps nuevas. Filtro de búsqueda en memoria.
- **iHost**: Solo cambios en componentes React. Sin impacto en backend.
- **Accesibilidad**: Toggle accesible, focus trap en modal.

## 3. Investigación y Decisiones Técnicas

### 3.1 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Modal con búsqueda + toggle | Escalable, UX moderna, ocupa poco espacio | Requiere componente modal nuevo | ✅ Seleccionada |
| Checkboxes inline (actual) | Simple | No escala, UI pobre con muchos analizadores | ❌ Reemplazada |
| Dropdown multiselect | Compacto | Difícil ver seleccionados, mala UX para remover | ❌ Rechazada |

### 3.2 Decisiones arquitectónicas

**ADR-013-001**: Nuevo componente `AnalyzerPickerModal` reutilizable.
- **Contexto**: Se necesita un selector de analizadores en el formulario de institución.
- **Decisión**: Crear `frontend/src/components/AnalyzerPickerModal.tsx` con búsqueda y toggles, similar al `IconPickerModal` existente.
- **Consecuencias**: Componente reutilizable para futuras mejoras.

**ADR-013-002**: Card de analizadores asignados en el formulario.
- **Contexto**: El usuario necesita ver claramente qué analizadores están activos.
- **Decisión**: Mostrar los asignados como tags/chips en un card debajo del nombre, con botón de remover (X).
- **Consecuencias**: UI limpia y accionable.

## 4. Diseño Técnico

### 4.1 Componentes a modificar/crear

| Archivo | Cambio |
|---------|--------|
| `frontend/src/components/AnalyzerPickerModal.tsx` | Crear modal con búsqueda y toggles |
| `frontend/src/pages/InstitutionFormPage.tsx` | Reemplazar checkboxes por botón + card de asignados |
| `frontend/src/pages/InstitutionsPage.tsx` | Mostrar conteo de analizadores con icono PDF |
| `frontend/src/components/Icons.tsx` | Agregar icono `pdf-document` si no existe (reutilizar `bill`) |

### 4.2 APIs existentes (sin cambios)

- `GET /api/analyzers` → lista de analizadores disponibles
- `PUT /api/institutions/{id}/analyzers` → guardar asignados
- `GET /api/institutions/{id}/analyzers` → obtener asignados

### 4.3 Flujo de interacción

```
[InstitutionFormPage]
  ├── Campo nombre
  ├── Card "Analizadores asignados"
  │     ├── [Analizador 1] [X]
  │     ├── [Analizador 2] [X]
  │     └── Botón "Agregar analizador"
  └── Botones Guardar/Cancelar

[Botón Agregar analizador] → [AnalyzerPickerModal]
  ├── Input búsqueda
  └── Lista de analizadores:
        [Nombre]  [Toggle on/off]
```

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: El formulario de institución ya no muestra checkboxes inline de analizadores.
- [x] CA-002: Existe botón "Agregar analizador" que abre modal.
- [x] CA-003: El modal permite buscar analizadores por nombre.
- [x] CA-004: El toggle en el modal agrega/remueve analizadores de la lista asignada.
- [x] CA-005: Los analizadores asignados se muestran en un card con opción de remover.
- [x] CA-006: Al guardar, los analizadores asignados se persisten correctamente.
- [x] CA-007: En el listado de instituciones se muestra "N analizador(es)" con icono.

### 5.2 No funcionales

- [x] CA-NF-001: Build de Vite sin errores.
- [x] CA-NF-002: Sin dependencias npm nuevas.

## 6. Plan de Implementación

| Fase | Descripción | Estimación |
|------|-------------|------------|
| 1 | Crear `AnalyzerPickerModal` | 30 min |
| 2 | Modificar `InstitutionFormPage` | 30 min |
| 3 | Modificar `InstitutionsPage` | 15 min |
| 4 | Build + tests manuales | 15 min |

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Estado no se sincroniza entre modal y card | Baja | Medio | Usar mismo array de `analyzer_ids` en el componente padre |
| Búsqueda lenta con muchos analizadores | Baja | Bajo | Filtro en memoria con `useMemo` |

## 8. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
| 2026-08-15 | paulomcnally | Implementación: AnalyzerPickerModal, InstitutionFormPage, InstitutionsPage, icono pdf. Build Vite OK |
