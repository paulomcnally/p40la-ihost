import { useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'

interface EmailRecipientsModalProps {
  existing: string[]
  onAdd: (email: string) => void | Promise<void>
  onClose: () => void
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export default function EmailRecipientsModal({ existing, onAdd, onClose }: EmailRecipientsModalProps) {
  const { t } = useI18nStore()
  const [value, setValue] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    const email = value.trim()
    if (!email) {
      setError(t('settings.email_alerts.recipients_empty'))
      return
    }
    if (!EMAIL_RE.test(email)) {
      setError(t('settings.email_alerts.recipients_invalid'))
      return
    }
    if (existing.some((e) => e.toLowerCase() === email.toLowerCase())) {
      setError(t('settings.email_alerts.recipients_duplicate'))
      return
    }
    setSubmitting(true)
    try {
      await onAdd(email)
      onClose()
    } catch {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div
        className="bg-card rounded-ios shadow-ios w-full max-w-sm p-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-bold mb-4">{t('settings.email_alerts.add_recipient')}</h3>
        <input
          type="email"
          autoFocus
          value={value}
          onChange={(e) => {
            setValue(e.target.value)
            if (error) setError('')
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleSubmit()
          }}
          placeholder="a@x.com"
          className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
        />
        {error && <p className="text-sm text-red-500 mt-2">{error}</p>}
        <div className="flex justify-end gap-3 mt-4">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
          >
            {t('app.cancel')}
          </button>
          <button
            onClick={handleSubmit}
            disabled={submitting}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
          >
            {submitting ? '...' : t('app.add')}
          </button>
        </div>
      </div>
    </div>
  )
}
