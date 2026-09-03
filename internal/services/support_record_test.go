package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type supportRecordFixture struct {
	svc     *SupportRecordService
	closing *MonthClosingService
	childID int64
	catID   int64
}

func newSupportRecordFixture(t *testing.T) *supportRecordFixture {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	closingStorage := storage.NewMonthClosingStorage(database)
	closingSvc := NewMonthClosingService(closingStorage)

	dataDir := t.TempDir()
	svc := NewSupportRecordService(
		storage.NewSupportRecordStorage(database),
		closingStorage,
		storage.NewChildStorage(database),
		storage.NewPensionCategoryStorage(database),
		dataDir,
	)

	ctx := context.Background()
	childSvc := NewChildService(storage.NewChildStorage(database))
	child, err := childSvc.Create(ctx, "Juan", "Pérez", "2015-01-01", "")
	if err != nil {
		t.Fatalf("crear hijo: %v", err)
	}
	catSvc := NewPensionCategoryService(storage.NewPensionCategoryStorage(database))
	cat, err := catSvc.Create(ctx, "Colegio", "", false)
	if err != nil {
		t.Fatalf("crear categoría: %v", err)
	}

	return &supportRecordFixture{svc: svc, closing: closingSvc, childID: child.ID, catID: cat.ID}
}

func TestSupportRecordService_CreateValidation(t *testing.T) {
	f := newSupportRecordFixture(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		childID int64
		catID   int64
		year    int
		month   int
		amount  float64
		cur     string
		wantErr bool
	}{
		{"campos requeridos ok", f.childID, f.catID, 2026, 8, 1500, "NIO", false},
		{"monto cero", f.childID, f.catID, 2026, 8, 0, "NIO", true},
		{"monto negativo", f.childID, f.catID, 2026, 8, -100, "NIO", true},
		{"hijo inexistente", 9999, f.catID, 2026, 8, 1500, "NIO", true},
		{"categoría inexistente", f.childID, 9999, 2026, 8, 1500, "NIO", true},
		{"mes inválido", f.childID, f.catID, 2026, 13, 1500, "NIO", true},
		{"mes cero", f.childID, f.catID, 2026, 0, 1500, "NIO", true},
		{"año inválido", f.childID, f.catID, 1999, 8, 1500, "NIO", true},
		{"moneda default", f.childID, f.catID, 2026, 9, 1500, "", false},
		{"moneda lowercase normalizada", f.childID, f.catID, 2026, 10, 1500, "usd", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := f.svc.Create(ctx, tc.childID, tc.catID, tc.year, tc.month, tc.amount, tc.cur, "")
			if tc.wantErr && err == nil {
				t.Fatalf("se esperaba error pero no hubo")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("no se esperaba error, got: %v", err)
			}
			if err == nil {
				if rec.Status != "pending" {
					t.Fatalf("status debería ser pending, got=%s", rec.Status)
				}
				if rec.ChildName == "" || rec.CategoryName == "" {
					t.Fatalf("se esperaba child_name y category_name resueltos")
				}
			}
		})
	}
}

func TestSupportRecordService_DuplicateUnique(t *testing.T) {
	f := newSupportRecordFixture(t)
	ctx := context.Background()

	_, err := f.svc.Create(ctx, f.childID, f.catID, 2026, 8, 1500, "NIO", "")
	if err != nil {
		t.Fatalf("crear registro: %v", err)
	}
	_, err = f.svc.Create(ctx, f.childID, f.catID, 2026, 8, 1500, "NIO", "")
	if err == nil {
		t.Fatal("se esperaba error por duplicado de hijo+categoría+período")
	}
}

func TestSupportRecordService_StateFlow(t *testing.T) {
	f := newSupportRecordFixture(t)
	ctx := context.Background()

	rec, err := f.svc.Create(ctx, f.childID, f.catID, 2026, 8, 1500, "NIO", "")
	if err != nil {
		t.Fatalf("crear registro: %v", err)
	}

	// mark-paid
	paidAt := parseTime(t, "2026-08-15T10:00:00")
	origAmount := 40.0
	rate := 37.5
	rec, err = f.svc.MarkPaid(ctx, rec.ID, &paidAt, "bank_transfer", "REF-1", "Transferencia", &origAmount, "USD", &rate)
	if err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	if rec.Status != "paid" || rec.PaymentMethod == nil || *rec.PaymentMethod != "bank_transfer" {
		t.Fatalf("datos de pago incorrectos: %+v", rec)
	}
	if rec.PaidAt == nil {
		t.Fatal("se esperaba paid_at")
	}
	if rec.OriginalAmount == nil || *rec.OriginalAmount != 40.0 || *rec.ExchangeRate != 37.5 {
		t.Fatalf("conversión no persistida: %+v", rec)
	}

	// mark-pending limpia campos de pago y conversión
	rec, err = f.svc.MarkPending(ctx, rec.ID)
	if err != nil {
		t.Fatalf("mark pending: %v", err)
	}
	if rec.Status != "pending" {
		t.Fatalf("status debería ser pending, got=%s", rec.Status)
	}
	if rec.PaidAt != nil || rec.PaymentMethod != nil || rec.OriginalAmount != nil || rec.ExchangeRate != nil {
		t.Fatalf("mark pending debería limpiar campos de pago: %+v", rec)
	}

	// mark-rejected requiere motivo
	_, err = f.svc.MarkRejected(ctx, rec.ID, "   ")
	if err == nil {
		t.Fatal("se esperaba error por motivo vacío")
	}
	rec, err = f.svc.MarkRejected(ctx, rec.ID, "Comprobante inválido")
	if err != nil {
		t.Fatalf("mark rejected: %v", err)
	}
	if rec.Status != "rejected" || rec.Notes == nil || *rec.Notes != "Comprobante inválido" {
		t.Fatalf("datos de rechazo incorrectos: %+v", rec)
	}
}

