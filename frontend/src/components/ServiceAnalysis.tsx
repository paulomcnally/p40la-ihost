import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import LoadingSpinner from './LoadingSpinner'
import PayBillModal from './PayBillModal'
import Select from './Select'
import type { ServiceBillDetail } from '../types'

const PALETTE = ['#0a84ff', '#ff9f0a', '#30d158', '#ff375f', '#bf5af2', '#64d2ff', '#ffd60a', '#5e5ce6']

function polarToCartesian(cx: number, cy: number, r: number, angleDeg: number) {
  const rad = ((angleDeg - 90) * Math.PI) / 180
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) }
}

function donutWedge(cx: number, cy: number, outerR: number, innerR: number, startAngle: number, endAngle: number) {
  const startOuter = polarToCartesian(cx, cy, outerR, endAngle)
  const endOuter = polarToCartesian(cx, cy, outerR, startAngle)
  const startInner = polarToCartesian(cx, cy, innerR, endAngle)
  const endInner = polarToCartesian(cx, cy, innerR, startAngle)
  const largeArc = endAngle - startAngle <= 180 ? '0' : '1'
  return [
    `M ${startOuter.x} ${startOuter.y}`,
    `A ${outerR} ${outerR} 0 ${largeArc} 0 ${endOuter.x} ${endOuter.y}`,
    `L ${endInner.x} ${endInner.y}`,
    `A ${innerR} ${innerR} 0 ${largeArc} 1 ${startInner.x} ${startInner.y}`,
    'Z',
  ].join(' ')
}

interface ServiceAnalysisProps {
  homeId: number | null
  year: number
  month: number
  onChangePeriod: (year: number, month: number) => void
}

type Totals = { total: number; paid: number; pending: number }

