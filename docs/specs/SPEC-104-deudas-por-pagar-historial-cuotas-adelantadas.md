---
title: "Deudas: separar por pagar del historial, excluir cuotas adelantadas de los totales y finalización automática"
id: "SPEC-104"
status: "in_progress"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 107
---

# Deudas: separar por pagar del historial, excluir cuotas adelantadas de los totales y finalización automática

**ID**: SPEC-104  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

Hoy el módulo de Deudas mezcla tres conceptos que para el usuario son distintos: **lo que falta pagar**, **lo que ya se pagó** y **lo que se pagó por adelantado**. En la base de producción hay 3 deudas (Lavadora Samsung, Lentes y Lentes oscuros) con **todas sus cuotas pagadas** que sin embargo siguen con `status = 'activa'`, porque nada en el sistema las cierra. Esas deudas siguen apareciendo en el listado activo, siguen sumando en la vista Gráfica (`debt.total` original) y obligan a revisar a mano si ya se terminaron. Además, cuando se adelantan cuotas (caso real: el 2026-10-09 se adelantaron las cuotas de noviembre, diciembre y enero por C$13,162.38), esas cuotas siguen apareciendo en los meses futuros sin una marca clara de que ya están pagadas, y el "total del mes" del bloque de Análisis las incluye, distorsionando la lectura del compromiso mensual.

La propuesta es separar explícitamente **"Por pagar"** de **"Pagadas"** dentro del mismo tab de Deudas (sin páginas nuevas), hacer que **solo las cuotas pendientes de deudas activas** alimenten los totales mensuales, marcar las cuotas pagadas antes de su vencimiento como **"pagada por adelantado"** y **finalizar automáticamente** una deuda cuando se paga su última cuota. Con eso el usuario ve, por un lado, el dinero que realmente tiene que salir y, por el otro, un historial auditable de lo ya cerrado, sin que lo pagado contamine las sumas del mes.

