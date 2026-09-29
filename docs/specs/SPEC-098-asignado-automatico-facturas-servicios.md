---
title: "Auto-completar 'Asignado' de una categoría con la suma de la factura más reciente por servicio vinculado"
id: "SPEC-098"
status: "draft"
author: "opencode"
created: "2026-09-28"
updated: "2026-09-28"
github_issue: 101
---

# Auto-completar 'Asignado' de una categoría con la suma de la factura más reciente por servicio vinculado

**ID**: SPEC-098  
**Estado**: draft  
**Autor**: opencode  
**Creado**: 2026-09-28  
**Actualizado**: 2026-09-28

---

## 1. Resumen Ejecutivo

En el módulo de Presupuesto (SPEC-093/094/096) una categoría puede vincularse a uno o más servicios. Hoy, la columna **"Asignado"** (columna `assigned` de la vista mensual) solo se puebla manualmente a través del modal de asignación o por reglas recurrentes. El usuario pide un **auto-completado inteligente**: al abrir el modal de asignación de una categoría que tiene servicios vinculados y que **no tiene ningún monto asignado en ese mes**, el campo de monto debe venir pre-rellenado con la **suma de la factura más reciente de cada servicio vinculado** (1 factura por servicio). Esto ahorra trabajo manual recurrente: para una categoría tipo "Internet Móvil" con servicios Claro y Tigo, lo asignado sugerido sería el monto de la última factura de Claro + el de la última factura de Tigo.

Es una **sugerencia en el modal** (el usuario puede editarla antes de guardar), no una escritura automática en la DB: se decide así para no sobrescribir decisiones del usuario y mantener el flujo de confirmación existente (SPEC-093 REQ-007). El auto-completado aplica a **cada mes sin asignar** para esa categoría: si el mes ya tiene un `assignment`, no se toca; si no lo tiene, se pre-rellena la sugerencia.

**Consideraciones iHost**: el costo adicional es despreciable: al abrir el modal de asignación se ejecutan a lo sumo N+1 queries simples (`SELECT` de la factura más reciente por servicio, una por servicio vinculado, típicamente 1-3). Sin tablas nuevas, sin dependencias externas, sin cambios de esquema (se reutiliza la tabla `bills` existente, ordenando por `year DESC, month DESC`). El cálculo puede hacerse en el backend (endpoint dedicado o campo calculado en la vista mensual) o en el frontend si ya se dispone de los datos de las últimas facturas. Se evaluará en la fase de diseño (sección 3.2).

