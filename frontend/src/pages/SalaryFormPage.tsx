import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import Select from '../components/Select'
import type { Salary } from '../types'

export default function SalaryFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { currencies, loadCurrencies } = useAppStore()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
  const [employer, setEmployer] = useState('')
  const [amount, setAmount] = useState('')
  const [currencyId, setCurrencyId] = useState<number>(0)
  const [paymentDay, setPaymentDay] = useState('')
  const [active, setActive] = useState(true)
  const [note, setNote] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingData, setLoadingData] = useState(isEdit)

  useEffect(() => {
    loadCurrencies()
  }, [])

  useEffect(() => {
    if (isEdit) {
      api.salaries.get(Number(id)).then((salary) => {
        if (salary) {
          setEmployer(salary.employer)
          setAmount(String(salary.amount))
          setCurrencyId(salary.currency_id)
          setPaymentDay(String(salary.payment_day))
          setActive(salary.active)
          setNote(salary.note)
        }
        setLoadingData(false)
      })
    }
  }, [id, isEdit])

  useEffect(() => {
    if (!isEdit && currencies.length > 0) {
      setCurrencyId(currencies[0].id)
    }
  }, [currencies, isEdit])

  usePageTitle(isEdit ? (employer || null) : t('salaries.create'))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const data: Partial<Salary> = {
        employer,
        amount: Number(amount),
        currency_id: currencyId,
        payment_day: Number(paymentDay),
        active,
        note,
      }
      if (isEdit) {
        await api.salaries.update(Number(id), data)
        showToast(t('salaries.updated'), 'success')
      } else {
        await api.salaries.create(data)
        showToast(t('salaries.created'), 'success')
      }
      navigate('/pension/salarios')
    } catch (err: any) {
      showToast(err.message || t('salaries.save_error'), 'error')
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
        {isEdit ? t('salaries.edit') : t('salaries.new')}
      </h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('salaries.employer')} *</label>
          <input
            type="text"
            value={employer}
            onChange={(e) => setEmployer(e.target.value)}
            placeholder={t('salaries.employer_placeholder')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('salaries.amount')} *</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="0.00"
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('salaries.currency')} *</label>
            <Select
              options={currencies.map(c => ({ value: c.id, label: `${c.code} (${c.symbol})` }))}
              value={currencyId}
              onChange={(v) => setCurrencyId(v as number)}
              searchable
            />
          </div>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('salaries.payment_day')} *</label>
            <input
              type="number"
              min="1"
              max="31"
              value={paymentDay}
              onChange={(e) => setPaymentDay(e.target.value)}
              placeholder="15"
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              required
            />
          </div>
          <div className="flex items-end pb-2">
            <div className="flex items-center gap-3">
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={active}
                  onChange={(e) => setActive(e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-border peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-success"></div>
              </label>
              <span className="text-sm font-medium">{t('salaries.active')}</span>
            </div>
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('salaries.note')}</label>
          <textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={t('salaries.note_placeholder')}
            rows={3}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] resize-none"
          />
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/pension/salarios')}
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