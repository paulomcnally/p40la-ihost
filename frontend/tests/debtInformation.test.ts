import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  buildDebtInformationContext,
  formatDebtInformationDate,
  resolveDebtInformationText,
  slugifyFieldKey,
  validateDebtInformationTemplate,
  validateDebtInformationValues,
} from '../src/utils/debtInformation.ts'
import type { DebtInformationField } from '../src/types/index.ts'

// SPEC-131: plantillas de información de deuda.

test('slugifyFieldKey genera claves ASCII y estables', () => {
  assert.equal(slugifyFieldKey('Monto de la cuota'), 'monto_de_la_cuota')
  assert.equal(slugifyFieldKey('# Cuenta'), 'cuenta')
  assert.equal(slugifyFieldKey('Descripción'), 'descripcion')
  assert.equal(slugifyFieldKey('123 banco'), 'campo_123_banco')
  assert.equal(slugifyFieldKey('   '), 'campo')
})

test('resolveDebtInformationText reemplaza tokens y conserva los desconocidos', () => {
  const context = {
    'deuda.descripcion': 'Lote Demo 1',
    'cuota.numero': '12',
    'cuota.monto_formateado': '$100.00',
  }
  assert.equal(
    resolveDebtInformationText('{{deuda.descripcion}} - Cuota {{cuota.numero}}', context),
    'Lote Demo 1 - Cuota 12',
  )
  assert.equal(
    resolveDebtInformationText('Monto: {{ cuota.monto_formateado }}', context),
    'Monto: $100.00',
  )
  assert.equal(
    resolveDebtInformationText('{{cuota.total}}', context),
    '{{cuota.total}}',
  )
})

test('formatDebtInformationDate no desfasa la fecha', () => {
  assert.equal(formatDebtInformationDate('2026-12-10', 'es'), '10/12/2026')
  assert.equal(formatDebtInformationDate('2026-12-10', 'en'), '12/10/2026')
  assert.equal(formatDebtInformationDate('', 'es'), '')
})

test('buildDebtInformationContext arma los tokens de deuda y cuota', () => {
  const context = buildDebtInformationContext({
    debt: {
      description: 'Lote Demo 1',
      institution_name: 'Banco Demo',
      identifier: 'demo-account-001',
      currency_code: 'USD',
      total: 100,
    },
    bill: {
      installment_number: 12,
      amount: 100,
      due_date: '2026-12-15',
      status: 'pending',
    },
    symbol: '$',
    lang: 'es',
    paidAt: '2026-12-10',
    statusLabel: 'Pendiente',
    formatMoney: (amount, symbol) => `${symbol || ''}${amount.toFixed(2)}`,
  })

  assert.equal(context['deuda.descripcion'], 'Lote Demo 1')
  assert.equal(context['cuota.numero'], '12')
  assert.equal(context['cuota.monto_formateado'], '$100.00')
  assert.equal(context['cuota.vencimiento_formateado'], '15/12/2026')
  assert.equal(context['pago.fecha_formateada'], '10/12/2026')
  assert.equal(context['sistema.moneda'], '$')
})

test('validateDebtInformationTemplate acepta una plantilla válida', () => {
  const fields: DebtInformationField[] = [
    { key: 'monto', label: 'Monto', type: 'computed', value_template: '{{cuota.monto_formateado}}' },
    { key: 'banco', label: 'Banco', type: 'text', required: true },
  ]
  assert.deepEqual(validateDebtInformationTemplate('Transferencia Demo', fields), [])
})

test('validateDebtInformationTemplate detecta errores', () => {
  const fields: DebtInformationField[] = [
    { key: 'monto', label: 'Monto', type: 'computed', value_template: '{{cuota.total}}' },
    { key: 'metodo', label: 'Método', type: 'select', options: [] },
    { key: 'monto', label: '', type: 'text' },
  ]
  const issues = validateDebtInformationTemplate('', fields)
  const keys = issues.map((issue) => issue.key)
  assert.ok(keys.includes('deudas.validation_name_required'))
  assert.ok(keys.includes('deudas.validation_token'))
  assert.ok(keys.includes('deudas.validation_select_options'))
  assert.ok(keys.includes('deudas.validation_key_duplicate'))
  assert.ok(keys.includes('deudas.validation_label_required'))
})

test('validateDebtInformationValues valida requeridos, tipos y opciones', () => {
  const fields: DebtInformationField[] = [
    { key: 'banco', label: 'Banco', type: 'text', required: true },
    { key: 'monto', label: 'Monto', type: 'number' },
    { key: 'fecha', label: 'Fecha', type: 'date' },
    { key: 'metodo', label: 'Método', type: 'select', options: ['Transferencia', 'Efectivo'] },
    { key: 'acepta', label: 'Acepta', type: 'checkbox', required: true },
    { key: 'monto_calculado', label: 'Calculado', type: 'computed', value_template: '{{cuota.monto}}' },
  ]

  assert.ok(
    validateDebtInformationValues(fields, {}).some(
      (issue) => issue.key === 'deudas.validation_required',
    ),
  )

  const valid = validateDebtInformationValues(fields, {
    banco: 'Banco Demo',
    monto: 100,
    fecha: '2026-12-10',
    metodo: 'Transferencia',
    acepta: true,
  })
  assert.deepEqual(valid, [])

  const invalid = validateDebtInformationValues(fields, {
    banco: 'Banco Demo',
    monto: 'no-es-numero',
    fecha: '2026-02-31',
    metodo: 'Tarjeta',
    acepta: false,
  }).map((issue) => issue.key)
  assert.ok(invalid.includes('deudas.validation_number'))
  assert.ok(invalid.includes('deudas.validation_date'))
  assert.ok(invalid.includes('deudas.validation_select'))
  assert.ok(invalid.includes('deudas.validation_required'))
})
