---
title: "Progreso visual de cuotas en detalle de deuda"
id: "SPEC-064"
status: "released"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 65
---

# Progreso visual de cuotas en detalle de deuda

**ID**: SPEC-064  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

En la página de detalle de una deuda (`/deudas/:id`, componente `DebtBillsPage`) el usuario ve solo el chip **"60 cuota(s)"** que muestra el total de cuotas de la deuda. No hay ninguna referencia a cuántas de esas cuotas ya pagó, lo que obliga a hacer scroll y contar cuota por cuota (o abrir otra vista) para saber el avance del pago.

Esta spec agrega, en la misma página, el indicador **"X de Y cuotas"** (donde X = cuotas pagadas, Y = total) y un **bloque visual de progreso** (barra de progreso con porcentaje + montos pagado/pendiente) que permite identificar el estado de pago de la deuda de un vistazo, sin scroll. Como la página ya carga todas las cuotas de la deuda (`api.debts.listBills`), el conteo de pagadas se calcula en el frontend: **no requiere cambios de backend, ni migraciones, ni dependencias nuevas**, respetando las restricciones de iHost (bajo consumo de memoria, bundle contenido).

Consideraciones de UI obligatorias: todo input/componente nuevo DEBE usar los tokens del tema (`bg-card`, `text-text`, `text-text-secondary`, `bg-border`, `bg-success`); jamás colores hardcodeados. El bloque se verificará en darkmode (texto y porcentaje legibles). El alcance es **solo la página de detalle**; las cards del listado (`DeudasPage`) quedan sin cambios.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En `DebtBillsPage`, reemplazar el chip existente `{debt.installments_total} cuota(s)` por **"X de Y cuotas"**, donde **X = número de cuotas pagadas** (calculado del array de `bills` cargado) e **Y = `debt.installments_total`**. Ejemplo: si la deuda tiene 60 cuotas y 10 pagadas → **"10 de 60 cuotas"**.
2. **REQ-002**: Agregar un **bloque de progreso visual** prominente en el header de la página (debajo del título/descripción de la deuda, antes del listado de cuotas), que incluya:
   - **Barra de progreso** (track `bg-border`, fill `bg-success`) con ancho = porcentaje de cuotas pagadas sobre el total.
   - **Porcentaje** numérico a la derecha de la barra (ej: "17%").
   - Texto breve bajo la barra indicando **montos**: cuánto se ha pagado y cuánto queda pendiente (en la moneda de la deuda), usando `formatMoney` con el símbolo de la moneda (patrón de `DebtAnalysis`).
3. **REQ-003**: **Estados visuales del progreso según el avance** (deuda finalizada vs en curso):
   - 100% pagado (todas las cuotas `paid`): barra `bg-success` y texto/porcentaje en verde (emerald), comunicando deuda saldada.
   - 0% pagado: barra vacía (track neutral) y texto neutro.
   - Parcial (1-99%): barra `bg-success` con porcentaje en amber/primary para dar sensación de "en progreso".
