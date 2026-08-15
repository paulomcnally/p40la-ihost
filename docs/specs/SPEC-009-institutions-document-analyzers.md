---
title: "Módulo de Instituciones, Analizadores de Documentos y Extracción Automática de Facturas"
id: "SPEC-009"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-15"
updated: "2026-08-15"
github_issue: 10
---

# Módulo de Instituciones, Analizadores de Documentos y Extracción Automática de Facturas

**ID**: SPEC-009  
**Estado**: draft  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-15  
**Actualizado**: 2026-08-15

---

## 1. Resumen Ejecutivo

Esta especificación define un sistema completo de gestión de instituciones y análisis automático de documentos de facturación. Cada institución (ej: Claro, Movistar) representa un proveedor de servicios que genera facturas en PDF o imagen. Las instituciones pueden tener múltiples analizadores asociados, y cada servicio selecciona qué analizador usar para su institución.

El flujo de uso es: desde la sección de facturación de un servicio, el usuario sube un documento PDF/imagen, el sistema identifica la institución y el analizador asociado al servicio, ejecuta el analizador correspondiente, extrae los datos (monto, número, período) y crea la factura automáticamente tras un paso de review.

Los analizadores viven dentro del mismo proyecto como paquetes Go en `internal/analyzers/`, registrándose mediante un patrón de registry en `init()`. Esto evita procesos externos, consume mínima memoria, y es completamente nativo para iHost.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: CRUD completo de instituciones con nombre.
2. **REQ-002**: Menú lateral con sección "Instituciones" (similar a "Servicios").
3. **REQ-003**: Cada institución puede tener N analizadores asociados (relación N:M).
4. **REQ-004**: Cada servicio se asocia a una institución y selecciona cuál analizador usar de los disponibles.
5. **REQ-005**: Interfaz Go común (`DocumentAnalyzer`) que todos los analizadores deben implementar.
6. **REQ-006**: Sistema de registry para que los analizadores se registren automáticamente al estar en el proyecto.
7. **REQ-007**: Endpoint de subida de documentos (PDF/imagen) desde la sección de facturación de un servicio.
8. **REQ-008**: Al subir un documento, el sistema ejecuta el analizador configurado del servicio y extrae los datos de la factura.
9. **REQ-009**: Los datos extraídos se presentan para review antes de crear la factura (edición del monto permitida).
10. **REQ-010**: Si un servicio no tiene institución o analizador configurado, se bloquea el upload con mensaje claro.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

11. **REQ-011**: Soporte para formatos PDF, PNG, JPG como documentos de entrada.
12. **REQ-012**: Timeout configurable para el análisis de documentos (evitar bloqueo del servidor).
13. **REQ-013**: Log de análisis con resultado (éxito/error) y datos extraídos.
14. **REQ-014**: Reemplazar el campo libre `institution TEXT` en services por FK `institution_id`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

