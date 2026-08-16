---
title: "Analizador de facturas Claro: internet residencial e internet móvil"
id: "SPEC-012"
status: "pending_release"
author: "paulomcnally"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 13
---

# Analizador de facturas Claro: internet residencial e internet móvil

**ID**: SPEC-012  
**Estado**: pending_release  
**Autor**: paulomcnally  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15 (implementación completa y tests exitosos)

---

## 1. Resumen Ejecutivo

El sistema de analizadores de p40la-ihost necesita un nuevo analizador para facturas de **Claro Nicaragua** (Empresa Nicaragüense de Telecomunicaciones, S.A.). Se identificaron **2 formatos distintos** de factura que el analizador debe soportar:

1. **Internet Residencial** (formato "A"): Facturas de servicios fijos (TV, Turbonett Fijo). Número de factura con prefijo `A` (ej: `A0061376765`). Incluye campos como número de contrato, ciclo, mes facturado.
2. **Internet Móvil** (formato "FAC"): Facturas de telefonía móvil. Número de factura con prefijo `FAC` (ej: `FAC0257566732026`). Incluye campos como teléfono titular, número de contrato móvil.

Ambos formatos comparten estructura visual similar (header rojo con logo Claro, tablas con headers rojos, sección de productos, footer con info de pago) pero difieren en campos específicos y formato de período.

**Consideraciones de iHost**: Se usará extracción de texto de PDF con `github.com/ledongthuc/pdf` (Go puro, compatible con Go 1.22 del proyecto, sin dependencias externas pesadas). No se usará OCR de imagen ya que los PDFs de Claro son PDFs vectoriales con texto seleccionable. Zero impacto en memoria.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Implementar analizador `claro` que detecte automáticamente si la factura es formato residencial o móvil basándose en el prefijo del número de factura (`A` = residencial, `FAC` = móvil).

2. **REQ-002**: Extraer los siguientes campos de **ambos formatos**:
   - `invoice_number`: No. de factura (ej: `A0061376765` o `FAC0257566732026`)
   - `amount`: Total a pagar (campo `TOTAL A PAGAR` en tabla roja, ej: `C$7,089.20`)
   - `year`: Año del período facturado (extraído de `MES FACTURADO` o `PERÍODO FACTURADO`)
   - `month`: Mes del período facturado (1-12)
   - `due_date`: Fecha límite de pago (campo `FECHA LÍMITE DE PAGO`)

3. **REQ-003**: Soporte para extracción de texto de PDFs vectoriales (texto seleccionable) usando librería Go nativa.

4. **REQ-004**: El analizador debe registrarse en el registry de analyzers con ID `claro` y nombre `Claro Nicaragua`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

5. **REQ-005**: Extraer campo adicional `period_label` para mostrar en UI (ej: `AGO/2026` o `Jul/2026`).

6. **REQ-006**: Soporte para imágenes (PNG/JPG) como fallback usando OCR con `gocv` o `tesseract` solo si el PDF no tiene texto extraíble.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

7. **REQ-007**: Extraer campo `client_id` (ID CLIENTE) y `contract_number` (NÚMERO DE CONTRATO o TELÉFONO TITULAR) para validación.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Análisis de PDF < 2 segundos en iHost (ARM64, recursos limitados).
- **Memoria**: Sin dependencias pesadas. `pdfcpu` es ~5MB binario.
- **iHost**: Sin servicios externos, sin red, sin Python/Node runtime.
- **Precisión**: 100% en campos P0 para PDFs vectoriales de Claro.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

**Formato Internet Residencial** (factura `A0061376765`):
- Header: "EMPRESA NICARAGÜENSE DE TELECOMUNICACIONES, S.A." + logo Claro rojo
- No. de factura: `A0061376765` (prefijo `A` + dígitos)
- Datos cliente: nombre, cédula, ID cliente, número de contrato, cupón número
- Tabla 1 (headers rojos): PERÍODO FACTURADO | FECHA DE EMISIÓN | CICLO | MES FACTURADO | FECHA LÍMITE DE PAGO
  - Período: `06/JUL/2026 - 05/AGO/2026`
  - Mes facturado: `AGO/2026`
  - Fecha límite: `04/SEP/2026`
