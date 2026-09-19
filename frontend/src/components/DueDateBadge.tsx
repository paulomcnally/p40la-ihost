import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { getBrowserTimeZone } from '../constants/timezones'

let cachedTz: string | null = null
let tzPromise: Promise<string> | null = null

// Zona horaria configurada del sistema (SPEC-078) con fallback a la del
// navegador. Se obtiene una sola vez y se cachea (SPEC-081, ADR-004).
function getSystemTimeZone(): Promise<string> {
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

function todayInTimeZone(tz: string): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: tz,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date())
  const map: Record<string, string> = {}
  for (const p of parts) map[p.type] = p.value
  return `${map.year}-${map.month}-${map.day}`
}

// Días de calendario entre hoy (en la zona configurada) y dueDate (YYYY-MM-DD).
// Negativo si ya venció. Math.round absorbe el desplazamiento de DST.
export function daysUntilDue(dueDate: string, tz: string): number {
  const today = todayInTimeZone(tz)
  const ms = Date.parse(`${dueDate}T00:00:00`) - Date.parse(`${today}T00:00:00`)
  return Math.round(ms / 86400000)
}

interface DueDateBadgeProps {
  dueDate?: string | null
}

// Label semáforo de vencimiento (SPEC-081, REQ-008):
// verde si faltan más de 10 días, amarillo entre 10 y 1 día,
// rojo si vence hoy o ya venció. El caller no lo renderiza para facturas
// pagadas; aquí solo se omite si no hay due_date.
export default function DueDateBadge({ dueDate }: DueDateBadgeProps) {
  const { t } = useI18nStore()
  const [days, setDays] = useState<number | null>(null)

  useEffect(() => {
    if (!dueDate) {
      setDays(null)
      return
    }
    let cancelled = false
    getSystemTimeZone().then((tz) => {
      if (!cancelled) setDays(daysUntilDue(dueDate, tz))
    })
    return () => {
      cancelled = true
    }
  }, [dueDate])

  if (days === null) return null

  let text: string
  let cls: string
  if (days > 10) {
    text = t('bills.due_in_days').replace('{n}', String(days))
    cls = 'bg-success/20 text-green-800 dark:text-green-400'
  } else if (days >= 1) {
    text = t('bills.due_in_days').replace('{n}', String(days))
    cls = 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
  } else if (days === 0) {
    text = t('bills.due_today')
    cls = 'bg-danger/20 text-red-800 dark:text-red-400'
  } else {
    text = t('bills.due_overdue').replace('{n}', String(Math.abs(days)))
    cls = 'bg-danger/20 text-red-800 dark:text-red-400'
  }

  return <span className={`text-xs font-semibold px-2.5 py-1 rounded-full whitespace-nowrap ${cls}`}>{text}</span>
}