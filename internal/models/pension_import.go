package models

import "time"

// PensionCategoryKeyword asocia un texto (palabra o frase) a una categoría de
// pensión. El matching usa normalized_keyword y el usuario puede editarlas.
type PensionCategoryKeyword struct {
	ID                int64     `json:"id"`
	PensionCategoryID int64     `json:"pension_category_id"`
	Keyword           string    `json:"keyword"`
	NormalizedKeyword string    `json:"normalized_keyword"`
	Source            string    `json:"source"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// PensionImportBatch representa una importación mensual (preview o confirmada).
type PensionImportBatch struct {
	ID            int64      `json:"id"`
	Filename      string     `json:"filename"`
	FileHash      string     `json:"file_hash"`
	PeriodYear    int        `json:"period_year"`
	PeriodMonth   int        `json:"period_month"`
	Status        string     `json:"status"`
	TotalLines    int        `json:"total_lines"`
	DeclaredTotal float64    `json:"declared_total"`
	ComputedTotal float64    `json:"computed_total"`
	CreatedCount  int        `json:"created_count"`
	SkippedCount  int        `json:"skipped_count"`
	CreatedAt     time.Time  `json:"created_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
}

// PensionImportRow es una línea parseada y validada de una importación.
type PensionImportRow struct {
	ID                  int64     `json:"id"`
	BatchID             int64     `json:"batch_id"`
	LineNo              int       `json:"line_no"`
	RawText             string    `json:"raw_text"`
	Concept             string    `json:"concept"`
	AmountTotal         float64   `json:"amount_total"`
	AmountToRecord      float64   `json:"amount_to_record"`
	RoundingAdjustment  float64   `json:"rounding_adjustment"`
	SharePercent        int       `json:"share_percent"`
	Currency            string    `json:"currency"`
	SuggestedCategoryID *int64    `json:"suggested_category_id,omitempty"`
	ChosenCategoryID    *int64    `json:"chosen_category_id,omitempty"`
	ChildID             *int64    `json:"child_id,omitempty"`
	Confidence          float64   `json:"confidence"`
	MatchReason         string    `json:"match_reason"`
	MatchedKeyword      string    `json:"matched_keyword,omitempty"`
	ValidationStatus    string    `json:"validation_status"`
	ValidationCode      string    `json:"validation_code"`
	Action              string    `json:"action"`
	CategoryName        string    `json:"category_name,omitempty"`
	Message             string    `json:"message,omitempty"`
	PreviousAmount      *float64  `json:"previous_amount,omitempty"`
	ExistingAmount      *float64  `json:"existing_amount,omitempty"`
	VariationPct        *float64  `json:"variation_pct,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// PensionImportValidation resume el resultado del control del lote.
type PensionImportValidation struct {
	Status                    string                  `json:"status"`
	DeclaredReimbursableTotal float64                 `json:"declared_reimbursable_total"`
	ComputedReimbursableTotal float64                 `json:"computed_reimbursable_total"`
	ComputedGrandTotal        float64                 `json:"computed_grand_total"`
	Errors                    []ImportValidationError `json:"errors"`
}

// ImportValidationError describe un error del lote o de una fila.
type ImportValidationError struct {
	LineNo  int    `json:"line_no,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PensionImportPreview es la respuesta del endpoint de preview.
type PensionImportPreview struct {
	BatchID        int64                      `json:"batch_id"`
	Period         PensionImportPeriod        `json:"period"`
	Source         string                     `json:"source"`
	Filename       string                     `json:"filename,omitempty"`
	Validation     PensionImportValidation    `json:"validation"`
	DocumentPeriod *PensionImportPeriod       `json:"document_period,omitempty"`
	PeriodMismatch bool                       `json:"period_mismatch"`
	ControlLines   []PensionImportControlLine `json:"control_lines"`
	Warnings       []string                   `json:"warnings,omitempty"`
	Lines          []PensionImportRow         `json:"lines"`
}

// PensionImportControlLine es una línea de total/subtotal del documento.
type PensionImportControlLine struct {
	LineNo  int     `json:"line_no"`
	RawText string  `json:"raw_text"`
	Amount  float64 `json:"amount"`
}

// PensionImportPeriod identifica el período detectado.
type PensionImportPeriod struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

// PensionImportConfirmRow es una fila enviada por el cliente al confirmar.
type PensionImportConfirmRow struct {
	LineNo      int     `json:"line_no"`
	Action      string  `json:"action"`
	CategoryID  int64   `json:"category_id"`
	AmountTotal float64 `json:"amount_total"`
	Status      string  `json:"status"`
}

// PensionImportConfirmRequest es el cuerpo de confirmación.
type PensionImportConfirmRequest struct {
	BatchID int64                     `json:"batch_id"`
	ChildID int64                     `json:"child_id"`
	Rows    []PensionImportConfirmRow `json:"rows"`
}

// PensionImportConfirmResult resume la confirmación.
type PensionImportConfirmResult struct {
	BatchID         int64                   `json:"batch_id"`
	Created         int                     `json:"created"`
	SkippedExisting int                     `json:"skipped_existing"`
	LearnedKeywords int                     `json:"learned_keywords"`
	Errors          []ImportValidationError `json:"errors"`
}
