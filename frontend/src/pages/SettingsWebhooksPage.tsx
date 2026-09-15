import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import Toggle from '../components/Toggle'
import { Icon } from '../components/Icons'
import { api } from '../api'
import { useToast } from '../components/Toast'
import { copyToClipboard } from '../utils/clipboard'

export default function SettingsWebhooksPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.webhooks.title'))
  const { showToast } = useToast()
  const [webhookEnabled, setWebhookEnabled] = useState(false)
  const [webhookApiKey, setWebhookApiKey] = useState('')
  const [webhookBaseUrl, setWebhookBaseUrl] = useState('http://ihost.local:8088')
  const [savingWebhookBaseUrl, setSavingWebhookBaseUrl] = useState(false)

  useEffect(() => {
    loadWebhookSettings()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadWebhookSettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setWebhookEnabled(data.webhook_enabled ?? false)
        if (data.webhook_base_url) setWebhookBaseUrl(data.webhook_base_url)
      }
      const keyRes = await api.webhooks.getApiKey()
      if (keyRes?.api_key) setWebhookApiKey(keyRes.api_key)
    } catch {
      // ignore
    }
  }

  const handleWebhookEnabledChange = async (value: boolean) => {
    setWebhookEnabled(value)
    try {
      await api.systemSettings.update({ webhook_enabled: value })
      if (value) {
        const keyRes = await api.webhooks.getApiKey()
        if (keyRes?.api_key) setWebhookApiKey(keyRes.api_key)
      }
    } catch {
      setWebhookEnabled(!value)
      showToast(t('settings.webhooks.disabled_hint') || 'Error al guardar', 'error')
    }
  }

  const handleRegenerateWebhookKey = async () => {
    if (!window.confirm(t('settings.webhooks.regenerate_confirm'))) return
    try {
      const res = await api.webhooks.regenerateApiKey()
      if (res?.api_key) setWebhookApiKey(res.api_key)
      showToast(t('settings.webhooks.regenerated'), 'success')
    } catch {
      showToast(t('errors.generic'), 'error')
    }
  }

  const handleCopyWebhookKey = async () => {
    try {
      await copyToClipboard(webhookApiKey)
      showToast(t('settings.webhooks.copied'), 'success')
    } catch {
      showToast(t('errors.generic'), 'error')
    }
  }

  const handleSaveWebhookBaseUrl = async () => {
    setSavingWebhookBaseUrl(true)
    try {
      await api.systemSettings.update({ webhook_base_url: webhookBaseUrl.trim() })
      showToast(t('settings.webhooks.base_url_saved'), 'success')
    } catch {
      showToast(t('errors.generic'), 'error')
    } finally {
      setSavingWebhookBaseUrl(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border flex items-center justify-between gap-3">
          <div className="flex-1">
            <div className="font-medium">{t('settings.webhooks.enable')}</div>
            <div className="text-sm text-text-secondary">{t('settings.webhooks.enable_hint')}</div>
          </div>
          <Toggle checked={webhookEnabled} onChange={handleWebhookEnabledChange} />
        </div>

        {!webhookEnabled ? (
          <div className="px-4 py-3 border-b border-border">
            <p className="text-xs text-amber-600 dark:text-amber-400">{t('settings.webhooks.disabled_hint')}</p>
          </div>
        ) : (
          <>
            <div className="px-4 py-3.5 border-b border-border">
              <div className="font-medium mb-1">{t('settings.webhooks.base_url')}</div>
              <div className="text-sm text-text-secondary mb-3">{t('settings.webhooks.base_url_desc')}</div>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={webhookBaseUrl}
                  onChange={(e) => setWebhookBaseUrl(e.target.value)}
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px] text-sm"
                />
                <button
                  type="button"
                  onClick={handleSaveWebhookBaseUrl}
                  disabled={savingWebhookBaseUrl}
                  className="shrink-0 px-4 py-2 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
                >
                  {savingWebhookBaseUrl ? '...' : t('app.save')}
                </button>
              </div>
            </div>
            <div className="px-4 py-3.5 border-b border-border">
              <div className="font-medium mb-1">{t('settings.webhooks.api_key')}</div>
              <div className="text-sm text-text-secondary mb-3">{t('settings.webhooks.api_key_desc')}</div>
              <div className="flex gap-2">
                <input
                  type="text"
                  readOnly
                  value={webhookApiKey}
                  placeholder="••••••••"
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px] text-sm font-mono"
                />
                <button
                  type="button"
                  onClick={handleCopyWebhookKey}
                  className="shrink-0 px-3 py-2 bg-bg border border-border rounded-ios-sm hover:border-primary/50 transition-colors flex items-center gap-1 min-h-[44px] text-sm"
                  title={t('settings.webhooks.copy')}
                >
                  <Icon name="copy" className="w-4 h-4" />
                </button>
                <button
                  type="button"
                  onClick={handleRegenerateWebhookKey}
                  className="shrink-0 px-3 py-2 bg-bg border border-border rounded-ios-sm hover:border-primary/50 transition-colors flex items-center gap-1 min-h-[44px] text-sm"
                  title={t('settings.webhooks.regenerate')}
                >
                  <Icon name="refresh" className="w-4 h-4" />
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}