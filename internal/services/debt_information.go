package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const (
	maxDebtInformationFields      = 30
	maxDebtInformationName        = 100
	maxDebtInformationDescription = 500
	maxDebtInformationFieldLabel  = 100
	maxDebtInformationFieldHelp   = 300
	maxDebtInformationPlaceholder = 200
	maxDebtInformationDefault     = 500
	maxDebtInformationTemplate    = 500
	maxDebtInformationText        = 500
	maxDebtInformationTextarea    = 5000
	maxDebtInformationOptions     = 50
	maxDebtInformationOption      = 100
	maxDebtInformationFieldsJSON  = 32 * 1024
	maxDebtInformationValuesJSON  = 16 * 1024
)

var (
	// ErrDebtInformationDebtNotFound indica que la deuda no existe.
	ErrDebtInformationDebtNotFound = errors.New("deuda no encontrada")
	// Alias de errores de storage para que los handlers dependan de services.
	ErrDebtInformationTemplateNotFound   = storage.ErrDebtInformationTemplateNotFound
	ErrDebtInformationTemplateNameExists = storage.ErrDebtInformationTemplateNameExists
	ErrDebtInformationTemplateInUse      = storage.ErrDebtInformationTemplateInUse
)

var debtInformationKeyRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
var debtInformationTokenRegex = regexp.MustCompile(`\{\{\s*([a-z0-9_.]+)\s*\}\}`)
var debtInformationVariableRegex = regexp.MustCompile(`\{\s*([a-z][a-z0-9_]*)\s*\}`)

// DebtInformationValidationError es un error de validación con un código
// apto para la respuesta HTTP (validation_error / invalid_token).
type DebtInformationValidationError struct {
	Code    string
	Message string
}

func (e *DebtInformationValidationError) Error() string { return e.Message }

func newValidationError(message string) error {
	return &DebtInformationValidationError{Code: "validation_error", Message: message}
}

func newInvalidTokenError(message string) error {
	return &DebtInformationValidationError{Code: "invalid_token", Message: message}
}

func newInvalidVariableError(message string) error {
	return &DebtInformationValidationError{Code: "invalid_variable", Message: message}
}

// Tokens disponibles en los campos calculados. El frontend resuelve los
// valores; el backend solo valida que el token exista.
var debtInformationTokens = []models.DebtInformationToken{
	{Key: "deuda.descripcion", Label: "Descripción de la deuda", Example: "Lote Demo 1"},
	{Key: "deuda.institucion", Label: "Institución o acreedor", Example: "Banco Demo"},
	{Key: "deuda.identificador", Label: "Identificador de la deuda", Example: "demo-account-001"},
	{Key: "deuda.moneda", Label: "Código de la moneda", Example: "USD"},
	{Key: "deuda.total", Label: "Total de la deuda sin formato", Example: "100.00"},
	{Key: "deuda.total_formateado", Label: "Total de la deuda formateado", Example: "$100.00"},
	{Key: "cuota.numero", Label: "Número de cuota", Example: "12"},
	{Key: "cuota.monto", Label: "Monto de la cuota sin formato", Example: "100.00"},
	{Key: "cuota.monto_formateado", Label: "Monto de la cuota formateado", Example: "$100.00"},
	{Key: "cuota.vencimiento", Label: "Vencimiento de la cuota (ISO)", Example: "2026-12-15"},
	{Key: "cuota.vencimiento_formateado", Label: "Vencimiento de la cuota localizado", Example: "15/12/2026"},
	{Key: "cuota.estado", Label: "Estado de la cuota", Example: "Pendiente"},
	{Key: "pago.fecha", Label: "Fecha de pago seleccionada (ISO)", Example: "2026-12-10"},
	{Key: "pago.fecha_formateada", Label: "Fecha de pago seleccionada localizada", Example: "10/12/2026"},
	{Key: "sistema.moneda", Label: "Símbolo de la moneda", Example: "$"},
}

