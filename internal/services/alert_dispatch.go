package services

import (
	"context"
	"log/slog"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// Helpers compartidos de dispatch multicanal (SPEC-032/SPEC-033).
// Cada scheduler decide, para su alerta, si envía por mail, por voz o ambos.

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

// dispatchVoice anuncia por voz si la alerta tiene voz habilitada y Voice Monkey
// está activo (master + enviar alertas + configurado). No bloqueante: un error
// se loguea sin interrumpir el resto del flujo del scheduler.
func dispatchVoice(ctx context.Context, alerts *AlertService, vm *VoiceMonkeyService, key, speech string) {
	if !alertVoiceEnabled(ctx, alerts, key) {
		slog.Debug("alert dispatch: alerta sin voz habilitada", "key", key)
		return
	}

	vmEnabled, err := vm.IsEnabled(ctx)
	if err != nil {
		slog.Error("alert dispatch: error al leer toggle master Voice Monkey", "error", err)
		return
	}
	if !vmEnabled {
		slog.Debug("alert dispatch: Voice Monkey deshabilitado", "key", key)
		return
	}

	sending, err := vm.IsSendingAlerts(ctx)
	if err != nil {
		slog.Error("alert dispatch: error al leer toggle enviar alertas", "error", err)
		return
	}
	if !sending {
		slog.Debug("alert dispatch: envío de alertas por voz deshabilitado", "key", key)
		return
	}

	if err := vm.Announce(ctx, speech); err != nil {
		slog.Error("alert dispatch: error al anunciar por voz", "key", key, "error", err.Error())
	}
}
