import { useEffect, useMemo, useState } from 'react'
import { useI18nStore } from '../../../../stores/i18nStore'
import { useToast } from '../../../../components/Toast'
import { Icon } from '../../../../components/Icons'
import Select from '../../../../components/Select'
import type { CategoryGroup } from '../../../../types'
import type { CardTransaction, CardTransactionType } from '../types'
import { TRANSACTION_TYPES, todayISO, transactionTypeLabel } from '../utils'

const inputClass =
  'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]'

export interface TransactionInput {
  posted_at: string
  concept: string
  category_id?: number
  type: CardTransactionType
  amount_local: number
  amount_usd: number
  note: string
}

export default function TransactionFormModal({
  transaction,
  categoryGroups,
  onSave,
  onClose,
}: {
  transaction: CardTransaction | null
  categoryGroups: CategoryGroup[]
  onSave: (input: TransactionInput) => Promise<void>
  onClose: () => void
}) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [saving, setSaving] = useState(false)

  const [postedAt, setPostedAt] = useState(transaction?.posted_at || todayISO())
  const [concept, setConcept] = useState(transaction?.concept || '')
  const [categoryId, setCategoryId] = useState<number>(transaction?.category_id || 0)
  const [type, setType] = useState<CardTransactionType>(transaction?.type || 'purchase')
  const [amountLocal, setAmountLocal] = useState(transaction?.amount_local || 0)
  const [amountUSD, setAmountUSD] = useState(transaction?.amount_usd || 0)
  const [note, setNote] = useState(transaction?.note || '')

  const categoryOptions = useMemo(() => {
    const flat = categoryGroups.flatMap((group) => group.categories || [])
    return [
      { value: 0, label: t('creditCards.no_category') },
      ...flat.map((category) => ({ value: category.id, label: category.name })),
    ]
  }, [categoryGroups, t])

  useEffect(() => {
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = ''
    }
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!concept.trim() || !postedAt) {
      showToast(t('creditCards.validation_required'), 'error')
      return
    }
    setSaving(true)
    try {
      await onSave({
        posted_at: postedAt,
        concept: concept.trim(),
        category_id: categoryId || undefined,
        type,
        amount_local: Number(amountLocal) || 0,
        amount_usd: Number(amountUSD) || 0,
        note: note.trim(),
      })
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-[60] p-4">
      <div className="bg-card rounded-ios shadow-ios-lg w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between gap-3 p-4 border-b border-border">
          <h3 className="text-lg font-semibold">
            {transaction ? t('creditCards.edit_transaction') : t('creditCards.add_transaction')}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="w-9 h-9 rounded-full flex items-center justify-center hover:bg-bg"
            aria-label={t('app.close')}
          >
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">
                {t('creditCards.transaction_date')} <span className="text-danger">*</span>
              </label>
              <input
                type="date"
                value={postedAt}
                onChange={(e) => setPostedAt(e.target.value)}
                className={inputClass}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.transaction_type')}</label>
              <Select
                options={TRANSACTION_TYPES.map((value) => ({
                  value,
                  label: transactionTypeLabel(value, t),
                }))}
                value={type}
                onChange={(value) => setType(value as CardTransactionType)}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              {t('creditCards.transaction_concept')} <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={concept}
              onChange={(e) => setConcept(e.target.value)}
              maxLength={200}
              className={inputClass}
              autoComplete="off"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('creditCards.transaction_category')}</label>
            <Select
              options={categoryOptions}
              value={categoryId}
              onChange={(value) => setCategoryId(Number(value))}
              searchable={categoryOptions.length > 8}
              searchPlaceholder={t('app.search')}
              emptyLabel={t('app.no_results')}
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.transaction_local')}</label>
              <input
                type="number"
                step="0.01"
                value={amountLocal}
                onChange={(e) => setAmountLocal(Number(e.target.value))}
                className={inputClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('creditCards.transaction_usd')}</label>
              <input
                type="number"
                step="0.01"
                value={amountUSD}
                onChange={(e) => setAmountUSD(Number(e.target.value))}
                className={inputClass}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('creditCards.transaction_note')}</label>
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              maxLength={500}
              rows={2}
              className={`${inputClass} resize-y`}
            />
          </div>

          <div className="flex flex-col sm:flex-row gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={saving}
              className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              {t('app.cancel')}
            </button>
            <button
              type="submit"
              disabled={saving}
              className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
            >
              {saving ? t('app.saving') : t('app.save')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
