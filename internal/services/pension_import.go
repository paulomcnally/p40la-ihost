package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// PensionImportService orquesta el importador mensual (SPEC-119).
type PensionImportService struct {
	imports    *storage.PensionImportStorage
	categories *storage.PensionCategoryStorage
	children   *storage.ChildStorage
	records    *storage.SupportRecordStorage
	closings   *storage.MonthClosingStorage
}

// NewPensionImportService crea un nuevo PensionImportService.
func NewPensionImportService(
	imports *storage.PensionImportStorage,
	categories *storage.PensionCategoryStorage,
	children *storage.ChildStorage,
	records *storage.SupportRecordStorage,
	closings *storage.MonthClosingStorage,
) *PensionImportService {
	return &PensionImportService{
		imports:    imports,
		categories: categories,
		children:   children,
		records:    records,
		closings:   closings,
	}
}

// PensionImportPreviewInput es la entrada del preview.
type PensionImportPreviewInput struct {
	Filename string
	MIME     string
	Content  []byte
	Text     string
	Year     int
	Month    int
	ChildID  int64
}

// ImportError describe un error de negocio del importador.
type ImportError struct {
	Code       string
	Message    string
	Validation *models.PensionImportValidation
}

func (e *ImportError) Error() string { return e.Message }

// Preview parsea el documento, sugiere categorías y valida el lote sin escribir registros.
func (s *PensionImportService) Preview(ctx context.Context, in PensionImportPreviewInput) (*models.PensionImportPreview, error) {
	doc, err := s.parseInput(in)
	if err != nil {
		return nil, err
	}

	year, month := in.Year, in.Month
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		year, month = 0, 0
	}
	var documentPeriod *models.PensionImportPeriod
	periodMismatch := false
	if doc.Year > 0 && doc.Month > 0 {
		documentPeriod = &models.PensionImportPeriod{Year: doc.Year, Month: doc.Month}
		if year > 0 && (year != doc.Year || month != doc.Month) {
			periodMismatch = true
		}
		// El período del documento manda: evita cargar en el mes equivocado.
		year, month = doc.Year, doc.Month
	}
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		return nil, &ImportError{Code: "missing_period", Message: "No se pudo detectar el período del documento."}
	}

	categories, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}
	keywords, err := s.imports.ListKeywords(ctx)
	if err != nil {
		return nil, err
	}
	children, err := s.children.List(ctx)
	if err != nil {
		return nil, err
	}

	childID := in.ChildID
	if childID == 0 && len(children) == 1 {
		childID = children[0].ID
	}

	previous := map[int64]float64{}
	existing := map[int64]float64{}
	if childID > 0 {
		py, pm := importPreviousPeriod(year, month)
		prevRecords, err := s.records.ListByFilters(ctx, py, pm, childID)
		if err == nil {
			for _, r := range prevRecords {
				previous[r.PensionCategoryID] = r.Amount
			}
		}
		currentRecords, err := s.records.ListByFilters(ctx, year, month, childID)
		if err == nil {
			for _, r := range currentRecords {
				existing[r.PensionCategoryID] = r.Amount
			}
		}
	}

	categoryByID := make(map[int64]models.PensionCategory, len(categories))
	for _, c := range categories {
		categoryByID[c.ID] = c
	}
	categoryByName := make(map[string]int64, len(categories))
	for _, c := range categories {
		categoryByName[normalizeText(c.Name)] = c.ID
	}

	rows := make([]models.PensionImportRow, 0, len(doc.Lines))
	for _, line := range doc.Lines {
		row := models.PensionImportRow{
			LineNo:           line.LineNo,
			RawText:          line.RawText,
			Concept:          line.Concept,
			AmountTotal:      line.AmountTotal,
			Currency:         line.Currency,
			ValidationStatus: "ok",
			Action:           "create",
		}

		var match MatchResult
		if line.Category != "" {
			if id, ok := categoryByName[normalizeText(line.Category)]; ok {
				match = MatchResult{CategoryID: &id, Confidence: 1, Reason: "name"}
			}
		}
		if match.CategoryID == nil {
			match = matchCategory(line.Concept, categories, keywords)
		}

		if match.Mixed {
			row.ValidationStatus = "invalid"
			row.ValidationCode = "mixed_concept"
			row.Message = "La línea mezcla conceptos de dos categorías distintas."
			row.SharePercent = 50
			if row.Currency == "" {
				row.Currency = "NIO"
			}
			rows = append(rows, row)
			continue
		}
		if match.CategoryID == nil {
			row.ValidationStatus = "invalid"
			row.ValidationCode = "missing_category"
			row.Message = "No se encontró una categoría para el concepto."
			row.SharePercent = 50
			if row.Currency == "" {
				row.Currency = "NIO"
			}
			rows = append(rows, row)
			continue
		}

		category := categoryByID[*match.CategoryID]
		row.SuggestedCategoryID = match.CategoryID
		row.ChosenCategoryID = match.CategoryID
		row.Confidence = match.Confidence
		row.MatchReason = match.Reason
		row.MatchedKeyword = match.Keyword
		row.SharePercent = category.SharePercent
		if row.Currency == "" {
			row.Currency = category.Currency
		}
		if childID > 0 {
			id := childID
			row.ChildID = &id
		}

		row.AmountToRecord, row.RoundingAdjustment = roundShare(line.AmountTotal, category.SharePercent)
		row.CategoryName = category.Name

		if existingAmount, ok := existing[category.ID]; ok {
			v := existingAmount
			row.ExistingAmount = &v
		}
		if prev, ok := previous[category.ID]; ok {
			p := prev
			row.PreviousAmount = &p
			if prev != 0 {
				v := (row.AmountToRecord - prev) / prev * 100
				row.VariationPct = &v
			}
		}

		if line.AmountTotal <= 0 {
			row.ValidationStatus = "invalid"
			row.ValidationCode = "invalid_amount"
			row.Message = "El monto debe ser mayor a cero."
		} else if line.Currency != "" && !strings.EqualFold(line.Currency, category.Currency) {
			row.ValidationStatus = "invalid"
			row.ValidationCode = "currency_mismatch"
			row.Message = fmt.Sprintf("La moneda del archivo (%s) no coincide con la de la categoría (%s).", line.Currency, category.Currency)
		}

		rows = append(rows, row)
	}

	// Duplicados dentro del archivo.
	seen := map[string]int{}
	for i := range rows {
		if rows[i].ChosenCategoryID == nil {
			continue
		}
		key := fmt.Sprintf("%d-%d", rows[i].ChosenCategoryID, rows[i].LineNo)
		_ = key
		dupKey := fmt.Sprintf("%v", *rows[i].ChosenCategoryID)
		if first, ok := seen[dupKey]; ok {
			rows[i].ValidationStatus = "invalid"
			rows[i].ValidationCode = "duplicate_in_file"
			rows[i].Message = fmt.Sprintf("Duplica la línea %d para la misma categoría.", first)
		} else {
			seen[dupKey] = rows[i].LineNo
		}
	}

	// Totales.
	validation := models.PensionImportValidation{Status: "ok", Errors: []models.ImportValidationError{}}
	if doc.DeclaredTotal != nil {
		validation.DeclaredReimbursableTotal = *doc.DeclaredTotal
	}
	var exactShared, roundedShared, grand float64
	for _, r := range rows {
		if r.ChosenCategoryID == nil {
			continue
		}
		grand += r.AmountToRecord
		if r.SharePercent == 50 {
			exactShared += r.AmountTotal * 0.5
			roundedShared += r.AmountToRecord
		}
	}
	validation.ComputedReimbursableTotal = roundedShared
	validation.ComputedGrandTotal = grand

	if doc.DeclaredTotal != nil {
		minTotal := math.Min(exactShared, roundedShared)
		maxTotal := math.Max(exactShared, roundedShared)
		if *doc.DeclaredTotal < minTotal-0.0001 || *doc.DeclaredTotal > maxTotal+0.0001 {
			validation.Status = "invalid"
			validation.Errors = append(validation.Errors, models.ImportValidationError{
				Code: "total_mismatch",
				Message: fmt.Sprintf("El total declarado (%.2f) no coincide con la suma calculada (%.2f).",
					*doc.DeclaredTotal, roundedShared),
			})
		}
	} else {
		doc.Warnings = append(doc.Warnings, "sin línea de control")
	}

	for _, r := range rows {
		if r.ValidationStatus == "invalid" {
			validation.Status = "invalid"
			validation.Errors = append(validation.Errors, models.ImportValidationError{
				LineNo:  r.LineNo,
				Code:    r.ValidationCode,
				Message: r.Message,
			})
		}
	}

	hashInput := in.Content
	if len(hashInput) == 0 {
		hashInput = []byte(in.Text)
	}
	sum := sha256.Sum256(hashInput)
	fileHash := hex.EncodeToString(sum[:])

	batchID, err := s.imports.CreateBatch(ctx, in.Filename, fileHash, year, month, len(rows))
	if err != nil {
		return nil, err
	}
	if err := s.imports.UpdateBatchTotals(ctx, batchID, validation.DeclaredReimbursableTotal, validation.ComputedReimbursableTotal); err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].BatchID = batchID
	}
	if err := s.imports.SaveRows(ctx, batchID, rows); err != nil {
		return nil, err
	}

	controlLines := make([]models.PensionImportControlLine, 0, len(doc.ControlLines))
	for _, c := range doc.ControlLines {
		controlLines = append(controlLines, models.PensionImportControlLine{
			LineNo: c.LineNo, RawText: c.RawText, Amount: c.Amount,
		})
	}

	return &models.PensionImportPreview{
		BatchID:        batchID,
		Period:         models.PensionImportPeriod{Year: year, Month: month},
		DocumentPeriod: documentPeriod,
		PeriodMismatch: periodMismatch,
		ControlLines:   controlLines,
		Source:         doc.Source,
		Filename:       in.Filename,
		Validation:     validation,
		Warnings:       doc.Warnings,
		Lines:          rows,
	}, nil
}

