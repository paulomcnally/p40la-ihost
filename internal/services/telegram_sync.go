package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
)

// syncCallbackPrefix identifica los botones inline de sincronización
// (SPEC-103). El callback_data completo es "sync:<service_id>" y queda muy por
// debajo del límite de 64 bytes de Telegram.
const syncCallbackPrefix = "sync:"

// handleSincronizarServicios implementa /sincronizar (SPEC-103 REQ-001): lista
// los servicios con cuenta de automation vinculada y envía un mensaje por
// servicio con un botón inline para sincronizarlo.
func (s *TelegramBotService) handleSincronizarServicios(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if !s.checkAuthorized(ctx, b, update) {
		return
	}

	configured, err := s.settings.IsAutomationConfigured(ctx)
	if err != nil {
		slog.Error("telegram_bot: /sincronizar — leer config automation", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al verificar la configuración de Automation.")
		return
	}
	if !configured {
		s.reply(ctx, b, update, "⚠️ Automation no está configurada.\nConfigurala en P40LA → Configuración → Automation.")
		return
	}

	services, err := s.syncableServices(ctx)
	if err != nil {
		slog.Error("telegram_bot: /sincronizar — error de storage", "error", err)
		s.reply(ctx, b, update, "⚠️ Error al consultar los servicios. Revisá los logs.")
		return
	}
	if len(services) == 0 {
		s.reply(ctx, b, update, "No hay servicios con una cuenta de Automation vinculada.\nEditá un servicio en la app → «ID de cuenta en Automation».")
		return
	}

	loc, err := s.settings.GetTimezoneLocation(ctx)
	if err != nil {
		loc = time.UTC
	}

	chatID := update.Message.Chat.ID
	listed := 0
	for _, svc := range services {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        formatSyncServiceMessage(svc, loc),
			ParseMode:   tgmodels.ParseModeMarkdownV1,
			ReplyMarkup: syncKeyboard(svc.ID),
		})
		if err != nil {
			// REQ-011: un envío fallido no aborta el resto de la lista.
			slog.Warn("telegram_bot: /sincronizar — enviar servicio", "service_id", svc.ID, "error", err)
			continue
		}
		listed++
	}

	summary := fmt.Sprintf("🔌 *%d servicios con plugin de sincronización*\nTocá «🔄 Sincronizar» en el mensaje del servicio que quieras actualizar.", listed)
	if listed == 1 {
		summary = "🔌 *1 servicio con plugin de sincronización*\nTocá «🔄 Sincronizar» en su mensaje para actualizarlo."
	}
	if listed < len(services) {
		summary = fmt.Sprintf("🔌 *%d de %d servicios listados*\nAlgunos mensajes no se pudieron enviar (revisá los logs).", listed, len(services))
	}
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      summary,
		ParseMode: tgmodels.ParseModeMarkdownV1,
	}); err != nil {
		slog.Warn("telegram_bot: /sincronizar — enviar resumen", "error", err)
	}
}

// syncableServices devuelve los servicios no eliminados con una cuenta de
// automation vinculada (SPEC-103 ADR-001). El "plugin de sincronización" se
// representa en p40la-ihost por services.automation_account_id.
func (s *TelegramBotService) syncableServices(ctx context.Context) ([]appmodels.Service, error) {
	all, err := s.serviceStorage.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	out := make([]appmodels.Service, 0, len(all))
	for _, svc := range all {
		if svc.AutomationAccountID != nil {
			out = append(out, svc)
		}
	}
	return out, nil
}

// formatSyncServiceMessage arma el texto del mensaje de un servicio en el
// listado de /sincronizar (SPEC-103 REQ-002, REQ-013).
func formatSyncServiceMessage(svc appmodels.Service, loc *time.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🔌 *%s*\n", escapeMarkdownV1(svc.Name))
	if svc.AutomationAccountID != nil {
		fmt.Fprintf(&b, "  Cuenta automation: %d\n", *svc.AutomationAccountID)
	}
	state := "inactivo"
	if svc.Active {
		state = "activo"
	}
	fmt.Fprintf(&b, "  Estado: %s\n", state)
	fmt.Fprintf(&b, "  Última factura: %s\n", latestBillStatusLabel(svc.LatestBillStatus))
	if svc.LastWebhookRequest != nil {
		fmt.Fprintf(&b, "  Último webhook: %s\n", syncTimeLabel(svc.LastWebhookRequest.In(loc)))
	}
	return strings.TrimRight(b.String(), "\n")
}

