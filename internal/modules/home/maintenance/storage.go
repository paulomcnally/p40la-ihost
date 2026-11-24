package maintenance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Storage es el único punto con SQL del módulo (ARCH-MODULES).
type Storage struct {
	db *sql.DB
}

// NewStorage crea el storage del módulo.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

const maintenanceColumns = `
	m.id, m.home_id, m.performed_at, m.kind, m.area,
	m.institution_id, COALESCE(i.name, ''),
	m.description, m.created_at, m.updated_at
`

const maintenanceFrom = `
	FROM home_maintenances m
	LEFT JOIN institutions i ON i.id = m.institution_id
`

func scanMaintenance(row interface{ Scan(...any) error }) (*Maintenance, error) {
	var item Maintenance
	var institutionID sql.NullInt64
	err := row.Scan(
		&item.ID, &item.HomeID, &item.PerformedAt, &item.Kind, &item.Area,
		&institutionID, &item.InstitutionName, &item.Description,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear mantenimiento: %w", err)
	}
	if institutionID.Valid {
		item.InstitutionID = &institutionID.Int64
	}
	if item.TotalsByCurrency == nil {
		item.TotalsByCurrency = []CurrencyTotal{}
	}
	if item.Items == nil {
		item.Items = []Item{}
	}
	return &item, nil
}

func filterClause(homeID int64, filter Filter) (string, []any) {
	where := []string{"m.home_id = ?"}
	args := []any{homeID}
	if filter.Year > 0 {
		where = append(where, "substr(m.performed_at, 1, 4) = ?")
		args = append(args, fmt.Sprintf("%04d", filter.Year))
	}
	if filter.Date != "" {
		where = append(where, "m.performed_at = ?")
		args = append(args, filter.Date)
	}
	if filter.Kind != "" {
		where = append(where, "m.kind = ?")
		args = append(args, filter.Kind)
	}
	if filter.ItemType != "" {
		where = append(where, "EXISTS (SELECT 1 FROM home_maintenance_items fit WHERE fit.maintenance_id = m.id AND fit.item_type = ?)")
		args = append(args, filter.ItemType)
	}
	if filter.Query != "" {
		pattern := likePattern(filter.Query)
		where = append(where, `(m.description LIKE ? ESCAPE '\' OR EXISTS (SELECT 1 FROM home_maintenance_items qit WHERE qit.maintenance_id = m.id AND qit.description LIKE ? ESCAPE '\'))`)
		args = append(args, pattern, pattern)
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

func likePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(strings.TrimSpace(value)) + "%"
}

// List devuelve el histórico de una vivienda sin las líneas.
func (s *Storage) List(ctx context.Context, homeID int64, filter Filter) ([]Maintenance, error) {
	clause, args := filterClause(homeID, filter)
	query := `
		SELECT ` + maintenanceColumns + maintenanceFrom + clause + `
		ORDER BY m.performed_at DESC, m.id DESC
	`
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar mantenimientos: %w", err)
	}
	defer rows.Close()

	items := make([]Maintenance, 0)
	for rows.Next() {
		item, err := scanMaintenance(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer mantenimientos: %w", err)
	}
	return items, nil
}

// TotalsByMaintenance agrupa el gasto por moneda de cada registro del histórico.
func (s *Storage) TotalsByMaintenance(ctx context.Context, homeID int64, filter Filter) (map[int64][]CurrencyTotal, error) {
	clause, args := filterClause(homeID, filter)
	query := `
		SELECT it.maintenance_id, it.currency_id, c.code,
			COALESCE(SUM(CASE WHEN it.item_type = 'labor' THEN it.amount END), 0),
			COALESCE(SUM(CASE WHEN it.item_type = 'supplies' THEN it.amount END), 0)
		FROM home_maintenances m
		JOIN home_maintenance_items it ON it.maintenance_id = m.id
		JOIN currencies c ON c.id = it.currency_id
	` + clause + `
		GROUP BY it.maintenance_id, it.currency_id, c.code
		ORDER BY c.code
	`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("totalizar mantenimientos: %w", err)
	}
	defer rows.Close()

	grouped := make(map[int64][]CurrencyTotal)
	for rows.Next() {
		var maintenanceID int64
		var total CurrencyTotal
		if err := rows.Scan(&maintenanceID, &total.CurrencyID, &total.CurrencyCode,
			&total.LaborCost, &total.SuppliesCost); err != nil {
			return nil, fmt.Errorf("escanear total de mantenimiento: %w", err)
		}
		total.Total = total.LaborCost + total.SuppliesCost
		grouped[maintenanceID] = append(grouped[maintenanceID], total)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer totales de mantenimiento: %w", err)
	}
	return grouped, nil
}

