import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import type { Bank } from '../types'

export default function BanksPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  usePageTitle(t('banks.title'))
  const { showToast } = useToast()
  const [banks, setBanks] = useState<Bank[]>([])
  const [deleteTarget, setDeleteTarget] = useState<Bank | null>(null)

  const load = useCallback(async () => {
    try {
      const list = await api.banks.list()
      setBanks(list || [])
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [showToast, t])

  useEffect(() => {
    load()
  }, [load])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await api.banks.delete(deleteTarget.id)
      setDeleteTarget(null)
      showToast(t('banks.bank_deleted'), 'success')
      load()
    } catch (err: unknown) {
      const raw = err instanceof Error ? err.message : ''
      const message = /cuentas|accounts/i.test(raw)
        ? t('banks.delete_has_accounts')
        : raw || t('errors.generic')
      showToast(message, 'error')
    }
  }, [deleteTarget, load, showToast, t])

  if (banks.length === 0) {
    return (
      <EmptyCard
        icon="bank"
        title={t('banks.empty_title')}
        subtitle={t('banks.empty_subtitle')}
        actionLabel={t('banks.create')}
        onAction={() => navigate('/bancos/new')}
      />
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <div>
          <h2 className="text-xl sm:text-2xl font-bold">{t('banks.title')}</h2>
          <p className="text-sm text-text-secondary mt-1">{t('banks.subtitle')}</p>
        </div>
        <CreateMenu
          options={[
            { label: t('banks.create'), icon: 'plus', onClick: () => navigate('/bancos/new') },
          ]}
        />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {banks.map((bank) => (
          <div
            key={bank.id}
            onClick={() => navigate(`/bancos/${bank.id}`)}
            className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
          >
            <CardMenu
              options={[
                { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/bancos/edit/${bank.id}`) },
                {
                  label: t('app.delete'),
                  icon: 'delete',
                  danger: true,
                  onClick: () => setDeleteTarget(bank),
                },
              ]}
            />
            <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
              <Icon name="bank" className="w-6 h-6" />
            </div>
            <h3 className="font-semibold text-base truncate">{bank.name}</h3>
            {bank.short_code && (
              <span className="inline-block mt-1 text-xs font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                {bank.short_code}
              </span>
            )}
            <p className="text-sm text-text-secondary mt-1">
              {bank.country || t('banks.no_country')}
            </p>
            <p className="text-sm text-text-secondary mt-1">
              {bank.account_count === 1
                ? t('banks.account_count_one')
                : t('banks.account_count_other').replace('{count}', String(bank.account_count))}
            </p>
          </div>
        ))}
      </div>

      {deleteTarget && (
        <DeleteModal
          title={t('banks.delete_title')}
          subtitle={t('banks.delete_subtitle').replace('{name}', deleteTarget.name)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function EmptyCard({ icon, title, subtitle, actionLabel, onAction }: {
  icon: string; title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name={icon} className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-text-secondary mb-6">{subtitle}</p>
      <button
        onClick={onAction}
        className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
      >
        <Icon name="plus" className="w-5 h-5" />
        {actionLabel}
      </button>
    </div>
  )
}
