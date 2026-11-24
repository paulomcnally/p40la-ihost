package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// Códigos estables de error para las operaciones masivas de facturas
// (SPEC-145). El handler HTTP los traduce a estados 4xx/5xx y el MCP los
// muestra al agente sin filtrar detalles internos.
const (
	BulkErrorInvalidRange      = "invalid_range"
	BulkErrorInvalidSelector   = "invalid_selector"
	BulkErrorInvalidPaidAt     = "invalid_paid_at"
	BulkErrorInvalidAmount     = "invalid_amount"
	BulkErrorServiceNotFound   = "service_not_found"
	BulkErrorServiceInactive   = "service_inactive"
	BulkErrorAmountUnresolved  = "amount_unresolved"
	BulkErrorInvalidBillID     = "invalid_bill_id"
	BulkErrorInvalidSource     = "invalid_source"
	BulkErrorInternal          = "internal_error"
	maxBulkPeriods             = 36
	bulkStatusCreated          = "created"
	bulkStatusPlanned          = "planned"
	bulkStatusSkipped          = "skipped"
	bulkStatusFailed           = "failed"
	bulkStatusPaid             = "paid"
	bulkReasonExistsPending    = "exists_pending"
	bulkReasonExistsPaid       = "exists_paid"
	bulkReasonDeletedSlot      = "deleted_slot"
	bulkReasonAlreadyPaid      = "already_paid"
	bulkReasonFuturePeriod     = "future_period"
	bulkReasonDatabase         = "db_error"
	bulkReasonPaymentFailed    = "pay_failed"
	bulkSelectorBillIDs        = "bill_ids"
	bulkSelectorMonth          = "month"
	bulkSelectorRange          = "range"
	bulkSelectorAllPendingUpTo = "all_pending_up_to"
)

var billPeriodRegex = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)

// BulkError es un error de negocio estable de las operaciones masivas.
type BulkError struct {
	Code    string
	Message string
}

func (e *BulkError) Error() string {
	return e.Code + ": " + e.Message
}

