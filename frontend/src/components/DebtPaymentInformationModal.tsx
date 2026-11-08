import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from './Toast'
import { Icon } from './Icons'
import Select from './Select'
import LoadingSpinner from './LoadingSpinner'
import ConfirmWordModal from './ConfirmWordModal'
import DebtInformationFieldInput from './DebtInformationFieldInput'
import {
  buildDebtInformationContext,
  resolveDebtInformationText,
  validateDebtInformationValues,
  type DebtInformationValidationIssue,
} from '../utils/debtInformation'
import type {
  Debt,
  DebtBill,
  DebtInformationField,
  DebtInformationTemplate,
  DebtInformationValue,
  DebtPaymentInformation,
} from '../types'

interface DebtPaymentInformationModalProps {
  debt: Debt
  paymentInfo: DebtPaymentInformation | null
  nextBill?: DebtBill | null
  symbol?: string
  onClose: () => void
  onSaved: () => void
}

function defaultValueForField(field: DebtInformationField): DebtInformationValue {
  if (field.default_value === undefined || field.default_value === '') {
    return field.type === 'checkbox' ? false : ''
  }
  if (field.type === 'number') {
    const parsed = Number(field.default_value)
    return Number.isFinite(parsed) ? parsed : ''
  }
  if (field.type === 'checkbox') {
    return field.default_value === 'true'
  }
  return field.default_value
}

function issueText(
  issue: DebtInformationValidationIssue,
  t: (key: string, fallback?: string) => string,
): string {
  let message = t(issue.key)
  if (issue.params) {
    for (const [key, value] of Object.entries(issue.params)) {
      message = message.split(`{${key}}`).join(String(value))
    }
  }
  return message
}

