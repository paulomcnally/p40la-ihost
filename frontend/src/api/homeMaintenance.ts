import type {
  HomeMaintenance,
  HomeMaintenanceInput,
  HomeMaintenanceItem,
  HomeMaintenanceItemInput,
  HomeMaintenanceKind,
  HomeMaintenanceList,
} from '../types/homeMaintenance'

async function json<T>(path: string, options: RequestInit = {}): Promise<T | null> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  })
  if (res.status === 401) {
    window.location.href = '/login'
    return null
  }
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error(data.message || 'Request failed')
  }
  if (res.status === 204) return null
  return res.json()
}

const get = <T>(path: string) => json<T>(path, { method: 'GET' })
const post = <T>(path: string, body?: unknown) => json<T>(path, { method: 'POST', body: JSON.stringify(body ?? {}) })
const put = <T>(path: string, body: unknown) => json<T>(path, { method: 'PUT', body: JSON.stringify(body) })
const del = <T>(path: string) => json<T>(path, { method: 'DELETE' })

export const homeMaintenanceApi = {
  list: (homeId: number, params: { year?: number; kind?: HomeMaintenanceKind } = {}) => {
    const query = new URLSearchParams()
    if (params.year) query.set('year', String(params.year))
    if (params.kind) query.set('kind', params.kind)
    const suffix = query.toString() ? `?${query.toString()}` : ''
    return get<HomeMaintenanceList>(`/api/homes/${homeId}/maintenances${suffix}`)
  },
  get: (homeId: number, id: number) => get<HomeMaintenance>(`/api/homes/${homeId}/maintenances/${id}`),
  create: (homeId: number, body: HomeMaintenanceInput) =>
    post<HomeMaintenance>(`/api/homes/${homeId}/maintenances`, body),
  update: (homeId: number, id: number, body: HomeMaintenanceInput) =>
    put<HomeMaintenance>(`/api/homes/${homeId}/maintenances/${id}`, body),
  delete: (homeId: number, id: number) => del(`/api/homes/${homeId}/maintenances/${id}`),
  createItem: (homeId: number, id: number, body: HomeMaintenanceItemInput) =>
    post<HomeMaintenanceItem>(`/api/homes/${homeId}/maintenances/${id}/items`, body),
  updateItem: (homeId: number, id: number, itemId: number, body: HomeMaintenanceItemInput) =>
    put<HomeMaintenanceItem>(`/api/homes/${homeId}/maintenances/${id}/items/${itemId}`, body),
  deleteItem: (homeId: number, id: number, itemId: number) =>
    del(`/api/homes/${homeId}/maintenances/${id}/items/${itemId}`),
}
