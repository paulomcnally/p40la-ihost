import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import type { Institution, AnalyzerInfo } from '../types'

export default function InstitutionsPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [analyzers, setAnalyzers] = useState<AnalyzerInfo[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [instList, analyzerList] = await Promise.all([
        api.institutions.list(),
        api.analyzers.list(),
      ])
      setInstitutions(instList || [])
      setAnalyzers(analyzerList || [])
    } catch {
      // ignore
    }
  }

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
  }, [deleteTarget])

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
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-2xl font-bold">Instituciones</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {institutions.map((inst) => {
          const instAnalyzers = analyzers.filter(() => true)
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
                <Icon name="building" className="w-6 h-6" />
              </div>
              <h3 className="font-semibold text-base">{inst.name}</h3>
              <p className="text-sm text-text-secondary mt-1">
                {analyzers.length > 0 ? `${analyzers.length} analizador${analyzers.length !== 1 ? 'es' : ''} disponible${analyzers.length !== 1 ? 's' : ''}` : 'Sin analizadores configurados'}
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
    </div>
  )
}

function EmptyCard({ icon, title, subtitle, actionLabel, onAction }: {
  icon: string; title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name={icon} className="w-full h-full" />
      </div>
      <h3 className="text-xl font-semibold mb-2">{title}</h3>
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
