import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { bankAccountTypeLabel } from '../utils/bank'
import type { Bank, BankAccount } from '../types'

export default function BankDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [bank, setBank] = useState<Bank | null>(null)
  const [accounts, setAccounts] = useState<BankAccount[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteTarget, setDeleteTarget] = useState<BankAccount | null>(null)

  usePageTitle(bank?.name || t('banks.title'))

  const load = useCallback(async () => {
    if (!id) return
    try {
      const [bankDetail, accountList] = await Promise.all([
        api.banks.get(Number(id)),
        api.banks.accounts(Number(id)),
      ])
      setBank(bankDetail)
      setAccounts(accountList || [])
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [id, showToast, t])

  useEffect(() => {
    load()
  }, [load])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await api.bankAccounts.delete(deleteTarget.id)
      setDeleteTarget(null)
      showToast(t('banks.account_deleted'), 'success')
      load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [deleteTarget, load, showToast, t])

  if (loading) return <LoadingSpinner />

  if (!bank) {
    return (
      <p className="text-sm text-text-secondary text-center py-10">{t('banks.empty_title')}</p>
    )
  }

  return (
    <div>
      <div className="flex items-start justify-between gap-3 mb-4 sm:mb-5">
        <div>
          <h2 className="text-xl sm:text-2xl font-bold">{bank.name}</h2>
          <p className="text-sm text-text-secondary mt-1">
            {bank.short_code ? `${bank.short_code} · ` : ''}
            {bank.country || t('banks.no_country')}
          </p>
          {bank.notes && <p className="text-sm text-text-secondary mt-1">{bank.notes}</p>}
        </div>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => navigate(`/bancos/edit/${bank.id}`)}
            className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
            title={t('app.edit')}
            aria-label={t('app.edit')}
          >
            <Icon name="edit" className="w-5 h-5" />
          </button>
          <CreateMenu
            options={[
              {
                label: t('banks.create_account'),
                icon: 'plus',
                onClick: () => navigate(`/bancos/${bank.id}/cuentas/new`),
              },
            ]}
          />
        </div>
      </div>

      <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide mb-2">
        {t('banks.accounts')}
      </h3>

      {accounts.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-4">
          <div className="w-14 h-14 mx-auto mb-4 text-primary opacity-80">
            <Icon name="bank" className="w-full h-full" />
          </div>
          <h4 className="text-lg font-semibold mb-2">{t('banks.empty_accounts_title')}</h4>
          <p className="text-text-secondary mb-5">{t('banks.empty_accounts_subtitle')}</p>
          <button
            onClick={() => navigate(`/bancos/${bank.id}/cuentas/new`)}
            className="inline-flex items-center justify-center gap-2 px-5 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 min-h-[44px]"
          >
            <Icon name="plus" className="w-4 h-4" />
            {t('banks.create_account')}
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
          {accounts.map((account) => (
            <div
              key={account.id}
              onClick={() => navigate(`/bancos/${bank.id}/cuentas/${account.id}`)}
              className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
            >
              <CardMenu
                options={[
                  {
                    label: t('app.edit'),
                    icon: 'edit',
                    onClick: () => navigate(`/bancos/${bank.id}/cuentas/${account.id}`),
                  },
                  {
                    label: t('app.delete'),
                    icon: 'delete',
                    danger: true,
                    onClick: () => setDeleteTarget(account),
                  },
                ]}
              />
              <div className="w-10 h-10 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
                <Icon name="wallet" className="w-5 h-5" />
              </div>
              <h4 className="font-semibold truncate">{account.alias}</h4>
              <p className="text-sm text-text-secondary font-mono truncate">{account.account_number}</p>
              <p className="text-sm text-text-secondary mt-1">
                {bankAccountTypeLabel(account.account_type, t)}
                {account.currency_code ? ` · ${account.currency_code}` : ''}
              </p>
              <p className="text-sm text-text-secondary mt-1 truncate">{account.beneficiary}</p>
            </div>
          ))}
        </div>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('banks.delete_account_title')}
          subtitle={t('banks.delete_account_subtitle').replace('{name}', deleteTarget.alias)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
