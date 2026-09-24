package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
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
	settings       *SystemSettingsService
	bills          *storage.BillStorage
	debtBills      *storage.DebtBillStorage
	serviceStorage *storage.ServiceStorage
	automation     *AutomationClient

	mu         sync.Mutex
	cancel     context.CancelFunc
	reloadCh   chan struct{}
	lastChatID int64 // último chat autorizado que interactuó (REQ-010, P2)

	botMu     sync.Mutex
	activeBot *bot.Bot // instancia del bot en polling (para alertas push, SPEC-088)
}

// NewTelegramBotService crea el servicio. Requiere arrancarlo con Start().
func NewTelegramBotService(settings *SystemSettingsService, bills *storage.BillStorage, debtBills *storage.DebtBillStorage, serviceStorage *storage.ServiceStorage, automation *AutomationClient) *TelegramBotService {
	return &TelegramBotService{
		settings:       settings,
		bills:          bills,
		debtBills:      debtBills,
		serviceStorage: serviceStorage,
		automation:     automation,
		reloadCh:       make(chan struct{}, 1),
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

	// La instancia del polling queda disponible para las alertas push
	// (SPEC-088): reutiliza la misma conexión y evita crear bots efímeros.
	s.botMu.Lock()
	s.activeBot = b
	s.botMu.Unlock()
	defer func() {
		s.botMu.Lock()
		s.activeBot = nil
		s.botMu.Unlock()
	}()

	// SPEC-080: los patrones van SIN slash — el matcher de MatchTypeCommand de
	// go-telegram/bot compara data[Offset+1:Offset+Length] (sin la barra).
	b.RegisterHandler(bot.HandlerTypeMessageText, "servicios_pendientes", bot.MatchTypeCommand, s.handleServiciosPendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "deudas_pendientes", bot.MatchTypeCommand, s.handleDeudasPendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "sincronizar_servicio", bot.MatchTypeCommand, s.handleSincronizarServicio)
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
			{Command: "sincronizar_servicio", Description: "Sincronizar un servicio con su ID (ej: /sincronizar_servicio 1)"},
		},
	})
	if err != nil {
		slog.Warn("telegram_bot: setMyCommands falló (el menú puede mostrar comandos viejos)", "error", err)
		return
	}
	slog.Info("telegram_bot: comandos registrados en Telegram")
}

// SendAlerts envía mensajes proactivos a TODOS los chat_ids autorizados
// (SPEC-088). No depende de un update de Telegram: es el canal push de los
// schedulers. Gate defensivo: si el bot no está habilitado, sin token o sin
// chat_ids, NO envía nada y jamás contacta la API de Telegram (el dispatch
// ya corta antes, ver dispatchTelegram).
func (s *TelegramBotService) SendAlerts(ctx context.Context, texts []string) error {
	if len(texts) == 0 {
		return nil
	}
	cfg, err := s.settings.GetTelegramBotConfig(ctx)
	if err != nil {
		return fmt.Errorf("leer config del bot: %w", err)
	}
	if !cfg.Enabled || strings.TrimSpace(cfg.Token) == "" {
		slog.Debug("telegram_bot: bot deshabilitado o sin token, no se envían alertas")
		return nil
	}
	if len(cfg.ChatIDs) == 0 {
		slog.Warn("telegram_bot: sin chat_ids autorizados, no se envían alertas")
		return nil
	}

	b := s.currentBot()
	if b == nil {
		// Polling inactivo (p.ej. durante el arranque): instancia efímera
		// solo para el envío.
		b, err = bot.New(cfg.Token, bot.WithDefaultHandler(s.handleDefault))
		if err != nil {
			return err
		}
	}

	for _, chatID := range cfg.ChatIDs {
		id, err := strconv.ParseInt(chatID, 10, 64)
		if err != nil {
			slog.Warn("telegram_bot: chat_id inválido", "chat_id", chatID)
			continue
		}
		for _, text := range texts {
			if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    id,
				Text:      text,
				ParseMode: tgmodels.ParseModeMarkdownV1,
			}); err != nil {
				slog.Warn("telegram_bot: enviar alerta", "chat_id", id, "error", err)
			}
		}
	}
	return nil
}