- Tabla 2 (headers rojos): TOTAL FACTURA | COMPRAS A PLAZOS | TOTAL MES | OTROS CARGOS Y/O CRÉDITOS | SALDO PENDIENTE | *VALOR RECLAMO | TOTAL A PAGAR
  - Total a pagar: `C$7,089.20`
- Sección productos: lista de servicios con montos
- Total Factura (footer): `C$ 1,821.60`

**Formato Internet Móvil** (factura `FAC0257566732026`):
- Header: mismo header rojo con logo Claro
- No. de factura: `FAC0257566732026` (prefijo `FAC` + dígitos)
- Datos cliente: nombre, cédula, dirección, ID cliente, total de servicios, **TELÉFONO TITULAR NO.** (ej: `8412-6107`)
- Tabla 1 (headers rojos): PERÍODO FACTURADO | FECHA DE EMISIÓN | FECHA LÍMITE DE PAGO | FECHA DE ACREDITACIÓN
  - Período: `21/Jun/2026 - 20/Jul/2026`
  - Fecha límite: `17/08/2026`
- Tabla 2 (headers rojos): TOTAL FACTURA | VENTA EN PLAZOS | TOTAL MES | SALDO A FAVOR | SALDO PENDIENTE | AJUSTE | TOTAL A PAGAR
  - Total a pagar: `C$975.97`
- Sección productos: CATEGORÍA con items (Cargo basico mensual, Cargo mensual, Renta mensual por Datos GPRS, IVA FACTURADO)
- Total Factura (footer): `C$ 841.93`

**Diferencias clave entre formatos**:
| Campo | Residencial | Móvil |
|-------|-------------|-------|
| Prefijo factura | `A` | `FAC` |
| Mes facturado | Columna dedicada (`AGO/2026`) | No existe, extraer de período |
| Teléfono titular | No existe | Campo dedicado (`8412-6107`) |
| Número de contrato | Existe | Existe (móvil) |
| Formato período | `06/JUL/2026 - 05/AGO/2026` | `21/Jun/2026 - 20/Jul/2026` |
| Columnas tabla totales | 7 columnas | 7 columnas (diferentes labels) |

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| `ledongthuc/pdf` (Go puro) | Compatible con Go 1.22, binario ligero, funciona en ARM64, extrae texto de PDFs vectoriales | No funciona con PDFs escaneados (imágenes) | ✅ Seleccionada para PDFs vectoriales |
| `pdfcpu` (Go puro) | Moderno, robusto | Requiere Go >= 1.23, incompatible con el proyecto (Go 1.22) | ❌ Rechazada |
| `gocv` + Tesseract (OCR) | Funciona con PDFs escaneados | Dependencia pesada (~200MB), requiere librerías C, complejo en ARM64 | ❌ Rechazada (overkill para PDFs vectoriales) |
| Regex sobre texto extraído | Simple, sin ML | Requiere patrones precisos | ✅ Complemento de ledongthuc/pdf |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-012-001**: Extracción de texto con `ledongthuc/pdf` + regex, sin OCR
- **Contexto**: Los PDFs de Claro son generados digitalmente (texto seleccionable), no son escaneos. El proyecto usa Go 1.22.
- **Decisión**: Usar `github.com/ledongthuc/pdf` para extraer texto del PDF completo, luego aplicar regex para extraer campos.
- **Consecuencias**: Rápido, ligero, sin dependencias externas. Si en el futuro Claro envía PDFs escaneados, se necesitaría OCR como fallback.

**ADR-012-002**: Detección de formato por prefijo de número de factura
- **Contexto**: Hay 2 formatos con estructuras ligeramente diferentes.
- **Decisión**: Extraer primero el número de factura con regex `No\.? de factura:\s*(A\d+|FAC\d+)`. Si prefijo es `A` → formato residencial, si es `FAC` → formato móvil.
- **Consecuencias**: Simple, confiable. Si Claro introduce un tercer formato con otro prefijo, se agrega un nuevo parser.

