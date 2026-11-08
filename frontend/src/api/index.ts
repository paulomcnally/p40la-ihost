import type { Home, Currency, Service, ServiceCycle, Bill, BillHistoryEntry, Settings, Institution, InstitutionCategory, AnalyzerInfo, Auto, AutoService, ServiceAuto, AutoAttachment, AutoMaintenance, AutoMaintenanceInput, AutoMaintenanceItem, AutoMaintenanceItemInput, AutoMaintenanceList, AutoInspection, AutoInspectionList, AutoInspectionAlert, Alert, Notification, Child, Salary, PensionCategory, PensionCategoryKeyword, PensionImportPreview, PensionImportBatch, PensionImportConfirmRow, PensionImportConfirmResult, SupportRecord, SalaryPayment, MonthClosing, ChildSupportConfig, Debt, DebtBill, DebtHistoryResponse, DebtInformationTemplate, DebtInformationField, DebtPaymentInformation, DebtInformationToken, DebtInformationValue, CategoryGroup, BudgetCategory, Account, Assignment, BudgetTransaction, BudgetMonthView, RecurringRule, SuggestedAssignment, Transfer, AutomationAccount, AutomationAccountOption, AutomationPlugin, AutomationBill, AutomationWebhookLog, AutomationCredentialField, AutomationBillItem, AutomationAccountInput, ServiceBillDetail, ServiceBillSummaryResponse, ServiceProjectionResponse, DocumentRecognition, ConfirmRecognitionInput, AssociationResult } from '../types'

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

function multipartRequest<T>(path: string, form: FormData, method: 'POST' | 'PUT' = 'POST'): Promise<T | null> {
  return fetch(path, { method, body: form }).then(async (res) => {
    if (res.status === 401) {
      window.location.href = '/login'
      return null
    }
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      const err = new Error(data.message || 'Request failed') as Error & { status?: number; validation?: unknown; code?: string }
      err.status = res.status
      err.validation = data.validation
      err.code = data.error
      throw err
    }
    return data
  })
}

