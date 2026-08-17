---
title: "Subida múltiple de facturas para importación masiva de pagos"
id: "SPEC-041"
status: "pending_release"
author: "paulomcnally"
created: "2026-08-16"
updated: "2026-08-16"
github_issue: 41
---

# Subida múltiple de facturas para importación masiva de pagos

**ID**: SPEC-041  
**Estado**: pending_release  
**Autor**: paulomcnally  
**Creado**: 2026-08-16  
**Actualizado**: 2026-08-17

---

## 1. Resumen Ejecutivo

Hoy el modal de subida de facturas (`UploadBillModal`, de la SPEC-014) solo permite seleccionar **un archivo a la vez**: el input `type="file"` no tiene el atributo `multiple` y el flujo es seleccionar → analizar → guardar. El usuario tiene registros de pago de todo un año (múltiples PDF por servicio) y quiere cargarlos todos de una vez, en vez de repetir el proceso 12+ veces.

Esta spec implementa la **selección múltiple de archivos por defecto** y el flujo de **importación masiva**: seleccionar N PDF, analizarlos (reutilizando el endpoint existente por archivo), revisar/editar los datos extraídos en una lista y guardarlos todos en lote. El `from-extracted` ya hace *upsert* por `(service_id, year, month)` (SPEC-021), así que cargar el año completo no duplica facturas: actualiza las existentes y crea las faltantes.

Para garantizar que un archivo ya importado **no se importe dos veces**, se almacena el **hash MD5 del archivo** (`file_hash`) en cada factura generada desde un PDF. Al subir un archivo cuyo hash ya existe para el servicio, se salta (flag `duplicate`) sin crear ni modificar nada. El MD5 se calcula en el backend (stdlib `crypto/md5`, sin dependencias) porque el frontend no puede usar `crypto.subtle` sobre HTTP —el iHost se accede por IP, no por HTTPS/localhost—. El hash es una capa de *dedup exacto* complementaria al upsert por periodo: cubre el caso "mismo PDF re-subido", mientras el upsert cubre "mismo mes, archivo distinto".

**Consideraciones de iHost**: el enfoque P0 es solo frontend + reutilización de APIs existentes. El análisis se hace de forma **secuencial (un request por archivo)**, lo que mantiene bajo el pico de memoria en el iHost en lugar de cargar todos los PDF en una sola petición. Zero cambios de DB y sin dependencias nuevas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El input de archivo del `UploadBillModal` debe tener el atributo `multiple` (selección múltiple por defecto), con `accept=".pdf,.png,.jpg,.jpeg"`.

2. **REQ-002**: Al seleccionar archivos, mostrar la lista completa con nombre y tamaño, validando por archivo extensión y tamaño máximo (10MB) antes de analizar.

3. **REQ-003**: El botón "Analizar facturas" procesa cada archivo de forma secuencial con `POST /api/services/{service_id}/bills/upload` (request por archivo), acumulando un resultado por archivo: datos extraídos o error individual.

4. **REQ-004**: En el estado de resultado, mostrar una **lista** de cards (una por archivo) con los datos extraídos editables (año, mes, monto, número de factura) y el analizador usado; las que fallaron muestran su error sin bloquear a las demás.

5. **REQ-005**: Botón "Guardar facturas" que itera los registros válidos llamando a `POST /api/services/{service_id}/bills/from-extracted` (uno por registro), mostrando estado de guardado y un resumen final (ej: "10 facturas guardadas, 2 errores").

6. **REQ-006**: Mantener el flujo actual de archivo único funcionando (seleccionar 1 → analizar → guardar) sin regresiones.

7. **REQ-007**: Al terminar de guardar, cerrar el modal y refrescar la lista de facturas.

8. **REQ-008**: Calcular el hash **MD5** de cada archivo al subirlo (backend), devolverlo en la respuesta del upload y persistirlo en la factura (`file_hash`). Si un archivo ya fue importado para el mismo servicio (mismo hash), **saltarlo** con flag `duplicate` sin crear ni modificar facturas.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Endpoint batch backend `POST /api/services/{service_id}/bills/upload-many` que acepte múltiples archivos (campo `files`) en un solo request y devuelva `[{filename, extracted, analyzer_used, file_hash, error}]`, reduciendo round-trips de red para importaciones grandes. Frontend lo usa si está disponible, con fallback al loop de P0.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Drag & drop de archivos en el área de selección.
2. **REQ-011**: Barra de progreso por archivo durante el análisis y botón de cancelar.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Análisis secuencial para limitar pico de memoria en iHost. Sin procesamiento en paralelo.
- **Seguridad**: Mantener validaciones existentes por archivo (extensión, MIME, máx 10MB) en backend; el frontend valida antes de enviar para feedback inmediato.
- **Almacenamiento**: No se persisten archivos en disco; el análisis es en memoria por archivo, liberando tras cada uno. Solo se guarda el hash MD5 (`file_hash`, 32 chars) por factura para dedup.
- **iHost**: Zero deps nuevas. MD5 con stdlib `crypto/md5`. Solo React (frontend) y reutilización de servicios Go existentes.
- **UX**: Feedback claro por archivo (ok/error) y resumen al finalizar.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se revisó el estado actual de la subida de facturas:

