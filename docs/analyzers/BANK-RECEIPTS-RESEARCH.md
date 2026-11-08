---
title: "Reconocimiento de comprobantes bancarios (nivel proyecto)"
status: "research"
created: "2026-11-08"
updated: "2026-11-08"
related_spec: "SPEC-127"
---

# Reconocimiento de comprobantes bancarios (nivel proyecto)

> **Nota de investigación para una spec futura.** No implementa código ni define todavía un contrato definitivo. Todos los ejemplos son sintéticos: no contiene nombres, cuentas, referencias, montos ni PDFs reales.

---

## 1. Objetivo

Documentar el conocimiento obtenido al leer comprobantes de pago bancarios en PDF, con foco en:

- Cómo identificar el **ID del comprobante** (referencia de la transacción).
- Cómo extraer **fecha, monto, moneda y número de cuota**.
- Cómo asociar cada comprobante a un pago ya registrado sin confiar ciegamente en el texto del banco.
- Qué casos ambiguos y anomalías debe manejar un reconocedor a nivel proyecto.

La idea es que este documento sirva como insumo para una spec de un **sistema interno de reconocimiento de documentos**, reutilizable por deudas, servicios o cualquier módulo que tenga pagos. No debe ser un analyzer atado a una institución, servicio o deuda específica.

---

## 2. Alcance y diferencia con los analyzers actuales

El proyecto ya tiene analyzers de facturas en `internal/analyzers/`:

- `DocumentAnalyzer` devuelve un `ExtractedBill` orientado a facturas de servicios: `Amount`, `InvoiceNumber`, `Year`, `Month`, `DueDate`.
- Los analyzers se registran por institución/servicio y están pensados para facturas, no para comprobantes de pago.

Un comprobante bancario necesita un resultado distinto:

- Tipo de documento (`nota_de_debito`, `confirmacion_transferencia`, etc.).
- **Referencia/ID del comprobante**.
- Fecha y hora del pago.
- Monto y moneda originales.
- Cuota o concepto detectado.
- Nivel de confianza y advertencias.
- Hash del documento para idempotencia.

Por eso se propone un reconocedor **project-level**, separado del registry de analyzers de instituciones. La spec futura debe decidir si vive en `internal/recognizers/`, `internal/documents/` o una capa equivalente.

---

## 3. Formatos observados

### 3.1 Formato A — Nota de débito bancaria

Layout de una nota de débito (ejemplo sintético; el banco real puede variar):

```text
Comprobante
Hora 09:15:22 (GMT-0600) | Fecha: 01/02/2026
Nota de débito

Nombre            Cliente Demo
Fecha             01/02/2026
Referencia        902000001
Cuenta bancaria   *****0000
Concepto
Detalle           Lote Demo - Cuota 7
Moneda            Dólares
Monto             123.45

Banco Demo
```

Campos relevantes:

| Campo | Ejemplo sintético | Notas |
|---|---|---|
| Encabezado | `Comprobante` / `Nota de débito` | El título superior puede decir "Comprobante"; el tipo real está en "Nota de débito". |
| Hora/Fecha de emisión | `09:15:22 (GMT-0600)` / `01/02/2026` | La fecha de emisión puede diferir de la fecha del pago. |
| Nombre | `Cliente Demo` | PII: no almacenar ni loguear. |
| Fecha del pago | `01/02/2026` | Usar esta fecha para el pago, no la de emisión. |
| Referencia | `902000001` | **ID del comprobante**. Es el campo principal para asociar. |
| Cuenta bancaria | `*****0000` | PII: no usar para matching ni almacenar. |
| Detalle | `Lote Demo - Cuota 7` | Puede traer `Cuota N` o `Lote X - Cuota N`. **No siempre es confiable** (ver §6). |
| Moneda | `Dólares` / `Córdobas` | Puede diferir de la moneda de la deuda. |
| Monto | `123.45` / `4,567.89` | Formato con coma de miles y punto decimal. |
| Banco | `Banco Demo` | Sirve como ancla de formato, no como dato de negocio. |

