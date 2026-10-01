---
title: "Tab Gráfica en Deudas: barras por mes/año de finalización"
id: "SPEC-100"
status: "released"
author: "opencode"
created: "2026-09-30"
updated: "2026-09-30"
github_issue: 103
---

# Tab Gráfica en Deudas: barras por mes/año de finalización

**ID**: SPEC-100  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-30  
**Actualizado**: 2026-09-30

---

## 1. Resumen Ejecutivo

La sección **Deudas** tiene tres tabs (`Calendario`, `Deudas` y `Análisis`). Este requerimiento agrega un **cuarto tab "Gráfica"** que muestra un **gráfico de barras verticales** con las deudas agrupadas por el mes/año en que **finalizan** (se calcula con la fecha de finalización de cada deuda). El gráfico permite alternar entre dos métricas: **cantidad de deudas** que terminan en cada mes (modo por defecto) y **monto total** de esas deudas. Al pasar el cursor sobre una barra, un tooltip muestra el mes/año, la cantidad, el monto total y el nombre de cada deuda de ese mes.

Es importante porque hoy la información de cuándo vencen las deudas está dispersa entre el calendario de cuotas y el análisis mensual, pero no hay una vista consolidada que responda "¿cuántas deudas y por cuánto dinero terminan cada mes?". El gráfico da esa lectura de un vistazo.

Resultado esperado: un tab adicional que reutiliza el patrón de UI existente (cards, tabs con query params, tokens de tema) y una librería de gráficos hecha a mano en SVG/CSS — coherente con `BillAnalysis`/`DebtAnalysis` — sin agregar dependencias nuevas, respetando las restricciones de recursos del iHost.

**Consideraciones iHost**: el gráfico se renderiza en el cliente (navegador), no en el server. No se agregan dependencias npm (memoria de build y bundle). El cálculo de agrupación es una función pura en el frontend. No hay cambios de esquema SQLite ni de API backend.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar el tab "Gráfica" (clave `tab=grafica`) a la sección Deudas, junto a los tabs actuales, sin modificar la lógica ni los tabs existentes.
2. **REQ-002**: Mostrar un gráfico de **barras verticales** con meses agrupados por año (ej. "Ene 2026", "Feb 2026", ...), ordenados cronológicamente **de la fecha más cercana a la más lejana**, mostrando solo los meses que tengan **al menos una deuda** que finalice en ellos.
3. **REQ-003**: La fecha de finalización de cada deuda se calcula con la fecha de vencimiento de su última cuota (`debt_bills.due_date` máxima de la deuda), o derivada de `start_date` + `installments_total` cuando no haya cuotas generadas. Ver ADR-001.
4. **REQ-004**: El eje Y tiene un **selector (toggle) con dos modos**:
   - **a)** `Cantidad de deudas` (por defecto): cuántas deudas terminan en ese mes.
   - **b)** `Monto total`: suma de los montos (`total`) de las deudas que terminan en ese mes.
5. **REQ-005**: **Tooltip** al pasar el cursor sobre una barra que muestre: mes/año, cantidad de deudas, monto total y el **nombre (description) de cada deuda** de ese mes.
6. **REQ-006**: Si **no hay deudas**, mostrar un **mensaje de estado vacío** en lugar de un gráfico en blanco (patrón EmptyCard existente).
7. **REQ-007**: El gráfico debe ser **responsive**, soportar **modo claro/oscuro** (el proyecto ya lo usa) y tener etiquetas y ejes legibles.
8. **REQ-008**: Extraer la **agrupación por mes/año** a una **función separada y reutilizable** (ver ADR-002).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: La selección del tab (y del modo del toggle) se refleja en la URL (`?tab=grafica`), siguiendo el patrón ADR-004 de SPEC-054/055. El modo del toggle puede persistirse solo en estado local del componente.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Los meses con deudas que ya están totalmente pagadas podrían atenuarse visualmente (baja prioridad, no bloquea release).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El gráfico se calcula 100% en el cliente a partir de `api.debts.list()` + cuotas ya cargadas. Sin llamadas N+1 nuevas. La función de agrupación es O(n·m).
- **Seguridad**: El tab usa las APIs autenticadas existentes; sin datos sensibles nuevos.
- **Almacenamiento**: Sin cambios de esquema ni datos nuevos.
- **Disponibilidad**: Sin cambios en el server; solo frontend estático (build con Vite).
- **iHost**: **Cero dependencias nuevas**. Gráfico SVG/CSS hecho a mano como `BillAnalysis`/`DebtAnalysis`. Bundle mínimo.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se investigó la construcción actual de la sección Deudas:

