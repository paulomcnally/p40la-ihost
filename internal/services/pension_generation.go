package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// PensionGenerationService genera los pagos de salario y registros de
// manutención de un mes, a partir de los salarios activos y las configs de
// pensión activas con auto_generate (SPEC-051). Idempotente: no duplica.
type PensionGenerationService struct {
	salaryStorage        *storage.SalaryStorage
	currencyStorage      *storage.CurrencyStorage
	salaryPaymentStorage *storage.SalaryPaymentStorage
	recordStorage        *storage.SupportRecordStorage
	configStorage        *storage.ChildSupportConfigStorage
	notification         *PensionNotificationService
}

// NewPensionGenerationService crea un nuevo PensionGenerationService.
func NewPensionGenerationService(
	salaryStorage *storage.SalaryStorage,
	currencyStorage *storage.CurrencyStorage,
	salaryPaymentStorage *storage.SalaryPaymentStorage,
	recordStorage *storage.SupportRecordStorage,
	configStorage *storage.ChildSupportConfigStorage,
	notification *PensionNotificationService,
) *PensionGenerationService {
	return &PensionGenerationService{
		salaryStorage:        salaryStorage,
		currencyStorage:      currencyStorage,
		salaryPaymentStorage: salaryPaymentStorage,
		recordStorage:        recordStorage,
		configStorage:        configStorage,
		notification:         notification,
	}
}

// GenerateResult resume lo creado en una generación.
type GenerateResult struct {
	Year                  int `json:"year"`
	Month                 int `json:"month"`
	CreatedSalaryPayments int `json:"created_salary_payments"`
	CreatedSupportRecords int `json:"created_support_records"`
}

// Generate genera los pagos de salario y registros del mes.
func (s *PensionGenerationService) Generate(ctx context.Context, year, month int) (*GenerateResult, error) {
	if err := validatePeriod(year, month); err != nil {
		return nil, err
	}

	createdSalaryPayments, err := s.generateSalaryPayments(ctx, year, month)
	if err != nil {
		return nil, err
	}

	createdSupportRecords, err := s.generateSupportRecords(ctx, year, month)
	if err != nil {
		return nil, err
	}

	if len(createdSalaryPayments) > 0 || len(createdSupportRecords) > 0 {
		s.notification.SendRecordsCreated(ctx, createdSalaryPayments, createdSupportRecords, year, month)
	}

	return &GenerateResult{
		Year:                  year,
		Month:                 month,
		CreatedSalaryPayments: len(createdSalaryPayments),
		CreatedSupportRecords: len(createdSupportRecords),
	}, nil
}

func (s *PensionGenerationService) generateSalaryPayments(ctx context.Context, year, month int) ([]models.SalaryPayment, error) {
	salaries, err := s.salaryStorage.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar salarios: %w", err)
	}

	var created []models.SalaryPayment
	for i := range salaries {
		sal := &salaries[i]
		if !sal.Active {
			continue
		}

		exists, err := s.salaryPaymentStorage.Exists(ctx, sal.ID, year, month)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}

		currency := "NIO"
		if cur, err := s.currencyStorage.GetByID(ctx, sal.CurrencyID); err == nil && cur != nil {
			currency = cur.Code
		}

		payment, err := s.salaryPaymentStorage.Create(ctx, &models.SalaryPayment{
			SalaryID: sal.ID,
			Year:     year,
			Month:    month,
			Amount:   sal.Amount,
			Currency: currency,
			Status:   "pending",
		})
		if err != nil {
			return nil, fmt.Errorf("crear pago de salario: %w", err)
		}
		payment.Employer = sal.Employer
		created = append(created, *payment)
		slog.Info("pension generation: pago de salario creado", "salary_id", sal.ID, "period", fmt.Sprintf("%d/%d", month, year))
	}
	return created, nil
}

func (s *PensionGenerationService) generateSupportRecords(ctx context.Context, year, month int) ([]models.SupportRecord, error) {
	// Solo genera registros si hay al menos un pago de salario en el mes.
	salaryPayments, err := s.salaryPaymentStorage.ListByFilters(ctx, year, month, 0)
	if err != nil {
		return nil, fmt.Errorf("listar pagos de salario: %w", err)
	}
	if len(salaryPayments) == 0 {
		return nil, nil
	}

	configs, err := s.configStorage.ListActiveAutoGenerate(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar configs activas: %w", err)
	}

	var created []models.SupportRecord
	for i := range configs {
		cfg := &configs[i]

		exists, err := s.recordStorage.Exists(ctx, cfg.ChildID, cfg.PensionCategoryID, year, month)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}

		record, err := s.recordStorage.Create(ctx, &models.SupportRecord{
			ChildID:           cfg.ChildID,
			PensionCategoryID: cfg.PensionCategoryID,
			Year:              year,
			Month:             month,
			Amount:            cfg.Amount,
			Currency:          cfg.Currency,
			Status:            "pending",
		})
		if err != nil {
			return nil, fmt.Errorf("crear registro de manutención: %w", err)
		}
		created = append(created, *record)
		slog.Info("pension generation: registro creado", "child_id", cfg.ChildID, "category_id", cfg.PensionCategoryID, "period", fmt.Sprintf("%d/%d", month, year))
	}
	return created, nil
}
