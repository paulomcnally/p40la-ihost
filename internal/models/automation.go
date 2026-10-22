package models

import "time"

// AutomationAccount es una cuenta de un proveedor (institución) en el módulo
// de automation unificado. Credentials nunca se serializa en JSON.
type AutomationAccount struct {
	ID                int64      `json:"id"`
	InstitutionID     int64      `json:"institution_id"`
	InstitutionName   string     `json:"institution_name,omitempty"`
	Identifier        string     `json:"identifier"`
	Label             string     `json:"label,omitempty"`
	Credentials       string     `json:"-"`
	PluginName        *string    `json:"plugin_name,omitempty"`
	WebhookURL        *string    `json:"webhook_url,omitempty"`
	ScheduleTime      *string    `json:"schedule_time,omitempty"`
	ScheduleEnabled   bool       `json:"schedule_enabled"`
	LastRunAt         *time.Time `json:"last_run_at,omitempty"`
	ScheduleFrequency string     `json:"schedule_frequency"`
	ScheduleInterval  *int       `json:"schedule_interval,omitempty"`
	ScheduleDays      *string    `json:"schedule_days,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// AutomationAccountOption es la opción que consume el selector de servicios
// (SPEC-111): institución · identificador · plugin.
type AutomationAccountOption struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

// AutomationPlugin es una entrada del catálogo de plugins compilados.
type AutomationPlugin struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Source      string    `json:"source"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AutomationBill es una factura obtenida por un plugin.
type AutomationBill struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"account_id"`
	PluginName    string    `json:"plugin_name"`
	PluginVersion string    `json:"plugin_version"`
	Source        string    `json:"source"`
	Status        string    `json:"status"`
	Error         *string   `json:"error,omitempty"`
	Raw           string    `json:"-"`
	FetchedAt     time.Time `json:"fetched_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// AutomationWebhookLog registra una entrega (o intento) al webhook.
type AutomationWebhookLog struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	WebhookURL string    `json:"webhook_url"`
	Status     string    `json:"status"`
	Payload    string    `json:"-"`
	Response   string    `json:"response,omitempty"`
	RanAt      time.Time `json:"ran_at"`
	HTTPStatus *int      `json:"http_status,omitempty"`
}
