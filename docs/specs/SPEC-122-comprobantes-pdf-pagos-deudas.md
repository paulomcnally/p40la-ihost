---
title: "Facturas y comprobantes PDF en pagos de deudas: upload, almacenamiento en volumen y visualización"
id: "SPEC-122"
status: "in_progress"
author: "opencode"
created: "2026-11-07"
updated: "2026-11-07"
github_issue: 125
---

# Facturas y comprobantes PDF en pagos de deudas: upload, almacenamiento en volumen y visualización

**ID**: SPEC-122
**Estado**: in_progress
**Autor**: opencode
**Creado**: 2026-11-07
**Actualizado**: 2026-11-07

---

## 1. Resumen Ejecutivo

Hoy el pago de una cuota de deuda solo registra la fecha de pago y una referencia de texto (`DebtPayModal` → `PUT /api/debt-bills/{id}/pay`). No existe forma de adjuntar los documentos del pago, por lo que el usuario pierde la evidencia documental y debe buscarla fuera de la app.

Esta spec agrega **dos documentos PDF independientes por pago**:

1. **Factura** (`invoice`): el documento que emite o entrega quien brinda el préstamo o la deuda.
2. **Comprobante** (`receipt`): el comprobante que genera el usuario al pagar (transferencia, depósito, etc.).

Ambos se pueden adjuntar al pagar mediante botones de selección y áreas de **drag & drop**, y también se pueden adjuntar o reemplazar después del pago. Se guardan en el volumen Docker, junto a la base SQLite, con rutas determinísticas basadas en los IDs de la deuda y el número de cuota:

```
{DATA_DIR}/documents/{debt_id}/invoice_{installment_number}.pdf
{DATA_DIR}/documents/{debt_id}/receipt_{installment_number}.pdf
```

Ejemplo sintético: deuda `id=1`, cuota `installment_number=1` → `documents/1/invoice_1.pdf` y `documents/1/receipt_1.pdf`.

Toda cuota que tenga factura y/o comprobante muestra **iconos separados** en las vistas donde aparece (detalle de deuda, calendario y análisis). El menú de 3 puntos incluye las opciones **"Factura"** y **"Comprobante"** para abrir cada archivo en una pestaña nueva.

Consideraciones iHost: no se agregan dependencias nuevas (solo `net/http`, `mime/multipart`, `os`, `path/filepath`, `io`, `bytes` de la stdlib de Go), no se modifica el esquema de `debt_bills` (la existencia de cada documento se deduce del filesystem) y no se crea un volumen nuevo: `documents/` vive dentro de `DATA_DIR` (`/app/data` en el contenedor), que ya está montado en el volumen `p40la`. Cada archivo se sube en streaming a disco con un límite de 10 MB para no inflar la RAM del iHost.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En el modal de pago de una cuota (`DebtPayModal`) agregar **dos áreas de upload independientes**:
   - **Factura**: arrastrar y soltar un PDF o seleccionarlo con un botón.
   - **Comprobante**: arrastrar y soltar un PDF o seleccionarlo con un botón.
   - Cada área muestra el nombre, tamaño y un botón para quitar el archivo antes de confirmar.
   - Ambos archivos son **opcionales**: se puede pagar sin documentos, solo con factura, solo con comprobante o con ambos.
2. **REQ-002**: Validar cada archivo en frontend y backend:
   - Solo formato **PDF** (extensión `.pdf` y contenido que empiece con la firma `%PDF-`).
   - Tamaño máximo **10 MB por archivo** (consistente con los comprobantes de pensión, SPEC-051).
   - Rechazar archivos vacíos, corruptos o de otro tipo con un mensaje claro e indicando si es la factura o el comprobante.
3. **REQ-003**: Guardar los PDFs en el volumen Docker con rutas determinísticas:
   ```
   {DATA_DIR}/documents/{debt_id}/invoice_{installment_number}.pdf
   {DATA_DIR}/documents/{debt_id}/receipt_{installment_number}.pdf
   ```
   - `DATA_DIR` es el mismo directorio donde vive `app.db` (`/app/data` en el contenedor, volumen Docker `p40la`).
   - Crear el directorio `documents/{debt_id}` con `os.MkdirAll(..., 0o755)` si no existe.
   - No crear el directorio al arrancar el server: solo en el primer upload.
4. **REQ-004**: Extender `PUT /api/debt-bills/{id}/pay` para aceptar `multipart/form-data` con los campos:
   - `paid_at` (obligatorio).
   - `payment_reference` (opcional).
   - `invoice` (opcional, PDF).
   - `receipt` (opcional, PDF).
   Mantener compatibilidad con el body JSON actual (`application/json`) para no romper clientes existentes ni tests.
5. **REQ-005**: Agregar endpoints autenticados:
   - `GET /api/debt-bills/{id}/invoice`
   - `GET /api/debt-bills/{id}/receipt`
   - `DELETE /api/debt-bills/{id}/invoice`
   - `DELETE /api/debt-bills/{id}/receipt`
   Los `GET` devuelven el PDF con `Content-Type: application/pdf` y `Content-Disposition: inline; filename="<kind>_{n}.pdf"`. Si la cuota no existe, está eliminada o no tiene ese archivo → `404`. Con `?download=1` el `Content-Disposition` pasa a `attachment`. Los `DELETE` remueven solo el archivo indicado y devuelven la cuota con los flags actualizados.
6. **REQ-006**: Exponer en las respuestas de cuotas los campos booleanos `has_invoice` y `has_receipt`:
   - `GET /api/debts/{id}/bills` (detalle de deuda).
   - `GET /api/debt-bills?year=&month=` (calendario).
   - Respuesta de `PUT /api/debt-bills/{id}/pay`.
   Los campos se calculan verificando la existencia de cada archivo en disco (`os.Stat`), sin agregar columnas ni migraciones. En `GET /api/debts` y `GET /api/debts/{id}` exponer `has_documents` si la deuda tiene al menos un archivo.