// currentBot devuelve la instancia del bot en polling, o nil si no hay.
func (s *TelegramBotService) currentBot() *bot.Bot {
	s.botMu.Lock()
	defer s.botMu.Unlock()
	return s.activeBot
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
	s.reply(ctx, b, update, "🤖 *p40la-ihost Bot*\n\nConsulta la base de datos del iHost (solo lectura) y sincronización manual.\n\nComandos:\n  `/servicios_pendientes` — servicios con facturas pendientes\n  `/deudas_pendientes` — deudas con cuotas pendientes\n  `/sincronizar_servicio <id>` — sincroniza un servicio con su ID (ej: `/sincronizar_servicio 1`)\n\nEl ID de cada servicio y deuda se muestra en las listas de la app (badge `ID: N`).\n\nAdemás, si activás las alertas por Telegram en P40LA (Configuración → Alertas), este bot te envía los avisos automáticos directamente.")
}

func (s *TelegramBotService) handleDefault(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}
	s.reply(ctx, b, update, "Comando no reconocido. Usa `/servicios_pendientes`, `/deudas_pendientes` o `/sincronizar_servicio <id>`.")
}

// handleSincronizarServicio ejecuta la sincronización manual de un servicio
// (SPEC-092): `/sincronizar_servicio <id>` dispara el job de automation
// (plugin + webhook) y responde con el resultado. El ID es el badge `ID: N`
// de la lista de servicios.
func (s *TelegramBotService) handleSincronizarServicio(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}

	text := ""
	if update.Message != nil {
		text = strings.TrimSpace(update.Message.Text)
	}
	parts := strings.Fields(text)
	if len(parts) < 2 {
		s.reply(ctx, b, update, "Uso: `/sincronizar_servicio <id>`\n\nSincroniza un servicio con su ID (lo encontrás en la lista de servicios, badge `ID: N`). Ej: `/sincronizar_servicio 1`")
		return
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		s.reply(ctx, b, update, "⚠️ El ID debe ser un número entero positivo. Uso: `/sincronizar_servicio <id>`")
		return
	}

	svc, err := s.serviceStorage.GetByID(ctx, id)
	if err != nil {
		slog.Error("telegram_bot: /sincronizar_servicio — error de storage", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al consultar el servicio. Revisá los logs.")
		return
	}
	if svc == nil {
		s.reply(ctx, b, update, fmt.Sprintf("⚠️ No existe un servicio con ID %d.", id))
		return
	}
	if svc.AutomationAccountID == nil {
		s.reply(ctx, b, update, fmt.Sprintf("⚠️ El servicio *%s* no tiene vinculada una cuenta de automation. Editá el servicio en la app y configurá el ID de cuenta en Automation.", svc.Name))
		return
	}

	configured, err := s.settings.IsAutomationConfigured(ctx)
	if err != nil {
		slog.Error("telegram_bot: /sincronizar_servicio — leer config automation", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al verificar la configuración de Automation.")
		return
	}
	if !configured {
		s.reply(ctx, b, update, "⚠️ Automation no está configurada. Configurala en P40LA → Configuración → Automation.")
		return
	}

	delivered, failed, err := s.automation.SyncAccount(ctx, *svc.AutomationAccountID)
	if err != nil {
		s.reply(ctx, b, update, fmt.Sprintf("⚠️ Sincronización fallida: %v", err))
		return
	}
	msg := fmt.Sprintf("✅ Sincronización de *%s*\n  Entregadas: %d\n  Fallidas: %d", svc.Name, delivered, failed)
	s.reply(ctx, b, update, msg)
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

	// "Hoy" en la zona horaria configurada (SPEC-084, ADR-004); fallback UTC.
	loc, err := s.settings.GetTimezoneLocation(ctx)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	// Cantidad de guiones del separador (SPEC-084 REQ-014).
	sepLen, err := s.settings.GetTelegramBotSeparatorLength(ctx)
	if err != nil {
		sepLen = DefaultTelegramBotSeparatorLength
	}
	// Rango de meses futuros a mostrar (SPEC-084 REQ-016).
	showMonths, err := s.settings.GetTelegramBotShowMonths(ctx)
	if err != nil {
		showMonths = DefaultTelegramBotShowMonths
	}

	s.sendMany(ctx, b, update, formatServiciosPendientes(pending, currencyFormat, now, sepLen, showMonths))
}

// handleDeudasPendientes responde con un mensaje por cada deuda que tiene
// cuotas pendientes dentro del rango configurado (showMonths) y un mensaje
// final con los totales por moneda (REQ-002, REQ-004). El formato replica el
// de /servicios_pendientes (SPEC-086): semáforo, fecha legible, espaciado,
// separador configurable y negritas.
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

	// "Hoy" en la zona horaria configurada (SPEC-084, ADR-004); fallback UTC.
	loc, err := s.settings.GetTimezoneLocation(ctx)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	// Cantidad de guiones del separador (SPEC-084 REQ-014).
	sepLen, err := s.settings.GetTelegramBotSeparatorLength(ctx)
	if err != nil {
		sepLen = DefaultTelegramBotSeparatorLength
	}
	// Rango de meses futuros a mostrar (SPEC-084 REQ-016).
	showMonths, err := s.settings.GetTelegramBotShowMonths(ctx)
	if err != nil {
		showMonths = DefaultTelegramBotShowMonths
	}

	s.sendMany(ctx, b, update, formatDeudasPendientes(pending, currencyFormat, now, sepLen, showMonths))
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
	bills   []pendingBill // facturas itemizadas, SPEC-084
}

