import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import HousePickerModal from '../components/HousePickerModal'
import LoadingSpinner from '../components/LoadingSpinner'
import WebhookModal from '../components/WebhookModal'
import ServiceAnalysis from '../components/ServiceAnalysis'
import ServiceCalendar from '../components/ServiceCalendar'
import ServiceChart from '../components/ServiceChart'
import { useToast } from '../components/Toast'
import { resolveCurrency } from '../utils/currency'
import { formatRelativeTime } from '../utils/relativeTime'
import type { Service } from '../types'

type TabKey = 'analisis' | 'calendario' | 'servicios' | 'grafica'
type ServiceView = 'pending' | 'paid' | 'inactive' | 'all'
type ChartView = 'history' | 'projection'

const TABS: TabKey[] = ['analisis', 'calendario', 'servicios', 'grafica']
const VIEWS: ServiceView[] = ['pending', 'paid', 'inactive', 'all']

export default function ServicesPage() {
  const navigate = useNavigate()
  const { homes, currencies, loadAll } = useAppStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const { t } = useI18nStore()
  const lang = useI18nStore((s) => s.lang)
  const { showToast } = useToast()
  const [searchParams, setSearchParams] = useSearchParams()
  const now = useMemo(() => new Date(), [])

  const rawTab = searchParams.get('tab')
  const tab: TabKey = TABS.includes(rawTab as TabKey) ? (rawTab as TabKey) : 'analisis'

  const rawStatus = searchParams.get('status')
  const status: ServiceView = VIEWS.includes(rawStatus as ServiceView) ? (rawStatus as ServiceView) : 'pending'

  const rawGview = searchParams.get('gview')
  const gview: ChartView = rawGview === 'projection' ? 'projection' : 'history'

  const rawYear = Number(searchParams.get('year'))
  const rawMonth = Number(searchParams.get('month'))
  const year = Number.isInteger(rawYear) && rawYear >= 1900 && rawYear <= 2100 ? rawYear : now.getFullYear()
  const month = Number.isInteger(rawMonth) && rawMonth >= 1 && rawMonth <= 12 ? rawMonth : now.getMonth() + 1

  const rawHome = Number(searchParams.get('home'))
  const homeParam = Number.isInteger(rawHome) && rawHome > 0 ? rawHome : null
  const homeId = homeParam && homes.some((h) => h.id === homeParam) ? homeParam : null

  const tabTitle = tab === 'analisis'
    ? t('services.tab_analysis')
    : tab === 'calendario'
      ? t('services.tab_calendar')
      : tab === 'servicios'
        ? t('services.tab_services')
        : t('services.tab_chart')
  usePageTitle(tabTitle)

  const [services, setServices] = useState<Service[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [pickerOpen, setPickerOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [servicesLoading, setServicesLoading] = useState(true)
  const [webhookService, setWebhookService] = useState<Service | null>(null)
  const [syncingId, setSyncingId] = useState<number | null>(null)

  const updateParams = useCallback((updates: Record<string, string | null>) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      for (const [key, value] of Object.entries(updates)) {
        if (value === null || value === '') next.delete(key)
        else next.set(key, value)
      }
      return next
    }, { replace: false })
  }, [setSearchParams])

  useEffect(() => {
    loadAll().finally(() => setLoading(false))
  }, [loadAll])

  const loadServices = useCallback(async () => {
    setServicesLoading(true)
    try {
      const list = await api.services.list(homeId ?? undefined)
      setServices(list || [])
    } catch {
      setServices([])
    } finally {
      setServicesLoading(false)
    }
  }, [homeId])

  useEffect(() => {
    loadServices()
  }, [loadServices])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.services.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadServices()
  }, [deleteTarget, loadServices])

  const handleSync = useCallback(async (svc: Service) => {
    setSyncingId(svc.id)
    try {
      const res = await api.services.sync(svc.id)
      if (res) {
        showToast(`${t('services.sync_ok')} (${res.delivered}/${res.failed})`, 'success')
        loadServices()
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('services.sync_error'), 'error')
    } finally {
      setSyncingId(null)
    }
  }, [loadServices, showToast, t])

  const filteredServices = useMemo(() => {
    const sorted = [...services].sort((a, b) => a.name.localeCompare(b.name))
    switch (status) {
      case 'pending':
        return sorted.filter((svc) => svc.active && svc.latest_bill_status === 'pending')
      case 'paid':
        return sorted.filter((svc) => svc.active && svc.latest_bill_status === 'paid')
      case 'inactive':
        return sorted.filter((svc) => !svc.active)
      default:
        return sorted
    }
  }, [services, status])

  if (loading) return <LoadingSpinner />

  if (homes.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="home" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('services.empty_no_home')}</h3>
        <p className="text-text-secondary mb-6">{t('services.subtitle')}</p>
        <button
          onClick={() => navigate('/home/new')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
        >
          <Icon name="plus" className="w-5 h-5" />
          {t('home.create')}
        </button>
      </div>
    )
  }

  if (currencies.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="savings" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('services.empty_no_currency')}</h3>
        <button
          onClick={() => navigate('/settings/currency')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
        >
          <Icon name="plus" className="w-5 h-5" />
          {t('services.new_currency')}
        </button>
      </div>
    )
  }

  const createOptions = [
    { label: t('services.create'), icon: 'plus', onClick: () => navigate('/services/new') },
  ]

  const counts = {
    pending: services.filter((svc) => svc.active && svc.latest_bill_status === 'pending').length,
    paid: services.filter((svc) => svc.active && svc.latest_bill_status === 'paid').length,
    inactive: services.filter((svc) => !svc.active).length,
    all: services.length,
  }

  const emptyViewMessage = status === 'pending'
    ? t('services.empty_pending')
    : status === 'paid'
      ? t('services.empty_paid')
      : status === 'inactive'
        ? t('services.empty_inactive')
        : t('services.empty')

  const renderServiceCard = (svc: Service) => {
    const currency = resolveCurrency(currencies, { id: svc.currency_id })
    const home = homes.find((h) => h.id === svc.home_id)
    const isExpired = svc.is_recurring && svc.end_date && new Date(svc.end_date) < new Date()
    return (
      <div
        key={svc.id}
        onClick={() => navigate(`/services/bills/${svc.id}`)}
        className={`rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow ${
          svc.active ? 'bg-card' : 'bg-gray-100 opacity-60 dark:bg-[#2c2c2e]'
        }`}
      >
        <CardMenu
          options={[
            { label: t('services.webhook_title'), icon: 'link', onClick: () => setWebhookService(svc) },
            { label: syncingId === svc.id ? t('services.syncing') : t('services.sync'), icon: 'refresh', onClick: () => handleSync(svc) },
            { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/services/edit/${svc.id}`) },
            { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: svc.id, name: svc.name }) },
          ]}
        />
        <div className="flex items-center justify-between mb-3">
          <div className={`w-11 h-11 rounded-ios flex items-center justify-center ${
            svc.active ? 'bg-primary/10 text-primary' : 'bg-gray-200 text-gray-400 dark:bg-[#2c2c2e] dark:text-gray-500'
          }`}>
            <Icon name={svc.icon_key || 'other'} className="w-6 h-6" />
          </div>
        </div>
        <h3 className="font-semibold text-base truncate flex items-center gap-2">
          {svc.name}
          <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400 shrink-0">
            {`${t('services.id')}: ${svc.id}`}
          </span>
        </h3>
        <p className="text-sm text-text-secondary mt-1">
          {svc.institution && `${svc.institution} · `}{home?.name || ''}
        </p>
        <div className="flex items-center gap-2 mt-2 flex-wrap">
          {!svc.active && (
            <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400">
              {t('services.inactive')}
            </span>
          )}
          {isExpired && (
            <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300">
              {t('services.expired')}
            </span>
          )}
          <span className={`inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full ${
            svc.last_webhook_request
              ? 'bg-primary/10 text-primary'
              : 'bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400'
          }`}>
            <Icon name="link" className="w-3 h-3" />
            {svc.last_webhook_request
              ? formatRelativeTime(svc.last_webhook_request, lang)
              : t('services.webhook_never')}
          </span>
          {svc.is_recurring && svc.start_date && (
            <span className="text-xs text-text-secondary">
              {svc.start_date}{svc.end_date ? ` → ${svc.end_date}` : ' → ∞'}
            </span>
          )}
        </div>
        <div className="flex items-center justify-between mt-1">
          <p className="text-xs text-text-secondary">
            {formatMoney(svc.suggested_amount, currency.symbol || currency.code)} · {t(`frequency.${svc.frequency}`)}
          </p>
          <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
            svc.latest_bill_status === 'paid' ? 'bg-success/20 text-green-800 dark:text-green-400' :
            svc.latest_bill_status === 'pending' ? 'bg-warning/20 text-yellow-800 dark:text-yellow-400' :
            'bg-gray-100 text-gray-500 dark:bg-[#2c2c2e] dark:text-gray-400'
          }`}>
            {svc.latest_bill_status === 'paid' ? t('bills.status_paid') :
             svc.latest_bill_status === 'pending' ? t('bills.status_pending') :
             t('bills.no_bills')}
          </span>
        </div>
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between gap-2 mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('services.title')}</h2>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setPickerOpen(true)}
            className="inline-flex items-center gap-2 px-3 py-2 border border-border rounded-ios-sm bg-card hover:border-primary/50 focus:outline-none focus:border-primary transition-colors text-sm"
          >
            <Icon name="home" className="w-4 h-4 text-text-secondary shrink-0" />
            <span className={homeId ? 'max-w-32 truncate' : 'text-text-secondary'}>
              {homeId ? homes.find((h) => h.id === homeId)?.name : t('services.homes')}
            </span>
            <Icon name="chevron" className="w-4 h-4 text-text-secondary shrink-0" />
          </button>
          <CreateMenu options={createOptions} />
        </div>
      </div>

      <div className="flex flex-wrap gap-2 mb-4">
        {([
          { key: 'analisis', label: t('services.tab_analysis'), icon: 'chart' },
          { key: 'calendario', label: t('services.tab_calendar'), icon: 'calendar' },
          { key: 'servicios', label: t('services.tab_services'), icon: 'services' },
          { key: 'grafica', label: t('services.tab_chart'), icon: 'bar' },
        ] as { key: TabKey; label: string; icon: string }[]).map((item) => (
          <button
            key={item.key}
            onClick={() => updateParams({ tab: item.key })}
            aria-pressed={tab === item.key}
            className={`px-4 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] inline-flex items-center gap-1.5 ${
              tab === item.key ? 'bg-primary text-white' : 'bg-card text-text-secondary hover:bg-border'
            }`}
          >
            <Icon name={item.icon} className="w-4 h-4" />
            {item.label}
          </button>
        ))}
      </div>

      {tab === 'analisis' ? (
        <ServiceAnalysis
          homeId={homeId}
          year={year}
          month={month}
          onChangePeriod={(y, m) => updateParams({ year: String(y), month: String(m) })}
        />
      ) : tab === 'calendario' ? (
        <ServiceCalendar
          homeId={homeId}
          year={year}
          month={month}
          onChangePeriod={(y, m) => updateParams({ year: String(y), month: String(m) })}
        />
      ) : tab === 'grafica' ? (
        <ServiceChart
          homeId={homeId}
          year={year}
          month={month}
          gview={gview}
          onChangeGview={(view) => updateParams({ gview: view })}
        />
      ) : servicesLoading ? (
        <LoadingSpinner />
      ) : services.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="services" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('services.empty')}</h3>
          <p className="text-text-secondary mb-6">{t('services.subtitle')}</p>
          <button
            onClick={() => navigate('/services/new')}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('services.create')}
          </button>
        </div>
      ) : (
        <>
          <div className="flex gap-2 mb-4 flex-wrap">
            {VIEWS.map((key) => (
              <button
                key={key}
                onClick={() => updateParams({ status: key })}
                className={`px-4 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
                  status === key ? 'bg-primary text-white' : 'bg-card text-text-secondary hover:bg-border'
                }`}
              >
                {t(`services.view_${key}`)} ({counts[key]})
              </button>
            ))}
          </div>

          {filteredServices.length === 0 ? (
            <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
              <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
                <Icon name={status === 'paid' ? 'check' : 'services'} className="w-full h-full" />
              </div>
              <h3 className="text-lg sm:text-xl font-semibold mb-2">{emptyViewMessage}</h3>
              <p className="text-text-secondary mb-6">{t('services.subtitle')}</p>
              <button
                onClick={() => navigate('/services/new')}
                className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
              >
                <Icon name="plus" className="w-5 h-5" />
                {t('services.create')}
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
              {filteredServices.map(renderServiceCard)}
            </div>
          )}
        </>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('services.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
      <HousePickerModal
        isOpen={pickerOpen}
        homes={homes}
        selectedHomeId={homeId}
        onSelect={(id) => updateParams({ home: id ? String(id) : null })}
        onClose={() => setPickerOpen(false)}
      />
      {webhookService && (
        <WebhookModal service={webhookService} onClose={() => setWebhookService(null)} />
      )}
    </div>
  )
}
