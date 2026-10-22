import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import type {
  AutomationAccount,
  AutomationBill,
  AutomationWebhookLog,
  Service,
} from '../types'

export default function AutomationDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const accountId = Number(id)
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [account, setAccount] = useState<AutomationAccount | null>(null)
  const [services, setServices] = useState<Service[]>([])
  const [bills, setBills] = useState<AutomationBill[]>([])
  const [logs, setLogs] = useState<AutomationWebhookLog[]>([])
  const [loading, setLoading] = useState(true)
  const [fetching, setFetching] = useState(false)
  const [syncing, setSyncing] = useState<number | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [revealOpen, setRevealOpen] = useState(false)

  usePageTitle(account ? (account.label || account.identifier) : t('automation.title'))

  const load = async () => {
    const [acc, billList, logList, serviceList] = await Promise.all([
      api.automation.get(accountId),
      api.automation.bills(accountId),
      api.automation.logs(accountId, 20, 0),
      api.services.list(),
    ])
    setAccount(acc)
    setBills(billList || [])
    setLogs(logList || [])
    setServices((serviceList || []).filter((svc) => svc.automation_account_id === accountId))
  }

  useEffect(() => {
    load().finally(() => setLoading(false))
  }, [accountId])

  const handleFetch = async () => {
    setFetching(true)
    try {
      const result = await api.automation.fetchBills(accountId)
      const count = result?.bills?.length ?? 0
      showToast(count > 0 ? t('automation.fetched_ok').replace('{count}', String(count)) : t('automation.fetched_empty'), 'success')
      const billList = await api.automation.bills(accountId)
      setBills(billList || [])
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('automation.fetch_error'), 'error')
    } finally {
      setFetching(false)
    }
  }

  const handleSync = async (serviceId: number) => {
    setSyncing(serviceId)
    try {
      const result = await api.services.sync(serviceId)
      if (result) {
        showToast(
          t('automation.synced')
            .replace('{delivered}', String(result.delivered))
            .replace('{failed}', String(result.failed)),
          result.failed > 0 ? 'info' : 'success'
        )
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSyncing(null)
    }
  }

  const handleDelete = async () => {
    try {
      await api.automation.delete(accountId)
      showToast(t('automation.deleted'), 'success')
      navigate('/automation')
    } catch (err) {
      const message = err instanceof Error ? err.message : ''
      setDeleteOpen(false)
      showToast(/vinculad/i.test(message) ? t('automation.delete_in_use') : message || t('errors.generic'), 'error')
    }
  }

  if (loading) return <LoadingSpinner />
  if (!account) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <p className="text-text-secondary">{t('errors.generic')}</p>
        <button
          onClick={() => navigate('/automation')}
          className="mt-4 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px]"
        >
          {t('automation.title')}
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu
          options={[
            { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/automation/edit/${account.id}`) },
            { label: t('automation.reveal'), icon: 'key', onClick: () => setRevealOpen(true) },
            {
              label: fetching ? t('automation.fetching') : t('automation.fetch'),
              icon: 'refresh',
              onClick: handleFetch,
            },
            { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteOpen(true) },
          ]}
        />
        <div className="min-w-0 pr-12">
          <h2 className="text-lg sm:text-xl font-bold truncate">{account.label || account.identifier}</h2>
          <p className="text-sm text-text-secondary mt-1">
            {account.institution_name || `#${account.institution_id}`} · {account.identifier}
          </p>
        </div>

        <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4 mt-5 text-sm">
          <div>
            <dt className="text-text-secondary">{t('automation.plugin')}</dt>
            <dd className="font-medium mt-0.5">{account.plugin_name || '—'}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t('automation.schedule')}</dt>
            <dd className="font-medium mt-0.5">{scheduleText(account, t)}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t('automation.last_run')}</dt>
            <dd className="font-medium mt-0.5">{formatDate(account.last_run_at) || t('automation.never')}</dd>
          </div>
        </dl>

      </div>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <h3 className="font-semibold mb-3">{t('automation.linked_services')}</h3>
        {services.length === 0 ? (
          <p className="text-sm text-text-secondary">{t('automation.no_linked_services')}</p>
        ) : (
          <div className="space-y-2">
            {services.map((svc) => (
              <div key={svc.id} className="flex items-center justify-between gap-3 bg-bg rounded-ios-sm px-3 py-2.5">
                <button
                  onClick={() => navigate(`/services/bills/${svc.id}`)}
                  className="min-w-0 text-left"
                >
                  <div className="font-medium text-sm truncate">{svc.name}</div>
                  <div className="text-xs text-text-secondary truncate">{svc.institution}</div>
                </button>
                <button
                  onClick={() => handleSync(svc.id)}
                  disabled={syncing === svc.id}
                  className="inline-flex items-center gap-2 px-3 py-2 rounded-ios-sm bg-primary/10 text-primary text-sm font-medium hover:bg-primary/20 transition-colors min-h-[40px] disabled:opacity-50 shrink-0"
                >
                  <Icon name="refresh" className={`w-4 h-4 ${syncing === svc.id ? 'animate-spin' : ''}`} />
                  {t('automation.sync')}
                </button>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <h3 className="font-semibold mb-3">{t('automation.bills')}</h3>
        {bills.length === 0 ? (
          <p className="text-sm text-text-secondary">{t('automation.no_bills')}</p>
        ) : (
          <div className="overflow-x-auto -mx-1">
            <table className="w-full text-sm min-w-[640px]">
              <thead>
                <tr className="text-left text-text-secondary border-b border-border">
                  <th className="py-2 px-1 font-medium">{t('automation.status')}</th>
                  <th className="py-2 px-1 font-medium">{t('automation.plugin')}</th>
                  <th className="py-2 px-1 font-medium">{t('automation.version')}</th>
                  <th className="py-2 px-1 font-medium">{t('automation.status_error')}</th>
                  <th className="py-2 px-1 font-medium">{t('automation.last_run')}</th>
                </tr>
              </thead>
              <tbody>
                {bills.map((bill) => (
                  <tr key={bill.id} className="border-b border-border/60 last:border-0">
                    <td className="py-2 px-1">
                      <StatusBadge status={bill.status} label={bill.status === 'ok' ? t('automation.status_ok') : t('automation.status_error')} />
                    </td>
                    <td className="py-2 px-1">{bill.plugin_name}</td>
                    <td className="py-2 px-1 text-text-secondary">{bill.plugin_version}</td>
                    <td className="py-2 px-1 text-text-secondary max-w-[220px] truncate" title={bill.error || ''}>
                      {bill.error || '—'}
                    </td>
                    <td className="py-2 px-1 text-text-secondary whitespace-nowrap">{formatDate(bill.fetched_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <h3 className="font-semibold mb-3">{t('automation.logs')}</h3>
        {logs.length === 0 ? (
          <p className="text-sm text-text-secondary">{t('automation.no_logs')}</p>
        ) : (
          <div className="overflow-x-auto -mx-1">
            <table className="w-full text-sm min-w-[560px]">
              <thead>
                <tr className="text-left text-text-secondary border-b border-border">
                  <th className="py-2 px-1 font-medium">{t('automation.status')}</th>
                  <th className="py-2 px-1 font-medium">{t('automation.status_error')}</th>
                  <th className="py-2 px-1 font-medium">{t('automation.last_run')}</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id} className="border-b border-border/60 last:border-0">
                    <td className="py-2 px-1">
                      <StatusBadge status={log.status} label={log.status === 'ok' ? t('automation.status_ok') : t('automation.status_error')} />
                    </td>
                    <td className="py-2 px-1 text-text-secondary max-w-[320px] truncate" title={log.response || ''}>
                      {log.response || '—'}
                    </td>
                    <td className="py-2 px-1 text-text-secondary whitespace-nowrap">{formatDate(log.ran_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {deleteOpen && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('automation.accounts')}: ${account.label || account.identifier}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteOpen(false)}
        />
      )}

      {revealOpen && (
        <RevealModal
          accountId={account.id}
          onClose={() => setRevealOpen(false)}
        />
      )}
    </div>
  )
}

function RevealModal({ accountId, onClose }: { accountId: number; onClose: () => void }) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [password, setPassword] = useState('')
  const [credentials, setCredentials] = useState('')
  const [loading, setLoading] = useState(false)

  const handleReveal = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!password) return
    setLoading(true)
    try {
      const result = await api.automation.revealCredentials(accountId, password)
      if (result?.credentials) {
        try {
          setCredentials(JSON.stringify(JSON.parse(result.credentials), null, 2))
        } catch {
          setCredentials(result.credentials)
        }
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('automation.reveal_error'), 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-md overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5">
          <h3 className="text-lg font-semibold mb-1">{t('automation.reveal')}</h3>
          <p className="text-sm text-text-secondary mb-4">{t('automation.reveal_desc')}</p>
          {credentials ? (
            <pre className="text-xs bg-bg rounded-ios-sm p-3 overflow-auto max-h-64 whitespace-pre-wrap break-all">
              {credentials}
            </pre>
          ) : (
            <form onSubmit={handleReveal} className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1">{t('automation.password')}</label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                  placeholder={t('automation.password_placeholder')}
                  autoComplete="current-password"
                  autoFocus
                />
              </div>
              <button
                type="submit"
                disabled={loading || !password}
                className="w-full px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium hover:opacity-90 transition-opacity min-h-[44px] disabled:opacity-50"
              >
                {loading ? t('app.loading') : t('automation.reveal')}
              </button>
            </form>
          )}
        </div>
        <div className="px-5 pb-5">
          <button
            onClick={onClose}
            className="w-full px-4 py-2.5 rounded-ios-sm border border-border hover:bg-bg transition-colors min-h-[44px]"
          >
            {t('app.close')}
          </button>
        </div>
      </div>
    </div>
  )
}

function StatusBadge({ status, label }: { status: string; label: string }) {
  const ok = status === 'ok'
  return (
    <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${
      ok ? 'bg-success/10 text-success' : 'bg-danger/10 text-danger'
    }`}>
      {label}
    </span>
  )
}

function scheduleText(account: AutomationAccount, t: (key: string, fallback?: string) => string): string {
  if (!account.schedule_enabled || !account.schedule_time) return t('automation.schedule_disabled')
  const time = account.schedule_time
  switch (account.schedule_frequency) {
    case 'every_n_days':
      return `${t('automation.interval')} ${account.schedule_interval ?? 1} ${t('automation.interval_days')} · ${time}`
    case 'weekly': {
      const days = (account.schedule_days || '')
        .split(',')
        .map((part) => Number(part.trim()))
        .filter((day) => day >= 0 && day <= 6)
        .map((day) => t(`automation.weekday_${day}`))
        .join(', ')
      return `${t('automation.frequency_weekly')}${days ? ` (${days})` : ''} · ${time}`
    }
    case 'monthly':
      return `${t('automation.frequency_monthly')} · ${t('automation.day_of_month')} ${account.schedule_days || '—'} · ${time}`
    default:
      return `${t('automation.frequency_daily')} · ${time}`
  }
}

function formatDate(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
