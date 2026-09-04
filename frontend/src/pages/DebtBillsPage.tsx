import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CardMenu, { type CardMenuOption } from '../components/CardMenu'
import LoadingSpinner from '../components/LoadingSpinner'
import DebtPayModal from '../components/DebtPayModal'
import type { Debt, DebtBill } from '../types'

export default function DebtBillsPage() {
  const navigate = useNavigate()
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

  if (loading) return <LoadingSpinner />

  if (!debt) return <div className="text-center py-8 text-text-secondary">Loading...</div>

  const currency = currencies.find((c) => c.id === debt.currency_id)
  const billMenuOptions = (bill: DebtBill): CardMenuOption[] => {
    const options: CardMenuOption[] = []
    if (bill.status === 'pending') {
      options.push({ label: t('deudas.pay'), icon: 'credit', onClick: () => setPayTarget(bill) })
    }
    return options
  }

  return (
    <div>
      <div className="flex items-center gap-3 mb-4 sm:mb-5">
        <button
          onClick={() => navigate('/deudas')}
          className="flex items-center gap-1 text-text-secondary hover:text-text transition-colors min-h-[44px]"
        >
          <Icon name="back" className="w-4 h-4" />
          {t('menu.deudas')}
        </button>
      </div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <div>
          <h2 className="text-lg sm:text-2xl font-bold">{debt.description}</h2>
          <p className="text-sm text-text-secondary">
            {debt.institution_name}
            {debt.identifier && ` · ${debt.identifier}`} · {t(`deudas.status_${debt.status}`)}
          </p>
        </div>
      </div>

      <div className="mb-4 flex items-center gap-2 flex-wrap text-sm text-text-secondary">
        <span className="bg-card rounded-ios-sm px-3 py-1 shadow-ios">
          {t('deudas.total')}: {formatMoney(debt.total, currency?.symbol || debt.currency_code)}
        </span>
        <span className="bg-card rounded-ios-sm px-3 py-1 shadow-ios">
          {debt.installments_total} {t('deudas.installment').toLowerCase()}(s)
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