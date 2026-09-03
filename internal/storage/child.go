package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// ChildStorage encapsula el acceso a la tabla children.
type ChildStorage struct {
	db *sql.DB
}

// NewChildStorage crea un nuevo ChildStorage.
func NewChildStorage(db *sql.DB) *ChildStorage {
	return &ChildStorage{db: db}
}

// List devuelve todos los hijos.
func (s *ChildStorage) List(ctx context.Context) ([]models.Child, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, first_name, last_name, birth_date, notes, created_at, updated_at
		FROM children
		ORDER BY birth_date
	`)
	if err != nil {
		return nil, fmt.Errorf("listar hijos: %w", err)
	}
	defer rows.Close()

	return scanChildren(rows)
}

// GetByID busca un hijo por su ID.
func (s *ChildStorage) GetByID(ctx context.Context, id int64) (*models.Child, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, birth_date, notes, created_at, updated_at
		FROM children
		WHERE id = ?
	`, id)
	return scanChild(row)
}

// Create inserta un nuevo hijo.
func (s *ChildStorage) Create(ctx context.Context, firstName, lastName, birthDate, notes string) (*models.Child, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO children (first_name, last_name, birth_date, notes) VALUES (?, ?, ?, ?)
	`, firstName, lastName, birthDate, notes)
	if err != nil {
		return nil, fmt.Errorf("insertar hijo: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de hijo: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza un hijo existente.
func (s *ChildStorage) Update(ctx context.Context, id int64, firstName, lastName, birthDate, notes string) (*models.Child, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE children
		SET first_name = ?, last_name = ?, birth_date = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, firstName, lastName, birthDate, notes, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar hijo: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina un hijo.
func (s *ChildStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM children WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar hijo: %w", err)
	}
	return nil
}

func scanChild(row *sql.Row) (*models.Child, error) {
	var c models.Child
	var notes sql.NullString
	if err := row.Scan(&c.ID, &c.FirstName, &c.LastName, &c.BirthDate, &notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear hijo: %w", err)
	}
	c.Notes = notes.String
	return &c, nil
}

func scanChildren(rows *sql.Rows) ([]models.Child, error) {
	var children []models.Child
	for rows.Next() {
		var c models.Child
		var notes sql.NullString
		if err := rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.BirthDate, &notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear hijo: %w", err)
		}
		c.Notes = notes.String
		children = append(children, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return children, nil
}