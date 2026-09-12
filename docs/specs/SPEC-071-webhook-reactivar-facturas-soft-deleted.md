---
title: "Webhook: reactivar facturas soft-deleted en vez de fallar con UNIQUE"
id: "SPEC-071"
status: "released"
author: "paulomcnally"
created: "2026-09-12"
updated: "2026-09-12"
github_issue: 74
---

# Webhook: reactivar facturas soft-deleted en vez de fallar con UNIQUE

**ID**: SPEC-071  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-12  
**Actualizado**: 2026-09-12

---

## 1. Resumen Ejecutivo

El endpoint `POST /webhooks/{uuid}` (SPEC-069) deja de sincronizar ciertos periodos con
un `400 {"error":"invalid_body","message":"crear factura: insertar factura: constraint failed: UNIQUE constraint failed: bills.service_id, bills.year, bills.month (2067)"}`.
El problema se descubrió con la integración Claro Nicaragua del proyecto
`p40la-ihost-automation` (SPEC-004), que envía las facturas cada día. Los periodos
afectados (2026/08 y 2026/09 en la reproducción en vivo) quedan permanentemente
bloqueados: el emisor reenvía el mismo payload y recibe siempre el mismo 400.

La causa está en la combinación del **soft-delete** de `bills` con el índice
`UNIQUE(service_id, year, month)` a nivel de tabla. Cuando un usuario borra una factura
desde la UI (`DELETE /api/bills/{id}` → `SoftDelete`, setea `deleted_at`), la fila
sigue ocupando la clave única. `WebhookService.UpsertBill` busca con
`FindByServicePeriod` que filtra `deleted_at IS NULL`, no encuentra la factura, intenta
`INSERT` y choca con el UNIQUE. El `(2067)` del mensaje es el código extendido de
SQLite `SQLITE_CONSTRAINT_UNIQUE`, no un año del periodo.

El resultado esperado: el webhook es **resiliente al soft-delete** — ante un conflicto
de UNIQUE por una fila borrada lógicamente, reactiva/actualiza esa factura en lugar de
fallar, manteniendo la idempotencia documentada en `docs/webhooks-api.md` §Idempotencia.
Sin impacto en memoria ni dependencias nuevas (iHost).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: `WebhookService.UpsertBill` debe manejar el caso en que exista una factura **soft-deleted** para `(service_id, year, month)`: en lugar de propagar el 400 por UNIQUE, reactivar esa fila (limpiar `deleted_at`) y actualizarla con los campos del payload, respondiendo `200` con `created: true|false`.
2. **REQ-002**: `BillStorage` debe exponer un método para buscar por periodo **incluyendo filas soft-deleted** (sin el filtro `deleted_at IS NULL`) y otro para **reactivar** una factura (setear `deleted_at = NULL`).
3. **REQ-003**: No romper la idempotencia para facturas activas: reenviar un periodo existente no borrado sigue haciendo `UPDATE` (`created: false`).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Evaluar y, si se decide, migrar el esquema de `bills` a un índice **UNIQUE parcial** (`CREATE UNIQUE INDEX ... ON bills(service_id, year, month) WHERE deleted_at IS NULL`) como refuerzo estructural, con backfill de posibles duplicados. Si el fix de REQ-001 es suficiente, la migración puede omitirse.
2. **REQ-005**: Documentar el comportamiento en `docs/webhooks-api.md` (idempotencia y soft-delete).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-006**: El registro de historial de facturas (SPEC-070) debe reflejar la reactivación/actualización por webhook (acción `updated` o equivalente).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Una query adicional solo cuando `Create` falle por UNIQUE. Sin impacto en el flujo normal.
- **Seguridad**: Sin cambios en autenticación; se mantienen la api_key global y el contrato `X-Webhook-Key`.
- **Almacenamiento**: Sin tablas nuevas obligatorias; si se adopta REQ-004, una migración SQLite numerada que recrea la tabla o reemplaza la restricción por índice parcial.
- **Disponibilidad**: Reenviar un periodo fallido ya no queda bloqueado permanentemente; la próxima ejecución del job o el botón de test del emisor lo resuelven.
- **iHost**: Cero dependencias nuevas. Detección del error con `errors.Is(err, sqlite3.ErrConstraintUnique)` (mattn/go-sqlite3, ya en el proyecto) o fallback por string.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Reproducción en vivo** contra el webhook real (`http://192.168.1.191:8088/webhooks/bf05bab4-6421-42c5-b9c3-449dc6c1b421`, key provista por el usuario):
  - `year 2026 month 8` → `400 UNIQUE constraint ... (2067)`.
  - `year 2026 month 9` (nuevo) → `400` también.
  - `year 2025 month 1` → `200` creada (`id 31`, `service_id 3`).
  - `year 2025 month 2` → `200` creada (`id 32`) → el servicio es **mensual** (el mes no se normaliza a 0).
  - Reenviar `2025 month 1` → `200 created: false` (UPDATE correcto) → la idempotencia normal funciona.
  - `year 2026 months 1, 3, 10, 11, 12` (nunca enviados) → `200` creados.
  - `year 2026 months 4-7` (existentes) → `200 created: false` (UPDATE).
  - **Conclusión**: solo fallan los periodos con una fila **soft-deleted** en `bills` (2026/08 y 2026/09). Meses nuevos y periodos activos funcionan.
