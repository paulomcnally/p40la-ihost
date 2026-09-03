---
title: "Backend de Registros Mensuales de Pensión Alimenticia (support_records, salary_payments, month_closings)"
id: "SPEC-049"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-02"
updated: "2026-09-03"
github_issue: 49
---

# Backend de Registros Mensuales de Pensión Alimenticia

**ID**: SPEC-049  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-02  
**Actualizado**: 2026-09-03

---

## 1. Resumen Ejecutivo

El módulo de **Pensión Alimenticia** de p40la-ihost tiene hasta hoy los CRUD base: Hijos (SPEC-045), Notificaciones/destinatarios (SPEC-046), Salarios (SPEC-047) y Categorías (SPEC-048). La sección **Registros mensuales** (`/pension/registros`) es todavía un placeholder (SPEC-044). Esta spec es la **primera de tres** (SPEC-049 → 050 → 051) que replican la página `child-support/records` del proyecto P4OLA (una app NestJS+React que gestiona pensión alimenticia mensual), adaptándola a las guías de estilo y arquitectura de p40la-ihost (Go + SQLite + React/Vite, iHost de recursos limitados).

Esta spec cubre **exclusivamente el backend de datos y APIs** necesario para la página de registros: las tablas `support_records` (registros de manutención por hijo/categoría/mes), `salary_payments` (pagos de salario del mes) y `month_closings` (cierre/reapertura de mes), con sus modelos, capa de storage, servicios de negocio y handlers HTTP. Incluye además el **almacenamiento y descarga de comprobantes** (proof) asociados a un registro pagado, sin análisis IA (decisión tomada con el usuario: el análisis IA de P4OLA se omite; el comprobante se sube como adjunto y el usuario llena los campos manualmente).

El frontend de la página de registros se desarrolla en **SPEC-050** y las notificaciones por email + generación mensual en **SPEC-051**. Esta spec deja la API lista y verificable por `curl`, sin UI.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Migración SQLite `0020_create_support_records` (up/down) con tabla `support_records`: id, child_id (FK→children.id), pension_category_id (FK→pension_categories.id), year, month, amount (REAL), currency (TEXT 3), status (TEXT: pending|paid|rejected), paid_at, payment_method (TEXT NULL), payment_reference (TEXT NULL), evidence_notes (TEXT NULL), notes (TEXT NULL), proof_file_name (TEXT NULL), original_amount (REAL NULL), original_currency (TEXT 3 NULL), exchange_rate (REAL NULL), created_at, updated_at. **UNIQUE (child_id, pension_category_id, year, month)**.
2. **REQ-002**: Migración SQLite `0021_create_salary_payments` (up/down) con tabla `salary_payments`: id, salary_id (FK→salaries.id), year, month, amount (REAL), currency (TEXT 3), status (TEXT: pending|received), received_amount (REAL NULL), received_at, notes (TEXT NULL), created_at, updated_at. **UNIQUE (salary_id, year, month)**.
3. **REQ-003**: Migración SQLite `0022_create_month_closings` (up/down) con tabla `month_closings`: id, year, month, closed_at. **UNIQUE (year, month)**.
4. **REQ-004**: Modelos Go (`internal/models/`): `SupportRecord`, `SalaryPayment`, `MonthClosing` con JSON tags consistentes con el resto del proyecto.
5. **REQ-005**: Capa storage (`internal/storage/`): `support_record.go`, `salary_payment.go`, `month_closing.go` con queries SQL crudo.
6. **REQ-006**: Capa services (`internal/services/`): validaciones de negocio:
   - Crear/editar registro verifica que el mes no esté cerrado.
   - `mark-paid` guarda paid_at (default ahora), payment_method, payment_reference, evidence_notes, original_amount/currency/exchange_rate.
   - `mark-pending` limpia los campos de pago.
   - `mark-rejected` guarda el motivo en `notes`.
   - `mark-received` (salary) requiere received_at; guarda received_amount y notes.
   - Cerrar mes: crea `month_closings` (error si ya existe); reabrir: elimina (error si no existe).
