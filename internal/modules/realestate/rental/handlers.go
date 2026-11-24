package rental

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type handlers struct {
	service *Service
}

// RegisterRoutes monta los endpoints del módulo.
func (h *handlers) register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/rental/people", auth(http.HandlerFunc(h.listPeople)))
	mux.Handle("POST /api/rental/people", auth(http.HandlerFunc(h.createPerson)))
	mux.Handle("GET /api/rental/people/{id}", auth(http.HandlerFunc(h.getPerson)))
	mux.Handle("PUT /api/rental/people/{id}", auth(http.HandlerFunc(h.updatePerson)))
	mux.Handle("DELETE /api/rental/people/{id}", auth(http.HandlerFunc(h.deletePerson)))

	mux.Handle("GET /api/rental/homes/{id}", auth(http.HandlerFunc(h.getHomeStatus)))

	mux.Handle("GET /api/rental/leases", auth(http.HandlerFunc(h.listLeases)))
	mux.Handle("POST /api/rental/leases", auth(http.HandlerFunc(h.createLease)))
	mux.Handle("GET /api/rental/leases/{id}", auth(http.HandlerFunc(h.getLease)))
	mux.Handle("PUT /api/rental/leases/{id}", auth(http.HandlerFunc(h.updateLease)))
	mux.Handle("DELETE /api/rental/leases/{id}", auth(http.HandlerFunc(h.deleteLease)))
	mux.Handle("POST /api/rental/leases/{id}/activate", auth(http.HandlerFunc(h.activateLease)))
	mux.Handle("POST /api/rental/leases/{id}/terminate", auth(http.HandlerFunc(h.terminateLease)))
	mux.Handle("GET /api/rental/leases/{id}/balance", auth(http.HandlerFunc(h.balance)))
	mux.Handle("POST /api/rental/leases/{id}/parties", auth(http.HandlerFunc(h.addParty)))
	mux.Handle("DELETE /api/rental/leases/{id}/parties/{partyID}", auth(http.HandlerFunc(h.deleteParty)))
	mux.Handle("GET /api/rental/leases/{id}/charges", auth(http.HandlerFunc(h.listCharges)))
	mux.Handle("POST /api/rental/leases/{id}/charges", auth(http.HandlerFunc(h.generateCharges)))
	mux.Handle("GET /api/rental/leases/{id}/documents", auth(http.HandlerFunc(h.listDocuments)))
	mux.Handle("POST /api/rental/leases/{id}/documents", auth(http.HandlerFunc(h.uploadDocument)))
	mux.Handle("GET /api/rental/leases/{id}/documents/{docID}", auth(http.HandlerFunc(h.downloadDocument)))
	mux.Handle("DELETE /api/rental/leases/{id}/documents/{docID}", auth(http.HandlerFunc(h.deleteDocument)))
	mux.Handle("GET /api/rental/leases/{id}/reminders", auth(http.HandlerFunc(h.listReminders)))
	mux.Handle("POST /api/rental/leases/{id}/reminders", auth(http.HandlerFunc(h.createLeaseReminder)))

	mux.Handle("GET /api/rental/reservations", auth(http.HandlerFunc(h.listReservations)))
	mux.Handle("POST /api/rental/reservations", auth(http.HandlerFunc(h.createReservation)))
	mux.Handle("GET /api/rental/reservations/{id}", auth(http.HandlerFunc(h.getReservation)))
	mux.Handle("PUT /api/rental/reservations/{id}", auth(http.HandlerFunc(h.updateReservation)))
	mux.Handle("POST /api/rental/reservations/{id}/convert", auth(http.HandlerFunc(h.convertReservation)))
	mux.Handle("POST /api/rental/reservations/{id}/cancel", auth(http.HandlerFunc(h.cancelReservation)))
	mux.Handle("POST /api/rental/reservations/{id}/forfeit", auth(http.HandlerFunc(h.forfeitReservation)))
	mux.Handle("GET /api/rental/reservations/{id}/pdf", auth(http.HandlerFunc(h.reservationPDF)))

	mux.Handle("GET /api/rental/invoices", auth(http.HandlerFunc(h.listInvoices)))
	mux.Handle("POST /api/rental/invoices", auth(http.HandlerFunc(h.issueInvoice)))
	mux.Handle("GET /api/rental/invoices/{id}", auth(http.HandlerFunc(h.getInvoice)))
	mux.Handle("GET /api/rental/invoices/{id}/pdf", auth(http.HandlerFunc(h.invoicePDF)))
	mux.Handle("POST /api/rental/invoices/{id}/send", auth(http.HandlerFunc(h.sendInvoice)))
	mux.Handle("POST /api/rental/invoices/{id}/void", auth(http.HandlerFunc(h.voidInvoice)))

	mux.Handle("GET /api/rental/payments", auth(http.HandlerFunc(h.listPayments)))
	mux.Handle("POST /api/rental/payments", auth(http.HandlerFunc(h.createPayment)))
	mux.Handle("GET /api/rental/payments/{id}/receipt", auth(http.HandlerFunc(h.createReceipt)))
	mux.Handle("GET /api/rental/payments/{id}/receipt.pdf", auth(http.HandlerFunc(h.receiptPDF)))
	mux.Handle("POST /api/rental/payments/{id}/proofs", auth(http.HandlerFunc(h.uploadProof)))
	mux.Handle("GET /api/rental/payments/{id}/proofs/{proofID}", auth(http.HandlerFunc(h.downloadProof)))
	mux.Handle("DELETE /api/rental/payments/{id}/proofs/{proofID}", auth(http.HandlerFunc(h.deleteProof)))

	mux.Handle("GET /api/rental/reminders", auth(http.HandlerFunc(h.listReminders)))
	mux.Handle("POST /api/rental/reminders", auth(http.HandlerFunc(h.createReminder)))
	mux.Handle("PUT /api/rental/reminders/{id}", auth(http.HandlerFunc(h.updateReminder)))
	mux.Handle("DELETE /api/rental/reminders/{id}", auth(http.HandlerFunc(h.deleteReminder)))
	mux.Handle("POST /api/rental/reminders/{id}/run", auth(http.HandlerFunc(h.runReminder)))

	mux.Handle("GET /api/rental/settings", auth(http.HandlerFunc(h.getSettings)))
	mux.Handle("PUT /api/rental/settings", auth(http.HandlerFunc(h.updateSettings)))
}

