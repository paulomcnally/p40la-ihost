package models

import "time"

// SalaryPayment representa el pago mensual de un salario (SPEC-049).
// employer se resuelve con JOIN al listar, desde la tabla salaries.
type SalaryPayment struct {
	ID             int64      `json:"id"`
	SalaryID       int64      `json:"salary_id"`
	Employer       string     `json:"employer,omitempty"`
	Year           int        `json:"year"`
	Month          int        `json:"month"`
	Amount         float64    `json:"amount"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	ReceivedAmount *float64   `json:"received_amount,omitempty"`
	ReceivedAt     *time.Time `json:"received_at,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
