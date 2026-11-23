package rental

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// ValidationError describe un error de validación con código estable.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(code, message string) error {
	return &ValidationError{Code: code, Message: message}
}

// Service contiene la lógica de negocio del módulo (ARCH-MODULES).
type Service struct {
	storage  *Storage
	email    *services.EmailService
	settings *services.SystemSettingsService
	dataDir  string
	now      func() time.Time
}

// NewService crea el servicio del módulo.
func NewService(st *Storage, email *services.EmailService, settings *services.SystemSettingsService, dataDir string) *Service {
	return &Service{storage: st, email: email, settings: settings, dataDir: dataDir, now: time.Now}
}

// Now expone el reloj para tests.
func (s *Service) Now() time.Time { return s.now() }

// ---------------------------------------------------------------- personas

// ListPeople busca personas para los pickers.
func (s *Service) ListPeople(ctx context.Context, query string) ([]Person, error) {
	return s.storage.ListPeople(ctx, query)
}

// GetPerson devuelve una persona.
func (s *Service) GetPerson(ctx context.Context, id int64) (*Person, error) {
	p, err := s.storage.GetPerson(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPersonNotFound
	}
	return p, nil
}

// CreatePerson valida e inserta una persona.
func (s *Service) CreatePerson(ctx context.Context, p *Person) (*Person, error) {
	if err := preparePerson(p); err != nil {
		return nil, err
	}
	return s.storage.CreatePerson(ctx, p)
}

// UpdatePerson valida y actualiza una persona.
func (s *Service) UpdatePerson(ctx context.Context, p *Person) (*Person, error) {
	if err := preparePerson(p); err != nil {
		return nil, err
	}
	return s.storage.UpdatePerson(ctx, p)
}

// DeletePerson hace soft-delete.
func (s *Service) DeletePerson(ctx context.Context, id int64) error {
	return s.storage.DeletePerson(ctx, id)
}

// ---------------------------------------------------------------- contratos

// ListLeases lista contratos con derivados.
func (s *Service) ListLeases(ctx context.Context, status string, homeID int64) ([]Lease, error) {
	leases, err := s.storage.ListLeases(ctx, status, homeID)
	if err != nil {
		return nil, err
	}
	for i := range leases {
		s.withDerived(ctx, &leases[i])
	}
	return leases, nil
}

// GetLease devuelve un contrato con partes y saldo.
func (s *Service) GetLease(ctx context.Context, id int64) (*Lease, error) {
	l, err := s.storage.GetLease(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLeaseNotFound
	}
	parties, err := s.storage.ListParties(ctx, id)
	if err != nil {
		return nil, err
	}
	l.Parties = parties
	s.withDerived(ctx, l)
	return l, nil
}

// CreateLease valida e inserta un contrato en borrador.
func (s *Service) CreateLease(ctx context.Context, l *Lease) (*Lease, error) {
	l.Status = LeaseDraft
	l.ID = 0
	if err := s.prepareLease(ctx, l); err != nil {
		return nil, err
	}
	created, err := s.storage.CreateLease(ctx, l)
	if err != nil {
		return nil, err
	}
	for _, party := range l.Parties {
		if err := s.saveParty(ctx, created.ID, party); err != nil {
			return nil, err
		}
	}
	return s.GetLease(ctx, created.ID)
}

// UpdateLease actualiza un contrato en borrador.
func (s *Service) UpdateLease(ctx context.Context, l *Lease) (*Lease, error) {
	existing, err := s.storage.GetLease(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrLeaseNotFound
	}
	if existing.Status != LeaseDraft {
		return nil, ErrLeaseNotDraft
	}
	l.HomeID = existing.HomeID
	l.Status = existing.Status
	if err := s.prepareLease(ctx, l); err != nil {
		return nil, err
	}
	if _, err := s.storage.UpdateLease(ctx, l); err != nil {
		return nil, err
	}
	for _, party := range l.Parties {
		if err := s.saveParty(ctx, l.ID, party); err != nil {
			return nil, err
		}
	}
	return s.GetLease(ctx, l.ID)
}

