package services

import (
	"context"
	"log/slog"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// Helpers compartidos de dispatch multicanal (SPEC-032/SPEC-033).
// Cada scheduler decide, para su alerta, si envía por mail, por voz o ambos.

// AlertSendResult describe el resultado del envío manual de una alerta
// (SPEC-090): qué canales se despacharon y cuántos items tenía el evento.
type AlertSendResult struct {
	Key          string   `json:"key"`
	Title        string   `json:"title"`
	SentChannels []string `json:"sent_channels"`
	Items        int      `json:"items"`
	Detail       string   `json:"detail"`
}

// newSendResult crea un AlertSendResult con slice de canales inicializado.
func newSendResult(key, title string) AlertSendResult {
	return AlertSendResult{
		Key:          key,
		Title:        title,
		SentChannels: []string{},
	}
}

// alertMailEnabled indica si la alerta tiene el canal mail habilitado.
func alertMailEnabled(ctx context.Context, alerts *AlertService, key string) bool {
	enabled, err := alerts.IsEnabled(ctx, key, models.AlertChannelMail)
	if err != nil {
		slog.Error("alert dispatch: error al leer canal mail", "key", key, "error", err)
		return false
	}
	return enabled
}

// alertVoiceEnabled indica si la alerta tiene el canal voz habilitado.
func alertVoiceEnabled(ctx context.Context, alerts *AlertService, key string) bool {
	enabled, err := alerts.IsEnabled(ctx, key, models.AlertChannelVoice)
	if err != nil {
		slog.Error("alert dispatch: error al leer canal voz", "key", key, "error", err)
		return false
	}
	return enabled
}

// alertTelegramEnabled indica si la alerta tiene el canal telegram habilitado.
func alertTelegramEnabled(ctx context.Context, alerts *AlertService, key string) bool {
	enabled, err := alerts.IsEnabled(ctx, key, models.AlertChannelTelegram)
	if err != nil {
		slog.Error("alert dispatch: error al leer canal telegram", "key", key, "error", err)
		return false
	}
	return enabled
}

// dispatchVoice anuncia por voz si la alerta tiene voz habilitada y Voice Monkey
// está activo (master + enviar alertas + configurado). No bloqueante: un error
// se loguea sin interrumpir el resto del flujo del scheduler. Devuelve true si
// efectivamente se intentó anunciar (gate maestros cumplidos).
func dispatchVoice(ctx context.Context, alerts *AlertService, vm *VoiceMonkeyService, key, speech string) bool {
	if !alertVoiceEnabled(ctx, alerts, key) {
		slog.Debug("alert dispatch: alerta sin voz habilitada", "key", key)
		return false
	}

	vmEnabled, err := vm.IsEnabled(ctx)
	if err != nil {
		slog.Error("alert dispatch: error al leer toggle master Voice Monkey", "error", err)
		return false
	}
	if !vmEnabled {
		slog.Debug("alert dispatch: Voice Monkey deshabilitado", "key", key)
		return false
	}

	sending, err := vm.IsSendingAlerts(ctx)
	if err != nil {
		slog.Error("alert dispatch: error al leer toggle enviar alertas", "error", err)
		return false
	}
	if !sending {
		slog.Debug("alert dispatch: envío de alertas por voz deshabilitado", "key", key)
		return false
	}

	if err := vm.Announce(ctx, speech); err != nil {
		slog.Error("alert dispatch: error al anunciar por voz", "key", key, "error", err.Error())
		return true
	}
	return true
}

// dispatchTelegram envía los mensajes push del bot si la alerta tiene el canal
// telegram habilitado (SPEC-088). El gate de envío real (bot habilitado,
// token y chat_ids) vive en TelegramBotService.SendAlerts: si el bot no está
// habilitado en settings, jamás se contacta la API de Telegram. No
// bloqueante: un error se loguea sin interrumpir el resto del scheduler.
// Devuelve true si se intentó enviar (gate de la alerta cumplido).
func dispatchTelegram(ctx context.Context, alerts *AlertService, bot *TelegramBotService, key string, texts []string) bool {
	if bot == nil {
		return false
	}
	if !alertTelegramEnabled(ctx, alerts, key) {
		slog.Debug("alert dispatch: alerta sin telegram habilitado", "key", key)
		return false
	}
	if len(texts) == 0 {
		return false
	}
	if err := bot.SendAlerts(ctx, texts); err != nil {
		slog.Error("alert dispatch: error al enviar alerta por Telegram", "key", key, "error", err.Error())
		return true
	}
	return true
}