var debtInformationTokenSet = func() map[string]bool {
	set := make(map[string]bool, len(debtInformationTokens))
	for _, token := range debtInformationTokens {
		set[token.Key] = true
	}
	return set
}()

// Variables disponibles en los valores por defecto de campos text/textarea
// (SPEC-132). Se persisten en inglés y se sugieren en español.
var debtInformationValueVariables = []models.DebtInformationVariable{
	{Key: "installment_number", Label: "Número de cuota", Example: "5", Description: "Número de la cuota que se está pagando"},
	{Key: "installment_amount", Label: "Monto de la cuota (sin formato)", Example: "100.00", Description: "Monto de la cuota sin símbolo ni separadores"},
	{Key: "installment_amount_formatted", Label: "Monto de la cuota con moneda", Example: "$100.00", Description: "Monto de la cuota formateado con el símbolo de la moneda"},
	{Key: "installment_due_date", Label: "Vencimiento de la cuota (ISO)", Example: "2026-12-15", Description: "Fecha de vencimiento en formato YYYY-MM-DD"},
	{Key: "installment_due_date_formatted", Label: "Vencimiento de la cuota (localizado)", Example: "15/12/2026", Description: "Fecha de vencimiento en formato local"},
	{Key: "installment_status", Label: "Estado de la cuota", Example: "Pendiente", Description: "Pendiente o Pagada, según el idioma"},
	{Key: "installment_progress", Label: "Progreso de cuotas", Example: "5/12", Description: "Cuota actual sobre el total del plan"},
	{Key: "installments_total", Label: "Total de cuotas del plan", Example: "12", Description: "Cantidad total de cuotas de la deuda"},
	{Key: "debt_description", Label: "Descripción de la deuda", Example: "Lote Demo 13", Description: "Descripción de la deuda"},
	{Key: "debt_identifier", Label: "Identificador de la deuda", Example: "demo-account-001", Description: "Identificador o número de contrato"},
	{Key: "institution_name", Label: "Institución o acreedor", Example: "Proveedor Demo", Description: "Nombre de la institución de la deuda"},
	{Key: "currency_code", Label: "Código de moneda", Example: "USD", Description: "Código ISO de la moneda de la deuda"},
	{Key: "currency_symbol", Label: "Símbolo de moneda", Example: "$", Description: "Símbolo de la moneda de la deuda"},
	{Key: "debt_total", Label: "Total de la deuda (sin formato)", Example: "1200.00", Description: "Total de la deuda sin formato"},
	{Key: "debt_total_formatted", Label: "Total de la deuda con moneda", Example: "$1,200.00", Description: "Total de la deuda formateado"},
	{Key: "payment_date", Label: "Fecha de pago (ISO)", Example: "2026-12-10", Description: "Fecha de pago seleccionada, si ya se pagó"},
	{Key: "payment_date_formatted", Label: "Fecha de pago (localizada)", Example: "10/12/2026", Description: "Fecha de pago en formato local"},
}

var debtInformationValueVariableSet = func() map[string]bool {
	set := make(map[string]bool, len(debtInformationValueVariables))
	for _, variable := range debtInformationValueVariables {
		set[variable.Key] = true
	}
	return set
}()

// DebtInformationService contiene la lógica de plantillas de información y
// forma de pago por deuda (SPEC-131).
type DebtInformationService struct {
	info  *storage.DebtInformationStorage
	debts *storage.DebtStorage
}

// NewDebtInformationService crea el servicio.
func NewDebtInformationService(info *storage.DebtInformationStorage, debts *storage.DebtStorage) *DebtInformationService {
	return &DebtInformationService{info: info, debts: debts}
}

// Tokens devuelve el catálogo de tokens disponibles.
func (s *DebtInformationService) Tokens() []models.DebtInformationToken {
	result := make([]models.DebtInformationToken, len(debtInformationTokens))
	copy(result, debtInformationTokens)
	return result
}

