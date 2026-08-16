import type { Home, Currency, Service, Bill, Settings, Institution, AnalyzerInfo } from '../types'

async function request<T>(path: string, options: RequestInit = {}): Promise<T | null> {
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

function get<T>(path: string) { return request<T>(path, { method: 'GET' }) }
function post<T>(path: string, body: unknown) { return request<T>(path, { method: 'POST', body: JSON.stringify(body) }) }
function put<T>(path: string, body: unknown) { return request<T>(path, { method: 'PUT', body: JSON.stringify(body) }) }
function del<T>(path: string) { return request<T>(path, { method: 'DELETE' }) }

export const api = {
  settings: {
    get: () => get<Settings>('/api/settings'),
    setLanguage: (language: string) => post('/api/settings/language', { language }),
  },
  currencies: {
    list: () => get<Currency[]>('/api/currencies'),
    create: (body: Partial<Currency>) => post<Currency>('/api/currencies', body),
    update: (id: number, body: Partial<Currency>) => put<Currency>(`/api/currencies/${id}`, body),
    delete: (id: number) => del(`/api/currencies/${id}`),
  },
  homes: {
    list: () => get<Home[]>('/api/homes'),
    get: (id: number) => get<Home>(`/api/homes/${id}`),
    create: (body: Partial<Home>) => post<Home>('/api/homes', body),
    update: (id: number, body: Partial<Home>) => put<Home>(`/api/homes/${id}`, body),
    delete: (id: number) => del(`/api/homes/${id}`),
  },
  services: {
    list: (homeId?: number) => get<Service[]>(homeId ? `/api/services?home_id=${homeId}` : '/api/services'),
    get: (id: number) => get<Service>(`/api/services/${id}`),
    create: (body: Partial<Service>) => post<Service>('/api/services', body),
    update: (id: number, body: Partial<Service>) => put<Service>(`/api/services/${id}`, body),
    delete: (id: number) => del(`/api/services/${id}`),
    getAnalyzerOptions: (id: number) => get<Array<{id: number, institution_id: number, analyzer_id: string, analyzer_name: string}>>(`/api/services/${id}/analyzer-options`),
  },
  bills: {
    list: (serviceId: number) => get<Bill[]>(`/api/services/${serviceId}/bills`),
    get: (id: number) => get<Bill>(`/api/bills/${id}`),
    create: (body: Partial<Bill>) => post<Bill>('/api/bills', body),
    update: (id: number, body: Partial<Bill>) => put<Bill>(`/api/bills/${id}`, body),
    delete: (id: number) => del(`/api/bills/${id}`),
    uploadAndAnalyze: (serviceId: number, file: File) => {
      const form = new FormData()
      form.append('file', file)
      return fetch(`/api/services/${serviceId}/bills/upload`, { method: 'POST', body: form })
        .then(async (res) => {
          const data = await res.json().catch(() => ({}))
          if (res.status === 401) {
            window.location.href = '/login'
            return null
          }
          if (!res.ok) throw new Error(data.message || 'Error al analizar el documento')
          return data
        })
    },
    createBillFromExtracted: (serviceId: number, body: { amount: number; invoice_number: string; year: number; month: number }) =>
      post<Bill & { updated?: boolean }>(`/api/services/${serviceId}/bills/from-extracted`, body),
  },
  auth: {
    login: (email: string, password: string, remember: boolean) =>
      request('/api/login', { method: 'POST', body: JSON.stringify({ email, password, remember }) }),
    logout: () => fetch('/api/logout', { method: 'POST' }),
    setup: (email: string, password: string, password_confirm: string) =>
      request('/api/setup', { method: 'POST', body: JSON.stringify({ email, password, password_confirm }) }),
    setupStatus: () => get<{ setup: boolean }>('/api/setup-status'),
    me: () => get('/api/me'),
  },
  systemSettings: {
    get: () => get<{ billing_generation_hour: number }>('/api/system-settings'),
    update: (body: { billing_generation_hour: number }) => put<{ billing_generation_hour: number }>('/api/system-settings', body),
  },
  institutions: {
    list: () => get<Institution[]>('/api/institutions'),
    get: (id: number) => get<Institution>(`/api/institutions/${id}`),
    create: (body: { name: string }) => post<Institution>('/api/institutions', body),
    update: (id: number, body: { name: string }) => put<Institution>(`/api/institutions/${id}`, body),
    delete: (id: number) => del(`/api/institutions/${id}`),
    setAnalyzers: (id: number, analyzer_ids: string[]) => put(`/api/institutions/${id}/analyzers`, { analyzer_ids }),
    getAnalyzers: (id: number) => get<Array<{ id: number; institution_id: number; analyzer_id: string; created_at: string }>>(`/api/institutions/${id}/analyzers`),
  },
  analyzers: {
    list: () => get<AnalyzerInfo[]>('/api/analyzers'),
  },
}
