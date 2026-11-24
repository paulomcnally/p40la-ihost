---
title: "MCP de cuentas bancarias y tarjetas de crédito: alta, búsqueda y edición"
id: "SPEC-152"
status: "released"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 155
---

# MCP de cuentas bancarias y tarjetas de crédito: alta, búsqueda y edición

**ID**: SPEC-152  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-24  
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

El MCP de p40la-ihost ya opera facturas (SPEC-145) y está sumando el alta de entidades base (SPEC-146). Sin embargo, un agente conectado no puede registrar ni mantener las **cuentas bancarias** y las **tarjetas de crédito**, que son las entidades que el usuario usa a diario para transferencias, conciliaciones y estados de cuenta. Hoy tiene que salir del MCP y usar la UI.

Esta spec agrega seis tools: `bank_accounts_search`, `bank_accounts_create`, `bank_accounts_update`, `credit_cards_search`, `credit_cards_create` y `credit_cards_update`. Buscan por nombre, alias, número, últimos cuatro dígitos o banco; resuelven banco y moneda por nombre/código para que el agente no dependa de IDs; y editan con semántica parcial (leen la entidad, mezclan solo los campos enviados y hacen el `PUT` completo que exige el backend). Las escrituras quedan detrás de `P40LA_MCP_ALLOW_WRITES`, igual que el resto del MCP.

