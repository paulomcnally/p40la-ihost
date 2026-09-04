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
// agrupadas por casa (SPEC-031), por los canales habilitados (mail y/o voz).
// Sigue el patrón de AlertScheduler.
type BillSummaryScheduler struct {
	billStorage     *storage.BillStorage
	emailService    *EmailService
	settingsService *SystemSettingsService
	alertService    *AlertService
	voiceMonkey     *VoiceMonkeyService
	stopCh          chan struct{}
	lastCheckKey    string
}

func NewBillSummaryScheduler(
	billStorage *storage.BillStorage,
	emailService *EmailService,
	settingsService *SystemSettingsService,
	alertService *AlertService,
	voiceMonkey *VoiceMonkeyService,
) *BillSummaryScheduler {
	return &BillSummaryScheduler{
		billStorage:     billStorage,
		emailService:    emailService,
		settingsService: settingsService,
		alertService:    alertService,
		voiceMonkey:     voiceMonkey,
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

	now := time.Now()
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
	} else {
		slog.Info("bill summary scheduler: no hay facturas pendientes")
	}

	_ = s.settingsService.Set(ctx, s.lastCheckKey, today)
	slog.Info("bill summary scheduler: check completado", "pending", len(pending))
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
	content := renderBillSummaryContent(pending, format)

	html := s.emailService.RenderTemplate(subject, content)
	return s.emailService.Send(ctx, recipients, subject, html)
}
