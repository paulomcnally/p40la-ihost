package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// TelegramBotService — bot de Telegram embebido en el server (SPEC-079).
// Consulta la DB local directamente (via storages) y responde comandos por
// long polling. Solo inicia polling si la config tiene enabled=1 y token.
type TelegramBotService struct {
	settings *SystemSettingsService
	bills    *storage.BillStorage

	mu         sync.Mutex
	cancel     context.CancelFunc
	reloadCh   chan struct{}
	lastChatID int64 // último chat autorizado que interactuó (REQ-010, P2)
}

// NewTelegramBotService crea el servicio. Requiere arrancarlo con Start().
func NewTelegramBotService(settings *SystemSettingsService, bills *storage.BillStorage) *TelegramBotService {
	return &TelegramBotService{
		settings: settings,
		bills:    bills,
		reloadCh: make(chan struct{}, 1),
	}
}

// Start arranca el loop supervisor del bot en una goroutine. El loop evalúa
// la config periódicamente y (re)inicia el polling cuando está habilitado.
func (s *TelegramBotService) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.run(ctx)
}

// Stop cancela el contexto y detiene el polling (si está activo).
func (s *TelegramBotService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// NotifyConfigChanged avisa al supervisor que la config cambió para que
// arranque/detenga/reinicie el polling sin esperar el intervalo de chequeo.
func (s *TelegramBotService) NotifyConfigChanged() {
	select {
	case s.reloadCh <- struct{}{}:
	default:
	}
}

// run es el supervisor: loop infinito que decide si el bot debe estar activo.
func (s *TelegramBotService) run(ctx context.Context) {
	const idleCheck = 30 * time.Second
	const backoff = 15 * time.Second

	for {
		cfg, err := s.settings.GetTelegramBotConfig(ctx)
		if err != nil {
			slog.Error("telegram_bot: leer config", "error", err)
			if !wait(ctx, backoff) {
				return
			}
			continue
		}

		if !cfg.Enabled || strings.TrimSpace(cfg.Token) == "" {
			// Inactivo: esperar señal de cambio o re-chequeo periódico.
			select {
			case <-ctx.Done():
				return
			case <-s.reloadCh:
			case <-time.After(idleCheck):
			}
			continue
		}

		s.runBot(ctx, cfg)
		if ctx.Err() != nil {
			return
		}
		if !wait(ctx, backoff) {
			return
		}
	}
}

// runBot arranca el polling con la config dada y bloquea hasta que el bot
// se detiene (cancelación o error de getUpdates). Los errores se loguean y
// el supervisor reintenta con backoff.
func (s *TelegramBotService) runBot(ctx context.Context, cfg *appmodels.TelegramBotConfig) {
	b, err := bot.New(cfg.Token, bot.WithDefaultHandler(s.handleDefault))
	if err != nil {
		slog.Warn("telegram_bot: token inválido o sin acceso a Telegram (getMe falló)", "error", err)
		return
	}

	// SPEC-080: los patrones van SIN slash — el matcher de MatchTypeCommand de
	// go-telegram/bot compara data[Offset+1:Offset+Length] (sin la barra).
	b.RegisterHandler(bot.HandlerTypeMessageText, "pendientes", bot.MatchTypeCommand, s.handlePendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, s.handleStart)

	s.registerCommands(ctx, b)

	slog.Info("telegram_bot: iniciando polling", "chat_ids", len(cfg.ChatIDs))
	b.Start(ctx)
	slog.Info("telegram_bot: polling detenido")
}

// registerCommands sobrescribe la lista de comandos del bot en Telegram
// (SPEC-080). El bot Python anterior dejó 9 comandos obsoletos via
// set_my_commands; esta lista persiste server-side y se reemplaza aquí.
// Best-effort: si falla (red/API), solo se loguea y el polling continúa.
func (s *TelegramBotService) registerCommands(ctx context.Context, b *bot.Bot) {
	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := b.SetMyCommands(cmdCtx, &bot.SetMyCommandsParams{
		Commands: []tgmodels.BotCommand{
			{Command: "start", Description: "Bienvenida y comandos disponibles"},
			{Command: "pendientes", Description: "Servicios con facturas pendientes"},
		},
	})
	if err != nil {
		slog.Warn("telegram_bot: setMyCommands falló (el menú puede mostrar comandos viejos)", "error", err)
		return
	}
	slog.Info("telegram_bot: comandos registrados en Telegram")
}

// wait duerme hasta timeout o cancelación. Devuelve false si el contexto fue
// cancelado (el supervisor debe terminar).
func wait(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// isAuthorized indica si el chat está permitido. Si la allowlist está vacía,
// todos los chats son válidos (comportamiento heredado del bot Python).
func (s *TelegramBotService) isAuthorized(ctx context.Context, chatID int64) bool {
	cfg, err := s.settings.GetTelegramBotConfig(ctx)
	if err != nil {
		return false
	}
	if len(cfg.ChatIDs) == 0 {
		return true
	}
	id := strconv.FormatInt(chatID, 10)
	for _, c := range cfg.ChatIDs {
		if c == id {
			return true
		}
	}
	return false
}

func (s *TelegramBotService) handleStart(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}
	s.reply(ctx, b, update, "🤖 *p40la-ihost Bot*\n\nConsulta la base de datos del iHost (solo lectura).\n\nComandos:\n  `/pendientes` — servicios con facturas pendientes")
}

func (s *TelegramBotService) handleDefault(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}
	s.reply(ctx, b, update, "Comando no reconocido. Usa `/pendientes` para ver los servicios con facturas pendientes.")
}

