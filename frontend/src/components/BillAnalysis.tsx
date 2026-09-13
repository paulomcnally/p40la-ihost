import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import LoadingSpinner from './LoadingSpinner'
import Select from './Select'
import type { Bill } from '../types'

const INCREASE_COLOR = '#ff3b30'
const DECREASE_COLOR = '#30d158'
const NEUTRAL_COLOR = '#0a84ff'

type GroupBy = 'periodo' | 'anio'

interface Period {
  key: string
  label: string
  short: string
  year: number
  month: number
  amount: number
  paid: number
  pending: number
  bills: Bill[]
}

interface PeriodChange extends Period {
  delta: number
  pct: number | null
  previousAmount: number
  previousLabel: string
}

function periodSortValue(year: number, month: number) {
  return year * 100 + month
}

function formatPct(pct: number | null): string {
  if (pct === null) return '—'
  const sign = pct > 0 ? '+' : ''
  return `${sign}${pct.toFixed(1)}%`
}

export default function BillAnalysis({ serviceId, currencySymbol }: { serviceId: number; currencySymbol?: string }) {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)

  const [bills, setBills] = useState<Bill[]>([])
  const [loading, setLoading] = useState(true)
  const [yearFilter, setYearFilter] = useState<string>('all')
  const [groupBy, setGroupBy] = useState<GroupBy>('periodo')

  useEffect(() => {
    let active = true
    setLoading(true)
    api.bills
      .list(serviceId)
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
  }, [serviceId])

  const years = useMemo(() => {
    const set = new Set<number>()
    for (const b of bills) set.add(b.year)
    return [...set].sort((a, b) => b - a)
  }, [bills])

  const yearOptions = useMemo(
    () => [
      { value: 'all', label: t('bills.analysis_all_years') },
      ...years.map((y) => ({ value: String(y), label: String(y) })),
    ],
    [years, t]
  )

  const scopedBills = useMemo(
    () => (yearFilter === 'all' ? bills : bills.filter((b) => String(b.year) === yearFilter)),
    [bills, yearFilter]
  )

  const periods = useMemo<Period[]>(() => {
    const map = new Map<string, Period>()
    for (const b of scopedBills) {
      const key = groupBy === 'anio' ? String(b.year) : `${b.year}-${String(b.month).padStart(2, '0')}`
      const entry =
        map.get(key) ||
        ({
          key,
          label: '',
          short: '',
          year: b.year,
          month: b.month,
          amount: 0,
          paid: 0,
          pending: 0,
          bills: [],
        } as Period)
      entry.amount += b.amount
      if (b.status === 'paid') entry.paid += b.amount
      else entry.pending += b.amount
      entry.bills.push(b)
      map.set(key, entry)
    }
    const list = [...map.values()]
    for (const p of list) {
      if (groupBy === 'anio') {
        p.label = String(p.year)
        p.short = String(p.year)
      } else {
        const yy = String(p.year).slice(-2)
        p.label = p.month === 0 ? `${t('bills.annual')} ${p.year}` : `${t(`months.${p.month}`)} ${p.year}`
        p.short = p.month === 0 ? `${t('bills.annual')} ${yy}` : `${p.month}/${yy}`
      }
    }
    return list.sort((a, b) =>
      groupBy === 'anio' ? a.year - b.year : periodSortValue(a.year, a.month) - periodSortValue(b.year, b.month)
    )
  }, [scopedBills, groupBy, t])

  const changes = useMemo<PeriodChange[]>(() => {
    const list: PeriodChange[] = []
    for (let i = 1; i < periods.length; i++) {
      const prev = periods[i - 1]
      const cur = periods[i]
      const delta = cur.amount - prev.amount
      if (Math.abs(delta) < 0.005) continue
      list.push({
        ...cur,
        delta,
        pct: prev.amount !== 0 ? (delta / prev.amount) * 100 : null,
        previousAmount: prev.amount,
        previousLabel: prev.label,
      })
    }
    return list
  }, [periods])

  const summary = useMemo(() => {
    const total = scopedBills.reduce((s, b) => s + b.amount, 0)
    const paid = scopedBills.filter((b) => b.status === 'paid').reduce((s, b) => s + b.amount, 0)
    const pending = scopedBills.filter((b) => b.status !== 'paid').reduce((s, b) => s + b.amount, 0)
    const average = periods.length > 0 ? total / periods.length : 0
    const increases = changes.filter((c) => c.delta > 0)
    const decreases = changes.filter((c) => c.delta < 0)
    const biggestIncrease = increases.sort((a, b) => b.delta - a.delta)[0] || null
    const biggestDecrease = decreases.sort((a, b) => a.delta - b.delta)[0] || null
    const minPeriod = periods.reduce<Period | null>((m, p) => (m === null || p.amount < m.amount ? p : m), null)
    const maxPeriod = periods.reduce<Period | null>((m, p) => (m === null || p.amount > m.amount ? p : m), null)
    return { total, paid, pending, average, count: scopedBills.length, biggestIncrease, biggestDecrease, minPeriod, maxPeriod }
  }, [scopedBills, periods, changes])

  const latestChangeByKey = useMemo(() => {
    const map = new Map<string, PeriodChange>()
    for (const c of changes) map.set(c.key, c)
    return map
  }, [changes])

  if (loading) return <LoadingSpinner text={t('app.loading')} />

  if (bills.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="bill" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('bills.analysis_empty')}</h3>
        <p className="text-text-secondary">{t('bills.analysis_empty_subtitle')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <div className="w-44">
            <Select options={yearOptions} value={yearFilter} onChange={(v) => setYearFilter(String(v))} />
          </div>
          <div className="flex bg-card rounded-ios shadow-ios p-1">
            {(['periodo', 'anio'] as GroupBy[]).map((g) => (
              <button
                key={g}
                onClick={() => setGroupBy(g)}
                className={`px-3 py-1.5 rounded-ios-sm text-sm font-medium transition-colors min-h-[36px] ${
                  groupBy === g ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
                }`}
              >
                {t(`bills.analysis_group_${g}`)}
              </button>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-2 text-sm text-text-secondary">
          <Icon name="chart" className="w-4 h-4" />
          {t('bills.analysis_changes_count', String(changes.length))}
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
        <SummaryCard label={t('bills.analysis_total')} value={`${currencySymbol || ''} ${formatMoney(summary.total)}`} color="text-text" />
        <SummaryCard label={t('bills.analysis_paid')} value={`${currencySymbol || ''} ${formatMoney(summary.paid)}`} color="text-emerald-600 dark:text-emerald-400" />
        <SummaryCard label={t('bills.analysis_pending')} value={`${currencySymbol || ''} ${formatMoney(summary.pending)}`} color="text-amber-500 dark:text-amber-400" />
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 sm:gap-4">
        <MiniStat label={t('bills.analysis_bills_count')} value={String(summary.count)} />
        <MiniStat label={t('bills.analysis_average')} value={`${currencySymbol || ''} ${formatMoney(summary.average)}`} />
        <MiniStat label={t('bills.analysis_periods')} value={String(periods.length)} />
        <MiniStat label={t('bills.analysis_last_change')} value={changes.length > 0 ? formatPct(changes[changes.length - 1].pct) : '—'} />
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
        <div className="mb-3">
          <h3 className="font-semibold">{t('bills.analysis_evolution')}</h3>
          <p className="text-xs text-text-secondary">{t('bills.analysis_evolution_subtitle')}</p>
        </div>
        <EvolutionChart
          periods={periods}
          changeKeys={latestChangeByKey}
          currencySymbol={currencySymbol || ''}
          formatMoney={formatMoney}
          emptyLabel={t('bills.analysis_no_changes')}
        />
      </div>

      {changes.length > 0 && (
        <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
          <h3 className="font-semibold mb-1">{t('bills.analysis_changes_title')}</h3>
          <p className="text-xs text-text-secondary mb-4">{t('bills.analysis_changes_subtitle')}</p>
          <div className="space-y-2">
            {[...changes].reverse().map((c) => {
              const up = c.delta > 0
              const color = up ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'
              const bg = up ? 'bg-red-500/10' : 'bg-emerald-500/10'
              const target = c.bills[c.bills.length - 1]
              return (
                <button
                  key={c.key}
                  onClick={() => navigate(`/bills/edit/${target.id}?service=${serviceId}`)}
                  className="w-full flex items-center justify-between gap-3 p-3 rounded-ios-sm border border-border hover:border-primary/50 transition-colors text-left min-h-[44px]"
                >
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium truncate">{c.label}</p>
                    <p className="text-xs text-text-secondary truncate">
                      {c.previousLabel}: {currencySymbol} {formatMoney(c.previousAmount)}
                    </p>
                  </div>
                  <div className="text-right shrink-0">
                    <p className="text-sm font-semibold">{currencySymbol} {formatMoney(c.amount)}</p>
                    <span className={`inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full ${bg} ${color}`}>
                      {up ? '↑' : '↓'} {currencySymbol} {formatMoney(Math.abs(c.delta))} · {formatPct(c.pct)}
                    </span>
                  </div>
                </button>
              )
            })}
          </div>
        </div>
      )}

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
        <div className="mb-3">
          <h3 className="font-semibold">{t('bills.analysis_by_period')}</h3>
          <p className="text-xs text-text-secondary">{t('bills.analysis_by_period_subtitle')}</p>
        </div>
        <BarChart periods={periods} currencySymbol={currencySymbol || ''} formatMoney={formatMoney} />
      </div>

      {(summary.biggestIncrease || summary.biggestDecrease) && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
          <StatChange
            label={t('bills.analysis_biggest_increase')}
            periodLabel={summary.biggestIncrease?.label || '—'}
            amount={summary.biggestIncrease ? summary.biggestIncrease.delta : 0}
            pct={summary.biggestIncrease?.pct ?? null}
            currencySymbol={currencySymbol || ''}
            formatMoney={formatMoney}
            up
          />
          <StatChange
            label={t('bills.analysis_biggest_decrease')}
            periodLabel={summary.biggestDecrease?.label || '—'}
            amount={summary.biggestDecrease ? Math.abs(summary.biggestDecrease.delta) : 0}
            pct={summary.biggestDecrease?.pct ?? null}
            currencySymbol={currencySymbol || ''}
            formatMoney={formatMoney}
            up={false}
          />
        </div>
      )}
    </div>
  )
}

function SummaryCard({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-3 sm:p-4">
      <p className="text-text-secondary text-xs font-medium uppercase tracking-wide">{label}</p>
      <p className={`mt-1 text-lg sm:text-xl font-bold break-words ${color}`}>{value}</p>
    </div>
  )
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-3">
      <p className="text-text-secondary text-xs font-medium">{label}</p>
      <p className="mt-0.5 text-sm font-semibold break-words">{value}</p>
    </div>
  )
}

