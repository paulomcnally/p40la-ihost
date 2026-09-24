package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestDispatchVoice_Disabled(t *testing.T) {
	alertService, _, vm := newAlertService(t)
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	// Voz deshabilitada para la alerta → no se anuncia.
	mail := true
	voice := false
	if err := alertService.SetFlags(ctx, models.AlertKeyBillSummary, &mail, &voice, nil); err != nil {
		t.Fatalf("set flags: %v", err)
	}
	dispatchVoice(ctx, alertService, vm, models.AlertKeyBillSummary, "Prueba")
	if called {
		t.Error("no debería anunciarse con voz deshabilitada")
	}
}

func TestDispatchVoice_VMMasterOff(t *testing.T) {
	alertService, _, vm := newAlertService(t)
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	// Voz habilitada para la alerta, pero Voice Monkey desactivado.
	mail := true
	voice := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, &mail, &voice, nil); err != nil {
		t.Fatalf("set flags: %v", err)
	}
	dispatchVoice(ctx, alertService, vm, models.AlertKeyInsurance, "Prueba")
	if called {
		t.Error("no debería anunciarse con Voice Monkey desactivado")
	}
}

func TestDispatchVoice_Enabled(t *testing.T) {
	alertService, settingsService, vm := newAlertService(t)
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	// Activar Voice Monkey: master on + enviar alertas on + token/device.
	if err := settingsService.SetVoiceMonkeyEnabled(ctx, true); err != nil {
		t.Fatalf("set master: %v", err)
	}
	if err := settingsService.SetVoiceMonkeySendAlerts(ctx, true); err != nil {
		t.Fatalf("set send alerts: %v", err)
	}
	if err := settingsService.SetVoiceMonkeyConfig(ctx, &models.VoiceMonkeyConfig{Token: "tok", Device: "dev"}); err != nil {
		t.Fatalf("set config VM: %v", err)
	}

	mail := true
	voice := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, &mail, &voice, nil); err != nil {
		t.Fatalf("set flags: %v", err)
	}

	dispatchVoice(ctx, alertService, vm, models.AlertKeyInsurance, "Prueba")
	if !called {
		t.Error("debería haberse anunciado por voz")
	}
}

// newTelegramBotForTest crea un TelegramBotService con el bot fake activo
// (misma técnica que TestCommandDispatchRealMatcher, SPEC-080).
func newTelegramBotForTest(t *testing.T, sent *[]string, mu *sync.Mutex) (*TelegramBotService, *SystemSettingsService) {
	t.Helper()
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, nil, nil)

	srv := fakeTelegramAPI(t, sent, mu)
	b, err := bot.New("TESTTOKEN", bot.WithServerURL(srv.URL), bot.WithNotAsyncHandlers(), bot.WithDefaultHandler(svc.handleDefault))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	svc.botMu.Lock()
	svc.activeBot = b
	svc.botMu.Unlock()
	return svc, settings
}

func TestDispatchTelegram_AlertChannelOff(t *testing.T) {
	alertService, _, _ := newAlertService(t)
	ctx := context.Background()

	var sent []string
	var mu sync.Mutex
	svc, settings := newTelegramBotForTest(t, &sent, &mu)
	if err := settings.SetTelegramBotEnabled(ctx, true); err != nil {
		t.Fatalf("set bot enabled: %v", err)
	}
	if err := settings.SetTelegramBotToken(ctx, "TESTTOKEN"); err != nil {
		t.Fatalf("set bot token: %v", err)
	}
	if err := settings.SetTelegramBotChatIDs(ctx, []string{"111"}); err != nil {
		t.Fatalf("set chat ids: %v", err)
	}

	// Canal telegram deshabilitado para la alerta → no se envía nada.
	mail := true
	voice := false
	telegram := false
	if err := alertService.SetFlags(ctx, models.AlertKeyBillSummary, &mail, &voice, &telegram); err != nil {
		t.Fatalf("set flags: %v", err)
	}
	dispatchTelegram(ctx, alertService, svc, models.AlertKeyBillSummary, []string{"Hola"})
	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 0 {
		t.Errorf("no debería enviarse con canal telegram deshabilitado, sent %d", len(sent))
	}
}

func TestDispatchTelegram_BotDisabledNeverContactsAPI(t *testing.T) {
	alertService, _, _ := newAlertService(t)
	ctx := context.Background()

	var sent []string
	var mu sync.Mutex
	svc, settings := newTelegramBotForTest(t, &sent, &mu)
	// Bot NO habilitado en settings (sin token tampoco).

	telegram := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, nil, nil, &telegram); err != nil {
		t.Fatalf("set flags: %v", err)
	}

	// El bot no está habilitado: SendAlerts corta antes de contactar la API.
	if err := svc.SendAlerts(ctx, []string{"Hola"}); err != nil {
		t.Fatalf("SendAlerts con bot apagado no debe fallar: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 0 {
		t.Errorf("bot apagado no debe enviar nada, sent %d", len(sent))
	}

	// También vía dispatchTelegram (el gate se aplica igual).
	dispatchTelegram(ctx, alertService, svc, models.AlertKeyInsurance, []string{"Hola"})
	if len(sent) != 0 {
		t.Errorf("dispatchTelegram con bot apagado no debe enviar nada, sent %d", len(sent))
	}
	_ = settings
}

func TestDispatchTelegram_SendsToAllAuthorizedChatIDs(t *testing.T) {
	alertService, _, _ := newAlertService(t)
	ctx := context.Background()

	var sent []string
	var mu sync.Mutex
	svc, settings := newTelegramBotForTest(t, &sent, &mu)
	if err := settings.SetTelegramBotEnabled(ctx, true); err != nil {
		t.Fatalf("set bot enabled: %v", err)
	}
	if err := settings.SetTelegramBotToken(ctx, "TESTTOKEN"); err != nil {
		t.Fatalf("set bot token: %v", err)
	}
	if err := settings.SetTelegramBotChatIDs(ctx, []string{"111", "222"}); err != nil {
		t.Fatalf("set chat ids: %v", err)
	}

	telegram := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, nil, nil, &telegram); err != nil {
		t.Fatalf("set flags: %v", err)
	}

	dispatchTelegram(ctx, alertService, svc, models.AlertKeyInsurance, []string{"msg1", "msg2"})
	mu.Lock()
	defer mu.Unlock()
	// 2 chat_ids × 2 mensajes.
	if len(sent) != 4 {
		t.Errorf("esperados 4 envíos (2 chats × 2 mensajes), sent %d: %q", len(sent), sent)
	}
}

func TestSendAlerts_NoChatIDs(t *testing.T) {
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, nil, nil)
	ctx := context.Background()

	if err := settings.SetTelegramBotEnabled(ctx, true); err != nil {
		t.Fatalf("set bot enabled: %v", err)
	}
	if err := settings.SetTelegramBotToken(ctx, "TESTTOKEN"); err != nil {
		t.Fatalf("set bot token: %v", err)
	}

	// Sin chat_ids autorizados: no envía y no falla.
	if err := svc.SendAlerts(ctx, []string{"Hola"}); err != nil {
		t.Fatalf("SendAlerts sin chat_ids no debe fallar: %v", err)
	}
}
