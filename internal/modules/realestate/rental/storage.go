package rental

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrPersonNotFound      = errors.New("persona no encontrada")
	ErrLeaseNotFound       = errors.New("contrato no encontrado")
	ErrReservationNotFound = errors.New("apartado no encontrado")
	ErrHomeNotFound        = errors.New("vivienda no encontrada")
	ErrAgencyNotFound      = errors.New("inmobiliaria no encontrada")
	ErrChargeNotFound      = errors.New("cargo no encontrado")
	ErrInvoiceNotFound     = errors.New("factura no encontrada")
	ErrPaymentNotFound     = errors.New("pago no encontrado")
	ErrReceiptNotFound     = errors.New("recibo no encontrado")
	ErrDocumentNotFound    = errors.New("documento no encontrado")
	ErrProofNotFound       = errors.New("comprobante no encontrado")
	ErrReminderNotFound    = errors.New("recordatorio no encontrado")
	ErrHomeUnavailable     = errors.New("la vivienda ya tiene un apartado o contrato vigente")
	ErrLeaseNotDraft       = errors.New("solo se puede editar un contrato en borrador")
	ErrInvoiceImmutable    = errors.New("una factura emitida no se puede editar ni eliminar")
	ErrDocumentLimit       = errors.New("se alcanzó el máximo de documentos del contrato")
	ErrProofLimit          = errors.New("se alcanzó el máximo de comprobantes del pago")
	ErrDuplicateParty      = errors.New("la persona ya tiene ese rol en el contrato")
)

// Storage es el único punto con SQL del módulo (ARCH-MODULES).
type Storage struct {
	db *sql.DB
}

// NewStorage crea el storage del módulo.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// dbtx abstrae *sql.DB y *sql.Tx para compartir helpers dentro de transacciones.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ---------------------------------------------------------------- personas

const personColumns = `
	id, full_name, email, phone, tax_id, address, occupation, employer,
	monthly_income, marital_status, notes, deleted_at, created_at, updated_at
`

func scanPerson(row interface{ Scan(...any) error }) (*Person, error) {
	var p Person
	var deletedAt sql.NullTime
	err := row.Scan(
		&p.ID, &p.FullName, &p.Email, &p.Phone, &p.TaxID, &p.Address,
		&p.Occupation, &p.Employer, &p.MonthlyIncome, &p.MaritalStatus,
		&p.Notes, &deletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear persona: %w", err)
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	return &p, nil
}

// ListPeople busca personas activas por nombre, email o documento.
func (s *Storage) ListPeople(ctx context.Context, query string) ([]Person, error) {
	q := `
		SELECT ` + personColumns + `
		FROM rental_people
		WHERE deleted_at IS NULL
	`
	args := []any{}
	if strings.TrimSpace(query) != "" {
		like := "%" + strings.TrimSpace(query) + "%"
		q += ` AND (full_name LIKE ? COLLATE NOCASE OR email LIKE ? COLLATE NOCASE OR tax_id LIKE ? COLLATE NOCASE)`
		args = append(args, like, like, like)
	}
	q += ` ORDER BY full_name COLLATE NOCASE LIMIT 100`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar personas: %w", err)
	}
	defer rows.Close()

	var out []Person
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// GetPerson devuelve una persona activa o nil.
func (s *Storage) GetPerson(ctx context.Context, id int64) (*Person, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+personColumns+` FROM rental_people WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanPerson(row)
}

// CreatePerson inserta una persona.
func (s *Storage) CreatePerson(ctx context.Context, p *Person) (*Person, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_people
			(full_name, email, phone, tax_id, address, occupation, employer, monthly_income, marital_status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.FullName, p.Email, p.Phone, p.TaxID, p.Address, p.Occupation, p.Employer,
		p.MonthlyIncome, p.MaritalStatus, p.Notes)
	if err != nil {
		return nil, fmt.Errorf("insertar persona: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de persona: %w", err)
	}
	return s.GetPerson(ctx, id)
}

// UpdatePerson actualiza una persona.
func (s *Storage) UpdatePerson(ctx context.Context, p *Person) (*Person, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_people SET
			full_name = ?, email = ?, phone = ?, tax_id = ?, address = ?,
			occupation = ?, employer = ?, monthly_income = ?, marital_status = ?,
			notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, p.FullName, p.Email, p.Phone, p.TaxID, p.Address, p.Occupation, p.Employer,
		p.MonthlyIncome, p.MaritalStatus, p.Notes, p.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar persona: %w", err)
	}
	return s.GetPerson(ctx, p.ID)
}

// DeletePerson hace soft-delete de una persona.
func (s *Storage) DeletePerson(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_people SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar persona: %w", err)
	}
	return nil
}

// PersonExists indica si la persona existe.
func (s *Storage) PersonExists(ctx context.Context, id int64) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rental_people WHERE id = ? AND deleted_at IS NULL`, id).Scan(&n); err != nil {
		return false, fmt.Errorf("verificar persona: %w", err)
	}
	return n > 0, nil
}

