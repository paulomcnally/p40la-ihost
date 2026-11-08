import { useRef, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import { useConfirmWordInput } from '../hooks/useConfirmWordInput'

export default function DeleteModal({ title, subtitle, onConfirm, onCancel }: {
  title: string
  subtitle: string
  onConfirm: () => void | Promise<void>
  onCancel: () => void
}) {
  const { t } = useI18nStore()
  const [submitting, setSubmitting] = useState(false)
  const submittingRef = useRef(false)

  const handleConfirm = async () => {
    // Guard síncrono: Enter/click repetidos antes del re-render de `submitting`.
    if (submittingRef.current) return
    submittingRef.current = true
    setSubmitting(true)
    try {
      await onConfirm()
    } finally {
      submittingRef.current = false
      setSubmitting(false)
    }
  }

  const { value, setValue, inputRef, isValid, handleKeyDown, reset } = useConfirmWordInput({
    word: 'confirmo',
    onConfirm: handleConfirm,
    disabled: submitting,
  })

  const handleCancel = () => {
    reset()
    onCancel()
  }

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-[70] p-4"
    >
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5 text-center">
          <div className="w-14 h-14 mx-auto mb-3 text-danger">
            <Icon name="warning" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-1">{title}</h3>
          <p className="text-sm text-text-secondary">{subtitle}</p>
        </div>
        <div className="px-5 pb-4">
          <div>
            <label className="block text-sm font-medium mb-1">Escribe "confirmo" para eliminar</label>
            <input
              ref={inputRef}
              type="text"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              onKeyDown={handleKeyDown}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              placeholder="confirmo"
              autoComplete="off"
            />
          </div>
        </div>
        <div className="flex gap-3 p-4 border-t border-border">
          <button
            onClick={handleCancel}
            className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={!isValid || submitting}
            className="flex-1 px-4 py-2.5 bg-danger text-white rounded-ios-sm hover:bg-red-700 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            <Icon name="delete" className="w-4 h-4" />
            {t('app.delete')}
          </button>
        </div>
      </div>
    </div>
  )
}
