---
title: "Fix zoom móvil global (todas las páginas)"
id: "SPEC-068"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-11"
updated: "2026-09-11"
github_issue: 70
---

# Fix zoom móvil global (todas las páginas)

**ID**: SPEC-068  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-11  
**Actualizado**: 2026-09-11

---

## 1. Resumen Ejecutivo

En el móvil, varias páginas aplican un zoom automático no deseado al cargar o al interactuar con ellas. El usuario reporta que la página **Autos** queda "cortada" a un lado (probablemente por la data de las cards de seguros, que fuerza overflow horizontal) y que en la sección del **calendario de Deudas**, al tocar algunos días, se genera un "efecto de zoom". Sin embargo, el usuario pidió explícitamente que el fix aplique a **TODAS las páginas**, no solo a esas dos. La experiencia es mala en general en el dispositivo real.

La causa raíz es doble. Por un lado, cuando una página tiene **contenido más ancho que el viewport** (overflow horizontal), iOS Safari ajusta automáticamente el factor de zoom del layout para "acomodar" todo el ancho, lo que hace que la página se vea con zoom y que los elementos queden cortados. Por otro lado, iOS Safari **auto-hace zoom al tocar inputs con `font-size` < 16px** (regla conocida: cualquier `input`/`select`/`textarea` con menos de 16px dispara zoom al enfocarlo). En la página Autos, los números de póliza/certificado/aseguradora largos y el bloque derecho de montos/fechas de las cards de seguros (con `flex-shrink-0`) pueden empujar el ancho del layout más allá de la pantalla. El calendario de Deudas dispara zoom al tocar días por el doble-tap zoom de iOS.

