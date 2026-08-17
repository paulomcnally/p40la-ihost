package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// BillStorage encapsula el acceso a la tabla bills.
type BillStorage struct {
	db *sql.DB
}

// NewBillStorage crea un nuevo BillStorage.
func NewBillStorage(db *sql.DB) *BillStorage {
	return &BillStorage{db: db}
}

// ListByService devuelve las facturas no eliminadas de un servicio.
func (s *BillStorage) ListByService(ctx context.Context, serviceID int64) ([]models.Bill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, service_id, year, month, amount, invoice_number, status, drive_url,
		       file_hash, deleted_at, created_at, updated_at
		FROM bills
		WHERE service_id = ? AND deleted_at IS NULL
		ORDER BY year DESC, month DESC
	`, serviceID)
	if err != nil {
		return nil, fmt.Errorf("listar facturas: %w", err)
	}
	defer rows.Close()

	return scanBills(rows)
}

// GetByID busca una factura por su ID.
func (s *BillStorage) GetByID(ctx context.Context, id int64) (*models.Bill, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, service_id, year, month, amount, invoice_number, status, drive_url,
		       file_hash, deleted_at, created_at, updated_at
		FROM bills
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanBill(row)
}

// FindByServicePeriod busca una factura existente por servicio, año y mes.
func (s *BillStorage) FindByServicePeriod(ctx context.Context, serviceID int64, year, month int) (*models.Bill, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, service_id, year, month, amount, invoice_number, status, drive_url,
		       file_hash, deleted_at, created_at, updated_at
		FROM bills
		WHERE service_id = ? AND year = ? AND month = ? AND deleted_at IS NULL
	`, serviceID, year, month)
	return scanBill(row)
}

// FindByServiceFileHash busca una factura que ya fue importada con ese hash
// de archivo para el servicio (dedup de subidas, SPEC-041).
func (s *BillStorage) FindByServiceFileHash(ctx context.Context, serviceID int64, fileHash string) (*models.Bill, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, service_id, year, month, amount, invoice_number, status, drive_url,
		       file_hash, deleted_at, created_at, updated_at
		FROM bills
		WHERE service_id = ? AND file_hash = ? AND deleted_at IS NULL
		LIMIT 1
	`, serviceID, fileHash)
	return scanBill(row)
}

// Create inserta una nueva factura.
func (s *BillStorage) Create(ctx context.Context, bill *models.Bill) (*models.Bill, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO bills (service_id, year, month, amount, invoice_number, status, drive_url, file_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, bill.ServiceID, bill.Year, bill.Month, bill.Amount, bill.InvoiceNumber, bill.Status, bill.DriveURL, bill.FileHash)
	if err != nil {
		return nil, fmt.Errorf("insertar factura: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de factura: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una factura existente.
func (s *BillStorage) Update(ctx context.Context, bill *models.Bill) (*models.Bill, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE bills
		SET year = ?, month = ?, amount = ?, invoice_number = ?, status = ?, drive_url = ?, file_hash = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, bill.Year, bill.Month, bill.Amount, bill.InvoiceNumber, bill.Status, bill.DriveURL, bill.FileHash, bill.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar factura: %w", err)
	}
	return s.GetByID(ctx, bill.ID)
}

// UpdateFromExtracted actualiza solo los campos provistos por el analizador.
func (s *BillStorage) UpdateFromExtracted(ctx context.Context, billID int64, amount float64, invoiceNumber, fileHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE bills
		SET amount = ?, invoice_number = ?, file_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, amount, invoiceNumber, fileHash, billID)
	if err != nil {
		return fmt.Errorf("actualizar factura desde analizador: %w", err)
	}
	return nil
}

// SoftDelete marca una factura como eliminada.
func (s *BillStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE bills
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar factura: %w", err)
	}
	return nil
}

// ListPendingWithDetails devuelve las facturas pendientes con contexto de
// casa, institución, servicio y moneda, excluyendo servicios eliminados (SPEC-031).
func (s *BillStorage) ListPendingWithDetails(ctx context.Context) ([]models.PendingBillDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.service_id, b.year, b.month, b.amount, b.status, b.created_at,
		       s.home_id, h.name, COALESCE(i.name, s.institution, ''), s.name, COALESCE(c.symbol, '')
		FROM bills b
		JOIN services s ON s.id = b.service_id AND s.deleted_at IS NULL
		JOIN homes h ON h.id = s.home_id
		LEFT JOIN institutions i ON i.id = s.institution_id
		LEFT JOIN currencies c ON c.id = s.currency_id
		WHERE b.status = 'pending' AND b.deleted_at IS NULL
		ORDER BY b.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar facturas pendientes: %w", err)
	}
	defer rows.Close()

	var pending []models.PendingBillDetail
	for rows.Next() {
		var d models.PendingBillDetail
		if err := rows.Scan(&d.BillID, &d.ServiceID, &d.Year, &d.Month, &d.Amount, &d.Status,
			&d.CreatedAt, &d.HomeID, &d.HomeName, &d.Institution, &d.ServiceName, &d.CurrencySymbol); err != nil {
			return nil, fmt.Errorf("escanear factura pendiente: %w", err)
		}
		pending = append(pending, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar facturas pendientes: %w", err)
	}
	return pending, nil
}

func scanBill(row *sql.Row) (*models.Bill, error) {
	var b models.Bill
	var deletedAt sql.NullTime
	if err := row.Scan(&b.ID, &b.ServiceID, &b.Year, &b.Month, &b.Amount, &b.InvoiceNumber,
		&b.Status, &b.DriveURL, &b.FileHash, &deletedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear factura: %w", err)
	}
	if deletedAt.Valid {
		b.DeletedAt = &deletedAt.Time
	}
	return &b, nil
}

func scanBills(rows *sql.Rows) ([]models.Bill, error) {
	var bills []models.Bill
	for rows.Next() {
		var b models.Bill
		var deletedAt sql.NullTime
		if err := rows.Scan(&b.ID, &b.ServiceID, &b.Year, &b.Month, &b.Amount, &b.InvoiceNumber,
			&b.Status, &b.DriveURL, &b.FileHash, &deletedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear factura: %w", err)
		}
		if deletedAt.Valid {
			b.DeletedAt = &deletedAt.Time
		}
		bills = append(bills, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bills, nil
}
