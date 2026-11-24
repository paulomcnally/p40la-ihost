import { useEffect, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import LoadingSpinner from '../components/LoadingSpinner'
import Select, { addOption } from '../components/Select'
import NumberInput from '../components/NumberInput'
import PhoneInput from '../components/PhoneInput'
import Toggle from '../components/Toggle'
import { Icon } from '../components/Icons'
import { Field, PrimaryButton, SecondaryButton, SectionCard, TextArea, TextInput } from '../components/rental/RentalUI'
import type { RentalPerson, Reservation } from '../types/rental'

function toLocalInput(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export default function ReservationFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const isEdit = Boolean(id)
  const queryHomeID = Number(searchParams.get('home') || 0)
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { homes, loadHomes, currencies, loadCurrencies } = useAppStore()

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [people, setPeople] = useState<RentalPerson[]>([])
  const [form, setForm] = useState({
    home_id: Number(searchParams.get('home') || 0),
    applicant_person_id: 0,
    full_name: '',
    email: '',
    phone: '',
    agent_person_id: 0,
    status: 'active' as Reservation['status'],
    amount: 0,
    currency: 'USD',
    reference: '',
    applied_to_deposit: true,
    starts_at: toLocalInput(new Date().toISOString()),
    expires_at: toLocalInput(new Date(Date.now() + 15 * 86400000).toISOString()),
    contract_date: '',
    move_in_date: '',
    rent_amount: 0,
    maintenance_amount: 0,
    owner_split_percent: 50,
    agency_split_percent: 50,
    refund_policy: 'forfeitable' as Reservation['refund_policy'],
    required_documents: '',
    notes: '',
  })

  usePageTitle(isEdit ? t('rental.reservation.edit') : t('rental.reservation.create'))

  useEffect(() => {
    loadHomes()
    loadCurrencies()
    api.rental.people.list().then((list) => setPeople(list || [])).catch(() => undefined)
  }, [])

  useEffect(() => {
    if (!isEdit) { setLoading(false); return }
    const load = async () => {
      try {
        const r = await api.rental.reservations.get(Number(id))
        if (!r) return
        setForm({
          home_id: r.home_id,
          applicant_person_id: r.applicant_person_id,
          full_name: r.applicant?.full_name || '',
          email: r.applicant?.email || '',
          phone: r.applicant?.phone || '',
          agent_person_id: r.agent_person_id || 0,
          status: r.status,
          amount: r.amount,
          currency: r.currency,
          reference: r.reference,
          applied_to_deposit: r.applied_to_deposit,
          starts_at: toLocalInput(r.starts_at),
          expires_at: toLocalInput(r.expires_at),
          contract_date: r.contract_date,
          move_in_date: r.move_in_date,
          rent_amount: r.rent_amount,
          maintenance_amount: r.maintenance_amount,
          owner_split_percent: r.owner_split_percent,
          agency_split_percent: r.agency_split_percent,
          refund_policy: r.refund_policy,
          required_documents: r.required_documents,
          notes: r.notes,
        })
      } catch (err: unknown) {
        showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [id])

  const set = <K extends keyof typeof form>(key: K, value: (typeof form)[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }))

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.home_id) {
      showToast(t('rental.lease.home_required'), 'error')
      return
    }
    setSaving(true)
    try {
      let applicantID = form.applicant_person_id
      if (!applicantID) {
        if (!form.full_name.trim()) {
          showToast(t('rental.reservation.applicant_required'), 'error')
          setSaving(false)
          return
        }
        const person = await api.rental.people.create({
          full_name: form.full_name.trim(), email: form.email.trim(), phone: form.phone.trim(),
        })
        if (!person) return
        applicantID = person.id
      }
      const payload: Partial<Reservation> = {
        home_id: form.home_id,
        applicant_person_id: applicantID,
        agent_person_id: form.agent_person_id || null,
        amount: form.amount,
        currency: form.currency,
        reference: form.reference,
        applied_to_deposit: form.applied_to_deposit,
        starts_at: new Date(form.starts_at).toISOString(),
        expires_at: new Date(form.expires_at).toISOString(),
        contract_date: form.contract_date,
        move_in_date: form.move_in_date,
        rent_amount: form.rent_amount,
        maintenance_amount: form.maintenance_amount,
        owner_split_percent: form.owner_split_percent,
        agency_split_percent: form.agency_split_percent,
        refund_policy: form.refund_policy,
        required_documents: form.required_documents,
        notes: form.notes,
      }
      const saved = isEdit
        ? await api.rental.reservations.update(Number(id), payload)
        : await api.rental.reservations.create(payload)
      if (saved) {
        showToast(isEdit ? t('rental.reservation.updated') : t('rental.reservation.created'), 'success')
        navigate(`/reservations/${saved.id}`)
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  const homeOptions = homes.filter((h) => h.usage === 'rental').map((h) => ({ value: h.id, label: h.name }))
  const lockedHomeID = isEdit ? form.home_id : queryHomeID
  const selectedHome = homes.find((h) => h.id === lockedHomeID)
  const agentOptions = [{ value: 0, label: t('rental.none') }, ...people.map((p) => ({ value: p.id, label: p.full_name }))]
  const currencyOptions = (currencies.length > 0 ? currencies.map((c) => c.code) : ['USD', 'NIO', 'EUR'])
    .map((code) => ({ value: code, label: code }))

  return (
    <form onSubmit={submit} className="max-w-3xl mx-auto space-y-4" noValidate>
      <h2 className="text-lg sm:text-xl font-bold">{isEdit ? t('rental.reservation.edit') : t('rental.reservation.create')}</h2>

      <SectionCard title={t('rental.reservation.property')}>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label={t('rental.home')} hint={homeOptions.length === 0 ? t('home.no_rental_homes') : undefined}>
            {lockedHomeID > 0 ? (
              <div className="flex items-center gap-2 min-h-[44px] px-3 py-2 border border-border rounded-ios-sm bg-bg/60 text-text">
                <Icon name="home" className="w-4 h-4 text-text-secondary shrink-0" />
                <span className="truncate font-medium">
                  {selectedHome?.name || `${t('rental.home')} #${lockedHomeID}`}
                </span>
                {selectedHome?.address && (
                  <span className="text-xs text-text-secondary truncate">· {selectedHome.address}</span>
                )}
              </div>
            ) : (
              <Select options={homeOptions} value={form.home_id} onChange={(v) => set('home_id', Number(v))} searchable placeholder={t('home.select')} />
            )}
          </Field>
          <Field label={t('rental.agent')}>
            <Select options={agentOptions} value={form.agent_person_id} onChange={(v) => set('agent_person_id', Number(v))} searchable />
          </Field>
        </div>
      </SectionCard>

      <SectionCard title={t('rental.reservation.applicant')}>
        {form.applicant_person_id > 0 ? (
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">{form.full_name}</p>
            <button
              type="button"
              className="text-sm text-primary"
              onClick={() => set('applicant_person_id', 0)}
            >
              {t('rental.reservation.change_applicant')}
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field label={t('rental.person')}>
              <Select
                options={[addOption(t('app.add'), 0), ...people.map((p) => ({ value: p.id, label: p.full_name }))]}
                value={0}
                onChange={(v) => {
                  const pid = Number(v)
                  const person = people.find((p) => p.id === pid)
                  if (person) {
                    setForm((prev) => ({
                      ...prev, applicant_person_id: person.id,
                      full_name: person.full_name, email: person.email, phone: person.phone,
                    }))
                  }
                }}
                searchable
              />
            </Field>
            <div className="hidden sm:block" />
            <Field label={t('rental.person_name')}>
              <TextInput value={form.full_name} onChange={(e) => set('full_name', e.target.value)} required />
            </Field>
            <Field label={t('rental.email')}>
              <TextInput type="email" value={form.email} onChange={(e) => set('email', e.target.value)} />
            </Field>
            <Field label={t('rental.phone')}>
              <PhoneInput value={form.phone} onChange={(v) => set('phone', v)} />
            </Field>
          </div>
        )}
      </SectionCard>

      <SectionCard title={t('rental.reservation.conditions')}>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label={t('rental.reservation.amount')}>
            <NumberInput  min={0} step="0.01" value={form.amount} onChange={(e) => set('amount', Number(e.target.value))} />
          </Field>
          <Field label={t('rental.currency')}>
            <Select options={currencyOptions} value={form.currency} onChange={(v) => set('currency', String(v))} searchable />
          </Field>
          <Field label={t('rental.reservation.reference')} hint={t('rental.reservation.reference_hint')}>
            <TextInput value={form.reference} onChange={(e) => set('reference', e.target.value)} />
          </Field>
          <Field label={t('rental.reservation.refund_policy')}>
            <Select
              options={[
                { value: 'forfeitable', label: t('rental.refund.forfeitable') },
                { value: 'refundable', label: t('rental.refund.refundable') },
                { value: 'custom', label: t('rental.refund.custom') },
              ]}
              value={form.refund_policy}
              onChange={(v) => set('refund_policy', v as Reservation['refund_policy'])}
            />
          </Field>
          <Field label={t('rental.reservation.starts')}>
            <TextInput type="datetime-local" value={form.starts_at} onChange={(e) => set('starts_at', e.target.value)} required />
          </Field>
          <Field label={t('rental.reservation.expires')}>
            <TextInput type="datetime-local" value={form.expires_at} onChange={(e) => set('expires_at', e.target.value)} required />
          </Field>
          <Field label={t('rental.reservation.contract_date')}>
            <TextInput type="date" value={form.contract_date} onChange={(e) => set('contract_date', e.target.value)} />
          </Field>
          <Field label={t('rental.reservation.move_in')}>
            <TextInput type="date" value={form.move_in_date} onChange={(e) => set('move_in_date', e.target.value)} />
          </Field>
          <Field label={t('rental.rent')}>
            <NumberInput  min={0} step="0.01" value={form.rent_amount} onChange={(e) => set('rent_amount', Number(e.target.value))} />
          </Field>
          <Field label={t('rental.maintenance')}>
            <NumberInput  min={0} step="0.01" value={form.maintenance_amount} onChange={(e) => set('maintenance_amount', Number(e.target.value))} />
          </Field>
          <Field label={t('rental.reservation.owner_split')}>
            <NumberInput  min={0} max={100} step="0.01" value={form.owner_split_percent} onChange={(e) => set('owner_split_percent', Number(e.target.value))} />
          </Field>
          <Field label={t('rental.reservation.agency_split')}>
            <NumberInput  min={0} max={100} step="0.01" value={form.agency_split_percent} onChange={(e) => set('agency_split_percent', Number(e.target.value))} />
          </Field>
        </div>
        <div className="flex items-center justify-between gap-4 mt-4 pt-4 border-t border-border">
          <div>
            <p className="text-sm font-medium">{t('rental.reservation.applied_to_deposit')}</p>
            <p className="text-xs text-text-secondary">{t('rental.reservation.applied_to_deposit_hint')}</p>
          </div>
          <Toggle checked={form.applied_to_deposit} onChange={(v) => set('applied_to_deposit', v)} ariaLabel={t('rental.reservation.applied_to_deposit')} />
        </div>
        <div className="grid gap-4 mt-4">
          <Field label={t('rental.reservation.required_documents')} hint={t('rental.reservation.required_documents_hint')}>
            <TextArea rows={2} value={form.required_documents} onChange={(e) => set('required_documents', e.target.value)} />
          </Field>
          <Field label={t('rental.notes')}>
            <TextArea rows={3} value={form.notes} onChange={(e) => set('notes', e.target.value)} />
          </Field>
        </div>
      </SectionCard>

      <div className="flex justify-end gap-2">
        <SecondaryButton type="button" onClick={() => navigate(-1)}>
          <Icon name="cancel" className="w-4 h-4" />
          {t('app.cancel')}
        </SecondaryButton>
        <PrimaryButton type="submit" disabled={saving}>
          <Icon name="save" className="w-4 h-4" />
          {saving ? t('app.saving') : t('app.save')}
        </PrimaryButton>
      </div>
    </form>
  )
}
