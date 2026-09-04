package services

import (
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestRenderDebtDueContent(t *testing.T) {
	due := []models.DebtBill{
		{
			ID:                1,
			DebtID:            1,
			DebtDescription:   "Tarjeta de crédito",
			InstitutionName:   "BAC Credomatic",
			CurrencyCode:      "NIO",
			InstallmentNumber: 1,
			DueDate:           "2026-10-15",
			Amount:            1000,
			Status:            "pending",
		},
		{
			ID:                2,
			DebtID:            2,
			DebtDescription:   "Préstamo personal",
			InstitutionName:   "BANPRO",
			CurrencyCode:      "NIO",
			InstallmentNumber: 3,
			DueDate:           "2026-10-15",
			Amount:            500,
			Status:            "pending",
		},
	}

	html := renderDebtDueContent(due, DefaultCurrencyFormat())

	for _, want := range []string{
		"Tarjeta de crédito",
		"Préstamo personal",
		"Cuota #1",
		"Cuota #3",
		"Total del día",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("el email no contiene %q", want)
		}
	}
	if !strings.Contains(html, "1,500.00") && !strings.Contains(html, "1500.00") {
		t.Error("el total del día es incorrecto, esperado 1500.00")
	}
}

func TestPlural(t *testing.T) {
	if plural(1) != "" {
		t.Errorf("plural(1) = %q, esperado vacío", plural(1))
	}
	if plural(2) != "s" {
		t.Errorf("plural(2) = %q, esperado 's'", plural(2))
	}
}
