# Guia para LLM: Generacion de Analyzers

> **Propuesto**: Esta guia esta disena para que un LLM externo (con capacidad de leer PDF/imagenes) genere codigo Go correcto para nuevos analyzers de facturas.
>
> **Workflow**: El usuario sube una imagen/PDF de una factura a un LLM externo -> el LLM genera el codigo usando esta guia -> el usuario trae el codigo a este proyecto -> se ejecutan tests para validar.

---

## 1. Que es un Analyzer?

Un analyzer es un componente Go que extrae datos de facturas/documentos. Cada analyzer:

- Implementa la interfaz `DocumentAnalyzer`
- Se auto-registra via `init()`
- Extrae: monto total, numero de factura, anio, mes, y fecha de vencimiento (opcional)

El sistema actualmente tiene UN solo analyzer: **Claro Nicaragua** (facturas de internet/telefonia).

---

## 2. Interfaz Obligatoria

Todo analyzer DEBE implementar exactamente esta interfaz:

```go
package analyzers

import (
    "io"
    "time"
)

type AnalyzerInfo struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type ExtractedBill struct {
    Amount        float64                `json:"amount"`
    InvoiceNumber string                 `json:"invoice_number,omitempty"`
    Year          int                    `json:"year"`
    Month         int                    `json:"month"`
    DueDate       *time.Time             `json:"due_date,omitempty"`
    RawData       map[string]interface{} `json:"raw_data,omitempty"`
}

type DocumentAnalyzer interface {
    Info() AnalyzerInfo
    Analyze(reader io.Reader, mimeType string) (*ExtractedBill, error)
}
```

### Campos de ExtractedBill

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `Amount` | `float64` | **SI** | Monto total a pagar (sin decimales raros) |
| `InvoiceNumber` | `string` | **SI** | Numero unico de la factura |
| `Year` | `int` | **SI** | Anio de la factura (ej: 2026) |
| `Month` | `int` | **SI** | Mes como numero (1-12) |
| `DueDate` | `*time.Time` | No | Fecha limite de pago |
| `RawData` | `map[string]interface{}` | No | Datos extra que quieras guardar ( opcionales) |

### Reglas criticas

- **NO usar puntero a ExtractedBill como nil**. Siempre devolver `&ExtractedBill{}`.
- Si falta un campo obligatorio, devolver **error** con `fmt.Errorf(...)`.
- El `mimeType` sera `"application/pdf"` por ahora. Rechazar otros formatos con error claro.
- Los meses van de 1 a 12 (ENERO=1, DICIEMBRE=12).

---

## 3. Estructura de Archivo

Cada analyzer es UN archivo Go en `internal/analyzers/all/`:

```
internal/analyzers/all/
├── all.go                    # NO TOCAR - solo imports
├── claro.go                  # Analyzer existente (referencia)
├── claro_test.go             # Tests del analyzer claro
├── tu_institucion.go         # <-- TU NUEVO ARCHIVO AQUI
└── tu_institucion_test.go    # <-- TUS TESTS AQUI
```

### Template minimo para tu archivo

