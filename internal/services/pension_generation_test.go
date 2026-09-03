package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type generationFixture struct {
	gen            *PensionGenerationService
	salarySvc      *SalaryService
	configSvc      *ChildSupportConfigService
	salaryID       int64
	childID        int64
	catID          int64
	salaryPayments *storage.SalaryPaymentStorage
	records        *storage.SupportRecordStorage
}

func newGenerationFixture(t *testing.T) *generationFixture {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	salaryStorage := storage.NewSalaryStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	salaryPaymentStorage := storage.NewSalaryPaymentStorage(database)
	recordStorage := storage.NewSupportRecordStorage(database)
	configStorage := storage.NewChildSupportConfigStorage(database)

	// Notificaciones con servicio vacío (nunca envía en tests).
	notif := NewPensionNotificationService(
		storage.NewNotificationStorage(database),
		NewEmailService(NewSystemSettingsService(storage.NewSystemSettingsStorage(database))),
		NewAlertService(storage.NewAlertStorage(database)),
		NewSystemSettingsService(storage.NewSystemSettingsStorage(database)),
		recordStorage,
		salaryPaymentStorage,
	)

	gen := NewPensionGenerationService(salaryStorage, currencyStorage, salaryPaymentStorage, recordStorage, configStorage, notif)
	salarySvc := NewSalaryService(salaryStorage, currencyStorage)
	configSvc := NewChildSupportConfigService(configStorage, storage.NewChildStorage(database), storage.NewPensionCategoryStorage(database))

	ctx := context.Background()
	child, err := NewChildService(storage.NewChildStorage(database)).Create(ctx, "Juan", "Pérez", "2015-01-01", "")
	if err != nil {
		t.Fatalf("crear hijo: %v", err)
	}
	cat, err := NewPensionCategoryService(storage.NewPensionCategoryStorage(database)).Create(ctx, "Colegio", "", false)
	if err != nil {
		t.Fatalf("crear categoría: %v", err)
	}
	salary, err := salarySvc.Create(ctx, "Empresa XYZ", 15000, 1, 15, true, "")
	if err != nil {
		t.Fatalf("crear salario: %v", err)
	}
	// Config activa con auto_generate para el hijo+categoría.
	if _, err := configSvc.Create(ctx, child.ID, cat.ID, 1500, "NIO", true, true); err != nil {
		t.Fatalf("crear config: %v", err)
	}

	return &generationFixture{
		gen:            gen,
		salarySvc:      salarySvc,
		configSvc:      configSvc,
		salaryID:       salary.ID,
		childID:        child.ID,
		catID:          cat.ID,
		salaryPayments: salaryPaymentStorage,
		records:        recordStorage,
	}
}

func TestPensionGenerationService_Generate(t *testing.T) {
	f := newGenerationFixture(t)
	ctx := context.Background()

	result, err := f.gen.Generate(ctx, 2026, 8)
	if err != nil {
		t.Fatalf("generar mes: %v", err)
	}
	if result.CreatedSalaryPayments != 1 {
		t.Fatalf("se esperaba 1 pago de salario, count=%d", result.CreatedSalaryPayments)
	}
	if result.CreatedSupportRecords != 1 {
		t.Fatalf("se esperaba 1 registro, count=%d", result.CreatedSupportRecords)
	}

	// Verificar que los registros quedaron pendientes.
	payments, err := f.salaryPayments.ListByFilters(ctx, 2026, 8, 0)
	if err != nil {
		t.Fatalf("list pagos: %v", err)
	}
	if len(payments) != 1 || payments[0].Status != "pending" || payments[0].Employer != "Empresa XYZ" {
		t.Fatalf("pago de salario incorrecto: %+v", payments)
	}
	records, err := f.records.ListByFilters(ctx, 2026, 8, 0)
	if err != nil {
		t.Fatalf("list registros: %v", err)
	}
	if len(records) != 1 || records[0].Status != "pending" || records[0].Amount != 1500 {
		t.Fatalf("registro incorrecto: %+v", records)
	}
}

func TestPensionGenerationService_Idempotent(t *testing.T) {
	f := newGenerationFixture(t)
	ctx := context.Background()

	if _, err := f.gen.Generate(ctx, 2026, 8); err != nil {
		t.Fatalf("primera generación: %v", err)
	}
	result, err := f.gen.Generate(ctx, 2026, 8)
	if err != nil {
		t.Fatalf("segunda generación: %v", err)
	}
	if result.CreatedSalaryPayments != 0 || result.CreatedSupportRecords != 0 {
		t.Fatalf("la generación debería ser idempotente: %+v", result)
	}
}

func TestPensionGenerationService_NoSalaryNoRecords(t *testing.T) {
	f := newGenerationFixture(t)
	ctx := context.Background()

	// Desactivar el salario: no se crean pagos → no se generan registros.
	if _, err := f.salarySvc.Update(ctx, f.salaryID, "Empresa XYZ", 15000, 1, 15, false, ""); err != nil {
		t.Fatalf("desactivar salario: %v", err)
	}
	result, err := f.gen.Generate(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("generar mes: %v", err)
	}
	if result.CreatedSalaryPayments != 0 || result.CreatedSupportRecords != 0 {
		t.Fatalf("sin salarios activos no debería crear nada: %+v", result)
	}
}

func TestPensionGenerationService_ConfigNotAutoGenerate(t *testing.T) {
	f := newGenerationFixture(t)
	ctx := context.Background()

	// Config sin auto_generate: el salario se genera pero el registro no.
	if _, err := f.configSvc.Update(ctx, mustConfigID(t, f), f.catID, 1500, "NIO", true, false); err != nil {
		t.Fatalf("update config: %v", err)
	}
	result, err := f.gen.Generate(ctx, 2026, 10)
	if err != nil {
		t.Fatalf("generar mes: %v", err)
	}
	if result.CreatedSalaryPayments != 1 {
		t.Fatalf("se esperaba 1 pago de salario, count=%d", result.CreatedSalaryPayments)
	}
	if result.CreatedSupportRecords != 0 {
		t.Fatalf("config sin auto_generate no debería generar registros, count=%d", result.CreatedSupportRecords)
	}
}

func mustConfigID(t *testing.T, f *generationFixture) int64 {
	t.Helper()
	configs, err := f.configSvc.List(context.Background())
	if err != nil || len(configs) == 0 {
		t.Fatalf("no hay configs: %v", err)
	}
	return configs[0].ID
}

func TestPensionGenerationService_InvalidPeriod(t *testing.T) {
	f := newGenerationFixture(t)
	ctx := context.Background()

	if _, err := f.gen.Generate(ctx, 2026, 0); err == nil {
		t.Fatal("se esperaba error con mes 0")
	}
	if _, err := f.gen.Generate(ctx, 2026, 13); err == nil {
		t.Fatal("se esperaba error con mes 13")
	}
}
