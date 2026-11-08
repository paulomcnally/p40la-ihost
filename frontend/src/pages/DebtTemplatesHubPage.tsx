import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import type { DebtInformationTemplate } from '../types'

function formatUpdated(value: string | undefined, lang: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(lang === 'en' ? 'en-US' : 'es-AR', {
    dateStyle: 'medium',
  }).format(date)
}

export default function DebtTemplatesHubPage() {
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  usePageTitle(t('deudas.templates_hub_title'))

  const [templates, setTemplates] = useState<DebtInformationTemplate[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.debtInformationTemplates
      .list()
      .then((list) => setTemplates(list || []))
      .catch(() => setTemplates([]))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <LoadingSpinner />

  const updatedValues = templates
    .map((template) => template.updated_at)
    .filter(Boolean)
    .sort()
  const lastUpdated = updatedValues[updatedValues.length - 1]

  return (
    <div>
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('deudas.templates_hub_title')}</h2>
        <p className="text-sm text-text-secondary mt-1">{t('deudas.templates_hub_subtitle')}</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        <div
          role="button"
          tabIndex={0}
          onClick={() => navigate('/deudas/plantillas/informacion')}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              navigate('/deudas/plantillas/informacion')
            }
          }}
          className="text-left bg-card rounded-ios shadow-ios p-4 hover:shadow-ios-lg transition-shadow cursor-pointer"
        >
          <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
            <Icon name="copy" className="w-6 h-6" />
          </div>
          <h3 className="font-semibold text-base">{t('deudas.templates_info')}</h3>
          <p className="text-sm text-text-secondary mt-1">{t('deudas.templates_info_description')}</p>
          <div className="flex flex-wrap items-center gap-2 mt-3">
            <span className="text-xs font-medium px-2 py-1 rounded-full bg-bg text-text-secondary">
              {t('deudas.templates_info_count').replace('{n}', String(templates.length))}
            </span>
            {lastUpdated && (
              <span className="text-xs text-text-secondary">
                {t('deudas.templates_info_updated')}: {formatUpdated(lastUpdated, lang)}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
