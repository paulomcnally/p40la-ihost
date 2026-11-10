import { useEffect, useMemo, useRef, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useToast } from '../components/Toast'
import { Icon } from '../components/Icons'
import ConfirmWordModal from '../components/ConfirmWordModal'
import {
  BANK_TRANSFER_SUGGESTED_KEYS,
  bankAccountTypeLabel,
  buildBankAccountVariableContext,
  fieldsToRows,
  rowsToFields,
  unknownBankFormatVariables,
  validateBankFormatRows,
  type BankFormatFieldRow,
} from '../utils/bank'
import { resolveDebtInformationVariables } from '../utils/debtInformation'
import { api } from '../api'
import type {
  Bank,
  BankAccount,
  BankTransferFormat,
  BankTransferKind,
  DebtInformationVariable,
} from '../types'

interface TransferFormatEditorProps {
  accountId: number
  account: BankAccount
  bank: Bank | null
  kind: BankTransferKind
  initialFormat?: BankTransferFormat
  variables: DebtInformationVariable[]
  onSaved: () => void
}

const inputClass =
  'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]'

function normalizeSearch(value: string): string {
  return value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase()
}

export default function TransferFormatEditor({
  accountId,
  account,
  bank,
  kind,
  initialFormat,
  variables,
  onSaved,
}: TransferFormatEditorProps) {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const [rows, setRows] = useState<BankFormatFieldRow[]>(() => fieldsToRows(initialFormat?.fields, kind))
  const [variableSearch, setVariableSearch] = useState('')
  const [activeRow, setActiveRow] = useState(0)
  const [saving, setSaving] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const valueRefs = useRef<Record<number, HTMLInputElement | null>>({})

  useEffect(() => {
    setRows(fieldsToRows(initialFormat?.fields, kind))
  }, [accountId, kind, initialFormat?.id, initialFormat?.updated_at])

  const allowedVariables = useMemo(() => new Set(variables.map((variable) => variable.key)), [variables])

  const filteredVariables = useMemo(() => {
    const term = normalizeSearch(variableSearch.trim())
    if (!term) return variables
    return variables.filter(
      (variable) =>
        normalizeSearch(variable.label).includes(term) ||
        normalizeSearch(variable.key).includes(term),
    )
  }, [variables, variableSearch])

  const context = useMemo(
    () => buildBankAccountVariableContext(bank, account, bankAccountTypeLabel(account.account_type, t)),
    [bank, account, t],
  )

  const preview = useMemo(() => {
    const resolved: Record<string, string> = {}
    for (const row of rows) {
      const key = row.key.trim()
      if (!key) continue
      resolved[key] = resolveDebtInformationVariables(row.value, context)
    }
    return JSON.stringify(resolved, null, 2)
  }, [rows, context])

  const updateRow = (index: number, patch: Partial<BankFormatFieldRow>) => {
    setRows((prev) => prev.map((row, i) => (i === index ? { ...row, ...patch } : row)))
  }

  const addRow = () => {
    setRows((prev) => [...prev, { key: '', value: '' }])
  }

  const removeRow = (index: number) => {
    setRows((prev) => prev.filter((_, i) => i !== index))
  }

  const restoreSuggested = () => {
    setRows(BANK_TRANSFER_SUGGESTED_KEYS[kind].map((key) => ({ key, value: '' })))
  }

  const insertVariable = (index: number, key: string) => {
    const current = rows[index]?.value || ''
    const input = valueRefs.current[index]
    const start = input?.selectionStart ?? current.length
    const end = input?.selectionEnd ?? start
    const token = `{${key}}`
    const next = `${current.slice(0, start)}${token}${current.slice(end)}`
    updateRow(index, { value: next })
    requestAnimationFrame(() => {
      input?.focus()
      const position = start + token.length
      input?.setSelectionRange(position, position)
    })
  }

  const handleSave = async () => {
    const validationKey = validateBankFormatRows(rows)
    if (validationKey) {
      showToast(t(validationKey), 'error')
      return
    }
    const unknown = unknownBankFormatVariables(rows, allowedVariables)
    if (unknown.length > 0) {
      showToast(`${t('banks.validation_variable')} {${unknown.join('}, {')}}`, 'error')
      return
    }
    setSaving(true)
    try {
      await api.bankAccounts.saveFormat(accountId, kind, rowsToFields(rows))
      showToast(t('banks.format_saved'), 'success')
      onSaved()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    setSaving(true)
    try {
      await api.bankAccounts.deleteFormat(accountId, kind)
      setDeleteConfirm(false)
      setRows(BANK_TRANSFER_SUGGESTED_KEYS[kind].map((key) => ({ key, value: '' })))
      showToast(t('banks.format_deleted'), 'success')
      onSaved()
    } catch (err) {
      showToast(err instanceof Error ? err.message : t('errors.generic'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const copyJSON = async () => {
    try {
      await navigator.clipboard.writeText(preview)
      showToast(t('app.copied'), 'success')
    } catch {
      showToast(t('errors.generic'), 'error')
    }
  }

  const kindLabel = kind === 'ach' ? t('banks.ach') : t('banks.transfer')

  return (
    <section className="bg-card rounded-ios shadow-ios p-4 sm:p-5 space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h3 className="font-semibold">{kindLabel}</h3>
          {!initialFormat && (
            <p className="text-xs text-text-secondary mt-0.5">{t('banks.format_empty_hint')}</p>
          )}
        </div>
        {initialFormat && (
          <button
            type="button"
            onClick={() => setDeleteConfirm(true)}
            disabled={saving}
            className="text-sm font-medium text-danger hover:underline min-h-[36px]"
          >
            {t('banks.format_delete')}
          </button>
        )}
      </div>

      <div className="space-y-2">
        {rows.map((row, index) => (
          <div key={index} className="grid grid-cols-1 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.6fr)_auto] gap-2">
            <input
              type="text"
              value={row.key}
              onChange={(e) => updateRow(index, { key: e.target.value })}
              placeholder={t('banks.format_key')}
              maxLength={40}
              className={`${inputClass} font-mono text-sm`}
              autoComplete="off"
            />
            <input
              ref={(element) => {
                valueRefs.current[index] = element
              }}
              type="text"
              value={row.value}
              onChange={(e) => updateRow(index, { value: e.target.value })}
              onFocus={() => setActiveRow(index)}
              placeholder={t('banks.format_value')}
              maxLength={500}
              className={`${inputClass} text-sm`}
              autoComplete="off"
            />
            <button
              type="button"
              onClick={() => removeRow(index)}
              className="w-9 h-9 rounded-full flex items-center justify-center text-danger hover:bg-danger/10 self-center"
              title={t('banks.format_remove_field')}
              aria-label={t('banks.format_remove_field')}
            >
              <Icon name="delete" className="w-4 h-4" />
            </button>
          </div>
        ))}
      </div>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={addRow}
          className="inline-flex items-center gap-1.5 px-3 py-2 rounded-ios-sm bg-bg text-text text-sm font-medium hover:bg-border min-h-[44px]"
        >
          <Icon name="plus" className="w-4 h-4" />
          {t('banks.format_add_field')}
        </button>
        <button
          type="button"
          onClick={restoreSuggested}
          className="inline-flex items-center gap-1.5 px-3 py-2 rounded-ios-sm bg-bg text-text text-sm font-medium hover:bg-border min-h-[44px]"
        >
          <Icon name="refresh" className="w-4 h-4" />
          {t('banks.format_restore_suggested')}
        </button>
      </div>

      <details className="rounded-ios-sm bg-bg/60 p-2">
        <summary className="text-xs font-medium text-primary cursor-pointer">
          {t('banks.variables')}
        </summary>
        <p className="text-[11px] text-text-secondary mt-1">{t('banks.variables_hint')}</p>
        <input
          type="text"
          value={variableSearch}
          onChange={(e) => setVariableSearch(e.target.value)}
          placeholder={t('banks.variables_search')}
          className="mt-2 w-full px-2 py-1.5 text-xs border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[36px]"
        />
        <div className="flex flex-wrap gap-1.5 mt-2">
          {filteredVariables.map((variable) => (
            <button
              key={variable.key}
              type="button"
              onClick={() => {
                if (rows.length > 0) {
                  insertVariable(Math.min(activeRow, rows.length - 1), variable.key)
                }
              }}
              className="text-[11px] px-2 py-1 rounded-full bg-card border border-border hover:border-primary"
              title={`${variable.description || variable.label}. Ejemplo: ${variable.example}`}
            >
              <span className="font-medium">{variable.label}</span>
              <code className="ml-1 font-mono text-text-secondary">{`{${variable.key}}`}</code>
            </button>
          ))}
          {filteredVariables.length === 0 && (
            <span className="text-xs text-text-secondary">{t('app.no_results')}</span>
          )}
        </div>
        <p className="text-[11px] text-text-secondary mt-1">
          {t('banks.variables_insert_hint')}
        </p>
      </details>

      <div>
        <div className="flex items-center justify-between gap-2 mb-1">
          <h4 className="text-sm font-medium">{t('banks.preview')}</h4>
          <button
            type="button"
            onClick={copyJSON}
            className="inline-flex items-center gap-1 text-xs text-primary hover:underline min-h-[32px]"
          >
            <Icon name="copy" className="w-3.5 h-3.5" />
            {t('banks.copy_json')}
          </button>
        </div>
        <pre className="bg-bg/70 border border-border/60 rounded-ios-sm p-3 text-xs text-text overflow-x-auto whitespace-pre-wrap break-words">
          {preview}
        </pre>
      </div>

      <div className="flex justify-end">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="inline-flex items-center gap-2 px-4 py-2.5 bg-primary text-white rounded-ios-sm hover:opacity-90 disabled:opacity-50 min-h-[44px]"
        >
          {saving ? (
            <div className="w-4 h-4 rounded-full border-2 border-white/30 border-t-white animate-spin" />
          ) : (
            <Icon name="save" className="w-4 h-4" />
          )}
          {saving ? t('app.saving') : t('banks.format_save')}
        </button>
      </div>

      {deleteConfirm && (
        <ConfirmWordModal
          title={t('banks.format_delete')}
          subtitle={t('banks.format_delete_confirm')}
          confirmLabel={t('app.delete')}
          onConfirm={handleDelete}
          onCancel={() => setDeleteConfirm(false)}
        />
      )}
    </section>
  )
}
