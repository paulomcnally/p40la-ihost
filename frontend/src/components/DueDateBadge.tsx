import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { daysUntilDue, getSystemTimeZone } from '../utils/date'

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
