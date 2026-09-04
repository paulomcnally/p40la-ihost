---
title: "Módulo Deudas: CRUD con generación automática de cuotas y vista Calendario"
id: "SPEC-054"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 54
---

# Módulo Deudas: CRUD con generación automática de cuotas y vista Calendario

**ID**: SPEC-054  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

El usuario necesita administrar sus deudas (préstamos, tarjetas de crédito) desde la app. Hoy solo se gestionan servicios con facturas recurrentes y no existe ninguna entidad de deudas en el repositorio. Este spec agrega un menú **Deudas** en el Sidebar con un CRUD completo: cada deuda guarda acreedor, identificador, descripción, totales, moneda, número de cuotas, tasa de interés, día de pago, fecha de inicio y estado.

Lo novedoso es que **cada deuda genera su propio sistema de cobranza de inicio a fin** según su configuración: al crear/activar una deuda se genera automáticamente la serie completa de cuotas (facturas) con su fecha de vencimiento, reutilizando la misma lógica de estados (`pending`/`paid`) que ya tienen las facturas de servicios (SPEC-008/018/043). En lugar de listar deudas solo en cards, la página Deudas tiene **dos pestañas sincronizadas con la URL**: **Calendario** (vista mensual donde se reflejan las cuotas generadas por fecha de vencimiento, con detalle por día y estado de pago) y **Deudas** (listado en cards con el patrón existente).

Consideraciones iHost: sin nuevas dependencias (SQL nativo con `database/sql`), una tabla nueva de cuotas con índices simples, y generación de cuotas acotada (N cuotas, no indefinida) para no impactar memoria ni CPU. La lógica de generación replica el patrón del `BillingScheduler`/`ReconcileBills` ya probado.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Menú **Deudas** en el Sidebar (nivel superior, como Servicios/Autos) con icono y título i18n.
2. **REQ-002**: Acción de creación **"Nueva Deuda"** vía `CreateMenu` (3 puntos) en el header de la página Deudas.
3. **REQ-003**: CRUD completo de deudas (crear, listar, editar, eliminar con modal de confirmación).
4. **REQ-004**: Campos del formulario de deuda:
   - **Acreedor** — `Select` (dropdown, searchable) desde la lista de instituciones existentes.
   - **Identificador** — texto libre (ej: "1111 2222 3333 4444").
   - **Descripción** — texto libre (ej: "Tarjeta de crédito").
   - **Total** — numérico decimal ≥ 0 (ej: 0.00).
   - **Capital Prestado** — numérico decimal ≥ 0 (ej: 0.00).
   - **Moneda** — `Select` desde la tabla `currencies` (ej: NIO).
   - **Total de Cuotas** — entero ≥ 1 (ej: 12).
   - **Por Cuota** — numérico decimal ≥ 0 (ej: 0.00; si es 0, se calcula `Total / Total de Cuotas`).
   - **Tasa de Interés (%)** — numérico decimal ≥ 0 (ej: 0.00).
   - **Día de Pago del Mes** — entero 1-31 (ej: 15).
   - **Fecha de Inicio** — date picker nativo (`<input type="date">`, formato mm/dd/yyyy).
   - **Estado** — `Select` con `activa` (default) / `inactiva` / `finalizada`.