// Variables devuelve el catálogo de variables disponibles en los valores por
// defecto de campos text/textarea (SPEC-132).
func (s *DebtInformationService) Variables() []models.DebtInformationVariable {
	result := make([]models.DebtInformationVariable, len(debtInformationValueVariables))
	copy(result, debtInformationValueVariables)
	return result
}

// ListTemplates devuelve las plantillas sin los campos completos.
func (s *DebtInformationService) ListTemplates(ctx context.Context) ([]models.DebtInformationTemplate, error) {
	return s.info.ListTemplates(ctx)
}

// GetTemplate devuelve una plantilla con sus campos o nil si no existe.
func (s *DebtInformationService) GetTemplate(ctx context.Context, id int64) (*models.DebtInformationTemplate, error) {
	return s.info.GetTemplateByID(ctx, id)
}

// CreateTemplate normaliza, valida y crea una plantilla.
func (s *DebtInformationService) CreateTemplate(ctx context.Context, input *models.DebtInformationTemplate) (*models.DebtInformationTemplate, error) {
	if err := s.prepareTemplate(input); err != nil {
		return nil, err
	}
	return s.info.CreateTemplate(ctx, input)
}

// UpdateTemplate normaliza, valida y actualiza una plantilla.
func (s *DebtInformationService) UpdateTemplate(ctx context.Context, input *models.DebtInformationTemplate) (*models.DebtInformationTemplate, error) {
	if input.ID <= 0 {
		return nil, newValidationError("El ID de la plantilla es obligatorio.")
	}
	if err := s.prepareTemplate(input); err != nil {
		return nil, err
	}
	return s.info.UpdateTemplate(ctx, input)
}

// DeleteTemplate elimina una plantilla si no está asignada.
func (s *DebtInformationService) DeleteTemplate(ctx context.Context, id int64) error {
	return s.info.DeleteTemplate(ctx, id)
}

// GetForDebt devuelve la forma de pago de una deuda. Si no está configurada,
// devuelve una respuesta vacía con template_id/template en null y values {}.
func (s *DebtInformationService) GetForDebt(ctx context.Context, debtID int64) (*models.DebtPaymentInformation, error) {
	if err := s.ensureDebt(ctx, debtID); err != nil {
		return nil, err
	}
	info, err := s.info.GetPaymentInformationByDebtID(ctx, debtID)
	if err != nil {
		return nil, err
	}
	if info != nil {
		return info, nil
	}
	return &models.DebtPaymentInformation{
		DebtID:     debtID,
		TemplateID: nil,
		Template:   nil,
		Values:     map[string]any{},
	}, nil
}

// SaveForDebt valida los valores contra la plantilla y guarda la asignación.
func (s *DebtInformationService) SaveForDebt(
	ctx context.Context,
	debtID int64,
	templateID int64,
	values map[string]any,
) (*models.DebtPaymentInformation, error) {
	if err := s.ensureDebt(ctx, debtID); err != nil {
		return nil, err
	}
	if templateID <= 0 {
		return nil, newValidationError("La plantilla de información es obligatoria.")
	}
	template, err := s.info.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, ErrDebtInformationTemplateNotFound
	}
	cleanValues, err := validateDebtInformationValues(template.Fields, values)
	if err != nil {
		return nil, err
	}
	return s.info.UpsertPaymentInformation(ctx, debtID, templateID, cleanValues)
}

// ClearForDebt quita la forma de pago de una deuda.
func (s *DebtInformationService) ClearForDebt(ctx context.Context, debtID int64) error {
	if err := s.ensureDebt(ctx, debtID); err != nil {
		return err
	}
	return s.info.DeletePaymentInformation(ctx, debtID)
}

func (s *DebtInformationService) ensureDebt(ctx context.Context, debtID int64) error {
	if debtID <= 0 {
		return ErrDebtInformationDebtNotFound
	}
	debt, err := s.debts.GetByID(ctx, debtID)
	if err != nil {
		return err
	}
	if debt == nil {
		return ErrDebtInformationDebtNotFound
	}
	return nil
}

