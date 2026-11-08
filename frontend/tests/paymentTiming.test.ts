import { test } from 'node:test'
import assert from 'node:assert/strict'
import { paymentTiming } from '../src/utils/paymentTiming.ts'

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
