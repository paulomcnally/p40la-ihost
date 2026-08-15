package services

import (
	"context"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type SystemSettingsService struct {
	storage *storage.SystemSettingsStorage
}

func NewSystemSettingsService(storage *storage.SystemSettingsStorage) *SystemSettingsService {
	return &SystemSettingsService{storage: storage}
}

func (s *SystemSettingsService) GetBillingGenerationHour(ctx context.Context) (int, error) {
	val, err := s.storage.Get(ctx, "billing_generation_hour")
	if err != nil {
		return 0, err
	}
	if val == "" {
		return 0, nil
	}
	hour, err := strconv.Atoi(val)
	if err != nil {
		return 0, nil
	}
	return hour, nil
}

func (s *SystemSettingsService) SetBillingGenerationHour(ctx context.Context, hour int) error {
	if hour < 0 || hour > 23 {
		return nil
	}
	return s.storage.Set(ctx, "billing_generation_hour", strconv.Itoa(hour))
}

func (s *SystemSettingsService) Set(ctx context.Context, key, value string) error {
	return s.storage.Set(ctx, key, value)
}

func (s *SystemSettingsService) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	return s.storage.GetSetting(ctx, key)
}
