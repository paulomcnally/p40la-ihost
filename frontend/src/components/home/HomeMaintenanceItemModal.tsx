import { useEffect, useMemo, useState } from 'react'
import { api } from '../../api'
import { useAppStore } from '../../stores/appStore'
import { useI18nStore } from '../../stores/i18nStore'
import { Icon } from '../Icons'
import NumberInput from '../NumberInput'
import Select, { ADD_OPTION_VALUE, addOption } from '../Select'
import { useToast } from '../Toast'
import type {
  HomeMaintenanceItem,
  HomeMaintenanceItemType,
  HomeMaintenanceUnit,
} from '../../types/homeMaintenance'
import type { Institution, InstitutionCategory } from '../../types'

const UNITS: HomeMaintenanceUnit[] = ['unit', 'm', 'm2', 'kg', 'lb', 'l', 'gal', 'sack', 'box']
const PRIORITY_CATEGORIES = new Set(['hardware', 'maintenance'])

export default function HomeMaintenanceItemModal({
  homeId,
  maintenanceId,
  itemType,
  item,
  onSaved,
  onCancel,
}: {
  homeId: number
  maintenanceId: number
  itemType: HomeMaintenanceItemType
  item?: HomeMaintenanceItem | null
  onSaved: () => void
  onCancel: () => void
}) {
  const { t } = useI18nStore()
  const { currencies, loadCurrencies } = useAppStore()
  const { showToast } = useToast()
  const isEditing = !!item
  const isSupplies = itemType === 'supplies'

  const [description, setDescription] = useState(item?.description ?? '')
  const [quantity, setQuantity] = useState<number | ''>(item?.quantity ?? 1)
  const [unit, setUnit] = useState<HomeMaintenanceUnit>(item?.unit ?? 'unit')
  const [amount, setAmount] = useState<number | ''>(item ? item.amount : '')
  const [currencyId, setCurrencyId] = useState<number | ''>(item?.currency_id ?? '')
  const [institutionId, setInstitutionId] = useState<number | ''>(item?.institution_id ?? '')
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [quickOpen, setQuickOpen] = useState(false)
  const [quickName, setQuickName] = useState('')
  const [quickCategoryId, setQuickCategoryId] = useState<number | ''>('')
  const [quickSaving, setQuickSaving] = useState(false)
  const [quickError, setQuickError] = useState('')

  useEffect(() => {
    loadCurrencies()
    Promise.all([api.institutions.list(), api.institutionCategories.list()])
      .then(([institutionList, categoryList]) => {
        setInstitutions(institutionList || [])
        setCategories(categoryList || [])
      })
      .finally(() => setLoading(false))
  }, [loadCurrencies])

  useEffect(() => {
    if (currencyId === '' && currencies.length > 0) {
      setCurrencyId(currencies[0].id)
    }
  }, [currencies, currencyId])

  const title = isEditing
    ? t('homeMaintenance.edit_item')
    : isSupplies
      ? t('homeMaintenance.add_supply')
      : t('homeMaintenance.add_labor')

  const categoryKeyById = useMemo(() => {
    const map = new Map<number, string>()
    for (const category of categories) map.set(category.id, category.key)
    return map
  }, [categories])

  const institutionOptions = useMemo(() => {
    const sorted = [...institutions].sort((a, b) => {
      const aPriority = PRIORITY_CATEGORIES.has(categoryKeyById.get(a.category_id ?? -1) ?? '') ? 0 : 1
      const bPriority = PRIORITY_CATEGORIES.has(categoryKeyById.get(b.category_id ?? -1) ?? '') ? 0 : 1
      if (aPriority !== bPriority) return aPriority - bPriority
      return a.name.localeCompare(b.name)
    })
    return [
      ...sorted.map((institution) => ({ value: institution.id, label: institution.name })),
      addOption(t('app.add')),
    ]
  }, [institutions, categoryKeyById, t])

  const categoryOptions = useMemo(
    () => [
      { value: '', label: t('institutions.uncategorized') },
      ...categories.map((category) => ({ value: category.id, label: category.name })),
    ],
    [categories, t],
  )

  const openQuickCreate = () => {
    const preferred = categories.find(
      (category) => category.key === (isSupplies ? 'hardware' : 'maintenance'),
    )
    setQuickName('')
    setQuickCategoryId(preferred ? preferred.id : '')
    setQuickError('')
    setQuickOpen(true)
  }

  const handleQuickCreate = async () => {
    const name = quickName.trim()
    if (!name) {
      setQuickError(t('institutions.name_required'))
      return
    }
    setQuickSaving(true)
    setQuickError('')
    try {
      const created = await api.institutions.create({
        name,
        category_id: quickCategoryId === '' ? null : Number(quickCategoryId),
      })
      const list = await api.institutions.list()
      setInstitutions(list || [])
      if (created) {
        setInstitutionId(created.id)
        showToast(t('institutions.created'), 'success')
      }
      setQuickOpen(false)
    } catch (err) {
      setQuickError(err instanceof Error ? err.message : t('errors.generic'))
    } finally {
      setQuickSaving(false)
    }
  }

  const unitOptions = UNITS.map((value) => ({
    value,
    label: t(`homeMaintenance.unit_${value}`),
  }))
  const currencyOptions = currencies.map((currency) => ({
    value: currency.id,
    label: `${currency.code} (${currency.symbol})`,
  }))

  const handleSubmit = async () => {
    setError('')
    if (!description.trim()) {
      setError(t('homeMaintenance.item_description_required'))
      return
    }
    if (amount === '' || amount < 0) {
      setError(t('homeMaintenance.item_amount_invalid'))
      return
    }
    if (!currencyId) {
      setError(t('homeMaintenance.currency_required'))
      return
    }
    if (isSupplies && (quantity === '' || quantity <= 0)) {
      setError(t('homeMaintenance.quantity_invalid'))
      return
    }

    setSubmitting(true)
    try {
      const body = {
        item_type: itemType,
        description: description.trim(),
        quantity: isSupplies ? Number(quantity) : 1,
        unit: isSupplies ? unit : ('unit' as HomeMaintenanceUnit),
        amount: Number(amount),
        currency_id: Number(currencyId),
        institution_id: institutionId === '' ? null : Number(institutionId),
      }
      if (isEditing && item) {
        await api.homeMaintenance.updateItem(homeId, maintenanceId, item.id, body)
      } else {
        await api.homeMaintenance.createItem(homeId, maintenanceId, body)
      }
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.generic'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
      <div className="bg-card rounded-ios shadow-ios w-full max-w-md max-h-[90vh] flex flex-col">
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

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {loading ? (
            <p className="text-text-secondary text-center py-6">{t('app.loading')}</p>
          ) : (
            <>
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('homeMaintenance.item_description')}
                </label>
                <input
                  type="text"
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                  placeholder={t('homeMaintenance.item_description_placeholder')}
                  className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] placeholder:text-text-secondary"
                  autoFocus
                />
              </div>

              {isSupplies && (
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-sm font-medium mb-1">{t('homeMaintenance.quantity')}</label>
                    <NumberInput
                      min={0}
                      step="0.01"
                      value={quantity}
                      onValueChange={(value) => setQuantity(value)}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">{t('homeMaintenance.unit')}</label>
                    <Select
                      options={unitOptions}
                      value={unit}
                      onChange={(value) => setUnit(String(value) as HomeMaintenanceUnit)}
                      placeholder={t('homeMaintenance.unit')}
                      searchable
                      searchPlaceholder={t('app.search')}
                      emptyLabel={t('app.no_results')}
                    />
                  </div>
                </div>
              )}

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium mb-1">{t('homeMaintenance.amount')}</label>
                  <NumberInput
                    min={0}
                    step="0.01"
                    value={amount}
                    onValueChange={(value) => setAmount(value)}
                    placeholder="0.00"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">{t('homeMaintenance.currency')}</label>
                  <Select
                    options={currencyOptions}
                    value={currencyId}
                    onChange={(value) => setCurrencyId(Number(value))}
                    placeholder={t('homeMaintenance.select_currency')}
                    searchable
                    searchPlaceholder={t('app.search')}
                    emptyLabel={t('app.no_results')}
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  {isSupplies ? t('homeMaintenance.item_provider') : t('homeMaintenance.item_contractor')}
                </label>
                <Select
                  options={institutionOptions}
                  value={institutionId}
                  onChange={(value) => {
                    if (value === ADD_OPTION_VALUE) {
                      openQuickCreate()
                      return
                    }
                    setInstitutionId(value === '' ? '' : Number(value))
                  }}
                  placeholder={t('homeMaintenance.select_institution')}
                  searchable
                  searchPlaceholder={t('app.search')}
                  emptyLabel={t('app.no_results')}
                />
                <p className="text-xs text-text-secondary mt-1">
                  {isSupplies ? t('homeMaintenance.item_provider_hint') : t('homeMaintenance.item_contractor_hint')}
                </p>
              </div>

              {error && <p className="text-sm text-danger">{error}</p>}
            </>
          )}
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
            disabled={submitting || loading}
            className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
          >
            {submitting ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </div>
    </div>

    {quickOpen && (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[70] p-4">
        <div className="bg-card rounded-ios shadow-ios w-full max-w-sm max-h-[90vh] flex flex-col">
          <div className="p-4 border-b border-border flex items-center justify-between gap-3">
            <h3 className="text-lg font-bold">{t('homeMaintenance.add_institution')}</h3>
            <button
              type="button"
              onClick={() => setQuickOpen(false)}
              className="p-2 text-text-secondary hover:text-text rounded-full transition-colors"
              aria-label={t('app.cancel')}
            >
              <Icon name="cancel" className="w-5 h-5" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('institutions.name')}</label>
              <input
                type="text"
                value={quickName}
                onChange={(event) => setQuickName(event.target.value)}
                placeholder={t('institutions.name_placeholder')}
                className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px] placeholder:text-text-secondary"
                autoFocus
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('institutions.category')}</label>
              <Select
                options={categoryOptions}
                value={quickCategoryId}
                onChange={(value) => setQuickCategoryId(value === '' ? '' : Number(value))}
                placeholder={t('institutions.select_category')}
                searchable
                searchPlaceholder={t('app.search')}
                emptyLabel={t('app.no_results')}
              />
            </div>
            {quickError && <p className="text-sm text-danger">{quickError}</p>}
          </div>

          <div className="p-4 border-t border-border flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setQuickOpen(false)}
              className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
            >
              {t('app.cancel')}
            </button>
            <button
              type="button"
              onClick={handleQuickCreate}
              disabled={quickSaving}
              className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
            >
              {quickSaving ? t('app.saving') : t('app.save')}
            </button>
          </div>
        </div>
      </div>
    )}
    </>
  )
}
