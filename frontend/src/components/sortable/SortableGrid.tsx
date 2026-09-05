import { useMemo } from 'react'
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  rectSortingStrategy,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'

interface SortableGridProps<T extends { id: number }> {
  items: T[]
  onReorder: (ids: number[]) => void
  layout?: 'grid' | 'vertical'
  className?: string
  children: (item: T) => React.ReactNode
}

// Grid/listado sortable reutilizable. `onReorder` recibe el array de IDs en el
// nuevo orden al soltar el drag (mouse, touch o teclado).
export function SortableGrid<T extends { id: number }>({
  items,
  onReorder,
  layout = 'grid',
  className,
  children,
}: SortableGridProps<T>) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const ids = useMemo(() => items.map((item) => item.id), [items])
  const strategy = layout === 'grid' ? rectSortingStrategy : verticalListSortingStrategy

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const oldIndex = ids.indexOf(Number(active.id))
    const newIndex = ids.indexOf(Number(over.id))
    if (oldIndex < 0 || newIndex < 0) return
    onReorder(arrayMove(ids, oldIndex, newIndex))
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={ids} strategy={strategy}>
        <div className={className}>{items.map((item) => children(item))}</div>
      </SortableContext>
    </DndContext>
  )
}