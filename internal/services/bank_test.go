package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func setupBankTest(t *testing.T) (*BankService, *sql.DB) {
	t.Helper()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	bankStorage := storage.NewBankStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	return NewBankService(bankStorage, currencyStorage), database
}

func currencyID(t *testing.T, database *sql.DB) int64 {
	t.Helper()
	var id int64
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = 'NIO'`).Scan(&id); err != nil {
		t.Fatalf("obtener moneda: %v", err)
	}
	return id
}

func validBank(name string) *models.Bank {
	return &models.Bank{Name: name, ShortCode: "BDEMO", Country: "Nicaragua", Notes: "Banco de prueba", Active: true}
}

func validBankAccount(bankID, currencyID int64) *models.BankAccount {
	return &models.BankAccount{
		BankID:        bankID,
		Alias:         "Cuenta principal",
		AccountNumber: "0000000001",
		AccountType:   models.BankAccountTypeSavings,
		CurrencyID:    currencyID,
		Beneficiary:   "Beneficiario Demo",
		Active:        true,
	}
}

func TestBankCRUD(t *testing.T) {
	service, _ := setupBankTest(t)
	ctx := context.Background()

	created, err := service.CreateBank(ctx, validBank("Banco Demo"))
	if err != nil {
		t.Fatalf("CreateBank: %v", err)
	}
	if created.ID == 0 || created.AccountCount != 0 {
		t.Fatalf("banco inesperado: %+v", created)
	}

	if _, err := service.CreateBank(ctx, validBank("banco demo")); !errors.Is(err, ErrBankNameExists) {
		t.Fatalf("se esperaba ErrBankNameExists, got %v", err)
	}

	update := validBank("Banco Demo Actualizado")
	update.ID = created.ID
	updated, err := service.UpdateBank(ctx, update)
	if err != nil {
		t.Fatalf("UpdateBank: %v", err)
	}
	if updated.Name != "Banco Demo Actualizado" {
		t.Fatalf("nombre inesperado: %s", updated.Name)
	}

	if err := service.DeleteBank(ctx, created.ID); err != nil {
		t.Fatalf("DeleteBank: %v", err)
	}
	if err := service.DeleteBank(ctx, created.ID); !errors.Is(err, ErrBankNotFound) {
		t.Fatalf("se esperaba ErrBankNotFound, got %v", err)
	}
}

func TestBankValidation(t *testing.T) {
	service, database := setupBankTest(t)
	ctx := context.Background()

	if _, err := service.CreateBank(ctx, &models.Bank{Name: "   "}); err == nil {
		t.Fatal("se esperaba error por nombre vacío")
	}

	bank, err := service.CreateBank(ctx, validBank("Banco Demo"))
	if err != nil {
		t.Fatalf("CreateBank: %v", err)
	}
	id := currencyID(t, database)
	account := validBankAccount(bank.ID, id)
	account.CurrencyID = 999999
	if _, err := service.CreateAccount(ctx, account); err == nil {
		t.Fatal("se esperaba error por moneda inexistente")
	}
}

func TestBankAccountsAndFormats(t *testing.T) {
	service, database := setupBankTest(t)
	ctx := context.Background()
	currency := currencyID(t, database)

	bank, err := service.CreateBank(ctx, validBank("Banco Demo"))
	if err != nil {
		t.Fatalf("CreateBank: %v", err)
	}

	account, err := service.CreateAccount(ctx, validBankAccount(bank.ID, currency))
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if account.CurrencyCode != "NIO" || account.BankName != "Banco Demo" {
		t.Fatalf("cuenta inesperada: %+v", account)
	}

	duplicate := validBankAccount(bank.ID, currency)
	if _, err := service.CreateAccount(ctx, duplicate); !errors.Is(err, ErrBankAccountDuplicate) {
		t.Fatalf("se esperaba ErrBankAccountDuplicate, got %v", err)
	}

	if err := service.DeleteBank(ctx, bank.ID); !errors.Is(err, ErrBankHasAccounts) {
		t.Fatalf("se esperaba ErrBankHasAccounts, got %v", err)
	}

	transfer, err := service.UpsertFormat(ctx, account.ID, models.BankTransferKindTransfer, map[string]string{
		"from":        "{account_number}",
		"to":          "00000000000001",
		"amount":      "{installment_amount}",
		"description": "Pago Lote Demo 13 - Cuota {installment_number}",
		"beneficiary": "{account_beneficiary}",
	})
	if err != nil {
		t.Fatalf("UpsertFormat transfer: %v", err)
	}
	if transfer.Fields["description"] != "Pago Lote Demo 13 - Cuota {installment_number}" {
		t.Fatalf("descripción inesperada: %+v", transfer.Fields)
	}

	ach, err := service.UpsertFormat(ctx, account.ID, models.BankTransferKindACH, map[string]string{
		"from":     "0000000000",
		"to":       "{account_number}",
		"bank":     "{bank_name}",
		"currency": "{account_currency}",
	})
	if err != nil {
		t.Fatalf("UpsertFormat ach: %v", err)
	}
	if ach.Kind != models.BankTransferKindACH {
		t.Fatalf("kind inesperado: %s", ach.Kind)
	}

	formats, err := service.ListFormats(ctx, account.ID)
	if err != nil {
		t.Fatalf("ListFormats: %v", err)
	}
	if len(formats) != 2 {
		t.Fatalf("se esperaban 2 formatos, got %d", len(formats))
	}

	if err := service.DeleteFormat(ctx, account.ID, models.BankTransferKindTransfer); err != nil {
		t.Fatalf("DeleteFormat: %v", err)
	}
	if err := service.DeleteFormat(ctx, account.ID, models.BankTransferKindTransfer); !errors.Is(err, ErrBankTransferFormatNotFound) {
		t.Fatalf("se esperaba ErrBankTransferFormatNotFound, got %v", err)
	}

	// Borrar la cuenta debe borrar también sus formatos.
	if err := service.DeleteAccount(ctx, account.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	var formatCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM bank_transfer_formats`).Scan(&formatCount); err != nil {
		t.Fatalf("contar formatos: %v", err)
	}
	if formatCount != 0 {
		t.Fatalf("se esperaban 0 formatos tras borrar la cuenta, got %d", formatCount)
	}
	if err := service.DeleteBank(ctx, bank.ID); err != nil {
		t.Fatalf("DeleteBank sin cuentas: %v", err)
	}
}