// Confirm crea los registros del lote de forma atómica (todo o nada).
func (s *PensionImportService) Confirm(ctx context.Context, req models.PensionImportConfirmRequest) (*models.PensionImportConfirmResult, error) {
	batch, err := s.imports.GetBatch(ctx, req.BatchID)
	if err != nil {
		return nil, err
	}
	if batch == nil {
		return nil, &ImportError{Code: "batch_not_found", Message: "Lote de importación no encontrado."}
	}
	if batch.Status == "confirmed" {
		return nil, &ImportError{Code: "already_confirmed", Message: "El lote ya fue confirmado."}
	}
	closed, err := s.closings.IsClosed(ctx, batch.PeriodYear, batch.PeriodMonth)
	if err != nil {
		return nil, err
	}
	if closed {
		return nil, &ImportError{Code: "month_closed", Message: "El mes está cerrado y no puede modificarse."}
	}

	rows, err := s.imports.ListRows(ctx, batch.ID)
	if err != nil {
		return nil, err
	}
	rowByLine := make(map[int]models.PensionImportRow, len(rows))
	for _, r := range rows {
		rowByLine[r.LineNo] = r
	}

	categories, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}
	categoryByID := make(map[int64]models.PensionCategory, len(categories))
	for _, c := range categories {
		categoryByID[c.ID] = c
	}

	if req.ChildID == 0 {
		return nil, &ImportError{Code: "missing_child", Message: "Seleccioná un hijo para importar."}
	}
	if len(req.Rows) == 0 {
		return nil, &ImportError{Code: "empty_batch", Message: "El lote no tiene filas para importar."}
	}

	type decision struct {
		row      models.PensionImportRow
		input    storage.ImportRecordInput
		category models.PensionCategory
		total    float64
	}
	var decisions []decision
	var validation models.PensionImportValidation
	validation.Status = "ok"
	validation.Errors = []models.ImportValidationError{}
	validation.DeclaredReimbursableTotal = batch.DeclaredTotal

	for _, rr := range req.Rows {
		row, ok := rowByLine[rr.LineNo]
		if !ok {
			validation.Errors = append(validation.Errors, models.ImportValidationError{LineNo: rr.LineNo, Code: "unknown_line", Message: "Línea desconocida en el lote."})
			continue
		}
		categoryID := rr.CategoryID
		if categoryID == 0 && row.ChosenCategoryID != nil {
			categoryID = *row.ChosenCategoryID
		}
		category, ok := categoryByID[categoryID]
		if !ok {
			validation.Errors = append(validation.Errors, models.ImportValidationError{LineNo: rr.LineNo, Code: "missing_category", Message: "Categoría inválida."})
			continue
		}
		total := rr.AmountTotal
		if total <= 0 {
			total = row.AmountTotal
		}
		if total <= 0 {
			validation.Errors = append(validation.Errors, models.ImportValidationError{LineNo: rr.LineNo, Code: "invalid_amount", Message: "El monto debe ser mayor a cero."})
			continue
		}
		if row.Currency != "" && !strings.EqualFold(row.Currency, category.Currency) {
			validation.Errors = append(validation.Errors, models.ImportValidationError{LineNo: rr.LineNo, Code: "currency_mismatch", Message: "La moneda no coincide con la categoría."})
			continue
		}
		amount, _ := roundShare(total, category.SharePercent)
		status := rr.Status
		if status == "" {
			status = "pending"
		}
		decisions = append(decisions, decision{
			row:      row,
			input:    storage.ImportRecordInput{LineNo: rr.LineNo, CategoryID: category.ID, Amount: amount, Status: status, SharePercent: category.SharePercent, Currency: category.Currency},
			category: category,
			total:    total,
		})
	}

	var exactShared, roundedShared, grandTotal float64
	for _, d := range decisions {
		grandTotal += d.input.Amount
		if d.category.SharePercent == 50 {
			exactShared += d.total * 0.5
			roundedShared += d.input.Amount
		}
	}
	validation.ComputedReimbursableTotal = roundedShared
	validation.ComputedGrandTotal = grandTotal
	if batch.DeclaredTotal > 0 {
		minTotal := math.Min(exactShared, roundedShared)
		maxTotal := math.Max(exactShared, roundedShared)
		if batch.DeclaredTotal < minTotal-0.0001 || batch.DeclaredTotal > maxTotal+0.0001 {
			validation.Errors = append(validation.Errors, models.ImportValidationError{Code: "total_mismatch", Message: "El total declarado no coincide con la suma calculada."})
		}
	}

	if len(validation.Errors) > 0 {
		validation.Status = "invalid"
		return nil, &ImportError{Code: "batch_invalid", Message: "El lote tiene filas inválidas; no se importó ningún registro.", Validation: &validation}
	}

	records := make([]storage.ImportRecordInput, 0, len(decisions))
	keywords := make([]storage.ImportKeywordInput, 0, len(decisions))
	for _, d := range decisions {
		records = append(records, d.input)
		kw := learnableKeyword(d.row.Concept)
		if kw != "" {
			keywords = append(keywords, storage.ImportKeywordInput{CategoryID: d.category.ID, Keyword: kw, Normalized: kw})
		}
	}

	created, skipped, learned, err := s.imports.ConfirmImport(ctx, batch.ID, req.ChildID, batch.PeriodYear, batch.PeriodMonth, records, keywords)
	if err != nil {
		return nil, &ImportError{Code: "confirm_failed", Message: err.Error()}
	}

	for _, d := range decisions {
		action := "create"
		if skipped > 0 {
			action = "existing"
		}
		_ = s.imports.UpdateRowDecision(ctx, batch.ID, d.input, req.ChildID, "ok", "", action)
	}

	return &models.PensionImportConfirmResult{
		BatchID:         batch.ID,
		Created:         created,
		SkippedExisting: skipped,
		LearnedKeywords: learned,
		Errors:          []models.ImportValidationError{},
	}, nil
}

