import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import MaintenanceModal from '../components/MaintenanceModal'
import MaintenanceItemModal from '../components/MaintenanceItemModal'
import AutoAttachmentsField from '../components/AutoAttachmentsField'
import type {
  Auto,
  AutoMaintenance,
  AutoMaintenanceItem,
  AutoMaintenanceItemType,
} from '../types'
import { resolveCurrency } from '../utils/currency'

export default function AutoMaintenanceShowPage() {
  const { id, maintenanceId } = useParams()
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const { currencies, loadCurrencies } = useAppStore()
  const [auto, setAuto] = useState<Auto | null>(null)
  const [maintenance, setMaintenance] = useState<AutoMaintenance | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [itemModal, setItemModal] = useState<{ itemType: AutoMaintenanceItemType; item?: AutoMaintenanceItem | null } | null>(null)
  const [deleteItemTarget, setDeleteItemTarget] = useState<AutoMaintenanceItem | null>(null)

  usePageTitle(t('autos.maintenances.detail_title'))

  const loadDetail = useCallback(async () => {
    const [autoData, maintenanceData] = await Promise.all([
      api.autos.get(Number(id)),
      api.autos.getMaintenance(Number(id), Number(maintenanceId)),
    ])
    setAuto(autoData)
    setMaintenance(maintenanceData)
  }, [id, maintenanceId])

  useEffect(() => {
    loadCurrencies()
    loadDetail().finally(() => setLoading(false))
  }, [loadCurrencies, loadDetail])

  const currencySymbol = (currencyId: number) => {
    const resolved = resolveCurrency(currencies, { id: currencyId })
    return resolved.symbol || resolved.code
  }

  const handleDeleteMaintenance = useCallback(async () => {
    await api.autos.deleteMaintenance(Number(id), Number(maintenanceId))
    setDeleteOpen(false)
    navigate(`/autos/${id}?tab=mantenimientos`)
  }, [id, maintenanceId, navigate])

  const handleDeleteItem = useCallback(async () => {
    if (!deleteItemTarget || !maintenance) return
    await api.autos.deleteMaintenanceItem(Number(id), maintenance.id, deleteItemTarget.id)
    setDeleteItemTarget(null)
    loadDetail()
  }, [deleteItemTarget, id, maintenance, loadDetail])

  const handleUploadFiles = async (files: File[]) => {
    if (!maintenance) return
    for (const file of files) {
      await api.autos.uploadMaintenanceAttachment(Number(id), maintenance.id, file)
    }
    await loadDetail()
  }

  const handleDeleteAttachment = async (attachmentId: number) => {
    if (!maintenance) return
    await api.autos.deleteMaintenanceAttachment(Number(id), maintenance.id, attachmentId)
    await loadDetail()
  }

  if (loading) return <LoadingSpinner />
  if (!maintenance || !auto) {
    return (
      <div className="text-center text-text-secondary p-8">
        <p className="font-medium">{t('autos.maintenances.not_found')}</p>
        <p className="text-sm mt-1">{t('autos.maintenances.not_found_hint')}</p>
      </div>
    )
  }

  const symbol = currencySymbol(maintenance.currency_id)
  const renderItems = (itemType: AutoMaintenanceItemType) => {
    const rows = maintenance.items.filter((item) => item.item_type === itemType)
    const isLabor = itemType === 'labor'
    const total = isLabor ? maintenance.labor_cost : maintenance.supplies_cost
    return (
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <div className="flex items-center gap-2 min-w-0">
            <Icon name={isLabor ? 'wrench' : 'recycling'} className="w-5 h-5 text-text-secondary flex-shrink-0" />
            <h3 className="text-lg font-bold truncate">
              {isLabor ? t('autos.maintenances.labor_items') : t('autos.maintenances.supplies_items')}
            </h3>
            <span className="text-sm text-text-secondary">{formatMoney(total, symbol)}</span>
          </div>
          <button
            type="button"
            onClick={() => setItemModal({ itemType })}
            className="inline-flex items-center gap-2 px-3 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors text-sm min-h-[44px]"
          >
            <Icon name="plus" className="w-4 h-4" />
            {isLabor ? t('autos.maintenances.add_labor') : t('autos.maintenances.add_supply')}
          </button>
        </div>

        {rows.length === 0 ? (
          <p className="text-sm text-text-secondary">{t('autos.maintenances.items_empty')}</p>
        ) : (
          <div className="space-y-2">
            {rows.map((item) => (
              <div key={item.id} className="flex items-center gap-3 rounded-ios-sm border border-border bg-bg px-3 py-2">
                <span className="flex-1 min-w-0 text-sm break-words">{item.description}</span>
                <span className="text-sm font-semibold flex-shrink-0">{formatMoney(item.amount, symbol)}</span>
                <button
                  type="button"
                  onClick={() => setItemModal({ itemType, item })}
                  className="p-2 text-text-secondary hover:text-primary transition-colors rounded-full"
                  aria-label={t('app.edit')}
                >
                  <Icon name="edit" className="w-4 h-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setDeleteItemTarget(item)}
                  className="p-2 text-text-secondary hover:text-danger transition-colors rounded-full"
                  aria-label={t('app.delete')}
                >
                  <Icon name="delete" className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  return (
    <div>
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6 mb-6">
        <CardMenu
          options={[
            { label: t('app.edit'), icon: 'edit', onClick: () => setEditOpen(true) },
            { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteOpen(true) },
          ]}
        />
        <div className="pr-10">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <h2 className="text-xl sm:text-2xl font-bold break-words">{maintenance.performed_at}</h2>
            <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-primary/10 text-primary">
              {maintenance.mileage.toLocaleString()} km
            </span>
          </div>
          <p className="text-text-secondary mt-1 break-words">{maintenance.institution_name}</p>
          {maintenance.description && (
            <p className="text-sm text-text-secondary mt-2 break-words">{maintenance.description}</p>
          )}
          <div className="flex flex-wrap gap-x-4 gap-y-1 mt-3 text-sm">
            <span className="text-text-secondary">
              {t('autos.maintenances.labor_cost')}:{' '}
              <strong className="text-text">{formatMoney(maintenance.labor_cost, symbol)}</strong>
            </span>
            <span className="text-text-secondary">
              {t('autos.maintenances.supplies_cost')}:{' '}
              <strong className="text-text">{formatMoney(maintenance.supplies_cost, symbol)}</strong>
            </span>
            <span className="font-semibold">
              {t('autos.maintenances.total')}: {formatMoney(maintenance.total_cost, symbol)}
            </span>
          </div>
        </div>
      </div>

      <div className="space-y-6">
        {renderItems('labor')}
        {renderItems('supplies')}

        <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
          <AutoAttachmentsField
            attachments={maintenance.attachments}
            attachmentUrl={(attachment) =>
              api.autos.maintenanceAttachmentUrl(Number(id), maintenance.id, attachment.id)
            }
            onUpload={handleUploadFiles}
            onDelete={(attachment) => handleDeleteAttachment(attachment.id)}
          />
        </div>
      </div>

      {editOpen && (
        <MaintenanceModal
          autoId={Number(id)}
          maintenance={maintenance}
          onSaved={() => {
            setEditOpen(false)
            loadDetail()
          }}
          onCancel={() => setEditOpen(false)}
        />
      )}

      {itemModal && (
        <MaintenanceItemModal
          autoId={Number(id)}
          maintenanceId={maintenance.id}
          itemType={itemModal.itemType}
          item={itemModal.item}
          onSaved={() => {
            setItemModal(null)
            loadDetail()
          }}
          onCancel={() => setItemModal(null)}
        />
      )}

      {deleteOpen && (
        <DeleteModal
          title={t('autos.maintenances.delete_title')}
          subtitle={t('autos.maintenances.delete_subtitle').replace('{date}', maintenance.performed_at)}
          onConfirm={handleDeleteMaintenance}
          onCancel={() => setDeleteOpen(false)}
        />
      )}

      {deleteItemTarget && (
        <DeleteModal
          title={t('autos.maintenances.delete_item_title')}
          subtitle={t('autos.maintenances.delete_item_subtitle').replace('{description}', deleteItemTarget.description)}
          onConfirm={handleDeleteItem}
          onCancel={() => setDeleteItemTarget(null)}
        />
      )}
    </div>
  )
}
