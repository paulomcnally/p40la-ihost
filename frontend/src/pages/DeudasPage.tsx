import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import DeleteModal from '../components/DeleteModal'
import LoadingSpinner from '../components/LoadingSpinner'
import DebtCalendar from '../components/DebtCalendar'
import DebtAnalysis from '../components/DebtAnalysis'
import type { Debt, Institution } from '../types'

type TabKey = 'calendario' | 'deudas' | 'analisis'

export default function DeudasPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { currencies, loadAll } = useAppStore()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = (searchParams.get('tab') as TabKey | null) ?? 'calendario'

  const [debts, setDebts] = useState<Debt[]>([])
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [loading, setLoading] = useState(true)

  const loadDebts = useCallback(async () => {
    const list = await api.debts.list()
    setDebts(list || [])
  }, [])

  useEffect(() => {
    loadAll()
    api.institutions
      .list()
      .then((list) => setInstitutions(list || []))
      .catch(() => setInstitutions([]))
  }, [])

  useEffect(() => {
    loadDebts().finally(() => setLoading(false))
  }, [loadDebts])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.debts.delete(deleteTarget.id)
    setDeleteTarget(null)
    loadDebts()
  }, [deleteTarget, loadDebts])

  const setTab = (key: TabKey) => {
    if (key === 'calendario') setSearchParams({}, { replace: false })
    else setSearchParams({ tab: key })
  }

  if (loading) return <LoadingSpinner />

  if (institutions.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="building" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.empty_no_institution')}</h3>
        <button
          onClick={() => navigate('/institutions/new')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
        >
          <Icon name="plus" className="w-5 h-5" />
          {'Nueva institución'}
        </button>
      </div>
    )
  }

  if (currencies.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="savings" className="w-full h-full" />
        </div>
        <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.empty_no_currency')}</h3>
        <button
          onClick={() => navigate('/settings/currency')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
        >
          <Icon name="plus" className="w-5 h-5" />
          {'Nueva moneda'}
        </button>
      </div>
    )
  }

  const createOptions = [
    { label: t('deudas.create'), icon: 'plus', onClick: () => navigate('/deudas/new') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">{t('deudas.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>

      <div className="flex gap-2 mb-4">
        {(
          [
            { key: 'calendario', label: t('deudas.tab_calendar') },
            { key: 'deudas', label: t('deudas.tab_debts') },
            { key: 'analisis', label: t('deudas.tab_analysis') },
          ] as { key: TabKey; label: string }[]
        ).map((item) => (
          <button
            key={item.key}
            onClick={() => setTab(item.key)}
            className={`px-4 py-2 rounded-ios-sm text-sm font-medium transition-colors min-h-[44px] ${
              tab === item.key
                ? 'bg-primary text-white'
                : 'bg-card text-text-secondary hover:bg-border'
            }`}
          >
            {item.label}
          </button>
        ))}
      </div>

      {tab === 'calendario' ? (
        <DebtCalendar />
      ) : tab === 'analisis' ? (
        <DebtAnalysis />
      ) : debts.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-8 sm:p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="credit" className="w-full h-full" />
          </div>
          <h3 className="text-lg sm:text-xl font-semibold mb-2">{t('deudas.empty')}</h3>
          <p className="text-text-secondary mb-6">{t('deudas.subtitle')}</p>
          <button
            onClick={() => navigate('/deudas/new')}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('deudas.create')}
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {debts.map((debt) => {
            const currency = currencies.find((c) => c.id === debt.currency_id)
            return (
              <div
                key={debt.id}
                onClick={() => navigate(`/deudas/${debt.id}`)}
                className={`rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow ${
                  debt.status === 'activa' ? 'bg-card' : 'bg-gray-100 opacity-60 dark:bg-[#2c2c2e]'
                }`}
              >
                <CardMenu
                  options={[
                    { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/deudas/edit/${debt.id}`) },
                    { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: debt.id, name: debt.description }) },
                  ]}
                />
                <div className="flex items-center justify-between mb-3">
                  <div className={`w-11 h-11 rounded-ios flex items-center justify-center ${
                    debt.status === 'activa' ? 'bg-primary/10 text-primary' : 'bg-gray-200 text-gray-400 dark:bg-[#2c2c2e] dark:text-gray-500'
                  }`}>
                    <Icon name="credit" className="w-6 h-6" />
                  </div>
                </div>
                <h3 className="font-semibold text-base">{debt.description}</h3>
                <p className="text-sm text-text-secondary mt-1">
                  {debt.institution_name}
                  {debt.identifier && ` · ${debt.identifier}`}
                </p>
                <div className="flex items-center gap-2 mt-2 flex-wrap">
                  {debt.status !== 'activa' && (
                    <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-gray-200 text-gray-600 dark:bg-[#2c2c2e] dark:text-gray-400">
                      {t(`deudas.status_${debt.status}`)}
                    </span>
                  )}
                  <span className="text-xs text-text-secondary">
                    {debt.installments_total} {t('deudas.installment').toLowerCase()}(s) · día {debt.payment_day}
                  </span>
                </div>
                <div className="flex items-center justify-between mt-1">
                  <p className="text-sm font-semibold">
                    {currency?.symbol}{debt.installment_amount.toFixed(2)}/{t('deudas.installment').toLowerCase()}
                  </p>
                  <span className="text-xs text-text-secondary">
                    {debt.currency_code} {debt.total.toFixed(2)}
                  </span>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('deudas.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}