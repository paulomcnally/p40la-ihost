---
title: "MCP: registro de mano de obra e insumos en reparaciones del hogar"
id: "SPEC-155"
status: "pending_release"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 158
---

# MCP: registro de mano de obra e insumos en reparaciones del hogar

**ID**: SPEC-155<br>
**Estado**: pending_release<br>
**Autor**: opencode<br>
**Creado**: 2026-11-24<br>
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

El módulo `home.maintenance` ya registra reparaciones del hogar con líneas de **mano de obra** (`labor`) e **insumos** (`supplies`), cada una con su moneda, cantidad, unidad y proveedor (SPEC-141/SPEC-142). El servidor MCP solo expone `home_maintenance_list` en modo lectura (SPEC-154). Hoy un agente no puede registrar una línea: debe pedirle al usuario que lo haga desde la web.

Esta spec agrega **dos tools MCP de escritura**: `home_maintenance_labor_register` y `home_maintenance_supplies_register`. Ambas reutilizan los endpoints existentes del módulo, respetan `dry_run` (default `true`) y la compuerta `P40LA_MCP_ALLOW_WRITES`, y **buscan los registros del mismo día antes de escribir**. Si encuentran una línea equivalente, devuelven `status=duplicate` con el registro existente y no crean nada, salvo que el llamador use `allow_duplicate=true`. Para que el agente pueda **ver lo existente**, `home_maintenance_list` gana filtros opcionales por fecha exacta, tipo de línea, texto y líneas incluidas.

