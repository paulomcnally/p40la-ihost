import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import SettingsNavRow from '../components/SettingsNavRow'
import type { StatusPillTone } from '../components/StatusPill'
import Toggle from '../components/Toggle'
import { Icon } from '../components/Icons'
import { api } from '../api'
import { formatCurrency } from '../utils/currency'
import type { Alert } from '../types'

type HourFormat = '12h' | '24h'

const HOUR_FORMAT_KEY = 'hourFormat'
const DARK_MODE_KEY = 'darkMode'

function getInitialHourFormat(): HourFormat {
  return localStorage.getItem(HOUR_FORMAT_KEY) === '24h' ? '24h' : '12h'
}

function getInitialDarkMode(): boolean {
  return localStorage.getItem(DARK_MODE_KEY) !== 'light'
}

function formatHourLabel(hour: number, format: HourFormat): string {
  if (format === '24h') {
    return `${String(hour).padStart(2, '0')}:00`
  }
  const period = hour < 12 ? 'AM' : 'PM'
  const h12 = hour % 12 === 0 ? 12 : hour % 12
  return `${h12}:00 ${period}`
}

function parseEmails(value?: string | null): string[] {
  if (!value) return []
  return value
    .split(',')
    .map((e) => e.trim())
    .filter(Boolean)
}

function interpolate(template: string, vars: Record<string, string>): string {
  return template.replace(/\{(\w+)\}/g, (_, k) => vars[k] ?? '')
}

interface IndexRow {
  key: string
  icon: string
  title: string
  subtitle?: string
  statusLabel?: string
  statusTone?: StatusPillTone
  warning?: boolean
  right?: React.ReactNode
  onClick: () => void
}

