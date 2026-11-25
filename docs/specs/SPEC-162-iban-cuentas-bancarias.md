---
title: "Agregar IBAN a cuentas bancarias y tools de gestión"
id: "SPEC-162"
status: "pending_release"
author: "opencode"
created: "2026-11-25"
updated: "2026-11-25"
github_issue: 164
---

# Agregar IBAN a cuentas bancarias y tools de gestión

**ID**: SPEC-162
**Estado**: pending_release
**Autor**: opencode
**Creado**: 2026-11-25
**Actualizado**: 2026-11-25

---

## 1. Resumen Ejecutivo

Las cuentas bancarias de Nicaragua incorporaron el **IBAN** como nueva identificación estándar (aprobado por el Banco Central de Nicaragua en octubre de 2018). Hoy el modelo `BankAccount` de p40la-ihost no tiene ese campo: la UI y las tools MCP de gestión de cuentas (`bank_accounts_create`, `bank_accounts_update`, `bank_accounts_search`) no lo reconocen, por lo que un usuario no puede registrar ni consultar el IBAN de sus cuentas.

Esta spec agrega el campo **`iban`** como propiedad opcional de las cuentas bancarias, de punta a punta: columna en SQLite, modelo/storage/service/handler del backend, formulario y detalle del frontend, y las tools MCP de alta/edición/búsqueda. El campo se trata de forma **conservadora**: texto alfanumérico opcional con un límite de longitud (34, el máximo de ISO 13616), **sin** validación de formato ni de dígitos de control, porque el usuario no pidió reglas de validación y el alcance acordado es «reconocer IBAN como propiedad/columna».

El impacto en el iHost es bajo: una migración aditiva (`ALTER TABLE … ADD COLUMN`), sin dependencias nuevas, sin cambios de esquema de tablas protegidas y con un volumen de datos despreciable (una columna de texto corto por cuenta).