5. **REQ-005**: **Validación de prerrequisitos** en backend y frontend: si no existen instituciones, redirigir a `/institutions/new`; si no existen monedas, redirigir a `/settings/currency` (patrón `ServiceFormPage`/`DependencyWarning`).
6. **REQ-006**: **Generación automática de cuotas**: al crear/editar una deuda en estado `activa`, generar la serie completa de N cuotas (una por mes a partir del mes siguiente de la fecha de inicio, vencimiento en el día de pago del mes, monto = Por Cuota). Deduplicación por `(debt_id, installment_number)`. Si la deuda pasa a `inactiva`, no generar nuevas cuotas; si vuelve a `activa`, regenerar las faltantes. Reconcile al listar deudas (patrón `ReconcileBills`).
7. **REQ-007**: **Estados de cuota idénticos a facturas de servicios**: `pending` (default) / `paid`, con acción **Pagar** por cuota (fecha de pago + referencia, patrón SPEC-043 / `PayBillModal`).
8. **REQ-008**: **Página Deudas con pestañas que afectan la URL**: pestaña **Calendario** (default) y pestaña **Deudas**, sincronizadas vía query param `?tab=calendario|deudas` (`useSearchParams`, patrón `RegistrosPage`).
9. **REQ-009**: **Pestaña Calendario**: vista mensual (navegación prev/next mes) donde cada día muestra marcadores de las cuotas vencidas ese día. Al hacer clic en un día, ver detalle de cuotas (descripción de deuda, monto, estado) con acción Pagar.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010**: Página de detalle de deuda `/deudas/:id` mostrando sus cuotas (listado con estado, monto, vencimiento y acción Pagar), replicando `BillsPage`.
2. **REQ-011**: En el calendario, poder filtrar por deuda específica (opcional) y/o incluir también facturas de servicios en la misma vista (P2 por defecto).
3. **REQ-012**: **Email diario agrupado de cuotas vencidas**: el día en que vence al menos una cuota (`due_date = hoy` y `status = pending`), se envía **UN SOLO email** que agrupa todas las cuotas de deudas que vencen ese día, con el desglose (descripción de deuda, número de cuota, monto) y el **total del día**. No se envía un email por cada cuota. **La hora de envío se basa en la hora de notificaciones del sistema (`alert_check_hour`)** y el envío se controla con un toggle de alerta configurable (`debt_due`). Reutiliza el sistema de emails/alerts existente (patrón `bill_summary_scheduler` de SPEC-031).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013**: Marcador visual de deuda "finalizada" cuando todas sus cuotas están pagadas (estado calculado).
2. **REQ-014**: Alerta multicanal (Voice Monkey) de cuotas vencidas o próximas a vencer reutilizando el sistema de alertas (`alert_service.go`).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Generación de cuotas O(N) con N acotado (≤ ~60). Queries de calendario indexadas por `due_date`. Respuestas API < 300ms en iHost.
- **Seguridad**: Endpoints protegidos con `authMiddleware` como el resto del API. Sin datos sensibles adicionales.
- **Almacenamiento**: Tabla `debt_bills` pequeña (una fila por cuota). Sin rotación necesaria.
- **Disponibilidad**: Health check `/health` sin cambios. Sin dependencias externas nuevas.
- **iHost**: Cero dependencias nuevas (Go stdlib + SQLite). Migración `0024` ligera. Build multi-arch (`linux/amd64`, `linux/arm/v7`, `linux/arm64`) sin cambios.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Sistema de facturación de servicios: `internal/services/billing_scheduler.go` (`generateBillForService`, líneas 144-193), `internal/services/service.go` (`generateCurrentBill` 206-232, `ReconcileBills` 135-147), deduplicación `FindByServicePeriod` + `UNIQUE(service_id, year, month)`.
- Modelo de factura: `internal/models/bill.go` con estados `pending`/`paid` (CHECK en `migrations/0005_create_bills.up.sql`), pago con `paid_at` + `payment_reference` (SPEC-043, `internal/storage/bill.go` `Pay` 121-132).
- Prerrequisitos: `ServiceFormPage.tsx` (112-134) muestra `DependencyWarning` y redirige a `/home/new` o `/institutions/new`.
- Patrón de tabs con URL: no existe componente de tabs; el patrón más cercano es `useSearchParams` en `RegistrosPage.tsx` (200-206) para sincronizar año/mes a la URL.
- Entidad "Acreedor": `internal/models/institution.go` con categorías seed incluyendo `loans`/`banking` (`migrations/0011_create_institution_categories.up.sql`).
- Moneda: tabla `currencies` (migración `0002`) con API `GET /api/currencies` ya disponible.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Acreedor = Institution (FK `institution_id`)** | Cero entidades nuevas, categorías existentes (loans), prerrequisito ya probado | Un acreedor personal no cabe como institución | ✅ Seleccionada |
| Acreedor = tabla nueva `creditors` | Semántica exacta | Duplica instituciones, más código y datos | ❌ Rechazada |
| Cuotas en tabla `bills` existente (con `debt_id` nullable) | Un solo listado de facturas | Rompe `UNIQUE(service_id, year, month)`, `service_id` NOT NULL, mezcla lógica | ❌ Rechazada |
| **Cuotas en tabla nueva `debt_bills`** | Estados idénticos a `bills`, `due_date` real, dedup por `(debt_id, installment_number)` | Tabla adicional (ligera) | ✅ Seleccionada |
| Generación mensual vía scheduler (como servicios) | Igual a servicios | Complejidad de ticker + hora configurable para un número acotado de cuotas | ❌ Rechazada |
| **Generación completa upfront + reconcile** | Simple, determinista, "de inicio a fin", sin scheduler nuevo | Si N grande, varias filas de una vez (acotado) | ✅ Seleccionada |
| Tabs con sub-rutas `/deudas/calendario` y `/deudas/deudas` | URLs limpias | Colisión conceptual con `/deudas/new`/`/deudas/edit/:id` | ❌ Rechazada |
| **Tabs con query param `?tab=`** | Consistente con `RegistrosPage`, sin colisiones de rutas | URL con query param | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Acreedor = Institution**
- **Contexto**: El campo "Acreedor" pide un dropdown de una lista existente. Las instituciones ya tienen categorías (incluida `loans`) y el patrón de prerrequisitos está resuelto.
- **Decisión**: `debt.institution_id` → FK a `institutions`. El dropdown lista instituciones con `Select` searchable. Si no hay instituciones, redirigir a `/institutions/new`.
- **Consecuencias**: Menos entidades y código. Si a futuro se necesitan acreedores personales, se agrega un flag o tabla dedicada sin tocar deudas.

