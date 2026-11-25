---
title: "MCP: consulta, edición y borrado de líneas en reparaciones del hogar"
id: "SPEC-159"
status: "pending_execution"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 162
---

# MCP: consulta, edición y borrado de líneas en reparaciones del hogar

**ID**: SPEC-159<br>
**Estado**: pending_execution<br>
**Autor**: opencode<br>
**Creado**: 2026-11-24<br>
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

El servidor MCP ya expone cinco tools de reparaciones del hogar: `home_maintenance_list` (SPEC-154), `home_maintenance_labor_register` y `home_maintenance_supplies_register` (SPEC-155), y `home_maintenance_create` y `home_maintenance_item_move` (SPEC-157). Con ese catálogo un agente puede listar, crear una reparación vacía, registrar líneas y moverlas entre reparaciones. Sin embargo, **no puede consultar el detalle de una reparación, corregir la cabecera, corregir una línea ni borrar una línea registrada por error**.

El backend ya soporta esas operaciones: `GET /api/homes/{id}/maintenances/{maintenance_id}`, `PUT /api/homes/{id}/maintenances/{maintenance_id}`, `PUT /api/homes/{id}/maintenances/{maintenance_id}/items/{item_id}` y `DELETE /api/homes/{id}/maintenances/{maintenance_id}/items/{item_id}`. Esta spec agrega **cuatro tools MCP delgadas** que reutilizan esos endpoints, sin cambios de esquema, migraciones ni frontend:

1. `home_maintenance_get`: detalle de una reparación con sus líneas y totales por moneda.
2. `home_maintenance_update`: edición parcial de la cabecera (fecha, tipo, área, institución y descripción).
3. `home_maintenance_item_update`: edición parcial de una línea (tipo, descripción, cantidad, unidad, monto, moneda e institución).
4. `home_maintenance_item_delete`: borrado de una línea, con `dry_run` por defecto.

El problema que resuelve es concreto: si una línea se registra con el monto, la moneda o la fecha equivocada, hoy el agente solo puede moverla a otra reparación o pedirle al usuario que la corrija desde la web. Con estas tools el agente puede inspeccionar, corregir y borrar sin salir del flujo MCP, manteniendo la compuerta de escritura (`P40LA_MCP_ALLOW_WRITES`) y la previsualización (`dry_run=true`).

