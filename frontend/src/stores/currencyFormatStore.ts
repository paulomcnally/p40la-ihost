import { create } from 'zustand'
import { api } from '../api'
import { DEFAULT_CURRENCY_FORMAT, formatCurrencyWithSymbol } from '../utils/currency'

interface CurrencyFormatState {
  thousandsSeparator: string
  decimalSeparator: string
  decimalDigits: number
  loaded: boolean
  load: () => Promise<void>
  formatMoney: (amount: number, symbol?: string) => string
  updateFormat: (patch: {
    thousandsSeparator?: string
    decimalSeparator?: string
    decimalDigits?: number
  }) => Promise<void>
}

// SPEC-113: el formateador se recrea cuando cambia la configuración para que
// los componentes que seleccionan `formatMoney` vuelvan a renderizar con el
// formato vigente. Con un closure estable sobre `get()` no se re-suscribían.
function createFormatter(
  thousandsSeparator: string,
  decimalSeparator: string,
  decimalDigits: number,
) {
  return (amount: number, symbol?: string) =>
    formatCurrencyWithSymbol(amount, symbol, { thousandsSeparator, decimalSeparator, decimalDigits })
}

export const useCurrencyFormatStore = create<CurrencyFormatState>((set, get) => ({
  thousandsSeparator: DEFAULT_CURRENCY_FORMAT.thousandsSeparator,
  decimalSeparator: DEFAULT_CURRENCY_FORMAT.decimalSeparator,
  decimalDigits: DEFAULT_CURRENCY_FORMAT.decimalDigits,
  loaded: false,

  formatMoney: createFormatter(
    DEFAULT_CURRENCY_FORMAT.thousandsSeparator,
    DEFAULT_CURRENCY_FORMAT.decimalSeparator,
    DEFAULT_CURRENCY_FORMAT.decimalDigits,
  ),

  load: async () => {
    try {
      const s = await api.systemSettings.get()
      if (s) {
        const thousandsSeparator = s.currency_thousands_separator ?? DEFAULT_CURRENCY_FORMAT.thousandsSeparator
        const decimalSeparator = s.currency_decimal_separator ?? DEFAULT_CURRENCY_FORMAT.decimalSeparator
        const decimalDigits = s.currency_decimal_digits ?? DEFAULT_CURRENCY_FORMAT.decimalDigits
        set({
          thousandsSeparator,
          decimalSeparator,
          decimalDigits,
          loaded: true,
          formatMoney: createFormatter(thousandsSeparator, decimalSeparator, decimalDigits),
        })
      } else {
        set({ loaded: true })
      }
    } catch {
      set({ loaded: true })
    }
  },

  updateFormat: async (patch) => {
    await api.systemSettings.update({
      currency_thousands_separator: patch.thousandsSeparator,
      currency_decimal_separator: patch.decimalSeparator,
      currency_decimal_digits: patch.decimalDigits,
    })
    const state = get()
    const thousandsSeparator = patch.thousandsSeparator ?? state.thousandsSeparator
    const decimalSeparator = patch.decimalSeparator ?? state.decimalSeparator
    const decimalDigits = patch.decimalDigits ?? state.decimalDigits
    set({
      thousandsSeparator,
      decimalSeparator,
      decimalDigits,
      formatMoney: createFormatter(thousandsSeparator, decimalSeparator, decimalDigits),
    })
  },
}))
