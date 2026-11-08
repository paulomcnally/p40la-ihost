package models

import "time"

// Tipos de campo soportados por las plantillas de información de deuda (SPEC-131).
const (
	DebtInformationFieldText     = "text"
	DebtInformationFieldTextarea = "textarea"
	DebtInformationFieldNumber   = "number"
	DebtInformationFieldDate     = "date"
	DebtInformationFieldSelect   = "select"
	DebtInformationFieldCheckbox = "checkbox"
	DebtInformationFieldComputed = "computed"
)

// DebtInformationFieldTypes es la lista canónica de tipos de campo.
var DebtInformationFieldTypes = []string{
	DebtInformationFieldText,
	DebtInformationFieldTextarea,
	DebtInformationFieldNumber,
	DebtInformationFieldDate,
	DebtInformationFieldSelect,
	DebtInformationFieldCheckbox,
	DebtInformationFieldComputed,
}

// DebtInformationField es un campo de una plantilla de información de deuda.
// Los campos computed se resuelven en el frontend con tokens; no se persisten
// en values_json.
type DebtInformationField struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	Type          string   `json:"type"`
	Required      bool     `json:"required,omitempty"`
	Placeholder   string   `json:"placeholder,omitempty"`
	DefaultValue  string   `json:"default_value,omitempty"`
	HelpText      string   `json:"help_text,omitempty"`
	Options       []string `json:"options,omitempty"`
	ValueTemplate string   `json:"value_template,omitempty"`
}

// DebtInformationTemplate es una plantilla reutilizable de información de
// pago. Fields se omite en los listados; FieldCount y UsageCount son
// calculados.
type DebtInformationTemplate struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Fields      []DebtInformationField `json:"fields,omitempty"`
	FieldCount  int                    `json:"field_count"`
	UsageCount  int                    `json:"usage_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// DebtPaymentInformation es la asignación de una plantilla a una deuda, con
// sus valores guardados como JSON. TemplateID y Template son null cuando la
// deuda no tiene forma de pago configurada.
type DebtPaymentInformation struct {
	DebtID     int64                    `json:"debt_id"`
	TemplateID *int64                   `json:"template_id"`
	Template   *DebtInformationTemplate `json:"template"`
	Values     map[string]any           `json:"values"`
	CreatedAt  *time.Time               `json:"created_at,omitempty"`
	UpdatedAt  *time.Time               `json:"updated_at,omitempty"`
}

// DebtInformationToken describe un token disponible en los campos calculados.
type DebtInformationToken struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Example string `json:"example"`
}