- **El `(2067)`** es el código extendido de SQLite `SQLITE_CONSTRAINT_UNIQUE` (2067), NO un año.
- Código involucrado:
  - `internal/services/webhook.go:126` → `FindByServicePeriod(service.ID, year, month)`; si `nil` → `Create` (`:145`) que falla con `crear factura: insertar factura: ...`.
  - `internal/storage/bill.go:51-59` → `FindByServicePeriod` filtra `AND deleted_at IS NULL`.
  - `internal/storage/bill.go:165-175` → `SoftDelete` setea `deleted_at = CURRENT_TIMESTAMP` (no borrado físico).
  - `internal/api/bill_handlers.go:126` → `DELETE /api/bills/{id}` → `SoftDelete`.
  - `migrations/0005_create_bills.up.sql` → `UNIQUE(service_id, year, month)` a nivel de tabla → cubre también filas soft-deleted.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Ante UNIQUE en `Create`, buscar la fila soft-deleted por periodo (sin filtro `deleted_at`) y reactivarla + actualizarla | Fix localizado, sin cambios de esquema; conserva `id` e historial | Requiere manejo del error 2067 y dos métodos nuevos en storage | ✅ Seleccionada |
| Índice UNIQUE parcial `WHERE deleted_at IS NULL` | Fix estructural: el soft-delete nunca más bloquea inserts | SQLite no permite alterar UNIQUE de tabla; requiere recrear tabla + backfill de duplicados | 🟡 Recomendada como refuerzo (REQ-004) |
| No reenviar periodos que fallaron (del lado emisor) | Evita el error | Pierde sincronización; el problema queda oculto | ❌ Rechazada (depende del cliente externo) |
| Solo documentar el comportamiento | Cero código | No resuelve el bloqueo | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El fix vive en el receptor (`p40la-ihost`), no en el emisor
- **Contexto**: La reproducción demuestra que el payload del emisor es válido y que el fallo es un conflicto de UNIQUE en `bills` por filas soft-deleted.
- **Decisión**: Corregir `WebhookService.UpsertBill` y `BillStorage` para recuperar y reactivar la fila soft-deleted cuando `Create` falle por `SQLITE_CONSTRAINT_UNIQUE`.
- **Consecuencias**: La integración vuelve a ser idempotente incluso tras borrar facturas desde la UI. El proyecto emisor `p40la-ihost-automation` solo mejora el registro del error (trackeado en su propio repo).

**ADR-002**: `(2067)` es el código de error de SQLite, no un año
- **Contexto**: El mensaje de error confundía al interpretar `2067` como un año del periodo.
- **Decisión**: Es `SQLITE_CONSTRAINT_UNIQUE = 2067`; el periodo real es el del payload (2026/08, 2026/09).
- **Consecuencias**: Permite detectar el caso de forma robusta en el código (`errors.Is(err, sqlite3.ErrConstraintUnique)`).

**ADR-003**: Detección del conflicto UNIQUE
- **Contexto**: `BillStorage.Create` devuelve un error envuelto; hay que distinguir el conflicto UNIQUE de otros errores.
- **Decisión**: El proyecto usa `modernc.org/sqlite` (no `mattn/go-sqlite3`). Detectar con `errors.As(err, *sqlite.Error)` y `Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE` (constante del paquete `modernc.org/sqlite/lib`), con fallback a `strings.Contains(err.Error(), "constraint failed")` por robustez.
- **Consecuencias**: Lógica de recuperación solo se dispara para el caso real (código SQLite 2067 = `SQLITE_CONSTRAINT_UNIQUE`).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
POST /webhooks/{uuid}  (X-Webhook-Key)
        |
        v
[WebhookHandlers.UpsertBill]  internal/api/webhook_handlers.go
        |
        v
[WebhookService.UpsertBill]   internal/services/webhook.go
        |
        v
