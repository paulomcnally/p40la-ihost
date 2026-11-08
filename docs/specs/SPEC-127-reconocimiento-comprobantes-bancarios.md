---
title: "Sistema interno de reconocimiento de comprobantes bancarios (nivel proyecto)"
id: "SPEC-127"
status: "in_progress"
author: "opencode"
created: "2026-11-08"
updated: "2026-11-08"
github_issue: 130
---

# Sistema interno de reconocimiento de comprobantes bancarios (nivel proyecto)

**ID**: SPEC-127
**Estado**: in_progress
**Autor**: opencode
**Creado**: 2026-11-08
**Actualizado**: 2026-11-08

---

## 1. Resumen Ejecutivo

El proyecto ya tiene analyzers de facturas por institución (`internal/analyzers`), pero no tiene una forma genérica de leer **comprobantes de pago bancarios** y asociarlos a pagos ya registrados. Hoy esa asociación se hace a mano: se lee el PDF, se identifica la referencia, la fecha, el monto y la cuota, y se actualiza la DB con SQL.

Esta spec define un **sistema interno de reconocimiento de documentos a nivel proyecto**, separado de los analyzers de instituciones y servicios. El sistema recibe un PDF, detecta el formato, extrae los campos relevantes (ID del comprobante, fecha/hora, monto, moneda, cuota/concepto), calcula un nivel de confianza, y devuelve una **vista previa** para que el usuario revise y confirme la asociación. Nunca asocia ni sobrescribe datos de pago automáticamente cuando hay ambigüedad.

El primer caso de uso es asociar comprobantes bancarios a cuotas de deudas ya pagadas: guardar el PDF en `{DATA_DIR}/documents/{debt_id}/receipt_{n}.pdf` y actualizar `debt_bills.payment_reference` con el ID del comprobante. La arquitectura queda preparada para reutilizarse en servicios y otros módulos en fases posteriores.

Consideraciones iHost: sin dependencias nuevas pesadas, usando la librería Go de PDF que ya usa el proyecto (`github.com/ledongthuc/pdf`), sin OCR en el MVP y sin llamadas de red externas. El reconocimiento es stateless salvo por el registro de la vista previa; el PDF final se guarda en el volumen Docker junto a SQLite.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear un paquete de reconocimiento **a nivel proyecto**, separado de `internal/analyzers` y del registry de instituciones. No debe depender de `Institution`, `Service` ni `Debt` concretos; debe exponer un contrato genérico de documento reconocido.
2. **REQ-002**: Detectar como mínimo estos dos formatos:
   - **Formato A**: nota de débito bancaria (ancla `Nota de débito` + campo `Referencia`).
   - **Formato B**: confirmación de transferencia (ancla `Confirmación de Transferencia` + campo `Número de Comprobante`).
   - Devolver `unsupported_format` si no coincide con ninguno.
3. **REQ-003**: Extraer y normalizar:
   - **ID del comprobante**: `Referencia` (Formato A) o `Número de Comprobante` (Formato B). Guardar como texto.
   - Fecha y hora del pago (formatos `dd/mm/yyyy` y `dd/mm/yyyy - HH:MM`).
   - Monto y moneda originales (`Dólares`/`Córdobas` → `USD`/`NIO`).
   - `Detalle`/`Motivo` y número de cuota (si aparece `Cuota N`).
   - Tipo de documento y layout detectado.
4. **REQ-004**: Calcular un **hash SHA-256** del PDF y usarlo para idempotencia y detección de duplicados. Si el mismo hash ya fue reconocido/asociado, no volver a procesarlo sin confirmación.
5. **REQ-005**: Implementar un motor de **matching con confianza y advertencias**. Señales:
   - Referencia exacta contra pagos/registros existentes (peso máximo).
   - Monto + fecha (peso alto).
   - `Detalle`/`Motivo` con `Cuota N` (peso medio; puede repetir la cuota anterior).
   - Nombre/patrón del archivo y secuencia de cuotas (peso bajo).
   Si hay conflicto o más de un candidato, devolver `needs_review`; no asociar automáticamente.
6. **REQ-006**: Exponer `POST /api/documents/recognize` (`multipart/form-data`, campo `file`) que:
   - Valide autenticación, tamaño (máx. 10 MB), tipo PDF y firma `%PDF-`.
   - Extraiga el texto y ejecute el recognizer.
   - Devuelva una vista previa con `recognition_id`, campos extraídos, formato detectado, confianza, advertencias y candidatos de asociación.
   - **No escriba** en `debt_bills` ni en otras tablas de negocio.
