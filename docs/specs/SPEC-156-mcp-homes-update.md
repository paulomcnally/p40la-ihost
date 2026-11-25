---
title: "MCP: tool homes_update para editar hogares"
id: "SPEC-156"
status: "released"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 159
---

# MCP: tool `homes_update` para editar hogares

**ID**: SPEC-156<br>
**Estado**: released<br>
**Autor**: opencode<br>
**Creado**: 2026-11-24<br>
**Actualizado**: 2026-11-24<br>
**Release**: 2026-11-24 — merge a `main` `1a5c025d`; implementación `14960d85`; catálogo MCP 41 → 42 tools; edición parcial de hogares con validación de uso y `already_exists`; build de Go y frontend verdes; sin migraciones ni dependencias nuevas; imagen MCP no publicada en este ciclo; issue #159 cerrado

---

## 1. Resumen Ejecutivo

El servidor MCP expone `homes_create` (SPEC-146) y `entities_search`, pero **no tiene ninguna tool para editar un hogar**. El backend ya soporta la edición con `PUT /api/homes/{id}`: nombre, dirección y uso (`rental`, `inhabited`, `unused`). Un agente que intenta cambiar el estado de una casa —por ejemplo de `rental` a `inhabited`— no encuentra una tool y queda obligado a pedirle al usuario que lo haga desde la web.

Esta spec agrega **`homes_update`**, una tool de escritura que reutiliza el endpoint existente y resuelve el hogar por `home_id` o por nombre. La tool hace **edición parcial**: los campos omitidos conservan el valor actual, lo que evita dos problemas del PUT directo: (a) tener que reenviar nombre y dirección para cambiar solo el estado, y (b) que un `usage` vacío se normalice a `rental` y revierta el estado sin que el usuario lo pida. Valida el uso contra el enum del dominio y traduce el conflicto `home_has_active_rental` (409) a un error accionable cuando la casa tiene un contrato o reserva activa.