15. **REQ-015**: Preview del documento subido antes de procesar.
16. **REQ-016**: Historial de documentos procesados por institución.
17. **REQ-017**: Indicador visual en la UI si un servicio tiene institución y analizador configurados.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Procesamiento de documento en < 10 segundos. Timeout por defecto: 30s.
- **Almacenamiento**: Documentos no se persisten, solo datos extraídos. Archivos temporales se eliminan tras procesar.
- **iHost**: Analizadores como paquetes Go nativos dentro del proyecto. Sin procesos externos. Procesamiento síncrono con timeout.
- **Memoria**: Cada análisis libera memoria al completar. Sin caché permanente de documentos.
- **Seguridad**: Solo usuarios autenticados pueden subir documentos y gestionar instituciones. Validación de tipo de archivo (solo PDF, PNG, JPG). Tamaño máximo: 10MB.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- El campo `institution TEXT` en services es libre, no referencial. Debe convertirse en FK.
- Go permite registrar paquetes mediante `init()` y un patrón de registry map.
- Los analizadores viven dentro del mismo proyecto (`internal/analyzers/`), sin repos externos.
- El proyecto ya tiene migraciones con `golang-migrate` y estructura de storage layer.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Analizadores como Go packages internos | Nativo, sin overhead, type-safe, ideal para iHost | Requiere rebuild al agregar nuevo analizador | ✅ Seleccionada |
| Analizadores como binarios externos | Dinámico sin rebuild | Procesos separados, RAM extra, complejidad, seguridad | ❌ Rechazada |
| Analizadores como plugins Go | Hot-reload | No soportado en GOOS=linux sin CGO, inestable | ❌ Rechazada |
| Analizadores como WASM | Aislamiento | Binarios grandes, toolchain compleja, overkill | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Registry pattern para analizadores con `init()`
- **Contexto**: Necesidad de que cada analizador se registre automáticamente al estar en el proyecto.
- **Decisión**: Crear `internal/analyzers/registry.go` con un map global. Cada paquete de analizador llama `registry.Register(analyzer)` en su `init()`. El registry expone `List()` para obtener todos los analizadores disponibles.
- **Consecuencias**: Simple, nativo, sin dependencias. Los analizadores se importan en `internal/analyzers/all/all.go`.

**ADR-002**: Interfaz `DocumentAnalyzer` unificada
- **Contexto**: Todos los analizadores deben seguir el mismo contrato.
- **Decisión**: Definir interfaz con `Info() AnalyzerInfo` y `Analyze(reader io.Reader, mimeType string) (*ExtractedBill, error)`. `AnalyzerInfo` incluye solo ID y nombre.
- **Consecuencias**: Cada analizador implementa su lógica específica pero retorna el mismo formato. Fácil de testear con mocks.

**ADR-003**: Relación N:M entre instituciones y analizadores
- **Contexto**: Una institución puede tener múltiples analizadores (ej: Claro Facturas Mensuales, Claro Internet). Un servicio elige cuál usar.
- **Decisión**: Tabla intermedia `institution_analyzers` con `institution_id` + `analyzer_id`. Services tiene FK a `institution_id` y a `institution_analyzer_id`.
- **Consecuencias**: Permite flexibilidad. Un servicio puede cambiar de analyzer sin cambiar de institución.

**ADR-004**: Documentos no se persisten, solo datos extraídos
- **Contexto**: iHost tiene almacenamiento limitado.
- **Decisión**: Documentos se suben a archivo temporal, se procesan, se extraen datos, se eliminan. Solo la factura creada persiste en SQLite.
- **Consecuencias**: Bajo consumo de disco.

**ADR-005**: Upload bloqueado sin analyzer configurado
- **Contexto**: No tiene sentido permitir upload si no hay analyzer para procesar.
- **Decisión**: Validar en el servicio que `institution_id` y `institution_analyzer_id` estén configurados antes de permitir upload.
- **Consecuencias**: UX clara: el usuario debe configurar la institución y el analyzer antes de poder subir documentos.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Frontend: Service Billing Section]
         |
         | (upload PDF/imagen)
         v
[internal/api/document.go] --(multipart form)--> [internal/services/document.go]
                                                          |
                                                          | (lookup service.institution_analyzer_id)
                                                          v
                                              [internal/analyzers/registry.go]
                                                          |
                                                          | (get analyzer by ID)
                                                          v
                                              [internal/analyzers/claro/]
                                              [internal/analyzers/movistar/]
                                                          |
                                                          v
                                              ExtractedBill{amount, invoice_number, year, month}
                                                          |
                                                          v
                                              [internal/storage/bill.go] --(SQLite)--> [bills table]
