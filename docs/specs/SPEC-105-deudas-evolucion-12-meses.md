---
title: "Evolución de deudas en los últimos 12 meses: gráfico comparativo y totales"
id: "SPEC-105"
status: "draft"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 108
---

# Evolución de deudas en los últimos 12 meses: gráfico comparativo y totales

**ID**: SPEC-105  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

La sección **Deudas** permite hoy ver la deuda del mes (Análisis), el calendario de cuotas, el listado de deudas y una pestaña **Gráfica** con dos vistas proyectadas al futuro ("Cronograma" y "Saldo total"). No existe, sin embargo, una vista **histórica** que responda a la pregunta más básica de seguimiento financiero: *¿mi deuda total está bajando o subiendo mes a mes?*

Esta spec agrega una **tercera vista "Últimos 12 meses"** dentro del tab Gráfica (`?tab=grafica`), con un gráfico de barras simple (un valor por mes) y un bloque de **números totales** al pie: deuda al inicio del período, deuda actual, variación absoluta y porcentual, y cantidad de deudas cerradas en el período. La métrica elegida por el usuario es **el total de deudas abiertas al cierre de cada mes**, no la suma de cuotas: una deuda cuenta desde su fecha de inicio y **deja de contar desde el mes en que se cierra/finaliza**, para que el gráfico no arrastre deudas ya pagadas.

El cálculo se hace en el backend con una consulta agregada liviana (una fila por deuda más el último pago registrado) y se expone en un endpoint autenticado nuevo; el frontend solo dibuja SVG y formatea. No hay cambios de esquema, no se agregan dependencias npm y no se hacen llamadas N+1.

**Consideraciones iHost**: el agregado es O(12 × n) sobre las deudas no eliminadas (decenas como máximo), con dos consultas SQL y memoria despreciable. El gráfico es SVG nativo renderizado en el navegador (mismo patrón que `DebtChart`/`DebtAnalysis`), sin librerías de charts.

**Consideraciones de UI (SPEC-060/063)**: la vista no agrega formularios; el selector de moneda y el segmented control usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y deben verificarse en darkmode. No se crean páginas de detalle ni links "← Título".

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar una **tercera vista "Últimos 12 meses"** al segmented control del tab Gráfica de Deudas, junto a "Cronograma" y "Saldo total", sin modificar el contenido ni el comportamiento de esas dos vistas ni de los demás tabs (Análisis, Calendario, Deudas).
2. **REQ-002**: La vista muestra un **gráfico de barras verticales** con **12 puntos mensuales**, del mes más antiguo al mes actual (incluido). Cada barra representa el **total de deudas abiertas al cierre de ese mes** en la moneda seleccionada.
3. **REQ-003**: Una deuda **cuenta como abierta** en un mes si su `start_date` es anterior o igual al cierre de ese mes y **no estaba cerrada/finalizada** en esa fecha. Las deudas con estado `finalizada` dejan de contar **desde el mes en que se cierran** (inclusive), de modo que el gráfico refleje la baja y no arrastre deudas ya pagadas.
4. **REQ-004**: La **fecha de cierre** de una deuda `finalizada` se determina, en este orden: (a) `MAX(paid_at)` de sus cuotas no eliminadas; (b) `updated_at` de la deuda; (c) fecha de vencimiento de la última cuota derivada de `start_date + installments_total`. Se documenta en tooltip/nota que es una reconstrucción (ver ADR-001).
5. **REQ-005**: Mostrar **números totales** debajo o encima del gráfico, en tarjetas:
   - **Deuda al inicio del período**: total al cierre del mes más antiguo de la ventana.
   - **Deuda actual**: total al cierre del mes en curso.
   - **Variación**: diferencia absoluta (`actual − inicio`) y porcentual, con signo y color semántico (verde si bajó, rojo si subió, neutro si no cambió).
   - **Deudas cerradas**: cantidad de deudas finalizadas dentro de la ventana de 12 meses.
