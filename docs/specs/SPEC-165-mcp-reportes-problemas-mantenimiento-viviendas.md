---
title: "MCP: reportes de problemas y mantenimiento en viviendas"
id: "SPEC-165"
status: "pending_release"
author: "opencode"
created: "2026-11-27"
updated: "2026-11-27"
github_issue: 167
---

# MCP: reportes de problemas y mantenimiento en viviendas

**ID**: SPEC-165
**Estado**: pending_release
**Autor**: opencode
**Creado**: 2026-11-27
**Actualizado**: 2026-11-27

---

## 1. Resumen Ejecutivo

SPEC-164 agregó a la app web los **reportes de problemas y mantenimiento por vivienda** (`home.workrequest`): un reporte nace en estado `reported`, sin insumos ni mano de obra, con evidencia en fotos/video, y puede convertirse en una reparación de `home.maintenance`, descartarse o reactivarse. El módulo HTTP ya expone `GET|POST /api/homes/{id}/workrequests`, `GET|PUT|DELETE .../{work_request_id}`, `.../convert` y `.../status`.

El servidor MCP no tiene tools para ese módulo. Un agente que opera p40la-ihost puede consultar y editar reparaciones (`home_maintenance_*`) pero no puede ver los reportes pendientes, reportar una falla nueva, convertirla en reparación ni descartarla. Esta spec agrega seis tools sobre la API existente, sin tocar el backend ni el esquema.

El resultado esperado es un toolset `home_work_request_*` de dos tools de lectura y cuatro de escritura, con `dry_run` por defecto y la compuerta de escritura del MCP. No hay migraciones, dependencias ni cambios en la app web; el release publica una nueva imagen `p40la-mcp`.

