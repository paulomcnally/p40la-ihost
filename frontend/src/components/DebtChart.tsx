import { useMemo, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import Select from './Select'
import DebtHistoryChart from './DebtHistoryChart'
import DebtProjectionChart from './DebtProjectionChart'
import { debtPendingEndDate, leftTime } from '../utils/debtMonthlyAggregation'
import type { Debt, Currency } from '../types'

type View = 'cronograma' | 'saldo' | 'historial' | 'proyeccion'

const COL = ['#0a84ff', '#30b0c7', '#5e5ce6', '#34c759', '#ff9f0a', '#ff375f']

const ACCENT = 'rgb(var(--color-primary))'
const TEXT_COLOR = 'rgb(var(--color-text))'
const TEXT_SECONDARY = 'rgb(var(--color-text-secondary))'
const BORDER_COLOR = 'rgb(var(--color-border))'

const DAY_MS = 86400000

// Ancho de diseño del SVG; el navegador lo escala al contenedor con viewBox
// (patrón responsive de BillAnalysis). min-w en el wrapper evita aplastar en pantallas medias.
const DESIGN_W = 720

const TIP_MAX_W = 240 // coincide con max-w-60 del tooltip

function monthShort(t: (k: string) => string, m: number) {
  return t(`months.${m}`).slice(0, 3)
}

function truncateLabel(s: string, max = 26) {
  return s.length > max ? `${s.slice(0, max - 1)}…` : s
}

export default function DebtChart({ debts, currencies }: { debts: Debt[]; currencies: Currency[] }) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const now = useMemo(() => new Date(), [])

  // SPEC-106: el tab Gráfica abre en la evolución de los últimos 12 meses.
  // Cronograma y Saldo total siguen disponibles manualmente.
  const [view, setView] = useState<View>('historial')
  const [currencyFilter, setCurrencyFilter] = useState<string>('')
  const [tip, setTip] = useState<{ left: number; top: number; html: string } | null>(null)

  // SPEC-104 REQ-008: solo deudas con saldo pendiente; fin proyectado por la
  // última cuota pendiente (no por el plan original).
  const items = useMemo(() => {
    return debts
      .filter((debt) => (debt.pending_count ?? 0) > 0)
      .map((debt) => {
        const end = debtPendingEndDate(debt)
        if (!end) return null
        return { debt, endDate: new Date(end.year, end.month - 1, end.day) }
      })
      .filter((x): x is { debt: Debt; endDate: Date } => x !== null)
      .sort((a, b) => a.endDate.getTime() - b.endDate.getTime())
  }, [debts])

  const closedCount = useMemo(
    () => debts.filter((d) => (d.pending_count ?? 0) === 0).length,
    [debts]
  )

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
      sums.set(code, (sums.get(code) || 0) + (item.debt.pending_amount || 0))
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

  // Todas las vistas del tab se alimentan del mismo conjunto filtrado por moneda
  // (cronograma móvil + SVG, saldo total y tarjetas). Ver ADR-005.
  const filteredItems = useMemo(
    () => (effectiveCurrency ? items.filter((i) => (i.debt.currency_code || '') === effectiveCurrency) : []),
    [items, effectiveCurrency]
  )

  const lastEnd = filteredItems.length > 0 ? filteredItems[filteredItems.length - 1].endDate : now
  const totalOwed = filteredItems.reduce((s, i) => s + (i.debt.pending_amount || 0), 0)
  const currencySymbol = currencies.find((c) => c.code === effectiveCurrency)?.symbol || effectiveCurrency

  const segmented = (
    <div className="flex flex-wrap items-center gap-1 bg-bg rounded-ios p-1 max-w-full">
      <button
        onClick={() => setView('cronograma')}
        aria-pressed={view === 'cronograma'}
        className={`px-2.5 sm:px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] whitespace-nowrap shrink-0 ${
          view === 'cronograma' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
        }`}
      >
        {t('deudas.chart_view_cronograma')}
      </button>
      <button
        onClick={() => setView('saldo')}
        aria-pressed={view === 'saldo'}
        className={`px-2.5 sm:px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] whitespace-nowrap shrink-0 ${
          view === 'saldo' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
        }`}
      >
        {t('deudas.chart_view_saldo')}
      </button>
      <button
        onClick={() => setView('historial')}
        aria-pressed={view === 'historial'}
        className={`px-2.5 sm:px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] whitespace-nowrap shrink-0 ${
          view === 'historial' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
        }`}
      >
        {t('deudas.chart_view_history')}
      </button>
      <button
        onClick={() => setView('proyeccion')}
        aria-pressed={view === 'proyeccion'}
        className={`px-2.5 sm:px-3 py-1.5 rounded-ios-sm text-xs font-medium transition-colors min-h-[44px] whitespace-nowrap shrink-0 ${
          view === 'proyeccion' ? 'bg-primary text-white' : 'text-text-secondary hover:bg-border'
        }`}
      >
        {t('deudas.chart_view_projection')}
      </button>
    </div>
  )

  const emptyCard = (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name="bar" className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.chart_empty')}</h3>
      <p className="text-text-secondary">{t('deudas.chart_empty_subtitle')}</p>
    </div>
  )

  // SPEC-105: la vista histórica se muestra aunque no queden deudas abiertas.
  if (view === 'historial') {
    return (
      <div className="space-y-3 sm:space-y-4">
        <div className="flex justify-end">{segmented}</div>
        <DebtHistoryChart />
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="space-y-3 sm:space-y-4">
        <div className="flex justify-end">{segmented}</div>
        {emptyCard}
      </div>
    )
  }

  if (filteredItems.length === 0) {
    return (
      <div className="space-y-3 sm:space-y-4">
        <div className="flex justify-end">{segmented}</div>
        {emptyCard}
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
    const left = Math.min(Math.max(e.clientX + 14, 8), window.innerWidth - TIP_MAX_W - 8)
    const top = Math.min(Math.max(e.clientY + 14, 8), window.innerHeight - 96)
    setTip({ left, top, html })
  }

  const rowTooltip = (item: { debt: Debt; endDate: Date }) => {
    const sym = currencies.find((c) => c.code === item.debt.currency_code)?.symbol || item.debt.currency_code || ''
    return `<b>${item.debt.description}</b><br>${formatMoney(item.debt.pending_amount, sym)}<br>${t('deudas.chart_ends')} ${fm(item.endDate)}<br>${t('deudas.chart_remains')} ${leftLabel(item.endDate)}`
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

  // SPEC-105: Cronograma vertical. Una columna por deuda; la altura es el
  // tiempo restante hasta su finalización. En pantallas angostas se permite
  // scroll horizontal manteniendo el tamaño de fuente legible.
  const renderCronograma = () => {
    const n = filteredItems.length
    const L = 40
    const R = 12
    const TOP = 30
    const BOTTOM = 78
    const H = 360
    const slot = Math.max(48, Math.min(84, 480 / Math.max(n, 1)))
    const plotW = slot * n
    const W = L + plotW + R
    const plotH = H - TOP - BOTTOM
    const baseline = TOP + plotH
    const Y = (tt: number) => TOP + plotH - ((tt - t0) / SPAN) * plotH
    const barW = Math.min(44, Math.max(14, slot * 0.52))
    const nameMax = Math.max(8, Math.floor(slot / 5.5))
    const firstYear = now.getFullYear() + 1
    const lastYear = lastEnd.getFullYear() + 1
    const yearStep = Math.max(1, Math.ceil((lastYear - firstYear + 1) / 8))
    const tickYears: number[] = []
    for (let y = firstYear; y <= lastYear; y += yearStep) tickYears.push(y)

    return (
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="block w-full"
        style={{ minWidth: `${Math.max(320, W)}px` }}
        preserveAspectRatio="xMidYMid meet"
        role="img"
        aria-label={t('deudas.chart_title_cronograma')}
      >
        {tickYears.map((year) => {
          const y = Y(new Date(year, 0, 1).getTime())
          if (y < TOP || y > baseline) return null
          return (
            <g key={year}>
              <line x1={L} x2={L + plotW} y1={y} y2={y} stroke={BORDER_COLOR} strokeDasharray="3 4" />
              <text className="m" x={L - 6} y={y + 4} fontSize="11" textAnchor="end" fill={TEXT_SECONDARY}>
                {year}
              </text>
            </g>
          )
        })}

        {filteredItems.map((item, i) => {
          const cx = L + i * slot + slot / 2
          const yEnd = Math.min(baseline, Math.max(TOP, Y(item.endDate.getTime())))
          const barH = Math.max(3, baseline - yEnd)
          const sym = currencies.find((c) => c.code === item.debt.currency_code)?.symbol || item.debt.currency_code || ''
          return (
            <g key={item.debt.id} style={{ cursor: 'default' }}>
              <rect
                x={L + i * slot}
                y={TOP}
                width={slot}
                height={plotH}
                fill="transparent"
                onMouseMove={(e) => showTip(e, rowTooltip(item))}
                onMouseLeave={() => setTip(null)}
              />
              <rect x={cx - barW / 2} y={yEnd} width={barW} height={barH} rx={4} fill={COL[i % COL.length]} pointerEvents="none" />
              <text className="m" x={cx} y={yEnd - 6} fontSize="11" textAnchor="middle" fill={TEXT_SECONDARY} pointerEvents="none">
                {`${monthShort(t, item.endDate.getMonth() + 1)} ${String(item.endDate.getFullYear()).slice(2)}`}
              </text>
              <text
                className="m"
                x={cx + 3}
                y={baseline + 14}
                fontSize="11"
                textAnchor="end"
                fill={TEXT_COLOR}
                transform={`rotate(-38 ${cx + 3} ${baseline + 14})`}
                pointerEvents="none"
              >
                {truncateLabel(item.debt.description, nameMax)}
              </text>
              <text className="m" x={cx} y={H - 6} fontSize="11" textAnchor="middle" fill={TEXT_SECONDARY} pointerEvents="none">
                {formatMoney(item.debt.pending_amount, sym)}
              </text>
            </g>
          )
        })}

        <line x1={L} x2={L + plotW} y1={baseline} y2={baseline} stroke={ACCENT} strokeWidth="1.5" />
        <text className="m" x={L + plotW} y={baseline + 12} fontSize="11" textAnchor="end" fill={ACCENT}>
          {t('deudas.chart_today')}
        </text>
      </svg>
    )
  }

  const renderSaldo = () => {
    const W = DESIGN_W
    const L = 56
    const R = 20
    const top = 24
    const H = 300
    const w = W - L - R
    const h = H - top
    const X = (tt: number) => L + (tt - t0) / SPAN * w
    const Y = (v: number) => top + h - (v / Math.max(totalOwed, 1)) * h

    let bal = totalOwed
    const pts: [number, number][] = [[t0, bal]]
    for (const item of filteredItems) {
      pts.push([item.endDate.getTime(), bal])
      bal -= item.debt.pending_amount || 0
      pts.push([item.endDate.getTime(), bal])
    }
    pts.push([lastEnd.getTime() + 30 * DAY_MS, 0])
    const line = pts.map((p, i) => `${i ? 'L' : 'M'}${X(p[0]).toFixed(1)} ${Y(p[1]).toFixed(1)}`).join(' ')

    return (
      <svg viewBox={`0 0 ${W} ${H + 26}`} className="block w-full min-w-[360px]" preserveAspectRatio="xMidYMid meet" role="img" aria-label={t('deudas.chart_title_saldo')}>
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
          return filteredItems.map((item, i) => {
            balAfter -= item.debt.pending_amount || 0
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
          <b className="block text-2xl font-bold mt-1">{filteredItems.length}</b>
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
              {view === 'cronograma'
                ? t('deudas.chart_title_cronograma')
                : view === 'proyeccion'
                  ? t('deudas.chart_title_projection')
                  : t('deudas.chart_title_saldo')}
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
          {segmented}
        </div>

        <div className="overflow-x-auto">
          {view === 'cronograma' ? (
            renderCronograma()
          ) : view === 'proyeccion' ? (
            <DebtProjectionChart
              items={filteredItems}
              currencies={currencies}
              effectiveCurrency={effectiveCurrency}
            />
          ) : (
            renderSaldo()
          )}
        </div>

        {view !== 'proyeccion' && (
          <p className="text-[10px] text-text-secondary mt-2">
            {view === 'cronograma' ? t('deudas.chart_note_cronograma') : t('deudas.chart_note_saldo')}
          </p>
        )}
        {closedCount > 0 && (
          <p className="text-[10px] text-text-secondary mt-1">
            {closedCount} {t('deudas.chart_closed_out')}
          </p>
        )}
      </div>

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