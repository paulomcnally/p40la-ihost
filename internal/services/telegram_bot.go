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
	settings  *SystemSettingsService
	bills     *storage.BillStorage
	debtBills *storage.DebtBillStorage

	mu         sync.Mutex
	cancel     context.CancelFunc
	reloadCh   chan struct{}
	lastChatID int64 // último chat autorizado que interactuó (REQ-010, P2)
}

// NewTelegramBotService crea el servicio. Requiere arrancarlo con Start().
func NewTelegramBotService(settings *SystemSettingsService, bills *storage.BillStorage, debtBills *storage.DebtBillStorage) *TelegramBotService {
	return &TelegramBotService{
		settings:  settings,
		bills:     bills,
		debtBills: debtBills,
		reloadCh:  make(chan struct{}, 1),
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
	b.RegisterHandler(bot.HandlerTypeMessageText, "servicios_pendientes", bot.MatchTypeCommand, s.handleServiciosPendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "deudas_pendientes", bot.MatchTypeCommand, s.handleDeudasPendientes)
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
			{Command: "servicios_pendientes", Description: "Servicios con facturas pendientes"},
			{Command: "deudas_pendientes", Description: "Deudas con cuotas pendientes"},
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
	s.reply(ctx, b, update, "🤖 *p40la-ihost Bot*\n\nConsulta la base de datos del iHost (solo lectura).\n\nComandos:\n  `/servicios_pendientes` — servicios con facturas pendientes\n  `/deudas_pendientes` — deudas con cuotas pendientes")
}

func (s *TelegramBotService) handleDefault(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}
	s.reply(ctx, b, update, "Comando no reconocido. Usa `/servicios_pendientes` o `/deudas_pendientes`.")
}

// handleServiciosPendientes responde con un mensaje por cada servicio que
// tiene facturas pendientes y un mensaje final con los totales por moneda
// (REQ-001, REQ-003).
func (s *TelegramBotService) handleServiciosPendientes(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}

	pending, err := s.bills.ListPendingWithDetails(ctx)
	if err != nil {
		slog.Error("telegram_bot: /servicios_pendientes — error de storage", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al consultar las facturas pendientes. Revisa los logs.")
		return
	}

	currencyFormat, err := s.settings.GetCurrencyFormat(ctx)
	if err != nil {
		slog.Error("telegram_bot: /servicios_pendientes — error de formato de moneda", "error", err)
		currencyFormat = DefaultCurrencyFormat()
	}

	s.sendMany(ctx, b, update, formatServiciosPendientes(pending, currencyFormat))
}

// handleDeudasPendientes responde con un mensaje por cada deuda que tiene
// cuotas pendientes y un mensaje final con los totales por moneda
// (REQ-002, REQ-004).
func (s *TelegramBotService) handleDeudasPendientes(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}

	pending, err := s.debtBills.ListPendingWithDetails(ctx)
	if err != nil {
		slog.Error("telegram_bot: /deudas_pendientes — error de storage", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al consultar las deudas pendientes. Revisa los logs.")
		return
	}

	currencyFormat, err := s.settings.GetCurrencyFormat(ctx)
	if err != nil {
		slog.Error("telegram_bot: /deudas_pendientes — error de formato de moneda", "error", err)
		currencyFormat = DefaultCurrencyFormat()
	}

	s.sendMany(ctx, b, update, formatDeudasPendientes(pending, currencyFormat))
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

// sendMany envía varios mensajes de forma secuencial (SPEC-082). Si un envío
// falla se loguea y se continúa con el siguiente.
func (s *TelegramBotService) sendMany(ctx context.Context, b *bot.Bot, update *tgmodels.Update, texts []string) {
	if update.Message == nil {
		return
	}
	for _, text := range texts {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      text,
			ParseMode: tgmodels.ParseModeMarkdownV1,
		}); err != nil {
			slog.Warn("telegram_bot: enviar mensaje", "error", err)
		}
	}
}

// pendingGroup agrupa las facturas pendientes de un servicio para
// /servicios_pendientes.
type pendingGroup struct {
	service string
	home    string
	count   int
	amount  float64
	symbol  string
}