El cambio es acotado: no agrega rutas, migraciones ni dependencias; no toca el frontend ni la base de datos. En el iHost el costo es una consulta `GET /api/homes` para resolver el hogar y un `PUT` por edición, sin estado en memoria ni procesos adicionales.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec toca el servidor MCP y su documentación; **no toca frontend, rutas de UI, i18n, migraciones ni escrituras en producción**. Las reglas de UI/UX se declaran `N/A` con justificación.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go stdlib + el SDK MCP existente; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | El cliente HTTP vive en `internal/mcp/api/entities.go`; la tool en `internal/mcp/tools/entities/homes.go`; el dominio sigue en `internal/{api,services,storage}` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Tool MCP → `api.Client.UpdateHome` → `PUT /api/homes/{id}` → handler → service → storage. **Prohibido** SQL desde el MCP | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | **N/A**: `homes` es código legacy en `internal/{api,services,storage}`; no se crea un módulo nuevo | — |
| `ARCH-MCP-TOOLS-MAP` | La tool se registra con `mcp.AddTool` y nombre literal; entrada nueva en `docs/mcp/tools/impact.json`; contrato regenerado de forma intencional | `CA-ARCH-MCP-TOOLS-MAP` |
| `ARCH-IHOST` | Sin caché ni estado; una lista y un PUT por edición; respuesta acotada por el cliente (1 MiB) | `CA-ARCH-IHOST` |
| `ARCH-GO` | `gofmt`, nombres en inglés, errores explícitos, sin nuevas dependencias | `CA-ARCH-GO` |
| `DATA-SQLITE` | **N/A**: no se agrega SQL; se reutiliza `UpdateHome` del storage existente sobre la tabla `homes` | — |
| `DATA-MIGRATIONS` | **N/A**: sin cambios de esquema | — |
| `DATA-PROTECTED-SCHEMA` | **N/A**: no se toca `debts` / `debt_bills` | — |
| `DATA-NO-DROP` | **N/A**: no se ejecuta nada sobre `data/app.db` | — |
| `DATA-PRODCOPY` | **N/A**: los tests usan SQLite en memoria y fixtures sintéticos; no dependen de datos reales | — |
| `DATA-PROD-WRITE` | **N/A**: no se escribe en producción | — |
| `SEC-NO-PII` | La tool devuelve solo `id`, `name`, `address` y `usage` del hogar; no agrega emails, teléfonos ni identificadores. Ejemplos sintéticos | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Hogar Demo`, `Calle Demo 123` y `demo-*`; nunca datos reales | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | No se agregan logs con nombre o dirección del hogar | `CA-SEC-LOGS` |
| `SEC-COMMIT` | `git diff --cached` revisado; sin DBs, PDFs, exports ni secretos | `CA-SEC-COMMIT` |
| `UI-*` (todas) | **N/A**: no se toca frontend ni i18n | — |
| `UX-BACK` | **N/A**: no se agregan rutas ni pantallas | — |
| `UX-MODAL` / `UX-MODAL-CONFIRM` | **N/A**: no se agregan modales | — |
| `UX-DETAIL-ACTIONS` / `UX-DETAIL-READONLY` | **N/A**: no se modifica ninguna página de detalle | — |
| `UX-PREREQ` | Aplica al MCP: el hogar se resuelve y valida antes de escribir; si no existe, `not_found` con opciones `id=nombre` | `CA-UX-PREREQ` |
| `UX-EMPTY` / `UX-LOADING` / `UX-FEEDBACK` / `UX-FORMS` | **N/A**: no hay UI; el feedback del MCP se expresa con el hogar actualizado o un error estable | — |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` antes del commit; tests unitarios y de contrato | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | Se actualiza `docs/mcp/tools/contracts.json` de forma intencional y se corren `TestToolContracts` + `TestMCPBackendContract` | `CA-TEST-MCP-CONTRACT` |
| `TEST-TESTUSER` | **N/A**: los tests usan `httptest` y la DB en memoria del contrato backend; no requieren el server local | — |
| `DELIV-WORKTREE` | La spec vive en `../p40la-ihost-spec-156` sobre `feature/SPEC-156`; `./scripts/check-worktree.sh` pasa | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push se hacen con la autorización explícita del usuario («pasalo a released») y sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-IMPACT` | Sección `## Impacto en tools MCP` completa; `./scripts/check-mcp-impact.sh --files ...` y `--staged --gate` antes del commit | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; veredicto citado en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-MCP-DOCS` | `docs/mcp/tools/entities.md`, `docs/mcp/tools/README.md`, `docs/mcp/README.md`, `docs/mcp/security.md` y `docs/mcp/tools/impact.md` se actualizan en el mismo commit | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente en paralelo a la implementación | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-COMMIT` | Commits `SPEC-156: ...` | `CA-DELIV-COMMIT` |
| `DELIV-LABELS` | Issue con `spec/draft`, transición a `spec/in-progress`, `spec/pending-release` y `spec/released` | `CA-DELIV-LABELS` |
| `DELIV-RELEASE` | Merge, push, tracker, label, cierre de issue y limpieza del worktree en el mismo ciclo | `CA-DELIV-RELEASE` |
| `DELIV-DOCKER` | **N/A**: esta spec no publica imagen ni cambia el Dockerfile | — |
| `DELIV-DEPLOY` | **N/A**: no se despliega al iHost en esta spec | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Registrar la tool `homes_update` en `internal/mcp/tools/entities/tools.go` con nombre literal y anotaciones de escritura.
2. **REQ-002**: Resolver el hogar destino por `home_id` o por `home` (nombre normalizado sin mayúsculas, acentos ni espacios sobrantes). Si vienen ambos, gana `home_id`. Si no existe, devolver `not_found` con hasta 10 opciones `id=nombre`; si el nombre es ambiguo, `ambiguous_reference`.
3. **REQ-003**: Edición parcial con merge sobre el hogar actual:
   - `name`: opcional; si se omite, conserva el nombre actual. Si viene, se recorta y no puede quedar vacío (`invalid_request`).
   - `address`: opcional; si se omite, conserva la dirección actual. Una cadena vacía la limpia.
   - `usage`: opcional; si se omite, conserva el uso actual. Si viene, se normaliza a minúsculas y debe ser `rental`, `inhabited` o `unused` (`invalid_value` con hint de valores válidos).