El cambio no agrega dependencias, migraciones ni rutas nuevas. El impacto en el iHost es mínimo: una consulta por día antes de escribir, un `BEGIN/COMMIT` corto cuando se crea mantenimiento + línea, y respuestas acotadas. No toca frontend ni datos de producción.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec toca backend, MCP y SQLite; **no toca frontend, rutas de UI, i18n ni migraciones**. Las reglas de UI/UX se declaran `N/A` con justificación.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go stdlib + SQLite; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | El cliente MCP vive en `internal/mcp/api/`; las tools en `internal/mcp/tools/homemaint/`; el dominio sigue en `internal/modules/home/maintenance/` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Tool MCP → cliente HTTP → handler → service → storage. **Prohibido** SQL desde el MCP; el módulo conserva `handlers.go`, `service.go`, `storage.go` | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | `home.maintenance` mantiene su namespace y no importa storage de otros módulos; la creación transaccional usa `database/sql` dentro de su propio storage | `CA-ARCH-MODULES` |
| `ARCH-MCP-TOOLS-MAP` | Las dos tools nuevas se registran con nombre literal y entrada en `docs/mcp/tools/impact.json`; el contrato se regenera de forma intencional | `CA-ARCH-MCP-TOOLS-MAP` |
| `ARCH-IHOST` | Sin caché ni estado en el MCP; consultas por día, `LIMIT` y tope de líneas; `BEGIN/COMMIT` corto; respuesta ≤ 1 MiB en el cliente | `CA-ARCH-IHOST` |
| `ARCH-GO` | `gofmt`, nombres en inglés, errores explícitos, `log/slog` sin cambios | `CA-ARCH-GO` |
| `DATA-SQLITE` | Se reutilizan `home_maintenances` y `home_maintenance_items`; SQL crudo con `database/sql` y transacción para el alta combinada | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | **N/A**: no hay cambios de esquema; las tablas e índices de `0057_home_maintenance` alcanzan | — |
| `DATA-PROTECTED-SCHEMA` | **N/A**: no se toca `debts` / `debt_bills` | — |
| `DATA-NO-DROP` | **N/A**: no se ejecuta nada sobre `data/app.db` | — |
| `DATA-PRODCOPY` | **N/A**: la validación usa DB temporal y fixtures sintéticos; no depende de datos reales | — |
| `DATA-PROD-WRITE` | **N/A**: no se escribe en producción | — |
| `SEC-NO-PII` | Las tools devuelven descripciones libres y montos del propio usuario; no agregan emails, teléfonos ni identificadores. Los ejemplos son sintéticos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Casa Demo`, `Proveedor Demo`, `demo-account-001`, montos ficticios y fechas `2026-03-10` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Los logs del MCP no agregan argumentos ni montos; la auditoría existente no cambia | `CA-SEC-LOGS` |
| `SEC-COMMIT` | `git diff --cached` se revisa; sin DBs, PDFs, exports ni secretos | `CA-SEC-COMMIT` |
| `UI-*` (todas) | **N/A**: la spec no toca frontend ni i18n; `home_maintenance_list` ya está documentada en la web | — |
| `UX-BACK` | **N/A**: no se agregan rutas ni pantallas | — |
| `UX-MODAL` / `UX-MODAL-CONFIRM` | **N/A**: no se agregan modales | — |
| `UX-DETAIL-ACTIONS` / `UX-DETAIL-READONLY` | **N/A**: no se modifica ninguna página de detalle | — |
| `UX-PREREQ` | Aplica al backend MCP: `home_id`, `currency_id` e `institution_id` se validan antes de escribir; el agente debe resolver el `home_id` con `entities_search` | `CA-UX-PREREQ` |
| `UX-EMPTY` / `UX-LOADING` / `UX-FEEDBACK` / `UX-FORMS` | **N/A**: no hay UI; el feedback del MCP se expresa con `status`, `warnings` y errores estables | — |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` antes del commit; pruebas con DB temporal | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | Se actualiza `docs/mcp/tools/contracts.json` de forma intencional y se corren `TestToolContracts` + `TestMCPBackendContract` | `CA-TEST-MCP-CONTRACT` |
| `TEST-TESTUSER` | **N/A**: las pruebas del módulo usan httptest y DB temporal, no el server local con usuario | — |
| `DELIV-WORKTREE` | La spec vive en `../p40la-ihost-spec-155` sobre `feature/SPEC-155`; `./scripts/check-worktree.sh` pasa | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push se hacen con la autorización explícita del usuario («pasalo a released») y sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa; `./scripts/check-mcp-impact.sh --files ...` y `--staged --gate` antes del commit | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; veredicto citado en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-MCP-DOCS` | `docs/mcp/tools/home-maintenance.md` y `docs/mcp/tools/README.md` se actualizan en el mismo commit | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente en paralelo a la implementación | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-COMMIT` | Commits `SPEC-155: ...` | `CA-DELIV-COMMIT` |
| `DELIV-LABELS` | Issue con `spec/draft`, transiciones a `spec/in-progress`, `spec/pending-release` y `spec/released` | `CA-DELIV-LABELS` |
| `DELIV-RELEASE` | Merge, push, tracker, label, cierre de issue y limpieza del worktree en el mismo ciclo | `CA-DELIV-RELEASE` |
| `DELIV-DOCKER` | **N/A**: no se publica imagen ni cambia el Dockerfile | — |
| `DELIV-DEPLOY` | **N/A**: no se despliega al iHost en esta spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Exponer `home_maintenance_labor_register` para registrar una línea de mano de obra (`item_type=labor`) en una reparación del hogar.
2. **REQ-002**: Exponer `home_maintenance_supplies_register` para registrar una línea de insumos (`item_type=supplies`) con cantidad y unidad.
3. **REQ-003**: Ambas tools deben resolver el mantenimiento destino:
   - si reciben `maintenance_id`, validan que pertenezca al `home_id`;
   - si no lo reciben, buscan los mantenimientos del mismo `home_id` y `performed_at`;
   - si hay exactamente uno compatible con `area`, agregan la línea;
   - si no hay ninguno, crean el mantenimiento y la línea en una sola transacción;
   - si hay más de uno, devuelven `status=ambiguous_maintenance` con los candidatos y no escriben.