El impacto en el iHost es bajo: **no hay migraciones ni endpoints nuevos**, no se agregan dependencias y cada tool hace un número acotado de llamadas HTTP a la API existente. La búsqueda recorre los bancos (pocos) y sus cuentas/tarjetas, sin cargar datos en memoria del servidor.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec no toca frontend, rutas de UI, modales ni el esquema de la base de datos, así que los dominios `20-ui` y `30-ux` quedan fuera.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go + `modelcontextprotocol/go-sdk` ya presente; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Tools en `internal/mcp/tools/banking/`; DTOs y cliente en `internal/mcp/api/banking*.go` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | Las tools solo parsean, resuelven referencias y llaman al cliente HTTP; la validación de dominio queda en `internal/services` y `internal/modules/bank/creditcard` | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | N/A: no se crean módulos nuevos; se consumen los endpoints de `bank.creditcard` y del módulo legacy de bancos | — |
| `ARCH-GO` | `gofmt`, errores explícitos, nombres en inglés, `log/slog` a stderr, sin `panic` | `CA-ARCH-GO` |
| `ARCH-IHOST` | Sin migraciones, sin dependencias nuevas y sin cambios en el server; las tools corren en el proceso MCP y hacen llamadas HTTP acotadas | `CA-ARCH-IHOST` |
| `DATA-MIGRATIONS` | N/A: no hay migraciones ni cambios de esquema | — |
| `DATA-NO-DROP` / `DATA-PRODCOPY` | La validación manual de búsquedas se corre contra una copia de producción en `/tmp`; nunca contra `data/app.db` ni producción | `CA-DATA-PRODCOPY` |
| `SEC-NO-PII` | Spec, issue, docs y tests solo con datos sintéticos; la tool no loguea payloads ni resultados | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `Banco Demo`, `Cuenta Demo`, `demo-account-001`, `0000 0000 0000 0000` | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | El logger HTTP sigue registrando solo método, path, status y duración | `CA-SEC-LOGS` |
| `SEC-COMMIT` / `SEC-HOOKS` | Commit `SPEC-152: ...`; `git diff --cached` revisado; hooks instalados y sin `--no-verify` | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` verdes antes del commit; tests de tools con API falsa | `CA-TEST-LOCAL` |
| `TEST-SERVER` | Si se prueba contra el server local, `./scripts/check-server.sh` antes de avisar | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | No se pasa a `pending_release` ni `released` sin pruebas manuales confirmadas por el usuario | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo exclusivo en `../p40la-ihost-spec-152`, rama `feature/SPEC-152` | `CA-DELIV-WORKTREE` |
| `DELIV-COMMIT` | Commits `SPEC-152: descripción corta` | `CA-DELIV-COMMIT` |
| `DELIV-CONFIRM` | Merge a `main` y push solo con confirmación explícita del usuario; sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-LABELS` | Issue con `spec/draft`; al liberar, `spec/released` y cierre | `CA-DELIV-LABELS` |
| `DELIV-MCP-DOCS` | El mismo commit actualiza `docs/mcp/` (`tools/banking.md`, catálogo, README) | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente y se revisa | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-DOCKER` | N/A: no se publican imágenes en esta spec | — |
| `UI-*` / `UX-*` | N/A: no hay cambios de frontend, rutas, pantallas ni modales | — |

> Reglas fuente: `docs/rules/10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — `bank_accounts_search`**: busca cuentas por texto libre (`query` opcional) y banco (`bank` o `bank_id` opcional). Devuelve `id`, `bank_id`, `bank_name`, `alias`, `account_number`, `account_type`, `currency_id`, `currency_code`, `beneficiary` y `active`. Si `query` viene vacía, lista las primeras `limit` cuentas activas.
2. **REQ-002 — `bank_accounts_create`**: crea una cuenta con `bank` (nombre, `short_code` o `bank_id`), `alias`, `account_number`, `currency` (código o `currency_id`) y `beneficiary` obligatorios; `account_type` (`savings` default / `checking`), `notes` y `active` (default `true`) opcionales. Llama a `POST /api/banks/{bank_id}/accounts`.
3. **REQ-003 — `bank_accounts_update`**: actualiza por `id` con semántica parcial: lee `GET /api/bank-accounts/{id}`, mezcla solo los campos enviados y hace `PUT /api/bank-accounts/{id}` con el cuerpo completo. El banco no cambia (el backend lo ignora) y se documenta.
4. **REQ-004 — `credit_cards_search`**: busca tarjetas por `query` (alias, `last4`, titular, marca o banco) y banco opcional. Devuelve `id`, `bank_id`, `bank_name`, `alias`, `network`, `last4`, `holder`, `currency_id`, `currency_code`, `credit_limit`, `limit_currency`, `statement_day`, `payment_due_day` y `active`.
5. **REQ-005 — `credit_cards_create`**: crea una tarjeta con `bank`, `alias`, `network` (`visa|mastercard|amex|diners|other`), `last4` (4 dígitos) y `currency` obligatorios. Opcionales con defaults: `statement_day=1`, `payment_due_day=1`, `min_payment_pct=4`, `limit_currency=NIO`, `color=blue`, `icon=credit`, `active=true`; también `holder`, `secondary_currency`, `category`, `exchange_rate`, `credit_limit`, `credit_limit_usd`, `bonus_due_day`, tasas de interés y `notes`. Llama a `POST /api/banks/{bank_id}/credit-cards`.
6. **REQ-006 — `credit_cards_update`**: actualiza por `id` con la misma semántica parcial de REQ-003 sobre `PUT /api/credit-cards/{id}`.
7. **REQ-007 — Resolución de banco**: `bank` se compara sin distinguir mayúsculas ni acentos contra `name` y `short_code` de `GET /api/banks`. Exacto → prefijo → contiene. Si no existe → `not_found` con hint; si es ambiguo → `ambiguous_reference` con candidatos.
8. **REQ-008 — Resolución de moneda**: `currency` se resuelve contra `GET /api/currencies` por `code` (ej. `NIO`, `USD`) o `id`. Acepta `currency_id` directo. Si no existe → `not_found` con la lista de códigos disponibles.
9. **REQ-009 — Gate de escritura**: con `P40LA_MCP_ALLOW_WRITES=false` (default), las tools de alta y edición responden `writes_disabled` sin llamar al backend. Las búsquedas funcionan siempre.
10. **REQ-010 — Duplicados**: antes de crear o editar, se detecta una cuenta con el mismo `bank_id` + `account_number` (normalizado) o una tarjeta con el mismo `bank_id` + `alias` (sin distinguir mayúsculas ni acentos). Se responde `already_exists` con el `id` existente, sin escribir.
11. **REQ-011 — Registro**: las seis tools se montan en `internal/mcp/tools/all/register.go` y aparecen en `tools/list` junto a las tools existentes.
12. **REQ-012 — Documentación**: `docs/mcp/tools/banking.md` documenta entradas, salidas, defaults, errores y ejemplos sintéticos; se actualizan el catálogo de tools y el README del MCP.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013 — Filtro por banco**: `bank`/`bank_id` reduce la búsqueda a un solo banco (una llamada a la lista de cuentas/tarjetas en vez de recorrer todos).
2. **REQ-014 — Filtro `active`**: `active` opcional (`true`/`false`) filtra el resultado; sin valor devuelve activos e inactivos.
3. **REQ-015 — `truncated` y `total`**: si hay más coincidencias que `limit`, la respuesta incluye `truncated: true` y el conteo completo.
4. **REQ-016 — Orden determinista**: exactos primero, luego prefijo, luego contiene; dentro de cada grupo por alias y `id` ascendente.
5. **REQ-017 — Errores de backend accionables**: `409 duplicate` → `already_exists`; `400 validation_error` → `invalid_value` con el mensaje del backend y hint; `404` → `not_found`.
6. **REQ-018 — Búsqueda sin resultados**: responde `count: 0`, `items: []` y un `hint` con sugerencias; no es `isError`.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-019 — `bank_accounts_delete` / `credit_cards_delete`**: se difieren a otra spec; esta spec no expone borrado.
2. **REQ-020 — Búsqueda por titular de tarjeta**: incluida en `query` (P0); un filtro dedicado `holder` se difiere.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: búsqueda sin banco ≤ 1 + B llamadas HTTP (B = cantidad de bancos, esperado < 20); con banco ≤ 2. Respuesta < 2 s en LAN.
- **Memoria**: sin caché ni índices en memoria; cada llamada se descarta al terminar.
- **Seguridad**: API key global obligatoria; escrituras deshabilitadas por defecto; sin logs de payloads, números de cuenta ni últimos cuatro dígitos.
- **Almacenamiento**: cero migraciones y cero archivos de datos nuevos.
- **iHost**: sin impacto en el proceso del server; el MCP corre fuera del contenedor.
- **Compatibilidad**: no rompe las tools de facturas ni las de entidades.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- Patrón MCP existente: `internal/mcp/tools/bills` (Toolset + Register + gate `WritesEnabled`), `internal/mcp/registry`, `internal/mcp/tools/all/register.go` y `docs/mcp/development.md`.
- SPEC-146 (en progreso en otro worktree) extrae `internal/mcp/tools/toolkit` y amplía `api.API` con entidades; SPEC-150 (draft) agrega `entities_search`. Esta spec **no depende de esos cambios** para poder liberarse sola.
- Endpoints existentes:
  - `GET /api/banks` → lista de bancos (`id`, `name`, `short_code`, `country`, `active`).
  - `GET /api/banks/{id}/accounts` y `GET /api/bank-accounts/{id}`.
  - `POST /api/banks/{id}/accounts`, `PUT /api/bank-accounts/{id}`.
  - `GET /api/banks/{id}/credit-cards`, `GET /api/credit-cards/{id}`.
  - `POST /api/banks/{id}/credit-cards`, `PUT /api/credit-cards/{id}`.
  - `GET /api/currencies`.
