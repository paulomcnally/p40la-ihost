import { useRef, useState } from 'react'
import { api } from '../../api'
import { useI18nStore } from '../../stores/i18nStore'
import { useToast } from '../../components/Toast'
import { Icon } from '../../components/Icons'
import DeleteModal from '../../components/DeleteModal'
import type {
  HomeWorkRequestAttachment,
  HomeWorkRequestMediaType,
} from '../../types/homeWorkRequest'

const MAX_ATTACHMENTS = 10

function mediaTypeFor(file: File): HomeWorkRequestMediaType | null {
  if (file.type.startsWith('video/')) return 'video'
  if (file.type.startsWith('image/')) return 'image'
  const extension = file.name.toLowerCase().split('.').pop() ?? ''
  if (['mp4', 'webm', 'mov'].includes(extension)) return 'video'
  if (['jpg', 'jpeg', 'png', 'webp', 'heic'].includes(extension)) return 'image'
  return null
}

export default function HomeWorkRequestAttachmentField({ homeId, workRequestId, attachments, onChange }: {
  homeId: number
  workRequestId: number
  attachments: HomeWorkRequestAttachment[]
  onChange: () => void
}) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<HomeWorkRequestAttachment | null>(null)

  const atLimit = attachments.length >= MAX_ATTACHMENTS

  const handleFile = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    const mediaType = mediaTypeFor(file)
    if (!mediaType) {
      showToast(t('homeWorkRequest.invalid_file'), 'error')
      return
    }
    setUploading(true)
    try {
      await api.homeWorkRequest.uploadAttachment(homeId, workRequestId, file, mediaType)
      showToast(t('homeWorkRequest.attachment_added_ok'), 'success')
      onChange()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setUploading(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.homeWorkRequest.deleteAttachment(homeId, workRequestId, deleteTarget.id)
      showToast(t('homeWorkRequest.attachment_deleted_ok'), 'success')
      setDeleteTarget(null)
      onChange()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-text-secondary">{t('homeWorkRequest.attachment_hint')}</p>
        <button
          type="button"
          disabled={uploading || atLimit}
          onClick={() => inputRef.current?.click()}
          className="inline-flex items-center gap-2 px-3 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px] disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Icon name="upload" className="w-4 h-4" />
          {uploading ? t('homeWorkRequest.uploading') : t('homeWorkRequest.add_attachment')}
        </button>
        <input
          ref={inputRef}
          type="file"
          className="hidden"
          accept="image/jpeg,image/png,image/webp,image/heic,video/mp4,video/webm,video/quicktime,.jpg,.jpeg,.png,.webp,.heic,.mp4,.webm,.mov"
          onChange={handleFile}
        />
      </div>
      {atLimit && (
        <p className="text-xs text-warning">{t('homeWorkRequest.attachment_limit')}</p>
      )}

      {attachments.length === 0 ? (
        <div className="border-2 border-dashed border-border rounded-ios p-6 text-center">
          <Icon name="camera" className="w-8 h-8 text-text-secondary mx-auto" />
          <p className="text-sm text-text-secondary mt-2">{t('homeWorkRequest.attachment_empty')}</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {attachments.map((attachment) => {
            const url = api.homeWorkRequest.attachmentUrl(homeId, workRequestId, attachment.id)
            return (
              <div key={attachment.id} className="relative bg-bg rounded-ios-sm border border-border overflow-hidden">
                <button
                  type="button"
                  onClick={() => setDeleteTarget(attachment)}
                  className="absolute top-1.5 right-1.5 z-10 w-8 h-8 rounded-full bg-black/60 text-white flex items-center justify-center hover:bg-danger transition-colors"
                  aria-label={t('app.delete')}
                >
                  <Icon name="delete" className="w-4 h-4" />
                </button>
                {attachment.media_type === 'image' ? (
                  <a href={url} target="_blank" rel="noreferrer">
                    <img src={url} alt={attachment.file_name} className="w-full h-32 object-cover" />
                  </a>
                ) : (
                  <video src={url} controls preload="metadata" className="w-full h-32 bg-black" />
                )}
                <p className="text-xs text-text-secondary truncate px-2 py-1.5">{attachment.file_name}</p>
              </div>
            )
          })}
        </div>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('homeWorkRequest.delete_attachment_title')}
          subtitle={t('homeWorkRequest.delete_attachment_subtitle').replace('{name}', deleteTarget.file_name)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
