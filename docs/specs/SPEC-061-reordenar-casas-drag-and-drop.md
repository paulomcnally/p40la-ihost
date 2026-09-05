---
title: "Reordenar casas con drag & drop en la página de Casas"
id: "SPEC-061"
status: "in_progress"
author: "paulomcnally"
created: "2026-09-04"
updated: "2026-09-04"
github_issue: 62
---

# Reordenar casas con drag & drop en la página de Casas

**ID**: SPEC-061  
**Estado**: in_progress  
**Autor**: paulomcnally  
**Creado**: 2026-09-04  
**Actualizado**: 2026-09-04

---

## 1. Resumen Ejecutivo

Actualmente las casas se listan en la página "Casas" (`HomesPage.tsx`) ordenadas alfabéticamente por nombre (`ORDER BY name` en `internal/storage/home.go:28`). El usuario quiere poder **arrastrar las cards de casas** para posicionarlas en un orden personalizado (primera, segunda, tercera, etc.), persistiendo ese orden.

La interacción debe adaptarse al dispositivo: en **desktop** (grid de 2-3 columnas) se arrastra hacia los costados o arriba/abajo; en **mobile** (una sola columna) se arrastra verticalmente en ambos sentidos. La feature se limita a la página de Casas, pero la spec documenta un **flujo genérico reutilizable** (migración + columna `sort_order` + endpoint de reordenamiento + componente sortable) para replicarlo después en otras páginas (servicios, autos, instituciones, etc.) cuando el usuario lo pida.

**Consideraciones iHost**: la base de datos sigue siendo SQLite (WAL). El cambio de esquema es una migración ligera (`ALTER TABLE` + backfill). No se agrega ninguna dependencia al runtime Go: el reordenamiento es una única transacción SQLite. La librería de drag & drop (`@dnd-kit`) es **solo del frontend estático**, que se compila fuera del iHost y se sirve como archivos estáticos; no impacta la RAM/CPU del runtime del servidor.

