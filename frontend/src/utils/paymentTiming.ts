export type PaymentTimingTone = 'success' | 'warning' | 'danger'
export type PaymentTimingStatus = 'on_time' | 'early' | 'late'

export interface PaymentTiming {
  /** Diferencia firmada: negativo = pagó antes, 0 = en fecha, positivo = pagó tarde. */
  days: number
  status: PaymentTimingStatus
  tone: PaymentTimingTone
}

export const PAYMENT_TIMING_WARNING_MAX_DAYS = 7

const DATE_ONLY_RE = /^(\d{4})-(\d{2})-(\d{2})$/

function parseDateOnly(value: string): number | null {
  const match = DATE_ONLY_RE.exec(value)
  if (!match) return null

  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  if (month < 1 || month > 12 || day < 1 || day > 31) return null

  const ms = Date.UTC(year, month - 1, day)
  const date = new Date(ms)
  if (
    date.getUTCFullYear() !== year ||
    date.getUTCMonth() !== month - 1 ||
    date.getUTCDate() !== day
  ) {
    return null
  }

  return Math.floor(ms / 86400000)
}

function formatDateInTimeZone(date: Date, timeZone: string): string | null {
  try {
    const parts = new Intl.DateTimeFormat('en-CA', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).formatToParts(date)
    const map: Record<string, string> = {}
    for (const part of parts) map[part.type] = part.value
    if (!map.year || !map.month || !map.day) return null
    return `${map.year}-${map.month}-${map.day}`
  } catch {
    return null
  }
}

/**
 * Convierte `paid_at` (RFC3339) al día de calendario `YYYY-MM-DD` en la zona
 * horaria indicada. Si `paidAt` ya es una fecha sin hora, la devuelve tal cual.
 */
export function paymentDateInTimeZone(
  paidAt: string | null | undefined,
  timeZone?: string,
): string | null {
  if (!paidAt) return null

  const value = paidAt.trim()
  if (!value) return null

  // Si viene como fecha sin hora, se valida y se conserva tal cual.
  if (DATE_ONLY_RE.test(value)) {
    return parseDateOnly(value) !== null ? value : null
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null

  const tz = timeZone || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  return formatDateInTimeZone(date, tz)
}

/**
 * Calcula la diferencia de días de calendario entre el pago y el vencimiento.
 * Devuelve `null` si falta alguna fecha o si no se pueden interpretar.
 */
export function getPaymentTiming(
  paidAt: string | null | undefined,
  dueDate: string | null | undefined,
  timeZone?: string,
): PaymentTiming | null {
  if (!dueDate) return null

  const paidDate = paymentDateInTimeZone(paidAt, timeZone)
  if (!paidDate) return null

  const paidDay = parseDateOnly(paidDate)
  const dueDay = parseDateOnly(dueDate.trim())
  if (paidDay === null || dueDay === null) return null

  const days = paidDay - dueDay
  const status: PaymentTimingStatus = days === 0 ? 'on_time' : days < 0 ? 'early' : 'late'
  const tone: PaymentTimingTone =
    days <= 0 ? 'success' : days <= PAYMENT_TIMING_WARNING_MAX_DAYS ? 'warning' : 'danger'

  return { days, status, tone }
}
