import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { Icon } from '../components/Icons'
import { useToast } from '../components/Toast'
import Select from '../components/Select'
import { AUTO_COLORS, isAutoColor } from '../constants/autoColors'
import type { Auto } from '../types'

const vehicleIcons = ['van', 'truck', 'vehicle', 'moto']

const fieldClass =
  'w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]'

export default function AutoFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { showToast } = useToast()
  const { t } = useI18nStore()
  const isEdit = !!id
  const [year, setYear] = useState(new Date().getFullYear())
  const [model, setModel] = useState('')
  const [brand, setBrand] = useState('')
  const [color, setColor] = useState('')
  const [icon, setIcon] = useState('vehicle')
  const [motor, setMotor] = useState('')
  const [chasis, setChasis] = useState('')
  const [vin, setVin] = useState('')
  const [placa, setPlaca] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingData, setLoadingData] = useState(isEdit)

  useEffect(() => {
    if (isEdit) {
      api.autos.get(Number(id)).then((auto) => {
        if (auto) {
          setYear(auto.year)
          setModel(auto.model)
          setBrand(auto.brand)
          // Los colores fuera de la paleta (legado) requieren reselección.
          setColor(isAutoColor(auto.color) ? auto.color : '')
          setIcon(auto.icon)
          setMotor(auto.motor)
          setChasis(auto.chasis)
          setVin(auto.vin)
          setPlaca(auto.placa)
        }
        setLoadingData(false)
      })
    }
  }, [id, isEdit])

  usePageTitle(isEdit ? (brand ? `${brand} ${model}`.trim() : null) : t('autos.create'))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!color) {
      showToast(t('autos.color_required'), 'error')
      return
    }
    setLoading(true)
    try {
      const data: Partial<Auto> = { year, model, brand, color, icon, motor, chasis, vin, placa }
      if (isEdit) {
        await api.autos.update(Number(id), data)
        showToast(t('autos.updated'), 'success')
      } else {
        await api.autos.create(data)
        showToast(t('autos.created'), 'success')
      }
      navigate('/autos')
    } catch (err: any) {
      showToast(err.message || t('autos.save_error'), 'error')
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
        {isEdit ? t('autos.form_title_edit') : t('autos.form_title_create')}
      </h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('autos.year')}</label>
          <input
            type="number"
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
            min={1900}
            max={2100}
            className={fieldClass}
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('autos.brand')}</label>
          <input
            type="text"
            value={brand}
            onChange={(e) => setBrand(e.target.value)}
            placeholder={t('autos.brand_placeholder')}
            className={fieldClass}
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('autos.model')}</label>
          <input
            type="text"
            value={model}
            onChange={(e) => setModel(e.target.value)}
            placeholder={t('autos.model_placeholder')}
            className={fieldClass}
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('autos.color')}</label>
          <Select
            options={AUTO_COLORS.map((code) => ({ value: code, label: t(`autos.colors.${code}`) }))}
            value={color}
            onChange={(value) => setColor(String(value))}
            placeholder={t('autos.color_placeholder')}
            searchable
            searchPlaceholder={t('app.search')}
            emptyLabel={t('app.no_results')}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">
            {t('autos.motor')}{' '}
            <span className="text-text-secondary font-normal">{t('autos.optional')}</span>
          </label>
          <input
            type="text"
            value={motor}
            onChange={(e) => setMotor(e.target.value)}
            placeholder={t('autos.motor_placeholder')}
            className={fieldClass}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">
            {t('autos.chasis')}{' '}
            <span className="text-text-secondary font-normal">{t('autos.optional')}</span>
          </label>
          <input
            type="text"
            value={chasis}
            onChange={(e) => setChasis(e.target.value)}
            placeholder={t('autos.chasis_placeholder')}
            className={fieldClass}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">
            {t('autos.vin')}{' '}
            <span className="text-text-secondary font-normal">{t('autos.optional')}</span>
          </label>
          <input
            type="text"
            value={vin}
            onChange={(e) => setVin(e.target.value.toUpperCase())}
            placeholder={t('autos.vin_placeholder')}
            className={fieldClass}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">
            {t('autos.placa')}{' '}
            <span className="text-text-secondary font-normal">{t('autos.optional')}</span>
          </label>
          <input
            type="text"
            value={placa}
            onChange={(e) => setPlaca(e.target.value.toUpperCase())}
            placeholder={t('autos.placa_placeholder')}
            className={fieldClass}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">{t('autos.icon')}</label>
          <div className="grid grid-cols-4 gap-2">
            {vehicleIcons.map((iconName) => (
              <button
                key={iconName}
                type="button"
                onClick={() => setIcon(iconName)}
                className={`flex flex-col items-center gap-1 p-3 rounded-ios-sm border-2 transition-colors min-h-[44px] ${
                  icon === iconName
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border hover:border-primary/50 text-text-secondary'
                }`}
              >
                <Icon name={iconName} className="w-6 h-6" />
                <span className="text-xs">{t(`autos.icons.${iconName}`)}</span>
              </button>
            ))}
          </div>
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/autos')}
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
            {isEdit ? t('autos.update') : t('app.save')}
          </button>
        </div>
      </form>
    </div>
  )
}
