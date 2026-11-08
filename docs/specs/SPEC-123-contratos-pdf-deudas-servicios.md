---
title: "Contratos PDF para deudas y servicios: upload, visualización y eliminación"
id: "SPEC-123"
status: "in_progress"
author: "opencode"
created: "2026-11-07"
updated: "2026-11-07"
github_issue: 126
---

# Contratos PDF para deudas y servicios: upload, visualización y eliminación

**ID**: SPEC-123  
**Estado**: in_progress
**Autor**: opencode  
**Creado**: 2026-11-07  
**Actualizado**: 2026-11-07

---

## 1. Resumen Ejecutivo

Hoy ni las deudas ni los servicios tienen un lugar donde guardar el contrato que respalda la relación con la institución. El usuario debe mantener esos PDFs fuera de la app y, cuando los necesita, buscarlos manualmente. Esta spec agrega un **contrato PDF único por deuda** y **un contrato PDF único por servicio**, accesible desde el menú de 3 puntos de cada entidad.

El flujo es completamente modal, sin páginas nuevas:

- Si la entidad **no tiene contrato**, el modal muestra un área de **drag & drop** y un botón para seleccionar un PDF.
- Si la entidad **ya tiene contrato**, el mismo modal permite **abrirlo en una pestaña nueva** o **quitarlo**. La eliminación exige un segundo modal de confirmación en el que se debe escribir `confirmo`, reutilizando el componente `DeleteModal` existente.

Los archivos se guardan en el volumen Docker, con la misma raíz que usa la spec más reciente (SPEC-122), pero bajo el directorio `contracts/`:

```
{DATA_DIR}/contracts/debts/{debt_id}/contract.pdf
{DATA_DIR}/contracts/services/{service_id}/contract.pdf
```

`DATA_DIR` es el mismo directorio de `app.db` (`/app/data` en el contenedor, volumen `p40la`). En el iHost no se agregan dependencias, no se modifica el esquema SQLite y no se crea un volumen nuevo: la existencia del contrato se deduce del filesystem y se expone como `has_contract`. El upload se hace en streaming a disco con un límite de 10 MB para no cargar el PDF completo en RAM.

---

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar la opción **"Contrato"** en los menús de 3 puntos de deudas y servicios:
   - Tarjetas de la lista de deudas (`DeudasPage`).
   - Header del detalle de deuda (`DebtBillsPage`, menú de acciones actualmente usado para "Editar").
   - Tarjetas de la lista de servicios (`ServicesPage`).
   - Header del detalle de servicio (`BillsPage`, menú de acciones que ya incluye editar/subir factura/crear/renovar/webhook).
   La opción se rotula siempre **"Contrato"** / **"Contract"** y usa el icono `pdf` existente en `frontend/src/components/Icons.tsx`.
2. **REQ-002**: Al seleccionar "Contrato" se abre un **modal reutilizable** (`ContractModal`) con dos estados:
   - **Sin contrato**: área de drag & drop, botón "Seleccionar PDF", nombre/tamaño del archivo elegido, botón para quitarlo antes de subir y botón "Subir contrato".
   - **Con contrato**: indicador de que el contrato existe, botón **"Abrir contrato"** y botón de peligro **"Quitar contrato"**.
   No se crea ninguna página ni ruta nueva. Todo ocurre en el modal sobre la página actual.
3. **REQ-003**: Validar el archivo en frontend y backend:
   - Solo formato **PDF** (extensión `.pdf`, sin distinguir mayúsculas, y firma de contenido `%PDF-`).
   - Tamaño máximo **10 MB** (consistente con comprobantes de pensión, SPEC-051, y comprobantes de deuda, SPEC-122).
   - Rechazar archivos vacíos, corruptos o de otro tipo con un mensaje claro.
   - El archivo es obligatorio para subir un contrato; no se permite guardar un contrato vacío.
4. **REQ-004**: Guardar el contrato en el volumen Docker con ruta determinística y nombre fijo:
   ```
   {DATA_DIR}/contracts/debts/{debt_id}/contract.pdf
   {DATA_DIR}/contracts/services/{service_id}/contract.pdf
   ```
   - Crear el directorio con `os.MkdirAll(..., 0o755)` solo en el primer upload.
   - No crear directorios al arrancar el server.
   - El nombre original del archivo **nunca** se usa para construir la ruta; el nombre final es siempre `contract.pdf`.
   - `debt_id` y `service_id` se formatean como enteros; el "owner" (`debts`/`services`) se elige de una whitelist interna, nunca de input del usuario.
   - El archivo final queda con permisos `0644` para que los respaldos del volumen puedan leerlo; el directorio se crea con `0755`.
5. **REQ-005**: Agregar endpoints autenticados para cada entidad:
   ```
   POST   /api/debts/{id}/contract    # subir o reemplazar
   GET    /api/debts/{id}/contract    # ver/descargar
   DELETE /api/debts/{id}/contract    # quitar

   POST   /api/services/{id}/contract
   GET    /api/services/{id}/contract
   DELETE /api/services/{id}/contract
   ```
   - `POST` acepta `multipart/form-data` con el campo `file` y responde `200 {"has_contract": true}`.
   - `GET` responde el PDF inline (`Content-Type: application/pdf`, `Content-Disposition: inline; filename="contract.pdf"`, `X-Content-Type-Options: nosniff`). Si no existe el contrato o la entidad no existe/está eliminada → `404`.
   - `DELETE` elimina el archivo y responde `204`. Si no existe → `404`.
6. **REQ-006**: Exponer el campo booleano calculado `has_contract` en toda respuesta que devuelva una deuda o un servicio (`GET /api/debts`, `GET /api/debts/{id}`, `GET /api/services`, `GET /api/services/{id}`, y las respuestas de crear/actualizar). Se calcula con `os.Stat` sobre la ruta determinística; **no se persiste** ni requiere migración.
7. **REQ-007**: En el estado "con contrato" del modal:
   - "Abrir contrato" abre el PDF en una pestaña nueva con `window.open(url, '_blank', 'noopener,noreferrer')`.
   - "Quitar contrato" abre un segundo modal de confirmación reutilizando `frontend/src/components/DeleteModal.tsx`, que exige escribir exactamente `confirmo` para habilitar la eliminación. Al confirmar, se llama al `DELETE`, se muestra un toast de éxito y el modal vuelve al estado "sin contrato" o se cierra.
   - El texto del modal de confirmación debe dejar claro que el PDF se elimina del volumen y que la acción no se puede deshacer.