- Validaciones del backend:
  - Cuenta: `alias`, `account_number`, `account_type ∈ savings|checking`, `currency_id` existente, `beneficiary`; límites de longitud.
  - Tarjeta: `alias`, `network` de la allowlist, `last4` de 4 dígitos, `currency_id` existente, `secondary_currency_id` distinto, días 1..31, `bonus_due_day` 0..31 y distinto de `payment_due_day`, `limit_currency ∈ NIO|USD`, `min_payment_pct` 0..100, tasas ≥ 0.
- No existe endpoint global de cuentas ni de tarjetas: los listados son por banco.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Tools en el MCP que recorren bancos y usan los endpoints existentes | Cero cambios en backend, cero migraciones, despliegue inmediato | Hasta 1+B llamadas por búsqueda | ✅ Seleccionada |
| B. Endpoints globales `GET /api/bank-accounts` y `GET /api/credit-cards` | Una sola llamada y búsqueda centralizada | Toca handlers, services y storage de dos módulos; más superficie y tests | ❌ Rechazada para esta spec |
| C. Tools que exigen IDs | Menos código | El agente no conoce IDs; fricción y errores | ❌ Rechazada |
| D. Una tool combinada `banking_*` con `entity` | Menos tools | Schemas muy distintos entre cuenta y tarjeta; peor descubrimiento | ❌ Rechazada |
| E. `dry_run` en altas/ediciones | Consistente con facturas | Duplica validación y no aporta: el gate de escritura ya protege | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — Paquete único `banking` con seis tools**
- **Contexto**: cuentas y tarjetas comparten resolución de banco/moneda, búsqueda y errores.
- **Decisión**: un solo paquete `internal/mcp/tools/banking` con `accounts.go`, `cards.go`, `resolve.go`, `results.go` y `tools.go`.
- **Consecuencias**: helpers sin duplicar; el paquete crece con futuras tools bancarias.

