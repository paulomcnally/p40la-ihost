import type { ReactNode, CSSProperties } from 'react'
import { CSS } from '@dnd-kit/utilities'
import { useSortable } from '@dnd-kit/sortable'

interface SortableCardProps {
  id: number
  children: ReactNode
  handle?: ReactNode
  handleAriaLabel?: string
  className?: string
}

// Card reordenable dentro de un SortableGrid. Toda la card es arrastrable
// (listeners en el nodo). El handle (si se provee) actúa como activador de
// teclado y como pista visual: sutil en hover (desktop), siempre visible en
// mobile. `touch-none` en el handle evita interferir con el scroll en touch;
// el resto de la card mantiene el scroll nativo (drag por long-press).
export function SortableCard({ id, children, handle, handleAriaLabel, className = '' }: SortableCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id })

  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    ...(isDragging ? { opacity: 0.7, zIndex: 1 } : {}),
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={`group ${className} cursor-grab active:cursor-grabbing ${isDragging ? 'shadow-ios-lg' : ''}`}
    >
      {handle ? (
        <span
          ref={setActivatorNodeRef}
          {...attributes}
          {...listeners}
          aria-label={handleAriaLabel}
          className="absolute top-3 left-1/2 -translate-x-1/2 z-30 cursor-grab active:cursor-grabbing touch-none p-1 rounded-ios text-text-secondary opacity-60 group-hover:opacity-100 group-hover:text-text transition-opacity"
        >
          {handle}
        </span>
      ) : null}
      {children}
    </div>
  )
}