El cambio **no requiere migraciones de esquema**, no agrega dependencias y su costo en el iHost es despreciable: un `LEFT JOIN` con agregados sobre `debt_bills` (hoy 690 filas) al listar deudas, y una condición extra al marcar una cuota como pagada. El frontend reutiliza los tabs y filtros existentes (`filter_pending` / `filter_paid` ya están definidos en i18n pero sin uso).

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Finalización automática al pagar la última cuota.** Al marcar una cuota como `paid`, si la deuda estaba `activa` y ya no le quedan cuotas `pending`, la deuda pasa a `status = 'finalizada'` en la misma operación de negocio (no en el handler). La fecha de cierre no se persiste como columna nueva: se deriva de `MAX(paid_at)` de sus cuotas.
2. **REQ-002 — Reconciliación de cierre para deudas ya completadas.** En el mismo reconciliador que hoy genera cuotas faltantes (`DebtService.ReconcileDebtBills`, invocado desde `GET /api/debts`), finalizar toda deuda `activa` cuyas cuotas estén todas pagadas. Esto corrige automáticamente al desplegar las 3 deudas de producción que hoy están completas pero activas, sin SQL manual.
3. **REQ-003 — "Por pagar" como vista principal.** El tab Deudas muestra por defecto **solo las deudas que tienen al menos una cuota pendiente y `status = 'activa'`**. Cada card muestra: saldo pendiente, cuotas pagadas/total, próxima cuota (fecha y monto) y porcentaje de avance. No se muestran deudas cerradas en esta vista.
4. **REQ-004 — Sección "Pagadas" (historial centralizado).** Un filtro conmutador en el mismo tab Deudas (`Por pagar` · `Pagadas` · `En pausa` · `Todas`) centraliza las deudas sin cuotas pendientes. Las cards de "Pagadas" son de solo lectura visual y muestran: institución, total pagado (suma de cuotas pagadas), cantidad de cuotas, **fecha de cierre** (`MAX(paid_at)`) y, si aplica, el **ahorro mensual liberado** (monto de cuota que dejó de salir). Se agrupan por año de cierre descendente. El detalle (`/deudas/:id`) sigue accesible para auditar cuotas.
5. **REQ-005 — Regla única de suma mensual.** Todo total de deuda mostrado como "a pagar" (bloque "Necesario del mes" de Análisis, totales por moneda, Calendario y Gráfica) se calcula **solo** con cuotas `pending` de deudas `status = 'activa'`. Quedan excluidas: (a) cuotas `paid`, (b) cuotas pendientes de deudas `inactiva`, (c) deudas `finalizada`. Una cuota pagada por adelantado **no suma** en el mes de su vencimiento.
6. **REQ-006 — Marca de "pagada por adelantado".** Una cuota se considera adelantada cuando `substr(paid_at,1,7) < substr(due_date,1,7)` (comparación por prefijo `YYYY-MM`, robusta a los formatos heterogéneos de `paid_at` que existen hoy). En el mes de vencimiento se muestra en un bloque colapsable **"Pagadas por adelantado"** con su fecha de pago, y se informa el monto ya cubierto del mes (ej.: "C$4,387.46 ya pagados por adelantado"). No desaparece: se puede auditar, pero no cuenta como necesidad de caja.
7. **REQ-007 — Resumen por deuda en el listado.** `GET /api/debts` devuelve, por deuda, campos calculados (no persistidos): `pending_count`, `pending_amount`, `paid_count`, `paid_amount`, `next_due_date`, `last_due_date`, `last_paid_at`. El listado se resuelve en **una sola consulta** con agregados (sin N+1).
8. **REQ-008 — Gráfica basada en saldo real.** El tab Gráfica deja de usar `debt.total` (monto original) para el "Saldo total" y el cronograma: usa `pending_amount` como saldo y `MAX(due_date)` de las cuotas pendientes como fecha de finalización proyectada. Las deudas cerradas no se grafican (o se muestran en una nota "N deudas cerradas, fuera del gráfico").
9. **REQ-009 — Coherencia al editar cuotas.** (a) Si se aumenta `installments_total` de una deuda `finalizada`, la deuda vuelve a `activa` y se generan las cuotas nuevas (hoy `ensureDebtBills` no hace nada para deudas no activas y la deuda quedaría cerrada con cuotas faltantes). (b) Si se reduce `installments_total`, las cuotas sobrantes (`installment_number > installments_total`) se eliminan lógicamente, para que no impidan el cierre automático ni aparezcan como deuda fantasma.
10. **REQ-010 — i18n y tema.** Todas las cadenas nuevas en `frontend/public/i18n/es.json` y `en.json` (fuente de verdad). Los controles nuevos usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`); sin colores hardcodeados.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011 — Vista "Por caja" vs "Por vencimiento" en Análisis.** Toggle que cambia la base del mes: **Por vencimiento** (compromiso: `due_date`, comportamiento actual) y **Por caja** (dinero que salió: `paid_at`). La vista "Por caja" responde "¿cuánto pagué realmente en octubre?" y hace visible el efecto de los adelantos.
2. **REQ-012 — Ahorro liberado por deudas cerradas.** En "Pagadas", mostrar por moneda cuánto dejó de salir por mes tras el cierre y el total pagado del año, para dimensionar el alivio real del presupuesto.
3. **REQ-013 — Búsqueda y filtro por acreedor/año en Pagadas.** Para que el historial siga siendo usable cuando crezca.
4. **REQ-014 — Calendario con adelantadas.** En la vista Calendario, las cuotas adelantadas se muestran con badge "Adelantada" y no se suman al total del día.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015 — Exportar historial de deudas cerradas a CSV** (descarga desde el navegador, sin proceso en el iHost).
2. **REQ-016 — Reabrir deuda finalizada** desde el detalle, con confirmación, para corregir un pago cargado por error (requiere soporte de "desmarcar pagada", hoy inexistente).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: `GET /api/debts` con ≤ 20 deudas y ≤ 1,000 cuotas en < 30 ms; agregados resueltos por SQLite, sin cargar cuotas en memoria del servidor.
- **Memoria iHost**: sin cachés nuevas ni estructuras persistentes; el cálculo de agrupaciones queda en el frontend sobre arreglos ya existentes (≤ 1,000 elementos).
- **Almacenamiento**: sin migraciones de esquema; sin tablas nuevas; sin archivos nuevos en disco.
- **Compatibilidad**: los campos nuevos de `GET /api/debts` son aditivos; el contrato actual no se rompe (bot de Telegram y demás consumidores siguen funcionando).
- **Dependencias**: ninguna librería nueva (backend stdlib + `database/sql`; frontend React/Tailwind existentes).
- **Consistencia de datos**: la finalización debe ser idempotente y segura ante ejecuciones concurrentes del reconciliador (`UPDATE ... WHERE status = 'activa' AND NOT EXISTS (...pending...)`).

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se auditó la base de producción del iHost (solo lectura) y el código del módulo:

- **Datos** (`/app/data/app.db`, 2026-10-21): 11 deudas no eliminadas, 690 cuotas. **Ninguna** deuda en `finalizada`; 3 deudas (ids 2, 3 y 4) tienen `pending_count = 0` y siguen `activa`. El 2026-10-09 se marcaron 11 cuotas pagadas por C$27,058.02, de las cuales C$13,162.38 correspondían a cuotas de nov/dic/ene.
- **Backend**: `DebtService.PayBill` (`internal/services/debt.go:117`) solo marca la cuota; no evalúa el estado de la deuda. `ReconcileDebtBills` (línea 133) genera cuotas faltantes de deudas activas, pero no cierra nada. `DebtStorage.List` no trae ningún agregado de cuotas.
- **API**: `GET /api/debts` devuelve `Debt` tal cual; `GET /api/debts/bills?year&month` devuelve todas las cuotas del mes (pagadas y pendientes) con `paid_at`.
- **Frontend**: `DeudasPage.tsx` filtra por `status` con chips `activa/all/inactiva/finalizada` (default `activa`), pero como nadie finaliza, "activa" incluye deudas cerradas. `DebtAnalysis.tsx` ya calcula `pending` correctamente, pero muestra además "Total del mes" que incluye pagadas. `DebtChart.tsx` usa `debt.total` (monto original) y `debtEndDate` (inicio + cuotas), por lo que suma deuda ya pagada. `DebtBillsPage.tsx` ya tiene filtros `pending/paid` y progreso por deuda.
- **i18n**: `deudas.filter_pending` y `deudas.filter_paid` existen y no se usan.
- **Formatos de `paid_at`**: conviven 3 formatos (252 `YYYY-MM-DD HH:MM:SS`, 26 `YYYY-MM-DD`, 21 `YYYY-MM-DD HH:MM:SS +0000 UTC`). Conclusión: comparar por prefijo de 7 caracteres en SQL y no parsear en Go para la detección de adelantos.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| A. Derivar estado y totales al vuelo (JOIN con agregados) | Sin migraciones, sin datos duplicados, siempre consistente | Una consulta algo más compleja | ✅ Seleccionada |
| B. Tabla `closed_debts` / archivado explícito | Historial "congelado" | Duplica datos, requiere migración y sincronización, más RAM/disco | ❌ Rechazada |
| C. Reportar `finalizada` solo en el read model, sin persistir | Cero escrituras | El estado seguiría mal en la DB, la API y el bot; cada consumidor tendría que replicar la regla | ❌ Rechazada |
| D. Mantener las cuotas adelantadas sumando en su mes y restarlas en la UI | Nada que cambiar en backend | Dos fuentes de verdad, fácil de desincronizar, el bot seguiría mostrando montos inflados | ❌ Rechazada |
| E. Página nueva `/deudas/pagadas` | Separación visual fuerte | Página + ruta + `BACK_ROUTES` + más bundle; el filtro en el mismo tab ya resuelve | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: "Por pagar" = cuotas pendientes de deudas activas**
- **Contexto**: el usuario necesita una sola cifra confiable de lo que le falta desembolsar. Hoy conviven cuotas pagadas, adelantadas, deudas pausadas y cerradas en el mismo conjunto.
- **Decisión**: todo agregado de caja futura se calcula con `debt_bills.status = 'pending'` **y** `debts.status = 'activa'`. Las deudas `inactiva` se muestran aparte ("En pausa") y no suman.
- **Consecuencias**: una deuda pausada deja de presionar el flujo mensual (esperado). Si el usuario quiere verla, la encuentra en su grupo, con su saldo visible pero fuera de totales.

**ADR-002: Persistir `finalizada` al cerrar la última cuota**
- **Contexto**: el estado `finalizada` existe en el modelo y en la UI, pero ningún flujo lo asigna. Eso obliga a revisar deudas a mano y rompe la semántica del filtro "activa".
- **Decisión**: al pagar la última cuota, y en cada reconciliación, pasar a `finalizada` (solo desde `activa`). La fecha de cierre se deriva de `MAX(paid_at)`; no se agrega columna.
- **Consecuencias**: los filtros existentes vuelven a significar lo que dicen. Riesgo asumido: si el usuario cargó un pago por error, debe corregirlo (REQ-016, P2) — se documenta como limitación.

**ADR-003: Las cuotas adelantadas se muestran, pero no suman**
- **Contexto**: el 2026-10-09 se adelantaron 3 meses de dos deudas. Si esas cuotas se ocultan, el usuario pierde trazabilidad; si suman, el mes parece más caro de lo que es.
- **Decisión**: se conservan visibles en un bloque "Pagadas por adelantado" del mes de vencimiento y se excluyen de toda suma de "a pagar". Detección por prefijo `YYYY-MM` de `paid_at` vs `due_date`.
- **Consecuencias**: la vista "Por vencimiento" muestra el compromiso del mes con su parte ya cubierta; la vista "Por caja" (P1) muestra el desembolso real del mes en que se pagó. Ambas responden preguntas distintas sin contradecirse.

**ADR-004: Historial dentro del tab Deudas, no página aparte**
- **Contexto**: "centralizar las pagadas" no justifica una ruta nueva en un iHost con recursos limitados, y el patrón del módulo ya usa filtros conmutadores.
- **Decisión**: chips `Por pagar` (default) · `Pagadas` · `En pausa` · `Todas` en el mismo tab, reutilizando `DeudasPage.tsx`.
- **Consecuencias**: menos código, menos bundle, sin cambios en `BACK_ROUTES`. El detalle de una deuda cerrada sigue disponible para auditar cuotas.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DeudasPage] --GET /api/debts--> [DebtHandlers] --> [DebtService.ReconcileDebtBills]
      |                                                  |
      |                                                  v
      |                                          [DebtStorage.List]
      |                                                  |
      |                                        SQL: debts + LEFT JOIN agregados
      |                                                  v
      |                                              [SQLite]
      |
      +--GET /api/debts/bills?year&month--> [ListByMonth] (paid_at incluido)
      |
      +--POST /api/debts/bills/:id/pay---> [PayBill] --> si no quedan pending => finalizar
```

