import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import Toggle from '../components/Toggle'
import Select from '../components/Select'
import { api } from '../api'
import { useToast } from '../components/Toast'
import { getBrowserTimeZone, getTimeZoneOptions } from '../constants/timezones'

type HourFormat = '12h' | '24h'

const HOUR_FORMAT_KEY = 'hourFormat'

function getInitialHourFormat(): HourFormat {
  return localStorage.getItem(HOUR_FORMAT_KEY) === '24h' ? '24h' : '12h'
}

function formatHourLabel(hour: number, format: HourFormat): string {
  if (format === '24h') {
    return `${String(hour).padStart(2, '0')}:00`
  }
  const period = hour < 12 ? 'AM' : 'PM'
  const h12 = hour % 12 === 0 ? 12 : hour % 12
  return `${h12}:00 ${period}`
}

export default function SettingsBillingPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.nav.billing'))
  const { showToast } = useToast()
  const [billingHour, setBillingHour] = useState(0)
  const [alertCheckHour, setAlertCheckHour] = useState(0)
  const [timezone, setTimezone] = useState('')
  const [hourFormat, setHourFormat] = useState<HourFormat>(getInitialHourFormat)

  useEffect(() => {
    loadBillingHour()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadBillingHour = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setBillingHour(data.billing_generation_hour ?? 0)
        setAlertCheckHour(data.alert_check_hour ?? 0)
        const saved = data.timezone ?? ''
        if (saved) {
          setTimezone(saved)
        } else {
          const browserTz = getBrowserTimeZone()
          setTimezone(browserTz)
          try {
            await api.systemSettings.update({ timezone: browserTz })
          } catch {
            // el fallback UTC del backend sigue activo
          }
        }
      }
    } catch {
      // ignore
    }
  }

  const handleBillingHourChange = async (hour: number) => {
    try {
      await api.systemSettings.update({ billing_generation_hour: hour })
      setBillingHour(hour)
      showToast(t('settings.billing.hour_generation_saved'), 'success')
    } catch {
      showToast(t('settings.billing.save_error'), 'error')
    }
  }

  const handleAlertCheckHourChange = async (hour: number) => {
    try {
      await api.systemSettings.update({ alert_check_hour: hour })
      setAlertCheckHour(hour)
      showToast(t('settings.billing.alert_check_saved'), 'success')
    } catch {
      showToast(t('settings.billing.save_error'), 'error')
    }
  }

  const handleHourFormatChange = (next: boolean) => {
    const format: HourFormat = next ? '24h' : '12h'
    setHourFormat(format)
    localStorage.setItem(HOUR_FORMAT_KEY, format)
  }

  const handleTimezoneChange = async (tz: string) => {
    try {
      await api.systemSettings.update({ timezone: tz })
      setTimezone(tz)
      showToast(t('settings.billing.timezone_saved'), 'success')
    } catch {
      showToast(t('settings.billing.save_error'), 'error')
    }
  }

  const hours = Array.from({ length: 24 }, (_, i) => ({
    value: i,
    label: formatHourLabel(i, hourFormat),
  }))

  const timezoneOptions = getTimeZoneOptions()
  const effectiveTimezone = timezone || getBrowserTimeZone()

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium">{t('settings.billing.timezone')}</div>
          <div className="text-sm text-text-secondary mb-3">
            {t('settings.billing.timezone_desc')}
          </div>
          <Select
            options={timezoneOptions}
            value={effectiveTimezone}
            onChange={(v) => handleTimezoneChange(String(v))}
            searchable
          />
        </div>
        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium">{t('settings.billing.hour_generation')}</div>
          <div className="text-sm text-text-secondary mb-3">{t('settings.billing.hour_generation_desc')}</div>
          <Select
            options={hours}
            value={billingHour}
            onChange={(v) => handleBillingHourChange(Number(v))}
            searchable
          />
        </div>
        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium">{t('settings.billing.alert_check_hour')}</div>
          <div className="text-sm text-text-secondary mb-3">{t('settings.billing.alert_check_hour_desc')}</div>
          <Select
            options={hours}
            value={alertCheckHour}
            onChange={(v) => handleAlertCheckHourChange(Number(v))}
            searchable
          />
        </div>
        <div className="px-4 py-3.5 flex items-center justify-between gap-3">
          <div>
            <div className="font-medium">{t('settings.billing.hour_format')}</div>
            <div className="text-sm text-text-secondary">{t('settings.billing.hour_format_desc')}</div>
          </div>
          <Toggle checked={hourFormat === '24h'} onChange={handleHourFormatChange} />
        </div>
      </div>
    </div>
  )
}