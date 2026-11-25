---
title: "MCP: alta de reparaciones sin líneas y movimiento de insumos/mano de obra entre reparaciones"
id: "SPEC-157"
status: "in_progress"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 160
---

# MCP: alta de reparaciones sin líneas y movimiento de insumos/mano de obra entre reparaciones

**ID**: SPEC-157<br>
**Estado**: in_progress<br>
**Autor**: opencode<br>
**Creado**: 2026-11-24<br>
**Actualizado**: 2026-11-24<br>

---

## 1. Resumen Ejecutivo

El módulo `home.maintenance` registra reparaciones del hogar con líneas de **mano de obra** (`labor`) e **insumos** (`supplies`). SPEC-155 agregó las tools MCP `home_maintenance_labor_register` y `home_maintenance_supplies_register`, pero ambas **exigen** una línea: no existe forma de crear una reparación vacía (0 insumos y 0 mano de obra). Además, si una línea se registra por error en la reparación equivocada, el agente solo puede borrarla y volver a crearla; no hay una operación de **mover** que preserve el registro original.

Esta spec agrega dos tools MCP de escritura y un endpoint transaccional:

1. `home_maintenance_create`: crea **solo** la reparación, sin líneas.
2. `home_maintenance_item_move`: mueve una o varias líneas (mano de obra y/o insumos) de una reparación origen a una destino del mismo hogar. El destino puede existir (`target_maintenance_id`) o crearse en la misma llamada (`target`), de forma atómica.
3. `POST /api/homes/{id}/maintenances/{maintenance_id}/items/move`: endpoint del módulo que ejecuta el movimiento en una transacción.

Ambas tools respetan `dry_run` (default `true`) y la compuerta `P40LA_MCP_ALLOW_WRITES` / `P40LA_MCP_REMOTE_WRITES`. El movimiento valida que las líneas pertenezcan al origen, prohíbe mover al mismo mantenimiento y solo borra el origen vacío si el llamador lo pide explícitamente (`delete_source_if_empty=false` por defecto).

