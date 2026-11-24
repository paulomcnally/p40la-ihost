---
title: "MCP de p40la-ihost"
sidebar_position: 1
description: "Servidor MCP y tools para operar p40la-ihost desde agentes"
---

# MCP de p40la-ihost

`p40la-mcp` es un servidor [Model Context Protocol](https://modelcontextprotocol.io/) escrito en Go que expone tools para operar p40la-ihost desde un agente (OpenCode, Claude Desktop u otro cliente MCP). Habla **stdio** y usa la **API HTTP** del sistema; no accede a SQLite ni al volumen de datos.

La primera entrega cubre facturas de servicios:

| Tool | Tipo | Qué hace |
|---|---|---|
| `bills_list` | lectura | Lista facturas de un servicio con filtros de período y estado |
| `bills_generate` | escritura | Genera facturas de un rango de meses (por ejemplo, enero a la fecha) |
| `bills_pay` | escritura | Paga una factura, un mes, un rango o todas las pendientes hasta una fecha |

## Requisitos

- Go 1.26 o superior para compilar.
- Acceso HTTP al servidor de p40la-ihost (`http://ihost.local:8088` por defecto).
- Un usuario válido del sistema, o una cookie de sesión.
- Opcional: `P40LA_MCP_ALLOW_WRITES=true` para habilitar las tools que escriben.

## Inicio rápido

```bash
# Compilar el binario
go build -o bin/p40la-mcp ./cmd/mcp-server

# Variables mínimas (no las commitees)
export P40LA_BASE_URL="http://ihost.local:8088"
export P40LA_EMAIL="demo@example.com"
export P40LA_PASSWORD="demo-password"

# El cliente MCP lanza el binario; también podés probarlo a mano con un
# mensaje JSON-RPC por stdin.
./bin/p40la-mcp
```

Configuración típica en OpenCode (ver [configuration.md](configuration.md)):

```json
{
  "mcp": {
    "p40la": {
      "type": "local",
      "command": ["/home/demo/bin/p40la-mcp"],
      "environment": {
        "P40LA_BASE_URL": "http://ihost.local:8088",
        "P40LA_EMAIL": "demo@example.com",
        "P40LA_PASSWORD": "{env:P40LA_PASSWORD}",
        "P40LA_MCP_ALLOW_WRITES": "false"
      }
    }
  }
}
```

## Flujo seguro de escritura

1. El agente llama `bills_generate` o `bills_pay` **sin** `dry_run=false` (el valor por defecto es `true`).
2. Muestra el resumen al usuario: períodos, montos, facturas omitidas.
3. El usuario confirma.
4. El agente repite con `dry_run=false`. La compuerta `P40LA_MCP_ALLOW_WRITES=true` debe estar activa.

Las operaciones son idempotentes por período: repetir una generación no duplica facturas y pagar una factura ya pagada devuelve `skipped`.

## Documentación

- [Arquitectura](architecture.md)
- [Configuración](configuration.md)
- [Seguridad](security.md)
- [Operación](operations.md)
- [Desarrollo de nuevas tools](development.md)
- [Catálogo de tools](tools/README.md)
- [Tools de facturas](tools/bills.md)

## Publicación web

Esta carpeta es Markdown plano con frontmatter de Docusaurus (`title`, `sidebar_position`, `description`) y `_category_.json`. Para publicarla, un sitio Docusaurus puede usar `docs/mcp` como `path` de un plugin de docs o copiar la carpeta dentro de su `docs/`. No se requiere build de Node en este repositorio.
