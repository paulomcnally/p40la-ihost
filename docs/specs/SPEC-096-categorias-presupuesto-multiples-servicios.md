---
title: "Categorías de presupuesto vinculadas a múltiples servicios + fix dropdown detrás del modal"
id: "SPEC-096"
status: "draft"
author: "opencode"
created: "2026-09-28"
updated: "2026-09-28"
github_issue: 99
---

# Categorías de presupuesto vinculadas a múltiples servicios + fix dropdown detrás del modal

**ID**: SPEC-096  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-09-28  
**Actualizado**: 2026-09-28

---

## 1. Resumen Ejecutivo

El usuario reporta dos problemas en el modal de crear/editar categoría del módulo de Presupuesto (SPEC-093/094). Primero, **un bug de UI**: al hacer click en el dropdown de "Servicio" dentro del modal, la lista de opciones aparece **detrás del modal** y es imposible seleccionar. La causa raíz es que el componente custom `Select` (`frontend/src/components/Select.tsx`) renderiza su menú con `z-50` hardcodeado y lo porta a `document.body`, mientras que el modal `CategoryFormModal` usa `z-[60]` y `BudgetConfigModal` `z-50`: el overlay del modal (que llega después en el DOM y con igual/mayor z-index) tapa el menú. El fix es hacer configurable el z-index del menú del `Select` y elevar el que se usa dentro de modales.

Segundo, **un requerimiento funcional**: una categoría de presupuesto debe poder vincularse a **uno o más servicios** del sistema. Hoy el modelo fuerza 1:1: `categories.service_id` con un índice único parcial (`0036_add_category_service_link`) garantiza que un servicio solo se vincula a una categoría. El usuario quiere, por ejemplo, la categoría "Internet Móvil" vinculada a los dos servicios de internet móvil que tiene (uno de Claro y otro de Tigo), de modo que al pagar una factura de cualquiera de esos servicios se registre automáticamente la transacción en esa categoría. Se necesita pasar de la columna `service_id` a una tabla de enlace `category_service_links(category_id, service_id)`.

**Consideraciones iHost**: la tabla de enlace es liviana (dos FKs + PK compuesta); se agrega una migración `0037`. El hook de pago (`BudgetBillLinkService.OnBillPaid`, SPEC-094) se adapta para resolver la categoría desde la tabla de enlace. La relación **servicio → categoría sigue siendo a lo sumo 1** (una factura de un servicio genera transacción en una única categoría, sin ambigüedad de reparto); solo cambia que una **categoría → N servicios**. Memoria/CPU despreciables (una query más por pago). Sin dependencias nuevas.

