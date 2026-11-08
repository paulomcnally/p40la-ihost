package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

var (
	// ErrDebtInformationTemplateNotFound indica que la plantilla no existe.
	ErrDebtInformationTemplateNotFound = errors.New("plantilla de información no encontrada")
	// ErrDebtInformationTemplateNameExists indica un nombre duplicado.
	ErrDebtInformationTemplateNameExists = errors.New("ya existe una plantilla de información con ese nombre")
	// ErrDebtInformationTemplateInUse indica que la plantilla está asignada.
	ErrDebtInformationTemplateInUse = errors.New("la plantilla de información está asignada a una o más deudas")
)

// DebtInformationStorage encapsula el acceso a las plantillas de información
// y a la asignación de forma de pago por deuda (SPEC-131).
type DebtInformationStorage struct {
	db *sql.DB
}

// NewDebtInformationStorage crea un nuevo storage.
func NewDebtInformationStorage(db *sql.DB) *DebtInformationStorage {
	return &DebtInformationStorage{db: db}
}

const debtInformationTemplateColumns = `
	t.id, t.name, t.description, t.fields_json, t.created_at, t.updated_at,
	(SELECT COUNT(*) FROM debt_payment_information p WHERE p.template_id = t.id) AS usage_count
`

// ListTemplates devuelve las plantillas ordenadas por nombre, sin los campos
// completos (solo field_count y usage_count).
func (s *DebtInformationStorage) ListTemplates(ctx context.Context) ([]models.DebtInformationTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+debtInformationTemplateColumns+`
		FROM debt_information_templates t
		ORDER BY t.name COLLATE NOCASE
	`)
	if err != nil {
		return nil, fmt.Errorf("listar plantillas de información: %w", err)
	}
	defer rows.Close()

	var templates []models.DebtInformationTemplate
	for rows.Next() {
		template, err := scanDebtInformationTemplate(rows, false)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar plantillas de información: %w", err)
	}
	return templates, nil
}

// GetTemplateByID devuelve una plantilla con sus campos completos.
func (s *DebtInformationStorage) GetTemplateByID(ctx context.Context, id int64) (*models.DebtInformationTemplate, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+debtInformationTemplateColumns+`
		FROM debt_information_templates t
		WHERE t.id = ?
	`, id)
	return scanDebtInformationTemplateRow(row, true)
}

// GetTemplateByName busca una plantilla por nombre sin distinguir mayúsculas.
func (s *DebtInformationStorage) GetTemplateByName(ctx context.Context, name string) (*models.DebtInformationTemplate, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+debtInformationTemplateColumns+`
		FROM debt_information_templates t
		WHERE t.name = ? COLLATE NOCASE
	`, strings.TrimSpace(name))
	return scanDebtInformationTemplateRow(row, true)
}

// CreateTemplate inserta una plantilla y devuelve la fila creada.
func (s *DebtInformationStorage) CreateTemplate(ctx context.Context, input *models.DebtInformationTemplate) (*models.DebtInformationTemplate, error) {
	existing, err := s.GetTemplateByName(ctx, input.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDebtInformationTemplateNameExists
	}

	fieldsJSON, err := marshalDebtInformationFields(input.Fields)
	if err != nil {
		return nil, err
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO debt_information_templates (name, description, fields_json)
		VALUES (?, ?, ?)
	`, strings.TrimSpace(input.Name), strings.TrimSpace(input.Description), fieldsJSON)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrDebtInformationTemplateNameExists
		}
		return nil, fmt.Errorf("insertar plantilla de información: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de plantilla de información: %w", err)
	}
	return s.GetTemplateByID(ctx, id)
}

// UpdateTemplate actualiza una plantilla existente.
func (s *DebtInformationStorage) UpdateTemplate(ctx context.Context, input *models.DebtInformationTemplate) (*models.DebtInformationTemplate, error) {
	existing, err := s.GetTemplateByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrDebtInformationTemplateNotFound
	}

	duplicate, err := s.GetTemplateByName(ctx, input.Name)
	if err != nil {
		return nil, err
	}
	if duplicate != nil && duplicate.ID != input.ID {
		return nil, ErrDebtInformationTemplateNameExists
	}

	fieldsJSON, err := marshalDebtInformationFields(input.Fields)
	if err != nil {
		return nil, err
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE debt_information_templates
		SET name = ?, description = ?, fields_json = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, strings.TrimSpace(input.Name), strings.TrimSpace(input.Description), fieldsJSON, input.ID); err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrDebtInformationTemplateNameExists
		}
		return nil, fmt.Errorf("actualizar plantilla de información: %w", err)
	}
	return s.GetTemplateByID(ctx, input.ID)
}

// DeleteTemplate elimina una plantilla si no está asignada a ninguna deuda.
func (s *DebtInformationStorage) DeleteTemplate(ctx context.Context, id int64) error {
	existing, err := s.GetTemplateByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrDebtInformationTemplateNotFound
	}
	usage, err := s.CountTemplateUsage(ctx, id)
	if err != nil {
		return err
	}
	if usage > 0 {
		return ErrDebtInformationTemplateInUse
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM debt_information_templates WHERE id = ?`, id); err != nil {
		return fmt.Errorf("eliminar plantilla de información: %w", err)
	}
	return nil
}

