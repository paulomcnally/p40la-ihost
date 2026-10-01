import type { Debt } from '../types'

export interface MonthlyBucket {
  year: number
  month: number // 1-12
  label: string // "Ene 2026"
  count: number
  total: number
  debts: Debt[]
}

export interface LeftTime {
  years: number
  months: number
  thisMonth: boolean
}

// Deriva el mes/año de finalización de una deuda: start_date + installments_total meses,
// replicando dueDateForInstallment del backend (SPEC-054). Devuelve null si no se puede calcular.
export function debtEndMonth(debt: Debt): { year: number; month: number } | null {
  if (!debt.start_date || debt.installments_total <= 0) return null
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(debt.start_date)
  if (!match) return null
  const startYear = Number(match[1])
  const startMonth = Number(match[2])
  if (startMonth < 1 || startMonth > 12) return null

  const totalMonths = startYear * 12 + (startMonth - 1) + debt.installments_total
  const year = Math.floor(totalMonths / 12)
  const month = (totalMonths % 12) + 1
  return { year, month }
}

// Agrupa deudas por mes/año de finalización, ordenadas de la fecha más cercana a la más lejana.
// Solo incluye meses con al menos una deuda. El label se arma con monthLabel (ej. mes corto + año).
export function aggregateDebtsByEndMonth(
  debts: Debt[],
  endDateOf: (d: Debt) => { year: number; month: number } | null,
  monthLabel: (month: number) => string
): MonthlyBucket[] {
  const map = new Map<string, MonthlyBucket>()

  for (const debt of debts) {
    const end = endDateOf(debt)
    if (!end) continue
    const key = `${end.year}-${end.month}`
    let bucket = map.get(key)
    if (!bucket) {
      bucket = {
        year: end.year,
        month: end.month,
        label: `${monthLabel(end.month)} ${end.year}`,
        count: 0,
        total: 0,
        debts: [],
      }
      map.set(key, bucket)
    }
    bucket.count += 1
    bucket.total += debt.total || 0
    bucket.debts.push(debt)
  }

  return [...map.values()].sort((a, b) => b.year - a.year || b.month - a.month)
}

// Deriva la fecha exacta de finalización de una deuda: start_date + installments_total meses,
// clampeando al último día del mes (replica dueDateForInstallment del backend, SPEC-054).
// Devuelve null si no se puede calcular.
export function debtEndDate(debt: Debt): { year: number; month: number; day: number } | null {
  if (!debt.start_date || debt.installments_total <= 0) return null
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(debt.start_date)
  if (!match) return null
  const startYear = Number(match[1])
  const startMonth = Number(match[2])
  if (startMonth < 1 || startMonth > 12) return null

  const totalMonths = startYear * 12 + (startMonth - 1) + debt.installments_total
  const year = Math.floor(totalMonths / 12)
  const month = (totalMonths % 12) + 1
  const day = new Date(year, month, 0).getDate()
  return { year, month, day }
}

// Calcula el tiempo restante desde `now` hasta `end`, en años y meses.
// `thisMonth` es true cuando quedan menos de un mes (incluye fechas pasadas).
export function leftTime(end: Date, now: Date): LeftTime {
  const months = (end.getFullYear() - now.getFullYear()) * 12 + (end.getMonth() - now.getMonth())
  if (months < 1) return { years: 0, months: 0, thisMonth: true }
  const years = Math.floor(months / 12)
  return { years, months: months % 12, thisMonth: false }
}