4. **REQ-004**: Cuando la deuda **no tiene cuotas** (`bills.length === 0`, ej: deuda inactiva o sin cuotas generadas), el bloque muestra 0 pagadas, 0% y los montos en 0, sin romper el layout (no mostrar NaN ni dividir por cero).
5. **REQ-005**: Agregar las **claves i18n** nuevas en `frontend/public/i18n/{es,en}.json` (fuente de verdad, NO `public/i18n/`) para: el conector "de/of", el plural de cuota, y las etiquetas del bloque de progreso (pagado/pendiente).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Gráfico complementario **donut SVG** (pagado vs pendiente) junto a la barra de progreso, reutilizando el patrón de `donutWedge`/`polarToCartesian` de `DebtAnalysis`. Idealmente extraer los helpers a un componente compartido `DonutChart` para reutilizarlo en ambas vistas (refactor opcional de `DebtAnalysis` para usar el mismo componente, sin cambio de comportamiento).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Extender el mismo indicador "X de Y" a las **cards del listado de deudas** (`DeudasPage`), mostrando un mini-progreso en cada card. Fuera de alcance en esta iteración (el usuario lo limitó a la página de detalle).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El conteo y los cálculos del progreso se hacen con `useMemo`/cálculo directo sobre el array ya cargado (volumen pequeño, ≤ ~60 cuotas). Sin renders extra ni llamadas adicionales.
- **Seguridad**: Sin cambios de backend ni nuevos endpoints; se reutiliza la data ya autenticada.
- **Almacenamiento**: Sin cambios de esquema ni migraciones.
- **Disponibilidad**: Sin servicios nuevos; la página ya carga `debt` y `bills`.
- **iHost**: **Cero dependencias nuevas**. Barra de progreso con Tailwind y, si aplica, donut SVG puro (sin librería de gráficos). No agregar peso al bundle.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `frontend/src/pages/DebtBillsPage.tsx` (líneas 74-84): el header muestra tres chips, entre ellos `{debt.installments_total} {t('deudas.installment').toLowerCase()}(s)`. La página ya carga todas las cuotas con `api.debts.listBills(Number(id))` (línea 29) en `bills`, cada una con `status: 'pending' | 'paid'`.
- `frontend/src/types/index.ts` (línea 244 `installments_total`, 260 `installment_number`, 263 `status`) y `Debt` (con `currency_code`, `currency_id`) — toda la data necesaria existe.
- `frontend/src/components/DebtAnalysis.tsx` ya implementa una **barra de progreso** (líneas 218-226) y un **donut SVG** (líneas 14-32 con `polarToCartesian`/`donutWedge`, usadas en 254-267) con la paleta `PALETTE`. Es el patrón visual a reutilizar.
- El store de i18n (`frontend/src/stores/i18nStore.ts`) **no soporta interpolación de parámetros** (`t(key, fallback)` devuelve la cadena literal), por lo que el texto "X de Y" debe componerse en JSX concatenando valores + claves i18n.
- Claves i18n existentes en `frontend/public/i18n/es.json` sección `deudas` (líneas 37-90): `installment` ("Cuota"), `total`, `payment_day`, `pay`, `bills_empty`, `analysis_paid`, etc. No existe clave para "de", plural de cuotas, ni para montos pagado/pendiente en el detalle.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Calcular progreso en el **frontend** con los `bills` ya cargados | Cero backend, cero migraciones, trivial (`filter(status==='paid').length`) | Lógica de conteo en cliente | ✅ Seleccionada |
| Nuevo endpoint backend con conteos (`GET /api/debts/{id}/stats`) | Conteo servido | Handler + storage + tests nuevos; innecesario para ≤60 filas | ❌ Rechazada |
| Usar librería de gráficos (recharts) | Donut listo | Bundle pesado, contra restricciones iHost | ❌ Rechazada |
| **Barra de progreso Tailwind + donut SVG propio** | Zero dependencias, consistente con `DebtAnalysis` | Implementación manual | ✅ Seleccionada |
| Solo barra de progreso (sin donut) | Más simple | Menos "visual" de lo pedido | ❌ Rechazada como P0 (donut queda P1) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Cálculo del progreso en el frontend, sin backend.**
- **Contexto**: La página ya descarga todas las cuotas de la deuda; el número de pagadas es `bills.filter(b => b.status === 'paid').length`.
- **Decisión**: Calcular `paidCount`, `pendingCount`, porcentaje y montos con un `useMemo` sobre `bills`. `Y = debt.installments_total` (autoritativo). Proteger división por cero cuando `Y = 0` o no hay cuotas.
- **Consecuencias**: Sin cambios de backend, migraciones ni tests de Go; menor riesgo. El dato es tan fresco como la última carga de `load()`.

**ADR-002: Barra de progreso + donut con Tailwind y SVG puro, sin librerías.**
- **Contexto**: `DebtAnalysis` ya usa este patrón con éxito; el proyecto exige mínimas dependencias (AGENTS.md).
- **Decisión**: Barra con clases Tailwind de tema (`bg-border`/`bg-success`) y donut SVG con helpers copiados/extraídos de `DebtAnalysis`.
- **Consecuencias**: Bundle contenido, estilo consistente, sin riesgos de breaking changes.

