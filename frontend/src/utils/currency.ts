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