6. **REQ-006**: **Tooltip por barra** con: nombre del mes y año, total de deudas abiertas, cantidad de deudas abiertas al cierre y cantidad de deudas cerradas en ese mes. En móvil, el tooltip se posiciona clampeado al viewport (mismo patrón que `DebtChart`).
7. **REQ-007**: **Selector de moneda** (solo si hay más de una moneda entre las deudas). El default es la moneda con mayor total de deudas abiertas en el mes actual. Todo el gráfico, los totales y los conteos corresponden a la moneda seleccionada.
8. **REQ-008**: **Estado vacío** con el patrón existente (card centrada, ícono, título y subtítulo) cuando no hay deudas o no hay deudas en la moneda seleccionada.
9. **REQ-009**: Endpoint autenticado **`GET /api/debts/history?months=12`** que devuelve la serie mensual y los totales para todas las monedas en una sola respuesta (ver §4.4), evitando N+1 y refetch al cambiar de moneda.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: **Tabla de datos mensuales** colapsable/de apoyo (debajo del gráfico) con columnas: mes, total abierto, deudas abiertas, deudas cerradas. Sirve como respaldo numérico accesible y para lectores de pantalla.
2. **REQ-011**: Eje Y con 4–5 líneas de grilla y etiquetas abreviadas (`k` para miles, `M` para millones) usando el formato de moneda del store; eje X con etiquetas de mes corto en el idioma activo (`months.N`), mostrando todos los meses en desktop y alternando en móvil si el ancho es insuficiente.
3. **REQ-012**: Resaltar visualmente la barra del **mes actual** (color primario pleno) frente a los meses anteriores (primario con menor opacidad), y marcar con línea punteada el nivel del **promedio del período**.
4. **REQ-013**: Nota explicativa bajo el gráfico: qué se cuenta, qué significa "cerrada" y que las cifras son una reconstrucción a partir del estado actual y de las fechas de pago.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Permitir consultar períodos de **6 o 24 meses** además de 12 mediante un selector (el endpoint acepta `months` entre 1 y 24).
2. **REQ-015**: Indicador de **mejor mes** (mayor reducción respecto del mes anterior) en la nota o tooltip.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: respuesta del endpoint < 50 ms con hasta 500 deudas y 24 meses; cálculo O(24 × n); sin N+1. Máximo 2 consultas SQL por request.
- **Seguridad**: endpoint bajo `authMiddleware`, igual que el resto de `/api/debts`; sin datos sensibles nuevos.
- **Almacenamiento**: **sin migraciones ni cambios de esquema**; se leen `debts`, `debt_bills` y `currencies` existentes.
- **Disponibilidad**: cambio aditivo; no altera endpoints ni respuestas existentes.
- **iHost**: cero dependencias nuevas (backend stdlib, frontend SVG); payload acotado (≤ 24 meses × monedas); sin proceso en background.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Estado actual del tab Gráfica** (`frontend/src/components/DebtChart.tsx`, SPEC-101/102): segmented control con `view: 'cronograma' | 'saldo'`; tres tarjetas de resumen; SVG con `viewBox` de ancho de diseño 720 y `min-w` para scroll en pantallas chicas; tooltip flotante clampeado; selector de moneda derivado de `debts` y default por mayor saldo. La tercera vista se integra al mismo segmented control y reutiliza esos patrones.
- **Página y datos** (`frontend/src/pages/DeudasPage.tsx`): carga `api.debts.list()` una sola vez y pasa `debts` + `currencies` a `DebtChart`. No hay datos históricos en el cliente: `Debt` no incluye cuotas ni `paid_at`.
- **Modelo** (`migrations/0024_create_debts_and_debt_bills.up.sql`, `0030_add_bills_debt_bills_dates.up.sql`):
  - `debts`: `total`, `start_date`, `installments_total`, `payment_day`, `status` (`activa`/`inactiva`/`finalizada`), `deleted_at`, `created_at`, `updated_at`.
  - `debt_bills`: `debt_id`, `installment_number`, `due_date`, `amount`, `status` (`pending`/`paid`), `paid_at`, `deleted_at`.
  - No existe tabla de historial de estados ni snapshots mensuales. `updated_at` se actualiza en cada edición y al marcar la deuda como finalizada.
- **Semántica de estados** (`internal/services/debt.go`): `finalizada` no genera cuotas; `activa` sí; `inactiva` no genera pero conserva la deuda. No hay transición automática a `finalizada` al pagar la última cuota: depende de que el usuario cambie el estado.
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

**ADR-001**: Reconstruir la fecha de cierre a partir de pagos y `updated_at`
- **Contexto**: no hay historial de cambios de estado. Para saber en qué mes dejó de contar una deuda finalizada hace falta una fecha de cierre.
- **Decisión**: `closing_date = MAX(paid_at)` de cuotas no eliminadas; si es NULL, `updated_at`; si tampoco fuera válido, el vencimiento de la última cuota (`start_date + installments_total`). La deuda deja de contar si `closing_date <= fin_del_mes`.
- **Consecuencias**: historia útil sin migraciones. Si el usuario finaliza una deuda sin registrar el pago de la última cuota, la fecha de cierre será la de la edición (`updated_at`), que puede ser posterior al pago real. Se documenta en la nota de la vista.

