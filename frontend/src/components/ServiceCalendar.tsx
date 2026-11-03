import { useEffect, useMemo, useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import LoadingSpinner from './LoadingSpinner'
import PayBillModal from './PayBillModal'
import type { ServiceBillDetail } from '../types'

const WEEKDAYS_ES = ['Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb', 'Dom']
const WEEKDAYS_EN = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

function dateKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

interface ServiceCalendarProps {
  homeId: number | null
  year: number
  month: number
  onChangePeriod: (year: number, month: number) => void
}

function totalsByCurrency(bills: ServiceBillDetail[]) {
  const map = new Map<string, number>()
  for (const b of bills) {
    const code = b.currency_code || 'NIO'
    map.set(code, (map.get(code) || 0) + b.amount)
  }
  return map
}

function BillRow({ bill, onPay }: { bill: ServiceBillDetail; onPay: (b: ServiceBillDetail) => void }) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  return (
    <div className="flex items-center justify-between gap-3 border border-border rounded-ios-sm p-3">
      <div className="min-w-0">
        <p className="text-sm font-medium truncate">{bill.service_name}</p>
        <p className="text-xs text-text-secondary truncate">
          {bill.institution}
          {bill.date_estimated && ` · ${t('services.analysis_date_estimated')}`}
        </p>
        <p className="text-sm font-semibold mt-1">
          {bill.currency_code || 'NIO'} {formatMoney(bill.amount, bill.currency_symbol || bill.currency_code)}
        </p>
      </div>
      <div className="flex flex-col items-end gap-2 shrink-0 min-w-0">
        <span className={`text-xs font-semibold px-2.5 py-1 rounded-full whitespace-nowrap ${
          bill.status === 'paid' ? 'bg-success/20 text-green-800 dark:text-green-400' : 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
        }`}>
          {t(`bills.status_${bill.status}`)}
        </span>
        {bill.status === 'pending' && (
          <button onClick={() => onPay(bill)} className="text-xs font-semibold text-primary hover:underline min-h-[36px]">
            {t('bills.pay')}
          </button>
        )}
      </div>
    </div>
  )
}

