package models

// TelegramBotConfig — configuración completa del bot de Telegram. Uso interno
// (TelegramBotService). NUNCA se serializa a JSON para la API. Contiene
// credenciales sensibles (Token).
type TelegramBotConfig struct {
	Enabled         bool
	Token           string
	ChatIDs         []string
	SeparatorLength int // guiones del separador del resumen (SPEC-084 REQ-014)
	ShowMonths      int // rango de meses futuros de /servicios_pendientes (SPEC-084 REQ-016)
}

// TelegramBotConfigPublic — versión segura para API responses. Sin
// credenciales. telegram_bot_token nunca se devuelve. ChatIDsCount permite a
// la UI gatear el canal de alertas por Telegram (SPEC-088); ChatIDs expone la
// lista real (no secreta) para precargar el formulario (SPEC-089).
type TelegramBotConfigPublic struct {
	Enabled         bool     `json:"telegram_bot_enabled"`
	Configured      bool     `json:"telegram_bot_configured"`
	ChatIDs         []string `json:"telegram_bot_chat_ids,omitempty"`
	ChatIDsCount    int      `json:"telegram_bot_chat_ids_count"`
	SeparatorLength int      `json:"telegram_bot_separator_length"`
	ShowMonths      int      `json:"telegram_bot_show_months"`
}