// ------------------------------------------------------- viviendas y listing

// HomeExists indica si la vivienda existe.
func (s *Storage) HomeExists(ctx context.Context, id int64) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM homes WHERE id = ? AND deleted_at IS NULL`, id).Scan(&n); err != nil {
		return false, fmt.Errorf("verificar vivienda: %w", err)
	}
	return n > 0, nil
}

// SetListingEnabled activa o desactiva la publicación de la vivienda.
func (s *Storage) SetListingEnabled(ctx context.Context, homeID int64, enabled bool) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE homes SET listing_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, boolInt(enabled), homeID)
	if err != nil {
		return fmt.Errorf("actualizar publicación: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrHomeNotFound
	}
	return nil
}

// GetHomeListing devuelve nombre, dirección, flag y estado derivado.
func (s *Storage) GetHomeListing(ctx context.Context, homeID int64) (*HomeRentalStatus, error) {
	st := &HomeRentalStatus{HomeID: homeID}
	var enabled int
	err := s.db.QueryRowContext(ctx, `
		SELECT name, COALESCE(address, ''), listing_enabled FROM homes
		WHERE id = ? AND deleted_at IS NULL
	`, homeID).Scan(&st.HomeName, &st.HomeAddress, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHomeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener vivienda: %w", err)
	}
	st.ListingEnabled = enabled == 1
	return st, nil
}

// AgencyExists indica si la institución existe.
func (s *Storage) AgencyExists(ctx context.Context, id int64) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM institutions WHERE id = ?`, id).Scan(&n); err != nil {
		return false, fmt.Errorf("verificar inmobiliaria: %w", err)
	}
	return n > 0, nil
}

