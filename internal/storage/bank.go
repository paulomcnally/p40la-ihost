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
	// ErrBankNotFound indica que el banco no existe.
	ErrBankNotFound = errors.New("banco no encontrado")
	// ErrBankNameExists indica un nombre de banco duplicado.
	ErrBankNameExists = errors.New("ya existe un banco con ese nombre")
	// ErrBankHasAccounts indica que el banco todavía tiene cuentas.
	ErrBankHasAccounts = errors.New("el banco tiene cuentas asociadas")
	// ErrBankAccountNotFound indica que la cuenta no existe.
	ErrBankAccountNotFound = errors.New("cuenta bancaria no encontrada")
	// ErrBankAccountDuplicate indica un número de cuenta repetido en el banco.
	ErrBankAccountDuplicate = errors.New("ya existe una cuenta con ese número en el banco")
	// ErrBankTransferFormatNotFound indica que el formato no existe.
	ErrBankTransferFormatNotFound = errors.New("formato de transferencia no encontrado")
)

// BankStorage encapsula el acceso a bancos, cuentas y formatos (SPEC-133).
type BankStorage struct {
	db *sql.DB
}

// NewBankStorage crea un nuevo storage.
func NewBankStorage(db *sql.DB) *BankStorage {
	return &BankStorage{db: db}
}

const bankColumns = `
	b.id, b.name, b.short_code, b.country, b.notes, b.active, b.created_at, b.updated_at,
	(SELECT COUNT(*) FROM bank_accounts a WHERE a.bank_id = b.id) AS account_count
`

// ListBanks devuelve todos los bancos con la cantidad de cuentas.
func (s *BankStorage) ListBanks(ctx context.Context) ([]models.Bank, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+bankColumns+`
		FROM banks b
		ORDER BY b.name COLLATE NOCASE
	`)
	if err != nil {
		return nil, fmt.Errorf("listar bancos: %w", err)
	}
	defer rows.Close()

	var banks []models.Bank
	for rows.Next() {
		bank, err := scanBank(rows)
		if err != nil {
			return nil, err
		}
		banks = append(banks, *bank)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar bancos: %w", err)
	}
	return banks, nil
}

// GetBankByID devuelve un banco o nil si no existe.
func (s *BankStorage) GetBankByID(ctx context.Context, id int64) (*models.Bank, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+bankColumns+`
		FROM banks b
		WHERE b.id = ?
	`, id)
	bank, err := scanBank(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return bank, err
}

// GetBankByName busca un banco por nombre sin distinguir mayúsculas.
func (s *BankStorage) GetBankByName(ctx context.Context, name string) (*models.Bank, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+bankColumns+`
		FROM banks b
		WHERE b.name = ? COLLATE NOCASE
	`, strings.TrimSpace(name))
	bank, err := scanBank(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return bank, err
}

// CreateBank inserta un banco.
func (s *BankStorage) CreateBank(ctx context.Context, bank *models.Bank) (*models.Bank, error) {
	existing, err := s.GetBankByName(ctx, bank.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrBankNameExists
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO banks (name, short_code, country, notes, active)
		VALUES (?, ?, ?, ?, ?)
	`, bank.Name, bank.ShortCode, bank.Country, bank.Notes, bank.Active)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrBankNameExists
		}
		return nil, fmt.Errorf("crear banco: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de banco: %w", err)
	}
	return s.GetBankByID(ctx, id)
}

// UpdateBank actualiza un banco.
func (s *BankStorage) UpdateBank(ctx context.Context, bank *models.Bank) (*models.Bank, error) {
	existing, err := s.GetBankByID(ctx, bank.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrBankNotFound
	}
	duplicate, err := s.GetBankByName(ctx, bank.Name)
	if err != nil {
		return nil, err
	}
	if duplicate != nil && duplicate.ID != bank.ID {
		return nil, ErrBankNameExists
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE banks
		SET name = ?, short_code = ?, country = ?, notes = ?, active = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, bank.Name, bank.ShortCode, bank.Country, bank.Notes, bank.Active, bank.ID); err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrBankNameExists
		}
		return nil, fmt.Errorf("actualizar banco: %w", err)
	}
	return s.GetBankByID(ctx, bank.ID)
}

// DeleteBank elimina un banco si no tiene cuentas.
func (s *BankStorage) DeleteBank(ctx context.Context, id int64) error {
	existing, err := s.GetBankByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrBankNotFound
	}
	if existing.AccountCount > 0 {
		return ErrBankHasAccounts
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM banks WHERE id = ?`, id); err != nil {
		return fmt.Errorf("eliminar banco: %w", err)
	}
	return nil
}

