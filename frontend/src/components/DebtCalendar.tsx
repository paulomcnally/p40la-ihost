import { useEffect, useMemo, useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import DebtPayModal from './DebtPayModal'
import type { DebtBill } from '../types'

const WEEKDAYS_ES = ['Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb', 'Dom']
const WEEKDAYS_EN = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

function dateKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export default function DebtCalendar() {
  const { t, lang } = useI18nStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [bills, setBills] = useState<DebtBill[]>([])
  const [selectedDay, setSelectedDay] = useState<string | null>(dateKey(now))
  const [payTarget, setPayTarget] = useState<DebtBill | null>(null)

  const load = async (y: number, m: number) => {
    const list = await api.debts.billsByMonth(y, m)
    setBills(list || [])
  }

  useEffect(() => {
    load(year, month)
    setSelectedDay((prev) => {
      if (!prev) return prev
      const prefix = `${year}-${String(month).padStart(2, '0')}`
      return prev.startsWith(prefix) ? prev : null
    })
  }, [year, month])

  const goPrev = () => {
    if (month === 1) {
      setYear((y) => y - 1)
      setMonth(12)
    } else {
      setMonth((m) => m - 1)
    }
  }

  const goNext = () => {
    if (month === 12) {
      setYear((y) => y + 1)
      setMonth(1)
    } else {
      setMonth((m) => m + 1)
    }
  }

  const cells = useMemo(() => {
    const offset = (new Date(year, month - 1, 1).getDay() + 6) % 7
    const dim = new Date(year, month, 0).getDate()
    const out: (number | null)[] = Array(offset).fill(null)
    for (let d = 1; d <= dim; d++) out.push(d)
    return out
  }, [year, month])

  const dayKey = (day: number) =>
    `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`

  const billsByDay = (day: number) => bills.filter((b) => b.due_date === dayKey(day))

  const selectedBills = selectedDay ? bills.filter((b) => b.due_date === selectedDay) : []
  const totalDay = selectedBills.reduce((sum, b) => sum + b.amount, 0)
  const weekdays = lang === 'es' ? WEEKDAYS_ES : WEEKDAYS_EN
  const todayKey = dateKey(now)

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <button
          onClick={goPrev}
          className="w-10 h-10 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
          title={t('app.close')}
        >
          <Icon name="chevron" className="w-4 h-4 rotate-180" />
        </button>
        <h3 className="text-base font-semibold">
          {t(`months.${month}`)} {year}
        </h3>
        <button
          onClick={goNext}
          className="w-10 h-10 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
          title={t('app.close')}
        >
          <Icon name="chevron" className="w-4 h-4" />
        </button>
      </div>

      <div className="grid grid-cols-7 gap-1 sm:gap-2">
        {weekdays.map((wd) => (
          <div key={wd} className="text-center text-xs font-semibold text-text-secondary py-1">
            {wd}
          </div>
        ))}
        {cells.map((day, i) =>
          day === null ? (
            <div key={`empty-${i}`} />
          ) : (
            <button
              key={day}
              onClick={() => setSelectedDay(dayKey(day))}
              className={`relative min-h-[44px] sm:min-h-[56px] rounded-ios-sm border text-sm flex flex-col items-center justify-center gap-1 transition-colors ${
                selectedDay === dayKey(day)
                  ? 'bg-primary/10 border-primary text-primary font-semibold'
                  : 'bg-card border-border hover:bg-bg'
              } ${todayKey === dayKey(day) ? 'ring-2 ring-primary/40' : ''}`}
            >
              <span>{day}</span>
              {billsByDay(day).length > 0 && (
                <span className="flex gap-0.5">
                  {billsByDay(day).slice(0, 3).map((b) => (
                    <span
                      key={b.id}
                      className={`w-1.5 h-1.5 rounded-full ${
                        b.status === 'paid' ? 'bg-success' : 'bg-warning'
                      }`}
                    />
                  ))}
                </span>
              )}
            </button>
          )
        )}
      </div>

      {selectedDay && (
        <div className="mt-4 bg-card rounded-ios shadow-ios p-4">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-semibold">{selectedDay}</h4>
            {selectedBills.length > 0 && (
              <span className="text-sm text-text-secondary">
                {t('deudas.total_day')}:{' '}
                <strong>{formatMoney(totalDay, selectedBills[0].currency_code)}</strong>
              </span>
            )}
          </div>
          {selectedBills.length === 0 ? (
            <p className="text-sm text-text-secondary">{t('deudas.no_bills_day')}</p>
          ) : (
            <div className="space-y-2">
              {selectedBills.map((b) => (
                <div
                  key={b.id}
                  className="flex items-center justify-between gap-3 border border-border rounded-ios-sm p-3"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{b.debt_description}</p>
                    <p className="text-xs text-text-secondary">
                      {t('deudas.installment')} #{b.installment_number} · {b.institution_name}
                    </p>
                    <p className="text-sm font-semibold mt-1">
                      {formatMoney(b.amount, b.currency_code)}
                    </p>
                  </div>
                  <div className="flex flex-col items-end gap-2 shrink-0">
                    <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                      b.status === 'paid' ? 'bg-success/20 text-green-800 dark:text-green-400' : 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
                    }`}>
                      {t(`bills.status_${b.status}`)}
                    </span>
                    {b.status === 'pending' && (
                      <button
                        onClick={() => setPayTarget(b)}
                        className="text-xs font-semibold text-primary hover:underline"
                      >
                        {t('deudas.pay')}
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {payTarget && (
        <DebtPayModal
          bill={payTarget}
          onClose={() => setPayTarget(null)}
          onSuccess={() => load(year, month)}
        />
      )}
    </div>
  )
}