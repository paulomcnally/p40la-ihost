import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { useAuthStore } from '../stores/authStore'
import { Icon } from '../components/Icons'
import Sidebar from '../components/Sidebar'

export default function DashboardLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useI18nStore()
  const { logout } = useAuthStore()

  const activeBase = location.pathname.split('/')[1] || 'home'

  const handleLogout = async () => {
    await logout()
  }

  return (
    <div className="flex min-h-screen">
      <Sidebar activeBase={activeBase} />
      <div className="flex-1 ml-60 flex flex-col min-h-screen">
        <header className="h-14 bg-card border-b border-border flex items-center justify-between px-5 sticky top-0 z-50">
          <h1 className="text-lg font-semibold">{t(`${activeBase}.title`, t('app.title'))}</h1>
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
        <main className="flex-1 p-5 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