// ActivateLease valida, genera cargos y activa el contrato.
func (s *Service) ActivateLease(ctx context.Context, id int64) (*Lease, error) {
	l, err := s.storage.GetLease(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLeaseNotFound
	}
	if l.Status == LeaseActive {
		return s.GetLease(ctx, id)
	}
	if l.Status != LeaseDraft {
		return nil, invalid("invalid_status", "Solo un contrato en borrador puede activarse")
	}
	if err := s.prepareLease(ctx, l); err != nil {
		return nil, err
	}
	parties, err := s.storage.ListParties(ctx, id)
	if err != nil {
		return nil, err
	}
	hasTenant := false
	for _, p := range parties {
		if p.Role == RoleTenant && p.Person != nil && strings.TrimSpace(p.Person.Email) != "" {
			hasTenant = true
			break
		}
	}
	if !hasTenant {
		return nil, invalid("tenant_required", "El contrato necesita al menos un inquilino con email")
	}
	overlap, err := s.storage.LeaseOverlap(ctx, l.HomeID, l.StartDate, l.EndDate, l.ID)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, ErrHomeUnavailable
	}
	if err := s.storage.SetLeaseStatus(ctx, id, LeaseActive, ""); err != nil {
		return nil, err
	}
	if _, err := s.GenerateCharges(ctx, id); err != nil {
		return nil, err
	}
	return s.GetLease(ctx, id)
}

// TerminateLease termina un contrato activo.
func (s *Service) TerminateLease(ctx context.Context, id int64, reason, endDate string) (*Lease, error) {
	l, err := s.storage.GetLease(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLeaseNotFound
	}
	if l.Status != LeaseActive {
		return nil, invalid("invalid_status", "Solo un contrato activo puede terminarse")
	}
	if endDate == "" {
		endDate = s.now().UTC().Format("2006-01-02")
	}
	if !validDate(endDate) {
		return nil, invalid("invalid_date", "La fecha de fin es inválida")
	}
	if reason != "" {
		l.Notes = strings.TrimSpace(l.Notes + "\nTerminado: " + reason)
		if _, err := s.storage.UpdateLease(ctx, l); err != nil {
			return nil, err
		}
	}
	if err := s.storage.SetLeaseStatus(ctx, id, LeaseTerminated, endDate); err != nil {
		return nil, err
	}
	return s.GetLease(ctx, id)
}

// DeleteLease elimina un contrato en borrador.
func (s *Service) DeleteLease(ctx context.Context, id int64) error {
	l, err := s.storage.GetLease(ctx, id)
	if err != nil {
		return err
	}
	if l == nil {
		return ErrLeaseNotFound
	}
	if l.Status != LeaseDraft {
		return ErrLeaseNotDraft
	}
	return s.storage.SoftDeleteLease(ctx, id)
}

