import { create } from 'zustand'
import esDict from '../i18n/es.json'
import enDict from '../i18n/en.json'

const dictionaries: Record<string, Record<string, unknown>> = {
  es: esDict as unknown as Record<string, unknown>,
  en: enDict as unknown as Record<string, unknown>,
}

interface I18nState {
  lang: string
  dictionary: Record<string, unknown>
  load: (lang: string) => Promise<void>
  t: (key: string, fallback?: string) => string
}

const initialLang = localStorage.getItem('p40la-lang') || 'es'

export const useI18nStore = create<I18nState>((set, get) => ({
  lang: initialLang,
  dictionary: dictionaries[initialLang] || dictionaries.es,

  load: async (lang: string) => {
    try {
      const res = await fetch(`/i18n/${lang}.json`)
      if (!res.ok) throw new Error('Failed to load dictionary')
      const dictionary = await res.json()
      localStorage.setItem('p40la-lang', lang)
      document.documentElement.lang = lang
      set({ lang, dictionary })
    } catch (err) {
      console.error('i18n load error:', err)
    }
  },

  t: (key: string, fallback?: string) => {
    const { dictionary } = get()
    const parts = key.split('.')
    let value: unknown = dictionary
    for (const part of parts) {
      if (value == null || typeof value !== 'object') return fallback || key
      value = (value as Record<string, unknown>)[part]
    }
    if (typeof value === 'string') return value
    return fallback || key
  },
}))
