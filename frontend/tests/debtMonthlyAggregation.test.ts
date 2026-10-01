import { test } from 'node:test'
import assert from 'node:assert/strict'
import { debtEndMonth, aggregateDebtsByEndMonth } from '../src/utils/debtMonthlyAggregation.ts'
import type { Debt } from '../src/types/index.ts'

function makeDebt(overrides: Partial<Debt>): Debt {
  return {
    id: 1,
    institution_id: 1,
    identifier: '',
    description: 'Deuda',
    total: 100,
    principal: 100,
    currency_id: 1,
    currency_code: 'NIO',
    installments_total: 12,
    installment_amount: 10,
    interest_rate: 0,
    payment_day: 5,
    start_date: '2026-01-10',
    status: 'activa',
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

test('debtEndMonth deriva el mes de finalización', () => {
  assert.deepEqual(debtEndMonth(makeDebt({ start_date: '2026-01-10', installments_total: 12 })), {
    year: 2027,
    month: 1,
  })
  assert.deepEqual(debtEndMonth(makeDebt({ start_date: '2026-01-10', installments_total: 1 })), {
    year: 2026,
    month: 2,
  })
})

test('debtEndMonth cruza el límite de año', () => {
  assert.deepEqual(debtEndMonth(makeDebt({ start_date: '2026-11-20', installments_total: 2 })), {
    year: 2027,
    month: 1,
  })
})

test('debtEndMonth devuelve null con datos inválidos', () => {
  assert.equal(debtEndMonth(makeDebt({ start_date: '' })), null)
  assert.equal(debtEndMonth(makeDebt({ installments_total: 0 })), null)
  assert.equal(debtEndMonth(makeDebt({ start_date: 'invalida' })), null)
  assert.equal(debtEndMonth(makeDebt({ start_date: '2026-13-10' })), null)
})

test('aggregateDebtsByEndMonth agrupa y ordena de más cercana a más lejana', () => {
  const debts = [
    makeDebt({ id: 1, start_date: '2025-01-10', installments_total: 12, total: 100 }), // ene 2026
    makeDebt({ id: 2, start_date: '2026-01-10', installments_total: 1, total: 200 }), // feb 2026
    makeDebt({ id: 3, start_date: '2025-01-10', installments_total: 12, total: 50 }), // ene 2026
  ]
  const buckets = aggregateDebtsByEndMonth(debts, debtEndMonth, (m) => `M${m}`)
  assert.equal(buckets.length, 2)
  assert.equal(buckets[0].label, 'M2 2026')
  assert.equal(buckets[0].count, 1)
  assert.equal(buckets[0].total, 200)
  assert.equal(buckets[1].label, 'M1 2026')
  assert.equal(buckets[1].count, 2)
  assert.equal(buckets[1].total, 150)
})

test('aggregateDebtsByEndMonth excluye meses sin deudas y deudas sin fecha válida', () => {
  const debts = [
    makeDebt({ id: 1, start_date: '2026-01-10', installments_total: 12 }),
    makeDebt({ id: 2, start_date: '', installments_total: 12 }),
  ]
  const buckets = aggregateDebtsByEndMonth(debts, debtEndMonth, (m) => String(m))
  assert.equal(buckets.length, 1)
  assert.equal(buckets[0].debts.length, 1)
})