// AddParty agrega una parte al contrato (crea la persona si viene sin ID).
func (s *Service) AddParty(ctx context.Context, leaseID int64, party *LeaseParty) (*LeaseParty, error) {
	l, err := s.storage.GetLease(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLeaseNotFound
	}
	if l.Status != LeaseDraft && l.Status != LeaseActive {
		return nil, invalid("invalid_status", "No se pueden agregar partes a este contrato")
	}
	if err := s.saveParty(ctx, leaseID, *party); err != nil {
		return nil, err
	}
	parties, err := s.storage.ListParties(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	for _, p := range parties {
		if p.PersonID == party.PersonID && p.Role == party.Role {
			return &p, nil
		}
	}
	return party, nil
}

// DeleteParty quita una parte.
func (s *Service) DeleteParty(ctx context.Context, leaseID, partyID int64) error {
	return s.storage.DeleteParty(ctx, leaseID, partyID)
}

func (s *Service) saveParty(ctx context.Context, leaseID int64, party LeaseParty) error {
	if party.Role == "" {
		return invalid("role_required", "El rol de la parte es obligatorio")
	}
	if !contains(PartyRoles, party.Role) {
		return invalid("invalid_role", "Rol de parte inválido")
	}
	personID := party.PersonID
	if personID == 0 && party.Person != nil {
		if err := preparePerson(party.Person); err != nil {
			return err
		}
		created, err := s.storage.CreatePerson(ctx, party.Person)
		if err != nil {
			return err
		}
		personID = created.ID
	}
	if personID == 0 {
		return invalid("person_required", "La persona es obligatoria")
	}
	exists, err := s.storage.PersonExists(ctx, personID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrPersonNotFound
	}
	_, err = s.storage.AddParty(ctx, leaseID, personID, party.Role, party.NotifyEmail)
	return err
}

// ---------------------------------------------------------------- apartados

// ListReservations lista apartados.
func (s *Service) ListReservations(ctx context.Context, homeID int64, status string) ([]Reservation, error) {
	items, err := s.storage.ListReservations(ctx, homeID, status)
	if err != nil {
		return nil, err
	}
	for i := range items {
		s.withApplicant(ctx, &items[i])
	}
	return items, nil
}

// GetReservation devuelve un apartado con el solicitante.
func (s *Service) GetReservation(ctx context.Context, id int64) (*Reservation, error) {
	r, err := s.storage.GetReservation(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrReservationNotFound
	}
	s.withApplicant(ctx, r)
	return r, nil
}

// CreateReservation valida la disponibilidad e inserta el apartado.
func (s *Service) CreateReservation(ctx context.Context, r *Reservation) (*Reservation, error) {
	r.ID = 0
	r.Status = ReservationActive
	if err := s.prepareReservation(ctx, r, true); err != nil {
		return nil, err
	}
	overlap, err := s.storage.ReservationOverlap(ctx, r.HomeID, r.StartsAt, r.ExpiresAt, 0)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, ErrHomeUnavailable
	}
	active, err := s.storage.ActiveLeaseForHome(ctx, r.HomeID)
	if err != nil {
		return nil, err
	}
	if active != nil && active.Status == LeaseActive {
		return nil, ErrHomeUnavailable
	}
	created, err := s.storage.CreateReservation(ctx, r)
	if err != nil {
		return nil, err
	}
	s.withApplicant(ctx, created)
	return created, nil
}

// UpdateReservation actualiza un apartado activo.
func (s *Service) UpdateReservation(ctx context.Context, r *Reservation) (*Reservation, error) {
	existing, err := s.storage.GetReservation(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrReservationNotFound
	}
	if existing.Status != ReservationActive {
		return nil, invalid("invalid_status", "Solo se puede editar un apartado activo")
	}
	r.HomeID = existing.HomeID
	r.ApplicantPersonID = existing.ApplicantPersonID
	r.Status = existing.Status
	if err := s.prepareReservation(ctx, r, false); err != nil {
		return nil, err
	}
	updated, err := s.storage.UpdateReservation(ctx, r)
	if err != nil {
		return nil, err
	}
	s.withApplicant(ctx, updated)
	return updated, nil
}

// ConvertReservation crea el contrato en borrador y marca el apartado.
func (s *Service) ConvertReservation(ctx context.Context, id int64, req ConvertReservationInput) (*Lease, error) {
	r, err := s.storage.GetReservation(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrReservationNotFound
	}
	if r.Status != ReservationActive {
		return nil, invalid("invalid_status", "Solo un apartado activo puede convertirse en contrato")
	}
	term := req.TermMonths
	if term == 0 {
		term = 12
	}
	paymentDay := req.PaymentDay
	if paymentDay == 0 {
		paymentDay = 1
	}
	start := r.MoveInDate
	if start == "" {
		start = r.ContractDate
	}
	if start == "" {
		start = s.now().UTC().Format("2006-01-02")
	}
	deposit := req.DepositAmount
	if deposit == 0 && r.AppliedToDeposit {
		deposit = r.Amount
	}
	lease := &Lease{
		HomeID:            r.HomeID,
		AgentPersonID:     r.AgentPersonID,
		Status:            LeaseDraft,
		StartDate:         start,
		TermMonths:        term,
		RentAmount:        r.RentAmount,
		MaintenanceAmount: r.MaintenanceAmount,
		Currency:          r.Currency,
		PaymentDay:        paymentDay,
		DepositAmount:     deposit,
		LateFeeType:       "none",
		NoticeDays:        30,
		CommissionPercent: req.CommissionPercent,
		CommissionBase:    "collected",
		OwnerBankAccount:  req.OwnerBankAccount,
		Notes:             "Creado desde el apartado #" + fmt.Sprintf("%d", r.ID),
	}
	created, err := s.CreateLease(ctx, lease)
	if err != nil {
		return nil, err
	}
	personID := r.ApplicantPersonID
	if _, err := s.storage.AddParty(ctx, created.ID, personID, RoleTenant, true); err != nil {
		return nil, err
	}
	leaseID := created.ID
	if err := s.storage.SetReservationStatus(ctx, r.ID, ReservationConverted, &leaseID); err != nil {
		return nil, err
	}
	return s.GetLease(ctx, created.ID)
}

