import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { Icon } from '../components/Icons'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import AddInsuranceModal from '../components/AddInsuranceModal'
import type { Auto, AutoService } from '../types'

export default function AutoShowPage() {
  const { id } = useParams()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const [auto, setAuto] = useState<Auto | null>(null)
  const [insurance, setInsurance] = useState<AutoService[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddModal, setShowAddModal] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<{ serviceId: number; serviceName: string } | null>(null)

  const loadAuto = useCallback(async () => {
    const data = await api.autos.get(Number(id))
    setAuto(data)
  }, [id])

  const loadInsurance = useCallback(async () => {
    const data = await api.autos.listServices(Number(id))
    setInsurance(data || [])
  }, [id])

  useEffect(() => {
    Promise.all([loadAuto(), loadInsurance()]).finally(() => setLoading(false))
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
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6 mb-6">
        <div className="flex items-start gap-4">
          <div className="w-14 h-14 rounded-ios bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
            <Icon name={auto.icon} className="w-8 h-8" />
          </div>
          <div className="flex-1 min-w-0">
            <h2 className="text-xl sm:text-2xl font-bold">{auto.brand} {auto.model}</h2>
            <p className="text-text-secondary mt-1">{auto.year} · {auto.color} · {auto.placa}</p>
            <div className="mt-3 space-y-1 text-sm text-text-secondary">
              <p><span className="font-medium">Motor:</span> {auto.motor}</p>
              <p><span className="font-medium">Chasis:</span> {auto.chasis}</p>
              <p><span className="font-medium">VIN:</span> {auto.vin}</p>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-bold">Seguros</h3>
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
                <div className="flex items-center gap-2 mb-2">
                  <Icon name="building" className="w-4 h-4 text-text-secondary" />
                  <span className="text-sm font-semibold text-text-secondary">{instName}</span>
                  <span className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full dark:bg-[#2c2c2e] dark:text-gray-400">{items.length}</span>
                </div>
                <div className="space-y-2">
                  {items.map((item) => {
                    const isExpired = item.end_date && new Date(item.end_date) < new Date()
                    return (
                      <div
                        key={item.id}
                        className={`flex items-center gap-3 p-3 rounded-ios-sm border border-border ${
                          !item.active ? 'bg-gray-50 opacity-60 dark:bg-[#2c2c2e]' : 'bg-bg'
                        }`}
                      >
                        <div className={`w-9 h-9 rounded-ios flex items-center justify-center flex-shrink-0 ${
                          item.active ? 'bg-primary/10 text-primary' : 'bg-gray-200 text-gray-400 dark:bg-[#2c2c2e] dark:text-gray-500'
                        }`}>
                          <Icon name={item.icon_key || 'other'} className="w-5 h-5" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="font-medium text-sm truncate">{item.service_name}</p>
                          <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                            <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                              item.coverage_type === 'full_cover' ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
                            }`}>
                              {item.coverage_type === 'full_cover' ? 'Full Cover' : 'Daños a terceros'}
                            </span>
                            <span className="text-xs text-text-secondary">Póliza: {item.policy_number}</span>
                            <span className="text-xs text-text-secondary">Aseguradora: {item.insurer_number}</span>
                            {item.certificate && (
                              <span className="text-xs text-text-secondary">Certificado: {item.certificate}</span>
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
                        <div className="text-right flex-shrink-0">
                          <p className="text-sm font-semibold">{formatMoney(item.suggested_amount)}</p>
                          <p className="text-xs text-text-secondary">{item.frequency === 'monthly' ? 'Mensual' : 'Anual'}</p>
                          {item.start_date && (
                            <p className="text-xs text-text-secondary mt-0.5">
                              {item.start_date}{item.end_date ? ` → ${item.end_date}` : ''}
                            </p>
                          )}
                        </div>
                        <button
                          onClick={() => setDeleteTarget({ serviceId: item.service_id, serviceName: item.service_name })}
                          className="p-2 text-gray-400 hover:text-red-500 transition-colors flex-shrink-0 dark:text-gray-500 dark:hover:text-red-400"
                        >
                          <Icon name="delete" className="w-4 h-4" />
                        </button>
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {showAddModal && (
        <AddInsuranceModal
          autoId={Number(id)}
          onAdd={handleAddInsurance}
          onCancel={() => setShowAddModal(false)}
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
    </div>
  )
}
