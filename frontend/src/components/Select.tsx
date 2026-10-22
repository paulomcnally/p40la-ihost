import { useState, useRef, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Icon } from './Icons'

export interface SelectOption {
  value: string | number
  label: string
}

export interface SelectProps {
  options: SelectOption[]
  value: string | number
  onChange: (value: string | number) => void
  placeholder?: string
  searchable?: boolean
  searchPlaceholder?: string
  emptyLabel?: string
  zIndex?: string
}

// normalizeText quita acentos y mayúsculas para que el buscador encuentre
// "marron" al escribir sobre "Marrón" (SPEC-118).
function normalizeText(value: string): string {
  return value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase()
}

export default function Select({
  options,
  value,
  onChange,
  placeholder,
  searchable = false,
  searchPlaceholder = 'Buscar...',
  emptyLabel = 'Sin resultados',
  zIndex = 'z-50',
}: SelectProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [menuPos, setMenuPos] = useState<{ top: number; left: number; width: number } | null>(null)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const selected = options.find(o => o.value === value)
  const filtered = searchable
    ? options.filter(o => normalizeText(o.label).includes(normalizeText(search)))
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

  const handleSelect = (opt: SelectOption) => {
    onChange(opt.value)
    close()
  }

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full min-h-[44px] flex items-center justify-between px-3 py-2 border border-border rounded-ios-sm bg-card hover:border-primary/50 focus:outline-none focus:border-primary transition-colors text-left text-text"
      >
        <span className={selected ? '' : 'text-text-secondary'}>
          {selected?.label || placeholder || 'Seleccionar...'}
        </span>
        <Icon name="chevron" className={`w-4 h-4 text-text-secondary transition-transform ${open ? 'rotate-180' : ''}`} />
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
                  placeholder={searchPlaceholder}
                  className="w-full px-2 py-1.5 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card"
                />
              </div>
            )}
            <div className="max-h-48 overflow-y-auto">
              {filtered.length === 0 ? (
                <div className="px-3 py-2 text-sm text-text-secondary">{emptyLabel}</div>
              ) : (
                filtered.map(opt => (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => handleSelect(opt)}
                    className={`w-full text-left px-3 py-2 text-sm hover:bg-bg transition-colors ${
                      opt.value === value ? 'text-primary font-medium bg-primary/5' : ''
                    }`}
                  >
                    {opt.label}
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