- **Página principal**: `frontend/src/pages/DeudasPage.tsx`. Tabs inline (no hay componente `Tabs` compartido). `type TabKey = 'calendario' | 'deudas' | 'analisis'` (línea 17); tab por defecto `analisis` (línea 26); barra de tabs en líneas 131–152 con `{ key, label, icon }`; render por rama en líneas 154–272. El switch usa `?tab=` vía `useSearchParams` (ADR-004 de SPEC-054).
- **Modelo de datos**: `Debt` (`frontend/src/types/index.ts` líneas 283–301) tiene `total`, `installments_total`, `start_date`, `status`, `description`, pero **no tiene `end_date`**. Las cuotas `DebtBill` (líneas 303–318) tienen `due_date`, `amount`, `status`, `debt_description`. El backend genera cuotas con `dueDateForInstallment(start, k, paymentDay)` (`internal/services/debt.go` líneas 195–204): vencimiento de la cuota k = `start + k` meses, clampeado al último día del mes.
- **APIs**: `api.debts.list()` → `GET /api/debts` (lista de `Debt[]`); `api.debts.listBills(debtId)` → `GET /api/debts/{id}/bills`; `api.debts.billsByMonth(y, m)` → `GET /api/debt-bills?year=&month=` (usado por Calendario/Análisis).
- **Librería de gráficas**: **no existe ninguna** en `frontend/package.json`. SPEC-054 (ADR) y SPEC-055 (ADR-002) rechazaron explícitamente agregar dependencias de gráficos por recursos del iHost. Existen gráficos hechos a mano: `DonutChart.tsx` (SVG reutilizable), `DebtAnalysis.tsx` (donut SVG inline), `BillAnalysis.tsx` (línea/área SVG + `BarChart` CSS inline, líneas 356–466).
- **Fecha/meses**: sin librería de fechas. Meses en español vía i18n `months.{1..12}` (`t(\`months.${month}\`)`), patrón usado en `DebtAnalysis.tsx` línea 159 y `DebtCalendar.tsx` línea 90.
- **i18n**: fuente de verdad `frontend/public/i18n/{es,en}.json`, bloque `deudas.*`. Claves de tabs: `deudas.tab_calendar`, `deudas.tab_debts`, `deudas.tab_analysis`. `npm run build` sincroniza a `frontend/src/i18n/`.
- **Estilo**: cards `bg-card rounded-ios shadow-ios p-4 sm:p-5`; empty state `bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto`; iconos vía `<Icon name="chart" />`; darkmode con clase `.dark`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **SVG/CSS hecho a mano** (reutilizar patrón `BillAnalysis`/`DebtAnalysis`) | Cero dependencias, coherente con SPEC-054/055, bundle mínimo, control total de darkmode y tooltip | Más código de bajo nivel (escalas, tooltip) | ✅ **Seleccionada** |
| Recharts | Tooltips/ejes ya resueltos, integración React | +100KB gzip, contradice ADR de SPEC-054/055, recursos iHost | ❌ Rechazada |
| Chart.js | Ligero (~60KB), canvas, tooltips incluidos | Dependencia nueva (contradice SPEC-054/055), API imperativa menos idiomática en React | ❌ Rechazada |
| Backend agrega `end_date` calculado vía SQL | Datos listos desde la API | Cambio de modelo + migración innecesarios; el cálculo es trivial en el cliente | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: La "fecha de finalización" de una deuda se deriva, no se persiste**
- **Contexto**: El modelo `Debt` no tiene `end_date`. El requerimiento pide agrupar por mes de finalización.
- **Decisión**: La fecha de finalización de una deuda se deriva de `start_date + installments_total` meses, replicando la lógica del backend `dueDateForInstallment` (SPEC-054): el último mes de la deuda es el mes de inicio más la cantidad de cuotas. No es necesario hacer N+1 llamadas a `api.debts.listBills` para obtener el `due_date` máximo: el resultado del mes/año es idéntico porque el backend genera las cuotas de forma determinista (`start + k` meses). Si `installments_total` es 0 o la fecha no es válida, la deuda se excluye del gráfico.
- **Consecuencias**: Sin migraciones ni cambios de API. La función de agrupación recibe una deuda y devuelve el mes/año de finalización derivado; el componente solo usa `debts` ya cargados por `DeudasPage` (`api.debts.list()`), sin llamadas extra.

