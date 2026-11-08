---
title: "Detalle de deuda: referencia de pago visible y semáforo de puntualidad"
id: "SPEC-128"
status: "in_progress"
author: "opencode"
created: "2026-11-08"
updated: "2026-11-08"
github_issue: 131
---

# Detalle de deuda: referencia de pago visible y semáforo de puntualidad

**ID**: SPEC-128
**Estado**: in_progress
**Autor**: opencode
**Creado**: 2026-11-08
**Actualizado**: 2026-11-08

---

## 1. Resumen Ejecutivo

El detalle de una deuda (`/deudas/:id`) ya muestra, para cada cuota pagada, la fecha de pago, pero **no muestra la referencia del pago** aunque el dato existe: se captura en el modal de pago, se persiste en `debt_bills.payment_reference` y la API `GET /api/debts/{id}/bills` ya lo devuelve como `payment_reference`. Tampoco indica si el pago se hizo en fecha, antes o después del vencimiento: el usuario debe comparar manualmente dos fechas para saber si cumplió.

Esta spec agrega dos etiquetas visibles en cada cuota pagada, tanto en `?status=paid` como en `?status=all`:

1. **Referencia del pago**: si existe, se muestra su valor; si no existe (o está vacía), se muestra el badge rojo **"Sin referencia"**.
2. **Semáforo de puntualidad**: compara la fecha de pago (`paid_at`) con el vencimiento (`due_date`) usando días de calendario:
   - **Verde**: pagada en fecha (`En fecha`) o antes (`N días antes`).
   - **Amarillo**: pagada entre 1 y 7 días después del vencimiento (`N días tarde`).
   - **Rojo**: pagada 8 días o más después del vencimiento (`N días tarde`).

No requiere cambios de esquema, migraciones ni de API: los tres campos (`payment_reference`, `paid_at`, `due_date`) ya existen y ya viajan al frontend. El cálculo y la presentación son exclusivamente frontend, con tests unitarios de la lógica de fechas y validación manual en local.

Consideraciones iHost: el cambio no toca SQLite, no agrega dependencias ni llamadas de red y no incrementa el consumo del backend. El impacto es un pequeño bundle de React y un puñado de claves i18n.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En la tarjeta de cada cuota con `status === 'paid'` de `DebtBillsPage` (`/deudas/:id`), mostrar la referencia de pago:
   - Si `payment_reference` tiene texto (después de `trim()`), mostrar un label/chip con la etiqueta **"Referencia"** y el valor (por ejemplo `Referencia: 902000001`).
   - Si `payment_reference` está ausente, es `null`, cadena vacía o solo espacios, mostrar un badge rojo con el texto **"Sin referencia"**.
   - El badge de referencia se muestra en `?status=paid` y en `?status=all`.
2. **REQ-002**: En la tarjeta de cada cuota con `status === 'paid'`, mostrar un label de **puntualidad** calculado como diferencia de días de calendario entre la fecha de pago y el vencimiento:
   - `diff <= 0` → **verde**:
     - `diff === 0`: `En fecha`.
     - `diff < 0`: `N días antes` (singular `1 día antes`).
   - `1 <= diff <= 7` → **amarillo**: `N días tarde` (singular `1 día tarde`).
   - `diff >= 8` → **rojo**: `N días tarde` (singular `1 día tarde`).
   - `diff = fecha de pago - fecha de vencimiento` (positivo = pago tardío, negativo = pago anticipado).
3. **REQ-003**: Los dos labels se renderizan únicamente para cuotas pagadas. Las cuotas pendientes conservan exactamente su comportamiento actual (`DueDateBadge` de vencimiento), sin referencia ni semáforo de pago.
4. **REQ-004**: La comparación debe ser por **día de calendario en la zona horaria del sistema** (la misma que usa `getSystemTimeZone()` en `frontend/src/utils/date.ts`), no por diferencia de horas ni por `paid_at.slice(0, 10)`:
   - `paid_at` llega como timestamp RFC3339 (con hora y offset).
   - `due_date` llega como `YYYY-MM-DD`.
   - Un pago a las 23:30 local del día de vencimiento debe contar como `En fecha`, aunque en UTC ya sea el día siguiente.
   - La fecha de pago que se muestra en la tarjeta también debe calcularse en la zona horaria del sistema (hoy se muestra con `slice(0, 10)`, lo que puede desfasarse un día).
