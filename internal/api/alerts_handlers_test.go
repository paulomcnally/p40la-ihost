package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newAlertsTestHandler(t *testing.T) (*AlertsHandlers, *services.SystemSettingsService) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	alertStorage := storage.NewAlertStorage(database)
	alertService := services.NewAlertService(alertStorage)
	if err := alertService.Seed(context.Background()); err != nil {
		t.Fatalf("seed alertas: %v", err)
	}

	settingsService := services.NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
	return NewAlertsHandlers(alertService, settingsService), settingsService
}

func doUpdateAlert(h *AlertsHandlers, key string, body map[string]bool) *httptest.ResponseRecorder {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/alerts/"+key, bytes.NewReader(payload))
	req.SetPathValue("key", key)
	rr := httptest.NewRecorder()
	h.UpdateAlert(rr, req)
	return rr
}

func TestUpdateAlert_GatingMailDisabledWithoutPrereqs(t *testing.T) {
	h, _ := newAlertsTestHandler(t)

	rr := doUpdateAlert(h, models.AlertKeyInsurance, map[string]bool{"mail_enabled": true})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperaba 422 sin prerequisitos, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateAlert_GatingMailEnabledWithPrereqs(t *testing.T) {
	h, settings := newAlertsTestHandler(t)
	ctx := context.Background()

	if err := settings.SetEmailAlertsEnabled(ctx, true); err != nil {
		t.Fatalf("set email master: %v", err)
	}
	if err := settings.SetSMTPConfig(ctx, &models.SMTPConfig{Host: "smtp.test", Port: 587, User: "u", Password: "p"}); err != nil {
		t.Fatalf("set smtp: %v", err)
	}
	if err := settings.SetAlertEmails(ctx, []string{"a@b.com"}); err != nil {
		t.Fatalf("set recipients: %v", err)
	}

	rr := doUpdateAlert(h, models.AlertKeyInsurance, map[string]bool{"mail_enabled": true})
	if rr.Code != http.StatusOK {
		t.Fatalf("esperaba 200 con prerequisitos, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateAlert_GatingMailNeedsRecipients(t *testing.T) {
	h, settings := newAlertsTestHandler(t)
	ctx := context.Background()

	if err := settings.SetEmailAlertsEnabled(ctx, true); err != nil {
		t.Fatalf("set email master: %v", err)
	}
	if err := settings.SetSMTPConfig(ctx, &models.SMTPConfig{Host: "smtp.test", Port: 587, User: "u", Password: "p"}); err != nil {
		t.Fatalf("set smtp: %v", err)
	}

	rr := doUpdateAlert(h, models.AlertKeyInsurance, map[string]bool{"mail_enabled": true})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperaba 422 sin destinatarios, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateAlert_GatingVoiceDisabledWithoutVM(t *testing.T) {
	h, _ := newAlertsTestHandler(t)

	rr := doUpdateAlert(h, models.AlertKeyInsurance, map[string]bool{"voice_enabled": true})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperaba 422 sin Voice Monkey activo, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateAlert_DisablingMailNotGated(t *testing.T) {
	h, _ := newAlertsTestHandler(t)

	rr := doUpdateAlert(h, models.AlertKeyInsurance, map[string]bool{"mail_enabled": false})
	if rr.Code != http.StatusOK {
		t.Fatalf("desactivar mail no debe estar gateado, got %d: %s", rr.Code, rr.Body.String())
	}
}
