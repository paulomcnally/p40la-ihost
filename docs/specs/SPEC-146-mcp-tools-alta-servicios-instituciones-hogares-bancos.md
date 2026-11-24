---
title: "MCP de entidades: tools de alta para servicios, instituciones, hogares y bancos"
id: "SPEC-146"
status: "in_progress"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-24"
github_issue: 149
---

# MCP de entidades: tools de alta para servicios, instituciones, hogares y bancos

**ID**: SPEC-146  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

SPEC-145 dejó un servidor MCP (`p40la-mcp`) con tools de facturas (`bills_list`, `bills_generate`, `bills_pay`) y una API key global. Hoy un agente puede operar facturas, pero **no puede dar de alta las entidades base** que esas facturas necesitan: hogares, instituciones, bancos y servicios. Para preparar un escenario completo debe salir del MCP y usar la UI o `curl`.

Esta spec agrega cuatro tools de alta al MCP existente: `homes_create`, `institutions_create`, `banks_create` y `services_create`. Las tools son clientes HTTP de los endpoints `POST` que ya existen y ya validan reglas de negocio; no agregan SQL, migraciones ni lógica de dominio al MCP. Para que sean usables por un agente sin conocer IDs de memoria, resuelven referencias por nombre o código (`home`, `currency`, `category`) y fallan con `hint` cuando hay ambigüedad o no existe el prerrequisito. Las escrituras quedan detrás de `P40LA_MCP_ALLOW_WRITES`, igual que en SPEC-145, y se bloquean duplicados obvios por nombre antes de llamar al backend.

El cambio es de bajo impacto para el iHost: no toca el servidor web ni la DB, no agrega dependencias (reutiliza `modelcontextprotocol/go-sdk`) y el proceso MCP corre fuera del contenedor.

---

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Se reutiliza Go + `modelcontextprotocol/go-sdk`; no se agregan dependencias | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Tools en `internal/mcp/tools/entities/`; DTOs y cliente en `internal/mcp/api/` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Las tools solo parsean entrada y llaman al cliente HTTP; validación de dominio queda en `internal/services` | `CA-ARCH-LAYERS` |
| `ARCH-GO` | `gofmt`, errores explícitos, `log/slog` a stderr, nombres en inglés | `CA-ARCH-GO` |
| `ARCH-IHOST` | Sin migraciones ni cambios en el server; sin dependencias nuevas; proceso MCP liviano | `CA-ARCH-IHOST` |
| `DATA-MIGRATIONS` | N/A: no hay migraciones; se usan endpoints existentes | — |
| `DATA-NO-DROP` / `DATA-PRODCOPY` | La validación manual se corre contra copia de producción en `/tmp`; nunca contra `data/app.db` ni producción | `CA-DATA-PRODCOPY` |
| `SEC-NO-PII` | Spec, issues y docs sin datos reales; solo ejemplos `demo-*` | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Todos los ejemplos usan `Hogar Demo`, `Banco Demo`, `Proveedor Demo`, `demo-account-001` | `CA-SEC-SYNTHETIC` |
| `SEC-HOOKS` / `SEC-COMMIT` | Commits con `SPEC-146:` y revisión de `git diff --cached` antes de commitear | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` antes de commit; tests de tools con API falsa | `CA-TEST-LOCAL` |
| `TEST-SERVER` | Si se prueba contra el server local, `./scripts/check-server.sh` antes de avisar | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | No se marca `pending_release` sin pruebas manuales confirmadas por el usuario | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo exclusivo en `../p40la-ihost-spec-146`, rama `feature/SPEC-146` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita; sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-DOCS` | El mismo commit actualiza `docs/mcp/` (`tools/entities.md`, catálogo, desarrollo) | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente en paralelo | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `UI-*` / `UX-*` | N/A: la spec no toca frontend, rutas ni pantallas | — |