func newBulkError(code, format string, args ...any) error {
	return &BulkError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// GenerateBillsOptions configura la generación masiva de facturas de un servicio.
type GenerateBillsOptions struct {
	From          string
	To            string
	Amount        *float64
	DryRun        bool
	AllowInactive bool
	Source        string
	Now           time.Time
}

// GenerateBillItem describe el resultado por período.
type GenerateBillItem struct {
	Year    int     `json:"year"`
	Month   int     `json:"month"`
	Period  string  `json:"period"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status"`
	BillID  int64   `json:"bill_id,omitempty"`
	Reason  string  `json:"reason,omitempty"`
	Message string  `json:"message,omitempty"`
}

// GenerateBillsResult es el resumen de una generación masiva.
type GenerateBillsResult struct {
	ServiceID    int64              `json:"service_id"`
	ServiceName  string             `json:"service_name,omitempty"`
	From         string             `json:"from"`
	To           string             `json:"to"`
	DryRun       bool               `json:"dry_run"`
	Amount       float64            `json:"amount"`
	CurrencyCode string             `json:"currency_code,omitempty"`
	Created      int                `json:"created"`
	Skipped      int                `json:"skipped"`
	Failed       int                `json:"failed"`
	Items        []GenerateBillItem `json:"items"`
}

// PayBillsOptions configura el pago masivo de facturas de un servicio. Se debe
// indicar exactamente un selector: BillIDs, Month, From+To o AllPendingUpTo.
type PayBillsOptions struct {
	BillIDs          []int64
	Month            string
	From             string
	To               string
	AllPendingUpTo   string
	PaidAt           string
	DriveURL         string
	PaymentReference string
	DryRun           bool
	Source           string
	Now              time.Time
}

// PayBillItem describe el resultado por factura.
type PayBillItem struct {
	BillID  int64   `json:"bill_id"`
	Year    int     `json:"year"`
	Month   int     `json:"month"`
	Period  string  `json:"period"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status"`
	Reason  string  `json:"reason,omitempty"`
	Message string  `json:"message,omitempty"`
}

// PayBillsResult es el resumen de un pago masivo.
type PayBillsResult struct {
	ServiceID    int64         `json:"service_id"`
	ServiceName  string        `json:"service_name,omitempty"`
	Selector     string        `json:"selector"`
	PaidAt       string        `json:"paid_at"`
	DryRun       bool          `json:"dry_run"`
	Paid         int           `json:"paid"`
	Skipped      int           `json:"skipped"`
	Failed       int           `json:"failed"`
	TotalAmount  float64       `json:"total_amount"`
	CurrencyCode string        `json:"currency_code,omitempty"`
	Items        []PayBillItem `json:"items"`
}

func parseBulkPeriod(raw string) (billPeriod, error) {
	raw = strings.TrimSpace(raw)
	if !billPeriodRegex.MatchString(raw) {
		return billPeriod{}, newBulkError(BulkErrorInvalidRange, "el período debe tener formato YYYY-MM")
	}
	parsed, err := time.Parse("2006-01", raw)
	if err != nil {
		return billPeriod{}, newBulkError(BulkErrorInvalidRange, "el período debe tener formato YYYY-MM")
	}
	return billPeriod{Year: parsed.Year(), Month: int(parsed.Month())}, nil
}

func (ym billPeriod) String() string {
	return fmt.Sprintf("%04d-%02d", ym.Year, ym.Month)
}

func (ym billPeriod) addMonth() billPeriod {
	if ym.Month == 12 {
		return billPeriod{Year: ym.Year + 1, Month: 1}
	}
	return billPeriod{Year: ym.Year, Month: ym.Month + 1}
}

func (ym billPeriod) before(other billPeriod) bool {
	if ym.Year != other.Year {
		return ym.Year < other.Year
	}
	return ym.Month < other.Month
}

func (ym billPeriod) after(other billPeriod) bool {
	if ym.Year != other.Year {
		return ym.Year > other.Year
	}
	return ym.Month > other.Month
}

func (ym billPeriod) equal(other billPeriod) bool {
	return ym.Year == other.Year && ym.Month == other.Month
}

func bulkMonthDistance(from, to billPeriod) int {
	return (to.Year-from.Year)*12 + (to.Month - from.Month) + 1
}

func currentBulkPeriod(now time.Time) billPeriod {
	return billPeriod{Year: now.Year(), Month: int(now.Month())}
}

// bulkPeriodOfBill devuelve el período calendario de una factura. Las facturas
// anuales se guardan con month=0 y su período es enero.
func bulkPeriodOfBill(bill models.Bill) billPeriod {
	month := bill.Month
	if month == 0 {
		month = 1
	}
	return billPeriod{Year: bill.Year, Month: month}
}

func normalizeBulkSource(source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return models.BillSourceDashboard, nil
	}
	if source != models.BillSourceDashboard && source != models.BillSourceMCP {
		return "", newBulkError(BulkErrorInvalidSource, "origen inválido: usá dashboard o mcp")
	}
	return source, nil
}

func (s *BillService) serviceCurrencyCode(ctx context.Context, currencyID int64) string {
	if s.currency == nil || currencyID == 0 {
		return ""
	}
	currency, err := s.currency.GetByID(ctx, currencyID)
	if err != nil || currency == nil {
		return ""
	}
	return currency.Code
}