> **Las reglas no se copian en la spec.** Se citan por ID desde `docs/rules/` y se declaran en la sección 2.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. El cambio toca backend (modelo/datos), frontend (formulario/detalle) y MCP (tools de cuentas).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go + SQLite + React existentes; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Cambios en las rutas ya existentes de cada capa; sin archivos fuera de lugar | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | `iban` fluye por `models → storage → services → api`; la lógica de validación queda en `services` | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | N/A: no se crean módulos nuevos; se modifica el módulo legacy de bancos (SPEC-133) | — |
| `ARCH-GO` | `gofmt`, errores explícitos, nombres en inglés en código | `CA-ARCH-GO` |
| `ARCH-IHOST` | Migración aditiva ligera; sin dependencias de red; sin cambios en `cmd/server` salvo el arranque normal | `CA-ARCH-IHOST` |
| `UI-SELECT` | N/A: IBAN es un input de texto libre, no un selector | — |
| `UI-INPUT-TOKENS` | El input de IBAN usa `bg-card`/`text-text` (mismo `inputClass` del formulario) | `CA-UI-INPUT-TOKENS` |
| `UI-DARKMODE` | El formulario y el detalle se verifican en darkmode | `CA-UI-DARKMODE` |
| `UI-NUMBER-INPUT` | N/A: IBAN es alfanumérico, no numérico | — |
| `UI-PHONE-INPUT` | N/A: no es teléfono | — |
| `UI-I18N` | Clave `banks.iban` en `es.json` y `en.json` (paridad) | `CA-UI-I18N` |
| `UI-EDIT-LABEL` | N/A: no se agregan acciones de edición nuevas | — |
| `UX-BACK` | N/A: no se agregan rutas nuevas con lista padre | — |
| `UX-DETAIL-READONLY` | El detalle muestra `iban` en modo lectura; editar sigue navegando a `/edit` | `CA-UX-DETAIL-READONLY` |
| `UX-MODAL` / `UX-MODAL-CONFIRM` | N/A: no se agregan modales | — |
| `UX-PREREQ` | N/A: no hay prerrequisito nuevo | — |
| `DATA-MIGRATIONS` | Migración `0061_add_bank_account_iban` con `.up.sql` y `.down.sql` reversible | `CA-DATA-MIGRATIONS` |
| `DATA-NO-DROP` | Solo `ALTER TABLE ADD COLUMN`; nunca `DROP`/reemplazo de la DB del usuario | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | La validación que dependa de datos se corre contra una copia en `/tmp`, nunca contra `data/app.db` | `CA-DATA-PRODCOPY` |
| `SEC-NO-PII` | Spec, tests y docs usan IBAN sintético; no se commitean datos reales | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplo `NI00DEMO00000000000000000000` (ilustrativo) | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` / `SEC-HOOKS` | `git diff --cached` revisado; hooks instalados; sin `--no-verify` | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...`, `npm run check:*` verdes | `CA-TEST-LOCAL` |
| `TEST-MCP-CONTRACT` | `TestToolContracts` y `TestMCPBackendContract` verdes; golden actualizado y revisado | `CA-TEST-MCP-CONTRACT` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar al usuario | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | Prueba manual confirmada por el usuario antes de `pending_release` | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo exclusivo en `../p40la-ihost-spec-162`, rama `feature/SPEC-162` | `CA-DELIV-WORKTREE` |
| `DELIV-COMMIT` | Commits `SPEC-162: descripción corta` | `CA-DELIV-COMMIT` |
| `DELIV-CONFIRM` | Merge a `main` y push solo con confirmación explícita; sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-DOCS` | El mismo commit actualiza `docs/mcp/tools/banking.md` | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La redacción de `docs/mcp/` se delega a un subagente y se integra | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-MCP-IMPACT` | Sección 6 completa; `check-mcp-impact.sh --staged --gate` verde | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Validación delegada a subagente `explore` y registrada en la sección 6 | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-DOCKER` | El release reconstruye la imagen principal multi-arch (`linux/amd64,arm/v7,arm64`) | `CA-DELIV-DOCKER` |

> Reglas fuente: `docs/rules/10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Columna `iban`**: la tabla `bank_accounts` incorpora la columna `iban TEXT NOT NULL DEFAULT ''` mediante una migración reversible (`0061_add_bank_account_iban`).
2. **REQ-002 — Backend**: `models.BankAccount` incluye `IBAN string` con JSON `"iban"`; `storage` lo lee (SELECT/scan) y lo escribe (INSERT/UPDATE); `services` lo normaliza (trim) y valida su longitud; el handler acepta `iban` en el request de alta y edición (`bankAccountRequest`).
3. **REQ-003 — Frontend formulario**: `BankAccountFormPage` agrega un input de texto opcional «IBAN» (máx 34), lo incluye en el payload de alta y edición, y lo precarga al editar.
4. **REQ-004 — Frontend detalle**: `BankAccountShowPage` muestra el IBAN en la sección «Información de la cuenta» en modo lectura (solo si hay valor, o `—` si está vacío).
5. **REQ-005 — MCP alta/edición**: `bank_accounts_create` y `bank_accounts_update` aceptan el campo opcional `iban` (string); en edición, semántica parcial: si se envía `iban`, se actualiza; si se envía `""`, se limpia; si se omite, se conserva el valor actual.
6. **REQ-006 — MCP búsqueda**: `bank_accounts_search` devuelve `iban` en cada `item` de la respuesta (campo de la cuenta).
7. **REQ-007 — Contrato y documentación**: `docs/mcp/tools/contracts.json` se regenera y revisa; `docs/mcp/tools/banking.md` documenta el campo `iban` en las tres tools de cuentas.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008 — Búsqueda por IBAN**: la búsqueda de `bank_accounts_search` incluye `iban` entre los campos de coincidencia de `query` (junto a alias, número, beneficiario y banco), con la misma normalización sin mayúsculas ni acentos.
2. **REQ-009 — Detalle de la tool de alta/edición**: las descripciones de `bank_accounts_create` y `bank_accounts_update` mencionan `iban` como opcional.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010 — IBAN en la card del listado**: mostrar el IBAN en la card de cuenta de `BankDetailPage`. Se difiere: no lo pide el criterio de éxito y agrega ruido visual; si el usuario lo quiere, otra spec.
2. **REQ-011 — Variable de plantilla `account_iban`**: exponer el IBAN como variable de formatos de transferencia. Fuera de alcance: toca el catálogo de variables (`services/bank.go` y `utils/bank.ts`) y no fue solicitado.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: sin impacto medible; una columna de texto corto por cuenta, índice no requerido.
- **Seguridad**: sin cambios de auth; el IBAN no se loguea en el MCP (el logger sigue registrando método/path/status).
- **Almacenamiento**: una columna `TEXT` por cuenta; crecimiento despreciable.
- **Disponibilidad**: sin cambios; `GET /health` intacto.
- **iHost**: migración aditiva rápida; sin dependencias nuevas; sin Node en runtime.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- **Código** (explorado en el repo):
  - Modelo `internal/models/bank.go` → `BankAccount` (sin `iban` hoy).
  - Storage `internal/storage/bank.go` → `bankAccountColumns`, `scanBankAccount`, `CreateAccount`, `UpdateAccount`.
  - Service `internal/services/bank.go` → `prepareAccount` (normaliza y valida; constantes `maxBankAccount*`).
  - Handler `internal/api/bank_handlers.go` → `bankAccountRequest` + `CreateAccount`/`UpdateAccount`.
  - Frontend `BankAccountFormPage.tsx`, `BankAccountShowPage.tsx`, `types/index.ts`, `api/index.ts`, `utils/bank.ts`.
  - MCP `internal/mcp/api/banking.go` (DTOs), `internal/mcp/tools/banking/accounts.go` (inputs/handlers), `tools.go` (descripciones).
  - Manifiesto `docs/mcp/tools/impact.json` y golden `docs/mcp/tools/contracts.json`.
  - Specs relacionadas: SPEC-133 (módulo de bancos), SPEC-152 (MCP de cuentas/tarjetas).
