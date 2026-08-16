import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { Icon, iconNames } from '../components/Icons'
import { useToast } from '../components/Toast'
import type { Auto } from '../types'

const vehicleIcons = ['van', 'truck', 'vehicle', 'moto']

export default function AutoFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { showToast } = useToast()
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
          setColor(auto.color)
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const data: Partial<Auto> = { year, model, brand, color, icon, motor, chasis, vin, placa }
      if (isEdit) {
        await api.autos.update(Number(id), data)
        showToast('Auto actualizado', 'success')
      } else {
        await api.autos.create(data)
        showToast('Auto creado', 'success')
      }
      navigate('/autos')
    } catch (err: any) {
      showToast(err.message || 'Error al guardar', 'error')
    } finally {
      setLoading(false)
    }
  }

  if (loadingData) {
    return (
      <div className="flex items-center justify-center min-h-[200px]">
        <div className="text-text-secondary">Cargando...</div>
      </div>
    )
  }

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">{isEdit ? 'Editar Auto' : 'Crear Auto'}</h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">Año</label>
          <input
            type="number"
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
            min={1900}
            max={2100}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Marca</label>
          <input
            type="text"
            value={brand}
            onChange={(e) => setBrand(e.target.value)}
            placeholder="Ej: Toyota"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Modelo</label>
          <input
            type="text"
            value={model}
            onChange={(e) => setModel(e.target.value)}
            placeholder="Ej: Corolla"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Color</label>
          <input
            type="text"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            placeholder="Ej: Rojo"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Motor</label>
          <input
            type="text"
            value={motor}
            onChange={(e) => setMotor(e.target.value)}
            placeholder="Ej: 2ZR12345678"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Chasis</label>
          <input
            type="text"
            value={chasis}
            onChange={(e) => setChasis(e.target.value)}
            placeholder="Ej: JTDBU4EE1B9123456"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">VIN</label>
          <input
            type="text"
            value={vin}
            onChange={(e) => setVin(e.target.value.toUpperCase())}
            placeholder="Ej: JTDBU4EE1B9123456"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Placa</label>
          <input
            type="text"
            value={placa}
            onChange={(e) => setPlaca(e.target.value.toUpperCase())}
            placeholder="Ej: P123ABC"
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">Ícono</label>
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
                <span className="text-xs capitalize">{iconName}</span>
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
            Cancelar
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors flex items-center gap-2 min-h-[44px]"
          >
            <Icon name="save" className="w-4 h-4" />
            {isEdit ? 'Actualizar' : 'Guardar'}
          </button>
        </div>
      </form>
    </div>
  )
}