8. **REQ-008**: Agregar las claves i18n en `frontend/public/i18n/es.json` y `frontend/public/i18n/en.json`: título del modal, textos de drag & drop, botón de selección, formatos/tamaño aceptados, errores de validación, "Abrir contrato", "Quitar contrato", títulos/subtítulos de confirmación, éxitos y errores de red. La fuente de verdad es `frontend/public/i18n/`; nunca editar `public/i18n/` directamente.
9. **REQ-009**: La opción "Contrato" debe estar disponible para **todas** las deudas y servicios, tengan o no contrato. El estado inicial del modal se determina con `has_contract`.
10. **REQ-010**: Documentar que `contracts/` forma parte del volumen Docker y debe incluirse en los respaldos junto con `app.db` y `documents/`. El respaldo del solo archivo SQLite no preserva los contratos. No crear un volumen adicional ni cambiar `docker-compose.yml`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Mostrar un indicador discreto (icono `pdf`) en las tarjetas de deuda y servicio cuando `has_contract=true`, para que el usuario detecte de un vistazo qué entidades tienen contrato.
2. **REQ-012**: Soportar `GET /api/{owner}/{id}/contract?download=1` para forzar descarga (`Content-Disposition: attachment`), como alternativa cuando el navegador móvil no previsualiza PDF inline (Safari/iOS).
3. **REQ-013**: Mostrar en el modal el tamaño y la fecha de modificación del contrato existente. Si se implementa, exponerlos como campos calculados (`contract_size`, `contract_updated_at`) o con un `HEAD`/metadata endpoint; no persistirlos en la DB.
4. **REQ-014**: Al eliminar definitivamente una deuda o un servicio, borrar su directorio de contrato para no dejar archivos huérfanos. Tener en cuenta que el borrado actual es lógico (`deleted_at`); la limpieza solo debe ejecutarse si el registro se elimina físicamente, o documentarse como tarea programada.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Permitir reemplazar el contrato sin quitarlo primero, con un botón "Reemplazar contrato" en el estado "con contrato".
2. **REQ-016**: Soportar múltiples contratos por entidad (por ejemplo, contrato principal + anexos) con una tabla de metadatos. Fuera de alcance: el requerimiento pide un PDF único.
3. **REQ-017**: Registro de auditoría del upload/eliminación (quién y cuándo), reutilizando el patrón de historial de facturas (SPEC-070).
4. **REQ-018**: Extracción de datos del contrato (OCR, número de contrato, partes, vigencia). Fuera de alcance.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el upload de un PDF de 10 MB debe completarse en < 5 s en la red local del iHost. El cálculo de `has_contract` es un `os.Stat` por entidad en los listados; N está acotado por la cantidad de deudas/servicios (< 500 típicamente) y el costo es despreciable.
- **Memoria**: el archivo se copia en streaming desde `multipart.File` al disco; no se carga completo en un `[]byte` de 10 MB. `r.ParseMultipartForm(1 << 20)` limita a 1 MB en RAM y el resto va a archivos temporales.
- **Almacenamiento**: máximo 10 MB por contrato, un archivo por deuda/servicio. No se agrega rotación automática en esta spec.
- **Seguridad**:
  - Endpoints protegidos con `authMiddleware`, igual que el resto del API.
  - Validación de firma `%PDF-` además de extensión y `Content-Type`.
  - `X-Content-Type-Options: nosniff` en la descarga.
  - Ruta construida solo con IDs enteros y owner de whitelist; sin directory listing ni servido estático de `contracts/`.
  - Límite de tamaño aplicado antes de escribir el archivo final (`http.MaxBytesReader` + `io.LimitReader`).
  - No se aceptan nombres de archivo del usuario para construir rutas (evita path traversal).
- **Disponibilidad**: sin cambios en `/health`. Si `DATA_DIR/contracts` no existe, el server arranca normalmente y lo crea en el primer upload.
- **iHost**: cero dependencias nuevas, cero migraciones, cero procesos en background. Build multi-arch (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) sin cambios.
- **UI**: tokens del tema (`bg-card`, `text-text`, `text-text-secondary`), darkmode verificado, sin colores hardcodeados. No hay selectores nuevos; si se agregara alguno, debe usar el componente `Select` del repo (prohibido `<select>` nativo).

---

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Spec más reciente (SPEC-122)**: define comprobantes PDF de pagos de deudas en `{DATA_DIR}/documents/{debt_id}/payment_{installment_number}.pdf`, con escritura atómica, validación de PDF de 10 MB y `has_receipt` calculado del filesystem. Esta spec **reutiliza la misma raíz de volumen** (`DATA_DIR`, `/app/data`, volumen `p40la`) pero usa el directorio `contracts/`, tal como pidió el usuario.
- **Precedente de comprobantes de pensión (SPEC-051)**: `internal/services/support_record.go` guarda archivos en `{DATA_DIR}/uploads/payment-proofs/{year}/{month}/{id}.ext` con límite de 10 MB; `UploadProof`/`DownloadProof` en `internal/api/support_record_handlers.go` muestran el patrón de `r.FormFile("file")` y `http.ServeFile`.
- **Modelo y UI de deudas**:
  - `internal/models/debt.go`: `Debt` sin campos de archivos.
  - `internal/services/debt.go`: `DebtService` con `List`, `GetByID`, `Delete`.
  - `frontend/src/pages/DeudasPage.tsx`: tarjetas con `CardMenu` (editar/eliminar).
  - `frontend/src/pages/DebtBillsPage.tsx`: header con `CreateMenu` (hoy solo "Editar").
- **Modelo y UI de servicios**:
  - `internal/models/service.go`: `Service` sin campos de archivos.
  - `internal/services/service.go`: `ServiceService` con `List`, `GetByID`, `Delete` (soft delete).
  - `frontend/src/pages/ServicesPage.tsx`: tarjetas con `CardMenu` (webhook/sync/editar/eliminar).
  - `frontend/src/pages/BillsPage.tsx`: header con `CreateMenu` (editar/subir factura/crear/renovar/webhook).
