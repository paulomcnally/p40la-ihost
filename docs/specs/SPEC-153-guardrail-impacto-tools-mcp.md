---
title: "Guardrail de impacto en tools MCP al crear specs y features"
id: "SPEC-153"
status: "in_progress"
author: "opencode"
created: "2026-11-24"
updated: "2026-11-24"
github_issue: 156
---

# Guardrail de impacto en tools MCP al crear specs y features

**ID**: SPEC-153<br>
**Estado**: in_progress<br>
**Autor**: opencode<br>
**Creado**: 2026-11-24<br>
**Actualizado**: 2026-11-24

---

## 1. Resumen Ejecutivo

El MCP de p40la-ihost crece por dominios: facturas (SPEC-145), alta de entidades (SPEC-146), búsqueda (SPEC-150), autos (SPEC-151) y cuentas/tarjetas (SPEC-152). Cada tool depende de una cadena frágil: registro en `internal/mcp/tools`, cliente HTTP, endpoint del backend, handler, servicio, storage y tablas SQLite. Hoy un cambio en cualquiera de esos eslabones puede romper una tool existente sin que ningún guardrail lo detecte: `check-mcp-docs.sh` solo exige documentación, y los tests actuales verifican que las tools existan, no que su contrato y sus dependencias sigan intactos.

Esta spec agrega un **guardrail de impacto en tools MCP**. Un manifiesto versionado declara las dependencias de cada tool; un analizador determinista cruza los archivos cambiados contra ese manifiesto y reporta qué tools se verían afectadas; y el hook `pre-commit` exige que la spec en curso declare el impacto y lo valide un subagente. Además, un test de contrato congela nombres, descripciones, esquemas de entrada y anotaciones de `tools/list`, para que cualquier cambio de contrato sea explícito.

El impacto en el iHost es **nulo en runtime**: el manifiesto, el analizador y el test de contrato viven en el repositorio y se ejecutan en desarrollo/CI. No se agregan dependencias, no se tocan migraciones ni el binario del servidor, y el MCP sigue usando la misma API HTTP y SQLite.

---

## 2. Cumplimiento de reglas

