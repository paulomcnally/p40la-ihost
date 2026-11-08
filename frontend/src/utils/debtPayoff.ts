import type { Debt, DebtBill } from '../types'

export type PayoffState = 'paid' | 'missing-bills' | 'no-bills' | 'paused' | 'overdue' | 'active'

export interface RemainingTime {
  totalMonths: number
  years: number
  months: number
  kind: 'past' | 'due-today' | 'less-than-month' | 'months'
}

export interface TimelinePoint {
  key: 'start' | 'today' | 'next' | 'end'
  date: string
  position: number
  state: 'done' | 'current' | 'future' | 'overdue'
  label?: 'next' | 'overdue'
}

const DATE_ONLY_RE = /^(\d{4})-(\d{2})-(\d{2})/

// Normaliza una fecha de la API/DB a YYYY-MM-DD. Acepta date-only, timestamps
// con zona y RFC3339. Devuelve null si no es una fecha calendario válida.
export function toDateOnly(value: string | null | undefined): string | null {
  if (!value) return null
  const match = DATE_ONLY_RE.exec(value.trim())
  if (!match) return null
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  if (month < 1 || month > 12 || day < 1 || day > 31) return null
  const date = new Date(year, month - 1, day)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null
  }
  return `${match[1]}-${match[2]}-${match[3]}`
}

// Parsea una fecha date-only en horario local. Nunca usar new Date('YYYY-MM-DD'):
// interpreta UTC y puede mostrar el día anterior.
export function parseDateOnly(value: string | null | undefined): Date | null {
  const iso = toDateOnly(value)
  if (!iso) return null
  const [year, month, day] = iso.split('-').map(Number)
  return new Date(year, month - 1, day)
}

// Meses calendario completos entre now y end. Puede ser negativo si end < now.
export function monthsUntil(end: Date, now: Date): number {
  let months = (end.getFullYear() - now.getFullYear()) * 12 + (end.getMonth() - now.getMonth())
  if (end.getDate() < now.getDate()) months -= 1
  return months
}

// Tiempo restante desde `now` hasta `end`, en años y meses completos.
// La comparación de "pasado" usa la fecha exacta, no solo el mes.
export function remainingTime(end: Date | null, now: Date): RemainingTime {
  if (!end) return { totalMonths: 0, years: 0, months: 0, kind: 'months' }

  const endDay = new Date(end.getFullYear(), end.getMonth(), end.getDate())
  const nowDay = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const signedMonths = monthsUntil(endDay, nowDay)
  const absMonths = Math.abs(signedMonths)
  const years = Math.floor(absMonths / 12)
  const months = absMonths % 12

  if (endDay.getTime() < nowDay.getTime()) {
    return { totalMonths: signedMonths, years, months, kind: 'past' }
  }
  if (endDay.getTime() === nowDay.getTime()) {
    return { totalMonths: 0, years: 0, months: 0, kind: 'due-today' }
  }
  if (signedMonths < 1) {
    return { totalMonths: 0, years: 0, months: 0, kind: 'less-than-month' }
  }
  return { totalMonths: signedMonths, years, months, kind: 'months' }
}

export function paidBills(bills: DebtBill[]): DebtBill[] {
  return bills.filter((b) => b.status === 'paid')
}

// Próxima cuota pendiente: menor due_date, desempate por número de cuota.
export function nextPendingBill(bills: DebtBill[]): DebtBill | null {
  const pending = bills
    .filter((b) => b.status === 'pending')
    .map((bill) => ({ bill, iso: toDateOnly(bill.due_date) }))
    .filter((item): item is { bill: DebtBill; iso: string } => item.iso !== null)
    .sort((a, b) => a.iso.localeCompare(b.iso) || a.bill.installment_number - b.bill.installment_number)
  return pending[0]?.bill ?? null
}

// Cuotas pendientes cuya fecha de vencimiento es anterior a hoy. La cuota que
// vence hoy no cuenta como vencida.
export function overdueBills(bills: DebtBill[], todayISO: string): DebtBill[] {
  return bills.filter((b) => {
    if (b.status !== 'pending') return false
    const iso = toDateOnly(b.due_date)
    return iso !== null && iso < todayISO
  })
}

