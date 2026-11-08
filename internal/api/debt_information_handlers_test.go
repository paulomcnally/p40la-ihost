package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func setupDebtInformationHandlers(t *testing.T) (*DebtInformationHandlers, *sql.DB) {
	t.Helper()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(`INSERT INTO institutions (name) VALUES ('Banco Demo')`); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO debts (institution_id, identifier, description, total, principal,
		                   currency_id, installments_total, installment_amount,
		                   interest_rate, payment_day, start_date, status)
		VALUES (1, 'demo-account-001', 'Lote Demo 1', 100, 100, 1, 1, 100, 0, 5, '2026-09-01', 'activa')
	`); err != nil {
		t.Fatalf("insertar deuda: %v", err)
	}

	infoStorage := storage.NewDebtInformationStorage(database)
	debtStorage := storage.NewDebtStorage(database)
	service := services.NewDebtInformationService(infoStorage, debtStorage)
	return NewDebtInformationHandlers(service), database
}

func debtInformationRequest(t *testing.T, handler http.HandlerFunc, method, path, id string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("serializar body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, path, reader)
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func decodeDebtInformationError(t *testing.T, rec *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	var response errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decodificar error: %v (body=%s)", err, rec.Body.String())
	}
	return response
}

func TestDebtInformationTemplateHandlers(t *testing.T) {
	handler, _ := setupDebtInformationHandlers(t)

	payload := map[string]any{
		"name":        "Transferencia Demo",
		"description": "Datos de prueba",
		"fields": []map[string]any{
			{"key": "monto", "label": "Monto", "type": "computed", "value_template": "{{cuota.monto_formateado}}"},
			{"key": "descripcion", "label": "Descripción", "type": "computed", "value_template": "{{deuda.descripcion}} - Cuota {{cuota.numero}}"},
			{"key": "banco", "label": "Banco", "type": "text", "required": true},
			{"key": "cuenta", "label": "# Cuenta", "type": "text", "required": true},
			{"key": "nombre", "label": "Nombre", "type": "text", "required": true},
		},
	}

	rec := debtInformationRequest(t, handler.CreateTemplate, http.MethodPost, "/api/debt-information-templates", "", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("CreateTemplate: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created models.DebtInformationTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decodificar plantilla: %v", err)
	}
	if created.ID == 0 || created.FieldCount != 5 || created.UsageCount != 0 {
		t.Fatalf("plantilla creada inesperada: %+v", created)
	}

	rec = debtInformationRequest(t, handler.ListTemplates, http.MethodGet, "/api/debt-information-templates", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ListTemplates: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var list []models.DebtInformationTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decodificar lista: %v", err)
	}
	if len(list) != 1 || list[0].FieldCount != 5 || len(list[0].Fields) != 0 {
		t.Fatalf("lista inesperada: %+v", list)
	}

	rec = debtInformationRequest(t, handler.GetTemplate, http.MethodGet, "/api/debt-information-templates/1", "1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GetTemplate: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var detail models.DebtInformationTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decodificar detalle: %v", err)
	}
	if len(detail.Fields) != 5 || detail.Fields[0].Key != "monto" {
		t.Fatalf("detalle inesperado: %+v", detail)
	}

	rec = debtInformationRequest(t, handler.ListTokens, http.MethodGet, "/api/debt-information-templates/tokens", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ListTokens: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var tokens []models.DebtInformationToken
	if err := json.Unmarshal(rec.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("decodificar tokens: %v", err)
	}
	if len(tokens) != 15 || tokens[0].Key == "" {
		t.Fatalf("tokens inesperados: %+v", tokens)
	}

	duplicate := map[string]any{
		"name": "transferencia demo",
		"fields": []map[string]any{
			{"key": "banco", "label": "Banco", "type": "text"},
		},
	}
	rec = debtInformationRequest(t, handler.CreateTemplate, http.MethodPost, "/api/debt-information-templates", "", duplicate)
	if rec.Code != http.StatusConflict || decodeDebtInformationError(t, rec).Error != "template_name_exists" {
		t.Fatalf("duplicado: status=%d body=%s", rec.Code, rec.Body.String())
	}

	invalidToken := map[string]any{
		"name": "Plantilla inválida",
		"fields": []map[string]any{
			{"key": "monto", "label": "Monto", "type": "computed", "value_template": "{{cuota.total}}"},
		},
	}
	rec = debtInformationRequest(t, handler.CreateTemplate, http.MethodPost, "/api/debt-information-templates", "", invalidToken)
	if rec.Code != http.StatusBadRequest || decodeDebtInformationError(t, rec).Error != "invalid_token" {
		t.Fatalf("token inválido: status=%d body=%s", rec.Code, rec.Body.String())
	}

	update := map[string]any{
		"name":        "Transferencia Demo actualizada",
		"description": "Datos de prueba",
		"fields": []map[string]any{
			{"key": "banco", "label": "Banco", "type": "text", "required": true},
			{"key": "cuenta", "label": "# Cuenta", "type": "text", "required": true},
		},
	}
	rec = debtInformationRequest(t, handler.UpdateTemplate, http.MethodPut, "/api/debt-information-templates/1", "1", update)
	if rec.Code != http.StatusOK {
		t.Fatalf("UpdateTemplate: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated models.DebtInformationTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decodificar actualización: %v", err)
	}
	if updated.Name != "Transferencia Demo actualizada" || updated.FieldCount != 2 {
		t.Fatalf("actualización inesperada: %+v", updated)
	}

	rec = debtInformationRequest(t, handler.GetTemplate, http.MethodGet, "/api/debt-information-templates/999", "999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GetTemplate inexistente: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDebtPaymentInformationHandlers(t *testing.T) {
	handler, _ := setupDebtInformationHandlers(t)

	templatePayload := map[string]any{
		"name": "Transferencia Demo",
		"fields": []map[string]any{
			{"key": "banco", "label": "Banco", "type": "text", "required": true},
			{"key": "cuenta", "label": "# Cuenta", "type": "text", "required": true},
			{"key": "nombre", "label": "Nombre", "type": "text", "required": true},
		},
	}
	rec := debtInformationRequest(t, handler.CreateTemplate, http.MethodPost, "/api/debt-information-templates", "", templatePayload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("CreateTemplate: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var template models.DebtInformationTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &template); err != nil {
		t.Fatalf("decodificar plantilla: %v", err)
	}

	rec = debtInformationRequest(t, handler.GetDebtPaymentInformation, http.MethodGet, "/api/debts/1/payment-information", "1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Get vacío: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var empty models.DebtPaymentInformation
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatalf("decodificar vacío: %v", err)
	}
	if empty.TemplateID != nil || empty.Template != nil || len(empty.Values) != 0 {
		t.Fatalf("se esperaba forma de pago vacía: %+v", empty)
	}

	savePayload := map[string]any{
		"template_id": template.ID,
		"values": map[string]any{
			"banco":  "Banco Demo",
			"cuenta": "demo-account-001",
			"nombre": "Titular Demo",
		},
	}
	rec = debtInformationRequest(t, handler.PutDebtPaymentInformation, http.MethodPut, "/api/debts/1/payment-information", "1", savePayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("Put: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var saved models.DebtPaymentInformation
	if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decodificar guardado: %v", err)
	}
	if saved.TemplateID == nil || *saved.TemplateID != template.ID || saved.Template == nil {
		t.Fatalf("guardado inesperado: %+v", saved)
	}
	if saved.Values["cuenta"] != "demo-account-001" {
		t.Fatalf("valores inesperados: %+v", saved.Values)
	}

	missingRequired := map[string]any{
		"template_id": template.ID,
		"values":      map[string]any{"banco": "Banco Demo"},
	}
	rec = debtInformationRequest(t, handler.PutDebtPaymentInformation, http.MethodPut, "/api/debts/1/payment-information", "1", missingRequired)
	if rec.Code != http.StatusBadRequest || decodeDebtInformationError(t, rec).Error != "validation_error" {
		t.Fatalf("validación: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = debtInformationRequest(t, handler.DeleteTemplate, http.MethodDelete, "/api/debt-information-templates/1", "1", nil)
	if rec.Code != http.StatusConflict || decodeDebtInformationError(t, rec).Error != "template_in_use" {
		t.Fatalf("borrar en uso: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = debtInformationRequest(t, handler.GetDebtPaymentInformation, http.MethodGet, "/api/debts/999/payment-information", "999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("deuda inexistente: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = debtInformationRequest(t, handler.DeleteDebtPaymentInformation, http.MethodDelete, "/api/debts/1/payment-information", "1", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Delete: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = debtInformationRequest(t, handler.GetDebtPaymentInformation, http.MethodGet, "/api/debts/1/payment-information", "1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Get después de borrar: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatalf("decodificar vacío: %v", err)
	}
	if empty.TemplateID != nil || empty.Template != nil {
		t.Fatalf("se esperaba forma de pago vacía: %+v", empty)
	}

	rec = debtInformationRequest(t, handler.DeleteTemplate, http.MethodDelete, "/api/debt-information-templates/1", "1", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Delete plantilla: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDebtInformationBadRequests(t *testing.T) {
	handler, _ := setupDebtInformationHandlers(t)

	rec := debtInformationRequest(t, handler.GetTemplate, http.MethodGet, "/api/debt-information-templates/invalid", "invalid", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ID inválido: status=%d body=%s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/debt-information-templates", bytes.NewBufferString("{"))
	rec = httptest.NewRecorder()
	handler.CreateTemplate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("JSON inválido: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