**ADR-002**: La métrica es el total de deudas abiertas al cierre de cada mes
- **Contexto**: el usuario eligió explícitamente esta métrica frente al saldo pendiente por cuotas, priorizando simplicidad y foco en estados.
- **Decisión**: cada deuda aporta su `total` completo mientras esté abierta; el gráfico baja de golpe cuando se cierra una deuda. No se descuentan pagos parciales.
- **Consecuencias**: serie simple y estable, fácil de explicar. No muestra amortización intra-deuda; eso ya lo cubre la vista "Saldo total" (proyección) y el tab Análisis (cuotas del mes).

**ADR-003**: `inactiva` cuenta como abierta
- **Contexto**: `inactiva` no genera cuotas nuevas pero la deuda sigue existiendo; `finalizada` es el único estado que implica deuda saldada.
- **Decisión**: solo `finalizada` (con fecha de cierre) deja de contar. `activa` e `inactiva` suman mientras hayan comenzado.
- **Consecuencias**: una deuda pausada sigue reflejándose, que es lo correcto desde el punto de vista del pasivo. Riesgo: deudas canceladas que el usuario deje en `inactiva` inflarán la serie; mitigado por la nota y porque el flujo esperado es marcar `finalizada`.

**ADR-004**: Serie por moneda, no suma cruzada
- **Contexto**: las deudas tienen `currency_id` y sumar monedas distintas produce números sin significado.
- **Decisión**: el endpoint devuelve una serie y totales por moneda; la UI muestra un selector (si hay más de una) y usa como default la moneda con mayor total abierto actual.
- **Consecuencias**: consistente con `DebtChart` (SPEC-101 REQ-010). El usuario debe cambiar de moneda para ver el resto.

**ADR-005**: Endpoint devuelve todas las monedas en una respuesta
- **Contexto**: el gráfico permite cambiar de moneda; hacer un request por cambio agrega latencia en el iHost.
- **Decisión**: una respuesta con `currencies[]`, cada una con `series` y `summary`. El cambio de moneda es client-side.
- **Consecuencias**: payload mayor (12 meses × monedas, unos pocos KB) pero una sola llamada y sin recálculo de backend.

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
- **Responsabilidad**: una consulta con deudas no eliminadas, moneda y último `paid_at`:
```sql
SELECT d.id, d.currency_id, c.code, c.symbol, d.total, d.start_date,
       d.installments_total, d.payment_day, d.status, d.updated_at,
       (SELECT MAX(b.paid_at)
          FROM debt_bills b
         WHERE b.debt_id = d.id AND b.deleted_at IS NULL) AS last_paid_at
FROM debts d
JOIN currencies c ON c.id = d.currency_id
WHERE d.deleted_at IS NULL
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
- updated_at

Entidad: DebtBill (existente, solo lectura agregada)
- debt_id, paid_at, deleted_at

Entidad: Currency (existente)
- id, code, symbol
```

Regla de cálculo por deuda y mes `m` (cierre = último instante de `m`):

```
abierta(m) = start_date <= cierre(m) AND NOT (status = 'finalizada' AND closing_date <= cierre(m))

closing_date = MAX(paid_at) > updated_at > última cuota (start_date + installments_total, día payment_day clampeado)
total_moneda(m) = Σ total de deudas abiertas(m) de esa moneda
open_count(m)   = cantidad de deudas abiertas(m)
closed_count(m) = cantidad de deudas con closing_date dentro de m
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
        { "month": "2025-11", "total": 185000, "open_count": 5, "closed_count": 0 },
        { "month": "2025-12", "total": 185000, "open_count": 5, "closed_count": 0 },
        { "month": "2026-01", "total": 150000, "open_count": 4, "closed_count": 1 }
      ],
      "summary": {
        "start_total": 185000,
        "end_total": 98000,
        "change": -87000,
        "change_pct": -47.03,
        "closed_count": 3,
        "end_open_count": 2
      }
    }
  ]
}
```

- `series` siempre tiene exactamente `months` elementos, del más antiguo al actual, con `month` en formato `YYYY-MM`.
- `change_pct` es `null` si `start_total` es 0.
- `currencies` incluye todas las monedas con al menos una deuda no eliminada; si no hay deudas, `currencies: []`.
- `total` se redondea a 2 decimales.

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
- [ ] CA-005: Una deuda `inactiva` iniciada dentro de la ventana sigue sumando (ADR-003).
- [ ] CA-006: Las tarjetas muestran deuda al inicio, deuda actual, variación (monto y %) y cantidad de deudas cerradas; los valores coinciden con la serie y con la DB.
- [ ] CA-007: El tooltip de cada barra muestra mes, total, deudas abiertas y deudas cerradas de ese mes.
- [ ] CA-008: Con deudas en más de una moneda, el selector cambia toda la vista sin volver a pedir datos al backend; el default es la moneda de mayor deuda actual.
- [ ] CA-009: Sin deudas, se muestra el estado vacío con ícono, título y subtítulo (patrón existente), sin barras en cero.
- [ ] CA-010: La tabla de apoyo lista los mismos 12 valores que las barras.
- [ ] CA-011: `GET /api/debts/history` sin sesión devuelve 401; con sesión devuelve 200 y el contrato de §4.4.
- [ ] CA-012: `months=0`, `months=25` o no numérico devuelve 400 con el error documentado (o se clampea explícitamente; decidir en implementación y testear).
- [ ] CA-DARK: Los controles (segmented control, selector de moneda) usan tokens del tema y se verificaron en darkmode; tooltip y textos legibles.
- [ ] CA-BACK: La vista vive dentro del tab Gráfica; no se crean páginas de detalle ni links "← Título", por lo que no requiere cambios en `BACK_ROUTES`.