// formatServiciosPendientes construye un mensaje por cada servicio con
// facturas pendientes y agrega al final el mensaje de totales por moneda
// (SPEC-082). Si no hay pendientes devuelve un único mensaje.
func formatServiciosPendientes(pending []appmodels.PendingBillDetail, format CurrencyFormat) []string {
	if len(pending) == 0 {
		return []string{"✅ No hay facturas pendientes."}
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

	msgs := make([]string, 0, len(sorted)+1)
	for _, g := range sorted {
		msgs = append(msgs, fmt.Sprintf(
			"⚡ *%s*\n  Casa: %s\n  Facturas: %d\n  Pendiente: %s",
			g.service, g.home, g.count, formatAmount(g.amount, g.symbol, format),
		))
	}
	return append(msgs, formatServiciosTotales(pending, format))
}

// currencyTotal acumula el monto pendiente de una moneda para el mensaje de
// totales (SPEC-082).
type currencyTotal struct {
	code   string
	symbol string
	amount float64
}

// formatServiciosTotales construye el mensaje final de /servicios_pendientes
// con el total de facturas pendientes agrupado por moneda (REQ-003). Solo se
// muestran las monedas presentes en los datos.
func formatServiciosTotales(pending []appmodels.PendingBillDetail, format CurrencyFormat) string {
	totals := make(map[string]*currencyTotal)
	var order []string
	for _, p := range pending {
		code := p.CurrencyCode
		if code == "" {
			code = p.CurrencySymbol
		}
		t, ok := totals[code]
		if !ok {
			t = &currencyTotal{code: code, symbol: p.CurrencySymbol}
			totals[code] = t
			order = append(order, code)
		}
		t.amount += p.Amount
	}

	msg := "💰 *Totales — Facturas pendientes*\n"
	for _, code := range order {
		t := totals[code]
		msg += fmt.Sprintf("  %s: %s\n", t.code, formatAmount(t.amount, t.symbol, format))
	}
	return strings.TrimRight(msg, "\n")
}

// debtGroup agrupa las cuotas pendientes de una deuda para /deudas_pendientes.
type debtGroup struct {
	debt        string
	institution string
	count       int
	amount      float64
	symbol      string
}

// formatDeudasPendientes construye un mensaje por cada deuda con cuotas
// pendientes y agrega al final el mensaje de totales por moneda (SPEC-082).
// Si no hay pendientes devuelve un único mensaje.
func formatDeudasPendientes(pending []appmodels.PendingDebtDetail, format CurrencyFormat) []string {
	if len(pending) == 0 {
		return []string{"✅ No hay deudas pendientes."}
	}

	groups := make(map[int64]*debtGroup)
	var order []int64
	for _, p := range pending {
		g, ok := groups[p.DebtID]
		if !ok {
			g = &debtGroup{debt: p.DebtDescription, institution: p.InstitutionName, symbol: p.CurrencySymbol}
			groups[p.DebtID] = g
			order = append(order, p.DebtID)
		}
		g.count++
		g.amount += p.Amount
	}

	sorted := sortDebtGroups(groups, order)

	msgs := make([]string, 0, len(sorted)+1)
	for _, g := range sorted {
		msgs = append(msgs, fmt.Sprintf(
			"💳 *%s*\n  Institución: %s\n  Cuotas: %d\n  Pendiente: %s",
			g.debt, g.institution, g.count, formatAmount(g.amount, g.symbol, format),
		))
	}
	return append(msgs, formatDeudasTotales(pending, format))
}

// formatDeudasTotales construye el mensaje final de /deudas_pendientes con el
// total de cuotas pendientes agrupado por moneda (REQ-004). Solo se muestran
// las monedas presentes en los datos.
func formatDeudasTotales(pending []appmodels.PendingDebtDetail, format CurrencyFormat) string {
	totals := make(map[string]*currencyTotal)
	var order []string
	for _, p := range pending {
		code := p.CurrencyCode
		if code == "" {
			code = p.CurrencySymbol
		}
		t, ok := totals[code]
		if !ok {
			t = &currencyTotal{code: code, symbol: p.CurrencySymbol}
			totals[code] = t
			order = append(order, code)
		}
		t.amount += p.Amount
	}

	msg := "💰 *Totales — Cuotas pendientes*\n"
	for _, code := range order {
		t := totals[code]
		msg += fmt.Sprintf("  %s: %s\n", t.code, formatAmount(t.amount, t.symbol, format))
	}
	return strings.TrimRight(msg, "\n")
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

// sortDebtGroups ordena los grupos de deudas por monto total descendente.
func sortDebtGroups(groups map[int64]*debtGroup, order []int64) []*debtGroup {
	result := make([]*debtGroup, 0, len(order))
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