7. **REQ-007**: Mostrar **iconos separados** clickeables en toda cuota que tenga documentos:
   - Detalle de deuda (`DebtBillsPage`): icono `bill` para factura e icono `pdf` para comprobante.
   - Calendario (`DebtCalendar`): ambos iconos en la grilla desktop y en la agenda móvil.
   - Análisis (`DebtAnalysis`): ambos iconos en resultados de búsqueda y en cuotas pagadas por adelantado.
   - Tarjeta de deuda (`DeudasPage`): indicador genérico si `has_documents=true`.
   Al hacer clic, abrir el PDF correspondiente en una pestaña nueva (`window.open(url, '_blank', 'noopener,noreferrer')`).
8. **REQ-008**: En el menú de 3 puntos (`CardMenu`) del detalle de deuda, para cuotas pagadas mostrar **únicamente**:
   - **"Factura"** (`deudas.invoice`).
   - **"Comprobante"** (`deudas.receipt`).
   Cada opción abre el modal de documento correspondiente. No usar textos "Adjuntar", "Reemplazar", "Ver factura" ni "Ver comprobante" en el menú. La opción **"Pagar"** se mantiene para cuotas pendientes.
9. **REQ-009**: Agregar las claves i18n en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` para: factura, comprobante, textos de drag & drop, formatos, tamaño máximo, errores, nombres de archivo, acciones Abrir/Subir/Remover y confirmaciones. La fuente de verdad del i18n es `frontend/public/i18n/`, nunca `public/i18n/`.
10. **REQ-010**: Documentar que `documents/` forma parte del volumen Docker y debe incluirse en los respaldos junto con `app.db`. El respaldo del solo archivo SQLite no preserva los documentos. No crear un volumen adicional ni cambiar el `docker-compose.yml`.
11. **REQ-011**: El guardado de cada archivo debe ser **atómico**: escribir primero en un archivo temporal dentro del mismo directorio y luego `os.Rename` al nombre final. Si el pago falla, eliminar todos los temporales. Si se sube una factura o un comprobante nuevo para una cuota que ya tenía ese documento, reemplazar solo ese archivo (última versión gana) sin dejar temporales ni duplicados.
12. **REQ-012**: Nunca usar el nombre de archivo provisto por el usuario para construir la ruta. Los nombres finales se derivan exclusivamente de `debt_id`, `installment_number` y el tipo de documento (`invoice`/`receipt`), evitando path traversal.
13. **REQ-020**: Al tocar "Factura" o "Comprobante" en el menú de 3 puntos se abre un **modal de documento**:
    - Si el documento no existe: mostrar un área de upload (`PdfUploadField`) y un botón **"Subir"**.
    - Si el documento existe: mostrar el nombre `invoice_{n}.pdf`/`receipt_{n}.pdf`, un botón **"Abrir"** y un botón **"Remover"**.
    - Después de subir un documento, el modal **permanece abierto** y pasa al estado "Abrir"/"Remover" sin cerrarse.
    - El modal **no se cierra al hacer click fuera**; solo se cierra con la **X** o con los botones **Cancelar/Cerrar**.
    - **"Remover"** abre un segundo modal de confirmación (reutilizar `DeleteModal`) que exige escribir **"confirmo"** para habilitar la acción. Al confirmar, se elimina solo ese archivo y se actualizan los flags.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: Los endpoints `POST /api/debt-bills/{id}/invoice` y `POST /api/debt-bills/{id}/receipt` (`multipart/form-data`, campo `file`) permiten subir el documento desde el modal cuando no existe. Solo para cuotas en estado `paid`. El menú de 3 puntos **no** muestra opciones de adjuntar ni reemplazar; el flujo de reemplazo es remover y volver a subir desde el modal.
2. **REQ-014**: Mostrar en la tarjeta de deuda (`DeudasPage`) un indicador discreto si la deuda tiene al menos un documento (`has_documents`). No debe degradar el listado (máximo un `os.ReadDir` por deuda).
3. **REQ-015**: Si el navegador no puede previsualizar el PDF inline (caso Safari/iOS), ofrecer un enlace de descarga alternativo (`?download=1`) en ambos endpoints.
4. **REQ-019**: En el detalle de deuda, la fila de cada cuota debe reservar espacio a la derecha (`pr-14`) para que el menú de 3 puntos no se superponga con el badge de estado ni con el monto. El menú debe quedar visible y clickeable en móvil y desktop.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Limpieza de archivos huérfanos cuando una cuota se elimina lógicamente por reducción de `installments_total` o cuando una deuda se elimina definitivamente. No implementar en el MVP; documentar como deuda técnica.
2. **REQ-017**: Soporte de múltiples facturas o comprobantes por cuota, o de imágenes (JPG/PNG). Fuera de alcance: el requerimiento pide un PDF de cada tipo por pago.
3. **REQ-018**: Registro de auditoría del upload/reemplazo de documentos (quién y cuándo), reutilizando el patrón de historial de facturas (SPEC-070).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el upload de dos PDFs de 10 MB debe completarse en < 8 s en la red local del iHost. El cálculo de `has_invoice`/`has_receipt` es O(N) con `os.Stat` por cuota; N está acotado por la cantidad de cuotas de una deuda o de un mes (< 100 típicamente).
- **Memoria**: cada archivo se copia en streaming desde `multipart.File` al disco; no se carga completo en memoria. `r.ParseMultipartForm(1 << 20)` limita a 1 MB en RAM y el resto va a archivos temporales.
- **Almacenamiento**: máximo 10 MB por documento y hasta dos documentos por cuota (20 MB por pago). El volumen del iHost debe tener espacio suficiente; no se agrega rotación automática en esta spec.
- **Seguridad**:
  - Endpoints protegidos con `authMiddleware`, igual que el resto del API.
  - Validación de firma `%PDF-` además de extensión.
  - `X-Content-Type-Options: nosniff` en las descargas.
  - Rutas construidas solo con enteros de la DB y un enum de tipo; sin directory listing ni servido estático de `documents/`.
  - Límite de tamaño aplicado antes de escribir el archivo final (`http.MaxBytesReader`).
- **Disponibilidad**: sin cambios en `/health`. Si `DATA_DIR/documents` no existe, el server arranca normalmente y lo crea en el primer upload.
- **iHost**: cero dependencias nuevas, cero migraciones, cero procesos en background. Build multi-arch (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) sin cambios.

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Flujo actual de pago de deudas**:
  - Modelo `internal/models/debt_bill.go`: `DebtBill` con `paid_at`, `payment_reference`, `status` (`pending`/`paid`).
  - Servicio `internal/services/debt.go`: `PayBill` valida existencia/estado y llama a `DebtBillStorage.Pay`.
  - Storage `internal/storage/debt_bill.go`: `Pay` actualiza `status`, `paid_at`, `payment_reference`.
  - Handler `internal/api/debt_handlers.go`: `PayDebtBill` decodifica JSON y parsea `paid_at`.
  - Ruta `internal/api/routes.go`: `PUT /api/debt-bills/{id}/pay`.
  - Frontend `frontend/src/components/DebtPayModal.tsx`: fecha + referencia; `api.debts.payBill` envía JSON.
- **Precedente de comprobantes en pensión (SPEC-051)**:
  - `internal/services/support_record.go`: `SaveProof` valida 10 MB, extensiones PDF/PNG/JPG/WEBP y escribe en `{DATA_DIR}/uploads/payment-proofs/{year}/{month}/{id}.ext`.
  - `ProofPath` devuelve ruta y nombre para descarga.
  - `internal/api/support_record_handlers.go`: `UploadProof` usa `r.FormFile("file")`; `DownloadProof` usa `http.ServeFile`.
  - Rutas: `POST /api/pension/records/{id}/upload-proof` y `GET /api/pension/records/{id}/proof`.
  - Frontend `RegistrosPage.tsx` muestra el nombre del archivo y un botón de upload.
- **Precedente de facturas de servicios (SPEC-043)**: el "comprobante" es un **link de Google Drive** (`drive_url`), no un archivo subido. No se reutiliza ese enfoque porque el requerimiento pide almacenar los PDFs en el volumen.
- **Configuración de datos**: `internal/config/config.go` define `DATA_DIR` (default `./data`); `cmd/server/main.go` crea `cfg.DataDir` y abre `app.db` en `filepath.Join(cfg.DataDir, "app.db")`. El `Dockerfile` define `ENV DATA_DIR=/app/data` y `VOLUME ["/app/data"]`; `docker-compose.yml` monta el volumen externo `p40la` en `/app/data`.
- **UI existente**:
  - `CardMenu` (`frontend/src/components/CardMenu.tsx`) renderiza el menú de 3 puntos con `label`, `icon` y `onClick`.
  - `Icon` ya incluye los nombres `pdf`, `bill`, `upload`, `check`, `cancel`, etc.
  - Patrón de upload con drag & drop: `UploadBillModal.tsx` (usa `<input type="file">` y un área visual; no hay librería externa).
  - Reglas de UI: inputs con tokens del tema (`bg-card`, `text-text`), darkmode verificado, `Select` del repo para dropdowns (no aplica aquí porque no hay selectores nuevos).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Un solo PDF combinado (factura + comprobante) | Un archivo por pago | El usuario necesita distinguirlos y abrirlos por separado; no cumple el requerimiento | ❌ Rechazada |
| Guardar los PDFs como BLOB en SQLite | Un solo backup, transaccional | Infla la DB, WAL y copias; malo para iHost; el usuario pidió el volumen | ❌ Rechazada |
| Guardar solo links externos (Drive, S3) | Cero disco local | Dependencia externa, no cumple "en el volumen", se rompe sin internet | ❌ Rechazada |
| **Guardar `invoice_{n}.pdf` y `receipt_{n}.pdf` en el volumen** | Simple, determinístico, mismo volumen que SQLite, sin dependencias | Hay que respaldar el volumen completo; archivos fuera de la DB | ✅ Seleccionada |
| Agregar columnas `invoice_file_name`/`receipt_file_name` a `debt_bills` | Metadatos explícitos, consulta simple | Migración y cambio de esquema; la ruta determinística no lo necesita | ❌ Rechazada (MVP) |
| **Deducir `has_invoice`/`has_receipt` del filesystem** | Cero migraciones, los iconos reflejan la realidad del disco | `os.Stat` por cuota; si se restaura la DB sin archivos, no hay icono (comportamiento correcto) | ✅ Seleccionada |
| Un único campo multipart `file` | Menos campos | No permite distinguir factura de comprobante | ❌ Rechazada |
| **Campos multipart `invoice` y `receipt` separados** | Claro, permite subir uno, otro o ambos | El handler debe parsear dos partes | ✅ Seleccionada |
| Endpoint de upload separado después del pago | Desacopla pago de archivos | Dos requests, estados parciales; peor UX si falla el segundo | ❌ Rechazada para el pago inicial (se usa como P1 para adjuntar después) |
| **`PUT /pay` con multipart y dos archivos opcionales, manteniendo JSON** | Una sola operación, backwards compatible | Handler debe detectar `Content-Type` y parsear ambos formatos | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Almacenamiento en filesystem dentro de `DATA_DIR`**
- **Contexto**: El usuario exige que los archivos vivan en el volumen Docker, en el mismo path que SQLite, no en la DB ni en un servicio externo.
- **Decisión**: `{DATA_DIR}/documents/{debt_id}/invoice_{n}.pdf` y `receipt_{n}.pdf`. En el contenedor: `/app/data/documents/...`, dentro del volumen `p40la`.
- **Consecuencias**: El respaldo del volumen debe incluir `documents/`; el tamaño de la DB no crece. No se requiere configuración nueva: `DATA_DIR` ya existe.

**ADR-002: Nomenclatura determinística por tipo, sin columnas nuevas**
- **Contexto**: El nombre debe ser determinístico por deuda, cuota y tipo de documento. El usuario propuso inicialmente `payment_1.pdf`; durante la implementación se definió `invoice_1.pdf` para la factura y `receipt_1.pdf` para el comprobante, que reflejan mejor los dos documentos.
- **Decisión**: Usar `installment_number` como número de pago y derivar la ruta con `debt_id`, `installment_number` y `DebtDocumentKind` (`invoice`/`receipt`). No agregar columnas a `debt_bills` en esta spec.
- **Consecuencias**: La presencia de cada documento se calcula con `os.Stat`. Si en el futuro una cuota admite múltiples facturas o comprobantes, habrá que migrar a una tabla de documentos o a una columna de metadatos (documentado como P2).

**ADR-003: Multipart con dos archivos opcionales en el endpoint de pago**
- **Contexto**: El frontend ya usa `PUT /api/debt-bills/{id}/pay` con JSON. Cambiar el contrato a solo multipart rompería tests y clientes. Además hay dos tipos de documento.
- **Decisión**: El handler inspecciona `Content-Type`. Si es `multipart/form-data`, parsea `invoice` y `receipt` por separado; si no, decodifica JSON como hoy. El frontend usa multipart y envía solo los archivos seleccionados.
- **Consecuencias**: Un solo request para pagar + adjuntar hasta dos documentos. La compatibilidad se cubre con tests de ambos formatos.

**ADR-004: Flags calculados, no persistidos**
- **Contexto**: Los iconos deben aparecer en varias vistas. Persistir booleanos requiere migración y puede desincronizarse si un archivo se borra o se restaura mal.
- **Decisión**: Agregar `HasInvoice`/`HasReceipt` a `DebtBill` y `HasDocuments` a `Debt` como campos solo JSON, completados en el service con `os.Stat`/`os.ReadDir`.
- **Consecuencias**: Una llamada a `os.Stat` por cuota y tipo. El costo es despreciable con N acotado. Si un archivo falta, su icono no se muestra (consistente con el disco).

**ADR-005: Escritura atómica y staging de ambos documentos**
- **Contexto**: Un corte de luz o un error a mitad de escritura no debe dejar PDFs corruptos ni bloquear el pago. El iHost tiene RAM limitada.
- **Decisión**: Copiar cada `multipart.File` a un temporal en el directorio destino con `io.Copy` y límite de 10 MB; validar la firma; renombrar con `os.Rename` al path final. Si un documento es inválido, se eliminan los temporales y no se marca el pago. El pago se marca en DB después de validar todos los archivos y antes de los renombres finales.
- **Consecuencias**: Los archivos finales siempre están completos. Si un rename falla después de marcar el pago, la cuota queda pagada con uno o ningún documento; el usuario puede reintentar con `POST /invoice` o `POST /receipt` (P1).

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌──────────────────────────────────┐
│ Frontend (React)                 │
│  DebtPayModal                    │
│   ├─ PdfUploadField (Factura)    │
│   ├─ PdfUploadField (Comprobante)│
│   └─ api.debts.payBill(...)      │
│  DebtBillsPage / Calendar        │
│   └─ DebtDocumentButtons         │
└───────────────┬──────────────────┘
                │ multipart/form-data
                │ invoice + receipt opcionales
                ▼
┌──────────────────────────────────┐
│ Go API                           │
│  PUT /api/debt-bills/{id}/pay    │
│  GET/POST .../invoice            │
│  GET/POST .../receipt            │
└───────────────┬──────────────────┘
                │ DebtService
                ▼
┌──────────────────────────────────┐        ┌──────────────────────────────────┐
│ SQLite (app.db)                  │        │ DATA_DIR/documents/              │
│  debt_bills (sin cambios)        │        │  └─ {debt_id}/                   │
│  paid_at, payment_reference      │        │      ├─ invoice_{n}.pdf          │
└──────────────────────────────────┘        │      └─ receipt_{n}.pdf          │
                                            └──────────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 Backend — `DebtService`

- **Responsabilidad**: validar y persistir factura y comprobante, calcular flags, resolver rutas de descarga.
- **Interfaz**:
  ```go
  type DebtDocumentKind string

  const (
      DebtDocumentInvoice DebtDocumentKind = "invoice"
      DebtDocumentReceipt DebtDocumentKind = "receipt"
  )

  type PDFUpload struct {
      FileName string
      Reader   io.Reader
  }

  func (s *DebtService) PayBill(ctx context.Context, id int64, paidAt time.Time, paymentReference string) (*models.DebtBill, error) // compatibilidad
  func (s *DebtService) PayBillWithDocuments(ctx context.Context, id int64, paidAt time.Time, paymentReference string, invoice, receipt *PDFUpload) (*models.DebtBill, error)
  func (s *DebtService) SaveDocument(ctx context.Context, id int64, kind DebtDocumentKind, upload PDFUpload) (*models.DebtBill, error) // P1
  func (s *DebtService) DocumentPath(ctx context.Context, id int64, kind DebtDocumentKind) (filePath, fileName string, err error)
  func (s *DebtService) ListBills(ctx context.Context, debtID int64) ([]models.DebtBill, error)
  func (s *DebtService) ListBillsByMonth(ctx context.Context, year, month int) ([]models.DebtBill, error)
  ```

  Errores centinela: `ErrDocumentAfterPayment` (la cuota quedó pagada pero falló el guardado de un documento) y `ErrDocumentNotFound` (la cuota no tiene el documento solicitado). `PayBill` se conserva como wrapper de `PayBillWithDocuments(..., nil, nil)`.
- **Dependencias**: `storage`, `os`, `path/filepath`, `io`, `bytes`, `strconv`, `time`.
- **Ubicación**: `internal/services/debt.go`.

#### 4.2.2 Backend — `DebtHandlers`

- **Responsabilidad**: parsear JSON o multipart, delegar al service, responder errores HTTP y servir los PDFs.
- **Interfaz**:
  - `PayDebtBill`: detecta `Content-Type`; si es multipart, parsea `invoice` y `receipt`; si no, JSON como hoy.
  - `DownloadDebtInvoice` / `DownloadDebtReceipt`: `GET /api/debt-bills/{id}/invoice|receipt`.
  - `UploadDebtInvoice` / `UploadDebtReceipt` (P1): `POST /api/debt-bills/{id}/invoice|receipt`.
  - Helper `formFileUpload(r, field)` para obtener cada parte opcional.
- **Ubicación**: `internal/api/debt_handlers.go`, rutas en `internal/api/routes.go`.

#### 4.2.3 Modelo

- **Responsabilidad**: exponer los flags al frontend sin tocar la tabla.
- **Cambio**:
  ```go
  // internal/models/debt_bill.go
  HasInvoice bool `json:"has_invoice"`
  HasReceipt bool `json:"has_receipt"`
  // internal/models/debt.go
  HasDocuments bool `json:"has_documents"`
  ```
- **Nota**: no se agrega ninguna columna a `debt_bills` ni migración. Los tres campos se calculan desde el filesystem.

#### 4.2.4 Frontend — `DebtPayModal` y `PdfUploadField`

- **Responsabilidad**: seleccionar/arrastrar la factura y el comprobante, validarlos en cliente, enviarlos junto con el pago y mostrar errores.
- **Componente reutilizable** `frontend/src/components/PdfUploadField.tsx`:
  - Área con borde punteado (`border-dashed`), tokens `bg-card`/`text-text-secondary`, `rounded-ios-sm`, `min-h-[92px]`.
  - Botón/área clickeable y soporte de drag & drop + teclado.
  - Muestra nombre, tamaño, icono `pdf` y botón de quitar (`cancel`).
  - Valida PDF, archivo vacío y máximo 10 MB; muestra toast con claves genéricas `pdf_*`.
- **Modal** `frontend/src/components/DebtPayModal.tsx`:
  - Mantiene `invoiceFile` y `receiptFile` en estado.
  - Renderiza dos `PdfUploadField`: "Factura" y "Comprobante", ambos opcionales.
  - Envía `invoice` y `receipt` por separado en el `FormData`.
- **Utilidad** `frontend/src/utils/pdfFile.ts`: `MAX_RECEIPT_BYTES`, `isPdfFile`, `formatFileSize`.

#### 4.2.5 Frontend — indicadores y menú

- `frontend/src/components/DebtDocumentButtons.tsx`: botones compartidos con icono `bill` (factura) y `pdf` (comprobante), cada uno abre su endpoint.
- `frontend/src/components/DebtDocumentModal.tsx`: modal único por tipo de documento.
  - Sin archivo: muestra `PdfUploadField` y botón "Subir".
  - Con archivo: muestra `invoice_{n}.pdf`/`receipt_{n}.pdf`, botón "Abrir" y botón "Remover".
  - "Remover" abre `DeleteModal` como segundo modal; el usuario debe escribir "confirmo" para habilitar la eliminación.
- `frontend/src/pages/DebtBillsPage.tsx`: usa `DebtDocumentButtons` en cada cuota. Para cuotas pagadas, `billMenuOptions` muestra solo "Factura" y "Comprobante", que abren `DebtDocumentModal`. No hay opciones "Adjuntar" ni "Reemplazar" en el menú.
- `frontend/src/pages/DeudasPage.tsx`: indicador PDF si `has_documents=true`.
- `frontend/src/components/DebtCalendar.tsx`: `DebtDocumentButtons` en `BillRow` (desktop y móvil).
- `frontend/src/components/DebtAnalysis.tsx`: `DebtDocumentButtons` en resultados de búsqueda y cuotas pagadas por adelantado.
- `frontend/src/api/index.ts`: `payBill(id, body, invoice?, receipt?)`; `invoiceUrl`, `receiptUrl`, `uploadInvoice`, `uploadReceipt`, `removeInvoice`, `removeReceipt`.
- `frontend/src/types/index.ts`: `DebtBill.has_invoice?`, `DebtBill.has_receipt?`, `Debt.has_documents?`.
- i18n: `frontend/public/i18n/es.json` y `en.json`.

### 4.3 Modelo de datos

No hay cambios de esquema. Se reutiliza `debt_bills`:

```
Entidad: DebtBill (tabla debt_bills)
- id: INTEGER PK
- debt_id: INTEGER FK debts
- installment_number: INTEGER (1..N, UNIQUE con debt_id)
- due_date: TEXT
- amount: REAL
- status: TEXT ('pending' | 'paid')
- paid_at: DATETIME
- payment_reference: TEXT
- deleted_at: DATETIME
- created_at, updated_at
```

Campos calculados (no persistidos):
```
- has_invoice: bool → os.Stat({DATA_DIR}/documents/{debt_id}/invoice_{n}.pdf)
- has_receipt: bool → os.Stat({DATA_DIR}/documents/{debt_id}/receipt_{n}.pdf)
- has_documents: bool → os.ReadDir({DATA_DIR}/documents/{debt_id}) con al menos un invoice_*/receipt_*.pdf
```

Layout en disco:
```
/app/data/                      # volumen Docker p40la
├── app.db
├── app.db-wal
├── app.db-shm
└── documents/
    ├── 1/
    │   ├── invoice_1.pdf       # deuda 1, cuota 1: factura
    │   ├── receipt_1.pdf       # deuda 1, cuota 1: comprobante
    │   └── invoice_2.pdf       # deuda 1, cuota 2: factura
    └── 2/
        └── receipt_1.pdf       # deuda 2, cuota 1: comprobante
