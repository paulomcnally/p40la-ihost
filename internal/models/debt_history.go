package models

import "time"

// DebtHistoryPoint es un punto mensual de la evolución de deudas (SPEC-105/106).
// Incluye la cantidad de deudas abiertas y el monto total adeudado al cierre
// del mes en la moneda de la serie.
type DebtHistoryPoint struct {
	Month             string  `json:"month"` // YYYY-MM
	OpenCount         int     `json:"open_count"`
	ClosedCount       int     `json:"closed_count"`
	OutstandingAmount float64 `json:"outstanding_amount"`
}

// DebtHistorySummary contiene los totales del período para una moneda.
type DebtHistorySummary struct {
	StartCount           int      `json:"start_count"`
	EndCount             int      `json:"end_count"`
	ChangeCount          int      `json:"change_count"`
	ChangePct            *float64 `json:"change_pct"`
	ClosedCount          int      `json:"closed_count"`
	PendingAmountCurrent float64  `json:"pending_amount_current"`
	StartAmount          float64  `json:"start_amount"`
	EndAmount            float64  `json:"end_amount"`
	ChangeAmount         float64  `json:"change_amount"`
	ChangePctAmount      *float64 `json:"change_pct_amount"`
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
