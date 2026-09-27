package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// BudgetCategoryGroup es la respuesta de un grupo con sus categorías.
type BudgetCategoryGroup struct {
	ID         int64             `json:"id"`
	Name       string            `json:"name"`
	Icon       string            `json:"icon"`
	SortOrder  int               `json:"sort_order"`
	Categories []models.Category `json:"categories"`
}

// BudgetCategoryRow es una fila de la vista mensual con los valores calculados
// por moneda (map currency_id -> monto).
type BudgetCategoryRow struct {
	models.Category
	Assigned     map[int64]float64 `json:"assigned"`
	Activity     map[int64]float64 `json:"activity"`
	Available    map[int64]float64 `json:"available"`
	RecurringRule *models.RecurringRule `json:"recurring_rule,omitempty"`
}

// CurrencyTotal es un total agregado por moneda para el encabezado de la vista.
type CurrencyTotal struct {
	CurrencyID  int64   `json:"currency_id"`
	Code        string  `json:"code"`
	Symbol      string  `json:"symbol"`
	TotalIncome float64 `json:"total_income"`
	TotalAssigned float64 `json:"total_assigned"`
	Unassigned  float64 `json:"unassigned"`
}

// BudgetMonthView es la respuesta completa de la vista mensual.
type BudgetMonthView struct {
	Year           int                `json:"year"`
	Month          int                `json:"month"`
	CurrencyTotals []CurrencyTotal    `json:"currency_totals"`
	Groups         []BudgetMonthGroup `json:"groups"`
}

// BudgetMonthGroup agrupa las filas de la vista por CategoryGroup.
type BudgetMonthGroup struct {
	ID        int64              `json:"id"`
	Name      string             `json:"name"`
	Icon      string             `json:"icon"`
	SortOrder int                `json:"sort_order"`
	Categories []BudgetCategoryRow `json:"categories"`
}

// BudgetService contiene la lógica del presupuesto (SPEC-093).
type BudgetService struct {
	groups     *storage.CategoryGroupStorage
	categories *storage.CategoryStorage
	months     *storage.BudgetMonthStorage
	assigns    *storage.AssignmentStorage
	rules      *storage.RecurringRuleStorage
	transactions *storage.TransactionStorage
	currencies *storage.CurrencyStorage
}

// NewBudgetService crea un nuevo BudgetService.
func NewBudgetService(
	groups *storage.CategoryGroupStorage,
	categories *storage.CategoryStorage,
	months *storage.BudgetMonthStorage,
	assigns *storage.AssignmentStorage,
	rules *storage.RecurringRuleStorage,
	transactions *storage.TransactionStorage,
	currencies *storage.CurrencyStorage,
) *BudgetService {
	return &BudgetService{
		groups:       groups,
		categories:   categories,
		months:       months,
		assigns:      assigns,
		rules:        rules,
		transactions: transactions,
		currencies:   currencies,
	}
}

