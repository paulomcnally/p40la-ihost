---
title: "Loading states en listas de casas y servicios"
id: "SPEC-022"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 22
---

# Loading states en listas de casas y servicios

**ID**: SPEC-022  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

Las páginas de listado (HomesPage, ServicesPage, BillsPage) no tienen un estado de carga. Cuando el usuario navega a estas páginas, se muestra inmediatamente la UI de "no hay registros" antes de que los datos lleguen del backend. Esto genera una experiencia de usuario confusa y percibida como lenta.

El problema raíz es que se evalúa `items.length === 0` antes de que la llamada API complete, mostrando el componente `EmptyCard` incorrectamente.

**Resultado esperado**: Mostrar un spinner/skeleton mientras se cargan los datos. Solo mostrar "no hay registros" si la carga completa retorna una lista vacía.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear un componente `LoadingSpinner` reutilizable (ligero, sin dependencias externas)
2. **REQ-002**: Agregar estado `loading` a HomesPage, ServicesPage y BillsPage
3. **REQ-003**: Mientras `loading === true`, mostrar spinner en lugar de EmptyCard o lista
4. **REQ-004**: Solo mostrar EmptyCard cuando `loading === false` y `items.length === 0`

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-005**: Mantener consistencia visual con el estilo iOS existente (colores primary, bordes redondeados)
2. **REQ-006**: El spinner no debe causar reflow significativo (mantener dimensiones fijas)

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Considerar skeleton screens como alternativa al spinner (muestra la estructura de las cards)

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El componente loading no debe agregar dependencias ni aumentar bundle significativamente
- **iHost**: Consumo de RAM mínimo, sin animaciones pesadas
- **Consistencia**: Mismo patrón en todas las páginas de listado

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

**Páginas afectadas**:
- `frontend/src/pages/HomesPage.tsx:28` — `if (homes.length === 0)` se evalúa sin loading
- `frontend/src/pages/ServicesPage.tsx:41` — `if (homes.length === 0)` y `:82` — `if (services.length === 0)`
- `frontend/src/pages/BillsPage.tsx` — Misma situación

**Store used**: `useAppStore` en `frontend/src/stores/appStore.ts` maneja `homes`, `currencies`, `loadHomes()`, `loadAll()`

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A) LoadingSpinner simple con CSS | Ultra ligero, stdlib only | Menos informativo visualmente | ✅ Seleccionada |
| B) Skeleton screens | Mejor UX, muestra estructura | Más código, más complejo | ❌ Deseable para futuro |
| C) Skeleton + Spinner hybrid | Mejor experiencia | Complejidad innecesaria ahora | ❌ Over-engineering |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Usar spinner simple con CSS puro
- **Contexto**: iHost tiene recursos limitados. Necesitamos ligereza extrema.
- **Decisión**: Spinner con CSS animation, sin librerías externas. Un solo componente reutilizable.
- **Consecuencias**: MVP rápido, mejora perceptible inmediata. Skeleton como futuro P2.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Page Component] 
    → useState(loading: true)
    → useEffect: load data → set loading false
    → render: loading ? <LoadingSpinner /> : (items.length === 0 ? <EmptyCard /> : <List />)
```

### 4.2 Componentes

#### 4.2.1 LoadingSpinner (nuevo)
- **Responsabilidad**: Mostrar indicador de carga centrado
- **Interfaz**: Props opcionales `size`, `text`
- **Dependencias**: Ninguna (CSS puro)
- **Ubicación**: `frontend/src/components/LoadingSpinner.tsx`

### 4.3 Modelo de datos

No aplica — solo cambio de UI.

### 4.4 Patrón de código

```tsx
// Antes (problemático)
const [data, setData] = useState([])
useEffect(() => { api.list().then(setData) }, [])

if (data.length === 0) return <EmptyCard /> // ← Muestra vacío antes de cargar

// Después (correcto)
const [data, setData] = useState([])
const [loading, setLoading] = useState(true)
useEffect(() => { 
  api.list().then(setData).finally(() => setLoading(false)) 
}, [])

if (loading) return <LoadingSpinner />
if (data.length === 0) return <EmptyCard />
return <List />
```

### 4.5 Dependencias

- **Internas**: Ninguna — solo modificación de páginas existentes
- **Externas**: Ninguna — CSS puro

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Al navegar a HomesPage, se muestra spinner mientras cargan las casas
- [ ] CA-002: Al navegar a ServicesPage, se muestra spinner mientras cargan servicios
- [ ] CA-003: Al navegar a BillsPage, se muestra spinner mientras cargan facturas
- [ ] CA-004: Cuando no hay registros Y la carga completó, se muestra EmptyCard correctamente
- [ ] CA-005: El componente LoadingSpinner se puede reutilizar desde cualquier página

### 5.2 No funcionales

- [ ] CA-NF-001: No se agregan dependencias externas para el spinner
- [ ] CA-NF-002: El bundle no aumenta más de 2KB (solo el componente + CSS)

### 5.3 Testing

- **Manual**: Navegar a cada lista, verificar que aparece spinner antes de los datos
- **Edge case**: Conexión lenta — verificar que el spinner se muestra el tiempo suficiente

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Crear componente LoadingSpinner | 15 min | Ninguna |
| 2 | Actualizar HomesPage con loading state | 15 min | Fase 1 |
| 3 | Actualizar ServicesPage con loading state | 15 min | Fase 1 |
| 4 | Actualizar BillsPage con loading state | 15 min | Fase 1 |
| 5 | Testing manual completo | 15 min | Fases 2-4 |

**Total estimado**: ~1.5 horas

### 6.2 Milestones

1. **MVP**: Loading en las 3 páginas principales
2. **Futuro**: Skeleton screens como mejora P2

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Spinner parpadea si la carga es muy rápida | Alta | Bajo | Agregar delay mínimo o transición fade |
| Página no considerada no tiene loading | Media | Medio | Revisar todas las páginas con listados |

## 8. Notas y Referencias

- Patrón de loading: estándar React para async data
- Componente existente similar: `DeleteModal` ya usa overlay centrado
- CSS del proyecto: Tailwind CSS (ver `frontend/tailwind.config.js`)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |
