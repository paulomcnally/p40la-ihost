---
title: "API key global"
sidebar_position: 2
description: "Creación, rotación y revocación de la API key del sistema"
---

# API key global

p40la-ihost usa **una única API key global** para que el MCP opere sin la contraseña del usuario. La key tiene todos los permisos del sistema.

## Características

- Formato `p40la_` + 32 bytes aleatorios en base64url.
- Se guarda **hasheada** (SHA-256) en `system_settings`; nunca en texto plano.
- Se muestra completa **una sola vez**, al generarla o rotarla.
- Se valida en tiempo constante.
- Se puede rotar o revocar de inmediato.
- Registra `last_used_at` para auditoría.

## Crear o rotar con el script

```bash
EMAIL="demo@example.com" \
PASSWORD="demo-password" \
./scripts/create-api-key.sh
```

El script inicia sesión una vez, llama `POST /api/api-key/regenerate`, imprime la key y descarta la cookie temporal. Si definís `KEY_FILE`, la guarda con permisos `0600`:

```bash
EMAIL="demo@example.com" PASSWORD="demo-password" \
KEY_FILE="$HOME/.config/p40la/api-key" \
./scripts/create-api-key.sh
```

> Rotar invalida la key anterior de inmediato. Actualizá `P40LA_API_KEY` en el cliente MCP después de rotar.

## Endpoints (solo sesión)

La gestión de la key requiere sesión de usuario; una API key **no** puede crear ni rotar keys.

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/api/api-key` | Estado: `configured`, `prefix`, `created_at`, `last_used_at` |
| `POST` | `/api/api-key/regenerate` | Genera una key nueva y devuelve el secreto una vez |
| `DELETE` | `/api/api-key` | Revoca la key vigente |

Ejemplo de respuesta al regenerar:

```json
{
  "api_key": "p40la_...",
  "prefix": "p40la_AbCdEfGh",
  "created_at": "2026-11-23T00:00:00Z"
}
```

## Uso desde el MCP

```bash
export P40LA_API_KEY="p40la_..."
```

El MCP envía en cada request:

```http
Authorization: Bearer p40la_...
```

También se acepta `X-API-Key: p40la_...` para clientes que no usan Bearer.

## Revocar

```bash
curl -X DELETE -b /tmp/cookies.txt http://ihost.local:8088/api/api-key
```

Después de revocar, cualquier request con la key anterior devuelve `401 auth_failed`.

## Seguridad

- La key global equivale a un usuario con acceso completo. Si se filtra, rotala y revisá `last_used_at`.
- No la commitees, no la pegues en issues ni la incluyas en capturas.
- Usá HTTPS o una red de confianza (Tailscale/VPN) cuando el servidor no esté en localhost.
- El MCP nunca ve la contraseña del usuario: solo la key.
