---
title: "Rediseño del tab Gráfica en Deudas: Cronograma y Saldo total (prototipo deudas-grafica.html)"
id: "SPEC-101"
status: "in_progress"
author: "opencode"
created: "2026-09-30"
updated: "2026-09-30"
github_issue: 104
---

# Rediseño del tab Gráfica en Deudas: Cronograma y Saldo total (prototipo deudas-grafica.html)

**ID**: SPEC-101  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-09-30  
**Actualizado**: 2026-09-30

---

## 1. Resumen Ejecutivo

El tab "Gráfica" (`?tab=grafica`) de la sección Deudas fue implementado en SPEC-100 como un gráfico de **barras verticales** agrupadas por mes/año de finalización. El usuario quiere **reemplazar el contenido** de ese tab para que se vea y funcione como el **prototipo HTML adjunto** (`deudas-grafica.html`): un panel con **dos vistas** seleccionables ("Cronograma" y "Saldo total"), **tarjetas de resumen** ("Total adeudado", "Deudas activas", "Libre de deudas"), y **tooltips** sobre cada elemento. En lugar del arreglo RAW de ejemplo del prototipo, se deben usar **las deudas reales del proyecto** (nombre = `description`, fecha de finalización, monto/saldo = `total`).

Es importante porque hoy la gráfica muestra solo la distribución por mes de vencimiento (barras verticales), pero no responde visualmente a "¿cuánto debo en total?", "¿cuándo quedo libre de deudas?" ni "¿qué deuda termina primero/última?". El prototipo añade un **cronograma tipo Gantt** (cada deuda es una fila con una barra que va desde hoy hasta su fecha de finalización) y una **curva de saldo total en el tiempo** (el balance baja cada vez que termina una deuda). Ambas vistas conservan el estado vacío y los tooltips.

Resultado esperado: reemplazar el contenido del tab "Gráfica" manteniendo la estructura de `DeudasPage.tsx` y los demás tabs intactos, usando los mismos patrones de UI (cards, tokens del tema, i18n) y las deudas reales ya cargadas por `api.debts.list()`. No se tocan los tabs Calendario, Deudas ni Análisis.

**Consideraciones iHost**: el gráfico se renderiza en el cliente (navegador), sin llamadas N+1 nuevas ni dependencias npm. El cálculo de fechas y saldos se hace con funciones puras en el frontend. No hay cambios de esquema SQLite ni de API backend.