---

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-MCP-TOOLS-MAP` | Las seis tools se declaran en `docs/mcp/tools/impact.json` y en `contracts.json`. | `CA-MCP-WR-MAP` |
| `DELIV-MCP-DOCS` | Se crea `docs/mcp/tools/home-work-requests.md` y se actualizan los índices de `docs/mcp/`. | `CA-MCP-WR-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La documentación la redacta un subagente y se integra en el mismo commit. | `CA-MCP-WR-DOCS-SUBAGENT` |
| `DELIV-MCP-IMPACT` | Sección §6 completa y `./scripts/check-mcp-impact.sh --staged --gate` verde. | `CA-MCP-WR-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis delegado a un subagente `explore`; veredicto citado en §6. | `CA-MCP-WR-IMPACT-SUBAGENT` |
| `TEST-MCP-CONTRACT` | `TestToolContracts`, `TestMCPBackendContract` y tests unitarios del toolset y del cliente. | `CA-MCP-WR-CONTRACT` |
| `ARCH-STACK` | Go + `net/http` y SDK MCP existentes; cero dependencias nuevas. | `CA-MCP-WR-STACK` |
| `ARCH-LAYERS` | Las tools validan forma; el backend sigue siendo la fuente de verdad del dominio. | `CA-MCP-WR-LAYERS` |
| `ARCH-IHOST` | Sin acceso a SQLite, sin archivos y sin cron; solo llamadas HTTP al server. | `CA-MCP-WR-IHOST` |
| `DATA-MIGRATIONS` | `N/A`: no hay migraciones. | — |
| `UI-*` / `UX-*` | `N/A`: no hay cambios de UI. | — |
| `SEC-NO-PII` / `SEC-SYNTHETIC` | Ejemplos y tests con `Casa Demo`, `demo-*`, sin datos reales. | `CA-MCP-WR-SYNTHETIC` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-165`, rama `feature/SPEC-165`. | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge, release y publicación de la imagen ordenados explícitamente por el usuario. | `CA-DELIV-CONFIRM` |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Listado.** `home_work_request_list` lista los reportes de una vivienda con filtros opcionales `status` (`reported`/`converted`/`dismissed`) y `category` (`repair`/`improvement`/`maintenance`/`other`). Devuelve `pending_count`, `truncated` y los reportes con `attachment_count`, ordenados por `reported_at DESC, id DESC`.
2. **REQ-002 — Detalle.** `home_work_request_get` devuelve un reporte por vivienda con sus metadatos de evidencia (no el binario) y el vínculo `maintenance_id`/`converted_at`.
3. **REQ-003 — Alta.** `home_work_request_create` crea un reporte con `title` obligatorio, `description`, `category`, `priority` y `reported_at` (default: hoy, no futura). El backend fija `status=reported` y el reporte nace sin costos.
4. **REQ-004 — Edición.** `home_work_request_update` edita parcialmente `title`, `description`, `category`, `priority` y `reported_at`: los campos omitidos conservan su valor y el estado no se modifica. Devuelve `changed_fields` y `status=unchanged` si no hay cambios efectivos.
5. **REQ-005 — Conversión.** `home_work_request_convert` convierte un reporte `reported` en una reparación vacía, con `kind` (default `corrective`) y `area` (default `other`). Devuelve `maintenance_id` y el reporte `converted`. Si ya fue convertido, devuelve `already_converted`; un reporte descartado no se convierte.
6. **REQ-006 — Descarte y reactivación.** `home_work_request_status` cambia `reported → dismissed` o `dismissed → reported`; no admite `converted` ni otros estados.
7. **REQ-007 — Compuerta de escritura.** Las cuatro tools de escritura respetan `dry_run` (`true` por defecto) y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). Las de lectura no dependen de la compuerta.
8. **REQ-008 — Contrato y documentación.** Las seis tools se registran, se documentan, se declaran en el manifiesto de impacto y se congelan en `contracts.json`; `TestMCPBackendContract` cubre la cadena real contra el módulo `home.workrequest`.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009 — Salida accionable.** Las tools de escritura devuelven `status` (`planned`/`created`/`updated`/`unchanged`/`converted`), `dry_run`, `changed_fields` cuando aplica y el registro completo; los `dry_run` no hacen POST/PUT.
2. **REQ-010 — Errores estables.** Se propagan `home_not_found`, `work_request_not_found`, `already_converted`, `invalid_status`, `writes_disabled` y `auth_failed` con mensajes y hints accionables.

### 3.3 Requerimientos No Funcionales

- **Dependencias**: cero nuevas en `go.mod`; se reutilizan `internal/mcp/api`, `toolkit` y el SDK MCP.
- **iHost**: el MCP sigue siendo un cliente HTTP; no abre SQLite ni el volumen de datos.
- **Rendimiento**: una llamada HTTP por lectura; las escrituras hacen como máximo un GET previo y una escritura.
- **Alcance**: no se modifica la app web, el backend ni las migraciones; solo el MCP, sus tests y su documentación.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- El módulo `home.workrequest` (SPEC-164) expone endpoints autenticados bajo `/api/homes/{id}/workrequests` y usa `status`: `reported`, `converted`, `dismissed`.
- Las tools existentes de `home.maintenance` (`internal/mcp/tools/homemaint/`) son el patrón canónico: `api.HomeMaintenanceAPI`, toolset con `writeMode`, `dry_run` y `toolkit.APIError`.
- El catálogo MCP se monta en `internal/mcp/tools/all/register.go`; los contratos se congelan con `TestToolContracts` y el manifiesto `impact.json` se valida con `check-mcp-impact.sh`.
- `FullAPI` en `internal/mcp/api/autos.go` es la unión de contratos que usa el servidor MCP; agregar un contrato nuevo exige actualizar el stub de `contract_test.go`.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Un toolset nuevo por dominio (`workrequest`) | Aísla el dominio, sigue el patrón de `homemaint` | Un paquete más | ✅ Seleccionada |
| Meter las tools dentro de `homemaint` | Menos paquetes | Mezcla reportes con reparaciones y confunde el catálogo | ❌ Rechazada |
| Tool única `home_work_request_manage` con `action` | Menos tools | Pierde esquemas tipados y validación por acción; no sigue la convención del proyecto | ❌ Rechazada |
| Incluir subida de evidencia por MCP | Cobertura total | Requiere binarios/base64 y multipart; el MCP no maneja archivos y la web ya lo cubre | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Seis tools tipadas, no una tool genérica.**
- **Contexto**: el catálogo usa un verbo por tool y esquemas de entrada congelados.
- **Decisión**: `home_work_request_list`, `_get`, `_create`, `_update`, `_convert` y `_status`.
- **Consecuencias**: contratos claros y validación por tool; el agente descubre capacidades por nombre.

