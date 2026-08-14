import { create } from 'zustand'
import type { Home, Currency } from '../types'
import { api } from '../api'

interface AppState {
  homes: Home[]
  currencies: Currency[]
  loading: boolean
  loadHomes: () => Promise<void>
  loadCurrencies: () => Promise<void>
  loadAll: () => Promise<void>
}

export const useAppStore = create<AppState>((set) => ({
  homes: [],
  currencies: [],
  loading: false,

  loadHomes: async () => {
    const homes = await api.homes.list()
    set({ homes: homes || [] })
  },

  loadCurrencies: async () => {
    const currencies = await api.currencies.list()
    set({ currencies: currencies || [] })
  },

  loadAll: async () => {
    set({ loading: true })
    const [homes, currencies] = await Promise.all([
      api.homes.list(),
      api.currencies.list(),
    ])
    set({ homes: homes || [], currencies: currencies || [], loading: false })
  },
}))