El cambio no agrega dependencias ni migraciones. En el iHost el costo es mínimo: una consulta de detalle, una transacción corta con un `UPDATE` acotado a 100 líneas y una consulta final para devolver las líneas movidas. No toca frontend.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec toca backend, MCP y SQLite; **no toca frontend, rutas de UI, i18n ni migraciones**. Las reglas de UI/UX se declaran `N/A` con justificación.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go stdlib + SQLite; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | El cliente MCP vive en `internal/mcp/api/`; las tools en `internal/mcp/tools/homemaint/`; el dominio sigue en `internal/modules/home/maintenance/` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Tool MCP → cliente HTTP → handler → service → storage. **Prohibido** SQL desde el MCP; el módulo conserva `handlers.go`, `service.go`, `storage.go` | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | `home.maintenance` mantiene su namespace y no importa storage de otros módulos; el movimiento transaccional usa `database/sql` dentro de su propio storage | `CA-ARCH-MODULES` |
| `ARCH-MCP-TOOLS-MAP` | Las dos tools nuevas se registran con nombre literal y entrada en `docs/mcp/tools/impact.json`; el contrato se regenera de forma intencional | `CA-ARCH-MCP-TOOLS-MAP` |
| `ARCH-IHOST` | Sin caché ni estado en el MCP; máximo 100 líneas por movimiento; una transacción corta; respuesta acotada; cliente corta en 1 MiB | `CA-ARCH-IHOST` |
| `ARCH-GO` | `gofmt`, nombres en inglés, errores explícitos, `log/slog` sin cambios | `CA-ARCH-GO` |
| `DATA-SQLITE` | Se reutilizan `home_maintenances` y `home_maintenance_items`; SQL crudo con `database/sql` y transacción para mover/crear el destino | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | **N/A**: no hay cambios de esquema; las tablas e índices de `0057_home_maintenance` alcanzan | — |
| `DATA-PROTECTED-SCHEMA` | **N/A**: no se toca `debts` / `debt_bills` | — |
| `DATA-NO-DROP` | **N/A**: no se ejecuta nada sobre `data/app.db` | — |
| `DATA-PRODCOPY` | **N/A**: la validación usa DB temporal y fixtures sintéticos; no depende de datos reales | — |
| `DATA-PROD-WRITE` | **N/A**: no se escribe en producción | — |
| `SEC-NO-PII` | Las tools devuelven descripciones libres y montos del propio usuario; no agregan emails, teléfonos ni identificadores. Los ejemplos son sintéticos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Casa Demo`, `Taller Demo`, `Proveedor Demo`, montos ficticios y fechas `2026-03-10`/`2026-03-11` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Los logs del MCP no agregan argumentos ni montos; la auditoría existente no cambia | `CA-SEC-LOGS` |
| `SEC-COMMIT` | `git diff --cached` se revisa; sin DBs, PDFs, exports ni secretos | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks instalados; sin `--no-verify`; el gate MCP se corre antes del commit | `CA-SEC-HOOKS` |
| `UI-*` (todas) | **N/A**: la spec no toca frontend ni i18n | — |
| `UX-BACK` | **N/A**: no se agregan rutas ni pantallas | — |
| `UX-MODAL` / `UX-MODAL-CONFIRM` | **N/A**: no se agregan modales | — |
| `UX-DETAIL-ACTIONS` / `UX-DETAIL-READONLY` | **N/A**: no se modifica ninguna página de detalle | — |
| `UX-PREREQ` | Aplica al backend MCP: `home_id`, `source_maintenance_id`, `target_maintenance_id`, `currency_id` e `institution_id` se validan antes de escribir; el destino nuevo usa las mismas validaciones que el alta | `CA-UX-PREREQ` |
| `UX-EMPTY` / `UX-LOADING` / `UX-FEEDBACK` / `UX-FORMS` | **N/A**: no hay UI; el feedback del MCP se expresa con `status`, `warnings` y errores estables | — |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` antes del commit; pruebas con DB temporal | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | Se actualiza `docs/mcp/tools/contracts.json` de forma intencional y se corren `TestToolContracts` + `TestMCPBackendContract` | `CA-TEST-MCP-CONTRACT` |
| `TEST-TESTUSER` | **N/A**: las pruebas del módulo y del contrato usan `httptest` y DB temporal | — |
| `TEST-SERVER` | Si el usuario pide probar en local, `./scripts/check-server.sh` antes de avisar | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | No se marca la spec como completada ni se cierra sin la confirmación explícita del usuario | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | La spec vive en `../p40la-ihost-spec-157` sobre `feature/SPEC-157`; `./scripts/check-worktree.sh` pasa | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push se hacen con la autorización explícita del usuario y sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa; `./scripts/check-mcp-impact.sh --files ...` y `--staged --gate` antes del commit | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; veredicto citado en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-MCP-DOCS` | `docs/mcp/tools/home-maintenance.md`, `README.md`, `docs/mcp/README.md` y `security.md` se actualizan en el mismo commit | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente en paralelo a la implementación | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-COMMIT` | Commits `SPEC-157: ...` | `CA-DELIV-COMMIT` |
| `DELIV-HOOKS` | Hooks instalados; `./scripts/install-hooks.sh` al crear el worktree | `CA-DELIV-HOOKS` |
| `DELIV-LABELS` | Issue con `spec/draft` y transiciones a `spec/in-progress`, `spec/pending-release` y `spec/released` | `CA-DELIV-LABELS` |
| `DELIV-RELEASE` | Merge, push, tracker, label, cierre de issue y limpieza del worktree en el mismo ciclo | `CA-DELIV-RELEASE` |
| `DELIV-DOCKER` | **N/A**: esta spec no publica imagen ni cambia el Dockerfile; la imagen MCP no se publica en este ciclo | — |
| `DELIV-DEPLOY` | **N/A**: no se despliega al iHost en esta spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Exponer `home_maintenance_create` para crear una reparación **sin líneas** (0 insumos y 0 mano de obra). La tool no debe exigir `currency_id` ni `amount`.
2. **REQ-002**: Exponer `home_maintenance_item_move` para mover una o más líneas existentes (`labor` y/o `supplies`) de una reparación origen a una destino, dentro del mismo `home_id`.
3. **REQ-003**: El destino del movimiento debe resolverse de exactamente una de estas dos formas:
   - `target_maintenance_id`: reparación existente del mismo `home_id`;
   - `target`: objeto con `performed_at`, `kind`, `area`, `institution_id` y `description` para crear la reparación destino.
   Si no se envía ninguno o se envían ambos, la tool devuelve un error estable y no escribe.
