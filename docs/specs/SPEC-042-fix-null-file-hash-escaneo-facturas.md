---
title: "Fix error NULL en file_hash al escanear facturas con registros existentes en iHost"
id: "SPEC-042"
status: "released"
author: "paulomcnally"
created: "2026-08-17"
updated: "2026-08-17"
github_issue: 42
---

# Fix error NULL en file_hash al escanear facturas con registros existentes en iHost

**ID**: SPEC-042  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-08-17  
**Actualizado**: 2026-08-17

---

## 1. Resumen Ejecutivo

Tras la migración de la SPEC-041, la columna `bills.file_hash` se agregó como `TEXT NULL` y el índice único parcial correspondiente se diseñó explícitamente para permitir NULL (los bills creados manualmente, sin PDF, no tienen hash). El test de la migración (`migrate_014_test.go`) incluso valida que insertar `file_hash = NULL` sea válido.

Sin embargo, las funciones de escaneo del storage (`scanBill` y `scanBills` en `internal/storage/bill.go`) escanean la columna directamente en `b.FileHash`, que es un `string` de Go. Cuando un registro existente en la base de datos del iHost tiene `file_hash = NULL` (facturas creadas antes de la SPEC-041 o facturas manuales), el escaneo falla con:

```
escanear factura: sql: Scan error on column index 8, name "file_hash": converting NULL to string is unsupported
```

Esto rompe endpoints que listan/leen facturas, como `GET /api/services/8/bills`, haciendo que la app devuelva un error interno en lugar de la lista de facturas. Es un hotfix de robustez de lectura: **la DB del iHost es producción y ya contiene registros con NULL**, por lo que el fix debe ser retrocompatible sin necesidad de migración ni de tocar datos.

Durante la implementación se detectó que **no es el único campo nullable escaneado a `string`**: según el schema de la tabla `bills` (`migrations/0005`/`0006`), las columnas `invoice_number` y `drive_url` también son `TEXT` nullable (facturas manuales o legacy no las llenan). El fix se extiende a **todas las columnas string nullable** (`invoice_number`, `drive_url`, `file_hash`) para prevenir el mismo error en cualquier registro.

Durante la validación manual con el usuario se confirmó que **el mismo defecto existe en otras entidades**: la validación local reprodujo `escanear hogar: sql: Scan error on column index 2, name "address": converting NULL to string is unsupported` al listar casas (`GET /api/homes`) y crear una casa falla. La tabla `homes.address` es `TEXT` nullable y se escanea directo a `string`. El análisis del schema detectó el mismo patrón en `services.institution` (`TEXT` nullable). Esta spec cubre **todas las columnas string nullable de todas las entidades** (`homes.address`, `services.institution`, y las 3 de `bills`), no solo las de `bills`.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Todas las funciones de escaneo de storage (`scanBill`/`scanBills`, `scanHome`/`scanHomes`, `scanService`/`scanServices`) deben manejar las columnas string nullable sin error, escaneándolas con `sql.NullString` y asignando el valor al modelo solo cuando es válido (patrón idéntico al ya usado para `deleted_at` con `sql.NullTime`). Columnas afectadas:
   - `bills`: `invoice_number`, `drive_url`, `file_hash`
   - `homes`: `address`
   - `services`: `institution`

2. **REQ-002**: Los endpoints de lectura de todas las entidades (`GET /api/services/{service_id}/bills`, `GET /api/bills/{id}`, `GET /api/homes`, `GET /api/services`, upserts por periodo/hash, pendientes) deben funcionar con registros que tengan estas columnas NULL, vacío o con valor.

