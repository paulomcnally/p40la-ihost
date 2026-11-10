export interface Home {
  id: number
  name: string
  address: string
  sort_order: number
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
  is_insurance: boolean
  webhook_uuid?: string
  last_webhook_request?: string | null
  automation_account_id?: number
  latest_bill_status: 'paid' | 'pending' | null
  has_contract?: boolean
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
  currency_id: number
  suggested_amount: number
  frequency: string
  icon_key: string
  active: boolean
  start_date?: string
  end_date?: string
  is_recurring: boolean
  created_at: string
}

export interface ServiceAuto {
  auto_id: number
  brand: string
  model: string
  year: number
  color: string
  icon: string
  placa: string
  coverage_type: 'daños_a_terceros' | 'full_cover'
  policy_number: string
  certificate?: string
  insurer_number: string
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
  cycle_id?: number
  year: number
  month: number
  amount: number
  invoice_number: string
  status: 'pending' | 'paid'
  drive_url: string
  issue_date?: string
  due_date?: string
  paid_at?: string
  payment_reference?: string
}

// Factura con contexto para el home global de Servicios (SPEC-120).
export interface ServiceBillDetail extends Bill {
  service_name: string
  institution: string
  home_id: number
  home_name: string
  icon_key: string
  currency_id: number
  currency_code: string
  currency_symbol: string
  frequency: 'monthly' | 'yearly'
  billing_day: number | null
  active: boolean
  period_year: number
  period_month: number
  period_date: string
  date_estimated: boolean
}

export interface ServiceBillSummaryRow {
  period: string
  year: number
  month: number
  currency_code: string
  currency_symbol: string
  total: number
  paid: number
  pending: number
  count: number
}

export interface ServiceBillSummaryResponse {
  from: string
  to: string
  periods: ServiceBillSummaryRow[]
}

export interface ServiceProjectionItem {
  service_id: number
  name: string
  amount: number
  source: 'bill' | 'estimate'
  currency_code: string
}

export interface ServiceProjectionPeriod {
  period: string
  year: number
  month: number
  currency_code: string
  currency_symbol: string
  confirmed: number
  estimated: number
  total: number
  services: ServiceProjectionItem[]
}

export interface ServiceProjectionResponse {
  from: string
  months: number
  periods: ServiceProjectionPeriod[]
  estimate_note: boolean
}

export interface ServiceCycle {
  id: number
  service_id: number
  sequence: number
  start_date?: string
  end_date?: string
  created_at: string
}

export interface FieldChange {
  field: string
  old?: unknown
  new?: unknown
}