// CancelReservation cancela el apartado.
func (s *Service) CancelReservation(ctx context.Context, id int64, reason string) error {
	r, err := s.storage.GetReservation(ctx, id)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrReservationNotFound
	}
	if r.Status != ReservationActive {
		return invalid("invalid_status", "Solo un apartado activo puede cancelarse")
	}
	if reason != "" {
		r.Notes = strings.TrimSpace(r.Notes + "\nCancelado: " + reason)
		if _, err := s.storage.UpdateReservation(ctx, r); err != nil {
			return err
		}
	}
	return s.storage.SetReservationStatus(ctx, id, ReservationCancelled, nil)
}

// ForfeitReservation retiene la seña por desistimiento del solicitante.
func (s *Service) ForfeitReservation(ctx context.Context, id int64, reason string) error {
	r, err := s.storage.GetReservation(ctx, id)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrReservationNotFound
	}
	if r.Status != ReservationActive {
		return invalid("invalid_status", "Solo un apartado activo puede retenerse")
	}
	if r.RefundPolicy == RefundRefundable {
		return invalid("refund_policy", "La política del apartado es reembolsable; usá cancelar")
	}
	if reason != "" {
		r.Notes = strings.TrimSpace(r.Notes + "\nRetenido: " + reason)
		if _, err := s.storage.UpdateReservation(ctx, r); err != nil {
			return err
		}
	}
	return s.storage.SetReservationStatus(ctx, id, ReservationForfeited, nil)
}

