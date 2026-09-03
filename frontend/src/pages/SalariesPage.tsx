import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import type { Salary } from '../types'

export default function SalariesPage() {
  const navigate = useNavigate()
  const { currencies, loadCurrencies } = useAppStore()
  const { t } = useI18nStore()
  const [salaries, setSalaries] = useState<Salary[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadCurrencies().finally(() => setLoading(false))
    loadSalaries()
  }, [])

  const loadSalaries = useCallback(async () => {
    const data = await api.salaries.list()
    setSalaries(data || [])
  }, [])

  useEffect(() => {
    loadSalaries()
  }, [loadSalaries])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.salaries.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadSalaries()
  }, [deleteTarget, loadSalaries])

  if (loading) {
    return <LoadingSpinner />
  }

  if (salaries.length === 0) {
    return (
      <EmptyCard
        icon="savings"
        title={t('salaries.empty')}
        subtitle={t('salaries.subtitle')}
        actionLabel={t('salaries.create')}
        onAction={() => navigate('/pension/salarios/new')}
      />
    )
  }

  const createOptions = [
    { label: t('salaries.create'), icon: 'plus', onClick: () => navigate('/pension/salarios/new') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('salaries.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {salaries.map((salary) => {
          const currency = currencies.find(c => c.id === salary.currency_id)
          return (
            <div
              key={salary.id}
              onClick={() => {}}
              className={`rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow ${
                salary.active ? 'bg-card' : 'bg-gray-100 opacity-60 dark:bg-[#2c2c2e]'
              }`}
            >
              <CardMenu
                options={[
                  { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/pension/salarios/edit/${salary.id}`) },
                  { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: salary.id, name: salary.employer }) },
                ]}
              />
              <div className="flex items-center justify-between mb-3">
                <div className={`w-11 h-11 rounded-ios flex items-center justify-center ${
                  salary.active ? 'bg-primary/10 text-primary' : 'bg-gray-200 text-gray-400 dark:bg-[#2c2c2e] dark:text-gray-500'
                }`}>
                  <Icon name="savings" className="w-6 h-6" />
                </div>
                {!salary.active && (
                  <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400">
                    {t('salaries.inactive')}
                  </span>
                )}
              </div>
              <h3 className="font-semibold text-base">{salary.employer}</h3>
              <p className="text-sm text-text-secondary mt-1">
                {currency?.symbol}{salary.amount.toFixed(2)}
              </p>
              <p className="text-xs text-text-secondary mt-1">
                {t('salaries.payment_day')}: {salary.payment_day}
              </p>
              {salary.note && (
                <p className="text-sm text-text-secondary mt-2 line-clamp-2">{salary.note}</p>
              )}
            </div>
          )
        })}
      </div>
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('salaries.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
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