7. **REQ-007**: Persistir la vista previa en una tabla nueva `document_recognitions` (ver §4.3) y guardar temporalmente el PDF en `{DATA_DIR}/documents/_inbox/{recognition_id}.pdf` hasta que se confirme o expire.
8. **REQ-008**: Exponer `POST /api/documents/recognitions/{id}/confirm` que, con los campos revisados por el usuario:
   - Mueva/copie el PDF a la ruta determinística del destino (para deudas: `{DATA_DIR}/documents/{debt_id}/receipt_{installment}.pdf`).
   - Actualice `debt_bills.payment_reference` solo para cuotas `paid` y solo si el usuario confirmó.
   - Marque la `document_recognitions` como `associated`.
   - Ejecute la actualización de DB dentro de una transacción y el archivo de forma atómica (temp + rename).
   - Si ya existe un archivo en el destino, pedir confirmación explícita antes de reemplazarlo.
9. **REQ-009**: Nunca cambiar `status` ni `paid_at` de una cuota en P0. La asociación de P0 solo guarda el comprobante y la referencia; la cuota ya debe estar pagada. Marcar pagos desde un comprobante queda para P1.
10. **REQ-010**: Si la cuota ya tiene `payment_reference` con otro valor, mostrar el conflicto en la vista previa y requerir confirmación explícita para sobrescribir.
11. **REQ-011**: UI de revisión en el detalle de deuda (acción de 3 puntos **"Reconocer comprobante"** o similar) que permita:
    - Subir/arrastrar un PDF.
    - Ver los campos extraídos y editarlos si hace falta (referencia, fecha, monto, moneda, cuota).
    - Ver confianza, advertencias y candidatos.
    - Elegir la cuota destino y confirmar.
    - Ver el estado final y los errores.
    Todos los modales deben seguir las reglas del proyecto: no cerrar al hacer click fuera, overlay bloqueante, cierre con X/Cancelar, y confirmación destructiva con `confirmo`.
12. **REQ-012**: Seguridad y privacidad:
    - No loguear texto crudo, nombres, cuentas, referencias ni montos.
    - No persistir `Nombre`, `Cuenta bancaria`, `De`, `Para` ni el texto completo.
    - Sanitizar `Detalle`/`Motivo` antes de guardarlo.
    - Los PDFs y fixtures reales nunca se commitean.
13. **REQ-013**: Tests con PDFs sintéticos (sin datos reales) que cubran ambos formatos, referencias, fechas, montos, monedas, cuota, PDFs sin texto, duplicados y conflictos.
14. **REQ-014**: No modificar el esquema de `debts`/`debt_bills`; solo insertar/actualizar datos. La única migración nueva es la tabla de reconocimientos.
15. **REQ-028** (agregado durante la validación manual): Mostrar la diferencia en **días de calendario** entre la fecha de pago y el vencimiento de la cuota, en todo el ciclo de vida:
    - **Cuota pagada**: en la tarjeta del detalle de deuda y en el listado de cuotas adelantadas del análisis, debajo de la fecha de pago, mostrar `Pagada N días antes del vencimiento`, `Pagada el día de vencimiento` o `Pagada N días después del vencimiento`.
    - **Antes de pagar**: en `DebtPayModal`, mientras el usuario elige la fecha de pago, mostrar `Estás pagando N días antes/del vencimiento`, `Estás pagando el día de vencimiento` o `Estás pagando N días después del vencimiento`.
    - **Revisión de comprobante**: en `RecognizeDocumentModal`, con la fecha detectada/editada y la cuota destino seleccionada, mostrar `El pago fue N días antes/del vencimiento`, `El pago fue el día de vencimiento` o `El pago fue N días después del vencimiento`.
    - Usar diferencia de días de calendario (no horas), singular/plural correcto, i18n `es`/`en` y colores del tema (verde para pago antes o en fecha, ámbar para pago posterior).
    - No requiere cambios de API ni de DB: `paid_at` y `due_date` ya existen en `debt_bills`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-015**: Asociar comprobantes a **facturas de servicios** y otros pagos, reutilizando el mismo recognizer y un `entity_type`/`entity_id` genérico.
2. **REQ-016**: Soportar **carga por lote** de varios PDFs con una cola de revisión.
3. **REQ-017**: Registrar el **equivalente en USD/NIO** y el tipo de cambio cuando el comprobante y la deuda usan monedas distintas; no convertir silenciosamente.
4. **REQ-018**: Historial/auditoría de reconocimientos y asociaciones (quién, cuándo, qué se cambió, confianza original).
5. **REQ-019**: Detección de duplicados por hash y por referencia, con aviso claro y opción de descartar.
6. **REQ-020**: Reintentar el reconocimiento con otra versión del recognizer o tras corregir un PDF.
7. **REQ-021**: Página project-level de documentos/reconocimientos, accesible desde el sidebar o configuración, con filtros por estado y entidad.
8. **REQ-022**: Permitir marcar un pago como pagado desde el comprobante cuando la cuota esté pendiente, con confirmación explícita y auditoría.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-023**: OCR para PDFs escaneados (sin capa de texto). Evaluar dependencias y impacto en iHost antes de implementarlo.
2. **REQ-024**: Auto-asociación cuando la confianza sea muy alta y no haya conflicto; siempre con opción de deshacer.
3. **REQ-025**: Soporte para más formatos bancarios mediante recognizers configurables/plugins.
4. **REQ-026**: Fallback con LLM local/remoto para casos no reconocidos; fuera del MVP por privacidad y recursos.
5. **REQ-027**: Exportación de reconocimientos a CSV/JSON para auditoría.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: reconocer un PDF de 10 MB en < 2 s en el iHost; sin bloquear el request más de lo necesario.
- **Memoria**: extracción de texto en streaming por páginas; límite de 10 MB por archivo; evitar cargar múltiples PDFs a la vez.
- **Almacenamiento**: los PDFs finales viven en `{DATA_DIR}/documents/...` (volumen Docker). La tabla de reconocimientos es liviana; expirar/limpiar vistas previas no confirmadas.
- **Seguridad**: endpoints autenticados, validación de PDF, límite de tamaño, sin directory listing, sin logs con PII.
- **Disponibilidad**: no agregar dependencias de red; el reconocimiento funciona offline.
- **iHost**: Go puro, sin poppler ni OCR en el MVP; build multi-arch (`amd64`, `arm/v7`, `arm64`) sin cambios.
- **UI**: inputs con tokens del tema, darkmode, modales con las reglas de SPEC-122 y `docs/project-rules.md` §4.7.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

