---
title: "Comprobantes PDF en pagos de deudas: upload, almacenamiento en volumen y visualización"
id: "SPEC-122"
status: "draft"
author: "opencode"
created: "2026-11-07"
updated: "2026-11-07"
github_issue: 125
---

# Comprobantes PDF en pagos de deudas: upload, almacenamiento en volumen y visualización

**ID**: SPEC-122  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-11-07  
**Actualizado**: 2026-11-07

---

## 1. Resumen Ejecutivo

Hoy el pago de una cuota de deuda solo registra la fecha de pago y una referencia de texto (`DebtPayModal` → `PUT /api/debt-bills/{id}/pay`). No existe forma de adjuntar el comprobante bancario, por lo que el usuario pierde la evidencia documental del pago y debe buscarla fuera de la app.

Esta spec agrega la posibilidad de adjuntar **un archivo PDF** al pagar una cuota, mediante un botón de selección y un área de **drag & drop** en el modal de pago. El archivo se guarda en el volumen Docker, junto a la base SQLite, con una ruta determinística basada en los IDs de la deuda: `{DATA_DIR}/documents/{debt_id}/payment_{installment_number}.pdf`. El directorio se crea si no existe. Además, toda cuota que tenga comprobante muestra un **icono de PDF/factura** en las vistas donde aparece (detalle de deuda, calendario y análisis), y el menú de 3 puntos incluye la opción **"Comprobante"** para abrirlo en una pestaña nueva.

Consideraciones iHost: no se agregan dependencias nuevas (solo `net/http`, `mime/multipart`, `os`, `path/filepath`, `io`, `bytes` de la stdlib de Go), no se modifica el esquema de `debt_bills` (la existencia del comprobante se deduce del filesystem) y no se crea un volumen nuevo: `documents/` vive dentro de `DATA_DIR` (`/app/data` en el contenedor), que ya está montado en el volumen `p40la`. El upload se hace en streaming a disco con un límite de 10 MB para no inflar la RAM del iHost.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: En el modal de pago de una cuota (`DebtPayModal`) agregar un **área de upload** que permita:
   - Arrastrar y soltar un archivo PDF.
   - Seleccionar el archivo con un botón ("Seleccionar PDF" / "Choose PDF").
   - Ver el nombre del archivo seleccionado, su tamaño y un botón para quitarlo antes de confirmar.
   - El archivo es **opcional**: se puede pagar sin comprobante.
2. **REQ-002**: Validar el archivo en frontend y backend:
   - Solo formato **PDF** (extensión `.pdf` y contenido que empiece con la firma `%PDF-`).
   - Tamaño máximo **10 MB** (consistente con los comprobantes de pensión, SPEC-051).
   - Rechazar archivos vacíos, corruptos o de otro tipo con un mensaje claro.
3. **REQ-003**: Guardar el PDF en el volumen Docker con la ruta determinística:
   ```
   {DATA_DIR}/documents/{debt_id}/payment_{installment_number}.pdf
   ```
   Ejemplo sintético: deuda `id=1`, cuota `installment_number=1` → `documents/1/payment_1.pdf`.
   - `DATA_DIR` es el mismo directorio donde vive `app.db` (`/app/data` en el contenedor, volumen Docker `p40la`).
   - Crear el directorio `documents/{debt_id}` con `os.MkdirAll(..., 0o755)` si no existe.
   - No crear el directorio al arrancar el server: solo en el primer upload.
4. **REQ-004**: Extender `PUT /api/debt-bills/{id}/pay` para aceptar `multipart/form-data` con los campos `paid_at` (obligatorio), `payment_reference` (opcional) y `file` (opcional, PDF). Mantener compatibilidad con el body JSON actual (`application/json`) para no romper clientes existentes ni tests.
5. **REQ-005**: Agregar `GET /api/debt-bills/{id}/receipt` (autenticado) que devuelva el PDF con `Content-Type: application/pdf` y `Content-Disposition: inline; filename="payment_{n}.pdf"`. Si la cuota no existe, está eliminada o no tiene archivo → `404`.
6. **REQ-006**: Exponer en las respuestas de cuotas el campo booleano `has_receipt`:
   - `GET /api/debts/{id}/bills` (detalle de deuda).
   - `GET /api/debt-bills?year=&month=` (calendario).
   - Respuesta de `PUT /api/debt-bills/{id}/pay`.
   El campo se calcula verificando la existencia del archivo en disco (`os.Stat`), sin agregar columnas ni migraciones.
