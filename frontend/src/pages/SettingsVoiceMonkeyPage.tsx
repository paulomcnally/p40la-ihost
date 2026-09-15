import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import Toggle from '../components/Toggle'
import HelpPanel from '../components/HelpPanel'
import { api } from '../api'
import { useToast } from '../components/Toast'

export default function SettingsVoiceMonkeyPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.voicemonkey.title'))
  const { showToast } = useToast()
  const [vmEnabled, setVmEnabled] = useState(false)
  const [vmSendAlerts, setVmSendAlerts] = useState(false)
  const [vmToken, setVmToken] = useState('')
  const [vmDevice, setVmDevice] = useState('')
  const [vmConfigured, setVmConfigured] = useState(false)
  const [savingVoice, setSavingVoice] = useState(false)
  const [testingVoice, setTestingVoice] = useState(false)

  useEffect(() => {
    loadVoiceMonkeySettings()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadVoiceMonkeySettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setVmEnabled(data.voicemonkey_enabled ?? false)
        setVmSendAlerts(data.voicemonkey_send_alerts ?? false)
        setVmConfigured(data.voicemonkey_configured ?? false)
      }
    } catch {
      // ignore
    }
  }

  const vmActive = vmEnabled && vmConfigured && vmSendAlerts
  const vmNeedsConfig = vmEnabled && !vmActive

  const handleVmEnabledChange = async (value: boolean) => {
    setVmEnabled(value)
    try {
      await api.systemSettings.update({ voicemonkey_enabled: value })
    } catch {
      setVmEnabled(!value)
      showToast(t('settings.voicemonkey.save_error'), 'error')
    }
  }

  const handleVmSendChange = async (value: boolean) => {
    setVmSendAlerts(value)
    try {
      await api.systemSettings.update({ voicemonkey_send_alerts: value })
    } catch {
      setVmSendAlerts(!value)
      showToast(t('settings.voicemonkey.save_error'), 'error')
    }
  }

  const handleSaveVoiceMonkey = async () => {
    setSavingVoice(true)
    try {
      const body: Record<string, unknown> = {}
      if (vmToken.trim()) body.voicemonkey_token = vmToken.trim()
      if (vmDevice.trim()) body.voicemonkey_device = vmDevice.trim()
      await api.systemSettings.update(body)
      const data = await api.systemSettings.get()
      if (data) setVmConfigured(data.voicemonkey_configured ?? false)
      setVmToken('')
      setVmDevice('')
      showToast(t('settings.voicemonkey.saved'), 'success')
    } catch {
      showToast(t('settings.voicemonkey.save_error'), 'error')
    } finally {
      setSavingVoice(false)
    }
  }

  const handleTestVoice = async () => {
    setTestingVoice(true)
    try {
      await api.systemSettings.testVoice()
      showToast(t('settings.voicemonkey.test_sent'), 'success')
    } catch {
      showToast(t('settings.voicemonkey.test_error'), 'error')
    } finally {
      setTestingVoice(false)
    }
  }

  const handleReconfigureVoiceMonkey = async () => {
    if (!window.confirm(t('settings.voicemonkey.reconfigure_confirm'))) return
    try {
      await api.systemSettings.disconnectVoiceMonkey()
      setVmEnabled(false)
      setVmSendAlerts(false)
      setVmConfigured(false)
      setVmToken('')
      setVmDevice('')
      showToast(t('settings.voicemonkey.reconfigured'), 'success')
    } catch {
      showToast(t('settings.voicemonkey.save_error'), 'error')
    }
  }

  const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px]'
  const labelCls = 'text-sm font-medium text-text-secondary mb-1'

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border flex items-center justify-between gap-3">
          <div className="flex-1">
            <div className="font-medium">{t('settings.voicemonkey.enable')}</div>
            <div className="text-sm text-text-secondary">{t('settings.voicemonkey.enable_hint')}</div>
          </div>
          <Toggle checked={vmEnabled} onChange={handleVmEnabledChange} />
        </div>

        {vmNeedsConfig && (
          <div className="px-4 py-3 border-b border-border">
            <p className="text-xs text-amber-600 dark:text-amber-400">{t('settings.voicemonkey.needs_config_hint')}</p>
          </div>
        )}

        {vmEnabled && (
          <>
            {vmConfigured ? (
              <div className="px-4 py-3.5 border-b border-border flex items-center justify-between">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300">
                  {t('settings.voicemonkey.configured_status')}
                </span>
                <button
                  onClick={handleReconfigureVoiceMonkey}
                  className="px-4 py-2 rounded-ios-sm border border-border font-medium min-h-[44px]"
                >
                  {t('settings.voicemonkey.reconfigure')}
                </button>
              </div>
            ) : (
              <>
                <div className="px-4 py-3.5 border-b border-border">
                  <div className="space-y-3">
                    <div>
                      <label className={labelCls}>{t('settings.voicemonkey.token')}</label>
                      <input
                        type="password"
                        value={vmToken}
                        onChange={(e) => setVmToken(e.target.value)}
                        className={inputCls}
                      />
                    </div>
                    <div>
                      <label className={labelCls}>{t('settings.voicemonkey.device')}</label>
                      <input
                        type="text"
                        value={vmDevice}
                        onChange={(e) => setVmDevice(e.target.value)}
                        className={inputCls}
                      />
                    </div>
                  </div>
                </div>
                <HelpPanel title={t('settings.voicemonkey.help.title')}>
                  <ol className="list-decimal pl-5 space-y-2">
                    <li>
                      {t('settings.voicemonkey.help.step1')}{' '}
                      <a href="https://app.voicemonkey.io/tokens" target="_blank" rel="noopener noreferrer" className="text-primary underline">
                        {t('settings.voicemonkey.help.open_tokens')}
                      </a>
                    </li>
                    <li>{t('settings.voicemonkey.help.step2')}</li>
                    <li>
                      {t('settings.voicemonkey.help.step3')}{' '}
                      <a href="https://app.voicemonkey.io/speakers" target="_blank" rel="noopener noreferrer" className="text-primary underline">
                        {t('settings.voicemonkey.help.open_speakers')}
                      </a>
                    </li>
                    <li>{t('settings.voicemonkey.help.step4')}</li>
                  </ol>
                </HelpPanel>
              </>
            )}

            <div className={`px-4 py-3.5 border-b border-border flex items-center justify-between ${vmConfigured ? '' : 'opacity-50'}`}>
              <div>
                <div className="font-medium">{t('settings.voicemonkey.send_alerts')}</div>
              </div>
              <Toggle checked={vmSendAlerts} onChange={handleVmSendChange} disabled={!vmConfigured} />
            </div>

            <div className="px-4 py-3.5 flex flex-col sm:flex-row gap-3">
              {!vmConfigured && (
                <button
                  onClick={handleSaveVoiceMonkey}
                  disabled={savingVoice}
                  className="flex-1 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
                >
                  {savingVoice ? '...' : t('app.save')}
                </button>
              )}
              <button
                onClick={handleTestVoice}
                disabled={testingVoice || !vmConfigured}
                className="flex-1 px-4 py-2.5 rounded-ios-sm border border-border font-medium min-h-[44px] disabled:opacity-50"
              >
                {t('settings.voicemonkey.test')}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}