func (s *DebtInformationService) prepareTemplate(input *models.DebtInformationTemplate) error {
	if input == nil {
		return newValidationError("La plantilla de información es obligatoria.")
	}
	normalizeDebtInformationTemplate(input)
	if err := validateDebtInformationTemplate(input); err != nil {
		return err
	}
	return nil
}

func normalizeDebtInformationTemplate(input *models.DebtInformationTemplate) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Fields == nil {
		input.Fields = []models.DebtInformationField{}
	}
	for i := range input.Fields {
		field := &input.Fields[i]
		field.Key = strings.TrimSpace(field.Key)
		field.Label = strings.TrimSpace(field.Label)
		field.Type = strings.TrimSpace(strings.ToLower(field.Type))
		field.Placeholder = strings.TrimSpace(field.Placeholder)
		field.DefaultValue = strings.TrimSpace(field.DefaultValue)
		field.HelpText = strings.TrimSpace(field.HelpText)
		field.ValueTemplate = strings.TrimSpace(field.ValueTemplate)

		if field.Type == models.DebtInformationFieldComputed {
			field.Required = false
			field.Placeholder = ""
			field.DefaultValue = ""
			field.Options = nil
			continue
		}

		field.ValueTemplate = ""
		if field.Type != models.DebtInformationFieldSelect {
			field.Options = nil
			continue
		}

		options := make([]string, 0, len(field.Options))
		for _, option := range field.Options {
			option = strings.TrimSpace(option)
			if option != "" {
				options = append(options, option)
			}
		}
		field.Options = options
	}
}

