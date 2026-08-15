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
  billing_day: number
  auto_generate: boolean
  institution_id?: number
  institution_analyzer_id?: number
}

export interface Institution {
  id: number
  name: string
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
}

export interface Settings {
  language: string
}
