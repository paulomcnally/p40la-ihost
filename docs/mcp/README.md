---
title: "MCP de p40la-ihost"
sidebar_position: 1
description: "Servidor MCP y tools para operar p40la-ihost desde agentes"
---

# MCP de p40la-ihost

`p40la-mcp` es un servidor [Model Context Protocol](https://modelcontextprotocol.io/) escrito en Go que expone tools para operar p40la-ihost desde un agente (OpenCode, Claude Desktop u otro cliente MCP). Habla **stdio** y usa la **API HTTP** del sistema con una **API key global**; no accede a SQLite ni al volumen de datos.

La primera entrega cubre facturas de servicios:

| Tool | Tipo | Qué hace |
|---|---|---|
| `bills_list` | lectura | Lista facturas de un servicio con filtros de período y estado |
| `bills_generate` | escritura | Genera facturas de un rango de meses (por ejemplo, enero a la fecha) |
| `bills_pay` | escritura | Paga una factura, un mes, un rango o todas las pendientes hasta una fecha |

## Requisitos

- Go 1.26 o superior para compilar.
- Acceso HTTP al servidor de p40la-ihost (`http://ihost.local:8088` por defecto).
- Una **API key global** del sistema. Se crea una vez con `scripts/create-api-key.sh` o desde `POST /api/api-key/regenerate` con una sesión autenticada.
- Opcional: `P40LA_MCP_ALLOW_WRITES=true` para habilitar las tools que escriben.

La key es global y tiene **todos los permisos**. No uses tu contraseña en la configuración del MCP: la contraseña solo se necesita para provisionar o rotar la key.

## Inicio rápido

```bash
# 1. Compilar el binario
go build -o bin/p40la-mcp ./cmd/mcp-server

# 2. Crear/rotar la API key global (una sola vez; la contraseña no queda en el MCP)
EMAIL="demo@example.com" PASSWORD="demo-password" ./scripts/create-api-key.sh

# 3. Exportar la key y arrancar el MCP
export P40LA_API_KEY="p40la_..."
export P40LA_BASE_URL="http://ihost.local:8088"
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
        "P40LA_API_KEY": "{env:P40LA_API_KEY}",
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

- [API key global](api-key.md)
- [Arquitectura](architecture.md)
- [Configuración](configuration.md)
- [Seguridad](security.md)
- [Operación](operations.md)
- [Desarrollo de nuevas tools](development.md)
- [Catálogo de tools](tools/README.md)
- [Tools de facturas](tools/bills.md)

## Publicación web

Esta carpeta es Markdown plano con frontmatter de Docusaurus (`title`, `sidebar_position`, `description`) y `_category_.json`. Para publicarla, un sitio Docusaurus puede usar `docs/mcp` como `path` de un plugin de docs o copiar la carpeta dentro de su `docs/`. No se requiere build de Node en este repositorio.
