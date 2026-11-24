package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newBankingTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClient(t, server.URL)
}

func TestCreateBankAccountSendsPayload(t *testing.T) {
	var gotPath, gotSource string
	var got BankAccountInput
	client := newBankingTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSource = r.Header.Get(sourceHeader)
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":9,"bank_id":3,"alias":"Cuenta Demo","account_number":"demo-account-001","account_type":"savings","currency_id":1,"beneficiary":"Titular Demo","active":true}`))
	})

	account, err := client.CreateBankAccount(context.Background(), 3, BankAccountInput{
		Alias:         "Cuenta Demo",
		AccountNumber: "demo-account-001",
		AccountType:   "savings",
		CurrencyID:    1,
		Beneficiary:   "Titular Demo",
		Active:        true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if gotPath != "/api/banks/3/accounts" {
		t.Fatalf("path inesperado: %s", gotPath)
	}
	if gotSource != sourceMCP {
		t.Fatalf("header de origen inesperado: %q", gotSource)
	}
	if got.AccountNumber != "demo-account-001" || got.CurrencyID != 1 || !got.Active {
		t.Fatalf("payload inesperado: %+v", got)
	}
	if account.ID != 9 || account.BankID != 3 {
		t.Fatalf("respuesta inesperada: %+v", account)
	}
}

func TestUpdateCreditCardSendsPayload(t *testing.T) {
	var gotPath string
	var got CreditCardInput
	client := newBankingTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":7,"bank_id":1,"alias":"Tarjeta Demo","network":"visa","last4":"0000","currency_id":1,"limit_currency":"NIO","statement_day":1,"payment_due_day":1,"min_payment_pct":4,"active":true}`))
	})

	card, err := client.UpdateCreditCard(context.Background(), 7, CreditCardInput{
		Alias:         "Tarjeta Demo",
		Network:       "visa",
		Last4:         "0000",
		CurrencyID:    1,
		LimitCurrency: "NIO",
		StatementDay:  1,
		PaymentDueDay: 1,
		MinPaymentPct: 4,
		Active:        true,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if gotPath != "/api/credit-cards/7" {
		t.Fatalf("path inesperado: %s", gotPath)
	}
	if got.Alias != "Tarjeta Demo" || got.Network != "visa" || got.Last4 != "0000" {
		t.Fatalf("payload inesperado: %+v", got)
	}
	if card.ID != 7 {
		t.Fatalf("respuesta inesperada: %+v", card)
	}
}

func TestListCreditCardsDecodesArray(t *testing.T) {
	client := newBankingTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/banks/2/credit-cards" {
			t.Errorf("path inesperado: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"alias":"Tarjeta Demo","last4":"0000","active":true}]`))
	})

	cards, err := client.ListCreditCards(context.Background(), 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(cards) != 1 || cards[0].ID != 1 || cards[0].Last4 != "0000" {
		t.Fatalf("resultado inesperado: %+v", cards)
	}
}

func TestAPIErrorKeepsConflictCode(t *testing.T) {
	client := newBankingTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"duplicate_card","message":"Ya existe una tarjeta con ese alias en el banco"}`))
	})

	_, err := client.CreateCreditCard(context.Background(), 1, CreditCardInput{})
	if err == nil {
		t.Fatal("se esperaba error")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Status != http.StatusConflict || apiErr.Code != "duplicate_card" {
		t.Fatalf("error inesperado: %#v", err)
	}
}