### 5.2 No funcionales

- [ ] CA-NF-001: El endpoint responde en < 50 ms con 500 deudas y 24 meses en la DB local.
- [ ] CA-NF-002: No se agregan dependencias npm ni Go; `go vet`/`gofmt` y `npm run build` pasan.
- [ ] CA-NF-003: La vista es usable a 360 px de ancho (scroll horizontal o etiquetas alternadas, sin solapamientos) y en desktop.
- [ ] CA-NF-004: No hay cambios de esquema ni migraciones.

### 5.3 Testing

- **Unit tests (Go)**: lógica pura de cálculo con `now` fijo; casos: deuda finalizada con `paid_at` dentro y fuera de la ventana, deuda sin `paid_at` (fallback `updated_at`), deuda iniciada a mitad de ventana, deuda finalizada antes de la ventana, `inactiva`, múltiples monedas, `start_total = 0`, `months` variable.
- **Integration tests (Go)**: handler con DB de test (`/tmp/test-app.db`), contrato JSON, 401 sin sesión, meses fuera de rango.
- **Frontend**: `npm run build` sin errores de TypeScript; verificación manual de las 3 vistas, cambio de moneda, tooltip, darkmode y responsive (360/768/1280).
- **Datos reales**: probar contra la DB local con al menos una deuda finalizada y una activa; contrastar los totales con consultas SQL manuales.
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

1. **MVP**: endpoint + vista con barras, tarjetas de totales y estado vacío (REQ-001 a REQ-009).
2. **V1.0**: tabla de apoyo, promedio, resaltado del mes actual, nota explicativa y tests completos (REQ-010 a REQ-013).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Fecha de cierre inexacta porque el usuario finaliza sin pagar la última cuota | Media | Medio | ADR-001 + nota visible en la vista; usar `MAX(paid_at)` como primera fuente |
| Deudas soft-deleted desaparecen también del pasado | Media | Bajo | Documentar como limitación; el borrado lógico no conserva snapshot histórico |
| Deudas canceladas que quedan en `inactiva` inflan la serie | Media | Medio | Nota explicativa y flujo esperado `finalizada`; evaluar en pruebas con datos reales |
| Barra del mes actual incompleta (mes en curso) se lee como "bajó" | Alta | Bajo | Etiquetar el mes actual como "en curso" en tooltip/nota; comparar cierre de mes |
| Mezcla de monedas | Alta | Alto | Serie y totales por moneda (ADR-004); nunca sumar códigos distintos |
| Serie poco informativa si no hay deudas cerradas en el período (línea plana) | Media | Bajo | Mostrar igualmente totales y variación; el promedio y la tabla aportan contexto |
| Etiquetas de meses superpuestas en móvil | Media | Bajo | `viewBox` responsive + etiquetas alternadas / scroll horizontal (patrón SPEC-102) |
| Consulta con subquery por deuda se vuelve lenta con muchas deudas | Baja | Bajo | Índice existente `idx_debt_bills_debt_id`; medir con 500 deudas |

## 8. Notas y Referencias

- `frontend/src/components/DebtChart.tsx` — patrón de segmented control, SVG responsive, tooltip y selector de moneda (SPEC-101/102).
- `frontend/src/pages/DeudasPage.tsx` — carga de deudas y tabs (`?tab=grafica`).
- `migrations/0024_create_debts_and_debt_bills.up.sql`, `migrations/0030_add_bills_debt_bills_dates.up.sql` — modelo.
- `internal/services/debt.go` — estados y generación de cuotas (`dueDateForInstallment`).
- SPEC-054 (modelo de deudas), SPEC-081 (fechas), SPEC-100/101/102 (gráficas de deudas).
- SPEC-104 (por pagar vs historial) — spec en desarrollo; **no solapa** esta vista histórica de totales.
- `docs/specs/templates/spec-template.md` — estructura.
- Decisión del usuario (2026-10-21): tercera vista dentro del tab Gráfica; métrica = total de deudas abiertas por mes.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación |
