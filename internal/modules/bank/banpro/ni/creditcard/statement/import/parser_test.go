package statementimport

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// fixture reproduce el layout del estado de cuenta BANPRO con datos sintéticos
// (SEC-SYNTHETIC): sin números de tarjeta, titulares ni montos reales.
const fixture = `BANCO DE LA PRODUCCION ESTADO DE CUENTA DATOS DEL CLIENTE TITULAR DEMO RESUMEN
DESCRIPCION CORDOBAS DOLARES Saldo Anterior: 10,000.00 0.00 Compras y Retiros: 0.00 0.00
Cargos y Debitos 930.22 0.00 - Pagos y Creditos: -1,000.00 0.00 = Pago de Contado: 9,930.22 0.00
Pago Minimo 550.00 0.00 Credito Disponible 0.00
NUMERO DE TARJETA: XXXX-XXXX-XXXX-0000
FECHA DE CORTE FECHA DE PAGO LIMITE DE CREDITO 17/11/2026 14/12/2026 U$ 1,000.00
REFERENCIA FECHA DESCRIPCION CORDOBAS DOLARES
XXXX-XXXX-XXXX-0000 TITULAR DEMO
00000000000000000000001 11/01 MEMBRESIA ANUAL DEMO 550.00
00000000000000000000002 11/03 CREDITO TRANSFERIDO DE CUENTA BANCARIA -1,000.00
00000000000000000000003 11/16 CARGO ADMINISTRATIVO DE MORA 371.40
11/17 CARGO DE INTERESES POR MORA 8.82
INTERES CORRIENTES 0.00 0.00 INTERES BONIFICABLE 628.60 0.00
SALDOS AL CORTE 10,558.82 0.00 Monto en Mora 1,000.00 0.00
Pagos Atrasados 1 0 Monto en Exceso 500.00 0.00
Saldo en puntos anterior: +1,000.00 Puntos Acreditados: +10.00
Puntos Debitados: -5.00 Puntos Disponibles: +1,005.00`

func TestParseStatement(t *testing.T) {
	parsed, err := parseText(fixture)
	if err != nil {
		t.Fatalf("parseText: %v", err)
	}
	if parsed.Last4 != "0000" {
		t.Fatalf("last4 = %q, esperaba 0000", parsed.Last4)
	}
	if parsed.PeriodEnd != "2026-11-17" || parsed.MinPaymentDue != "2026-12-14" {
		t.Fatalf("fechas inesperadas: %s / %s", parsed.PeriodEnd, parsed.MinPaymentDue)
	}
	if parsed.PreviousBalance != 10000 || parsed.TotalPayment != 9930.22 {
		t.Fatalf("saldos inesperados: %v / %v", parsed.PreviousBalance, parsed.TotalPayment)
	}
	if parsed.InterestBonifiable != 628.60 || parsed.NewBalance != 10558.82 {
		t.Fatalf("bonificable/saldo al corte: %v / %v", parsed.InterestBonifiable, parsed.NewBalance)
	}
	if parsed.MoraAmount != 1000 || parsed.ArrearsCount != 1 || parsed.OverLimitAmount != 500 {
		t.Fatalf("mora/atrasos/exceso: %v / %d / %v", parsed.MoraAmount, parsed.ArrearsCount, parsed.OverLimitAmount)
	}
	if parsed.MinPayment != 550 {
		t.Fatalf("pago mínimo = %v", parsed.MinPayment)
	}
	if parsed.PointsPrevious != 1000 || parsed.PointsEarned != 10 || parsed.PointsUsed != -5 || parsed.PointsAvailable != 1005 {
		t.Fatalf("puntos inesperados: %+v", parsed)
	}
	if len(parsed.Transactions) != 4 {
		t.Fatalf("movimientos = %d, esperaba 4: %+v", len(parsed.Transactions), parsed.Transactions)
	}

	want := []struct {
		date, concept, kind string
		local               float64
	}{
		{"2026-11-01", "MEMBRESIA ANUAL DEMO", "fee", 550},
		{"2026-11-03", "CREDITO TRANSFERIDO DE CUENTA BANCARIA", "payment", -1000},
		{"2026-11-16", "CARGO ADMINISTRATIVO DE MORA", "fee", 371.40},
		{"2026-11-17", "CARGO DE INTERESES POR MORA", "interest_late", 8.82},
	}
	for i, item := range want {
		got := parsed.Transactions[i]
		if got.PostedAt != item.date || got.Concept != item.concept || got.Type != item.kind || got.AmountLocal != item.local {
			t.Errorf("fila %d = %+v, esperaba %+v", i+1, got, item)
		}
	}
	if parsed.Transactions[0].ExternalRef == "" {
		t.Error("la fila con referencia bancaria debería usarla como external_ref")
	}
	if parsed.Transactions[3].ExternalRef != "" {
		t.Error("la fila sin referencia no debería tener external_ref")
	}
	if len(parsed.Warnings) != 1 || parsed.Warnings[0].Code != "single_payment_date" {
		t.Fatalf("warnings inesperados: %+v", parsed.Warnings)
	}
}

func TestParseStatementRejectsForeignLayout(t *testing.T) {
	_, err := parseText("ESTADO DE CUENTA OTRO BANCO Saldo Anterior: 1.00 0.00")
	if !errors.Is(err, modules.ErrUnsupportedStatement) {
		t.Fatalf("err = %v, esperaba ErrUnsupportedStatement", err)
	}
}

func TestParseStatementNeedsMovements(t *testing.T) {
	text := strings.Replace(fixture, "00000000000000000000001 11/01 MEMBRESIA ANUAL DEMO 550.00", "", 1)
	text = strings.Replace(text, "00000000000000000000002 11/03 CREDITO TRANSFERIDO DE CUENTA BANCARIA -1,000.00", "", 1)
	text = strings.Replace(text, "00000000000000000000003 11/16 CARGO ADMINISTRATIVO DE MORA 371.40", "", 1)
	text = strings.Replace(text, "11/17 CARGO DE INTERESES POR MORA 8.82", "", 1)
	if _, err := parseText(text); !errors.Is(err, modules.ErrUnsupportedStatement) {
		t.Fatalf("err = %v, esperaba ErrUnsupportedStatement", err)
	}
}

func TestInferDate(t *testing.T) {
	cases := []struct {
		month, day int
		want       string
	}{
		{11, 1, "2026-11-01"},
		{11, 30, "2026-11-30"},
		{12, 28, "2025-12-28"}, // mes posterior al corte → año anterior
		{1, 15, "2026-01-15"},
		{13, 1, ""},
		{2, 30, ""},
	}
	for _, tc := range cases {
		if got := inferDate("2026-11-17", tc.month, tc.day); got != tc.want {
			t.Errorf("inferDate(11/17, %d/%d) = %q, esperaba %q", tc.month, tc.day, got, tc.want)
		}
	}
}

func TestModuleInfo(t *testing.T) {
	info := New().Info()
	if info.ID != "bank.banpro.ni.creditcard.statement.import" || info.Provider != "banpro" || info.Country != "ni" {
		t.Fatalf("info inesperada: %+v", info)
	}
	if !New().CanParse("estado.PDF", "application/octet-stream") {
		t.Fatal("debería aceptar PDFs")
	}
	if New().CanParse("estado.csv", "text/csv") {
		t.Fatal("no debería aceptar CSV")
	}
}

func TestParseEmptyPDF(t *testing.T) {
	if _, err := New().Parse(context.Background(), strings.NewReader("")); !errors.Is(err, modules.ErrUnsupportedStatement) {
		t.Fatalf("err = %v, esperaba ErrUnsupportedStatement", err)
	}
}
