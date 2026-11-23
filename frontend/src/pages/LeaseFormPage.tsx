import { useEffect, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import LoadingSpinner from '../components/LoadingSpinner'
import Select, { addOption } from '../components/Select'
import DeleteModal from '../components/DeleteModal'
import NumberInput from '../components/NumberInput'
import Toggle from '../components/Toggle'
import { Icon } from '../components/Icons'
import { Badge, EmptyBlock, Field, Modal, PrimaryButton, SecondaryButton, SectionCard, TextArea, TextInput } from '../components/rental/RentalUI'
import type { Institution, InstitutionCategory } from '../types'
import type { Lease, LeaseParty, PartyRole, RentalPerson } from '../types/rental'

interface PartyDraft {
  id?: number
  person_id?: number
  role: PartyRole
  notify_email: boolean
  full_name: string
  email: string
  phone: string
}

const emptyParty = (role: PartyRole = 'tenant'): PartyDraft => ({
  role, notify_email: true, full_name: '', email: '', phone: '',
})

export default function LeaseFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const isEdit = Boolean(id)
  const queryHomeID = Number(searchParams.get('home') || 0)
  type LeaseTab = 'general' | 'terms' | 'parties'
  const activeTab: LeaseTab = searchParams.get('tab') === 'terms'
    ? 'terms'
    : searchParams.get('tab') === 'parties' ? 'parties' : 'general'
  const setActiveTab = (tab: LeaseTab) => {
    const next = new URLSearchParams(searchParams)
    if (tab === 'general') next.delete('tab')
    else next.set('tab', tab)
    setSearchParams(next, { replace: true })
  }
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { homes, loadHomes, currencies, loadCurrencies } = useAppStore()

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [leaseStatus, setLeaseStatus] = useState('')
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [categories, setCategories] = useState<InstitutionCategory[]>([])
  const [people, setPeople] = useState<RentalPerson[]>([])
  const [agencyModalOpen, setAgencyModalOpen] = useState(false)
  const [agentModalOpen, setAgentModalOpen] = useState(false)
  const [quickAgencyName, setQuickAgencyName] = useState('')
  const [quickAgent, setQuickAgent] = useState({ full_name: '', email: '', phone: '' })
  const [quickSaving, setQuickSaving] = useState(false)
  const [removePartyIndex, setRemovePartyIndex] = useState<number | null>(null)
  const [partyModalOpen, setPartyModalOpen] = useState(false)
  const [partySaving, setPartySaving] = useState(false)
  const [partyForm, setPartyForm] = useState<PartyDraft>(emptyParty('tenant'))
  const [partyMode, setPartyMode] = useState<'new' | 'existing'>('new')
  const [parties, setParties] = useState<PartyDraft[]>([])
  const [initialPartyIds, setInitialPartyIds] = useState<number[]>([])
  const [customTerm, setCustomTerm] = useState(false)
  const [form, setForm] = useState({
    home_id: Number(searchParams.get('home') || 0),
    agency_institution_id: 0,
    agent_person_id: 0,
    start_date: new Date().toISOString().slice(0, 10),
    term_months: 12,
    rent_amount: 0,
    maintenance_amount: 0,
    currency: 'USD',
    payment_day: 1,
    deposit_amount: 0,
    late_fee_type: 'none' as 'none' | 'fixed' | 'percent',
    late_fee_value: 0,
    late_fee_cap: 0,
    auto_renew: false,
    notice_days: 30,
    commission_percent: 0,
    commission_type: 'percent' as 'percent' | 'fixed',
    commission_amount: 0,
    commission_base: 'collected' as 'collected' | 'invoiced',
    owner_bank_account: '',
    notes: '',
  })

  usePageTitle(isEdit ? t('rental.lease.edit') : t('rental.lease.create'))

  useEffect(() => {
    loadHomes()
    loadCurrencies()
    api.institutions.list().then((list) => setInstitutions(list || [])).catch(() => undefined)
    api.institutionCategories.list().then((list) => setCategories(list || [])).catch(() => undefined)
    api.rental.people.list().then((list) => setPeople(list || [])).catch(() => undefined)
  }, [])

  useEffect(() => {
    if (!isEdit) {
      setLoading(false)
      return
    }
    const load = async () => {
      try {
        const lease = await api.rental.leases.get(Number(id))
        if (!lease) return
        setLeaseStatus(lease.status)
        setForm({
          home_id: lease.home_id,
          agency_institution_id: lease.agency_institution_id || 0,
          agent_person_id: lease.agent_person_id || 0,
          start_date: lease.start_date,
          term_months: lease.term_months,
          rent_amount: lease.rent_amount,
          maintenance_amount: lease.maintenance_amount,
          currency: lease.currency,
          payment_day: lease.payment_day,
          deposit_amount: lease.deposit_amount,
          late_fee_type: lease.late_fee_type,
          late_fee_value: lease.late_fee_value,
          late_fee_cap: lease.late_fee_cap,
          auto_renew: lease.auto_renew,
          notice_days: lease.notice_days,
          commission_percent: lease.commission_percent,
          commission_type: lease.commission_type || 'percent',
          commission_amount: lease.commission_amount || 0,
          commission_base: lease.commission_base,
          owner_bank_account: lease.owner_bank_account,
          notes: lease.notes,
        })
        setCustomTerm(![6, 12, 18, 24, 36].includes(lease.term_months))
        const loaded = (lease.parties || []).map((p) => ({
          id: p.id,
          person_id: p.person_id,
          role: p.role,
          notify_email: p.notify_email,
          full_name: p.person?.full_name || '',
          email: p.person?.email || '',
          phone: p.person?.phone || '',
        }))
        setParties(loaded)
        setInitialPartyIds((lease.parties || []).map((p) => p.id))
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

  const closeAgencyModal = () => {
    setAgencyModalOpen(false)
    setQuickAgencyName('')
  }

  const closeAgentModal = () => {
    setAgentModalOpen(false)
    setQuickAgent({ full_name: '', email: '', phone: '' })
  }

  const createAgency = async () => {
    const name = quickAgencyName.trim()
    if (!name) {
      showToast(t('rental.name_required'), 'error')
      return
    }
    if (!realEstateCategoryID) {
      showToast(t('rental.agency_category_missing'), 'error')
      return
    }
    setQuickSaving(true)
    try {
      const created = await api.institutions.create({ name, category_id: realEstateCategoryID })
      const list = await api.institutions.list()
      setInstitutions(list || [])
      if (created) {
        set('agency_institution_id', created.id)
        closeAgencyModal()
        showToast(t('rental.agency_created'), 'success')
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setQuickSaving(false)
    }
  }

  const createAgent = async () => {
    const name = quickAgent.full_name.trim()
    if (!name) {
      showToast(t('rental.name_required'), 'error')
      return
    }
    setQuickSaving(true)
    try {
      const created = await api.rental.people.create({
        full_name: name,
        email: quickAgent.email.trim(),
        phone: quickAgent.phone.trim(),
      })
      const list = await api.rental.people.list()
      setPeople(list || [])
      if (created) {
        set('agent_person_id', created.id)
        closeAgentModal()
        showToast(t('rental.agent_created'), 'success')
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setQuickSaving(false)
    }
  }

  const openPartyModal = () => {
    setPartyForm(emptyParty('tenant'))
    setPartyMode('new')
    setPartyModalOpen(true)
  }

  const savePartyModal = async () => {
    if (partyMode === 'existing' && !partyForm.person_id) {
      showToast(t('rental.person_required'), 'error')
      return
    }
    if (partyMode === 'new' && !partyForm.full_name.trim()) {
      showToast(t('rental.name_required'), 'error')
      return
    }
    setPartySaving(true)
    try {
      let personID = partyForm.person_id || 0
      let fullName = partyForm.full_name.trim()
      let email = partyForm.email.trim()
      let phone = partyForm.phone.trim()
      if (!personID) {
        const created = await api.rental.people.create({ full_name: fullName, email, phone })
        if (!created) return
        personID = created.id
        const list = await api.rental.people.list()
        setPeople(list || [])
      } else {
        const person = people.find((p) => p.id === personID)
        if (person) {
          fullName = person.full_name
          email = person.email
          phone = person.phone
        }
      }
      if (parties.some((p) => p.role === partyForm.role && p.person_id === personID)) {
        showToast(t('rental.party_exists'), 'error')
        return
      }
      setParties((prev) => [...prev, {
        person_id: personID,
        role: partyForm.role,
        notify_email: partyForm.notify_email,
        full_name: fullName,
        email,
        phone,
      }])
      setPartyModalOpen(false)
      setPartyForm(emptyParty('tenant'))
      showToast(t('rental.party_added'), 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setPartySaving(false)
    }
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.home_id) {
      setActiveTab('general')
      showToast(t('rental.lease.home_required'), 'error')
      return
    }
    if (form.rent_amount <= 0 && form.maintenance_amount <= 0) {
      setActiveTab('terms')
      showToast(t('rental.lease.amount_required'), 'error')
      return
    }
    setSaving(true)
    try {
      const payload: Partial<Lease> = {
        home_id: form.home_id,
        agency_institution_id: form.agency_institution_id || null,
        agent_person_id: form.agent_person_id || null,
        start_date: form.start_date,
        term_months: form.term_months,
        rent_amount: form.rent_amount,
        maintenance_amount: form.maintenance_amount,
        currency: form.currency,
        payment_day: form.payment_day,
        deposit_amount: form.deposit_amount,
        late_fee_type: form.late_fee_type,
        late_fee_value: form.late_fee_value,
        late_fee_cap: form.late_fee_cap,
        auto_renew: form.auto_renew,
        notice_days: form.notice_days,
        commission_percent: form.commission_percent,
        commission_type: form.commission_type,
        commission_amount: form.commission_amount,
        commission_base: form.commission_base,
        owner_bank_account: form.owner_bank_account,
        notes: form.notes,
        parties: parties.filter((p) => p.person_id || p.full_name.trim()).map((p) => ({
          person_id: p.person_id || 0,
          role: p.role,
          notify_email: p.notify_email,
          ...(p.person_id ? {} : { person: { full_name: p.full_name.trim(), email: p.email.trim(), phone: p.phone.trim() } }),
        } as LeaseParty)),
      }
      let saved: Lease | null
      if (isEdit) {
        saved = await api.rental.leases.update(Number(id), payload)
        if (saved) {
          const keep = new Set(parties.map((p) => p.id).filter(Boolean))
          await Promise.all(initialPartyIds.filter((pid) => !keep.has(pid)).map((pid) =>
            api.rental.leases.deleteParty(saved!.id, pid).catch(() => undefined)))
        }
      } else {
        saved = await api.rental.leases.create(payload)
      }
      if (saved) {
        showToast(isEdit ? t('rental.lease.updated') : t('rental.lease.created'), 'success')
        navigate(`/leases/${saved.id}`)
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  const homeOptions = homes.map((h) => ({ value: h.id, label: h.name }))
  const lockedHomeID = isEdit ? form.home_id : queryHomeID
  const selectedHome = homes.find((h) => h.id === lockedHomeID)
  const lockedTerms = isEdit && leaseStatus === 'active'
  const realEstateCategoryID = categories.find((c) => c.key === 'real_estate')?.id
  const agencies = institutions.filter((i) => realEstateCategoryID && i.category_id === realEstateCategoryID)
  const agencyOptions = [
    { value: 0, label: t('rental.none') },
    ...agencies.map((i) => ({ value: i.id, label: i.name })),
    addOption(t('app.add')),
  ]
  const agentOptions = [
    { value: 0, label: t('rental.none') },
    ...people.map((p) => ({ value: p.id, label: p.full_name })),
    addOption(t('app.add')),
  ]
  const currencyOptions = (currencies.length > 0 ? currencies.map((c) => c.code) : ['USD', 'NIO', 'EUR'])
    .map((code) => ({ value: code, label: code }))
  const roleOptions = [
    { value: 'tenant', label: t('rental.role.tenant') },
    { value: 'guarantor', label: t('rental.role.guarantor') },
    { value: 'owner', label: t('rental.role.owner') },
    { value: 'agency_contact', label: t('rental.role.agency_contact') },
  ]

  return (
    <>
    <form onSubmit={submit} className="max-w-3xl mx-auto space-y-4" noValidate>
      <h2 className="text-lg sm:text-xl font-bold">{isEdit ? t('rental.lease.edit') : t('rental.lease.create')}</h2>

      <div className="flex gap-1 p-1 bg-bg rounded-ios-sm" role="tablist">
        {(['general', 'terms', 'parties'] as LeaseTab[]).map((tab) => (
          <button
            key={tab}
            type="button"
            role="tab"
            aria-selected={activeTab === tab}
            onClick={() => setActiveTab(tab)}
            className={`flex-1 px-3 py-2 rounded-ios-sm text-sm font-medium min-h-[40px] transition-colors ${
              activeTab === tab ? 'bg-card text-primary shadow-ios' : 'text-text-secondary hover:text-text'
            }`}
          >
            {t(`rental.lease.tab_${tab}`)}
          </button>
        ))}
      </div>

      {activeTab === 'general' && (
      <SectionCard title={t('rental.lease.property')}>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label={t('rental.home')}>
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
          <Field label={t('rental.agency')}>
            <Select
              options={agencyOptions}
              value={form.agency_institution_id}
              onChange={(v) => {
                const value = Number(v)
                if (value === -1) {
                  setAgencyModalOpen(true)
                  return
                }
                if (value === 0) {
                  setForm((prev) => ({
                    ...prev,
                    agency_institution_id: 0,
                    agent_person_id: 0,
                    commission_type: 'percent',
                    commission_percent: 0,
                    commission_amount: 0,
                    commission_base: 'collected',
                  }))
                  return
                }
                set('agency_institution_id', value)
              }}
              searchable
            />
          </Field>
          {form.agency_institution_id === 0 && (
            <p className="text-xs text-text-secondary sm:col-span-2">{t('rental.lease.no_agency_hint')}</p>
          )}
          {form.agency_institution_id > 0 && (
            <>
          <Field label={t('rental.agent')}>
            <Select
              options={agentOptions}
              value={form.agent_person_id}
              onChange={(v) => {
                const value = Number(v)
                if (value === -1) {
                  setAgentModalOpen(true)
                  return
                }
                set('agent_person_id', value)
              }}
              searchable
            />
          </Field>
          <Field label={t('rental.commission_type')}>
            <Select
              options={[
                { value: 'percent', label: t('rental.commission_type_percent') },
                { value: 'fixed', label: t('rental.commission_type_fixed') },
              ]}
              value={form.commission_type}
              onChange={(v) => set('commission_type', v as 'percent' | 'fixed')}
            />
          </Field>
          {form.commission_type === 'percent' ? (
            <>
              <Field label={t('rental.commission_percent')}>
                <NumberInput  min={0} max={100} step="0.01" value={form.commission_percent} onChange={(e) => set('commission_percent', Number(e.target.value))} />
              </Field>
              <Field label={t('rental.commission_base')}>
                <Select
                  options={[
                    { value: 'collected', label: t('rental.commission.collected') },
                    { value: 'invoiced', label: t('rental.commission.invoiced') },
                  ]}
                  value={form.commission_base}
                  onChange={(v) => set('commission_base', v as 'collected' | 'invoiced')}
                />
              </Field>
            </>
          ) : (
            <Field label={t('rental.commission_amount')} hint={t('rental.commission_amount_hint')}>
              <NumberInput  min={0} step="0.01" value={form.commission_amount} onChange={(e) => set('commission_amount', Number(e.target.value))} />
            </Field>
          )}
            </>
          )}
        </div>
      </SectionCard>
      )}

      {activeTab === 'terms' && (
      <SectionCard title={t('rental.lease.terms')}>
        {lockedTerms && (
          <p className="text-xs text-text-secondary mb-3 rounded-ios-sm bg-bg/60 px-3 py-2">
            {t('rental.lease.active_edit_hint')}
          </p>
        )}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label={t('rental.start_date')}>
            <TextInput type="date" value={form.start_date} onChange={(e) => set('start_date', e.target.value)} disabled={lockedTerms} required />
          </Field>
          <Field label={t('rental.term_months')}>
            <Select
              options={[
                { value: 6, label: `6 ${t('rental.months')}` },
                { value: 12, label: `12 ${t('rental.months')}` },
                { value: 18, label: `18 ${t('rental.months')}` },
                { value: 24, label: `24 ${t('rental.months')}` },
                { value: 36, label: `36 ${t('rental.months')}` },
                { value: -1, label: t('rental.custom') },
              ]}
              value={customTerm ? -1 : form.term_months}
              disabled={lockedTerms}
              onChange={(v) => {
                if (Number(v) === -1) setCustomTerm(true)
                else { setCustomTerm(false); set('term_months', Number(v)) }
              }}
            />
            {customTerm && (
              <NumberInput
                className="mt-2"
                
                min={1}
                max={120}
                value={form.term_months}
                onChange={(e) => set('term_months', Number(e.target.value))}
                disabled={lockedTerms}
                aria-label={t('rental.term_months')}
              />
            )}
          </Field>
          <Field label={t('rental.rent')}>
            <NumberInput  min={0} step="0.01" value={form.rent_amount} onChange={(e) => set('rent_amount', Number(e.target.value))} disabled={lockedTerms} />
          </Field>
          <Field label={t('rental.maintenance')} hint={t('rental.maintenance_hint')}>
            <NumberInput  min={0} step="0.01" value={form.maintenance_amount} onChange={(e) => set('maintenance_amount', Number(e.target.value))} disabled={lockedTerms} />
          </Field>
          <Field label={t('rental.currency')}>
            <Select options={currencyOptions} value={form.currency} onChange={(v) => set('currency', String(v))} disabled={lockedTerms} searchable />
          </Field>
          <Field label={t('rental.payment_day')} hint={t('rental.payment_day_hint')}>
            <NumberInput  min={1} max={28} value={form.payment_day} onChange={(e) => set('payment_day', Number(e.target.value))} disabled={lockedTerms} />
          </Field>
          <Field label={t('rental.deposit')}>
            <NumberInput  min={0} step="0.01" value={form.deposit_amount} onChange={(e) => set('deposit_amount', Number(e.target.value))} disabled={lockedTerms} />
          </Field>
          <Field label={t('rental.notice_days')}>
            <NumberInput  min={0} value={form.notice_days} onChange={(e) => set('notice_days', Number(e.target.value))} />
          </Field>
          <Field label={t('rental.late_fee')}>
            <Select
              options={[
                { value: 'none', label: t('rental.late_fee_none') },
                { value: 'fixed', label: t('rental.late_fee_fixed') },
                { value: 'percent', label: t('rental.late_fee_percent') },
              ]}
              value={form.late_fee_type}
              onChange={(v) => set('late_fee_type', v as 'none' | 'fixed' | 'percent')}
            />
          </Field>
          {form.late_fee_type !== 'none' && (
            <>
              <Field label={t('rental.late_fee_value')}>
                <NumberInput  min={0} step="0.01" value={form.late_fee_value} onChange={(e) => set('late_fee_value', Number(e.target.value))} />
              </Field>
              <Field label={t('rental.late_fee_cap')}>
                <NumberInput  min={0} step="0.01" value={form.late_fee_cap} onChange={(e) => set('late_fee_cap', Number(e.target.value))} />
              </Field>
            </>
          )}
        </div>
        <div className="flex items-center justify-between gap-4 mt-4 pt-4 border-t border-border">
          <div>
            <p className="text-sm font-medium">{t('rental.auto_renew')}</p>
            <p className="text-xs text-text-secondary">{t('rental.auto_renew_hint')}</p>
          </div>
          <Toggle checked={form.auto_renew} onChange={(v) => set('auto_renew', v)} ariaLabel={t('rental.auto_renew')} />
        </div>
        <div className="mt-4 grid gap-4">
          <Field label={t('rental.owner_bank_account')} hint={t('rental.owner_bank_account_hint')}>
            <TextArea rows={2} value={form.owner_bank_account} onChange={(e) => set('owner_bank_account', e.target.value)} />
          </Field>
          <Field label={t('rental.notes')}>
            <TextArea rows={3} value={form.notes} onChange={(e) => set('notes', e.target.value)} />
          </Field>
        </div>
      </SectionCard>
      )}

      {activeTab === 'parties' && (
      <SectionCard
        title={t('rental.parties')}
        action={
          <button type="button" onClick={openPartyModal} className="text-sm text-primary font-medium">
            + {t('rental.add_party')}
          </button>
        }
      >
        {parties.length === 0 ? (
          <EmptyBlock
            title={t('rental.parties_empty')}
            subtitle={t('rental.parties_empty_hint')}
            action={<PrimaryButton type="button" onClick={openPartyModal}>{t('rental.add_party')}</PrimaryButton>}
          />
        ) : (
          <ul className="divide-y divide-border">
            {parties.map((party, index) => (
              <li key={`${party.person_id || party.full_name}-${party.role}-${index}`} className="py-2 flex items-center justify-between gap-3 text-sm">
                <div className="min-w-0">
                  <p className="font-medium truncate">{party.full_name || `#${index + 1}`}</p>
                  <p className="text-xs text-text-secondary truncate">{party.email || '—'}</p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Badge tone="neutral">{t(`rental.role.${party.role}`)}</Badge>
                  {party.notify_email && <Badge tone="info">{t('rental.notify_short')}</Badge>}
                  <button type="button" onClick={() => setRemovePartyIndex(index)} className="text-sm text-danger">
                    {t('app.remove')}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </SectionCard>
      )}

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

    <Modal
      open={agencyModalOpen}
      title={t('rental.agency_add')}
      onClose={closeAgencyModal}
      footer={(
        <>
          <SecondaryButton type="button" onClick={closeAgencyModal}>{t('app.cancel')}</SecondaryButton>
          <PrimaryButton type="button" disabled={quickSaving} onClick={createAgency}>
            {quickSaving ? t('app.saving') : t('app.create')}
          </PrimaryButton>
        </>
      )}
    >
      <Field label={t('rental.person_name')}>
        <TextInput
          autoFocus
          value={quickAgencyName}
          onChange={(e) => setQuickAgencyName(e.target.value)}
          placeholder={t('rental.agency_name_placeholder')}
        />
      </Field>
    </Modal>

    <Modal
      open={agentModalOpen}
      title={t('rental.agent_add')}
      onClose={closeAgentModal}
      footer={(
        <>
          <SecondaryButton type="button" onClick={closeAgentModal}>{t('app.cancel')}</SecondaryButton>
          <PrimaryButton type="button" disabled={quickSaving} onClick={createAgent}>
            {quickSaving ? t('app.saving') : t('app.create')}
          </PrimaryButton>
        </>
      )}
    >
      <Field label={t('rental.person_name')}>
        <TextInput
          autoFocus
          value={quickAgent.full_name}
          onChange={(e) => setQuickAgent((prev) => ({ ...prev, full_name: e.target.value }))}
        />
      </Field>
      <Field label={t('rental.email')}>
        <TextInput type="email" value={quickAgent.email} onChange={(e) => setQuickAgent((prev) => ({ ...prev, email: e.target.value }))} />
      </Field>
      <Field label={t('rental.phone')}>
        <TextInput value={quickAgent.phone} onChange={(e) => setQuickAgent((prev) => ({ ...prev, phone: e.target.value }))} />
      </Field>
    </Modal>

    <Modal
      open={partyModalOpen}
      title={t('rental.add_party')}
      onClose={() => setPartyModalOpen(false)}
      footer={(
        <>
          <SecondaryButton type="button" onClick={() => setPartyModalOpen(false)}>{t('app.cancel')}</SecondaryButton>
          <PrimaryButton type="button" disabled={partySaving} onClick={savePartyModal}>
            {partySaving ? t('app.saving') : t('app.add')}
          </PrimaryButton>
        </>
      )}
    >
      <Field label={t('rental.role_label')}>
        <Select
          options={roleOptions}
          value={partyForm.role}
          onChange={(v) => setPartyForm((prev) => ({ ...prev, role: v as PartyRole }))}
        />
      </Field>
      <div className="flex gap-1 p-1 bg-bg rounded-ios-sm" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={partyMode === 'new'}
          onClick={() => {
            setPartyMode('new')
            setPartyForm((prev) => ({ ...prev, person_id: undefined, full_name: '', email: '', phone: '' }))
          }}
          className={`flex-1 px-3 py-2 rounded-ios-sm text-sm font-medium min-h-[40px] transition-colors ${
            partyMode === 'new' ? 'bg-card text-primary shadow-ios' : 'text-text-secondary hover:text-text'
          }`}
        >
          {t('rental.new_person')}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={partyMode === 'existing'}
          onClick={() => {
            setPartyMode('existing')
            setPartyForm((prev) => ({ ...prev, person_id: undefined, full_name: '', email: '', phone: '' }))
          }}
          className={`flex-1 px-3 py-2 rounded-ios-sm text-sm font-medium min-h-[40px] transition-colors ${
            partyMode === 'existing' ? 'bg-card text-primary shadow-ios' : 'text-text-secondary hover:text-text'
          }`}
        >
          {t('rental.existing_person')}
        </button>
      </div>
      {partyMode === 'existing' ? (
        <Field label={t('rental.person')}>
          <Select
            options={people.map((p) => ({ value: p.id, label: p.full_name }))}
            value={partyForm.person_id || 0}
            onChange={(v) => {
              const pid = Number(v)
              const person = people.find((p) => p.id === pid)
              setPartyForm((prev) => ({
                ...prev,
                person_id: pid || undefined,
                full_name: person?.full_name || '',
                email: person?.email || '',
                phone: person?.phone || '',
              }))
            }}
            searchable
            placeholder={t('rental.select_person')}
          />
        </Field>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <Field label={t('rental.person_name')}>
            <TextInput value={partyForm.full_name} onChange={(e) => setPartyForm((prev) => ({ ...prev, full_name: e.target.value }))} />
          </Field>
          <Field label={t('rental.email')}>
            <TextInput type="email" value={partyForm.email} onChange={(e) => setPartyForm((prev) => ({ ...prev, email: e.target.value }))} />
          </Field>
          <Field label={t('rental.phone')}>
            <TextInput value={partyForm.phone} onChange={(e) => setPartyForm((prev) => ({ ...prev, phone: e.target.value }))} />
          </Field>
        </div>
      )}
      <label className="flex items-center justify-between gap-3 text-sm">
        <span>{t('rental.notify_email')}</span>
        <Toggle
          checked={partyForm.notify_email}
          onChange={(v) => setPartyForm((prev) => ({ ...prev, notify_email: v }))}
          ariaLabel={t('rental.notify_email')}
        />
      </label>
    </Modal>

    {removePartyIndex !== null && (
      <DeleteModal
        title={t('app.confirm')}
        subtitle={`${t('rental.person')}: ${parties[removePartyIndex]?.full_name || `#${removePartyIndex + 1}`}`}
        onConfirm={() => {
          setParties((prev) => prev.filter((_, i) => i !== removePartyIndex))
          setRemovePartyIndex(null)
        }}
        onCancel={() => setRemovePartyIndex(null)}
      />
    )}
    </>
  )
}
