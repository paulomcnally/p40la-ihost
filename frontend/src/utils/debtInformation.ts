import type {
  Debt,
  DebtBill,
  DebtInformationField,
  DebtInformationValue,
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

const TOKEN_RE = /\{\{\s*([a-z0-9_.]+)\s*\}\}/g

export interface DebtInformationValidationIssue {
  key: string
  fieldIndex?: number
  params?: Record<string, string | number>
}

export interface DebtInformationContextInput {
  debt: Pick<Debt, 'description' | 'institution_name' | 'identifier' | 'currency_code' | 'total'>
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

/** Valida los valores de una forma de pago contra su plantilla. */
export function validateDebtInformationValues(
  fields: DebtInformationField[],
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
