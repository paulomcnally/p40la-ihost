import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAppStore } from '../stores/appStore'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import LoadingSpinner from '../components/LoadingSpinner'
import Select from '../components/Select'
import { Icon } from '../components/Icons'
import { Field, PrimaryButton, SecondaryButton, SectionCard, TextArea, TextInput } from '../components/rental/RentalUI'
import type { RentalSettings } from '../types/rental'

export default function RentalSettingsPage() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { currencies, loadCurrencies } = useAppStore()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState<RentalSettings>({
    issuer_name: '', issuer_tax_id: '', issuer_address: '', issuer_email: '',
    invoice_series: 'ALQ', receipt_series: 'REC', default_currency: 'USD',
    default_lead_days: 3, default_owner_split_percent: 50, default_agency_split_percent: 50,
  })

  usePageTitle(t('rental.settings.title'))

  useEffect(() => {
    loadCurrencies()
    api.rental.settings.get()
      .then((data) => { if (data) setForm(data) })
      .catch((err: unknown) => showToast(err instanceof Error ? err.message : t('errors.generic'), 'error'))
      .finally(() => setLoading(false))
  }, [])

  const set = <K extends keyof RentalSettings>(key: K, value: RentalSettings[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }))

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      const saved = await api.rental.settings.update(form)
      if (saved) setForm(saved)
      showToast(t('rental.settings.saved'), 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  const currencyOptions = (currencies.length > 0 ? currencies.map((c) => c.code) : ['USD', 'NIO', 'EUR'])
    .map((code) => ({ value: code, label: code }))

  return (
    <form onSubmit={submit} className="max-w-2xl mx-auto space-y-4" noValidate>
      <h2 className="text-lg sm:text-xl font-bold">{t('rental.settings.title')}</h2>

      <SectionCard title={t('rental.settings.issuer')} subtitle={t('rental.settings.issuer_hint')}>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label={t('rental.settings.issuer_name')}>
            <TextInput value={form.issuer_name} onChange={(e) => set('issuer_name', e.target.value)} />
          </Field>
          <Field label={t('rental.settings.issuer_tax_id')}>
            <TextInput value={form.issuer_tax_id} onChange={(e) => set('issuer_tax_id', e.target.value)} />
          </Field>
          <Field label={t('rental.settings.issuer_email')}>
            <TextInput type="email" value={form.issuer_email} onChange={(e) => set('issuer_email', e.target.value)} />
          </Field>
          <Field label={t('rental.settings.default_currency')}>
            <Select options={currencyOptions} value={form.default_currency} onChange={(v) => set('default_currency', String(v))} searchable />
          </Field>
        </div>
        <div className="mt-4">
          <Field label={t('rental.settings.issuer_address')}>
            <TextArea rows={2} value={form.issuer_address} onChange={(e) => set('issuer_address', e.target.value)} />
          </Field>
        </div>
      </SectionCard>

      <SectionCard title={t('rental.settings.documents')}>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label={t('rental.settings.invoice_series')}>
            <TextInput value={form.invoice_series} onChange={(e) => set('invoice_series', e.target.value.toUpperCase())} maxLength={10} />
          </Field>
          <Field label={t('rental.settings.receipt_series')}>
            <TextInput value={form.receipt_series} onChange={(e) => set('receipt_series', e.target.value.toUpperCase())} maxLength={10} />
          </Field>
          <Field label={t('rental.settings.default_lead_days')}>
            <TextInput type="number" min={0} value={form.default_lead_days} onChange={(e) => set('default_lead_days', Number(e.target.value))} />
          </Field>
          <div className="hidden sm:block" />
          <Field label={t('rental.settings.owner_split')}>
            <TextInput type="number" min={0} max={100} step="0.01" value={form.default_owner_split_percent} onChange={(e) => set('default_owner_split_percent', Number(e.target.value))} />
          </Field>
          <Field label={t('rental.settings.agency_split')}>
            <TextInput type="number" min={0} max={100} step="0.01" value={form.default_agency_split_percent} onChange={(e) => set('default_agency_split_percent', Number(e.target.value))} />
          </Field>
        </div>
        <p className="text-xs text-text-secondary mt-3">{t('rental.settings.splits_hint')}</p>
      </SectionCard>

      <div className="flex justify-end gap-2">
        <SecondaryButton type="button" onClick={() => navigate('/settings')}>
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
