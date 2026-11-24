---
title: "OAuth 2.1"
sidebar_position: 9
description: "Proveedor OAuth de usuario único para el MCP remoto: discovery, DCR, PKCE y tokens"
---

# OAuth 2.1

El modo HTTP del MCP está protegido por un proveedor **OAuth 2.1 de usuario único** implementado en `internal/mcp/server/oauth`. No hay proveedor de identidad externo, no se emiten JWT y no se guardan contraseñas en texto plano.

## Por qué OAuth

claude.ai, las apps móviles y Cowork no ejecutan el conector en tu máquina: lo ejecutan desde la infraestructura de Anthropic. Por eso:

- Exigen un endpoint **HTTPS público en el puerto 443**.
- Rechazan direcciones privadas y el rango CGNAT (`100.64/10`); una IP de Tailscale no es alcanzable para ellos.
- Usan **Streamable HTTP** y esperan autenticación OAuth 2.1 con registro dinámico de clientes (DCR). Una URL “secreta” o un header inventado no alcanza.

Tailscale Funnel resuelve la parte de red (nombre público + certificado + proxy al loopback). OAuth resuelve la autenticación. Sin OAuth no se expone el MCP.

## Modelo

- **Un solo usuario.** La contraseña se guarda como hash bcrypt en `P40LA_MCP_OAUTH_PASSWORD_HASH`. El `subject` que aparece en la auditoría es un identificador opaco (`owner` por defecto).
- **Clientes públicos.** DCR solo acepta `token_endpoint_auth_method=none`; no se emiten `client_secret`.
- **PKCE obligatorio** con `code_challenge_method=S256`.
- **Tokens opacos**, no JWT: 32 bytes aleatorios en base64url, guardados como SHA-256 en `oauth.json` con permisos `0600`.
- **Un solo emisor**: la URL pública configurada en `P40LA_MCP_PUBLIC_URL`, sin path.

## Flujo

```
Cliente (claude.ai)
   │ 1. GET /.well-known/oauth-protected-resource
   │ 2. GET /.well-known/oauth-authorization-server
   │ 3. POST /register                      (DCR)
   │ 4. GET /authorize?response_type=code&code_challenge=...&state=...
   ▼
Navegador del usuario
   │ 5. formulario de contraseña → POST /authorize
   ▼
Cliente
   │ 6. redirect_uri?code=...&state=...
   │ 7. POST /token  (code + code_verifier)
   ▼
Access token (1 h) + refresh token (30 días)
   │ 8. POST /mcp  Authorization: Bearer <access token>
   ▼
Tools MCP
```

El cliente conserva el `state` y lo valida al volver; el servidor lo devuelve sin modificarlo. PKCE protege el código aunque el redirect se intercepte.

## Endpoints

| Endpoint | Método | Descripción |
|---|---|---|
| `/.well-known/oauth-protected-resource` | `GET` | Recurso protegido y servidores de autorización |
| `/.well-known/oauth-protected-resource/mcp` | `GET` | Igual, con el path del recurso |
| `/.well-known/oauth-authorization-server` | `GET` | Metadatos del servidor de autorización |
| `/register` | `POST` | Registro dinámico de clientes (RFC 7591) |
| `/authorize` | `GET` / `POST` | Consentimiento y código de autorización |
| `/token` | `POST` | Canje del código y refresh |

Todos viven en el listener público y se publican por Funnel. Están pensados para el flujo de Anthropic, no para uso desde JavaScript de un navegador (no hay CORS).

### Discovery

`GET /.well-known/oauth-protected-resource`:

```json
{
  "resource": "https://demo-node.example.ts.net/mcp",
  "authorization_servers": ["https://demo-node.example.ts.net"],
  "scopes_supported": ["mcp:read", "mcp:write"],
  "bearer_methods_supported": ["header"]
}
```

`GET /.well-known/oauth-authorization-server`:

```json
{
  "issuer": "https://demo-node.example.ts.net",
  "authorization_endpoint": "https://demo-node.example.ts.net/authorize",
  "token_endpoint": "https://demo-node.example.ts.net/token",
  "registration_endpoint": "https://demo-node.example.ts.net/register",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"],
  "token_endpoint_auth_methods_supported": ["none"],
  "scopes_supported": ["mcp:read", "mcp:write"]
}
```

Los valores se derivan de `P40LA_MCP_PUBLIC_URL`. Si la URL pública no coincide con el hostname de Funnel, el cliente descubre un emisor inalcanzable y el flujo falla.

## Registro dinámico (DCR)

`POST /register` acepta entre 1 y 5 `redirect_uris`. Solo se permiten:

| Cliente | Redirect URI |
|---|---|
| claude.ai / apps de Anthropic | `https://claude.ai/api/mcp/auth_callback` (host exacto, sin puerto, path exacto) |
| Clientes locales / pruebas | `http://localhost/callback`, `http://127.0.0.1/callback` o `http://[::1]/callback`, con **cualquier puerto** |

No se aceptan otros esquemas, fragmentos, userinfo ni hosts distintos. Un `redirect_uri` fuera de la allowlist devuelve `400 invalid_redirect_uri`.

El cliente recibe un `client_id` opaco; no hay secreto. Los clientes registrados se guardan en `oauth.json` y sobreviven a reinicios.

## Autorización y PKCE

`GET /authorize` valida:

- `response_type=code`
- `code_challenge_method=S256`
- `code_challenge` de 43 a 128 caracteres
- `state` de hasta 512 caracteres
- `client_id` registrado y `redirect_uri` exactamente igual a uno de los registrados

