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

	// Dependencias del módulo de presupuesto para /budget_transaction
	// (SPEC-097). Solo lectura de catálogos + creación validada de transacciones.
	budgetGroups *CategoryGroupService
	accounts     *AccountService
	currencies   *CurrencyService
	transactions *BudgetTransactionService

	mu         sync.Mutex
	cancel     context.CancelFunc
	reloadCh   chan struct{}
	lastChatID int64 // último chat autorizado que interactuó (REQ-010, P2)

	botMu     sync.Mutex
	activeBot *bot.Bot // instancia del bot en polling (para alertas push, SPEC-088)

	// Sesiones de conversación guiada de /budget_transaction (SPEC-097),
	// en memoria por chat_id con TTL. Protegidas por txMu.
	txMu      sync.Mutex
	txSession map[int64]*budgetTxSession
}

// NewTelegramBotService crea el servicio. Requiere arrancarlo con Start().
// Los parámetros del módulo de presupuesto (SPEC-097) pueden ser nil: en ese
// caso el comando /budget_transaction se degrada a un aviso de no disponible.
func NewTelegramBotService(
	settings *SystemSettingsService,
	bills *storage.BillStorage,
	debtBills *storage.DebtBillStorage,
	serviceStorage *storage.ServiceStorage,
	automation *AutomationClient,
	budgetGroups *CategoryGroupService,
	accounts *AccountService,
	currencies *CurrencyService,
	transactions *BudgetTransactionService,
) *TelegramBotService {
	return &TelegramBotService{
		settings:       settings,
		bills:          bills,
		debtBills:      debtBills,
		serviceStorage: serviceStorage,
		automation:     automation,
		budgetGroups:   budgetGroups,
		accounts:       accounts,
		currencies:     currencies,
		transactions:   transactions,
		reloadCh:       make(chan struct{}, 1),
		txSession:      make(map[int64]*budgetTxSession),
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
	b.RegisterHandler(bot.HandlerTypeMessageText, "budget_transaction", bot.MatchTypeCommand, s.handleBudgetTransaction)
	// Botones inline de la conversación /budget_transaction (SPEC-097, ADR-003).
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "bt:", bot.MatchTypePrefix, s.handleBudgetCallback)

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
			{Command: "budget_transaction", Description: "Registrar una transacción del presupuesto"},
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
	s.reply(ctx, b, update, "🤖 *p40la-ihost Bot*\n\nConsulta la base de datos del iHost (solo lectura) y sincronización manual.\n\nComandos:\n  `/servicios_pendientes` — servicios con facturas pendientes\n  `/deudas_pendientes` — deudas con cuotas pendientes\n  `/sincronizar_servicio <id>` — sincroniza un servicio con su ID (ej: `/sincronizar_servicio 1`)\n  `/budget_transaction` — registra una transacción del presupuesto (guía paso a paso)\n\nEl ID de cada servicio y deuda se muestra en las listas de la app (badge `ID: N`).\n\nAdemás, si activás las alertas por Telegram en P40LA (Configuración → Alertas), este bot te envía los avisos automáticos directamente.")
}

func (s *TelegramBotService) handleDefault(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}
	// Si hay una conversación /budget_transaction activa, el mensaje es una
	// respuesta a un paso de texto (SPEC-097). Si no, cae en el aviso por defecto.
	if s.budgetHandleText(ctx, b, update) {
		return
	}
	s.reply(ctx, b, update, "Comando no reconocido. Usa `/servicios_pendientes`, `/deudas_pendientes`, `/sincronizar_servicio <id>` o `/budget_transaction`.")
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

// ---------------------------------------------------------------------------
// /budget_transaction — conversación guiada para registrar una transacción del
// presupuesto (SPEC-097). Máquina de estados en memoria por chat_id (ADR-001)
// con pasos: grupo → categoría → tipo → monto → fecha → cuenta → payee →
// confirmación. Los pasos de selección usan botones inline; monto/fecha/payee
// se ingresan por texto y caen en handleDefault → budgetHandleText.
// ---------------------------------------------------------------------------

// budgetTxStep es la etapa actual de una conversación /budget_transaction.
type budgetTxStep int

const (
	stepGroup budgetTxStep = iota
	stepCategory
	stepType
	stepAmount
	stepDate
	stepAccount
	stepPayee
	stepConfirm
)