// handlePendientes responde con los servicios que tienen facturas pendientes
// (REQ-004): nombre del servicio, casa, cantidad de facturas y monto total.
func (s *TelegramBotService) handlePendientes(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}

	pending, err := s.bills.ListPendingWithDetails(ctx)
	if err != nil {
		slog.Error("telegram_bot: /pendientes — error de storage", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al consultar las facturas pendientes. Revisa los logs.")
		return
	}

	currencyFormat, err := s.settings.GetCurrencyFormat(ctx)
	if err != nil {
		slog.Error("telegram_bot: /pendientes — error de formato de moneda", "error", err)
		currencyFormat = DefaultCurrencyFormat()
	}

	s.reply(ctx, b, update, formatPendientes(pending, currencyFormat))
}

// checkAuthorized valida el chat contra la allowlist y guarda el último chat
// autorizado. Devuelve false si el chat no está permitido (se ignora en
// silencio — REQ-008).
func (s *TelegramBotService) checkAuthorized(ctx context.Context, b *bot.Bot, update *tgmodels.Update) bool {
	if update.Message == nil {
		return false
	}
	chatID := update.Message.Chat.ID
	if !s.isAuthorized(ctx, chatID) {
		slog.Debug("telegram_bot: mensaje ignorado de chat no autorizado", "chat_id", chatID)
		return false
	}
	s.mu.Lock()
	s.lastChatID = chatID
	s.mu.Unlock()
	return true
}

func (s *TelegramBotService) reply(ctx context.Context, b *bot.Bot, update *tgmodels.Update, text string) {
	if update.Message == nil {
		return
	}
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: tgmodels.ParseModeMarkdownV1,
	}); err != nil {
		slog.Warn("telegram_bot: enviar mensaje", "error", err)
	}
}

// pendingGroup agrupa las facturas pendientes de un servicio para /pendientes.
type pendingGroup struct {
	service string
	home    string
	count   int
	amount  float64
	symbol  string
}

// formatPendientes construye el mensaje Markdown de /pendientes agrupando las
// facturas pendientes por servicio.
func formatPendientes(pending []appmodels.PendingBillDetail, format CurrencyFormat) string {
	if len(pending) == 0 {
		return "✅ No hay facturas pendientes."
	}

	groups := make(map[int64]*pendingGroup)
	var order []int64
	for _, p := range pending {
		g, ok := groups[p.ServiceID]
		if !ok {
			g = &pendingGroup{service: p.ServiceName, home: p.HomeName, symbol: p.CurrencySymbol}
			groups[p.ServiceID] = g
			order = append(order, p.ServiceID)
		}
		g.count++
		g.amount += p.Amount
	}

	sorted := sortGroups(groups, order)

	msg := fmt.Sprintf("⏳ *Servicios con facturas pendientes* (%d)\n", len(sorted))
	for _, g := range sorted {
		msg += fmt.Sprintf(
			"\n⚡ *%s*\n  Casa: %s\n  Facturas: %d\n  Pendiente: %s",
			g.service, g.home, g.count, formatAmount(g.amount, g.symbol, format),
		)
	}
	return msg
}

// sortGroups ordena los grupos por monto total descendente.
func sortGroups(groups map[int64]*pendingGroup, order []int64) []*pendingGroup {
	result := make([]*pendingGroup, 0, len(order))
	for _, id := range order {
		result = append(result, groups[id])
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].amount > result[i].amount {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}
