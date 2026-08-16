package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type DocumentService struct {
	serviceStorage  *storage.ServiceStorage
	billStorage     *storage.BillStorage
	instStorage     *storage.InstitutionStorage
}

func NewDocumentService(serviceStorage *storage.ServiceStorage, billStorage *storage.BillStorage, instStorage *storage.InstitutionStorage) *DocumentService {
	return &DocumentService{
		serviceStorage:  serviceStorage,
		billStorage:     billStorage,
		instStorage:     instStorage,
	}
}

var allowedMimeTypes = map[string]bool{
	"application/pdf":      true,
	"image/png":            true,
	"image/jpeg":           true,
	"application/octet-stream": true,
}

var allowedExtensions = map[string]bool{
	".pdf":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

func (s *DocumentService) UploadAndAnalyze(ctx context.Context, serviceID int64, file multipart.File, header *multipart.FileHeader) (*analyzers.ExtractedBill, string, error) {
	svc, err := s.serviceStorage.GetByID(ctx, serviceID)
	if err != nil {
		return nil, "", fmt.Errorf("obtener servicio: %w", err)
	}
	if svc == nil {
		return nil, "", fmt.Errorf("servicio no encontrado")
	}
	if svc.InstitutionID == nil {
		return nil, "", fmt.Errorf("el servicio no tiene institución asociada")
	}
	if svc.InstitutionAnalyzerID == nil {
		return nil, "", fmt.Errorf("el servicio no tiene analizador configurado")
	}

	instAnalyzers, err := s.instStorage.GetAnalyzers(ctx, *svc.InstitutionID)
	if err != nil {
		return nil, "", fmt.Errorf("obtener analyzers de institución: %w", err)
	}

	var analyzerID string
	for _, ia := range instAnalyzers {
		if ia.ID == *svc.InstitutionAnalyzerID {
			analyzerID = ia.AnalyzerID
			break
		}
	}
	if analyzerID == "" {
		return nil, "", fmt.Errorf("analizador no encontrado para este servicio")
	}

	analyzer, ok := analyzers.Get(analyzerID)
	if !ok {
		return nil, "", fmt.Errorf("analizador %q no está registrado", analyzerID)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		return nil, "", fmt.Errorf("formato de archivo no soportado. Use PDF, PNG o JPG")
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !allowedMimeTypes[mimeType] && !allowedExtensions[ext] {
		return nil, "", fmt.Errorf("tipo de archivo no permitido")
	}

	result, err := analyzer.Analyze(file, mimeType)
	if err != nil {
		return nil, "", fmt.Errorf("error al analizar documento: %w", err)
	}

	return result, analyzerID, nil
}

func (s *DocumentService) CreateBillFromExtracted(ctx context.Context, serviceID int64, extracted *analyzers.ExtractedBill) (*models.Bill, bool, error) {
	existing, err := s.billStorage.FindByServicePeriod(ctx, serviceID, extracted.Year, extracted.Month)
	if err != nil {
		return nil, false, fmt.Errorf("verificar factura existente: %w", err)
	}
	if existing != nil {
		err := s.billStorage.UpdateFromExtracted(ctx, existing.ID, extracted.Amount, extracted.InvoiceNumber)
		if err != nil {
			return nil, false, fmt.Errorf("actualizando factura: %w", err)
		}
		existing.Amount = extracted.Amount
		existing.InvoiceNumber = extracted.InvoiceNumber
		return existing, true, nil
	}

	bill := &models.Bill{
		ServiceID:     serviceID,
		Year:          extracted.Year,
		Month:         extracted.Month,
		Amount:        extracted.Amount,
		InvoiceNumber: extracted.InvoiceNumber,
		Status:        "pending",
	}
	if extracted.DueDate != nil {
		bill.UpdatedAt = *extracted.DueDate
	}

	b, err := s.billStorage.Create(ctx, bill)
	if err != nil {
		return nil, false, err
	}
	return b, false, nil
}

func (s *DocumentService) GetAnalyzerOptions(ctx context.Context, institutionID int64) ([]map[string]interface{}, error) {
	instAnalyzers, err := s.instStorage.GetAnalyzers(ctx, institutionID)
	if err != nil {
		return nil, err
	}

	var options []map[string]interface{}
	for _, ia := range instAnalyzers {
		analyzer, ok := analyzers.Get(ia.AnalyzerID)
		if !ok {
			continue
		}
		info := analyzer.Info()
		options = append(options, map[string]interface{}{
			"id":              ia.ID,
			"institution_id":  ia.InstitutionID,
			"analyzer_id":     ia.AnalyzerID,
			"analyzer_name":   info.Name,
		})
	}
	return options, nil
}

func validateDocumentFile(header *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		return fmt.Errorf("formato no soportado: %s", ext)
	}
	if header.Size > 10*1024*1024 {
		return fmt.Errorf("archivo demasiado grande (máx 10MB)")
	}
	return nil
}

func (s *DocumentService) ValidateAndPrepare(file multipart.File, header *multipart.FileHeader) (multipart.File, error) {
	if err := validateDocumentFile(header); err != nil {
		return nil, err
	}
	return file, nil
}

func (s *DocumentService) GetAvailableAnalyzers() []analyzers.AnalyzerInfo {
	return analyzers.List()
}