4. **REQ-004**: Enviar el `PUT` con los tres campos completos (`name`, `address`, `usage`) para respetar la semántica de reemplazo del backend; el merge ocurre en la tool y nunca deja `usage` vacío.
5. **REQ-005**: Respetar la compuerta de escritura existente: `P40LA_MCP_ALLOW_WRITES=true` (y `P40LA_MCP_REMOTE_WRITES=true` en HTTP). Sin la compuerta, devolver `writes_disabled` antes de resolver o escribir.
6. **REQ-006**: Traducir el conflicto `home_has_active_rental` (409) a un error con el mismo código y un hint: cerrar el contrato o la reserva activa antes de cambiar el uso; aclarar que `name` y `address` sí se pueden editar sin cambiar `usage`.
7. **REQ-007**: Marcar la escritura con `X-P40LA-Source: mcp` mediante el cliente HTTP existente.
8. **REQ-008**: Agregar `UpdateHome` a la interfaz `api.API`, al cliente `internal/mcp/api/entities.go` y a los fakes de test; actualizar manifiesto, contrato congelado y documentación MCP en el mismo ciclo.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Devolver el hogar actualizado (`id`, `name`, `address`, `usage`) con el mismo sobre JSON que `homes_create`.
2. **REQ-010**: Cubrir con tests: edición solo de `usage`, conservación de `name`/`address`, limpieza de `address`, `invalid_value`, `not_found`, `writes_disabled` y conflicto de alquiler activo.
3. **REQ-011**: Incluir la tool en el test de protocolo `TestProtocolToolsList` y en el contrato backend (`TestMCPBackendContract`), ejercitando el `PUT` real contra un servidor `httptest`.
4. **REQ-013**: Si se cambia el nombre, rechazar un nombre que ya pertenezca a otro hogar (comparación normalizada) con `already_exists` y el `id` existente, igual que `homes_create`. No se aplica cuando el nombre no cambia.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Evaluar en una spec futura una tool `homes_delete` para el borrado lógico; no forma parte de esta spec.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: una consulta `GET /api/homes` por edición y un `PUT`; sin caché ni estado persistente.
- **Memoria**: la lista de hogares ya se usa en `entities_search`/`homes_create`; no se agregan estructuras nuevas de tamaño relevante.
- **Seguridad**: el endpoint sigue autenticado con API key/sesión; la tool no agrega secretos ni PII.
- **Almacenamiento**: sin migraciones ni archivos nuevos de datos.
- **Disponibilidad**: si el `PUT` falla, no se modifica nada; el error de la API se propaga con código estable.
- **iHost**: sin dependencias nuevas, sin procesos en background y sin escrituras adicionales.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `internal/mcp/tools/entities/homes.go` solo implementa `homesCreate`; no hay handler de edición.
- `internal/api/routes.go` ya expone `PUT /api/homes/{id}`; `HomeHandlers.UpdateHome` decodifica `name`, `address` y `usage`.
- `HomeService.Update` obtiene el hogar actual, normaliza el uso y bloquea el cambio fuera de `rental` si `HomeRentalGuard.HasActiveRental` devuelve `true` (`ErrHomeHasActiveRental`).
- `HomeService.Update` recibe `usage ...string`; como el handler siempre pasa `req.Usage`, un `usage` vacío se normaliza a `rental`. Por eso la tool debe completar el valor actual antes de llamar al `PUT`.
- `entities_search` ya devuelve `usage` del hogar y `resolveHome` ya resuelve por `home_id` o nombre, con `not_found` y `ambiguous_reference`.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Ampliar `homes_create` con un `home_id` opcional | Una sola tool de hogares | Mezcla alta y edición; el esquema y la descripción se vuelven ambiguos | ❌ Rechazada |
| B. `homes_update` con edición parcial y merge en la tool | Permite cambiar solo el estado; no requiere reenviar nombre/dirección; evita el reset a `rental` | Una consulta previa para resolver y completar campos | ✅ Seleccionada |
| C. `homes_update` que exige `name`, `address` y `usage` completos | Refleja el PUT tal cual | El agente debe leer la casa y reenviar todo; un error de copia pisa datos | ❌ Rechazada |
| D. Tool en un paquete nuevo `internal/mcp/tools/homes` | Separación por dominio | `entities` ya concentra hogares, instituciones, bancos y servicios y comparte `resolveHome` | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Merge de campos en la tool, no en el backend**
- **Contexto**: `PUT /api/homes/{id}` reemplaza el registro y un `usage` vacío se normaliza a `rental`.
- **Decisión**: `homes_update` resuelve el hogar, completa `name`, `address` y `usage` con los valores actuales cuando se omiten, y envía el cuerpo completo.
- **Consecuencias**: el backend mantiene su semántica de reemplazo; la tool es segura para ediciones parciales. El costo es una consulta de resolución.

