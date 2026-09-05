import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useAuthStore } from '../stores/authStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from '../components/Icons'
import Sidebar from '../components/Sidebar'

const BACK_ROUTES: { pattern: RegExp; to: string }[] = [
  { pattern: /^\/deudas\/\d+$/, to: '/deudas' },
  { pattern: /^\/deudas\/(new|edit\/\d+)$/, to: '/deudas' },
  { pattern: /^\/services\/bills\/\d+$/, to: '/services' },
  { pattern: /^\/services\/(new|edit\/\d+)$/, to: '/services' },
  { pattern: /^\/bills\/(new|edit\/\d+)$/, to: '/services' },
  { pattern: /^\/autos\/\d+$/, to: '/autos' },
  { pattern: /^\/autos\/(new|edit\/\d+)$/, to: '/autos' },
  { pattern: /^\/home\/(new|edit\/\d+)$/, to: '/home' },
  { pattern: /^\/institutions\/(new|edit\/\d+)$/, to: '/institutions' },
  { pattern: /^\/pension\/hijos\/(new|edit\/\d+)$/, to: '/pension/hijos' },
  { pattern: /^\/pension\/categorias\/(new|edit\/\d+)$/, to: '/pension/categorias' },
  { pattern: /^\/pension\/salarios\/(new|edit\/\d+)$/, to: '/pension/salarios' },
  { pattern: /^\/pension\/notificaciones\/(new|edit\/\d+)$/, to: '/pension/notificaciones' },
  { pattern: /^\/settings\/(language|currency(\/\d+)?)$/, to: '/settings' },
]

export default function DashboardLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useI18nStore()
  const { logout } = useAuthStore()
  const loadCurrencyFormat = useCurrencyFormatStore(s => s.load)
  const [sidebarOpen, setSidebarOpen] = useState(false)

  useEffect(() => {
    loadCurrencyFormat()
  }, [loadCurrencyFormat])

  const activeBase = location.pathname.split('/')[1] || 'home'

  const backRoute = BACK_ROUTES.find(r => r.pattern.test(location.pathname))

  const handleLogout = async () => {
    await logout()
  }

  return (
    <div className="flex min-h-screen">
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/30 z-40 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}
      <Sidebar
        activeBase={activeBase}
        isOpen={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
      />
      <div className="flex-1 ml-0 lg:ml-60 flex flex-col min-h-screen">
        <header className="h-14 bg-card border-b border-border flex items-center justify-between px-3 sm:px-5 sticky top-0 z-50">
          <div className="flex items-center gap-2">
            {backRoute ? (
              <button
                onClick={() => navigate(backRoute.to)}
                className="lg:hidden w-10 h-10 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
                title={t('app.back')}
              >
                <Icon name="back" className="w-5 h-5" />
              </button>
            ) : (
              <button
                onClick={() => setSidebarOpen(true)}
                className="lg:hidden w-10 h-10 rounded-full flex items-center justify-center hover:bg-bg transition-colors min-h-[44px]"
                title={t('menu.home')}
              >
                <Icon name="menu" className="w-5 h-5" />
              </button>
            )}
            <h1 className="text-base sm:text-lg font-semibold">{t(`${activeBase}.title`, t('app.title'))}</h1>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => navigate('/settings')}
              className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
              title={t('menu.settings')}
            >
              <Icon name="settings" className="w-5 h-5" />
            </button>
            <button
              onClick={handleLogout}
              className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
              title={t('app.close')}
            >
              <Icon name="logout" className="w-5 h-5" />
            </button>
          </div>
        </header>
        <main className="flex-1 p-3 sm:p-5 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
