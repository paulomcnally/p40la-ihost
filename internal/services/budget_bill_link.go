package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// BillBudgetLinker es el contrato que BillService usa para notificar el pago
// de una factura y dejar que el módulo de presupuesto genere su transacción
// automática (SPEC-094). Se inyecta con SetBillBudgetLinker; si no se configura,
// el flujo de pago no cambia.
type BillBudgetLinker interface {
	// OnBillPaid crea la transacción de presupuesto para una factura pagada de
	// un servicio vinculado a una categoría. Devuelve error solo si la creación
	// falla (el pago ya está commiteado); la idempotencia evita duplicados.
	OnBillPaid(ctx context.Context, bill *models.Bill) error
}

// BudgetBillLinkService orquesta el vínculo categoría ↔ servicio: al pagar una
// factura de un servicio vinculado, crea la transacción de presupuesto
// correspondiente (SPEC-094).
type BudgetBillLinkService struct {
	services     *storage.ServiceStorage
	categories   *storage.CategoryStorage
	accounts     *storage.AccountStorage
	transactions *storage.TransactionStorage
}

// NewBudgetBillLinkService crea un nuevo BudgetBillLinkService.
func NewBudgetBillLinkService(
	services *storage.ServiceStorage,
	categories *storage.CategoryStorage,
	accounts *storage.AccountStorage,
	transactions *storage.TransactionStorage,
) *BudgetBillLinkService {
	return &BudgetBillLinkService{
		services:     services,
		categories:   categories,
		accounts:     accounts,
		transactions: transactions,
	}
}

// OnBillPaid genera la transacción de presupuesto para una factura pagada
// (SPEC-094). No-op si el servicio no tiene categoría vinculada activa, si la
// cuenta configurada ya no existe o si la transacción ya fue creada
// (idempotencia por source_bill_id).
func (s *BudgetBillLinkService) OnBillPaid(ctx context.Context, bill *models.Bill) error {
	if bill == nil {
		return nil
	}
	if bill.Amount < 0 {
		return fmt.Errorf("monto de factura negativo: %d", bill.ID)
	}

	// Idempotencia: si la transacción para esta factura ya existe, no duplicar.
	existing, err := s.transactions.GetBySourceBill(ctx, bill.ID)
	if err != nil {
		return fmt.Errorf("verificar transacción de factura %d: %w", bill.ID, err)
	}
	if existing != nil {
		return nil
	}

	service, err := s.services.GetByID(ctx, bill.ServiceID)
	if err != nil {
		return fmt.Errorf("obtener servicio %d: %w", bill.ServiceID, err)
	}
	if service == nil {
		return fmt.Errorf("servicio %d no existe", bill.ServiceID)
	}

	category, err := s.categories.GetByServiceID(ctx, service.ID)
	if err != nil {
		return fmt.Errorf("buscar categoría vinculada al servicio %d: %w", service.ID, err)
	}
	if category == nil {
		// Servicio sin categoría vinculada: no generar transacción.
		return nil
	}
	if category.AccountID == nil {
		slog.Warn("categoría vinculada sin cuenta configurada",
			"category_id", category.ID, "service_id", service.ID, "bill_id", bill.ID)
		return nil
	}

	account, err := s.accounts.GetByID(ctx, *category.AccountID)
	if err != nil {
		return fmt.Errorf("obtener cuenta %d: %w", *category.AccountID, err)
	}
	if account == nil {
		slog.Warn("cuenta de la categoría vinculada ya no existe",
			"category_id", category.ID, "account_id", *category.AccountID, "bill_id", bill.ID)
		return nil
	}

	paidAt := time.Now()
	if bill.PaidAt != nil {
		paidAt = *bill.PaidAt
	}

	tx := &models.Transaction{
		AccountID:    account.ID,
		CategoryID:   &category.ID,
		CurrencyID:   service.CurrencyID,
		Date:         paidAt.Format("2006-01-02"),
		Payee:        service.Name,
		Memo:         billMemo(bill, service.Name),
		Outflow:      bill.Amount,
		Inflow:       0,
		Cleared:      true,
		SourceBillID: &bill.ID,
	}
	if _, err := s.transactions.Create(ctx, tx); err != nil {
		return fmt.Errorf("crear transacción de presupuesto para factura %d: %w", bill.ID, err)
	}
	return nil
}

// billMemo arma el memo de la transacción con la referencia de la factura.
func billMemo(bill *models.Bill, serviceName string) string {
	ref := ""
	if bill.InvoiceNumber != "" {
		ref = fmt.Sprintf("Factura #%s", bill.InvoiceNumber)
	} else {
		period := fmt.Sprintf("%04d-%02d", bill.Year, bill.Month)
		if bill.Month == 0 {
			period = fmt.Sprintf("%d", bill.Year)
		}
		ref = "Factura " + period
	}
	return fmt.Sprintf("%s — %s", serviceName, ref)
}