**ADR-002: Tabla `debt_bills` separada de `bills`**
- **Contexto**: Las facturas de servicios usan `(service_id, year, month)` y `month=0` para anuales; las cuotas de deuda tienen un vencimiento concreto (`due_date`) y número de cuota.
- **Decisión**: Nueva tabla `debt_bills` con `UNIQUE(debt_id, installment_number)` y CHECK `status IN ('pending','paid')`, replicando la semántica de pago de `bills` (`paid_at`, `payment_reference`).
- **Consecuencias**: Coexisten dos tablas de facturación con la misma lógica de estados; el calendario puede unirlas por `due_date`/periodo sin acoplarlas.

**ADR-003: Generación completa de cuotas upfront + reconcile (sin scheduler)**
- **Contexto**: El número de cuotas es finito y conocido (`total_cuotas`). El scheduler de servicios existe porque los servicios son indefinidos.
- **Decisión**: Al crear/editar una deuda `activa`, `ensureDebtBills(debt)` genera todas las cuotas faltantes (1..N), con `due_date(k)` = día de pago del mes `(fecha_inicio + k meses)`, clampeado al último día del mes (reutilizar `daysInMonth`). Un `ReconcileDebtBills()` corre al listar deudas para rellenar huecos.
- **Consecuencias**: Sin procesos en background extra; el calendario siempre ve la serie completa.

**ADR-004: Tabs vía query param `?tab=`**
- **Contexto**: No existe componente de tabs y el requisito pide que la URL cambie al alternar.
- **Decisión**: `/deudas` con `?tab=calendario` (default) y `?tab=deudas`, sincronizado con `useSearchParams`. Las rutas CRUD (`/deudas/new`, `/deudas/edit/:id`) quedan libres.
- **Consecuencias**: Consistente con `RegistrosPage`; el header de `DashboardLayout` (`t('deudas.title')`) se mantiene estable.

**ADR-005: Email diario agrupado por día (un solo mail, no uno por cuota)**
- **Contexto**: El día en que vence una cuota se debe avisar, pero enviar un mail por cada cuota generaría spam. Ya existe infraestructura de email/alertas (`bill_summary_scheduler.go`, SPEC-031).
- **Decisión**: Nuevo `DebtDueScheduler` que corre una vez al día **a la hora de notificaciones del sistema** (`alert_check_hour` vía `GetAlertCheckHour`, la misma que usan `alert_scheduler.go` y `bill_summary_scheduler.go`; **no se agrega una hora nueva**). Ticker horario + clave de idempotencia `last_debt_due_check` en `system_settings` (patrón `bill_summary_scheduler`). Consulta `debt_bills` con `due_date = hoy` y `status = pending`, **agrupa por día** (hoy) y envía un único email con el desglose de cada cuota y el **total del día**. El envío se controla con un toggle de alerta nuevo (`debt_due`) en el catálogo de `alert_service.go` y depende de SMTP configurado + destinatarios, igual que `bill_created`/`bill_summary`.
- **Consecuencias**: Un solo scheduler más (patrón clonado, sin dependencias nuevas). Sin ajustes nuevos en Configuración para la hora: usa la hora de notificaciones ya existente. Voice Monkey queda como REQ-014 (P2).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[DeudasPage /deudas?tab=...]
   ├─ tab=calendario ──> [DebtCalendar: mes + detalle por día + Pagar]
   └─ tab=deudas ──────> [DebtList: cards + CardMenu + EmptyCard]
        └─ /deudas/new, /deudas/edit/:id ──> [DebtForm]
        └─ /deudas/:id ──> [DebtBillsPage: cuotas + Pagar]