**ADR-003: Texto "X de Y" compuesto en JSX, no con interpolación.**
- **Contexto**: El store i18n no interpola parámetros.
- **Decisión**: Componer `{paidCount} {t('deudas.of')} {Y} {t(plural o singular de cuota)}` con claves i18n para el conector y el plural.
- **Consecuencias**: Cambiar la clave `installment` existente no se toca; se agregan claves nuevas (`deudas.of`, `deudas.installments`).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DebtBillsPage /deudas/:id]  (frontend, sin backend)
   ├─ api.debts.get(id) ──────────────┐
   └─ api.debts.listBills(id) ────────┤  (existente, sin cambios)
                                      v
   [useMemo] paidCount / pendingCount / pct / montosPagado / montosPendiente
                                      v
   Header: "10 de 60 cuotas"  +  [Bloque de progreso: barra + % + montos]  [+ donut P1]
                                      v
   Listado de cuotas (existente, sin cambios)
```

### 4.2 Componentes

#### 4.2.1 `DebtBillsPage` (modificar)
- **Responsabilidad**: Agregar el conteo pagado/total y el bloque de progreso en el header.
- **Interfaz**: Sin cambios de props; se modifica internamente.
- **Dependencias**: `useMemo`, `t` (i18n), `formatMoney` (ya importados), `currencies` (ya importado).
- **Ubicación**: `frontend/src/pages/DebtBillsPage.tsx`.

#### 4.2.2 (P1) `DonutChart` (nuevo, extraído)
- **Responsabilidad**: Donut SVG genérico pagado-vs-pendiente.
- **Interfaz**: props `{ segments: { label, value, color }[], size?, thickness?, centerLabel?, centerValue? }`.
- **Dependencias**: Solo React.
- **Ubicación**: `frontend/src/components/DonutChart.tsx` (los helpers `polarToCartesian`/`donutWedge` se mueven aquí; `DebtAnalysis` puede refactorizarse para usarlo, opcional).

### 4.3 Modelo de datos

Sin cambios. Se usan los modelos existentes:

```
Debt (existente)
- installments_total: number  // Y (total de cuotas)
- currency_code, currency_id  // moneda para formatMoney

