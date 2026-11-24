---
title: "MCP remoto para Claude: transporte HTTP, OAuth y Tailscale Funnel"
id: "SPEC-147"
status: "released"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 150
---

# MCP remoto para Claude: transporte HTTP, OAuth y Tailscale Funnel

**ID**: SPEC-147  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

El servidor MCP de SPEC-145 expone tools por **stdio** y está pensado para ejecutarse en la máquina del cliente. Eso funciona con Claude Code o Claude Desktop local, pero **no con claude.ai, Claude móvil ni Cowork**, porque los conectores remotos los ejecuta Anthropic desde su nube: exigen un endpoint **HTTPS público en el puerto 443**, transporte **Streamable HTTP** y autenticación (OAuth 2.1 o, en beta limitada, headers estáticos).

Esta spec agrega un **modo servidor HTTP** al binario MCP, con OAuth 2.1 de usuario único (PKCE + DCR/CIMD), auditoría y límites de uso, y define cómo publicarlo con **Tailscale Funnel** desde un nodo de la tailnet. El proceso sigue siendo un cliente de la API de p40la-ihost: no toca SQLite, no agrega lógica de dominio y no reemplaza el servidor web.

El objetivo es poder agregar `https://<nodo>.<tailnet>.ts.net/mcp` como conector en claude.ai. Como el endpoint queda en Internet, la autenticación y el mínimo privilegio son requisitos P0: **sin OAuth no hay exposición**. Las escrituras quedan dobles-gateadas (`P40LA_MCP_ALLOW_WRITES` + `P40LA_MCP_REMOTE_WRITES`) y auditadas; el deploy inicial las habilita solo con aprobación explícita del usuario.

> Este repo es **público**: la spec usa placeholders (`<nodo>.<tailnet>.ts.net`, `192.0.2.10`) y datos sintéticos. Prohibido pegar el nombre real de la tailnet, IPs, tokens o API keys.

---