[API REST]
   /api/debts (CRUD + reconcile)
   /api/debts/{id}/bills
   /api/debt-bills/{id}/pay
   /api/debt-bills?year&month  (calendario)

[Backend]
   internal/api/debt_handlers.go ──> internal/services/debt.go (lógica + ensureDebtBills)
        └─> internal/storage/debt.go + internal/storage/debt_bill.go
        └─> internal/models/debt.go + debt_bill.go
        └─> SQLite (migración 0024)
```

### 4.2 Componentes

#### 4.2.1 Backend: modelo `Debt`
- **Responsabilidad**: Entidad de dominio de una deuda.
- **Interfaz**: JSON idéntico a los campos del formulario.
- **Dependencias**: `Institution`, `Currency`.
- **Ubicación**: `internal/models/debt.go`.

#### 4.2.2 Backend: `debt_bills` (cuotas)
- **Responsabilidad**: Facturas/cuotas generadas por cada deuda, con estado `pending`/`paid`.
- **Interfaz**: Listar por deuda, listar por rango de fechas (calendario), pagar.
- **Dependencias**: `Debt`.
- **Ubicación**: `internal/models/debt_bill.go`.

#### 4.2.3 Backend: servicio `DebtService`
- **Responsabilidad**: CRUD + validaciones + `ensureDebtBills` + `ReconcileDebtBills` + `PayDebtBill`.
- **Ubicación**: `internal/services/debt.go`.

#### 4.2.4 Backend: scheduler `DebtDueScheduler`
- **Responsabilidad**: Una vez al día, **a la hora de notificaciones del sistema** (`alert_check_hour` vía `GetAlertCheckHour`, sin hora propia), consulta las cuotas de deuda con `due_date = hoy` y `status = pending`, las agrupa y envía **un único email** con el desglose y el total del día. Controlado por el toggle de alerta `debt_due` (patrón `bill_summary_scheduler.go`), con clave de idempotencia `last_debt_due_check`.
- **Interfaz**: `Start()`/`Stop()`; se registra en `cmd/server/main.go` junto a los demás schedulers.
- **Dependencias**: `DebtBillStorage`, `emailService`, `alertService`, `settingsService`.
- **Ubicación**: `internal/services/debt_due_scheduler.go`.

#### 4.2.5 Frontend: `DeudasPage` (tabs)
- **Responsabilidad**: Contenedor con tab bar (Calendario/Deudas), header con `CreateMenu` ("Nueva Deuda"), sincroniza `?tab=` con la URL.
- **Ubicación**: `frontend/src/pages/DeudasPage.tsx`.

#### 4.2.6 Frontend: `DebtCalendar`
- **Responsabilidad**: Grilla mensual (prev/next mes), marcadores por día, detalle de cuotas del día con acción Pagar.
- **Ubicación**: `frontend/src/components/DebtCalendar.tsx` (o dentro de `DeudasPage`).

#### 4.2.7 Frontend: `DebtFormPage` / `DebtBillsPage`
- **Responsabilidad**: Formulario crear/editar (modelo `ServiceFormPage`) y listado de cuotas de una deuda (modelo `BillsPage`).
- **Ubicación**: `frontend/src/pages/DebtFormPage.tsx`, `frontend/src/pages/DebtBillsPage.tsx`.

### 4.3 Modelo de datos

```
Entidad: Debt
- id: INTEGER PK
- institution_id: INTEGER NOT NULL FK -> institutions.id
- identifier: TEXT NOT NULL (ej: "1111 2222 3333 4444")
- description: TEXT NOT NULL (ej: "Tarjeta de crédito")
- total: REAL NOT NULL DEFAULT 0
- principal: REAL NOT NULL DEFAULT 0 (capital prestado)
- currency_id: INTEGER NOT NULL FK -> currencies.id
- installments_total: INTEGER NOT NULL CHECK (>= 1)
- installment_amount: REAL NOT NULL DEFAULT 0 (por cuota; 0 => total/installments_total)
- interest_rate: REAL NOT NULL DEFAULT 0 (porcentaje)
- payment_day: INTEGER NOT NULL CHECK (1..31)
- start_date: TEXT NOT NULL (YYYY-MM-DD)
- status: TEXT NOT NULL DEFAULT 'activa' CHECK IN ('activa','inactiva','finalizada')
- created_at / updated_at: TIMESTAMP
- Relaciones: Institution (N:1), Currency (N:1), DebtBill (1:N)

