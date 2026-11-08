import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { Icon } from '../components/Icons'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import AddInsuranceModal from '../components/AddInsuranceModal'
import CardMenu from '../components/CardMenu'
import MaintenanceModal from '../components/MaintenanceModal'
import InspectionModal from '../components/InspectionModal'
import InspectionStatusBadge from '../components/InspectionStatusBadge'
import AutoAttachmentsModal from '../components/AutoAttachmentsModal'
import type {
  Auto,
  AutoService,
  AutoMaintenance,
  AutoMaintenanceList,
  AutoInspection,
  AutoInspectionList,
} from '../types'
import { resolveCurrency } from '../utils/currency'
import { autoSubtitle } from '../utils/auto'

type TabKey = 'seguros' | 'mantenimientos' | 'inspecciones'

export default function AutoShowPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const [searchParams, setSearchParams] = useSearchParams()
  const tabParam = searchParams.get('tab')
  const tab: TabKey =
    tabParam === 'mantenimientos' || tabParam === 'inspecciones' ? tabParam : 'seguros'
  const setTab = (key: TabKey) => {
    const next = new URLSearchParams(searchParams)
    next.set('tab', key)
    setSearchParams(next, { replace: false })
  }
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const { currencies, loadCurrencies } = useAppStore()
  const currencyLabel = (currencyId: number) => {
    const resolved = resolveCurrency(currencies, { id: currencyId })
    return resolved.symbol || resolved.code
  }
  const [auto, setAuto] = useState<Auto | null>(null)
  const [insurance, setInsurance] = useState<AutoService[]>([])
  const [maintenances, setMaintenances] = useState<AutoMaintenanceList | null>(null)
  const [inspections, setInspections] = useState<AutoInspectionList | null>(null)
  const [loading, setLoading] = useState(true)
  const [showAddModal, setShowAddModal] = useState(false)
  const [editInsurance, setEditInsurance] = useState<AutoService | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ serviceId: number; serviceName: string } | null>(null)
  const [showMaintenanceModal, setShowMaintenanceModal] = useState(false)
  const [editMaintenance, setEditMaintenance] = useState<AutoMaintenance | null>(null)
  const [deleteMaintenanceTarget, setDeleteMaintenanceTarget] = useState<AutoMaintenance | null>(null)
  const [showInspectionModal, setShowInspectionModal] = useState(false)
  const [editInspection, setEditInspection] = useState<AutoInspection | null>(null)
  const [deleteInspectionTarget, setDeleteInspectionTarget] = useState<AutoInspection | null>(null)
  const [pdfMaintenanceTarget, setPdfMaintenanceTarget] = useState<AutoMaintenance | null>(null)
  const [pdfInspectionTarget, setPdfInspectionTarget] = useState<AutoInspection | null>(null)

  const loadAuto = useCallback(async () => {
    const data = await api.autos.get(Number(id))
    setAuto(data)
  }, [id])

  const loadInsurance = useCallback(async () => {
    const data = await api.autos.listServices(Number(id))
    setInsurance(data || [])
  }, [id])

  const loadMaintenances = useCallback(async () => {
    const data = await api.autos.listMaintenances(Number(id))
    setMaintenances(data)
  }, [id])

  const loadInspections = useCallback(async () => {
    const data = await api.autos.listInspections(Number(id))
    setInspections(data)
  }, [id])

  useEffect(() => {
    loadCurrencies()
    Promise.all([loadAuto(), loadInsurance(), loadMaintenances(), loadInspections()])
      .finally(() => setLoading(false))
  }, [])

  usePageTitle(auto ? `${auto.brand} ${auto.model}` : null)

  const handleAddInsurance = useCallback(async () => {
    await loadInsurance()
    setShowAddModal(false)
  }, [loadInsurance])

  const handleRemoveInsurance = useCallback(async () => {
    if (!deleteTarget) return
    await api.autos.removeService(Number(id), deleteTarget.serviceId)
    setDeleteTarget(null)
    loadInsurance()
  }, [deleteTarget, id, loadInsurance])

  const handleMaintenanceSaved = useCallback(() => {
    setShowMaintenanceModal(false)
    setEditMaintenance(null)
    loadMaintenances()
  }, [loadMaintenances])

  const handleInspectionSaved = useCallback(() => {
    setShowInspectionModal(false)
    setEditInspection(null)
    loadInspections()
  }, [loadInspections])

  const handleRemoveMaintenance = useCallback(async () => {
    if (!deleteMaintenanceTarget) return
    await api.autos.deleteMaintenance(Number(id), deleteMaintenanceTarget.id)
    setDeleteMaintenanceTarget(null)
    loadMaintenances()
  }, [deleteMaintenanceTarget, id, loadMaintenances])

  const handleRemoveInspection = useCallback(async () => {
    if (!deleteInspectionTarget) return
    await api.autos.deleteInspection(Number(id), deleteInspectionTarget.id)
    setDeleteInspectionTarget(null)
    loadInspections()
  }, [deleteInspectionTarget, id, loadInspections])

  if (loading) return <LoadingSpinner />
  if (!auto) return <div className="text-center text-text-secondary p-8">Auto no encontrado</div>

  const groupedByInstitution = insurance.reduce((acc, item) => {
    const key = item.institution_name || 'Sin institución'
    if (!acc[key]) acc[key] = []
    acc[key].push(item)
    return acc
  }, {} as Record<string, AutoService[]>)

  return (
    <div>
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6 mb-6">
        <CardMenu
          options={[
            { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/autos/edit/${auto.id}`) },
          ]}
        />
        <div className="flex items-start gap-4 pr-8">
          <div className="w-14 h-14 rounded-ios bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
            <Icon name={auto.icon} className="w-8 h-8" />
          </div>
          <div className="flex-1 min-w-0">
            <h2 className="text-xl sm:text-2xl font-bold break-words">{auto.brand} {auto.model}</h2>
            <p className="text-text-secondary mt-1 break-words">{autoSubtitle(t, auto)}</p>
            {(auto.motor || auto.chasis || auto.vin) && (
              <div className="mt-3 space-y-1 text-sm text-text-secondary">
                {auto.motor && (
                  <p className="break-words"><span className="font-medium">{t('autos.motor')}:</span> {auto.motor}</p>
                )}
                {auto.chasis && (
                  <p className="break-words"><span className="font-medium">{t('autos.chasis')}:</span> {auto.chasis}</p>
                )}
                {auto.vin && (
                  <p className="break-words"><span className="font-medium">{t('autos.vin')}:</span> {auto.vin}</p>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-2 mb-4">
        {([
          { key: 'seguros' as TabKey, label: t('autos.tabs.insurance'), icon: 'insurance', count: insurance.length },
          { key: 'mantenimientos' as TabKey, label: t('autos.tabs.maintenances'), icon: 'wrench', count: maintenances?.items.length ?? 0 },
          { key: 'inspecciones' as TabKey, label: t('autos.tabs.inspections'), icon: 'calendar', count: inspections?.items.length ?? 0 },
        ]).map((item) => (
          <button
            key={item.key}
            onClick={() => setTab(item.key)}
            aria-pressed={tab === item.key}
            className={`inline-flex items-center justify-center gap-1.5 px-2 sm:px-4 py-2 rounded-ios-sm text-[11px] sm:text-sm font-medium transition-colors min-h-[44px] ${
              tab === item.key ? 'bg-primary text-white' : 'bg-card text-text-secondary hover:bg-border'
            }`}
          >
            <Icon name={item.icon} className="w-4 h-4 flex-shrink-0" />
            <span className="truncate">{item.label}</span>
            <span
              className={`hidden sm:inline-flex items-center justify-center min-w-5 px-1.5 py-0.5 rounded-full text-[10px] ${
                tab === item.key ? 'bg-white/20 text-white' : 'bg-bg text-text-secondary'
              }`}
            >
              {item.count}
            </span>
            {item.key === 'inspecciones' && inspections && inspections.schedule.status !== 'ok' && (
              <span
                className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  inspections.schedule.status === 'overdue' ? 'bg-danger' : 'bg-warning'
                }`}
              />
            )}
          </button>
        ))}
      </div>

      {tab === 'seguros' && (
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex justify-end mb-4">
          <button
            onClick={() => setShowAddModal(true)}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors flex items-center gap-2 text-sm min-h-[44px]"
          >
            <Icon name="plus" className="w-4 h-4" />
            Agregar Seguro
          </button>
        </div>

        {insurance.length === 0 ? (
          <div className="text-center py-8">
            <div className="w-12 h-12 mx-auto mb-3 text-text-secondary opacity-50">
              <Icon name="insurance" className="w-full h-full" />
            </div>
            <p className="text-text-secondary">No hay seguros asociados</p>
          </div>
        ) : (
          <div className="space-y-4">
            {Object.entries(groupedByInstitution).map(([instName, items]) => (
              <div key={instName}>
                <div className="flex items-center gap-2 mb-2 min-w-0">
                  <Icon name="building" className="w-4 h-4 text-text-secondary flex-shrink-0" />
                  <span className="text-sm font-semibold text-text-secondary truncate">{instName}</span>
                  <span className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full dark:bg-[#2c2c2e] dark:text-gray-400 flex-shrink-0">{items.length}</span>
                </div>
                <div className="space-y-2">
                  {items.map((item) => {
                    const isExpired = item.end_date && new Date(item.end_date) < new Date()
                    return (
<div
                        key={item.id}
                        className={`relative rounded-ios-sm border border-border p-3 ${
                          !item.active ? 'bg-gray-50 opacity-60 dark:bg-[#2c2c2e]' : 'bg-bg'
                        }`}
                      >
                        <div className="flex items-center gap-3">
                          <div className={`w-9 h-9 rounded-ios flex items-center justify-center flex-shrink-0 ${
                            item.active ? 'bg-primary/10 text-primary' : 'bg-gray-200 text-gray-400 dark:bg-[#2c2c2e] dark:text-gray-500'
                          }`}>
                            <Icon name={item.icon_key || 'other'} className="w-5 h-5" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="font-medium text-sm truncate">{item.service_name}</p>
                            <div className="flex flex-wrap items-center gap-x-2 gap-y-1 mt-0.5">
                              <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                                item.coverage_type === 'full_cover' ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
                              }`}>
                                {item.coverage_type === 'full_cover' ? 'Full Cover' : 'Daños a terceros'}
                              </span>
                              <span className="text-xs text-text-secondary break-all">Póliza: {item.policy_number}</span>
                              <span className="text-xs text-text-secondary break-all">Aseguradora: {item.insurer_number}</span>
                              {item.certificate && (
                                <span className="text-xs text-text-secondary break-all">Certificado: {item.certificate}</span>
                              )}
                              {!item.active && (
                                <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400">
                                  Inactivo
                                </span>
                              )}
                              {isExpired && (
                                <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300">
                                  Vencido
                                </span>
                              )}
                            </div>
                          </div>
                          <div className="text-right shrink-0 min-w-0 max-w-[45%] pr-8">
                            <p className="text-sm font-semibold break-words">{formatMoney(item.suggested_amount, currencyLabel(item.currency_id))}</p>
                            <p className="text-xs text-text-secondary">{item.frequency === 'monthly' ? 'Mensual' : 'Anual'}</p>
                            {item.start_date && (
                              <p className="text-xs text-text-secondary mt-0.5 break-words">
                                {item.start_date}{item.end_date ? ` → ${item.end_date}` : ''}
                              </p>
                            )}
                          </div>
                          <CardMenu
                            options={[
                              { label: 'Editar', icon: 'edit', onClick: () => setEditInsurance(item) },
                              { label: 'Eliminar', icon: 'delete', danger: true, onClick: () => setDeleteTarget({ serviceId: item.service_id, serviceName: item.service_name }) },
                            ]}
                          />
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      )}

      {tab === 'mantenimientos' && (
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex justify-end mb-4">
          <button
            onClick={() => setShowMaintenanceModal(true)}
            className="px-3 sm:px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors flex items-center gap-2 text-sm min-h-[44px] flex-shrink-0"
          >
            <Icon name="plus" className="w-4 h-4" />
            <span className="hidden sm:inline">{t('autos.maintenances.add')}</span>
            <span className="sm:hidden">{t('app.add')}</span>
          </button>
        </div>

        {maintenances && (maintenances.latest_mileage > 0 || maintenances.totals_by_currency.length > 0) && (
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 mb-4 text-sm text-text-secondary">
            <span>
              {t('autos.maintenances.latest_mileage')}:{' '}
              <strong className="text-text">{maintenances.latest_mileage.toLocaleString()} km</strong>
            </span>
            {maintenances.totals_by_currency.map((total) => (
              <span key={total.currency_id}>
                {t('autos.maintenances.total')}:{' '}
                <strong className="text-text">{formatMoney(total.total, currencyLabel(total.currency_id))}</strong>
              </span>
            ))}
          </div>
        )}

        {(maintenances?.items.length ?? 0) === 0 ? (
          <div className="text-center py-8">
            <div className="w-12 h-12 mx-auto mb-3 text-text-secondary opacity-50">
              <Icon name="wrench" className="w-full h-full" />
            </div>
            <p className="text-text-secondary">{t('autos.maintenances.empty')}</p>
            <p className="text-sm text-text-secondary mt-1">{t('autos.maintenances.empty_hint')}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {maintenances?.items.map((item) => (
              <div key={item.id} className="relative rounded-ios-sm border border-border p-3 pr-12 bg-bg">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                  <span className="text-sm font-semibold">{item.performed_at}</span>
                  <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                    {item.mileage.toLocaleString()} km
                  </span>
                  <span className="text-sm text-text-secondary break-words">{item.workshop_name}</span>
                </div>
                {item.description && (
                  <p className="text-sm text-text-secondary mt-1 break-words">{item.description}</p>
                )}
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 mt-2 text-xs text-text-secondary">
                  <span>{t('autos.maintenances.labor_cost')}: {formatMoney(item.labor_cost, currencyLabel(item.currency_id))}</span>
                  <span>{t('autos.maintenances.supplies_cost')}: {formatMoney(item.supplies_cost, currencyLabel(item.currency_id))}</span>
                  <span className="font-semibold text-text">
                    {t('autos.maintenances.total')}: {formatMoney(item.total_cost, currencyLabel(item.currency_id))}
                  </span>
                </div>
                {item.attachments.length > 0 && (
                  <div className="flex flex-wrap gap-2 mt-2">
                    {item.attachments.map((attachment) => (
                      <a
                        key={attachment.id}
                        href={api.autos.maintenanceAttachmentUrl(auto.id, item.id, attachment.id)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-card border border-border text-xs hover:border-primary/50 transition-colors max-w-full"
                      >
                        <Icon name="pdf" className="w-3.5 h-3.5 text-danger flex-shrink-0" />
                        <span className="truncate max-w-[12rem]">{attachment.file_name}</span>
                      </a>
                    ))}
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => setPdfMaintenanceTarget(item)}
                  className="mt-2 inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-full border border-border bg-card text-xs font-medium text-text-secondary hover:border-primary/50 hover:text-primary transition-colors min-h-[36px]"
                >
                  <Icon name="pdf" className="w-3.5 h-3.5 text-danger flex-shrink-0" />
                  {t('autos.attachments.add_pdf')}
                  {item.attachments.length > 0 && (
                    <span className="inline-flex items-center justify-center min-w-4 px-1 rounded-full bg-bg text-[10px]">
                      {item.attachments.length}
                    </span>
                  )}
                </button>
                <CardMenu
                  options={[
                    { label: t('app.edit'), icon: 'edit', onClick: () => setEditMaintenance(item) },
                    { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteMaintenanceTarget(item) },
                  ]}
                />
              </div>
            ))}
          </div>
        )}
      </div>
      )}

      {tab === 'inspecciones' && (
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex justify-end mb-4">
          <button
            onClick={() => setShowInspectionModal(true)}
            className="px-3 sm:px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors flex items-center gap-2 text-sm min-h-[44px] flex-shrink-0"
          >
            <Icon name="plus" className="w-4 h-4" />
            <span className="hidden sm:inline">{t('autos.inspections.add')}</span>
            <span className="sm:hidden">{t('app.add')}</span>
          </button>
        </div>

        {inspections && (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-ios-sm border border-border bg-bg p-3 mb-4">
            <InspectionStatusBadge status={inspections.schedule.status} />
            <span className="text-sm">
              {t('autos.inspections.next_due')}: <strong>{inspections.schedule.next_due_date}</strong>
            </span>
            <span className="text-xs text-text-secondary">
              {t('autos.inspections.first_due_hint').replace('{year}', String(inspections.schedule.first_due_year))}
            </span>
          </div>
        )}

        {(inspections?.items.length ?? 0) === 0 ? (
          <div className="text-center py-8">
            <div className="w-12 h-12 mx-auto mb-3 text-text-secondary opacity-50">
              <Icon name="calendar" className="w-full h-full" />
            </div>
            <p className="text-text-secondary">{t('autos.inspections.empty')}</p>
            <p className="text-sm text-text-secondary mt-1">{t('autos.inspections.empty_hint')}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {inspections?.items.map((item) => (
              <div key={item.id} className="relative rounded-ios-sm border border-border p-3 pr-12 bg-bg">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                  <span className="text-sm font-semibold">{t('autos.inspections.cycle_year')} {item.inspection_year}</span>
                  <span className="text-xs text-text-secondary">{item.performed_at}</span>
                  <span
                    className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                      item.result === 'approved'
                        ? 'bg-success/10 text-success'
                        : 'bg-danger/10 text-danger'
                    }`}
                  >
                    {item.result === 'approved'
                      ? t('autos.inspections.result_approved')
                      : t('autos.inspections.result_rejected')}
                  </span>
                </div>
                <p className="text-sm text-text-secondary mt-1 break-words">{item.institution_name}</p>
                {item.notes && <p className="text-sm text-text-secondary mt-1 break-words">{item.notes}</p>}
                {item.attachments.length > 0 && (
                  <div className="flex flex-wrap gap-2 mt-2">
                    {item.attachments.map((attachment) => (
                      <a
                        key={attachment.id}
                        href={api.autos.inspectionAttachmentUrl(auto.id, item.id, attachment.id)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-card border border-border text-xs hover:border-primary/50 transition-colors max-w-full"
                      >
                        <Icon name="pdf" className="w-3.5 h-3.5 text-danger flex-shrink-0" />
                        <span className="truncate max-w-[12rem]">{attachment.file_name}</span>
                      </a>
                    ))}
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => setPdfInspectionTarget(item)}
                  className="mt-2 inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-full border border-border bg-card text-xs font-medium text-text-secondary hover:border-primary/50 hover:text-primary transition-colors min-h-[36px]"
                >
                  <Icon name="pdf" className="w-3.5 h-3.5 text-danger flex-shrink-0" />
                  {t('autos.attachments.add_pdf')}
                  {item.attachments.length > 0 && (
                    <span className="inline-flex items-center justify-center min-w-4 px-1 rounded-full bg-bg text-[10px]">
                      {item.attachments.length}
                    </span>
                  )}
                </button>
                <CardMenu
                  options={[
                    { label: t('app.edit'), icon: 'edit', onClick: () => setEditInspection(item) },
                    { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteInspectionTarget(item) },
                  ]}
                />
              </div>
            ))}
          </div>
        )}
      </div>
      )}

      {showAddModal && (
        <AddInsuranceModal
          autoId={Number(id)}
          onAdd={handleAddInsurance}
          onCancel={() => setShowAddModal(false)}
        />
      )}
      {editInsurance && (
        <AddInsuranceModal
          autoId={Number(id)}
          insurance={editInsurance}
          onAdd={() => {
            setEditInsurance(null)
            loadInsurance()
          }}
          onCancel={() => setEditInsurance(null)}
        />
      )}
      {deleteTarget && (
        <DeleteModal
          title="Eliminar seguro"
          subtitle={`¿Eliminar "${deleteTarget.serviceName}" de este auto?`}
          onConfirm={handleRemoveInsurance}
          onCancel={() => setDeleteTarget(null)}
        />
      )}

      {(showMaintenanceModal || editMaintenance) && (
        <MaintenanceModal
          autoId={auto.id}
          maintenance={editMaintenance}
          onSaved={handleMaintenanceSaved}
          onCancel={() => {
            setShowMaintenanceModal(false)
            setEditMaintenance(null)
          }}
        />
      )}
      {deleteMaintenanceTarget && (
        <DeleteModal
          title={t('autos.maintenances.delete_title')}
          subtitle={t('autos.maintenances.delete_subtitle').replace('{date}', deleteMaintenanceTarget.performed_at)}
          onConfirm={handleRemoveMaintenance}
          onCancel={() => setDeleteMaintenanceTarget(null)}
        />
      )}

      {(showInspectionModal || editInspection) && (
        <InspectionModal
          autoId={auto.id}
          autoYear={auto.year}
          inspection={editInspection}
          nextDueYear={inspections?.schedule.next_due_year}
          onSaved={handleInspectionSaved}
          onCancel={() => {
            setShowInspectionModal(false)
            setEditInspection(null)
          }}
        />
      )}
      {deleteInspectionTarget && (
        <DeleteModal
          title={t('autos.inspections.delete_title')}
          subtitle={t('autos.inspections.delete_subtitle').replace('{year}', String(deleteInspectionTarget.inspection_year))}
          onConfirm={handleRemoveInspection}
          onCancel={() => setDeleteInspectionTarget(null)}
        />
      )}

      {pdfMaintenanceTarget && (
        <AutoAttachmentsModal
          title={t('autos.maintenances.title')}
          attachments={pdfMaintenanceTarget.attachments}
          attachmentUrl={(attachment) =>
            api.autos.maintenanceAttachmentUrl(auto.id, pdfMaintenanceTarget.id, attachment.id)
          }
          upload={(file) => api.autos.uploadMaintenanceAttachment(auto.id, pdfMaintenanceTarget.id, file)}
          remove={(attachment) => api.autos.deleteMaintenanceAttachment(auto.id, pdfMaintenanceTarget.id, attachment.id)}
          onChanged={loadMaintenances}
          onClose={() => {
            setPdfMaintenanceTarget(null)
            loadMaintenances()
          }}
        />
      )}

      {pdfInspectionTarget && (
        <AutoAttachmentsModal
          title={t('autos.inspections.title')}
          attachments={pdfInspectionTarget.attachments}
          attachmentUrl={(attachment) =>
            api.autos.inspectionAttachmentUrl(auto.id, pdfInspectionTarget.id, attachment.id)
          }
          upload={(file) => api.autos.uploadInspectionAttachment(auto.id, pdfInspectionTarget.id, file)}
          remove={(attachment) => api.autos.deleteInspectionAttachment(auto.id, pdfInspectionTarget.id, attachment.id)}
          onChanged={loadInspections}
          onClose={() => {
            setPdfInspectionTarget(null)
            loadInspections()
          }}
        />
      )}
    </div>
  )
}