// ------------------------------------------------------------------ personas

func (h *handlers) listPeople(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPeople(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []Person{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) getPerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.GetPerson(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) createPerson(w http.ResponseWriter, r *http.Request) {
	var req Person
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.CreatePerson(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) updatePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Person
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.ID = id
	item, err := h.service.UpdatePerson(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) deletePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeletePerson(r.Context(), id); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ----------------------------------------------------------------- vivienda

func (h *handlers) getHomeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.GetHomeStatus(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

// ----------------------------------------------------------------- contratos

func (h *handlers) listLeases(w http.ResponseWriter, r *http.Request) {
	homeID, _ := strconv.ParseInt(r.URL.Query().Get("home_id"), 10, 64)
	items, err := h.service.ListLeases(r.Context(), r.URL.Query().Get("status"), homeID)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []Lease{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) getLease(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.GetLease(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) createLease(w http.ResponseWriter, r *http.Request) {
	var req Lease
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.CreateLease(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) updateLease(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Lease
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.ID = id
	item, err := h.service.UpdateLease(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) deleteLease(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteLease(r.Context(), id); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) activateLease(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.ActivateLease(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) terminateLease(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req struct {
		Reason  string `json:"reason"`
		EndDate string `json:"end_date"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.TerminateLease(r.Context(), id, req.Reason, req.EndDate)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) balance(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.Balance(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) addParty(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req LeaseParty
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.AddParty(r.Context(), id, &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) deleteParty(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	partyID, err := pathID(r, "partyID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteParty(r.Context(), id, partyID); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) listCharges(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	items, err := h.service.ListCharges(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []Charge{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) generateCharges(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	created, err := h.service.GenerateCharges(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]int{"created": created})
}

// --------------------------------------------------------------- apartados

func (h *handlers) listReservations(w http.ResponseWriter, r *http.Request) {
	homeID, _ := strconv.ParseInt(r.URL.Query().Get("home_id"), 10, 64)
	items, err := h.service.ListReservations(r.Context(), homeID, r.URL.Query().Get("status"))
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []Reservation{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) getReservation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.GetReservation(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) createReservation(w http.ResponseWriter, r *http.Request) {
	var req Reservation
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.CreateReservation(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) updateReservation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Reservation
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.ID = id
	item, err := h.service.UpdateReservation(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) convertReservation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req ConvertReservationInput
	_ = decodeJSON(r, &req)
	item, err := h.service.ConvertReservation(r.Context(), id, req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) cancelReservation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = decodeJSON(r, &req)
	if err := h.service.CancelReservation(r.Context(), id, req.Reason); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) forfeitReservation(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = decodeJSON(r, &req)
	if err := h.service.ForfeitReservation(r.Context(), id, req.Reason); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) reservationPDF(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	data, name, err := h.service.ReservationPDF(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	servePDF(w, r, data, name)
}

// ----------------------------------------------------------------- facturas

func (h *handlers) listInvoices(w http.ResponseWriter, r *http.Request) {
	leaseID, _ := strconv.ParseInt(r.URL.Query().Get("lease_id"), 10, 64)
	items, err := h.service.ListInvoices(r.Context(), leaseID, r.URL.Query().Get("status"))
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []Invoice{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) getInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.GetInvoice(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) issueInvoice(w http.ResponseWriter, r *http.Request) {
	var req IssueInvoiceInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.IssueInvoice(r.Context(), req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) invoicePDF(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	data, name, err := h.service.InvoicePDF(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	servePDF(w, r, data, name)
}

func (h *handlers) sendInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	count, err := h.service.SendInvoice(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]int{"recipients": count})
}

func (h *handlers) voidInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.VoidInvoice(r.Context(), id, req.Reason)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

// -------------------------------------------------------------------- pagos

func (h *handlers) listPayments(w http.ResponseWriter, r *http.Request) {
	leaseID, _ := strconv.ParseInt(r.URL.Query().Get("lease_id"), 10, 64)
	invoiceID, _ := strconv.ParseInt(r.URL.Query().Get("invoice_id"), 10, 64)
	items, err := h.service.ListPayments(r.Context(), leaseID, invoiceID)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []Payment{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) createPayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.CreatePayment(r.Context(), req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) createReceipt(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	item, err := h.service.CreateReceipt(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) receiptPDF(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	receipt, err := h.service.storage.GetReceiptByPayment(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if receipt == nil {
		respondError(w, http.StatusNotFound, "receipt_not_found", "El pago no tiene recibo")
		return
	}
	data, name, err := h.service.ReceiptPDF(r.Context(), receipt.ID)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	servePDF(w, r, data, name)
}

func (h *handlers) uploadProof(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	upload, err := readUpload(w, r)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	item, err := h.service.UploadProof(r.Context(), id, upload)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) downloadProof(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	proofID, err := pathID(r, "proofID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	proof, path, err := h.service.ProofPath(r.Context(), id, proofID)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	serveFile(w, r, path, proof.OriginalName, proof.ContentType)
}

func (h *handlers) deleteProof(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	proofID, err := pathID(r, "proofID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteProof(r.Context(), id, proofID); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -------------------------------------------------------------- documentos

func (h *handlers) listDocuments(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	items, err := h.service.ListDocuments(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []LeaseDocument{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) uploadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	upload, err := readUpload(w, r)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	docType := strings.TrimSpace(r.FormValue("doc_type"))
	item, err := h.service.UploadDocument(r.Context(), id, docType, upload)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) downloadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	docID, err := pathID(r, "docID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	doc, path, err := h.service.DocumentPath(r.Context(), id, docID)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	serveFile(w, r, path, doc.OriginalName, doc.ContentType)
}

func (h *handlers) deleteDocument(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	docID, err := pathID(r, "docID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteDocument(r.Context(), id, docID); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ------------------------------------------------------------ recordatorios

func (h *handlers) listReminders(w http.ResponseWriter, r *http.Request) {
	homeID, _ := strconv.ParseInt(r.URL.Query().Get("home_id"), 10, 64)
	leaseID, _ := strconv.ParseInt(r.URL.Query().Get("lease_id"), 10, 64)
	if id := r.PathValue("id"); id != "" {
		leaseID, _ = strconv.ParseInt(id, 10, 64)
	}
	items, err := h.service.ListReminders(r.Context(), homeID, leaseID)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	if items == nil {
		items = []Reminder{}
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *handlers) createReminder(w http.ResponseWriter, r *http.Request) {
	var req Reminder
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.CreateReminder(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) createLeaseReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Reminder
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.LeaseID = &id
	item, err := h.service.CreateReminder(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) updateReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Reminder
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.ID = id
	item, err := h.service.UpdateReminder(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) deleteReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteReminder(r.Context(), id); err != nil {
		respondRentalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) runReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	count, err := h.service.RunReminderNow(r.Context(), id)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]int{"recipients": count})
}

// ---------------------------------------------------------------- settings

func (h *handlers) getSettings(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetSettings(r.Context())
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req RentalSettings
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.UpdateSettings(r.Context(), &req)
	if err != nil {
		respondRentalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

// ------------------------------------------------------------------ helpers

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
}

func readUpload(w http.ResponseWriter, r *http.Request) (FileUpload, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRentalAttachmentBytes+1024*1024)
	file, header, err := r.FormFile("file")
	if err != nil {
		return FileUpload{}, invalid("invalid_file", "No se encontró el archivo")
	}
	return FileUpload{FileName: header.Filename, Reader: file}, nil
}

func servePDF(w http.ResponseWriter, r *http.Request, data []byte, name string) {
	w.Header().Set("Content-Type", "application/pdf")
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", disposition+`; filename="`+strings.ReplaceAll(name, `"`, "")+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

func serveFile(w http.ResponseWriter, r *http.Request, path, name, contentType string) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", disposition+`; filename="`+strings.ReplaceAll(name, `"`, "")+`"`)
	http.ServeFile(w, r, path)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, errorBody{Error: code, Message: message})
}

func respondRentalError(w http.ResponseWriter, err error) {
	var validation *ValidationError
	if errors.As(err, &validation) {
		status := http.StatusUnprocessableEntity
		if validation.Code == "invalid_request" || validation.Code == "invalid_file" {
			status = http.StatusBadRequest
		}
		if validation.Code == "already_sent" {
			status = http.StatusConflict
		}
		respondError(w, status, validation.Code, validation.Message)
		return
	}
	switch {
	case errors.Is(err, ErrPersonNotFound):
		respondError(w, http.StatusNotFound, "person_not_found", "Persona no encontrada")
	case errors.Is(err, ErrLeaseNotFound):
		respondError(w, http.StatusNotFound, "lease_not_found", "Contrato no encontrado")
	case errors.Is(err, ErrReservationNotFound):
		respondError(w, http.StatusNotFound, "reservation_not_found", "Apartado no encontrado")
	case errors.Is(err, ErrHomeNotFound):
		respondError(w, http.StatusNotFound, "home_not_found", "Vivienda no encontrada")
	case errors.Is(err, ErrAgencyNotFound):
		respondError(w, http.StatusNotFound, "agency_not_found", "Inmobiliaria no encontrada")
	case errors.Is(err, ErrChargeNotFound):
		respondError(w, http.StatusNotFound, "charge_not_found", "Cargo no encontrado")
	case errors.Is(err, ErrInvoiceNotFound):
		respondError(w, http.StatusNotFound, "invoice_not_found", "Factura no encontrada")
	case errors.Is(err, ErrPaymentNotFound):
		respondError(w, http.StatusNotFound, "payment_not_found", "Pago no encontrado")
	case errors.Is(err, ErrReceiptNotFound):
		respondError(w, http.StatusNotFound, "receipt_not_found", "Recibo no encontrado")
	case errors.Is(err, ErrDocumentNotFound):
		respondError(w, http.StatusNotFound, "document_not_found", "Documento no encontrado")
	case errors.Is(err, ErrProofNotFound):
		respondError(w, http.StatusNotFound, "proof_not_found", "Comprobante no encontrado")
	case errors.Is(err, ErrReminderNotFound):
		respondError(w, http.StatusNotFound, "reminder_not_found", "Recordatorio no encontrado")
	case errors.Is(err, ErrHomeUnavailable):
		respondError(w, http.StatusConflict, "home_not_available", "La vivienda ya tiene un apartado o contrato vigente")
	case errors.Is(err, ErrHomeNotAvailable):
		respondError(w, http.StatusConflict, "home_not_available", "La vivienda no está disponible para alquilar")
	case errors.Is(err, ErrLeaseNotDraft):
		respondError(w, http.StatusConflict, "lease_not_draft", "Solo se puede editar un contrato en borrador")
	case errors.Is(err, ErrInvoiceImmutable):
		respondError(w, http.StatusConflict, "invoice_immutable", "La factura ya no admite cambios")
	case errors.Is(err, ErrDocumentLimit):
		respondError(w, http.StatusUnprocessableEntity, "document_limit", "Se alcanzó el máximo de documentos del contrato")
	case errors.Is(err, ErrProofLimit):
		respondError(w, http.StatusUnprocessableEntity, "proof_limit", "Se alcanzó el máximo de comprobantes del pago")
	default:
		respondError(w, http.StatusInternalServerError, "internal_error", "Error interno")
	}
}
