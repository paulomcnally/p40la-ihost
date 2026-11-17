---
title: "Reordenar grupos de presupuesto con drag & drop en la vista mensual"
id: "SPEC-095"
status: "released"
author: "opencode"
created: "2026-09-27"
updated: "2026-09-27"
github_issue: 98
---

# Reordenar grupos de presupuesto con drag & drop en la vista mensual

**ID**: SPEC-095  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-27  
**Actualizado**: 2026-11-17

---

## 1. Resumen Ejecutivo

El usuario quiere poder **arrastrar y soltar los grupos de categorías del presupuesto** hacia arriba y abajo dentro de la vista mensual (`/budget`) para ordenarlos "a su gusto y antojo", persistiendo ese orden. Hoy los grupos se listan verticalmente como `BudgetGroupTable` (cards con sus categorías) en el orden que devuelve el backend (`ORDER BY sort_order, id`), y **no hay ninguna forma de cambiar ese orden desde la UI**.

El trabajo es mayormente **frontend + un endpoint faltante**: el backend ya tiene el modelo (`category_groups.sort_order` desde SPEC-093) y las piezas de reordenamiento a nivel storage (`CategoryGroupStorage.Reorder`) y service (`CategoryGroupService.Reorder`), pero **no están conectadas a la capa API** (no existe `PUT /api/budget/category-groups/reorder`). El frontend ya cuenta con los componentes reutilizables de drag & drop (`SortableGrid`, `SortableCard`, hook `useSortableOrder`, sensores `NoDndMouseSensor`/`NoDndTouchSensor`) y la dependencia `@dnd-kit/*` instalada desde SPEC-061 (Casas). Por lo tanto esta spec reutiliza el flujo genérico documentado en SPEC-061 §9.

