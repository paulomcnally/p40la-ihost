import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import Select from '../components/Select'
import type { Debt, DebtStatus, Institution } from '../types'

export default function DebtFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { currencies, loadAll } = useAppStore()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id

  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [noInstitutions, setNoInstitutions] = useState(false)
  const [loading, setLoading] = useState(false)
  const [formData, setFormData] = useState({
    institution_id: 0,
    identifier: '',
    description: '',
    total: '0.00',
    principal: '0.00',
    currency_id: 0,
    installments_total: '12',
    installment_amount: '0.00',
    interest_rate: '0.00',
    payment_day: '15',
    start_date: '',
    status: 'activa' as DebtStatus,
  })

  useEffect(() => {
    loadAll()
    api.institutions
      .list()
      .then((list) => {
        setInstitutions(list || [])
        setNoInstitutions(!list || list.length === 0)
      })
      .catch(() => setNoInstitutions(true))
  }, [])

  useEffect(() => {
    if (isEdit) {
      const load = async () => {
        const debt = await api.debts.get(Number(id))
        if (debt) {
          setFormData({
            institution_id: debt.institution_id,
            identifier: debt.identifier || '',
            description: debt.description || '',
            total: String(debt.total),
            principal: String(debt.principal),
            currency_id: debt.currency_id,
            installments_total: String(debt.installments_total),
            installment_amount: String(debt.installment_amount),
            interest_rate: String(debt.interest_rate),
            payment_day: String(debt.payment_day),
            start_date: debt.start_date || '',
            status: debt.status,
          })
        }
      }
      load()
    } else if (currencies.length > 0) {
      setFormData((prev) => ({
        ...prev,
        currency_id: currencies[0].id,
        start_date: new Date().toISOString().slice(0, 10),
      }))
    }
  }, [id, isEdit, currencies])

  usePageTitle(isEdit ? (formData.description || null) : t('deudas.create'))

  if (noInstitutions) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-amber-500 opacity-80">
          <Icon name="building" className="w-full h-full" />
        </div>
        <h3 className="text-xl font-semibold mb-2">{t('deudas.empty_no_institution')}</h3>
        <button
          onClick={() => navigate('/institutions/new')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
        >
          <Icon name="plus" className="w-5 h-5" />
          {'Nueva institución'}
        </button>
      </div>
    )
  }

  if (currencies.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-amber-500 opacity-80">
          <Icon name="savings" className="w-full h-full" />
        </div>
        <h3 className="text-xl font-semibold mb-2">{t('deudas.empty_no_currency')}</h3>
        <button
          onClick={() => navigate('/settings/currency')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
        >
          <Icon name="plus" className="w-5 h-5" />
          {'Nueva moneda'}
        </button>
      </div>
    )
  }

  const handleChange = (field: string, value: string | number | boolean | null) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const paymentDay = parseInt(formData.payment_day, 10) || 0
    if (paymentDay < 1 || paymentDay > 31) {
      showToast(t('deudas.payment_day') + ': 1-31', 'error')
      return
    }
    if (!formData.start_date) {
      showToast(t('deudas.start_date'), 'error')
      return
    }
    setLoading(true)
    try {
      const data: Partial<Debt> = {
        institution_id: formData.institution_id,
        identifier: formData.identifier.trim(),
        description: formData.description.trim(),
        total: parseFloat(formData.total) || 0,
        principal: parseFloat(formData.principal) || 0,
        currency_id: formData.currency_id,
        installments_total: parseInt(formData.installments_total, 10) || 1,
        installment_amount: parseFloat(formData.installment_amount) || 0,
        interest_rate: parseFloat(formData.interest_rate) || 0,
        payment_day: paymentDay,
        start_date: formData.start_date,
        status: formData.status,
      }
      if (isEdit) {
        await api.debts.update(Number(id), data)
      } else {
        await api.debts.create(data)
      }
      showToast(t('deudas.saved_success'), 'success')
      navigate('/deudas')
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
      showToast(message, 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">
        {t(isEdit ? 'deudas.edit' : 'deudas.create')}
      </h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('deudas.creditor')}</label>
          <Select
            options={institutions.map((i) => ({ value: i.id, label: i.name }))}
            value={formData.institution_id}
            onChange={(v) => handleChange('institution_id', v as number)}
            searchable
            placeholder={t('deudas.creditor')}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('deudas.identifier')}</label>
          <input
            type="text"
            value={formData.identifier}
            onChange={(e) => handleChange('identifier', e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            placeholder="1111 2222 3333 4444"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('deudas.description')}</label>
          <input
            type="text"
            value={formData.description}
            onChange={(e) => handleChange('description', e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            placeholder="Tarjeta de crédito"
            required
          />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.total')}</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={formData.total}
              onChange={(e) => handleChange('total', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.principal')}</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={formData.principal}
              onChange={(e) => handleChange('principal', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            />
          </div>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.currency')}</label>
            <Select
              options={currencies.map((c) => ({ value: c.id, label: `${c.code} (${c.symbol})` }))}
              value={formData.currency_id}
              onChange={(v) => handleChange('currency_id', v as number)}
              searchable
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.status')}</label>
            <Select
              options={[
                { value: 'activa', label: t('deudas.status_activa') },
                { value: 'inactiva', label: t('deudas.status_inactiva') },
                { value: 'finalizada', label: t('deudas.status_finalizada') },
              ]}
              value={formData.status}
              onChange={(v) => handleChange('status', v as DebtStatus)}
            />
          </div>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.installments_total')}</label>
            <input
              type="number"
              min="1"
              step="1"
              value={formData.installments_total}
              onChange={(e) => handleChange('installments_total', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.installment_amount')}</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={formData.installment_amount}
              onChange={(e) => handleChange('installment_amount', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            />
          </div>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.interest_rate')}</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={formData.interest_rate}
              onChange={(e) => handleChange('interest_rate', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.payment_day')}</label>
            <input
              type="number"
              min="1"
              max="31"
              step="1"
              value={formData.payment_day}
              onChange={(e) => handleChange('payment_day', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('deudas.start_date')}</label>
          <input
            type="date"
            value={formData.start_date}
            onChange={(e) => handleChange('start_date', e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
            required
          />
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/deudas')}
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