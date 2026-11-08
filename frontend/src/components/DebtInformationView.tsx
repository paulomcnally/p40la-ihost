import { useMemo } from 'react'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
import { Icon } from './Icons'
import {
  buildDebtInformationContext,
  debtInformationValueIsEmpty,
  resolveDebtInformationText,
} from '../utils/debtInformation'
import type {
  Debt,
  DebtBill,
  DebtInformationField,
  DebtInformationTemplate,
  DebtInformationValue,
} from '../types'

interface DebtInformationViewProps {
  template: DebtInformationTemplate | null
  values: Record<string, DebtInformationValue>
  debt: Debt
  bill: DebtBill
  symbol?: string
  paidAt?: string
}

export default function DebtInformationView({
  template,
  values,
  debt,
  bill,
  symbol,
  paidAt,
}: DebtInformationViewProps) {
  const { t, lang } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const { showToast } = useToast()

  const context = useMemo(
    () =>
      buildDebtInformationContext({
        debt,
        bill,
        symbol,
        lang,
        paidAt,
        statusLabel: t(bill.status === 'paid' ? 'bills.status_paid' : 'bills.status_pending'),
        formatMoney,
      }),
    [debt, bill, symbol, lang, paidAt, formatMoney, t],
  )

  const fields = template?.fields ?? []
  if (fields.length === 0) {
    return (
      <p className="text-sm text-text-secondary text-center py-6">
        {t('deudas.payment_info_empty')}
      </p>
    )
  }

  const displayValue = (field: DebtInformationField): string => {
    if (field.type === 'computed') {
      return resolveDebtInformationText(field.value_template || '', context)
    }
    const value = values[field.key]
    if (debtInformationValueIsEmpty(value)) {
      return ''
    }
    if (field.type === 'checkbox') {
      return value === true ? t('app.yes') : t('app.no')
    }
    return String(value)
  }

  const copyValue = async (value: string) => {
    if (!value) return
    try {
      await navigator.clipboard.writeText(value)
      showToast(t('app.copied'), 'success')
    } catch {
      showToast(t('errors.generic'), 'error')
    }
  }

  return (
    <div className="space-y-2.5">
      {fields.map((field) => {
        const value = displayValue(field)
        const shown = value.trim() === '' ? t('deudas.payment_info_value_missing') : value
        return (
          <div key={field.key} className="rounded-ios-sm bg-bg/70 border border-border/60 p-3">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-xs font-medium text-text-secondary">{field.label}</p>
                <p
                  className={`text-sm font-medium text-text mt-0.5 break-words ${
                    field.type === 'textarea' ? 'whitespace-pre-wrap' : ''
                  }`}
                >
                  {shown}
                </p>
                {field.help_text && (
                  <p className="text-[11px] text-text-secondary mt-1 break-words">{field.help_text}</p>
                )}
              </div>
              {value.trim() !== '' && (
                <button
                  type="button"
                  onClick={() => copyValue(value)}
                  className="shrink-0 w-9 h-9 rounded-full flex items-center justify-center text-text-secondary hover:bg-card transition-colors"
                  title={t('app.copy')}
                  aria-label={`${t('app.copy')} ${field.label}`}
                >
                  <Icon name="copy" className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