// budgetTxSession guarda el estado en memoria de una conversación
// /budget_transaction para un chat (SPEC-097, ADR-001).
type budgetTxSession struct {
	chatID         int64
	step           budgetTxStep
	groupID        int64
	categoryID     int64
	categoryName   string
	isInflow       bool
	amount         float64
	date           string
	accountID      int64
	accountName    string
	currencyID     int64
	currencySymbol string
	payee          string
	expiresAt      time.Time
}

// budgetTxTTL es el tiempo máximo de inactividad de una sesión antes de
// descartarse (SPEC-097 REQ-012).
const budgetTxTTL = 15 * time.Minute

// handleBudgetTransaction inicia (o reinicia) la conversación /budget_transaction
// (SPEC-097 REQ-001). Valida prerequisitos antes de arrancar (REQ-002).
func (s *TelegramBotService) handleBudgetTransaction(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID

	if s.budgetGroups == nil || s.accounts == nil || s.transactions == nil {
		s.reply(ctx, b, update, "⚠️ El módulo de presupuesto no está disponible en este servidor.")
		return
	}

	groups, err := s.budgetGroups.List(ctx)
	if err != nil {
		slog.Error("telegram_bot: /budget_transaction — listar grupos", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al leer los grupos de presupuesto. Revisá los logs.")
		return
	}
	hasCategories := false
	for i := range groups {
		if len(groups[i].Categories) > 0 {
			hasCategories = true
			break
		}
	}
	if !hasCategories {
		s.reply(ctx, b, update, "⚠️ Aún no hay categorías de presupuesto. Creá un grupo y una categoría en P40LA → Presupuesto → Vista mensual.")
		return
	}
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		slog.Error("telegram_bot: /budget_transaction — listar cuentas", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al leer las cuentas. Revisá los logs.")
		return
	}
	if len(accounts) == 0 {
		s.reply(ctx, b, update, "⚠️ Aún no hay cuentas. Creá una cuenta en P40LA → Presupuesto → Transacciones.")
		return
	}

	s.budgetCleanup()
	session := &budgetTxSession{
		chatID:    chatID,
		step:      stepGroup,
		expiresAt: time.Now().Add(budgetTxTTL),
	}
	s.txMu.Lock()
	s.txSession[chatID] = session
	s.txMu.Unlock()

	s.budgetSend(ctx, b, chatID, "📝 *Nueva transacción de presupuesto*\n\nElegí el *grupo*:", s.budgetGroupKeyboard(groups))
}

// handleBudgetCallback procesa los botones inline `bt:...` de /budget_transaction
// (SPEC-097 REQ-003..010). Valida la sesión activa y que el callback corresponda
// al paso actual; los callbacks fuera de paso se ignoran en silencio.
func (s *TelegramBotService) handleBudgetCallback(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.CallbackQuery == nil {
		return
	}
	msg := update.CallbackQuery.Message.Message
	if msg == nil {
		// Mensaje inaccesible: responder el callback y salir (no hay Chat/MessageID).
		s.answerCallback(ctx, b, update, "")
		return
	}
	chatID := msg.Chat.ID
	if !s.isAuthorized(ctx, chatID) {
		return
	}
	s.mu.Lock()
	s.lastChatID = chatID
	s.mu.Unlock()

	s.txMu.Lock()
	session := s.txSession[chatID]
	if session != nil {
		session.expiresAt = time.Now().Add(budgetTxTTL)
	}
	s.txMu.Unlock()

	if session == nil {
		s.answerCallback(ctx, b, update, "La sesión expiró o no existe. Usá /budget_transaction para empezar.")
		return
	}

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) < 2 || parts[0] != "bt" {
		s.answerCallback(ctx, b, update, "")
		return
	}
	msgID := msg.ID

	switch parts[1] {
	case "cancel":
		s.txMu.Lock()
		delete(s.txSession, chatID)
		s.txMu.Unlock()
		s.answerCallback(ctx, b, update, "Operación cancelada.")
		s.budgetClearKeyboard(ctx, b, chatID, msgID)

	case "grp":
		if session.step != stepGroup || len(parts) < 3 {
			s.answerCallback(ctx, b, update, "")
			return
		}
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			s.answerCallback(ctx, b, update, "")
			return
		}
		group, err := s.budgetGroupByID(ctx, id)
		if err != nil || group == nil {
			s.answerCallback(ctx, b, update, "Grupo no encontrado.")
			return
		}
		session.groupID = group.ID
		session.step = stepCategory
		s.answerCallback(ctx, b, update, "")
		s.budgetClearKeyboard(ctx, b, chatID, msgID)
		s.budgetSend(ctx, b, chatID, fmt.Sprintf("Grupo: *%s*\n\nElegí la *categoría*:", s.budgetGroupLabel(group)), s.budgetCategoryKeyboard(group.Categories))

	case "cat":
		if session.step != stepCategory || len(parts) < 3 {
			s.answerCallback(ctx, b, update, "")
			return
		}
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			s.answerCallback(ctx, b, update, "")
			return
		}
		cat, err := s.budgetCategoryByID(ctx, id)
		if err != nil || cat == nil {
			s.answerCallback(ctx, b, update, "Categoría no encontrada.")
			return
		}
		session.categoryID = cat.ID
		session.categoryName = cat.Name
		session.step = stepType
		s.answerCallback(ctx, b, update, "")
		s.budgetClearKeyboard(ctx, b, chatID, msgID)
		s.budgetSend(ctx, b, chatID, fmt.Sprintf("Categoría: *%s*\n\n¿Es un *gasto* o un *ingreso*?", session.categoryName), s.budgetTypeKeyboard())

	case "type":
		if session.step != stepType || len(parts) < 3 {
			s.answerCallback(ctx, b, update, "")
			return
		}
		switch parts[2] {
		case "outflow":
			session.isInflow = false
		case "inflow":
			session.isInflow = true
		default:
			s.answerCallback(ctx, b, update, "")
			return
		}
		session.step = stepAmount
		s.answerCallback(ctx, b, update, "")
		s.budgetClearKeyboard(ctx, b, chatID, msgID)
		s.budgetSend(ctx, b, chatID, s.budgetAmountPrompt(session), nil)

	case "acct":
		if session.step != stepAccount || len(parts) < 3 {
			s.answerCallback(ctx, b, update, "")
			return
		}
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			s.answerCallback(ctx, b, update, "")
			return
		}
		acc, err := s.budgetAccountByID(ctx, id)
		if err != nil || acc == nil {
			s.answerCallback(ctx, b, update, "Cuenta no encontrada.")
			return
		}
		session.accountID = acc.ID
		session.accountName = acc.Name
		session.currencyID = acc.CurrencyID
		cur, err := s.currencyByID(ctx, acc.CurrencyID)
		if err != nil || cur == nil {
			s.answerCallback(ctx, b, update, "Moneda de la cuenta no encontrada.")
			return
		}
		session.currencySymbol = cur.Symbol
		session.step = stepPayee
		s.answerCallback(ctx, b, update, "")
		s.budgetClearKeyboard(ctx, b, chatID, msgID)
		s.budgetSend(ctx, b, chatID, fmt.Sprintf("Cuenta: *%s*\n\n¿*Concepto/payee*? (opcional — escribí algo o /saltar)", session.accountName), nil)

	case "ok":
		if session.step != stepConfirm {
			s.answerCallback(ctx, b, update, "")
			return
		}
		s.answerCallback(ctx, b, update, "Guardando…")
		s.budgetClearKeyboard(ctx, b, chatID, msgID)
		s.budgetCreateAndReply(ctx, b, chatID, session)

	default:
		s.answerCallback(ctx, b, update, "")
	}
}