- **Web** (investigación de IBAN, subagente `general`):
  - El IBAN es un estándar ISO 13616 con **longitud máxima de 34 caracteres alfanuméricos** (mayúsculas; el formato electrónico se escribe sin espacios ni guiones).
  - El **Banco Central de Nicaragua** aprobó el formato IBAN para Nicaragua en **octubre de 2018** (fuente: `bcn.gob.ni/informacion-sobre-el-iban`). El IBAN de Nicaragua es de **28 caracteres**: `NI` (2) + dígitos de control (2) + 4 letras del BIC del banco + 20 del número de cuenta. Ejemplo oficial: `NI79BAMC00000000000003123123`.
  - Ambos datos respaldan reservar 34 caracteres (cubre Nicaragua y cualquier IBAN extranjero).

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Campo `iban` opcional, texto, máx 34, sin validación de formato | Conservador, cumple «reconocer IBAN como columna» sin inventar reglas; cubre el IBAN NI | No detecta IBAN malformados | ✅ Seleccionada |
| B. Validar formato/checksum del IBAN (regex por país + mod-97) | Mayor rigor de datos | Inventa reglas no solicitadas; depende de un registro de países que puede cambiar; fricción | ❌ Rechazada |
| C. IBAN solo en el MCP (sin columna en la DB) | Menos cambios | La UI y la API no lo reconocen; rompe el criterio de «columna/propiedad» | ❌ Rechazada |
| D. IBAN como campo obligatorio | Datos completos | La mayoría de cuentas existentes no lo tienen; obligaría a backfill | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — `iban` opcional, texto libre, sin validación de formato ni checksum**
- **Contexto**: el alcance acordado es «las cuentas pueden tener IBAN» y «la tool lo reconoce como propiedad/columna»; no se pidió validación.
- **Decisión**: el backend solo recorta extremos (`strings.TrimSpace`) y valida longitud ≤ 34; no se aplica regex ni checksum mod-97.
- **Consecuencias**: implementación simple y sin dependencias; se acepta almacenar IBAN con formato no estandarizado si el usuario lo ingresa así. Si en el futuro se quiere validar, es una spec aparte.

**ADR-002 — Longitud máxima 34 (ISO 13616)**
- **Contexto**: el estándar fija un máximo de 34 caracteres alfanuméricos; el IBAN de Nicaragua (28) queda dentro.
- **Decisión**: constante `maxBankAccountIBAN = 34` en `services/bank.go`, alineada con las demás constantes `maxBankAccount*`.
- **Consecuencias**: rechaza valores anormalmente largos; no restringe a la longitud exacta del país para no asumir formato.

