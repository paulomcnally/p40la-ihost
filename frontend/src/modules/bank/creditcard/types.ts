// SPEC-137: tipos del módulo bank.creditcard.

export type CardNetwork = 'visa' | 'mastercard' | 'amex' | 'diners' | 'other'

export type CardTransactionType =
  | 'purchase'
  | 'payment'
  | 'payment_bonus'
  | 'cash_advance'
  | 'cash_advance_fee'
  | 'interest_current'
  | 'interest_late'
  | 'fee'
  | 'tax'
  | 'bonus'
  | 'refund'
  | 'adjustment'
  | 'other'

export type StatementStatus = 'unpaid' | 'partial' | 'paid'
export type StatementSource = 'manual' | 'import'
export type TransactionStatus = 'posted' | 'pending'

// SPEC-138: la moneda en la que se publica el límite y la categoría del producto.
export type LimitCurrency = 'NIO' | 'USD'
export type CardCategory = '' | 'classic' | 'gold' | 'platinum' | 'black' | 'business' | 'other'

export interface CreditCard {
  id: number
  bank_id: number
  bank_name?: string
  alias: string
  network: CardNetwork
  last4: string
  holder: string
  currency_id: number
  currency_code?: string
  secondary_currency_id?: number
  secondary_currency_code?: string
  category: CardCategory
  limit_currency: LimitCurrency
  exchange_rate: number
  credit_limit: number
  credit_limit_usd: number
  statement_day: number
  payment_due_day: number
  bonus_due_day: number
  min_payment_pct: number
  interest_current_rate: number
  interest_late_rate: number
  interest_current_rate_usd: number
  interest_late_rate_usd: number
  late_total_rate: number
  late_total_rate_usd: number
  color: string
  icon: string
  active: boolean
  notes: string
  statement_count: number
  balance_local: number
  balance_usd: number
  available_local: number
  available_usd: number
  last_statement_id?: number
  created_at: string
  updated_at: string
}

export interface StatementSummary {
  type: CardTransactionType
  amount_local: number
  amount_usd: number
  count: number
}

export interface Reconciliation {
  expected_local: number
  actual_local: number
  difference_local: number
  expected_usd: number
  actual_usd: number
  difference_usd: number
  balanced: boolean
}

export interface CardStatement {
  id: number
  card_id: number
  period_start?: string
  period_end: string
  min_payment_due_date: string
  full_payment_due_date: string
  previous_balance: number
  previous_balance_usd: number
  interest_current: number
  interest_current_usd: number
  min_payment: number
  min_payment_usd: number
  total_payment: number
  total_payment_usd: number
  new_balance: number
  new_balance_usd: number
  status: StatementStatus
  source: StatementSource
  source_module_id?: string
  notes: string
  transaction_count: number
  summary?: StatementSummary[]
  reconciliation?: Reconciliation
  created_at: string
  updated_at: string
}

export interface CardTransaction {
  id: number
  card_id: number
  statement_id?: number
  posted_at: string
  concept: string
  category_id?: number
  category_name?: string
  type: CardTransactionType
  amount_local: number
  amount_usd: number
  currency_original: 'NIO' | 'USD'
  status: TransactionStatus
  external_ref?: string
  note: string
  created_at: string
  updated_at: string
}

export interface ParsedTransaction {
  posted_at: string
  concept: string
  type: CardTransactionType
  amount_local: number
  amount_usd: number
  currency_original: 'NIO' | 'USD'
  external_ref: string
}

export interface ParseWarning {
  code: string
  message: string
}

export interface ParsedStatement {
  module_id: string
  module_version: string
  last4: string
  period_end: string
  min_payment_due_date: string
  full_payment_due_date: string
  previous_balance: number
  previous_balance_usd: number
  interest_current: number
  interest_current_usd: number
  min_payment: number
  min_payment_usd: number
  total_payment: number
  total_payment_usd: number
  new_balance: number
  new_balance_usd: number
  transactions: ParsedTransaction[]
  warnings?: ParseWarning[]
  preview_hash: string
}
