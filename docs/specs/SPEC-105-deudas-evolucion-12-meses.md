---
title: "Evolución de deudas en los últimos 12 meses: gráfico comparativo y totales"
id: "SPEC-105"
status: "in_progress"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 108
---

# Evolución de deudas en los últimos 12 meses: gráfico comparativo y totales

**ID**: SPEC-105  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

La sección **Deudas** permite hoy ver la deuda del mes (Análisis), el calendario de cuotas, el listado de deudas y una pestaña **Gráfica** con dos vistas proyectadas al futuro ("Cronograma" y "Saldo total"). No existe, sin embargo, una vista **histórica** que responda a la pregunta más básica de seguimiento financiero: *¿mi deuda total está bajando o subiendo mes a mes?*

Esta spec agrega una **tercera vista "Últimos 12 meses"** dentro del tab Gráfica (`?tab=grafica`), con un gráfico de barras simple y un bloque de **números totales** al pie: deudas abiertas al inicio del período, deudas abiertas actuales, variación en cantidad, deudas cerradas en el período y saldo pendiente actual como referencia. La métrica elegida por el usuario es la **cantidad de deudas abiertas al cierre de cada mes**, no el dinero: una deuda cuenta desde su fecha de inicio y **deja de contar desde el mes en que se paga/cierra o se elimina**, para que el gráfico no arrastre deudas ya pagadas ni pierda la historia de las que se borraron. Los montos se evitan como métrica principal porque la mora, los intereses y los ajustes pueden cambiarlos.

El cálculo se hace en el backend con una consulta agregada liviana (una fila por deuda más el último pago registrado) y se expone en un endpoint autenticado nuevo; el frontend solo dibuja SVG y formatea. No hay cambios de esquema, no se agregan dependencias npm y no se hacen llamadas N+1.

**Consideraciones iHost**: el agregado es O(12 × n) sobre todas las deudas (incluidas las eliminadas lógicamente, decenas como máximo), con dos consultas SQL y memoria despreciable. El gráfico es SVG nativo renderizado en el navegador (mismo patrón que `DebtChart`/`DebtAnalysis`), sin librerías de charts.

**Consideraciones de UI (SPEC-060/063)**: la vista no agrega formularios; el selector de moneda y el segmented control usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y deben verificarse en darkmode. No se crean páginas de detalle ni links "← Título".

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar una **tercera vista "Últimos 12 meses"** al segmented control del tab Gráfica de Deudas, junto a "Cronograma" y "Saldo total", sin modificar el contenido ni el comportamiento de esas dos vistas ni de los demás tabs (Análisis, Calendario, Deudas). La vista por defecto sigue siendo **Cronograma**; "Últimos 12 meses" se activa manualmente. La vista debe poder abrirse **aunque todas las deudas estén cerradas**: el early-return actual de `DebtChart` basado en `items.length === 0` (deudas con `pending_count > 0`) debe volverse condicional a las vistas Cronograma/Saldo, no aplicar a la vista histórica.
2. **REQ-002**: La vista muestra un **gráfico de barras verticales** con **12 puntos mensuales**, del mes más antiguo al mes actual (incluido). Cada barra representa la **cantidad de deudas abiertas al cierre de ese mes** en la moneda seleccionada. La altura de la barra es un conteo entero, no un monto.
3. **REQ-003**: Una deuda **cuenta como abierta** en un mes `m` si su `start_date` es anterior o igual al cierre de `m` y **no estaba cerrada** a esa fecha. Se considera cerrada cuando:
   - (a) fue **eliminada lógicamente** (`deleted_at` no nulo) y la eliminación ocurrió en `m` o antes;
   - (b) está `finalizada` y su fecha de cierre (REQ-004) ocurrió en `m` o antes;
   - (c) **no le quedan cuotas pendientes y tiene al menos una cuota pagada** (`pending_count = 0` y `paid_count > 0`), aunque su estado siga siendo `activa` o `inactiva`, y su fecha de cierre (REQ-004) ocurrió en `m` o antes. Esto cubre deudas completas que quedaron con estado desactualizado (SPEC-104 solo finaliza automáticamente las `activa`).
   Las deudas cerradas o eliminadas dejan de contar **desde el mes de su cierre/eliminación inclusive**, pero **nunca desaparecen retroactivamente**: siguen sumando en todos los meses anteriores.
4. **REQ-004**: La **fecha de cierre** se determina así:
   - Si `deleted_at` no es NULL (deuda eliminada): `closing_date = deleted_at`, sin importar el estado.
   - Si no, para una deuda cerrada por (b) o (c) de REQ-003: (a) `last_paid_at` (que SPEC-104 ya calcula como `MAX(paid_at)` de cuotas no eliminadas); (b) `updated_at` de la deuda; (c) fecha de vencimiento de la última cuota derivada de `start_date + installments_total`.
   - Si la deuda está abierta (tiene cuotas pendientes, o no tiene cuotas pagadas ni pendientes) y no fue eliminada, no tiene fecha de cierre y sigue contando hasta el mes actual.
   Las fechas de pago y de eliminación se comparan por **prefijo `YYYY-MM`** (`substr(...,1,7)`), por los tres formatos históricos de `paid_at`. Se documenta en tooltip/nota que es una reconstrucción (ver ADR-001 y ADR-006).
