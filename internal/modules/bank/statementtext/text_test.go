package statementtext

import "testing"

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"MEMBRESIA ANUAL SEGUNDO AÑO":            TypeFee,
		"CREDITO TRANSFERIDO DE CUENTA BANCARIA": TypePayment,
		"SU PAGO RECIBIDO GRACIAS":               TypePayment,
		"NOTA DE CREDITO DEMO":                   TypePayment,
		"ABONO A CUENTA":                         TypePayment,
		"BONIFICACION PAGO SALDO LOCAL":          TypePaymentBonus,
		"CARGO DE INTERESES POR MORA":            TypeInterestLate,
		"CARGO ADMINISTRATIVO DE MORA":           TypeFee,
		"INTERESES CORRIENTES":                   TypeInterestCurrent,
		"COMISION ADELANTO DE EFECTIVO":          TypeCashAdvanceFee,
		"RETIRO DE EFEC. CAJERO":                 TypeCashAdvance,
		"IMPUESTO DE VENTA":                      TypeTax,
		"SEGURO DE VIDA":                         TypeFee,
		"PEDIDOSYA RESTAURANTES":                 TypePurchase,
	}
	for concept, want := range cases {
		if got := Classify(concept); got != want {
			t.Errorf("Classify(%q) = %q, esperaba %q", concept, got, want)
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize("  Interés   anual "); got != "INTERES ANUAL" {
		t.Fatalf("Normalize = %q", got)
	}
}

func TestDecodeLegacyEncodingKeepsCleanText(t *testing.T) {
	clean := "BANCO DE LA PRODUCCION Saldo Anterior: 1,000.00"
	if got := DecodeLegacyEncoding(clean); got != clean {
		t.Fatalf("el texto limpio no debe cambiar: %q", got)
	}
}

func TestDecodeLegacyEncodingShiftsGarbledText(t *testing.T) {
	// «BANCO» desplazado 31 posiciones.
	garbled := string(rune('B'-31)) + string(rune('A'-31)) + string(rune('N'-31)) +
		string(rune('C'-31)) + string(rune('O'-31)) + "\x10\x11\x12"
	// Los dígitos 0-2 se codifican como 0x11-0x13; se agregan caracteres de
	// control adicionales para disparar la detección.
	garbled += "\x13\x13\x13\x13\x13\x13"
	got := DecodeLegacyEncoding(garbled)
	if got[:5] != "BANCO" {
		t.Fatalf("decodificado = %q", got)
	}
}
