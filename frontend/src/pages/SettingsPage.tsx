import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from '../components/Icons'
import { api } from '../api'
import { useToast } from '../components/Toast'

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

  useEffect(() => {
    loadCurrencies()
    loadBillingHour()
    loadEmailSettings()
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

  const handleBillingHourChange = async (hour: number) => {
    try {
      await api.systemSettings.update({ billing_generation_hour: hour })
      setBillingHour(hour)
      showToast('Hora de facturación actualizada', 'success')
    } catch {
      showToast('Error al actualizar', 'error')
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

  const hours = Array.from({ length: 24 }, (_, i) => ({
    value: i,
    label: `${String(i).padStart(2, '0')}:00`,
  }))

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

      {/* Email alerts */}
      <div className="mb-6">
        <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
          {t('settings.email_alerts.title')}
        </div>
        <div className="bg-card rounded-ios shadow-ios overflow-hidden">
          <div className="px-4 py-3.5 border-b border-border">
            <div className="font-medium">{t('settings.email_alerts.smtp')}</div>
            <div className="text-sm text-text-secondary mb-3">{t('settings.email_alerts.smtp_subtitle')}</div>
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
                <input
                  type="text"
                  value={smtpUser}
                  onChange={(e) => setSmtpUser(e.target.value)}
                  className={inputCls}
                  placeholder={smtpConfigured ? t('settings.email_alerts.leave_blank') : ''}
                />
              </div>
              <div>
                <label className={labelCls}>{t('settings.email_alerts.smtp_password')}</label>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={smtpPassword}
                    onChange={(e) => setSmtpPassword(e.target.value)}
                    className={inputCls}
                    placeholder={smtpConfigured ? t('settings.email_alerts.leave_blank') : ''}
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
              <div className="flex items-center gap-2 text-sm">
                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${smtpConfigured ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                  {smtpConfigured ? t('settings.email_alerts.configured') : t('settings.email_alerts.not_configured')}
                </span>
              </div>
            </div>
          </div>

          <div className="px-4 py-3.5 border-b border-border">
            <div className="font-medium">{t('settings.email_alerts.recipients')}</div>
            <div className="text-sm text-text-secondary mb-3">{t('settings.email_alerts.recipients_hint')}</div>
            <input type="text" value={alertEmails} onChange={(e) => setAlertEmails(e.target.value)} className={inputCls} />
          </div>

          <div className="px-4 py-3.5 flex flex-col sm:flex-row gap-3">
            <button
              onClick={handleSaveEmail}
              disabled={savingEmail}
              className="flex-1 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
            >
              {savingEmail ? '...' : 'Guardar'}
            </button>
            <button
              onClick={handleTestEmail}
              disabled={sendingTest}
              className="flex-1 px-4 py-2.5 rounded-ios-sm border border-border font-medium min-h-[44px] disabled:opacity-50"
            >
              {t('settings.email_alerts.test_email')}
            </button>
          </div>
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