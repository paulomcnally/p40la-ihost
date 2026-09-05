import { useCallback, useRef } from 'react'

interface UseSortableOrderOptions<T extends { id: number }> {
  items: T[]
  setItems: (next: T[]) => void
  onPersist: (ids: number[]) => Promise<unknown>
  onError: (err: unknown) => void
}

// Hook reutilizable para persistir un reordenamiento: aplica el nuevo orden de
// forma optimista, persiste vía `onPersist` y revierte con `onError` si falla.
export function useSortableOrder<T extends { id: number }>({
  items,
  setItems,
  onPersist,
  onError,
}: UseSortableOrderOptions<T>) {
  const prevRef = useRef<T[]>(items)

  const handleReorder = useCallback(
    (ids: number[]) => {
      const byId = new Map(items.map((item) => [item.id, item]))
      const next = ids.map((id) => byId.get(id)).filter((x): x is T => Boolean(x))
      if (next.length !== ids.length) return
      prevRef.current = items
      setItems(next)
      onPersist(ids).catch((err) => {
        setItems(prevRef.current)
        onError(err)
      })
    },
    [items, setItems, onPersist, onError],
  )

  return { handleReorder }
}