**Consideraciones de UI**: no se agregan inputs nuevos en formularios, por lo que no aplica la regla de tokens de darkmode a form controls. Sí se debe mantener el patrón de cards existente y verificar que el drag handle y los estados visuales (hover/arrastrando) sean legibles en darkmode.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: El usuario puede arrastrar una card de casa dentro de la cuadrícula de la página Casas para reordenarla (primero, segundo, tercero, etc.).
2. **REQ-002**: En desktop (grid `sm:grid-cols-2 lg:grid-cols-3`) el drag funciona hacia los costados, arriba y abajo (movimiento 2D dentro de la cuadrícula).
3. **REQ-003**: En mobile (una columna) el drag funciona de arriba a abajo y viceversa (movimiento vertical).
4. **REQ-004**: El nuevo orden se persiste en SQLite (columna `sort_order` en `homes`) y se mantiene tras recargar la página.
5. **REQ-005**: Las casas existentes reciben un `sort_order` de backfill que preserva el orden alfabético actual (para no reordenar de forma abrupta en el upgrade).
6. **REQ-006**: Al crear una casa nueva, se agrega al final del orden actual.
7. **REQ-007**: Al arrastrar, se aplica una actualización visual optimista (el orden cambia al instante) y, si el guardado falla, se revierte y se muestra un toast de error.
8. **REQ-008**: Toda la card es arrastrable (grab en cualquier punto). En desktop se muestra `cursor-grab` + elevación sutil en hover. El handle de agarre es una **franja vertical de alto completo al lado izquierdo de la card** con ícono de grip vertical centrado (pista visual siempre visible, con mayor énfasis en hover). Sin íconos flotantes en el centro o pegados a otros controles.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: El drag debe ser accesible también por teclado (tab + espacio/enter + flechas) usando el `KeyboardSensor` de dnd-kit, con el handle de agarre como activador.
2. **REQ-010**: Documentar el **flujo genérico de reordenamiento** (backend + frontend) en la spec para poder replicarlo en otras páginas (servicios, autos, instituciones, pensiones, etc.) en specs futuras.
3. **REQ-011**: En mobile, el drag se activa con **long-press** (~250-300ms) en cualquier parte de la card, de modo que el scroll nativo de la página no se interfiere. Los elementos interactivos (botón de menú de la card) se excluyen del drag con `data-no-dnd`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Toast de éxito ("Orden guardado") al persistir el nuevo orden.
2. **REQ-013**: Botón/acción de "restaurar orden alfabético" en el menú de la página.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El reorder endpoint debe ser una única petición con el array ordenado de IDs; la operación en SQLite es una transacción atómica con N updates (N = cantidad de casas, típicamente < 20).
- **Seguridad**: El endpoint de reorder requiere autenticación (mismo `authMiddleware` que el resto de rutas). Solo se actualizan IDs de casas activas del usuario; se ignora/valida cualquier ID inexistente o eliminado.
- **Almacenamiento**: Un solo `INTEGER` por fila en `homes`; sin tablas nuevas.
- **Disponibilidad**: Sin cambios de uptime; migración idempotente y reversible.
- **iHost**: Sin nuevas dependencias Go. El bundle del frontend crece ~25 kB gzip (solo se sirve como estático). Consumo de RAM del servidor inalterado.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- El estado actual ordena casas por nombre (`internal/storage/home.go:28` `ORDER BY name`), sin columna de posición.
- El modelo `Home` (`internal/models/home.go`) no tiene campo de orden.
- El frontend de Casas es un grid responsive (`grid-cols-1 sm:grid-cols-2 lg:grid-cols-3` en `HomesPage.tsx:56`).
- Dependencias actuales del frontend: solo `react`, `react-dom`, `react-router-dom`, `zustand`. No hay librería de drag & drop.
- Investigación sobre DnD (fuentes: clauderic/dnd-kit discussion #306, guías 2026 de dnd-kit, comparativas de librerías DnD React):
  - **HTML5 Drag and Drop API nativo**: solo mouse/pen; **no soporta touch** (Chrome, Firefox, Safari); sin keyboard; requiere código adicional y polyfills. Descartado por el requisito mobile.
  - **Pointer Events + dnd-kit**: un solo stream de eventos para mouse/touch/pen, sensor de teclado incluido, ~14 kB core / ~25 kB con sortable+modifiers (gzip). Estándar de facto para React sortable.
- Investigación de patrones UX de reordenamiento en apps TODO-list (NN/g "Drag-and-Drop: How to Design for Ease of Use", saasui.design, Tim Graf, y apps como Todoist/Things/Notion/Trello):
  - **Cards/objetos simples** → toda la card es arrastrable (no un handle en esquina); la señal es `cursor-grab` + elevación/sombra sutil en hover. Un grip fijo pegado a un botón (menú) genera "drag-versus-click confusion" (anti-patrón).
  - **Mobile** → el drag se inicia con **long-press** (~250-500ms); el scroll nativo no se interfiere. No hay hover.
  - **Handle como activador**: el grip sirve como activador de teclado y como pista visual (revelado en hover en desktop, siempre visible en mobile).
  - Elementos interactivos dentro de un objeto arrastrable se excluyen del drag (patrón `data-no-dnd`, usado en implementaciones reales con dnd-kit).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| `@dnd-kit/core` + `@dnd-kit/sortable` + `@dnd-kit/utilities` | Soporta mouse + touch + keyboard con una sola API; grid sortable 2D; árbol de dependencias pequeño; WCAG 2.1 AA; activo y mantenido | +~25 kB gzip al bundle del frontend (no impacta runtime iHost) | ✅ **Seleccionada** |
| HTML5 Drag and Drop nativo (sin dependencia) | Cero dependencias | No funciona en touch (falla el requisito mobile); sin keyboard; comportamiento inconsistente entre browsers | ❌ Rechazada |
| Implementación custom con Pointer Events | Cero dependencias; control total | Mucho código a mantener (colisiones, animaciones, a11y, edge cases touch/scroll); riesgo alto | ❌ Rechazada |
| `react-beautiful-dnd` | API amigable | Deprecada/mantenimiento detenido; soporte grid limitado | ❌ Rechazada |
| `react-dnd` | Madura | Requiere backend de touch aparte; a11y manual; complejidad extra | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Columna `sort_order` en la tabla `homes` para persistir el orden.
- **Contexto**: No existe columna de posición y el listado se ordena por nombre.
- **Decisión**: Agregar `sort_order INTEGER NOT NULL DEFAULT 0` vía migración `0026_add_homes_sort_order`, con backfill que preserva el orden alfabético actual y `ORDER BY sort_order ASC, name ASC` como tie-breaker.
- **Consecuencias**: Cambio de esquema mínimo y reversible (`.down.sql` hace `DROP COLUMN`). Las casas nuevas se insertan al final (se calcula `MAX(sort_order)+1` en storage).

**ADR-002**: Endpoint `PUT /api/homes/reorder` con el array completo de IDs en el nuevo orden.
- **Contexto**: Se necesita persistir el resultado de un drag. dnd-kit produce el array final ordenado tras soltar.
- **Decisión**: El cliente envía `{ "ids": [5, 2, 8, ...] }` (todos los IDs activos en el nuevo orden); el server asigna `sort_order = índice` dentro de una transacción y valida que los IDs pertenezcan a casas activas.
- **Consecuencias**: Una sola petición por drag, idempotente, simple de validar. Es el patrón estándar para listas sortables.

**ADR-003**: `@dnd-kit` para el frontend (drag con mouse, touch y teclado), con toda la card arrastrable.
- **Contexto**: El requisito exige drag en desktop (2D en grid) y mobile (vertical), además de accesibilidad por teclado. El patrón de UX investigado para cards (Todoist/Things/Notion/Trello) indica que el objeto completo debe ser arrastrable, con handle solo como activador de teclado/pista visual, y long-press en mobile.
- **Decisión**: Usar `@dnd-kit/core`, `@dnd-kit/sortable` y `@dnd-kit/utilities` con `MouseSensor` (distancia 6) + `TouchSensor` (long-press delay ~250ms) + `KeyboardSensor`. Sensores custom que rechazan la activación cuando el evento nace en un elemento `[data-no-dnd]` (para no interferir con el botón de menú de la card). El build de React se hace fuera del iHost; el incremento de bundle no afecta la memoria del servidor.
- **Consecuencias**: Interacción consistente en todas las entradas. El código de sortable queda aislado en componentes reutilizables (ver sección 4) para poder aplicarse a otras páginas.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[HomesPage (grid de cards)]
        |  drag (dnd-kit: PointerSensor + KeyboardSensor)
        v
[useSortableGrid hook / SortableCard]
        |  array final de ids
        v
[api.homes.reorder({ ids })]  ──PUT /api/homes/reorder──▶  [api/home_handlers.go]
                                                              |
                                                              v
                                                    [services/home.go: Reorder]
                                                              |
                                                              v
                                            [storage/home.go: ReorderTx (BEGIN/COMMIT)]
                                                              |
                                                              v
                                                       [SQLite homes.sort_order]
```

### 4.2 Componentes

#### 4.2.1 Migración `0026_add_homes_sort_order`
- **Responsabilidad**: Agregar `sort_order` y backfill con el orden alfabético actual.
- **Ubicación**: `migrations/0026_add_homes_sort_order.up.sql` / `.down.sql`

`0026_add_homes_sort_order.up.sql` (referencia de diseño):
```sql
ALTER TABLE homes ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

UPDATE homes
SET sort_order = (
  SELECT seq FROM (
    SELECT id, ROW_NUMBER() OVER (ORDER BY name) - 1 AS seq
    FROM homes WHERE deleted_at IS NULL
  ) ranked
  WHERE ranked.id = homes.id
)
WHERE deleted_at IS NULL;
```

`0026_add_homes_sort_order.down.sql`:
```sql
ALTER TABLE homes DROP COLUMN sort_order;
```

#### 4.2.2 Modelo y storage (backend)
- **`internal/models/home.go`**: agregar `SortOrder int64 \`json:"sort_order"\``.
- **`internal/storage/home.go`**:
  - `List`: `ORDER BY sort_order ASC, name ASC`.
  - `Create`: calcular `MAX(sort_order)+1` (o usar `SELECT COALESCE(MAX(sort_order), -1) + 1`) para insertar al final.
  - Nuevo `Reorder(ctx, ids []int64) error`: dentro de una transacción, para cada `idx, id` ejecuta `UPDATE homes SET sort_order = ? WHERE id = ? AND deleted_at IS NULL`. Validar que la cantidad de filas afectadas coincida con los IDs recibidos (todos deben existir y estar activos); si no, `ROLLBACK`.
- **`internal/services/home.go`**: `Reorder(ctx, ids)` con validación de negocio (lista no vacía, sin IDs duplicados).

#### 4.2.3 Handler y ruta (backend)
- **`internal/api/home_handlers.go`**: `ReorderHomes` que decodifica `{ "ids": [...] }`, valida formato y delega al service. Respuesta `200 {"message": "Orden actualizado"}`.
- **`internal/api/routes.go`**: `mux.Handle("PUT /api/homes/reorder", authMiddleware(...))`. **IMPORTANTE**: registrar ANTES de `PUT /api/homes/{id}` (o verificar el orden de patrones de Go 1.22 `http.ServeMux`, que prioriza el patrón más específico; `/api/homes/reorder` es literal y no conflictúa con `{id}`).

#### 4.2.4 Frontend — API y tipos
- **`frontend/src/types/index.ts`**: agregar `sort_order: number` a `Home`.
- **`frontend/src/api/index.ts`**: `reorder: (ids: number[]) => put('/api/homes/reorder', { ids })`.

#### 4.2.5 Frontend — componentes reutilizables de sortable (flujo genérico)
- **`frontend/src/components/sortable/SortableGrid.tsx`**: wrapper genérico que recibe `items: {id, ...}[]`, `onReorder(orderedIds)`, `children` (render de cada card) y usa `DndContext` + `SortableContext` con `rectSortingStrategy` (grid) o `verticalListSortingStrategy` (lista). Expone un prop `layout` para soportar grid (desktop) y lista (mobile) sin código duplicado. Configura los sensores: `MouseSensor` (distance 6), `TouchSensor` (delay 250/tolerance 8, long-press) y `KeyboardSensor` (`sortableKeyboardCoordinates`).
- **`frontend/src/components/sortable/dndSensors.ts`**: subclases `NoDndMouseSensor`/`NoDndTouchSensor` de dnd-kit que **rechazan la activación** cuando el evento nace en un elemento `[data-no-dnd]`. Permite tener toda la card arrastrable sin romper el tap/click de los botones internos (menú de la card).
- **`frontend/src/components/sortable/SortableCard.tsx`**: wrapper de `useSortable` que aplica `transform`/`transition` (vía `CSS.Transform.toString`), hace **toda la card arrastrable** (listeners en el nodo) con `cursor-grab` + elevación al arrastrar. El handle de agarre (grip) se usa como activador de teclado y como pista visual: sutil en hover (desktop) y siempre visible en mobile. Clase `group` para el efecto hover.
- **`frontend/src/hooks/useSortableOrder.ts`**: hook que recibe `items` y `apiReorderFn`; maneja el estado local del orden (optimistic update con `arrayMove`), llama a la API en `onDragEnd`, y revierte con toast de error si falla.

#### 4.2.6 Frontend — HomesPage
- Envolver el grid actual (`HomesPage.tsx:56`) en `SortableGrid` con `rectSortingStrategy` y las cards en `SortableCard`.
- **Toda la card es arrastrable**; se añade el handle de agarre (ícono `grip`) centrado arriba de la card: se revela en hover en desktop y es siempre visible en mobile. La card gana `cursor-grab`, elevación en hover y sombra al arrastrar.
- El botón de menú (`CardMenu`) recibe `data-no-dnd` (y `onKeyDown` con `stopPropagation`) para que su tap/click/teclado nunca inicien un drag.
- En `onDragEnd`: `arrayMove(homes, oldIndex, newIndex)` optimista + `api.homes.reorder(nuevosIds)`; rollback con toast de error en caso de fallo.
- Empty state y menú de creación (`CreateMenu`) permanecen sin cambios.

### 4.3 Modelo de datos

```
Entidad: homes (tabla existente)
- sort_order: INTEGER NOT NULL DEFAULT 0 (nuevo; 0 = primera posición)
- Relaciones: sin cambios
```

### 4.4 APIs / Contratos

#### Endpoint: `PUT /api/homes/reorder`

**Request**:
```json
{
  "ids": [3, 1, 2, 5, 4]
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

- **Internas**: `models.Home`, `storage.HomeStorage`, `services.HomeService`, `api.HomeHandlers`, `routes.go`, `appStore`/`api` del frontend, `HomesPage.tsx`, `i18n`.
- **Externas (frontend, solo build)**: `@dnd-kit/core`, `@dnd-kit/sortable`, `@dnd-kit/utilities` (+ `@dnd-kit/modifiers` solo si se necesita `restrictToParentElement`). Sin dependencias nuevas en Go.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: En desktop (≥2 columnas), arrastrar una card hacia los costados o arriba/abajo la recoloca en la posición prevista y el orden se mantiene tras recargar la página.
- [ ] CA-002: En mobile (1 columna), arrastrar hacia arriba/abajo recoloca la card y el orden persiste tras recargar.
- [ ] CA-003: El endpoint `PUT /api/homes/reorder` persiste `sort_order` correctamente (verificado contra SQLite con `PRAGMA`/query) y solo afecta casas activas.
- [ ] CA-004: Tras aplicar la migración, las casas existentes conservan su orden alfabético previo (backfill correcto).
- [ ] CA-005: Una casa nueva se agrega al final del orden.
- [ ] CA-006: Si el guardado del orden falla, la UI revierte al orden anterior y muestra un toast de error.
- [ ] CA-007: Pista visual de arrastre en hover (desktop: cursor grab + elevación) y siempre (mobile). El handle es una franja vertical de alto completo al lado izquierdo de la card, sin íconos flotantes en el centro ni pegados a otros controles.
- [ ] CA-008: El scroll mobile no se interfiere al arrastrar (drag por long-press; los controles con `data-no-dnd` siguen respondiendo a su tap).
- [ ] CA-009: El reordenamiento funciona por teclado (tab al handle, espacio/enter para levantar, flechas para mover, escape para cancelar).

### 5.2 No funcionales

- [ ] CA-NF-001: Reordenar N casas requiere 1 petición HTTP y 1 transacción SQLite.
- [ ] CA-NF-002: Sin nuevas dependencias en el runtime Go del iHost; el bundle del frontend crece ≤ ~30 kB gzip.
- [ ] CA-NF-003: El estado de la card arrastrada (sombra/opacidad) y el handle son legibles en darkmode.

### 5.3 Testing

- **Unit tests**:
  - `storage/home_test.go`: `Reorder` persiste orden, rechaza IDs inexistentes/duplicados (rollback), `List` ordena por `sort_order`.
  - `services/home_test.go`: validación de `Reorder`.
  - Migración `0026`: test que aplica up, verifica backfill alfabético y aplica down (`internal/db/migrate_026_test.go`).
- **Integration tests**: `PUT /api/homes/reorder` con autenticación devuelve 200; sin auth devuelve 401.
- **E2E tests** (manual en local): drag en grid desktop (2D), drag vertical mobile, persistencia tras reload, rollback con API simulada fallando, keyboard drag.
- **Carga/Performance**: con 10-20 casas, drag fluido (60fps) y reorder atómico en SQLite.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0026_add_homes_sort_order` (up/down + backfill) + test de migración | 0.5 días | Ninguna |
| 2 | Backend: `sort_order` en modelo, storage (`List`/`Create`/`Reorder`), service y handler `PUT /api/homes/reorder` + ruta + tests | 1 día | Fase 1 |
| 3 | Frontend: agregar `@dnd-kit/*`, componentes `SortableGrid`/`SortableCard`, hook `useSortableOrder`, api `homes.reorder`, tipo `sort_order` | 1 día | Fase 2 |
| 4 | Integrar en `HomesPage.tsx` (grid + handle + optimistic update + rollback + toast) + i18n (`home.reorder_hint`, `home.order_saved`, `home.order_error`) | 0.5 días | Fase 3 |
| 5 | Pruebas manuales locales (desktop, mobile, darkmode, keyboard), build del frontend, run server y validación con el usuario | 1 día | Fase 4 |
| 6 | Documentar en la spec el **flujo genérico** para replicar en otras páginas (checklist backend + frontend) | 0.5 días | Fase 5 |

### 6.2 Milestones

1. **MVP**: Fases 1-4 — reordenar casas con persistencia y UX completa.
2. **V1.0**: Fases 5-6 — validado con usuario y flujo genérico documentado para futuras specs.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Conflicto de ruta `PUT /api/homes/reorder` vs `PUT /api/homes/{id}` | Baja | Medio | Verificar el patrón más específico de `http.ServeMux`; si conflictúa, usar `POST /api/homes/reorder`. |
| El drag en mobile interfiere con el scroll nativo | Media | Alto | Activación por handle o `activationConstraint` (distancia/long-press) de dnd-kit. |
| Backfill de `sort_order` reordena casas de forma abrupta en producción | Baja | Medio | Backfill preservando `ORDER BY name`; probar la migración sobre copia local del backup antes de aplicar en iHost. |
| Rollback optimista complejo si la API falla a mitad del drag | Baja | Medio | Hook centralizado `useSortableOrder` que revierte con el array anterior y muestra toast. |
| Bundle del frontend crece y ralentiza carga en red lenta del iHost | Baja | Bajo | dnd-kit es tree-shakeable (~25 kB gzip); los estáticos se sirven desde el iHost sin costo de RAM. |

## 8. Notas y Referencias

- dnd-kit: https://github.com/clauderic/dnd-kit (discusión #306: por qué no usa HTML5 DnD; Pointer Events + TouchSensor + KeyboardSensor).
- dnd-kit sortable grid: `rectSortingStrategy` / `verticalListSortingStrategy`, `arrayMove`, `CSS.Transform.toString`, `sortableKeyboardCoordinates`.
- Migraciones existentes: `migrations/0025_add_currency_format_settings.*` es el último par (0001-0025); la nueva será `0026_*`. Mecanismo: `internal/db/db.go` aplica todos los `*.up.sql` en orden y registra en `schema_migrations`.

## 9. Flujo Genérico de Reordenamiento (para otras páginas)

> Objetivo: documentar el patrón para poder pedir "hacer lo mismo en [página]" en specs futuras. El flujo es idéntico salvo la entidad.

**Backend (checklist):**
1. Migración `NNNN_add_<entidad>_sort_order.up.sql`: `ALTER TABLE <entidad> ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;` + backfill que preserve el orden de visualización actual. `.down.sql`: `DROP COLUMN`.
2. `internal/models/<entidad>.go`: agregar campo `SortOrder`.
3. `internal/storage/<entidad>.go`: `ORDER BY sort_order ASC, <tie-breaker> ASC` en `List`; `MAX(sort_order)+1` en `Create`; método `Reorder(ctx, ids)` en transacción con validación de IDs activos.
4. `internal/services/<entidad>.go`: método `Reorder` con validación de negocio.
5. `internal/api/<entidad>_handlers.go`: handler `Reorder<Entidad>`; `routes.go`: `PUT /api/<entidad>/reorder` con auth.
6. Tests de storage/service/migración.

**Frontend (checklist):**
1. Agregar `sort_order` al tipo en `frontend/src/types/index.ts` y `reorder` al objeto `api` en `frontend/src/api/index.ts`.
2. Reutilizar `SortableGrid`/`SortableCard`/`useSortableOrder` (se crean en esta spec). Elegir `rectSortingStrategy` (grid) o `verticalListSortingStrategy` (lista vertical) según la página.
3. Envolver el grid/listado actual de la página y conectar `onReorder` al endpoint.
4. Agregar drag handle + hint visual + i18n (`<seccion>.reorder_hint`, `<seccion>.order_saved`, `<seccion>.order_error`).

**Criterios mínimos para una página candidata:** tiene un listado persistido en SQLite con una entidad propia y una página React que renderiza cards/filas.

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-04 | paulomcnally | Creación inicial de la especificación. Alcance: solo Casas; flujo genérico documentado en sección 9 para replicarlo en otras páginas a futuro. |
| 2026-09-04 | paulomcnally | Estado a `in_progress`. Inicio del desarrollo en rama `feature/SPEC-061` (worktree aislado). |
| 2026-09-04 | paulomcnally | Cambio de UX solicitado en evaluación manual: se reemplaza el handle fijo en la esquina (junto al menú) por **toda la card arrastrable** + handle revelado en hover (desktop) / siempre visible (mobile) + **long-press** en mobile, excluyendo controles interactivos con `data-no-dnd`. Basado en investigación de patrones de apps TODO-list (Todoist/Things/Notion/Trello, NN/g). |
| 2026-09-04 | paulomcnally | Ajuste de handle en evaluación manual: el handle pasa de estar centrado arriba a ser una **franja vertical de alto completo al lado izquierdo** de la card, con el ícono de grip vertical centrado y padding izquierdo en el contenido de la card para dejarlo visible. |