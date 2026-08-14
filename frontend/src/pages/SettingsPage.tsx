import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from '../components/Icons'

export default function SettingsPage() {
  const navigate = useNavigate()
  const { currencies, loadCurrencies } = useAppStore()
  const { t, lang } = useI18nStore()

  useEffect(() => {
    loadCurrencies()
  }, [])

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold mb-6">{t('settings.title')}</h2>

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
              <span>{t(`settings.language.${lang}`)}</span>
              <Icon name="chevron" className="w-4 h-4" />
            </div>
          </button>
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
