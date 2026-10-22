import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { useToast } from '../components/Toast'
import type { AutomationAccount } from '../types'

export default function AutomationPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  usePageTitle(t('automation.title'))
  const { showToast } = useToast()
  const [accounts, setAccounts] = useState<AutomationAccount[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteTarget, setDeleteTarget] = useState<AutomationAccount | null>(null)

  const load = async () => {
    const data = await api.automation.accounts()
    setAccounts(data || [])
  }

  useEffect(() => {
    load().finally(() => setLoading(false))
  }, [])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.automation.delete(deleteTarget.id)
      setDeleteTarget(null)
      showToast(t('automation.deleted'), 'success')
      await load()
    } catch (err) {
      const message = err instanceof Error ? err.message : ''
      showToast(/vinculad/i.test(message) ? t('automation.delete_in_use') : message || t('errors.generic'), 'error')
    }
  }

  if (loading) return <LoadingSpinner />

  if (accounts.length === 0) {
    return (
      <EmptyCard
        title={t('automation.empty')}
        subtitle={t('automation.empty_desc')}
        actionLabel={t('automation.create')}
        onAction={() => navigate('/automation/new')}
      />
    )
  }

  const createOptions = [
    { label: t('automation.create'), icon: 'plus', onClick: () => navigate('/automation/new') },
    { label: t('automation.plugin_catalog'), icon: 'bolt', onClick: () => navigate('/automation/plugins') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('automation.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {accounts.map((account) => (
          <div
            key={account.id}
            className="relative bg-card rounded-ios shadow-ios p-4 cursor-pointer hover:shadow-ios-lg transition-shadow"
            onClick={() => navigate(`/automation/${account.id}`)}
          >
            <CardMenu
              options={[
                { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/automation/edit/${account.id}`) },
                { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget(account) },
              ]}
            />
            <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
              <Icon name="bolt" className="w-6 h-6" />
            </div>
            <h3 className="font-semibold text-base truncate pr-8">
              {account.label || account.identifier}
            </h3>
            <p className="text-sm text-text-secondary mt-1 truncate">
              {account.institution_name || `#${account.institution_id}`} · {account.identifier}
            </p>
            <div className="flex flex-wrap gap-2 mt-3">
              <span className="text-xs px-2 py-1 rounded-full bg-bg text-text-secondary">
                {account.plugin_name || t('automation.no_plugins')}
              </span>
              <span className={`text-xs px-2 py-1 rounded-full ${
                account.schedule_enabled ? 'bg-success/10 text-success' : 'bg-bg text-text-secondary'
              }`}>
                {account.schedule_enabled ? scheduleShort(account, t) : t('automation.schedule_disabled')}
              </span>
            </div>
          </div>
        ))}
      </div>
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('automation.accounts')}: ${deleteTarget.label || deleteTarget.identifier}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function scheduleShort(account: AutomationAccount, t: (key: string) => string): string {
  const time = account.schedule_time || ''
  switch (account.schedule_frequency) {
    case 'every_n_days':
      return `${t('automation.interval')} ${account.schedule_interval ?? 1} ${t('automation.interval_days')} ${time}`
    case 'weekly':
      return `${t('automation.frequency_weekly')} ${time}`
    case 'monthly':
      return `${t('automation.frequency_monthly')} ${time}`
    default:
      return `${t('automation.frequency_daily')} ${time}`
  }
}

function EmptyCard({ title, subtitle, actionLabel, onAction }: {
  title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name="bolt" className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-sm text-text-secondary mb-5">{subtitle}</p>
      <button
        onClick={onAction}
        className="px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium hover:opacity-90 transition-opacity min-h-[44px]"
      >
        {actionLabel}
      </button>
    </div>
  )
}