Luego muestra una página HTML mínima con el nombre del cliente, el host de callback y un campo de contraseña. El formulario no revela si el usuario existe: hay un solo usuario. La respuesta incluye `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; form-action 'self'` y `Cache-Control: no-store`.

Con la contraseña correcta se emite un código de un solo uso, válido 5 minutos, y se redirige al `redirect_uri` con `code` y `state`.

## Tokens

| Token | Formato | Vigencia | Rotación |
|---|---|---|---|
| Código de autorización | 32 bytes base64url | 5 minutos | Un solo uso |
| Access token | 32 bytes base64url | 1 hora | Se reemplaza al refrescar |
| Refresh token | 32 bytes base64url | 30 días | Rotativo: el anterior se invalida |
| `client_id` | 16 bytes base64url | Sin vencimiento | — |

- El canje `authorization_code` exige `code`, `client_id`, `redirect_uri` y `code_verifier`; el `code_verifier` debe corresponder al `code_challenge` (SHA-256 + base64url sin padding).
- El refresh (`grant_type=refresh_token`) rota el refresh token **y** invalida el access token anterior de esa entrada.
- Los tokens se validan contra el hash SHA-256 almacenado; el texto plano solo existe en la respuesta al cliente.
- Errores estándar: `invalid_request`, `invalid_grant`, `unsupported_grant_type`, `invalid_client_metadata`, `invalid_redirect_uri`, `too_many_requests`.

## Persistencia

El estado vive en `P40LA_MCP_STATE_DIR` (default `/data`):

| Archivo | Contenido | Permisos |
|---|---|---|
| `oauth.json` | Clientes, hashes de códigos y hashes de tokens | `0600` |
| `audit.jsonl` | Auditoría de llamadas a tools | `0600` |

La escritura es atómica (`archivo.tmp` + `rename`) y el directorio se crea con `0700`. Los códigos vencidos y las entradas con refresh vencido se limpian al cargar y en cada operación. No hay PII: el `subject` es `owner` y no se guardan argumentos ni resultados.

## Revocación

```bash
curl -s -X POST http://127.0.0.1:8091/admin/revoke
```

- Invalida **todos** los access tokens, refresh tokens y códigos pendientes.
- **Conserva** los clientes registrados: Claude puede reintentar la autorización sin volver a registrarse.
- El endpoint solo responde en el listener administrativo y solo si `RemoteAddr` es loopback.

Para un reinicio total (incluidos clientes), detener el proceso, mover o borrar `oauth.json` y arrancar de nuevo. Todos los conectores deberán registrarse y autorizar otra vez.

## Generar el hash de la contraseña

El binario incluye un comando que lee la contraseña por stdin y escribe el hash bcrypt (costo 12) por stdout:

```bash
read -rs -p "Contraseña: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | ./bin/p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD

# Guardar el hash en el entorno del servicio, nunca en el repo
printf 'P40LA_MCP_OAUTH_PASSWORD_HASH=%s\n' "$HASH" | sudo tee -a /etc/p40la-mcp.env >/dev/null
```

El comando no requiere `P40LA_API_KEY` ni arranca el servidor. El hash empieza con `$2a$`, `$2b$` o `$2y$`.

> Rotar la contraseña **no invalida los tokens existentes**. Después de cambiarla, reiniciar el proceso y ejecutar `POST /admin/revoke` para forzar una reautorización.

## Límites por IP

Además del rate limit por token en `/mcp`, los endpoints OAuth tienen límites en memoria por IP:

| Endpoint | Límite |
|---|---|
| `POST /authorize` (intentos de contraseña) | 5 por minuto |
| `POST /register` | 10 por hora |
| `POST /token` | 20 por minuto |

Al excederse responden `429` con `Retry-After`. La IP se toma de `X-Forwarded-For` (que Funnel propaga) y, si no está, de `RemoteAddr`.

## Troubleshooting

| Síntoma | Causa probable | Acción |
|---|---|---|
| `/mcp` responde `401 invalid_token` | Access token vencido (1 h), revocado o corrupto | El cliente debe refrescar; si el refresh también falla, quitá y volvé a agregar el conector |
| Claude no muestra la pantalla de autorización | Funnel caído, certificado inválido o `PUBLIC_URL` distinto del hostname real | Verificar discovery desde fuera de la tailnet |
| `400 invalid_redirect_uri` en `/register` | El callback no está en la allowlist | Usar el callback exacto de Claude o un loopback `/callback` |
| “Contraseña incorrecta” en `/authorize` | Hash de otra contraseña o error de tipeo | Regenerar el hash, reiniciar y volver a autorizar |
| `429` en `/authorize` o `/token` | Límite por IP alcanzado | Esperar un minuto y reintentar |
| `invalid_grant` al canjear el código | Código vencido (> 5 min), reutilizado o `code_verifier` distinto | Reiniciar el flujo de autorización |
| `invalid_grant` al refrescar | Refresh revocado o vencido, o `client_id` distinto | Volver a autorizar el conector |
| Discovery devuelve un `issuer` viejo | `P40LA_MCP_PUBLIC_URL` cambió y el proceso no se reinició | Reiniciar y revocar tokens |
| `oauth.json: permission denied` | `/data` no es escribible por el UID 65532 | Ajustar dueño y permisos del volumen |
| El login entra en loop | Un proxy intermedio reescribe el redirect o el host | Probar sin proxy y verificar que Funnel apunte solo a 8090 |

## Referencias

- RFC 9728 — OAuth 2.0 Protected Resource Metadata.
- RFC 8414 — Authorization Server Metadata.
- RFC 7591 — Dynamic Client Registration.
- RFC 7636 — PKCE.
- [Servidor HTTP remoto](http.md) · [Tailscale Funnel](funnel.md) · [Seguridad](security.md)