```

### 4.2 Componentes

#### 4.2.1 Modelo Institution (nuevo)
- **Responsabilidad**: Representar una institución/proveedor de servicios.
- **Ubicación**: `internal/models/institution.go`
- **Campos**:
  - `ID int64`
  - `Name string`
  - `CreatedAt time.Time`
  - `UpdatedAt time.Time`

#### 4.2.2 Modelo Service (actualizado)
- **Nuevos campos**:
  - `InstitutionID *int64` (FK → institutions.id, nullable)
  - `InstitutionAnalyzerID *int64` (FK → institution_analyzers.id, nullable)
- **Campo existente**: `Institution string` se depreca pero se mantiene temporalmente.

#### 4.2.3 Interfaz DocumentAnalyzer
- **Responsabilidad**: Contrato que todos los analizadores deben implementar.
- **Ubicación**: `internal/analyzers/analyzer.go`
- **Definición**:
```go
type AnalyzerInfo struct {
    ID   string
    Name string
}

type ExtractedBill struct {
    Amount        float64
    InvoiceNumber string
    Year          int
    Month         int
    DueDate       *time.Time
    RawData       map[string]interface{}
}

type DocumentAnalyzer interface {
    Info() AnalyzerInfo
    Analyze(reader io.Reader, mimeType string) (*ExtractedBill, error)
}
```

#### 4.2.4 Registry de Analizadores
- **Responsabilidad**: Mantener registro de analizadores disponibles.
- **Ubicación**: `internal/analyzers/registry.go`
- **Funciones**:
  - `Register(analyzer DocumentAnalyzer)` - Registra un analizador
  - `Get(id string) (DocumentAnalyzer, bool)` - Obtiene por ID
  - `List() []AnalyzerInfo` - Lista todos los analizadores registrados

#### 4.2.5 Analizador de ejemplo (Claro)
- **Responsabilidad**: Analizar facturas de Claro.
- **Ubicación**: `internal/analyzers/claro/analyzer.go`
- **Registro**: En `init()`, llama `registry.Register(&ClaroAnalyzer{})`

#### 4.2.6 Importador de todos los analizadores
- **Responsabilidad**: Importar todos los analizadores para que se registren.
- **Ubicación**: `internal/analyzers/all/all.go`

#### 4.2.7 Servicio de Documentos
- **Responsabilidad**: Orquestar subida → análisis → extracción.
- **Ubicación**: `internal/services/document.go`
- **Funciones**:
  - `UploadAndAnalyze(serviceID int64, file multipart.File, header *multipart.FileHeader) (*ExtractedBill, error)`
  - `CreateBillFromExtracted(serviceID int64, extracted *ExtractedBill) (*models.Bill, error)`

#### 4.2.8 Servicio de Instituciones
- **Responsabilidad**: CRUD de instituciones y gestión de analyzers asociados.
- **Ubicación**: `internal/services/institution.go`

### 4.3 Modelo de datos

```
Entidad: institutions (nueva)
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- name: TEXT NOT NULL UNIQUE
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP

Entidad: institution_analyzers (nueva, N:M)
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- institution_id: INTEGER NOT NULL (FK → institutions.id)
- analyzer_id: TEXT NOT NULL (ID del analizador en el registry)
- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- UNIQUE(institution_id, analyzer_id)

Entidad: services (actualizada)
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- home_id: INTEGER NOT NULL
- name: TEXT NOT NULL
- institution: TEXT (DEPRECATED)
- institution_id: INTEGER NULL (FK → institutions.id)
- institution_analyzer_id: INTEGER NULL (FK → institution_analyzers.id)
- currency_id: INTEGER NOT NULL
- frequency: TEXT NOT NULL
- suggested_amount: REAL NOT NULL
- active: BOOLEAN DEFAULT 1
- icon_key: TEXT NOT NULL
- deleted_at: DATETIME
- created_at: DATETIME
- updated_at: DATETIME

