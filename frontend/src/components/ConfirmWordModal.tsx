import { useRef, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import { useConfirmWordInput } from '../hooks/useConfirmWordInput'

interface ConfirmWordModalProps {
  title: string
  subtitle: string
  confirmLabel: string
  onConfirm: () => void | Promise<void>
  onCancel: () => void
}

// SPEC-126: modal de confirmación por palabra reutilizable. Enfoca el input
// al abrirse y permite confirmar con Enter cuando la palabra es válida.
export default function ConfirmWordModal({
  title,
  subtitle,
  confirmLabel,
  onConfirm,
  onCancel,
}: ConfirmWordModalProps) {
  const { t } = useI18nStore()
  const [submitting, setSubmitting] = useState(false)
  const submittingRef = useRef(false)

  const handleConfirm = async () => {
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
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[80] p-4">
      <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm overflow-hidden">
        <div className="p-5 text-center">
          <div className="w-14 h-14 mx-auto mb-3 text-warning">
            <Icon name="warning" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-1">{title}</h3>
          <p className="text-sm text-text-secondary">{subtitle}</p>
        </div>
        <div className="px-5 pb-4">
          <label className="block text-sm font-medium mb-1">
            {t('documents.confirm_word_label')}
          </label>
          <input
            ref={inputRef}
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={handleKeyDown}
            className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
            placeholder="confirmo"
            autoComplete="off"
          />
        </div>
        <div className="flex gap-3 p-4 border-t border-border">
          <button
            onClick={handleCancel}
            disabled={submitting}
            className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px] disabled:opacity-50"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={!isValid || submitting}
            className="flex-1 px-4 py-2.5 bg-danger text-white rounded-ios-sm hover:bg-red-700 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            <Icon name="warning" className="w-4 h-4" />
            {submitting ? t('app.loading') : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
