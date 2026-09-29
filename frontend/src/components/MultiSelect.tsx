import { useState, useRef, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Icon } from './Icons'
import type { SelectOption } from './Select'

export interface MultiSelectProps {
  options: SelectOption[]
  value: (string | number)[]
  onChange: (value: (string | number)[]) => void
  placeholder?: string
  searchable?: boolean
  zIndex?: string
}

export default function MultiSelect({ options, value, onChange, placeholder, searchable = false, zIndex = 'z-50' }: MultiSelectProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [menuPos, setMenuPos] = useState<{ top: number; left: number; width: number } | null>(null)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const selected = options.filter(o => value.includes(o.value))
  const filtered = searchable
    ? options.filter(o => o.label.toLowerCase().includes(search.toLowerCase()))
    : options

  const close = () => {
    setOpen(false)
    setSearch('')
  }

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        close()
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  useEffect(() => {
    if (!open) return
    const updatePos = () => {
      if (!ref.current) return
      const rect = ref.current.getBoundingClientRect()
      setMenuPos({ top: rect.bottom + 4, left: rect.left, width: rect.width })
    }
    updatePos()
    window.addEventListener('scroll', updatePos, true)
    window.addEventListener('resize', updatePos)
    return () => {
      window.removeEventListener('scroll', updatePos, true)
      window.removeEventListener('resize', updatePos)
    }
  }, [open])

  useEffect(() => {
    if (open && searchable && inputRef.current) {
      inputRef.current.focus()
    }
  }, [open, searchable])

  const toggle = (optValue: string | number) => {
    const next = value.includes(optValue)
      ? value.filter(v => v !== optValue)
      : [...value, optValue]
    onChange(next)
  }

  const label = selected.length === 0
    ? (placeholder || 'Seleccionar...')
    : selected.map(o => o.label).join(', ')

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-3 py-2 border border-border rounded-ios-sm bg-card hover:border-primary/50 focus:outline-none focus:border-primary transition-colors text-left text-text"
      >
        <span className={`truncate ${selected.length === 0 ? 'text-text-secondary' : ''}`}>
          {label}
        </span>
        <Icon name="chevron" className={`w-4 h-4 text-text-secondary flex-shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open &&
        menuPos &&
        createPortal(
          <div
            className={`fixed bg-card border border-border rounded-ios-sm shadow-ios-lg ${zIndex} overflow-hidden`}
            style={{ top: menuPos.top, left: menuPos.left, width: menuPos.width }}
            onMouseDown={(e) => e.stopPropagation()}
          >
            {searchable && (
              <div className="p-2 border-b border-border">
                <input
                  ref={inputRef}
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Buscar..."
                  className="w-full px-2 py-1.5 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card text-text"
                />
              </div>
            )}
            <div className="max-h-48 overflow-y-auto">
              {filtered.length === 0 ? (
                <div className="px-3 py-2 text-sm text-text-secondary">Sin resultados</div>
              ) : (
                filtered.map(opt => (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => toggle(opt.value)}
                    className={`w-full flex items-center gap-2 text-left px-3 py-2 text-sm hover:bg-bg transition-colors ${
                      value.includes(opt.value) ? 'text-primary font-medium bg-primary/5' : ''
                    }`}
                  >
                    <span className={`w-4 h-4 rounded border flex items-center justify-center flex-shrink-0 ${
                      value.includes(opt.value) ? 'bg-primary border-primary text-white' : 'border-border'
                    }`}>
                      {value.includes(opt.value) && <Icon name="check" className="w-3 h-3" />}
                    </span>
                    <span className="truncate">{opt.label}</span>
                  </button>
                ))
              )}
            </div>
          </div>,
          document.body
        )}
    </div>
  )
}