Reglas aplicables según `docs/rules/00-index.md`. La spec no toca frontend, rutas de UI, modales ni el esquema de la base de datos.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Solo Go stdlib para el analizador y el test; sin dependencias nuevas | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Manifiesto y contrato en `docs/mcp/tools/`; analizador en `scripts/mcp-impact/`; hook en `.githooks/` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | No se agrega lógica de negocio al server; el guardrail es tooling de desarrollo | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | N/A: no se crean módulos de dominio nuevos | — |
| `ARCH-GO` | `gofmt`, errores explícitos, nombres en inglés, tests con `go test` | `CA-ARCH-GO` |
| `ARCH-IHOST` | Sin cambios en el binario, la imagen ni el consumo de RAM del iHost; el guardrail corre fuera del iHost | `CA-ARCH-IHOST` |
| `ARCH-MCP-TOOLS-MAP` | Regla nueva: todo `mcp.AddTool` debe estar en `docs/mcp/tools/impact.json`; `--check-map` lo verifica | `CA-ARCH-MCP-TOOLS-MAP` |
| `DATA-MIGRATIONS` | N/A: no hay migraciones | — |
| `DATA-NO-DROP` | No se ejecuta ningún comando sobre `data/app.db`; las pruebas usan fixtures sintéticos | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | N/A: el guardrail no valida datos reales ni consulta producción | — |
| `SEC-NO-PII` | Manifiesto, contratos, docs y tests solo con datos sintéticos; no se copian payloads reales | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `demo@example.com`, `Servicio Demo`, `demo-account-001`, `F000000000000001` | `CA-SEC-SYNTHETIC` |
| `SEC-COMMIT` / `SEC-HOOKS` | El hook nuevo es fail-safe y no reemplaza la revisión de `git diff --cached`; sin `--no-verify` | `CA-SEC-COMMIT` |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` verdes; tests unitarios del analizador y del contrato | `CA-TEST-LOCAL` |
| `TEST-SERVER` | N/A: no se levanta el server para este cambio; si se prueba el hook, se usa `--files` | — |
| `TEST-USER-FLOW` | No se pasa a `pending_release` sin pruebas del usuario | `CA-TEST-USER-FLOW` |
| `TEST-MCP-CONTRACT` | Regla nueva: `TestToolContracts` congela `tools/list` y `TestMCPBackendContract` verifica la cadena backend↔MCP real | `CA-TEST-MCP-CONTRACT` |
| `DELIV-WORKTREE` | Trabajo exclusivo en `../p40la-ihost-spec-153`, rama `feature/SPEC-153` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge y push solo con confirmación explícita; sin `--force` | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Commits `SPEC-153: descripción corta` | `CA-DELIV-COMMIT` |
| `DELIV-HOOKS` | `pre-commit` ejecuta `./scripts/check-mcp-impact.sh --staged --gate` y falla de forma explícita | `CA-DELIV-HOOKS` |
| `DELIV-MCP-DOCS` | El mismo commit actualiza `docs/mcp/` (`tools/impact.md`, contrato y catálogo) | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La documentación de `docs/mcp/` se delega a un subagente y se revisa | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-MCP-IMPACT` | Regla nueva: toda spec incluye `## Impacto en tools MCP` y el gate lo valida | `CA-DELIV-MCP-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Regla nueva: el análisis de impacto lo valida un subagente y queda registrado en la spec | `CA-DELIV-MCP-IMPACT-SUBAGENT` |
| `DELIV-LABELS` | Issue con `spec/draft`; transición a `spec/in-progress` al implementar | `CA-DELIV-LABELS` |
| `UI-*` / `UX-*` | N/A: no hay cambios de frontend, rutas, pantallas ni modales | — |

> Las reglas `ARCH-MCP-TOOLS-MAP`, `DELIV-MCP-IMPACT`, `DELIV-MCP-IMPACT-SUBAGENT` y `TEST-MCP-CONTRACT` se crean en esta spec antes de implementar el código.

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Crear `docs/mcp/tools/impact.json` como fuente de verdad de las tools registradas, sus archivos de entrada, dependencias transitivas, endpoints, tests y documentación.
2. **REQ-002**: Crear un analizador determinista (`scripts/mcp-impact/` + `scripts/check-mcp-impact.sh`) con modos `--check-map`, `--files`, `--staged`, `--gate`, `--json` y `--spec`.
3. **REQ-003**: Integrar el analizador en `.githooks/pre-commit`: si hay superficie MCP afectada, el commit se bloquea salvo que la spec en curso declare el impacto, todas las tools afectadas y la validación de un subagente; además corre `go test ./internal/mcp/...`.
4. **REQ-004**: Agregar la sección obligatoria `## Impacto en tools MCP` al template de specs y al flujo `/spec-create` de la skill `spec-manager`, con validación por subagente.
5. **REQ-005**: Agregar un test de contrato (`TestToolContracts`) que compare `tools/list` contra `docs/mcp/tools/contracts.json` (nombre, descripción, esquema de entrada, esquema de salida y anotaciones) y un test backend↔MCP (`TestMCPBackendContract`) que use el router real y una DB temporal para verificar auth, rutas, tags JSON y `source=mcp`.
6. **REQ-006**: Actualizar las reglas del proyecto, `AGENTS.md`, el plugin `rules-guard`, el template y la documentación MCP para que el guardrail sea descubrible y aplicable.
7. **REQ-007**: Detectar automáticamente tools nuevas o eliminadas comparando el código (`mcp.AddTool`) contra el manifiesto; una tool sin manifiesto es un error bloqueante.
8. **REQ-008**: El plugin de OpenCode debe bloquear la creación de una spec nueva sin la sección de impacto y la creación de una tool sin entrada en el manifiesto.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Salida humana clara (tools afectadas + motivo + archivo) y salida `--json` para futuras integraciones de CI.
2. **REQ-010**: Tests unitarios del analizador para matching, manifiesto, tools registradas y parsing de la sección de impacto.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-011**: Integración futura en CI (GitHub Actions) usando `--staged` o `--files` con el diff del PR.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: `--check-map` y `--files` deben resolver en menos de 2 segundos en un checkout local; el hook no debe ejecutar tests del server completo, solo `./internal/mcp/...`.
- **Seguridad**: sin secretos, PII ni datos reales en manifiesto, contratos, fixtures, specs, issues ni mensajes de commit.
- **Almacenamiento**: el manifiesto y el contrato son JSON pequeños (estimado < 20 KB); no se guardan artefactos generados fuera del repo.
- **Disponibilidad**: el guardrail es de desarrollo; un fallo del plugin es fail-safe y no bloquea la sesión. El hook solo bloquea patrones positivos (impacto no declarado, contrato desactualizado, tests rojos).
- **iHost**: cero cambios en el runtime, la imagen, SQLite o el consumo de memoria del iHost.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- `scripts/check-mcp-docs.sh` solo verifica que un cambio en la superficie MCP incluya cambios en `docs/mcp/`; no analiza dependencias ni contratos.
- `internal/mcp/tools/bills/protocol_test.go` verifica que las tools existan y que una llamada funcione; no congela esquemas ni detecta cambios de comportamiento en el backend.
- `internal/mcp/tools/bills/tools_test.go` usa una API falsa; no cubre la cadena real handler → servicio → storage.
- El pre-commit actual ya tiene guardrails de UI, i18n, números, documentación MCP y secretos; el nuevo guardrail se suma con el mismo patrón fail-safe.
- El plugin `rules-guard` ya intercepta `write`/`edit`/`patch` y comandos de shell; es el lugar natural para bloquear specs nuevas sin sección de impacto y tools nuevas sin manifiesto.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. Solo documentar el proceso en la skill | Cero código | No previene regresiones; depende de la memoria del agente | ❌ Rechazada |
| B. Manifiesto estático + analizador + hook + contrato | Determinista, barato, sin dependencias, auditable | Mantenimiento del manifiesto al agregar tools | ✅ Seleccionada |
| C. Grafo dinámico con `go/packages` y análisis de llamadas | Mayor precisión | Agrega dependencia `golang.org/x/tools`, más lento y frágil en un repo grande | ❌ Rechazada |
| D. Solo tests de integración contra el server real | Detecta regresiones reales | Requiere DB, server y datos; lento y no apto para pre-commit | ❌ Rechazada |
| E. Contrato con snapshot de `tools/list` | Detecta cambios de schema/descripción/anotaciones | No detecta cambios de comportamiento del backend | ✅ Complemento de B |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Manifiesto estático en JSON, no en Go**
- **Contexto**: el guardrail debe ser legible por humanos, por el analizador Go y por el plugin Node.
- **Decisión**: `docs/mcp/tools/impact.json` con `version`, `tools[]` y `shared.affects_all[]`.
- **Consecuencias**: una sola fuente de verdad; el plugin puede leerlo sin compilar Go. Requiere actualizarlo al agregar tools (lo exige `--check-map`).

