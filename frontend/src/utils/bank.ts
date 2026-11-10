import type {
  Bank,
  BankAccount,
  BankAccountType,
  BankTransferKind,
  DebtInformationVariable,
} from '../types'

// SPEC-133: utilidades puras del módulo de bancos.

export const BANK_ACCOUNT_TYPE_VALUES: BankAccountType[] = ['savings', 'checking']

export const BANK_TRANSFER_KINDS: BankTransferKind[] = ['transfer', 'ach']

export const BANK_TRANSFER_SUGGESTED_KEYS: Record<BankTransferKind, string[]> = {
  transfer: ['from', 'to', 'amount', 'description', 'beneficiary'],
  ach: ['from', 'to', 'bank', 'beneficiary', 'amount', 'currency', 'description'],
}

export const BANK_TRANSFER_VARIABLE_FALLBACK: DebtInformationVariable[] = [
  { key: 'bank_name', label: 'Nombre del banco', example: 'Banco Demo', description: 'Nombre del banco de la cuenta' },
  { key: 'bank_short_code', label: 'Código corto del banco', example: 'BDEMO', description: 'Código corto del banco' },
  { key: 'account_alias', label: 'Alias de la cuenta', example: 'Cuenta principal', description: 'Alias o etiqueta de la cuenta' },
  { key: 'account_number', label: 'Número de cuenta', example: '0000000001', description: 'Número de la cuenta bancaria' },
  { key: 'account_type', label: 'Tipo de cuenta', example: 'Ahorro', description: 'Ahorro o Corriente, según el idioma' },
  { key: 'account_beneficiary', label: 'Beneficiario de la cuenta', example: 'Beneficiario Demo', description: 'Nombre del beneficiario de la cuenta' },
  { key: 'account_currency', label: 'Moneda de la cuenta', example: 'COR', description: 'Código de la moneda de la cuenta' },
  { key: 'account_currency_code', label: 'Código de moneda de la cuenta', example: 'COR', description: 'Alias de account_currency' },
  { key: 'account_currency_symbol', label: 'Símbolo de la moneda de la cuenta', example: 'C$', description: 'Símbolo de la moneda de la cuenta' },
  { key: 'account_notes', label: 'Notas de la cuenta', example: 'Cuenta para pagos', description: 'Notas de la cuenta bancaria' },
]

export interface BankFormatFieldRow {
  key: string
  value: string
}

const KEY_RE = /^[a-z][a-z0-9_]{0,39}$/
const VARIABLE_RE = /\{\s*([a-z][a-z0-9_]*)\s*\}/g

export function bankAccountTypeLabel(type: BankAccountType, t: (key: string) => string): string {
  return type === 'checking' ? t('banks.account_type_checking') : t('banks.account_type_savings')
}

export function fieldsToRows(
  fields: Record<string, string> | undefined,
  kind: BankTransferKind,
): BankFormatFieldRow[] {
  const entries = Object.entries(fields || {})
  if (entries.length > 0) {
    return entries.map(([key, value]) => ({ key, value }))
  }
  return BANK_TRANSFER_SUGGESTED_KEYS[kind].map((key) => ({ key, value: '' }))
}

export function rowsToFields(rows: BankFormatFieldRow[]): Record<string, string> {
  const fields: Record<string, string> = {}
  for (const row of rows) {
    const key = row.key.trim()
    if (!key) continue
    fields[key] = row.value
  }
  return fields
}

/** Devuelve la clave i18n del primer error o null si el formato es válido. */
export function validateBankFormatRows(rows: BankFormatFieldRow[]): string | null {
  if (rows.length > 30) return 'banks.validation_fields_max'
  const seen = new Set<string>()
  for (const row of rows) {
    const key = row.key.trim()
    if (!KEY_RE.test(key)) return 'banks.validation_key'
    if (seen.has(key)) return 'banks.validation_key_duplicate'
    seen.add(key)
    if (row.value.length > 500) return 'banks.validation_value'
    if (row.value.includes('{{') || row.value.includes('}}')) return 'banks.validation_braces'
    const rest = row.value.replace(VARIABLE_RE, '')
    if (rest.includes('{') || rest.includes('}')) return 'banks.validation_variable'
  }
  return null
}

/** Devuelve las variables desconocidas del formato (vacío si todas existen). */
export function unknownBankFormatVariables(
  rows: BankFormatFieldRow[],
  allowed: Set<string>,
): string[] {
  const unknown = new Set<string>()
  for (const row of rows) {
    for (const match of row.value.matchAll(VARIABLE_RE)) {
      if (!allowed.has(match[1])) unknown.add(match[1])
    }
  }
  return Array.from(unknown)
}

export function buildBankAccountVariableContext(
  bank: Bank | null,
  account: BankAccount,
  accountTypeLabel: string,
): Record<string, string> {
  return {
    bank_name: bank?.name || account.bank_name || '',
    bank_short_code: bank?.short_code || '',
    account_alias: account.alias || '',
    account_number: account.account_number || '',
    account_type: accountTypeLabel,
    account_beneficiary: account.beneficiary || '',
    account_currency: account.currency_code || '',
    account_currency_code: account.currency_code || '',
    account_currency_symbol: account.currency_symbol || '',
    account_notes: account.notes || '',
  }
}

export function bankAccountTypeFromValue(value: string): BankAccountType {
  return value === 'checking' ? 'checking' : 'savings'
}