**Consideraciones de UI obligatorias**: el modal de asignación (`AssignModal` en `BudgetPage.tsx`) ya usa tokens del tema (`bg-card`, `text-text`) para su input de monto; la sugerencia se muestra como el valor pre-rellenado del input (editable), sin nuevos componentes. Verificar legibilidad en darkmode (SPEC-060). No se crean páginas de detalle nuevas, por lo que no aplica `BACK_ROUTES` (CA-BACK).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Al abrir el modal de asignación de una categoría que tiene **servicios vinculados** (`service_ids` no vacío) y que **no tiene assignment en el mes seleccionado**, el campo de monto se pre-rellena con la **suma de la factura más reciente por cada servicio vinculado** (1 factura por servicio, la de mayor `year`/`month` con `deleted_at IS NULL`). La sugerencia es editable antes de guardar.
2. **REQ-002**: El cálculo aplica a **cada mes sin asignar** para esa categoría. Si el mes ya tiene un `assignment` (monto asignado, de cualquier moneda), NO se pre-rellena (se muestra el monto existente, comportamiento actual).
3. **REQ-003**: La moneda de la sugerencia debe ser coherente con la categoría/servicios: la suma se agrupa por moneda (los servicios pueden tener `currency_id` distinto). Si los servicios vinculados tienen monedas distintas, se muestra la sugerencia de la moneda seleccionada en el `Select` de moneda del modal (o se calcula por moneda y se pre-selecciona la moneda del primer servicio). Definir el comportamiento exacto en el diseño (ver 3.3 ADR-002).
4. **REQ-004**: Si la categoría **no tiene servicios vinculados** o **ninguno de sus servicios tiene facturas** (historial vacío o todas soft-deleted), el campo de monto queda vacío/0 como hoy (sin sugerencia).
5. **REQ-005**: El endpoint/consulta que provee la sugerencia respeta la sesión (authMiddleware) y devuelve también un flag que indique si la sugerencia aplica (para que la UI distinga "sugerencia de sistema" de "sin valor"), sin confundir con un monto real.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Mostrar en el modal una **etiqueta/indicador** (ej. subtítulo) que aclare que el monto es una sugerencia automática basada en las últimas facturas de los servicios vinculados (ej. "Sugerido según última factura de Claro y Tigo"), con clave i18n en `frontend/public/i18n/{es,en}.json`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: Botón "Usar sugerencia" explícito que rellena el campo con el valor sugerido en caso de que el usuario lo haya borrado (opcional; si el pre-relleno inicial es suficiente, se descarta).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el cálculo agrega ≤ N+1 queries simples por apertura de modal (N = servicios vinculados, típicamente 1-3). Si se implementa en la vista mensual, debe evitarse N+1 (agregar a la query existente). Impacto despreciable en iHost.
- **Seguridad**: el endpoint que calcula la sugerencia bajo `authMiddleware` (sesión). Validación de IDs existente se reutiliza.
- **Almacenamiento**: sin tablas nuevas ni cambios de esquema. Se reutiliza `bills` existente.
- **Disponibilidad**: la sugerencia es solo frontend/lectura; si falla la query, el modal abre con monto vacío sin bloquear la asignación manual.
- **iHost**: sin dependencias externas nuevas (Go stdlib + SQLite). Build multi-arch intacto.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Requerimiento fuente**: solicitud del usuario (sesión 2026-09-28): *"si agrego los servicios a una categoría, en lo asignado debería auto agregarse el valor de la suma de las dos facturas más recientes"*. Relevado en entrevista: **1 factura más reciente por servicio** (no 2), la suma de las facturas más recientes de todos los servicios vinculados; **sugerencia pre-rellena en el modal de asignación** (no escritura automática); aplica a **cada mes sin asignar**.
- **Modelo actual (SPEC-093/094/096)**: `categories` con `service_ids`/`service_names` (via `category_service_links`), `account_id`, `target_amount` (decorativo, no alimenta `assigned`). `assignments(budget_month_id, category_id, currency_id, amount)` con UNIQUE; `assigned` = `SUM(amount)` de `assignments` (storage `SumByMonthGrouped`, `internal/storage/assignment.go:115-143`). La columna `assigned` solo se puebla manualmente (modal) o por recurrentes (`materialize`).
- **Bills**: tabla `bills` con columnas `service_id, year, month, amount, deleted_at, ...`. `internal/storage/bill.go:23-37` (`ListByService`) ordena `ORDER BY year DESC, month DESC` pero **sin LIMIT**; no existe helper "última factura por servicio" aún. Se necesita una nueva query (ej. `SELECT ... WHERE service_id = ? AND deleted_at IS NULL ORDER BY year DESC, month DESC LIMIT 1`).
- **Modal de asignación**: `AssignModal` en `frontend/src/pages/BudgetPage.tsx` (líneas 474-667): input de monto, `Select` de moneda, checkbox `make_recurring`; guarda via `api.budget.assign` (líneas 508-512). El monto se puede pre-rellenar desde el estado inicial del modal.
- **Vista mensual**: `internal/services/budget.go:93-208` (`MonthView`): `assigned`/`activity`/`available` por categoría+moneda. Los datos de servicios vinculados (`service_names`) ya viajan en cada `BudgetCategoryRow`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Calcular la sugerencia en el **backend** (endpoint dedicado `GET /api/budget/categories/{id}/suggested-assignment` o campo en MonthView) | Fuente de verdad centralizada, reutilizable, no expone datos de facturas al frontend | Un endpoint/query más | ✅ Seleccionada |
| Calcular en el **frontend** reutilizando la lista de facturas ya cargada | Cero endpoints nuevos | Requiere que el frontend ya tenga las últimas facturas por servicio (no las tiene hoy); lógica duplicada | ❌ Rechazada |
| Endpoint dedicado | Solo se ejecuta al abrir el modal (bajo costo) | Query adicional por apertura | ✅ Seleccionada (un endpoint nuevo) |
| Campo calculado embebido en `MonthView` | Sin request extra al abrir el modal | Se calcula para todas las categorías aunque no se abra el modal; complica el contrato de MonthView | ⚠️ Evaluada; puede agregarse en P2 si se desea una sola request |
| Escribir automáticamente el assignment (sin confirmación) | Cero interacción | Sobrescribe decisiones del usuario; contradice SPEC-093 REQ-007 | ❌ Rechazada (decisión usuario: sugerencia) |
| Pre-rellenar el input con la suma sugerida | Mínima fricción; el usuario confirma/edita | El usuario podría guardar sin revisar (aceptable) | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Sugerencia editable en el modal (no escritura automática en DB).
- **Contexto**: el usuario pidió que "se auto-agregue" el valor, pero al relevar eligió explícitamente "sugerencia en el modal (Recomendado)". El flujo de asignación existente es por confirmación manual (SPEC-093 REQ-007).
- **Decisión**: el campo de monto del `AssignModal` se inicializa con la suma sugerida (si aplica), editable. No se escriben `assignments` automáticamente.
- **Consecuencias**: se preserva el control del usuario; el auto-completado es una conveniencia de pre-relleno. Bajo riesgo de guardado sin revisar, mitigado con el indicador REQ-006.

