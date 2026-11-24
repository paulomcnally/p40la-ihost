// Package all es el único punto de entrada que registra los módulos
// namespaced del proyecto (ARCH-MODULES).
package all

import (
	"database/sql"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
	bacimport "github.com/paulomcnally/p40la-ihost/internal/modules/bank/bac/ni/creditcard/statement/import"
	banproimport "github.com/paulomcnally/p40la-ihost/internal/modules/bank/banpro/ni/creditcard/statement/import"
	"github.com/paulomcnally/p40la-ihost/internal/modules/bank/creditcard"
	homemaintenance "github.com/paulomcnally/p40la-ihost/internal/modules/home/maintenance"
	"github.com/paulomcnally/p40la-ihost/internal/modules/realestate/portfolio"
	"github.com/paulomcnally/p40la-ihost/internal/modules/realestate/rental"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// Register registra todos los módulos disponibles en el registry.
func Register(registry *modules.Registry, db *sql.DB, email *services.EmailService, settings *services.SystemSettingsService, dataDir string) {
	registry.Register(creditcard.New(db, registry))
	registry.Register(bacimport.New())
	registry.Register(banproimport.New())
	rentalModule := rental.New(db, email, settings, dataDir)
	registry.Register(rentalModule)
	registry.Register(portfolio.New(db, rentalModule))
	registry.Register(homemaintenance.New(db))
}
