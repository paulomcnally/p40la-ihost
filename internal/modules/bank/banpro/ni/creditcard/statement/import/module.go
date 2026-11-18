package statementimport

import (
	"path/filepath"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// Identificador canónico del importador (ARCH-MODULES).
const (
	ModuleID      = "bank.banpro.ni.creditcard.statement.import"
	ModuleVersion = "1.0.0"
)

// Module implementa el importador del estado de cuenta de BANPRO (SPEC-139).
type Module struct{}

// New crea el importador.
func New() *Module { return &Module{} }

// Info devuelve los metadatos canónicos del módulo.
func (m *Module) Info() modules.Info {
	return modules.Info{
		ID:          ModuleID,
		Version:     ModuleVersion,
		Kind:        "provider",
		Provider:    "banpro",
		Country:     "ni",
		Name:        "Estado de cuenta BANPRO Nicaragua",
		Description: "Importador del PDF Estado de cuenta mensual",
	}
}

// CanParse acepta PDFs.
func (m *Module) CanParse(filename, mimeType string) bool {
	if strings.EqualFold(filepath.Ext(filename), ".pdf") {
		return true
	}
	return strings.EqualFold(mimeType, "application/pdf")
}