**ADR-002: Matching por archivo/patrón, no por AST de llamadas**
- **Contexto**: el repo usa capas claras y los tools dependen de archivos identificables; un grafo de llamadas con `go/packages` agrega dependencia y fragilidad.
- **Decisión**: cada tool declara `entrypoints`, `dependencies` (archivos, directorios con `/` final o globs `*`) y el manifiesto declara rutas compartidas que afectan a todas.
- **Consecuencias**: simple y rápido; puede dar falsos positivos cuando un archivo compartido cambia sin alterar la tool. El sesgo es deliberado: preferimos revisar de más antes que romper una tool.

**ADR-003: El contrato se congela con un golden file**
- **Contexto**: un cambio de schema o de descripción puede romper clientes MCP sin que falle ningún test funcional.
- **Decisión**: `TestToolContracts` compara `tools/list` contra `docs/mcp/tools/contracts.json`; se regenera solo con `-update-contracts`.
- **Consecuencias**: todo cambio de contrato es visible en el diff y exige una decisión explícita; el golden debe revisarse, no aceptarse a ciegas.

**ADR-004: La validación de impacto es un requisito de proceso con evidencia en la spec**
- **Contexto**: un script no puede saber si un cambio de comportamiento es intencional; un subagente puede leer la feature y razonar sobre efectos colaterales.
- **Decisión**: la sección `## Impacto en tools MCP` incluye tools afectadas, ruta de dependencia, contrato y una línea `Validación subagente:` con agente, sesión y veredicto.
- **Consecuencias**: el gate puede verificar la evidencia; la calidad del veredicto sigue dependiendo del subagente y de la revisión del agente principal.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
[Spec nueva / feature]
        │
        ▼
