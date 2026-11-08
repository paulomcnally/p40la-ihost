package services

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const testAutoPDF = "%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n"

type autoRecordsFixture struct {
	database     *sql.DB
	autoID       int64
	workshopID   int64
	inspectionID int64
	currencyID   int64
	dataDir      string
	maintenance  *AutoMaintenanceService
	inspection   *AutoInspectionService
	attachments  *AutoAttachmentService
	autoService  *AutoService
}

func newAutoRecordsFixture(t *testing.T) *autoRecordsFixture {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	autoStorage := storage.NewAutoStorage(database)
	institutionStorage := storage.NewInstitutionStorage(database)
	categoryStorage := storage.NewInstitutionCategoryStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	maintenanceStorage := storage.NewAutoMaintenanceStorage(database)
	inspectionStorage := storage.NewAutoInspectionStorage(database)
	attachmentStorage := storage.NewAutoAttachmentStorage(database)

	auto, err := autoStorage.Create(ctx, 2020, "Corolla", "Toyota", "white", "vehicle", "", "", "", "")
	if err != nil {
		t.Fatalf("crear auto: %v", err)
	}

	workshopCategory, err := categoryStorage.GetByKey(ctx, institutionCategoryWorkshop)
	if err != nil || workshopCategory == nil {
		t.Fatalf("categoría workshop: %v", err)
	}
	inspectionCategory, err := categoryStorage.GetByKey(ctx, institutionCategoryMechanicalInspection)
	if err != nil || inspectionCategory == nil {
		t.Fatalf("categoría mechanical_inspection: %v", err)
	}

	workshop, err := institutionStorage.Create(ctx, &models.Institution{
		Name:       "Taller Demo",
		CategoryID: &workshopCategory.ID,
	})
	if err != nil {
		t.Fatalf("crear taller: %v", err)
	}
	inspectionCenter, err := institutionStorage.Create(ctx, &models.Institution{
		Name:       "Centro Demo",
		CategoryID: &inspectionCategory.ID,
	})
	if err != nil {
		t.Fatalf("crear centro de inspección: %v", err)
	}

	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("listar monedas: %v", err)
	}

	dataDir := t.TempDir()
	attachmentService := NewAutoAttachmentService(attachmentStorage, autoStorage, dataDir)
	maintenanceService := NewAutoMaintenanceService(maintenanceStorage, autoStorage, institutionStorage, currencyStorage, attachmentService)
	inspectionService := NewAutoInspectionService(inspectionStorage, autoStorage, institutionStorage, attachmentService)
	fixedNow := time.Date(2026, time.November, 7, 12, 0, 0, 0, time.UTC)
	maintenanceService.now = func() time.Time { return fixedNow }
	inspectionService.now = func() time.Time { return fixedNow }

	autoService := NewAutoService(autoStorage)
	autoService.SetFilesCleaner(attachmentService)

	return &autoRecordsFixture{
		database:     database,
		autoID:       auto.ID,
		workshopID:   workshop.ID,
		inspectionID: inspectionCenter.ID,
		currencyID:   currencies[0].ID,
		dataDir:      dataDir,
		maintenance:  maintenanceService,
		inspection:   inspectionService,
		attachments:  attachmentService,
		autoService:  autoService,
	}
}

func (f *autoRecordsFixture) createMaintenance(t *testing.T, performedAt string, mileage int64) *models.AutoMaintenanceDetail {
	t.Helper()
	item, err := f.maintenance.Create(context.Background(), f.autoID, models.AutoMaintenanceInput{
		PerformedAt:   performedAt,
		Mileage:       mileage,
		InstitutionID: &f.workshopID,
		LaborCost:     45,
		SuppliesCost:  80,
		CurrencyID:    f.currencyID,
	})
	if err != nil {
		t.Fatalf("crear mantenimiento: %v", err)
	}
	return item
}

func (f *autoRecordsFixture) createInspection(t *testing.T, performedAt string, year int64, result string) *models.AutoInspectionDetail {
	t.Helper()
	item, err := f.inspection.Create(context.Background(), f.autoID, models.AutoInspectionInput{
		PerformedAt:    performedAt,
		InspectionYear: year,
		InstitutionID:  f.inspectionID,
		Result:         result,
		Notes:          "Demo",
	})
	if err != nil {
		t.Fatalf("crear inspección: %v", err)
	}
	return item
}