[BillStorage.FindByServicePeriod(service, year, month)]  -- deleted_at IS NULL
        |
        ├─ existe → UPDATE (ok, created:false)
        └─ nil → Create (INSERT)
                 |
                 └─ UNIQUE (2067) ──→ [FindByServicePeriodIncludingDeleted]
                                         ├─ soft-deleted → Reactivate + Update → 200
                                         └─ no existe (race) → propagar error
```

### 4.2 Componentes

#### 4.2.1 WebhookService (`internal/services/webhook.go`)
- **Responsabilidad**: Orquestar el upsert del webhook.
- **Interfaz**: `UpsertBill(ctx, service, payload)` (existe).
- **Cambio**: En la rama `existing == nil`, si `Create` falla por `ErrConstraintUnique`, recuperar la fila soft-deleted y ejecutar la misma lógica de actualización que para `existing` (status paid/pending).
- **Ubicación**: `internal/services/webhook.go`.

#### 4.2.2 BillStorage (`internal/storage/bill.go`)
- **Responsabilidad**: Acceso a `bills`.
- **Cambio**: Nuevos métodos `FindByServicePeriodIncludingDeleted(ctx, serviceID, year, month)` (sin filtro `deleted_at`) y `Reactivate(ctx, id)` (`UPDATE bills SET deleted_at = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`).
- **Ubicación**: `internal/storage/bill.go`.

### 4.3 Modelo de datos

```
bills (sin cambios obligatorios para el fix REQ-001)
- UNIQUE(service_id, year, month)  ← cubre filas soft-deleted (causa del bug)
- deleted_at: DATETIME NULL        ← soft-delete de la UI

