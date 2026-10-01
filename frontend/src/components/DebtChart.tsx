import { useEffect, useMemo, useRef, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import Select from './Select'
import { debtEndDate, leftTime } from '../utils/debtMonthlyAggregation'
import type { Debt, Currency } from '../types'

type View = 'cronograma' | 'saldo'

const COL = ['#0a84ff', '#30b0c7', '#5e5ce6', '#34c759', '#ff9f0a', '#ff375f']

const ACCENT = 'rgb(var(--color-primary))'
const TEXT_COLOR = 'rgb(var(--color-text))'
const TEXT_SECONDARY = 'rgb(var(--color-text-secondary))'
const BORDER_COLOR = 'rgb(var(--color-border))'

const DAY_MS = 86400000

function monthShort(t: (k: string) => string, m: number) {
  return t(`months.${m}`).slice(0, 3)
}

export default function DebtChart({ debts, currencies }: { debts: Debt[]; currencies: Currency[] }) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const now = useMemo(() => new Date(), [])

  const [view, setView] = useState<View>('cronograma')
  const [currencyFilter, setCurrencyFilter] = useState<string>('')
  const [tip, setTip] = useState<{ left: number; top: number; html: string } | null>(null)
  const chartRef = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(640)

  useEffect(() => {
    const el = chartRef.current
    if (!el) return
    const update = () => setWidth(Math.max(el.clientWidth, 320))
    update()
    const ro = new ResizeObserver(update)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  const items = useMemo(() => {
    return debts
      .map((debt) => {
        const end = debtEndDate(debt)
        if (!end) return null
        return { debt, endDate: new Date(end.year, end.month - 1, end.day) }
      })
      .filter((x): x is { debt: Debt; endDate: Date } => x !== null)
      .sort((a, b) => a.endDate.getTime() - b.endDate.getTime())
  }, [debts])

  const currencyOptions = useMemo(() => {
    const codes = [...new Set(debts.map((d) => d.currency_code).filter((c): c is string => Boolean(c)))]
    return codes.map((code) => {
      const c = currencies.find((x) => x.code === code)
      return { value: code, label: c ? `${c.code} · ${c.name}` : code }
    })
  }, [debts, currencies])

  const defaultCurrency = useMemo(() => {
    const sums = new Map<string, number>()
    for (const item of items) {
      const code = item.debt.currency_code || ''
      sums.set(code, (sums.get(code) || 0) + (item.debt.total || 0))
    }
    let best = ''
    let bestSum = -1
    for (const [code, sum] of sums) {
      if (sum > bestSum) {
        best = code
        bestSum = sum
      }
    }
    return best
  }, [items])

  const effectiveCurrency = currencyFilter && currencyOptions.some((o) => o.value === currencyFilter)
    ? currencyFilter
    : defaultCurrency

  const saldoItems = useMemo(
    () => (effectiveCurrency ? items.filter((i) => (i.debt.currency_code || '') === effectiveCurrency) : []),
    [items, effectiveCurrency]
  )

  const lastEnd = items.length > 0 ? items[items.length - 1].endDate : now
  const totalOwed = saldoItems.reduce((s, i) => s + (i.debt.total || 0), 0)
  const currencySymbol = currencies.find((c) => c.code === effectiveCurrency)?.symbol || effectiveCurrency

  if (items.length === 0) {
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

  const leftLabel = (end: Date) => {
    const { years, months, thisMonth } = leftTime(end, now)
    if (thisMonth) return t('deudas.chart_this_month')
    const parts: string[] = []
    if (years) parts.push(`${years} ${years > 1 ? t('deudas.chart_years') : t('deudas.chart_year')}`)
    if (months) parts.push(`${months} ${months > 1 ? t('deudas.chart_months') : t('deudas.chart_month')}`)
    return parts.join(' ')
  }

  const fm = (d: Date) => `${monthShort(t, d.getMonth() + 1)} ${d.getFullYear()}`

  const showTip = (e: React.MouseEvent, html: string) => {
    setTip({
      left: Math.min(e.clientX + 14, window.innerWidth - 250),
      top: e.clientY + 14,
      html,
    })
  }

  const rowTooltip = (item: { debt: Debt; endDate: Date }) => {
    const sym = currencies.find((c) => c.code === item.debt.currency_code)?.symbol || item.debt.currency_code || ''
    return `<b>${item.debt.description}</b><br>${formatMoney(item.debt.total, sym)}<br>${t('deudas.chart_ends')} ${fm(item.endDate)}<br>${t('deudas.chart_remains')} ${leftLabel(item.endDate)}`
  }

  const pointTooltip = (item: { debt: Debt; endDate: Date }, balanceAfter: number) => {
    const sym = currencies.find((c) => c.code === item.debt.currency_code)?.symbol || item.debt.currency_code || ''
    return `${rowTooltip(item)}<br>${t('deudas.chart_balance_after')} ${formatMoney(balanceAfter, sym)}`
  }

  const t0 = now.getTime()
  const SPAN = lastEnd.getTime() - t0 + 30 * DAY_MS

  const yearTicks = (x0: number, plotW: number, h: number) => {
    const y0 = now.getFullYear()
    const y1 = lastEnd.getFullYear()
    const step = y1 - y0 > 12 ? 2 : 1
    const ticks: { x: number; y: number }[] = []
    for (let y = y0 + 1; y <= y1 + 1; y += step) {
      const x = x0 + (new Date(y, 0, 1).getTime() - t0) / SPAN * plotW
      if (x > x0 + plotW) continue
      ticks.push({ x, y })
    }
    return (
      <>
        {ticks.map((tk) => (
          <g key={tk.y}>
            <line x1={tk.x} x2={tk.x} y1={8} y2={h} stroke={BORDER_COLOR} />
            <text className="m" x={tk.x} y={h + 16} fontSize="11" textAnchor="middle" fill={TEXT_SECONDARY}>
              {tk.y}
            </text>
          </g>
        ))}
      </>
    )
  }

  const renderCronograma = () => {
    const row = 38
    const top = 8
    const L = Math.min(190, width * 0.34)
    const R = 84
    const w = width - L - R
    const H = top + items.length * row
    return (
      <svg width={width} height={H + 26} role="img" aria-label={t('deudas.chart_title_cronograma')}>
        {yearTicks(L, w, H)}
        {items.map((item, i) => {
          const y = top + i * row
          const x2 = L + (item.endDate.getTime() - t0) / SPAN * w
          const bw = Math.max(5, x2 - L)
          const sym = currencies.find((c) => c.code === item.debt.currency_code)?.symbol || item.debt.currency_code || ''
          return (
            <g key={item.debt.id} style={{ cursor: 'default' }}>
              <rect x={0} y={y} width={width} height={row} fill="transparent" onMouseMove={(e) => showTip(e, rowTooltip(item))} onMouseLeave={() => setTip(null)} />
              <text x={0} y={y + 16} fontSize="13" fontWeight="600" fill={TEXT_COLOR}>
                {item.debt.description}
              </text>
              <text className="m" x={0} y={y + 30} fontSize="11" fill={TEXT_SECONDARY}>
                {formatMoney(item.debt.total, sym)}
              </text>
              <rect x={L} y={y + 8} width={w} height={20} rx={5} fill={BORDER_COLOR} />
              <rect x={L} y={y + 8} width={bw} height={20} rx={5} fill={COL[i % COL.length]} />
              <text x={L + bw + 8} y={y + 23} fontSize="12" fill={TEXT_COLOR}>
                {fm(item.endDate)}
              </text>
            </g>
          )
        })}
        <line x1={L} x2={L} y1={8} y2={H} stroke={ACCENT} strokeWidth="2" />
        <text x={L + 4} y={H + 16} fontSize="11" fill={ACCENT} style={{ fill: ACCENT }}>
          {t('deudas.chart_today')}
        </text>
      </svg>
    )
  }

  const renderSaldo = () => {
    const L = 56
    const R = 20
    const top = 24
    const H = 300
    const w = width - L - R
    const h = H - top
    const X = (tt: number) => L + (tt - t0) / SPAN * w
    const Y = (v: number) => top + h - (v / Math.max(totalOwed, 1)) * h

    let bal = totalOwed
    const pts: [number, number][] = [[t0, bal]]
    for (const item of saldoItems) {
      pts.push([item.endDate.getTime(), bal])
      bal -= item.debt.total || 0
      pts.push([item.endDate.getTime(), bal])
    }
    pts.push([lastEnd.getTime() + 30 * DAY_MS, 0])
    const line = pts.map((p, i) => `${i ? 'L' : 'M'}${X(p[0]).toFixed(1)} ${Y(p[1]).toFixed(1)}`).join(' ')

    return (
      <svg width={width} height={H + 26} role="img" aria-label={t('deudas.chart_title_saldo')}>
        {[0, 1, 2, 3, 4].map((i) => {
          const v = (totalOwed * i) / 4
          const y = Y(v)
          const label = v >= 1000 ? `${Math.round(v / 1000)}k` : Math.round(v).toString()
          return (
            <g key={i}>
              <line x1={L} x2={L + w} y1={y} y2={y} stroke={BORDER_COLOR} strokeDasharray="3 4" />
              <text className="m" x={L - 8} y={y + 4} fontSize="11" textAnchor="end" fill={TEXT_SECONDARY}>
                {label}
              </text>
            </g>
          )
        })}
        {yearTicks(L, w, H)}
        <path d={`${line} L${X(lastEnd.getTime() + 30 * DAY_MS)} ${Y(0)} L${X(t0)} ${Y(0)}Z`} fill={ACCENT} opacity="0.18" />
        <path d={line} fill="none" stroke={ACCENT} strokeWidth="2.5" />
        {(() => {
          let balAfter = totalOwed
          return saldoItems.map((item, i) => {
            balAfter -= item.debt.total || 0
            const sym = currencies.find((c) => c.code === item.debt.currency_code)?.symbol || item.debt.currency_code || ''
            return (
              <circle
                key={item.debt.id}
                cx={X(item.endDate.getTime())}
                cy={Y(balAfter)}
                r={6}
                fill={COL[i % COL.length]}
                stroke="rgb(var(--color-card))"
                strokeWidth="2"
                style={{ cursor: 'default' }}
                onMouseMove={(e) => showTip(e, pointTooltip(item, balAfter))}
                onMouseLeave={() => setTip(null)}
              />
            )
          })
        })()}
        <text x={X(lastEnd.getTime())} y={Y(0) - 12} fontSize="12" fontWeight="600" textAnchor="end" fill={TEXT_COLOR}>
          {t('deudas.chart_free_of')} {fm(lastEnd)}
        </text>
      </svg>
    )
  }

  return (
    <div className="space-y-3 sm:space-y-4">
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
        <div className="bg-card rounded-ios shadow-ios p-4">
          <span className="text-xs text-text-secondary">{t('deudas.chart_stat_total')}</span>
          <b className="block text-2xl font-bold mt-1">{formatMoney(totalOwed, currencySymbol)}</b>
        </div>
        <div className="bg-card rounded-ios shadow-ios p-4">
          <span className="text-xs text-text-secondary">{t('deudas.chart_stat_active')}</span>
          <b className="block text-2xl font-bold mt-1">{items.length}</b>
        </div>
        <div className="bg-card rounded-ios shadow-ios p-4">
          <span className="text-xs text-text-secondary">{t('deudas.chart_stat_free')}</span>
          <b className="block text-2xl font-bold mt-1">{fm(lastEnd)}</b>
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <div className="flex items-center gap-3 flex-wrap">
            <p className="text-sm font-semibold">
              {view === 'cronograma' ? t('deudas.chart_title_cronograma') : t('deudas.chart_title_saldo')}
            </p>
            {currencyOptions.length > 1 && (
              <div className="w-44">
                <Select
                  options={currencyOptions}
                  value={effectiveCurrency}
                  onChange={(v) => setCurrencyFilter(String(v))}
                />
              </div>
            )}
          </div>
          <div className="flex items-center gap-1 bg-bg rounded-ios p-1">
            <button
              onClick={() => setView('cronograma')}
              aria-pressed={view === 'cronograma'}
              className={`px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] ${
                view === 'cronograma' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
              }`}
            >
              {t('deudas.chart_view_cronograma')}
            </button>
            <button
              onClick={() => setView('saldo')}
              aria-pressed={view === 'saldo'}
              className={`px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] ${
                view === 'saldo' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
              }`}
            >
              {t('deudas.chart_view_saldo')}
            </button>
          </div>
        </div>

        <div ref={chartRef} className="overflow-x-auto">
          {view === 'cronograma' ? renderCronograma() : renderSaldo()}
        </div>

        <p className="text-[10px] text-text-secondary mt-2">
          {view === 'cronograma' ? t('deudas.chart_note_cronograma') : t('deudas.chart_note_saldo')}
        </p>
      </div>

      {tip && (
        <div
          className="fixed pointer-events-none bg-text text-bg px-2.5 py-2 rounded-ios-sm text-xs leading-relaxed z-[60] max-w-60"
          style={{ left: tip.left, top: tip.top }}
          dangerouslySetInnerHTML={{ __html: tip.html }}
        />
      )}
    </div>
  )
}