7. **REQ-007**: Handlers HTTP + rutas en `internal/api/routes.go` (protegidas por auth):
   - `GET /api/pension/records?year&month&child_id`
   - `POST /api/pension/records`
   - `PUT /api/pension/records/{id}`
   - `POST /api/pension/records/{id}/mark-paid`
   - `POST /api/pension/records/{id}/mark-pending`
   - `POST /api/pension/records/{id}/mark-rejected`
   - `GET /api/pension/salary-payments?year&month`
   - `POST /api/pension/salary-payments/{id}/mark-received`
   - `POST /api/pension/salary-payments/{id}/mark-pending`
   - `GET /api/pension/closing/{year}/{month}`
   - `POST /api/pension/closing/{year}/{month}`
   - `DELETE /api/pension/closing/{year}/{month}`
8. **REQ-008**: Subida de comprobante (`POST /api/pension/records/{id}/upload-proof`, multipart) y descarga (`GET /api/pension/records/{id}/proof`). Archivo guardado en disco bajo `DATA_DIR/uploads/payment-proofs/{year}/{month}/{record_id}.{ext}`; en DB solo `proof_file_name`. Sin análisis IA.
9. **REQ-009**: Wiring completo en `cmd/server/main.go` (storage → service → handler → `api.NewHandler`).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: `GET /api/pension/records` devuelve cada registro con sus relaciones resueltas: `child_name` (concat first_name + last_name) y `category_name` (nombre de pension_category), para que el frontend no necesite joins.
2. **REQ-011**: Validaciones de negocio: amount > 0, currency de 3 letras (default NIO), child_id y pension_category_id existentes, month 1-12, año razonable (2000-2100).
3. **REQ-012**: Respuestas de error JSON consistentes: `{"error": "código", "message": "descripción"}` (patrón existente en handlers del proyecto).
4. **REQ-013**: `GET /api/pension/closing/{year}/{month}` responde `{"closed": bool, "closed_at": ...}`.
5. **REQ-014**: Extensión y tipos en `frontend/src/types/index.ts` + métodos en el api client (`frontend/src/api/index.ts`) para consumir las nuevas APIs (los archivos de UI reales se crean en SPEC-050, pero el contrato de tipos/cliente API queda en esta spec para que SPEC-050 sea solo UI).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Endpoint de resumen `GET /api/pension/records/summary?year&month` que devuelva totales (total, pending, paid, rejected, y montos por estado) para las cards de resumen del frontend.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Queries SQLite simples con índices en (year, month) y FKs; listado de un mes acotado a pocos registros.
- **Seguridad**: Misma autenticación existente (`authMiddleware`) en todas las rutas nuevas; validación de tipos de archivo y tamaño máximo (10MB) para comprobantes.
- **Almacenamiento**: 3 tablas nuevas (~200-400 bytes/registro) + archivos de comprobante en disco dentro de `DATA_DIR`.
- **Disponibilidad**: Sin cambios en health checks ni schedulers existentes (SPEC-051 agrega scheduler de generación si procede).
- **iHost**: Sin dependencias nuevas; solo stdlib Go + SQLite; comprobantes en disco del volumen de datos.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se analizó el módulo `child-support` de P4OLA (`apps/api/src/modules/child-support/` y `apps/dashboard/src/pages/child-support/RecordsPage.tsx`) y el estado actual de p40la-ihost:

