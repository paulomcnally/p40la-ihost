import { useEffect, useRef, useState } from 'react'

type NumberInputProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type' | 'value' | 'onChange'> & {
  value: number | string
  onChange?: React.ChangeEventHandler<HTMLInputElement>
  onValueChange?: (value: number | '') => void
}

const BASE_CLASSES =
  'w-full px-3 py-2 border border-border rounded-ios-sm bg-card text-text placeholder:text-text-secondary ' +
  'focus:outline-none focus:border-primary min-h-[44px] disabled:opacity-60 disabled:cursor-not-allowed'

// cleanNumber permite vacío mientras se escribe, conserva un signo y un
// decimal, y elimina ceros a la izquierda (01 → 1, 002 → 2).
function cleanNumber(raw: string): string {
  let value = raw.replace(/,/g, '.').replace(/[^\d.-]/g, '')
  value = value.replace(/(?!^)-/g, '')
  const dot = value.indexOf('.')
  if (dot >= 0) {
    value = value.slice(0, dot + 1) + value.slice(dot + 1).replace(/\./g, '')
  }
  value = value.replace(/^(-?)0+(?=\d)/, '$1')
  return value
}

function parseValue(text: string): number | '' {
  if (text === '' || text === '-' || text === '.' || text === '-.') return ''
  const parsed = Number(text)
  return Number.isFinite(parsed) ? parsed : ''
}

// NumberInput es el único control válido para campos numéricos (UI-NUMBER-INPUT).
// Mantiene el texto mientras el usuario edita para permitir borrar el 0 y
// empezar a escribir, sin que el estado numérico lo fuerce de vuelta.
export default function NumberInput({
  value,
  onChange,
  onValueChange,
  className = '',
  inputMode = 'decimal',
  onFocus,
  onBlur,
  ...rest
}: NumberInputProps) {
  const [text, setText] = useState(() => (value === '' || value === null || value === undefined ? '' : String(value)))
  const focused = useRef(false)

  useEffect(() => {
    if (focused.current) return
    setText(value === '' || value === null || value === undefined ? '' : String(value))
  }, [value])

  const emit = (next: string) => {
    setText(next)
    if (onValueChange) onValueChange(parseValue(next))
    if (onChange) {
      const event = { target: { value: next }, currentTarget: { value: next } } as unknown as React.ChangeEvent<HTMLInputElement>
      onChange(event)
    }
  }

  return (
    <input
      {...rest}
      type="text"
      inputMode={inputMode}
      value={text}
      onChange={(e) => emit(cleanNumber(e.target.value))}
      onFocus={(e) => {
        focused.current = true
        onFocus?.(e)
      }}
      onBlur={(e) => {
        focused.current = false
        if (text === '-' || text === '.' || text === '-.') setText('')
        onBlur?.(e)
      }}
      className={`${BASE_CLASSES} ${className}`}
    />
  )
}
