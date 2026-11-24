package maintenance

import (
	"errors"
	"time"
)

// ModuleID es el identificador canónico del módulo (ARCH-MODULES).
const (
	ModuleID      = "home.maintenance"
	ModuleVersion = "1.0.0"
)

// Tipos de mantenimiento.
const (
	KindCorrective = "corrective"
	KindPreventive = "preventive"
)

// Kinds contiene los tipos válidos.
var Kinds = []string{KindCorrective, KindPreventive}

// Áreas de la vivienda.
const (
	AreaPlumbing   = "plumbing"
	AreaElectrical = "electrical"
	AreaRoof       = "roof"
	AreaPaint      = "paint"
	AreaCarpentry  = "carpentry"
	AreaAppliances = "appliances"
	AreaGarden     = "garden"
	AreaCleaning   = "cleaning"
	AreaOther      = "other"
)

// Areas contiene las áreas válidas.
var Areas = []string{
	AreaPlumbing, AreaElectrical, AreaRoof, AreaPaint, AreaCarpentry,
	AreaAppliances, AreaGarden, AreaCleaning, AreaOther,
}

// Tipos de línea de detalle.
const (
	ItemLabor    = "labor"
	ItemSupplies = "supplies"
)

// ItemTypes contiene los tipos de línea válidos.
var ItemTypes = []string{ItemLabor, ItemSupplies}

// Unidades de medida para insumos.
const (
	UnitUnit   = "unit"
	UnitMeter  = "m"
	UnitSquare = "m2"
	UnitKilo   = "kg"
	UnitPound  = "lb"
	UnitLiter  = "l"
	UnitGallon = "gal"
	UnitSack   = "sack"
	UnitBox    = "box"
)

// Units contiene las unidades válidas.
var Units = []string{
	UnitUnit, UnitMeter, UnitSquare, UnitKilo, UnitPound,
	UnitLiter, UnitGallon, UnitSack, UnitBox,
}

// Errores de dominio.
var (
	ErrHomeNotFound        = errors.New("vivienda no encontrada")
	ErrMaintenanceNotFound = errors.New("mantenimiento no encontrado")
	ErrItemNotFound        = errors.New("línea de mantenimiento no encontrada")
	ErrCurrencyNotFound    = errors.New("moneda no encontrada")
	ErrInstitutionNotFound = errors.New("institución no encontrada")
)

// ValidationError describe un error de validación con código estable.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(code, message string) error {
	return &ValidationError{Code: code, Message: message}
}

// Maintenance es una reparación o mantenimiento registrado.
type Maintenance struct {
	ID               int64           `json:"id"`
	HomeID           int64           `json:"home_id"`
	PerformedAt      string          `json:"performed_at"`
	Kind             string          `json:"kind"`
	Area             string          `json:"area"`
	InstitutionID    *int64          `json:"institution_id"`
	InstitutionName  string          `json:"institution_name"`
	Description      string          `json:"description"`
	TotalsByCurrency []CurrencyTotal `json:"totals_by_currency"`
	Items            []Item          `json:"items"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// Item es una línea de mano de obra o insumos.
type Item struct {
	ID              int64     `json:"id"`
	MaintenanceID   int64     `json:"maintenance_id"`
	ItemType        string    `json:"item_type"`
	Description     string    `json:"description"`
	Quantity        float64   `json:"quantity"`
	Unit            string    `json:"unit"`
	Amount          float64   `json:"amount"`
	CurrencyID      int64     `json:"currency_id"`
	CurrencyCode    string    `json:"currency_code"`
	InstitutionID   *int64    `json:"institution_id"`
	InstitutionName string    `json:"institution_name"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Input es el cuerpo para crear o actualizar un mantenimiento.
// La moneda no se define acá: cada línea lleva la suya.
type Input struct {
	PerformedAt   string `json:"performed_at"`
	Kind          string `json:"kind"`
	Area          string `json:"area"`
	InstitutionID *int64 `json:"institution_id"`
	Description   string `json:"description"`
}

// CreateInput es el cuerpo del POST: mantenimiento y, opcionalmente, su
// primera línea creada en la misma transacción. El PUT sigue usando Input.
type CreateInput struct {
	Input
	Item *ItemInput `json:"item,omitempty"`
}

// ItemInput es el cuerpo para crear o actualizar una línea.
type ItemInput struct {
	ItemType      string  `json:"item_type"`
	Description   string  `json:"description"`
	Quantity      float64 `json:"quantity"`
	Unit          string  `json:"unit"`
	Amount        float64 `json:"amount"`
	CurrencyID    int64   `json:"currency_id"`
	InstitutionID *int64  `json:"institution_id"`
}

// CurrencyTotal es el total de una moneda, con desglose por tipo de gasto.
type CurrencyTotal struct {
	CurrencyID   int64   `json:"currency_id"`
	CurrencyCode string  `json:"currency_code"`
	LaborCost    float64 `json:"labor_cost"`
	SuppliesCost float64 `json:"supplies_cost"`
	Total        float64 `json:"total"`
}

// List es la respuesta del histórico de una vivienda.
type List struct {
	Items            []Maintenance   `json:"items"`
	TotalsByCurrency []CurrencyTotal `json:"totals_by_currency"`
	Truncated        bool            `json:"truncated"`
}

// Filter restringe el histórico por año o fecha exacta, tipo, tipo de línea y
// texto. Limit es interno (topes de la API) y no se expone como query param.
type Filter struct {
	Year         int
	Kind         string
	Date         string
	ItemType     string
	Query        string
	IncludeItems bool
	Limit        int
}