El cambio no agrega dependencias ni migraciones. En el iHost el costo es acotado: una consulta de detalle para el `GET` y para el merge previo a cada edición, un `PUT`/`DELETE` por escritura y, en las ediciones y borrados de líneas, una consulta final para devolver los totales recalculados. No hay estado en memoria ni procesos en background.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec toca backend, MCP y SQLite; **no toca frontend, rutas de UI, i18n ni migraciones**. Las reglas de UI/UX se declaran `N/A` con justificación.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go stdlib + SQLite; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | El cliente sigue en `internal/mcp/api/`; las tools en `internal/mcp/tools/homemaint/`; el dominio sigue en `internal/modules/home/maintenance/` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Tool MCP → cliente HTTP → handler → service → storage. **Prohibido** SQL desde el MCP; el módulo conserva `handlers.go`, `service.go`, `storage.go` | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | `home.maintenance` mantiene su namespace y no importa storage de otros módulos; no se agregan migraciones ni tablas | `CA-ARCH-MODULES` |
| `ARCH-MCP-TOOLS-MAP` | Las cuatro tools se registran con nombre literal y entrada en `docs/mcp/tools/impact.json`; el contrato se regenera de forma intencional | `CA-ARCH-MCP-TOOLS-MAP` |
| `ARCH-IHOST` | Sin caché ni estado; máximo una consulta de detalle por edición y una consulta final para totales; respuestas acotadas por el cliente (1 MiB) | `CA-ARCH-IHOST` |
| `ARCH-GO` | `gofmt`, nombres en inglés, errores explícitos, `log/slog` sin cambios | `CA-ARCH-GO` |
| `DATA-SQLITE` | Se reutilizan `home_maintenances` y `home_maintenance_items`; las escrituras siguen en `storage.go` con SQL crudo | `CA-DATA-SQLITE` |
| `DATA-MIGRATIONS` | **N/A**: no hay cambios de esquema; las tablas e índices de `0057_home_maintenance` alcanzan | — |
| `DATA-PROTECTED-SCHEMA` | **N/A**: no se toca `debts` / `debt_bills` | — |
| `DATA-NO-DROP` | **N/A**: no se ejecuta nada sobre `data/app.db` | — |
| `DATA-PRODCOPY` | **N/A**: la validación usa DB temporal y fixtures sintéticos; no depende de datos reales | — |
| `DATA-PROD-WRITE` | **N/A**: no se escribe en producción | — |
| `SEC-NO-PII` | Las tools devuelven descripciones libres y montos del propio usuario; no agregan emails, teléfonos ni identificadores. Los ejemplos son sintéticos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Casa Demo`, `Proveedor Demo`, montos ficticios y fechas `2026-03-10`/`2026-03-11` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Los logs del MCP no agregan argumentos, montos ni descripciones; la auditoría existente no cambia | `CA-SEC-LOGS` |
| `SEC-COMMIT` | `git diff --cached` se revisa; sin DBs, PDFs, exports ni secretos | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks instalados con `./scripts/install-hooks.sh`; sin `--no-verify` | `CA-SEC-HOOKS` |
| `SEC-INCIDENT` | **N/A**: la spec no introduce secretos ni PII; si apareciera un dato real en pruebas, se aplica el protocolo | — |
| `UI-*` (todas) | **N/A**: no se toca frontend ni i18n | — |
| `UX-BACK` / `UX-MODAL` / `UX-MODAL-CONFIRM` | **N/A**: no se agregan rutas ni modales | — |
| `UX-DETAIL-ACTIONS` / `UX-DETAIL-READONLY` | **N/A**: no se modifica ninguna página de detalle | — |
| `UX-PREREQ` | Aplica al MCP: la vivienda, la reparación y la línea se validan antes de escribir; la línea debe pertenecer a la reparación y la reparación a la vivienda | `CA-UX-PREREQ` |
| `UX-EMPTY` / `UX-LOADING` / `UX-FEEDBACK` / `UX-FORMS` | **N/A**: no hay UI; el feedback del MCP se expresa con `status`, `changed_fields`, `warnings` y errores estables | — |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract` antes del commit | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | Golden `contracts.json` regenerado de forma intencional y revisado; contrato backend↔MCP verde | `CA-TEST-MCP-CONTRACT` |
| `TEST-SERVER` | Si se levanta el server para pruebas manuales, `./scripts/check-server.sh` antes de avisar | `CA-TEST-SERVER` |
| `TEST-KILLALL` | **N/A** en esta spec: no se matan procesos; si se levanta el server se usa `killall`, nunca `pkill` | — |
| `TEST-TESTUSER` | **N/A**: las pruebas son unitarias/integración con API key sintética y `httptest` | — |
| `TEST-USER-FLOW` | Antes de `pending_release` se pregunta al usuario por las pruebas manuales y se espera confirmación | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | La spec vive en `../p40la-ihost-spec-159` sobre `feature/SPEC-159`; `./scripts/check-worktree.sh` pasa | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | No hay merge ni push sin confirmación explícita del usuario | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Commits `SPEC-159: ...` | `CA-DELIV-COMMIT` |
| `DELIV-RELEASE` | Merge, push, tracker, label, cierre de issue y limpieza del worktree en el mismo ciclo | `CA-DELIV-RELEASE` |
| `DELIV-MCP-DOCS` | `docs/mcp/tools/home-maintenance.md`, `docs/mcp/tools/README.md`, `docs/mcp/README.md` y `docs/mcp/security.md` se actualizan en el mismo commit | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente en paralelo a la implementación | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa; `./scripts/check-mcp-impact.sh --files ...` y `--staged --gate` antes del commit | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; veredicto citado en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-HOOKS` | Hooks instalados; `./scripts/install-hooks.sh` al crear el worktree | `CA-DELIV-HOOKS` |
| `DELIV-LABELS` | Issue con `spec/draft` y transiciones a `spec/in-progress`, `spec/pending-release` y `spec/released` | `CA-DELIV-LABELS` |
| `DELIV-DOCKER` | **N/A**: esta spec no publica imagen ni cambia el Dockerfile; la imagen MCP no se publica en este ciclo | — |
| `DELIV-DEPLOY` | **N/A**: no se despliega al iHost en esta spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Exponer `home_maintenance_get` para devolver el detalle de una reparación: cabecera, líneas y `totals_by_currency`. Entrada obligatoria: `home_id` y `maintenance_id`. Tool de solo lectura, sin compuerta de escritura.
2. **REQ-002**: Exponer `home_maintenance_update` para editar la cabecera con **merge parcial**: `performed_at`, `kind`, `area`, `institution_id` y `description` son opcionales; lo omitido conserva el valor actual. La tool hace `GET` del detalle y envía un `PUT` con los cinco campos completos, porque el backend usa semántica de reemplazo.
3. **REQ-003**: `home_maintenance_update` valida los valores efectivos antes de escribir: fecha `YYYY-MM-DD` no futura, `kind` en `corrective|preventive`, `area` en el enum del dominio, `description` de hasta 500 caracteres e `institution_id` existente. El backend revalida.
4. **REQ-004**: Exponer `home_maintenance_item_update` para editar una línea con merge parcial sobre el detalle: `item_type`, `description`, `quantity`, `unit`, `amount`, `currency_id` e `institution_id`. Envía un `PUT` con el `ItemInput` completo.
5. **REQ-005**: `home_maintenance_item_update` valida los valores efectivos: `item_type` en `labor|supplies`, descripción obligatoria de hasta 200 caracteres, `amount` finito y mayor o igual a 0, `currency_id` existente, `quantity` mayor que 0 y `unit` del enum. Para `labor` fuerza `quantity=1` y `unit=unit`, igual que el backend.
6. **REQ-006**: Exponer `home_maintenance_item_delete` para borrar una línea por `item_id`. La tool valida que la línea exista en la reparación y en la vivienda antes de borrar. **No** borra la cabecera de la reparación aunque quede sin líneas.
7. **REQ-007**: `dry_run` viene activado por defecto en las tres tools de escritura. En modo preview no se escribe y se devuelve el plan: para updates, `current`, `proposed` y `changed_fields`; para delete, la línea que se borraría, `remaining_item_count` y `maintenance_empty`.
8. **REQ-008**: Respetar la compuerta de escritura existente: `P40LA_MCP_ALLOW_WRITES=true` (y `P40LA_MCP_REMOTE_WRITES=true` en HTTP). Sin la compuerta, `dry_run=false` devuelve `writes_disabled` antes de resolver o escribir.
9. **REQ-009**: Después de editar o borrar una línea, devolver `totals_by_currency` **recalculados** por el backend. La tool no calcula totales por su cuenta: hace un `GET` del detalle después de la escritura. Esto evita repetir la lógica de desglose `labor`/`supplies` y cubre el caso de líneas en varias monedas.
10. **REQ-010**: Si el merge no produce cambios efectivos, devolver `status=unchanged` sin ejecutar `PUT`. La operación es idempotente y no actualiza `updated_at` innecesariamente.
11. **REQ-011**: Marcar las escrituras con `X-P40LA-Source: mcp` mediante el cliente HTTP existente (`sourceHeaders()`).
12. **REQ-012**: Ampliar `api.HomeMaintenanceAPI` con `UpdateHomeMaintenance`, `UpdateHomeMaintenanceItem` y `DeleteHomeMaintenanceItem`; implementar los métodos en `internal/mcp/api/home_maintenance.go` y actualizar los fakes de test.
13. **REQ-013**: Actualizar `docs/mcp/tools/impact.json`, `docs/mcp/tools/contracts.json`, `docs/mcp/tools/home-maintenance.md`, `docs/mcp/tools/README.md`, `docs/mcp/tools/impact.md`, `docs/mcp/README.md` y `docs/mcp/security.md` en el mismo ciclo.
14. **REQ-014**: **Fuera de alcance**: `home_maintenance_delete` (borrado de la cabecera y sus líneas en cascada). Se evalúa en una spec propia por ser destructiva y por el riesgo de perder el histórico completo.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-015**: En `home_maintenance_update` y `home_maintenance_item_update`, `institution_id=0` limpia la institución; omitirlo conserva la actual. En la cabecera, `description=""` limpia la descripción; en la línea, la descripción vacía es inválida (`item_description_required`).
2. **REQ-016**: Devolver `changed_fields` con los nombres de los campos que cambian, tanto en `dry_run` como en la escritura, para que el agente pueda explicar la edición.
3. **REQ-017**: `home_maintenance_item_update` permite cambiar `item_type`; documentar que `labor → supplies` conserva `quantity=1` y `unit=unit` del merge, y que `supplies → labor` descarta cantidad y unidad y fuerza los valores de mano de obra.
4. **REQ-018**: `home_maintenance_item_delete` devuelve `maintenance_empty=true` y un warning cuando se borra la última línea: la reparación vacía queda visible en `home_maintenance_list` y se puede completar después o mover a otra.
5. **REQ-019**: Si el `GET` posterior a un borrado falla, devolver `status=deleted` con un warning y sin totales; no convertir un borrado exitoso en error.
6. **REQ-020**: `home_maintenance_get` devuelve `item_count` además del detalle, para que el agente sepa si la reparación está vacía sin recorrer el array.
7. **REQ-021**: Errores estables y accionables: `invalid_request`, `invalid_date`, `future_date`, `invalid_kind`, `invalid_area`, `invalid_item_type`, `item_description_required`, `invalid_amount`, `currency_required`, `invalid_quantity`, `invalid_unit`, `writes_disabled` (400); `home_not_found`, `maintenance_not_found`, `item_not_found`, `currency_not_found`, `institution_not_found` (404); `upstream_error`.
8. **REQ-022**: Para `item_not_found`, incluir un hint que indique usar `home_maintenance_get` o `home_maintenance_list` con `include_items=true` para obtener los IDs correctos.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-023**: Evaluar en una spec futura el borrado de la cabecera (`home_maintenance_delete`) con confirmación reforzada y advertencia de cascada.
2. **REQ-024**: Evaluar en una spec futura el control de concurrencia optimista (`updated_at` o ETag) para evitar el lost update entre el `GET` de merge y el `PUT`.
3. **REQ-025**: Evaluar en una spec futura una tool de edición masiva de líneas (`home_maintenance_items_update`) si el uso real lo justifica; esta spec se limita a una línea por llamada.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: `home_maintenance_get` es una consulta de detalle; cada update hace un `GET` + un `PUT`; cada edición o borrado de línea agrega un `GET` final para totales. Sin caché ni estado.
- **Memoria**: las respuestas reutilizan los structs existentes y el cliente corta en 1 MiB; no se agregan estructuras de tamaño relevante.
- **Seguridad**: los endpoints siguen autenticados por API key/sesión; las tools no agregan secretos ni PII. `item_delete` declara `destructiveHint=true`.
- **Almacenamiento**: sin migraciones ni archivos nuevos de datos; solo `UPDATE`/`DELETE` sobre `home_maintenances` y `home_maintenance_items`.
- **Disponibilidad**: si el `PUT` falla, no se modifica nada; si el `DELETE` falla, no se borra nada. El `GET` posterior a un borrado exitoso es best-effort (REQ-019).
- **iHost**: sin dependencias nuevas, sin procesos en background y con un máximo de tres llamadas HTTP por escritura.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `internal/modules/home/maintenance/handlers.go` ya monta `GET`, `PUT` y `DELETE` de la cabecera y `POST`, `PUT` y `DELETE` de las líneas. No hay que agregar rutas.
- `Service.Update` valida con `prepare()`: fecha obligatoria y no futura, `kind`/`area` con defaults, descripción de hasta 500 e institución existente. `Storage.Update` hace un `UPDATE` de los cinco campos.
- `Service.UpdateItem` valida con `prepareItem()`: tipo obligatorio, descripción obligatoria de hasta 200, monto finito y no negativo, moneda obligatoria y existente, cantidad/unidad para insumos e institución existente. Para `labor` fuerza `quantity=1` y `unit=unit`.
- `Service.Get` devuelve la cabecera con las líneas y `totals_by_currency`; `Service.Update` devuelve `Get` después del `UPDATE`.
- `Service.UpdateItem` devuelve solo la línea; `Service.DeleteItem` devuelve solo error. Por eso la tool necesita un `GET` posterior para los totales.
- `internal/mcp/api/home_maintenance.go` ya tiene `GetHomeMaintenance`, `CreateHomeMaintenance`, `CreateHomeMaintenanceItem` y `MoveHomeMaintenanceItems`, pero no `Update` ni `Delete`.
- `internal/mcp/api/client.go` (`doJSON`) ya soporta `DELETE` con `out=nil`, por lo que no hace falta un helper nuevo.
- El patrón de merge parcial ya se usó en `homes_update` (SPEC-156): `GET`, completar los campos omitidos y `PUT` con el cuerpo completo.
- El patrón de `dry_run` + compuerta ya está en `writeMode` (`internal/mcp/tools/homemaint/register.go`), reutilizable por las tres tools de escritura.
- Antecedente de uso: un agente necesitó mover líneas de una reparación de pintura a otra y corregir importes; SPEC-157 cubrió el alta vacía y el movimiento, pero no la corrección ni el borrado de líneas. Esta spec cierra ese ciclo.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Agregar un `PATCH` parcial en el backend | Semántica explícita de merge | Ruta nueva, más superficie y validaciones duplicadas; el `PUT` actual ya funciona | ❌ Rechazada |
| B. Merge parcial en la tool sobre el `PUT` existente | Sin rutas nuevas; consistente con `homes_update`; el backend sigue siendo la única validación de negocio | Una consulta extra (`GET`) por edición | ✅ Seleccionada |
| C. Exigir todos los campos en las tools de update | Menos lógica en la tool | El agente debe conocer y reenviar todos los valores; riesgo de pisar datos | ❌ Rechazada |
| D. Calcular los totales en la tool | Evita el `GET` final | Duplica la lógica de desglose por moneda y tipo; riesgo de divergencia | ❌ Rechazada |
| E. Exponer también `home_maintenance_delete` | Ciclo completo | Borra la cabecera y sus líneas en cascada; el catálogo lo marca como alto riesgo | ❌ Fuera de alcance (REQ-014) |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Merge parcial en la tool, no en el backend.
- **Contexto**: el `PUT` de la cabecera y el de la línea son reemplazos completos; enviar solo un campo deja el resto con defaults (`kind=corrective`, `area=other`, `quantity=1`, `unit=unit`).
- **Decisión**: la tool hace `GET`, completa los campos omitidos con los valores actuales y envía el cuerpo completo.
- **Consecuencias**: una consulta extra por edición; el backend no cambia y sigue validando todo. Es el mismo patrón de `homes_update` (SPEC-156).

**ADR-002**: Centinelas explícitos para limpiar campos.
- **Contexto**: en JSON, omitir un campo y enviarlo como `null` se decodifican igual en los structs con punteros.
- **Decisión**: `institution_id=0` limpia la institución; `description=""` limpia la descripción de la cabecera. Omitir conserva el valor actual.
- **Consecuencias**: la semántica queda documentada en el schema de la tool y en `docs/mcp/tools/home-maintenance.md`.

**ADR-003**: Los totales se recalculan en el backend.
- **Contexto**: `totals_by_currency` agrupa por moneda y separa `labor_cost`/`supplies_cost`; recalcularlo en la tool duplicaría `totalsFromItems`.
- **Decisión**: después de `PUT`/`DELETE` de una línea, la tool hace `GET` del detalle y devuelve los totales del backend.
- **Consecuencias**: una consulta extra por edición/borrado de línea; los totales siempre coinciden con la web y con `home_maintenance_get`.

**ADR-004**: El borrado de línea no borra la cabecera.
- **Contexto**: una reparación sin líneas es válida (SPEC-157) y puede completarse después; borrarla automáticamente perdería la cabecera y su histórico.
- **Decisión**: `home_maintenance_item_delete` solo borra la línea y reporta `maintenance_empty=true` con un warning.
- **Consecuencias**: el agente debe decidir explícitamente si quiere mover la reparación vacía o dejarla; el borrado de cabecera queda para una spec futura (REQ-023).

**ADR-005**: Sin control de concurrencia optimista en esta spec.
- **Contexto**: el backend no expone versión ni ETag; el `GET` de merge y el `PUT` son dos llamadas.
- **Decisión**: aceptar last-write-wins y documentarlo como riesgo; evaluar `updated_at`/ETag en una spec futura (REQ-024).
- **Consecuencias**: en un uso concurrente, una edición puede pisar otra; el caso esperado es un agente por usuario.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
Agente MCP
   |  home_maintenance_get / update / item_update / item_delete
   v
internal/mcp/tools/homemaint  --(cliente HomeMaintenanceAPI)-->  API HTTP
   |                                                                 |
   |  GET    /api/homes/{id}/maintenances/{mid}                       v
   |  PUT    /api/homes/{id}/maintenances/{mid}              handlers.go (home.maintenance)
   |  PUT    /api/homes/{id}/maintenances/{mid}/items/{item_id}      |
   |  DELETE /api/homes/{id}/maintenances/{mid}/items/{item_id}      v
   +----------------------------------------------------------> service.go
                                                                     |
                                                                storage.go
                                                                     |
                                                          SQLite: home_maintenances
                                                                  home_maintenance_items
```

