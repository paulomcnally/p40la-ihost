import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { Icon } from '../components/Icons'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import HomeMaintenanceItemModal from '../components/home/HomeMaintenanceItemModal'
import type { HomeMaintenance, HomeMaintenanceItem, HomeMaintenanceItemType } from '../types/homeMaintenance'
import { resolveCurrency } from '../utils/currency'

type TabKey = 'labor' | 'supplies'

export default function HomeMaintenanceShowPage() {
  const navigate = useNavigate()
  const { id, maintenanceId } = useParams()
  const homeId = Number(id)
  const recordId = Number(maintenanceId)
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { currencies, loadCurrencies } = useAppStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)

  const [homeName, setHomeName] = useState('')
  const [record, setRecord] = useState<HomeMaintenance | null>(null)
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState<TabKey>('labor')
  const [itemModal, setItemModal] = useState<{ itemType: HomeMaintenanceItemType; item?: HomeMaintenanceItem | null } | null>(null)
  const [deleteRecord, setDeleteRecord] = useState(false)
  const [deleteItemTarget, setDeleteItemTarget] = useState<HomeMaintenanceItem | null>(null)

  usePageTitle(t('homeMaintenance.detail_title'))

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [data, home] = await Promise.all([
        api.homeMaintenance.get(homeId, recordId),
        api.homes.get(homeId),
      ])
      setRecord(data)
      setHomeName(home?.name ?? '')
    } catch {
      setRecord(null)
    } finally {
      setLoading(false)
    }
  }, [homeId, recordId])

  useEffect(() => {
    loadCurrencies()
    load()
  }, [load, loadCurrencies])

  const backToHome = () => navigate(`/home/${homeId}?tab=maintenance`)

  const handleDeleteRecord = useCallback(async () => {
    try {
      await api.homeMaintenance.delete(homeId, recordId)
      showToast(t('homeMaintenance.deleted_ok'), 'success')
      setDeleteRecord(false)
      backToHome()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [homeId, recordId, showToast, t])

  const handleDeleteItem = useCallback(async () => {
    if (!deleteItemTarget) return
    try {
      await api.homeMaintenance.deleteItem(homeId, recordId, deleteItemTarget.id)
      showToast(t('homeMaintenance.item_deleted_ok'), 'success')
      setDeleteItemTarget(null)
      load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [deleteItemTarget, homeId, recordId, load, showToast, t])

  if (loading) return <LoadingSpinner />

  if (!record) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <h3 className="text-lg font-semibold mb-2">{t('homeMaintenance.not_found')}</h3>
        <p className="text-sm text-text-secondary mb-4">{t('homeMaintenance.not_found_hint')}</p>
        <button
          type="button"
          onClick={backToHome}
          className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors min-h-[44px]"
        >
          {t('homeMaintenance.back_to_home')}
        </button>
      </div>
    )
  }

  const symbolFor = (code: string) => {
    const resolved = resolveCurrency(currencies, { code })
    return resolved.symbol || resolved.code || code
  }

  const renderItems = (itemType: HomeMaintenanceItemType) => {
    const rows = record.items.filter((item) => item.item_type === itemType)
    const isLabor = itemType === 'labor'
    const typeTotals = record.totals_by_currency
      .map((total) => ({
        code: total.currency_code,
        amount: isLabor ? total.labor_cost : total.supplies_cost,
      }))
      .filter((total) => total.amount > 0)

    return (
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <div className="flex items-center gap-2 min-w-0">
            <Icon name={isLabor ? 'wrench' : 'recycling'} className="w-5 h-5 text-text-secondary flex-shrink-0" />
            <h3 className="text-lg font-bold truncate">
              {isLabor ? t('homeMaintenance.labor') : t('homeMaintenance.supplies')}
            </h3>
            <span className="flex flex-wrap gap-x-3 text-sm text-text-secondary">
              {typeTotals.map((total) => (
                <span key={total.code}>{formatMoney(total.amount, symbolFor(total.code))}</span>
              ))}
            </span>
          </div>
          <button
            type="button"
            onClick={() => setItemModal({ itemType })}
            className="inline-flex items-center gap-2 px-3 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors text-sm min-h-[44px]"
          >
            <Icon name="plus" className="w-4 h-4" />
            {isLabor ? t('homeMaintenance.add_labor') : t('homeMaintenance.add_supply')}
          </button>
        </div>

        {rows.length === 0 ? (
          <p className="text-sm text-text-secondary">{t('homeMaintenance.items_empty')}</p>
        ) : (
          <div className="space-y-2">
            {rows.map((item) => (
              <div
                key={item.id}
                className="relative flex items-center gap-3 rounded-ios-sm border border-border bg-bg pl-3 pr-12 py-2 my-1"
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm break-words">{item.description}</p>
                  <p className="text-xs text-text-secondary break-words">
                    {item.item_type === 'supplies' && (
                      <span className="mr-2">
                        {item.quantity} {t(`homeMaintenance.unit_${item.unit}`)}
                      </span>
                    )}
                    {item.institution_name || ''}
                  </p>
                </div>
                <span className="text-sm font-semibold flex-shrink-0">
                  {formatMoney(item.amount, symbolFor(item.currency_code))}
                </span>
                <CardMenu
                  options={[
                    {
                      label: t('app.edit'),
                      icon: 'edit',
                      onClick: () => setItemModal({ itemType, item }),
                    },
                    {
                      label: t('app.delete'),
                      icon: 'delete',
                      danger: true,
                      onClick: () => setDeleteItemTarget(item),
                    },
                  ]}
                />
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  const laborCount = record.items.filter((item) => item.item_type === 'labor').length
  const suppliesCount = record.items.filter((item) => item.item_type === 'supplies').length

  return (
    <div className="space-y-4">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu
          options={[
            {
              label: t('app.edit'),
              icon: 'edit',
              onClick: () => navigate(`/home/${homeId}/maintenances/${record.id}/edit`),
            },
            {
              label: t('app.delete'),
              icon: 'delete',
              danger: true,
              onClick: () => setDeleteRecord(true),
            },
          ]}
        />
        <div className="pr-10">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-xl font-bold">{record.performed_at}</h2>
            <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-primary/10 text-primary">
              {t(`homeMaintenance.kind_${record.kind}`)}
            </span>
            <span className="text-xs px-2 py-0.5 rounded-full bg-bg text-text-secondary">
              {t(`homeMaintenance.area_${record.area}`)}
            </span>
          </div>
          {homeName && <p className="text-sm text-text-secondary mt-1">{homeName}</p>}
          {record.institution_name && (
            <p className="text-sm text-text-secondary mt-1 break-words">{record.institution_name}</p>
          )}
          {record.description && (
            <p className="text-sm text-text-secondary mt-2 break-words">{record.description}</p>
          )}
          <div className="mt-3 space-y-1 text-sm">
            {record.totals_by_currency.length === 0 ? (
              <p className="text-text-secondary">{t('homeMaintenance.no_currency_totals')}</p>
            ) : (
              record.totals_by_currency.map((total) => (
                <div key={total.currency_id} className="flex flex-wrap gap-x-4 gap-y-1">
                  <span className="text-text-secondary">
                    {t('homeMaintenance.labor_cost')}:{' '}
                    <strong className="text-text">
                      {formatMoney(total.labor_cost, symbolFor(total.currency_code))}
                    </strong>
                  </span>
                  <span className="text-text-secondary">
                    {t('homeMaintenance.supplies_cost')}:{' '}
                    <strong className="text-text">
                      {formatMoney(total.supplies_cost, symbolFor(total.currency_code))}
                    </strong>
                  </span>
                  <span className="font-semibold">
                    {t('homeMaintenance.total')}: {formatMoney(total.total, symbolFor(total.currency_code))}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        {([
          { key: 'labor' as TabKey, label: t('homeMaintenance.labor'), icon: 'wrench', count: laborCount },
          { key: 'supplies' as TabKey, label: t('homeMaintenance.supplies'), icon: 'recycling', count: suppliesCount },
        ]).map((item) => (
          <button
            key={item.key}
            type="button"
            onClick={() => setTab(item.key)}
            aria-pressed={tab === item.key}
            className={`inline-flex items-center justify-center gap-1.5 px-2 sm:px-3 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
              tab === item.key ? 'bg-primary text-white' : 'bg-card text-text-secondary hover:bg-border'
            }`}
          >
            <Icon name={item.icon} className="w-4 h-4 flex-shrink-0" />
            <span className="truncate">{item.label}</span>
            {item.count > 0 && (
              <span
                className={`inline-flex items-center justify-center min-w-5 px-1.5 py-0.5 rounded-full text-[10px] ${
                  tab === item.key ? 'bg-white/20 text-white' : 'bg-bg text-text-secondary'
                }`}
              >
                {item.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {renderItems(tab)}

      {itemModal && (
        <HomeMaintenanceItemModal
          homeId={homeId}
          maintenanceId={record.id}
          itemType={itemModal.itemType}
          item={itemModal.item}
          onSaved={() => {
            setItemModal(null)
            showToast(t('homeMaintenance.item_saved_ok'), 'success')
            load()
          }}
          onCancel={() => setItemModal(null)}
        />
      )}

      {deleteRecord && (
        <DeleteModal
          title={t('homeMaintenance.delete_title')}
          subtitle={t('homeMaintenance.delete_subtitle').replace('{date}', record.performed_at)}
          onConfirm={handleDeleteRecord}
          onCancel={() => setDeleteRecord(false)}
        />
      )}

      {deleteItemTarget && (
        <DeleteModal
          title={t('homeMaintenance.delete_item_title')}
          subtitle={t('homeMaintenance.delete_item_subtitle').replace(
            '{description}',
            deleteItemTarget.description,
          )}
          onConfirm={handleDeleteItem}
          onCancel={() => setDeleteItemTarget(null)}
        />
      )}
    </div>
  )
}
