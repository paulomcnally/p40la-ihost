import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { Icon } from '../components/Icons'
import LoadingSpinner from '../components/LoadingSpinner'
import ConfirmWordModal from '../components/ConfirmWordModal'
import Select from '../components/Select'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { useToast } from '../components/Toast'
import {
  DEBT_INFORMATION_FIELD_TYPES,
  DEBT_INFORMATION_TOKEN_KEYS,
  slugifyFieldKey,
  validateDebtInformationTemplate,
  type DebtInformationValidationIssue,
} from '../utils/debtInformation'
import type {
  DebtInformationField,
  DebtInformationFieldType,
  DebtInformationToken,
} from '../types'

interface BuilderField extends DebtInformationField {
  clientId: string
  persisted: boolean
}

function newBuilderField(index: number): BuilderField {
  return {
    clientId: `new-${Date.now()}-${index}-${Math.random().toString(36).slice(2)}`,
    persisted: false,
    key: '',
    label: '',
    type: 'text',
    options: [],
  }
}

function toBuilderFields(fields: DebtInformationField[]): BuilderField[] {
  return fields.map((field) => ({
    ...field,
    clientId: field.key,
    persisted: true,
  }))
}

function issueText(
  issue: DebtInformationValidationIssue,
  t: (key: string, fallback?: string) => string,
): string {
  let message = t(issue.key)
  if (issue.params) {
    for (const [key, value] of Object.entries(issue.params)) {
      message = message.split(`{${key}}`).join(String(value))
    }
  }
  return message
}