// budgetHandleText avanza los pasos de texto (monto, fecha, payee) de
// /budget_transaction. Devuelve true si consumió el mensaje (había sesión
// activa); false si el mensaje debe caer en la respuesta por defecto
// (SPEC-097 REQ-012).
func (s *TelegramBotService) budgetHandleText(ctx context.Context, b *bot.Bot, update *tgmodels.Update) bool {
	if update.Message == nil {
		return false
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	s.txMu.Lock()
	session := s.txSession[chatID]
	if session != nil {
		session.expiresAt = time.Now().Add(budgetTxTTL)
	}
	s.txMu.Unlock()
	if session == nil {
		return false
	}

	switch session.step {
	case stepAmount:
		amount, ok := parseAmount(text)
		if !ok {
			s.reply(ctx, b, update, "⚠️ Monto inválido. Escribí un número mayor a cero (ej: 150.50 o 150,50).")
			return true
		}
		session.amount = amount
		session.step = stepDate
		s.reply(ctx, b, update, s.budgetDatePrompt())
	case stepDate:
		date, ok := parseDate(text, time.Now().In(s.budgetLocation(ctx)))
		if !ok {
			s.reply(ctx, b, update, "⚠️ Fecha inválida. Usá el formato *YYYY-MM-DD* (ej: 2026-09-28) o escribí *hoy*.")
			return true
		}
		session.date = date
		session.step = stepAccount
		s.budgetAskAccount(ctx, b, chatID, session)
	case stepPayee:
		if text != "" && text != "/saltar" {
			session.payee = text
		}
		session.step = stepConfirm
		format, err := s.settings.GetCurrencyFormat(ctx)
		if err != nil {
			format = DefaultCurrencyFormat()
		}
		s.budgetSend(ctx, b, chatID, s.budgetSummary(session, format), s.budgetConfirmKeyboard())
	default:
		s.reply(ctx, b, update, "Respondé con los botones de arriba para continuar la transacción.")
	}
	return true
}

// budgetCreateAndReply persiste la transacción vía BudgetTransactionService
// (validación de SPEC-093) y responde el resultado. Siempre elimina la sesión
// del chat, exitosa o no (SPEC-097 REQ-010).
func (s *TelegramBotService) budgetCreateAndReply(ctx context.Context, b *bot.Bot, chatID int64, session *budgetTxSession) {
	tx := budgetBuildTransaction(session)

	created, err := s.transactions.Create(ctx, tx)
	if err != nil {
		slog.Error("telegram_bot: budget_transaction — crear transacción", "error", err)
		s.txMu.Lock()
		delete(s.txSession, chatID)
		s.txMu.Unlock()
		s.budgetSend(ctx, b, chatID, "⚠️ No se pudo guardar la transacción: "+err.Error(), nil)
		return
	}

	format, err := s.settings.GetCurrencyFormat(ctx)
	if err != nil {
		format = DefaultCurrencyFormat()
	}
	msg := fmt.Sprintf("✅ *Transacción guardada* (#%d)\n  %s — %s\n  %s\n  Fecha: %s · Cuenta: %s",
		created.ID,
		session.categoryName,
		formatAmount(session.amount, session.currencySymbol, format),
		budgetKindLabel(session.isInflow),
		session.date,
		session.accountName,
	)
	s.txMu.Lock()
	delete(s.txSession, chatID)
	s.txMu.Unlock()
	s.budgetSend(ctx, b, chatID, msg, nil)
}

// budgetBuildTransaction arma el models.Transaction a partir de la sesión
// (SPEC-097 REQ-010). Función pura para testear el armado del payload.
func budgetBuildTransaction(session *budgetTxSession) *appmodels.Transaction {
	tx := &appmodels.Transaction{
		AccountID:  session.accountID,
		CategoryID: &session.categoryID,
		CurrencyID: session.currencyID,
		Date:       session.date,
		Payee:      session.payee,
		Memo:       "",
		Cleared:    false,
	}
	if session.isInflow {
		tx.Inflow = session.amount
	} else {
		tx.Outflow = session.amount
	}
	return tx
}

// budgetAskAccount lista las cuentas con botones inline para el paso Cuenta
// (SPEC-097 REQ-008). La moneda se deriva de la cuenta elegida.
func (s *TelegramBotService) budgetAskAccount(ctx context.Context, b *bot.Bot, chatID int64, session *budgetTxSession) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		slog.Error("telegram_bot: budget_transaction — listar cuentas", "error", err)
		s.budgetSend(ctx, b, chatID, "⚠️ Error al leer las cuentas. Cancelá y volvé a intentar.", nil)
		return
	}
	s.budgetSend(ctx, b, chatID, fmt.Sprintf("Fecha: *%s*\n\nElegí la *cuenta* (la moneda será la de la cuenta):", session.date), s.budgetAccountKeyboard(accounts))
}

