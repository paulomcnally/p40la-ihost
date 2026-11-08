import { useCallback, useEffect, useState } from 'react'

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>
}

interface InstallState {
  prompt: BeforeInstallPromptEvent | null
  installed: boolean
}

type InstallSubscriber = (state: InstallState) => void

// El evento beforeinstallprompt puede dispararse antes de que el usuario abra
// Settings. Se captura a nivel de módulo para no perderlo y se notifica a los
// componentes suscriptos (SPEC-130, REQ-009).
let installState: InstallState = { prompt: null, installed: false }
const subscribers = new Set<InstallSubscriber>()

function setInstallState(next: Partial<InstallState>) {
  installState = { ...installState, ...next }
  subscribers.forEach((subscriber) => subscriber(installState))
}

if (typeof window !== 'undefined') {
  window.addEventListener('beforeinstallprompt', (event) => {
    // Evita el mini-infobar automático; la app muestra su propia acción.
    event.preventDefault()
    setInstallState({ prompt: event as BeforeInstallPromptEvent })
  })

  window.addEventListener('appinstalled', () => {
    setInstallState({ prompt: null, installed: true })
  })
}

function detectIOS(): boolean {
  if (typeof navigator === 'undefined') return false
  const iPadOS = navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1
  return /iPad|iPhone|iPod/.test(navigator.userAgent) || iPadOS
}

function detectAndroid(): boolean {
  if (typeof navigator === 'undefined') return false
  return /Android/i.test(navigator.userAgent)
}

function detectStandalone(): boolean {
  if (typeof window === 'undefined') return false
  const nav = navigator as Navigator & { standalone?: boolean }
  return window.matchMedia('(display-mode: standalone)').matches || nav.standalone === true
}

/**
 * SPEC-130: encapsula el ciclo de vida de la instalación PWA.
 * Android/Chrome expone beforeinstallprompt; iOS no tiene API y se resuelve
 * con instrucciones manuales.
 */
export function useInstallPrompt() {
  const [state, setState] = useState<InstallState>(installState)
  const [standalone, setStandalone] = useState(detectStandalone)
  const [isIOS] = useState(detectIOS)
  const [isAndroid] = useState(detectAndroid)

  useEffect(() => {
    subscribers.add(setState)
    setState(installState)

    const media = window.matchMedia('(display-mode: standalone)')
    const handleDisplayMode = () => {
      const nav = navigator as Navigator & { standalone?: boolean }
      setStandalone(media.matches || nav.standalone === true)
    }
    media.addEventListener?.('change', handleDisplayMode)

    return () => {
      subscribers.delete(setState)
      media.removeEventListener?.('change', handleDisplayMode)
    }
  }, [])

  const promptInstall = useCallback(async (): Promise<'accepted' | 'dismissed' | 'unavailable'> => {
    const current = installState.prompt
    if (!current) return 'unavailable'

    await current.prompt()
    const choice = await current.userChoice
    setInstallState({ prompt: null })
    return choice.outcome
  }, [])

  return {
    canInstall: Boolean(state.prompt) && !state.installed && !standalone,
    installed: state.installed,
    isAndroid,
    isIOS,
    isStandalone: standalone,
    promptInstall,
  }
}