// pendingBill es una factura pendiente itemizada con su estado de
// vencimiento para /servicios_pendientes (SPEC-084).
type pendingBill struct {
	label  string // fecha legible ("07 sep 2026") o periodo ("ago 2026")
	status string // semáforo + texto ("🔴 Vencida hace 3 días")
	days   int    // días hasta vencimiento (negativo si vencida)
	hasDue bool   // true si tiene due_date (las sin fecha van al final)
	amount float64
}

// maxBillsPerMessage limita la cantidad de facturas itemizadas por mensaje
// para respetar el límite de 4096 caracteres de Telegram (SPEC-084, ADR-002).
// Mismo valor que maxInstallmentsPerMessage (SPEC-083).
const maxBillsPerMessage = 25

// esMonthAbbr son los meses en español abreviados para fechas legibles
// (SPEC-084, REQ-010).
var esMonthAbbr = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

// billDueDate devuelve la fecha de vencimiento real de la factura y su label
// legible, o false en hasDue si no tiene due_date (SPEC-084, ADR-001).
func billDueDate(p appmodels.PendingBillDetail) (string, bool) {
	if p.DueDate == nil {
		return "", false
	}
	return billDueDateLabel(*p.DueDate)
}

// billDueDateLabel devuelve la fecha legible ("07 sep 2026") de un string
// YYYY-MM-DD, o false si está vacía o es inválida (SPEC-084, REQ-010).
func billDueDateLabel(dueDate string) (string, bool) {
	if dueDate == "" {
		return "", false
	}
	t, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%02d %s %d", t.Day(), esMonthAbbr[t.Month()-1], t.Year()), true
}

// periodLabel construye el label de periodo (año-mes) de una factura sin
// due_date: "ago 2026" (SPEC-084, ADR-001).
func periodLabel(p appmodels.PendingBillDetail) string {
	if p.Month < 1 || p.Month > 12 {
		return fmt.Sprintf("%04d", p.Year)
	}
	return fmt.Sprintf("%s %d", esMonthAbbr[p.Month-1], p.Year)
}

// daysUntilDue devuelve los días calendario entre hoy (en la zona del server,
// ADR-004) y la fecha de vencimiento. Negativo si ya venció.
func daysUntilDue(dueDate string, now time.Time) int {
	t, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return 0
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	due := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
	return int(due.Sub(today).Hours() / 24)
}

// billStatus construye el semáforo de vencimiento de una factura con la misma
// regla que DueDateBadge (SPEC-081 REQ-008): verde >10 días, amarillo 1-10,
// rojo vence hoy o vencida. Sin due_date → 🟢 neutro (ADR-003).
func billStatus(days int, hasDue bool) string {
	if !hasDue {
		return "🟢 Sin fecha"
	}
	switch {
	case days > 10:
		return fmt.Sprintf("🟢 Vence en %d %s", days, dayWord(days))
	case days >= 1:
		return fmt.Sprintf("🟡 Vence en %d %s", days, dayWord(days))
	case days == 0:
		return "🔴 Vence hoy"
	default:
		return fmt.Sprintf("🔴 Vencida hace %d %s", -days, dayWord(-days))
	}
}

// dayWord devuelve "día" o "días" según la cantidad (singular/plural).
func dayWord(n int) string {
	if n == 1 {
		return "día"
	}
	return "días"
}

// currentMonthEnd devuelve el último día del mes de now (en su zona horaria).
func currentMonthEnd(now time.Time) time.Time {
	firstNext := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	return firstNext.AddDate(0, 0, -1)
}

