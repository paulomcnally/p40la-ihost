import { useEffect, useState } from 'react'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import Select from '../components/Select'
import { formatCurrency } from '../utils/currency'
import { useToast } from '../components/Toast'

export default function SettingsCurrencyFormatPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.currency_format.title'))
  const { showToast } = useToast()
  const { thousandsSeparator, decimalSeparator, decimalDigits, updateFormat } = useCurrencyFormatStore()
  const loadCurrencyFormat = useCurrencyFormatStore(s => s.load)
  const [cfThousands, setCfThousands] = useState(thousandsSeparator)
  const [cfDecimal, setCfDecimal] = useState(decimalSeparator)
  const [cfDigits, setCfDigits] = useState(decimalDigits)
  const [savingFormat, setSavingFormat] = useState(false)

  useEffect(() => {
    loadCurrencyFormat()
  }, [loadCurrencyFormat])

  useEffect(() => {
    setCfThousands(thousandsSeparator)
    setCfDecimal(decimalSeparator)
    setCfDigits(decimalDigits)
  }, [thousandsSeparator, decimalSeparator, decimalDigits])

  const applyFormatPreset = (preset: 'ni' | 'us' | 'eu') => {
    if (preset === 'eu') {
      setCfThousands('.')
      setCfDecimal(',')
    } else {
      setCfThousands(',')
      setCfDecimal('.')
    }
    setCfDigits(2)
  }

  const handleSaveFormat = async () => {
    setSavingFormat(true)
    try {
      await updateFormat({ thousandsSeparator: cfThousands, decimalSeparator: cfDecimal, decimalDigits: cfDigits })
      showToast(t('settings.currency_format.saved'), 'success')
    } catch {
      showToast(t('settings.currency_format.save_error'), 'error')
    } finally {
      setSavingFormat(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium">{t('settings.currency_format.presets')}</div>
          <div className="text-sm text-text-secondary mb-3">{t('settings.currency_format.presets_hint')}</div>
          <div className="flex flex-wrap gap-2">
            {(['ni', 'us', 'eu'] as const).map((preset) => (
              <button
                key={preset}
                type="button"
                onClick={() => applyFormatPreset(preset)}
                className={`px-3 py-1.5 rounded-full text-sm border transition-colors ${
                  (preset === 'eu'
                    ? cfThousands === '.' && cfDecimal === ','
                    : cfThousands === ',' && cfDecimal === '.') && cfDigits === 2
                    ? 'bg-primary/10 border-primary text-primary'
                    : 'border-border'
                }`}
              >
                {t(`settings.currency_format.preset_${preset}`)}
              </button>
            ))}
          </div>
        </div>
        <div className="px-4 py-3.5 border-b border-border">
          <label className="block text-sm font-medium mb-1">{t('settings.currency_format.thousands')}</label>
          <Select
            options={[
              { value: ',', label: t('settings.currency_format.sep_comma') },
              { value: '.', label: t('settings.currency_format.sep_point') },
              { value: ' ', label: t('settings.currency_format.sep_space') },
              { value: "'", label: t('settings.currency_format.sep_apostrophe') },
              { value: '', label: t('settings.currency_format.sep_none') },
            ]}
            value={cfThousands}
            onChange={(v) => setCfThousands(String(v))}
          />
        </div>
        <div className="px-4 py-3.5 border-b border-border">
          <label className="block text-sm font-medium mb-1">{t('settings.currency_format.decimal')}</label>
          <Select
            options={[
              { value: '.', label: t('settings.currency_format.sep_point') },
              { value: ',', label: t('settings.currency_format.sep_comma') },
            ]}
            value={cfDecimal}
            onChange={(v) => setCfDecimal(String(v))}
          />
        </div>
        <div className="px-4 py-3.5 border-b border-border">
          <label className="block text-sm font-medium mb-1">{t('settings.currency_format.digits')}</label>
          <Select
            options={[0, 1, 2, 3, 4].map((d) => ({ value: d, label: String(d) }))}
            value={cfDigits}
            onChange={(v) => setCfDigits(Number(v))}
          />
        </div>
        <div className="px-4 py-3.5 border-b border-border">
          <div className="text-sm text-text-secondary mb-1">{t('settings.currency_format.preview')}</div>
          <p className="text-lg font-semibold">
            C${formatCurrency(1234567.5, { thousandsSeparator: cfThousands, decimalSeparator: cfDecimal, decimalDigits: cfDigits })}
          </p>
        </div>
        <div className="px-4 py-3.5 flex justify-end">
          <button
            onClick={handleSaveFormat}
            disabled={savingFormat}
            className="px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
          >
            {savingFormat ? '...' : t('app.save')}
          </button>
        </div>
      </div>
    </div>
  )
}