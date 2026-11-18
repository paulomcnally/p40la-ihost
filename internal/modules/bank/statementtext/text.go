// Package statementtext contiene utilidades puras compartidas por los
// importadores de estados de cuenta (SPEC-139). No conoce bancos ni layouts:
// solo normaliza conceptos impresos y los clasifica con el contrato de
// bank.creditcard (ARCH-MODULES permite reutilizar helpers puros).
package statementtext

import "strings"

// DecodeLegacyEncoding corrige los PDFs cuyas fuentes no traen ToUnicode: los
// glifos llegan desplazados 31 posiciones (los dígitos como caracteres de
// control). Se detecta por la presencia de caracteres de control no blancos y,
// en ese caso, se desplaza todo carácter gráfico por debajo de 0x60.
// Es determinista y no toca documentos bien codificados.
func DecodeLegacyEncoding(text string) string {
	controls := 0
	for _, r := range text {
		if r < 0x20 && r != '\n' && r != '\r' && r != '\t' {
			controls++
			if controls > 5 {
				return shiftLegacy(text)
			}
		}
	}
	return text
}

func shiftLegacy(text string) string {
	var out strings.Builder
	out.Grow(len(text))
	for _, r := range text {
		switch {
		case r == ' ' || r == '\n' || r == '\t':
			out.WriteRune(r)
		case r > 0 && r < 0x60:
			out.WriteRune(r + 31)
		case r >= 0x7F:
			// Glifos sin mapeo (acentos): se descartan para no romper anclas.
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// Tipos de movimiento del contrato con bank.creditcard.
const (
	TypePurchase        = "purchase"
	TypePayment         = "payment"
	TypePaymentBonus    = "payment_bonus"
	TypeCashAdvance     = "cash_advance"
	TypeCashAdvanceFee  = "cash_advance_fee"
	TypeInterestCurrent = "interest_current"
	TypeInterestLate    = "interest_late"
	TypeFee             = "fee"
	TypeTax             = "tax"
	TypeOther           = "other"
)

// Normalize quita acentos y normaliza espacios para comparar conceptos.
func Normalize(concept string) string {
	replacer := strings.NewReplacer(
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U",
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u",
	)
	return strings.ToUpper(strings.Join(strings.Fields(replacer.Replace(concept)), " "))
}

// Classify asigna un tipo de movimiento según palabras clave del concepto
// impreso. La mora se evalúa antes que el interés genérico y los abonos antes
// que las compras (SPEC-139): un abono nunca puede quedar como compra negativa.
func Classify(concept string) string {
	value := Normalize(concept)

	switch {
	case strings.Contains(value, "BONIFICACION"):
		return TypePaymentBonus
	case strings.Contains(value, "SU PAGO RECIBIDO"),
		strings.Contains(value, "CREDITO TRANSFERIDO"),
		strings.Contains(value, "NOTA DE CREDITO"),
		strings.Contains(value, "ABONO"),
		strings.Contains(value, "PAGO"):
		return TypePayment
	case strings.Contains(value, "COMISION ADELANTO"):
		return TypeCashAdvanceFee
	case strings.Contains(value, "RETIRO DE EFEC"):
		return TypeCashAdvance
	case strings.Contains(value, "MORA") && strings.Contains(value, "INTERES"):
		return TypeInterestLate
	case strings.Contains(value, "MORA"):
		return TypeFee
	case strings.Contains(value, "INTERES"):
		return TypeInterestCurrent
	case strings.Contains(value, "IMPUESTO"), strings.Contains(value, "IVA"):
		return TypeTax
	case strings.Contains(value, "COMISION"),
		strings.Contains(value, "MEMBRESIA"),
		strings.Contains(value, "SEGURO"),
		strings.Contains(value, "ADMINISTRATIVO"):
		return TypeFee
	default:
		return TypePurchase
	}
}