**ADR-012-003**: Parser separado por formato dentro del mismo analyzer
- **Contexto**: Ambos formatos comparten campos pero con ubicaciones diferentes.
- **Decisión**: Un solo analyzer `claro` con 2 funciones internas: `parseResidential(text)` y `parseMobile(text)`. El método `Analyze` detecta el formato y delega.
- **Consecuencias**: Código organizado, fácil de mantener. Si un formato cambia, solo se modifica su parser.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[PDF upload] → [DocumentService] → [claro analyzer]
                                          │
                                          ├── pdfcpu: extractText()
                                          ├── detectFormat(): A→residential, FAC→mobile
                                          ├── parseResidential(): regex patterns
                                          └── parseMobile(): regex patterns
                                                    │
                                                    └── [ExtractedBill] → [BillStorage]
```

### 4.2 Componentes

#### 4.2.1 `internal/analyzers/all/claro.go` (nuevo archivo)
- **Responsabilidad**: Analizar facturas de Claro Nicaragua (residencial y móvil)
- **Interfaz**: Implementa `DocumentAnalyzer`
- **Dependencias**: `github.com/pdfcpu/pdfcpu/pkg/api`, `regexp`, `strings`, `strconv`
- **Ubicación**: `internal/analyzers/all/claro.go`

#### 4.2.2 `internal/analyzers/all/all.go` (modificar)
- **Cambio**: Agregar `import _ "github.com/paulomcnally/p40la-ihost/internal/analyzers/all/claro"` para registro automático

### 4.3 Modelo de datos

No cambia. Se usa `ExtractedBill` existente:
```
ExtractedBill:
- amount: float64 (Total a pagar, sin símbolo C$)
- invoice_number: string (ej: "A0061376765")
- year: int (ej: 2026)
- month: int (ej: 8 para agosto)
- due_date: *time.Time (fecha límite de pago)
- raw_data: map[string]interface{} (datos adicionales: period_label, client_id, etc.)
```

### 4.4 Patrones de extracción (regex)

**Comunes a ambos formatos**:
```go
// Número de factura
reInvoiceNumber = regexp.MustCompile(`No\.?\s*de\s*factura:\s*(A\d+|FAC\d+)`)

// Total a pagar (tabla roja, última columna)
reTotalAPagar = regexp.MustCompile(`TOTAL A PAGAR\s*\n?\s*C\$\s*([\d,]+\.?\d*)`)

// Fecha límite de pago
reFechaLimite = regexp.MustCompile(`FECHA LÍMITE DE PAGO\s*\n?\s*(\d{2}/\w{3}/\d{4})`)
```

**Formato Residencial**:
```go
// Mes facturado (columna dedicada)
reMesFacturado = regexp.MustCompile(`MES FACTURADO\s*\n?\s*(\w{3}/\d{4})`)

// Período facturado (para año)
rePeriodo = regexp.MustCompile(`PERÍODO FACTURADO\s*\n?\s*\d{2}/(\w{3})/\d{4}\s*-\s*\d{2}/(\w{3})/(\d{4})`)
```

**Formato Móvil**:
```go
// Teléfono titular
reTelefonoTitular = regexp.MustCompile(`TELÉFONO TITULAR NO\.?:\s*([\d-]+)`)

