import { useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import AutoAttachmentsField from './AutoAttachmentsField'
import type { AutoAttachment } from '../types'

export default function AutoAttachmentsModal({
  title,
  attachments,
  attachmentUrl,
  upload,
  remove,
  onClose,
  onChanged,
}: {
  title: string
  attachments: AutoAttachment[]
  attachmentUrl: (attachment: AutoAttachment) => string
  upload: (file: File) => Promise<AutoAttachment | null>
  remove: (attachment: AutoAttachment) => Promise<unknown>
  onClose: () => void
  onChanged?: () => void
}) {
  const { t } = useI18nStore()
  const [items, setItems] = useState<AutoAttachment[]>(attachments)

  const handleUpload = async (files: File[]) => {
    for (const file of files) {
      const attachment = await upload(file)
      if (attachment) setItems((prev) => [...prev, attachment])
    }
    onChanged?.()
  }

  const handleDelete = async (attachment: AutoAttachment) => {
    await remove(attachment)
    setItems((prev) => prev.filter((item) => item.id !== attachment.id))
    onChanged?.()
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
      <div
        className="bg-card rounded-ios shadow-ios w-full max-w-lg max-h-[85vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-border flex items-center justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-lg font-bold truncate">{title}</h3>
            <p className="text-xs text-text-secondary">{t('autos.attachments.title')}</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-2 text-text-secondary hover:text-text rounded-full transition-colors"
            aria-label={t('app.close')}
          >
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          <AutoAttachmentsField
            attachments={items}
            attachmentUrl={attachmentUrl}
            onUpload={handleUpload}
            onDelete={handleDelete}
          />
        </div>

        <div className="p-4 border-t border-border flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
          >
            {t('app.close')}
          </button>
        </div>
      </div>
    </div>
  )
}