// budgetGroupByID devuelve un grupo con sus categorías activas por ID.
func (s *TelegramBotService) budgetGroupByID(ctx context.Context, id int64) (*BudgetCategoryGroup, error) {
	groups, err := s.budgetGroups.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].ID == id {
			return &groups[i], nil
		}
	}
	return nil, nil
}

// budgetCategoryByID busca una categoría activa por ID en todos los grupos.
func (s *TelegramBotService) budgetCategoryByID(ctx context.Context, id int64) (*appmodels.Category, error) {
	groups, err := s.budgetGroups.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		for j := range groups[i].Categories {
			if groups[i].Categories[j].ID == id {
				return &groups[i].Categories[j], nil
			}
		}
	}
	return nil, nil
}

// budgetAccountByID busca una cuenta activa por ID.
func (s *TelegramBotService) budgetAccountByID(ctx context.Context, id int64) (*appmodels.Account, error) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		if accounts[i].ID == id {
			return &accounts[i], nil
		}
	}
	return nil, nil
}

// currencyByID busca una moneda por ID.
func (s *TelegramBotService) currencyByID(ctx context.Context, id int64) (*appmodels.Currency, error) {
	currencies, err := s.currencies.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range currencies {
		if currencies[i].ID == id {
			return &currencies[i], nil
		}
	}
	return nil, nil
}

