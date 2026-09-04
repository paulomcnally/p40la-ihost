---
title: "Fix iconos invisibles en iOS Safari (iPhone)"
id: "SPEC-057"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 57
---

# Fix iconos invisibles en iOS Safari (iPhone)

**ID**: SPEC-057  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

Al acceder a la aplicación desde un iPhone (iOS Safari) mediante `http://ihost.local:8088` (misma red WiFi del iHost), los iconos de la interfaz no se muestran. El resto de la página carga correctamente, pero todos los iconos (sidebar, cards, menús, modales) aparecen invisibles, lo que degrada seriamente la experiencia de usuario.

La investigación del código confirma que los iconos son **SVG inline renderizados por React** (`frontend/src/components/Icons.tsx`), es decir, no hacen ninguna petición de red. El problema no es de rutas/assets (`/assets/...` funciona, ya que la página carga) sino un **bug de renderizado de WebKit/iOS Safari**: Safari computa `width: 0` para un SVG con `viewBox` pero **sin atributos `width`/`height` explícitos** cuando está dentro de un contenedor `flex`/`inline-flex`. En este código, el componente `Icon` aplica el tamaño (ej. `w-6 h-6`) al `<span>` contenedor y no al `<svg>`, dejando al SVG sin dimensiones definidas → invisible en iOS.

El resultado esperado es que los iconos se rendericen correctamente en iOS Safari/iPhone, manteniendo el comportamiento actual en desktop (Chrome/Firefox) y sin cambios en la API del componente ni en los ~80 call sites.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Los iconos de toda la UI deben verse correctamente en iOS Safari (iPhone/iPad) accediendo a `http://ihost.local:8088`.
2. **REQ-002**: Los iconos deben seguir viéndose igual en desktop (Chrome, Firefox, Safari macOS) — no romper el render actual.
3. **REQ-003**: La solución debe aplicarse de forma centralizada en el componente `Icon` (o CSS global), sin modificar los ~80 call sites que usan `<Icon name="..." className="w-6 h-6" />`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Añadir icono de sitio para iOS Safari (favicon + `apple-touch-icon` 180x180 PNG) para que la pestaña y el acceso desde home screen de iOS muestren un ícono en lugar del globo genérico (mejora de UX complementaria, hoy no existe ningún `<link rel="icon">` en `frontend/index.html`).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-005**: Documentar en el repo (AGENTS.md o comentario en Icons.tsx) la regla de "todo icono SVG inline debe tener `width`/`height` definidos" para evitar regresiones en futuros iconos.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: La solución no debe añadir requests de red ni dependencias nuevas. Costo de render despreciable (clonar un elemento React por icono).
- **Seguridad**: Sin cambios de autenticación ni acceso a datos.
- **Almacenamiento**: Sin impacto (el favicon añade ~pocos KB al bundle estático).
- **Disponibilidad**: Sin impacto en uptime.
- **iHost**: Sin dependencias nuevas, sin aumento relevante de RAM/CPU. Cambio solo en frontend estático (build fuera del iHost).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Mecanismo de iconos verificado en código** (`frontend/src/components/Icons.tsx:644-651`): el componente `Icon` renderiza `<span className={"inline-flex items-center justify-center " + className}>` y dentro el SVG. Los SVGs se crean con `createElement('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, ... })` **sin `width` ni `height`** (`Icons.tsx:3-638`). El tamaño (`w-6 h-6`) se aplica al `<span>`, no al `<svg>`.
- **Sin reglas CSS globales para `svg`**: `frontend/src/index.css` no define `svg { width/height }`. El CSS compilado (`public/assets/index-*.css`) tampoco contiene reglas `svg{...}`. Los iconos dependen exclusivamente del dimensionado por defecto del navegador.
- **Rutas/red descartadas como causa**: los iconos son SVG inline (sin requests). El JS/CSS sí se sirven por rutas absolutas `/assets/...` (`public/index.html:15-16`) y cargan bien (la página funciona). El i18n carga por `fetch('/i18n/${lang}.json')` (`frontend/src/stores/i18nStore.ts:25`) y también funciona.
- **Bug documentado de WebKit**: Safari/iOS Safari computa el ancho de un SVG sin `width`/`height` explícitos dentro de contenedores flex como `0px` (invisible). Fuentes:
  - CSS-Tricks, "6 Common SVG Fails (and How to Fix Them)": *"Excluding `width` or `height` in these cases prevents us from seeing the full image"* (Safari computes `width` as `0px` en flex containers).
  - philipwalton/flexbugs#1: *"Chrome & Firefox do not apply the minimum sizing rule to inline SVG, but Edge & Safari do"*; fix recomendado: *"Always making sure image/SVG elements in a flex container have an explicit size"*.
  - Stack Overflow "SVG invisible on Safari without height attribute": requiere `width`/`height` explícitos.
  - Stack Overflow "Flex + svg behaving strange in ios Safari": contenedor `inline-flex` hace que el SVG tenga size `0`; `width: 100%` lo arregla.
