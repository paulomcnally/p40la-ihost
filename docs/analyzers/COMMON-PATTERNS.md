# Patrones Comunes de Facturas (Nicaragua)

Referencia de regex para los analyzers mas comunes.

---

## Claro (ya implementado)

**Tipos**: Residencial (prefijo `A`) y Movil (prefijo `FAC`)

```go
// Numero de factura
regexp.MustCompile(`(?is)No\.?\s*de\s*factura:\s*(A\d+|FAC\d+)`)

// Monto total
regexp.MustCompile(`C\$\s*([\d,]+\.?\d*)`)

// Periodo facturado (ej: "06/JUL/2026 - 05/AGO/2026")
regexp.MustCompile(`(?is)MES FACTURADO\s+.*?\d{2}/\w+/\d{4}\s*-\s*\d{2}/(\w{3})/(\d{4})`)
```

---

## Tigo (ejemplo - NO implementado)

**Posibles patrones** (verificar con factura real):

```go
// Numero de factura - Tigo usa formatos variados
regexp.MustCompile(`(?is)(?:Factura|Invoice|No\.?)\s*(?:No\.?)?\s*[:#]?\s*(\w+\d+)`)

// Monto total - puede usar Q (quetzales) o $ (dolares)
regexp.MustCompile(`(?:Total|TOTAL|Total a pagar)[^Q$]*[Q$]\s*([\d,]+\.?\d*)`)

// Periodo
regexp.MustCompile(`(?is)Periodo\s*[:#]?\s*(\d{2})/(\w+)/(\d{4})`)
```

---

## Cable Color / Gotel (ejemplo)

```go
// Numero de factura - formato comun en cable
regexp.MustCompile(`(?is)Factura\s*(?:No\.?|#)?\s*[:#]?\s*(\d{4,})`)

// Monto
regexp.MustCompile(`(?:Total|TOTAL)[^C$]*C\$\s*([\d,]+\.?\d*)`)
```

---

## Patron generico para cualquier factura

Si no sabes el formato exacto, usa este patron generico que busca keywords comunes:

```go
// Buscar numero de factura - busca "factura" + cualquier numero
reInvoice := regexp.MustCompile(`(?is)(?:factura|invoice|bill|recibo)\s*(?:no\.?|#|:)?\s*[:#]?\s*([A-Z]{0,3}\d{4,})`)

// Buscar monto - busca simbolo de moneda + numero
reAmount := regexp.MustCompile(`(?:C\$|Q\$|\$|USD|NIO)\s*([\d,]+\.?\d*)`)

// Buscar periodo - busca patron DD/MES/YYYY
rePeriod := regexp.MustCompile(`(\d{2})/(?:ENE|FEB|MAR|ABR|MAY|JUN|JUL|AGO|SEP|OCT|NOV|DIC|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)/(\d{4})`)

// Buscar fecha de vencimiento
reDue := regexp.MustCompile(`(?is)(?:vencimiento|limite|due|fecha de pago)\s*[:#]?\s*(\d{2}/\w+/\d{4})`)
```

---

## Mapa de meses (copia para tu analyzer)

```go
var monthMap = map[string]int{
    "ENE": 1, "JAN": 1,
    "FEB": 2,
    "MAR": 3,
    "ABR": 4, "APR": 4,
    "MAY": 5,
    "JUN": 6,
    "JUL": 7,
    "AGO": 8, "AUG": 8,
    "SEP": 9,
    "OCT": 10,
    "NOV": 11,
    "DIC": 12, "DEC": 12,
}

func parseMonth(m string) int {
    return monthMap[strings.ToUpper(strings.TrimSpace(m))]
}
```

---

## Monedas comunes en la region

| Pais | Simbolo | Regex |
|------|---------|-------|
| Nicaragua | C$ | `C\$\s*([\d,]+\.?\d*)` |
| Guatemala | Q | `Q\s*([\d,]+\.?\d*)` |
| El Salvador | $ | `\$\s*([\d,]+\.?\d*)` |
| Honduras | L | `L\s*([\d,]+\.?\d*)` |
| Costa Rica | ₡ | `₡\s*([\d,]+\.?\d*)` |

---

## Tips para el LLM

1. **Siempre extraer el texto plano primero** antes de escribir regex
2. **Probar regex con el texto real** - no asumas el formato
3. **Los meses pueden venir en espanol o ingles** - cubrir ambos
4. **Los montos pueden tener comas** - eliminarlas antes de ParseFloat
5. **Busca markers** ("TOTAL A PAGAR", "Factura No.") para ubicar la zona correcta
6. **Corta el texto** en secciones para evitar falsos positivos