5. **REQ-005**: La serie es **histórica e inmutable hacia atrás**: si una deuda se paga/finaliza hoy, **sigue contando en todos los meses anteriores a su fecha de cierre** y deja de contar solo desde el mes de cierre inclusive. Pagar o cerrar una deuda hoy **nunca** debe borrar ni alterar las barras pasadas.
6. **REQ-006**: La **métrica principal es la cantidad de deudas**, no el dinero. Los montos no definen las barras ni los totales principales, porque la mora, los intereses o los ajustes pueden cambiar el valor de una deuda con el tiempo.
7. **REQ-007**: Mostrar **números totales** debajo o encima del gráfico, en tarjetas:
   - **Deudas abiertas al inicio**: cantidad al cierre del mes más antiguo de la ventana.
   - **Deudas abiertas actuales**: cantidad al cierre del mes en curso.
   - **Variación**: diferencia de cantidad (`actual − inicio`) y porcentual, con signo y color semántico (verde si bajó, rojo si subió, neutro si no cambió).
   - **Deudas cerradas en el período**: cantidad de deudas cerradas o eliminadas dentro de la ventana.
   - **Saldo pendiente actual (referencia)**: suma de `pending_amount` de las deudas abiertas en el mes actual, en la moneda seleccionada, con nota de que es un dato de referencia y puede variar por mora/intereses.
8. **REQ-008**: **Tooltip por barra** con: nombre del mes y año, cantidad de deudas abiertas al cierre y cantidad de deudas cerradas o eliminadas en ese mes. En móvil, el tooltip se posiciona clampeado al viewport (mismo patrón que `DebtChart`).
9. **REQ-009**: **Selector de moneda** (solo si hay más de una moneda con actividad). El default es la moneda con mayor `pending_amount` actual (mismo criterio que SPEC-104). El gráfico, los totales y el saldo de referencia corresponden a la moneda seleccionada.
10. **REQ-010**: **Estado vacío** con el patrón existente (card centrada, ícono, título y subtítulo) cuando el endpoint devuelve `currencies: []` (sin actividad en la ventana) o cuando la moneda seleccionada no tiene puntos. El estado vacío se basa en los datos de historia, **no** en el listado de deudas no eliminadas: si todas las deudas se eliminaron dentro de la ventana, la vista debe mostrar su serie.
11. **REQ-011**: Endpoint autenticado **`GET /api/debts/history?months=12`** que devuelve la serie mensual de conteos, los totales y el saldo pendiente actual de referencia para todas las monedas en una sola respuesta (ver §4.4), evitando N+1 y refetch al cambiar de moneda.
### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: **Tabla de datos mensuales** colapsable/de apoyo (debajo del gráfico) con columnas: mes, deudas abiertas y deudas cerradas/eliminadas. Sirve como respaldo numérico accesible y para lectores de pantalla.
2. **REQ-013**: Eje Y con 4–5 líneas de grilla y **etiquetas enteras de cantidad** (sin abreviaturas monetarias); eje X con etiquetas de mes corto en el idioma activo (`months.N`), mostrando todos los meses en desktop y alternando en móvil si el ancho es insuficiente.
3. **REQ-014**: Resaltar visualmente la barra del **mes actual** (color primario pleno) frente a los meses anteriores (primario con menor opacidad), y marcar con línea punteada el nivel del **promedio del período**.
4. **REQ-015**: Nota explicativa bajo el gráfico: qué se cuenta, qué significa "cerrada" (finalizada, sin cuotas pendientes con pagos, o eliminada), que las cifras son una reconstrucción a partir del estado actual y de las fechas de pago, y que el saldo pendiente mostrado es solo de referencia y puede variar por mora o intereses.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Permitir consultar períodos de **6 o 24 meses** además de 12 mediante un selector (el endpoint acepta `months` entre 1 y 24).
2. **REQ-017**: Indicador de **mejor mes** (mayor reducción respecto del mes anterior) en la nota o tooltip.

### 2.4 Cambio solicitado durante la evaluación (P0)

1. **REQ-018**: Cambiar la vista **Cronograma** de barras horizontales a **barras verticales**: una columna por deuda, ordenadas por fecha de finalización, con la altura de la barra representando el tiempo restante hasta esa fecha. Debe verse bien en móvil, tablet y desktop: tamaño de fuente legible, etiquetas rotadas y scroll horizontal solo dentro de la tarjeta cuando no entren las columnas. Para que el scroll interno funcione, el contenedor del layout (`DashboardLayout`) debe permitir encogerse (`min-w-0` en el wrapper y el `<main>`), evitando que un gráfico ancho ensanche toda la página en móvil. Aplica solo a Cronograma; las vistas Saldo total e histórica no cambian.

