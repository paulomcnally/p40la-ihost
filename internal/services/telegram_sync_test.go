package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ---- Infra de test ----

type syncSentMessage struct {
	Text        string
	ReplyMarkup string
}

type syncFakeAPI struct {
	mu       sync.Mutex
	messages []syncSentMessage
	answers  []string
	edits    []syncSentMessage
	server   *httptest.Server
}

func newSyncFakeAPI(t *testing.T) *syncFakeAPI {
	t.Helper()
	f := &syncFakeAPI{}
	mux := http.NewServeMux()

	mux.HandleFunc("/botTESTTOKEN/getMe", func(w http.ResponseWriter, r *http.Request) {
		writeTGJSON(t, w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"id": 1, "is_bot": true, "first_name": "Test", "username": "testbot",
			},
		})
	})

	mux.HandleFunc("/botTESTTOKEN/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("sendMessage: parse form: %v", err)
			writeTGJSON(t, w, map[string]any{"ok": false})
			return
		}
		f.mu.Lock()
		f.messages = append(f.messages, syncSentMessage{
			Text:        r.FormValue("text"),
			ReplyMarkup: r.FormValue("reply_markup"),
		})
		f.mu.Unlock()
		writeTGJSON(t, w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": 1,
				"chat":       map[string]any{"id": 123},
				"text":       r.FormValue("text"),
			},
		})
	})

	mux.HandleFunc("/botTESTTOKEN/answerCallbackQuery", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("answerCallbackQuery: parse form: %v", err)
		}
		f.mu.Lock()
		f.answers = append(f.answers, r.FormValue("text"))
		f.mu.Unlock()
		writeTGJSON(t, w, map[string]any{"ok": true, "result": true})
	})

	mux.HandleFunc("/botTESTTOKEN/editMessageText", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("editMessageText: parse form: %v", err)
		}
		f.mu.Lock()
		f.edits = append(f.edits, syncSentMessage{
			Text:        r.FormValue("text"),
			ReplyMarkup: r.FormValue("reply_markup"),
		})
		f.mu.Unlock()
		writeTGJSON(t, w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": 55,
				"chat":       map[string]any{"id": 123},
				"text":       r.FormValue("text"),
			},
		})
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *syncFakeAPI) snapshot() (messages []syncSentMessage, answers []string, edits []syncSentMessage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]syncSentMessage(nil), f.messages...), append([]string(nil), f.answers...), append([]syncSentMessage(nil), f.edits...)
}

type syncTestEnv struct {
	botSvc     *TelegramBotService
	settings   *SystemSettingsService
	serviceSvc *ServiceService
	homeID     int64
	currencyID int64
}

func newSyncTestEnv(t *testing.T) *syncTestEnv {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	settings := NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
	homeStorage := storage.NewHomeStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	billStorage := storage.NewBillStorage(database)
	homeSvc := NewHomeService(homeStorage)
	currencySvc := NewCurrencyService(currencyStorage)
	serviceSvc := NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)

	home, err := homeSvc.Create(ctx, "Casa Sync", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := currencySvc.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("monedas de seed: %v", err)
	}

	return &syncTestEnv{
		botSvc:     NewTelegramBotService(settings, billStorage, storage.NewDebtBillStorage(database), serviceStorage, NewAutomationClient(settings), nil, nil, nil, nil),
		settings:   settings,
		serviceSvc: serviceSvc,
		homeID:     home.ID,
		currencyID: currencies[0].ID,
	}
}

func (e *syncTestEnv) createService(t *testing.T, name string, accountID *int64) *appmodels.Service {
	t.Helper()
	svc, err := e.serviceSvc.Create(context.Background(), &appmodels.Service{
		HomeID:              e.homeID,
		Name:                name,
		CurrencyID:          e.currencyID,
		Frequency:           "monthly",
		SuggestedAmount:     100,
		Active:              true,
		IconKey:             "internet",
		BillingType:         "variable",
		AutomationAccountID: accountID,
	})
	if err != nil {
		t.Fatalf("crear servicio %q: %v", name, err)
	}
	return svc
}

func newSyncTestBot(t *testing.T, env *syncTestEnv, api *syncFakeAPI) *bot.Bot {
	t.Helper()
	b, err := bot.New("TESTTOKEN",
		bot.WithServerURL(api.server.URL),
		bot.WithNotAsyncHandlers(),
		bot.WithDefaultHandler(env.botSvc.handleDefault),
	)
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	b.RegisterHandler(bot.HandlerTypeMessageText, "sincronizar", bot.MatchTypeCommand, env.botSvc.handleSincronizarServicios)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, syncCallbackPrefix, bot.MatchTypePrefix, env.botSvc.handleSyncCallback)
	return b
}

func sincronizarUpdate(text string) *tgmodels.Update {
	cmdLen := len(text)
	if idx := strings.IndexByte(text, ' '); idx > 0 {
		cmdLen = idx
	}
	return &tgmodels.Update{
		ID: 1,
		Message: &tgmodels.Message{
			Chat: tgmodels.Chat{ID: 123},
			Text: text,
			Entities: []tgmodels.MessageEntity{
				{Type: tgmodels.MessageEntityTypeBotCommand, Offset: 0, Length: cmdLen},
			},
		},
	}
}