### 5.2 Componentes

#### 5.2.1 Módulo `home.maintenance` (backend)

- **Responsabilidad**: validar la cabecera y las líneas, aplicar los `UPDATE`/`DELETE` y devolver el detalle con totales.
- **Interfaz**: `GET`, `PUT` y `DELETE` ya existentes; **sin cambios de rutas**.
- **Dependencias**: `database/sql`, tablas `home_maintenances`, `home_maintenance_items`, `homes`, `currencies`, `institutions`.
- **Ubicación**: `internal/modules/home/maintenance/`.

#### 5.2.2 Cliente MCP

- **Responsabilidad**: transportar los `PUT`/`DELETE`, decodificar las respuestas y enviar `X-P40LA-Source: mcp`.
- **Interfaz**: se amplía `HomeMaintenanceAPI` con `UpdateHomeMaintenance`, `UpdateHomeMaintenanceItem` y `DeleteHomeMaintenanceItem`. Para la cabecera se usa un tipo dedicado `HomeMaintenanceUpdateRequest` (sin `item`) para que el `PUT` no pueda arrastrar una línea por accidente.
- **Dependencias**: `internal/mcp/api/client.go` (`doJSON`, `sourceHeaders`).
- **Ubicación**: `internal/mcp/api/home_maintenance.go`.

