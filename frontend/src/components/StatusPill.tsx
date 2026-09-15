export type StatusPillTone = 'success' | 'muted' | 'warning'

interface StatusPillProps {
  label: string
  tone: StatusPillTone
}

const toneClasses: Record<StatusPillTone, string> = {
  success: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
  muted: 'bg-bg text-text-secondary border border-border',
  warning: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
}

export default function StatusPill({ label, tone }: StatusPillProps) {
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium shrink-0 ${toneClasses[tone]}`}>
      {label}
    </span>
  )
}