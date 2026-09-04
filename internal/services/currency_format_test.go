package services

import "testing"

func TestCurrencyFormatFormat(t *testing.T) {
	cases := []struct {
		name   string
		format CurrencyFormat
		amount float64
		want   string
	}{
		{"default Nicaragua miles", DefaultCurrencyFormat(), 1500, "1,500.00"},
		{"default Nicaragua sin miles", DefaultCurrencyFormat(), 300, "300.00"},
		{"default Nicaragua millones", DefaultCurrencyFormat(), 1234567.5, "1,234,567.50"},
		{"default Nicaragua cero", DefaultCurrencyFormat(), 0, "0.00"},
		{"sin separador de miles", CurrencyFormat{ThousandsSeparator: "", DecimalSeparator: ".", DecimalDigits: 2}, 1500, "1500.00"},
		{"formato europeo", CurrencyFormat{ThousandsSeparator: ".", DecimalSeparator: ",", DecimalDigits: 2}, 1500, "1.500,00"},
		{"espacio como miles", CurrencyFormat{ThousandsSeparator: " ", DecimalSeparator: ".", DecimalDigits: 2}, 1500, "1 500.00"},
		{"apóstrofo como miles", CurrencyFormat{ThousandsSeparator: "'", DecimalSeparator: ".", DecimalDigits: 2}, 1500, "1'500.00"},
		{"cero decimales con redondeo", CurrencyFormat{ThousandsSeparator: ",", DecimalSeparator: ".", DecimalDigits: 0}, 1234567.5, "1,234,568"},
		{"un decimal", CurrencyFormat{ThousandsSeparator: ",", DecimalSeparator: ".", DecimalDigits: 1}, 1234.56, "1,234.6"},
		{"cuatro decimales", CurrencyFormat{ThousandsSeparator: ",", DecimalSeparator: ".", DecimalDigits: 4}, 1.234567, "1.2346"},
		{"monto negativo", DefaultCurrencyFormat(), -1500, "-1,500.00"},
	}
	for _, c := range cases {
		if got := c.format.Format(c.amount); got != c.want {
			t.Errorf("%s: Format(%v) = %q, want %q", c.name, c.amount, got, c.want)
		}
	}
}

func TestFormatAmountAndPensionAmount(t *testing.T) {
	def := DefaultCurrencyFormat()
	if got := formatAmount(1500, "C$", def); got != "C$1,500.00" {
		t.Errorf("formatAmount = %q", got)
	}
	if got := formatAmount(1500, "", def); got != "1,500.00" {
		t.Errorf("formatAmount sin símbolo = %q", got)
	}
	if got := pensionAmount(1500, "NIO", def); got != "NIO 1,500.00" {
		t.Errorf("pensionAmount = %q", got)
	}
}

func TestValidSeparator(t *testing.T) {
	for _, s := range []string{",", ".", " ", "'", ""} {
		if !validSeparator(s) {
			t.Errorf("validSeparator(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"-", "|", "a", "1", "," + "."} {
		if validSeparator(s) {
			t.Errorf("validSeparator(%q) = true, want false", s)
		}
	}
}
