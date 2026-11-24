package banking

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

var last4Pattern = regexp.MustCompile(`^\d{4}$`)

var cardNetworks = map[string]bool{
	"visa":       true,
	"mastercard": true,
	"amex":       true,
	"diners":     true,
	"other":      true,
}

var cardCategories = map[string]bool{
	"":         true,
	"classic":  true,
	"gold":     true,
	"platinum": true,
	"black":    true,
	"business": true,
	"other":    true,
}

// CardSearchInput es la entrada de credit_cards_search.
type CardSearchInput struct {
	Query  *string `json:"query,omitempty" jsonschema:"Texto a buscar en alias, últimos 4 dígitos, titular, marca o banco (opcional)"`
	Bank   *string `json:"bank,omitempty" jsonschema:"Nombre o short_code del banco para filtrar (opcional)"`
	BankID *int64  `json:"bank_id,omitempty" jsonschema:"ID del banco para filtrar (opcional; tiene prioridad sobre bank)"`
	Active *bool   `json:"active,omitempty" jsonschema:"Filtrar por estado activo/inactivo (opcional; sin valor devuelve ambos)"`
	Limit  *int    `json:"limit,omitempty" jsonschema:"Máximo de resultados, 1 a 50 (default 10)"`
}

// CardItem es una tarjeta resumida para las tools de banca.
type CardItem struct {
	ID                    int64   `json:"id"`
	BankID                int64   `json:"bank_id"`
	BankName              string  `json:"bank_name,omitempty"`
	Alias                 string  `json:"alias"`
	Network               string  `json:"network"`
	Last4                 string  `json:"last4"`
	Holder                string  `json:"holder,omitempty"`
	CurrencyID            int64   `json:"currency_id"`
	CurrencyCode          string  `json:"currency_code,omitempty"`
	SecondaryCurrencyCode string  `json:"secondary_currency_code,omitempty"`
	CreditLimit           float64 `json:"credit_limit"`
	CreditLimitUSD        float64 `json:"credit_limit_usd"`
	LimitCurrency         string  `json:"limit_currency"`
	StatementDay          int     `json:"statement_day"`
	PaymentDueDay         int     `json:"payment_due_day"`
	Active                bool    `json:"active"`
}

// CardSearchOutput es el resultado de credit_cards_search.
type CardSearchOutput struct {
	Count     int        `json:"count"`
	Total     int        `json:"total"`
	Truncated bool       `json:"truncated"`
	Hint      string     `json:"hint,omitempty"`
	Items     []CardItem `json:"items"`
}

