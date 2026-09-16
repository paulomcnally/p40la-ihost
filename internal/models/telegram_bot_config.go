package models

// TelegramBotConfig — configuración completa del bot de Telegram. Uso interno
// (TelegramBotService). NUNCA se serializa a JSON para la API. Contiene
// credenciales sensibles (Token).
type TelegramBotConfig struct {
	Enabled bool
	Token   string
	ChatIDs []string
}

// TelegramBotConfigPublic — versión segura para API responses. Sin
// credenciales. telegram_bot_token nunca se devuelve.
type TelegramBotConfigPublic struct {
	Enabled    bool `json:"telegram_bot_enabled"`
	Configured bool `json:"telegram_bot_configured"`
}
