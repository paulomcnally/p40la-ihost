package entities

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

func searchResult(t *testing.T, output any) EntitiesSearchOutput {
	t.Helper()
	result, ok := output.(EntitiesSearchOutput)
	if !ok {
		t.Fatalf("output = %T, quiero EntitiesSearchOutput", output)
	}
	return result
}

func TestEntitiesSearchHomesRankingAndAccents(t *testing.T) {
	f := &fakeAPI{homes: []api.Home{
		{ID: 1, Name: "Internet"},
		{ID: 2, Name: "Internet Demo"},
		{ID: 3, Name: "Mi Internet"},
		{ID: 4, Name: "Córdoba"},
	}}
	tool := newTestToolset(f, false)
	ctx := context.Background()

	result, output, err := tool.entitiesSearch(ctx, nil, EntitiesSearchInput{Entity: "home", Query: "internet"})
	if err != nil || result.IsError {
		t.Fatalf("search: err=%v result=%#v", err, result)
	}
	got := searchResult(t, output)
	if got.Count != 3 || got.Total != 3 {
		t.Fatalf("count=%d total=%d", got.Count, got.Total)
	}
	want := []int64{1, 2, 3}
	for i, id := range want {
		if got.Items[i].ID != id {
			t.Fatalf("orden = %v, quiero %v", ids(got.Items), want)
		}
	}

	result, output, _ = tool.entitiesSearch(ctx, nil, EntitiesSearchInput{Entity: "home", Query: "cordoba"})
	if result.IsError {
		t.Fatalf("acentos: %#v", result)
	}
	if got := searchResult(t, output); got.Count != 1 || got.Items[0].ID != 4 {
		t.Fatalf("acentos count=%d items=%v", got.Count, ids(got.Items))
	}
}

func TestEntitiesSearchServicesByHome(t *testing.T) {
	f := &fakeAPI{
		homes: []api.Home{{ID: 7, Name: "Hogar Demo"}},
		services: []api.Service{
			{ID: 12, Name: "Internet Demo", HomeID: 7, Active: true, Frequency: "monthly"},
			{ID: 13, Name: "Internet Playa", HomeID: 8, Active: true, Frequency: "monthly"},
		},
	}
	tool := newTestToolset(f, true)
	ctx := context.Background()

	result, output, _ := tool.entitiesSearch(ctx, nil, EntitiesSearchInput{Entity: "service", Query: "internet"})
	if result.IsError {
		t.Fatalf("global: %#v", result)
	}
	if got := searchResult(t, output); got.Count != 2 {
		t.Fatalf("global count=%d", got.Count)
	}

	result, output, _ = tool.entitiesSearch(ctx, nil, EntitiesSearchInput{Entity: "service", Query: "internet", Home: "hogar demo"})
	if result.IsError {
		t.Fatalf("filtro: %#v", result)
	}
	got := searchResult(t, output)
	if got.Count != 1 || got.Items[0].ID != 12 || got.Items[0].HomeID == nil || *got.Items[0].HomeID != 7 {
		t.Fatalf("filtro items=%v", got.Items)
	}
}

func TestEntitiesSearchAmbiguousHome(t *testing.T) {
	f := &fakeAPI{
		homes:    []api.Home{{ID: 7, Name: "Hogar Demo"}, {ID: 8, Name: "Hogar Demo"}},
		services: []api.Service{{ID: 12, Name: "Internet Demo", HomeID: 7}},
	}
	tool := newTestToolset(f, true)
	_, output, _ := tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "service", Home: "Hogar Demo"})
	if output.(map[string]any)["error"] != "ambiguous_reference" {
		t.Fatalf("output=%v", output)
	}
}

func TestEntitiesSearchBanksByShortCode(t *testing.T) {
	f := &fakeAPI{banks: []api.Bank{
		{ID: 5, Name: "Banco Demo", ShortCode: "DEMO", Country: "NI", Active: true},
		{ID: 6, Name: "Otro Banco", ShortCode: "OTRO", Country: "PA", Active: false},
	}}
	tool := newTestToolset(f, true)
	result, output, _ := tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "bank", Query: "dem"})
	if result.IsError {
		t.Fatalf("bank: %#v", result)
	}
	got := searchResult(t, output)
	if got.Count != 1 || got.Items[0].ID != 5 || got.Items[0].ShortCode != "DEMO" {
		t.Fatalf("items=%v", got.Items)
	}
}

func TestEntitiesSearchInstitutions(t *testing.T) {
	category := int64(3)
	f := &fakeAPI{institutions: []api.Institution{{ID: 9, Name: "Proveedor Demo", CategoryID: &category}}}
	tool := newTestToolset(f, true)
	result, output, _ := tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "institution", Query: "proveedor"})
	if result.IsError {
		t.Fatalf("institution: %#v", result)
	}
	got := searchResult(t, output)
	if got.Count != 1 || got.Items[0].ID != 9 || got.Items[0].CategoryID == nil {
		t.Fatalf("items=%v", got.Items)
	}
}

func TestEntitiesSearchEmptyResultIsNotError(t *testing.T) {
	tool := newTestToolset(&fakeAPI{}, true)
	result, output, _ := tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "home", Query: "zzz"})
	if result.IsError {
		t.Fatal("una búsqueda vacía no debe ser error")
	}
	got := searchResult(t, output)
	if got.Count != 0 || got.Total != 0 || got.Hint == "" || got.Items == nil {
		t.Fatalf("output=%#v", got)
	}
}

func TestEntitiesSearchLimitAndTruncated(t *testing.T) {
	f := &fakeAPI{homes: []api.Home{
		{ID: 1, Name: "Alfa"}, {ID: 2, Name: "Beta"}, {ID: 3, Name: "Gamma"},
	}}
	tool := newTestToolset(f, true)
	limit := 2
	_, output, _ := tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "home", Limit: &limit})
	got := searchResult(t, output)
	if got.Count != 2 || got.Total != 3 || !got.Truncated {
		t.Fatalf("output=%#v", got)
	}
}

func TestEntitiesSearchEmptyQueryListsSorted(t *testing.T) {
	f := &fakeAPI{homes: []api.Home{{ID: 2, Name: "Beta"}, {ID: 1, Name: "Alfa"}}}
	tool := newTestToolset(f, true)
	_, output, _ := tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "home"})
	got := searchResult(t, output)
	if len(got.Items) != 2 || got.Items[0].ID != 1 || got.Items[1].ID != 2 {
		t.Fatalf("items=%v", ids(got.Items))
	}
}

func TestEntitiesSearchValidation(t *testing.T) {
	tool := newTestToolset(&fakeAPI{}, true)
	result, output, _ := tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "otro"})
	if !result.IsError || output.(map[string]any)["error"] != "invalid_value" {
		t.Fatalf("entity: %#v %v", result, output)
	}
	limit := 51
	result, output, _ = tool.entitiesSearch(context.Background(), nil, EntitiesSearchInput{Entity: "home", Limit: &limit})
	if !result.IsError || output.(map[string]any)["error"] != "invalid_value" {
		t.Fatalf("limit: %#v %v", result, output)
	}
}

func ids(items []SearchItem) []int64 {
	out := make([]int64, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}
