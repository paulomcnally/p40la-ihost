import { useState, useEffect } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import type { Service, ServiceCycle } from '../types'

interface RenewServiceModalProps {
  service: Service
  onSaved: () => void
  onCancel: () => void
}

function fmtDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function addDays(dateStr: string, days: number): string {
  const d = new Date(dateStr + 'T12:00:00')
  d.setDate(d.getDate() + days)
  return fmtDate(d)
}

function addYears(dateStr: string, years: number): string {
  const d = new Date(dateStr + 'T12:00:00')
  d.setFullYear(d.getFullYear() + years)
  return fmtDate(d)
}

export default function RenewServiceModal({ service, onSaved, onCancel }: RenewServiceModalProps) {
  const { t } = useI18nStore()
  const [cycles, setCycles] = useState<ServiceCycle[]>([])
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [amount, setAmount] = useState(service.suggested_amount ?? 0)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.services.listCycles(service.id).then((list) => {
      setCycles(list || [])
    })
  }, [service.id])

  // Prefill del nuevo período: un día después del fin actual → +1 año.
  useEffect(() => {
    if (!endDate && service.end_date) {
      setStartDate(addDays(service.end_date, 1))
      setEndDate(addYears(service.end_date, 1))
    }
  }, [service.end_date, endDate])

  const currentCycle = cycles.length > 0 ? cycles[cycles.length - 1] : null
  const currentPeriod =
    currentCycle?.start_date && currentCycle?.end_date
      ? `${currentCycle.start_date} → ${currentCycle.end_date}`
      : service.start_date && service.end_date
        ? `${service.start_date} → ${service.end_date}`
        : null

  const handleSubmit = async () => {
    if (!startDate || !endDate) {
      setError(t('bills.renew_required'))
      return
    }
    setSubmitting(true)
    setError('')
    try {
      await api.services.renew(service.id, {
        start_date: startDate,
        end_date: endDate,
        suggested_amount: amount,
      })
      onSaved()
    } catch (e) {
      setError((e as Error).message || t('errors.generic'))
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onCancel}>
      <div className="bg-card rounded-ios shadow-ios w-full max-w-md max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
        <div className="p-4 border-b border-border">
          <h3 className="text-lg font-bold">{t('bills.renew_title')}</h3>
          <p className="text-sm text-text-secondary mt-1">{service.name}</p>
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          {currentPeriod && (
            <div className="mb-4 p-3 rounded-ios-sm bg-bg border border-border">
              <p className="text-xs font-semibold text-text-secondary uppercase mb-1">
                {currentCycle ? `${t('bills.renew_current_cycle')} #${currentCycle.sequence}` : t('bills.renew_current_period')}
              </p>
              <p className="text-sm font-medium">{currentPeriod}</p>
            </div>
          )}
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium mb-1">{t('bills.renew_start')}</label>
              <input
                type="date"
                value={startDate}
                onChange={e => setStartDate(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('bills.renew_end')}</label>
              <input
                type="date"
                value={endDate}
                onChange={e => setEndDate(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('bills.renew_amount')}</label>
              <input
                type="number"
                value={amount}
                onChange={e => setAmount(Number(e.target.value))}
                step="0.01"
                min="0"
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              />
            </div>
          </div>
          {error && <p className="mt-3 text-sm text-red-600 dark:text-red-400">{error}</p>}
        </div>
        <div className="p-4 border-t border-border flex justify-end gap-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
          >
            {t('app.cancel')}
          </button>
          <button
            onClick={handleSubmit}
            disabled={submitting}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
          >
            {submitting ? t('app.saving') : t('bills.renew_confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}