### 4.2 Componentes

#### 4.2.1 `DebtService` (backend)
- **Responsabilidad**: regla de cierre automático y reconciliación.
- **Interfaz**: `PayBill`, `ReconcileDebtBills`, `Update`.
- **Ubicación**: `internal/services/debt.go`.
- Cambios:
  - `PayBill`: tras `debtBills.Pay(...)`, invocar `finalizeIfCompleted(ctx, bill.DebtID)`.
  - `ReconcileDebtBills`: tras `ensureDebtBills`, invocar `finalizeCompletedDebts(ctx)` (una sola sentencia SQL en storage).
  - `Update`: si `installments_total` aumentó sobre una deuda `finalizada`, forzar `status = 'activa'` antes de regenerar; si disminuyó, eliminar lógicamente cuotas sobrantes.

#### 4.2.2 `DebtStorage` / `DebtBillStorage` (backend)
- **Responsabilidad**: agregados y transiciones de estado.
- **Ubicación**: `internal/storage/debt.go`, `internal/storage/debt_bill.go`.
- Nuevos métodos:
  - `DebtBillStorage.CountPending(ctx, debtID) (int, error)`.
  - `DebtBillStorage.FinalizeCompleted(ctx) (int64, error)`:
    ```sql
    UPDATE debts
    SET status = 'finalizada', updated_at = CURRENT_TIMESTAMP
    WHERE status = 'activa' AND deleted_at IS NULL
      AND EXISTS (SELECT 1 FROM debt_bills b WHERE b.debt_id = debts.id AND b.deleted_at IS NULL)
      AND NOT EXISTS (SELECT 1 FROM debt_bills b WHERE b.debt_id = debts.id
                      AND b.deleted_at IS NULL AND b.status = 'pending');
    ```
  - `DebtBillStorage.SoftDeleteOverflow(ctx, debtID, maxInstallment int) error` (REQ-009b).

