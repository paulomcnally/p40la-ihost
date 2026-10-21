import { useMemo, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import type { Currency, Debt } from '../types'

const ACCENT = 'rgb(var(--color-primary))'
const TEXT_COLOR = 'rgb(var(--color-text))'
const TEXT_SECONDARY = 'rgb(var(--color-text-secondary))'
const BORDER_COLOR = 'rgb(var(--color-border))'

const DESIGN_W = 720
const CHART_H = 300
const TIP_MAX_W = 240

type Item = { debt: Debt; endDate: Date }

type Bucket = {
  key: string
  label: string
  fullLabel: string
  year: number
  month: number | null
  amount: number
  count: number
  debts: string[]
}

export type ProjectionPoint = Bucket & { remaining: number }

function monthShort(t: (k: string) => string, month: number) {
  return t(`months.${month}`).slice(0, 3)
}

function monthName(t: (k: string) => string, month: number) {
  return t(`months.${month}`)
}

function pad2(value: number) {
  return String(value).padStart(2, '0')
}

function compactMoney(value: number, symbol: string) {
  if (value >= 1_000_000) return `${symbol}${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${symbol}${Math.round(value / 1_000)}k`
  return `${symbol}${Math.round(value)}`
}

function niceMax(value: number) {
  if (value <= 0) return 1
  const pow = Math.pow(10, Math.floor(Math.log10(value)))
  const n = value / pow
  const mult = n <= 1 ? 1 : n <= 2 ? 2 : n <= 5 ? 5 : 10
  return mult * pow
}

function esc(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

// SPEC-107 REQ-003: granularidad adaptativa. Mensual si la fecha de fin más
// lejana cae dentro de los próximos 12 meses (mes actual + 11); anual si no.
export function isMonthlyHorizon(items: Item[], now: Date) {
  const nowIdx = now.getFullYear() * 12 + now.getMonth()
  let maxIdx = nowIdx
  for (const item of items) {
    const idx = item.endDate.getFullYear() * 12 + item.endDate.getMonth()
    if (idx > maxIdx) maxIdx = idx
  }
  return maxIdx - nowIdx <= 11
}

// Construye la serie agrupada. Las fechas ya vencidas se imputan al primer
// período para no perder monto (SPEC-107 REQ-004).
export function buildProjection(
  items: Item[],
  now: Date,
  t: (k: string) => string,
  monthly: boolean
): ProjectionPoint[] {
  const buckets = new Map<string, Bucket>()
  const nowYear = now.getFullYear()
  const nowMonth = now.getMonth() + 1
  const nowIdx = nowYear * 12 + (nowMonth - 1)

  const ensure = (key: string, year: number, month: number | null): Bucket => {
    const existing = buckets.get(key)
    if (existing) return existing
    const bucket: Bucket = {
      key,
      label: month !== null ? `${monthShort(t, month)} ${String(year).slice(2)}` : String(year),
      fullLabel: month !== null ? `${monthName(t, month)} ${year}` : String(year),
      year,
      month,
      amount: 0,
      count: 0,
      debts: [],
    }
    buckets.set(key, bucket)
    return bucket
  }

  // Pre-crear todos los períodos del rango para que no queden huecos.
  if (monthly) {
    let maxIdx = nowIdx
    for (const item of items) {
      const idx = item.endDate.getFullYear() * 12 + item.endDate.getMonth()
      if (idx > maxIdx) maxIdx = idx
    }
    for (let idx = nowIdx; idx <= maxIdx; idx++) {
      const year = Math.floor(idx / 12)
      const month = (idx % 12) + 1
      ensure(`${year}-${pad2(month)}`, year, month)
    }
  } else {
    let maxYear = nowYear
    for (const item of items) {
      if (item.endDate.getFullYear() > maxYear) maxYear = item.endDate.getFullYear()
    }
    for (let year = nowYear; year <= maxYear; year++) ensure(String(year), year, null)
  }

  for (const item of items) {
    let key: string
    if (monthly) {
      const idx = Math.max(nowIdx, item.endDate.getFullYear() * 12 + item.endDate.getMonth())
      const year = Math.floor(idx / 12)
      const month = (idx % 12) + 1
      key = `${year}-${pad2(month)}`
    } else {
      key = String(Math.max(nowYear, item.endDate.getFullYear()))
    }
    const bucket = buckets.get(key)
    if (!bucket) continue
    bucket.amount += item.debt.pending_amount || 0
    bucket.count += 1
    bucket.debts.push(item.debt.description)
  }

  const ordered = [...buckets.values()].sort((a, b) => a.key.localeCompare(b.key))
  let remaining = items.reduce((sum, item) => sum + (item.debt.pending_amount || 0), 0)
  return ordered.map((bucket) => {
    remaining = Math.max(0, Math.round((remaining - bucket.amount) * 100) / 100)
    return { ...bucket, remaining }
  })
}

export default function DebtProjectionChart({
  items,
  currencies,
  effectiveCurrency,
}: {
  items: Item[]
  currencies: Currency[]
  effectiveCurrency: string
}) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const [tip, setTip] = useState<{ left: number; top: number; html: string } | null>(null)
  const now = useMemo(() => new Date(), [])

  const symbol = currencies.find((c) => c.code === effectiveCurrency)?.symbol || effectiveCurrency
  const total = items.reduce((sum, item) => sum + (item.debt.pending_amount || 0), 0)
  const monthly = isMonthlyHorizon(items, now)
  const points = useMemo(
    () => buildProjection(items, now, t, monthly),
    // t cambia de identidad en cada render del store; el idioma activo alcanza.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [items, monthly, now, t]
  )

  if (points.length === 0 || total <= 0) return null

  const yTop = Math.max(niceMax(total), 1)
  const W = DESIGN_W
  const H = CHART_H
  const L = 58
  const R = 16
  const TOP = 22
  const BOTTOM = 42
  const plotW = W - L - R
  const plotH = H - TOP - BOTTOM
  const slot = plotW / points.length
  const Y = (value: number) => TOP + plotH - (value / yTop) * plotH

  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => yTop * f)

  let previous = total
  let path = `M${L} ${Y(total)}`
  for (let i = 0; i < points.length; i++) {
    const xEnd = L + (i + 1) * slot
    path += ` L${xEnd} ${Y(previous)} L${xEnd} ${Y(points[i].remaining)}`
    previous = points[i].remaining
  }
  const fill = `${path} L${L + plotW} ${Y(0)} L${L} ${Y(0)} Z`

  const showTip = (e: React.MouseEvent, html: string) => {
    const left = Math.min(Math.max(e.clientX + 14, 8), window.innerWidth - TIP_MAX_W - 8)
    const top = Math.min(Math.max(e.clientY + 14, 8), window.innerHeight - 110)
    setTip({ left, top, html })
  }

  const pointTooltip = (point: ProjectionPoint) => {
    const names = point.debts
    const list = names.length
      ? names.length <= 3
        ? names.map((name) => `<br>• ${esc(name)}`).join('')
        : `<br>• ${names.slice(0, 3).map(esc).join('<br>• ')}<br>• ${names.length - 3} ${t('deudas.projection_and_more')}`
      : `<br><span style="opacity:.7">${t('deudas.projection_empty_period')}</span>`
    return [
      `<b>${point.fullLabel}</b>`,
      `${t('deudas.projection_ends_amount')}: <b>${formatMoney(point.amount, symbol)}</b>`,
      `${t('deudas.projection_debts_count')}: <b>${point.count}</b>`,
      `${t('deudas.projection_remaining_balance')}: <b>${formatMoney(point.remaining, symbol)}</b>`,
      list,
    ].join('<br>')
  }

  const firstZero = points.findIndex((point) => point.remaining <= 0)
  const currentKey = monthly ? `${now.getFullYear()}-${pad2(now.getMonth() + 1)}` : String(now.getFullYear())

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <span className="inline-flex items-center px-2.5 py-1 rounded-full bg-bg text-[11px] font-medium text-text-secondary">
          {monthly ? t('deudas.projection_granularity_monthly') : t('deudas.projection_granularity_yearly')}
        </span>
      </div>

      <div className="overflow-x-auto">
        <svg
          viewBox={`0 0 ${W} ${H}`}
          className="block w-full"
          style={{ minWidth: `${Math.max(360, points.length * 44)}px` }}
          preserveAspectRatio="xMidYMid meet"
          role="img"
          aria-label={t('deudas.chart_title_projection')}
        >
          {points.map((point, i) => (
            <rect
              key={`bg-${point.key}`}
              x={L + i * slot}
              y={TOP}
              width={slot}
              height={plotH}
              fill={point.key === currentKey ? ACCENT : 'transparent'}
              opacity={point.key === currentKey ? 0.07 : 1}
              pointerEvents="none"
            />
          ))}

          {ticks.map((value) => (
            <g key={value}>
              <line x1={L} x2={L + plotW} y1={Y(value)} y2={Y(value)} stroke={BORDER_COLOR} strokeDasharray="3 4" />
              <text className="m" x={L - 8} y={Y(value) + 4} fontSize="11" textAnchor="end" fill={TEXT_SECONDARY}>
                {compactMoney(value, symbol)}
              </text>
            </g>
          ))}

          <line
            x1={L}
            x2={L + plotW}
            y1={Y(total / 2)}
            y2={Y(total / 2)}
            stroke={TEXT_SECONDARY}
            strokeDasharray="6 4"
            strokeWidth="1"
            opacity="0.8"
          />
          <text className="m" x={L + 4} y={Y(total / 2) - 5} fontSize="10" fill={TEXT_SECONDARY}>
            {t('deudas.projection_half_balance')}
          </text>

          <path d={fill} fill={ACCENT} opacity="0.16" />
          <path d={path} fill="none" stroke={ACCENT} strokeWidth="2.5" strokeLinejoin="round" />

          {points.map((point, i) => {
            const x = L + i * slot
            return (
              <g key={point.key}>
                <rect
                  x={x}
                  y={TOP}
                  width={slot}
                  height={plotH}
                  fill="transparent"
                  onMouseMove={(e) => showTip(e, pointTooltip(point))}
                  onMouseLeave={() => setTip(null)}
                />
                <circle
                  cx={x + slot}
                  cy={Y(point.remaining)}
                  r={4}
                  fill={ACCENT}
                  stroke="rgb(var(--color-card))"
                  strokeWidth="2"
                  pointerEvents="none"
                />
                <text
                  className="m"
                  x={x + slot / 2}
                  y={H - 16}
                  fontSize="11"
                  textAnchor="middle"
                  fill={point.key === currentKey ? TEXT_COLOR : TEXT_SECONDARY}
                >
                  {point.label}
                </text>
              </g>
            )
          })}

          <line x1={L} x2={L + plotW} y1={Y(0)} y2={Y(0)} stroke={BORDER_COLOR} />

          {firstZero >= 0 && (
            <text
              className="m"
              x={L + (firstZero + 1) * slot - 4}
              y={Y(0) - 10}
              fontSize="11"
              fontWeight="600"
              textAnchor="end"
              fill={ACCENT}
            >
              {t('deudas.projection_free')}
            </text>
          )}
        </svg>
      </div>

      <p className="text-[10px] text-text-secondary">{t('deudas.chart_note_projection')}</p>

      <details className="bg-card rounded-ios shadow-ios">
        <summary className="px-4 py-3 cursor-pointer text-sm font-semibold min-h-[44px] flex items-center">
          {t('deudas.projection_table_title')}
        </summary>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-text-secondary border-t border-border">
                <th className="px-4 py-2 font-medium">{t('deudas.projection_period')}</th>
                <th className="px-4 py-2 font-medium text-right">{t('deudas.projection_ends_amount')}</th>
                <th className="px-4 py-2 font-medium text-right">{t('deudas.projection_debts_count')}</th>
                <th className="px-4 py-2 font-medium text-right">{t('deudas.projection_remaining_balance')}</th>
              </tr>
            </thead>
            <tbody>
              {points.map((point, i) => (
                <tr key={point.key} className="border-t border-border">
                  <td className="px-4 py-2">{point.fullLabel}</td>
                  <td className="px-4 py-2 text-right">{formatMoney(point.amount, symbol)}</td>
                  <td className="px-4 py-2 text-right">{point.count}</td>
                  <td className={`px-4 py-2 text-right ${i === firstZero ? 'text-success font-medium' : ''}`}>
                    {formatMoney(point.remaining, symbol)}
                  </td>
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
