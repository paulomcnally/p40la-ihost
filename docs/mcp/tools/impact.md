---
title: "Impacto y contrato de tools"
sidebar_position: 3
description: "Manifiesto de impacto, contrato congelado y guardrails de las tools MCP"
---

# Impacto y contrato de tools

## Para qué sirve

Cada tool MCP depende de una cadena de archivos: registro en `internal/mcp/tools`, cliente HTTP, endpoint del backend, handler, servicio, storage y tablas SQLite. Un cambio en cualquier eslabón puede romper una tool sin tocar su entrypoint ni sus tests funcionales.

Este guardrail cubre dos huecos:

1. **Dependencias invisibles.** Un cambio en `internal/storage/bill.go` puede alterar lo que devuelve `bills_list` aunque `internal/mcp/tools/bills/list.go` no se modifique. El manifiesto `docs/mcp/tools/impact.json` declara esas dependencias y el analizador informa qué tools se ven afectadas.
2. **Contrato silencioso.** Un cambio de nombre, descripción, schema o anotación en `tools/list` rompe clientes MCP sin fallar ningún test funcional. Además, un cambio de ruta, tag JSON, auth o `source=mcp` en la cadena backend↔MCP rompe la tool aunque su schema no cambie. `docs/mcp/tools/contracts.json` congela el contrato visible y `TestMCPBackendContract` verifica la cadena real.

Todos estos controles son de **desarrollo**: viven en el repositorio, se ejecutan en el hook o en `go test`, no corren en el iHost y no cambian el runtime del servidor ni del MCP.

### Caso de ejemplo

Se agrega a `bills` la cuenta de pago y el cambio toca `internal/storage/bill.go` y `internal/storage/account.go`. El autor piensa en `bills_pay`, que registra el pago desde la cuenta `demo-account-001`, pero `bills_list` también lee `internal/storage/bill.go` y puede devolver importes incorrectos para `Servicio Demo` (`service_id: 12`). Sin manifiesto, el segundo efecto pasa desapercibido; con el guardrail, el análisis reporta `bills_list` y `bills_pay` antes de escribir código.

## Manifiesto `docs/mcp/tools/impact.json`

Es la **fuente de verdad** del impacto. Lo leen el analizador, el hook y el plugin de OpenCode.

| Campo | Tipo | Obligatorio | Regla |
|---|---|---|---|
| `version` | integer | Sí | Debe ser `1`. |
| `description` | string | No | Texto para humanos. |
| `tools[]` | array | Sí, no vacío | Una entrada por tool registrada con `mcp.AddTool`. |
| `tools[].name` | string | Sí | `^[a-z][a-z0-9_]*$`, sin duplicados. |
| `tools[].docs` | string[] | Sí, no vacío | Archivos exactos existentes (por ejemplo `docs/mcp/tools/bills.md`). |
| `tools[].entrypoints` | string[] | Sí, no vacío | Archivos exactos donde se registra la tool. |
| `tools[].dependencies` | string[] | Sí, no vacío | Archivo exacto, directorio con `/` final o glob (`*`, `?`, `[...]`). |
| `tools[].endpoints` | string[] | Sí, no vacío | Formato `METHOD /ruta` con `GET`, `POST`, `PUT`, `PATCH` o `DELETE`. |
| `tools[].tests` | string[] | Sí, no vacío | Archivos exactos de test que cubren la tool. |
| `shared.affects_all` | string[] | Sí, no vacío | Rutas que, si cambian, afectan a **todas** las tools. |
| `shared.notes` | string | No | Aclaración para humanos. |

El JSON se decodifica con campos desconocidos prohibidos: un campo mal escrito o de más hace fallar `--check-map`. Cada `docs`, `entrypoints` y `tests` debe ser un archivo existente; si no, el manifiesto es inválido.

