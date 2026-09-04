package services

import (
	"context"
	"log/slog"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// PensionNotificationService envía las notificaciones por email del módulo
// Pensión Alimenticia de forma best-effort: nunca rompe el flujo HTTP.
// Usa el EmailService existente, los destinatarios de la tabla notifications
// (solo activos) y los toggles de alerta del catálogo (SPEC-051).
type PensionNotificationService struct {
	notificationStorage  *storage.NotificationStorage
	emailService         *EmailService
	alertService         *AlertService
	settingsService      *SystemSettingsService
	recordStorage        *storage.SupportRecordStorage
	salaryPaymentStorage *storage.SalaryPaymentStorage
}

// NewPensionNotificationService crea un nuevo PensionNotificationService.
func NewPensionNotificationService(
	notificationStorage *storage.NotificationStorage,
	emailService *EmailService,
	alertService *AlertService,
	settingsService *SystemSettingsService,
	recordStorage *storage.SupportRecordStorage,
	salaryPaymentStorage *storage.SalaryPaymentStorage,
) *PensionNotificationService {
	return &PensionNotificationService{
		notificationStorage:  notificationStorage,
		emailService:         emailService,
		alertService:         alertService,
		settingsService:      settingsService,
		recordStorage:        recordStorage,
		salaryPaymentStorage: salaryPaymentStorage,
	}
}

// SendRecordsCreated notifica que se generaron registros/salarios del mes.
func (s *PensionNotificationService) SendRecordsCreated(ctx context.Context, salaryPayments []models.SalaryPayment, records []models.SupportRecord, year, month int) {
	if len(salaryPayments) == 0 && len(records) == 0 {
		return
	}
	title, content := buildPensionRecordsCreatedEmail(salaryPayments, records, year, month, s.currencyFormat(ctx))
	s.send(ctx, models.AlertKeyPensionRecordsCreated, title, content)
}

// SendRecordPaid notifica el pago de un registro de manutención.
func (s *PensionNotificationService) SendRecordPaid(ctx context.Context, record *models.SupportRecord) {
	if record == nil {
		return
	}
	title, content := buildPensionRecordPaidEmail(record, s.currencyFormat(ctx))
	s.send(ctx, models.AlertKeyPensionRecordPaid, title, content)
}

// SendSalaryReceived notifica la recepción de un pago de salario.
func (s *PensionNotificationService) SendSalaryReceived(ctx context.Context, payment *models.SalaryPayment) {
	if payment == nil {
		return
	}
	title, content := buildPensionSalaryReceivedEmail(payment, s.currencyFormat(ctx))
	s.send(ctx, models.AlertKeyPensionSalaryReceived, title, content)
}

// SendRecordRejected notifica el rechazo de un registro de manutención.
func (s *PensionNotificationService) SendRecordRejected(ctx context.Context, record *models.SupportRecord, reason string) {
	if record == nil {
		return
	}
	title, content := buildPensionRecordRejectedEmail(record, reason, s.currencyFormat(ctx))
	s.send(ctx, models.AlertKeyPensionRecordRejected, title, content)
}

// SendMonthClosing notifica el cierre de un mes con su resumen.
func (s *PensionNotificationService) SendMonthClosing(ctx context.Context, year, month int) {
	records, err := s.recordStorage.ListByFilters(ctx, year, month, 0)
	if err != nil {
		slog.Warn("pension notifications: error al listar registros para cierre", "error", err.Error())
		return
	}
	salaryPayments, err := s.salaryPaymentStorage.ListByFilters(ctx, year, month, 0)
	if err != nil {
		slog.Warn("pension notifications: error al listar salarios para cierre", "error", err.Error())
		return
	}
	title, content := buildPensionMonthClosingEmail(records, salaryPayments, year, month, s.currencyFormat(ctx))
	s.send(ctx, models.AlertKeyPensionMonthClosing, title, content)
}

// currencyFormat devuelve el formato de moneda configurado (default si error).
func (s *PensionNotificationService) currencyFormat(ctx context.Context) CurrencyFormat {
	format, err := s.settingsService.GetCurrencyFormat(ctx)
	if err != nil {
		slog.Warn("pension notifications: error al leer formato de moneda, usando default", "error", err)
		return DefaultCurrencyFormat()
	}
	return format
}

// send envía un email si la alerta tiene el canal mail habilitado, hay
// destinatarios activos y el SMTP está configurado. Best-effort.
func (s *PensionNotificationService) send(ctx context.Context, alertKey, subject, content string) {
	enabled, err := s.alertService.IsEnabled(ctx, alertKey, models.AlertChannelMail)
	if err != nil {
		slog.Error("pension notifications: error al leer alerta", "key", alertKey, "error", err.Error())
		return
	}
	if !enabled {
		slog.Debug("pension notifications: alerta con mail deshabilitado", "key", alertKey)
		return
	}

	recipients, err := s.activeRecipients(ctx)
	if err != nil {
		slog.Error("pension notifications: error al obtener destinatarios", "error", err.Error())
		return
	}
	if len(recipients) == 0 {
		slog.Warn("pension notifications: sin destinatarios activos, no se envía", "key", alertKey)
		return
	}

	configured, err := s.settingsService.IsSMTPConfigured(ctx)
	if err != nil || !configured {
		slog.Warn("pension notifications: SMTP no configurado, no se envía", "key", alertKey)
		return
	}

	html := s.emailService.RenderTemplate(subject, content)
	if err := s.emailService.Send(ctx, recipients, subject, html); err != nil {
		slog.Error("pension notifications: error al enviar email", "key", alertKey, "error", err.Error())
	}
}

// activeRecipients devuelve los emails de las notificaciones activas.
func (s *PensionNotificationService) activeRecipients(ctx context.Context) ([]string, error) {
	notifications, err := s.notificationStorage.List(ctx)
	if err != nil {
		return nil, err
	}
	var emails []string
	for _, n := range notifications {
		if n.Active {
			emails = append(emails, n.Email)
		}
	}
	return emails, nil
}
