import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../../api'
import { useI18nStore } from '../../stores/i18nStore'
import { Badge, Money, SectionCard } from '../rental/RentalUI'
import Select from '../Select'
import LoadingSpinner from '../LoadingSpinner'
import { FinanceBars, FinanceDonut, type DonutSlice, type MonthPoint } from './HomeFinanceCharts'
import type { FinanceBreakdownKind, FinanceItem, HomeFinances, PortfolioMoney } from '../../types/rental'

const BREAKDOWN_COLORS: Record<FinanceBreakdownKind, string> = {
  services: 'rgb(var(--color-primary))',
  maintenance: 'rgb(var(--color-warning))',
  commission: 'rgb(var(--color-accent))',
}

const TYPE_TONES: Record<FinanceItem['type'], 'success' | 'warning' | 'danger' | 'neutral' | 'info' | 'accent'> = {
  payment: 'success',
  reservation: 'warning',
  bill: 'danger',
  maintenance: 'warning',
  commission: 'accent',
}

function amountOf(list: PortfolioMoney[] | undefined, currency: string): number {
  return list?.find((item) => item.currency === currency)?.amount ?? 0
}

function pickCurrency(data: HomeFinances): string {
  let best = ''
  let bestTotal = -1
  for (const currency of data.currencies) {
    const total = data.months.reduce(
      (sum, month) => sum + amountOf(month.income, currency) + amountOf(month.expenses, currency),
      0,
    )
    if (total > bestTotal) {
      best = currency
      bestTotal = total
    }
  }
  return best
}

