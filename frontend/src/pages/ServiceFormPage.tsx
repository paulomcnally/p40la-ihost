import { useState, useEffect } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import IconPickerModal from '../components/IconPickerModal'
import Select, { ADD_OPTION_VALUE, addOption } from '../components/Select'
import Toggle from '../components/Toggle'
import type { Service, Institution, InstitutionCategory, AnalyzerInfo } from '../types'

export default function ServiceFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { homes, currencies, loadAll } = useAppStore()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
  const [searchParams] = useSearchParams()
  const rawQueryHomeId = Number(searchParams.get('home') || 0)
  const queryHomeId = Number.isInteger(rawQueryHomeId) && rawQueryHomeId > 0 ? rawQueryHomeId : 0
  const [formData, setFormData] = useState({
    home_id: 0,
    name: '',
    institution: '',
    currency_id: 0,
    frequency: 'monthly' as 'monthly' | 'yearly',
    suggested_amount: '0.00',
    active: true,
    icon_key: 'other',
    billing_type: 'variable' as 'fixed' | 'variable',
    billing_day: null as number | null,
    auto_generate: false,
    institution_id: 0,
    institution_analyzer_id: 0,
    is_recurring: false,
    start_date: '',
    end_date: '',
    automation_account_id: '',
  })
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [analyzerOptions, setAnalyzerOptions] = useState<{id: number, analyzer_id: string, analyzer_name: string}[]>([])
  const [loading, setLoading] = useState(false)
  const [showIconPicker, setShowIconPicker] = useState(false)
  const [noInstitutions, setNoInstitutions] = useState(false)
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [quickOpen, setQuickOpen] = useState(false)
  const [quickName, setQuickName] = useState('')
  const [quickCategoryId, setQuickCategoryId] = useState<number | ''>('')
  const [quickSaving, setQuickSaving] = useState(false)
  const [quickError, setQuickError] = useState('')

  useEffect(() => {
    loadAll()
    loadInstitutions(true)
    loadCategories()
  }, [])

  const loadInstitutions = async (initial = false) => {
    try {
      const list = await api.institutions.list()
      setInstitutions(list || [])
      setNoInstitutions(!list || list.length === 0)
    } catch {
      if (initial) setNoInstitutions(true)
    }
  }

  const loadCategories = async () => {
    try {
      const list = await api.institutionCategories.list()
      setCategories(list || [])
    } catch {
      setCategories([])
    }
  }

  useEffect(() => {
    if (formData.institution_id) {
      loadAnalyzerOptions(formData.institution_id)
    } else {
      setAnalyzerOptions([])
    }
  }, [formData.institution_id])

  const loadAnalyzerOptions = async (institutionId: number) => {
    try {
      const options = await api.services.getAnalyzerOptions(institutionId)
      setAnalyzerOptions(options || [])
    } catch {
      setAnalyzerOptions([])
    }
  }

  useEffect(() => {
    if (isEdit && homes.length > 0) {
      const load = async () => {
        const svc = await api.services.get(Number(id))
        if (svc) {
          setFormData({
            home_id: svc.home_id,
            name: svc.name,
            institution: svc.institution || '',
            currency_id: svc.currency_id,
            frequency: svc.frequency,
            suggested_amount: String(svc.suggested_amount),
            active: svc.active,
            icon_key: svc.icon_key || 'other',
            billing_type: svc.billing_type || 'variable',
            billing_day: svc.billing_day ?? null,
            auto_generate: svc.auto_generate || false,
            institution_id: svc.institution_id || 0,
            institution_analyzer_id: svc.institution_analyzer_id || 0,
            is_recurring: svc.is_recurring || false,
            start_date: svc.start_date || '',
            end_date: svc.end_date || '',
            automation_account_id: svc.automation_account_id ? String(svc.automation_account_id) : '',
          })
        }
      }
      load()
    } else if (!isEdit && homes.length > 0 && currencies.length > 0) {
      const filteredHome = homes.find((h) => h.id === queryHomeId)
      setFormData(prev => ({
        ...prev,
        home_id: filteredHome ? filteredHome.id : homes[0].id,
        currency_id: currencies[0].id,
      }))
    }
  }, [id, isEdit, homes, currencies, queryHomeId])

  usePageTitle(isEdit ? (formData.name || null) : t('services.create'))

  if (homes.length === 0) {
    return (
      <DependencyWarning
        icon="home"
        title="Primero debes crear una casa."
        subtitle="Los servicios pertenecen a una casa. Crea una casa antes de continuar."
        actionLabel="Nueva casa"
        onAction={() => navigate('/home/new')}
      />
    )
  }

  if (noInstitutions) {
    return (
      <DependencyWarning
        icon="building"
        title="Primero debes crear una institución."
        subtitle="Los servicios requieren una institución asociada. Crea una institución antes de continuar."
        actionLabel="Nueva institución"
        onAction={() => navigate('/institutions/new')}
      />
    )
  }

  const handleChange = (field: string, value: string | number | boolean | null) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const filteredHomeId = !isEdit && homes.some((h) => h.id === queryHomeId) ? queryHomeId : 0
  const servicesPath = filteredHomeId ? `/services?home=${filteredHomeId}` : '/services'

  const openQuickCreate = () => {
    setQuickName('')
    setQuickCategoryId('')
    setQuickError('')
    setQuickOpen(true)
  }

  const handleQuickCreate = async () => {
    const name = quickName.trim()
    if (!name) {
      setQuickError(t('institutions.name_required'))
      return
    }
    setQuickSaving(true)
    setQuickError('')
    try {
      const created = await api.institutions.create({
        name,
        category_id: quickCategoryId === '' ? null : Number(quickCategoryId),
      })
      if (created) {
        setInstitutions(prev => (
          prev.some((institution) => institution.id === created.id)
            ? prev
            : [...prev, created].sort((a, b) => a.name.localeCompare(b.name))
        ))
        setNoInstitutions(false)
        handleChange('institution_id', created.id)
        handleChange('institution_analyzer_id', 0)
        showToast(t('institutions.created'), 'success')
      }
      setQuickOpen(false)
      loadInstitutions()
    } catch (err: unknown) {
      setQuickError(err instanceof Error ? err.message : t('errors.generic'))
    } finally {
      setQuickSaving(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (formData.auto_generate && formData.billing_type === 'fixed' && !formData.billing_day) {
      showToast(t('services.billing_day_required') || 'El día de facturación es requerido para facturación automática', 'error')
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const data: Partial<Service> = {
        home_id: formData.home_id,
        name: formData.name,
        institution: formData.institution,
        currency_id: formData.currency_id,
        frequency: formData.frequency,
        suggested_amount: parseFloat(formData.suggested_amount) || 0,
        active: formData.active,
        icon_key: formData.icon_key,
        billing_type: formData.billing_type,
        billing_day: formData.billing_day,
        auto_generate: formData.auto_generate,
        institution_id: formData.institution_id || undefined,
        institution_analyzer_id: formData.institution_analyzer_id || undefined,
        is_recurring: formData.is_recurring,
        start_date: formData.is_recurring && formData.start_date ? formData.start_date : undefined,
        end_date: formData.is_recurring && formData.end_date ? formData.end_date : undefined,
        automation_account_id: formData.automation_account_id
          ? Number(formData.automation_account_id)
          : undefined,
      }
      if (isEdit) {
        await api.services.update(Number(id), data)
      } else {
        await api.services.create(data)
      }
      showToast(t('services.saved_success') || 'Servicio guardado', 'success')
      navigate(servicesPath)
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
      showToast(message, 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">{t(isEdit ? 'services.edit' : 'services.create')}</h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('services.home')}</label>
          <Select
            options={homes.map(h => ({ value: h.id, label: h.name }))}
            value={formData.home_id}
            onChange={(v) => handleChange('home_id', v as number)}
            searchable
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('services.name')}</label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => handleChange('name', e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('services.institution')}</label>
          <Select
            options={[
              ...institutions.map(i => ({ value: i.id, label: i.name })),
              addOption(t('app.add')),
            ]}
            value={formData.institution_id}
            onChange={(v) => {
              if (v === ADD_OPTION_VALUE) {
                openQuickCreate()
                return
              }
              handleChange('institution_id', v as number)
              handleChange('institution_analyzer_id', 0)
            }}
            searchable
            placeholder="Seleccionar institución"
          />
        </div>
        {analyzerOptions.length > 0 && (
          <div>
            <label className="block text-sm font-medium mb-1">Analizador</label>
            <Select
              options={analyzerOptions.map(a => ({ value: a.id, label: a.analyzer_name }))}
              value={formData.institution_analyzer_id}
              onChange={(v) => handleChange('institution_analyzer_id', v as number)}
            />
          </div>
        )}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('services.currency')}</label>
            <Select
              options={currencies.map(c => ({ value: c.id, label: `${c.code} (${c.symbol})` }))}
              value={formData.currency_id}
              onChange={(v) => handleChange('currency_id', v as number)}
              searchable
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('services.frequency')}</label>
            <Select
              options={[
                { value: 'monthly', label: t('frequency.monthly') },
                { value: 'yearly', label: t('frequency.yearly') },
              ]}
              value={formData.frequency}
              onChange={(v) => handleChange('frequency', v as 'monthly' | 'yearly')}
            />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('services.suggested_amount')}</label>
          <input
            type="number"
            step="0.01"
            value={formData.suggested_amount}
            onChange={(e) => handleChange('suggested_amount', e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('services.billing_type')}</label>
            <Select
              options={[
                { value: 'variable', label: t('billing_type.variable') },
                { value: 'fixed', label: t('billing_type.fixed') },
              ]}
              value={formData.billing_type}
              onChange={(v) => handleChange('billing_type', v as 'fixed' | 'variable')}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('services.billing_day')}</label>
            <input
              type="number"
              min="1"
              max="31"
              value={formData.billing_day ?? ''}
              onChange={(e) => {
                const val = e.target.value
                handleChange('billing_day', val === '' ? null : parseInt(val) || null)
              }}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            />
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Toggle
            checked={formData.auto_generate}
            onChange={(checked) => handleChange('auto_generate', checked)}
            ariaLabel={t('services.auto_generate')}
          />
          <span className="text-sm font-medium">{t('services.auto_generate')}</span>
        </div>
        <div className="flex items-center gap-3">
          <Toggle
            checked={formData.is_recurring}
            onChange={(checked) => {
              handleChange('is_recurring', checked)
              if (!checked) {
                handleChange('start_date', '')
                handleChange('end_date', '')
              }
            }}
            ariaLabel="Recurrente"
          />
          <span className="text-sm font-medium">Recurrente</span>
        </div>
        {formData.is_recurring && (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Fecha de inicio</label>
              <input
                type="date"
                value={formData.start_date}
                onChange={(e) => handleChange('start_date', e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Fecha de fin (opcional)</label>
              <input
                type="date"
                value={formData.end_date}
                onChange={(e) => handleChange('end_date', e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              />
            </div>
          </div>
        )}
        <div>
          <label className="block text-sm font-medium mb-2">{t('services.icon')}</label>
          <button
            type="button"
            onClick={() => setShowIconPicker(true)}
            className="flex items-center gap-3 px-4 py-3 border border-border rounded-ios-sm hover:border-primary/50 transition-colors w-full"
          >
            <Icon name={formData.icon_key} className="w-6 h-6 text-primary" />
            <span className="text-sm text-text-secondary">{t('services.select_icon')}</span>
            <Icon name="chevron" className="w-4 h-4 ml-auto text-text-secondary" />
          </button>
        </div>
        <IconPickerModal
          isOpen={showIconPicker}
          selectedIcon={formData.icon_key}
          onSelect={(key) => setFormData(prev => ({ ...prev, icon_key: key }))}
          onClose={() => setShowIconPicker(false)}
        />
        <div className="flex items-center gap-3">
          <Toggle
            checked={formData.active}
            onChange={(checked) => handleChange('active', checked)}
            ariaLabel={t('services.active')}
          />
          <span className="text-sm font-medium">{t('services.active')}</span>
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate(servicesPath)}
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
            {t('app.save')}
          </button>
        </div>
      </form>
    </div>

    {quickOpen && (
      <div
        className="fixed inset-0 bg-black/50 flex items-center justify-center z-[70] p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="quick-institution-title"
      >
        <div className="bg-card rounded-ios shadow-ios w-full max-w-sm max-h-[90vh] flex flex-col">
          <div className="p-4 border-b border-border flex items-center justify-between gap-3">
            <h3 id="quick-institution-title" className="text-lg font-bold">{t('institutions.create')}</h3>
            <button
              type="button"
              onClick={() => setQuickOpen(false)}
              className="p-2 text-text-secondary hover:text-text rounded-full transition-colors"
              aria-label={t('app.close')}
            >
              <Icon name="cancel" className="w-5 h-5" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('institutions.name')}</label>
              <input
                type="text"
                value={quickName}
                onChange={(event) => setQuickName(event.target.value)}
                placeholder={t('institutions.name_placeholder')}
                className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] placeholder:text-text-secondary"
                autoFocus
              />
              {quickError && <p className="text-sm text-danger mt-1">{quickError}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('institutions.category')}</label>
              <Select
                options={[
                  { value: '', label: t('institutions.uncategorized') },
                  ...categories.map((category) => ({ value: category.id, label: category.name })),
                ]}
                value={quickCategoryId}
                onChange={(value) => setQuickCategoryId(value === '' ? '' : Number(value))}
                placeholder={t('institutions.select_category')}
                searchable
                searchPlaceholder={t('app.search')}
                emptyLabel={t('app.no_results')}
              />
            </div>
          </div>

          <div className="p-4 border-t border-border flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setQuickOpen(false)}
              className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
            >
              {t('app.cancel')}
            </button>
            <button
              type="button"
              onClick={handleQuickCreate}
              disabled={quickSaving}
              className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
            >
              {quickSaving ? t('app.saving') : t('app.create')}
            </button>
          </div>
        </div>
      </div>
    )}
    </>
  )
}

function DependencyWarning({ icon, title, subtitle, actionLabel, onAction }: {
  icon: string; title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-amber-500 opacity-80">
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
