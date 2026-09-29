import { useState, useMemo, useEffect, useRef } from 'react'
import { Icon } from './Icons'
import type { Service } from '../types'

interface ServicePickerModalProps {
  isOpen: boolean
  services: Service[]
  selectedIds: number[]
  onToggle: (id: number) => void
  onClose: () => void
  title?: string
  searchPlaceholder?: string
  emptyText?: string
  doneText?: string
}

export default function ServicePickerModal({
  isOpen,
  services,
  selectedIds,
  onToggle,
  onClose,
  title = 'Agregar servicio',
  searchPlaceholder = 'Buscar servicio...',
  emptyText = 'No se encontraron servicios',
  doneText = 'Listo',
}: ServicePickerModalProps) {
  const [search, setSearch] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isOpen])

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    let list = services
    if (term) {
      list = services.filter(s => s.name.toLowerCase().includes(term))
    }
    // Pinned selected first
    return [...list].sort((a, b) => {
      const aSelected = selectedIds.includes(a.id) ? 0 : 1
      const bSelected = selectedIds.includes(b.id) ? 0 : 1
      return aSelected - bSelected || a.name.localeCompare(b.name)
    })
  }, [services, search, selectedIds])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/40 p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-md mx-4 max-h-[80vh] flex flex-col">
        <div className="p-4 border-b border-border">
          <h3 className="text-lg font-bold">{title}</h3>
          <p className="text-sm text-text-secondary mt-1">
            {selectedIds.length > 0 ? `${selectedIds.length} ${selectedIds.length === 1 ? 'seleccionado' : 'seleccionados'}` : ''}
          </p>
        </div>

        <div className="p-4 border-b border-border">
          <div className="relative">
            <Icon name="search" className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-secondary" />
            <input
              ref={inputRef}
              type="text"
              placeholder={searchPlaceholder}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          {filtered.length === 0 ? (
            <p className="text-center text-text-secondary text-sm py-8">{emptyText}</p>
          ) : (
            <div className="space-y-1">
              {filtered.map((service) => {
                const isSelected = selectedIds.includes(service.id)
                return (
                  <button
                    key={service.id}
                    type="button"
                    onClick={() => onToggle(service.id)}
                    className={`w-full flex items-center justify-between px-3 py-3 rounded-ios-sm transition-colors min-h-[44px] ${
                      isSelected ? 'bg-primary/10' : 'hover:bg-bg'
                    }`}
                  >
                    <span className="flex items-center gap-2 min-w-0">
                      <Icon name={service.icon_key || 'services'} className="w-4 h-4 text-text-secondary flex-shrink-0" />
                      <span className={`text-sm font-medium truncate ${isSelected ? 'text-primary' : 'text-text'}`}>
                        {service.name}
                      </span>
                    </span>
                    <div
                      className={`relative w-11 h-6 rounded-full transition-colors flex-shrink-0 ${
                        isSelected ? 'bg-primary' : 'bg-border'
                      }`}
                    >
                      <div
                        className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
                          isSelected ? 'translate-x-5' : 'translate-x-0'
                        }`}
                      />
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </div>

        <div className="p-4 border-t border-border flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
          >
            {doneText}
          </button>
        </div>
      </div>
    </div>
  )
}