[Subagente explore/general] ──► veredicto de impacto
        │
        ▼
[docs/specs/SPEC-XXX.md]  ──► sección "Impacto en tools MCP"
        │
        ▼
[git add] ──► [.githooks/pre-commit]
        │
        ├──► scripts/check-mcp-impact.sh --staged --gate
        │        ├── lee docs/mcp/tools/impact.json
        │        ├── cruza archivos cambiados → tools afectadas
        │        ├── valida la spec en curso
        │        └── go test ./internal/mcp/...
        │
        └──► scripts/check-mcp-docs.sh --staged
                 └── exige docs/mcp/ si cambió la superficie MCP

[go test ./internal/mcp/...] ──► TestToolContracts
        └── compara tools/list contra docs/mcp/tools/contracts.json
```

### 5.2 Componentes

#### 5.2.1 Manifiesto de impacto

- **Responsabilidad**: declarar tools, dependencias, endpoints, tests y documentación; declarar rutas compartidas que afectan a todas.
- **Interfaz**: JSON versionado.
- **Dependencias**: ninguna.
- **Ubicación**: `docs/mcp/tools/impact.json`.

Ejemplo sintético:

```json
{
  "version": 1,
  "tools": [
    {
      "name": "bills_list",
      "docs": ["docs/mcp/tools/bills.md"],
      "entrypoints": ["internal/mcp/tools/bills/list.go"],
      "dependencies": ["internal/api/bill_handlers.go", "internal/services/bill.go"],
      "endpoints": ["GET /api/services/{id}", "GET /api/services/{id}/bills"],
      "tests": ["internal/mcp/tools/bills/tools_test.go"]
    }
  ],
  "shared": {
    "affects_all": ["internal/mcp/server/", "internal/models/", "migrations/"]
  }
}
```

#### 5.2.2 Analizador `scripts/mcp-impact`

- **Responsabilidad**: cargar el manifiesto, validar su cobertura contra `mcp.AddTool`, calcular impacto y aplicar el gate.
- **Interfaz**:
  - `--check-map`: valida manifiesto ↔ código registrado y existencia de rutas.
  - `--files <ruta>...`: calcula impacto para archivos explícitos.
  - `--staged`: usa `git diff --cached --name-only --diff-filter=ACMR`.
  - `--gate`: exige spec con impacto declarado, tools mencionadas y validación de subagente; valida toda spec nueva staged; corre `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract`.
  - `--json`: salida estructurada.
  - `--spec <ruta>`: fuerza la spec a validar (útil en CI).
- **Dependencias**: Go stdlib (`go/parser`, `go/ast`, `encoding/json`, `os/exec`).
- **Ubicación**: `scripts/mcp-impact/main.go`, `scripts/check-mcp-impact.sh`.

#### 5.2.3 Test de contrato

- **Responsabilidad**: congelar el contrato visible de `tools/list` y verificar la cadena backend↔MCP real.
- **Interfaz**:
  - `TestToolContracts` en `internal/mcp/tools/all/contract_test.go`; flag `-update-contracts`.
  - `TestMCPBackendContract` en `internal/api/mcp_backend_contract_test.go`; DB temporal, router real, API key real y cliente MCP real.
- **Dependencias**: `modelcontextprotocol/go-sdk` ya presente.
- **Ubicación**: `internal/mcp/tools/all/contract_test.go`, `internal/api/mcp_backend_contract_test.go`, `docs/mcp/tools/contracts.json`.

#### 5.2.4 Hook de pre-commit

- **Responsabilidad**: bloquear commits con impacto MCP no declarado o contrato roto.
- **Interfaz**: sección 8 de `.githooks/pre-commit`.
- **Dependencias**: `scripts/check-mcp-impact.sh`, `go`.
- **Ubicación**: `.githooks/pre-commit`.

#### 5.2.5 Plugin `rules-guard`

- **Responsabilidad**: prevención en tiempo de escritura.
- **Interfaz**:
  - spec nueva (`docs/specs/SPEC-*.md` que no existe) sin `## Impacto en tools MCP` o sin `Validación subagente:` → bloquea;
  - archivo de tools con `mcp.AddTool` y un `Name` ausente del manifiesto → bloquea;
  - banner de sesión menciona el guardrail.
