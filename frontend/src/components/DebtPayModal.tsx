import { useRef, useState, type ChangeEvent, type DragEvent, type KeyboardEvent } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
import { formatFileSize, isPdfFile, MAX_RECEIPT_BYTES } from '../utils/receiptFile'
import type { DebtBill } from '../types'

interface DebtPayModalProps {
  bill: DebtBill
  onClose: () => void
  onSuccess: () => void
}

function todayISO(): string {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export default function DebtPayModal({ bill, onClose, onSuccess }: DebtPayModalProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [paidAt, setPaidAt] = useState(todayISO())
  const [paymentReference, setPaymentReference] = useState('')
  const [receiptFile, setReceiptFile] = useState<File | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const [saving, setSaving] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const selectReceipt = (file: File | null | undefined) => {
    if (!file) return
    if (!isPdfFile(file)) {
      showToast(t('deudas.receipt_invalid_type'), 'error')
      return
    }
    if (file.size === 0) {
      showToast(t('deudas.receipt_empty'), 'error')
      return
    }
    if (file.size > MAX_RECEIPT_BYTES) {
      showToast(t('deudas.receipt_too_large'), 'error')
      return
    }
    setReceiptFile(file)
  }

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    selectReceipt(e.target.files?.[0])
    e.target.value = ''
  }

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
    selectReceipt(e.dataTransfer.files?.[0])
  }

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(true)
  }

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
  }

  const openFilePicker = () => fileInputRef.current?.click()

  const handleDropzoneKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      openFilePicker()
    }
  }

  const handleConfirm = async () => {
    if (!paidAt) {
      showToast(t('deudas.pay_date'), 'error')
      return
    }
    setSaving(true)
    try {
      await api.debts.payBill(
        bill.id,
        {
          paid_at: paidAt,
          payment_reference: paymentReference.trim() || undefined,
        },
        receiptFile || undefined,
      )
      showToast(t('deudas.pay_success'), 'success')
      onSuccess()
      onClose()
    } catch (err: unknown) {
      const e = err as { message?: string; code?: string }
      const message = e?.message || t('errors.generic')
      showToast(message, 'error')
      // La cuota quedó pagada pero el comprobante no se pudo guardar: se
      // refresca la deuda para reflejar el pago y se permite reintentar luego.
      if (e?.code === 'receipt_save_failed') {
        onSuccess()
        onClose()
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-sm max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5">
          <div className="w-14 h-14 mx-auto mb-3 text-success">
            <Icon name="credit" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-1 text-center">{t('deudas.pay')}</h3>
          <p className="text-sm text-text-secondary text-center mb-5 break-words">
            {bill.debt_description} — {t('deudas.installment')} #{bill.installment_number}
          </p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('deudas.pay_date')}</label>
              <input
                type="date"
                value={paidAt}
                max={todayISO()}
                onChange={(e) => setPaidAt(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('deudas.payment_reference')}</label>
              <input
                type="text"
                value={paymentReference}
                onChange={(e) => setPaymentReference(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card"
                autoComplete="off"
              />
            </div>

            <div>
              <div className="flex items-center justify-between gap-2 mb-1">
                <label className="block text-sm font-medium">{t('deudas.receipt')}</label>
                <span className="text-xs text-text-secondary">{t('deudas.receipt_optional')}</span>
              </div>

              {receiptFile ? (
                <div className="flex items-center gap-3 rounded-ios-sm border border-border bg-bg px-3 py-2">
                  <Icon name="pdf" className="w-5 h-5 text-primary shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm truncate">{receiptFile.name}</p>
                    <p className="text-xs text-text-secondary">{formatFileSize(receiptFile.size)}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setReceiptFile(null)}
                    disabled={saving}
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
                  tabIndex={0}
                  onClick={openFilePicker}
                  onKeyDown={handleDropzoneKeyDown}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                  className={`w-full rounded-ios-sm border-2 border-dashed px-3 py-4 min-h-[92px] flex flex-col items-center justify-center gap-1 text-center cursor-pointer transition-colors ${
                    dragActive
                      ? 'border-primary bg-primary/5'
                      : 'border-border bg-bg hover:border-primary/60'
                  }`}
                >
                  <Icon name="upload" className="w-5 h-5 text-text-secondary" />
                  <p className="text-sm text-text">{t('deudas.upload_receipt_hint')}</p>
                  <p className="text-xs text-text-secondary">{t('deudas.upload_receipt_formats')}</p>
                </div>
              )}

              <input
                ref={fileInputRef}
                type="file"
                accept="application/pdf,.pdf"
                className="hidden"
                onChange={handleFileChange}
              />
            </div>
          </div>
        </div>
        <div className="flex gap-3 p-4 border-t border-border">
          <button
            onClick={onClose}
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-success text-white rounded-ios-sm hover:bg-green-600 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            {saving ? (
              <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
            ) : (
              <Icon name="credit" className="w-4 h-4" />
            )}
            {saving ? t('app.loading') : t('deudas.pay')}
          </button>
        </div>
      </div>
    </div>
  )
}
