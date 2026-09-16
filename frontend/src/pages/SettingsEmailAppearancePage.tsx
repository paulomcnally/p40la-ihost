import { useCallback, useEffect, useRef, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import { api } from '../api'
import { useToast } from '../components/Toast'

interface Palette {
  primary: string
  background: string
  card: string
  text: string
  muted: string
  border: string
}

const DEFAULT_PALETTE: Palette = {
  primary: '#007aff',
  background: '#f5f5f7',
  card: '#ffffff',
  text: '#1d1d1f',
  muted: '#8e8e93',
  border: '#e5e5ea',
}

const HEX_RE = /^#[0-9A-Fa-f]{6}$/

const COLOR_FIELDS: { key: keyof Palette; labelKey: string }[] = [
  { key: 'primary', labelKey: 'settings.email_appearance.color_primary' },
  { key: 'background', labelKey: 'settings.email_appearance.color_background' },
  { key: 'card', labelKey: 'settings.email_appearance.color_card' },
  { key: 'text', labelKey: 'settings.email_appearance.color_text' },
  { key: 'muted', labelKey: 'settings.email_appearance.color_muted' },
  { key: 'border', labelKey: 'settings.email_appearance.color_border' },
]

function paletteFromResponse(data: {
  email_color_primary?: string
  email_color_background?: string
  email_color_card?: string
  email_color_text?: string
  email_color_muted?: string
  email_color_border?: string
}): Palette {
  return {
    primary: data.email_color_primary ?? DEFAULT_PALETTE.primary,
    background: data.email_color_background ?? DEFAULT_PALETTE.background,
    card: data.email_color_card ?? DEFAULT_PALETTE.card,
    text: data.email_color_text ?? DEFAULT_PALETTE.text,
    muted: data.email_color_muted ?? DEFAULT_PALETTE.muted,
    border: data.email_color_border ?? DEFAULT_PALETTE.border,
  }
}

export default function SettingsEmailAppearancePage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.email_appearance.title'))
  const { showToast } = useToast()
  const [palette, setPalette] = useState<Palette>(DEFAULT_PALETTE)
  const [saving, setSaving] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [previewHtml, setPreviewHtml] = useState('')
  const debounceRef = useRef<number | null>(null)

  useEffect(() => {
    loadPalette()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadPalette = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) setPalette(paletteFromResponse(data))
    } catch {
      // ignore
    }
  }

  const loadPreview = useCallback(() => {
    if (debounceRef.current) window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(async () => {
      try {
        const data = await api.systemSettings.previewEmail({
          email_color_primary: palette.primary,
          email_color_background: palette.background,
          email_color_card: palette.card,
          email_color_text: palette.text,
          email_color_muted: palette.muted,
          email_color_border: palette.border,
        })
        if (data?.html) setPreviewHtml(data.html)
      } catch {
        // ignore
      }
    }, 400)
  }, [palette])

  useEffect(() => {
    loadPreview()
    return () => {
      if (debounceRef.current) window.clearTimeout(debounceRef.current)
    }
  }, [loadPreview])

  const setColor = (key: keyof Palette, value: string) => {
    setPalette((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = async () => {
    const invalid = COLOR_FIELDS.find((f) => !HEX_RE.test(palette[f.key]))
    if (invalid) {
      showToast(t('settings.email_appearance.invalid_hex'), 'error')
      return
    }
    setSaving(true)
    try {
      await api.systemSettings.update({
        email_color_primary: palette.primary,
        email_color_background: palette.background,
        email_color_card: palette.card,
        email_color_text: palette.text,
        email_color_muted: palette.muted,
        email_color_border: palette.border,
      })
      showToast(t('settings.email_appearance.saved'), 'success')
    } catch {
      showToast(t('settings.email_appearance.save_error'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleReset = async () => {
    if (!window.confirm(t('settings.email_appearance.reset_confirm'))) return
    setResetting(true)
    try {
      const data = await api.systemSettings.resetEmailPalette()
      if (data) setPalette(paletteFromResponse(data))
      showToast(t('settings.email_appearance.reset_done'), 'success')
    } catch {
      showToast(t('settings.email_appearance.save_error'), 'error')
    } finally {
      setResetting(false)
    }
  }

  const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card text-text min-h-[44px]'
  const labelCls = 'text-sm font-medium text-text-secondary mb-1'

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium">{t('settings.email_appearance.colors')}</div>
          <div className="text-sm text-text-secondary">{t('settings.email_appearance.colors_hint')}</div>
        </div>

        {COLOR_FIELDS.map(({ key, labelKey }) => {
          const value = palette[key]
          const valid = HEX_RE.test(value)
          return (
            <div key={key} className="px-4 py-3.5 border-b border-border">
              <label className={labelCls}>{t(labelKey)}</label>
              <div className="flex items-center gap-3">
                <input
                  type="color"
                  value={valid ? value : DEFAULT_PALETTE[key]}
                  onChange={(e) => setColor(key, e.target.value)}
                  className="w-14 h-11 p-1 border border-border rounded-ios-sm bg-card cursor-pointer shrink-0"
                  aria-label={t(labelKey)}
                />
                <input
                  type="text"
                  value={value}
                  onChange={(e) => setColor(key, e.target.value)}
                  spellCheck={false}
                  className={`${inputCls} ${valid ? '' : 'border-red-500'}`}
                />
              </div>
            </div>
          )
        })}

        <div className="px-4 py-3.5 flex flex-col sm:flex-row gap-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex-1 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
          >
            {saving ? '...' : t('app.save')}
          </button>
          <button
            onClick={handleReset}
            disabled={resetting}
            className="flex-1 px-4 py-2.5 rounded-ios-sm border border-border font-medium min-h-[44px] disabled:opacity-50"
          >
            {t('settings.email_appearance.reset')}
          </button>
        </div>
      </div>

      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border">
          <div className="font-medium">{t('settings.email_appearance.preview')}</div>
          <div className="text-sm text-text-secondary">{t('settings.email_appearance.preview_hint')}</div>
        </div>
        <div className="px-4 py-3.5">
          {previewHtml ? (
            <iframe
              sandbox="allow-same-origin"
              srcDoc={previewHtml}
              title={t('settings.email_appearance.preview')}
              className="w-full h-[480px] border border-border rounded-ios-sm bg-card"
            />
          ) : (
            <div className="h-40 flex items-center justify-center text-text-secondary text-sm">
              {t('settings.email_appearance.preview_loading')}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}