- **Dependencias**: `node:fs`, `node:path`.
- **Ubicación**: `.opencode/plugins/rules-guard/index.js`.

#### 5.2.6 Skill y template

- **Responsabilidad**: hacer que `/spec-create` ejecute el análisis de impacto con un subagente y complete la sección.
- **Interfaz**: instrucciones de la skill + sección del template.
- **Ubicación**: `.opencode/skills/spec-manager/SKILL.md`, `docs/specs/templates/spec-template.md`.

### 5.3 Modelo de datos

No hay cambios en SQLite. El manifiesto y el contrato son archivos JSON versionados:

```
Manifiesto
- version: int
- tools[]: name, docs[], entrypoints[], dependencies[], endpoints[], tests[]
- shared.affects_all[]: patrones de archivos/directorios

Contrato
- version: int
- generated_by: string
- tools[]: name, description, input_schema, output_schema?, annotations?
```

### 5.4 APIs / Contratos

No se agregan endpoints HTTP. El contrato interno es la interfaz CLI del analizador:

```
scripts/check-mcp-impact.sh --check-map
scripts/check-mcp-impact.sh --files <archivo>...
scripts/check-mcp-impact.sh --staged
scripts/check-mcp-impact.sh --staged --gate
scripts/check-mcp-impact.sh --staged --json
```

Salida humana esperada:

```
[mcp-impact] tools afectadas: bills_generate, bills_pay
  - bills_generate: dependencia internal/services/bill_bulk.go (cambio en internal/services/bill_bulk.go)
  - bills_pay: dependencia internal/services/bill_bulk.go (cambio en internal/services/bill_bulk.go)
[mcp-impact] ✓ docs/specs/SPEC-153-guardrail-impacto-tools-mcp.md declara impacto y validación subagente; tests MCP y contrato backend verdes
```

### 5.5 Dependencias

- **Internas**: `docs/mcp/tools/impact.json`, `docs/mcp/tools/contracts.json`, `docs/mcp/tools/impact.md`, `.githooks/pre-commit`, `internal/mcp/tools/all/contract_test.go`, `internal/api/mcp_backend_contract_test.go`, `.opencode/plugins/rules-guard/index.js`, skill y template.
- **Externas**: ninguna nueva. Solo Go stdlib y el SDK MCP ya presente.

---

## 6. Impacto en tools MCP

> Análisis obligatorio (regla `DELIV-MCP-IMPACT`). El manifiesto `docs/mcp/tools/impact.json` se completa en esta spec.

| Tool | ¿Afectada? | Motivo / ruta de dependencia | Contrato |
|---|---|---|---|
| `bills_list` | No | No se modifica su handler, schema, cliente, endpoint ni dependencias de backend; solo se agrega cobertura de contrato | Sin cambios |
| `bills_generate` | No | No se modifica su handler, schema, cliente, endpoint ni dependencias de backend; solo se agrega cobertura de contrato | Sin cambios |
| `bills_pay` | No | No se modifica su handler, schema, cliente, endpoint ni dependencias de backend; solo se agrega cobertura de contrato | Sin cambios |

Ninguna tool existente cambia su contrato. La tabla lista las tres tools de facturas; tras el merge con `main`, el catálogo incluye además `homes_create`, `institutions_create`, `banks_create`, `services_create` y `entities_search`. El manifiesto y el golden cubren las ocho sin que esta spec modifique su comportamiento.

