package creditcard

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

type handlers struct {
	service *Service
}

func (h *handlers) listCards(w http.ResponseWriter, r *http.Request) {
	bankID, err := pathID(r, "bank_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de banco inválido")
		return
	}
	cards, err := h.service.ListCards(r.Context(), bankID)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	if cards == nil {
		cards = []CreditCard{}
	}
	respondJSON(w, http.StatusOK, cards)
}

func (h *handlers) createCard(w http.ResponseWriter, r *http.Request) {
	bankID, err := pathID(r, "bank_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de banco inválido")
		return
	}
	var req CreditCard
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.BankID = bankID
	card, err := h.service.CreateCard(r.Context(), &req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, card)
}

func (h *handlers) getCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	card, err := h.service.GetCard(r.Context(), id)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, card)
}

func (h *handlers) updateCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req CreditCard
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.ID = id
	card, err := h.service.UpdateCard(r.Context(), &req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, card)
}

func (h *handlers) deleteCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteCard(r.Context(), id); err != nil {
		respondModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) listStatements(w http.ResponseWriter, r *http.Request) {
	cardID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	statements, err := h.service.ListStatements(r.Context(), cardID)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	if statements == nil {
		statements = []Statement{}
	}
	respondJSON(w, http.StatusOK, statements)
}

func (h *handlers) createStatement(w http.ResponseWriter, r *http.Request) {
	cardID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Statement
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.CardID = cardID
	req.Source = SourceManual
	req.SourceModuleID = ""
	req.ImportHash = ""
	statement, err := h.service.CreateStatement(r.Context(), &req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, statement)
}

func (h *handlers) getStatement(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	statement, err := h.service.GetStatement(r.Context(), id)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, statement)
}

func (h *handlers) updateStatement(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Statement
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.ID = id
	statement, err := h.service.UpdateStatement(r.Context(), &req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, statement)
}

func (h *handlers) deleteStatement(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteStatement(r.Context(), id); err != nil {
		respondModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) listTransactions(w http.ResponseWriter, r *http.Request) {
	statementID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	transactions, err := h.service.ListTransactions(r.Context(), statementID, r.URL.Query().Get("type"), r.URL.Query().Get("q"))
	if err != nil {
		respondModuleError(w, err)
		return
	}
	if transactions == nil {
		transactions = []Transaction{}
	}
	respondJSON(w, http.StatusOK, transactions)
}

func (h *handlers) createTransaction(w http.ResponseWriter, r *http.Request) {
	statementID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Transaction
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	transaction, err := h.service.CreateTransaction(r.Context(), statementID, &req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, transaction)
}

func (h *handlers) updateTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Transaction
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	req.ID = id
	transaction, err := h.service.UpdateTransaction(r.Context(), &req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, transaction)
}

func (h *handlers) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteTransaction(r.Context(), id); err != nil {
		respondModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) previewImport(w http.ResponseWriter, r *http.Request) {
	cardID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+1024)
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	defer file.Close()

	if header.Size > maxFileSize {
		respondError(w, http.StatusBadRequest, "invalid_file", "El archivo supera el máximo de 5 MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" {
		respondError(w, http.StatusBadRequest, "invalid_file", "Solo se aceptan archivos PDF")
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer el archivo")
		return
	}
	if len(data) > maxFileSize {
		respondError(w, http.StatusBadRequest, "invalid_file", "El archivo supera el máximo de 5 MB")
		return
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		respondError(w, http.StatusBadRequest, "invalid_file", "El archivo no es un PDF válido")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	parsed, hash, err := h.service.PreviewImport(r.Context(), cardID, header.Filename, mimeType, data)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, previewResponse{ParsedStatement: parsed, PreviewHash: hash})
}

func (h *handlers) confirmImport(w http.ResponseWriter, r *http.Request) {
	cardID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	statement, err := h.service.ConfirmImport(r.Context(), cardID, &req.ParsedStatement, req.PreviewHash)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, statement)
}

type previewResponse struct {
	*models.ParsedStatement
	PreviewHash string `json:"preview_hash"`
}

type confirmRequest struct {
	models.ParsedStatement
	PreviewHash string `json:"preview_hash"`
}

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
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

func respondModuleError(w http.ResponseWriter, err error) {
	var validation *ValidationError
	if errors.As(err, &validation) {
		code := validation.Code
		if code == "" {
			code = "validation_error"
		}
		status := http.StatusBadRequest
		if code == "unsupported_statement" || code == "card_mismatch" {
			status = http.StatusUnprocessableEntity
		}
		respondError(w, status, code, validation.Message)
		return
	}

	switch {
	case errors.Is(err, ErrBankNotFound):
		respondError(w, http.StatusNotFound, "bank_not_found", "Banco no encontrado")
	case errors.Is(err, ErrCardNotFound):
		respondError(w, http.StatusNotFound, "credit_card_not_found", "Tarjeta de crédito no encontrada")
	case errors.Is(err, ErrStatementNotFound):
		respondError(w, http.StatusNotFound, "statement_not_found", "Estado de cuenta no encontrado")
	case errors.Is(err, ErrTransactionNotFound):
		respondError(w, http.StatusNotFound, "transaction_not_found", "Movimiento no encontrado")
	case errors.Is(err, ErrDuplicateCard):
		respondError(w, http.StatusConflict, "duplicate_card", "Ya existe una tarjeta con ese alias en el banco")
	case errors.Is(err, ErrStatementExists):
		respondError(w, http.StatusConflict, "statement_exists", "Ya existe un estado de cuenta para esa fecha de corte")
	case errors.Is(err, ErrCardMismatch):
		respondError(w, http.StatusUnprocessableEntity, "card_mismatch", "El estado de cuenta no corresponde a la tarjeta")
	case errors.Is(err, ErrUnsupportedStatement):
		respondError(w, http.StatusUnprocessableEntity, "unsupported_statement",
			"Ningún importador reconoce este PDF: descargá el estado de cuenta del banco (no el resumen de transacciones) y verificá que el banco esté soportado")
	case errors.Is(err, ErrPreviewMismatch):
		respondError(w, http.StatusBadRequest, "preview_mismatch", "Los datos cambiaron desde el preview")
	case errors.Is(err, ErrInvalidFile):
		respondError(w, http.StatusBadRequest, "invalid_file", "Archivo inválido")
	default:
		respondError(w, http.StatusInternalServerError, "internal_error", "Error interno")
	}
}