**ADR-002 — La evidencia se consulta, no se sube por MCP.**
- **Contexto**: las tools MCP hablan JSON; la evidencia es multipart y puede pesar 25 MB.
- **Decisión**: `home_work_request_get` devuelve metadatos de adjuntos y `home_work_request_list` el conteo; la carga/borrado de archivos queda en la UI.
- **Consecuencias**: sin base64 ni límites de contexto; el agente puede pedir al usuario que adjunte desde la web.

**ADR-003 — `status` separado de `update`.**
- **Contexto**: el backend no permite editar `status` por PUT; los cambios de estado son acciones de negocio.
- **Decisión**: `home_work_request_status` con `status` `reported`/`dismissed`; `convert` es una acción aparte.
- **Consecuencias**: no se puede dejar un reporte en un estado inválido por edición.

---

## 5. Diseño Técnico

### 5.1 Diagrama

```
[Agente MCP] -> [tool: home_work_request_*] -> [api.Client] -> HTTP /api/homes/{id}/workrequests...
                                                          |
                                                [módulo home.workrequest]
                                                          |
                                            [home_maintenances] (solo al convertir)
```

### 5.2 Componentes

#### 5.2.1 Cliente API — `internal/mcp/api/home_work_request.go`
- `HomeWorkRequestAPI` con `List`, `Get`, `Create`, `Update`, `Convert`, `SetStatus`.
- DTOs espejo del JSON del backend: `HomeWorkRequest`, `HomeWorkRequestAttachment`, `HomeWorkRequestList`, request de alta/edición, conversión y estado.
- Escrituras con `sourceHeaders()` (`X-P40LA-Source: mcp`).
- `FullAPI` incorpora `HomeWorkRequestAPI`.

#### 5.2.2 Toolset — `internal/mcp/tools/workrequest/`
- `workrequest.go`: `Toolset`, `New`, `Register`, `today`, `writeMode`.
- `list.go`, `get.go`, `create.go`, `update.go`, `convert.go`, `status.go`.
- Validación de enums y fechas; merge parcial en `update`; GET previo en `update`, `convert` y `status`.
- Registro en `internal/mcp/tools/all/register.go`.

### 5.3 Contratos de tools

#### `home_work_request_list`
Entrada: `home_id` (req), `status?`, `category?`.
Salida: `home_id`, `count`, `pending_count`, `truncated`, `work_requests[]`.

#### `home_work_request_get`
Entrada: `home_id`, `work_request_id`.
Salida: `work_request` con `attachments[]` (metadatos).

#### `home_work_request_create`
Entrada: `home_id`, `title`, `description?`, `category?`, `priority?`, `reported_at?`, `dry_run?`.
Salida: `status` (`planned`/`created`), `dry_run`, `work_request`.

#### `home_work_request_update`
Entrada: `home_id`, `work_request_id`, `title?`, `description?`, `category?`, `priority?`, `reported_at?`, `dry_run?`.
Salida: `status` (`planned`/`updated`/`unchanged`), `changed_fields[]`, `work_request`.

#### `home_work_request_convert`
Entrada: `home_id`, `work_request_id`, `kind?`, `area?`, `dry_run?`.
Salida: `status` (`planned`/`converted`), `maintenance_id`, `work_request`.