### 2.5 Requerimientos No Funcionales

- **Rendimiento**: respuesta del endpoint < 50 ms con hasta 500 deudas y 24 meses; cálculo O(24 × n); sin N+1. Máximo 2 consultas SQL por request.
- **Seguridad**: endpoint bajo `authMiddleware`, igual que el resto de `/api/debts`; sin datos sensibles nuevos.
- **Almacenamiento**: **sin migraciones ni cambios de esquema**; se leen `debts`, `debt_bills` y `currencies` existentes.
- **Disponibilidad**: cambio aditivo; no altera endpoints ni respuestas existentes.
- **iHost**: cero dependencias nuevas (backend stdlib, frontend SVG); payload acotado (≤ 24 meses × monedas); sin proceso en background.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Estado actual del tab Gráfica** (`frontend/src/components/DebtChart.tsx`, SPEC-101/102/104): segmented control con `view: 'cronograma' | 'saldo'`; tres tarjetas de resumen; SVG con `viewBox` de ancho de diseño 720 y `min-w` para scroll en pantallas chicas; tooltip flotante clampeado; selector de moneda derivado de `debts` y default por mayor saldo. **Desde SPEC-104** solo grafica deudas con `pending_count > 0`, usa `pending_amount` como saldo y `debtPendingEndDate` como fin proyectado; si no queda ninguna deuda abierta, hace un early-return al estado vacío antes de renderizar el panel. La tercera vista debe convivir con ese comportamiento (REQ-001).
- **Página y datos** (`frontend/src/pages/DeudasPage.tsx`): carga `api.debts.list()` una sola vez y pasa `debts` + `currencies` a `DebtChart`. **Desde SPEC-104**, cada `Debt` incluye campos calculados `pending_count`, `pending_amount`, `paid_count`, `paid_amount`, `next_due_date`, `last_due_date` y `last_paid_at` (este último = `MAX(paid_at)` de cuotas no eliminadas). No hay serie histórica mensual en el cliente.
- **Modelo** (`migrations/0024_create_debts_and_debt_bills.up.sql`, `0030_add_bills_debt_bills_dates.up.sql`):
  - `debts`: `total`, `start_date`, `installments_total`, `payment_day`, `status` (`activa`/`inactiva`/`finalizada`), `deleted_at`, `created_at`, `updated_at`.
  - `debt_bills`: `debt_id`, `installment_number`, `due_date`, `amount`, `status` (`pending`/`paid`), `paid_at`, `deleted_at`.
  - No existe tabla de historial de estados ni snapshots mensuales. `updated_at` se actualiza en cada edición y al marcar la deuda como finalizada.
  - **Formatos heterogéneos de `paid_at`** (hallazgo de SPEC-104): conviven `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD` y `YYYY-MM-DD HH:MM:SS +0000 UTC`. Cualquier comparación de fechas de pago debe hacerse por prefijo `YYYY-MM` (`substr(paid_at,1,7)`), sin parseo estricto en Go.
- **Semántica de estados** (`internal/services/debt.go`, SPEC-104 released en `main`): `finalizada` no genera cuotas; `activa` sí; `inactiva` no genera pero conserva la deuda. **Ya existe finalización automática**: al pagar la última cuota pendiente la deuda pasa a `finalizada`, y `ReconcileDebtBills` corrige las deudas completas que quedaron activas. Por lo tanto, el estado `finalizada` es confiable y la fecha de cierre se deriva de `MAX(paid_at)` (`last_paid_at`).
- **Rutas y capas** (`internal/api/routes.go`, `internal/api/debt_handlers.go`, `internal/services/debt.go`, `internal/storage/debt.go`): patrón handler → service → storage, con `respondJSON` y errores estándar. El nuevo endpoint sigue ese patrón.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Calcular en el frontend con `debts` + `GET /api/debts/{id}/bills` por deuda | Sin cambios de backend | N+1 requests (una por deuda) en el iHost; lógica de cierre duplicada en el cliente; lento con muchas deudas | ❌ Rechazada |
| B. Endpoint agregado `GET /api/debts/history` | Una sola llamada, 2 queries, testable, sin dependencias; centraliza la semántica de cierre | Agrega superficie de API | ✅ Seleccionada |
| C. Tabla de snapshots mensuales (`debt_monthly_snapshots`) | Historia exacta e inmutable | Cambio de esquema (política del usuario: no modificar esquema de deudas), proceso de escritura mensual, datos históricos inexistentes al inicio | ❌ Rechazada |
| D. Sumar cuotas impagas con `due_date <= fin de mes` | Refleja pagos parciales | El usuario pidió explícitamente no basar el gráfico en cuotas, sino en el estado/total de las deudas; mezcla cuotas futuras y deudas cerradas | ❌ Rechazada |
| E. Sumar `total` de deudas con `status != 'finalizada'` para todos los meses | Trivial | No distingue cuándo se cerró cada deuda: una deuda cerrada hoy desaparecería también de los meses pasados, falseando la historia | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Reconstruir la fecha de cierre a partir de la eliminación, los pagos y `updated_at`
- **Contexto**: no hay historial de cambios de estado. Para saber en qué mes dejó de contar una deuda finalizada o eliminada hace falta una fecha de cierre, y las deudas eliminadas no deben borrar su pasado.
- **Decisión**: si `deleted_at` no es NULL, `closing_date = deleted_at` (prioridad absoluta, incluso si estaba `activa`). Si no, para `finalizada`: `last_paid_at` (calculado por SPEC-104); si es NULL, `updated_at`; si tampoco fuera válido, el vencimiento de la última cuota (`start_date + installments_total`). La deuda deja de contar si el mes de `closing_date` es menor o igual al mes evaluado. La consulta de historia **incluye** deudas con `deleted_at` no nulo.
- **Consecuencias**: historia completa sin migraciones: eliminar una deuda hoy conserva sus barras de todos los meses anteriores y solo la excluye desde el mes de borrado. Si el usuario finaliza una deuda sin registrar el pago de la última cuota, la fecha de cierre será la de la edición (`updated_at`), que puede ser posterior al pago real. Se documenta en la nota de la vista. Si el usuario paga todas las cuotas pero **no** marca la deuda como `finalizada`, la deuda igual deja de contar por REQ-003(c), usando `last_paid_at` como cierre (ver ADR-003).