3. **REQ-003**: No debe cambiar el contrato JSON del modelo: los campos `invoice_number`, `drive_url`, `file_hash`, `address` e `institution` se mantienen con `omitempty`, por lo que registros sin esos datos no exponen el campo (NULL o vacío se comportan igual hacia el frontend).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Agregar tests unitarios de storage que inserten registros con las columnas string nullable en NULL y verifiquen que `ListByService`/`GetByID` (bills), `List`/`GetByID` (homes) y `List`/`GetByID` (services) los devuelven sin error y con los campos vacíos.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-005**: Verificar (sin cambios de código necesarios) que la escritura de `file_hash` existente (Create/Update/UpdateFromExtracted) sigue persistiendo correctamente para facturas de PDF.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Cambio mínimo de scanning (un `sql.NullString` por fila), sin impacto medible en iHost.
- **Seguridad**: Sin cambios de autenticación ni exposición de datos adicionales.
- **Almacenamiento**: Sin migración nueva y sin modificación de datos existentes (la DB de iHost es producción).
- **Disponibilidad**: Restaurar `GET /api/services/{service_id}/bills` y endpoints de lectura de facturas en iHost.
- **iHost**: Zero dependencias nuevas; solo stdlib `database/sql`. Cambio acotado a `internal/storage/bill.go` + tests.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Migración**: `migrations/0014_add_bills_file_hash.up.sql` define `file_hash TEXT NULL` con índice único parcial `WHERE file_hash IS NOT NULL AND file_hash != ''`. El comentario del propio archivo indica: "file_hash es NULL para facturas creadas manualmente (sin PDF)".
- **Test de migración**: `internal/db/migrate_014_test.go` (líneas 64-66) inserta explícitamente `file_hash = NULL` y espera éxito.
- **Modelo**: `internal/models/bill.go` — `FileHash string` con `json:"file_hash,omitempty"`.
- **Storage**: `internal/storage/bill.go` — los SELECTs (ListByService, GetByID, FindByServicePeriod, FindByServiceFileHash) incluyen `file_hash` y ambos `scanBill`/`scanBills` lo escanean directo a `string` (líneas 167 y 185). El resto de columnas nullable (`deleted_at`) ya se maneja con `sql.NullTime`, sentando el precedente de patrón.
- **Endpoint afectado reportado**: `GET /api/services/8/bills` devuelve el error al escanear filas con `file_hash` NULL (registros pre-existentes en la DB del iHost).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **`sql.NullString` en scanBill/scanBills** (asignar solo si `Valid`) | Mínimo cambio, patrón ya usado para `deleted_at`, no toca SQL ni JSON | Ninguna relevante | ✅ Seleccionada |
| `COALESCE(file_hash, '')` en cada SELECT | Fuerza string vacío en SQL | Cambia 4 queries, más código, no es idiomático en Go, rompe consistencia con el patrón NullTime | ❌ Rechazada |
| Migración/backfill para setear `''` donde hay NULL | Sin cambio de scan | Toca datos de producción innecesariamente, requiere nueva migración, no aporta valor (NULL y '' se serializan igual por omitempty) | ❌ Rechazada |
| Cambiar el modelo a `*string` | Explícito | Más cambios (handlers, comparaciones), sin beneficio para el contrato JSON | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-042-001**: Manejar NULL en el scan con `sql.NullString`, no en SQL ni en el modelo.
- **Contexto**: Las columnas `invoice_number`, `drive_url`, `file_hash`, `address` e `institution` son nullable por diseño (registros manuales/legacy) y ya hay registros NULL en producción. El escaneo actual a `string` rompe la lectura.
- **Decisión**: En todas las funciones de scan de storage (`bill.go`, `home.go`, `service.go`), escanear estas columnas en variables `sql.NullString` locales y asignar al modelo solo si `Valid` (dejando `""` si son NULL). Mismo patrón exacto que `deletedAt` con `sql.NullTime`.
- **Consecuencias**: Los endpoints de lectura vuelven a funcionar con datos legacy y manuales. Sin cambios de schema, datos, contrato JSON ni dependencias.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[GET /api/services/{id}/bills]
        │
        ▼
[BillStorage.ListByService] → rows.Scan(...)
        │
        ▼
[scanBills]  -- file_hash via sql.NullString -->
        │              (Valid=false → FileHash="")
        ▼
