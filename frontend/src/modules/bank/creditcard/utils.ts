import type {
  CardNetwork,
  CardTransactionType,
  StatementStatus,
} from './types'

type Translator = (key: string, fallback?: string) => string

export const CARD_NETWORKS: CardNetwork[] = ['visa', 'mastercard', 'amex', 'diners', 'other']

export const TRANSACTION_TYPES: CardTransactionType[] = [
  'purchase',
  'payment',
  'payment_bonus',
  'cash_advance',
  'cash_advance_fee',
  'interest_current',
  'interest_late',
  'fee',
  'tax',
  'bonus',
  'refund',
  'adjustment',
  'other',
]

export const STATEMENT_STATUSES: StatementStatus[] = ['unpaid', 'partial', 'paid']

export function networkLabel(value: CardNetwork, t: Translator): string {
  return t(`creditCards.network_${value}`)
}

export function transactionTypeLabel(value: CardTransactionType, t: Translator): string {
  return t(`creditCards.type_${value}`)
}

export function statementStatusLabel(value: StatementStatus, t: Translator): string {
  return t(`creditCards.status_${value}`)
}

export function formatISODate(value: string | undefined, lang: string): string {
  if (!value) return '—'
  const date = new Date(`${value}T12:00:00`)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString(lang === 'en' ? 'en-US' : 'es-ES', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

export function isNegativeCredit(type: CardTransactionType): boolean {
  return type === 'payment' || type === 'payment_bonus' || type === 'refund' || type === 'bonus'
}
