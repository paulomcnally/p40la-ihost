package maintenance

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxDescriptionLen        = 500
	maxItemDescription       = 200
	maxQueryLen              = 100
	maxMaintenancesWithItems = 100
	maxItemsPerList          = 1000
	maxMoveItems             = 100
	dateLayout               = "2006-01-02"
	defaultArea              = AreaOther
	defaultKind              = KindCorrective
	defaultUnit              = UnitUnit
	defaultQuantity          = 1.0
)

// Service contiene la lógica de negocio del módulo (ARCH-MODULES).
type Service struct {
	storage *Storage
	now     func() time.Time
}

// NewService crea el servicio del módulo.
func NewService(storage *Storage) *Service {
	return &Service{storage: storage, now: time.Now}
}

// SetClock permite fijar el reloj en tests.
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// List devuelve el histórico de la vivienda con totales por moneda.
func (s *Service) List(ctx context.Context, homeID int64, filter Filter) (*List, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	if filter.Kind != "" && !contains(Kinds, filter.Kind) {
		return nil, invalid("invalid_kind", "El tipo de mantenimiento no es válido")
	}
	if filter.Year > 0 && filter.Date != "" {
		return nil, invalid("invalid_filter", "year y date no se pueden combinar")
	}
	if filter.Date != "" {
		parsed, err := time.Parse(dateLayout, strings.TrimSpace(filter.Date))
		if err != nil {
			return nil, invalid("invalid_date", "La fecha debe tener formato YYYY-MM-DD")
		}
		filter.Date = parsed.Format(dateLayout)
	}
	if filter.ItemType != "" && !contains(ItemTypes, filter.ItemType) {
		return nil, invalid("invalid_item_type", "El tipo de línea debe ser mano de obra o insumos")
	}
	filter.Query = strings.TrimSpace(filter.Query)
	if utf8.RuneCountInString(filter.Query) > maxQueryLen {
		return nil, invalid("invalid_query", "El texto de búsqueda supera el máximo permitido")
	}
	if filter.IncludeItems {
		filter.Limit = maxMaintenancesWithItems + 1
	}

	items, err := s.storage.List(ctx, homeID, filter)
	if err != nil {
		return nil, err
	}
	truncated := false
	if filter.IncludeItems && len(items) > maxMaintenancesWithItems {
		items = items[:maxMaintenancesWithItems]
		truncated = true
	}
	grouped, err := s.storage.TotalsByMaintenance(ctx, homeID, filter)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].TotalsByCurrency = roundTotals(grouped[items[index].ID])
	}
	if filter.IncludeItems && len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		details, itemsTruncated, err := s.storage.ListItemsByMaintenanceIDs(ctx, ids, maxItemsPerList)
		if err != nil {
			return nil, err
		}
		truncated = truncated || itemsTruncated
		for index := range items {
			if list, ok := details[items[index].ID]; ok {
				items[index].Items = list
			} else {
				items[index].Items = []Item{}
			}
		}
	}
	totals, err := s.storage.TotalsByCurrency(ctx, homeID, filter)
	if err != nil {
		return nil, err
	}
	return &List{Items: items, TotalsByCurrency: roundTotals(totals), Truncated: truncated}, nil
}

// Get devuelve un mantenimiento con sus líneas.
func (s *Service) Get(ctx context.Context, homeID, id int64) (*Maintenance, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	item, err := s.storage.Get(ctx, homeID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrMaintenanceNotFound
	}
	details, err := s.storage.ListItems(ctx, id)
	if err != nil {
		return nil, err
	}
	item.Items = details
	item.TotalsByCurrency = roundTotals(totalsFromItems(details))
	return item, nil
}

// Create registra un mantenimiento.
func (s *Service) Create(ctx context.Context, homeID int64, input Input) (*Maintenance, error) {
	return s.CreateWithItem(ctx, homeID, input, nil)
}