- **Frontend**: `frontend/src/components/UploadBillModal.tsx` — input `type="file"` sin `multiple`, estado `select → analyzing → result/error`, guardado individual. (SPEC-014)
- **API frontend**: `frontend/src/api/index.ts` — `bills.uploadAndAnalyze(serviceId, file)` (FormData campo `file`) y `bills.createBillFromExtracted(serviceId, body)`.
- **Backend**: `internal/api/document_handlers.go` — `UploadAndAnalyze` lee `r.FormFile("file")` (un solo archivo), valida y analiza; `CreateBillFromExtracted` hace *upsert* de la factura.
- **Servicio**: `internal/services/document.go` — `UploadAndAnalyze` valida extensión/MIME/10MB y delega en el analizador de la institución; `CreateBillFromExtracted` busca por `(service_id, year, month)` y actualiza o crea (no duplica).
- **Rutas**: `internal/api/routes.go` (líneas 70-71) registran los endpoints de upload y from-extracted con auth middleware.
- **Analizadores**: `DocumentAnalyzer.Analyze(reader io.Reader, mimeType)` — procesa por archivo en memoria; el upsert por periodo garantiza idempotencia al importar un año completo.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Frontend: `multiple` + loop secuencial sobre APIs existentes | Zero cambios backend, pico de memoria bajo (un PDF a la vez), bajo riesgo | N requests de red (aceptable para ~12-24 archivos) | ✅ Seleccionada (P0) |
| Endpoint batch backend `upload-many` | Un solo request, resultados agrupados | Más código backend, carga todos los archivos a memoria en el iHost | ⚠️ P1 (optimización opcional) |
| Frontend con análisis en paralelo (Promise.all) | Más rápido | Pico de memoria × N en iHost, mayor complejidad de errores | ❌ Rechazada |
| Subir todos y guardar sin revisión | Más simple | No permite corregir datos mal extraídos (perdía valor de SPEC-014) | ❌ Rechazada |

#### Dedup de archivos re-importados

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **MD5 del archivo en `bills.file_hash`** (backend) | Detecta el archivo exacto re-subido; stdlib `crypto/md5`; barato en iHost | Requiere migración + columna nueva | ✅ Seleccionada |
| `invoice_number` como único | Sin migración | Analizador puede no extraerlo; se repite entre años | ❌ Rechazada |
| Nombre de archivo | Sin migración | Se puede renombrar; no es único | ❌ Rechazada |
| Solo upsert por periodo (existente) | Ya implementado | No detecta el mismo PDF re-subido | ⚠️ Complementaria, no suficiente |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-041-001**: Importación masiva reutilizando endpoints existentes (P0).
- **Contexto**: Se quiere subir N PDF a la vez con revisión previa al guardado.
- **Decisión**: El modal usa `multiple` y procesa los archivos de forma secuencial con `bills.uploadAndAnalyze` y `bills.createBillFromExtracted` existentes. El upsert por periodo de `CreateBillFromExtracted` garantiza que importar un año completo no duplique facturas.
- **Consecuencias**: Cero cambios de backend para el MVP; N requests de red pero con uso de memoria estable en iHost.

**ADR-041-002**: Análisis secuencial, nunca en paralelo.
- **Contexto**: El iHost tiene recursos limitados (RAM/CPU).
- **Decisión**: Procesar un archivo a la vez, liberando memoria entre cada análisis. Nunca `Promise.all` con N PDF.
- **Consecuencias**: Importaciones de ~12-24 archivos tardan más, pero el riesgo de OOM es mínimo.

**ADR-041-003**: Dedup por hash MD5 del archivo, calculado en backend.
- **Contexto**: Evitar que un mismo PDF se importe dos veces, incluso si el usuario re-subió el archivo o corrigió los datos extraídos y reintentó.
- **Decisión**: El backend calcula el MD5 del archivo durante el análisis (via `io.TeeReader` hacia el hash writer, sin lectura extra), lo devuelve en `POST /bills/upload` como `file_hash`, y `POST /bills/from-extracted` lo persiste en `bills.file_hash`. Índice único parcial por `(service_id, file_hash) WHERE file_hash IS NOT NULL`. Si ya existe un bill con ese hash para el servicio → skip con `duplicate: true`. MD5 se usa por *dedup* (no seguridad), es determinístico y barato.
- **Consecuencias**: Nueva columna y migración; los bills creados manualmente (sin PDF) no tienen hash y no se ven afectados (NULL es permitido). Se pierde dedup si el usuario edita el PDF y lo re-subir (nuevo hash) — comportamiento esperado.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[UploadBillModal (multiple)]
      │ selecciona N archivos
      ▼
