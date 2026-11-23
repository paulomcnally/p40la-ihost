export type LeaseStatus = 'draft' | 'active' | 'ended' | 'terminated' | 'cancelled'
export type ReservationStatus = 'active' | 'converted' | 'expired' | 'cancelled' | 'forfeited'
export type ListingStatus = 'available' | 'reserved' | 'rented' | 'unlisted'
export type PartyRole = 'tenant' | 'guarantor' | 'owner' | 'agency_contact' | 'agent'
export type ChargeKind = 'rent' | 'maintenance' | 'deposit' | 'late_fee' | 'utility' | 'other'
export type ChargeStatus = 'pending' | 'invoiced' | 'partially_paid' | 'paid' | 'void'

export interface RentalPerson {
  id: number
  full_name: string
  email: string
  phone: string
  tax_id: string
  address: string
  occupation: string
  employer: string
  monthly_income: number
  marital_status: string
  notes: string
}

export interface LeaseParty {
  id: number
  lease_id: number
  person_id: number
  role: PartyRole
  notify_email: boolean
  person?: RentalPerson
}

export interface Lease {
  id: number
  home_id: number
  home_name?: string
  home_address?: string
  agency_institution_id?: number | null
  agency_name?: string
  agent_person_id?: number | null
  agent_name?: string
  status: LeaseStatus
  start_date: string
  end_date: string
  term_months: number
  rent_amount: number
  maintenance_amount: number
  currency: string
  payment_day: number
  deposit_amount: number
  late_fee_type: 'none' | 'fixed' | 'percent'
  late_fee_value: number
  late_fee_cap: number
  auto_renew: boolean
  notice_days: number
  commission_percent: number
  commission_base: 'collected' | 'invoiced'
  owner_bank_account: string
  notes: string
  expiring?: boolean
  parties?: LeaseParty[]
  charge_count?: number
  invoice_count?: number
  balance?: number
  paid_total?: number
  charged_total?: number
}

export interface Reservation {
  id: number
  home_id: number
  home_name?: string
  home_address?: string
  applicant_person_id: number
  applicant?: RentalPerson
  agent_person_id?: number | null
  agent_name?: string
  converted_lease_id?: number | null
  status: ReservationStatus
  amount: number
  currency: string
  reference: string
  applied_to_deposit: boolean
  starts_at: string
  expires_at: string
  contract_date: string
  move_in_date: string
  rent_amount: number
  maintenance_amount: number
  owner_split_percent: number
  agency_split_percent: number
  refund_policy: 'refundable' | 'forfeitable' | 'custom'
  required_documents: string
  notes: string
}

export interface Charge {
  id: number
  lease_id: number
  kind: ChargeKind
  period: string
  description: string
  amount: number
  currency: string
  due_date: string
  status: ChargeStatus
  invoice_id?: number | null
}

export interface InvoiceLine {
  id: number
  invoice_id: number
  charge_id?: number | null
  description: string
  amount: number
}

export interface Invoice {
  id: number
  lease_id: number
  series: string
  number: number
  year: number
  status: 'issued' | 'void'
  issue_date: string
  due_date: string
  currency: string
  subtotal: number
  total: number
  pdf_sha256: string
  pdf_size_bytes: number
  sent_at?: string | null
  void_reason?: string
  number_label: string
  lines?: InvoiceLine[]
}

export interface PaymentProof {
  id: number
  payment_id: number
  original_name: string
  content_type: string
  size_bytes: number
  sha256: string
}

export interface Payment {
  id: number
  lease_id: number
  invoice_id?: number | null
  paid_at: string
  amount: number
  currency: string
  method: string
  reference: string
  notes: string
  receipt_id?: number | null
  proofs?: PaymentProof[]
}

export interface Receipt {
  id: number
  payment_id: number
  series: string
  number: number
  year: number
  status: string
  issue_date: string
  amount: number
  currency: string
}

export interface LeaseDocument {
  id: number
  lease_id: number
  doc_type: string
  original_name: string
  content_type: string
  size_bytes: number
  sha256: string
}

export interface Reminder {
  id: number
  home_id?: number | null
  lease_id?: number | null
  kind: 'gardening' | 'maintenance' | 'inspection' | 'payment_reminder' | 'custom'
  title: string
  description: string
  schedule_type: 'once' | 'monthly' | 'every_n_days' | 'yearly'
  interval_n: number
  anchor_date: string
  next_due_at: string
  lead_days: number
  notify_roles: string
  extra_emails: string
  enabled: boolean
}

export interface RentalSettings {
  issuer_name: string
  issuer_tax_id: string
  issuer_address: string
  issuer_email: string
  invoice_series: string
  receipt_series: string
  default_currency: string
  default_lead_days: number
  default_owner_split_percent: number
  default_agency_split_percent: number
}

export interface RentalBalance {
  charged: number
  paid: number
  balance: number
  currency: string
}

export interface HomeRentalStatus {
  home_id: number
  home_name: string
  home_address: string
  listing_enabled: boolean
  listing_status: ListingStatus
  active_lease?: Lease | null
  active_reservation?: Reservation | null
  next_due_date?: string
  upcoming_charges?: Charge[]
  reminders?: Reminder[]
}
