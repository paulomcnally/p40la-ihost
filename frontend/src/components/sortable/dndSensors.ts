import { MouseSensor, TouchSensor } from '@dnd-kit/core'

// Rechaza la activación del drag cuando el evento nace dentro de un elemento
// marcado con [data-no-dnd] (ej: botones del menú de la card), para que toda la
// card sea arrastrable sin romper el tap/click de sus controles internos.
function isInsideNoDnd(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  return Boolean(el && typeof el.closest === 'function' && el.closest('[data-no-dnd]'))
}

export class NoDndMouseSensor extends MouseSensor {
  static activators = [
    {
      eventName: 'onMouseDown' as const,
      handler: ({ nativeEvent: event }: { nativeEvent: MouseEvent }) => {
        if (isInsideNoDnd(event.target)) return false
        return event.button === 0
      },
    },
  ]
}

export class NoDndTouchSensor extends TouchSensor {
  static activators = [
    {
      eventName: 'onTouchStart' as const,
      handler: ({ nativeEvent: event }: { nativeEvent: TouchEvent }) => {
        return !isInsideNoDnd(event.target)
      },
    },
  ]
}