func syncCallbackUpdate(data string) *tgmodels.Update {
	return &tgmodels.Update{
		ID: 1,
		CallbackQuery: &tgmodels.CallbackQuery{
			ID:   "callback-1",
			Data: data,
			Message: tgmodels.MaybeInaccessibleMessage{
				Message: &tgmodels.Message{
					ID:   55,
					Chat: tgmodels.Chat{ID: 123},
					Text: "servicio",
				},
			},
		},
	}
}

func keyboardButtonData(t *testing.T, raw string) string {
	t.Helper()
	if raw == "" {
		return ""
	}
	var kb tgmodels.InlineKeyboardMarkup
	if err := json.Unmarshal([]byte(raw), &kb); err != nil {
		t.Fatalf("parse reply_markup: %v (raw=%s)", err, raw)
	}
	if len(kb.InlineKeyboard) == 0 || len(kb.InlineKeyboard[0]) == 0 {
		return ""
	}
	return kb.InlineKeyboard[0][0].CallbackData
}

// ---- Tests ----

func TestSyncableServicesFiltersByAutomationAccount(t *testing.T) {
	env := newSyncTestEnv(t)
	accountID := int64(7)
	withAccount := env.createService(t, "Internet Sync", &accountID)
	env.createService(t, "Sin Plugin", nil)

	got, err := env.botSvc.syncableServices(context.Background())
	if err != nil {
		t.Fatalf("syncableServices: %v", err)
	}
	if len(got) != 1 || got[0].ID != withAccount.ID {
		t.Fatalf("syncableServices = %v, want solo servicio %d", got, withAccount.ID)
	}
}

func TestSincronizarServiciosCommandListsServicesWithButtons(t *testing.T) {
	env := newSyncTestEnv(t)
	accountID := int64(7)
	svc := env.createService(t, "Internet Sync", &accountID)
	env.createService(t, "Sin Plugin", nil)

	if err := env.settings.SetAutomationConfig(context.Background(), "http://automation.test:8089", "key-1234567890"); err != nil {
		t.Fatalf("config automation: %v", err)
	}

	api := newSyncFakeAPI(t)
	b := newSyncTestBot(t, env, api)
	b.ProcessUpdate(context.Background(), sincronizarUpdate("/sincronizar"))

	messages, _, _ := api.snapshot()
	if len(messages) != 2 {
		t.Fatalf("mensajes = %d, want 2 (servicio + resumen): %+v", len(messages), messages)
	}
	if !strings.Contains(messages[0].Text, "Internet Sync") {
		t.Errorf("primer mensaje sin el servicio: %q", messages[0].Text)
	}
	if !strings.Contains(messages[0].Text, "Cuenta automation: 7") {
		t.Errorf("primer mensaje sin la cuenta de automation: %q", messages[0].Text)
	}
	if data := keyboardButtonData(t, messages[0].ReplyMarkup); data != fmt.Sprintf("sync:%d", svc.ID) {
		t.Errorf("callback_data = %q, want sync:%d", data, svc.ID)
	}
	for _, m := range messages {
		if strings.Contains(m.Text, "Sin Plugin") {
			t.Errorf("se listó un servicio sin automation: %q", m.Text)
		}
	}
	if !strings.Contains(messages[1].Text, "1 servicio") {
		t.Errorf("resumen inesperado: %q", messages[1].Text)
	}
}

func TestSincronizarServiciosCommandWithoutServices(t *testing.T) {
	env := newSyncTestEnv(t)
	if err := env.settings.SetAutomationConfig(context.Background(), "http://automation.test:8089", "key-1234567890"); err != nil {
		t.Fatalf("config automation: %v", err)
	}

	api := newSyncFakeAPI(t)
	b := newSyncTestBot(t, env, api)
	b.ProcessUpdate(context.Background(), sincronizarUpdate("/sincronizar"))

	messages, _, _ := api.snapshot()
	if len(messages) != 1 {
		t.Fatalf("mensajes = %d, want 1: %+v", len(messages), messages)
	}
	if !strings.Contains(messages[0].Text, "No hay servicios") {
		t.Errorf("mensaje inesperado: %q", messages[0].Text)
	}
}

func TestSincronizarServiciosCommandWithoutAutomationConfig(t *testing.T) {
	env := newSyncTestEnv(t)
	accountID := int64(7)
	env.createService(t, "Internet Sync", &accountID)

	api := newSyncFakeAPI(t)
	b := newSyncTestBot(t, env, api)
	b.ProcessUpdate(context.Background(), sincronizarUpdate("/sincronizar"))

	messages, _, _ := api.snapshot()
	if len(messages) != 1 {
		t.Fatalf("mensajes = %d, want 1: %+v", len(messages), messages)
	}
	if !strings.Contains(messages[0].Text, "Automation no está configurada") {
		t.Errorf("mensaje inesperado: %q", messages[0].Text)
	}
	if messages[0].ReplyMarkup != "" {
		t.Errorf("no debe enviar botones sin automation configurada: %q", messages[0].ReplyMarkup)
	}
}

