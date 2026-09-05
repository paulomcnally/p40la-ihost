import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import type { Notification } from '../types'

export default function NotificationsPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  usePageTitle(t('notifications.title'))
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  const loadNotifications = useCallback(async () => {
    const data = await api.notifications.list()
    setNotifications(data || [])
  }, [])

  useEffect(() => {
    loadNotifications().finally(() => setLoading(false))
  }, [])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.notifications.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadNotifications()
  }, [deleteTarget])

  if (loading) {
    return <LoadingSpinner />
  }

  if (notifications.length === 0) {
    return (
      <EmptyCard
        icon="bell"
        title={t('notifications.empty_title')}
        subtitle={t('notifications.empty_subtitle')}
        actionLabel={t('notifications.create')}
        onAction={() => navigate('/pension/notificaciones/new')}
      />
    )
  }

  const createOptions = [
    { label: t('notifications.create'), icon: 'plus', onClick: () => navigate('/pension/notificaciones/new') },
  ]

  return (
    <div className="space-y-8">
      <div>
        <div className="flex items-center justify-between mb-4 sm:mb-5">
          <h2 className="text-xl sm:text-2xl font-bold">{t('pension.notifications')}</h2>
          <CreateMenu options={createOptions} />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {notifications.map((notification) => (
            <div
              key={notification.id}
              className={`bg-card rounded-ios shadow-ios p-4 relative transition-shadow ${notification.active ? '' : 'opacity-70'}`}
            >
              <CardMenu
                options={[
                  { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/pension/notificaciones/edit/${notification.id}`) },
                  { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: notification.id, name: notification.name }) },
                ]}
              />
              <div className="flex items-center justify-between mb-3">
                <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center">
                  <Icon name="bell" className="w-6 h-6" />
                </div>
                <span
                  className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                    notification.active
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                      : 'bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400'
                  }`}
                >
                  {notification.active ? t('notifications.active') : t('notifications.inactive')}
                </span>
              </div>
              <h3 className="font-semibold text-base">{notification.name}</h3>
              <p className="text-sm text-text-secondary mt-1 break-all">{notification.email}</p>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios p-6 sm:p-8">
        <h3 className="text-lg font-semibold mb-2 flex items-center gap-2">
          <Icon name="mail" className="w-5 h-5 text-primary" />
          {t('notifications.history_title')}
        </h3>
        <p className="text-text-secondary text-sm">{t('notifications.history_placeholder')}</p>
      </div>

      {deleteTarget && (
        <DeleteModal
          title={t('notifications.delete_confirm')}
          subtitle={deleteTarget.name}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function EmptyCard({ icon, title, subtitle, actionLabel, onAction }: {
  icon: string; title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name={icon} className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-text-secondary mb-6">{subtitle}</p>
      <button
        onClick={onAction}
        className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
      >
        <Icon name="plus" className="w-5 h-5" />
        {actionLabel}
      </button>
    </div>
  )
}