**ADR-003 — IBAN de primer nivel en backend + frontend + MCP**
- **Contexto**: el criterio de éxito exige que la tool lo reconozca, y la UI debe poder cargarlo.
- **Decisión**: una sola propiedad `iban` end-to-end (columna → modelo → storage → service → handler → DTO MCP → form/detalle), con JSON key `"iban"` y `omitempty` donde el resto de la capa lo usa.
- **Consecuencias**: una fuente de verdad; el contrato backend↔MCP cambia de forma aditiva (campo opcional nuevo).

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
[Formulario/Detalle React] --(GET/POST/PUT /api/bank-accounts, /api/banks/{id}/accounts)--> [Handler]
                                                                                              |
                                                                                              v
[Agente MCP] --(bank_accounts_create/update/search)--> [internal/mcp/api Client HTTP] --> [Service] --> [Storage] --> [SQLite bank_accounts.iban]
```

### 5.2 Componentes

#### 5.2.1 Migración `migrations/0061_add_bank_account_iban.{up,down}.sql`
- **Responsabilidad**: agregar/quitar la columna `iban`.
- **Up**: `ALTER TABLE bank_accounts ADD COLUMN iban TEXT NOT NULL DEFAULT '';`
- **Down**: `ALTER TABLE bank_accounts DROP COLUMN iban;` (reversible; sin datos críticos).

#### 5.2.2 `internal/models/bank.go`
- Agregar `IBAN string \`json:"iban"\`` a `BankAccount`.

#### 5.2.3 `internal/storage/bank.go`
- Agregar `a.iban` a `bankAccountColumns` (y ajustar `scanBankAccount`).
- Incluir `iban` en el `INSERT` de `CreateAccount` y en el `UPDATE` de `UpdateAccount`.

#### 5.2.4 `internal/services/bank.go`
- Constante `maxBankAccountIBAN = 34`.
- En `prepareAccount`: `account.IBAN = strings.TrimSpace(account.IBAN)` y validación de longitud (error si `> 34`).

#### 5.2.5 `internal/api/bank_handlers.go`
- Agregar `IBAN string \`json:"iban"\`` a `bankAccountRequest` y mapearlo en `CreateAccount`/`UpdateAccount`.

#### 5.2.6 Frontend
- `types/index.ts`: `iban: string` en `BankAccount`.
- `BankAccountFormPage.tsx`: estado `iban`, input opcional (máx 34), en payload y en `applyAccount`.
- `BankAccountShowPage.tsx`: `InfoRow` con `t('banks.iban')` y `account.iban`.
- `i18n/{es,en}.json`: clave `banks.iban` («IBAN»).

#### 5.2.7 MCP `internal/mcp/api/banking.go`
- `BankAccount`: `IBAN string \`json:"iban,omitempty"\``.
- `BankAccountInput`: `IBAN string \`json:"iban,omitempty"\``.

#### 5.2.8 MCP `internal/mcp/tools/banking/accounts.go`
- `AccountItem`: `IBAN string \`json:"iban,omitempty"\``.
- `AccountCreateInput`: `IBAN *string \`json:"iban,omitempty"\``.
- `AccountUpdateInput`: `IBAN *string \`json:"iban,omitempty"\``.
- `accountItem()`: mapear `IBAN`.
- `accountsCreate()`: `iban := strings.TrimSpace(stringValue(input.IBAN))` y pasarlo al `BankAccountInput`.
- `accountsUpdate()`: `iban := current.IBAN`; si `input.IBAN != nil`, `iban = strings.TrimSpace(*input.IBAN)` (permite limpiar con `""`); pasarlo al `BankAccountInput`.
- `accountsSearch()`: incluir `account.IBAN` en `matchScore` (REQ-008).

#### 5.2.9 MCP `internal/mcp/tools/banking/tools.go`
- Actualizar descripciones de `bank_accounts_create`/`bank_accounts_update` para mencionar `iban` opcional.

### 5.3 Modelo de datos

```
Entidad: BankAccount (tabla bank_accounts)
- iban: TEXT NOT NULL DEFAULT ''   (nuevo, opcional en la práctica, máx 34)
- (resto sin cambios)
```

No cambian `banks`, `credit_cards` ni `bank_transfer_formats`.

### 5.4 APIs / Contratos

#### Endpoint `POST /api/banks/{id}/accounts` (existente)

