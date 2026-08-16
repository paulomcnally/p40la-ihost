import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import InstitutionCategoriesModal from '../components/InstitutionCategoriesModal'
import type { Institution, InstitutionCategory } from '../types'

export default function InstitutionsPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [analyzerCounts, setAnalyzerCounts] = useState<Record<number, number>>({})
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [showCategoriesModal, setShowCategoriesModal] = useState(false)

  const loadData = useCallback(async () => {
    try {
      const [instList, catList] = await Promise.all([
        api.institutions.list(),
        api.institutionCategories.list(),
      ])
      setInstitutions(instList || [])
      setCategories(catList || [])
      const counts: Record<number, number> = {}
      for (const inst of instList || []) {
        const list = await api.institutions.getAnalyzers(inst.id)
        counts[inst.id] = (list || []).length
      }
      setAnalyzerCounts(counts)
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await api.institutions.delete(deleteTarget.id)
      setDeleteTarget(null)
      showToast('Institución eliminada', 'success')
      loadData()
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || 'Error'
      showToast(message, 'error')
    }
  }, [deleteTarget, loadData])

  if (institutions.length === 0) {
    return (
      <EmptyCard
        icon="building"
        title="No hay instituciones registradas"
        subtitle="Las instituciones son proveedores de servicios que generan facturas"
        actionLabel="Nueva institución"
        onAction={() => navigate('/institutions/new')}
      />
    )
  }

  const createOptions = [
    { label: 'Nueva institución', icon: 'plus', onClick: () => navigate('/institutions/new') },
    { label: 'Categorías de instituciones', icon: 'tag', onClick: () => setShowCategoriesModal(true) },
  ]

  const getCategoryName = (categoryId?: number) => {
    if (!categoryId) return null
    return categories.find(c => c.id === categoryId)?.name || null
  }

  const getCategoryIcon = (categoryId?: number) => {
    if (!categoryId) return null
    return categories.find(c => c.id === categoryId)?.icon_key || 'other'
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">Instituciones</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {institutions.map((inst) => {
          const count = analyzerCounts[inst.id] || 0
          const catName = getCategoryName(inst.category_id)
          const catIcon = getCategoryIcon(inst.category_id)
          return (
            <div
              key={inst.id}
              onClick={() => navigate(`/institutions/edit/${inst.id}`)}
              className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
            >
              <CardMenu
                options={[
                  { label: 'Editar', icon: 'edit', onClick: () => navigate(`/institutions/edit/${inst.id}`) },
                  { label: 'Eliminar', icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: inst.id, name: inst.name }) },
                ]}
              />
              <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
                <Icon name={catIcon || 'building'} className="w-6 h-6" />
              </div>
              <h3 className="font-semibold text-base">{inst.name}</h3>
              {catName && (
                <span className="inline-block mt-1 text-xs font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                  {catName}
                </span>
              )}
              <p className="text-sm text-text-secondary mt-1 flex items-center gap-1.5">
                {count > 0 ? (
                  <>
                    <Icon name="pdf" className="w-4 h-4" />
                    {count} analizador{count !== 1 ? 'es' : ''}
                  </>
                ) : (
                  'Sin analizadores configurados'
                )}
              </p>
            </div>
          )
        })}
      </div>
      {deleteTarget && (
        <DeleteModal
          title="¿Eliminar institución?"
          subtitle={`Institución: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
      <InstitutionCategoriesModal isOpen={showCategoriesModal} onClose={() => setShowCategoriesModal(false)} />
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
