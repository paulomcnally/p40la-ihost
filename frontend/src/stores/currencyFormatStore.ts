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

export const useCurrencyFormatStore = create<CurrencyFormatState>((set, get) => ({
  thousandsSeparator: DEFAULT_CURRENCY_FORMAT.thousandsSeparator,
  decimalSeparator: DEFAULT_CURRENCY_FORMAT.decimalSeparator,
  decimalDigits: DEFAULT_CURRENCY_FORMAT.decimalDigits,
  loaded: false,

  load: async () => {
    try {
      const s = await api.systemSettings.get()
      if (s) {
        set({
          thousandsSeparator: s.currency_thousands_separator ?? DEFAULT_CURRENCY_FORMAT.thousandsSeparator,
          decimalSeparator: s.currency_decimal_separator ?? DEFAULT_CURRENCY_FORMAT.decimalSeparator,
          decimalDigits: s.currency_decimal_digits ?? DEFAULT_CURRENCY_FORMAT.decimalDigits,
          loaded: true,
        })
      } else {
        set({ loaded: true })
      }
    } catch {
      set({ loaded: true })
    }
  },

  formatMoney: (amount, symbol) => {
    const { thousandsSeparator, decimalSeparator, decimalDigits } = get()
    return formatCurrencyWithSymbol(amount, symbol, { thousandsSeparator, decimalSeparator, decimalDigits })
  },

  updateFormat: async (patch) => {
    await api.systemSettings.update({
      currency_thousands_separator: patch.thousandsSeparator,
      currency_decimal_separator: patch.decimalSeparator,
      currency_decimal_digits: patch.decimalDigits,
    })
    set((state) => ({
      thousandsSeparator: patch.thousandsSeparator ?? state.thousandsSeparator,
      decimalSeparator: patch.decimalSeparator ?? state.decimalSeparator,
      decimalDigits: patch.decimalDigits ?? state.decimalDigits,
    }))
  },
}))