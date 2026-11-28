import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import LoadingSpinner from '../components/LoadingSpinner'
import CardMenu from '../components/CardMenu'
import { Icon } from '../components/Icons'
import { Badge, Money, PrimaryButton, SectionCard } from '../components/rental/RentalUI'
import HomeMaintenanceTab from '../components/home/HomeMaintenanceTab'
import HomeWorkRequestTab from '../components/home/HomeWorkRequestTab'
import HomeFinancesTab from '../components/home/HomeFinancesTab'
import HomeStateBadge from '../components/home/HomeStateBadge'
import type { Home } from '../types'
import type { HomeRentalStatus } from '../types/rental'

type Tab = 'summary' | 'maintenance' | 'rental' | 'finances' | 'reports'

export default function HomeShowPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const homeId = Number(id)
  const { t } = useI18nStore()
  const [searchParams, setSearchParams] = useSearchParams()
  const rawTab = searchParams.get('tab')
  const tab: Tab = rawTab === 'rental' ? 'rental'
    : rawTab === 'maintenance' ? 'maintenance'
      : rawTab === 'finances' ? 'finances'
        : rawTab === 'reports' ? 'reports'
          : 'summary'
  const [home, setHome] = useState<Home | null>(null)
  const [status, setStatus] = useState<HomeRentalStatus | null>(null)
  const [loading, setLoading] = useState(true)

  usePageTitle(home ? home.name : t('home.title'))

  const load = useCallback(async () => {
    if (!Number.isFinite(homeId) || homeId <= 0) {
      setHome(null)
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const [h, s] = await Promise.all([api.homes.get(homeId), api.rental.homes.status(homeId)])
      setHome(h)
      setStatus(s)
    } catch {
      setHome(null)
    } finally {
      setLoading(false)
    }
  }, [homeId])

  useEffect(() => { load() }, [load])

  const setTab = (next: Tab) => {
    const params = new URLSearchParams(searchParams)
    if (next === 'summary') params.delete('tab')
    else params.set('tab', next)
    setSearchParams(params, { replace: true })
  }

  if (loading) return <LoadingSpinner />

  if (!home) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <h3 className="text-lg font-semibold mb-2">{t('rental.home_not_found')}</h3>
        <PrimaryButton onClick={() => navigate('/home')}>{t('rental.back_to_homes')}</PrimaryButton>
      </div>
    )
  }

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu options={[
          { label: t('app.edit'), onClick: () => navigate(`/home/edit/${home.id}`) },
        ]} />
        <div className="flex items-start gap-3">
          <div className="w-12 h-12 rounded-ios bg-primary/10 text-primary flex items-center justify-center shrink-0">
            <Icon name="home" className="w-6 h-6" />
          </div>
          <div className="min-w-0 pr-10">
            <h2 className="text-xl font-bold truncate">{home.name}</h2>
            {home.address && <p className="text-sm text-text-secondary mt-0.5">{home.address}</p>}
            {status && (
              <div className="mt-2">
                <HomeStateBadge state={status.listing_status} />
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="flex gap-1 p-1 bg-bg rounded-ios-sm overflow-x-auto" role="tablist">
        {([
          { key: 'summary' as Tab, label: t('rental.tab_summary') },
          { key: 'rental' as Tab, label: t('rental.tab_rental') },
          { key: 'finances' as Tab, label: t('home.finances.tab') },
          { key: 'maintenance' as Tab, label: t('homeMaintenance.tab') },
          { key: 'reports' as Tab, label: t('homeWorkRequest.tab') },
        ]).map((item) => (
          <button
            key={item.key}
            type="button"
            role="tab"
            aria-selected={tab === item.key}
            onClick={() => setTab(item.key)}
            className={`flex-1 min-w-[88px] px-2 py-2 rounded-ios-sm text-xs sm:text-sm font-medium whitespace-nowrap min-h-[40px] transition-colors ${
              tab === item.key ? 'bg-card text-primary shadow-ios' : 'text-text-secondary hover:text-text'
            }`}
          >
            {item.label}
          </button>
        ))}
      </div>

      {tab === 'summary' ? (
        <SectionCard title={t('rental.summary.title')}>
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
            <div>
              <dt className="text-text-secondary">{t('home.name')}</dt>
              <dd className="font-medium mt-0.5">{home.name}</dd>
            </div>
            <div>
              <dt className="text-text-secondary">{t('home.address')}</dt>
              <dd className="font-medium mt-0.5">{home.address || '—'}</dd>
            </div>
          </dl>
        </SectionCard>
      ) : tab === 'maintenance' ? (
        <HomeMaintenanceTab homeId={homeId} />
      ) : tab === 'reports' ? (
        <HomeWorkRequestTab homeId={homeId} />
      ) : tab === 'finances' ? (
        <HomeFinancesTab homeId={homeId} />
      ) : (
        <div className="space-y-4">
          {status?.usage && status.usage !== 'rental' ? (
            <SectionCard title={t('rental.rental_title')}>
              <div className="border-2 border-dashed border-border rounded-ios p-6 text-center">
                <p className="font-medium">{t('home.rental_blocked')}</p>
                <p className="text-sm text-text-secondary mt-1">{t('home.rental_blocked_hint')}</p>
                <div className="mt-4 flex justify-center">
                  <PrimaryButton onClick={() => navigate(`/home/edit/${home.id}`)}>
                    <Icon name="edit" className="w-4 h-4" />
                    {t('app.edit')}
                  </PrimaryButton>
                </div>
              </div>
            </SectionCard>
          ) : status?.active_lease ? (
            <SectionCard
              title={t('rental.lease.current')}
              action={<Badge tone="info">{t(`rental.lease_status.${status.active_lease.status}`)}</Badge>}
            >
              <div className="space-y-2 text-sm">
                <p className="font-medium">{status.active_lease.start_date} → {status.active_lease.end_date}</p>
                <p className="text-text-secondary">
                  {t('rental.rent')}: <Money amount={status.active_lease.rent_amount} currency={status.active_lease.currency} />
                  {status.active_lease.maintenance_amount > 0 && (
                    <> · {t('rental.maintenance')}: <Money amount={status.active_lease.maintenance_amount} currency={status.active_lease.currency} /></>
                  )}
                </p>
                {status.active_lease.parties && status.active_lease.parties.length > 0 && (
                  <p className="text-text-secondary">
                    {t('rental.tenant')}: {status.active_lease.parties.filter(p => p.role === 'tenant').map(p => p.person?.full_name).filter(Boolean).join(', ')}
                  </p>
                )}
                <PrimaryButton onClick={() => navigate(`/leases/${status.active_lease!.id}`)}>
                  {t('rental.lease.open')}
                </PrimaryButton>
              </div>
            </SectionCard>
          ) : status?.active_reservation ? (
            <SectionCard
              title={t('rental.reservation.current')}
              action={<Badge tone="warning">{t('rental.reservation_status.active')}</Badge>}
            >
              <div className="space-y-2 text-sm">
                <p className="font-medium">{status.active_reservation.applicant?.full_name}</p>
                <p className="text-text-secondary">
                  {t('rental.reservation.expires')}: {new Date(status.active_reservation.expires_at).toLocaleString()}
                </p>
                <p className="text-text-secondary">
                  {t('rental.reservation.amount')}: <Money amount={status.active_reservation.amount} currency={status.active_reservation.currency} />
                </p>
                <PrimaryButton onClick={() => navigate(`/reservations/${status.active_reservation!.id}`)}>
                  {t('rental.reservation.open')}
                </PrimaryButton>
              </div>
            </SectionCard>
          ) : (
            <SectionCard title={t('rental.rental_title')}>
              <div className="border-2 border-dashed border-border rounded-ios p-6 text-center">
                <p className="font-medium">{t('rental.empty_rental')}</p>
                <p className="text-sm text-text-secondary mt-1">{t('rental.empty_rental_hint')}</p>
                <div className="mt-4 flex flex-col sm:flex-row gap-2 justify-center">
                  <PrimaryButton onClick={() => navigate(`/leases/new?home=${home.id}`)}>
                    <Icon name="plus" className="w-4 h-4" />
                    {t('rental.lease.create')}
                  </PrimaryButton>
                  <button
                    type="button"
                    onClick={() => navigate(`/reservations/new?home=${home.id}`)}
                    className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-ios-sm border border-border font-medium hover:bg-bg min-h-[44px]"
                  >
                    {t('rental.reservation.create')}
                  </button>
                </div>
              </div>
            </SectionCard>
          )}

          {status?.upcoming_charges && status.upcoming_charges.length > 0 && (
            <SectionCard title={t('rental.upcoming_charges')}>
              <ul className="divide-y divide-border">
                {status.upcoming_charges.map((c) => (
                  <li key={c.id} className="py-2 flex items-center justify-between gap-3 text-sm">
                    <div className="min-w-0">
                      <p className="font-medium truncate">{c.description}</p>
                      <p className="text-xs text-text-secondary">{t('rental.due')}: {c.due_date}</p>
                    </div>
                    <Money amount={c.amount} currency={c.currency} />
                  </li>
                ))}
              </ul>
            </SectionCard>
          )}

          {status?.reminders && status.reminders.length > 0 && (
            <SectionCard title={t('rental.reminders')}>
              <ul className="divide-y divide-border">
                {status.reminders.map((r) => (
                  <li key={r.id} className="py-2 text-sm">
                    <p className="font-medium">{r.title}</p>
                    <p className="text-xs text-text-secondary">{new Date(r.next_due_at).toLocaleString()}</p>
                  </li>
                ))}
              </ul>
            </SectionCard>
          )}
        </div>
      )}
    </div>
  )
}
