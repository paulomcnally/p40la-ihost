import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import type { AnalyzerInfo, Institution, InstitutionCategory } from '../types'

interface AssignedAnalyzer {
  id: string
  name: string
}

export default function InstitutionShowPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const institutionId = Number(id)
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [institution, setInstitution] = useState<Institution | null>(null)
  const [category, setCategory] = useState<InstitutionCategory | null>(null)
  const [analyzers, setAnalyzers] = useState<AssignedAnalyzer[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteOpen, setDeleteOpen] = useState(false)

  usePageTitle(institution ? institution.name : t('institutions.title'))

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      if (!Number.isFinite(institutionId) || institutionId <= 0) {
        setInstitution(null)
        setLoading(false)
        return
      }

      try {
        const [inst, assigned, available, categories] = await Promise.all([
          api.institutions.get(institutionId),
          api.institutions.getAnalyzers(institutionId).catch(() => []),
          api.analyzers.list().catch(() => []),
          api.institutionCategories.list().catch(() => []),
        ])
        if (cancelled) return

        setInstitution(inst)
        const analyzerNames = new Map<string, string>(
          (available || []).map((a: AnalyzerInfo): [string, string] => [a.id, a.name])
        )
        setAnalyzers(
          (assigned || []).map((a) => ({
            id: a.analyzer_id,
            name: analyzerNames.get(a.analyzer_id) || a.analyzer_id,
          }))
        )
        setCategory((categories || []).find((c) => c.id === inst?.category_id) || null)
      } catch {
        if (!cancelled) {
          setInstitution(null)
          setAnalyzers([])
          setCategory(null)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [institutionId])

  const handleDelete = async () => {
    try {
      await api.institutions.delete(institutionId)
      showToast(t('institutions.deleted'), 'success')
      navigate('/institutions')
    } catch (err: unknown) {
      setDeleteOpen(false)
      const raw = err instanceof Error ? err.message : ''
      const message = /FOREIGN KEY|constraint failed/i.test(raw)
        ? t('institutions.delete_in_use')
        : raw || t('errors.generic')
      showToast(message, 'error')
    }
  }

  if (loading) return <LoadingSpinner />

  if (!institution) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <div className="w-14 h-14 mx-auto mb-4 text-text-secondary opacity-60">
          <Icon name="building" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('institutions.not_found')}</h3>
        <p className="text-text-secondary mb-5">{t('institutions.not_found_hint')}</p>
        <button
          type="button"
          onClick={() => navigate('/institutions')}
          className="inline-flex items-center justify-center px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px]"
        >
          {t('institutions.back_to_list')}
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu
          options={[
            { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/institutions/edit/${institution.id}`) },
            { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteOpen(true) },
          ]}
        />

        <div className="flex items-start gap-3 min-w-0 pr-12">
          <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <Icon name={category?.icon_key || 'building'} className="w-6 h-6" />
          </div>
          <div className="min-w-0">
            <h2 className="text-lg sm:text-xl font-bold truncate">{institution.name}</h2>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 mt-1.5">
              {category ? (
                <span className="inline-block text-xs font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                  {category.name}
                </span>
              ) : (
                <span className="text-sm text-text-secondary">{t('institutions.uncategorized')}</span>
              )}
              <span className="text-sm text-text-secondary flex items-center gap-1.5">
                <Icon name="pdf" className="w-4 h-4" />
                {analyzerCountText(analyzers.length, t)}
              </span>
            </div>
          </div>
        </div>
      </div>

      <section className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex items-center justify-between gap-3 mb-3">
          <h3 className="font-semibold">{t('institutions.assigned_analyzers')}</h3>
          {analyzers.length > 0 && (
            <span className="text-xs text-text-secondary">{analyzers.length}</span>
          )}
        </div>

        {analyzers.length === 0 ? (
          <div>
            <p className="text-sm text-text-secondary">{t('institutions.no_analyzers')}</p>
            <p className="text-sm text-text-secondary mt-1">{t('institutions.no_analyzers_hint')}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {analyzers.map((analyzer) => (
              <div
                key={analyzer.id}
                className="flex items-center gap-2 bg-bg rounded-ios-sm px-3 py-2.5"
              >
                <Icon name="pdf" className="w-4 h-4 text-primary shrink-0" />
                <span className="text-sm font-medium truncate">{analyzer.name}</span>
              </div>
            ))}
          </div>
        )}
      </section>

      {deleteOpen && (
        <DeleteModal
          title={t('institutions.delete_title')}
          subtitle={t('institutions.delete_subtitle').replace('{name}', institution.name)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteOpen(false)}
        />
      )}
    </div>
  )
}

function analyzerCountText(count: number, t: (key: string, fallback?: string) => string): string {
  if (count === 1) return t('institutions.analyzer_count_one')
  return t('institutions.analyzer_count_other').replace('{count}', String(count))
}
