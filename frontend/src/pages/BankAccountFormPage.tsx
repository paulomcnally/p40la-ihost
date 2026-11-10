import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import Select from '../components/Select'
import LoadingSpinner from '../components/LoadingSpinner'
import TransferFormatEditor from '../components/TransferFormatEditor'
import {
  BANK_ACCOUNT_TYPE_VALUES,
  BANK_TRANSFER_VARIABLE_FALLBACK,
  bankAccountTypeFromValue,
  bankAccountTypeLabel,
} from '../utils/bank'
import { DEBT_INFORMATION_VARIABLE_FALLBACK } from '../utils/debtInformation'
import type {
  Bank,
  BankAccount,
  BankAccountType,
  BankTransferKind,
  Currency,
  DebtInformationVariable,
} from '../types'

const inputClass =
  'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]'

export default function BankAccountFormPage() {
  const { id, accountId } = useParams()
  const bankId = Number(id)
  const isNew = !accountId
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [bank, setBank] = useState<Bank | null>(null)
  const [account, setAccount] = useState<BankAccount | null>(null)
  const [currencies, setCurrencies] = useState<Currency[]>([])
  const [variables, setVariables] = useState<DebtInformationVariable[]>([
    ...BANK_TRANSFER_VARIABLE_FALLBACK,
    ...DEBT_INFORMATION_VARIABLE_FALLBACK,
  ])
  const [alias, setAlias] = useState('')
  const [accountNumber, setAccountNumber] = useState('')
  const [accountType, setAccountType] = useState<BankAccountType>('savings')
  const [currencyId, setCurrencyId] = useState<number | ''>('')
  const [beneficiary, setBeneficiary] = useState('')
  const [notes, setNotes] = useState('')
  const [active, setActive] = useState(true)
  const [tab, setTab] = useState<BankTransferKind>('transfer')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  usePageTitle(isNew ? t('banks.create_account') : alias || t('banks.edit_account'))

  const loadAccount = useCallback(async () => {
    if (isNew) return
    try {
      const detail = await api.bankAccounts.get(Number(accountId))
      if (!detail) return
      setAccount(detail)
      setAlias(detail.alias)
      setAccountNumber(detail.account_number)
      setAccountType(detail.account_type)
      setCurrencyId(detail.currency_id)
      setBeneficiary(detail.beneficiary)
      setNotes(detail.notes || '')
      setActive(detail.active)
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [accountId, isNew, showToast, t])

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const [bankDetail, currencyList, variableList] = await Promise.all([
          api.banks.get(bankId),
          api.currencies.list(),
          api.bankTransferVariables.list().catch(() => null),
        ])
        if (!mounted) return
        setBank(bankDetail)
        setCurrencies(currencyList || [])
        if (variableList && variableList.length > 0) setVariables(variableList)
        await loadAccount()
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
  }, [bankId, loadAccount, showToast, t])

  const currencyOptions = useMemo(
    () => currencies.map((currency) => ({ value: currency.id, label: `${currency.code} - ${currency.name}` })),
    [currencies],
  )

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!alias.trim() || !accountNumber.trim() || !currencyId || !beneficiary.trim()) {
      showToast(t('banks.account_required'), 'error')
      return
    }
    setSaving(true)
    try {
      const payload = {
        alias: alias.trim(),
        account_number: accountNumber.trim(),
        account_type: accountType,
        currency_id: Number(currencyId),
        beneficiary: beneficiary.trim(),
        notes: notes.trim(),
        active,
      }
      if (isNew) {
        const created = await api.banks.createAccount(bankId, payload)
        showToast(t('banks.account_created'), 'success')
        if (created) {
          navigate(`/bancos/${bankId}/cuentas/${created.id}`, { replace: true })
        }
      } else {
        const updated = await api.bankAccounts.update(Number(accountId), payload)
        setAccount(updated)
        showToast(t('banks.account_updated'), 'success')
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const reloadAccount = useCallback(async () => {
    if (isNew) return
    try {
      const detail = await api.bankAccounts.get(Number(accountId))
      if (detail) setAccount(detail)
    } catch {
      // ignore
    }
  }, [accountId, isNew])

  if (loading) return <LoadingSpinner />

  const activeFormat = account?.formats?.find((format) => format.kind === tab)

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">
          {isNew ? t('banks.create_account') : t('banks.edit_account')}
        </h2>
        <p className="text-sm text-text-secondary mt-1">
          {bank?.name || ''}
          {bank?.short_code ? ` · ${bank.short_code}` : ''}
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">
                {t('banks.alias')} <span className="text-danger">*</span>
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
              <label className="block text-sm font-medium mb-1">
                {t('banks.account_number')} <span className="text-danger">*</span>
              </label>
              <input
                type="text"
                value={accountNumber}
                onChange={(e) => setAccountNumber(e.target.value)}
                maxLength={50}
                className={`${inputClass} font-mono`}
                autoComplete="off"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('banks.account_type')}</label>
              <Select
                options={BANK_ACCOUNT_TYPE_VALUES.map((value) => ({
                  value,
                  label: bankAccountTypeLabel(value, t),
                }))}
                value={accountType}
                onChange={(value) => setAccountType(bankAccountTypeFromValue(String(value)))}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">
                {t('banks.currency')} <span className="text-danger">*</span>
              </label>
              <Select
                options={currencyOptions}
                value={currencyId}
                onChange={(value) => setCurrencyId(Number(value))}
                searchable={currencyOptions.length > 8}
                searchPlaceholder={t('app.search')}
                emptyLabel={t('app.no_results')}
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">
              {t('banks.beneficiary')} <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={beneficiary}
              onChange={(e) => setBeneficiary(e.target.value)}
              maxLength={150}
              className={inputClass}
              autoComplete="off"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('banks.account_notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              maxLength={500}
              rows={2}
              className={`${inputClass} resize-y`}
            />
          </div>
          <label className="flex items-center gap-2 text-sm min-h-[44px]">
            <input
              type="checkbox"
              checked={active}
              onChange={(e) => setActive(e.target.checked)}
              className="w-4 h-4 accent-primary"
            />
            {t('banks.active_account')}
          </label>
        </section>

        <div className="flex flex-col sm:flex-row gap-3">
          <button
            type="button"
            onClick={() => navigate(`/bancos/${bankId}`)}
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

      {account && (
        <div className="mt-6 space-y-3 pb-6">
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
              {t('banks.formats')}
            </h3>
          </div>
          <div className="flex gap-2">
            {(['transfer', 'ach'] as BankTransferKind[]).map((value) => (
              <button
                key={value}
                type="button"
                onClick={() => setTab(value)}
                className={`px-3 py-2 rounded-ios-sm text-sm font-medium min-h-[44px] ${
                  tab === value ? 'bg-primary text-white' : 'bg-bg text-text hover:bg-border'
                }`}
              >
                {value === 'ach' ? t('banks.ach') : t('banks.transfer')}
              </button>
            ))}
          </div>
          <TransferFormatEditor
            key={tab}
            accountId={account.id}
            account={account}
            bank={bank}
            kind={tab}
            initialFormat={activeFormat}
            variables={variables}
            onSaved={reloadAccount}
          />
        </div>
      )}
    </div>
  )
}