Entidad: DebtBill (cuota)
- id: INTEGER PK
- debt_id: INTEGER NOT NULL FK -> debts.id
- installment_number: INTEGER NOT NULL (1..N)
- due_date: TEXT NOT NULL (YYYY-MM-DD, día de pago clampeado)
- amount: REAL NOT NULL
- status: TEXT NOT NULL DEFAULT 'pending' CHECK IN ('pending','paid')
- paid_at: TEXT NULL
- payment_reference: TEXT NULL
- created_at / updated_at: TIMESTAMP
- UNIQUE(debt_id, installment_number)
- INDEX(due_date)
- Relaciones: Debt (N:1)
```

**Cálculo de vencimiento**: `due_date(k) = min(payment_day, días del mes)` del mes `(start_date + k meses)`, para `k = 1..N`. Reutilizar helper `daysInMonth` de `billing_scheduler.go`.

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/debts`
Ejecuta `ReconcileDebtBills` antes de listar (patrón `GET /api/services`).

**Response 200**:
```json
{
  "debts": [
    {
      "id": 1,
      "institution_id": 3,
      "institution_name": "BAC",
      "identifier": "1111 2222 3333 4444",
      "description": "Tarjeta de crédito",
      "total": 12000,
      "principal": 10000,
      "currency_id": 1,
      "currency_code": "NIO",
      "installments_total": 12,
      "installment_amount": 1000,
      "interest_rate": 0,
      "payment_day": 15,
      "start_date": "2026-09-04",
      "status": "activa",
      "created_at": "2026-09-04T10:00:00Z",
      "updated_at": "2026-09-04T10:00:00Z"
    }
  ]
}
```

#### Endpoint: `POST /api/debts` / `PUT /api/debts/{id}`
**Request**:
```json
{
  "institution_id": 3,
  "identifier": "1111 2222 3333 4444",
  "description": "Tarjeta de crédito",
  "total": 12000,
  "principal": 10000,
  "currency_id": 1,
  "installments_total": 12,
  "installment_amount": 1000,
  "interest_rate": 0,
  "payment_day": 15,
  "start_date": "2026-09-04",
  "status": "activa"
}
```
**Response 201/200**: deuda creada/actualizada + cuotas generadas (`ensureDebtBills`).
**Response Error**:
```json
{ "error": "validation", "message": "El día de pago debe estar entre 1 y 31" }
```

#### Endpoint: `GET /api/debts/{id}/bills`
**Response 200**:
```json
{
  "bills": [
    {
      "id": 1,
      "debt_id": 1,
      "installment_number": 1,
      "due_date": "2026-10-15",
      "amount": 1000,
      "status": "pending",
      "paid_at": null,
      "payment_reference": null
    }
  ]
}
```

#### Endpoint: `GET /api/debt-bills?year=2026&month=10` (calendario)
**Response 200**:
```json
{
  "bills": [
    {
      "id": 1,
      "debt_id": 1,
      "debt_description": "Tarjeta de crédito",
      "installment_number": 1,
      "due_date": "2026-10-15",
      "amount": 1000,
      "status": "pending"
    }
  ]
}
```

#### Endpoint: `PUT /api/debt-bills/{id}/pay`
**Request**:
```json
{
  "paid_at": "2026-10-15T09:00:00Z",
  "payment_reference": "REF-1234"
}
```
**Response 200**: `{ "bill": { "...": "status: paid" } }`

### 4.5 Dependencias