**ADR-002: Sin `dry_run`**
- **Contexto**: Las altas de `entities` no tienen previsualización (SPEC-146); las tools de autos y reparaciones sí.
- **Decisión**: `homes_update` no declara `dry_run`; usa la compuerta `P40LA_MCP_ALLOW_WRITES` y una descripción explícita.
- **Consecuencias**: una sola llamada para cambiar el estado; el agente debe mostrar el cambio y pedir confirmación al usuario antes de invocar la tool. La edición es reversible.

**ADR-003: Conflicto de alquiler activo como error accionable**
- **Contexto**: El backend devuelve 409 `home_has_active_rental` cuando se intenta salir de `rental` con contrato o reserva activa.
- **Decisión**: La tool conserva el código y agrega un hint que explica cómo destrabar el cambio y qué campos sí se pueden editar.
- **Consecuencias**: el agente no reintenta a ciegas ni inventa un estado; el usuario recibe una acción concreta.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
Cliente MCP
   │  homes_update {home_id|home, name?, address?, usage?}
   ▼
internal/mcp/tools/entities/homes.go
   │ resolveHome (GET /api/homes)  ── not_found / ambiguous_reference
   │ merge name/address/usage
   ▼
internal/mcp/api.Client.UpdateHome
   │ PUT /api/homes/{id}  (X-P40LA-Source: mcp)
   ▼
internal/api/home_handlers.go → internal/services/home.go
   │ ErrHomeHasActiveRental → 409 home_has_active_rental
   ▼