// latestBillStatusLabel traduce el estado de la última factura a un texto
// legible para el bot.
func latestBillStatusLabel(status *string) string {
	if status == nil || strings.TrimSpace(*status) == "" {
		return "sin facturas"
	}
	switch strings.ToLower(strings.TrimSpace(*status)) {
	case "paid":
		return "pagada"
	case "pending":
		return "pendiente"
	default:
		return *status
	}
}

// syncTimeLabel formatea fecha/hora con meses abreviados en español.
func syncTimeLabel(t time.Time) string {
	return fmt.Sprintf("%02d %s %d %02d:%02d", t.Day(), esMonthAbbr[int(t.Month())-1], t.Year(), t.Hour(), t.Minute())
}

// syncKeyboard arma el teclado inline de un servicio con el botón de sync.
func syncKeyboard(serviceID int64) *tgmodels.InlineKeyboardMarkup {
	return &tgmodels.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgmodels.InlineKeyboardButton{{
			{Text: "🔄 Sincronizar", CallbackData: fmt.Sprintf("%s%d", syncCallbackPrefix, serviceID)},
		}},
	}
}

// handleSyncCallback procesa los callback_query "sync:<id>" (SPEC-103
// REQ-005..REQ-009): valida, evita duplicados, responde el callback antes de
// sincronizar y edita el mensaje con el resultado, conservando el botón.
func (s *TelegramBotService) handleSyncCallback(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.CallbackQuery == nil {
		return
	}
	chatID, messageID, ok := callbackMessageRef(update)
	if !ok {
		s.answerCallback(ctx, b, update, "")
		return
	}
	if !s.isAuthorized(ctx, chatID) {
		return
	}
	s.mu.Lock()
	s.lastChatID = chatID
	s.mu.Unlock()

	serviceID, ok := parseSyncCallbackData(update.CallbackQuery.Data)
	if !ok {
		s.answerCallback(ctx, b, update, "")
		return
	}

	if !s.beginSync(serviceID) {
		s.answerCallback(ctx, b, update, "Ya hay una sincronización en curso para este servicio.")
		return
	}
	defer s.endSync(serviceID)

	fail := func(answer, detail string) {
		s.answerCallback(ctx, b, update, answer)
		text := fmt.Sprintf("⚠️ *Servicio ID %d*\n  %s", serviceID, escapeMarkdownV1(detail))
		s.editSyncMessage(ctx, b, chatID, messageID, text, syncKeyboard(serviceID), true)
	}

	svc, err := s.serviceStorage.GetByID(ctx, serviceID)
	if err != nil {
		slog.Error("telegram_bot: sync — consultar servicio", "service_id", serviceID, "error", err)
		fail("No se pudo consultar el servicio.", "No se pudo consultar el servicio. Revisá los logs.")
		return
	}
	if svc == nil {
		fail("El servicio ya no existe.", "El servicio ya no existe.")
		return
	}
	if svc.AutomationAccountID == nil {
		fail("El servicio no tiene cuenta de automation.", fmt.Sprintf("El servicio %s no tiene una cuenta de Automation vinculada.", svc.Name))
		return
	}

	configured, err := s.settings.IsAutomationConfigured(ctx)
	if err != nil {
		slog.Error("telegram_bot: sync — leer config automation", "error", err)
		fail("Error al verificar Automation.", "Error al verificar la configuración de Automation.")
		return
	}
	if !configured {
		fail("Automation no está configurada.", "Automation no está configurada. Configurala en P40LA → Configuración → Automation.")
		return
	}

	// Responder antes de sincronizar: el sync puede tardar hasta 30 s y
	// Telegram invalida el spinner del botón (ADR-003, REQ-006).
	s.answerCallback(ctx, b, update, "Sincronizando…")

	delivered, failed, err := s.automation.SyncAccount(ctx, *svc.AutomationAccountID)
	if err != nil {
		slog.Warn("telegram_bot: sync — fallo de automation", "service_id", serviceID, "account_id", *svc.AutomationAccountID, "error", err)
		text := fmt.Sprintf("⚠️ *%s*\n  No se pudo sincronizar: %s", escapeMarkdownV1(svc.Name), escapeMarkdownV1(err.Error()))
		s.editSyncMessage(ctx, b, chatID, messageID, text, syncKeyboard(serviceID), true)
		return
	}

	loc, err := s.settings.GetTimezoneLocation(ctx)
	if err != nil {
		loc = time.UTC
	}
	text := fmt.Sprintf("✅ *%s* sincronizado\n  Entregadas: %d\n  Fallidas: %d\n  Hora: %s",
		escapeMarkdownV1(svc.Name), delivered, failed, time.Now().In(loc).Format("15:04"))
	slog.Info("telegram_bot: sync completado", "service_id", serviceID, "delivered", delivered, "failed", failed)
	s.editSyncMessage(ctx, b, chatID, messageID, text, syncKeyboard(serviceID), true)
}

