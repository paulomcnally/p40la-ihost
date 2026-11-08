import { useEffect, useState } from 'react'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { Icon } from './Icons'
import Select from './Select'
import AutoAttachmentsField from './AutoAttachmentsField'
import type { AutoAttachment, AutoInspection, AutoInspectionResult, Institution, InstitutionCategory } from '../types'

const MAX_ATTACHMENTS = 20

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
  const [savedId, setSavedId] = useState<number | null>(inspection?.id ?? null)
  const [attachments, setAttachments] = useState<AutoAttachment[]>(inspection?.attachments ?? [])
  const [pendingFiles, setPendingFiles] = useState<File[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [activeFormTab, setActiveFormTab] = useState<'info' | 'invoices'>('info')

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

  const handleUpload = async (files: File[]) => {
    if (!savedId) {
      setPendingFiles((prev) => [...prev, ...files].slice(0, MAX_ATTACHMENTS - attachments.length))
      return
    }
    for (const file of files) {
      const attachment = await api.autos.uploadInspectionAttachment(autoId, savedId, file)
      if (attachment) setAttachments((prev) => [...prev, attachment])
    }
  }

  const handleDeleteAttachment = async (attachment: AutoAttachment) => {
    if (!savedId) return
    await api.autos.deleteInspectionAttachment(autoId, savedId, attachment.id)
    setAttachments((prev) => prev.filter((item) => item.id !== attachment.id))
  }

  const handleSubmit = async () => {
    setError('')
    if (!performedAt) {
      setActiveFormTab('info')
      setError(t('autos.inspections.date_required'))
      return
    }
    if (!institutionId) {
      setActiveFormTab('info')
      setError(t('autos.inspections.institution_required'))
      return
    }

    setSubmitting(true)
    let savedRecord: AutoInspection | null = null
    try {
      const body = {
        performed_at: performedAt,
        inspection_year: inspectionYear,
        institution_id: Number(institutionId),
        result,
        notes: notes.trim(),
      }
      savedRecord = savedId
        ? await api.autos.updateInspection(autoId, savedId, body)
        : await api.autos.createInspection(autoId, body)
      if (!savedRecord) throw new Error(t('autos.inspections.save_error'))

      setSavedId(savedRecord.id)
      setAttachments(savedRecord.attachments ?? [])
      for (const file of pendingFiles) {
        const attachment = await api.autos.uploadInspectionAttachment(autoId, savedRecord.id, file)
        if (attachment) setAttachments((prev) => [...prev, attachment])
      }
      setPendingFiles([])
      onSaved()
    } catch (err) {
      if (savedRecord) {
        setSavedId(savedRecord.id)
        setAttachments(savedRecord.attachments ?? [])
      }
      setError(err instanceof Error ? err.message : t('autos.inspections.save_error'))
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
            </>
          ) : (
            <>
              <AutoAttachmentsField
                attachments={attachments}
                attachmentUrl={(attachment) =>
                  savedId
                    ? api.autos.inspectionAttachmentUrl(autoId, savedId, attachment.id)
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
              if (savedId && savedId !== inspection?.id) onSaved()
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