**ADR-002 — Búsqueda en el MCP sobre los listados por banco**
- **Contexto**: no hay endpoint global y agregarlo toca dos módulos.
- **Decisión**: listar bancos (o resolver el banco pedido) y consultar cuentas/tarjetas por banco; filtrar y ordenar en el MCP. Se valida `limit` (1..50) y se corta al alcanzar el máximo.
- **Consecuencias**: más llamadas HTTP, pero acotadas y sin estado; si el volumen de bancos crece, se justifica el endpoint global en otra spec (riesgo documentado).

**ADR-003 — Edición parcial con lectura + mezcla + `PUT` completo**
- **Contexto**: los `PUT` del backend reemplazan el recurso; omitir campos los pondría en cero.
- **Decisión**: la tool lee la entidad, aplica solo los campos presentes y envía el cuerpo completo. `secondary_currency: ""` limpia la moneda secundaria.
- **Consecuencias**: una llamada GET extra por edición; nunca se pierden campos.

**ADR-004 — Helpers locales en `banking` en lugar de `toolkit`**
- **Contexto**: SPEC-146/150 crean `internal/mcp/tools/toolkit` en ramas paralelas; depender de código no mergeado bloquea esta spec.
- **Decisión**: `results.go` local con `jsonResult`, `toolError` y `apiError`, equivalentes a los de `bills`.
- **Consecuencias**: ~60 líneas duplicadas; cuando `toolkit` esté en `main`, una spec de limpieza puede migrar `banking` y `bills`. Se anota como deuda menor.

**ADR-005 — Sin borrado**
- **Contexto**: el pedido es registrar, buscar y editar; el borrado es destructivo y requiere confirmación por palabra en la UI.
- **Decisión**: no se exponen tools de borrado en esta spec.
- **Consecuencias**: menor superficie de riesgo; se puede agregar después con `dry_run`/confirmación.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
Agente / cliente MCP
        │ stdio o HTTP (MCP)
        ▼
cmd/mcp-server ──► internal/mcp/tools/banking
        │                 │
        │                 ├── resolve.go (banco, moneda, duplicados, ranking)
        │                 ▼
        │          internal/mcp/api (cliente HTTP + X-API-Key)
        │                 │
        ▼                 ▼
   stderr (logs)   p40la-ihost :8088
                    ├── /api/banks, /api/currencies
                    ├── /api/bank-accounts, /api/banks/{id}/accounts
                    └── /api/credit-cards, /api/banks/{id}/credit-cards
