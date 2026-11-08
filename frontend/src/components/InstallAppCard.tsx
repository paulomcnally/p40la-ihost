import { useState } from 'react'
import { useI18nStore } from '../stores/i18nStore'
import { useInstallPrompt } from '../hooks/useInstallPrompt'
import { useToast } from './Toast'
import { Icon } from './Icons'

/**
 * SPEC-130: card de instalación de la PWA en Settings.
 * - Android/Chrome: botón "Instalar app" con beforeinstallprompt.
 * - iOS: instrucciones manuales (no existe API de instalación).
 * - Si ya corre en standalone, no se muestra.
 */
export default function InstallAppCard() {
  const { t } = useI18nStore()
  const { showToast } = useToast()
  const { canInstall, installed, isAndroid, isIOS, isStandalone, promptInstall } = useInstallPrompt()
  const [busy, setBusy] = useState(false)
  const [showIosHelp, setShowIosHelp] = useState(false)

  if (installed || isStandalone) return null
  if (!canInstall && !isIOS && !isAndroid) return null

  const handleInstall = async () => {
    setBusy(true)
    try {
      const outcome = await promptInstall()
      if (outcome === 'accepted') {
        showToast(t('settings.app.installed'), 'success')
      }
    } catch {
      showToast(t('settings.app.install_error'), 'error')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="bg-card rounded-ios shadow-ios p-4 mb-4 sm:mb-6">
      <div className="flex items-start gap-3">
        <span className="w-10 h-10 shrink-0 rounded-full bg-primary/10 text-primary flex items-center justify-center">
          <Icon name="download" className="w-5 h-5" />
        </span>
        <div className="flex-1 min-w-0">
          <h3 className="font-semibold">{t('settings.app.title')}</h3>
          <p className="text-sm text-text-secondary">{t('settings.app.subtitle')}</p>

          {canInstall && (
            <button
              type="button"
              onClick={handleInstall}
              disabled={busy}
              className="mt-3 inline-flex items-center gap-2 px-3 py-2 rounded-ios-sm bg-primary text-white text-sm font-medium disabled:opacity-60"
            >
              <Icon name="download" className="w-4 h-4" />
              {busy ? t('app.loading') : t('settings.app.install')}
            </button>
          )}

          {isIOS && (
            <>
              <button
                type="button"
                onClick={() => setShowIosHelp((value) => !value)}
                className="mt-3 inline-flex items-center gap-2 px-3 py-2 rounded-ios-sm border border-border text-sm font-medium text-primary"
              >
                <Icon name={showIosHelp ? 'chevron' : 'info'} className="w-4 h-4" />
                {showIosHelp ? t('app.close') : t('settings.app.ios_how')}
              </button>
              {showIosHelp && (
                <ol className="mt-3 space-y-2 text-sm text-text-secondary list-decimal list-inside">
                  <li>{t('settings.app.ios_step_1')}</li>
                  <li>{t('settings.app.ios_step_2')}</li>
                  <li>{t('settings.app.ios_step_3')}</li>
                </ol>
              )}
              {showIosHelp && <p className="mt-2 text-xs text-text-secondary">{t('settings.app.ios_hint')}</p>}
            </>
          )}

          {isAndroid && !canInstall && (
            <p className="mt-2 text-xs text-text-secondary">{t('settings.app.android_unavailable')}</p>
          )}
        </div>
      </div>
    </section>
  )
}