// AgencyIsRealEstate indica si la institución existe y tiene la categoría
// canónica real_estate (SPEC-140).
func (s *Storage) AgencyIsRealEstate(ctx context.Context, id int64) (bool, bool, error) {
	var key sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT c.key
		FROM institutions i
		LEFT JOIN institution_categories c ON c.id = i.category_id
		WHERE i.id = ?
	`, id).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("verificar categoría de inmobiliaria: %w", err)
	}
	// Los keys se guardan en minúsculas; comparamos normalizado por robustez.
	return true, strings.EqualFold(strings.TrimSpace(key.String), "real_estate"), nil
}

// ---------------------------------------------------------------- contratos

const leaseColumns = `
	l.id, l.home_id, h.name, COALESCE(h.address, ''),
	l.agency_institution_id, COALESCE(i.name, ''),
	l.agent_person_id, COALESCE(ap.full_name, ''),
	l.status, l.start_date, l.end_date, l.term_months,
	l.rent_amount, l.maintenance_amount, l.currency, l.payment_day,
	l.deposit_amount, l.late_fee_type, l.late_fee_value, l.late_fee_cap,
	l.auto_renew, l.notice_days, l.commission_percent, l.commission_base,
	l.owner_bank_account, l.notes, l.deleted_at, l.created_at, l.updated_at,
	(SELECT COUNT(*) FROM rental_charges c WHERE c.lease_id = l.id),
	(SELECT COUNT(*) FROM rental_invoices i2 WHERE i2.lease_id = l.id)
`

const leaseFrom = `
	FROM rental_leases l
	JOIN homes h ON h.id = l.home_id
	LEFT JOIN institutions i ON i.id = l.agency_institution_id
	LEFT JOIN rental_people ap ON ap.id = l.agent_person_id
`

func scanLease(row interface{ Scan(...any) error }) (*Lease, error) {
	var l Lease
	var agencyID, agentID sql.NullInt64
	var deletedAt sql.NullTime
	err := row.Scan(
		&l.ID, &l.HomeID, &l.HomeName, &l.HomeAddress,
		&agencyID, &l.AgencyName, &agentID, &l.AgentName,
		&l.Status, &l.StartDate, &l.EndDate, &l.TermMonths,
		&l.RentAmount, &l.MaintenanceAmount, &l.Currency, &l.PaymentDay,
		&l.DepositAmount, &l.LateFeeType, &l.LateFeeValue, &l.LateFeeCap,
		&l.AutoRenew, &l.NoticeDays, &l.CommissionPercent, &l.CommissionBase,
		&l.OwnerBankAccount, &l.Notes, &deletedAt, &l.CreatedAt, &l.UpdatedAt,
		&l.ChargeCount, &l.InvoiceCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear contrato: %w", err)
	}
	if agencyID.Valid {
		l.AgencyInstitutionID = &agencyID.Int64
	}
	if agentID.Valid {
		l.AgentPersonID = &agentID.Int64
	}
	if deletedAt.Valid {
		l.DeletedAt = &deletedAt.Time
	}
	return &l, nil
}

// ListLeases devuelve contratos filtrados por estado y vivienda.
func (s *Storage) ListLeases(ctx context.Context, status string, homeID int64) ([]Lease, error) {
	q := `SELECT ` + leaseColumns + leaseFrom + ` WHERE l.deleted_at IS NULL`
	args := []any{}
	if status != "" {
		q += ` AND l.status = ?`
		args = append(args, status)
	}
	if homeID > 0 {
		q += ` AND l.home_id = ?`
		args = append(args, homeID)
	}
	q += ` ORDER BY l.start_date DESC, l.id DESC`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar contratos: %w", err)
	}
	defer rows.Close()

	var out []Lease
	for rows.Next() {
		l, err := scanLease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

// GetLease devuelve un contrato o nil.
func (s *Storage) GetLease(ctx context.Context, id int64) (*Lease, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+leaseColumns+leaseFrom+` WHERE l.id = ? AND l.deleted_at IS NULL`, id)
	return scanLease(row)
}

// CreateLease inserta un contrato.
func (s *Storage) CreateLease(ctx context.Context, l *Lease) (*Lease, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_leases (
			home_id, agency_institution_id, agent_person_id, status, start_date, end_date,
			term_months, rent_amount, maintenance_amount, currency, payment_day, deposit_amount,
			late_fee_type, late_fee_value, late_fee_cap, auto_renew, notice_days,
			commission_percent, commission_base, owner_bank_account, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.HomeID, nullableID(l.AgencyInstitutionID), nullableID(l.AgentPersonID), l.Status, l.StartDate, l.EndDate,
		l.TermMonths, l.RentAmount, l.MaintenanceAmount, l.Currency, l.PaymentDay, l.DepositAmount,
		l.LateFeeType, l.LateFeeValue, l.LateFeeCap, boolInt(l.AutoRenew), l.NoticeDays,
		l.CommissionPercent, l.CommissionBase, l.OwnerBankAccount, l.Notes)
	if err != nil {
		return nil, fmt.Errorf("insertar contrato: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de contrato: %w", err)
	}
	return s.GetLease(ctx, id)
}

// UpdateLease actualiza un contrato en borrador.
func (s *Storage) UpdateLease(ctx context.Context, l *Lease) (*Lease, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_leases SET
			agency_institution_id = ?, agent_person_id = ?, start_date = ?, end_date = ?,
			term_months = ?, rent_amount = ?, maintenance_amount = ?, currency = ?, payment_day = ?,
			deposit_amount = ?, late_fee_type = ?, late_fee_value = ?, late_fee_cap = ?,
			auto_renew = ?, notice_days = ?, commission_percent = ?, commission_base = ?,
			owner_bank_account = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, nullableID(l.AgencyInstitutionID), nullableID(l.AgentPersonID), l.StartDate, l.EndDate,
		l.TermMonths, l.RentAmount, l.MaintenanceAmount, l.Currency, l.PaymentDay,
		l.DepositAmount, l.LateFeeType, l.LateFeeValue, l.LateFeeCap,
		boolInt(l.AutoRenew), l.NoticeDays, l.CommissionPercent, l.CommissionBase,
		l.OwnerBankAccount, l.Notes, l.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar contrato: %w", err)
	}
	return s.GetLease(ctx, l.ID)
}

// SetLeaseStatus cambia el estado de un contrato.
func (s *Storage) SetLeaseStatus(ctx context.Context, id int64, status, endDate string) error {
	var err error
	if endDate != "" {
		_, err = s.db.ExecContext(ctx, `
			UPDATE rental_leases SET status = ?, end_date = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND deleted_at IS NULL
		`, status, endDate, id)
	} else {
		_, err = s.db.ExecContext(ctx, `
			UPDATE rental_leases SET status = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND deleted_at IS NULL
		`, status, id)
	}
	if err != nil {
		return fmt.Errorf("cambiar estado del contrato: %w", err)
	}
	return nil
}

// SoftDeleteLease elimina lógicamente un contrato en borrador.
func (s *Storage) SoftDeleteLease(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_leases SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar contrato: %w", err)
	}
	return nil
}

// ActiveLeaseForHome devuelve el contrato activo de una vivienda.
func (s *Storage) ActiveLeaseForHome(ctx context.Context, homeID int64) (*Lease, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+leaseColumns+leaseFrom+`
		WHERE l.home_id = ? AND l.status = 'active' AND l.deleted_at IS NULL
		ORDER BY l.start_date DESC LIMIT 1`, homeID)
	return scanLease(row)
}

// LeaseOverlap indica si hay un contrato activo que solape el rango.
func (s *Storage) LeaseOverlap(ctx context.Context, homeID int64, start, end string, excludeID int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM rental_leases
		WHERE home_id = ? AND status = 'active' AND deleted_at IS NULL
			AND id <> ? AND start_date <= ? AND end_date >= ?
	`, homeID, excludeID, end, start).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("verificar solape de contratos: %w", err)
	}
	return n > 0, nil
}

