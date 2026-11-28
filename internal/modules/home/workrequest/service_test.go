package workrequest

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	homemaintenance "github.com/paulomcnally/p40la-ihost/internal/modules/home/maintenance"
)

type fakeMaintenance struct {
	db     *sql.DB
	inputs []homemaintenance.Input
	nextID int64
}

func (f *fakeMaintenance) Create(_ context.Context, homeID int64, input homemaintenance.Input) (*homemaintenance.Maintenance, error) {
	f.inputs = append(f.inputs, input)
	res, err := f.db.Exec(`
		INSERT INTO home_maintenances (home_id, performed_at, kind, area, description)
		VALUES (?, '2026-11-27', ?, ?, ?)
	`, homeID, input.Kind, input.Area, input.Description)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	f.nextID = id
	return &homemaintenance.Maintenance{ID: id, HomeID: homeID, Kind: input.Kind, Area: input.Area, Description: input.Description}, nil
}

func (f *fakeMaintenance) Delete(context.Context, int64, int64) error { return nil }

func newTestService(t *testing.T) (*Service, *sql.DB, int64, *fakeMaintenance) {
	t.Helper()
	dir := t.TempDir()
	database, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "workrequests.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	migrationsDir, err := filepath.Abs("../../../../migrations")
	if err != nil {
		t.Fatalf("ruta de migraciones: %v", err)
	}
	if err := db.Migrate(database, migrationsDir); err != nil {
		t.Fatalf("migrar: %v", err)
	}
	res, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Demo')`)
	if err != nil {
		t.Fatalf("insertar vivienda: %v", err)
	}
	homeID, _ := res.LastInsertId()

	fake := &fakeMaintenance{db: database}
	service := NewService(NewStorage(database), fake, dir)
	service.SetClock(func() time.Time { return time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC) })
	return service, database, homeID, fake
}

func validInput() Input {
	return Input{
		Title:       "Filtración de agua en el techo",
		Description: "Gotea cuando llueve, sobre el cuarto principal.",
		Category:    CategoryRepair,
		Priority:    PriorityHigh,
		ReportedAt:  "2026-11-26",
	}
}

func jpegData() []byte {
	return append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte("demo"), 16)...)
}

func pngData() []byte {
	return append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("demo"), 16)...)
}

func mp4Data() []byte {
	return append([]byte("\x00\x00\x00\x18ftypmp42"), bytes.Repeat([]byte("demo"), 16)...)
}

func webmData() []byte {
	return append([]byte{0x1A, 0x45, 0xDF, 0xA3}, []byte("webmdemo")...)
}

func TestCreateReportStartsReportedWithoutCosts(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	created, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear reporte: %v", err)
	}
	if created.Status != StatusReported {
		t.Fatalf("estado inicial = %q", created.Status)
	}
	if created.Category != CategoryRepair || created.Priority != PriorityHigh {
		t.Fatalf("categoría/prioridad = %q/%q", created.Category, created.Priority)
	}
	if created.ReportedAt != "2026-11-26" {
		t.Fatalf("fecha = %q", created.ReportedAt)
	}
	if created.MaintenanceID != nil || created.ConvertedAt != nil {
		t.Fatalf("reporte nuevo no debería estar convertido: %+v", created)
	}
	if created.AttachmentCount != 0 || len(created.Attachments) != 0 {
		t.Fatalf("adjuntos iniciales = %d/%d", created.AttachmentCount, len(created.Attachments))
	}

	list, err := service.List(ctx, homeID, Filter{})
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(list.Items) != 1 || list.PendingCount != 1 || list.Truncated {
		t.Fatalf("listado = %+v", list)
	}
}

