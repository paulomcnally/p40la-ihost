---
title: "Historial de cambios de facturas (auditoría)"
id: "SPEC-070"
status: "released"
author: "paulomcnally"
created: "2026-09-11"
updated: "2026-09-11"
github_issue: 73
---

# Historial de cambios de facturas (auditoría)

**ID**: SPEC-070  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-11  
**Actualizado**: 2026-09-11

---

## 1. Resumen Ejecutivo

Las facturas del sistema se crean y modifican desde múltiples orígenes: el dashboard (formularios de alta/edición, pago, subida/análisis de documentos, generación automática) y los webhooks por servicio (SPEC-069). Hoy no existe trazabilidad de quién o qué cambió una factura: solo se ve el valor final con `updated_at`. Esto impide auditar errores (por ejemplo, un webhook que sobrescribe un monto con 0) o reconstruir el historial de un período.

Esta spec agrega un historial de auditoría por factura: cada evento registra la acción (`created`, `updated`, `paid`), la fuente (`dashboard` o `webhook`) y el detalle del cambio (campos modificados con valor anterior y nuevo, en JSON). Desde el menú de 3 puntos de cada factura se accede a una nueva sección **Historial** que abre un modal con la línea de tiempo de cambios.

Consideraciones de iHost: la solución usa una tabla SQLite ligera (`bill_history`) con un índice por `bill_id`; el volumen de datos es bajo (una fila por evento de factura). No agrega dependencias nuevas ni procesos en background. El modal solo carga el historial on-demand (`GET /api/bills/{id}/history`).

Consideraciones de UI obligatorias: el modal es un componente reutilizable con `bg-card`, texto con `text-text`/`text-text-secondary`; los inputs no aplican aquí (modal de solo lectura), pero el contenedor debe verificarse en darkmode.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear la tabla `bill_history` con migración SQL (up/down) que persista: `id`, `bill_id`, `action` (`created`/`updated`/`paid`), `source` (`dashboard`/`webhook`), `changes` (JSON con campos `field`, `old`, `new`) y `created_at`. Índice por `bill_id`.
2. **REQ-002**: Registrar el evento `created` con fuente correspondiente cuando una factura se crea desde el dashboard (formulario) o desde un webhook (SPEC-069). La creación por analizador de documentos y por generación automática se registra como fuente `dashboard`.
3. **REQ-003**: Registrar el evento `updated` con un diff de campos (monto, número de factura, estado, drive_url, año, mes, referencia de pago, fecha de pago) cuando una factura existente se modifica desde el dashboard o el webhook. Si no hay cambios reales (mismos valores), no registrar evento.
4. **REQ-004**: Registrar el evento `paid` cuando una factura se marca como pagada (dashboard `POST /api/bills/{id}/pay` o webhook con `status: paid`), incluyendo fecha de pago y referencia en el diff cuando aplique.
5. **REQ-005**: Exponer `GET /api/bills/{id}/history` (autenticado) que devuelve los eventos de auditoría de una factura ordenados por `created_at` descendente.
6. **REQ-006**: En el menú de 3 puntos de cada factura (móvil y desktop, `BillsPage`), agregar la opción **Historial** que abre un modal de solo lectura con la línea de tiempo de eventos: acción, fuente, fecha/hora y detalle de cada cambio (campo: anterior → nuevo).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Mostrar los cambios con etiquetas legibles (i18n es/en): "Creada", "Editada", "Pagada", "Dashboard", "Webhook", nombres de campo amigables (Monto, Nº factura, Estado, Enlace Drive, Año, Mes, Referencia, Fecha de pago).
2. **REQ-008**: Registrar también el evento `updated` (con diff) cuando el analizador de documentos sobrescribe una factura existente (`UpdateFromExtracted`), con fuente `dashboard`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Registrar el evento `deleted` (soft delete) con fuente para auditar bajas de facturas.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El registro de historial es síncrono en el mismo flujo de escritura; debe sumar overhead despreciable (una consulta INSERT por evento). La lectura del historial es on-demand y acotada por factura (índice `idx_bill_history_bill_id`).
- **Seguridad**: El endpoint de historial usa el middleware `authMiddleware` existente (mismas protecciones que el resto de la API de facturas). No expone datos sensibles adicionales.
- **Almacenamiento**: Fila por evento; un JSON de diff por fila de tamaño acotado (pocos campos). Sin rotación necesaria a escala esperada.
- **Disponibilidad**: Sin impacto: el registro de historial falla si falla la escritura (transacción con el cambio principal cuando aplica).
- **iHost**: Solo SQLite + Go stdlib. Sin dependencias nuevas, sin consumo extra de RAM (no hay workers).

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

