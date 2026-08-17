import { useState } from 'react'
import type { ReactNode } from 'react'
import { Icon } from './Icons'

export default function HelpPanel({ title, children }: { title: string; children: ReactNode }) {
  const [open, setOpen] = useState(false)

  return (
    <div className="border-b border-border">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-4 py-3 hover:bg-bg/50 transition-colors"
      >
        <span className="inline-flex items-center gap-2 text-sm text-text-secondary">
          <Icon name="info" className="w-4 h-4 text-primary" />
          {title}
        </span>
        <Icon name="chevron" className={`w-4 h-4 text-text-secondary transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && <div className="px-4 pb-4 -mt-1 space-y-3 text-sm text-text-secondary">{children}</div>}
    </div>
  )
}
