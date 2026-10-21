import { useEffect, useMemo, useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import LoadingSpinner from './LoadingSpinner'
import Select from './Select'
import type { DebtHistoryCurrency, DebtHistoryResponse } from '../types'

const ACCENT = 'rgb(var(--color-primary))'
const TEXT_COLOR = 'rgb(var(--color-text))'
const TEXT_SECONDARY = 'rgb(var(--color-text-secondary))'
const BORDER_COLOR = 'rgb(var(--color-border))'
const DESIGN_W = 720
const CHART_H = 260
const TIP_MAX_W = 240

function monthShort(t: (k: string) => string, month: number) {
  return t(`months.${month}`).slice(0, 3)
}

function monthIndex(month: string) {
  return Number(month.slice(5, 7))
}

function monthLabel(t: (k: string) => string, month: string, withYear: boolean) {
  const idx = monthIndex(month)
  const name = t(`months.${idx}`)
  return withYear ? `${name} ${month.slice(0, 4)}` : name
}

export default function DebtHistoryChart() {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)

  const [data, setData] = useState<DebtHistoryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [currencyCode, setCurrencyCode] = useState('')
  const [tip, setTip] = useState<{ left: number; top: number; html: string } | null>(null)

  const load = () => {
    setLoading(true)
    setError(false)
    api.debts
      .history(12)
      .then((res) => setData(res))
      .catch(() => setError(true))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [])

  const currencies = data?.currencies ?? []

  const defaultCurrency = useMemo(() => {
    let best = ''
    let bestAmount = -1
    for (const c of currencies) {
      if (c.summary.pending_amount_current > bestAmount) {
        best = c.code
        bestAmount = c.summary.pending_amount_current
      }
    }
    return best || currencies[0]?.code || ''
  }, [currencies])

  const effectiveCurrency = currencies.some((c) => c.code === currencyCode) ? currencyCode : defaultCurrency
  const selected: DebtHistoryCurrency | undefined = currencies.find((c) => c.code === effectiveCurrency)

  const currencyOptions = currencies.map((c) => ({ value: c.code, label: c.code }))

  if (loading) return <LoadingSpinner text={t('app.loading')} />

  if (error) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="bar" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.history_error')}</h3>
        <button
          onClick={load}
          className="mt-4 inline-flex items-center justify-center px-5 py-2.5 bg-primary text-white rounded-ios-sm text-sm font-medium min-h-[44px]"
        >
          {t('deudas.history_retry')}
        </button>
      </div>
    )
  }

  if (!selected) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="bar" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.chart_empty')}</h3>
        <p className="text-text-secondary">{t('deudas.chart_empty_subtitle')}</p>
      </div>
    )
  }

  const { series, summary } = selected
  const currentMonth = series[series.length - 1]?.month ?? ''
  const maxOpen = Math.max(1, ...series.map((p) => p.open_count))
  const step = Math.max(1, Math.ceil(maxOpen / 4))
  const yTop = Math.ceil(maxOpen / step) * step
  const avg = series.reduce((sum, p) => sum + p.open_count, 0) / Math.max(1, series.length)

  const W = DESIGN_W
  const H = CHART_H
  const L = 34
  const R = 10
  const TOP = 14
  const BOTTOM = 34
  const plotW = W - L - R
  const plotH = H - TOP - BOTTOM
  const slot = plotW / series.length
  const barW = Math.max(8, slot * 0.58)
  const Y = (v: number) => TOP + plotH - (v / yTop) * plotH

  const ticks: number[] = []
  for (let v = 0; v <= yTop; v += step) ticks.push(v)

  const showTip = (e: React.MouseEvent, html: string) => {
    const left = Math.min(Math.max(e.clientX + 14, 8), window.innerWidth - TIP_MAX_W - 8)
    const top = Math.min(Math.max(e.clientY + 14, 8), window.innerHeight - 96)
    setTip({ left, top, html })
  }

  const pointTooltip = (month: string, open: number, closed: number) => {
    const idx = monthIndex(month)
    const label = `${t(`months.${idx}`)} ${month.slice(0, 4)}`
    return `<b>${label}</b><br>${t('deudas.history_open_debts')}: <b>${open}</b><br>${t('deudas.history_closed_debts')}: <b>${closed}</b>`
  }

  const changeLabel = `${summary.change_count > 0 ? '+' : ''}${summary.change_count}`
  const changeClass =
    summary.change_count < 0 ? 'text-success' : summary.change_count > 0 ? 'text-danger' : 'text-text'
  const pctLabel =
    summary.change_pct == null
      ? '—'
      : `${summary.change_pct > 0 ? '+' : ''}${summary.change_pct.toFixed(1)}%`

  const cards = [
    { label: t('deudas.history_stat_start'), value: String(summary.start_count) },
    { label: t('deudas.history_stat_current'), value: String(summary.end_count) },
    { label: t('deudas.history_stat_change'), value: changeLabel, sub: pctLabel, valueClass: changeClass },
    { label: t('deudas.history_stat_closed'), value: String(summary.closed_count) },
    {
      label: t('deudas.history_stat_pending'),
      value: formatMoney(summary.pending_amount_current, selected.symbol || selected.code),
      sub: t('deudas.history_pending_hint'),
    },
  ]

  return (
    <div className="space-y-3 sm:space-y-4">
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
        {cards.map((card) => (
          <div key={card.label} className="bg-card rounded-ios shadow-ios p-4">
            <span className="text-xs text-text-secondary">{card.label}</span>
            <b className={`block text-2xl font-bold mt-1 ${card.valueClass ?? ''}`}>{card.value}</b>
            {card.sub && <span className="block text-[11px] text-text-secondary mt-1 leading-tight">{card.sub}</span>}
          </div>
        ))}
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <p className="text-sm font-semibold">{t('deudas.chart_title_history')}</p>
          {currencyOptions.length > 1 && (
            <div className="w-28">
              <Select options={currencyOptions} value={effectiveCurrency} onChange={(v) => setCurrencyCode(String(v))} />
            </div>
          )}
        </div>

        <div className="overflow-x-auto">
          <svg
            viewBox={`0 0 ${W} ${H}`}
            className="block w-full"
            style={{ minWidth: `${Math.max(360, series.length * 44)}px` }}
            preserveAspectRatio="xMidYMid meet"
            role="img"
            aria-label={t('deudas.chart_title_history')}
          >
            {ticks.map((v) => (
              <g key={v}>
                <line x1={L} x2={L + plotW} y1={Y(v)} y2={Y(v)} stroke={BORDER_COLOR} strokeDasharray="3 4" />
                <text className="m" x={L - 6} y={Y(v) + 4} fontSize="11" textAnchor="end" fill={TEXT_SECONDARY}>
                  {v}
                </text>
              </g>
            ))}

            {avg > 0 && (
              <line
                x1={L}
                x2={L + plotW}
                y1={Y(avg)}
                y2={Y(avg)}
                stroke={TEXT_SECONDARY}
                strokeDasharray="5 4"
                strokeWidth="1"
              />
            )}

            {series.map((p, i) => {
              const isCurrent = p.month === currentMonth
              const x = L + i * slot + (slot - barW) / 2
              const y = Y(p.open_count)
              const h = Math.max(1, Y(0) - y)
              return (
                <g key={p.month}>
                  <rect
                    x={x}
                    y={TOP}
                    width={barW}
                    height={plotH}
                    fill="transparent"
                    onMouseMove={(e) => showTip(e, pointTooltip(p.month, p.open_count, p.closed_count))}
                    onMouseLeave={() => setTip(null)}
                  />
                  <rect
                    x={x}
                    y={p.open_count > 0 ? y : Y(0) - 1}
                    width={barW}
                    height={p.open_count > 0 ? h : 1}
                    rx={4}
                    fill={ACCENT}
                    opacity={isCurrent ? 1 : 0.35}
                    pointerEvents="none"
                  />
                  <text
                    className="m"
                    x={x + barW / 2}
                    y={H - 18}
                    fontSize="11"
                    textAnchor="middle"
                    fill={isCurrent ? TEXT_COLOR : TEXT_SECONDARY}
                  >
                    {monthShort(t, monthIndex(p.month))}
                  </text>
                  {(i === 0 || monthIndex(p.month) === 1) && (
                    <text
                      className="m"
                      x={x + barW / 2}
                      y={H - 6}
                      fontSize="11"
                      textAnchor="middle"
                      fill={TEXT_SECONDARY}
                    >
                      {p.month.slice(0, 4)}
                    </text>
                  )}
                </g>
              )
            })}
          </svg>
        </div>

        <p className="text-xs text-text-secondary mt-2">{t('deudas.chart_note_history')}</p>
      </div>

      <details className="bg-card rounded-ios shadow-ios">
        <summary className="px-4 py-3 cursor-pointer text-sm font-semibold min-h-[44px] flex items-center">
          {t('deudas.history_table_title')}
        </summary>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-text-secondary border-t border-border">
                <th className="px-4 py-2 font-medium">{t('deudas.history_month')}</th>
                <th className="px-4 py-2 font-medium text-right">{t('deudas.history_open_debts')}</th>
                <th className="px-4 py-2 font-medium text-right">{t('deudas.history_closed_debts')}</th>
              </tr>
            </thead>
            <tbody>
              {series.map((p) => (
                <tr key={p.month} className="border-t border-border">
                  <td className="px-4 py-2">{monthLabel(t, p.month, true)}</td>
                  <td className="px-4 py-2 text-right">{p.open_count}</td>
                  <td className="px-4 py-2 text-right">{p.closed_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>

      {tip && (
        <div
          className="fixed pointer-events-none bg-text text-bg px-2.5 py-2 rounded-ios-sm text-xs leading-relaxed z-[60] max-w-60"
          style={{ left: tip.left, top: tip.top, maxWidth: TIP_MAX_W }}
          dangerouslySetInnerHTML={{ __html: tip.html }}
        />
      )}
    </div>
  )
}