#### 4.2.3 `models.Debt` (backend)
- Campos calculados (no persistidos): `PendingCount int`, `PendingAmount float64`, `PaidCount int`, `PaidAmount float64`, `NextDueDate string`, `LastDueDate string`, `LastPaidAt string`.
- No se persisten: no hay migración.

#### 4.2.4 `DeudasPage` (frontend)
- Chips: `Por pagar` (default) · `Pagadas` · `En pausa` · `Todas` (reemplaza el filtro por `status`; se conserva el parámetro `status` en la URL para no romper enlaces existentes).
- Card "Por pagar": descripción, institución, saldo pendiente, `pagadas/total`, próxima cuota y avance.
- Card "Pagadas": total pagado, fecha de cierre, cuotas, "ahorro mensual" (P1) y estado visual atenuado (patrón actual de `status !== 'activa'`).
- Agrupación por año en "Pagadas" y contador ("12 deudas cerradas").

#### 4.2.5 `DebtAnalysis` / `DebtCalendar` / `DebtChart` (frontend)
- `DebtAnalysis`: el bloque principal pasa a **"Tenés que pagar"** (`pending` de deudas activas); "Total del mes" queda como referencia etiquetada y se agrega la nota de adelantos + bloque colapsable "Pagadas por adelantado". Toggle "Por vencimiento / Por caja" (P1).
- `DebtCalendar`: badge "Adelantada", totales del día solo con pendientes activas.
- `DebtChart`: saldo por `pending_amount`; fin proyectado por `MAX(due_date)` pendiente; deudas cerradas listadas fuera del gráfico.

