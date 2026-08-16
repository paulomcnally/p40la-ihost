import { useState, useEffect } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import type { Service } from '../types'

interface AddInsuranceModalProps {
  autoId: number
  onAdd: () => void
  onCancel: () => void
}

export default function AddInsuranceModal({ autoId, onAdd, onCancel }: AddInsuranceModalProps) {
  const [services, setServices] = useState<Service[]>([])
  const [search, setSearch] = useState('')
  const [selectedServiceId, setSelectedServiceId] = useState<number | null>(null)
  const [coverageType, setCoverageType] = useState<'daños_a_terceros' | 'full_cover'>('daños_a_terceros')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    api.autos.availableServices(autoId).then((data) => {
      setServices(data || [])
      setLoading(false)
    })
  }, [autoId])

  const filtered = services.filter(s =>
    s.name.toLowerCase().includes(search.toLowerCase())
  )

  const handleSubmit = async () => {
    if (!selectedServiceId) return
    setSubmitting(true)
    try {
      await api.autos.addService(autoId, { service_id: selectedServiceId, coverage_type: coverageType })
      onAdd()
    } catch {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onCancel}>
      <div className="bg-card rounded-ios shadow-ios w-full max-w-lg max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
        <div className="p-4 border-b border-border">
          <h3 className="text-lg font-bold">Agregar Seguro</h3>
          <input
            type="text"
            placeholder="Buscar servicio..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full mt-3 px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
          />
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          {loading ? (
            <p className="text-text-secondary text-center py-4">Cargando...</p>
          ) : filtered.length === 0 ? (
            <p className="text-text-secondary text-center py-4">No hay servicios disponibles</p>
          ) : (
            <div className="space-y-2">
              {filtered.map(svc => (
                <button
                  key={svc.id}
                  onClick={() => setSelectedServiceId(svc.id)}
                  className={`w-full flex items-center gap-3 p-3 rounded-ios-sm border text-left transition-colors ${
                    selectedServiceId === svc.id
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:border-primary/50'
                  }`}
                >
                  <div className="w-9 h-9 rounded-ios bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
                    <Icon name={svc.icon_key || 'other'} className="w-5 h-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-sm truncate">{svc.name}</p>
                    <p className="text-xs text-text-secondary">${svc.suggested_amount.toFixed(2)} · {svc.frequency === 'monthly' ? 'Mensual' : 'Anual'}</p>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
        {selectedServiceId && (
          <div className="p-4 border-t border-border">
            <p className="text-sm font-medium mb-2">Tipo de cobertura</p>
            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setCoverageType('daños_a_terceros')}
                className={`flex-1 px-3 py-2 rounded-ios-sm border text-sm font-medium transition-colors ${
                  coverageType === 'daños_a_terceros'
                    ? 'border-amber-400 bg-amber-50 text-amber-700'
                    : 'border-border hover:border-amber-300'
                }`}
              >
                Daños a terceros
              </button>
              <button
                onClick={() => setCoverageType('full_cover')}
                className={`flex-1 px-3 py-2 rounded-ios-sm border text-sm font-medium transition-colors ${
                  coverageType === 'full_cover'
                    ? 'border-green-400 bg-green-50 text-green-700'
                    : 'border-border hover:border-green-300'
                }`}
              >
                Full Cover
              </button>
            </div>
            <div className="flex justify-end gap-3">
              <button
                onClick={onCancel}
                className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
              >
                Cancelar
              </button>
              <button
                onClick={handleSubmit}
                disabled={submitting}
                className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
              >
                {submitting ? 'Asociando...' : 'Asociar'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
