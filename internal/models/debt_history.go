package models

import "time"

// DebtHistoryPoint es un punto mensual de la evolución de deudas (SPEC-105).
// La métrica es cantidad de deudas, no montos.
type DebtHistoryPoint struct {
	Month       string `json:"month"` // YYYY-MM
	OpenCount   int    `json:"open_count"`
	ClosedCount int    `json:"closed_count"`
}

// DebtHistorySummary contiene los totales del período para una moneda.
type DebtHistorySummary struct {
	StartCount           int      `json:"start_count"`
	EndCount             int      `json:"end_count"`
	ChangeCount          int      `json:"change_count"`
	ChangePct            *float64 `json:"change_pct"`
	ClosedCount          int      `json:"closed_count"`
	PendingAmountCurrent float64  `json:"pending_amount_current"`
}

// DebtHistoryCurrency agrupa la serie y los totales de una moneda.
type DebtHistoryCurrency struct {
	Code    string             `json:"code"`
	Symbol  string             `json:"symbol"`
	Series  []DebtHistoryPoint `json:"series"`
	Summary DebtHistorySummary `json:"summary"`
}

// DebtHistory es la respuesta de GET /api/debts/history (SPEC-105).
type DebtHistory struct {
	Months      int                   `json:"months"`
	GeneratedAt time.Time             `json:"generated_at"`
	Currencies  []DebtHistoryCurrency `json:"currencies"`
}
