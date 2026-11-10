import type {
  Debt,
  DebtBill,
  DebtInformationField,
  DebtInformationValue,
  DebtInformationVariable,
} from '../types'

// SPEC-131: utilidades puras de plantillas de información de deuda.

export const DEBT_INFORMATION_FIELD_TYPES: DebtInformationField['type'][] = [
  'text',
  'textarea',
  'number',
  'date',
  'select',
  'checkbox',
  'computed',
]

export const DEBT_INFORMATION_TOKEN_KEYS = [
  'deuda.descripcion',
  'deuda.institucion',
  'deuda.identificador',
  'deuda.moneda',
  'deuda.total',
  'deuda.total_formateado',
  'cuota.numero',
  'cuota.monto',
  'cuota.monto_formateado',
  'cuota.vencimiento',
  'cuota.vencimiento_formateado',
  'cuota.estado',
  'pago.fecha',
  'pago.fecha_formateada',
  'sistema.moneda',
] as const

export const DEBT_INFORMATION_TOKEN_SET = new Set<string>(DEBT_INFORMATION_TOKEN_KEYS)

// SPEC-132: variables para valores por defecto de campos text/textarea.
// La clave se persiste en inglés; la etiqueta se sugiere en español.
export const DEBT_INFORMATION_VARIABLE_FALLBACK: DebtInformationVariable[] = [
  { key: 'installment_number', label: 'Número de cuota', example: '5', description: 'Número de la cuota que se está pagando' },
  { key: 'installment_amount', label: 'Monto de la cuota (sin formato)', example: '100.00', description: 'Monto de la cuota sin símbolo ni separadores' },
  { key: 'installment_amount_formatted', label: 'Monto de la cuota con moneda', example: '$100.00', description: 'Monto de la cuota formateado con el símbolo de la moneda' },
  { key: 'installment_due_date', label: 'Vencimiento de la cuota (ISO)', example: '2026-12-15', description: 'Fecha de vencimiento en formato YYYY-MM-DD' },
  { key: 'installment_due_date_formatted', label: 'Vencimiento de la cuota (localizado)', example: '15/12/2026', description: 'Fecha de vencimiento en formato local' },
  { key: 'installment_status', label: 'Estado de la cuota', example: 'Pendiente', description: 'Pendiente o Pagada, según el idioma' },
  { key: 'installment_progress', label: 'Progreso de cuotas', example: '5/12', description: 'Cuota actual sobre el total del plan' },
  { key: 'installments_total', label: 'Total de cuotas del plan', example: '12', description: 'Cantidad total de cuotas de la deuda' },
  { key: 'debt_description', label: 'Descripción de la deuda', example: 'Lote Demo 13', description: 'Descripción de la deuda' },
  { key: 'debt_identifier', label: 'Identificador de la deuda', example: 'demo-account-001', description: 'Identificador o número de contrato' },
  { key: 'institution_name', label: 'Institución o acreedor', example: 'Proveedor Demo', description: 'Nombre de la institución de la deuda' },
  { key: 'currency_code', label: 'Código de moneda', example: 'USD', description: 'Código ISO de la moneda de la deuda' },
  { key: 'currency_symbol', label: 'Símbolo de moneda', example: '$', description: 'Símbolo de la moneda de la deuda' },
  { key: 'debt_total', label: 'Total de la deuda (sin formato)', example: '1200.00', description: 'Total de la deuda sin formato' },
  { key: 'debt_total_formatted', label: 'Total de la deuda con moneda', example: '$1,200.00', description: 'Total de la deuda formateado' },
  { key: 'payment_date', label: 'Fecha de pago (ISO)', example: '2026-12-10', description: 'Fecha de pago seleccionada, si ya se pagó' },
  { key: 'payment_date_formatted', label: 'Fecha de pago (localizada)', example: '10/12/2026', description: 'Fecha de pago en formato local' },
]

export const DEBT_INFORMATION_VARIABLE_KEYS = DEBT_INFORMATION_VARIABLE_FALLBACK.map((variable) => variable.key)
export const DEBT_INFORMATION_VARIABLE_SET = new Set<string>(DEBT_INFORMATION_VARIABLE_KEYS)

const VARIABLE_RE = /\{\s*([a-z][a-z0-9_]*)\s*\}/g

const TOKEN_RE = /\{\{\s*([a-z0-9_.]+)\s*\}\}/g

export interface DebtInformationValidationIssue {
  key: string
  fieldIndex?: number
  params?: Record<string, string | number>
}

export interface DebtInformationContextInput {
  debt: Pick<Debt, 'description' | 'institution_name' | 'identifier' | 'currency_code' | 'total' | 'installments_total'>
  bill: Pick<DebtBill, 'installment_number' | 'amount' | 'due_date' | 'status'>
  symbol?: string
  lang: string
  paidAt?: string
  statusLabel: string
  formatMoney: (amount: number, symbol?: string) => string
}

/** Convierte una etiqueta en una clave estable: minúsculas, sin acentos y con `_`. */
export function slugifyFieldKey(label: string): string {
  const normalized = label
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
  let key = normalized
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 40)

  if (!/^[a-z]/.test(key)) {
    key = `campo_${key}`.slice(0, 40).replace(/_+$/g, '')
  }
  return key || 'campo'
}