### 3.2 Formato B — Confirmación de transferencia

Layout de una confirmación de transferencia (ejemplo sintético):

```text
Confirmación de Transferencia

Número de Comprobante           52700001
De                              Cuenta de...0000
Para                            Proveedor Demo ****-0000
Fecha y Hora                    01/02/2026 - 23:02
Monto debitado                  $ 123.45
Tipo de Cambio                  $ 1.00
Monto acreditado                $ 123.45
Motivo                          Lote Demo - Cuota 40
```

Campos relevantes:

| Campo | Ejemplo sintético | Notas |
|---|---|---|
| Tipo | `Confirmación de Transferencia` | Ancla de formato. |
| Número de Comprobante | `52700001` | **ID del comprobante** en este formato. |
| Fecha y Hora | `01/02/2026 - 23:02` | Formato distinto al Formato A. |
| Monto debitado / acreditado | `123.45` | Usar el monto debitado para el pago; el acreditado puede variar por comisiones. |
| Tipo de Cambio | `1.00` | Útil si la moneda de origen y destino difieren. |
| Motivo | `Lote Demo - Cuota 40` | Equivalente a `Detalle` en el Formato A. |
| De / Para | `Cuenta de...0000` / `Proveedor Demo` | PII: no almacenar ni loguear. |

---

## 4. Extracción del ID del comprobante (referencia)

El ID del comprobante es el dato más importante para asociar un PDF a un pago. Debe extraerse con anclas específicas por formato, no con “el primer número largo del documento”.

### 4.1 Reglas propuestas

```text
Formato A (nota de débito):
  Referencia\s+(\d+)

Formato B (confirmación de transferencia):
  N[uú]mero de Comprobante\s+(\d+)
```

Recomendaciones:

- Guardar la referencia como **texto**, no como entero, para preservar ceros a la izquierda y evitar límites numéricos.
- Normalizar con `strings.TrimSpace`; no quitar guiones si el banco los usa (en los formatos observados son solo dígitos).
- Validar que sea numérica y de longitud razonable (por ejemplo, 6–20 dígitos) antes de aceptarla.
- No usar `Cuenta bancaria`, `Nombre`, `De` ni `Para` como identificadores.
- Si el PDF no tiene `Referencia` ni `Número de Comprobante`, el documento debe quedar en estado `needs_review`; no inventar un ID.
- Si hay varias referencias en el texto (por ejemplo, una referencia de lote y una de transacción), priorizar el campo etiquetado `Referencia` o `Número de Comprobante`; nunca el primer número que aparezca.

### 4.2 Hash del documento

Además de la referencia, calcular un hash SHA-256 del PDF para:

- Detectar el mismo archivo subido dos veces.
- Evitar re-procesar documentos idénticos.
- Auditar sin guardar el PDF completo en la DB.

---

## 5. Extracción de cuota, fecha, monto y moneda

### 5.1 Cuota / concepto

Anclas:

```text
Cuota\s+(\d+)
Detalle\s+(.+)
Motivo\s+(.+)
```

Ejemplos sintéticos:

- `Lote Demo - Cuota 7` → cuota `7`.
- `Cuota 7` → cuota `7`.
- `Lote Demo -Cuota 37` (sin espacio antes de `Cuota`) → cuota `37`.

El número de cuota extraído es una **señal**, no una verdad absoluta. Ver §6.

### 5.2 Fecha y hora

Formatos observados:

```text
Formato A: Fecha:\s*(\d{2}/\d{2}/\d{4}) + Hora(\d{2}:\d{2}:\d{2})
Formato B: Fecha y Hora\s+(\d{2}/\d{2}/\d{4}) - (\d{2}:\d{2})
```

Recomendaciones:

