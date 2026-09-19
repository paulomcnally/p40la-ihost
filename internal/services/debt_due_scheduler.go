package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// DebtDueScheduler envía un único email por día agrupando todas las cuotas de
// deudas que vencen ese día (pending), con el total del día (SPEC-054).
// Corre a la hora de notificaciones del sistema (alert_check_hour), igual que
// BillSummaryScheduler. Si la alerta tiene el canal telegram habilitado,
// también envía el aviso por Telegram (SPEC-088).
type DebtDueScheduler struct {
	debtBillStorage *storage.DebtBillStorage
	emailService    *EmailService
	settingsService *SystemSettingsService
	alertService    *AlertService
	telegramBot     *TelegramBotService
	stopCh          chan struct{}
	lastCheckKey    string
}

func NewDebtDueScheduler(
	debtBillStorage *storage.DebtBillStorage,
	emailService *EmailService,
	settingsService *SystemSettingsService,
	alertService *AlertService,
	telegramBot *TelegramBotService,
) *DebtDueScheduler {
	return &DebtDueScheduler{
		debtBillStorage: debtBillStorage,
		emailService:    emailService,
		settingsService: settingsService,
		alertService:    alertService,
		telegramBot:     telegramBot,
		stopCh:          make(chan struct{}),
		lastCheckKey:    "last_debt_due_check",
	}
}

func (s *DebtDueScheduler) Start() {
	go s.run()
}

func (s *DebtDueScheduler) Stop() {
	close(s.stopCh)
}

func (s *DebtDueScheduler) run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	slog.Info("debt due scheduler iniciado")

	for {
		select {
		case <-ticker.C:
			s.checkAndSend()
		case <-s.stopCh:
			slog.Info("debt due scheduler detenido")
			return
		}
	}
}

// CheckNow fuerza una ejecución del check (tests y validación manual).
func (s *DebtDueScheduler) CheckNow() {
	s.checkAndSend()
}

// SendNow ejecuta el envío manual de la alerta de cuotas que vencen hoy
// (SPEC-090): salta la hora configurada y el dedup diario, respeta los canales
// habilitados y NO escribe last_debt_due_check (el automático queda intacto).
func (s *DebtDueScheduler) SendNow() AlertSendResult {
	return s.sendDueNow()
}

func (s *DebtDueScheduler) checkAndSend() {
	ctx := context.Background()

	if !alertMailEnabled(ctx, s.alertService, models.AlertKeyDebtDue) &&
		!alertTelegramEnabled(ctx, s.alertService, models.AlertKeyDebtDue) {
		slog.Debug("debt due scheduler: alerta deshabilitada en todos los canales")
		return
	}

	hour, err := s.settingsService.GetAlertCheckHour(ctx)
	if err != nil {
		slog.Error("debt due scheduler: error al obtener hora", "error", err)
		return
	}

	now, err := currentUserNow(ctx, s.settingsService)
	if err != nil {
		slog.Error("debt due scheduler: error al obtener zona horaria", "error", err)
		return
	}
	if now.Hour() != hour {
		return
	}

	lastCheck, err := s.settingsService.GetSetting(ctx, s.lastCheckKey)
	if err != nil {
		slog.Error("debt due scheduler: error al obtener último check", "error", err)
		return
	}

	today := now.Format("2006-01-02")
	if lastCheck != nil && lastCheck.Value == today {
		return
	}

	result := s.sendDueNow()
	_ = s.settingsService.Set(ctx, s.lastCheckKey, today)
	slog.Info("debt due scheduler: check completado", "due_today", result.Items)
}

