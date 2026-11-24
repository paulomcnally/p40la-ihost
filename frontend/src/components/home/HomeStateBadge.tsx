import { Badge } from '../rental/RentalUI'
import { useI18nStore } from '../../stores/i18nStore'
import type { HomeState } from '../../types'

type Tone = 'success' | 'warning' | 'danger' | 'neutral' | 'info' | 'accent'

// Colores por estado (REQ-017): verde, ámbar, azul, violeta y gris.
const TONES: Record<HomeState, Tone> = {
  listed: 'success',
  reserved: 'warning',
  rented: 'info',
  inhabited: 'accent',
  unused: 'neutral',
}

// HomeStateBadge muestra el estado derivado de una vivienda (SPEC-143).
export default function HomeStateBadge({ state }: { state: HomeState }) {
  const { t } = useI18nStore()
  return (
    <Badge tone={TONES[state] ?? 'neutral'}>
      <span className="inline-flex items-center gap-1.5">
        <span className="w-1.5 h-1.5 rounded-full bg-current" aria-hidden="true" />
        {t(`home.state.${state}`, state)}
      </span>
    </Badge>
  )
}