export default function DebtInformationTemplateFormPage() {
  const { id } = useParams()
  const isEdit = Boolean(id)
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const { showToast } = useToast()
  usePageTitle(isEdit ? t('deudas.template_edit') : t('deudas.template_create'))

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [fields, setFields] = useState<BuilderField[]>([])
  const [tokens, setTokens] = useState<DebtInformationToken[]>([])
  const [usageCount, setUsageCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [removeIndex, setRemoveIndex] = useState<number | null>(null)

  useEffect(() => {
    let active = true
    const load = async () => {
      try {
        const fallbackTokens = DEBT_INFORMATION_TOKEN_KEYS.map((key) => ({
          key,
          label: key,
          example: '',
        }))
        const tokenList = await api.debtInformationTemplates.tokens().catch(() => null)
        if (active) setTokens(tokenList && tokenList.length > 0 ? tokenList : fallbackTokens)

        if (isEdit) {
          const template = await api.debtInformationTemplates.get(Number(id))
          if (!template) {
            showToast(t('errors.generic'), 'error')
            navigate('/deudas/plantillas/informacion')
            return
          }
          if (!active) return
          setName(template.name)
          setDescription(template.description || '')
          setFields(toBuilderFields(template.fields || []))
          setUsageCount(template.usage_count || 0)
        }
      } catch (err) {
        if (active) {
          showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
        }
      } finally {
        if (active) setLoading(false)
      }
    }
    load()
    return () => {
      active = false
    }
  }, [id, isEdit, navigate, showToast])

  const issues = useMemo(
    () => validateDebtInformationTemplate(name, fields),
    [name, fields],
  )

  const nameIssue = issues.find(
    (issue) => issue.fieldIndex === undefined && issue.key.startsWith('deudas.validation_name'),
  )
  const globalIssue = issues.find(
    (issue) => issue.fieldIndex === undefined && !issue.key.startsWith('deudas.validation_name'),
  )
  const fieldIssues = useMemo(() => {
    const map = new Map<number, DebtInformationValidationIssue>()
    for (const issue of issues) {
      if (issue.fieldIndex !== undefined && !map.has(issue.fieldIndex)) {
        map.set(issue.fieldIndex, issue)
      }
    }
    return map
  }, [issues])

  const updateField = (index: number, patch: Partial<BuilderField>) => {
    setFields((prev) =>
      prev.map((field, i) => {
        if (i !== index) return field
        const next = { ...field, ...patch }
        if (!field.persisted && Object.prototype.hasOwnProperty.call(patch, 'label')) {
          next.key = slugifyFieldKey(next.label)
        }
        return next
      }),
    )
  }

  const changeFieldType = (index: number, type: DebtInformationFieldType) => {
    setFields((prev) =>
      prev.map((field, i) => {
        if (i !== index) return field
        if (type === 'computed') {
          return {
            ...field,
            type,
            required: false,
            placeholder: '',
            default_value: '',
            options: [],
            value_template: field.value_template || '',
          }
        }
        if (type === 'select') {
          return {
            ...field,
            type,
            options: field.options && field.options.length > 0 ? field.options : [],
            value_template: '',
          }
        }
        return {
          ...field,
          type,
          options: [],
          value_template: '',
        }
      }),
    )
  }

  const addField = () => {
    setFields((prev) => [...prev, newBuilderField(prev.length + 1)])
  }

  const removeField = (index: number) => {
    const field = fields[index]
    if (field.persisted) {
      setRemoveIndex(index)
      return
    }
    setFields((prev) => prev.filter((_, i) => i !== index))
  }

  const moveField = (index: number, direction: -1 | 1) => {
    setFields((prev) => {
      const target = index + direction
      if (target < 0 || target >= prev.length) return prev
      const copy = [...prev]
      const [item] = copy.splice(index, 1)
      copy.splice(target, 0, item)
      return copy
    })
  }

  const insertToken = (index: number, token: string) => {
    const field = fields[index]
    const current = field.value_template || ''
    updateField(index, { value_template: `${current}{{${token}}}` })
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (issues.length > 0) {
      showToast(issueText(issues[0], t), 'error')
      return
    }
    setSaving(true)
    try {
      const payload = {
        name: name.trim(),
        description: description.trim(),
        fields: fields.map(({ clientId: _clientId, persisted: _persisted, ...field }) => field),
      }
      if (isEdit) {
        await api.debtInformationTemplates.update(Number(id), payload)
      } else {
        await api.debtInformationTemplates.create(payload)
      }
      showToast(t('deudas.template_saved'), 'success')
      navigate('/deudas/plantillas/informacion')
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-4 sm:mb-5">
        <h2 className="text-xl sm:text-2xl font-bold">
          {isEdit ? t('deudas.template_edit') : t('deudas.template_create')}
        </h2>
        <p className="text-sm text-text-secondary mt-1">{t('deudas.template_form_hint')}</p>
      </div>

      {usageCount > 0 && (
        <div className="mb-4 rounded-ios-sm border border-warning/40 bg-warning/10 px-4 py-3 text-sm text-text">
          {t('deudas.template_usage_warning').replace('{n}', String(usageCount))}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
          <h3 className="font-semibold">{t('deudas.template_general')}</h3>
          <div>
            <label className="block text-sm font-medium mb-1">
              {t('deudas.template_name')} <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={100}
              className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
              placeholder={t('deudas.template_name_placeholder')}
              autoComplete="off"
            />
            {nameIssue && (
              <p className="text-xs text-danger mt-1">{issueText(nameIssue, t)}</p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('deudas.template_description')}</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              maxLength={500}
              rows={2}
              className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary resize-y"
              placeholder={t('deudas.template_description_placeholder')}
            />
          </div>
        </section>

        <section className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h3 className="font-semibold">{t('deudas.template_fields')}</h3>
              {globalIssue && (
                <p className="text-xs text-danger mt-1">{issueText(globalIssue, t)}</p>
              )}
            </div>
            <button
              type="button"
              onClick={addField}
              className="inline-flex items-center gap-1.5 px-3 py-2 rounded-ios-sm bg-primary text-white text-sm font-medium hover:opacity-90 min-h-[44px]"
            >
              <Icon name="plus" className="w-4 h-4" />
              {t('deudas.template_add_field')}
            </button>
          </div>

          {fields.length === 0 && (
            <div className="bg-card rounded-ios shadow-ios p-6 text-center text-sm text-text-secondary">
              {t('deudas.template_fields_empty')}
            </div>
          )}

          {fields.map((field, index) => {
            const issue = fieldIssues.get(index)
            return (
              <div key={field.clientId} className="bg-card rounded-ios shadow-ios p-4 space-y-3">
                <div className="flex items-start justify-between gap-2">
                  <span className="text-xs font-semibold text-text-secondary bg-bg rounded-full px-2 py-1 shrink-0">
                    {index + 1}
                  </span>
                  <div className="flex items-center gap-1 shrink-0">
                    <button
                      type="button"
                      onClick={() => moveField(index, -1)}
                      disabled={index === 0}
                      className="w-9 h-9 rounded-full flex items-center justify-center text-text-secondary hover:bg-bg disabled:opacity-30"
                      title={t('deudas.template_field_move_up')}
                      aria-label={t('deudas.template_field_move_up')}
                    >
                      <Icon name="chevron" className="w-4 h-4 -rotate-90" />
                    </button>
                    <button
                      type="button"
                      onClick={() => moveField(index, 1)}
                      disabled={index === fields.length - 1}
                      className="w-9 h-9 rounded-full flex items-center justify-center text-text-secondary hover:bg-bg disabled:opacity-30"
                      title={t('deudas.template_field_move_down')}
                      aria-label={t('deudas.template_field_move_down')}
                    >
                      <Icon name="chevron" className="w-4 h-4 rotate-90" />
                    </button>
                    <button
                      type="button"
                      onClick={() => removeField(index)}
                      className="w-9 h-9 rounded-full flex items-center justify-center text-danger hover:bg-danger/10"
                      title={t('deudas.template_field_remove')}
                      aria-label={t('deudas.template_field_remove')}
                    >
                      <Icon name="delete" className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="block text-sm font-medium mb-1">
                      {t('deudas.template_field_label')} <span className="text-danger">*</span>
                    </label>
                    <input
                      type="text"
                      value={field.label}
                      onChange={(e) => updateField(index, { label: e.target.value })}
                      maxLength={100}
                      className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                      placeholder={t('deudas.template_field_label_placeholder')}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">{t('deudas.template_field_type')}</label>
                    <Select
                      options={DEBT_INFORMATION_FIELD_TYPES.map((type) => ({
                        value: type,
                        label: t(`deudas.field_type_${type}`),
                      }))}
                      value={field.type}
                      onChange={(value) => changeFieldType(index, String(value) as DebtInformationFieldType)}
                    />
                  </div>
                </div>

                <div className="text-xs text-text-secondary">
                  {t('deudas.template_field_key')}: <code className="font-mono">{field.key || '—'}</code>
                </div>

                {field.type === 'computed' ? (
                  <div className="space-y-2">
                    <label className="block text-sm font-medium">
                      {t('deudas.template_field_expression')} <span className="text-danger">*</span>
                    </label>
                    <input
                      type="text"
                      value={field.value_template || ''}
                      onChange={(e) => updateField(index, { value_template: e.target.value })}
                      maxLength={500}
                      className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px] font-mono text-sm"
                      placeholder="{{cuota.monto_formateado}}"
                    />
                    <details className="rounded-ios-sm bg-bg/60 p-2">
                      <summary className="text-xs font-medium text-primary cursor-pointer">
                        {t('deudas.template_tokens')}
                      </summary>
                      <div className="flex flex-wrap gap-1.5 mt-2">
                        {tokens.map((token) => (
                          <button
                            key={token.key}
                            type="button"
                            onClick={() => insertToken(index, token.key)}
                            className="text-[11px] px-2 py-1 rounded-full bg-card border border-border hover:border-primary font-mono"
                            title={token.example ? `${token.label}: ${token.example}` : token.label}
                          >
                            {`{{${token.key}}}`}
                          </button>
                        ))}
                      </div>
                    </details>
                  </div>
                ) : (
                  <>
                    <div>
                      <label className="block text-sm font-medium mb-1">{t('deudas.template_field_placeholder')}</label>
                      <input
                        type="text"
                        value={field.placeholder || ''}
                        onChange={(e) => updateField(index, { placeholder: e.target.value })}
                        maxLength={200}
                        className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-1">{t('deudas.template_field_default')}</label>
                      <input
                        type="text"
                        value={field.default_value || ''}
                        onChange={(e) => updateField(index, { default_value: e.target.value })}
                        maxLength={500}
                        className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                      />
                    </div>
                    {field.type === 'select' && (
                      <div>
                        <label className="block text-sm font-medium mb-1">
                          {t('deudas.template_field_options')} <span className="text-danger">*</span>
                        </label>
                        <textarea
                          value={(field.options || []).join('\n')}
                          onChange={(e) => updateField(index, { options: e.target.value.split('\n') })}
                          rows={3}
                          className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary resize-y"
                          placeholder={t('deudas.template_field_options_hint')}
                        />
                      </div>
                    )}
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={Boolean(field.required)}
                        onChange={(e) => updateField(index, { required: e.target.checked })}
                        className="w-4 h-4 accent-primary"
                      />
                      {t('deudas.template_field_required')}
                    </label>
                  </>
                )}

                <div>
                  <label className="block text-sm font-medium mb-1">{t('deudas.template_field_help')}</label>
                  <input
                    type="text"
                    value={field.help_text || ''}
                    onChange={(e) => updateField(index, { help_text: e.target.value })}
                    maxLength={300}
                    className="w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]"
                  />
                </div>

                {issue && (
                  <p className="text-xs text-danger">{issueText(issue, t)}</p>
                )}
              </div>
            )
          })}
        </section>

        <div className="flex flex-col sm:flex-row gap-3 pb-6">
          <button
            type="button"
            onClick={() => navigate('/deudas/plantillas/informacion')}
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-bg text-text rounded-ios-sm hover:bg-border transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            <Icon name="cancel" className="w-4 h-4" />
            {t('app.cancel')}
          </button>
          <button
            type="submit"
            disabled={saving}
            className="flex-1 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 transition-colors flex items-center justify-center gap-2 min-h-[44px]"
          >
            {saving ? (
              <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
            ) : (
              <Icon name="save" className="w-4 h-4" />
            )}
            {saving ? t('app.saving') : t('app.save')}
          </button>
        </div>
      </form>

      {removeIndex !== null && (
        <ConfirmWordModal
          title={t('deudas.template_field_remove')}
          subtitle={t('deudas.template_field_remove_confirm')}
          confirmLabel={t('app.remove')}
          onConfirm={() => {
            setFields((prev) => prev.filter((_, i) => i !== removeIndex))
            setRemoveIndex(null)
          }}
          onCancel={() => setRemoveIndex(null)}
        />
      )}
    </div>
  )
}