> Reglas fuente: `docs/rules/10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — `homes_create`**: crea un hogar (`POST /api/homes`) con `name` obligatorio y `address`/`usage` opcionales; devuelve el hogar creado con `id`.
2. **REQ-002 — `institutions_create`**: crea una institución (`POST /api/institutions`) con `name` obligatorio y `category_id` o `category` opcional; devuelve la institución creada con `id`.
3. **REQ-003 — `banks_create`**: crea un banco (`POST /api/banks`) con `name` obligatorio y `short_code`, `country` (ISO-2), `notes`, `active` opcionales; devuelve el banco creado con `id`.
4. **REQ-004 — `services_create`**: crea un servicio (`POST /api/services`) con `name`, hogar y moneda obligatorios; acepta `home_id`/`home` y `currency_id`/`currency`; opcionales `institution`, `frequency`, `suggested_amount`, `icon_key`, `billing_type`, `billing_day`, `active`, `auto_generate`, `is_recurring`, `start_date`, `end_date`.
5. **REQ-005 — Registro**: las cuatro tools se montan en `cmd/mcp-server` vía `internal/mcp/tools/all/register.go` y aparecen en `tools/list`.
6. **REQ-006 — Gate de escritura**: con `P40LA_MCP_ALLOW_WRITES=false` (default), las tools responden `writes_disabled` y no llaman al backend.
7. **REQ-007 — Duplicados**: antes de crear, se busca una entidad con el mismo nombre (sin distinguir mayúsculas ni acentos) en el mismo ámbito (`home` para servicios; global para hogar, institución y banco). Si existe, se responde `already_exists` con `id` y datos existentes, sin escribir.
8. **REQ-008 — Resolución de referencias**: `home`, `currency` y `category` se resuelven por nombre/código contra `GET /api/{homes,currencies,institution-categories}`. Si no existe → `not_found` con `hint` de opciones disponibles; si hay más de una coincidencia → `ambiguous_reference` con la lista de candidatos.
9. **REQ-009 — Documentación**: `docs/mcp/tools/entities.md` documenta las cuatro tools con entradas, salidas y ejemplos sintéticos; se actualizan catálogo y guía de desarrollo.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-010 — Hints de error**: los errores `not_found`/`ambiguous_reference` incluyen hasta 10 opciones `{id, name}` para que el agente corrija sin adivinar.
2. **REQ-011 — Validación temprana**: `usage`, `frequency`, `billing_type`, `billing_day`, `country` y `suggested_amount` se validan en la tool antes del POST, con mensajes en español.
3. **REQ-012 — `institution_id`**: `services_create` acepta `institution_id` opcional además del texto libre `institution`.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013 — `categories_list`**: tool de solo lectura para descubrir categorías de institución. Se difiere a otra spec; esta spec solo resuelve `category` por nombre/clave.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: cada tool hace como máximo 3 llamadas HTTP (lista de referencia + verificación de duplicado + POST); respuesta < 2 s en LAN.
- **Seguridad**: API key global obligatoria; escrituras deshabilitadas por defecto; sin logs de payloads con datos de usuario.
- **Almacenamiento**: cero migraciones y cero archivos nuevos de datos.
- **iHost**: sin impacto en el proceso del server; el MCP corre en el equipo del agente y usa el mismo binario liviano de SPEC-145.
- **Compatibilidad**: no rompe las tools de facturas ni sus contratos.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- SPEC-145 (`docs/specs/SPEC-145-*.md`) y `docs/mcp/development.md`: patrón `Toolset` + `Register`, cliente HTTP con API key, `registry` y `all.Register`.
- Endpoints existentes y sus validaciones:
  - `POST /api/homes` → `HomeService.Create`: `name` obligatorio; `usage` ∈ `rental|inhabited|unused` (default `rental`).
  - `POST /api/institutions` → `InstitutionService.Create`: `name` obligatorio; `category_id` opcional.
  - `POST /api/banks` → `BankService.CreateBank` + `prepareBank`: `name` obligatorio; `country` ISO-2 opcional; límites de longitud.
  - `POST /api/services` → `ServiceService.Create` + `validate`: `name`, `home_id`, `currency_id`, `frequency ∈ monthly|yearly`, `billing_type ∈ fixed|variable`, `icon_key` de la allowlist, `suggested_amount ≥ 0`; si `active` genera la factura del período actual.
- `internal/mcp/config`: `AllowWrites` solo es `true` con `P40LA_MCP_ALLOW_WRITES=true`.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Una tool por entidad en un paquete `entities` | Helpers y resolución compartidos; 4 tools sin duplicar código | Paquete con 4 dominios | ✅ Seleccionada |
| B. Cuatro paquetes (`tools/services`, `tools/homes`, …) | Espeja `bills` | Duplica helpers de resultado y resolución ×4 | ❌ Rechazada |
| C. Tools que exigen solo IDs | Menos código | El agente no puede descubrir IDs; fricción y errores | ❌ Rechazada |
| D. `dry_run` en las altas | Consistente con `bills_*` | Requiere `dry_run` en 4 handlers/servicios; duplica lógica de validación | ❌ Rechazada |
| E. Duplicar helpers de `bills` | No toca SPEC-145 | Duplicación y divergencia futura | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Alta inmediata con gate de entorno, sin `dry_run`**
- **Contexto**: `bills_generate`/`bills_pay` declaran `dry_run` porque son operaciones financieras masivas. Las altas de entidades son aditivas y ya requieren `P40LA_MCP_ALLOW_WRITES=true`.
- **Decisión**: las tools de alta escriben directo cuando el gate está habilitado; no exponen `dry_run`. La previsualización se logra con la verificación de duplicados y la resolución de referencias, que no escriben.
- **Consecuencias**: menos superficie y cero cambios en el backend. Se documenta que el default es no escribir y que la tool devuelve la entidad creada para confirmación.

**ADR-002 — Resolución por nombre en el MCP, reglas en el backend**
- **Contexto**: un agente no conoce IDs; el backend ya valida existencia y reglas.
- **Decisión**: el MCP resuelve `home`, `currency` y `category` por nombre/código, y traduce los errores del backend. No replica allowlists ni validaciones de dominio salvo las baratas y deterministas (`usage`, `frequency`, `billing_type`, `billing_day`, `country`).
- **Consecuencias**: mejor usabilidad; si el backend agrega valores válidos, la tool debe actualizarse (test de contrato lo detecta).

**ADR-003 — Helpers compartidos en `internal/mcp/tools/toolkit`**
- **Contexto**: `toolError`, `jsonResult` y `apiError` viven en el paquete `bills` y las nuevas tools los necesitan.
- **Decisión**: extraerlos a `toolkit` y actualizar `bills` para usarlos; sin cambios de comportamiento.
- **Consecuencias**: evita duplicación; el diff toca `bills` (cubierto por sus tests existentes).

**ADR-004 — Detección de duplicados previa**
- **Contexto**: un agente puede reintentar una tool y crear duplicados.
- **Decisión**: comparar por nombre normalizado (trim, minúsculas, sin acentos) contra la lista del backend; para servicios, dentro del mismo `home_id`.
- **Consecuencias**: una llamada GET extra; falso positivo posible si el usuario realmente quiere un duplicado → se puede forzar con `allow_duplicate=true` (P1).

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
Agente / cliente MCP
        │ stdio (MCP)
        ▼
cmd/mcp-server ──► internal/mcp/tools/entities
        │                 │
        │                 ├── resolve.go (nombre → id, duplicados)
        │                 ▼
        │          internal/mcp/api (HTTP + X-API-Key)
        │                 │
        ▼                 ▼
   stderr (logs)   p40la-ihost :8088
                    └── internal/api → internal/services → SQLite
```

