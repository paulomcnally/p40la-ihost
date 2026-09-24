import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { useToast } from '../components/Toast'

export default function SettingsAutomationPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.automation.title'))
  const { showToast } = useToast()
  const [baseUrl, setBaseUrl] = useState('http://ihost.local:8089')
  const [apiKey, setApiKey] = useState('')
  const [configured, setConfigured] = useState(false)
  const [savingBaseUrl, setSavingBaseUrl] = useState(false)
  const [savingApiKey, setSavingApiKey] = useState(false)

  useEffect(() => {
    loadSettings()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadSettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        if (data.automation_base_url) setBaseUrl(data.automation_base_url)
        setConfigured(data.automation_configured ?? false)
      }
    } catch {
      // ignore
    }
  }

  const handleSaveBaseUrl = async () => {
    setSavingBaseUrl(true)
    try {
      await api.systemSettings.update({ automation_base_url: baseUrl.trim() })
      showToast(t('settings.automation.saved'), 'success')
      await loadSettings()
    } catch {
      showToast(t('errors.generic'), 'error')
    } finally {
      setSavingBaseUrl(false)
    }
  }

  const handleSaveApiKey = async () => {
    setSavingApiKey(true)
    try {
      await api.systemSettings.update({ automation_api_key: apiKey.trim() })
      setApiKey('')
      showToast(t('settings.automation.saved'), 'success')
      await loadSettings()
    } catch {
      showToast(t('errors.generic'), 'error')
    } finally {
      setSavingApiKey(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium mb-1">{t('settings.automation.title')}</div>
          <div className="text-sm text-text-secondary">{t('settings.automation.desc')}</div>
        </div>

        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium mb-1">{t('settings.automation.base_url')}</div>
          <div className="text-sm text-text-secondary mb-3">{t('settings.automation.base_url_desc')}</div>
          <div className="flex gap-2">
            <input
              type="text"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px] text-sm"
            />
            <button
              type="button"
              onClick={handleSaveBaseUrl}
              disabled={savingBaseUrl}
              className="shrink-0 px-4 py-2 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
            >
              {savingBaseUrl ? '...' : t('app.save')}
            </button>
          </div>
        </div>

        <div className="px-4 py-3.5 border-b border-border">
          <div className="flex items-center gap-2">
            <div className="font-medium mb-1">{t('settings.automation.api_key')}</div>
            {configured && (
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-success/20 text-green-800 dark:text-green-400">
                {t('settings.automation.configured')}
              </span>
            )}
          </div>
          <div className="text-sm text-text-secondary mb-3">{t('settings.automation.api_key_desc')}</div>
          <div className="flex gap-2">
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder={t('settings.automation.api_key_placeholder')}
              autoComplete="off"
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px] text-sm"
            />
            <button
              type="button"
              onClick={handleSaveApiKey}
              disabled={savingApiKey || apiKey.trim() === ''}
              className="shrink-0 px-4 py-2 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
            >
              {savingApiKey ? '...' : t('app.save')}
            </button>
          </div>
        </div>

        <div className="px-4 py-3">
          <p className="text-xs text-text-secondary">{t('settings.automation.hint')}</p>
        </div>
      </div>
    </div>
  )
}