**Consideraciones de UI obligatorias**: el dropdown de servicios pasa a ser una **selección múltiple** (multi-select) manteniendo el patrón de componentes existente; el menú del `Select` DEBE quedar por encima de cualquier modal (fix de z-index). Todos los inputs/selects usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verifica darkmode (SPEC-060). No se crean páginas de detalle nuevas: la vinculación vive en los modales existentes de `BudgetPage.tsx`, por lo que no aplica `BACK_ROUTES` (CA-BACK).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Fix z-index del dropdown del componente `Select` — agregar una prop opcional `zIndex` (default `z-50`, compatible con uso actual) y aplicarla al menú portado a `document.body`. En `CategoryFormModal` y `BudgetConfigModal` (y cualquier modal que use `Select`) pasar un valor mayor al del modal (ej. `z-[80]`) para que el menú se abra por encima del overlay. Verificar que también se aplica a `TransactionFormModal`/`AccountFormModal` de `BudgetTransactionsPage.tsx` si usan `Select` (mismo patrón de bug).
2. **REQ-002**: Migración `0037` — crear tabla `category_service_links (category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE, service_id INTEGER NOT NULL REFERENCES services(id), PRIMARY KEY (category_id, service_id))` + índice por `service_id` (búsqueda servicio→categoría). Migrar los datos existentes de `categories.service_id` a la tabla de enlace. Luego `DROP INDEX idx_categories_service_link` y `DROP COLUMN categories.service_id`. Se conserva `categories.account_id` (la cuenta destino sigue siendo una sola por categoría). El `.down.sql` revierte: recrea la columna `service_id`, repuebla desde la tabla de enlace (tomando el primer servicio de cada categoría), restaura el índice único parcial y dropea la tabla de enlace.
3. **REQ-003**: Modelo y storage — `Category` cambia `ServiceID *int64`/`ServiceName string` por `ServiceIDs []int64` y `ServiceNames []string` (o una lista de enlaces con nombre para el badge). `CategoryStorage`: `ListByGroup`/`ListAll`/`GetByID` obtienen los servicios vinculados con un join a `category_service_links` + `services`; `GetByServiceID(serviceID)` pasa a consultar la tabla de enlace y devuelve la categoría activa (a lo sumo una); `ServiceLinkInUse` consulta la tabla de enlace; `Create`/`Update` reciben `service_ids []int64` y persisten las filas de enlace en una transacción (reemplazo completo del set).
4. **REQ-004**: Backend CRUD — `categoryRequest` acepta `service_ids []int64` (además de `account_id`). Validación: si viene `service_ids`, `account_id` obligatorio; cada servicio debe existir y no estar vinculado a otra categoría activa distinta; `service_ids` vacío limpia el vínculo. Respuestas incluyen `service_ids` y `service_names`.
5. **REQ-005**: Hook de pago `BudgetBillLinkService.OnBillPaid` (SPEC-094) — resolver la categoría vía `GetByServiceID` (que ahora consulta la tabla de enlace). Sin cambios de contrato: la factura de cualquiera de los N servicios vinculados a la categoría genera la transacción en esa categoría, idempotente por `source_bill_id`.
6. **REQ-006**: UI crear/editar categoría (`CategoryFormModal` de `BudgetPage.tsx` y `BudgetConfigModal`) — el campo "Servicio" pasa a ser una **selección múltiple**: lista de servicios con checkboxes (modo `multiple` en el `Select`, o un componente `MultiSelect` ligero siguiendo el patrón), con búsqueda si hay muchos servicios. Se muestra la cuenta seleccionada como hoy. El badge de la vista mensual muestra el/los nombre(s) de servicio vinculados (si son varios, mostrar el primero + "+N" o lista truncada).
7. **REQ-007**: i18n — actualizar/agregar claves `budget` en `frontend/public/i18n/{es,en}.json` para la selección múltiple (ej. `link_services`, `linked_services`, `N_services`) y correr `npm run build`. Fuente de verdad: `frontend/public/i18n/`, nunca `public/i18n/`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: Verificar los demás lugares que consumen `Category.ServiceID`/`ServiceName` y migrarlos (badge en `BudgetPage.tsx`, edición de categoría, tipos frontend `BudgetCategory`/`BudgetCategoryRow`, `api/index.ts`) para que no queden referencias a la columna eliminada.
2. **REQ-009**: Mantener la validación de que un servicio no puede quedar vinculado a dos categorías activas (la tabla de enlace no tiene índice único sobre `service_id`, por lo que la validación se hace en `CategoryService.validate` + si se desea, un índice único parcial en `category_service_links(service_id)` — evaluar en desarrollo).
3. **REQ-010**: Al desvincular o archivar la categoría, las transacciones ya creadas se conservan; el hook deja de generar nuevas para esos servicios (comportamiento actual, confirmar tras la migración).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-011**: Mostrar en el badge de la vista mensual un tooltip/expand con los nombres de todos los servicios vinculados si son varios.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el hook agrega a lo sumo 3-4 queries por pago (buscar categoría por servicio en link table + crear transacción). La vista mensual agrega un JOIN a la tabla de enlace por listado de categorías. Impacto despreciable en iHost.
- **Seguridad**: endpoints bajo `authMiddleware` (sesión). Validación de IDs existente se reutiliza.
- **Almacenamiento**: tabla de enlace pequeña (2 enteros por fila); se elimina `categories.service_id`. Migración `0037` aplica/revierte limpiamente.
- **Disponibilidad**: el hook es síncrono dentro del request de pago; si falla la creación de la transacción, el pago ya está commiteado y el error se loguea (no bloquea el pago) — comportamiento heredado de SPEC-094.
- **iHost**: sin dependencias nuevas (Go stdlib + SQLite). Build multi-arch intacto. Migración en el mecanismo existente.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Requerimiento fuente**: solicitud del usuario (sesión 2026-09-28): (1) el dropdown de servicio en el modal de crear/editar categoría de presupuesto aparece detrás del modal; (2) una categoría debe poder vincularse a uno o más servicios (ej. categoría "Internet Móvil" con 2 servicios: Claro y Tigo).
- **Causa del bug de z-index**: `frontend/src/components/Select.tsx` renderiza el menú con `createPortal(document.body)` y clase `fixed ... z-50` (línea 89). `BudgetPage.tsx`: `CategoryFormModal` es `fixed inset-0 z-[60]` (línea 943), `BudgetConfigModal` `z-50` (línea 750), `GroupFormModal` `z-[60]`, e `IconPickerModal` recibe `zIndex="z-[70]"` (líneas 288/859). El overlay del modal (mayor z) queda por encima del menú portado con `z-50` → el menú se ve "detrás". Referencia del patrón correcto: `IconPickerModal` ya acepta prop `zIndex` (default `z-50`) que se eleva a `z-[70]` desde el caller.
- **Modelo actual (SPEC-094)**: `categories.service_id INTEGER REFERENCES services(id)` + `idx_categories_service_link` único parcial (`service_id IS NOT NULL AND deleted_at IS NULL`) → garantiza 1 servicio → 1 categoría activa. `categories.account_id` (cuenta destino, única por categoría). `transactions.source_bill_id` para idempotencia.
- **Flujo de pago**: `POST /api/bills/{id}/pay` → `BillService.PayBill` → `SetBillBudgetLinker.OnBillPaid(bill)` → `BudgetBillLinkService.OnBillPaid` → `CategoryStorage.GetByServiceID(serviceID)` → crea transacción con `account_id`, `category_id`, `currency_id` del servicio, `payee` nombre del servicio, `memo` referencia de factura, `source_bill_id`.
- **Patrón de UI**: `Select` custom obligatorio para dropdowns (SPEC-004 REQ-024). Se puede extender con modo `multiple` o crear un `MultiSelect` ligero sin dependencias (iHost: minimizar bundle).
- **Migraciones**: última es `0036` (SPEC-094). La nueva es `0037` (`NNNN_descripcion.{up,down}.sql`).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Tabla de enlace `category_service_links` (N:N) | Soporta categoría→N servicios; modelo limpio; extensible | Una tabla nueva; migración de datos | ✅ Seleccionada |
| Comma-separated `service_ids` en `categories` | Sin tabla nueva | Sin FKs ni índices; queries complejas; contra reglas | ❌ Rechazada |
| JSON array en `categories.service_ids` | Sin tabla nueva | SQLite sin tipo array; joins imposibles; frágil | ❌ Rechazada |
| Mantener `service_id` y agregar tabla de enlace en paralelo | Compatibilidad | Dos fuentes de verdad; confusión en hooks | ❌ Rechazada (se elimina `service_id`) |
| `Select` con prop `zIndex` (fix puntual) | Mínimo cambio, consistente con `IconPickerModal` | Hay que pasar la prop en cada modal | ✅ Seleccionada |
| Subir el z-index del menú de `Select` globalmente a `z-[80]` | Un solo cambio | Podría pisar otros overlays no modales (bajo riesgo) | ⚠️ Evaluada; se prefiere prop explícita para control |
| Multi-select como checklist inline en el modal | Sin componente nuevo complejo | Ocupa espacio; sin búsqueda | ⚠️ Evaluada; se prefiere `Select` con modo `multiple` si es simple, si no checklist con búsqueda |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Tabla de enlace `category_service_links` en vez de columna `service_id`.
- **Contexto**: una categoría debe poder vincularse a N servicios; un servicio sigue mapeando a a lo sumo 1 categoría (sin reparto de montos, SPEC-094 REQ-010 diferido).
- **Decisión**: `CREATE TABLE category_service_links (category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE, service_id INTEGER NOT NULL REFERENCES services(id), PRIMARY KEY (category_id, service_id))` con índice por `service_id`. Se migran los datos existentes y se elimina `categories.service_id` (+ su índice único parcial). `categories.account_id` se conserva.
- **Consecuencias**: el hook de pago resuelve la categoría desde la tabla de enlace; no hay ambigüedad servicio→categoría (se valida en `CategoryService.validate` que un servicio no pertenezca a dos categorías activas). La vista mensual hace un JOIN para los nombres.

