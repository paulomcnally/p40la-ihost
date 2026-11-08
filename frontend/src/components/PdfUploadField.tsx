import { useRef, useState, type ChangeEvent, type DragEvent, type KeyboardEvent } from 'react'
import { Icon } from './Icons'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
import { formatFileSize, isPdfFile, MAX_RECEIPT_BYTES } from '../utils/pdfFile'

interface PdfUploadFieldProps {
  label: string
  optionalLabel: string
  hint: string
  formats: string
  file: File | null
  disabled?: boolean
  onSelect: (file: File) => void
  onRemove: () => void
}

export default function PdfUploadField({
  label,
  optionalLabel,
  hint,
  formats,
  file,
  disabled = false,
  onSelect,
  onRemove,
}: PdfUploadFieldProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [dragActive, setDragActive] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const selectFile = (candidate: File | null | undefined) => {
    if (!candidate || disabled) return
    if (!isPdfFile(candidate)) {
      showToast(t('deudas.pdf_invalid_type'), 'error')
      return
    }
    if (candidate.size === 0) {
      showToast(t('deudas.pdf_empty'), 'error')
      return
    }
    if (candidate.size > MAX_RECEIPT_BYTES) {
      showToast(t('deudas.pdf_too_large'), 'error')
      return
    }
    onSelect(candidate)
  }

  const openFilePicker = () => {
    if (!disabled) inputRef.current?.click()
  }

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    selectFile(e.target.files?.[0])
    e.target.value = ''
  }

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
    selectFile(e.dataTransfer.files?.[0])
  }

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    if (!disabled) setDragActive(true)
  }

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      openFilePicker()
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between gap-2 mb-1">
        <label className="block text-sm font-medium">{label}</label>
        <span className="text-xs text-text-secondary">{optionalLabel}</span>
      </div>

      {file ? (
        <div className="flex items-center gap-3 rounded-ios-sm border border-border bg-bg px-3 py-2">
          <Icon name="pdf" className="w-5 h-5 text-primary shrink-0" />
          <div className="min-w-0 flex-1">
            <p className="text-sm truncate">{file.name}</p>
            <p className="text-xs text-text-secondary">{formatFileSize(file.size)}</p>
          </div>
          <button
            type="button"
            onClick={onRemove}
            disabled={disabled}
            className="w-8 h-8 shrink-0 rounded-full flex items-center justify-center text-text-secondary hover:bg-border transition-colors disabled:opacity-50"
            title={t('deudas.remove_file')}
            aria-label={t('deudas.remove_file')}
          >
            <Icon name="cancel" className="w-4 h-4" />
          </button>
        </div>
      ) : (
        <div
          role="button"
          tabIndex={disabled ? -1 : 0}
          onClick={openFilePicker}
          onKeyDown={handleKeyDown}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          className={`w-full rounded-ios-sm border-2 border-dashed px-3 py-4 min-h-[92px] flex flex-col items-center justify-center gap-1 text-center transition-colors ${
            disabled
              ? 'border-border bg-bg opacity-60 cursor-not-allowed'
              : dragActive
                ? 'border-primary bg-primary/5 cursor-pointer'
                : 'border-border bg-bg hover:border-primary/60 cursor-pointer'
          }`}
        >
          <Icon name="upload" className="w-5 h-5 text-text-secondary" />
          <p className="text-sm text-text">{hint}</p>
          <p className="text-xs text-text-secondary">{formats}</p>
        </div>
      )}

      <input
        ref={inputRef}
        type="file"
        accept="application/pdf,.pdf"
        className="hidden"
        disabled={disabled}
        onChange={handleFileChange}
      />
    </div>
  )
}