7. **REQ-007**: Mostrar un **icono de PDF/factura** clickeable en toda cuota que tenga comprobante, en las siguientes vistas:
   - Filas de cuotas del detalle de deuda (`DebtBillsPage`).
   - Filas de cuotas del calendario, tanto en la grilla desktop como en la agenda móvil (`DebtCalendar`).
   - Resultados de búsqueda y listado de cuotas pagadas por adelantado del análisis (`DebtAnalysis`).
   Al hacer clic, abrir el PDF en una pestaña nueva (`window.open(url, '_blank', 'noopener,noreferrer')`).
8. **REQ-008**: En el menú de 3 puntos (`CardMenu`) del detalle de deuda, agregar la opción **"Comprobante"** (`deudas.view_receipt`) cuando la cuota tenga archivo. Debe abrir el PDF igual que el icono. La opción **"Pagar"** se mantiene para cuotas pendientes.
9. **REQ-009**: Agregar las claves i18n en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json` para: título de la sección de comprobante, botón de selección, texto de drag & drop, formatos aceptados, tamaño máximo, errores y acción "Comprobante". La fuente de verdad del i18n es `frontend/public/i18n/`, nunca `public/i18n/`.
10. **REQ-010**: Documentar que `documents/` forma parte del volumen Docker y debe incluirse en los respaldos junto con `app.db`. El respaldo del solo archivo SQLite no preserva los comprobantes. No crear un volumen adicional ni cambiar el `docker-compose.yml`.
11. **REQ-011**: El guardado debe ser **atómico**: escribir primero en un archivo temporal dentro del mismo directorio y luego `os.Rename` al nombre final. Si el pago falla, eliminar el temporal. Si se sube un comprobante nuevo para una cuota que ya tenía uno, reemplazarlo (última versión gana) sin dejar archivos huérfanos.
12. **REQ-012**: Nunca usar el nombre de archivo provisto por el usuario para construir la ruta. El nombre final se deriva exclusivamente de `debt_id` e `installment_number` (enteros), evitando path traversal.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: Permitir adjuntar o reemplazar el comprobante **después** de pagar, mediante `POST /api/debt-bills/{id}/receipt` (`multipart/form-data`, campo `file`). Solo para cuotas en estado `paid`. En el menú de 3 puntos mostrar "Adjuntar comprobante" / "Reemplazar comprobante" según corresponda.
2. **REQ-014**: Agregar en la tarjeta de deuda (`DeudasPage`) un indicador discreto si la deuda tiene al menos un comprobante, usando un resumen calculado (`has_receipts`) en `GET /api/debts`. Es opcional para el MVP; si se implementa, no debe degradar el listado (máximo un `os.ReadDir` por deuda).
3. **REQ-015**: Si el navegador no puede previsualizar el PDF inline (caso Safari/iOS), ofrecer un enlace de descarga alternativo (`?download=1`) en el mismo endpoint. No es obligatorio para el MVP, pero sí recomendable para móvil.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Limpieza de archivos huérfanos cuando una cuota se elimina lógicamente por reducción de `installments_total` o cuando una deuda se elimina definitivamente. No implementar en el MVP; documentar como deuda técnica.
2. **REQ-017**: Soporte de múltiples comprobantes por cuota o de imágenes (JPG/PNG). Fuera de alcance: el requerimiento pide un PDF único.
3. **REQ-018**: Registro de auditoría del upload/reemplazo de comprobantes (quién y cuándo), reutilizando el patrón de historial de facturas (SPEC-070).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el upload de 10 MB debe completarse en < 5 s en la red local del iHost. El cálculo de `has_receipt` es O(N) con `os.Stat` por cuota; N está acotado por la cantidad de cuotas de una deuda o de un mes (< 100 típicamente).
- **Memoria**: el archivo se copia en streaming desde `multipart.File` al disco; no se carga completo en un `[]byte` de 10 MB. `r.ParseMultipartForm(1 << 20)` limita a 1 MB en RAM y el resto va a archivos temporales.
- **Almacenamiento**: máximo 10 MB por cuota. El volumen del iHost debe tener espacio suficiente; no se agrega rotación automática en esta spec.
- **Seguridad**:
  - Endpoints protegidos con `authMiddleware`, igual que el resto del API.
  - Validación de firma `%PDF-` además de extensión y `Content-Type`.
  - `X-Content-Type-Options: nosniff` en la descarga.
  - Ruta construida solo con enteros de la DB; sin directory listing ni servido estático de `documents/`.
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
- **Precedente de facturas de servicios (SPEC-043)**: el "comprobante" es un **link de Google Drive** (`drive_url`), no un archivo subido. No se reutiliza ese enfoque porque el requerimiento pide almacenar el PDF en el volumen.
- **Configuración de datos**: `internal/config/config.go` define `DATA_DIR` (default `./data`); `cmd/server/main.go` crea `cfg.DataDir` y abre `app.db` en `filepath.Join(cfg.DataDir, "app.db")`. El `Dockerfile` define `ENV DATA_DIR=/app/data` y `VOLUME ["/app/data"]`; `docker-compose.yml` monta el volumen externo `p40la` en `/app/data`.
- **UI existente**:
  - `CardMenu` (`frontend/src/components/CardMenu.tsx`) renderiza el menú de 3 puntos con `label`, `icon` y `onClick`.
  - `Icon` ya incluye los nombres `pdf`, `bill`, `upload`, `check`, `cancel`, etc.
  - Patrón de upload con drag & drop: `UploadBillModal.tsx` (usa `<input type="file">` y un área visual; no hay librería externa).
  - Reglas de UI: inputs con tokens del tema (`bg-card`, `text-text`), darkmode verificado, `Select` del repo para dropdowns (no aplica aquí porque no hay selectores nuevos).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Guardar el PDF como BLOB en SQLite | Un solo backup, transaccional | Infla la DB, WAL y copias; malo para iHost; el usuario pidió explícitamente el volumen | ❌ Rechazada |
| Guardar solo un link externo (Drive, S3) | Cero disco local | Dependencia externa, no cumple "en el volumen", se rompe sin internet | ❌ Rechazada |
| **Guardar en `{DATA_DIR}/documents/{debt_id}/payment_{n}.pdf`** | Simple, determinístico, mismo volumen que SQLite, sin dependencias | Hay que respaldar el volumen completo; archivos fuera de la DB | ✅ Seleccionada |
| Agregar columna `receipt_file_name` a `debt_bills` | Metadatos explícitos, consulta simple | Migración y cambio de esquema; el requerimiento no lo necesita porque la ruta es determinística | ❌ Rechazada (MVP) |
| **Deducir `has_receipt` del filesystem** | Cero migraciones, el icono refleja la realidad del disco | `os.Stat` por cuota; si se restaura la DB sin los archivos, no hay icono (comportamiento correcto) | ✅ Seleccionada |
| Endpoint de upload separado después del pago | Desacopla pago de archivo | Dos requests, estados parciales; peor UX si falla el segundo | ❌ Rechazada para el pago inicial (se usa solo como P1 para adjuntar después) |
| **`PUT /pay` con multipart y archivo opcional, manteniendo JSON** | Una sola operación, backwards compatible | Handler debe detectar `Content-Type` y parsear ambos formatos | ✅ Seleccionada |
| PDF + imágenes | Más flexible | El requerimiento pide PDF; validar imágenes agrega superficie y peso | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Almacenamiento en filesystem dentro de `DATA_DIR`**
- **Contexto**: El usuario exige que los archivos vivan en el volumen Docker, en el mismo path que SQLite, no en la DB ni en un servicio externo.
- **Decisión**: `{DATA_DIR}/documents/{debt_id}/payment_{installment_number}.pdf`. En el contenedor: `/app/data/documents/...`, dentro del volumen `p40la`.
- **Consecuencias**: El respaldo del volumen debe incluir `documents/`; el tamaño de la DB no crece. No se requiere configuración nueva: `DATA_DIR` ya existe.

**ADR-002: Nomenclatura determinística y sin columna nueva**
- **Contexto**: El requerimiento define el nombre `payment_1.pdf` para el pago 1 de la deuda. Cada cuota tiene un `installment_number` único por deuda (`UNIQUE(debt_id, installment_number)`).
- **Decisión**: Usar `installment_number` como "número de pago" y derivar la ruta con `debt_id` e `installment_number`. No agregar columnas a `debt_bills` en esta spec.
- **Consecuencias**: La presencia del comprobante se calcula con `os.Stat`. Si en el futuro una cuota admite pagos parciales o múltiples comprobantes, habrá que migrar a una tabla `debt_bill_receipts` o a una columna de metadatos (documentado como P2).

**ADR-003: Multipart en el endpoint de pago existente**
- **Contexto**: El frontend ya usa `PUT /api/debt-bills/{id}/pay` con JSON. Cambiar el contrato a solo multipart rompería tests y clientes.
- **Decisión**: El handler inspecciona `Content-Type`. Si es `multipart/form-data`, parsea campos + archivo; si no, decodifica JSON como hoy. El frontend usa multipart cuando hay archivo y JSON cuando no lo hay (o siempre multipart, ambas variantes son válidas).
- **Consecuencias**: Un solo request para pagar + adjuntar. La compatibilidad se cubre con tests de ambos formatos.

**ADR-004: `has_receipt` calculado, no persistido**
- **Contexto**: El icono debe aparecer en varias vistas. Persistir un booleano requiere migración y puede desincronizarse si el archivo se borra o se restaura mal.
- **Decisión**: Agregar `HasReceipt bool` al modelo `DebtBill` (solo JSON, no DB) y completarlo en el service con `os.Stat` después de leer las cuotas.
- **Consecuencias**: Una llamada a `os.Stat` por cuota. El costo es despreciable con N acotado. Si el archivo falta, el icono no se muestra (consistente con el disco).

**ADR-005: Escritura atómica y streaming**
- **Contexto**: Un corte de luz o un error a mitad de escritura no debe dejar un PDF corrupto ni bloquear el pago. El iHost tiene RAM limitada.
- **Decisión**: Copiar el `multipart.File` a un temporal en el directorio destino con `io.Copy` y límite de 10 MB; validar la firma; renombrar con `os.Rename` al path final. El pago se marca en DB después de validar el archivo y antes del rename final; si el pago falla, se borra el temporal.
- **Consecuencias**: El archivo final siempre está completo. Si el rename falla después de marcar el pago, la cuota queda pagada sin comprobante y el usuario puede reintentar con `POST /receipt` (P1). Se documenta ese caso en los criterios de aceptación.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌─────────────────────────────┐
│ Frontend (React)            │
│  DebtPayModal               │
│   ├─ input file / drag&drop │
│   └─ api.debts.payBill(...) │
│  DebtBillsPage / Calendar   │
│   └─ icono PDF + CardMenu   │
└──────────────┬──────────────┘
               │ multipart/form-data (file opcional)
               ▼
┌─────────────────────────────┐
│ Go API                      │
│  PUT /api/debt-bills/{id}/pay│
│  GET /api/debt-bills/{id}/receipt
│  POST /api/debt-bills/{id}/receipt (P1)
└──────────────┬──────────────┘
               │ DebtService
               ▼
┌─────────────────────────────┐        ┌──────────────────────────────┐
│ SQLite (app.db)             │        │ DATA_DIR/documents/          │
│  debt_bills (sin cambios)   │        │  └─ {debt_id}/               │
│  paid_at, payment_reference │        │      └─ payment_{n}.pdf      │
└─────────────────────────────┘        └──────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 Backend — `DebtService`

- **Responsabilidad**: validar y persistir el comprobante, calcular `has_receipt`, resolver la ruta de descarga.
- **Interfaz**:
  ```go
  type ReceiptUpload struct {
      FileName string
      Reader   io.Reader
  }

  func NewDebtService(
      debts *storage.DebtStorage,
      debtBills *storage.DebtBillStorage,
      institutions *storage.InstitutionStorage,
      currencies *storage.CurrencyStorage,
      dataDir string,
  ) *DebtService

  func (s *DebtService) PayBill(ctx context.Context, id int64, paidAt time.Time, paymentReference string, receipt *ReceiptUpload) (*models.DebtBill, error)
  func (s *DebtService) SaveReceipt(ctx context.Context, id int64, receipt ReceiptUpload) (*models.DebtBill, error) // P1
  func (s *DebtService) ReceiptPath(ctx context.Context, id int64) (filePath, fileName string, err error)
  func (s *DebtService) ListBills(ctx context.Context, debtID int64) ([]models.DebtBill, error)
  func (s *DebtService) ListBillsByMonth(ctx context.Context, year, month int) ([]models.DebtBill, error)
  ```
- **Dependencias**: `storage`, `os`, `path/filepath`, `io`, `bytes`, `strconv`, `time`.
- **Ubicación**: `internal/services/debt.go`.

#### 4.2.2 Backend — `DebtHandlers`

- **Responsabilidad**: parsear JSON o multipart, delegar al service, responder errores HTTP y servir el PDF.
- **Interfaz**:
  - `PayDebtBill`: detecta `Content-Type`; si es multipart, `r.ParseMultipartForm(1 << 20)` y `r.FormFile("file")`; si no, JSON como hoy.
  - `DownloadDebtReceipt`: `GET /api/debt-bills/{id}/receipt`.
  - `UploadDebtReceipt` (P1): `POST /api/debt-bills/{id}/receipt`.
- **Ubicación**: `internal/api/debt_handlers.go`, rutas en `internal/api/routes.go`.

#### 4.2.3 Modelo

- **Responsabilidad**: exponer `has_receipt` al frontend sin tocar la tabla.
- **Cambio**:
  ```go
  // internal/models/debt_bill.go
  HasReceipt bool `json:"has_receipt"`
  ```
- **Nota**: no se agrega ninguna columna a `debt_bills` ni migración.

#### 4.2.4 Frontend — `DebtPayModal`

- **Responsabilidad**: seleccionar/arrastrar el PDF, validarlo en cliente, enviarlo junto con el pago y mostrar errores.
- **UI**:
  - Área con borde punteado (`border-dashed`), tokens `bg-card`/`text-text-secondary`, `rounded-ios-sm`, `min-h-[44px]`.
  - Texto de ayuda: "Arrastrá y soltá un PDF o seleccioná un archivo (máx. 10 MB)".
  - Botón "Seleccionar PDF" con `<input type="file" accept="application/pdf,.pdf" className="hidden">`.
  - Al seleccionar: mostrar nombre + tamaño + icono `pdf` + botón de quitar (`cancel`).
  - Durante el guardado: spinner y botón deshabilitado.
- **Ubicación**: `frontend/src/components/DebtPayModal.tsx`.

#### 4.2.5 Frontend — indicadores y menú

- `frontend/src/pages/DebtBillsPage.tsx`: agregar icono PDF y opción "Comprobante" en `billMenuOptions`.
- `frontend/src/components/DebtCalendar.tsx`: icono PDF en `BillRow` (grilla desktop y agenda móvil).
- `frontend/src/components/DebtAnalysis.tsx`: icono PDF en resultados de búsqueda y en la lista de cuotas pagadas por adelantado.
- `frontend/src/api/index.ts`: `payBill` con `FormData` cuando hay archivo; `receiptUrl(id)`; `uploadReceipt(id, file)` (P1).
- `frontend/src/types/index.ts`: `DebtBill.has_receipt?: boolean`.
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

Campo calculado (no persistido):
```
- has_receipt: bool → os.Stat({DATA_DIR}/documents/{debt_id}/payment_{installment_number}.pdf)
```

Layout en disco:
```
/app/data/                      # volumen Docker p40la
├── app.db
├── app.db-wal
├── app.db-shm
└── documents/
    ├── 1/
    │   ├── payment_1.pdf       # deuda 1, cuota 1
    │   └── payment_2.pdf       # deuda 1, cuota 2
    └── 2/
        └── payment_1.pdf       # deuda 2, cuota 1
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
file: <PDF opcional>
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
  "has_receipt": true
}
```

**Response Error**:
```json
{
  "error": "invalid_file",
  "message": "El comprobante debe ser un PDF de máximo 10 MB"
}
```

Códigos:
- `400 invalid_paid_at`: fecha faltante o inválida.
- `400 invalid_file`: archivo vacío, no PDF, mayor a 10 MB o error de escritura.
- `400 already_paid`: la cuota ya estaba pagada.
- `404 not_found`: la cuota no existe o está eliminada.
- `413 request_too_large`: body mayor al límite (si se aplica `MaxBytesReader`).
- `500 internal_error`: error inesperado de DB o filesystem.

#### Endpoint: `GET /api/debt-bills/{id}/receipt`

**Response 200**:
```
Content-Type: application/pdf
Content-Disposition: inline; filename="payment_1.pdf"
X-Content-Type-Options: nosniff
<bytes del PDF>
```

**Response Error**:
```json
{
  "error": "not_found",
  "message": "La cuota no tiene comprobante"
}
```

#### Endpoint: `POST /api/debt-bills/{id}/receipt` (P1)

**Request multipart/form-data**:
```
file: <PDF obligatorio>
```

**Response 200**: `DebtBill` actualizado con `has_receipt: true`.

**Errores**: `400 invalid_file`, `400 not_paid`, `404 not_found`.

### 4.5 Dependencias

- **Internas**:
  - `internal/services/debt.go`, `internal/api/debt_handlers.go`, `internal/api/routes.go`, `internal/models/debt_bill.go`, `cmd/server/main.go`.
  - Frontend: `DebtPayModal.tsx`, `DebtBillsPage.tsx`, `DebtCalendar.tsx`, `DebtAnalysis.tsx`, `api/index.ts`, `types/index.ts`, i18n.
- **Externas**: ninguna nueva. Solo stdlib de Go y APIs nativas del navegador (`File`, `FormData`, `DragEvent`).

### 4.6 Manejo de errores y estados parciales

| Situación | Comportamiento |
|-----------|----------------|
| Archivo no PDF o > 10 MB | Rechazo antes de escribir; la cuota no se paga. |
| Falla la escritura del temporal | Rechazo; la cuota no se paga; se borra el temporal. |
| Falla el update de pago en DB | Se borra el temporal; la cuota sigue pendiente. |
| Falla el `os.Rename` después de pagar | La cuota queda pagada sin comprobante; se informa el error y el usuario puede reintentar con `POST /receipt` (P1). |
| El archivo final no existe al consultar | `has_receipt=false` y `GET /receipt` → 404. |
| Reemplazo de comprobante | Se escribe el nuevo temporal y se renombra sobre el anterior; el anterior se pierde solo si el rename tiene éxito. |

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: Al abrir el modal de pago de una cuota pendiente, se ve un área de upload con soporte de drag & drop y un botón para seleccionar un PDF.
- [ ] **CA-002**: Se puede pagar una cuota **sin** archivo; el flujo actual sigue funcionando igual.
- [ ] **CA-003**: Al pagar con un PDF válido, se crea `{DATA_DIR}/documents/{debt_id}/payment_{installment_number}.pdf` (si no existe el directorio, se crea) y la cuota queda `paid` con `has_receipt=true`.
- [ ] **CA-004**: El archivo se rechaza con mensaje claro si no es PDF, está vacío o supera 10 MB; en esos casos la cuota no cambia de estado.
- [ ] **CA-005**: `GET /api/debt-bills/{id}/receipt` devuelve el PDF con `Content-Type: application/pdf` y se abre en una pestaña nueva desde el icono y desde la opción "Comprobante" del menú de 3 puntos.
- [ ] **CA-006**: El icono de PDF aparece en el detalle de la deuda, en el calendario (desktop y móvil) y en el análisis (búsqueda y pagadas por adelantado) para toda cuota con comprobante.
- [ ] **CA-007**: La opción del menú de 3 puntos se rotula **"Comprobante"** / **"Receipt"** y solo aparece si `has_receipt=true`.
- [ ] **CA-008**: Re-subir un comprobante para la misma cuota reemplaza el archivo anterior sin dejar temporales ni archivos duplicados.
- [ ] **CA-009**: El endpoint de pago sigue aceptando JSON (sin archivo) y devuelve `has_receipt` en la respuesta.
- [ ] **CA-010**: `documents/` está dentro de `DATA_DIR` (`/app/data`), sobrevive a un reinicio del contenedor y no requiere un volumen nuevo.
- [ ] **CA-011**: No se agregó ninguna migración ni columna a `debt_bills`; `PRAGMA table_info(debt_bills)` no cambia.
- [ ] **CA-012**: Los archivos se sirven solo por el endpoint autenticado; no hay listado estático ni acceso directo por URL a `documents/`.
- [ ] **CA-013**: Si el archivo se borra manualmente del volumen, `has_receipt` pasa a `false` y el endpoint devuelve 404 (sin errores 500).
- [ ] **CA-014**: La opción "Comprobante" no aparece en cuotas sin archivo ni en cuotas eliminadas.
- [ ] **CA-015**: (P1) Se puede adjuntar/reemplazar el comprobante de una cuota ya pagada desde el menú de 3 puntos.
- [ ] **CA-DARK**: El área de upload, el nombre del archivo y los textos de ayuda usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificaron en darkmode (texto y placeholders legibles).
- [ ] **CA-I18N**: Las claves nuevas existen en `frontend/public/i18n/es.json` y `en.json`, y `npm run build` las copia a `public/i18n/`.
- [ ] **CA-BUILD**: `go test ./...`, `go vet ./...` y `npm run build` (que ejecuta `check-ui-rules.cjs`) pasan sin errores.

### 5.2 No funcionales

- [ ] **CA-NF-001**: El upload de un PDF de 10 MB no supera ~2 MB de RAM adicional en el server (streaming a disco).
- [ ] **CA-NF-002**: El listado de cuotas de una deuda con 60 cuotas responde en < 300 ms en el iHost (con `os.Stat` por cuota).
- [ ] **CA-NF-003**: El PDF se escribe de forma atómica: no quedan archivos `.tmp` visibles tras un upload exitoso.
- [ ] **CA-NF-004**: El endpoint de descarga envía `X-Content-Type-Options: nosniff` y solo sirve archivos con firma `%PDF-`.
- [ ] **CA-NF-005**: La ruta final no contiene ningún componente provisto por el usuario; `debt_id` e `installment_number` se formatean como enteros.

### 5.3 Testing

- **Unit tests** (`internal/services/debt_receipt_test.go`):
  - `receiptPath` genera `documents/1/payment_1.pdf` para deuda 1, cuota 1.
  - Validación: archivo vacío, no PDF, PDF válido, tamaño > 10 MB (usar `bytes.NewReader` con datos sintéticos).
  - `SaveReceipt` crea directorios, escribe el archivo y devuelve `has_receipt=true`.
  - Reemplazo sobrescribe el archivo anterior.
  - `ReceiptPath` devuelve 404-like error si no existe.
- **Integration tests** (`internal/api/debt_handlers_test.go` o similar):
  - `PUT /pay` con JSON sigue funcionando.
  - `PUT /pay` con multipart + PDF crea el archivo y marca `paid`.
  - `GET /receipt` devuelve 200, `application/pdf` y los bytes correctos.
  - `GET /receipt` sin archivo devuelve 404.
  - `POST /receipt` (P1) rechaza cuotas pendientes.
  - Usar `t.TempDir()` como `DATA_DIR` y PDFs sintéticos generados en el test (nunca archivos reales).
- **Frontend**:
  - `npm run check:ui` (no `<select>` nativo).
  - `npm run build`.
  - Prueba manual en local: pagar con y sin PDF, ver icono en las tres vistas, abrir el PDF, recargar la página.
- **iHost**:
  - Tras el deploy, subir un PDF sintético, verificar la ruta con SSH (`/app/data/documents/...`), reiniciar el contenedor y confirmar que el icono y la descarga siguen funcionando.
  - Verificar que el respaldo del volumen incluya `documents/` (ver REQ-010).
  - No usar comprobantes reales en pruebas ni en el repo.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: ruta de comprobantes, validación PDF, escritura atómica, `has_receipt` y tests unitarios | 0.5 día | Ninguna |
| 2 | API: multipart en `PUT /pay`, `GET /receipt`, tests de handlers y rutas | 0.5 día | Fase 1 |
| 3 | Frontend: upload con drag & drop en `DebtPayModal`, API client y tipos | 0.5 día | Fase 2 |
| 4 | Frontend: iconos en detalle/calendario/análisis, opción "Comprobante" y i18n | 0.5 día | Fase 3 |
| 5 | P1: `POST /receipt` para adjuntar/reemplazar después del pago | 0.25 día | Fase 2 |
| 6 | Validación local, build, pruebas manuales y documentación de respaldo | 0.25 día | Fases 3-5 |

**Total estimado**: ~2 días de desarrollo + pruebas.

### 6.2 Milestones

1. **MVP**: pagar con PDF opcional, guardar en `{DATA_DIR}/documents/...`, `has_receipt`, icono y opción "Comprobante" en el detalle de deuda; descarga inline.
2. **V1.0**: iconos en calendario y análisis, P1 para adjuntar/reemplazar después del pago, documentación de respaldo del volumen, verificación en iHost.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El respaldo actual solo copia `app.db` y se pierden los PDFs | Alta | Alto | Documentar y actualizar el procedimiento de backup para incluir `documents/` (REQ-010); verificar en el release. |
| Crecimiento de disco por PDFs de 10 MB acumulados | Media | Medio | Límite de 10 MB, un archivo por cuota; monitorear el volumen; limpieza de huérfanos como P2. |
| Nombres de archivo basados en `installment_number` dejan de ser únicos si se agregan pagos parciales | Baja | Medio | Documentar la limitación; si se necesita, migrar a tabla de comprobantes con ID propio (P2). |
| `os.Stat` por cuota degrada listados grandes | Baja | Bajo | N acotado; si crece, cachear por request o listar el directorio una vez por deuda. |
| Carrera entre dos uploads simultáneos de la misma cuota | Baja | Bajo | `os.Rename` atómico; la última escritura gana; no se corrompe el archivo. |
| PDF malicioso o con contenido activo | Baja | Medio | Validar firma `%PDF-`, `Content-Type: application/pdf`, `nosniff`, solo servir por endpoint autenticado; no ejecutar el archivo en el server. |
| Safari/iOS no previsualiza PDF inline | Media | Bajo | P1: `?download=1` para forzar descarga; en el MVP se abre en pestaña nueva. |
| `DATA_DIR` mal configurado en desarrollo (default `./data`) | Baja | Bajo | Usar `t.TempDir()` en tests; documentar que en Docker es `/app/data`; no crear directorios en startup. |
| Error de permisos en el volumen al escribir | Baja | Alto | El contenedor corre como root; verificar en iHost con un PDF sintético antes del release; `MkdirAll` con 0755. |
| Borrar una deuda o reducir cuotas deja archivos huérfanos | Media | Bajo | Aceptado en MVP; documentar limpieza P2; los archivos no se sirven si la cuota ya no existe. |
| Compatibilidad con clientes que envían JSON al endpoint de pago | Baja | Medio | Mantener el parseo JSON y cubrirlo con tests; el frontend usa multipart solo cuando hay archivo. |

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
- **Datos sintéticos**: los ejemplos usan `debt_id=1`, `installment_number=1`, `payment_1.pdf`, `demo-reference-001`. Nunca usar comprobantes reales en tests, specs ni commits.

### Fuera de alcance

- OCR o extracción de datos del PDF.
- Múltiples comprobantes por cuota.
- Sincronización con Google Drive u otro almacenamiento externo.
- Comprobantes para facturas de servicios (ya existe `drive_url` en SPEC-043).
- Limpieza automática de archivos huérfanos.
- Firma digital o validación de autenticidad del comprobante.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-07 | opencode | Creación inicial de la especificación (SPEC-122). |
