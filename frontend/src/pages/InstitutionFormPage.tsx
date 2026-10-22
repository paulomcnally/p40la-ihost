import { useState, useEffect, useRef } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import AnalyzerPickerModal from '../components/AnalyzerPickerModal'
import Select from '../components/Select'
import type { AnalyzerInfo, InstitutionCategory } from '../types'

type FormTab = 'info' | 'analyzers'

export default function InstitutionFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
  const nameInputRef = useRef<HTMLInputElement>(null)
  const [formData, setFormData] = useState({ name: '', category_id: null as number | null, analyzer_ids: [] as string[] })
  const [analyzers, setAnalyzers] = useState<AnalyzerInfo[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [nameError, setNameError] = useState(false)

  // La tab activa vive en la URL (?tab=info|analyzers) para que el formulario
  // se pueda compartir y sobreviva a una recarga (SPEC-116 REQ-010).
  const [searchParams, setSearchParams] = useSearchParams()
  const activeTab: FormTab = searchParams.get('tab') === 'analyzers' ? 'analyzers' : 'info'
  const setActiveTab = (tab: FormTab) => {
    const next = new URLSearchParams(searchParams)
    if (tab === 'info') {
      next.delete('tab')
    } else {
      next.set('tab', tab)
    }
    setSearchParams(next, { replace: true })
  }

  useEffect(() => {
    if (nameError && activeTab === 'info') {
      nameInputRef.current?.focus()
    }
  }, [nameError, activeTab])

  useEffect(() => {
    loadAnalyzers()
    loadCategories()
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

  const loadCategories = async () => {
    try {
      const list = await api.institutionCategories.list()
      setCategories(list || [])
    } catch {
      // ignore
    }
  }

  const loadInstitution = async () => {
    try {
      const inst = await api.institutions.get(Number(id))
      if (inst) {
        setFormData(prev => ({ ...prev, name: inst.name, category_id: inst.category_id || null }))
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

  usePageTitle(isEdit ? (formData.name || null) : t('institutions.create'))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.name.trim()) {
      setNameError(true)
      setActiveTab('info')
      showToast(t('institutions.name_required'), 'error')
      return
    }
    setNameError(false)
    setLoading(true)
    try {
      const payload = { name: formData.name, category_id: formData.category_id }
      if (isEdit) {
        await api.institutions.update(Number(id), payload)
        await api.institutions.setAnalyzers(Number(id), formData.analyzer_ids)
        showToast(t('institutions.updated'), 'success')
      } else {
        const inst = await api.institutions.create(payload)
        if (inst) {
          await api.institutions.setAnalyzers(inst.id, formData.analyzer_ids)
        }
        showToast(t('institutions.created'), 'success')
      }
      navigate('/institutions')
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
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

  const tabs: { key: FormTab; labelKey: string; hasError: boolean }[] = [
    { key: 'info', labelKey: 'institutions.tab_info', hasError: nameError },
    { key: 'analyzers', labelKey: 'institutions.tab_analyzers', hasError: false },
  ]

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">{isEdit ? t('institutions.edit') : t('institutions.create')}</h2>
      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <div className="flex gap-1 p-1 bg-bg rounded-ios-sm">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              type="button"
              onClick={() => setActiveTab(tab.key)}
              aria-current={activeTab === tab.key ? 'page' : undefined}
              className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[40px] ${
                activeTab === tab.key
                  ? 'bg-card text-primary shadow-ios'
                  : 'text-text-secondary hover:text-text'
              }`}
            >
              {t(tab.labelKey)}
              {tab.hasError && (
                <span className="w-1.5 h-1.5 rounded-full bg-danger shrink-0" aria-hidden="true" />
              )}
            </button>
          ))}
        </div>

        {activeTab === 'info' && (
          <div className="space-y-5">
            <div>
              <label htmlFor="institution-name" className="block text-sm font-medium mb-1">{t('institutions.name')}</label>
              <input
                id="institution-name"
                ref={nameInputRef}
                type="text"
                value={formData.name}
                onChange={(e) => {
                  setFormData(prev => ({ ...prev, name: e.target.value }))
                  if (nameError) setNameError(false)
                }}
                placeholder={t('institutions.name_placeholder')}
                aria-invalid={nameError}
                className={`w-full px-3 py-2 border rounded-ios-sm bg-card text-text placeholder:text-text-secondary focus:outline-none min-h-[44px] ${
                  nameError ? 'border-danger focus:border-danger' : 'border-border focus:border-primary'
                }`}
                required
              />
              {nameError && <p className="text-sm text-danger mt-1">{t('institutions.name_required')}</p>}
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">{t('institutions.category')}</label>
              <Select
                options={[
                  { value: '', label: t('institutions.uncategorized') },
                  ...categories.map(cat => ({ value: cat.id, label: cat.name })),
                ]}
                value={formData.category_id || ''}
                onChange={(value) => setFormData(prev => ({ ...prev, category_id: value ? Number(value) : null }))}
                placeholder={t('institutions.select_category')}
              />
            </div>
          </div>
        )}

        {activeTab === 'analyzers' && (
          <div className="border border-border rounded-ios p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">{t('institutions.assigned_analyzers')}</span>
              {assignedAnalyzers.length > 0 && (
                <span className="text-xs text-text-secondary">{assignedAnalyzers.length}</span>
              )}
            </div>

            {assignedAnalyzers.length === 0 ? (
              <p className="text-sm text-text-secondary">
                {t('institutions.no_analyzers')}. {t('institutions.no_analyzers_hint')}
              </p>
            ) : (
              <div className="space-y-2">
                {assignedAnalyzers.map((a) => (
                  <div
                    key={a.id}
                    className="flex items-center justify-between gap-2 bg-bg rounded-ios-sm px-3 py-2"
                  >
                    <span className="flex items-center gap-2 text-sm min-w-0">
                      <Icon name="pdf" className="w-4 h-4 text-primary shrink-0" />
                      <span className="truncate">{a.name}</span>
                    </span>
                    <button
                      type="button"
                      onClick={() => removeAnalyzer(a.id)}
                      className="p-2 text-text-secondary hover:text-danger transition-colors rounded-full shrink-0"
                      aria-label={`${t('institutions.remove_analyzer')}: ${a.name}`}
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
              {t('institutions.add_analyzer')}
            </button>
          </div>
        )}

        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/institutions')}
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
            {loading ? t('app.saving') : t('app.save')}
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