- Parsear como fecha local del banco (`dd/mm/yyyy`), no como ISO.
- Conservar la hora para desambiguar pagos del mismo día.
- Guardar `paid_at` como fecha/hora con zona horaria si el formato la informa (`GMT-0600`); si no, guardar fecha y hora locales del documento.
- No usar la fecha de emisión del comprobante como fecha de pago si existe una fecha específica de pago.

### 5.3 Monto y moneda

Anclas:

```text
Formato A: Moneda\s+(Dólares|Córdobas) + Monto\s+([\d,]+\.\d{2})
Formato B: Monto debitado\s+\$?\s*([\d,]+\.\d{2})
```

Recomendaciones:

- Parsear montos con coma de miles y punto decimal (`4,567.89` → `4567.89`).
- Guardar `amount` como decimal y `currency` como código ISO (`USD`, `NIO`) o como texto normalizado.
- No asumir que la moneda del comprobante es la misma que la de la deuda/servicio.
- Si la deuda está en USD y el comprobante en NIO, guardar ambos datos y, si se conoce el tipo de cambio, el equivalente. No convertir silenciosamente sin registrar la tasa.
- El Formato B tiene `Monto debitado` y `Monto acreditado`: usar el debitado para el pago; el acreditado puede ser distinto por comisiones.

---

## 6. El problema del campo `Detalle` (caso crítico)

Durante la lectura de una serie de comprobantes se observó un patrón que el reconocedor **debe** manejar:

- El campo `Detalle` del banco puede repetir la cuota anterior en una transacción posterior.
- Los comprobantes son transacciones distintas: tienen **referencia, hora y, a veces, monto diferentes**.
- La secuencia de fechas y la numeración de los archivos permiten inferir que la transacción corresponde a la cuota siguiente, aunque el banco haya impreso mal el `Detalle`.

Ejemplo sintético del patrón (no son datos reales):

```text
Archivo: comprobante_06.pdf
  Referencia: 902000006
  Fecha: 01/06/2026
  Detalle: Lote Demo - Cuota 6

Archivo: comprobante_07.pdf
  Referencia: 902000007
  Fecha: 01/06/2026   ← misma fecha, otra hora/referencia
  Detalle: Lote Demo - Cuota 6   ← el banco repite la cuota anterior

Archivo: comprobante_08.pdf
  Referencia: 902000008
  Fecha: 01/07/2026
  Detalle: Lote Demo - Cuota 8
```

Conclusión: **no asociar únicamente por `Detalle`**. Hay que combinar señales y marcar conflictos.

### 6.1 Señales de matching propuestas

| Señal | Peso | Observaciones |
|---|---|---|
| Referencia ya registrada para esa cuota | Muy alto | Match exacto; si la referencia ya está en otra cuota, es conflicto. |
| Monto + fecha del pago | Alto | Útil cuando el `Detalle` falta o es dudoso. |
| `Detalle`/`Motivo` con `Cuota N` | Medio | Puede repetir la cuota anterior. |
| Nombre/patrón del archivo (`..._07.pdf`) | Bajo | Solo como pista; nunca como verdad absoluta. |
| Secuencia esperada de cuotas/fechas | Medio | Ayuda a detectar un `Detalle` atrasado o adelantado. |

### 6.2 Algoritmo sugerido

1. Extraer todos los campos del PDF.
2. Buscar un pago candidato por **referencia exacta**.
3. Si no hay referencia, buscar por **monto + fecha** (con tolerancia si hay comisión/tipo de cambio).
4. Usar `Detalle` y la secuencia de cuotas como desempate.
5. Si hay más de un candidato o el `Detalle` contradice la secuencia:
   - No escribir en la DB automáticamente.
   - Devolver `needs_review` con una lista de candidatos y advertencias.
6. Si el usuario confirma, recién ahí actualizar `payment_reference`, `paid_at`, etc.
7. Ser idempotente: si la referencia ya está asociada a la cuota, no volver a escribir.