**ADR-002**: La métrica es la cantidad de deudas abiertas al cierre de cada mes
- **Contexto**: el usuario descartó los montos como métrica principal porque la mora, los intereses y los ajustes cambian el valor de una deuda con el tiempo; un gráfico de dinero puede mostrar subas o bajas que no reflejan la realidad del compromiso.
- **Decisión**: cada barra cuenta **deudas**, no pesos. El gráfico baja cuando una deuda pasa a pagada/cerrada y sube cuando se registra una deuda nueva. El monto pendiente actual se muestra solo como dato de referencia en una tarjeta, no como serie.
- **Consecuencias**: serie simple, estable y comparable entre monedas. No muestra el peso financiero de cada deuda; eso queda cubierto por el "Saldo total" de la Gráfica (SPEC-104) y el tab Análisis.

**ADR-003**: "Pagada" = finalizada o sin cuotas pendientes con al menos un pago
- **Contexto**: SPEC-104 finaliza automáticamente solo las deudas `activa`. Una deuda `inactiva` con todas sus cuotas pagadas puede quedar con estado desactualizado y no cerrarse nunca.
- **Decisión**: para la historia, una deuda está cerrada si `status = 'finalizada'`, si fue eliminada (`deleted_at`) o si no tiene cuotas pendientes y tiene al menos una cuota pagada (`pending_count = 0 AND paid_count > 0`). En los tres casos se usa la fecha de cierre de REQ-004. Una deuda sin cuotas pagadas ni pendientes no se considera cerrada.
- **Consecuencias**: la serie refleja la realidad de pago aunque el estado esté mal. `inactiva` con cuotas pendientes sigue contando como abierta (sigue siendo deuda). Riesgo: deudas canceladas sin pagos que el usuario deje `inactiva` seguirán contando; mitigado con la nota y el flujo esperado de `finalizada`/eliminación.

**ADR-004**: Serie por moneda, no suma cruzada
- **Contexto**: las deudas tienen `currency_id` y sumar monedas distintas produce números sin significado.
- **Decisión**: el endpoint devuelve una serie y totales por moneda; la UI muestra un selector (si hay más de una) y usa como default la moneda con mayor `pending_amount` actual.
- **Consecuencias**: consistente con `DebtChart` (SPEC-101 REQ-010). El usuario debe cambiar de moneda para ver el resto.

**ADR-005**: Endpoint devuelve todas las monedas en una respuesta
- **Contexto**: el gráfico permite cambiar de moneda; hacer un request por cambio agrega latencia en el iHost.
- **Decisión**: una respuesta con `currencies[]`, cada una con `series` y `summary`. El cambio de moneda es client-side.
- **Consecuencias**: payload mayor (12 meses × monedas, unos pocos KB) pero una sola llamada y sin recálculo de backend.

