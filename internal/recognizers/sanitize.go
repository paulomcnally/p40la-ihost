package recognizers

import (
	"regexp"
	"strings"
)

var (
	reConceptEmail      = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	reConceptLongDigits = regexp.MustCompile(`\d{8,}`)
)

// SanitizeConcept limpia y acota el texto libre del comprobante antes de
// persistirlo. Redacta emails y secuencias largas de dígitos (posibles
// cuentas, teléfonos o referencias) y elimina caracteres de control.
// No se guardan nombres, cuentas ni el texto completo del PDF (SPEC-127).
func SanitizeConcept(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	value = reConceptEmail.ReplaceAllString(value, "***")
	value = reConceptLongDigits.ReplaceAllString(value, "***")

	runes := []rune(value)
	if len(runes) > 200 {
		value = string(runes[:200])
	}
	return strings.TrimSpace(value)
}