```go
package all

import (
    "bytes"
    "fmt"
    "io"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/ledongthuc/pdf"
    "github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

func init() {
    analyzers.Register(&TuAnalyzer{})
}

type TuAnalyzer struct{}

func (a *TuAnalyzer) Info() analyzers.AnalyzerInfo {
    return analyzers.AnalyzerInfo{
        ID:   "tu_id",           // snake_case, unico, sin espacios
        Name: "Nombre Visible",  // Nombre que ve el usuario en la UI
    }
}

func (a *TuAnalyzer) Analyze(reader io.Reader, mimeType string) (*analyzers.ExtractedBill, error) {
    if mimeType != "application/pdf" {
        return nil, fmt.Errorf("formato no soportado: %s", mimeType)
    }

    // 1. Leer el PDF completo a memoria
    buf := new(bytes.Buffer)
    if _, err := buf.ReadFrom(reader); err != nil {
        return nil, fmt.Errorf("leer archivo: %w", err)
    }

    // 2. Parsear el PDF y extraer texto plano
    pdfReader, err := pdf.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
    if err != nil {
        return nil, fmt.Errorf("parsear PDF: %w", err)
    }

    var textBuilder strings.Builder
    for i := 1; i <= pdfReader.NumPage(); i++ {
        page := pdfReader.Page(i)
        content, err := page.GetPlainText(nil)
        if err != nil {
            continue
        }
        textBuilder.WriteString(content)
        textBuilder.WriteString("\n")
    }

    text := textBuilder.String()
    if strings.TrimSpace(text) == "" {
        return nil, fmt.Errorf("no se pudo extraer texto del PDF")
    }

    // 3. Extraer datos con regex
    result := &analyzers.ExtractedBill{}

    // -- Numero de factura --
    reInvoice := regexp.MustCompile(`TU_REGEX_AQUI`)
    if m := reInvoice.FindStringSubmatch(text); len(m) > 1 {
        result.InvoiceNumber = m[1]
    }

    // -- Monto total --
    // Busca el total a pagar despues de un marker como "TOTAL A PAGAR"
    // y antes de secciones como "PRODUCTOS", "DETALLE", etc.
    reAmount := regexp.MustCompile(`C\$\s*([\d,]+\.?\d*)`)
    if m := reAmount.FindStringSubmatch(text); len(m) > 1 {
        amt, err := strconv.ParseFloat(
            strings.ReplaceAll(strings.ReplaceAll(m[1], ",", ""), " ", ""),
            64,
        )
        if err == nil {
            result.Amount = amt
        }
    }

    // -- Mes y anio --
    rePeriod := regexp.MustCompile(`TU_REGEX_DE_PERIODO`)
    if m := rePeriod.FindStringSubmatch(text); len(m) > 1 {
        result.Month = parseMonth(m[1])  // usa tu map de meses
        y, _ := strconv.Atoi(m[2])
        result.Year = y
    }

    // -- Fecha de vencimiento (opcional) --
    result.DueDate = extractDueDate(text)

    // 4. Validar campos obligatorios
    if result.InvoiceNumber == "" || result.Amount == 0 || result.Year == 0 || result.Month == 0 {
        return nil, fmt.Errorf("campos obligatorios no encontrados (invoice=%s, amount=%.2f, year=%d, month=%d)",
            result.InvoiceNumber, result.Amount, result.Year, result.Month)
    }

    return result, nil
}

// --- Funciones auxiliares ---

var monthMap = map[string]int{
    "ENE": 1, "JAN": 1, "FEB": 2, "MAR": 3, "ABR": 4, "APR": 4,
    "MAY": 5, "JUN": 6, "JUL": 7, "AGO": 8, "AUG": 8, "SEP": 9,
    "OCT": 10, "NOV": 11, "DIC": 12, "DEC": 12,
}

func parseMonth(m string) int {
    return monthMap[strings.ToUpper(strings.TrimSpace(m))]
}

func extractDueDate(text string) *time.Time {
    // Implementar logica para fecha de vencimiento
    // Buscar "FECHA LIMITE DE PAGO" y extraer la fecha
    return nil
}
```

---

## 4. Como Generar los Regex (Guia para el LLM)

### Paso 1: Identificar los campos en el PDF

Cuando el usuario te muestre una imagen/PDF de factura, busca estos elementos:

| Campo | Que buscar | Ejemplo comun |
|-------|-----------|---------------|
| Numero de factura | "No. de factura", "Factura No.", "Invoice #" | `A0061376765`, `FAC0257566732026` |
| Monto total | "Total a pagar", "Total facturado", "Amount due" | `C$7,089.20`, `$1,234.56` |
| Periodo | "Periodo facturado", "Mes facturado", "Billing period" | `06/JUL/2026 - 05/AGO/2026` |
| Fecha de vencimiento | "Fecha limite de pago", "Due date" | `04/SEP/2026` |

### Paso 2: Construir los regex

**Numero de factura** - busca un prefijo + digitos:
```go
// Patron generico
regexp.MustCompile(`(?is)No\.?\s*de\s*factura:\s*([A-Z]{0,3}\d+)`)
// (?i) = case insensitive, (?s) = dotall
// El grupo ([A-Z]{0,3}\d+) captura el numero
```

