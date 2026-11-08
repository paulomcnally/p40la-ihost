package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

var (
	// ErrRecognitionNotFound indica que el reconocimiento no existe.
	ErrRecognitionNotFound = errors.New("reconocimiento no encontrado")
	// ErrRecognitionAlreadyAssociated indica que el reconocimiento ya fue asociado.
	ErrRecognitionAlreadyAssociated = errors.New("el reconocimiento ya fue asociado")
	// ErrDebtBillNotPayable indica que la cuota no existe, está eliminada o no está pagada.
	ErrDebtBillNotPayable = errors.New("la cuota no existe o no está pagada")
)

// DocumentRecognitionStorage encapsula el acceso a document_recognitions.
type DocumentRecognitionStorage struct {
	db *sql.DB
}

// NewDocumentRecognitionStorage crea un nuevo storage de reconocimientos.
func NewDocumentRecognitionStorage(db *sql.DB) *DocumentRecognitionStorage {
	return &DocumentRecognitionStorage{db: db}
}

const documentRecognitionColumns = `
	id, document_hash, recognizer_id, document_type, status, reference,
	paid_at, amount, currency, installment_number, concept_text,
	entity_type, entity_id, confidence, warnings, inbox_path, final_path,
	created_at, updated_at, confirmed_at
`

// Create inserta una vista previa de reconocimiento y devuelve la fila creada.
func (s *DocumentRecognitionStorage) Create(ctx context.Context, recognition *models.DocumentRecognition) (*models.DocumentRecognition, error) {
	warnings, err := marshalWarnings(recognition.Warnings)
	if err != nil {
		return nil, err
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO document_recognitions (
			document_hash, recognizer_id, document_type, status, reference,
			paid_at, amount, currency, installment_number, concept_text,
			entity_type, entity_id, confidence, warnings, inbox_path, final_path
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, recognition.DocumentHash, recognition.RecognizerID, recognition.DocumentType,
		recognition.Status, recognition.Reference, paidAtValue(recognition.PaidAt), recognition.Amount,
		recognition.Currency, nullableInt(recognition.InstallmentNumber), recognition.ConceptText,
		recognition.EntityType, nullableInt64(recognition.EntityID), recognition.Confidence,
		warnings, recognition.InboxPath, recognition.FinalPath)
	if err != nil {
		return nil, fmt.Errorf("insertar reconocimiento: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de reconocimiento: %w", err)
	}
	return s.GetByID(ctx, id)
}

// GetByID busca un reconocimiento por ID.
func (s *DocumentRecognitionStorage) GetByID(ctx context.Context, id int64) (*models.DocumentRecognition, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+documentRecognitionColumns+`
		FROM document_recognitions
		WHERE id = ?
	`, id)
	return scanDocumentRecognition(row)
}

// GetByHash busca un reconocimiento por el SHA-256 del PDF.
func (s *DocumentRecognitionStorage) GetByHash(ctx context.Context, hash string) (*models.DocumentRecognition, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+documentRecognitionColumns+`
		FROM document_recognitions
		WHERE document_hash = ?
	`, hash)
	return scanDocumentRecognition(row)
}

