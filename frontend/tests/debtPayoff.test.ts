import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  buildTimeline,
  lastPaymentDate,
  monthsUntil,
  nextPendingBill,
  overdueBills,
  paidBills,
  payoffState,
  pendingEndDate,
  progressPct,
  remainingTime,
  toDateOnly,
} from '../src/utils/debtPayoff.ts'
import type { Debt, DebtBill } from '../src/types/index.ts'

function makeDebt(overrides: Partial<Debt> = {}): Debt {
  return {
    id: 1,
    institution_id: 1,
    identifier: 'DEMO-001',
    description: 'Deuda Demo',
    total: 1200,
    principal: 1200,
    currency_id: 1,
    currency_code: 'XXX',
    installments_total: 12,
    installment_amount: 100,
    interest_rate: 0,
    payment_day: 5,
    start_date: '2026-01-05',
    status: 'activa',
    pending_count: 0,
    pending_amount: 0,
    paid_count: 0,
    paid_amount: 0,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

function makeBill(overrides: Partial<DebtBill> = {}): DebtBill {
  return {
    id: 1,
    debt_id: 1,
    installment_number: 1,
    due_date: '2026-12-05',
    amount: 100,
    status: 'pending',
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

test('toDateOnly acepta date-only, timestamps y RFC3339', () => {
  assert.equal(toDateOnly('2026-11-07'), '2026-11-07')
  assert.equal(toDateOnly('2026-11-07 00:00:00'), '2026-11-07')
  assert.equal(toDateOnly('2026-11-07 00:00:00 +0000 UTC'), '2026-11-07')
  assert.equal(toDateOnly('2026-11-07T12:34:56Z'), '2026-11-07')
  assert.equal(toDateOnly('2026-13-07'), null)
  assert.equal(toDateOnly('2026-02-30'), null)
  assert.equal(toDateOnly('invalida'), null)
  assert.equal(toDateOnly(''), null)
  assert.equal(toDateOnly(null), null)
})

test('monthsUntil cuenta meses calendario completos', () => {
  const now = new Date(2026, 10, 7) // 7 nov 2026
  assert.equal(monthsUntil(new Date(2029, 7, 15), now), 33)
  assert.equal(monthsUntil(new Date(2029, 7, 5), now), 32)
  assert.equal(monthsUntil(new Date(2027, 11, 7), now), 13)
  assert.equal(monthsUntil(new Date(2027, 11, 6), now), 12)
  assert.equal(monthsUntil(new Date(2027, 10, 7), now), 12)
  assert.equal(monthsUntil(new Date(2026, 11, 7), now), 1)
  assert.equal(monthsUntil(new Date(2026, 10, 7), now), 0)
  assert.equal(monthsUntil(new Date(2026, 9, 7), now), -1)
})

test('remainingTime distingue pasado, hoy, menos de un mes y meses', () => {
  const now = new Date(2026, 10, 7) // 7 nov 2026

  assert.deepEqual(remainingTime(new Date(2026, 10, 6), now), {
    totalMonths: -1,
    years: 0,
    months: 1,
    kind: 'past',
  })
  assert.deepEqual(remainingTime(new Date(2026, 10, 7), now), {
    totalMonths: 0,
    years: 0,
    months: 0,
    kind: 'due-today',
  })
  assert.deepEqual(remainingTime(new Date(2026, 10, 20), now), {
    totalMonths: 0,
    years: 0,
    months: 0,
    kind: 'less-than-month',
  })
  assert.deepEqual(remainingTime(new Date(2026, 11, 7), now), {
    totalMonths: 1,
    years: 0,
    months: 1,
    kind: 'months',
  })
  assert.deepEqual(remainingTime(new Date(2027, 11, 7), now), {
    totalMonths: 13,
    years: 1,
    months: 1,
    kind: 'months',
  })
  assert.deepEqual(remainingTime(new Date(2028, 11, 7), now), {
    totalMonths: 25,
    years: 2,
    months: 1,
    kind: 'months',
  })
  assert.deepEqual(remainingTime(new Date(2029, 7, 15), now), {
    totalMonths: 33,
    years: 2,
    months: 9,
    kind: 'months',
  })
})

test('nextPendingBill ordena por vencimiento y desempata por número', () => {
  const bills = [
    makeBill({ id: 1, installment_number: 2, due_date: '2026-12-15' }),
    makeBill({ id: 2, installment_number: 3, due_date: '2026-11-15' }),
    makeBill({ id: 3, installment_number: 1, due_date: '2026-11-15' }),
    makeBill({ id: 4, installment_number: 4, due_date: '2026-10-15', status: 'paid', paid_at: '2026-10-14' }),
  ]
  assert.equal(nextPendingBill(bills)?.id, 3)
  assert.equal(nextPendingBill([]), null)
})

test('overdueBills no marca la cuota que vence hoy', () => {
  const bills = [
    makeBill({ id: 1, due_date: '2026-11-06' }),
    makeBill({ id: 2, due_date: '2026-11-07' }),
    makeBill({ id: 3, due_date: '2026-11-08' }),
    makeBill({ id: 4, due_date: '2026-11-01', status: 'paid', paid_at: '2026-10-31' }),
  ]
  assert.deepEqual(overdueBills(bills, '2026-11-07').map((b) => b.id), [1])
})

test('lastPaymentDate normaliza formatos y cae a las cuotas pagadas', () => {
  assert.equal(
    lastPaymentDate(makeDebt({ last_paid_at: '2026-10-09 00:00:00 +0000 UTC' }), []),
    '2026-10-09',
  )
  assert.equal(
    lastPaymentDate(makeDebt({ last_paid_at: '2026-10-09T12:00:00Z' }), []),
    '2026-10-09',
  )
  assert.equal(
    lastPaymentDate(makeDebt(), [
      makeBill({ id: 1, status: 'paid', paid_at: '2026-09-09' }),
      makeBill({ id: 2, status: 'paid', paid_at: '2026-10-09' }),
    ]),
    '2026-10-09',
  )
  assert.equal(lastPaymentDate(makeDebt(), []), null)
})

test('pendingEndDate usa la última cuota pendiente y cae a last_due_date', () => {
  const bills = [
    makeBill({ id: 1, due_date: '2026-11-15' }),
    makeBill({ id: 2, due_date: '2029-08-15' }),
    makeBill({ id: 3, due_date: '2026-10-15', status: 'paid', paid_at: '2026-10-14' }),
  ]
  assert.equal(pendingEndDate(makeDebt(), bills), '2029-08-15')
  assert.equal(pendingEndDate(makeDebt({ last_due_date: '2029-09-15' }), []), '2029-09-15')
  assert.equal(pendingEndDate(makeDebt(), []), null)
})

test('buildTimeline ordena, fusiona fechas y clampea posiciones', () => {
  const points = buildTimeline({
    startDate: '2026-01-05',
    endDate: '2029-08-15',
    nextDueDate: '2026-12-05',
    todayISO: '2026-11-07',
  })
  assert.deepEqual(points.map((p) => p.key), ['start', 'today', 'next', 'end'])
  assert.equal(points[0].position, 0)
  assert.equal(points[points.length - 1].position, 100)
  assert.ok(points[1].position > 0 && points[1].position < points[2].position)
})

test('buildTimeline omite el inicio futuro y la próxima posterior al fin', () => {
  const points = buildTimeline({
    startDate: '2027-01-05',
    endDate: '2029-08-15',
    nextDueDate: '2029-09-15',
    todayISO: '2026-11-07',
  })
  assert.deepEqual(points.map((p) => p.key), ['today', 'end'])
})

test('buildTimeline marca vencida la próxima anterior a hoy', () => {
  const points = buildTimeline({
    startDate: '2026-01-05',
    endDate: '2029-08-15',
    nextDueDate: '2026-10-05',
    todayISO: '2026-11-07',
  })
  const next = points.find((p) => p.key === 'next')
  assert.equal(next?.state, 'overdue')
  assert.equal(next?.label, 'overdue')
  assert.ok((next?.position ?? 0) < (points.find((p) => p.key === 'today')?.position ?? 0))
})

test('buildTimeline maneja fin pasado y fechas repetidas', () => {
  const past = buildTimeline({
    startDate: '2026-01-05',
    endDate: '2026-10-05',
    nextDueDate: '2026-10-05',
    todayISO: '2026-11-07',
  })
  assert.deepEqual(past.map((p) => p.key), ['start', 'end', 'today'])
  assert.equal(past.find((p) => p.key === 'end')?.state, 'overdue')
  assert.equal(past.find((p) => p.key === 'today')?.position, 100)

  const same = buildTimeline({
    startDate: '2026-11-07',
    endDate: '2026-12-05',
    nextDueDate: '2026-12-05',
    todayISO: '2026-11-07',
  })
  assert.deepEqual(same.map((p) => p.key), ['today', 'end'])
})

test('payoffState respeta la precedencia de estados', () => {
  const today = '2026-11-07'
  const paid = [makeBill({ status: 'paid', paid_at: '2026-11-01' })]

  assert.equal(payoffState(makeDebt({ pending_count: 0 }), paid, today), 'paid')
  assert.equal(
    payoffState(makeDebt({ pending_count: 0, status: 'finalizada' }), [], today),
    'paid',
  )
  assert.equal(
    payoffState(makeDebt({ pending_count: 3, installments_total: 12 }), [], today),
    'missing-bills',
  )
  assert.equal(
    payoffState(makeDebt({ pending_count: 3, status: 'finalizada' }), [], today),
    'missing-bills',
  )
  assert.equal(
    payoffState(makeDebt({ pending_count: 0, installments_total: 0 }), [], today),
    'no-bills',
  )
  assert.equal(
    payoffState(makeDebt({ status: 'inactiva', pending_count: 1 }), [makeBill({ due_date: '2026-12-05' })], today),
    'paused',
  )
  assert.equal(
    payoffState(makeDebt({ pending_count: 1 }), [makeBill({ due_date: '2026-11-06' })], today),
    'overdue',
  )
  assert.equal(
    payoffState(makeDebt({ pending_count: 1 }), [makeBill({ due_date: '2026-11-08' })], today),
    'active',
  )
  assert.equal(
    payoffState(makeDebt({ status: 'finalizada', pending_count: 1 }), [makeBill({ due_date: '2026-11-08' })], today),
    'active',
  )
  assert.equal(
    payoffState(makeDebt({ pending_count: 0 }), [makeBill({ due_date: '2026-11-08' })], today),
    'active',
  )
})

test('progressPct limita a 99 con pendientes y a 100 sin pendientes', () => {
  assert.equal(progressPct(7, 24, true), 29)
  assert.equal(progressPct(24, 24, false), 100)
  assert.equal(progressPct(24, 24, true), 99)
  assert.equal(progressPct(0, 0, true), 0)
  assert.equal(progressPct(5, 4, true), 99)
})

test('paidBills filtra solo las pagadas', () => {
  const bills = [
    makeBill({ id: 1, status: 'paid' }),
    makeBill({ id: 2, status: 'pending' }),
  ]
  assert.deepEqual(paidBills(bills).map((b) => b.id), [1])
})
