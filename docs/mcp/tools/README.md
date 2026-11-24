---
title: "Catálogo de tools"
sidebar_position: 1
description: "Convenciones y listado de tools MCP disponibles"
---

# Catálogo de tools

## Convención de nombres

- `snake_case`, con el dominio primero: `bills_list`, `bills_generate`, `bills_pay`.
- Verbo al final: `list` (lectura), `generate` (alta masiva), `pay` (acción).
- Las tools de escritura declaran `dry_run` y respetan `P40LA_MCP_ALLOW_WRITES`.

## Tools actuales

| Tool | Dominio | Tipo | Documentación |
|---|---|---|---|
| `bills_list` | Facturas | Lectura | [bills.md](bills.md) |
| `bills_generate` | Facturas | Escritura | [bills.md](bills.md) |
| `bills_pay` | Facturas | Escritura | [bills.md](bills.md) |

## Contrato congelado y manifiesto de impacto

El catálogo se controla con:

- [`impact.json`](impact.json) — manifiesto de dependencias: entrypoints, archivos de backend, endpoints, tests y documentación. Lo valida `./scripts/check-mcp-impact.sh --check-map` y lo exige el hook `pre-commit`. El formato, los comandos y el flujo de trabajo están en [impact.md](impact.md).
- [`contracts.json`](contracts.json) — golden de `tools/list`: nombres, descripciones, esquemas de entrada/salida y anotaciones. `TestToolContracts` (`internal/mcp/tools/all/contract_test.go`) falla si el contrato cambia sin actualizar el archivo.
- `TestMCPBackendContract` (`internal/api/mcp_backend_contract_test.go`) — sin golden: recorre API key → router → handler → servicio → storage → JSON con una DB temporal y detecta drift de rutas, tags JSON, auth o `source=mcp`.

Un cambio de contrato es intencional solo si la spec lo declara. Para regenerar el golden:

```bash
go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts
```

Después revisá el diff de `contracts.json`, corré el test sin `-update-contracts` y verificá el impacto:

```bash
go test ./internal/mcp/...
go test ./internal/api -run TestMCPBackendContract
./scripts/check-mcp-impact.sh --staged --gate
```

## Mismos tools en stdio y HTTP

El catálogo no cambia según el transporte. `p40la-mcp` monta el mismo `mcp.Server` en stdio (Claude Desktop/Code local, OpenCode) y en Streamable HTTP (`/mcp`, claude.ai). No hay tools “remotas” ni permisos distintos por cliente: la API key global tiene todos los permisos y el `subject` OAuth (`owner`) es solo un identificador de auditoría.

Lo que cambia es la **compuerta de escritura**:

| Transporte | Lectura (`bills_list`) | Escritura (`dry_run=false`) |
|---|---|---|
| stdio | Siempre disponible | Requiere `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | Siempre disponible | Requiere `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

Si falta la compuerta remota, la tool responde:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true, o repetí con dry_run=true para previsualizar."
}
```

El modo HTTP no agrega tools nuevas ni cambia los schemas de entrada. Los límites de body (1 MiB), rate limit y auditoría son del servidor, no de cada tool. Ver [http.md](../http.md), [oauth.md](../oauth.md) y [security.md](../security.md).

## Respuestas

Las tools devuelven contenido de texto JSON y `structuredContent` con el mismo objeto. Los errores de negocio usan `isError=true` y un objeto:

```json
{
  "error": "invalid_range",
  "message": "el período final no puede ser anterior al inicial",
  "hint": "Usá from y to en formato YYYY-MM"
}
```

## Anotaciones MCP

- `bills_list`: `readOnlyHint=true`.
- `bills_generate` y `bills_pay`: `destructiveHint=false`, `idempotentHint=true`, `openWorldHint=false`.

`destructiveHint=false` significa que son aditivas (crean o actualizan estado), no borran datos. La confirmación humana sigue siendo obligatoria por el impacto financiero.