- **Patrón de inyección existente**: el repo usa setters para dependencias opcionales, por ejemplo `billService.SetBillHistoryStorage(...)` y `categoryService.SetServiceStorage(...)`. Se adopta el mismo patrón para no romper los 17 call sites de `NewDebtService`/`NewServiceService` en tests.
- **Modal de confirmación existente**: `frontend/src/components/DeleteModal.tsx` ya exige escribir `confirmo` para habilitar la eliminación. Se reutiliza tal cual; no se crea un modal de confirmación nuevo.
- **Patrón de drag & drop existente**: `frontend/src/components/UploadBillModal.tsx` usa un `<input type="file" accept="application/pdf,.pdf" className="hidden">`, área con borde punteado y `onDrop`/`onDragOver`; no hay librería externa.
- **i18n**: `frontend/public/i18n/es.json` y `en.json`; el build de Vite copia esa carpeta a `public/i18n/`.
- **Configuración**: `internal/config/config.go` define `DATA_DIR` (default `./data`); `cmd/server/main.go` crea `cfg.DataDir` y abre `app.db` allí. `Dockerfile` define `ENV DATA_DIR=/app/data` y `VOLUME ["/app/data"]`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Guardar el PDF como BLOB en SQLite | Un solo backup, transaccional | Infla la DB, WAL y copias; malo para iHost; el usuario pidió el volumen | ❌ Rechazada |
| Guardar solo un link externo (Drive, S3) | Cero disco local | Dependencia externa, no cumple "en el volumen", se rompe sin internet | ❌ Rechazada |
| **Guardar en `{DATA_DIR}/contracts/{owner}/{id}/contract.pdf`** | Simple, determinístico, mismo volumen que SQLite, sin dependencias, separado de comprobantes | Hay que respaldar el volumen completo; archivos fuera de la DB | ✅ Seleccionada |
| Reutilizar `{DATA_DIR}/documents/` para contratos | Un solo directorio de documentos | Mezcla comprobantes de pago con contratos y complica la limpieza; el usuario pidió `contracts/` | ❌ Rechazada |
| Agregar columnas `contract_file_name` a `debts`/`services` | Metadatos explícitos, consulta simple | Dos migraciones y cambio de esquema; la ruta es determinística y no hace falta | ❌ Rechazada (MVP) |
| **Deducir `has_contract` del filesystem** | Cero migraciones, el icono refleja la realidad del disco | `os.Stat` por entidad; si se restaura la DB sin archivos, no hay contrato (comportamiento correcto) | ✅ Seleccionada |
| Endpoint genérico `/api/contracts/{owner}/{id}` | Menos rutas | Rompe la convención REST del repo (`/api/debts/...`, `/api/services/...`) y obliga a parsear el owner desde la URL | ❌ Rechazada |
| **Endpoints por entidad (`/api/debts/{id}/contract`, `/api/services/{id}/contract`)** | Consistente con las rutas existentes; auth y validación reutilizadas | Alguna duplicación de handlers | ✅ Seleccionada |
| Página dedicada para subir/ver el contrato | Más espacio para el formulario | Agrega rutas, `BACK_ROUTES`, navegación y estados; el usuario aceptó modal | ❌ Rechazada |
| **Modal reutilizable con estados "sin contrato" / "con contrato"** | Sin rutas nuevas, UX directa, reutiliza `DeleteModal` | Hay que manejar el estado local y refrescar la lista | ✅ Seleccionada |
| Modal de confirmación nuevo | Control total del texto | Duplica un componente que ya existe y ya exige `confirmo` | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Directorio `contracts/` dentro de `DATA_DIR`, con subdirectorio por tipo de entidad**
- **Contexto**: SPEC-122 guarda comprobantes en `{DATA_DIR}/documents/...`; el usuario pidió la misma ruta de volumen pero en `contracts/`. Deudas y servicios pueden tener IDs repetidos, así que un único nivel `contracts/{id}.pdf` colisionaría.
- **Decisión**: `{DATA_DIR}/contracts/debts/{debt_id}/contract.pdf` y `{DATA_DIR}/contracts/services/{service_id}/contract.pdf`. En el contenedor: `/app/data/contracts/...`, dentro del volumen `p40la`.
- **Consecuencias**: El respaldo del volumen debe incluir `contracts/`; el tamaño de la DB no crece. No se requiere configuración nueva: `DATA_DIR` ya existe. La estructura es extensible a futuros tipos de entidad.

**ADR-002: `has_contract` calculado, no persistido**
- **Contexto**: El modal y las tarjetas necesitan saber si existe un contrato. Persistir un booleano requiere migraciones y puede desincronizarse si el archivo se borra o se restaura mal.
- **Decisión**: Agregar `HasContract bool` al modelo `Debt` y `Service` (solo JSON, no DB) y completarlo en el service con `os.Stat` en las respuestas de list/get/create/update.
- **Consecuencias**: Una llamada a `os.Stat` por entidad. Si el archivo falta, `has_contract=false` y el `GET` devuelve 404 (consistente con el disco). No hay migración.

**ADR-003: Inyección del almacén de contratos con setter**
- **Contexto**: `NewDebtService` y `NewServiceService` se usan en 17 lugares (producción y tests). Cambiar sus firmas obliga a tocar todos los tests aunque no prueben contratos.
- **Decisión**: Crear `ContractStore` en `internal/services/contract.go` y agregar `SetContractStore(*ContractStore)` a `DebtService` y `ServiceService`. `cmd/server/main.go` lo inyecta una vez con `cfg.DataDir`. Si el store es `nil`, `has_contract` queda en `false` y los endpoints de contrato no se usan.
- **Consecuencias**: Menos churn en tests; el patrón ya existe en el repo (`SetBillHistoryStorage`, `SetServiceStorage`). El store es compartido y sin estado mutable.

**ADR-004: Escritura atómica y streaming**
- **Contexto**: Un corte de luz o un error a mitad de escritura no debe dejar un PDF corrupto. El iHost tiene RAM limitada.
- **Decisión**: Validar extensión; crear un temporal en el directorio destino con `os.CreateTemp`; leer los primeros 5 bytes para validar `%PDF-` (usando `bufio.Reader.Peek`); copiar el resto con `io.Copy` y un `io.LimitReader` de 10 MB + 1 byte; verificar que no se superó el límite; aplicar `Chmod(0o644)`; `Sync`/`Close` y `os.Rename` al nombre final. Ante cualquier error, borrar el temporal. Si ya existía un contrato, el rename lo reemplaza atómicamente.
- **Consecuencias**: El archivo final siempre está completo. No se cargan 10 MB en RAM. Si el rename falla, queda el contrato anterior (si existía) y se devuelve error; el usuario puede reintentar.

**ADR-005: Modal único con dos estados y reutilización de `DeleteModal`**
- **Contexto**: El usuario pidió que la opción "Contrato" abra el flujo de carga si no hay archivo, o el de ver/quitar si ya existe, y que la eliminación pida escribir `confirmo`. La app ya tiene `DeleteModal` con esa validación y `UploadBillModal` con el patrón visual de drag & drop.
- **Decisión**: `ContractModal` recibe `owner`, `entityId`, `entityName` y `hasContract`; mantiene un estado local `hasContract` que se sincroniza al abrir. El estado "sin contrato" muestra upload; el estado "con contrato" muestra abrir/quitar. "Quitar" monta `DeleteModal`; al confirmar, llama al `DELETE` y vuelve al estado "sin contrato".
- **Consecuencias**: No se agregan rutas ni `BACK_ROUTES`. El modal debe manejar `Escape`, click fuera y bloqueo durante upload/delete, siguiendo el patrón de los modales existentes.

**ADR-006: `POST` para subir o reemplazar**
- **Contexto**: El archivo tiene una ruta fija y una segunda subida reemplaza al anterior. Semánticamente sería un `PUT`, pero el repo usa `POST` para uploads (`/api/services/{id}/bills/upload`, `/api/pension/records/{id}/upload-proof`).
- **Decisión**: `POST /api/{owner}/{id}/contract` para subir o reemplazar. La respuesta es `200 {"has_contract": true}`.
- **Consecuencias**: Consistencia con los uploads existentes. Si en el futuro se quiere idempotencia estricta, se puede migrar a `PUT` sin cambiar la ruta.