func validateDebtInformationTemplate(input *models.DebtInformationTemplate) error {
	if input.Name == "" {
		return newValidationError("El nombre de la plantilla es obligatorio.")
	}
	if utf8.RuneCountInString(input.Name) > maxDebtInformationName {
		return newValidationError(fmt.Sprintf("El nombre de la plantilla no puede superar los %d caracteres.", maxDebtInformationName))
	}
	if utf8.RuneCountInString(input.Description) > maxDebtInformationDescription {
		return newValidationError(fmt.Sprintf("La descripción no puede superar los %d caracteres.", maxDebtInformationDescription))
	}
	if len(input.Fields) == 0 {
		return newValidationError("La plantilla debe tener al menos un campo.")
	}
	if len(input.Fields) > maxDebtInformationFields {
		return newValidationError(fmt.Sprintf("La plantilla no puede tener más de %d campos.", maxDebtInformationFields))
	}

	seenKeys := make(map[string]bool, len(input.Fields))
	for i := range input.Fields {
		field := &input.Fields[i]
		position := i + 1
		if !debtInformationKeyRegex.MatchString(field.Key) {
			return newValidationError(fmt.Sprintf("El campo %d tiene una clave inválida. Usá minúsculas, números y guion bajo, empezando por una letra.", position))
		}
		if seenKeys[field.Key] {
			return newValidationError(fmt.Sprintf("La clave '%s' está repetida en la plantilla.", field.Key))
		}
		seenKeys[field.Key] = true

		if field.Label == "" {
			return newValidationError(fmt.Sprintf("El campo %d necesita una etiqueta.", position))
		}
		if utf8.RuneCountInString(field.Label) > maxDebtInformationFieldLabel {
			return newValidationError(fmt.Sprintf("La etiqueta del campo %d no puede superar los %d caracteres.", position, maxDebtInformationFieldLabel))
		}
		if !isDebtInformationFieldType(field.Type) {
			return newValidationError(fmt.Sprintf("El tipo del campo '%s' no es válido.", field.Label))
		}
		if utf8.RuneCountInString(field.HelpText) > maxDebtInformationFieldHelp {
			return newValidationError(fmt.Sprintf("La ayuda del campo '%s' no puede superar los %d caracteres.", field.Label, maxDebtInformationFieldHelp))
		}

		if field.Type == models.DebtInformationFieldComputed {
			if field.ValueTemplate == "" {
				return newValidationError(fmt.Sprintf("El campo calculado '%s' necesita una expresión.", field.Label))
			}
			if utf8.RuneCountInString(field.ValueTemplate) > maxDebtInformationTemplate {
				return newValidationError(fmt.Sprintf("La expresión del campo '%s' no puede superar los %d caracteres.", field.Label, maxDebtInformationTemplate))
			}
			if err := validateDebtInformationValueTemplate(field.ValueTemplate); err != nil {
				return err
			}
			continue
		}

		if utf8.RuneCountInString(field.Placeholder) > maxDebtInformationPlaceholder {
			return newValidationError(fmt.Sprintf("El placeholder del campo '%s' no puede superar los %d caracteres.", field.Label, maxDebtInformationPlaceholder))
		}
		if utf8.RuneCountInString(field.DefaultValue) > maxDebtInformationDefault {
			return newValidationError(fmt.Sprintf("El valor por defecto del campo '%s' no puede superar los %d caracteres.", field.Label, maxDebtInformationDefault))
		}
		if err := validateDebtInformationDefaultValue(field); err != nil {
			return err
		}

		if field.Type == models.DebtInformationFieldSelect {
			if len(field.Options) == 0 {
				return newValidationError(fmt.Sprintf("El campo '%s' necesita al menos una opción.", field.Label))
			}
			if len(field.Options) > maxDebtInformationOptions {
				return newValidationError(fmt.Sprintf("El campo '%s' no puede tener más de %d opciones.", field.Label, maxDebtInformationOptions))
			}
			seenOptions := make(map[string]bool, len(field.Options))
			for _, option := range field.Options {
				if option == "" {
					return newValidationError(fmt.Sprintf("El campo '%s' tiene una opción vacía.", field.Label))
				}
				if utf8.RuneCountInString(option) > maxDebtInformationOption {
					return newValidationError(fmt.Sprintf("La opción '%s' del campo '%s' no puede superar los %d caracteres.", option, field.Label, maxDebtInformationOption))
				}
				if seenOptions[option] {
					return newValidationError(fmt.Sprintf("La opción '%s' está repetida en el campo '%s'.", option, field.Label))
				}
				seenOptions[option] = true
			}
			if field.DefaultValue != "" && !seenOptions[field.DefaultValue] {
				return newValidationError(fmt.Sprintf("El valor por defecto del campo '%s' debe ser una de sus opciones.", field.Label))
			}
		}

		if field.Type == models.DebtInformationFieldDate && field.DefaultValue != "" {
			if _, err := time.Parse("2006-01-02", field.DefaultValue); err != nil {
				return newValidationError(fmt.Sprintf("El valor por defecto del campo '%s' debe ser una fecha YYYY-MM-DD.", field.Label))
			}
		}
		if field.Type == models.DebtInformationFieldNumber && field.DefaultValue != "" {
			if _, err := strconv.ParseFloat(field.DefaultValue, 64); err != nil {
				return newValidationError(fmt.Sprintf("El valor por defecto del campo '%s' debe ser un número.", field.Label))
			}
		}
		if field.Type == models.DebtInformationFieldCheckbox && field.DefaultValue != "" {
			if field.DefaultValue != "true" && field.DefaultValue != "false" {
				return newValidationError(fmt.Sprintf("El valor por defecto del campo '%s' debe ser true o false.", field.Label))
			}
		}
	}

	fieldsJSON, err := json.Marshal(input.Fields)
	if err != nil {
		return newValidationError("No se pudieron serializar los campos de la plantilla.")
	}
	if len(fieldsJSON) > maxDebtInformationFieldsJSON {
		return newValidationError("La definición de la plantilla es demasiado grande.")
	}
	return nil
}