func TestBankFormatValidation(t *testing.T) {
	service, database := setupBankTest(t)
	ctx := context.Background()
	currency := currencyID(t, database)

	bank, _ := service.CreateBank(ctx, validBank("Banco Demo"))
	account, _ := service.CreateAccount(ctx, validBankAccount(bank.ID, currency))

	cases := []struct {
		name   string
		fields map[string]string
	}{
		{"llave inválida", map[string]string{"From": "x"}},
		{"variable desconocida", map[string]string{"amount": "{installment_numer}"}},
		{"llaves dobles", map[string]string{"amount": "{{installment_number}}"}},
		{"llave mal formada", map[string]string{"amount": "{installment_number"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.UpsertFormat(ctx, account.ID, models.BankTransferKindTransfer, tc.fields)
			var validation *BankValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("se esperaba BankValidationError, got %v", err)
			}
		})
	}

	if _, err := service.UpsertFormat(ctx, account.ID, "otro", map[string]string{"from": "x"}); err == nil {
		t.Fatal("se esperaba error por kind inválido")
	}
}

func TestBankVariablesCatalog(t *testing.T) {
	service, _ := setupBankTest(t)
	variables := map[string]bool{}
	for _, variable := range service.Variables() {
		variables[variable.Key] = true
		if strings.TrimSpace(variable.Label) == "" {
			t.Fatalf("variable sin etiqueta: %+v", variable)
		}
	}
	for _, key := range []string{"account_number", "account_beneficiary", "bank_name", "installment_number"} {
		if !variables[key] {
			t.Fatalf("falta la variable %s en el catálogo", key)
		}
	}
}
