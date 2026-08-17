package models

import "time"

// Alert representa una funcionalidad de alerta del sistema y sus canales.
// La tabla `alerts` se siembra desde código (seed) y es la fuente de verdad
// sobre qué alerta es cuál y qué canales tiene habilitados (mail / voz / futuro).
type Alert struct {
	ID           int64     `json:"id"`
	Key          string    `json:"key"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	MailEnabled  bool      `json:"mail_enabled"`
	VoiceEnabled bool      `json:"voice_enabled"`
	Speech       string    `json:"speech"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Keys estables de alertas (usadas por los schedulers y la UI).
const (
	AlertKeyInsurance   = "insurance"
	AlertKeyBillCreated = "bill_created"
	AlertKeyBillSummary = "bill_summary"
)

// AlertChannel identifica un canal de entrega de una alerta.
type AlertChannel string

const (
	AlertChannelMail  AlertChannel = "mail"
	AlertChannelVoice AlertChannel = "voice"
)
