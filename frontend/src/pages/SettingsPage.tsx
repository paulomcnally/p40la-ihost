import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from '../components/Icons'
import Toggle from '../components/Toggle'
import { api } from '../api'
import { useToast } from '../components/Toast'
import type { Alert } from '../types'

export default function SettingsPage() {
  const navigate = useNavigate()
  const { currencies, loadCurrencies } = useAppStore()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [billingHour, setBillingHour] = useState(0)

  // Email alerts
  const [smtpHost, setSmtpHost] = useState('')
  const [smtpPort, setSmtpPort] = useState(587)
  const [smtpUser, setSmtpUser] = useState('')
  const [smtpPassword, setSmtpPassword] = useState('')
  const [smtpFromEmail, setSmtpFromEmail] = useState('')
  const [smtpFromName, setSmtpFromName] = useState('')
  const [alertEmails, setAlertEmails] = useState('')
  const [smtpConfigured, setSmtpConfigured] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [sendingTest, setSendingTest] = useState(false)
  const [savingEmail, setSavingEmail] = useState(false)
  const [smtpOpen, setSmtpOpen] = useState(false)

  // Alertas (catálogo + toggles de canal mail/alexa)
  const [alerts, setAlerts] = useState<Alert[]>([])

  // Voice Monkey (Alexa)
  const [vmEnabled, setVmEnabled] = useState(false)
  const [vmSendAlerts, setVmSendAlerts] = useState(false)
  const [vmToken, setVmToken] = useState('')
  const [vmDevice, setVmDevice] = useState('')
  const [vmConfigured, setVmConfigured] = useState(false)
  const [savingVoice, setSavingVoice] = useState(false)
  const [testingVoice, setTestingVoice] = useState(false)

  useEffect(() => {
    loadCurrencies()
    loadBillingHour()
    loadEmailSettings()
    loadVoiceMonkeySettings()
    loadAlerts()
  }, [])

  const loadBillingHour = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setBillingHour(data.billing_generation_hour ?? 0)
      }
    } catch {
      // ignore
    }
  }

  const loadEmailSettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setSmtpHost(data.smtp_host ?? '')
        setSmtpPort(data.smtp_port ?? 587)
        setSmtpUser('') // user nunca se devuelve (sensible)
        setSmtpFromEmail(data.smtp_from_email ?? '')
        setSmtpFromName(data.smtp_from_name ?? '')
        setAlertEmails(data.alert_emails ?? '')
        setSmtpConfigured(data.smtp_configured ?? false)
      }
    } catch {
      // ignore
    }
  }

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

  const loadAlerts = async () => {
    try {
      const data = await api.alerts.list()
      if (data) setAlerts(data)
    } catch {
      // ignore
    }
  }

  const handleBillingHourChange = async (hour: number) => {
    try {
      await api.systemSettings.update({ billing_generation_hour: hour })
      setBillingHour(hour)
      showToast('Hora de facturación actualizada', 'success')
    } catch {
      showToast('Error al actualizar', 'error')
    }
  }

  const handleToggleAlert = async (key: string, field: 'mail_enabled' | 'voice_enabled', value: boolean) => {
    setAlerts((prev) => prev.map((a) => (a.key === key ? { ...a, [field]: value } : a)))
    try {
      await api.alerts.update(key, { [field]: value })
      showToast(t('settings.alerts.saved'), 'success')
    } catch {
      setAlerts((prev) => prev.map((a) => (a.key === key ? { ...a, [field]: !value } : a)))
      showToast(t('settings.alerts.save_error'), 'error')
    }
  }

  const handleSaveEmail = async () => {
    setSavingEmail(true)
    try {
      const body: Record<string, unknown> = {
        smtp_host: smtpHost,
        smtp_port: Number(smtpPort) || 587,
        smtp_from_email: smtpFromEmail,
        smtp_from_name: smtpFromName,
        alert_emails: alertEmails,
      }
      // Solo se envían user/password si el usuario escribió algo nuevo.
      if (smtpUser.trim()) body.smtp_user = smtpUser.trim()
      if (smtpPassword.trim()) body.smtp_password = smtpPassword.trim()
      await api.systemSettings.update(body)
      const data = await api.systemSettings.get()
      if (data) setSmtpConfigured(data.smtp_configured ?? false)
      setSmtpPassword('')
      setSmtpUser('')
      showToast(t('settings.email_alerts.saved'), 'success')
    } catch {
      showToast(t('settings.email_alerts.save_error'), 'error')
    } finally {
      setSavingEmail(false)
    }
  }

  const handleTestEmail = async () => {
    setSendingTest(true)
    try {
      await api.systemSettings.testEmail()
      showToast(t('settings.email_alerts.test_sent'), 'success')
    } catch {
      showToast(t('settings.email_alerts.test_error'), 'error')
    } finally {
      setSendingTest(false)
    }
  }

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

  const handleReconfigureSMTP = async () => {
    if (!window.confirm(t('settings.email_alerts.reconfigure_confirm'))) return
    try {
      await api.systemSettings.disconnectSMTP()
      setSmtpHost('')
      setSmtpPort(587)
      setSmtpUser('')
      setSmtpPassword('')
      setSmtpFromEmail('')
      setSmtpFromName('')
      setSmtpConfigured(false)
      showToast(t('settings.email_alerts.reconfigured'), 'success')
    } catch {
      showToast(t('settings.email_alerts.save_error'), 'error')
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

  const hours = Array.from({ length: 24 }, (_, i) => ({
    value: i,
    label: `${String(i).padStart(2, '0')}:00`,
  }))

  // Voice Monkey está plenamente activo solo si master on + configurado + enviar alertas on.
  const vmActive = vmEnabled && vmConfigured && vmSendAlerts

  const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-white min-h-[44px]'
  const labelCls = 'text-sm font-medium text-text-secondary mb-1'

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-xl sm:text-2xl font-bold mb-4 sm:mb-6">{t('settings.title')}</h2>

      <div className="mb-6">
        <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
          {t('settings.section_general')}
        </div>
        <div className="bg-card rounded-ios shadow-ios overflow-hidden">
          <button
            onClick={() => navigate('/settings/language')}
            className="w-full flex items-center justify-between px-4 py-3.5 border-b border-border hover:bg-bg/50 transition-colors"
          >
            <div>
              <div className="font-medium">{t('settings.language.title')}</div>
              <div className="text-sm text-text-secondary">{t('settings.language.subtitle')}</div>
            </div>
            <div className="flex items-center gap-1 text-text-secondary">
              <Icon name="chevron" className="w-4 h-4" />
            </div>
          </button>
        </div>
      </div>

      <div className="mb-6">
        <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
          Facturación automática
        </div>
        <div className="bg-card rounded-ios shadow-ios overflow-hidden">
          <div className="px-4 py-3.5 border-b border-border">
            <div className="font-medium">Hora de generación</div>
            <div className="text-sm text-text-secondary mb-3">Hora del día para generar facturas automáticamente</div>
            <select
              value={billingHour}
              onChange={(e) => handleBillingHourChange(Number(e.target.value))}
              className={inputCls}
            >
              {hours.map(h => (
                <option key={h.value} value={h.value}>{h.label}</option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Alertas */}
      <div className="mb-6">
        <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
          {t('settings.alerts.title')}
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
                <div className="flex items-center gap-2">
                  <Toggle checked={a.mail_enabled} onChange={(v) => handleToggleAlert(a.key, 'mail_enabled', v)} />
                  <span className="text-sm font-medium">{t('settings.alerts.mail')}</span>
                </div>
                <div className={`flex items-center gap-2 ${vmActive ? '' : 'opacity-50'}`}>
                  <Toggle checked={a.voice_enabled} onChange={(v) => handleToggleAlert(a.key, 'voice_enabled', v)} disabled={!vmActive} />
                  <span className="text-sm font-medium">{t('settings.alerts.alexa')}</span>
                </div>
              </div>
              {!vmActive && (
                <p className="text-xs text-text-secondary mt-2">{t('settings.alerts.alexa_disabled_hint')}</p>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Email alerts */}
      <div className="mb-6">
        <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
          {t('settings.email_alerts.title')}
        </div>
        <div className="bg-card rounded-ios shadow-ios overflow-hidden">
          {/* SMTP como acordeón colapsable, cerrado por defecto */}
          <button
            type="button"
            onClick={() => setSmtpOpen(!smtpOpen)}
            className="w-full flex items-center justify-between px-4 py-3.5 border-b border-border hover:bg-bg/50 transition-colors"
          >
            <div className="text-left">
              <div className="font-medium">{t('settings.email_alerts.smtp')}</div>
              <div className="text-sm text-text-secondary">{t('settings.email_alerts.smtp_subtitle')}</div>
            </div>
            <Icon name="chevron" className={`w-4 h-4 text-text-secondary transition-transform ${smtpOpen ? 'rotate-180' : ''}`} />
          </button>
          {smtpOpen && (
            <>
              {smtpConfigured ? (
                <div className="px-4 py-3.5 border-b border-border flex items-center justify-between">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700">
                    {t('settings.email_alerts.configured_status')}
                  </span>
                  <button
                    onClick={handleReconfigureSMTP}
                    className="px-4 py-2 rounded-ios-sm border border-border font-medium min-h-[44px]"
                  >
                    {t('settings.email_alerts.reconfigure')}
                  </button>
                </div>
              ) : (
                <div className="px-4 py-3.5 border-b border-border">
                  <div className="space-y-3">
                    <div>
                      <label className={labelCls}>{t('settings.email_alerts.smtp_host')}</label>
                      <input type="text" value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} className={inputCls} />
                    </div>
                    <div>
                      <label className={labelCls}>{t('settings.email_alerts.smtp_port')}</label>
                      <input type="number" value={smtpPort} onChange={(e) => setSmtpPort(Number(e.target.value))} className={inputCls} />
                    </div>
                    <div>
                      <label className={labelCls}>{t('settings.email_alerts.smtp_user')}</label>
                      <input type="text" value={smtpUser} onChange={(e) => setSmtpUser(e.target.value)} className={inputCls} />
                    </div>
                    <div>
                      <label className={labelCls}>{t('settings.email_alerts.smtp_password')}</label>
                      <div className="relative">
                        <input
                          type={showPassword ? 'text' : 'password'}
                          value={smtpPassword}
                          onChange={(e) => setSmtpPassword(e.target.value)}
                          className={inputCls}
                        />
                        <button
                          type="button"
                          onClick={() => setShowPassword(!showPassword)}
                          className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-text-secondary px-2 py-1"
                        >
                          {showPassword ? t('settings.email_alerts.hide') : t('settings.email_alerts.show')}
                        </button>
                      </div>
                    </div>
                    <div>
                      <label className={labelCls}>{t('settings.email_alerts.smtp_from_email')}</label>
                      <input type="email" value={smtpFromEmail} onChange={(e) => setSmtpFromEmail(e.target.value)} className={inputCls} />
                    </div>
                    <div>
                      <label className={labelCls}>{t('settings.email_alerts.smtp_from_name')}</label>
                      <input type="text" value={smtpFromName} onChange={(e) => setSmtpFromName(e.target.value)} className={inputCls} />
                    </div>
                  </div>
                </div>
              )}
            </>
          )}

          <div className="px-4 py-3.5 border-b border-border">
            <div className="font-medium">{t('settings.email_alerts.recipients')}</div>
            <div className="text-sm text-text-secondary mb-3">{t('settings.email_alerts.recipients_hint')}</div>
            <input type="text" value={alertEmails} onChange={(e) => setAlertEmails(e.target.value)} className={inputCls} />
          </div>

          <div className="px-4 py-3.5 flex flex-col sm:flex-row gap-3">
            {!smtpConfigured && (
              <button
                onClick={handleSaveEmail}
                disabled={savingEmail}
                className="flex-1 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
              >
                {savingEmail ? '...' : 'Guardar'}
              </button>
            )}
            <button
              onClick={handleTestEmail}
              disabled={sendingTest || !smtpConfigured}
              className="flex-1 px-4 py-2.5 rounded-ios-sm border border-border font-medium min-h-[44px] disabled:opacity-50"
            >
              {t('settings.email_alerts.test_email')}
            </button>
          </div>
        </div>
      </div>

      {/* Voice Monkey (Alexa) */}
      <div className="mb-6">
        <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
          {t('settings.voicemonkey.title')}
        </div>
        <div className="bg-card rounded-ios shadow-ios overflow-hidden">
          <div className="px-4 py-3.5 border-b border-border flex items-center justify-between">
            <div>
              <div className="font-medium">{t('settings.voicemonkey.enable')}</div>
              <div className="text-sm text-text-secondary">{t('settings.voicemonkey.enable_hint')}</div>
            </div>
            <Toggle checked={vmEnabled} onChange={handleVmEnabledChange} />
          </div>

          {vmEnabled && (
            <>
              {vmConfigured ? (
                <div className="px-4 py-3.5 border-b border-border flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700">
                      {t('settings.voicemonkey.configured_status')}
                    </span>
                  </div>
                  <button
                    onClick={handleReconfigureVoiceMonkey}
                    className="px-4 py-2 rounded-ios-sm border border-border font-medium min-h-[44px]"
                  >
                    {t('settings.voicemonkey.reconfigure')}
                  </button>
                </div>
              ) : (
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

      <div>
        <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
          {t('settings.section_currencies')}
        </div>
        <div className="bg-card rounded-ios shadow-ios overflow-hidden">
          {currencies.map(c => (
            <button
              key={c.id}
              onClick={() => navigate(`/settings/currency/${c.id}`)}
              className="w-full flex items-center justify-between px-4 py-3.5 border-b border-border last:border-b-0 hover:bg-bg/50 transition-colors"
            >
              <div>
                <div className="font-medium">{c.code}</div>
                <div className="text-sm text-text-secondary">{c.name} · {c.symbol}</div>
              </div>
              <Icon name="chevron" className="w-4 h-4 text-text-secondary" />
            </button>
          ))}
          <button
            onClick={() => navigate('/settings/currency')}
            className="w-full flex items-center justify-between px-4 py-3.5 hover:bg-bg/50 transition-colors"
          >
            <div className="font-medium">{t('settings.currencies.create')}</div>
            <Icon name="plus" className="w-4 h-4 text-text-secondary" />
          </button>
        </div>
      </div>
    </div>
  )
}