package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// BillSummaryScheduler envía diariamente un resumen de las facturas pendientes,
// agrupadas por casa (SPEC-031), por los canales habilitados (mail, voz y/o
// telegram). Sigue el patrón de AlertScheduler.
type BillSummaryScheduler struct {
	billStorage     *storage.BillStorage
	emailService    *EmailService
	settingsService *SystemSettingsService
	alertService    *AlertService
	voiceMonkey     *VoiceMonkeyService
	telegramBot     *TelegramBotService
	stopCh          chan struct{}
	lastCheckKey    string
}

func NewBillSummaryScheduler(
	billStorage *storage.BillStorage,
	emailService *EmailService,
	settingsService *SystemSettingsService,
	alertService *AlertService,
	voiceMonkey *VoiceMonkeyService,
	telegramBot *TelegramBotService,
) *BillSummaryScheduler {
	return &BillSummaryScheduler{
		billStorage:     billStorage,
		emailService:    emailService,
		settingsService: settingsService,
		alertService:    alertService,
		voiceMonkey:     voiceMonkey,
		telegramBot:     telegramBot,
		stopCh:          make(chan struct{}),
		lastCheckKey:    "last_bill_summary_check",
	}
}

func (s *BillSummaryScheduler) Start() {
	go s.run()
}

func (s *BillSummaryScheduler) Stop() {
	close(s.stopCh)
}

func (s *BillSummaryScheduler) run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	slog.Info("bill summary scheduler iniciado")

	for {
		select {
		case <-ticker.C:
			s.checkAndSend()
		case <-s.stopCh:
			slog.Info("bill summary scheduler detenido")
			return
		}
	}
}

// CheckNow fuerza una ejecución del check (usado por tests y validación manual).
func (s *BillSummaryScheduler) CheckNow() {
	s.checkAndSend()
}

func (s *BillSummaryScheduler) checkAndSend() {
	ctx := context.Background()

	if !alertMailEnabled(ctx, s.alertService, models.AlertKeyBillSummary) &&
		!alertVoiceEnabled(ctx, s.alertService, models.AlertKeyBillSummary) {
		slog.Debug("bill summary scheduler: resumen deshabilitado en todos los canales")
		return
	}

	hour, err := s.settingsService.GetAlertCheckHour(ctx)
	if err != nil {
		slog.Error("bill summary scheduler: error al obtener hora", "error", err)
		return
	}

	now, err := currentUserNow(ctx, s.settingsService)
	if err != nil {
		slog.Error("bill summary scheduler: error al obtener zona horaria", "error", err)
		return
	}
	if now.Hour() != hour {
		return
	}

	lastCheck, err := s.settingsService.GetSetting(ctx, s.lastCheckKey)
	if err != nil {
		slog.Error("bill summary scheduler: error al obtener último check", "error", err)
		return
	}

	today := now.Format("2006-01-02")
	if lastCheck != nil && lastCheck.Value == today {
		return
	}

	pending, err := s.billStorage.ListPendingWithDetails(ctx)
	if err != nil {
		slog.Error("bill summary scheduler: error al listar facturas pendientes", "error", err)
		return
	}

	if len(pending) > 0 {
		if alertMailEnabled(ctx, s.alertService, models.AlertKeyBillSummary) {
			if err := s.sendSummaryEmail(ctx, pending); err != nil {
				slog.Error("bill summary scheduler: error al enviar email de resumen", "error", err.Error())
			}
		} else {
			slog.Debug("bill summary scheduler: resumen sin mail habilitado")
		}

		speech, err := s.alertService.Speech(ctx, models.AlertKeyBillSummary)
		if err != nil {
			slog.Error("bill summary scheduler: error al obtener speech", "error", err)
		} else {
			dispatchVoice(ctx, s.alertService, s.voiceMonkey, models.AlertKeyBillSummary, summarySpeech(speech, len(pending)))
		}

		s.dispatchTelegramSummary(ctx, pending, now)
	} else {
		slog.Info("bill summary scheduler: no hay facturas pendientes")
	}

	_ = s.settingsService.Set(ctx, s.lastCheckKey, today)
	slog.Info("bill summary scheduler: check completado", "pending", len(pending))
}

// dispatchTelegramSummary envía el resumen por Telegram con el mismo formato
// de /servicios_pendientes (SPEC-088 REQ-004).
func (s *BillSummaryScheduler) dispatchTelegramSummary(ctx context.Context, pending []models.PendingBillDetail, now time.Time) {
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
	texts := formatServiciosPendientes(pending, format, now, sepLen, showMonths)
	dispatchTelegram(ctx, s.alertService, s.telegramBot, models.AlertKeyBillSummary, texts)
}

// summarySpeech reemplaza el placeholder {n} del speech con la cantidad de
// facturas pendientes (con concordancia singular/plural).
func summarySpeech(base string, n int) string {
	if n == 1 {
		base = strings.ReplaceAll(base, "{n}", "una")
		base = strings.ReplaceAll(base, "facturas pendientes", "factura pendiente")
		return base
	}
	return strings.ReplaceAll(base, "{n}", strconv.Itoa(n))
}

// sendSummaryEmail renderiza y envía el resumen de facturas pendientes.
func (s *BillSummaryScheduler) sendSummaryEmail(ctx context.Context, pending []models.PendingBillDetail) error {
	recipients, err := s.settingsService.GetAlertEmails(ctx)
	if err != nil {
		return fmt.Errorf("obtener destinatarios: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no hay destinatarios configurados")
	}

	subject := fmt.Sprintf("P40LA — Resumen de facturas pendientes (%d pendientes)", len(pending))

	format, err := s.settingsService.GetCurrencyFormat(ctx)
	if err != nil {
		format = DefaultCurrencyFormat()
	}
	palette, err := s.settingsService.GetEmailPalette(ctx)
	if err != nil {
		palette = DefaultEmailPalette()
	}
	content := renderBillSummaryContent(pending, format, palette)

	html := s.emailService.RenderTemplate(ctx, subject, content)
	return s.emailService.Send(ctx, recipients, subject, html)
}