**ADR-002**: 1 factura más reciente por servicio, suma agrupada por moneda.
- **Contexto**: el usuario aclaró que es "1 factura más reciente por cada servicio" (no 2 por servicio). Los servicios pueden tener monedas distintas (SPEC-093 multi-moneda).
- **Decisión**: para cada `service_id` vinculado se toma la última factura (`ORDER BY year DESC, month DESC LIMIT 1`), y se suma por moneda (`currency_id` del servicio). El modal recibe un mapa `{currency_id: total}`; se pre-selecciona el monto correspondiente a la moneda elegida en el `Select` (default: moneda del primer servicio, o la moneda de la cuenta si la hay).
- **Consecuencias**: si una categoría tiene servicios en 2 monedas, la sugerencia se muestra para la moneda seleccionada; cambiar la moneda en el `Select` muestra el total de esa moneda. Simple y consistente con el modelo multi-moneda.

**ADR-003**: Endpoint dedicado `GET /api/budget/categories/{id}/suggested-assignment`.
- **Contexto**: el frontend no tiene las últimas facturas por servicio; calcular en backend centraliza la regla.
- **Decisión**: nuevo endpoint bajo `authMiddleware` que devuelve `{ "suggested": { "<currency_id>": <monto> }, "applies": bool, "source_service_ids": [...], "source_service_names": [...] }`. `applies=false` si no hay servicios vinculados o ninguno tiene facturas.
- **Consecuencias**: una request solo al abrir el modal de asignación; costo despreciable. Alternativa P2: embeker en MonthView.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[BudgetPage → AssignModal (abre con categoría + mes)]
        │  GET /api/budget/categories/{id}/suggested-assignment
        ▼
[api.BudgetHandlers.SuggestedAssignment] --[services.BudgetService/ CategoryService]-->
        │
        ▼
[storage.BillStorage.LatestByService(serviceID)]  (bills ORDER BY year DESC, month DESC LIMIT 1)
        │   (1 query por servicio vinculado)
        ▼
[SQLite: bills + category_service_links + services]
        │
        ▼  respuesta JSON { suggested: {cur: monto}, applies, source_service_names }