// ListBatches devuelve los últimos lotes de importación.
func (s *PensionImportService) ListBatches(ctx context.Context, limit int) ([]models.PensionImportBatch, error) {
	return s.imports.ListBatches(ctx, limit)
}

// ListKeywords devuelve el diccionario de palabras clave.
func (s *PensionImportService) ListKeywords(ctx context.Context) ([]models.PensionCategoryKeyword, error) {
	return s.imports.ListKeywords(ctx)
}

// CreateKeyword agrega una palabra clave manual a una categoría.
func (s *PensionImportService) CreateKeyword(ctx context.Context, categoryID int64, keyword string) (*models.PensionCategoryKeyword, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("la palabra clave es requerida")
	}
	normalized := normalizeText(keyword)
	if normalized == "" {
		return nil, fmt.Errorf("la palabra clave no es válida")
	}
	return s.imports.CreateKeyword(ctx, categoryID, keyword, normalized, "manual")
}

// DeleteKeyword elimina una palabra clave.
func (s *PensionImportService) DeleteKeyword(ctx context.Context, id int64) error {
	return s.imports.DeleteKeyword(ctx, id)
}

// TemplateCSV devuelve la plantilla del formato canónico.
func (s *PensionImportService) TemplateCSV() string {
	return "periodo,concepto,monto,moneda,categoria,hijo,estado,notas\n" +
		"2026-10,PENSION OCTUBRE,12000.00,NIO,Pensión Alimenticia,,pending,\n" +
		"2026-10,COLEGIATURA,8000.00,NIO,Colegio - Mensualidad,,pending,\n" +
		"2026-10,COMBUSTIBLE,1200.00,NIO,Colegio - Transporte,,pending,\n"
}