#### `home_work_request_status`
Entrada: `home_id`, `work_request_id`, `status` (`reported`/`dismissed`), `dry_run?`.
Salida: `status` (`planned`/`updated`), `work_request`.

### 5.4 Dependencias

- **Internas**: `internal/mcp/api` (nuevo contrato), `internal/mcp/tools/toolkit`, `internal/mcp/tools/all`.
- **Backend**: endpoints de `home.workrequest` de SPEC-164, sin cambios.
- **Externas**: ninguna nueva.

---

## 6. Impacto en tools MCP

### 6.1 Veredicto

**Ninguna tool existente se ve afectada funcionalmente.** Se agregan seis tools nuevas sobre endpoints ya existentes de SPEC-164. No se modifica `internal/mcp/tools/homemaint/`, `internal/modules/home/**` ni la app web. El único cambio compartido es `api.FullAPI` (unión de contratos) y el stub de `contract_test.go`, que se actualizan para compilar.

### 6.2 Tools nuevas

`home_work_request_list`, `home_work_request_get`, `home_work_request_create`, `home_work_request_update`, `home_work_request_convert`, `home_work_request_status`.

### 6.3 Tools existentes afectadas

Ninguna con cambio funcional, de contrato o de esquema. El guardrail marca las **53 tools existentes** solo por recompilación compartida (`internal/mcp/api/` y `internal/mcp/tools/all/` están en `shared.affects_all`); el impacto real es nulo. Se listan para que la declaración sea explícita:

`bills_list`, `bills_generate`, `bills_pay`, `homes_create`, `homes_update`, `institutions_create`, `banks_create`, `services_create`, `entities_search`, `autos_list`, `autos_create`, `autos_update`, `autos_maintenance_register`, `autos_inspection_register`, `autos_insurance_register`, `bank_accounts_search`, `bank_accounts_create`, `bank_accounts_update`, `credit_cards_search`, `credit_cards_create`, `credit_cards_update`, `pension_month_summary`, `pension_catalogs_list`, `rental_leases_list`, `rental_lease_get`, `rental_home_status`, `rental_portfolio_summary`, `debt_bills_list`, `credit_card_statements_list`, `credit_card_statements_get`, `home_maintenance_list`, `home_maintenance_labor_register`, `home_maintenance_supplies_register`, `home_maintenance_create`, `home_maintenance_item_move`, `home_maintenance_get`, `home_maintenance_update`, `home_maintenance_item_update`, `home_maintenance_item_delete`, `autos_inspection_alerts`, `bills_by_month`, `bills_summary`, `bills_projection`, `services_get`, `automation_accounts_list`, `automation_runs_list`, `alerts_list`, `settings_status`, `budget_categories_list`, `budget_accounts_list`, `budget_transactions_create`, `budget_transactions_list`, `budget_month_view`.

`home_work_request_convert` crea filas en `home_maintenances`; las tools `home_maintenance_*` verán esa reparación nueva, pero es un efecto de datos esperado, no un cambio de contrato ni de comportamiento.

### 6.4 Manifiesto y contrato

- `docs/mcp/tools/impact.json`: seis entradas nuevas con entrypoint, dependencias, endpoints y tests.
- `docs/mcp/tools/contracts.json`: regenerado con `TestToolContracts -update-contracts`.
- `TestMCPBackendContract`: cubre el módulo `home.workrequest` real (crear, listar, obtener, editar, convertir, estado).

**Validación subagente:** explore — sesión `ses_ddfeb1507ffewd7G8cAXhjBZyc` — veredicto: **ninguna tool existente afectada funcionalmente**; las 53 aparecen solo por recompilación compartida. El fake `contractAPI` de `internal/mcp/tools/all/contract_test.go` es el único que implementa `FullAPI` y recibió los 6 stubs nuevos; no hay que tocar `api.API` para no romper los fakes de facturas, entidades y banca. Se debe registrar el módulo `home.workrequest` en el test backend y regenerar `contracts.json`.

