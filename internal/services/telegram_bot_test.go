package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func testTelegramSettings(t *testing.T) *SystemSettingsService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
}

func TestFormatServiciosPendientes(t *testing.T) {
	format := DefaultCurrencyFormat()

	t.Run("sin pendientes", func(t *testing.T) {
		got := formatServiciosPendientes(nil, format)
		if len(got) != 1 || got[0] != "✅ No hay facturas pendientes." {
			t.Errorf("esperado único mensaje vacío, got %q", got)
		}
	})

	t.Run("un mensaje por servicio + totales, ordenado por monto", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Amount: 50, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{ServiceID: 2, ServiceName: "ENATREL", HomeName: "Casa B", Amount: 500, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatServiciosPendientes(pending, format)
		// 2 servicios + 1 totales.
		if len(got) != 3 {
			t.Fatalf("esperados 3 mensajes, got %d: %q", len(got), got)
		}
		// ENATREL (500) antes que Claro (150).
		enatrelPos := indexOf(got[0], "ENATREL")
		claroPos := indexOf(got[1], "Claro")
		if enatrelPos == -1 || claroPos == -1 {
			t.Fatalf("mensajes por servicio incorrectos:\n%s\n%s", got[0], got[1])
		}
		if !contains(got[1], "Facturas: 2") || !contains(got[1], "C$150.00") {
			t.Errorf("conteo/monto de Claro incorrecto:\n%s", got[1])
		}
		if !contains(got[0], "C$500.00") {
			t.Errorf("monto de ENATREL incorrecto:\n%s", got[0])
		}
		// Último mensaje: totales por moneda.
		if !contains(got[2], "Totales") || !contains(got[2], "NIO: C$650.00") {
			t.Errorf("mensaje de totales incorrecto:\n%s", got[2])
		}
	})

	t.Run("formato de moneda personalizado", func(t *testing.T) {
		format := CurrencyFormat{ThousandsSeparator: ",", DecimalSeparator: ".", DecimalDigits: 2}
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Amount: 1250.5, CurrencySymbol: "$", CurrencyCode: "USD"},
		}
		got := formatServiciosPendientes(pending, format)
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		if !contains(got[0], "$1,250.50") {
			t.Errorf("formato personalizado incorrecto:\n%s", got[0])
		}
		if !contains(got[1], "USD: $1,250.50") {
			t.Errorf("totales con formato personalizado incorrecto:\n%s", got[1])
		}
	})

	t.Run("totales multi-moneda", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Amount: 1000, CurrencySymbol: "$", CurrencyCode: "USD"},
			{ServiceID: 2, ServiceName: "S2", HomeName: "H", Amount: 5000, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatServiciosTotales(pending, format)
		if !contains(got, "USD: $1,000.00") || !contains(got, "NIO: C$5,000.00") {
			t.Errorf("totales multi-moneda incorrectos:\n%s", got)
		}
	})

	t.Run("totales con código vacío usa símbolo", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Amount: 100, CurrencySymbol: "C$"},
		}
		got := formatServiciosTotales(pending, format)
		if !contains(got, "C$: C$100.00") {
			t.Errorf("totales con fallback a símbolo incorrectos:\n%s", got)
		}
	})
}

func TestFormatDeudasPendientes(t *testing.T) {
	format := DefaultCurrencyFormat()

	t.Run("sin pendientes", func(t *testing.T) {
		got := formatDeudasPendientes(nil, format)
		if len(got) != 1 || got[0] != "✅ No hay deudas pendientes." {
			t.Errorf("esperado único mensaje vacío, got %q", got)
		}
	})

	t.Run("un mensaje por deuda + totales, ordenado por monto", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "Préstamo LAFISE", InstitutionName: "Banco LAFISE", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 1, DebtDescription: "Préstamo LAFISE", InstitutionName: "Banco LAFISE", Amount: 50, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 2, DebtDescription: "Tarjeta BAC", InstitutionName: "BAC Credomatic", Amount: 500, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasPendientes(pending, format)
		// 2 deudas + 1 totales.
		if len(got) != 3 {
			t.Fatalf("esperados 3 mensajes, got %d: %q", len(got), got)
		}
		if !contains(got[0], "Tarjeta BAC") || !contains(got[0], "C$500.00") {
			t.Errorf("mensaje de BAC incorrecto:\n%s", got[0])
		}
		if !contains(got[1], "Préstamo LAFISE") || !contains(got[1], "Cuotas: 2") || !contains(got[1], "C$150.00") {
			t.Errorf("mensaje de LAFISE incorrecto:\n%s", got[1])
		}
		if !contains(got[2], "Totales") || !contains(got[2], "NIO: C$650.00") {
			t.Errorf("mensaje de totales incorrecto:\n%s", got[2])
		}
	})

	t.Run("totales multi-moneda", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", Amount: 200, CurrencySymbol: "$", CurrencyCode: "USD"},
			{DebtID: 2, DebtDescription: "D2", InstitutionName: "I2", Amount: 300, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasTotales(pending, format)
		if !contains(got, "USD: $200.00") || !contains(got, "NIO: C$300.00") {
			t.Errorf("totales multi-moneda incorrectos:\n%s", got)
		}
	})

	t.Run("totales con código vacío usa símbolo", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", Amount: 75, CurrencySymbol: "C$"},
		}
		got := formatDeudasTotales(pending, format)
		if !contains(got, "C$: C$75.00") {
			t.Errorf("totales con fallback a símbolo incorrectos:\n%s", got)
		}
	})
}