**ADR-006**: Reutilizar la semántica y los agregados de SPEC-104
- **Contexto**: SPEC-104 ya está en `main` y define la finalización automática, el cierre derivado de `MAX(paid_at)` y los campos `last_paid_at`/`pending_amount` en `GET /api/debts`.
- **Decisión**: la historia usa esa misma semántica: `last_paid_at` como fuente primaria de cierre, comparación por prefijo `YYYY-MM` por los formatos heterogéneos de `paid_at`, y `debtJoins`/agregados existentes como patrón de consulta. No se duplica lógica de cierre en el frontend ni se agregan columnas.
- **Consecuencias**: una deuda completada se cierra sola y deja de sumar sin intervención manual; la historia es consistente con "Pagadas" y con la Gráfica actual. La serie cuenta deudas (ADR-002), por lo que es inmune a cambios de monto por mora; `pending_amount` se usa solo para el saldo de referencia y para el default de moneda.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DeudasPage] --(GET /api/debts/history?months=12)--> [DebtHandlers.History]
                                                            |
                                                            v
                                                   [DebtService.History]
                                                            |
                                              +-------------+-------------+
                                              v                           v
                                   [DebtStorage.ListForHistory]  [países/currencies]
                                              |                  (JOIN currencies)
                                              v
                                        [SQLite: debts + MAX(paid_at) de debt_bills]
                                              |
                                              v
                             [Cálculo 12 cierres mensuales en Go]
                                              |
                                              v
                     [JSON: currencies[] → series[] + summary]
                                              |
                                              v
                       [DebtHistoryChart.tsx: SVG + tarjetas + tabla]
```

### 4.2 Componentes

#### 4.2.1 `DebtHistoryChart` (nuevo, frontend)
- **Responsabilidad**: recibir la respuesta del endpoint, dibujar barras SVG de 12 meses, tarjetas de totales, tabla de apoyo y tooltip; manejar selector de moneda y estados vacíos.
- **Interfaz**: `<DebtHistoryChart data={data} currencies={currencies} loading={loading} />` o fetch interno vía `api.debts.history(12)`. Se prefiere fetch interno con `useEffect` para mantener `DebtChart` sin lógica de datos y permitir estados de carga/error propios.
- **Dependencias**: `Icon`, `useI18nStore`, `useCurrencyFormatStore`, `Select` (solo si hay >1 moneda).
- **Ubicación**: `frontend/src/components/DebtHistoryChart.tsx`.

#### 4.2.2 `DebtChart` (modificado)
- **Responsabilidad**: agregar la tercera opción al segmented control (`'historial'`) y renderizar `DebtHistoryChart` cuando está activa, manteniendo las dos vistas actuales.
- **Ubicación**: `frontend/src/components/DebtChart.tsx`.

#### 4.2.3 `DebtService.History` (nuevo, backend)
- **Responsabilidad**: obtener deudas para historia, agrupar por moneda, calcular la serie mensual y los totales. Recibe `months` y `now` (inyectable para tests).
- **Ubicación**: `internal/services/debt.go` (método) y `internal/services/debt_history.go` (lógica pura + tests) si se prefiere aislar.
- **Dependencias**: `DebtStorage.ListForHistory`.

#### 4.2.4 `DebtStorage.ListForHistory` (nuevo, backend)
- **Responsabilidad**: una consulta con **todas** las deudas (incluidas las eliminadas lógicamente), moneda, último `paid_at` y `deleted_at`, reutilizando el patrón de agregados de SPEC-104 (`debtJoins`):
```sql
SELECT d.id, d.currency_id, c.code, c.symbol, d.total, d.start_date,
       d.installments_total, d.payment_day, d.status, d.updated_at, d.deleted_at,
       COALESCE(s.last_paid_at, '') AS last_paid_at,
       COALESCE(s.pending_amount, 0) AS pending_amount
FROM debts d
JOIN currencies c ON c.id = d.currency_id
LEFT JOIN (
    SELECT debt_id,
           MAX(CASE WHEN status = 'paid' THEN paid_at END) AS last_paid_at,
           SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END) AS pending_amount
    FROM debt_bills
    WHERE deleted_at IS NULL
    GROUP BY debt_id
) s ON s.debt_id = d.id
-- Sin filtro deleted_at: la historia debe conservar las deudas eliminadas
```
- **Ubicación**: `internal/storage/debt.go`.

#### 4.2.5 Handler y ruta
- **Responsabilidad**: parsear `months` (default 12, clamp 1..24), llamar al service, responder JSON. Errores en formato estándar.
- **Ubicación**: `internal/api/debt_handlers.go` + `internal/api/routes.go`.

#### 4.2.6 Cliente API y tipos (frontend)
- **Responsabilidad**: `api.debts.history(months)` en `frontend/src/api/index.ts` (o el archivo equivalente) y tipos `DebtHistoryResponse`, `DebtHistoryCurrency`, `DebtHistoryPoint`, `DebtHistorySummary` en `frontend/src/types/index.ts`.

### 4.3 Modelo de datos

Sin cambios de esquema. Se reutilizan:

```
Entidad: Debt (existente)
- id, currency_id, total, start_date, installments_total, payment_day
- status: activa | inactiva | finalizada
- updated_at, deleted_at

Entidad: DebtBill (existente, solo lectura agregada)
- debt_id, paid_at, deleted_at

Entidad: Currency (existente)
- id, code, symbol
```

Regla de cálculo por deuda y mes `m` (cierre = último instante de `m`):

```
abierta(m) = mes(start_date) <= m AND NOT (mes(closing_date) <= m)

