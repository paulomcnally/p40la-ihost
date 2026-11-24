import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { useToast } from '../components/Toast'
import { copyToClipboard } from '../utils/clipboard'
import ConfirmWordModal from '../components/ConfirmWordModal'
import { Icon } from '../components/Icons'
import StatusPill from '../components/StatusPill'

interface KeyStatus {
  configured: boolean
  prefix?: string
  created_at?: string
  last_used_at?: string
}

function formatDate(value: string | undefined, lang: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString(lang === 'en' ? 'en-US' : 'es-NI', {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

export default function SettingsMCPPage() {
  const { t, lang } = useI18nStore()
  usePageTitle(t('settings.mcp.page_title'))
  const { showToast } = useToast()

  const [status, setStatus] = useState<KeyStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [newKey, setNewKey] = useState('')
  const [showKey, setShowKey] = useState(false)
  const [confirmAction, setConfirmAction] = useState<'regenerate' | 'revoke' | null>(null)

  const load = async () => {
    try {
      const data = await api.mcpAPIKey.get()
      setStatus(data ?? { configured: false })
    } catch {
      showToast(t('settings.mcp.error'), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleGenerate = async () => {
    setBusy(true)
    try {
      const res = await api.mcpAPIKey.regenerate()
      if (!res?.api_key) {
        showToast(t('settings.mcp.error'), 'error')
        return
      }
      setNewKey(res.api_key)
      setShowKey(true)
      setStatus({ configured: true, prefix: res.prefix, created_at: res.created_at })
      showToast(t('settings.mcp.generated'), 'success')
    } catch {
      showToast(t('settings.mcp.error'), 'error')
    } finally {
      setBusy(false)
      setConfirmAction(null)
    }
  }

  const handleRevoke = async () => {
    setBusy(true)
    try {
      await api.mcpAPIKey.revoke()
      setStatus({ configured: false })
      showToast(t('settings.mcp.revoked'), 'success')
    } catch {
      showToast(t('settings.mcp.error'), 'error')
    } finally {
      setBusy(false)
      setConfirmAction(null)
    }
  }

  const handleCopy = async () => {
    try {
      await copyToClipboard(newKey)
      showToast(t('settings.mcp.copied'), 'success')
    } catch {
      showToast(t('settings.mcp.error'), 'error')
    }
  }

  const closeKey = () => {
    setShowKey(false)
    setNewKey('')
  }

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto">
        <div className="bg-card rounded-ios shadow-ios p-6 space-y-3 animate-pulse">
          <div className="h-5 w-40 bg-border rounded" />
          <div className="h-4 w-full bg-border rounded" />
          <div className="h-10 w-48 bg-border rounded" />
        </div>
      </div>
    )
  }

  const configured = status?.configured ?? false

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="font-medium">{t('settings.mcp.title')}</div>
            <div className="text-sm text-text-secondary">{t('settings.mcp.description')}</div>
          </div>
          <StatusPill
            label={configured ? t('settings.mcp.status_configured') : t('settings.mcp.status_missing')}
            tone={configured ? 'success' : 'muted'}
          />
        </div>

        {configured ? (
          <>
            <dl className="divide-y divide-border">
              <div className="px-4 py-3 flex items-center justify-between gap-3">
                <dt className="text-sm text-text-secondary">{t('settings.mcp.prefix')}</dt>
                <dd className="font-mono text-sm truncate">{status?.prefix || '—'}</dd>
              </div>
              <div className="px-4 py-3 flex items-center justify-between gap-3">
                <dt className="text-sm text-text-secondary">{t('settings.mcp.created_at')}</dt>
                <dd className="text-sm">{formatDate(status?.created_at, lang)}</dd>
              </div>
              <div className="px-4 py-3 flex items-center justify-between gap-3">
                <dt className="text-sm text-text-secondary">{t('settings.mcp.last_used_at')}</dt>
                <dd className="text-sm">
                  {status?.last_used_at ? formatDate(status.last_used_at, lang) : t('settings.mcp.never')}
                </dd>
              </div>
            </dl>
            <div className="px-4 py-4 flex flex-col sm:flex-row gap-2">
              <button
                type="button"
                onClick={() => setConfirmAction('regenerate')}
                disabled={busy}
                className="px-4 py-2.5 rounded-ios-sm bg-bg border border-border font-medium min-h-[44px] hover:border-primary/50 transition-colors disabled:opacity-50"
              >
                {t('settings.mcp.regenerate')}
              </button>
              <button
                type="button"
                onClick={() => setConfirmAction('revoke')}
                disabled={busy}
                className="px-4 py-2.5 rounded-ios-sm bg-danger/10 text-danger border border-danger/30 font-medium min-h-[44px] disabled:opacity-50"
              >
                {t('settings.mcp.revoke')}
              </button>
            </div>
          </>
        ) : (
          <div className="px-4 py-4">
            <p className="text-sm text-text-secondary mb-3">{t('settings.mcp.generate_hint')}</p>
            <button
              type="button"
              onClick={handleGenerate}
              disabled={busy}
              className="px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
            >
              {t('settings.mcp.generate')}
            </button>
          </div>
        )}
      </div>

      {showKey && (
        <KeyModal
          value={newKey}
          onCopy={handleCopy}
          onClose={closeKey}
          t={t}
        />
      )}

      {confirmAction === 'regenerate' && (
        <ConfirmWordModal
          title={t('settings.mcp.regenerate_title')}
          subtitle={t('settings.mcp.regenerate_subtitle')}
          confirmLabel={t('settings.mcp.regenerate')}
          onConfirm={handleGenerate}
          onCancel={() => setConfirmAction(null)}
        />
      )}

      {confirmAction === 'revoke' && (
        <ConfirmWordModal
          title={t('settings.mcp.revoke_title')}
          subtitle={t('settings.mcp.revoke_subtitle')}
          confirmLabel={t('settings.mcp.revoke')}
          onConfirm={handleRevoke}
          onCancel={() => setConfirmAction(null)}
        />
      )}
    </div>
  )
}

function KeyModal({
  value,
  onCopy,
  onClose,
  t,
}: {
  value: string
  onCopy: () => Promise<void>
  onClose: () => void
  t: (key: string) => string
}) {
  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
      <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-md overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center justify-between gap-3">
          <h3 className="font-semibold">{t('settings.mcp.key_title')}</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('settings.mcp.done')}
            className="text-text-secondary hover:text-text"
          >
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>
        <div className="p-5 space-y-3">
          <p className="text-sm text-amber-600 dark:text-amber-400">{t('settings.mcp.key_warning')}</p>
          <input
            readOnly
            value={value}
            onFocus={(e) => e.currentTarget.select()}
            className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text font-mono text-sm min-h-[44px] focus:outline-none focus:border-primary"
          />
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => { void onCopy() }}
              className="flex-1 flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px]"
            >
              <Icon name="copy" className="w-4 h-4" />
              {t('settings.mcp.copy')}
            </button>
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2.5 rounded-ios-sm bg-bg border border-border font-medium min-h-[44px]"
            >
              {t('settings.mcp.done')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