### 5.2 Componentes

#### 5.2.1 `internal/mcp/tools/entities`
- **Responsabilidad**: schemas de entrada, validación barata, resolución de referencias, detección de duplicados y mapeo de errores.
- **Interfaz**: `Register(server, client, cfg)` monta las cuatro tools.
- **Dependencias**: `internal/mcp/api`, `internal/mcp/config`, `internal/mcp/tools/toolkit`, `modelcontextprotocol/go-sdk/mcp`.
- **Ubicación**: `internal/mcp/tools/entities/{tools,services,institutions,homes,banks,resolve}.go`.

#### 5.2.2 `internal/mcp/tools/toolkit`
- **Responsabilidad**: `JSONResult`, `ToolError` y `APIError` compartidos.
- **Ubicación**: `internal/mcp/tools/toolkit/toolkit.go`.

#### 5.2.3 `internal/mcp/api`
- **Responsabilidad**: DTOs de entidades y métodos `Create*`/`List*`.
- **Ubicación**: `internal/mcp/api/entities.go` (tipos) y `entities_client.go` (métodos).

#### 5.2.4 `internal/mcp/tools/all/register.go`
- **Responsabilidad**: registrar `entities.Register` junto a `bills.Register`.

### 5.3 Modelo de datos

No hay cambios de esquema. DTOs nuevos (subconjunto de la respuesta del backend):

