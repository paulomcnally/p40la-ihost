import type { Home, Currency, Service, Bill, Settings, Institution, InstitutionCategory, AnalyzerInfo, Auto, AutoService, Alert, Notification, Child, Salary, PensionCategory, SupportRecord, SalaryPayment, MonthClosing, ChildSupportConfig, Debt, DebtBill } from '../types'

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

function postFormData<T>(path: string, file: File): Promise<T | null> {
  const form = new FormData()
  form.append('file', file)
  return fetch(path, { method: 'POST', body: form }).then(async (res) => {
    if (res.status === 401) {
      window.location.href = '/login'
      return null
    }
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.message || 'Request failed')
    return data
  })
}

function periodParams(filters?: { year?: number; month?: number; id?: number }): string {
  const params = new URLSearchParams()
  if (filters?.year) params.set('year', String(filters.year))
  if (filters?.month) params.set('month', String(filters.month))
  if (filters?.id) params.set('child_id', String(filters.id))
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

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
    reorder: (ids: number[]) => put<{ message: string }>('/api/homes/reorder', { ids }),
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
      currency_thousands_separator: string
      currency_decimal_separator: string
      currency_decimal_digits: number
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
  pensionCategories: {
    list: () => get<PensionCategory[]>('/api/pension-categories'),
    get: (id: number) => get<PensionCategory>(`/api/pension-categories/${id}`),
    create: (body: Partial<PensionCategory>) => post<PensionCategory>('/api/pension-categories', body),
    update: (id: number, body: Partial<PensionCategory>) => put<PensionCategory>(`/api/pension-categories/${id}`, body),
    delete: (id: number) => del(`/api/pension-categories/${id}`),
  },
  pensionRecords: {
    list: (filters?: { year?: number; month?: number; child_id?: number }) =>
      get<SupportRecord[]>(`/api/pension/records${periodParams(filters)}`),
    get: (id: number) => get<SupportRecord>(`/api/pension/records/${id}`),
    create: (body: { child_id: number; pension_category_id: number; year: number; month: number; amount: number; currency?: string; notes?: string }) =>
      post<SupportRecord>('/api/pension/records', body),
    update: (id: number, body: { amount: number; pension_category_id: number; notes?: string | null }) =>
      put<SupportRecord>(`/api/pension/records/${id}`, body),
    markPaid: (id: number, body: { paid_at?: string; payment_method?: string; payment_reference?: string; evidence_notes?: string; original_amount?: number; original_currency?: string; exchange_rate?: number }) =>
      post<SupportRecord>(`/api/pension/records/${id}/mark-paid`, body),
    markPending: (id: number) => post<SupportRecord>(`/api/pension/records/${id}/mark-pending`, {}),
    markRejected: (id: number, reason: string) => post<SupportRecord>(`/api/pension/records/${id}/mark-rejected`, { reason }),
    uploadProof: (id: number, file: File) =>
      postFormData<{ ok: boolean; proof_file_name: string; id: number; status: string }>(`/api/pension/records/${id}/upload-proof`, file),
    proofUrl: (id: number) => `/api/pension/records/${id}/proof`,
  },
  pensionSalaryPayments: {
    list: (filters?: { year?: number; month?: number; salary_id?: number }) =>
      get<SalaryPayment[]>(`/api/pension/salary-payments${filters?.salary_id ? `?year=${filters.year ?? ''}&month=${filters.month ?? ''}&salary_id=${filters.salary_id}` : periodParams(filters)}`),
    get: (id: number) => get<SalaryPayment>(`/api/pension/salary-payments/${id}`),
    markReceived: (id: number, body: { received_at: string; received_amount?: number; notes?: string }) =>
      post<SalaryPayment>(`/api/pension/salary-payments/${id}/mark-received`, body),
    markPending: (id: number) => post<SalaryPayment>(`/api/pension/salary-payments/${id}/mark-pending`, {}),
  },
  pensionClosing: {
    status: (year: number, month: number) => get<MonthClosing>(`/api/pension/closing/${year}/${month}`),
    close: (year: number, month: number) => post<{ ok: boolean; closed_at: string }>(`/api/pension/closing/${year}/${month}`, {}),
    reopen: (year: number, month: number) => del<{ ok: boolean }>(`/api/pension/closing/${year}/${month}`),
  },
  pensionGenerate: {
    generate: (year: number, month: number) =>
      post<{ ok: boolean; created_salary_payments: number; created_support_records: number }>(`/api/pension/generate?year=${year}&month=${month}`, {}),
  },
  pensionConfigs: {
    list: () => get<ChildSupportConfig[]>('/api/pension/configs'),
    get: (id: number) => get<ChildSupportConfig>(`/api/pension/configs/${id}`),
    create: (body: { child_id: number; pension_category_id: number; amount: number; currency: string; is_active?: boolean; auto_generate?: boolean }) =>
      post<ChildSupportConfig>('/api/pension/configs', body),
    update: (id: number, body: { pension_category_id: number; amount: number; currency: string; is_active: boolean; auto_generate: boolean }) =>
      put<ChildSupportConfig>(`/api/pension/configs/${id}`, body),
    delete: (id: number) => del(`/api/pension/configs/${id}`),
  },
  debts: {
    list: () => get<Debt[]>('/api/debts'),
    get: (id: number) => get<Debt>(`/api/debts/${id}`),
    create: (body: Partial<Debt>) => post<Debt>('/api/debts', body),
    update: (id: number, body: Partial<Debt>) => put<Debt>(`/api/debts/${id}`, body),
    delete: (id: number) => del(`/api/debts/${id}`),
    listBills: (debtId: number) => get<DebtBill[]>(`/api/debts/${debtId}/bills`),
    billsByMonth: (year: number, month: number) => get<DebtBill[]>(`/api/debt-bills?year=${year}&month=${month}`),
    payBill: (id: number, body: { paid_at: string; payment_reference?: string }) =>
      put<DebtBill>(`/api/debt-bills/${id}/pay`, body),
  },
}