func validateDebtInformationValueTemplate(value string) error {
	for _, match := range debtInformationTokenRegex.FindAllStringSubmatch(value, -1) {
		if !debtInformationTokenSet[match[1]] {
			return newInvalidTokenError(fmt.Sprintf("El token {{%s}} no es válido.", match[1]))
		}
	}
	rest := debtInformationTokenRegex.ReplaceAllString(value, "")
	if strings.Contains(rest, "{{") || strings.Contains(rest, "}}") {
		return newInvalidTokenError("La expresión tiene un token mal formado. Usá el formato {{token}}.")
	}
	return nil
}

// validateVariableText valida que un texto solo use variables del catálogo
// permitido. Traduce llaves dobles y llaves mal formadas a invalid_variable.
func validateVariableText(value string, allowed map[string]bool) error {
	if value == "" {
		return nil
	}
	if strings.Contains(value, "{{") || strings.Contains(value, "}}") {
		return newInvalidVariableError("El valor usa llaves dobles. En este campo usá el formato {variable}.")
	}
	for _, match := range debtInformationVariableRegex.FindAllStringSubmatch(value, -1) {
		if !allowed[match[1]] {
			return newInvalidVariableError(fmt.Sprintf("La variable {%s} no es válida.", match[1]))
		}
	}
	rest := debtInformationVariableRegex.ReplaceAllString(value, "")
	if strings.Contains(rest, "{") || strings.Contains(rest, "}") {
		return newInvalidVariableError("La variable está mal formada. Usá el formato {variable}.")
	}
	return nil
}

func validateDebtInformationDefaultValue(field *models.DebtInformationField) error {
	if field.DefaultValue == "" {
		return nil
	}
	if field.Type == models.DebtInformationFieldText || field.Type == models.DebtInformationFieldTextarea {
		if err := validateVariableText(field.DefaultValue, debtInformationValueVariableSet); err != nil {
			var validation *DebtInformationValidationError
			if errors.As(err, &validation) {
				return newInvalidVariableError(fmt.Sprintf("%s (campo '%s')", validation.Message, field.Label))
			}
			return err
		}
		return nil
	}
	if strings.Contains(field.DefaultValue, "{") || strings.Contains(field.DefaultValue, "}") {
		return newInvalidVariableError(fmt.Sprintf("El campo '%s' de tipo %s no admite variables en el valor por defecto.", field.Label, field.Type))
	}
	return nil
}

func isDebtInformationFieldType(value string) bool {
	for _, fieldType := range models.DebtInformationFieldTypes {
		if fieldType == value {
			return true
		}
	}
	return false
}

