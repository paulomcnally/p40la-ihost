---
title: "Acción Pagar en facturas con fecha de pago, comprobante y referencia"
id: "SPEC-043"
status: "released"
author: "p40la-ihost-team"
created: "2026-08-31"
updated: "2026-09-03"
github_issue: 43
---

# Acción Pagar en facturas con fecha de pago, comprobante y referencia

**ID**: SPEC-043  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-08-31  
**Actualizado**: 2026-09-03

---

## 1. Resumen Ejecutivo

Las facturas de los servicios se crean por defecto con estado `pending` (pendiente) y solo pueden pasar a `paid` (pagada) editando la factura completa y cargando obligatoriamente un link de Google Drive como comprobante. Hoy no existe un flujo dedicado para marcar una factura como pagada, ni se registra la fecha en que se realizó el pago ni una referencia interna del mismo.

Este spec agrega la acción **Pagar** al menú de acciones (3 puntos) de cada factura en la página de listado, visible únicamente en facturas pendientes. Al hacer clic, se abre un modal que pide la **fecha del pago** (obligatoria), un **link de Google Drive del comprobante** (opcional, label "Comprobante") y una **referencia del pago** (opcional, texto libre, ej. número de transacción). Al confirmar, la factura pasa a `paid` y deja de aparecer en los emails de "facturas pendientes" (SPEC-031), en el badge de estado del listado y en el estado derivado de la card del servicio (SPEC-018).

Es una mejora de baja complejidad que no requiere dependencias externas nuevas. En iHost se mantiene el patrón actual: SQLite + Go + frontend estático React, sin consumo adicional relevante de memoria.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar la opción **Pagar** en el dropdown de acciones de cada factura en `BillsPage` (tanto en cards móviles como en la tabla desktop), ubicada **después de "Editar"** y antes de "Eliminar". El orden debe ser: Editar | Pagar | Eliminar.
2. **REQ-002**: La opción **Pagar** solo se muestra cuando la factura está en estado `pending`. Las facturas `paid` no muestran esta opción.
3. **REQ-003**: Al hacer clic en **Pagar** se abre un modal que solicita la **fecha en que se realizó el pago** (campo de fecha, obligatorio).
4. **REQ-004**: El modal permite ingresar un **link de Google Drive** del comprobante, **opcional**, cuyo label visible es **"Comprobante"**.
5. **REQ-005**: El modal permite ingresar una **referencia del pago** (texto libre, ej. número de transacción), **opcional**.
6. **REQ-006**: Al confirmar, la factura pasa a estado `paid`, se persisten `paid_at` (fecha de pago), `drive_url` (comprobante, si se cargó) y `payment_reference` (si se cargó).
7. **REQ-007**: La factura pagada deja de aparecer en los emails de resumen de facturas pendientes (SPEC-031) y en el badge/estado derivado de la card del servicio (SPEC-018). Esto se logra automáticamente al cambiar `status` a `paid`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: Si se ingresa un link de comprobante, debe validarse que sea una URL de Google Drive (`drive.google.com` / `docs.google.com`) usando el mismo criterio de validación existente en `internal/services/bill.go`.
2. **REQ-009**: El modal muestra la fecha de hoy como valor por defecto sugerido, editable por el usuario.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Mostrar la fecha de pago (y referencia si existe) en el detalle de la factura (badge o línea adicional) para feedback visual post-pago.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El flujo agrega un único `UPDATE` a SQLite; sin impacto perceptible. El modal no introduce re-renders costosos.
- **Seguridad**: El endpoint de pago debe requerir autenticación como el resto de la API. Validar en backend que la factura exista y no esté eliminada.
- **Almacenamiento**: Dos columnas nuevas en `bills` (`paid_at` datetime nullable, `payment_reference` text nullable). Sin impacto de tamaño relevante.
- **Disponibilidad**: Sin cambios en health checks ni en schedulers.
- **iHost**: Sin dependencias nuevas; mismo stack (Go stdlib + SQLite). Consumo de memoria despreciable.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `frontend/src/pages/BillsPage.tsx`: dos `CardMenu` (móvil líneas 100-105, desktop 167-172) con opciones Editar/Eliminar.
- `frontend/src/components/CardMenu.tsx`: componente reutilizable de menú de acciones con opciones `{ label, icon?, danger?, onClick }`.
- `internal/models/bill.go`: modelo `Bill` con `Status` (`pending`/`paid`) y `DriveURL`. No existen campos de fecha de pago ni referencia.
- `migrations/0005_create_bills.up.sql` y `0006_relax_month_constraint.up.sql`: tabla `bills` con columna `status` y `CHECK (status IN ('pending','paid'))`. No existe `paid_at` ni `payment_reference`.
- `internal/services/bill.go` (`validate()`, líneas 60-117): hoy **obliga** a cargar `drive_url` válido (Google Drive) cuando `status == "paid"`. Para que el comprobante sea opcional, esta validación debe relajarse.
- `internal/services/bill_summary_scheduler.go` + `internal/storage/bill.go` (`ListPendingWithDetails`, líneas 131-161): el email diario de pendientes filtra `WHERE b.status = 'pending'`. Marcar `paid` excluye automáticamente la factura del email.
- `internal/storage/service.go` (línea 24): `latest_bill_status` derivado en la card del servicio (SPEC-018) lee el estado de la factura más reciente; se actualiza automáticamente.
- `internal/api/bill_handlers.go` + `internal/api/routes.go` (líneas 62-71): rutas actuales `GET/POST/PUT/DELETE /api/bills...`. El `PUT /api/bills/{id}` recibe cuerpo completo y **no** propaga `file_hash` desde el request (lo resetea a vacío), por lo que reutilizarlo para "pagar" podría perder el hash de dedup.
- Frontend: no existe modal date-picker; hay `<input type="date">` nativo en `ServiceFormPage.tsx:323-334`. El patrón de overlay es `fixed inset-0 z-50 bg-black/40 ... rounded-ios shadow-ios`.
- i18n: fuente de verdad en `frontend/public/i18n/es.json` y `en.json` (nunca `public/i18n/`, es salida del build). Falta la clave `bills.pay`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Endpoint dedicado `POST /api/bills/{id}/pay`** | Flujo atómico; no resetea `file_hash`; valida pendiente→pagada; registro claro | Un handler nuevo | ✅ Seleccionada |
| Reusar `PUT /api/bills/{id}` con `status:"paid"` | Sin endpoint nuevo | Requiere cuerpo completo (riesgo de pisar datos), resetea `file_hash`, permite editar todo el flujo | ❌ Rechazada |
| Migración nueva (`0015_add_bills_paid_at_reference`) | Modelo de datos correcto para fecha/referencia | Una migración adicional | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Endpoint dedicado para pagar facturas**
- **Contexto**: El `PUT` actual de facturas no propaga `file_hash` (lo resetea) y exige cuerpo completo, lo que complica un flujo de "pagar" simple y arriesga datos.
- **Decisión**: Crear `POST /api/bills/{id}/pay` que recibe `{ paid_at, drive_url?, payment_reference? }`, valida que la factura esté `pending`, y persiste `status='paid'`, `paid_at`, `drive_url`, `payment_reference`.
- **Consecuencias**: Handler nuevo pequeño; flujo aislado y seguro. El formulario de edición sigue usando PUT.

