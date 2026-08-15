package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type BillingScheduler struct {
	serviceStorage    *storage.ServiceStorage
	billStorage       *storage.BillStorage
	settingsService   *SystemSettingsService
	stopCh            chan struct{}
	lastGenerationKey string
}

func NewBillingScheduler(
	serviceStorage *storage.ServiceStorage,
	billStorage *storage.BillStorage,
	settingsService *SystemSettingsService,
) *BillingScheduler {
	return &BillingScheduler{
		serviceStorage:    serviceStorage,
		billStorage:       billStorage,
		settingsService:   settingsService,
		stopCh:            make(chan struct{}),
		lastGenerationKey: "last_billing_generation",
	}
}

func (s *BillingScheduler) Start() {
	go s.run()
}

func (s *BillingScheduler) Stop() {
	close(s.stopCh)
}

func (s *BillingScheduler) run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	slog.Info("billing scheduler iniciado")

	for {
		select {
		case <-ticker.C:
			s.checkAndGenerate()
		case <-s.stopCh:
			slog.Info("billing scheduler detenido")
			return
		}
	}
}

func (s *BillingScheduler) checkAndGenerate() {
	ctx := context.Background()

	hour, err := s.settingsService.GetBillingGenerationHour(ctx)
	if err != nil {
		slog.Error("billing scheduler: error al obtener hora de generación", "error", err)
		return
	}

	now := time.Now()
	if now.Hour() != hour {
		return
	}

	lastGen, err := s.settingsService.GetSetting(ctx, s.lastGenerationKey)
	if err != nil {
		slog.Error("billing scheduler: error al obtener última generación", "error", err)
		return
	}

	today := now.Format("2006-01-02")
	if lastGen != nil && lastGen.Value == today {
		return
	}

	slog.Info("billing scheduler: generando facturas automáticas", "date", today)

	services, err := s.serviceStorage.List(ctx, nil)
	if err != nil {
		slog.Error("billing scheduler: error al listar servicios", "error", err)
		return
	}

	generated := 0
	for _, svc := range services {
		if !svc.AutoGenerate || !svc.Active {
			continue
		}

		if err := s.generateBillForService(ctx, &svc, now); err != nil {
			slog.Error("billing scheduler: error al generar factura", "service_id", svc.ID, "error", err)
			continue
		}
		generated++
	}

	if err := s.settingsService.SetBillingGenerationHour(ctx, hour); err != nil {
		slog.Error("billing scheduler: error al actualizar última generación", "error", err)
	}
	_ = s.settingsService.Set(ctx, s.lastGenerationKey, today)

	slog.Info("billing scheduler: generación completada", "generated", generated)
}

func (s *BillingScheduler) generateBillForService(ctx context.Context, svc *models.Service, now time.Time) error {
	year := now.Year()
	month := now.Month()

	if svc.Frequency == "yearly" {
		if month != time.January {
			return nil
		}
	} else {
		targetDay := svc.BillingDay
		lastDay := daysInMonth(year, month)
		if targetDay > lastDay {
			targetDay = lastDay
		}
		if now.Day() != targetDay {
			return nil
		}
	}

	billMonth := int(month)
	if svc.Frequency == "yearly" {
		billMonth = 0
	}

	existing, err := s.billStorage.FindByServicePeriod(ctx, svc.ID, year, billMonth)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	bill := &models.Bill{
		ServiceID: svc.ID,
		Year:      year,
		Month:     billMonth,
		Amount:    svc.SuggestedAmount,
		Status:    "pending",
	}
	_, err = s.billStorage.Create(ctx, bill)
	return err
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