### 4.3 Modelo de datos

Sin cambios de esquema. Se usan columnas existentes:

```
debts
- status: 'activa' | 'inactiva' | 'finalizada'
- (cierre derivado) MAX(debt_bills.paid_at) WHERE status='paid'

debt_bills
- due_date: fecha de vencimiento (compromiso)
- status: 'pending' | 'paid'
- paid_at: fecha de pago (caja); 3 formatos históricos → comparar por prefijo YYYY-MM
```

Agregado del listado (una consulta):

```sql
SELECT d.id, ..., 
       COALESCE(s.pending_count, 0), COALESCE(s.pending_amount, 0),
       COALESCE(s.paid_count, 0),    COALESCE(s.paid_amount, 0),
       COALESCE(s.next_due_date, ''), COALESCE(s.last_due_date, ''), COALESCE(s.last_paid_at, '')
FROM debts d
LEFT JOIN (
  SELECT debt_id,
         SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending_count,
         SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END) AS pending_amount,
         SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) AS paid_count,
         SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END) AS paid_amount,
         MIN(CASE WHEN status = 'pending' THEN due_date END) AS next_due_date,
         MAX(CASE WHEN status = 'pending' THEN due_date END) AS last_due_date,
         MAX(CASE WHEN status = 'paid' THEN paid_at END) AS last_paid_at
  FROM debt_bills
  WHERE deleted_at IS NULL
  GROUP BY debt_id
) s ON s.debt_id = d.id
WHERE d.deleted_at IS NULL
ORDER BY d.description;
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/debts` (extensión aditiva)

**Response 200** (fragmento por deuda):

```json
{
  "id": 2,
  "description": "Lavadora Samsung",
  "status": "finalizada",
  "installments_total": 12,
  "pending_count": 0,
  "pending_amount": 0,
  "paid_count": 12,
  "paid_amount": 62221.39,
  "next_due_date": "",
  "last_due_date": "",
  "last_paid_at": "2026-10-09 00:00:00 +0000 UTC"
}
```

#### Endpoint: `POST /api/debts/bills/{id}/pay` (mismo contrato)

**Response 200**: la cuota pagada (igual que hoy). Efecto colateral documentado: si era la última pendiente, la deuda pasa a `finalizada` (visible en el siguiente `GET /api/debts`).

#### Endpoint: `GET /api/debts/{id}/bills` y `GET /api/debts/bills?year=&month=` (extensión aditiva)

Cada cuota incluye `debt_status` (estado de la deuda dueña) para que el frontend aplique la regla REQ-005 sin consultas adicionales. Se mantiene `paid_at`, con el que el frontend detecta adelantos (`substr(paid_at,1,7) < substr(due_date,1,7)`).

### 4.5 Dependencias

