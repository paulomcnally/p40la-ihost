import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import InstitutionCategoriesModal from '../components/InstitutionCategoriesModal'
import type { Institution, InstitutionCategory } from '../types'

export default function InstitutionsPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  usePageTitle(t('institutions.title'))
  const { showToast } = useToast()
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [analyzerCounts, setAnalyzerCounts] = useState<Record<number, number>>({})
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [showCategoriesModal, setShowCategoriesModal] = useState(false)

  const loadData = useCallback(async () => {
    try {
      const [instList, catList] = await Promise.all([
        api.institutions.list(),
        api.institutionCategories.list(),
      ])
      setInstitutions(instList || [])
      setCategories(catList || [])
      const counts: Record<number, number> = {}
      for (const inst of instList || []) {
        const list = await api.institutions.getAnalyzers(inst.id)
        counts[inst.id] = (list || []).length
      }
      setAnalyzerCounts(counts)
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await api.institutions.delete(deleteTarget.id)
      setDeleteTarget(null)
      showToast(t('institutions.deleted'), 'success')
      loadData()
    } catch (err: unknown) {
      const raw = err instanceof Error ? err.message : ''
      const message = /FOREIGN KEY|constraint failed/i.test(raw)
        ? t('institutions.delete_in_use')
        : raw || t('errors.generic')
      showToast(message, 'error')
    }
  }, [deleteTarget, loadData])

  if (institutions.length === 0) {
    return (
      <EmptyCard
        icon="building"
        title={t('institutions.empty_title')}
        subtitle={t('institutions.empty_subtitle')}
        actionLabel={t('institutions.new')}
        onAction={() => navigate('/institutions/new')}
      />
    )
  }

  const createOptions = [
    { label: t('institutions.new'), icon: 'plus', onClick: () => navigate('/institutions/new') },
    { label: t('institutions.categories'), icon: 'tag', onClick: () => setShowCategoriesModal(true) },
  ]

  const getCategoryName = (categoryId?: number) => {
    if (!categoryId) return null
    return categories.find(c => c.id === categoryId)?.name || null
  }

  const getCategoryIcon = (categoryId?: number) => {
    if (!categoryId) return null
    return categories.find(c => c.id === categoryId)?.icon_key || 'other'
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('institutions.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {institutions.map((inst) => {
          const count = analyzerCounts[inst.id] || 0
          const catName = getCategoryName(inst.category_id)
          const catIcon = getCategoryIcon(inst.category_id)
          return (
            <div
              key={inst.id}
              onClick={() => navigate(`/institutions/${inst.id}`)}
              className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
            >
              <CardMenu
                options={[
                  { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/institutions/edit/${inst.id}`) },
                  { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: inst.id, name: inst.name }) },
                ]}
              />
              <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
                <Icon name={catIcon || 'building'} className="w-6 h-6" />
              </div>
              <h3 className="font-semibold text-base truncate">{inst.name}</h3>
              {catName && (
                <span className="inline-block mt-1 text-xs font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                  {catName}
                </span>
              )}
              <p className="text-sm text-text-secondary mt-1 flex items-center gap-1.5">
                {count > 0 ? (
                  <>
                    <Icon name="pdf" className="w-4 h-4" />
                    {analyzerCountText(count, t)}
                  </>
                ) : (
                  t('institutions.no_analyzers_configured')
                )}
              </p>
            </div>
          )
        })}
      </div>
      {deleteTarget && (
        <DeleteModal
          title={t('institutions.delete_title')}
          subtitle={t('institutions.delete_subtitle').replace('{name}', deleteTarget.name)}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
      <InstitutionCategoriesModal isOpen={showCategoriesModal} onClose={() => setShowCategoriesModal(false)} />
    </div>
  )
}

function analyzerCountText(count: number, t: (key: string, fallback?: string) => string): string {
  if (count === 1) return t('institutions.analyzer_count_one')
  return t('institutions.analyzer_count_other').replace('{count}', String(count))
}

function EmptyCard({ icon, title, subtitle, actionLabel, onAction }: {
  icon: string; title: string; subtitle: string; actionLabel: string; onAction: () => void
}) {
  return (
    <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
      <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
        <Icon name={icon} className="w-full h-full" />
      </div>
      <h3 className="text-lg sm:text-xl font-semibold mb-2">{title}</h3>
      <p className="text-text-secondary mb-6">{subtitle}</p>
      <button
        onClick={onAction}
        className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
      >
        <Icon name="plus" className="w-5 h-5" />
        {actionLabel}
      </button>
    </div>
  )
}