**Request** (se agrega `iban`):
```json
{
  "alias": "Cuenta Demo",
  "account_number": "demo-account-001",
  "account_type": "savings",
  "currency_id": 1,
  "beneficiary": "Titular Demo",
  "iban": "NI00DEMO00000000000000000000",
  "notes": "",
  "active": true
}
```

**Response 201**: la cuenta creada, ahora con `"iban": "..."`.

#### Endpoint `PUT /api/bank-accounts/{id}` (existente)

Igual que el alta: `iban` opcional. El service lo recorta y valida longitud.

#### Tool `bank_accounts_create` / `bank_accounts_update`

Se agrega el parámetro opcional `iban` (string). En `update`, omisión = conservar; `""` = limpiar.

#### Tool `bank_accounts_search`

`items[].iban` presente en la salida.

**Error de validación** (nuevo caso): `iban` con más de 34 caracteres → `400 validation_error` con mensaje «El IBAN no puede superar los 34 caracteres.».

### 5.5 Dependencias

- **Internas**: `models`, `storage`, `services`, `api` (módulo legacy de bancos); `internal/mcp/api`, `internal/mcp/tools/banking`; frontend `pages`, `types`, `api`, `i18n`.
- **Externas**: ninguna nueva.

---

## 6. Impacto en tools MCP

> Obligatorio (`DELIV-MCP-IMPACT`). El gate `./scripts/check-mcp-impact.sh` es conservador: marca por rutas compartidas (`shared.affects_all`) y por dependencias exactas. Como el cambio toca `internal/models/bank.go`, `migrations/` y `internal/mcp/api/banking.go` (todos en `shared.affects_all`), el gate reporta las 48 tools; el **cambio funcional real es aditivo y acotado a 3 tools de cuentas**. Se listan todas por nombre para declarar el impacto.

### 6.1 Tools funcionalmente afectadas

| Tool | ¿Afectada? | Motivo / ruta de dependencia | Contrato |
|---|---|---|---|
| `bank_accounts_search` | **Sí** | `iban` en `items[]` y en la búsqueda por `query`. Ruta: `bank_accounts_search` → `internal/mcp/api/banking.go` (DTO) → `GET /api/bank-accounts/{id}` / `GET /api/banks/{id}/accounts` → `internal/api/bank_handlers.go` → `internal/services/bank.go` → `internal/storage/bank.go` → `bank_accounts.iban` | Descripción actualizada; salida con `iban` |
| `bank_accounts_create` | **Sí** | Parámetro `iban` opcional. Ruta: `bank_accounts_create` → `internal/mcp/api/banking.go` → `POST /api/banks/{id}/accounts` → handler → service → storage → `bank_accounts.iban` | `input_schema` agrega `iban` |
| `bank_accounts_update` | **Sí** | Parámetro `iban` opcional (semántica parcial). Ruta: `bank_accounts_update` → `internal/mcp/api/banking.go` → `PUT /api/bank-accounts/{id}` → handler → service → storage → `bank_accounts.iban` | `input_schema` agrega `iban` |

### 6.2 Tools sin cambio funcional (solo reportadas por rutas compartidas)

> El cambio en `internal/models/bank.go`, `migrations/` y `internal/mcp/api/banking.go` es aditivo (campo opcional nuevo) y no altera el contrato de las siguientes tools. Se mencionan por nombre porque el gate conservador las reporta.

- Por dependencias del módulo de bancos (operan sobre la entidad **banco**, no sobre cuentas): `banks_create`, `entities_search`.
- Por compartir el directorio `internal/mcp/tools/banking/` y el DTO `internal/mcp/api/banking.go` (usan DTOs de **tarjeta**, sin `iban`): `credit_cards_create`, `credit_cards_search`, `credit_cards_update`.
- Por `shared.affects_all` (`internal/models/`, `migrations/`, `internal/mcp/api/`): `alerts_list`, `automation_accounts_list`, `automation_runs_list`, `autos_create`, `autos_inspection_alerts`, `autos_inspection_register`, `autos_insurance_register`, `autos_list`, `autos_maintenance_register`, `autos_update`, `bills_by_month`, `bills_generate`, `bills_list`, `bills_pay`, `bills_projection`, `bills_summary`, `credit_card_statements_get`, `credit_card_statements_list`, `debt_bills_list`, `home_maintenance_create`, `home_maintenance_get`, `home_maintenance_item_delete`, `home_maintenance_item_move`, `home_maintenance_item_update`, `home_maintenance_labor_register`, `home_maintenance_list`, `home_maintenance_supplies_register`, `home_maintenance_update`, `homes_create`, `homes_update`, `institutions_create`, `pension_catalogs_list`, `pension_month_summary`, `rental_home_status`, `rental_lease_get`, `rental_leases_list`, `rental_portfolio_summary`, `services_create`, `services_get`, `settings_status`.

