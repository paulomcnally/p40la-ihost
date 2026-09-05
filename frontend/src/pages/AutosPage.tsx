import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import type { Auto } from '../types'

export default function AutosPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const [autos, setAutos] = useState<Auto[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  usePageTitle(t('autos.title'))

  const loadAutos = useCallback(async () => {
    const data = await api.autos.list()
    setAutos(data || [])
  }, [])

  useEffect(() => {
    loadAutos().finally(() => setLoading(false))
  }, [])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.autos.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadAutos()
  }, [deleteTarget])

  if (loading) {
    return <LoadingSpinner />
  }

  if (autos.length === 0) {
    return (
      <EmptyCard
        icon="vehicle"
        title="No hay autos"
        subtitle="Registra tu primer vehículo para comenzar"
        actionLabel="Crear Auto"
        onAction={() => navigate('/autos/new')}
      />
    )
  }

  const createOptions = [
    { label: 'Crear Auto', icon: 'plus', onClick: () => navigate('/autos/new') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">Autos</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {autos.map((auto) => (
          <div
            key={auto.id}
            onClick={() => navigate(`/autos/${auto.id}`)}
            className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
          >
            <CardMenu
              options={[
                { label: 'Editar', icon: 'edit', onClick: () => navigate(`/autos/edit/${auto.id}`) },
                { label: 'Eliminar', icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: auto.id, name: `${auto.brand} ${auto.model}` }) },
              ]}
            />
            <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
              <Icon name={auto.icon} className="w-6 h-6" />
            </div>
            <h3 className="font-semibold text-base">{auto.brand} {auto.model}</h3>
            <p className="text-sm text-text-secondary mt-1">{auto.year} · {auto.color} · {auto.placa}</p>
          </div>
        ))}
      </div>
      {deleteTarget && (
        <DeleteModal
          title="Confirmar eliminación"
          subtitle={`Auto: ${deleteTarget.name}`}
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