const bankAccountColumns = `
	a.id, a.bank_id, b.name, a.alias, a.account_number, a.account_type,
	a.currency_id, COALESCE(c.code, ''), COALESCE(c.symbol, ''),
	a.beneficiary, a.notes, a.active, a.created_at, a.updated_at
`

// ListAccountsByBank devuelve las cuentas de un banco (sin formatos).
func (s *BankStorage) ListAccountsByBank(ctx context.Context, bankID int64) ([]models.BankAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+bankAccountColumns+`
		FROM bank_accounts a
		JOIN banks b ON b.id = a.bank_id
		LEFT JOIN currencies c ON c.id = a.currency_id
		WHERE a.bank_id = ?
		ORDER BY a.alias COLLATE NOCASE
	`, bankID)
	if err != nil {
		return nil, fmt.Errorf("listar cuentas bancarias: %w", err)
	}
	defer rows.Close()

	var accounts []models.BankAccount
	for rows.Next() {
		account, err := scanBankAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar cuentas bancarias: %w", err)
	}
	return accounts, nil
}

// GetAccountByID devuelve una cuenta con sus formatos.
func (s *BankStorage) GetAccountByID(ctx context.Context, id int64) (*models.BankAccount, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+bankAccountColumns+`
		FROM bank_accounts a
		JOIN banks b ON b.id = a.bank_id
		LEFT JOIN currencies c ON c.id = a.currency_id
		WHERE a.id = ?
	`, id)
	account, err := scanBankAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	formats, err := s.ListFormats(ctx, id)
	if err != nil {
		return nil, err
	}
	account.Formats = formats
	return account, nil
}

// GetAccountByNumber busca una cuenta por banco y número.
func (s *BankStorage) GetAccountByNumber(ctx context.Context, bankID int64, number string) (*models.BankAccount, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+bankAccountColumns+`
		FROM bank_accounts a
		JOIN banks b ON b.id = a.bank_id
		LEFT JOIN currencies c ON c.id = a.currency_id
		WHERE a.bank_id = ? AND a.account_number = ?
	`, bankID, number)
	account, err := scanBankAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return account, err
}

// CreateAccount inserta una cuenta bancaria.
func (s *BankStorage) CreateAccount(ctx context.Context, account *models.BankAccount) (*models.BankAccount, error) {
	duplicate, err := s.GetAccountByNumber(ctx, account.BankID, account.AccountNumber)
	if err != nil {
		return nil, err
	}
	if duplicate != nil {
		return nil, ErrBankAccountDuplicate
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO bank_accounts
			(bank_id, alias, account_number, account_type, currency_id, beneficiary, notes, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, account.BankID, account.Alias, account.AccountNumber, account.AccountType,
		account.CurrencyID, account.Beneficiary, account.Notes, account.Active)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrBankAccountDuplicate
		}
		return nil, fmt.Errorf("crear cuenta bancaria: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de cuenta bancaria: %w", err)
	}
	return s.GetAccountByID(ctx, id)
}

// UpdateAccount actualiza una cuenta bancaria.
func (s *BankStorage) UpdateAccount(ctx context.Context, account *models.BankAccount) (*models.BankAccount, error) {
	existing, err := s.GetAccountByID(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrBankAccountNotFound
	}
	duplicate, err := s.GetAccountByNumber(ctx, account.BankID, account.AccountNumber)
	if err != nil {
		return nil, err
	}
	if duplicate != nil && duplicate.ID != account.ID {
		return nil, ErrBankAccountDuplicate
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE bank_accounts
		SET bank_id = ?, alias = ?, account_number = ?, account_type = ?,
		    currency_id = ?, beneficiary = ?, notes = ?, active = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, account.BankID, account.Alias, account.AccountNumber, account.AccountType,
		account.CurrencyID, account.Beneficiary, account.Notes, account.Active,
		account.ID); err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrBankAccountDuplicate
		}
		return nil, fmt.Errorf("actualizar cuenta bancaria: %w", err)
	}
	return s.GetAccountByID(ctx, account.ID)
}

// DeleteAccount elimina una cuenta y sus formatos.
func (s *BankStorage) DeleteAccount(ctx context.Context, id int64) error {
	existing, err := s.GetAccountByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrBankAccountNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar borrado de cuenta bancaria: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM bank_transfer_formats WHERE bank_account_id = ?`, id); err != nil {
		return fmt.Errorf("eliminar formatos de cuenta bancaria: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM bank_accounts WHERE id = ?`, id); err != nil {
		return fmt.Errorf("eliminar cuenta bancaria: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar borrado de cuenta bancaria: %w", err)
	}
	return nil
}

