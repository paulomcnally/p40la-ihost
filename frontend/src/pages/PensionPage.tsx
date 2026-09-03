import { useI18nStore } from '../stores/i18nStore'
import { Icon } from '../components/Icons'

interface PensionPageProps {
  section: string
}

const sectionMeta: Record<string, { icon: string; titleKey: string; emptyKey: string }> = {
  hijos: { icon: 'baby', titleKey: 'pension.children', emptyKey: 'pension.children_empty' },
  categorias: { icon: 'tag', titleKey: 'pension.categories', emptyKey: 'pension.categories_empty' },
  salarios: { icon: 'savings', titleKey: 'pension.salaries', emptyKey: 'pension.salaries_empty' },
  registros: { icon: 'calendar', titleKey: 'pension.records', emptyKey: 'pension.records_empty' },
  notificaciones: { icon: 'bell', titleKey: 'pension.notifications', emptyKey: 'pension.notifications_empty' },
}

export default function PensionPage({ section }: PensionPageProps) {
  const { t } = useI18nStore()
  const meta = sectionMeta[section] || sectionMeta.hijos

  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name={meta.icon} className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{t(meta.titleKey)}</h3>
      <p className="text-text-secondary">{t(meta.emptyKey)}</p>
    </div>
  )
}