export default function ServiceAnalysis({ homeId, year, month, onChangePeriod }: ServiceAnalysisProps) {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const now = useMemo(() => new Date(), [])
  const [bills, setBills] = useState<ServiceBillDetail[]>([])
  const [loading, setLoading] = useState(true)
  const [payTarget, setPayTarget] = useState<ServiceBillDetail | null>(null)
  const [currencyFilter, setCurrencyFilter] = useState<string>('all')
  const [search, setSearch] = useState('')

  const isCurrent = year === now.getFullYear() && month === now.getMonth() + 1

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.serviceBills.byMonth(year, month, homeId)
      setBills(list || [])
    } catch {
      setBills([])
    } finally {
      setLoading(false)
    }
  }, [year, month, homeId])

  useEffect(() => {
    load()
  }, [load])

  const goPrev = () => {
    if (month === 1) onChangePeriod(year - 1, 12)
    else onChangePeriod(year, month - 1)
  }
  const goNext = () => {
    if (month === 12) onChangePeriod(year + 1, 1)
    else onChangePeriod(year, month + 1)
  }
  const goCurrent = () => onChangePeriod(now.getFullYear(), now.getMonth() + 1)

  const totals = useMemo(() => {
    const map = new Map<string, Totals>()
    for (const b of bills) {
      const code = b.currency_code || 'NIO'
      const entry = map.get(code) || { total: 0, paid: 0, pending: 0 }
      entry.total += b.amount
      if (b.status === 'paid') entry.paid += b.amount
      else entry.pending += b.amount
      map.set(code, entry)
    }
    return map
  }, [bills])

  const currencyOptions = useMemo(
    () => [
      { value: 'all', label: t('services.analysis_all_currencies') },
      ...[...totals.keys()].sort().map((code) => ({ value: code, label: code })),
    ],
    [totals, t]
  )

  const summaryTotals = useMemo(() => {
    if (currencyFilter === 'all') return totals
    const entry = totals.get(currencyFilter)
    return entry ? new Map([[currencyFilter, entry]]) : new Map<string, Totals>()
  }, [totals, currencyFilter])

  const mainCurrency = useMemo(() => {
    return [...totals.entries()].sort(
      (a, b) => b[1].pending - a[1].pending || b[1].total - a[1].total
    )[0]?.[0] || ''
  }, [totals])

  const displayCurrency = currencyFilter !== 'all' ? currencyFilter : mainCurrency
  const displayTotals = totals.get(displayCurrency) || { total: 0, paid: 0, pending: 0 }
  const paidPct = displayTotals.total > 0 ? Math.round((displayTotals.paid / displayTotals.total) * 100) : 0
  const currencySymbol = bills.find((b) => (b.currency_code || 'NIO') === displayCurrency)?.currency_symbol || displayCurrency

  const scopedBills = useMemo(
    () => (currencyFilter === 'all' ? bills : bills.filter((b) => (b.currency_code || 'NIO') === currencyFilter)),
    [bills, currencyFilter]
  )

  // El desglose por servicio y el donut se muestran siempre en una sola
  // moneda para no mezclar montos (SPEC-120 ADR-004).
  const displayBills = useMemo(
    () => bills.filter((b) => (b.currency_code || 'NIO') === displayCurrency),
    [bills, displayCurrency]
  )

  const searchResults = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return scopedBills
    return scopedBills.filter(
      (b) =>
        b.service_name.toLowerCase().includes(q) ||
        (b.institution || '').toLowerCase().includes(q) ||
        (b.invoice_number || '').toLowerCase().includes(q)
    )
  }, [scopedBills, search])

  const byService = useMemo(() => {
    const map = new Map<number, { name: string; total: number; paid: number; pending: number }>()
    for (const b of displayBills) {
      const entry = map.get(b.service_id) || { name: b.service_name, total: 0, paid: 0, pending: 0 }
      entry.total += b.amount
      if (b.status === 'paid') entry.paid += b.amount
      else entry.pending += b.amount
      map.set(b.service_id, entry)
    }
    return [...map.values()].sort((a, b) => b.pending - a.pending || b.total - a.total)
  }, [displayBills])

  const donutData = useMemo(() => {
    const scoped = byService.filter((s) => s.pending > 0)
    const totalPending = scoped.reduce((sum, s) => sum + s.pending, 0)
    if (totalPending === 0) return []
    const top = scoped.slice(0, 6)
    const others = scoped.slice(6).reduce((sum, s) => sum + s.pending, 0)
    const items = top.map((s) => ({ label: s.name, value: s.pending }))
    if (others > 0) items.push({ label: t('services.analysis_others'), value: others })
    let acc = 0
    return items.map((item, i) => {
      const start = (acc / totalPending) * 360
      acc += item.value
      const end = (acc / totalPending) * 360
      return { ...item, color: PALETTE[i % PALETTE.length], startAngle: start, endAngle: end }
    })
  }, [byService, t])

  const periodLabel = `${t(`months.${month}`)} ${year}`

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <div className="flex items-center gap-2 flex-wrap">
          <div className="flex items-center gap-1 bg-card rounded-ios shadow-ios px-2 py-1">
            <button onClick={goPrev} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors" aria-label={t('services.analysis_prev_month')}>
              <Icon name="chevron" className="w-4 h-4 rotate-180" />
            </button>
            <span className="text-sm font-semibold min-w-28 text-center">{periodLabel}</span>
            <button onClick={goNext} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors" aria-label={t('services.analysis_next_month')}>
              <Icon name="chevron" className="w-4 h-4" />
            </button>
          </div>
          {totals.size > 1 && (
            <div className="w-full sm:w-44">
              <Select
                options={currencyOptions}
                value={currencyFilter}
                onChange={(value) => setCurrencyFilter(String(value))}
              />
            </div>
          )}
        </div>
        {!isCurrent && (
          <button
            onClick={goCurrent}
            className="flex items-center gap-1 text-sm font-medium text-primary hover:underline min-h-[44px]"
          >
            <Icon name="refresh" className="w-3 h-3" />
            {t('services.analysis_current_month')}
          </button>
        )}
      </div>

      {loading ? (
        <LoadingSpinner text={t('app.loading')} />
      ) : bills.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="bill" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('services.analysis_empty')}</h3>
          <p className="text-text-secondary">{t('services.analysis_empty_subtitle')}</p>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
            <div className="flex flex-wrap items-end justify-between gap-3 mb-3">
              <div>
                <p className="text-xs font-medium uppercase tracking-wide text-text-secondary">{t('services.analysis_needed')}</p>
                <p className="text-2xl sm:text-3xl font-bold text-amber-500 dark:text-amber-400 flex flex-wrap gap-x-3 gap-y-1">
                  {[...summaryTotals.entries()].map(([code, entry]) => (
                    <span key={code}>{code} {formatMoney(entry.pending)}</span>
                  ))}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Icon name="clock" className="w-4 h-4 text-amber-500 dark:text-amber-400" />
                <span className="text-sm text-text-secondary">{periodLabel}</span>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex-1 h-2 rounded-full bg-border overflow-hidden">
                <div className="h-full rounded-full bg-success transition-all" style={{ width: `${paidPct}%` }} />
              </div>
              <span className="text-sm font-semibold text-emerald-600 dark:text-emerald-400">{paidPct}%</span>
            </div>
            <p className="text-xs text-text-secondary mt-1.5">{t('services.analysis_paid_of_month')}</p>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
            <SummaryCard
              label={t('services.analysis_total_due')}
              lines={[...summaryTotals.entries()].map(([code, entry]) => `${code} ${formatMoney(entry.total)}`)}
              color="text-text"
            />
            <SummaryCard
              label={t('services.analysis_paid')}
              lines={[...summaryTotals.entries()].map(([code, entry]) => `${code} ${formatMoney(entry.paid)}`)}
              color="text-emerald-600 dark:text-emerald-400"
            />
            <SummaryCard
              label={t('services.analysis_pending')}
              lines={[...summaryTotals.entries()].map(([code, entry]) => `${code} ${formatMoney(entry.pending)}`)}
              color="text-amber-500 dark:text-amber-400"
            />
          </div>

          {donutData.length > 0 && (
            <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
              <h3 className="font-semibold mb-1">{t('services.analysis_by_service')}</h3>
              <p className="text-xs text-text-secondary mb-4">{t('services.analysis_by_service_subtitle')}</p>
              <div className="flex flex-col sm:flex-row items-center gap-6">
                <div className="relative w-full max-w-[180px] mx-auto sm:mx-0 shrink-0">
                  <svg viewBox="0 0 180 180" className="block w-full h-auto" role="img" aria-label={t('services.analysis_by_service')}>
                    <circle cx="90" cy="90" r="72" fill="none" stroke="var(--color-border, #e5e5ea)" strokeWidth="22" />
                    {donutData.map((seg) => (
                      <path key={seg.label} d={donutWedge(90, 90, 72, 50, seg.startAngle, seg.endAngle)} fill={seg.color} />
                    ))}
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                    <span className="text-xs text-text-secondary">{t('services.analysis_pending')}</span>
                    <span className="text-[10px] font-medium text-text-secondary">{displayCurrency}</span>
                    <span className="text-lg font-bold">{formatMoney(displayTotals.pending, currencySymbol)}</span>
                  </div>
                </div>
                <div className="flex-1 w-full min-w-0">
                  <div className="space-y-2.5">
                    {donutData.map((seg) => (
                      <div key={seg.label} className="flex items-center gap-3">
                        <span className="w-3 h-3 rounded-full shrink-0" style={{ backgroundColor: seg.color }} />
                        <span className="flex-1 text-sm truncate">{seg.label}</span>
                        <span className="text-sm font-semibold shrink-0">
                          {displayCurrency} {formatMoney(seg.value || 0, currencySymbol)}
                          <span className="text-text-secondary font-normal">
                            {' '}({Math.round(((seg.value || 0) / Math.max(1, donutData.reduce((sum, d) => sum + d.value, 0))) * 100)}%)
                          </span>
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}

          {byService.length > 0 && (
            <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
              <h3 className="font-semibold mb-4">{t('services.analysis_per_service')}</h3>
              <div className="space-y-4">
                {byService.map((service) => {
                  const total = service.total || 1
                  const paidPctService = Math.round((service.paid / total) * 100)
                  const pendingPctService = Math.round((service.pending / total) * 100)
                  return (
                    <div key={service.name}>
                      <div className="flex items-center justify-between gap-3 mb-1.5">
                        <span className="text-sm font-medium truncate">{service.name}</span>
                        <span className="text-sm font-semibold shrink-0">{displayCurrency} {formatMoney(service.total, currencySymbol)}</span>
                      </div>
                      <div className="h-2 rounded-full bg-border overflow-hidden flex">
                        <div className="h-full bg-success" style={{ width: `${paidPctService}%` }} />
                        <div className="h-full bg-amber-400" style={{ width: `${pendingPctService}%` }} />
                      </div>
                      <div className="flex items-center justify-between gap-3 mt-1 text-xs text-text-secondary">
                        <span>{t('services.analysis_paid')}: {displayCurrency} {formatMoney(service.paid, currencySymbol)}</span>
                        <span>{t('services.analysis_pending')}: {displayCurrency} {formatMoney(service.pending, currencySymbol)}</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          <div>
            <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">{t('services.analysis_bills_title')}</h3>
            <div className="relative mb-2">
              <Icon name="search" className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-secondary pointer-events-none" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('services.analysis_search_placeholder')}
                className="w-full pl-9 pr-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card text-text placeholder:text-text-secondary min-h-[44px]"
              />
            </div>
            <div className="space-y-2">
              {searchResults.map((bill) => (
                <div key={bill.id} className="bg-card rounded-ios shadow-ios p-3 sm:p-4">
                  <div className="flex items-center justify-between gap-3">
                    <button
                      onClick={() => navigate(`/services/bills/${bill.service_id}`)}
                      className="flex-1 min-w-0 flex items-center gap-3 text-left min-h-[44px] group"
                    >
                      <div className="w-9 h-9 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
                        <Icon name={bill.icon_key || 'other'} className="w-5 h-5" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium truncate group-hover:text-primary transition-colors">{bill.service_name}</p>
                        <p className="text-xs text-text-secondary truncate">
                          {bill.institution ? `${bill.institution} · ` : ''}{bill.period_date}
                          {bill.date_estimated && ` · ${t('services.analysis_date_estimated')}`}
                        </p>
                        <p className="text-sm font-semibold mt-1">{bill.currency_code || 'NIO'} {formatMoney(bill.amount, bill.currency_symbol || bill.currency_code)}</p>
                      </div>
                      <Icon name="chevron" className="w-4 h-4 text-text-secondary shrink-0" />
                    </button>
                    <div className="flex flex-col items-end gap-2 shrink-0">
                      <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                        bill.status === 'paid'
                          ? 'bg-success/20 text-green-800 dark:text-green-400'
                          : 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
                      }`}>
                        {t(`bills.status_${bill.status}`)}
                      </span>
                      {bill.status === 'pending' && (
                        <button
                          onClick={() => setPayTarget(bill)}
                          className="text-xs font-semibold text-primary hover:underline min-h-[36px]"
                        >
                          {t('bills.pay')}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
            {searchResults.length === 0 && (
              <p className="text-sm text-text-secondary mt-3">{t('services.analysis_no_results')}</p>
            )}
          </div>
        </div>
      )}

      {payTarget && (
        <PayBillModal
          bill={payTarget}
          onClose={() => setPayTarget(null)}
          onSuccess={load}
        />
      )}
    </div>
  )
}

function SummaryCard({ label, lines, color }: { label: string; lines: string[]; color: string }) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-3 sm:p-4">
      <p className="text-text-secondary text-xs font-medium uppercase tracking-wide">{label}</p>
      <div className={`mt-1 text-lg sm:text-xl font-bold break-words ${color}`}>
        {lines.map((line) => (
          <p key={line}>{line}</p>
        ))}
      </div>
    </div>
  )
}