- **Manifest**: `docs/mcp/tools/impact.json` ya declara las rutas `internal/mcp/api/banking.go`, `internal/mcp/tools/banking/`, `internal/api/bank_handlers.go`, `internal/services/bank.go`, `internal/storage/bank.go` para las tools de banca, y `internal/models/`/`migrations/`/`internal/mcp/api/` en `shared.affects_all`. No requiere ampliar el manifiesto.
- **Contrato**: `docs/mcp/tools/contracts.json` se regenera con `go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts` y se revisa el diff (agrega `iban` a los `input_schema` de `bank_accounts_create`/`bank_accounts_update` y a la descripción de `bank_accounts_search`). El cambio es aditivo y no rompe el contrato backend↔MCP (`TestMCPBackendContract` no toca cuentas bancarias y sigue verde).
- **Tests**: `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract`.

Validación subagente: explore — sesión ses_debfd4218ffe8IKTtKnlWOHlBM — veredicto: tools afectadas: bank_accounts_search, bank_accounts_create, bank_accounts_update (aditivo; el resto solo por rutas compartidas, sin cambio de contrato).

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [ ] CA-001: tras aplicar la migración, `PRAGMA table_info(bank_accounts)` incluye la columna `iban` con default `''`.
- [ ] CA-002: `POST /api/banks/{id}/accounts` con `iban` crea la cuenta y `GET /api/bank-accounts/{id}` devuelve `iban`.
- [ ] CA-003: `PUT /api/bank-accounts/{id}` con `iban` actualiza el valor; con `iban: ""` lo limpia; omitido lo conserva.
- [ ] CA-004: `iban` con más de 34 caracteres devuelve `400 validation_error` con mensaje claro.
- [ ] CA-005: el formulario `BankAccountFormPage` muestra el campo IBAN opcional, lo guarda en alta y edición y lo precarga al editar.
- [ ] CA-006: `BankAccountShowPage` muestra el IBAN en modo lectura (o `—` si está vacío).
- [ ] CA-007: `bank_accounts_create` acepta `iban` y lo persiste; `bank_accounts_update` aplica semántica parcial sobre `iban`; `bank_accounts_search` devuelve `iban` en cada item.
- [ ] CA-008: `bank_accounts_search` encuentra una cuenta por su `iban` vía `query` (REQ-008).
- [ ] CA-UI-I18N: la clave `banks.iban` existe como string en `es.json` y `en.json`.
- [ ] CA-UI-INPUT-TOKENS: el input de IBAN usa tokens del tema y se verificó legibilidad en darkmode.
- [ ] CA-UX-DETAIL-READONLY: el IBAN en el detalle es de solo lectura; la edición navega a `/edit`.
- [ ] CA-SEC-SYNTHETIC: ejemplos y fixtures usan IBAN sintético (`NI00DEMO00000000000000000000`), nunca un IBAN real.
- [ ] CA-DELIV-MCP-IMPACT: `./scripts/check-mcp-impact.sh --staged --gate` pasa.
- [ ] CA-DELIV-MCP-DOCS: `docs/mcp/tools/banking.md` documenta `iban` en las tres tools.

### 7.2 No funcionales

- [ ] CA-ARCH-STACK: `go.mod` sin dependencias nuevas.
- [ ] CA-ARCH-GO: `gofmt -l` sin archivos y `go vet ./...` limpio.
- [ ] CA-ARCH-IHOST: el server sigue respondiendo `GET /health`.
- [ ] CA-SEC-NO-PII: `git diff --cached` sin IBAN, números de cuenta ni datos reales.
- [ ] CA-DATA-MIGRATIONS: la migración tiene `.down.sql` reversible y se probó sobre copia de producción.
- [ ] CA-TEST-LOCAL: `go build ./...` y `go test ./...` pasan; `cd frontend && npm run check:ui && npm run check:i18n && npm run check:numbers` pasa.
- [ ] CA-TEST-MCP-CONTRACT: `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract` verdes.