// TotalsByCurrency agrupa todo el gasto del histórico por moneda.
func (s *Storage) TotalsByCurrency(ctx context.Context, homeID int64, filter Filter) ([]CurrencyTotal, error) {
	clause, args := filterClause(homeID, filter)
	query := `
		SELECT it.currency_id, c.code,
			COALESCE(SUM(CASE WHEN it.item_type = 'labor' THEN it.amount END), 0),
			COALESCE(SUM(CASE WHEN it.item_type = 'supplies' THEN it.amount END), 0)
		FROM home_maintenances m
		JOIN home_maintenance_items it ON it.maintenance_id = m.id
		JOIN currencies c ON c.id = it.currency_id
	` + clause + `
		GROUP BY it.currency_id, c.code
		ORDER BY c.code
	`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("totalizar histórico: %w", err)
	}
	defer rows.Close()

	totals := make([]CurrencyTotal, 0)
	for rows.Next() {
		var total CurrencyTotal
		if err := rows.Scan(&total.CurrencyID, &total.CurrencyCode,
			&total.LaborCost, &total.SuppliesCost); err != nil {
			return nil, fmt.Errorf("escanear total: %w", err)
		}
		total.Total = total.LaborCost + total.SuppliesCost
		totals = append(totals, total)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer totales: %w", err)
	}
	return totals, nil
}

// Get devuelve un mantenimiento de la vivienda o nil si no existe.
func (s *Storage) Get(ctx context.Context, homeID, id int64) (*Maintenance, error) {
	query := `
		SELECT ` + maintenanceColumns + maintenanceFrom + `
		WHERE m.home_id = ? AND m.id = ?
	`
	return scanMaintenance(s.db.QueryRowContext(ctx, query, homeID, id))
}

// Create inserta un mantenimiento y devuelve su ID.
func (s *Storage) Create(ctx context.Context, homeID int64, input Input) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO home_maintenances
			(home_id, performed_at, kind, area, institution_id, description)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		homeID, input.PerformedAt, input.Kind, input.Area,
		nullableID(input.InstitutionID), input.Description,
	)
	if err != nil {
		return 0, fmt.Errorf("crear mantenimiento: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("id de mantenimiento: %w", err)
	}
	return id, nil
}

// CreateWithItem inserta el mantenimiento y su primera línea en una sola
// transacción. Si la línea falla, no queda un mantenimiento vacío.
func (s *Storage) CreateWithItem(ctx context.Context, homeID int64, input Input, item ItemInput) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("iniciar transacción del mantenimiento: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO home_maintenances
			(home_id, performed_at, kind, area, institution_id, description)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		homeID, input.PerformedAt, input.Kind, input.Area,
		nullableID(input.InstitutionID), input.Description,
	)
	if err != nil {
		return 0, fmt.Errorf("crear mantenimiento: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("id de mantenimiento: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO home_maintenance_items
			(maintenance_id, item_type, description, quantity, unit, amount, currency_id, institution_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id, item.ItemType, item.Description, item.Quantity,
		item.Unit, item.Amount, item.CurrencyID, nullableID(item.InstitutionID),
	); err != nil {
		return 0, fmt.Errorf("crear línea del mantenimiento: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("confirmar mantenimiento: %w", err)
	}
	return id, nil
}

// Update actualiza un mantenimiento de la vivienda.
func (s *Storage) Update(ctx context.Context, homeID, id int64, input Input) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE home_maintenances
		SET performed_at = ?, kind = ?, area = ?, institution_id = ?,
		    description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND home_id = ?
	`,
		input.PerformedAt, input.Kind, input.Area, nullableID(input.InstitutionID),
		input.Description, id, homeID,
	)
	if err != nil {
		return false, fmt.Errorf("actualizar mantenimiento: %w", err)
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

// Delete elimina un mantenimiento de la vivienda.
func (s *Storage) Delete(ctx context.Context, homeID, id int64) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM home_maintenances WHERE id = ? AND home_id = ?`, id, homeID)
	if err != nil {
		return false, fmt.Errorf("eliminar mantenimiento: %w", err)
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

const itemColumns = `
	it.id, it.maintenance_id, it.item_type, it.description, it.quantity, it.unit,
	it.amount, it.currency_id, c.code, it.institution_id, COALESCE(i.name, ''),
	it.created_at, it.updated_at
`

const itemFrom = `
	FROM home_maintenance_items it
	JOIN currencies c ON c.id = it.currency_id
	LEFT JOIN institutions i ON i.id = it.institution_id
`

func scanItem(row interface{ Scan(...any) error }) (*Item, error) {
	var item Item
	var institutionID sql.NullInt64
	err := row.Scan(
		&item.ID, &item.MaintenanceID, &item.ItemType, &item.Description,
		&item.Quantity, &item.Unit, &item.Amount,
		&item.CurrencyID, &item.CurrencyCode,
		&institutionID, &item.InstitutionName, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear línea: %w", err)
	}
	if institutionID.Valid {
		item.InstitutionID = &institutionID.Int64
	}
	return &item, nil
}

// ListItems devuelve las líneas de un mantenimiento.
func (s *Storage) ListItems(ctx context.Context, maintenanceID int64) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+itemColumns+itemFrom+`
		WHERE it.maintenance_id = ?
		ORDER BY it.item_type, it.id
	`, maintenanceID)
	if err != nil {
		return nil, fmt.Errorf("listar líneas: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer líneas: %w", err)
	}
	return items, nil
}

