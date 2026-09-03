import type { Home, Currency, Service, Bill, Settings, Institution, InstitutionCategory, AnalyzerInfo, Auto, AutoService, Alert, Notification, Child, Salary } from '../types'

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
    pay: (id: number, body: { paid_at: string; drive_url?: string; payment_reference?: string }) =>
      post<Bill>(`/api/bills/${id}/pay`, body),
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
    createBillFromExtracted: (serviceId: number, body: { amount: number; invoice_number: string; year: number; month: number; file_hash?: string }) =>
      post<Bill & { updated?: boolean; duplicate?: boolean }>(`/api/services/${serviceId}/bills/from-extracted`, body),
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
    get: () => get<{
      billing_generation_hour: number
      alert_check_hour: number
      smtp_host: string
      smtp_port: number
      smtp_user: string
      smtp_from_email: string
      smtp_from_name: string
      smtp_configured: boolean
      alert_emails: string
      voicemonkey_enabled: boolean
      voicemonkey_send_alerts: boolean
      voicemonkey_configured: boolean
      email_alerts_enabled: boolean
    }>('/api/system-settings'),
    update: (body: Record<string, unknown>) => put<{ billing_generation_hour: number; smtp_configured: boolean }>('/api/system-settings', body),
    testEmail: () => post<{ message: string; recipients: string }>('/api/system-settings/test-email', {}),
    testVoice: () => post<{ message: string }>('/api/system-settings/test-voice', {}),
    disconnectVoiceMonkey: () => del<{ voicemonkey_enabled: boolean; voicemonkey_send_alerts: boolean; voicemonkey_configured: boolean }>('/api/system-settings/voicemonkey'),
    disconnectSMTP: () => del<{ smtp_configured: boolean }>('/api/system-settings/smtp'),
  },
  alerts: {
    list: () => get<Alert[]>('/api/alerts'),
    update: (key: string, body: { mail_enabled?: boolean; voice_enabled?: boolean }) => put(`/api/alerts/${key}`, body),
  },
  institutions: {
    list: () => get<Institution[]>('/api/institutions'),
    get: (id: number) => get<Institution>(`/api/institutions/${id}`),
    create: (body: { name: string; category_id?: number | null }) => post<Institution>('/api/institutions', body),
    update: (id: number, body: { name: string; category_id?: number | null }) => put<Institution>(`/api/institutions/${id}`, body),
    delete: (id: number) => del(`/api/institutions/${id}`),
    setAnalyzers: (id: number, analyzer_ids: string[]) => put(`/api/institutions/${id}/analyzers`, { analyzer_ids }),
    getAnalyzers: (id: number) => get<Array<{ id: number; institution_id: number; analyzer_id: string; created_at: string }>>(`/api/institutions/${id}/analyzers`),
  },
  institutionCategories: {
    list: () => get<InstitutionCategory[]>('/api/institution-categories'),
    get: (id: number) => get<InstitutionCategory>(`/api/institution-categories/${id}`),
    create: (body: { key: string; name: string; description: string; icon_key: string }) => post<InstitutionCategory>('/api/institution-categories', body),
    update: (id: number, body: { name: string; description: string; icon_key: string }) => put<InstitutionCategory>(`/api/institution-categories/${id}`, body),
    delete: (id: number) => del(`/api/institution-categories/${id}`),
  },
  analyzers: {
    list: () => get<AnalyzerInfo[]>('/api/analyzers'),
  },
  autos: {
    list: () => get<Auto[]>('/api/autos'),
    get: (id: number) => get<Auto>(`/api/autos/${id}`),
    create: (body: Partial<Auto>) => post<Auto>('/api/autos', body),
    update: (id: number, body: Partial<Auto>) => put<Auto>(`/api/autos/${id}`, body),
    delete: (id: number) => del(`/api/autos/${id}`),
    listServices: (id: number) => get<AutoService[]>(`/api/autos/${id}/services`),
    addService: (id: number, body: { service_id: number; coverage_type: string; policy_number: string; certificate?: string; insurer_number: string }) => post(`/api/autos/${id}/services`, body),
    removeService: (id: number, serviceId: number) => del(`/api/autos/${id}/services/${serviceId}`),
    availableServices: (id: number) => get<Service[]>(`/api/autos/${id}/available-services`),
  },
  notifications: {
    list: () => get<Notification[]>('/api/notifications'),
    get: (id: number) => get<Notification>(`/api/notifications/${id}`),
    create: (body: Partial<Notification>) => post<Notification>('/api/notifications', body),
    update: (id: number, body: Partial<Notification>) => put<Notification>(`/api/notifications/${id}`, body),
    delete: (id: number) => del(`/api/notifications/${id}`),
  },
  children: {
    list: () => get<Child[]>('/api/children'),
    get: (id: number) => get<Child>(`/api/children/${id}`),
    create: (body: Partial<Child>) => post<Child>('/api/children', body),
    update: (id: number, body: Partial<Child>) => put<Child>(`/api/children/${id}`, body),
    delete: (id: number) => del(`/api/children/${id}`),
  },
  salaries: {
    list: () => get<Salary[]>('/api/salaries'),
    get: (id: number) => get<Salary>(`/api/salaries/${id}`),
    create: (body: Partial<Salary>) => post<Salary>('/api/salaries', body),
    update: (id: number, body: Partial<Salary>) => put<Salary>(`/api/salaries/${id}`, body),
    delete: (id: number) => del(`/api/salaries/${id}`),
  },
}
