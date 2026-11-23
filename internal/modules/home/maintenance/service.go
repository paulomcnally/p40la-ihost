package maintenance

import (
	"context"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxDescriptionLen  = 500
	maxItemDescription = 200
	dateLayout         = "2006-01-02"
	defaultArea        = AreaOther
	defaultKind        = KindCorrective
	defaultUnit        = UnitUnit
	defaultQuantity    = 1.0
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

	items, err := s.storage.List(ctx, homeID, filter)
	if err != nil {
		return nil, err
	}
	totals, err := s.storage.TotalsByCurrency(ctx, homeID, filter)
	if err != nil {
		return nil, err
	}
	return &List{Items: items, TotalsByCurrency: totals}, nil
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
	return item, nil
}

// Create registra un mantenimiento.
func (s *Service) Create(ctx context.Context, homeID int64, input Input) (*Maintenance, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	if err := s.prepare(ctx, &input); err != nil {
		return nil, err
	}
	id, err := s.storage.Create(ctx, homeID, input)
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

	if input.CurrencyID <= 0 {
		return invalid("currency_required", "La moneda es requerida")
	}
	ok, err := s.storage.CurrencyExists(ctx, input.CurrencyID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCurrencyNotFound
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

	if input.ItemType == ItemLabor {
		if input.InstitutionID != nil && *input.InstitutionID > 0 {
			return invalid("labor_institution", "La mano de obra no lleva institución de compra")
		}
		input.InstitutionID = nil
		input.Quantity = defaultQuantity
		input.Unit = defaultUnit
		return nil
	}

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

// AreaOptions devuelve las áreas válidas (útil para tests y handlers).
func AreaOptions() []string { return append([]string(nil), Areas...) }

// KindOptions devuelve los tipos válidos.
func KindOptions() []string { return append([]string(nil), Kinds...) }

// UnitOptions devuelve las unidades válidas.
func UnitOptions() []string { return append([]string(nil), Units...) }