func TestTelegramBotServiceLifecycle(t *testing.T) {
	settings := testTelegramSettings(t)
	bills := storage.NewBillStorage(nil)
	svc := NewTelegramBotService(settings, bills, storage.NewDebtBillStorage(nil))

	// Sin config: Start no debe panicear y el supervisor queda inactivo.
	svc.Start()
	svc.Stop()

	// Notify sin polling activo no debe bloquearse.
	svc.NotifyConfigChanged()
}

func TestTelegramBotConfigSettings(t *testing.T) {
	ctx := context.Background()
	settings := testTelegramSettings(t)

	cfg, err := settings.GetTelegramBotConfig(ctx)
	if err != nil {
		t.Fatalf("config inicial: %v", err)
	}
	if cfg.Enabled || cfg.Token != "" || len(cfg.ChatIDs) != 0 {
		t.Errorf("config inicial no vacía: %+v", cfg)
	}

	if err := settings.SetTelegramBotEnabled(ctx, true); err != nil {
		t.Fatalf("enabled: %v", err)
	}
	if err := settings.SetTelegramBotToken(ctx, "token-123"); err != nil {
		t.Fatalf("token: %v", err)
	}
	if err := settings.SetTelegramBotChatIDs(ctx, []string{"111", "222"}); err != nil {
		t.Fatalf("chat_ids: %v", err)
	}

	cfg, err = settings.GetTelegramBotConfig(ctx)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !cfg.Enabled || cfg.Token != "token-123" || len(cfg.ChatIDs) != 2 {
		t.Errorf("config guardada incorrecta: %+v", cfg)
	}

	// El token no debe sobrescribirse con vacío (setIfNonEmpty).
	if err := settings.SetTelegramBotToken(ctx, ""); err != nil {
		t.Fatalf("token vacío: %v", err)
	}
	cfg, _ = settings.GetTelegramBotConfig(ctx)
	if cfg.Token != "token-123" {
		t.Errorf("token fue sobrescrito con vacío: %q", cfg.Token)
	}

	// Config pública: sin token, con flag configured.
	pub, err := settings.GetTelegramBotConfigPublic(ctx)
	if err != nil {
		t.Fatalf("public: %v", err)
	}
	if !pub.Enabled || !pub.Configured {
		t.Errorf("public incorrecta: %+v", pub)
	}

	// Clear: limpia todo y apaga.
	if err := settings.ClearTelegramBot(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	cfg, _ = settings.GetTelegramBotConfig(ctx)
	if cfg.Enabled || cfg.Token != "" || len(cfg.ChatIDs) != 0 {
		t.Errorf("config tras clear incorrecta: %+v", cfg)
	}
}

func TestIsAuthorized(t *testing.T) {
	ctx := context.Background()
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil), storage.NewDebtBillStorage(nil))

	// Sin allowlist: todos autorizados.
	if !svc.isAuthorized(ctx, 12345) {
		t.Error("sin allowlist debería autorizar")
	}

	if err := settings.SetTelegramBotChatIDs(ctx, []string{"111", "222"}); err != nil {
		t.Fatalf("chat_ids: %v", err)
	}
	if !svc.isAuthorized(ctx, 111) || !svc.isAuthorized(ctx, 222) {
		t.Error("chat en allowlist debería autorizar")
	}
	if svc.isAuthorized(ctx, 333) {
		t.Error("chat fuera de allowlist no debería autorizar")
	}
}

func TestCheckAuthorizedIgnoresNonMessage(t *testing.T) {
	ctx := context.Background()
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil), storage.NewDebtBillStorage(nil))
	// update.Message == nil → no autorizado (no panic).
	ok := svc.checkAuthorized(ctx, nil, &tgmodels.Update{})
	if ok {
		t.Error("update sin message no debería autorizarse")
	}
}