```

### 5.2 Componentes

#### 5.2.1 `internal/mcp/tools/banking`
- **Responsabilidad**: schemas de entrada, validación barata, resolución de banco/moneda, detección de duplicados, ranking de búsqueda y mapeo de errores.
- **Interfaz**: `Register(server, client, cfg)` monta las seis tools.
- **Dependencias**: `internal/mcp/api`, `internal/mcp/config`, `modelcontextprotocol/go-sdk/mcp`.
- **Ubicación**: `internal/mcp/tools/banking/{tools,accounts,cards,resolve,results}.go`.

#### 5.2.2 `internal/mcp/api`
- **Responsabilidad**: DTOs y métodos HTTP de bancos, monedas, cuentas y tarjetas.
- **Ubicación**: `internal/mcp/api/banking.go` (tipos) y `internal/mcp/api/banking_client.go` (métodos).

#### 5.2.3 `internal/mcp/tools/all/register.go`
- **Responsabilidad**: registrar `banking.Register` junto a `bills.Register` (y las tools de otras specs cuando estén en `main`).

### 5.3 Modelo de datos

No hay cambios de esquema. DTOs nuevos (subconjunto de la respuesta del backend):

```
BankRef:      id, name, short_code, active
CurrencyRef:  id, code, name
BankAccount:  id, bank_id, bank_name, alias, account_number, account_type,
              currency_id, currency_code, beneficiary, notes, active
CreditCard:   id, bank_id, bank_name, alias, network, last4, holder,
              currency_id, currency_code, secondary_currency_id,
              secondary_currency_code, category, limit_currency, exchange_rate,
              credit_limit, credit_limit_usd, statement_day, payment_due_day,
              bonus_due_day, min_payment_pct, interest rates, color, icon,
              active, notes
