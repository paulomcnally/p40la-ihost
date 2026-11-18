package statementimport

import "testing"

const syntheticStatement = `Transacciones del periodo
Producto: 4000-00**-****-0000   Nombre: TITULAR DEMO   Fecha de corte: 15/03/2026
Pago Mínimo/Fecha límite: 14/04/2026   Pago Mínimo/Monto Local: 200.00   Pago Mínimo/Monto Dólares: 20.00
Pago Contado/Fecha limite: 07/04/2026   Pago Contado/Monto Local: 1124.56   Pago Contado/Monto Dólares: 51.11
Previous balance1000.00100.00
4000-00**-****-0000 0.00 0.00
TITULAR DEMO 0.00 0.00
02/03/2026 Comercio Demo NI 123.45 0.00
05/03/2026 SU PAGO RECIBIDO GRACIAS 0.00 -50.00
BONIFICACION PAGO SALDO LOCAL -10.00 0.00
PRICECASH GANADOS C$ 0.00 0.00
DISPONIBLE 1,468 0.00 0.00
INTERESES CORRIENTES DEL MES/LOCAL: 11.11 INTERESES CORRIENTES DEL MES/DÓLARES: 1.11
Saldo Al Corte/Local: 1124.56 Saldo Al Corte/Dólares: 51.11
`

func TestParseTextSyntheticStatement(t *testing.T) {
	parsed, err := parseText(syntheticStatement)
	if err != nil {
		t.Fatalf("parseText: %v", err)
	}
	if parsed.Last4 != "0000" {
		t.Fatalf("last4 = %q, quiero 0000", parsed.Last4)
	}
	if parsed.PeriodEnd != "2026-03-15" || parsed.MinPaymentDue != "2026-04-14" || parsed.FullPaymentDue != "2026-04-07" {
		t.Fatalf("fechas inesperadas: %s / %s / %s", parsed.PeriodEnd, parsed.MinPaymentDue, parsed.FullPaymentDue)
	}
	if parsed.PreviousBalance != 1000 || parsed.PreviousBalanceUSD != 100 {
		t.Fatalf("saldo anterior inesperado: %v / %v", parsed.PreviousBalance, parsed.PreviousBalanceUSD)
	}
	if parsed.MinPayment != 200 || parsed.MinPaymentUSD != 20 {
		t.Fatalf("pago mínimo inesperado: %v / %v", parsed.MinPayment, parsed.MinPaymentUSD)
	}
	if parsed.NewBalance != 1124.56 || parsed.NewBalanceUSD != 51.11 {
		t.Fatalf("saldo al corte inesperado: %v / %v", parsed.NewBalance, parsed.NewBalanceUSD)
	}
	if parsed.InterestCurrent != 11.11 || parsed.InterestCurrentUSD != 1.11 {
		t.Fatalf("intereses inesperados: %v / %v", parsed.InterestCurrent, parsed.InterestCurrentUSD)
	}

	if len(parsed.Transactions) != 4 {
		t.Fatalf("movimientos = %d, quiero 4: %+v", len(parsed.Transactions), parsed.Transactions)
	}
	if parsed.Transactions[0].Type != typePurchase || parsed.Transactions[0].Concept != "Comercio Demo" {
		t.Fatalf("primera transacción inesperada: %+v", parsed.Transactions[0])
	}
	if parsed.Transactions[0].AmountLocal != 123.45 || parsed.Transactions[0].CurrencyOriginal != "NIO" {
		t.Fatalf("montos de compra inesperados: %+v", parsed.Transactions[0])
	}
	var payment, bonus, interest bool
	for _, tx := range parsed.Transactions {
		switch tx.Type {
		case typePayment:
			payment = true
			if tx.AmountUSD != -50 {
				t.Fatalf("pago inesperado: %+v", tx)
			}
		case typePaymentBonus:
			bonus = true
			if tx.PostedAt != "2026-03-15" || tx.AmountLocal != -10 {
				t.Fatalf("bonificación inesperada: %+v", tx)
			}
		case typeInterest:
			interest = true
		}
		if tx.Concept == "PRICECASH GANADOS C$" || tx.Concept == "DISPONIBLE 1,468" {
			t.Fatalf("línea informativa importada como movimiento: %+v", tx)
		}
	}
	if !payment || !bonus || !interest {
		t.Fatalf("faltan tipos: payment=%v bonus=%v interest=%v", payment, bonus, interest)
	}
	if len(parsed.Warnings) != 0 {
		t.Fatalf("warnings inesperados: %+v", parsed.Warnings)
	}
}

func TestParseTextReconciliationMismatch(t *testing.T) {
	text := replaceOnce(syntheticStatement, "Saldo Al Corte/Local: 1124.56", "Saldo Al Corte/Local: 1125.06")
	parsed, err := parseText(text)
	if err != nil {
		t.Fatalf("parseText: %v", err)
	}
	if len(parsed.Warnings) != 1 || parsed.Warnings[0].Code != "reconciliation_mismatch" {
		t.Fatalf("warnings = %+v, quiero reconciliation_mismatch", parsed.Warnings)
	}
}

func TestParseTextMissingDates(t *testing.T) {
	text := replaceOnce(syntheticStatement, "Fecha de corte: 15/03/2026", "")
	if _, err := parseText(text); err == nil {
		t.Fatal("esperaba error por falta de fecha de corte")
	}
}

func TestParseTextNoTransactions(t *testing.T) {
	text := `Fecha de corte: 15/03/2026
Pago Mínimo/Fecha límite: 14/04/2026
Pago Contado/Fecha limite: 07/04/2026
Previous balance1000.00100.00
Saldo Al Corte/Local: 1000.00 Saldo Al Corte/Dólares: 100.00
`
	if _, err := parseText(text); err == nil {
		t.Fatal("esperaba error sin movimientos")
	}
}

func replaceOnce(value, old, replacement string) string {
	index := indexOf(value, old)
	if index < 0 {
		return value
	}
	return value[:index] + replacement + value[index+len(old):]
}

func indexOf(value, old string) int {
	for i := 0; i+len(old) <= len(value); i++ {
		if value[i:i+len(old)] == old {
			return i
		}
	}
	return -1
}

func TestClassifyConcept(t *testing.T) {
	cases := map[string]string{
		"SU PAGO RECIBIDO GRACIAS":      typePayment,
		"BONIFICACION PAGO SALDO LOCAL": typePaymentBonus,
		"COMISION ADELANTO DE EFECTIVO": typeCashAdvanceFee,
		"Retiro de Efec.Comercio Demo":  typeCashAdvance,
		"INTERESES CORRIENTES":          typeInterest,
		"SEGURO DE TARJETA":             typeFee,
		"IMPUESTO DE VENTA":             typeTax,
		"Comercio Demo":                 typePurchase,
	}
	for concept, want := range cases {
		if got := classifyConcept(concept); got != want {
			t.Errorf("classifyConcept(%q) = %q, quiero %q", concept, got, want)
		}
	}
}

func TestExtractLast4(t *testing.T) {
	if got := extractLast4("4000-00**-****-0000"); got != "0000" {
		t.Fatalf("last4 = %q", got)
	}
	if got := extractLast4(""); got != "" {
		t.Fatalf("last4 vacío = %q", got)
	}
}