```
Home:         id, name, address, usage
Institution:  id, name, category_id, category_name (si el backend lo devuelve)
Bank:         id, name, short_code, country, notes, active
Currency:     id, code, name, symbol
Category:     id, key, name
ServiceInput: home_id, name, institution, currency_id, frequency, suggested_amount,
              active, icon_key, billing_type, billing_day, auto_generate,
              is_recurring, start_date, end_date, institution_id
```

### 5.4 APIs / Contratos

#### Tool: `homes_create`

**Entrada**
```json
{
  "name": "Hogar Demo",
  "address": "Calle Demo 123",
  "usage": "rental"
}
```
- `name` obligatorio; `usage` ∈ `rental|inhabited|unused` (default `rental`).

**Salida 200**
```json
{
  "id": 41,
  "name": "Hogar Demo",
  "address": "Calle Demo 123",
  "usage": "rental"
}
```

#### Tool: `institutions_create`

**Entrada**
```json
{
  "name": "Proveedor Demo",
  "category": "servicios"
}
```
- `name` obligatorio; `category_id` o `category` opcionales (excluyentes; si vienen ambos, gana `category_id` y se valida que exista).

**Salida 200**
```json
{
  "id": 87,
  "name": "Proveedor Demo",
  "category_id": 3
}
```

#### Tool: `banks_create`

**Entrada**
```json
{
  "name": "Banco Demo",
  "short_code": "DEMO",
  "country": "NI",
  "notes": "Cuenta sintética",
  "active": true
}
```
- `name` obligatorio; `country` ISO-2; `active` default `true`.

**Salida 200**
```json
{
  "id": 5,
  "name": "Banco Demo",
  "short_code": "DEMO",
  "country": "NI",
  "notes": "Cuenta sintética",
  "active": true
}
```

#### Tool: `services_create`

**Entrada**
```json
{
  "name": "Internet Demo",
  "home": "Hogar Demo",
  "currency": "NIO",
  "institution": "Proveedor Demo",
  "frequency": "monthly",
  "suggested_amount": 1000.00,
  "icon_key": "internet",
  "billing_type": "fixed",
  "billing_day": 5,
  "active": true
}
```
- Requiere `name` y exactamente un identificador de hogar (`home` o `home_id`) y uno de moneda (`currency` o `currency_id`).
- `frequency` default `monthly`; `billing_type` default `variable`; `icon_key` default `other`; `suggested_amount` default `0`; `active` default `true`; `auto_generate`/`is_recurring` default `false`.
- `billing_day` ∈ 1..31 o ausente.
- **Efecto documentado**: con `active: true`, el backend genera la factura del período actual (comportamiento existente de `POST /api/services`).

**Salida 200**
```json
{
  "id": 12,
  "name": "Internet Demo",
  "home_id": 41,
  "currency_id": 1,
  "frequency": "monthly",
  "suggested_amount": 1000.00,
  "billing_type": "fixed",
  "icon_key": "internet",
  "active": true,
  "current_bill_generated": true
}
```

#### Errores comunes

```json
{
  "error": "already_exists",
  "message": "Ya existe un hogar con ese nombre.",
  "hint": "Usá el existente: id=41, name=\"Hogar Demo\""
}
```

