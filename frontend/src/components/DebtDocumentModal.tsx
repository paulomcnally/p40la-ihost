import { useState } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import PdfUploadField from './PdfUploadField'
import DeleteModal from './DeleteModal'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
import type { DebtBill } from '../types'

interface DebtDocumentModalProps {
  bill: DebtBill
  kind: 'invoice' | 'receipt'
  onClose: () => void
  onChanged: () => void
}

export default function DebtDocumentModal({ bill, kind, onClose, onChanged }: DebtDocumentModalProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [file, setFile] = useState<File | null>(null)
  const [saving, setSaving] = useState(false)
  const [confirmRemove, setConfirmRemove] = useState(false)

  const isInvoice = kind === 'invoice'
  const hasDocument = isInvoice ? Boolean(bill.has_invoice) : Boolean(bill.has_receipt)
  const title = isInvoice ? t('deudas.invoice') : t('deudas.receipt')
  const url = isInvoice ? api.debts.invoiceUrl(bill.id) : api.debts.receiptUrl(bill.id)
  const fileName = `${kind}_${bill.installment_number}.pdf`

  const handleOpen = () => {
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const handleUpload = async () => {
    if (!file) {
      showToast(t('deudas.pdf_required'), 'error')
      return
    }
    setSaving(true)
    try {
      if (isInvoice) {
        await api.debts.uploadInvoice(bill.id, file)
      } else {
        await api.debts.uploadReceipt(bill.id, file)
      }
      showToast(t(isInvoice ? 'deudas.invoice_uploaded' : 'deudas.receipt_uploaded'), 'success')
      onChanged()
      onClose()
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
      showToast(message, 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleRemove = async () => {
    if (saving) return
    setSaving(true)
    try {
      if (isInvoice) {
        await api.debts.removeInvoice(bill.id)
      } else {
        await api.debts.removeReceipt(bill.id)
      }
      showToast(t(isInvoice ? 'deudas.invoice_removed' : 'deudas.receipt_removed'), 'success')
      setConfirmRemove(false)
      onChanged()
      onClose()
    } catch (err: unknown) {
      const message = (err as { message?: string })?.message || t('errors.generic')
      showToast(message, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <div
        className="fixed inset-0 bg-black/40 flex items-center justify-center z-40 p-4"
        onClick={onClose}
      >
        <div
          className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm overflow-hidden"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="p-5">
            <div className="flex items-center justify-between gap-3 mb-4">
              <div className="flex items-center gap-2 min-w-0">
                <Icon name={isInvoice ? 'bill' : 'pdf'} className="w-5 h-5 text-primary shrink-0" />
                <h3 className="text-lg sm:text-xl font-semibold truncate">{title}</h3>
              </div>
              <button
                onClick={onClose}
                disabled={saving}
                className="w-8 h-8 rounded-full flex items-center justify-center text-text-secondary hover:bg-bg transition-colors disabled:opacity-50"
                title={t('app.close')}
                aria-label={t('app.close')}
              >
                <Icon name="cancel" className="w-4 h-4" />
              </button>
            </div>

            {hasDocument ? (
              <>
                <div className="flex items-center gap-3 rounded-ios-sm border border-border bg-bg px-3 py-3">
                  <Icon name="pdf" className="w-6 h-6 text-primary shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium truncate">{fileName}</p>
                    <p className="text-xs text-text-secondary">{t('deudas.document_attached')}</p>
                  </div>
                </div>
                <div className="flex gap-3 mt-5">
                  <button
                    onClick={() => setConfirmRemove(true)}
                    disabled={saving}
                    className="flex-1 px-4 py-2.5 bg-danger/10 text-danger rounded-ios-sm hover:bg-danger/20 transition-colors flex items-center justify-center gap-2 min-h-[44px] disabled:opacity-50"
                  >
                    <Icon name="delete" className="w-4 h-4" />
                    {t('app.remove')}
                  </button>
                  <button
                    onClick={handleOpen}
                    disabled={saving}
                    className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary/90 transition-colors flex items-center justify-center gap-2 min-h-[44px] disabled:opacity-50"
                  >
                    <Icon name="pdf" className="w-4 h-4" />
                    {t('deudas.open')}
                  </button>
                </div>
              </>
            ) : (
              <>
                <p className="text-sm text-text-secondary mb-3">{t('deudas.document_missing')}</p>
                <PdfUploadField
                  label={t('deudas.upload_document')}
                  optionalLabel=""
                  hint={isInvoice ? t('deudas.upload_invoice_hint') : t('deudas.upload_receipt_hint')}
                  formats={t('deudas.pdf_formats')}
                  file={file}
                  disabled={saving}
                  onSelect={setFile}
                  onRemove={() => setFile(null)}
                />
                <div className="flex gap-3 mt-5">
                  <button
                    onClick={onClose}
                    disabled={saving}
                    className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
                  >
                    <Icon name="cancel" className="w-4 h-4" />
                    {t('app.cancel')}
                  </button>
                  <button
                    onClick={handleUpload}
                    disabled={saving || !file}
                    className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
                  >
                    {saving ? (
                      <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
                    ) : (
                      <Icon name="upload" className="w-4 h-4" />
                    )}
                    {saving ? t('app.loading') : t('deudas.upload')}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {confirmRemove && (
        <DeleteModal
          title={t(isInvoice ? 'deudas.remove_invoice' : 'deudas.remove_receipt')}
          subtitle={t(isInvoice ? 'deudas.remove_invoice_confirm' : 'deudas.remove_receipt_confirm')}
          onConfirm={handleRemove}
          onCancel={() => setConfirmRemove(false)}
        />
      )}
    </>
  )
}