```

Las claves de salida usan `snake_case` y los campos opcionales se omiten con `omitempty` cuando no aplican.

### 5.4 APIs / Contratos

#### Tool: `bank_accounts_search`

**Entrada**
```json
{
  "query": "demo",
  "bank": "Banco Demo",
  "active": true,
  "limit": 10
}
```

**Salida**
```json
{
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 1,
      "bank_id": 1,
      "bank_name": "Banco Demo",
      "alias": "Cuenta Demo",
      "account_number": "demo-account-001",
      "account_type": "savings",
      "currency_id": 1,
      "currency_code": "NIO",
      "beneficiary": "Titular Demo",
      "active": true
    }
  ]
}
```

#### Tool: `bank_accounts_create`

**Entrada**
```json
{
  "bank": "Banco Demo",
  "alias": "Cuenta Demo",
  "account_number": "demo-account-001",
  "currency": "NIO",
  "beneficiary": "Titular Demo",
  "account_type": "savings",
  "notes": "",
  "active": true
}
```

**Salida**: la cuenta creada con `id` y los campos de REQ-001.

#### Tool: `bank_accounts_update`

**Entrada**: `id` obligatorio + cualquier campo de REQ-002 (todos opcionales). `bank`/`bank_id` se rechazan con `invalid_value` porque el banco no se cambia.

**Salida**: la cuenta actualizada.

#### Tool: `credit_cards_search`

**Entrada**
```json
{
  "query": "demo",
  "bank": "Banco Demo",
  "active": true,
  "limit": 10
}
```

**Salida**
```json
{
  "count": 1,
  "total": 1,
  "truncated": false,
  "items": [
    {
      "id": 1,
      "bank_id": 1,
      "bank_name": "Banco Demo",
      "alias": "Tarjeta Demo",
      "network": "visa",
      "last4": "0000",
      "holder": "Titular Demo",
      "currency_id": 1,
      "currency_code": "NIO",
      "credit_limit": 0,
      "limit_currency": "NIO",
      "statement_day": 1,
      "payment_due_day": 1,
      "active": true
    }
  ]
}
```

#### Tool: `credit_cards_create`

**Entrada**
```json
{
  "bank": "Banco Demo",
  "alias": "Tarjeta Demo",
  "network": "visa",
  "last4": "0000",
  "currency": "NIO",
  "holder": "Titular Demo",
  "statement_day": 1,
  "payment_due_day": 1,
  "min_payment_pct": 4,
  "limit_currency": "NIO",
  "credit_limit": 0
}
```

**Salida**: la tarjeta creada con `id` y los campos de REQ-004.

#### Tool: `credit_cards_update`

**Entrada**: `id` obligatorio + cualquier campo de REQ-005 (todos opcionales). `bank`/`bank_id` se rechazan porque el banco no se cambia.

#### Errores comunes

| Código | Cuándo | `hint` |
|---|---|---|
| `writes_disabled` | Alta/edición con el gate apagado | Definir `P40LA_MCP_ALLOW_WRITES=true` |
| `not_found` | Banco, moneda o entidad inexistente | Lista de candidatos o valores válidos |
| `ambiguous_reference` | Varios bancos coinciden | Enviar `bank_id` |
| `already_exists` | Cuenta o tarjeta duplicada | Devuelve `id` y datos existentes |
| `invalid_value` | Campo fuera de allowlist/rango o payload inválido | Mensaje del backend |
| `upstream_error` | Fallo de red o 5xx | Verificar `P40LA_BASE_URL` y `/health` |

### 5.5 Dependencias

- **Internas**: `internal/mcp/api`, `internal/mcp/config`, `internal/mcp/registry`, `internal/mcp/tools/all`.
- **Externas**: ninguna nueva; `modelcontextprotocol/go-sdk` ya está en `go.mod`.
- **Backend**: endpoints existentes; no se modifica.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [x] `CA-REQ-001`: `bank_accounts_search` devuelve cuentas por alias, número, beneficiario y banco, con `count/total/truncated`.
- [x] `CA-REQ-002`: `bank_accounts_create` crea la cuenta con banco y moneda resueltos por nombre/código y devuelve el `id`.
- [x] `CA-REQ-003`: `bank_accounts_update` cambia solo los campos enviados y conserva el resto (verificado con un GET posterior).
- [x] `CA-REQ-004`: `credit_cards_search` devuelve tarjetas por alias, `last4`, titular y banco.
- [x] `CA-REQ-005`: `credit_cards_create` aplica los defaults documentados y crea la tarjeta.
- [x] `CA-REQ-006`: `credit_cards_update` cambia solo los campos enviados.
- [x] `CA-REQ-007`: banco inexistente → `not_found` con candidatos; banco ambiguo → `ambiguous_reference`.
- [x] `CA-REQ-008`: moneda inexistente → `not_found` con códigos disponibles.
- [x] `CA-REQ-009`: con escrituras deshabilitadas, las tools de escritura responden `writes_disabled` y no llaman al backend.
- [x] `CA-REQ-010`: duplicados de cuenta (`bank_id` + `account_number`) y tarjeta (`bank_id` + `alias`) responden `already_exists`.
- [x] `CA-REQ-011`: las seis tools aparecen en `tools/list`.
- [x] `CA-REQ-012`: `docs/mcp/tools/banking.md` documenta las seis tools y el catálogo las lista.
- [x] `CA-UX-*` / `CA-UI-*`: N/A, no hay UI.

### 6.2 No funcionales

- [x] `CA-ARCH-STACK`: `go.mod` sin dependencias nuevas.
- [x] `CA-ARCH-FOLDERS`: archivos en las rutas declaradas en 5.2.
- [x] `CA-ARCH-LAYERS`: el paquete `banking` no importa `internal/services`, `internal/storage` ni SQL.
- [x] `CA-ARCH-GO`: `gofmt -l` sin archivos y `go vet ./internal/mcp/...` limpio.
- [x] `CA-ARCH-IHOST`: sin migraciones ni cambios en `cmd/server`; el server sigue respondiendo `/health`.
- [x] `CA-SEC-NO-PII`: `git diff --cached` sin datos reales, tokens ni números de cuenta/tarjeta.
- [x] `CA-SEC-SYNTHETIC`: ejemplos y fixtures usan solo datos `demo-*`.
- [x] `CA-TEST-LOCAL`: `go build ./...` y `go test ./...` pasan.
- [x] `CA-DELIV-WORKTREE`: cambios solo en `../p40la-ihost-spec-152`.
- [x] `CA-DELIV-MCP-DOCS`: el commit incluye `docs/mcp/`.

### 6.3 Testing

- **Unit tests**: normalización de nombres, ranking de búsqueda, resolución de banco/moneda, defaults de tarjeta, detección de duplicados, edición parcial.
- **Integration tests**: tools registradas en `tools/list` (transporte en memoria) y llamadas con API falsa.
- **Live test opcional**: `TestLiveAccountsAndCards` corre el flujo completo contra un server real cuando se definen `P40LA_LIVE_URL` y `P40LA_LIVE_API_KEY`; se saltea por defecto.
- **Manual**: crear/buscar/editar una cuenta y una tarjeta en local contra una copia de producción en `/tmp`.
- **Performance**: búsqueda sin banco con 10 bancos < 2 s en LAN.

**Validación técnica ejecutada (2026-11-24)**: `go build ./...`, `go vet ./internal/mcp/...` y `go test ./...` verdes. El live test se corrió contra un server local con `DATA_DIR` sintético en `/tmp/opencode/p40la-spec152`: alta, búsqueda, edición y duplicado de cuenta y tarjeta OK. La validación manual del usuario queda pendiente (`TEST-USER-FLOW`).

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | DTOs y métodos del cliente HTTP | 0.5 día | Ninguna |
| 2 | Resolución y búsqueda en `banking` | 0.5 día | Fase 1 |
| 3 | Seis tools + registro | 1 día | Fase 2 |
| 4 | Tests unitarios y de protocolo | 0.5 día | Fase 3 |
| 5 | Documentación MCP (subagente) y release | 0.5 día | Fase 3 |

### 7.2 Milestones

1. **MVP**: búsqueda, alta y edición de cuentas y tarjetas con gate de escritura.
2. **V1.0**: tests, documentación y release.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Muchos bancos → búsqueda con demasiadas llamadas | Baja | Medio | Filtro `bank`/`bank_id`; si crece, endpoint global en otra spec |
| `PUT` completo pisa campos no enviados | Media | Alto | Leer + mezclar + enviar el cuerpo completo (ADR-003) |
| Duplicados por reintentos del agente | Media | Medio | Verificación previa por `bank_id` + clave y `already_exists` |
| Colisión de helpers con `toolkit` de SPEC-146/150 | Media | Bajo | Helpers locales; limpieza posterior documentada |
| Nombres de banco ambiguos | Media | Bajo | `ambiguous_reference` con candidatos y soporte de `bank_id` |
| Datos sensibles en logs | Baja | Alto | Sin logging de payloads; middleware solo método/path/status |

---

## 9. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-133 (cuentas y formatos), SPEC-137/138 (tarjetas), SPEC-145 (MCP de facturas), SPEC-146 (alta de entidades), SPEC-150 (búsqueda de entidades), SPEC-151 (MCP de autos).
- Documentación MCP: `docs/mcp/README.md`, `docs/mcp/development.md`, `docs/mcp/tools/README.md`.
- Endpoints: ver 4.1.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial de la especificación |
| 2026-11-24 | opencode | Implementación de DTOs, cliente HTTP, seis tools, resolución y tests; documentación MCP delegada a subagente |
| 2026-11-24 | opencode | Preparación de release: build, vet, tests y live test con DB sintética; validación manual omitida por autorización explícita del usuario |
| 2026-11-24 | opencode | Release: merge `cbf7aab1` a `main`; issue #155; sin migraciones, bump ni imagen (cambio solo del MCP) |