**ADR-002: Agrupación en función pura reutilizable**
- **Contexto**: REQ-008 exige extraer la agrupación mes/año.
- **Decisión**: Crear `frontend/src/utils/debtMonthlyAggregation.ts` con una función pura `aggregateDebtsByEndMonth(debts, endDateOf)` que devuelva buckets `{ year, month, label, count, total, debts: Debt[] }` ordenados de la fecha más cercana a la más lejana, filtrando meses sin deudas. El `label` ("Ene 2026") se arma con el mes corto + año. Reutilizable por el gráfico y por cualquier vista futura.
- **Consecuencias**: Lógica testeable con unit tests (Node `node --test`, sin dependencias nuevas). El componente `DebtChart` solo renderiza.

**ADR-003: Gráfico de barras SVG hecho a mano**
- **Contexto**: No hay librería de gráficas y las specs previas la rechazaron.
- **Decisión**: Componente `frontend/src/components/DebtChart.tsx` con SVG (rect por barra, escalas calculadas con el ancho disponible vía `viewBox` responsive, etiquetas de eje con `t(\`months.${month}\`)`) y **tooltip CSS/HTML posicionado** (no SVG `<title>`): al hover de una barra se muestra un tooltip absoluto con mes/año, cantidad, monto total y lista de nombres de deuda. Usa tokens de tema (`--color-text`, `--color-card`, `--color-primary`) para claro/oscuro.
- **Consecuencias**: Sin dependencias. El tooltip requiere manejo de hover/posición en el componente.

**ADR-004: Toggle de métricas en estado local**
- **Contexto**: REQ-004 pide dos modos en el eje Y.
- **Decisión**: Un toggle (`Cantidad de deudas` / `Monto total`) en estado local del componente `DebtChart` (o del tab), con estilos de tokens del tema. La altura de cada barra se escala según la métrica activa. Monto formateado con `useCurrencyFormatStore().formatMoney`.
- **Consecuencias**: Simplicidad. No persiste en URL (los tabs sí, vía `?tab=`).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DeudasPage.tsx] --tab=grafica--> [DebtChart (nuevo componente)]
        |                                |
        | api.debts.list()               | aggregateDebtsByEndMonth + debtEndMonth (puras)
        v                                v
