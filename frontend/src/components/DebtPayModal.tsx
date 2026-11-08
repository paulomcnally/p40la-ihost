import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { Icon } from './Icons'
import PdfUploadField from './PdfUploadField'
import PaymentTimingLabel from './PaymentTimingLabel'
import LoadingSpinner from './LoadingSpinner'
import DebtInformationView from './DebtInformationView'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
import type { Debt, DebtBill, DebtPaymentInformation } from '../types'

type DebtPayTab = 'pay' | 'info'

interface DebtPayModalProps {
  bill: DebtBill
  debt?: Debt | null
  paymentInfo?: DebtPaymentInformation | null
  symbol?: string
  onClose: () => void
  onSuccess: () => void
  onConfigure?: () => void
}

function todayISO(): string {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export default function DebtPayModal({
  bill,
  debt,
  paymentInfo,
  symbol,
  onClose,
  onSuccess,
  onConfigure,
}: DebtPayModalProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<DebtPayTab>('pay')
  const [paidAt, setPaidAt] = useState(todayISO())
  const [paymentReference, setPaymentReference] = useState('')
  const [invoiceFile, setInvoiceFile] = useState<File | null>(null)
  const [receiptFile, setReceiptFile] = useState<File | null>(null)
  const [saving, setSaving] = useState(false)

  // En calendario/análisis el modal solo recibe la cuota; se completan la
  // deuda y la forma de pago bajo demanda (SPEC-131).
  const [loadedDebt, setLoadedDebt] = useState<Debt | null>(debt ?? null)
  const [loadedInfo, setLoadedInfo] = useState<DebtPaymentInformation | null | undefined>(paymentInfo)
  const [infoLoading, setInfoLoading] = useState(!(debt && paymentInfo !== undefined))

  useEffect(() => {
    if (debt && paymentInfo !== undefined) return
    let active = true
    setInfoLoading(true)
    Promise.all([
      debt ? Promise.resolve(debt) : api.debts.get(bill.debt_id),
      paymentInfo !== undefined
        ? Promise.resolve(paymentInfo)
        : api.debtPaymentInformation.get(bill.debt_id).catch(() => null),
    ])
      .then(([debtData, info]) => {
        if (!active) return
        if (!debt) setLoadedDebt(debtData)
        if (paymentInfo === undefined) setLoadedInfo(info)
      })
      .catch(() => undefined)
      .finally(() => {
        if (active) setInfoLoading(false)
      })
    return () => {
      active = false
    }
  }, [bill.debt_id, debt, paymentInfo])

  const effectiveDebt = debt ?? loadedDebt
  const effectiveInfo = paymentInfo !== undefined ? paymentInfo : loadedInfo
  const effectiveSymbol = symbol || effectiveDebt?.currency_code

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

  const handleConfigure = () => {
    if (onConfigure) {
      onConfigure()
      return
    }
    onClose()
    navigate(`/deudas/${bill.debt_id}`)
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-lg max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5 pb-3">
          <div className="flex items-start gap-3">
            <div className="w-12 h-12 rounded-ios bg-success/10 text-success flex items-center justify-center shrink-0">
              <Icon name="credit" className="w-6 h-6" />
            </div>
            <div className="min-w-0">
              <h3 className="text-lg sm:text-xl font-semibold">{t('deudas.pay')}</h3>
              <p className="text-sm text-text-secondary break-words">
                {bill.debt_description} — {t('deudas.installment')} #{bill.installment_number}
              </p>
            </div>
          </div>

          <div className="mt-4 flex gap-1 p-1 bg-bg rounded-ios-sm" role="tablist" aria-label={t('deudas.pay')}>
            <button
              type="button"
              role="tab"
              aria-selected={activeTab === 'pay'}
              disabled={saving}
              onClick={() => setActiveTab('pay')}
              className={`flex-1 px-3 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
                activeTab === 'pay' ? 'bg-card text-text shadow-ios' : 'text-text-secondary'
              }`}
            >
              {t('deudas.pay_tab_pay')}
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={activeTab === 'info'}
              disabled={saving}
              onClick={() => setActiveTab('info')}
              className={`flex-1 px-3 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
                activeTab === 'info' ? 'bg-card text-text shadow-ios' : 'text-text-secondary'
              }`}
            >
              {t('deudas.pay_tab_information')}
            </button>
          </div>
        </div>

        <div className="px-5 pb-5">
          {activeTab === 'pay' ? (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">{t('deudas.pay_date')}</label>
                <input
                  type="date"
                  value={paidAt}
                  max={todayISO()}
                  onChange={(e) => setPaidAt(e.target.value)}
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
                  required
                />
                {paidAt && (
                  <PaymentTimingLabel
                    paidDate={paidAt}
                    dueDate={bill.due_date}
                    mode="paying"
                    className="block text-xs mt-1"
                  />
                )}
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">{t('deudas.payment_reference')}</label>
                <input
                  type="text"
                  value={paymentReference}
                  onChange={(e) => setPaymentReference(e.target.value)}
                  className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
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
          ) : infoLoading ? (
            <div className="py-8">
              <LoadingSpinner />
            </div>
          ) : effectiveInfo?.template && effectiveDebt ? (
            <DebtInformationView
              template={effectiveInfo.template}
              values={effectiveInfo.values || {}}
              debt={effectiveDebt}
              bill={bill}
              symbol={effectiveSymbol}
              paidAt={paidAt}
            />
          ) : (
            <div className="py-6 text-center">
              <p className="text-sm text-text-secondary">{t('deudas.payment_info_empty')}</p>
              <p className="text-sm text-text-secondary mt-1">{t('deudas.payment_info_empty_hint')}</p>
              <button
                type="button"
                onClick={handleConfigure}
                className="mt-4 inline-flex items-center gap-2 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium hover:opacity-90 min-h-[44px]"
              >
                <Icon name="edit" className="w-4 h-4" />
                {t('deudas.payment_info_configure')}
              </button>
            </div>
          )}
        </div>

        <div className="flex gap-3 p-4 border-t border-border">
          {activeTab === 'pay' ? (
            <>
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
            </>
          ) : (
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              <Icon name="check" className="w-4 h-4" />
              {t('app.close')}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
