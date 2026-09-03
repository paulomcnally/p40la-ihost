import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import Toggle from '../components/Toggle'
import { useToast } from '../components/Toast'
import type { PensionCategory } from '../types'

export default function CategoryFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [autoGenerate, setAutoGenerate] = useState(false)
  const [loading, setLoading] = useState(false)
  const [loadingData, setLoadingData] = useState(isEdit)

  useEffect(() => {
    if (isEdit) {
      api.pensionCategories.get(Number(id)).then((category) => {
        if (category) {
          setName(category.name)
          setDescription(category.description)
          setAutoGenerate(category.auto_generate)
        }
        setLoadingData(false)
      })
    }
  }, [id, isEdit])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const data: Partial<PensionCategory> = { name, description, auto_generate: autoGenerate }
      if (isEdit) {
        await api.pensionCategories.update(Number(id), data)
        showToast(t('categorias.updated'), 'success')
      } else {
        await api.pensionCategories.create(data)
        showToast(t('categorias.created'), 'success')
      }
      navigate('/pension/categorias')
    } catch (err: any) {
      showToast(err.message || t('categorias.save_error'), 'error')
    } finally {
      setLoading(false)
    }
  }

  if (loadingData) {
    return (
      <div className="flex items-center justify-center min-h-[200px]">
        <div className="text-text-secondary">{t('app.loading')}</div>
      </div>
    )
  }

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">
        {isEdit ? t('categorias.edit') : t('categorias.new')}
      </h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('categorias.name')} *</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('categorias.name_placeholder')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('categorias.description')}</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('categorias.description_placeholder')}
            rows={3}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] resize-none"
          />
        </div>
        <div className="flex items-center justify-between gap-4 pt-2">
          <div>
            <p className="text-sm font-medium">{t('categorias.auto_generate')}</p>
            <p className="text-xs text-text-secondary mt-0.5">{t('categorias.auto_generate_hint')}</p>
          </div>
          <Toggle checked={autoGenerate} onChange={setAutoGenerate} />
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/pension/categorias')}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center gap-2 min-h-[44px]"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px]"
          >
            <Icon name="save" className="w-4 h-4" />
            {isEdit ? t('app.save') : t('app.create')}
          </button>
        </div>
      </form>
    </div>
  )
}