---
title: "Responsividad de los gráficos de Deudas: Gráfica, Análisis y Calendario"
id: "SPEC-102"
status: "released"
author: "opencode"
created: "2026-09-30"
updated: "2026-09-30"
github_issue: 105
---

# Responsividad de los gráficos de Deudas: Gráfica, Análisis y Calendario

**ID**: SPEC-102  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-30  
**Actualizado**: 2026-09-30

---

## 1. Resumen Ejecutivo

Las tres vistas de Deudas — el tab **"Gráfica"** (`DebtChart`), el tab **"Análisis"** (`DebtAnalysis`) y el tab **"Calendario"** (`DebtCalendar`) — fueron diseñadas pensando en desktop y presentan problemas de legibilidad/uso en pantallas pequeñas. El requerimiento pide que **las páginas de gráficos de deudas sean responsivas**, con un enfoque doble: (a) **escalar los gráficos al ancho del contenedor** (SVG responsive sin scroll horizontal forzado) y (b) **una versión móvil** donde los elementos se reordenan/apilan para lectura vertical en pantallas chicas. El usuario confirmó que aplica a los tres componentes.

Es importante porque la app se usa principalmente desde el celular (acceso al iHost desde el móvil, responsive en SPEC-011) y hoy el gráfico de `DebtChart` usa un `width` medido con `ResizeObserver` + `overflow-x-auto`, lo que en móvil obliga a scrollear horizontalmente y recorta el tooltip; el donut de `DebtAnalysis` es fijo de 180px; y el calendario usa una grilla de 7 columnas que en 320px deja celdas muy angostas. El resultado esperado es que los tres tabs se vean y usen bien en un rango de 320px a desktop, manteniendo los patrones de UI existentes.

Resultado esperado: convertir `DebtChart` a SVG con `viewBox` responsive (patrón ya usado en `BillAnalysis`) + ajustar el layout de tarjetas/selectores en móvil; revisar y ajustar `DebtAnalysis` (donut, SummaryCards, listas) y `DebtCalendar` (grilla, día seleccionado) para pantallas pequeñas. Sin cambios de backend ni de datos.

**Consideraciones iHost**: todo es frontend estático (build Vite). Sin dependencias nuevas. `ResizeObserver` ya está disponible en navegadores modernos; se mantiene para re-escalado, no para fijar ancho. No hay cambios de esquema SQLite ni APIs.

**Consideraciones de UI (SPEC-060/063)**: selectores/toggles con tokens del tema (`bg-card`, `text-text`/`text-text-secondary`) y verificación en darkmode. No se crean páginas de detalle ni links "← Título".

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Hacer que el tab **"Gráfica"** (`DebtChart`) sea **responsive**: el SVG (Cronograma y Saldo total) se **escala al ancho del contenedor** mediante `viewBox` + `preserveAspectRatio`, sin obligar a scroll horizontal en móvil (ver ADR-001).
2. **REQ-002**: En **móvil** (aprox. `< 640px`), el layout del tab "Gráfica" se **reordena**: las 3 tarjetas de resumen quedan apiladas (ya lo hace `grid-cols-1 sm:grid-cols-3`), el título + selector de moneda + selector de vistas se acomodan con `flex-wrap`, y los tooltips se posicionan dentro del viewport (clamping) para no cortarse.
3. **REQ-003**: Hacer que el tab **"Análisis"** (`DebtAnalysis`) sea **responsive**: el donut SVG (hoy 180px fijo) se escala al ancho disponible en móvil, las SummaryCards se apilan (`sm:grid-cols-3`), el selector de año/mes/moneda se envuelve con `flex-wrap` y las listas por deuda usan truncado y `shrink-0` para no desbordar (patrón ya parcialmente presente).
4. **REQ-004**: Hacer que el tab **"Calendario"** (`DebtCalendar`) sea **responsive**: la grilla de 7 columnas mantiene proporción en 320px (celdas con `min-h-[44px]`, gap reducido en móvil), los días con cuotas muestran los indicadores sin desbordar, y el detalle del día seleccionado apila/trunca correctamente (nombre, cuota, monto, estado y botón Pagar).
5. **REQ-005**: Mantener el **darkmode** y los **tokens del tema** en todos los ajustes (nada de colores hardcodeados).
6. **REQ-006**: No agregar **ninguna dependencia npm** nueva; los gráficos siguen siendo SVG/CSS hechos a mano.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: En `DebtChart`, el tooltip debe seguir al cursor con posicionamiento absoluto que **no se salga del viewport** en móvil (clamping a `window.innerWidth`), manteniendo el contenido actual (nombre, monto, "Termina:", "Restan:", "Saldo después:").
2. **REQ-008**: El cronograma (vista por filas) en móvil debe seguir siendo legible: nombre con truncado, barra + fecha visible sin scroll horizontal (el ancho del texto de la fila se ajusta al contenedor).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Verificar en desktop que no se pierde información ni se empeora la legibilidad tras los cambios responsivos (regresión visual de las vistas actuales).