La investigación completa está en `docs/analyzers/BANK-RECEIPTS-RESEARCH.md`. Resumen:

- **Formato A — Nota de débito**: `Comprobante` / `Nota de débito`, `Hora`/`Fecha`, `Referencia`, `Cuenta bancaria`, `Detalle` (`Lote Demo - Cuota 7`), `Moneda` (`Dólares`/`Córdobas`), `Monto`.
- **Formato B — Confirmación de transferencia**: `Número de Comprobante`, `De`, `Para`, `Fecha y Hora`, `Monto debitado`, `Tipo de Cambio`, `Monto acreditado`, `Motivo` (`Lote Demo - Cuota 40`).
- El **ID del comprobante** es `Referencia` en A y `Número de Comprobante` en B.
- El campo `Detalle` **no es confiable**: puede repetir la numeración de la cuota anterior en una transacción distinta (misma fecha o fecha siguiente, referencia distinta). No asociar solo por ese campo.
- Se observaron pagos el mismo día, diferencias de monto, comprobantes en córdobas para deudas en USD y comprobantes faltantes.
- Los PDFs tenían capa de texto; no se necesitó OCR para la muestra.

### 3.2 Estado actual del código

- `internal/analyzers/analyzer.go` define `DocumentAnalyzer` y `ExtractedBill` (monto, número de factura, año/mes, vencimiento). Es un contrato orientado a facturas de servicios.
- `internal/analyzers/all/assa_seguro_auto.go` muestra el patrón de extracción de texto con `github.com/ledongthuc/pdf` y regex.
- `internal/analyzers/registry.go` registra analyzers por institución/servicio.
- SPEC-122 agregó documentos PDF de deudas (`invoice_N.pdf`/`receipt_N.pdf`) y `payment_reference`; la asociación actual se hace por archivo y actualización manual de DB.
- `debt_bills.payment_reference` es el campo natural para el ID del comprobante.

### 3.3 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Reutilizar `DocumentAnalyzer`/`ExtractedBill` | Cero contrato nuevo | `ExtractedBill` no tiene referencia, moneda, cuota ni confianza; está atado a facturas | ❌ Rechazada |
| Nuevo recognizer project-level | Contrato rico, reutilizable, separado de instituciones | Más código y una tabla nueva | ✅ Seleccionada |
| Reconocer y asociar en un solo request | Menos pasos | No permite revisar ni corregir; riesgo de asociar mal | ❌ Rechazada |
| **Preview + confirmación** | Control del usuario, auditable, seguro ante ambigüedad | Dos requests y persistencia temporal | ✅ Seleccionada |
| OCR en el MVP | Cubre PDFs escaneados | Dependencia pesada, lenta en iHost, más superficie | ❌ Rechazada (P2) |
| LLM para extracción | Tolera layouts variables | Privacidad, latencia, costo, dependencia externa | ❌ Rechazada (P2) |
| Guardar PDFs en la DB | Un solo backup | Infla SQLite/WAL; malo para iHost | ❌ Rechazada |
| **Guardar PDFs en el volumen** | Igual que SPEC-122, bajo impacto | Hay que respaldar el volumen | ✅ Seleccionada |

### 3.4 Decisiones arquitectónicas (ADRs)

**ADR-001: Recognizer project-level, separado de analyzers**
- **Contexto**: El reconocimiento de comprobantes no pertenece a una institución ni a un servicio; debe reutilizarse en deudas, servicios y otros módulos.
- **Decisión**: Nuevo paquete `internal/recognizers` (nombre a confirmar en implementación) con `ReceiptRecognizer`, `RecognizedDocument` y un registry propio.
- **Consecuencias**: No se toca el registry de analyzers de facturas; se evita acoplar el reconocimiento a `institution_id`.

