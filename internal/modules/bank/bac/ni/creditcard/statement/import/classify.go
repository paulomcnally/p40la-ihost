package statementimport

import "strings"

// Tipos de movimiento que el importador puede clasificar.
const (
	typePurchase       = "purchase"
	typePayment        = "payment"
	typePaymentBonus   = "payment_bonus"
	typeCashAdvance    = "cash_advance"
	typeCashAdvanceFee = "cash_advance_fee"
	typeInterest       = "interest_current"
	typeFee            = "fee"
	typeTax            = "tax"
)

// classifyConcept asigna un tipo según palabras clave del concepto impreso.
func classifyConcept(concept string) string {
	value := normalizeConcept(concept)

	switch {
	case strings.Contains(value, "SU PAGO RECIBIDO"):
		return typePayment
	case strings.Contains(value, "BONIFICACION"):
		return typePaymentBonus
	case strings.Contains(value, "COMISION ADELANTO"):
		return typeCashAdvanceFee
	case strings.Contains(value, "RETIRO DE EFEC"):
		return typeCashAdvance
	case strings.Contains(value, "INTERES"):
		return typeInterest
	case strings.Contains(value, "IMPUESTO"), strings.Contains(value, "IVA"):
		return typeTax
	case strings.Contains(value, "COMISION"),
		strings.Contains(value, "MEMBRESIA"),
		strings.Contains(value, "SEGURO"):
		return typeFee
	default:
		return typePurchase
	}
}

// normalizeConcept quita acentos y normaliza espacios para comparar.
func normalizeConcept(value string) string {
	replacer := strings.NewReplacer(
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U",
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u",
	)
	value = replacer.Replace(value)
	return strings.ToUpper(strings.Join(strings.Fields(value), " "))
}
