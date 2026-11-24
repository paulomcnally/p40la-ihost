---
title: "Catálogo de tools"
sidebar_position: 1
description: "Convenciones y listado de tools MCP disponibles"
---

# Catálogo de tools

## Convención de nombres

- `snake_case`, con el dominio primero: `bills_list`, `entities_search`, `autos_maintenance_register`, `homes_create`.
- Verbo al final: `list`/`search` (lectura), `create` (alta), `generate` (alta masiva), `pay` (acción), `register` (registro de un hecho).
- Las tools de escritura respetan `P40LA_MCP_ALLOW_WRITES`. Las de facturas y autos declaran `dry_run`; las de alta de entidades (`*_create`) no tienen previsualización y verifican duplicados antes del POST. `entities_search` es de solo lectura y no depende de la compuerta.

## Tools actuales

El catálogo tiene **14 tools**: 3 de lectura y 11 de escritura.

| Tool | Dominio | Tipo | Documentación |
|---|---|---|---|
| `bills_list` | Facturas | Lectura | [bills.md](bills.md) |
| `entities_search` | Entidades | Lectura | [entities.md](entities.md) |
| `autos_list` | Autos | Lectura | [autos.md](autos.md) |
| `bills_generate` | Facturas | Escritura | [bills.md](bills.md) |
| `bills_pay` | Facturas | Escritura | [bills.md](bills.md) |
| `autos_create` | Autos | Escritura | [autos.md](autos.md) |
| `autos_update` | Autos | Escritura | [autos.md](autos.md) |
| `autos_maintenance_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_inspection_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_insurance_register` | Autos | Escritura | [autos.md](autos.md) |
| `homes_create` | Hogares | Escritura | [entities.md](entities.md) |
| `institutions_create` | Instituciones | Escritura | [entities.md](entities.md) |
| `banks_create` | Bancos | Escritura | [entities.md](entities.md) |
| `services_create` | Servicios | Escritura | [entities.md](entities.md) |

## Mismos tools en stdio y HTTP

El catálogo no cambia según el transporte. `p40la-mcp` monta el mismo `mcp.Server` en stdio (Claude Desktop/Code local, OpenCode) y en Streamable HTTP (`/mcp`, claude.ai). No hay tools “remotas” ni permisos distintos por cliente: la API key global tiene todos los permisos y el `subject` OAuth (`owner`) es solo un identificador de auditoría.

Lo que cambia es la **compuerta de escritura**:

| Transporte | Lectura (`bills_list`, `entities_search`, `autos_list`) | Escritura (`dry_run=false` o cualquier `*_create`) |
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

En las tools `*_create` el `hint` solo pide habilitar las variables, porque no existe `dry_run` para previsualizar. La compuerta se evalúa antes de validar la entrada: sin permisos, la tool no llama al backend.

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

- Lectura: `bills_list`, `entities_search` y `autos_list` usan `readOnlyHint=true`, `openWorldHint=false`.
- Escritura idempotente: `bills_generate`, `bills_pay` y `autos_update` usan `destructiveHint=false`, `idempotentHint=true`, `openWorldHint=false`.
- Altas de autos: `autos_create`, `autos_maintenance_register`, `autos_inspection_register` y `autos_insurance_register` usan `destructiveHint=false`, `idempotentHint=false`, `openWorldHint=false`.
- Altas de entidades: `homes_create`, `institutions_create`, `banks_create` y `services_create` usan `destructiveHint=true`, `openWorldHint=false`. No declaran `idempotentHint`.

`destructiveHint=false` en facturas y autos significa que son aditivas (crean o actualizan estado), no borran datos. En las tools de alta de entidades el valor es `true` de forma conservadora: la tool no garantiza ser segura ante reintentos, aunque en la práctica solo agrega registros y nunca borra. La confirmación humana sigue siendo obligatoria por el impacto en los datos. `autos_insurance_register` declara `idempotentHint=false`; repetir con el mismo `service_id` devuelve `already_exists` y no crea un duplicado.