**ADR-002: Relajar validación de `drive_url` para estado `paid`**
- **Contexto**: Hoy `validate()` obliga a `drive_url` válido cuando `status == "paid"`. El nuevo requerimiento hace el comprobante opcional.
- **Decisión**: Mantener la validación de formato (regex Google Drive) **solo cuando** `drive_url` esté presente; ya no será obligatorio al pagar.
- **Consecuencias**: Facturas pagadas sin comprobante quedan permitidas (decisión de producto). Si se ingresa una URL inválida, sigue rechazándose.

**ADR-003: Persistir fecha y referencia en columnas nuevas**
- **Contexto**: No existen `paid_at` ni `payment_reference` en `bills`; el estado por sí solo no registra cuándo ni cómo se pagó.
- **Decisión**: Nueva migración `0015_add_bills_paid_at_payment_reference.up.sql` agregando ambas columnas nullable a `bills`.
- **Consecuencias**: Lecturas/escrituras existentes siguen funcionando (columnas nullable); el modelo `Bill` y el DTO de la API se amplían.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[BillsPage] --click "Pagar"--> [PayBillModal (fecha + comprobante + referencia)]
      |                                  |
      v                                  v
[CardMenu (solo status=pending)]   [POST /api/bills/{id}/pay]
                                           |
                                           v
                              [services.BillService.PayBill]
                                           |
                                           v
                                    [storage: UPDATE bills
                                      SET status='paid', paid_at=?,
                                          drive_url=?, payment_reference=?]
                                           |
                                           v
                     [email pendientes SPEC-031 excluye factura]
                     [card servicio SPEC-018 badge verde]