// ---------------------------------------------------------- partes del contrato

// ListParties devuelve las partes con sus datos de persona.
func (s *Storage) ListParties(ctx context.Context, leaseID int64) ([]LeaseParty, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.lease_id, p.person_id, p.role, p.notify_email,
			pp.id, pp.full_name, pp.email, pp.phone, pp.tax_id, pp.address,
			pp.occupation, pp.employer, pp.monthly_income, pp.marital_status,
			pp.notes, pp.deleted_at, pp.created_at, pp.updated_at
		FROM rental_lease_parties p
		JOIN rental_people pp ON pp.id = p.person_id
		WHERE p.lease_id = ?
		ORDER BY CASE p.role
			WHEN 'tenant' THEN 1 WHEN 'guarantor' THEN 2 WHEN 'owner' THEN 3
			WHEN 'agent' THEN 4 ELSE 5 END, pp.full_name COLLATE NOCASE
	`, leaseID)
	if err != nil {
		return nil, fmt.Errorf("listar partes: %w", err)
	}
	defer rows.Close()

	var out []LeaseParty
	for rows.Next() {
		var p LeaseParty
		var person Person
		var deletedAt sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.LeaseID, &p.PersonID, &p.Role, &p.NotifyEmail,
			&person.ID, &person.FullName, &person.Email, &person.Phone, &person.TaxID,
			&person.Address, &person.Occupation, &person.Employer, &person.MonthlyIncome,
			&person.MaritalStatus, &person.Notes, &deletedAt, &person.CreatedAt, &person.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("escanear parte: %w", err)
		}
		if deletedAt.Valid {
			person.DeletedAt = &deletedAt.Time
		}
		p.Person = &person
		out = append(out, p)
	}
	return out, rows.Err()
}

// AddParty vincula una persona con el contrato.
func (s *Storage) AddParty(ctx context.Context, leaseID, personID int64, role string, notify bool) (*LeaseParty, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_lease_parties (lease_id, person_id, role, notify_email)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(lease_id, person_id, role) DO UPDATE SET notify_email = excluded.notify_email
	`, leaseID, personID, role, boolInt(notify))
	if err != nil {
		return nil, fmt.Errorf("agregar parte: %w", err)
	}
	id, _ := res.LastInsertId()
	return &LeaseParty{ID: id, LeaseID: leaseID, PersonID: personID, Role: role, NotifyEmail: notify}, nil
}