func TestCreateReportValidation(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	cases := []struct {
		name string
		edit func(*Input)
		code string
	}{
		{"título vacío", func(in *Input) { in.Title = "   " }, "title_required"},
		{"categoría inválida", func(in *Input) { in.Category = "broken" }, "invalid_category"},
		{"prioridad inválida", func(in *Input) { in.Priority = "urgent" }, "invalid_priority"},
		{"fecha inválida", func(in *Input) { in.ReportedAt = "26/11/2026" }, "invalid_date"},
		{"fecha futura", func(in *Input) { in.ReportedAt = "2026-11-28" }, "future_date"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validInput()
			tc.edit(&input)
			_, err := service.Create(ctx, homeID, CreateInput{Input: input})
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Code != tc.code {
				t.Fatalf("error = %v, want code %s", err, tc.code)
			}
		})
	}
}

func TestUpdateKeepsStatusAndListFilters(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	first, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	secondInput := validInput()
	secondInput.Title = "Agregar canaleta para lluvia"
	secondInput.Category = CategoryImprovement
	secondInput.Priority = PriorityNormal
	secondInput.ReportedAt = "2026-11-27"
	second, err := service.Create(ctx, homeID, CreateInput{Input: secondInput})
	if err != nil {
		t.Fatalf("crear segunda: %v", err)
	}

	update := validInput()
	update.Title = "Filtración corregida en revisión"
	update.Category = CategoryMaintenance
	update.Priority = PriorityLow
	updated, err := service.Update(ctx, homeID, first.ID, update)
	if err != nil {
		t.Fatalf("editar: %v", err)
	}
	if updated.Status != StatusReported {
		t.Fatalf("editar cambió el estado: %q", updated.Status)
	}
	if updated.Title != update.Title || updated.Category != CategoryMaintenance || updated.Priority != PriorityLow {
		t.Fatalf("edición no aplicada: %+v", updated)
	}

	byCategory, err := service.List(ctx, homeID, Filter{Category: CategoryImprovement})
	if err != nil {
		t.Fatalf("listar por categoría: %v", err)
	}
	if len(byCategory.Items) != 1 || byCategory.Items[0].ID != second.ID {
		t.Fatalf("filtro por categoría = %+v", byCategory.Items)
	}
	byStatus, err := service.List(ctx, homeID, Filter{Status: StatusConverted})
	if err != nil {
		t.Fatalf("listar por estado: %v", err)
	}
	if len(byStatus.Items) != 0 || byStatus.PendingCount != 2 {
		t.Fatalf("filtro por estado = %+v", byStatus)
	}
}

func TestDismissAndReactivate(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	dismissed, err := service.SetStatus(ctx, homeID, item.ID, StatusInput{Status: StatusDismissed})
	if err != nil {
		t.Fatalf("descartar: %v", err)
	}
	if dismissed.Status != StatusDismissed {
		t.Fatalf("estado = %q", dismissed.Status)
	}
	reactivated, err := service.SetStatus(ctx, homeID, item.ID, StatusInput{Status: StatusReported})
	if err != nil {
		t.Fatalf("reactivar: %v", err)
	}
	if reactivated.Status != StatusReported {
		t.Fatalf("estado = %q", reactivated.Status)
	}
	if _, err := service.SetStatus(ctx, homeID, item.ID, StatusInput{Status: StatusConverted}); err == nil {
		t.Fatal("estado convertido no debería aceptarse desde status")
	}
}

func TestConvertCreatesMaintenanceAndLinks(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, fake := newTestService(t)

	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	result, err := service.Convert(ctx, homeID, item.ID, ConvertInput{})
	if err != nil {
		t.Fatalf("convertir: %v", err)
	}
	if len(fake.inputs) != 1 {
		t.Fatalf("llamadas a maintenance = %d", len(fake.inputs))
	}
	input := fake.inputs[0]
	if input.Kind != homemaintenance.KindCorrective || input.Area != homemaintenance.AreaOther {
		t.Fatalf("kind/area por defecto = %q/%q", input.Kind, input.Area)
	}
	if input.PerformedAt != "2026-11-27" {
		t.Fatalf("performed_at = %q", input.PerformedAt)
	}
	if !strings.Contains(input.Description, item.Title) || !strings.Contains(input.Description, item.Description) {
		t.Fatalf("descripción convertida = %q", input.Description)
	}
	if result.MaintenanceID == 0 || result.WorkRequest.Status != StatusConverted {
		t.Fatalf("resultado = %+v", result)
	}
	if result.WorkRequest.MaintenanceID == nil || *result.WorkRequest.MaintenanceID != result.MaintenanceID {
		t.Fatalf("maintenance_id no enlazado: %+v", result.WorkRequest)
	}
	if result.WorkRequest.ConvertedAt == nil {
		t.Fatal("converted_at vacío")
	}

	if _, err := service.Convert(ctx, homeID, item.ID, ConvertInput{}); !errors.Is(err, ErrAlreadyConverted) {
		t.Fatalf("segunda conversión = %v, want ErrAlreadyConverted", err)
	}
}