```

### 4.2 Componentes

#### 4.2.1 `PayBillModal` (nuevo, `frontend/src/components/PayBillModal.tsx`)
- **Responsabilidad**: Capturar fecha de pago (obligatoria), link de comprobante Google Drive (opcional, label "Comprobante") y referencia del pago (opcional).
- **Interfaz**: Props `{ bill: Bill, onClose: () => void, onSuccess: () => void }`.
- **Dependencias**: `api.bills.pay`, `useToast`, patrón de overlay existente, `useState` para el formulario local.
- **Ubicación**: `frontend/src/components/PayBillModal.tsx`.

#### 4.2.2 `BillsPage` (modificar, `frontend/src/pages/BillsPage.tsx`)
- **Responsabilidad**: Agregar opción "Pagar" en ambos `CardMenu` solo cuando `bill.status === 'pending'`, en orden Editar | Pagar | Eliminar.
- **Interfaz**: Sin cambios de contrato; agrega estado local `payTarget: Bill | null` y renderiza `PayBillModal`.
- **Dependencias**: `PayBillModal`, i18n `bills.pay`.

#### 4.2.3 `POST /api/bills/{id}/pay` (nuevo handler)
- **Responsabilidad**: Validar y ejecutar el pago.
- **Interfaz**: Request `{ paid_at: string (RFC3339 o YYYY-MM-DD), drive_url?: string, payment_reference?: string }`. Response `{ status: "paid", paid_at, drive_url, payment_reference }`.
- **Dependencias**: `BillService.PayBill`, storage.

#### 4.2.4 `BillService.PayBill` (nuevo método, `internal/services/bill.go`)
- **Responsabilidad**: Lógica de negocio del pago: verificar que la factura existe, está `pending` y no eliminada; validar `paid_at` requerido; validar formato de `drive_url` si viene; persistir.
- **Interfaz**: `PayBill(ctx, id, paidAt time.Time, driveURL, reference string) (*models.Bill, error)`.
- **Dependencias**: `BillStorage`, `validate()` (relajado).

#### 4.2.5 Cliente API frontend (`frontend/src/api/index.ts`)
- **Responsabilidad**: Agregar `bills.pay(id, payload)`.
- **Interfaz**: `pay(id: number, data: { paid_at: string; drive_url?: string; payment_reference?: string }): Promise<Bill>`.
- **Dependencias**: `http` helper existente.

### 4.3 Modelo de datos

```
Entidad: Bill (bills)
- status: text ('pending'|'paid') — existe
- paid_at: datetime NULL (nuevo) — fecha en que se realizó el pago
- payment_reference: text NULL (nuevo) — referencia interna del pago (ej. nro de transacción)
- drive_url: text NULL — comprobante Google Drive (ahora opcional incluso con status='paid')
- Relaciones: Service (N:1)
```

Migración nueva: `migrations/0015_add_bills_paid_at_payment_reference.up.sql`

```sql
ALTER TABLE bills ADD COLUMN paid_at DATETIME;
ALTER TABLE bills ADD COLUMN payment_reference TEXT;
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/bills/{id}/pay`

**Request**:
```json
{
  "paid_at": "2026-08-31",
  "drive_url": "https://drive.google.com/file/d/abc123/view",
  "payment_reference": "TRX-001234"
}
```

**Response 200**:
```json
{
  "id": 42,
  "service_id": 7,
  "status": "paid",
  "paid_at": "2026-08-31",
  "drive_url": "https://drive.google.com/file/d/abc123/view",
  "payment_reference": "TRX-001234"
}
```

**Response Error**:
```json
{
  "error": "invalid_status",
  "message": "la factura ya está pagada"
}
```
- `400` si `paid_at` falta o no es fecha válida, si `drive_url` no cumple el formato Google Drive, o si la factura ya está `paid`.
- `404` si la factura no existe o está eliminada.

### 4.5 Dependencias

- **Internas**: `internal/api/bill_handlers.go`, `internal/api/routes.go`, `internal/services/bill.go`, `internal/storage/bill.go`, `internal/models/bill.go`, `frontend/src/pages/BillsPage.tsx`, `frontend/src/api/index.ts`, `frontend/src/types/index.ts`, `frontend/src/components/CardMenu.tsx` (sin cambios), `frontend/public/i18n/{es,en}.json`.
- **Externas**: Ninguna. Sigue stack Go stdlib + SQLite + React/Tailwind build estático.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: En `BillsPage`, una factura `pending` muestra el menú con opciones **Editar | Pagar | Eliminar** en ese orden (móvil y desktop).
- [x] CA-002: Una factura `paid` **no** muestra la opción "Pagar".
- [x] CA-003: Al hacer clic en "Pagar" se abre un modal con campo de fecha (por defecto hoy, editable), campo "Comprobante" (link Google Drive, opcional) y campo "Referencia" (opcional).
- [x] CA-004: Confirmar sin fecha de pago muestra error y no persiste.
- [x] CA-005: Confirmar con fecha y sin comprobante ni referencia marca la factura como `paid` correctamente.
- [x] CA-006: Confirmar con comprobante y referencia persiste `drive_url` y `payment_reference`, y el badge muestra "Pagada".
- [x] CA-007: Si se ingresa comprobante, solo acepta URLs de `drive.google.com` / `docs.google.com` (rechaza otras URLs con toast de error).
- [x] CA-008: La factura pagada desaparece del próximo email de "Resumen de facturas pendientes" (SPEC-031) sin cambios en el scheduler.
- [x] CA-009: La card del servicio (SPEC-018) refleja el estado `paid` de la factura más reciente automáticamente.
- [x] CA-010: Los textos nuevos (`bills.pay`, labels del modal) aparecen en ES y EN tras `npm run build` y se sirven en `/i18n/{lang}.json`.

### 5.2 No funcionales

- [x] CA-NF-001: El endpoint `POST /api/bills/{id}/pay` requiere autenticación.
- [x] CA-NF-002: El flujo completo no agrega dependencias nuevas y mantiene el consumo de RAM del iHost sin cambios perceptibles.

### 5.3 Testing

- **Unit tests**: `PayBill` en `internal/services/bill_test.go`: factura pending→paid persiste campos; factura ya `paid` → error; `paid_at` faltante → error; `drive_url` inválida → error; `drive_url` vacía/ausente → OK.
- **Storage tests**: `internal/storage/bill_pay_test.go` (o ampliar los existentes): verifica persistencia de `paid_at`, `payment_reference`, `drive_url` y que `ListPendingWithDetails` ya no devuelve la factura pagada.
- **Integration tests**: `POST /api/bills/{id}/pay` happy path + errores (400/404).
- **E2E tests (manual)**: En `BillsPage` (móvil y desktop) verificar visibilidad de "Pagar" solo en pendientes, modal, validaciones y refresh del listado.
- **Carga/Performance**: Verificar en iHost que el `UPDATE` y el refresh no degradan la respuesta.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0015_add_bills_paid_at_payment_reference` + actualización de `models.Bill`, storage y servicios (`PayBill`, relajar validación `drive_url`) | 0.5 días | Ninguna |
| 2 | Endpoint `POST /api/bills/{id}/pay` + tests de service/storage | 0.5 días | Fase 1 |
| 3 | Frontend: `PayBillModal`, opción "Pagar" en `BillsPage` (Editar|Pagar|Eliminar, solo pending), `api.bills.pay`, tipos | 1 día | Fase 2 |
| 4 | i18n (`frontend/public/i18n/{es,en}.json`) + `npm run build` + verificación de claves servidas | 0.5 días | Fase 3 |
| 5 | Pruebas locales (tests + server + validación manual con usuario) | 1 día | Fase 4 |

