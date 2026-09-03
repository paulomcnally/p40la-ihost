import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'

interface SidebarProps {
  activeBase: string
  isOpen?: boolean
  onClose?: () => void
}

export default function Sidebar({ activeBase, isOpen, onClose }: SidebarProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useI18nStore()
  const [pensionOpen, setPensionOpen] = useState(false)

  const items = [
    { key: 'home', icon: 'home', label: t('menu.home') },
    { key: 'services', icon: 'services', label: t('menu.services') },
    { key: 'institutions', icon: 'building', label: 'Instituciones' },
    { key: 'autos', icon: 'vehicle', label: 'Autos' },
  ]

  const pensionItems = [
    { path: '/pension/hijos', icon: 'baby', label: t('pension.children') },
    { path: '/pension/categorias', icon: 'tag', label: t('pension.categories') },
    { path: '/pension/salarios', icon: 'savings', label: t('pension.salaries') },
    { path: '/pension/registros', icon: 'calendar', label: t('pension.records') },
    { path: '/pension/notificaciones', icon: 'bell', label: t('pension.notifications') },
  ]

  const isPensionActive = activeBase === 'pension'

  const handleNavigate = (key: string) => {
    navigate(`/${key}`)
    onClose?.()
  }

  return (
    <aside className={`
      bg-card border-border flex flex-col
      fixed top-0 bottom-0 z-50
      w-60 left-0
      lg:fixed lg:left-0 lg:top-0 lg:bottom-0
      border-r
      transition-transform duration-200 ease-in-out
      ${isOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
    `}>
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
              onClick={() => handleNavigate(item.key)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-ios-sm text-sm transition-colors min-h-[44px] ${
                activeBase === item.key
                  ? 'bg-primary/10 text-primary font-medium'
                  : 'hover:bg-bg'
              }`}
            >
              <Icon name={item.icon} className="w-5 h-5" />
              <span>{item.label}</span>
            </button>
          ))}
          <button
            onClick={() => setPensionOpen((o) => !o)}
            className={`w-full flex items-center justify-between gap-3 px-3 py-2.5 rounded-ios-sm text-sm transition-colors min-h-[44px] ${
              isPensionActive
                ? 'bg-primary/10 text-primary font-medium'
                : 'hover:bg-bg'
            }`}
          >
            <span className="flex items-center gap-3">
              <Icon name="pension" className="w-5 h-5" />
              <span>{t('menu.pension')}</span>
            </span>
            <Icon
              name="chevron"
              className={`w-4 h-4 transition-transform ${pensionOpen ? 'rotate-90' : ''}`}
            />
          </button>
          {pensionOpen && (
            <div className="mt-1 space-y-1">
              {pensionItems.map((item) => {
                const isActive = location.pathname === item.path
                return (
                  <button
                    key={item.path}
                    onClick={() => handleNavigate(item.path.slice(1))}
                    className={`w-full flex items-center gap-3 pl-9 pr-3 py-2 rounded-ios-sm text-sm transition-colors min-h-[44px] ${
                      isActive
                        ? 'bg-primary/10 text-primary font-medium'
                        : 'hover:bg-bg'
                    }`}
                  >
                    <Icon name={item.icon} className="w-5 h-5" />
                    <span>{item.label}</span>
                  </button>
                )
              })}
            </div>
          )}
        </div>
      </nav>
    </aside>
  )
}