Se revisó el código actual de facturas en el worktree:

- `internal/storage/bill.go`: funciones `Create`, `Update`, `UpdateFromExtracted`, `UpdateWebhookFields`, `Pay`, `MarkPending`, `SoftDelete`. Todas son puntos de escritura sobre `bills`.
- `internal/services/bill.go`: `BillService.Create/Update/PayBill/Delete` (fuente dashboard).
- `internal/services/webhook.go`: `WebhookService.UpsertBill` (fuente webhook; crea, actualiza con `UpdateWebhookFields`, paga con `Pay`, revierte con `MarkPending`).
- `internal/api/bill_handlers.go` y `routes.go`: rutas REST actuales (`GET/POST/PUT /api/bills...`, `POST /api/bills/{id}/pay`).
- `frontend/src/pages/BillsPage.tsx`: menú de 3 puntos (`billMenuOptions`) con Editar / Pagar / Eliminar.
- Migraciones SQL manuales: `migrations/NNNN_*.up.sql`/`.down.sql` aplicadas por `internal/db/db.go` en orden. Última migración: `0027`.
- i18n: `frontend/public/i18n/{es,en}.json` (fuente de verdad; se reconstruye con `npm run build`).

No existe ninguna tabla de auditoría/bitácora previa en el proyecto.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Tabla `bill_history` propia + registro síncrono en el service layer | Simple, SQLite nativo, diffs precisos calculados en Go, cero dependencias | Requiere tocar los flujos de escritura existentes | ✅ Seleccionada |
| Tabla genérica `audit_log` con entidad/campo/valor | Reutilizable para otras entidades a futuro | Difícil de mostrar "diff" por factura, más indirección | ❌ Rechazada |
| Triggers SQL que copien filas viejas | No toca código Go | No captura la fuente (dashboard/webhook) fácilmente, diffs complejos en SQL, menos legible | ❌ Rechazada |
| Paquete externo de auditoría/event sourcing | Potente | Dependencia pesada, viola la regla de mínimas dependencias para iHost | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Registrar historial en la capa de servicios (no en storage).
- **Contexto**: Las funciones de storage son compartidas entre dashboard y webhook; en storage no se conoce la fuente.
- **Decisión**: El registro de eventos se hace en `BillService` y `WebhookService`, que saben la fuente; se agrega un método `BillHistoryStorage.Record(...)`. El diff se calcula en el service comparando la factura existente (antes del UPDATE) con la nueva (después).
- **Consecuencias**: Cada punto de escritura se toca explícitamente (cambio acotado), pero se mantiene la fuente correcta y el diff fiel.

**ADR-002**: Almacenar el diff como JSON en una columna `changes`.
- **Contexto**: Se quiere mostrar "qué cambió" de forma legible y auditable.
- **Decisión**: `changes` es `TEXT` con un JSON array de `{"field": "...", "old": ..., "new": ...}`. El frontend lo renderiza con etiquetas i18n. No se parsea en el backend (solo serialización).
- **Consecuencias**: Almacenamiento compacto, sin esquema rígido; si se agregan campos a `bills` en el futuro, el diff los incluye sin migración extra.

**ADR-003**: Fuentes discretas `dashboard` y `webhook`.
- **Contexto**: El usuario pide distinguir la fuente de la edición (dashboard o webhook).
- **Decisión**: Enum `source` con `dashboard` (UI, analizador de documentos, generación automática) y `webhook` (SPEC-069). Se define la constante `SourceDashboard`/`SourceWebhook` en Go.
- **Consecuencias**: Trazabilidad clara del problema reportado (webhook sobrescribiendo montos); si a futuro hay más fuentes, se agrega valor al enum.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Dashboard (BillService)]  ──┐
                             ├──▶ [BillHistoryStorage.Record] ──▶ [SQLite: bill_history]
[Webhook (WebhookService)] ──┘
                                      ▲
[GET /api/bills/{id}/history] ────────┘  (auth)
                                      │
                          [Frontend: BillHistoryModal]