- **Entidades P4OLA**: `support-record.entity.ts` (estatus pending/paid/rejected, métodos de pago, comprobante, conversión de moneda), `salary-payment.entity.ts` (pending/received), `month-closing.entity.ts`. Mapeadas 1:1 al modelo SQLite de p40la-ihost.
- **Controllers P4OLA**: `support-records.controller.ts`, `salary-payments.controller.ts`, `month-closings.controller.ts`, `child-support-import.controller.ts` (upload-proof). Se adaptan los endpoints a la convención de rutas de p40la-ihost (`/api/pension/...`).
- **Guardado de comprobantes P4OLA**: `child-support-proof.service.ts` escribe a disco `/app/uploads/payment-proofs/{year}/{month}/{id}.{ext}` y guarda `proofFileName` en DB. En p40la-ihost se usa `DATA_DIR` (config) que es el volumen persistente del iHost → `{DATA_DIR}/uploads/payment-proofs/...`.
- **Sistema de archivos actual**: p40la-ihost no persiste archivos de facturas (solo `file_hash` para dedup, ver `internal/services/document.go`). Para comprobantes SÍ se persiste el archivo, siguiendo el patrón del volumen de datos ya montado.
- **Patrón de capas Go**: `internal/storage/*.go` (SQL crudo con `database/sql`), `internal/services/*.go` (validación), `internal/api/*_handlers.go`, wiring en `cmd/server/main.go`. Referencia: `internal/storage/salary.go`, `internal/services/salary.go`, `internal/api/salary_handlers.go`.
- **Modelos existentes**: `children`, `salaries` (con `employer`, `amount`, `currency_id`, `payment_day`, `active`, `note`), `pension_categories` (con `auto_generate`), `notifications` (destinatarios). El mapeo con P4OLA: `Salary.sourceName → salaries.employer`, `Salary.payDayOfMonth → salaries.payment_day`, `Salary.currency (string) → salaries.currency_id` (se resuelve el código vía tabla currencies).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Replicar entidades P4OLA como tablas SQLite nuevas | Modelo probado en producción P4OLA, mapeo directo | Más migraciones | ✅ Seleccionada |
| Reutilizar `bills` para registros de pensión | Cero tablas nuevas | Mezcla dominios distintos, sin salary_payments ni closing | ❌ Rechazada |
| Guardar comprobantes en DB (BLOB) | Sin archivos en disco | DB crece rápido en iHost, sin CDN/streaming | ❌ Rechazada |
| Endpoints bajo `/api/pension/...` | Consistente con rutas existentes del módulo | Difiere de P4OLA (`/child-support/...`) | ✅ Seleccionada (adaptación a convención del proyecto) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Nombres y rutas adaptados a p40la-ihost
- **Contexto**: P4OLA usa `/child-support/records` y nombres `support_record`/`support_category`. p40la-ihost ya usa `pension_categories`, `/pension/*`, y nombres en español para la UI.
- **Decisión**: Tabla `support_records` (mantiene el nombre del dominio, consistente con `pension_categories`), rutas bajo `/api/pension/records`, `/api/pension/salary-payments`, `/api/pension/closing`.
- **Consecuencias**: Coherente con el resto del proyecto; el frontend usa `/pension/registros` (ya definido en SPEC-044).

**ADR-002**: Sin análisis IA de comprobantes
- **Contexto**: P4OLA usa OpenAI para analizar el comprobante al marcar pagado. p40la-ihost no tiene integración OpenAI y prioriza cero dependencias externas en iHost.
- **Decisión**: El comprobante se sube y queda linkeado; fecha, método, referencia y notas se ingresan manualmente.
- **Consecuencias**: Sin clave API ni llamadas externas; flujo más simple. El campo `evidence_notes` mantiene la nota de evidencia.

**ADR-003**: Comprobantes en disco bajo `DATA_DIR`
- **Contexto**: El iHost persiste datos en el volumen montado (`DATA_DIR`, default `./data`).
- **Decisión**: `{DATA_DIR}/uploads/payment-proofs/{year}/{month}/{id}.{ext}`; en DB solo `proof_file_name` (nombre original para el link de descarga).
- **Consecuencias**: Descarga directa por el backend; limpieza manual si se borran registros (P2, no bloqueante).

**ADR-004**: Relaciones resueltas en el backend
- **Contexto**: P4OLA usa TypeORM `relations`. p40la-ihost usa SQL crudo.
- **Decisión**: El `GET /api/pension/records` hace JOIN a `children` y `pension_categories` y devuelve `child_name` y `category_name` en el JSON.
- **Consecuencias**: El frontend no necesita joins ni resolver nombres; respuesta ligera.

**ADR-005**: Cierre de mes como tabla de presencia
- **Contexto**: P4OLA modela `month_closings` con una fila por (year, month) cerrado.
- **Decisión**: Igual aquí: fila = mes cerrado. `closed` = existe fila. Cerrar inserta; reabrir borra.
- **Consecuencias**: Semántica simple; sin campos de estado redundantes en registros.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Handlers HTTP /api/pension/records*] --(validación)--> [Services] --(SQL)--> [Storage] --> [SQLite]
       |                                                                                       |
       +-- upload-proof (multipart) --> [disco: DATA_DIR/uploads/payment-proofs/...]
       +-- GET proof --> sirve archivo con Content-Disposition