func TestSupportRecordService_MonthClosedBlocksMutations(t *testing.T) {
	f := newSupportRecordFixture(t)
	ctx := context.Background()

	rec, err := f.svc.Create(ctx, f.childID, f.catID, 2026, 8, 1500, "NIO", "")
	if err != nil {
		t.Fatalf("crear registro: %v", err)
	}
	if _, err := f.closing.Close(ctx, 2026, 8); err != nil {
		t.Fatalf("cerrar mes: %v", err)
	}

	// Crear nuevo registro en mes cerrado
	if _, err := f.svc.Create(ctx, f.childID, f.catID, 2026, 8, 900, "NIO", ""); err == nil {
		t.Fatal("se esperaba error al crear en mes cerrado")
	}

	// mark-paid en mes cerrado
	paidAt := parseTime(t, "2026-08-16T10:00:00")
	if _, err := f.svc.MarkPaid(ctx, rec.ID, &paidAt, "cash", "", "", nil, "", nil); err == nil {
		t.Fatal("se esperaba error al pagar en mes cerrado")
	}

	// mark-pending en mes cerrado
	if _, err := f.svc.MarkPending(ctx, rec.ID); err == nil {
		t.Fatal("se esperaba error al marcar pendiente en mes cerrado")
	}

	// mark-rejected en mes cerrado
	if _, err := f.svc.MarkRejected(ctx, rec.ID, "motivo"); err == nil {
		t.Fatal("se esperaba error al rechazar en mes cerrado")
	}

	// update en mes cerrado
	if _, err := f.svc.Update(ctx, rec.ID, 1800, f.catID, nil); err == nil {
		t.Fatal("se esperaba error al editar en mes cerrado")
	}

	// Reabrir y verificar que la lista sigue accesible
	if err := f.closing.Reopen(ctx, 2026, 8); err != nil {
		t.Fatalf("reabrir mes: %v", err)
	}
	records, err := f.svc.List(ctx, 2026, 8, 0)
	if err != nil {
		t.Fatalf("list registros: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("se esperaba 1 registro, count=%d", len(records))
	}
}

func TestSupportRecordService_SaveProof(t *testing.T) {
	f := newSupportRecordFixture(t)
	ctx := context.Background()

	rec, err := f.svc.Create(ctx, f.childID, f.catID, 2026, 8, 1500, "NIO", "")
	if err != nil {
		t.Fatalf("crear registro: %v", err)
	}

	content := []byte("%PDF-1.4 fake")
	rec, err = f.svc.SaveProof(ctx, rec.ID, "comprobante.pdf", content)
	if err != nil {
		t.Fatalf("save proof: %v", err)
	}
	if rec.ProofFileName == nil || *rec.ProofFileName != "comprobante.pdf" {
		t.Fatalf("proof_file_name no guardado: %+v", rec)
	}

	filePath, fileName, err := f.svc.ProofPath(ctx, rec.ID)
	if err != nil {
		t.Fatalf("proof path: %v", err)
	}
	if fileName != "comprobante.pdf" {
		t.Fatalf("nombre de descarga incorrecto: %s", fileName)
	}
	if filepath.Base(filePath) == "" {
		t.Fatal("ruta de archivo inválida")
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("el archivo debería existir en disco: %v", err)
	}

	// Formato no soportado
	if _, err := f.svc.SaveProof(ctx, rec.ID, "malware.exe", content); err == nil {
		t.Fatal("se esperaba error por extensión no soportada")
	}

	// Comprobante en mes cerrado
	if _, err := f.closing.Close(ctx, 2026, 8); err != nil {
		t.Fatalf("cerrar mes: %v", err)
	}
	if _, err := f.svc.SaveProof(ctx, rec.ID, "otro.pdf", content); err == nil {
		t.Fatal("se esperaba error al subir comprobante en mes cerrado")
	}
}

func parseTime(t *testing.T, s string) (out time.Time) {
	t.Helper()
	out, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		t.Fatalf("parse fecha: %v", err)
	}
	return out
}