// SetInboxPath guarda la ruta del PDF temporal del reconocimiento.
func (s *DocumentRecognitionStorage) SetInboxPath(ctx context.Context, id int64, inboxPath string) error {
	if _, err := s.db.ExecContext(ctx, `
		UPDATE document_recognitions
		SET inbox_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, inboxPath, id); err != nil {
		return fmt.Errorf("guardar ruta temporal de reconocimiento: %w", err)
	}
	return nil
}

// Delete elimina un reconocimiento (se usa solo para limpiar una vista previa
// que no pudo completar su archivo temporal).
func (s *DocumentRecognitionStorage) Delete(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM document_recognitions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("eliminar reconocimiento: %w", err)
	}
	return nil
}

// Associate confirma la asociación y actualiza la referencia de la cuota en
// una sola transacción (SPEC-127, REQ-008).
func (s *DocumentRecognitionStorage) Associate(
	ctx context.Context,
	recognitionID int64,
	debtBillID int64,
	reference string,
	paidAt *time.Time,
	amount float64,
	currency string,
	installmentNumber int,
	concept string,
	entityType string,
	finalPath string,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transacción de asociación: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	billResult, err := tx.ExecContext(ctx, `
		UPDATE debt_bills
		SET payment_reference = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'paid' AND deleted_at IS NULL
	`, reference, debtBillID)
	if err != nil {
		return fmt.Errorf("actualizar referencia de la cuota: %w", err)
	}
	affected, err := billResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar actualización de la cuota: %w", err)
	}
	if affected == 0 {
		return ErrDebtBillNotPayable
	}

	recognitionResult, err := tx.ExecContext(ctx, `
		UPDATE document_recognitions
		SET status = 'associated',
		    reference = ?,
		    paid_at = ?,
		    amount = ?,
		    currency = ?,
		    installment_number = ?,
		    concept_text = ?,
		    entity_type = ?,
		    entity_id = ?,
		    final_path = ?,
		    confirmed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'recognized'
	`, reference, paidAtValue(paidAt), amount, currency, nullableInt(installmentNumber), concept,
		entityType, debtBillID, finalPath, recognitionID)
	if err != nil {
		return fmt.Errorf("marcar reconocimiento como asociado: %w", err)
	}
	affected, err = recognitionResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar asociación del reconocimiento: %w", err)
	}
	if affected == 0 {
		return ErrRecognitionAlreadyAssociated
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar transacción de asociación: %w", err)
	}
	return nil
}

func marshalWarnings(warnings []string) (string, error) {
	if warnings == nil {
		return "[]", nil
	}
	data, err := json.Marshal(warnings)
	if err != nil {
		return "", fmt.Errorf("serializar advertencias: %w", err)
	}
	return string(data), nil
}

// paidAtValue guarda la fecha como RFC3339 para conservar el offset del
// comprobante. El driver SQLite devuelve como texto los time.Time con zona
// distinta de UTC, por lo que se almacena y parsea explícitamente.
func paidAtValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(time.RFC3339)
}

func parseStoredTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	return nil
}

func nullableInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}

func nullableInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func scanDocumentRecognition(row *sql.Row) (*models.DocumentRecognition, error) {
	var recognition models.DocumentRecognition
	var installment, entityID sql.NullInt64
	var paidAt sql.NullString
	var confirmedAt sql.NullTime
	var reference, currency, concept, entityType, inboxPath, finalPath, warnings string

	if err := row.Scan(
		&recognition.ID, &recognition.DocumentHash, &recognition.RecognizerID,
		&recognition.DocumentType, &recognition.Status, &reference,
		&paidAt, &recognition.Amount, &currency, &installment, &concept,
		&entityType, &entityID, &recognition.Confidence, &warnings,
		&inboxPath, &finalPath, &recognition.CreatedAt, &recognition.UpdatedAt,
		&confirmedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear reconocimiento: %w", err)
	}

	recognition.Reference = reference
	recognition.Currency = currency
	recognition.ConceptText = concept
	recognition.EntityType = entityType
	recognition.InboxPath = inboxPath
	recognition.FinalPath = finalPath
	if installment.Valid {
		recognition.InstallmentNumber = int(installment.Int64)
	}
	if entityID.Valid {
		recognition.EntityID = entityID.Int64
	}
	if paidAt.Valid {
		recognition.PaidAt = parseStoredTime(paidAt.String)
	}
	if confirmedAt.Valid {
		recognition.ConfirmedAt = &confirmedAt.Time
	}
	if warnings != "" {
		_ = json.Unmarshal([]byte(warnings), &recognition.Warnings)
	}
	return &recognition, nil
}
