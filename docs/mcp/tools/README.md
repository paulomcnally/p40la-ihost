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