// Fecha del último pago realizado. `last_paid_at` es la fuente principal;
// si no existe, se toma el máximo paid_at de las cuotas pagadas.
export function lastPaymentDate(debt: Debt, bills: DebtBill[]): string | null {
  const fromDebt = toDateOnly(debt.last_paid_at)
  if (fromDebt) return fromDebt

  const paid = paidBills(bills)
    .map((b) => toDateOnly(b.paid_at))
    .filter((iso): iso is string => iso !== null)
    .sort()
  return paid.length > 0 ? paid[paid.length - 1] : null
}

// Fecha de la última cuota pendiente. Usa las cuotas reales y cae a
// last_due_date; nunca deriva start_date + installments_total.
export function pendingEndDate(debt: Debt, bills: DebtBill[]): string | null {
  const pending = bills
    .filter((b) => b.status === 'pending')
    .map((b) => toDateOnly(b.due_date))
    .filter((iso): iso is string => iso !== null)
    .sort()
  if (pending.length > 0) return pending[pending.length - 1]
  return toDateOnly(debt.last_due_date)
}

// Estado principal del payoff, con precedencia explícita.
export function payoffState(debt: Debt, bills: DebtBill[], todayISO: string): PayoffState {
  const pending = bills.filter((b) => b.status === 'pending')
  const pendingCount = debt.pending_count ?? pending.length
  const installmentsTotal = debt.installments_total ?? 0
  const hasBills = bills.length > 0
  const isFinalized = debt.status === 'finalizada'

  if (hasBills && pending.length === 0) return 'paid'
  if (!hasBills && isFinalized && pendingCount === 0) return 'paid'
  if (!hasBills) {
    if (pendingCount > 0 || installmentsTotal > 0) return 'missing-bills'
    return 'no-bills'
  }
  if (debt.status === 'inactiva' && pending.length > 0) return 'paused'
  if (overdueBills(bills, todayISO).length > 0) return 'overdue'
  return 'active'
}

// Porcentaje de cuotas pagadas. 100% solo cuando no quedan pendientes; con
// pendientes se limita a 99% para no mostrar un 100% engañoso.
export function progressPct(paid: number, total: number, hasPending: boolean): number {
  if (!hasPending) return 100
  if (total <= 0) return 0
  return Math.max(0, Math.min(99, Math.round((paid / total) * 100)))
}

// Construye los hitos de la línea de tiempo en orden cronológico. Incluye el
// inicio solo si ya empezó; fusiona fechas repetidas; omite la próxima cuota
// si cae después del fin; y clampea las posiciones a 0..100.
export function buildTimeline(input: {
  startDate: string | null
  endDate: string | null
  nextDueDate: string | null
  todayISO: string
}): TimelinePoint[] {
  const today = toDateOnly(input.todayISO)
  if (!today) return []

  const start = toDateOnly(input.startDate)
  const end = toDateOnly(input.endDate)
  const next = toDateOnly(input.nextDueDate)
  const points: TimelinePoint[] = []
  const seen = new Set<string>()

  const push = (key: TimelinePoint['key'], date: string | null, state: TimelinePoint['state'], label?: TimelinePoint['label']) => {
    if (!date || seen.has(date)) return
    seen.add(date)
    points.push({ key, date, position: 0, state, label })
  }

  push('today', today, 'current')
  if (start && start < today) push('start', start, 'done')
  if (next && next !== today && next !== end && (!end || next < end)) {
    push('next', next, next < today ? 'overdue' : 'future', next < today ? 'overdue' : 'next')
  }
  if (end && end !== today) {
    push('end', end, end < today ? 'overdue' : 'future')
  }

  points.sort((a, b) => a.date.localeCompare(b.date))

  const rangeStart = points[0]?.date
  const rangeEnd = points[points.length - 1]?.date
  if (!rangeStart || !rangeEnd) return points

  const startMs = Date.parse(`${rangeStart}T00:00:00`)
  const endMs = Date.parse(`${rangeEnd}T00:00:00`)
  const span = endMs - startMs

  points.forEach((point, index) => {
    if (span > 0) {
      const ms = Date.parse(`${point.date}T00:00:00`)
      point.position = Math.max(0, Math.min(100, ((ms - startMs) / span) * 100))
    } else {
      point.position = points.length === 1 ? 0 : (index / (points.length - 1)) * 100
    }
  })

  if (points.length > 1) {
    points[0].position = 0
    points[points.length - 1].position = 100
  }
  return points
}