#### 5.2.3 Tools MCP

- **Responsabilidad**: validar la forma, resolver el detalle actual, aplicar el merge, respetar `dry_run`/compuerta y devolver un resultado accionable.
- **Interfaz**: `home_maintenance_get`, `home_maintenance_update`, `home_maintenance_item_update`, `home_maintenance_item_delete`.
- **Dependencias**: `api.HomeMaintenanceAPI`, `config.Config`, `toolkit`, helpers existentes de `homemaint`.
- **Ubicación**: `internal/mcp/tools/homemaint/get.go`, `update.go`, `item_update.go`, `item_delete.go`.

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

`home_maintenance_update` cambia columnas de la cabecera y `updated_at`; `home_maintenance_item_update` cambia columnas de la línea y `updated_at`; `home_maintenance_item_delete` borra una fila de `home_maintenance_items`.

### 5.4 APIs / Contratos

#### Tool `home_maintenance_get`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda de la reparación |
| `maintenance_id` | integer | Sí | Reparación a consultar |

**Salida**:

```json
{
  "home_id": 3,
  "item_count": 2,
  "maintenance": {
    "id": 12,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "paint",
    "institution_id": 7,
    "institution_name": "Proveedor Demo",
    "description": "Reparación demo de pintura",
    "items": [
      {
        "id": 101,
        "maintenance_id": 12,
        "item_type": "labor",
        "description": "Mano de obra demo",
        "quantity": 1,
        "unit": "unit",
        "amount": 500.00,
        "currency_id": 1,
        "currency_code": "NIO"
      }
    ],
    "totals_by_currency": [
      {
        "currency_id": 1,
        "currency_code": "NIO",
        "labor_cost": 500.00,
        "supplies_cost": 0,
        "total": 500.00
      }
    ]
  }
}
```

**Anotaciones**: `readOnlyHint=true`, `openWorldHint=false`.
**Errores**: `invalid_request`, `home_not_found`, `maintenance_not_found`, `upstream_error`.

