package banking

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

// TestLiveAccountsAndCards corre contra un server real con datos sintéticos.
// Se saltea salvo que P40LA_LIVE_URL y P40LA_LIVE_API_KEY estén definidos.
//
//	DATA_DIR=/tmp/p40la-live PORT=8099 ./scripts/check-server.sh
//	P40LA_LIVE_URL=http://localhost:8099 P40LA_LIVE_API_KEY=... go test ./internal/mcp/tools/banking -run TestLive -v
func TestLiveAccountsAndCards(t *testing.T) {
	baseURL := os.Getenv("P40LA_LIVE_URL")
	apiKey := os.Getenv("P40LA_LIVE_API_KEY")
	if baseURL == "" || apiKey == "" {
		t.Skip("P40LA_LIVE_URL y P40LA_LIVE_API_KEY no definidos")
	}
	client, err := api.New(config.Config{BaseURL: baseURL, APIKey: apiKey, Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	toolset := New(client, config.Config{AllowWrites: true, Timezone: "UTC"})
	ctx := context.Background()
	suffix := time.Now().Format("150405")

	accountAlias := "Cuenta Demo " + suffix
	accountNumber := "demo-account-" + suffix
	_, created, err := toolset.accountsCreate(ctx, nil, AccountCreateInput{
		Bank:          strPtr("Banco Demo"),
		Alias:         accountAlias,
		AccountNumber: accountNumber,
		Currency:      strPtr("NIO"),
		Beneficiary:   "Titular Demo",
	})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}
	account := created.(AccountItem)
	if account.ID <= 0 || account.BankName != "Banco Demo" || account.AccountType != "savings" || !account.Active {
		t.Fatalf("cuenta inesperada: %+v", account)
	}

	_, found, err := toolset.accountsSearch(ctx, nil, AccountSearchInput{Query: strPtr(suffix), Limit: intPtrOf(50)})
	if err != nil {
		t.Fatalf("buscar cuenta: %v", err)
	}
	search := found.(AccountSearchOutput)
	if search.Count == 0 || search.Items[0].ID != account.ID {
		t.Fatalf("búsqueda inesperada: %+v", search)
	}

	newAlias := "Cuenta Demo Editada " + suffix
	_, updated, err := toolset.accountsUpdate(ctx, nil, AccountUpdateInput{ID: account.ID, Alias: strPtr(newAlias)})
	if err != nil {
		t.Fatalf("editar cuenta: %v", err)
	}
	edited := updated.(AccountItem)
	if edited.Alias != newAlias || edited.AccountNumber != accountNumber || edited.Beneficiary != "Titular Demo" {
		t.Fatalf("edición inesperada: %+v", edited)
	}

	result, output, err := toolset.accountsCreate(ctx, nil, AccountCreateInput{
		Bank: strPtr("Banco Demo"), Alias: "Cuenta Duplicada", AccountNumber: accountNumber,
		Currency: strPtr("NIO"), Beneficiary: "Titular Demo",
	})
	if err != nil || result == nil {
		t.Fatalf("se esperaba duplicado: %v", err)
	}
	if code := resultErrorCode(t, output); code != "already_exists" {
		t.Fatalf("código inesperado: %s", code)
	}

	cardAlias := "Tarjeta Demo " + suffix
	_, createdCard, err := toolset.cardsCreate(ctx, nil, CardCreateInput{
		Bank:     strPtr("Banco Demo"),
		Alias:    cardAlias,
		Network:  "visa",
		Last4:    "0000",
		Currency: strPtr("NIO"),
	})
	if err != nil {
		t.Fatalf("crear tarjeta: %v", err)
	}
	card := createdCard.(CardItem)
	if card.ID <= 0 || card.StatementDay != 1 || card.PaymentDueDay != 1 || card.LimitCurrency != "NIO" {
		t.Fatalf("tarjeta inesperada: %+v", card)
	}

	_, foundCards, err := toolset.cardsSearch(ctx, nil, CardSearchInput{Query: strPtr("0000"), Limit: intPtrOf(50)})
	if err != nil {
		t.Fatalf("buscar tarjeta: %v", err)
	}
	cardSearch := foundCards.(CardSearchOutput)
	if cardSearch.Count == 0 {
		t.Fatalf("búsqueda de tarjetas vacía: %+v", cardSearch)
	}
	foundID := false
	for _, item := range cardSearch.Items {
		if item.ID == card.ID {
			foundID = true
			break
		}
	}
	if !foundID {
		t.Fatalf("no se encontró la tarjeta %d: %+v", card.ID, cardSearch.Items)
	}

	cardAliasEdit := "Tarjeta Demo Editada " + suffix
	_, updatedCard, err := toolset.cardsUpdate(ctx, nil, CardUpdateInput{ID: card.ID, Alias: strPtr(cardAliasEdit)})
	if err != nil {
		t.Fatalf("editar tarjeta: %v", err)
	}
	editedCard := updatedCard.(CardItem)
	if editedCard.Alias != cardAliasEdit || editedCard.Last4 != "0000" || editedCard.Network != "visa" {
		t.Fatalf("edición de tarjeta inesperada: %+v", editedCard)
	}

	fmt.Printf("live OK: cuenta %d, tarjeta %d\n", account.ID, card.ID)
}