4. **REQ-004**: Antes de escribir, buscar líneas existentes del **mismo día** y comparar tipo, descripción normalizada, monto, moneda, proveedor y (para insumos) cantidad y unidad. Si hay una coincidencia, devolver `status=duplicate` con el registro existente y no escribir. `allow_duplicate=true` permite crear el duplicado de forma explícita y devuelve `warnings`.
5. **REQ-005**: `dry_run` viene activado por defecto. En modo preview la tool devuelve el plan y `existing_today` (hasta 50 líneas del día) para que el agente vea lo existente antes de confirmar.
6. **REQ-006**: Ampliar `home_maintenance_list` con los filtros opcionales `date` (YYYY-MM-DD), `item_type` (`labor`/`supplies`), `query` (texto en descripción del mantenimiento o de sus líneas) e `include_items` (bool). `date` y `year` son mutuamente excluyentes.
7. **REQ-007**: Respetar la compuerta de escritura existente: `P40LA_MCP_ALLOW_WRITES=true` (y `P40LA_MCP_REMOTE_WRITES=true` en HTTP). Sin la compuerta, `dry_run=false` devuelve `writes_disabled`.
8. **REQ-008**: Validar `home_id`, `maintenance_id`, `currency_id` e `institution_id` antes de escribir; devolver errores estables (`home_not_found`, `maintenance_not_found`, `currency_not_found`, `institution_not_found`, `invalid_date`, `invalid_request`).
9. **REQ-009**: Marcar las escrituras con `X-P40LA-Source: mcp` mediante el cliente existente.
10. **REQ-010**: Actualizar manifiesto, contrato congelado, stubs de `testMCPBackendContract` y documentación MCP en el mismo commit.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-011**: Limitar la búsqueda del día a un máximo de 100 mantenimientos y 1.000 líneas; si se supera, devolver `truncated=true` sin fallar.
2. **REQ-012**: Escapar `%`, `_` y `\` en `query` para que el texto no se interprete como patrón SQL.
3. **REQ-013**: Devolver `reused_maintenance=true` cuando la línea se agregó a un mantenimiento existente, para que el agente pueda explicarlo.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-014**: Evaluar en una spec futura una clave de idempotencia persistida (`dedup_key` + índice único) para eliminar la carrera entre dos escrituras concurrentes. En esta spec la deduplicación es de aplicación, igual que el control de duplicados por período de `bills_create`.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: la consulta del día usa el índice `idx_home_maintenances_home_date`; el alta combinada es un `BEGIN/COMMIT` de dos inserts. Sin caché ni estado.
- **Memoria**: máximo 100 mantenimientos y 1.000 líneas por respuesta de búsqueda; el cliente MCP ya corta en 1 MiB.
- **Seguridad**: los endpoints siguen autenticados por API key/sesión; las tools no agregan secretos ni PII.
- **Almacenamiento**: no hay migraciones ni archivos nuevos de datos.
- **Disponibilidad**: si la creación de la línea falla dentro de la transacción, no queda un mantenimiento vacío.
- **iHost**: sin dependencias nuevas; una consulta extra por escritura; sin procesos en background.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `internal/modules/home/maintenance/model.go` ya define `ItemLabor`, `ItemSupplies`, unidades, `ItemInput` y `CurrencyTotal`.
- `POST /api/homes/{id}/maintenances` crea el mantenimiento; `POST /api/homes/{id}/maintenances/{mid}/items` crea la línea. Hoy son dos llamadas no transaccionales.
- `home_maintenance_list` (`internal/mcp/tools/homemaint/homemaint.go`) solo reutiliza `GET /api/homes/{id}/maintenances?year=&kind=` y no devuelve las líneas.
- `autos_maintenance_register` es el precedente de registro con `dry_run`, compuerta de escritura y estado parcial; en autos la moneda es del mantenimiento, en el hogar es **por línea**.
- `bills_create` valida duplicados por período a nivel de aplicación; no hay índice único de negocio.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Dos tools que solo agregan líneas a un `maintenance_id` existente | Simple, una sola escritura | No hay tool para crear el mantenimiento; el agente queda bloqueado si no existe | ❌ Rechazada |
| B. Dos tools que siempre crean un mantenimiento nuevo | Una sola llamada por tool | Mano de obra e insumos de la misma reparación quedan en dos mantenimientos; duplica registros | ❌ Rechazada |
| C. Dos tools que resuelven el mantenimiento del día, deduplican y crean/agregan | Reutiliza la estructura existente; evita mantenimientos duplicados; el agente ve lo existente | Requiere consulta previa y una transacción para el alta combinada | ✅ Seleccionada |
| D. Una sola tool `home_maintenance_register` con N líneas | Más compacta | El usuario pidió dos tools separadas por tipo; no permite resolver una línea sin la otra | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: `CreateInput` separado y alta transaccional**
- **Contexto**: crear mantenimiento y línea en dos llamadas deja un mantenimiento vacío si la segunda falla. `Input` se comparte entre POST y PUT; agregarle `item` haría que el PUT lo decodifique y lo ignore.
- **Decisión**: mantener `Input` intacto; el handler de POST decodifica `CreateInput{ Input; Item *ItemInput }` y llama a `Service.CreateWithItem`, que valida ambas partes y usa `Storage.CreateWithItem` dentro de una transacción. El PUT sigue usando `Input`; si llega `item`, se ignora y se documenta que no aplica a edición.
- **Consecuencias**: una sola llamada atómica para el alta; la web y los clientes actuales no cambian. No hay migración.

**ADR-002: deduplicación de aplicación, no índice único**
- **Contexto**: la clave incluye descripción libre normalizada, monto, moneda, proveedor y, en insumos, cantidad/unidad. Un índice único sobre texto libre es frágil y requiere migración.
- **Decisión**: la tool busca las líneas del día y compara antes de escribir; `allow_duplicate=true` permite el alta explícita. La validación no es atómica frente a dos escrituras concurrentes.
- **Consecuencias**: se evita el duplicado accidental típico sin cambiar el esquema; la carrera queda documentada como riesgo y se mitiga con revisión vía `home_maintenance_list?date=`.

**ADR-003: `home_maintenance_list` como superficie de consulta**
- **Contexto**: el usuario pidió «buscar o ver las existentes» para no duplicar.
- **Decisión**: ampliar la tool de lectura existente con `date`, `item_type`, `query` e `include_items` en lugar de crear una tercera tool.
- **Consecuencias**: el catálogo crece solo con las dos tools de escritura; el contrato de `home_maintenance_list` cambia de forma intencional y se regenera.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
Agente MCP
   |  home_maintenance_labor_register / supplies_register
   v
internal/mcp/tools/homemaint  --(cliente HomeMaintenanceAPI)-->  API HTTP /api/homes/{id}/maintenances
   |                                                                      |
   |  GET ?date=&include_items=1 (dedup)                                 v
   |                                                        handlers.go (módulo home.maintenance)
   |                                                                      |
   +--------------------------------------------------------------> service.go
                                                                          |
                                                                     storage.go
                                                                          |
                                                              SQLite: home_maintenances
                                                                      home_maintenance_items
```