#### Tool `home_maintenance_update`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda de la reparación |
| `maintenance_id` | integer | Sí | Reparación a editar |
| `performed_at` | string | No | `YYYY-MM-DD`, no futura; omitir conserva |
| `kind` | string | No | `corrective` o `preventive`; omitir conserva |
| `area` | string | No | Enum de áreas; omitir conserva |
| `institution_id` | integer | No | ID existente; `0` limpia; omitir conserva |
| `description` | string | No | Hasta 500; `""` limpia; omitir conserva |
| `dry_run` | boolean | No | Default `true` |

Debe venir al menos un campo editable; si no, `invalid_request`.

**Salida con `dry_run`**:

```json
{
  "status": "planned",
  "dry_run": true,
  "maintenance_id": 12,
  "changed_fields": ["area", "description"],
  "current": {
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "other",
    "institution_id": 7,
    "description": "Reparación demo"
  },
  "proposed": {
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "paint",
    "institution_id": 7,
    "description": "Reparación demo de pintura"
  }
}
```

**Salida con escritura**:

```json
{
  "status": "updated",
  "dry_run": false,
  "maintenance_id": 12,
  "changed_fields": ["area", "description"],
  "maintenance": {
    "id": 12,
    "home_id": 3,
    "performed_at": "2026-03-12",
    "kind": "corrective",
    "area": "paint",
    "institution_id": 7,
    "institution_name": "Proveedor Demo",
    "description": "Reparación demo de pintura",
    "items": [],
    "totals_by_currency": []
  }
}
```

Si no hay cambios efectivos: `{"status":"unchanged","dry_run":false,"maintenance_id":12,"changed_fields":[]}` y no se ejecuta `PUT`.

**Anotaciones**: `destructiveHint=true` (la edición reemplaza los cinco campos de la cabecera), `idempotentHint=true`, `openWorldHint=false`.
**Errores**: `invalid_request`, `invalid_date`, `future_date`, `invalid_kind`, `invalid_area`, `home_not_found`, `maintenance_not_found`, `institution_not_found`, `writes_disabled`, `upstream_error`.

#### Tool `home_maintenance_item_update`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda de la reparación |
| `maintenance_id` | integer | Sí | Reparación que contiene la línea |
| `item_id` | integer | Sí | Línea a editar |
| `item_type` | string | No | `labor` o `supplies`; omitir conserva |
| `description` | string | No | Hasta 200; vacía es inválida; omitir conserva |
| `quantity` | number | No | Mayor que 0 para insumos; `labor` fuerza `1` |
| `unit` | string | No | Enum de unidades; `labor` fuerza `unit` |
| `amount` | number | No | Finito y mayor o igual a 0 |
| `currency_id` | integer | No | Moneda existente |
| `institution_id` | integer | No | ID existente; `0` limpia; omitir conserva |
| `dry_run` | boolean | No | Default `true` |

Debe venir al menos un campo editable; si no, `invalid_request`.

**Salida con `dry_run`**:

```json
{
  "status": "planned",
  "dry_run": true,
  "maintenance_id": 12,
  "item_id": 101,
  "changed_fields": ["amount", "currency_id"],
  "current": {
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 500.00,
    "currency_id": 1,
    "institution_id": 7
  },
  "proposed": {
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 650.00,
    "currency_id": 2,
    "institution_id": 7
  }
}
```

**Salida con escritura**:

```json
{
  "status": "updated",
  "dry_run": false,
  "maintenance_id": 12,
  "item_id": 101,
  "changed_fields": ["amount", "currency_id"],
  "item": {
    "id": 101,
    "maintenance_id": 12,
    "item_type": "labor",
    "description": "Mano de obra demo",
    "quantity": 1,
    "unit": "unit",
    "amount": 650.00,
    "currency_id": 2,
    "currency_code": "USD",
    "institution_id": 7,
    "institution_name": "Proveedor Demo"
  },
  "totals_by_currency": [
    {
      "currency_id": 2,
      "currency_code": "USD",
      "labor_cost": 650.00,
      "supplies_cost": 0,
      "total": 650.00
    }
  ]
}
```

**Anotaciones**: `destructiveHint=true` (la edición reemplaza los siete campos de la línea), `idempotentHint=true`, `openWorldHint=false`.
**Errores**: `invalid_request`, `invalid_item_type`, `item_description_required`, `item_description_too_long`, `invalid_amount`, `currency_required`, `invalid_quantity`, `invalid_unit`, `home_not_found`, `maintenance_not_found`, `item_not_found`, `currency_not_found`, `institution_not_found`, `writes_disabled`, `upstream_error`.

#### Tool `home_maintenance_item_delete`

**Entrada**:

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | Vivienda de la reparación |
| `maintenance_id` | integer | Sí | Reparación que contiene la línea |
| `item_id` | integer | Sí | Línea a borrar |
| `dry_run` | boolean | No | Default `true` |

**Salida con `dry_run`**:

```json
{
  "status": "planned",
  "dry_run": true,
  "maintenance_id": 12,
  "item_id": 101,
  "would_delete": {
    "id": 101,
    "item_type": "labor",
    "description": "Mano de obra demo",
    "amount": 650.00,
    "currency_id": 2,
    "currency_code": "USD"
  },
  "remaining_item_count": 1,
  "maintenance_empty": false,
  "warnings": []
}
```

**Salida con escritura**:

```json
{
  "status": "deleted",
  "dry_run": false,
  "maintenance_id": 12,
  "item_id": 101,
  "remaining_item_count": 0,
  "maintenance_empty": true,
  "totals_by_currency": [],
  "warnings": [
    "La reparación quedó sin líneas. home_maintenance_item_delete no borra la cabecera; usá home_maintenance_create o move líneas si querés conservarla."
  ]
}
```

**Anotaciones**: `destructiveHint=true`, `idempotentHint=true` (repetir el borrado no cambia el estado; el segundo intento devuelve `item_not_found`), `openWorldHint=false`.
**Errores**: `invalid_request`, `home_not_found`, `maintenance_not_found`, `item_not_found`, `writes_disabled`, `upstream_error`.

### 5.5 Dependencias

- **Internas**: `internal/modules/home/maintenance`, `internal/mcp/api`, `internal/mcp/tools/homemaint`, `internal/mcp/tools/all`, `internal/api` (tests de contrato).
- **Externas**: ninguna nueva.

### 5.6 Archivos previstos

