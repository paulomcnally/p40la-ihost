import { useState, useRef, useEffect } from 'react'
import { Icon } from './Icons'

export interface CreateMenuOption {
  label: string
  icon?: string
  onClick: () => void
}

export default function CreateMenu({ options }: { options: CreateMenuOption[] }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
        title="Crear"
      >
        <Icon name="more" className="w-5 h-5" />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 bg-card border border-border rounded-ios-sm shadow-ios-lg z-50 min-w-44 overflow-hidden">
          {options.map((opt, i) => (
            <button
              key={i}
              onClick={() => {
                setOpen(false)
                opt.onClick()
              }}
              className="w-full flex items-center gap-2 px-3 py-2.5 text-sm hover:bg-bg transition-colors"
            >
              {opt.icon && <Icon name={opt.icon} className="w-4 h-4" />}
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
