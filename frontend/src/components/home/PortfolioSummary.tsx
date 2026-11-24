import type { ReactNode } from 'react'
import { Money } from '../rental/RentalUI'
import { useI18nStore } from '../../stores/i18nStore'
import type { HomeState } from '../../types'
import type { PortfolioMoney, PortfolioSummary } from '../../types/rental'

interface Props {
  summary: PortfolioSummary | null
  loading: boolean
  error: string | null
  year: number
  month: number
  onPrev: () => void
  onNext: () => void
}

const STATE_ORDER: HomeState[] = ['rented', 'reserved', 'listed', 'inhabited', 'unused']

function MoneyList({ items }: { items: PortfolioMoney[] }) {
  if (!items || items.length === 0) return <span className="text-text-secondary">—</span>
  return (
    <span className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
      {items.map((item) => (
        <Money key={item.currency} amount={item.amount} currency={item.currency} />
      ))}
    </span>
  )
}

function Metric({ label, children, strong = false }: { label: string; children: ReactNode; strong?: boolean }) {
  return (
    <div className={`rounded-ios-sm bg-bg/60 px-3 py-2 ${strong ? 'ring-1 ring-border' : ''}`}>
      <p className="text-xs text-text-secondary">{label}</p>
      <p className={`mt-0.5 tabular-nums ${strong ? 'text-base font-semibold' : 'text-sm font-medium'}`}>{children}</p>
    </div>
  )
}

// PortfolioSummary muestra el resumen mensual de ingresos y gastos (SPEC-143).
export default function PortfolioSummary({ summary, loading, error, year, month, onPrev, onNext }: Props) {
  const { t } = useI18nStore()
  const totals = summary?.totals
  const counts = summary?.state_counts ?? {}

  return (
    <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="font-semibold">{t('home.portfolio.title')}</h3>
          <p className="text-xs text-text-secondary mt-0.5 max-w-2xl">{t('home.portfolio.criteria')}</p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            type="button"
            onClick={onPrev}
            aria-label={t('home.portfolio.prev_month')}
            className="w-9 h-9 rounded-ios-sm border border-border hover:bg-bg flex items-center justify-center text-lg leading-none"
          >
            ‹
          </button>
          <span className="text-sm font-medium min-w-[112px] text-center">
            {t(`months.${month}`)} {year}
          </span>
          <button
            type="button"
            onClick={onNext}
            aria-label={t('home.portfolio.next_month')}
            className="w-9 h-9 rounded-ios-sm border border-border hover:bg-bg flex items-center justify-center text-lg leading-none"
          >
            ›
          </button>
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-2 lg:grid-cols-5 gap-2" aria-busy="true">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-16 rounded-ios-sm bg-bg animate-pulse" />
          ))}
        </div>
      ) : error ? (
        <p className="text-sm text-danger">{error}</p>
      ) : (
        <>
          <div className="grid grid-cols-2 lg:grid-cols-5 gap-2">
            <Metric label={t('home.portfolio.income')}><MoneyList items={totals?.income ?? []} /></Metric>
            <Metric label={t('home.portfolio.expenses')}><MoneyList items={totals?.expenses ?? []} /></Metric>
            <Metric label={t('home.portfolio.net')} strong><MoneyList items={totals?.net ?? []} /></Metric>
            <Metric label={t('home.portfolio.pending_income')}><MoneyList items={totals?.pending_income ?? []} /></Metric>
            <Metric label={t('home.portfolio.pending_expenses')}><MoneyList items={totals?.pending_expenses ?? []} /></Metric>
          </div>

          <div className="flex flex-wrap gap-2">
            {STATE_ORDER.filter((state) => (counts[state] ?? 0) > 0).map((state) => (
              <span key={state} className="inline-flex items-center gap-1.5 text-xs text-text-secondary">
                <span className="px-2 py-0.5 rounded-full bg-bg font-medium text-text">
                  {t(`home.state.${state}`, state)}
                </span>
                {counts[state]}
              </span>
            ))}
          </div>
        </>
      )}
    </section>
  )
}
