// Utilidades de período para el home global de Servicios (SPEC-120).

export interface YearMonth {
  year: number
  month: number
}

/** Clave YYYY-MM usada por la API. */
export function periodKey(year: number, month: number): string {
  return `${year}-${String(month).padStart(2, '0')}`
}

export function parsePeriodKey(value: string | null): YearMonth | null {
  if (!value || !/^\d{4}-\d{2}$/.test(value)) return null
  const year = Number(value.slice(0, 4))
  const month = Number(value.slice(5, 7))
  if (month < 1 || month > 12) return null
  return { year, month }
}

/** Desplaza un período n meses (n puede ser negativo). */
export function addMonths(period: YearMonth, n: number): YearMonth {
  const index = period.year * 12 + (period.month - 1) + n
  return {
    year: Math.floor(index / 12),
    month: (index % 12) + 1,
  }
}

export function currentPeriod(): YearMonth {
  const now = new Date()
  return { year: now.getFullYear(), month: now.getMonth() + 1 }
}

/** Genera la lista de períodos consecutivos que termina en `end`. */
export function periodRange(end: YearMonth, count: number): YearMonth[] {
  const start = addMonths(end, -(count - 1))
  return Array.from({ length: count }, (_, i) => addMonths(start, i))
}

/** Compara dos períodos: <0 si a es anterior a b. */
export function comparePeriods(a: YearMonth, b: YearMonth): number {
  return a.year * 12 + a.month - (b.year * 12 + b.month)
}

/** Fecha de hoy en formato YYYY-MM-DD (hora local). */
export function todayISO(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
