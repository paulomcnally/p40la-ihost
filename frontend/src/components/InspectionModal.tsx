import { useEffect, useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import Select from './Select'
import type { AutoInspection, AutoInspectionResult, Institution, InstitutionCategory } from '../types'

function todayISO(): string {
  const now = new Date()
  const offset = now.getTimezoneOffset()
  return new Date(now.getTime() - offset * 60 * 1000).toISOString().slice(0, 10)
}

export default function InspectionModal({
  autoId,
  autoYear,
  inspection,
  nextDueYear,
  onSaved,
  onCancel,
}: {
  autoId: number
  autoYear: number
  inspection?: AutoInspection | null
  nextDueYear?: number
  onSaved: () => void
  onCancel: () => void
}) {
  const { t } = useI18nStore()
  const isEditing = !!inspection

  const firstDueYear = autoYear + 2
  const currentYear = new Date().getFullYear()
  const highestYear = Math.max(
    currentYear + 1,
    firstDueYear,
    nextDueYear ?? firstDueYear,
    inspection?.inspection_year ?? firstDueYear,
  )
  const yearOptions = []
  for (let year = firstDueYear; year <= highestYear; year++) {
    yearOptions.push({ value: year, label: String(year) })
  }

  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [performedAt, setPerformedAt] = useState(inspection?.performed_at ?? todayISO())
  const [inspectionYear, setInspectionYear] = useState<number>(
    inspection?.inspection_year ?? nextDueYear ?? firstDueYear,
  )
  const [institutionId, setInstitutionId] = useState<number | ''>(inspection?.institution_id ?? '')
  const [result, setResult] = useState<AutoInspectionResult>(inspection?.result ?? 'approved')
  const [notes, setNotes] = useState(inspection?.notes ?? '')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([api.institutions.list(), api.institutionCategories.list()])
      .then(([instList, catList]) => {
        setInstitutions(instList || [])
        setCategories(catList || [])
      })
      .finally(() => setLoading(false))
  }, [])

  const inspectionCategory = categories.find((category) => category.key === 'mechanical_inspection')
  const centers = inspectionCategory
    ? institutions.filter((institution) => institution.category_id === inspectionCategory.id)
    : []
  const centerOptions = centers.map((institution) => ({
    value: institution.id,
    label: institution.name,
  }))
  const resultOptions = [
    { value: 'approved', label: t('autos.inspections.result_approved') },
    { value: 'rejected', label: t('autos.inspections.result_rejected') },
  ]

  const handleSubmit = async () => {
    setError('')
    if (!performedAt) {
      setError(t('autos.inspections.date_required'))
      return
    }
    if (!institutionId) {
      setError(t('autos.inspections.institution_required'))
      return
    }

    setSubmitting(true)
    try {
      const body = {
        performed_at: performedAt,
        inspection_year: inspectionYear,
        institution_id: Number(institutionId),
        result,
        notes: notes.trim(),
      }
      const saved = isEditing && inspection
        ? await api.autos.updateInspection(autoId, inspection.id, body)
        : await api.autos.createInspection(autoId, body)
      if (!saved) throw new Error(t('autos.inspections.save_error'))
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('autos.inspections.save_error'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
      <div
        className="bg-card rounded-ios shadow-ios w-full max-w-lg max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-border flex items-center justify-between gap-3">
          <h3 className="text-lg font-bold">
            {isEditing ? t('autos.inspections.edit') : t('autos.inspections.add')}
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
                <label className="block text-sm font-medium mb-1">{t('autos.inspections.date')}</label>
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
                <label className="block text-sm font-medium mb-1">{t('autos.inspections.cycle_year')}</label>
                <Select
                  options={yearOptions}
                  value={inspectionYear}
                  onChange={(value) => setInspectionYear(Number(value))}
                  placeholder={t('autos.inspections.select_year')}
                />
                <p className="text-xs text-text-secondary mt-1">
                  {t('autos.inspections.first_due_hint').replace('{year}', String(firstDueYear))}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.inspections.institution')}</label>
                <Select
                  options={centerOptions}
                  value={institutionId}
                  onChange={(value) => setInstitutionId(value === '' ? '' : Number(value))}
                  placeholder={t('autos.inspections.select_institution')}
                  searchable
                  searchPlaceholder={t('app.search')}
                  emptyLabel={t('app.no_results')}
                />
                {centers.length === 0 && (
                  <p className="text-xs text-text-secondary mt-1">
                    {t('autos.inspections.no_centers')}{' '}
                    <a href="/institutions/new" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                      {t('autos.inspections.create_institution')}
                    </a>
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.inspections.result')}</label>
                <Select
                  options={resultOptions}
                  value={result}
                  onChange={(value) => setResult(value as AutoInspectionResult)}
                  placeholder={t('autos.inspections.select_result')}
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">{t('autos.inspections.notes')}</label>
                <textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder={t('autos.inspections.notes_placeholder')}
                  rows={2}
                  className="w-full px-3 py-2 bg-card text-text border border-border rounded-ios-sm focus:outline-none focus:border-primary placeholder:text-text-secondary"
                />
              </div>

              <p className="text-xs text-text-secondary">{t('autos.inspections.detail_hint')}</p>
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
