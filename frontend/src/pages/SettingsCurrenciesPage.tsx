import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { Icon } from '../components/Icons'

export default function SettingsCurrenciesPage() {
  const navigate = useNavigate()
  const { currencies, loadCurrencies } = useAppStore()
  const { t } = useI18nStore()
  usePageTitle(t('settings.nav.currencies'))

  useEffect(() => {
    loadCurrencies()
  }, [loadCurrencies])

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        {currencies.map((c) => (
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
  )
}