// DeleteParty quita una parte del contrato.
func (s *Storage) DeleteParty(ctx context.Context, leaseID, partyID int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM rental_lease_parties WHERE id = ? AND lease_id = ?
	`, partyID, leaseID)
	if err != nil {
		return fmt.Errorf("quitar parte: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrPersonNotFound
	}
	return nil
}

// NotifyParties devuelve personas con notify_email del contrato filtrando roles.
func (s *Storage) NotifyParties(ctx context.Context, leaseID int64, roles []string) ([]Person, error) {
	q := `
		SELECT DISTINCT pp.id, pp.full_name, pp.email, pp.phone, pp.tax_id, pp.address,
			pp.occupation, pp.employer, pp.monthly_income, pp.marital_status,
			pp.notes, pp.deleted_at, pp.created_at, pp.updated_at
		FROM rental_lease_parties p
		JOIN rental_people pp ON pp.id = p.person_id
		WHERE p.lease_id = ? AND p.notify_email = 1 AND pp.email <> '' AND pp.deleted_at IS NULL
	`
	args := []any{leaseID}
	if len(roles) > 0 {
		q += ` AND p.role IN (` + placeholders(len(roles)) + `)`
		for _, r := range roles {
			args = append(args, r)
		}
	}
	q += ` ORDER BY pp.full_name`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar destinatarios: %w", err)
	}
	defer rows.Close()

	var out []Person
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------- apartados

const reservationColumns = `
	r.id, r.home_id, h.name, COALESCE(h.address, ''),
	r.applicant_person_id, r.agent_person_id, COALESCE(ag.full_name, ''),
	r.converted_lease_id, r.status, r.amount, r.currency, r.reference,
	r.applied_to_deposit, r.starts_at, r.expires_at, r.contract_date, r.move_in_date,
	r.rent_amount, r.maintenance_amount, r.owner_split_percent, r.agency_split_percent,
	r.refund_policy, r.required_documents, r.notes, r.created_at, r.updated_at
`

const reservationFrom = `
	FROM rental_reservations r
	JOIN homes h ON h.id = r.home_id
	JOIN rental_people ap ON ap.id = r.applicant_person_id
	LEFT JOIN rental_people ag ON ag.id = r.agent_person_id
`

func scanReservation(row interface{ Scan(...any) error }) (*Reservation, error) {
	var r Reservation
	var agentID, leaseID sql.NullInt64
	err := row.Scan(
		&r.ID, &r.HomeID, &r.HomeName, &r.HomeAddress,
		&r.ApplicantPersonID, &agentID, &r.AgentName,
		&leaseID, &r.Status, &r.Amount, &r.Currency, &r.Reference,
		&r.AppliedToDeposit, &r.StartsAt, &r.ExpiresAt, &r.ContractDate, &r.MoveInDate,
		&r.RentAmount, &r.MaintenanceAmount, &r.OwnerSplitPercent, &r.AgencySplitPercent,
		&r.RefundPolicy, &r.RequiredDocuments, &r.Notes, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear apartado: %w", err)
	}
	if agentID.Valid {
		r.AgentPersonID = &agentID.Int64
	}
	if leaseID.Valid {
		r.ConvertedLeaseID = &leaseID.Int64
	}
	return &r, nil
}

// ListReservations lista apartados filtrados.
func (s *Storage) ListReservations(ctx context.Context, homeID int64, status string) ([]Reservation, error) {
	q := `SELECT ` + reservationColumns + reservationFrom + ` WHERE 1=1`
	args := []any{}
	if homeID > 0 {
		q += ` AND r.home_id = ?`
		args = append(args, homeID)
	}
	if status != "" {
		q += ` AND r.status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY r.expires_at DESC, r.id DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar apartados: %w", err)
	}
	defer rows.Close()

	var out []Reservation
	for rows.Next() {
		r, err := scanReservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// GetReservation devuelve un apartado o nil.
func (s *Storage) GetReservation(ctx context.Context, id int64) (*Reservation, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+reservationColumns+reservationFrom+` WHERE r.id = ?`, id)
	return scanReservation(row)
}

