import { useState, useMemo, useEffect, useRef } from 'react'
import { Icon } from './Icons'
import type { AnalyzerInfo } from '../types'

interface AnalyzerPickerModalProps {
  isOpen: boolean
  available: AnalyzerInfo[]
  selectedIds: string[]
  onToggle: (id: string) => void
  onClose: () => void
}

export default function AnalyzerPickerModal({
  isOpen,
  available,
  selectedIds,
  onToggle,
  onClose,
}: AnalyzerPickerModalProps) {
  const [search, setSearch] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const modalRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isOpen])

  useEffect(() => {
    if (!isOpen) return
    const handler = (e: MouseEvent) => {
      if (modalRef.current && !modalRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [isOpen, onClose])

  useEffect(() => {
    if (!isOpen) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [isOpen, onClose])

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    let list = available
    if (term) {
      list = available.filter(a => a.name.toLowerCase().includes(term))
    }
    // Pinned selected first
    return [...list].sort((a, b) => {
      const aSelected = selectedIds.includes(a.id) ? 0 : 1
      const bSelected = selectedIds.includes(b.id) ? 0 : 1
      return aSelected - bSelected || a.name.localeCompare(b.name)
    })
  }, [available, search, selectedIds])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div
        ref={modalRef}
        className="bg-card rounded-ios shadow-ios w-full max-w-md mx-4 max-h-[80vh] flex flex-col"
      >
        <div className="p-4 border-b border-border">
          <h3 className="text-lg font-bold">Agregar analizador</h3>
          <p className="text-sm text-text-secondary mt-1">
            Buscá y activá los analizadores para esta institución
          </p>
        </div>

        <div className="p-4 border-b border-border">
          <div className="relative">
            <Icon name="search" className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-secondary" />
            <input
              ref={inputRef}
              type="text"
              placeholder="Buscar analizador..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary text-sm min-h-[44px]"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          {filtered.length === 0 ? (
            <p className="text-center text-text-secondary text-sm py-8">No se encontraron analizadores</p>
          ) : (
            <div className="space-y-1">
              {filtered.map((analyzer) => {
                const isSelected = selectedIds.includes(analyzer.id)
                return (
                  <button
                    key={analyzer.id}
                    type="button"
                    onClick={() => onToggle(analyzer.id)}
                    className={`w-full flex items-center justify-between px-3 py-3 rounded-ios-sm transition-colors min-h-[44px] ${
                      isSelected ? 'bg-primary/10' : 'hover:bg-bg'
                    }`}
                  >
                    <span className={`text-sm font-medium ${isSelected ? 'text-primary' : 'text-text'}`}>
                      {analyzer.name}
                    </span>
                    <div
                      className={`relative w-11 h-6 rounded-full transition-colors ${
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
            Listo
          </button>
        </div>
      </div>
    </div>
  )
}
