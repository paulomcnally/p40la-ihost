import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { Icon } from '../components/Icons'
import { useToast } from '../components/Toast'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import type { Notification } from '../types'

export default function NotificationFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const { showToast } = useToast()
  const { t } = useI18nStore()
  const isEdit = !!id
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [active, setActive] = useState(true)
  const [loading, setLoading] = useState(false)
  const [loadingData, setLoadingData] = useState(isEdit)

  useEffect(() => {
    if (isEdit) {
      api.notifications.get(Number(id)).then((notification) => {
        if (notification) {
          setName(notification.name)
          setEmail(notification.email)
          setActive(notification.active)
        }
        setLoadingData(false)
      })
    }
  }, [id, isEdit])

  usePageTitle(isEdit ? (name || null) : t('notifications.create'))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const data: Partial<Notification> = { name, email, active }
      if (isEdit) {
        await api.notifications.update(Number(id), data)
        showToast(t('notifications.updated'), 'success')
      } else {
        await api.notifications.create(data)
        showToast(t('notifications.created'), 'success')
      }
      navigate('/pension/notificaciones')
    } catch (err: any) {
      showToast(err.message || t('notifications.save_error'), 'error')
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
        {isEdit ? t('notifications.edit') : t('notifications.create')}
      </h2>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('notifications.name')} *</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('notifications.name_placeholder')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('notifications.email')} *</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t('notifications.email_placeholder')}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            required
          />
        </div>
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
          <span className="text-sm font-medium">{t('notifications.active_label')}</span>
        </div>
        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate('/pension/notificaciones')}
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
            {isEdit ? t('app.edit') : t('app.save')}
          </button>
        </div>
      </form>
    </div>
  )
}