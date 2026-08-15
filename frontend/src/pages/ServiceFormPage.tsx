import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import IconPickerModal from '../components/IconPickerModal'
import Select from '../components/Select'
import type { Service } from '../types'

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
  })
  const [loading, setLoading] = useState(false)
  const [showIconPicker, setShowIconPicker] = useState(false)

  useEffect(() => {
    loadAll()
  }, [])

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
    navigate('/home/new')
    return null
  }

  const handleChange = (field: string, value: string | number | boolean) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
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
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-6">
      <h2 className="text-xl font-bold mb-6">{t(isEdit ? 'services.edit' : 'services.create')}</h2>
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
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('services.institution')}</label>
          <input
            type="text"
            value={formData.institution}
            onChange={(e) => handleChange('institution', e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
          />
        </div>
        <div className="grid grid-cols-2 gap-4">
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
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
            required
          />
        </div>
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
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center gap-2"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2"
          >
            <Icon name="save" className="w-4 h-4" />
            {t('app.save')}
          </button>
        </div>
      </form>
    </div>
  )
}
