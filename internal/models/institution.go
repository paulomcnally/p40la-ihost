package models

import "time"

type Institution struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	CategoryID *int64     `json:"category_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type InstitutionAnalyzer struct {
	ID            int64     `json:"id"`
	InstitutionID int64     `json:"institution_id"`
	AnalyzerID    string    `json:"analyzer_id"`
	CreatedAt     time.Time `json:"created_at"`
}
