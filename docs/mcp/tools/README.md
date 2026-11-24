---
title: "Catálogo de tools"
sidebar_position: 1
description: "Convenciones y listado de tools MCP disponibles"
---

# Catálogo de tools

## Convención de nombres

- `snake_case`, con el dominio primero: `bills_list`, `bills_generate`, `autos_maintenance_register`.
- Verbo al final: `list` (lectura), `generate` (alta masiva), `pay` (acción), `register` (registro de un hecho).
- Las tools de escritura declaran `dry_run` y respetan `P40LA_MCP_ALLOW_WRITES`.

## Tools actuales

| Tool | Dominio | Tipo | Documentación |
|---|---|---|---|
| `bills_list` | Facturas | Lectura | [bills.md](bills.md) |
| `bills_generate` | Facturas | Escritura | [bills.md](bills.md) |
| `bills_pay` | Facturas | Escritura | [bills.md](bills.md) |
| `autos_list` | Autos | Lectura | [autos.md](autos.md) |
| `autos_create` | Autos | Escritura | [autos.md](autos.md) |
| `autos_update` | Autos | Escritura | [autos.md](autos.md) |
| `autos_maintenance_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_inspection_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_insurance_register` | Autos | Escritura | [autos.md](autos.md) |

## Mismos tools en stdio y HTTP

El catálogo no cambia según el transporte. `p40la-mcp` monta el mismo `mcp.Server` en stdio (Claude Desktop/Code local, OpenCode) y en Streamable HTTP (`/mcp`, claude.ai). No hay tools “remotas” ni permisos distintos por cliente: la API key global tiene todos los permisos y el `subject` OAuth (`owner`) es solo un identificador de auditoría.

Lo que cambia es la **compuerta de escritura**:

| Transporte | Lectura (`bills_list`, `autos_list`) | Escritura (`dry_run=false`) |
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

- Lectura: `bills_list` y `autos_list` usan `readOnlyHint=true`.
- Escritura idempotente: `bills_generate`, `bills_pay` y `autos_update` usan `destructiveHint=false`, `idempotentHint=true`, `openWorldHint=false`.
- Altas no idempotentes: `autos_create`, `autos_maintenance_register`, `autos_inspection_register` y `autos_insurance_register` usan `destructiveHint=false`, `idempotentHint=false`, `openWorldHint=false`.

`destructiveHint=false` significa que son aditivas (crean o actualizan estado), no borran datos. La confirmación humana sigue siendo obligatoria por el impacto financiero. `autos_insurance_register` declara `idempotentHint=false`; repetir con el mismo `service_id` devuelve `already_exists` y no crea un duplicado.
