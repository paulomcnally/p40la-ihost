---
title: "MCP de p40la-ihost"
sidebar_position: 1
description: "Servidor MCP y tools para operar p40la-ihost desde agentes"
---

# MCP de p40la-ihost

`p40la-mcp` es un servidor [Model Context Protocol](https://modelcontextprotocol.io/) escrito en Go que expone tools para operar p40la-ihost desde un agente (OpenCode, Claude Desktop, claude.ai u otro cliente MCP). Usa la **API HTTP** del sistema con una **API key global**; no accede a SQLite ni al volumen de datos.

El binario tiene dos transportes con el mismo catálogo de tools:

| Modo | Transporte | Cliente | Exposición |
|---|---|---|---|
| Local (default) | `stdio` | OpenCode, Claude Code y Claude Desktop local | Ninguna |
| Remoto | Streamable HTTP en `/mcp` | claude.ai, Claude móvil, Cowork | HTTPS 443 vía Tailscale Funnel + OAuth 2.1 |

El modo remoto está pensado para conectores que Anthropic ejecuta desde su infraestructura: exige HTTPS público, OAuth 2.1 y una contraseña de administración. El modo local no necesita nada de eso. Ver [http.md](http.md), [oauth.md](oauth.md) y [funnel.md](funnel.md).

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
- Opcional para el modo local: `P40LA_MCP_ALLOW_WRITES=true` para habilitar las tools que escriben.
- Para el modo remoto: una URL HTTPS pública (`P40LA_MCP_PUBLIC_URL`), un hash bcrypt (`p40la-mcp -hash-password`) y, si se publica desde el iHost, Tailscale Funnel habilitado por el usuario.

La key es global y tiene **todos los permisos**. No uses la contraseña del dashboard en la configuración del MCP: solo se necesita para provisionar la key. El modo remoto usa además una **contraseña de administración propia**, distinta de la del dashboard, que solo se guarda como hash bcrypt.

## Inicio rápido (local)

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

## Inicio rápido (remoto)

Para claude.ai o Claude móvil no se configura un comando local: se publica `https://<nodo>.<tailnet>.ts.net/mcp` y se agrega como conector. El resumen es:

```bash
# 1. Hash de la contraseña de administración
read -rs -p "Contraseña: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | ./bin/p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD

# 2. Arrancar en modo HTTP (solo loopback)
P40LA_BASE_URL="http://ihost.local:8088" \
P40LA_API_KEY="p40la_demo-demo-demo-demo-demo-demo" \
P40LA_MCP_TRANSPORT=http \
P40LA_MCP_PUBLIC_URL="https://demo-node.example.ts.net" \
P40LA_MCP_OAUTH_PASSWORD_HASH="$HASH" \
P40LA_MCP_REMOTE_WRITES=false \
./bin/p40la-mcp

# 3. Publicar 443 → 127.0.0.1:8090 (requiere Funnel habilitado en la tailnet)
tailscale funnel --bg 8090
```

El paso 3 es una decisión explícita del usuario y deja el endpoint en Internet. Seguí el runbook de [funnel.md](funnel.md) y el checklist de [security.md](security.md). Claude Desktop local sigue usando **stdio**: el modo HTTP es para conectores remotos.

Si corrés el binario fuera del contenedor, definí `P40LA_MCP_STATE_DIR` en un directorio escribible por tu usuario (por ejemplo `/tmp/p40la-mcp-state`); el default `/data` es para la imagen Docker.

## Flujo seguro de escritura

1. El agente llama `bills_generate` o `bills_pay` **sin** `dry_run=false` (el valor por defecto es `true`).
2. Muestra el resumen al usuario: períodos, montos, facturas omitidas.
3. El usuario confirma.
4. El agente repite con `dry_run=false`. La compuerta `P40LA_MCP_ALLOW_WRITES=true` debe estar activa; si el transporte es HTTP, también `P40LA_MCP_REMOTE_WRITES=true`.

Las operaciones son idempotentes por período: repetir una generación no duplica facturas y pagar una factura ya pagada devuelve `skipped`.

## Documentación

- [API key global](api-key.md)
- [Configuración](configuration.md)
- [Arquitectura](architecture.md)
- [Seguridad](security.md)
- [Operación](operations.md)
- [Desarrollo de nuevas tools](development.md)
- [Servidor HTTP remoto](http.md)
- [OAuth 2.1](oauth.md)
- [Tailscale Funnel](funnel.md)
- [Catálogo de tools](tools/README.md)
- [Tools de facturas](tools/bills.md)

## Publicación web

Esta carpeta es Markdown plano con frontmatter de Docusaurus (`title`, `sidebar_position`, `description`) y `_category_.json`. Para publicarla, un sitio Docusaurus puede usar `docs/mcp` como `path` de un plugin de docs o copiar la carpeta dentro de su `docs/`. No se requiere build de Node en este repositorio.
