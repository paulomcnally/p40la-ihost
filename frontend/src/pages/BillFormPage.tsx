import { useState, useEffect } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import Select from '../components/Select'
import type { Bill, Service } from '../types'

export default function BillFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const serviceId = searchParams.get('service')
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { currencies } = useAppStore()
  const isEdit = !!id
  const [service, setService] = useState<Service | null>(null)
  const [formData, setFormData] = useState({
    year: new Date().getFullYear(),
    month: new Date().getMonth() + 1,
    amount: '0.00',
    invoice_number: '',
    status: 'pending' as 'pending' | 'paid',
    drive_url: '',
  })
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (serviceId) {
      api.services.get(Number(serviceId)).then(setService)
    }
  }, [serviceId])

  useEffect(() => {
    if (isEdit) {
      const load = async () => {
        const bill = await api.bills.get(Number(id))
        if (bill) {
          setFormData({
            year: bill.year,
            month: bill.month,
            amount: String(bill.amount),
            invoice_number: bill.invoice_number || '',
            status: bill.status,
            drive_url: bill.drive_url || '',
          })
        }
      }
      load()
    }
  }, [id, isEdit])

  if (!serviceId) {
    navigate('/services')
    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const data: Partial<Bill> = {
        service_id: Number(serviceId),
        year: formData.year,
        month: service?.frequency === 'yearly' ? 0 : formData.month,
        amount: parseFloat(formData.amount) || 0,
        invoice_number: formData.invoice_number,
        status: formData.status,
        drive_url: formData.drive_url,
      }
      if (isEdit) {
        await api.bills.update(Number(id), data)
      } else {
        await api.bills.create(data)
      }
      showToast(t('bills.saved_success') || 'Factura guardada', 'success')
      navigate(`/services/bills/${serviceId}`)
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
      showToast(message, 'error')
    } finally {
      setLoading(false)
    }
  }

  const months = Array.from({ length: 12 }, (_, i) => i + 1)
  const isYearly = service?.frequency === 'yearly'

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-6">
      <h2 className="text-xl font-bold mb-2">{t(isEdit ? 'bills.edit' : 'bills.create')}</h2>
      {service && <p className="text-text-secondary text-sm mb-6">{service.name}</p>}
      <form onSubmit={handleSubmit} className="space-y-5">
        <div className={isYearly ? '' : 'grid grid-cols-2 gap-4'}>
          <div>
            <label className="block text-sm font-medium mb-1">{t('bills.year')}</label>
            <input
              type="number"
              value={formData.year}
              onChange={(e) => setFormData(prev => ({ ...prev, year: Number(e.target.value) }))}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
              required
            />
          </div>
          {!isYearly && (
            <div>
              <label className="block text-sm font-medium mb-1">{t('bills.month')}</label>
              <Select
                options={months.map(m => ({ value: m, label: t(`months.${m}`, String(m)) }))}
                value={formData.month}
                onChange={(v) => setFormData(prev => ({ ...prev, month: v as number }))}
                searchable
              />
            </div>
          )}
          {isYearly && (
            <div className="flex items-center px-3 py-2 bg-bg rounded-ios-sm border border-border">
              <span className="text-sm text-text-secondary">{t('bills.annual')}</span>
            </div>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('bills.amount')}</label>
          <input
            type="number"
            step="0.01"
            value={formData.amount}
            onChange={(e) => setFormData(prev => ({ ...prev, amount: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('bills.invoice_number')}</label>
          <input
            type="text"
            value={formData.invoice_number}
            onChange={(e) => setFormData(prev => ({ ...prev, invoice_number: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('bills.status')}</label>
          <Select
            options={[
              { value: 'pending', label: t('bills.status_pending') },
              { value: 'paid', label: t('bills.status_paid') },
            ]}
            value={formData.status}
            onChange={(v) => setFormData(prev => ({ ...prev, status: v as 'pending' | 'paid' }))}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('bills.drive_url')}</label>
          <input
            type="url"
            value={formData.drive_url}
            onChange={(e) => setFormData(prev => ({ ...prev, drive_url: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
            required={formData.status === 'paid'}
          />
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate(`/services/bills/${serviceId}`)}
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
