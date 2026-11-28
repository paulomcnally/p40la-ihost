import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../../api'
import { useI18nStore } from '../../stores/i18nStore'
import { useToast } from '../../components/Toast'
import { Icon } from '../../components/Icons'
import CardMenu from '../../components/CardMenu'
import DeleteModal from '../../components/DeleteModal'
import LoadingSpinner from '../../components/LoadingSpinner'
import Select from '../../components/Select'
import type {
  HomeWorkRequest,
  HomeWorkRequestCategory,
  HomeWorkRequestStatus,
} from '../../types/homeWorkRequest'

const STATUSES: HomeWorkRequestStatus[] = ['reported', 'converted', 'dismissed']
const CATEGORIES: HomeWorkRequestCategory[] = ['repair', 'improvement', 'maintenance', 'other']

export default function HomeWorkRequestTab({ homeId }: { homeId: number }) {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [items, setItems] = useState<HomeWorkRequest[]>([])
  const [pendingCount, setPendingCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [status, setStatus] = useState<HomeWorkRequestStatus | ''>('')
  const [category, setCategory] = useState<HomeWorkRequestCategory | ''>('')
  const [deleteTarget, setDeleteTarget] = useState<HomeWorkRequest | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await api.homeWorkRequest.list(homeId, { status, category })
      setItems(data?.items ?? [])
      setPendingCount(data?.pending_count ?? 0)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.generic'))
    } finally {
      setLoading(false)
    }
  }, [homeId, status, category, t])

  useEffect(() => {
    load()
  }, [load])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await api.homeWorkRequest.delete(homeId, deleteTarget.id)
      showToast(t('homeWorkRequest.deleted_ok'), 'success')
      setDeleteTarget(null)
      load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }, [deleteTarget, homeId, load, showToast, t])

  const statusOptions = [
    { value: '', label: t('homeWorkRequest.filter_all_statuses') },
    ...STATUSES.map((value) => ({ value, label: t(`homeWorkRequest.status_${value}`) })),
  ]
  const categoryOptions = [
    { value: '', label: t('homeWorkRequest.filter_all_categories') },
    ...CATEGORIES.map((value) => ({ value, label: t(`homeWorkRequest.category_${value}`) })),
  ]
  const hasFilters = status !== '' || category !== ''
  const newURL = `/home/${homeId}/workrequests/new`

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center gap-3 sm:justify-between">
        <div className="grid grid-cols-2 gap-2 sm:flex sm:items-center">
          <Select
            options={statusOptions}
            value={status}
            onChange={(value) => setStatus(String(value) as HomeWorkRequestStatus | '')}
            placeholder={t('homeWorkRequest.filter_status')}
            emptyLabel={t('app.no_results')}
          />
          <Select
            options={categoryOptions}
            value={category}
            onChange={(value) => setCategory(String(value) as HomeWorkRequestCategory | '')}
            placeholder={t('homeWorkRequest.filter_category')}
            emptyLabel={t('app.no_results')}
          />
        </div>
        <button
          type="button"
          onClick={() => navigate(newURL)}
          className="inline-flex items-center justify-center gap-2 px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors min-h-[44px]"
        >
          <Icon name="plus" className="w-4 h-4" />
          {t('homeWorkRequest.new')}
        </button>
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : error ? (
        <div className="bg-card rounded-ios shadow-ios p-6 text-center">
          <p className="text-sm text-danger">{error}</p>
          <button type="button" onClick={load} className="mt-3 text-sm text-primary hover:underline">
            {t('homeWorkRequest.retry')}
          </button>
        </div>
      ) : items.length === 0 && !hasFilters ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center">
          <div className="w-12 h-12 rounded-ios bg-primary/10 text-primary flex items-center justify-center mx-auto">
            <Icon name="warning" className="w-6 h-6" />
          </div>
          <h3 className="text-lg font-semibold mt-3">{t('homeWorkRequest.empty_title')}</h3>
          <p className="text-sm text-text-secondary mt-1">{t('homeWorkRequest.empty_subtitle')}</p>
          <button
            type="button"
            onClick={() => navigate(newURL)}
            className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover transition-colors min-h-[44px]"
          >
            <Icon name="plus" className="w-4 h-4" />
            {t('homeWorkRequest.empty_action')}
          </button>
        </div>
      ) : (
        <>
          <div className="bg-card rounded-ios shadow-ios p-4">
            <p className="text-xs uppercase tracking-wide text-text-secondary">
              {t('homeWorkRequest.pending_label')}
            </p>
            <p className="text-2xl font-bold mt-1">{pendingCount}</p>
            <p className="text-xs text-text-secondary mt-1">
              {t('homeWorkRequest.pending_hint')}
            </p>
          </div>

          {items.length === 0 ? (
            <div className="bg-card rounded-ios shadow-ios p-6 text-center text-sm text-text-secondary">
              {t('app.no_results')}
            </div>
          ) : (
            <div className="space-y-3">
              {items.map((item) => (
                <div
                  key={item.id}
                  className="relative bg-card rounded-ios shadow-ios p-4 hover:shadow-ios-lg transition-shadow"
                >
                  <CardMenu
                    options={[
                      {
                        label: t('app.edit'),
                        icon: 'edit',
                        onClick: () => navigate(`/home/${homeId}/workrequests/${item.id}/edit`),
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
                    onClick={() => navigate(`/home/${homeId}/workrequests/${item.id}`)}
                    className="block w-full text-left pr-10"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-semibold">{item.title}</span>
                      <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                        item.status === 'converted'
                          ? 'bg-success/10 text-success'
                          : item.status === 'dismissed'
                            ? 'bg-bg text-text-secondary'
                            : 'bg-primary/10 text-primary'
                      }`}>
                        {t(`homeWorkRequest.status_${item.status}`)}
                      </span>
                      <span className="text-xs px-2 py-0.5 rounded-full bg-bg text-text-secondary">
                        {t(`homeWorkRequest.category_${item.category}`)}
                      </span>
                      <span className={`text-xs px-2 py-0.5 rounded-full ${item.priority === 'high' ? 'bg-warning/10 text-warning' : 'bg-bg text-text-secondary'}`}>
                        {t(`homeWorkRequest.priority_${item.priority}`)}
                      </span>
                    </div>
                    {item.description && (
                      <p className="text-sm text-text-secondary mt-1 break-words">{item.description}</p>
                    )}
                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 mt-2 text-xs text-text-secondary">
                      <span>{item.reported_at}</span>
                      <span>{t('homeWorkRequest.attachment_count_label')}: {item.attachment_count}</span>
                      {item.maintenance_id && <span>{t('homeWorkRequest.converted_link')}</span>}
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
          title={t('homeWorkRequest.delete_title')}
          subtitle={t('homeWorkRequest.delete_subtitle').replace('{title}', deleteTarget.title)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
