import { useEffect, useState } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
import { copyToClipboard } from '../utils/clipboard'
import type { Service } from '../types'

interface WebhookModalProps {
  service: Service
  onClose: () => void
}

export default function WebhookModal({ service, onClose }: WebhookModalProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [webhookUrl, setWebhookUrl] = useState('')
  const [webhookEnabled, setWebhookEnabled] = useState(true)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const settings = await api.systemSettings.get()
      setWebhookEnabled(settings?.webhook_enabled ?? true)
      if (service.webhook_uuid) {
        const base = (settings?.webhook_base_url || `${window.location.origin}`).replace(/\/+$/, '')
        setWebhookUrl(`${base}/webhooks/${service.webhook_uuid}`)
      }
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  const handleCopy = async (text: string, toastKey: string) => {
    try {
      await copyToClipboard(text)
      showToast(t(toastKey) || 'Copiado', 'success')
    } catch {
      showToast(t('errors.generic') || 'Error', 'error')
    }
  }

  const handleRegenerateWebhook = async () => {
    try {
      const res = await api.services.regenerateWebhook(service.id)
      if (res?.webhook_url) setWebhookUrl(res.webhook_url)
      showToast(t('services.webhook_regenerated') || 'Webhook regenerado', 'success')
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
      showToast(message, 'error')
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-md overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-ios flex items-center justify-center bg-primary/10 text-primary">
              <Icon name="link" className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-lg font-semibold">{t('services.webhook_title')}</h3>
              <p className="text-sm text-text-secondary">{service.name}</p>
            </div>
          </div>

          {loading ? (
            <div className="flex justify-center py-8">
              <div className="w-6 h-6 rounded-full border-2 border-primary/30 border-t-primary animate-spin" />
            </div>
          ) : !webhookEnabled ? (
            <div className="text-center py-6">
              <div className="w-12 h-12 mx-auto mb-3 text-text-secondary opacity-60">
                <Icon name="warning" className="w-full h-full" />
              </div>
              <p className="text-text-secondary">{t('services.webhook_disabled_hint')}</p>
              <button
                onClick={onClose}
                className="mt-4 px-4 py-2 rounded-ios-sm bg-primary text-white font-medium min-h-[44px]"
              >
                {t('app.close')}
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">{t('services.webhook_url')}</label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    readOnly
                    value={webhookUrl}
                    className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] text-sm"
                  />
                  <button
                    type="button"
                    onClick={() => handleCopy(webhookUrl, 'services.webhook_copied')}
                    className="shrink-0 px-3 py-2 bg-bg border border-border rounded-ios-sm hover:border-primary/50 transition-colors flex items-center gap-1 min-h-[44px] text-sm"
                    title={t('services.webhook_copy')}
                  >
                    <Icon name="copy" className="w-4 h-4" />
                  </button>
                  <button
                    type="button"
                    onClick={handleRegenerateWebhook}
                    className="shrink-0 px-3 py-2 bg-bg border border-border rounded-ios-sm hover:border-primary/50 transition-colors flex items-center gap-1 min-h-[44px] text-sm"
                    title={t('services.webhook_regenerate')}
                  >
                    <Icon name="refresh" className="w-4 h-4" />
                  </button>
                </div>
                <p className="text-xs text-text-secondary mt-1">{t('services.webhook_url_desc')}</p>
                <p className="text-xs text-text-secondary mt-1">{t('services.webhook_schema_desc')}</p>
              </div>
            </div>
          )}
        </div>
        {webhookEnabled && !loading && (
          <div className="flex gap-3 p-4 border-t border-border">
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
            >
              {t('app.close')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}