### 6.3 Regla de oro

> Un reconocedor puede **sugerir** una asociación, pero no debe **sobrescribir** datos de pago existentes sin confirmación explícita cuando la confianza no es alta.

---

## 7. Casos observados que el reconocedor debe soportar

1. **Dos pagos el mismo día**: misma fecha, distinta hora y referencia. No colapsar por fecha.
2. **`Detalle` repetido**: el banco imprime la cuota anterior; usar referencia/fecha/secuencia y marcar advertencia.
3. **Monto con pequeña variación**: comisiones, ajustes o tipo de cambio. No exigir igualdad exacta sin tolerancia.
4. **Moneda distinta a la deuda**: deuda en USD pagada en NIO (o viceversa). Guardar monto/moneda originales y el equivalente si se conoce.
5. **Formatos distintos**: nota de débito vs confirmación de transferencia; usar anclas separadas.
6. **Comprobantes faltantes**: no todas las cuotas tienen PDF; el reconocedor no debe asumir que la serie está completa.
7. **PDFs sin capa de texto**: si el PDF es una imagen escaneada, la extracción de texto falla; debe devolver `needs_ocr` o `unsupported` (OCR fuera del MVP salvo que la spec lo incluya).
8. **Referencias largas o con ceros**: guardar como texto.
9. **Fechas `dd/mm/yyyy`**: no interpretar como `mm/dd/yyyy`.
10. **Encabezado engañoso**: el título superior puede decir "Comprobante"; el tipo real está en "Nota de débito" o "Confirmación de Transferencia".

---

## 8. Modelo de datos propuesto (para la spec futura)

No implementar todavía; es una propuesta para discutir.

```text
Entidad: RecognizedDocument
- id
- document_hash (SHA-256, único)
- recognizer_id (ej. "bank_receipt_v1")
- document_type ("nota_debito" | "confirmacion_transferencia" | "unknown")
- reference (texto, ID del comprobante)
- paid_at (fecha/hora)
- amount (decimal)
- currency (texto normalizado)
- amount_usd_equivalent (opcional)
- exchange_rate (opcional)
- installment_number (opcional)
- concept_text (Detalle/Motivo, sanitizado)
- entity_type ("debt" | "service" | "other")
- entity_id (opcional)
- confidence (0..1)
- status ("recognized" | "needs_review" | "associated" | "rejected")
- warnings (JSON/texto)
- created_at / updated_at
```

Notas:

- No guardar `Nombre`, `Cuenta bancaria`, `De`, `Para` ni el texto crudo completo: son PII.
- `concept_text` puede contener un lote o descripción; sanitizar antes de persistir.
- Si se guarda el PDF, hacerlo en el volumen (`documents/...`) y no en la DB.
- Si se guarda el resultado de reconocimiento, una tabla nueva es aceptable; no modificar el esquema de `debt_bills` sin spec.

---

## 9. Arquitectura propuesta

### 9.1 Interfaz

Separar el reconocedor de comprobantes del `DocumentAnalyzer` de facturas:

```go
type RecognizerInfo struct {
    ID   string
    Name string
}

type RecognizedDocument struct {
    DocumentType      string
    Reference         string
    PaidAt            time.Time
    Amount            float64
    Currency          string
    InstallmentNumber int
    Concept           string
    Confidence        float64
    Warnings          []string
}

type ReceiptRecognizer interface {
    Info() RecognizerInfo
    Recognize(reader io.Reader, mimeType string) (*RecognizedDocument, error)
}
```

La spec puede preferir un `DocumentExtraction` más genérico; lo importante es no reutilizar `ExtractedBill` para comprobantes de pago.

### 9.2 Detección y registry

- Detectar el formato por anclas en el texto extraído:
  - `Nota de débito` + `Referencia` → Formato A.
  - `Confirmación de Transferencia` + `Número de Comprobante` → Formato B.