## 2. Cumplimiento de reglas

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `modelcontextprotocol/go-sdk`; OAuth con `x/crypto` ya presente; sin framework HTTP | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | `cmd/mcp-server` (modo), `internal/mcp/server` (HTTP/OAuth), `internal/mcp/audit` | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | El MCP sigue siendo cliente HTTP; cero SQL y cero reglas de dominio | `CA-ARCH-LAYERS` |
| `ARCH-IHOST` | Bind a loopback, binario liviano, estado en volumen, sin Node en runtime | `CA-ARCH-IHOST` |
| `ARCH-GO` | `gofmt`, errores explícitos, `log/slog` a stderr, timeouts | `CA-ARCH-GO` |
| `DATA-MIGRATIONS` | N/A: sin migraciones; el estado OAuth/auditoría es un archivo propio | — |
| `DATA-NO-DROP` / `DATA-PRODCOPY` | La validación E2E usa copia de producción en `/tmp`; nunca `data/app.db` | `CA-DATA-PRODCOPY` |
| `SEC-NO-PII` | Repo público: solo placeholders y datos sintéticos; sin secretos ni IPs reales | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | Ejemplos con `demo-*`, `192.0.2.10`, UUID cero | `CA-SEC-SYNTHETIC` |
| `SEC-LOGS` | Auditoría sin cuerpos de request ni PII; solo tool, ids, resultado y duración | `CA-SEC-LOGS` |
| `SEC-COMMIT` / `SEC-HOOKS` | Commits `SPEC-147:`; `git diff --cached` revisado; hooks activos | `CA-SEC-COMMIT` |
| `SEC-INCIDENT` | Si se filtra un token: revocar, rotar y documentar | `CA-SEC-INCIDENT` |
| `TEST-LOCAL` | `go test ./...` + MCP Inspector + Claude Code HTTP antes del conector | `CA-TEST-LOCAL` |
| `TEST-SERVER` | `./scripts/check-server.sh` antes de avisar del entorno local | `CA-TEST-SERVER` |
| `TEST-USER-FLOW` | `pending_release` solo con pruebas del usuario en claude.ai | `CA-TEST-USER-FLOW` |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-147`, rama `feature/SPEC-147` | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Funnel, deploy, merge y push requieren aprobación explícita | `CA-DELIV-CONFIRM` |
| `DELIV-MCP-DOCS` | `docs/mcp/` actualizado en el mismo commit (transporte, OAuth, operación) | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | Redacción de docs delegada a un subagente | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-DOCKER` | Imagen multi-arch `linux/amd64,linux/arm/v7,linux/arm64` si se containeriza | `CA-DELIV-DOCKER` |
| `DELIV-DEPLOY` | Deploy al iHost aprobado por el usuario; health check posterior | `CA-DELIV-DEPLOY` |
| `UI-*` / `UX-*` | N/A: no hay frontend ni pantallas; el login OAuth es HTML mínimo sin React | — |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Transporte Streamable HTTP**: el binario MCP debe arrancar en modo HTTP (`P40LA_MCP_LISTEN`) sirviendo `/mcp` con `mcp.NewStreamableHTTPHandler`, además del modo stdio actual (default).
2. **REQ-002 — Bind local**: el listener HTTP solo acepta `127.0.0.1`/`::1`; nunca `0.0.0.0` en producción. La exposición la hace Funnel.
3. **REQ-003 — OAuth 2.1 protegido**: `/mcp` exige bearer token válido. Sin token → `401` con `WWW-Authenticate: Bearer resource_metadata="<public_url>/.well-known/oauth-protected-resource"`.
4. **REQ-004 — Descubrimiento**: servir `/.well-known/oauth-protected-resource` (RFC 9728) y `/.well-known/oauth-authorization-server` (RFC 8414) con `resource` igual a la URL pública exacta.
5. **REQ-005 — Registro dinámico**: `POST /register` (RFC 7591) acepta clientes públicos y solo los redirect URIs permitidos (`https://claude.ai/api/mcp/auth_callback`, loopback de Claude Code).
6. **REQ-006 — Autorización**: `GET /authorize` con `response_type=code`, `code_challenge` S256 obligatorio, `state`; login de usuario único contra un hash bcrypt y pantalla de consentimiento que muestra el host de callback.
7. **REQ-007 — Token**: `POST /token` con `authorization_code` + PKCE y `refresh_token`; refresh rotativo; tokens opacos guardados hasheados.
8. **REQ-008 — Persistencia**: registros de clientes, códigos, tokens y auditoría en un volumen propio (`/data`), archivos `0600` con escritura atómica.
9. **REQ-009 — Doble gate de escritura**: las tools de escritura requieren `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` en modo HTTP. Default: solo lectura.
10. **REQ-010 — Auditoría**: toda llamada a `/mcp` registra `ts`, `tool`, `result`, `duration_ms`, `subject` y `request_id`; nunca el body ni datos financieros.
11. **REQ-011 — Límites**: body máximo 1 MiB, timeout de 30 s, rate limit configurable (default 60 req/min por token) con `429` y `Retry-After`.
12. **REQ-012 — Endurecimiento HTTP**: `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, sin CORS, sin listado de directorios, `/healthz` solo en loopback.
13. **REQ-013 — Revocación**: comando local o endpoint loopback para revocar todos los tokens y clientes; rotar el secreto de OAuth sin reiniciar.
14. **REQ-014 — Publicación con Funnel**: documentar y automatizar (script/runbook) `tailscale funnel` para `https://<nodo>.<tailnet>.ts.net` → `http://127.0.0.1:8090`, con `AllowFunnel` explícito y sin otros paths.
15. **REQ-015 — Documentación**: `docs/mcp/` actualizado con transporte HTTP, OAuth, Funnel, configuración, operación y troubleshooting.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-016 — Allowlist de egress**: opción de aceptar solo rangos de Anthropic (configurable, deshabilitada por defecto por rotación de rangos).
2. **REQ-017 — Imagen MCP**: `Dockerfile.mcp` multi-arch y add-on en el iHost con red host, volumen de estado y reinicio automático.
3. **REQ-018 — Modo híbrido**: permitir `P40LA_MCP_REMOTE_WRITES=false` aunque el proceso local tenga escrituras, para usar Claude web en lectura y Claude Code local con escritura.
4. **REQ-019 — Rotación de logs**: el JSONL de auditoría rota por tamaño (ej. 5 MB × 3) para no llenar el volumen del iHost.
5. **REQ-020 — Métricas mínimas**: contadores en memoria expuestos en `/healthz` (llamadas, errores, tokens activos), sin datos sensibles.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-021 — Scopes por tool**: separar lectura/escritura en scopes OAuth para que Claude pida solo lo necesario.
2. **REQ-022 — Aprobación humana en escrituras remotas**: confirmación fuera de banda (Telegram/UI) antes de ejecutar una tool destructiva.
3. **REQ-023 — Headers estáticos**: soporte opcional si la organización obtiene acceso a `static_headers` de Claude (beta), como alternativa a OAuth.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: `initialize` + `tools/list` < 300 ms en LAN; RSS del proceso < 50 MB.
- **Seguridad**: HTTPS 443 terminado por Tailscale; sin HTTP público; tokens opacos, hasheados y rotativos; sin secretos en logs.
- **Disponibilidad**: si el nodo que corre el MCP está apagado, el conector falla de forma limpia; Funnel no debe quedar como proxy abierto a otros puertos.
- **iHost**: si se despliega ahí, presupuesto de RAM medido y rollback con `tailscale funnel reset`.
- **Privacidad**: los resultados de las tools viajan a Anthropic; el usuario debe aceptarlo explícitamente.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- Documentación de Claude: los conectores remotos se ejecutan desde la infraestructura de Anthropic, exigen HTTPS en **443**, rechazan IPs privadas y **CGNAT `100.64.0.0/10`**, y soportan Streamable HTTP. Claude es **IPv4-only**: el hostname debe publicar un **A record** público.
- Tailscale Funnel: el DNS público resuelve el nombre del nodo a los relays de Funnel (IPs públicas), no a la IP del iHost; puede tardar hasta ~10 minutos en propagar. Sirve para el caso porque expone 443 con certificado válido.
- Autenticación: OAuth 2.1 (DCR/CIMD) por defecto; `static_headers` es beta para organizaciones limitadas; *sin auth* no es aceptable para este caso.
- SDK `modelcontextprotocol/go-sdk v1.8.0`: `NewStreamableHTTPHandler`, `StreamableHTTPOptions{Stateless, JSONResponse}`.
- Tailscale: `serve.json` ya versionado en `deploy/tailscale/` con `AllowFunnel: false`; Funnel publica en 443 con certificado automático. El iHost ya corre Tailscale y está en la tailnet.
- Estado actual: el MCP es stdio-only; no hay listener ni OAuth.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. HTTP MCP en el iHost + Funnel desde el iHost | Siempre disponible; API en `127.0.0.1:8088`; sin laptop encendida | Suma proceso/RAM al iHost; endpoint público junto a la DB; toca el add-on Tailscale | ✅ Destino final |
| B. HTTP MCP en la laptop + Funnel desde la laptop | Cero impacto en iHost; iteración rápida | Requiere laptop encendida; expone un host de la LAN | ✅ Primera fase |
| C. Solo Claude Code/Desktop local (stdio) | Sin exposición, sin OAuth | No sirve para claude.ai/móvil | ❌ No cumple el objetivo |
| D. Conector remoto sin auth (URL secreta) | Simple | Prohibido por MCP/Claude y por `SEC-*`; cualquiera con la URL escribe | ❌ Rechazada |
| E. Static headers (beta) | Sin implementar OAuth | No disponible para todos; dependencia externa | ⚠️ Alternativa P2 |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — OAuth 2.1 propio, usuario único, sin dependencias nuevas**
- **Contexto**: claude.ai necesita OAuth; el proyecto no tiene proveedor de identidad.
- **Decisión**: implementar un AS mínimo en `internal/mcp/server/oauth` con `net/http`, `crypto/rand`, `crypto/sha256` y `golang.org/x/crypto/bcrypt` (ya en `go.mod`). DCR abierto solo a redirect URIs allowlisted; un solo usuario con password en env/hash.
- **Consecuencias**: superficie de seguridad propia que debe testearse y auditarse; a cambio, cero servicios externos y control total del flujo.