| Archivo | Cambio |
|---|---|
| `internal/mcp/api/home_maintenance.go` | tipo `HomeMaintenanceUpdateRequest` y métodos `UpdateHomeMaintenance`, `UpdateHomeMaintenanceItem`, `DeleteHomeMaintenanceItem` |
| `internal/mcp/api/home_maintenance_test.go` | encode/decode de `PUT`/`DELETE` y `X-P40LA-Source` |
| `internal/mcp/tools/homemaint/homemaint.go` | registro de las cuatro tools nuevas |
| `internal/mcp/tools/homemaint/get.go` | nuevo; `home_maintenance_get` |
| `internal/mcp/tools/homemaint/update.go` | nuevo; `home_maintenance_update` y helpers de merge |
| `internal/mcp/tools/homemaint/item_update.go` | nuevo; `home_maintenance_item_update` |
| `internal/mcp/tools/homemaint/item_delete.go` | nuevo; `home_maintenance_item_delete` |
| `internal/mcp/tools/homemaint/homemaint_test.go` | get/update/item_update/item_delete, dry-run, compuerta, merge, no-op, totales |
| `internal/mcp/tools/all/contract_test.go` | stub de los tres métodos nuevos en el fake del contrato |
| `internal/api/mcp_backend_contract_test.go` | cadena real: get, update, item_update e item_delete con `source=mcp` |
| `internal/modules/home/maintenance/service_test.go` | update de cabecera, update de línea y delete de línea con totales (si faltan casos) |
| `internal/modules/home/maintenance/handlers_test.go` | `httptest` de `PUT`/`DELETE` y mapeo de errores (si faltan casos) |
| `docs/mcp/tools/impact.json` | 4 tools nuevas con dependencias, endpoints y tests |
| `docs/mcp/tools/contracts.json` | golden regenerado (44 → 48) |
| `docs/mcp/tools/home-maintenance.md` | secciones de get/update/item_update/item_delete |
| `docs/mcp/tools/README.md` | catálogo 44 → 48 (25 lectura + 23 escritura) |
| `docs/mcp/tools/impact.md` | conteos 44 → 48 y tools nuevas |
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
| `home_maintenance_get` | `internal/mcp/tools/homemaint/get.go` | `GET /api/homes/{id}/maintenances/{maintenance_id}` | Nueva en `contracts.json`; solo lectura |
| `home_maintenance_update` | `internal/mcp/tools/homemaint/update.go` | `GET` + `PUT /api/homes/{id}/maintenances/{maintenance_id}` | Nueva en `contracts.json`; merge parcial y `dry_run` |
| `home_maintenance_item_update` | `internal/mcp/tools/homemaint/item_update.go` | `GET /api/homes/{id}/maintenances/{maintenance_id}` + `PUT .../items/{item_id}` | Nueva en `contracts.json`; merge parcial y `dry_run` |
| `home_maintenance_item_delete` | `internal/mcp/tools/homemaint/item_delete.go` | `GET /api/homes/{id}/maintenances/{maintenance_id}` + `DELETE .../items/{item_id}` | Nueva en `contracts.json`; `destructiveHint=true` y `dry_run` |

### 6.2 Tools existentes afectadas funcionalmente

| Tool | Ruta de dependencia | Motivo | Contrato |
|---|---|---|---|
| `home_maintenance_list` | tool → `internal/mcp/api/home_maintenance.go` → `GET /api/homes/{id}/maintenances` → `handlers.go` → `service.go` → `storage.go` → `home_maintenances` + `home_maintenance_items` | Sin cambio funcional; el paquete compartido se recompila y las nuevas tools modifican datos que el listado ya muestra | Sin cambios |
| `home_maintenance_create` | tool → `internal/mcp/api/home_maintenance.go` → `POST /api/homes/{id}/maintenances` → handler → service → storage | Sin cambio funcional | Sin cambios |
| `home_maintenance_labor_register` | tool → `internal/mcp/api/home_maintenance.go` → `POST /api/homes/{id}/maintenances` / `.../items` → handler → service → storage | Sin cambio funcional | Sin cambios |
| `home_maintenance_supplies_register` | idem | Sin cambio funcional | Sin cambios |
| `home_maintenance_item_move` | tool → `internal/mcp/api/home_maintenance.go` → `GET` + `POST .../items/move` → handler → service → storage | Sin cambio funcional | Sin cambios |

### 6.3 Ruta compartida `internal/mcp/api/` y `internal/mcp/tools/all/`

El manifiesto declara ambos directorios en `shared.affects_all`. Agregar tres métodos a `HomeMaintenanceAPI`, registrar cuatro tools y tocar el directorio `internal/mcp/tools/homemaint/` hace que el analizador reporte **las 44 tools existentes** por recompilación, aunque su comportamiento no cambia. El gate exige declararlas; se listan a continuación para que quede explícito:

`alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bank_accounts_create`, `bank_accounts_search`, `bank_accounts_update`, `banks_create`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `credit_card_statements_get`, `credit_card_statements_list`, `credit_cards_create`, `credit_cards_search`, `credit_cards_update`, `debt_bills_list`, `entities_search`, `home_maintenance_create`, `home_maintenance_item_move`, `home_maintenance_labor_register`, `home_maintenance_list`, `home_maintenance_supplies_register`, `homes_create`, `homes_update`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

**Impacto real**: nulo en el comportamiento de esas 44 tools; solo se recompila el paquete `internal/mcp/api`, el paquete `internal/mcp/tools/homemaint` y el catálogo de tests. Las tools con cambio funcional son las cuatro nuevas.

### 6.4 Contrato y tests

- `docs/mcp/tools/contracts.json`: se regenera con `go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts`; se revisa el diff.
- `docs/mcp/tools/impact.json`: se agregan las cuatro tools nuevas con `docs`, `entrypoints`, `dependencies`, `endpoints` y `tests`.
- `internal/mcp/tools/all/contract_test.go`: el fake del contrato debe implementar `UpdateHomeMaintenance`, `UpdateHomeMaintenanceItem` y `DeleteHomeMaintenanceItem`, o el paquete de tests no compila.
- `internal/api/mcp_backend_contract_test.go`: agregar la cadena real `GET` → `PUT` → `PUT item` → `DELETE item`; verificar auth, `X-P40LA-Source: mcp`, tags JSON y `totals_by_currency` recalculados.
- `docs/mcp/README.md`, `docs/mcp/security.md` y `docs/mcp/tools/README.md`: actualizar el catálogo 44 → 48 (25 lectura + 23 escritura) y la lista de tools sujetas a `P40LA_MCP_ALLOW_WRITES`.

