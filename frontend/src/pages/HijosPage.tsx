import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import { calcularEdad } from '../utils/age'
import type { Child } from '../types'

export default function HijosPage() {
  const navigate = useNavigate()
  const { t, lang } = useI18nStore()
  usePageTitle(t('hijos.title'))
  const [children, setChildren] = useState<Child[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  const loadChildren = useCallback(async () => {
    const data = await api.children.list()
    setChildren(data || [])
  }, [])

  useEffect(() => {
    loadChildren().finally(() => setLoading(false))
  }, [])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.children.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadChildren()
  }, [deleteTarget])

  if (loading) {
    return <LoadingSpinner />
  }

  if (children.length === 0) {
    return (
      <EmptyCard
        icon="baby"
        title={t('hijos.empty')}
        subtitle={t('hijos.subtitle')}
        actionLabel={t('hijos.create')}
        onAction={() => navigate('/pension/hijos/new')}
      />
    )
  }

  const createOptions = [
    { label: t('hijos.create'), icon: 'plus', onClick: () => navigate('/pension/hijos/new') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('hijos.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {children.map((child) => {
          const age = calcularEdad(child.birth_date)
          return (
            <div
              key={child.id}
              onClick={() => {}}
              className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
            >
              <CardMenu
                options={[
                  { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/pension/hijos/edit/${child.id}`) },
                  { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: child.id, name: `${child.first_name} ${child.last_name}` }) },
                ]}
              />
              <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center mb-3">
                <Icon name="baby" className="w-6 h-6" />
              </div>
              <h3 className="font-semibold text-base">{child.first_name} {child.last_name}</h3>
              <p className="text-sm text-text-secondary mt-1">
                {formatAge(age, t)} · {formatDate(child.birth_date, lang)}
              </p>
              {child.notes && (
                <p className="text-sm text-text-secondary mt-2 line-clamp-2">{child.notes}</p>
              )}
            </div>
          )
        })}
      </div>
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('hijos.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

function formatAge(age: number, t: (key: string, fallback?: string) => string): string {
  return `${age} ${age === 1 ? t('hijos.year') : t('hijos.years')}`
}

function formatDate(dateStr: string, lang: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  if (!y || !m || !d) return dateStr
  const date = new Date(y, m - 1, d)
  return date.toLocaleDateString(lang, { day: 'numeric', month: 'short', year: 'numeric' })
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