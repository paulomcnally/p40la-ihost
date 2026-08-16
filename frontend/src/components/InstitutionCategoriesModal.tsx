import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import DeleteModal from './DeleteModal'
import { useToast } from './Toast'
import type { InstitutionCategory } from '../types'

interface InstitutionCategoriesModalProps {
  isOpen: boolean
  onClose: () => void
}

const ICON_OPTIONS = [
  'shield', 'signal', 'monitor', 'bolt', 'flame', 'water', 'trash', 'bank',
  'credit', 'refresh', 'film', 'education', 'health', 'vehicle', 'home',
  'lock', 'globe', 'cloud', 'wrench', 'briefcase', 'other',
]

export default function InstitutionCategoriesModal({ isOpen, onClose }: InstitutionCategoriesModalProps) {
  const { showToast } = useToast()
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<InstitutionCategory | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [formData, setFormData] = useState({ key: '', name: '', description: '', icon_key: 'other' })
  const [saving, setSaving] = useState(false)

  const loadCategories = useCallback(async () => {
    const data = await api.institutionCategories.list()
    setCategories(data || [])
    setLoading(false)
  }, [])

  useEffect(() => {
    if (isOpen) {
      loadCategories()
      setShowForm(false)
      setEditing(null)
    }
  }, [isOpen, loadCategories])

  const handleCreate = () => {
    setEditing(null)
    setFormData({ key: '', name: '', description: '', icon_key: 'other' })
    setShowForm(true)
  }

  const handleEdit = (cat: InstitutionCategory) => {
    setEditing(cat)
    setFormData({ key: cat.key, name: cat.name, description: cat.description, icon_key: cat.icon_key })
    setShowForm(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      if (editing) {
        await api.institutionCategories.update(editing.id, {
          name: formData.name,
          description: formData.description,
          icon_key: formData.icon_key,
        })
        showToast('Categoría actualizada', 'success')
      } else {
        await api.institutionCategories.create({
          key: formData.key,
          name: formData.name,
          description: formData.description,
          icon_key: formData.icon_key,
        })
        showToast('Categoría creada', 'success')
      }
      setShowForm(false)
      loadCategories()
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || 'Error'
      showToast(message, 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.institutionCategories.delete(deleteTarget.id)
      setDeleteTarget(null)
      showToast('Categoría eliminada', 'success')
      loadCategories()
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || 'Error'
      showToast(message, 'error')
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div className="bg-card rounded-ios shadow-ios w-full max-w-lg max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
        <div className="p-4 border-b border-border flex items-center justify-between">
          <h3 className="text-lg font-bold">Categorías de instituciones</h3>
          {!showForm && (
            <button onClick={handleCreate} className="px-3 py-1.5 bg-primary text-white rounded-ios-sm text-sm font-medium flex items-center gap-1.5 min-h-[36px]">
              <Icon name="plus" className="w-4 h-4" />
              Nueva
            </button>
          )}
        </div>

        {showForm ? (
          <form onSubmit={handleSubmit} className="p-4 space-y-4">
            {!editing && (
              <div>
                <label className="block text-sm font-medium mb-1">Key (identificador interno)</label>
                <input
                  type="text"
                  value={formData.key}
                  onChange={e => setFormData(prev => ({ ...prev, key: e.target.value }))}
                  placeholder="ej: insurance"
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] text-sm"
                  required
                  disabled={!!editing}
                  pattern="[a-z][a-z0-9_]*"
                  title="Lowercase, sin espacios, solo letras, números y guiones bajos"
                />
                <p className="text-xs text-text-secondary mt-1">No se puede cambiar después de creado</p>
              </div>
            )}
            <div>
              <label className="block text-sm font-medium mb-1">Nombre</label>
              <input
                type="text"
                value={formData.name}
                onChange={e => setFormData(prev => ({ ...prev, name: e.target.value }))}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] text-sm"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Descripción</label>
              <input
                type="text"
                value={formData.description}
                onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">Ícono</label>
              <div className="flex flex-wrap gap-2">
                {ICON_OPTIONS.map(icon => (
                  <button
                    key={icon}
                    type="button"
                    onClick={() => setFormData(prev => ({ ...prev, icon_key: icon }))}
                    className={`w-10 h-10 rounded-ios-sm flex items-center justify-center border transition-colors ${
                      formData.icon_key === icon
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border hover:border-primary/50'
                    }`}
                  >
                    <Icon name={icon} className="w-5 h-5" />
                  </button>
                ))}
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button type="button" onClick={() => setShowForm(false)} className="px-4 py-2 text-sm bg-bg rounded-ios-sm hover:bg-border transition-colors min-h-[44px]">
                Cancelar
              </button>
              <button type="submit" disabled={saving} className="px-4 py-2 text-sm bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]">
                {saving ? 'Guardando...' : 'Guardar'}
              </button>
            </div>
          </form>
        ) : (
          <div className="flex-1 overflow-y-auto p-4">
            {loading ? (
              <p className="text-text-secondary text-center py-4">Cargando...</p>
            ) : categories.length === 0 ? (
              <p className="text-text-secondary text-center py-4">No hay categorías</p>
            ) : (
              <div className="space-y-2">
                {categories.map(cat => (
                  <div key={cat.id} className="flex items-center gap-3 p-3 rounded-ios-sm border border-border hover:border-primary/30 transition-colors">
                    <div className="w-9 h-9 rounded-ios bg-primary/10 text-primary flex items-center justify-center flex-shrink-0">
                      <Icon name={cat.icon_key} className="w-5 h-5" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-sm">{cat.name}</p>
                      <p className="text-xs text-text-secondary truncate">{cat.key}{cat.description ? ` · ${cat.description}` : ''}</p>
                    </div>
                    <button onClick={() => handleEdit(cat)} className="p-2 text-text-secondary hover:text-primary transition-colors">
                      <Icon name="edit" className="w-4 h-4" />
                    </button>
                    <button onClick={() => setDeleteTarget({ id: cat.id, name: cat.name })} className="p-2 text-text-secondary hover:text-red-500 transition-colors">
                      <Icon name="delete" className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {deleteTarget && (
        <DeleteModal
          title="Eliminar categoría"
          subtitle={`¿Eliminar "${deleteTarget.name}"?`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