// MonthView devuelve la vista mensual completa (materializando recurrentes si
// es necesario).
func (s *BudgetService) MonthView(ctx context.Context, year, month int) (*BudgetMonthView, error) {
	monthRow, err := s.months.GetOrCreate(ctx, year, month)
	if err != nil {
		return nil, err
	}

	if err := s.MaterializeRecurring(ctx, year, month); err != nil {
		return nil, err
	}

	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, err
	}

	assignedByCategory, err := s.assigns.SumByMonthGrouped(ctx, monthRow.ID)
	if err != nil {
		return nil, err
	}
	activityByCategory, err := s.transactions.ActivityByCategoryGrouped(ctx, year, month)
	if err != nil {
		return nil, err
	}
	incomeByCurrency, err := s.transactions.IncomeByMonthGrouped(ctx, year, month)
	if err != nil {
		return nil, err
	}

	currencyCodes, err := s.currencyCodesByID(ctx)
	if err != nil {
		return nil, err
	}

	// Agregar las categorías con transacciones pero sin grupo (Sin categorizar) no
	// se muestran como fila; se reflejan en el total de ingresos.

	view := &BudgetMonthView{Year: year, Month: month, Groups: []BudgetMonthGroup{}, CurrencyTotals: []CurrencyTotal{}}
	assignedTotals := make(map[int64]float64)

	for i := range groups {
		group := &groups[i]
		cats, err := s.categories.ListByGroup(ctx, group.ID)
		if err != nil {
			return nil, err
		}

		mg := BudgetMonthGroup{
			ID:         group.ID,
			Name:       group.Name,
			Icon:       group.Icon,
			SortOrder:  group.SortOrder,
			Categories: []BudgetCategoryRow{},
		}

		for j := range cats {
			cat := &cats[j]
			row := BudgetCategoryRow{
				Category: *cat,
				Assigned:  sumMap(assignedByCategory[cat.ID]),
				Activity:  sumMap(activityByCategory[cat.ID]),
			}
			row.Available = make(map[int64]float64)
			for cid, assigned := range row.Assigned {
				row.Available[cid] = assigned - row.Activity[cid]
				assignedTotals[cid] += assigned
			}
			for cid, activity := range row.Activity {
				if _, ok := row.Assigned[cid]; !ok {
					row.Available[cid] = -activity
				}
			}

			rule, err := s.activeRuleForCategory(ctx, cat.ID, year, month)
			if err != nil {
				return nil, err
			}
			if rule != nil {
				row.RecurringRule = rule
			}

			mg.Categories = append(mg.Categories, row)
		}

		view.Groups = append(view.Groups, mg)
	}

	// Totales por moneda.
	currencyIDs := make(map[int64]struct{})
	for cid := range incomeByCurrency {
		currencyIDs[cid] = struct{}{}
	}
	for cid := range assignedTotals {
		currencyIDs[cid] = struct{}{}
	}
	ids := make([]int64, 0, len(currencyIDs))
	for cid := range currencyIDs {
		ids = append(ids, cid)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })

	for _, cid := range ids {
		info := currencyCodes[cid]
		assigned := assignedTotals[cid]
		income := incomeByCurrency[cid]
		view.CurrencyTotals = append(view.CurrencyTotals, CurrencyTotal{
			CurrencyID:    cid,
			Code:          info.Code,
			Symbol:        info.Symbol,
			TotalIncome:   income,
			TotalAssigned: assigned,
			Unassigned:    income - assigned,
		})
	}

	return view, nil
}

// Assign asigna un monto a una categoría en un mes. Si source=recurring y no
// existe regla, la crea desde ese mes. Devuelve el assignment.
func (s *BudgetService) Assign(ctx context.Context, year, month int, categoryID, currencyID int64, amount float64, makeRecurring bool) (*models.Assignment, error) {
	if amount < 0 {
		return nil, fmt.Errorf("el monto no puede ser negativo")
	}
	if categoryID == 0 {
		return nil, fmt.Errorf("debe seleccionar una categoría")
	}
	cat, err := s.categories.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if cat == nil || cat.DeletedAt != nil {
		return nil, fmt.Errorf("la categoría no existe")
	}
	currency, err := s.currencies.GetByID(ctx, currencyID)
	if err != nil {
		return nil, err
	}
	if currency == nil {
		return nil, fmt.Errorf("la moneda no existe")
	}

	monthRow, err := s.months.GetOrCreate(ctx, year, month)
	if err != nil {
		return nil, err
	}

	var ruleID *int64
	source := "one_time"
	if makeRecurring {
		start := fmt.Sprintf("%04d-%02d", year, month)
		existing, err := s.rules.ListActiveByCategory(ctx, categoryID)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			// Editar la regla activa existente (afecta desde este mes en adelante).
			rule := existing[0]
			rule.Amount = amount
			rule.StartMonth = start
			updated, err := s.rules.Update(ctx, &rule)
			if err != nil {
				return nil, err
			}
			id := updated.ID
			ruleID = &id
			source = "recurring"
		} else {
			created, err := s.rules.Create(ctx, &models.RecurringRule{
				CategoryID: categoryID,
				Amount:     amount,
				Frequency:  "monthly",
				StartMonth: start,
				Active:     true,
			})
			if err != nil {
				return nil, err
			}
			id := created.ID
			ruleID = &id
			source = "recurring"
		}
	}

	return s.assigns.Upsert(ctx, monthRow.ID, categoryID, currencyID, amount, source, ruleID)
}

// ToggleRecurring pausa o reanuda la regla recurrente de una categoría.
func (s *BudgetService) ToggleRecurring(ctx context.Context, ruleID int64, active bool) (*models.RecurringRule, error) {
	rule, err := s.rules.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("la regla recurrente no existe")
	}
	rule.Active = active
	return s.rules.Update(ctx, rule)
}