// parseSyncCallbackData extrae el service_id de un callback "sync:<id>".
func parseSyncCallbackData(data string) (int64, bool) {
	if !strings.HasPrefix(data, syncCallbackPrefix) {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(data, syncCallbackPrefix), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// callbackMessageRef devuelve chat_id y message_id del mensaje que contiene el
// botón, contemplando el caso de mensaje inaccesible (REQ-008).
func callbackMessageRef(update *tgmodels.Update) (chatID int64, messageID int, ok bool) {
	if update.CallbackQuery == nil {
		return 0, 0, false
	}
	if m := update.CallbackQuery.Message.Message; m != nil {
		return m.Chat.ID, m.ID, true
	}
	if m := update.CallbackQuery.Message.InaccessibleMessage; m != nil {
		return m.Chat.ID, m.MessageID, true
	}
	return 0, 0, false
}

// beginSync marca un servicio como "sincronización en curso". Devuelve false
// si ya había una ejecución activa para ese mismo servicio (ADR-004).
func (s *TelegramBotService) beginSync(serviceID int64) bool {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.syncInFlight == nil {
		s.syncInFlight = make(map[int64]struct{})
	}
	if _, exists := s.syncInFlight[serviceID]; exists {
		return false
	}
	s.syncInFlight[serviceID] = struct{}{}
	return true
}

// endSync libera el guard de un servicio.
func (s *TelegramBotService) endSync(serviceID int64) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	delete(s.syncInFlight, serviceID)
}

// editSyncMessage edita el mensaje del botón con el resultado y conserva el
// teclado. Si el edit falla (mensaje inaccesible o idéntico), cae a un mensaje
// nuevo salvo que Telegram indique que el contenido no cambió (REQ-007/REQ-008).
func (s *TelegramBotService) editSyncMessage(ctx context.Context, b *bot.Bot, chatID int64, messageID int, text string, keyboard *tgmodels.InlineKeyboardMarkup, markdown bool) {
	params := &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ReplyMarkup: keyboard,
	}
	if markdown {
		params.ParseMode = tgmodels.ParseModeMarkdownV1
	}
	if _, err := b.EditMessageText(ctx, params); err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return
		}
		slog.Warn("telegram_bot: sync — editar mensaje, se envía uno nuevo", "chat_id", chatID, "error", err)
		send := &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ReplyMarkup: keyboard,
		}
		if markdown {
			send.ParseMode = tgmodels.ParseModeMarkdownV1
		}
		if _, sendErr := b.SendMessage(ctx, send); sendErr != nil {
			slog.Warn("telegram_bot: sync — enviar resultado", "chat_id", chatID, "error", sendErr)
		}
	}
}

// escapeMarkdownV1 escapa los caracteres especiales de MarkdownV1 en textos
// dinámicos (nombres de servicio y mensajes de error) para evitar que Telegram
// rechace el mensaje por entidades mal formadas.
func escapeMarkdownV1(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`_`, `\_`,
		`*`, `\*`,
		"`", "\\`",
		`[`, `\[`,
	)
	return replacer.Replace(s)
}
