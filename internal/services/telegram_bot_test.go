package services

import (
	"context"
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

func TestFormatPendientes(t *testing.T) {
	format := DefaultCurrencyFormat()

	t.Run("sin pendientes", func(t *testing.T) {
		got := formatPendientes(nil, format)
		if got != "✅ No hay facturas pendientes." {
			t.Errorf("esperado mensaje vacío, got %q", got)
		}
	})

	t.Run("agrupa por servicio y ordena por monto", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Amount: 100, CurrencySymbol: "C$"},
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Amount: 50, CurrencySymbol: "C$"},
			{ServiceID: 2, ServiceName: "ENATREL", HomeName: "Casa B", Amount: 500, CurrencySymbol: "C$"},
		}
		got := formatPendientes(pending, format)
		if len(got) == 0 {
			t.Fatal("mensaje vacío")
		}
		// ENATREL (500) debe aparecer antes que Claro (150).
		enatrelPos := indexOf(got, "ENATREL")
		claroPos := indexOf(got, "Claro")
		if enatrelPos == -1 || claroPos == -1 || enatrelPos > claroPos {
			t.Errorf("orden incorrecto: ENATREL=%d Claro=%d\n%s", enatrelPos, claroPos, got)
		}
		if !contains(got, "Facturas: 2") || !contains(got, "C$150.00") {
			t.Errorf("conteo/monto de Claro incorrecto:\n%s", got)
		}
		if !contains(got, "C$500.00") {
			t.Errorf("monto de ENATREL incorrecto:\n%s", got)
		}
	})

	t.Run("formato de moneda personalizado", func(t *testing.T) {
		format := CurrencyFormat{ThousandsSeparator: ",", DecimalSeparator: ".", DecimalDigits: 2}
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Amount: 1250.5, CurrencySymbol: "$"},
		}
		got := formatPendientes(pending, format)
		if !contains(got, "$1,250.50") {
			t.Errorf("formato personalizado incorrecto:\n%s", got)
		}
	})
}

func TestTelegramBotServiceLifecycle(t *testing.T) {
	settings := testTelegramSettings(t)
	bills := storage.NewBillStorage(nil)
	svc := NewTelegramBotService(settings, bills)

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
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil))

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
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil))
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
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil))

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

var _ = bot.HandlerFunc(nil) // mantener import de bot en tests
