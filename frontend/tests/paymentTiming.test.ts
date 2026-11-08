import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  getPaymentTiming,
  paymentDateInTimeZone,
  paymentTiming,
  PAYMENT_TIMING_WARNING_MAX_DAYS,
} from '../src/utils/paymentTiming.ts'

// SPEC-127: helper existente para la fecha de pago vs vencimiento.
test('paymentTiming detecta pago antes del vencimiento', () => {
  assert.deepEqual(paymentTiming('2026-02-01', '2026-02-05'), {
    days: -4,
    absDays: 4,
    kind: 'early',
  })
})

test('paymentTiming detecta pago el mismo día', () => {
  assert.deepEqual(paymentTiming('2026-02-05', '2026-02-05'), {
    days: 0,
    absDays: 0,
    kind: 'on_time',
  })
})

test('paymentTiming detecta un día después', () => {
  assert.deepEqual(paymentTiming('2026-02-06', '2026-02-05'), {
    days: 1,
    absDays: 1,
    kind: 'late',
  })
})

test('paymentTiming acepta paid_at con hora y zona horaria', () => {
  assert.deepEqual(paymentTiming('2026-02-01T09:15:22-06:00', '2026-02-05'), {
    days: -4,
    absDays: 4,
    kind: 'early',
  })
})

test('paymentTiming cruza el límite de año', () => {
  assert.deepEqual(paymentTiming('2027-01-02', '2026-12-31'), {
    days: 2,
    absDays: 2,
    kind: 'late',
  })
})

test('paymentTiming devuelve null con fechas faltantes o inválidas', () => {
  assert.equal(paymentTiming('', '2026-02-05'), null)
  assert.equal(paymentTiming('2026-02-01', ''), null)
  assert.equal(paymentTiming('invalida', '2026-02-05'), null)
  assert.equal(paymentTiming('2026-02-31', '2026-02-05'), null)
  assert.equal(paymentTiming('2026-13-01', '2026-02-05'), null)
})

// SPEC-128: semáforo con zona horaria del sistema y umbrales verde/amarillo/rojo.
const TZ = 'America/Managua'

test('mismo día de pago y vencimiento → en fecha y verde', () => {
  assert.deepEqual(getPaymentTiming('2026-03-10T15:00:00Z', '2026-03-10', TZ), {
    days: 0,
    status: 'on_time',
    tone: 'success',
  })
})

test('un día antes del vencimiento → temprano y verde', () => {
  assert.deepEqual(getPaymentTiming('2026-03-09T15:00:00Z', '2026-03-10', TZ), {
    days: -1,
    status: 'early',
    tone: 'success',
  })
})

test('varios días antes del vencimiento → temprano y verde', () => {
  assert.deepEqual(getPaymentTiming('2026-03-07T15:00:00Z', '2026-03-10', TZ), {
    days: -3,
    status: 'early',
    tone: 'success',
  })
})

test('un día tarde → amarillo', () => {
  assert.deepEqual(getPaymentTiming('2026-03-11T15:00:00Z', '2026-03-10', TZ), {
    days: 1,
    status: 'late',
    tone: 'warning',
  })
})

test('siete días tarde (límite amarillo) → amarillo', () => {
  assert.deepEqual(getPaymentTiming('2026-03-17T15:00:00Z', '2026-03-10', TZ), {
    days: PAYMENT_TIMING_WARNING_MAX_DAYS,
    status: 'late',
    tone: 'warning',
  })
})

test('ocho días tarde (primer día rojo) → rojo', () => {
  assert.deepEqual(getPaymentTiming('2026-03-18T15:00:00Z', '2026-03-10', TZ), {
    days: 8,
    status: 'late',
    tone: 'danger',
  })
})

test('treinta días tarde → rojo', () => {
  assert.deepEqual(getPaymentTiming('2026-04-09T15:00:00Z', '2026-03-10', TZ), {
    days: 30,
    status: 'late',
    tone: 'danger',
  })
})

test('cambio de mes y año se calcula por días de calendario', () => {
  assert.deepEqual(getPaymentTiming('2026-01-02T15:00:00Z', '2025-12-30', TZ), {
    days: 3,
    status: 'late',
    tone: 'warning',
  })
})

test('pago 23:30 local del día de vencimiento → en fecha aunque en UTC sea el día siguiente', () => {
  // 2026-03-11T05:30:00Z = 2026-03-10 23:30 en America/Managua (UTC-6).
  assert.deepEqual(getPaymentTiming('2026-03-11T05:30:00Z', '2026-03-10', TZ), {
    days: 0,
    status: 'on_time',
    tone: 'success',
  })
})

test('paymentDateInTimeZone convierte el timestamp a la fecha local', () => {
  assert.equal(paymentDateInTimeZone('2026-03-11T05:30:00Z', TZ), '2026-03-10')
})

test('paymentDateInTimeZone conserva una fecha sin hora', () => {
  assert.equal(paymentDateInTimeZone('2026-03-08', TZ), '2026-03-08')
})

test('una fecha sin hora inválida no se reinterpreta como otro día', () => {
  assert.equal(paymentDateInTimeZone('2026-02-30', TZ), null)
  assert.equal(getPaymentTiming('2026-02-30', '2026-03-10', TZ), null)
})

test('paid_at ausente o inválido → null, sin falso "en fecha"', () => {
  assert.equal(getPaymentTiming(null, '2026-03-10', TZ), null)
  assert.equal(getPaymentTiming(undefined, '2026-03-10', TZ), null)
  assert.equal(getPaymentTiming('not-a-date', '2026-03-10', TZ), null)
})

test('due_date ausente o inválida → null', () => {
  assert.equal(getPaymentTiming('2026-03-10T15:00:00Z', null, TZ), null)
  assert.equal(getPaymentTiming('2026-03-10T15:00:00Z', '', TZ), null)
  assert.equal(getPaymentTiming('2026-03-10T15:00:00Z', '2026-02-30', TZ), null)
  assert.equal(getPaymentTiming('2026-03-10T15:00:00Z', '10/03/2026', TZ), null)
})

test('zona horaria inválida → null en vez de excepción', () => {
  assert.equal(paymentDateInTimeZone('2026-03-10T15:00:00Z', 'Not/AZone'), null)
  assert.equal(getPaymentTiming('2026-03-10T15:00:00Z', '2026-03-10', 'Not/AZone'), null)
})