// GenerateForService crea las facturas de un servicio para un rango inclusivo
// de períodos. Es idempotente por período: omite las facturas existentes y
// reporta los slots ocupados por facturas soft-deleted. SPEC-145.
func (s *BillService) GenerateForService(ctx context.Context, serviceID int64, opts GenerateBillsOptions) (*GenerateBillsResult, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	source, err := normalizeBulkSource(opts.Source)
	if err != nil {
		return nil, err
	}

	from, err := parseBulkPeriod(opts.From)
	if err != nil {
		return nil, err
	}
	toRaw := strings.TrimSpace(opts.To)
	if toRaw == "" {
		toRaw = currentBulkPeriod(opts.Now).String()
	}
	to, err := parseBulkPeriod(toRaw)
	if err != nil {
		return nil, err
	}
	if to.before(from) {
		return nil, newBulkError(BulkErrorInvalidRange, "el período final no puede ser anterior al inicial")
	}
	current := currentBulkPeriod(opts.Now)
	if to.after(current) {
		return nil, newBulkError(BulkErrorInvalidRange, "no se pueden generar facturas de períodos futuros")
	}
	if bulkMonthDistance(from, to) > maxBulkPeriods {
		return nil, newBulkError(BulkErrorInvalidRange, "el rango máximo es de %d meses", maxBulkPeriods)
	}

	svc, err := s.services.GetByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", BulkErrorInternal, err)
	}
	if svc == nil {
		return nil, newBulkError(BulkErrorServiceNotFound, "el servicio no existe")
	}
	if !svc.Active && !opts.AllowInactive {
		return nil, newBulkError(BulkErrorServiceInactive, "el servicio está inactivo; activalo o usá allow_inactive")
	}

	amount, err := s.resolveGenerateAmount(ctx, svc, opts.Amount)
	if err != nil {
		return nil, err
	}

	result := &GenerateBillsResult{
		ServiceID:    serviceID,
		ServiceName:  svc.Name,
		From:         from.String(),
		To:           to.String(),
		DryRun:       opts.DryRun,
		Amount:       amount,
		CurrencyCode: s.serviceCurrencyCode(ctx, svc.CurrencyID),
		Items:        make([]GenerateBillItem, 0),
	}

	for period := from; !period.after(to); period = period.addMonth() {
		billMonth := period.Month
		if svc.Frequency == FrequencyYearly {
			if period.Month != 1 {
				continue
			}
			billMonth = 0
		}

		item := GenerateBillItem{
			Year:   period.Year,
			Month:  billMonth,
			Period: period.String(),
			Amount: amount,
		}

		existing, findErr := s.storage.FindByServicePeriodIncludingDeleted(ctx, serviceID, period.Year, billMonth)
		if findErr != nil {
			item.Status = bulkStatusFailed
			item.Reason = bulkReasonDatabase
			item.Message = "no se pudo verificar el período"
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}
		if existing != nil {
			item.BillID = existing.ID
			item.Status = bulkStatusSkipped
			switch {
			case existing.DeletedAt != nil:
				item.Reason = bulkReasonDeletedSlot
				item.Message = "existe una factura eliminada para el período; restaurala o eliminá el registro antes de regenerar"
			case existing.Status == "paid":
				item.Reason = bulkReasonExistsPaid
				item.Message = "el período ya tiene una factura pagada"
			default:
				item.Reason = bulkReasonExistsPending
				item.Message = "el período ya tiene una factura pendiente"
			}
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}

		if opts.DryRun {
			item.Status = bulkStatusPlanned
			result.Items = append(result.Items, item)
			continue
		}

		created, createErr := s.CreateWithSource(ctx, &models.Bill{
			ServiceID: serviceID,
			Year:      period.Year,
			Month:     billMonth,
			Amount:    amount,
			Status:    "pending",
		}, source)
		if createErr != nil {
			item.Status = bulkStatusFailed
			item.Reason = bulkReasonDatabase
			item.Message = createErr.Error()
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}
		item.Status = bulkStatusCreated
		item.BillID = created.ID
		result.Created++
		result.Items = append(result.Items, item)
	}

	return result, nil
}

func (s *BillService) resolveGenerateAmount(ctx context.Context, svc *models.Service, explicit *float64) (float64, error) {
	if explicit != nil {
		if *explicit < 0 {
			return 0, newBulkError(BulkErrorInvalidAmount, "el monto no puede ser negativo")
		}
		return *explicit, nil
	}
	bills, err := s.storage.ListByService(ctx, svc.ID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", BulkErrorInternal, err)
	}
	for _, bill := range bills {
		if bill.Amount > 0 {
			return bill.Amount, nil
		}
	}
	if svc.SuggestedAmount > 0 {
		return svc.SuggestedAmount, nil
	}
	return 0, newBulkError(BulkErrorAmountUnresolved,
		"no se pudo determinar el monto: indicá amount o configurá un monto sugerido en el servicio")
}

