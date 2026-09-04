package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// Estados de una deuda.
const (
	DebtStatusActive   = "activa"
	DebtStatusInactive = "inactiva"
	DebtStatusFinished = "finalizada"
)

// DebtService contiene la lógica de negocio para deudas (SPEC-054).
type DebtService struct {
	debts        *storage.DebtStorage
	debtBills    *storage.DebtBillStorage
	institutions *storage.InstitutionStorage
	currencies   *storage.CurrencyStorage
}

// NewDebtService crea un nuevo DebtService.
func NewDebtService(
	debts *storage.DebtStorage,
	debtBills *storage.DebtBillStorage,
	institutions *storage.InstitutionStorage,
	currencies *storage.CurrencyStorage,
) *DebtService {
	return &DebtService{
		debts:        debts,
		debtBills:    debtBills,
		institutions: institutions,
		currencies:   currencies,
	}
}

// List devuelve todas las deudas.
func (s *DebtService) List(ctx context.Context) ([]models.Debt, error) {
	return s.debts.List(ctx)
}

// GetByID busca una deuda por ID.
func (s *DebtService) GetByID(ctx context.Context, id int64) (*models.Debt, error) {
	return s.debts.GetByID(ctx, id)
}

// Create crea una deuda y genera su serie completa de cuotas si está activa.
func (s *DebtService) Create(ctx context.Context, debt *models.Debt) (*models.Debt, error) {
	if err := s.validate(ctx, debt); err != nil {
		return nil, err
	}
	if debt.InstallmentAmount <= 0 {
		debt.InstallmentAmount = computeInstallmentAmount(debt.Total, debt.InstallmentsTotal)
	}

	created, err := s.debts.Create(ctx, debt)
	if err != nil {
		return nil, err
	}

	if err := s.ensureDebtBills(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

// Update actualiza una deuda y regenera las cuotas faltantes si está activa.
func (s *DebtService) Update(ctx context.Context, debt *models.Debt) (*models.Debt, error) {
	if debt.ID == 0 {
		return nil, fmt.Errorf("id de deuda requerido")
	}
	if err := s.validate(ctx, debt); err != nil {
		return nil, err
	}
	if debt.InstallmentAmount <= 0 {
		debt.InstallmentAmount = computeInstallmentAmount(debt.Total, debt.InstallmentsTotal)
	}

	updated, err := s.debts.Update(ctx, debt)
	if err != nil {
		return nil, err
	}

	if err := s.ensureDebtBills(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete elimina lógicamente una deuda y sus cuotas.
func (s *DebtService) Delete(ctx context.Context, id int64) error {
	if err := s.debtBills.SoftDeleteByDebt(ctx, id); err != nil {
		return err
	}
	return s.debts.SoftDelete(ctx, id)
}

// ListBills devuelve las cuotas de una deuda.
func (s *DebtService) ListBills(ctx context.Context, debtID int64) ([]models.DebtBill, error) {
	return s.debtBills.ListByDebt(ctx, debtID)
}

// ListBillsByMonth devuelve las cuotas cuyo vencimiento cae en un mes
// (vista Calendario).
func (s *DebtService) ListBillsByMonth(ctx context.Context, year, month int) ([]models.DebtBill, error) {
	return s.debtBills.ListByMonth(ctx, year, month)
}

// PayBill marca una cuota como pagada.
func (s *DebtService) PayBill(ctx context.Context, id int64, paidAt time.Time, paymentReference string) (*models.DebtBill, error) {
	bill, err := s.debtBills.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if bill == nil {
		return nil, fmt.Errorf("la cuota no existe")
	}
	if bill.Status == "paid" {
		return nil, fmt.Errorf("la cuota ya está pagada")
	}
	return s.debtBills.Pay(ctx, id, paidAt, paymentReference)
}

// ReconcileDebtBills genera las cuotas faltantes de todas las deudas activas
// (patrón ReconcileBills de servicios).
func (s *DebtService) ReconcileDebtBills(ctx context.Context) error {
	debts, err := s.debts.List(ctx)
	if err != nil {
		return err
	}
	for i := range debts {
		if debts[i].Status == DebtStatusActive {
			if err := s.ensureDebtBills(ctx, &debts[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureDebtBills genera la serie completa de cuotas (1..N) que falten para
// una deuda activa. Dedup por (debt_id, installment_number).
func (s *DebtService) ensureDebtBills(ctx context.Context, debt *models.Debt) error {
	if debt.Status != DebtStatusActive {
		return nil
	}
	amount := debt.InstallmentAmount
	if amount <= 0 {
		amount = computeInstallmentAmount(debt.Total, debt.InstallmentsTotal)
	}

	start, err := time.Parse("2006-01-02", debt.StartDate)
	if err != nil {
		return fmt.Errorf("fecha de inicio inválida: %w", err)
	}

	for k := 1; k <= debt.InstallmentsTotal; k++ {
		existing, err := s.debtBills.FindByDebtInstallment(ctx, debt.ID, k)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}

		bill := &models.DebtBill{
			DebtID:            debt.ID,
			InstallmentNumber: k,
			DueDate:           dueDateForInstallment(start, k, debt.PaymentDay),
			Amount:            amount,
			Status:            "pending",
		}
		if _, err := s.debtBills.Create(ctx, bill); err != nil {
			return err
		}
	}
	return nil
}

// computeInstallmentAmount calcula el monto por cuota cuando no se especifica.
func computeInstallmentAmount(total float64, installments int) float64 {
	if installments <= 0 {
		return 0
	}
	return math.Round(total/float64(installments)*100) / 100
}

// dueDateForInstallment calcula el vencimiento de la cuota k (1-indexed):
// día de pago del mes (start + k meses), clampeado al último día del mes.
func dueDateForInstallment(start time.Time, k, paymentDay int) string {
	months := start.Month() + time.Month(k)
	year := start.Year() + (int(months)-1)/12
	month := time.Month((int(months)-1)%12 + 1)

	last := daysInMonth(year, month)
	day := paymentDay
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func (s *DebtService) validate(ctx context.Context, debt *models.Debt) error {
	debt.Identifier = strings.TrimSpace(debt.Identifier)
	debt.Description = strings.TrimSpace(debt.Description)
	debt.StartDate = strings.TrimSpace(debt.StartDate)
	debt.Status = strings.TrimSpace(debt.Status)
	if debt.Status == "" {
		debt.Status = DebtStatusActive
	}
	if debt.Status != DebtStatusActive && debt.Status != DebtStatusInactive && debt.Status != DebtStatusFinished {
		return fmt.Errorf("el estado debe ser 'activa', 'inactiva' o 'finalizada'")
	}
	if debt.InstitutionID == 0 {
		return fmt.Errorf("debe seleccionar un acreedor")
	}
	if debt.CurrencyID == 0 {
		return fmt.Errorf("debe seleccionar una moneda")
	}
	if debt.Description == "" {
		return fmt.Errorf("la descripción es requerida")
	}
	if debt.Total < 0 || debt.Principal < 0 || debt.InstallmentAmount < 0 || debt.InterestRate < 0 {
		return fmt.Errorf("los montos no pueden ser negativos")
	}
	if debt.InstallmentsTotal < 1 {
		return fmt.Errorf("el total de cuotas debe ser al menos 1")
	}
	if debt.PaymentDay < 1 || debt.PaymentDay > 31 {
		return fmt.Errorf("el día de pago debe estar entre 1 y 31")
	}
	if _, err := time.Parse("2006-01-02", debt.StartDate); err != nil {
		return fmt.Errorf("la fecha de inicio es requerida (formato YYYY-MM-DD)")
	}

	institution, err := s.institutions.GetByID(ctx, debt.InstitutionID)
	if err != nil {
		return fmt.Errorf("validar acreedor: %w", err)
	}
	if institution == nil {
		return fmt.Errorf("el acreedor seleccionado no existe")
	}

	currency, err := s.currencies.GetByID(ctx, debt.CurrencyID)
	if err != nil {
		return fmt.Errorf("validar moneda: %w", err)
	}
	if currency == nil {
		return fmt.Errorf("la moneda seleccionada no existe")
	}

	return nil
}
