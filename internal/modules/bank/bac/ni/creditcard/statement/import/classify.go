package statementimport

import "github.com/paulomcnally/p40la-ihost/internal/modules/bank/statementtext"

// Tipos de movimiento que el importador puede clasificar.
const (
	typePurchase       = statementtext.TypePurchase
	typePayment        = statementtext.TypePayment
	typePaymentBonus   = statementtext.TypePaymentBonus
	typeCashAdvance    = statementtext.TypeCashAdvance
	typeCashAdvanceFee = statementtext.TypeCashAdvanceFee
	typeInterest       = statementtext.TypeInterestCurrent
	typeFee            = statementtext.TypeFee
	typeTax            = statementtext.TypeTax
)

// classifyConcept asigna un tipo según palabras clave del concepto impreso.
// La regla vive en statementtext para compartirla con otros importadores.
func classifyConcept(concept string) string {
	return statementtext.Classify(concept)
}

// normalizeConcept quita acentos y normaliza espacios para comparar.
func normalizeConcept(value string) string {
	return statementtext.Normalize(value)
}