closing_date (comparada por prefijo YYYY-MM):
  - deleted_at, si la deuda fue eliminada (prioridad absoluta)
  - last_paid_at > updated_at > última cuota, si status = 'finalizada'
    o si (pending_count = 0 AND paid_count > 0)
  - sin cierre, en cualquier otro caso

open_count(m)   = cantidad de deudas abiertas(m) de la moneda
closed_count(m) = cantidad de deudas cerradas/eliminadas con closing_date dentro de m
pending_amount_current(moneda) = Σ pending_amount de las deudas abiertas en el mes actual
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/debts/history?months=12`

**Request**: sin body. Query opcional `months` (int, 1..24, default 12).

**Response 200**:
```json
{
  "months": 12,
  "generated_at": "2026-10-21T12:00:00Z",
  "currencies": [
    {
      "code": "NIO",
      "symbol": "C$",
      "series": [
        { "month": "2025-11", "open_count": 5, "closed_count": 0 },
        { "month": "2025-12", "open_count": 5, "closed_count": 0 },
        { "month": "2026-01", "open_count": 4, "closed_count": 1 }
      ],
      "summary": {
        "start_count": 5,
        "end_count": 2,
        "change_count": -3,
        "change_pct": -60.0,
        "closed_count": 3,
        "pending_amount_current": 98000
      }
    }
  ]
}
```

- `series` siempre tiene exactamente `months` elementos, del más antiguo al actual, con `month` en formato `YYYY-MM`.
- `change_pct` es `null` si `start_count` es 0.
- `currencies` incluye toda moneda con **actividad en la ventana**: al menos un mes con `open_count > 0` o `closed_count > 0` (esto incluye deudas eliminadas dentro de la ventana). Si una moneda solo tiene deudas eliminadas antes del inicio de la ventana, no se incluye. Si no hay monedas con actividad, `currencies: []`.
- `closed_count` cuenta deudas cerradas (finalizadas, sin pendientes con pagos, o eliminadas) cuyo cierre cayó en el mes; `end_count` es la cantidad de deudas abiertas al cierre del mes actual.
- `pending_amount_current` se redondea a 2 decimales y es solo de referencia (puede variar por mora/intereses); no define las barras.

**Response Error**:
```json
{ "error": "invalid_months", "message": "months debe estar entre 1 y 24" }
```
- `401` sin sesión (middleware existente), `500` ante fallo de DB.

### 4.5 Dependencias

- **Internas**: `DebtStorage`, `DebtService`, `DebtHandlers`, rutas, `frontend/src/api`, tipos, `DebtChart`, i18n es/en.
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el tab Gráfica, cuando el usuario abre el segmented control, entonces ve tres opciones ("Cronograma", "Saldo total", "Últimos 12 meses") y la tercera muestra el nuevo gráfico.
- [ ] CA-002: El gráfico muestra 12 barras, una por mes, del más antiguo al actual, con etiquetas de mes/año legibles en es y en.
- [ ] CA-003: Una deuda con estado `finalizada` y último `paid_at` en marzo deja de sumar desde la barra de marzo inclusive (no aparece en abril ni posteriores).
- [ ] CA-004: Una deuda `activa` iniciada hace 2 meses aparece solo en las 2–3 barras correspondientes y no en meses anteriores a `start_date`.
- [ ] CA-005: Una deuda `inactiva` iniciada dentro de la ventana sigue sumando mientras tenga cuotas pendientes. Si no le quedan pendientes y tiene al menos un pago, deja de sumar desde el mes de su último pago (ADR-003).
- [ ] CA-006: Las tarjetas muestran deudas abiertas al inicio, deudas abiertas actuales, variación en cantidad (y %), deudas cerradas/eliminadas y saldo pendiente actual de referencia; los conteos coinciden con la serie y con la DB, y el saldo coincide con la suma de `pending_amount` de las deudas abiertas.
- [ ] CA-007: El tooltip de cada barra muestra mes, cantidad de deudas abiertas y cantidad de deudas cerradas/eliminadas de ese mes, sin montos.
- [ ] CA-008: Con deudas en más de una moneda, el selector cambia toda la vista sin volver a pedir datos al backend; el default es la moneda con mayor `pending_amount` actual.
- [ ] CA-009: Sin actividad en la ventana, se muestra el estado vacío con ícono, título y subtítulo (patrón existente), sin barras en cero. Si existen deudas eliminadas dentro de la ventana, se muestra la serie y **no** el estado vacío.
- [ ] CA-010: La tabla de apoyo lista los mismos 12 valores que las barras.
- [ ] CA-011: `GET /api/debts/history` sin sesión devuelve 401; con sesión devuelve 200 y el contrato de §4.4.
- [ ] CA-012: `months=0`, `months=25` o no numérico devuelve 400 con el error documentado (o se clampea explícitamente; decidir en implementación y testear).
- [ ] CA-013: Dada una deuda activa con cuotas pendientes, cuando el usuario paga la última cuota y la marca `finalizada` hoy, entonces los meses anteriores **siguen mostrando esa deuda y su total** y solo el mes actual (y posteriores) deja de sumarla. La serie pasada no cambia tras el cierre.
- [ ] CA-014: Dada una deuda abierta durante varios meses, cuando el usuario la **elimina** hoy, entonces los meses anteriores **siguen mostrando esa deuda y su total** y solo el mes de eliminación (y posteriores) deja de sumarla. La consulta de historia no filtra `deleted_at`.
- [ ] CA-015: Una deuda eliminada hace más de 12 meses no aparece en ninguna barra de la ventana actual, pero tampoco rompe totales ni conteos; si su moneda no tiene otra actividad en la ventana, no se ofrece en el selector.
- [ ] CA-017: La vista Cronograma muestra columnas verticales por deuda, legibles en 390 px (móvil), 768 px (tablet) y desktop; no hay solapamiento de etiquetas ni texto menor a 11 px en ejes; los controles respetan un área táctil mínima de 44 px.
- [ ] CA-016: Con **todas** las deudas cerradas (sin `pending_count > 0`), el tab Gráfica sigue mostrando el segmented control y la vista "Últimos 12 meses" con su serie histórica; no se aplica el early-return de estado vacío de Cronograma/Saldo.
- [ ] CA-DARK: Los controles (segmented control, selector de moneda) usan tokens del tema y se verificaron en darkmode; tooltip y textos legibles.
- [ ] CA-BACK: La vista vive dentro del tab Gráfica; no se crean páginas de detalle ni links "← Título", por lo que no requiere cambios en `BACK_ROUTES`.

### 5.2 No funcionales

- [ ] CA-NF-001: El endpoint responde en < 50 ms con 500 deudas y 24 meses en la DB local.
- [ ] CA-NF-002: No se agregan dependencias npm ni Go; `go vet`/`gofmt` y `npm run build` pasan.
- [ ] CA-NF-003: La vista es usable a 360 px de ancho (scroll horizontal o etiquetas alternadas, sin solapamientos) y en desktop.
- [ ] CA-NF-004: No hay cambios de esquema ni migraciones.

### 5.3 Testing

- **Unit tests (Go)**: lógica pura de cálculo con `now` fijo; casos: deuda finalizada con `paid_at` dentro y fuera de la ventana, deuda sin `paid_at` (fallback `updated_at`), deuda iniciada a mitad de ventana, deuda finalizada antes de la ventana, `inactiva` con y sin cuotas pendientes, deuda con `pending_count = 0` y `paid_count > 0` pero estado desactualizado, múltiples monedas, `start_count = 0`, `months` variable, y los 3 formatos de `paid_at`. **Casos clave de historia**: (a) deuda abierta en los 12 meses que se finaliza con `paid_at = hoy`; los 11 meses anteriores la cuentan y solo el mes actual no (CA-013); (b) deuda abierta durante la ventana que se elimina hoy (`deleted_at = hoy`); los meses anteriores la cuentan y el mes de eliminación no (CA-014); (c) deuda eliminada antes de la ventana, excluida de todas las barras (CA-015); (d) deuda inactiva con todas las cuotas pagadas: se cierra por REQ-003(c) en el mes de `last_paid_at`.
- **Integration tests (Go)**: handler con DB de test (`/tmp/test-app.db`), contrato JSON, 401 sin sesión, meses fuera de rango.
- **Frontend**: `npm run build` sin errores de TypeScript; verificación manual de las 3 vistas, cambio de moneda, tooltip, darkmode y responsive (360/768/1280).
- **Datos reales**: probar contra la DB local con al menos una deuda finalizada, una activa y una eliminada; contrastar los totales con consultas SQL manuales.
- **Performance**: medir el endpoint con `curl -w` sobre la DB local.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Tipos + `ListForHistory` + cálculo `History` en service con tests unitarios | 0.5 día | Ninguna |
| 2 | Handler + ruta + tests de contrato | 0.25 día | Fase 1 |
| 3 | Cliente API + tipos frontend + `DebtHistoryChart` (barras, tarjetas, tooltip, tabla) | 1 día | Fase 2 |
| 4 | Integración en `DebtChart` (tercera vista) + i18n es/en | 0.25 día | Fase 3 |
| 5 | Pruebas manuales con datos reales, darkmode, responsive, ajustes | 0.5 día | Fase 4 |

### 6.2 Milestones

1. **MVP**: endpoint + vista con barras, tarjetas de totales y estado vacío (REQ-001 a REQ-011).
2. **V1.0**: tabla de apoyo, promedio, resaltado del mes actual, nota explicativa y tests completos (REQ-012 a REQ-015).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Fecha de cierre inexacta porque el usuario finaliza manualmente sin pagos registrados | Baja | Medio | ADR-001 + nota visible; `last_paid_at` es la primera fuente y SPEC-104 finaliza automáticamente al pagar la última cuota |
| Deudas eliminadas por error quedan en la historia hasta el mes de borrado | Media | Bajo | Es el comportamiento buscado (no se pierde historia). Para corregir un alta equivocada, eliminarla el mismo mes: solo afecta la barra de ese mes |
| Formatos heterogéneos de `paid_at` rompen la fecha de cierre | Media | Alto | Comparar por prefijo `YYYY-MM` (`substr(paid_at,1,7)`), nunca parsear en Go (ADR-006); tests con los 3 formatos de SPEC-104 |
| Deudas canceladas que quedan en `inactiva` inflan la serie | Media | Medio | Nota explicativa y flujo esperado `finalizada`; evaluar en pruebas con datos reales |
| Barra del mes actual incompleta (mes en curso) se lee como "bajó" | Alta | Bajo | Etiquetar el mes actual como "en curso" en tooltip/nota; comparar cierre de mes |
| Mezcla de monedas | Alta | Alto | Serie y totales por moneda (ADR-004); nunca sumar códigos distintos |
| Serie poco informativa si no hay deudas cerradas en el período (línea plana) | Media | Bajo | Mostrar igualmente totales y variación; el promedio y la tabla aportan contexto |
| El saldo pendiente de referencia cambia por mora/intereses y no coincide con la serie | Media | Bajo | Etiquetarlo explícitamente como "referencia" y no usarlo para las barras ni para la variación |
| Etiquetas de meses superpuestas en móvil | Media | Bajo | `viewBox` responsive + etiquetas alternadas / scroll horizontal (patrón SPEC-102) |
| Consulta con subquery por deuda se vuelve lenta con muchas deudas | Baja | Bajo | Índice existente `idx_debt_bills_debt_id`; medir con 500 deudas |

## 8. Notas y Referencias

- `frontend/src/components/DebtChart.tsx` — patrón de segmented control, SVG responsive, tooltip y selector de moneda (SPEC-101/102).
- `frontend/src/pages/DeudasPage.tsx` — carga de deudas y tabs (`?tab=grafica`).
- `migrations/0024_create_debts_and_debt_bills.up.sql`, `migrations/0030_add_bills_debt_bills_dates.up.sql` — modelo.
- `internal/services/debt.go` — estados y generación de cuotas (`dueDateForInstallment`).
- SPEC-054 (modelo de deudas), SPEC-081 (fechas), SPEC-100/101/102 (gráficas de deudas).
- Buenas prácticas de diseño: [impeccable.style](https://impeccable.style) / [pbakaus/impeccable](https://github.com/pbakaus/impeccable) — tipografía legible, áreas táctiles ≥44 px, responsive real y evitar anti-patrones de UI generada.
- **SPEC-104** (released, merge `ec2e28c`): finalización automática, `last_paid_at`, `pending_amount` y advertencia de formatos de `paid_at`. Es la base de ADR-006; la vista histórica es complementaria y no reemplaza sus tabs.
- `docs/specs/templates/spec-template.md` — estructura.
- Decisión del usuario (2026-10-21): tercera vista dentro del tab Gráfica, con Cronograma como vista por defecto; métrica = **cantidad** de deudas abiertas por mes; "pagada" = finalizada o sin cuotas pendientes con al menos un pago; el mes de pago no cuenta; totales en cantidades + saldo pendiente actual de referencia; selector por moneda; ventana de 12 meses incluyendo el actual.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación |
| 2026-10-21 | opencode | Aclarar invariante histórico: pagar/finalizar hoy no borra los meses anteriores (REQ-005, CA-013) y que una deuda no finalizada cuenta por su total |
| 2026-10-21 | opencode | Incluir deudas eliminadas lógicamente en la historia, usando `deleted_at` como fecha de cierre (REQ-003/REQ-004, CA-014/CA-015) |
| 2026-10-21 | opencode | Incorporar SPEC-104 (ya en `main`): finalización automática, `last_paid_at`, comparación `YYYY-MM` de `paid_at`, early-return de `DebtChart` y ADR-006 |
| 2026-10-21 | opencode | Cambiar la métrica a cantidad de deudas (montos solo como referencia), definir "pagada" como finalizada/sin pendientes con pagos, mes de pago excluido y totales en cantidades (ADR-002/003, REQ-002..009) |
| 2026-10-21 | opencode | Confirmar Cronograma como vista por defecto del tab Gráfica (REQ-001) |
| 2026-10-21 | opencode | Cambio solicitado: Cronograma a barras verticales (una columna por deuda, altura = tiempo restante), con requisito de responsive en móvil y tablet (REQ-018, CA-017) |
| 2026-10-21 | opencode | Responsive móvil/tablet: min-w-0 en el layout para scroll interno, tabs de Deudas con wrap, tipografía de ejes ≥11 px y guía de diseño impeccable |
| 2026-10-21 | opencode | Inicio de implementación |