function StatChange({
  label,
  periodLabel,
  amount,
  pct,
  currencySymbol,
  formatMoney,
  up,
}: {
  label: string
  periodLabel: string
  amount: number
  pct: number | null
  currencySymbol: string
  formatMoney: (v: number, symbol?: string) => string
  up: boolean
}) {
  const color = up ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'
  return (
    <div className="bg-card rounded-ios shadow-ios p-4">
      <p className="text-text-secondary text-xs font-medium uppercase tracking-wide">{label}</p>
      <p className="text-sm mt-1 truncate">{periodLabel}</p>
      <p className={`text-lg font-bold mt-0.5 ${color}`}>
        {up ? '↑' : '↓'} {currencySymbol} {formatMoney(amount)}
        <span className="text-sm font-normal text-text-secondary"> ({formatPct(pct)})</span>
      </p>
    </div>
  )
}

function EvolutionChart({
  periods,
  changeKeys,
  currencySymbol,
  formatMoney,
  emptyLabel,
}: {
  periods: Period[]
  changeKeys: Map<string, PeriodChange>
  currencySymbol: string
  formatMoney: (v: number, symbol?: string) => string
  emptyLabel: string
}) {
  if (periods.length === 0) {
    return <p className="text-sm text-text-secondary">{emptyLabel}</p>
  }
  const W = 640
  const H = 240
  const padL = 12
  const padR = 12
  const padT = 24
  const padB = 34
  const max = Math.max(...periods.map((p) => p.amount), 1)
  const n = periods.length
  const x = (i: number) => (n === 1 ? W / 2 : padL + (i * (W - padL - padR)) / (n - 1))
  const y = (v: number) => H - padB - (v / max) * (H - padT - padB)

  const linePoints = periods.map((p, i) => `${x(i)},${y(p.amount)}`).join(' ')
  const areaPath = `M ${x(0)} ${H - padB} L ${periods.map((p, i) => `${x(i)} ${y(p.amount)}`).join(' L ')} L ${x(n - 1)} ${H - padB} Z`

  const labelEvery = Math.max(1, Math.ceil(n / 8))

  return (
    <div className="w-full overflow-x-auto">
      <svg viewBox={`0 0 ${W} ${H}`} className="block w-full min-w-[420px]" preserveAspectRatio="xMidYMid meet">
        <defs>
          <linearGradient id="billArea" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={NEUTRAL_COLOR} stopOpacity="0.28" />
            <stop offset="100%" stopColor={NEUTRAL_COLOR} stopOpacity="0.02" />
          </linearGradient>
        </defs>
        {[0.25, 0.5, 0.75, 1].map((f) => (
          <line
            key={f}
            x1={padL}
            x2={W - padR}
            y1={y(max * f)}
            y2={y(max * f)}
            stroke="var(--color-border, #e5e5ea)"
            strokeWidth="1"
            strokeDasharray="3 4"
          />
        ))}
        <path d={areaPath} fill="url(#billArea)" />
        <polyline points={linePoints} fill="none" stroke={NEUTRAL_COLOR} strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />
        {periods.map((p, i) => {
          const change = changeKeys.get(p.key)
          const color = change ? (change.delta > 0 ? INCREASE_COLOR : DECREASE_COLOR) : NEUTRAL_COLOR
          return (
            <g key={p.key}>
              <circle cx={x(i)} cy={y(p.amount)} r={change ? 5 : 3.5} fill={color} stroke="var(--color-card, #fff)" strokeWidth="2" />
              {change && (
                <text x={x(i)} y={y(p.amount) - 12} textAnchor="middle" fontSize="10" fontWeight="700" fill={color}>
                  {change.delta > 0 ? '+' : '−'}
                  {formatMoney(Math.abs(change.delta), currencySymbol)}
                </text>
              )}
              {i % labelEvery === 0 && (
                <text x={x(i)} y={H - 12} textAnchor="middle" fontSize="10" fill="var(--color-text-secondary, #8e8e93)">
                  {p.short}
                </text>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}

function BarChart({
  periods,
  currencySymbol,
  formatMoney,
}: {
  periods: Period[]
  currencySymbol: string
  formatMoney: (v: number, symbol?: string) => string
}) {
  if (periods.length === 0) return null
  const max = Math.max(...periods.map((p) => p.amount), 1)
  return (
    <div className="flex items-end gap-2 sm:gap-3 h-40 overflow-x-auto pb-1">
      {periods.map((p) => {
        const height = Math.max(4, (p.amount / max) * 100)
        const isMax = p.amount === max
        return (
          <div key={p.key} className="flex-1 min-w-[36px] flex flex-col items-center justify-end h-full">
            <span className="text-[10px] text-text-secondary mb-1 whitespace-nowrap">{formatMoney(p.amount, currencySymbol)}</span>
            <div
              className={`w-full rounded-t-ios-sm transition-all ${isMax ? 'bg-primary' : 'bg-primary/40'}`}
              style={{ height: `${height}%` }}
              title={`${p.label}: ${currencySymbol} ${formatMoney(p.amount)}`}
            />
            <span className="text-[10px] text-text-secondary mt-1 whitespace-nowrap">{p.short}</span>
          </div>
        )
      })}
    </div>
  )
}