- Registrar reconocedores en un registry project-level, separado del registry de analyzers de instituciones.
- Devolver `unsupported_format` si ninguna ancla coincide.

### 9.3 Extracción de texto

- Reutilizar el patrón existente de `internal/analyzers/all/assa_seguro_auto.go`: `github.com/ledongthuc/pdf` para extraer texto en Go, sin depender de `pdftotext` en runtime.
- `pdftotext -layout` fue útil para investigación local, pero **no** debe ser una dependencia del iHost.
- Si el PDF no tiene texto, devolver `needs_ocr`; no intentar OCR en el MVP sin spec.

### 9.4 API y UI propuestas

- `POST /api/documents/recognize` (multipart, PDF) → preview del resultado, sin escribir en la DB.
- `POST /api/documents/{id}/associate` (opcional) → asociación confirmada por el usuario.
- UI de revisión: mostrar referencia, fecha, monto, moneda, cuota detectada, confianza y advertencias; permitir corregir antes de confirmar.
- No asociar automáticamente pagos existentes sin confirmación cuando hay conflictos.

### 9.5 iHost

- Sin dependencias nuevas pesadas; preferir Go puro.
- Límite de tamaño (ej. 10 MB), validación de firma `%PDF-`, lectura en streaming.
- Sin llamadas externas obligatorias.
- Memoria acotada; evitar cargar el PDF completo si se puede procesar por páginas.

---

## 10. Seguridad y privacidad

- Los comprobantes contienen **PII** (nombre, cuenta bancaria, referencias) y datos financieros. No commitear PDFs ni fixtures reales.
- Usar PDFs sintéticos en tests y documentación.
- No loguear el texto extraído completo; loguear solo IDs de reconocimiento, tipo de documento y códigos de error.
- Sanitizar cualquier texto que se muestre en la UI o se persista.
- Validar autenticación, tamaño y tipo de archivo antes de procesar.
- Guardar los archivos en el volumen Docker, nunca en el repo.

---

## 11. Preguntas abiertas para la spec

1. ¿El reconocimiento se ejecuta al subir el archivo, al pagar o bajo demanda?
2. ¿Se guarda el resultado en una tabla nueva o se calcula on-the-fly?
3. ¿Cómo se asocia a deudas, servicios y otros módulos sin acoplar el reconocedor a uno solo?
4. ¿Se permite asociación automática cuando la confianza es alta, o siempre requiere confirmación?
5. ¿Cómo se maneja la conversión NIO/USD y el tipo de cambio?
6. ¿Se soporta OCR para PDFs escaneados en el MVP?
7. ¿Qué campos PII se deben enmascarar en la UI y en los logs?
8. ¿Cómo se evita duplicar un comprobante ya subido (hash + referencia)?
9. ¿Qué hacer si la referencia ya está asociada a otra cuota?
10. ¿Cómo se audita quién asoció o corrigió un documento?

---

## 12. Referencias internas

- `internal/analyzers/analyzer.go` — interfaz `DocumentAnalyzer` y `ExtractedBill` (orientada a facturas).
- `internal/analyzers/registry.go` — registry de analyzers.
- `internal/analyzers/all/assa_seguro_auto.go` — ejemplo de extracción de texto PDF y regex en Go.
- `docs/analyzers/COMMON-PATTERNS.md` — patrones de analyzers.
- `docs/analyzers/LLM-GUIDE.md` — guía para analizadores.
- `docs/analyzers/PROMPT-TEMPLATE.md` — template de prompts.
- `docs/specs/SPEC-122-comprobantes-pdf-pagos-deudas.md` — almacenamiento y asociación de facturas/comprobantes por cuota.
- `docs/ssh-ihost-access.md` — volumen `/app/data` y operación de la DB.
- `docs/project-rules.md` §8 — datos privados y secretos.

---

## 13. Historial

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-08 | opencode | Creación de la nota de investigación a partir de la lectura de comprobantes bancarios reales, sin incluir datos reales. |