**Consideraciones iHost**: sin dependencias nuevas en el runtime Go (el reorder es una transacción SQLite sobre una columna existente) ni tablas nuevas. El bundle del frontend no crece (dnd-kit ya está incluido). Memoria/CPU despreciables. Se respeta el patrón de Casas (SPEC-061): drag con mouse + long-press en mobile + teclado, iniciado desde un handle de agarre.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El usuario puede **arrastrar un grupo de categorías hacia arriba o abajo** dentro de la vista mensual (`/budget`) para reordenarlo (primero, segundo, tercero, etc.). El orden se persiste en SQLite (`category_groups.sort_order`) y se mantiene tras recargar la página y al cambiar de mes.
2. **REQ-002**: El reorder aplica a **todos los grupos activos** y el endpoint persiste `sort_order` como índice del array recibido, en una única transacción atómica (una petición por drag, patrón de `PUT /api/homes/reorder` de SPEC-061).
3. **REQ-003**: Al **crear un grupo nuevo**, se agrega al final del orden actual (fix de `CategoryGroupService.Create`, que hoy inserta `sort_order = 0` y quedaría primero tras un reorder).
4. **REQ-004**: El drag se inicia **solo desde un handle de agarre** (franja vertical con ícono de grip, patrón de `SortableCard` de SPEC-061). El resto de la card del grupo (que incluye el `CardMenu` de 3 puntos) no es arrastrable y conserva su interacción.
5. **REQ-005**: Si el guardado del orden falla, la UI revierte al orden anterior y muestra un toast de error. El reorder se refleja al instante (actualización optimista).
6. **REQ-006**: El reorder funciona en mobile (drag vertical con long-press sin romper el scroll) y por teclado (tab + espacio/enter + flechas), usando los sensores estándar de dnd-kit ya configurados en `SortableGrid`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Toast de éxito ("Orden guardado") al persistir el nuevo orden, consistente con el patrón de i18n de Casas (`budget.reorder`, `budget.order_saved`, `budget.order_error`).
2. **REQ-008**: El orden reordenado también se respeta en el **modal de configuración de grupos/categorías** (`BudgetConfigModal`, que lista los grupos con la misma fuente backend `GET /api/budget/category-groups`): al abrirse, los grupos aparecen en el orden persistido (esto ya funciona vía `ORDER BY sort_order`; solo verificar).
3. **REQ-009**: Tests: storage/service del reorder de grupos (persistencia, rollback) y del fix de `Create` (apendiza al final); integración del endpoint `PUT /api/budget/category-groups/reorder`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Reordenar **categorías dentro de un grupo** con drag & drop (el backend `CategoryStorage.Reorder` y `CategoryService.Reorder` ya existen pero no tienen endpoint ni UI). Se deja documentado el diseño para una spec futura; no se implementa en esta spec.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Una petición HTTP + una transacción SQLite por drag (N updates, N = grupos, típicamente < 15). Sin impacto en la vista mensual (ya usa `ORDER BY sort_order`).
- **Seguridad**: Endpoint bajo `authMiddleware`. Solo se actualizan grupos existentes; se valida el formato del body.
- **Almacenamiento**: Sin tablas nuevas; solo se actualizan valores de `sort_order` ya existentes.
- **Disponibilidad**: Sin cambios de uptime ni schedulers.
- **iHost**: Sin dependencias nuevas en Go; el bundle del frontend ya incluye dnd-kit (SPEC-061), no crece.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- La tabla `category_groups` ya tiene `sort_order INTEGER NOT NULL DEFAULT 0` (migración `0035_create_budget_core`, SPEC-093). El modelo `models.CategoryGroup` tiene `SortOrder`.
- `internal/storage/category_group.go` ya tiene `Reorder(ctx, ids []int64) error` (línea 110): transacción que asigna `sort_order = i` a cada ID. `List` ya ordena `ORDER BY sort_order, id`.
- `internal/services/category_group.go` ya tiene `CategoryGroupService.Reorder(ctx, ids)` (línea 93), pero **ningún handler ni ruta la expone**. `routes.go` solo registra `GET/POST/PUT/DELETE /api/budget/category-groups[/{id}]`.
- `CategoryGroupService.Create` (línea 56) inserta con `sort_order = 0` → tras un reorder (donde los grupos quedan con 0..N), un grupo nuevo con `sort_order=0` aparecería **primero** por orden, no al final. Bug a corregir (REQ-003), siguiendo el patrón de `home.go` (`MAX(sort_order)+1`).
- El frontend ya tiene desde SPEC-061: `frontend/src/components/sortable/SortableGrid.tsx` (con `verticalListSortingStrategy` soportado), `SortableCard.tsx` (handle de agarre), `frontend/src/hooks/useSortableOrder.ts`, `dndSensors.ts`, y `@dnd-kit/*` en `package.json`. `HomesPage.tsx` es la referencia de integración.
- La vista mensual (`BudgetPage.tsx`) renderiza los grupos como `<BudgetGroupTable>` dentro de `<div className="space-y-4">` (línea 233): el wrapper ideal para `SortableGrid` con estrategia vertical. Los grupos vienen del payload `monthView.groups` (que ya está ordenado por `sort_order`).
- i18n: `frontend/public/i18n/{es,en}.json` tienen `home.reorder` / `home.order_error`; hay que agregar `budget.reorder`, `budget.order_saved`, `budget.order_error` (editar SIEMPRE en `frontend/public/i18n/`, ver AGENTS.md).
- API frontend: `frontend/src/api/index.ts` tiene `budget.categoryGroups.{list,create,update,delete}`; falta `reorder`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Reutilizar `SortableGrid`/`SortableCard`/`useSortableOrder` (SPEC-061) en la vista mensual | Cero código de DnD nuevo, patrón ya validado en Casas, keyboard + touch + mouse listos | Hay que integrar el handle dentro de la card de grupo existente | ✅ **Seleccionada** |
| Implementar reorder solo con controles de orden (flechas ↑↓ en el CardMenu) | Más simple de implementar | El usuario pidió explícitamente "arrastrar y soltar" | ❌ Rechazada (requerimiento explícito del usuario) |
| Crear componente DnD propio nuevo | Control total | Código duplicado con `SortableGrid`; riesgo alto | ❌ Rechazada |
| Persistir el orden solo en frontend (localStorage) | Cero cambios backend | No persiste entre dispositivos ni respeta el orden en el backend | ❌ Rechazada |
| Reordenar también categorías en esta spec | Completo de una vez | El usuario pidió solo "los grupos"; categorías es P2 y amplía riesgo | ❌ Rechazada en esta spec (P2, REQ-010) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Endpoint `PUT /api/budget/category-groups/reorder` con el array completo de IDs en el nuevo orden.
- **Contexto**: El backend ya tiene `CategoryGroupService.Reorder` y `CategoryGroupStorage.Reorder` sin exponer. dnd-kit produce el array final ordenado al soltar.
- **Decisión**: Se agrega el handler `ReorderCategoryGroups` en `budget_handlers.go` y la ruta `PUT /api/budget/category-groups/reorder` en `routes.go` (registrada junto a las demás de presupuesto). Body `{ "ids": [1, 3, 2, ...] }`; el server asigna `sort_order = índice` en una transacción. Es literal y no conflictúa con `PUT /api/budget/category-groups/{id}` (Go 1.22 `http.ServeMux` prioriza el patrón más específico).
- **Consecuencias**: Una sola petición por drag, idempotente, simple de validar. Reutiliza el código de reorder ya escrito en storage/service.

