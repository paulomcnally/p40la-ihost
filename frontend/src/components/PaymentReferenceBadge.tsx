import { useI18nStore } from '../stores/i18nStore'

interface PaymentReferenceBadgeProps {
  reference?: string | null
}

// Referencia del pago (SPEC-128): chip neutro con el valor o badge rojo
// cuando la cuota pagada no tiene referencia.
export default function PaymentReferenceBadge({ reference }: PaymentReferenceBadgeProps) {
  const { t } = useI18nStore()
  const value = (reference ?? '').trim()

  if (!value) {
    return (
      <span className="text-xs font-semibold px-2.5 py-1 rounded-full whitespace-nowrap bg-danger/20 text-red-800 dark:text-red-400">
        {t('deudas.payment_reference_missing')}
      </span>
    )
  }

  return (
    <span className="inline-flex min-w-0 max-w-full items-center gap-1 text-xs font-medium px-2.5 py-1 rounded-full bg-border/60 text-text-secondary">
      <span className="shrink-0">{t('deudas.payment_reference_short')}:</span>
      <span className="truncate max-w-[14rem]" title={value}>
        {value}
      </span>
    </span>
  )
}
