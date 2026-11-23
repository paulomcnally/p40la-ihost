package rental

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ListReminders lista recordatorios por casa o contrato.
func (s *Service) ListReminders(ctx context.Context, homeID, leaseID int64) ([]Reminder, error) {
	return s.storage.ListReminders(ctx, homeID, leaseID)
}

// CreateReminder valida e inserta un recordatorio.
func (s *Service) CreateReminder(ctx context.Context, r *Reminder) (*Reminder, error) {
	r.ID = 0
	if err := s.prepareReminder(ctx, r); err != nil {
		return nil, err
	}
	return s.storage.CreateReminder(ctx, r)
}

// UpdateReminder valida y actualiza un recordatorio.
func (s *Service) UpdateReminder(ctx context.Context, r *Reminder) (*Reminder, error) {
	existing, err := s.storage.GetReminder(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrReminderNotFound
	}
	if err := s.prepareReminder(ctx, r); err != nil {
		return nil, err
	}
	return s.storage.UpdateReminder(ctx, r)
}

// DeleteReminder elimina un recordatorio.
func (s *Service) DeleteReminder(ctx context.Context, id int64) error {
	return s.storage.DeleteReminder(ctx, id)
}

// RunReminderNow envía un recordatorio de inmediato.
func (s *Service) RunReminderNow(ctx context.Context, id int64) (int, error) {
	r, err := s.storage.GetReminder(ctx, id)
	if err != nil {
		return 0, err
	}
	if r == nil {
		return 0, ErrReminderNotFound
	}
	return s.sendReminder(ctx, r, "manual")
}

// ProcessDueReminders envía los recordatorios cuya anticipación se cumplió.
func (s *Service) ProcessDueReminders(ctx context.Context) (int, error) {
	reminders, err := s.storage.ListEnabledReminders(ctx)
	if err != nil {
		return 0, err
	}
	sent := 0
	now := s.now().UTC()
	for _, r := range reminders {
		due, err := time.Parse(time.RFC3339, r.NextDueAt)
		if err != nil {
			continue
		}
		notifyAt := due.AddDate(0, 0, -r.LeadDays)
		if now.Before(notifyAt) {
			continue
		}
		occurrence := r.NextDueAt
		exists, err := s.storage.ReminderEventExists(ctx, r.ID, occurrence)
		if err != nil {
			return sent, err
		}
		if exists {
			continue
		}
		if _, err := s.sendReminder(ctx, &r, "scheduler"); err != nil {
			slog.Warn("recordatorio de alquiler falló", "reminder_id", r.ID, "error", err.Error())
		}
		sent++
	}
	return sent, nil
}

func (s *Service) prepareReminder(ctx context.Context, r *Reminder) error {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	r.ScheduleType = strings.ToLower(strings.TrimSpace(r.ScheduleType))
	r.NotifyRoles = strings.TrimSpace(r.NotifyRoles)
	r.ExtraEmails = strings.TrimSpace(r.ExtraEmails)
	r.AnchorDate = strings.TrimSpace(r.AnchorDate)
	r.NextDueAt = strings.TrimSpace(r.NextDueAt)

	if r.Title == "" {
		return invalid("title_required", "El título es obligatorio")
	}
	if r.HomeID == nil && r.LeaseID == nil {
		return invalid("owner_required", "El recordatorio debe asociarse a una vivienda o contrato")
	}
	if r.HomeID != nil {
		ok, err := s.storage.HomeExists(ctx, *r.HomeID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrHomeNotFound
		}
	}
	if r.LeaseID != nil {
		l, err := s.storage.GetLease(ctx, *r.LeaseID)
		if err != nil {
			return err
		}
		if l == nil {
			return ErrLeaseNotFound
		}
	}
	if r.Kind == "" {
		r.Kind = "custom"
	}
	if !contains(ReminderKinds, r.Kind) {
		return invalid("invalid_kind", "Tipo de recordatorio inválido")
	}
	if r.ScheduleType == "" {
		r.ScheduleType = "once"
	}
	if !contains(ScheduleTypes, r.ScheduleType) {
		return invalid("invalid_schedule", "Frecuencia inválida")
	}
	if r.IntervalN <= 0 {
		r.IntervalN = 1
	}
	if !validDate(r.AnchorDate) {
		return invalid("invalid_date", "La fecha de referencia es inválida")
	}
	if r.NextDueAt == "" {
		r.NextDueAt = r.AnchorDate + "T09:00:00Z"
	}
	if !validDateTime(r.NextDueAt) {
		return invalid("invalid_date", "La próxima fecha del recordatorio es inválida")
	}
	if r.LeadDays < 0 || r.LeadDays > 365 {
		return invalid("invalid_lead_days", "Los días de anticipación deben estar entre 0 y 365")
	}
	if r.NotifyRoles == "" {
		r.NotifyRoles = RoleTenant
	}
	roles := splitCSV(r.NotifyRoles)
	for _, role := range roles {
		if !contains(PartyRoles, role) {
			return invalid("invalid_roles", "Rol de destinatario inválido: "+role)
		}
	}
	if len(splitCSV(r.ExtraEmails)) == 0 && r.LeaseID == nil {
		return invalid("recipients_required", "Sin contrato, agregá al menos un email de destinatario")
	}
	return nil
}

