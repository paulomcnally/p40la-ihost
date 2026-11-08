import { useMemo, useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { Icon } from './Icons'
import PdfUploadField from './PdfUploadField'
import PaymentTimingLabel from './PaymentTimingLabel'
import Select from './Select'
import ConfirmWordModal from './ConfirmWordModal'
import { useToast } from './Toast'
import type { Debt, DebtBill, DocumentRecognition, RecognitionCandidate } from '../types'

interface RecognizeDocumentModalProps {
  debt: Debt
  bills: DebtBill[]
  onClose: () => void
  onChanged: () => void
}

function parseAmountInput(value: string): number {
  const trimmed = value.trim()
  if (!trimmed) return 0
  const normalized = trimmed.includes(',') && !trimmed.includes('.')
    ? trimmed.replace(',', '.')
    : trimmed.replace(/,/g, '')
  return Number(normalized)
}

function recognitionErrorMessage(err: unknown, t: (key: string, fallback?: string) => string): string {
  const code = (err as { code?: string })?.code
  switch (code) {
    case 'unsupported_format':
      return t('documents.error_unsupported_format')
    case 'needs_ocr':
      return t('documents.error_needs_ocr')
    case 'invalid_file':
      return t('documents.error_invalid_file')
    case 'file_exists':
      return t('documents.error_file_exists')
    case 'reference_conflict':
      return t('documents.error_reference_conflict')
    case 'reference_in_use':
      return t('documents.error_reference_in_use')
    case 'not_paid':
      return t('documents.error_not_paid')
    case 'already_associated':
      return t('documents.error_already_associated')
    default:
      return (err as { message?: string })?.message || t('documents.error_generic')
  }
}

export default function RecognizeDocumentModal({
  debt,
  bills,
  onClose,
  onChanged,
}: RecognizeDocumentModalProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { currencies } = useAppStore()
  const formatMoney = useCurrencyFormatStore((state) => state.formatMoney)

  const [file, setFile] = useState<File | null>(null)
  const [recognition, setRecognition] = useState<DocumentRecognition | null>(null)
  const [analyzing, setAnalyzing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [reference, setReference] = useState('')
  const [paidAt, setPaidAt] = useState('')
  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState('')
  const [concept, setConcept] = useState('')
  const [targetBillId, setTargetBillId] = useState('')
  const [pendingOverwrite, setPendingOverwrite] = useState(false)

  const paidBills = useMemo(() => bills.filter((bill) => bill.status === 'paid'), [bills])
  const selectedBill = paidBills.find((bill) => String(bill.id) === targetBillId) || null
  const currencySymbol = debt.currency_code || ''

  const currencyOptions = useMemo(() => {
    const codes = new Set<string>(['USD', 'NIO'])
    for (const item of currencies) codes.add(item.code)
    if (debt.currency_code) codes.add(debt.currency_code)
    if (currency) codes.add(currency)
    return Array.from(codes).sort().map((code) => ({ value: code, label: code }))
  }, [currencies, debt.currency_code, currency])

  const billOptions = useMemo(
    () => paidBills.map((bill) => ({
      value: String(bill.id),
      label: `#${bill.installment_number} · ${bill.due_date} · ${formatMoney(bill.amount, currencySymbol)}${bill.payment_reference ? ` · ${bill.payment_reference}` : ''}`,
    })),
    [paidBills, currencySymbol, formatMoney],
  )

  const localCandidates = useMemo(
    () => (recognition?.candidates || []).filter((candidate) => candidate.debt_id === debt.id),
    [recognition, debt.id],
  )
  const foreignCandidates = useMemo(
    () => (recognition?.candidates || []).filter((candidate) => candidate.debt_id !== debt.id),
    [recognition, debt.id],
  )

  const existingReference = selectedBill?.payment_reference?.trim() || ''
  const referenceConflict = Boolean(existingReference && existingReference !== reference.trim())
  const fileConflict = Boolean(selectedBill?.has_receipt)

  const canConfirm = recognition?.status === 'recognized'
  const confidencePercent = Math.round(Number(recognition?.confidence || 0) * 100)
  const confidenceColor = confidencePercent >= 80
    ? 'bg-success'
    : confidencePercent >= 50
      ? 'bg-warning'
      : 'bg-danger'

  const warningLabel = (warning: string) => t(`documents.warning_${warning}`, warning)
  const reasonLabel = (reason: string) => t(`documents.reason_${reason}`, reason)
  const documentTypeLabel = recognition?.document_type === 'nota_debito'
    ? t('documents.type_debit')
    : recognition?.document_type === 'confirmacion_transferencia'
      ? t('documents.type_transfer')
      : recognition?.document_type || ''

  const handleSelectFile = (candidate: File | null) => {
    setFile(candidate)
    setRecognition(null)
    setPendingOverwrite(false)
  }

  const handleRecognize = async () => {
    if (!file) {
      showToast(t('documents.pdf_required'), 'error')
      return
    }
    setAnalyzing(true)
    try {
      const result = await api.documents.recognize(file)
      if (!result) return
      setRecognition(result)
      setReference(result.reference || '')
      setPaidAt(result.paid_at ? result.paid_at.slice(0, 10) : '')
      setAmount(result.amount ? String(result.amount) : '')
      setCurrency(result.currency || debt.currency_code || 'USD')
      setConcept(result.concept || '')

      const candidates = result.candidates || []
      const local = candidates.filter((candidate) => candidate.debt_id === debt.id && candidate.status === 'paid')
      const preferred = local.find((candidate) => candidate.installment_number === result.installment_number) || local[0]
      const target = preferred
        ? paidBills.find((bill) => bill.id === preferred.debt_bill_id)
        : paidBills.find((bill) => bill.installment_number === result.installment_number) || paidBills[0]
      setTargetBillId(target ? String(target.id) : '')

      if (result.duplicate) showToast(t('documents.duplicate'), 'info')
    } catch (err) {
      showToast(recognitionErrorMessage(err, t), 'error')
    } finally {
      setAnalyzing(false)
    }
  }

  const executeConfirm = async (overwriteReference: boolean, overwriteFile: boolean) => {
    if (!recognition || !selectedBill) return
    const parsedAmount = parseAmountInput(amount)
    if (!Number.isFinite(parsedAmount) || parsedAmount < 0) {
      showToast(t('documents.amount_invalid'), 'error')
      return
    }

    setSaving(true)
    try {
      await api.documents.confirm(recognition.recognition_id, {
        entity_type: 'debt_bill',
        debt_id: debt.id,
        debt_bill_id: selectedBill.id,
        installment_number: selectedBill.installment_number,
        reference: reference.trim(),
        paid_at: paidAt || undefined,
        amount: parsedAmount,
        currency: currency || undefined,
        concept: concept.trim() || undefined,
        overwrite_reference: overwriteReference,
        overwrite_file: overwriteFile,
      })
      showToast(t('documents.success'), 'success')
      onChanged()
      onClose()
    } catch (err) {
      showToast(recognitionErrorMessage(err, t), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleConfirmClick = () => {
    if (!recognition || !canConfirm) return
    if (!reference.trim()) {
      showToast(t('documents.reference_required'), 'error')
      return
    }
    if (!selectedBill) {
      showToast(t('documents.target_required'), 'error')
      return
    }
    if ((referenceConflict || fileConflict) && !pendingOverwrite) {
      setPendingOverwrite(true)
      return
    }
    void executeConfirm(referenceConflict, fileConflict)
  }

  const handleCandidateClick = (candidate: RecognitionCandidate) => {
    const bill = paidBills.find((item) => item.id === candidate.debt_bill_id)
    if (bill) {
      setTargetBillId(String(bill.id))
      return
    }
    showToast(t('documents.foreign_candidates'), 'info')
  }

  return (
    <>
      <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
        <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-lg max-h-[90vh] overflow-y-auto">
          <div className="p-5">
            <div className="flex items-start justify-between gap-3 mb-4">
              <div className="flex items-start gap-2 min-w-0">
                <Icon name="pdf" className="w-5 h-5 text-primary shrink-0 mt-0.5" />
                <div className="min-w-0">
                  <h3 className="text-lg sm:text-xl font-semibold">{t('documents.title')}</h3>
                  <p className="text-xs text-text-secondary mt-0.5">{t('documents.subtitle')}</p>
                </div>
              </div>
              <button
                onClick={onClose}
                disabled={saving}
                className="w-8 h-8 shrink-0 rounded-full flex items-center justify-center text-text-secondary hover:bg-bg transition-colors disabled:opacity-50"
                title={t('app.close')}
                aria-label={t('app.close')}
              >
                <Icon name="cancel" className="w-4 h-4" />
              </button>
            </div>

            {!recognition ? (
              <div className="space-y-4">
                <PdfUploadField
                  label={t('documents.upload_label')}
                  optionalLabel=""
                  hint={t('documents.upload_hint')}
                  formats={t('documents.upload_formats')}
                  file={file}
                  disabled={analyzing || saving}
                  onSelect={handleSelectFile}
                  onRemove={() => handleSelectFile(null)}
                />
                {analyzing && (
                  <div className="flex items-center gap-2 text-sm text-text-secondary">
                    <div className="w-4 h-4 rounded-full border-2 border-border border-t-primary animate-spin" />
                    {t('documents.analyzing')}
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-4">
                <div className="rounded-ios-sm border border-border bg-bg p-3">
                  <div className="flex items-center justify-between gap-2 flex-wrap">
                    <span className="text-xs font-semibold px-2 py-1 rounded-full bg-primary/10 text-primary">
                      {documentTypeLabel}
                    </span>
                    <div className="flex items-center gap-2">
                      {recognition.duplicate && (
                        <span className="text-xs font-medium text-text-secondary">
                          {t('documents.duplicate')}
                        </span>
                      )}
                      {recognition.review_required && (
                        <span className="text-xs font-semibold px-2 py-1 rounded-full bg-warning/20 text-yellow-800 dark:text-yellow-400">
                          {t('documents.review_required')}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="mt-3">
                    <div className="flex items-center justify-between text-xs text-text-secondary mb-1">
                      <span>{t('documents.confidence')}</span>
                      <span>{confidencePercent}%</span>
                    </div>
                    <div className="h-2 rounded-full bg-border overflow-hidden">
                      <div className={`h-full ${confidenceColor}`} style={{ width: `${confidencePercent}%` }} />
                    </div>
                  </div>
                  <div className="mt-3 flex items-center justify-between gap-2">
                    <span className="text-xs text-text-secondary truncate">
                      {recognition.installment_number
                        ? `${t('documents.installment')}: #${recognition.installment_number}`
                        : t('documents.warning_sin_numero_cuota')}
                    </span>
                    <button
                      onClick={() => handleSelectFile(null)}
                      disabled={saving}
                      className="text-xs font-medium text-primary hover:underline disabled:opacity-50"
                    >
                      {t('documents.change_file')}
                    </button>
                  </div>
                </div>

                {recognition.warnings && recognition.warnings.length > 0 && (
                  <div className="rounded-ios-sm bg-warning/10 px-3 py-2">
                    <p className="text-xs font-semibold text-yellow-800 dark:text-yellow-400 mb-1">
                      {t('documents.warnings')}
                    </p>
                    <ul className="list-disc list-inside space-y-0.5 text-xs text-text-secondary">
                      {recognition.warnings.map((warning) => (
                        <li key={warning}>{warningLabel(warning)}</li>
                      ))}
                    </ul>
                  </div>
                )}

                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium mb-1">{t('documents.reference')}</label>
                    <input
                      type="text"
                      value={reference}
                      onChange={(e) => setReference(e.target.value)}
                      maxLength={64}
                      autoComplete="off"
                      className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
                    />
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-sm font-medium mb-1">{t('documents.paid_at')}</label>
                      <input
                        type="date"
                        value={paidAt}
                        onChange={(e) => setPaidAt(e.target.value)}
                        className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-1">{t('documents.amount')}</label>
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        value={amount}
                        onChange={(e) => setAmount(e.target.value)}
                        className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-sm font-medium mb-1">{t('documents.currency')}</label>
                      <Select
                        options={currencyOptions}
                        value={currency}
                        onChange={(value) => setCurrency(String(value))}
                        placeholder={t('documents.currency')}
                        searchable
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-1">{t('documents.target')}</label>
                      <Select
                        options={billOptions}
                        value={targetBillId}
                        onChange={(value) => setTargetBillId(String(value))}
                        placeholder={t('documents.target_placeholder')}
                        searchable={billOptions.length > 8}
                      />
                    </div>
                  </div>

                  {paidBills.length === 0 && (
                    <p className="text-xs text-danger">{t('documents.no_paid_bills')}</p>
                  )}

                  <div>
                    <label className="block text-sm font-medium mb-1">{t('documents.concept')}</label>
                    <input
                      type="text"
                      value={concept}
                      onChange={(e) => setConcept(e.target.value)}
                      maxLength={200}
                      autoComplete="off"
                      className="w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] bg-card text-text"
                    />
                  </div>

                  {selectedBill && (
                    <div className="text-xs text-text-secondary">
                      <p>
                        {t('documents.target')}: #{selectedBill.installment_number} · {selectedBill.due_date}
                        {selectedBill.payment_reference ? ` · ${selectedBill.payment_reference}` : ''}
                        {selectedBill.has_receipt ? ` · ${t('documents.has_receipt')}` : ''}
                      </p>
                      {paidAt && (
                        <PaymentTimingLabel
                          paidDate={paidAt}
                          dueDate={selectedBill.due_date}
                          mode="receipt"
                          className="block text-xs mt-1"
                        />
                      )}
                    </div>
                  )}

                  {(referenceConflict || fileConflict) && canConfirm && (
                    <div className="rounded-ios-sm bg-warning/10 px-3 py-2 text-xs text-yellow-800 dark:text-yellow-400">
                      {referenceConflict && <p>{t('documents.error_reference_conflict')}</p>}
                      {fileConflict && <p>{t('documents.error_file_exists')}</p>}
                      <p className="mt-1">{t('documents.overwrite_hint')}</p>
                    </div>
                  )}

                  {!canConfirm && (
                    <div className="rounded-ios-sm bg-warning/10 px-3 py-2 text-xs text-yellow-800 dark:text-yellow-400">
                      {t('documents.error_already_associated')}
                    </div>
                  )}
                </div>

                <div>
                  <p className="text-sm font-medium mb-2">{t('documents.candidates')}</p>
                  {localCandidates.length === 0 ? (
                    <p className="text-xs text-text-secondary">{t('documents.no_candidates')}</p>
                  ) : (
                    <div className="space-y-2">
                      {localCandidates.map((candidate) => (
                        <button
                          key={candidate.debt_bill_id}
                          type="button"
                          onClick={() => handleCandidateClick(candidate)}
                          className={`w-full text-left rounded-ios-sm border px-3 py-2 transition-colors ${
                            String(candidate.debt_bill_id) === targetBillId
                              ? 'border-primary bg-primary/5'
                              : 'border-border bg-bg hover:border-primary/50'
                          }`}
                        >
                          <div className="flex items-center justify-between gap-2">
                            <span className="text-sm font-medium">
                              #{candidate.installment_number} · {formatMoney(candidate.amount, candidate.currency_code || currencySymbol)}
                            </span>
                            <span className="text-xs font-semibold text-primary">
                              {Math.round(candidate.score * 100)}%
                            </span>
                          </div>
                          <p className="text-xs text-text-secondary truncate">
                            {candidate.debt_description} · {candidate.due_date}
                          </p>
                          {candidate.existing_reference && (
                            <p className="text-xs text-text-secondary mt-1">
                              {t('documents.current_reference')}: {candidate.existing_reference}
                            </p>
                          )}
                          {candidate.reasons && candidate.reasons.length > 0 && (
                            <p className="text-xs text-text-secondary mt-1">
                              {candidate.reasons.map(reasonLabel).join(' · ')}
                            </p>
                          )}
                          {candidate.warnings && candidate.warnings.length > 0 && (
                            <p className="text-xs text-yellow-800 dark:text-yellow-400 mt-1">
                              {candidate.warnings.map(warningLabel).join(' · ')}
                            </p>
                          )}
                        </button>
                      ))}
                    </div>
                  )}
                  {foreignCandidates.length > 0 && (
                    <p className="text-xs text-text-secondary mt-2">
                      {t('documents.foreign_candidates')}:{' '}
                      {foreignCandidates.map((candidate) => `${candidate.debt_description} #${candidate.installment_number}`).join(', ')}
                    </p>
                  )}
                </div>
              </div>
            )}
          </div>

          <div className="flex gap-3 p-4 border-t border-border">
            <button
              onClick={onClose}
              disabled={saving}
              className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px] disabled:opacity-50"
            >
              <Icon name="cancel" className="w-4 h-4" />
              {t('app.cancel')}
            </button>
            {!recognition ? (
              <button
                onClick={handleRecognize}
                disabled={!file || analyzing || saving}
                className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
              >
                {analyzing ? (
                  <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
                ) : (
                  <Icon name="pdf" className="w-4 h-4" />
                )}
                {analyzing ? t('documents.analyzing') : t('documents.analyze')}
              </button>
            ) : (
              <button
                onClick={handleConfirmClick}
                disabled={saving || !canConfirm || !selectedBill || !reference.trim()}
                className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
              >
                {saving ? (
                  <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
                ) : (
                  <Icon name="check" className="w-4 h-4" />
                )}
                {saving ? t('documents.confirming') : t('documents.confirm')}
              </button>
            )}
          </div>
        </div>
      </div>

      {pendingOverwrite && (
        <ConfirmWordModal
          title={t('documents.overwrite_title')}
          subtitle={t('documents.overwrite_subtitle')}
          confirmLabel={t('documents.overwrite_confirm')}
          onConfirm={async () => {
            setPendingOverwrite(false)
            await executeConfirm(referenceConflict, fileConflict)
          }}
          onCancel={() => setPendingOverwrite(false)}
        />
      )}
    </>
  )
}