// CreateReservation inserta un apartado.
func (s *Storage) CreateReservation(ctx context.Context, r *Reservation) (*Reservation, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_reservations (
			home_id, applicant_person_id, agent_person_id, status, amount, currency, reference,
			applied_to_deposit, starts_at, expires_at, contract_date, move_in_date,
			rent_amount, maintenance_amount, owner_split_percent, agency_split_percent,
			refund_policy, required_documents, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.HomeID, r.ApplicantPersonID, nullableID(r.AgentPersonID), r.Status, r.Amount, r.Currency, r.Reference,
		boolInt(r.AppliedToDeposit), r.StartsAt, r.ExpiresAt, r.ContractDate, r.MoveInDate,
		r.RentAmount, r.MaintenanceAmount, r.OwnerSplitPercent, r.AgencySplitPercent,
		r.RefundPolicy, r.RequiredDocuments, r.Notes)
	if err != nil {
		return nil, fmt.Errorf("insertar apartado: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de apartado: %w", err)
	}
	return s.GetReservation(ctx, id)
}

// UpdateReservation actualiza un apartado activo.
func (s *Storage) UpdateReservation(ctx context.Context, r *Reservation) (*Reservation, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_reservations SET
			agent_person_id = ?, amount = ?, currency = ?, reference = ?, applied_to_deposit = ?,
			starts_at = ?, expires_at = ?, contract_date = ?, move_in_date = ?,
			rent_amount = ?, maintenance_amount = ?, owner_split_percent = ?, agency_split_percent = ?,
			refund_policy = ?, required_documents = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'active'
	`, nullableID(r.AgentPersonID), r.Amount, r.Currency, r.Reference, boolInt(r.AppliedToDeposit),
		r.StartsAt, r.ExpiresAt, r.ContractDate, r.MoveInDate, r.RentAmount, r.MaintenanceAmount,
		r.OwnerSplitPercent, r.AgencySplitPercent, r.RefundPolicy, r.RequiredDocuments, r.Notes, r.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar apartado: %w", err)
	}
	return s.GetReservation(ctx, r.ID)
}

// SetReservationStatus cambia el estado del apartado.
func (s *Storage) SetReservationStatus(ctx context.Context, id int64, status string, convertedLeaseID *int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_reservations
		SET status = ?, converted_lease_id = COALESCE(?, converted_lease_id), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, convertedLeaseID, id)
	if err != nil {
		return fmt.Errorf("cambiar estado del apartado: %w", err)
	}
	return nil
}

// ActiveReservationForHome devuelve el apartado activo vigente.
func (s *Storage) ActiveReservationForHome(ctx context.Context, homeID int64, now time.Time) (*Reservation, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+reservationColumns+reservationFrom+`
		WHERE r.home_id = ? AND r.status = 'active' AND r.expires_at > ?
		ORDER BY r.expires_at DESC LIMIT 1`, homeID, now.UTC().Format(time.RFC3339))
	return scanReservation(row)
}

// ReservationOverlap indica si hay otro apartado activo que solape.
func (s *Storage) ReservationOverlap(ctx context.Context, homeID int64, start, end string, excludeID int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM rental_reservations
		WHERE home_id = ? AND status = 'active' AND id <> ?
			AND starts_at < ? AND expires_at > ?
	`, homeID, excludeID, end, start).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("verificar solape de apartados: %w", err)
	}
	return n > 0, nil
}

// ExpireReservations marca como expirados los apartados vencidos y devuelve los IDs.
func (s *Storage) ExpireReservations(ctx context.Context, now time.Time) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM rental_reservations WHERE status = 'active' AND expires_at <= ?
	`, now.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("buscar apartados vencidos: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if err := s.SetReservationStatus(ctx, id, ReservationExpired, nil); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func nullableID(id *int64) any {
	if id == nil {
		return nil
	}
	return *id
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
