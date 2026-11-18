import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../../../../stores/i18nStore'
import { usePageTitle } from '../../../../hooks/usePageTitle'
import { useToast } from '../../../../components/Toast'
import { useCurrencyFormatStore } from '../../../../stores/currencyFormatStore'
import { Icon } from '../../../../components/Icons'
import LoadingSpinner from '../../../../components/LoadingSpinner'
import { creditCardsApi } from '../api'
import type { CreditCard, ParsedStatement } from '../types'
import { formatISODate, transactionTypeLabel } from '../utils'

export default function CreditCardImportPage() {
  const { id, cardId } = useParams()
  const bankId = Number(id)
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  const { showToast } = useToast()
  const formatMoney = useCurrencyFormatStore((state) => state.formatMoney)

  const [card, setCard] = useState<CreditCard | null>(null)
  const [loading, setLoading] = useState(true)
  const [analyzing, setAnalyzing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<ParsedStatement | null>(null)

  usePageTitle(t('creditCards.import_title'))

  useEffect(() => {
    let mounted = true
    creditCardsApi
      .get(Number(cardId))
      .then((detail) => {
        if (mounted) setCard(detail)
      })
      .catch((err) => {
        if (mounted) showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      })
      .finally(() => {
        if (mounted) setLoading(false)
      })
    return () => {
      mounted = false
    }
  }, [cardId, showToast, t])

  const handleAnalyze = useCallback(async () => {
    if (!file) {
      showToast(t('creditCards.import_select_file'), 'error')
      return
    }
    if (file.size > 5 * 1024 * 1024) {
      showToast(t('creditCards.import_too_large'), 'error')
      return
    }
    setAnalyzing(true)
    try {
      const parsed = await creditCardsApi.previewImport(Number(cardId), file)
      if (parsed) setPreview(parsed)
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setAnalyzing(false)
    }
  }, [cardId, file, showToast, t])

  const handleConfirm = useCallback(async () => {
    if (!preview) return
    setSaving(true)
    try {
      const statement = await creditCardsApi.confirmImport(Number(cardId), preview)
      showToast(t('creditCards.import_success'), 'success')
      if (statement) {
        navigate(`/bancos/${bankId}/tarjetas/${cardId}/estados/${statement.id}`, { replace: true })
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }, [bankId, cardId, navigate, preview, showToast, t])

  if (loading) return <LoadingSpinner />

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('creditCards.import_title')}</h2>
        <p className="text-sm text-text-secondary mt-1">
          {card?.alias || ''}
          {card ? ` · •••• ${card.last4}` : ''}
        </p>
      </div>

      {!preview ? (
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6 space-y-4">
          <p className="text-sm text-text-secondary">{t('creditCards.import_step1')}</p>
          <div>
            <label className="block text-sm font-medium mb-1">{t('creditCards.import_file')}</label>
            <input
              type="file"
              accept="application/pdf,.pdf"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
              className="block w-full text-sm text-text-secondary file:mr-3 file:px-4 file:py-2 file:rounded-ios-sm file:border-0 file:bg-bg file:text-text hover:file:bg-border"
            />
          </div>
          <div className="flex flex-col sm:flex-row gap-3">
            <button
              type="button"
              onClick={() => navigate(`/bancos/${bankId}/tarjetas/${cardId}`)}
              disabled={analyzing}
              className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              <Icon name="cancel" className="w-4 h-4" />
              {t('app.cancel')}
            </button>
            <button
              type="button"
              onClick={handleAnalyze}
              disabled={analyzing || !file}
              className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              {analyzing ? (
                <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
              ) : (
                <Icon name="upload" className="w-4 h-4" />
              )}
              {analyzing ? t('creditCards.import_analyzing') : t('creditCards.import_analyze')}
            </button>
          </div>
        </section>
      ) : (
        <div className="space-y-4">
          <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-sm text-text-secondary">{t('creditCards.import_step2')}</p>
                <h3 className="text-lg font-semibold mt-1">
                  {formatISODate(preview.period_end, lang)}
                </h3>
                <p className="text-sm text-text-secondary">
                  {t('creditCards.statement_due_min')}: {formatISODate(preview.min_payment_due_date, lang)} ·{' '}
                  {t('creditCards.statement_due_full')}: {formatISODate(preview.full_payment_due_date, lang)}
                </p>
              </div>
              <span
                className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                  preview.warnings && preview.warnings.length > 0
                    ? 'bg-warning/10 text-warning'
                    : 'bg-success/10 text-success'
                }`}
              >
                {preview.warnings && preview.warnings.length > 0
                  ? t('creditCards.import_mismatch')
                  : t('creditCards.import_balanced')}
              </span>
            </div>

            <div className="mt-4 grid grid-cols-2 lg:grid-cols-4 gap-3">
              <Metric label={t('creditCards.previous_balance')} value={formatMoney(preview.previous_balance)} sub={formatMoney(preview.previous_balance_usd) + ' USD'} />
              <Metric label={t('creditCards.new_balance')} value={formatMoney(preview.new_balance)} sub={formatMoney(preview.new_balance_usd) + ' USD'} />
              <Metric label={t('creditCards.min_payment')} value={formatMoney(preview.min_payment)} sub={formatMoney(preview.min_payment_usd) + ' USD'} />
              <Metric label={t('creditCards.total_payment')} value={formatMoney(preview.total_payment)} sub={formatMoney(preview.total_payment_usd) + ' USD'} />
              {preview.interest_bonifiable > 0 && (
                <Metric label={t('creditCards.interest_bonifiable')} value={formatMoney(preview.interest_bonifiable)} sub={formatMoney(preview.interest_bonifiable_usd) + ' USD'} />
              )}
              {preview.mora_amount > 0 && (
                <Metric label={t('creditCards.mora_amount')} value={formatMoney(preview.mora_amount)} sub={formatMoney(preview.mora_amount_usd) + ' USD'} />
              )}
              {preview.arrears_count > 0 && (
                <Metric label={t('creditCards.arrears_count')} value={String(preview.arrears_count)} />
              )}
              {preview.over_limit_amount > 0 && (
                <Metric label={t('creditCards.over_limit_amount')} value={formatMoney(preview.over_limit_amount)} sub={formatMoney(preview.over_limit_amount_usd) + ' USD'} />
              )}
            </div>

            <p className="mt-3 text-xs text-text-secondary font-mono">
              {t('creditCards.import_module')}: {preview.module_id} v{preview.module_version}
            </p>

            {preview.warnings && preview.warnings.length > 0 && (
              <ul className="mt-3 space-y-1">
                {preview.warnings.map((warning, index) => (
                  <li key={`${warning.code}-${index}`} className="text-sm text-warning flex gap-2">
                    <Icon name="warning" className="w-4 h-4 shrink-0 mt-0.5" />
                    {warning.message}
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="bg-card rounded-ios shadow-ios overflow-hidden">
            <div className="p-4 border-b border-border flex items-center justify-between gap-2">
              <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wide">
                {t('creditCards.transactions')}
              </h3>
              <span className="text-sm text-text-secondary">{preview.transactions.length}</span>
            </div>
            <div className="max-h-[420px] overflow-y-auto divide-y divide-border">
              {preview.transactions.map((tx, index) => (
                <div key={`${tx.external_ref}-${index}`} className="p-3 sm:px-4 flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-3 text-sm">
                  <span className="text-text-secondary w-24 shrink-0">{formatISODate(tx.posted_at, lang)}</span>
                  <span className="flex-1 min-w-0 truncate">{tx.concept}</span>
                  <span className="text-xs px-2 py-0.5 rounded-full bg-bg text-text-secondary self-start sm:self-auto">
                    {transactionTypeLabel(tx.type, t)}
                  </span>
                  <span className="font-mono text-right w-32 shrink-0">
                    {tx.amount_local !== 0 ? formatMoney(tx.amount_local) : formatMoney(tx.amount_usd) + ' USD'}
                  </span>
                </div>
              ))}
            </div>
          </section>

          <div className="flex flex-col sm:flex-row gap-3">
            <button
              type="button"
              onClick={() => setPreview(null)}
              disabled={saving}
              className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              <Icon name="back" className="w-4 h-4" />
              {t('creditCards.import_choose_another')}
            </button>
            <button
              type="button"
              onClick={handleConfirm}
              disabled={saving}
              className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              {saving ? (
                <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
              ) : (
                <Icon name="check" className="w-4 h-4" />
              )}
              {saving ? t('app.saving') : t('creditCards.import_confirm')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function Metric({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="bg-bg rounded-ios-sm p-3">
      <p className="text-xs text-text-secondary uppercase tracking-wide">{label}</p>
      <p className="text-sm sm:text-base font-semibold mt-0.5">{value}</p>
      {sub && <p className="text-xs text-text-secondary">{sub}</p>}
    </div>
  )
}