- **Internas**:
  - `internal/storage/institution.go` (listar acreedores)
  - `internal/storage/currency.go` (listar monedas)
  - Helper `daysInMonth` de `internal/services/billing_scheduler.go`
  - Patrón de autenticación `authMiddleware` (`internal/api/routes.go`)
  - Infraestructura de emails/alerts: `internal/services/email_service.go`, `alert_service.go` (catálogo + toggles), `bill_summary_scheduler.go` (patrón del scheduler diario), `system_settings.go` (clave de hora + idempotencia)
  - Componentes frontend: `Select`, `CreateMenu`, `CardMenu`, `EmptyCard`, `DeleteModal`, `PayBillModal`, `Toggle`/inputs
- **Externas**: Ninguna. Go stdlib + SQLite (database/sql).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: El Sidebar muestra el menú **Deudas** y navega a `/deudas`.
- [ ] CA-002: El header de `/deudas` muestra `CreateMenu` con la opción **"Nueva Deuda"** que abre `/deudas/new`.
- [ ] CA-003: El formulario permite crear una deuda con los 12 campos (REQ-004) y guardarla; al guardar, se generan `installments_total` cuotas `pending`.
- [ ] CA-004: Cada cuota tiene `due_date` = día de pago del mes correspondiente (a partir del mes siguiente de `start_date`), clampeado al último día del mes.
- [ ] CA-005: Editar una deuda (cambiar N, monto, día, estado) regenera/rellena cuotas sin duplicar (dedup por `(debt_id, installment_number)`).
- [ ] CA-006: Si no hay instituciones, `/deudas/new` redirige a `/institutions/new` con aviso; si no hay monedas, redirige a `/settings/currency`.
- [ ] CA-007: La página `/deudas` muestra dos pestañas; al cambiar de pestaña cambia `?tab=` en la URL y la vista (recargar mantiene la pestaña).
- [ ] CA-008: La pestaña Calendario muestra un mes navegable con marcadores en los días con cuotas; al hacer clic en un día se ven las cuotas (descripción, monto, estado).
- [ ] CA-009: La pestaña Deudas lista deudas en cards (patrón existente) con menú editar/eliminar (modal de confirmación).
- [ ] CA-010: Una cuota puede marcarse como **pagada** (fecha de pago + referencia) y pasa a estado `paid`; el calendario refleja el cambio.
- [ ] CA-011: Eliminar una deuda elimina (soft delete) la deuda y sus cuotas; no aparecen en listados ni calendario.
- [ ] CA-012: El día en que vence al menos una cuota `pending`, se envía **un único email** que agrupa todas las cuotas que vencen ese día (descripción de deuda, número de cuota, monto) e incluye el **total del día**. No se envía un email por cada cuota. El envío ocurre **a la hora de notificaciones del sistema (`alert_check_hour`)** y se controla con el toggle de alerta `debt_due` en Configuración (SMTP configurado + destinatarios requeridos).

### 5.2 No funcionales

- [ ] CA-NF-001: Sin dependencias nuevas; build Go pasa `go build ./...` y `gofmt`.
- [ ] CA-NF-002: Migración `0024` idempotente (up/down).
- [ ] CA-NF-003: `npm run build` en `frontend/` genera el bundle sin errores y las claves i18n nuevas están servidas en `/i18n/es.json`.
- [ ] CA-NF-004: Endpoints protegidos con `authMiddleware`; sin endpoints públicos nuevos.

### 5.3 Testing