export default function DebtPaymentInformationModal({
  debt,
  paymentInfo,
  nextBill,
  symbol,
  onClose,
  onSaved,
}: DebtPaymentInformationModalProps) {
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  const formatMoney = useCurrencyFormatStore((s) => s.formatMoney)
  const { showToast } = useToast()

  const [templates, setTemplates] = useState<DebtInformationTemplate[]>([])
  const [selectedId, setSelectedId] = useState<number | ''>(paymentInfo?.template_id ?? '')
  const [template, setTemplate] = useState<DebtInformationTemplate | null>(paymentInfo?.template ?? null)
  const [values, setValues] = useState<Record<string, DebtInformationValue>>(paymentInfo?.values ?? {})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [pendingTemplateId, setPendingTemplateId] = useState<number | null>(null)
  const [clearConfirm, setClearConfirm] = useState(false)

  useEffect(() => {
    let active = true
    api.debtInformationTemplates
      .list()
      .then((list) => {
        if (!active) return
        setTemplates(list || [])
        if (paymentInfo?.template_id && !paymentInfo.template) {
          api.debtInformationTemplates
            .get(paymentInfo.template_id)
            .then((detail) => {
              if (active && detail) setTemplate(detail)
            })
            .catch(() => undefined)
        }
      })
      .catch((err) => {
        if (active) showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [paymentInfo, showToast])

  const issues = useMemo(
    () => validateDebtInformationValues(template?.fields ?? [], values),
    [template, values],
  )

  const computedContext = useMemo(() => {
    if (!nextBill) return null
    return buildDebtInformationContext({
      debt,
      bill: nextBill,
      symbol,
      lang,
      statusLabel: t(nextBill.status === 'paid' ? 'bills.status_paid' : 'bills.status_pending'),
      formatMoney,
    })
  }, [debt, nextBill, symbol, lang, t, formatMoney])

  const applyTemplate = async (templateId: number) => {
    try {
      const detail = await api.debtInformationTemplates.get(templateId)
      if (!detail) {
        showToast(t('errors.generic'), 'error')
        return
      }
      const nextValues: Record<string, DebtInformationValue> = {}
      for (const field of detail.fields ?? []) {
        if (field.type === 'computed') continue
        if (values[field.key] !== undefined && values[field.key] !== '') {
          nextValues[field.key] = values[field.key]
        } else {
          nextValues[field.key] = defaultValueForField(field)
        }
      }
      setSelectedId(templateId)
      setTemplate(detail)
      setValues(nextValues)
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    }
  }

  const handleTemplateChange = (value: string | number) => {
    const templateId = Number(value)
    if (!templateId || templateId === selectedId) return
    const hasValues = Object.values(values).some(
      (item) => item !== '' && item !== false && item !== null && item !== undefined,
    )
    if (hasValues) {
      setPendingTemplateId(templateId)
      return
    }
    applyTemplate(templateId)
  }

  const handleSave = async () => {
    if (!selectedId) {
      showToast(t('deudas.payment_method_template_required'), 'error')
      return
    }
    if (issues.length > 0) {
      showToast(issueText(issues[0], t), 'error')
      return
    }
    setSaving(true)
    try {
      await api.debtPaymentInformation.save(debt.id, {
        template_id: Number(selectedId),
        values,
      })
      showToast(t('deudas.payment_method_saved'), 'success')
      onSaved()
      onClose()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleClear = async () => {
    setSaving(true)
    try {
      await api.debtPaymentInformation.clear(debt.id)
      showToast(t('deudas.payment_method_removed'), 'success')
      setClearConfirm(false)
      onSaved()
      onClose()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const createTemplate = () => {
    onClose()
    navigate('/deudas/plantillas/informacion/new')
  }

  const manageTemplates = () => {
    onClose()
    navigate('/deudas/plantillas/informacion')
  }

  const inputFields = (template?.fields ?? []).filter((field) => field.type !== 'computed')
  const computedFields = (template?.fields ?? []).filter((field) => field.type === 'computed')

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
      <div
        className="bg-card rounded-ios shadow-ios-lg w-full max-w-lg max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-5">
          <div className="flex items-start gap-3">
            <div className="w-12 h-12 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
              <Icon name="bank" className="w-6 h-6" />
            </div>
            <div className="min-w-0">
              <h3 className="text-lg sm:text-xl font-semibold">{t('deudas.payment_method')}</h3>
              <p className="text-sm text-text-secondary break-words">{debt.description}</p>
            </div>
          </div>

          {loading ? (
            <div className="py-8">
              <LoadingSpinner />
            </div>
          ) : templates.length === 0 ? (
            <div className="py-6 text-center">
              <p className="text-sm text-text-secondary">{t('deudas.payment_method_empty')}</p>
              <p className="text-sm text-text-secondary mt-1">{t('deudas.payment_method_empty_hint')}</p>
              <button
                type="button"
                onClick={createTemplate}
                className="mt-4 inline-flex items-center gap-2 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium hover:opacity-90 min-h-[44px]"
              >
                <Icon name="plus" className="w-4 h-4" />
                {t('deudas.template_create')}
              </button>
            </div>
          ) : (
            <div className="mt-5 space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('deudas.payment_method_template')} <span className="text-danger">*</span>
                </label>
                <Select
                  options={templates.map((item) => ({ value: item.id, label: item.name }))}
                  value={selectedId}
                  onChange={handleTemplateChange}
                  placeholder={t('deudas.payment_method_template_placeholder')}
                  searchable={templates.length > 8}
                  searchPlaceholder={t('app.search')}
                  emptyLabel={t('app.no_results')}
                />
                <button
                  type="button"
                  onClick={manageTemplates}
                  className="text-xs text-primary hover:underline mt-1.5"
                >
                  {t('deudas.payment_method_manage')}
                </button>
              </div>

              {!template && (
                <p className="text-sm text-text-secondary py-3">
                  {t('deudas.payment_method_select_hint')}
                </p>
              )}

              {template && (
                <div className="space-y-4">
                  {inputFields.map((field) => (
                    <div key={field.key}>
                      <label className="block text-sm font-medium mb-1">
                        {field.label}
                        {field.required && <span className="text-danger ml-0.5">*</span>}
                      </label>
                      <DebtInformationFieldInput
                        field={field}
                        value={values[field.key]}
                        disabled={saving}
                        onChange={(value) =>
                          setValues((prev) => ({ ...prev, [field.key]: value }))
                        }
                      />
                      {field.help_text && (
                        <p className="text-xs text-text-secondary mt-1">{field.help_text}</p>
                      )}
                    </div>
                  ))}

                  {computedFields.length > 0 && (
                    <div className="rounded-ios-sm bg-bg/60 border border-border/60 p-3 space-y-2">
                      <p className="text-xs font-semibold text-text-secondary uppercase tracking-wide">
                        {t('deudas.payment_method_preview')}
                      </p>
                      {computedFields.map((field) => (
                        <div key={field.key}>
                          <p className="text-xs font-medium text-text-secondary">{field.label}</p>
                          <p className="text-sm text-text break-words whitespace-pre-wrap">
                            {computedContext
                              ? resolveDebtInformationText(field.value_template || '', computedContext)
                              : (field.value_template || '—')}
                          </p>
                          {!computedContext && (
                            <p className="text-[11px] text-text-secondary mt-0.5">
                              {t('deudas.payment_method_preview_hint')}
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  )}

                  {issues.length > 0 && (
                    <p className="text-xs text-danger">{issueText(issues[0], t)}</p>
                  )}
                </div>
              )}

              {paymentInfo?.template_id && (
                <button
                  type="button"
                  onClick={() => setClearConfirm(true)}
                  disabled={saving}
                  className="text-sm font-medium text-danger hover:underline min-h-[36px]"
                >
                  {t('deudas.payment_method_clear')}
                </button>
              )}
            </div>
          )}
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
          {templates.length > 0 && (
            <button
              onClick={handleSave}
              disabled={saving || !template}
              className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              {saving ? (
                <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
              ) : (
                <Icon name="save" className="w-4 h-4" />
              )}
              {saving ? t('app.saving') : t('app.save')}
            </button>
          )}
        </div>
      </div>

      {pendingTemplateId !== null && (
        <ConfirmWordModal
          title={t('deudas.payment_method_change_template')}
          subtitle={t('deudas.payment_method_change_template_confirm')}
          confirmLabel={t('deudas.payment_method_change_template_action')}
          onConfirm={() => {
            const next = pendingTemplateId
            setPendingTemplateId(null)
            applyTemplate(next)
          }}
          onCancel={() => setPendingTemplateId(null)}
        />
      )}

      {clearConfirm && (
        <ConfirmWordModal
          title={t('deudas.payment_method_clear')}
          subtitle={t('deudas.payment_method_clear_confirm')}
          confirmLabel={t('app.remove')}
          onConfirm={handleClear}
          onCancel={() => setClearConfirm(false)}
        />
      )}
    </div>
  )
}
