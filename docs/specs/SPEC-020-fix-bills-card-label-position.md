---
title: "Fix posición del label de estado en cards de Bills"
id: "SPEC-020"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 20
---

# Fix posición del label de estado en cards de Bills

**ID**: SPEC-020  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

En la vista mobile de la página de bills (`BillsPage.tsx`), el badge de estado de pago (pendiente/pagado) se superpone con el ícono de menú de 3 puntos (`CardMenu`). Esto ocurre porque ambos elementos compiten por la misma región superior-derecha de la card: el `CardMenu` está posicionado con `absolute top-3 right-3`, y el badge de estado está en el lado derecho del primer `flex justify-between`.

Este es el mismo bug que existía previamente en la lista de servicios, el cual fue corregido en SPEC-018 al mover el badge de estado a la parte inferior derecha de la card. La solución aplicada a services debe replicarse en bills.

El impacto es puramente visual/UI y no afecta funcionalidad, memoria ni almacenamiento del iHost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El badge de estado de pago en las cards de bills NO debe superponerse con el ícono de menú de 3 puntos
2. **REQ-002**: El badge de estado debe ubicarse abajo a la derecha de la card, similar al patrón usado en `HomesPage.tsx` y el fix de services

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-003**: Mantener consistencia visual entre cards de services y bills en mobile

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-004**: Ninguno adicional

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin cambios (solo reordenamiento de elementos DOM existentes)
- **Seguridad**: Sin cambios
- **Almacenamiento**: Sin cambios
- **Disponibilidad**: Sin cambios
- **iHost**: Sin impacto en recursos

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se comparó la estructura de las cards en `BillsPage.tsx` (líneas 92-120) y `ServicesPage.tsx` (líneas 103-138):

**ServicesPage (corregido)**:
- Primera fila: solo ícono a la izquierda, nada a la derecha
- Badge de estado: abajo a la derecha en última fila `flex justify-between`
- CardMenu en `absolute top-3 right-3` no tiene conflicto

**BillsPage (bug actual)**:
- Primera fila: fecha a la izquierda + badge de estado a la derecha (`flex justify-between`)
- Badge de estado en `text-xs font-semibold px-2.5 py-1 rounded-full` compite con CardMenu

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Mover badge a última fila (estilo services) | Consistente, probado, limpio | Requiere reordenar DOM | ✅ Seleccionada |
| Cambiar posición del CardMenu a `bottom-3` | Mínimo cambio | Inconsistente con otros componentes | ❌ Rechazada |
| Agregar `z-index` al badge | Rápido | No resuelve superposición visual | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Replicar patrón de ServicesPage para badge de estado
- **Contexto**: El bug ya fue resuelto en services con un patrón específico
- **Decisión**: Mover el badge de estado de la primera fila a la última fila de la card, posicionado abajo a la derecha
- **Consecuencias**: Consistencia visual entre módulos, código más mantenible

## 4. Diseño Técnico

### 4.1 Estructura actual (BillsPage mobile card)

```
┌─────────────────────────────────┐
│ [fecha]              [badge] ⚠️  │  ← badge superpuesto con CardMenu
│                    [CardMenu]    │
│ [monto]                         │
│ [#factura] [Drive]              │
└─────────────────────────────────┘
```

### 4.2 Estructura objetivo

```
┌─────────────────────────────────┐
│ [fecha]              [CardMenu] │  ← CardMenu sin conflicto
│                                 │
│ [monto]                         │
│ [#factura] [Drive]      [badge] │  ← badge abajo a la derecha
└─────────────────────────────────┘
```

### 4.3 Cambios en `BillsPage.tsx`

Mover el `<span>` del badge de estado desde la primera fila `flex justify-between` hasta una nueva posición al final de la card, usando `flex items-center justify-between` en la última fila existente o creando una nueva fila inferior con el badge alineado a la derecha.

**Antes** (líneas ~95-100):
```tsx
<div className="flex items-center justify-between mb-2">
  <span className="text-sm text-text-secondary">...</span>
  <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${...}`}>
    {t(`bills.status_${bill.status}`)}
  </span>
</div>
```

**Después**:
```tsx
<div className="mb-2">
  <span className="text-sm text-text-secondary">...</span>
</div>
```
Y el badge se mueve a la última fila:
```tsx
<div className="flex items-center justify-between text-sm text-text-secondary">
  <div className="flex items-center gap-4">
    {bill.invoice_number && <span>#{bill.invoice_number}</span>}
    {bill.drive_url && (...)}
  </div>
  <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${...}`}>
    {t(`bills.status_${bill.status}`)}
  </span>
</div>
```

### 4.4 Dependencias

- **Internas**: `CardMenu.tsx` (sin cambios), `BillsPage.tsx` (modificación de layout)
- **Externas**: Ninguna

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: En mobile, el badge de estado de pago NO se superpone con el ícono de 3 puntos
- [ ] CA-002: El badge de estado aparece abajo a la derecha de la card, en la misma fila que #factura y Drive
- [ ] CA-003: El menú de 3 puntos funciona correctamente (tap abre dropdown)
- [ ] CA-004: El layout se ve consistente con el patrón de services

### 5.2 No funcionales

- [ ] CA-NF-001: Sin regresiones en desktop (la tabla no debe afectarse)

### 5.3 Testing

- **Manual**: Verificar en mobile que el badge no se superpone y el menú funciona
- **Visual**: Comparar cards de bills y services para confirmar consistencia

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Mover badge de estado a última fila en BillsPage mobile | 5 min | Ninguna |
| 2 | Verificar en local que no hay superposición | 5 min | Fase 1 |
| 3 | Commit y push | 2 min | Fase 2 |

### 6.2 Milestones

1. **Fix**: Badge reposicionado sin superposición
2. **Verificación**: App corriendo en local para validación manual

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Break layout en desktop | Baja | Medio | Verificar que solo se modifica la rama mobile/cards |
| Inconsistencia con otras pages | Baja | Bajo | Seguir patrón exacto de ServicesPage |

## 8. Notas y Referencias

- **Spec relacionada**: SPEC-018 (Estado de pago dinámico en cards de servicios) - fix aplicado primero en services
- **Archivo a modificar**: `frontend/src/pages/BillsPage.tsx`
- **Componente reference**: `frontend/src/pages/ServicesPage.tsx` (patrón correcto)
- **Componente shared**: `frontend/src/components/CardMenu.tsx`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