func (s *Service) sendReminder(ctx context.Context, r *Reminder, source string) (int, error) {
	var to []string
	roles := splitCSV(r.NotifyRoles)
	if r.LeaseID != nil {
		people, err := s.storage.NotifyParties(ctx, *r.LeaseID, roles)
		if err != nil {
			return 0, err
		}
		for _, p := range people {
			to = append(to, p.Email)
		}
	}
	for _, email := range splitCSV(r.ExtraEmails) {
		if strings.Contains(email, "@") {
			to = append(to, email)
		}
	}
	to = uniqueStrings(to)
	if len(to) == 0 {
		if err := s.storage.InsertReminderEvent(ctx, r.ID, r.NextDueAt, r.NextDueAt, "skipped", "no_recipients"); err != nil {
			return 0, err
		}
		s.advanceReminder(ctx, r)
		return 0, invalid("no_recipients", "No hay destinatarios con email")
	}

	homeName := ""
	if r.LeaseID != nil {
		if l, err := s.storage.GetLease(ctx, *r.LeaseID); err == nil && l != nil {
			homeName = l.HomeName
		}
	}
	if homeName == "" && r.HomeID != nil {
		if st, err := s.storage.GetHomeListing(ctx, *r.HomeID); err == nil {
			homeName = st.HomeName
		}
	}
	title := "Recordatorio: " + r.Title
	content := fmt.Sprintf(
		`<p><strong>%s</strong></p><p>%s</p><p>Vivienda: %s<br>Fecha prevista: %s</p>`,
		html.EscapeString(r.Title), html.EscapeString(r.Description),
		html.EscapeString(homeName), html.EscapeString(r.NextDueAt))
	body := s.email.RenderTemplate(ctx, title, content)
	err := s.email.Send(ctx, to, "P40LA — "+title, body)
	result := "sent"
	errorCode := ""
	if err != nil {
		result = "failed"
		errorCode = "smtp_error"
	}
	if logErr := s.storage.InsertReminderEvent(ctx, r.ID, r.NextDueAt, r.NextDueAt, result, errorCode); logErr != nil && err == nil {
		err = logErr
	}
	_ = s.storage.InsertEmailLog(ctx, "reminder", r.ID, r.NextDueAt, len(to), r.NotifyRoles, result, errorCode)
	if err != nil {
		// El evento evita reintentos infinitos de la misma ocurrencia.
		s.advanceReminder(ctx, r)
		return len(to), err
	}
	s.advanceReminder(ctx, r)
	slog.Info("recordatorio de alquiler enviado", "reminder_id", r.ID, "source", source, "recipients", len(to))
	return len(to), nil
}

func (s *Service) advanceReminder(ctx context.Context, r *Reminder) {
	due, err := time.Parse(time.RFC3339, r.NextDueAt)
	if err != nil {
		return
	}
	var next time.Time
	disable := false
	switch r.ScheduleType {
	case "once":
		disable = true
	case "monthly":
		next = due.AddDate(0, 1, 0)
	case "yearly":
		next = due.AddDate(1, 0, 0)
	case "every_n_days":
		next = due.AddDate(0, 0, r.IntervalN)
	default:
		disable = true
	}
	if disable {
		r.Enabled = false
		if _, err := s.storage.UpdateReminder(ctx, r); err != nil {
			slog.Warn("no se pudo desactivar el recordatorio", "reminder_id", r.ID, "error", err)
		}
		return
	}
	if next.Before(s.now().UTC()) {
		next = s.now().UTC().AddDate(0, 0, 1)
	}
	if err := s.storage.AdvanceReminder(ctx, r.ID, next.Format(time.RFC3339)); err != nil {
		slog.Warn("no se pudo avanzar el recordatorio", "reminder_id", r.ID, "error", err)
	}
}

// RentalScheduler ejecuta tareas idempotentes del módulo.
type RentalScheduler struct {
	service  *Service
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	once     sync.Once
	mu       sync.Mutex
	started  bool
}

// NewRentalScheduler crea el scheduler.
func NewRentalScheduler(service *Service, interval time.Duration) *RentalScheduler {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &RentalScheduler{service: service, interval: interval, stop: make(chan struct{}), done: make(chan struct{})}
}

// Start arranca el loop en background.
func (s *RentalScheduler) Start() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		// Primera pasada al arrancar, sin esperar el tick.
		_, _ = s.service.RunOnce(context.Background())
		for {
			select {
			case <-ticker.C:
				_, _ = s.service.RunOnce(context.Background())
			case <-s.stop:
				return
			}
		}
	}()
}

// Stop detiene el loop.
func (s *RentalScheduler) Stop() {
	s.mu.Lock()
	started := s.started
	s.mu.Unlock()
	if !started {
		return
	}
	s.once.Do(func() { close(s.stop) })
	<-s.done
}

// RunOnce ejecuta vencimiento de apartados y recordatorios.
func (s *Service) RunOnce(ctx context.Context) (int, error) {
	expired, err := s.ExpireReservations(ctx)
	if err != nil {
		return 0, err
	}
	invoiced, err := s.AutoInvoiceDuePeriods(ctx)
	if err != nil {
		return expired, err
	}
	sent, err := s.ProcessDueReminders(ctx)
	if err != nil {
		return expired + invoiced, err
	}
	return expired + invoiced + sent, nil
}

// RunOnce del scheduler delega en el servicio.
func (s *RentalScheduler) RunOnce(ctx context.Context) (int, error) {
	return s.service.RunOnce(ctx)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		key := strings.ToLower(v)
		if v == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
	}
	return out
}