[[]models.Bill]  → JSON (file_hash omitido por omitempty si vacío)
```

### 4.2 Componentes

#### 4.2.1 `internal/storage/bill.go` (modificado)
- **Responsabilidad**: Corregir el escaneo de las columnas string nullable (`invoice_number`, `drive_url`, `file_hash`) para tolerar NULL.
- **Interfaz**: Sin cambios de firma.
- **Dependencias**: `database/sql` (stdlib).
- **Ubicación**: `internal/storage/bill.go` — funciones `scanBill` (línea ~163) y `scanBills` (línea ~179).

Cambio en ambas funciones:
```go
var invoiceNumber, driveURL, fileHash sql.NullString
// ... al escanear:
&invoiceNumber, &driveURL, &fileHash
// ... después del scan:
b.InvoiceNumber = invoiceNumber.String // "" si NULL o vacío
b.DriveURL = driveURL.String           // "" si NULL o vacío
b.FileHash = fileHash.String           // "" si NULL o vacío
```

#### 4.2.2 `internal/storage/home.go` (modificar)
- **Responsabilidad**: Corregir el escaneo de `homes.address` (TEXT nullable) para tolerar NULL.
- **Interfaz**: Sin cambios de firma.
- **Ubicación**: `internal/storage/home.go` — `scanHome` (línea ~99) y `scanHomes` (línea ~114).

Cambio en ambas funciones:
```go
var address sql.NullString
// ... al escanear:
&address
// ... después del scan:
h.Address = address.String // "" si NULL o vacío
```

#### 4.2.3 `internal/storage/service.go` (modificar)
- **Responsabilidad**: Corregir el escaneo de `services.institution` (TEXT nullable) para tolerar NULL.
- **Interfaz**: Sin cambios de firma.
- **Ubicación**: `internal/storage/service.go` — `scanService` (línea ~110) y `scanServices` (línea ~151).

Cambio en ambas funciones:
```go
var institution sql.NullString
// ... al escanear:
&institution
// ... después del scan:
svc.Institution = institution.String // "" si NULL o vacío
```

#### 4.2.2 Test nuevo `internal/storage/bill_null_file_hash_test.go` (creado)
- **Responsabilidad**: Verificar que `ListByService`/`GetByID` toleran `file_hash`/`invoice_number`/`drive_url` NULL.
- **Interfaz**: Test unitario estándar del proyecto.
- **Dependencias**: `database/sql` + driver sqlite usado en tests del repo.
- **Ubicación**: `internal/storage/`.

#### 4.2.3 Test nuevo `internal/storage/home_null_test.go` (crear)
- **Responsabilidad**: Verificar que `List`/`GetByID` de homes toleran `address` NULL.
- **Ubicación**: `internal/storage/`.

#### 4.2.4 Test nuevo `internal/storage/service_null_test.go` (crear)
- **Responsabilidad**: Verificar que `List`/`GetByID` de services toleran `institution` NULL.
- **Ubicación**: `internal/storage/`.

### 4.3 Modelo de datos

Sin cambios de modelo ni migración.

```
Entidad: Bill (sin cambios)
- FileHash string `json:"file_hash,omitempty"` → "" cuando DB tiene NULL o ''
```

### 4.4 APIs / Contratos

Sin cambios de contrato. Comportamiento restaurado:

#### Endpoint: `GET /api/services/{service_id}/bills`

**Response 200** (con factura manual, campos NULL en DB):
```json
{ "id": 8, "service_id": 8, "year": 2026, "month": 1, "amount": 100, "status": "pending", ... }
```
(sin los campos `invoice_number`, `drive_url`, `file_hash` cuando son NULL, por `omitempty`)

### 4.5 Dependencias

- **Internas**: `internal/storage/bill.go`, nuevo test en `internal/storage/`.
- **Externas**: Ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un bill con `invoice_number`, `drive_url` y `file_hash` NULL en la DB, cuando se llama a `GET /api/services/{service_id}/bills`, entonces responde 200 con la lista completa sin error de scan.
- [x] CA-002: Dado un bill con `invoice_number`, `drive_url` o `file_hash` en `''` o con valor real, cuando se escanea, entonces se devuelve el valor correcto en el modelo (vacío o dato).
- [x] CA-003: Dado un registro con campos NULL, cuando se escanea, entonces esos campos NO aparecen en el JSON (omitempty), igual que antes de la SPEC-041.
- [x] CA-004: Los endpoints de lectura que usan los scans (`GET /api/bills/{id}`, `FindByServicePeriod`, `FindByServiceFileHash`, `GET /api/homes`, `GET /api/services`) funcionan con filas NULL sin regresión.
- [x] CA-005: Dado un hogar con `address` NULL en la DB, cuando se llama a `GET /api/homes`, entonces responde 200 con la lista completa sin error de scan (caso reportado por el usuario).
- [x] CA-006: Dado un servicio con `institution` NULL en la DB, cuando se llama a `GET /api/services`, entonces responde 200 con la lista completa sin error de scan.

### 5.2 No funcionales

- [x] CA-NF-001: `go build` sin errores.
- [x] CA-NF-002: Tests del storage pasan (`go test ./internal/storage/...`).
- [x] CA-NF-003: Sin migración nueva y sin modificar datos de la DB de iHost.

### 5.3 Testing

- **Unit tests**: Test nuevo que inserta bills con `invoice_number`/`drive_url`/`file_hash` NULL, `''` y valores reales, y verifica `ListByService`/`GetByID` (CA-001..004).
- **Integration tests**: Verificación local con la DB real (o copia) reproduciendo el escenario del reporte (`service_id=8` con registros NULL).
- **E2E tests**: Navegar a una página de facturas con registros manuales legacy en iHost y ver que lista correctamente.
- **Carga/Performance**: No aplica (cambio trivial de scanning).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modificar `scanBill` y `scanBills` para usar `sql.NullString` en `file_hash` | 0.25 día | Ninguna |
| 2 | Crear test de storage con filas NULL/vacío/hash y verificar que pasan | 0.25 día | Fase 1 |
| 3 | `go build` + `go test ./internal/storage/...` + prueba local del endpoint | 0.25 día | Fase 2 |
| 4 | Validación manual con el usuario en iHost (endpoint del reporte) | 0.25 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Fix de scan + test que lo cubre (Fases 1-3).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Otras entidades con columnas string nullable sin cubrir | Media | Medio | Auditoría del schema de todas las tablas (`bills`, `homes`, `services`, `autos`, `institutions`, `institution_categories`, `auto_services`, `alerts`, `currencies`). Las columnas nullable identificadas (`invoice_number`, `drive_url`, `file_hash`, `address`, `institution`) se cubren en esta spec; el resto son NOT NULL o ya usan NullString/NullInt64 |
| Test requiere driver/setup de DB | Baja | Bajo | Reutilizar el helper/setup de DB usado por `bill_pending_test.go` existente |
| El usuario reintenta antes de deploy a iHost | Media | Bajo | Prueba local primero; deploy a iHost solo tras confirmación del usuario |

## 8. Notas y Referencias

- Reporte del usuario: `curl http://ihost.local:8088/api/services/8/bills` → `sql: Scan error on column index 8, name "file_hash": converting NULL to string is unsupported`
- SPEC-041 (subida múltiple de facturas con `file_hash`): `docs/specs/SPEC-041-subida-multiple-facturas.md`
- Migración: `migrations/0014_add_bills_file_hash.up.sql`
- Código afectado: `internal/storage/bill.go` (scanBill/scanBills), `internal/models/bill.go`
- Precedente de patrón: `deletedAt` con `sql.NullTime` en el mismo archivo.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-17 | paulomcnally | Creación inicial de la especificación (fix hotfix de robustez de lectura) |
| 2026-08-17 | paulomcnally | Cambio de estado a pending_execution (issue #42) |
| 2026-08-17 | paulomcnally | Alcance ampliado durante implementación: el test reveló que `invoice_number` y `drive_url` también son NULL-ables y rompían igual que `file_hash`. La spec cubre ahora las 3 columnas string nullable de `bills` |
| 2026-08-17 | paulomcnally | Cambio de estado a in_progress. Implementación: fix de scanBill/scanBills con sql.NullString para las 3 columnas + test `bill_null_file_hash_test.go`. `go build` y `go test ./...` en verde |
| 2026-08-17 | paulomcnally | Alcance ampliado tras validación con el usuario: el mismo defecto se reprodujo en `homes.address` (GET /api/homes) y se detectó en `services.institution`. La spec cubre ahora todas las columnas string nullable: bills (3), homes.address y services.institution |
| 2026-08-17 | paulomcnally | Validación manual del usuario satisfactoria (bills, homes, services). Criterios de aceptación marcados como pass. Cambio de estado a pending_release |
| 2026-08-17 | paulomcnally | Release: merge de feature/SPEC-042 a main (commit b0534f0) y push. Estado a released. Issue #42 cerrado |