4. **REQ-004**: Validar que **todas** las líneas de `item_ids` existan y pertenezcan a la reparación origen y al `home_id`. Si una línea no pertenece al origen, no se mueve ninguna.
5. **REQ-005**: Prohibir mover líneas a la misma reparación origen (`same_maintenance`).
6. **REQ-006**: Ejecutar el movimiento en una **única transacción**: crear el destino (si aplica) + actualizar `home_maintenance_items.maintenance_id`. Si la actualización no afecta exactamente las líneas pedidas, se revierte todo (`move_conflict`).
7. **REQ-007**: `dry_run` viene activado por defecto en ambas tools. En modo preview no se escribe y se devuelve el plan, las líneas que se moverían y las que quedarían en el origen.
8. **REQ-008**: Respetar la compuerta de escritura existente: `P40LA_MCP_ALLOW_WRITES=true` (y `P40LA_MCP_REMOTE_WRITES=true` en HTTP). Sin la compuerta, `dry_run=false` devuelve `writes_disabled`.
9. **REQ-009**: Agregar el endpoint `POST /api/homes/{id}/maintenances/{maintenance_id}/items/move` con el contrato definido en 5.4.
10. **REQ-010**: Devolver `source_item_count` y `source_deleted` para que el agente sepa si el origen quedó vacío o se borró.
11. **REQ-011**: Marcar las escrituras con `X-P40LA-Source: mcp` mediante el cliente existente.
12. **REQ-012**: Actualizar manifiesto, contrato congelado, stubs de `TestMCPBackendContract` y documentación MCP en el mismo commit.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: `home_maintenance_item_move` acepta `item_type` opcional (`labor`/`supplies`) como salvaguarda: si viene, todas las líneas movidas deben ser de ese tipo; si no, devuelve `invalid_request` sin escribir.
2. **REQ-014**: `delete_source_if_empty` opcional, default `false`. Solo borra la reparación origen si, después del movimiento, queda con **cero** líneas. Si quedan líneas, no borra y devuelve `source_deleted=false` con un warning.
3. **REQ-015**: Limitar `item_ids` a **100** elementos, sin duplicados y con IDs positivos.
4. **REQ-016**: Devolver `target_created=true` cuando la reparación destino se creó en la misma llamada.
5. **REQ-017**: `home_maintenance_create` valida `performed_at` (formato y no futura), `kind`, `area`, `institution_id` y la longitud de `description` (máximo 500).

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-018**: Evaluar en una spec futura una acción de UI para mover líneas entre reparaciones. Esta spec se limita al MCP y al endpoint backend.
2. **REQ-019**: Evaluar en una spec futura un `dedup_key` persistido si se necesita idempotencia estricta del movimiento ante reintentos concurrentes.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: una consulta de detalle del origen, una transacción corta (opcional `INSERT` + `UPDATE` acotado a 100 filas) y una consulta final de las líneas movidas. Sin caché ni estado.
- **Memoria**: máximo 100 líneas por movimiento y respuesta acotada; el cliente MCP ya corta en 1 MiB.
- **Seguridad**: los endpoints siguen autenticados por API key/sesión; las tools no agregan secretos ni PII.
- **Almacenamiento**: no hay migraciones ni archivos nuevos de datos.
- **Disponibilidad**: si falla el `UPDATE` o la creación del destino, la transacción revierte y no queda un destino vacío ni líneas a medio mover.
- **iHost**: sin dependencias nuevas; una transacción por movimiento; sin procesos en background.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `internal/modules/home/maintenance/model.go` define `Maintenance`, `Item`, `Input`, `CreateInput` e `ItemInput`; `quantity` y `unit` solo aplican a insumos.
- `POST /api/homes/{id}/maintenances` ya acepta crear un mantenimiento sin `item` (`CreateWithItem(..., nil)`). **La limitación de las 0 líneas es de la tool MCP, no del backend.**
- `internal/mcp/tools/homemaint/{labor,supplies}.go` exigen `description`, `amount` y `currency_id`; no existe `home_maintenance_create`.
- `internal/modules/home/maintenance/storage.go` solo usa transacción en `CreateWithItem` (INSERT mantenimiento + INSERT línea).
- `home_maintenance_list` ya devuelve las líneas con `include_items=true`, lo que permite al agente armar `item_ids` antes de mover.
- El analizador `scripts/mcp-impact/main.go` marca todas las tools cuando se toca `internal/mcp/api/` o `internal/mcp/tools/all/` por `shared.affects_all`.
- No hay precedente de tool MCP de reubicación; sí hay transacciones multi-fila en `internal/modules/bank/creditcard/storage.go` y `internal/modules/realestate/rental/storage_billing.go`.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Extender `labor_register`/`supplies_register` con un modo «sin línea» | Cero tools nuevas | Rompe la semántica de «registrar línea»; schemas con campos opcionales contradictorios | ❌ Rechazada |
| B. `home_maintenance_create` separada + mover con dos llamadas (crear destino y luego mover) | Tools simples | Si falla el segundo paso queda un destino vacío; el usuario pidió crearlo y mover | ❌ Rechazada |
| C. `home_maintenance_create` + `home_maintenance_item_move` con destino existente o creado en la misma transacción | Atómico; cubre los dos casos del usuario; reutiliza el alta existente | Endpoint nuevo y más validaciones | ✅ Seleccionada |
| D. Una sola tool `home_maintenance_manage` con acciones create/move | Menos tools | Contrato ambiguo, difícil de descubrir y de testear | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: alta vacía como tool propia**
- **Contexto**: el backend ya crea mantenimientos sin líneas; las tools de registro no lo permiten porque su contrato exige una línea.
- **Decisión**: `home_maintenance_create` llama a `POST /api/homes/{id}/maintenances` sin `item`. No se modifican las tools de registro.
- **Consecuencias**: el contrato de las tools existentes no cambia; el catálogo crece en una tool de escritura. Una reparación vacía del día puede ser reutilizada después por `home_maintenance_labor_register`/`home_maintenance_supplies_register` al resolver el mantenimiento del mismo día y área.