**ADR-002**: La validación servicio→1 categoría se mantiene a nivel de servicio (no de tabla).
- **Contexto**: la tabla de enlace permite N:N técnicamente; la regla de negocio es que un servicio pertenezca a una sola categoría activa (para que el hook no tenga que repartir).
- **Decisión**: `CategoryService.validate` rechaza si `service_ids` incluye un servicio ya vinculado a otra categoría activa distinta. Opcional: índice único parcial `CREATE UNIQUE INDEX ... ON category_service_links(service_id) WHERE service_id IS NOT NULL` en la migración (más seguro); decidir en desarrollo según costo de migración.
- **Consecuencias**: el hook sigue siendo no-ambigüo; se evitan transacciones duplicadas/espejo entre categorías.

**ADR-003**: Fix de z-index del `Select` con prop `zIndex` explícita (patrón `IconPickerModal`).
- **Contexto**: el menú del `Select` es `z-50` portado a `body`; los modales usan `z-50`/`z-[60]`/`z-[70]`; el menú queda detrás.
- **Decisión**: agregar prop `zIndex?: string` a `Select` (default `z-50`, no rompe usos actuales) y pasarla en los modales que contienen selects (ej. `z-[80]`). El menú renderizado conserva la clase indicada.
- **Consecuencias**: control explícito por caller; sin efectos colaterales en selects fuera de modales.

