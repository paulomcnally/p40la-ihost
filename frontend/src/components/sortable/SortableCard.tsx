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

// Card reordenable dentro de un SortableGrid. El drag se inicia SOLO desde el
// handle (activador): el resto de la card no es arrastrable. `touch-none` en el
// handle evita interferir con el scroll en touch (drag por long-press en mobile).
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
    <div ref={setNodeRef} style={style} className={`${className} ${isDragging ? 'shadow-ios-lg' : ''}`}>
      {handle ? (
        <span
          ref={setActivatorNodeRef}
          {...attributes}
          {...listeners}
          aria-label={handleAriaLabel}
          className="absolute inset-y-0 left-0 z-30 flex w-8 cursor-grab active:cursor-grabbing touch-none items-center justify-center rounded-l-ios text-text-secondary opacity-60 hover:opacity-100 hover:text-text hover:bg-bg/40 transition-opacity"
        >
          {handle}
        </span>
      ) : null}
      {children}
    </div>
  )
}