// UpdateRecurringAmount edita el monto de una regla (desde el mes actual).
func (s *BudgetService) UpdateRecurringAmount(ctx context.Context, ruleID int64, amount float64) (*models.RecurringRule, error) {
	if amount < 0 {
		return nil, fmt.Errorf("el monto no puede ser negativo")
	}
	rule, err := s.rules.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("la regla recurrente no existe")
	}
	now := time.Now()
	rule.Amount = amount
	rule.StartMonth = fmt.Sprintf("%04d-%02d", now.Year(), int(now.Month()))
	return s.rules.Update(ctx, rule)
}

// MaterializeRecurring genera los assignments del mes derivados de reglas
// activas (idempotente vía UNIQUE + upsert conservando asignaciones manuales).
func (s *BudgetService) MaterializeRecurring(ctx context.Context, year, month int) error {
	monthRow, err := s.months.GetOrCreate(ctx, year, month)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%04d-%02d", year, month)

	rules, err := s.rules.ListActive(ctx)
	if err != nil {
		return err
	}
	for i := range rules {
		rule := &rules[i]
		if !ruleAppliesToMonth(rule, key) {
			continue
		}
		cat, err := s.categories.GetByID(ctx, rule.CategoryID)
		if err != nil {
			return err
		}
		if cat == nil || cat.DeletedAt != nil {
			continue
		}
		// La moneda de la regla no se almacena: se usa la de la categoría si la
		// hubiera en assignments previos; si no, se ignora (requiere asignación).
		// Simplificación: las reglas usan la moneda de la primera asignación del
		// mes de inicio. Para la v1, la moneda se fija al crear la regla vía
		// Assign con la moneda provista. Aquí materializamos para cada moneda
		// en la que la categoría ya tenga asignaciones (para no inventar moneda).
		existing, err := s.assigns.ListByMonth(ctx, monthRow.ID)
		if err != nil {
			return err
		}
		currencyID := int64(0)
		for _, a := range existing {
			if a.CategoryID == rule.CategoryID {
				currencyID = a.CurrencyID
				break
			}
		}
		if currencyID == 0 {
			// Buscar la moneda en el mes de inicio de la regla (histórico).
			startYear, startMonth, err := parseYearMonth(rule.StartMonth)
			if err == nil {
				startMonthRow, err := s.months.GetByYearMonth(ctx, startYear, startMonth)
				if err == nil && startMonthRow != nil {
					startAssigns, err := s.assigns.ListByMonth(ctx, startMonthRow.ID)
					if err == nil {
						for _, a := range startAssigns {
							if a.CategoryID == rule.CategoryID {
								currencyID = a.CurrencyID
								break
							}
						}
					}
				}
			}
		}
		if currencyID == 0 {
			continue
		}
		if _, err := s.assigns.UpsertGenerated(ctx, monthRow.ID, rule.CategoryID, currencyID, rule.Amount, rule.ID); err != nil {
			return err
		}
	}
	return nil
}

// DeleteRule elimina una regla recurrente.
func (s *BudgetService) DeleteRule(ctx context.Context, ruleID int64) error {
	return s.rules.Delete(ctx, ruleID)
}

func (s *BudgetService) activeRuleForCategory(ctx context.Context, categoryID int64, year, month int) (*models.RecurringRule, error) {
	rules, err := s.rules.ListActiveByCategory(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%04d-%02d", year, month)
	for i := range rules {
		if ruleAppliesToMonth(&rules[i], key) {
			return &rules[i], nil
		}
	}
	return nil, nil
}

func ruleAppliesToMonth(rule *models.RecurringRule, key string) bool {
	if !rule.Active {
		return false
	}
	if rule.StartMonth != "" && key < rule.StartMonth {
		return false
	}
	if rule.EndMonth != "" && key > rule.EndMonth {
		return false
	}
	return true
}

func parseYearMonth(s string) (int, int, error) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return 0, 0, err
	}
	return t.Year(), int(t.Month()), nil
}

func (s *BudgetService) currencyCodesByID(ctx context.Context) (map[int64]models.Currency, error) {
	currencies, err := s.currencies.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]models.Currency)
	for i := range currencies {
		result[currencies[i].ID] = currencies[i]
	}
	return result, nil
}

func sumMap(m map[int64]float64) map[int64]float64 {
	if m == nil {
		return map[int64]float64{}
	}
	return m
}