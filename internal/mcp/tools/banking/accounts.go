package banking

import (
	"context"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// AccountSearchInput es la entrada de bank_accounts_search.
type AccountSearchInput struct {
	Query  *string `json:"query,omitempty" jsonschema:"Texto a buscar en alias, número de cuenta, beneficiario o banco (opcional)"`
	Bank   *string `json:"bank,omitempty" jsonschema:"Nombre o short_code del banco para filtrar (opcional)"`
	BankID *int64  `json:"bank_id,omitempty" jsonschema:"ID del banco para filtrar (opcional; tiene prioridad sobre bank)"`
	Active *bool   `json:"active,omitempty" jsonschema:"Filtrar por estado activo/inactivo (opcional; sin valor devuelve ambos)"`
	Limit  *int    `json:"limit,omitempty" jsonschema:"Máximo de resultados, 1 a 50 (default 10)"`
}

// AccountItem es una cuenta resumida para las tools de banca.
type AccountItem struct {
	ID            int64  `json:"id"`
	BankID        int64  `json:"bank_id"`
	BankName      string `json:"bank_name,omitempty"`
	Alias         string `json:"alias"`
	AccountNumber string `json:"account_number"`
	IBAN          string `json:"iban,omitempty"`
	AccountType   string `json:"account_type"`
	CurrencyID    int64  `json:"currency_id"`
	CurrencyCode  string `json:"currency_code,omitempty"`
	Beneficiary   string `json:"beneficiary"`
	Notes         string `json:"notes,omitempty"`
	Active        bool   `json:"active"`
}

// AccountSearchOutput es el resultado de bank_accounts_search.
type AccountSearchOutput struct {
	Count     int           `json:"count"`
	Total     int           `json:"total"`
	Truncated bool          `json:"truncated"`
	Hint      string        `json:"hint,omitempty"`
	Items     []AccountItem `json:"items"`
}

// AccountCreateInput es la entrada de bank_accounts_create.
type AccountCreateInput struct {
	Bank          *string `json:"bank,omitempty" jsonschema:"Nombre o short_code del banco (obligatorio si no se envía bank_id)"`
	BankID        *int64  `json:"bank_id,omitempty" jsonschema:"ID del banco (alternativa a bank)"`
	Alias         string  `json:"alias" jsonschema:"Alias de la cuenta (obligatorio)"`
	AccountNumber string  `json:"account_number" jsonschema:"Número de cuenta (obligatorio)"`
	IBAN          *string `json:"iban,omitempty" jsonschema:"IBAN de la cuenta (opcional, máximo 34 caracteres)"`
	Currency      *string `json:"currency,omitempty" jsonschema:"Código de moneda, ej. NIO o USD (obligatorio si no se envía currency_id)"`
	CurrencyID    *int64  `json:"currency_id,omitempty" jsonschema:"ID de la moneda (alternativa a currency)"`
	Beneficiary   string  `json:"beneficiary" jsonschema:"Beneficiario de la cuenta (obligatorio)"`
	AccountType   *string `json:"account_type,omitempty" jsonschema:"savings (default) o checking"`
	Notes         *string `json:"notes,omitempty" jsonschema:"Notas (opcional)"`
	Active        *bool   `json:"active,omitempty" jsonschema:"Activa (default true)"`
}

// AccountUpdateInput es la entrada de bank_accounts_update.
type AccountUpdateInput struct {
	ID            int64   `json:"id" jsonschema:"ID de la cuenta (obligatorio)"`
	Alias         *string `json:"alias,omitempty" jsonschema:"Nuevo alias (opcional)"`
	AccountNumber *string `json:"account_number,omitempty" jsonschema:"Nuevo número de cuenta (opcional)"`
	IBAN          *string `json:"iban,omitempty" jsonschema:"Nuevo IBAN; vacío lo limpia (opcional, máximo 34 caracteres)"`
	Currency      *string `json:"currency,omitempty" jsonschema:"Nuevo código de moneda (opcional)"`
	CurrencyID    *int64  `json:"currency_id,omitempty" jsonschema:"Nuevo ID de moneda (opcional)"`
	Beneficiary   *string `json:"beneficiary,omitempty" jsonschema:"Nuevo beneficiario (opcional)"`
	AccountType   *string `json:"account_type,omitempty" jsonschema:"savings o checking (opcional)"`
	Notes         *string `json:"notes,omitempty" jsonschema:"Nuevas notas; vacío las limpia (opcional)"`
	Active        *bool   `json:"active,omitempty" jsonschema:"Nuevo estado (opcional)"`
	Bank          *string `json:"bank,omitempty" jsonschema:"No permitido: el banco de una cuenta no se cambia"`
	BankID        *int64  `json:"bank_id,omitempty" jsonschema:"No permitido: el banco de una cuenta no se cambia"`
}

func (t *Toolset) accountsSearch(ctx context.Context, _ *mcp.CallToolRequest, input AccountSearchInput) (*mcp.CallToolResult, any, error) {
	limit, result, output := resolveLimit(input.Limit)
	if result != nil {
		return result, output, nil
	}

	var banks []api.BankRef
	if (input.BankID != nil && *input.BankID > 0) || (input.Bank != nil && strings.TrimSpace(*input.Bank) != "") {
		var bankID int64
		var bankName string
		if input.BankID != nil {
			bankID = *input.BankID
		}
		if input.Bank != nil {
			bankName = *input.Bank
		}
		bank, err := t.resolveBank(ctx, bankName, bankID)
		if err != nil {
			return lookupFailure(err)
		}
		banks = []api.BankRef{*bank}
	} else {
		list, err := t.client.ListBankRefs(ctx)
		if err != nil {
			return apiError(err)
		}
		banks = list
	}

	query := ""
	if input.Query != nil {
		query = fold(*input.Query)
	}

	type scored struct {
		item  AccountItem
		score int
	}
	var matches []scored
	for _, bank := range banks {
		accounts, err := t.client.ListBankAccounts(ctx, bank.ID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return apiError(err)
		}
		for _, account := range accounts {
			if input.Active != nil && account.Active != *input.Active {
				continue
			}
			score := matchScore([]string{
				account.Alias,
				account.AccountNumber,
				account.IBAN,
				account.Beneficiary,
				account.BankName,
				bank.Name,
			}, query)
			if score < 0 {
				continue
			}
			item := accountItem(account)
			if item.BankName == "" {
				item.BankName = bank.Name
			}
			matches = append(matches, scored{item: item, score: score})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		left := fold(matches[i].item.Alias)
		right := fold(matches[j].item.Alias)
		if left != right {
			return left < right
		}
		return matches[i].item.ID < matches[j].item.ID
	})

	outputValue := AccountSearchOutput{
		Total: len(matches),
		Items: make([]AccountItem, 0, min(len(matches), limit)),
	}
	for _, match := range matches {
		if len(outputValue.Items) >= limit {
			outputValue.Truncated = true
			break
		}
		outputValue.Items = append(outputValue.Items, match.item)
	}
	outputValue.Count = len(outputValue.Items)
	if outputValue.Count == 0 {
		outputValue.Hint = "No hay cuentas que coincidan. Probá con otro texto, revisá active o creá una con bank_accounts_create."
	}
	return jsonResult(outputValue)
}

func (t *Toolset) accountsCreate(ctx context.Context, _ *mcp.CallToolRequest, input AccountCreateInput) (*mcp.CallToolResult, any, error) {
	if ok, result, output := t.writeEnabled(); !ok {
		return result, output, nil
	}

	bankID := int64(0)
	if input.BankID != nil {
		bankID = *input.BankID
	}
	bank, err := t.resolveBank(ctx, stringValue(input.Bank), bankID)
	if err != nil {
		return lookupFailure(err)
	}
	currencyID := int64(0)
	if input.CurrencyID != nil {
		currencyID = *input.CurrencyID
	}
	currency, err := t.resolveCurrency(ctx, stringValue(input.Currency), currencyID)
	if err != nil {
		return lookupFailure(err)
	}

	alias := strings.TrimSpace(input.Alias)
	accountNumber := strings.TrimSpace(input.AccountNumber)
	beneficiary := strings.TrimSpace(input.Beneficiary)
	accountType := "savings"
	if input.AccountType != nil && strings.TrimSpace(*input.AccountType) != "" {
		accountType = strings.ToLower(strings.TrimSpace(*input.AccountType))
	}
	if alias == "" {
		return toolError("invalid_value", "alias es obligatorio.", "")
	}
	if accountNumber == "" {
		return toolError("invalid_value", "account_number es obligatorio.", "")
	}
	if beneficiary == "" {
		return toolError("invalid_value", "beneficiary es obligatorio.", "")
	}
	if accountType != "savings" && accountType != "checking" {
		return toolError("invalid_value", "account_type debe ser savings o checking.", "")
	}

	accounts, err := t.client.ListBankAccounts(ctx, bank.ID)
	if err != nil {
		return apiError(err)
	}
	if duplicate := findAccount(accounts, accountNumber, 0); duplicate != nil {
		return toolError("already_exists",
			"Ya existe una cuenta con ese número en "+bank.Name+".",
			"Usá bank_accounts_update con id="+itoa(duplicate.ID)+" o cambiá account_number.")
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}
	notes := ""
	if input.Notes != nil {
		notes = strings.TrimSpace(*input.Notes)
	}
	iban := strings.TrimSpace(stringValue(input.IBAN))
	created, err := t.client.CreateBankAccount(ctx, bank.ID, api.BankAccountInput{
		Alias:         alias,
		AccountNumber: accountNumber,
		IBAN:          iban,
		AccountType:   accountType,
		CurrencyID:    currency.ID,
		Beneficiary:   beneficiary,
		Notes:         notes,
		Active:        active,
	})
	if err != nil {
		return apiError(err)
	}
	item := accountItem(*created)
	if item.BankName == "" {
		item.BankName = bank.Name
	}
	if item.CurrencyCode == "" {
		item.CurrencyCode = currency.Code
	}
	return jsonResult(item)
}

func (t *Toolset) accountsUpdate(ctx context.Context, _ *mcp.CallToolRequest, input AccountUpdateInput) (*mcp.CallToolResult, any, error) {
	if ok, result, output := t.writeEnabled(); !ok {
		return result, output, nil
	}
	if input.ID <= 0 {
		return toolError("invalid_value", "id debe ser mayor que cero.", "")
	}
	if input.Bank != nil || input.BankID != nil {
		return toolError("invalid_value", "El banco de una cuenta no se cambia.",
			"Creá una cuenta nueva en el otro banco con bank_accounts_create.")
	}

	current, err := t.client.GetBankAccount(ctx, input.ID)
	if err != nil {
		return apiError(err)
	}

	alias := current.Alias
	accountNumber := current.AccountNumber
	iban := current.IBAN
	beneficiary := current.Beneficiary
	accountType := current.AccountType
	currencyID := current.CurrencyID
	notes := current.Notes
	active := current.Active

	if input.Alias != nil {
		alias = strings.TrimSpace(*input.Alias)
	}
	if input.AccountNumber != nil {
		accountNumber = strings.TrimSpace(*input.AccountNumber)
	}
	if input.IBAN != nil {
		iban = strings.TrimSpace(*input.IBAN)
	}
	if input.Beneficiary != nil {
		beneficiary = strings.TrimSpace(*input.Beneficiary)
	}
	if input.AccountType != nil && strings.TrimSpace(*input.AccountType) != "" {
		accountType = strings.ToLower(strings.TrimSpace(*input.AccountType))
	}
	if input.Notes != nil {
		notes = strings.TrimSpace(*input.Notes)
	}
	if input.Active != nil {
		active = *input.Active
	}
	currencyCode := ""
	if input.CurrencyID != nil && *input.CurrencyID > 0 {
		currencyID = *input.CurrencyID
		currency, err := t.resolveCurrency(ctx, "", currencyID)
		if err != nil {
			return lookupFailure(err)
		}
		currencyCode = currency.Code
	} else if input.Currency != nil && strings.TrimSpace(*input.Currency) != "" {
		currency, err := t.resolveCurrency(ctx, *input.Currency, 0)
		if err != nil {
			return lookupFailure(err)
		}
		currencyID = currency.ID
		currencyCode = currency.Code
	}

	if alias == "" {
		return toolError("invalid_value", "alias no puede quedar vacío.", "")
	}
	if accountNumber == "" {
		return toolError("invalid_value", "account_number no puede quedar vacío.", "")
	}
	if beneficiary == "" {
		return toolError("invalid_value", "beneficiary no puede quedar vacío.", "")
	}
	if accountType != "savings" && accountType != "checking" {
		return toolError("invalid_value", "account_type debe ser savings o checking.", "")
	}

	accounts, err := t.client.ListBankAccounts(ctx, current.BankID)
	if err != nil {
		return apiError(err)
	}
	if duplicate := findAccount(accounts, accountNumber, current.ID); duplicate != nil {
		return toolError("already_exists",
			"Ya existe otra cuenta con ese número en el banco.",
			"Usá bank_accounts_update con id="+itoa(duplicate.ID)+" o cambiá account_number.")
	}

	updated, err := t.client.UpdateBankAccount(ctx, input.ID, api.BankAccountInput{
		Alias:         alias,
		AccountNumber: accountNumber,
		IBAN:          iban,
		AccountType:   accountType,
		CurrencyID:    currencyID,
		Beneficiary:   beneficiary,
		Notes:         notes,
		Active:        active,
	})
	if err != nil {
		return apiError(err)
	}
	item := accountItem(*updated)
	if item.BankName == "" {
		item.BankName = current.BankName
	}
	if item.CurrencyCode == "" {
		item.CurrencyCode = currencyCode
	}
	if item.CurrencyCode == "" {
		item.CurrencyCode = current.CurrencyCode
	}
	return jsonResult(item)
}

func accountItem(account api.BankAccount) AccountItem {
	return AccountItem{
		ID:            account.ID,
		BankID:        account.BankID,
		BankName:      account.BankName,
		Alias:         account.Alias,
		AccountNumber: account.AccountNumber,
		IBAN:          account.IBAN,
		AccountType:   account.AccountType,
		CurrencyID:    account.CurrencyID,
		CurrencyCode:  account.CurrencyCode,
		Beneficiary:   account.Beneficiary,
		Notes:         account.Notes,
		Active:        account.Active,
	}
}

func findAccount(accounts []api.BankAccount, number string, excludeID int64) *api.BankAccount {
	target := strings.TrimSpace(number)
	for i := range accounts {
		if accounts[i].ID == excludeID {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(accounts[i].AccountNumber), target) {
			return &accounts[i]
		}
	}
	return nil
}