**ADR-004**: El badge de la vista mensual muestra el/los servicios vinculados.
- **Contexto**: antes se mostraba `service_name` único; ahora puede haber N.
- **Decisión**: si hay 1 servicio se muestra su nombre; si hay N se muestra el primero + "+N" (con tooltip/listado en P2 REQ-011).
- **Consecuencias**: UI compacta en móvil; sin sobrecarga visual.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Frontend BudgetPage / BudgetConfigModal]
        │  create/update category { service_ids: [...], account_id }
        ▼
[api.BudgetHandlers.Create/UpdateCategory] --[validar]--> [services.CategoryService]
        │
        ▼
[SQLite: categories (+account_id) ← category_service_links → services]

[Frontend BillsPage -> PayBillModal]
        │  POST /api/bills/{id}/pay { paid_at }
        ▼
[api.BillHandlers.PayBill] --> [services.BillService.PayBill]
        │                      └─ SetBillBudgetLinker.OnBillPaid(bill)
        ▼
[services.BudgetBillLinkService]
        ├─ CategoryStorage.GetByServiceID(service_id)  (via category_service_links)
        ├─ verificar idempotencia por source_bill_id
        └─ TransactionStorage.Create({source_bill_id, ...})
```

### 4.2 Componentes

#### 4.2.1 Backend

- **`internal/models/category.go`**: reemplazar `ServiceID *int64` y `ServiceName string` por `ServiceIDs []int64` y `ServiceNames []string` (json `service_ids,omitempty`, `service_names,omitempty`). Mantener `AccountID *int64`.
- **`internal/storage/category.go`**: `categoryColumns`/`categoryFromJoins` dejan de leer `c.service_id`/`services s`; se agrega una query auxiliar para obtener servicios vinculados por categoría (o un `LEFT JOIN` agregado). Nuevos métodos: `ListServicesByCategory(ctx, ids)` o resolver en el scan; `GetByServiceID(serviceID)` consulta `category_service_links` + `categories` (activa, `deleted_at IS NULL`); `ServiceLinkInUse(ctx, serviceID, excludeCategoryID)` consulta la tabla de enlace; `Create`/`Update` persisten `service_ids` en una transacción (delete + insert de enlaces).
- **`internal/services/category.go`**: `validate` recibe `service_ids []int64`; si hay servicios → `account_id` obligatorio; cada servicio existe (`ServiceStorage`) y no está en uso por otra categoría activa. Ajustar `Create`/`Update` para pasar `service_ids`.
- **`internal/services/budget_bill_link.go`**: `OnBillPaid` sigue igual; `GetByServiceID` ya resuelve desde la tabla de enlace. Sin cambios de contrato.
- **`internal/api/budget_handlers.go`**: `categoryRequest.ServiceID *int64` → `ServiceIDs []int64`; mapear al modelo.
- **`cmd/server/main.go`**: sin cambios (wiring ya existe).
- **`migrations/0037_add_category_service_links.{up,down}.sql`** (nuevo).

#### 4.2.2 Frontend

- **`frontend/src/components/Select.tsx`**: agregar prop `zIndex?: string` (default `z-50`) aplicada al menú portado. Opcional: soporte `multiple` (si se decide el multi-select con `Select`).
- **`frontend/src/components/MultiSelect.tsx`** (nuevo, si no se extiende `Select`): lista con checkboxes + búsqueda, siguiendo el patrón de `Select` (menú portado, tokens del tema, `zIndex` prop).
- **`frontend/src/pages/BudgetPage.tsx`**: `CategoryFormModal` y `BudgetConfigModal` usan multi-select de servicios; pasar `zIndex` alto a los `Select`/`MultiSelect` dentro de modales. Badge de vista mensual con nombre(s) de servicio (1 → nombre; N → primero + "+N"). Estado del form con `serviceIds?: number[]`.
- **`frontend/src/types/index.ts`**: `BudgetCategory`/`BudgetCategoryRow` cambian `service_id?: number|null`/`service_name?: string` por `service_ids?: number[]` y `service_names?: string[]`.
- **`frontend/src/api/index.ts`**: `budget.categories.create/update` aceptan `service_ids: number[]`.
- **`frontend/public/i18n/{es,en}.json`**: claves `budget.*` para multi-selección (`link_services`, `linked_services`, `N_services`, etc.). Luego `npm run build`.

### 4.3 Modelo de datos

Migración `0037_add_category_service_links.{up,down}.sql`:

```sql
-- 0037_add_category_service_links.up.sql
CREATE TABLE IF NOT EXISTS category_service_links (
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    service_id  INTEGER NOT NULL REFERENCES services(id),
    PRIMARY KEY (category_id, service_id)
);
CREATE INDEX IF NOT EXISTS idx_category_service_links_service
    ON category_service_links(service_id);