Tests:
- `internal/mcp/tools/homemaint/homemaint_test.go` (get; update: dry-run/escritura/no-op/merge/compuerta/errores; item_update: monto/moneda/tipo/totales; item_delete: dry-run/escritura/última línea/errores).
- `internal/mcp/api/home_maintenance_test.go` (métodos nuevos, cuerpos y `X-P40LA-Source`; `DELETE` 204).
- `internal/modules/home/maintenance/service_test.go` (update de cabecera, update de línea, delete de línea y totales multinivel).
- `internal/modules/home/maintenance/handlers_test.go` (`PUT`/`DELETE`, códigos 400/404).
- `internal/mcp/tools/all/contract_test.go` (stub + golden).
- `internal/api/mcp_backend_contract_test.go` (cadena real).
- `go test ./internal/mcp/...`
- `go test ./internal/api -run TestMCPBackendContract`
- `./scripts/check-mcp-impact.sh --staged --gate`
- `./scripts/check-mcp-docs.sh --staged`

**Validación subagente:** explore — sesión `ses_deebd1edcffeeWdFFChdo75fxB` — veredicto: **viable con cambios**. El backend ya expone `GET`/`PUT` de cabecera y `PUT`/`DELETE` de línea; no hacen falta endpoints ni migraciones. Los cambios obligatorios son de contrato, manifiesto, stubs de test y docs: `docs/mcp/tools/impact.json` y `contracts.json` en el mismo commit, `contractAPI` y `fakeAPI` deben implementar los tres métodos nuevos, y `mcp_backend_contract_test.go` debe recorrer `GET` → `PUT` → `PUT item` → `DELETE item` con `source=mcp`. El analizador reporta las 44 tools existentes por `shared.affects_all` (`internal/mcp/api/`), sin cambio funcional. El `PUT` de cabecera y el de línea son reemplazos totales: el merge debe enviar siempre los cinco campos de la cabecera y los siete de la línea. `quantity=0` se normaliza a `1` en el backend, así que la tool rechaza explícitamente `<=0`. Un `item_id` de otra reparación válida devuelve `item_not_found`; una reparación de otro hogar devuelve `maintenance_not_found`.

**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [ ] `CA-REQ-001`: `home_maintenance_get` devuelve la reparación con sus líneas y `totals_by_currency`; con `maintenance_id` inexistente devuelve `maintenance_not_found`.
- [ ] `CA-REQ-002`: `home_maintenance_update` con solo `area` conserva fecha, tipo, institución y descripción actuales; el `PUT` envía los cinco campos completos.
- [ ] `CA-REQ-003`: `home_maintenance_update` rechaza fecha futura, `kind`/`area` inválidos, descripción de más de 500 e institución inexistente sin escribir.
- [ ] `CA-REQ-004`: `home_maintenance_item_update` con solo `amount` conserva tipo, descripción, cantidad, unidad, moneda e institución; el `PUT` envía el `ItemInput` completo.
- [ ] `CA-REQ-005`: `home_maintenance_item_update` rechaza `amount` negativo, moneda inexistente, cantidad inválida y unidad inválida; para `labor` fuerza `quantity=1` y `unit=unit`.
- [ ] `CA-REQ-006`: `home_maintenance_item_delete` borra solo la línea; la cabecera sigue existiendo después del borrado.
- [ ] `CA-REQ-007`: Las tres tools de escritura con `dry_run` omitido no escriben y devuelven `planned`; con `dry_run=false` escriben.
- [ ] `CA-REQ-008`: Sin `P40LA_MCP_ALLOW_WRITES`, `dry_run=false` devuelve `writes_disabled` y no ejecuta `GET`/`PUT`/`DELETE`.
- [ ] `CA-REQ-009`: Después de editar o borrar una línea, `totals_by_currency` refleja el nuevo monto/moneda y separa `labor_cost` y `supplies_cost`.
- [ ] `CA-REQ-010`: Un update sin cambios efectivos devuelve `status=unchanged` y no ejecuta `PUT`.
- [ ] `CA-REQ-011`: Las escrituras llevan `X-P40LA-Source: mcp`.
- [ ] `CA-REQ-012`: La interfaz `HomeMaintenanceAPI` y sus fakes compilan con los tres métodos nuevos.
- [ ] `CA-REQ-013`: Manifiesto, contrato, documentación y tests se actualizan en el mismo ciclo.
- [ ] `CA-REQ-014`: No existe `home_maintenance_delete` en esta spec; el borrado de cabecera queda para una spec futura.
- [ ] `CA-REQ-015`: `institution_id=0` limpia la institución; `description=""` limpia la descripción de cabecera y la descripción vacía de una línea es inválida.
- [ ] `CA-REQ-016`: `changed_fields` se devuelve en `dry_run` y en la escritura.
- [ ] `CA-REQ-017`: Cambiar `item_type` de `labor` a `supplies` y viceversa se comporta como el backend.
- [ ] `CA-REQ-018`: Borrar la última línea devuelve `maintenance_empty=true` y un warning; no se borra la cabecera.
- [ ] `CA-REQ-019`: Si el `GET` posterior al borrado falla, la tool devuelve `deleted` con warning y sin totales.
- [ ] `CA-REQ-020`: `home_maintenance_get` devuelve `item_count`.
- [ ] `CA-REQ-021`: Los errores usan los códigos estables listados en REQ-021.
- [ ] `CA-REQ-022`: `item_not_found` incluye el hint para obtener IDs con `home_maintenance_get` o `home_maintenance_list?include_items=true`.
- [ ] `CA-ARCH-STACK`: Sin dependencias nuevas.
- [ ] `CA-ARCH-IHOST`: Sin estado en memoria ni procesos en background.
- [ ] `CA-SEC-NO-PII`: Ejemplos y tests usan solo datos sintéticos.
- [ ] `CA-SEC-SYNTHETIC`: No se usan nombres, montos ni IDs reales.
- [ ] `CA-TEST-LOCAL`: `go build ./...` y `go test ./...` pasan.
- [ ] `CA-TEST-MCP-CONTRACT`: `TestToolContracts` y `TestMCPBackendContract` pasan.
- [ ] `CA-DELIV-MCP-IMPACT`: `./scripts/check-mcp-impact.sh --staged --gate` pasa.
- [ ] `CA-DELIV-MCP-IMPACT-SUBAGENT`: La sección 6 cita agente, sesión y veredicto.
- [ ] `CA-DELIV-MCP-DOCS`: `docs/mcp/` actualizado en el mismo commit.
- [ ] `CA-DELIV-COMMIT`: Commits `SPEC-159: ...`.
- [ ] `CA-DELIV-WORKTREE`: Trabajo en `../p40la-ihost-spec-159` sobre `feature/SPEC-159`.
- [ ] `CA-DELIV-CONFIRM`: Merge y push solo con confirmación del usuario.
- [ ] `CA-DELIV-RELEASE`: Tracker, label, cierre y limpieza en el release.

