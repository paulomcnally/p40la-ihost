import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { useToast } from '../components/Toast'
import Select from '../components/Select'
import { Icon } from '../components/Icons'
import {
  DEFAULT_COUNTRY_CODE,
  countryName,
  countryOptions,
  resolveCountryCode,
} from '../constants/countries'

export default function SettingsCountryPage() {
  const { t, lang } = useI18nStore()
  usePageTitle(t('settings.country.title'))
  const { showToast } = useToast()
  const [countryCode, setCountryCode] = useState(DEFAULT_COUNTRY_CODE)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let mounted = true
    api.systemSettings
      .get()
      .then((data) => {
        if (!mounted || !data) return
        setCountryCode(resolveCountryCode(data.country_code || '') || DEFAULT_COUNTRY_CODE)
      })
      .catch(() => {
        // El default NI sigue disponible si falla la lectura.
      })
      .finally(() => {
        if (mounted) setLoading(false)
      })
    return () => {
      mounted = false
    }
  }, [])

  const handleSave = async (code: string) => {
    setSaving(true)
    try {
      await api.systemSettings.update({ country_code: code })
      setCountryCode(code)
      showToast(t('settings.country.saved'), 'success')
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const options = countryOptions(lang)

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5">
          <div className="font-medium">{t('settings.country.title')}</div>
          <div className="text-sm text-text-secondary mb-3">
            {t('settings.country.description')}
          </div>
          {loading ? (
            <div className="py-3 text-sm text-text-secondary">{t('app.loading')}</div>
          ) : (
            <>
              <Select
                options={options}
                value={countryCode}
                onChange={(value) => handleSave(String(value))}
                searchable
                searchPlaceholder={t('banks.country_search')}
                placeholder={t('banks.country_placeholder')}
              />
              <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
                <p className="text-sm text-text-secondary">
                  {countryName(countryCode, lang)}
                </p>
                <button
                  type="button"
                  onClick={() => handleSave(DEFAULT_COUNTRY_CODE)}
                  disabled={saving || countryCode === DEFAULT_COUNTRY_CODE}
                  className="inline-flex items-center justify-center gap-2 px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border disabled:opacity-50 transition-colors min-h-[44px]"
                >
                  <Icon name="refresh" className="w-4 h-4" />
                  {t('settings.country.restore')}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
