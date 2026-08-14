import { useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'

export default function DeleteModal({ title, subtitle, onConfirm, onCancel }: {
  title: string
  subtitle: string
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useI18nStore()
  const [confirmText, setConfirmText] = useState('')
  const isValid = confirmText.trim().toLowerCase() === 'confirmo'

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      onClick={onCancel}
    >
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5 text-center">
          <div className="w-14 h-14 mx-auto mb-3 text-danger">
            <Icon name="warning" className="w-full h-full" />
          </div>
          <h3 className="text-xl font-semibold mb-1">{title}</h3>
          <p className="text-sm text-text-secondary">{subtitle}</p>
        </div>
        <div className="px-5 pb-4">
          <div>
            <label className="block text-sm font-medium mb-1">Escribe "confirmo" para eliminar</label>
            <input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary"
              placeholder="confirmo"
              autoComplete="off"
            />
          </div>
        </div>
        <div className="flex gap-3 p-4 border-t border-border">
          <button
            onClick={onCancel}
            className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            onClick={onConfirm}
            disabled={!isValid}
            className="flex-1 px-4 py-2.5 bg-danger text-white rounded-ios-sm hover:bg-red-700 disabled:opacity-50 transition-colors flex items-center justify-center gap-2"
          >
            <Icon name="delete" className="w-4 h-4" />
            {t('app.delete')}
          </button>
        </div>
      </div>
    </div>
  )
}