- **Internas**: `internal/services/debt.go`, `internal/storage/debt.go`, `internal/storage/debt_bill.go`, `internal/models/debt.go`, `internal/api/debt_handlers.go` (sin cambios de lógica), `frontend/src/pages/DeudasPage.tsx`, `frontend/src/pages/DebtBillsPage.tsx`, `frontend/src/components/DebtAnalysis.tsx`, `DebtCalendar.tsx`, `DebtChart.tsx`, `frontend/src/types/index.ts`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: ninguna.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Dada una deuda activa con una única cuota pendiente, al marcarla pagada la deuda queda `finalizada` y desaparece de "Por pagar" (aparece en "Pagadas").
- [ ] **CA-002**: Al abrir Deudas por primera vez tras el deploy, las deudas de producción con todas sus cuotas pagadas (hoy ids 2, 3 y 4) figuran como `finalizada` sin intervención manual, y `SELECT COUNT(*) FROM debts WHERE status='finalizada'` devuelve 3.
- [ ] **CA-003**: "Por pagar" (default) lista solo deudas activas con al menos una cuota pendiente; el contador coincide con `SELECT COUNT(DISTINCT debt_id) FROM debt_bills WHERE status='pending' AND deleted_at IS NULL` ∩ deudas activas.
- [ ] **CA-004**: "Pagadas" lista todas las deudas sin cuotas pendientes, con total pagado y fecha de cierre; sumar sus `paid_amount` por moneda coincide con la suma de cuotas `paid` de esas deudas.
- [ ] **CA-005**: En Análisis, la cifra "Tenés que pagar" de noviembre 2026 es **USD 2,433.63** y **NIO 0.00** (las 2 cuotas NIO de noviembre ya están pagadas por adelantado); no incluye cuotas `paid`.
- [ ] **CA-006**: Las cuotas NIO de nov/dic/ene aparecen en su mes bajo "Pagadas por adelantado" con fecha 2026-10-09 y no incrementan el total a pagar (0 pendiente).
- [ ] **CA-007**: `GET /api/debts` devuelve `pending_count`, `pending_amount`, `paid_count`, `paid_amount`, `next_due_date`, `last_due_date` y `last_paid_at` coherentes para las 11 deudas; la consulta SQL es única (sin N+1).
- [ ] **CA-008**: La Gráfica "Saldo total" muestra USD 197,794.40 como saldo agregado (suma de `pending_amount`) y no grafica las 3 deudas cerradas; la fecha de fin de cada deuda corresponde a su última cuota pendiente.
- [ ] **CA-009**: Aumentar `installments_total` de una deuda finalizada la devuelve a `activa` y genera las cuotas nuevas; reducirlo elimina lógicamente las cuotas sobrantes y permite que la deuda se cierre.
- [ ] **CA-010**: Una deuda `inactiva` con cuotas pendientes se muestra en "En pausa" y no suma en ningún total mensual; los totales del bot de Telegram (`/deudas_pendientes`) coinciden con la regla de REQ-005.
- [ ] **CA-DARK**: Los chips y notas nuevos usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad de texto y placeholders en darkmode.
- [ ] **CA-BACK**: No se crean páginas ni rutas nuevas; `BACK_ROUTES` no requiere cambios y no se agregan links "← Título" dentro del contenido.

### 5.2 No funcionales

- [ ] **CA-NF-001**: `GET /api/debts` responde en < 30 ms con los datos de producción; `EXPLAIN QUERY PLAN` no muestra escaneos repetidos por deuda.
- [ ] **CA-NF-002**: Sin migraciones (`migrations/` sin archivos nuevos) y sin dependencias nuevas en `go.mod` / `package.json`.
- [ ] **CA-NF-003**: `go test ./...` y `npm run build` pasan sin errores ni warnings nuevos.
- [ ] **CA-NF-004**: `GET /health` sigue respondiendo `{"status":"ok"}` tras el deploy y `PRAGMA integrity_check` devuelve `ok`.

### 5.3 Testing

- **Unit tests** (Go): finalización al pagar la última cuota (idempotente, no toca `inactiva`/`finalizada`), reconciliación de deudas ya completas, `Update` con aumento/reducción de cuotas, agregados del listado (pending/paid/next/last), regla de suma (pending + activa).
- **Integration tests**: flujo pagar última cuota → listado; mes con cuotas adelantadas → totales; deuda inactiva → excluida; deuda finalizada editada → reactivada.
- **E2E/manual en local**: reproducir el estado de producción en una copia de la DB (`/tmp/test-app.db`) con las 3 deudas cerradas y verificar CA-001..CA-009 en navegador, en light y darkmode.
- **Performance**: medir `GET /api/debts` con 20 deudas y 1,000 cuotas sembradas; validar < 30 ms en la dev machine y sin crecimiento de RSS relevante en el iHost.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: agregados en `DebtStorage.List` + campos calculados en `models.Debt` | 0.5 día | Ninguna |
| 2 | Backend: finalización automática (`PayBill`, reconciliador) + tests | 0.5 día | Fase 1 |
| 3 | Backend: coherencia de edición de cuotas (REQ-009) + tests | 0.5 día | Fase 2 |
| 4 | Frontend: chips "Por pagar/Pagadas/En pausa/Todas" y cards con saldo/cierre | 1 día | Fase 1 |
| 5 | Frontend: Análisis (nota + bloque de adelantadas) y Calendario | 0.5 día | Fase 4 |
| 6 | Frontend: Gráfica por saldo pendiente (REQ-008) | 0.5 día | Fase 1 |
| 7 | i18n es/en, darkmode, pruebas locales con copia de producción | 0.5 día | Fases 2–6 |
| 8 | (P1) Toggle "Por caja" + ahorro liberado + búsqueda en Pagadas | 1 día | Fases 4–5 |

