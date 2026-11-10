import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'

export default function BankFormPage() {
  const { id } = useParams()
  const isEdit = Boolean(id)
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  usePageTitle(isEdit ? t('banks.edit') : t('banks.create'))

  const [name, setName] = useState('')
  const [shortCode, setShortCode] = useState('')
  const [country, setCountry] = useState('')
  const [notes, setNotes] = useState('')
  const [active, setActive] = useState(true)
  const [loading, setLoading] = useState(isEdit)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!isEdit) return
    let mounted = true
    api.banks
      .get(Number(id))
      .then((bank) => {
        if (!mounted || !bank) return
        setName(bank.name)
        setShortCode(bank.short_code || '')
        setCountry(bank.country || '')
        setNotes(bank.notes || '')
        setActive(bank.active)
      })
      .catch((err) => {
        if (mounted) showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      })
      .finally(() => {
        if (mounted) setLoading(false)
      })
    return () => {
      mounted = false
    }
  }, [id, isEdit, showToast, t])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      showToast(t('banks.validation_name'), 'error')
      return
    }
    setSaving(true)
    try {
      const payload = {
        name: name.trim(),
        short_code: shortCode.trim(),
        country: country.trim(),
        notes: notes.trim(),
        active,
      }
      if (isEdit) {
        await api.banks.update(Number(id), payload)
        showToast(t('banks.updated'), 'success')
      } else {
        await api.banks.create(payload)
        showToast(t('banks.created'), 'success')
      }
      navigate('/bancos')
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  const inputClass =
    'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]'

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">
          {isEdit ? t('banks.edit') : t('banks.create')}
        </h2>
        <p className="text-sm text-text-secondary mt-1">{t('banks.subtitle')}</p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">
              {t('banks.name')} <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={100}
              className={inputClass}
              autoComplete="off"
            />
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">{t('banks.short_code')}</label>
              <input
                type="text"
                value={shortCode}
                onChange={(e) => setShortCode(e.target.value)}
                maxLength={20}
                className={inputClass}
                autoComplete="off"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('banks.country')}</label>
              <input
                type="text"
                value={country}
                onChange={(e) => setCountry(e.target.value)}
                maxLength={80}
                className={inputClass}
                autoComplete="off"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('banks.notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              maxLength={500}
              rows={3}
              className={`${inputClass} resize-y`}
            />
          </div>
          <label className="flex items-center gap-2 text-sm min-h-[44px]">
            <input
              type="checkbox"
              checked={active}
              onChange={(e) => setActive(e.target.checked)}
              className="w-4 h-4 accent-primary"
            />
            {t('banks.active')}
          </label>
        </section>

        <div className="flex flex-col sm:flex-row gap-3 pb-6">
          <button
            type="button"
            onClick={() => navigate('/bancos')}
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            type="submit"
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            {saving ? (
              <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
            ) : (
              <Icon name="save" className="w-4 h-4" />
            )}
            {saving ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </form>
    </div>
  )
}
