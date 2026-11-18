import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../../../../stores/i18nStore'
import { usePageTitle } from '../../../../hooks/usePageTitle'
import { useToast } from '../../../../components/Toast'
import { api } from '../../../../api'
import { Icon } from '../../../../components/Icons'
import Select from '../../../../components/Select'
import Toggle from '../../../../components/Toggle'
import LoadingSpinner from '../../../../components/LoadingSpinner'
import IconPickerModal from '../../../../components/IconPickerModal'
import { CARD_COLORS, isCardColor } from '../../../../constants/cardColors'
import type { Bank, Currency } from '../../../../types'
import { creditCardsApi } from '../api'
import type { CardNetwork, CreditCard } from '../types'
import { CARD_NETWORKS, networkLabel } from '../utils'

const inputClass =
  'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]'

export default function CreditCardFormPage() {
  const { id, cardId } = useParams()
  const bankId = Number(id)
  const isNew = !cardId
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [bank, setBank] = useState<Bank | null>(null)
  const [currencies, setCurrencies] = useState<Currency[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [iconPickerOpen, setIconPickerOpen] = useState(false)

  const [alias, setAlias] = useState('')
  const [network, setNetwork] = useState<CardNetwork>('visa')
  const [last4, setLast4] = useState('')
  const [holder, setHolder] = useState('')
  const [currencyId, setCurrencyId] = useState<number | ''>('')
  const [secondaryId, setSecondaryId] = useState<number>(0)
  const [creditLimit, setCreditLimit] = useState(0)
  const [creditLimitUSD, setCreditLimitUSD] = useState(0)
  const [statementDay, setStatementDay] = useState(1)
  const [paymentDueDay, setPaymentDueDay] = useState(1)
  const [minPaymentPct, setMinPaymentPct] = useState(4)
  const [interestCurrentRate, setInterestCurrentRate] = useState(0)
  const [interestLateRate, setInterestLateRate] = useState(0)
  const [interestCurrentRateUSD, setInterestCurrentRateUSD] = useState(0)
  const [interestLateRateUSD, setInterestLateRateUSD] = useState(0)
  const [color, setColor] = useState('blue')
  const [icon, setIcon] = useState('credit')
  const [active, setActive] = useState(true)
  const [notes, setNotes] = useState('')

  usePageTitle(isNew ? t('creditCards.create') : t('creditCards.edit'))

  const applyCard = useCallback((card: CreditCard) => {
    setAlias(card.alias)
    setNetwork(card.network)
    setLast4(card.last4)
    setHolder(card.holder || '')
    setCurrencyId(card.currency_id)
    setSecondaryId(card.secondary_currency_id || 0)
    setCreditLimit(card.credit_limit)
    setCreditLimitUSD(card.credit_limit_usd)
    setStatementDay(card.statement_day)
    setPaymentDueDay(card.payment_due_day)
    setMinPaymentPct(card.min_payment_pct)
    setInterestCurrentRate(card.interest_current_rate)
    setInterestLateRate(card.interest_late_rate)
    setInterestCurrentRateUSD(card.interest_current_rate_usd)
    setInterestLateRateUSD(card.interest_late_rate_usd)
    setColor(isCardColor(card.color) ? card.color : 'blue')
    setIcon(card.icon || 'credit')
    setActive(card.active)
    setNotes(card.notes || '')
  }, [])

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const [bankDetail, currencyList, card] = await Promise.all([
          api.banks.get(bankId),
          api.currencies.list(),
          isNew ? Promise.resolve(null) : creditCardsApi.get(Number(cardId)),
        ])
        if (!mounted) return
        setBank(bankDetail)
        setCurrencies(currencyList || [])
        if (card) applyCard(card)
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
  }, [applyCard, bankId, cardId, isNew, showToast, t])

  const currencyOptions = useMemo(
    () => currencies.map((currency) => ({ value: currency.id, label: `${currency.code} - ${currency.name}` })),
    [currencies],
  )

  const secondaryOptions = useMemo(
    () => [
      { value: 0, label: t('creditCards.no_secondary_currency') },
      ...currencyOptions.filter((option) => option.value !== currencyId),
    ],
    [currencyId, currencyOptions, t],
  )

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!alias.trim() || !currencyId || !/^\d{4}$/.test(last4)) {
      showToast(t('creditCards.validation_required'), 'error')
      return
    }
    setSaving(true)
    try {
      const payload = {
        alias: alias.trim(),
        network,
        last4,
        holder: holder.trim(),
        currency_id: Number(currencyId),
        secondary_currency_id: secondaryId ? Number(secondaryId) : undefined,
        credit_limit: Number(creditLimit) || 0,
        credit_limit_usd: Number(creditLimitUSD) || 0,
        statement_day: Number(statementDay) || 1,
        payment_due_day: Number(paymentDueDay) || 1,
        min_payment_pct: Number(minPaymentPct) || 0,
        interest_current_rate: Number(interestCurrentRate) || 0,
        interest_late_rate: Number(interestLateRate) || 0,
        interest_current_rate_usd: Number(interestCurrentRateUSD) || 0,
        interest_late_rate_usd: Number(interestLateRateUSD) || 0,
        color,
        icon,
        active,
        notes: notes.trim(),
      }
      if (isNew) {
        const created = await creditCardsApi.create(bankId, payload)
        showToast(t('creditCards.card_created'), 'success')
        if (created) {
          navigate(`/bancos/${bankId}/tarjetas/${created.id}`, { replace: true })
        }
      } else {
        await creditCardsApi.update(Number(cardId), payload)
        showToast(t('creditCards.card_updated'), 'success')
        navigate(`/bancos/${bankId}/tarjetas/${cardId}`, { replace: true })
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  const cancelTo = isNew ? `/bancos/${bankId}?tab=tarjetas` : `/bancos/${bankId}/tarjetas/${cardId}`

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">
          {isNew ? t('creditCards.create') : t('creditCards.edit')}
        </h2>
        <p className="text-sm text-text-secondary mt-1">
          {bank?.name || ''}
          {bank?.short_code ? ` · ${bank.short_code}` : ''}
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
            {t('creditCards.info')}
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">
                {t('creditCards.alias')} <span className="text-danger">*</span>
              </label>
              <input
                type="text"
                value={alias}
                onChange={(e) => setAlias(e.target.value)}
                maxLength={100}
                className={inputClass}
                autoComplete="off"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.network')}</label>
              <Select
                options={CARD_NETWORKS.map((value) => ({ value, label: networkLabel(value, t) }))}
                value={network}
                onChange={(value) => setNetwork(value as CardNetwork)}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">
                {t('creditCards.last4')} <span className="text-danger">*</span>
              </label>
              <input
                type="text"
                inputMode="numeric"
                value={last4}
                onChange={(e) => setLast4(e.target.value.replace(/\D/g, '').slice(0, 4))}
                maxLength={4}
                className={`${inputClass} font-mono`}
                autoComplete="off"
                placeholder="0000"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.holder')}</label>
              <input
                type="text"
                value={holder}
                onChange={(e) => setHolder(e.target.value)}
                maxLength={150}
                className={inputClass}
                autoComplete="off"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">
                {t('creditCards.currency')} <span className="text-danger">*</span>
              </label>
              <Select
                options={currencyOptions}
                value={currencyId}
                onChange={(value) => {
                  setCurrencyId(Number(value))
                  if (Number(value) === secondaryId) setSecondaryId(0)
                }}
                searchable={currencyOptions.length > 8}
                searchPlaceholder={t('app.search')}
                emptyLabel={t('app.no_results')}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.secondary_currency')}</label>
              <Select
                options={secondaryOptions}
                value={secondaryId}
                onChange={(value) => setSecondaryId(Number(value))}
              />
            </div>
          </div>
        </section>

        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
            {t('creditCards.limits')}
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <NumberField label={t('creditCards.credit_limit')} value={creditLimit} onChange={setCreditLimit} />
            <NumberField label={t('creditCards.credit_limit_usd')} value={creditLimitUSD} onChange={setCreditLimitUSD} />
            <NumberField label={t('creditCards.statement_day')} value={statementDay} onChange={setStatementDay} min={1} max={31} />
            <NumberField label={t('creditCards.payment_due_day')} value={paymentDueDay} onChange={setPaymentDueDay} min={1} max={31} />
            <NumberField label={t('creditCards.min_payment_pct')} value={minPaymentPct} onChange={setMinPaymentPct} min={0} max={100} />
          </div>
        </section>

        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
            {t('creditCards.rates')}
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <NumberField label={t('creditCards.interest_current_rate')} value={interestCurrentRate} onChange={setInterestCurrentRate} />
            <NumberField label={t('creditCards.interest_late_rate')} value={interestLateRate} onChange={setInterestLateRate} />
            <NumberField label={t('creditCards.interest_current_rate_usd')} value={interestCurrentRateUSD} onChange={setInterestCurrentRateUSD} />
            <NumberField label={t('creditCards.interest_late_rate_usd')} value={interestLateRateUSD} onChange={setInterestLateRateUSD} />
          </div>
        </section>

        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
            {t('creditCards.appearance')}
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.color')}</label>
              <Select
                options={CARD_COLORS.map((value) => ({ value, label: t(`creditCards.colors.${value}`) }))}
                value={color}
                onChange={(value) => setColor(String(value))}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.icon')}</label>
              <button
                type="button"
                onClick={() => setIconPickerOpen(true)}
                className={`${inputClass} flex items-center gap-3`}
              >
                <Icon name={icon} className="w-5 h-5" />
                <span className="text-sm text-text-secondary">{icon}</span>
              </button>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('creditCards.notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              maxLength={500}
              rows={2}
              className={`${inputClass} resize-y`}
            />
          </div>
          <div className="flex items-center justify-between gap-3 min-h-[44px]">
            <span className="text-sm font-medium">{t('creditCards.active')}</span>
            <Toggle checked={active} onChange={setActive} ariaLabel={t('creditCards.active')} />
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

      <IconPickerModal
        isOpen={iconPickerOpen}
        selectedIcon={icon}
        onSelect={(value: string) => {
          setIcon(value)
          setIconPickerOpen(false)
        }}
        onClose={() => setIconPickerOpen(false)}
      />
    </div>
  )
}

function NumberField({
  label,
  value,
  onChange,
  min,
  max,
}: {
  label: string
  value: number
  onChange: (value: number) => void
  min?: number
  max?: number
}) {
  return (
    <div>
      <label className="block text-sm font-medium mb-1">{label}</label>
      <input
        type="number"
        value={value}
        min={min}
        max={max}
        step="0.01"
        onChange={(e) => onChange(e.target.value === '' ? 0 : Number(e.target.value))}
        className={inputClass}
      />
    </div>
  )
}