### 2.4 Requerimientos P0 adicionales (cambio iterativo solicitado por el usuario)

1. **REQ-010**: **Versión móvil dedicada para el tab "Gráfica" (Cronograma)**: en pantallas móviles (`< sm`), reemplazar el SVG compactado del cronograma por una **lista de filas/tarjetas con barras de progreso horizontales** — cada deuda muestra su nombre, monto, fecha de finalización y una barra que va desde "hoy" hasta su fecha de fin (proporcional al span total). Ver ADR-004. El Saldo total se mantiene como SVG escalable. (Solicitado por el usuario: "la versión móvil de la gráfica no es muy buena... ¿podríamos tener una versión mobile solo para mobile?")
2. **REQ-011**: La vista móvil dedicada aplica **solo al tab Gráfica** (`DebtChart`); Análisis y Calendario se mantienen con los ajustes responsivos de REQ-003/004.
3. **REQ-012**: El **selector de moneda filtra TODO el tab "Gráfica"** (Cronograma móvil, Cronograma SVG, Saldo total, tarjeta "Total adeudado", "Deudas activas" y "Libre de deudas"): al elegir una moneda, todas las vistas muestran solo las deudas de esa moneda. Sin selector visible cuando hay una sola moneda. (Solicitado por el usuario: "cambia la moneda, ¿cambia los datos?")
4. **REQ-013**: **Vista móvil del tab "Calendario" como agenda/list por mes**: en pantallas móviles (`< sm`) se reemplaza la grilla de 7 columnas por una **lista de cuotas del mes agrupadas por fecha** (orden cronológico), con navegación `< mes >`. La grilla queda solo para desktop (`hidden sm:block`). (Solicitado por el usuario: "ahora el tab de calendario, busquemos algo más amigable para mobile")

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El re-escalado es CSS/SVG nativo (sin re-render costoso). El `ResizeObserver` solo actualiza dimensiones; las escalas de los gráficos se recalculan en el render. O(n) por gráfico.
- **Seguridad**: Sin cambios en APIs ni datos sensibles.
- **Almacenamiento**: Sin cambios de esquema ni datos.
- **Disponibilidad**: Solo frontend estático; build con Vite.
- **iHost**: **Cero dependencias nuevas**. SVG/CSS a mano como `BillAnalysis`/`DebtAnalysis`/`DebtChart`.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **`frontend/src/components/DebtChart.tsx`** (SPEC-101): 
  - Usa `ResizeObserver` (líneas 35–43) para medir `el.clientWidth` y fija `width = max(clientWidth, 320)`, luego renderiza `<svg width={width} height={H+26}>` (líneas 170, 223) con `overflow-x-auto` en el contenedor (línea 323). Esto produce **scroll horizontal** en móvil y el gráfico no se escala: si el ancho es 320px mínimo, en una pantalla de 320–360px apenas entra; en pantallas más chicas se corta.
  - El **tooltip** (líneas ~330–341) usa `position: fixed` con `left: min(e.clientX+14, innerWidth-250)` y `top: e.clientY+14`, ya con clamping básico, pero en móvil el ancho máximo de 250px y el offset pueden cortar contenido.
  - Las **tarjetas de resumen** ya son `grid-cols-1 sm:grid-cols-3` (patrón OK).
  - El header del panel usa `flex-wrap` (línea ~289) para título + selector de moneda + segmented control.
