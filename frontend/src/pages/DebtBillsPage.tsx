import { useEffect, useState, useCallback, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CardMenu, { type CardMenuOption } from '../components/CardMenu'
import LoadingSpinner from '../components/LoadingSpinner'
import DebtPayModal from '../components/DebtPayModal'
import DonutChart from '../components/DonutChart'
import type { Debt, DebtBill } from '../types'

export default function DebtBillsPage() {
  const { id } = useParams()
  const { t } = useI18nStore()
  const { currencies } = useAppStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const [debt, setDebt] = useState<Debt | null>(null)
  const [bills, setBills] = useState<DebtBill[]>([])
  const [payTarget, setPayTarget] = useState<DebtBill | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    const [debtData, billList] = await Promise.all([
      api.debts.get(Number(id)),
      api.debts.listBills(Number(id)),
    ])
    setDebt(debtData)
    setBills(billList || [])
    setLoading(false)
  }, [id])

  useEffect(() => {
    load()
  }, [load])

  usePageTitle(debt?.description ?? null)

  const progress = useMemo(() => {
    const paidBills = bills.filter((b) => b.status === 'paid')
    const pendingBills = bills.filter((b) => b.status === 'pending')
    const totalCount = debt?.installments_total ?? 0
    const paidCount = paidBills.length
    const pct = totalCount > 0 ? Math.round((paidCount / totalCount) * 100) : 0
    const paidAmount = paidBills.reduce((s, b) => s + b.amount, 0)
    const pendingAmount = pendingBills.reduce((s, b) => s + b.amount, 0)
    return {
      paidCount,
      pendingCount: pendingBills.length,
      totalCount,
      pct,
      paidAmount,
      pendingAmount,
      donutSegments: [
        { label: t('deudas.progress_paid'), value: paidAmount, color: '#30d158' },
        { label: t('deudas.progress_pending'), value: pendingAmount, color: '#ff9f0a' },
      ],
    }
  }, [bills, debt, t])

  if (loading) return <LoadingSpinner />

  if (!debt) return <div className="text-center py-8 text-text-secondary">Loading...</div>

  const currency = currencies.find((c) => c.id === debt.currency_id)
  const symbol = currency?.symbol || debt.currency_code
  const installmentLabel = (debt.installments_total === 1 ? t('deudas.installment') : t('deudas.installments')).toLowerCase()

  const pctColorClass =
    progress.pct === 100
      ? 'text-emerald-600 dark:text-emerald-400'
      : progress.pct === 0
      ? 'text-text-secondary'
      : 'text-amber-600 dark:text-amber-400'

  const billMenuOptions = (bill: DebtBill): CardMenuOption[] => {
    const options: CardMenuOption[] = []
    if (bill.status === 'pending') {
      options.push({ label: t('deudas.pay'), icon: 'credit', onClick: () => setPayTarget(bill) })
    }
    return options
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <div>
          <h2 className="text-lg sm:text-2xl font-bold">{debt.description}</h2>
          <p className="text-sm text-text-secondary">
            {debt.institution_name}
            {debt.identifier && ` · ${debt.identifier}`} · {t(`deudas.status_${debt.status}`)}
          </p>
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5 mb-4">
        <div className="flex items-center gap-4">
          <DonutChart
            segments={progress.donutSegments}
            size={104}
            thickness={13}
            centerValue={`${progress.pct}%`}
          />
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium uppercase tracking-wide text-text-secondary mb-1">
              {t('deudas.progress_label')}
            </p>
            <p className={`text-2xl sm:text-3xl font-bold ${pctColorClass}`}>
              {progress.paidCount} {t('deudas.of')} {progress.totalCount} {installmentLabel}
            </p>
            <div className="h-2 rounded-full bg-border overflow-hidden mt-2">
              <div
                className="h-full rounded-full bg-success transition-all"
                style={{ width: `${progress.pct}%` }}
              />
            </div>
            <div className="flex items-center justify-between gap-3 mt-2 text-xs text-text-secondary">
              <span className="flex items-center gap-1.5 min-w-0">
                <span className="w-2.5 h-2.5 rounded-full bg-success shrink-0" />
                <span className="truncate">
                  {t('deudas.progress_paid')}: {formatMoney(progress.paidAmount, symbol)} ({progress.paidCount})
                </span>
              </span>
              <span className="flex items-center gap-1.5 min-w-0">
                <span className="w-2.5 h-2.5 rounded-full bg-warning shrink-0" />
                <span className="truncate">
                  {t('deudas.progress_pending')}: {formatMoney(progress.pendingAmount, symbol)} ({progress.pendingCount})
                </span>
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="mb-4 flex items-center gap-2 flex-wrap text-sm text-text-secondary">
        <span className="bg-card rounded-ios-sm px-3 py-1 shadow-ios">
          {t('deudas.total')}: {formatMoney(debt.total, currency?.symbol || debt.currency_code)}
        </span>
        <span className="bg-card rounded-ios-sm px-3 py-1 shadow-ios">
          {progress.paidCount} {t('deudas.of')} {debt.installments_total} {installmentLabel}
        </span>
        <span className="bg-card rounded-ios-sm px-3 py-1 shadow-ios">
          {t('deudas.payment_day')}: {debt.payment_day}
        </span>
      </div>

      {bills.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="bill" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.bills_empty')}</h3>
        </div>
      ) : (
        <div className="space-y-2">
          {bills.map((bill) => (
            <div
              key={bill.id}
              className="bg-card rounded-ios shadow-ios p-4 relative flex items-center justify-between gap-3"
            >
              <CardMenu options={billMenuOptions(bill)} />
              <div className="min-w-0">
                <p className="text-sm font-medium">
                  {t('deudas.installment')} #{bill.installment_number}
                </p>
                <p className="text-sm text-text-secondary">{bill.due_date}</p>
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <p className="text-base font-semibold">
                  {formatMoney(bill.amount, currency?.symbol || bill.currency_code)}
                </p>
                <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                  bill.status === 'paid' ? 'bg-success/20 text-green-800 dark:text-green-400' : 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
                }`}>
                  {t(`bills.status_${bill.status}`)}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {payTarget && (
        <DebtPayModal
          bill={payTarget}
          onClose={() => setPayTarget(null)}
          onSuccess={load}
        />
      )}
    </div>
  )
}