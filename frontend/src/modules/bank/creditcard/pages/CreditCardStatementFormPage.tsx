import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../../../../stores/i18nStore'
import { usePageTitle } from '../../../../hooks/usePageTitle'
import { useToast } from '../../../../components/Toast'
import { Icon } from '../../../../components/Icons'
import Select from '../../../../components/Select'
import LoadingSpinner from '../../../../components/LoadingSpinner'
import { creditCardsApi } from '../api'
import type { CardStatement, CreditCard, StatementStatus } from '../types'
import { STATEMENT_STATUSES, statementStatusLabel } from '../utils'

const inputClass =
  'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]'

export default function CreditCardStatementFormPage() {
  const { id, cardId, statementId } = useParams()
  const bankId = Number(id)
  const isNew = !statementId
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [card, setCard] = useState<CreditCard | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const [periodStart, setPeriodStart] = useState('')
  const [periodEnd, setPeriodEnd] = useState('')
  const [minDue, setMinDue] = useState('')
  const [fullDue, setFullDue] = useState('')
  const [previous, setPrevious] = useState(0)
  const [previousUSD, setPreviousUSD] = useState(0)
  const [interest, setInterest] = useState(0)
  const [interestUSD, setInterestUSD] = useState(0)
  const [bonifiable, setBonifiable] = useState(0)
  const [bonifiableUSD, setBonifiableUSD] = useState(0)
  const [mora, setMora] = useState(0)
  const [moraUSD, setMoraUSD] = useState(0)
  const [arrears, setArrears] = useState(0)
  const [overLimit, setOverLimit] = useState(0)
  const [overLimitUSD, setOverLimitUSD] = useState(0)
  const [minPayment, setMinPayment] = useState(0)
  const [minPaymentUSD, setMinPaymentUSD] = useState(0)
  const [totalPayment, setTotalPayment] = useState(0)
  const [totalPaymentUSD, setTotalPaymentUSD] = useState(0)
  const [newBalance, setNewBalance] = useState(0)
  const [newBalanceUSD, setNewBalanceUSD] = useState(0)
  const [status, setStatus] = useState<StatementStatus>('unpaid')
  const [notes, setNotes] = useState('')

  usePageTitle(isNew ? t('creditCards.create_statement') : t('creditCards.edit_statement'))

  const applyStatement = useCallback((statement: CardStatement) => {
    setPeriodStart(statement.period_start || '')
    setPeriodEnd(statement.period_end)
    setMinDue(statement.min_payment_due_date)
    setFullDue(statement.full_payment_due_date)
    setPrevious(statement.previous_balance)
    setPreviousUSD(statement.previous_balance_usd)
    setInterest(statement.interest_current)
    setInterestUSD(statement.interest_current_usd)
    setBonifiable(statement.interest_bonifiable)
    setBonifiableUSD(statement.interest_bonifiable_usd)
    setMora(statement.mora_amount)
    setMoraUSD(statement.mora_amount_usd)
    setArrears(statement.arrears_count)
    setOverLimit(statement.over_limit_amount)
    setOverLimitUSD(statement.over_limit_amount_usd)
    setMinPayment(statement.min_payment)
    setMinPaymentUSD(statement.min_payment_usd)
    setTotalPayment(statement.total_payment)
    setTotalPaymentUSD(statement.total_payment_usd)
    setNewBalance(statement.new_balance)
    setNewBalanceUSD(statement.new_balance_usd)
    setStatus(statement.status)
    setNotes(statement.notes || '')
  }, [])

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const [cardDetail, statement] = await Promise.all([
          creditCardsApi.get(Number(cardId)),
          isNew ? Promise.resolve(null) : creditCardsApi.getStatement(Number(statementId)),
        ])
        if (!mounted) return
        setCard(cardDetail)
        if (statement) applyStatement(statement)
      } catch (err) {
        if (mounted) showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      } finally {
        if (mounted) setLoading(false)
      }
    }
    load()
    return () => {
      mounted = false
    }
  }, [applyStatement, cardId, isNew, showToast, statementId, t])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!periodEnd || !minDue || !fullDue) {
      showToast(t('creditCards.validation_required'), 'error')
      return
    }
    setSaving(true)
    try {
      const payload = {
        period_start: periodStart || undefined,
        period_end: periodEnd,
        min_payment_due_date: minDue,
        full_payment_due_date: fullDue,
        previous_balance: Number(previous) || 0,
        previous_balance_usd: Number(previousUSD) || 0,
        interest_current: Number(interest) || 0,
        interest_current_usd: Number(interestUSD) || 0,
        interest_bonifiable: Number(bonifiable) || 0,
        interest_bonifiable_usd: Number(bonifiableUSD) || 0,
        mora_amount: Number(mora) || 0,
        mora_amount_usd: Number(moraUSD) || 0,
        arrears_count: Number(arrears) || 0,
        over_limit_amount: Number(overLimit) || 0,
        over_limit_amount_usd: Number(overLimitUSD) || 0,
        min_payment: Number(minPayment) || 0,
        min_payment_usd: Number(minPaymentUSD) || 0,
        total_payment: Number(totalPayment) || 0,
        total_payment_usd: Number(totalPaymentUSD) || 0,
        new_balance: Number(newBalance) || 0,
        new_balance_usd: Number(newBalanceUSD) || 0,
        status,
        notes: notes.trim(),
      }
      if (isNew) {
        const created = await creditCardsApi.createStatement(Number(cardId), payload)
        showToast(t('creditCards.statement_created'), 'success')
        if (created) {
          navigate(`/bancos/${bankId}/tarjetas/${cardId}/estados/${created.id}`, { replace: true })
        }
      } else {
        await creditCardsApi.updateStatement(Number(statementId), payload)
        showToast(t('creditCards.statement_updated'), 'success')
        navigate(`/bancos/${bankId}/tarjetas/${cardId}/estados/${statementId}`, { replace: true })
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  const cancelTo = isNew
    ? `/bancos/${bankId}/tarjetas/${cardId}`
    : `/bancos/${bankId}/tarjetas/${cardId}/estados/${statementId}`

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">
          {isNew ? t('creditCards.create_statement') : t('creditCards.edit_statement')}
        </h2>
        <p className="text-sm text-text-secondary mt-1">
          {card?.alias || ''}
          {card ? ` · •••• ${card.last4}` : ''}
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <DateField label={t('creditCards.statement_period_start')} value={periodStart} onChange={setPeriodStart} />
            <DateField label={t('creditCards.statement_period')} value={periodEnd} onChange={setPeriodEnd} required />
            <DateField label={t('creditCards.statement_due_min')} value={minDue} onChange={setMinDue} required />
            <DateField label={t('creditCards.statement_due_full')} value={fullDue} onChange={setFullDue} required />
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.statement_status')}</label>
              <Select
                options={STATEMENT_STATUSES.map((value) => ({
                  value,
                  label: statementStatusLabel(value, t),
                }))}
                value={status}
                onChange={(value) => setStatus(value as StatementStatus)}
              />
            </div>
          </div>
        </section>

        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
            {t('creditCards.amounts')}
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <NumberField label={t('creditCards.previous_balance')} value={previous} onChange={setPrevious} />
            <NumberField label={t('creditCards.previous_balance_usd')} value={previousUSD} onChange={setPreviousUSD} />
            <NumberField label={t('creditCards.interest_current')} value={interest} onChange={setInterest} />
            <NumberField label={t('creditCards.interest_current_usd')} value={interestUSD} onChange={setInterestUSD} />
            <NumberField label={t('creditCards.min_payment')} value={minPayment} onChange={setMinPayment} />
            <NumberField label={t('creditCards.min_payment_usd')} value={minPaymentUSD} onChange={setMinPaymentUSD} />
            <NumberField label={t('creditCards.total_payment')} value={totalPayment} onChange={setTotalPayment} />
            <NumberField label={t('creditCards.total_payment_usd')} value={totalPaymentUSD} onChange={setTotalPaymentUSD} />
            <NumberField label={t('creditCards.new_balance')} value={newBalance} onChange={setNewBalance} />
            <NumberField label={t('creditCards.new_balance_usd')} value={newBalanceUSD} onChange={setNewBalanceUSD} />
          </div>
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
            {t('creditCards.statement_aggregates')}
          </h3>
          <p className="text-xs text-text-secondary">{t('creditCards.statement_aggregates_help')}</p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <NumberField label={t('creditCards.interest_bonifiable')} value={bonifiable} onChange={setBonifiable} />
            <NumberField label={t('creditCards.interest_bonifiable_usd')} value={bonifiableUSD} onChange={setBonifiableUSD} />
            <NumberField label={t('creditCards.mora_amount')} value={mora} onChange={setMora} />
            <NumberField label={t('creditCards.mora_amount_usd')} value={moraUSD} onChange={setMoraUSD} />
            <NumberField label={t('creditCards.arrears_count')} value={arrears} onChange={setArrears} min={0} />
            <NumberField label={t('creditCards.over_limit_amount')} value={overLimit} onChange={setOverLimit} />
            <NumberField label={t('creditCards.over_limit_amount_usd')} value={overLimitUSD} onChange={setOverLimitUSD} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('creditCards.statement_notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              maxLength={500}
              rows={2}
              className={`${inputClass} resize-y`}
            />
          </div>
        </section>

        <div className="flex flex-col sm:flex-row gap-3">
          <button
            type="button"
            onClick={() => navigate(cancelTo)}
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            type="submit"
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            {saving ? (
              <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
            ) : (
              <Icon name="save" className="w-4 h-4" />
            )}
            {saving ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </form>
    </div>
  )
}

function DateField({
  label,
  value,
  onChange,
  required,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  required?: boolean
}) {
  return (
    <div>
      <label className="block text-sm font-medium mb-1">
        {label} {required && <span className="text-danger">*</span>}
      </label>
      <input type="date" value={value} onChange={(e) => onChange(e.target.value)} className={inputClass} required={required} />
    </div>
  )
}

function NumberField({
  label,
  value,
  onChange,
  min,
}: {
  label: string
  value: number
  onChange: (value: number) => void
  min?: number
}) {
  return (
    <div>
      <label className="block text-sm font-medium mb-1">{label}</label>
      <input
        type="number"
        step="0.01"
        value={value}
        onChange={(e) => onChange(e.target.value === '' ? 0 : Number(e.target.value))}
        className={inputClass}
      />
    </div>
  )
}
