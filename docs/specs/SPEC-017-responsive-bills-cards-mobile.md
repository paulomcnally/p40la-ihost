---
title: "Responsive BillsPage: tabla en desktop, cards en móvil"
id: "SPEC-017"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 17
---

# Responsive BillsPage: tabla en desktop, cards en móvil

**ID**: SPEC-017  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

La página de facturas (`BillsPage`) actualmente muestra una tabla HTML en todas las resoluciones. En móvil, esto genera scroll horizontal obligatorio (`min-w-[600px]`) y una experiencia de usuario deficiente: las columnas se comprimen, los textos se truncan y el menú de acciones queda fuera de vista.

Se propone cambiar la presentación a **cards en móvil** y mantener la **tabla en desktop**, similar al patrón ya utilizado en `HomesPage` y otras páginas del proyecto. Esto mejora la legibilidad, la usabilidad táctil y la consistencia de la UI en dispositivos móviles.

Consideraciones de iHost: sin cambios en backend ni base de datos. Es un cambio puramente de frontend (React + Tailwind CSS). No impacta consumo de memoria ni rendimiento del servidor.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En pantallas `< sm` (mobile, <640px), la lista de facturas se muestra como cards apiladas verticalmente.
2. **REQ-002**: En pantallas `>= sm` (desktop), la lista se muestra como tabla (layout actual).
3. **REQ-003**: Cada card en móvil debe mostrar: año, mes, monto (con símbolo de moneda), estado (badge), número de factura (si existe), link a Drive (si existe).
4. **REQ-004**: Cada card en móvil debe incluir el menú de acciones (editar, eliminar) accesible desde un botón de 3 puntos, consistente con el patrón `CardMenu` existente.
5. **REQ-005**: El estado (paid/pending) se muestra como badge con colores consistentes con la tabla desktop.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Las cards deben tener hover effect y sombra consistente con el estilo iOS del proyecto (`rounded-ios`, `shadow-ios`).
2. **REQ-007**: El monto debe ser prominente en la card (font-semibold o similar) para fácil lectura rápida.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-008**: Animación suave al aparecer las cards (fade-in o similar ligero).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Sin impacto. Cambio puramente de presentación CSS.
- **iHost**: Sin cambios en backend, base de datos, ni dependencias. Solo Tailwind CSS (ya en el proyecto).
- **Compatibilidad**: Debe funcionar en iOS Safari, Chrome mobile, y navegadores desktop modernos.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **BillsPage.tsx actual** (línea 90): Usa `<table className="w-full min-w-[600px]">` con `overflow-x-auto` en el contenedor. En mobile provoca scroll horizontal.
- **HomesPage.tsx**: Ya implementa el patrón de cards con `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3`. Es la referencia para el diseño de cards.
- **Patrón del proyecto**: Usar `CardMenu` para acciones, `DeleteModal` para confirmaciones, `Icon` para iconos.
- **Breakpoint de Tailwind**: `sm:` = 640px. Por debajo de esto se considera mobile.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A: CSS media queries en la misma tabla | Sin duplicar JSX | Difícil ocultar/mostrar columnas individualmente, la tabla sigue siendo una tabla en mobile | ❌ Rechazada |
| B: Componente separado `BillCard` | Separación limpia, reusable | Más archivos, más complejidad | ❌ Rechazada (over-engineering) |
| C: Renderizado condicional con `useMediaQuery` o `sm:` de Tailwind | Simple, un solo archivo, patrón existente | Ligeramente más de JSX en BillsPage | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Renderizado condicional inline vs componente separado
- **Contexto**: BillsPage ya es un archivo relativamente pequeño (~155 líneas). Agregar un componente separado para 40-50 líneas de JSX de card parece over-engineering.
- **Decisión**: Hacer renderizado condicional directo en BillsPage usando la clase `hidden sm:block` para la tabla y `sm:hidden` para las cards. Alternativamente, usar un breakpoint check si se necesita lógica JS.
- **Consecuencias**: Manteniene el archivo compacto. Fácil de mantener. Si en el futuro se necesitan más vistas, se puede extraer a componente.

