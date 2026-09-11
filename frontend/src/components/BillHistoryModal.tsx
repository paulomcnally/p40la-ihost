import { useEffect, useState } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import LoadingSpinner from './LoadingSpinner'
import type { Bill, BillHistoryEntry } from '../types'

interface BillHistoryModalProps {
  bill: Bill
  onClose: () => void
}

const FIELD_LABELS: Record<string, string> = {
  amount: 'bills.field_amount',
  invoice_number: 'bills.field_invoice_number',
  status: 'bills.field_status',
  drive_url: 'bills.field_drive_url',
  year: 'bills.field_year',
  month: 'bills.field_month',
  payment_reference: 'bills.field_payment_reference',
  paid_at: 'bills.field_paid_at',
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  return String(value)
}

export default function BillHistoryModal({ bill, onClose }: BillHistoryModalProps) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const [history, setHistory] = useState<BillHistoryEntry[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    api.bills.history(bill.id)
      .then(data => { if (!cancelled) setHistory(data || []) })
      .catch(() => { if (!cancelled) setError(t('errors.generic')) })
    return () => { cancelled = true }
  }, [bill.id, t])

  const actionLabel = (action: BillHistoryEntry['action']): string =>
    t(`bills.history_action_${action}`)

  const sourceLabel = (source: BillHistoryEntry['source']): string =>
    t(`bills.history_source_${source}`)

  const formatFieldValue = (field: string, value: unknown): string => {
    if (field === 'amount') {
      const num = Number(value)
      if (!Number.isNaN(num)) return formatMoney(num)
    }
    return formatValue(value)
  }

  const formatDate = (iso: string): string => {
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return iso
    return d.toLocaleString()
  }

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-lg max-h-[85vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5 pb-3 flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="w-11 h-11 rounded-full bg-primary/10 text-primary flex items-center justify-center shrink-0">
              <Icon name="clock" className="w-6 h-6" />
            </div>
            <div>
              <h3 className="text-lg font-semibold">{t('bills.history')}</h3>
              <p className="text-sm text-text-secondary">
                #{bill.invoice_number || bill.id} — {bill.month === 0 ? t('bills.annual') : t(`months.${bill.month}`)} {bill.year}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
            aria-label={t('app.cancel')}
          >
            <Icon name="cancel" className="w-5 h-5 text-text-secondary" />
          </button>
        </div>

        <div className="px-5 pb-5 overflow-y-auto flex-1">
          {error && (
            <p className="text-sm text-danger bg-danger/5 border border-danger/20 rounded-ios-sm p-3">{error}</p>
          )}
          {!error && !history && <LoadingSpinner />}
          {!error && history && history.length === 0 && (
            <p className="text-sm text-text-secondary text-center py-8">{t('bills.history_empty')}</p>
          )}
          {!error && history && history.length > 0 && (
            <ol className="relative space-y-5">
              {history.map((entry, i) => (
                <li key={entry.id} className="relative pl-6">
                  {i < history.length - 1 && (
                    <span className="absolute left-[7px] top-5 bottom-[-22px] w-px bg-border" />
                  )}
                  <span className={`absolute left-0 top-1.5 w-[15px] h-[15px] rounded-full border-2 ${
                    entry.action === 'paid' ? 'bg-success border-success'
                    : entry.action === 'created' ? 'bg-primary border-primary'
                    : 'bg-card border-border'
                  }`} />
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm font-semibold">{actionLabel(entry.action)}</span>
                    <span className="text-xs text-text-secondary">{formatDate(entry.created_at)}</span>
                  </div>
                  <div className="mt-1">
                    <span className={`inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full ${
                      entry.source === 'webhook'
                        ? 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
                        : 'bg-primary/10 text-primary'
                    }`}>
                      <Icon name="link" className="w-3 h-3" />
                      {sourceLabel(entry.source)}
                    </span>
                  </div>
                  {entry.changes && entry.changes.length > 0 && (
                    <div className="mt-2 space-y-1.5">
                      {entry.changes.map((change, j) => (
                        <div key={j} className="bg-bg rounded-ios-sm px-3 py-2 text-sm">
                          <span className="font-medium text-text">{t(FIELD_LABELS[change.field] || change.field)}</span>
                          <div className="flex items-center gap-2 text-xs text-text-secondary mt-0.5">
                            <span className="line-through">{formatFieldValue(change.field, change.old)}</span>
                            <Icon name="chevron" className="w-3 h-3 rotate-90" />
                            <span className="text-primary font-medium">{formatFieldValue(change.field, change.new)}</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </div>
  )
}