### 7.2 No funcionales

- [ ] `CA-NF-001`: `home_maintenance_get` responde en menos de 1 segundo en LAN con una reparación de hasta 100 líneas.
- [ ] `CA-NF-002`: Cada update de línea hace como máximo 2 llamadas HTTP (`PUT` + `GET` de totales); cada delete hace como máximo 3 (`GET` previo + `DELETE` + `GET` final).
- [ ] `CA-NF-003`: La respuesta de `home_maintenance_get` respeta el límite de 1 MiB del cliente.
- [ ] `CA-NF-004`: No se agregan tablas, columnas, índices ni migraciones.
- [ ] `CA-NF-005`: Un fallo de `PUT`/`DELETE` no deja datos parciales; el backend mantiene la transacción de una sola fila.

### 7.3 Testing

- **Unit**: `internal/mcp/tools/homemaint` con `fakeAPI`; merge, no-op, dry-run, compuerta, errores y totales.
- **Integration**: `internal/api/mcp_backend_contract_test.go` con servidor `httptest` y API key sintética; cadena real get → update → item_update → item_delete.
- **Backend**: `internal/modules/home/maintenance` con DB temporal y fixtures sintéticos; update/delete y recálculo de totales.
- **E2E**: prueba manual con `./scripts/check-server.sh` y usuario de prueba, si el usuario la pide.
- **Carga/Performance**: N/A; el volumen esperado es de decenas de líneas por reparación.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Cliente MCP: `UpdateHomeMaintenance`, `UpdateHomeMaintenanceItem`, `DeleteHomeMaintenanceItem` + tests de encode/decode y `X-P40LA-Source` | 0.5 día | — |
| 2 | `home_maintenance_get` + registro + tests | 0.25 día | Fase 1 |
| 3 | `home_maintenance_update` con merge, no-op, dry-run y compuerta + tests | 0.5 día | Fase 1 |
| 4 | `home_maintenance_item_update` y `home_maintenance_item_delete` con totales recalculados + tests | 0.75 día | Fase 1 |
| 5 | Manifiesto, contrato, docs MCP y tests de contrato backend↔MCP | 0.5 día | Fases 2–4 |
| 6 | Verificación local, pruebas manuales y release | 0.5 día | Fase 5 |

### 8.2 Milestones

- **MVP**: las cuatro tools funcionan con `dry_run`, compuerta y tests unitarios.
- **V1.0**: contrato congelado, `docs/mcp/` actualizado, `TestToolContracts` y `TestMCPBackendContract` verdes, pruebas manuales confirmadas.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| El `PUT` de cabecera pisa campos omitidos con defaults | Alta | Alto | Merge en la tool + `GET` previo; test de edición parcial por campo |
| El `PUT` de línea pisa cantidad/unidad/moneda | Alta | Alto | Merge en la tool + validación de valores efectivos; test de edición solo de monto |
| Lost update entre el `GET` de merge y el `PUT` | Baja | Medio | Documentado (ADR-005); un agente por usuario; evaluar ETag/`updated_at` en spec futura |
| `item_delete` borra la última línea y deja una cabecera vacía | Media | Bajo | `maintenance_empty=true` + warning; no se borra la cabecera; el agente decide |
| Totales desincronizados tras editar monto/moneda | Media | Alto | `GET` posterior y totales del backend; tests con dos monedas y con `labor`+`supplies` |
| Cambio de `item_type` altera cantidad/unidad | Media | Medio | Reglas explícitas (REQ-017) y tests de ida y vuelta |
| El contrato congelado cambia sin intención | Baja | Medio | Golden regenerado y revisado; `TestToolContracts` en el gate |
| Las 44 tools existentes se reportan por `shared.affects_all` | Alta | Bajo | Declaradas en 6.3; sin cambio funcional ni de contrato |
| El `DELETE` no es idempotente y un reintento da `item_not_found` | Media | Bajo | `idempotentHint=true` (el efecto no cambia); documentar que el segundo intento devuelve `item_not_found` y que el agente debe consultar con `home_maintenance_get` antes de reintentar |
| Un registro legacy con `performed_at` futura falla al editar otro campo | Baja | Bajo | El backend revalida la fecha; documentar que primero hay que corregir la fecha. No se auto-corrige |

---

## 10. Notas y Referencias

- Reglas aplicadas: `docs/rules/00-index.md`, `10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-141 (módulo `home.maintenance`), SPEC-142 (proveedor por línea), SPEC-145 (servidor MCP y compuerta), SPEC-153 (guardrail de impacto), SPEC-154 (catálogo de tools), SPEC-155 (registro de mano de obra e insumos), SPEC-156 (edición de hogares con merge parcial), SPEC-157 (alta vacía y movimiento de líneas).
- Endpoints reutilizados: `GET/PUT /api/homes/{id}/maintenances/{maintenance_id}`, `PUT/DELETE /api/homes/{id}/maintenances/{maintenance_id}/items/{item_id}`.
- Precedente de uso: un agente necesitó crear una reparación de pintura y mover líneas desde otra reparación; SPEC-157 cubrió ese caso. La corrección y el borrado de líneas quedaron pendientes y son el objeto de esta spec.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|---|---|---|
| 2026-11-24 | opencode | Creación inicial: get, update, item_update e item_delete para reparaciones del hogar |
| 2026-11-24 | opencode | Spec lista para implementar; issue #162 y veredicto de impacto MCP integrado |
