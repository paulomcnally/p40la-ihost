import { useEffect, useState } from 'react'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useCurrencyFormatStore } from '../stores/currencyFormatStore'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import Select from './Select'
import AutoAttachmentsField from './AutoAttachmentsField'
import type { AutoAttachment, AutoMaintenance, AutoMaintenanceItemType, Institution, InstitutionCategory } from '../types'

const MAX_ATTACHMENTS = 20

function todayISO(): string {
  const now = new Date()
  const offset = now.getTimezoneOffset()
  return new Date(now.getTime() - offset * 60 * 1000).toISOString().slice(0, 10)
}

function trimNumber(value: string): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

type MaintenanceItemForm = {
  item_type: AutoMaintenanceItemType
  description: string
  amount: string
}

export default function MaintenanceModal({
  autoId,
  maintenance,
  onSaved,
  onCancel,
}: {
  autoId: number
  maintenance?: AutoMaintenance | null
  onSaved: () => void
  onCancel: () => void
}) {
  const { t } = useI18nStore()
  const formatMoney = useCurrencyFormatStore(s => s.formatMoney)
  const { currencies, loadCurrencies } = useAppStore()
  const isEditing = !!maintenance

  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [performedAt, setPerformedAt] = useState(maintenance?.performed_at ?? todayISO())
  const [mileage, setMileage] = useState(maintenance ? String(maintenance.mileage) : '')
  const [institutionId, setInstitutionId] = useState<number | ''>(maintenance?.institution_id ?? '')
  const [description, setDescription] = useState(maintenance?.description ?? '')
  const [items, setItems] = useState<MaintenanceItemForm[]>(
    maintenance?.items?.map((item) => ({
      item_type: item.item_type,
      description: item.description,
      amount: String(item.amount),
    })) ?? [],
  )
  const [currencyId, setCurrencyId] = useState<number | ''>(maintenance?.currency_id ?? '')
  const [savedId, setSavedId] = useState<number | null>(maintenance?.id ?? null)
  const [attachments, setAttachments] = useState<AutoAttachment[]>(maintenance?.attachments ?? [])
  const [pendingFiles, setPendingFiles] = useState<File[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [activeFormTab, setActiveFormTab] = useState<'info' | 'invoices'>('info')

  useEffect(() => {
    loadCurrencies()
    Promise.all([api.institutions.list(), api.institutionCategories.list()])
      .then(([instList, catList]) => {
        setInstitutions(instList || [])
        setCategories(catList || [])
      })
      .finally(() => setLoading(false))
  }, [loadCurrencies])

  useEffect(() => {
    if (!currencyId && currencies.length > 0) {
      setCurrencyId(maintenance?.currency_id ?? currencies[0].id)
    }
  }, [currencies, currencyId, maintenance])

  const workshopCategory = categories.find((category) => category.key === 'workshop')
  const workshops = workshopCategory
    ? institutions.filter((institution) => institution.category_id === workshopCategory.id)
    : []

  const institutionOptions = workshops.map((institution) => ({
    value: institution.id,
    label: institution.name,
  }))
  const currencyOptions = currencies.map((currency) => ({
    value: currency.id,
    label: `${currency.code} (${currency.symbol})`,
  }))
  const selectedCurrency = currencies.find((currency) => currency.id === currencyId)
  const currencySymbol = selectedCurrency?.symbol || selectedCurrency?.code || ''

  const addItem = (itemType: AutoMaintenanceItemType) => {
    setItems((prev) => [...prev, { item_type: itemType, description: '', amount: '' }])
  }

  const updateItem = (index: number, patch: Partial<MaintenanceItemForm>) => {
    setItems((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)))
  }

  const removeItem = (index: number) => {
    setItems((prev) => prev.filter((_, i) => i !== index))
  }

  const itemTotal = (itemType: AutoMaintenanceItemType) =>
    items
      .filter((item) => item.item_type === itemType)
      .reduce((sum, item) => sum + trimNumber(item.amount), 0)

  const grandTotal = itemTotal('labor') + itemTotal('supplies')

  const handleUpload = async (files: File[]) => {
    if (!savedId) {
      setPendingFiles((prev) => [...prev, ...files].slice(0, MAX_ATTACHMENTS - attachments.length))
      return
    }
    for (const file of files) {
      const attachment = await api.autos.uploadMaintenanceAttachment(autoId, savedId, file)
      if (attachment) setAttachments((prev) => [...prev, attachment])
    }
  }

  const handleDeleteAttachment = async (attachment: AutoAttachment) => {
    if (!savedId) return
    await api.autos.deleteMaintenanceAttachment(autoId, savedId, attachment.id)
    setAttachments((prev) => prev.filter((item) => item.id !== attachment.id))
  }

  const handleSubmit = async () => {
    setError('')
    if (!performedAt) {
      setActiveFormTab('info')
      setError(t('autos.maintenances.date_required'))
      return
    }
    if (mileage === '' || trimNumber(mileage) < 0) {
      setActiveFormTab('info')
      setError(t('autos.maintenances.mileage_required'))
      return
    }
    if (!institutionId) {
      setActiveFormTab('info')
      setError(t('autos.maintenances.institution_required'))
      return
    }
    if (items.length === 0) {
      setActiveFormTab('info')
      setError(t('autos.maintenances.items_required'))
      return
    }
    for (const item of items) {
      if (!item.description.trim()) {
        setActiveFormTab('info')
        setError(t('autos.maintenances.item_description_required'))
        return
      }
      const amount = Number(item.amount)
      if (item.amount.trim() === '' || !Number.isFinite(amount) || amount < 0) {
        setActiveFormTab('info')
        setError(t('autos.maintenances.item_amount_invalid'))
        return
      }
    }
    if (!currencyId) {
      setActiveFormTab('info')
      setError(t('autos.maintenances.currency_required'))
      return
    }

    setSubmitting(true)
    let savedRecord: AutoMaintenance | null = null
    try {
      const body = {
        performed_at: performedAt,
        mileage: trimNumber(mileage),
        institution_id: Number(institutionId),
        description: description.trim(),
        currency_id: Number(currencyId),
        items: items.map((item) => ({
          item_type: item.item_type,
          description: item.description.trim(),
          amount: Number(item.amount),
        })),
      }
      savedRecord = savedId
        ? await api.autos.updateMaintenance(autoId, savedId, body)
        : await api.autos.createMaintenance(autoId, body)
      if (!savedRecord) throw new Error(t('autos.maintenances.save_error'))

      setSavedId(savedRecord.id)
      setAttachments(savedRecord.attachments ?? [])
      for (const file of pendingFiles) {
        const attachment = await api.autos.uploadMaintenanceAttachment(autoId, savedRecord.id, file)
        if (attachment) setAttachments((prev) => [...prev, attachment])
      }
      setPendingFiles([])
      onSaved()
    } catch (err) {
      if (savedRecord) {
        setSavedId(savedRecord.id)
        setAttachments(savedRecord.attachments ?? [])
      }
      setError(err instanceof Error ? err.message : t('autos.maintenances.save_error'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onCancel}>
      <div
        className="bg-card rounded-ios shadow-ios w-full max-w-lg max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-border flex items-center justify-between gap-3">
          <h3 className="text-lg font-bold">
            {isEditing ? t('autos.maintenances.edit') : t('autos.maintenances.add')}
          </h3>
          <button
            type="button"
            onClick={onCancel}
            className="p-2 text-text-secondary hover:text-text rounded-full transition-colors"
            aria-label={t('app.cancel')}
          >
            <Icon name="cancel" className="w-5 h-5" />
          </button>
        </div>

        {!loading && (
          <div className="px-4 pt-4">
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => setActiveFormTab('info')}
                aria-pressed={activeFormTab === 'info'}
                className={`inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
                  activeFormTab === 'info' ? 'bg-primary text-white' : 'bg-bg text-text-secondary hover:bg-border'
                }`}
              >
                {t('autos.form_tabs.information')}
              </button>
              <button
                type="button"
                onClick={() => setActiveFormTab('invoices')}
                aria-pressed={activeFormTab === 'invoices'}
                className={`inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
                  activeFormTab === 'invoices' ? 'bg-primary text-white' : 'bg-bg text-text-secondary hover:bg-border'
                }`}
              >
                {t('autos.form_tabs.invoices')}
                <span
                  className={`inline-flex items-center justify-center min-w-5 px-1.5 py-0.5 rounded-full text-[10px] ${
                    activeFormTab === 'invoices' ? 'bg-white/20 text-white' : 'bg-card text-text-secondary'
                  }`}
                >
                  {attachments.length + pendingFiles.length}
                </span>
              </button>
            </div>
          </div>
        )}

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {loading ? (
            <p className="text-text-secondary text-center py-6">{t('app.loading')}</p>
          ) : activeFormTab === 'info' ? (
            <>
              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.maintenances.date')}</label>
                <input
                  type="date"
                  value={performedAt}
                  max={todayISO()}
                  onChange={(e) => setPerformedAt(e.target.value)}
                  className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.maintenances.mileage')}</label>
                <input
                  type="number"
                  inputMode="numeric"
                  min={0}
                  value={mileage}
                  onChange={(e) => setMileage(e.target.value)}
                  placeholder={t('autos.maintenances.mileage_placeholder')}
                  className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] placeholder:text-text-secondary"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.maintenances.institution')}</label>
                <Select
                  options={institutionOptions}
                  value={institutionId}
                  onChange={(value) => setInstitutionId(value === '' ? '' : Number(value))}
                  placeholder={t('autos.maintenances.select_institution')}
                  searchable
                  searchPlaceholder={t('app.search')}
                  emptyLabel={t('app.no_results')}
                />
                {workshops.length === 0 && (
                  <p className="text-xs text-text-secondary mt-1">
                    {t('autos.maintenances.no_workshops')}{' '}
                    <a href="/institutions/new" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                      {t('autos.maintenances.create_institution')}
                    </a>
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.maintenances.description')}</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder={t('autos.maintenances.description_placeholder')}
                  rows={2}
                  className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary placeholder:text-text-secondary"
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.maintenances.currency')}</label>
                <Select
                  options={currencyOptions}
                  value={currencyId}
                  onChange={(value) => setCurrencyId(Number(value))}
                  placeholder={t('autos.maintenances.select_currency')}
                  searchable
                  searchPlaceholder={t('app.search')}
                  emptyLabel={t('app.no_results')}
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-sm font-medium">{t('autos.maintenances.items')}</label>
                  <span className="text-xs text-text-secondary">{formatMoney(grandTotal, currencySymbol)}</span>
                </div>
                <div className="space-y-4">
                  {([
                    { type: 'labor' as AutoMaintenanceItemType, label: t('autos.maintenances.labor_items'), add: t('autos.maintenances.add_labor') },
                    { type: 'supplies' as AutoMaintenanceItemType, label: t('autos.maintenances.supplies_items'), add: t('autos.maintenances.add_supply') },
                  ]).map((section) => {
                    const sectionRows = items
                      .map((item, index) => ({ item, index }))
                      .filter(({ item }) => item.item_type === section.type)
                    return (
                      <div key={section.type}>
                        <div className="flex items-center justify-between gap-2 mb-2">
                          <span className="text-sm font-medium">{section.label}</span>
                          <span className="text-xs text-text-secondary">{formatMoney(itemTotal(section.type), currencySymbol)}</span>
                        </div>
                        <div className="space-y-2">
                          {sectionRows.map(({ item, index }) => (
                            <div key={index} className="flex items-start gap-2">
                              <input
                                type="text"
                                value={item.description}
                                onChange={(e) => updateItem(index, { description: e.target.value })}
                                placeholder={t('autos.maintenances.item_description_placeholder')}
                                className="flex-1 min-w-0 px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] placeholder:text-text-secondary"
                              />
                              <input
                                type="number"
                                inputMode="decimal"
                                min={0}
                                step="0.01"
                                value={item.amount}
                                onChange={(e) => updateItem(index, { amount: e.target.value })}
                                placeholder="0.00"
                                className="w-24 sm:w-28 px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
                              />
                              <button
                                type="button"
                                onClick={() => removeItem(index)}
                                className="p-2 text-text-secondary hover:text-danger transition-colors rounded-full min-h-[44px] min-w-[44px] flex items-center justify-center"
                                aria-label={t('app.remove')}
                              >
                                <Icon name="delete" className="w-4 h-4" />
                              </button>
                            </div>
                          ))}
                          <button
                            type="button"
                            onClick={() => addItem(section.type)}
                            className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-full border border-border bg-card text-xs font-medium text-text-secondary hover:border-primary/50 hover:text-primary transition-colors min-h-[36px]"
                          >
                            <Icon name="plus" className="w-3.5 h-3.5" />
                            {section.add}
                          </button>
                        </div>
                      </div>
                    )
                  })}
                </div>
                {items.length === 0 && (
                  <p className="text-xs text-text-secondary mt-2">{t('autos.maintenances.items_empty')}</p>
                )}
                <div className="flex flex-wrap items-center gap-x-4 gap-y-1 mt-3 text-sm">
                  <span className="text-text-secondary">
                    {t('autos.maintenances.labor_cost')}:{' '}
                    <strong className="text-text">{formatMoney(itemTotal('labor'), currencySymbol)}</strong>
                  </span>
                  <span className="text-text-secondary">
                    {t('autos.maintenances.supplies_cost')}:{' '}
                    <strong className="text-text">{formatMoney(itemTotal('supplies'), currencySymbol)}</strong>
                  </span>
                  <span className="font-semibold">
                    {t('autos.maintenances.total')}: {formatMoney(grandTotal, currencySymbol)}
                  </span>
                </div>
              </div>
            </>
          ) : (
            <>
              <AutoAttachmentsField
                attachments={attachments}
                attachmentUrl={(attachment) =>
                  savedId
                    ? api.autos.maintenanceAttachmentUrl(autoId, savedId, attachment.id)
                    : '#'
                }
                onUpload={handleUpload}
                onDelete={handleDeleteAttachment}
                max={MAX_ATTACHMENTS}
              />
              {pendingFiles.length > 0 && (
                <div className="space-y-1">
                  <p className="text-xs text-text-secondary">{t('autos.attachments.pending')}</p>
                  {pendingFiles.map((file, index) => (
                    <div key={`${file.name}-${index}`} className="flex items-center gap-2 text-sm">
                      <Icon name="pdf" className="w-4 h-4 text-danger flex-shrink-0" />
                      <span className="flex-1 truncate">{file.name}</span>
                      <button
                        type="button"
                        onClick={() => setPendingFiles((prev) => prev.filter((_, i) => i !== index))}
                        className="p-1.5 text-text-secondary hover:text-danger transition-colors rounded-full"
                        aria-label={t('app.remove')}
                      >
                        <Icon name="delete" className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
          {!loading && error && <p className="text-sm text-danger">{error}</p>}
        </div>

        <div className="p-4 border-t border-border flex justify-end gap-3">
          <button
            type="button"
            onClick={() => {
              if (savedId && savedId !== maintenance?.id) onSaved()
              else onCancel()
            }}
            className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
          >
            {t('app.cancel')}
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={submitting || loading}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
          >
            {submitting ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </div>
    </div>
  )
}