// CardCreateInput es la entrada de credit_cards_create.
type CardCreateInput struct {
	Bank                   *string  `json:"bank,omitempty" jsonschema:"Nombre o short_code del banco (obligatorio si no se envía bank_id)"`
	BankID                 *int64   `json:"bank_id,omitempty" jsonschema:"ID del banco (alternativa a bank)"`
	Alias                  string   `json:"alias" jsonschema:"Alias de la tarjeta (obligatorio)"`
	Network                string   `json:"network" jsonschema:"visa, mastercard, amex, diners u other (obligatorio)"`
	Last4                  string   `json:"last4" jsonschema:"Últimos 4 dígitos, exactamente 4 números (obligatorio)"`
	Currency               *string  `json:"currency,omitempty" jsonschema:"Código de moneda, ej. NIO o USD (obligatorio si no se envía currency_id)"`
	CurrencyID             *int64   `json:"currency_id,omitempty" jsonschema:"ID de la moneda (alternativa a currency)"`
	Holder                 *string  `json:"holder,omitempty" jsonschema:"Titular (opcional)"`
	SecondaryCurrency      *string  `json:"secondary_currency,omitempty" jsonschema:"Código de moneda secundaria (opcional)"`
	SecondaryCurrencyID    *int64   `json:"secondary_currency_id,omitempty" jsonschema:"ID de moneda secundaria (alternativa)"`
	Category               *string  `json:"category,omitempty" jsonschema:"classic, gold, platinum, black, business u other (opcional)"`
	LimitCurrency          *string  `json:"limit_currency,omitempty" jsonschema:"NIO (default) o USD"`
	ExchangeRate           *float64 `json:"exchange_rate,omitempty" jsonschema:"Tipo de cambio (default 0)"`
	CreditLimit            *float64 `json:"credit_limit,omitempty" jsonschema:"Límite de crédito (default 0)"`
	CreditLimitUSD         *float64 `json:"credit_limit_usd,omitempty" jsonschema:"Límite de crédito en USD (default 0)"`
	StatementDay           *int     `json:"statement_day,omitempty" jsonschema:"Día de corte, 1 a 31 (default 1)"`
	PaymentDueDay          *int     `json:"payment_due_day,omitempty" jsonschema:"Día límite de pago, 1 a 31 (default 1)"`
	BonusDueDay            *int     `json:"bonus_due_day,omitempty" jsonschema:"Día límite de bonificación, 0 a 31 (default 0)"`
	MinPaymentPct          *float64 `json:"min_payment_pct,omitempty" jsonschema:"Porcentaje de pago mínimo, 0 a 100 (default 4)"`
	InterestCurrentRate    *float64 `json:"interest_current_rate,omitempty" jsonschema:"Tasa de interés corriente (default 0)"`
	InterestLateRate       *float64 `json:"interest_late_rate,omitempty" jsonschema:"Tasa de interés moratorio (default 0)"`
	InterestCurrentRateUSD *float64 `json:"interest_current_rate_usd,omitempty" jsonschema:"Tasa corriente en USD (default 0)"`
	InterestLateRateUSD    *float64 `json:"interest_late_rate_usd,omitempty" jsonschema:"Tasa moratoria en USD (default 0)"`
	Color                  *string  `json:"color,omitempty" jsonschema:"Color de la tarjeta (default blue)"`
	Icon                   *string  `json:"icon,omitempty" jsonschema:"Ícono de la tarjeta (default credit)"`
	Notes                  *string  `json:"notes,omitempty" jsonschema:"Notas (opcional)"`
	Active                 *bool    `json:"active,omitempty" jsonschema:"Activa (default true)"`
}

// CardUpdateInput es la entrada de credit_cards_update.
type CardUpdateInput struct {
	ID                     int64    `json:"id" jsonschema:"ID de la tarjeta (obligatorio)"`
	Alias                  *string  `json:"alias,omitempty" jsonschema:"Nuevo alias (opcional)"`
	Network                *string  `json:"network,omitempty" jsonschema:"Nueva marca (opcional)"`
	Last4                  *string  `json:"last4,omitempty" jsonschema:"Nuevos últimos 4 dígitos (opcional)"`
	Currency               *string  `json:"currency,omitempty" jsonschema:"Nuevo código de moneda (opcional)"`
	CurrencyID             *int64   `json:"currency_id,omitempty" jsonschema:"Nuevo ID de moneda (opcional)"`
	Holder                 *string  `json:"holder,omitempty" jsonschema:"Nuevo titular (opcional)"`
	SecondaryCurrency      *string  `json:"secondary_currency,omitempty" jsonschema:"Nueva moneda secundaria; vacío la limpia (opcional)"`
	SecondaryCurrencyID    *int64   `json:"secondary_currency_id,omitempty" jsonschema:"Nuevo ID de moneda secundaria; 0 la limpia (opcional)"`
	Category               *string  `json:"category,omitempty" jsonschema:"Nueva categoría (opcional)"`
	LimitCurrency          *string  `json:"limit_currency,omitempty" jsonschema:"NIO o USD (opcional)"`
	ExchangeRate           *float64 `json:"exchange_rate,omitempty" jsonschema:"Nuevo tipo de cambio (opcional)"`
	CreditLimit            *float64 `json:"credit_limit,omitempty" jsonschema:"Nuevo límite de crédito (opcional)"`
	CreditLimitUSD         *float64 `json:"credit_limit_usd,omitempty" jsonschema:"Nuevo límite en USD (opcional)"`
	StatementDay           *int     `json:"statement_day,omitempty" jsonschema:"Nuevo día de corte (opcional)"`
	PaymentDueDay          *int     `json:"payment_due_day,omitempty" jsonschema:"Nuevo día de pago (opcional)"`
	BonusDueDay            *int     `json:"bonus_due_day,omitempty" jsonschema:"Nuevo día de bonificación (opcional)"`
	MinPaymentPct          *float64 `json:"min_payment_pct,omitempty" jsonschema:"Nuevo porcentaje mínimo (opcional)"`
	InterestCurrentRate    *float64 `json:"interest_current_rate,omitempty" jsonschema:"Nueva tasa corriente (opcional)"`
	InterestLateRate       *float64 `json:"interest_late_rate,omitempty" jsonschema:"Nueva tasa moratoria (opcional)"`
	InterestCurrentRateUSD *float64 `json:"interest_current_rate_usd,omitempty" jsonschema:"Nueva tasa corriente USD (opcional)"`
	InterestLateRateUSD    *float64 `json:"interest_late_rate_usd,omitempty" jsonschema:"Nueva tasa moratoria USD (opcional)"`
	Color                  *string  `json:"color,omitempty" jsonschema:"Nuevo color (opcional)"`
	Icon                   *string  `json:"icon,omitempty" jsonschema:"Nuevo ícono (opcional)"`
	Notes                  *string  `json:"notes,omitempty" jsonschema:"Nuevas notas; vacío las limpia (opcional)"`
	Active                 *bool    `json:"active,omitempty" jsonschema:"Nuevo estado (opcional)"`
	Bank                   *string  `json:"bank,omitempty" jsonschema:"No permitido: el banco de una tarjeta no se cambia"`
	BankID                 *int64   `json:"bank_id,omitempty" jsonschema:"No permitido: el banco de una tarjeta no se cambia"`
}

