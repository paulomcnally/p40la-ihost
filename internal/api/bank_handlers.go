package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// BankHandlers agrupa los endpoints del módulo de bancos (SPEC-133).
type BankHandlers struct {
	service *services.BankService
}

// NewBankHandlers crea un nuevo BankHandlers.
func NewBankHandlers(service *services.BankService) *BankHandlers {
	return &BankHandlers{service: service}
}

type bankRequest struct {
	Name      string `json:"name"`
	ShortCode string `json:"short_code"`
	Country   string `json:"country"`
	Notes     string `json:"notes"`
	Active    *bool  `json:"active"`
}

type bankAccountRequest struct {
	Alias         string `json:"alias"`
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	CurrencyID    int64  `json:"currency_id"`
	Beneficiary   string `json:"beneficiary"`
	Notes         string `json:"notes"`
	Active        *bool  `json:"active"`
}

type bankTransferFormatRequest struct {
	Fields map[string]string `json:"fields"`
}

func (h *BankHandlers) ListBanks(w http.ResponseWriter, r *http.Request) {
	banks, err := h.service.ListBanks(r.Context())
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, banks)
}

func (h *BankHandlers) GetBank(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	bank, err := h.service.GetBank(r.Context(), id)
	if err != nil {
		respondBankError(w, err)
		return
	}
	if bank == nil {
		respondBankError(w, services.ErrBankNotFound)
		return
	}
	respondJSON(w, http.StatusOK, bank)
}

func (h *BankHandlers) CreateBank(w http.ResponseWriter, r *http.Request) {
	var req bankRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	bank, err := h.service.CreateBank(r.Context(), &models.Bank{
		Name:      req.Name,
		ShortCode: req.ShortCode,
		Country:   req.Country,
		Notes:     req.Notes,
		Active:    boolValue(req.Active, true),
	})
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, bank)
}

func (h *BankHandlers) UpdateBank(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req bankRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	bank, err := h.service.UpdateBank(r.Context(), &models.Bank{
		ID:        id,
		Name:      req.Name,
		ShortCode: req.ShortCode,
		Country:   req.Country,
		Notes:     req.Notes,
		Active:    boolValue(req.Active, true),
	})
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, bank)
}

func (h *BankHandlers) DeleteBank(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteBank(r.Context(), id); err != nil {
		respondBankError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BankHandlers) ListAccounts(w http.ResponseWriter, r *http.Request) {
	bankID, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	accounts, err := h.service.ListAccounts(r.Context(), bankID)
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, accounts)
}

func (h *BankHandlers) CreateAccount(w http.ResponseWriter, r *http.Request) {
	bankID, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req bankAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	account, err := h.service.CreateAccount(r.Context(), &models.BankAccount{
		BankID:        bankID,
		Alias:         req.Alias,
		AccountNumber: req.AccountNumber,
		AccountType:   req.AccountType,
		CurrencyID:    req.CurrencyID,
		Beneficiary:   req.Beneficiary,
		Notes:         req.Notes,
		Active:        boolValue(req.Active, true),
	})
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, account)
}

func (h *BankHandlers) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	account, err := h.service.GetAccount(r.Context(), id)
	if err != nil {
		respondBankError(w, err)
		return
	}
	if account == nil {
		respondBankError(w, services.ErrBankAccountNotFound)
		return
	}
	respondJSON(w, http.StatusOK, account)
}

func (h *BankHandlers) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req bankAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	account, err := h.service.UpdateAccount(r.Context(), &models.BankAccount{
		ID:            id,
		Alias:         req.Alias,
		AccountNumber: req.AccountNumber,
		AccountType:   req.AccountType,
		CurrencyID:    req.CurrencyID,
		Beneficiary:   req.Beneficiary,
		Notes:         req.Notes,
		Active:        boolValue(req.Active, true),
	})
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, account)
}

func (h *BankHandlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteAccount(r.Context(), id); err != nil {
		respondBankError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BankHandlers) ListFormats(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	formats, err := h.service.ListFormats(r.Context(), accountID)
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, formats)
}

func (h *BankHandlers) UpsertFormat(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req bankTransferFormatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	format, err := h.service.UpsertFormat(r.Context(), accountID, r.PathValue("kind"), req.Fields)
	if err != nil {
		respondBankError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, format)
}

func (h *BankHandlers) DeleteFormat(w http.ResponseWriter, r *http.Request) {
	accountID, err := parseID(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteFormat(r.Context(), accountID, r.PathValue("kind")); err != nil {
		respondBankError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BankHandlers) ListVariables(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, h.service.Variables())
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func respondBankError(w http.ResponseWriter, err error) {
	var validation *services.BankValidationError
	if errors.As(err, &validation) {
		code := validation.Code
		if code == "" {
			code = "validation_error"
		}
		respondError(w, http.StatusBadRequest, code, validation.Message)
		return
	}

	switch {
	case errors.Is(err, services.ErrBankNotFound):
		respondError(w, http.StatusNotFound, "bank_not_found", "Banco no encontrado")
	case errors.Is(err, services.ErrBankAccountNotFound):
		respondError(w, http.StatusNotFound, "account_not_found", "Cuenta bancaria no encontrada")
	case errors.Is(err, services.ErrBankTransferFormatNotFound):
		respondError(w, http.StatusNotFound, "format_not_found", "Formato de transferencia no encontrado")
	case errors.Is(err, services.ErrBankNameExists):
		respondError(w, http.StatusConflict, "bank_name_exists", "Ya existe un banco con ese nombre")
	case errors.Is(err, services.ErrBankHasAccounts):
		respondError(w, http.StatusConflict, "bank_has_accounts", "El banco tiene cuentas asociadas")
	case errors.Is(err, services.ErrBankAccountDuplicate):
		respondError(w, http.StatusConflict, "duplicate_account", "Ya existe una cuenta con ese número en el banco")
	default:
		respondError(w, http.StatusInternalServerError, "internal_error", "Error interno")
	}
}