INSERT INTO category_service_links (category_id, service_id)
    SELECT id, service_id FROM categories
    WHERE service_id IS NOT NULL;

DROP INDEX IF EXISTS idx_categories_service_link;
ALTER TABLE categories DROP COLUMN service_id;
```

```sql
-- 0037_add_category_service_links.down.sql
ALTER TABLE categories ADD COLUMN service_id INTEGER REFERENCES services(id);
UPDATE categories SET service_id = (
    SELECT service_id FROM category_service_links
    WHERE category_id = categories.id
    ORDER BY service_id LIMIT 1
) WHERE id IN (SELECT category_id FROM category_service_links);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_service_link
    ON categories(service_id) WHERE service_id IS NOT NULL AND deleted_at IS NULL;
DROP TABLE IF EXISTS category_service_links;
```

**Notas**:
- `ALTER TABLE ... DROP COLUMN` requiere SQLite ≥ 3.35; ya usado en `0036.down.sql` (verificado en SPEC-094).
- La relación servicio→categoría se valida en servicio (`CategoryService.validate`); opcional índice único parcial en `category_service_links(service_id)` (evaluar en desarrollo).
- `AccountID` permanece en `categories`: una categoría tiene una única cuenta destino para las transacciones auto-generadas, sin importar cuántos servicios tenga.

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/budget/categories`

