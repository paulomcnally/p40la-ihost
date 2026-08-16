package models

// SMTPConfig — configuración completa SMTP. Uso interno (EmailService).
// NUNCA se serializa a JSON para la API. Contiene credenciales sensibles.
type SMTPConfig struct {
	Host      string
	Port      int
	User      string
	Password  string
	FromEmail string
	FromName  string
}

// SMTPConfigPublic — versión segura para API responses. Sin credenciales.
// smtp_user y smtp_password nunca se devuelven.
type SMTPConfigPublic struct {
	Host       string `json:"smtp_host"`
	Port       int    `json:"smtp_port"`
	User       string `json:"smtp_user"`
	FromEmail  string `json:"smtp_from_email"`
	FromName   string `json:"smtp_from_name"`
	Configured bool   `json:"smtp_configured"`
}
