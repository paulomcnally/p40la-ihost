import { useRef, useState } from 'react'
import { Icon } from './Icons'
import { useI18nStore } from '../stores/i18nStore'
import type { AutoAttachment } from '../types'

const MAX_AUTO_ATTACHMENT_BYTES = 10 * 1024 * 1024
const DEFAULT_MAX_ATTACHMENTS = 20

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function AutoAttachmentsField({
  attachments,
  attachmentUrl,
  onUpload,
  onDelete,
  max = DEFAULT_MAX_ATTACHMENTS,
}: {
  attachments: AutoAttachment[]
  attachmentUrl: (attachment: AutoAttachment) => string
  onUpload?: (files: File[]) => Promise<void>
  onDelete?: (attachment: AutoAttachment) => Promise<void>
  max?: number
}) {
  const { t } = useI18nStore()
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const readOnly = !onUpload

  const handleFiles = async (files: FileList | File[] | null) => {
    if (!files || !onUpload) return
    setError('')
    const selected = Array.from(files)
    const remaining = max - attachments.length
    if (remaining <= 0) {
      setError(t('autos.attachments.limit'))
      return
    }
    const accepted: File[] = []
    for (const file of selected.slice(0, remaining)) {
      const isPdf = file.type === 'application/pdf' || /\.pdf$/i.test(file.name)
      if (!isPdf) {
        setError(t('autos.attachments.pdf_only'))
        continue
      }
      if (file.size === 0) {
        setError(t('autos.attachments.empty'))
        continue
      }
      if (file.size > MAX_AUTO_ATTACHMENT_BYTES) {
        setError(t('autos.attachments.too_large'))
        continue
      }
      accepted.push(file)
    }
    if (selected.length > remaining) {
      setError(t('autos.attachments.limit'))
    }
    if (accepted.length === 0) return

    setUploading(true)
    try {
      await onUpload(accepted)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('autos.attachments.upload_error'))
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  const handleDelete = async (attachment: AutoAttachment) => {
    if (!onDelete) return
    setError('')
    try {
      await onDelete(attachment)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('autos.attachments.delete_error'))
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between gap-2 mb-1">
        <label className="block text-sm font-medium">{t('autos.attachments.title')}</label>
        <span className="text-xs text-text-secondary">{attachments.length}/{max}</span>
      </div>

      {attachments.length > 0 && (
        <div className="space-y-2 mb-2">
          {attachments.map((attachment) => (
            <div key={attachment.id} className="flex items-center gap-2 bg-bg border border-border rounded-ios-sm px-3 py-2">
              <Icon name="pdf" className="w-4 h-4 text-danger flex-shrink-0" />
              <a
                href={attachmentUrl(attachment)}
                target="_blank"
                rel="noopener noreferrer"
                className="flex-1 min-w-0 text-sm truncate hover:text-primary transition-colors"
              >
                {attachment.file_name}
              </a>
              <span className="text-xs text-text-secondary flex-shrink-0">{formatBytes(attachment.size_bytes)}</span>
              {!readOnly && (
                <button
                  type="button"
                  onClick={() => handleDelete(attachment)}
                  className="p-1.5 text-text-secondary hover:text-danger transition-colors rounded-full"
                  aria-label={t('app.delete')}
                >
                  <Icon name="delete" className="w-4 h-4" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {!readOnly && (
        <>
          <input
            ref={inputRef}
            type="file"
            accept="application/pdf,.pdf"
            multiple
            className="hidden"
            onChange={(e) => handleFiles(e.target.files)}
          />
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            disabled={uploading || attachments.length >= max}
            className="inline-flex items-center gap-2 px-3 py-2 bg-bg border border-border rounded-ios-sm text-sm font-medium hover:border-primary/50 disabled:opacity-50 transition-colors min-h-[44px]"
          >
            <Icon name="upload" className="w-4 h-4" />
            {uploading ? t('autos.attachments.uploading') : t('autos.attachments.attach')}
          </button>
          <p className="text-xs text-text-secondary mt-1">{t('autos.attachments.hint')}</p>
        </>
      )}

      {error && <p className="text-sm text-danger mt-1">{error}</p>}
    </div>
  )
}