El analizador descubre las tools registradas con `go/parser`: solo reconoce `mcp.AddTool(server, &mcp.Tool{Name: "..."}, ...)` con el nombre **literal** en un archivo de `internal/mcp/tools/`. Un nombre armado con variables o constantes no se detecta y `--check-map` falla porque la tool registrada no aparece en el manifiesto.

### Ejemplo (extracto)

```json
{
  "version": 1,
  "description": "Mapa de impacto de las tools MCP.",
  "tools": [
    {
      "name": "bills_list",
      "docs": ["docs/mcp/tools/bills.md"],
      "entrypoints": ["internal/mcp/tools/bills/list.go"],
      "dependencies": [
        "internal/api/bill_handlers.go",
        "internal/api/service_handlers.go",
        "internal/services/bill.go",
        "internal/storage/bill.go",
        "internal/storage/service_bill.go"
      ],
      "endpoints": [
        "GET /api/services/{id}",
        "GET /api/services/{id}/bills"
      ],
      "tests": [
        "internal/mcp/tools/bills/tools_test.go",
        "internal/mcp/tools/all/contract_test.go"
      ]
    }
  ],
  "shared": {
    "affects_all": [
      "internal/mcp/server/",
      "internal/models/",
      "migrations/"
    ]
  }
}
```

Es un extracto: el manifiesto real declara las **42 tools** del catálogo (24 de lectura y 18 de escritura) y todas las rutas compartidas. `--check-map` falla si una tool registrada falta en el manifiesto, si el manifiesto declara una tool que el código no registra o si un archivo o patrón declarado no existe.

### Cómo se calcula el impacto

El analizador cruza cada archivo cambiado contra el manifiesto y acumula un motivo por tool:

| Origen | Motivo en la salida |
|---|---|
| `shared.affects_all` | `ruta compartida <patrón> (cambio en <archivo>)` |
| `tools[].entrypoints` | `entrypoint <patrón> (cambio en <archivo>)` |
| `tools[].dependencies` | `dependencia <patrón> (cambio en <archivo>)` |

El sesgo es **conservador**: si una ruta compartida cambia, se reportan todas las tools aunque el cambio no altere su comportamiento. Los archivos `*_test.go` se excluyen del cálculo porque no cambian el runtime; sus rutas se validan igual como parte de `tools[].tests`. Un falso positivo se resuelve declarando el impacto en la spec; un falso negativo se corrige ampliando el manifiesto.

## Contrato congelado

Hay dos capas de contrato y las dos corren en el gate:

### `tools/list` — `contracts.json`

`docs/mcp/tools/contracts.json` guarda nombre, descripción, esquema de entrada/salida y anotaciones de cada tool. `TestToolContracts` (`internal/mcp/tools/all/contract_test.go`) levanta el servidor MCP en memoria y compara `tools/list` contra el golden. Cualquier diferencia falla.

Un cambio intencional se regenera solo con:

```bash
go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts
```

El golden cubre el catálogo completo que registra `internal/mcp/tools/all/register.go`; al agregar un dominio, el test falla hasta regenerar `contracts.json`. Revisá siempre el diff antes de commitear.

### Cadena backend↔MCP — `TestMCPBackendContract`

`TestMCPBackendContract` (`internal/api/mcp_backend_contract_test.go`) recorre la cadena real: API key → router → handler → servicio → storage → JSON, con una DB temporal y el cliente MCP real. Detecta drift de rutas, tags JSON, autenticación y auditoría `source=mcp` sin tocar datos de producción.

```bash
go test ./internal/api -run TestMCPBackendContract
```

No hay golden que regenerar: el test usa datos sintéticos (`Servicio Demo`, `Proveedor Demo`) y falla si el contrato de la cadena cambia. Si el cambio de backend es intencional, se ajusta el test y se declara en `## Impacto en tools MCP`.

## Comandos y salida

El entrypoint es `./scripts/check-mcp-impact.sh`, que ejecuta `go run ./scripts/mcp-impact`.