**ADR-002: movimiento atómico en el backend**
- **Contexto**: crear el destino y mover en dos llamadas deja un destino vacío si la segunda falla. El `UPDATE` de `maintenance_id` es la única operación que preserva la línea y sus importes.
- **Decisión**: el endpoint `.../items/move` hace `INSERT` opcional del destino y `UPDATE ... WHERE maintenance_id = origen AND id IN (...)` dentro de una transacción; si `RowsAffected != len(item_ids)` revierte con `move_conflict`.
- **Consecuencias**: no hay estados intermedios; el origen y el destino siempre son consistentes. La validación previa de pertenencia evita mover líneas de otro hogar.

**ADR-003: destino explícito, sin resolución automática por fecha**
- **Contexto**: las tools de registro resuelven el mantenimiento del día y pueden devolver `ambiguous_maintenance`. Para un movimiento, elegir el destino equivocado es peor que fallar.
- **Decisión**: el destino se indica con `target_maintenance_id` o con un objeto `target` explícito; no se resuelve por fecha/área.
- **Consecuencias**: el agente debe listar primero (`home_maintenance_list?date=&include_items=true`) para conocer el ID correcto. Menos automatismo, más seguridad.

**ADR-004: borrado del origen vacío opt-in**
- **Contexto**: mover todas las líneas deja una reparación sin importes. Borrarla automáticamente podría eliminar un registro que el usuario quiere conservar.
- **Decisión**: `delete_source_if_empty=false` por defecto; si es `true`, el backend borra el origen solo si quedó con 0 líneas, dentro de la misma transacción.
- **Consecuencias**: el caso normal conserva el origen vacío y lo reporta (`source_item_count=0`, `source_deleted=false`); el borrado es explícito y queda documentado.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
Agente MCP
   |  home_maintenance_create / home_maintenance_item_move
   v
internal/mcp/tools/homemaint  --(cliente HomeMaintenanceAPI)-->  API HTTP
   |                                                                 |
   |  GET  /api/homes/{id}/maintenances/{mid}                         v
   |  POST /api/homes/{id}/maintenances                    handlers.go (home.maintenance)
   |  POST /api/homes/{id}/maintenances/{mid}/items/move            |
   +----------------------------------------------------------> service.go
                                                                     |
                                                                storage.go
                                                                     |
                                                          SQLite: home_maintenances
                                                                  home_maintenance_items
```

### 5.2 Componentes

#### 5.2.1 Módulo `home.maintenance` (backend)

- **Responsabilidad**: validar el origen, el destino y las líneas; ejecutar el movimiento transaccional; devolver las líneas movidas y el estado del origen.
- **Interfaz**:
  - `POST /api/homes/{id}/maintenances` (sin cambios; ya permite `item` omitido)
  - `POST /api/homes/{id}/maintenances/{maintenance_id}/items/move` (nuevo)
  - `GET /api/homes/{id}/maintenances/{maintenance_id}` (sin cambios)
- **Dependencias**: `database/sql`, tablas `home_maintenances`, `home_maintenance_items`, `homes`, `currencies`, `institutions`.
- **Ubicación**: `internal/modules/home/maintenance/`.

#### 5.2.2 Cliente MCP

- **Responsabilidad**: transportar el movimiento y decodificar el resultado; enviar `X-P40LA-Source: mcp`.
- **Interfaz**: se amplía `HomeMaintenanceAPI` con `MoveHomeMaintenanceItems`.
- **Dependencias**: `internal/mcp/api/client.go` (`doJSON`).
- **Ubicación**: `internal/mcp/api/home_maintenance.go`.

#### 5.2.3 Tools MCP

- **Responsabilidad**: validar forma y fecha, resolver el destino, aplicar `dry_run`/compuerta y devolver un resultado accionable.
- **Interfaz**: `home_maintenance_create`, `home_maintenance_item_move`.
- **Dependencias**: `api.HomeMaintenanceAPI`, `config.Config`, `toolkit`.
- **Ubicación**: `internal/mcp/tools/homemaint/create.go`, `move.go`.

### 5.3 Modelo de datos

No cambia el esquema.

```
home_maintenances
- id, home_id, performed_at (TEXT ISO), kind, area
- institution_id (nullable), description, created_at, updated_at
- Índice: (home_id, performed_at DESC, id DESC)

