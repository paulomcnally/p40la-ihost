package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type BillingScheduler struct {
	serviceStorage    *storage.ServiceStorage
	billStorage       *storage.BillStorage
	settingsService   *SystemSettingsService
	emailService      *EmailService
	currencyStorage   *storage.CurrencyStorage
	alertService      *AlertService
	voiceMonkey       *VoiceMonkeyService
	telegramBot       *TelegramBotService
	stopCh            chan struct{}
	lastGenerationKey string
}

func NewBillingScheduler(
	serviceStorage *storage.ServiceStorage,
	billStorage *storage.BillStorage,
	settingsService *SystemSettingsService,
	emailService *EmailService,
	currencyStorage *storage.CurrencyStorage,
	alertService *AlertService,
	voiceMonkey *VoiceMonkeyService,
	telegramBot *TelegramBotService,
) *BillingScheduler {
	return &BillingScheduler{
		serviceStorage:    serviceStorage,
		billStorage:       billStorage,
		settingsService:   settingsService,
		emailService:      emailService,
		currencyStorage:   currencyStorage,
		alertService:      alertService,
		voiceMonkey:       voiceMonkey,
		telegramBot:       telegramBot,
		stopCh:            make(chan struct{}),
		lastGenerationKey: "last_billing_generation",
	}
}

func (s *BillingScheduler) Start() {
	go s.run()
}

func (s *BillingScheduler) Stop() {
	close(s.stopCh)
}

func (s *BillingScheduler) run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	slog.Info("billing scheduler iniciado")

	for {
		select {
		case <-ticker.C:
			s.checkAndGenerate()
		case <-s.stopCh:
			slog.Info("billing scheduler detenido")
			return
		}
	}
}

func (s *BillingScheduler) checkAndGenerate() {
	ctx := context.Background()

	hour, err := s.settingsService.GetBillingGenerationHour(ctx)
	if err != nil {
		slog.Error("billing scheduler: error al obtener hora de generación", "error", err)
		return
	}

	now, err := currentUserNow(ctx, s.settingsService)
	if err != nil {
		slog.Error("billing scheduler: error al obtener zona horaria", "error", err)
		return
	}
	if now.Hour() != hour {
		return
	}

	lastGen, err := s.settingsService.GetSetting(ctx, s.lastGenerationKey)
	if err != nil {
		slog.Error("billing scheduler: error al obtener última generación", "error", err)
		return
	}

	today := now.Format("2006-01-02")
	if lastGen != nil && lastGen.Value == today {
		return
	}

	slog.Info("billing scheduler: generando facturas automáticas", "date", today)

	services, err := s.serviceStorage.List(ctx, nil)
	if err != nil {
		slog.Error("billing scheduler: error al listar servicios", "error", err)
		return
	}

	generated := 0
	for _, svc := range services {
		if !svc.AutoGenerate || !svc.Active {
			continue
		}

		if err := s.generateBillForService(ctx, &svc, now); err != nil {
			slog.Error("billing scheduler: error al generar factura", "service_id", svc.ID, "error", err)
			continue
		}
		generated++
	}

	if err := s.settingsService.SetBillingGenerationHour(ctx, hour); err != nil {
		slog.Error("billing scheduler: error al actualizar última generación", "error", err)
	}
	_ = s.settingsService.Set(ctx, s.lastGenerationKey, today)

	// Voz: se anuncia UNA vez por corrida (no por factura) para evitar ráfagas TTS.
	if generated > 0 {
		speech, err := s.alertService.Speech(ctx, models.AlertKeyBillCreated)
		if err != nil {
			slog.Error("billing scheduler: error al obtener speech", "error", err)
		} else {
			dispatchVoice(ctx, s.alertService, s.voiceMonkey, models.AlertKeyBillCreated, createdSpeech(speech, generated))
		}

		// Telegram: un aviso por corrida (mismo criterio que la voz, SPEC-088).
		dispatchTelegram(ctx, s.alertService, s.telegramBot, models.AlertKeyBillCreated, formatBillCreatedAlert(generated))
	}

	slog.Info("billing scheduler: generación completada", "generated", generated)
}

// SendNowBillCreated envía un aviso de prueba de "nueva factura generada"
// (SPEC-090) SIN generar facturas ni tocar last_billing_generation. Salta la
// hora configurada y el dedup, respeta los canales habilitados de la alerta.
func (s *BillingScheduler) SendNowBillCreated() AlertSendResult {
	ctx := context.Background()
	res := newSendResult(models.AlertKeyBillCreated, "Nueva factura generada")

	if alertMailEnabled(ctx, s.alertService, models.AlertKeyBillCreated) {
		if err := s.sendBillCreatedTestEmail(ctx); err != nil {
			slog.Error("billing scheduler: error al enviar email de prueba de factura", "error", err.Error())
			res.Detail = "Error al enviar email"
		} else {
			res.SentChannels = append(res.SentChannels, string(models.AlertChannelMail))
		}
	} else {
		slog.Debug("billing scheduler: aviso de factura sin mail habilitado")
	}

	speech, err := s.alertService.Speech(ctx, models.AlertKeyBillCreated)
	if err != nil {
		slog.Error("billing scheduler: error al obtener speech", "error", err)
	} else if dispatchVoice(ctx, s.alertService, s.voiceMonkey, models.AlertKeyBillCreated, speech) {
		res.SentChannels = append(res.SentChannels, string(models.AlertChannelVoice))
	}

	if dispatchTelegram(ctx, s.alertService, s.telegramBot, models.AlertKeyBillCreated, formatBillCreatedAlert(1)) {
		res.SentChannels = append(res.SentChannels, string(models.AlertChannelTelegram))
	}

	res.Items = 1
	if res.Detail == "" {
		res.Detail = "Aviso de prueba enviado (sin generar facturas)"
	}
	return res
}

