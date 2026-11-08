import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CardMenu from '../components/CardMenu'
import CreateMenu from '../components/CreateMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import type { DebtInformationTemplate } from '../types'

function formatUpdated(value: string | undefined, lang: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(lang === 'en' ? 'en-US' : 'es-AR', {
    dateStyle: 'medium',
  }).format(date)
}

export default function DebtInformationTemplatesPage() {
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  const { showToast } = useToast()
  usePageTitle(t('deudas.templates_info'))

  const [templates, setTemplates] = useState<DebtInformationTemplate[]>([])
  const [deleteTarget, setDeleteTarget] = useState<DebtInformationTemplate | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      const list = await api.debtInformationTemplates.list()
      setTemplates(list || [])
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast, t])

  useEffect(() => {
    load()
  }, [load])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await api.debtInformationTemplates.delete(deleteTarget.id)
      showToast(t('deudas.template_deleted'), 'success')
      setDeleteTarget(null)
      await load()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      setDeleteTarget(null)
    }
  }, [deleteTarget, load, showToast, t])

  if (loading) return <LoadingSpinner />

  const createOptions = [
    {
      label: t('deudas.template_create'),
      icon: 'plus',
      onClick: () => navigate('/deudas/plantillas/informacion/new'),
    },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <div>
          <h2 className="text-xl sm:text-2xl font-bold">{t('deudas.templates_info')}</h2>
          <p className="text-sm text-text-secondary mt-1">{t('deudas.templates_info_description')}</p>
        </div>
        <CreateMenu options={createOptions} title={t('app.actions')} />
      </div>

      {templates.length === 0 ? (
        <EmptyCard
          title={t('deudas.template_empty')}
          subtitle={t('deudas.template_empty_hint')}
          actionLabel={t('deudas.template_create')}
          onAction={() => navigate('/deudas/plantillas/informacion/new')}
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {templates.map((template) => (
            <div
              key={template.id}
              onClick={() => navigate(`/deudas/plantillas/informacion/edit/${template.id}`)}
              className="relative bg-card rounded-ios shadow-ios p-4 pr-14 cursor-pointer hover:shadow-ios-lg transition-shadow"
            >
              <CardMenu
                options={[
                  {
                    label: t('app.edit'),
                    icon: 'edit',
                    onClick: () => navigate(`/deudas/plantillas/informacion/edit/${template.id}`),
                  },
                  {
                    label: t('app.delete'),
                    icon: 'delete',
                    danger: true,
                    onClick: () => {
                      if (template.usage_count > 0) {
                        showToast(t('deudas.template_in_use'), 'error')
                        return
                      }
                      setDeleteTarget(template)
                    },
                  },
                ]}
              />
              <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
                <Icon name="copy" className="w-6 h-6" />
              </div>
              <h3 className="font-semibold text-base break-words">{template.name}</h3>
              {template.description && (
                <p className="text-sm text-text-secondary mt-1 break-words">{template.description}</p>
              )}
              <div className="flex flex-wrap items-center gap-2 mt-3">
                <span className="text-xs font-medium px-2 py-1 rounded-full bg-bg text-text-secondary">
                  {t('deudas.template_field_count').replace('{n}', String(template.field_count))}
                </span>
                {template.usage_count > 0 && (
                  <span className="text-xs font-medium px-2 py-1 rounded-full bg-primary/10 text-primary">
                    {t('deudas.template_usage_count').replace('{n}', String(template.usage_count))}
                  </span>
                )}
                {template.updated_at && (
                  <span className="text-xs text-text-secondary">
                    {formatUpdated(template.updated_at, lang)}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('deudas.templates_info')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function EmptyCard({ title, subtitle, actionLabel, onAction }: {
  title: string
  subtitle: string
  actionLabel: string
  onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name="copy" className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-sm text-text-secondary mb-5">{subtitle}</p>
      <button
        onClick={onAction}
        className="px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium hover:opacity-90 transition-opacity min-h-[44px]"
      >
        {actionLabel}
      </button>
    </div>
  )
}