```

### 4.2 Componentes

#### 4.2.1 `BillHistoryStorage`
- **Responsabilidad**: Persistir y listar eventos de auditoría de facturas.
- **Interfaz**:
  - `Record(ctx, event *models.BillHistory) (*models.BillHistory, error)`
  - `ListByBill(ctx, billID int64) ([]models.BillHistory, error)`
- **Dependencias**: `*sql.DB`.
- **Ubicación**: `internal/storage/bill_history.go`.

#### 4.2.2 Modelo `BillHistory`
- **Responsabilidad**: Entidad de dominio de un evento.
- **Campos**: `ID`, `BillID`, `Action` (`created`|`updated`|`paid`), `Source` (`dashboard`|`webhook`), `Changes` (`[]FieldChange`), `CreatedAt`.
- **Ubicación**: `internal/models/bill_history.go`.

#### 4.2.3 `BillService` (modificado)
- **Responsabilidad**: Registrar eventos con fuente `dashboard` en `Create`, `Update`, `PayBill` y (P1) en el flujo del analizador que sobrescribe facturas existentes.
- **Ubicación**: `internal/services/bill.go`.

#### 4.2.4 `WebhookService` (modificado)
- **Responsabilidad**: Registrar eventos con fuente `webhook` en `UpsertBill` (creación, actualización, pago y reversión).
- **Ubicación**: `internal/services/webhook.go`.

#### 4.2.5 `BillHandlers` (modificado)
- **Responsabilidad**: Agregar `GetBillHistory` para `GET /api/bills/{id}/history`.
- **Ubicación**: `internal/api/bill_handlers.go` + ruta en `internal/api/routes.go`.

#### 4.2.6 `BillHistoryModal` (nuevo)
- **Responsabilidad**: Modal de solo lectura con la línea de tiempo de eventos de una factura.
- **Ubicación**: `frontend/src/components/BillHistoryModal.tsx`.
- **Dependencias**: `api.bills.history`, `useI18nStore`, tokens del tema.

### 4.3 Modelo de datos

Migración `migrations/0028_create_bill_history.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS bill_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bill_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('created', 'updated', 'paid')),
    source TEXT NOT NULL CHECK (source IN ('dashboard', 'webhook')),
    changes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bill_id) REFERENCES bills(id)
);

CREATE INDEX IF NOT EXISTS idx_bill_history_bill_id ON bill_history(bill_id);
```

`.down.sql`:
```sql
DROP TABLE IF EXISTS bill_history;
```

`internal/models/bill_history.go`:
```go
type FieldChange struct {
    Field string `json:"field"`
    Old   any    `json:"old,omitempty"`
    New   any    `json:"new,omitempty"`
}

type BillHistory struct {
    ID        int64         `json:"id"`
    BillID    int64         `json:"bill_id"`
    Action    string        `json:"action"`   // created | updated | paid
    Source    string        `json:"source"`   // dashboard | webhook
    Changes   []FieldChange `json:"changes,omitempty"`
    CreatedAt time.Time     `json:"created_at"`
}
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/bills/{id}/history` (auth)

**Response 200**:
```json
[
  {
    "id": 42,
    "bill_id": 7,
    "action": "updated",
    "source": "webhook",
    "changes": [
      { "field": "amount", "old": 350.5, "new": 0 }
    ],
    "created_at": "2026-09-11T16:00:00Z"
  }
]
```

**Response Error**:
```json
{ "error": "not_found", "message": "Factura no encontrada" }
```
- `404` si la factura no existe.
- `401` sin sesión (middleware existente).

### 4.5 Dependencias

- **Internas**: `BillService`, `WebhookService`, `BillHandlers`, `routes.go`, `internal/db/db.go` (migración automática por directorio), `frontend/src/pages/BillsPage.tsx`, `frontend/src/api/index.ts`, `frontend/public/i18n/{es,en}.json`, `frontend/src/types/index.ts`.
- **Externas**: Ninguna. Go stdlib (`encoding/json`, `database/sql`) y React existente.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Crear una factura desde el dashboard registra un evento `created` con `source: dashboard` en `bill_history`.
- [ ] CA-002: Un webhook con `status: pending` que crea una factura registra `created` con `source: webhook`.
- [ ] CA-003: Editar una factura desde el formulario (cambiar monto y/o número de factura) registra `updated` con `source: dashboard` y un diff con campo, valor anterior y nuevo.
- [ ] CA-004: Un webhook con `status: paid` sobre una factura existente registra `updated`/`paid` con `source: webhook` y el diff de monto/número/fecha de pago.
- [ ] CA-005: Marcar una factura como pagada desde el dashboard registra `paid` con `source: dashboard`.
- [ ] CA-006: Si un webhook reenvía los mismos valores (sin cambios reales), NO se crea un evento `updated` (el diff está vacío → se omite).
- [ ] CA-007: `GET /api/bills/{id}/history` devuelve los eventos ordenados por fecha descendente, con autenticación obligatoria y `404` si la factura no existe.
- [ ] CA-008: En `BillsPage` (móvil y desktop), el menú de 3 puntos de cada factura tiene la opción **Historial** que abre `BillHistoryModal` mostrando acción, fuente, fecha y detalle de cambios.
- [ ] CA-009: El modal se verifica en darkmode: texto y badges legibles (tokens `bg-card`, `text-text`, `text-text-secondary`). (CA-DARK)
- [ ] CA-010: No se crean links "← Título" dentro del contenido; la navegación de `BillsPage` no cambia (es una página de listado raíz, mantiene hamburguesa). (CA-BACK)

### 5.2 No funcionales

- [ ] CA-NF-001: El overhead del registro de historial es una única INSERT por evento, síncrona y despreciable frente al UPDATE principal.
- [ ] CA-NF-002: No se agregan dependencias nuevas; todo es Go stdlib y SQLite.

### 5.3 Testing

- **Unit tests** (`internal/services/bill_history_test.go`, `webhook_test.go`):
  - Registro de `created` con fuente correcta.
  - Diff de campos al editar (solo campos modificados, sin no-ops).
  - Omisión de evento cuando no hay cambios.
  - `paid` registrado desde dashboard y webhook.
- **Integration tests** (`internal/api/bill_handlers_test.go`):
  - `GET /api/bills/{id}/history` con auth → 200 y eventos ordenados.
  - Sin auth → 401.
  - Factura inexistente → 404.
- **Manual E2E**:
  - Crear/editar/pagar desde dashboard y ver historial en el modal.
  - Enviar webhook (pendiente y pagado) y ver eventos con fuente `webhook` en el modal.
  - Reenviar webhook idéntico → no genera evento nuevo.
  - Darkmode: modal legible.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0028_create_bill_history` (up/down) + modelo `BillHistory` + `BillHistoryStorage` | 0.5 día | Ninguna |