### 5.2 Componentes

#### 5.2.1 Módulo `home.maintenance` (backend)

- **Responsabilidad**: validar y persistir mantenimientos y líneas; resolver filtros de consulta; crear mantenimiento + línea en una transacción.
- **Interfaz**:
  - `GET /api/homes/{id}/maintenances?year=&kind=&date=&item_type=&q=&include_items=1`
  - `POST /api/homes/{id}/maintenances` con `item` opcional
  - `GET /api/homes/{id}/maintenances/{mid}`
  - `POST /api/homes/{id}/maintenances/{mid}/items`
- **Dependencias**: `database/sql`, tablas `homes`, `currencies`, `institutions`.
- **Ubicación**: `internal/modules/home/maintenance/`.

#### 5.2.2 Cliente MCP

- **Responsabilidad**: transportar y decodificar las llamadas del módulo; construir query strings con `net/url`; enviar `X-P40LA-Source: mcp`.
- **Interfaz**: `HomeMaintenanceAPI` con `ListHomeMaintenances`, `GetHomeMaintenance`, `CreateHomeMaintenance`, `CreateHomeMaintenanceItem`.
- **Dependencias**: `internal/mcp/api/client.go` (`doJSON`, `ReadJSON`).
- **Ubicación**: `internal/mcp/api/home_maintenance.go`.

#### 5.2.3 Tools MCP

