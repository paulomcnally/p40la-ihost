package rental

import "time"

// ModuleID es el identificador canónico del módulo (ARCH-MODULES).
const (
	ModuleID      = "realestate.rental"
	ModuleVersion = "1.0.0"
)

// Estados de contrato.
const (
	LeaseDraft      = "draft"
	LeaseActive     = "active"
	LeaseEnded      = "ended"
	LeaseTerminated = "terminated"
	LeaseCancelled  = "cancelled"
)

// LeaseStatuses contiene los estados válidos.
var LeaseStatuses = []string{LeaseDraft, LeaseActive, LeaseEnded, LeaseTerminated, LeaseCancelled}

// Roles de las partes.
const (
	RoleTenant        = "tenant"
	RoleGuarantor     = "guarantor"
	RoleOwner         = "owner"
	RoleAgencyContact = "agency_contact"
	RoleAgent         = "agent"
)

// PartyRoles contiene los roles válidos.
var PartyRoles = []string{RoleTenant, RoleGuarantor, RoleOwner, RoleAgencyContact, RoleAgent}

// Estados de apartado.
const (
	ReservationActive    = "active"
	ReservationConverted = "converted"
	ReservationExpired   = "expired"
	ReservationCancelled = "cancelled"
	ReservationForfeited = "forfeited"
)

// ReservationStatuses contiene los estados válidos.
var ReservationStatuses = []string{ReservationActive, ReservationConverted, ReservationExpired, ReservationCancelled, ReservationForfeited}

// Políticas de devolución de la seña.
const (
	RefundRefundable  = "refundable"
	RefundForfeitable = "forfeitable"
	RefundCustom      = "custom"
)

var RefundPolicies = []string{RefundRefundable, RefundForfeitable, RefundCustom}

// Estados de promoción derivados (ADR-002).
const (
	ListingAvailable = "available"
	ListingReserved  = "reserved"
	ListingRented    = "rented"
	ListingUnlisted  = "unlisted"
)

// Tipos de cargo.
const (
	ChargeRent        = "rent"
	ChargeMaintenance = "maintenance"
	ChargeDeposit     = "deposit"
	ChargeLateFee     = "late_fee"
	ChargeUtility     = "utility"
	ChargeOther       = "other"
)

var ChargeKinds = []string{ChargeRent, ChargeMaintenance, ChargeDeposit, ChargeLateFee, ChargeUtility, ChargeOther}

// Estados de cargo.
const (
	ChargePending       = "pending"
	ChargeInvoiced      = "invoiced"
	ChargePartiallyPaid = "partially_paid"
	ChargePaid          = "paid"
	ChargeVoid          = "void"
)

// Tipos de documento del contrato.
var DocumentTypes = []string{"contract", "addendum", "inventory", "id", "income_proof", "insurance", "agent_credential", "other"}

// Métodos de pago.
var PaymentMethods = []string{"cash", "transfer", "check", "card", "other"}

// Tipos de recordatorio y frecuencias.
var ReminderKinds = []string{"gardening", "maintenance", "inspection", "payment_reminder", "custom"}
var ScheduleTypes = []string{"once", "monthly", "every_n_days", "yearly"}

// Lista de claves de settings propias del módulo.
const (
	SettingIssuerName         = "rental_issuer_name"
	SettingIssuerTaxID        = "rental_issuer_tax_id"
	SettingIssuerAddress      = "rental_issuer_address"
	SettingIssuerEmail        = "rental_issuer_email"
	SettingInvoiceSeries      = "rental_invoice_series"
	SettingReceiptSeries      = "rental_receipt_series"
	SettingDefaultCurrency    = "rental_default_currency"
	SettingDefaultLeadDays    = "rental_default_lead_days"
	SettingDefaultOwnerSplit  = "rental_owner_split_percent"
	SettingDefaultAgencySplit = "rental_agency_split_percent"
	DefaultInvoiceSeries      = "ALQ"
	DefaultReceiptSeries      = "REC"
	DefaultCurrency           = "USD"
	DefaultLeadDays           = 3
	DefaultOwnerSplit         = 50.0
	DefaultAgencySplit        = 50.0
	MaxRentalAttachmentBytes  = 10 * 1024 * 1024
	MaxLeaseDocuments         = 20
	MaxPaymentProofs          = 10
)