Entidad: bills (sin cambios)
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- service_id: INTEGER NOT NULL
- year: INTEGER NOT NULL
- month: INTEGER NOT NULL
- amount: REAL NOT NULL
- invoice_number: TEXT
- status: TEXT DEFAULT 'pending'
- drive_url: TEXT
- deleted_at: DATETIME
- created_at: DATETIME
- updated_at: DATETIME
- UNIQUE(service_id, year, month)
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/services/:id/bills/upload`

**Request**: Multipart form con archivo

**Response 200** (datos extraídos para review):
```json
{
  "extracted": {
    "amount": 59.90,
    "invoice_number": "FAC-2026-001234",
    "year": 2026,
    "month": 8,
    "due_date": "2026-09-15T00:00:00Z",
    "raw_data": {}
  },
  "analyzer_used": "claro"
}
```

#### Endpoint: `POST /api/services/:id/bills/from-extracted`

**Request**:
```json
{
  "amount": 59.90,
  "invoice_number": "FAC-2026-001234",
  "year": 2026,
  "month": 8
}
```

**Response 201**: Bill creada

#### Endpoint: `POST /api/institutions`

**Request**:
```json
{
  "name": "Claro"
}
```

**Response 201**: Institución creada

#### Endpoint: `PUT /api/institutions/:id/analyzers`

**Request**:
```json
{
  "analyzer_ids": ["claro", "claro-internet"]
}
```

**Response 200**: Analyzers asociados actualizados

#### Endpoint: `GET /api/analyzers`

**Response 200**:
```json
[
  {"id": "claro", "name": "Claro Facturas"},
  {"id": "movistar", "name": "Movistar Facturas"}
]
```

#### Endpoint: `GET /api/services/:id/analyzer-options`

**Response 200**:
```json
[
  {"id": 1, "institution_id": 1, "analyzer_id": "claro", "analyzer_name": "Claro Facturas"},
  {"id": 2, "institution_id": 1, "analyzer_id": "claro-internet", "analyzer_name": "Claro Internet"}
]
```

### 4.5 Dependencias

- **Internas**:
  - `internal/models/institution.go` (nuevo)
  - `internal/models/service.go` (actualizado)
  - `internal/analyzers/analyzer.go` (nuevo)
  - `internal/analyzers/registry.go` (nuevo)
  - `internal/analyzers/claro/analyzer.go` (nuevo, ejemplo)
  - `internal/analyzers/all/all.go` (nuevo)
  - `internal/services/document.go` (nuevo)
  - `internal/services/institution.go` (nuevo)
  - `internal/storage/institution.go` (nuevo)
  - `internal/api/document.go` (nuevo)
  - `internal/api/institution.go` (nuevo)
  - `internal/api/service.go` (actualizado)
  - `frontend/src/pages/InstitutionsPage.tsx` (nuevo)
  - `frontend/src/components/DocumentUpload.tsx` (nuevo)
  - `frontend/src/pages/ServiceFormPage.tsx` (actualizado)
- **Externas**:
  - `mime/multipart`, `io`, `context` (Go stdlib)

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: CRUD completo de instituciones desde el menú lateral.
- [ ] CA-002: Al crear/editar institución, se pueden seleccionar múltiples analizadores del registry.
- [ ] CA-003: Al crear/editar servicio, se selecciona institución y luego el analyzer disponible.
- [ ] CA-004: Si institución tiene 1 analyzer, se auto-selecciona en el servicio.
- [ ] CA-005: Si institución tiene 0 analyzers, se muestra mensaje y no permite seleccionar.
- [ ] CA-006: Upload bloqueado si servicio no tiene institution_id o institution_analyzer_id.
- [ ] CA-007: Upload de PDF/imagen → análisis → datos extraídos para review → edición permitida → crear factura.
- [ ] CA-008: Analyzer que falla retorna error descriptivo sin crash.
- [ ] CA-009: Archivos no soportados o >10MB retornan error de validación.
- [ ] CA-010: `GET /api/analyzers` retorna lista del registry.

### 5.2 No funcionales

- [ ] CA-NF-001: Análisis completa en < 10 segundos.
- [ ] CA-NF-002: Timeout de 30s cancela el análisis.
- [ ] CA-NF-003: Archivos temporales se eliminan tras procesar.
- [ ] CA-NF-004: Registry consume < 1MB de memoria.
- [ ] CA-NF-005: Migración se ejecuta en < 1 segundo.

### 5.3 Testing

- **Unit tests**: Registry, mock analyzer, validación de archivos, servicio document.
- **Integration tests**: CRUD instituciones, flujo upload→analyze→create bill, timeout, error de analyzer.
- **E2E tests**: Crear institución → asociar analyzers → crear servicio con analyzer → subir PDF → review → crear factura.
- **Carga/Performance**: Procesamiento no bloquea otras requests en iHost.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración DB: tabla `institutions`, `institution_analyzers`, agregar FKs a services | 0.5 días | Ninguna |
| 2 | Modelos Institution, Service actualizado | 0.5 días | Fase 1 |
| 3 | Interfaz DocumentAnalyzer + registry | 0.5 días | Ninguna |
| 4 | Analyzer ejemplo (Claro) + all/all.go | 1 día | Fase 3 |
| 5 | Storage layer para instituciones | 0.5 días | Fase 2 |
| 6 | Servicio de documentos (upload + analyze) | 1 día | Fases 2, 3, 5 |
| 7 | API endpoints (instituciones, documentos, analyzers) | 1 día | Fases 4, 5, 6 |
| 8 | InstitutionsPage.tsx (CRUD + gestión de analyzers) | 1 día | Fase 7 |
| 9 | ServiceFormPage actualizado (selector institución + analyzer) | 0.5 días | Fase 7 |
| 10 | DocumentUpload.tsx + flujo de review | 1 día | Fase 7 |
| 11 | Integrar analyzers en main.go | 0.5 días | Fase 4 |
| 12 | Tests unitarios e integración | 1.5 días | Fases 1-11 |
| 13 | Validación manual en local | 0.5 días | Fase 12 |

### 6.2 Milestones

1. **MVP**: Modelo instituciones + registry + analyzer ejemplo + endpoints (Fases 1-7).
2. **V1.0**: UI completa + upload/analyze/review + tests + validación iHost (Fases 8-13).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Analyzer falla o retorna datos incorrectos | Media | Alto | Review obligatorio antes de crear factura. Log de errores. |
| Timeout bloquea servidor | Baja | Alto | Context cancellation a 30s. |
| Upload sin analyzer configurado | Media | Medio | Validación en API + mensaje claro en UI. |
| Migración de `institution TEXT` a FK | Media | Medio | Mantener campo TEXT temporalmente. |
| PDFs grandes consumen memoria | Media | Medio | Límite 10MB. Temp files se eliminan. |

## 8. Notas y Referencias

- SPEC-004: Dashboard con módulo de servicios y facturas.
- SPEC-008: Sistema de facturación automática.
- Go `init()` pattern: https://go.dev/doc/effective_go#init
- Go `mime/multipart`: https://pkg.go.dev/mime/multipart

### Proceso para agregar un nuevo analyzer

1. Crear carpeta `internal/analyzers/nuevo-analyzer/analyzer.go`
2. Implementar interfaz `DocumentAnalyzer`
3. Registrar en `init()` con `registry.Register(&NuevoAnalyzer{})`
4. Agregar import en `internal/analyzers/all/all.go`
5. Rebuild y deploy desde UI de iHost

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-15 | p40la-ihost-team | Creación inicial de la especificación |

## 10. Lecciones Aprendidas

### Errores cometidos durante la implementación

1. **NUNCA ejecutar `rm -f data/app.db`**: Se borró la base de datos local del usuario al probar migraciones. La DB local contiene datos de producción del usuario. Las migraciones deben probarse contra la DB existente o en un archivo temporal separado (`/tmp/test-app.db`).

2. **Seguir el patrón de UI existente**: InstitutionsPage se creó inicialmente con un formulario inline en lugar del patrón card + EmptyCard. El patrón correcto es: listado en cards con menú de acciones (3 puntos), y cuando no hay registros mostrar EmptyCard con título, descripción y botón que navega al formulario. Referencia: `HomesPage.tsx`.

3. **Validación de prerequisitos**: No se validó que existan instituciones antes de permitir crear servicios. Se debe replicar el patrón de validación de homes: backend check en CreateService + frontend redirect.
