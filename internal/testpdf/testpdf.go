// Package testpdf genera PDFs mínimos con texto para tests. Solo se usa en
// pruebas: no contiene datos reales ni se enlaza en el binario del servidor.
package testpdf

import (
	"bytes"
	"fmt"
	"strings"
)

// Build devuelve un PDF de una página con las líneas indicadas usando la
// fuente Helvetica. El texto debe ser ASCII para evitar problemas de
// codificación en la fuente estándar.
func Build(lines []string) []byte {
	var content strings.Builder
	content.WriteString("BT\n/F1 11 Tf\n50 760 Td\n14 TL\n")
	for _, line := range lines {
		content.WriteString("(")
		content.WriteString(escape(line))
		content.WriteString(") Tj\nT*\n")
	}
	content.WriteString("ET\n")
	return build(content.String())
}

func escape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "(", `\(`)
	value = strings.ReplaceAll(value, ")", `\)`)
	return value
}

func build(content string) []byte {
	streamData := content + "\n"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(streamData), streamData),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)

	return buf.Bytes()
}
