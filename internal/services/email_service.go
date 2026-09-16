package services

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// emailTemplateHTML es la plantilla única para todos los emails.
// Misma apariencia siempre; solo varían {{TITLE}}, {{CONTENT}}, {{DATE}}
// y los colores de la paleta ({{COLOR_*}}), que se inyectan al renderizar
// (SPEC-077). Es fluida (width 100% + max-width 600px). Los datos de los
// emails (cards) usan estilos inline y NO dependen de este <style> (ADR-001).
const emailTemplateHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin:0; padding:0; background-color:{{COLOR_BG}}; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif; }
    a { color:{{COLOR_PRIMARY}}; }

    @media screen and (max-width: 480px) {
      .p40la-header { padding:16px 20px !important; }
      .p40la-title { font-size:18px !important; padding:20px 20px 0 20px !important; }
      .p40la-content { padding:12px 20px 20px 20px !important; font-size:14px !important; }
    }
  </style>
</head>
<body>
  <table width="100%" cellpadding="0" cellspacing="0" style="background-color:{{COLOR_BG}};padding:24px 0;">
    <tr>
      <td align="center">
        <table width="100%" cellpadding="0" cellspacing="0" style="max-width:600px;background-color:{{COLOR_CARD}};border-radius:16px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
          <tr>
            <td class="p40la-header" style="background-color:{{COLOR_PRIMARY}};padding:24px 32px;">
              <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:700;">P40LA</h1>
            </td>
          </tr>
          <tr>
            <td class="p40la-title" style="padding:32px 32px 0 32px;">
              <h2 style="margin:0;color:{{COLOR_TEXT}};font-size:20px;font-weight:600;">{{TITLE}}</h2>
            </td>
          </tr>
          <tr>
            <td class="p40la-content" style="padding:16px 32px 32px 32px;color:{{COLOR_TEXT}};font-size:15px;line-height:1.6;">
              {{CONTENT}}
            </td>
          </tr>
          <tr>
            <td style="padding:16px 32px;border-top:1px solid {{COLOR_BORDER}};background-color:{{COLOR_BG}};">
              <p style="margin:0;color:{{COLOR_MUTED}};font-size:12px;">
                P40LA iHost — Alerta automática generada el {{DATE}}
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`

// EmailService envía emails vía SMTP usando net/smtp (stdlib).
// NO guarda ni loguea credenciales (user/password).
type EmailService struct {
	settings *SystemSettingsService
}

func NewEmailService(settings *SystemSettingsService) *EmailService {
	return &EmailService{settings: settings}
}

// Send envía un email HTML a los destinatarios usando la config SMTP.
// El error devuelto NUNCA incluye user ni password.
func (s *EmailService) Send(ctx context.Context, to []string, subject, htmlContent string) error {
	if len(to) == 0 {
		return fmt.Errorf("no hay destinatarios")
	}

	cfg, err := s.settings.GetSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("obtener config SMTP: %w", err)
	}
	if !s.settings.isConfigured(cfg) {
		return fmt.Errorf("config SMTP incompleta (falta host, user o password)")
	}

	msg, err := buildMessage(cfg, to, subject, htmlContent)
	if err != nil {
		return fmt.Errorf("construir mensaje: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)

	// IMPORTANTE: no loguear user/password. Solo host.
	slog.Info("email: enviando", "host", cfg.Host, "port", cfg.Port, "to", strings.Join(to, ","), "subject", subject)

	if err := smtp.SendMail(addr, auth, cfg.FromEmail, to, msg); err != nil {
		slog.Error("email: fallo SMTP", "host", cfg.Host, "error", err.Error())
		return fmt.Errorf("error al enviar email via SMTP: %w", err)
	}

	slog.Info("email: enviado", "to", strings.Join(to, ","), "subject", subject)
	return nil
}

// SendTest envía un email de prueba a los destinatarios configurados.
func (s *EmailService) SendTest(ctx context.Context, to []string) error {
	html := s.RenderTemplate(
		ctx,
		"Email de prueba P40LA",
		"<p>Si estás viendo este mensaje, la configuración SMTP de P40LA iHost funciona correctamente.</p>",
	)
	return s.Send(ctx, to, "P40LA — Email de prueba", html)
}

// RenderTemplate renderiza la plantilla HTML única con título, contenido
// y la paleta de colores configurada (o los defaults si no hay nada
// configurado). Requiere ctx para leer la paleta desde SystemSettingsService.
func (s *EmailService) RenderTemplate(ctx context.Context, title, contentHTML string) string {
	palette := DefaultEmailPalette()
	if s.settings != nil {
		if p, err := s.settings.GetEmailPalette(ctx); err == nil {
			palette = p
		}
	}
	return renderTemplateWithPalette(title, contentHTML, palette)
}

// renderTemplateWithPalette renderiza la plantilla con una paleta dada.
// Interno: usado por RenderTemplate (paleta persistida) y por la preview
// de Settings (paleta candidata, sin persistir — REQ-014).
func renderTemplateWithPalette(title, contentHTML string, palette EmailPalette) string {
	date := time.Now().Format("02/01/2006")
	out := strings.ReplaceAll(emailTemplateHTML, "{{TITLE}}", title)
	out = strings.ReplaceAll(out, "{{CONTENT}}", contentHTML)
	out = strings.ReplaceAll(out, "{{DATE}}", date)
	out = strings.ReplaceAll(out, "{{COLOR_PRIMARY}}", palette.Primary)
	out = strings.ReplaceAll(out, "{{COLOR_BG}}", palette.Background)
	out = strings.ReplaceAll(out, "{{COLOR_CARD}}", palette.Card)
	out = strings.ReplaceAll(out, "{{COLOR_TEXT}}", palette.Text)
	out = strings.ReplaceAll(out, "{{COLOR_MUTED}}", palette.Muted)
	out = strings.ReplaceAll(out, "{{COLOR_BORDER}}", palette.Border)
	return out
}

// RenderPreviewHTML renderiza un email de ejemplo (resumen de facturas con
// datos de muestra) usando una paleta candidata, sin persistir nada. Se usa
// en el preview en vivo de Settings (REQ-014).
func (s *EmailService) RenderPreviewHTML(ctx context.Context, palette EmailPalette) string {
	now := time.Now()
	sample := []models.PendingBillDetail{
		{
			BillID: 1, ServiceName: "Internet residencial", HomeName: "Casa Central",
			Institution: "Claro", Month: int(now.Month()), Year: now.Year(), Amount: 1500,
			CurrencySymbol: "C$", CreatedAt: now.AddDate(0, 0, -3),
		},
		{
			BillID: 2, ServiceName: "Luz", HomeName: "Casa Central",
			Institution: "Enel", Month: int(now.Month()), Year: now.Year(), Amount: 1200.50,
			CurrencySymbol: "C$", CreatedAt: now.AddDate(0, 0, -20),
		},
		{
			BillID: 3, ServiceName: "Seguro auto", HomeName: "Casa Playa",
			Institution: "Seguros XYZ", Month: int(now.Month()), Year: now.Year(), Amount: 300,
			CurrencySymbol: "$", CreatedAt: now,
		},
	}
	content := renderBillSummaryContent(sample, DefaultCurrencyFormat(), palette)
	return renderTemplateWithPalette("P40LA — Vista previa de email", content, palette)
}

// buildMessage construye el mensaje MIME multipart/alternative con
// texto plano + HTML, para máxima compatibilidad con clientes de email.
func buildMessage(cfg *models.SMTPConfig, to []string, subject, htmlContent string) ([]byte, error) {
	fromHeader := cfg.FromName
	if fromHeader == "" {
		fromHeader = cfg.FromEmail
	}
	if fromHeader != cfg.FromEmail {
		fromHeader = fmt.Sprintf("%s <%s>", fromHeader, cfg.FromEmail)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")

	mw := multipart.NewWriter(&buf)
	contentType := fmt.Sprintf("multipart/alternative; boundary=%q", mw.Boundary())
	fmt.Fprintf(&buf, "Content-Type: %s\r\n\r\n", contentType)

	// Parte texto plano
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	textPart, err := mw.CreatePart(textHeader)
	if err != nil {
		return nil, err
	}
	text := stripHTML(htmlContent)
	if _, err := textPart.Write([]byte(text)); err != nil {
		return nil, err
	}

	// Parte HTML
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlPart, err := mw.CreatePart(htmlHeader)
	if err != nil {
		return nil, err
	}
	if _, err := htmlPart.Write([]byte(htmlContent)); err != nil {
		return nil, err
	}

	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// stripHTML convierte HTML a texto plano simple (para la parte text/plain).
func stripHTML(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.Join(strings.Fields(out), " ")
	return strings.TrimSpace(out)
}