// ExpireReservations marca los vencidos (lo llama el scheduler).
func (s *Service) ExpireReservations(ctx context.Context) (int, error) {
	ids, err := s.storage.ExpireReservations(ctx, s.now())
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// ------------------------------------------------------------ estado casa

// GetHomeStatus devuelve el estado de promoción derivado y lo próximo.
func (s *Service) GetHomeStatus(ctx context.Context, homeID int64) (*HomeRentalStatus, error) {
	st, err := s.storage.GetHomeListing(ctx, homeID)
	if err != nil {
		return nil, err
	}
	lease, err := s.storage.ActiveLeaseForHome(ctx, homeID)
	if err != nil {
		return nil, err
	}
	if lease != nil {
		s.withDerived(ctx, lease)
		parties, err := s.storage.ListParties(ctx, lease.ID)
		if err != nil {
			return nil, err
		}
		lease.Parties = parties
		st.ActiveLease = lease
	}
	res, err := s.storage.ActiveReservationForHome(ctx, homeID, s.now())
	if err != nil {
		return nil, err
	}
	if res != nil {
		s.withApplicant(ctx, res)
		st.ActiveReservation = res
	}
	switch {
	case !st.ListingEnabled:
		st.ListingStatus = ListingUnlisted
	case lease != nil && lease.Status == LeaseActive:
		st.ListingStatus = ListingRented
	case res != nil:
		st.ListingStatus = ListingReserved
	default:
		st.ListingStatus = ListingAvailable
	}
	if lease != nil {
		charges, err := s.storage.ListChargesByStatus(ctx, lease.ID, ChargePending)
		if err != nil {
			return nil, err
		}
		if len(charges) > 5 {
			charges = charges[:5]
		}
		st.UpcomingCharges = charges
		if len(charges) > 0 {
			st.NextDueDate = charges[0].DueDate
		}
	}
	reminders, err := s.storage.RemindersForHome(ctx, homeID)
	if err != nil {
		return nil, err
	}
	if len(reminders) > 5 {
		reminders = reminders[:5]
	}
	st.Reminders = reminders
	return st, nil
}

// SetListingEnabled controla la publicación de la vivienda.
func (s *Service) SetListingEnabled(ctx context.Context, homeID int64, enabled bool) (*HomeRentalStatus, error) {
	if err := s.storage.SetListingEnabled(ctx, homeID, enabled); err != nil {
		return nil, err
	}
	return s.GetHomeStatus(ctx, homeID)
}

// ---------------------------------------------------------------- settings

// GetSettings devuelve la configuración del módulo.
func (s *Service) GetSettings(ctx context.Context) (*RentalSettings, error) {
	get := func(key, def string) string {
		v, err := s.settings.GetSetting(ctx, key)
		if err != nil || v == nil || strings.TrimSpace(v.Value) == "" {
			return def
		}
		return v.Value
	}
	lead := DefaultLeadDays
	fmt.Sscanf(get(SettingDefaultLeadDays, "3"), "%d", &lead)
	autoInvoice := get(SettingAutoInvoice, "true") != "false"
	autoSendInvoice := get(SettingAutoSendInvoice, "true") != "false"
	owner := DefaultOwnerSplit
	fmt.Sscanf(get(SettingDefaultOwnerSplit, "50"), "%f", &owner)
	agency := DefaultAgencySplit
	fmt.Sscanf(get(SettingDefaultAgencySplit, "50"), "%f", &agency)
	return &RentalSettings{
		IssuerName:         get(SettingIssuerName, ""),
		IssuerTaxID:        get(SettingIssuerTaxID, ""),
		IssuerAddress:      get(SettingIssuerAddress, ""),
		IssuerEmail:        get(SettingIssuerEmail, ""),
		InvoiceSeries:      get(SettingInvoiceSeries, DefaultInvoiceSeries),
		ReceiptSeries:      get(SettingReceiptSeries, DefaultReceiptSeries),
		DefaultCurrency:    get(SettingDefaultCurrency, DefaultCurrency),
		DefaultLeadDays:    lead,
		DefaultOwnerSplit:  owner,
		DefaultAgencySplit: agency,
		AutoInvoice:        autoInvoice,
		AutoSendInvoice:    autoSendInvoice,
	}, nil
}

// UpdateSettings persiste la configuración del módulo.
func (s *Service) UpdateSettings(ctx context.Context, in *RentalSettings) (*RentalSettings, error) {
	in.IssuerName = strings.TrimSpace(in.IssuerName)
	in.InvoiceSeries = strings.ToUpper(strings.TrimSpace(in.InvoiceSeries))
	in.ReceiptSeries = strings.ToUpper(strings.TrimSpace(in.ReceiptSeries))
	in.DefaultCurrency = strings.ToUpper(strings.TrimSpace(in.DefaultCurrency))
	if in.InvoiceSeries == "" {
		in.InvoiceSeries = DefaultInvoiceSeries
	}
	if in.ReceiptSeries == "" {
		in.ReceiptSeries = DefaultReceiptSeries
	}
	if in.DefaultCurrency == "" {
		in.DefaultCurrency = DefaultCurrency
	}
	if in.DefaultLeadDays < 0 {
		return nil, invalid("invalid_lead_days", "Los días de anticipación no pueden ser negativos")
	}
	if in.DefaultOwnerSplit < 0 || in.DefaultAgencySplit < 0 || in.DefaultOwnerSplit+in.DefaultAgencySplit > 100.001 {
		return nil, invalid("invalid_split", "El reparto entre propietario y agencia es inválido")
	}
	values := map[string]string{
		SettingIssuerName:         in.IssuerName,
		SettingIssuerTaxID:        in.IssuerTaxID,
		SettingIssuerAddress:      in.IssuerAddress,
		SettingIssuerEmail:        in.IssuerEmail,
		SettingInvoiceSeries:      in.InvoiceSeries,
		SettingReceiptSeries:      in.ReceiptSeries,
		SettingDefaultCurrency:    in.DefaultCurrency,
		SettingDefaultLeadDays:    fmt.Sprintf("%d", in.DefaultLeadDays),
		SettingDefaultOwnerSplit:  fmt.Sprintf("%.2f", in.DefaultOwnerSplit),
		SettingDefaultAgencySplit: fmt.Sprintf("%.2f", in.DefaultAgencySplit),
		SettingAutoInvoice:        fmt.Sprintf("%t", in.AutoInvoice),
		SettingAutoSendInvoice:    fmt.Sprintf("%t", in.AutoSendInvoice),
	}
	for k, v := range values {
		if err := s.settings.Set(ctx, k, v); err != nil {
			return nil, err
		}
	}
	return s.GetSettings(ctx)
}

// ---------------------------------------------------------------- helpers

func (s *Service) withDerived(ctx context.Context, l *Lease) {
	today := s.now().UTC().Format("2006-01-02")
	limit := s.now().UTC().AddDate(0, 0, 60).Format("2006-01-02")
	l.Expiring = l.Status == LeaseActive && l.EndDate >= today && l.EndDate <= limit
	charged, err := s.storage.SumCharged(ctx, l.ID)
	if err == nil {
		l.ChargedTotal = round2(charged)
	}
	paid, err := s.storage.SumPaid(ctx, l.ID)
	if err == nil {
		l.PaidTotal = round2(paid)
	}
	l.Balance = round2(l.ChargedTotal - l.PaidTotal)
}

func (s *Service) withApplicant(ctx context.Context, r *Reservation) {
	person, err := s.storage.GetPerson(ctx, r.ApplicantPersonID)
	if err == nil {
		r.Applicant = person
	}
}

func (s *Service) prepareLease(ctx context.Context, l *Lease) error {
	l.StartDate = strings.TrimSpace(l.StartDate)
	l.Currency = strings.ToUpper(strings.TrimSpace(l.Currency))
	l.OwnerBankAccount = strings.TrimSpace(l.OwnerBankAccount)
	l.Notes = strings.TrimSpace(l.Notes)
	l.CommissionBase = strings.ToLower(strings.TrimSpace(l.CommissionBase))
	l.CommissionType = strings.ToLower(strings.TrimSpace(l.CommissionType))
	l.LateFeeType = strings.ToLower(strings.TrimSpace(l.LateFeeType))

	if l.HomeID <= 0 {
		return invalid("home_required", "La vivienda es obligatoria")
	}
	exists, err := s.storage.HomeExists(ctx, l.HomeID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrHomeNotFound
	}
	if !validDate(l.StartDate) {
		return invalid("invalid_date", "La fecha de inicio es inválida (YYYY-MM-DD)")
	}
	if l.TermMonths < 1 || l.TermMonths > 120 {
		return invalid("invalid_term", "El plazo debe estar entre 1 y 120 meses")
	}
	end, err := leaseEndDate(l.StartDate, l.TermMonths)
	if err != nil {
		return invalid("invalid_date", "No se pudo calcular la fecha de fin")
	}
	l.EndDate = end
	if l.RentAmount < 0 || l.MaintenanceAmount < 0 || l.DepositAmount < 0 {
		return invalid("invalid_amount", "Los montos no pueden ser negativos")
	}
	if l.Currency == "" {
		settings, err := s.GetSettings(ctx)
		if err == nil && settings.DefaultCurrency != "" {
			l.Currency = settings.DefaultCurrency
		} else {
			l.Currency = DefaultCurrency
		}
	}
	if len(l.Currency) != 3 {
		return invalid("invalid_currency", "La moneda debe ser un código ISO de 3 letras")
	}
	if l.PaymentDay < 1 || l.PaymentDay > 28 {
		return invalid("invalid_payment_day", "El día de pago debe estar entre 1 y 28")
	}
	if l.LateFeeType == "" {
		l.LateFeeType = "none"
	}
	if !contains([]string{"none", "fixed", "percent"}, l.LateFeeType) {
		return invalid("invalid_late_fee", "Tipo de mora inválido")
	}
	if l.LateFeeValue < 0 || l.LateFeeCap < 0 {
		return invalid("invalid_late_fee", "Los valores de mora no pueden ser negativos")
	}
	if l.LateFeeType == "percent" && l.LateFeeValue > 100 {
		return invalid("invalid_late_fee", "El porcentaje de mora no puede superar 100")
	}
	if l.CommissionType == "" {
		l.CommissionType = "percent"
	}
	switch l.CommissionType {
	case "percent":
		if l.CommissionPercent < 0 || l.CommissionPercent > 100 {
			return invalid("invalid_commission", "El porcentaje de comisión debe estar entre 0 y 100")
		}
		l.CommissionAmount = 0
	case "fixed":
		if l.CommissionAmount < 0 {
			return invalid("invalid_commission", "El monto de la comisión no puede ser negativo")
		}
		l.CommissionPercent = 0
	default:
		return invalid("invalid_commission", "Tipo de comisión inválido")
	}
	if l.CommissionBase == "" {
		l.CommissionBase = "collected"
	}
	if l.CommissionBase != "collected" && l.CommissionBase != "invoiced" {
		return invalid("invalid_commission_base", "Base de comisión inválida")
	}
	if l.NoticeDays < 0 {
		return invalid("invalid_notice", "Los días de preaviso no pueden ser negativos")
	}
	if l.AgencyInstitutionID != nil {
		exists, realEstate, err := s.storage.AgencyIsRealEstate(ctx, *l.AgencyInstitutionID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrAgencyNotFound
		}
		if !realEstate {
			return invalid("agency_category", "La institución seleccionada no tiene la categoría Inmobiliaria")
		}
	}
	if l.AgentPersonID != nil {
		ok, err := s.storage.PersonExists(ctx, *l.AgentPersonID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrPersonNotFound
		}
	}
	if l.RentAmount == 0 && l.MaintenanceAmount == 0 {
		return invalid("rent_required", "La renta o la cuota de mantenimiento debe ser mayor que cero")
	}
	l.RentAmount = round2(l.RentAmount)
	l.MaintenanceAmount = round2(l.MaintenanceAmount)
	l.DepositAmount = round2(l.DepositAmount)
	l.CommissionPercent = round2(l.CommissionPercent)
	l.CommissionAmount = round2(l.CommissionAmount)
	return nil
}

func (s *Service) prepareReservation(ctx context.Context, r *Reservation, checkApplicant bool) error {
	r.Currency = strings.ToUpper(strings.TrimSpace(r.Currency))
	r.Reference = strings.TrimSpace(r.Reference)
	r.RequiredDocuments = strings.TrimSpace(r.RequiredDocuments)
	r.Notes = strings.TrimSpace(r.Notes)
	r.RefundPolicy = strings.ToLower(strings.TrimSpace(r.RefundPolicy))

	if r.HomeID <= 0 {
		return invalid("home_required", "La vivienda es obligatoria")
	}
	exists, err := s.storage.HomeExists(ctx, r.HomeID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrHomeNotFound
	}
	if checkApplicant {
		if r.ApplicantPersonID <= 0 {
			return invalid("applicant_required", "El solicitante es obligatorio")
		}
		ok, err := s.storage.PersonExists(ctx, r.ApplicantPersonID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrPersonNotFound
		}
	}
	if !validDateTime(r.StartsAt) || !validDateTime(r.ExpiresAt) {
		return invalid("invalid_date", "Las fechas del apartado son inválidas")
	}
	start, _ := time.Parse(time.RFC3339, r.StartsAt)
	end, _ := time.Parse(time.RFC3339, r.ExpiresAt)
	if !end.After(start) {
		return invalid("invalid_date", "El vencimiento debe ser posterior al inicio")
	}
	if r.ContractDate != "" && !validDate(r.ContractDate) {
		return invalid("invalid_date", "La fecha de firma es inválida")
	}
	if r.MoveInDate != "" && !validDate(r.MoveInDate) {
		return invalid("invalid_date", "La fecha de mudanza es inválida")
	}
	if r.Amount < 0 || r.RentAmount < 0 || r.MaintenanceAmount < 0 {
		return invalid("invalid_amount", "Los montos no pueden ser negativos")
	}
	if r.Currency == "" {
		r.Currency = DefaultCurrency
	}
	if len(r.Currency) != 3 {
		return invalid("invalid_currency", "La moneda debe ser un código ISO de 3 letras")
	}
	if r.RefundPolicy == "" {
		r.RefundPolicy = RefundForfeitable
	}
	if !contains(RefundPolicies, r.RefundPolicy) {
		return invalid("invalid_refund_policy", "Política de devolución inválida")
	}
	if r.OwnerSplitPercent < 0 || r.AgencySplitPercent < 0 ||
		r.OwnerSplitPercent+r.AgencySplitPercent > 100.001 {
		return invalid("invalid_split", "El reparto entre propietario y agencia es inválido")
	}
	if r.AgentPersonID != nil {
		ok, err := s.storage.PersonExists(ctx, *r.AgentPersonID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrPersonNotFound
		}
	}
	r.Amount = round2(r.Amount)
	r.RentAmount = round2(r.RentAmount)
	r.MaintenanceAmount = round2(r.MaintenanceAmount)
	return nil
}

func preparePerson(p *Person) error {
	p.FullName = strings.TrimSpace(p.FullName)
	p.Email = strings.TrimSpace(p.Email)
	p.Phone = strings.TrimSpace(p.Phone)
	p.TaxID = strings.TrimSpace(p.TaxID)
	p.Address = strings.TrimSpace(p.Address)
	p.Occupation = strings.TrimSpace(p.Occupation)
	p.Employer = strings.TrimSpace(p.Employer)
	p.MaritalStatus = strings.TrimSpace(p.MaritalStatus)
	p.Notes = strings.TrimSpace(p.Notes)
	if p.FullName == "" {
		return invalid("name_required", "El nombre es obligatorio")
	}
	if len(p.FullName) > 150 {
		return invalid("name_too_long", "El nombre es demasiado largo")
	}
	if p.Email != "" && !strings.Contains(p.Email, "@") {
		return invalid("invalid_email", "El email es inválido")
	}
	if p.MonthlyIncome < 0 {
		return invalid("invalid_income", "El ingreso mensual no puede ser negativo")
	}
	p.MonthlyIncome = round2(p.MonthlyIncome)
	return nil
}

// ConvertReservationInput son los datos que completa el usuario al convertir.
type ConvertReservationInput struct {
	TermMonths        int     `json:"term_months"`
	PaymentDay        int     `json:"payment_day"`
	DepositAmount     float64 `json:"deposit_amount"`
	CommissionPercent float64 `json:"commission_percent"`
	CommissionType    string  `json:"commission_type"`
	CommissionAmount  float64 `json:"commission_amount"`
	OwnerBankAccount  string  `json:"owner_bank_account"`
}

func leaseEndDate(start string, months int) (string, error) {
	t, err := time.Parse("2006-01-02", start)
	if err != nil {
		return "", err
	}
	y, m, d := t.Date()
	firstNext := time.Date(y, m+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstNext.AddDate(0, 1, -1).Day()
	if d > lastDay {
		d = lastDay
	}
	end := time.Date(firstNext.Year(), firstNext.Month(), d, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	return end.Format("2006-01-02"), nil
}

func validDate(value string) bool {
	if len(value) != 10 {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func validDateTime(value string) bool {
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

var errNoEmail = errors.New("no hay destinatarios con email")
