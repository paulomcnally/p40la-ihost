import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import Toggle from '../components/Toggle'
import { api } from '../api'
import { useToast } from '../components/Toast'
import type { Alert } from '../types'

type AlertChannelField = 'mail_enabled' | 'voice_enabled' | 'telegram_enabled'

type SendNowResult = {
  key: string
  title: string
  sent_channels: string[]
  items: number
  detail: string
}

export default function SettingsAlertsPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.alerts.title'))
  const { showToast } = useToast()
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [emailActive, setEmailActive] = useState(false)
  const [vmActive, setVmActive] = useState(false)
  const [tbActive, setTbActive] = useState(false)
  const [sending, setSending] = useState(false)

  useEffect(() => {
    loadAlerts()
    loadSystemSettings()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadAlerts = async () => {
    try {
      const data = await api.alerts.list()
      if (data) setAlerts(data)
    } catch {
      // ignore
    }
  }

  const loadSystemSettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        const emailActive = (data.email_alerts_enabled ?? false)
          && (data.smtp_configured ?? false)
          && (data.alert_emails ?? '').split(',').map((e) => e.trim()).filter(Boolean).length > 0
        const vmActive = (data.voicemonkey_enabled ?? false)
          && (data.voicemonkey_configured ?? false)
          && (data.voicemonkey_send_alerts ?? false)
        const tbActive = (data.telegram_bot_enabled ?? false)
          && (data.telegram_bot_configured ?? false)
          && (data.telegram_bot_chat_ids_count ?? 0) > 0
        setEmailActive(emailActive)
        setVmActive(vmActive)
        setTbActive(tbActive)
      }
    } catch {
      // ignore
    }
  }

  const handleToggleAlert = async (key: string, field: AlertChannelField, value: boolean) => {
    setAlerts((prev) => prev.map((a) => (a.key === key ? { ...a, [field]: value } : a)))
    try {
      await api.alerts.update(key, { [field]: value })
      showToast(t('settings.alerts.saved'), 'success')
    } catch {
      setAlerts((prev) => prev.map((a) => (a.key === key ? { ...a, [field]: !value } : a)))
      showToast(t('settings.alerts.save_error'), 'error')
    }
  }

  const channelLabel = (ch: string): string => {
    switch (ch) {
      case 'mail': return t('settings.alerts.channel_mail')
      case 'voice': return t('settings.alerts.channel_voice')
      case 'telegram': return t('settings.alerts.channel_telegram')
      default: return ch
    }
  }

  const handleSendNow = async () => {
    if (sending) return
    setSending(true)
    try {
      const data = await api.alerts.sendNow()
      if (data) {
        const lines = (data.results ?? []).map((r: SendNowResult) => {
          const channels = r.sent_channels.length > 0
            ? r.sent_channels.map(channelLabel).join(', ')
            : t('settings.alerts.send_now_no_channels')
          return `${r.title}: ${channels}`
        })
        showToast(`${t('settings.alerts.send_now_sent')}${lines.length ? ` — ${lines.join(' · ')}` : ''}`, 'success')
      }
    } catch {
      showToast(t('settings.alerts.send_now_error'), 'error')
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden mb-4">
        <div className="px-4 py-3.5">
          <div className="font-medium">{t('settings.alerts.send_now')}</div>
          <div className="text-sm text-text-secondary mb-3">{t('settings.alerts.send_now_hint')}</div>
          <button
            onClick={handleSendNow}
            disabled={sending}
            className="w-full px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
          >
            {sending ? t('settings.alerts.send_now_sending') : t('settings.alerts.send_now')}
          </button>
        </div>
      </div>
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        {alerts.length === 0 && (
          <div className="px-4 py-3.5 text-sm text-text-secondary">...</div>
        )}
        {alerts.map((a, idx) => (
          <div key={a.key} className={`px-4 py-3.5 ${idx < alerts.length - 1 ? 'border-b border-border' : ''}`}>
            <div className="font-medium">{a.title}</div>
            <div className="text-sm text-text-secondary mb-3">{a.description}</div>
            <div className="flex items-center gap-6">
              <div className={`flex items-center gap-2 ${emailActive ? '' : 'opacity-50'}`}>
                <Toggle checked={a.mail_enabled} onChange={(v) => handleToggleAlert(a.key, 'mail_enabled', v)} disabled={!emailActive} />
                <span className="text-sm font-medium">{t('settings.alerts.mail')}</span>
              </div>
              <div className={`flex items-center gap-2 ${vmActive ? '' : 'opacity-50'}`}>
                <Toggle checked={a.voice_enabled} onChange={(v) => handleToggleAlert(a.key, 'voice_enabled', v)} disabled={!vmActive} />
                <span className="text-sm font-medium">{t('settings.alerts.alexa')}</span>
              </div>
              <div className={`flex items-center gap-2 ${tbActive ? '' : 'opacity-50'}`}>
                <Toggle checked={a.telegram_enabled} onChange={(v) => handleToggleAlert(a.key, 'telegram_enabled', v)} disabled={!tbActive} />
                <span className="text-sm font-medium">{t('settings.alerts.telegram')}</span>
              </div>
            </div>
            {!vmActive && (
              <p className="text-xs text-text-secondary mt-2">{t('settings.alerts.alexa_disabled_hint')}</p>
            )}
            {!emailActive && (
              <p className="text-xs text-text-secondary mt-2">{t('settings.alerts.mail_disabled_hint')}</p>
            )}
            {!tbActive && (
              <p className="text-xs text-text-secondary mt-2">{t('settings.alerts.telegram_disabled_hint')}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}