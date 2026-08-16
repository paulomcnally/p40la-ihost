import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import AnalyzerPickerModal from '../components/AnalyzerPickerModal'
import type { AnalyzerInfo } from '../types'

export default function InstitutionFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
  const [formData, setFormData] = useState({ name: '', analyzer_ids: [] as string[] })
  const [analyzers, setAnalyzers] = useState<AnalyzerInfo[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    loadAnalyzers()
  }, [])

  useEffect(() => {
    if (isEdit) {
      loadInstitution()
    }
  }, [id])

  const loadAnalyzers = async () => {
    try {
      const list = await api.analyzers.list()
      setAnalyzers(list || [])
    } catch {
      // ignore
    }
  }

  const loadInstitution = async () => {
    try {
      const inst = await api.institutions.get(Number(id))
      if (inst) {
        setFormData(prev => ({ ...prev, name: inst.name }))
      }
      const instAnalyzers = await api.institutions.getAnalyzers(Number(id))
      if (instAnalyzers) {
        const ids = instAnalyzers.map((a: { analyzer_id: string }) => a.analyzer_id)
        setFormData(prev => ({ ...prev, analyzer_ids: ids }))
      }
    } catch {
      // ignore
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.name.trim()) return
    setLoading(true)
    try {
      if (isEdit) {
        await api.institutions.update(Number(id), { name: formData.name })
        await api.institutions.setAnalyzers(Number(id), formData.analyzer_ids)
        showToast('Institución actualizada', 'success')
      } else {
        const inst = await api.institutions.create({ name: formData.name })
        if (inst) {
          await api.institutions.setAnalyzers(inst.id, formData.analyzer_ids)
        }
        showToast('Institución creada', 'success')
      }
      navigate('/institutions')
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || 'Error'
      showToast(message, 'error')
    } finally {
      setLoading(false)
    }
  }

  const toggleAnalyzer = (analyzerId: string) => {
    setFormData(prev => ({
      ...prev,
      analyzer_ids: prev.analyzer_ids.includes(analyzerId)
        ? prev.analyzer_ids.filter(aId => aId !== analyzerId)
        : [...prev.analyzer_ids, analyzerId],
    }))
  }

  const removeAnalyzer = (analyzerId: string) => {
    setFormData(prev => ({
      ...prev,
      analyzer_ids: prev.analyzer_ids.filter(aId => aId !== analyzerId),
    }))
  }

  const assignedAnalyzers = analyzers.filter(a => formData.analyzer_ids.includes(a.id))

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">{isEdit ? 'Editar institución' : 'Nueva institución'}</h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">Nombre</label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>

        <div className="border border-border rounded-ios p-4 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Analizadores asignados</span>
            {assignedAnalyzers.length > 0 && (
              <span className="text-xs text-text-secondary">{assignedAnalyzers.length}</span>
            )}
          </div>

          {assignedAnalyzers.length === 0 ? (
            <p className="text-sm text-text-secondary">
              No hay analizadores asignados. Agregá uno para que el servicio pueda analizar facturas automáticamente.
            </p>
          ) : (
            <div className="space-y-2">
              {assignedAnalyzers.map((a) => (
                <div
                  key={a.id}
                  className="flex items-center justify-between bg-bg rounded-ios-sm px-3 py-2"
                >
                  <span className="flex items-center gap-2 text-sm">
                    <Icon name="pdf" className="w-4 h-4 text-primary" />
                    {a.name}
                  </span>
                  <button
                    type="button"
                    onClick={() => removeAnalyzer(a.id)}
                    className="p-2 text-text-secondary hover:text-red-500 transition-colors rounded-full"
                    aria-label={`Remover ${a.name}`}
                  >
                    <Icon name="cancel" className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          )}

          <button
            type="button"
            onClick={() => setModalOpen(true)}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-primary/10 text-primary rounded-ios-sm hover:bg-primary/20 transition-colors min-h-[44px] text-sm font-medium"
          >
            <Icon name="plus" className="w-4 h-4" />
            Agregar analizador
          </button>
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/institutions')}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center gap-2 min-h-[44px]"
          >
            <Icon name="cancel" className="w-4 h-4" />
            Cancelar
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px]"
          >
            <Icon name="save" className="w-4 h-4" />
            Guardar
          </button>
        </div>
      </form>

      <AnalyzerPickerModal
        isOpen={modalOpen}
        available={analyzers}
        selectedIds={formData.analyzer_ids}
        onToggle={toggleAnalyzer}
        onClose={() => setModalOpen(false)}
      />
    </div>
  )
}