export interface BillHistoryEntry {
  id: number
  bill_id: number
  action: 'created' | 'updated' | 'paid'
  source: 'dashboard' | 'webhook'
  changes?: FieldChange[]
  created_at: string
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

export interface AutoAttachment {
  id: number
  auto_id: number
  owner_type: 'maintenance' | 'inspection'
  owner_id: number
  file_name: string
  size_bytes: number
  created_at: string
}

export type AutoMaintenanceItemType = 'labor' | 'supplies'

export interface AutoMaintenanceItem {
  id: number
  maintenance_id: number
  item_type: AutoMaintenanceItemType
  description: string
  amount: number
  created_at: string
  updated_at: string
}

export interface AutoMaintenanceItemInput {
  item_type: AutoMaintenanceItemType
  description: string
  amount: number
}

export interface AutoMaintenanceInput {
  performed_at: string
  mileage: number
  institution_id: number
  description: string
  currency_id: number
}

export interface AutoMaintenance {
  id: number
  auto_id: number
  performed_at: string
  mileage: number
  institution_id: number
  institution_name: string
  description: string
  labor_cost: number
  supplies_cost: number
  total_cost: number
  currency_id: number
  currency_code: string
  items: AutoMaintenanceItem[]
  attachments: AutoAttachment[]
  created_at: string
  updated_at: string
}

export interface AutoMaintenanceCurrencyTotal {
  currency_id: number
  currency_code: string
  total: number
}

export interface AutoMaintenanceList {
  items: AutoMaintenance[]
  latest_mileage: number
  totals_by_currency: AutoMaintenanceCurrencyTotal[]
}

export type AutoInspectionResult = 'approved' | 'rejected'
export type InspectionStatus = 'overdue' | 'due_soon' | 'ok'

export interface AutoInspection {
  id: number
  auto_id: number
  performed_at: string
  inspection_year: number
  institution_id: number
  institution_name: string
  result: AutoInspectionResult
  notes: string
  attachments: AutoAttachment[]
  created_at: string
  updated_at: string
}

export interface InspectionSchedule {
  first_due_year: number
  next_due_year: number
  next_due_date: string
  status: InspectionStatus
  days_until_due: number
  approved_years: number[]
  rejected_years: number[]
}

export interface AutoInspectionList {
  items: AutoInspection[]
  schedule: InspectionSchedule
}

export interface AutoInspectionAlert {
  auto_id: number
  next_due_year: number
  next_due_date: string
  status: InspectionStatus
  days_until_due: number
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

export interface PensionCategory {
  id: number
  name: string
  description: string
  auto_generate: boolean
  share_percent: number
  currency: string
  created_at: string
  updated_at: string
}

export interface PensionCategoryKeyword {
  id: number
  pension_category_id: number
  keyword: string
  normalized_keyword: string
  source: string
  created_at: string
  updated_at: string
}

export interface PensionImportPeriod {
  year: number
  month: number
}

export interface ImportValidationError {
  line_no?: number
  code: string
  message: string
}

export interface PensionImportValidation {
  status: 'ok' | 'invalid'
  declared_reimbursable_total: number
  computed_reimbursable_total: number
  computed_grand_total: number
  errors: ImportValidationError[]
}

export interface PensionImportLine {
  id?: number
  batch_id?: number
  line_no: number
  raw_text: string
  concept: string
  amount_total: number
  amount_to_record: number
  rounding_adjustment: number
  share_percent: number
  currency: string
  suggested_category_id?: number
  chosen_category_id?: number
  child_id?: number
  confidence: number
  match_reason: string
  matched_keyword?: string
  category_name?: string
  validation_status: 'ok' | 'invalid'
  validation_code: string
  action: string
  message?: string
  previous_amount?: number
  existing_amount?: number
  variation_pct?: number
}

export interface PensionImportControlLine {
  line_no: number
  raw_text: string
  amount: number
}

export interface PensionImportPreview {
  batch_id: number
  period: PensionImportPeriod
  document_period?: PensionImportPeriod
  period_mismatch: boolean
  control_lines: PensionImportControlLine[]
  source: string
  filename?: string
  validation: PensionImportValidation
  warnings?: string[]
  lines: PensionImportLine[]
}

export interface PensionImportBatch {
  id: number
  filename: string
  period_year: number
  period_month: number
  status: string
  total_lines: number
  declared_total: number
  computed_total: number
  created_count: number
  skipped_count: number
  created_at: string
  confirmed_at?: string
}

export interface PensionImportConfirmRow {
  line_no: number
  action: string
  category_id: number
  amount_total: number
  status: string
}

export interface PensionImportConfirmResult {
  batch_id: number
  created: number
  skipped_existing: number
  learned_keywords: number
  errors: ImportValidationError[]
}

export interface Alert {
  id: number
  key: string
  title: string
  description: string
  mail_enabled: boolean
  voice_enabled: boolean
  telegram_enabled: boolean
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

export type SupportRecordStatus = 'pending' | 'paid' | 'rejected'
export type SalaryPaymentStatus = 'pending' | 'received'

export interface SupportRecord {
  id: number
  child_id: number
  child_name: string
  pension_category_id: number
  category_name: string
  year: number
  month: number
  amount: number
  currency: string
  status: SupportRecordStatus
  paid_at?: string
  payment_method?: string
  payment_reference?: string
  evidence_notes?: string
  notes?: string
  proof_file_name?: string
  original_amount?: number
  original_currency?: string
  exchange_rate?: number
  created_at: string
  updated_at: string
}

export interface SalaryPayment {
  id: number
  salary_id: number
  employer: string
  year: number
  month: number
  amount: number
  currency: string
  status: SalaryPaymentStatus
  received_amount?: number
  received_at?: string
  notes?: string
  created_at: string
  updated_at: string
}

export interface MonthClosing {
  closed: boolean
  closed_at: string | null
}

export interface ChildSupportConfig {
  id: number
  child_id: number
  child_name?: string
  pension_category_id: number
  category_name?: string
  amount: number
  currency: string
  is_active: boolean
  auto_generate: boolean
  created_at: string
  updated_at: string
}

export type DebtStatus = 'activa' | 'inactiva' | 'finalizada'

export interface Debt {
  id: number
  institution_id: number
  institution_name?: string
  identifier: string
  description: string
  total: number
  principal: number
  currency_id: number
  currency_code?: string
  installments_total: number
  installment_amount: number
  interest_rate: number
  payment_day: number
  start_date: string
  status: DebtStatus
  pending_count: number
  pending_amount: number
  paid_count: number
  paid_amount: number
  next_due_date?: string
  last_due_date?: string
  last_paid_at?: string
  has_contract?: boolean
  has_documents?: boolean
  created_at: string
  updated_at: string
}

export interface DebtBill {
  id: number
  debt_id: number
  debt_description?: string
  institution_name?: string
  currency_code?: string
  debt_status?: DebtStatus
  installment_number: number
  due_date: string
  issue_date?: string
  amount: number
  status: 'pending' | 'paid'
  paid_at?: string
  payment_reference?: string
  has_invoice?: boolean
  has_receipt?: boolean
  created_at: string
  updated_at: string
}

// SPEC-131: plantillas de información y forma de pago por deuda.
export type DebtInformationFieldType =
  | 'text'
  | 'textarea'
  | 'number'
  | 'date'
  | 'select'
  | 'checkbox'
  | 'computed'

export interface DebtInformationField {
  key: string
  label: string
  type: DebtInformationFieldType
  required?: boolean
  placeholder?: string
  default_value?: string
  help_text?: string
  options?: string[]
  value_template?: string
}

export interface DebtInformationTemplate {
  id: number
  name: string
  description: string
  fields?: DebtInformationField[]
  field_count: number
  usage_count: number
  created_at: string
  updated_at: string
}

export type DebtInformationValue = string | number | boolean

export interface DebtPaymentInformation {
  debt_id: number
  template_id: number | null
  template: DebtInformationTemplate | null
  values: Record<string, DebtInformationValue>
  created_at?: string
  updated_at?: string
}

export interface DebtInformationToken {
  key: string
  label: string
  example: string
}

// SPEC-132: variables para valores por defecto de campos text/textarea.
export interface DebtInformationVariable {
  key: string
  label: string
  example: string
  description?: string
}

// SPEC-133: bancos, cuentas y formatos de transferencia.
export interface Bank {
  id: number
  name: string
  short_code: string
  country: string
  notes: string
  active: boolean
  account_count: number
  created_at: string
  updated_at: string
}

export type BankAccountType = 'savings' | 'checking'
export type BankTransferKind = 'transfer' | 'ach'

export interface BankAccount {
  id: number
  bank_id: number
  bank_name?: string
  alias: string
  account_number: string
  account_type: BankAccountType
  currency_id: number
  currency_code?: string
  currency_symbol?: string
  beneficiary: string
  notes: string
  active: boolean
  formats?: BankTransferFormat[]
  created_at: string
  updated_at: string
}

export interface BankTransferFormat {
  id: number
  bank_account_id: number
  kind: BankTransferKind
  fields: Record<string, string>
  created_at: string
  updated_at: string
}

// SPEC-127: reconocimiento de comprobantes bancarios.
export interface RecognitionCandidate {
  entity_type: string
  debt_id: number
  debt_bill_id: number
  installment_number: number
  debt_description?: string
  institution_name?: string
  due_date: string
  amount: number
  currency_code?: string
  status: string
  existing_reference?: string
  score: number
  reasons?: string[]
  warnings?: string[]
}

export interface DocumentRecognition {
  recognition_id: number
  document_hash?: string
  recognizer_id?: string
  document_type: string
  status: 'recognized' | 'associated' | 'rejected' | 'expired'
  reference?: string
  paid_at?: string
  amount: number
  currency?: string
  installment_number?: number
  concept?: string
  entity_type?: string
  entity_id?: number
  confidence: number
  warnings?: string[]
  final_path?: string
  review_required: boolean
  duplicate?: boolean
  candidates?: RecognitionCandidate[]
  created_at: string
  updated_at: string
  confirmed_at?: string
}

export interface ConfirmRecognitionInput {
  entity_type: string
  debt_id: number
  debt_bill_id: number
  installment_number: number
  reference: string
  paid_at?: string
  amount: number
  currency?: string
  concept?: string
  overwrite_reference: boolean
  overwrite_file: boolean
}

export interface AssociationResult {
  recognition_id: number
  status: string
  debt_bill_id: number
  payment_reference: string
  file_path: string
}

// SPEC-105: evolución mensual de deudas (cantidad, no montos).
export interface DebtHistoryPoint {
  month: string
  open_count: number
  closed_count: number
  outstanding_amount: number
}

export interface DebtHistorySummary {
  start_count: number
  end_count: number
  change_count: number
  change_pct: number | null
  closed_count: number
  pending_amount_current: number
  start_amount: number
  end_amount: number
  change_amount: number
  change_pct_amount: number | null
}

export interface DebtHistoryCurrency {
  code: string
  symbol: string
  series: DebtHistoryPoint[]
  summary: DebtHistorySummary
}

export interface DebtHistoryResponse {
  months: number
  generated_at: string
  currencies: DebtHistoryCurrency[]
}

export interface CategoryGroup {
  id: number
  name: string
  icon: string
  sort_order: number
  categories: BudgetCategory[]
}

export interface BudgetCategory {
  id: number
  category_group_id: number
  name: string
  icon: string
  sort_order: number
  target_amount?: number | null
  target_type?: string | null
  target_date?: string | null
  service_ids?: number[]
  account_id?: number | null
  service_names?: string[]
  deleted_at?: string | null
  created_at: string
  updated_at: string
}

export interface Account {
  id: number
  name: string
  type: 'checking' | 'savings' | 'credit_card' | 'cash'
  currency_id: number
  currency_code?: string
  starting_balance: number
  balance: number
  created_at: string
  updated_at: string
}

export interface RecurringRule {
  id: number
  category_id: number
  amount: number
  frequency: string
  start_month: string
  end_month?: string
  active: boolean
}

export interface Assignment {
  id: number
  budget_month_id: number
  category_id: number
  currency_id: number
  currency_code?: string
  amount: number
  source: 'one_time' | 'recurring'
  recurring_rule_id?: number
}

export interface BudgetTransaction {
  id: number
  account_id: number
  account_name?: string
  category_id?: number | null
  category_name?: string
  currency_id: number
  currency_code?: string
  date: string
  payee: string
  memo: string
  outflow: number
  inflow: number
  cleared: boolean
  created_at: string
  updated_at: string
}

export interface Transfer {
  id: number
  from_account_id: number
  from_account_name?: string
  to_account_id: number
  to_account_name?: string
  currency_id: number
  currency_code?: string
  date: string
  payee: string
  memo: string
  amount: number
  cleared: boolean
  created_at: string
  updated_at: string
}

export interface BudgetCategoryRow {
  id: number
  category_group_id: number
  name: string
  icon: string
  sort_order: number
  target_amount?: number | null
  service_ids?: number[]
  account_id?: number | null
  service_names?: string[]
  assigned: Record<string, number>
  activity: Record<string, number>
  available: Record<string, number>
  recurring_rule?: RecurringRule | null
}

export interface BudgetMonthGroup {
  id: number
  name: string
  icon: string
  sort_order: number
  categories: BudgetCategoryRow[]
}

export interface CurrencyTotal {
  currency_id: number
  code: string
  symbol: string
  total_income: number
  total_assigned: number
  unassigned: number
}

export interface BudgetMonthView {
  year: number
  month: number
  currency_totals: CurrencyTotal[]
  groups: BudgetMonthGroup[]
}

export interface SuggestedAssignment {
  suggested: Record<string, number>
  applies: boolean
  source_service_ids?: number[]
  source_service_names?: string[]
}

// --- Automation unificada (SPEC-111) ---

export interface AutomationAccountOption {
  id: number
  label: string
}

export interface AutomationAccount {
  id: number
  institution_id: number
  institution_name?: string
  identifier: string
  label?: string
  plugin_name?: string | null
  webhook_url?: string | null
  schedule_time?: string | null
  schedule_enabled: boolean
  last_run_at?: string | null
  schedule_frequency: string
  schedule_interval?: number | null
  schedule_days?: string | null
  created_at: string
  updated_at: string
}

export interface AutomationPlugin {
  id: number
  name: string
  version: string
  source: string
  description?: string
  status: string
  created_at: string
  updated_at: string
}

export interface AutomationBill {
  id: number
  account_id: number
  plugin_name: string
  plugin_version: string
  source: string
  status: string
  error?: string | null
  error_code?: string | null
  fetched_at: string
  created_at: string
}

export interface AutomationWebhookLog {
  id: number
  account_id: number
  webhook_url: string
  status: string
  response?: string
  ran_at: string
  http_status?: number | null
}

export interface AutomationCredentialField {
  key: string
  label: string
  type: 'text' | 'password'
  required: boolean
  secret: boolean
  description?: string
}

export interface AutomationBillItem {
  period: string
  amount: string
  due_date: string
  issue_date: string
  status: string
}

export interface AutomationAccountInput {
  institution_id: number
  identifier: string
  label?: string
  service_id?: number
  credentials?: string
  plugin_name?: string
  webhook_url?: string
  schedule_enabled?: boolean
  schedule_time?: string
  schedule_frequency?: string
  schedule_interval?: number | null
  schedule_days?: string | null
}
