import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import Toggle from '../components/Toggle'
import EmailRecipientsModal from '../components/EmailRecipientsModal'
import HelpPanel from '../components/HelpPanel'
import { Icon } from '../components/Icons'
import { api } from '../api'
import { useToast } from '../components/Toast'

function parseEmails(value?: string | null): string[] {
  if (!value) return []
  return value
    .split(',')
    .map((e) => e.trim())
    .filter(Boolean)
}

export default function SettingsEmailAlertsPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.email_alerts.title'))
  const { showToast } = useToast()
  const [smtpHost, setSmtpHost] = useState('')
  const [smtpPort, setSmtpPort] = useState(587)
  const [smtpUser, setSmtpUser] = useState('')
  const [smtpPassword, setSmtpPassword] = useState('')
  const [smtpFromEmail, setSmtpFromEmail] = useState('')
  const [smtpFromName, setSmtpFromName] = useState('')
  const [alertEmails, setAlertEmails] = useState<string[]>([])
  const [smtpConfigured, setSmtpConfigured] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [sendingTest, setSendingTest] = useState(false)
  const [savingEmail, setSavingEmail] = useState(false)
  const [smtpOpen, setSmtpOpen] = useState(false)
  const [showRecipientModal, setShowRecipientModal] = useState(false)
  const [savingRecipients, setSavingRecipients] = useState(false)
  const [emailAlertsEnabled, setEmailAlertsEnabled] = useState(false)

  useEffect(() => {
    loadEmailSettings()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadEmailSettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setSmtpHost(data.smtp_host ?? '')
        setSmtpPort(data.smtp_port ?? 587)
        setSmtpUser('')
        setSmtpFromEmail(data.smtp_from_email ?? '')
        setSmtpFromName(data.smtp_from_name ?? '')
        setAlertEmails(parseEmails(data.alert_emails))
        setSmtpConfigured(data.smtp_configured ?? false)
        setEmailAlertsEnabled(data.email_alerts_enabled ?? false)
      }
    } catch {
      // ignore
    }
  }

  const emailActive = emailAlertsEnabled && smtpConfigured && alertEmails.length > 0
  const emailNeedsConfig = emailAlertsEnabled && !emailActive

  const handleSaveEmail = async () => {
    setSavingEmail(true)
    try {
      const body: Record<string, unknown> = {
        smtp_host: smtpHost,
        smtp_port: Number(smtpPort) || 587,
        smtp_from_email: smtpFromEmail,
        smtp_from_name: smtpFromName,
      }
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

  const handleRecipientsChange = async (next: string[]) => {
    const prev = alertEmails
    setAlertEmails(next)
    setSavingRecipients(true)
    try {
      await api.systemSettings.update({ alert_emails: next.join(',') })
      showToast(t('settings.email_alerts.saved'), 'success')
    } catch {
      setAlertEmails(prev)
      showToast(t('settings.email_alerts.save_error'), 'error')
      throw new Error('save failed')
    } finally {
      setSavingRecipients(false)
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

  const handleEmailAlertsEnabledChange = async (value: boolean) => {
    setEmailAlertsEnabled(value)
    try {
      await api.systemSettings.update({ email_alerts_enabled: value })
    } catch {
      setEmailAlertsEnabled(!value)
      showToast(t('settings.email_alerts.save_error'), 'error')
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

  const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px]'
  const labelCls = 'text-sm font-medium text-text-secondary mb-1'

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border flex items-center justify-between gap-3">
          <div className="flex-1">
            <div className="font-medium">{t('settings.email_alerts.enable')}</div>
            <div className="text-sm text-text-secondary">{t('settings.email_alerts.enable_hint')}</div>
          </div>
          <Toggle checked={emailAlertsEnabled} onChange={handleEmailAlertsEnabledChange} />
        </div>

        {emailNeedsConfig && (
          <div className="px-4 py-3 border-b border-border">
            <p className="text-xs text-amber-600 dark:text-amber-400">{t('settings.email_alerts.needs_config_hint')}</p>
          </div>
        )}

        {emailAlertsEnabled && (
          <>
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
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300">
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
                    <HelpPanel title={t('settings.email_alerts.help.title')}>
                      <ol className="list-decimal pl-5 space-y-2">
                        <li>
                          {t('settings.email_alerts.help.step1')}{' '}
                          <a href="https://app.mailgun.com/mg/sending/paulomcnally.com/settings?tab=smtp" target="_blank" rel="noopener noreferrer" className="text-primary underline">
                            {t('settings.email_alerts.help.open_link')}
                          </a>
                        </li>
                        <li>{t('settings.email_alerts.help.step2')}</li>
                        <li>{t('settings.email_alerts.help.step3')}</li>
                        <li>{t('settings.email_alerts.help.step4')}</li>
                        <li>{t('settings.email_alerts.help.step5')}</li>
                        <li>{t('settings.email_alerts.help.step6')}</li>
                      </ol>
                    </HelpPanel>
                  </div>
                )}
              </>
            )}

            <div className="px-4 py-3.5 border-b border-border">
              <div className="font-medium">{t('settings.email_alerts.recipients')}</div>
              <div className="text-sm text-text-secondary mb-3">{t('settings.email_alerts.recipients_hint')}</div>
              <div className="flex flex-wrap gap-2">
                {alertEmails.map((email) => (
                  <span
                    key={email}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-primary/10 text-primary text-sm min-h-[32px]"
                  >
                    {email}
                    <button
                      type="button"
                      onClick={() => handleRecipientsChange(alertEmails.filter((e) => e !== email)).catch(() => {})}
                      disabled={savingRecipients}
                      className="w-5 h-5 flex items-center justify-center rounded-full hover:bg-primary/20 transition-colors disabled:opacity-50"
                      aria-label={t('settings.email_alerts.remove_recipient')}
                    >
                      <Icon name="cancel" className="w-3 h-3" />
                    </button>
                  </span>
                ))}
              </div>
              <button
                type="button"
                onClick={() => setShowRecipientModal(true)}
                disabled={savingRecipients}
                className="mt-3 px-4 py-2 rounded-ios-sm border border-border font-medium min-h-[44px] disabled:opacity-50"
              >
                {t('settings.email_alerts.add_recipient')}
              </button>
            </div>

            <div className="px-4 py-3.5 flex flex-col sm:flex-row gap-3">
              {!smtpConfigured && (
                <button
                  onClick={handleSaveEmail}
                  disabled={savingEmail}
                  className="flex-1 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
                >
                  {savingEmail ? '...' : t('app.save')}
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
          </>
        )}
      </div>

      {showRecipientModal && (
        <EmailRecipientsModal
          existing={alertEmails}
          onAdd={(email) => handleRecipientsChange([...alertEmails, email])}
          onClose={() => setShowRecipientModal(false)}
        />
      )}
    </div>
  )
}