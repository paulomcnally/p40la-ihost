package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const (
	institutionCategoryWorkshop             = "workshop"
	institutionCategoryMechanicalInspection = "mechanical_inspection"
)

// AutoMaintenanceService contiene la lógica de negocio de los mantenimientos
// por kilometraje (SPEC-124).
type AutoMaintenanceService struct {
	storage      *storage.AutoMaintenanceStorage
	autos        *storage.AutoStorage
	institutions *storage.InstitutionStorage
	currencies   *storage.CurrencyStorage
	attachments  *AutoAttachmentService
	now          func() time.Time
}

// NewAutoMaintenanceService crea un nuevo AutoMaintenanceService.
func NewAutoMaintenanceService(
	st *storage.AutoMaintenanceStorage,
	autos *storage.AutoStorage,
	institutions *storage.InstitutionStorage,
	currencies *storage.CurrencyStorage,
	attachments *AutoAttachmentService,
) *AutoMaintenanceService {
	return &AutoMaintenanceService{
		storage:      st,
		autos:        autos,
		institutions: institutions,
		currencies:   currencies,
		attachments:  attachments,
		now:          time.Now,
	}
}

// List devuelve los mantenimientos de un auto, el último kilometraje y los
// totales por moneda.
func (s *AutoMaintenanceService) List(ctx context.Context, autoID int64) (*models.AutoMaintenanceList, error) {
	if _, err := s.requireAuto(ctx, autoID); err != nil {
		return nil, err
	}

	items, err := s.storage.ListByAuto(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if err := s.attachFiles(ctx, autoID, items); err != nil {
		return nil, err
	}
	latest, err := s.storage.LatestMileage(ctx, autoID)
	if err != nil {
		return nil, err
	}
	totals, err := s.storage.TotalsByCurrency(ctx, autoID)
	if err != nil {
		return nil, err
	}

	return &models.AutoMaintenanceList{
		Items:            items,
		LatestMileage:    latest,
		TotalsByCurrency: totals,
	}, nil
}

// Get devuelve un mantenimiento con sus adjuntos.
func (s *AutoMaintenanceService) Get(ctx context.Context, autoID, id int64) (*models.AutoMaintenanceDetail, error) {
	if _, err := s.requireAuto(ctx, autoID); err != nil {
		return nil, err
	}
	item, err := s.storage.GetByID(ctx, autoID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrMaintenanceNotFound
	}
	if err := s.attachOne(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// Create registra un mantenimiento.
func (s *AutoMaintenanceService) Create(ctx context.Context, autoID int64, input models.AutoMaintenanceInput) (*models.AutoMaintenanceDetail, error) {
	auto, err := s.requireAuto(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if err := s.validate(ctx, auto, &input); err != nil {
		return nil, err
	}

	created, err := s.storage.Create(
		ctx, autoID, input.PerformedAt, input.Mileage, input.InstitutionID,
		input.WorkshopName, input.Description, input.LaborCost, input.SuppliesCost, input.CurrencyID,
	)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, autoID, created.ID)
}

// Update actualiza un mantenimiento.
func (s *AutoMaintenanceService) Update(ctx context.Context, autoID, id int64, input models.AutoMaintenanceInput) (*models.AutoMaintenanceDetail, error) {
	auto, err := s.requireAuto(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if err := s.validate(ctx, auto, &input); err != nil {
		return nil, err
	}

	exists, err := s.storage.Update(
		ctx, autoID, id, input.PerformedAt, input.Mileage, input.InstitutionID,
		input.WorkshopName, input.Description, input.LaborCost, input.SuppliesCost, input.CurrencyID,
	)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMaintenanceNotFound
	}
	return s.Get(ctx, autoID, id)
}

// Delete elimina un mantenimiento y sus adjuntos.
func (s *AutoMaintenanceService) Delete(ctx context.Context, autoID, id int64) error {
	if _, err := s.requireAuto(ctx, autoID); err != nil {
		return err
	}
	exists, err := s.storage.Delete(ctx, autoID, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrMaintenanceNotFound
	}
	return s.attachments.DeleteByOwner(ctx, autoID, models.AutoAttachmentOwnerMaintenance, id)
}

func (s *AutoMaintenanceService) validate(ctx context.Context, auto *models.Auto, input *models.AutoMaintenanceInput) error {
	performedAt, err := parseDateOnly(strings.TrimSpace(input.PerformedAt))
	if err != nil {
		return err
	}
	if isFutureDate(performedAt, s.now()) {
		return fmt.Errorf("la fecha del mantenimiento no puede ser futura")
	}
	if input.Mileage < 0 {
		return fmt.Errorf("el kilometraje no puede ser negativo")
	}
	if input.LaborCost < 0 || input.SuppliesCost < 0 {
		return fmt.Errorf("los montos no pueden ser negativos")
	}
	if input.CurrencyID <= 0 {
		return ErrCurrencyNotFound
	}
	currency, err := s.currencies.GetByID(ctx, input.CurrencyID)
	if err != nil {
		return err
	}
	if currency == nil {
		return ErrCurrencyNotFound
	}

	input.WorkshopName = strings.TrimSpace(input.WorkshopName)
	input.Description = strings.TrimSpace(input.Description)
	input.PerformedAt = performedAt.Format("2006-01-02")

	if input.InstitutionID != nil && *input.InstitutionID <= 0 {
		input.InstitutionID = nil
	}
	if input.InstitutionID != nil {
		institution, err := s.institutions.GetByID(ctx, *input.InstitutionID)
		if err != nil {
			return err
		}
		if institution == nil {
			return ErrInstitutionNotFound
		}
		key, exists, err := s.institutions.GetCategoryKey(ctx, *input.InstitutionID)
		if err != nil {
			return err
		}
		if !exists || key != institutionCategoryWorkshop {
			return fmt.Errorf("%w: se esperaba un taller mecánico", ErrInstitutionCategory)
		}
		if input.WorkshopName == "" {
			input.WorkshopName = institution.Name
		}
	}
	if input.WorkshopName == "" {
		return fmt.Errorf("el nombre del taller es requerido")
	}
	return nil
}

func (s *AutoMaintenanceService) requireAuto(ctx context.Context, autoID int64) (*models.Auto, error) {
	auto, err := s.autos.GetByID(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if auto == nil {
		return nil, ErrAutoNotFound
	}
	return auto, nil
}

func (s *AutoMaintenanceService) attachFiles(ctx context.Context, autoID int64, items []models.AutoMaintenanceDetail) error {
	all, err := s.attachments.ListByAuto(ctx, autoID)
	if err != nil {
		return err
	}
	grouped := make(map[int64][]models.AutoAttachment)
	for _, att := range all {
		if att.OwnerType == models.AutoAttachmentOwnerMaintenance {
			grouped[att.OwnerID] = append(grouped[att.OwnerID], att)
		}
	}
	for index := range items {
		if attachments, ok := grouped[items[index].ID]; ok {
			items[index].Attachments = attachments
		}
	}
	return nil
}

func (s *AutoMaintenanceService) attachOne(ctx context.Context, item *models.AutoMaintenanceDetail) error {
	attachments, err := s.attachments.ListByOwner(ctx, models.AutoAttachmentOwnerMaintenance, item.ID)
	if err != nil {
		return err
	}
	if attachments == nil {
		attachments = []models.AutoAttachment{}
	}
	item.Attachments = attachments
	return nil
}
