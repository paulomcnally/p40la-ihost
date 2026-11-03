import { useEffect, useMemo, useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import LoadingSpinner from './LoadingSpinner'
import Select from './Select'
import { addMonths, periodKey, periodRange } from '../utils/serviceBills'
import type { ServiceBillSummaryResponse, ServiceProjectionResponse } from '../types'

type ChartView = 'history' | 'projection'

interface ServiceChartProps {
  homeId: number | null
  year: number
  month: number
  gview: ChartView
  onChangeGview: (view: ChartView) => void
}

const COLORS = {
  paid: '#30d158',
  pending: '#ff9f0a',
  confirmed: '#0a84ff',
  estimated: '#64d2ff',
}

function monthShort(t: (key: string) => string, month: number) {
  return t(`months.${month}`).slice(0, 3)
}

function BarsChart({
  periods,
  max,
  series,
  formatValue,
  emptyLabel,
}: {
  periods: { key: string; label: string; short: string }[]
  max: number
  series: { key: string; values: number[]; color: string; dashed?: boolean }[]
  formatValue: (value: number) => string
  emptyLabel: string
}) {
  const W = 720
  const H = 260
  const padL = 10
  const padR = 10
  const padT = 18
  const padB = 34
  const n = periods.length
  const slot = n > 0 ? (W - padL - padR) / n : 0
  const barW = Math.max(8, Math.min(30, slot * 0.55))
  const y = (value: number) => H - padB - (value / max) * (H - padT - padB)
  const labelEvery = Math.max(1, Math.ceil(n / 8))

  if (n === 0) return <p className="text-sm text-text-secondary">{emptyLabel}</p>

  return (
    <div className="w-full overflow-x-auto">
      <svg viewBox={`0 0 ${W} ${H}`} className="block w-full min-w-[560px]" preserveAspectRatio="xMidYMid meet" role="img" aria-label={emptyLabel}>
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
        {periods.map((period, i) => {
          let acc = 0
          const cx = padL + slot * i + slot / 2
          return (
            <g key={period.key}>
              {series.map((s) => {
                const value = s.values[i] || 0
                const height = (value / max) * (H - padT - padB)
                const yTop = y(acc + value)
                acc += value
                if (value <= 0) return null
                return (
                  <rect
                    key={s.key}
                    x={cx - barW / 2}
                    y={yTop}
                    width={barW}
                    height={Math.max(1, height)}
                    rx={4}
                    fill={s.color}
                    opacity={s.dashed ? 0.65 : 1}
                  >
                    <title>{`${period.label} · ${s.key}: ${formatValue(value)}`}</title>
                  </rect>
                )
              })}
              {i % labelEvery === 0 && (
                <text x={cx} y={H - 12} textAnchor="middle" fontSize="10" fill="var(--color-text-secondary, #8e8e93)">
                  {period.short}
                </text>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}

export default function ServiceChart({ homeId, year, month, gview, onChangeGview }: ServiceChartProps) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const [summary, setSummary] = useState<ServiceBillSummaryResponse | null>(null)
  const [projection, setProjection] = useState<ServiceProjectionResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [currencyFilter, setCurrencyFilter] = useState('')
  const [error, setError] = useState(false)

  const current = useMemo(() => ({ year, month }), [year, month])
  const historyPeriods = useMemo(() => periodRange(current, 12), [current])
  const projectionPeriods = useMemo(() => periodRange(addMonths(current, 11), 12), [current])

  useEffect(() => {
    let active = true
    setLoading(true)
    setError(false)
    const historyFrom = periodKey(historyPeriods[0].year, historyPeriods[0].month)
    const historyTo = periodKey(current.year, current.month)
    const projectionFrom = periodKey(current.year, current.month)
    Promise.all([
      api.serviceBills.summary(historyFrom, historyTo, homeId),
      api.serviceBills.projection(projectionFrom, 12, homeId),
    ])
      .then(([summaryRes, projectionRes]) => {
        if (!active) return
        setSummary(summaryRes)
        setProjection(projectionRes)
      })
      .catch(() => {
        if (active) setError(true)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [homeId, current, historyPeriods])

  const summaryRows = summary?.periods || []
  const projectionRows = projection?.periods || []

  const currencies = useMemo(() => {
    const codes = new Set<string>()
    for (const row of summaryRows) if (row.currency_code) codes.add(row.currency_code)
    for (const row of projectionRows) if (row.currency_code) codes.add(row.currency_code)
    return [...codes].sort()
  }, [summaryRows, projectionRows])

  const defaultCurrency = useMemo(() => {
    const sums = new Map<string, number>()
    if (gview === 'projection') {
      for (const row of projectionRows) {
        sums.set(row.currency_code, (sums.get(row.currency_code) || 0) + row.estimated)
      }
      // Si no hay estimados, priorizar la moneda con más confirmado.
      if ([...sums.values()].every((value) => value === 0)) {
        for (const row of projectionRows) {
          sums.set(row.currency_code, (sums.get(row.currency_code) || 0) + row.total)
        }
      }
    } else {
      for (const row of summaryRows) {
        sums.set(row.currency_code, (sums.get(row.currency_code) || 0) + row.total)
      }
    }
    let best = ''
    let bestSum = -1
    for (const [code, sum] of sums) {
      if (sum > bestSum) {
        best = code
        bestSum = sum
      }
    }
    return best || currencies[0] || ''
  }, [gview, summaryRows, projectionRows, currencies])

  const effectiveCurrency = currencyFilter && currencies.includes(currencyFilter)
    ? currencyFilter
    : defaultCurrency

  const currencySymbol = useMemo(() => {
    const row = summaryRows.find((r) => r.currency_code === effectiveCurrency)
      || projectionRows.find((r) => r.currency_code === effectiveCurrency)
    return row?.currency_symbol || effectiveCurrency
  }, [summaryRows, projectionRows, effectiveCurrency])

  const historyData = useMemo(() => {
    const map = new Map<string, { total: number; paid: number; pending: number; count: number }>()
    for (const row of summaryRows) {
      if (row.currency_code !== effectiveCurrency) continue
      const entry = map.get(row.period) || { total: 0, paid: 0, pending: 0, count: 0 }
      entry.total += row.total
      entry.paid += row.paid
      entry.pending += row.pending
      entry.count += row.count
      map.set(row.period, entry)
    }
    return historyPeriods.map((p) => {
      const key = periodKey(p.year, p.month)
      return {
        key,
        label: `${t(`months.${p.month}`)} ${p.year}`,
        short: `${monthShort(t, p.month)} ${String(p.year).slice(-2)}`,
        ...(map.get(key) || { total: 0, paid: 0, pending: 0, count: 0 }),
      }
    })
  }, [summaryRows, historyPeriods, effectiveCurrency, t])

  const historyStats = useMemo(() => {
    const total = historyData.reduce((sum, p) => sum + p.total, 0)
    const paid = historyData.reduce((sum, p) => sum + p.paid, 0)
    const pending = historyData.reduce((sum, p) => sum + p.pending, 0)
    const withData = historyData.filter((p) => p.total > 0)
    const first = withData[0]
    const last = withData[withData.length - 1]
    const variation = first && last && first.total > 0 ? ((last.total - first.total) / first.total) * 100 : null
    const count = historyData.reduce((sum, p) => sum + p.count, 0)
    return { total, paid, pending, average: total / 12, variation, count }
  }, [historyData])

  const projectionData = useMemo(() => {
    const map = new Map<string, { confirmed: number; estimated: number; total: number }>()
    for (const row of projectionRows) {
      if (row.currency_code !== effectiveCurrency) continue
      const entry = map.get(row.period) || { confirmed: 0, estimated: 0, total: 0 }
      entry.confirmed += row.confirmed
      entry.estimated += row.estimated
      entry.total += row.total
      map.set(row.period, entry)
    }
    return projectionPeriods.map((p) => {
      const key = periodKey(p.year, p.month)
      return {
        key,
        label: `${t(`months.${p.month}`)} ${p.year}`,
        short: `${monthShort(t, p.month)} ${String(p.year).slice(-2)}`,
        ...(map.get(key) || { confirmed: 0, estimated: 0, total: 0 }),
      }
    })
  }, [projectionRows, projectionPeriods, effectiveCurrency, t])

  const historyMax = Math.max(...historyData.map((p) => p.total), 1)
  const projectionMax = Math.max(...projectionData.map((p) => p.total), 1)
  const projectionConfirmed = projectionData.reduce((sum, p) => sum + p.confirmed, 0)
  const projectionEstimated = projectionData.reduce((sum, p) => sum + p.estimated, 0)

  if (loading) return <LoadingSpinner text={t('app.loading')} />

  if (error) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="chart" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('services.chart_empty')}</h3>
        <p className="text-text-secondary">{t('services.chart_error')}</p>
      </div>
    )
  }

  if (currencies.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="chart" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('services.chart_empty')}</h3>
        <p className="text-text-secondary">{t('services.chart_empty_subtitle')}</p>
      </div>
    )
  }

  const currencyOptions = currencies.map((code) => ({ value: code, label: code }))
  const chartPeriods = gview === 'history' ? historyData : projectionData
  const series = gview === 'history'
    ? [
        { key: t('services.chart_paid'), values: historyData.map((p) => p.paid), color: COLORS.paid },
        { key: t('services.chart_pending'), values: historyData.map((p) => p.pending), color: COLORS.pending },
      ]
    : [
        { key: t('services.chart_confirmed'), values: projectionData.map((p) => p.confirmed), color: COLORS.confirmed },
        { key: t('services.chart_estimated'), values: projectionData.map((p) => p.estimated), color: COLORS.estimated, dashed: true },
      ]

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex bg-card rounded-ios shadow-ios p-1">
            {(['history', 'projection'] as ChartView[]).map((view) => (
              <button
                key={view}
                onClick={() => onChangeGview(view)}
                aria-pressed={gview === view}
                className={`px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] whitespace-nowrap ${
                  gview === view ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
                }`}
              >
                {t(`services.chart_view_${view}`)}
              </button>
            ))}
          </div>
          <div className="w-28">
            <Select options={currencyOptions} value={effectiveCurrency} onChange={(value) => setCurrencyFilter(String(value))} />
          </div>
        </div>
        <div className="flex items-center gap-2 text-sm text-text-secondary">
          <Icon name="chart" className="w-4 h-4" />
          {t(`services.chart_period_${gview}`)}
        </div>
      </div>

      {gview === 'history' ? (
        <>
          <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
            <div className="mb-3">
              <h3 className="font-semibold">{t('services.chart_title_history')}</h3>
              <p className="text-xs text-text-secondary">{t('services.chart_note_history')}</p>
            </div>
            <BarsChart
              periods={chartPeriods}
              max={historyMax}
              series={series}
              formatValue={(value) => `${effectiveCurrency} ${formatMoney(value, currencySymbol)}`}
              emptyLabel={t('services.chart_empty_subtitle')}
            />
            <div className="flex flex-wrap items-center gap-4 mt-3 text-xs text-text-secondary">
              <span className="inline-flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm" style={{ backgroundColor: COLORS.paid }} />{t('services.chart_paid')}</span>
              <span className="inline-flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm" style={{ backgroundColor: COLORS.pending }} />{t('services.chart_pending')}</span>
            </div>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 sm:gap-4">
            <MiniStat label={t('services.chart_total')} value={`${effectiveCurrency} ${formatMoney(historyStats.total, currencySymbol)}`} />
            <MiniStat label={t('services.chart_paid')} value={`${effectiveCurrency} ${formatMoney(historyStats.paid, currencySymbol)}`} />
            <MiniStat label={t('services.chart_pending')} value={`${effectiveCurrency} ${formatMoney(historyStats.pending, currencySymbol)}`} />
            <MiniStat label={t('services.chart_average')} value={`${effectiveCurrency} ${formatMoney(historyStats.average, currencySymbol)}`} />
            <MiniStat label={t('services.chart_count')} value={String(historyStats.count)} />
            <MiniStat
              label={t('services.chart_variation')}
              value={historyStats.variation === null ? '—' : `${historyStats.variation > 0 ? '+' : ''}${historyStats.variation.toFixed(1)}%`}
              color={historyStats.variation === null ? 'text-text' : historyStats.variation > 0 ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'}
            />
          </div>
        </>
      ) : (
        <>
          <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
            <div className="mb-3">
              <h3 className="font-semibold">{t('services.chart_title_projection')}</h3>
              <p className="text-xs text-text-secondary">{t('services.chart_note_projection')}</p>
            </div>
            <BarsChart
              periods={chartPeriods}
              max={projectionMax}
              series={series}
              formatValue={(value) => `${effectiveCurrency} ${formatMoney(value, currencySymbol)}`}
              emptyLabel={t('services.chart_empty_subtitle')}
            />
            <div className="flex flex-wrap items-center gap-4 mt-3 text-xs text-text-secondary">
              <span className="inline-flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm" style={{ backgroundColor: COLORS.confirmed }} />{t('services.chart_confirmed')}</span>
              <span className="inline-flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm" style={{ backgroundColor: COLORS.estimated }} />{t('services.chart_estimated')}</span>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
            <MiniStat label={t('services.chart_confirmed')} value={`${effectiveCurrency} ${formatMoney(projectionConfirmed, currencySymbol)}`} />
            <MiniStat label={t('services.chart_estimated')} value={`${effectiveCurrency} ${formatMoney(projectionEstimated, currencySymbol)}`} />
            <MiniStat label={t('services.chart_total')} value={`${effectiveCurrency} ${formatMoney(projectionConfirmed + projectionEstimated, currencySymbol)}`} />
          </div>

          <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
            <h3 className="font-semibold mb-1">{t('services.chart_table_title')}</h3>
            <p className="text-xs text-text-secondary mb-4">{t('services.chart_note_estimate')}</p>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[420px]">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-left px-3 py-2 text-xs font-semibold text-text-secondary uppercase">{t('services.chart_period')}</th>
                    <th className="text-right px-3 py-2 text-xs font-semibold text-text-secondary uppercase">{t('services.chart_confirmed')}</th>
                    <th className="text-right px-3 py-2 text-xs font-semibold text-text-secondary uppercase">{t('services.chart_estimated')}</th>
                    <th className="text-right px-3 py-2 text-xs font-semibold text-text-secondary uppercase">{t('services.chart_total')}</th>
                  </tr>
                </thead>
                <tbody>
                  {projectionData.map((p) => (
                    <tr key={p.key} className="border-b border-border last:border-b-0">
                      <td className="px-3 py-2 text-sm">{p.label}</td>
                      <td className="px-3 py-2 text-sm text-right">{formatMoney(p.confirmed, currencySymbol)}</td>
                      <td className="px-3 py-2 text-sm text-right">{formatMoney(p.estimated, currencySymbol)}</td>
                      <td className="px-3 py-2 text-sm text-right font-semibold">{formatMoney(p.total, currencySymbol)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

function MiniStat({ label, value, color = 'text-text' }: { label: string; value: string; color?: string }) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-3">
      <p className="text-text-secondary text-xs font-medium">{label}</p>
      <p className={`mt-0.5 text-sm font-semibold break-words ${color}`}>{value}</p>
    </div>
  )
}
