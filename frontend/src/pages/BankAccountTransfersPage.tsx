import { useCallback, useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import LoadingSpinner from '../components/LoadingSpinner'
import TransferFormatEditor from '../components/TransferFormatEditor'
import { BANK_TRANSFER_VARIABLE_FALLBACK } from '../utils/bank'
import { DEBT_INFORMATION_VARIABLE_FALLBACK } from '../utils/debtInformation'
import type {
  Bank,
  BankAccount,
  BankTransferKind,
  DebtInformationVariable,
} from '../types'

const TAB_VALUES: BankTransferKind[] = ['transfer', 'ach']

export default function BankAccountTransfersPage() {
  const { id, accountId } = useParams()
  const bankId = Number(id)
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [searchParams, setSearchParams] = useSearchParams()

  const tabParam = searchParams.get('tab')
  const tab: BankTransferKind = TAB_VALUES.includes(tabParam as BankTransferKind)
    ? (tabParam as BankTransferKind)
    : 'transfer'

  useEffect(() => {
    if (tabParam !== 'transfer' && tabParam !== 'ach') {
      const params = new URLSearchParams(searchParams)
      params.set('tab', 'transfer')
      setSearchParams(params, { replace: true })
    }
  }, [searchParams, setSearchParams, tabParam])

  const [bank, setBank] = useState<Bank | null>(null)
  const [account, setAccount] = useState<BankAccount | null>(null)
  const [variables, setVariables] = useState<DebtInformationVariable[]>([
    ...BANK_TRANSFER_VARIABLE_FALLBACK,
    ...DEBT_INFORMATION_VARIABLE_FALLBACK,
  ])
  const [loading, setLoading] = useState(true)

  usePageTitle(t('banks.transfers'))

  const loadAccount = useCallback(async () => {
    const detail = await api.bankAccounts.get(Number(accountId))
    if (detail) setAccount(detail)
    return detail
  }, [accountId])

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const [bankDetail, accountDetail, variableList] = await Promise.all([
          api.banks.get(bankId),
          api.bankAccounts.get(Number(accountId)),
          api.bankTransferVariables.list().catch(() => null),
        ])
        if (!mounted) return
        setBank(bankDetail)
        setAccount(accountDetail)
        if (variableList && variableList.length > 0) setVariables(variableList)
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
  }, [accountId, bankId, showToast, t])

  const setTab = (next: BankTransferKind) => {
    const params = new URLSearchParams(searchParams)
    params.set('tab', next)
    setSearchParams(params, { replace: true })
  }

  const handleSaved = useCallback(() => {
    loadAccount().catch(() => {
      // El editor ya muestra el toast de error; mantener la página usable.
    })
  }, [loadAccount])

  if (loading) return <LoadingSpinner />

  if (!account) {
    return (
      <p className="text-sm text-text-secondary text-center py-10">{t('banks.account_not_found')}</p>
    )
  }

  const activeFormat = account.formats?.find((format) => format.kind === tab)

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('banks.transfers')}</h2>
        <p className="text-sm text-text-secondary mt-1">
          {account.alias}
          {bank ? ` · ${bank.name}` : ''}
        </p>
      </div>

      <div
        role="tablist"
        aria-label={t('banks.transfers')}
        className="flex gap-2 mb-4 overflow-x-auto"
      >
        {TAB_VALUES.map((value) => (
          <button
            key={value}
            type="button"
            role="tab"
            aria-selected={tab === value}
            onClick={() => setTab(value)}
            className={`px-4 py-2 rounded-ios-sm text-sm font-medium min-h-[44px] whitespace-nowrap ${
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
        onSaved={handleSaved}
      />
    </div>
  )
}
