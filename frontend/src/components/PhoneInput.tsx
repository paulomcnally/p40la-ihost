import { useEffect, useMemo, useRef, useState } from 'react'
import {
  AsYouType, getCountryCallingCode, parsePhoneNumberFromString, type CountryCode,
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
// Muestra selector de país con código telefónico, autoselecciona el país del
// sistema y guarda el número en formato E.164 (+código+número).
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

  const emit = (nextCountry: string, nextNational: string) => {
    const digits = onlyDigits(nextNational)
    const dial = dialCode(nextCountry)
    const next = digits && dial ? `+${dial}${digits}` : ''
    lastEmitted.current = next
    onChange(next)
  }

  const hydrate = (raw: string, fallbackCountry: string) => {
    if (!raw) {
      setNational('')
      return
    }
    const parsed = parsePhoneNumberFromString(raw, fallbackCountry as CountryCode)
    if (parsed?.country) {
      countryRef.current = parsed.country
      setCountry(parsed.country)
      setNational(new AsYouType(parsed.country).input(parsed.nationalNumber))
      return
    }
    setNational(new AsYouType(fallbackCountry as CountryCode).input(raw))
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
        emit(code, value)
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
    emit(code, formatted)
  }

  const handlePhone = (raw: string) => {
    // Pegar un número internacional cambia el país automáticamente.
    if (raw.trim().startsWith('+')) {
      const parsed = parsePhoneNumberFromString(raw)
      if (parsed?.country) {
        countryRef.current = parsed.country
        setCountry(parsed.country)
        const formatted = new AsYouType(parsed.country).input(parsed.nationalNumber)
        setNational(formatted)
        emit(parsed.country, formatted)
        return
      }
    }
    const formatted = new AsYouType(countryRef.current as CountryCode).input(onlyDigits(raw))
    setNational(formatted)
    emit(countryRef.current, formatted)
  }

  const dial = dialCode(country)

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
            disabled={disabled}
            placeholder={placeholder ?? t('rental.phone_placeholder')}
            aria-invalid={error}
            className={`w-full px-3 py-2 border rounded-ios-sm bg-card text-text placeholder:text-text-secondary focus:outline-none min-h-[44px] disabled:opacity-60 disabled:cursor-not-allowed ${
              error ? 'border-danger focus:border-danger' : 'border-border focus:border-primary'
            }`}
          />
        </div>
      </div>
      {dial && <p className="text-xs text-text-secondary pl-1">+{dial}</p>}
    </div>
  )
}
