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
  credit_limit: number
  credit_limit_usd: number
  statement_day: number
  payment_due_day: number
  min_payment_pct: number
  interest_current_rate: number
  interest_late_rate: number
  interest_current_rate_usd: number
  interest_late_rate_usd: number
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

// SPEC-139: conciliación de dos bases.
export interface Reconciliation {
  expected_cash: number
  printed_cash: number
  difference_cash: number
  expected_balance: number
  printed_balance: number
  difference_balance: number
  expected_cash_usd: number
  printed_cash_usd: number
  difference_cash_usd: number
  expected_balance_usd: number
  printed_balance_usd: number
  difference_balance_usd: number
  cash_applicable: boolean
  balance_applicable: boolean
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
  interest_bonifiable: number
  interest_bonifiable_usd: number
  mora_amount: number
  mora_amount_usd: number
  arrears_count: number
  over_limit_amount: number
  over_limit_amount_usd: number
  points_previous: number
  points_earned: number
  points_used: number
  points_available: number
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
  interest_bonifiable: number
  interest_bonifiable_usd: number
  mora_amount: number
  mora_amount_usd: number
  arrears_count: number
  over_limit_amount: number
  over_limit_amount_usd: number
  points_previous: number
  points_earned: number
  points_used: number
  points_available: number
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
