import { useMemo, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import { aggregateDebtsByEndMonth, debtEndMonth } from '../utils/debtMonthlyAggregation'
import type { Debt } from '../types'

type Metric = 'count' | 'total'

const BAR_COLOR = 'rgb(var(--color-primary))'
const BAR_MUTED = 'rgba(var(--color-primary) / 0.45)'
const TEXT_COLOR = 'rgb(var(--color-text))'
const TEXT_SECONDARY = 'rgb(var(--color-text-secondary))'
const BORDER_COLOR = 'rgb(var(--color-border))'
const CARD_COLOR = 'rgb(var(--color-card))'

function formatCount(v: number) {
  return new Intl.NumberFormat('es-NI').format(v)
}

export default function DebtChart({ debts }: { debts: Debt[] }) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const [metric, setMetric] = useState<Metric>('count')
  const [hovered, setHovered] = useState<number | null>(null)

  const buckets = useMemo(
    () =>
      aggregateDebtsByEndMonth(debts, debtEndMonth, (m) => t(`months.${m}`).slice(0, 3)),
    [debts, t]
  )

  if (buckets.length === 0) {
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

  const max = metric === 'count' ? Math.max(...buckets.map((b) => b.count)) : Math.max(...buckets.map((b) => b.total))
  const safeMax = Math.max(max, 1)

  const W = 720
  const H = 260
  const padL = 48
  const padR = 16
  const padT = 28
  const padB = 44
  const n = buckets.length
  const plotW = W - padL - padR
  const slot = plotW / n
  const barW = Math.max(10, Math.min(48, slot * 0.55))
  const x = (i: number) => padL + i * slot + (slot - barW) / 2
  const y = (v: number) => H - padB - (v / safeMax) * (H - padT - padB)

  const gridTicks = [0, 0.25, 0.5, 0.75, 1]
  const valueOf = (b: (typeof buckets)[number]) => (metric === 'count' ? b.count : b.total)

  return (
    <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <p className="text-sm font-semibold">{t('deudas.chart_title')}</p>
        <div className="flex items-center gap-1 bg-bg rounded-ios p-1">
          <button
            onClick={() => setMetric('count')}
            aria-pressed={metric === 'count'}
            className={`px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] ${
              metric === 'count' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
            }`}
          >
            {t('deudas.chart_metric_count')}
          </button>
          <button
            onClick={() => setMetric('total')}
            aria-pressed={metric === 'total'}
            className={`px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] ${
              metric === 'total' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
            }`}
          >
            {t('deudas.chart_metric_total')}
          </button>
        </div>
      </div>

      <div className="relative">
        <svg viewBox={`0 0 ${W} ${H}`} className="block w-full" preserveAspectRatio="xMidYMid meet" role="img" aria-label={t('deudas.chart_title')}>
          <defs>
            <clipPath id="debtChartPlot">
              <rect x={padL} y={padT} width={plotW} height={H - padT - padB} />
            </clipPath>
          </defs>

          {gridTicks.map((f) => {
            const gy = y(safeMax * f)
            const label = metric === 'count' ? formatCount(Math.round(safeMax * f)) : formatMoney(safeMax * f)
            return (
              <g key={f}>
                <line x1={padL} x2={W - padR} y1={gy} y2={gy} stroke={BORDER_COLOR} strokeWidth="1" strokeDasharray="3 4" />
                <text x={padL - 8} y={gy + 3} textAnchor="end" fontSize="10" fill={TEXT_SECONDARY}>
                  {label}
                </text>
              </g>
            )
          })}

          {buckets.map((b, i) => {
            const v = valueOf(b)
            const bh = Math.max(2, y(0) - y(v))
            const isMax = v === safeMax
            return (
              <g key={`${b.year}-${b.month}`}>
                <rect
                  x={x(i)}
                  y={y(v)}
                  width={barW}
                  height={bh}
                  rx={4}
                  fill={isMax ? BAR_COLOR : BAR_MUTED}
                />
                <text
                  x={x(i) + barW / 2}
                  y={y(v) - 6}
                  textAnchor="middle"
                  fontSize="10"
                  fontWeight="600"
                  fill={TEXT_COLOR}
                >
                  {metric === 'count' ? formatCount(v) : formatMoney(v)}
                </text>
                <text
                  x={x(i) + barW / 2}
                  y={H - padB + 20}
                  textAnchor="middle"
                  fontSize="11"
                  fill={TEXT_SECONDARY}
                >
                  {b.label}
                </text>
                <rect
                  x={padL + i * slot}
                  y={padT}
                  width={slot}
                  height={H - padT - padB}
                  fill="transparent"
                  onMouseEnter={() => setHovered(i)}
                  onMouseLeave={() => setHovered(null)}
                />
              </g>
            )
          })}
        </svg>

        {hovered !== null && buckets[hovered] && (
          <div
            className="absolute z-10 pointer-events-none bg-card border border-border rounded-ios shadow-ios-lg p-3 w-56"
            style={{
              left: `${Math.min(Math.max((x(hovered) + barW / 2) / W * 100, 28), 72)}%`,
              top: `${y(valueOf(buckets[hovered])) / H * 100}%`,
              transform: 'translate(-50%, -105%)',
            }}
          >
            <p className="text-sm font-semibold mb-1">{buckets[hovered].label}</p>
            <p className="text-xs text-text-secondary mb-1">
              {t('deudas.chart_tooltip_count')}: {formatCount(buckets[hovered].count)}
            </p>
            <p className="text-xs text-text-secondary mb-1">
              {t('deudas.chart_tooltip_total')}: {formatMoney(buckets[hovered].total)}
            </p>
            <ul className="text-xs text-text space-y-0.5 max-h-28 overflow-y-auto">
              {buckets[hovered].debts.map((d) => (
                <li key={d.id} className="truncate">{d.description}</li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <p className="text-[10px] text-text-secondary mt-2">{t('deudas.chart_hint')}</p>
    </div>
  )
}