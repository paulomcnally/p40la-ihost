package models

import "time"

type Service struct {
	ID                    int64      `json:"id"`
	HomeID                int64      `json:"home_id"`
	Name                  string     `json:"name"`
	Institution           string     `json:"institution,omitempty"`
	CurrencyID            int64      `json:"currency_id"`
	Frequency             string     `json:"frequency"`
	SuggestedAmount       float64    `json:"suggested_amount"`
	Active                bool       `json:"active"`
	IconKey               string     `json:"icon_key"`
	BillingType           string     `json:"billing_type"`
	BillingDay            int        `json:"billing_day"`
	AutoGenerate          bool       `json:"auto_generate"`
	InstitutionID         *int64     `json:"institution_id,omitempty"`
	InstitutionAnalyzerID *int64     `json:"institution_analyzer_id,omitempty"`
	DeletedAt             *time.Time `json:"deleted_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}