### 6.2 Milestones

1. **MVP**: REQ-001..REQ-010 (P0) — deudas cerradas fuera de los totales, historial centralizado, cuotas adelantadas excluidas de las sumas y gráfica con saldo real.
2. **V1.0**: REQ-011..REQ-014 (P1) — vista "Por caja" y métricas de alivio presupuestal.
3. **V1.1**: REQ-015..REQ-016 (P2) — exportación y reapertura.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Finalizar una deuda por un pago cargado por error | Baja | Medio | La deuda pasa a "Pagadas" visible (no se borra); REQ-016 (reabrir) queda como P2 y se documenta la limitación |
| Deudas `inactiva` dejan de sumar y el usuario espera lo contrario | Media | Medio | Mostrarlas en "En pausa" con su saldo visible y una nota explícita de que no suman |
| Datos históricos con cuotas faltantes impiden el cierre | Baja | Medio | `ReconcileDebtBills` genera faltantes antes de evaluar el cierre; test con deuda incompleta |
| Formatos heterogéneos de `paid_at` rompen la detección de adelantos | Media | Bajo | Comparación por prefijo `YYYY-MM` en SQL (no parseo estricto); tests con los 3 formatos presentes |
| Regresión en el bot de Telegram (usa cuotas pendientes) | Media | Medio | La regla REQ-005 es la misma que ya aplica `ListPendingWithDetails`; se agrega test de paridad de totales |
| Gráfica con cifras distintas a las esperadas por el usuario | Media | Bajo | Mostrar nota "N deudas cerradas fuera del gráfico" y tooltip con saldo pendiente |
| Consulta de listado más costosa en iHost | Baja | Bajo | Índice existente por `debt_id`; medir con `EXPLAIN QUERY PLAN` y 1,000 cuotas |
| i18n editado en el directorio equivocado | Baja | Medio | Editar solo `frontend/public/i18n/{es,en}.json`, correr `npm run build` y verificar por `curl` |

---

## 8. Notas y Referencias

- **SPEC-054**: modelo de deudas y cuotas (`debts`, `debt_bills`), calendario y email diario.
- **SPEC-080**: `/deudas_pendientes` del bot de Telegram (debe respetar la misma regla de suma).
- **SPEC-100/101/102**: tabs Gráfica y su responsividad (REQ-008 las modifica).
- **SPEC-063**: navegación de retorno (no aplica: no se crean páginas nuevas).
- **SPEC-060**: tokens de tema en formularios y darkmode.
- **Datos de producción usados como caso de prueba** (solo lectura, 2026-10-21): 11 deudas activas, C$0 de saldo en NIO, USD 197,794.40 pendiente; 3 deudas completas sin finalizar (ids 2, 3, 4); adelanto de C$13,162.38 del 2026-10-09.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación (draft) a partir de la auditoría de producción y del pedido del usuario |
| 2026-10-21 | opencode | Estado `in_progress`. Se agrega `last_due_date` al resumen por deuda: la Gráfica necesita la última cuota pendiente para proyectar el fin real |
| 2026-10-21 | opencode | MVP implementado (REQ-001 a REQ-010). Se agrega `debt_status` a las cuotas para que Análisis y Calendario puedan excluir de las sumas las deudas no activas. Validado contra una copia de la DB de producción: las 3 deudas completas se finalizan solas y noviembre queda en USD 2,433.63 / NIO 0.00. Pendientes: P1 (REQ-011 a REQ-014) y P2 (REQ-015, REQ-016) |