**Validación subagente**: `explore` (sesión `ses_df33b054bffeAmsAjIt1f7vESs`) — veredicto: el binario MCP no importa el backend; la dependencia real es HTTP/JSON. No se modifica ningún handler, schema ni contrato de las tools existentes. El subagente confirmó la cadena de dependencias y recomendó mapear también `internal/api/handlers.go`, `internal/api/middleware.go`, `internal/storage/system_settings.go`, `internal/services/api_key.go`, `internal/config/`, `internal/db/` y `migrations/`; esos paths quedaron en el manifiesto. También recomendó un contrato backend↔MCP como siguiente capa; se implementó como `TestMCPBackendContract`.
**Manifest**: `docs/mcp/tools/impact.json`
**Contrato**: `docs/mcp/tools/contracts.json`

> El cambio agrega tooling y tests; no altera el comportamiento de las tools existentes. Si el subagente detecta una dependencia no mapeada, se actualiza el manifiesto antes de implementar.

---

## 7. Criterios de Aceptación

### 7.1 Funcionales

- [ ] CA-ARCH-MCP-TOOLS-MAP: `./scripts/check-mcp-impact.sh --check-map` falla si una tool registrada con `mcp.AddTool` no está en el manifiesto, o si el manifiesto declara una tool que el código no registra.
- [ ] CA-DELIV-MCP-IMPACT-001: `./scripts/check-mcp-impact.sh --files internal/services/bill_bulk.go` reporta `bills_generate` y `bills_pay`, y no reporta `bills_list`.
- [ ] CA-DELIV-MCP-IMPACT-002: `--files internal/api/routes.go` reporta las ocho tools por ser una dependencia compartida.
- [ ] CA-DELIV-MCP-IMPACT-003: `--files internal/api/bill_handlers.go` reporta solo `bills_list`.
- [ ] CA-DELIV-MCP-IMPACT-004: `--staged --gate` bloquea si la spec en curso no existe, no tiene `## Impacto en tools MCP`, no menciona una tool afectada o no registra `Validación subagente:`.
- [ ] CA-DELIV-MCP-IMPACT-006: `--staged --gate` bloquea una spec nueva sin sección de impacto o sin validación subagente, aunque no haya tools afectadas.
- [ ] CA-DELIV-MCP-IMPACT-005: una spec nueva sin la sección de impacto es bloqueada por el plugin `rules-guard`.
- [ ] CA-TEST-MCP-CONTRACT: `go test ./internal/mcp/...` pasa con `TestToolContracts`; un cambio de schema/descripción/anotaciones falla hasta regenerar `contracts.json` con `-update-contracts`.
- [ ] CA-TEST-MCP-CONTRACT-002: `go test ./internal/api -run TestMCPBackendContract` pasa y detecta drift de auth, rutas, tags JSON o `source=mcp` en la cadena backend↔MCP.
- [ ] CA-ARCH-MCP-TOOLS-MAP-002: el manifiesto se valida con `--check-map` y cada `docs`, `entrypoints`, `dependencies` y `tests` apunta a un archivo o patrón existente.
- [ ] CA-DELIV-HOOKS: `.githooks/pre-commit` ejecuta el guardrail y muestra un mensaje accionable; el hook es fail-safe si el script no existe.
- [ ] CA-SEC-SYNTHETIC: manifiesto, contrato, tests, docs y ejemplos usan solo datos sintéticos.
- [ ] CA-DELIV-MCP-DOCS: `docs/mcp/` se actualiza en el mismo commit con `tools/impact.md`, el catálogo y las referencias a `contracts.json`.
- [ ] CA-DELIV-MCP-DOCS-SUBAGENT: la documentación MCP se delega a un subagente y el agente principal la revisa.
- [ ] CA-DELIV-MCP-IMPACT-SUBAGENT: la sección de impacto registra agente, sesión y veredicto del subagente.

### 7.2 No funcionales

- [ ] CA-NF-001: `--check-map` y `--files` resuelven en menos de 2 segundos en un checkout local.
- [ ] CA-NF-002: no se agregan dependencias a `go.mod`/`go.sum`.
- [ ] CA-NF-003: no hay cambios en el binario del servidor, la imagen Docker ni el consumo de RAM del iHost.
- [ ] CA-NF-004: `go build ./...`, `go test ./...` y `node --check .opencode/plugins/rules-guard/index.js` pasan.

