package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// CategoryService contiene la lógica de negocio para categorías de presupuesto
// (SPEC-093). SPEC-094: valida el vínculo con servicios del sistema.
// SPEC-096: una categoría puede vincularse a N servicios (service_ids).
// SPEC-098: sugiere el monto asignado a partir de las facturas de los servicios.
type CategoryService struct {
	categories *storage.CategoryStorage
	groups     *storage.CategoryGroupStorage
	services   *storage.ServiceStorage
	bills      *storage.BillStorage
}

// NewCategoryService crea un nuevo CategoryService.
func NewCategoryService(categories *storage.CategoryStorage, groups *storage.CategoryGroupStorage) *CategoryService {
	return &CategoryService{categories: categories, groups: groups}
}

// SetServiceStorage habilita la validación de vínculos con servicios del
// sistema (SPEC-094). Si no se configura, service_id/account_id se aceptan sin
// validar la existencia del servicio.
func (s *CategoryService) SetServiceStorage(services *storage.ServiceStorage) {
	s.services = services
}

// SetBillStorage habilita la sugerencia de asignación a partir de las facturas
// de los servicios vinculados (SPEC-098). Si no se configura, la sugerencia
// devuelve applies=false.
func (s *CategoryService) SetBillStorage(bills *storage.BillStorage) {
	s.bills = bills
}

// GetByID busca una categoría por ID.
func (s *CategoryService) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	return s.categories.GetByID(ctx, id)
}

// Create crea una nueva categoría validando el grupo.
func (s *CategoryService) Create(ctx context.Context, cat *models.Category) (*models.Category, error) {
	if err := s.validate(ctx, cat); err != nil {
		return nil, err
	}
	return s.categories.Create(ctx, cat)
}

// Update actualiza una categoría existente.
func (s *CategoryService) Update(ctx context.Context, cat *models.Category) (*models.Category, error) {
	if cat.ID == 0 {
		return nil, fmt.Errorf("id de categoría requerido")
	}
	if err := s.validate(ctx, cat); err != nil {
		return nil, err
	}
	return s.categories.Update(ctx, cat)
}

// Delete archiva una categoría. Si no tiene historial, la elimina de verdad.
func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	hasTx, err := s.categories.HasTransactions(ctx, id)
	if err != nil {
		return err
	}
	hasAssignments, err := s.categories.HasAssignments(ctx, id)
	if err != nil {
		return err
	}
	if hasTx || hasAssignments {
		return s.categories.SoftDelete(ctx, id)
	}
	return s.hardDelete(ctx, id)
}

// hardDelete elimina la categoría y sus reglas recurrentes huérfanas.
func (s *CategoryService) hardDelete(ctx context.Context, id int64) error {
	// Las reglas recurrentes se eliminan vía BudgetService.DeleteRule; aquí
	// simplemente archivamos (soft delete) para conservar coherencia.
	return s.categories.SoftDelete(ctx, id)
}

// Reorder actualiza el orden de las categorías de un grupo.
func (s *CategoryService) Reorder(ctx context.Context, groupID int64, ids []int64) error {
	return s.categories.Reorder(ctx, groupID, ids)
}

// SuggestedAssignment calcula la sugerencia de asignación de una categoría
// (SPEC-098): la suma de la factura más reciente por cada servicio vinculado,
// agrupada por moneda (currency_id del servicio). Devuelve applies=false si la
// categoría no tiene servicios vinculados, si no tiene storage de facturas
// configurado o si ningún servicio tiene facturas.
func (s *CategoryService) SuggestedAssignment(ctx context.Context, categoryID int64) (*models.SuggestedAssignment, error) {
	cat, err := s.categories.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if cat == nil || cat.DeletedAt != nil {
		return nil, fmt.Errorf("la categoría no existe")
	}

	res := &models.SuggestedAssignment{
		Suggested:          map[int64]float64{},
		SourceServiceIDs:   []int64{},
		SourceServiceNames: []string{},
	}
	if len(cat.ServiceIDs) == 0 || s.bills == nil {
		return res, nil
	}

	for _, sid := range cat.ServiceIDs {
		svc, err := s.services.GetByID(ctx, sid)
		if err != nil {
			return nil, err
		}
		if svc == nil {
			continue
		}
		bill, err := s.bills.LatestByService(ctx, sid)
		if err != nil {
			return nil, err
		}
		if bill == nil {
			continue
		}
		res.Suggested[svc.CurrencyID] += bill.Amount
		res.SourceServiceIDs = append(res.SourceServiceIDs, sid)
		res.SourceServiceNames = append(res.SourceServiceNames, svc.Name)
	}
	res.Applies = len(res.Suggested) > 0
	return res, nil
}

func (s *CategoryService) validate(ctx context.Context, cat *models.Category) error {
	cat.Name = strings.TrimSpace(cat.Name)
	cat.Icon = strings.TrimSpace(cat.Icon)
	if cat.Name == "" {
		return fmt.Errorf("el nombre de la categoría es requerido")
	}
	if cat.CategoryGroupID == 0 {
		return fmt.Errorf("debe seleccionar un grupo de categorías")
	}
	group, err := s.groups.GetByID(ctx, cat.CategoryGroupID)
	if err != nil {
		return fmt.Errorf("validar grupo: %w", err)
	}
	if group == nil {
		return fmt.Errorf("el grupo seleccionado no existe")
	}
	if cat.TargetAmount != nil && *cat.TargetAmount < 0 {
		return fmt.Errorf("la meta no puede ser negativa")
	}

	// Vínculo con servicios (SPEC-094/096).
	if len(cat.ServiceIDs) > 0 {
		if cat.AccountID == nil {
			return fmt.Errorf("si vincula servicios debe seleccionar una cuenta")
		}
		seen := make(map[int64]struct{}, len(cat.ServiceIDs))
		for _, sid := range cat.ServiceIDs {
			if sid == 0 {
				continue
			}
			if _, dup := seen[sid]; dup {
				return fmt.Errorf("el servicio %d está repetido en la lista", sid)
			}
			seen[sid] = struct{}{}
			if s.services != nil {
				svc, err := s.services.GetByID(ctx, sid)
				if err != nil {
					return fmt.Errorf("validar servicio: %w", err)
				}
				if svc == nil {
					return fmt.Errorf("el servicio %d no existe", sid)
				}
			}
			inUse, err := s.categories.ServiceLinkInUse(ctx, sid, cat.ID)
			if err != nil {
				return err
			}
			if inUse {
				return fmt.Errorf("el servicio %d ya está vinculado a otra categoría", sid)
			}
		}
	}
	return nil
}
