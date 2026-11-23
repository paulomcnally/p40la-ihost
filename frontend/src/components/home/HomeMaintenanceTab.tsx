import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../../api'
import { useAppStore } from '../../stores/appStore'
import { useCurrencyFormatStore } from '../../stores/currencyFormatStore'
import { useI18nStore } from '../../stores/i18nStore'
import { useToast } from '../../components/Toast'
import { Icon } from '../../components/Icons'
import CardMenu from '../../components/CardMenu'
import DeleteModal from '../../components/DeleteModal'
import LoadingSpinner from '../../components/LoadingSpinner'
import Select from '../../components/Select'
import type { HomeMaintenance, HomeMaintenanceKind } from '../../types/homeMaintenance'
import { resolveCurrency } from '../../utils/currency'

const KINDS: HomeMaintenanceKind[] = ['corrective', 'preventive']

export default function HomeMaintenanceTab({ homeId }: { homeId: number }) {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { currencies, loadCurrencies } = useAppStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)

  const [items, setItems] = useState<HomeMaintenance[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [year, setYear] = useState('')
  const [kind, setKind] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<HomeMaintenance | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await api.homeMaintenance.list(homeId)
      setItems(data?.items ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.generic'))
    } finally {
      setLoading(false)
    }
  }, [homeId, t])

  useEffect(() => {
    loadCurrencies()
    load()
  }, [load, loadCurrencies])

  const years = useMemo(
    () => Array.from(new Set(items.map((item) => item.performed_at.slice(0, 4)))).sort().reverse(),
    [items],
  )

  const visible = useMemo(
    () =>
      items.filter(
        (item) =>
          (!year || item.performed_at.startsWith(year)) && (!kind || item.kind === kind),
      ),
    [items, year, kind],
  )

  const totals = useMemo(() => {
    const grouped = new Map<string, number>()
    for (const item of visible) {
      const key = item.currency_code || String(item.currency_id)
      grouped.set(key, (grouped.get(key) ?? 0) + item.total_cost)
    }
    return Array.from(grouped.entries()).map(([code, total]) => ({ code, total }))
  }, [visible])

  const symbolFor = (code: string) => {
    const resolved = resolveCurrency(currencies, { code })
    return resolved.symbol || resolved.code || code
  }

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await api.homeMaintenance.delete(homeId, deleteTarget.id)
      showToast(t('homeMaintenance.deleted_ok'), 'success')
      setDeleteTarget(null)
      load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [deleteTarget, homeId, load, showToast, t])

  const yearOptions = [
    { value: '', label: t('homeMaintenance.filter_all_years') },
    ...years.map((value) => ({ value, label: value })),
  ]
  const kindOptions = [
    { value: '', label: t('homeMaintenance.filter_all_kinds') },
    ...KINDS.map((value) => ({ value, label: t(`homeMaintenance.kind_${value}`) })),
  ]

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center gap-3 sm:justify-between">
        <div className="grid grid-cols-2 gap-2 sm:flex sm:items-center">
          <Select
            options={yearOptions}
            value={year}
            onChange={(value) => setYear(String(value))}
            placeholder={t('homeMaintenance.filter_year')}
            emptyLabel={t('app.no_results')}
          />
          <Select
            options={kindOptions}
            value={kind}
            onChange={(value) => setKind(String(value))}
            placeholder={t('homeMaintenance.filter_kind')}
            emptyLabel={t('app.no_results')}
          />
        </div>
        <button
          type="button"
          onClick={() => navigate(`/home/${homeId}/maintenances/new`)}
          className="inline-flex items-center justify-center gap-2 px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors min-h-[44px]"
        >
          <Icon name="plus" className="w-4 h-4" />
          {t('homeMaintenance.new')}
        </button>
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : error ? (
        <div className="bg-card rounded-ios shadow-ios p-6 text-center">
          <p className="text-sm text-danger">{error}</p>
          <button type="button" onClick={load} className="mt-3 text-sm text-primary hover:underline">
            {t('homeMaintenance.retry')}
          </button>
        </div>
      ) : items.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center">
          <div className="w-12 h-12 rounded-ios bg-primary/10 text-primary flex items-center justify-center mx-auto">
            <Icon name="wrench" className="w-6 h-6" />
          </div>
          <h3 className="text-lg font-semibold mt-3">{t('homeMaintenance.empty_title')}</h3>
          <p className="text-sm text-text-secondary mt-1">{t('homeMaintenance.empty_subtitle')}</p>
          <button
            type="button"
            onClick={() => navigate(`/home/${homeId}/maintenances/new`)}
            className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors min-h-[44px]"
          >
            <Icon name="plus" className="w-4 h-4" />
            {t('homeMaintenance.empty_action')}
          </button>
        </div>
      ) : (
        <>
          <div className="bg-card rounded-ios shadow-ios p-4">
            <p className="text-xs uppercase tracking-wide text-text-secondary">
              {t('homeMaintenance.total_history')}
            </p>
            <div className="flex flex-wrap gap-x-6 gap-y-1 mt-1">
              {totals.length === 0 ? (
                <span className="text-sm text-text-secondary">{t('homeMaintenance.no_currency_totals')}</span>
              ) : (
                totals.map((total) => (
                  <span key={total.code} className="text-lg font-bold">
                    {formatMoney(total.total, symbolFor(total.code))}
                  </span>
                ))
              )}
            </div>
            <p className="text-xs text-text-secondary mt-1">
              {visible.length} {t('homeMaintenance.records')}
            </p>
          </div>

          {visible.length === 0 ? (
            <div className="bg-card rounded-ios shadow-ios p-6 text-center text-sm text-text-secondary">
              {t('app.no_results')}
            </div>
          ) : (
            <div className="space-y-3">
              {visible.map((item) => (
                <div
                  key={item.id}
                  className="relative bg-card rounded-ios shadow-ios p-4 hover:shadow-ios-lg transition-shadow"
                >
                  <CardMenu
                    options={[
                      {
                        label: t('app.edit'),
                        icon: 'edit',
                        onClick: () => navigate(`/home/${homeId}/maintenances/${item.id}/edit`),
                      },
                      {
                        label: t('app.delete'),
                        icon: 'delete',
                        danger: true,
                        onClick: () => setDeleteTarget(item),
                      },
                    ]}
                  />
                  <button
                    type="button"
                    onClick={() => navigate(`/home/${homeId}/maintenances/${item.id}`)}
                    className="block w-full text-left pr-10"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-semibold">{item.performed_at}</span>
                      <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                        {t(`homeMaintenance.kind_${item.kind}`)}
                      </span>
                      <span className="text-xs px-2 py-0.5 rounded-full bg-bg text-text-secondary">
                        {t(`homeMaintenance.area_${item.area}`)}
                      </span>
                    </div>
                    {item.description && (
                      <p className="text-sm text-text-secondary mt-1 break-words">{item.description}</p>
                    )}
                    {item.institution_name && (
                      <p className="text-xs text-text-secondary mt-1 break-words">{item.institution_name}</p>
                    )}
                    <div className="flex flex-wrap gap-x-4 gap-y-1 mt-2 text-sm">
                      <span className="text-text-secondary">
                        {t('homeMaintenance.labor_cost')}:{' '}
                        <strong className="text-text">
                          {formatMoney(item.labor_cost, symbolFor(item.currency_code))}
                        </strong>
                      </span>
                      <span className="text-text-secondary">
                        {t('homeMaintenance.supplies_cost')}:{' '}
                        <strong className="text-text">
                          {formatMoney(item.supplies_cost, symbolFor(item.currency_code))}
                        </strong>
                      </span>
                      <span className="font-semibold">
                        {t('homeMaintenance.total')}:{' '}
                        {formatMoney(item.total_cost, symbolFor(item.currency_code))}
                      </span>
                    </div>
                  </button>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('homeMaintenance.delete_title')}
          subtitle={t('homeMaintenance.delete_subtitle').replace('{date}', deleteTarget.performed_at)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
