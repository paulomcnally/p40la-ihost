---
title: "Soporte Responsive para Dispositivos Móviles"
id: "SPEC-011"
status: "released"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 12
---

# Soporte Responsive para Dispositivos Móviles

**ID**: SPEC-011  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15 (released: pruebas manuales satisfactorias en local)

---

## 1. Resumen Ejecutivo

El frontend de p40la-ihost fue diseñado exclusivamente para pantallas de escritorio. Al acceder desde un dispositivo móvil (ej: celular navegando a `ihost.local:8088`), la experiencia es inutilizable: la sidebar fija de 240px ocupa gran parte de la pantalla, los formularios se desbordan, los targets táctiles son demasiado pequeños y el layout no se adapta al viewport.

Esta spec define la implementación de soporte responsive completo usando las utilidades de breakpoints de Tailwind CSS (`sm`, `md`, `lg`, `xl`), priorizando la experiencia en móviles (< 640px) y tablets (640px-1024px), sin afectar la experiencia de escritorio existente.

Consideraciones de iHost: no se agregan dependencias nuevas. Solo se utilizan utilidades de Tailwind CSS (ya configurado) y lógica mínima de React para el toggle del sidebar mobile. Zero impacto en memoria o CPU del backend.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Sidebar debe ocultarse en móvil (< 640px) y mostrarse como drawer overlay con botón de toggle (hamburger menu) en el header.
2. **REQ-002**: Todo el layout debe funcionar correctamente en pantallas de 320px de ancho mínimo (iPhone SE, dispositivos Android pequeños).
3. **REQ-003**: Los targets táctiles (botones, links, inputs) deben tener un tamaño mínimo de 44x44px en móvil (recomendación Apple HIG).
4. **REQ-004**: Los formularios deben ser legibles y operables en móvil: campos full-width, grids de 2 columnas colapsan a 1 columna.
5. **REQ-005**: Los modales (DeleteModal, IconPickerModal) deben ocupar casi todo el ancho en móvil y ser correctamente dismissibles con touch.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

6. **REQ-006**: Las cards de servicios/hogares deben mostrarse en 1 columna en móvil, 2 en tablet, 3+ en desktop (ya parcialmente implementado).
7. **REQ-007**: El IconPickerModal debe adaptar el grid de iconos: 4 columnas en móvil, 6 en desktop.
8. **REQ-008**: El header debe reducir su padding y tamaño de fuente en móvil para maximizar espacio de contenido.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

9. **REQ-009**: Animación suave (slide) para la apertura/cierre del sidebar mobile.
10. **REQ-010**: Soporte para gesture de swipe para abrir/cerrar sidebar en móvil.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Zero impacto en bundle size (solo clases Tailwind ya incluidas). Sin librerías adicionales.
- **iHost**: Sin cambios en backend, DB ni infraestructura. 100% frontend.
- **Accesibilidad**: Mantener soporte de teclado y screen readers. Touch targets >= 44px.
- **Compatibilidad**: iOS Safari 15+, Chrome Android 100+, Samsung Internet.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Tailwind CSS breakpoints**: `sm` (640px), `md` (768px), `lg` (1024px), `xl` (1280px). Mobile-first por defecto (sin prefijo = todos los tamaños).
- **Apple Human Interface Guidelines**: Touch targets mínimo 44x44pt.
- **Google Material Design**: Touch targets mínimo 48x48dp.
- **Estado actual del código**: Se revisaron todos los componentes `.tsx` del frontend. Algunos ya tienen clases responsive parciales (ej: `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3` en HomesPage), pero la mayoría no.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Tailwind responsive utilities (mobile-first) | Ya configurado, zero deps, bundle sin cambios | Requiere editar muchos archivos | ✅ Seleccionada |
| CSS media queries custom | Control total | Duplica lógica, rompe convención Tailwind | ❌ Rechazada |
| Librería de componentes responsive (ej: Headless UI) | Componentes probados | Agrega dependencia, aumenta bundle | ❌ Rechazada |
| Viewport meta tag adjustment | Simple | No resuelve layout, solo escala | ❌ Insuficiente (ya existe) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Sidebar mobile como drawer overlay con estado React
- **Contexto**: La sidebar actual es `fixed w-60` siempre visible. En móvil esto rompe el layout.
- **Decisión**: En pantallas < 640px, la sidebar se oculta por defecto. Un botón hamburger en el header la muestra como overlay con `fixed inset-0 z-50`. Se usa estado React (`useState<boolean>`) para controlar visibilidad. El overlay se cierra al tocar fuera o al navegar.
- **Consecuencias**: Impacto mínimo (1 estado booleano + clases condicionales). Sin librerías externas.

**ADR-002**: Mobile-first approach con Tailwind
- **Contexto**: Tailwind usa mobile-first por defecto.
- **Decisión**: Las clases sin prefijo aplican a móvil. Se usan `sm:`, `md:`, `lg:` para progressively enhance en pantallas más grandes.
- **Consecuencias**: Código más limpio, sigue convención Tailwind. Requiere pensar primero en móvil.