// Person es una persona reutilizable del módulo (inquilino, garante, propietario, agente).
type Person struct {
	ID            int64      `json:"id"`
	FullName      string     `json:"full_name"`
	Email         string     `json:"email"`
	Phone         string     `json:"phone"`
	TaxID         string     `json:"tax_id"`
	Address       string     `json:"address"`
	Occupation    string     `json:"occupation"`
	Employer      string     `json:"employer"`
	MonthlyIncome float64    `json:"monthly_income"`
	MaritalStatus string     `json:"marital_status"`
	Notes         string     `json:"notes"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LeaseCount    int        `json:"lease_count,omitempty"`
}

// LeaseParty vincula una persona con un contrato y un rol.
type LeaseParty struct {
	ID          int64   `json:"id"`
	LeaseID     int64   `json:"lease_id"`
	PersonID    int64   `json:"person_id"`
	Role        string  `json:"role"`
	NotifyEmail bool    `json:"notify_email"`
	Person      *Person `json:"person,omitempty"`
}

// Lease es un contrato de alquiler.
type Lease struct {
	ID                  int64      `json:"id"`
	HomeID              int64      `json:"home_id"`
	HomeName            string     `json:"home_name,omitempty"`
	HomeAddress         string     `json:"home_address,omitempty"`
	AgencyInstitutionID *int64     `json:"agency_institution_id,omitempty"`
	AgencyName          string     `json:"agency_name,omitempty"`
	AgentPersonID       *int64     `json:"agent_person_id,omitempty"`
	AgentName           string     `json:"agent_name,omitempty"`
	Status              string     `json:"status"`
	StartDate           string     `json:"start_date"`
	EndDate             string     `json:"end_date"`
	TermMonths          int        `json:"term_months"`
	RentAmount          float64    `json:"rent_amount"`
	MaintenanceAmount   float64    `json:"maintenance_amount"`
	Currency            string     `json:"currency"`
	PaymentDay          int        `json:"payment_day"`
	DepositAmount       float64    `json:"deposit_amount"`
	LateFeeType         string     `json:"late_fee_type"`
	LateFeeValue        float64    `json:"late_fee_value"`
	LateFeeCap          float64    `json:"late_fee_cap"`
	AutoRenew           bool       `json:"auto_renew"`
	NoticeDays          int        `json:"notice_days"`
	CommissionPercent   float64    `json:"commission_percent"`
	CommissionBase      string     `json:"commission_base"`
	OwnerBankAccount    string     `json:"owner_bank_account"`
	Notes               string     `json:"notes"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// Derivados.
	Expiring     bool         `json:"expiring"`
	Parties      []LeaseParty `json:"parties,omitempty"`
	ChargeCount  int          `json:"charge_count,omitempty"`
	InvoiceCount int          `json:"invoice_count,omitempty"`
	Balance      float64      `json:"balance,omitempty"`
	PaidTotal    float64      `json:"paid_total,omitempty"`
	ChargedTotal float64      `json:"charged_total,omitempty"`
}

