import type { Currency } from '../types'

export interface ResolvedCurrency {
  id?: number
  code?: string
  symbol?: string
}

export interface CurrencyLike {
  id: number
  code: string
  symbol?: string | null
}

export const UNKNOWN_CURRENCY_SYMBOL = '¤'

export interface CurrencyFormatConfig {
  thousandsSeparator?: string
  decimalSeparator?: string
  decimalDigits?: number
}

export const DEFAULT_CURRENCY_FORMAT = {
  thousandsSeparator: ',',
  decimalSeparator: '.',
  decimalDigits: 2,
} as const

export const VALID_SEPARATORS = [',', '.', ' ', "'", ''] as const

function groupThousands(num: string, sep: string): string {
  if (!sep || num.length <= 3) return num
  let sign = ''
  if (num.startsWith('-')) {
    sign = '-'
    num = num.slice(1)
  }
  const parts: string[] = []
  for (let i = 0; i < num.length; i++) {
    if (i > 0 && (num.length - i) % 3 === 0) parts.push(sep)
    parts.push(num[i])
  }
  return sign + parts.join('')
}

export function formatCurrency(amount: number, cfg: CurrencyFormatConfig = {}): string {
  const thousandsSeparator = cfg.thousandsSeparator ?? DEFAULT_CURRENCY_FORMAT.thousandsSeparator
  const decimalSeparator = cfg.decimalSeparator ?? DEFAULT_CURRENCY_FORMAT.decimalSeparator
  const decimalDigits = cfg.decimalDigits ?? DEFAULT_CURRENCY_FORMAT.decimalDigits
  const numStr = amount.toFixed(decimalDigits)
  const [intPart, fracPart] = numStr.split('.')
  const grouped = groupThousands(intPart, thousandsSeparator)
  if (decimalDigits === 0) return grouped
  return grouped + decimalSeparator + fracPart
}

export function formatCurrencyWithSymbol(
  amount: number,
  symbol: string | undefined,
  cfg?: CurrencyFormatConfig,
): string {
  const formatted = formatCurrency(amount, cfg)
  return symbol ? `${symbol}${formatted}` : formatted
}

/**
 * Resuelve la moneda de un registro a partir del catálogo y de su id/código.
 * El id tiene prioridad; si no hay match se usa el código ISO como fallback.
 * Nunca inventa una moneda cuando no hay datos (evita asumir NIO, SPEC-113).
 */
export function resolveCurrency(
  currencies: CurrencyLike[] | Currency[] | null | undefined,
  opts: { id?: number | null; code?: string | null } = {},
): ResolvedCurrency {
  const list = currencies || []
  const id = opts.id && opts.id > 0 ? opts.id : undefined
  const code = opts.code?.trim() || undefined
  const match =
    (id ? list.find((c) => c.id === id) : undefined) ||
    (code ? list.find((c) => c.code.toLowerCase() === code.toLowerCase()) : undefined)

  return {
    id: match?.id ?? id,
    code: match?.code || code,
    symbol: match?.symbol?.trim() || undefined,
  }
}