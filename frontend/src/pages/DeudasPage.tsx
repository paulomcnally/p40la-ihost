import { useEffect, useState, useCallback, useMemo } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import DebtCalendar from '../components/DebtCalendar'
import DebtAnalysis from '../components/DebtAnalysis'
import DebtChart from '../components/DebtChart'
import { debtClosedDate, isDebtPaid, isDebtPayable } from '../utils/debtMonthlyAggregation'
import type { Debt, Institution } from '../types'

type TabKey = 'calendario' | 'deudas' | 'analisis' | 'grafica'
type DebtView = 'pending' | 'paid' | 'paused' | 'all'

const VIEWS: DebtView[] = ['pending', 'paid', 'paused', 'all']

// Compatibilidad con enlaces viejos que usaban el estado crudo en la URL.
const LEGACY_VIEWS: Record<string, DebtView> = {
  activa: 'pending',
  finalizada: 'paid',
  inactiva: 'paused',
}

export default function DeudasPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { currencies, loadAll } = useAppStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = (searchParams.get('tab') as TabKey | null) ?? 'analisis'
  const rawFilter = searchParams.get('status')
  const debtView: DebtView = VIEWS.includes(rawFilter as DebtView)
    ? (rawFilter as DebtView)
    : LEGACY_VIEWS[rawFilter ?? ''] ?? 'pending'
  const tabTitle = tab === 'calendario' ? t('deudas.tab_calendar') : tab === 'deudas' ? t('deudas.tab_debts') : tab === 'grafica' ? t('deudas.tab_chart') : t('deudas.tab_analysis')
  usePageTitle(tabTitle)

  const [debts, setDebts] = useState<Debt[]>([])
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  const loadDebts = useCallback(async () => {
    const list = await api.debts.list()
    setDebts(list || [])
  }, [])

  useEffect(() => {
    loadAll()
    api.institutions
      .list()
      .then((list) => setInstitutions(list || []))
      .catch(() => setInstitutions([]))
  }, [])

  useEffect(() => {
    loadDebts().finally(() => setLoading(false))
  }, [loadDebts])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.debts.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadDebts()
  }, [deleteTarget, loadDebts])

  const setTab = (key: TabKey) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('tab', key)
      return next
    }, { replace: false })
  }

  const setDebtView = (key: DebtView) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('status', key)
      return next
    }, { replace: false })
  }

  const counts = useMemo(() => ({
    pending: debts.filter(isDebtPayable).length,
    paid: debts.filter(isDebtPaid).length,
    paused: debts.filter((d) => d.status !== 'activa' && !isDebtPaid(d)).length,
    all: debts.length,
  }), [debts])

  const filteredDebts = useMemo(() => {
    const sorted = [...debts].sort((a, b) => a.description.localeCompare(b.description))
    switch (debtView) {
      case 'pending':
        return sorted.filter(isDebtPayable)
      case 'paid':
        return sorted
          .filter(isDebtPaid)
          .sort((a, b) => debtClosedDate(b).localeCompare(debtClosedDate(a)) || a.description.localeCompare(b.description))
      case 'paused':
        return sorted.filter((d) => d.status !== 'activa' && !isDebtPaid(d))
      default:
        return sorted
    }
  }, [debts, debtView])

  // Historial agrupado por año de cierre (SPEC-104 REQ-004).
  const paidGroups = useMemo(() => {
    if (debtView !== 'paid') return []
    const map = new Map<string, Debt[]>()
    for (const debt of filteredDebts) {
      const year = debtClosedDate(debt).slice(0, 4) || '—'
      const group = map.get(year) ?? []
      group.push(debt)
      map.set(year, group)
    }
    return [...map.entries()]
  }, [filteredDebts, debtView])

  if (loading) return <LoadingSpinner />

  if (institutions.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="building" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.empty_no_institution')}</h3>
        <button
          onClick={() => navigate('/institutions/new')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
        >
          <Icon name="plus" className="w-5 h-5" />
          {'Nueva institución'}
        </button>
      </div>
    )
  }

  if (currencies.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="savings" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.empty_no_currency')}</h3>
        <button
          onClick={() => navigate('/settings/currency')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
        >
          <Icon name="plus" className="w-5 h-5" />
          {'Nueva moneda'}
        </button>
      </div>
    )
  }

  const createOptions = [
    { label: t('deudas.create'), icon: 'plus', onClick: () => navigate('/deudas/new') },
  ]

  const emptyMessage = debtView === 'pending'
    ? t('deudas.empty_pending')
    : debtView === 'paid'
      ? t('deudas.empty_paid')
      : debtView === 'paused'
        ? t('deudas.empty_paused')
        : t('deudas.empty')

  const renderCard = (debt: Debt) => {
    const currency = currencies.find((c) => c.id === debt.currency_id)
    const symbol = currency?.symbol || debt.currency_code
    const closed = isDebtPaid(debt)
    const closedDate = debtClosedDate(debt)
    const paidPct = debt.installments_total > 0
      ? Math.round((debt.paid_count / debt.installments_total) * 100)
      : 0
    return (
      <div
        key={debt.id}
        onClick={() => navigate(`/deudas/${debt.id}`)}
        className={`rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow ${
          debt.status === 'activa' ? 'bg-card' : 'bg-gray-100 opacity-70 dark:bg-[#2c2c2e]'
        }`}
      >
        <CardMenu
          options={[
            { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/deudas/edit/${debt.id}`) },
            { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: debt.id, name: debt.description }) },
          ]}
        />
        <div className="flex items-center justify-between mb-3">
          <div className={`w-11 h-11 rounded-ios flex items-center justify-center ${
            closed
              ? 'bg-success/15 text-emerald-600 dark:text-emerald-400'
              : debt.status === 'activa'
                ? 'bg-primary/10 text-primary'
                : 'bg-gray-200 text-gray-400 dark:bg-[#2c2c2e] dark:text-gray-500'
          }`}>
            <Icon name={closed ? 'check' : 'credit'} className="w-6 h-6" />
          </div>
        </div>
        <h3 className="font-semibold text-base flex items-center gap-2 min-w-0">
          <span className="truncate">{debt.description}</span>
          <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400 shrink-0">
            {`${t('deudas.id')}: ${debt.id}`}
          </span>
        </h3>
        <p className="text-sm text-text-secondary mt-1 truncate">
          {debt.institution_name}
          {debt.identifier && ` · ${debt.identifier}`}
        </p>
        <div className="flex items-center gap-2 mt-2 flex-wrap">
          {debt.status !== 'activa' && (
            <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400">
              {t(`deudas.status_${debt.status}`)}
            </span>
          )}
          {closed && closedDate && (
            <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-success/15 text-emerald-700 dark:text-emerald-400">
              {t('deudas.closed_on')} {closedDate}
            </span>
          )}
        </div>

        {closed ? (
          <>
            <div className="flex items-center justify-between mt-3">
              <p className="text-sm font-semibold">
                {t('deudas.total_paid')}: {formatMoney(debt.paid_amount, symbol)}
              </p>
              <span className="text-xs text-text-secondary">
                {debt.paid_count} {t('deudas.installments').toLowerCase()}
              </span>
            </div>
            <p className="text-xs text-text-secondary mt-1">
              {t('deudas.frees_per_month')}: {formatMoney(debt.installment_amount, symbol)}/{t('deudas.month').toLowerCase()}
            </p>
          </>
        ) : (
          <>
            <div className="flex items-center justify-between mt-3">
              <p className="text-sm font-semibold text-amber-600 dark:text-amber-400">
                {t('deudas.pending_balance')}: {formatMoney(debt.pending_amount, symbol)}
              </p>
              <span className="text-xs text-text-secondary">
                {debt.paid_count}/{debt.installments_total} {t('deudas.installments').toLowerCase()}
              </span>
            </div>
            <div className="h-1.5 rounded-full bg-border overflow-hidden mt-2">
              <div className="h-full rounded-full bg-success" style={{ width: `${paidPct}%` }} />
            </div>
            <p className="text-xs text-text-secondary mt-1.5 truncate">
              {debt.next_due_date
                ? `${t('deudas.next_due')}: ${debt.next_due_date} · ${formatMoney(debt.installment_amount, symbol)}`
                : `${t('deudas.installment_amount')}: ${formatMoney(debt.installment_amount, symbol)}`}
            </p>
          </>
        )}
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('deudas.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>

      <div className="flex gap-2 mb-4">
        {(
          [
            { key: 'analisis', label: t('deudas.tab_analysis'), icon: 'chart' },
            { key: 'calendario', label: t('deudas.tab_calendar'), icon: 'calendar' },
            { key: 'deudas', label: t('deudas.tab_debts'), icon: 'credit' },
            { key: 'grafica', label: t('deudas.tab_chart'), icon: 'bar' },
          ] as { key: TabKey; label: string; icon: string }[]
        ).map((item) => (
          <button
            key={item.key}
            onClick={() => setTab(item.key)}
            className={`px-4 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] inline-flex items-center gap-1.5 ${
              tab === item.key
                ? 'bg-primary text-white'
                : 'bg-card text-text-secondary hover:bg-border'
            }`}
          >
            <Icon name={item.icon} className="w-4 h-4" />
            {item.label}
          </button>
        ))}
      </div>

      {tab === 'grafica' ? (
        <DebtChart debts={debts} currencies={currencies} />
      ) : tab === 'calendario' ? (
        <DebtCalendar />
      ) : tab === 'analisis' ? (
        <DebtAnalysis />
      ) : debts.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="credit" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.empty')}</h3>
          <p className="text-text-secondary mb-6">{t('deudas.subtitle')}</p>
          <button
            onClick={() => navigate('/deudas/new')}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('deudas.create')}
          </button>
        </div>
      ) : (
        <>
          <div className="flex gap-2 mb-4 flex-wrap">
            {VIEWS.map((key) => (
              <button
                key={key}
                onClick={() => setDebtView(key)}
                className={`px-4 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
                  debtView === key
                    ? 'bg-primary text-white'
                    : 'bg-card text-text-secondary hover:bg-border'
                }`}
              >
                {t(`deudas.view_${key}`)} ({counts[key]})
              </button>
            ))}
          </div>

          {filteredDebts.length === 0 ? (
            <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
              <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
                <Icon name={debtView === 'paid' ? 'check' : 'credit'} className="w-full h-full" />
              </div>
              <h3 className="text-lg sm:text-xl font-semibold mb-2">{emptyMessage}</h3>
              <p className="text-text-secondary mb-6">{t('deudas.subtitle')}</p>
              <button
                onClick={() => navigate('/deudas/new')}
                className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
              >
                <Icon name="plus" className="w-5 h-5" />
                {t('deudas.create')}
              </button>
            </div>
          ) : debtView === 'paid' ? (
            <div className="space-y-6">
              {paidGroups.map(([year, group]) => (
                <section key={year}>
                  <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">
                    {year} · {group.length}
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
                    {group.map(renderCard)}
                  </div>
                </section>
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
              {filteredDebts.map(renderCard)}
            </div>
          )}
        </>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('deudas.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