---

## 7. Criterios de Aceptación

- [ ] `CA-MCP-WR-001`: `home_work_request_list` filtra por estado/categoría y devuelve `pending_count`, `truncated` y `attachment_count`.
- [ ] `CA-MCP-WR-002`: `home_work_request_get` devuelve el detalle con metadatos de evidencia y el vínculo a la reparación.
- [ ] `CA-MCP-WR-003`: `home_work_request_create` en `dry_run` no escribe y en modo escritura crea `reported` sin costos.
- [ ] `CA-MCP-WR-004`: `home_work_request_update` hace merge parcial y devuelve `unchanged` cuando no hay cambios.
- [ ] `CA-MCP-WR-005`: `home_work_request_convert` crea la reparación vacía, enlaza `maintenance_id` y rechaza una segunda conversión con `already_converted`.
- [ ] `CA-MCP-WR-006`: `home_work_request_status` descarta y reactiva; rechaza `converted`.
- [ ] `CA-MCP-WR-007`: las escrituras sin compuerta devuelven `writes_disabled`; `dry_run` sigue disponible.
- [ ] `CA-MCP-WR-008`: `TestToolContracts`, `TestMCPBackendContract` y `go test ./internal/mcp/...` verdes.
- [ ] `CA-MCP-WR-DOCS`: `docs/mcp/tools/home-work-requests.md` y los índices actualizados (catálogo 53 → 59).
- [ ] `CA-MCP-WR-IMPACT`: `check-mcp-impact.sh --staged --gate` verde.
- [ ] `CA-MCP-WR-STACK`: sin dependencias nuevas ni cambios en `go.mod`.
- [ ] `CA-MCP-WR-SYNTHETIC`: tests con `Casa Demo`, `demo-*`, sin PII.
- [ ] `CA-MCP-WR-RELEASE`: `mcp-v0.6.4` publicado multi-arch; issue cerrado.

---

## 8. Plan de Implementación

| Fase | Descripción |
|---|---|
| 1 | Cliente API y DTOs (`home_work_request.go`, `FullAPI`) |
| 2 | Toolset `workrequest` (6 tools) y registro |
| 3 | Tests unitarios, contrato y backend real |
| 4 | Manifiesto, `contracts.json` y documentación (subagente) |
| 5 | Release `mcp-v0.6.4` y verificación de imagen |

---

## 9. Riesgos y Mitigaciones

| Riesgo | Mitigación |
|---|---|
| Duplicar reglas de dominio en el toolset | Validar solo forma; el backend decide (fechas, límites, estados) |
| Cambiar el contrato sin regenerar el golden | `TestToolContracts` falla y exige `-update-contracts` |
| Escrituras accidentales | `dry_run` default true + compuerta de escritura |
| Cargar evidencia binaria en el contexto | Solo metadatos; la web sigue siendo el canal de archivos |

---

## 10. Notas y Referencias

- SPEC-164: módulo `home.workrequest` (endpoints y semántica de estados).
- SPEC-153: manifiesto de impacto y gate MCP.
- SPEC-159: patrón de tools de detalle/edición de reparaciones.
- `docs/mcp/tools/home-maintenance.md`: referencia de estilo y documentación.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-27 | opencode | Creación inicial para agregar tools MCP de reportes de vivienda |
| 2026-11-27 | opencode | Implementación: cliente `HomeWorkRequestAPI`, toolset `workrequest` con 6 tools, registro, tests, manifiesto y `contracts.json`; documentación delegada a subagente |
| 2026-11-27 | opencode | Verificación local: `go build ./...`, `go test ./...`, `check-mcp-docs` y `check-mcp-impact --staged --gate` verdes; preparada para release `mcp-v0.6.4` |