func TestConvertRejectsInvalidKindAndDismissed(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	if _, err := service.Convert(ctx, homeID, item.ID, ConvertInput{Kind: "unknown"}); err == nil {
		t.Fatal("kind inválido debería fallar")
	}
	if _, err := service.Convert(ctx, homeID, item.ID, ConvertInput{Area: "unknown"}); err == nil {
		t.Fatal("area inválida debería fallar")
	}
	if _, err := service.SetStatus(ctx, homeID, item.ID, StatusInput{Status: StatusDismissed}); err != nil {
		t.Fatalf("descartar: %v", err)
	}
	if _, err := service.Convert(ctx, homeID, item.ID, ConvertInput{}); err == nil {
		t.Fatal("un reporte descartado no debería convertirse")
	}
}

func TestAttachmentsLifecycle(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	photo, err := service.AddAttachment(ctx, homeID, item.ID, MediaImage, "foto-demo.jpg", bytes.NewReader(jpegData()))
	if err != nil {
		t.Fatalf("subir foto: %v", err)
	}
	if photo.MimeType != "image/jpeg" || photo.SizeBytes == 0 || photo.FileName != "foto-demo.jpg" {
		t.Fatalf("adjunto = %+v", photo)
	}
	if _, err := os.Stat(service.attachmentFilePath(item.ID, photo.ID, photo.FileName)); err != nil {
		t.Fatalf("archivo no guardado: %v", err)
	}

	video, err := service.AddAttachment(ctx, homeID, item.ID, MediaVideo, "video-demo.mp4", bytes.NewReader(mp4Data()))
	if err != nil {
		t.Fatalf("subir video: %v", err)
	}
	if video.MimeType != "video/mp4" {
		t.Fatalf("mime de video = %q", video.MimeType)
	}

	detail, err := service.Get(ctx, homeID, item.ID)
	if err != nil {
		t.Fatalf("detalle: %v", err)
	}
	if detail.AttachmentCount != 2 || len(detail.Attachments) != 2 {
		t.Fatalf("evidencia en detalle = %d/%d", detail.AttachmentCount, len(detail.Attachments))
	}

	attachment, path, err := service.AttachmentPath(ctx, homeID, item.ID, photo.ID)
	if err != nil || attachment.ID != photo.ID || path == "" {
		t.Fatalf("ruta de adjunto: %+v %q %v", attachment, path, err)
	}
	if err := service.DeleteAttachment(ctx, homeID, item.ID, video.ID); err != nil {
		t.Fatalf("borrar adjunto: %v", err)
	}
	if _, err := os.Stat(service.attachmentFilePath(item.ID, video.ID, video.FileName)); !os.IsNotExist(err) {
		t.Fatalf("archivo del video sigue en disco: %v", err)
	}
}