**ADR-002: Preview y confirmación, sin auto-asociación**
- **Contexto**: El `Detalle` puede estar mal; una asociación automática puede pisar datos correctos.
- **Decisión**: El reconocimiento devuelve una vista previa; la asociación requiere confirmación del usuario. La confianza alta puede sugerir, pero no escribir sola.
- **Consecuencias**: Más pasos de UI, pero auditable y seguro. P2 puede habilitar auto-asociación con deshacer.

**ADR-003: Tabla `document_recognitions` y PDF temporal en `_inbox`**
- **Contexto**: La vista previa necesita persistir el PDF y los campos extraídos entre los dos requests.
- **Decisión**: Guardar el PDF en `{DATA_DIR}/documents/_inbox/{recognition_id}.pdf` y una fila en `document_recognitions` con estado `recognized`. Al confirmar, mover a la ruta final y marcar `associated`.
- **Consecuencias**: Se necesita limpieza de reconocimientos no confirmados; la tabla es liviana y no modifica `debt_bills`.

**ADR-004: Matching por referencia + fecha/monto + secuencia, con warnings**
- **Contexto**: El ID del comprobante es la señal más fuerte, pero no siempre se conoce el pago destino; el `Detalle` puede repetir la cuota anterior.
- **Decisión**: Motor de scoring con señales ponderadas; conflictos → `needs_review`; nunca sobrescribir `payment_reference` sin confirmación.
- **Consecuencias**: La UI debe mostrar candidatos y advertencias; el usuario decide.

**ADR-005: Sin OCR ni LLM en el MVP**
- **Contexto**: iHost tiene recursos limitados y los PDFs de la muestra tienen texto.
- **Decisión**: Extracción de texto con `github.com/ledongthuc/pdf`. Si no hay texto → `needs_ocr`/`unsupported`.
- **Consecuencias**: PDFs escaneados no se reconocen en P0; queda documentado para P2.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────┐
│ Frontend (React)            │
│  Modal "Reconocer comprobante"
│   ├─ PdfUploadField         │
│   ├─ Preview editable       │
│   └─ Confirmar asociación   │
└──────────────┬──────────────┘
               │ multipart PDF
               ▼
┌─────────────────────────────┐
│ Go API                      │
│  POST /api/documents/recognize
│  POST /api/documents/recognitions/{id}/confirm
└──────────────┬──────────────┘
               │ DocumentRecognitionService
               ▼
┌─────────────────────────────┐   ┌──────────────────────────────┐
│ SQLite                      │   │ DATA_DIR/documents/          │
│  document_recognitions      │   │  ├─ _inbox/{id}.pdf          │
│  debt_bills.payment_reference│  │  └─ {debt_id}/receipt_N.pdf   │
└─────────────────────────────┘   └──────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 Backend — `internal/recognizers`

- **Responsabilidad**: detectar formato, extraer texto y campos, devolver un `RecognizedDocument`.
- **Interfaz propuesta**:
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
- **Registry**: lista de recognizers; se prueban en orden hasta que uno detecte el formato.
- **Ubicación**: `internal/recognizers/` (o `internal/documents/` si la implementación prefiere un nombre más amplio).

#### 4.2.2 Backend — `BankReceiptRecognizer`

- **Responsabilidad**: implementar Formato A y Formato B.
- **Detección**:
  - `Nota de débito` + `Referencia` → Formato A.
  - `Confirmación de Transferencia` + `Número de Comprobante` → Formato B.
- **Extracción**:
  - `Referencia\s+(\d+)`
  - `N[uú]mero de Comprobante\s+(\d+)`
  - `Cuota\s+(\d+)`
  - `Detalle\s+(.+)` / `Motivo\s+(.+)`
  - `Fecha:\s*(\d{2}/\d{2}/\d{4})`, `Hora(\d{2}:\d{2}:\d{2})`, `Fecha y Hora\s+(\d{2}/\d{2}/\d{4}) - (\d{2}:\d{2})`
  - `Moneda\s+(Dólares|Córdobas)`, `Monto\s+([\d,]+\.\d{2})`, `Monto debitado\s+\$?\s*([\d,]+\.\d{2})`
- **Normalización**: `USD`/`NIO`, montos con coma de miles, referencias como texto, fechas `dd/mm/yyyy`.
- **Warnings**: `detalle_repetido`, `moneda_distinta`, `monto_distinto`, `sin_numero_cuota`, `referencia_duplicada`.
- **Ubicación**: `internal/recognizers/bank_receipt.go` (nombre tentativo).

#### 4.2.3 Backend — `DocumentRecognitionService`

- **Responsabilidad**: orquestar validación de archivo, hash, extracción, matching, persistencia de la vista previa y confirmación.
- **Métodos**:
  - `Recognize(ctx, file, filename) (*RecognitionPreview, error)`
  - `Confirm(ctx, recognitionID, input ConfirmInput) (*AssociationResult, error)`
  - `List(ctx, filters) ([]Recognition, error)` (P1)
