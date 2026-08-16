import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import IconPickerModal from '../components/IconPickerModal'
import Select from '../components/Select'
import type { Service, Institution, AnalyzerInfo } from '../types'

export default function ServiceFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { homes, currencies, loadAll } = useAppStore()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
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
  })
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [analyzerOptions, setAnalyzerOptions] = useState<{id: number, analyzer_id: string, analyzer_name: string}[]>([])
  const [loading, setLoading] = useState(false)
  const [showIconPicker, setShowIconPicker] = useState(false)
  const [noInstitutions, setNoInstitutions] = useState(false)

  useEffect(() => {
    loadAll()
    loadInstitutions()
  }, [])

  const loadInstitutions = async () => {
    try {
      const list = await api.institutions.list()
      setInstitutions(list || [])
      if (!list || list.length === 0) {
        setNoInstitutions(true)
      }
    } catch {
      setNoInstitutions(true)
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
          })
        }
      }
      load()
    } else if (!isEdit && homes.length > 0 && currencies.length > 0) {
      setFormData(prev => ({
        ...prev,
        home_id: homes[0].id,
        currency_id: currencies[0].id,
      }))
    }
  }, [id, isEdit, homes, currencies])

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
      }
      if (isEdit) {
        await api.services.update(Number(id), data)
      } else {
        await api.services.create(data)
      }
      showToast(t('services.saved_success') || 'Servicio guardado', 'success')
      navigate('/services')
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
      showToast(message, 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
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
            options={institutions.map(i => ({ value: i.id, label: i.name }))}
            value={formData.institution_id}
            onChange={(v) => {
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
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={formData.auto_generate}
              onChange={(e) => handleChange('auto_generate', e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-border peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-success"></div>
          </label>
          <span className="text-sm font-medium">{t('services.auto_generate')}</span>
        </div>
        <div className="flex items-center gap-3">
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={formData.is_recurring}
              onChange={(e) => {
                handleChange('is_recurring', e.target.checked)
                if (!e.target.checked) {
                  handleChange('start_date', '')
                  handleChange('end_date', '')
                }
              }}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-border peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-success"></div>
          </label>
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
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={formData.active}
              onChange={(e) => handleChange('active', e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-border peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-success"></div>
          </label>
          <span className="text-sm font-medium">{t('services.active')}</span>
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/services')}
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