| Código | Cuándo |
|---|---|
| `invalid_request` | Campo obligatorio ausente o tipo inválido |
| `invalid_value` | Enum o rango inválido (`usage`, `frequency`, `billing_type`, `billing_day`, `country`) |
| `writes_disabled` | `P40LA_MCP_ALLOW_WRITES` distinto de `true` |
| `not_found` | Referencia inexistente; `hint` con opciones |
| `ambiguous_reference` | Más de una coincidencia por nombre |
| `already_exists` | Duplicado por nombre normalizado |
| `api_error` | Error del backend no clasificado |
| `invalid_response` | Respuesta del backend no decodificable |

### 5.5 Dependencias

- **Internas**: `internal/mcp/api`, `internal/mcp/config`, `internal/mcp/registry`, `internal/mcp/tools/all`, `internal/mcp/tools/bills` (solo migración a `toolkit`).
- **Externas**: `github.com/modelcontextprotocol/go-sdk/mcp` (ya presente). Sin dependencias nuevas.

---

## 6. Criterios de Aceptación

> IDs canónicos por dominio.

### 6.1 Funcionales

- [x] **CA-MCP-ENT-001**: `tools/list` incluye `homes_create`, `institutions_create`, `banks_create` y `services_create` con schemas válidos y descripciones en español.
- [x] **CA-MCP-ENT-002**: `homes_create` crea un hogar con `name` y devuelve `id`; `usage` inválido responde `invalid_value` sin POST.
- [x] **CA-MCP-ENT-003**: `institutions_create` crea una institución; `category` inexistente responde `not_found` con hint.
- [x] **CA-MCP-ENT-004**: `banks_create` crea un banco; `country` distinto de ISO-2 responde `invalid_value`.
- [x] **CA-MCP-ENT-005**: `services_create` resuelve `home` y `currency` por nombre/código y crea el servicio; devuelve `id` y `current_bill_generated`.
- [x] **CA-MCP-ENT-006**: con `P40LA_MCP_ALLOW_WRITES=false`, las cuatro tools responden `writes_disabled` y no ejecutan `POST`.
- [x] **CA-MCP-ENT-007**: un duplicado normalizado responde `already_exists` con el `id` existente y no ejecuta `POST`.
- [x] **CA-MCP-ENT-008**: una referencia inexistente responde `not_found` con hasta 10 opciones `{id, name}`.
- [x] **CA-MCP-ENT-009**: dos hogares con el mismo nombre y distinto id responden `ambiguous_reference` con candidatos.
- [x] **CA-MCP-ENT-010**: servicio con `active: true` documenta y reporta la generación de la factura actual; con `active: false` no la genera.
- [x] **CA-MCP-ENT-011**: `docs/mcp/tools/entities.md`, el catálogo y `development.md` documentan las tools y el gate de escritura.
- [x] **CA-MCP-ENT-012**: las tools de facturas siguen pasando sus tests tras extraer `toolkit`.

### 6.2 No funcionales

- [x] **CA-NF-001**: ninguna tool contiene SQL ni importa `internal/storage`/`internal/services`.
- [x] **CA-NF-002**: `go build ./...` y `go test ./...` pasan; sin dependencias nuevas en `go.mod`.
- [x] **CA-NF-003**: el binario MCP arranca con `P40LA_BASE_URL` y `P40LA_API_KEY` y registra 7 tools en total.
- [x] **CA-NF-004**: la documentación no expone secretos ni datos reales.

### 6.3 Testing

- **Unit tests**: validación de enums/rangos, resolución por nombre/código, normalización de duplicados, gate de escritura y mapeo de errores, con una `API` falsa.
- **Protocol tests**: `mcp.NewInMemoryTransports` para `tools/list` y una llamada de alta end-to-end con API falsa.
- **Client tests**: path, método, header `X-API-Key` y decodificación de `Create*`/`List*`.
- **Integración manual**: copia de producción en `/tmp` servida en local; crear hogar → institución → banco → servicio; verificar duplicados y `active`.

### 6.4 Criterios de proceso y reglas

