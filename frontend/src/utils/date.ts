import { api } from '../api'
import { getBrowserTimeZone } from '../constants/timezones'

let cachedTz: string | null = null
let tzPromise: Promise<string> | null = null

// Zona horaria configurada del sistema (SPEC-078) con fallback a la del
// navegador. Se obtiene una sola vez y se cachea (SPEC-081, ADR-004).
export function getSystemTimeZone(): Promise<string> {
  if (cachedTz) return Promise.resolve(cachedTz)
  if (!tzPromise) {
    tzPromise = api.systemSettings
      .get()
      .then((s) => (s?.timezone ? s.timezone : getBrowserTimeZone()))
      .catch(() => getBrowserTimeZone())
      .then((tz) => {
        cachedTz = tz
        return tz
      })
  }
  return tzPromise
}

export function todayInTimeZone(tz: string, now = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: tz,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(now)
  const map: Record<string, string> = {}
  for (const p of parts) map[p.type] = p.value
  return `${map.year}-${map.month}-${map.day}`
}

// Día local del navegador, usado como valor inicial mientras se resuelve la
// zona configurada. No usar toISOString(): interpreta UTC y puede adelantar o
// atrasar un día (SPEC-125).
export function todayISO(now = new Date()): string {
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

// Días de calendario entre hoy (en la zona configurada) y dueDate (YYYY-MM-DD).
// Negativo si ya venció. Math.round absorbe el desplazamiento de DST.
export function daysUntilDue(dueDate: string, tz: string): number {
  const today = todayInTimeZone(tz)
  const ms = Date.parse(`${dueDate}T00:00:00`) - Date.parse(`${today}T00:00:00`)
  return Math.round(ms / 86400000)
}
