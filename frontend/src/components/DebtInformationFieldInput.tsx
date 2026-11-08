import { useI18nStore } from '../stores/i18nStore'
import Select from './Select'
import type { DebtInformationField, DebtInformationValue } from '../types'

interface DebtInformationFieldInputProps {
  field: DebtInformationField
  value: DebtInformationValue | undefined
  onChange: (value: DebtInformationValue) => void
  disabled?: boolean
}

const inputClass =
  'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text focus:outline-none focus:border-primary min-h-[44px]'

export default function DebtInformationFieldInput({
  field,
  value,
  onChange,
  disabled = false,
}: DebtInformationFieldInputProps) {
  const { t } = useI18nStore()

  if (field.type === 'computed') return null

  switch (field.type) {
    case 'textarea':
      return (
        <textarea
          value={typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          disabled={disabled}
          rows={3}
          className={`${inputClass} resize-y`}
        />
      )

    case 'number':
      return (
        <input
          type="number"
          step="any"
          value={typeof value === 'number' || typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value === '' ? '' : Number(e.target.value))}
          placeholder={field.placeholder}
          disabled={disabled}
          className={inputClass}
        />
      )

    case 'date':
      return (
        <input
          type="date"
          value={typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          className={inputClass}
        />
      )

    case 'select':
      return (
        <Select
          options={(field.options || []).map((option) => ({ value: option, label: option }))}
          value={typeof value === 'string' ? value : ''}
          onChange={(next) => onChange(String(next))}
          placeholder={field.placeholder || t('app.select')}
          searchable={(field.options || []).length > 8}
          searchPlaceholder={t('app.search')}
          emptyLabel={t('app.no_results')}
        />
      )

    case 'checkbox':
      return (
        <label className="flex items-center gap-3 min-h-[44px] px-3 py-2 border border-border rounded-ios-sm bg-card text-text cursor-pointer">
          <input
            type="checkbox"
            checked={value === true}
            onChange={(e) => onChange(e.target.checked)}
            disabled={disabled}
            className="w-5 h-5 accent-primary"
          />
          <span className="text-sm">{value === true ? t('app.yes') : t('app.no')}</span>
        </label>
      )

    default:
      return (
        <input
          type="text"
          value={typeof value === 'string' ? value : ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          disabled={disabled}
          className={inputClass}
          autoComplete="off"
        />
      )
  }
}