- **La solución estándar** en librerías de iconos (lucide, heroicons, feather): cada SVG lleva `width` y `height` explícitos.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Clonar el SVG en `Icon` e inyectar `className` + `width`/`height` por defecto al `<svg>` | Centralizado (1 archivo, ~10 líneas); API intacta; dimensiones explícitas garantizadas; robusto en WebKit | `cloneElement` por render (costo despreciable) | ✅ **Seleccionada** |
| B. Añadir `width`/`height` a cada uno de los ~100 SVGs de `Icons.tsx` | Muy explícito | Repetitivo, tocamos 100+ definiciones, riesgo de error manual | ❌ Rechazada |
| C. Regla CSS global: `svg { width: 100%; height: 100% }` sobre el wrapper | Mínimo cambio (1 línea CSS) | Depende de que el `<span>` tenga tamaño fijo (lo tiene, pero `w-full h-full` y flex shrink pueden comportarse distinto en WebKit); menos explícito | ❌ Rechazada (queda como fallback) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Dimensiones explícitas en el SVG, de forma centralizada en `Icon`
- **Contexto**: WebKit renderiza a `0px` los SVG sin `width`/`height` en contenedores flex. La causa exacta está en `Icons.tsx:644-651`.
- **Decisión**: En el componente `Icon`, usar `cloneElement` para inyectar el `className` de tamaño y atributos `width="1em" height="1em"` (valores por defecto) directamente sobre el `<svg>`. El `<span>` queda solo como wrapper `inline-flex items-center justify-center`. Los call sites no cambian: las clases Tailwind (`w-6 h-6`, etc.) pasan al SVG y anulan el `1em` por defecto.
- **Consecuencias**: Positivo: un solo punto de cambio, robusto en todos los navegadores, sin tocar call sites. Negativo: leve costo de `cloneElement` por icono (despreciable en React).

**ADR-002**: Añadir favicon + apple-touch-icon para iOS
- **Contexto**: `frontend/index.html` no tiene ningún `<link rel="icon">`; iOS muestra globo genérico en pestaña/home screen.
- **Decisión**: Añadir un favicon SVG y un `apple-touch-icon.png` (180x180) servidos desde la raíz, con `link rel="icon"` y `link rel="apple-touch-icon"`.
- **Consecuencias**: Positivo: mejor UX en iOS. Negativo: dos assets estáticos adicionales (~pocos KB).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Call sites: <Icon name="..." className="w-6 h-6" />]  (sin cambios)
        |
        v
[Icon] -> cloneElement(svg, { className, width: "1em", height: "1em" })
        |
        v
<span class="inline-flex items-center justify-center">   (sin tamaño)
        |-> <svg viewBox="0 0 24 24" class="w-6 h-6" width="1em" height="1em" stroke="currentColor" ...>  (visible en WebKit)
```

### 4.2 Componentes

#### 4.2.1 Componente `Icon` (`frontend/src/components/Icons.tsx`)
- **Responsabilidad**: Renderizar el icono por nombre con dimensiones explícitas.
- **Interfaz**: `Icon({ name: string; className?: string })` — API sin cambios.
- **Dependencias**: `react` (`cloneElement`, `isValidElement`).
- **Ubicación**: `frontend/src/components/Icons.tsx:644-651`.

Cambio propuesto (diseño de referencia, se valida en desarrollo):

```tsx
import { cloneElement, isValidElement, createElement } from 'react'

export function Icon({ name, className = '' }: { name: string; className?: string }) {
  const icon = icons[name] || icons.other
  const svg = isValidElement(icon)
    ? cloneElement(icon as React.ReactElement, { className, width: '1em', height: '1em' })
    : icon
  return (
    <span className="inline-flex items-center justify-center">
      {svg}
    </span>
  )
}
```

Notas:
- El `className` del call site pasa al `<svg>` (así `w-6 h-6` dimensiona el SVG directamente, con dimensión explícita para WebKit).
- `width="1em" height="1em"` garantizan tamaño no-cero incluso si no se pasa className (los SVGs se escalan vía CSS si hay clase; si no, 1em por defecto).
- El `<span>` queda sin tamaño para no interferir con el dimensionado del SVG.

#### 4.2.2 Assets de icono de sitio (`frontend/public/`)
- **Responsabilidad**: Favicon y touch icon para iOS.
- **Archivos**: `frontend/public/favicon.svg`, `frontend/public/apple-touch-icon.png` (180x180).
- **HTML**: `frontend/index.html` → `<link rel="icon" type="image/svg+xml" href="/favicon.svg">` y `<link rel="apple-touch-icon" href="/apple-touch-icon.png">`.
- **Nota**: `frontend/public/` es la fuente de verdad (el build Vite con `emptyOutDir:true` copia su contenido a `public/`). Nunca editar `public/` directamente.

### 4.3 Modelo de datos

Sin cambios de base de datos. No aplica.

### 4.4 APIs / Contratos

Sin cambios de API backend. La UI usa el componente `Icon` con la misma firma.

### 4.5 Dependencias

- **Internas**: `frontend/src/components/Icons.tsx` (único archivo de lógica). Opcional: `frontend/index.html` + assets estáticos para REQ-004.
- **Externas**: Ninguna. No se agregan librerías.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un iPhone/iPad con iOS Safari (o Safari con device emulation), cuando se abre `http://ihost.local:8088`, entonces todos los iconos de sidebar, cards, menús y modales se renderizan visibles y con el tamaño correcto (`w-* h-*`).
- [x] CA-002: Dado desktop (Chrome y Firefox), cuando se abre la misma app, entonces los iconos se ven igual que antes (sin regresión visual).
- [x] CA-003: Dado un call site cualquiera de `<Icon>` (ej. `<Icon name="home" className="w-6 h-6" />`), cuando se renderiza, entonces no hay cambios de API ni errores de tipo/compilación.
- [x] CA-004: Dado un `<Icon>` sin `className`, cuando se renderiza, entonces el icono se ve a tamaño `1em` (no invisible).
- [x] CA-005: Dado iOS Safari, cuando se abre la app o se agrega al home screen, entonces aparece el icono de sitio (favicon/apple-touch-icon) en lugar del globo genérico.