// Período (mes en formato mixed case)
rePeriodoMobile = regexp.MustCompile(`PERÍODO FACTURADO\s*\n?\s*\d{2}/(\w{3})/\d{4}\s*-\s*\d{2}/(\w{3})/(\d{4})`)
```

**Mapeo de meses**:
```go
var monthMap = map[string]int{
    "ENE": 1, "JAN": 1, "FEB": 2, "MAR": 3, "ABR": 4, "APR": 4,
    "MAY": 5, "JUN": 6, "JUL": 7, "AGO": 8, "AUG": 8, "SEP": 9,
    "OCT": 10, "NOV": 11, "DIC": 12, "DEC": 12,
}
```

### 4.5 Dependencias

- **Internas**: `internal/analyzers` (registry, interfaces)
- **Externas**: `github.com/ledongthuc/pdf` (extracción de texto de PDF)

### 4.6 Archivos a crear/modificar

| Archivo | Acción |
|---------|--------|
| `internal/analyzers/all/claro.go` | Crear - analizador Claro |
| `internal/analyzers/all/all.go` | Modificar - agregar import de claro |
| `go.mod` | Modificar - agregar dependencia pdfcpu |
| `go.sum` | Auto-generado |

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un PDF de factura Claro residencial (formato A), cuando se analiza, entonces se extrae correctamente: invoice_number, amount, year, month, due_date.
- [x] CA-002: Dado un PDF de factura Claro móvil (formato FAC), cuando se analiza, entonces se extrae correctamente: invoice_number, amount, year, month, due_date.
- [x] CA-003: Dado un PDF de Claro, cuando el número de factura empieza con `A`, entonces se usa el parser residencial.
- [x] CA-004: Dado un PDF de Claro, cuando el número de factura empieza con `FAC`, entonces se usa el parser móvil.
- [x] CA-005: Dado un monto `C$7,089.20`, cuando se extrae, entonces amount = 7089.20 (float64).
- [x] CA-006: Dado un mes `AGO/2026`, cuando se extrae, entonces month = 8, year = 2026.
- [x] CA-007: Dado una fecha `04/SEP/2026`, cuando se extrae, entonces due_date = 2026-09-04.
- [x] CA-008: El analyzer se registra con ID `claro` y nombre `Claro Nicaragua`.
- [x] CA-009: Dado un PDF que no es de Claro, cuando se analiza, entonces retorna error descriptivo.

### 5.2 No funcionales

- [ ] CA-NF-001: Análisis de PDF < 2 segundos en iHost.
- [ ] CA-NF-002: Binario final no aumenta más de 10MB por la dependencia pdfcpu.
- [ ] CA-NF-003: Sin dependencias de red externa en runtime.

### 5.3 Testing

- **Unit tests**: Tests con texto extraído de facturas reales (mock de pdfcpu output).
- **Integration tests**: Tests con archivos PDF reales de Claro (residencial y móvil).
- **Archivos de test**: Guardar los 2 PDFs reales en `internal/analyzers/all/testdata/` para tests.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Agregar dependencia `pdfcpu` a go.mod | 5 min | Ninguna |
| 2 | Crear `internal/analyzers/all/claro.go` con estructura base | 15 min | Fase 1 |
| 3 | Implementar `parseResidential()` con regex | 30 min | Fase 2 |
| 4 | Implementar `parseMobile()` con regex | 30 min | Fase 2 |
| 5 | Implementar `detectFormat()` y método `Analyze()` | 15 min | Fases 3-4 |
| 6 | Registrar analyzer en `all.go` | 5 min | Fase 5 |
| 7 | Unit tests con texto mock | 30 min | Fase 6 |
| 8 | Integration tests con PDFs reales | 20 min | Fase 7 |
| 9 | Build y verificación en iHost | 10 min | Fase 8 |

### 6.2 Milestones

1. **MVP**: Analyzer básico con detección de formato y extracción de campos P0 (Fases 1-6)
2. **V1.0**: Tests completos, build verificado (Fases 7-9)

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Claro cambia formato de factura | Media | Alto | Regex flexibles, tests con PDFs reales, fácil de actualizar |
| PDFs escaneados (sin texto) | Baja | Alto | Fallback a OCR en futura iteración. Por ahora, PDFs de Claro son vectoriales |
| pdfcpu incompatible con ARM64 | Baja | Alto | pdfcpu es Go puro, sin CGO. Compatible con todas las arquitecturas |
| Monto con formato diferente (ej: sin C$) | Media | Medio | Regex con grupo opcional para símbolo de moneda |
| Mes en formato inesperado | Baja | Medio | Mapeo de meses extensible, log de meses no reconocidos |

## 8. Notas y Referencias

- Facturas proporcionadas por el usuario: 2 formatos (residencial A0061376765, móvil FAC0257566732026)
- pdfcpu docs: https://pdfcpu.io/
- Moneda: Córdobas nicaragüenses (C$)
- Empresa: Empresa Nicaragüense de Telecomunicaciones, S.A. (Claro Nicaragua)
- RUC: J0310000003050

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | paulomcnally | Creación inicial de la especificación |
| 2026-08-15 | paulomcnally | Implementación completa: analyzer claro para formatos residencial y móvil, tests unitarios, build verificado |
| 2026-08-15 | paulomcnally | Implementación iniciada |
