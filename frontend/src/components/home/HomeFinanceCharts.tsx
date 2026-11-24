import { useI18nStore } from '../../stores/i18nStore'
import { formatCurrency } from '../../utils/currency'

export interface MonthPoint {
  year: number
  month: number
  income: number
  expenses: number
}

export interface DonutSlice {
  kind: string
  label: string
  amount: number
  color: string
}

const INCOME_COLOR = 'rgb(var(--color-success))'
const EXPENSE_COLOR = 'rgb(var(--color-danger))'

function formatCompact(value: number): string {
  const abs = Math.abs(value)
  if (abs >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (abs >= 1_000) return `${(value / 1_000).toFixed(1)}k`
  return String(Math.round(value))
}

function niceMax(value: number): number {
  if (value <= 0) return 1
  const exponent = Math.floor(Math.log10(value))
  const base = Math.pow(10, exponent)
  const scaled = value / base
  const nice = scaled <= 1 ? 1 : scaled <= 2 ? 2 : scaled <= 5 ? 5 : 10
  return nice * base
}

// FinanceBars dibuja ingresos vs gastos de 12 meses en SVG (sin dependencias).
export function FinanceBars({ points, currency }: { points: MonthPoint[]; currency: string }) {
  const { t } = useI18nStore()
  const width = 720
  const height = 240
  const pad = { top: 14, right: 8, bottom: 30, left: 48 }
  const chartW = width - pad.left - pad.right
  const chartH = height - pad.top - pad.bottom
  const max = niceMax(Math.max(0, ...points.map((p) => Math.max(p.income, p.expenses))))
  const groupW = chartW / Math.max(points.length, 1)
  const barW = Math.min(16, groupW * 0.3)

  const y = (value: number) => pad.top + chartH - (value / max) * chartH
  const ticks = [0, max / 2, max]

  return (
    <div>
      <div className="flex items-center gap-4 mb-2 text-xs text-text-secondary">
        <span className="inline-flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-sm" style={{ background: INCOME_COLOR }} />
          {t('home.finances.income_legend')}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-sm" style={{ background: EXPENSE_COLOR }} />
          {t('home.finances.expenses_legend')}
        </span>
      </div>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="w-full h-auto"
        role="img"
        aria-label={t('home.finances.series_title')}
      >
        {ticks.map((tick) => (
          <g key={tick}>
            <line
              x1={pad.left}
              x2={width - pad.right}
              y1={y(tick)}
              y2={y(tick)}
              stroke="rgb(var(--color-border))"
              strokeWidth={1}
            />
            <text
              x={pad.left - 6}
              y={y(tick) + 3}
              textAnchor="end"
              fontSize={9}
              fill="rgb(var(--color-text-secondary))"
            >
              {formatCompact(tick)}
            </text>
          </g>
        ))}

        {points.map((point, index) => {
          const center = pad.left + index * groupW + groupW / 2
          const incomeH = Math.max(0, (point.income / max) * chartH)
          const expenseH = Math.max(0, (point.expenses / max) * chartH)
          const label = t(`months.${point.month}`).slice(0, 3)
          return (
            <g key={`${point.year}-${point.month}`}>
              <rect
                x={center - barW - 2}
                y={pad.top + chartH - incomeH}
                width={barW}
                height={incomeH}
                rx={2}
                fill={INCOME_COLOR}
              >
                <title>{`${label} ${point.year} · ${t('home.finances.income_legend')}: ${currency} ${formatCurrency(point.income)}`}</title>
              </rect>
              <rect
                x={center + 2}
                y={pad.top + chartH - expenseH}
                width={barW}
                height={expenseH}
                rx={2}
                fill={EXPENSE_COLOR}
              >
                <title>{`${label} ${point.year} · ${t('home.finances.expenses_legend')}: ${currency} ${formatCurrency(point.expenses)}`}</title>
              </rect>
              <text
                x={center}
                y={height - 10}
                textAnchor="middle"
                fontSize={9}
                fill="rgb(var(--color-text-secondary))"
              >
                {label}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

// FinanceDonut muestra la composición de gastos del mes en SVG.
export function FinanceDonut({ slices, currency }: { slices: DonutSlice[]; currency: string }) {
  const { t } = useI18nStore()
  const visible = slices.filter((slice) => slice.amount > 0)
  const total = visible.reduce((sum, slice) => sum + slice.amount, 0)

  if (total <= 0) {
    return (
      <div className="h-40 flex items-center justify-center text-sm text-text-secondary">
        {t('home.finances.no_expenses')}
      </div>
    )
  }

  const radius = 52
  const circumference = 2 * Math.PI * radius
  let offset = 0

  return (
    <div className="flex flex-col sm:flex-row items-center gap-4">
      <svg viewBox="0 0 140 140" className="w-36 h-36 shrink-0" role="img" aria-label={t('home.finances.breakdown_title')}>
        <circle cx={70} cy={70} r={radius} fill="none" stroke="rgb(var(--color-border))" strokeWidth={18} opacity={0.4} />
        {visible.map((slice) => {
          const dash = (slice.amount / total) * circumference
          const circle = (
            <circle
              key={slice.kind}
              cx={70}
              cy={70}
              r={radius}
              fill="none"
              stroke={slice.color}
              strokeWidth={18}
              strokeDasharray={`${dash} ${circumference - dash}`}
              strokeDashoffset={-offset}
              transform="rotate(-90 70 70)"
            >
              <title>{`${slice.label}: ${currency} ${formatCurrency(slice.amount)}`}</title>
            </circle>
          )
          offset += dash
          return circle
        })}
        <text x={70} y={66} textAnchor="middle" fontSize={10} fill="rgb(var(--color-text-secondary))">
          {currency}
        </text>
        <text x={70} y={82} textAnchor="middle" fontSize={13} fontWeight={600} fill="rgb(var(--color-text))">
          {formatCompact(total)}
        </text>
      </svg>

      <ul className="flex-1 space-y-1.5 w-full">
        {visible.map((slice) => (
          <li key={slice.kind} className="flex items-center justify-between gap-3 text-sm">
            <span className="inline-flex items-center gap-2 min-w-0">
              <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ background: slice.color }} />
              <span className="truncate">{slice.label}</span>
            </span>
            <span className="tabular-nums shrink-0">
              {currency} {formatCurrency(slice.amount)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}