**ADR-002**: El drag se integra en la **vista mensual** (`BudgetPage.tsx`) envolviendo el listado de `BudgetGroupTable` en `SortableGrid` con `verticalListSortingStrategy`.
- **Contexto**: El usuario dijo "dentro de presupuesto... ordenar los grupos"; la vista mensual es donde se ven los grupos como cards verticales.
- **Decisión**: `BudgetGroupTable` se envuelve en `SortableCard` con un handle de agarre (franja izquierda con grip, patrón de `SortableCard`). `CardMenu` y demás controles internos conservan `data-no-dnd`. `onReorder` llama a `api.budget.categoryGroups.reorder` con actualización optimista (via `useSortableOrder` o equivalente inline) y reload de la vista tras guardar.
- **Consecuencias**: UX idéntica a Casas (SPEC-061). El handle requiere acomodar el padding izquierdo de la card de grupo para no solapar contenido.

**ADR-003**: Fix de `CategoryGroupService.Create` para apendizar al final del orden.
- **Contexto**: Hoy `Create` inserta `sort_order = 0`, lo que tras un reorder colocaría al grupo nuevo primero (orden `0,1,2,...`).
- **Decisión**: Calcular `MAX(sort_order)+1` al crear (patrón de `internal/storage/home.go:62`), insertando el grupo nuevo al final del orden vigente.
- **Consecuencias**: Comportamiento consistente con Casas (SPEC-061 REQ-006) y con la expectativa del usuario.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[BudgetPage.tsx (vista mensual)]
        |  drag (SortableGrid vertical + SortableCard + useSortableOrder)
        v
[api.budget.categoryGroups.reorder({ ids })]  ──PUT /api/budget/category-groups/reorder──▶  [budget_handlers.go]
                                                                                              |
                                                                                              v
                                                                                    [services.CategoryGroupService.Reorder]  (ya existe)
                                                                                              |
                                                                                              v
                                                                                [storage.CategoryGroupStorage.Reorder]  (ya existe, BEGIN/COMMIT)
                                                                                              |
                                                                                              v
                                                                                   [SQLite category_groups.sort_order]