**Consideraciones de UI (SPEC-060/063)**: el tab no agrega inputs ni formularios; los selectores/toggles deben usar tokens del tema (`bg-card`, `text-text`/`text-text-secondary`) y verificarse en darkmode. No se crean páginas de detalle ni links "← Título".

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Reemplazar el contenido del tab "Gráfica" (`?tab=grafica`) de Deudas por el nuevo diseño del prototipo `deudas-grafica.html`, **sin modificar** los tabs Calendario, Deudas ni Análisis.
2. **REQ-002**: Mostrar **tres tarjetas de resumen** arriba del panel: **Total adeudado** (suma de `total` de las deudas), **Deudas activas** (cantidad de deudas) y **Libre de deudas** (fecha de finalización de la deuda más lejana, formato "Ago 2028"). Con las deudas reales.
3. **REQ-003**: Panel con **selector de dos vistas** (segmented control): **"Cronograma"** (por defecto) y **"Saldo total"**. El selector usa tokens del tema y se verifica en darkmode.
4. **REQ-004**: Vista **"Cronograma"**: cada deuda es una **fila** con su nombre, su monto (`total` con su moneda), una **barra horizontal** que va desde "hoy" hasta su **fecha de finalización**, y la etiqueta de esa fecha al final. Ordenadas de la fecha de finalización más cercana a la más lejana. Marcador vertical "Hoy" al inicio. Ticks de año en el eje X. Replica el prototipo con las deudas reales.
5. **REQ-005**: Vista **"Saldo total"**: **gráfico de línea/área** con el balance total sobre el tiempo, que **baja escalonadamente** cada vez que termina una deuda. Cada punto es una deuda saldada (círculo). Etiqueta "Libre de deudas: {fecha}" al final de la curva. Grid horizontal y ticks de año. Replica el prototipo con las deudas reales.
6. **REQ-006**: **Tooltips** sobre cada fila (Cronograma) y cada punto (Saldo total) que muestren: nombre de la deuda, monto, "Termina: {fecha}" y "Restan: {X años Y meses}". En Saldo total, además "Saldo después: {monto}". Texto relativo ("Este mes", "1 año", "2 años 3 meses", etc.) como en el prototipo.
7. **REQ-007**: Si **no hay deudas**, mostrar un **mensaje de estado vacío** (patrón EmptyCard existente) en lugar de tarjetas/gráficos vacíos.
8. **REQ-008**: Las deudas se cargan desde las **deudas reales** del proyecto (`api.debts.list()`, ya disponibles en `DeudasPage`), usando `description` como nombre y `total` como monto.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: La **fecha de finalización** de cada deuda se deriva de `start_date + installments_total` meses (clampeando al último día del mes), replicando la lógica del backend `dueDateForInstallment` (SPEC-054) y la función `debtEndMonth` existente (SPEC-100), extendida para devolver el **día** exacto. Sin llamadas N+1 a `listBills`. Ver ADR-001.
2. **REQ-010**: **Multi-moneda**: el cronograma muestra cada monto con el símbolo de su moneda (`currency_code`). El "Total adeudado" y la curva "Saldo total" se calculan **por moneda seleccionada**: un selector de moneda (patrón existente en `DebtAnalysis`) filtra las deudas de la vista de saldo y el total, con default la primera moneda de mayor saldo. Ver ADR-002.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-011**: Notas explicativas bajo cada vista ("De la deuda que termina primero a la que termina último...", "El saldo baja cada vez que termina una deuda...") como en el prototipo.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El gráfico se calcula 100% en el cliente a partir de `api.debts.list()` ya cargada. Sin llamadas N+1 nuevas. Funciones de agregación O(n).
- **Seguridad**: El tab usa las APIs autenticadas existentes; sin datos sensibles nuevos.
- **Almacenamiento**: Sin cambios de esquema ni datos nuevos.
- **Disponibilidad**: Sin cambios en el server; solo frontend estático (build con Vite).
- **iHost**: **Cero dependencias nuevas**. Gráficos SVG/CSS hechos a mano como `BillAnalysis`/`DebtAnalysis`/`DebtChart`. Bundle mínimo.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se investigó el estado actual y el prototipo adjunto:

- **Prototipo** `deudas-grafica.html` (archivo local `~/Downloads/Deudas · Gráfica (prototipo).html`):
  - Estructura: `<h1>Deudas</h1>` + `div.stats` (3 tarjetas: Total adeudado, Deudas activas, Libre de deudas) + `div.panel` con header (`h2#title` + segmented `Cronograma | Saldo total`) + `#chart` (SVG) + nota.
  - Vista **cronograma** (`cronograma(W)`): SVG de ancho fijo con scroll horizontal; cada deuda = fila de `row=38px` con nombre (font-weight 600), monto debajo, track `rect` gris (`var(--track)`) de alto 20 y un `rect` coloreado (`d.c` de una paleta `COL`) que va desde hoy (`x0=L`) hasta su fecha de finalización; etiqueta de fecha a la derecha; línea vertical "Hoy" en `x=L` con texto "Hoy"; ticks de año en el eje X (`ticks(...)`: años desde `today+1` hasta `last+1`, step 2 si el rango es >12 años).
  - Vista **saldo total** (`saldo(W)`): SVG con grid horizontal (`total*i/4`, etiquetas abreviadas `k` para >=1000), área bajo la curva con `fill var(--acc) opacity .18`, línea `stroke var(--acc) width 2.5`, puntos (círculos r=6) coloreados por deuda en cada fecha de finalización, y texto "Libre de deudas: {fm(last.end)}" al final. La curva: `[[t0,total], [+d.end,bal], ...]` — en cada fecha de fin baja `bal -= d.a`.
  - **Tooltips** (`show/hide`): div `fixed` con `pointer-events:none`, `background var(--ink)`, opacidad toggle; contenido HTML: `<b>{nombre}</b><br>{monto}<br>Termina: {fm}<br>Restan: {left}` y `Saldo después: {monto}` solo en saldo. Posiciona en `clientX+14/clientY+14` clampado a `innerWidth-250`.
  - **Helper `left(d)`**: años/meses restantes desde hoy; "Este mes" si <1 mes; formato "N años M meses".
  - **Formato money**: `"$"+Math.round(v).toLocaleString("en-US")` (en el proyecto se reemplaza por `formatMoney` del store).