[SQLite debts]                [utils/debtMonthlyAggregation.ts]
                                          (SVG + tooltip, tokens del tema)
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/components/DebtChart.tsx` (nuevo)
- **Responsabilidad**: Renderiza el gráfico de barras verticales con el toggle de métricas y el tooltip.
- **Interfaz**: `Props: { debts: Debt[] }` (las deudas ya cargadas por `DeudasPage`). Internamente carga las cuotas necesarias para derivar la fecha de finalización (ADR-001) y delega el cálculo a la función de agrupación.
- **Dependencias**: `useI18nStore`, `useCurrencyFormatStore`, `aggregateDebtsByEndMonth`/`debtEndMonth`, `Icon`, tokens de tema.
- **Ubicación**: `frontend/src/components/DebtChart.tsx`.

#### 4.2.2 `frontend/src/utils/debtMonthlyAggregation.ts` (nuevo)
- **Responsabilidad**: Función pura `aggregateDebtsByEndMonth` que agrupa deudas por mes/año de finalización.
- **Interfaz**:
  ```ts
  export type MonthlyBucket = {
    year: number
    month: number // 1-12
    label: string // "Ene 2026"
    count: number
    total: number
    debts: Debt[]
  }
  // Deriva el mes/año de finalización de una deuda (start_date + installments_total meses).
  export function debtEndMonth(debt: Debt): { year: number; month: number } | null
  export function aggregateDebtsByEndMonth(
    debts: Debt[],
    endDateOf: (d: Debt) => { year: number; month: number } | null,
    monthLabel: (month: number) => string // ej: (m) => t('months.'+m).slice(0,3)
  ): MonthlyBucket[]
  ```
  Orden descendente por fecha (más cercana primero), solo buckets con `count > 0`.
- **Dependencias**: ninguna (función pura; recibe el cálculo de fecha por inyección para ser testeable).
- **Ubicación**: `frontend/src/utils/debtMonthlyAggregation.ts`.

#### 4.2.3 `DeudasPage.tsx` (modificar)
- **Responsabilidad**: agregar `'grafica'` a `TabKey` (línea 17), una entrada `{ key: 'grafica', label: t('deudas.tab_chart'), icon: 'bar' }` en la barra de tabs, y una rama de render `tab === 'grafica'` → `<DebtChart debts={debts} />`. No tocar la lógica de los tabs existentes.
- **Icono**: se agrega un icono `bar` (barras verticales) a `frontend/src/components/Icons.tsx` para diferenciarlo del icono `chart` del tab Análisis.

### 4.3 Modelo de datos

Sin cambios. Se usan los tipos existentes:

```
Entidad: Debt
- total: float (monto)
- installments_total: int
- start_date: string (YYYY-MM-DD)
- status: activa | inactiva | finalizada
- description: string (nombre)
- Relaciones: DebtBill (1:N)