function formatDate(value: string, lang: string): string {
  if (!value) return ''
  const date = new Date(value.length >= 10 ? `${value.slice(0, 10)}T00:00:00` : value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString(lang, { day: '2-digit', month: 'short', year: 'numeric' })
}

function itemLink(item: FinanceItem, homeId: number): string | null {
  switch (item.type) {
    case 'payment':
      return item.lease_id ? `/leases/${item.lease_id}` : null
    case 'reservation':
      return item.reservation_id ? `/reservations/${item.reservation_id}` : null
    case 'bill':
      return item.service_id ? `/services/bills/${item.service_id}` : null
    case 'maintenance':
      return item.maintenance_id ? `/home/${homeId}/maintenances/${item.maintenance_id}` : null
    case 'commission':
      return item.lease_id ? `/leases/${item.lease_id}` : null
    default:
      return null
  }
}

function Kpi({ label, children, strong = false }: { label: string; children: ReactNode; strong?: boolean }) {
  return (
    <div className={`rounded-ios-sm bg-bg/60 px-3 py-2 ${strong ? 'ring-1 ring-border' : ''}`}>
      <p className="text-xs text-text-secondary">{label}</p>
      <p className={`mt-0.5 tabular-nums ${strong ? 'text-base font-semibold' : 'text-sm font-medium'}`}>{children}</p>
    </div>
  )
}

// HomeFinancesTab muestra la serie de 12 meses, la composición de gastos y los
// movimientos del mes de una casa (REQ-019 a REQ-022).
export default function HomeFinancesTab({ homeId }: { homeId: number }) {
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  const [period, setPeriod] = useState(() => {
    const now = new Date()
    return { year: now.getFullYear(), month: now.getMonth() + 1 }
  })
  const [data, setData] = useState<HomeFinances | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [currency, setCurrency] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    api.rental.portfolio.homeFinances(homeId, period.year, period.month)
      .then((result) => {
        if (!cancelled) setData(result)
      })
      .catch(() => {
        if (!cancelled) {
          setData(null)
          setError(t('home.finances.load_error'))
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [homeId, period.year, period.month, reloadKey])

  useEffect(() => {
    if (!data) return
    if (data.currencies.length === 0) {
      setCurrency('')
      return
    }
    setCurrency((current) => (data.currencies.includes(current) ? current : pickCurrency(data)))
  }, [data])

  const shiftMonth = (delta: number) => {
    setPeriod((current) => {
      const date = new Date(current.year, current.month - 1 + delta, 1)
      return { year: date.getFullYear(), month: date.getMonth() + 1 }
    })
  }

  if (loading && !data) return <LoadingSpinner />

  if (error && !data) {
    return (
      <SectionCard title={t('home.finances.title')}>
        <p className="text-sm text-danger">{error}</p>
        <button
          type="button"
          onClick={() => setReloadKey((value) => value + 1)}
          className="mt-3 px-4 py-2 rounded-ios-sm border border-border hover:bg-bg min-h-[44px]"
        >
          {t('home.finances.retry')}
        </button>
      </SectionCard>
    )
  }

  if (!data) return null

  const points: MonthPoint[] = data.months.map((month) => ({
    year: month.year,
    month: month.month,
    income: amountOf(month.income, currency),
    expenses: amountOf(month.expenses, currency),
  }))

  const slices: DonutSlice[] = data.breakdown
    .map((item) => ({
      kind: item.kind,
      label: t(`home.finances.category.${item.kind}`, item.kind),
      amount: amountOf(item.amounts, currency),
      color: BREAKDOWN_COLORS[item.kind] ?? BREAKDOWN_COLORS.services,
    }))
    .filter((slice) => slice.amount > 0)

  const currencyOptions = data.currencies.map((code) => ({ value: code, label: code }))
  const amountNode = (list: PortfolioMoney[]) => (
    currency
      ? <Money amount={amountOf(list, currency)} currency={currency} />
      : <span className="text-text-secondary">—</span>
  )

  return (
    <div className="space-y-4">
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="font-semibold">{t('home.finances.title')}</h3>
            <p className="text-sm text-text-secondary mt-0.5">
              {t(`months.${data.month}`)} {data.year}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {currencyOptions.length > 1 && (
              <div className="w-24">
                <Select
                  options={currencyOptions}
                  value={currency}
                  onChange={(value) => setCurrency(String(value))}
                  placeholder={t('home.finances.currency')}
                />
              </div>
            )}
            <button
              type="button"
              onClick={() => shiftMonth(-1)}
              aria-label={t('home.portfolio.prev_month')}
              className="w-9 h-9 rounded-ios-sm border border-border hover:bg-bg flex items-center justify-center text-lg leading-none"
            >
              ‹
            </button>
            <button
              type="button"
              onClick={() => shiftMonth(1)}
              aria-label={t('home.portfolio.next_month')}
              className="w-9 h-9 rounded-ios-sm border border-border hover:bg-bg flex items-center justify-center text-lg leading-none"
            >
              ›
            </button>
          </div>
        </div>

        <div className="grid grid-cols-2 lg:grid-cols-5 gap-2">
          <Kpi label={t('home.portfolio.income')}>{amountNode(data.income)}</Kpi>
          <Kpi label={t('home.portfolio.expenses')}>{amountNode(data.expenses)}</Kpi>
          <Kpi label={t('home.portfolio.net')} strong>{amountNode(data.net)}</Kpi>
          <Kpi label={t('home.portfolio.pending_income')}>{amountNode(data.pending_income)}</Kpi>
          <Kpi label={t('home.portfolio.pending_expenses')}>{amountNode(data.pending_expenses)}</Kpi>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <SectionCard title={t('home.finances.series_title')}>
          <FinanceBars points={points} currency={currency} />
        </SectionCard>
        <SectionCard title={t('home.finances.breakdown_title')}>
          <FinanceDonut slices={slices} currency={currency} />
        </SectionCard>
      </div>

      <SectionCard title={t('home.finances.movements_title')}>
        {data.items.length === 0 ? (
          <p className="text-sm text-text-secondary py-4 text-center">{t('home.finances.no_movements')}</p>
        ) : (
          <ul className="divide-y divide-border">
            {data.items.map((item, index) => {
              const link = itemLink(item, homeId)
              const detail = item.type === 'payment'
                ? [item.method ? t(`rental.method.${item.method}`, item.method) : '', item.detail].filter(Boolean).join(' · ')
                : item.type === 'maintenance'
                  ? [item.kind ? t(`homeMaintenance.kind_${item.kind}`, item.kind) : '', item.area ? t(`homeMaintenance.area_${item.area}`, item.area) : ''].filter(Boolean).join(' · ')
                  : item.type === 'commission' && item.detail === 'fixed'
                    ? t('home.finances.fixed')
                    : item.detail

              const content = (
                <>
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge tone={TYPE_TONES[item.type] ?? 'neutral'}>
                        {t(`home.finances.type.${item.type}`, item.type)}
                      </Badge>
                      <span className="text-xs text-text-secondary">{formatDate(item.date, lang)}</span>
                      {item.status && (
                        <span className="text-xs text-text-secondary">
                          {t(`home.finances.status.${item.status}`, item.status)}
                        </span>
                      )}
                    </div>
                    {item.description && (
                      <p className="text-sm font-medium mt-0.5 truncate">{item.description}</p>
                    )}
                    {detail && <p className="text-xs text-text-secondary truncate">{detail}</p>}
                  </div>
                  <div className="text-right shrink-0 space-y-0.5">
                    {item.amounts.map((amount) => (
                      <p key={amount.currency} className="text-sm">
                        <Money amount={amount.amount} currency={amount.currency} />
                      </p>
                    ))}
                  </div>
                </>
              )

              return (
                <li key={`${item.type}-${index}`}>
                  {link ? (
                    <button
                      type="button"
                      onClick={() => navigate(link)}
                      className="w-full py-2.5 flex items-start justify-between gap-3 text-left hover:bg-bg/50 rounded-ios-sm px-1"
                    >
                      {content}
                    </button>
                  ) : (
                    <div className="py-2.5 flex items-start justify-between gap-3 px-1">{content}</div>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </SectionCard>
    </div>
  )
}