// sendBillCreatedTestEmail envía el email de prueba del aviso de nueva factura
// a los destinatarios configurados, sin asociarlo a un servicio/factura real.
func (s *BillingScheduler) sendBillCreatedTestEmail(ctx context.Context) error {
	if s.emailService == nil {
		return fmt.Errorf("email service no disponible")
	}

	recipients, err := s.settingsService.GetAlertEmails(ctx)
	if err != nil {
		return fmt.Errorf("obtener destinatarios: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no hay destinatarios configurados")
	}

	configured, err := s.settingsService.IsSMTPConfigured(ctx)
	if err != nil {
		return err
	}
	if !configured {
		return fmt.Errorf("SMTP no configurado")
	}

	title := "Nueva factura generada — Aviso de prueba"
	content := `<p>Este es un <strong>aviso de prueba</strong> de la alerta "Nueva factura generada".</p>
<p>Cuando el sistema genere facturas automáticamente, recibirás un email como este con los detalles del período, el monto y el servicio.</p>
<p style="margin-top:24px;color:#8e8e93;font-size:13px;">Enviado desde el botón "Enviar alertas ahora" de Configuración → Alertas.</p>`

	html := s.emailService.RenderTemplate(ctx, title, content)
	return s.emailService.Send(ctx, recipients, title, html)
}

// createdSpeech adapta el speech de "nueva factura" según la cantidad generada.
func createdSpeech(base string, n int) string {
	if n == 1 {
		return base
	}
	return fmt.Sprintf("Paulo, se generaron %d facturas automáticas.", n)
}

func (s *BillingScheduler) generateBillForService(ctx context.Context, svc *models.Service, now time.Time) error {
	year := now.Year()
	month := now.Month()

	if svc.Frequency == "yearly" {
		if month != time.January {
			return nil
		}
	} else {
		if svc.BillingDay == nil {
			return nil
		}
		targetDay := *svc.BillingDay
		lastDay := daysInMonth(year, month)
		if targetDay > lastDay {
			targetDay = lastDay
		}
		if now.Day() != targetDay {
			return nil
		}
	}

	billMonth := int(month)
	if svc.Frequency == "yearly" {
		billMonth = 0
	}

	existing, err := s.billStorage.FindByServicePeriod(ctx, svc.ID, year, billMonth)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	bill := &models.Bill{
		ServiceID: svc.ID,
		Year:      year,
		Month:     billMonth,
		Amount:    svc.SuggestedAmount,
		Status:    "pending",
	}
	created, err := s.billStorage.Create(ctx, bill)
	if err != nil {
		return err
	}

	s.sendBillCreatedEmail(ctx, created, svc)
	return nil
}

// sendBillCreatedEmail envía un email informativo por cada factura generada
// automáticamente (SPEC-030), solo si la alerta "nueva factura" tiene el canal
// mail habilitado (SPEC-032). Si no, loguea (debug) y no envía.
func (s *BillingScheduler) sendBillCreatedEmail(ctx context.Context, bill *models.Bill, svc *models.Service) {
	if s.emailService == nil {
		return
	}

	if !alertMailEnabled(ctx, s.alertService, models.AlertKeyBillCreated) {
		slog.Debug("billing scheduler: email de factura deshabilitado", "service_id", svc.ID)
		return
	}

	recipients, err := s.settingsService.GetAlertEmails(ctx)
	if err != nil {
		slog.Warn("billing scheduler: no se pudo obtener destinatarios", "error", err)
		return
	}
	if len(recipients) == 0 {
		slog.Warn("billing scheduler: sin destinatarios, no se envía email de factura", "service_id", svc.ID)
		return
	}

	configured, err := s.settingsService.IsSMTPConfigured(ctx)
	if err != nil || !configured {
		slog.Warn("billing scheduler: SMTP no configurado, no se envía email de factura", "service_id", svc.ID)
		return
	}

	symbol := ""
	if s.currencyStorage != nil {
		if currency, err := s.currencyStorage.GetByID(ctx, svc.CurrencyID); err == nil && currency != nil {
			symbol = currency.Symbol
		}
	}

	format, err := s.settingsService.GetCurrencyFormat(ctx)
	if err != nil {
		slog.Warn("billing scheduler: error al leer formato de moneda, usando default", "error", err)
		format = DefaultCurrencyFormat()
	}

	title, content := buildBillCreatedEmail(svc, bill, symbol, format)
	html := s.emailService.RenderTemplate(ctx, title, content)

	if err := s.emailService.Send(ctx, recipients, title, html); err != nil {
		slog.Error("billing scheduler: error al enviar email de factura", "bill_id", bill.ID, "service_id", svc.ID, "error", err.Error())
	}
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