### 5.2 No funcionales

- [x] CA-NF-001: El bundle de frontend no agrega dependencias nuevas ni requests externos (solo 2 assets estáticos locales para el favicon).
- [x] CA-NF-002: El build de Vite (`npm run build` en `frontend/`) compila sin errores y los assets nuevos quedan en `public/`.

### 5.3 Testing

- **Unit tests**: (si existe framework de test en frontend) test del componente `Icon` verificando que el `<svg>` recibe `width`, `height` y `className`.
- **Integration tests**: build de producción y verificación de que `public/index.html` referencia los assets nuevos.
- **E2E tests**: prueba manual en iPhone real (iOS Safari) + emulación de dispositivo iOS en Safari devtools (desktop).
- **Carga/Performance**: sin métricas nuevas; verificar visualmente en el iHost que no hay degradación.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modificar `Icon` en `Icons.tsx` (cloneElement + width/height + className al svg) | 0.5 día | Ninguna |
| 2 | Build de producción (`npm run build` en `frontend/`) y verificación visual desktop | 0.5 día | Fase 1 |
| 3 | Validación en iPhone real (iOS Safari) contra el server local y/o iHost | 0.5 día | Fase 2 |
| 4 | (P1) Favicon + apple-touch-icon + links en `index.html` | 0.5 día | Fase 1 |
| 5 | (P2) Documentar regla de dimensiones en iconos | 0.5 día | Fase 1 |

### 6.2 Milestones

1. **MVP (P0)**: Iconos visibles en iOS Safari con el fix centralizado en `Icon` (Fases 1-3).
2. **V1.0**: + favicon/apple-touch-icon para iOS (Fase 4) y documentación (Fase 5).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| `cloneElement` no inyecte correctamente atributos en algún SVG complejo | Baja | Medio | Verificar con el catálogo completo (`IconPickerModal`) y casos con `w-full h-full` |
| Regresión visual en desktop (tamaños distintos a los actuales) | Media | Medio | Comparar antes/después en Chrome/Firefox; el `1em` por defecto solo aplica sin className |
| Apple rechace `apple-touch-icon` con fondo transparente | Baja | Bajo | Usar PNG 180x180 con fondo sólido según recomendaciones de Apple |
| Caché de iOS Safari muestre iconos viejos | Media | Bajo | Rebuild con nombre de hash nuevo en assets; hard refresh en pruebas |

## 8. Notas y Referencias

- CSS-Tricks — "6 Common SVG Fails (and How to Fix Them)": SVG sin `width`/`height` en flex → width `0px` en Safari. https://css-tricks.com/6-common-svg-fails-and-how-to-fix-them/
- philipwalton/flexbugs — Bug #1 (minimum content sizing): https://github.com/philipwalton/flexbugs#1-minimum-content-sizing-of-flex-items-not-honored
- Stack Overflow — "SVG invisible on Safari without height attribute": https://stackoverflow.com/questions/71022435/svg-invisible-on-safari-without-height-attribute-but-issue-not-recreatable
- Stack Overflow — "Flex + svg behaving strange in ios Safari 14": https://stackoverflow.com/questions/66532071/flex-svg-behaving-strange-in-ios-safari-14-0-3
- Apple — guía de apple-touch-icon (PNG 180x180, fondo opaco): https://favicondl.com/blog/favicon-ios-safari.html
- Código relevante: `frontend/src/components/Icons.tsx:644-651`, `frontend/src/components/Icons.tsx:3-638`, `frontend/index.html:18`, `frontend/src/index.css`, `frontend/src/stores/i18nStore.ts:25`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación: fix iconos invisibles en iOS Safari (iPhone) |
| 2026-09-04 | paulomcnally | Implementación: `cloneElement` con `w-full h-full` al `<svg>` en `Icons.tsx` (fallback 1em), favicon + apple-touch-icon. Build OK y server local verificado |
| 2026-09-04 | paulomcnally | Release: commit de implementación `44e8098`. Confirmado por el usuario |