export default function SettingsPage() {
  const navigate = useNavigate()
  const { currencies, loadCurrencies } = useAppStore()
  const { t } = useI18nStore()
  usePageTitle(t('settings.title'))
  const { thousandsSeparator, decimalSeparator, decimalDigits } = useCurrencyFormatStore()
  const loadCurrencyFormat = useCurrencyFormatStore(s => s.load)

  const [hourFormat, setHourFormat] = useState<HourFormat>(getInitialHourFormat)
  const [darkMode, setDarkMode] = useState<boolean>(getInitialDarkMode)
  const [billingHour, setBillingHour] = useState(0)
  const [alertCheckHour, setAlertCheckHour] = useState(0)
  const [emailAlertsEnabled, setEmailAlertsEnabled] = useState(false)
  const [smtpConfigured, setSmtpConfigured] = useState(false)
  const [alertEmails, setAlertEmails] = useState<string[]>([])
  const [vmEnabled, setVmEnabled] = useState(false)
  const [vmSendAlerts, setVmSendAlerts] = useState(false)
  const [vmConfigured, setVmConfigured] = useState(false)
  const [webhookEnabled, setWebhookEnabled] = useState(false)
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [query, setQuery] = useState('')

  useEffect(() => {
    loadCurrencies()
    loadCurrencyFormat()
    loadSettings()
    loadAlerts()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode)
  }, [darkMode])

  const loadSettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setBillingHour(data.billing_generation_hour ?? 0)
        setAlertCheckHour(data.alert_check_hour ?? 0)
        setEmailAlertsEnabled(data.email_alerts_enabled ?? false)
        setSmtpConfigured(data.smtp_configured ?? false)
        setAlertEmails(parseEmails(data.alert_emails))
        setVmEnabled(data.voicemonkey_enabled ?? false)
        setVmSendAlerts(data.voicemonkey_send_alerts ?? false)
        setVmConfigured(data.voicemonkey_configured ?? false)
        setWebhookEnabled(data.webhook_enabled ?? false)
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

  const handleDarkModeChange = (next: boolean) => {
    setDarkMode(next)
    localStorage.setItem(DARK_MODE_KEY, next ? 'dark' : 'light')
  }

  const vmActive = vmEnabled && vmConfigured && vmSendAlerts
  const emailActive = emailAlertsEnabled && smtpConfigured && alertEmails.length > 0
  const emailNeedsConfig = emailAlertsEnabled && !emailActive
  const vmNeedsConfig = vmEnabled && !vmActive

  const activeAlertsCount = alerts.filter((a) => a.mail_enabled || a.voice_enabled).length
  const formatPreview = `C$${formatCurrency(1234567.5, { thousandsSeparator, decimalSeparator, decimalDigits })}`

  const rows = useMemo<IndexRow[]>(() => {
    const billingSubtitle = `${formatHourLabel(billingHour, hourFormat)} · ${formatHourLabel(alertCheckHour, hourFormat)}`
    const alertsSubtitle = interpolate(t('settings.nav.alerts_subtitle'), {
      active: String(activeAlertsCount),
      total: String(alerts.length),
    })
    const recipientsSubtitle = interpolate(t('settings.nav.recipients_count'), {
      count: String(alertEmails.length),
    })

    return [
      {
        key: 'language',
        icon: 'globe',
        title: t('settings.language.title'),
        subtitle: t('settings.language.subtitle'),
        onClick: () => navigate('/settings/language'),
      },
      {
        key: 'darkMode',
        icon: 'moon',
        title: t('settings.dark_mode.title'),
        subtitle: t('settings.dark_mode.subtitle'),
        right: <Toggle checked={darkMode} onChange={handleDarkModeChange} />,
        onClick: () => handleDarkModeChange(!darkMode),
      },
      {
        key: 'billing',
        icon: 'clock',
        title: t('settings.nav.billing'),
        subtitle: billingSubtitle,
        onClick: () => navigate('/settings/facturacion'),
      },
      {
        key: 'alerts',
        icon: 'bell',
        title: t('settings.alerts.title'),
        subtitle: alertsSubtitle,
        onClick: () => navigate('/settings/alertas'),
      },
      {
        key: 'email',
        icon: 'mail',
        title: t('settings.nav.email_alerts'),
        subtitle: emailNeedsConfig
          ? t('settings.nav.status_pending')
          : emailActive
            ? recipientsSubtitle
            : t('settings.nav.status_deactivated'),
        statusLabel: emailActive
          ? t('settings.nav.status_configured')
          : emailNeedsConfig
            ? t('settings.nav.status_pending')
            : t('settings.nav.status_deactivated'),
        statusTone: emailActive ? 'success' : emailNeedsConfig ? 'warning' : 'muted',
        warning: emailNeedsConfig,
        onClick: () => navigate('/settings/alertas/email'),
      },
      {
        key: 'voice',
        icon: 'radio',
        title: t('settings.nav.voice_monkey'),
        subtitle: vmActive
          ? t('settings.nav.status_configured')
          : vmEnabled
            ? t('settings.nav.status_pending')
            : t('settings.nav.status_deactivated'),
        statusLabel: vmActive
          ? t('settings.nav.status_configured')
          : vmEnabled
            ? t('settings.nav.status_pending')
            : t('settings.nav.status_deactivated'),
        statusTone: vmActive ? 'success' : vmEnabled ? 'warning' : 'muted',
        warning: vmNeedsConfig,
        onClick: () => navigate('/settings/alertas/voz'),
      },
      {
        key: 'webhooks',
        icon: 'link',
        title: t('settings.webhooks.title'),
        subtitle: webhookEnabled
          ? t('settings.nav.status_enabled')
          : t('settings.nav.status_disabled'),
        statusLabel: webhookEnabled
          ? t('settings.nav.status_enabled')
          : t('settings.nav.status_disabled'),
        statusTone: webhookEnabled ? 'success' : 'muted',
        onClick: () => navigate('/settings/webhooks'),
      },
      {
        key: 'currencies',
        icon: 'savings',
        title: t('settings.nav.currencies'),
        subtitle: interpolate(t('settings.nav.currencies_count'), {
          count: String(currencies.length),
        }),
        onClick: () => navigate('/settings/monedas'),
      },
      {
        key: 'currencyFormat',
        icon: 'chart',
        title: t('settings.currency_format.title'),
        subtitle: formatPreview,
        onClick: () => navigate('/settings/formato-moneda'),
      },
    ]
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    billingHour,
    alertCheckHour,
    hourFormat,
    activeAlertsCount,
    alerts.length,
    emailNeedsConfig,
    emailActive,
    alertEmails.length,
    vmActive,
    vmEnabled,
    vmNeedsConfig,
    webhookEnabled,
    currencies.length,
    thousandsSeparator,
    decimalSeparator,
    decimalDigits,
    darkMode,
  ])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return rows
    return rows.filter((r) => `${r.title} ${r.subtitle ?? ''}`.toLowerCase().includes(q))
  }, [rows, query])

  const renderRow = (row: (typeof rows)[number]) => (
    <div key={row.key} className="border-b border-border last:border-b-0">
      <SettingsNavRow
        icon={row.icon}
        title={row.title}
        subtitle={row.subtitle}
        statusLabel={row.statusLabel}
        statusTone={row.statusTone}
        warning={row.warning}
        right={row.right}
        onClick={row.onClick}
      />
    </div>
  )

  const advancedKeys = ['email', 'voice', 'webhooks', 'currencies', 'currencyFormat']
  const frequent = filtered.filter((r) => !advancedKeys.includes(r.key))
  const advanced = filtered.filter((r) => advancedKeys.includes(r.key))

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-xl sm:text-2xl font-bold mb-4 sm:mb-6">{t('settings.title')}</h2>

      <div className="relative mb-6">
        <Icon name="search" className="w-4 h-4 text-text-secondary absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('settings.nav.search')}
          className="w-full pl-9 pr-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card text-text"
        />
      </div>

      {filtered.length === 0 && (
        <div className="text-center py-10 text-text-secondary">{t('settings.nav.no_results')}</div>
      )}

      {frequent.length > 0 && (
        <div className="mb-6">
          <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
            {t('settings.nav.section_frequent')}
          </div>
          <div className="bg-card rounded-ios shadow-ios overflow-hidden">{frequent.map(renderRow)}</div>
        </div>
      )}

      {advanced.length > 0 && (
        <div className="mb-6">
          <div className="text-xs uppercase text-text-secondary font-semibold mb-2 ml-3">
            {t('settings.nav.section_advanced')}
          </div>
          <div className="bg-card rounded-ios shadow-ios overflow-hidden">{advanced.map(renderRow)}</div>
        </div>
      )}
    </div>
  )
}