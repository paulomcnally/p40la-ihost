import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import LoadingSpinner from '../components/LoadingSpinner'
import CardMenu from '../components/CardMenu'
import { Icon } from '../components/Icons'
import { Badge, Field, Modal, Money, PrimaryButton, SecondaryButton, SectionCard, TextArea, TextInput } from '../components/rental/RentalUI'
import NumberInput from '../components/NumberInput'
import type { Reservation } from '../types/rental'

export default function ReservationShowPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const reservationId = Number(id)
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [loading, setLoading] = useState(true)
  const [reservation, setReservation] = useState<Reservation | null>(null)
  const [busy, setBusy] = useState(false)
  const [convertOpen, setConvertOpen] = useState(false)
  const [action, setAction] = useState<'cancel' | 'forfeit' | null>(null)
  const [reason, setReason] = useState('')
  const [convertForm, setConvertForm] = useState({ term_months: 12, payment_day: 1, deposit_amount: 0 })

  usePageTitle(reservation ? t('rental.reservation.title') : t('rental.reservation.title'))

  const load = useCallback(async () => {
    if (!Number.isFinite(reservationId) || reservationId <= 0) { setLoading(false); return }
    setLoading(true)
    try {
      const data = await api.rental.reservations.get(reservationId)
      setReservation(data)
      if (data) {
        setConvertForm((prev) => ({
          ...prev,
          deposit_amount: data.applied_to_deposit ? data.amount : 0,
        }))
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [reservationId])

  useEffect(() => { load() }, [load])

  const convert = async () => {
    setBusy(true)
    try {
      const lease = await api.rental.reservations.convert(reservationId, convertForm)
      if (lease) {
        showToast(t('rental.reservation.converted'), 'success')
        navigate(`/leases/${lease.id}`)
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const confirmAction = async () => {
    if (!action) return
    setBusy(true)
    try {
      if (action === 'cancel') await api.rental.reservations.cancel(reservationId, reason)
      else await api.rental.reservations.forfeit(reservationId, reason)
      showToast(t(`rental.reservation.${action === 'cancel' ? 'cancelled' : 'forfeited'}`), 'success')
      setAction(null)
      setReason('')
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <LoadingSpinner />
  if (!reservation) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <h3 className="text-lg font-semibold mb-3">{t('rental.reservation.not_found')}</h3>
        <PrimaryButton onClick={() => navigate('/home')}>{t('rental.back_to_homes')}</PrimaryButton>
      </div>
    )
  }

  const isActive = reservation.status === 'active'
  const menu = isActive ? [{ label: t('app.edit'), onClick: () => navigate(`/reservations/${reservation.id}/edit`) }] : []

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu options={menu} />
        <div className="pr-10">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-xl font-bold">{t('rental.reservation.title')} #{reservation.id}</h2>
            <Badge tone={isActive ? 'warning' : reservation.status === 'converted' ? 'success' : 'neutral'}>
              {t(`rental.reservation_status.${reservation.status}`)}
            </Badge>
          </div>
          <p className="text-sm text-text-secondary mt-1">{reservation.home_name}{reservation.home_address ? ` — ${reservation.home_address}` : ''}</p>
          <p className="font-medium mt-3">{reservation.applicant?.full_name}</p>
          <p className="text-sm text-text-secondary">{reservation.applicant?.email} {reservation.applicant?.phone}</p>
        </div>
      </div>

      {isActive && (
        <div className="flex flex-wrap gap-2 justify-end">
          <SecondaryButton onClick={() => window.open(api.rental.reservations.pdfUrl(reservation.id), '_blank')}>
            <Icon name="pdf" className="w-4 h-4" />
            {t('rental.reservation.pdf')}
          </SecondaryButton>
          <SecondaryButton onClick={() => setAction('forfeit')}>{t('rental.reservation.forfeit')}</SecondaryButton>
          <SecondaryButton onClick={() => setAction('cancel')}>{t('rental.reservation.cancel')}</SecondaryButton>
          <PrimaryButton onClick={() => setConvertOpen(true)}>
            <Icon name="check" className="w-4 h-4" />
            {t('rental.reservation.convert')}
          </PrimaryButton>
        </div>
      )}

      {reservation.converted_lease_id && (
        <SectionCard title={t('rental.reservation.lease_created')}>
          <PrimaryButton onClick={() => navigate(`/leases/${reservation.converted_lease_id}`)}>
            {t('rental.lease.open')}
          </PrimaryButton>
        </SectionCard>
      )}

      <SectionCard title={t('rental.reservation.conditions')}>
        <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
          <div><dt className="text-text-secondary">{t('rental.reservation.amount')}</dt><dd className="font-medium mt-0.5"><Money amount={reservation.amount} currency={reservation.currency} /></dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.reference')}</dt><dd className="font-medium mt-0.5">{reservation.reference || '—'}</dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.starts')}</dt><dd className="font-medium mt-0.5">{new Date(reservation.starts_at).toLocaleString()}</dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.expires')}</dt><dd className="font-medium mt-0.5">{new Date(reservation.expires_at).toLocaleString()}</dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.contract_date')}</dt><dd className="font-medium mt-0.5">{reservation.contract_date || '—'}</dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.move_in')}</dt><dd className="font-medium mt-0.5">{reservation.move_in_date || '—'}</dd></div>
          <div><dt className="text-text-secondary">{t('rental.rent')}</dt><dd className="font-medium mt-0.5"><Money amount={reservation.rent_amount} currency={reservation.currency} /></dd></div>
          <div><dt className="text-text-secondary">{t('rental.maintenance')}</dt><dd className="font-medium mt-0.5"><Money amount={reservation.maintenance_amount} currency={reservation.currency} /></dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.split')}</dt><dd className="font-medium mt-0.5">{reservation.owner_split_percent}% / {reservation.agency_split_percent}%</dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.refund_policy')}</dt><dd className="font-medium mt-0.5">{t(`rental.refund.${reservation.refund_policy}`)}</dd></div>
          <div><dt className="text-text-secondary">{t('rental.reservation.applied_to_deposit')}</dt><dd className="font-medium mt-0.5">{reservation.applied_to_deposit ? t('app.yes') : t('app.no')}</dd></div>
          <div><dt className="text-text-secondary">{t('rental.agent')}</dt><dd className="font-medium mt-0.5">{reservation.agent_name || '—'}</dd></div>
        </dl>
        {reservation.required_documents && (
          <div className="mt-4 pt-4 border-t border-border">
            <p className="text-sm font-medium mb-1">{t('rental.reservation.required_documents')}</p>
            <p className="text-sm text-text-secondary whitespace-pre-line">{reservation.required_documents}</p>
          </div>
        )}
        {reservation.notes && (
          <div className="mt-4 pt-4 border-t border-border">
            <p className="text-sm font-medium mb-1">{t('rental.notes')}</p>
            <p className="text-sm text-text-secondary whitespace-pre-line">{reservation.notes}</p>
          </div>
        )}
      </SectionCard>

      <Modal
        open={convertOpen}
        title={t('rental.reservation.convert')}
        onClose={() => setConvertOpen(false)}
        footer={(
          <>
            <SecondaryButton onClick={() => setConvertOpen(false)}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy} onClick={convert}>{t('rental.reservation.convert')}</PrimaryButton>
          </>
        )}
      >
        <p className="text-sm text-text-secondary">{t('rental.reservation.convert_hint')}</p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Field label={t('rental.term_months')}>
            <NumberInput  min={1} max={120} value={convertForm.term_months} onChange={(e) => setConvertForm((p) => ({ ...p, term_months: Number(e.target.value) }))} />
          </Field>
          <Field label={t('rental.payment_day')}>
            <NumberInput  min={1} max={28} value={convertForm.payment_day} onChange={(e) => setConvertForm((p) => ({ ...p, payment_day: Number(e.target.value) }))} />
          </Field>
          <Field label={t('rental.deposit')}>
            <NumberInput  min={0} step="0.01" value={convertForm.deposit_amount} onChange={(e) => setConvertForm((p) => ({ ...p, deposit_amount: Number(e.target.value) }))} />
          </Field>
        </div>
      </Modal>

      <Modal
        open={action !== null}
        title={action === 'cancel' ? t('rental.reservation.cancel') : t('rental.reservation.forfeit')}
        onClose={() => { setAction(null); setReason('') }}
        footer={(
          <>
            <SecondaryButton onClick={() => { setAction(null); setReason('') }}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy} onClick={confirmAction}>{t('app.confirm')}</PrimaryButton>
          </>
        )}
      >
        <p className="text-sm text-text-secondary">
          {action === 'cancel' ? t('rental.reservation.cancel_hint') : t('rental.reservation.forfeit_hint')}
        </p>
        <Field label={t('rental.reason')}>
          <TextArea rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
        </Field>
      </Modal>
    </div>
  )
}