// sendDueNow recolecta las cuotas que vencen hoy y las despacha por los canales
// habilitados. No consulta la hora configurada ni el dedup diario.
func (s *DebtDueScheduler) sendDueNow() AlertSendResult {
	ctx := context.Background()
	res := newSendResult(models.AlertKeyDebtDue, "Cuotas de deudas que vencen hoy")

	now, err := currentUserNow(ctx, s.settingsService)
	if err != nil {
		slog.Error("debt due scheduler: error al obtener zona horaria", "error", err)
		res.Detail = "Error al obtener zona horaria"
		return res
	}
	today := now.Format("2006-01-02")

	due, err := s.debtBillStorage.ListDueOnDate(ctx, today)
	if err != nil {
		slog.Error("debt due scheduler: error al listar cuotas del día", "error", err)
		res.Detail = "Error al listar cuotas del día"
		return res
	}
	res.Items = len(due)

	if len(due) > 0 {
		if alertMailEnabled(ctx, s.alertService, models.AlertKeyDebtDue) {
			if err := s.sendDueEmail(ctx, due); err != nil {
				slog.Error("debt due scheduler: error al enviar email", "error", err.Error())
				res.Detail = "Error al enviar email"
			} else {
				res.SentChannels = append(res.SentChannels, string(models.AlertChannelMail))
			}
		} else {
			slog.Debug("debt due scheduler: alerta sin mail habilitado")
		}

		if s.dispatchTelegramDueToday(ctx) {
			res.SentChannels = append(res.SentChannels, string(models.AlertChannelTelegram))
		}

		if res.Detail == "" {
			res.Detail = fmt.Sprintf("%d cuota%s que vencen hoy", len(due), plural(len(due)))
		}
	} else {
		slog.Info("debt due scheduler: no hay cuotas que vencen hoy")
		res.Detail = "No hay cuotas que vencen hoy"
	}

	return res
}

// dispatchTelegramDueToday envía por Telegram las cuotas que vencen hoy con
// el layout de /deudas_pendientes (SPEC-088 REQ-004). Devuelve true si envió.
func (s *DebtDueScheduler) dispatchTelegramDueToday(ctx context.Context) bool {
	pending, err := s.debtBillStorage.ListPendingWithDetails(ctx)
	if err != nil {
		slog.Error("debt due scheduler: error al listar cuotas pendientes para Telegram", "error", err)
		return false
	}
	format, err := s.settingsService.GetCurrencyFormat(ctx)
	if err != nil {
		format = DefaultCurrencyFormat()
	}
	sepLen, err := s.settingsService.GetTelegramBotSeparatorLength(ctx)
	if err != nil {
		sepLen = DefaultTelegramBotSeparatorLength
	}
	showMonths, err := s.settingsService.GetTelegramBotShowMonths(ctx)
	if err != nil {
		showMonths = DefaultTelegramBotShowMonths
	}
	now, err := currentUserNow(ctx, s.settingsService)
	if err != nil {
		slog.Error("debt due scheduler: error al obtener zona horaria", "error", err)
		return false
	}
	texts := formatDebtsDueToday(pending, format, now, sepLen, showMonths)
	return dispatchTelegram(ctx, s.alertService, s.telegramBot, models.AlertKeyDebtDue, texts)
}

// sendDueEmail envía el email agrupado con las cuotas que vencen hoy.
func (s *DebtDueScheduler) sendDueEmail(ctx context.Context, due []models.DebtBill) error {
	recipients, err := s.settingsService.GetAlertEmails(ctx)
	if err != nil {
		return fmt.Errorf("obtener destinatarios: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no hay destinatarios configurados")
	}

	total := 0.0
	for _, b := range due {
		total += b.Amount
	}

	format, err := s.settingsService.GetCurrencyFormat(ctx)
	if err != nil {
		format = DefaultCurrencyFormat()
	}
	palette, err := s.settingsService.GetEmailPalette(ctx)
	if err != nil {
		palette = DefaultEmailPalette()
	}

	subject := fmt.Sprintf("P40LA — %d cuota%s que vencen hoy (%s)", len(due), plural(len(due)), formatAmount(total, "", format))
	content := renderDebtDueContent(due, format, palette)

	html := s.emailService.RenderTemplate(ctx, subject, content)
	return s.emailService.Send(ctx, recipients, subject, html)
}

// plural agrega la "s" según la cantidad.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
