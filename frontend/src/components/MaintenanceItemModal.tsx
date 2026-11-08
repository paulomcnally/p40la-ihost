import { useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import type { AutoMaintenanceItem, AutoMaintenanceItemType } from '../types'

export default function MaintenanceItemModal({
  autoId,
  maintenanceId,
  itemType,
  item,
  onSaved,
  onCancel,
}: {
  autoId: number
  maintenanceId: number
  itemType: AutoMaintenanceItemType
  item?: AutoMaintenanceItem | null
  onSaved: () => void
  onCancel: () => void
}) {
  const { t } = useI18nStore()
  const isEditing = !!item
  const [description, setDescription] = useState(item?.description ?? '')
  const [amount, setAmount] = useState(item ? String(item.amount) : '')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const title = isEditing
    ? t('autos.maintenances.edit_item')
    : itemType === 'labor'
      ? t('autos.maintenances.add_labor')
      : t('autos.maintenances.add_supply')

  const handleSubmit = async () => {
    setError('')
    if (!description.trim()) {
      setError(t('autos.maintenances.item_description_required'))
      return
    }
    const parsedAmount = Number(amount)
    if (amount.trim() === '' || !Number.isFinite(parsedAmount) || parsedAmount < 0) {
      setError(t('autos.maintenances.item_amount_invalid'))
      return
    }

    setSubmitting(true)
    try {
      const body = {
        item_type: itemType,
        description: description.trim(),
        amount: parsedAmount,
      }
      if (isEditing && item) {
        await api.autos.updateMaintenanceItem(autoId, maintenanceId, item.id, body)
      } else {
        await api.autos.createMaintenanceItem(autoId, maintenanceId, body)
      }
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('autos.maintenances.save_error'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onCancel}>
      <div
        className="bg-card rounded-ios shadow-ios w-full max-w-sm"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-border flex items-center justify-between gap-3">
          <h3 className="text-lg font-bold">{title}</h3>
          <button
            type="button"
            onClick={onCancel}
            className="p-2 text-text-secondary hover:text-text rounded-full transition-colors"
            aria-label={t('app.cancel')}
          >
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('autos.maintenances.item_description')}</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('autos.maintenances.item_description_placeholder')}
              className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] placeholder:text-text-secondary"
              autoFocus
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('autos.maintenances.item_amount')}</label>
            <input
              type="number"
              inputMode="decimal"
              min={0}
              step="0.01"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="0.00"
              className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
            />
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
        </div>

        <div className="p-4 border-t border-border flex justify-end gap-3">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
          >
            {t('app.cancel')}
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={submitting}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
          >
            {submitting ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </div>
    </div>
  )
}