// TestBotNewFailsInvalidToken verifica que un token inválido no tumba el
// supervisor: bot.New devuelve error y runBot retorna (CA-NF-001).
func TestBotNewFailsInvalidToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil), storage.NewDebtBillStorage(nil))

	cfg := &appmodels.TelegramBotConfig{Enabled: true, Token: "token-invalido"}
	done := make(chan struct{})
	go func() {
		svc.runBot(ctx, cfg)
		close(done)
	}()
	select {
	case <-done:
		// runBot retornó sin panic (getMe falló).
	case <-time.After(1 * time.Second):
		t.Fatal("runBot no retornó con token inválido")
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// fakeTelegramAPI simula la Bot API de Telegram para testear el dispatcher
// real de go-telegram/bot (SPEC-080): getMe, setMyCommands y sendMessage.
func fakeTelegramAPI(t *testing.T, sent *[]string, mu *sync.Mutex) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/botTESTTOKEN/getMe", func(w http.ResponseWriter, r *http.Request) {
		writeTGJSON(t, w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"id": 1, "is_bot": true, "first_name": "Test", "username": "testbot",
			},
		})
	})

	mux.HandleFunc("/botTESTTOKEN/setMyCommands", func(w http.ResponseWriter, r *http.Request) {
		writeTGJSON(t, w, map[string]any{"ok": true, "result": true})
	})

	mux.HandleFunc("/botTESTTOKEN/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("sendMessage: parse form: %v", err)
			writeTGJSON(t, w, map[string]any{"ok": false})
			return
		}
		mu.Lock()
		*sent = append(*sent, r.FormValue("text"))
		mu.Unlock()
		writeTGJSON(t, w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": 1,
				"chat":       map[string]any{"id": 123},
				"text":       r.FormValue("text"),
			},
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeTGJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("escribir respuesta fake: %v", err)
	}
}

// TestCommandDispatchRealMatcher verifica el dispatcher REAL de la librería
// (SPEC-080): los handlers se registran SIN slash y el matcher de
// MatchTypeCommand los hace coincidir con /pendientes y /start. Con el
// patrón viejo ("/pendientes" con slash) el matcher NO coincidía y todo
// caía al handler default ("Comando no reconocido").
func TestCommandDispatchRealMatcher(t *testing.T) {
	settings := testTelegramSettings(t)

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	svc := NewTelegramBotService(settings, storage.NewBillStorage(database), storage.NewDebtBillStorage(database))

	var sent []string
	var mu sync.Mutex
	srv := fakeTelegramAPI(t, &sent, &mu)

	b, err := bot.New("TESTTOKEN", bot.WithServerURL(srv.URL), bot.WithNotAsyncHandlers(), bot.WithDefaultHandler(svc.handleDefault))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	b.RegisterHandler(bot.HandlerTypeMessageText, "servicios_pendientes", bot.MatchTypeCommand, svc.handleServiciosPendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "deudas_pendientes", bot.MatchTypeCommand, svc.handleDeudasPendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, svc.handleStart)

	ctx := context.Background()

	msg := func(text string) *tgmodels.Update {
		return &tgmodels.Update{
			ID: 1,
			Message: &tgmodels.Message{
				Chat: tgmodels.Chat{ID: 123},
				Text: text,
				Entities: []tgmodels.MessageEntity{
					{Type: tgmodels.MessageEntityTypeBotCommand, Offset: 0, Length: len(text)},
				},
			},
		}
	}

	t.Run("servicios_pendientes matchea y responde datos", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/servicios_pendientes"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta: /servicios_pendientes cayó al default handler")
		}
		last := replies[len(replies)-1]
		if strings.Contains(last, "Comando no reconocido") {
			t.Fatalf("/servicios_pendientes respondió con el default handler: %q", last)
		}
		if !strings.Contains(last, "pendientes") {
			t.Errorf("respuesta inesperada: %q", last)
		}
	})

	t.Run("deudas_pendientes matchea y responde datos", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/deudas_pendientes"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta: /deudas_pendientes cayó al default handler")
		}
		last := replies[len(replies)-1]
		if strings.Contains(last, "Comando no reconocido") {
			t.Fatalf("/deudas_pendientes respondió con el default handler: %q", last)
		}
		if !strings.Contains(last, "pendientes") {
			t.Errorf("respuesta inesperada: %q", last)
		}
	})

	t.Run("start matchea y responde bienvenida", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/start"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta: /start cayó al default handler")
		}
		last := replies[len(replies)-1]
		if !strings.Contains(last, "Bienvenida") && !strings.Contains(last, "Comandos") {
			t.Errorf("respuesta de /start inesperada: %q", last)
		}
	})

	t.Run("comando desconocido cae al default", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/otro"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta para comando desconocido")
		}
		last := replies[len(replies)-1]
		if !strings.Contains(last, "Comando no reconocido") {
			t.Errorf("esperado default handler, got: %q", last)
		}
	})
}

var _ = bot.HandlerFunc(nil) // mantener import de bot en tests