```

### 4.2 Componentes

#### 4.2.1 Migraciones
- **Responsabilidad**: Crear `support_records`, `salary_payments`, `month_closings`.
- **Ubicación**: `migrations/0020_*.up.sql`/`.down.sql`, `0021_*`, `0022_*`.

#### 4.2.2 Modelos (`internal/models/`)
- `support_record.go`: SupportRecord (con campos de pago, conversión, proof).
- `salary_payment.go`: SalaryPayment.
- `month_closing.go`: MonthClosing.

#### 4.2.3 Storage (`internal/storage/`)
- `support_record.go`: ListByFilters (year/month/childID con JOIN), GetByID, Create, Update, Exists (para UNIQUE), MarkPaid, MarkPending, MarkRejected, UpdateProof.
- `salary_payment.go`: ListByFilters, GetByID, Create, Exists, MarkReceived, MarkPending.
- `month_closing.go`: IsClosed, Find, Close, Reopen.

#### 4.2.4 Services (`internal/services/`)
- `support_record.go`: validaciones + delegación a storage + chequeo de mes cerrado.
- `salary_payment.go`: validaciones + chequeo de mes cerrado.
- `month_closing.go`: lógica de cierre/reapertura.

#### 4.2.5 Handlers (`internal/api/`)
- `support_record_handlers.go`, `salary_payment_handlers.go`, `month_closing_handlers.go`.

### 4.3 Modelo de datos

```
Entidad: support_record
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- child_id: INTEGER NOT NULL (FK → children.id, ON DELETE CASCADE)
- pension_category_id: INTEGER NOT NULL (FK → pension_categories.id, ON DELETE CASCADE)
- year: INTEGER NOT NULL
- month: INTEGER NOT NULL (1-12)
- amount: REAL NOT NULL
- currency: TEXT(3) NOT NULL DEFAULT 'NIO'
- status: TEXT NOT NULL DEFAULT 'pending' (pending|paid|rejected)
- paid_at: DATETIME NULL
- payment_method: TEXT NULL (bank_transfer|cash|check|mobile|other)
- payment_reference: TEXT NULL
- evidence_notes: TEXT NULL
- notes: TEXT NULL (motivo de rechazo)
- proof_file_name: TEXT NULL
- original_amount: REAL NULL
- original_currency: TEXT(3) NULL
- exchange_rate: REAL NULL
- created_at / updated_at
- UNIQUE (child_id, pension_category_id, year, month)

Entidad: salary_payment
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- salary_id: INTEGER NOT NULL (FK → salaries.id, ON DELETE CASCADE)
- year: INTEGER NOT NULL
- month: INTEGER NOT NULL
- amount: REAL NOT NULL
- currency: TEXT(3) NOT NULL
- status: TEXT NOT NULL DEFAULT 'pending' (pending|received)
- received_amount: REAL NULL
- received_at: DATETIME NULL
- notes: TEXT NULL
- created_at / updated_at
- UNIQUE (salary_id, year, month)