### 7.3 Testing

- **Unit tests (Go)**: `prepareAccount` acepta `iban` vacío/≤34 y rechaza `>34`; `CreateAccount`/`UpdateAccount` persisten `iban`; `scanBankAccount` mapea `iban`.
- **Unit tests (MCP)**: `accountsCreate`/`accountsUpdate` pasan `iban`; `accountsUpdate` conserva/limpia/actualiza según omisión/`""`/valor; `accountsSearch` matchea por `iban`; `accountItem` expone `iban`.
- **Contract tests**: `TestToolContracts` (golden `contracts.json`) y `TestMCPBackendContract` (cadena real API key → handler → storage → JSON).
- **UI (manual + guardrails)**: `check:ui`, `check:i18n`, `check:numbers`; verificación visual del formulario y detalle en light y darkmode.
- **Datos**: validación del alta/edición contra una copia de producción en `/tmp` (`DATA-PRODCOPY`), nunca contra `data/app.db`.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0061` + modelo + storage (columns/scan/insert/update) | 0.5 día | Ninguna |
| 2 | Service (`maxBankAccountIBAN` + validación) + handler (`iban` en request) | 0.5 día | Fase 1 |
| 3 | Frontend: tipo, formulario, detalle e i18n | 0.5 día | Fase 2 |
| 4 | MCP: DTO, inputs/handlers, descripciones | 0.5 día | Fase 2 |
| 5 | Tests (unit + contract), regenerar golden y docs MCP (subagente) | 1 día | Fases 3 y 4 |

### 8.2 Milestones

1. **MVP**: IBAN persistido y visible en backend + UI + MCP (alta/edición/búsqueda), con migración reversible.
2. **V1.0**: tests, contrato regenerado, documentación MCP y release.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Scope creep de validación de IBAN (regex/checksum) | Media | Bajo | ADR-001/002: sin validación de formato; solo longitud |
| Migración sobre producción | Baja | Alto | Reversible (`.down.sql`); validación contra copia en `/tmp` (`DATA-PRODCOPY`) |
| Drift del golden `contracts.json` | Media | Medio | Regenerar con `-update-contracts`, revisar diff y declararlo (`TEST-MCP-CONTRACT`) |
| `bank_accounts_update` pise `iban` no enviado | Media | Alto | Semántica parcial: leer entidad, mezclar solo campos enviados (mismo patrón de SPEC-152) |
| Falsa alarma del gate de impacto (rutas compartidas) | Alta | Bajo | Sección 6 declara tools funcionalmente afectadas vs. reportadas por `shared.affects_all` |
| i18n sin paridad es/en | Baja | Bajo | `check:i18n` en build y pre-commit |

---

## 10. Notas y Referencias

- Reglas: `docs/rules/00-index.md`, `10-architecture.md`, `20-ui.md`, `30-ux.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- Specs relacionadas: SPEC-133 (módulo de bancos), SPEC-152 (MCP de cuentas/tarjetas), SPEC-134/135 (UI de bancos).
- IBAN: ISO 13616 (máx 34 caracteres); Banco Central de Nicaragua — `bcn.gob.ni/informacion-sobre-el-iban` (formato IBAN aprobado en octubre de 2018, 28 caracteres); SWIFT IBAN Registry (entrada NI).
- Endpoints: ver 5.4 y `internal/api/routes.go`.
- Documentación MCP: `docs/mcp/tools/banking.md`, `docs/mcp/tools/impact.json`, `docs/mcp/tools/contracts.json`.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-25 | opencode | Creación inicial de la especificación (IBAN en cuentas bancarias y tools MCP) |
| 2026-11-25 | opencode | Pasa a `pending_execution`; issue de GitHub #164 |
| 2026-11-25 | opencode | Implementación completa (migración 0061, backend, frontend, MCP y docs); pasa a `in_progress` |
| 2026-11-25 | DevOps | QA y Security aprobados; preparada para release |