- **Unit tests**: cálculo de `due_date` (clampeo 28/29/30/31, meses y año bisiesto), `ensureDebtBills` (dedup, N, estado inactiva), `installment_amount` = 0 → `total/N`.
- **Integration tests**: crear deuda → N cuotas; pagar cuota → `paid`; GET calendario por rango; validación de prerrequisitos.
- **E2E tests**: flujo completo en local (crear deuda con 12 cuotas, ver calendario, pagar cuota).
- **Carga/Performance**: deuda con 60 cuotas genera sin latencia perceptible; consulta de calendario indexada por `due_date`.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0024` (tablas `debts`, `debt_bills`) + modelos + storage | 1 día | Ninguna |
| 2 | `DebtService` (CRUD + validaciones + `ensureDebtBills` + `ReconcileDebtBills` + `PayDebtBill`) | 1 día | Fase 1 |
| 3 | Handlers + rutas (`/api/debts*`, `/api/debt-bills*`) | 1 día | Fase 2 |
| 4 | i18n (`frontend/public/i18n/{es,en}.json` + `frontend/src/i18n/`) + Sidebar + API client `api/debt.ts` | 1 día | Ninguna |
| 5 | `DeudasPage` (tabs + `?tab=`) + `DebtList` (cards) + `DebtCalendar` | 2 días | Fase 4 |
| 6 | `DebtFormPage` (crear/editar + DependencyWarning) + `DebtBillsPage` (cuotas + Pagar) | 2 días | Fase 5 |
| 7 | **`DebtDueScheduler`**: alerta `debt_due` en catálogo + email diario agrupado de cuotas vencidas (desglose + total) | 1 día | Fase 2 |
| 8 | Build frontend, tests, validación local en iHost + evaluación manual | 1 día | Fases 1-7 |

### 6.2 Milestones

1. **MVP**: Fases 1-3 (backend completo) + fase 6 (form + cuotas) sin calendario ni email.
2. **V1.0**: Todo, incluyendo pestañas Calendario/Deudas, pago de cuotas y email diario agrupado de cuotas vencidas (REQ-012).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Fechas de vencimiento incorrectas (clampeo de día 31 en meses cortos) | Media | Alto | Unit tests exhaustivos del cálculo de `due_date`; reutilizar `daysInMonth` probado |
| Duplicación de cuotas al editar | Media | Medio | `UNIQUE(debt_id, installment_number)` + `ensureDebtBills` idempotente |
| Aumento de complejidad de la página de tabs | Media | Medio | Reutilizar patrón `useSearchParams` de `RegistrosPage`; sin librería de calendario externa |
| Confusión Acreedor vs Institución | Baja | Bajo | ADR-001 documentado; label "Acreedor" claro en UI |
| Crecimiento de `debt_bills` con deudas históricas | Baja | Bajo | Soft delete + índice `due_date`; volumen acotado (N cuotas por deuda) |

## 8. Notas y Referencias

- Patrón de generación de facturas: SPEC-008, `internal/services/billing_scheduler.go`
- Estado de pago dinámico y pago de facturas: SPEC-018, SPEC-043 (`Pay` en `internal/storage/bill.go`)
- Email diario agrupado de facturas pendientes: SPEC-031, `internal/services/bill_summary_scheduler.go`
- Sistema de alertas multicanal con toggles: SPEC-032/033, `internal/services/alert_service.go`
- Página de facturas por servicio: `frontend/src/pages/BillsPage.tsx`
- Formulario con prerrequisitos: `frontend/src/pages/ServiceFormPage.tsx`
- Sincronización de vista con URL: `frontend/src/pages/RegistrosPage.tsx` (200-206)
- Categorías de instituciones (loans/banking): `migrations/0011_create_institution_categories.up.sql`
- Migraciones existentes hasta `0023`; esta spec agrega `0024`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-04 | p40la-ihost-team | Agregado REQ-012/ADR-005/CA-012: email diario agrupado de cuotas vencidas (un solo mail por día con desglose y total, toggle de alerta `debt_due`); nuevo `DebtDueScheduler` y fase 7 en el plan |
| 2026-09-04 | p40la-ihost-team | La hora de envío del email de cuotas usa la **hora de notificaciones del sistema** (`alert_check_hour` vía `GetAlertCheckHour`, como `alert_scheduler`/`bill_summary_scheduler`); sin hora nueva en Configuración |
| 2026-09-04 | p40la-ihost-team | Fix dark mode: los inputs del formulario/cuotas no mostraban el texto escrito (preflight de Tailwind fuerza `color: inherit` y con dark mode activo el texto claro quedaba invisible sobre el fondo blanco del input). Se aplica `bg-card` a los inputs nuevos y al input de búsqueda de `Select` (convención ya usada en `PayBillModal`). |
| 2026-09-04 | p40la-ihost-team | Calendario: al cargar preselecciona el **día de hoy** y muestra su detalle de cuotas debajo sin necesidad de hacer clic (antes cargaba sin selección). |
| 2026-09-04 | p40la-ihost-team | Fix calendario: el día quedaba resaltado pero el detalle no se mostraba hasta hacer clic — el `useEffect` de limpieza corría también al montar y borraba la selección de hoy. Ahora solo se limpia la selección si queda fuera del mes visible. |
| 2026-09-04 | p40la-ihost-team | **Release**: implementación en commit `9acef72` (rama `feature/SPEC-054`), merge a `main`. Validado por el usuario con datos reales importados de P4OLA en DB temporal. |