func TestAutoMaintenanceServiceCreateListAndTotals(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()

	first := f.createMaintenance(t, "2026-10-15", 42000)
	if first.WorkshopName != "Taller Demo" {
		t.Errorf("workshop_name autocompletado: esperado Taller Demo, got %q", first.WorkshopName)
	}
	if first.TotalCost != 125 {
		t.Errorf("total_cost: esperado 125, got %v", first.TotalCost)
	}

	second, err := f.maintenance.Create(ctx, f.autoID, models.AutoMaintenanceInput{
		PerformedAt:  "2026-10-20",
		Mileage:      50000,
		WorkshopName: "Otro Taller Demo",
		LaborCost:    10,
		SuppliesCost: 5,
		CurrencyID:   f.currencyID,
	})
	if err != nil {
		t.Fatalf("crear segundo mantenimiento: %v", err)
	}
	if second.InstitutionID != nil {
		t.Errorf("institution_id debería ser nulo, got %v", *second.InstitutionID)
	}

	list, err := f.maintenance.List(ctx, f.autoID)
	if err != nil {
		t.Fatalf("listar mantenimientos: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("items: esperado 2, got %d", len(list.Items))
	}
	if list.LatestMileage != 50000 {
		t.Errorf("latest_mileage: esperado 50000, got %d", list.LatestMileage)
	}
	if len(list.TotalsByCurrency) != 1 || list.TotalsByCurrency[0].Total != 140 {
		t.Errorf("totales por moneda inesperados: %+v", list.TotalsByCurrency)
	}

	updated, err := f.maintenance.Update(ctx, f.autoID, first.ID, models.AutoMaintenanceInput{
		PerformedAt:  "2026-10-16",
		Mileage:      43000,
		WorkshopName: "Taller Demo",
		LaborCost:    50,
		SuppliesCost: 90,
		CurrencyID:   f.currencyID,
	})
	if err != nil {
		t.Fatalf("actualizar mantenimiento: %v", err)
	}
	if updated.Mileage != 43000 || updated.TotalCost != 140 {
		t.Errorf("mantenimiento actualizado inesperado: %+v", updated)
	}

	if err := f.maintenance.Delete(ctx, f.autoID, first.ID); err != nil {
		t.Fatalf("eliminar mantenimiento: %v", err)
	}
	list, err = f.maintenance.List(ctx, f.autoID)
	if err != nil {
		t.Fatalf("listar después de eliminar: %v", err)
	}
	if len(list.Items) != 1 {
		t.Errorf("items después de eliminar: esperado 1, got %d", len(list.Items))
	}
}

func TestAutoMaintenanceServiceUsaNombreDeInstitucion(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()

	item, err := f.maintenance.Create(ctx, f.autoID, models.AutoMaintenanceInput{
		PerformedAt:   "2026-10-15",
		Mileage:       1000,
		InstitutionID: &f.workshopID,
		WorkshopName:  "Nombre ingresado por el cliente",
		LaborCost:     1,
		SuppliesCost:  2,
		CurrencyID:    f.currencyID,
	})
	if err != nil {
		t.Fatalf("crear mantenimiento: %v", err)
	}
	if item.WorkshopName != "Taller Demo" {
		t.Fatalf("workshop_name debería derivarse de la institución: got %q", item.WorkshopName)
	}
}

func TestAutoMaintenanceServiceValidaciones(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		input   models.AutoMaintenanceInput
		wantErr error
	}{
		{
			name: "taller requerido",
			input: models.AutoMaintenanceInput{
				PerformedAt: "2026-10-15", Mileage: 1000, CurrencyID: f.currencyID,
			},
		},
		{
			name: "kilometraje negativo",
			input: models.AutoMaintenanceInput{
				PerformedAt: "2026-10-15", Mileage: -1, WorkshopName: "Taller Demo", CurrencyID: f.currencyID,
			},
		},
		{
			name: "fecha futura",
			input: models.AutoMaintenanceInput{
				PerformedAt: "2026-12-01", Mileage: 1000, WorkshopName: "Taller Demo", CurrencyID: f.currencyID,
			},
		},
		{
			name: "institución de categoría incorrecta",
			input: models.AutoMaintenanceInput{
				PerformedAt: "2026-10-15", Mileage: 1000, InstitutionID: &f.inspectionID,
				WorkshopName: "Centro Demo", CurrencyID: f.currencyID,
			},
			wantErr: ErrInstitutionCategory,
		},
		{
			name: "moneda inexistente",
			input: models.AutoMaintenanceInput{
				PerformedAt: "2026-10-15", Mileage: 1000, WorkshopName: "Taller Demo", CurrencyID: 999999,
			},
			wantErr: ErrCurrencyNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.maintenance.Create(ctx, f.autoID, tc.input)
			if err == nil {
				t.Fatal("esperado error, got nil")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("esperado %v, got %v", tc.wantErr, err)
			}
		})
	}

	if _, err := f.maintenance.List(ctx, 999999); !errors.Is(err, ErrAutoNotFound) {
		t.Fatalf("auto inexistente: esperado ErrAutoNotFound, got %v", err)
	}
}