// ExportCSV reconstruye el formato canónico desde los registros del mes.
func (s *PensionImportService) ExportCSV(ctx context.Context, year, month int) (string, error) {
	records, err := s.records.ListByFilters(ctx, year, month, 0)
	if err != nil {
		return "", err
	}
	categories, err := s.categories.List(ctx)
	if err != nil {
		return "", err
	}
	shareByID := make(map[int64]int, len(categories))
	for _, c := range categories {
		shareByID[c.ID] = c.SharePercent
	}

	var b strings.Builder
	b.WriteString("periodo,concepto,monto,moneda,categoria,hijo,estado,notas\n")
	period := fmt.Sprintf("%04d-%02d", year, month)
	for _, r := range records {
		share := shareByID[r.PensionCategoryID]
		if share == 0 {
			share = 100
		}
		total := r.Amount * 100 / float64(share)
		status := r.Status
		if status == "" {
			status = "pending"
		}
		fmt.Fprintf(&b, "%s,%s,%.2f,%s,%s,,%s,\n",
			period, csvEscape(r.CategoryName), total, r.Currency, csvEscape(r.CategoryName), status)
	}
	return b.String(), nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func (s *PensionImportService) parseInput(in PensionImportPreviewInput) (*ParsedDocument, error) {
	if strings.TrimSpace(in.Text) != "" {
		doc, err := ParseText(in.Text)
		if err != nil {
			return nil, err
		}
		doc.Source = "text"
		return doc, nil
	}
	if len(in.Content) == 0 {
		return nil, &ImportError{Code: "empty_file", Message: "El archivo está vacío."}
	}

	ext := strings.ToLower(filepath.Ext(in.Filename))
	mime := strings.ToLower(in.MIME)
	switch {
	case ext == ".csv" || strings.Contains(mime, "csv"):
		doc, err := ParseCSV(strings.NewReader(string(in.Content)))
		if err != nil {
			return nil, &ImportError{Code: "invalid_csv", Message: err.Error()}
		}
		return doc, nil
	case ext == ".pdf" || strings.Contains(mime, "pdf"):
		text, err := ExtractPDFText(in.Content)
		if err != nil {
			if err.Error() == "pdf_without_text" {
				return nil, &ImportError{Code: "pdf_without_text", Message: "El PDF no tiene capa de texto. Usá CSV o pegá el texto."}
			}
			return nil, &ImportError{Code: "invalid_pdf", Message: err.Error()}
		}
		doc, err := ParseText(text)
		if err != nil {
			return nil, err
		}
		doc.Source = "pdf"
		return doc, nil
	case ext == ".txt" || strings.Contains(mime, "text/plain"):
		doc, err := ParseText(string(in.Content))
		if err != nil {
			return nil, err
		}
		doc.Source = "txt"
		return doc, nil
	default:
		return nil, &ImportError{Code: "unsupported_format", Message: "Formato no soportado. Usá PDF, TXT o CSV."}
	}
}

func importPreviousPeriod(year, month int) (int, int) {
	if month == 1 {
		return year - 1, 12
	}
	return year, month - 1
}

// learnableKeyword quita meses, años y palabras vacías para aprender un término
// reutilizable entre meses ("PENSION OCTUBRE" -> "PENSION").
func learnableKeyword(concept string) string {
	n := normalizeText(concept)
	fields := strings.Fields(n)
	var out []string
	for _, f := range fields {
		if _, isMonth := importMonthNames[f]; isMonth {
			continue
		}
		if len(f) == 4 && strings.HasPrefix(f, "20") {
			continue
		}
		switch f {
		case "DE", "DEL", "LA", "EL", "LOS", "LAS", "Y", "A", "EN", "POR", "PARA", "MES":
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}
