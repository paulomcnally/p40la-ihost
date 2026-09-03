import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import type { PensionCategory } from '../types'

export default function CategoriasPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const [categories, setCategories] = useState<PensionCategory[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  const loadCategories = useCallback(async () => {
    const data = await api.pensionCategories.list()
    setCategories(data || [])
  }, [])

  useEffect(() => {
    loadCategories().finally(() => setLoading(false))
  }, [])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.pensionCategories.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadCategories()
  }, [deleteTarget])

  if (loading) {
    return <LoadingSpinner />
  }

  if (categories.length === 0) {
    return (
      <EmptyCard
        icon="tag"
        title={t('categorias.empty')}
        subtitle={t('categorias.subtitle')}
        actionLabel={t('categorias.create')}
        onAction={() => navigate('/pension/categorias/new')}
      />
    )
  }

  const createOptions = [
    { label: t('categorias.create'), icon: 'plus', onClick: () => navigate('/pension/categorias/new') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('categorias.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {categories.map((category) => (
          <div
            key={category.id}
            onClick={() => {}}
            className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
          >
            <CardMenu
              options={[
                { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/pension/categorias/edit/${category.id}`) },
                { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: category.id, name: category.name }) },
              ]}
            />
            <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
              <Icon name="tag" className="w-6 h-6" />
            </div>
            <h3 className="font-semibold text-base">{category.name}</h3>
            {category.description && (
              <p className="text-sm text-text-secondary mt-1 line-clamp-2">{category.description}</p>
            )}
            {category.auto_generate && (
              <p className="inline-flex items-center gap-1 mt-2 text-xs font-medium text-success">
                <Icon name="refresh" className="w-3.5 h-3.5" />
                {t('categorias.auto_generate_badge')}
              </p>
            )}
          </div>
        ))}
      </div>
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('categorias.title')}: ${deleteTarget.name}`}
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