// ListItemsByMaintenanceIDs devuelve las líneas de varios mantenimientos en una
// sola consulta, agrupadas por mantenimiento. Corta en limit y avisa si quedó
// truncado para no exceder la memoria del iHost.
func (s *Storage) ListItemsByMaintenanceIDs(ctx context.Context, maintenanceIDs []int64, limit int) (map[int64][]Item, bool, error) {
	grouped := make(map[int64][]Item)
	if len(maintenanceIDs) == 0 {
		return grouped, false, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(maintenanceIDs)), ",")
	args := make([]any, 0, len(maintenanceIDs)+1)
	for _, id := range maintenanceIDs {
		args = append(args, id)
	}
	query := `
		SELECT ` + itemColumns + itemFrom + `
		WHERE it.maintenance_id IN (` + placeholders + `)
		ORDER BY it.maintenance_id, it.item_type, it.id
	`
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit+1)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("listar líneas por mantenimiento: %w", err)
	}
	defer rows.Close()

	count := 0
	truncated := false
	for rows.Next() {
		if limit > 0 && count >= limit {
			truncated = true
			break
		}
		item, err := scanItem(rows)
		if err != nil {
			return nil, false, err
		}
		if item == nil {
			continue
		}
		grouped[item.MaintenanceID] = append(grouped[item.MaintenanceID], *item)
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("recorrer líneas por mantenimiento: %w", err)
	}
	return grouped, truncated, nil
}

// GetItem devuelve una línea del mantenimiento o nil si no existe.
func (s *Storage) GetItem(ctx context.Context, maintenanceID, itemID int64) (*Item, error) {
	return scanItem(s.db.QueryRowContext(ctx, `
		SELECT `+itemColumns+itemFrom+`
		WHERE it.maintenance_id = ? AND it.id = ?
	`, maintenanceID, itemID))
}

// CreateItem inserta una línea.
func (s *Storage) CreateItem(ctx context.Context, maintenanceID int64, input ItemInput) (*Item, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO home_maintenance_items
			(maintenance_id, item_type, description, quantity, unit, amount, currency_id, institution_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		maintenanceID, input.ItemType, input.Description, input.Quantity,
		input.Unit, input.Amount, input.CurrencyID, nullableID(input.InstitutionID),
	)
	if err != nil {
		return nil, fmt.Errorf("crear línea: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("id de línea: %w", err)
	}
	return s.GetItem(ctx, maintenanceID, id)
}

// UpdateItem actualiza una línea del mantenimiento.
func (s *Storage) UpdateItem(ctx context.Context, maintenanceID, itemID int64, input ItemInput) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE home_maintenance_items
		SET item_type = ?, description = ?, quantity = ?, unit = ?, amount = ?,
		    currency_id = ?, institution_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND maintenance_id = ?
	`,
		input.ItemType, input.Description, input.Quantity, input.Unit,
		input.Amount, input.CurrencyID, nullableID(input.InstitutionID), itemID, maintenanceID,
	)
	if err != nil {
		return false, fmt.Errorf("actualizar línea: %w", err)
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

// DeleteItem elimina una línea del mantenimiento.
func (s *Storage) DeleteItem(ctx context.Context, maintenanceID, itemID int64) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM home_maintenance_items WHERE id = ? AND maintenance_id = ?`,
		itemID, maintenanceID)
	if err != nil {
		return false, fmt.Errorf("eliminar línea: %w", err)
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

// HomeExists indica si la vivienda existe y no está eliminada.
func (s *Storage) HomeExists(ctx context.Context, id int64) (bool, error) {
	return s.exists(ctx, `SELECT COUNT(*) FROM homes WHERE id = ? AND deleted_at IS NULL`, id)
}

// CurrencyExists indica si la moneda existe y no está eliminada.
func (s *Storage) CurrencyExists(ctx context.Context, id int64) (bool, error) {
	return s.exists(ctx, `SELECT COUNT(*) FROM currencies WHERE id = ? AND deleted_at IS NULL`, id)
}

// InstitutionExists indica si la institución existe.
func (s *Storage) InstitutionExists(ctx context.Context, id int64) (bool, error) {
	return s.exists(ctx, `SELECT COUNT(*) FROM institutions WHERE id = ?`, id)
}

func (s *Storage) exists(ctx context.Context, query string, id int64) (bool, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&count); err != nil {
		return false, fmt.Errorf("verificar existencia: %w", err)
	}
	return count > 0, nil
}

func nullableID(id *int64) any {
	if id == nil || *id <= 0 {
		return nil
	}
	return *id
}