---

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
┌──────────────────────────────────────┐
│ Frontend (React)                     │
│  DeudasPage / DebtBillsPage          │
│    └─ CardMenu / CreateMenu          │
│         └─ "Contrato"                │
│  ServicesPage / BillsPage            │
│    └─ CardMenu / CreateMenu          │
│         └─ "Contrato"                │
│  ContractModal                       │
│    ├─ drag & drop + input file       │
│    ├─ open → window.open(...)        │
│    └─ remove → DeleteModal("confirmo")│
└───────────────┬──────────────────────┘
                │ multipart/form-data (file)
                ▼
┌──────────────────────────────────────┐
│ Go API                               │
│  POST   /api/debts/{id}/contract     │
│  GET    /api/debts/{id}/contract     │
│  DELETE /api/debts/{id}/contract     │
│  POST   /api/services/{id}/contract  │
│  GET    /api/services/{id}/contract  │
│  DELETE /api/services/{id}/contract  │
└───────────────┬──────────────────────┘
                │ DebtService / ServiceService
                ▼
┌──────────────────────┐      ┌─────────────────────────────────────┐
│ SQLite (app.db)      │      │ DATA_DIR/contracts/                 │
│  debts (sin cambios) │      │  ├─ debts/{debt_id}/contract.pdf    │
│  services (sin cambios)│    │  └─ services/{id}/contract.pdf     │
└──────────────────────┘      └─────────────────────────────────────┘
```

### 4.2 Componentes

#### 4.2.1 Backend — `ContractStore`

- **Responsabilidad**: resolver rutas, validar PDF, escribir de forma atómica, comprobar existencia y eliminar contratos. No conoce la DB ni valida que la entidad exista (eso lo hacen los services de deuda/servicio).
- **Interfaz**:
  ```go
  type ContractOwner string

  const (
      ContractOwnerDebt    ContractOwner = "debts"
      ContractOwnerService ContractOwner = "services"
  )

  var (
      ErrContractNotFound     = errors.New("el contrato no existe")
      ErrContractInvalid      = errors.New("el archivo de contrato no es un PDF válido")
      ErrContractTooLarge     = errors.New("el contrato supera el tamaño máximo de 10 MB")
      ErrContractEntityNotFound = errors.New("la deuda o el servicio no existe")
  )

  type ContractStore struct {
      rootDir string // {DATA_DIR}/contracts
  }

  func NewContractStore(dataDir string) *ContractStore
  func (s *ContractStore) Exists(owner ContractOwner, id int64) bool
  func (s *ContractStore) Save(owner ContractOwner, id int64, fileName string, r io.Reader) error
  func (s *ContractStore) Path(owner ContractOwner, id int64) (string, error)
  func (s *ContractStore) Delete(owner ContractOwner, id int64) error
  ```
- **Reglas de `Save`**:
  1. Validar `owner` contra la whitelist `debts`/`services`; si no, error.
  2. Validar `id > 0`.
  3. Validar extensión `.pdf` (case-insensitive) y nombre no vacío.
  4. `dir := filepath.Join(rootDir, string(owner), strconv.FormatInt(id, 10))`; `os.MkdirAll(dir, 0o755)`.
  5. `tmp, err := os.CreateTemp(dir, ".contract-*.tmp")`.
  6. `br := bufio.NewReader(r)`; `head, _ := br.Peek(5)`; si `string(head) != "%PDF-"` → `ErrContractInvalid` y limpiar.
  7. Copiar con `io.Copy(tmp, io.LimitReader(br, maxContractSize+1))`; si `n > maxContractSize` → `ErrContractTooLarge`.
  8. `tmp.Sync()`, `tmp.Close()`, `os.Rename(tmp.Name(), filepath.Join(dir, "contract.pdf"))`.
  9. En cualquier error, `os.Remove(tmp.Name())`.
- **Reglas de `Path`**: `os.Stat` del archivo fijo; si no existe → `ErrContractNotFound`. No seguir symlinks provistos por el usuario porque la ruta es fija.
- **Reglas de `Delete`**: `os.Remove` del archivo fijo; si `os.IsNotExist` → `ErrContractNotFound`. El directorio puede quedar vacío; no es necesario borrarlo.
- **Dependencias**: `bufio`, `errors`, `io`, `os`, `path/filepath`, `strconv`, `strings`, `fmt`.
- **Ubicación**: `internal/services/contract.go`.

#### 4.2.2 Backend — `DebtService` y `ServiceService`

- **Responsabilidad**: validar que la entidad existe antes de tocar el archivo y completar `has_contract`.
- **Cambios**:
  ```go
  // internal/services/debt.go
  type DebtService struct {
      // ...
      contracts *ContractStore
  }

  func (s *DebtService) SetContractStore(c *ContractStore)
  func (s *DebtService) SaveContract(ctx context.Context, id int64, fileName string, r io.Reader) error
  func (s *DebtService) ContractPath(ctx context.Context, id int64) (string, error)
  func (s *DebtService) DeleteContract(ctx context.Context, id int64) error
  ```
  Equivalentes en `internal/services/service.go`:
  ```go
  func (s *ServiceService) SetContractStore(c *ContractStore)
  func (s *ServiceService) SaveContract(ctx context.Context, id int64, fileName string, r io.Reader) error
  func (s *ServiceService) ContractPath(ctx context.Context, id int64) (string, error)
  func (s *ServiceService) DeleteContract(ctx context.Context, id int64) error
  ```
- **Validación de entidad**: `s.debts.GetByID` / `s.services.GetByID`; si devuelve `nil` o error de no encontrado → `ErrContractEntityNotFound`. No se permite subir contrato a una entidad eliminada lógicamente.
- **`has_contract`**: en `List`, `GetByID`, `Create` y `Update` se completa `HasContract = s.contracts != nil && s.contracts.Exists(owner, id)`. Si `contracts` es `nil` (tests que no lo inyectan), queda `false`.
- **Ubicación**: `internal/services/debt.go`, `internal/services/service.go`.

#### 4.2.3 Backend — handlers y rutas

- **Responsabilidad**: parsear multipart, aplicar límites, delegar en el service, servir el PDF y mapear errores a HTTP.
- **Handlers en `DebtHandlers`** (`internal/api/debt_handlers.go`):
  ```go
  func (h *DebtHandlers) UploadContract(w http.ResponseWriter, r *http.Request)
  func (h *DebtHandlers) DownloadContract(w http.ResponseWriter, r *http.Request)
  func (h *DebtHandlers) DeleteContract(w http.ResponseWriter, r *http.Request)
  ```
- **Handlers en `ServiceHandlers`** (`internal/api/service_handlers.go`): los mismos tres métodos.
- **Helper compartido** (puede vivir en `internal/api/contract_handlers.go`):
  - `parseContractID(r) (int64, bool)`.
  - `readContractUpload(w, r) (multipart.File, string, bool)`:
    - `r.Body = http.MaxBytesReader(w, r.Body, maxContractSize+1<<20)`.
    - `r.ParseMultipartForm(1 << 20)`.
    - `file, header, err := r.FormFile("file")`; si falla → `400 invalid_file`.
  - `writeContractHeaders(w, download bool)`.
- **Mapeo de errores**:
  - `ErrContractEntityNotFound` / `ErrContractNotFound` → `404 not_found`.
  - `ErrContractInvalid` → `400 invalid_file`.
  - `ErrContractTooLarge` → `413 request_too_large`.
  - Cualquier otro → `500 internal_error`.
- **Rutas en `internal/api/routes.go`** (después de las rutas existentes de cada entidad):
  ```go
  mux.Handle("POST /api/debts/{id}/contract", authMiddleware(http.HandlerFunc(handler.debt.UploadContract)))
  mux.Handle("GET /api/debts/{id}/contract", authMiddleware(http.HandlerFunc(handler.debt.DownloadContract)))
  mux.Handle("DELETE /api/debts/{id}/contract", authMiddleware(http.HandlerFunc(handler.debt.DeleteContract)))

  mux.Handle("POST /api/services/{id}/contract", authMiddleware(http.HandlerFunc(handler.service.UploadContract)))
  mux.Handle("GET /api/services/{id}/contract", authMiddleware(http.HandlerFunc(handler.service.DownloadContract)))
  mux.Handle("DELETE /api/services/{id}/contract", authMiddleware(http.HandlerFunc(handler.service.DeleteContract)))
  ```
- **Wiring en `cmd/server/main.go`**:
  ```go
  contractStore := services.NewContractStore(cfg.DataDir)
  serviceService.SetContractStore(contractStore)
  debtService.SetContractStore(contractStore)
  ```
  No cambia la firma de `NewHandler` ni se agrega un handler nuevo al struct.

#### 4.2.4 Modelos

- **Cambio**:
  ```go
  // internal/models/debt.go
  HasContract bool `json:"has_contract"`

  // internal/models/service.go
  HasContract bool `json:"has_contract"`
  ```
- **Nota**: no se agrega ninguna columna a `debts` ni a `services`; no hay migración.

#### 4.2.5 Frontend — `ContractModal`

- **Ubicación**: `frontend/src/components/ContractModal.tsx`.
- **Props**:
  ```ts
  type ContractOwner = 'debts' | 'services'

  interface ContractModalProps {
    owner: ContractOwner
    entityId: number
    entityName: string
    hasContract: boolean
    onClose: () => void
    onChanged: () => void // refresca la lista/detalle
  }
  ```
- **Estado**:
  - `localHasContract` inicializado con `hasContract` y sincronizado cuando cambia la prop o se abre.
  - `mode: 'empty' | 'uploading' | 'existing' | 'deleting'`.
  - `file: File | null`, `error: string | null`, `dragging: boolean`.
- **UI estado "sin contrato"**:
  - Área con borde punteado (`border-dashed`), tokens `bg-card`/`text-text-secondary`, `rounded-ios-sm`, `min-h-[160px]`.
  - Texto: "Arrastrá y soltá un PDF o seleccioná un archivo (máx. 10 MB)".
  - Botón "Seleccionar PDF" con `<input type="file" accept="application/pdf,.pdf" className="hidden">`.
  - Al seleccionar/dropear: validar en cliente (extensión, `file.type === 'application/pdf'` o extensión, tamaño ≤ 10 MB); mostrar nombre + tamaño + icono `pdf` + botón de quitar (`cancel`).
  - Botón "Subir contrato" deshabilitado sin archivo; spinner y texto "Subiendo..." durante el upload.
- **UI estado "con contrato"**:
  - Icono `pdf` grande, texto "Contrato cargado" y, si se implementa REQ-013, tamaño/fecha.
  - Botón primario "Abrir contrato" → `window.open(api.contracts.url(owner, entityId), '_blank', 'noopener,noreferrer')`.
  - Botón de peligro "Quitar contrato" → `mode='deleting'`, monta `DeleteModal` con `title`, `subtitle`, `onConfirm`, `onCancel`.
  - Confirmar → `api.contracts.remove(owner, entityId)`, toast de éxito, `localHasContract=false`, `onChanged()`, volver a `mode='empty'` o cerrar el modal.
- **Cierre**: click en el overlay y tecla `Escape` cierran el modal, salvo mientras `mode` es `uploading` o `deleting`. Botón `cancel` visible.
- **Accesibilidad**: `role="dialog"`, `aria-modal="true"`, `aria-labelledby`; el input de archivo accesible vía botón; foco inicial en el botón de seleccionar/abrir.

#### 4.2.6 Frontend — integración en páginas

- **`DeudasPage.tsx`**: agregar opción `{ label: t('contracts.menu'), icon: 'pdf', onClick: () => setContractTarget(debt) }` en `CardMenu`; renderizar `<ContractModal owner="debts" entityId={...} hasContract={...} ... />`. `Debt` debe exponer `has_contract`.
- **`DebtBillsPage.tsx`**: agregar la misma opción al `CreateMenu` del header; renderizar el modal con `debt.id`, `debt.description`, `debt.has_contract`.
- **`ServicesPage.tsx`**: agregar la opción en `CardMenu`; renderizar el modal con `svc.id`, `svc.name`, `svc.has_contract`.
- **`BillsPage.tsx`**: agregar la opción al `CreateMenu` del header; renderizar el modal con `service.id`, `service.name`, `service.has_contract`.
- **API client** (`frontend/src/api/index.ts`):
  ```ts
  contracts: {
    upload: (owner: 'debts' | 'services', id: number, file: File) =>
      postFormData<{ has_contract: boolean }>(`/api/${owner}/${id}/contract`, file),
    remove: (owner: 'debts' | 'services', id: number) =>
      del(`/api/${owner}/${id}/contract`),
    url: (owner: 'debts' | 'services', id: number) =>
      `/api/${owner}/${id}/contract`,
  },
  ```
- **Tipos** (`frontend/src/types/index.ts`): `has_contract?: boolean` en `Debt` y `Service`.
- **i18n**: claves bajo `"contracts"` en `es.json`/`en.json`, por ejemplo:
  ```json
  "contracts": {
    "menu": "Contrato",
    "title": "Contrato",
    "upload_title": "Subir contrato",
    "drop_hint": "Arrastrá y soltá un PDF o seleccioná un archivo",
    "only_pdf": "Solo se aceptan archivos PDF",
    "max_size": "Máximo 10 MB",
    "select_file": "Seleccionar PDF",
    "upload": "Subir contrato",
    "uploading": "Subiendo...",
    "uploaded_title": "Contrato cargado",
    "uploaded_hint": "Podés abrirlo o quitarlo.",
    "open": "Abrir contrato",
    "remove": "Quitar contrato",
    "remove_title": "¿Quitar el contrato?",
    "remove_subtitle": "El PDF se elimina del volumen y la acción no se puede deshacer. Escribí \"confirmo\" para continuar.",
    "upload_success": "Contrato subido",
    "remove_success": "Contrato eliminado",
    "upload_error": "No se pudo subir el contrato",
    "remove_error": "No se pudo quitar el contrato",
    "invalid_type": "El archivo debe ser un PDF",
    "empty_file": "El archivo está vacío",
    "file_too_large": "El archivo supera el máximo de 10 MB"
  }
  ```
  Ajustar las claves finales al estilo existente; el texto en inglés debe ser equivalente.

### 4.3 Modelo de datos

No hay cambios de esquema. Se reutilizan `debts` y `services`:

```
Entidad: Debt (tabla debts)
- id: INTEGER PK
- institution_id: INTEGER FK institutions
- description, identifier, total, principal, ...
- deleted_at: DATETIME

