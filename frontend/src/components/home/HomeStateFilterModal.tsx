import { useEffect, useState } from 'react'
import { Icon } from '../Icons'
import HomeStateBadge from './HomeStateBadge'
import { useI18nStore } from '../../stores/i18nStore'
import type { HomeState } from '../../types'

const STATES: HomeState[] = ['listed', 'reserved', 'rented', 'inhabited', 'unused']

interface Props {
  isOpen: boolean
  selected: HomeState[]
  counts: Record<string, number>
  onApply: (states: HomeState[]) => void
  onClose: () => void
}

// HomeStateFilterModal filtra las casas por estado (REQ-018, UX-MODAL).
export default function HomeStateFilterModal({ isOpen, selected, counts, onApply, onClose }: Props) {
  const { t } = useI18nStore()
  const [draft, setDraft] = useState<HomeState[]>(selected)

  useEffect(() => {
    if (isOpen) setDraft(selected)
  }, [isOpen, selected])

  if (!isOpen) return null

  const toggle = (state: HomeState) => {
    setDraft((current) => (
      current.includes(state)
        ? current.filter((item) => item !== state)
        : [...current, state]
    ))
  }

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="home-state-filter-title"
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm overflow-hidden"
      >
        <div className="flex items-start justify-between gap-3 p-4 border-b border-border">
          <div className="min-w-0">
            <h3 id="home-state-filter-title" className="text-lg font-bold">{t('home.filter.title')}</h3>
            <p className="text-sm text-text-secondary mt-0.5">{t('home.filter.subtitle')}</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('app.close')}
            className="w-10 h-10 -mr-1 -mt-1 rounded-full flex items-center justify-center text-text-secondary hover:text-text hover:bg-bg shrink-0"
          >
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        <div className="p-2">
          {STATES.map((state) => {
            const active = draft.includes(state)
            return (
              <button
                key={state}
                type="button"
                aria-pressed={active}
                onClick={() => toggle(state)}
                className={`w-full flex items-center justify-between gap-3 px-3 py-2.5 rounded-ios-sm text-left transition-colors ${
                  active ? 'bg-bg' : 'hover:bg-bg/60'
                }`}
              >
                <span className="flex items-center gap-2 min-w-0">
                  <HomeStateBadge state={state} />
                  {(counts[state] ?? 0) > 0 && (
                    <span className="text-xs text-text-secondary tabular-nums">{counts[state]}</span>
                  )}
                </span>
                <span
                  className={`w-6 h-6 rounded-full border flex items-center justify-center shrink-0 ${
                    active ? 'bg-primary border-primary text-white' : 'border-border text-transparent'
                  }`}
                  aria-hidden="true"
                >
                  <Icon name="check" className="w-4 h-4" />
                </span>
              </button>
            )
          })}
        </div>

        {draft.length === 0 && (
          <p className="px-4 pb-2 text-sm text-danger">{t('home.filter.empty')}</p>
        )}

        <div className="flex items-center justify-between gap-2 p-4 border-t border-border">
          <button
            type="button"
            onClick={() => setDraft(STATES)}
            className="px-3 py-2 text-sm font-medium text-primary hover:bg-primary/5 rounded-ios-sm min-h-[44px]"
          >
            {t('home.filter.all')}
          </button>
          <button
            type="button"
            disabled={draft.length === 0}
            onClick={() => onApply(draft)}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px] font-medium"
          >
            {t('app.done')}
          </button>
        </div>
      </div>
    </div>
  )
}
