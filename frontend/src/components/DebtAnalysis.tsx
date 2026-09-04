import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import LoadingSpinner from './LoadingSpinner'
import DebtPayModal from './DebtPayModal'
import Select from './Select'
import type { DebtBill } from '../types'

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

function formatMoney(code: string, amount: number) {
  return `${code} ${amount.toFixed(2)}`
}

export default function DebtAnalysis() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [bills, setBills] = useState<DebtBill[]>([])
  const [loading, setLoading] = useState(true)
  const [payTarget, setPayTarget] = useState<DebtBill | null>(null)
  const [currencyFilter, setCurrencyFilter] = useState<string>('all')
  const [search, setSearch] = useState('')

  const isCurrent = year === now.getFullYear() && month === now.getMonth() + 1

  const load = useCallback(async (y: number, m: number) => {
    setLoading(true)
    try {
      const list = await api.debts.billsByMonth(y, m)
      setBills(list || [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load(year, month)
  }, [year, month, load])

  const goPrev = () => {
    if (month === 1) {
      setYear((y) => y - 1)
      setMonth(12)
    } else {
      setMonth((m) => m - 1)
    }
  }
  const goNext = () => {
    if (month === 12) {
      setYear((y) => y + 1)
      setMonth(1)
    } else {
      setMonth((m) => m + 1)
    }
  }
  const goCurrent = () => {
    setYear(now.getFullYear())
    setMonth(now.getMonth() + 1)
  }

  const currencyOptions = useMemo(() => {
    const codes = [...new Set(bills.map((b) => b.currency_code || 'NIO'))].sort()
    return [
      { value: 'all', label: t('deudas.analysis_all_currencies') },
      ...codes.map((c) => ({ value: c, label: c })),
    ]
  }, [bills, t])

  const filteredBills = useMemo(
    () => (currencyFilter === 'all' ? bills : bills.filter((b) => (b.currency_code || 'NIO') === currencyFilter)),
    [bills, currencyFilter]
  )

  const searchResults = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return filteredBills
    return filteredBills.filter(
      (b) =>
        (b.debt_description || '').toLowerCase().includes(q) ||
        (b.institution_name || '').toLowerCase().includes(q) ||
        String(b.installment_number).includes(q)
    )
  }, [filteredBills, search])

  const totals = useMemo(() => {
    const byCurrency = new Map<string, { total: number; paid: number; pending: number }>()
    for (const b of filteredBills) {
      const cur = b.currency_code || 'NIO'
      const entry = byCurrency.get(cur) || { total: 0, paid: 0, pending: 0 }
      entry.total += b.amount
      if (b.status === 'paid') entry.paid += b.amount
      else entry.pending += b.amount
      byCurrency.set(cur, entry)
    }
    return byCurrency
  }, [filteredBills])

  const totalAll = useMemo(() => [...totals.values()].reduce((s, e) => s + e.total, 0), [totals])
  const paidAll = useMemo(() => [...totals.values()].reduce((s, e) => s + e.paid, 0), [totals])
  const pendingAll = useMemo(() => [...totals.values()].reduce((s, e) => s + e.pending, 0), [totals])
  const paidPct = totalAll > 0 ? Math.round((paidAll / totalAll) * 100) : 0
  const mainCurrency = [...totals.entries()].sort((a, b) => b[1].pending - a[1].pending)[0]?.[0] || ''
  const displayCurrency = currencyFilter !== 'all' ? currencyFilter : mainCurrency

  const byDebt = useMemo(() => {
    const map = new Map<string, { total: number; paid: number; pending: number }>()
    for (const b of filteredBills) {
      const key = b.debt_description || `#${b.debt_id}`
      const entry = map.get(key) || { total: 0, paid: 0, pending: 0 }
      entry.total += b.amount
      if (b.status === 'paid') entry.paid += b.amount
      else entry.pending += b.amount
      map.set(key, entry)
    }
    return [...map.entries()]
      .map(([name, v]) => ({ name, ...v }))
      .sort((a, b) => b.pending - a.pending)
  }, [filteredBills])

  const donutData = useMemo(() => {
    const totalPending = byDebt.reduce((s, d) => s + d.pending, 0)
    if (totalPending === 0) return []
    const top = byDebt.filter((d) => d.pending > 0).slice(0, 6)
    const othersPending = byDebt.filter((d) => d.pending > 0).slice(6).reduce((s, d) => s + d.pending, 0)
    const items = top.map((d) => ({ label: d.name, value: d.pending }))
    if (othersPending > 0) items.push({ label: t('deudas.analysis_others'), value: othersPending })
    let acc = 0
    return items.map((item, i) => {
      const start = (acc / totalPending) * 360
      acc += item.value
      const end = (acc / totalPending) * 360
      return { ...item, color: PALETTE[i % PALETTE.length], startAngle: start, endAngle: end }
    })
  }, [byDebt, t])

  const periodLabel = `${t(`months.${month}`)} ${year}`

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <div className="flex items-center gap-2 flex-wrap">
          <div className="flex items-center gap-1 bg-card rounded-ios shadow-ios px-2 py-1">
            <button onClick={goPrev} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors" aria-label="prev">
              <Icon name="chevron" className="w-4 h-4 rotate-180" />
            </button>
            <span className="text-sm font-semibold min-w-28 text-center whitespace-nowrap">{periodLabel}</span>
            <button onClick={goNext} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors" aria-label="next">
              <Icon name="chevron" className="w-4 h-4" />
            </button>
          </div>
          {currencyOptions.length > 1 && (
            <div className="w-44">
              <Select options={currencyOptions} value={currencyFilter} onChange={(v) => setCurrencyFilter(String(v))} />
            </div>
          )}
        </div>
        {!isCurrent && (
          <button
            onClick={goCurrent}
            className="flex items-center gap-1 text-sm font-medium text-primary hover:underline min-h-[44px]"
          >
            <Icon name="refresh" className="w-3 h-3" />
            {t('deudas.analysis_current_month')}
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
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.analysis_empty')}</h3>
          <p className="text-text-secondary">{t('deudas.analysis_empty_subtitle')}</p>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
            <div className="flex flex-wrap items-end justify-between gap-3 mb-3">
              <div>
                <p className="text-xs font-medium uppercase tracking-wide text-text-secondary">{t('deudas.analysis_needed')}</p>
                <p className="text-2xl sm:text-3xl font-bold text-amber-500 dark:text-amber-400">
                  {[...totals.entries()].map(([cur, e]) => (
                    <span key={cur} className="mr-3">{formatMoney(cur, e.pending)}</span>
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
                <div
                  className="h-full rounded-full bg-success transition-all"
                  style={{ width: `${paidPct}%` }}
                />
              </div>
              <span className="text-sm font-semibold text-emerald-600 dark:text-emerald-400">{paidPct}%</span>
            </div>
            <p className="text-xs text-text-secondary mt-1.5">{t('deudas.analysis_paid_of_month')}</p>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
            <SummaryCard
              label={t('deudas.analysis_total_due')}
              lines={[...totals.entries()].map(([cur, e]) => formatMoney(cur, e.total))}
              color="text-text"
            />
            <SummaryCard
              label={t('deudas.analysis_paid')}
              lines={[...totals.entries()].map(([cur, e]) => formatMoney(cur, e.paid))}
              color="text-emerald-600 dark:text-emerald-400"
            />
            <SummaryCard
              label={t('deudas.analysis_pending')}
              lines={[...totals.entries()].map(([cur, e]) => formatMoney(cur, e.pending))}
              color="text-amber-500 dark:text-amber-400"
            />
          </div>

          {donutData.length > 0 && (
            <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
              <h3 className="font-semibold mb-1">{t('deudas.analysis_by_debt')}</h3>
              <p className="text-xs text-text-secondary mb-4">{t('deudas.analysis_by_debt_subtitle')}</p>
              <div className="flex flex-col sm:flex-row items-center gap-6">
                <div className="relative shrink-0">
                  <svg width="180" height="180" viewBox="0 0 180 180" className="block">
                    <circle cx="90" cy="90" r="72" fill="none" stroke="var(--color-border, #e5e5ea)" strokeWidth="22" />
                    {donutData.map((seg) => (
                      <path
                        key={seg.label}
                        d={donutWedge(90, 90, 72, 50, seg.startAngle, seg.endAngle)}
                        fill={seg.color}
                      />
                    ))}
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                    <span className="text-xs text-text-secondary">{t('deudas.analysis_pending')}</span>
                    <span className="text-lg font-bold">{displayCurrency}{' '}{pendingAll.toFixed(2)}</span>
                  </div>
                </div>
                <div className="flex-1 w-full min-w-0">
                  <div className="space-y-2.5">
                    {donutData.map((seg) => (
                      <div key={seg.label} className="flex items-center gap-3">
                        <span className="w-3 h-3 rounded-full shrink-0" style={{ backgroundColor: seg.color }} />
                        <span className="flex-1 text-sm truncate">{seg.label}</span>
                        <span className="text-sm font-semibold shrink-0">
                          {displayCurrency}{' '}{(seg.value || 0).toFixed(2)}
                          <span className="text-text-secondary font-normal"> ({Math.round(((seg.value || 0) / Math.max(1, donutData.reduce((s, d) => s + d.value, 0))) * 100)}%)</span>
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}

          {byDebt.length > 0 && (
            <div className="bg-card rounded-ios shadow-ios p-4 sm:p-5">
              <h3 className="font-semibold mb-4">{t('deudas.analysis_per_debt')}</h3>
              <div className="space-y-4">
                {byDebt.map((d) => {
                  const total = d.total || 1
                  const paidPctDebt = Math.round((d.paid / total) * 100)
                  const pendingPctDebt = Math.round((d.pending / total) * 100)
                  return (
                    <div key={d.name}>
                      <div className="flex items-center justify-between gap-3 mb-1.5">
                        <span className="text-sm font-medium truncate">{d.name}</span>
                        <span className="text-sm font-semibold shrink-0">{formatMoney(displayCurrency, total)}</span>
                      </div>
                      <div className="h-2 rounded-full bg-border overflow-hidden flex">
                        <div className="h-full bg-success" style={{ width: `${paidPctDebt}%` }} />
                        <div className="h-full bg-amber-400" style={{ width: `${pendingPctDebt}%` }} />
                      </div>
                      <div className="flex items-center justify-between gap-3 mt-1 text-xs text-text-secondary">
                        <span>{t('deudas.analysis_paid')}: {formatMoney(displayCurrency, d.paid)}</span>
                        <span>{t('deudas.analysis_pending')}: {formatMoney(displayCurrency, d.pending)}</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          <div>
            <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">{t('deudas.analysis_installments_title')}</h3>
            <div className="relative mb-2">
              <Icon name="search" className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-secondary pointer-events-none" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('deudas.analysis_search_placeholder')}
                className="w-full pl-9 pr-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px]"
              />
            </div>
            <div className="space-y-2">
              {searchResults.map((bill) => (
                <div key={bill.id} className="bg-card rounded-ios shadow-ios p-3 sm:p-4">
                  <div className="flex items-center justify-between gap-3">
                    <button
                      onClick={() => navigate(`/deudas/${bill.debt_id}`)}
                      className="flex-1 min-w-0 flex items-center gap-3 text-left min-h-[44px] group"
                    >
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium truncate group-hover:text-primary transition-colors">{bill.debt_description}</p>
                        <p className="text-xs text-text-secondary">
                          {t('deudas.installment')} #{bill.installment_number} · {bill.institution_name} · {bill.due_date}
                        </p>
                        <p className="text-sm font-semibold mt-1">{formatMoney(bill.currency_code || 'NIO', bill.amount)}</p>
                      </div>
                      <Icon name="chevron" className="w-4 h-4 text-text-secondary shrink-0" />
                    </button>
                    <div className="flex flex-col items-end gap-2 shrink-0">
                      <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                        bill.status === 'paid' ? 'bg-success/20 text-green-800 dark:text-green-400' : 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
                      }`}>
                        {t(`bills.status_${bill.status}`)}
                      </span>
                      {bill.status === 'pending' && (
                        <button
                          onClick={() => setPayTarget(bill)}
                          className="text-xs font-semibold text-primary hover:underline min-h-[36px]"
                        >
                          {t('deudas.pay')}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
            {searchResults.length === 0 && (
              <p className="text-sm text-text-secondary mt-3">{t('deudas.analysis_no_results')}</p>
            )}
          </div>
        </div>
      )}

      {payTarget && (
        <DebtPayModal
          bill={payTarget}
          onClose={() => setPayTarget(null)}
          onSuccess={() => load(year, month)}
        />
      )}
    </div>
  )
}

function SummaryCard({ label, lines, color }: { label: string; lines: string[]; color: string }) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-3 sm:p-4">
      <p className="text-text-secondary text-xs font-medium uppercase tracking-wide">{label}</p>
      <div className={`mt-1 text-lg sm:text-xl font-bold ${color}`}>
        {lines.map((line) => (
          <p key={line}>{line}</p>
        ))}
      </div>
    </div>
  )
}