```

### 4.4 APIs / Contratos

#### Endpoint: `PUT /api/debt-bills/{id}/pay`

**Request JSON (compatibilidad)**:
```json
{
  "paid_at": "2026-11-07",
  "payment_reference": "demo-reference-001"
}
```

**Request multipart/form-data**:
```
paid_at: 2026-11-07
payment_reference: demo-reference-001
invoice: <PDF opcional>
receipt: <PDF opcional>
```

**Response 200**:
```json
{
  "id": 1,
  "debt_id": 1,
  "installment_number": 1,
  "due_date": "2026-11-15",
  "amount": 100.0,
  "status": "paid",
  "paid_at": "2026-11-07T00:00:00Z",
  "payment_reference": "demo-reference-001",
  "has_invoice": true,
  "has_receipt": true
}
```

**Response Error**:
```json
{
  "error": "invalid_file",
  "message": "El documento debe ser un archivo PDF"
}
```

Códigos:
- `400 invalid_paid_at`: fecha faltante o inválida.
- `400 invalid_file`: documento vacío, no PDF, mayor a 10 MB o error de escritura.
- `400 already_paid`: la cuota ya estaba pagada.
- `404 not_found`: la cuota no existe o está eliminada.
- `413 request_too_large`: body mayor al límite.
- `500 document_save_failed`: la cuota quedó pagada pero un documento no pudo guardarse.
- `500 internal_error`: error inesperado de DB o filesystem.

#### Endpoint: `GET /api/debt-bills/{id}/invoice` y `GET /api/debt-bills/{id}/receipt`

**Response 200**:
```
Content-Type: application/pdf
Content-Disposition: inline; filename="invoice_1.pdf"   # o receipt_1.pdf
X-Content-Type-Options: nosniff
<bytes del PDF>
```

**Response Error**:
```json
{
  "error": "not_found",
  "message": "La cuota no tiene el documento solicitado"
}
```

Con `?download=1` el `Content-Disposition` pasa a `attachment`.

#### Endpoints: `POST /api/debt-bills/{id}/invoice` y `POST /api/debt-bills/{id}/receipt`

Se usan desde el modal de documento cuando falta el archivo.

**Request multipart/form-data**:
```
file: <PDF obligatorio>
```

**Response 200**: `DebtBill` actualizado con `has_invoice`/`has_receipt` correspondiente.

**Errores**: `400 invalid_file`, `400 not_paid`, `404 not_found`.

#### Endpoints: `DELETE /api/debt-bills/{id}/invoice` y `DELETE /api/debt-bills/{id}/receipt`

**Request**: sin body.

**Response 200**: `DebtBill` actualizado; el flag del documento removido queda en `false` y el otro no cambia.

**Errores**: `400 not_paid`, `404 not_found` (la cuota o el documento no existe), `400 invalid_kind`.

### 4.5 Dependencias

- **Internas**:
  - `internal/services/debt.go`, `internal/api/debt_handlers.go`, `internal/api/routes.go`, `internal/models/debt_bill.go`, `internal/models/debt.go`, `cmd/server/main.go`.
  - Frontend: `DebtPayModal.tsx`, `PdfUploadField.tsx`, `DebtDocumentButtons.tsx`, `DebtDocumentModal.tsx`, `DeleteModal.tsx`, `DebtBillsPage.tsx`, `DeudasPage.tsx`, `DebtCalendar.tsx`, `DebtAnalysis.tsx`, `api/index.ts`, `types/index.ts`, `utils/pdfFile.ts`, i18n.
- **Externas**: ninguna nueva. Solo stdlib de Go y APIs nativas del navegador (`File`, `FormData`, `DragEvent`).

### 4.6 Manejo de errores y estados parciales

| Situación | Comportamiento |
|-----------|----------------|
| Un documento no es PDF o supera 10 MB | Rechazo antes de pagar; se limpian los temporales del otro documento; la cuota sigue pendiente. |
| Ambos documentos válidos | Se escriben dos temporales, se paga, se renombran ambos. |
| Falla la escritura de un temporal | Rechazo; la cuota no se paga; se limpian los temporales. |
| Falla el update de pago en DB | Se limpian los temporales; la cuota sigue pendiente. |
| Falla el rename de un documento después de pagar | La cuota queda pagada con los documentos que sí se guardaron; se informa el error y se puede reintentar con `POST /invoice` o `POST /receipt`. |
| Un archivo falta al consultar | Su flag es `false` y su endpoint devuelve 404; el otro documento sigue disponible. |
| Reemplazo de un documento | Se escribe el nuevo temporal y se renombra sobre el anterior; el otro tipo no se toca. |

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: El modal de pago muestra dos áreas de upload independientes, "Factura" y "Comprobante", con drag & drop y botón de selección.
- [ ] **CA-002**: Se puede pagar sin documentos, solo con factura, solo con comprobante o con ambos.
- [ ] **CA-003**: Al pagar con ambos PDFs válidos se crean `{DATA_DIR}/documents/{debt_id}/invoice_{n}.pdf` y `receipt_{n}.pdf` (creando el directorio si no existe), la cuota queda `paid` con `has_invoice=true` y `has_receipt=true`.
- [ ] **CA-004**: Al pagar solo con factura o solo con comprobante, se crea únicamente el archivo correspondiente y el otro flag queda `false`.
- [ ] **CA-005**: Se rechaza con mensaje claro cualquier documento que no sea PDF, esté vacío o supere 10 MB; en esos casos la cuota no cambia de estado y no queda ningún archivo final ni temporal.
- [ ] **CA-006**: `GET /api/debt-bills/{id}/invoice` y `GET /api/debt-bills/{id}/receipt` devuelven el PDF correcto con `Content-Type: application/pdf`; los iconos y opciones del menú abren el documento correspondiente en una pestaña nueva.
- [ ] **CA-007**: Los iconos de factura (`bill`) y comprobante (`pdf`) aparecen por separado en el detalle de la deuda, el calendario (desktop y móvil) y el análisis (búsqueda y pagadas por adelantado). La tarjeta de deuda muestra un indicador si `has_documents=true`.
- [ ] **CA-008**: El menú de 3 puntos de una cuota pagada muestra únicamente **"Factura"** y **"Comprobante"**; cada opción abre el modal del documento correspondiente. No existen textos "Adjuntar", "Reemplazar", "Ver factura" ni "Ver comprobante" en el menú.
- [ ] **CA-009**: Re-subir una factura reemplaza solo `invoice_{n}.pdf` y no afecta el comprobante; re-subir un comprobante reemplaza solo `receipt_{n}.pdf`.
- [ ] **CA-010**: El endpoint de pago sigue aceptando JSON (sin documentos) y devuelve ambos flags en `false`.
- [ ] **CA-011**: `documents/` está dentro de `DATA_DIR` (`/app/data`), sobrevive a un reinicio del contenedor y no requiere un volumen nuevo.
- [ ] **CA-012**: No se agregó ninguna migración ni columna a `debt_bills`; `PRAGMA table_info(debt_bills)` no cambia.
- [ ] **CA-013**: Los archivos se sirven solo por los endpoints autenticados; no hay listado estático ni acceso directo por URL a `documents/`.
- [ ] **CA-014**: Si se borra un archivo manualmente del volumen, solo su flag pasa a `false` y su endpoint devuelve 404; el otro documento sigue visible y descargable.
- [ ] **CA-015**: El modal de documento permite subir el PDF si no existe; si existe, muestra los botones "Abrir" y "Remover". "Remover" abre el modal de confirmación que exige escribir "confirmo" y, al confirmar, elimina solo ese archivo.
- [ ] **CA-018**: `DELETE /api/debt-bills/{id}/invoice` y `DELETE /api/debt-bills/{id}/receipt` remueven solo el archivo indicado, actualizan sus flags y dejan intacto el otro documento.
- [ ] **CA-019**: Remover un documento inexistente devuelve `404`; remover en una cuota pendiente devuelve `400 not_paid`.
- [ ] **CA-020**: Después de subir una factura o comprobante desde el modal, el modal permanece abierto y muestra los botones "Abrir" y "Remover" del documento recién subido.
- [ ] **CA-021**: Hacer click fuera del modal de documento no lo cierra; solo lo cierran la X o los botones Cancelar/Cerrar.
- [ ] **CA-016**: Con `?download=1`, ambos endpoints devuelven `Content-Disposition: attachment`.
- [ ] **CA-017**: En el detalle de deuda, el menú de 3 puntos de cada cuota queda visible y clickeable sin superposición con el badge de estado ("Pendiente"/"Pagada") ni con el monto, en móvil y desktop.
- [ ] **CA-DARK**: Las dos áreas de upload, los nombres de archivo y los textos de ayuda usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificaron en darkmode.
- [ ] **CA-I18N**: Las claves nuevas existen en `frontend/public/i18n/es.json` y `en.json`, y `npm run build` las copia a `public/i18n/`.
- [ ] **CA-BUILD**: `go test ./...`, `go vet ./...` y `npm run build` (que ejecuta `check-ui-rules.cjs`) pasan sin errores.

### 5.2 No funcionales

- [ ] **CA-NF-001**: El upload de dos PDFs de 10 MB no supera ~2 MB de RAM adicional en el server (streaming a disco).
- [ ] **CA-NF-002**: El listado de cuotas de una deuda con 60 cuotas responde en < 300 ms en el iHost (con `os.Stat` por cuota y tipo).
- [ ] **CA-NF-003**: No quedan archivos `.tmp` visibles tras un upload exitoso.
- [ ] **CA-NF-004**: Los endpoints de descarga envían `X-Content-Type-Options: nosniff` y solo sirven archivos con firma `%PDF-`.
- [ ] **CA-NF-005**: Las rutas finales no contienen ningún componente provisto por el usuario; `debt_id` e `installment_number` se formatean como enteros y el tipo es un enum.

### 5.3 Testing

- **Unit tests** (`internal/services/debt_receipt_test.go`):
  - Pago con ambos PDFs crea `invoice_1.pdf` y `receipt_1.pdf` y devuelve ambos flags.
  - Pago solo con factura / solo con comprobante.
  - Validación: archivo vacío, extensión no PDF, contenido sin firma `%PDF-`, tamaño > 10 MB.
  - Si un documento es inválido, no se paga y no queda ningún archivo (limpieza de temporales del otro).
  - `SaveDocument` solo permite cuotas pagadas, crea directorios y devuelve el flag correcto.
  - Reemplazo de un documento no afecta al otro.
  - `DELETE` remueve solo el documento indicado, actualiza flags y devuelve `ErrDocumentNotFound` si no existe.
  - `DocumentPath` y los flags reflejan la ausencia de un archivo.
  - `List` y `GetByID` calculan `has_documents`.
- **Integration tests** (`internal/api/debt_receipt_test.go`):
  - `PUT /pay` con JSON sigue funcionando.
  - `PUT /pay` con multipart + factura y comprobante crea ambos archivos y devuelve ambos flags.
  - `GET /invoice` y `GET /receipt` devuelven 200, `application/pdf`, `nosniff` y los bytes correctos.
  - `GET` sin archivo devuelve 404 para cada tipo.
  - `POST /invoice` y `POST /receipt` adjuntan a una cuota pagada y rechazan cuotas pendientes.
  - `DELETE /invoice` y `DELETE /receipt` remueven solo el documento indicado; repetir el delete devuelve 404.
  - Usar `t.TempDir()` como `DATA_DIR` y PDFs sintéticos generados en el test (nunca archivos reales).
- **Frontend**:
  - `npm run check:ui` (no `<select>` nativo).
  - `npm run build`.
  - Prueba manual en local: pagar con ambos, solo factura y solo comprobante; ver iconos separados; abrir cada PDF; abrir el modal Factura/Comprobante; subir si falta; remover con confirmación "confirmo"; recargar la página.
- **iHost**:
  - Tras el deploy, subir PDFs sintéticos, verificar las rutas con SSH (`/app/data/documents/...`), reiniciar el contenedor y confirmar que los iconos y las descargas siguen funcionando.
  - Verificar que el respaldo del volumen incluya `documents/` (ver REQ-010).
  - No usar facturas ni comprobantes reales en pruebas ni en el repo.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: rutas de factura/comprobante, validación PDF, staging atómico, flags y tests unitarios | 0.75 día | Ninguna |
| 2 | API: multipart con `invoice`/`receipt`, endpoints GET/POST/DELETE, tests de handlers | 0.5 día | Fase 1 |
| 3 | Frontend: `PdfUploadField`, modal de pago con dos áreas, API client y tipos | 0.5 día | Fase 2 |
| 4 | Frontend: `DebtDocumentButtons`, `DebtDocumentModal` con Abrir/Remover, menú Factura/Comprobante, tarjeta de deuda e i18n | 0.5 día | Fase 3 |
| 5 | Validación local, build, pruebas manuales y documentación de respaldo | 0.25 día | Fases 3-4 |

**Total estimado**: ~2.5 días de desarrollo + pruebas.

### 6.2 Milestones

1. **MVP**: pagar con factura y/o comprobante, guardar ambos en `{DATA_DIR}/documents/...`, flags separados, iconos y opciones "Factura"/"Comprobante", descarga inline.
2. **V1.0**: adjuntar/reemplazar después del pago, indicador en la tarjeta de deuda, fallback `?download=1`, documentación de respaldo del volumen y verificación en iHost.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El respaldo actual solo copia `app.db` y se pierden los PDFs | Alta | Alto | Documentar y actualizar el procedimiento de backup para incluir `documents/` (REQ-010); verificar en el release. |
| Crecimiento de disco por hasta 20 MB por pago | Media | Medio | Límite de 10 MB por archivo, un archivo de cada tipo por cuota; monitorear el volumen; limpieza de huérfanos como P2. |
| Nombres basados en `installment_number` dejan de ser únicos si se agregan pagos parciales o múltiples facturas | Baja | Medio | Documentar la limitación; si se necesita, migrar a tabla de documentos con ID propio (P2). |
| `os.Stat` por cuota y tipo degrada listados grandes | Baja | Bajo | N acotado; si crece, listar el directorio una vez por deuda y cachear por request. |
| Carrera entre dos uploads simultáneos del mismo documento | Baja | Bajo | `os.Rename` atómico; la última escritura gana; no se corrompe el archivo. |
| PDF malicioso o con contenido activo | Baja | Medio | Validar firma `%PDF-`, `Content-Type: application/pdf`, `nosniff`, solo servir por endpoints autenticados; no ejecutar el archivo en el server. |
| Safari/iOS no previsualiza PDF inline | Media | Bajo | `?download=1` para forzar descarga; en el MVP se abre en pestaña nueva. |
| `DATA_DIR` mal configurado en desarrollo (default `./data`) | Baja | Bajo | Usar `t.TempDir()` en tests; documentar que en Docker es `/app/data`; no crear directorios en startup. |
| Error de permisos en el volumen al escribir | Baja | Alto | El contenedor corre como root; verificar en iHost con PDFs sintéticos antes del release; `MkdirAll` con 0755. |
| Borrar una deuda o reducir cuotas deja archivos huérfanos | Media | Bajo | Aceptado en MVP; documentar limpieza P2; los archivos no se sirven si la cuota ya no existe. |
| Compatibilidad con clientes que envían JSON al endpoint de pago | Baja | Medio | Mantener el parseo JSON y cubrirlo con tests; el frontend usa multipart solo cuando hay documentos. |
| Un rename falla después de pagar y queda un solo documento | Baja | Bajo | La cuota queda pagada; el usuario puede reintentar el documento faltante con el endpoint P1. |

---

## 8. Notas y Referencias

- **SPEC-043**: Acción Pagar en facturas con fecha de pago, comprobante (link de Drive) y referencia. Precedente de campos de pago, pero **no** de upload de archivos.
- **SPEC-051**: Emails y generación mensual de registros de pensión; incluye el patrón de comprobantes en `uploads/payment-proofs` (validación 10 MB, `SaveProof`/`ProofPath`, `UploadProof`/`DownloadProof`).
- **SPEC-054**: Módulo Deudas, modelo `debt_bills`, generación de cuotas y acción Pagar.
- **SPEC-104**: Finalización automática de deudas y separación de cuotas pendientes/pagadas.
- **SPEC-066**: worktrees obligatorios; esta spec se crea en `feature/SPEC-122`.
- **SPEC-060 / SPEC-118**: reglas de UI, tokens del tema y guardrail de dropdowns.
- **Docker**: `Dockerfile` (`ENV DATA_DIR=/app/data`, `VOLUME ["/app/data"]`) y `docker-compose.yml` (volumen externo `p40la` montado en `/app/data`).
- **Config**: `internal/config/config.go` (`DATA_DIR`, default `./data`).
- **UI**: `frontend/src/components/CardMenu.tsx`, `frontend/src/components/Icons.tsx` (iconos `pdf`, `bill`, `upload`), `frontend/src/components/UploadBillModal.tsx` (patrón visual de drag & drop).
- **Datos sintéticos**: los ejemplos usan `debt_id=1`, `installment_number=1`, `invoice_1.pdf`, `receipt_1.pdf`, `demo-reference-001`. Nunca usar facturas o comprobantes reales en tests, specs ni commits.

### Fuera de alcance

- OCR o extracción de datos de los PDFs.
- Múltiples facturas o comprobantes por cuota.
- Sincronización con Google Drive u otro almacenamiento externo.
- Comprobantes para facturas de servicios (ya existe `drive_url` en SPEC-043).
- Limpieza automática de archivos huérfanos.
- Firma digital o validación de autenticidad de los documentos.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-07 | opencode | Creación inicial de la especificación (SPEC-122). |
| 2026-11-07 | opencode | Implementación inicial con un único comprobante y renombre de `payment_{n}.pdf` a `receipt_{n}.pdf`. |
| 2026-11-07 | opencode | Cambio de alcance a pedido del usuario: dos documentos independientes por pago, **factura** (`invoice_{n}.pdf`) y **comprobante** (`receipt_{n}.pdf`), con upload, iconos y acciones separadas. |
| 2026-11-07 | opencode | Fix de UI solicitado en pruebas: la fila de cuota reserva espacio a la derecha (`pr-14`) para que el badge de estado no tape el menú de 3 puntos. |
| 2026-11-07 | opencode | Cambio de UX a pedido del usuario: el menú solo muestra "Factura" y "Comprobante", que abren un modal con Subir (si falta), Abrir y Remover. Remover exige confirmar escribiendo "confirmo" en un segundo modal. Se agregaron endpoints DELETE. |
| 2026-11-07 | opencode | Ajuste solicitado en pruebas: después de subir un documento, el modal permanece abierto en el estado Abrir/Remover; el refresco de datos es silencioso para no desmontarlo. |
| 2026-11-07 | opencode | Ajuste solicitado en pruebas: el modal de documento no se cierra al hacer click fuera; solo con la X o los botones Cancelar/Cerrar. |