Entidad: Service (tabla services)
- id: INTEGER PK
- home_id: INTEGER FK homes
- name, institution, frequency, ...
- deleted_at: DATETIME
```

Campos calculados (no persistidos):

```
- has_contract: bool → os.Stat({DATA_DIR}/contracts/debts/{id}/contract.pdf)
- has_contract: bool → os.Stat({DATA_DIR}/contracts/services/{id}/contract.pdf)
```

Layout en disco:

```
/app/data/                              # volumen Docker p40la
├── app.db
├── app.db-wal
├── app.db-shm
├── documents/                         # SPEC-122 (comprobantes de pago)
│   └── {debt_id}/payment_{n}.pdf
└── contracts/                         # SPEC-123 (esta spec)
    ├── debts/
    │   ├── 1/contract.pdf             # deuda 1
    │   └── 2/contract.pdf             # deuda 2
    └── services/
        ├── 1/contract.pdf             # servicio 1
        └── 2/contract.pdf             # servicio 2
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/debts/{id}/contract` y `POST /api/services/{id}/contract`

**Request** `multipart/form-data`:
```
file: <PDF obligatorio, máx. 10 MB>
```

**Response 200**:
```json
{
  "has_contract": true
}
```

**Response Error**:
```json
{
  "error": "invalid_file",
  "message": "El archivo debe ser un PDF válido de máximo 10 MB"
}
```

Códigos:
- `400 invalid_id`: ID no numérico o ≤ 0.
- `400 invalid_file`: archivo faltante, vacío, no PDF o con firma inválida.
- `404 not_found`: la deuda/servicio no existe o está eliminado.
- `413 request_too_large`: body mayor al límite permitido.
- `500 internal_error`: error inesperado de filesystem.

#### Endpoint: `GET /api/debts/{id}/contract` y `GET /api/services/{id}/contract`

**Response 200**:
```
Content-Type: application/pdf
Content-Disposition: inline; filename="contract.pdf"
X-Content-Type-Options: nosniff
Cache-Control: private, no-store
<bytes del PDF>
```

**Response Error**:
```json
{
  "error": "not_found",
  "message": "El contrato no existe"
}
```

Códigos:
- `400 invalid_id`.
- `404 not_found`: entidad inexistente/eliminada o sin contrato.

#### Endpoint: `DELETE /api/debts/{id}/contract` y `DELETE /api/services/{id}/contract`

**Response 204**: sin body.

**Response Error**:
```json
{
  "error": "not_found",
  "message": "El contrato no existe"
}
```

Códigos:
- `400 invalid_id`.
- `404 not_found`.
- `500 internal_error`: error de filesystem (por ejemplo, permisos).

### 4.5 Dependencias

- **Internas**:
  - Backend: `internal/services/contract.go` (nuevo), `internal/services/debt.go`, `internal/services/service.go`, `internal/models/debt.go`, `internal/models/service.go`, `internal/api/debt_handlers.go`, `internal/api/service_handlers.go`, `internal/api/routes.go`, `cmd/server/main.go`.
  - Frontend: `frontend/src/components/ContractModal.tsx` (nuevo), `frontend/src/pages/DeudasPage.tsx`, `frontend/src/pages/DebtBillsPage.tsx`, `frontend/src/pages/ServicesPage.tsx`, `frontend/src/pages/BillsPage.tsx`, `frontend/src/api/index.ts`, `frontend/src/types/index.ts`, `frontend/public/i18n/es.json`, `frontend/public/i18n/en.json`.
  - Componentes reutilizados: `CardMenu`, `CreateMenu`, `DeleteModal`, `Icons`, `Toast`.
- **Externas**: ninguna nueva. Solo stdlib de Go (`bufio`, `errors`, `io`, `mime/multipart`, `net/http`, `os`, `path/filepath`, `strconv`, `strings`, `fmt`) y APIs nativas del navegador (`File`, `FormData`, `DragEvent`).

### 4.6 Manejo de errores y estados parciales

| Situación | Comportamiento |
|-----------|----------------|
| Archivo no PDF o > 10 MB | Rechazo antes de escribir; no se reemplaza el contrato existente. |
| Archivo vacío | `400 invalid_file`; no se escribe nada. |
| Falla la escritura del temporal | Se borra el temporal; se devuelve `500`; el contrato anterior queda intacto. |
| Falla el `os.Rename` | Se borra el temporal; se devuelve `500`; el contrato anterior queda intacto. |
| Se sube un contrato nuevo sobre uno existente | El temporal se renombra sobre `contract.pdf`; el anterior se reemplaza atómicamente. |
| El contrato no existe al abrir | `GET` → 404; el frontend muestra "sin contrato" y vuelve al estado de upload. |
| La entidad se elimina con el modal abierto | El `POST`/`GET`/`DELETE` devuelve 404; el frontend muestra toast de error y refresca la lista. |
| El archivo se borra manualmente del volumen | `has_contract=false`; `GET` → 404; no hay errores 500. |
| Dos uploads simultáneos de la misma entidad | Cada uno escribe su temporal y renombra; la última escritura gana; no se corrompe el archivo. |

---

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] **CA-001**: La opción **"Contrato"** aparece en el menú de 3 puntos de las tarjetas de deudas y servicios, y en el menú de acciones del detalle de deuda y del detalle de servicio.
- [ ] **CA-002**: Al abrir "Contrato" en una deuda/servicio **sin** contrato, se muestra un modal con drag & drop y botón "Seleccionar PDF"; se puede elegir un PDF, ver su nombre/tamaño, quitarlo antes de subir y subirlo.
- [ ] **CA-003**: Al subir un PDF válido, se crea `{DATA_DIR}/contracts/debts/{id}/contract.pdf` o `{DATA_DIR}/contracts/services/{id}/contract.pdf`; el directorio se crea si no existe; la respuesta es `200 {"has_contract": true}`.
- [ ] **CA-004**: El archivo se rechaza con mensaje claro si no es PDF, está vacío o supera 10 MB; en esos casos no se crea ni se reemplaza ningún archivo.
- [ ] **CA-005**: Al abrir "Contrato" en una entidad **con** contrato, el modal muestra "Abrir contrato" y "Quitar contrato"; "Abrir" abre el PDF en una pestaña nueva.
- [ ] **CA-006**: "Quitar contrato" abre el modal de confirmación que exige escribir `confirmo`; al confirmar se elimina el archivo, se muestra un toast de éxito y `has_contract` pasa a `false`.
- [ ] **CA-007**: Cancelar el modal de confirmación **no** elimina el archivo y vuelve al estado "con contrato".
- [ ] **CA-008**: `GET /api/debts/{id}/contract` y `GET /api/services/{id}/contract` devuelven `application/pdf` con `X-Content-Type-Options: nosniff`; sin archivo devuelven 404.
- [ ] **CA-009**: `DELETE` devuelve `204` y el archivo deja de existir en disco; una segunda llamada devuelve `404`.
- [ ] **CA-010**: `has_contract` aparece en `GET /api/debts`, `GET /api/debts/{id}`, `GET /api/services`, `GET /api/services/{id}` y refleja la existencia real del archivo.
- [ ] **CA-011**: No se agregó ninguna migración ni columna a `debts`/`services`; `PRAGMA table_info(debts)` y `PRAGMA table_info(services)` no cambian.
- [ ] **CA-012**: Los endpoints están protegidos por sesión; sin login devuelven 401 y no sirven archivos.
- [ ] **CA-013**: No hay listado estático ni acceso directo por URL a `contracts/`; el único acceso es por los endpoints autenticados.
- [ ] **CA-014**: Si el archivo se borra manualmente del volumen, `has_contract` pasa a `false` y el `GET` devuelve 404 (sin errores 500).
- [ ] **CA-015**: Si la deuda/servicio no existe o está eliminado, los tres endpoints devuelven 404 y no crean directorios.
- [ ] **CA-016**: La subida reemplaza un contrato existente sin dejar temporales `.tmp` visibles.
- [ ] **CA-DARK**: El área de drag & drop, el nombre del archivo, los textos de ayuda y los botones usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificaron en darkmode (texto y placeholders legibles).
- [ ] **CA-I18N**: Las claves nuevas existen en `frontend/public/i18n/es.json` y `en.json`, y `npm run build` las copia a `public/i18n/`.
- [ ] **CA-BUILD**: `go test ./...`, `go vet ./...` y `npm run build` (que ejecuta `check-ui-rules.cjs`) pasan sin errores.
- [ ] **CA-BACK**: No aplica: la funcionalidad es completamente modal, no crea páginas de detalle ni formularios con lista padre; no se modifica `BACK_ROUTES`.

### 5.2 No funcionales

- [ ] **CA-NF-001**: El upload de un PDF de 10 MB no supera ~2 MB de RAM adicional en el server (streaming a disco).
- [ ] **CA-NF-002**: El listado de 100 deudas/servicios con `os.Stat` por entidad responde en < 300 ms en el iHost.
- [ ] **CA-NF-003**: El PDF se escribe de forma atómica: no quedan archivos `.tmp` tras un upload exitoso ni tras un error.
- [ ] **CA-NF-004**: El endpoint de descarga envía `X-Content-Type-Options: nosniff` y solo sirve archivos con firma `%PDF-`.
- [ ] **CA-NF-005**: La ruta final no contiene ningún componente provisto por el usuario; `owner` es whitelist e `id` se formatea como entero.
- [ ] **CA-NF-006**: `contracts/` está dentro de `DATA_DIR`, sobrevive a un reinicio del contenedor y no requiere un volumen nuevo.

### 5.3 Testing

- **Unit tests** (`internal/services/contract_test.go`):
  - `NewContractStore(t.TempDir())` genera rutas `contracts/debts/1/contract.pdf` y `contracts/services/1/contract.pdf`.
  - `Save` con PDF sintético válido crea el archivo y `Exists` devuelve `true`.
  - `Save` con archivo vacío, sin extensión `.pdf`, con firma inválida o > 10 MB devuelve el error correspondiente y no deja archivos.
  - Reemplazo: un segundo `Save` sobrescribe el archivo sin dejar `.tmp`.
  - `Delete` elimina el archivo; una segunda llamada devuelve `ErrContractNotFound`.
  - `Path` devuelve `ErrContractNotFound` si no existe.
  - Owner inválido devuelve error y no crea rutas fuera de `contracts/`.
- **Service tests** (`internal/services/debt_contract_test.go`, `internal/services/service_contract_test.go`):
  - `has_contract` es `false` sin archivo y `true` después de `SaveContract`.
  - `SaveContract`/`ContractPath`/`DeleteContract` devuelven `ErrContractEntityNotFound` si la entidad no existe.
  - El listado completa `has_contract` para cada entidad.
- **Integration tests** (`internal/api/contract_handlers_test.go`):
  - `POST` multipart con PDF válido → 200, archivo en `t.TempDir()`, `has_contract=true`.
  - `POST` sin archivo / no PDF / > 10 MB → 400/413.
  - `GET` con archivo → 200, `Content-Type: application/pdf`, `nosniff`, bytes correctos.
  - `GET` sin archivo → 404.
  - `DELETE` → 204; segunda llamada → 404.
  - Endpoints sin sesión → 401.
  - Usar `t.TempDir()` como `DATA_DIR` y PDFs sintéticos generados en el test (nunca archivos reales).
- **Frontend**:
  - `npm run check:ui` (no `<select>` nativo).
  - `npm run build`.
  - Prueba manual en local: subir contrato en deuda y servicio, abrirlo, recargar la página, quitarlo escribiendo `confirmo`, cancelar la confirmación, verificar darkmode y móvil.
- **iHost**:
  - Tras el deploy, subir un PDF sintético, verificar la ruta con SSH (`/app/data/contracts/...`), reiniciar el contenedor y confirmar que el contrato sigue disponible.
  - Verificar que el respaldo del volumen incluya `contracts/` (ver REQ-010).
  - No usar contratos reales en pruebas ni en el repo.

---

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: `ContractStore`, validación PDF, escritura atómica, tests unitarios | 0.5 día | Ninguna |
| 2 | Backend: `has_contract` en modelos/services, setters, métodos de entidad y tests de service | 0.5 día | Fase 1 |
| 3 | API: handlers, rutas, límites, tests de integración | 0.5 día | Fase 2 |
| 4 | Frontend: `ContractModal`, API client, tipos, i18n y conexión en las 4 páginas | 0.5 día | Fase 3 |
| 5 | Validación local, build, pruebas manuales, darkmode y documentación de respaldo | 0.25 día | Fases 3-4 |

**Total estimado**: ~2 días de desarrollo + pruebas.

### 6.2 Milestones

1. **MVP**: opción "Contrato" en las 4 ubicaciones; modal con upload drag & drop; almacenamiento en `{DATA_DIR}/contracts/...`; abrir y quitar con confirmación `confirmo`; `has_contract` en list/get.
2. **V1.0**: indicador PDF en tarjetas (P1), `?download=1` para móvil, limpieza de huérfanos y metadatos del contrato si se implementan; documentación de respaldo del volumen; verificación en iHost.

---

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El respaldo actual solo copia `app.db` y se pierden los contratos | Alta | Alto | Documentar y actualizar el procedimiento de backup para incluir `contracts/` y `documents/` (REQ-010); verificar en el release. |
| Crecimiento de disco por PDFs de 10 MB acumulados | Media | Medio | Límite de 10 MB, un archivo por entidad; monitorear el volumen; limpieza de huérfanos como P2. |
| Archivos huérfanos al eliminar deudas/servicios (soft delete) | Media | Bajo | Aceptado en MVP; documentar limpieza P1/P2; los archivos no se sirven si la entidad no existe. |
| `os.Stat` por entidad degrada listados grandes | Baja | Bajo | N acotado; si crece, cachear por request o listar el directorio una vez por owner. |
| Carrera entre dos uploads simultáneos de la misma entidad | Baja | Bajo | `os.Rename` atómico; la última escritura gana; no se corrompe el archivo. |
| PDF malicioso o con contenido activo | Baja | Medio | Validar firma `%PDF-`, `Content-Type: application/pdf`, `nosniff`, solo servir por endpoint autenticado; no ejecutar el archivo en el server. |
| Safari/iOS no previsualiza PDF inline | Media | Bajo | P1: `?download=1` para forzar descarga; en el MVP se abre en pestaña nueva. |
| `DATA_DIR` mal configurado en desarrollo (default `./data`) | Baja | Bajo | Usar `t.TempDir()` en tests; documentar que en Docker es `/app/data`; no crear directorios en startup. |
| Error de permisos en el volumen al escribir | Baja | Alto | El contenedor corre como root; verificar en iHost con un PDF sintético antes del release; `MkdirAll` con 0755. |
| Cambios en `NewDebtService`/`NewServiceService` rompen tests | Media | Medio | Usar setter injection (`SetContractStore`), no cambiar las firmas existentes; los tests sin store siguen funcionando con `has_contract=false`. |
| SPEC-122 todavía está `in_progress` y su ruta puede cambiar | Media | Bajo | Esta spec solo reutiliza la raíz `DATA_DIR` y el patrón `documents/`; si SPEC-122 cambia la convención, revisar `contracts/` antes del release. La implementación de contratos no depende del código de comprobantes. |
| El modal muestra un estado viejo si la lista no se refresca | Media | Bajo | `onChanged` recarga la lista/detalle después de subir o quitar; el modal mantiene un `localHasContract` para feedback inmediato. |
| Texto `confirmo` hardcodeado en `DeleteModal` | Baja | Bajo | Se reutiliza el componente existente tal cual; si se internacionaliza en el futuro, esta spec se beneficia sin cambios. |

---

## 8. Notas y Referencias

- **SPEC-122**: Comprobantes PDF en pagos de deudas; define `{DATA_DIR}/documents/{debt_id}/payment_{n}.pdf`, validación de 10 MB y `has_receipt` calculado del filesystem. Es la spec más reciente y la referencia de ruta de volumen para esta spec.
- **SPEC-051**: Comprobantes de pensión en `{DATA_DIR}/uploads/payment-proofs` (patrón `SaveProof`/`ProofPath`, `UploadProof`/`DownloadProof`).
- **SPEC-043**: Acción Pagar en facturas con fecha de pago, comprobante (link de Drive) y referencia; no es un precedente de upload de archivos.
- **SPEC-054**: Módulo Deudas, modelo `debts`/`debt_bills` y acción Pagar.
- **SPEC-066**: Worktrees obligatorios; esta spec se crea en `feature/SPEC-123`.
- **SPEC-060 / SPEC-118**: Reglas de UI, tokens del tema y guardrail de dropdowns (`frontend/scripts/check-ui-rules.cjs`).
- **Docker**: `Dockerfile` (`ENV DATA_DIR=/app/data`, `VOLUME ["/app/data"]`) y `docker-compose.yml` (volumen externo `p40la` montado en `/app/data`).
- **Config**: `internal/config/config.go` (`DATA_DIR`, default `./data`).
- **UI reutilizada**:
  - `frontend/src/components/CardMenu.tsx` (menú de 3 puntos en tarjetas).
  - `frontend/src/components/CreateMenu.tsx` (menú de acciones en headers).
  - `frontend/src/components/DeleteModal.tsx` (confirmación escribiendo `confirmo`).
  - `frontend/src/components/UploadBillModal.tsx` (patrón de drag & drop).
  - `frontend/src/components/Icons.tsx` (icono `pdf`).
- **Modelos y services**: `internal/models/debt.go`, `internal/models/service.go`, `internal/services/debt.go`, `internal/services/service.go`.
- **Rutas**: `internal/api/routes.go` (deudas en la sección de deudas, servicios en la sección de servicios).
- **Datos sintéticos**: los ejemplos usan `debt_id=1`, `service_id=1`, `contract.pdf`, `demo-reference-001`. Nunca usar contratos reales en tests, specs ni commits.

### Estado de implementación (2026-11-07)

- **P0 implementado**: `ContractStore`, validación PDF/10 MB, escritura atómica, endpoints `POST`/`GET`/`DELETE`, `has_contract` en deudas y servicios, `ContractModal` con drag & drop/abrir/quitar, confirmación con `confirmo`, integración en tarjetas y detalles de deudas/servicios, i18n es/en y tests unitarios/de handlers.
- **P1 pendiente**: indicador PDF en tarjetas, metadatos del contrato (tamaño/fecha), `?download=1` desde la UI y limpieza de huérfanos al eliminar entidades.
- **Validación local**: `go test ./...`, `go vet ./...` y `npm run build` pasan; flujo de upload/descarga/eliminación probado contra un `DATA_DIR` temporal con datos sintéticos.

### Fuera de alcance

- OCR o extracción de datos del contrato.
- Múltiples contratos por entidad o anexos.
- Firma digital o validación de autenticidad.
- Sincronización con Google Drive u otro almacenamiento externo.
- Contratos para facturas de servicios, autos, pólizas u otras entidades.
- Limpieza automática de archivos huérfanos en el MVP.
- Edición o anotación del PDF.

---

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-07 | opencode | Creación inicial de la especificación (SPEC-123). |
| 2026-11-07 | opencode | Inicio de implementación; estado `in_progress`. |
| 2026-11-07 | opencode | Implementación P0: almacén de contratos, endpoints, `has_contract`, modal, integración en 4 páginas, i18n y tests. |