// dueInRange indica si una fecha de vencimiento cae dentro del rango a
// mostrar (SPEC-084 REQ-012/016, SPEC-086 REQ-006): las vencidas se muestran
// SIEMPRE sin importar su antigüedad; las no vencidas solo si su due_date cae
// dentro de los próximos showMonths meses desde el mes actual (showMonths=1 →
// solo mes actual). Una fecha vacía o inválida siempre se mantiene.
func dueInRange(dueDate string, now time.Time, showMonths int) bool {
	if dueDate == "" {
		return true
	}
	t, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return true
	}
	due := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
	// Vencida → siempre.
	if due.Before(now) {
		return true
	}
	// No vencida: límite = fin del mes actual + (showMonths-1) meses.
	limit := currentMonthEnd(now).AddDate(0, showMonths-1, 0)
	return !due.After(limit)
}

// isCurrentPeriod indica si una factura es "del presente" para
// /servicios_pendientes (REQ-012, REQ-016). Las facturas sin due_date
// (periodo actual) siempre se mantienen.
func isCurrentPeriod(p appmodels.PendingBillDetail, now time.Time, showMonths int) bool {
	if p.DueDate == nil || *p.DueDate == "" {
		return true
	}
	return dueInRange(*p.DueDate, now, showMonths)
}

// formatServiciosPendientes construye un mensaje por cada servicio con
// facturas pendientes del periodo actual (REQ-012/016), itemizando cada
// factura con su semáforo de vencimiento (SPEC-084), y agrega al final el
// mensaje de totales por moneda (SPEC-082). Si un servicio tiene más de
// maxBillsPerMessage facturas, su mensaje se particiona en varios bloques
// (ADR-002). sepLen es la cantidad de guiones del separador del resumen
// (REQ-014; 0 = sin separador). Si no hay pendientes devuelve un único
// mensaje.
func formatServiciosPendientes(pending []appmodels.PendingBillDetail, format CurrencyFormat, now time.Time, sepLen, showMonths int) []string {
	filtered := make([]appmodels.PendingBillDetail, 0, len(pending))
	for _, p := range pending {
		if isCurrentPeriod(p, now, showMonths) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return []string{"✅ No hay facturas pendientes."}
	}

	groups := make(map[int64]*pendingGroup)
	var order []int64
	for _, p := range filtered {
		g, ok := groups[p.ServiceID]
		if !ok {
			g = &pendingGroup{service: p.ServiceName, home: p.HomeName, symbol: p.CurrencySymbol}
			groups[p.ServiceID] = g
			order = append(order, p.ServiceID)
		}
		g.count++
		g.amount += p.Amount

		dueLabel, hasDue := billDueDate(p)
		if !hasDue {
			dueLabel = periodLabel(p)
		}
		var days int
		if hasDue {
			days = daysUntilDue(*p.DueDate, now)
		}
		g.bills = append(g.bills, pendingBill{
			label:  dueLabel,
			status: billStatus(days, hasDue),
			days:   days,
			hasDue: hasDue,
			amount: p.Amount,
		})
	}

	sorted := sortGroups(groups, order)

	msgs := make([]string, 0, len(sorted)+1)
	for _, g := range sorted {
		msgs = append(msgs, formatServiceGroup(g, format, sepLen)...)
	}
	return append(msgs, formatServiciosTotales(filtered, format))
}

// formatServiceGroup construye los mensajes de un servicio itemizando sus
// facturas pendientes (REQ-001) ordenadas por urgencia (REQ-011). Si superan
// el límite por mensaje, particiona en varios bloques (REQ-005): el
// encabezado va en el primer bloque y el resumen al final del último
// (REQ-002, ADR-002). sepLen define los guiones del separador (REQ-014).
func formatServiceGroup(g *pendingGroup, format CurrencyFormat, sepLen int) []string {
	sortBillsByDue(g.bills)
	chunks := chunkBills(g.bills, maxBillsPerMessage)
	msgs := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		var b strings.Builder
		if i == 0 {
			fmt.Fprintf(&b, "⚡ *%s* — %s\n\n", g.service, g.home)
		}
		for _, bill := range chunk {
			fmt.Fprintf(&b, "  %s\n  📅 %s — %s\n\n", bill.status, bill.label, formatAmount(bill.amount, g.symbol, format))
		}
		if i == len(chunks)-1 {
			if sepLen > 0 {
				fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", sepLen))
			}
			fmt.Fprintf(&b, "  *Facturas*: %d\n  *Pendiente*: %s", g.count, formatAmount(g.amount, g.symbol, format))
		}
		msgs = append(msgs, strings.TrimRight(b.String(), "\n"))
	}
	return msgs
}