func TestAutoAttachmentServiceMultiplesPDFs(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()
	maintenance := f.createMaintenance(t, "2026-10-15", 42000)

	first, err := f.attachments.Save(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, AutoAttachmentUpload{
		FileName: "factura-demo.pdf",
		Reader:   strings.NewReader(testAutoPDF),
	})
	if err != nil {
		t.Fatalf("guardar primer PDF: %v", err)
	}
	second, err := f.attachments.Save(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, AutoAttachmentUpload{
		FileName: "orden-demo.pdf",
		Reader:   bytes.NewReader([]byte(testAutoPDF)),
	})
	if err != nil {
		t.Fatalf("guardar segundo PDF: %v", err)
	}

	items, err := f.attachments.ListByOwner(ctx, models.AutoAttachmentOwnerMaintenance, maintenance.ID)
	if err != nil {
		t.Fatalf("listar adjuntos: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("adjuntos: esperado 2, got %d", len(items))
	}

	att, path, err := f.attachments.Get(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, first.ID)
	if err != nil {
		t.Fatalf("obtener adjunto: %v", err)
	}
	if att.FileName != "factura-demo.pdf" {
		t.Errorf("file_name: esperado factura-demo.pdf, got %q", att.FileName)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.HasPrefix(string(data), "%PDF-") {
		t.Fatalf("leer archivo guardado: %v", err)
	}

	if err := f.attachments.Delete(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, second.ID); err != nil {
		t.Fatalf("eliminar adjunto: %v", err)
	}
	items, err = f.attachments.ListByOwner(ctx, models.AutoAttachmentOwnerMaintenance, maintenance.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("adjuntos después de eliminar: %v, %d", err, len(items))
	}
}

func TestAutoAttachmentServiceValidaciones(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()
	maintenance := f.createMaintenance(t, "2026-10-15", 42000)

	cases := []struct {
		name     string
		fileName string
		content  []byte
		wantErr  error
	}{
		{name: "extensión inválida", fileName: "factura.txt", content: []byte(testAutoPDF), wantErr: ErrAttachmentInvalid},
		{name: "vacío", fileName: "factura.pdf", content: []byte(""), wantErr: ErrAttachmentInvalid},
		{name: "sin firma PDF", fileName: "factura.pdf", content: []byte("no es un pdf"), wantErr: ErrAttachmentInvalid},
		{name: "demasiado grande", fileName: "factura.pdf", content: bytes.Repeat([]byte("a"), MaxAutoAttachmentBytes+1), wantErr: ErrAttachmentTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.attachments.Save(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, AutoAttachmentUpload{
				FileName: tc.fileName,
				Reader:   bytes.NewReader(tc.content),
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("esperado %v, got %v", tc.wantErr, err)
			}
		})
	}

	for i := 0; i < MaxAutoAttachmentsPerRecord; i++ {
		if _, err := f.attachments.Save(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, AutoAttachmentUpload{
			FileName: "demo.pdf",
			Reader:   strings.NewReader(testAutoPDF),
		}); err != nil {
			t.Fatalf("adjunto %d: %v", i+1, err)
		}
	}
	_, err := f.attachments.Save(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, AutoAttachmentUpload{
		FileName: "extra.pdf",
		Reader:   strings.NewReader(testAutoPDF),
	})
	if !errors.Is(err, ErrAttachmentLimit) {
		t.Fatalf("límite de adjuntos: esperado ErrAttachmentLimit, got %v", err)
	}
}

func TestAutoMaintenanceDeleteLimpiaArchivos(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()
	maintenance := f.createMaintenance(t, "2026-10-15", 42000)

	att, err := f.attachments.Save(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, AutoAttachmentUpload{
		FileName: "factura-demo.pdf",
		Reader:   strings.NewReader(testAutoPDF),
	})
	if err != nil {
		t.Fatalf("guardar PDF: %v", err)
	}
	_, path, err := f.attachments.Get(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, att.ID)
	if err != nil {
		t.Fatalf("obtener PDF: %v", err)
	}

	if err := f.maintenance.Delete(ctx, f.autoID, maintenance.ID); err != nil {
		t.Fatalf("eliminar mantenimiento: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("el archivo debería haber sido eliminado, err=%v", err)
	}
}

func TestAutoServiceDeleteLimpiaDirectorio(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()
	maintenance := f.createMaintenance(t, "2026-10-15", 42000)
	if _, err := f.attachments.Save(ctx, f.autoID, models.AutoAttachmentOwnerMaintenance, maintenance.ID, AutoAttachmentUpload{
		FileName: "factura-demo.pdf",
		Reader:   strings.NewReader(testAutoPDF),
	}); err != nil {
		t.Fatalf("guardar PDF: %v", err)
	}

	autoDir := filepath.Join(f.dataDir, "autos", fmt.Sprintf("%d", f.autoID))
	if _, err := os.Stat(autoDir); err != nil {
		t.Fatalf("el directorio del auto debería existir: %v", err)
	}
	if err := f.autoService.Delete(ctx, f.autoID); err != nil {
		t.Fatalf("eliminar auto: %v", err)
	}
	if _, err := os.Stat(autoDir); !os.IsNotExist(err) {
		t.Fatalf("el directorio del auto debería haber sido eliminado, err=%v", err)
	}
}

func TestAutoInspectionServiceCronogramaYValidaciones(t *testing.T) {
	f := newAutoRecordsFixture(t)
	ctx := context.Background()

	approved := f.createInspection(t, "2022-01-10", 2022, models.InspectionResultApproved)
	if approved.InspectionYear != 2022 {
		t.Fatalf("inspection_year: esperado 2022, got %d", approved.InspectionYear)
	}

	list, err := f.inspection.List(ctx, f.autoID)
	if err != nil {
		t.Fatalf("listar inspecciones: %v", err)
	}
	if list.Schedule.FirstDueYear != 2022 || list.Schedule.NextDueYear != 2023 {
		t.Fatalf("cronograma inesperado: %+v", list.Schedule)
	}
	if list.Schedule.Status != models.InspectionStatusOverdue {
		t.Errorf("status: esperado overdue, got %s", list.Schedule.Status)
	}

	f.createInspection(t, "2023-01-10", 2023, models.InspectionResultRejected)
	list, err = f.inspection.List(ctx, f.autoID)
	if err != nil {
		t.Fatalf("listar con rechazada: %v", err)
	}
	if list.Schedule.NextDueYear != 2023 {
		t.Errorf("una rechazada no debe avanzar el cronograma: got %d", list.Schedule.NextDueYear)
	}

	f.createInspection(t, "2023-02-10", 2023, models.InspectionResultApproved)
	list, err = f.inspection.List(ctx, f.autoID)
	if err != nil {
		t.Fatalf("listar con aprobada: %v", err)
	}
	if list.Schedule.NextDueYear != 2024 {
		t.Errorf("next_due_year: esperado 2024, got %d", list.Schedule.NextDueYear)
	}

	alerts, err := f.inspection.Alerts(ctx)
	if err != nil {
		t.Fatalf("listar alertas: %v", err)
	}
	if len(alerts) != 1 || alerts[0].AutoID != f.autoID {
		t.Fatalf("alertas inesperadas: %+v", alerts)
	}

	cases := []struct {
		name    string
		input   models.AutoInspectionInput
		wantErr error
	}{
		{
			name: "año anterior al mínimo",
			input: models.AutoInspectionInput{
				PerformedAt: "2022-01-10", InspectionYear: 2021, InstitutionID: f.inspectionID, Result: models.InspectionResultApproved,
			},
		},
		{
			name: "año posterior al permitido",
			input: models.AutoInspectionInput{
				PerformedAt: "2022-01-10", InspectionYear: 2024, InstitutionID: f.inspectionID, Result: models.InspectionResultApproved,
			},
		},
		{
			name: "resultado inválido",
			input: models.AutoInspectionInput{
				PerformedAt: "2022-01-10", InspectionYear: 2022, InstitutionID: f.inspectionID, Result: "pending",
			},
		},
		{
			name: "institución de categoría incorrecta",
			input: models.AutoInspectionInput{
				PerformedAt: "2022-01-10", InspectionYear: 2022, InstitutionID: f.workshopID, Result: models.InspectionResultApproved,
			},
			wantErr: ErrInstitutionCategory,
		},
		{
			name: "fecha futura",
			input: models.AutoInspectionInput{
				PerformedAt: "2026-12-01", InspectionYear: 2026, InstitutionID: f.inspectionID, Result: models.InspectionResultApproved,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.inspection.Create(ctx, f.autoID, tc.input)
			if err == nil {
				t.Fatal("esperado error, got nil")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("esperado %v, got %v", tc.wantErr, err)
			}
		})
	}

	if err := f.inspection.Delete(ctx, f.autoID, approved.ID); err != nil {
		t.Fatalf("eliminar inspección: %v", err)
	}
}
