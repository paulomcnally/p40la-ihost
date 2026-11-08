import { useEffect, useState } from 'react'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import Select from './Select'
import type { AutoMaintenance, Institution, InstitutionCategory } from '../types'

function todayISO(): string {
  const now = new Date()
  const offset = now.getTimezoneOffset()
  return new Date(now.getTime() - offset * 60 * 1000).toISOString().slice(0, 10)
}

function trimNumber(value: string): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
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
  const { currencies, loadCurrencies } = useAppStore()
  const isEditing = !!maintenance

  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [performedAt, setPerformedAt] = useState(maintenance?.performed_at ?? todayISO())
  const [mileage, setMileage] = useState(maintenance ? String(maintenance.mileage) : '')
  const [institutionId, setInstitutionId] = useState<number | ''>(maintenance?.institution_id ?? '')
  const [description, setDescription] = useState(maintenance?.description ?? '')
  const [currencyId, setCurrencyId] = useState<number | ''>(maintenance?.currency_id ?? '')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

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

  const handleSubmit = async () => {
    setError('')
    if (!performedAt) {
      setError(t('autos.maintenances.date_required'))
      return
    }
    if (mileage === '' || trimNumber(mileage) < 0) {
      setError(t('autos.maintenances.mileage_required'))
      return
    }
    if (!institutionId) {
      setError(t('autos.maintenances.institution_required'))
      return
    }
    if (!currencyId) {
      setError(t('autos.maintenances.currency_required'))
      return
    }

    setSubmitting(true)
    try {
      const body = {
        performed_at: performedAt,
        mileage: trimNumber(mileage),
        institution_id: Number(institutionId),
        description: description.trim(),
        currency_id: Number(currencyId),
      }
      const saved = isEditing && maintenance
        ? await api.autos.updateMaintenance(autoId, maintenance.id, body)
        : await api.autos.createMaintenance(autoId, body)
      if (!saved) throw new Error(t('autos.maintenances.save_error'))
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

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {loading ? (
            <p className="text-text-secondary text-center py-6">{t('app.loading')}</p>
          ) : (
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

              <p className="text-xs text-text-secondary">{t('autos.maintenances.detail_hint')}</p>
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
  )
}
