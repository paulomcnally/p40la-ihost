import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'

export default function Sidebar({ activeBase }: { activeBase: string }) {
  const navigate = useNavigate()
  const { t } = useI18nStore()

  const items = [
    { key: 'home', icon: 'home', label: t('menu.home') },
    { key: 'services', icon: 'services', label: t('menu.services') },
    { key: 'institutions', icon: 'building', label: 'Instituciones' },
  ]

  return (
    <aside className="w-60 bg-card border-r border-border fixed top-0 left-0 bottom-0 z-50 flex flex-col">
      <div className="px-5 py-4 text-xl font-bold text-primary border-b border-border">
        {t('app.title')}
      </div>
      <nav className="flex-1 p-3 overflow-y-auto">
        <div className="mb-4">
          <div className="text-xs uppercase tracking-wide text-text-secondary font-semibold px-3 py-2">
            {t('app.title')}
          </div>
          {items.map((item) => (
            <button
              key={item.key}
              onClick={() => navigate(`/${item.key}`)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-ios-sm text-sm transition-colors ${
                activeBase === item.key
                  ? 'bg-primary/10 text-primary font-medium'
                  : 'hover:bg-bg'
              }`}
            >
              <Icon name={item.icon} className="w-5 h-5" />
              <span>{item.label}</span>
            </button>
          ))}
        </div>
      </nav>
    </aside>
  )
}
