import { useEffect, useMemo, useRef, useState } from 'react'
import {
  AsYouType, getCountryCallingCode, parsePhoneNumberFromString, validatePhoneNumberLength, type CountryCode,
} from 'libphonenumber-js/min'
import { api } from '../api'
import { COUNTRIES, DEFAULT_COUNTRY_CODE, countryName, resolveCountryCode } from '../constants/countries'
import { useI18nStore } from '../stores/i18nStore'
import Select from './Select'

interface PhoneInputProps {
  value: string
  onChange: (value: string) => void
  defaultCountry?: string
  disabled?: boolean
  placeholder?: string
  error?: boolean
}

let systemCountryPromise: Promise<string> | null = null

function loadSystemCountry(): Promise<string> {
  if (!systemCountryPromise) {
    systemCountryPromise = api.systemSettings.get()
      .then((data) => resolveCountryCode(data?.country_code || '') || DEFAULT_COUNTRY_CODE)
      .catch(() => DEFAULT_COUNTRY_CODE)
  }
  return systemCountryPromise
}

function dialCode(country: string): string {
  try {
    return getCountryCallingCode(country as CountryCode)
  } catch {
    return ''
  }
}

function onlyDigits(value: string): string {
  return value.replace(/\D/g, '')
}

// PhoneInput es el único control válido para teléfonos (UI-PHONE-INPUT).
// Selector de país con código telefónico, autoselección del país del sistema,
// validación de longitud/formato y persistencia en E.164 (+código+número).
export default function PhoneInput({
  value,
  onChange,
  defaultCountry,
  disabled = false,
  placeholder,
  error = false,
}: PhoneInputProps) {
  const { t, lang } = useI18nStore()
  const initialCountry = resolveCountryCode(defaultCountry || '') || DEFAULT_COUNTRY_CODE
  const [country, setCountry] = useState(initialCountry)
  const [national, setNational] = useState('')
  const [validationError, setValidationError] = useState('')
  const countryRef = useRef(initialCountry)
  const lastEmitted = useRef('')

  const options = useMemo(
    () => COUNTRIES.flatMap((item) => {
      const dial = dialCode(item.code)
      if (!dial) return []
      return [{ value: item.code, label: `${countryName(item.code, lang)} (+${dial})` }]
    }),
    [lang],
  )

  const parseNational = (digits: string, code: string) =>
    parsePhoneNumberFromString(digits, code as CountryCode)

  const emitNational = (code: string, formatted: string) => {
    const digits = onlyDigits(formatted)
    if (!digits) {
      setValidationError('')
      lastEmitted.current = ''
      onChange('')
      return
    }
    if (validatePhoneNumberLength(digits, code as CountryCode) === 'TOO_LONG') {
      setValidationError(t('rental.phone_too_long'))
      return
    }
    const parsed = parseNational(digits, code)
    if (parsed?.isValid()) {
      setValidationError('')
      lastEmitted.current = parsed.number
      onChange(parsed.number)
      return
    }
    // Incompleto o inválido: no se persiste un número a medias.
    setValidationError('')
    lastEmitted.current = ''
    onChange('')
  }

  const hydrate = (raw: string, fallbackCountry: string) => {
    if (!raw) {
      setNational('')
      setValidationError('')
      return
    }
    const parsed = parsePhoneNumberFromString(raw, fallbackCountry as CountryCode)
    if (parsed?.country) {
      countryRef.current = parsed.country
      setCountry(parsed.country)
      setNational(new AsYouType(parsed.country).input(parsed.nationalNumber))
      setValidationError('')
      return
    }
    setNational(new AsYouType(fallbackCountry as CountryCode).input(raw))
    setValidationError('')
  }

  useEffect(() => {
    if (value === lastEmitted.current) return
    hydrate(value, countryRef.current)
  }, [value])

  useEffect(() => {
    if (defaultCountry) {
      const code = resolveCountryCode(defaultCountry)
      if (code) {
        countryRef.current = code
        setCountry(code)
      }
      return
    }
    let active = true
    loadSystemCountry().then((code) => {
      if (!active) return
      countryRef.current = code
      setCountry(code)
      // Si había un número local precargado, se reemite con el país correcto.
      if (value && !value.trim().startsWith('+')) {
        hydrate(value, code)
        emitNational(code, value)
      }
    })
    return () => { active = false }
  }, [defaultCountry])

  const handleCountry = (next: string) => {
    const code = resolveCountryCode(next)
    if (!code) return
    countryRef.current = code
    setCountry(code)
    const formatted = new AsYouType(code as CountryCode).input(onlyDigits(national))
    setNational(formatted)
    emitNational(code, formatted)
  }

  const handlePhone = (raw: string) => {
    // Pegar un número internacional cambia el país automáticamente.
    if (raw.trim().startsWith('+')) {
      const parsed = parsePhoneNumberFromString(raw)
      if (parsed?.country) {
        if (validatePhoneNumberLength(parsed.nationalNumber, parsed.country) === 'TOO_LONG') {
          setValidationError(t('rental.phone_too_long'))
          return
        }
        countryRef.current = parsed.country
        setCountry(parsed.country)
        const formatted = new AsYouType(parsed.country).input(parsed.nationalNumber)
        setNational(formatted)
        emitNational(parsed.country, formatted)
        return
      }
    }
    const digits = onlyDigits(raw)
    if (validatePhoneNumberLength(digits, countryRef.current as CountryCode) === 'TOO_LONG') {
      setValidationError(t('rental.phone_too_long'))
      return
    }
    const formatted = new AsYouType(countryRef.current as CountryCode).input(digits)
    setNational(formatted)
    emitNational(countryRef.current, formatted)
  }

  const handleBlur = () => {
    const digits = onlyDigits(national)
    if (!digits) {
      setValidationError('')
      return
    }
    const parsed = parseNational(digits, countryRef.current)
    setValidationError(parsed?.isValid() ? '' : t('rental.phone_invalid'))
  }

  const dial = dialCode(country)
  const hasError = error || Boolean(validationError)

  return (
    <div className="space-y-1">
      <div className="flex gap-2 items-start">
        <div className="w-36 sm:w-48 shrink-0">
          <Select
            options={options}
            value={country}
            onChange={(v) => handleCountry(String(v))}
            searchable
            placeholder={t('rental.country')}
            searchPlaceholder={t('app.search')}
            emptyLabel={t('app.no_results')}
            disabled={disabled}
          />
        </div>
        <div className="flex-1 min-w-0">
          <input
            type="tel"
            inputMode="tel"
            value={national}
            onChange={(e) => handlePhone(e.target.value)}
            onBlur={handleBlur}
            disabled={disabled}
            maxLength={20}
            placeholder={placeholder ?? t('rental.phone_placeholder')}
            aria-invalid={hasError}
            className={`w-full px-3 py-2 border rounded-ios-sm bg-card text-text placeholder:text-text-secondary focus:outline-none min-h-[44px] disabled:opacity-60 disabled:cursor-not-allowed ${
              hasError ? 'border-danger focus:border-danger' : 'border-border focus:border-primary'
            }`}
          />
        </div>
      </div>
      {validationError ? (
        <p className="text-xs text-danger pl-1">{validationError}</p>
      ) : dial ? (
        <p className="text-xs text-text-secondary pl-1">+{dial}</p>
      ) : null}
    </div>
  )
}