Entidad: DebtBill
- debt_id, due_date: string (YYYY-MM-DD), amount, status
```

### 4.4 APIs / Contratos

Sin endpoints nuevos. Uso de los existentes:
- `GET /api/debts` → `Debt[]`
- `GET /api/debts/{id}/bills` → `DebtBill[]` (para derivar fecha de finalización de deudas activas)

### 4.5 Dependencias

- **Internas**: `DeudasPage.tsx` (tab + rama), `DebtChart.tsx` (nuevo), `utils/debtMonthlyAggregation.ts` (nuevo), i18n `frontend/public/i18n/{es,en}.json` (claves nuevas).
- **Externas**: **ninguna**. Gráfico SVG/CSS hecho a mano.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: El tab "Gráfica" aparece en la sección Deudas y al seleccionarlo la URL queda `?tab=grafica` y se renderiza el gráfico, sin romper los tabs Calendario/Deudas/Análisis.
- [ ] CA-002: El gráfico muestra barras verticales con etiquetas "Ene 2026", "Feb 2026", ... agrupadas por año, ordenadas de la fecha más cercana a la más lejana, mostrando solo meses con al menos una deuda que finalice en ellos.
- [ ] CA-003: El toggle del eje Y alterna entre "Cantidad de deudas" (por defecto) y "Monto total"; al cambiar, la altura de las barras se recalcula con la métrica seleccionada.
- [ ] CA-004: Al pasar el cursor sobre una barra, el tooltip muestra mes/año, cantidad de deudas, monto total y el nombre de cada deuda de ese mes.
- [ ] CA-005: Si no hay deudas, se muestra un mensaje de estado vacío (patrón EmptyCard) en lugar de un gráfico en blanco.
- [ ] CA-006: El gráfico es responsive (se adapta al ancho del contenedor en móvil y desktop) y legible en modo claro y oscuro (tokens del tema, sin colores hardcodeados).
- [ ] CA-007: La agrupación mes/año está extraída en una función separada reutilizable con unit tests.
- [ ] CA-DARK: Cualquier toggle/select del tab usa tokens del tema (`bg-card`, `text-text`/`text-text-secondary`) y se verifica legibilidad en darkmode.

### 5.2 No funcionales

- [ ] CA-NF-001: No se agrega ninguna dependencia npm nueva (package.json sin cambios de deps).
- [ ] CA-NF-002: El build de Vite (`npm run build` en `frontend/`) compila sin errores y sirve el i18n actualizado.

### 5.3 Testing

- **Unit tests**: `debtEndMonth` y `aggregateDebtsByEndMonth` (derivación, agrupación, orden, filtro de meses sin deudas, label de mes/año). Ubicación: `frontend/src/utils/debtMonthlyAggregation.test.ts`, corridos con Node nativo (`node --test`, sin dependencias nuevas).
- **Integration tests**: render del tab `grafica` en `DeudasPage` con deudas y sin deudas (verificación manual + build).
- **E2E tests**: flujo manual — abrir `/deudas?tab=grafica`, alternar métricas, hover de barra.
- **Carga/Performance**: el gráfico con decenas de deudas renderiza sin degradación en el iHost (cálculo en cliente).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `debtEndMonth` + `aggregateDebtsByEndMonth` + unit tests (node --test) | 0.5 día | Ninguna |
| 2 | `DebtChart.tsx`: SVG barras + toggle + tooltip + empty state | 1 día | Fase 1 |
| 3 | Integración en `DeudasPage.tsx` (tab `grafica`) + i18n | 0.5 día | Fase 2 |
| 4 | Build, pruebas manuales (claro/oscuro, responsive), QA | 0.5 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Tab "Gráfica" funcional con barras por mes/año, toggle cantidad/monto, tooltip y empty state.
2. **V1.0**: Validación en local, i18n es/en, tests de la función de agrupación, responsive + darkmode verificado.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Fecha de finalización inconsistente si la deuda no tiene cuotas generadas | Media | Medio | Derivar con `start_date + installments_total` replicando `dueDateForInstallment` (ADR-001) |
| Tooltip cortado en pantallas pequeñas | Media | Bajo | Posicionamiento relativo con clamping al viewport; revisión en móvil |
| Etiquetas del eje X ilegibles con muchos meses | Media | Bajo | Rotar/espaciar etiquetas o abreviar; verificar con muchos buckets |
| Divergencia de estilos con el resto de Deudas | Baja | Bajo | Reutilizar tokens y patrones de `BillAnalysis`/`DebtAnalysis` |

## 8. Notas y Referencias

- SPEC-054 (Módulo Deudas + Calendario, ADR-003/004 de generación de cuotas y tabs por query param)
- SPEC-055 (Análisis de Deudas por Mes, ADR-002: rechazo de librerías de gráficas)
- SPEC-081 (Fechas de emisión y vencimiento en facturas de servicios y deudas)
- Referencia de implementación de barras: `frontend/src/components/BillAnalysis.tsx` (`BarChart`, líneas 436–466) y `frontend/src/components/DebtAnalysis.tsx`

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-30 | opencode | Creación inicial de la especificación (draft) |
| 2026-09-30 | opencode | Desarrollo completo (in_progress): función de agrupación + tests, DebtChart, tab en DeudasPage, icono bar, i18n |
| 2026-09-30 | opencode | Release: merge a main + issue #103 cerrado (spec/released) |