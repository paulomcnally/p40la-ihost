import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'
import Select from '../components/Select'
import type { Institution, InstitutionCategory } from '../types'
import type {
  HomeMaintenanceArea,
  HomeMaintenanceKind,
  HomeMaintenanceInput,
} from '../types/homeMaintenance'

const KINDS: HomeMaintenanceKind[] = ['corrective', 'preventive']
const AREAS: HomeMaintenanceArea[] = [
  'plumbing', 'electrical', 'roof', 'paint', 'carpentry',
  'appliances', 'garden', 'cleaning', 'other',
]
const PRIORITY_CATEGORIES = new Set(['hardware', 'maintenance'])

function todayISO(): string {
  const now = new Date()
  const offset = now.getTimezoneOffset()
  return new Date(now.getTime() - offset * 60 * 1000).toISOString().slice(0, 10)
}

export default function HomeMaintenanceFormPage() {
  const navigate = useNavigate()
  const { id, maintenanceId } = useParams()
  const homeId = Number(id)
  const isEdit = !!maintenanceId
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { currencies, loadCurrencies } = useAppStore()

  const [homeName, setHomeName] = useState('')
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [performedAt, setPerformedAt] = useState(todayISO())
  const [kind, setKind] = useState<HomeMaintenanceKind>('corrective')
  const [area, setArea] = useState<HomeMaintenanceArea>('other')
  const [institutionId, setInstitutionId] = useState<number | ''>('')
  const [description, setDescription] = useState('')
  const [currencyId, setCurrencyId] = useState<number | ''>('')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  usePageTitle(
    isEdit
      ? t('homeMaintenance.edit_title')
      : t('homeMaintenance.new'),
  )

  useEffect(() => {
    loadCurrencies()
    const tasks: Promise<unknown>[] = [
      api.institutions.list().then((list) => setInstitutions(list || [])),
      api.institutionCategories.list().then((list) => setCategories(list || [])),
      api.homes.get(homeId).then((home) => setHomeName(home?.name ?? '')),
    ]
    if (isEdit) {
      tasks.push(
        api.homeMaintenance.get(homeId, Number(maintenanceId)).then((record) => {
          if (!record) return
          setPerformedAt(record.performed_at)
          setKind(record.kind)
          setArea(record.area)
          setInstitutionId(record.institution_id ?? '')
          setDescription(record.description)
          setCurrencyId(record.currency_id)
        }),
      )
    }
    Promise.all(tasks).finally(() => setLoading(false))
  }, [homeId, isEdit, maintenanceId, loadCurrencies])

  useEffect(() => {
    if (!isEdit && currencyId === '' && currencies.length > 0) {
      setCurrencyId(currencies[0].id)
    }
  }, [currencies, currencyId, isEdit])

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
      { value: '', label: t('homeMaintenance.no_provider') },
      ...sorted.map((institution) => ({ value: institution.id, label: institution.name })),
    ]
  }, [institutions, categoryKeyById, t])

  const kindOptions = KINDS.map((value) => ({
    value,
    label: t(`homeMaintenance.kind_${value}`),
  }))
  const areaOptions = AREAS.map((value) => ({
    value,
    label: t(`homeMaintenance.area_${value}`),
  }))
  const currencyOptions = currencies.map((currency) => ({
    value: currency.id,
    label: `${currency.code} (${currency.symbol})`,
  }))

  const backToHome = () => navigate(`/home/${homeId}?tab=maintenance`)

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    if (!performedAt) {
      setError(t('homeMaintenance.date_required'))
      return
    }
    if (performedAt > todayISO()) {
      setError(t('homeMaintenance.date_future'))
      return
    }
    if (!currencyId) {
      setError(t('homeMaintenance.currency_required'))
      return
    }

    setSubmitting(true)
    try {
      const body: HomeMaintenanceInput = {
        performed_at: performedAt,
        kind,
        area,
        institution_id: institutionId === '' ? null : Number(institutionId),
        description: description.trim(),
        currency_id: Number(currencyId),
      }
      const saved = isEdit
        ? await api.homeMaintenance.update(homeId, Number(maintenanceId), body)
        : await api.homeMaintenance.create(homeId, body)
      if (!saved) throw new Error(t('errors.generic'))
      showToast(t('homeMaintenance.saved_ok'), 'success')
      navigate(`/home/${homeId}/maintenances/${saved.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.generic'))
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <form onSubmit={handleSubmit} className="max-w-2xl mx-auto space-y-4">
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex items-center gap-3">
          <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <Icon name="wrench" className="w-5 h-5" />
          </div>
          <div className="min-w-0">
            <h2 className="text-lg font-bold truncate">
              {isEdit ? t('homeMaintenance.edit_title') : t('homeMaintenance.new')}
            </h2>
            {homeName && <p className="text-sm text-text-secondary truncate">{homeName}</p>}
          </div>
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6 space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('homeMaintenance.date')}</label>
            <input
              type="date"
              value={performedAt}
              max={todayISO()}
              onChange={(event) => setPerformedAt(event.target.value)}
              className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary min-h-[44px]"
              required
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
          <div>
            <label className="block text-sm font-medium mb-1">{t('homeMaintenance.kind')}</label>
            <Select
              options={kindOptions}
              value={kind}
              onChange={(value) => setKind(String(value) as HomeMaintenanceKind)}
              placeholder={t('homeMaintenance.kind')}
              emptyLabel={t('app.no_results')}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('homeMaintenance.area')}</label>
            <Select
              options={areaOptions}
              value={area}
              onChange={(value) => setArea(String(value) as HomeMaintenanceArea)}
              placeholder={t('homeMaintenance.area')}
              searchable
              searchPlaceholder={t('app.search')}
              emptyLabel={t('app.no_results')}
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('homeMaintenance.provider')}</label>
          <Select
            options={institutionOptions}
            value={institutionId}
            onChange={(value) => setInstitutionId(value === '' ? '' : Number(value))}
            placeholder={t('homeMaintenance.select_provider')}
            searchable
            searchPlaceholder={t('app.search')}
            emptyLabel={t('app.no_results')}
          />
          <p className="text-xs text-text-secondary mt-1">{t('homeMaintenance.provider_hint')}</p>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('homeMaintenance.description')}</label>
          <textarea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder={t('homeMaintenance.description_placeholder')}
            rows={3}
            className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary placeholder:text-text-secondary"
          />
        </div>

        {error && <p className="text-sm text-danger">{error}</p>}
      </div>

      <div className="flex justify-end gap-3">
        <button
          type="button"
          onClick={backToHome}
          className="px-4 py-2 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors min-h-[44px]"
        >
          {t('app.cancel')}
        </button>
        <button
          type="submit"
          disabled={submitting}
          className="px-4 py-2 bg-primary text-white rounded-ios-sm hover:bg-primary-hover disabled:opacity-50 transition-colors min-h-[44px]"
        >
          {submitting ? t('app.saving') : t('app.save')}
        </button>
      </div>
    </form>
  )
}
