import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'
import type { AutomationPlugin } from '../types'

export default function AutomationPluginsPage() {
  const { t } = useI18nStore()
  usePageTitle(t('automation.plugin_catalog'))
  const [plugins, setPlugins] = useState<AutomationPlugin[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.automation.plugins()
      .then((list) => setPlugins(list || []))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <LoadingSpinner />

  return (
    <div>
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('automation.plugin_catalog')}</h2>
        <p className="text-sm text-text-secondary mt-1">{t('automation.plugin_catalog_desc')}</p>
      </div>

      {plugins.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="bolt" className="w-full h-full" />
          </div>
          <p className="text-sm text-text-secondary">{t('automation.no_plugins')}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {plugins.map((plugin) => (
            <div key={plugin.id} className="bg-card rounded-ios shadow-ios p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
                  <Icon name="bolt" className="w-6 h-6" />
                </div>
                <span className={`text-xs px-2 py-1 rounded-full ${
                  plugin.status === 'active' ? 'bg-success/10 text-success' : 'bg-bg text-text-secondary'
                }`}>
                  {plugin.status === 'active' ? t('automation.status_active') : plugin.status}
                </span>
              </div>
              <h3 className="font-semibold text-base mt-3 break-all">{plugin.name}</h3>
              <p className="text-sm text-text-secondary mt-1">
                {t('automation.version')} {plugin.version} · {plugin.source}
              </p>
              {plugin.description && (
                <p className="text-sm text-text-secondary mt-2">{plugin.description}</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