/** Reemplaza tokens `{{clave}}` por su valor. Los desconocidos quedan literales. */
export function resolveDebtInformationText(
  template: string,
  context: Record<string, string>,
): string {
  if (!template) return ''
  return template.replace(TOKEN_RE, (match, key: string) => context[key] ?? match)
}

/** Reemplaza variables `{clave}` por su valor. Los desconocidos quedan literales. */
export function resolveDebtInformationVariables(
  template: string,
  context: Record<string, string>,
): string {
  if (!template) return ''
  return template.replace(VARIABLE_RE, (match, key: string) => context[key] ?? match)
}

/** Devuelve las claves de variable presentes en un texto `{clave}`. */
export function extractDebtInformationVariables(template: string): string[] {
  if (!template) return []
  const keys: string[] = []
  for (const match of template.matchAll(VARIABLE_RE)) {
    keys.push(match[1])
  }
  return keys
}

/** Contexto de ejemplo (catálogo) para previsualizar valores por defecto. */
export function buildExampleVariableContext(
  variables: DebtInformationVariable[],
): Record<string, string> {
  const context: Record<string, string> = {}
  for (const variable of variables) {
    context[variable.key] = variable.example
  }
  return context
}

/** Valida una fecha ISO YYYY-MM-DD real (rechaza 2026-02-31). */
export function isValidDebtInformationDate(iso: string): boolean {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso || '')
  if (!match) return false
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  const date = new Date(Date.UTC(year, month - 1, day))
  return (
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day
  )
}

/** Formatea una fecha ISO YYYY-MM-DD sin depender de la zona horaria. */
export function formatDebtInformationDate(iso: string, lang: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso || '')
  if (!match) return iso || ''
  const [, year, month, day] = match
  return lang === 'en' ? `${month}/${day}/${year}` : `${day}/${month}/${year}`
}

export function buildDebtInformationContext(
  input: DebtInformationContextInput,
): Record<string, string> {
  const { debt, bill, symbol, lang, paidAt, statusLabel, formatMoney } = input
  return {
    'deuda.descripcion': debt.description || '',
    'deuda.institucion': debt.institution_name || '',
    'deuda.identificador': debt.identifier || '',
    'deuda.moneda': debt.currency_code || '',
    'deuda.total': debt.total == null ? '' : String(debt.total),
    'deuda.total_formateado': formatMoney(debt.total || 0, symbol),
    'cuota.numero': bill.installment_number == null ? '' : String(bill.installment_number),
    'cuota.monto': bill.amount == null ? '' : String(bill.amount),
    'cuota.monto_formateado': formatMoney(bill.amount || 0, symbol),
    'cuota.vencimiento': bill.due_date || '',
    'cuota.vencimiento_formateado': formatDebtInformationDate(bill.due_date || '', lang),
    'cuota.estado': statusLabel,
    'pago.fecha': paidAt || '',
    'pago.fecha_formateada': formatDebtInformationDate(paidAt || '', lang),
    'sistema.moneda': symbol || '',
  }
}

/** Contexto en inglés para resolver variables `{clave}` de SPEC-132. */
export function buildDebtInformationVariableContext(
  input: DebtInformationContextInput,
): Record<string, string> {
  const { debt, bill, symbol, lang, paidAt, statusLabel, formatMoney } = input
  const installmentNumber = bill.installment_number == null ? '' : String(bill.installment_number)
  const installmentsTotal = debt.installments_total == null ? '' : String(debt.installments_total)
  return {
    installment_number: installmentNumber,
    installment_amount: bill.amount == null ? '' : String(bill.amount),
    installment_amount_formatted: formatMoney(bill.amount || 0, symbol),
    installment_due_date: bill.due_date || '',
    installment_due_date_formatted: formatDebtInformationDate(bill.due_date || '', lang),
    installment_status: statusLabel,
    installment_progress: installmentNumber && installmentsTotal ? `${installmentNumber}/${installmentsTotal}` : installmentNumber,
    installments_total: installmentsTotal,
    debt_description: debt.description || '',
    debt_identifier: debt.identifier || '',
    institution_name: debt.institution_name || '',
    currency_code: debt.currency_code || '',
    currency_symbol: symbol || '',
    debt_total: debt.total == null ? '' : String(debt.total),
    debt_total_formatted: formatMoney(debt.total || 0, symbol),
    payment_date: paidAt || '',
    payment_date_formatted: formatDebtInformationDate(paidAt || '', lang),
  }
}