**Monto total** - busca moneda + numero:
```go
// Para cordobas (Nicaragua)
regexp.MustCompile(`C\$\s*([\d,]+\.?\d*)`)
// Para dolares
regexp.MustCompile(`\$\s*([\d,]+\.?\d*)`)
// Para genericos
regexp.MustCompile(`(?:Total|TOTAL)[^C$]*C\$\s*([\d,]+\.?\d*)`)
```

**Periodo/Mes** - busca formato DD/MES/YYYY:
```go
// Busca mes abreviado en mayusculas
regexp.MustCompile(`(?is)MES FACTURADO\s+.*?\d{2}/(\w{3})/(\d{4})`)
// Captura: grupo[1] = mes, grupo[2] = anio
```

**Fecha de vencimiento** - busca la fecha mas lejana en la seccion de pago:
```go
// Buscar "FECHA LIMITE DE PAGO" y extraer fechas del contexto
```

### Paso 3: Validar que los regex funcionan

El regex DEBE:
1. Usar grupos de captura `(...)` para extraer valores
2. Ser case-insensitive `(?is)` o `(?i)` cuando el texto puede tener variaciones
3. Manejar espacios opcionales `\s*`
4. No ser greedy (usar `.*?` en vez de `.*`)

---

## 5. Patron de Extraccion de Monto (detallado)

El patron mas comun es: buscar un marker ("TOTAL A PAGAR"), luego extraer el ultimo monto C$ antes de la siguiente seccion.

```go
func extractAmount(text string) float64 {
    idxTotal := strings.Index(strings.ToUpper(text), "TOTAL A PAGAR")
    if idxTotal < 0 {
        return 0
    }

    rest := text[idxTotal:]

    // Cortar en la siguiente seccion conocida
    endIdx := len(rest)
    for _, marker := range []string{"PRODUCTOS", "DETALLE", "PUNTOS DE PAGO"} {
        if i := strings.Index(rest, marker); i > 0 && i < endIdx {
            endIdx = i
        }
    }
    section := rest[:endIdx]

    // Extraer todos los montos C$ y tomar el ultimo
    reTotal := regexp.MustCompile(`C\$\s*([\d,]+\.?\d*)`)
    matches := reTotal.FindAllStringSubmatch(section, -1)
    if len(matches) == 0 {
        return 0
    }

    last := matches[len(matches)-1]
    s := strings.ReplaceAll(last[1], ",", "")
    s = strings.TrimSpace(s)
    amt, _ := strconv.ParseFloat(s, 64)
    return amt
}
```

---

## 6. Tests Obligatorios

Cada analyzer DEBE tener tests. Archivo: `internal/analyzers/all/tu_institucion_test.go`

### Estructura basica de tests

```go
package all

import (
    "strings"
    "testing"

    "github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

// Texto de ejemplo EXTRAIDO del PDF real (copia el texto plano)
const testText = `
AQUI EL TEXTO COMPLETO QUE EL LLM EXTRAE DEL PDF
No. de factura: ABC123456
TOTAL A PAGAR
C$1,234.56
`

func TestTuAnalyzer_Info(t *testing.T) {
    a := &TuAnalyzer{}
    info := a.Info()
    if info.ID != "tu_id" {
        t.Errorf("esperado ID 'tu_id', obtenido '%s'", info.ID)
    }
    if info.Name != "Nombre Visible" {
        t.Errorf("esperado Name 'Nombre Visible', obtenido '%s'", info.Name)
    }
}

func TestTuAnalyzer_Extract(t *testing.T) {
    result, err := parseTuFormat(testText)
    if err != nil {
        t.Fatalf("error: %v", err)
    }
    if result.InvoiceNumber != "ABC123456" {
        t.Errorf("InvoiceNumber = %q, want %q", result.InvoiceNumber, "ABC123456")
    }
    if result.Amount != 1234.56 {
        t.Errorf("Amount = %f, want %f", result.Amount, 1234.56)
    }
    if result.Year != 2026 {
        t.Errorf("Year = %d, want %d", result.Year, 2026)
    }
    if result.Month != 8 {
        t.Errorf("Month = %d, want %d", result.Month, 8)
    }
}

func TestTuAnalyzer_MissingFields(t *testing.T) {
    _, err := parseTuFormat("texto sin datos de factura")
    if err == nil {
        t.Error("esperado error por campos faltantes")
    }
    if !strings.Contains(err.Error(), "campos obligatorios") {
        t.Errorf("error inesperado: %v", err)
    }
}

func TestTuAnalyzer_Registered(t *testing.T) {
    a, ok := analyzers.Get("tu_id")
    if !ok {
        t.Fatal("analyzer 'tu_id' no esta registrado")
    }
    info := a.Info()
    if info.Name != "Nombre Visible" {
        t.Errorf("Name = %q, want %q", info.Name, "Nombre Visible")
    }
}
```