func validateDebtInformationValues(fields []models.DebtInformationField, input map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for i := range fields {
		field := &fields[i]
		if field.Type == models.DebtInformationFieldComputed {
			continue
		}

		raw, present := input[field.Key]
		if !present || raw == nil || isEmptyDebtInformationValue(raw) {
			raw = nil
			if field.DefaultValue != "" {
				raw = field.DefaultValue
			} else if field.Type == models.DebtInformationFieldCheckbox {
				raw = false
			}
		}

		if raw == nil || isEmptyDebtInformationValue(raw) {
			if field.Required {
				return nil, newValidationError(fmt.Sprintf("%s es obligatorio.", debtInformationFieldName(field)))
			}
			if field.Type == models.DebtInformationFieldCheckbox {
				result[field.Key] = false
			}
			continue
		}

		switch field.Type {
		case models.DebtInformationFieldText:
			value, ok := raw.(string)
			if !ok {
				return nil, newValidationError(fmt.Sprintf("%s debe ser texto.", debtInformationFieldName(field)))
			}
			value = strings.TrimSpace(value)
			if value == "" {
				if field.Required {
					return nil, newValidationError(fmt.Sprintf("%s es obligatorio.", debtInformationFieldName(field)))
				}
				continue
			}
			if utf8.RuneCountInString(value) > maxDebtInformationText {
				return nil, newValidationError(fmt.Sprintf("%s no puede superar los %d caracteres.", debtInformationFieldName(field), maxDebtInformationText))
			}
			result[field.Key] = value

		case models.DebtInformationFieldTextarea:
			value, ok := raw.(string)
			if !ok {
				return nil, newValidationError(fmt.Sprintf("%s debe ser texto.", debtInformationFieldName(field)))
			}
			value = strings.TrimSpace(value)
			if value == "" {
				if field.Required {
					return nil, newValidationError(fmt.Sprintf("%s es obligatorio.", debtInformationFieldName(field)))
				}
				continue
			}
			if utf8.RuneCountInString(value) > maxDebtInformationTextarea {
				return nil, newValidationError(fmt.Sprintf("%s no puede superar los %d caracteres.", debtInformationFieldName(field), maxDebtInformationTextarea))
			}
			result[field.Key] = value

		case models.DebtInformationFieldNumber:
			value, err := toDebtInformationNumber(raw)
			if err != nil {
				return nil, newValidationError(fmt.Sprintf("%s debe ser un número.", debtInformationFieldName(field)))
			}
			result[field.Key] = value

		case models.DebtInformationFieldDate:
			value, ok := raw.(string)
			if !ok {
				return nil, newValidationError(fmt.Sprintf("%s debe ser una fecha YYYY-MM-DD.", debtInformationFieldName(field)))
			}
			value = strings.TrimSpace(value)
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return nil, newValidationError(fmt.Sprintf("%s debe ser una fecha YYYY-MM-DD.", debtInformationFieldName(field)))
			}
			result[field.Key] = value

		case models.DebtInformationFieldSelect:
			value, ok := raw.(string)
			if !ok {
				return nil, newValidationError(fmt.Sprintf("%s debe ser una opción válida.", debtInformationFieldName(field)))
			}
			value = strings.TrimSpace(value)
			if !containsDebtInformationOption(field.Options, value) {
				return nil, newValidationError(fmt.Sprintf("%s debe ser una opción válida.", debtInformationFieldName(field)))
			}
			result[field.Key] = value

		case models.DebtInformationFieldCheckbox:
			value, ok := toDebtInformationBool(raw)
			if !ok {
				return nil, newValidationError(fmt.Sprintf("%s debe ser verdadero o falso.", debtInformationFieldName(field)))
			}
			if field.Required && !value {
				return nil, newValidationError(fmt.Sprintf("%s es obligatorio.", debtInformationFieldName(field)))
			}
			result[field.Key] = value

		default:
			return nil, newValidationError(fmt.Sprintf("El tipo del campo %s no es válido.", debtInformationFieldName(field)))
		}
	}

	valuesJSON, err := json.Marshal(result)
	if err != nil {
		return nil, newValidationError("No se pudieron serializar los valores de la forma de pago.")
	}
	if len(valuesJSON) > maxDebtInformationValuesJSON {
		return nil, newValidationError("Los valores de la forma de pago son demasiado grandes.")
	}
	return result, nil
}

func isEmptyDebtInformationValue(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case nil:
		return true
	default:
		return false
	}
}

func toDebtInformationNumber(value any) (float64, error) {
	var number float64
	switch typed := value.(type) {
	case float64:
		number = typed
	case float32:
		number = float64(typed)
	case int:
		number = float64(typed)
	case int64:
		number = float64(typed)
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, err
		}
		number = parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, err
		}
		number = parsed
	default:
		return 0, fmt.Errorf("tipo de número no soportado")
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("número inválido")
	}
	return number, nil
}

func toDebtInformationBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func containsDebtInformationOption(options []string, value string) bool {
	for _, option := range options {
		if option == value {
			return true
		}
	}
	return false
}

func debtInformationFieldName(field *models.DebtInformationField) string {
	if field.Label != "" {
		return fmt.Sprintf("El campo '%s'", field.Label)
	}
	return fmt.Sprintf("El campo '%s'", field.Key)
}
