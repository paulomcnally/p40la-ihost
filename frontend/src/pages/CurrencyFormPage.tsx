import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import DeleteModal from '../components/DeleteModal'
import type { Currency } from '../types'

export default function CurrencyFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { currencies, loadCurrencies } = useAppStore()
  const { t } = useI18nStore()
  const isEdit = !!id
  const [formData, setFormData] = useState({ code: '', name: '', symbol: '' })
  const [loading, setLoading] = useState(false)
  const [showDelete, setShowDelete] = useState(false)

  useEffect(() => {
    if (isEdit) {
      const currency = currencies.find(c => c.id === Number(id))
      if (currency) {
        setFormData({ code: currency.code, name: currency.name, symbol: currency.symbol })
      }
    }
  }, [id, isEdit, currencies])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      if (isEdit) {
        await api.currencies.update(Number(id), formData)
      } else {
        await api.currencies.create(formData)
      }
      loadCurrencies()
      navigate('/settings')
    } catch {
      // handled
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async () => {
    await api.currencies.delete(Number(id))
    loadCurrencies()
    navigate('/settings')
  }

  return (
    <div className="max-w-xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">{t(isEdit ? 'app.edit' : 'settings.currencies.create')}</h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('settings.currencies.code')}</label>
          <input
            type="text"
            maxLength={3}
            value={formData.code}
            onChange={(e) => setFormData(prev => ({ ...prev, code: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary uppercase min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('settings.currencies.name')}</label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('settings.currencies.symbol')}</label>
          <input
            type="text"
            value={formData.symbol}
            onChange={(e) => setFormData(prev => ({ ...prev, symbol: e.target.value }))}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div className="flex justify-between items-center pt-4 border-t border-border">
          {isEdit ? (
            <button
              type="button"
              onClick={() => setShowDelete(true)}
              className="px-4 py-2 bg-danger text-white rounded-ios-sm hover:bg-red-700 transition-colors flex items-center gap-2 min-h-[44px]"
            >
              <Icon name="delete" className="w-4 h-4" />
              {t('app.delete')}
            </button>
          ) : (
            <div />
          )}
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => navigate('/settings')}
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
        </div>
      </form>
      {showDelete && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('settings.currencies.title')}: ${formData.code}`}
          onConfirm={handleDelete}
          onCancel={() => setShowDelete(false)}
        />
      )}
    </div>
  )
}
