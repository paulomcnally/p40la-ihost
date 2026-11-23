import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import LoadingSpinner from '../components/LoadingSpinner'
import Select from '../components/Select'
import NumberInput from '../components/NumberInput'
import Toggle from '../components/Toggle'
import { Icon } from '../components/Icons'
import { Badge, Field, Modal, Money, PrimaryButton, SecondaryButton, SectionCard, TextArea, TextInput } from '../components/rental/RentalUI'
import type { Invoice, Payment } from '../types/rental'

export default function InvoiceShowPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const invoiceId = Number(id)
  const { t } = useI18nStore()
  const { showToast } = useToast()

  const [loading, setLoading] = useState(true)
  const [invoice, setInvoice] = useState<Invoice | null>(null)
  const [payments, setPayments] = useState<Payment[]>([])
  const [busy, setBusy] = useState(false)
  const [payOpen, setPayOpen] = useState(false)
  const [voidOpen, setVoidOpen] = useState(false)
  const [voidReason, setVoidReason] = useState('')
  const [payForm, setPayForm] = useState({
    amount: 0, paid_at: new Date().toISOString().slice(0, 10), method: 'transfer',
    reference: '', notes: '', with_receipt: true,
  })

  usePageTitle(invoice ? invoice.number_label : t('rental.invoice.title'))

  const load = useCallback(async () => {
    if (!Number.isFinite(invoiceId) || invoiceId <= 0) { setLoading(false); return }
    setLoading(true)
    try {
      const data = await api.rental.invoices.get(invoiceId)
      setInvoice(data)
      if (data) {
        setPayForm((p) => ({ ...p, amount: data.total }))
        const pays = await api.rental.payments.list({ invoiceId }).catch(() => [])
        setPayments(pays || [])
      }
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setLoading(false)
    }
  }, [invoiceId])

  useEffect(() => { load() }, [load])

  const send = async () => {
    setBusy(true)
    try {
      const result = await api.rental.invoices.send(invoiceId)
      showToast(`${t('rental.invoice.sent')} (${result?.recipients ?? 0})`, 'success')
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const voidInvoice = async () => {
    if (!voidReason.trim()) {
      showToast(t('rental.reason_required'), 'error')
      return
    }
    setBusy(true)
    try {
      await api.rental.invoices.void(invoiceId, voidReason.trim())
      showToast(t('rental.invoice.voided'), 'success')
      setVoidOpen(false)
      setVoidReason('')
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  const submitPayment = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!invoice) return
    setBusy(true)
    try {
      await api.rental.payments.create({
        lease_id: invoice.lease_id,
        invoice_id: invoice.id,
        amount: payForm.amount,
        currency: invoice.currency,
        paid_at: payForm.paid_at,
        method: payForm.method,
        reference: payForm.reference,
        notes: payForm.notes,
        with_receipt: payForm.with_receipt,
      })
      showToast(t('rental.payment.created'), 'success')
      setPayOpen(false)
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <LoadingSpinner />
  if (!invoice) {
    return (
      <div className="bg-card rounded-ios shadow-ios p-8 text-center max-w-md mx-auto mt-8">
        <h3 className="text-lg font-semibold mb-3">{t('rental.invoice.not_found')}</h3>
        <PrimaryButton onClick={() => navigate('/home')}>{t('rental.back_to_homes')}</PrimaryButton>
      </div>
    )
  }

  const paid = payments.reduce((sum, p) => sum + p.amount, 0)
  const balance = invoice.total - paid
  const isVoid = invoice.status === 'void'

  return (
    <div className="space-y-4 sm:space-y-5">
      <div className="bg-card rounded-ios shadow-ios p-4 sm:p-6">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-xl font-bold">{invoice.number_label}</h2>
          <Badge tone={isVoid ? 'danger' : 'success'}>{t(`rental.invoice_status.${invoice.status}`)}</Badge>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4 text-sm">
          <div><p className="text-text-secondary text-xs">{t('rental.invoice.issue_date')}</p><p className="font-medium">{invoice.issue_date}</p></div>
          <div><p className="text-text-secondary text-xs">{t('rental.due')}</p><p className="font-medium">{invoice.due_date}</p></div>
          <div><p className="text-text-secondary text-xs">{t('rental.invoice.total')}</p><p className="font-medium"><Money amount={invoice.total} currency={invoice.currency} /></p></div>
          <div><p className="text-text-secondary text-xs">{t('rental.balance')}</p><p className="font-medium"><Money amount={balance} currency={invoice.currency} /></p></div>
        </div>
        <div className="flex flex-wrap gap-2 mt-5">
          <SecondaryButton onClick={() => window.open(api.rental.invoices.pdfUrl(invoice.id), '_blank')}>
            <Icon name="pdf" className="w-4 h-4" />
            {t('rental.invoice.pdf')}
          </SecondaryButton>
          {!isVoid && (
            <>
              <SecondaryButton disabled={busy} onClick={send}>
                <Icon name="send" className="w-4 h-4" />
                {t('rental.invoice.send')}
              </SecondaryButton>
              <PrimaryButton onClick={() => setPayOpen(true)}>{t('rental.payment.create')}</PrimaryButton>
              <button type="button" className="px-4 py-2 text-danger text-sm font-medium" onClick={() => setVoidOpen(true)}>
                {t('rental.invoice.void')}
              </button>
            </>
          )}
          <button type="button" className="px-4 py-2 text-sm text-primary" onClick={() => navigate(`/leases/${invoice.lease_id}`)}>
            {t('rental.lease.open')}
          </button>
        </div>
        {invoice.sent_at && <p className="text-xs text-text-secondary mt-2">{t('rental.invoice.sent_at')}: {new Date(invoice.sent_at).toLocaleString()}</p>}
        {isVoid && invoice.void_reason && <p className="text-sm text-danger mt-2">{t('rental.reason')}: {invoice.void_reason}</p>}
      </div>

      <SectionCard title={t('rental.invoice.lines')}>
        <table className="w-full text-sm">
          <thead className="text-text-secondary text-xs">
            <tr>
              <th className="text-left py-2">{t('rental.concept')}</th>
              <th className="text-right py-2">{t('rental.amount')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {(invoice.lines || []).map((line) => (
              <tr key={line.id}>
                <td className="py-2">{line.description}</td>
                <td className="py-2 text-right"><Money amount={line.amount} currency={invoice.currency} /></td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <td className="py-2 font-semibold">{t('rental.invoice.total')}</td>
              <td className="py-2 text-right font-semibold"><Money amount={invoice.total} currency={invoice.currency} /></td>
            </tr>
          </tfoot>
        </table>
      </SectionCard>

      <SectionCard title={t('rental.payments')}>
        {payments.length === 0 ? (
          <p className="text-sm text-text-secondary">{t('rental.payments_empty')}</p>
        ) : (
          <ul className="divide-y divide-border">
            {payments.map((p) => (
              <li key={p.id} className="py-2 flex items-center justify-between gap-3 text-sm">
                <div>
                  <p className="font-medium">{p.paid_at} · {t(`rental.method.${p.method}`)}</p>
                  {p.reference && <p className="text-xs text-text-secondary">{p.reference}</p>}
                </div>
                <div className="flex items-center gap-3">
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

      <Modal
        open={payOpen}
        title={t('rental.payment.create')}
        onClose={() => setPayOpen(false)}
        footer={(
          <>
            <SecondaryButton onClick={() => setPayOpen(false)}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy} onClick={submitPayment}>{t('app.save')}</PrimaryButton>
          </>
        )}
      >
        <form onSubmit={submitPayment} className="space-y-3">
          <Field label={t('rental.amount')}>
            <NumberInput  min={0.01} step="0.01" value={payForm.amount} onChange={(e) => setPayForm((p) => ({ ...p, amount: Number(e.target.value) }))} required />
          </Field>
          <Field label={t('rental.payment_date')}>
            <TextInput type="date" value={payForm.paid_at} onChange={(e) => setPayForm((p) => ({ ...p, paid_at: e.target.value }))} />
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
              value={payForm.method}
              onChange={(v) => setPayForm((p) => ({ ...p, method: String(v) }))}
            />
          </Field>
          <Field label={t('rental.reference')}>
            <TextInput value={payForm.reference} onChange={(e) => setPayForm((p) => ({ ...p, reference: e.target.value }))} />
          </Field>
          <Field label={t('rental.notes')}>
            <TextArea rows={2} value={payForm.notes} onChange={(e) => setPayForm((p) => ({ ...p, notes: e.target.value }))} />
          </Field>
          <label className="flex items-center justify-between gap-3 text-sm">
            <span>{t('rental.payment.with_receipt')}</span>
            <Toggle checked={payForm.with_receipt} onChange={(v) => setPayForm((p) => ({ ...p, with_receipt: v }))} ariaLabel={t('rental.payment.with_receipt')} />
          </label>
        </form>
      </Modal>

      <Modal
        open={voidOpen}
        title={t('rental.invoice.void')}
        onClose={() => { setVoidOpen(false); setVoidReason('') }}
        footer={(
          <>
            <SecondaryButton onClick={() => { setVoidOpen(false); setVoidReason('') }}>{t('app.cancel')}</SecondaryButton>
            <PrimaryButton disabled={busy} onClick={voidInvoice}>{t('rental.invoice.void')}</PrimaryButton>
          </>
        )}
      >
        <p className="text-sm text-text-secondary">{t('rental.invoice.void_hint')}</p>
        <Field label={t('rental.reason')}>
          <TextArea rows={3} value={voidReason} onChange={(e) => setVoidReason(e.target.value)} />
        </Field>
      </Modal>
    </div>
  )
}