**Request** (cambio de `service_id` → `service_ids`):
```json
{
  "category_group_id": 1,
  "name": "Internet Móvil",
  "icon": "wifi",
  "target_amount": null,
  "service_ids": [7, 12],
  "account_id": 3
}
```

**Response 200**:
```json
{
  "id": 12,
  "category_group_id": 1,
  "name": "Internet Móvil",
  "icon": "wifi",
  "sort_order": 0,
  "target_amount": null,
  "service_ids": [7, 12],
  "service_names": ["Claro Internet Móvil", "Tigo Internet Móvil"],
  "account_id": 3
}
```

**Response Error**:
```json
{ "error": "invalid_request", "message": "Si vincula servicios debe seleccionar una cuenta" }
{ "error": "invalid_request", "message": "El servicio 7 ya está vinculado a otra categoría" }
```

#### Endpoint: `PUT /api/budget/categories/{id}`

Igual que POST; `service_ids: []` limpia el vínculo.

#### Endpoint: `POST /api/bills/{id}/pay` (sin cambios de contrato)

El hook se ejecuta de forma transparente; una factura de cualquier servicio vinculado a la categoría genera la transacción (idempotente por `source_bill_id`).

### 4.5 Dependencias

- **Internas**: `migrations/0037_*`, `internal/models/category.go`, `internal/storage/category.go`, `internal/services/category.go`, `internal/services/budget_bill_link.go`, `internal/api/budget_handlers.go`, `frontend/src/components/Select.tsx` (+ posible `MultiSelect.tsx`), `frontend/src/pages/BudgetPage.tsx`, `frontend/src/types/index.ts`, `frontend/src/api/index.ts`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Al hacer click en el dropdown de "Servicio" dentro del modal de crear/editar categoría, la lista se abre **por encima** del modal y es seleccionable (no queda detrás).
- [ ] CA-002: Se puede vincular una categoría a 2 o más servicios (ej. Claro y Tigo para "Internet Móvil"); la respuesta incluye `service_ids` y `service_names`.
- [ ] CA-003: Al pagar una factura de cualquiera de los servicios vinculados se crea la transacción en la categoría (outflow, moneda del servicio, payee, `source_bill_id`), y aparece en la grilla del mes y en activity/available.
- [ ] CA-004: Pagar la misma factura dos veces no crea transacciones duplicadas (idempotencia por `source_bill_id`).
- [ ] CA-005: Al editar se puede cambiar el set de servicios (agregar/quitar/vaciar) y la cuenta; las transacciones previas se conservan.
- [ ] CA-006: No se puede vincular el mismo servicio a dos categorías activas (backend rechaza con error claro).
- [ ] CA-007: Si se envían `service_ids` sin `account_id`, el backend rechaza con error claro.
- [ ] CA-008: Un servicio sin categoría vinculada sigue pagándose normalmente (hook no-op).
- [ ] CA-009: El badge de la vista mensual muestra el/los servicio(s) vinculado(s) (1 → nombre; N → primero + "+N").
- [ ] CA-010: Migración `0037` aplica y revierte limpiamente; los vínculos existentes (de `categories.service_id`) se migran a la tabla de enlace sin pérdida.
- [ ] CA-DARK: Los selects/inputs nuevos del modal usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (SPEC-060).
- [ ] CA-SELECT: Los dropdowns siguen usando componentes custom (`Select`/`MultiSelect`), nunca `<select>` nativo (SPEC-004 REQ-024).
- [ ] CA-BACK: No aplica (la vinculación vive en modales existentes; no hay páginas de detalle/formularios nuevos con lista padre). Verificar que no se introducen links "← Título".

### 5.2 No funcionales

- [ ] CA-NF-001: Migración `0037` aplica/revierte limpiamente (up y down).
- [ ] CA-NF-002: Sin dependencias externas nuevas; build multi-arch (`linux/amd64,linux/arm/v7,linux/arm64`) intacto.
- [ ] CA-NF-003: i18n actualizado en `frontend/public/i18n/{es,en}.json` y servido tras `npm run build` (verificar con `curl`).

### 5.3 Testing

