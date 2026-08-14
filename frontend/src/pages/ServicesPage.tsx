import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { api } from '../api'
import { Icon } from '../components/Icons'
import CreateMenu from '../components/CreateMenu'
import CardMenu from '../components/CardMenu'
import Select from '../components/Select'
import DeleteModal from '../components/DeleteModal'
import type { Service } from '../types'

export default function ServicesPage() {
  const navigate = useNavigate()
  const { homes, currencies, loadAll } = useAppStore()
  const { t } = useI18nStore()
  const [services, setServices] = useState<Service[]>([])
  const [homeFilter, setHomeFilter] = useState<number | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)

  useEffect(() => {
    loadAll()
  }, [])

  useEffect(() => {
    const load = async () => {
      const list = await api.services.list(homeFilter ?? undefined)
      setServices(list || [])
    }
    load()
  }, [homeFilter])

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return
    await api.services.delete(deleteTarget.id)
    setDeleteTarget(null)
    const list = await api.services.list(homeFilter ?? undefined)
    setServices(list || [])
  }, [deleteTarget, homeFilter])

  if (homes.length === 0) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-12 text-center max-w-md mx-auto mt-8">
        <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
          <Icon name="home" className="w-full h-full" />
        </div>
        <h3 className="text-xl font-semibold mb-2">{t('services.empty_no_home')}</h3>
        <p className="text-text-secondary mb-6">{t('services.subtitle')}</p>
        <button
          onClick={() => navigate('/home/new')}
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors min-w-48"
        >
          <Icon name="plus" className="w-5 h-5" />
          {t('home.create')}
        </button>
      </div>
    )
  }

  const createOptions = [
    { label: t('services.create'), icon: 'plus', onClick: () => navigate('/services/new') },
  ]

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-2xl font-bold">{t('services.title')}</h2>
        <CreateMenu options={createOptions} />
      </div>
      <div className="mb-5 max-w-xs">
        <Select
          options={[
            { value: '', label: t('services.all_homes') },
            ...homes.map(h => ({ value: String(h.id), label: h.name })),
          ]}
          value={homeFilter ?? ''}
          onChange={(v) => setHomeFilter(v ? Number(v) : null)}
          placeholder={t('services.all_homes')}
          searchable
        />
      </div>
      {services.length === 0 ? (
        <div className="bg-card rounded-ios shadow-ios p-12 text-center max-w-md mx-auto">
          <div className="w-16 h-16 mx-auto mb-5 text-primary opacity-80">
            <Icon name="services" className="w-full h-full" />
          </div>
          <h3 className="text-xl font-semibold mb-2">{t('services.empty')}</h3>
          <p className="text-text-secondary mb-6">{t('services.subtitle')}</p>
          <button
            onClick={() => navigate('/services/new')}
            className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-card border-2 border-dashed border-border rounded-ios text-primary font-semibold hover:border-primary hover:bg-primary/5 transition-colors"
          >
            <Icon name="plus" className="w-5 h-5" />
            {t('services.create')}
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {services.map((svc) => {
            const currency = currencies.find(c => c.id === svc.currency_id)
            const home = homes.find(h => h.id === svc.home_id)
            return (
              <div
                key={svc.id}
                onClick={() => navigate(`/services/bills/${svc.id}`)}
                className="bg-card rounded-ios shadow-ios p-4 relative cursor-pointer hover:shadow-ios-lg transition-shadow"
              >
                <CardMenu
                  options={[
                    { label: t('app.edit'), icon: 'edit', onClick: () => navigate(`/services/edit/${svc.id}`) },
                    { label: t('app.delete'), icon: 'delete', danger: true, onClick: () => setDeleteTarget({ id: svc.id, name: svc.name }) },
                  ]}
                />
                <div className="flex items-center justify-between mb-3">
                  <div className="w-11 h-11 rounded-ios bg-primary/10 text-primary flex items-center justify-center">
                    <Icon name={svc.icon_key || 'other'} className="w-6 h-6" />
                  </div>
                  <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                    svc.active ? 'bg-success/20 text-green-800' : 'bg-warning/20 text-yellow-800'
                  }`}>
                    {svc.active ? t('bills.status_paid') : t('bills.status_pending')}
                  </span>
                </div>
                <h3 className="font-semibold text-base">{svc.name}</h3>
                <p className="text-sm text-text-secondary mt-1">
                  {svc.institution && `${svc.institution} · `}{home?.name || ''}
                </p>
                <p className="text-xs text-text-secondary mt-1">
                  {currency?.symbol}{svc.suggested_amount.toFixed(2)} · {t(`frequency.${svc.frequency}`)}
                </p>
              </div>
            )
          })}
        </div>
      )}
      {deleteTarget && (
        <DeleteModal
          title={t('app.confirm')}
          subtitle={`${t('services.title')}: ${deleteTarget.name}`}
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
