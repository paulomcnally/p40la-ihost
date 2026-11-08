import { useEffect, useRef, useState, type Dispatch, type KeyboardEvent, type RefObject, type SetStateAction } from 'react'

// SPEC-126: regla única para modales de confirmación por palabra.
// Normaliza la palabra (sin espacios al inicio/fin y sin distinguir mayúsculas)
// y expone el foco inicial + el manejo de Enter para que todos los modales
// existentes y futuros se comporten igual.

export function normalizeConfirmWord(value: string): string {
  return value.trim().toLowerCase()
}

interface UseConfirmWordInputOptions {
  word: string
  onConfirm: () => void
  disabled?: boolean
}

interface UseConfirmWordInputResult {
  value: string
  setValue: Dispatch<SetStateAction<string>>
  inputRef: RefObject<HTMLInputElement | null>
  isValid: boolean
  handleKeyDown: (e: KeyboardEvent<HTMLInputElement>) => void
  reset: () => void
}

export function useConfirmWordInput({
  word,
  onConfirm,
  disabled = false,
}: UseConfirmWordInputOptions): UseConfirmWordInputResult {
  const [value, setValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const isValid = normalizeConfirmWord(value) === normalizeConfirmWord(word)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return
    if (e.nativeEvent.isComposing) return
    // Evita envíos de formulario o navegaciones accidentales.
    e.preventDefault()
    if (!isValid || disabled) return
    onConfirm()
  }

  const reset = () => setValue('')

  return { value, setValue, inputRef, isValid, handleKeyDown, reset }
}