| Comando | Qué hace |
|---|---|
| `--check-map` | Valida el manifiesto contra las tools registradas y que cada archivo, directorio o glob declarado exista. |
| `--files <ruta>...` | Calcula el impacto de archivos explícitos. Acepta rutas separadas por coma, repetir el flag o argumentos posicionales. |
| `--staged` | Usa `git diff --cached --name-only --diff-filter=ACMR`. |
| `--gate` | Valida toda spec nueva staged y, si hay tools afectadas, exige la spec en curso con impacto declarado y `Validación subagente:` válida; además corre `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract`. |
| `--json` | Devuelve el reporte estructurado en vez de texto. |
| `--spec <ruta\|SPEC-XXX>` | Fuerza la spec a validar (útil fuera de la rama `feature/SPEC-XXX`). |
| `--root <dir>` | Raíz del repositorio; por defecto `.`. |

### Validar el manifiesto

```bash
./scripts/check-mcp-impact.sh --check-map
```

```text
[mcp-impact] ✓ manifiesto válido: 42 tools registradas y todas sus rutas existen
```

### Consultar archivos antes de escribir código

```bash
./scripts/check-mcp-impact.sh --files internal/services/bill_bulk.go
```

```text
[mcp-impact] tools afectadas: bills_generate, bills_pay
  - bills_generate: dependencia internal/services/bill_bulk.go (cambio en internal/services/bill_bulk.go)
  - bills_pay: dependencia internal/services/bill_bulk.go (cambio en internal/services/bill_bulk.go)
```

```bash
./scripts/check-mcp-impact.sh --files internal/api/routes.go
```

```text
[mcp-impact] tools afectadas: bills_generate, bills_list, bills_pay
  - bills_generate: ruta compartida internal/api/routes.go (cambio en internal/api/routes.go)
  - bills_list: ruta compartida internal/api/routes.go (cambio en internal/api/routes.go)
  - bills_pay: ruta compartida internal/api/routes.go (cambio en internal/api/routes.go)
```

Si ningún patrón coincide:

```text
[mcp-impact] sin tools afectadas
```

El comando sale con `0` tanto si no hay tools afectadas como si las hay sin `--gate`. Un manifiesto inválido o un fallo del gate salen con `1`.

### Salida JSON

```bash
./scripts/check-mcp-impact.sh --files internal/services/bill_bulk.go --json
```

```json
{
  "affected": [
    {
      "name": "bills_generate",
      "reasons": [
        "dependencia internal/services/bill_bulk.go (cambio en internal/services/bill_bulk.go)"
      ]
    },
    {
      "name": "bills_pay",
      "reasons": [
        "dependencia internal/services/bill_bulk.go (cambio en internal/services/bill_bulk.go)"
      ]
    }
  ],
  "ok": true
}
```

Con `--gate`, el reporte agrega `spec` y, si algo falla, `ok: false` y `error`.

### Gate en el pre-commit

```bash
./scripts/check-mcp-impact.sh --staged --gate
```

Con `--staged --gate`, el gate valida **toda spec nueva staged** (`git diff --cached --diff-filter=A`): debe tener `## Impacto en tools MCP` y `Validación subagente:` válida, aunque el cambio no toque una tool. En `main` y en un merge (`MERGE_HEAD` presente) el gate de spec se omite porque son commits de release/integración; el manifiesto se sigue validando. Si no hay tools afectadas, después imprime `sin tools afectadas` y termina con éxito:

```text
[mcp-impact] ✓ 1 spec(s) nueva(s) con impacto y validación subagente
[mcp-impact] sin tools afectadas
```

Si hay tools afectadas, además resuelve la spec en curso en este orden: `--spec`, la rama `feature/SPEC-XXX`, o la última spec staged. Después verifica:

