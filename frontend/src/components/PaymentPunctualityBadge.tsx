import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { getSystemTimeZone } from '../utils/date'
import { getPaymentTiming, type PaymentPunctuality } from '../utils/paymentTiming'

interface PaymentPunctualityBadgeProps {
  paidAt?: string | null
  dueDate?: string | null
  timeZone?: string
}

const TONE_CLASSES: Record<PaymentPunctuality['tone'], string> = {
  success: 'bg-success/20 text-green-800 dark:text-green-400',
  warning: 'bg-warning/20 text-yellow-800 dark:text-yellow-400',
  danger: 'bg-danger/20 text-red-800 dark:text-red-400',
}

// Semáforo de puntualidad del pago (SPEC-128):
// verde en fecha o antes, amarillo 1-7 días tarde, rojo 8+ días tarde.
export default function PaymentPunctualityBadge({
  paidAt,
  dueDate,
  timeZone,
}: PaymentPunctualityBadgeProps) {
  const { t } = useI18nStore()
  const [timing, setTiming] = useState<PaymentPunctuality | null>(null)

  useEffect(() => {
    if (!paidAt || !dueDate) {
      setTiming(null)
      return
    }

    if (timeZone) {
      setTiming(getPaymentTiming(paidAt, dueDate, timeZone))
      return
    }

    let cancelled = false
    getSystemTimeZone()
      .then((tz) => {
        if (!cancelled) setTiming(getPaymentTiming(paidAt, dueDate, tz))
      })
      .catch(() => {
        if (!cancelled) setTiming(null)
      })

    return () => {
      cancelled = true
    }
  }, [paidAt, dueDate, timeZone])

  if (!timing) return null

  const days = Math.abs(timing.days)
  let text: string
  if (timing.status === 'on_time') {
    text = t('deudas.payment_timing_on_time')
  } else if (timing.status === 'early') {
    const key =
      days === 1
        ? 'deudas.payment_timing_days_early_one'
        : 'deudas.payment_timing_days_early_other'
    text = t(key).replace('{n}', String(days))
  } else {
    const key =
      days === 1
        ? 'deudas.payment_timing_days_late_one'
        : 'deudas.payment_timing_days_late_other'
    text = t(key).replace('{n}', String(days))
  }

  return (
    <span
      className={`text-xs font-semibold px-2.5 py-1 rounded-full whitespace-nowrap ${TONE_CLASSES[timing.tone]}`}
      aria-label={text}
    >
      {text}
    </span>
  )
}