// Reservation es un apartado con vencimiento.
type Reservation struct {
	ID                 int64     `json:"id"`
	HomeID             int64     `json:"home_id"`
	HomeName           string    `json:"home_name,omitempty"`
	HomeAddress        string    `json:"home_address,omitempty"`
	ApplicantPersonID  int64     `json:"applicant_person_id"`
	Applicant          *Person   `json:"applicant,omitempty"`
	AgentPersonID      *int64    `json:"agent_person_id,omitempty"`
	AgentName          string    `json:"agent_name,omitempty"`
	ConvertedLeaseID   *int64    `json:"converted_lease_id,omitempty"`
	Status             string    `json:"status"`
	Amount             float64   `json:"amount"`
	Currency           string    `json:"currency"`
	Reference          string    `json:"reference"`
	AppliedToDeposit   bool      `json:"applied_to_deposit"`
	StartsAt           string    `json:"starts_at"`
	ExpiresAt          string    `json:"expires_at"`
	ContractDate       string    `json:"contract_date"`
	MoveInDate         string    `json:"move_in_date"`
	RentAmount         float64   `json:"rent_amount"`
	MaintenanceAmount  float64   `json:"maintenance_amount"`
	OwnerSplitPercent  float64   `json:"owner_split_percent"`
	AgencySplitPercent float64   `json:"agency_split_percent"`
	RefundPolicy       string    `json:"refund_policy"`
	RequiredDocuments  string    `json:"required_documents"`
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// HomeRentalStatus es el estado de promoción derivado de una casa.
type HomeRentalStatus struct {
	HomeID            int64        `json:"home_id"`
	HomeName          string       `json:"home_name"`
	HomeAddress       string       `json:"home_address"`
	ListingEnabled    bool         `json:"listing_enabled"`
	ListingStatus     string       `json:"listing_status"`
	ActiveLease       *Lease       `json:"active_lease,omitempty"`
	ActiveReservation *Reservation `json:"active_reservation,omitempty"`
	NextDueDate       string       `json:"next_due_date,omitempty"`
	UpcomingCharges   []Charge     `json:"upcoming_charges,omitempty"`
	Reminders         []Reminder   `json:"reminders,omitempty"`
}

// Charge es un cargo del contrato.
type Charge struct {
	ID          int64     `json:"id"`
	LeaseID     int64     `json:"lease_id"`
	Kind        string    `json:"kind"`
	Period      string    `json:"period"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	DueDate     string    `json:"due_date"`
	Status      string    `json:"status"`
	InvoiceID   *int64    `json:"invoice_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Invoice es una factura interna de cobro.
type Invoice struct {
	ID          int64         `json:"id"`
	LeaseID     int64         `json:"lease_id"`
	Series      string        `json:"series"`
	Number      int64         `json:"number"`
	Year        int           `json:"year"`
	Status      string        `json:"status"`
	IssueDate   string        `json:"issue_date"`
	DueDate     string        `json:"due_date"`
	Currency    string        `json:"currency"`
	Subtotal    float64       `json:"subtotal"`
	Total       float64       `json:"total"`
	Snapshot    string        `json:"-"`
	PDFPath     string        `json:"-"`
	PDFSHA256   string        `json:"pdf_sha256"`
	PDFSize     int64         `json:"pdf_size_bytes"`
	SentAt      *time.Time    `json:"sent_at,omitempty"`
	VoidReason  string        `json:"void_reason,omitempty"`
	VoidedAt    *time.Time    `json:"voided_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	Lines       []InvoiceLine `json:"lines,omitempty"`
	NumberLabel string        `json:"number_label"`
}

// InvoiceLine es una línea de factura.
type InvoiceLine struct {
	ID          int64   `json:"id"`
	InvoiceID   int64   `json:"invoice_id"`
	ChargeID    *int64  `json:"charge_id,omitempty"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// Payment es un pago registrado.
type Payment struct {
	ID        int64     `json:"id"`
	LeaseID   int64     `json:"lease_id"`
	InvoiceID *int64    `json:"invoice_id,omitempty"`
	PaidAt    string    `json:"paid_at"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Method    string    `json:"method"`
	Reference string    `json:"reference"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	ReceiptID *int64    `json:"receipt_id,omitempty"`
	Proofs    []Proof   `json:"proofs,omitempty"`
}

// Proof es el comprobante adjunto de un pago.
type Proof struct {
	ID           int64     `json:"id"`
	PaymentID    int64     `json:"payment_id"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	SHA256       string    `json:"sha256"`
	CreatedAt    time.Time `json:"created_at"`
}

// Receipt es el recibo emitido por un pago.
type Receipt struct {
	ID        int64      `json:"id"`
	PaymentID int64      `json:"payment_id"`
	Series    string     `json:"series"`
	Number    int64      `json:"number"`
	Year      int        `json:"year"`
	Status    string     `json:"status"`
	IssueDate string     `json:"issue_date"`
	Amount    float64    `json:"amount"`
	Currency  string     `json:"currency"`
	PDFSHA256 string     `json:"pdf_sha256"`
	PDFSize   int64      `json:"pdf_size_bytes"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// LeaseDocument es un adjunto legal del contrato.
type LeaseDocument struct {
	ID           int64     `json:"id"`
	LeaseID      int64     `json:"lease_id"`
	DocType      string    `json:"doc_type"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	SHA256       string    `json:"sha256"`
	CreatedAt    time.Time `json:"created_at"`
}

// Reminder es un recordatorio programado.
type Reminder struct {
	ID           int64     `json:"id"`
	HomeID       *int64    `json:"home_id,omitempty"`
	LeaseID      *int64    `json:"lease_id,omitempty"`
	Kind         string    `json:"kind"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ScheduleType string    `json:"schedule_type"`
	IntervalN    int       `json:"interval_n"`
	AnchorDate   string    `json:"anchor_date"`
	NextDueAt    string    `json:"next_due_at"`
	LeadDays     int       `json:"lead_days"`
	NotifyRoles  string    `json:"notify_roles"`
	ExtraEmails  string    `json:"extra_emails"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RentalSettings agrupa la configuración del módulo.
type RentalSettings struct {
	IssuerName         string  `json:"issuer_name"`
	IssuerTaxID        string  `json:"issuer_tax_id"`
	IssuerAddress      string  `json:"issuer_address"`
	IssuerEmail        string  `json:"issuer_email"`
	InvoiceSeries      string  `json:"invoice_series"`
	ReceiptSeries      string  `json:"receipt_series"`
	DefaultCurrency    string  `json:"default_currency"`
	DefaultLeadDays    int     `json:"default_lead_days"`
	DefaultOwnerSplit  float64 `json:"default_owner_split_percent"`
	DefaultAgencySplit float64 `json:"default_agency_split_percent"`
}

// Balance resume cargos y pagos de un contrato.
type Balance struct {
	Charged  float64 `json:"charged"`
	Paid     float64 `json:"paid"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}
