import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import { useToast } from '../components/Toast'
import type { Child } from '../types'

export default function HijoFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const isEdit = !!id
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [notes, setNotes] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingData, setLoadingData] = useState(isEdit)

  useEffect(() => {
    if (isEdit) {
      api.children.get(Number(id)).then((child) => {
        if (child) {
          setFirstName(child.first_name)
          setLastName(child.last_name)
          setBirthDate(child.birth_date)
          setNotes(child.notes)
        }
        setLoadingData(false)
      })
    }
  }, [id, isEdit])

  usePageTitle(isEdit ? (firstName ? `${firstName} ${lastName}`.trim() : null) : t('hijos.create'))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const data: Partial<Child> = { first_name: firstName, last_name: lastName, birth_date: birthDate, notes }
      if (isEdit) {
        await api.children.update(Number(id), data)
        showToast(t('hijos.updated'), 'success')
      } else {
        await api.children.create(data)
        showToast(t('hijos.created'), 'success')
      }
      navigate('/pension/hijos')
    } catch (err: any) {
      showToast(err.message || t('hijos.save_error'), 'error')
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
        {isEdit ? t('hijos.edit') : t('hijos.new')}
      </h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('hijos.first_name')} *</label>
          <input
            type="text"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
            placeholder={t('hijos.first_name_placeholder')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('hijos.last_name')} *</label>
          <input
            type="text"
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
            placeholder={t('hijos.last_name_placeholder')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('hijos.birth_date')} *</label>
          <input
            type="date"
            value={birthDate}
            onChange={(e) => setBirthDate(e.target.value)}
            max={new Date().toISOString().split('T')[0]}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('hijos.notes')}</label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder={t('hijos.notes_placeholder')}
            rows={3}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] resize-none"
          />
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/pension/hijos')}
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