**ADR-002**: Breakpoint de cambio
- **Contexto**: `sm:` de Tailwind = 640px es el breakpoint estándar para mobile/desktop.
- **Decisión**: Usar `sm:` como punto de quiebre. Mobile = <640px, Desktop = >=640px.
- **Consecuencias**: Consistente con el resto del proyecto que ya usa `sm:`.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
BillsPage.tsx
├── [sm:hidden]  → Cards layout (mobile)
│   └── Card por cada bill con CardMenu
└── [sm:block]   → Table layout (desktop, actual sin cambios)
```

### 4.2 Componentes

#### BillsPage.tsx (modificado)
- **Responsabilidad**: Decidir layout según breakpoint y renderizar la lista de facturas.
- **Cambios**: Agregar bloque de cards para mobile, envolver tabla existente en `hidden sm:block`.
- **Sin nuevos componentes**: Se reutiliza `CardMenu`, `Icon`, `DeleteModal` existentes.

### 4.3 Modelo de datos

Sin cambios. Se usa el tipo `Bill` existente:
```
Bill {
  id: number
  service_id: number
  year: number
  month: number
  amount: number
  invoice_number: string | null
  status: 'paid' | 'pending'
  drive_url: string | null
}
```

### 4.4 Estructura de la card móvil

```
┌──────────────────────────────────┐
│ [Mes Año]                  [⋯]  │
│                                  │
│ $1,234.56                        │
│                                  │
│ Factura: #1234  │  ● Pagado     │
│                  │               │
│ [Drive ↗]                        │
└──────────────────────────────────┘
```

### 4.5 Dependencias

- **Internas**: Solo `BillsPage.tsx`. No requiere cambios en otros archivos.
- **Externas**: Ninguna nueva. Solo Tailwind CSS (ya en el proyecto).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: En viewport < 640px, la tabla NO se muestra y en su lugar se ven cards apiladas verticalmente.
- [ ] CA-002: En viewport >= 640px, se muestra la tabla exactamente como está actualmente.
- [ ] CA-003: Cada card muestra: mes/año, monto con símbolo de moneda, badge de estado, número de factura (si existe), link Drive (si existe).
- [ ] CA-004: Cada card tiene un menú de acciones (3 puntos) con opciones Editar y Eliminar, usando `CardMenu`.
- [ ] CA-005: Al eliminar una factura desde la card, se muestra `DeleteModal` de confirmación.
- [ ] CA-006: El estado `paid` se muestra con badge verde, `pending` con badge amarillo (consistente con la tabla desktop).

### 5.2 No funcionales

- [ ] CA-NF-001: Sin cambios en backend, API, ni base de datos.
- [ ] CA-NF-002: Sin nuevas dependencias npm.
- [ ] CA-NF-003: Las cards deben ser táctiles (min-h-[44px] en botones interactivos).

### 5.3 Testing

- **Manual**: Abrir BillsPage en móvil (Chrome DevTools responsive mode) y verificar cards. Abrir en desktop y verificar tabla.
- **Edge cases**: Service sin facturas (EmptyCard), service con muchas facturas (scroll vertical en mobile).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modificar BillsPage.tsx: envolver tabla en `hidden sm:block`, agregar bloque de cards para `sm:hidden` | 30 min | Ninguna |
| 2 | Testing manual en móvil y desktop | 15 min | Fase 1 |

### 6.2 Milestones

1. **MVP**: Cards funcionales en mobile con todas las acciones (editar, eliminar, link Drive).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Cards se ven rotas en tablets (640-768px) | Baja | Medio | Probar en ese rango; usar `sm:` que es 640px, razonable |
| Overflow de texto en cards con datos largos | Media | Bajo | Usar `truncate` o `break-words` en Tailwind |

## 8. Notas y Referencias

- BillsPage actual: `frontend/src/pages/BillsPage.tsx`
- Referencia de cards: `frontend/src/pages/HomesPage.tsx` (línea 50-72)
- Componente CardMenu: `frontend/src/components/CardMenu.tsx`
- Breakpoints Tailwind: https://tailwindcss.com/docs/responsive-design

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-08-15 | p40la-ihost-team | Estado cambiado a pending_execution |
| 2026-08-15 | p40la-ihost-team | Implementación completada, responsive layout listo |
| 2026-08-15 | p40la-ihost-team | Released - verificado en local |
