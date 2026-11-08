export type PaymentTimingKind = 'early' | 'on_time' | 'late'

export interface PaymentTiming {
  /** Días firmados: negativo si se pagó antes, 0 el día de vencimiento, positivo si fue después. */
  days: number
  absDays: number
  kind: PaymentTimingKind
}

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
