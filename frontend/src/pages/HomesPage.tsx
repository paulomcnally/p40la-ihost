import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import { Money } from '../components/rental/RentalUI'
import HomeStateBadge from '../components/home/HomeStateBadge'
import HomeStateFilterModal from '../components/home/HomeStateFilterModal'
import PortfolioSummary from '../components/home/PortfolioSummary'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { SortableGrid } from '../components/sortable/SortableGrid'
import { SortableCard } from '../components/sortable/SortableCard'
import { useSortableOrder } from '../hooks/useSortableOrder'
import { useToast } from '../components/Toast'
import type { Home, HomeState } from '../types'
import type { PortfolioHomeSummary, PortfolioSummary as PortfolioSummaryData } from '../types/rental'

const ALL_STATES: HomeState[] = ['listed', 'reserved', 'rented', 'inhabited', 'unused']

export default function HomesPage() {
  const navigate = useNavigate()
  const { homes, loadHomes, setHomes } = useAppStore()
  const { t } = useI18nStore()
  usePageTitle(t('home.title'))
  const { showToast } = useToast()
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)
  const [summary, setSummary] = useState<PortfolioSummaryData | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(true)
  const [summaryError, setSummaryError] = useState<string | null>(null)
  const [filterOpen, setFilterOpen] = useState(false)
  const [stateFilter, setStateFilter] = useState<HomeState[]>(ALL_STATES)
  const [period, setPeriod] = useState(() => {
    const now = new Date()
    return { year: now.getFullYear(), month: now.getMonth() + 1 }
  })

  useEffect(() => {
    loadHomes().finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    let cancelled = false
    setSummaryLoading(true)
    setSummaryError(null)
    api.rental.portfolio.summary(period.year, period.month)
      .then((data) => {
        if (!cancelled) setSummary(data)
      })
      .catch(() => {
        if (!cancelled) {
          setSummary(null)
          setSummaryError(t('home.portfolio.load_error'))
        }
      })
      .finally(() => {
        if (!cancelled) setSummaryLoading(false)
      })
    return () => { cancelled = true }
  }, [period.year, period.month])

  const shiftMonth = (delta: number) => {
    setPeriod((current) => {
      const date = new Date(current.year, current.month - 1 + delta, 1)
      return { year: date.getFullYear(), month: date.getMonth() + 1 }
    })
  }

  const { handleReorder } = useSortableOrder<Home>({
    items: homes,
    setItems: setHomes,
    onPersist: (ids) => api.homes.reorder(ids),
    onError: () => showToast(t('home.order_error'), 'error'),
  })

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.homes.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadHomes()
  }, [deleteTarget])

  if (loading) {
    return <LoadingSpinner />
  }

  if (homes.length === 0) {
    return (
      <EmptyCard
        icon="home"
        title={t('home.empty')}
        subtitle={t('home.subtitle')}
        actionLabel={t('home.create')}
        onAction={() => navigate('/home/new')}
      />
    )
  }

  const createOptions = [
    { label: t('home.create'), icon: 'plus', onClick: () => navigate('/home/new') },
  ]

  const summaries = new Map<number, PortfolioHomeSummary>(
    (summary?.homes ?? []).map((item) => [item.home_id, item]),
  )
  const filterActive = stateFilter.length !== ALL_STATES.length
  const visibleHomes = filterActive && summary
    ? homes.filter((home) => {
        const state = summaries.get(home.id)?.state
        return state ? stateFilter.includes(state) : false
      })
    : homes

  const cardBody = (home: Home) => {
    const cardSummary = summaries.get(home.id)
    return (
      <>
        <CardMenu
          options={[
            { label: t('app.view'), onClick: () => navigate(`/home/${home.id}`) },
            { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/home/edit/${home.id}`) },
            { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: home.id, name: home.name }) },
          ]}
        />
        <div
          role="link"
          tabIndex={0}
          onClick={() => navigate(`/home/${home.id}`)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              navigate(`/home/${home.id}`)
            }
          }}
          className="cursor-pointer"
          aria-label={`${t('app.view')}: ${home.name}`}
        >
          <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
            <Icon name="home" className="w-6 h-6" />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="font-semibold text-base truncate">{home.name}</h3>
            {cardSummary && <HomeStateBadge state={cardSummary.state} />}
          </div>
          {home.address && (
            <p className="text-sm text-text-secondary mt-1">{home.address}</p>
          )}
          {cardSummary && cardSummary.net.length > 0 && (
            <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-sm mt-2">
              <span className="text-xs text-text-secondary">{t('home.portfolio.month_net')}:</span>
              {cardSummary.net.map((item) => (
                <Money key={item.currency} amount={item.amount} currency={item.currency} />
              ))}
            </div>
          )}
        </div>
      </>
    )
  }

  const gridClass = 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4'

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-2 mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('home.title')}</h2>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setFilterOpen(true)}
            aria-label={t('home.filter.title')}
            className="inline-flex items-center gap-2 px-3 py-2 rounded-ios-sm border border-border bg-card hover:bg-bg transition-colors min-h-[44px] text-sm font-medium"
          >
            <Icon name="filter" className="w-4 h-4 text-text-secondary" />
            <span>{t('app.filter')}</span>
            {filterActive && (
              <span className="min-w-5 h-5 px-1 rounded-full bg-primary text-white text-xs flex items-center justify-center tabular-nums">
                {stateFilter.length}
              </span>
            )}
          </button>
          <CreateMenu options={createOptions} />
        </div>
      </div>

      <div className="mb-4">
        <PortfolioSummary
          summary={summary}
          loading={summaryLoading}
          error={summaryError}
          year={period.year}
          month={period.month}
          onPrev={() => shiftMonth(-1)}
          onNext={() => shiftMonth(1)}
        />
      </div>

      {visibleHomes.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center">
          <p className="font-medium">{t('home.filter.no_matches')}</p>
          <button
            type="button"
            onClick={() => setStateFilter(ALL_STATES)}
            className="mt-3 inline-flex items-center gap-2 px-4 py-2 rounded-ios-sm border border-border hover:bg-bg min-h-[44px]"
          >
            <Icon name="filter" className="w-4 h-4" />
            {t('home.filter.clear')}
          </button>
        </div>
      ) : filterActive ? (
        <div className={gridClass}>
          {visibleHomes.map((home) => (
            <div key={home.id} className="relative bg-card rounded-ios shadow-ios p-4">
              {cardBody(home)}
            </div>
          ))}
        </div>
      ) : (
        <SortableGrid items={homes} onReorder={handleReorder} className={gridClass}>
          {(home) => (
            <SortableCard
              key={home.id}
              id={home.id}
              handle={<Icon name="gripVertical" className="w-4 h-4" />}
              handleAriaLabel={t('home.reorder')}
              className="relative bg-card rounded-ios shadow-ios p-4 pl-10"
            >
              {cardBody(home)}
            </SortableCard>
          )}
        </SortableGrid>
      )}

      <HomeStateFilterModal
        isOpen={filterOpen}
        selected={stateFilter}
        counts={summary?.state_counts ?? {}}
        onApply={(states) => {
          setStateFilter(states)
          setFilterOpen(false)
        }}
        onClose={() => setFilterOpen(false)}
      />

      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('home.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function EmptyCard({ icon, title, subtitle, actionLabel, onAction }: {
  icon: string; title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name={icon} className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-text-secondary mb-6">{subtitle}</p>
      <button
        onClick={onAction}
        className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
      >
        <Icon name="plus" className="w-5 h-5" />
        {actionLabel}
      </button>
    </div>
  )
}