internal/storage/home.go → tabla homes (SQLite)
```

### 5.2 Componentes

#### 5.2.1 Tool `homes_update`
- **Responsabilidad**: resolver el hogar, validar el uso, completar campos omitidos y ejecutar el `PUT`.
- **Interfaz**: `HomesUpdateInput` y `homesUpdate` en `internal/mcp/tools/entities/homes.go`; registro en `tools.go`.
- **Dependencias**: `api.API` (`ListHomes`, `UpdateHome`), `toolkit` y la compuerta de escritura.
- **Ubicación**: `internal/mcp/tools/entities/homes.go`.

#### 5.2.2 Cliente HTTP
- **Responsabilidad**: enviar `PUT /api/homes/{id}` y decodificar el hogar.
- **Interfaz**: `UpdateHome(ctx, id, HomeUpdateRequest) (*Home, error)`.
- **Dependencias**: `net/http` y `doJSON` existentes.
- **Ubicación**: `internal/mcp/api/entities.go` + interfaz en `client.go`.

### 5.3 Modelo de datos

No hay cambios de esquema. La tabla `homes` ya tiene `name`, `address`, `usage`, `sort_order`, `deleted_at`, `created_at` y `updated_at`.

```
Home
- id: int64
- name: string (obligatorio)
- address: string (opcional)
- usage: rental | inhabited | unused
- deleted_at: timestamp (borrado lógico)
```

### 5.4 APIs / Contratos

#### Tool: `homes_update`

**Request** (todos los campos excepto uno de `home_id`/`home` son opcionales):
```json
{
  "home_id": 7,
  "usage": "inhabited"
}
```

**Response 200** (hogar actualizado):
```json
{
  "id": 7,
  "name": "Hogar Demo",
  "address": "Calle Demo 123",
  "usage": "inhabited"
}
```

**Errores**:
```json
{ "error": "invalid_value", "message": "usage debe ser rental, inhabited o unused.", "hint": "Valores válidos: rental, inhabited, unused." }
```
```json
{ "error": "home_has_active_rental", "message": "...", "hint": "Cerrá el contrato o la reserva activa antes de cambiar el uso. Podés editar name/address sin cambiar usage." }
```
```json
{ "error": "already_exists", "message": "Ya existe un hogar con ese nombre.", "hint": "Usá otro nombre o editá el existente: id=8, name=\"Casa Demo\"" }
```

#### Endpoint reutilizado: `PUT /api/homes/{id}`

**Request**:
```json
{ "name": "Hogar Demo", "address": "Calle Demo 123", "usage": "inhabited" }
```

**Response 200**: el hogar actualizado.
**Errores**: 400 `invalid_usage` / `invalid_request`, 404 `not_found`, 409 `home_has_active_rental`.

### 5.5 Dependencias

- **Internas**: `internal/mcp/api` (cliente e interfaz), `internal/mcp/tools/entities` (tool y registro), `internal/api/home_handlers.go`, `internal/services/home.go`, `internal/storage/home.go`.
- **Externas**: ninguna nueva.

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). Análisis con `./scripts/check-mcp-impact.sh --files ...` y validación de un subagente (`DELIV-MCP-IMPACT-SUBAGENT`).

### 6.1 Tool nueva

| Tool | Entrypoint | Endpoints | Contrato |
|---|---|---|---|
| `homes_update` | `internal/mcp/tools/entities/homes.go` | `GET /api/homes`, `PUT /api/homes/{id}` | Nueva en `contracts.json`; `home_id`/`home`, `name`, `address`, `usage`; `already_exists` al renombrar a un nombre tomado |

### 6.2 Tools existentes afectadas por superficie

| Tool | Ruta de dependencia | Motivo | Contrato |
|---|---|---|---|
| `homes_create` | tool → `internal/mcp/api/entities.go` → `POST /api/homes` → `internal/api/home_handlers.go` → `internal/services/home.go` → `internal/storage/home.go` → `homes` | Comparte entrypoint (`homes.go`) y registro (`tools.go`) con la tool nueva; su comportamiento no cambia | Sin cambios |
| `entities_search` | tool → `internal/mcp/api/entities.go` → `GET /api/homes` → handler → service → storage → `homes` | Comparte `resolveHome` y el registro `tools.go`; no cambia su contrato | Sin cambios |
| `institutions_create` | tool → `internal/mcp/api/entities.go` → `GET/POST /api/institutions` → handler → service → storage → `institutions` | Comparte el registro `tools.go`; no cambia su contrato | Sin cambios |
| `banks_create` | tool → `internal/mcp/api/entities.go` → `GET/POST /api/banks` → handler → service → storage → `banks` | Comparte el registro `tools.go`; no cambia su contrato | Sin cambios |
| `services_create` | tool → `internal/mcp/api/entities.go` → `POST /api/services` + GETs de hogar/moneda/institución → handler → service → storage → `services` | Comparte el registro `tools.go`; no cambia su contrato | Sin cambios |

### 6.3 Ruta compartida `internal/mcp/api/`

El manifiesto declara `internal/mcp/api/` en `shared.affects_all`. Agregar `UpdateHome` a `client.go` y `entities.go` hace que el analizador reporte **las 41 tools existentes** por recompilación del paquete compartido, aunque su comportamiento no cambia. Se listan para que el gate quede explícito:

`alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bank_accounts_create`, `bank_accounts_search`, `bank_accounts_update`, `banks_create`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `credit_card_statements_get`, `credit_card_statements_list`, `credit_cards_create`, `credit_cards_search`, `credit_cards_update`, `debt_bills_list`, `entities_search`, `home_maintenance_labor_register`, `home_maintenance_list`, `home_maintenance_supplies_register`, `homes_create`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

**Impacto real**: nulo en el comportamiento de esas 41 tools; solo se recompila el paquete `internal/mcp/api`. Las tools con cambio de superficie son `homes_update` (nueva) y las cinco tools de `entities` que comparten `tools.go` (`homes_create`, `entities_search`, `institutions_create`, `banks_create`, `services_create`), sin cambio de contrato en las existentes.

### 6.4 Contrato y tests

- `docs/mcp/tools/contracts.json`: se regenera con `go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts`; se revisa el diff y se espera **solo** el alta de `homes_update`.
- `docs/mcp/tools/impact.json`: se agrega `homes_update` con entrypoint, dependencias, endpoints (`GET /api/homes`, `PUT /api/homes/{id}`) y tests.
- Tests: `go test ./internal/mcp/...`, `go test ./internal/api -run TestMCPBackendContract`, `go test ./internal/mcp/tools/entities` y `go test ./internal/mcp/api -run TestClientEntityCalls`.
- No se esperan cambios en descripciones, esquemas ni anotaciones de tools existentes.

Validación subagente: `explore` — sesión `ses_def199f01ffe3qp9V3W9BDaBIF` — veredicto: las 41 tools existentes quedan reportadas por `shared.affects_all` al tocar `internal/mcp/api/`; hay impacto directo de superficie en `homes_create`, `entities_search`, `institutions_create`, `banks_create` y `services_create` por compartir `tools.go`, sin cambios de contrato; la única tool nueva es `homes_update` y el golden solo debe ganar esa entrada (42 tools).
**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [x] CA-ARCH-MCP-TOOLS-MAP: `homes_update` se registra con nombre literal y tiene entrada en `docs/mcp/tools/impact.json`.
- [x] CA-DELIV-MCP-IMPACT: `./scripts/check-mcp-impact.sh --staged --gate` pasa y la sección 6 menciona todas las tools reportadas.
- [x] CA-DELIV-MCP-IMPACT-SUBAGENT: la sección 6 registra agente, sesión y veredicto del subagente.
- [x] CA-DELIV-MCP-DOCS: `docs/mcp/tools/entities.md`, `docs/mcp/tools/README.md`, `docs/mcp/README.md`, `docs/mcp/security.md` y `docs/mcp/tools/impact.md` documentan `homes_update` en el mismo commit.
- [x] CA-TEST-MCP-CONTRACT: `TestToolContracts` y `TestMCPBackendContract` pasan; el golden solo agrega `homes_update`.
- [x] CA-FUNC-001: editar solo `usage` de un hogar existente devuelve el hogar con el nuevo uso y conserva `name` y `address`.
- [x] CA-FUNC-002: `address: ""` limpia la dirección; omitir `address` la conserva.
- [x] CA-FUNC-003: `usage` fuera de `rental|inhabited|unused` devuelve `invalid_value` y no llama al `PUT`.
- [x] CA-FUNC-004: un `home_id` inexistente devuelve `not_found` con opciones `id=nombre`; un nombre ambiguo devuelve `ambiguous_reference`.
- [x] CA-FUNC-005: con la compuerta cerrada devuelve `writes_disabled` y no resuelve ni escribe.
- [x] CA-FUNC-006: un 409 `home_has_active_rental` se propaga con hint accionable y no se reintenta.
- [x] CA-FUNC-007: renombrar a un nombre que ya tiene otro hogar devuelve `already_exists` y no llama al `PUT`; el mismo nombre actual no se considera duplicado.
- [x] CA-UX-PREREQ: el hogar se valida antes de escribir y el error indica cómo corregir la referencia.
- [x] CA-SEC-NO-PII: la respuesta solo contiene los campos del hogar; no se agregan datos personales.
- [x] CA-SEC-SYNTHETIC: ejemplos y tests usan `Hogar Demo`, `Calle Demo 123` y datos `demo-*`.
- [x] CA-ARCH-IHOST: sin dependencias nuevas, sin caché y sin procesos en background.
- [x] CA-TEST-LOCAL: `go build ./...` y `go test ./...` pasan.
- [x] CA-DELIV-WORKTREE: la spec vive en `../p40la-ihost-spec-156` y `./scripts/check-worktree.sh` pasa.
- [x] CA-DELIV-CONFIRM: merge y push autorizados explícitamente por el usuario, sin `--force`.
- [x] CA-DELIV-RELEASE: merge, push, tracker, label y cierre del issue en el mismo ciclo.

### 7.2 No funcionales

- [x] CA-NF-001: una edición hace como máximo una consulta de resolución y un `PUT`; no hay escrituras adicionales.
- [x] CA-NF-002: el binario MCP no incorpora dependencias nuevas.

### 7.3 Testing

- **Unit tests**: merge de campos, validación de uso, resolución por ID/nombre, compuerta y conflicto.
- **Integration tests**: `TestMCPBackendContract` con `httptest` y SQLite en memoria sobre `PUT /api/homes/{id}`.
- **Protocol tests**: `tools/list` incluye `homes_update` con esquema y anotaciones esperadas.
- **E2E**: no aplica; el cambio no toca la UI.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Cliente `UpdateHome` e interfaz `api.API` | 0.5 h | Ninguna |
| 2 | Tool `homes_update` y registro | 1 h | Fase 1 |
| 3 | Tests unitarios, de protocolo y contrato backend | 1 h | Fase 2 |
| 4 | Manifiesto, contrato congelado y documentación MCP | 1 h | Fases 2-3 |
| 5 | Build, verificación y release | 0.5 h | Fases 1-4 |

### 8.2 Milestones

1. **MVP**: `homes_update` funciona por `home_id` y cambia `usage`.
2. **V1.0**: resolución por nombre, edición parcial, errores accionables, contrato y docs actualizados.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El agente omite `usage` y el backend lo revierte a `rental` | Media | Alto | La tool hace merge y siempre envía el uso actual si no se indica uno nuevo |
| El nombre del hogar es ambiguo | Baja | Medio | `ambiguous_reference` con candidatos y `home_id` como solución |
| Renombrar a un nombre existente duplica la referencia | Baja | Medio | La tool rechaza el cambio con `already_exists` y el `id` existente |
| Se intenta cambiar el uso con alquiler activo | Media | Medio | 409 traducido con hint; el agente explica y edita solo nombre/dirección si aplica |
| El contrato congelado cambia en tools existentes | Baja | Alto | Se revisa el diff del golden; solo debe agregarse `homes_update` |
| Fakes de `api.API` incompletos | Media | Bajo | Se actualizan todos los fakes al agregar `UpdateHome`; el compilador lo detecta |

---

## 10. Notas y Referencias

- Reglas aplicadas: `docs/rules/00-index.md`, `10-architecture.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Precedentes: SPEC-146 (`homes_create` y `entities_search`), SPEC-143 (uso de casa y guard de alquiler), SPEC-153 (manifiesto y gate de impacto MCP), SPEC-155 (documentación y contrato de tools).
- Código relacionado: `internal/mcp/tools/entities/homes.go`, `internal/mcp/api/entities.go`, `internal/api/home_handlers.go`, `internal/services/home.go`, `internal/storage/home.go`.

---

## Historial de cambios

| Fecha | Estado | Cambio |
|---|---|---|
| 2026-11-24 | draft | Creación de la spec: tool `homes_update` con edición parcial, validación de uso y conflicto de alquiler activo |
| 2026-11-24 | pending_release | Implementación completa: tool, cliente, tests, manifiesto, contrato y docs; `go build ./...`, `go test ./...`, gate MCP y contrato backend verdes |
| 2026-11-24 | released | Release: merge `1a5c025d`; build de Go y frontend verdes; issue #159 cerrado |
