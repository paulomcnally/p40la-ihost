// Package modules define el contrato y el registro de módulos con namespace
// canónico (ARCH-MODULES, SPEC-137).
package modules

import (
	"context"
	"io"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// Info identifica un módulo de forma estable.
type Info struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Kind        string   `json:"kind"` // core | provider
	Provider    string   `json:"provider,omitempty"`
	Country     string   `json:"country,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Migrations  []string `json:"migrations,omitempty"`
}

// Module es cualquier módulo registrable.
type Module interface {
	Info() Info
}

// Router monta las rutas HTTP del módulo. El router central solo invoca este
// método, de modo que el módulo es dueño de sus paths.
type Router interface {
	Module
	RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler)
}

// StatementImporter parsea un estado de cuenta sin persistirlo.
type StatementImporter interface {
	Module
	CanParse(filename, mimeType string) bool
	Parse(ctx context.Context, r io.Reader) (*models.ParsedStatement, error)
}