func (t *Toolset) cardsSearch(ctx context.Context, _ *mcp.CallToolRequest, input CardSearchInput) (*mcp.CallToolResult, any, error) {
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
		item  CardItem
		score int
	}
	var matches []scored
	for _, bank := range banks {
		cards, err := t.client.ListCreditCards(ctx, bank.ID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return apiError(err)
		}
		for _, card := range cards {
			if input.Active != nil && card.Active != *input.Active {
				continue
			}
			score := matchScore([]string{
				card.Alias,
				card.Last4,
				card.Holder,
				card.Network,
				card.BankName,
				bank.Name,
			}, query)
			if score < 0 {
				continue
			}
			item := cardItem(card)
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

	outputValue := CardSearchOutput{
		Total: len(matches),
		Items: make([]CardItem, 0, min(len(matches), limit)),
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
		outputValue.Hint = "No hay tarjetas que coincidan. Probá con otro texto, revisá active o creá una con credit_cards_create."
	}
	return jsonResult(outputValue)
}

func (t *Toolset) cardsCreate(ctx context.Context, _ *mcp.CallToolRequest, input CardCreateInput) (*mcp.CallToolResult, any, error) {
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

	var secondaryID *int64
	if input.SecondaryCurrencyID != nil && *input.SecondaryCurrencyID > 0 {
		resolved, err := t.resolveCurrency(ctx, "", *input.SecondaryCurrencyID)
		if err != nil {
			return lookupFailure(err)
		}
		secondaryID = &resolved.ID
	} else if input.SecondaryCurrency != nil && strings.TrimSpace(*input.SecondaryCurrency) != "" {
		resolved, err := t.resolveCurrency(ctx, *input.SecondaryCurrency, 0)
		if err != nil {
			return lookupFailure(err)
		}
		secondaryID = &resolved.ID
	}

	network := strings.ToLower(strings.TrimSpace(input.Network))
	last4 := strings.TrimSpace(input.Last4)
	alias := strings.TrimSpace(input.Alias)
	if alias == "" {
		return toolError("invalid_value", "alias es obligatorio.", "")
	}
	if !cardNetworks[network] {
		return toolError("invalid_value", "network debe ser visa, mastercard, amex, diners u other.", "")
	}
	if !last4Pattern.MatchString(last4) {
		return toolError("invalid_value", "last4 debe ser exactamente 4 números.", "")
	}

	payload := api.CreditCardInput{
		Alias:                  alias,
		Network:                network,
		Last4:                  last4,
		Holder:                 stringValue(input.Holder),
		CurrencyID:             currency.ID,
		SecondaryCurrencyID:    secondaryID,
		Category:               strings.ToLower(strings.TrimSpace(stringValue(input.Category))),
		LimitCurrency:          strings.ToUpper(strings.TrimSpace(stringValue(input.LimitCurrency))),
		ExchangeRate:           floatValue(input.ExchangeRate),
		CreditLimit:            floatValue(input.CreditLimit),
		CreditLimitUSD:         floatValue(input.CreditLimitUSD),
		StatementDay:           intValue(input.StatementDay, 1),
		PaymentDueDay:          intValue(input.PaymentDueDay, 1),
		BonusDueDay:            intValue(input.BonusDueDay, 0),
		MinPaymentPct:          floatValueDefault(input.MinPaymentPct, 4),
		InterestCurrentRate:    floatValue(input.InterestCurrentRate),
		InterestLateRate:       floatValue(input.InterestLateRate),
		InterestCurrentRateUSD: floatValue(input.InterestCurrentRateUSD),
		InterestLateRateUSD:    floatValue(input.InterestLateRateUSD),
		Color:                  strings.ToLower(strings.TrimSpace(stringValue(input.Color))),
		Icon:                   strings.TrimSpace(stringValue(input.Icon)),
		Active:                 boolValue(input.Active, true),
		Notes:                  strings.TrimSpace(stringValue(input.Notes)),
	}
	if payload.LimitCurrency == "" {
		payload.LimitCurrency = "NIO"
	}
	if payload.Color == "" {
		payload.Color = "blue"
	}
	if payload.Icon == "" {
		payload.Icon = "credit"
	}
	if result, output, ok := validateCardPayload(payload); !ok {
		return result, output, nil
	}

	cards, err := t.client.ListCreditCards(ctx, bank.ID)
	if err != nil {
		return apiError(err)
	}
	if duplicate := findCard(cards, alias, 0); duplicate != nil {
		return toolError("already_exists",
			"Ya existe una tarjeta con ese alias en "+bank.Name+".",
			"Usá credit_cards_update con id="+itoa(duplicate.ID)+" o cambiá alias.")
	}

	created, err := t.client.CreateCreditCard(ctx, bank.ID, payload)
	if err != nil {
		return apiError(err)
	}
	item := cardItem(*created)
	if item.BankName == "" {
		item.BankName = bank.Name
	}
	if item.CurrencyCode == "" {
		item.CurrencyCode = currency.Code
	}
	return jsonResult(item)
}

func (t *Toolset) cardsUpdate(ctx context.Context, _ *mcp.CallToolRequest, input CardUpdateInput) (*mcp.CallToolResult, any, error) {
	if ok, result, output := t.writeEnabled(); !ok {
		return result, output, nil
	}
	if input.ID <= 0 {
		return toolError("invalid_value", "id debe ser mayor que cero.", "")
	}
	if input.Bank != nil || input.BankID != nil {
		return toolError("invalid_value", "El banco de una tarjeta no se cambia.",
			"Creá una tarjeta nueva en el otro banco con credit_cards_create.")
	}

	current, err := t.client.GetCreditCard(ctx, input.ID)
	if err != nil {
		return apiError(err)
	}

	payload := api.CreditCardInput{
		Alias:                  current.Alias,
		Network:                current.Network,
		Last4:                  current.Last4,
		Holder:                 current.Holder,
		CurrencyID:             current.CurrencyID,
		SecondaryCurrencyID:    current.SecondaryCurrencyID,
		Category:               current.Category,
		LimitCurrency:          current.LimitCurrency,
		ExchangeRate:           current.ExchangeRate,
		CreditLimit:            current.CreditLimit,
		CreditLimitUSD:         current.CreditLimitUSD,
		StatementDay:           current.StatementDay,
		PaymentDueDay:          current.PaymentDueDay,
		BonusDueDay:            current.BonusDueDay,
		MinPaymentPct:          current.MinPaymentPct,
		InterestCurrentRate:    current.InterestCurrentRate,
		InterestLateRate:       current.InterestLateRate,
		InterestCurrentRateUSD: current.InterestCurrentRateUSD,
		InterestLateRateUSD:    current.InterestLateRateUSD,
		Color:                  current.Color,
		Icon:                   current.Icon,
		Active:                 current.Active,
		Notes:                  current.Notes,
	}

	if input.Alias != nil {
		payload.Alias = strings.TrimSpace(*input.Alias)
	}
	if input.Network != nil {
		payload.Network = strings.ToLower(strings.TrimSpace(*input.Network))
	}
	if input.Last4 != nil {
		payload.Last4 = strings.TrimSpace(*input.Last4)
	}
	if input.Holder != nil {
		payload.Holder = strings.TrimSpace(*input.Holder)
	}
	if input.Category != nil {
		payload.Category = strings.ToLower(strings.TrimSpace(*input.Category))
	}
	if input.LimitCurrency != nil {
		payload.LimitCurrency = strings.ToUpper(strings.TrimSpace(*input.LimitCurrency))
	}
	if input.Color != nil {
		payload.Color = strings.ToLower(strings.TrimSpace(*input.Color))
	}
	if input.Icon != nil {
		payload.Icon = strings.TrimSpace(*input.Icon)
	}
	if input.Notes != nil {
		payload.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.Active != nil {
		payload.Active = *input.Active
	}
	if input.ExchangeRate != nil {
		payload.ExchangeRate = *input.ExchangeRate
	}
	if input.CreditLimit != nil {
		payload.CreditLimit = *input.CreditLimit
	}
	if input.CreditLimitUSD != nil {
		payload.CreditLimitUSD = *input.CreditLimitUSD
	}
	if input.StatementDay != nil {
		payload.StatementDay = *input.StatementDay
	}
	if input.PaymentDueDay != nil {
		payload.PaymentDueDay = *input.PaymentDueDay
	}
	if input.BonusDueDay != nil {
		payload.BonusDueDay = *input.BonusDueDay
	}
	if input.MinPaymentPct != nil {
		payload.MinPaymentPct = *input.MinPaymentPct
	}
	if input.InterestCurrentRate != nil {
		payload.InterestCurrentRate = *input.InterestCurrentRate
	}
	if input.InterestLateRate != nil {
		payload.InterestLateRate = *input.InterestLateRate
	}
	if input.InterestCurrentRateUSD != nil {
		payload.InterestCurrentRateUSD = *input.InterestCurrentRateUSD
	}
	if input.InterestLateRateUSD != nil {
		payload.InterestLateRateUSD = *input.InterestLateRateUSD
	}

	if input.CurrencyID != nil {
		if *input.CurrencyID > 0 {
			resolved, err := t.resolveCurrency(ctx, "", *input.CurrencyID)
			if err != nil {
				return lookupFailure(err)
			}
			payload.CurrencyID = resolved.ID
		}
	} else if input.Currency != nil && strings.TrimSpace(*input.Currency) != "" {
		resolved, err := t.resolveCurrency(ctx, *input.Currency, 0)
		if err != nil {
			return lookupFailure(err)
		}
		payload.CurrencyID = resolved.ID
	}

	switch {
	case input.SecondaryCurrencyID != nil && *input.SecondaryCurrencyID > 0:
		resolved, err := t.resolveCurrency(ctx, "", *input.SecondaryCurrencyID)
		if err != nil {
			return lookupFailure(err)
		}
		payload.SecondaryCurrencyID = &resolved.ID
	case input.SecondaryCurrencyID != nil:
		payload.SecondaryCurrencyID = nil
	case input.SecondaryCurrency != nil:
		trimmed := strings.TrimSpace(*input.SecondaryCurrency)
		if trimmed == "" {
			payload.SecondaryCurrencyID = nil
		} else {
			resolved, err := t.resolveCurrency(ctx, trimmed, 0)
			if err != nil {
				return lookupFailure(err)
			}
			payload.SecondaryCurrencyID = &resolved.ID
		}
	}

	if result, output, ok := validateCardPayload(payload); !ok {
		return result, output, nil
	}

	cards, err := t.client.ListCreditCards(ctx, current.BankID)
	if err != nil {
		return apiError(err)
	}
	if duplicate := findCard(cards, payload.Alias, current.ID); duplicate != nil {
		return toolError("already_exists",
			"Ya existe otra tarjeta con ese alias en el banco.",
			"Usá credit_cards_update con id="+itoa(duplicate.ID)+" o cambiá alias.")
	}

	updated, err := t.client.UpdateCreditCard(ctx, input.ID, payload)
	if err != nil {
		return apiError(err)
	}
	item := cardItem(*updated)
	if item.BankName == "" {
		item.BankName = current.BankName
	}
	if item.CurrencyCode == "" {
		item.CurrencyCode = current.CurrencyCode
	}
	return jsonResult(item)
}

func validateCardPayload(payload api.CreditCardInput) (*mcp.CallToolResult, any, bool) {
	if payload.Alias == "" {
		result, output, _ := toolError("invalid_value", "alias no puede quedar vacío.", "")
		return result, output, false
	}
	if !cardNetworks[payload.Network] {
		result, output, _ := toolError("invalid_value", "network debe ser visa, mastercard, amex, diners u other.", "")
		return result, output, false
	}
	if !last4Pattern.MatchString(payload.Last4) {
		result, output, _ := toolError("invalid_value", "last4 debe ser exactamente 4 números.", "")
		return result, output, false
	}
	if payload.StatementDay < 1 || payload.StatementDay > 31 {
		result, output, _ := toolError("invalid_value", "statement_day debe estar entre 1 y 31.", "")
		return result, output, false
	}
	if payload.PaymentDueDay < 1 || payload.PaymentDueDay > 31 {
		result, output, _ := toolError("invalid_value", "payment_due_day debe estar entre 1 y 31.", "")
		return result, output, false
	}
	if payload.BonusDueDay < 0 || payload.BonusDueDay > 31 {
		result, output, _ := toolError("invalid_value", "bonus_due_day debe estar entre 0 y 31.", "")
		return result, output, false
	}
	if payload.BonusDueDay > 0 && payload.BonusDueDay == payload.PaymentDueDay {
		result, output, _ := toolError("invalid_value", "bonus_due_day debe ser distinto de payment_due_day.", "")
		return result, output, false
	}
	if payload.LimitCurrency != "NIO" && payload.LimitCurrency != "USD" {
		result, output, _ := toolError("invalid_value", "limit_currency debe ser NIO o USD.", "")
		return result, output, false
	}
	if !cardCategories[payload.Category] {
		result, output, _ := toolError("invalid_value", "category debe ser classic, gold, platinum, black, business u other.", "")
		return result, output, false
	}
	if payload.MinPaymentPct < 0 || payload.MinPaymentPct > 100 {
		result, output, _ := toolError("invalid_value", "min_payment_pct debe estar entre 0 y 100.", "")
		return result, output, false
	}
	for _, rate := range []float64{
		payload.ExchangeRate,
		payload.CreditLimit,
		payload.CreditLimitUSD,
		payload.InterestCurrentRate,
		payload.InterestLateRate,
		payload.InterestCurrentRateUSD,
		payload.InterestLateRateUSD,
	} {
		if rate < 0 {
			result, output, _ := toolError("invalid_value", "Los montos y tasas no pueden ser negativos.", "")
			return result, output, false
		}
	}
	return nil, nil, true
}

func cardItem(card api.CreditCard) CardItem {
	return CardItem{
		ID:                    card.ID,
		BankID:                card.BankID,
		BankName:              card.BankName,
		Alias:                 card.Alias,
		Network:               card.Network,
		Last4:                 card.Last4,
		Holder:                card.Holder,
		CurrencyID:            card.CurrencyID,
		CurrencyCode:          card.CurrencyCode,
		SecondaryCurrencyCode: card.SecondaryCurrencyCode,
		CreditLimit:           card.CreditLimit,
		CreditLimitUSD:        card.CreditLimitUSD,
		LimitCurrency:         card.LimitCurrency,
		StatementDay:          card.StatementDay,
		PaymentDueDay:         card.PaymentDueDay,
		Active:                card.Active,
	}
}

func findCard(cards []api.CreditCard, alias string, excludeID int64) *api.CreditCard {
	target := fold(alias)
	for i := range cards {
		if cards[i].ID == excludeID {
			continue
		}
		if fold(cards[i].Alias) == target {
			return &cards[i]
		}
	}
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func floatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func floatValueDefault(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}