func TestSyncCallbackSincronizaYEditaMensaje(t *testing.T) {
	env := newSyncTestEnv(t)
	accountID := int64(7)
	svc := env.createService(t, "Internet Sync", &accountID)

	var hits int
	automationSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("X-Webhook-Key") != "key-1234567890" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/accounts/7/webhook:run" {
			t.Errorf("path inesperado: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]int{"delivered": 3, "failed": 1})
	}))
	defer automationSrv.Close()
	if err := env.settings.SetAutomationConfig(context.Background(), automationSrv.URL, "key-1234567890"); err != nil {
		t.Fatalf("config automation: %v", err)
	}

	api := newSyncFakeAPI(t)
	b := newSyncTestBot(t, env, api)
	b.ProcessUpdate(context.Background(), syncCallbackUpdate(fmt.Sprintf("sync:%d", svc.ID)))

	_, answers, edits := api.snapshot()
	if len(answers) != 1 || answers[0] != "Sincronizando…" {
		t.Fatalf("answers = %v, want [Sincronizando…]", answers)
	}
	if hits != 1 {
		t.Fatalf("automation hits = %d, want 1", hits)
	}
	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1: %+v", len(edits), edits)
	}
	if !strings.Contains(edits[0].Text, "Entregadas: 3") || !strings.Contains(edits[0].Text, "Fallidas: 1") {
		t.Errorf("resultado inesperado: %q", edits[0].Text)
	}
	if data := keyboardButtonData(t, edits[0].ReplyMarkup); data != fmt.Sprintf("sync:%d", svc.ID) {
		t.Errorf("el botón debe conservarse; callback_data = %q", data)
	}
}

func TestSyncCallbackEvitaDuplicadoEnCurso(t *testing.T) {
	env := newSyncTestEnv(t)
	accountID := int64(7)
	svc := env.createService(t, "Internet Sync", &accountID)

	var hits int
	automationSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		json.NewEncoder(w).Encode(map[string]int{"delivered": 0, "failed": 0})
	}))
	defer automationSrv.Close()
	if err := env.settings.SetAutomationConfig(context.Background(), automationSrv.URL, "key-1234567890"); err != nil {
		t.Fatalf("config automation: %v", err)
	}

	if !env.botSvc.beginSync(svc.ID) {
		t.Fatal("beginSync debió adquirir el guard")
	}
	defer env.botSvc.endSync(svc.ID)

	api := newSyncFakeAPI(t)
	b := newSyncTestBot(t, env, api)
	b.ProcessUpdate(context.Background(), syncCallbackUpdate(fmt.Sprintf("sync:%d", svc.ID)))

	_, answers, edits := api.snapshot()
	if hits != 0 {
		t.Fatalf("no debió llamar a automation; hits = %d", hits)
	}
	if len(answers) != 1 || !strings.Contains(answers[0], "en curso") {
		t.Errorf("answers = %v, want aviso de sync en curso", answers)
	}
	if len(edits) != 0 {
		t.Errorf("no debió editar el mensaje: %+v", edits)
	}
}

func TestSyncCallbackServicioInexistente(t *testing.T) {
	env := newSyncTestEnv(t)
	if err := env.settings.SetAutomationConfig(context.Background(), "http://automation.test:8089", "key-1234567890"); err != nil {
		t.Fatalf("config automation: %v", err)
	}

	api := newSyncFakeAPI(t)
	b := newSyncTestBot(t, env, api)
	b.ProcessUpdate(context.Background(), syncCallbackUpdate("sync:99999"))

	_, _, edits := api.snapshot()
	if len(edits) != 1 || !strings.Contains(edits[0].Text, "ya no existe") {
		t.Fatalf("edits = %+v, want aviso de servicio inexistente", edits)
	}
}

func TestParseSyncCallbackData(t *testing.T) {
	cases := []struct {
		data string
		id   int64
		ok   bool
	}{
		{"sync:1", 1, true},
		{"sync:42", 42, true},
		{"sync:0", 0, false},
		{"sync:-3", 0, false},
		{"sync:abc", 0, false},
		{"bt:1", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		id, ok := parseSyncCallbackData(tc.data)
		if id != tc.id || ok != tc.ok {
			t.Errorf("parseSyncCallbackData(%q) = (%d, %v), want (%d, %v)", tc.data, id, ok, tc.id, tc.ok)
		}
	}
}

func TestBeginEndSyncGuard(t *testing.T) {
	env := newSyncTestEnv(t)
	if !env.botSvc.beginSync(1) {
		t.Fatal("primer beginSync debió devolver true")
	}
	if env.botSvc.beginSync(1) {
		t.Fatal("segundo beginSync del mismo servicio debió devolver false")
	}
	if !env.botSvc.beginSync(2) {
		t.Fatal("beginSync de otro servicio debió devolver true")
	}
	env.botSvc.endSync(1)
	if !env.botSvc.beginSync(1) {
		t.Fatal("tras endSync el servicio debe poder sincronizarse de nuevo")
	}
}
