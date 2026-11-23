import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import LoadingSpinner from '../components/LoadingSpinner'
import CardMenu from '../components/CardMenu'
import Select from '../components/Select'
import Toggle from '../components/Toggle'
import { Icon } from '../components/Icons'
import {
  Badge, EmptyBlock, Field, Modal, Money, PrimaryButton, SecondaryButton, SectionCard, TextArea, TextInput,
} from '../components/rental/RentalUI'
import type { Charge, Invoice, Lease, LeaseDocument, Payment, Reminder, RentalBalance } from '../types/rental'

const statusTone = (status: string) => {
  if (status === 'active') return 'info' as const
  if (status === 'draft') return 'warning' as const
  if (status === 'terminated' || status === 'cancelled') return 'danger' as const
  return 'neutral' as const
}

// Selección múltiple de filas con semántica de checkbox y sin control nativo
// (UI-TOGGLE: la excepción de selección múltiple se implementa accesible).
function RowCheck({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
      className={`w-5 h-5 rounded border flex items-center justify-center transition-colors ${
        checked ? 'bg-primary border-primary text-white' : 'border-border text-transparent hover:border-primary'
      }`}
    >
      <Icon name="check" className="w-3.5 h-3.5" />
    </button>
  )
}

export default function LeaseShowPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const leaseId = Number(id)
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [loading, setLoading] = useState(true)
  const [lease, setLease] = useState<Lease | null>(null)
  const [charges, setCharges] = useState<Charge[]>([])
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [payments, setPayments] = useState<Payment[]>([])
  const [documents, setDocuments] = useState<LeaseDocument[]>([])
  const [reminders, setReminders] = useState<Reminder[]>([])
  const [balance, setBalance] = useState<RentalBalance | null>(null)
  const [selectedCharges, setSelectedCharges] = useState<number[]>([])
  const [busy, setBusy] = useState(false)

  const [confirmAction, setConfirmAction] = useState<'activate' | 'terminate' | 'delete' | null>(null)
  const [reason, setReason] = useState('')
  const [paymentOpen, setPaymentOpen] = useState(false)
  const [reminderOpen, setReminderOpen] = useState(false)
  const [documentOpen, setDocumentOpen] = useState(false)

  const [paymentForm, setPaymentForm] = useState({
    amount: 0, paid_at: new Date().toISOString().slice(0, 10), method: 'transfer',
    reference: '', notes: '', with_receipt: true, invoice_id: 0,
  })
  const [reminderForm, setReminderForm] = useState({
    title: '', description: '', kind: 'maintenance', schedule_type: 'monthly',
    interval_n: 1, anchor_date: new Date().toISOString().slice(0, 10),
    next_due_at: new Date().toISOString().slice(0, 16), lead_days: 3,
    notify_roles: 'tenant', extra_emails: '', enabled: true,
  })
  const [docType, setDocType] = useState('contract')
  const [docFile, setDocFile] = useState<File | null>(null)

  usePageTitle(lease ? `${t('rental.lease.title')} #${lease.id}` : t('rental.lease.title'))

  const load = useCallback(async () => {
    if (!Number.isFinite(leaseId) || leaseId <= 0) { setLoading(false); return }
    setLoading(true)
    try {
      const leaseData = await api.rental.leases.get(leaseId)
      setLease(leaseData)
      if (leaseData) {
        const [ch, inv, pay, docs, rem, bal] = await Promise.all([
          api.rental.leases.charges(leaseId).catch(() => []),
          api.rental.invoices.list({ leaseId }).catch(() => []),
          api.rental.payments.list({ leaseId }).catch(() => []),
          api.rental.leases.documents(leaseId).catch(() => []),
          api.rental.reminders.list({ leaseId }).catch(() => []),
          api.rental.leases.balance(leaseId).catch(() => null),
        ])
        setCharges(ch || [])
        setInvoices(inv || [])
        setPayments(pay || [])
        setDocuments(docs || [])
        setReminders(rem || [])
        setBalance(bal)
        setPaymentForm((prev) => ({ ...prev, amount: bal?.balance && bal.balance > 0 ? bal.balance : 0 }))
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [leaseId])

  useEffect(() => { load() }, [load])

  const run = async (fn: () => Promise<unknown>, success: string) => {
    setBusy(true)
    try {
      await fn()
      showToast(success, 'success')
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const handleConfirm = async () => {
    if (!lease || !confirmAction) return
    if (confirmAction === 'activate') {
      await run(() => api.rental.leases.activate(lease.id), t('rental.lease.activated'))
    } else if (confirmAction === 'terminate') {
      await run(() => api.rental.leases.terminate(lease.id, { reason, end_date: new Date().toISOString().slice(0, 10) }), t('rental.lease.terminated'))
    } else if (confirmAction === 'delete') {
      setBusy(true)
      try {
        await api.rental.leases.delete(lease.id)
        showToast(t('rental.lease.deleted'), 'success')
        navigate('/home')
      } catch (err: unknown) {
        showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      } finally {
        setBusy(false)
      }
    }
    setConfirmAction(null)
    setReason('')
  }

  const issueInvoice = async () => {
    if (!lease || selectedCharges.length === 0) return
    setBusy(true)
    try {
      const inv = await api.rental.invoices.issue({
        lease_id: lease.id,
        charge_ids: selectedCharges,
        due_date: new Date().toISOString().slice(0, 10),
      })
      if (inv) {
        showToast(t('rental.invoice.issued'), 'success')
        setSelectedCharges([])
        navigate(`/invoices/${inv.id}`)
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const submitPayment = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!lease) return
    setBusy(true)
    try {
      await api.rental.payments.create({
        lease_id: lease.id,
        invoice_id: paymentForm.invoice_id || null,
        amount: paymentForm.amount,
        currency: lease.currency,
        paid_at: paymentForm.paid_at,
        method: paymentForm.method,
        reference: paymentForm.reference,
        notes: paymentForm.notes,
        with_receipt: paymentForm.with_receipt,
      })
      showToast(t('rental.payment.created'), 'success')
      setPaymentOpen(false)
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const submitReminder = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!lease) return
    setBusy(true)
    try {
      await api.rental.reminders.create({
        lease_id: lease.id,
        title: reminderForm.title,
        description: reminderForm.description,
        kind: reminderForm.kind as Reminder['kind'],
        schedule_type: reminderForm.schedule_type as Reminder['schedule_type'],
        interval_n: reminderForm.interval_n,
        anchor_date: reminderForm.anchor_date,
        next_due_at: new Date(reminderForm.next_due_at).toISOString(),
        lead_days: reminderForm.lead_days,
        notify_roles: reminderForm.notify_roles,
        extra_emails: reminderForm.extra_emails,
        enabled: reminderForm.enabled,
      })
      showToast(t('rental.reminder.created'), 'success')
      setReminderOpen(false)
      setReminderForm((prev) => ({ ...prev, title: '', description: '' }))
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const submitDocument = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!lease || !docFile) return
    setBusy(true)
    try {
      await api.rental.leases.uploadDocument(lease.id, docFile, docType)
      showToast(t('rental.document.uploaded'), 'success')
      setDocumentOpen(false)
      setDocFile(null)
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <LoadingSpinner />
  if (!lease) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <h3 className="text-lg font-semibold mb-3">{t('rental.lease.not_found')}</h3>
        <PrimaryButton onClick={() => navigate('/home')}>{t('rental.back_to_homes')}</PrimaryButton>
      </div>
    )
  }

  const menuOptions = []
  if (lease.status === 'draft') {
    menuOptions.push({ label: t('app.edit'), onClick: () => navigate(`/leases/${lease.id}/edit`) })
    menuOptions.push({ label: t('rental.lease.activate'), onClick: () => setConfirmAction('activate') })
    menuOptions.push({ label: t('app.delete'), danger: true, onClick: () => setConfirmAction('delete') })
  }
  if (lease.status === 'active') {
    menuOptions.push({ label: t('rental.lease.terminate'), danger: true, onClick: () => setConfirmAction('terminate') })
  }

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="relative bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <CardMenu options={menuOptions} />
        <div className="pr-10">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-xl font-bold">{t('rental.lease.title')} #{lease.id}</h2>
            <Badge tone={statusTone(lease.status)}>{t(`rental.lease_status.${lease.status}`)}</Badge>
            {lease.expiring && <Badge tone="warning">{t('rental.lease.expiring')}</Badge>}
          </div>
          <p className="text-sm text-text-secondary mt-1">{lease.home_name}{lease.home_address ? ` — ${lease.home_address}` : ''}</p>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4 text-sm">
            <div><p className="text-text-secondary text-xs">{t('rental.period')}</p><p className="font-medium">{lease.start_date} → {lease.end_date}</p></div>
            <div><p className="text-text-secondary text-xs">{t('rental.rent')}</p><p className="font-medium"><Money amount={lease.rent_amount} currency={lease.currency} /></p></div>
            <div><p className="text-text-secondary text-xs">{t('rental.maintenance')}</p><p className="font-medium"><Money amount={lease.maintenance_amount} currency={lease.currency} /></p></div>
            <div><p className="text-text-secondary text-xs">{t('rental.balance')}</p><p className="font-medium"><Money amount={balance?.balance ?? lease.balance ?? 0} currency={lease.currency} /></p></div>
          </div>
        </div>
      </div>

      {lease.status === 'draft' && (
        <div className="flex justify-end">
          <PrimaryButton disabled={busy} onClick={() => setConfirmAction('activate')}>
            <Icon name="check" className="w-4 h-4" />
            {t('rental.lease.activate')}
          </PrimaryButton>
        </div>
      )}

      <SectionCard title={t('rental.parties')}>
        {(!lease.parties || lease.parties.length === 0) ? (
          <EmptyBlock title={t('rental.parties_empty')} subtitle={t('rental.parties_empty_hint')} />
        ) : (
          <ul className="divide-y divide-border">
            {lease.parties.map((p) => (
              <li key={p.id} className="py-2 flex items-center justify-between gap-3 text-sm">
                <div className="min-w-0">
                  <p className="font-medium truncate">{p.person?.full_name}</p>
                  <p className="text-xs text-text-secondary truncate">{p.person?.email || '—'}</p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Badge tone="neutral">{t(`rental.role.${p.role}`)}</Badge>
                  {p.notify_email && <Badge tone="info">{t('rental.notify_short')}</Badge>}
                </div>
              </li>
            ))}
          </ul>
        )}
      </SectionCard>

      <SectionCard title={t('rental.intermediation')}>
        <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
          <div>
            <dt className="text-text-secondary">{t('rental.agency')}</dt>
            <dd className="font-medium mt-0.5">{lease.agency_name || '—'}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t('rental.agent')}</dt>
            <dd className="font-medium mt-0.5">{lease.agent_name || '—'}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t('rental.commission_label')}</dt>
            <dd className="font-medium mt-0.5">
              {lease.commission_type === 'fixed'
                ? <Money amount={lease.commission_amount} currency={lease.currency} />
                : `${lease.commission_percent}%`}
            </dd>
          </div>
          {lease.commission_type !== 'fixed' && (
            <div>
              <dt className="text-text-secondary">{t('rental.commission_base')}</dt>
              <dd className="font-medium mt-0.5">{t(`rental.commission.${lease.commission_base}`)}</dd>
            </div>
          )}
        </dl>
      </SectionCard>

      <SectionCard
        title={t('rental.charges_title')}
        action={lease.status === 'active' ? (
          <div className="flex gap-2">
            <SecondaryButton disabled={busy} onClick={() => run(() => api.rental.leases.generateCharges(lease.id), t('rental.charges.generated'))}>
              {t('rental.charges.generate')}
            </SecondaryButton>
            <PrimaryButton disabled={busy || selectedCharges.length === 0} onClick={issueInvoice}>
              {t('rental.invoice.issue')} ({selectedCharges.length})
            </PrimaryButton>
          </div>
        ) : undefined}
      >
        {charges.length === 0 ? (
          <EmptyBlock title={t('rental.charges_empty')} subtitle={t('rental.charges_empty_hint')} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-text-secondary text-xs">
                <tr>
                  <th className="w-8" />
                  <th className="text-left py-2">{t('rental.concept')}</th>
                  <th className="text-left py-2">{t('rental.due')}</th>
                  <th className="text-right py-2">{t('rental.amount')}</th>
                  <th className="text-right py-2">{t('rental.status')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {charges.map((c) => (
                  <tr key={c.id}>
                    <td className="py-2">
                      {c.status === 'pending' && (
                        <RowCheck
                          checked={selectedCharges.includes(c.id)}
                          onChange={(checked) => setSelectedCharges((prev) => checked ? [...prev, c.id] : prev.filter((x) => x !== c.id))}
                          label={c.description}
                        />
                      )}
                    </td>
                    <td className="py-2">{c.description}</td>
                    <td className="py-2">{c.due_date}</td>
                    <td className="py-2 text-right"><Money amount={c.amount} currency={c.currency} /></td>
                    <td className="py-2 text-right"><Badge tone={c.status === 'paid' ? 'success' : c.status === 'pending' ? 'warning' : 'neutral'}>{t(`rental.charge_status.${c.status}`)}</Badge></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </SectionCard>

      <SectionCard
        title={t('rental.invoices')}
        action={<PrimaryButton onClick={() => setPaymentOpen(true)}>{t('rental.payment.create')}</PrimaryButton>}
      >
        {invoices.length === 0 ? (
          <EmptyBlock title={t('rental.invoices_empty')} subtitle={t('rental.invoices_empty_hint')} />
        ) : (
          <ul className="divide-y divide-border">
            {invoices.map((inv) => (
              <li key={inv.id} className="py-2 flex items-center justify-between gap-3 text-sm">
                <button type="button" onClick={() => navigate(`/invoices/${inv.id}`)} className="text-left min-w-0">
                  <p className="font-medium">{inv.number_label}</p>
                  <p className="text-xs text-text-secondary">{inv.issue_date} · {t('rental.due')} {inv.due_date}</p>
                </button>
                <div className="flex items-center gap-2 shrink-0">
                  <Money amount={inv.total} currency={inv.currency} />
                  <Badge tone={inv.status === 'void' ? 'danger' : 'success'}>{t(`rental.invoice_status.${inv.status}`)}</Badge>
                </div>
              </li>
            ))}
          </ul>
        )}
      </SectionCard>

      <SectionCard title={t('rental.payments')}>
        {payments.length === 0 ? (
          <EmptyBlock title={t('rental.payments_empty')} />
        ) : (
          <ul className="divide-y divide-border">
            {payments.map((p) => (
              <li key={p.id} className="py-2 flex items-center justify-between gap-3 text-sm">
                <div className="min-w-0">
                  <p className="font-medium">{p.paid_at} · {t(`rental.method.${p.method}`)}</p>
                  {p.reference && <p className="text-xs text-text-secondary">{p.reference}</p>}
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <Money amount={p.amount} currency={p.currency} />
                  {p.receipt_id && (
                    <a className="text-primary text-xs underline" href={api.rental.payments.receiptUrl(p.id)} target="_blank" rel="noreferrer">
                      {t('rental.receipt.download')}
                    </a>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </SectionCard>

      <SectionCard
        title={t('rental.documents')}
        action={<SecondaryButton onClick={() => setDocumentOpen(true)}>+ {t('rental.document.add')}</SecondaryButton>}
      >
        {documents.length === 0 ? (
          <EmptyBlock title={t('rental.documents_empty')} subtitle={t('rental.documents_empty_hint')} />
        ) : (
          <ul className="divide-y divide-border">
            {documents.map((d) => (
              <li key={d.id} className="py-2 flex items-center justify-between gap-3 text-sm">
                <div className="min-w-0">
                  <p className="font-medium truncate">{d.original_name}</p>
                  <p className="text-xs text-text-secondary">{t(`rental.doc_type.${d.doc_type}`)}</p>
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <a className="text-primary text-xs underline" href={api.rental.leases.documentUrl(lease.id, d.id)} target="_blank" rel="noreferrer">{t('rental.view')}</a>
                  <button
                    type="button"
                    className="text-danger text-xs"
                    onClick={() => run(() => api.rental.leases.deleteDocument(lease.id, d.id), t('rental.document.deleted'))}
                  >
                    {t('app.delete')}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </SectionCard>

      <SectionCard
        title={t('rental.reminders')}
        action={<SecondaryButton onClick={() => setReminderOpen(true)}>+ {t('rental.reminder.add')}</SecondaryButton>}
      >
        {reminders.length === 0 ? (
          <EmptyBlock title={t('rental.reminders_empty')} subtitle={t('rental.reminders_empty_hint')} />
        ) : (
          <ul className="divide-y divide-border">
            {reminders.map((r) => (
              <li key={r.id} className="py-2 flex items-center justify-between gap-3 text-sm">
                <div className="min-w-0">
                  <p className="font-medium">{r.title}</p>
                  <p className="text-xs text-text-secondary">{new Date(r.next_due_at).toLocaleString()} · {t(`rental.reminder_kind.${r.kind}`)}</p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Badge tone={r.enabled ? 'success' : 'neutral'}>{r.enabled ? t('rental.enabled') : t('rental.disabled')}</Badge>
                  <button type="button" className="text-primary text-xs" onClick={() => run(() => api.rental.reminders.run(r.id), t('rental.reminder.sent'))}>{t('rental.send_now')}</button>
                  <button type="button" className="text-danger text-xs" onClick={() => run(() => api.rental.reminders.delete(r.id), t('rental.reminder.deleted'))}>{t('app.delete')}</button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </SectionCard>

      <Modal
        open={confirmAction !== null}
        title={confirmAction === 'delete' ? t('app.confirm') : confirmAction === 'activate' ? t('rental.lease.activate') : t('rental.lease.terminate')}
        onClose={() => { setConfirmAction(null); setReason('') }}
        footer={(
          <>
            <SecondaryButton onClick={() => { setConfirmAction(null); setReason('') }}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy} onClick={handleConfirm}>{t('app.confirm')}</PrimaryButton>
          </>
        )}
      >
        <p className="text-sm text-text-secondary">
          {confirmAction === 'activate' ? t('rental.lease.activate_hint') : confirmAction === 'delete' ? t('rental.lease.delete_hint') : t('rental.lease.terminate_hint')}
        </p>
        {confirmAction === 'terminate' && (
          <Field label={t('rental.reason')}>
            <TextArea rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
          </Field>
        )}
      </Modal>

      <Modal
        open={paymentOpen}
        title={t('rental.payment.create')}
        onClose={() => setPaymentOpen(false)}
        footer={(
          <>
            <SecondaryButton onClick={() => setPaymentOpen(false)}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy} onClick={submitPayment}>{t('app.save')}</PrimaryButton>
          </>
        )}
      >
        <form onSubmit={submitPayment} className="space-y-3">
          <Field label={t('rental.amount')}>
            <TextInput type="number" min={0.01} step="0.01" value={paymentForm.amount} onChange={(e) => setPaymentForm((p) => ({ ...p, amount: Number(e.target.value) }))} required />
          </Field>
          <Field label={t('rental.payment_date')}>
            <TextInput type="date" value={paymentForm.paid_at} onChange={(e) => setPaymentForm((p) => ({ ...p, paid_at: e.target.value }))} />
          </Field>
          <Field label={t('rental.method_label')}>
            <Select
              options={[
                { value: 'transfer', label: t('rental.method.transfer') },
                { value: 'cash', label: t('rental.method.cash') },
                { value: 'check', label: t('rental.method.check') },
                { value: 'card', label: t('rental.method.card') },
                { value: 'other', label: t('rental.method.other') },
              ]}
              value={paymentForm.method}
              onChange={(v) => setPaymentForm((p) => ({ ...p, method: String(v) }))}
            />
          </Field>
          <Field label={t('rental.payment_invoice')}>
            <Select
              options={[{ value: 0, label: t('rental.none') }, ...invoices.filter((i) => i.status === 'issued').map((i) => ({ value: i.id, label: `${i.number_label} · ${i.total} ${i.currency}` }))]}
              value={paymentForm.invoice_id}
              onChange={(v) => setPaymentForm((p) => ({ ...p, invoice_id: Number(v) }))}
            />
            {paymentForm.invoice_id === 0 && (
              <p className="text-xs text-text-secondary mt-1">{t('rental.payment.auto_apply_hint')}</p>
            )}
          </Field>
          <Field label={t('rental.reference')}>
            <TextInput value={paymentForm.reference} onChange={(e) => setPaymentForm((p) => ({ ...p, reference: e.target.value }))} />
          </Field>
          <Field label={t('rental.notes')}>
            <TextArea rows={2} value={paymentForm.notes} onChange={(e) => setPaymentForm((p) => ({ ...p, notes: e.target.value }))} />
          </Field>
          <label className="flex items-center justify-between gap-3 text-sm">
            <span>{t('rental.payment.with_receipt')}</span>
            <Toggle checked={paymentForm.with_receipt} onChange={(v) => setPaymentForm((p) => ({ ...p, with_receipt: v }))} ariaLabel={t('rental.payment.with_receipt')} />
          </label>
        </form>
      </Modal>

      <Modal
        open={reminderOpen}
        title={t('rental.reminder.add')}
        onClose={() => setReminderOpen(false)}
        footer={(
          <>
            <SecondaryButton onClick={() => setReminderOpen(false)}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy} onClick={submitReminder}>{t('app.save')}</PrimaryButton>
          </>
        )}
      >
        <form onSubmit={submitReminder} className="space-y-3">
          <Field label={t('rental.reminder.title_label')}>
            <TextInput value={reminderForm.title} onChange={(e) => setReminderForm((p) => ({ ...p, title: e.target.value }))} required />
          </Field>
          <Field label={t('rental.notes')}>
            <TextArea rows={2} value={reminderForm.description} onChange={(e) => setReminderForm((p) => ({ ...p, description: e.target.value }))} />
          </Field>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field label={t('rental.reminder.kind')}>
              <Select
                options={[
                  { value: 'gardening', label: t('rental.reminder_kind.gardening') },
                  { value: 'maintenance', label: t('rental.reminder_kind.maintenance') },
                  { value: 'inspection', label: t('rental.reminder_kind.inspection') },
                  { value: 'payment_reminder', label: t('rental.reminder_kind.payment_reminder') },
                  { value: 'custom', label: t('rental.reminder_kind.custom') },
                ]}
                value={reminderForm.kind}
                onChange={(v) => setReminderForm((p) => ({ ...p, kind: String(v) }))}
              />
            </Field>
            <Field label={t('rental.reminder.schedule')}>
              <Select
                options={[
                  { value: 'once', label: t('rental.schedule.once') },
                  { value: 'monthly', label: t('rental.schedule.monthly') },
                  { value: 'yearly', label: t('rental.schedule.yearly') },
                  { value: 'every_n_days', label: t('rental.schedule.every_n_days') },
                ]}
                value={reminderForm.schedule_type}
                onChange={(v) => setReminderForm((p) => ({ ...p, schedule_type: String(v) }))}
              />
            </Field>
            <Field label={t('rental.reminder.next_due')}>
              <TextInput type="datetime-local" value={reminderForm.next_due_at} onChange={(e) => setReminderForm((p) => ({ ...p, next_due_at: e.target.value }))} required />
            </Field>
            <Field label={t('rental.reminder.lead_days')}>
              <TextInput type="number" min={0} value={reminderForm.lead_days} onChange={(e) => setReminderForm((p) => ({ ...p, lead_days: Number(e.target.value) }))} />
            </Field>
          </div>
          <Field label={t('rental.reminder.roles')} hint={t('rental.reminder.roles_hint')}>
            <TextInput value={reminderForm.notify_roles} onChange={(e) => setReminderForm((p) => ({ ...p, notify_roles: e.target.value }))} />
          </Field>
          <Field label={t('rental.reminder.extra_emails')} hint={t('rental.reminder.extra_emails_hint')}>
            <TextInput value={reminderForm.extra_emails} onChange={(e) => setReminderForm((p) => ({ ...p, extra_emails: e.target.value }))} />
          </Field>
          <label className="flex items-center justify-between gap-3 text-sm">
            <span>{t('rental.enabled')}</span>
            <Toggle checked={reminderForm.enabled} onChange={(v) => setReminderForm((p) => ({ ...p, enabled: v }))} ariaLabel={t('rental.enabled')} />
          </label>
        </form>
      </Modal>

      <Modal
        open={documentOpen}
        title={t('rental.document.add')}
        onClose={() => setDocumentOpen(false)}
        footer={(
          <>
            <SecondaryButton onClick={() => setDocumentOpen(false)}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy || !docFile} onClick={submitDocument}>{t('rental.upload')}</PrimaryButton>
          </>
        )}
      >
        <form onSubmit={submitDocument} className="space-y-3">
          <Field label={t('rental.document.type')}>
            <Select
              options={[
                { value: 'contract', label: t('rental.doc_type.contract') },
                { value: 'addendum', label: t('rental.doc_type.addendum') },
                { value: 'inventory', label: t('rental.doc_type.inventory') },
                { value: 'id', label: t('rental.doc_type.id') },
                { value: 'income_proof', label: t('rental.doc_type.income_proof') },
                { value: 'insurance', label: t('rental.doc_type.insurance') },
                { value: 'agent_credential', label: t('rental.doc_type.agent_credential') },
                { value: 'other', label: t('rental.doc_type.other') },
              ]}
              value={docType}
              onChange={(v) => setDocType(String(v))}
            />
          </Field>
          <Field label={t('rental.file')} hint={t('rental.file_hint')}>
            <input
              type="file"
              accept="application/pdf,.pdf"
              onChange={(e) => setDocFile(e.target.files?.[0] || null)}
              className="block w-full text-sm text-text-secondary"
            />
          </Field>
        </form>
      </Modal>
    </div>
  )
}
