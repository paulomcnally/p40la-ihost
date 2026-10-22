package services

import (
	pluginall "github.com/paulomcnally/p40la-ihost-automation-plugins/all"
	"github.com/paulomcnally/p40la-ihost-automation-plugins/plugins"
)

// PluginRegistry es el contrato mínimo del registry de plugins. Permite
// inyectar un registry falso en tests.
type PluginRegistry interface {
	Get(name string) (plugins.Plugin, bool)
	List() []plugins.Plugin
}

// NewAutomationPluginRegistry construye el registry con todos los plugins
// compilados del repo privado p40la-ihost-automation-plugins.
func NewAutomationPluginRegistry() PluginRegistry {
	registry := plugins.NewRegistry()
	pluginall.RegisterAll(registry)
	return registry
}
