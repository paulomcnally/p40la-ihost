package services

import (
	"context"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// diffBillChanges calcula los campos de negocio modificados entre una factura
// y su estado anterior (SPEC-070). Ignora no-ops y metadatos internos
// (file_hash, deleted_at, timestamps).
func diffBillChanges(oldBill, newBill *models.Bill) []models.FieldChange {
	var changes []models.FieldChange
	add := func(field string, oldV, newV any) {
		if oldV != newV {
			changes = append(changes, models.FieldChange{Field: field, Old: oldV, New: newV})
		}
	}

	add("amount", oldBill.Amount, newBill.Amount)
	add("invoice_number", oldBill.InvoiceNumber, newBill.InvoiceNumber)
	add("status", oldBill.Status, newBill.Status)
	add("drive_url", oldBill.DriveURL, newBill.DriveURL)
	add("year", oldBill.Year, newBill.Year)
	add("month", oldBill.Month, newBill.Month)
	add("payment_reference", oldBill.PaymentReference, newBill.PaymentReference)

	if oldBill.PaidAt != nil && newBill.PaidAt != nil {
		add("paid_at", oldBill.PaidAt.Format("2006-01-02 15:04:05"), newBill.PaidAt.Format("2006-01-02 15:04:05"))
	} else if oldBill.PaidAt != nil {
		add("paid_at", oldBill.PaidAt.Format("2006-01-02 15:04:05"), nil)
	} else if newBill.PaidAt != nil {
		add("paid_at", nil, newBill.PaidAt.Format("2006-01-02 15:04:05"))
	}

	return changes
}

// recordBillHistory persiste un evento de auditoría de una factura (SPEC-070).
// Si el storage no está configurado, es un no-op (no rompe flujos existentes).
// Los eventos "updated" sin cambios reales no se registran.
func recordBillHistory(ctx context.Context, h *storage.BillHistoryStorage, billID int64, action, source string, changes []models.FieldChange) error {
	if h == nil {
		return nil
	}
	if action == models.BillActionUpdated && len(changes) == 0 {
		return nil
	}
	_, err := h.Record(ctx, &models.BillHistory{
		BillID:  billID,
		Action:  action,
		Source:  source,
		Changes: changes,
	})
	return err
}