// PayForService paga facturas pendientes de un servicio según un selector
// único. Es idempotente: las facturas ya pagadas se omiten. SPEC-145.
func (s *BillService) PayForService(ctx context.Context, serviceID int64, opts PayBillsOptions) (*PayBillsResult, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	source, err := normalizeBulkSource(opts.Source)
	if err != nil {
		return nil, err
	}

	selector, err := paySelector(opts)
	if err != nil {
		return nil, err
	}

	svc, err := s.services.GetByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", BulkErrorInternal, err)
	}
	if svc == nil {
		return nil, newBulkError(BulkErrorServiceNotFound, "el servicio no existe")
	}

	paidAt, err := parsePaidAt(opts.PaidAt, opts.Now)
	if err != nil {
		return nil, err
	}

	candidates, err := s.payCandidates(ctx, svc, opts, selector)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := bulkPeriodOfBill(candidates[i]), bulkPeriodOfBill(candidates[j])
		if a.Year != b.Year {
			return a.Year < b.Year
		}
		if a.Month != b.Month {
			return a.Month < b.Month
		}
		return candidates[i].ID < candidates[j].ID
	})

	current := currentBulkPeriod(opts.Now)
	result := &PayBillsResult{
		ServiceID:    serviceID,
		ServiceName:  svc.Name,
		Selector:     selector,
		PaidAt:       paidAt.Format("2006-01-02"),
		DryRun:       opts.DryRun,
		CurrencyCode: s.serviceCurrencyCode(ctx, svc.CurrencyID),
		Items:        make([]PayBillItem, 0, len(candidates)),
	}

	for _, bill := range candidates {
		period := bulkPeriodOfBill(bill)
		item := PayBillItem{
			BillID: bill.ID,
			Year:   bill.Year,
			Month:  bill.Month,
			Period: period.String(),
			Amount: bill.Amount,
		}

		if period.after(current) {
			item.Status = bulkStatusSkipped
			item.Reason = bulkReasonFuturePeriod
			item.Message = "la factura es de un período futuro"
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}
		if bill.Status == "paid" {
			item.Status = bulkStatusSkipped
			item.Reason = bulkReasonAlreadyPaid
			item.Message = "la factura ya está pagada"
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}

		if opts.DryRun {
			item.Status = bulkStatusPlanned
			result.TotalAmount += bill.Amount
			result.Items = append(result.Items, item)
			continue
		}

		if _, payErr := s.PayBillWithSource(ctx, bill.ID, paidAt, opts.DriveURL, opts.PaymentReference, source); payErr != nil {
			if payErr.Error() == "la factura ya está pagada" {
				item.Status = bulkStatusSkipped
				item.Reason = bulkReasonAlreadyPaid
				item.Message = payErr.Error()
				result.Skipped++
			} else {
				item.Status = bulkStatusFailed
				item.Reason = bulkReasonPaymentFailed
				item.Message = payErr.Error()
				result.Failed++
			}
			result.Items = append(result.Items, item)
			continue
		}
		item.Status = bulkStatusPaid
		result.Paid++
		result.TotalAmount += bill.Amount
		result.Items = append(result.Items, item)
	}

	return result, nil
}