1. Que exista la sección `## Impacto en tools MCP` en la spec.
2. Que el nombre de **cada** tool afectada aparezca como palabra completa en esa sección.
3. Que haya una línea `Validación subagente:` con un valor válido.
4. Que `go test ./internal/mcp/...` pase.
5. Que `go test ./internal/api -run TestMCPBackendContract` pase.

Salida exitosa:

```text
[mcp-impact] ✓ docs/specs/SPEC-XXX-guardrail-impacto-tools-mcp.md declara impacto y validación subagente; tests MCP y contrato backend verdes
```

Mensajes de bloqueo habituales:

```text
[mcp-impact] ⛔ la spec nueva docs/specs/SPEC-XXX-<slug>.md no contiene la sección "## Impacto en tools MCP"
[mcp-impact] ⛔ la spec nueva docs/specs/SPEC-XXX-<slug>.md no registra una Validación subagente válida (valor actual: "pendiente")
[mcp-impact] ⛔ no se encontró la spec en curso; creá docs/specs/SPEC-XXX-<slug>.md o usá --spec
[mcp-impact] ⛔ la spec docs/specs/SPEC-XXX-<slug>.md no contiene la sección "## Impacto en tools MCP"
[mcp-impact] ⛔ la spec docs/specs/SPEC-XXX-<slug>.md no declara el impacto de: bills_pay
[mcp-impact] ⛔ la spec docs/specs/SPEC-XXX-<slug>.md no registra una Validación subagente válida (valor actual: "pendiente")
[mcp-impact] ⛔ go test ./internal/mcp/... falló: exit status 1
[mcp-impact] ⛔ el contrato backend↔MCP falló (go test ./internal/api -run TestMCPBackendContract): exit status 1
```

`.githooks/pre-commit` lo ejecuta después del guardrail de documentación MCP. Si falla, el commit se bloquea con la regla `DELIV-MCP-IMPACT`. Si el script no existe o no es ejecutable, el hook avisa y continúa (fail-safe); la revisión manual del diff sigue siendo obligatoria. **Prohibido** `git commit --no-verify` sin autorización explícita del usuario.

## Flujo al agregar una tool

La spec se escribe antes de tocar código (regla de oro). Dentro de esa spec, el orden de artefactos de la tool es:

1. **Manifiesto.** Agregá la entrada en `docs/mcp/tools/impact.json` con `docs`, `entrypoints` (el archivo donde vivirá `mcp.AddTool`), `dependencies`, `endpoints` y `tests`. Si la tool usa un endpoint nuevo, declaralo con el formato `METHOD /ruta`.
2. **Código.** Implementá la tool y registrala con `mcp.AddTool(server, &mcp.Tool{Name: "..."}, ...)` y el nombre literal: el analizador no resuelve variables. El plugin `rules-guard` bloquea un archivo de `internal/mcp/tools/` con `mcp.AddTool` cuyo `Name` no esté en el manifiesto, así que el paso 1 no es opcional.
3. **Contrato.** Generá el golden y revisá el diff:
   ```bash
   go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts
   ```
   Regenerá solo si el cambio de contrato es intencional. Después corré el test sin el flag para confirmar que queda verde. El golden cubre el catálogo completo de `all.Register`; al agregar un dominio, el test falla hasta regenerarlo.
4. **Documentación.** Actualizá `docs/mcp/tools/<dominio>.md`, el catálogo y cualquier referencia afectada en el mismo commit (`DELIV-MCP-DOCS`).
5. **Spec.** Completá `## Impacto en tools MCP` con la tool nueva, su cadena de dependencia, el impacto en `contracts.json` y en `TestMCPBackendContract`, y los tests que se correrán.
6. **Tests.** Agregá tests funcionales y de protocolo, extendé `TestMCPBackendContract` si la tool agrega o cambia un endpoint, y verificá:
   ```bash
   ./scripts/check-mcp-impact.sh --check-map
   go test ./internal/mcp/...
   go test ./internal/api -run TestMCPBackendContract
   go build ./...
   ```