- **Unit tests**: `CategoryStorage` (crear/editar con N servicios, `GetByServiceID` desde link table, `ServiceLinkInUse`); `CategoryService.validate` (service_ids+account, unicidad por servicio, vaciar vínculo); `BudgetBillLinkService.OnBillPaid` (factura de servicio 1 y de servicio 2 → misma categoría, idempotencia, sin categoría vinculada, cuenta faltante); migración `0037` up/down.
- **Integration tests**: flujo `POST /api/bills/{id}/pay` para servicios de una misma categoría → transacción creada con los valores correctos; re-pay idempotente; editar categoría (quitar un servicio) conserva transacciones.
- **E2E tests**: en local: crear categoría "Internet Móvil" vinculada a 2 servicios (Claro y Tigo) con su cuenta → pagar factura de Claro → ver transacción; pagar factura de Tigo → ver transacción; verificar badge "+2" en vista mensual; verificar que el dropdown se abre encima del modal.
- **Carga/Performance**: el hook agrega ≤ 4 queries por pago; medir en local con DB de prueba (despreciable).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Fix z-index `Select` (prop `zIndex`) + aplicar en modales de categoría y transacciones | 0.5 día | Ninguna |
| 2 | Migración `0037` up/down + modelo `Category` (`ServiceIDs`/`ServiceNames`) | 0.5 día | Ninguna |
| 3 | Storage: link table en `category.go` (query servicios por categoría, `GetByServiceID`, `ServiceLinkInUse`, `Create`/`Update` transaccional) | 1 día | Fase 2 |
| 4 | Services/API: `CategoryService.validate` con `service_ids`, `categoryRequest`, handlers | 0.5 día | Fase 3 |
| 5 | Frontend: multi-select (Select multiple o MultiSelect), `CategoryFormModal`/`BudgetConfigModal`, badge, tipos, api, i18n, `npm run build` | 1-2 días | Fase 1, 4 |
| 6 | Tests (unit + integration), verificación darkmode, migración en local con datos existentes, validación manual | 1 día | Fases 1-5 |

### 6.2 Milestones

1. **MVP (Fases 1-4)**: fix z-index + backend multi-servicio (migración, CRUD con `service_ids`, hook adaptado).
2. **V1.0 (Fases 5-6)**: multi-select en UI + badge + tests + polished.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| `ALTER TABLE DROP COLUMN` falla en el driver SQLite (down) | Baja | Bajo | Ya usado en `0036.down.sql`; verificar en local; si falla, recrear tabla |
| Datos existentes con `service_id` duplicados entre categorías (inconsistencia previa) | Baja | Medio | Índice único previo lo impidió; la migración falla si ocurre → revisar antes de aplicar |
| Multi-select complejo aumenta bundle en iHost | Media | Bajo | Componente ligero sin dependencias; checklist con búsqueda si `Select` multiple es complejo |
| Select con z-index fijo en `body` sigue quedando detrás en algún modal no cubierto | Media | Medio | CA-001/CA-SELECT; revisar todos los modales que usan `Select` (BudgetTransactionsPage, RegistrosPage, etc.) |
| Badge con muchos servicios ocupa espacio en fila | Media | Bajo | "+N" + tooltip (REQ-011 P2) |

## 8. Notas y Referencias

- SPEC-093 (módulo Presupuesto), SPEC-094 (vínculo categoría↔servicio, `service_id` 1:1, hook `OnBillPaid`, `source_bill_id`), SPEC-060 (darkmode inputs), SPEC-004 REQ-024 (dropdowns custom `Select`), SPEC-066 (worktrees).
- Archivos de referencia: `frontend/src/components/Select.tsx`, `frontend/src/components/IconPickerModal.tsx` (patrón prop `zIndex`), `frontend/src/pages/BudgetPage.tsx`, `internal/storage/category.go`, `internal/services/category.go`, `internal/services/budget_bill_link.go`, `migrations/0036_add_category_service_link.*`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-28 | opencode | Creación inicial de la especificación (requerimiento relevado con usuario: fix dropdown de servicio detrás del modal + soporte de múltiples servicios por categoría de presupuesto, ej. categoría "Internet Móvil" con servicios Claro y Tigo) |