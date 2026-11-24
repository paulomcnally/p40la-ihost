import type {
  Charge, HomeFinances, HomeRentalStatus, Invoice, Lease, LeaseDocument, LeaseParty, Payment, PaymentProof,
  PortfolioSummary, Receipt, Reminder, RentalBalance, RentalPerson, RentalSettings, Reservation,
} from '../types/rental'

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

function upload<T>(path: string, file: File, fields: Record<string, string> = {}): Promise<T | null> {
  const form = new FormData()
  form.append('file', file)
  Object.entries(fields).forEach(([key, value]) => form.append(key, value))
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

export const rentalApi = {
  people: {
    list: (query = '') => get<RentalPerson[]>(`/api/rental/people${query ? `?q=${encodeURIComponent(query)}` : ''}`),
    get: (id: number) => get<RentalPerson>(`/api/rental/people/${id}`),
    create: (body: Partial<RentalPerson>) => post<RentalPerson>('/api/rental/people', body),
    update: (id: number, body: Partial<RentalPerson>) => put<RentalPerson>(`/api/rental/people/${id}`, body),
    delete: (id: number) => del(`/api/rental/people/${id}`),
  },
  homes: {
    status: (id: number) => get<HomeRentalStatus>(`/api/rental/homes/${id}`),
  },
  portfolio: {
    summary: (year: number, month: number) =>
      get<PortfolioSummary>(`/api/rental/portfolio/summary?year=${year}&month=${month}`),
    homeFinances: (homeId: number, year: number, month: number) =>
      get<HomeFinances>(`/api/rental/portfolio/homes/${homeId}/finances?year=${year}&month=${month}`),
  },
  leases: {
    list: (params: { status?: string; homeId?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.status) q.set('status', params.status)
      if (params.homeId) q.set('home_id', String(params.homeId))
      const qs = q.toString()
      return get<Lease[]>(`/api/rental/leases${qs ? `?${qs}` : ''}`)
    },
    get: (id: number) => get<Lease>(`/api/rental/leases/${id}`),
    create: (body: Partial<Lease>) => post<Lease>('/api/rental/leases', body),
    update: (id: number, body: Partial<Lease>) => put<Lease>(`/api/rental/leases/${id}`, body),
    delete: (id: number) => del(`/api/rental/leases/${id}`),
    activate: (id: number) => post<Lease>(`/api/rental/leases/${id}/activate`),
    terminate: (id: number, body: { reason: string; end_date?: string }) =>
      post<Lease>(`/api/rental/leases/${id}/terminate`, body),
    balance: (id: number) => get<RentalBalance>(`/api/rental/leases/${id}/balance`),
    addParty: (id: number, body: Partial<LeaseParty>) => post<LeaseParty>(`/api/rental/leases/${id}/parties`, body),
    deleteParty: (id: number, partyId: number) => del(`/api/rental/leases/${id}/parties/${partyId}`),
    charges: (id: number) => get<Charge[]>(`/api/rental/leases/${id}/charges`),
    generateCharges: (id: number) => post<{ created: number }>(`/api/rental/leases/${id}/charges`),
    documents: (id: number) => get<LeaseDocument[]>(`/api/rental/leases/${id}/documents`),
    uploadDocument: (id: number, file: File, docType: string) =>
      upload<LeaseDocument>(`/api/rental/leases/${id}/documents`, file, { doc_type: docType }),
    deleteDocument: (id: number, docId: number) => del(`/api/rental/leases/${id}/documents/${docId}`),
    documentUrl: (id: number, docId: number, download = false) =>
      `/api/rental/leases/${id}/documents/${docId}${download ? '?download=1' : ''}`,
  },
  reservations: {
    list: (params: { homeId?: number; status?: string } = {}) => {
      const q = new URLSearchParams()
      if (params.homeId) q.set('home_id', String(params.homeId))
      if (params.status) q.set('status', params.status)
      const qs = q.toString()
      return get<Reservation[]>(`/api/rental/reservations${qs ? `?${qs}` : ''}`)
    },
    get: (id: number) => get<Reservation>(`/api/rental/reservations/${id}`),
    create: (body: Partial<Reservation>) => post<Reservation>('/api/rental/reservations', body),
    update: (id: number, body: Partial<Reservation>) => put<Reservation>(`/api/rental/reservations/${id}`, body),
    convert: (id: number, body: { term_months?: number; payment_day?: number; deposit_amount?: number; commission_type?: 'percent' | 'fixed'; commission_percent?: number; commission_amount?: number }) =>
      post<Lease>(`/api/rental/reservations/${id}/convert`, body),
    cancel: (id: number, reason: string) => post(`/api/rental/reservations/${id}/cancel`, { reason }),
    forfeit: (id: number, reason: string) => post(`/api/rental/reservations/${id}/forfeit`, { reason }),
    pdfUrl: (id: number, download = false) =>
      `/api/rental/reservations/${id}/pdf${download ? '?download=1' : ''}`,
  },
  invoices: {
    list: (params: { leaseId?: number; status?: string } = {}) => {
      const q = new URLSearchParams()
      if (params.leaseId) q.set('lease_id', String(params.leaseId))
      if (params.status) q.set('status', params.status)
      const qs = q.toString()
      return get<Invoice[]>(`/api/rental/invoices${qs ? `?${qs}` : ''}`)
    },
    get: (id: number) => get<Invoice>(`/api/rental/invoices/${id}`),
    issue: (body: { lease_id: number; charge_ids: number[]; due_date?: string; notes?: string }) =>
      post<Invoice>('/api/rental/invoices', body),
    send: (id: number) => post<{ recipients: number }>(`/api/rental/invoices/${id}/send`),
    void: (id: number, reason: string) => post<Invoice>(`/api/rental/invoices/${id}/void`, { reason }),
    pdfUrl: (id: number, download = false) =>
      `/api/rental/invoices/${id}/pdf${download ? '?download=1' : ''}`,
  },
  payments: {
    list: (params: { leaseId?: number; invoiceId?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.leaseId) q.set('lease_id', String(params.leaseId))
      if (params.invoiceId) q.set('invoice_id', String(params.invoiceId))
      const qs = q.toString()
      return get<Payment[]>(`/api/rental/payments${qs ? `?${qs}` : ''}`)
    },
    create: (body: {
      lease_id: number
      invoice_id?: number | null
      paid_at?: string
      amount: number
      currency: string
      method: string
      reference?: string
      notes?: string
      with_receipt?: boolean
    }) => post<Payment>('/api/rental/payments', body),
    receipt: (paymentId: number) => get<Receipt>(`/api/rental/payments/${paymentId}/receipt`),
    receiptUrl: (paymentId: number) => `/api/rental/payments/${paymentId}/receipt.pdf`,
    uploadProof: (paymentId: number, file: File) => upload<PaymentProof>(`/api/rental/payments/${paymentId}/proofs`, file),
    deleteProof: (paymentId: number, proofId: number) => del(`/api/rental/payments/${paymentId}/proofs/${proofId}`),
    proofUrl: (paymentId: number, proofId: number) =>
      `/api/rental/payments/${paymentId}/proofs/${proofId}`,
  },
  reminders: {
    list: (params: { homeId?: number; leaseId?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.homeId) q.set('home_id', String(params.homeId))
      if (params.leaseId) q.set('lease_id', String(params.leaseId))
      const qs = q.toString()
      return get<Reminder[]>(`/api/rental/reminders${qs ? `?${qs}` : ''}`)
    },
    create: (body: Partial<Reminder>) => post<Reminder>('/api/rental/reminders', body),
    update: (id: number, body: Partial<Reminder>) => put<Reminder>(`/api/rental/reminders/${id}`, body),
    delete: (id: number) => del(`/api/rental/reminders/${id}`),
    run: (id: number) => post<{ recipients: number }>(`/api/rental/reminders/${id}/run`),
  },
  settings: {
    get: () => get<RentalSettings>('/api/rental/settings'),
    update: (body: Partial<RentalSettings>) => put<RentalSettings>('/api/rental/settings', body),
  },
}