DebtBill (existente)
- installment_number: number
- amount: number
- status: 'pending' | 'paid'
```

### 4.4 APIs / Contratos

Sin endpoints nuevos ni modificaciones. Se reutilizan:
- `GET /api/debts/{id}` (ya usado en `DebtBillsPage`)
- `GET /api/debts/{id}/bills` (ya usado en `DebtBillsPage`)

### 4.5 Dependencias

- **Internas**:
  - `frontend/src/pages/DebtBillsPage.tsx` (modificar)
  - `frontend/src/types/index.ts` (solo lectura)
  - `frontend/src/components/DebtAnalysis.tsx` (patrón de barra/donut de referencia; refactor opcional en P1)
  - `frontend/public/i18n/{es,en}.json` (claves nuevas; recordar: fuente de verdad es `frontend/public/i18n/`, luego `npm run build` en `frontend/`)
  - `frontend/src/i18n/{es,en}.json` (bundled dictionaries que usa el store como fallback — verificar si requieren el mismo cambio)
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un usuario en `/deudas/:id` con 60 cuotas y 10 pagadas, entonces el header muestra el texto **"10 de 60 cuotas"** en lugar de "60 cuota(s)".
- [ ] CA-002: Dado el mismo caso, entonces se muestra el **bloque de progreso** con barra al 17%, porcentaje "17%" y los montos pagado/pendiente en la moneda de la deuda.
- [ ] CA-003: Dada una deuda con **todas** las cuotas `paid`, entonces la barra está al 100%, el porcentaje es "100%" y el estado visual es verde (saldada).
- [ ] CA-004: Dada una deuda con **0** cuotas pagadas, entonces la barra está vacía y el porcentaje es "0%" sin errores de división por cero.
- [ ] CA-005: Dada una deuda **sin cuotas** (`bills.length === 0`), entonces el bloque muestra 0 pagadas, 0% y montos en 0, sin romper el layout.
- [ ] CA-006: Dado el cambio de chip, cuando se pagina una cuota desde el listado (acción Pagar), entonces el texto "X de Y" y la barra se **actualizan** (el `onSuccess` de `DebtPayModal` ya recarga `load()`).
- [ ] CA-007: (P1) Dado el bloque de progreso, cuando hay un donut, entonces muestra la proporción pagado vs pendiente y el porcentaje central correcto.
- [ ] CA-DARK: El bloque de progreso usa tokens del tema (`bg-card`, `text-text`, `text-text-secondary`, `bg-border`, `bg-success`) y se verificó legibilidad en darkmode (texto y porcentaje).

### 5.2 No funcionales

- [ ] CA-NF-001: El build de frontend (`npm run build` en `frontend/`) compila sin errores y **sin nuevas dependencias** en `package.json`.
- [ ] CA-NF-002: No se realizan cambios de backend, migraciones ni esquema de DB.
- [ ] CA-NF-003: Las claves i18n nuevas se sirven correctamente desde `/i18n/es.json` y `/i18n/en.json` tras el build.

### 5.3 Testing

- **Unit tests**: Lógica de conteo/progreso (si se extrae a util): paidCount, pct con Y=0, casos 0%/100%/parcial.
- **Integration tests**: Flujo de carga de una deuda + sus cuotas y render del bloque; actualización tras pagar.
- **E2E tests**: Manual local — abrir una deuda con cuotas pagadas y pendientes, verificar texto y barra; pagar una cuota y verificar actualización; verificar deuda finalizada y deuda sin cuotas.
- **Carga/Performance**: Render fluido en iHost (deuda con 60 cuotas); sin llamadas extra.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Cálculo `useMemo` de paidCount/pendingCount/pct/montos en `DebtBillsPage` + reemplazo del chip "X de Y cuotas" | 0.25 día | Ninguna |
| 2 | Bloque de progreso visual (barra `bg-success` + % + montos) con estados 0/parcial/100 | 0.5 día | Fase 1 |
| 3 | Claves i18n (`deudas.of`, `deudas.installments`, progreso pagado/pendiente) en `frontend/public/i18n/{es,en}.json` + `frontend/src/i18n/` si aplica; `npm run build` y verificar `/i18n/es.json` | 0.25 día | Fases 1-2 |
| 4 | (P1) Componente `DonutChart` extraído de `DebtAnalysis` + integración en el bloque | 0.5 día | Fase 2 |
| 5 | Build frontend, validación local (server + evaluación manual del usuario) | 0.25 día | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: Fases 1-3 (texto "X de Y" + barra de progreso con % y montos).
2. **V1.0**: Fase 4 incluida (donut SVG complementario) + validación manual completa.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| División por cero si `installments_total = 0` o sin cuotas | Media | Medio | Guardar pct = 0 cuando Y ≤ 0 o no hay bills; no mostrar NaN |
| El texto "X de Y" se rompe en otro idioma (orden de palabras) | Baja | Bajo | Componer con claves i18n separadas (`deudas.of`, plural) y revisar `en.json` |
| Conflicto con otras sesiones por editar `DebtBillsPage.tsx` | Media | Alto | Trabajar en worktree aislado de la spec (AGENTS.md) |
| i18n editado en el destino equivocado (`public/i18n/`) y perdido en build | Baja | Alto | Editar SIEMPRE `frontend/public/i18n/` (regla AGENTS.md), luego `npm run build` y verificar servido |

## 8. Notas y Referencias

- SPEC-054 (módulo Deudas) — base del módulo; SPEC-055 (Análisis con gráficos) — patrón de barra de progreso y donut SVG.
- `frontend/src/pages/DebtBillsPage.tsx` — página a modificar (chip en líneas 78-80).
- `frontend/src/components/DebtAnalysis.tsx` — referencias: barra (218-226), donut/helpers (14-32, 254-267).
- `frontend/public/i18n/es.json` — sección `deudas` (líneas 37-90) para claves nuevas.
- Regla de i18n (AGENTS.md): fuente de verdad `frontend/public/i18n/`; el build regenera `public/`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación. Requerimiento de usuario: en el detalle de una deuda mostrar "X de Y cuotas" (en vez de "Y cuota(s)") + gráfico de progreso visual del estado de pago sin hacer scroll. Alcance confirmado con el usuario: solo página de detalle (`/deudas/:id`). |
| 2026-09-04 | paulomcnally | Implementación: chip "X de Y cuotas" + bloque de progreso (donut SVG, barra, % y montos pagado/pendiente) en `DebtBillsPage`; nuevo `DonutChart.tsx`; claves i18n `deudas.of`/`installments`/`progress_*`. Fix de Rules of Hooks (`useMemo` movido antes de los early returns) tras crash en Chrome reportado por el usuario. |
| 2026-09-04 | paulomcnally | **Release**: implementación en commit `84e4e4c` (main). Validado por el usuario en local (puerto 8090). Estado `released`. |