export default function ServiceCalendar({ homeId, year, month, onChangePeriod }: ServiceCalendarProps) {
  const { t, lang } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const now = useMemo(() => new Date(), [])
  const [bills, setBills] = useState<ServiceBillDetail[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedDay, setSelectedDay] = useState<string | null>(dateKey(now))
  const [payTarget, setPayTarget] = useState<ServiceBillDetail | null>(null)

  useEffect(() => {
    let active = true
    setLoading(true)
    api.serviceBills
      .byMonth(year, month, homeId)
      .then((list) => {
        if (active) setBills(list || [])
      })
      .catch(() => {
        if (active) setBills([])
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [year, month, homeId])

  useEffect(() => {
    setSelectedDay((prev) => {
      if (!prev) return prev
      const prefix = `${year}-${String(month).padStart(2, '0')}`
      return prev.startsWith(prefix) ? prev : null
    })
  }, [year, month])

  const goPrev = () => {
    if (month === 1) onChangePeriod(year - 1, 12)
    else onChangePeriod(year, month - 1)
  }
  const goNext = () => {
    if (month === 12) onChangePeriod(year + 1, 1)
    else onChangePeriod(year, month + 1)
  }

  const cells = useMemo(() => {
    const offset = (new Date(year, month - 1, 1).getDay() + 6) % 7
    const daysInMonth = new Date(year, month, 0).getDate()
    const out: (number | null)[] = Array(offset).fill(null)
    for (let d = 1; d <= daysInMonth; d++) out.push(d)
    return out
  }, [year, month])

  const dayKey = (day: number) =>
    `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`

  const billsByDay = (day: number) => bills.filter((b) => b.period_date === dayKey(day))

  const daysWithBills = useMemo(() => {
    const dates = [...new Set(bills.map((b) => b.period_date))]
    return dates.sort()
  }, [bills])

  const selectedBills = selectedDay ? bills.filter((b) => b.period_date === selectedDay) : []
  const selectedPending = selectedBills.filter((b) => b.status === 'pending')
  const selectedTotals = totalsByCurrency(selectedPending)
  const weekdays = lang === 'es' ? WEEKDAYS_ES : WEEKDAYS_EN
  const todayKey = dateKey(now)

  if (loading) return <LoadingSpinner text={t('app.loading')} />

  return (
    <div>
      <div className="flex items-center justify-between gap-1 mb-3">
        <button
          onClick={goPrev}
          className="w-10 h-10 shrink-0 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
          aria-label={t('services.analysis_prev_month')}
        >
          <Icon name="chevron" className="w-4 h-4 rotate-180" />
        </button>
        <h3 className="text-base font-semibold min-w-0 truncate text-center">
          {t(`months.${month}`)} {year}
        </h3>
        <button
          onClick={goNext}
          className="w-10 h-10 shrink-0 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
          aria-label={t('services.analysis_next_month')}
        >
          <Icon name="chevron" className="w-4 h-4" />
        </button>
      </div>

      {/* Grilla desktop */}
      <div className="hidden sm:block">
        {bills.length === 0 && (
          <div className="bg-card rounded-ios shadow-ios p-6 mb-4 text-center">
            <div className="w-12 h-12 mx-auto mb-3 text-primary opacity-80">
              <Icon name="calendar" className="w-full h-full" />
            </div>
            <p className="text-sm font-semibold">{t('services.calendar_empty_month')}</p>
            <p className="text-xs text-text-secondary mt-1">{t('services.calendar_empty_month_subtitle')}</p>
          </div>
        )}
        <div className="grid grid-cols-7 gap-2">
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
                className={`relative min-h-[56px] rounded-ios-sm border text-sm flex flex-col items-center justify-center gap-1 transition-colors ${
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
                        className={`w-1.5 h-1.5 rounded-full ${b.status === 'paid' ? 'bg-success' : 'bg-warning'}`}
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
            <div className="flex items-center justify-between gap-3 mb-3">
              <h4 className="font-semibold min-w-0 truncate">{selectedDay}</h4>
              {selectedBills.length > 0 && (
                <span className="text-sm text-text-secondary shrink-0 text-right">
                  {selectedPending.length > 0 ? (
                    <>
                      {t('services.calendar_total_day')}:{' '}
                      <strong>
                        {[...selectedTotals.entries()].map(([code, amount]) => `${code} ${formatMoney(amount)}`).join(' · ')}
                      </strong>
                    </>
                  ) : (
                    <span className="text-emerald-600 dark:text-emerald-400">{t('services.calendar_all_paid_day')}</span>
                  )}
                </span>
              )}
            </div>
            {selectedBills.length === 0 ? (
              <p className="text-sm text-text-secondary">{t('services.calendar_no_bills_day')}</p>
            ) : (
              <div className="space-y-2">
                {selectedBills.map((b) => (
                  <BillRow key={b.id} bill={b} onPay={setPayTarget} />
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Agenda móvil */}
      <div className="sm:hidden space-y-3">
        {daysWithBills.length === 0 ? (
          <div className="bg-card rounded-ios shadow-ios p-8 text-center">
            <div className="w-12 h-12 mx-auto mb-3 text-primary opacity-80">
              <Icon name="calendar" className="w-full h-full" />
            </div>
            <p className="text-sm font-semibold">{t('services.calendar_empty_month')}</p>
            <p className="text-xs text-text-secondary mt-1">{t('services.calendar_empty_month_subtitle')}</p>
          </div>
        ) : (
          daysWithBills.map((d) => {
            const dayBills = bills.filter((b) => b.period_date === d)
            const dayPending = dayBills.filter((b) => b.status === 'pending')
            const dayTotals = totalsByCurrency(dayPending)
            const isToday = d === todayKey
            const dObj = new Date(`${d}T00:00:00`)
            const weekday = weekdays[(dObj.getDay() + 6) % 7]
            const dayNum = Number(d.slice(8, 10))
            return (
              <div key={d} className="bg-card rounded-ios shadow-ios p-3">
                <div className="flex items-center justify-between gap-3 mb-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className={`w-9 h-9 shrink-0 rounded-full flex items-center justify-center text-sm font-bold ${
                      isToday ? 'bg-primary text-white' : 'bg-bg text-text'
                    }`}>
                      {dayNum}
                    </span>
                    <div className="min-w-0">
                      <p className={`text-sm font-semibold truncate ${isToday ? 'text-primary' : ''}`}>
                        {weekday}, {t(`months.${Number(d.slice(5, 7))}`)} {dayNum}
                      </p>
                      <p className="text-xs text-text-secondary truncate">
                        {dayPending.length > 0
                          ? `${dayPending.length} ${t('services.view_pending').toLowerCase()} · ${[...dayTotals.entries()].map(([code, amount]) => `${code} ${formatMoney(amount)}`).join(' · ')}`
                          : t('services.calendar_all_paid_day')}
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={() => setSelectedDay(d)}
                    className="text-xs font-semibold text-primary shrink-0 min-h-[44px]"
                  >
                    {t('services.calendar_view_day')}
                  </button>
                </div>
                <div className="space-y-2">
                  {dayBills.map((b) => (
                    <BillRow key={b.id} bill={b} onPay={setPayTarget} />
                  ))}
                </div>
              </div>
            )
          })
        )}
      </div>

      {payTarget && (
        <PayBillModal bill={payTarget} onClose={() => setPayTarget(null)} onSuccess={() => {
          api.serviceBills.byMonth(year, month, homeId).then((list) => setBills(list || [])).catch(() => undefined)
        }} />
      )}
    </div>
  )
}