/** Valida la definición de una plantilla. Devuelve claves i18n de error. */
export function validateDebtInformationTemplate(
  name: string,
  fields: DebtInformationField[],
): DebtInformationValidationIssue[] {
  const issues: DebtInformationValidationIssue[] = []
  if (!name.trim()) {
    issues.push({ key: 'deudas.validation_name_required' })
  } else if (name.trim().length > 100) {
    issues.push({ key: 'deudas.validation_name_length' })
  }
  if (fields.length === 0) {
    issues.push({ key: 'deudas.validation_fields_required' })
  }
  if (fields.length > 30) {
    issues.push({ key: 'deudas.validation_fields_max' })
  }

  const keys = new Set<string>()
  fields.forEach((field, index) => {
    if (!field.label.trim()) {
      issues.push({ key: 'deudas.validation_label_required', fieldIndex: index })
    }
    if (!/^[a-z][a-z0-9_]{0,39}$/.test(field.key)) {
      issues.push({ key: 'deudas.validation_key_invalid', fieldIndex: index })
    } else if (keys.has(field.key)) {
      issues.push({
        key: 'deudas.validation_key_duplicate',
        fieldIndex: index,
        params: { key: field.key },
      })
    }
    keys.add(field.key)

    if (field.type === 'computed') {
      if (!field.value_template?.trim()) {
        issues.push({ key: 'deudas.validation_computed_required', fieldIndex: index })
      } else {
        const tokens = field.value_template.matchAll(/\{\{\s*([a-z0-9_.]+)\s*\}\}/g)
        for (const token of tokens) {
          if (!DEBT_INFORMATION_TOKEN_SET.has(token[1])) {
            issues.push({
              key: 'deudas.validation_token',
              fieldIndex: index,
              params: { token: token[1] },
            })
          }
        }
        const rest = field.value_template.replace(/\{\{\s*([a-z0-9_.]+)\s*\}\}/g, '')
        if (rest.includes('{{') || rest.includes('}}')) {
          issues.push({ key: 'deudas.validation_token_malformed', fieldIndex: index })
        }
      }
    } else if (field.default_value) {
      validateDefaultValueVariables(field, index, issues)
    }

    if (field.type === 'select') {
      const options = (field.options || []).map((option) => option.trim()).filter(Boolean)
      if (options.length === 0) {
        issues.push({ key: 'deudas.validation_select_options', fieldIndex: index })
      }
    }
    if (field.type !== 'computed' && (field.label.length > 100 || (field.help_text?.length ?? 0) > 300)) {
      issues.push({ key: 'deudas.validation_value_length', fieldIndex: index })
    }
  })

  return issues
}

function validateDefaultValueVariables(
  field: DebtInformationField,
  index: number,
  issues: DebtInformationValidationIssue[],
): void {
  const value = field.default_value || ''
  if (field.type === 'text' || field.type === 'textarea') {
    if (value.includes('{{') || value.includes('}}')) {
      issues.push({ key: 'deudas.validation_variable_braces', fieldIndex: index })
      return
    }
    for (const key of extractDebtInformationVariables(value)) {
      if (!DEBT_INFORMATION_VARIABLE_SET.has(key)) {
        issues.push({
          key: 'deudas.validation_variable',
          fieldIndex: index,
          params: { token: key },
        })
      }
    }
    const rest = value.replace(VARIABLE_RE, '')
    if (rest.includes('{') || rest.includes('}')) {
      issues.push({ key: 'deudas.validation_variable_malformed', fieldIndex: index })
    }
    return
  }
  if (value.includes('{') || value.includes('}')) {
    issues.push({ key: 'deudas.validation_variable_type', fieldIndex: index })
  }
}

/** Valida los valores de una forma de pago contra su plantilla. */
export function validateDebtInformationValues(  fields: DebtInformationField[],
  values: Record<string, DebtInformationValue>,
): DebtInformationValidationIssue[] {
  const issues: DebtInformationValidationIssue[] = []

  fields.forEach((field, index) => {
    if (field.type === 'computed') return
    const raw = values[field.key]
    const empty = raw == null || (typeof raw === 'string' && raw.trim() === '')
    const required = field.required && !(field.type === 'checkbox' && raw === true)

    if (empty || (field.type === 'checkbox' && field.required && raw !== true)) {
      if (required) {
        issues.push({
          key: 'deudas.validation_required',
          fieldIndex: index,
          params: { label: field.label },
        })
      }
      return
    }

    if (field.type === 'number' && !Number.isFinite(Number(raw))) {
      issues.push({
        key: 'deudas.validation_number',
        fieldIndex: index,
        params: { label: field.label },
      })
    }
    if (field.type === 'date' && !isValidDebtInformationDate(String(raw))) {
      issues.push({
        key: 'deudas.validation_date',
        fieldIndex: index,
        params: { label: field.label },
      })
    }
    if (field.type === 'select' && !(field.options || []).includes(String(raw))) {
      issues.push({
        key: 'deudas.validation_select',
        fieldIndex: index,
        params: { label: field.label },
      })
    }
    if (field.type === 'text' && String(raw).length > 500) {
      issues.push({
        key: 'deudas.validation_value_length',
        fieldIndex: index,
        params: { label: field.label },
      })
    }
    if (field.type === 'textarea' && String(raw).length > 5000) {
      issues.push({
        key: 'deudas.validation_value_length',
        fieldIndex: index,
        params: { label: field.label },
      })
    }
  })

  return issues
}

export function debtInformationValueIsEmpty(value: DebtInformationValue | undefined): boolean {
  return value == null || (typeof value === 'string' && value.trim() === '')
}
