import { useState } from 'react'
import { api } from '../api'
import { Icon } from './Icons'
import PdfUploadField from './PdfUploadField'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
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
  const [invoiceFile, setInvoiceFile] = useState<File | null>(null)
  const [receiptFile, setReceiptFile] = useState<File | null>(null)
  const [saving, setSaving] = useState(false)

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
        invoiceFile || undefined,
        receiptFile || undefined,
      )
      showToast(t('deudas.pay_success'), 'success')
      onSuccess()
      onClose()
    } catch (err: unknown) {
      const e = err as { message?: string; code?: string }
      const message = e?.message || t('errors.generic')
      showToast(message, 'error')
      // La cuota quedó pagada pero algún documento no se pudo guardar: se
      // refresca la deuda para reflejar el pago y se permite reintentar luego.
      if (e?.code === 'document_save_failed') {
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

            <PdfUploadField
              label={t('deudas.invoice')}
              optionalLabel={t('deudas.optional')}
              hint={t('deudas.upload_invoice_hint')}
              formats={t('deudas.pdf_formats')}
              file={invoiceFile}
              disabled={saving}
              onSelect={setInvoiceFile}
              onRemove={() => setInvoiceFile(null)}
            />

            <PdfUploadField
              label={t('deudas.receipt')}
              optionalLabel={t('deudas.optional')}
              hint={t('deudas.upload_receipt_hint')}
              formats={t('deudas.pdf_formats')}
              file={receiptFile}
              disabled={saving}
              onSelect={setReceiptFile}
              onRemove={() => setReceiptFile(null)}
            />
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