[Flujo por archivo (loop secuencial)]
      ├── POST /bills/upload        → { extracted, analyzer_used } | error
      ▼
[Lista de resultados editables (card por archivo)]
      │ "Guardar facturas"
      ▼
[Loop secuencial]
      └── POST /bills/from-extracted → upsert por (service_id, year, month) → resumen final
```

### 4.2 Componentes

#### 4.2.1 `frontend/src/components/UploadBillModal.tsx` (modificar)
- **Responsabilidad**: Soporte multi-archivo: selección múltiple, lista de archivos, análisis secuencial, lista de resultados editables y guardado en lote.
- **Interfaz**: Props sin cambios (`isOpen`, `serviceId`, `frequency`, `onClose`, `onSaved`).
- **Dependencias**: `api.bills` existentes, `Icon`, `useToast`.
- **Ubicación**: `frontend/src/components/UploadBillModal.tsx`.

#### 4.2.2 `frontend/src/api/index.ts` (modificar, menor)
- **Responsabilidad**: (P0) sin cambios de firma; se reutilizan `uploadAndAnalyze` y `createBillFromExtracted`. (P1) agregar `uploadAndAnalyzeMany(serviceId, files)` si se implementa el endpoint batch.

#### 4.2.3 Backend (P0: dedup + P1 opcional)
- **Responsabilidad**: (P0) calcular y persistir `file_hash` (MD5) para dedup de archivos. (P1) endpoint `POST /api/services/{service_id}/bills/upload-many`.
- **Ubicación**: `internal/api/document_handlers.go`, `internal/api/routes.go`, `internal/services/document.go`, `internal/storage/bill.go`, `internal/models/bill.go`.

### 4.3 Modelo de datos

**Migración nueva**: `migrations/0014_add_bills_file_hash.up.sql` / `.down.sql`

```sql
-- up
ALTER TABLE bills ADD COLUMN file_hash TEXT NULL;
CREATE UNIQUE INDEX idx_bills_service_file_hash ON bills(service_id, file_hash) WHERE file_hash IS NOT NULL;

