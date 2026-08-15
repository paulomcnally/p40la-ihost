package analyzers

import (
	"io"
	"time"
)

type AnalyzerInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ExtractedBill struct {
	Amount        float64                `json:"amount"`
	InvoiceNumber string                 `json:"invoice_number,omitempty"`
	Year          int                    `json:"year"`
	Month         int                    `json:"month"`
	DueDate       *time.Time             `json:"due_date,omitempty"`
	RawData       map[string]interface{} `json:"raw_data,omitempty"`
}

type DocumentAnalyzer interface {
	Info() AnalyzerInfo
	Analyze(reader io.Reader, mimeType string) (*ExtractedBill, error)
}