[AssignModal: input de monto pre-rellenado + indicador "Sugerido según última factura de ..."]
```

### 4.2 Componentes

#### 4.2.1 Backend

- **`internal/storage/bill.go`**: nuevo método `LatestByService(ctx, serviceID) (*models.Bill, error)` — `SELECT ... FROM bills WHERE service_id = ? AND deleted_at IS NULL ORDER BY year DESC, month DESC LIMIT 1`.
- **`internal/services/category.go`** o **`internal/services/budget.go`**: método `SuggestedAssignment(ctx, categoryID) (*SuggestedAssignment, error)` que: obtiene la categoría (con `ServiceIDs`), por cada servicio obtiene `LatestByService`, suma por `currency_id` del servicio, arma el mapa `{currency_id: total}` y la lista `source_service_names`.
- **`internal/api/budget_handlers.go`**: handler `suggestedAssignment` + registro en `internal/api/routes.go` (`GET /api/budget/categories/{id}/suggested-assignment`, bajo `authMiddleware`).
- **`internal/models/`**: struct `SuggestedAssignment { Suggested map[int64]float64 `json:"suggested"`; Applies bool `json:"applies"`; SourceServiceIDs []int64 `json:"source_service_ids,omitempty"`; SourceServiceNames []string `json:"source_service_names,omitempty"` }` (definir ubicación exacta en desarrollo).

#### 4.2.2 Frontend

- **`frontend/src/pages/BudgetPage.tsx`**: `AssignModal` — al abrirse con una categoría sin assignment, si `category.service_ids?.length`, llama al endpoint de sugerencia y pre-rellena el input de monto (según moneda seleccionada). Mostrar indicador (REQ-006) con los `source_service_names`. Si `applies=false` o hay error, monto vacío.
- **`frontend/src/api/index.ts`**: método `budget.categories.suggestedAssignment(id)`.
- **`frontend/src/types/index.ts`**: tipo `SuggestedAssignment`.
- **`frontend/public/i18n/{es,en}.json`**: clave `budget.assigned_suggestion` (ej. "Sugerido según última factura de {services}") y correr `npm run build`. Fuente de verdad: `frontend/public/i18n/`, nunca `public/i18n/`.

### 4.3 Modelo de datos

Sin cambios de esquema. Se reutiliza `bills`:

```
Entidad: bills (existente)
- service_id: INTEGER → services(id)
- year, month: INTEGER  → periodocidad de la factura
- amount: REAL
- deleted_at: DATETIME (soft delete)
- Consulta: WHERE service_id = ? AND deleted_at IS NULL ORDER BY year DESC, month DESC LIMIT 1
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/budget/categories/{id}/suggested-assignment`

**Response 200** (aplica):
```json
{
  "suggested": { "1": 450.5 },
  "applies": true,
  "source_service_ids": [7, 12],
  "source_service_names": ["Claro Internet Móvil", "Tigo Internet Móvil"]
}
```

**Response 200** (no aplica — sin servicios o sin facturas):
```json
{ "suggested": {}, "applies": false, "source_service_ids": [], "source_service_names": [] }
```

**Response Error**:
```json
{ "error": "not_found", "message": "Categoría no encontrada" }
```

### 4.5 Dependencias

- **Internas**: `internal/storage/bill.go`, `internal/services/category.go`/`budget.go`, `internal/api/budget_handlers.go`, `internal/api/routes.go`, `frontend/src/pages/BudgetPage.tsx`, `frontend/src/api/index.ts`, `frontend/src/types/index.ts`, `frontend/public/i18n/{es,en}.json`.
- **Externas**: ninguna nueva.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Abrir el modal de asignación de una categoría con 2 servicios vinculados (ej. Claro y Tigo), ambos con facturas, en un mes **sin** assignment → el campo monto se pre-rellena con la suma de la última factura de cada servicio (1 por servicio). *(Verificar por API y UI.)*
- [ ] CA-002: Si el mes ya tiene un assignment para la categoría, el modal muestra el monto existente (no se pre-rellena ni se sobreescribe).
- [ ] CA-003: Si la categoría no tiene servicios vinculados o ninguno tiene facturas, el campo queda vacío (sin sugerencia, sin error).
- [ ] CA-004: La sugerencia es editable: el usuario puede cambiar el monto y guardar; se persiste el monto editado.
- [ ] CA-005: Con servicios en monedas distintas, el monto sugerido corresponde a la moneda seleccionada en el `Select` del modal.
- [ ] CA-006: El modal muestra un indicador claro (REQ-006) de que el monto es una sugerencia basada en las últimas facturas de los servicios (ej. "Sugerido según última factura de Claro y Tigo").
- [ ] CA-DARK: El input de monto usa tokens del tema (`bg-card`, `text-text`) y el indicador es legible en darkmode (SPEC-060). *(Código usa tokens; verificación visual darkmode pendiente de QA.)*
- [ ] CA-SELECT: El dropdown de moneda sigue usando el componente custom `Select`, nunca `<select>` nativo (SPEC-004 REQ-024).
- [ ] CA-BACK: No aplica (no hay páginas de detalle/formularios nuevos con lista padre; la sugerencia vive en el modal existente). Verificar que no se introducen links "← Título".

### 5.2 No funcionales

- [ ] CA-NF-001: El endpoint de sugerencia responde con ≤ N+1 queries simples (N = servicios vinculados); medir en local con 2-3 servicios.
- [ ] CA-NF-002: Sin dependencias externas nuevas; build multi-arch (`linux/amd64,linux/arm/v7,linux/arm64`) intacto.
- [ ] CA-NF-003: i18n actualizado en `frontend/public/i18n/{es,en}.json` y servido tras `npm run build` (verificar con `curl`).

### 5.3 Testing

- **Unit tests**: `BillStorage.LatestByService` (última factura por año/mes, ignora soft-deleted, sin facturas → nil); `SuggestedAssignment` (categoría sin servicios, 1 servicio, 2 servicios misma moneda, 2 servicios monedas distintas, sin facturas, categoría inexistente).
- **Integration tests**: `GET /api/budget/categories/{id}/suggested-assignment` con fixture de facturas; categoría con assignment existente (no aplica); auth requerida.
- **E2E tests**: en local: categoría "Internet Móvil" con Claro y Tigo (facturas existentes) → abrir modal de asignación de un mes sin asignar → ver monto pre-rellenado e indicador → editar/guardar → ver columna "Asignado" actualizada.
- **Carga/Performance**: con 3 servicios vinculados, medir tiempo del endpoint en local (despreciable).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `BillStorage.LatestByService` + service `SuggestedAssignment` (modelo, suma por moneda, `applies`) | 0.5 día | Ninguna |
| 2 | Handler `GET /api/budget/categories/{id}/suggested-assignment` + ruta | 0.25 día | Fase 1 |
| 3 | Frontend: API method, tipo, pre-relleno en `AssignModal`, indicador, i18n, `npm run build` | 0.5-1 día | Fase 2 |
| 4 | Tests (unit + integration), verificación darkmode, validación manual en local | 0.5 día | Fases 1-3 |

### 6.2 Milestones

1. **MVP (Fases 1-2)**: endpoint de sugerencia funcionando.
2. **V1.0 (Fases 3-4)**: pre-relleno en UI + indicador + tests + polished.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Confusión de "2 facturas vs 1 por servicio" | Media | Bajo | Relevado con usuario: **1 factura más reciente por servicio**; documentado en REQ-001 y ADR-002 |
| Monedas mixtas en una categoría → monto sugerido ambiguo | Media | Medio | Suma por moneda; el `Select` de moneda determina cuál se muestra (REQ-003, ADR-002) |
| El usuario guarda la sugerencia sin revisarla | Media | Bajo | Indicador visible "Sugerido según última factura de ..." (REQ-006) |
| Facturas de meses futuros o mal formadas | Baja | Bajo | Orden por `year DESC, month DESC`; opcional filtrar `year/month <= actual` si se detecta inconsistencia |
| Costo N+1 al abrir el modal | Baja | Bajo | N es 1-3 (servicios vinculados); despreciable en iHost; alternativas P2 (embeker en MonthView) |

## 8. Notas y Referencias

- SPEC-093 (módulo Presupuesto, `assigned/activity/available`, `assignments`), SPEC-094 (vínculo categoría↔servicio, hook `OnBillPaid`), SPEC-096 (multi-servicios por categoría, `category_service_links`), SPEC-060 (darkmode inputs), SPEC-004 REQ-024 (dropdowns custom `Select`), SPEC-066 (worktrees).
- Archivos de referencia: `internal/storage/bill.go` (`ListByService` ordenado sin LIMIT), `internal/storage/assignment.go` (`SumByMonthGrouped`), `internal/services/budget.go` (`MonthView`), `frontend/src/pages/BudgetPage.tsx` (`AssignModal`), `internal/api/routes.go`, `migrations/0035_create_budget_core.*`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-28 | opencode | Creación inicial de la especificación (requerimiento relevado con usuario: al agregar servicios a una categoría, auto-completar "Asignado" con la suma de la factura más reciente de cada servicio vinculado; como sugerencia editable en el modal; aplica a cada mes sin asignar; 1 factura por servicio, no 2) |