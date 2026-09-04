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
// BillSummaryScheduler.
type DebtDueScheduler struct {
	debtBillStorage *storage.DebtBillStorage
	emailService    *EmailService
	settingsService *SystemSettingsService
	alertService    *AlertService
	stopCh          chan struct{}
	lastCheckKey    string
}

func NewDebtDueScheduler(
	debtBillStorage *storage.DebtBillStorage,
	emailService *EmailService,
	settingsService *SystemSettingsService,
	alertService *AlertService,
) *DebtDueScheduler {
	return &DebtDueScheduler{
		debtBillStorage: debtBillStorage,
		emailService:    emailService,
		settingsService: settingsService,
		alertService:    alertService,
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

func (s *DebtDueScheduler) checkAndSend() {
	ctx := context.Background()

	if !alertMailEnabled(ctx, s.alertService, models.AlertKeyDebtDue) {
		slog.Debug("debt due scheduler: canal mail deshabilitado")
		return
	}

	hour, err := s.settingsService.GetAlertCheckHour(ctx)
	if err != nil {
		slog.Error("debt due scheduler: error al obtener hora", "error", err)
		return
	}

	now := time.Now()
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

	due, err := s.debtBillStorage.ListDueOnDate(ctx, today)
	if err != nil {
		slog.Error("debt due scheduler: error al listar cuotas del día", "error", err)
		return
	}

	if len(due) > 0 {
		if err := s.sendDueEmail(ctx, due); err != nil {
			slog.Error("debt due scheduler: error al enviar email", "error", err.Error())
		}
	} else {
		slog.Info("debt due scheduler: no hay cuotas que vencen hoy")
	}

	_ = s.settingsService.Set(ctx, s.lastCheckKey, today)
	slog.Info("debt due scheduler: check completado", "due_today", len(due))
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

	subject := fmt.Sprintf("P40LA — %d cuota%s que vencen hoy (%s)", len(due), plural(len(due)), formatAmount(total, "", format))
	content := renderDebtDueContent(due, format)

	html := s.emailService.RenderTemplate(subject, content)
	return s.emailService.Send(ctx, recipients, subject, html)
}

// plural agrega la "s" según la cantidad.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