// budgetCleanup elimina las sesiones expiradas (SPEC-097 REQ-012).
func (s *TelegramBotService) budgetCleanup() {
	now := time.Now()
	s.txMu.Lock()
	defer s.txMu.Unlock()
	for chatID, session := range s.txSession {
		if now.After(session.expiresAt) {
			delete(s.txSession, chatID)
		}
	}
}

// budgetLocation devuelve la zona horaria configurada para fechas (SPEC-084 ADR-004).
func (s *TelegramBotService) budgetLocation(ctx context.Context) *time.Location {
	loc, err := s.settings.GetTimezoneLocation(ctx)
	if err != nil {
		return time.UTC
	}
	return loc
}

// budgetSend envía un mensaje al chat, opcionalmente con un teclado inline.
func (s *TelegramBotService) budgetSend(ctx context.Context, b *bot.Bot, chatID int64, text string, keyboard *tgmodels.InlineKeyboardMarkup) {
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: tgmodels.ParseModeMarkdownV1,
	}
	if keyboard != nil {
		params.ReplyMarkup = keyboard
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		slog.Warn("telegram_bot: budget_transaction — enviar mensaje", "error", err)
	}
}

// answerCallback responde el callback query para quitar el "reloj" del botón
// (REQ-014) y mostrar feedback opcional.
func (s *TelegramBotService) answerCallback(ctx context.Context, b *bot.Bot, update *tgmodels.Update, text string) {
	if update.CallbackQuery == nil {
		return
	}
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            text,
	}); err != nil {
		slog.Warn("telegram_bot: budget_transaction — answer callback", "error", err)
	}
}

// budgetClearKeyboard quita los botones del mensaje previo al avanzar de paso
// (REQ-014): el chat no acumula teclados viejos.
func (s *TelegramBotService) budgetClearKeyboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	if _, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    chatID,
		MessageID: messageID,
	}); err != nil {
		slog.Debug("telegram_bot: budget_transaction — limpiar teclado", "error", err)
	}
}

