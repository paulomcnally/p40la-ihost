package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// AlertScheduler verifica diariamente el estado de los seguros de los autos
// y envía un email de alerta cuando detecta autos sin seguro o con seguro vencido.
type AlertScheduler struct {
	autoStorage        *storage.AutoStorage
	autoServiceStorage *storage.AutoServiceStorage
	emailService       *EmailService
	settingsService    *SystemSettingsService
	stopCh             chan struct{}
	lastCheckKey       string
}

func NewAlertScheduler(
	autoStorage *storage.AutoStorage,
	autoServiceStorage *storage.AutoServiceStorage,
	emailService *EmailService,
	settingsService *SystemSettingsService,
) *AlertScheduler {
	return &AlertScheduler{
		autoStorage:        autoStorage,
		autoServiceStorage: autoServiceStorage,
		emailService:       emailService,
		settingsService:    settingsService,
		stopCh:             make(chan struct{}),
		lastCheckKey:       "last_alert_check",
	}
}

func (s *AlertScheduler) Start() {
	go s.run()
}

func (s *AlertScheduler) Stop() {
	close(s.stopCh)
}

func (s *AlertScheduler) run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	slog.Info("alert scheduler iniciado")

	for {
		select {
		case <-ticker.C:
			s.checkAndAlert()
		case <-s.stopCh:
			slog.Info("alert scheduler detenido")
			return
		}
	}
}

// CheckNow fuerza una ejecución del check de alertas (usado por tests y API manual).
func (s *AlertScheduler) CheckNow() {
	s.checkAndAlert()
}

func (s *AlertScheduler) checkAndAlert() {
	ctx := context.Background()

	hour, err := s.settingsService.GetAlertCheckHour(ctx)
	if err != nil {
		slog.Error("alert scheduler: error al obtener hora de alertas", "error", err)
		return
	}

	now := time.Now()
	if now.Hour() != hour {
		return
	}

	lastCheck, err := s.settingsService.GetSetting(ctx, s.lastCheckKey)
	if err != nil {
		slog.Error("alert scheduler: error al obtener último check", "error", err)
		return
	}

	today := now.Format("2006-01-02")
	if lastCheck != nil && lastCheck.Value == today {
		return
	}

	alerts, err := s.collectAlerts(ctx)
	if err != nil {
		slog.Error("alert scheduler: error al recolectar alertas", "error", err)
		return
	}

	if len(alerts) > 0 {
		if err := s.sendAlertEmail(ctx, alerts); err != nil {
			slog.Error("alert scheduler: error al enviar email de alerta", "error", err.Error())
			return
		}
	} else {
		slog.Info("alert scheduler: no hay autos en condición de alerta")
	}

	_ = s.settingsService.Set(ctx, s.lastCheckKey, today)
	slog.Info("alert scheduler: check completado", "alerts", len(alerts))
}

// collectAlerts reúne los autos sin seguro y con seguro vencido.
func (s *AlertScheduler) collectAlerts(ctx context.Context) ([]models.AutoAlert, error) {
	var alerts []models.AutoAlert

	withoutInsurance, err := s.autoStorage.ListWithoutInsurance(ctx)
	if err != nil {
		return nil, err
	}
	alerts = append(alerts, withoutInsurance...)

	expired, err := s.autoStorage.ListWithExpiredInsurance(ctx)
	if err != nil {
		return nil, err
	}
	alerts = append(alerts, expired...)

	return alerts, nil
}

// sendAlertEmail renderiza el email de alerta y lo envía a los destinatarios.
func (s *AlertScheduler) sendAlertEmail(ctx context.Context, alerts []models.AutoAlert) error {
	recipients, err := s.settingsService.GetAlertEmails(ctx)
	if err != nil {
		return fmt.Errorf("obtener destinatarios: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no hay destinatarios configurados")
	}

	subject := "P40LA — Alertas de seguros de autos"
	content := renderAlertsContent(alerts)

	html := s.emailService.RenderTemplate(subject, content)
	return s.emailService.Send(ctx, recipients, subject, html)
}

// renderAlertsContent construye el HTML del cuerpo del email con la tabla de autos.
func renderAlertsContent(alerts []models.AutoAlert) string {
	var b strings.Builder
	b.WriteString("<p>Se detectaron los siguientes vehículos que requieren atención:</p>")
	b.WriteString(`<table width="100%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin-top:16px;">`)
	b.WriteString(`<tr style="background-color:#f5f5f7;">`)
	b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Vehículo</th>`)
	b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Placa</th>`)
	b.WriteString(`<th style="text-align:left;padding:12px;border-bottom:2px solid #e5e5ea;font-size:13px;">Motivo</th>`)
	b.WriteString(`</tr>`)

	for _, a := range alerts {
		vehicle := strings.TrimSpace(a.Brand + " " + a.Model)
		if a.Year > 0 {
			vehicle = fmt.Sprintf("%s %d", vehicle, a.Year)
		}
		if vehicle == "" {
			vehicle = fmt.Sprintf("Auto #%d", a.AutoID)
		}

		var reason string
		switch a.AlertType {
		case models.AlertTypeNoInsurance:
			reason = `<span style="color:#ff3b30;">Sin seguro asociado</span>`
		case models.AlertTypeExpired:
			reason = fmt.Sprintf(`<span style="color:#ff9500;">Seguro vencido el %s</span>`, formatDate(a.EndDate))
		default:
			reason = a.AlertType
		}

		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, vehicle))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, a.Placa))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px;border-bottom:1px solid #e5e5ea;">%s</td>`, reason))
		b.WriteString("</tr>")
	}

	b.WriteString("</table>")
	b.WriteString(`<p style="margin-top:24px;color:#8e8e93;font-size:13px;">`)
	b.WriteString(`Ingresá a P40LA para revisar y asociar un nuevo seguro.</p>`)

	return b.String()
}

// formatDate convierte YYYY-MM-DD a DD/MM/YYYY.
func formatDate(date string) string {
	if len(date) < 10 {
		return date
	}
	return fmt.Sprintf("%s/%s/%s", date[8:10], date[5:7], date[0:4])
}
