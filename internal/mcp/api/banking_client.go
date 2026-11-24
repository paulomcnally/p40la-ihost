package api

import (
	"context"
	"fmt"
)

// ListBankRefs devuelve todos los bancos.
func (c *Client) ListBankRefs(ctx context.Context) ([]BankRef, error) {
	var banks []BankRef
	if err := c.doJSON(ctx, "GET", "/api/banks", nil, &banks, nil); err != nil {
		return nil, err
	}
	return banks, nil
}

// ListCurrencyRefs devuelve todas las monedas.
func (c *Client) ListCurrencyRefs(ctx context.Context) ([]CurrencyRef, error) {
	var currencies []CurrencyRef
	if err := c.doJSON(ctx, "GET", "/api/currencies", nil, &currencies, nil); err != nil {
		return nil, err
	}
	return currencies, nil
}

// ListBankAccounts lista las cuentas de un banco.
func (c *Client) ListBankAccounts(ctx context.Context, bankID int64) ([]BankAccount, error) {
	var accounts []BankAccount
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/banks/%d/accounts", bankID), nil, &accounts, nil); err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetBankAccount devuelve una cuenta por ID.
func (c *Client) GetBankAccount(ctx context.Context, id int64) (*BankAccount, error) {
	var account BankAccount
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/bank-accounts/%d", id), nil, &account, nil); err != nil {
		return nil, err
	}
	return &account, nil
}

// CreateBankAccount crea una cuenta en un banco.
func (c *Client) CreateBankAccount(ctx context.Context, bankID int64, input BankAccountInput) (*BankAccount, error) {
	var account BankAccount
	headers := map[string]string{sourceHeader: sourceMCP}
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/banks/%d/accounts", bankID), input, &account, headers); err != nil {
		return nil, err
	}
	return &account, nil
}

// UpdateBankAccount reemplaza una cuenta existente.
func (c *Client) UpdateBankAccount(ctx context.Context, id int64, input BankAccountInput) (*BankAccount, error) {
	var account BankAccount
	headers := map[string]string{sourceHeader: sourceMCP}
	if err := c.doJSON(ctx, "PUT", fmt.Sprintf("/api/bank-accounts/%d", id), input, &account, headers); err != nil {
		return nil, err
	}
	return &account, nil
}

// ListCreditCards lista las tarjetas de un banco.
func (c *Client) ListCreditCards(ctx context.Context, bankID int64) ([]CreditCard, error) {
	var cards []CreditCard
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/banks/%d/credit-cards", bankID), nil, &cards, nil); err != nil {
		return nil, err
	}
	return cards, nil
}

// GetCreditCard devuelve una tarjeta por ID.
func (c *Client) GetCreditCard(ctx context.Context, id int64) (*CreditCard, error) {
	var card CreditCard
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/credit-cards/%d", id), nil, &card, nil); err != nil {
		return nil, err
	}
	return &card, nil
}

// CreateCreditCard crea una tarjeta en un banco.
func (c *Client) CreateCreditCard(ctx context.Context, bankID int64, input CreditCardInput) (*CreditCard, error) {
	var card CreditCard
	headers := map[string]string{sourceHeader: sourceMCP}
	if err := c.doJSON(ctx, "POST", fmt.Sprintf("/api/banks/%d/credit-cards", bankID), input, &card, headers); err != nil {
		return nil, err
	}
	return &card, nil
}

// UpdateCreditCard reemplaza una tarjeta existente.
func (c *Client) UpdateCreditCard(ctx context.Context, id int64, input CreditCardInput) (*CreditCard, error) {
	var card CreditCard
	headers := map[string]string{sourceHeader: sourceMCP}
	if err := c.doJSON(ctx, "PUT", fmt.Sprintf("/api/credit-cards/%d", id), input, &card, headers); err != nil {
		return nil, err
	}
	return &card, nil
}
