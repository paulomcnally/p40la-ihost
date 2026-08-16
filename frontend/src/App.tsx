import { useEffect, useState } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './stores/authStore'
import { useI18nStore } from './stores/i18nStore'
import { ToastProvider } from './components/Toast'
import DashboardLayout from './components/DashboardLayout'
import LoginPage from './pages/LoginPage'
import SetupPage from './pages/SetupPage'
import HomesPage from './pages/HomesPage'
import HomeFormPage from './pages/HomeFormPage'
import ServicesPage from './pages/ServicesPage'
import ServiceFormPage from './pages/ServiceFormPage'
import BillsPage from './pages/BillsPage'
import BillFormPage from './pages/BillFormPage'
import SettingsPage from './pages/SettingsPage'
import LanguagePage from './pages/LanguagePage'
import CurrencyFormPage from './pages/CurrencyFormPage'
import InstitutionsPage from './pages/InstitutionsPage'
import InstitutionFormPage from './pages/InstitutionFormPage'
import AutosPage from './pages/AutosPage'
import AutoFormPage from './pages/AutoFormPage'

function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore()
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

function App() {
  const { checkSetup, checkSession, isSetupComplete } = useAuthStore()
  const { lang, load } = useI18nStore()
  const [initialized, setInitialized] = useState(false)

  useEffect(() => {
    const init = async () => {
      await checkSetup()
      await checkSession()
      await load(lang)
      setInitialized(true)
    }
    init()
  }, [])

  if (!initialized) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-bg">
        <div className="text-text-secondary text-lg">Loading...</div>
      </div>
    )
  }

  if (isSetupComplete === false) {
    return <SetupPage />
  }

  return (
    <ToastProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/*"
          element={
            <AuthGuard>
              <Routes>
                <Route path="/" element={<DashboardLayout />}>
                  <Route index element={<Navigate to="/home" replace />} />
                  <Route path="home" element={<HomesPage />} />
                  <Route path="home/new" element={<HomeFormPage />} />
                  <Route path="home/edit/:id" element={<HomeFormPage />} />
                  <Route path="services" element={<ServicesPage />} />
                  <Route path="services/new" element={<ServiceFormPage />} />
                  <Route path="services/edit/:id" element={<ServiceFormPage />} />
                  <Route path="services/bills/:serviceId" element={<BillsPage />} />
                  <Route path="bills/new" element={<BillFormPage />} />
                  <Route path="bills/edit/:id" element={<BillFormPage />} />
                  <Route path="settings" element={<SettingsPage />} />
                  <Route path="settings/language" element={<LanguagePage />} />
                  <Route path="settings/currency" element={<CurrencyFormPage />} />
                  <Route path="settings/currency/:id" element={<CurrencyFormPage />} />
                  <Route path="institutions" element={<InstitutionsPage />} />
                  <Route path="institutions/new" element={<InstitutionFormPage />} />
                  <Route path="institutions/edit/:id" element={<InstitutionFormPage />} />
                  <Route path="autos" element={<AutosPage />} />
                  <Route path="autos/new" element={<AutoFormPage />} />
                  <Route path="autos/edit/:id" element={<AutoFormPage />} />
                </Route>
              </Routes>
            </AuthGuard>
          }
        />
      </Routes>
    </ToastProvider>
  )
}

export default App