**ADR-002 — Funnel, no Serve, para el conector remoto**
- **Contexto**: Serve solo es alcanzable dentro de la tailnet; Claude entra desde Internet.
- **Decisión**: Funnel en 443 con `AllowFunnel` para el host exacto y proxy únicamente a `/mcp` y `/.well-known/*`.
- **Consecuencias**: el endpoint es público; la seguridad depende de OAuth + rate limit + auditoría. Rollback: `tailscale funnel reset`.

**ADR-003 — Estado en archivo, no en SQLite**
- **Contexto**: el MCP no tiene acceso a la DB y no debe acoplarse a su esquema.
- **Decisión**: JSON con escritura atómica en `/data/oauth.json` y `/data/audit.jsonl`, permisos `0600`.
- **Consecuencias**: simple y portable; sin consultas concurrentes complejas (mutex en proceso).

**ADR-004 — Escrituras dobles-gateadas**
- **Contexto**: el usuario pidió habilitar escrituras, pero el endpoint es público.
- **Decisión**: `P40LA_MCP_ALLOW_WRITES` (proceso) **y** `P40LA_MCP_REMOTE_WRITES` (transporte) deben ser `true`. El deploy puede dejar el remoto en lectura y habilitar escritura solo en Claude Code local.
- **Consecuencias**: menos riesgo de un `services_create` accidental desde el móvil; se puede habilitar por entorno.

