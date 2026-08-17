package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// voiceMonkeyURL es el endpoint de anuncios TTS de Voice Monkey (API v3).
const voiceMonkeyURL = "https://api-v3.voicemonkey.io/announce"

// announceRequest mapea el body del endpoint /announce.
type announceRequest struct {
	Token  string `json:"token"`
	Device string `json:"device"`
	Speech string `json:"speech"`
	Voice  string `json:"voice"`
	Chime  string `json:"chime,omitempty"`
}

// announceResponse mapea la respuesta del endpoint /announce.
type announceResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data"`
	Error   string `json:"error"`
}

// VoiceMonkeyService envía anuncios TTS a altavoces Alexa via Voice Monkey.
// Usa solo stdlib (net/http + encoding/json). El token/device NUNCA se loguea.
type VoiceMonkeyService struct {
	settings *SystemSettingsService
	client   *http.Client
	url      string
}

func NewVoiceMonkeyService(settings *SystemSettingsService) *VoiceMonkeyService {
	return &VoiceMonkeyService{
		settings: settings,
		client:   &http.Client{Timeout: 10 * time.Second},
		url:      voiceMonkeyURL,
	}
}

// IsConfigured indica si token y device están guardados.
func (s *VoiceMonkeyService) IsConfigured(ctx context.Context) (bool, error) {
	return s.settings.IsVoiceMonkeyConfigured(ctx)
}

// IsEnabled indica si el toggle maestro está activo.
func (s *VoiceMonkeyService) IsEnabled(ctx context.Context) (bool, error) {
	return s.settings.IsVoiceMonkeyEnabled(ctx)
}

// IsSendingAlerts indica si el toggle "enviar alertas" está activo.
func (s *VoiceMonkeyService) IsSendingAlerts(ctx context.Context) (bool, error) {
	return s.settings.IsVoiceMonkeySendingAlerts(ctx)
}

// Announce envía un anuncio TTS con la voz Lucia (español).
// El error devuelto NUNCA incluye token/device.
func (s *VoiceMonkeyService) Announce(ctx context.Context, speech string) error {
	cfg, err := s.settings.GetVoiceMonkeyConfig(ctx)
	if err != nil {
		return fmt.Errorf("obtener config Voice Monkey: %w", err)
	}
	if !s.settings.isVoiceMonkeyConfigured(cfg) {
		return fmt.Errorf("config Voice Monkey incompleta (falta token o device)")
	}

	payload := announceRequest{
		Token:  cfg.Token,
		Device: cfg.Device,
		Speech: speech,
		Voice:  "Lucia",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("armar request Voice Monkey: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("crear request Voice Monkey: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	slog.Info("voicemonkey: enviando anuncio", "device", cfg.Device)

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Error("voicemonkey: fallo HTTP", "error", err.Error())
		return fmt.Errorf("error llamando a Voice Monkey: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("error leyendo respuesta Voice Monkey: %w", err)
	}

	var result announceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("error parseando respuesta Voice Monkey: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !result.Success {
		slog.Error("voicemonkey: fallo API", "status", resp.StatusCode)
		return fmt.Errorf("voice monkey devolvió error (status %d)", resp.StatusCode)
	}

	slog.Info("voicemonkey: anuncio enviado")
	return nil
}

// SendTest anuncia un mensaje de prueba. No depende de los toggles de alerta.
func (s *VoiceMonkeyService) SendTest(ctx context.Context) error {
	return s.Announce(ctx, "Paulo, este es un aviso de prueba de P40LA.")
}
