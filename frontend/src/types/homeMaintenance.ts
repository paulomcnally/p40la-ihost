export type HomeMaintenanceKind = 'corrective' | 'preventive'

export type HomeMaintenanceArea =
  | 'plumbing'
  | 'electrical'
  | 'roof'
  | 'paint'
  | 'carpentry'
  | 'appliances'
  | 'garden'
  | 'cleaning'
  | 'other'

export type HomeMaintenanceItemType = 'labor' | 'supplies'

export type HomeMaintenanceUnit =
  | 'unit'
  | 'm'
  | 'm2'
  | 'kg'
  | 'lb'
  | 'l'
  | 'gal'
  | 'sack'
  | 'box'

export interface HomeMaintenanceCurrencyTotal {
  currency_id: number
  currency_code: string
  labor_cost: number
  supplies_cost: number
  total: number
}

export interface HomeMaintenanceItem {
  id: number
  maintenance_id: number
  item_type: HomeMaintenanceItemType
  description: string
  quantity: number
  unit: HomeMaintenanceUnit
  amount: number
  currency_id: number
  currency_code: string
  institution_id: number | null
  institution_name: string
  created_at: string
  updated_at: string
}

export interface HomeMaintenanceItemInput {
  item_type: HomeMaintenanceItemType
  description: string
  quantity: number
  unit: HomeMaintenanceUnit
  amount: number
  currency_id: number
  institution_id: number | null
}

export interface HomeMaintenanceInput {
  performed_at: string
  kind: HomeMaintenanceKind
  area: HomeMaintenanceArea
  institution_id: number | null
  description: string
}

export interface HomeMaintenance {
  id: number
  home_id: number
  performed_at: string
  kind: HomeMaintenanceKind
  area: HomeMaintenanceArea
  institution_id: number | null
  institution_name: string
  description: string
  totals_by_currency: HomeMaintenanceCurrencyTotal[]
  items: HomeMaintenanceItem[]
  created_at: string
  updated_at: string
}

export interface HomeMaintenanceList {
  items: HomeMaintenance[]
  totals_by_currency: HomeMaintenanceCurrencyTotal[]
}
