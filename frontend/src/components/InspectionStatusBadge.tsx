import { useI18nStore } from '../stores/i18nStore'
import type { InspectionStatus } from '../types'

const statusClasses: Record<InspectionStatus, string> = {
  overdue: 'bg-danger/10 text-danger',
  due_soon: 'bg-warning/10 text-warning',
  ok: 'bg-success/10 text-success',
}

export default function InspectionStatusBadge({
  status,
  compact = false,
}: {
  status: InspectionStatus
  compact?: boolean
}) {
  const { t } = useI18nStore()
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full font-semibold ${statusClasses[status]} ${
        compact ? 'px-2 py-0.5 text-[11px]' : 'px-2.5 py-1 text-xs'
      }`}
    >
      {t(`autos.inspections.status_${status}`)}
    </span>
  )
}