### 7.3 Testing

- **Unit tests**: matching de archivos/patrones, carga y validación del manifiesto, tools registradas vs manifiesto, parsing de la sección de impacto.
- **Integration tests**: `go test ./internal/mcp/...` con el contrato congelado; `go test ./internal/api -run TestMCPBackendContract` con DB temporal y router real; simulación de archivos cambiados con `--files`.
- **E2E tests**: N/A; no hay flujo de usuario.
- **Carga/Performance**: verificar que `--check-map` y `--files` no tardan más de 2 segundos.

---

## 8. Plan de Implementación

### 8.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Manifiesto `impact.json` + analizador Go + tests unitarios | 1 día | Ninguna |
| 2 | Tests de contrato `tools/list` y backend↔MCP + `contracts.json` generado y revisado | 1 día | Fase 1 |
| 3 | Hook de pre-commit + plugin `rules-guard` | 0.5 día | Fases 1-2 |
| 4 | Reglas, AGENTS.md, template y skill | 0.5 día | Fase 3 |
| 5 | Documentación MCP delegada a subagente | 0.5 día | Fases 1-4 |
| 6 | Validación local (`go build`, `go test`, `--check-map`, `--files`, `--gate`) | 0.5 día | Fases 1-5 |

### 8.2 Milestones

1. **MVP**: manifiesto + analizador + `--check-map` + tests unitarios.
2. **V1.0**: contrato congelado, hook, plugin, reglas, template, skill y documentación.

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Falsos positivos por rutas compartidas amplias | Media | Medio | Manifiesto granular; la spec puede declarar el impacto y justificar que no hay cambio de contrato |
| Falsos negativos por una dependencia no mapeada | Media | Alto | Test de contrato + subagente + revisión del manifiesto; `--check-map` exige que cada tool esté declarada |
| Fricción del hook en commits pequeños | Media | Bajo | Analizador rápido, mensajes claros y `--files` para diagnóstico; solo bloquea si hay superficie MCP |
| Golden de contrato aceptado sin revisar | Baja | Alto | El diff de `contracts.json` se revisa; el flag de actualización es explícito y queda en el historial |
| Plugin bloquea una spec legacy al editarla | Baja | Medio | El bloqueo de sección solo aplica a specs nuevas; las existentes se pueden editar y el gate las evalúa solo si hay impacto MCP |
| Mantenimiento del manifiesto al agregar tools | Alta | Medio | `--check-map` falla ruidosamente; la documentación de `docs/mcp/development.md` explica el paso |

---

## 10. Notas y Referencias

- Reglas: `docs/rules/10-architecture.md` (`ARCH-MCP-TOOLS-MAP`), `docs/rules/60-testing.md` (`TEST-MCP-CONTRACT`), `docs/rules/70-delivery.md` (`DELIV-MCP-IMPACT`, `DELIV-MCP-IMPACT-SUBAGENT`).
- Documentación MCP: `docs/mcp/development.md`, `docs/mcp/tools/README.md`, `docs/mcp/architecture.md`.
- Precedentes: `scripts/check-mcp-docs.sh` (SPEC-145), `internal/mcp/tools/bills/protocol_test.go`, `.opencode/plugins/rules-guard/` (SPEC-136).
- Specs relacionadas: SPEC-145 (MCP de facturas), SPEC-146 (alta de entidades), SPEC-150 (búsqueda), SPEC-151 (autos), SPEC-152 (cuentas y tarjetas).

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-24 | opencode | Creación inicial de la especificación |
| 2026-11-24 | opencode | Transición a `pending_execution` y luego a `in_progress` |
| 2026-11-24 | opencode | Implementación: manifiesto, analizador, contrato `tools/list`, contrato backend↔MCP, hook, plugin, reglas, template, skill y docs. `go build ./...`, `go test ./...`, `--check-map` y `--staged --gate` verdes |
| 2026-11-24 | opencode | Documentación MCP (`docs/mcp/tools/impact.md`, catálogo, desarrollo y arquitectura) delegada al subagente `general` y revisada (`DELIV-MCP-DOCS-SUBAGENT`) |