- **`frontend/src/components/DebtAnalysis.tsx`**:
  - Donut SVG **fijo de 180px** (línea 254 `<svg width="180" height="180" viewBox="0 0 180 180">`), centrado con `flex-col sm:flex-row` en la sección "Análisis por deuda" (línea 252). En móvil el donut fijo deja espacio y la lista al lado se comprime.
  - `SummaryCard` (grid `sm:grid-cols-3`, líneas 230–246) y header con `flex-wrap` (líneas 163–189) ya son razonablemente responsivos.
  - Listas "Detalle por deuda" (líneas 287–314) usan `truncate` + `shrink-0` (patrón OK).
- **`frontend/src/components/DebtCalendar.tsx`**:
  - Grilla `grid-cols-7 gap-1 sm:gap-2` (línea 101) con celdas `min-h-[44px] sm:min-h-[56px]` (línea 114). En 320px cada celda mide ~43px, legible pero con indicadores de 3 puntitos que pueden quedar apretados.
  - Detalle del día (líneas 138–186): usa `flex justify-between`, `truncate`, `shrink-0` — razonablemente responsive ya.
- **Patrón de referencia responsive en el repo**: `frontend/src/components/BillAnalysis.tsx` usa `<svg viewBox={\`0 0 ${W} ${H}\`} className="block w-full min-w-[420px]" preserveAspectRatio="xMidYMid meet">` (líneas 389–390) dentro de `w-full overflow-x-auto` — escala al contenedor con un mínimo para no aplastar. Este es el patrón a replicar en `DebtChart`.
- **i18n**: bloque `deudas.*` en `frontend/public/i18n/{es,en}.json`. Podrían agregarse claves nuevas si se introducen textos de ayuda/tooltips nuevos (en general no hace falta).
- **Tokens de tema**: `bg-bg`, `bg-card`, `text-text`, `text-text-secondary`, `text-primary`, `bg-border`, etc. definidos en `frontend/tailwind.config.js`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **SVG con `viewBox` + `preserveAspectRatio` + ancho mínimo** (patrón `BillAnalysis`) | Escala al contenedor sin scroll forzado, reutiliza patrón existente, cero deps | Requiere reajustar escalas/etiquetas si el ancho mínimo se baja | ✅ **Seleccionada** para Gráfica |
| Mantener `overflow-x-auto` con ancho fijo actual | Cero cambios | Mala UX móvil (scroll horizontal), tooltip cortado | ❌ Rechazada |
| `ResizeObserver` para medir y escalar con `viewBox` dinámico | Reutiliza el observador ya presente | Más código de medición; `viewBox` nativo es suficiente | ❌ Rechazada (simplificar) |
| Rediseñar Análisis/Calendario desde cero | Solución "perfecta" | Alto riesgo de regresión, fuera de alcance | ❌ Rechazada |
| Donut fijo en Análisis (dejarlo igual) | Cero trabajo | Sigue sin escalar en móvil | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Gráficos SVG escalables con `viewBox` + ancho mínimo, no scroll horizontal**
- **Contexto**: `DebtChart` hoy fija `width` en px con `overflow-x-auto`, generando scroll horizontal en móvil.
- **Decisión**: Usar `<svg viewBox={\`0 0 ${W} ${H}\`} preserveAspectRatio="xMidYMid meet" className="block w-full min-w-[360px]">` (patrón de `BillAnalysis`). El `ResizeObserver` se simplifica o elimina: el navegador escala el SVG al contenedor. Para el cronograma, el ancho de la fila (texto de nombre a la izquierda + track + fecha) se calcula con un `W` de diseño fijo razonable (ej. 720) y el SVG escala; el `min-w` evita que en pantallas medianas quede ilegible, y si el contenedor es muy chico el scroll horizontal queda como fallback (no como default).
- **Consecuencias**: Sin scroll horizontal forzado en la mayoría de casos; el tooltip usa coordenadas de viewport (ya las tiene) y se mantiene el clamping. Las etiquetas del eje X se revalidan para no solaparse al escalar.

