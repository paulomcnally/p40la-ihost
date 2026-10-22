import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import { api } from '../api'
import { Icon } from '../components/Icons'
import Select from '../components/Select'
import LoadingSpinner from '../components/LoadingSpinner'
import type {
  AutomationAccountInput,
  AutomationCredentialField,
  AutomationPlugin,
  Institution,
} from '../types'

const WEEKDAYS = [1, 2, 3, 4, 5, 6, 0]

interface FormState {
  institution_id: number
  identifier: string
  label: string
  plugin_name: string
  webhook_url: string
  schedule_enabled: boolean
  schedule_time: string
  schedule_frequency: string
  schedule_interval: number
}

const emptyForm: FormState = {
  institution_id: 0,
  identifier: '',
  label: '',
  plugin_name: '',
  webhook_url: '',
  schedule_enabled: false,
  schedule_time: '07:00',
  schedule_frequency: 'daily',
  schedule_interval: 2,
}

export default function AutomationFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const isEdit = !!id
  const { t } = useI18nStore()
  const { showToast } = useToast()
  usePageTitle(isEdit ? t('automation.edit') : t('automation.create'))

  // `webhook_url` ya no se edita en la UI de automation: se conserva el valor
  // existente al editar como fallback de transición (SPEC-111 REQ-018).
  const [form, setForm] = useState<FormState>(emptyForm)
  const [credentials, setCredentials] = useState<Record<string, string>>({})
  const [schema, setSchema] = useState<AutomationCredentialField[]>([])
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [plugins, setPlugins] = useState<AutomationPlugin[]>([])
  const [weeklyDays, setWeeklyDays] = useState<number[]>([1])
  const [monthlyDay, setMonthlyDay] = useState(15)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const init = async () => {
      try {
        const [inst, plug] = await Promise.all([
          api.institutions.list(),
          api.automation.plugins(),
        ])
        setInstitutions(inst || [])
        setPlugins(plug || [])

        if (isEdit) {
          const account = await api.automation.get(Number(id))
          if (account) {
            setForm({
              institution_id: account.institution_id,
              identifier: account.identifier,
              label: account.label || '',
              plugin_name: account.plugin_name || '',
              webhook_url: account.webhook_url || '',
              schedule_enabled: account.schedule_enabled,
              schedule_time: account.schedule_time || '07:00',
              schedule_frequency: account.schedule_frequency || 'daily',
              schedule_interval: account.schedule_interval ?? 2,
            })
            setWeeklyDays(parseWeeklyDays(account.schedule_days))
            setMonthlyDay(parseMonthlyDay(account.schedule_days))
          }
        }
      } catch (err) {
        showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
      } finally {
        setLoading(false)
      }
    }
    init()
  }, [id])

  useEffect(() => {
    if (!form.plugin_name) {
      setSchema([])
      return
    }
    let active = true
    api.automation.pluginSchema(form.plugin_name)
      .then((fields) => {
        if (active) setSchema(fields || [])
      })
      .catch(() => {
        if (active) setSchema([])
      })
    return () => { active = false }
  }, [form.plugin_name])

  const update = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const handlePluginChange = (value: string | number) => {
    setCredentials({})
    update('plugin_name', String(value))
  }

  const toggleWeekday = (day: number) => {
    setWeeklyDays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day].sort((a, b) => a - b)
    )
  }

  const validate = (): boolean => {
    if (!form.institution_id) {
      showToast(t('automation.institution_required'), 'error')
      return false
    }
    if (!form.identifier.trim()) {
      showToast(t('services.name'), 'error')
      return false
    }
    if (!form.plugin_name) {
      showToast(t('automation.plugin_required'), 'error')
      return false
    }
    if (form.schedule_enabled) {
      if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(form.schedule_time)) {
        showToast(t('automation.invalid_time'), 'error')
        return false
      }
      if (form.schedule_frequency === 'every_n_days' && (!Number.isFinite(form.schedule_interval) || form.schedule_interval < 1)) {
        showToast(t('automation.invalid_interval'), 'error')
        return false
      }
      if (form.schedule_frequency === 'weekly' && weeklyDays.length === 0) {
        showToast(t('automation.invalid_weekday'), 'error')
        return false
      }
      if (form.schedule_frequency === 'monthly' && (monthlyDay < 1 || monthlyDay > 28)) {
        showToast(t('automation.invalid_day'), 'error')
        return false
      }
    }
    if (!isEdit) {
      const missing = schema.some((field) => field.required && !credentials[field.key]?.trim())
      if (missing) {
        showToast(t('automation.credentials_required'), 'error')
        return false
      }
    }
    return true
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!validate()) return
    setSaving(true)
    try {
      const cleanCredentials: Record<string, string> = {}
      for (const [key, value] of Object.entries(credentials)) {
        if (value.trim() !== '') cleanCredentials[key] = value.trim()
      }

      const payload: AutomationAccountInput = {
        institution_id: form.institution_id,
        identifier: form.identifier.trim(),
        label: form.label.trim(),
        plugin_name: form.plugin_name,
        webhook_url: form.webhook_url.trim() || undefined,
        schedule_enabled: form.schedule_enabled,
        schedule_time: form.schedule_enabled ? form.schedule_time : undefined,
        schedule_frequency: form.schedule_frequency,
        schedule_interval: form.schedule_frequency === 'every_n_days' ? form.schedule_interval : undefined,
        schedule_days: form.schedule_enabled && form.schedule_frequency === 'weekly'
          ? weeklyDays.join(',')
          : form.schedule_enabled && form.schedule_frequency === 'monthly'
            ? String(monthlyDay)
            : undefined,
      }

      if (Object.keys(cleanCredentials).length > 0) {
        payload.credentials = JSON.stringify(cleanCredentials)
      } else if (!isEdit) {
        payload.credentials = '{}'
      }

      const saved = isEdit
        ? await api.automation.update(Number(id), payload)
        : await api.automation.create(payload)

      showToast(t('automation.saved'), 'success')
      navigate(saved ? `/automation/${saved.id}` : '/automation')
    } catch (err) {
      const message = err instanceof Error ? err.message : ''
      showToast(message || t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="max-w-2xl mx-auto bg-card rounded-ios shadow-ios p-4 sm:p-6">
      <h2 className="text-lg sm:text-xl font-bold mb-4 sm:mb-6">
        {isEdit ? t('automation.edit') : t('automation.create')}
      </h2>

      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-sm font-medium mb-1">{t('automation.institution')}</label>
          {institutions.length === 0 ? (
            <div className="flex flex-col sm:flex-row sm:items-center gap-2 text-sm text-text-secondary">
              <span>{t('automation.no_institutions')}</span>
              <button
                type="button"
                onClick={() => navigate('/institutions/new')}
                className="text-primary font-medium hover:underline text-left"
              >
                {t('automation.no_institutions_action')}
              </button>
            </div>
          ) : (
            <Select
              options={institutions.map((inst) => ({ value: inst.id, label: inst.name }))}
              value={form.institution_id || ''}
              onChange={(value) => update('institution_id', Number(value))}
              placeholder={t('automation.institution')}
              searchable
            />
          )}
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('automation.identifier')}</label>
            <input
              type="text"
              value={form.identifier}
              onChange={(e) => update('identifier', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
              placeholder="demo-account-001"
              autoComplete="off"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('automation.label')}</label>
            <input
              type="text"
              value={form.label}
              onChange={(e) => update('label', e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
              placeholder={t('automation.label_placeholder')}
              autoComplete="off"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('automation.plugin')}</label>
          {plugins.length === 0 ? (
            <p className="text-sm text-text-secondary">{t('automation.no_plugins')}</p>
          ) : (
            <Select
              options={plugins.map((p) => ({ value: p.name, label: `${p.name} v${p.version}` }))}
              value={form.plugin_name}
              onChange={handlePluginChange}
              placeholder={t('automation.plugin')}
              searchable
            />
          )}
        </div>

        {schema.length > 0 && (
          <div className="border border-border rounded-ios p-4 space-y-4">
            <div>
              <div className="text-sm font-medium">{t('automation.credentials')}</div>
              {isEdit && (
                <p className="text-xs text-text-secondary mt-1">{t('automation.credentials_keep')}</p>
              )}
            </div>
            {schema.map((field) => (
              <div key={field.key}>
                <label className="block text-sm font-medium mb-1">
                  {field.label}
                  {field.required && !isEdit && <span className="text-danger ml-0.5">*</span>}
                </label>
                <input
                  type={field.secret || field.type === 'password' ? 'password' : 'text'}
                  value={credentials[field.key] || ''}
                  onChange={(e) => setCredentials((prev) => ({ ...prev, [field.key]: e.target.value }))}
                  className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                  placeholder={field.label}
                  autoComplete="new-password"
                />
              </div>
            ))}
          </div>
        )}

        <div className="border border-border rounded-ios p-4 space-y-4">
          <label className="flex items-center justify-between gap-3 cursor-pointer">
            <span className="text-sm font-medium">{t('automation.schedule_enabled')}</span>
            <input
              type="checkbox"
              checked={form.schedule_enabled}
              onChange={(e) => update('schedule_enabled', e.target.checked)}
              className="w-5 h-5 accent-primary"
            />
          </label>

          {form.schedule_enabled && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">{t('automation.time')}</label>
                  <input
                    type="time"
                    value={form.schedule_time}
                    onChange={(e) => update('schedule_time', e.target.value)}
                    className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">{t('automation.frequency')}</label>
                  <select
                    value={form.schedule_frequency}
                    onChange={(e) => update('schedule_frequency', e.target.value)}
                    className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                  >
                    <option value="daily">{t('automation.frequency_daily')}</option>
                    <option value="every_n_days">{t('automation.frequency_days')}</option>
                    <option value="weekly">{t('automation.frequency_weekly')}</option>
                    <option value="monthly">{t('automation.frequency_monthly')}</option>
                  </select>
                </div>
              </div>

              {form.schedule_frequency === 'every_n_days' && (
                <div>
                  <label className="block text-sm font-medium mb-1">{t('automation.interval')}</label>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min={1}
                      max={365}
                      value={form.schedule_interval}
                      onChange={(e) => update('schedule_interval', Number(e.target.value))}
                      className="w-28 px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                    />
                    <span className="text-sm text-text-secondary">{t('automation.interval_days')}</span>
                  </div>
                </div>
              )}

              {form.schedule_frequency === 'weekly' && (
                <div>
                  <label className="block text-sm font-medium mb-2">{t('automation.days')}</label>
                  <div className="flex flex-wrap gap-2">
                    {WEEKDAYS.map((day) => (
                      <button
                        key={day}
                        type="button"
                        onClick={() => toggleWeekday(day)}
                        aria-pressed={weeklyDays.includes(day)}
                        className={`px-3 py-2 rounded-ios-sm text-sm border transition-colors min-h-[40px] ${
                          weeklyDays.includes(day)
                            ? 'bg-primary/10 border-primary text-primary font-medium'
                            : 'bg-card border-border text-text-secondary hover:border-primary/50'
                        }`}
                      >
                        {t(`automation.weekday_${day}`)}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {form.schedule_frequency === 'monthly' && (
                <div>
                  <label className="block text-sm font-medium mb-1">{t('automation.day_of_month')}</label>
                  <input
                    type="number"
                    min={1}
                    max={28}
                    value={monthlyDay}
                    onChange={(e) => setMonthlyDay(Number(e.target.value))}
                    className="w-28 px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                  />
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t border-border">
          <button
            type="button"
            onClick={() => navigate(isEdit ? `/automation/${id}` : '/automation')}
            className="px-4 py-2.5 rounded-ios-sm border border-border hover:bg-bg transition-colors min-h-[44px]"
          >
            {t('app.cancel')}
          </button>
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium hover:opacity-90 transition-opacity min-h-[44px] disabled:opacity-50 flex items-center gap-2"
          >
            <Icon name="save" className="w-4 h-4" />
            {saving ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </form>
    </div>
  )
}

function parseWeeklyDays(raw?: string | null): number[] {
  if (!raw) return [1]
  const days = raw.split(',').map((part) => Number(part.trim())).filter((day) => day >= 0 && day <= 6)
  return days.length > 0 ? days : [1]
}

function parseMonthlyDay(raw?: string | null): number {
  if (!raw) return 15
  const value = Number(raw)
  return value >= 1 && value <= 28 ? value : 15
}