- **Responsabilidad**: validar entrada, resolver fecha y mantenimiento, deduplicar, aplicar `dry_run`/compuerta y devolver un resultado accionable.
- **Interfaz**: `home_maintenance_labor_register`, `home_maintenance_supplies_register`, `home_maintenance_list`.
- **Dependencias**: `api.HomeMaintenanceAPI`, `config.Config`, `toolkit`.
- **Ubicación**: `internal/mcp/tools/homemaint/`.

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
- Índice: (maintenance_id, item_type, id)
```

### 5.4 APIs / Contratos

#### Endpoint: `GET /api/homes/{id}/maintenances`

**Query params nuevos**:

| Param | Tipo | Regla |
|---|---|---|
| `date` | string `YYYY-MM-DD` | Fecha exacta; mutuamente excluyente con `year` |
| `item_type` | string | `labor` o `supplies`; filtra mantenimientos con al menos una línea de ese tipo |
| `q` | string | Texto en `m.description` o en `home_maintenance_items.description`; máximo 100 caracteres; `%`, `_`, `\` escapados |
| `include_items` | bool (`1`/`true`) | Carga las líneas de cada mantenimiento; tope 100 mantenimientos y 1.000 líneas |

**Response 200** (compatible hacia atrás; `truncated` es nuevo y opcional):

```json
{
  "items": [
    {
      "id": 10,
      "home_id": 3,
      "performed_at": "2026-03-10",
      "kind": "corrective",
      "area": "plumbing",
      "institution_id": 7,
      "institution_name": "Proveedor Demo",
      "description": "Reparación demo de fuga",
      "totals_by_currency": [
        {"currency_id": 1, "currency_code": "NIO", "labor_cost": 500, "supplies_cost": 1200, "total": 1700}
      ],
      "items": [
        {
          "id": 101,
          "maintenance_id": 10,
          "item_type": "labor",
          "description": "Mano de obra demo",
          "quantity": 1,
          "unit": "unit",
          "amount": 500,
          "currency_id": 1,
          "currency_code": "NIO",
          "institution_id": 7,
          "institution_name": "Proveedor Demo"
        }
      ]
    }
  ],
  "totals_by_currency": [],
  "truncated": false
}
```

**Errores**: `invalid_year`, `invalid_date`, `invalid_item_type`, `invalid_filter` (400); `home_not_found` (404).

#### Endpoint: `POST /api/homes/{id}/maintenances`

**Request** (`CreateInput`; el campo `item` es nuevo y opcional; si viene, se crea en la misma transacción):

```json
{
  "performed_at": "2026-03-10",
  "kind": "corrective",
  "area": "plumbing",
  "institution_id": 7,
  "description": "Reparación demo de fuga",
  "item": {
    "item_type": "labor",
    "description": "Mano de obra demo",
    "amount": 500,
    "currency_id": 1,
    "institution_id": 7
  }
}
```

**Response 201**: el mantenimiento con `items` (incluye la línea creada).

**Errores**: los existentes del módulo; si la línea es inválida, **no** se crea el mantenimiento. El `PUT /api/homes/{id}/maintenances/{mid}` conserva `Input` y no acepta `item`; ese campo solo aplica al alta.

#### Tool `home_maintenance_labor_register`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda destino |
| `maintenance_id` | integer | No | Mantenimiento existente; si viene, se ignora `performed_at` salvo que no coincida |
| `performed_at` | string | Condicional | `YYYY-MM-DD`, no futura; obligatoria si no hay `maintenance_id` |
| `area` | string | No | `plumbing`, `electrical`, `roof`, `paint`, `carpentry`, `appliances`, `garden`, `cleaning`, `other`; default `other` |
| `description` | string | Sí | Descripción de la mano de obra; máximo 200 |
| `amount` | number | Sí | Monto ≥ 0 en la moneda indicada |
| `currency_id` | integer | Sí | Moneda de la línea |
| `institution_id` | integer | No | Proveedor/contratista |
| `dry_run` | boolean | No | Default `true` |
| `allow_duplicate` | boolean | No | Default `false` |

**Salida** (ejemplo de alta):

```json
{
  "status": "created",
  "dry_run": false,
  "reused_maintenance": false,
  "maintenance": {"id": 10, "home_id": 3, "performed_at": "2026-03-10", "area": "plumbing"},
  "item": {"id": 101, "maintenance_id": 10, "item_type": "labor", "description": "Mano de obra demo", "amount": 500, "currency_id": 1}
}
```

**Salida duplicada**:

```json
{
  "status": "duplicate",
  "dry_run": false,
  "existing": {
    "maintenance_id": 10,
    "item": {"id": 101, "item_type": "labor", "description": "Mano de obra demo", "amount": 500, "currency_id": 1}
  },
  "hint": "Ya existe una línea equivalente el mismo día. Usá allow_duplicate=true solo si es un registro distinto."
}
```

#### Tool `home_maintenance_supplies_register`

Igual que la anterior, más:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `quantity` | number | No | Default `1`; debe ser > 0 |
| `unit` | string | No | `unit`, `m`, `m2`, `kg`, `lb`, `l`, `gal`, `sack`, `box`; default `unit` |

#### Tool `home_maintenance_list` (ampliada)

Nuevos campos de entrada: `date`, `item_type`, `query`, `include_items`. La salida conserva `home_id`, `count`, `maintenances`, `totals_by_currency` y agrega `truncated`.

`item_type` filtra **qué mantenimientos aparecen** (los que tienen al menos una línea de ese tipo); `totals_by_currency` sigue sumando todas las líneas de esos mantenimientos, con el desglose `labor_cost` / `supplies_cost`. No se anula la columna del otro tipo para no perder el total de la reparación.

### 5.5 Dependencias

- **Internas**: `internal/modules/home/maintenance`, `internal/mcp/api`, `internal/mcp/tools/homemaint`, `internal/mcp/tools/all`.
- **Externas**: ninguna nueva.

### 5.6 Archivos previstos

| Archivo | Cambio |
|---|---|
| `internal/modules/home/maintenance/model.go` | `CreateInput`, `Filter` ampliado, `List.Truncated` |
| `internal/modules/home/maintenance/service.go` | `CreateWithItem`, validación de filtros, `include_items` y topes |
| `internal/modules/home/maintenance/storage.go` | `CreateWithItem` transaccional, filtros `date`/`item_type`/`q`, `ListItemsByMaintenanceIDs` |
| `internal/modules/home/maintenance/handlers.go` | parseo de `date`, `item_type`, `q`, `include_items`; POST con `item` |
| `internal/modules/home/maintenance/service_test.go` | filtros, transacción y rollback |
| `internal/modules/home/maintenance/handlers_test.go` | nuevo; httptest de query params y POST |
| `internal/mcp/api/home_maintenance.go` | nuevo; `HomeMaintenanceAPI`, tipos y llamadas |
| `internal/mcp/api/autos.go` | `FullAPI` embebe `HomeMaintenanceAPI` |
| `internal/mcp/tools/homemaint/homemaint.go` | `Toolset`, `Register`, `home_maintenance_list` ampliada |
| `internal/mcp/tools/homemaint/register.go` | nuevo; `writeMode`, fechas, resolución, dedup y helpers |
| `internal/mcp/tools/homemaint/labor.go` | nuevo; `home_maintenance_labor_register` |
| `internal/mcp/tools/homemaint/supplies.go` | nuevo; `home_maintenance_supplies_register` |
| `internal/mcp/tools/homemaint/homemaint_test.go` | casos de escritura, dedup y filtros |
| `internal/mcp/tools/all/contract_test.go` | stubs de `HomeMaintenanceAPI` |
| `internal/api/mcp_backend_contract_test.go` | módulo montado + cadena real |
| `docs/mcp/tools/impact.json` | 2 tools nuevas y dependencias de `model.go`/`module.go` |
| `docs/mcp/tools/contracts.json` | golden regenerado |
| `docs/mcp/tools/home-maintenance.md` | documentación de las 2 tools y filtros |
| `docs/mcp/tools/README.md` | catálogo 39 → 41 |
| `docs/mcp/README.md` | conteo y tabla de escritura |
| `docs/mcp/security.md` | tools sujetas a la compuerta |
| `docs/specs/README.md` | tracker en el release |

**Sin migraciones**: el esquema de `0057_home_maintenance` no cambia.

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). Análisis con `./scripts/check-mcp-impact.sh --files ...` y validación de un subagente (`DELIV-MCP-IMPACT-SUBAGENT`).

### 6.1 Tools nuevas

| Tool | Entrypoint | Endpoints | Contrato |
|---|---|---|---|
| `home_maintenance_labor_register` | `internal/mcp/tools/homemaint/labor.go` | `GET /api/homes/{id}/maintenances`, `POST /api/homes/{id}/maintenances`, `POST /api/homes/{id}/maintenances/{mid}/items` | Nueva en `contracts.json`; `dry_run`, `allow_duplicate`, `maintenance_id` |
| `home_maintenance_supplies_register` | `internal/mcp/tools/homemaint/supplies.go` | idem + `quantity`/`unit` | Nueva en `contracts.json`; idem |

### 6.2 Tools existentes afectadas funcionalmente

| Tool | Ruta de dependencia | Motivo | Contrato |
|---|---|---|---|
| `home_maintenance_list` | tool → `internal/mcp/api/home_maintenance.go` → `GET /api/homes/{id}/maintenances` → `internal/modules/home/maintenance/handlers.go` → `service.go` → `storage.go` → `home_maintenances` + `home_maintenance_items` | Nuevos filtros `date`, `item_type`, `query`, `include_items` y campo `truncated` | Cambia: se regenera el golden |
| `home_maintenance_labor_register` | tool → `GET/POST` → mismo handler → service → storage → `home_maintenances` + `home_maintenance_items`; tabla `currencies`, `institutions`, `homes` | Tool nueva | Nueva |
| `home_maintenance_supplies_register` | idem | Tool nueva | Nueva |

### 6.3 Ruta compartida `internal/mcp/api/`

El manifiesto declara `internal/mcp/api/` en `shared.affects_all`. Agregar `internal/mcp/api/home_maintenance.go` y ampliar `FullAPI` hace que el analizador reporte **las 39 tools existentes** por compilación del paquete compartido, aunque su comportamiento no cambia. El gate exige declararlas; se listan a continuación para que quede explícito:

`alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bank_accounts_create`, `bank_accounts_search`, `bank_accounts_update`, `banks_create`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `credit_card_statements_get`, `credit_card_statements_list`, `credit_cards_create`, `credit_cards_search`, `credit_cards_update`, `debt_bills_list`, `entities_search`, `home_maintenance_list`, `homes_create`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

**Impacto real**: nulo en el comportamiento de esas 39 tools; solo se recompila el paquete `internal/mcp/api`. Las tools con cambio funcional son `home_maintenance_list` y las dos nuevas.

### 6.4 Contrato y tests

- `docs/mcp/tools/contracts.json`: se regenera con `go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts`; se revisa el diff.
- `docs/mcp/tools/impact.json`: se agregan las dos tools nuevas. A `home_maintenance_list` se le suman `internal/modules/home/maintenance/model.go` y `module.go` a `dependencies`: hoy no están y el analizador produce un falso negativo si solo cambian esos archivos.
- `internal/mcp/tools/all/contract_test.go`: `contractAPI` implementa `api.FullAPI` método por método; al ampliar `FullAPI` con `HomeMaintenanceAPI` hay que agregar los cuatro stubs o el paquete de tests no compila.
- `internal/api/mcp_backend_contract_test.go`: hoy no monta módulos (`moduleRegistry` queda `nil`). Hay que registrar `maintenance.New(database)` y cubrir: `POST /maintenances` con `item` (tags `item_type`, `quantity`, `unit`, `currency_id`, `amount`), `GET ?date=&item_type=&q=&include_items=1` (sobre `{items, totals_by_currency}`), `GET /{mid}`, `POST /{mid}/items` y errores `currency_not_found` / `future_date`.
- `docs/mcp/README.md`, `docs/mcp/security.md` y `docs/mcp/tools/README.md`: actualizar el catálogo 39 → 41 (24 lectura + 17 escritura) y la lista de tools sujetas a `P40LA_MCP_ALLOW_WRITES`.
- `internal/mcp/tools/toolkitest/fake.go`: el `Fake` solo implementa `ReadAPI`. Las pruebas de `homemaint` usan un fake local que embebe `toolkitest.Fake` y agrega los métodos de escritura; no se cambia el helper compartido.

Tests:
- `internal/modules/home/maintenance/service_test.go` (filtros, alta transaccional, rollback, totales con `item_type`).
- `internal/modules/home/maintenance/handlers_test.go` (query params, POST con `item`, errores 400).
- `internal/mcp/tools/homemaint/homemaint_test.go` (dry-run, duplicado, `allow_duplicate`, ambiguo, append, create, list filters).
- `internal/mcp/api/home_maintenance_test.go` (query string y decode).
- `internal/mcp/tools/all/contract_test.go` (stubs + golden).
- `go test ./internal/mcp/...`
- `go test ./internal/api -run TestMCPBackendContract`
- `./scripts/check-mcp-impact.sh --staged --gate`
- `./scripts/check-mcp-docs.sh --staged`

**Validación subagente:** explore — sesión `ses_df068cdb9ffeou8JNKr2tzqD6F` — veredicto: viable; 2 tools nuevas (`home_maintenance_labor_register`, `home_maintenance_supplies_register`) y catálogo 39 → 41; las 39 existentes se declaran por `shared.affects_all` (`internal/mcp/api/`) sin cambio funcional; `home_maintenance_list` es la única existente con cambio de contrato; requiere ampliar `contract_test.go`, montar el módulo en `TestMCPBackendContract` y agregar `model.go`/`module.go` al manifiesto.

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [ ] `CA-REQ-001`: `home_maintenance_labor_register` crea una línea `labor` con `dry_run=false` y la devuelve con su `id`.
- [ ] `CA-REQ-002`: `home_maintenance_supplies_register` crea una línea `supplies` con `quantity`/`unit` y la devuelve.
- [ ] `CA-REQ-003`: sin `maintenance_id`, la tool reutiliza el mantenimiento del mismo `home_id` + `performed_at` + `area`; si no existe, lo crea con la línea en una transacción.
- [ ] `CA-REQ-003B`: si hay dos mantenimientos candidatos del mismo día y área, devuelve `ambiguous_maintenance` sin escribir.
- [ ] `CA-REQ-004`: repetir la misma línea el mismo día devuelve `status=duplicate` y **no** crea una segunda fila.
- [ ] `CA-REQ-004B`: con `allow_duplicate=true` la segunda línea se crea y la respuesta incluye `warnings`.
- [ ] `CA-REQ-005`: `dry_run` default `true` devuelve `status=planned` y `existing_today`.
- [ ] `CA-REQ-006`: `home_maintenance_list?date=2026-03-10&include_items=true` devuelve el mantenimiento del día con sus líneas y `truncated=false`.
- [ ] `CA-REQ-007`: sin `P40LA_MCP_ALLOW_WRITES`, `dry_run=false` devuelve `writes_disabled`.
- [ ] `CA-REQ-008`: `home_id`, `currency_id` e `institution_id` inexistentes devuelven el error estable correspondiente.
- [ ] `CA-REQ-009`: las escrituras llegan con `X-P40LA-Source: mcp`.
- [ ] `CA-REQ-010`: `impact.json`, `contracts.json`, `home-maintenance.md` y los conteos de `docs/mcp/` actualizados.
- [ ] `CA-REQ-013`: `contractAPI` de `internal/mcp/tools/all/contract_test.go` implementa `HomeMaintenanceAPI` y el paquete de tests compila.
- [ ] `CA-REQ-014`: `TestMCPBackendContract` monta el módulo `home.maintenance` y cubre POST con `item`, GET con filtros, detalle y POST de línea.
- [ ] `CA-REQ-015`: `totals_by_currency` conserva ambas columnas aunque `item_type` filtre los mantenimientos.
- [ ] `CA-REQ-011`: una respuesta con más de 100 mantenimientos devuelve `truncated=true`.
- [ ] `CA-REQ-012`: `query` con `%` o `_` no altera el patrón SQL.
- [ ] `CA-UX-PREREQ`: el backend valida los prerrequisitos y la tool los reporta antes de escribir.

### 7.2 No funcionales

- [ ] `CA-ARCH-IHOST`: sin dependencias nuevas; `go build ./...` y `go test ./...` pasan.
- [ ] `CA-SEC-SYNTHETIC`: todos los ejemplos y fixtures usan datos `demo-*`.
- [ ] `CA-TEST-MCP-CONTRACT`: `TestToolContracts` y `TestMCPBackendContract` verdes; golden revisado.
- [ ] `CA-DELIV-MCP-IMPACT`: `./scripts/check-mcp-impact.sh --staged --gate` verde antes del merge.
- [ ] `CA-DELIV-WORKTREE`: `./scripts/check-worktree.sh` verde en `../p40la-ihost-spec-155`.

### 7.3 Testing

- **Unit tests**: normalización de descripción, comparación de duplicados, validación de fecha, query escaping, filtros del módulo.
- **Integration tests**: `httptest` del módulo para `GET` con filtros y `POST` con `item`; rollback cuando la línea es inválida.
- **MCP tool tests**: dry-run, create, append, duplicate, allow_duplicate, ambiguous, writes_disabled, list con `date`/`item_type`/`query`/`include_items`.
- **Contrato**: `TestToolContracts` con golden actualizado; `TestMCPBackendContract` sin regresiones.
- **Carga**: no aplica; el volumen esperado es bajo y hay topes.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Backend: `Input.Item`, `CreateWithItem`, filtros `date`/`item_type`/`q`/`include_items`, topes y tests | 0.5 día | Ninguna |
| 2 | Cliente MCP: `HomeMaintenanceAPI`, tipos y `FullAPI` | 0.25 día | Fase 1 |
| 3 | Tools: `labor`, `supplies`, helpers de resolución/dedup, ampliación de `list` y tests | 0.5 día | Fase 2 |
| 4 | Manifiesto, contrato, docs, README de tools | 0.25 día | Fase 3 |
| 5 | Gate completo, merge, release y limpieza | 0.25 día | Fase 4 |

### 8.2 Milestones

1. **MVP**: backend + cliente + dos tools con dedup y dry-run; tests verdes.
2. **V1.0**: list ampliada, documentación, manifiesto y contrato; release.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Dos escrituras concurrentes pasan el chequeo de duplicado | Baja | Medio | Documentado; `home_maintenance_list?date=` permite revisar; spec futura evalúa `dedup_key` único |
| El agente confunde mantenimientos del mismo día y área | Media | Medio | `ambiguous_maintenance` con candidatos y hint para usar `maintenance_id` |
| La respuesta con `include_items` crece demasiado en el iHost | Baja | Medio | Topes 100/1.000 y `truncated=true`; cliente corta en 1 MiB |
| El cambio en `internal/mcp/api/` se interpreta como impacto en las 39 tools | Alta (analizador) | Bajo | Declarado en 6.3; sin cambio de comportamiento; tests MCP completos |
| Regenerar el golden sin revisar el diff | Baja | Medio | Diff revisado y declarado en la spec; `TestToolContracts` sin `-update` |
| Duplicado legítimo bloqueado | Baja | Bajo | `allow_duplicate=true` con `warnings` |

---

## 10. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-141 (módulo `home.maintenance`), SPEC-142 (proveedor por línea), SPEC-145 (MCP y compuerta de escritura), SPEC-153 (guardrail de impacto), SPEC-154 (catálogo de tools, `home_maintenance_list`).
- Documentación: `docs/mcp/tools/home-maintenance.md`, `docs/mcp/tools/impact.md`.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial de la especificación; issue #158 |
| 2026-11-24 | opencode | Implementación: `CreateInput`/`CreateWithItem`, filtros `date`/`item_type`/`q`/`include_items`, cliente `HomeMaintenanceAPI`, tools `home_maintenance_labor_register` y `home_maintenance_supplies_register`, dedup y tests; docs MCP y contrato actualizados; `go build ./...` y `go test ./...` verdes |