- **Matching**:
  - Buscar candidatos en `debt_bills` por `payment_reference` (P0) y por monto/fecha.
  - Para servicios, en P1.
  - Devolver `candidates` con score, installment, due date, status.
- **Persistencia**: `storage.DocumentRecognitionStorage` para la tabla nueva; `os`/`filepath` para `_inbox` y destino final.
- **Ubicación**: `internal/services/document_recognition.go`.

#### 4.2.4 Backend — handlers y rutas

- `POST /api/documents/recognize` → `DocumentHandlers.Recognize` (multipart `file`).
- `POST /api/documents/recognitions/{id}/confirm` → `DocumentHandlers.ConfirmRecognition`.
- `GET /api/documents/recognitions` → historial (P1).
- `GET /api/documents/recognitions/{id}` → detalle (P1).
- Validaciones: auth, `MaxBytesReader` 10 MB, `%PDF-`, `application/pdf`, `recognition_id` numérico.
- **Ubicación**: `internal/api/document_recognition_handlers.go`, rutas en `internal/api/routes.go`.

#### 4.2.5 Frontend — modal de reconocimiento

- **Componente**: `frontend/src/components/RecognizeDocumentModal.tsx`.
- **Flujo**:
  1. Upload con `PdfUploadField`.
  2. Llamada a `POST /api/documents/recognize`.
  3. Preview editable: referencia, fecha, monto, moneda, cuota detectada, confianza, warnings.
  4. Selector de cuota destino (para deuda) y botón "Asociar".
  5. Confirmación destructiva si se sobrescribe `payment_reference` o un archivo existente.
  6. Resultado y refresco del detalle.
- **Integración P0**: desde el detalle de deuda (`DebtBillsPage`), opción **"Reconocer comprobante"** en el menú de 3 puntos. El modal se abre con la deuda precargada.
- **Reglas UI**: tokens del tema, darkmode, modal sin cierre por click fuera, overlay bloqueante, X/Cancelar, confirmación con `confirmo`.
- **API client**: `api.documents.recognize(file)`, `api.documents.confirm(id, body)`.
- **Tipos**: `RecognizedDocument`, `RecognitionPreview`, `RecognitionCandidate`, `ConfirmRecognitionInput`.

#### 4.2.6 Frontend — página project-level (P1)

- Página `/documents` o sección "Documentos" en el sidebar/configuración con:
  - Upload genérico.
  - Lista de reconocimientos por estado.
  - Filtros por entidad, confianza y fecha.
  - Acciones ver/reasociar/descartar.
- En P0 no es obligatoria; alcanza con el modal desde deuda.

### 4.3 Modelo de datos

Nueva tabla (migración siguiente a la última existente):

```
Entidad: document_recognitions
- id: INTEGER PK
- document_hash: TEXT NOT NULL UNIQUE
- recognizer_id: TEXT NOT NULL
- document_type: TEXT NOT NULL
- status: TEXT NOT NULL ('recognized' | 'associated' | 'rejected' | 'expired')
- reference: TEXT
- paid_at: DATETIME
- amount: REAL
- currency: TEXT
- amount_usd_equivalent: REAL NULL
- exchange_rate: REAL NULL
- installment_number: INTEGER NULL
- concept_text: TEXT NULL (sanitizado)
- entity_type: TEXT NULL ('debt_bill' | 'service_bill' | 'other')
- entity_id: INTEGER NULL
- confidence: REAL NOT NULL DEFAULT 0
- warnings: TEXT NULL (JSON array)
- raw_text_hash: TEXT NULL
- inbox_path: TEXT NULL
- final_path: TEXT NULL
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- confirmed_at: DATETIME NULL
```

