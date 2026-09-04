package services

import (
	"strconv"
	"strings"
)

// CurrencyFormat define cómo se muestran los montos (SPEC-058).
// Configurable por formato (no por país): separador de miles, separador
// decimal y dígitos decimales. El default es Nicaragua: 1,000.00.
type CurrencyFormat struct {
	ThousandsSeparator string
	DecimalSeparator   string
	DecimalDigits      int
}

// DefaultCurrencyFormat devuelve el formato Nicaragua:
// miles con coma, decimales con punto y 2 dígitos decimales (1,500.00).
func DefaultCurrencyFormat() CurrencyFormat {
	return CurrencyFormat{ThousandsSeparator: ",", DecimalSeparator: ".", DecimalDigits: 2}
}

// validSeparator valida que un separador esté en la whitelist permitida:
// coma, punto, espacio, apóstrofo o ninguno.
func validSeparator(s string) bool {
	switch s {
	case ",", ".", " ", "'", "":
		return true
	}
	return false
}

// Format devuelve el monto formateado según la configuración, sin símbolo.
// Ejemplos: 1500 → "1,500.00"; 1234567.5 con 0 decimales → "1,234,568".
func (f CurrencyFormat) Format(amount float64) string {
	if f.DecimalSeparator == "" {
		f.DecimalSeparator = "."
	}
	if f.DecimalDigits < 0 {
		f.DecimalDigits = 2
	}
	numStr := strconv.FormatFloat(amount, 'f', f.DecimalDigits, 64)
	parts := strings.SplitN(numStr, ".", 2)
	intPart := groupThousands(parts[0], f.ThousandsSeparator)
	if f.DecimalDigits == 0 {
		return intPart
	}
	return intPart + f.DecimalSeparator + parts[1]
}

// groupThousands agrupa la parte entera con el separador de miles.
// Respeta el signo negativo (ej: -1234 → "-1,234").
func groupThousands(num, sep string) string {
	if sep == "" || len(num) <= 3 {
		return num
	}
	sign := ""
	if strings.HasPrefix(num, "-") {
		sign = "-"
		num = num[1:]
	}
	var b strings.Builder
	n := len(num)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteByte(num[i])
	}
	return sign + b.String()
}

// formatAmount formatea un monto con el símbolo de la moneda según el formato
// configurado. Si symbol está vacío, devuelve solo el número formateado.
// Ejemplo: formatAmount(1500, "C$", DefaultCurrencyFormat()) → "C$1,500.00".
func formatAmount(amount float64, symbol string, format CurrencyFormat) string {
	s := format.Format(amount)
	if symbol == "" {
		return s
	}
	return symbol + s
}

// pensionAmount formatea un monto con el código de moneda (con espacio) según
// el formato configurado. Ejemplo: "NIO 1,500.00".
func pensionAmount(amount float64, currency string, format CurrencyFormat) string {
	return currency + " " + format.Format(amount)
}