home_maintenance_items
- id, maintenance_id, item_type (labor|supplies), description
- quantity (>0), unit (enum), amount (>=0), currency_id
- institution_id (nullable), created_at, updated_at
- FK maintenance_id → home_maintenances(id) ON DELETE CASCADE
- Índice: (maintenance_id, item_type, id)
```

El movimiento solo cambia `home_maintenance_items.maintenance_id` y `updated_at`.

### 5.4 APIs / Contratos

#### Endpoint: `POST /api/homes/{id}/maintenances/{maintenance_id}/items/move`

**Request** (exactamente uno de `target_maintenance_id` o `target`). Destino existente:

```json
{
  "item_ids": [101, 102],
  "target_maintenance_id": 11,
  "delete_source_if_empty": false
}
```

Destino nuevo, creado en la misma transacción:

```json
{
  "item_ids": [101, 102],
  "target": {
    "performed_at": "2026-03-11",
    "kind": "corrective",
    "area": "plumbing",
    "institution_id": 7,
    "description": "Reparación demo destino"
  },
  "delete_source_if_empty": false
}
```

| Campo | Tipo | Regla |
|---|---|---|
| `item_ids` | array de enteros | 1..100, positivos, sin duplicados |
| `item_type` | string | Opcional: `labor` o `supplies`; si viene, todas las líneas deben ser de ese tipo |
| `target_maintenance_id` | entero | Excluyente con `target`; debe existir, pertenecer al `home_id` y ser distinto del origen |
| `target` | objeto | Excluyente con `target_maintenance_id`; usa las mismas validaciones del alta (`performed_at` obligatoria, no futura) |
| `delete_source_if_empty` | boolean | Default `false`; borra el origen solo si queda con 0 líneas |

**Response 200**:

```json
{
  "source_maintenance_id": 10,
  "target_maintenance_id": 11,
  "target_created": false,
  "target": {
    "id": 11,
    "home_id": 3,
    "performed_at": "2026-03-11",
    "kind": "corrective",
    "area": "plumbing",
    "description": "Reparación demo destino",
    "items": [],
    "totals_by_currency": []
  },
  "moved_count": 2,
  "moved_items": [
    {
      "id": 101,
      "maintenance_id": 11,
      "item_type": "labor",
      "description": "Mano de obra demo",
      "quantity": 1,
      "unit": "unit",
      "amount": 500.00,
      "currency_id": 1,
      "currency_code": "NIO"
    },
    {
      "id": 102,
      "maintenance_id": 11,
      "item_type": "supplies",
      "description": "Insumos demo",
      "quantity": 2,
      "unit": "unit",
      "amount": 300.00,
      "currency_id": 1,
      "currency_code": "NIO"
    }
  ],
  "source_item_count": 0,
  "source_deleted": false
}
```

**Errores**: `invalid_request`, `invalid_item_ids`, `invalid_item_type`, `item_type_mismatch`, `target_required`, `target_conflict`, `same_maintenance`, `item_not_in_source`, `invalid_date`, `future_date`, `invalid_kind`, `invalid_area` (400); `home_not_found`, `maintenance_not_found`, `item_not_found`, `currency_not_found`, `institution_not_found` (404); `move_conflict` (409).

#### Tool `home_maintenance_create`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda destino |
| `performed_at` | string | Sí | `YYYY-MM-DD`, no futura |
| `kind` | string | No | `corrective` o `preventive`; default `corrective` |
| `area` | string | No | `plumbing`, `electrical`, `roof`, `paint`, `carpentry`, `appliances`, `garden`, `cleaning`, `other`; default `other` |
| `institution_id` | integer | No | Proveedor o contratista |
| `description` | string | No | Descripción libre; máximo 500 |
| `dry_run` | boolean | No | Default `true` |

**Salida con escritura**:

```json
{
  "status": "created",
  "dry_run": false,
  "item_count": 0,
  "maintenance": {
    "id": 12,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "preventive",
    "area": "roof",
    "description": "Revisión demo de techo",
    "items": [],
    "totals_by_currency": []
  }
}
```

La reparación vacía **sí aparece** en `home_maintenance_list` (la consulta no hace JOIN con las líneas), con `totals_by_currency=[]`. El filtro `item_type` no la devuelve porque exige al menos una línea de ese tipo.

#### Tool `home_maintenance_item_move`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda de origen y destino |
| `source_maintenance_id` | integer | Sí | Reparación que contiene las líneas |
| `item_ids` | array de enteros | Sí | 1..100 líneas de la reparación origen |
| `item_type` | string | No | `labor` o `supplies`: salvaguarda de tipo |
| `target_maintenance_id` | integer | Condicional | Destino existente; excluyente con `target` |
| `target` | objeto | Condicional | Destino nuevo (`performed_at`, `kind`, `area`, `institution_id`, `description`); excluyente con `target_maintenance_id` |
| `delete_source_if_empty` | boolean | No | Default `false` |
| `dry_run` | boolean | No | Default `true` |

**Salida con `dry_run`**:

```json
{
  "status": "planned",
  "dry_run": true,
  "source": {"maintenance_id": 10, "performed_at": "2026-03-10", "area": "plumbing", "description": "Reparación demo origen"},
  "target": {"action": "create_maintenance", "performed_at": "2026-03-11", "area": "plumbing", "description": "Reparación demo destino"},
  "moved_items": [{"id": 101, "item_type": "labor", "description": "Mano de obra demo", "amount": 500.00, "currency_id": 1}],
  "remaining_items": [{"id": 103, "item_type": "supplies", "description": "Insumo demo", "amount": 100.00, "currency_id": 1}],
  "source_item_count_after": 1,
  "would_delete_source": false,
  "warnings": []
}
```

**Salida con escritura**:

```json
{
  "status": "moved",
  "dry_run": false,
  "source_maintenance_id": 10,
  "target_maintenance_id": 11,
  "target_created": true,
  "moved_count": 1,
  "moved_items": [{"id": 101, "maintenance_id": 11, "item_type": "labor", "description": "Mano de obra demo", "amount": 500.00, "currency_id": 1}],
  "source_item_count": 1,
  "source_deleted": false
}
```

### 5.5 Dependencias

- **Internas**: `internal/modules/home/maintenance`, `internal/mcp/api`, `internal/mcp/tools/homemaint`, `internal/mcp/tools/all`.
- **Externas**: ninguna nueva.

### 5.6 Archivos previstos

| Archivo | Cambio |
|---|---|
| `internal/modules/home/maintenance/model.go` | `MoveInput`, `MoveResult`, `ErrMoveConflict`, `ModuleVersion` 1.0.0 → 1.1.0 |
| `internal/modules/home/maintenance/service.go` | `MoveItems`, validación de destino/líneas, `move_conflict` |
| `internal/modules/home/maintenance/storage.go` | `MoveItems` transaccional, `ListItemsByIDs`, conteo del origen |
| `internal/modules/home/maintenance/handlers.go` | ruta `POST .../items/move` y mapeo de errores |
| `internal/modules/home/maintenance/service_test.go` | movimiento, destino nuevo, rollback, borrado opt-in |
| `internal/modules/home/maintenance/handlers_test.go` | httptest del endpoint nuevo y errores |
| `internal/mcp/api/home_maintenance.go` | `MoveHomeMaintenanceItems`, tipos de request/result |
| `internal/mcp/tools/homemaint/homemaint.go` | registro de `home_maintenance_create` y `home_maintenance_item_move` |
| `internal/mcp/tools/homemaint/create.go` | nuevo; `home_maintenance_create` |
| `internal/mcp/tools/homemaint/move.go` | nuevo; `home_maintenance_item_move` |
| `internal/mcp/tools/homemaint/homemaint_test.go` | create/move, dry-run, errores |
| `internal/mcp/api/home_maintenance_test.go` | encode/decode del movimiento y `X-P40LA-Source` |
| `internal/mcp/tools/all/contract_test.go` | stub `MoveHomeMaintenanceItems` |
| `internal/api/mcp_backend_contract_test.go` | alta vacía + movimiento por la cadena real |
| `docs/mcp/tools/impact.json` | 2 tools nuevas y dependencia `internal/mcp/tools/homemaint/` para `home_maintenance_list` |
| `docs/mcp/tools/contracts.json` | golden regenerado (42 → 44) |
| `docs/mcp/tools/home-maintenance.md` | secciones de create/move |
| `docs/mcp/tools/README.md` | catálogo 42 → 44 |
| `docs/mcp/README.md` | conteo y tabla de escritura |
| `docs/mcp/security.md` | tools sujetas a la compuerta |
| `docs/mcp/tools/impact.md` | conteos 42 → 44 |
| `docs/specs/README.md` | tracker en el release |

**Sin migraciones**: el esquema de `0057_home_maintenance` no cambia.

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). Análisis con `./scripts/check-mcp-impact.sh --files ...` y validación de un subagente (`DELIV-MCP-IMPACT-SUBAGENT`).

### 6.1 Tools nuevas

| Tool | Entrypoint | Endpoints | Contrato |
|---|---|---|---|
| `home_maintenance_create` | `internal/mcp/tools/homemaint/create.go` | `POST /api/homes/{id}/maintenances` | Nueva en `contracts.json`; sin `item`, `dry_run` |
| `home_maintenance_item_move` | `internal/mcp/tools/homemaint/move.go` | `GET /api/homes/{id}/maintenances/{maintenance_id}`, `POST /api/homes/{id}/maintenances`, `POST /api/homes/{id}/maintenances/{maintenance_id}/items/move` | Nueva en `contracts.json`; `item_ids`, `target_maintenance_id`/`target`, `delete_source_if_empty`, `dry_run` |

### 6.2 Tools existentes afectadas funcionalmente

| Tool | Ruta de dependencia | Motivo | Contrato |
|---|---|---|---|
| `home_maintenance_list` | tool → `internal/mcp/api/home_maintenance.go` → `GET /api/homes/{id}/maintenances` → `internal/modules/home/maintenance/handlers.go` → `service.go` → `storage.go` → `home_maintenances` + `home_maintenance_items` | Sin cambio funcional; se recompila el paquete compartido y el módulo gana el endpoint de movimiento | Sin cambios |
| `home_maintenance_labor_register` | tool → `internal/mcp/api/home_maintenance.go` → `POST /api/homes/{id}/maintenances` / `.../items` → handler → service → storage → `home_maintenances` + `home_maintenance_items` | Sin cambio funcional; el alta vacía reutiliza el mismo endpoint | Sin cambios |
| `home_maintenance_supplies_register` | idem | Sin cambio funcional | Sin cambios |

### 6.3 Ruta compartida `internal/mcp/api/` y `internal/mcp/tools/all/`

El manifiesto declara ambos directorios en `shared.affects_all`. Agregar `MoveHomeMaintenanceItems` a `HomeMaintenanceAPI` y el stub en `contract_test.go` hace que el analizador reporte **las 42 tools existentes** por recompilación, aunque su comportamiento no cambia. El gate exige declararlas; se listan a continuación para que quede explícito:

`alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bank_accounts_create`, `bank_accounts_search`, `bank_accounts_update`, `banks_create`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `credit_card_statements_get`, `credit_card_statements_list`, `credit_cards_create`, `credit_cards_search`, `credit_cards_update`, `debt_bills_list`, `entities_search`, `home_maintenance_labor_register`, `home_maintenance_list`, `home_maintenance_supplies_register`, `homes_update`, `homes_create`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

**Impacto real**: nulo en el comportamiento de esas 42 tools; solo se recompila el paquete `internal/mcp/api` y el catálogo de tests. Las tools con cambio funcional son las dos nuevas.

### 6.4 Contrato y tests

- `docs/mcp/tools/contracts.json`: se regenera con `go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts`; se revisa el diff.
- `docs/mcp/tools/impact.json`: se agregan las dos tools nuevas con sus dependencias y endpoints.
- `internal/mcp/tools/all/contract_test.go`: `contractAPI` debe implementar `MoveHomeMaintenanceItems` o el paquete de tests no compila.
- `internal/api/mcp_backend_contract_test.go`: agregar el alta de una reparación **sin líneas** y el movimiento de una línea `labor` y otra `supplies` a un destino creado en la misma llamada; verificar `moved_count`, `source_item_count` y los tags JSON.
- `docs/mcp/README.md`, `docs/mcp/security.md` y `docs/mcp/tools/README.md`: actualizar el catálogo 42 → 44 (24 lectura + 20 escritura) y la lista de tools sujetas a `P40LA_MCP_ALLOW_WRITES`.

Tests:
- `internal/modules/home/maintenance/service_test.go` (movimiento, destino nuevo, rollback, `item_not_in_source`, `same_maintenance`, borrado opt-in).
- `internal/modules/home/maintenance/handlers_test.go` (endpoint, errores 400/404/409).
- `internal/mcp/tools/homemaint/homemaint_test.go` (create: dry-run/escritura/compuerta; move: dry-run/escritura/destino nuevo/tipo incorrecto/línea ajena).
- `internal/mcp/api/home_maintenance_test.go` (query/body y decode del movimiento).
- `internal/mcp/tools/all/contract_test.go` (stub + golden).
- `go test ./internal/mcp/...`
- `go test ./internal/api -run TestMCPBackendContract`
- `./scripts/check-mcp-impact.sh --staged --gate`
- `./scripts/check-mcp-docs.sh --staged`

**Validación subagente:** explore — sesión `ses_def14145effeMb6W52YuK1kjYz` — veredicto: **viable con cambios**; 2 tools nuevas y catálogo 42 → 44; las 42 existentes se declaran por `shared.affects_all` (`internal/mcp/api/`) sin cambio funcional ni de contrato; el backend ya permitía crear la reparación sin líneas, así que `home_maintenance_create` reutiliza el endpoint; requiere `impact.json` + `contracts.json`, stub `MoveHomeMaintenanceItems` en `contract_test.go` y `fakeAPI`, montar create/move en `TestMCPBackendContract` y validar `item_type` también en backend; el ruteo `POST .../items/move` no choca con `PUT/DELETE .../items/{item_id}`.

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [x] `CA-REQ-001`: `home_maintenance_create` con `dry_run=false` crea una reparación con `items=[]` y `item_count=0`; `home_maintenance_list` la devuelve sin totales.
- [x] `CA-REQ-002`: `home_maintenance_item_move` mueve una línea `labor` y una `supplies` de la reparación A a la B; los `maintenance_id` cambian y los importes se conservan.
- [x] `CA-REQ-003`: con `target` (sin `target_maintenance_id`) la tool crea el destino y mueve las líneas en una sola llamada; `target_created=true`.
- [x] `CA-REQ-003B`: si el `UPDATE` no afecta todas las líneas, la transacción revierte y no queda un destino vacío.
- [x] `CA-REQ-004`: una línea de otra reparación del mismo hogar devuelve `item_not_in_source` y no mueve ninguna.
- [x] `CA-REQ-005`: `target_maintenance_id` igual al origen devuelve `same_maintenance` sin escribir.
- [x] `CA-REQ-006`: `dry_run` default `true` devuelve `status=planned`, `moved_items` y `remaining_items` sin escribir.
- [x] `CA-REQ-007`: sin `P40LA_MCP_ALLOW_WRITES`, `dry_run=false` devuelve `writes_disabled`.
- [x] `CA-REQ-008`: `delete_source_if_empty=false` conserva el origen vacío (`source_deleted=false`); con `true` y origen vacío lo borra (`source_deleted=true`).
- [x] `CA-REQ-009`: con `item_type=labor`, una lista que incluye un insumo devuelve `invalid_request` sin escribir.
- [x] `CA-REQ-010`: `impact.json`, `contracts.json`, `home-maintenance.md` y los conteos de `docs/mcp/` actualizados.
- [x] `CA-REQ-011`: `contractAPI` de `internal/mcp/tools/all/contract_test.go` implementa `MoveHomeMaintenanceItems` y el paquete de tests compila.
- [x] `CA-REQ-012`: `TestMCPBackendContract` cubre el alta vacía y el movimiento por la cadena real.
- [x] `CA-UX-PREREQ`: el backend valida `home_id`, origen, destino, moneda e institución antes de escribir.
- [x] `CA-SEC-SYNTHETIC`: todos los ejemplos y fixtures usan datos `demo-*`.

### 7.2 No funcionales

- [x] `CA-ARCH-IHOST`: sin dependencias nuevas; `go build ./...` y `go test ./...` pasan.
- [x] `CA-TEST-MCP-CONTRACT`: `TestToolContracts` y `TestMCPBackendContract` verdes; golden revisado.
- [x] `CA-DELIV-MCP-IMPACT`: `./scripts/check-mcp-impact.sh --staged --gate` verde antes del commit.
- [x] `CA-DELIV-WORKTREE`: `./scripts/check-worktree.sh` verde en `../p40la-ihost-spec-157`.
- [ ] `CA-TEST-USER-FLOW`: el usuario confirma las pruebas manuales antes de `pending_release`.

### 7.3 Testing

- **Unit tests**: validación de `item_ids`, pertenencia al origen, `target_required`/`target_conflict`, `same_maintenance`, `item_type`, borrado opt-in y topes.
- **Integration tests**: `httptest` del módulo para el endpoint de movimiento y el alta vacía; rollback cuando una línea no pertenece al origen.
- **MCP tool tests**: create y move con `dry_run`, escritura, compuerta, destino nuevo y errores.
- **Contrato**: `TestToolContracts` con golden actualizado; `TestMCPBackendContract` sin regresiones.
- **Carga**: no aplica; máximo 100 líneas por movimiento.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Backend: `MoveInput`/`MoveResult`, `MoveItems`, endpoint, errores y tests | 0.5 día | Ninguna |
| 2 | Cliente MCP: `MoveHomeMaintenanceItems` y tipos | 0.25 día | Fase 1 |
| 3 | Tools: `home_maintenance_create`, `home_maintenance_item_move`, tests | 0.5 día | Fase 2 |
| 4 | Manifiesto, contrato, docs, README de tools | 0.25 día | Fase 3 |
| 5 | Gate completo y validación del usuario | 0.25 día | Fase 4 |

### 8.2 Milestones

1. **MVP**: backend + cliente + dos tools con dry-run; tests verdes.
2. **V1.0**: contrato, documentación, manifiesto y validación del usuario.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| El agente mueve líneas a la reparación equivocada | Media | Alto | Destino explícito, sin resolución automática; `dry_run` default y preview de líneas restantes |
| El origen queda vacío y el usuario no lo nota | Media | Bajo | `source_item_count` en la respuesta y warning; borrado solo con `delete_source_if_empty=true` |
| Dos movimientos concurrentes sobre las mismas líneas | Baja | Medio | `UPDATE ... WHERE maintenance_id = origen` + `RowsAffected`; `move_conflict` con rollback |
| La respuesta crece demasiado en el iHost | Baja | Medio | Máximo 100 líneas y cliente corta en 1 MiB |
| El cambio en `internal/mcp/api/` se interpreta como impacto en las 42 tools | Alta (analizador) | Bajo | Declarado en 6.3; sin cambio de comportamiento; tests MCP completos |
| Regenerar el golden sin revisar el diff | Baja | Medio | Diff revisado y declarado en la spec; `TestToolContracts` sin `-update` |
| El endpoint nuevo choca con `PUT/DELETE .../items/{item_id}` | Baja | Medio | `POST` + literal `move` es más específico que `{item_id}`; test de ruteo |

---

## 10. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-141 (módulo `home.maintenance`), SPEC-142 (proveedor por línea), SPEC-145 (MCP y compuerta de escritura), SPEC-153 (guardrail de impacto), SPEC-154 (catálogo de tools), SPEC-155 (registro de mano de obra e insumos), SPEC-156 (edición de hogares, integrada en el release).
- Documentación: `docs/mcp/tools/home-maintenance.md`, `docs/mcp/tools/impact.md`.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial de la especificación; issue #160 |
| 2026-11-24 | opencode | Implementación: endpoint transaccional `POST .../items/move`, `MoveInput`/`MoveResult`, tools `home_maintenance_create` y `home_maintenance_item_move`, validación de `item_type` en backend, manifiesto 41 → 43 (44 al integrar SPEC-156), contrato regenerado y tests. `go build ./...`, `go test ./...` y `TestMCPBackendContract` verdes |
| 2026-11-24 | opencode | Documentación MCP actualizada (catálogo 44 tools al integrar SPEC-156, secciones de create/move y compuerta); `check-mcp-docs.sh --staged` y `check-mcp-impact.sh --staged --gate` verdes. Pendiente la validación manual del usuario |
