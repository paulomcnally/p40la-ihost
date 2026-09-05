import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useAppStore } from '../stores/appStore'
import { api } from '../api'
import { Icon } from '../components/Icons'

export default function LanguagePage() {
  const navigate = useNavigate()
  const { t, lang, load } = useI18nStore()
  usePageTitle(t('settings.language.title'))
  const { loadAll } = useAppStore()

  const handleSelect = async (newLang: string) => {
    await api.settings.setLanguage(newLang)
    await load(newLang)
    await loadAll()
    navigate('/settings')
  }

  const languages = [
    { code: 'es', label: t('settings.language.es') },
    { code: 'en', label: t('settings.language.en') },
  ]

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3 border-b border-border">
        <button
          onClick={() => navigate('/settings')}
          className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg"
        >
          <Icon name="cancel" className="w-5 h-5" />
        </button>
        <h2 className="text-lg font-semibold">{t('settings.language.title')}</h2>
      </div>
      <div className="divide-y divide-border">
        {languages.map(l => (
          <button
            key={l.code}
            onClick={() => handleSelect(l.code)}
            className="w-full flex items-center justify-between px-4 py-3.5 hover:bg-bg/50 transition-colors"
          >
            <span className="font-medium">{l.label}</span>
            {lang === l.code && <span className="text-success font-bold">✓</span>}
          </button>
        ))}
      </div>
    </div>
  )
}