func paySelector(opts PayBillsOptions) (string, error) {
	hasIDs := len(opts.BillIDs) > 0
	hasMonth := strings.TrimSpace(opts.Month) != ""
	hasFrom := strings.TrimSpace(opts.From) != ""
	hasTo := strings.TrimSpace(opts.To) != ""
	hasAll := strings.TrimSpace(opts.AllPendingUpTo) != ""

	selectors := 0
	for _, present := range []bool{hasIDs, hasMonth, hasFrom || hasTo, hasAll} {
		if present {
			selectors++
		}
	}
	if selectors != 1 {
		return "", newBulkError(BulkErrorInvalidSelector,
			"indicá exactamente un selector: bill_ids, month, from+to o all_pending_up_to")
	}
	switch {
	case hasIDs:
		if hasMonth || hasFrom || hasTo || hasAll {
			return "", newBulkError(BulkErrorInvalidSelector, "no combines bill_ids con otros selectores")
		}
		return bulkSelectorBillIDs, nil
	case hasMonth:
		if hasFrom || hasTo || hasAll {
			return "", newBulkError(BulkErrorInvalidSelector, "no combines month con otros selectores")
		}
		return bulkSelectorMonth, nil
	case hasAll:
		if hasFrom || hasTo {
			return "", newBulkError(BulkErrorInvalidSelector, "no combines all_pending_up_to con from/to")
		}
		return bulkSelectorAllPendingUpTo, nil
	default:
		if !hasFrom || !hasTo {
			return "", newBulkError(BulkErrorInvalidSelector, "el rango requiere from y to")
		}
		return bulkSelectorRange, nil
	}
}

func parsePaidAt(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if raw == "" {
		return today, nil
	}
	paidAt, err := time.ParseInLocation("2006-01-02", raw, now.Location())
	if err != nil {
		return time.Time{}, newBulkError(BulkErrorInvalidPaidAt, "la fecha de pago debe tener formato YYYY-MM-DD")
	}
	if paidAt.After(today) {
		return time.Time{}, newBulkError(BulkErrorInvalidPaidAt, "la fecha de pago no puede ser futura")
	}
	return paidAt, nil
}

func (s *BillService) payCandidates(ctx context.Context, svc *models.Service, opts PayBillsOptions, selector string) ([]models.Bill, error) {
	switch selector {
	case bulkSelectorBillIDs:
		seen := make(map[int64]struct{}, len(opts.BillIDs))
		candidates := make([]models.Bill, 0, len(opts.BillIDs))
		for _, id := range opts.BillIDs {
			if id <= 0 {
				return nil, newBulkError(BulkErrorInvalidBillID, "ID de factura inválido")
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			bill, err := s.storage.GetByID(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", BulkErrorInternal, err)
			}
			if bill == nil || bill.ServiceID != svc.ID {
				return nil, newBulkError(BulkErrorInvalidBillID, "la factura %d no existe o no pertenece al servicio", id)
			}
			candidates = append(candidates, *bill)
		}
		return candidates, nil
	case bulkSelectorMonth:
		month, err := parseBulkPeriod(opts.Month)
		if err != nil {
			return nil, err
		}
		bills, err := s.storage.ListByService(ctx, svc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", BulkErrorInternal, err)
		}
		return filterBillsByBulkPeriod(bills, func(period billPeriod) bool {
			return period.equal(month)
		}), nil
	case bulkSelectorRange:
		from, err := parseBulkPeriod(opts.From)
		if err != nil {
			return nil, err
		}
		to, err := parseBulkPeriod(opts.To)
		if err != nil {
			return nil, err
		}
		if to.before(from) {
			return nil, newBulkError(BulkErrorInvalidRange, "el período final no puede ser anterior al inicial")
		}
		bills, err := s.storage.ListByService(ctx, svc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", BulkErrorInternal, err)
		}
		return filterBillsByBulkPeriod(bills, func(period billPeriod) bool {
			return !period.before(from) && !period.after(to)
		}), nil
	case bulkSelectorAllPendingUpTo:
		to, err := parseBulkPeriod(opts.AllPendingUpTo)
		if err != nil {
			return nil, err
		}
		bills, err := s.storage.ListByService(ctx, svc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", BulkErrorInternal, err)
		}
		return filterBillsByBulkPeriod(bills, func(period billPeriod) bool {
			return !period.after(to)
		}), nil
	default:
		return nil, newBulkError(BulkErrorInvalidSelector, "selector de pago inválido")
	}
}

func filterBillsByBulkPeriod(bills []models.Bill, keep func(billPeriod) bool) []models.Bill {
	filtered := make([]models.Bill, 0, len(bills))
	for _, bill := range bills {
		if keep(bulkPeriodOfBill(bill)) {
			filtered = append(filtered, bill)
		}
	}
	return filtered
}
