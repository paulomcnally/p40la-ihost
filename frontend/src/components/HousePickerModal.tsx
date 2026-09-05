import { useState, useEffect, useRef } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import type { Home } from '../types'

interface HousePickerModalProps {
  isOpen: boolean
  homes: Home[]
  selectedHomeId: number | null
  onSelect: (id: number | null) => void
  onClose: () => void
}

export default function HousePickerModal({ isOpen, homes, selectedHomeId, onSelect, onClose }: HousePickerModalProps) {
  const { t } = useI18nStore()
  const [search, setSearch] = useState('')
  const modalRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const filtered = homes.filter(h => h.name.toLowerCase().includes(search.toLowerCase()))

  useEffect(() => {
    if (isOpen) setSearch('')
  }, [isOpen])

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

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div ref={modalRef} className="bg-card rounded-ios shadow-ios w-full max-w-sm sm:max-w-lg mx-4 max-h-[80vh] flex flex-col">
        <div className="p-4 border-b border-border">
          <h3 className="text-base sm:text-lg font-bold mb-3">{t('services.homes')}</h3>
          <input
            ref={inputRef}
            type="text"
            placeholder={t('services.search_home')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary text-sm min-h-[44px]"
          />
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          <button
            type="button"
            onClick={() => { onSelect(null); onClose() }}
            className={`w-full text-left px-3 py-2.5 rounded-ios-sm flex items-center gap-3 transition-colors ${
              selectedHomeId === null ? 'bg-primary/5 text-primary font-medium' : 'hover:bg-bg'
            }`}
          >
            <Icon name="home" className="w-5 h-5 text-text-secondary shrink-0" />
            <span className="text-sm">{t('services.all_homes')}</span>
          </button>

          {filtered.length === 0 ? (
            <p className="text-center text-text-secondary text-sm py-8">{t('services.no_homes_found')}</p>
          ) : (
            filtered.map(home => (
              <button
                key={home.id}
                type="button"
                onClick={() => { onSelect(home.id); onClose() }}
                className={`w-full text-left px-3 py-2.5 rounded-ios-sm flex items-center gap-3 transition-colors ${
                  selectedHomeId === home.id ? 'bg-primary/5 text-primary font-medium' : 'hover:bg-bg'
                }`}
              >
                <Icon name="home" className="w-5 h-5 text-text-secondary shrink-0" />
                <span className="text-sm">{home.name}</span>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  )
}