// CreateWithItem registra un mantenimiento y, si item no es nil, su primera
// línea en la misma transacción.
func (s *Service) CreateWithItem(ctx context.Context, homeID int64, input Input, item *ItemInput) (*Maintenance, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	if err := s.prepare(ctx, &input); err != nil {
		return nil, err
	}
	if item != nil {
		if err := s.prepareItem(ctx, item); err != nil {
			return nil, err
		}
	}
	var id int64
	var err error
	if item == nil {
		id, err = s.storage.Create(ctx, homeID, input)
	} else {
		id, err = s.storage.CreateWithItem(ctx, homeID, input, *item)
	}
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, homeID, id)
}

// Update actualiza un mantenimiento.
func (s *Service) Update(ctx context.Context, homeID, id int64, input Input) (*Maintenance, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	if err := s.requireMaintenance(ctx, homeID, id); err != nil {
		return nil, err
	}
	if err := s.prepare(ctx, &input); err != nil {
		return nil, err
	}
	exists, err := s.storage.Update(ctx, homeID, id, input)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMaintenanceNotFound
	}
	return s.Get(ctx, homeID, id)
}

// Delete elimina un mantenimiento y sus líneas (cascada).
func (s *Service) Delete(ctx context.Context, homeID, id int64) error {
	if err := s.requireHome(ctx, homeID); err != nil {
		return err
	}
	exists, err := s.storage.Delete(ctx, homeID, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrMaintenanceNotFound
	}
	return nil
}

// CreateItem agrega una línea de mano de obra o insumos.
func (s *Service) CreateItem(ctx context.Context, homeID, maintenanceID int64, input ItemInput) (*Item, error) {
	if err := s.requireMaintenance(ctx, homeID, maintenanceID); err != nil {
		return nil, err
	}
	if err := s.prepareItem(ctx, &input); err != nil {
		return nil, err
	}
	return s.storage.CreateItem(ctx, maintenanceID, input)
}

// UpdateItem actualiza una línea.
func (s *Service) UpdateItem(ctx context.Context, homeID, maintenanceID, itemID int64, input ItemInput) (*Item, error) {
	if err := s.requireMaintenance(ctx, homeID, maintenanceID); err != nil {
		return nil, err
	}
	if err := s.prepareItem(ctx, &input); err != nil {
		return nil, err
	}
	exists, err := s.storage.UpdateItem(ctx, maintenanceID, itemID, input)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrItemNotFound
	}
	return s.storage.GetItem(ctx, maintenanceID, itemID)
}

// DeleteItem elimina una línea.
func (s *Service) DeleteItem(ctx context.Context, homeID, maintenanceID, itemID int64) error {
	if err := s.requireMaintenance(ctx, homeID, maintenanceID); err != nil {
		return err
	}
	exists, err := s.storage.DeleteItem(ctx, maintenanceID, itemID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrItemNotFound
	}
	return nil
}

