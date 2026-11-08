package models

import "time"

// Estados de un reconocimiento de comprobante (SPEC-127).
const (
	RecognitionStatusRecognized = "recognized"
	RecognitionStatusAssociated = "associated"
	RecognitionStatusRejected   = "rejected"
	RecognitionStatusExpired    = "expired"
)

// Tipos de entidad destino soportados por el reconocimiento (SPEC-127).
const (
	RecognitionEntityDebtBill = "debt_bill"
)

// DocumentRecognition representa la vista previa de un comprobante bancario
// reconocido y, tras la confirmación del usuario, su asociación a un pago.
type DocumentRecognition struct {
	ID                int64                  `json:"recognition_id"`
	DocumentHash      string                 `json:"document_hash,omitempty"`
	RecognizerID      string                 `json:"recognizer_id,omitempty"`
	DocumentType      string                 `json:"document_type"`
	Status            string                 `json:"status"`
	Reference         string                 `json:"reference,omitempty"`
	PaidAt            *time.Time             `json:"paid_at,omitempty"`
	Amount            float64                `json:"amount"`
	Currency          string                 `json:"currency,omitempty"`
	InstallmentNumber int                    `json:"installment_number,omitempty"`
	ConceptText       string                 `json:"concept,omitempty"`
	EntityType        string                 `json:"entity_type,omitempty"`
	EntityID          int64                  `json:"entity_id,omitempty"`
	Confidence        float64                `json:"confidence"`
	Warnings          []string               `json:"warnings,omitempty"`
	InboxPath         string                 `json:"-"`
	FinalPath         string                 `json:"final_path,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	ConfirmedAt       *time.Time             `json:"confirmed_at,omitempty"`
	ReviewRequired    bool                   `json:"review_required"`
	Duplicate         bool                   `json:"duplicate,omitempty"`
	Candidates        []RecognitionCandidate `json:"candidates,omitempty"`
}

// RecognitionCandidate es una cuota pagada que podría corresponder al
// comprobante reconocido. El score y los motivos ayudan al usuario a decidir.
type RecognitionCandidate struct {
	EntityType        string   `json:"entity_type"`
	DebtID            int64    `json:"debt_id"`
	DebtBillID        int64    `json:"debt_bill_id"`
	InstallmentNumber int      `json:"installment_number"`
	DebtDescription   string   `json:"debt_description,omitempty"`
	InstitutionName   string   `json:"institution_name,omitempty"`
	DueDate           string   `json:"due_date"`
	Amount            float64  `json:"amount"`
	CurrencyCode      string   `json:"currency_code,omitempty"`
	Status            string   `json:"status"`
	ExistingReference string   `json:"existing_reference,omitempty"`
	Score             float64  `json:"score"`
	Reasons           []string `json:"reasons,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
}

// ConfirmRecognitionInput contiene los datos revisados por el usuario antes
// de asociar un comprobante a una cuota pagada.
type ConfirmRecognitionInput struct {
	EntityType         string  `json:"entity_type"`
	DebtID             int64   `json:"debt_id"`
	DebtBillID         int64   `json:"debt_bill_id"`
	InstallmentNumber  int     `json:"installment_number"`
	Reference          string  `json:"reference"`
	PaidAt             string  `json:"paid_at,omitempty"`
	Amount             float64 `json:"amount"`
	Currency           string  `json:"currency,omitempty"`
	Concept            string  `json:"concept,omitempty"`
	OverwriteReference bool    `json:"overwrite_reference"`
	OverwriteFile      bool    `json:"overwrite_file"`
}

// AssociationResult es la respuesta de una asociación confirmada.
type AssociationResult struct {
	RecognitionID    int64  `json:"recognition_id"`
	Status           string `json:"status"`
	DebtBillID       int64  `json:"debt_bill_id"`
	PaymentReference string `json:"payment_reference"`
	FilePath         string `json:"file_path"`
}
