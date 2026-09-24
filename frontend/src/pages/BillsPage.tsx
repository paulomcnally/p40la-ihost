import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu, { type CardMenuOption } from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import PayBillModal from '../components/PayBillModal'
import UploadBillModal from '../components/UploadBillModal'
import BillHistoryModal from '../components/BillHistoryModal'
import LoadingSpinner from '../components/LoadingSpinner'
import WebhookModal from '../components/WebhookModal'
import BillAnalysis from '../components/BillAnalysis'
import DueDateBadge from '../components/DueDateBadge'
import AddPolicyModal from '../components/AddPolicyModal'
import RenewServiceModal from '../components/RenewServiceModal'
import type { Bill, Service, ServiceAuto } from '../types'

type TabKey = 'analisis' | 'facturas' | 'polizas'

const MONTHS = [
  '', 'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December'
]

export default function BillsPage() {
  const navigate = useNavigate()
  const { serviceId } = useParams()
  const { t } = useI18nStore()
  const { currencies } = useAppStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = (searchParams.get('tab') as TabKey | null) ?? 'analisis'
  const [bills, setBills] = useState<Bill[]>([])
  const [autos, setAutos] = useState<ServiceAuto[]>([])
  const [service, setService] = useState<Service | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null)
  const [payTarget, setPayTarget] = useState<Bill | null>(null)
  const [historyTarget, setHistoryTarget] = useState<Bill | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [webhookOpen, setWebhookOpen] = useState(false)
  const [renewOpen, setRenewOpen] = useState(false)
  const [addPolicyOpen, setAddPolicyOpen] = useState(false)
  const [editPolicy, setEditPolicy] = useState<ServiceAuto | null>(null)
  const [deletePolicyTarget, setDeletePolicyTarget] = useState<{ autoId: number; name: string } | null>(null)

  const loadAutos = useCallback(async () => {
    if (!serviceId) return
    const list = await api.services.listAutos(Number(serviceId))
    setAutos(list || [])
  }, [serviceId])

  const load = useCallback(async () => {
    if (!serviceId) return
    setLoading(true)
    const [svc, billList] = await Promise.all([
      api.services.get(Number(serviceId)),
      api.bills.list(Number(serviceId)),
    ])
    setService(svc)
    setBills(billList || [])
    setLoading(false)
  }, [serviceId])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (tab === 'polizas') loadAutos()
  }, [tab, loadAutos])

  usePageTitle(service?.name ?? null)

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.bills.delete(deleteTarget)
    setDeleteTarget(null)
    load()
  }, [deleteTarget, load])

  const billMenuOptions = (bill: Bill): CardMenuOption[] => {
    const options: CardMenuOption[] = [
      { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/bills/edit/${bill.id}?service=${serviceId}`) },
      { label: t('bills.history'), icon: 'clock', onClick: () => setHistoryTarget(bill) },
    ]
    if (bill.status === 'pending') {
      options.push({ label: t('bills.pay'), icon: 'credit', onClick: () => setPayTarget(bill) })
    }
    options.push({ label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget(bill.id) })
    return options
  }

  const handlePolicySaved = useCallback(() => {
    setAddPolicyOpen(false)
    setEditPolicy(null)
    loadAutos()
  }, [loadAutos])

  const handleDeletePolicy = useCallback(async () => {
    if (!deletePolicyTarget) return
    await api.autos.removeService(deletePolicyTarget.autoId, Number(serviceId))
    setDeletePolicyTarget(null)
    loadAutos()
  }, [deletePolicyTarget, serviceId, loadAutos])

  if (loading) return <LoadingSpinner />

  if (!service) return <div className="text-center py-8 text-text-secondary">Loading...</div>

  const currency = currencies.find(c => c.id === service.currency_id)

  const setTab = (key: TabKey) => {
    setSearchParams({ tab: key }, { replace: false })
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <div>
          <h2 className="text-lg sm:text-2xl font-bold">{service.name}</h2>
          <p className="text-sm text-text-secondary">{t('bills.subtitle')}</p>
        </div>
        <CreateMenu options={[
          { label: 'Subir factura', icon: 'upload', onClick: () => setUploadOpen(true) },
          { label: t('bills.create'), icon: 'plus', onClick: () => navigate(`/bills/new?service=${serviceId}`) },
          { label: t('bills.renew'), icon: 'refresh', onClick: () => setRenewOpen(true) },
          { label: t('services.webhook_title'), icon: 'link', onClick: () => setWebhookOpen(true) },
        ]} />
      </div>

      <div className="flex gap-2 mb-4">
        {(
          [
            { key: 'analisis', label: t('bills.tab_analysis'), icon: 'chart' },
            { key: 'facturas', label: t('bills.tab_bills'), icon: 'bill' },
            ...(service.is_insurance
              ? [{ key: 'polizas' as TabKey, label: t('bills.tab_polizas'), icon: 'insurance' }]
              : []),
          ] as { key: TabKey; label: string; icon: string }[]
        ).map((item) => (
          <button
            key={item.key}
            onClick={() => setTab(item.key)}
            className={`px-4 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] inline-flex items-center gap-1.5 ${
              tab === item.key
                ? 'bg-primary text-white'
                : 'bg-card text-text-secondary hover:bg-border'
            }`}
          >
            <Icon name={item.icon} className="w-4 h-4" />
            {item.label}
          </button>
        ))}
      </div>

      {tab === 'analisis' ? (
        <BillAnalysis serviceId={Number(serviceId)} currencySymbol={currency?.symbol} />
      ) : tab === 'polizas' ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-bold">{t('bills.tab_polizas')}</h3>
            <button
              onClick={() => setAddPolicyOpen(true)}
              className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors flex items-center gap-2 text-sm min-h-[44px]"
            >
              <Icon name="plus" className="w-4 h-4" />
              {t('bills.polizas_add')}
            </button>
          </div>
          {autos.length === 0 ? (
            <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
              <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
                <Icon name="insurance" className="w-full h-full" />
              </div>
              <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('bills.polizas_empty')}</h3>
              <p className="text-text-secondary text-sm mb-5">{t('bills.polizas_empty_subtitle')}</p>
              <button
                onClick={() => setAddPolicyOpen(true)}
                className="inline-flex items-center justify-center gap-2 px-5 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors text-sm min-h-[44px]"
              >
                <Icon name="plus" className="w-4 h-4" />
                {t('bills.polizas_add')}
              </button>
            </div>
          ) : (
            <div className="space-y-3">
              {autos.map((auto) => (
                <div key={auto.auto_id} className="relative bg-card rounded-ios shadow-ios p-4">
                  <CardMenu
                    options={[
                      { label: t('app.edit'), icon: 'edit', onClick: () => setEditPolicy(auto) },
                      {
                        label: t('app.delete'),
                        icon: 'delete',
                        danger: true,
                        onClick: () => setDeletePolicyTarget({ autoId: auto.auto_id, name: `${auto.brand} ${auto.model}` }),
                      },
                    ]}
                  />
                  <button
                    onClick={() => navigate(`/autos/${auto.auto_id}`)}
                    className="w-full text-left flex items-center gap-4"
                  >
                    <div className="w-10 h-10 rounded-ios bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
                      <Icon name={auto.icon || 'vehicle'} className="w-6 h-6" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-semibold truncate">
                        {auto.brand} {auto.model} <span className="text-text-secondary font-normal">{auto.year}</span>
                      </p>
                      <p className="text-xs text-text-secondary truncate">{auto.placa} · {auto.color}</p>
                      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 mt-1.5">
                        <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                          auto.coverage_type === 'full_cover'
                            ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                            : 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
                        }`}>
                          {auto.coverage_type === 'full_cover' ? t('bills.polizas_full_cover') : t('bills.polizas_third_party')}
                        </span>
                        <span className="text-xs text-text-secondary break-all">Póliza: {auto.policy_number}</span>
                        <span className="text-xs text-text-secondary break-all">Aseguradora: {auto.insurer_number}</span>
                        {auto.certificate && (
                          <span className="text-xs text-text-secondary break-all">Certificado: {auto.certificate}</span>
                        )}
                      </div>
                    </div>
                    <Icon name="chevron" className="w-4 h-4 text-text-secondary flex-shrink-0" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      ) : bills.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="bill" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('bills.empty')}</h3>
          <button
            onClick={() => navigate(`/bills/new?service=${serviceId}`)}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('bills.create')}
          </button>
        </div>
      ) : (
        <>
          {/* Mobile: cards */}
          <div className="sm:hidden space-y-3">
            {bills.map(bill => (
              <div key={bill.id} className="bg-card rounded-ios shadow-ios p-4 relative">
                <CardMenu options={billMenuOptions(bill)} />
                <div className="mb-2">
                  <span className="text-sm text-text-secondary">
                    {bill.month === 0 ? t('bills.annual') : t(`months.${bill.month}`, MONTHS[bill.month])} {bill.year}
                  </span>
                </div>
                <p className="text-xl font-semibold mb-2">{formatMoney(bill.amount, currency?.symbol)}</p>
                <div className="flex items-center justify-between text-sm text-text-secondary">
                  <div className="flex items-center gap-4">
                    {bill.invoice_number && <span>#{bill.invoice_number}</span>}
                    {bill.drive_url && (
                      <a href={bill.drive_url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                        Drive ↗
                      </a>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    {bill.status === 'pending' && <DueDateBadge dueDate={bill.due_date} />}
                    <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                      bill.status === 'paid' ? 'bg-success/20 text-green-800 dark:text-green-400' : 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
                    }`}>
                      {t(`bills.status_${bill.status}`)}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Desktop: tabla */}
          <div className="hidden sm:block bg-card rounded-ios shadow-ios overflow-x-auto">
            <table className="w-full min-w-[600px]">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.year')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.month')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.amount')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.invoice_number')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.status')}</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-secondary uppercase">{t('bills.drive_url')}</th>
                  <th className="w-10"></th>
                </tr>
              </thead>
              <tbody>
                {bills.map(bill => (
                  <tr key={bill.id} className="border-b border-border last:border-b-0 hover:bg-bg/50">
                    <td className="px-4 py-3 text-sm">{bill.year}</td>
                    <td className="px-4 py-3 text-sm">{bill.month === 0 ? t('bills.annual') : t(`months.${bill.month}`, MONTHS[bill.month])}</td>
                    <td className="px-4 py-3 text-sm font-medium">{formatMoney(bill.amount, currency?.symbol)}</td>
                    <td className="px-4 py-3 text-sm">{bill.invoice_number || '-'}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        {bill.status === 'pending' && <DueDateBadge dueDate={bill.due_date} />}
                        <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                          bill.status === 'paid' ? 'bg-success/20 text-green-800 dark:text-green-400' : 'bg-warning/20 text-yellow-800 dark:text-yellow-400'
                        }`}>
                          {t(`bills.status_${bill.status}`)}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {bill.drive_url ? (
                        <a href={bill.drive_url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                          Drive
                        </a>
                      ) : '-'}
                    </td>
                    <td className="px-4 py-3 relative">
                      <CardMenu options={billMenuOptions(bill)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('bills.title')} #${deleteTarget}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
      {payTarget && (
        <PayBillModal
          bill={payTarget}
          onClose={() => setPayTarget(null)}
          onSuccess={load}
        />
      )}
      {historyTarget && (
        <BillHistoryModal
          bill={historyTarget}
          onClose={() => setHistoryTarget(null)}
        />
      )}
      <UploadBillModal
        isOpen={uploadOpen}
        serviceId={Number(serviceId)}
        frequency={service.frequency}
        onClose={() => setUploadOpen(false)}
        onSaved={load}
      />
      {webhookOpen && service && (
        <WebhookModal service={service} onClose={() => setWebhookOpen(false)} />
      )}
      {renewOpen && service && (
        <RenewServiceModal service={service} onSaved={() => { setRenewOpen(false); load() }} onCancel={() => setRenewOpen(false)} />
      )}
      {addPolicyOpen && service && (
        <AddPolicyModal serviceId={service.id} onSaved={handlePolicySaved} onCancel={() => setAddPolicyOpen(false)} />
      )}
      {editPolicy && service && (
        <AddPolicyModal serviceId={service.id} policy={editPolicy} onSaved={handlePolicySaved} onCancel={() => setEditPolicy(null)} />
      )}
      {deletePolicyTarget && (
        <DeleteModal
          title={t('bills.polizas_delete_confirm')}
          subtitle={`${deletePolicyTarget.name}`}
          onConfirm={handleDeletePolicy}
          onCancel={() => setDeletePolicyTarget(null)}
        />
      )}
    </div>
  )
}

