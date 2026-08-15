import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import type { AnalyzerInfo } from '../types'

export default function InstitutionFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
  const [formData, setFormData] = useState({ name: '', analyzer_ids: [] as string[] })
  const [analyzers, setAnalyzers] = useState<AnalyzerInfo[]>([])
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
        if (inst && formData.analyzer_ids.length > 0) {
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

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-6">
      <h2 className="text-xl font-bold mb-6">{isEdit ? 'Editar institución' : 'Nueva institución'}</h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">Nombre</label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
            required
          />
        </div>
        {analyzers.length > 0 && (
          <div>
            <label className="block text-sm font-medium mb-2">Analizadores disponibles</label>
            <div className="space-y-2">
              {analyzers.map(a => (
                <label key={a.id} className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    checked={formData.analyzer_ids.includes(a.id)}
                    onChange={() => toggleAnalyzer(a.id)}
                    className="w-4 h-4 rounded border-border"
                  />
                  <span className="text-sm">{a.name}</span>
                </label>
              ))}
            </div>
          </div>
        )}
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/institutions')}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center gap-2"
          >
            <Icon name="cancel" className="w-4 h-4" />
            Cancelar
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2"
          >
            <Icon name="save" className="w-4 h-4" />
            Guardar
          </button>
        </div>
      </form>
    </div>
  )
}