- [x] **CA-ARCH-STACK / CA-ARCH-GO**: solo Go y `go-sdk` existente; `gofmt` y errores explícitos.
- [x] **CA-ARCH-FOLDERS / CA-ARCH-LAYERS**: tools en `internal/mcp/tools/entities/`; sin SQL ni lógica de dominio en el MCP.
- [x] **CA-ARCH-IHOST**: sin migraciones, dependencias nuevas ni cambios en el server.
- [ ] **CA-DATA-PRODCOPY**: validación manual solo contra copia de producción en `/tmp`.
- [x] **CA-SEC-NO-PII / CA-SEC-SYNTHETIC**: sin datos reales; ejemplos `demo-*`.
- [x] **CA-SEC-COMMIT**: `git diff --cached` revisado; commits con prefijo `SPEC-146:`.
- [x] **CA-TEST-LOCAL**: `go build ./...` y `go test ./...` en verde antes de commit.
- [x] **CA-TEST-SERVER**: `./scripts/check-server.sh` antes de avisar al usuario.
- [ ] **CA-TEST-USER-FLOW**: estado `pending_release` solo con pruebas manuales confirmadas (🤖).
- [x] **CA-DELIV-WORKTREE**: trabajo en `../p40la-ihost-spec-146`, rama `feature/SPEC-146`.
- [ ] **CA-DELIV-CONFIRM**: merge/push solo con confirmación explícita del usuario.
- [x] **CA-DELIV-MCP-DOCS**: `docs/mcp/` actualizado en el mismo commit; `./scripts/check-mcp-docs.sh --staged` pasa.
- [x] **CA-DELIV-MCP-DOCS-SUBAGENT**: redacción de `docs/mcp/` delegada a un subagente y revisada por el agente principal.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `toolkit` + DTOs y métodos del cliente `api` | 0.5 día | SPEC-145 en `main` |
| 2 | `homes_create` + `institutions_create` + tests | 0.5 día | Fase 1 |
| 3 | `banks_create` + tests | 0.25 día | Fase 1 |
| 4 | `services_create` con resolución y duplicados + tests | 1 día | Fases 1–3 |
| 5 | Registro en `all`, docs MCP (subagente) y guardrail | 0.5 día | Fase 4 |
| 6 | Validación local contra copia de producción y confirmación del usuario | 0.5 día | Fase 5 |

### 7.2 Milestones

1. **MVP**: cuatro tools registradas, gate de escritura, tests y docs.
2. **V1.0**: resolución por nombre, hints, duplicados y validación manual completa.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El agente crea entidades erróneas en producción | Media | Alto | Gate `P40LA_MCP_ALLOW_WRITES` (default `false`), duplicados y hints; el usuario aprueba el uso |
| Ambigüedad de nombres (`Hogar`, `hogar`) | Media | Medio | Normalización + `ambiguous_reference` con candidatos; `home_id` como escape |
| Drift de allowlists (`icon_key`, categorías) | Media | Bajo | Backend sigue siendo la fuente de verdad; validación temprana solo de enums estables |
| Efecto colateral de crear factura al activar un servicio | Alta | Medio | Documentado en la tool y en docs; opción `active: false` |
| Duplicado legítimo bloqueado | Baja | Bajo | `allow_duplicate=true` (P1) y hint con el existente |
| Regresión en tools de facturas al mover helpers | Baja | Medio | Tests existentes de `bills` + protocol test |
| El MCP apunta a una versión vieja del server | Baja | Medio | Error `api_error` claro; docs de configuración y health check |

---

## 9. Notas y Referencias

- Reglas aplicadas: `docs/rules/10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- SPEC-145: servidor MCP, API key global y convenciones de tools (`docs/mcp/`).
- Endpoints: `internal/api/routes.go` (`homes`, `institutions`, `banks`, `services`, `currencies`, `institution-categories`).
- Validaciones: `internal/services/{home,institution,bank,service}.go`.
- Ejemplos: datos sintéticos (`Hogar Demo`, `Banco Demo`, `Proveedor Demo`, `NIO`).

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | opencode | Creación inicial de la especificación |
| 2026-11-24 | opencode | Inicio de implementación: toolkit compartido, DTOs/cliente y las cuatro tools con tests |
| 2026-11-24 | opencode | Imagen `0.2.0` publicada y desplegada en el iHost; `tools/list` devuelve 7 tools (3 de facturas + 4 de entidades) |