**ADR-002: Responsividad por capas (layout) + escalado (gráficos)**
- **Contexto**: El requerimiento pide "ambos enfoques": escalar al contenedor y tener versión móvil.
- **Decisión**: Aplicar dos técnicas complementarias: (1) **escalado de gráficos** con `viewBox`/`preserveAspectRatio` para donut, cronograma y saldo; (2) **layout móvil** con las utilidades de Tailwind ya existentes (`grid-cols-1 sm:grid-cols-3`, `flex-wrap`, `min-w-0`, `truncate`, `shrink-0`) en Análisis, Calendario y las tarjetas/selector de Gráfica. No se crea un componente responsive separado; se ajustan los existentes.
- **Consecuencias**: Cambios acotados por componente, sin duplicación de vistas, coherentes con el patrón del resto de la app.

**ADR-005: Selector de moneda filtra todo el tab Gráfica**
- **Contexto**: En la implementación inicial el selector de moneda solo afectaba Saldo total y "Total adeudado", dejando el cronograma con todas las monedas mezcladas — inconsistencia reportada por el usuario.
- **Decisión**: Un único conjunto `filteredItems` (deudas filtradas por `effectiveCurrency`) alimenta **todas** las vistas del tab: Cronograma móvil, Cronograma SVG, Saldo total y las 3 tarjetas (Total adeudado, Deudas activas, Libre de deudas). Si no hay selector visible (una sola moneda), `effectiveCurrency` es esa moneda y el filtro es la identidad. Si el filtro queda vacío (moneda sin deudas), se muestra el empty state.
- **Consecuencias**: El cronograma responde al selector igual que el resto; la lógica de filtrado se centraliza en un solo `useMemo`, sin código duplicado entre vistas.

**ADR-006: Agenda por mes en móvil para el Calendario**
- **Contexto**: El usuario pidió algo más amigable para mobile en el tab Calendario; la grilla de 7 columnas es apretada en 320px y obliga a tocar día por día.
- **Decisión**: En móvil (`sm:hidden`) se muestra una **agenda/list de cuotas del mes agrupadas por fecha** (día → lista de cuotas, orden cronológico), con navegación `< mes >` y el mismo detalle de cuota (descripción, cuota #, institución, monto, estado, botón Pagar). La grilla existente se envuelve en `hidden sm:block` y queda solo para desktop. La agenda reutiliza los datos ya cargados (`bills`, `billsByDay`), `formatMoney` y el patrón de detalle de cuota existente.
- **Consecuencias**: Un solo componente con dos renders condicionados por breakpoint; sin JS de media query ni dependencias. La agenda móvil reutiliza el bloque de detalle de cuota del día (DRY).

**ADR-003: Sin dependencias y sin cambios de backend**
- **Contexto**: Restricción de iHost y del proyecto (SPEC-054/055/100).
- **Decisión**: Mantener SVG/CSS a mano y `ResizeObserver` solo donde aporte (o eliminarlo en favor de `viewBox`). Sin librerías de charts ni cambios de esquema/API.
- **Consecuencias**: Bundle mínimo, sin migraciones.

**ADR-004: Vista móvil dedicada con clases `sm:hidden` / `hidden sm:block`**
- **Contexto**: El usuario reportó que la versión móvil de la gráfica no es buena; escalar el cronograma SVG de desktop a 360px lo deja ilegible.
- **Decisión**: En `DebtChart`, renderizar **dos versiones del cronograma** que el CSS muestra según el breakpoint: (1) en móvil (`sm:hidden`) una **lista con barras de progreso horizontales** (cada deuda = fila con nombre, monto, fecha fin y barra proporcional desde hoy hasta su finalización), y (2) en desktop (`hidden sm:block`) el SVG `viewBox` existente. El Saldo total mantiene el SVG escalable para ambos. Se usa el patrón CSS de `BillsPage.tsx` (líneas 257/290), sin hooks de media query JS.
- **Consecuencias**: Sin JS adicional, sin dependencias; la barra móvil reutiliza los datos ya derivados (`items` con `endDate`), `formatMoney` y la paleta `COL`. El tooltip de escritorio queda sin cambios; en móvil la barra muestra la info directamente en la fila (no hace falta tooltip).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DeudasPage.tsx] --tabs--> DebtChart / DebtAnalysis / DebtCalendar
        |                          (SVG viewBox + layout móvil Tailwind)
        v
