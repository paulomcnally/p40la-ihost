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

// Card individual reordenable dentro de un SortableGrid. El handle (si se
// provee) concentra los listeners de arrastre y, gracias a `touch-none`,
// no interfiere con el scroll nativo en mobile.
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
    ...(isDragging ? { opacity: 0.6, zIndex: 1 } : {}),
  }

  return (
    <div ref={setNodeRef} style={style} className={`${className} ${isDragging ? 'shadow-ios-lg' : ''}`}>
      {handle ? (
        <span
          ref={setActivatorNodeRef}
          {...attributes}
          {...listeners}
          aria-label={handleAriaLabel}
          className="absolute top-3 right-12 z-40 cursor-grab active:cursor-grabbing touch-none p-1 rounded-ios text-text-secondary hover:text-text hover:bg-bg transition-colors"
        >
          {handle}
        </span>
      ) : null}
      {children}
    </div>
  )
}