- **Estado actual del código**:
  - `frontend/src/pages/DeudasPage.tsx`: `type TabKey = 'calendario' | 'deudas' | 'analisis' | 'grafica'` (línea 18); tab por defecto `analisis`; render `tab === 'grafica' ? <DebtChart debts={debts} />` (línea 157). Las deudas se cargan con `api.debts.list()` (líneas 38–41). `debts` ya es `Debt[]`.
  - `frontend/src/components/DebtChart.tsx` (de SPEC-100): barras verticales SVG por mes/año de finalización, toggle cantidad/monto, tooltip por barra. **Este componente es el que se reemplaza** por el nuevo diseño del prototipo.
  - `frontend/src/utils/debtMonthlyAggregation.ts` (de SPEC-100): `debtEndMonth(debt)` deriva **mes/año** de finalización = `start_date + installments_total` meses (replica `dueDateForInstallment`). Se extiende o se crea un helper hermano que devuelva la **fecha completa** (año, mes, día) para el cronograma y el saldo.
  - Modelo `Debt` (`frontend/src/types/index.ts` 283–301): `description`, `total`, `installments_total`, `start_date`, `status`, `currency_code`. No tiene `end_date`.
  - i18n `frontend/public/i18n/{es,en}.json`: bloque `deudas.*` con claves del tab `tab_chart`, `chart_*`. Se agregan claves nuevas para "Cronograma", "Saldo total", tarjetas y tooltips.
  - `useCurrencyFormatStore().formatMoney(amount, symbol)` formatea montos con separadores configurados (SPEC-058). `DebtAnalysis.tsx` ya tiene selector de moneda (`currencyFilter`) y paleta de colores `PALETTE` (línea 12) reutilizable.
  - Estilo: cards `bg-card rounded-ios shadow-ios p-4 sm:p-5`; segmented control patrón de `DebtChart` (líneas 68–87, `bg-bg` contenedor, botones `bg-primary text-white` activo); empty state `bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto`.
  - Tokens del tema disponibles: `--color-primary`, `--color-bg`, `--color-card`, `--color-text`, `--color-text-secondary`, `--color-border` (accesibles en CSS/SVG como `rgb(var(--color-*))` o via Tailwind `bg-card`, `text-text-secondary`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Reemplazar `DebtChart.tsx` por el nuevo diseño SVG del prototipo** (Cronograma + Saldo total, hecho a mano) | Cero dependencias, coherente con SPEC-100/054/055, control total de tooltip y darkmode, reutiliza `debts` ya cargados | Más código SVG de bajo nivel | ✅ **Seleccionada** |
| Añadir Recharts/Chart.js | Tooltips y ejes resueltos | +60–100KB, contradice ADR de SPEC-054/055, recursos iHost | ❌ Rechazada |
| Backend agrega `end_date` vía SQL | Dato listo | Cambio de modelo + migración innecesarios; derivación trivial en cliente | ❌ Rechazada |
| Mantener el `DebtChart` actual y agregar un segundo componente | Menos cambios | Duplica gráficas contradictorias en el mismo tab; el requerimiento pide reemplazar | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: La fecha de finalización exacta se deriva, no se persiste**
- **Contexto**: El modelo `Debt` no tiene `end_date`. El prototipo necesita una **fecha** (con día) para las barras del cronograma, la curva de saldo, la tarjeta "Libre de deudas" y el texto "Restan".
- **Decisión**: Extender la derivación existente de SPEC-100: `start_date + installments_total` meses, clampeando al **último día del mes** (replica `dueDateForInstallment` del backend, SPEC-054). Se agrega un helper puro `debtEndDate(debt): { year, month, day } | null` (hermano de `debtEndMonth`) en `frontend/src/utils/debtMonthlyAggregation.ts`. No se hacen llamadas N+1 a `listBills`. Si `installments_total <= 0` o fecha inválida, la deuda se excluye de las vistas y del total.
- **Consecuencias**: Sin migraciones ni cambios de API. El componente solo usa `debts` ya cargados por `DeudasPage`.

**ADR-002: Multi-moneda acotada**
- **Contexto**: El proyecto soporta varias monedas (`currency_id`/`currency_code`). Sumar `total` de monedas distintas en una sola tarjeta o en una sola curva es incorrecto.
- **Decisión**: El cronograma muestra **cada monto con el símbolo de su moneda** (soporta multi-moneda naturalmente). Para el "Total adeudado" y la curva "Saldo total", se agrega un **selector de moneda** (patrón de `DebtAnalysis`) que filtra las deudas; default = moneda de mayor saldo acumulado. "Deudas activas" y "Libre de deudas" se calculan sobre **todas** las deudas (no dependen de moneda).
- **Consecuencias**: El selector es local al componente (no persiste en URL). Se agregan claves i18n para el label "Moneda".

**ADR-003: Reutilización de patrones existentes**
- **Contexto**: El proyecto ya tiene gráficas SVG a mano y segmented controls.
- **Decisión**: El nuevo `DebtChart.tsx` reutiliza: el **segmented control** del `DebtChart` actual (tokens del tema), la **paleta** `PALETTE` de `DebtAnalysis`, `formatMoney` del store de moneda, los **meses cortos** vía `t(\`months.${m}\`)`, y los tokens `rgb(var(--color-*))` para SVG (grid, ejes, "Hoy"). Las tarjetas de resumen usan el patrón `bg-card rounded-ios` con `formatMoney`.
- **Consecuencias**: Coherencia visual total con el proyecto, sin estilos hardcodeados.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DeudasPage.tsx] --tab=grafica--> [DebtChart (REEMPLAZADO)]
        |                                |
        | api.debts.list()               | debtEndDate + helpers (puros)
        v                                v