Opcional (REQ-004), migración numerada:
- CREATE UNIQUE INDEX idx_bills_service_year_month
  ON bills(service_id, year, month) WHERE deleted_at IS NULL
  + limpieza/backfill de duplicados soft-deleted
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /webhooks/{uuid}` (comportamiento corregido)

**Request**:
```json
{
  "year": 2026,
  "month": 8,
  "amount": 975.97,
  "invoice_number": "FAC0259424672026",
  "status": "pending"
}
```

**Response 200** (periodo soft-deleted reactivado):
```json
{
  "created": false,
  "bill": { "id": 7, "service_id": 3, "year": 2026, "month": 8, "amount": 975.97, "status": "pending" }
}
```

**Response Error** (sin cambios para errores reales):
```json
{ "error": "invalid_body", "message": "..." }
```

### 4.5 Dependencias

- **Internas**: `internal/services/webhook.go`, `internal/storage/bill.go`, `internal/models`, `internal/db`.
- **Externas**: **Ninguna nueva obligatoria**. Para detección por tipo se usa `github.com/mattn/go-sqlite3` (ya presente en el proyecto). Frontend: sin cambios.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un periodo con factura soft-deleted, cuando se reenvía `POST /webhooks/{uuid}`, entonces responde `200` y la factura vuelve a estar activa con los datos del payload.
- [x] CA-002: Dado un periodo nuevo, cuando se envía, entonces responde `200 created: true` (sin regresión).
- [x] CA-003: Dado un periodo existente no borrado, cuando se reenvía, entonces responde `200 created: false` (idempotencia intacta).
- [x] CA-004: Con `status=paid` en un periodo reactivado, la factura queda pagada con `paid_at`/`payment_reference` (si aplica).
- [x] CA-005: `BillStorage` expone `FindByServicePeriodIncludingDeleted` y `Reactivate`, con tests unitarios.
- [x] CA-BACK/CA-DARK: No aplican (no hay UI nueva).

### 5.2 No funcionales

- [x] CA-NF-001: `go build ./...` compila sin dependencias nuevas obligatorias.
- [x] CA-NF-002: Test de integración del webhook: payload duplicado sobre fila soft-deleted → 200 y fila activa; repetición → 200 sin duplicar.

### 5.3 Testing

- **Unit tests**: `FindByServicePeriodIncludingDeleted` (devuelve fila soft-deleted), `Reactivate` (limpia `deleted_at`), detección de `ErrConstraintUnique`.
- **Integration tests**: `UpsertBill` con fila soft-deleted pre-existente → reactiva y actualiza; con periodo nuevo → crea; con periodo activo → actualiza.
- **E2E**: Reenviar los periodos 2026/08 y 2026/09 contra el webhook real → `200`.
- **Carga/Performance**: Sin impacto (query extra solo en el camino de recuperación).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `BillStorage`: `FindByServicePeriodIncludingDeleted` + `Reactivate` + tests | 0.25 día | Ninguna |
| 2 | `WebhookService.UpsertBill`: detección de UNIQUE y recuperación/actualización de soft-deleted | 0.25 día | Fase 1 |
| 3 | Test de integración del webhook (soft-delete → 200; idempotencia intacta) | 0.25 día | Fase 2 |
| 4 | (Opcional REQ-004) Migración a índice UNIQUE parcial + backfill | 0.25 día | Fase 2 |
| 5 | (Opcional REQ-006) Historial de cambios para reactivación por webhook | 0.1 día | Fase 2 |
| 6 | Verificación en vivo contra el webhook real (2026/08 y 2026/09 → 200) + actualizar `docs/webhooks-api.md` | 0.1 día | Fase 3 |

### 6.2 Milestones

- **MVP**: Fases 1-3 + verificación en vivo: los periodos 2026/08 y 2026/09 responden `200` y se sincronizan.
- **V1.0**: REQ-004 (índice UNIQUE parcial) si se decide como refuerzo estructural.

### 6.3 Decisión de alcance (REQ-004)

**REQ-004 queda fuera del alcance de esta iteración.** El fix de REQ-001 (recuperación y reactivación de la fila soft-deleted ante conflicto UNIQUE) es suficiente para resolver el bloqueo reportado: el webhook responde `200`, reactiva la factura y mantiene la idempotencia. La migración a un índice UNIQUE parcial requeriría recrear la tabla `bills` en la DB de producción del iHost (SQLite no permite alterar UNIQUE de tabla) con backfill de duplicados, un riesgo innecesario para un escenario ya resuelto. Queda documentado como refuerzo estructural futuro (REQ-004) en caso de que el patrón se repita.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Reactivar una fila soft-deleted conserva datos residuales (`paid_at`, `file_hash`) | Media | Medio | Actualizar campos descriptivos desde el payload y seguir la lógica de pago de SPEC-069 |
| Migración a índice UNIQUE parcial genera duplicados si ya hay soft-deleted del mismo periodo | Baja | Medio | Backfill/limpieza previa en la migración o descartar REQ-004 (REQ-001 es suficiente) |
| Detección del error 2067 por string frágil | Baja | Bajo | Usar `errors.Is(err, sqlite3.ErrConstraintUnique)` + fallback por string |
| Race entre dos envíos simultáneos del mismo periodo | Baja | Bajo | La recuperación re-consulta y actualiza; en el peor caso se propaga el error y el emisor reintenta |

## 8. Notas y Referencias

- Reproducción en vivo: `http://192.168.1.191:8088/webhooks/bf05bab4-6421-42c5-b9c3-449dc6c1b421` — payloads 2026/08 y 2026/09 → `400`; 2025/01, 2025/02, 2026/01, 2026/03, 2026/04-07, 2026/10-12 → `200`.
- Código: `internal/services/webhook.go`, `internal/storage/bill.go`, `internal/api/bill_handlers.go`, `migrations/0005_create_bills.up.sql`.
- Contrato webhook: `docs/webhooks-api.md` (§Idempotencia).
- Specs relacionadas: SPEC-069 (webhooks por servicio), SPEC-070 (historial de facturas), SPEC-043 (pago de facturas).
- Proyecto emisor: `p40la-ihost-automation` (SPEC-004 jobs de facturas; spec de diagnóstico cancelada allí, este repo es el de implementación).
- Código SQLite: `SQLITE_CONSTRAINT_UNIQUE = 2067`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-12 | paulomcnally | Creación inicial: error 400 `UNIQUE constraint (2067)` en webhook de Claro Nicaragua; reproducción en vivo; causa raíz (soft-delete + UNIQUE a nivel tabla vs `FindByServicePeriod` con `deleted_at IS NULL`); fix en el receptor: reactivar soft-deleted ante conflicto UNIQUE, con índice UNIQUE parcial opcional. Trasladada desde `p40la-ihost-automation/docs/specs/SPEC-006` (cancelada allí). |
| 2026-09-12 | p40la-ihost-team | Implementación: `BillStorage.FindByServicePeriodIncludingDeleted` + `Reactivate`; `WebhookService.UpsertBill` recupera y reactiva la fila soft-deleted ante `SQLITE_CONSTRAINT_UNIQUE` (modernc.org/sqlite); tests unitarios e integración; `docs/webhooks-api.md` actualizado. REQ-004 (índice UNIQUE parcial) fuera de alcance (REQ-001 suficiente). |
| 2026-09-12 | p40la-ihost-team | Release: pruebas manuales del usuario satisfactorias; criterios de aceptación pasan. Commit `d8271a2` (implementación) + merge a `main`. Estado `released`, issue #74 cerrado con label `spec/released`. |