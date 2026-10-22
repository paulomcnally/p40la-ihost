package services

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutomationScheduler ejecuta el job diario de las cuentas de automation
// dentro del proceso de p40la (SPEC-110 fase 3).
type AutomationScheduler struct {
	automation  *AutomationService
	settings    *SystemSettingsService
	stopCh      chan struct{}
	mu          sync.Mutex
	running     map[int64]bool
	lastAttempt map[int64]time.Time
}

func NewAutomationScheduler(automation *AutomationService, settings *SystemSettingsService) *AutomationScheduler {
	return &AutomationScheduler{
		automation:  automation,
		settings:    settings,
		stopCh:      make(chan struct{}),
		running:     make(map[int64]bool),
		lastAttempt: make(map[int64]time.Time),
	}
}

func (s *AutomationScheduler) Start() {
	go s.run()
}

func (s *AutomationScheduler) Stop() {
	close(s.stopCh)
}

func (s *AutomationScheduler) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	slog.Info("automation scheduler iniciado")
	s.runDue()

	for {
		select {
		case <-ticker.C:
			s.runDue()
		case <-s.stopCh:
			slog.Info("automation scheduler detenido")
			return
		}
	}
}

func (s *AutomationScheduler) runDue() {
	ctx := context.Background()
	accounts, err := s.automation.storage.ListEnabledAccounts(ctx)
	if err != nil {
		slog.Error("automation scheduler: listar cuentas", "error", err)
		return
	}

	loc := time.UTC
	if s.settings != nil {
		if l, err := s.settings.GetTimezoneLocation(ctx); err == nil {
			loc = l
		}
	}
	now := time.Now().In(loc)

	for _, account := range accounts {
		if !s.acquire(account.ID) {
			continue
		}
		if !isAutomationDue(account, now, s.lastAttempt[account.ID]) {
			s.release(account.ID)
			continue
		}
		s.lastAttempt[account.ID] = now
		delivered, failed, err := s.automation.SyncAccount(ctx, account.ID)
		s.release(account.ID)
		if err != nil {
			slog.Warn("automation scheduler: job fallido",
				"account_id", account.ID, "delivered", delivered, "failed", failed, "error", err)
			continue
		}
		slog.Info("automation scheduler: job completado",
			"account_id", account.ID, "delivered", delivered, "failed", failed)
	}
}

func (s *AutomationScheduler) acquire(accountID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[accountID] {
		return false
	}
	s.running[accountID] = true
	return true
}

func (s *AutomationScheduler) release(accountID int64) {
	s.mu.Lock()
	delete(s.running, accountID)
	s.mu.Unlock()
}

// isAutomationDue decide si una cuenta debe ejecutarse en `now`. Se evita
// reintentar dentro del mismo minuto con lastAttempt.
func isAutomationDue(account models.AutomationAccount, now time.Time, lastAttempt time.Time) bool {
	if !account.ScheduleEnabled || account.ScheduleTime == nil || *account.ScheduleTime == "" {
		return false
	}
	if now.Format("15:04") != *account.ScheduleTime {
		return false
	}
	if !lastAttempt.IsZero() && now.Sub(lastAttempt) < 55*time.Second {
		return false
	}

	switch account.ScheduleFrequency {
	case "every_n_days":
		if account.ScheduleInterval == nil || *account.ScheduleInterval < 1 {
			return false
		}
		if account.LastRunAt == nil {
			return true
		}
		return now.Sub(*account.LastRunAt) >= time.Duration(*account.ScheduleInterval)*24*time.Hour
	case "weekly":
		if account.ScheduleDays == nil || !containsAutomationWeekday(*account.ScheduleDays, int(now.Weekday())) {
			return false
		}
		return account.LastRunAt == nil || !sameAutomationDay(account.LastRunAt, now)
	case "monthly":
		if account.ScheduleDays == nil {
			return false
		}
		day, err := strconv.Atoi(strings.TrimSpace(*account.ScheduleDays))
		if err != nil || day != now.Day() {
			return false
		}
		return account.LastRunAt == nil || !sameAutomationDay(account.LastRunAt, now)
	default: // daily
		return account.LastRunAt == nil || !sameAutomationDay(account.LastRunAt, now)
	}
}

func containsAutomationWeekday(csv string, weekday int) bool {
	for _, part := range strings.Split(csv, ",") {
		day, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && day == weekday {
			return true
		}
	}
	return false
}

func sameAutomationDay(value *time.Time, now time.Time) bool {
	if value == nil {
		return false
	}
	y1, m1, d1 := value.Date()
	y2, m2, d2 := now.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