// sortBillsByDue ordena las facturas por urgencia (REQ-011): con due_date
// primero, por fecha de vencimiento ascendente (más vencida primero); las sin
// fecha al final, en su orden original.
func sortBillsByDue(bills []pendingBill) {
	sort.SliceStable(bills, func(i, j int) bool {
		if bills[i].hasDue != bills[j].hasDue {
			return bills[i].hasDue
		}
		if !bills[i].hasDue {
			return false
		}
		return bills[i].days < bills[j].days
	})
}

// chunkBills particiona la lista de facturas en bloques de tamaño max.
// Devuelve un único bloque si la lista no excede el máximo.
func chunkBills(bills []pendingBill, max int) [][]pendingBill {
	if len(bills) <= max {
		return [][]pendingBill{bills}
	}
	var chunks [][]pendingBill
	for start := 0; start < len(bills); start += max {
		end := start + max
		if end > len(bills) {
			end = len(bills)
		}
		chunks = append(chunks, bills[start:end])
	}
	return chunks
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
	bills       []pendingBill // cuotas itemizadas, SPEC-086 (layout de SPEC-084)
}

// maxInstallmentsPerMessage limita la cantidad de cuotas itemizadas por
// mensaje para respetar el límite de 4096 caracteres de Telegram y mantener
// mensajes legibles en móvil (SPEC-083, ADR-001).
const maxInstallmentsPerMessage = 25

// formatDeudasPendientes construye los mensajes de /deudas_pendientes
// itemizando cada cuota pendiente por deuda con el mismo formato de
// /servicios_pendientes (SPEC-086): semáforo de vencimiento, fecha legible,
// línea en blanco entre cuotas, separador configurable y resumen en negrita.
// El filtro usa dueInRange (REQ-006). Agrega al final el mensaje de totales
// por moneda (SPEC-082). Si no hay pendientes en rango devuelve un único
// mensaje.
func formatDeudasPendientes(pending []appmodels.PendingDebtDetail, format CurrencyFormat, now time.Time, sepLen, showMonths int) []string {
	filtered := make([]appmodels.PendingDebtDetail, 0, len(pending))
	for _, p := range pending {
		if dueInRange(p.DueDate, now, showMonths) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return []string{"✅ No hay deudas pendientes."}
	}

	groups := make(map[int64]*debtGroup)
	var order []int64
	for _, p := range filtered {
		g, ok := groups[p.DebtID]
		if !ok {
			g = &debtGroup{debt: p.DebtDescription, institution: p.InstitutionName, symbol: p.CurrencySymbol}
			groups[p.DebtID] = g
			order = append(order, p.DebtID)
		}
		g.count++
		g.amount += p.Amount

		dueLabel, hasDue := billDueDateLabel(p.DueDate)
		var days int
		if hasDue {
			days = daysUntilDue(p.DueDate, now)
		}
		g.bills = append(g.bills, pendingBill{
			label:  dueLabel,
			status: billStatus(days, hasDue),
			days:   days,
			hasDue: hasDue,
			amount: p.Amount,
		})
	}

	sorted := sortDebtGroups(groups, order)

	msgs := make([]string, 0, len(sorted)+1)
	for _, g := range sorted {
		msgs = append(msgs, formatDebtGroup(g, format, sepLen)...)
	}
	return append(msgs, formatDeudasTotales(filtered, format))
}

// formatDebtGroup construye los mensajes de una deuda itemizando sus cuotas
// pendientes con el layout de formatServiceGroup (SPEC-086): encabezado 💳 en
// el primer bloque, cuota con semáforo + fecha legible + línea en blanco,
// separador y resumen *Cuotas*/*Pendiente* al final del último bloque.
func formatDebtGroup(g *debtGroup, format CurrencyFormat, sepLen int) []string {
	sortBillsByDue(g.bills)
	chunks := chunkBills(g.bills, maxInstallmentsPerMessage)
	msgs := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		var b strings.Builder
		if i == 0 {
			fmt.Fprintf(&b, "💳 *%s*\n  *Institución*: %s\n\n", g.debt, g.institution)
		}
		for _, bill := range chunk {
			fmt.Fprintf(&b, "  %s\n  📅 %s — %s\n\n", bill.status, bill.label, formatAmount(bill.amount, g.symbol, format))
		}
		if i == len(chunks)-1 {
			if sepLen > 0 {
				fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", sepLen))
			}
			fmt.Fprintf(&b, "  *Cuotas*: %d\n  *Pendiente*: %s", g.count, formatAmount(g.amount, g.symbol, format))
		}
		msgs = append(msgs, strings.TrimRight(b.String(), "\n"))
	}
	return msgs
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