## Flujo de una feature que toca backend compartido

1. **Antes de codificar**, corré el análisis con los archivos previstos:
   ```bash
   ./scripts/check-mcp-impact.sh --files internal/services/bill_bulk.go internal/storage/account.go
   ```
2. Si hay tools afectadas, la spec DEBE declarar cada una, la ruta de dependencia (tool → cliente → endpoint → handler → servicio → storage → tabla), el impacto en el contrato y los tests que se correrán. Si el veredicto es «ninguna tool afectada», también se declara.
3. Delegá la validación a un subagente (ver abajo) y registrá el veredicto en la spec.
4. Implementá el cambio.
5. Verificá el conjunto:
   ```bash
   ./scripts/check-mcp-impact.sh --check-map
   ./scripts/check-mcp-impact.sh --staged --gate
   go test ./internal/mcp/...
   go test ./internal/api -run TestMCPBackendContract
   ```
6. Commiteá. El hook vuelve a correr `--staged --gate` y exige que `docs/mcp/` se haya actualizado si cambió la superficie MCP.

Si el análisis marca una tool que el cambio realmente no afecta, se declara en la spec y se justifica; **no** se relaja el manifiesto para silenciar el aviso. Si una dependencia real no está mapeada, se agrega al manifiesto y se vuelve a analizar.

## Validación con un subagente

El script no puede decidir si un cambio de comportamiento es intencional. Por eso la validación de impacto se delega a un subagente (`explore` o `general`) antes de completar la sección.

El brief del subagente incluye:

- feature y comportamiento esperado;
- archivos previstos y tools candidatas;
- endpoints y tablas involucradas;
- reglas `SEC-*` aplicables;
- ruta del manifiesto `docs/mcp/tools/impact.json`.

La spec registra el resultado dentro de `## Impacto en tools MCP`:

```text
**Validación subagente**: `explore` (sesión `ses_demo000000000000`) — veredicto: ninguna tool afectada; el cambio no altera handlers, schemas ni endpoints de las tools existentes.
```

El gate rechaza un valor vacío, uno que empiece por `pendiente` o que sea exactamente `no`, `n/a`, `-` o `...`. El agente principal **revisa e integra** el veredicto; no se autovalida.

## Reglas

| Regla | Qué exige | Dónde se aplica |
|---|---|---|
| `ARCH-MCP-TOOLS-MAP` | Toda tool registrada con `mcp.AddTool` tiene entrada en `impact.json`; el manifiesto no declara tools, archivos ni patrones inexistentes. | `./scripts/check-mcp-impact.sh --check-map` y plugin `rules-guard` al escribir. |
| `DELIV-MCP-IMPACT` | Toda spec incluye `## Impacto en tools MCP`; el gate valida toda spec nueva staged y, si hay tools afectadas, las nombra y corre los tests MCP y el contrato backend. | `./scripts/check-mcp-impact.sh --staged --gate` en `.githooks/pre-commit`. |
| `DELIV-MCP-IMPACT-SUBAGENT` | La validación la hace un subagente y la spec registra agente, sesión y veredicto en `Validación subagente:`. | Revisión del agente principal; el gate verifica la evidencia declarada. |
| `TEST-MCP-CONTRACT` | `tools/list` coincide con `contracts.json` y la cadena backend↔MCP pasa `TestMCPBackendContract`; un cambio intencional se regenera con `-update-contracts` o se ajusta en el test, y se declara en la spec. | `go test ./internal/mcp/...` + `go test ./internal/api -run TestMCPBackendContract`. |

Las reglas completas viven en `docs/rules/10-architecture.md`, `docs/rules/60-testing.md` y `docs/rules/70-delivery.md`. El formato del golden está en [contracts.json](contracts.json); la cadena real se cubre en `internal/api/mcp_backend_contract_test.go` y la referencia de las tools en [bills.md](bills.md).
