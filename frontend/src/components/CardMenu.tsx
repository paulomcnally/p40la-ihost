import { useState, useRef, useEffect } from 'react'
import { Icon } from './Icons'

export interface CardMenuOption {
  label: string
  icon?: string
  danger?: boolean
  onClick: () => void
}

export default function CardMenu({ options }: { options: CardMenuOption[] }) {
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
    <div className="absolute top-3 right-3 z-50" ref={ref}>
      <button
        onClick={(e) => {
          e.stopPropagation()
          setOpen(!open)
        }}
        className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
      >
        <Icon name="more" className="w-5 h-5 text-text-secondary" />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 bg-card border border-border rounded-ios-sm shadow-ios-lg min-w-36 overflow-hidden">
          {options.map((opt, i) => (
            <button
              key={i}
              onClick={(e) => {
                e.stopPropagation()
                setOpen(false)
                opt.onClick()
              }}
              className={`w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors ${
                opt.danger
                  ? 'text-danger hover:bg-danger/5'
                  : 'hover:bg-bg'
              }`}
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