// MoveItems mueve líneas de una reparación a otra del mismo hogar. Si el
// destino no existe, lo crea en la misma transacción.
func (s *Service) MoveItems(ctx context.Context, homeID, sourceID int64, input MoveInput) (*MoveResult, error) {
	if err := s.requireMaintenance(ctx, homeID, sourceID); err != nil {
		return nil, err
	}
	if len(input.ItemIDs) == 0 || len(input.ItemIDs) > maxMoveItems {
		return nil, invalid("invalid_item_ids", "item_ids debe tener entre 1 y 100 líneas")
	}
	seen := make(map[int64]struct{}, len(input.ItemIDs))
	for _, id := range input.ItemIDs {
		if id <= 0 {
			return nil, invalid("invalid_item_ids", "item_ids solo acepta IDs positivos")
		}
		if _, ok := seen[id]; ok {
			return nil, invalid("invalid_item_ids", "item_ids no admite líneas repetidas")
		}
		seen[id] = struct{}{}
	}
	input.ItemType = strings.TrimSpace(input.ItemType)
	if input.ItemType != "" && !contains(ItemTypes, input.ItemType) {
		return nil, invalid("invalid_item_type", "El tipo de línea debe ser mano de obra o insumos")
	}

	targets := 0
	if input.TargetMaintenanceID != nil {
		targets++
	}
	if input.Target != nil {
		targets++
	}
	if targets == 0 {
		return nil, invalid("target_required", "Indicá target_maintenance_id o target")
	}
	if targets > 1 {
		return nil, invalid("target_conflict", "target_maintenance_id y target son excluyentes")
	}

	targetID := int64(0)
	var newTarget *Input
	if input.TargetMaintenanceID != nil {
		if *input.TargetMaintenanceID <= 0 {
			return nil, invalid("invalid_request", "target_maintenance_id debe ser mayor que cero")
		}
		if *input.TargetMaintenanceID == sourceID {
			return nil, invalid("same_maintenance", "El destino no puede ser la misma reparación de origen")
		}
		if err := s.requireMaintenance(ctx, homeID, *input.TargetMaintenanceID); err != nil {
			return nil, err
		}
		targetID = *input.TargetMaintenanceID
	} else {
		candidate := *input.Target
		if err := s.prepare(ctx, &candidate); err != nil {
			return nil, err
		}
		newTarget = &candidate
	}

	items, err := s.storage.ListItemsByIDs(ctx, input.ItemIDs)
	if err != nil {
		return nil, err
	}
	if len(items) != len(input.ItemIDs) {
		return nil, ErrItemNotFound
	}
	for _, item := range items {
		if item.MaintenanceID != sourceID {
			return nil, invalid("item_not_in_source",
				fmt.Sprintf("La línea %d no pertenece a la reparación %d", item.ID, sourceID))
		}
		if input.ItemType != "" && item.ItemType != input.ItemType {
			return nil, invalid("item_type_mismatch",
				fmt.Sprintf("La línea %d no es de tipo %s", item.ID, input.ItemType))
		}
	}

	targetID, remaining, deleted, err := s.storage.MoveItems(
		ctx, homeID, sourceID, targetID, newTarget, input.ItemIDs, input.DeleteSourceIfEmpty)
	if err != nil {
		return nil, err
	}

	moved, err := s.storage.ListItemsByIDs(ctx, input.ItemIDs)
	if err != nil {
		return nil, err
	}
	target, err := s.storage.Get(ctx, homeID, targetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrMaintenanceNotFound
	}
	return &MoveResult{
		SourceMaintenanceID: sourceID,
		TargetMaintenanceID: targetID,
		TargetCreated:       newTarget != nil,
		Target:              target,
		MovedCount:          len(moved),
		MovedItems:          moved,
		SourceItemCount:     remaining,
		SourceDeleted:       deleted,
	}, nil
}

func (s *Service) prepare(ctx context.Context, input *Input) error {
	performedAt, err := time.Parse(dateLayout, strings.TrimSpace(input.PerformedAt))
	if err != nil {
		return invalid("invalid_date", "La fecha del mantenimiento no es válida")
	}
	today := s.now()
	if performedAt.After(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())) {
		return invalid("future_date", "La fecha del mantenimiento no puede ser futura")
	}
	input.PerformedAt = performedAt.Format(dateLayout)

	input.Kind = strings.TrimSpace(input.Kind)
	if input.Kind == "" {
		input.Kind = defaultKind
	}
	if !contains(Kinds, input.Kind) {
		return invalid("invalid_kind", "El tipo de mantenimiento no es válido")
	}

	input.Area = strings.TrimSpace(input.Area)
	if input.Area == "" {
		input.Area = defaultArea
	}
	if !contains(Areas, input.Area) {
		return invalid("invalid_area", "El área del mantenimiento no es válida")
	}

	input.Description = strings.TrimSpace(input.Description)
	if utf8.RuneCountInString(input.Description) > maxDescriptionLen {
		return invalid("description_too_long", "La descripción supera el máximo permitido")
	}

	if input.InstitutionID != nil && *input.InstitutionID > 0 {
		ok, err := s.storage.InstitutionExists(ctx, *input.InstitutionID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInstitutionNotFound
		}
	} else {
		input.InstitutionID = nil
	}
	return nil
}