5. **REQ-005**: Colores y estilos consistentes con los badges existentes:
   - Verde: `bg-success/20 text-green-800 dark:text-green-400`.
   - Amarillo: `bg-warning/20 text-yellow-800 dark:text-yellow-400`.
   - Rojo: `bg-danger/20 text-red-800 dark:text-red-400`.
   - Base: `text-xs font-semibold px-2.5 py-1 rounded-full whitespace-nowrap`.
   - El color nunca es la única señal: el texto siempre dice `En fecha`, `N días antes`, `N días tarde` o `Sin referencia`.
6. **REQ-006**: Agregar las traducciones en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` (fuente de verdad; el build sincroniza `frontend/src/i18n/`):
   - `deudas.payment_reference_short`: `Referencia` / `Reference`.
   - `deudas.payment_reference_missing`: `Sin referencia` / `No reference`.
   - `deudas.payment_timing_on_time`: `En fecha` / `On time`.
   - `deudas.payment_timing_days_early_one`: `{n} día antes` / `{n} day early`.
   - `deudas.payment_timing_days_early_other`: `{n} días antes` / `{n} days early`.
   - `deudas.payment_timing_days_late_one`: `{n} día tarde` / `{n} day late`.
   - `deudas.payment_timing_days_late_other`: `{n} días tarde` / `{n} days late`.
   - Usar el prefijo `deudas.payment_timing_*` para no colisionar con `deudas.timing_*` de SPEC-127 (REQ-028).
7. **REQ-007**: La tarjeta debe seguir siendo usable en móvil (375 px) y desktop:
   - Los labels deben envolver (`flex-wrap`) sin desbordar horizontalmente ni empujar el monto fuera de la tarjeta.
   - Referencias largas se truncan visualmente con `truncate` y `max-w-*`, conservando el valor completo en `title`/`aria-label`.
   - Verificar legibilidad en darkmode.
8. **REQ-008**: No modificar el esquema de `debts`/`debt_bills`, no agregar migraciones, no cambiar endpoints ni contratos JSON. El frontend consume los campos existentes.
9. **REQ-009**: Extraer la lógica de fechas a un helper puro y testeable, con tests unitarios en `frontend/tests/` ejecutables con `node --test` (patrón existente del repo). Cubrir como mínimo: mismo día, 1 día antes, varios días antes, 1 día tarde, 7 días tarde (amarillo), 8 días tarde (rojo), 30 días tarde (rojo), cambio de mes/año, timestamp cerca de medianoche con zona horaria y valores inválidos/ausentes.
10. **REQ-010**: Coordinar con SPEC-127 (REQ-028), que ya introduce una etiqueta de diferencia pago/vencimiento con verde/ámbar en la tarjeta pagada, `DebtPayModal`, `DebtAnalysis` y `RecognizeDocumentModal`:
    - Si `frontend/src/utils/paymentTiming.ts` y `PaymentTimingLabel.tsx` ya están en `main` al implementar, **extenderlos** (agregar el umbral rojo 8+ y el tono) en lugar de crear helpers/componentes duplicados.
    - Si no están en `main`, implementar el helper/componente de esta spec de forma compatible (misma semántica de días de calendario) para que el merge de SPEC-127 no duplique lógica.
    - No cambiar los textos ni el comportamiento de `DebtPayModal`, `DebtAnalysis` o `RecognizeDocumentModal` en esta spec; el alcance P0 es la tarjeta del detalle de deuda.
11. **REQ-011**: Privacidad: la referencia de pago es un dato financiero. No incluir referencias reales en tests, fixtures, specs, issues, capturas ni mensajes de commit; usar valores sintéticos (`902000001`, `DEMO-REF-001`). No loguear la referencia en consola.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-012**: Reutilizar el mismo label de puntualidad en el listado de cuotas adelantadas de `DebtAnalysis` y en cualquier historial de pagos que muestre cuotas pagadas, para que el criterio sea consistente en toda la app.
2. **REQ-013**: Permitir copiar la referencia al portapapeles desde la tarjeta (botón o click en el chip) con feedback tipo toast.
3. **REQ-014**: Si la referencia es muy larga, mostrar un tooltip/popover con el valor completo en desktop y mantener el `title` nativo en móvil.
4. **REQ-015**: Evaluar si el badge existente **"Adelantada"** (`deudas.paid_ahead_short`) se solapa visualmente con `N días antes`; si genera ruido, consolidar ambos en un solo label sin perder información.
5. **REQ-016**: Accesibilidad: `aria-label` en los badges que incluya el significado completo (por ejemplo "Pago realizado 3 días después del vencimiento") y foco visible si el chip se vuelve interactivo.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-017**: Filtro o resaltado "Pagadas tarde" en el detalle de deuda y en el análisis.
2. **REQ-018**: Estadísticas del análisis de deudas: porcentaje de cuotas pagadas en fecha, promedio de días de atraso y peor atraso.
3. **REQ-019**: Alerta/notificación cuando una cuota se paga fuera de fecha, integrada al sistema de alertas existente.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el cálculo es O(1) por cuota; no agrega requests ni trabajo de backend. Con 700+ cuotas el render sigue siendo despreciable.
- **Memoria**: sin nuevas dependencias; helper y componentes livianos. Impacto esperado < 5 KB en el bundle.
- **Almacenamiento**: ninguno; no se persisten datos nuevos.
- **Seguridad**: no se agregan endpoints ni logs con datos financieros.
- **Disponibilidad**: funciona offline; sin llamadas de red externas.
- **iHost**: sin cambios en Go, SQLite ni Docker; build multi-arch (`amd64`, `arm/v7`, `arm64`) intacto.
- **UI**: tokens del tema, darkmode y responsive según `docs/project-rules.md` §4.3.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizaron en paralelo el frontend y el backend del módulo de deudas (análisis multi-agente sobre el checkout `main`):

- **Frontend**:
  - `frontend/src/pages/DebtBillsPage.tsx` es la página de `/deudas/:id`; `?status=paid|all|pending` es un filtro de UI (no server-side) y se aplica en `filteredBills`.
  - La tarjeta de cuota muestra `installment_number`, `due_date` crudo, `paid_at.slice(0, 10)` y el estado; **nunca renderiza `payment_reference`**.
  - `DueDateBadge.tsx` ya implementa un semáforo de vencimiento con las clases `bg-success/20`, `bg-warning/20` y `bg-danger/20`; es el patrón visual a seguir.
  - `isPaidAhead` (`frontend/src/utils/debtMonthlyAggregation.ts`) ya detecta pagos adelantados por mes, pero no calcula días exactos.
  - `frontend/src/utils/date.ts` expone `getSystemTimeZone()` y `daysUntilDue()`; `frontend/src/utils/debtPayoff.ts` tiene `toDateOnly()`/`parseDateOnly()` para evitar el off-by-one de UTC.
  - El repo usa `node --test` para tests de utilidades (por ejemplo `frontend/tests/*.test.ts`).
- **Backend**:
  - `debt_bills` ya tiene `payment_reference TEXT`, `paid_at DATETIME` y `due_date TEXT` desde la migración `0024`; no hace falta migrar.
  - `internal/models/debt_bill.go` expone `payment_reference`, `paid_at` y `due_date` en JSON (con `omitempty` en los opcionales).
  - `GET /api/debts/{id}/bills` devuelve todas las cuotas (sin filtro de estado); el filtro es del frontend.
- **Antecedente directo**: SPEC-127 (en progreso en otro worktree, todavía no en `main`) incorpora en su REQ-028 una etiqueta de diferencia pago/vencimiento en la tarjeta pagada, el modal de pago y el de reconocimiento, con verde/ámbar y textos `Pagada N días antes/el día/N días después`. Esta spec **no la duplica**: define el semáforo verde/amarillo/rojo definitivo para la tarjeta y coordina la reutilización del helper.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Calcular los días y la referencia en el backend y devolver campos nuevos | Un solo lugar de verdad | Cambia el contrato JSON y agrega lógica de presentación al backend; no es necesario | ❌ Rechazada |
| Calcular en el frontend con `paid_at.slice(0, 10)` y comparar strings | Cero código nuevo | Ignora la hora/zona horaria; produce falsos "1 día tarde" cerca de medianoche | ❌ Rechazada |
| **Helper frontend con días de calendario en zona del sistema** | Correcto, testeable, sin API nueva, reutilizable | Requiere tests y cuidado con zonas horarias | ✅ Seleccionada |
| Mostrar solo la referencia y no el semáforo | Menos cambios | No resuelve la pregunta principal del usuario (cumplimiento de fecha) | ❌ Rechazada |
| **Referencia + semáforo en la tarjeta pagada** | Responde ambas preguntas en el mismo lugar | Dos labels nuevos por tarjeta; hay que cuidar el layout móvil | ✅ Seleccionada |
| Badge "Sin referencia" en gris/neutro | Menos alarmante | El usuario pidió explícitamente rojo para evidenciar la falta | ❌ Rechazada |
| Rojo para cualquier atraso | Simple | Pierde el matiz de tolerancia 1-7 días pedido por el usuario | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Cambio exclusivamente frontend, sin tocar DB ni API**
- **Contexto**: `payment_reference`, `paid_at` y `due_date` ya existen y ya se serializan.
- **Decisión**: Calcular y renderizar en `DebtBillsPage` con un helper puro y componentes de presentación.
- **Consecuencias**: Cero migraciones y cero riesgo para datos; el backend y el iHost no cambian.

**ADR-002: Días de calendario en la zona horaria del sistema**
- **Contexto**: `paid_at` es un timestamp; `due_date` es fecha sin hora. Comparar strings o restar milisegundos produce errores de un día.
- **Decisión**: Convertir `paid_at` a `YYYY-MM-DD` en la zona del sistema (`getSystemTimeZone()`), parsear ambas fechas como días UTC y restar. El helper acepta una zona inyectable para tests deterministas.
- **Consecuencias**: El resultado coincide con lo que el usuario ve en su calendario; los tests pueden fijar `America/Managua` sin depender del entorno.

**ADR-003: Umbrales verde/amarillo/rojo definidos por el usuario**
- **Contexto**: El usuario pidió un semáforo y eligió la tolerancia.
- **Decisión**: Verde `diff <= 0`, amarillo `1..7`, rojo `>= 8` días de atraso.
- **Consecuencias**: Los umbrales quedan centralizados en el helper y son fáciles de ajustar en una sola constante si cambia la política.

**ADR-004: La referencia faltante es un badge rojo, no texto vacío**
- **Contexto**: Una cuota pagada sin referencia es un dato faltante que el usuario quiere detectar.
- **Decisión**: Mostrar `Sin referencia` con las clases de peligro cuando el campo esté vacío/ausente; nunca ocultar la fila.
- **Consecuencias**: Se visualiza la deuda de datos; no se inventa ni se autocompleta la referencia.

**ADR-005: Coordinación explícita con SPEC-127**
- **Contexto**: SPEC-127 (en progreso en otro worktree) ya introduce `paymentTiming.ts`, `PaymentTimingLabel.tsx` y claves `deudas.timing_*` para el mismo cálculo en otros puntos de la app.
- **Decisión**: Esta spec usa `deudas.payment_timing_*` y define el helper/componente de forma compatible; al implementar, si SPEC-127 ya está en `main`, se extiende lo existente en vez de duplicar. El alcance P0 se limita a la tarjeta del detalle.
- **Consecuencias**: Menor riesgo de conflicto de merge y una sola fuente de verdad para la diferencia de días.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌──────────────────────────────────────────────┐
│ Frontend React                               │
│                                              │
│  DebtBillsPage (/deudas/:id?status=paid|all) │
│    ├─ PaymentReferenceBadge                  │
│    │    ├─ referencia presente → chip neutro │
│    │    └─ sin referencia → chip rojo        │
│    ├─ PaymentPunctualityBadge                │
│    │    ├─ verde: en fecha / N días antes    │
│    │    ├─ amarillo: 1-7 días tarde          │
│    │    └─ rojo: 8+ días tarde               │
│    └─ DueDateBadge (solo pendientes, igual)  │
└───────────────────┬──────────────────────────┘
                    │ GET /api/debts/{id}/bills (sin cambios)
                    ▼
┌──────────────────────────────────────────────┐
│ Go API + SQLite                              │
│  debt_bills.payment_reference / paid_at /    │
│  due_date (columnas existentes)              │
└──────────────────────────────────────────────┘
```

No hay componentes backend nuevos ni cambios de contrato.

### 4.2 Componentes

#### 4.2.1 Helper `frontend/src/utils/paymentTiming.ts`

- **Responsabilidad**: calcular la diferencia de días de calendario y el tono del semáforo.
- **Interfaz propuesta**:
  ```ts
  export type PaymentTimingTone = 'success' | 'warning' | 'danger'
  export type PaymentTimingStatus = 'on_time' | 'early' | 'late'

  export interface PaymentTiming {
    days: number          // firmado: negativo = antes, 0 = en fecha, positivo = tarde
    status: PaymentTimingStatus
    tone: PaymentTimingTone
  }

  export function getPaymentTiming(
    paidAt: string | null | undefined,
    dueDate: string | null | undefined,
    timeZone?: string,
  ): PaymentTiming | null

  export function paymentDateInTimeZone(
    paidAt: string | null | undefined,
    timeZone?: string,
  ): string | null // YYYY-MM-DD
  ```
- **Reglas**:
  - Si falta `paidAt`, falta `dueDate` o alguna fecha es inválida → `null` (el componente no renderiza nada; nunca muestra un falso "En fecha").
  - `days = paidDay - dueDay` en días de calendario.
  - `tone = success` si `days <= 0`; `warning` si `1 <= days <= 7`; `danger` si `days >= 8`.
  - `status = on_time` si `days === 0`; `early` si `days < 0`; `late` si `days > 0`.
- **Ubicación**: `frontend/src/utils/paymentTiming.ts`.
- **Coordinación**: si SPEC-127 ya está en `main`, extender su helper existente con el umbral rojo y el tono, manteniendo los nombres de campos que ya use el resto de la app.

#### 4.2.2 Componente `frontend/src/components/PaymentPunctualityBadge.tsx`

- **Responsabilidad**: renderizar el label de puntualidad con el texto i18n y las clases del tema.
- **Props**: `paidAt?: string | null`, `dueDate?: string | null`.
- **Comportamiento**:
  - Obtiene la zona horaria con `getSystemTimeZone()` (patrón de `DueDateBadge`) o recibe una zona opcional para tests.
  - Si `getPaymentTiming()` devuelve `null`, no renderiza nada.
  - Texto:
    - `on_time` → `t('deudas.payment_timing_on_time')`.
    - `early` → clave `_one` si `Math.abs(days) === 1`, si no `_other`; reemplaza `{n}`.
    - `late` → clave `_one` si `days === 1`, si no `_other`; reemplaza `{n}`.
  - Clases según `tone` (ver REQ-005).
  - `aria-label` descriptivo con el texto completo.
- **Ubicación**: `frontend/src/components/PaymentPunctualityBadge.tsx`.

#### 4.2.3 Componente `frontend/src/components/PaymentReferenceBadge.tsx`

- **Responsabilidad**: mostrar la referencia o su ausencia.
- **Props**: `reference?: string | null`.
- **Comportamiento**:
  - `reference.trim()` no vacío → chip neutro `bg-card`/`bg-border` con `text-text-secondary` y texto `Referencia: <valor>`; si supera ~40 caracteres, `truncate max-w-[16rem]` y `title` con el valor completo.
  - Vacío/ausente → chip rojo `bg-danger/20 text-red-800 dark:text-red-400` con `Sin referencia`.
- **Ubicación**: `frontend/src/components/PaymentReferenceBadge.tsx`.

#### 4.2.4 Integración en `frontend/src/pages/DebtBillsPage.tsx`

- En la columna izquierda de la tarjeta, debajo de la fecha de pago, agrupar la referencia y la puntualidad en un contenedor que envuelva en móvil:
  ```tsx
  {bill.status === 'paid' && bill.paid_at && (
    <p className="text-xs text-text-secondary mt-0.5">
      {t('deudas.pay_date')}: {paymentDateInTimeZone(bill.paid_at, timeZone) ?? '—'}
    </p>
  )}
  {bill.status === 'paid' && (
    <div className="mt-1 flex flex-wrap items-center gap-2 min-w-0">
      <PaymentReferenceBadge reference={bill.payment_reference} />
      <PaymentPunctualityBadge
        paidAt={bill.paid_at}
        dueDate={bill.due_date}
        timeZone={timeZone}
      />
    </div>
  )}
  ```
- La columna derecha (monto, documentos, estado y badge `Adelantada`) queda sin cambios para no sumar ancho en móvil.
- Mantener el badge existente `Adelantada` (`isPaidAhead`) y el `DueDateBadge` de pendientes sin cambios en P0.
- La tarjeta debe usar `flex-wrap` donde haga falta para evitar overflow en móvil.

#### 4.2.5 i18n

- Editar **solo** `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` (fuente de verdad).
- El build (`npm run build` en `frontend/`) sincroniza `frontend/src/i18n/` y regenera `public/`; no editar `public/i18n/` a mano.

### 4.3 Modelo de datos

Sin cambios. Campos existentes en `debt_bills`:

```
Entidad: debt_bills (existente)
- payment_reference: TEXT NULL   → referencia del pago
- paid_at: DATETIME NULL         → timestamp del pago
- due_date: TEXT NOT NULL        → vencimiento YYYY-MM-DD
- status: TEXT                   → 'pending' | 'paid'
```

### 4.4 APIs / Contratos

Sin cambios. `GET /api/debts/{id}/bills` ya devuelve:

```json
[
  {
    "id": 1,
    "installment_number": 7,
    "due_date": "2026-03-10",
    "amount": 123.45,
    "status": "paid",
    "paid_at": "2026-03-08T15:30:00Z",
    "payment_reference": "902000001"
  }
]
```

Los campos opcionales usan `omitempty`; el frontend debe tratar `undefined` como ausente.

### 4.5 Dependencias

- **Internas**: `DebtBillsPage.tsx`, `utils/date.ts` (`getSystemTimeZone`), i18n `es`/`en`, tipos `DebtBill` existentes.
- **Externas**: ninguna nueva. No se agregan librerías de fechas; se usa `Intl` y aritmética UTC de la stdlib.

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: En `/deudas/:id?status=paid`, una cuota pagada con `payment_reference = "902000001"` muestra el label `Referencia: 902000001`.
- [ ] **CA-002**: En `/deudas/:id?status=paid`, una cuota pagada sin `payment_reference` (ausente, `null`, `""` o espacios) muestra el badge rojo `Sin referencia`.
- [ ] **CA-003**: En `/deudas/:id?status=all`, las cuotas pagadas muestran ambos labels y las pendientes conservan solo su badge de vencimiento actual.
- [ ] **CA-004**: `paid_at` y `due_date` en el mismo día → badge verde `En fecha`.
- [ ] **CA-005**: Pago 1 día antes → verde `1 día antes`; 3 días antes → verde `3 días antes`.
- [ ] **CA-006**: Pago 1 día después → amarillo `1 día tarde`; 7 días después → amarillo `7 días tarde`.
- [ ] **CA-007**: Pago 8 días después → rojo `8 días tarde`; 30 días después → rojo `30 días tarde`.
- [ ] **CA-008**: Un pago a las 23:30 de la zona del sistema el día de vencimiento se considera `En fecha`, aunque en UTC sea el día siguiente.
- [ ] **CA-009**: Un `paid_at` inválido o ausente en una cuota pagada no muestra un falso `En fecha`; el label de puntualidad se omite y la referencia se sigue mostrando.
- [ ] **CA-010**: Los textos cambian correctamente entre `es` y `en`, con singular/plural (`1 día`/`N días`).
- [ ] **CA-011**: Los labels usan las clases de color de REQ-005 y son legibles en darkmode.
- [ ] **CA-012**: En 375 px de ancho no hay overflow horizontal; la referencia larga se trunca y el monto sigue visible.
- [ ] **CA-013**: Los tests unitarios del helper cubren todos los casos de CA-004 a CA-009 y pasan con `node --test frontend/tests/*.test.ts`.
- [ ] **CA-014**: No se modifica ningún archivo de `internal/`, `migrations/` ni el contrato de la API.
- [ ] **CA-015**: No se commitean referencias, montos ni fechas reales; los ejemplos son sintéticos.
- [ ] **CA-DARK**: Los labels nuevos se verificaron en darkmode en la tarjeta del detalle de deuda.
- [ ] **CA-BUILD**: `npm run build` en `frontend/` (incluye `check-ui-rules.cjs`) y `go test ./...` pasan.

### 5.2 No funcionales

- [ ] **CA-NF-001**: El cambio no agrega dependencias npm ni Go.
- [ ] **CA-NF-002**: No hay requests adicionales ni cambios en el consumo de SQLite/RAM del iHost.
- [ ] **CA-NF-003**: El bundle incremental es < 5 KB gzip.
- [ ] **CA-NF-004**: La app sigue funcionando sin conexión a red externa.

### 5.3 Testing

- **Unit tests** (`frontend/tests/paymentTiming.test.ts`):
  - Mismo día, 1/3 días antes, 1/7/8/30 días después, cambio de mes y año.
  - `paid_at` con hora y zona horaria cerca de medianoche.
  - `paid_at`/`due_date` ausentes o inválidos → `null`.
  - Verificación de `tone` y `status` en cada umbral.
- **Component tests/manual**:
  - Render de `PaymentReferenceBadge` con valor, vacío y `null`.
  - Render de `PaymentPunctualityBadge` en los tres colores.
- **Integración/manual**:
  - Ejecutar el server en local y validar `/deudas/:id?status=paid` y `?status=all` con una copia de la DB de producción (solo lectura, en `/tmp`), verificando cuotas con y sin referencia y con distintos atrasos.
  - Verificar darkmode, móvil (375 px) y desktop.
- **Build**:
  - `cd frontend && npm run build` (incluye `npm run check:ui`).
  - `go test ./...` y `go vet ./...` para confirmar que no hay regresiones.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Helper `paymentTiming.ts` + tests unitarios | 0.25 día | Ninguna |
| 2 | Componentes `PaymentPunctualityBadge` y `PaymentReferenceBadge` + i18n | 0.25 día | Fase 1 |
| 3 | Integración en `DebtBillsPage` (paid/all), responsive y darkmode | 0.25 día | Fase 2 |
| 4 | Validación con copia de producción, build y pruebas manuales | 0.25 día | Fase 3 |
| 5 | Actualización de spec/tracker y release | 0.25 día | Fase 4 |

**Total estimado**: ~1 día de desarrollo + pruebas.

### 6.2 Milestones

1. **MVP**: labels de referencia y puntualidad en las cuotas pagadas del detalle, con tests y darkmode verificado.
2. **V1.0**: reutilización en `DebtAnalysis`/historial, copiar referencia y consolidación con el badge `Adelantada`.
3. **P2**: filtros/estadísticas de atrasos y alertas.

### 6.3 Coordinación con SPEC-127

1. Antes de implementar, verificar si `main` ya tiene `frontend/src/utils/paymentTiming.ts` y `PaymentTimingLabel.tsx` (SPEC-127 REQ-028).
2. Si existen:
   - Extender `getPaymentTiming`/helper con el umbral rojo (8+) y el tono.
   - Reutilizar `PaymentTimingLabel` en la tarjeta o envolverlo en `PaymentPunctualityBadge` con una prop de variante; no duplicar cálculo.
   - Mantener las claves `deudas.timing_*` existentes para los otros puntos y usar `deudas.payment_timing_*` solo si no hay solapamiento.
3. Si no existen:
   - Crear `paymentTiming.ts` con la interfaz de §4.2.1 y `PaymentPunctualityBadge.tsx`.
   - Dejar el helper preparado para que SPEC-127 lo reutilice sin cambios de contrato.

### 6.4 Estado de implementación (2026-11-08)

Implementado en la rama `feature/SPEC-128`:

- **Helper** `frontend/src/utils/paymentTiming.ts`: `paymentDateInTimeZone()` y `getPaymentTiming()` con días de calendario, zona horaria inyectable, umbrales verde/amarillo/rojo y validación de fechas.
- **Componentes**:
  - `frontend/src/components/PaymentPunctualityBadge.tsx`: semáforo con textos i18n, singular/plural y clases del tema.
  - `frontend/src/components/PaymentReferenceBadge.tsx`: chip con la referencia o badge rojo `Sin referencia`.
- **Integración** en `frontend/src/pages/DebtBillsPage.tsx`: ambos labels en la columna izquierda de las cuotas pagadas, visibles en `?status=paid` y `?status=all`; la fecha de pago se calcula en la zona horaria del sistema. Las cuotas pendientes y el badge `Adelantada` quedan sin cambios.
- **i18n**: claves `deudas.payment_*` en `frontend/public/i18n/{es,en}.json`, sincronizadas al build.
- **Tests**: `frontend/tests/paymentTiming.test.ts` con 14 casos (mismo día, antes/después, límites 7/8, cambio de mes/año, medianoche, fechas inválidas y zona inválida).

Validaciones ejecutadas:

- `node --test frontend/tests/*.test.ts`: 45 tests OK (14 nuevos de SPEC-128 + los existentes de SPEC-127).
- `npm run build` (incluye `check-ui-rules.cjs`, `sync-i18n.cjs`, `tsc -b` y `vite build`): OK.
- `go test ./...`: OK.
- `go vet ./...`: OK.

**Merge con `main` (SPEC-127 ya liberada)**: se integró `origin/main` (commit `b3766ad`) en `feature/SPEC-128`:

- `frontend/src/utils/paymentTiming.ts` quedó como archivo único con ambas APIs: `paymentTiming()` de SPEC-127 (sin cambios, usada por `PaymentTimingLabel`, `DebtPayModal`, `DebtAnalysis` y `RecognizeDocumentModal`) y `getPaymentTiming()`/`paymentDateInTimeZone()` de SPEC-128 (zona horaria y semáforo verde/amarillo/rojo).
- En la tarjeta de `DebtBillsPage` se usa `PaymentPunctualityBadge` (SPEC-128) para el semáforo; el `PaymentTimingLabel` de SPEC-127 se conserva en los demás contextos. La acción **Reconocer comprobante** y su modal de SPEC-127 se mantienen intactos.
- Se conservaron ambas familias de claves i18n (`deudas.timing_*` y `deudas.payment_timing_*`).
- El tracker quedó con 126 specs, 125 released (SPEC-127) y 1 in_progress (SPEC-128).

Pendiente: prueba manual del usuario en local y release formal (merge a `main`, tracker, label e issue).

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Conflicto de merge con SPEC-127 (mismo helper/componente) | Alta | Medio | Coordinación explícita (§6.3); extender lo existente en lugar de duplicar. |
| Off-by-one por zona horaria/UTC | Media | Alto | Comparar días de calendario con `getSystemTimeZone()` y tests con timestamps cerca de medianoche. |
| `paid_at` ausente en cuotas pagadas legacy | Media | Medio | Omitir el badge de puntualidad; nunca mostrar un falso `En fecha`. La referencia se muestra igual. |
| Referencias largas que rompen el layout móvil | Media | Medio | `truncate`, `max-w-*`, `flex-wrap` y prueba en 375 px. |
| Dos badges ("Adelantada" y "N días antes") generan ruido | Media | Bajo | Mantener ambos en P0 y evaluar consolidación en P1 (REQ-015). |
| Claves i18n en el archivo equivocado | Baja | Medio | Editar solo `frontend/public/i18n/` y verificar con `curl /i18n/es.json` tras el build. |
| PII en tests/specs | Baja | Alto | Usar solo `902000001`/`DEMO-REF-001`; revisar `git diff --cached` antes de commitear. |
| Contraste insuficiente en darkmode | Baja | Medio | Usar las mismas clases que `DueDateBadge` y verificar visualmente en darkmode. |

---

## 8. Notas y Referencias

- **Página**: `frontend/src/pages/DebtBillsPage.tsx` (`/deudas/:id`, filtros `?status=paid|all|pending`).
- **Modelo backend**: `internal/models/debt_bill.go`; columnas en `migrations/0024_create_debts_and_debt_bills.up.sql`.
- **Endpoint**: `GET /api/debts/{id}/bills` (`internal/api/debt_handlers.go`).
- **Patrón visual**: `frontend/src/components/DueDateBadge.tsx` (verde/amarillo/rojo).
- **Utilidades de fecha**: `frontend/src/utils/date.ts`, `frontend/src/utils/debtPayoff.ts`.
- **SPEC-127**: reconocimiento de comprobantes bancarios; REQ-028 introduce la diferencia pago/vencimiento en otros puntos. Coordinar según §6.3.
- **SPEC-122**: comprobantes PDF de deudas y campo `payment_reference`.
- **SPEC-043**: referencia y comprobante en pagos de facturas de servicios (patrón previo).
- **Reglas UI**: `docs/project-rules.md` §4.3 (inputs/darkmode), §4.4 (navegación de detalle).
- **Datos sintéticos**: `902000001`, `DEMO-REF-001`, `2026-03-08`, `2026-03-10`, `123.45`. Nunca usar datos reales.

### Fuera de alcance (P0)

- Cambios en el modal de pago o en el reconocimiento de comprobantes.
- Cálculo o persistencia de la referencia en el backend.
- Filtros/estadísticas de atrasos.
- Migraciones o cambios de esquema.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-08 | opencode | Creación inicial de la especificación a partir del pedido del usuario: referencia de pago visible y semáforo de puntualidad en `/deudas/:id?status=paid` y `?status=all`. |
| 2026-11-08 | opencode | Estado `in_progress`: implementación frontend completa (helper, componentes, integración, i18n y tests), build y tests de Go validados. Pendiente prueba manual y release. |
| 2026-11-08 | opencode | Merge de `origin/main` (SPEC-127 released): se unificó `paymentTiming.ts` conservando ambas APIs, se mantuvieron las claves i18n de los dos specs y la tarjeta quedó con el semáforo de SPEC-128. 45 tests y build OK. |
