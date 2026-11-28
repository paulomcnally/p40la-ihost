import type {
  HomeWorkRequest,
  HomeWorkRequestAttachment,
  HomeWorkRequestCategory,
  HomeWorkRequestConvertInput,
  HomeWorkRequestConvertResult,
  HomeWorkRequestInput,
  HomeWorkRequestList,
  HomeWorkRequestMediaType,
  HomeWorkRequestStatus,
} from '../types/homeWorkRequest'

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

function upload<T>(path: string, file: File, mediaType: HomeWorkRequestMediaType): Promise<T | null> {
  const form = new FormData()
  form.append('file', file)
  form.append('media_type', mediaType)
  return fetch(path, { method: 'POST', body: form }).then(async (res) => {
    if (res.status === 401) {
      window.location.href = '/login'
      return null
    }
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.message || 'Request failed')
    return data as T
  })
}

export const homeWorkRequestApi = {
  list: (homeId: number, params: { status?: HomeWorkRequestStatus | ''; category?: HomeWorkRequestCategory | '' } = {}) => {
    const query = new URLSearchParams()
    if (params.status) query.set('status', params.status)
    if (params.category) query.set('category', params.category)
    const suffix = query.toString() ? `?${query.toString()}` : ''
    return get<HomeWorkRequestList>(`/api/homes/${homeId}/workrequests${suffix}`)
  },
  get: (homeId: number, id: number) => get<HomeWorkRequest>(`/api/homes/${homeId}/workrequests/${id}`),
  create: (homeId: number, body: HomeWorkRequestInput) =>
    post<HomeWorkRequest>(`/api/homes/${homeId}/workrequests`, body),
  update: (homeId: number, id: number, body: HomeWorkRequestInput) =>
    put<HomeWorkRequest>(`/api/homes/${homeId}/workrequests/${id}`, body),
  delete: (homeId: number, id: number) => del(`/api/homes/${homeId}/workrequests/${id}`),
  setStatus: (homeId: number, id: number, status: 'reported' | 'dismissed') =>
    post<HomeWorkRequest>(`/api/homes/${homeId}/workrequests/${id}/status`, { status }),
  convert: (homeId: number, id: number, body: HomeWorkRequestConvertInput) =>
    post<HomeWorkRequestConvertResult>(`/api/homes/${homeId}/workrequests/${id}/convert`, body),
  uploadAttachment: (homeId: number, id: number, file: File, mediaType: HomeWorkRequestMediaType) =>
    upload<HomeWorkRequestAttachment>(`/api/homes/${homeId}/workrequests/${id}/attachments`, file, mediaType),
  deleteAttachment: (homeId: number, id: number, attachmentId: number) =>
    del(`/api/homes/${homeId}/workrequests/${id}/attachments/${attachmentId}`),
  attachmentUrl: (homeId: number, id: number, attachmentId: number) =>
    `/api/homes/${homeId}/workrequests/${id}/attachments/${attachmentId}`,
}
