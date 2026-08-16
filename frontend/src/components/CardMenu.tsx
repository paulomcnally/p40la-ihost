import { useState, useRef, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Icon } from './Icons'

export interface CardMenuOption {
  label: string
  icon?: string
  danger?: boolean
  onClick: () => void
}

export default function CardMenu({ options }: { options: CardMenuOption[] }) {
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<{ top: number; right: number } | null>(null)
  const btnRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        btnRef.current && !btnRef.current.contains(e.target as Node) &&
        menuRef.current && !menuRef.current.contains(e.target as Node)
      ) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const toggle = () => {
    if (!open && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect()
      const menuWidth = 160
      let left = rect.right - menuWidth
      left = Math.max(8, Math.min(left, window.innerWidth - menuWidth - 8))
      setPos({ top: rect.bottom + 4, right: left })
    }
    setOpen(!open)
  }

  const handleOptionClick = (opt: CardMenuOption) => {
    setOpen(false)
    opt.onClick()
  }

  return (
    <div className="absolute top-3 right-3 z-50">
      <button
        ref={btnRef}
        onClick={(e) => {
          e.stopPropagation()
          toggle()
        }}
        className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors"
        aria-label="Acciones"
      >
        <Icon name="more" className="w-5 h-5 text-text-secondary" />
      </button>
      {open && pos &&
        createPortal(
          <div
            ref={menuRef}
            style={{ position: 'fixed', top: pos.top, left: pos.right, zIndex: 9999 }}
            className="bg-card border border-border rounded-ios-sm shadow-ios-lg min-w-40 overflow-hidden"
          >
            {options.map((opt, i) => (
              <button
                key={i}
                onClick={(e) => {
                  e.stopPropagation()
                  handleOptionClick(opt)
                }}
                className={`w-full flex items-center gap-2 px-3 py-2.5 text-sm transition-colors ${
                  opt.danger
                    ? 'text-danger hover:bg-danger/5'
                    : 'hover:bg-bg'
                }`}
              >
                {opt.icon && <Icon name={opt.icon} className="w-4 h-4" />}
                {opt.label}
              </button>
            ))}
          </div>,
          document.body
        )}
    </div>
  )
}