Relaciones:
- `entity_type='debt_bill'` + `entity_id=<debt_bill.id>` (o `debt_id` + `installment_number`; la spec de implementación debe fijar una sola convención).
- No se modifica el esquema de `debt_bills`; el confirm actualiza `payment_reference` y `updated_at`.

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/documents/recognize`

**Request**: `multipart/form-data`
```
file: <PDF>
```

**Response 200**:
```json
{
  "recognition_id": 1,
  "document_type": "nota_debito",
  "reference": "902000001",
  "paid_at": "2026-02-01T09:15:22-06:00",
  "amount": 123.45,
  "currency": "USD",
  "installment_number": 7,
  "concept": "Lote Demo - Cuota 7",
  "confidence": 0.92,
  "warnings": [],
  "candidates": [
    {
      "entity_type": "debt_bill",
      "debt_id": 1,
      "installment_number": 7,
      "status": "paid",
      "score": 0.95,
      "reasons": ["referencia", "monto", "cuota"]
    }
  ]
}
```

**Errores**: `400 invalid_file`, `400 unsupported_format`, `400 needs_ocr`, `401 unauthorized`, `413 request_too_large`, `500 internal_error`.

#### Endpoint: `POST /api/documents/recognitions/{id}/confirm`

**Request**:
```json
{
  "entity_type": "debt_bill",
  "debt_id": 1,
  "installment_number": 7,
  "reference": "902000001",
  "paid_at": "2026-02-01T09:15:22-06:00",
  "amount": 123.45,
  "currency": "USD",
  "overwrite_reference": false,
  "overwrite_file": false
}
```

**Response 200**:
```json
{
  "recognition_id": 1,
  "status": "associated",
  "payment_reference": "902000001",
  "file_path": "documents/1/receipt_7.pdf"
}
```

**Errores**: `400 invalid_target`, `400 reference_conflict`, `400 file_exists`, `404 not_found`, `409 already_associated`, `500 internal_error`.

### 4.5 Dependencias

- **Internas**: `internal/recognizers`, `internal/services/document_recognition.go`, `internal/storage/document_recognition.go`, `internal/api/document_recognition_handlers.go`, `internal/models/document_recognition.go`, migración nueva, frontend (`RecognizeDocumentModal`, `DebtBillsPage`, API client, tipos, i18n).
- **Externas**: `github.com/ledongthuc/pdf` (ya usado por los analyzers). Ninguna dependencia nueva de OCR/LLM.

### 4.6 Seguridad y privacidad

- No persistir ni loguear `Nombre`, `Cuenta bancaria`, `De`, `Para` ni texto completo.
- `concept_text` se guarda sanitizado y acotado (por ejemplo, máximo 200 caracteres).
- `raw_text_hash` solo si se necesita auditar; nunca el texto crudo.
- Validar `%PDF-`, `Content-Type`, tamaño y autenticación.
- Los PDFs viven en el volumen Docker; nunca en el repo.
- Tests y fixtures con PDFs sintéticos; nunca datos reales.

### 4.7 Manejo de errores y estados

| Situación | Comportamiento |
|-----------|----------------|
| PDF no válido o > 10 MB | `400 invalid_file`; no se crea registro. |
| Formato desconocido | `400 unsupported_format`; se puede guardar solo para revisión manual (P1). |
| PDF sin texto | `400 needs_ocr`; no se inventan campos. |
| Referencia faltante | `needs_review`; no se asocia. |
| Referencia ya usada en otra cuota | `409 conflict`; no sobrescribir sin confirmación. |
| `payment_reference` distinta en la cuota destino | `400 reference_conflict`; la UI debe pedir confirmación. |
| Archivo final ya existe | `400 file_exists`; la UI debe pedir confirmación para reemplazar. |
| Confirmación exitosa | Archivo atómico + transacción de DB; estado `associated`. |
| Reconocimiento no confirmado | Queda en `_inbox`; limpieza por expiración (P1/P2). |

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Un PDF del Formato A (nota de débito) se detecta correctamente y devuelve `document_type`, `reference`, `paid_at`, `amount`, `currency` y `installment_number` cuando están presentes.
- [ ] **CA-002**: Un PDF del Formato B (confirmación de transferencia) se detecta correctamente y devuelve `Número de Comprobante`, fecha/hora, monto debitado y `Motivo`.
- [ ] **CA-003**: Un PDF que no coincide con ningún formato devuelve `unsupported_format` y no crea asociaciones.
- [ ] **CA-004**: La referencia se guarda como texto y no se confunde con nombres, cuentas ni otros números del documento.
- [ ] **CA-005**: El motor de matching devuelve candidatos con score, razones y warnings; si el `Detalle` repite la cuota anterior, el resultado queda en `needs_review`.
- [ ] **CA-006**: `POST /api/documents/recognize` no modifica `debt_bills` ni otras tablas de negocio; solo crea el registro de reconocimiento y el PDF temporal.
- [ ] **CA-007**: La UI permite subir el PDF, ver/editar los campos, elegir la cuota destino y confirmar; el modal no se cierra por click fuera y respeta las reglas de modales del proyecto.
- [ ] **CA-008**: Al confirmar, el PDF se guarda en `{DATA_DIR}/documents/{debt_id}/receipt_{n}.pdf` de forma atómica y `debt_bills.payment_reference` queda con la referencia revisada.
- [ ] **CA-009**: Si la cuota destino ya tiene otra `payment_reference`, la UI muestra el conflicto; solo sobrescribe con confirmación explícita.
- [ ] **CA-010**: Si el archivo final ya existe, la UI pide confirmación antes de reemplazarlo; si el usuario cancela, no se toca el archivo.
- [ ] **CA-011**: El hash SHA-256 evita procesar/asociar dos veces el mismo PDF; una referencia duplicada se marca como conflicto.
- [ ] **CA-012**: El reconocimiento no cambia `status` ni `paid_at` de la cuota; solo asocia documentos/referencia en P0.
- [ ] **CA-013**: Un PDF sin capa de texto devuelve `needs_ocr` y no genera una asociación falsa.
- [ ] **CA-014**: Los tests usan PDFs sintéticos y no incluyen nombres, cuentas, referencias ni montos reales.
- [ ] **CA-015**: La tabla `document_recognitions` se crea con una migración nueva; `debt_bills` y `debts` no cambian de esquema.
- [ ] **CA-016**: La cuota pagada muestra la diferencia en días entre `paid_at` y `due_date`; el modal de pago la muestra mientras se elige la fecha; el modal de reconocimiento la muestra para la cuota destino. Los casos mismo día, 1 día y varios días usan singular/plural correcto en `es`/`en`.
- [ ] **CA-DARK**: El modal de reconocimiento usa tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó en darkmode.
- [ ] **CA-MODAL**: El modal no se cierra al hacer click fuera, el overlay bloquea el fondo, el cierre es con X/Cancelar y las acciones destructivas usan confirmación con `confirmo`.
- [ ] **CA-BUILD**: `go test ./...`, `go vet ./...` y `npm run build` (con `check-ui-rules.cjs`) pasan.

### 5.2 No funcionales

- [ ] **CA-NF-001**: Reconocer un PDF de 10 MB no supera ~2 MB de RAM adicional y responde en < 2 s en el iHost.
- [ ] **CA-NF-002**: No se agregan dependencias de OCR/LLM ni llamadas de red externas.
- [ ] **CA-NF-003**: Los logs no contienen PII ni texto crudo de los PDFs.
- [ ] **CA-NF-004**: El build multi-arch (`amd64`, `arm/v7`, `arm64`) sigue funcionando sin cambios de plataforma.

### 5.3 Testing

- **Unit tests**:
  - `BankReceiptRecognizer` con PDFs sintéticos de ambos formatos.
  - Extracción de referencia, fecha, monto, moneda, cuota y concepto.
  - Casos de `Detalle` repetido, mismo día, moneda distinta, PDF sin texto y campos faltantes.
  - Normalización de montos/fechas y referencias como texto.
- **Integration tests**:
  - `POST /recognize` con PDF sintético → preview con candidatos y warnings.
  - `POST /confirm` → archivo final + `payment_reference` en una copia de DB.
  - Conflictos: referencia distinta, archivo existente, cuota no pagada, duplicado por hash.
  - No modificar `status`/`paid_at`.
- **Frontend**:
  - `npm run check:ui`, `npm run build`.
  - Unit tests con Node nativo: `node --test frontend/tests/*.test.ts`, incluyendo `paymentTiming` (mismo día, 1 día antes/después, varios días, fechas con hora/zona y valores inválidos).
  - Prueba manual: subir ambos formatos, revisar/editar, confirmar, ver el comprobante en el detalle y la referencia en la DB; verificar la diferencia pago/vencimiento en la tarjeta pagada, en el modal de pago y en el de reconocimiento.
- **iHost**:
  - Probar con PDFs sintéticos, verificar memoria/tiempo y que el archivo quede en `/app/data/documents/...`.
  - Nunca usar PDFs reales en el repo.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modelo `document_recognitions`, storage y migración | 0.5 día | Ninguna |
| 2 | `internal/recognizers`: interfaz, registry y `BankReceiptRecognizer` + tests sintéticos | 1 día | Fase 1 |
| 3 | `DocumentRecognitionService`: matching, preview, confirmación atómica y tests | 1 día | Fase 2 |
| 4 | API handlers/rutas y validaciones | 0.5 día | Fase 3 |
| 5 | Frontend: modal de reconocimiento, preview editable, confirmación, i18n | 1 día | Fase 4 |
| 6 | Integración en detalle de deuda, auditoría básica y pruebas manuales | 0.5 día | Fase 5 |
| 7 | Validación en iHost y documentación | 0.5 día | Fase 6 |

**Total estimado**: ~5 días de desarrollo + pruebas.

### 6.2 Milestones

1. **MVP**: reconocer Formatos A/B, preview, confirmar asociación a una cuota de deuda, guardar PDF y `payment_reference`, tests sintéticos.
2. **V1.0**: historial/auditoría, duplicados, NIO/USD, página project-level y asociación a servicios.
3. **P2**: OCR, auto-asociación de alta confianza, más formatos.

---

### 6.3 Estado de implementación (2026-11-08)

Implementado en la rama `feature/SPEC-127`:

- **Migración 0045**: tabla `document_recognitions` con hash único, estados, referencia, fecha, monto, moneda, cuota, concepto sanitizado, entidad destino, confianza, advertencias, rutas de inbox/final y timestamps. No se modificó el esquema de `debts`/`debt_bills`.
- **Backend**:
  - `internal/recognizers/`: interfaz `ReceiptRecognizer`, registry project-level y `BankReceiptRecognizer` para nota de débito y confirmación de transferencia.
  - `internal/models/document_recognition.go`: modelos de reconocimiento, candidatos, confirmación y resultado.
  - `internal/storage/document_recognition.go`: CRUD, idempotencia por hash y asociación transaccional con `debt_bills.payment_reference`.
  - `internal/services/document_recognition.go`: validación de PDF, staging en `_inbox`, SHA-256, matching con score/warnings, confirmación atómica y reemplazo con backup.
  - `internal/api/document_recognition_handlers.go` + rutas:
    - `POST /api/documents/recognize`
    - `POST /api/documents/recognitions/{id}/confirm`
- **Frontend**:
  - `RecognizeDocumentModal.tsx`: upload, preview editable, confianza, advertencias, candidatos, selección de cuota y confirmación.
  - `ConfirmWordModal.tsx`: confirmación por palabra `confirmo` con foco automático y Enter (SPEC-126).
  - Acción **"Reconocer comprobante"** en el menú de 3 puntos del detalle de deuda.
  - API client, tipos e i18n `documents.*` en `frontend/public/i18n/{es,en}.json`.
  - **REQ-028**: helper `frontend/src/utils/paymentTiming.ts` y componente `PaymentTimingLabel` integrados en la tarjeta pagada, el listado de cuotas adelantadas de `DebtAnalysis`, `DebtPayModal` y `RecognizeDocumentModal`; i18n `deudas.timing_*`.

Validaciones ejecutadas:

- `go test ./...`: OK.
- `go vet ./...`: OK.
- `npm run build` (incluye `check:ui-rules.cjs`): OK.
- Migración 0045 probada sobre una copia de la DB de producción (`/tmp`, eliminada al terminar): deudas 12 → 12, cuotas 702 → 702, `PRAGMA integrity_check = ok`.
- Tests con PDFs sintéticos para ambos formatos, matching, `Detalle` repetido, duplicados, conflictos, PDF sin texto y PDF inválido.

Pendiente: prueba manual del usuario en local y release formal (merge a `main`, tracker, label e issue).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El banco cambia el layout del PDF | Media | Alto | Anclas + regex versionados; tests con fixtures sintéticos; fallback `unsupported_format`. |
| `Detalle` repetido o incorrecto | Alta | Alto | Nunca asociar solo por `Detalle`; scoring + `needs_review` + confirmación. |
| Asociación errónea por monto/fecha | Media | Alto | Referencia como señal principal; preview editable; no auto-asociación en P0. |
| PII en logs o DB | Media | Alto | No loguear texto crudo; sanitizar `concept`; no guardar nombre/cuenta. |
| PDFs escaneados sin texto | Media | Medio | `needs_ocr`; OCR en P2 con evaluación de dependencias. |
| Memoria/CPU en iHost | Media | Medio | Límite 10 MB, extracción por páginas, sin OCR/LLM, medir en iHost. |
| Duplicados y referencias repetidas | Media | Medio | Hash único, detección de referencia duplicada, estado `conflict`. |
| Archivos temporales en `_inbox` sin limpiar | Media | Bajo | Expiración/limpieza programada; no acumular PDFs. |
| Cambios de esquema en `debt_bills` | Baja | Alto | Solo `UPDATE payment_reference`; migración únicamente para la tabla nueva. |
| Reemplazo accidental de un comprobante | Baja | Medio | Confirmación explícita + backup/rename atómico. |

---

## 8. Notas y Referencias

- **Investigación completa**: `docs/analyzers/BANK-RECEIPTS-RESEARCH.md` (formatos, regex, casos ambiguos, modelo propuesto).
- **SPEC-122**: facturas y comprobantes PDF en pagos de deudas; rutas `documents/{debt_id}/receipt_N.pdf` y flags `has_receipt`.
- **SPEC-054/104**: modelo de deudas y cuotas; `payment_reference` en `debt_bills`.
- **Analyzers actuales**: `internal/analyzers/analyzer.go`, `internal/analyzers/registry.go`, `internal/analyzers/all/assa_seguro_auto.go`.
- **PDF en Go**: `github.com/ledongthuc/pdf` (ya usado por los analyzers; sin poppler en runtime).
- **iHost**: `DATA_DIR=/app/data`, volumen `p40la`; reglas de `docs/ssh-ihost-access.md`.
- **UI**: `docs/project-rules.md` §4.3 (inputs/darkmode), §4.7 (modales), `frontend/src/components/PdfUploadField.tsx` (patrón de upload).
- **Datos sintéticos**: todos los ejemplos usan `Cliente Demo`, `Banco Demo`, `Lote Demo`, `Cuota 7`, `902000001`, `52700001`, `123.45`. Nunca usar datos reales en tests, specs ni commits.

### Fuera de alcance (MVP)

- OCR de PDFs escaneados.
- Auto-asociación sin confirmación.
- LLM para extracción.
- Conciliación bancaria automática completa.
- Modificación de estados de pago desde el comprobante (P1).

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-08 | opencode | Creación inicial de la especificación a partir de la investigación en `docs/analyzers/BANK-RECEIPTS-RESEARCH.md`. |
| 2026-11-08 | opencode | Estado `in_progress`: comienza la implementación aprobada por el usuario. |
| 2026-11-08 | opencode | Implementación completa backend/frontend, tests automáticos y validación de la migración sobre copia de producción. Pendiente prueba manual y release. |
| 2026-11-08 | opencode | Se agrega REQ-028 (diferencia pago vs vencimiento) a partir de la validación del usuario: tarjeta pagada, modal de pago y modal de reconocimiento. |