**ADR-003**: Touch targets de 44px mínimo
- **Contexto**: Apple recomienda 44pt mínimo para targets táctiles.
- **Decisión**: Todos los botones interactivos deben tener `min-h-[44px]` o equivalente en móvil. Los botones actuales de `w-8 h-8` (32px) y `w-9 h-9` (36px) deben ampliarse.
- **Consecuencias**: Ligeramente más espacio vertical en móvil, mejor UX táctil.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Mobile viewport < 640px]
  ├── Header: hamburger button → toggle sidebarOpen state
  ├── Sidebar: fixed inset-0, conditional render (sidebarOpen)
  │     └── Overlay backdrop (cierra al tocar)
  ├── Main content: ml-0 (sin margen lateral)
  └── Cards: grid-cols-1

[Tablet viewport 640-1024px]
  ├── Sidebar: fixed w-48 (reducida)
  ├── Main content: ml-48
  └── Cards: grid-cols-2

[Desktop viewport > 1024px]
  ├── Sidebar: fixed w-60 (actual)
  ├── Main content: ml-60 (actual)
  └── Cards: grid-cols-3+
```

### 4.2 Componentes a modificar

#### 4.2.1 DashboardLayout.tsx
- **Responsabilidad**: Layout principal, header, sidebar container
- **Cambios**: 
  - Agregar estado `sidebarOpen` para mobile
  - Header: agregar botón hamburger (visible solo en `sm:hidden`)
  - Sidebar: condicional en mobile, overlay con backdrop
  - Main: `ml-0 sm:ml-48 lg:ml-60`
- **Ubicación**: `frontend/src/components/DashboardLayout.tsx`

#### 4.2.2 Sidebar.tsx
- **Responsabilidad**: Navegación lateral
- **Cambios**:
  - Recibir prop `isOpen` y `onClose` para modo mobile
  - En mobile: `fixed inset-0 w-60` con backdrop
  - En desktop: comportamiento actual (`fixed left-0 top-0 bottom-0 w-60`)
- **Ubicación**: `frontend/src/components/Sidebar.tsx`

#### 4.2.3 ServiceFormPage.tsx y demás formularios
- **Cambios**:
  - `grid-cols-2` → `grid-cols-1 sm:grid-cols-2`
  - Padding: `p-6` → `p-4 sm:p-6`
  - Inputs: asegurar `min-h-[44px]` en mobile
- **Ubicación**: `frontend/src/pages/ServiceFormPage.tsx`, `BillFormPage.tsx`, `HomeFormPage.tsx`, `InstitutionFormPage.tsx`, `CurrencyFormPage.tsx`

#### 4.2.4 IconPickerModal.tsx
- **Cambios**:
  - `grid-cols-6` → `grid-cols-4 sm:grid-cols-6`
  - `max-w-lg` → `max-w-sm sm:max-w-lg`
  - Botones de categorías: `min-h-[44px]` en mobile

#### 4.2.5 DeleteModal.tsx
- **Cambios**:
  - `max-w-sm` → `max-w-sm mx-4` (ya tiene `p-4` en parent)
  - Botones: asegurar `min-h-[44px]`

#### 4.2.6 Pages con cards (HomesPage, ServicesPage, BillsPage, InstitutionsPage)
- **Cambios**:
  - Verificar que todas tengan `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3`
  - Padding de página: `p-5` → `p-3 sm:p-5`

#### 4.2.7 LoginPage.tsx y SetupPage.tsx
- **Cambios**:
  - Verificar que funcionen en pantallas pequeñas
  - Inputs: `min-h-[44px]` en mobile

### 4.3 Breakpoints a utilizar

| Breakpoint | Ancho | Dispositivo típico | Comportamiento |
|------------|-------|-------------------|----------------|
| (default) | < 640px | Celular | Sidebar oculta, 1 columna, touch targets 44px |
| `sm:` | ≥ 640px | Celular grande / tablet pequeña | Sidebar reducida, 2 columnas |
| `md:` | ≥ 768px | Tablet | Sidebar reducida, 2 columnas |
| `lg:` | ≥ 1024px | Desktop | Comportamiento actual completo |

### 4.5 Dependencias

- **Internas**: Ninguna nueva. Solo se modifican componentes existentes.
- **Externas**: Ninguna. Solo Tailwind CSS (ya configurado).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un viewport de 375px (iPhone), cuando cargo la app, entonces veo el contenido completo sin scroll horizontal y con sidebar oculta.
- [ ] CA-002: Dado un viewport de 375px, cuando toco el botón hamburger, entonces la sidebar se muestra como overlay y puedo navegar.
- [ ] CA-003: Dado el sidebar overlay abierto, cuando toco fuera del sidebar, entonces se cierra.
- [ ] CA-004: Dado un viewport de 375px, cuando abro un formulario, entonces todos los campos son full-width y legibles sin zoom.
- [ ] CA-005: Dado un viewport de 375px, cuando toco cualquier botón, entonces el target táctil es al menos 44x44px.
- [ ] CA-006: Dado un viewport de 768px (tablet), cuando cargo la app, entonces veo la sidebar reducida y las cards en 2 columnas.
- [ ] CA-007: Dado un viewport de 1024px+, cuando cargo la app, entonces el layout es idéntico al actual (sin regresiones).
- [ ] CA-008: El IconPickerModal muestra 4 columnas en móvil y 6 en desktop.
- [ ] CA-009: El DeleteModal es usable y dismissible en móvil.

### 5.2 No funcionales

- [ ] CA-NF-001: El bundle size no aumenta más de 1KB (solo clases Tailwind).
- [ ] CA-NF-002: No se agregan dependencias npm nuevas.
- [ ] CA-NF-003: Lighthouse mobile score de performance >= 90.

### 5.3 Testing

- **Manual**: Probar en Chrome DevTools con device emulation (iPhone SE, iPhone 12, iPad, Pixel 5).
- **Manual**: Probar en dispositivo físico (celular del usuario).
- **E2E**: Verificar que no hay scroll horizontal en ningún breakpoint.
- **Carga/Performance**: Verificar que el build de Vite no aumenta significativamente.

### 5.4 Resultados de Pruebas

| Prueba | Resultado | Fecha |
|--------|-----------|-------|
| Build verificado (tsc + vite) | ✅ Pass | 2026-08-15 |
| Server en local (localhost:8088) | ✅ Pass | 2026-08-15 |
| Sidebar drawer mobile (toggle hamburger) | ✅ Pass | 2026-08-15 |
| Formularios responsive (1 columna mobile, 2 desktop) | ✅ Pass | 2026-08-15 |
| Touch targets >= 44px | ✅ Pass | 2026-08-15 |
| IconPickerModal responsive (4 cols mobile, 6 desktop) | ✅ Pass | 2026-08-15 |
| DeleteModal usable en mobile | ✅ Pass | 2026-08-15 |
| Cards responsive (1/2/3 columnas) | ✅ Pass | 2026-08-15 |
| LoginPage/SetupPage mobile | ✅ Pass | 2026-08-15 |
| Sin regresiones en desktop | ✅ Pass | 2026-08-15 |
| Tabla de bills con scroll horizontal | ✅ Pass | 2026-08-15 |

**Conclusión**: Todas las pruebas manuales fueron satisfactorias. La spec se considera completa y lista para release.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | DashboardLayout + Sidebar: drawer mobile con hamburger | 2h | Ninguna |
| 2 | Formularios: colapsar grids, full-width inputs, touch targets | 2h | Fase 1 |
| 3 | Modales: IconPickerModal, DeleteModal responsive | 1h | Fase 1 |
| 4 | Pages con cards: verificar/adaptar grids responsive | 1h | Fase 1 |
| 5 | LoginPage, SetupPage, ajustes finales | 1h | Fase 1 |
| 6 | Testing manual en móvil + desktop (regresiones) | 1h | Fases 1-5 |

### 6.2 Milestones

1. **MVP**: Sidebar drawer + layout básico responsive en DashboardLayout (Fase 1)
2. **V1.0**: Todos los componentes responsive, testing completado (Fases 1-6)

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Regresión en layout desktop | Media | Alto | Testear en desktop después de cada cambio. Usar `lg:` para preservar comportamiento actual. |
| Sidebar drawer con z-index conflicts | Baja | Medio | Usar `z-50` consistente. Verificar que modales (también `z-50`) no se superpongan mal. |
| Touch targets insuficientes en algún componente | Media | Medio | Revisar todos los botones e inputs manualmente. Usar busca de `w-8`, `w-9`, `h-8`, `h-9` como audit. |
| Bundle size crece por clases Tailwind no usadas | Baja | Bajo | Tailwind purga clases no usadas en build. Verificar con `npm run build`. |

## 8. Notas y Referencias

- Tailwind CSS Responsive Design: https://tailwindcss.com/docs/responsive-design
- Apple HIG - Touch Targets: https://developer.apple.com/design/human-interface-guidelines/touch
- Material Design - Touch Targets: https://m3.material.io/foundations/accessible-design/accessibility-basics#4f2b8b1b-1c15-4f0c-9c44-1f4b1d1d1d1d
- Viewport meta tag ya existe en `frontend/index.html`
- Tailwind config en `frontend/tailwind.config.js` (breakpoints por defecto)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
| 2026-08-15 | paulomcnally | Implementación completa: sidebar drawer, formularios, modales, cards, login/setup responsive |
| 2026-08-15 | paulomcnally | Released: pruebas manuales satisfactorias en localhost:8088 |