[SQLite debts/debt_bills]  --api-->  render cliente (sin cambios backend)
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/components/DebtChart.tsx` (modificado)
- **Responsabilidad**: Gráfica responsive con vista móvil dedicada.
- **Cambios**:
  - **Cronograma en móvil** (`sm:hidden`): lista de filas, una por deuda, con:
    - nombre (truncado) + fecha de finalización (derecha),
    - monto (con símbolo de moneda),
    - **barra de progreso horizontal**: `width: ${pct}%` donde `pct = (endDate - hoy) / spanTotal` (span desde hoy hasta la deuda más lejana + 30 días), con el color de la paleta `COL[i % len]`. Barra de 2px (h-2) con `bg-border` de track.
  - **Cronograma en desktop** (`hidden sm:block`): SVG `viewBox` escalable existente (`renderCronograma`).
  - **Saldo total**: SVG `viewBox` escalable existente (`renderSaldo`) para ambos breakpoints (sin cambios adicionales).
  - Render condicional con clases `sm:hidden` / `hidden sm:block` dentro del contenedor del gráfico, manteniendo el selector de vistas (Cronograma/Saldo total), tarjetas de resumen, selector de moneda y nota al pie compartidos.
- **Dependencias**: `useI18nStore`, `useCurrencyFormatStore`, `Icon`, `Select`, helpers de `utils/debtMonthlyAggregation`, tokens de tema.
- **Ubicación**: `frontend/src/components/DebtChart.tsx`.

#### 4.2.2 `frontend/src/components/DebtAnalysis.tsx` (modificado)
- **Responsabilidad**: Análisis responsive.
- **Cambios**:
  - Donut: `viewBox="0 0 180 180"` + `className="block w-full max-w-[180px] mx-auto"` (o `w-40` en móvil) para que escale al ancho disponible y se centre. Mantener `flex-col sm:flex-row` para apilar donut + lista en móvil.
  - Header (selector año/mes + moneda): ya usa `flex-wrap`; verificar que el `Select` de moneda (`w-44`) no desborde en 320px (ajustar a `w-full sm:w-44` si hace falta).
  - SummaryCards: confirmar `grid-cols-1 sm:grid-cols-3` y que los totales multi-moneda se envuelvan.
  - Listas por deuda y cuotas: ya usan `truncate`/`shrink-0`; verificación visual en 320px.
- **Dependencias**: existentes.
- **Ubicación**: `frontend/src/components/DebtAnalysis.tsx`.

#### 4.2.3 `frontend/src/components/DebtCalendar.tsx` (modificado)
- **Responsabilidad**: Calendario responsive con agenda móvil.
- **Cambios**:
  - **Agenda por mes en móvil** (`sm:hidden`): lista de días del mes con cuotas, orden cronológica. Por cada día con cuotas: encabezado de día (número + mes corto, destacado si es hoy) y las cuotas de ese día (reutilizando el bloque de detalle de cuota existente: descripción, cuota #, institución, monto, estado, botón Pagar). Días sin cuotas se omiten (o se muestra empty del mes si ninguno tiene cuotas).
  - **Grilla desktop** (`hidden sm:block`): la grilla `grid-cols-7` actual queda solo para desktop, incluyendo el detalle del día seleccionado.
  - Header de navegación `< mes >` compartido por ambas vistas (ya con `truncate`).
  - Mantener selección de día y `DebtPayModal` para ambas vistas.
- **Dependencias**: `api`, `useI18nStore`, `useCurrencyFormatStore`, `Icon`, `DebtPayModal`, tipos `DebtBill`.
- **Ubicación**: `frontend/src/components/DebtCalendar.tsx`.

### 4.3 Modelo de datos

Sin cambios. Se usan los modelos existentes (`Debt`, `DebtBill`, `Currency`).

### 4.4 APIs / Contratos

Sin endpoints nuevos. Uso de los existentes:
- `GET /api/debts` → `Debt[]`
- `GET /api/debt-bills?year=&month=` → `DebtBill[]`

### 4.5 Dependencias

- **Internas**: `DebtChart.tsx`, `DebtAnalysis.tsx`, `DebtCalendar.tsx` (modificados). i18n solo si se agregan textos nuevos.
- **Externas**: **ninguna**.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: El tab "Gráfica" muestra el cronograma y el saldo total **escalados al ancho del contenedor** (sin scroll horizontal en pantallas ≥ 360px) usando `viewBox`/`preserveAspectRatio`.
- [ ] CA-002: En móvil (< 640px) las tarjetas de resumen se apilan, el selector de moneda y el de vistas se envuelven, y los tooltips no se cortan (clamping al viewport).
- [ ] CA-003: El tab "Análisis" muestra el donut escalado/centrado en móvil y las SummaryCards apiladas; el selector de moneda no desborda en 320px.
- [ ] CA-004: El tab "Calendario" mantiene la grilla de 7 columnas legible en 320px (celdas táctiles ≥ 44px, indicadores visibles) y el detalle del día no desborda.
- [ ] CA-005: Los cambios se ven correctamente en **modo claro y oscuro** (tokens del tema, sin colores hardcodeados).
- [ ] CA-006: En **desktop** no hay regresión visual: las tres vistas se ven igual o mejor que antes.
- [ ] CA-007: En **móvil**, el tab "Gráfica" (vista Cronograma) muestra la **lista con barras de progreso** (nombre, monto, fecha fin, barra proporcional) en lugar del SVG compactado; el Saldo total sigue como SVG escalable.
- [ ] CA-008: Al **cambiar la moneda** en el selector, **todas** las vistas del tab (Cronograma móvil, Cronograma SVG, Saldo total y las 3 tarjetas) se actualizan mostrando solo las deudas de esa moneda; sin monedas seleccionables (una sola) no se muestra selector.
- [ ] CA-009: En **móvil**, el tab "Calendario" muestra la **agenda por mes** (lista de cuotas agrupadas por fecha, orden cronológico) con navegación `< mes >`; la grilla de 7 columnas queda solo en desktop.
- [ ] CA-DARK: Selectores/toggles del tab usan tokens del tema (`bg-card`, `text-text`/`text-text-secondary`) y se verifica legibilidad en darkmode.

### 5.2 No funcionales

- [ ] CA-NF-001: No se agrega ninguna dependencia npm nueva (package.json sin cambios de deps).
- [ ] CA-NF-002: El build de Vite (`npm run build` en `frontend/`) compila sin errores y sirve el i18n actualizado.

### 5.3 Testing

- **Unit tests**: No aplica lógica nueva pura (los ajustes son de layout/estilos). Si se extrae un helper de truncado de nombre para el cronograma, cubrirlo con `node --test`.
- **Integration tests**: render de los tres tabs en `DeudasPage` (verificación manual en 320px/640px/desktop + build).
- **E2E tests**: flujo manual — abrir `/deudas?tab=grafica`, `?tab=analisis`, `?tab=calendario` en un viewport móvil (DevTools) y desktop; hover de filas/puntos; cambiar moneda; darkmode.
- **Carga/Performance**: sin impacto (escalado CSS/SVG nativo).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `DebtChart`: `viewBox` responsive + tooltip clamping + truncado del cronograma | 0.5–1 día | Ninguna |
| 2 | `DebtChart`: vista móvil dedicada del cronograma (lista con barras de progreso, `sm:hidden`/`hidden sm:block`) | 0.5 día | Fase 1 |
| 3 | `DebtAnalysis`: donut escalable + ajustes de layout móvil | 0.5 día | Ninguna |
| 4 | `DebtCalendar`: agenda por mes en móvil (lista agrupada por fecha) + grilla solo desktop | 0.5 día | Ninguna |
| 5 | Build, pruebas manuales (320px/640px/desktop, darkmode), QA | 0.5 día | Fases 1–4 |

### 6.2 Milestones

1. **MVP**: Gráfica responsive (escalado + tooltip), Análisis y Calendario sin desbordes en 320px.
2. **V1.0**: Validación en local (móvil/desktop, darkmode), build OK.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Escalar con `viewBox` deja etiquetas ilegibles en pantallas muy chicas | Media | Medio | `min-w-[360px]` + `overflow-x-auto` como fallback (patrón `BillAnalysis`) |
| Regresión visual en desktop | Media | Medio | Verificar desktop en CA-006 antes del release |
| Donut 180px escalado pierde legibilidad del texto central | Baja | Bajo | `max-w-[180px]` y centrado; texto con tamaño relativo al SVG |
| Celdas del calendario desbordan en 320px | Baja | Medio | gap reducido + `min-h-[44px]` + `px-0` en móvil |
| Tooltip cortado en móvil | Media | Bajo | Clamping de `left`/`top` al viewport + ancho `max-w-60` |
| Vista móvil del cronograma duplica mantenimiento (dos renders) | Baja | Bajo | Comparten los mismos datos derivados (`items`/`endDate`) y helpers; solo cambia el markup |

## 8. Notas y Referencias

- SPEC-011 (Soporte Responsive para Dispositivos Móviles)
- SPEC-054 (Módulo Deudas + Calendario)
- SPEC-055 (Análisis de Deudas por Mes)
- SPEC-100 (Tab Gráfica en Deudas: barras)
- SPEC-101 (Rediseño del tab Gráfica: Cronograma + Saldo total — origen de `DebtChart` actual)
- SPEC-060 (Tokens del tema en inputs/selectores darkmode)
- Referencia de patrón responsive: `frontend/src/components/BillAnalysis.tsx` (`viewBox` + `min-w-[420px]` + `overflow-x-auto`, líneas 389–390)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-30 | opencode | Creación inicial de la especificación (draft) |
| 2026-09-30 | opencode | Desarrollo (in_progress): DebtChart con SVG viewBox responsive + tooltip clamping + truncado del cronograma; DebtAnalysis con donut escalable/centrado y totales multi-moneda con flex-wrap; DebtCalendar con grilla gap reducido en móvil y header con truncate |
| 2026-09-30 | opencode | Cambio iterativo solicitado por el usuario (in_progress): vista móvil dedicada del cronograma en Gráfica (lista con barras de progreso) + REQ-010/011, ADR-004, CA-007 |
| 2026-09-30 | opencode | Cambio iterativo solicitado por el usuario (in_progress): selector de moneda filtra todo el tab Gráfica (cronograma incluido) + REQ-012, ADR-005, CA-008 |
| 2026-09-30 | opencode | Cambio iterativo solicitado por el usuario (in_progress): agenda por mes en móvil para el tab Calendario + REQ-013, ADR-006, CA-009 |
| 2026-09-30 | opencode | Release: commit de implementación `2195f81`, merge a main, issue #105 cerrado (spec/released) |