Esta spec corrige el problema en el frontend de forma **global**: eliminar el overflow horizontal en todas las páginas y cards (con especial foco en Autos/segurados y Deudas), garantizar que todos los inputs tengan `font-size >= 16px`, y aplicar `touch-action: manipulation` en los elementos interactivos para prevenir el doble-tap zoom. Es puramente de UI/CSS (React + Tailwind), sin cambios de backend, DB ni infraestructura, por lo que el impacto en iHost es nulo a nivel de recursos.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Aplicar el fix de forma **global a TODAS las páginas del frontend**. Ninguna página, card, modal ni tabla debe producir overflow horizontal en móvil: el contenido debe caber 100% en el ancho del viewport sin recorte lateral ni scroll horizontal.
2. **REQ-002**: Eliminar el overflow horizontal en la página **Autos** (lista) y en la página **detalle de Auto** (`/autos/:id`), causado por las cards de seguros: número de póliza, certificado, aseguradora, montos y fechas largas.
3. **REQ-003**: Eliminar el "efecto de zoom" al tocar días en el **calendario de Deudas** (componente `DebtCalendar`). Tocar cualquier día (con o sin cuotas) no debe cambiar el zoom de la página ni desplazar el layout.
4. **REQ-004**: Garantizar que todos los `input`, `select` y `textarea` del frontend tengan un `font-size` computado de **al menos 16px** en móvil, para evitar el auto-zoom de iOS Safari al enfocarlos.
5. **REQ-005**: Aplicar `touch-action: manipulation` a los elementos interactivos (botones, día del calendario, cards clicables) para prevenir el doble-tap zoom y el delay de tap en iOS.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Auditar **todas** las páginas, componentes y modales que usan `text-sm` (14px) en inputs/selects/textareas y corregirlos para que no disparen zoom en iOS. Incluye (sin limitarse a): `IconPickerModal`, `HousePickerModal`, `AnalyzerPickerModal`, `InstitutionCategoriesModal`, `AddInsuranceModal`, formularios de creación/edición, y cualquier otro con `text-sm`.
2. **REQ-007**: Verificar que las páginas de listado (Autos, Deudas, Servicios, Homes, Instituciones, Pension, Hijos, Categorías, Salarios, Notificaciones) y sus detalles no presenten overflow horizontal en pantallas de 320px, 360px y 414px.
3. **REQ-008**: Verificar que los modales (subida de facturas, agregar seguro, selector de casas, picker de iconos, picker de analizadores, categorías de instituciones, emails, pagar factura) no desborden en móvil.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Agregar `-webkit-text-size-adjust: 100%` a nivel global (`html`) para prevenir el auto-ajuste de tamaño de texto de iOS en caso de overflow residual.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Los cambios son solo CSS/atributos de clase; sin impacto medible en RAM ni CPU del iHost.
- **Seguridad**: Sin cambios de autenticación ni datos sensibles.
- **Almacenamiento**: Sin cambios en DB.
- **Disponibilidad**: Sin impacto en endpoints ni health check.
- **iHost**: Sin dependencias nuevas; solo código frontend estático ya servido.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Overflow horizontal → zoom automático de iOS Safari**: WebKit ajusta el factor de zoom cuando el contenido se layout-ea más ancho que el viewport aunque exista `width=device-width` (webkit.org/blog/7367). Cuando el contenido es más ancho que la pantalla, Safari fuerza un "zoom out" inicial o permite scroll lateral, generando la sensación de zoom y de pantalla cortada.
- **Auto-zoom por `font-size < 16px` en inputs**: Documentado ampliamente (css-tricks.com/16px-or-larger-text-prevents-ios-form-zoom, weblog.west-wind.com). iOS Safari hace zoom automático al enfocar inputs cuyo tamaño de fuente es menor a 16px.
- **Doble-tap zoom**: iOS Safari permite el zoom por doble-tap en elementos interactivos. La propiedad `touch-action: manipulation` elimina el doble-tap zoom manteniendo panning y pinch (webkit.org/blog/5610).
- **Código auditado**: `AutoShowPage.tsx` (cards de seguros con `flex-shrink-0`, fechas `start → end`, badges con `flex-wrap`), `AutosPage.tsx` (cards de lista), `DebtCalendar.tsx` (grid `grid-cols-7`, botones de día con `min-h-[44px]` y `text-sm`), `index.css` (regla global de inputs sin `font-size` explícito), `DashboardLayout.tsx` (layout general), modales con inputs `text-sm`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Agregar `overflow-x: hidden` global en `body` | Simple, oculta el recorte | No arregla la causa; puede ocultar contenido inaccesible (pólizas/valores cortados) | ❌ Rechazada |
| Rediseñar las cards de seguros para que quepan (truncate, `break-words`, `flex-wrap`, mover datos a su propia línea) | Corrige la causa raíz del overflow; mejora la legibilidad | Más trabajo de ajuste fino de clases | ✅ Seleccionada |
| Fijar `font-size: 16px` (o `max(16px, 1em)`) en `input, select, textarea` a nivel global | Soluciona el auto-zoom de inputs en todo el frontend con una sola regla | Puede cambiar levemente la apariencia de inputs chicos; requiere verificación darkmode | ✅ Seleccionada |
| `user-scalable=no` / `maximum-scale=1` en el meta viewport | Bloquea el zoom | iOS 10+ lo ignora para accesibilidad; mala práctica de accesibilidad | ❌ Rechazada |
| `touch-action: manipulation` en interactivos | Elimina doble-tap zoom y delay de tap, mantiene scroll/pinch | Requiere aplicar en los elementos correctos | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Corregir la causa raíz del overflow en lugar de ocultarlo.
- **Contexto**: `overflow-x: hidden` enmascararía el problema pero dejaría datos de seguros (póliza, certificado, fechas) visualmente cortados e inaccesibles.
- **Decisión**: Rediseñar las cards y layouts con clases responsive: `min-w-0`, `truncate`, `break-words`, `flex-wrap`, y reorganizar los bloques rígidos (`flex-shrink-0`) para que nunca empujen el ancho del layout. Aplicar en todas las páginas que tengan cards/tablas con datos largos.
- **Consecuencias**: Layout más limpio y legible; requiere revisión visual en móvil real en todas las páginas.

**ADR-002**: Fijar `font-size >= 16px` en form controls de forma global.
- **Contexto**: iOS auto-zoomea al enfocar inputs con fuente < 16px; hay múltiples modales y formularios con `text-sm`.
- **Decisión**: Agregar en `frontend/src/index.css`: `input, select, textarea { font-size: max(16px, 1em); }` manteniendo los tokens de color existentes. Regla única que cubre todo el frontend.
- **Consecuencias**: Elimina el zoom de inputs en todo el frontend; mantener verificación de darkmode.

**ADR-003**: `touch-action: manipulation` en elementos interactivos.
- **Contexto**: El tap en días del calendario y otros botones puede disparar doble-tap zoom.
- **Decisión**: Aplicar `touch-action: manipulation` de forma global a los elementos interactivos (botones, cards clicables, enlaces) vía CSS, y puntualmente donde haga falta en JSX.
- **Consecuencias**: Tap más rápido (sin delay de 350ms) y sin zoom accidental en toda la app.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Frontend React + Tailwind]  --(cambios CSS/JSX)-->  [Layout responsive]
      |                                                    |
      v                                                    v
[AutoShowPage / AutosPage]                          [DebtCalendar / modales]
      |                                                    |
      v                                                    v