**ADR-005 — Primera fase en la laptop, luego iHost**
- **Contexto**: el iHost tiene recursos limitados y el add-on Tailscale ya está configurado con `AllowFunnel: false`.
- **Decisión**: validar transporte + OAuth + Funnel en la laptop; medir RSS y recién después evaluar el contenedor en el iHost.
- **Consecuencias**: dos despliegues documentados; el de iHost requiere presupuesto de memoria y aprobación.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
claude.ai / Claude móvil
        │ HTTPS 443 (OAuth 2.1 + Bearer)
        ▼
Tailscale Funnel  <nodo>.<tailnet>.ts.net
        │ proxy 127.0.0.1:8090
        ▼
p40la-mcp (HTTP mode)
  ├── /mcp                Streamable HTTP
  ├── /.well-known/*       OAuth discovery
  ├── /authorize /token    OAuth 2.1 + PKCE
  ├── /register            DCR
  ├── /healthz             loopback
  └── audit.jsonl          volumen /data
        │ HTTP + X-API-Key
        ▼
p40la-ihost :8088  (LAN o 127.0.0.1 si corre en el iHost)
```

### 5.2 Componentes

#### 5.2.1 `cmd/mcp-server`
- **Responsabilidad**: elegir transporte según env (`stdio` default, `http` con `P40LA_MCP_LISTEN`), cargar config y arrancar.
- **Cambios**: flag `-hash-password`, detección por env; logs a stderr; graceful shutdown.

#### 5.2.2 `internal/mcp/server`
- **Responsabilidad**: dos listeners loopback (público y admin), router, middleware (auth, límites, headers, rate limit), health. El listener admin no se publica con Funnel, así `/healthz` y `/admin/revoke` no quedan accesibles desde Internet.
- **Archivos**: `server.go`, `middleware.go`; `limiter/` y `oauth/` como subpaquetes.

#### 5.2.3 `internal/mcp/server/oauth`
- **Responsabilidad**: discovery, DCR, authorize, token, refresh, revocación, persistencia.
- **Archivos**: `provider.go`, `store.go`, `handlers.go`, `pkce.go`, `templates.go` (HTML mínimo).
- **Dependencias**: `x/crypto/bcrypt`; sin JWT (tokens opacos).

#### 5.2.4 `internal/mcp/audit`
- **Responsabilidad**: writer JSONL con rotación, redacción y mutex.
- **Archivos**: `audit.go`, `rotate.go`.

#### 5.2.5 `internal/mcp/config`
- **Responsabilidad**: nuevas variables, validación (URL pública absoluta, HTTPS, puerto loopback), defaults seguros.

### 5.3 Modelo de datos (estado local)

```
Client        { client_id, redirect_uris[], created_at }
AuthCode      { code_hash, client_id, redirect_uri, challenge, expires_at, used }
Token         { access_hash, refresh_hash, client_id, subject, scopes[], expires_at }
AuditEntry    { ts, request_id, tool, result, duration_ms, subject }
```

- Sin PII; `subject` es un identificador opaco (`owner`).
- Access token: 1 h; refresh: 30 días con rotación.

### 5.4 APIs / Contratos

#### `POST /mcp`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: application/json`, `Accept: application/json, text/event-stream`.
- **Sin token**: `401` + `WWW-Authenticate` con `resource_metadata`.
- **Rate limit**: `429` + `Retry-After`.
- **Cuerpo**: JSON-RPC MCP estándar.

#### `GET /.well-known/oauth-protected-resource`
```json
{
  "resource": "https://<nodo>.<tailnet>.ts.net/mcp",
  "authorization_servers": ["https://<nodo>.<tailnet>.ts.net"],
  "scopes_supported": ["mcp:read", "mcp:write"]
}
```

#### `GET /.well-known/oauth-authorization-server`
```json
{
  "issuer": "https://<nodo>.<tailnet>.ts.net",
  "authorization_endpoint": "https://<nodo>.<tailnet>.ts.net/authorize",
  "token_endpoint": "https://<nodo>.<tailnet>.ts.net/token",
  "registration_endpoint": "https://<nodo>.<tailnet>.ts.net/register",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"],
  "token_endpoint_auth_methods_supported": ["none"]
}
```

#### `POST /register`
```json
{
  "client_name": "Claude",
  "redirect_uris": ["https://claude.ai/api/mcp/auth_callback"],
  "token_endpoint_auth_method": "none",
  "grant_types": ["authorization_code", "refresh_token"]
}
```
- Solo se aceptan redirect URIs allowlisted; si no, `400 invalid_redirect_uri`.

#### `GET /authorize`
- Parámetros: `client_id`, `redirect_uri`, `response_type=code`, `code_challenge`, `code_challenge_method=S256`, `state`, `scope`.
- Muestra formulario de password + consentimiento con el host de callback.
- Error de credenciales: no revela si el usuario existe (usuario único).

#### `POST /token`
- `application/x-www-form-urlencoded`.
- `authorization_code`: valida PKCE S256, un solo uso, expiración 5 min.
- `refresh_token`: rota el refresh y invalida el anterior.
- Errores RFC 6749 (`invalid_grant`, `invalid_client`).

### 5.5 Variables de entorno

| Variable | Default | Descripción |
|---|---|---|
| `P40LA_MCP_TRANSPORT` | `stdio` | `stdio` o `http` |
| `P40LA_MCP_LISTEN` | `127.0.0.1:8090` | Bind loopback |
| `P40LA_MCP_ADMIN_LISTEN` | `127.0.0.1:8091` | Listener admin (`/healthz`, `/admin/revoke`), nunca proxificado por Funnel |
| `P40LA_MCP_PUBLIC_URL` | — | `https://<nodo>.<tailnet>.ts.net` (obligatoria en HTTP, sin `/mcp`) |
| `P40LA_MCP_OAUTH_SUBJECT` | `owner` | Identificador opaco del usuario |
| `P40LA_MCP_OAUTH_PASSWORD_HASH` | — | bcrypt; nunca la password en claro |
| `P40LA_MCP_STATE_DIR` | `/data` | Estado OAuth y auditoría |
| `P40LA_MCP_REMOTE_WRITES` | `false` | Habilita escrituras por HTTP |
| `P40LA_MCP_RATE_LIMIT` | `60` | Requests por minuto por token |
| `P40LA_MCP_AUDIT_MAX_MB` | `5` | Tamaño de rotación |
| `P40LA_BASE_URL` / `P40LA_API_KEY` | — | API de p40la-ihost (SPEC-145) |

### 5.6 Dependencias

- **Internas**: `internal/mcp/config`, `api`, `registry`, `tools/all`, `docs/mcp/`.
- **Externas**: `modelcontextprotocol/go-sdk` y `golang.org/x/crypto` (ya presentes). Sin librerías OAuth/JWT nuevas.

---

## 6. Criterios de Aceptación

### 6.1 Funcionales

- [ ] **CA-MCP-REMOTE-001**: modo HTTP sirve `/mcp` con Streamable HTTP; `initialize` y `tools/list` funcionan con MCP Inspector.
- [ ] **CA-MCP-REMOTE-002**: discovery de recurso y AS devuelve `resource`/`issuer` exactos a la URL pública.
- [ ] **CA-MCP-REMOTE-003**: DCR acepta el callback de Claude y rechaza redirect URIs no allowlisted.
- [ ] **CA-MCP-REMOTE-004**: flujo authorize → code → token con PKCE S256; PKCE inválido o code reutilizado falla con `invalid_grant`.
- [ ] **CA-MCP-REMOTE-005**: `/mcp` sin token devuelve `401` con `WWW-Authenticate` correcto; con token inválido/expirado, `401`.
- [ ] **CA-MCP-REMOTE-006**: refresh rota el token y el anterior deja de servir.
- [ ] **CA-MCP-REMOTE-007**: claude.ai agrega el conector, completa OAuth y ejecuta una tool de lectura.
- [ ] **CA-MCP-REMOTE-008**: con `P40LA_MCP_REMOTE_WRITES=false`, las tools de escritura responden `writes_disabled` y no llaman a la API.
- [ ] **CA-MCP-REMOTE-009**: con ambos flags en `true`, una escritura se ejecuta y queda auditada.
- [ ] **CA-MCP-REMOTE-010**: rate limit devuelve `429` con `Retry-After`; body > 1 MiB devuelve `413`.
- [ ] **CA-MCP-REMOTE-011**: auditoría JSONL sin bodies ni PII; rota por tamaño.
- [ ] **CA-MCP-REMOTE-012**: `/healthz` responde solo en loopback; el puerto no es accesible desde la LAN.
- [ ] **CA-MCP-REMOTE-013**: revocación invalida todos los tokens y obliga a reautorizar.
- [ ] **CA-MCP-REMOTE-014**: Funnel publica solo 443 → `127.0.0.1:8090`; `tailscale funnel reset` revierte.
- [ ] **CA-MCP-REMOTE-015**: certificado válido en `https://<nodo>.<tailnet>.ts.net` y redirect HTTP→HTTPS.

### 6.2 No funcionales

- [ ] **CA-NF-001**: RSS < 50 MB y `initialize` < 300 ms en LAN.
- [ ] **CA-NF-002**: sin dependencias nuevas en `go.mod` salvo `x/crypto` (ya presente).
- [ ] **CA-NF-003**: `go build ./...` y `go test ./...` verdes.
- [ ] **CA-NF-004**: imagen multi-arch (si se publica) con los 3 manifests.
- [ ] **CA-NF-005**: ningún secreto, IP real ni nombre de tailnet en el repo.

### 6.3 Testing

- **Unit**: PKCE, hash/rotación de tokens, allowlist de redirect, rate limit, redacción de auditoría, config inválida.
- **Integración HTTP**: `httptest` para discovery, DCR, authorize, token, 401/429/413.
- **Protocolo MCP**: `mcp.NewInMemoryTransports` o cliente Streamable HTTP contra el handler.
- **E2E**: MCP Inspector → Claude Code HTTP (con bearer de prueba) → claude.ai (OAuth) → Funnel.
- **Manual**: copia de producción en `/tmp`, tool de lectura y una escritura controlada.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Modo HTTP + `/healthz` + tests locales (sin auth, loopback) | 0.5 día | SPEC-145/146 en `main` |
| 2 | OAuth 2.1 + DCR + PKCE + persistencia | 2 días | Fase 1 |
| 3 | Endurecimiento: rate limit, headers, auditoría, revocación | 1 día | Fase 2 |
| 4 | Funnel en laptop + validación con MCP Inspector y Claude Code | 0.5 día | Fase 3 |
| 5 | Conector claude.ai + pruebas del usuario | 0.5 día | Fase 4 |
| 6 | Imagen multi-arch y evaluación en iHost (opcional) | 1 día | Fase 5 |
| 7 | Docs MCP (subagente) + release | 0.5 día | Fases 5–6 |

### 7.2 Milestones

1. **MVP**: HTTP + OAuth + Funnel en laptop, lectura habilitada.
2. **V1.0**: escrituras dobles-gateadas, auditoría, rate limit, conector claude.ai validado.
3. **V1.1**: add-on en iHost con presupuesto de memoria medido.

### 7.3 Estado de implementación

| Ítem | Estado |
|---|---|
| Modo HTTP (`/mcp`, Streamable HTTP, bind loopback) | ✅ Implementado |
| Listener admin separado (`/healthz`, `/admin/revoke`) | ✅ Implementado |
| OAuth 2.1 (discovery, DCR, PKCE S256, refresh rotativo, revocación) | ✅ Implementado |
| Tokens opacos hasheados y estado `0600` en `P40LA_MCP_STATE_DIR` | ✅ Implementado |
| Doble gate de escritura (`ALLOW_WRITES` + `REMOTE_WRITES`) | ✅ Implementado |
| Auditoría JSONL con rotación y sin argumentos | ✅ Implementado |
| Rate limit, body limit 1 MiB, headers de seguridad | ✅ Implementado |
| `Dockerfile.mcp` + `scripts/build-mcp-image.sh` multi-arch | ✅ Listo (sin publicar) |
| `deploy/tailscale/serve-funnel.example.json` | ✅ Listo |
| Tests unitarios/integración del MCP | ✅ `go test ./...` verde |
| E2E local: DCR → authorize → token → `initialize` → `tools/list` → `bills_list` | ✅ Contra copia de producción en `/tmp` (11 facturas, auditoría con `subject=owner`) |
| Conexión de Claude Code por HTTP con bearer token | ✅ `claude mcp list` → `✓ Connected` (entrada de prueba eliminada y tokens revocados) |
| Contenedor `Dockerfile.mcp` (build amd64 + smoke test de `/healthz`) | ✅ Validado localmente; publicación multi-arch pendiente |
| Documentación `docs/mcp/` (HTTP, OAuth, Funnel, seguridad, operación) | ✅ Actualizada y revisada |
| Funnel real habilitado | ⏳ Requiere admin de Tailscale y aprobación del usuario |
| Conector en claude.ai y pruebas del usuario | ⏳ Pendiente (depende de Funnel) |
| Deploy en iHost | ⏳ Pendiente (fase opcional, medir RSS) |

> La validación E2E se hizo con datos de una copia de producción; no se pegaron montos, nombres ni IDs reales en esta spec.

### 7.4 Release

- **Autorización del usuario**: 2026-11-23, orden explícita de pasar a `released` y ejecutar `./scripts/release.sh` para publicar la imagen del add-on.
- **Alcance liberado**: transporte HTTP, OAuth 2.1, doble gate de escritura, auditoría, límites, documentación y artefactos de build del MCP.
- **Excepción documentada**: Funnel y la validación del conector de claude.ai quedan como pasos operativos posteriores porque requieren exponer el servicio en Internet y aprobación de infraestructura. No bloquean el release del binario ni la publicación de la imagen del add-on web.
- **Versión y commit**: `v0.5.17` (imagen multi-arch publicada por GitHub Actions), merge `5ea27291`, bump `cf5f5fa7`, tag `v0.5.17`.
- **Post-release**: habilitar Funnel, agregar el conector en claude.ai y, si se desea, desplegar el MCP en el iHost; validar con la checklist de aceptación pendiente.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Endpoint público con escrituras | Media | Alto | OAuth obligatorio, doble flag, auditoría, rate limit, revocación |
| Bug en el AS OAuth (open redirect, PKCE, replay) | Media | Alto | Allowlist estricta, tests adversos, revisión, tokens opacos y rotación |
| Datos financieros enviados a Anthropic | Alta | Medio | Aceptación explícita del usuario; modo lectura por defecto; documentar |
| Filtración de tokens o password | Baja | Alto | Hash bcrypt, archivos `0600`, logs sin secretos, rotación, `SEC-INCIDENT` |
| Funnel expone más de lo previsto | Baja | Alto | Config declarativa solo `/mcp` + `/.well-known/*`; verificación; `funnel reset` |
| Funnel sin A público o DNS sin propagar | Media | Alto | Verificar con resolutor público que el hostname tenga A IPv4 antes de agregar el conector; esperar hasta 10 min; reintentar |
| iHost sin RAM suficiente | Media | Medio | Fase en laptop primero; medir RSS; rollback |
| API key global sin scopes | Alta | Medio | Aceptar en V1; scopes por tool en P2 |
| Claude cambia requisitos de conectores | Media | Medio | Docs y pruebas E2E; `static_headers` como alternativa P2 |
| Rate limit insuficiente | Baja | Bajo | Configurable + límites de body/timeout |

---

## 9. Notas y Referencias

- Claude, conectores remotos y autenticación: requisitos de HTTPS 443, rechazo de CGNAT/privadas, OAuth 2.1, `static_headers` beta.
- `docs/mcp/` (SPEC-145): arquitectura, tools, API key, operación.
- `docs/ihost.md` y `docs/ihost-pwa-tailscale-setup.md`: Tailscale Serve/Funnel en el iHost.
- `deploy/tailscale/serve.json`: `AllowFunnel: false` actual.
- Reglas: `docs/rules/{10-architecture,40-data,50-security,60-testing,70-delivery}.md`.
- Ejemplos sintéticos: `https://demo-node.example.ts.net/mcp`, `192.0.2.10`, `demo@example.com`.

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | opencode | Creación inicial de la especificación |
| 2026-11-23 | opencode | Implementación del transporte HTTP, OAuth 2.1, auditoría, límites, contenedor y documentación; E2E local y preparación de release |
| 2026-11-23 | opencode | Release v0.5.17: merge `5ea27291`, bump `cf5f5fa7`, tag `v0.5.17`; Funnel y conector claude.ai quedan como operación posterior |
