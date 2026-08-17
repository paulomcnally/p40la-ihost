package models

// VoiceMonkeyConfig — configuración completa de Voice Monkey. Uso interno
// (VoiceMonkeyService). NUNCA se serializa a JSON para la API. Contiene
// credenciales sensibles (Token, Device).
type VoiceMonkeyConfig struct {
	Enabled    bool
	SendAlerts bool
	Token      string
	Device     string
}

// VoiceMonkeyConfigPublic — versión segura para API responses. Sin credenciales.
// voicemonkey_token y voicemonkey_device nunca se devuelven.
type VoiceMonkeyConfigPublic struct {
	Enabled    bool `json:"voicemonkey_enabled"`
	SendAlerts bool `json:"voicemonkey_send_alerts"`
	Configured bool `json:"voicemonkey_configured"`
}