### Que probar obligatoriamente

1. **Info()** - que devuelva ID y Name correctos
2. **Extraccion exitosa** - con texto real de la factura
3. **Campos faltantes** - que devuelva error cuando falta info
4. **Registro** - que el analyzer este en el registry

---

## 7. Checklist para el LLM Generador

Antes de entregar el codigo, verifica:

- [ ] El archivo esta en `internal/analyzers/all/tu_institucion.go`
- [ ] El package es `package all`
- [ ] La funcion `init()` llama `analyzers.Register(&TuAnalyzer{})`
- [ ] `Info()` devuelve un ID unico (snake_case) y un Name descriptivo
- [ ] `Analyze()` rechaza mimeType != "application/pdf"
- [ ] `Analyze()` lee el PDF completo con `pdf.NewReader`
- [ ] Extrae texto de TODAS las paginas
- [ ] Usa regex para extraer cada campo
- [ ] Valida que los campos obligatorios esten presentes
- [ ] Devuelve error claro si falta algo
- [ ] Los meses van de 1 a 12
- [ ] El monto no tiene signos de moneda ni comas en el float
- [ ] NO hay imports sin usar
- [ ] El codigo compila (verificar con `go build ./...`)

---

## 8. Errores Comunes a Evitar

| Error | Solucion |
|-------|----------|
| Usar `nil` como puntero de ExtractedBill | Siempre devolver `&ExtractedBill{}` |
| Mes = 0 | Verificar que el map de meses cubre todas las abreviaciones |
| Monto con comas | Usar `strings.ReplaceAll(s, ",", "")` antes de `ParseFloat` |
| Regex no captura nada | Probar el regex con el texto real primero |
| Olvidar validar campos obligatorios | Siempre al final: `if invoice == "" \|\| amount == 0 \|\| ...` |
| No rechazar otros mime types | Primera linea de Analyze: `if mimeType != "application/pdf"` |
| Import circular | El package es `all`, el import path es `github.com/paulomcnally/p40la-ihost/internal/analyzers` |
| Olvidar el `init()` | Sin `init()`, el analyzer no se registra y nunca se usa |

---

## 9. Formato de Entrega

Cuando el LLM genere el codigo, debe entregar:

1. **Archivo principal**: `tu_institucion.go` con el analyzer completo
2. **Archivo de tests**: `tu_institucion_test.go` con tests basicos
3. **Texto extraido del PDF**: copia del texto plano que uso para validar los regex (para que el usuario pueda crear mas tests)

### Mensaje de ejemplo para el LLM

```
Aqui tienes la factura de [Institucion]. Generame el analyzer Go para extraer:
- Numero de factura
- Monto total
- Mes y anio facturado
- Fecha de vencimiento (si aplica)

Usa la guia en docs/analyzers/LLM-GUIDE.md como referencia.
El ID del analyzer debe ser "[institucion]_[tipo]" (ej: "tigo_hogar", "cable_color").
```

---

## 10. Referencia Rapida de Imports

```go
package all

import (
    "bytes"                                          // buffers
    "fmt"                                            // errores
    "io"                                             // interfaz reader
    "regexp"                                         // regex
    "strconv"                                        // ParseFloat, Atoi
    "strings"                                        // manipulacion de strings
    "time"                                           // time.Time, time.Date

    "github.com/ledongthuc/pdf"                      // parsear PDFs
    "github.com/paulomcnally/p40la-ihost/internal/analyzers"  // interfaz + registry
)
```

**NO usar**:
- `os` (no se leen archivos del filesystem)
- `net/http` (no hay requests)
- `database/sql` (no hay DB en el analyzer)
- Ninguna dependencia externa que no este en `go.mod`
