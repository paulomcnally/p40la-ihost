import { Icon } from './Icons'
import StatusPill, { type StatusPillTone } from './StatusPill'

interface SettingsNavRowProps {
  icon: string
  title: string
  subtitle?: string
  statusLabel?: string
  statusTone?: StatusPillTone
  warning?: boolean
  right?: React.ReactNode
  onClick: () => void
}

export default function SettingsNavRow({
  icon,
  title,
  subtitle,
  statusLabel,
  statusTone = 'muted',
  warning,
  right,
  onClick,
}: SettingsNavRowProps) {
  const inner = (
    <>
      <span className="w-9 h-9 shrink-0 rounded-full bg-primary/10 text-primary flex items-center justify-center">
        <Icon name={icon} className="w-5 h-5" />
      </span>
      <span className="flex-1 min-w-0">
        <span className="flex items-center gap-1.5">
          <span className="font-medium truncate">{title}</span>
          {warning && <Icon name="warning" className="w-4 h-4 text-amber-500 shrink-0" />}
        </span>
        {subtitle && <span className="block text-sm text-text-secondary truncate">{subtitle}</span>}
      </span>
      {statusLabel && <StatusPill label={statusLabel} tone={statusTone} />}
      {right ?? <Icon name="chevron" className="w-4 h-4 text-text-secondary shrink-0" />}
    </>
  )

  const cls = 'w-full flex items-center gap-3 px-4 py-3.5 hover:bg-bg/50 transition-colors text-left'

  if (right) {
    return <div className={cls}>{inner}</div>
  }
  return (
    <button type="button" onClick={onClick} className={cls}>
      {inner}
    </button>
  )
}