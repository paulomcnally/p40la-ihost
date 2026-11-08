export type PaymentTimingKind = 'early' | 'on_time' | 'late'

export interface PaymentTiming {
  /** Días firmados: negativo si se pagó antes, 0 el día de vencimiento, positivo si fue después. */
  days: number
  absDays: number
  kind: PaymentTimingKind
}

export type PaymentPunctualityTone = 'success' | 'warning' | 'danger'
export type PaymentPunctualityStatus = 'on_time' | 'early' | 'late'

export interface PaymentPunctuality {
  /** Diferencia firmada: negativo = pagó antes, 0 = en fecha, positivo = pagó tarde. */
  days: number
  status: PaymentPunctualityStatus
  tone: PaymentPunctualityTone
}

export const PAYMENT_TIMING_WARNING_MAX_DAYS = 7

const DATE_ONLY_RE = /^(\d{4})-(\d{2})-(\d{2})$/

// Parsea la parte YYYY-MM-DD de una fecha ISO o de un timestamp con zona.
// Usa UTC para comparar días de calendario sin verse afectado por DST.
function parseDateOnly(value?: string | null): number | null {
  if (!value) return null
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value)
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
  return ms
}

function parseDateOnlyStrict(value: string): number | null {
  if (!DATE_ONLY_RE.test(value)) return null
  return parseDateOnly(value)
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
    return parseDateOnlyStrict(value) !== null ? value : null
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null

  const tz = timeZone || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  return formatDateInTimeZone(date, tz)
}

/**
 * Calcula la diferencia de días de calendario entre el pago y el vencimiento,
 * con el tono del semáforo (verde/amarillo/rojo). Devuelve `null` si falta
 * alguna fecha o si no se pueden interpretar.
 */
export function getPaymentTiming(
  paidAt: string | null | undefined,
  dueDate: string | null | undefined,
  timeZone?: string,
): PaymentPunctuality | null {
  if (!dueDate) return null

  const paidDate = paymentDateInTimeZone(paidAt, timeZone)
  if (!paidDate) return null

  const paidDay = parseDateOnlyStrict(paidDate)
  const dueDay = parseDateOnlyStrict(dueDate.trim())
  if (paidDay === null || dueDay === null) return null

  const days = Math.round((paidDay - dueDay) / 86400000)
  const status: PaymentPunctualityStatus =
    days === 0 ? 'on_time' : days < 0 ? 'early' : 'late'
  const tone: PaymentPunctualityTone =
    days <= 0 ? 'success' : days <= PAYMENT_TIMING_WARNING_MAX_DAYS ? 'warning' : 'danger'

  return { days, status, tone }
}

// Diferencia en días de calendario entre la fecha de pago y el vencimiento.
// Devuelve null si alguna fecha falta o es inválida.
export function paymentTiming(
  paidDate?: string | null,
  dueDate?: string | null,
): PaymentTiming | null {
  const paid = parseDateOnly(paidDate)
  const due = parseDateOnly(dueDate)
  if (paid === null || due === null) return null

  const days = Math.round((paid - due) / 86400000)
  return {
    days,
    absDays: Math.abs(days),
    kind: days < 0 ? 'early' : days > 0 ? 'late' : 'on_time',
  }
}
