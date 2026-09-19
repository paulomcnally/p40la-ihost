import { useEffect, useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { usePageTitle } from '../hooks/usePageTitle'
import Toggle from '../components/Toggle'
import HelpPanel from '../components/HelpPanel'
import { api } from '../api'
import { useToast } from '../components/Toast'

export default function SettingsTelegramBotPage() {
  const { t } = useI18nStore()
  usePageTitle(t('settings.telegram_bot.title'))
  const { showToast } = useToast()
  const [tbEnabled, setTbEnabled] = useState(false)
  const [tbToken, setTbToken] = useState('')
  const [tbChatIDs, setTbChatIDs] = useState('')
  const [tbConfigured, setTbConfigured] = useState(false)
  const [tbSeparatorLength, setTbSeparatorLength] = useState(20)
  const [tbShowMonths, setTbShowMonths] = useState(1)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    loadTelegramBotSettings()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadTelegramBotSettings = async () => {
    try {
      const data = await api.systemSettings.get()
      if (data) {
        setTbEnabled(data.telegram_bot_enabled ?? false)
        setTbConfigured(data.telegram_bot_configured ?? false)
        setTbChatIDs((data.telegram_bot_chat_ids ?? []).join(', '))
        setTbSeparatorLength(data.telegram_bot_separator_length ?? 20)
        setTbShowMonths(data.telegram_bot_show_months ?? 1)
      }
    } catch {
      // ignore
    }
  }

  const tbActive = tbEnabled && tbConfigured
  const tbNeedsConfig = tbEnabled && !tbActive

  const handleTbEnabledChange = async (value: boolean) => {
    setTbEnabled(value)
    try {
      await api.systemSettings.update({ telegram_bot_enabled: value })
    } catch {
      setTbEnabled(!value)
      showToast(t('settings.telegram_bot.save_error'), 'error')
    }
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const body: Record<string, unknown> = {}
      if (tbToken.trim()) body.telegram_bot_token = tbToken.trim()
      body.telegram_bot_chat_ids = tbChatIDs.trim()
      body.telegram_bot_separator_length = Number(tbSeparatorLength)
      body.telegram_bot_show_months = Number(tbShowMonths)
      await api.systemSettings.update(body)
      const data = await api.systemSettings.get()
      if (data) {
        setTbConfigured(data.telegram_bot_configured ?? false)
        setTbChatIDs((data.telegram_bot_chat_ids ?? []).join(', '))
      }
      setTbToken('')
      showToast(t('settings.telegram_bot.saved'), 'success')
    } catch {
      showToast(t('settings.telegram_bot.save_error'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleSaveDisplay = async () => {
    setSaving(true)
    try {
      await api.systemSettings.update({
        telegram_bot_separator_length: Number(tbSeparatorLength),
        telegram_bot_show_months: Number(tbShowMonths),
      })
      showToast(t('settings.telegram_bot.saved'), 'success')
    } catch {
      showToast(t('settings.telegram_bot.save_error'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const inputCls = 'w-full px-3 py-2 border border-border rounded-ios-sm focus:outline-none focus:border-primary bg-card min-h-[44px]'
  const labelCls = 'text-sm font-medium text-text-secondary mb-1'

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-card rounded-ios shadow-ios overflow-hidden">
        <div className="px-4 py-3.5 border-b border-border flex items-center justify-between gap-3">
          <div className="flex-1">
            <div className="font-medium">{t('settings.telegram_bot.enable')}</div>
            <div className="text-sm text-text-secondary">{t('settings.telegram_bot.enable_hint')}</div>
          </div>
          <Toggle checked={tbEnabled} onChange={handleTbEnabledChange} />
        </div>

        {tbEnabled && (
          <>
            {tbConfigured ? (
              <div className="px-4 py-3.5 border-b border-border">
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300">
                  {t('settings.telegram_bot.configured_status')}
                </span>
              </div>
            ) : (
              <div className="px-4 py-3 border-b border-border">
                <p className="text-xs text-amber-600 dark:text-amber-400">{t('settings.telegram_bot.needs_config_hint')}</p>
              </div>
            )}

            <div className="px-4 py-3.5 border-b border-border">
              <div className="space-y-3">
                <div>
                  <label className={labelCls}>{t('settings.telegram_bot.token')}</label>
                  <input
                    type="password"
                    value={tbToken}
                    onChange={(e) => setTbToken(e.target.value)}
                    className={inputCls}
                    autoComplete="new-password"
                  />
                  {tbConfigured && (
                    <p className="text-xs text-text-secondary mt-1">{t('settings.telegram_bot.token_keep_hint')}</p>
                  )}
                </div>
                <div>
                  <label className={labelCls}>{t('settings.telegram_bot.chat_ids')}</label>
                  <input
                    type="text"
                    value={tbChatIDs}
                    onChange={(e) => setTbChatIDs(e.target.value)}
                    className={inputCls}
                  />
                  <p className="text-xs text-text-secondary mt-1">{t('settings.telegram_bot.chat_ids_hint')}</p>
                </div>
              </div>
            </div>

            <div className="px-4 py-3.5 flex flex-col sm:flex-row gap-3">
              <button
                onClick={handleSave}
                disabled={saving}
                className="flex-1 px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
              >
                {saving ? '...' : t('app.save')}
              </button>
            </div>

            <HelpPanel title={t('settings.telegram_bot.help.title')}>
              <ol className="list-decimal pl-5 space-y-2">
                <li>{t('settings.telegram_bot.help.step1')}</li>
                <li>{t('settings.telegram_bot.help.step2')}</li>
                <li>{t('settings.telegram_bot.help.step3')}</li>
                <li>{t('settings.telegram_bot.help.step4')}</li>
              </ol>
            </HelpPanel>

            <div className="px-4 py-3.5 border-t border-border">
              <div className="space-y-3">
                <div>
                  <label className={labelCls}>{t('settings.telegram_bot.separator_length')}</label>
                  <input
                    type="number"
                    min={0}
                    max={100}
                    value={tbSeparatorLength}
                    onChange={(e) => setTbSeparatorLength(Number(e.target.value))}
                    className={inputCls}
                  />
                  <p className="text-xs text-text-secondary mt-1">{t('settings.telegram_bot.separator_length_hint')}</p>
                </div>
                <div>
                  <label className={labelCls}>{t('settings.telegram_bot.show_months')}</label>
                  <input
                    type="number"
                    min={1}
                    max={12}
                    value={tbShowMonths}
                    onChange={(e) => setTbShowMonths(Number(e.target.value))}
                    className={inputCls}
                  />
                  <p className="text-xs text-text-secondary mt-1">{t('settings.telegram_bot.show_months_hint')}</p>
                </div>
                <button
                  onClick={handleSaveDisplay}
                  disabled={saving}
                  className="w-full px-4 py-2.5 rounded-ios-sm bg-primary text-white font-medium min-h-[44px] disabled:opacity-50"
                >
                  {saving ? '...' : t('app.save')}
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}