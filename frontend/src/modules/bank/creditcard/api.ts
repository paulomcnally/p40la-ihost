import { postFormData, request } from '../../../api'
import type {
  CardStatement,
  CardTransaction,
  CreditCard,
  ParsedStatement,
} from './types'

function del(path: string) {
  return request<null>(path, { method: 'DELETE' })
}

export const creditCardsApi = {
  listByBank: (bankId: number) => request<CreditCard[]>(`/api/banks/${bankId}/credit-cards`),
  create: (bankId: number, body: Partial<CreditCard>) =>
    request<CreditCard>(`/api/banks/${bankId}/credit-cards`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  get: (id: number) => request<CreditCard>(`/api/credit-cards/${id}`),
  update: (id: number, body: Partial<CreditCard>) =>
    request<CreditCard>(`/api/credit-cards/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  remove: (id: number) => del(`/api/credit-cards/${id}`),

  statements: (cardId: number) => request<CardStatement[]>(`/api/credit-cards/${cardId}/statements`),
  createStatement: (cardId: number, body: Partial<CardStatement>) =>
    request<CardStatement>(`/api/credit-cards/${cardId}/statements`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  getStatement: (id: number) => request<CardStatement>(`/api/credit-card-statements/${id}`),
  updateStatement: (id: number, body: Partial<CardStatement>) =>
    request<CardStatement>(`/api/credit-card-statements/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  removeStatement: (id: number) => del(`/api/credit-card-statements/${id}`),

  transactions: (statementId: number, params?: { type?: string; q?: string }) => {
    const query = new URLSearchParams()
    if (params?.type) query.set('type', params.type)
    if (params?.q) query.set('q', params.q)
    const suffix = query.toString() ? `?${query.toString()}` : ''
    return request<CardTransaction[]>(`/api/credit-card-statements/${statementId}/transactions${suffix}`)
  },
  createTransaction: (statementId: number, body: Partial<CardTransaction>) =>
    request<CardTransaction>(`/api/credit-card-statements/${statementId}/transactions`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  updateTransaction: (id: number, body: Partial<CardTransaction>) =>
    request<CardTransaction>(`/api/credit-card-transactions/${id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  removeTransaction: (id: number) => del(`/api/credit-card-transactions/${id}`),

  previewImport: (cardId: number, file: File) =>
    postFormData<ParsedStatement>(`/api/credit-cards/${cardId}/statements/import/preview`, file),
  confirmImport: (cardId: number, parsed: ParsedStatement) =>
    request<CardStatement>(`/api/credit-cards/${cardId}/statements/import`, {
      method: 'POST',
      body: JSON.stringify(parsed),
    }),
}