func (s *Service) prepareItem(ctx context.Context, input *ItemInput) error {
	input.ItemType = strings.TrimSpace(input.ItemType)
	if !contains(ItemTypes, input.ItemType) {
		return invalid("invalid_item_type", "El tipo de línea debe ser mano de obra o insumos")
	}

	input.Description = strings.TrimSpace(input.Description)
	if input.Description == "" {
		return invalid("item_description_required", "La descripción de la línea es requerida")
	}
	if utf8.RuneCountInString(input.Description) > maxItemDescription {
		return invalid("item_description_too_long", "La descripción de la línea supera el máximo permitido")
	}

	if math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
		return invalid("invalid_amount", "El monto de la línea no es válido")
	}
	if input.Amount < 0 {
		return invalid("invalid_amount", "El monto de la línea no puede ser negativo")
	}
	input.Amount = round2(input.Amount)

	if input.CurrencyID <= 0 {
		return invalid("currency_required", "La moneda de la línea es requerida")
	}
	ok, err := s.storage.CurrencyExists(ctx, input.CurrencyID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCurrencyNotFound
	}

	if input.ItemType == ItemLabor {
		input.Quantity = defaultQuantity
		input.Unit = defaultUnit
	} else {
		if input.Quantity == 0 {
			input.Quantity = defaultQuantity
		}
		if input.Quantity < 0 || math.IsNaN(input.Quantity) || math.IsInf(input.Quantity, 0) {
			return invalid("invalid_quantity", "La cantidad debe ser mayor que cero")
		}

		input.Unit = strings.TrimSpace(input.Unit)
		if input.Unit == "" {
			input.Unit = defaultUnit
		}
		if !contains(Units, input.Unit) {
			return invalid("invalid_unit", "La unidad de medida no es válida")
		}
	}

	if input.InstitutionID != nil && *input.InstitutionID > 0 {
		ok, err := s.storage.InstitutionExists(ctx, *input.InstitutionID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInstitutionNotFound
		}
	} else {
		input.InstitutionID = nil
	}
	return nil
}

func (s *Service) requireHome(ctx context.Context, homeID int64) error {
	if homeID <= 0 {
		return ErrHomeNotFound
	}
	ok, err := s.storage.HomeExists(ctx, homeID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrHomeNotFound
	}
	return nil
}

func (s *Service) requireMaintenance(ctx context.Context, homeID, maintenanceID int64) error {
	if err := s.requireHome(ctx, homeID); err != nil {
		return err
	}
	if maintenanceID <= 0 {
		return ErrMaintenanceNotFound
	}
	item, err := s.storage.Get(ctx, homeID, maintenanceID)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrMaintenanceNotFound
	}
	return nil
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// totalsFromItems agrupa las líneas por moneda con desglose labor/supplies.
func totalsFromItems(items []Item) []CurrencyTotal {
	order := make([]int64, 0)
	byID := make(map[int64]*CurrencyTotal)
	for _, item := range items {
		total, ok := byID[item.CurrencyID]
		if !ok {
			total = &CurrencyTotal{CurrencyID: item.CurrencyID, CurrencyCode: item.CurrencyCode}
			byID[item.CurrencyID] = total
			order = append(order, item.CurrencyID)
		}
		if item.ItemType == ItemLabor {
			total.LaborCost += item.Amount
		} else {
			total.SuppliesCost += item.Amount
		}
	}
	out := make([]CurrencyTotal, 0, len(order))
	for _, id := range order {
		total := byID[id]
		total.LaborCost = round2(total.LaborCost)
		total.SuppliesCost = round2(total.SuppliesCost)
		total.Total = round2(total.LaborCost + total.SuppliesCost)
		out = append(out, *total)
	}
	return out
}

func roundTotals(totals []CurrencyTotal) []CurrencyTotal {
	out := make([]CurrencyTotal, 0, len(totals))
	for _, total := range totals {
		total.LaborCost = round2(total.LaborCost)
		total.SuppliesCost = round2(total.SuppliesCost)
		total.Total = round2(total.LaborCost + total.SuppliesCost)
		out = append(out, total)
	}
	return out
}

// AreaOptions devuelve las áreas válidas (útil para tests y handlers).
func AreaOptions() []string { return append([]string(nil), Areas...) }

// KindOptions devuelve los tipos válidos.
func KindOptions() []string { return append([]string(nil), Kinds...) }

// UnitOptions devuelve las unidades válidas.
func UnitOptions() []string { return append([]string(nil), Units...) }