// ListFormats devuelve los formatos de una cuenta.
func (s *BankStorage) ListFormats(ctx context.Context, accountID int64) ([]models.BankTransferFormat, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, bank_account_id, kind, fields_json, created_at, updated_at
		FROM bank_transfer_formats
		WHERE bank_account_id = ?
		ORDER BY CASE kind WHEN 'transfer' THEN 0 ELSE 1 END
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("listar formatos de transferencia: %w", err)
	}
	defer rows.Close()

	var formats []models.BankTransferFormat
	for rows.Next() {
		format, err := scanBankTransferFormat(rows)
		if err != nil {
			return nil, err
		}
		formats = append(formats, *format)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar formatos de transferencia: %w", err)
	}
	return formats, nil
}

// GetFormat devuelve un formato por cuenta y tipo.
func (s *BankStorage) GetFormat(ctx context.Context, accountID int64, kind string) (*models.BankTransferFormat, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, bank_account_id, kind, fields_json, created_at, updated_at
		FROM bank_transfer_formats
		WHERE bank_account_id = ? AND kind = ?
	`, accountID, kind)
	format, err := scanBankTransferFormat(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return format, err
}

// UpsertFormat crea o actualiza un formato de una cuenta.
func (s *BankStorage) UpsertFormat(ctx context.Context, accountID int64, kind string, fields map[string]string) (*models.BankTransferFormat, error) {
	fieldsJSON, err := marshalBankTransferFields(fields)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO bank_transfer_formats (bank_account_id, kind, fields_json)
		VALUES (?, ?, ?)
		ON CONFLICT(bank_account_id, kind) DO UPDATE SET
			fields_json = excluded.fields_json,
			updated_at = CURRENT_TIMESTAMP
	`, accountID, kind, fieldsJSON); err != nil {
		return nil, fmt.Errorf("guardar formato de transferencia: %w", err)
	}
	return s.GetFormat(ctx, accountID, kind)
}

// DeleteFormat elimina un formato de una cuenta.
func (s *BankStorage) DeleteFormat(ctx context.Context, accountID int64, kind string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM bank_transfer_formats WHERE bank_account_id = ? AND kind = ?
	`, accountID, kind)
	if err != nil {
		return fmt.Errorf("eliminar formato de transferencia: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar borrado de formato: %w", err)
	}
	if affected == 0 {
		return ErrBankTransferFormatNotFound
	}
	return nil
}

func scanBank(scanner interface{ Scan(...any) error }) (*models.Bank, error) {
	var bank models.Bank
	if err := scanner.Scan(&bank.ID, &bank.Name, &bank.ShortCode, &bank.Country,
		&bank.Notes, &bank.Active, &bank.CreatedAt, &bank.UpdatedAt,
		&bank.AccountCount); err != nil {
		return nil, err
	}
	return &bank, nil
}

func scanBankAccount(scanner interface{ Scan(...any) error }) (*models.BankAccount, error) {
	var account models.BankAccount
	if err := scanner.Scan(&account.ID, &account.BankID, &account.BankName, &account.Alias,
		&account.AccountNumber, &account.AccountType, &account.CurrencyID,
		&account.CurrencyCode, &account.CurrencySymbol, &account.Beneficiary,
		&account.Notes, &account.Active, &account.CreatedAt, &account.UpdatedAt); err != nil {
		return nil, err
	}
	return &account, nil
}

func scanBankTransferFormat(scanner interface{ Scan(...any) error }) (*models.BankTransferFormat, error) {
	var (
		format     models.BankTransferFormat
		fieldsJSON string
	)
	if err := scanner.Scan(&format.ID, &format.BankAccountID, &format.Kind, &fieldsJSON,
		&format.CreatedAt, &format.UpdatedAt); err != nil {
		return nil, err
	}
	fields, err := unmarshalBankTransferFields(fieldsJSON)
	if err != nil {
		return nil, err
	}
	format.Fields = fields
	return &format, nil
}

func marshalBankTransferFields(fields map[string]string) (string, error) {
	if fields == nil {
		return "{}", nil
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("serializar formato de transferencia: %w", err)
	}
	return string(data), nil
}

func unmarshalBankTransferFields(raw string) (map[string]string, error) {
	fields := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return fields, nil
	}
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, fmt.Errorf("deserializar formato de transferencia: %w", err)
	}
	return fields, nil
}
