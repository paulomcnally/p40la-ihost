package services

import (
	"testing"
	"time"
)

func TestDueDateForInstallment(t *testing.T) {
	start := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		k          int
		paymentDay int
		want       string
	}{
		{name: "primera cuota mes siguiente", k: 1, paymentDay: 15, want: "2026-10-15"},
		{name: "segunda cuota", k: 2, paymentDay: 15, want: "2026-11-15"},
		{name: "salto de año", k: 4, paymentDay: 15, want: "2027-01-15"},
		{name: "día 31 clamp en febrero", k: 5, paymentDay: 31, want: "2027-02-28"},
		{name: "día 31 clamp en abril", k: 7, paymentDay: 31, want: "2027-04-30"},
		{name: "día 31 clamp en bisiesto", k: 29, paymentDay: 31, want: "2029-02-28"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dueDateForInstallment(start, tt.k, tt.paymentDay)
			if got != tt.want {
				t.Errorf("dueDateForInstallment(%d, %d) = %q, want %q", tt.k, tt.paymentDay, got, tt.want)
			}
		})
	}
}

func TestComputeInstallmentAmount(t *testing.T) {
	tests := []struct {
		name         string
		total        float64
		installments int
		want         float64
	}{
		{name: "exacto", total: 12000, installments: 12, want: 1000},
		{name: "con decimales", total: 100, installments: 3, want: 33.33},
		{name: "cero cuotas", total: 100, installments: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeInstallmentAmount(tt.total, tt.installments); got != tt.want {
				t.Errorf("computeInstallmentAmount(%v, %d) = %v, want %v", tt.total, tt.installments, got, tt.want)
			}
		})
	}
}