// CountTemplateUsage devuelve cuántas deudas tienen asignada la plantilla.
func (s *DebtInformationStorage) CountTemplateUsage(ctx context.Context, templateID int64) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM debt_payment_information WHERE template_id = ?`, templateID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("contar uso de plantilla de información: %w", err)
	}
	return count, nil
}

// GetPaymentInformationByDebtID devuelve la asignación de una deuda o nil si
// no tiene forma de pago configurada.
func (s *DebtInformationStorage) GetPaymentInformationByDebtID(ctx context.Context, debtID int64) (*models.DebtPaymentInformation, error) {
	var (
		templateID int64
		valuesJSON string
		createdAt  sql.NullTime
		updatedAt  sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT template_id, values_json, created_at, updated_at
		FROM debt_payment_information
		WHERE debt_id = ?
	`, debtID).Scan(&templateID, &valuesJSON, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("obtener forma de pago de deuda: %w", err)
	}

	template, err := s.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("plantilla de información %d no encontrada para la deuda %d", templateID, debtID)
	}

	values, err := unmarshalDebtInformationValues(valuesJSON)
	if err != nil {
		return nil, err
	}

	result := &models.DebtPaymentInformation{
		DebtID:     debtID,
		TemplateID: &templateID,
		Template:   template,
		Values:     values,
	}
	if createdAt.Valid {
		result.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		result.UpdatedAt = &updatedAt.Time
	}
	return result, nil
}

// UpsertPaymentInformation crea o actualiza la forma de pago de una deuda.
func (s *DebtInformationStorage) UpsertPaymentInformation(
	ctx context.Context,
	debtID int64,
	templateID int64,
	values map[string]any,
) (*models.DebtPaymentInformation, error) {
	valuesJSON, err := marshalDebtInformationValues(values)
	if err != nil {
		return nil, err
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO debt_payment_information (debt_id, template_id, values_json)
		VALUES (?, ?, ?)
		ON CONFLICT(debt_id) DO UPDATE SET
			template_id = excluded.template_id,
			values_json = excluded.values_json,
			updated_at = CURRENT_TIMESTAMP
	`, debtID, templateID, valuesJSON); err != nil {
		return nil, fmt.Errorf("guardar forma de pago de deuda: %w", err)
	}
	return s.GetPaymentInformationByDebtID(ctx, debtID)
}

// DeletePaymentInformation quita la forma de pago de una deuda.
func (s *DebtInformationStorage) DeletePaymentInformation(ctx context.Context, debtID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM debt_payment_information WHERE debt_id = ?`, debtID,
	); err != nil {
		return fmt.Errorf("quitar forma de pago de deuda: %w", err)
	}
	return nil
}

func scanDebtInformationTemplate(rows *sql.Rows, includeFields bool) (*models.DebtInformationTemplate, error) {
	var (
		template   models.DebtInformationTemplate
		fieldsJSON string
	)
	if err := rows.Scan(&template.ID, &template.Name, &template.Description, &fieldsJSON,
		&template.CreatedAt, &template.UpdatedAt, &template.UsageCount); err != nil {
		return nil, fmt.Errorf("escanear plantilla de información: %w", err)
	}
	fields, err := unmarshalDebtInformationFields(fieldsJSON)
	if err != nil {
		return nil, err
	}
	template.FieldCount = len(fields)
	if includeFields {
		template.Fields = fields
	}
	return &template, nil
}

func scanDebtInformationTemplateRow(row *sql.Row, includeFields bool) (*models.DebtInformationTemplate, error) {
	var (
		template   models.DebtInformationTemplate
		fieldsJSON string
	)
	if err := row.Scan(&template.ID, &template.Name, &template.Description, &fieldsJSON,
		&template.CreatedAt, &template.UpdatedAt, &template.UsageCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear plantilla de información: %w", err)
	}
	fields, err := unmarshalDebtInformationFields(fieldsJSON)
	if err != nil {
		return nil, err
	}
	template.FieldCount = len(fields)
	if includeFields {
		template.Fields = fields
	}
	return &template, nil
}

func marshalDebtInformationFields(fields []models.DebtInformationField) (string, error) {
	if fields == nil {
		return "[]", nil
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("serializar campos de plantilla: %w", err)
	}
	return string(data), nil
}

func unmarshalDebtInformationFields(raw string) ([]models.DebtInformationField, error) {
	if strings.TrimSpace(raw) == "" {
		return []models.DebtInformationField{}, nil
	}
	var fields []models.DebtInformationField
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, fmt.Errorf("leer campos de plantilla: %w", err)
	}
	if fields == nil {
		fields = []models.DebtInformationField{}
	}
	return fields, nil
}

func marshalDebtInformationValues(values map[string]any) (string, error) {
	if values == nil {
		return "{}", nil
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("serializar valores de forma de pago: %w", err)
	}
	return string(data), nil
}

func unmarshalDebtInformationValues(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("leer valores de forma de pago: %w", err)
	}
	if values == nil {
		values = map[string]any{}
	}
	return values, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed")
}