```

### 4.2 Componentes

#### 4.2.1 Backend — handler y ruta
- **`internal/api/budget_handlers.go`**: nuevo `ReorderCategoryGroups` que decodifica `{ "ids": []int64 }`, valida (no vacío, sin duplicados) y delega a `h.categoryGroups.Reorder(ctx, ids)`. Respuesta `200 {"message": "Orden actualizado"}`.
- **`internal/api/routes.go`**: `mux.Handle("PUT /api/budget/category-groups/reorder", authMiddleware(http.HandlerFunc(handler.budget.ReorderCategoryGroups)))` (junto a las rutas de presupuesto, líneas 201-221).
- **`internal/services/category_group.go`**: fix de `Create` para apendizar al final (ADR-003). `Reorder` ya existe y no cambia.

#### 4.2.2 Frontend — API y tipos
- **`frontend/src/api/index.ts`**: agregar `reorder: (ids: number[]) => put<{ message: string }>('/api/budget/category-groups/reorder', { ids })` dentro de `budget.categoryGroups`.
- **`frontend/src/types/index.ts`**: `CategoryGroup` ya tiene `sort_order`; verificar que `BudgetMonthGroup` también (la vista mensual la usa).

#### 4.2.3 Frontend — BudgetPage (vista mensual)
- Envolver el contenedor `<div className="space-y-4">` de grupos (línea 233) en `SortableGrid` con `layout="vertical"`, pasando `items={view.groups}` y `onReorder={handleReorderGroups}`.
- Cada `<BudgetGroupTable>` se envuelve en `SortableCard` con `handle` (ícono `gripVertical`). Ajustar el padding izquierdo del header del grupo para dejar visible la franja (patrón `pl-10` de Casas). El `CardMenu` del grupo conserva `data-no-dnd` (patrón `SortableCard`/SPEC-061).
- `handleReorderGroups(ids)`: actualización optimista del orden local (arrayMove) + `api.budget.categoryGroups.reorder(ids)`; al éxito, toast `budget.order_saved` y `loadView(year, month)`; al fallo, rollback + toast `budget.order_error`. Reutilizar `useSortableOrder` si aplica al shape de datos.
- Hint visual de reordenamiento: ícono/etiqueta sutil (clave i18n `budget.reorder`) en el encabezado cuando hay ≥ 2 grupos.

#### 4.2.4 Frontend — i18n
- **`frontend/public/i18n/es.json`**: en el namespace `budget`: `"reorder": "Arrastrar para reordenar", "order_saved": "Orden guardado", "order_error": "No se pudo guardar el orden"`.
- **`frontend/public/i18n/en.json`**: mismos keys en inglés.
- Después de editar, correr `npm run build` (en `frontend/`) y verificar que las claves se sirven (AGENTS.md).

### 4.3 Modelo de datos

```
Entidad: category_groups (tabla existente, SPEC-093)
- sort_order: INTEGER NOT NULL DEFAULT 0 (ya existe; 0 = primera posición)
- Relaciones: sin cambios
```

No hay migraciones nuevas: la columna `sort_order` ya existe. Solo cambia el comportamiento de `Create` (apendiza al final) y se expone el endpoint de reorder.

### 4.4 APIs / Contratos

#### Endpoint: `PUT /api/budget/category-groups/reorder`

**Request**:
```json
{
  "ids": [3, 1, 2]
}
```

**Response 200**:
```json
{
  "message": "Orden actualizado"
}
```

**Response Error 400**:
```json
{
  "error": "invalid_request",
  "message": "ids requeridos"
}
```

### 4.5 Dependencias

- **Internas**: `internal/api/budget_handlers.go`, `internal/api/routes.go`, `internal/services/category_group.go` (fix Create), `frontend/src/api/index.ts`, `frontend/src/pages/BudgetPage.tsx`, `frontend/src/components/sortable/*` (ya existentes), `frontend/public/i18n/{es,en}.json`.
- **Externas**: ninguna nueva. `@dnd-kit/*` ya está en `package.json` (SPEC-061). Go stdlib + `modernc.org/sqlite` ya presentes.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: En la vista mensual (`/budget`), arrastrar un grupo hacia arriba/abajo lo recoloca y el orden persiste tras recargar la página y al navegar entre meses.
- [ ] CA-002: `PUT /api/budget/category-groups/reorder` persiste `sort_order` correctamente (verificado con query a SQLite) y respeta el orden del array recibido.
- [ ] CA-003: Un grupo nuevo se agrega al final del orden (tras reordenar, crear un grupo lo deja último).
- [ ] CA-004: El drag se inicia SOLO desde el handle (franja izquierda con grip); el resto de la card (incluido el `CardMenu`) no es arrastrable y responde a su tap/click.
- [ ] CA-005: Si el guardado del orden falla, la UI revierte al orden anterior y muestra toast de error.
- [ ] CA-006: El reorder funciona en mobile (long-press, sin romper el scroll) y por teclado (tab + espacio/enter + flechas).
- [ ] CA-007: Toasts de éxito/error en español e inglés (`budget.order_saved` / `budget.order_error`) servidos tras `npm run build`.
- [ ] CA-BACK: No aplica páginas de detalle nuevas (la vista mensual es raíz `/budget`, sin flecha atrás; no se agregan sub-rutas de formulario).
- [ ] CA-DARK: El handle de agarre y el estado de la card arrastrada (sombra/opacidad) son legibles en darkmode.

### 5.2 No funcionales

- [ ] CA-NF-001: Reordenar N grupos requiere 1 petición HTTP y 1 transacción SQLite.
- [ ] CA-NF-002: Sin dependencias nuevas en el runtime Go ni en el bundle del frontend.
- [ ] CA-NF-003: La vista mensual sigue cargando en < 300ms con datos típicos (el reorder no agrega queries al render).

### 5.3 Testing

- **Unit tests**: `CategoryGroupStorage.Reorder` (persistencia de orden, rollback si falla); `CategoryGroupService.Create` apendiza al final (`MAX(sort_order)+1`); `CategoryGroupService.Reorder` valida lista no vacía/sin duplicados.
- **Integration tests**: `PUT /api/budget/category-groups/reorder` autenticado devuelve 200 y persiste; sin auth devuelve 401; body inválido devuelve 400.
- **E2E tests** (manual en local): drag vertical en la vista mensual, persistencia tras reload, rollback con API fallando, keyboard drag, darkmode, mobile (long-press).
- **Carga/Performance**: con 10-15 grupos, drag fluido y reorder atómico en SQLite.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Backend: handler `ReorderCategoryGroups` + ruta `PUT /api/budget/category-groups/reorder` + fix de `Create` (MAX+1) + tests | 0.5-1 día | Ninguna |
| 2 | Frontend: `api.budget.categoryGroups.reorder` + integrar `SortableGrid`/`SortableCard` en `BudgetPage` (handle, optimistic update, rollback, toasts) | 1 día | Fase 1 |
| 3 | i18n (`budget.reorder/order_saved/order_error` en `es/en`), build del frontend, verificación de claves servidas | 0.5 día | Fase 2 |
| 4 | Pruebas manuales locales (desktop, mobile, darkmode, keyboard), run server y validación con el usuario | 1 día | Fase 3 |

### 6.2 Milestones

1. **MVP**: Fases 1-3 — reordenar grupos con persistencia y UX completa.
2. **V1.0**: Fase 4 — validado con el usuario. (P2: reordenar categorías dentro de grupos queda para spec futura, REQ-010.)

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Conflicto de ruta `PUT /api/budget/category-groups/reorder` vs `PUT /api/budget/category-groups/{id}` | Baja | Medio | `http.ServeMux` prioriza el patrón literal más específico; test de integración cubre ambos. |
| El handle de la card de grupo se solapa con el `CardMenu` o el título | Media | Medio | Franja izquierda con padding `pl-*` en el header del grupo (patrón Casas) + `data-no-dnd` en el `CardMenu`. |
| Grupo nuevo aparece primero tras reordenar (bug actual) | Alta | Medio | Fix de `Create` con `MAX(sort_order)+1` (ADR-003) + test. |
| El drag en mobile interfiere con el scroll nativo | Media | Alto | Handle como activador + long-press (250ms) en el `TouchSensor` ya configurado en `SortableGrid`. |
| Rollback optimista complejo si la API falla a mitad del drag | Baja | Medio | `useSortableOrder`/arrayMove con rollback al array previo + toast de error. |

## 8. Notas y Referencias

- SPEC-061 (flujo genérico de reordenamiento §9, dnd-kit, Casas como referencia), SPEC-093 (módulo presupuesto, `category_groups`), SPEC-094 (vínculo categorías-servicios).
- `internal/storage/category_group.go` (`Reorder` ya implementado), `internal/services/category_group.go` (`Reorder` ya implementado, `Create` a corregir), `internal/api/routes.go` (líneas 201-221, rutas presupuesto), `internal/api/budget_handlers.go`.
- `frontend/src/pages/BudgetPage.tsx` (línea 233, listado de grupos), `frontend/src/components/sortable/SortableGrid.tsx`, `SortableCard.tsx`, `frontend/src/hooks/useSortableOrder.ts`, `frontend/src/api/index.ts` (línea 298+).
- i18n fuente: `frontend/public/i18n/{es,en}.json` (nunca `public/i18n/` — AGENTS.md).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-27 | opencode | Creación inicial de la especificación. Requerimiento del usuario: "dentro de presupuesto debería poder arrastrar y soltar hacia arriba y abajo para ordenar a mi gusto y antojo los grupos". Alcance: solo grupos en la vista mensual; categorías dentro de grupos queda como P2 (REQ-010). |
| 2026-11-17 | opencode | Release: rebase sobre `main` (incluye SPEC-136), rebuild del frontend, `go build` y tests OK; merge a `main`. |

---

## 10. Verificación de release

| Criterio | Evidencia |
|---|---|
| Compilación backend | `go build ./...` OK |
| Tests backend | `go test ./internal/services/ ./internal/storage/` OK |
| Tests nuevos | 6 tests de reorder (servicio y storage) PASS |
| Compilación frontend | `npm run build` (check:ui + check:pwa + tsc + vite) OK |
| Guardrails UI | `check-ui-rules.cjs` OK (sin `<select>` ni checkbox nativo) |
| i18n | Solo 3 claves nuevas en `frontend/public/i18n/{es,en}.json`; build regenera `public/i18n/` y `frontend/src/i18n/` |
| Integración | Rebase sobre `main` sin pérdida de reglas ni cambios de SPEC-131..136 |
