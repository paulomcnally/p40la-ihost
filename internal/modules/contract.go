// Package modules define el contrato y el registro de módulos con namespace
// canónico (ARCH-MODULES, SPEC-137).
package modules

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// ErrUnsupportedStatement indica que el archivo no corresponde al layout que un
// importador reconoce (SPEC-139). El registry prueba el siguiente importador.
var ErrUnsupportedStatement = errors.New("unsupported_statement")

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

// HomeState es el estado derivado de una vivienda (SPEC-143).
type HomeState struct {
	HomeID  int64  `json:"home_id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Usage   string `json:"usage"`
	State   string `json:"state"`
}

// HomeStateProvider expone los estados derivados de todas las viviendas.
// Lo implementa realestate.rental y lo consume realestate.portfolio.
type HomeStateProvider interface {
	HomeStates(ctx context.Context) ([]HomeState, error)
}

// StatementImporter parsea un estado de cuenta sin persistirlo.
type StatementImporter interface {
	Module
	CanParse(filename, mimeType string) bool
	Parse(ctx context.Context, r io.Reader) (*models.ParsedStatement, error)
}