[SQLite debts]                [utils/debtMonthlyAggregation.ts]
                                           (SVG Cronograma + Saldo total, tooltips, tokens)
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/components/DebtChart.tsx` (modificado — reemplazo del contenido)
- **Responsabilidad**: Renderiza las 3 tarjetas de resumen, el panel con selector "Cronograma"/"Saldo total", el gráfico de la vista activa y los tooltips, usando las deudas reales.
- **Interfaz**: `Props: { debts: Debt[]; currencies: Currency[] }` (las deudas ya cargadas por `DeudasPage` y las monedas del store). Estado interno: `view: 'cronograma' | 'saldo'` (default `cronograma`), `currencyFilter: string` (default moneda de mayor saldo), `hovered` (tooltip).
- **Sub-cálculos** (helpers puros en `utils/debtMonthlyAggregation.ts`):
  - `debtEndDate(debt)` → `{ year, month, day } | null`.
  - `totalAdeudado(debts)` → suma de `total` (opcionalmente filtrado por moneda).
  - `libreDeudas(debts)` → fecha de finalización máxima.
  - `leftText(end, now)` → "Este mes" | "1 año" | "2 años 3 meses" (función pura, testeable).
- **Dependencias**: `useI18nStore`, `useCurrencyFormatStore`, `Icon`, `PALETTE`, helpers, tokens de tema.
- **Ubicación**: `frontend/src/components/DebtChart.tsx`.

#### 4.2.2 `frontend/src/utils/debtMonthlyAggregation.ts` (modificado)
- **Responsabilidad**: agregar `debtEndDate` (fecha exacta) y `leftTime` (tiempo restante). Mantener `debtEndMonth`/`aggregateDebtsByEndMonth` intactos (usados por otros componentes si los hay).
- **Interfaz**:
  ```ts
  export function debtEndDate(debt: Debt): { year: number; month: number; day: number } | null
  export interface LeftTime { years: number; months: number; thisMonth: boolean }
  export function leftTime(end: Date, now: Date): LeftTime
  ```
  `leftTime` devuelve la estructura; el texto "Este mes"/"N años M meses" se arma en el componente con i18n.
- **Dependencias**: ninguna (funciones puras; `leftTime` recibe fechas por inyección para ser testeable).
- **Ubicación**: `frontend/src/utils/debtMonthlyAggregation.ts`.

#### 4.2.3 `DeudasPage.tsx` (modificar mínimamente)
- **Responsabilidad**: mantener la rama `tab === 'grafica'` y pasar las monedas al componente: `<DebtChart debts={debts} currencies={currencies} />`. No tocar los demás tabs ni el `TabKey`.

#### 4.2.4 i18n (`frontend/public/i18n/{es,en}.json`)
- Agregar claves nuevas bajo `deudas.*`: `chart_view_cronograma` ("Cronograma"), `chart_view_saldo` ("Saldo total"), `chart_stat_total` ("Total adeudado"), `chart_stat_active` ("Deudas activas"), `chart_stat_free` ("Libre de deudas"), `chart_ends` ("Termina:"), `chart_remains` ("Restan:"), `chart_balance_after` ("Saldo después:"), `chart_note_cronograma` / `chart_note_saldo` (notas), `chart_currency` ("Moneda"), `chart_this_month` ("Este mes"), reutilizando `months.*` para los nombres de mes.

### 4.3 Modelo de datos

Sin cambios. Se usan los tipos existentes:

```
Entidad: Debt
- description: string (nombre)
- total: float (monto/saldo)
- installments_total: int
- start_date: string (YYYY-MM-DD)
- status: activa | inactiva | finalizada
- currency_code?: string (símbolo de moneda)
- Relaciones: DebtBill (1:N) — no se consulta para este tab
```

### 4.4 APIs / Contratos

Sin endpoints nuevos. Uso de los existentes:
- `GET /api/debts` → `Debt[]` (ya cargadas por `DeudasPage`).

### 4.5 Dependencias

- **Internas**: `DebtChart.tsx` (reemplazo de contenido), `utils/debtMonthlyAggregation.ts` (helpers), `DeudasPage.tsx` (paso de `currencies`), i18n `frontend/public/i18n/{es,en}.json`.
- **Externas**: **ninguna**. SVG/CSS hecho a mano.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: El tab "Gráfica" (`?tab=grafica`) muestra las **3 tarjetas de resumen** (Total adeudado, Deudas activas, Libre de deudas) calculadas con las deudas reales, y el panel con selector "Cronograma" (activo por defecto) y "Saldo total". Los tabs Calendario/Deudas/Análisis quedan intactos.
- [ ] CA-002: Vista **Cronograma**: cada deuda real es una fila con nombre, monto (con símbolo de su moneda), barra horizontal desde "hoy" hasta su fecha de finalización y etiqueta de fecha; ordenadas de la más cercana a la más lejana; línea "Hoy" y ticks de año presentes.
- [ ] CA-003: Vista **Saldo total**: curva/área de balance que baja en cada fecha de finalización, puntos por deuda, grid, y etiqueta "Libre de deudas: {fecha}".
- [ ] CA-004: Los **tooltips** sobre filas y puntos muestran nombre, monto, "Termina: {fecha}", "Restan: {texto relativo}" y (en saldo) "Saldo después: {monto}".
- [ ] CA-005: Si **no hay deudas**, se muestra el estado vacío (patrón EmptyCard) en lugar de tarjetas/gráficos vacíos.
- [ ] CA-006: El selector de moneda filtra el total y la curva de saldo (default moneda de mayor saldo); el cronograma muestra cada monto con su moneda.
- [ ] CA-007: El componente es **responsive** (cronograma con scroll horizontal en móvil) y **legible en modo claro y oscuro** (tokens del tema, sin colores hardcodeados).
- [ ] CA-DARK: El selector de vistas y el selector de moneda usan tokens del tema (`bg-card`, `text-text`/`text-text-secondary`) y se verificó legibilidad en darkmode.

### 5.2 No funcionales

- [ ] CA-NF-001: No se agrega ninguna dependencia npm nueva (package.json sin cambios de deps).
- [ ] CA-NF-002: El build de Vite (`npm run build` en `frontend/`) compila sin errores y sirve el i18n actualizado.

### 5.3 Testing

- **Unit tests**: `debtEndDate` (derivación con clampeo a último día del mes, deudas sin cuotas, valores inválidos) y `leftTimeText` ("Este mes", "1 año", "2 años 3 meses", años sin meses). Ubicación: `frontend/src/utils/debtMonthlyAggregation.test.ts`, con Node nativo (`node --test`, sin dependencias nuevas).
- **Integration tests**: render del tab `grafica` con deudas y sin deudas, alternar vistas y moneda (verificación manual + build).
- **E2E tests**: flujo manual — abrir `/deudas?tab=grafica`, alternar Cronograma/Saldo total, hover de filas/puntos, cambiar moneda, darkmode.
- **Carga/Performance**: con decenas de deudas el gráfico renderiza sin degradación en el iHost (cálculo en cliente).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Helpers `debtEndDate` + `leftTimeText` + unit tests (node --test) | 0.5 día | Ninguna |
| 2 | `DebtChart.tsx`: tarjetas de resumen + selector vistas + SVG Cronograma + SVG Saldo total + tooltips + empty state + selector de moneda | 1.5 días | Fase 1 |
| 3 | Integración en `DeudasPage.tsx` (pasar `currencies`) + i18n es/en | 0.5 día | Fase 2 |
| 4 | Build, pruebas manuales (claro/oscuro, responsive, móvil), QA | 0.5 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Tarjetas de resumen + selector "Cronograma"/"Saldo total" funcionales con deudas reales, tooltips y empty state.
2. **V1.0**: Validación en local, i18n es/en, tests de helpers, multi-moneda, responsive + darkmode verificado.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Fecha de finalización inexacta si la deuda no tiene cuotas generadas | Media | Medio | Derivar con `start_date + installments_total` clampeado al último día (ADR-001), igual que SPEC-100 |
| Tooltip cortado en pantallas pequeñas / se sale del viewport | Media | Bajo | Posicionamiento relativo clampado al viewport; revisión en móvil (scroll horizontal del cronograma) |
| Sumar monedas distintas en el total | Media | Alto | Selector de moneda (ADR-002); cronograma con símbolo por fila |
| Cronograma ancho con scroll horizontal incómodo en móvil | Media | Bajo | `overflow-x:auto` en el contenedor del SVG (como el prototipo) |
| Divergencia de estilos con el resto de Deudas | Baja | Bajo | Reutilizar tokens, `PALETTE` y patrón de segmented control existentes |

## 8. Notas y Referencias

- SPEC-100 (Tab Gráfica en Deudas — implementación previa que se reemplaza en este tab)
- SPEC-054 (Módulo Deudas: `dueDateForInstallment`, calendario)
- SPEC-055 (Análisis de Deudas por Mes; ADR-002: rechazo de librerías de gráficas)
- SPEC-058 (Formato de moneda configurable)
- SPEC-060 (Inputs/selectores con tokens del tema en darkmode)
- Prototipo: `~/Downloads/Deudas · Gráfica (prototipo).html`
- Referencias de implementación: `frontend/src/components/BillAnalysis.tsx`, `frontend/src/components/DebtAnalysis.tsx` (`PALETTE`, selector de moneda), `frontend/src/components/DebtChart.tsx` (actual)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-30 | opencode | Creación inicial de la especificación (draft) |
| 2026-09-30 | opencode | Desarrollo (in_progress): helpers debtEndDate/leftTime + tests, reemplazo de DebtChart.tsx (tarjetas resumen, Cronograma + Saldo total, tooltips, selector de moneda, empty state), integración en DeudasPage, i18n es/en |