package services

import (
	"strings"
	"testing"
	"time"

	tgmodels "github.com/go-telegram/bot/models"
	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
)

// --- parseAmount (SPEC-097 REQ-006) ---

func TestParseAmount(t *testing.T) {
	cases := []struct {
		in     string
		want   float64
		wantOK bool
	}{
		{"150.50", 150.50, true},
		{"150,50", 150.50, true}, // coma decimal normalizada
		{"0.01", 0.01, true},     // mínimo válido
		{" 42 ", 42, true},       // con espacios
		{"", 0, false},           // vacío
		{"0", 0, false},          // cero no es un gasto válido
		{"-5", 0, false},         // negativo
		{"abc", 0, false},        // no numérico
		{"150.50.30", 0, false},  // doble punto
		{"1e13", 0, false},       // mayor al tope 1e12
		{"150.50 C$", 0, false},  // texto extra
	}
	for _, c := range cases {
		got, ok := parseAmount(c.in)
		if ok != c.wantOK {
			t.Errorf("parseAmount(%q) ok=%v, want %v", c.in, ok, c.wantOK)
			continue
		}
		if ok && got != c.want {
			t.Errorf("parseAmount(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// --- parseDate (SPEC-097 REQ-007) ---

func TestParseDate(t *testing.T) {
	now := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"2026-09-28", "2026-09-28", true},
		{"2026-09-05", "2026-09-05", true}, // día con cero inicial
		{"hoy", "2026-09-28", true},
		{"Hoy", "2026-09-28", true},
		{"today", "2026-09-28", true},
		{"", "2026-09-28", true},  // vacío = hoy
		{"28/09/2026", "", false}, // formato incorrecto
		{"2026-13-01", "", false}, // mes inválido
		{"2026-09-32", "", false}, // día inválido
	}
	for _, c := range cases {
		got, ok := parseDate(c.in, now)
		if ok != c.wantOK {
			t.Errorf("parseDate(%q) ok=%v, want %v", c.in, ok, c.wantOK)
			continue
		}
		if ok && got != c.want {
			t.Errorf("parseDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- budgetKindLabel / budgetSummary (SPEC-097 REQ-010) ---

func TestBudgetKindLabel(t *testing.T) {
	if got := budgetKindLabel(true); got != "💰 Ingreso" {
		t.Errorf("budgetKindLabel(true) = %q, want Ingreso", got)
	}
	if got := budgetKindLabel(false); got != "💸 Gasto" {
		t.Errorf("budgetKindLabel(false) = %q, want Gasto", got)
	}
}

func TestBudgetSummary(t *testing.T) {
	session := &budgetTxSession{
		categoryName:   "Supermercado",
		isInflow:       false,
		amount:         150.50,
		date:           "2026-09-28",
		accountName:    "BANPRO",
		currencySymbol: "C$",
		payee:          "La Colonia",
	}
	format := DefaultCurrencyFormat()

	got := budgetSummaryForTest(session, format)
	for _, want := range []string{"Supermercado", "💸 Gasto", "C$150.50", "2026-09-28", "BANPRO", "La Colonia"} {
		if !strings.Contains(got, want) {
			t.Errorf("budgetSummary sin %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Concepto") == false {
		t.Errorf("budgetSummary debería incluir el payee")
	}

	// Sin payee no debe aparecer la línea Concepto.
	session.payee = ""
	got = budgetSummaryForTest(session, format)
	if strings.Contains(got, "Concepto") {
		t.Errorf("budgetSummary no debería incluir Concepto sin payee:\n%s", got)
	}
}

// budgetSummaryForTest adapta budgetSummary (método con receiver) para test.
func budgetSummaryForTest(session *budgetTxSession, format CurrencyFormat) string {
	return (&TelegramBotService{}).budgetSummary(session, format)
}

// --- budgetBuildTransaction (SPEC-097 REQ-010) ---

func TestBudgetBuildTransaction(t *testing.T) {
	catID := int64(7)
	base := &budgetTxSession{
		accountID:      3,
		categoryID:     catID,
		currencyID:     1,
		date:           "2026-09-28",
		payee:          "Pago",
		amount:         100,
		currencySymbol: "C$",
	}

	outflow := *base
	outflow.isInflow = false
	tx := budgetBuildTransaction(&outflow)
	if tx.Outflow != 100 || tx.Inflow != 0 {
		t.Errorf("outflow mal armado: %+v", tx)
	}
	if tx.CategoryID == nil || *tx.CategoryID != catID {
		t.Errorf("category_id mal armado: %+v", tx)
	}
	if tx.Payee != "Pago" || tx.Date != "2026-09-28" || tx.AccountID != 3 || tx.CurrencyID != 1 {
		t.Errorf("campos base mal armados: %+v", tx)
	}

	inflow := *base
	inflow.isInflow = true
	tx = budgetBuildTransaction(&inflow)
	if tx.Inflow != 100 || tx.Outflow != 0 {
		t.Errorf("inflow mal armado: %+v", tx)
	}
}

// --- Teclados inline (SPEC-097 REQ-003..010) ---

func TestBudgetGroupKeyboardFiltersEmptyGroups(t *testing.T) {
	groups := []BudgetCategoryGroup{
		{ID: 1, Name: "Necesidades", Icon: "🏠", Categories: []appmodels.Category{{ID: 11, Name: "Alquiler"}}},
		{ID: 2, Name: "Vacío", Categories: []appmodels.Category{}}, // sin categorías → no aparece
	}
	kb := (&TelegramBotService{}).budgetGroupKeyboard(groups)
	buttons := flattenInlineButtons(kb)

	var foundCancel bool
	grpButtons := 0
	for _, b := range buttons {
		if b.CallbackData == "bt:cancel" {
			foundCancel = true
		}
		if strings.HasPrefix(b.CallbackData, "bt:grp:") {
			grpButtons++
		}
	}
	if grpButtons != 1 {
		t.Errorf("debería haber 1 botón de grupo, hay %d", grpButtons)
	}
	if !foundCancel {
		t.Error("debería incluir botón Cancelar")
	}
	if buttons[0].Text != "🏠 Necesidades" {
		t.Errorf("label del grupo con ícono incorrecto: %q", buttons[0].Text)
	}
}

func TestBudgetCategoryKeyboard(t *testing.T) {
	cats := []appmodels.Category{
		{ID: 11, Name: "Alquiler", Icon: "🏠"},
		{ID: 12, Name: "Luz"},
	}
	kb := (&TelegramBotService{}).budgetCategoryKeyboard(cats)
	buttons := flattenInlineButtons(kb)
	if len(buttons) != 3 { // 2 categorías + cancelar
		t.Fatalf("se esperaban 3 botones, hay %d", len(buttons))
	}
	if buttons[0].CallbackData != "bt:cat:11" || buttons[0].Text != "🏠 Alquiler" {
		t.Errorf("botón categoría 1 incorrecto: %+v", buttons[0])
	}
	if buttons[1].CallbackData != "bt:cat:12" || buttons[1].Text != "Luz" {
		t.Errorf("botón categoría 2 incorrecto: %+v", buttons[1])
	}
	if buttons[2].CallbackData != "bt:cancel" {
		t.Errorf("último botón debería ser Cancelar: %+v", buttons[2])
	}
}

func TestBudgetTypeAndConfirmKeyboards(t *testing.T) {
	svc := &TelegramBotService{}

	types := flattenInlineButtons(svc.budgetTypeKeyboard())
	wantTypes := map[string]bool{"bt:type:outflow": true, "bt:type:inflow": true, "bt:cancel": true}
	if len(types) != 3 {
		t.Fatalf("type keyboard: se esperaban 3 botones, hay %d", len(types))
	}
	for _, b := range types {
		if !wantTypes[b.CallbackData] {
			t.Errorf("callback inesperado en type keyboard: %q", b.CallbackData)
		}
	}

	confirm := flattenInlineButtons(svc.budgetConfirmKeyboard())
	if len(confirm) != 2 || confirm[0].CallbackData != "bt:ok" || confirm[1].CallbackData != "bt:cancel" {
		t.Errorf("confirm keyboard incorrecto: %+v", confirm)
	}
}

// --- budgetCleanup (SPEC-097 REQ-012) ---

func TestBudgetCleanupRemovesExpiredOnly(t *testing.T) {
	svc := &TelegramBotService{txSession: make(map[int64]*budgetTxSession)}
	now := time.Now()

	svc.txSession[1] = &budgetTxSession{chatID: 1, expiresAt: now.Add(-time.Minute)} // expirada
	svc.txSession[2] = &budgetTxSession{chatID: 2, expiresAt: now.Add(time.Hour)}    // vigente

	svc.budgetCleanup()

	if _, ok := svc.txSession[1]; ok {
		t.Error("la sesión expirada debería haberse eliminado")
	}
	if _, ok := svc.txSession[2]; !ok {
		t.Error("la sesión vigente no debería haberse eliminado")
	}
}

func flattenInlineButtons(kb *tgmodels.InlineKeyboardMarkup) []tgmodels.InlineKeyboardButton {
	var out []tgmodels.InlineKeyboardButton
	for _, row := range kb.InlineKeyboard {
		out = append(out, row...)
	}
	return out
}
