import { useEffect, useMemo, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import type { Debt, DebtBill } from '../types'
import {
  buildTimeline,
  lastPaymentDate,
  nextPendingBill,
  overdueBills,
  paidBills,
  payoffState,
  pendingEndDate,
  progressPct,
  remainingTime,
  type RemainingTime,
  type TimelinePoint,
} from '../utils/debtPayoff'
import { getSystemTimeZone, todayISO, todayInTimeZone } from '../utils/date'

interface DebtPayoffSummaryProps {
  debt: Debt
  bills: DebtBill[]
  symbol?: string
  lang: string
}

function formatDate(iso: string | null, lang: string, month: 'short' | 'long' = 'short'): string {
  if (!iso) return ''
  const [year, monthNumber, day] = iso.slice(0, 10).split('-').map(Number)
  if (!year || !monthNumber || !day) return iso
  const date = new Date(year, monthNumber - 1, day)
  try {
    return new Intl.DateTimeFormat(lang === 'en' ? 'en-US' : 'es-AR', {
      day: 'numeric',
      month,
      year: 'numeric',
    }).format(date)
  } catch {
    return iso
  }
}

function useTodayISO(): string {
  const [today, setToday] = useState(() => todayISO())

  useEffect(() => {
    let cancelled = false
    getSystemTimeZone()
      .then((tz) => {
        if (!cancelled) setToday(todayInTimeZone(tz))
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [])

  return today
}

function timelineLabel(point: TimelinePoint, t: (key: string) => string): string {
  if (point.key === 'start') return t('deudas.payoff_timeline_start')
  if (point.key === 'today') return t('deudas.payoff_timeline_today')
  if (point.key === 'next') {
    return point.label === 'overdue'
      ? t('deudas.payoff_timeline_overdue')
      : t('deudas.payoff_timeline_next')
  }
  return point.state === 'overdue'
    ? t('deudas.payoff_timeline_overdue_end')
    : t('deudas.payoff_timeline_end')
}

function dotClass(point: TimelinePoint): string {
  if (point.state === 'overdue') return 'bg-warning border-warning'
  if (point.key === 'today') return 'bg-card border-primary'
  if (point.key === 'next') return 'bg-card border-border'
  return 'bg-primary border-primary'
}

export default function DebtPayoffSummary({ debt, bills, symbol, lang }: DebtPayoffSummaryProps) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const today = useTodayISO()

  const data = useMemo(() => {
    const paid = paidBills(bills)
    const pending = bills.filter((b) => b.status === 'pending')
    const state = payoffState(debt, bills, today)
    const next = nextPendingBill(bills)
    const endDate = pendingEndDate(debt, bills)
    const lastPayment = lastPaymentDate(debt, bills)
    const overdues = overdueBills(bills, today)
    const effectiveTotal = Math.max(debt.installments_total || 0, bills.length)
    const pct = progressPct(paid.length, effectiveTotal, pending.length > 0)
    const paidAmount = paid.reduce((sum, bill) => sum + bill.amount, 0)
    const pendingAmount = pending.reduce((sum, bill) => sum + bill.amount, 0)
    const finalDate = state === 'paid' ? (lastPayment || endDate) : endDate
    const end = finalDate ? new Date(
      Number(finalDate.slice(0, 4)),
      Number(finalDate.slice(5, 7)) - 1,
      Number(finalDate.slice(8, 10)),
    ) : null
    const now = new Date(
      Number(today.slice(0, 4)),
      Number(today.slice(5, 7)) - 1,
      Number(today.slice(8, 10)),
    )
    const remaining = remainingTime(end, now)
    const timeline = buildTimeline({
      // Para horizontes largos, el inicio agrega ruido: la línea se enfoca en
      // hoy → próxima → última cuota.
      startDate: remaining.totalMonths > 36 ? null : debt.start_date,
      endDate: finalDate,
      nextDueDate: next?.due_date ?? null,
      todayISO: today,
    })

    return {
      paid,
      pending,
      state,
      next,
      lastPayment,
      overdues,
      effectiveTotal,
      pct,
      paidAmount,
      pendingAmount,
      finalDate,
      remaining,
      timeline,
    }
  }, [debt, bills, today])

  const headline = (() => {
    if (data.state === 'paid') return t('deudas.payoff_settled')
    if (data.state === 'missing-bills') return t('deudas.payoff_missing_bills')
    if (data.state === 'no-bills') return t('deudas.payoff_no_installments')
    if (data.state === 'paused') {
      const count = data.pending.length
      return count === 1
        ? t('deudas.payoff_paused_headline_one')
        : t('deudas.payoff_paused_headline').replace('{count}', String(count))
    }
    if (data.state === 'overdue') {
      if (!data.finalDate) return t('deudas.payoff_final_installment_unknown')
      if (data.finalDate < today) {
        return t('deudas.payoff_remaining_past').replace('{date}', formatDate(data.finalDate, lang, 'long'))
      }
      return t('deudas.payoff_catch_up').replace('{date}', formatDate(data.finalDate, lang, 'long'))
    }
    if (!data.finalDate) return t('deudas.payoff_final_installment_unknown')
    return remainingLabel(data.remaining, data.finalDate, lang, t)
  })()

  const subtitle = (() => {
    if (data.state === 'paid') {
      return data.lastPayment
        ? t('deudas.payoff_settled_on').replace('{date}', formatDate(data.lastPayment, lang, 'long'))
        : t('deudas.payoff_closed_manually')
    }
    if (data.state === 'missing-bills') return t('deudas.payoff_missing_bills_detail')
    if (data.state === 'no-bills') return t('deudas.payoff_no_installments_detail')
    if (data.state === 'paused') return t('deudas.payoff_paused_note')
    if (data.finalDate) {
      const isPast = data.finalDate < today
      if (isPast) return ''
      return `${t('deudas.payoff_final_installment')}: ${formatDate(data.finalDate, lang, 'long')} · ${t('deudas.payoff_final_installment_hint')}`
    }
    return t('deudas.payoff_final_installment_unknown')
  })()

  const isInconsistent =
    (debt.status === 'finalizada' && data.pending.length > 0) ||
    ((debt.pending_count ?? 0) === 0 && data.pending.length > 0)

  const badge = (() => {
    if (isInconsistent) {
      return { icon: 'warning', text: t('deudas.payoff_badge_review'), cls: 'bg-warning/20 text-text' }
    }
    if (data.state === 'paid') {
      return { icon: 'check', text: t('deudas.payoff_badge_settled'), cls: 'bg-success/20 text-text' }
    }
    if (data.state === 'overdue') {
      const count = data.overdues.length
      return {
        icon: 'warning',
        text: count === 1
          ? t('deudas.payoff_overdue_one').replace('{count}', '1')
          : t('deudas.payoff_overdue_many').replace('{count}', String(count)),
        cls: 'bg-warning/20 text-text',
      }
    }
    if (data.state === 'paused') {
      return { icon: 'clock', text: t('deudas.payoff_paused'), cls: 'bg-border text-text-secondary' }
    }
    if (data.state === 'missing-bills') {
      return { icon: 'warning', text: t('deudas.payoff_badge_review'), cls: 'bg-warning/20 text-text' }
    }
    if (data.state === 'no-bills') {
      return { icon: 'info', text: t('deudas.payoff_badge_no_bills'), cls: 'bg-border text-text-secondary' }
    }
    return { icon: 'check', text: t('deudas.payoff_on_track'), cls: 'bg-success/20 text-text' }
  })()

  const showProgress =
    data.effectiveTotal > 0 &&
    data.state !== 'paid' &&
    data.state !== 'missing-bills' &&
    data.state !== 'no-bills'

  const paidLabel = (() => {
    if (data.effectiveTotal === 1) {
      return data.paid.length === 1
        ? t('deudas.payoff_paid_single_done')
        : t('deudas.payoff_paid_of_one').replace('{paid}', String(data.paid.length))
    }
    return t('deudas.payoff_paid_of')
      .replace('{paid}', String(data.paid.length))
      .replace('{total}', String(data.effectiveTotal))
  })()

  const planIncomplete =
    bills.length > 0 && debt.installments_total > 0 && bills.length < debt.installments_total
  const completeButActive = data.state === 'paid' && debt.status !== 'finalizada'

  const showTimeline =
    !!data.finalDate &&
    data.timeline.length > 1 &&
    data.state !== 'paid' &&
    data.state !== 'missing-bills' &&
    data.state !== 'no-bills'

  const todayPosition = data.timeline.find((p) => p.key === 'today')?.position ?? 0
  const legendPoints = data.timeline.filter((p) => p.key !== 'next')
  const timelineSummary = data.timeline
    .map((p) => `${timelineLabel(p, t)}: ${formatDate(p.date, lang, 'long')}`)
    .join('; ')

  return (
    <section aria-labelledby="debt-payoff-title" className="bg-card rounded-ios shadow-ios p-4 sm:p-5 mb-4">
      <div>
        <div className="flex items-center justify-between gap-3">
          <p className="text-xs font-medium uppercase tracking-wide text-text-secondary">
            {t('deudas.payoff_eyebrow')}
          </p>
          <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold shrink-0 ${badge.cls}`}>
            <Icon name={badge.icon} className="w-3.5 h-3.5" />
            <span>{badge.text}</span>
          </span>
        </div>
        <h3 id="debt-payoff-title" className="mt-1 text-xl sm:text-2xl font-bold leading-tight text-text">
          {headline}
        </h3>
        {subtitle && <p className="mt-1 text-sm text-text-secondary break-words">{subtitle}</p>}
      </div>

      {data.state === 'paid' && (data.paidAmount > 0 || debt.paid_amount > 0) && (
        <p className="mt-3 text-sm text-text-secondary">
          {t('deudas.payoff_settled_total').replace(
            '{amount}',
            formatMoney(data.paidAmount || debt.paid_amount, symbol),
          )}
        </p>
      )}

      {showProgress && (
        <div className="mt-4">
          <div className="flex items-baseline justify-between gap-3">
            <p className="text-sm text-text">{paidLabel}</p>
            <span className="text-sm font-semibold tabular-nums text-text-secondary">{data.pct}%</span>
          </div>
          <div
            role="progressbar"
            aria-label={t('deudas.payoff_progress_label')}
            aria-valuemin={0}
            aria-valuemax={data.effectiveTotal}
            aria-valuenow={Math.min(data.paid.length, data.effectiveTotal)}
            aria-valuetext={t('deudas.payoff_aria_progress')
              .replace('{paid}', String(Math.min(data.paid.length, data.effectiveTotal)))
              .replace('{total}', String(data.effectiveTotal))
              .replace('{pct}', String(data.pct))}
            className="mt-2 h-2.5 overflow-hidden rounded-full bg-border"
          >
            <div className="h-full rounded-full bg-primary transition-[width] duration-300 motion-reduce:transition-none" style={{ width: `${data.pct}%` }} />
          </div>
          <div className="mt-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-1 text-xs text-text-secondary">
            <span>
              {t('deudas.progress_paid')}: {formatMoney(data.paidAmount, symbol)} ({data.paid.length})
            </span>
            <span>
              {t('deudas.payoff_remaining_balance')}: {formatMoney(data.pendingAmount, symbol)} ({data.pending.length})
            </span>
          </div>
          {planIncomplete && (
            <p className="mt-2 flex items-start gap-1.5 text-xs text-text-secondary">
              <Icon name="warning" className="w-3.5 h-3.5 mt-0.5 shrink-0 text-warning" />
              <span>
                {t('deudas.payoff_generated_of')
                  .replace('{generated}', String(bills.length))
                  .replace('{total}', String(debt.installments_total))}
                {' · '}
                {t('deudas.payoff_missing_count').replace(
                  '{count}',
                  String(debt.installments_total - bills.length),
                )}
              </span>
            </p>
          )}
        </div>
      )}

      {(data.state === 'active' || data.state === 'overdue' || data.state === 'paused') && (
        <dl className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
          <div className="min-w-0">
            <dt className="text-xs text-text-secondary">{t('deudas.payoff_last_payment')}</dt>
            <dd className="mt-0.5 text-sm font-medium text-text break-words">
              {data.lastPayment ? formatDate(data.lastPayment, lang) : t('deudas.payoff_no_payments')}
            </dd>
          </div>
          <div className="min-w-0">
            <dt className="text-xs text-text-secondary">
              {data.next && (data.next.due_date < today)
                ? t('deudas.payoff_overdue_installment')
                : t('deudas.payoff_next_installment')}
            </dt>
            <dd className="mt-0.5 text-sm font-medium text-text break-words">
              {data.next ? (
                <>
                  <span className="tabular-nums">
                    #{data.next.installment_number} · {formatDate(data.next.due_date, lang)}
                  </span>
                  <span className="block text-xs text-text-secondary tabular-nums">
                    {formatMoney(data.next.amount, symbol)}
                  </span>
                </>
              ) : (
                t('deudas.payoff_no_pending')
              )}
            </dd>
          </div>
          <div className="min-w-0 col-span-2 sm:col-span-1">
            <dt className="text-xs text-text-secondary">{t('deudas.payoff_final_installment')}</dt>
            <dd className="mt-0.5 text-sm font-medium text-text break-words">
              {data.finalDate ? formatDate(data.finalDate, lang, 'long') : t('deudas.payoff_final_installment_unknown')}
            </dd>
          </div>
        </dl>
      )}

      {showTimeline && (
        <div className="mt-4">
          <div className="relative h-6" aria-hidden="true">
            <div className="absolute left-0 right-0 top-3 h-0.5 bg-border" />
            <div className="absolute left-0 top-3 h-0.5 bg-primary" style={{ width: `${todayPosition}%` }} />
            {data.timeline.map((point) => (
              <span
                key={point.key}
                className={`absolute top-1.5 h-3 w-3 -translate-x-1/2 rounded-full border-2 ${dotClass(point)}`}
                style={{ left: `${Math.min(99, Math.max(1, point.position))}%` }}
              />
            ))}
          </div>
          <div className="mt-1 flex items-start justify-between gap-2 text-[10px] leading-tight text-text-secondary">
            {legendPoints.map((point, index) => {
              const align =
                legendPoints.length === 1
                  ? 'text-center'
                  : index === 0
                  ? 'text-left'
                  : index === legendPoints.length - 1
                  ? 'text-right'
                  : 'text-center'
              return (
                <span key={point.key} className={`min-w-0 flex-1 ${align}`}>
                  <span className="block font-medium text-text">{timelineLabel(point, t)}</span>
                  <span className="block tabular-nums">{formatDate(point.date, lang)}</span>
                </span>
              )
            })}
          </div>
          <p className="sr-only">{t('deudas.payoff_timeline_title')}: {timelineSummary}</p>
        </div>
      )}

      {isInconsistent && (
        <p className="mt-3 flex items-start gap-1.5 text-xs text-text-secondary">
          <Icon name="warning" className="w-3.5 h-3.5 mt-0.5 shrink-0 text-warning" />
          <span>{t('deudas.payoff_inconsistent_status')}</span>
        </p>
      )}

      {completeButActive && (
        <p className="mt-3 text-xs text-text-secondary">{t('deudas.payoff_complete_pending_reconcile')}</p>
      )}
    </section>
  )
}

function remainingLabel(
  remaining: RemainingTime,
  finalDate: string | null,
  lang: string,
  t: (key: string) => string,
): string {
  if (remaining.kind === 'past') {
    return t('deudas.payoff_remaining_past').replace('{date}', formatDate(finalDate, lang, 'long'))
  }
  if (remaining.kind === 'due-today') return t('deudas.payoff_remaining_due_today')
  if (remaining.kind === 'less-than-month') return t('deudas.payoff_remaining_less_than_month')

  const { years, months } = remaining
  if (years === 0) {
    if (months === 0) return t('deudas.payoff_remaining_less_than_month')
    if (months === 1) return t('deudas.payoff_remaining_one_month')
    return t('deudas.payoff_remaining_months').replace('{months}', String(months))
  }
  if (months === 0) {
    if (years === 1) return t('deudas.payoff_remaining_one_year')
    return t('deudas.payoff_remaining_years').replace('{years}', String(years))
  }
  if (years === 1 && months === 1) return t('deudas.payoff_remaining_one_year_one_month')
  if (years === 1) {
    return t('deudas.payoff_remaining_one_year_months').replace('{months}', String(months))
  }
  if (months === 1) {
    return t('deudas.payoff_remaining_years_one_month').replace('{years}', String(years))
  }
  return t('deudas.payoff_remaining_years_months')
    .replace('{years}', String(years))
    .replace('{months}', String(months))
}