// budgetGroupKeyboard arma el teclado de selección de grupos (solo grupos con
// categorías activas).
func (s *TelegramBotService) budgetGroupKeyboard(groups []BudgetCategoryGroup) *tgmodels.InlineKeyboardMarkup {
	rows := make([][]tgmodels.InlineKeyboardButton, 0, len(groups)+1)
	for i := range groups {
		if len(groups[i].Categories) == 0 {
			continue
		}
		rows = append(rows, []tgmodels.InlineKeyboardButton{{
			Text:         s.budgetGroupLabel(&groups[i]),
			CallbackData: fmt.Sprintf("bt:grp:%d", groups[i].ID),
		}})
	}
	rows = append(rows, []tgmodels.InlineKeyboardButton{{Text: "❌ Cancelar", CallbackData: "bt:cancel"}})
	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// budgetGroupLabel formatea el label de un grupo con su ícono si tiene.
func (s *TelegramBotService) budgetGroupLabel(g *BudgetCategoryGroup) string {
	if g.Icon != "" {
		return g.Icon + " " + g.Name
	}
	return g.Name
}

// budgetCategoryKeyboard arma el teclado de categorías de un grupo.
func (s *TelegramBotService) budgetCategoryKeyboard(cats []appmodels.Category) *tgmodels.InlineKeyboardMarkup {
	rows := make([][]tgmodels.InlineKeyboardButton, 0, len(cats)+1)
	for i := range cats {
		rows = append(rows, []tgmodels.InlineKeyboardButton{{
			Text:         s.budgetCategoryLabel(&cats[i]),
			CallbackData: fmt.Sprintf("bt:cat:%d", cats[i].ID),
		}})
	}
	rows = append(rows, []tgmodels.InlineKeyboardButton{{Text: "❌ Cancelar", CallbackData: "bt:cancel"}})
	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// budgetCategoryLabel formatea el label de una categoría con su ícono si tiene.
func (s *TelegramBotService) budgetCategoryLabel(c *appmodels.Category) string {
	if c.Icon != "" {
		return c.Icon + " " + c.Name
	}
	return c.Name
}

// budgetTypeKeyboard arma el teclado Gasto/Ingreso.
func (s *TelegramBotService) budgetTypeKeyboard() *tgmodels.InlineKeyboardMarkup {
	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: [][]tgmodels.InlineKeyboardButton{
		{{Text: "💸 Gasto", CallbackData: "bt:type:outflow"}},
		{{Text: "💰 Ingreso", CallbackData: "bt:type:inflow"}},
		{{Text: "❌ Cancelar", CallbackData: "bt:cancel"}},
	}}
}

// budgetAccountKeyboard arma el teclado de cuentas.
func (s *TelegramBotService) budgetAccountKeyboard(accounts []appmodels.Account) *tgmodels.InlineKeyboardMarkup {
	rows := make([][]tgmodels.InlineKeyboardButton, 0, len(accounts)+1)
	for i := range accounts {
		rows = append(rows, []tgmodels.InlineKeyboardButton{{
			Text:         accounts[i].Name,
			CallbackData: fmt.Sprintf("bt:acct:%d", accounts[i].ID),
		}})
	}
	rows = append(rows, []tgmodels.InlineKeyboardButton{{Text: "❌ Cancelar", CallbackData: "bt:cancel"}})
	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// budgetConfirmKeyboard arma el teclado Confirmar/Cancelar.
func (s *TelegramBotService) budgetConfirmKeyboard() *tgmodels.InlineKeyboardMarkup {
	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: [][]tgmodels.InlineKeyboardButton{
		{{Text: "✅ Confirmar", CallbackData: "bt:ok"}},
		{{Text: "❌ Cancelar", CallbackData: "bt:cancel"}},
	}}
}

// budgetAmountPrompt arma el mensaje del paso Monto.
func (s *TelegramBotService) budgetAmountPrompt(session *budgetTxSession) string {
	return fmt.Sprintf("Categoría: *%s* · %s\n\nEscribí el *monto* (ej: 150.50):", session.categoryName, budgetKindLabel(session.isInflow))
}

// budgetDatePrompt arma el mensaje del paso Fecha.
func (s *TelegramBotService) budgetDatePrompt() string {
	return "Escribí la *fecha* en formato *YYYY-MM-DD* (ej: 2026-09-28) o *hoy* para usar la fecha actual:"
}

// budgetSummary arma el resumen de la transacción para la confirmación.
func (s *TelegramBotService) budgetSummary(session *budgetTxSession, format CurrencyFormat) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📝 *Resumen*\n")
	fmt.Fprintf(&b, "  Categoría: %s\n", session.categoryName)
	fmt.Fprintf(&b, "  Tipo: %s\n", budgetKindLabel(session.isInflow))
	fmt.Fprintf(&b, "  Monto: %s\n", formatAmount(session.amount, session.currencySymbol, format))
	fmt.Fprintf(&b, "  Fecha: %s\n", session.date)
	fmt.Fprintf(&b, "  Cuenta: %s\n", session.accountName)
	if session.payee != "" {
		fmt.Fprintf(&b, "  Concepto: %s\n", session.payee)
	}
	return b.String()
}

// budgetKindLabel devuelve la etiqueta del tipo de transacción.
func budgetKindLabel(inflow bool) string {
	if inflow {
		return "💰 Ingreso"
	}
	return "💸 Gasto"
}

// parseAmount valida un monto positivo. Acepta punto o coma decimal.
func parseAmount(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, ",", ".")
	amount, err := strconv.ParseFloat(s, 64)
	if err != nil || amount <= 0 || amount > 1e12 {
		return 0, false
	}
	return amount, true
}

// parseDate normaliza la fecha a "YYYY-MM-DD". Vacío/"hoy"/"today" usa `now`.
func parseDate(s string, now time.Time) (string, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "hoy" || s == "today" {
		return fmt.Sprintf("%04d-%02d-%02d", now.Year(), now.Month(), now.Day()), true
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%04d-%02d-%02d", t.Year(), t.Month(), t.Day()), true
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