### 6.2 Milestones

1. **MVP**: Pagar factura con fecha obligatoria + comprobante y referencia opcionales; exclusión automática de emails y badges.
2. **V1.0**: Feedback visual de fecha de pago/referencia en la factura (REQ-010, si aplica).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Relajar `drive_url` obligatorio podría dejar facturas pagadas sin comprobante | Media | Bajo | Es decisión de producto (comprobante opcional); se mantiene validación de formato si se ingresa |
| `PUT /api/bills/{id}` podría sobrescribir `paid_at`/`payment_reference` con valores vacíos en ediciones posteriores | Media | Medio | Al editar, propagar los campos existentes en el form/DTO; o ignorar vacíos en `validate/update` para estos campos |
| Editar i18n en `public/i18n/` en vez de `frontend/public/i18n/` (fallo recurrente documentado en AGENTS.md) | Media | Alto | Editar SOLO `frontend/public/i18n/` y correr `npm run build` |
| Confundir `paid_at` (fecha de pago) con `created_at` | Baja | Bajo | Documentar claramente en modelo y UI (label "Fecha de pago") |

## 8. Notas y Referencias

- SPEC-018: Estado de pago dinámico en cards de servicios (campo derivado `latest_bill_status`).
- SPEC-030: Email al generar factura automática (crea facturas con `status='pending'`).
- SPEC-031: Resumen diario de facturas pendientes (`ListPendingWithDetails` filtra por `status='pending'`).
- `frontend/src/components/CardMenu.tsx`: patrón de menú de acciones a extender.
- AGENTS.md: reglas de i18n (fuente de verdad en `frontend/public/i18n/`) y flujo de specs.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-08-31 | p40la-ihost-team | Creación inicial de la especificación. Se agrega además el campo opcional "Referencia del pago" según pedido del usuario. |
| 2026-08-31 | p40la-ihost-team | Cambio de estado a `in_progress` para inicio de desarrollo. |
| 2026-09-03 | p40la-ihost-team | Cambio de estado a `released`. Feature mergeada a `main` (merge `4218991`, fix i18n `ac97a09`); disponible desde v0.4.12/v0.4.14. Issue #43 cerrado. |