| 2 | Registro de eventos en `BillService` (create/update/pay) y `WebhookService` (upsert) con helper de diff | 1 día | Fase 1 |
| 3 | Endpoint `GET /api/bills/{id}/history` + ruta | 0.5 día | Fase 1 |
| 4 | Frontend: `BillHistoryModal`, opción en `billMenuOptions`, `api.bills.history`, tipos e i18n | 1 día | Fase 2, 3 |
| 5 | Tests unitarios/integración + pruebas manuales locales en darkmode | 1 día | Fase 2-4 |

### 6.2 Milestones

1. **MVP**: Backend completo (Fases 1-3) con historial persistido y endpoint.
2. **V1.0**: Frontend con modal de historial y tests verdes (Fases 4-5).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| No registrar algún punto de escritura (por ejemplo, analizador) | Media | Medio | Inventario de flujos de escritura en investigación; CA-003/004 cubren los principales; P1 REQ-008 cubre el analizador |
| Crecimiento de `bill_history` en el tiempo | Baja | Bajo | Volumen bajo (una fila por evento); índice por `bill_id`; lectura on-demand acotada por factura |
| Modificar flujos críticos (pago/webhook) introduce regresiones | Media | Alto | Registro síncrono en service layer con tests unitarios que cubren los flujos existentes de webhook y pago |
| El diff incluya campos irrelevantes (por ejemplo, `updated_at`) | Media | Baja | El helper de diff compara solo campos de negocio explícitos; no se incluyen metadatos |

## 8. Notas y Referencias

- SPEC-069: webhooks por servicio para facturas (flujo que es fuente `webhook`).
- SPEC-043: acción Pagar con fecha de pago, comprobante y referencia.
- `internal/db/db.go`: mecanismo de migraciones por archivos `.up.sql`/`.down.sql`.
- `frontend/public/i18n/`: fuente de verdad de traducciones (reconstruida por `npm run build`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-11 | paulomcnally | Creación inicial de la especificación |
| 2026-09-11 | paulomcnally | Spec cancelada a petición del usuario |
| 2026-09-11 | paulomcnally | Spec reactivada (cancelled → pending_execution → in_progress) para implementación a petición del usuario |
| 2026-09-11 | paulomcnally | Implementación completa: migración 0028, BillHistoryStorage, registro en BillService/WebhookService/DocumentService, endpoint GET /api/bills/{id}/history, BillHistoryModal en el menú de 3 puntos, i18n es/en. Tests y build verdes. |
| 2026-09-11 | paulomcnally | Release: merge a main (commit 7823a2b) |