Entidad: month_closing
- id: INTEGER PRIMARY KEY AUTOINCREMENT
- year: INTEGER NOT NULL
- month: INTEGER NOT NULL
- closed_at: DATETIME DEFAULT CURRENT_TIMESTAMP
- UNIQUE (year, month)
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/pension/records?year=2026&month=8`
**Response 200**:
```json
[
  {
    "id": 1, "child_id": 2, "child_name": "Juan Pérez",
    "pension_category_id": 3, "category_name": "Colegio",
    "year": 2026, "month": 8, "amount": 1500.00, "currency": "NIO",
    "status": "pending",
    "paid_at": null, "payment_method": null, "payment_reference": null,
    "evidence_notes": null, "notes": null, "proof_file_name": null,
    "original_amount": null, "original_currency": null, "exchange_rate": null
  }
]
```

#### Endpoint: `POST /api/pension/records`
**Request**:
```json
{ "child_id": 2, "pension_category_id": 3, "year": 2026, "month": 8, "amount": 1500.00, "currency": "NIO", "notes": null }
```
**Response 201**: registro creado | **400**: `{"error":"validation_error","message":"..."}` | **409**: mes cerrado o duplicado.

#### Endpoint: `POST /api/pension/records/{id}/mark-paid`
**Request**:
```json
{ "paid_at": "2026-08-15T10:00:00", "payment_method": "bank_transfer", "payment_reference": "REF-123", "evidence_notes": "Transferencia BANPRO", "original_amount": 40.0, "original_currency": "USD", "exchange_rate": 36.5 }
```
**Response 200**: `{"ok": true}` | **409**: mes cerrado.

#### Endpoint: `POST /api/pension/records/{id}/mark-pending` | `POST /api/pension/records/{id}/mark-rejected` (body `{"reason":"..."}`)
**Response 200**: `{"ok": true}`.

#### Endpoint: `GET /api/pension/salary-payments?year=2026&month=8`
**Response 200**: `[{ "id":1, "salary_id":4, "employer":"Empresa XYZ", "year":2026, "month":8, "amount":15000.00, "currency":"NIO", "status":"pending", "received_amount":null, "received_at":null, "notes":null }]`

#### Endpoint: `POST /api/pension/salary-payments/{id}/mark-received`
**Request**: `{ "received_at": "2026-08-15T12:00:00", "received_amount": 14500.00, "notes": "Depósito" }`
**Response 200**: `{"ok": true}` | **400**: falta received_at.

#### Endpoint: `GET /api/pension/closing/2026/8`
**Response 200**: `{"closed": false, "closed_at": null}`

#### Endpoint: `POST /api/pension/closing/2026/8` (cerrar) / `DELETE /api/pension/closing/2026/8` (reabrir)
**Response 200**: `{"ok": true}` | **409**: ya cerrado | **404**: no está cerrado.

#### Endpoint: `POST /api/pension/records/{id}/upload-proof` (multipart, campo `file`, máx 10MB, .pdf/.png/.jpg/.jpeg/.webp)
**Response 200**: `{"ok": true, "proof_file_name": "comprobante.pdf"}`

#### Endpoint: `GET /api/pension/records/{id}/proof`
**Response**: archivo con `Content-Disposition: attachment; filename="..."` | **404**: sin comprobante.

### 4.5 Dependencias

- **Internas**: `internal/storage/child.go`, `pension_category.go`, `salary.go` (FKs y validaciones), `internal/api/routes.go`, `cmd/server/main.go`, `internal/config` (DATA_DIR), componentes de errores JSON existentes.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Las migraciones 0020/0021/0022 aplican sobre una DB limpia (`go test` de migraciones) y los down.sql revierten sin error.
- [ ] CA-002: `GET /api/pension/records?year=2026&month=8` devuelve los registros de ese mes con `child_name` y `category_name` resueltos.
- [ ] CA-003: `POST /api/pension/records` crea un registro; un segundo POST con mismo (child, category, year, month) devuelve 409.
- [ ] CA-004: `mark-paid` guarda fecha/método/referencia/notas/conversión; `mark-pending` limpia los campos de pago; `mark-rejected` guarda el motivo.
- [ ] CA-005: `mark-received` de salary requiere `received_at`; sin él responde 400.
- [ ] CA-006: `POST /api/pension/closing/2026/8` cierra el mes; luego crear/editar/mark-* de registros de ese mes devuelve 409; `DELETE` reabre.
- [ ] CA-007: Subir un comprobante PDF/PNG guarda el archivo en disco y `proof_file_name` en DB; `GET .../proof` descarga el archivo con el nombre original.
- [ ] CA-008: Todas las rutas nuevas responden 401 sin sesión.
- [ ] CA-009: `go build ./...` compila y `go test ./...` pasa (incluye tests nuevos de services/storage).

### 5.2 No funcionales

- [ ] CA-NF-001: No se agregan dependencias externas nuevas.
- [ ] CA-NF-002: Los archivos de comprobante quedan dentro de `DATA_DIR` (volumen persistente del iHost).
- [ ] CA-NF-003: Las queries de listado por mes usan índices (year, month).

### 5.3 Testing

- **Unit tests**: Validaciones de services (mes cerrado, amount<=0, currency, FKs, mark-paid limpia/carga campos).
- **Integration tests**: Migraciones + CRUD contra SQLite en memoria (`internal/db/migrate_*_test.go` como referencia); listado con JOINs; cierre/reapertura.
- **E2E tests**: Validación manual con `curl` (login → crear registro → pagar → cerrar mes → intentar editar (409) → reabrir). Se documenta en la spec el flujo exacto de curl.
- **Carga/Performance**: Listado de un mes con 20+ registros responde < 50ms (queries indexadas).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migraciones 0020/0021/0022 (up/down) + tests de migración | 30 min | Ninguna |
| 2 | Modelos Go (`internal/models/`) | 15 min | Fase 1 |
| 3 | Storage (`internal/storage/`) | 45 min | Fase 2 |
| 4 | Services (`internal/services/`) | 45 min | Fase 3 |
| 5 | Handlers + rutas + proof upload/download | 45 min | Fase 4 |
| 6 | Wiring en `cmd/server/main.go` + `api.NewHandler` | 15 min | Fase 5 |
| 7 | Tipos + api client frontend (contrato para SPEC-050) | 20 min | Fase 5 |
| 8 | Tests Go + validación curl + `go build`/`go test` | 45 min | Fase 6, 7 |
| **Total** | | **~4.5 horas** | |

### 6.2 Milestones

1. **MVP**: API completa y verificable por curl (Fases 1-7).
2. **V1.0**: MVP + tests + validación manual del usuario.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Colisión de número de migración con SPEC-043 (in_progress) | Baja | Medio | Verificar último archivo en `migrations/` antes de crear (hoy: 0019). Si SPEC-043 agrega 0020, usar 0023+ |
| Mapeo de moneda: `salaries.currency_id` vs `currency` string | Media | Medio | Al listar salary_payments, resolver el código de moneda vía tabla `currencies` (JOIN) en el backend |
| Comprobantes sin limpieza al borrar un registro | Baja | Bajo | P2 documentado; la migración usa CASCADE solo en DB, el archivo huérfano se tolera |
| `mark-paid` con conversión cuando original_currency == currency | Baja | Bajo | El service ignora/limpia la conversión si ambas monedas son iguales |
| Frontend de SPEC-050 espera campos distintos | Media | Medio | El contrato de tipos/api client se congela en esta spec (Fase 7) para que SPEC-050 solo consuma |

## 8. Notas y Referencias

- Referencia de dominio (P4OLA): `apps/api/src/modules/child-support/entities/{support-record,salary-payment,month-closing}.entity.ts`, `support-records.{controller,service}.ts`, `salary-payments.{controller,service}.ts`, `month-closings.{controller,service}.ts`, `child-support-proof.service.ts`.
- Referencia de estilo backend p40la-ihost: `internal/storage/salary.go`, `internal/services/salary.go`, `internal/api/salary_handlers.go`, `migrations/0017_create_salaries.*`, `cmd/server/main.go`.
- SPECs relacionadas: SPEC-044 (sidebar), SPEC-045 (hijos), SPEC-046 (notificaciones), SPEC-047 (salarios), SPEC-048 (categorías). Siguiente: SPEC-050 (frontend) y SPEC-051 (emails + generación).
- Restricciones iHost: ver `docs/project-rules.md` y AGENTS.md (nunca borrar `data/app.db`, usar `killall`, verificar `/health`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-02 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-02 | p40la-ihost-team | Estado cambiado a pending_execution (aprobada para desarrollo) |
| 2026-09-02 | p40la-ihost-team | Estado cambiado a in_progress; inicio de desarrollo en worktree feature/SPEC-049 |
| 2026-09-02 | p40la-ihost-team | Implementación completa: migraciones 0020/0021/0022, modelos, storage, services (mes cerrado), handlers, rutas /api/pension/*, proof upload/download, wiring, tests Go, contrato frontend (tipos + api client). Validado en local (go build, go test, server con flujo completo por curl). Fallo preexistente detectado: TestBillPayBill (no relacionado) |
| 2026-09-03 | p40la-ihost-team | Release: merge feature/SPEC-049 a main (commit d5efb77), validación manual del usuario aprobada, issue #49 cerrado, worktree limpiado |