Eliminar overflow horizontal (cards seguros)      touch-action + font-size 16px
```

No hay cambios de backend, API ni base de datos.

### 4.2 Componentes

#### 4.2.1 `frontend/src/pages/AutoShowPage.tsx`
- **Responsabilidad**: Detalle de auto + cards de seguros agrupadas por institución.
- **Cambios**: Rediseñar cada card de seguro para que el contenido no desborde el viewport en móvil:
  - Permitir `flex-wrap` en la fila principal de la card cuando haga falta.
  - `break-words` / `truncate` en póliza, certificado y aseguradora.
  - Bloque derecho (monto/frecuencia/fechas) sin `flex-shrink-0` rígido; permitir que las fechas se acorten o pasen a su propia línea en móvil.
  - Asegurar `min-w-0` en el contenedor flexible central.
- **Dependencias**: Solo clases Tailwind existentes.

#### 4.2.2 `frontend/src/components/DebtCalendar.tsx`
- **Responsabilidad**: Calendario mensual de cuotas con detalle por día.
- **Cambios**: Agregar `touch-action: manipulation` a los botones de día (y a los controles prev/next). Verificar que el grid `grid-cols-7 gap-1` no desborde en 320-360px. Asegurar que el detalle del día (card inferior) no cause overflow con textos largos (`truncate`/`break-words`).
- **Dependencias**: Ninguna nueva.

#### 4.2.3 `frontend/src/index.css`
- **Responsabilidad**: Estilos globales y tokens de tema.
- **Cambios**:
  - `input, select, textarea { font-size: max(16px, 1em); }` (junto a la regla de color existente).
  - `html { -webkit-text-size-adjust: 100%; }`.
  - Posible regla global `button { touch-action: manipulation; }` para cubrir interactivos (evaluar impacto).
- **Dependencias**: Ninguna.

#### 4.2.4 Modales y formularios con inputs `text-sm` (auditoría REQ-006)
- **Todos** los `input`/`select`/`textarea` con `text-sm` (14px) en el frontend: quitar `text-sm` o reemplazarlo por un tamaño >= 16px, manteniendo tokens de color del tema. Incluye `IconPickerModal.tsx`, `HousePickerModal.tsx`, `AnalyzerPickerModal.tsx`, `InstitutionCategoriesModal.tsx`, `AddInsuranceModal.tsx` y los formularios de creación/edición.
- **Dependencias**: Ninguna.

#### 4.2.5 Auditoría global de overflow (REQ-001, REQ-007, REQ-008)
- Recorrer **todas** las páginas (`frontend/src/pages/*`) y componentes (`frontend/src/components/*`) buscando:
  - `flex` con hijos `flex-shrink-0` que puedan empujar el ancho.
  - Textos largos sin `truncate`/`break-words` (pólizas, números, fechas, descripciones).
  - Grids/tablas con ancho fijo.
  - Cards con `p-*` + contenido inline ancho.
- Corregir cada caso con `min-w-0`, `truncate`, `break-words`, `flex-wrap`, `overflow-x-auto` solo cuando sea intencional (tablas desktop).
- **Dependencias**: Ninguna.

### 4.3 Modelo de datos

Sin cambios. No se toca el esquema SQLite ni los modelos.

### 4.4 APIs / Contratos

Sin cambios en endpoints.

### 4.5 Dependencias

- **Internas**: Componentes listados arriba; ninguna lógica de negocio se modifica.
- **Externas**: Ninguna nueva (solo Tailwind CSS ya incluido).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un auto con seguros con póliza/certificado/aseguradora largos, cuando se abre `/autos/:id` en un viewport de 360px, entonces la página cabe 100% sin scroll horizontal ni recorte lateral.
- [x] CA-002: Dado el calendario de Deudas en móvil, cuando se toca cualquier día del mes, entonces no hay cambio de zoom de la página ni desplazamiento lateral del layout.
- [x] CA-003: Dado **cualquier** formulario/modal del frontend en iOS Safari, cuando se enfoca un `input`/`select`/`textarea`, entonces NO se produce auto-zoom (font-size computado >= 16px).
- [x] CA-004: Dado un elemento interactivo (botón de día, card clicable, botón de acción), cuando se toca dos veces rápidamente, entonces no se produce doble-tap zoom.
- [x] CA-005: Dado **cualquier** página del frontend (listados y detalles de Autos, Deudas, Servicios, Homes, Instituciones, Pension, Hijos, Categorías, Salarios, Notificaciones, Bills), cuando se navega en móvil (320px/360px/414px), entonces NO hay scroll horizontal ni contenido cortado lateralmente.
- [x] CA-006: Dado **cualquier** modal (subida de facturas, agregar seguro, selector de casas, picker de iconos, picker de analizadores, categorías de instituciones, emails, pagar factura), cuando se abre en móvil, entonces el modal cabe en el viewport sin desbordes laterales.
- [x] CA-007: Dado el layout en desktop, cuando se navega por las páginas modificadas, entonces el layout no se rompe (grids y cards se mantienen correctos).
- [x] CA-DARK: Los inputs modificados mantienen los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verifica legibilidad en darkmode (texto y placeholders).
- [x] CA-BACK: No aplica (no se crean páginas de detalle nuevas; `AutoShowPage` ya tiene flecha atrás registrada en `BACK_ROUTES`).

### 5.2 No funcionales

- [x] CA-NF-001: El bundle del frontend no incrementa dependencias nuevas.
- [x] CA-NF-002: La app sigue pasando `npm run build` y el server local responde `GET /health` con `{"status":"ok"}`.

### 5.3 Testing

- **Unit tests**: No aplica lógica nueva; verificar build de Vite sin errores.
- **Integration tests**: Navegación móvil por todas las páginas (listados, detalles, formularios) en responsive.
- **E2E tests**: Prueba manual en dispositivo móvil real / DevTools mobile emulation (320px, 360px, 375px, 414px):
  - Sin scroll horizontal en ninguna página ni modal.
  - Sin zoom al tocar días del calendario.
  - Sin zoom al enfocar inputs de cualquier formulario/modal.
- **Carga/Performance**: Sin impacto (cambios CSS puros).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Fix global `index.css`: `font-size: max(16px,1em)` en form controls, `-webkit-text-size-adjust`, `touch-action` en interactivos | 0.5 día | Ninguna |
| 2 | Rediseño responsive de cards de seguros en `AutoShowPage.tsx` (eliminar overflow horizontal) | 1 día | Fase 1 |
| 3 | `touch-action: manipulation` + verificación responsive en `DebtCalendar.tsx` | 0.5 día | Fase 1 |
| 4 | Auditoría global: recorrer TODAS las páginas y componentes para eliminar overflow horizontal y fix inputs `text-sm` | 1.5 días | Fase 1 |
| 5 | Build de frontend, prueba local en modo responsive (320/360/414px), pruebas manuales con el usuario | 1 día | Fases 2-4 |

### 6.2 Milestones

1. **MVP**: Fase 1 + 2 + 3 (problemas reportados: Autos y calendario Deudas).
2. **V1.0**: Fase 4 (auditoría global completa) + validación final en todas las páginas.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El fix de font-size 16px cambia la apariencia de inputs chicos | Media | Bajo | Revisar visualmente en darkmode y light mode; ajustar con clases puntuales si es necesario |
| `touch-action: manipulation` global en botones altera scroll de contenedores | Baja | Bajo | Aplicarlo en elementos interactivos específicos y validar scroll vertical en listas |
| Los textos de seguros siguen desbordando con datos extremadamente largos | Media | Medio | Combinar `truncate` + `break-words` + `min-w-0`; validar con strings de prueba largos |
| Regresión visual en desktop | Baja | Medio | Verificar grids desktop en build local antes de release |

## 8. Notas y Referencias

- WebKit: "New Interaction Behaviors in iOS 10" — zoom automático por contenido ancho (https://webkit.org/blog/7367/new-interaction-behaviors-in-ios-10/)
- WebKit: "More Responsive Tapping on iOS" — `touch-action: manipulation` (https://webkit.org/blog/5610/more-responsive-tapping-on-ios/)
- CSS-Tricks: "16px or Larger Text Prevents iOS Form Zoom" (https://css-tricks.com/16px-or-larger-text-prevents-ios-form-zoom/)
- Rick Strahl: "Preventing iOS Textbox Auto Zooming and ViewPort Sizing" (https://weblog.west-wind.com/posts/2023/Apr/17/Preventing-iOS-Textbox-Auto-Zooming-and-ViewPort-Sizing)
- Dan Burzo: "Prevent the need to zoom into inputs with CSS" — `font-size: max(16px, 1em)` (https://danburzo.ro/css-safari-zoom-inputs/)
- Precedente relacionado: SPEC-011 (Soporte Responsive), SPEC-060 (fix global inputs darkmode), SPEC-014 (fix overflow BillsPage)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-11 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-11 | p40la-ihost-team | Implementación completa: fix global `index.css` (font-size 16px, text-size-adjust, touch-action), cards de seguros responsive en AutoShowPage, DebtCalendar, auditoría global de overflow e inputs text-sm. Build OK, pruebas manuales móvil satisfactorias. |
| 2026-09-11 | p40la-ihost-team | Release: merge a `main`, issue #70 cerrado con label `spec/released`. Commit: `SPEC-068: fix zoom móvil global (font-size 16px, touch-action, overflow horizontal)`. |