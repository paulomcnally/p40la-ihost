import { useState } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import DeleteModal from './DeleteModal'
import { useToast } from './Toast'
import { useI18nStore } from '../stores/i18nStore'

type ContractOwner = 'debts' | 'services'

interface ContractModalProps {
  owner: ContractOwner
  entityId: number
  entityName: string
  hasContract: boolean
  onClose: () => void
  onChanged: () => void
}

const MAX_SIZE = 10 * 1024 * 1024

function formatSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  if (bytes >= 1024) return `${Math.round(bytes / 1024)} KB`
  return `${bytes} B`
}

export default function ContractModal({
  owner,
  entityId,
  entityName,
  hasContract,
  onClose,
  onChanged,
}: ContractModalProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [localHasContract, setLocalHasContract] = useState(hasContract)
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [dragging, setDragging] = useState(false)
  const [saving, setSaving] = useState(false)
  const [confirming, setConfirming] = useState(false)

  const busy = saving || confirming

  const validateFile = (selected: File): string | null => {
    if (!selected.name.toLowerCase().endsWith('.pdf')) return t('contracts.invalid_type')
    if (selected.size === 0) return t('contracts.empty_file')
    if (selected.size > MAX_SIZE) return t('contracts.file_too_large')
    return null
  }

  const pickFile = (selected: File) => {
    const validationError = validateFile(selected)
    if (validationError) {
      setFile(null)
      setError(validationError)
      return
    }
    setFile(selected)
    setError(null)
  }

  const handleFileInput = (event: React.ChangeEvent<HTMLInputElement>) => {
    const selected = event.target.files?.[0]
    if (selected) pickFile(selected)
    event.target.value = ''
  }

  const handleDrop = (event: React.DragEvent) => {
    event.preventDefault()
    setDragging(false)
    const selected = event.dataTransfer.files?.[0]
    if (selected) pickFile(selected)
  }

  const handleUpload = async () => {
    if (!file) return
    setSaving(true)
    setError(null)
    try {
      const result = await api.contracts.upload(owner, entityId, file)
      if (!result?.has_contract) {
        throw new Error(t('contracts.upload_error'))
      }
      setLocalHasContract(true)
      setFile(null)
      showToast(t('contracts.upload_success'), 'success')
      onChanged()
    } catch (err) {
      const message = err instanceof Error ? err.message : t('contracts.upload_error')
      setError(message)
      showToast(message, 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleOpen = () => {
    window.open(api.contracts.url(owner, entityId), '_blank', 'noopener,noreferrer')
  }

  const handleRemove = async () => {
    setSaving(true)
    try {
      await api.contracts.remove(owner, entityId)
      setLocalHasContract(false)
      setFile(null)
      setError(null)
      showToast(t('contracts.remove_success'), 'success')
      onChanged()
    } catch (err) {
      const message = err instanceof Error ? err.message : t('contracts.remove_error')
      showToast(message, 'error')
    } finally {
      setSaving(false)
      setConfirming(false)
    }
  }

  return (
    <>
      <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="contract-modal-title"
          className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm overflow-hidden"
        >
        <div className="p-5">
          <div className="flex items-start justify-between gap-3 mb-4">
            <div className="flex items-center gap-3 min-w-0">
              <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
                <Icon name="pdf" className="w-6 h-6" />
              </div>
              <div className="min-w-0">
                <h3 id="contract-modal-title" className="text-lg font-semibold truncate">
                  {t('contracts.title')}
                </h3>
                <p className="text-sm text-text-secondary truncate">{entityName}</p>
              </div>
            </div>
            <button
              type="button"
              onClick={onClose}
              disabled={busy}
              className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-bg transition-colors disabled:opacity-50"
              aria-label={t('app.close')}
            >
              <Icon name="cancel" className="w-4 h-4 text-text-secondary" />
            </button>
          </div>

          {localHasContract ? (
            <>
              <div className="flex items-center gap-3 rounded-ios-sm border border-border bg-bg px-3 py-3">
                <Icon name="pdf" className="w-6 h-6 text-primary shrink-0" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium truncate">contract.pdf</p>
                  <p className="text-xs text-text-secondary">{t('contracts.document_attached')}</p>
                </div>
              </div>
              <div className="flex gap-3 mt-5">
                <button
                  type="button"
                  onClick={() => setConfirming(true)}
                  disabled={busy}
                  className="flex-1 px-4 py-2.5 bg-danger/10 text-danger rounded-ios-sm hover:bg-danger/20 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px] font-semibold"
                >
                  <Icon name="delete" className="w-4 h-4" />
                  {t('app.remove')}
                </button>
                <button
                  type="button"
                  onClick={handleOpen}
                  disabled={busy}
                  className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px] font-semibold"
                >
                  <Icon name="pdf" className="w-4 h-4" />
                  {t('contracts.open')}
                </button>
              </div>
            </>
          ) : (
            <>
              <p className="text-sm text-text-secondary mb-3">{t('contracts.document_missing')}</p>
              <label
                onDragOver={(event) => {
                  event.preventDefault()
                  setDragging(true)
                }}
                onDragLeave={() => setDragging(false)}
                onDrop={handleDrop}
                className={`block rounded-ios border-2 border-dashed p-6 text-center cursor-pointer transition-colors ${
                  dragging
                    ? 'border-primary bg-primary/5'
                    : 'border-border bg-bg hover:border-primary/60'
                }`}
              >
                <Icon name="upload" className="w-8 h-8 mx-auto text-primary mb-2" />
                <p className="text-sm font-medium">{t('contracts.drop_hint')}</p>
                <p className="text-xs text-text-secondary mt-1">
                  {t('contracts.only_pdf')} · {t('contracts.max_size')}
                </p>
                <input
                  type="file"
                  accept="application/pdf,.pdf"
                  className="hidden"
                  onChange={handleFileInput}
                />
              </label>

              {file && (
                <div className="mt-3 flex items-center gap-2 rounded-ios-sm bg-bg px-3 py-2">
                  <Icon name="pdf" className="w-5 h-5 text-primary shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm truncate">{file.name}</p>
                    <p className="text-xs text-text-secondary">{formatSize(file.size)}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setFile(null)
                      setError(null)
                    }}
                    disabled={saving}
                    className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-border transition-colors disabled:opacity-50"
                    aria-label={t('app.remove')}
                  >
                    <Icon name="cancel" className="w-4 h-4 text-text-secondary" />
                  </button>
                </div>
              )}

              {error && <p className="text-sm text-danger mt-2">{error}</p>}

              <div className="flex gap-3 mt-5">
                <button
                  type="button"
                  onClick={onClose}
                  disabled={saving}
                  className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
                >
                  <Icon name="cancel" className="w-4 h-4" />
                  {t('app.cancel')}
                </button>
                <button
                  type="button"
                  onClick={handleUpload}
                  disabled={saving || !file}
                  className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px] font-semibold"
                >
                  <Icon name="upload" className="w-4 h-4" />
                  {saving ? t('contracts.uploading') : t('contracts.upload')}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>

    {confirming && (
      <DeleteModal
        title={t('contracts.remove_title')}
        subtitle={t('contracts.remove_subtitle')}
        onConfirm={handleRemove}
        onCancel={() => setConfirming(false)}
      />
    )}
    </>
  )
}
