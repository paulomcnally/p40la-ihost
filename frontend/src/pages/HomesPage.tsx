import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { SortableGrid } from '../components/sortable/SortableGrid'
import { SortableCard } from '../components/sortable/SortableCard'
import { useSortableOrder } from '../hooks/useSortableOrder'
import { useToast } from '../components/Toast'
import type { Home } from '../types'

export default function HomesPage() {
  const navigate = useNavigate()
  const { homes, loadHomes, setHomes } = useAppStore()
  const { t } = useI18nStore()
  usePageTitle(t('home.title'))
  const { showToast } = useToast()
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadHomes().finally(() => setLoading(false))
  }, [])

  const { handleReorder } = useSortableOrder<Home>({
    items: homes,
    setItems: setHomes,
    onPersist: (ids) => api.homes.reorder(ids),
    onError: () => showToast(t('home.order_error'), 'error'),
  })

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.homes.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadHomes()
  }, [deleteTarget])

  if (loading) {
    return <LoadingSpinner />
  }

  if (homes.length === 0) {
    return (
      <EmptyCard
        icon="home"
        title={t('home.empty')}
        subtitle={t('home.subtitle')}
        actionLabel={t('home.create')}
        onAction={() => navigate('/home/new')}
      />
    )
  }

  const createOptions = [
    { label: t('home.create'), icon: 'plus', onClick: () => navigate('/home/new') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('home.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>
      <SortableGrid
        items={homes}
        onReorder={handleReorder}
        className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4"
      >
        {(home) => (
          <SortableCard
            key={home.id}
            id={home.id}
            handle={<Icon name="gripVertical" className="w-4 h-4" />}
            handleAriaLabel={t('home.reorder')}
            className="relative bg-card rounded-ios shadow-ios p-4 pl-10"
          >
            <CardMenu
              options={[
                { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/home/edit/${home.id}`) },
                { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: home.id, name: home.name }) },
              ]}
            />
            <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
              <Icon name="home" className="w-6 h-6" />
            </div>
            <h3 className="font-semibold text-base">{home.name}</h3>
            {home.address && (
              <p className="text-sm text-text-secondary mt-1">{home.address}</p>
            )}
          </SortableCard>
        )}
      </SortableGrid>
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('home.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function EmptyCard({ icon, title, subtitle, actionLabel, onAction }: {
  icon: string; title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name={icon} className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-text-secondary mb-6">{subtitle}</p>
      <button
        onClick={onAction}
        className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
      >
        <Icon name="plus" className="w-5 h-5" />
        {actionLabel}
      </button>
    </div>
  )
}