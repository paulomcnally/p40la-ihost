import { useI18nStore } from '../stores/i18nStore'
import { paymentTiming } from '../utils/paymentTiming'

type PaymentTimingMode = 'paid' | 'paying' | 'receipt'

interface PaymentTimingLabelProps {
  paidDate?: string | null
  dueDate?: string | null
  mode: PaymentTimingMode
  className?: string
}

// Muestra la diferencia en días de calendario entre el pago y el vencimiento.
// `mode` cambia el prefijo: cuota pagada, pago en curso o revisión de comprobante.
export default function PaymentTimingLabel({
  paidDate,
  dueDate,
  mode,
  className = '',
}: PaymentTimingLabelProps) {
  const { t } = useI18nStore()
  const timing = paymentTiming(paidDate, dueDate)
  if (!timing) return null

  const phrase = timing.kind === 'on_time'
    ? t('deudas.timing_on_time')
    : timing.kind === 'early'
      ? (timing.absDays === 1
          ? t('deudas.timing_early_one')
          : t('deudas.timing_early').replace('{n}', String(timing.absDays)))
      : (timing.absDays === 1
          ? t('deudas.timing_late_one')
          : t('deudas.timing_late').replace('{n}', String(timing.absDays)))

  const prefixKey = mode === 'paid'
    ? 'deudas.timing_paid_prefix'
    : mode === 'paying'
      ? 'deudas.timing_paying_prefix'
      : 'deudas.timing_receipt_prefix'

  const colorClass = timing.kind === 'late'
    ? 'text-yellow-800 dark:text-yellow-400'
    : 'text-green-700 dark:text-green-400'

  return (
    <span className={`${colorClass} ${className}`.trim()}>
      {`${t(prefixKey)} ${phrase}`}
    </span>
  )
}
