import { useState, useEffect, useRef } from 'react'
import { Icon } from './Icons'
import { iconCatalog, getIconCategories, filterIcons, type IconDefinition } from '../data/iconCatalog'
import { useI18nStore } from '../stores/i18nStore'

interface IconPickerModalProps {
  isOpen: boolean
  selectedIcon: string
  onSelect: (key: string) => void
  onClose: () => void
}

export default function IconPickerModal({ isOpen, selectedIcon, onSelect, onClose }: IconPickerModalProps) {
  const { t } = useI18nStore()
  const [search, setSearch] = useState('')
  const [activeCategory, setActiveCategory] = useState<string>('')
  const modalRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const categories = getIconCategories()
  const filtered = filterIcons(search, activeCategory || undefined)

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
          <h3 className="text-base sm:text-lg font-bold mb-3">{t('services.select_icon')}</h3>
          <input
            ref={inputRef}
            type="text"
            placeholder={t('services.search_icon')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary text-sm min-h-[44px]"
          />
        </div>

        <div className="flex gap-1 px-4 py-2 border-b border-border overflow-x-auto">
          <button
            onClick={() => setActiveCategory('')}
            className={`px-3 py-1 rounded-full text-xs whitespace-nowrap transition-colors ${
              !activeCategory ? 'bg-primary text-white' : 'bg-bg text-text-secondary hover:bg-border'
            }`}
          >
            {t('services.all')}
          </button>
          {categories.map(cat => (
            <button
              key={cat}
              onClick={() => setActiveCategory(cat === activeCategory ? '' : cat)}
              className={`px-3 py-1 rounded-full text-xs whitespace-nowrap transition-colors ${
                activeCategory === cat ? 'bg-primary text-white' : 'bg-bg text-text-secondary hover:bg-border'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          {filtered.length === 0 ? (
            <p className="text-center text-text-secondary text-sm py-8">{t('services.no_icons_found')}</p>
          ) : (
            <div className="grid grid-cols-4 sm:grid-cols-6 gap-2">
              {filtered.map(icon => (
                <button
                  key={icon.key}
                  type="button"
                  onClick={() => { onSelect(icon.key); onClose() }}
                  title={`${icon.label} (${icon.key})`}
                  className={`w-12 h-12 rounded-ios-sm border-2 flex items-center justify-center transition-colors min-h-[44px] ${
                    selectedIcon === icon.key
                      ? 'border-primary text-primary bg-primary/10'
                      : 'border-border text-text-secondary hover:border-primary/50'
                  }`}
                >
                  <Icon name={icon.key} className="w-5 h-5" />
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="p-3 border-t border-border text-xs text-text-secondary text-center">
          {filtered.length} {t('services.icons_available')}
        </div>
      </div>
    </div>
  )
}
