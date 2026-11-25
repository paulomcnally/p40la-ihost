import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { bankAccountTypeLabel } from '../utils/bank'
import type { Bank, BankAccount } from '../types'

function formatDate(value: string | undefined, lang: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString(lang === 'en' ? 'en-US' : 'es-ES', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex flex-col gap-0.5 py-2.5 border-b border-border last:border-b-0">
      <span className="text-xs font-medium text-text-secondary uppercase tracking-wide">{label}</span>
      <span className={`text-sm text-text break-words ${mono ? 'font-mono' : ''}`}>{value || '—'}</span>
    </div>
  )
}

export default function BankAccountShowPage() {
  const { id, accountId } = useParams()
  const bankId = Number(id)
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  const { showToast } = useToast()
  const [bank, setBank] = useState<Bank | null>(null)
  const [account, setAccount] = useState<BankAccount | null>(null)
  const [loading, setLoading] = useState(true)
  const [deleteOpen, setDeleteOpen] = useState(false)

  usePageTitle(account?.alias || t('banks.account_detail'))

  const load = useCallback(async () => {
    if (!Number.isFinite(bankId) || !Number.isFinite(Number(accountId))) return
    try {
      const [bankDetail, accountDetail] = await Promise.all([
        api.banks.get(bankId),
        api.bankAccounts.get(Number(accountId)),
      ])
      setBank(bankDetail)
      setAccount(accountDetail)
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [accountId, bankId, showToast, t])

  useEffect(() => {
    load()
  }, [load])

  const handleDelete = useCallback(async () => {
    if (!account) return
    try {
      await api.bankAccounts.delete(account.id)
      showToast(t('banks.account_deleted'), 'success')
      navigate(`/bancos/${bankId}`)
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [account, bankId, navigate, showToast, t])

  if (loading) return <LoadingSpinner />

  if (!account) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <div className="w-14 h-14 mx-auto mb-4 text-text-secondary opacity-60">
          <Icon name="wallet" className="w-full h-full" />
        </div>
        <h3 className="text-lg font-semibold mb-2">{t('banks.account_not_found')}</h3>
        <p className="text-sm text-text-secondary mb-5">{t('banks.account_not_found_hint')}</p>
        <button
          type="button"
          onClick={() => navigate(`/bancos/${bankId}`)}
          className="inline-flex items-center justify-center px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px]"
        >
          {t('banks.accounts')}
        </button>
      </div>
    )
  }

  const configuredFormats = new Set((account.formats || []).map((format) => format.kind))

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu
          options={[
            {
              label: t('app.edit'),
              icon: 'edit',
              onClick: () => navigate(`/bancos/${bankId}/cuentas/${account.id}/edit`),
            },
            {
              label: t('banks.transfers'),
              icon: 'wallet',
              onClick: () => navigate(`/bancos/${bankId}/cuentas/${account.id}/transferencias`),
            },
            {
              label: t('app.delete'),
              icon: 'delete',
              danger: true,
              onClick: () => setDeleteOpen(true),
            },
          ]}
        />

        <div className="flex items-start gap-3 min-w-0 pr-12">
          <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <Icon name="wallet" className="w-6 h-6" />
          </div>
          <div className="min-w-0">
            <h2 className="text-lg sm:text-xl font-bold truncate">{account.alias}</h2>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 mt-1.5">
              <span
                className={`inline-block text-xs font-medium px-2 py-0.5 rounded-full ${
                  account.active
                    ? 'bg-success/10 text-success'
                    : 'bg-bg text-text-secondary'
                }`}
              >
                {account.active ? t('banks.active_status') : t('banks.inactive_status')}
              </span>
              {bank && <span className="text-sm text-text-secondary truncate">{bank.name}</span>}
            </div>
            <p className="text-xs text-text-secondary mt-1">{t('banks.status_edit_hint')}</p>
          </div>
        </div>
      </div>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <h3 className="font-semibold mb-2">{t('banks.account_info')}</h3>
        <div>
          <InfoRow label={t('banks.account_number_label')} value={account.account_number} mono />
          <InfoRow label={t('banks.iban')} value={account.iban?.trim() || '—'} mono />
          <InfoRow label={t('banks.account_type')} value={bankAccountTypeLabel(account.account_type, t)} />
          <InfoRow
            label={t('banks.currency')}
            value={account.currency_code || account.currency_symbol || '—'}
          />
          <InfoRow label={t('banks.beneficiary')} value={account.beneficiary} />
        </div>
      </section>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <h3 className="font-semibold mb-2">{t('banks.notes_section')}</h3>
        <p className="text-sm text-text whitespace-pre-wrap break-words">
          {account.notes?.trim() || t('banks.no_notes')}
        </p>
      </section>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <h3 className="font-semibold mb-2">{t('banks.formats_summary')}</h3>
        <div className="space-y-2">
          {(['transfer', 'ach'] as const).map((kind) => (
            <div key={kind} className="flex items-center justify-between gap-3 text-sm">
              <span className="text-text">{kind === 'ach' ? t('banks.ach') : t('banks.transfer')}</span>
              <span className={configuredFormats.has(kind) ? 'text-success' : 'text-text-secondary'}>
                {configuredFormats.has(kind) ? t('banks.format_configured') : t('banks.format_not_configured')}
              </span>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <h3 className="font-semibold mb-2">{t('banks.metadata_section')}</h3>
        <div>
          <InfoRow label={t('banks.created_at')} value={formatDate(account.created_at, lang)} />
          <InfoRow label={t('banks.updated_at')} value={formatDate(account.updated_at, lang)} />
        </div>
      </section>

      {deleteOpen && (
        <DeleteModal
          title={t('banks.delete_account_title')}
          subtitle={t('banks.delete_account_subtitle').replace('{name}', account.alias)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteOpen(false)}
        />
      )}
    </div>
  )
}