function postMultipart<T>(path: string, form: FormData): Promise<T | null> {
  return multipartRequest<T>(path, form, 'POST')
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
    regenerateWebhook: (id: number) => post<{ webhook_uuid: string; webhook_url: string }>(`/api/services/${id}/webhook/regenerate`, {}),
    listAutos: (id: number) => get<ServiceAuto[]>(`/api/services/${id}/autos`),
    listCycles: (id: number) => get<ServiceCycle[]>(`/api/services/${id}/cycles`),
    renew: (id: number, body: { start_date: string; end_date: string; suggested_amount?: number }) =>
      post<{ service: Service; cycle: ServiceCycle }>(`/api/services/${id}/renew`, body),
    sync: (id: number) => post<{ delivered: number; failed: number }>(`/api/services/${id}/sync`, {}),
  },
  contracts: {
    upload: (owner: 'debts' | 'services', id: number, file: File) =>
      postFormData<{ has_contract: boolean }>(`/api/${owner}/${id}/contract`, file),
    remove: (owner: 'debts' | 'services', id: number) =>
      del(`/api/${owner}/${id}/contract`),
    url: (owner: 'debts' | 'services', id: number) =>
      `/api/${owner}/${id}/contract`,
  },
  webhooks: {
    getApiKey: () => get<{ api_key: string }>('/api/webhook/key'),
    regenerateApiKey: () => post<{ api_key: string }>('/api/webhook/key/regenerate', {}),
  },
  bills: {
    list: (serviceId: number) => get<Bill[]>(`/api/services/${serviceId}/bills`),
    get: (id: number) => get<Bill>(`/api/bills/${id}`),
    history: (id: number) => get<BillHistoryEntry[]>(`/api/bills/${id}/history`),
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
  serviceBills: {
    byMonth: (year: number, month: number, homeId?: number | null) => {
      const params = new URLSearchParams({ year: String(year), month: String(month) })
      if (homeId) params.set('home_id', String(homeId))
      return get<ServiceBillDetail[]>(`/api/service-bills?${params.toString()}`)
    },
    summary: (from: string, to: string, homeId?: number | null) => {
      const params = new URLSearchParams({ from, to })
      if (homeId) params.set('home_id', String(homeId))
      return get<ServiceBillSummaryResponse>(`/api/service-bills/summary?${params.toString()}`)
    },
    projection: (from: string, months: number, homeId?: number | null) => {
      const params = new URLSearchParams({ from, months: String(months) })
      if (homeId) params.set('home_id', String(homeId))
      return get<ServiceProjectionResponse>(`/api/service-bills/projection?${params.toString()}`)
    },
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
      timezone: string
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
      webhook_enabled: boolean
      webhook_base_url: string
      currency_thousands_separator: string
      currency_decimal_separator: string
      currency_decimal_digits: number
      email_color_primary: string
      email_color_background: string
      email_color_card: string
      email_color_text: string
      email_color_muted: string
      email_color_border: string
      telegram_bot_enabled: boolean
      telegram_bot_configured: boolean
      telegram_bot_chat_ids: string[]
      telegram_bot_chat_ids_count: number
      telegram_bot_separator_length: number
      telegram_bot_show_months: number
    }>('/api/system-settings'),
    update: (body: Record<string, unknown>) => put<{ billing_generation_hour: number; smtp_configured: boolean }>('/api/system-settings', body),
    testEmail: () => post<{ message: string; recipients: string }>('/api/system-settings/test-email', {}),
    testVoice: () => post<{ message: string }>('/api/system-settings/test-voice', {}),
    disconnectVoiceMonkey: () => del<{ voicemonkey_enabled: boolean; voicemonkey_send_alerts: boolean; voicemonkey_configured: boolean }>('/api/system-settings/voicemonkey'),
    disconnectSMTP: () => del<{ smtp_configured: boolean }>('/api/system-settings/smtp'),
    resetEmailPalette: () => post<{
      email_color_primary: string
      email_color_background: string
      email_color_card: string
      email_color_text: string
      email_color_muted: string
      email_color_border: string
      message: string
    }>('/api/system-settings/email-palette/reset', {}),
    previewEmail: (palette: Record<string, string>) => post<{ html: string }>('/api/system-settings/preview-email', palette),
  },
  alerts: {
    list: () => get<Alert[]>('/api/alerts'),
    update: (key: string, body: { mail_enabled?: boolean; voice_enabled?: boolean; telegram_enabled?: boolean }) => put(`/api/alerts/${key}`, body),
    sendNow: () => post<{
      message: string
      results: Array<{
        key: string
        title: string
        sent_channels: string[]
        items: number
        detail: string
      }>
    }>('/api/alerts/send-now', {}),
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
    updateService: (id: number, serviceId: number, body: { coverage_type: string; policy_number: string; certificate?: string; insurer_number: string }) => put(`/api/autos/${id}/services/${serviceId}`, body),
    removeService: (id: number, serviceId: number) => del(`/api/autos/${id}/services/${serviceId}`),
    availableServices: (id: number) => get<Service[]>(`/api/autos/${id}/available-services`),
    listMaintenances: (id: number) => get<AutoMaintenanceList>(`/api/autos/${id}/maintenances`),
    getMaintenance: (id: number, maintenanceId: number) => get<AutoMaintenance>(`/api/autos/${id}/maintenances/${maintenanceId}`),
    createMaintenance: (id: number, body: AutoMaintenanceInput) => post<AutoMaintenance>(`/api/autos/${id}/maintenances`, body),
    updateMaintenance: (id: number, maintenanceId: number, body: AutoMaintenanceInput) => put<AutoMaintenance>(`/api/autos/${id}/maintenances/${maintenanceId}`, body),
    deleteMaintenance: (id: number, maintenanceId: number) => del(`/api/autos/${id}/maintenances/${maintenanceId}`),
    createMaintenanceItem: (id: number, maintenanceId: number, body: AutoMaintenanceItemInput) =>
      post<AutoMaintenanceItem>(`/api/autos/${id}/maintenances/${maintenanceId}/items`, body),
    updateMaintenanceItem: (id: number, maintenanceId: number, itemId: number, body: AutoMaintenanceItemInput) =>
      put(`/api/autos/${id}/maintenances/${maintenanceId}/items/${itemId}`, body),
    deleteMaintenanceItem: (id: number, maintenanceId: number, itemId: number) =>
      del(`/api/autos/${id}/maintenances/${maintenanceId}/items/${itemId}`),
    uploadMaintenanceAttachment: (id: number, maintenanceId: number, file: File) =>
      postFormData<AutoAttachment>(`/api/autos/${id}/maintenances/${maintenanceId}/attachments`, file),
    maintenanceAttachmentUrl: (id: number, maintenanceId: number, attachmentId: number, download = false) =>
      `/api/autos/${id}/maintenances/${maintenanceId}/attachments/${attachmentId}${download ? '?download=1' : ''}`,
    deleteMaintenanceAttachment: (id: number, maintenanceId: number, attachmentId: number) =>
      del(`/api/autos/${id}/maintenances/${maintenanceId}/attachments/${attachmentId}`),
    listInspections: (id: number) => get<AutoInspectionList>(`/api/autos/${id}/inspections`),
    getInspection: (id: number, inspectionId: number) => get<AutoInspection>(`/api/autos/${id}/inspections/${inspectionId}`),
    createInspection: (id: number, body: Partial<AutoInspection>) => post<AutoInspection>(`/api/autos/${id}/inspections`, body),
    updateInspection: (id: number, inspectionId: number, body: Partial<AutoInspection>) => put<AutoInspection>(`/api/autos/${id}/inspections/${inspectionId}`, body),
    deleteInspection: (id: number, inspectionId: number) => del(`/api/autos/${id}/inspections/${inspectionId}`),
    uploadInspectionAttachment: (id: number, inspectionId: number, file: File) =>
      postFormData<AutoAttachment>(`/api/autos/${id}/inspections/${inspectionId}/attachments`, file),
    inspectionAttachmentUrl: (id: number, inspectionId: number, attachmentId: number, download = false) =>
      `/api/autos/${id}/inspections/${inspectionId}/attachments/${attachmentId}${download ? '?download=1' : ''}`,
    deleteInspectionAttachment: (id: number, inspectionId: number, attachmentId: number) =>
      del(`/api/autos/${id}/inspections/${inspectionId}/attachments/${attachmentId}`),
    inspectionAlerts: () => get<AutoInspectionAlert[]>('/api/autos/inspection-alerts'),
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
  pensionImport: {
    previewFile: (file: File, opts?: { year?: number; month?: number; child_id?: number }) => {
      const form = new FormData()
      form.append('file', file)
      if (opts?.year) form.append('year', String(opts.year))
      if (opts?.month) form.append('month', String(opts.month))
      if (opts?.child_id) form.append('child_id', String(opts.child_id))
      return postMultipart<PensionImportPreview>('/api/pension/import/preview', form)
    },
    previewText: (body: { text: string; year?: number; month?: number; child_id?: number }) =>
      post<PensionImportPreview>('/api/pension/import/preview', body),
    confirm: (body: { batch_id: number; child_id: number; rows: PensionImportConfirmRow[] }) =>
      post<PensionImportConfirmResult>('/api/pension/import/confirm', body),
    batches: () => get<PensionImportBatch[]>('/api/pension/import/batches'),
    templateUrl: '/api/pension/import/template.csv',
    exportUrl: (year: number, month: number) => `/api/pension/import/export?year=${year}&month=${month}`,
    keywords: {
      list: () => get<PensionCategoryKeyword[]>('/api/pension/keywords'),
      create: (body: { pension_category_id: number; keyword: string }) =>
        post<PensionCategoryKeyword>('/api/pension/keywords', body),
      delete: (id: number) => del(`/api/pension/keywords/${id}`),
    },
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
    history: (months = 12) => get<DebtHistoryResponse>(`/api/debts/history?months=${months}`),
    get: (id: number) => get<Debt>(`/api/debts/${id}`),
    create: (body: Partial<Debt>) => post<Debt>('/api/debts', body),
    update: (id: number, body: Partial<Debt>) => put<Debt>(`/api/debts/${id}`, body),
    delete: (id: number) => del(`/api/debts/${id}`),
    listBills: (debtId: number) => get<DebtBill[]>(`/api/debts/${debtId}/bills`),
    billsByMonth: (year: number, month: number) => get<DebtBill[]>(`/api/debt-bills?year=${year}&month=${month}`),
    payBill: (id: number, body: { paid_at: string; payment_reference?: string }, invoice?: File, receipt?: File) => {
      const form = new FormData()
      form.append('paid_at', body.paid_at)
      if (body.payment_reference) form.append('payment_reference', body.payment_reference)
      if (invoice) form.append('invoice', invoice)
      if (receipt) form.append('receipt', receipt)
      return multipartRequest<DebtBill>(`/api/debt-bills/${id}/pay`, form, 'PUT')
    },
    invoiceUrl: (id: number) => `/api/debt-bills/${id}/invoice`,
    receiptUrl: (id: number) => `/api/debt-bills/${id}/receipt`,
    uploadInvoice: (id: number, file: File) => {
      const form = new FormData()
      form.append('file', file)
      return multipartRequest<DebtBill>(`/api/debt-bills/${id}/invoice`, form, 'POST')
    },
    uploadReceipt: (id: number, file: File) => {
      const form = new FormData()
      form.append('file', file)
      return multipartRequest<DebtBill>(`/api/debt-bills/${id}/receipt`, form, 'POST')
    },
    removeInvoice: (id: number) => del<DebtBill>(`/api/debt-bills/${id}/invoice`),
    removeReceipt: (id: number) => del<DebtBill>(`/api/debt-bills/${id}/receipt`),
  },
  debtInformationTemplates: {
    list: () => get<DebtInformationTemplate[]>('/api/debt-information-templates'),
    get: (id: number) => get<DebtInformationTemplate>(`/api/debt-information-templates/${id}`),
    create: (body: { name: string; description?: string; fields: DebtInformationField[] }) =>
      post<DebtInformationTemplate>('/api/debt-information-templates', body),
    update: (id: number, body: { name: string; description?: string; fields: DebtInformationField[] }) =>
      put<DebtInformationTemplate>(`/api/debt-information-templates/${id}`, body),
    delete: (id: number) => del(`/api/debt-information-templates/${id}`),
    tokens: () => get<DebtInformationToken[]>('/api/debt-information-templates/tokens'),
  },
  debtPaymentInformation: {
    get: (debtId: number) => get<DebtPaymentInformation>(`/api/debts/${debtId}/payment-information`),
    save: (debtId: number, body: { template_id: number; values: Record<string, DebtInformationValue> }) =>
      put<DebtPaymentInformation>(`/api/debts/${debtId}/payment-information`, body),
    clear: (debtId: number) => del(`/api/debts/${debtId}/payment-information`),
  },
  documents: {
    recognize: (file: File) => {
      const form = new FormData()
      form.append('file', file)
      return multipartRequest<DocumentRecognition>('/api/documents/recognize', form, 'POST')
    },
    confirm: (recognitionId: number, body: ConfirmRecognitionInput) =>
      post<AssociationResult>(`/api/documents/recognitions/${recognitionId}/confirm`, body),
  },
  budget: {
    monthView: (year: number, month: number) => get<BudgetMonthView>(`/api/budget/months/${year}/${month}`),
    assign: (year: number, month: number, categoryId: number, body: { amount: number; currency_id: number; make_recurring?: boolean }) =>
      put<Assignment>(`/api/budget/months/${year}/${month}/assignments/${categoryId}`, body),
    toggleRecurring: (id: number, active: boolean) =>
      put<RecurringRule>(`/api/budget/recurring-rules/${id}`, { active }),
    deleteRecurringRule: (id: number) => del(`/api/budget/recurring-rules/${id}`),
    categoryGroups: {
      list: () => get<CategoryGroup[]>('/api/budget/category-groups'),
      create: (body: { name: string; icon: string }) => post<{ id: number; name: string; icon: string }>('/api/budget/category-groups', body),
      update: (id: number, body: { name: string; icon: string }) => put(`/api/budget/category-groups/${id}`, body),
      delete: (id: number) => del(`/api/budget/category-groups/${id}`),
    },
    categories: {
      create: (body: { category_group_id: number; name: string; icon: string; target_amount?: number | null; service_ids?: number[]; account_id?: number | null }) =>
        post<BudgetCategory>('/api/budget/categories', body),
      update: (id: number, body: { category_group_id: number; name: string; icon: string; target_amount?: number | null; service_ids?: number[]; account_id?: number | null }) =>
        put<BudgetCategory>(`/api/budget/categories/${id}`, body),
      delete: (id: number) => del(`/api/budget/categories/${id}`),
      suggestedAssignment: (id: number) => get<SuggestedAssignment>(`/api/budget/categories/${id}/suggested-assignment`),
    },
    accounts: {
      list: () => get<Account[]>('/api/budget/accounts'),
      create: (body: { name: string; type: string; currency_id: number; starting_balance: number }) =>
        post<Account>('/api/budget/accounts', body),
      update: (id: number, body: { name: string; type: string; currency_id: number; starting_balance: number }) =>
        put<Account>(`/api/budget/accounts/${id}`, body),
      delete: (id: number) => del(`/api/budget/accounts/${id}`),
    },
    transactions: {
      list: (year: number, month: number) => get<BudgetTransaction[]>(`/api/budget/transactions?year=${year}&month=${month}`),
      listByCategory: (categoryId: number, year: number, month: number) =>
        get<BudgetTransaction[]>(`/api/budget/categories/${categoryId}/transactions?year=${year}&month=${month}`),
      create: (body: Partial<BudgetTransaction>) => post<BudgetTransaction>('/api/budget/transactions', body),
      update: (id: number, body: Partial<BudgetTransaction>) => put<BudgetTransaction>(`/api/budget/transactions/${id}`, body),
      delete: (id: number) => del(`/api/budget/transactions/${id}`),
    },
    transfers: {
      list: (year?: number, month?: number) => {
        const q = year && month ? `?year=${year}&month=${month}` : ''
        return get<Transfer[]>(`/api/budget/transfers${q}`)
      },
      create: (body: Partial<Transfer>) => post<Transfer>('/api/budget/transfers', body),
      update: (id: number, body: Partial<Transfer>) => put<Transfer>(`/api/budget/transfers/${id}`, body),
      delete: (id: number) => del(`/api/budget/transfers/${id}`),
    },
  },
  automation: {
    accounts: () => get<AutomationAccount[]>('/api/automation/accounts'),
    accountOptions: () => get<AutomationAccountOption[]>('/api/automation/accounts?for=service'),
    get: (id: number) => get<AutomationAccount>(`/api/automation/accounts/${id}`),
    create: (body: AutomationAccountInput) => post<AutomationAccount>('/api/automation/accounts', body),
    update: (id: number, body: AutomationAccountInput) => put<AutomationAccount>(`/api/automation/accounts/${id}`, body),
    delete: (id: number) => del(`/api/automation/accounts/${id}`),
    revealCredentials: (id: number, password: string) =>
      post<{ credentials: string }>(`/api/automation/accounts/${id}/credentials:reveal`, { password }),
    plugins: () => get<AutomationPlugin[]>('/api/automation/plugins'),
    pluginSchema: (name: string) => get<AutomationCredentialField[]>(`/api/automation/plugins/${encodeURIComponent(name)}/schema`),
    bills: (id: number) => get<AutomationBill[]>(`/api/automation/accounts/${id}/bills`),
    fetchBills: (id: number) => post<{ bills: AutomationBillItem[] }>(`/api/automation/accounts/${id}/bills:fetch`, {}),
    logs: (id: number, limit = 20, offset = 0) =>
      get<AutomationWebhookLog[]>(`/api/automation/accounts/${id}/webhook/logs?limit=${limit}&offset=${offset}`),
  },
}
