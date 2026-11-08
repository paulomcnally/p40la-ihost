package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// AutoInspectionService contiene la lógica de negocio de las inspecciones
// mecánicas (SPEC-124).
type AutoInspectionService struct {
	storage      *storage.AutoInspectionStorage
	autos        *storage.AutoStorage
	institutions *storage.InstitutionStorage
	attachments  *AutoAttachmentService
	now          func() time.Time
}

// NewAutoInspectionService crea un nuevo AutoInspectionService.
func NewAutoInspectionService(
	st *storage.AutoInspectionStorage,
	autos *storage.AutoStorage,
	institutions *storage.InstitutionStorage,
	attachments *AutoAttachmentService,
) *AutoInspectionService {
	return &AutoInspectionService{
		storage:      st,
		autos:        autos,
		institutions: institutions,
		attachments:  attachments,
		now:          time.Now,
	}
}

// List devuelve las inspecciones de un auto y su cronograma calculado.
func (s *AutoInspectionService) List(ctx context.Context, autoID int64) (*models.AutoInspectionList, error) {
	auto, err := s.requireAuto(ctx, autoID)
	if err != nil {
		return nil, err
	}

	items, err := s.storage.ListByAuto(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if err := s.attachFiles(ctx, autoID, items); err != nil {
		return nil, err
	}

	approved, err := s.storage.YearsByResult(ctx, autoID, models.InspectionResultApproved)
	if err != nil {
		return nil, err
	}
	rejected, err := s.storage.YearsByResult(ctx, autoID, models.InspectionResultRejected)
	if err != nil {
		return nil, err
	}
	schedule := BuildInspectionSchedule(auto.Year, approved, rejected, s.now())

	return &models.AutoInspectionList{Items: items, Schedule: schedule}, nil
}

// Get devuelve una inspección con sus adjuntos.
func (s *AutoInspectionService) Get(ctx context.Context, autoID, id int64) (*models.AutoInspectionDetail, error) {
	if _, err := s.requireAuto(ctx, autoID); err != nil {
		return nil, err
	}
	return s.get(ctx, autoID, id)
}

// Create registra una inspección mecánica.
func (s *AutoInspectionService) Create(ctx context.Context, autoID int64, input models.AutoInspectionInput) (*models.AutoInspectionDetail, error) {
	auto, err := s.requireAuto(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if err := s.validate(ctx, auto, &input); err != nil {
		return nil, err
	}

	created, err := s.storage.Create(
		ctx, autoID, input.PerformedAt, input.InspectionYear, input.InstitutionID, input.Result, input.Notes,
	)
	if err != nil {
		return nil, err
	}
	return s.get(ctx, autoID, created.ID)
}

// Update actualiza una inspección mecánica.
func (s *AutoInspectionService) Update(ctx context.Context, autoID, id int64, input models.AutoInspectionInput) (*models.AutoInspectionDetail, error) {
	auto, err := s.requireAuto(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if err := s.validate(ctx, auto, &input); err != nil {
		return nil, err
	}

	exists, err := s.storage.Update(
		ctx, autoID, id, input.PerformedAt, input.InspectionYear, input.InstitutionID, input.Result, input.Notes,
	)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInspectionNotFound
	}
	return s.get(ctx, autoID, id)
}

// Delete elimina una inspección y sus adjuntos.
func (s *AutoInspectionService) Delete(ctx context.Context, autoID, id int64) error {
	if _, err := s.requireAuto(ctx, autoID); err != nil {
		return err
	}
	exists, err := s.storage.Delete(ctx, autoID, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrInspectionNotFound
	}
	return s.attachments.DeleteByOwner(ctx, autoID, models.AutoAttachmentOwnerInspection, id)
}

// Alerts devuelve el estado de inspección de todos los autos.
func (s *AutoInspectionService) Alerts(ctx context.Context) ([]models.AutoInspectionAlert, error) {
	autos, err := s.autos.List(ctx)
	if err != nil {
		return nil, err
	}
	approvedYears, err := s.storage.ApprovedYearsByAuto(ctx)
	if err != nil {
		return nil, err
	}

	alerts := make([]models.AutoInspectionAlert, 0, len(autos))
	for _, auto := range autos {
		schedule := BuildInspectionSchedule(auto.Year, approvedYears[auto.ID], nil, s.now())
		alerts = append(alerts, models.AutoInspectionAlert{
			AutoID:       auto.ID,
			NextDueYear:  schedule.NextDueYear,
			NextDueDate:  schedule.NextDueDate,
			Status:       schedule.Status,
			DaysUntilDue: schedule.DaysUntilDue,
		})
	}
	return alerts, nil
}

func (s *AutoInspectionService) validate(ctx context.Context, auto *models.Auto, input *models.AutoInspectionInput) error {
	performedAt, err := parseDateOnly(strings.TrimSpace(input.PerformedAt))
	if err != nil {
		return err
	}
	if isFutureDate(performedAt, s.now()) {
		return fmt.Errorf("la fecha de la inspección no puede ser futura")
	}
	if input.Result != models.InspectionResultApproved && input.Result != models.InspectionResultRejected {
		return fmt.Errorf("el resultado debe ser approved o rejected")
	}

	minYear := auto.Year + 2
	maxYear := int64(performedAt.Year()) + 1
	if input.InspectionYear < minYear {
		return fmt.Errorf("la inspección no puede ser anterior a %d", minYear)
	}
	if input.InspectionYear > maxYear {
		return fmt.Errorf("el año de inspección no puede superar %d", maxYear)
	}

	if input.InstitutionID <= 0 {
		return ErrInstitutionNotFound
	}
	institution, err := s.institutions.GetByID(ctx, input.InstitutionID)
	if err != nil {
		return err
	}
	if institution == nil {
		return ErrInstitutionNotFound
	}
	key, exists, err := s.institutions.GetCategoryKey(ctx, input.InstitutionID)
	if err != nil {
		return err
	}
	if !exists || key != institutionCategoryMechanicalInspection {
		return fmt.Errorf("%w: se esperaba un centro de inspección mecánica", ErrInstitutionCategory)
	}

	input.PerformedAt = performedAt.Format("2006-01-02")
	input.Notes = strings.TrimSpace(input.Notes)
	return nil
}

func (s *AutoInspectionService) requireAuto(ctx context.Context, autoID int64) (*models.Auto, error) {
	auto, err := s.autos.GetByID(ctx, autoID)
	if err != nil {
		return nil, err
	}
	if auto == nil {
		return nil, ErrAutoNotFound
	}
	return auto, nil
}

func (s *AutoInspectionService) get(ctx context.Context, autoID, id int64) (*models.AutoInspectionDetail, error) {
	item, err := s.storage.GetByID(ctx, autoID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrInspectionNotFound
	}
	attachments, err := s.attachments.ListByOwner(ctx, models.AutoAttachmentOwnerInspection, item.ID)
	if err != nil {
		return nil, err
	}
	if attachments == nil {
		attachments = []models.AutoAttachment{}
	}
	item.Attachments = attachments
	return item, nil
}

func (s *AutoInspectionService) attachFiles(ctx context.Context, autoID int64, items []models.AutoInspectionDetail) error {
	all, err := s.attachments.ListByAuto(ctx, autoID)
	if err != nil {
		return err
	}
	grouped := make(map[int64][]models.AutoAttachment)
	for _, att := range all {
		if att.OwnerType == models.AutoAttachmentOwnerInspection {
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