func TestAttachmentValidationAndLimits(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}

	cases := []struct {
		name      string
		mediaType string
		fileName  string
		data      []byte
		code      string
	}{
		{"texto no permitido", MediaImage, "nota.txt", []byte("hola"), "invalid_file"},
		{"tipo declarado no coincide", MediaVideo, "foto-demo.jpg", jpegData(), "invalid_file"},
		{"archivo vacío", MediaImage, "vacio.jpg", nil, "invalid_file"},
		{"demasiado grande", MediaImage, "grande.jpg", append(jpegData(), bytes.Repeat([]byte("x"), MaxAttachmentBytes)...), "request_too_large"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.AddAttachment(ctx, homeID, item.ID, tc.mediaType, tc.fileName, bytes.NewReader(tc.data))
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Code != tc.code {
				t.Fatalf("error = %v, want code %s", err, tc.code)
			}
		})
	}

	for index := 0; index < MaxAttachmentsPerRequest; index++ {
		_, err := service.AddAttachment(ctx, homeID, item.ID, MediaImage, "foto-demo.jpg", bytes.NewReader(jpegData()))
		if err != nil {
			t.Fatalf("adjunto %d: %v", index, err)
		}
	}
	_, err = service.AddAttachment(ctx, homeID, item.ID, MediaImage, "foto-demo.jpg", bytes.NewReader(jpegData()))
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Code != "attachment_limit" {
		t.Fatalf("límite de adjuntos = %v", err)
	}
}

func TestDifferentHomesCannotAccessEvidence(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)

	res, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Demo Dos')`)
	if err != nil {
		t.Fatalf("insertar segunda vivienda: %v", err)
	}
	otherHomeID, _ := res.LastInsertId()

	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	attachment, err := service.AddAttachment(ctx, homeID, item.ID, MediaImage, "foto-demo.jpg", bytes.NewReader(jpegData()))
	if err != nil {
		t.Fatalf("subir: %v", err)
	}
	if _, err := service.Get(ctx, otherHomeID, item.ID); !errors.Is(err, ErrWorkRequestNotFound) {
		t.Fatalf("detalle de otra vivienda = %v", err)
	}
	if _, _, err := service.AttachmentPath(ctx, otherHomeID, item.ID, attachment.ID); !errors.Is(err, ErrWorkRequestNotFound) {
		t.Fatalf("adjunto de otra vivienda = %v", err)
	}
}

func TestDeleteReportCascadesEvidenceFiles(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)

	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	attachment, err := service.AddAttachment(ctx, homeID, item.ID, MediaImage, "foto-demo.jpg", bytes.NewReader(jpegData()))
	if err != nil {
		t.Fatalf("subir: %v", err)
	}
	dir := service.attachmentDir(item.ID)
	if err := service.Delete(ctx, homeID, item.ID); err != nil {
		t.Fatalf("eliminar reporte: %v", err)
	}
	if _, err := service.Get(ctx, homeID, item.ID); !errors.Is(err, ErrWorkRequestNotFound) {
		t.Fatalf("reporte sigue existiendo: %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_work_request_attachments WHERE work_request_id = ?`, item.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("adjuntos en cascada = %d", count)
	}
	if _, err := os.Stat(service.attachmentFilePath(item.ID, attachment.ID, attachment.FileName)); !os.IsNotExist(err) {
		t.Fatalf("archivo de evidencia sigue en disco: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("directorio de evidencia sigue en disco: %v", err)
	}
}

func TestDetectFileKinds(t *testing.T) {
	cases := []struct {
		data      []byte
		mediaType string
		mimeType  string
	}{
		{jpegData(), MediaImage, "image/jpeg"},
		{pngData(), MediaImage, "image/png"},
		{append([]byte("RIFF\x00\x00\x00\x00WEBP"), []byte("demo")...), MediaImage, "image/webp"},
		{append([]byte("\x00\x00\x00\x18ftypheic"), []byte("demo")...), MediaImage, "image/heic"},
		{append([]byte("\x00\x00\x00\x18ftypqt  "), []byte("demo")...), MediaVideo, "video/quicktime"},
		{mp4Data(), MediaVideo, "video/mp4"},
		{webmData(), MediaVideo, "video/webm"},
	}
	for _, tc := range cases {
		got := detectFileKind(tc.data)
		if got.mediaType != tc.mediaType || got.mimeType != tc.mimeType {
			t.Fatalf("detect(%q) = %+v", tc.mimeType, got)
		}
	}
	if got := detectFileKind([]byte("%PDF-1.7")); got.mimeType != "" {
		t.Fatalf("PDF no debería aceptarse: %+v", got)
	}
}
