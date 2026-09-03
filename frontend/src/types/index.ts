export interface Home {
  id: number
  name: string
  address: string
}

export interface Currency {
  id: number
  code: string
  name: string
  symbol: string
}

export interface Service {
  id: number
  home_id: number
  name: string
  institution: string
  currency_id: number
  frequency: 'monthly' | 'yearly'
  suggested_amount: number
  active: boolean
  icon_key: string
  billing_type: 'fixed' | 'variable'
  billing_day: number | null
  auto_generate: boolean
  institution_id?: number
  institution_analyzer_id?: number
  start_date?: string
  end_date?: string
  is_recurring: boolean
  latest_bill_status: 'paid' | 'pending' | null
}

export interface AutoService {
  id: number
  auto_id: number
  service_id: number
  coverage_type: 'daños_a_terceros' | 'full_cover'
  policy_number: string
  certificate?: string
  insurer_number: string
  service_name: string
  institution_name: string
  institution_id?: number
  suggested_amount: number
  frequency: string
  icon_key: string
  active: boolean
  start_date?: string
  end_date?: string
  is_recurring: boolean
  created_at: string
}

export interface InstitutionCategory {
  id: number
  key: string
  name: string
  description: string
  icon_key: string
  created_at: string
  updated_at: string
}

export interface Institution {
  id: number
  name: string
  category_id?: number
  created_at: string
  updated_at: string
}

export interface InstitutionAnalyzer {
  id: number
  institution_id: number
  analyzer_id: string
  created_at: string
}

export interface AnalyzerInfo {
  id: string
  name: string
}

export interface Bill {
  id: number
  service_id: number
  year: number
  month: number
  amount: number
  invoice_number: string
  status: 'pending' | 'paid'
  drive_url: string
  paid_at?: string
  payment_reference?: string
}

export interface Settings {
  language: string
}

export interface Auto {
  id: number
  year: number
  model: string
  brand: string
  color: string
  icon: string
  motor: string
  chasis: string
  vin: string
  placa: string
  created_at: string
  updated_at: string
}

export interface Child {
  id: number
  first_name: string
  last_name: string
  birth_date: string
  notes: string
  created_at: string
  updated_at: string
}

export interface Salary {
  id: number
  employer: string
  amount: number
  currency_id: number
  payment_day: number
  active: boolean
  note: string
  created_at: string
  updated_at: string
}

export interface Alert {
  id: number
  key: string
  title: string
  description: string
  mail_enabled: boolean
  voice_enabled: boolean
  speech: string
  created_at: string
  updated_at: string
}

export interface Notification {
  id: number
  name: string
  email: string
  active: boolean
  created_at: string
  updated_at: string
}