-- down
DROP INDEX IF EXISTS idx_bills_service_file_hash;
ALTER TABLE bills DROP COLUMN file_hash;
```

```
Entidad: Bill
- file_hash: string (NULL si la factura no vino de un PDF) — MD5 del archivo importado
- Upsert por (service_id, year, month) → idempotencia por periodo
- Dedup por (service_id, file_hash) → no re-importar el mismo archivo
```

### 4.4 APIs / Contratos

#### Endpoint (existente, sin cambios - P0): `POST /api/services/{service_id}/bills/upload`

**Request**: FormData `file` (un archivo por request)
**Response 200**:
```json
{
  "extracted": { "amount": 123.45, "invoice_number": "ABC-123", "year": 2026, "month": 3 },
  "analyzer_used": "claro",
  "file_hash": "d41d8cd98f00b204e9800998ecf8427e"
}
```

#### Endpoint (existente, con cambios - P0): `POST /api/services/{service_id}/bills/from-extracted`

**Request**:
```json
{ "amount": 123.45, "invoice_number": "ABC-123", "year": 2026, "month": 3, "file_hash": "d41d8cd98f00b204e9800998ecf8427e" }
```
**Response 200/201**: `{ id, service_id, year, month, amount, invoice_number, status, updated, duplicate }`
- `duplicate: true` → el archivo ya fue importado para este servicio (mismo hash); no se crea ni modifica nada.

#### Endpoint (P1, opcional): `POST /api/services/{service_id}/bills/upload-many`

**Request**: FormData `files` (N archivos)
**Response 200**:
```json
[
  { "filename": "factura-ene.pdf", "extracted": {...}, "analyzer_used": "claro", "file_hash": "...", "error": null },
  { "filename": "factura-feb.pdf", "extracted": null, "analyzer_used": "", "file_hash": null, "error": "formato no soportado" }
]
```

### 4.5 Dependencias

- **Internas**: `UploadBillModal.tsx`, `frontend/src/api/index.ts` (frontend); `document_handlers.go`, `routes.go`, `document.go`, `storage/bill.go`, `models/bill.go`, `migrations/0014_*` (backend).
- **Externas**: Ninguna nueva (MD5 con stdlib `crypto/md5`).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: El input de archivo del modal tiene el atributo `multiple` (subida de varios archivos por defecto).
- [x] CA-002: Al seleccionar N archivos, se muestran todos con nombre/tamaño y validación por archivo (extensión y máx 10MB).
- [x] CA-003: Al analizar, cada archivo se procesa y el resultado se muestra como lista: datos extraídos editables + errores individuales por archivo.
- [x] CA-004: Al guardar, se crean/actualizan todas las facturas válidas en lote y se muestra resumen (ej: "N guardadas, M errores").
- [x] CA-005: Importar el mismo periodo dos veces no duplica facturas (upsert por servicio+mes+año).
- [x] CA-006: El flujo de archivo único sigue funcionando sin regresiones.
- [x] CA-007: Al terminar, el modal se cierra y la lista de facturas se refresca.
- [x] CA-008: Subir de nuevo el **mismo PDF** para el mismo servicio se salta con flag `duplicate` (no crea ni modifica facturas) y el resumen lo reporta como "ya importada".
- [x] CA-009: Las facturas creadas manualmente (sin PDF) funcionan igual (NULL en `file_hash`).

### 5.2 No funcionales

- [x] CA-NF-001: Build de Vite sin errores.
- [x] CA-NF-002: Sin dependencias npm nuevas.
- [x] CA-NF-003: Migración `0014` aplica y revierte correctamente (up/down).
- [ ] CA-NF-004: (P1) `upload-many` devuelve un resultado por archivo, con `error` null o mensaje.

### 5.3 Testing

- **Unit tests**: Lógica de agrupación de resultados (extraídos vs errores) y de resumen.
- **Integration tests**: (P1) endpoint `upload-many` con múltiples archivos.
- **E2E tests**: Seleccionar 12 PDF (un año) → analizar → editar algún dato → guardar → verificar facturas creadas/actualizadas sin duplicados.
- **Carga/Performance**: Importación de ~12-24 archivos en iHost sin picos de memoria; verificar comportamiento secuencial.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0014_add_bills_file_hash` (columna + índice único parcial) | 0.5 día | Ninguna |
| 2 | Backend dedup: MD5 en `UploadAndAnalyze` (TeeReader), `file_hash` en upload/from-extracted, `FindByServiceFileHash`, flag `duplicate` | 1 día | Fase 1 |
| 3 | Modificar `UploadBillModal` para selección múltiple + lista de archivos | 1 día | Ninguna |
| 4 | Loop secuencial de análisis + lista de resultados editables (con `file_hash` en cadena) | 1 día | Fase 3 |
| 5 | Guardado en lote + resumen (guardadas/actualizadas/ya importadas/errores) + refresh | 0.5 día | Fases 2 y 4 |
| 6 | Build Vite + `go build`/tests + pruebas manuales locales (importar un año + re-importar) | 0.5 día | Fase 5 |
| 7 | (P1) Endpoint backend `upload-many` | 1 día | Opcional |

### 6.2 Milestones

1. **MVP**: Selección múltiple + análisis secuencial + guardado en lote + dedup por MD5 (P0 completo).
2. **V1.0**: (Opcional) Endpoint batch `upload-many` para reducir round-trips.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Importación de muchos PDFs lenta (secuencial) | Alta | Bajo | Mostrar progreso por archivo; ~12-24 requests es aceptable en red local |
| Pico de memoria al analizar N PDF | Media | Alto | Análisis estrictamente secuencial; nunca Promise.all |
| Datos mal extraídos en algunos archivos | Media | Medio | Vista editable por archivo; errores individuales no bloquean al resto |
| Duplicados al importar un año completo | Baja | Alto | Upsert existente por (service_id, year, month) + dedup por file_hash |
| Mismo PDF re-subido tras editar datos extraídos | Media | Medio | Dedup por MD5 (independiente del periodo/valores) |

## 8. Notas y Referencias

- SPEC-014 (UI de subida de facturas con análisis automático)
- SPEC-021 (Sobrescribir bill existente cuando el analizador extrae datos — upsert por periodo)
- Archivos: `frontend/src/components/UploadBillModal.tsx`, `frontend/src/api/index.ts`, `internal/api/document_handlers.go`, `internal/services/document.go`, `internal/storage/bill.go`, `internal/models/bill.go`, `migrations/0014_*`
- MD5 para dedup (no seguridad): stdlib `crypto/md5`; alternativa `crypto/sha256` descartada por costo extra sin beneficio para este caso.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-16 | paulomcnally | Creación inicial de la especificación |
| 2026-08-16 | paulomcnally | Cambio de estado a pending_execution |
| 2026-08-16 | paulomcnally | Implementación: migración 0014 (file_hash + índice único parcial), backend dedup MD5 (TeeReader), flag duplicate en from-extracted, UploadBillModal multi-archivo (selección múltiple, análisis secuencial, resultados editables, guardado en lote), tests de migración/dedup/hash. Cambio de estado a in_progress |
| 2026-08-17 | paulomcnally | Validación manual del usuario satisfactoria. Criterios de aceptación P0 y NF-001..003 marcados como pass. Cambio de estado a pending_release |