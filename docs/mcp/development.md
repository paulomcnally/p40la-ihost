---
title: "Desarrollo de tools"
sidebar_position: 7
description: "Cómo agregar una tool nueva al servidor MCP"
---

# Desarrollo de tools

## Estructura

```
cmd/mcp-server/                 # entrypoint; stdio por defecto y HTTP opcional
internal/mcp/
├── config/                     # variables de entorno (stdio y HTTP)
├── api/                        # cliente HTTP + tipos DTO + interfaz API
├── registry/                   # montaje de conjuntos de tools
├── server/                     # modo HTTP (SPEC-147)
│   ├── server.go               # listeners, auth, límites, health y revoke
│   ├── middleware.go           # auditoría de tools/call
│   ├── oauth/                  # discovery, DCR, PKCE, token y estado
│   │   ├── provider.go
│   │   ├── handlers.go
│   │   └── store.go
│   └── limiter/                # ventana fija en memoria
├── audit/                      # JSONL con rotación
└── tools/
    ├── all/register.go         # único punto que importa los dominios
    ├── toolkit/                # helpers compartidos (JSONResult, ToolError, APIError)
    ├── bills/                  # tools de facturas
    │   ├── tools.go            # Toolset + Register + writeMode
    │   ├── period.go           # períodos YYYY-MM
    │   ├── list.go
    │   ├── generate.go
    │   └── pay.go
    ├── autos/                  # tools de autos, mantenimientos, inspecciones y seguros
    │   ├── tools.go            # Toolset + Register + writeMode
    │   ├── results.go          # apiError y aliases al toolkit
    │   ├── list.go
    │   ├── create.go
    │   ├── update.go
    │   ├── maintenance.go
    │   ├── inspection.go
    │   └── insurance.go
    ├── entities/               # búsqueda, altas y edición de hogares, instituciones, bancos y servicios
    │   ├── tools.go            # Toolset + Register + checkWrites
    │   ├── resolve.go          # resolución por nombre/ID y errores accionables
    │   ├── search.go           # entities_search (solo lectura)
    │   ├── homes.go
    │   ├── institutions.go
    │   ├── banks.go
    │   └── services.go
    └── banking/                # tools de cuentas y tarjetas (SPEC-152)
        ├── tools.go            # Toolset + Register + gate de escritura
        ├── resolve.go          # banco, moneda, duplicados y ranking
        ├── results.go          # helpers de resultado/error
        ├── util.go             # límite de resultados y helpers
        ├── accounts.go
        └── cards.go
```

Las tools no saben por qué transporte llegan: `cmd/mcp-server` monta el mismo `mcp.Server` en stdio o en Streamable HTTP. La compuerta de escritura sí consulta el transporte (`Config.WritesEnabled()`), porque el modo HTTP exige la segunda variable.

## Agregar una tool

1. **Declará la tool en el manifiesto de impacto** `docs/mcp/tools/impact.json`: `docs`, `entrypoints`, `dependencies`, `endpoints` y `tests`. Registrala con `mcp.AddTool(server, &mcp.Tool{Name: "..."}, ...)` y el nombre literal bajo `internal/mcp/tools/`; el analizador no resuelve variables. El plugin `rules-guard` bloquea un `mcp.AddTool` cuyo nombre no esté en el manifiesto. Ver [tools/impact.md](tools/impact.md).
2. **Definí el contrato HTTP** en el backend si la operación requiere lógica de negocio. La tool no debe contener SQL ni reglas de dominio.
3. **Agregá el método** en `internal/mcp/api` (tipo DTO + método del cliente + interfaz `API`).
4. **Creá el paquete de tools** `internal/mcp/tools/<dominio>` con un `Toolset` y un `Register`.
5. **Registrá el dominio** en `internal/mcp/tools/all/register.go` con un `r.Add(...)`.
6. **Congelá el contrato** del dominio y revisá el diff del golden (ver [Contrato y guardrails](#contrato-y-guardrails)).
7. **Escribí tests** con una API falsa y, si aplica, un test de protocolo con `mcp.NewInMemoryTransports`.
8. **Documentá** la tool en `docs/mcp/tools/<dominio>.md` (por ejemplo, `docs/mcp/tools/banking.md` para cuentas bancarias y tarjetas) y actualizá los catálogos de `docs/mcp/README.md` y `docs/mcp/tools/README.md`.
9. **Verificá** el manifiesto y los tests:
   ```bash
   ./scripts/check-mcp-impact.sh --check-map
   go test ./internal/mcp/...
   go test ./internal/api -run TestMCPBackendContract
   ```

## Contrato de una tool

```go
type ExampleInput struct {
    ServiceID int64   `json:"service_id" jsonschema:"ID del servicio (obligatorio)"`
    Note      *string `json:"note,omitempty" jsonschema:"Texto opcional"`
}

func (t *Toolset) example(ctx context.Context, _ *mcp.CallToolRequest, input ExampleInput) (*mcp.CallToolResult, any, error) {
    if input.ServiceID <= 0 {
        return toolError("invalid_request", "service_id debe ser mayor que cero", "")
    }
    result, err := t.client.Example(ctx, input.ServiceID)
    if err != nil {
        return apiError(err)
    }
    return jsonResult(result)
}
```

Reglas:

- Los campos obligatorios no llevan `omitempty`; los opcionales usan punteros.
- Las descripciones de `jsonschema` son la documentación que ve el modelo.
- Los errores de negocio devuelven `IsError=true` con `{error, message, hint}`.
- Los errores de la API se traducen con `apiError` para no filtrar detalles.
- Las escrituras deben pasar por `writeMode` y respetar `P40LA_MCP_ALLOW_WRITES`.
- Toda tool que escribe debe ser idempotente o explicar por qué no lo es.
- Las tools que aceptan nombre o ID resuelven contra la API con un GET, comparan sin distinguir mayúsculas ni acentos y devuelven `not_found`/`ambiguous_reference` con opciones accionables. El patrón está en `internal/mcp/tools/entities/resolve.go`.

## Registro

```go
// internal/mcp/tools/all/register.go
func Register(r *registry.Registry, client api.FullAPI, cfg config.Config) {
    r.Add(func(server *mcp.Server) {
        bills.Register(server, client, cfg)
    })
    r.Add(func(server *mcp.Server) {
        autos.Register(server, client, cfg)
    })
}
```

`cmd/mcp-server` no conoce los paquetes de dominio: solo monta el registry.

## Tests

- **Cliente HTTP**: `httptest` verificando `Authorization: Bearer`, 401 y mapeo de errores.
- **Tool**: API falsa que captura requests y devuelve respuestas; reloj fijo para períodos.
- **Protocolo**: `initialize`, `tools/list` y `tools/call` con transporte en memoria.
- **Contrato backend↔MCP**: router real, DB temporal, API key y cliente MCP; verifica rutas, tags JSON, auth y `source=mcp`.
- **Backend**: DB temporal con migraciones; nunca la DB local del usuario.
- **OAuth**: flujo completo con PKCE, contraseña incorrecta, redirect no allowlisted, token vencido.
- **Servidor HTTP**: 401 sin token, 429, 413, `/healthz` y revocación en el listener admin.
- **Auditoría**: JSONL y rotación por tamaño.

```bash
go test ./internal/mcp/...
go test ./internal/services/ ./internal/api/
go build ./...
```

### Contrato y guardrails

**Manifiesto de impacto.** `docs/mcp/tools/impact.json` declara, por tool, sus `entrypoints`, dependencias del backend, endpoints, tests y documentación; `shared.affects_all` lista las rutas que afectan a todas. `./scripts/check-mcp-impact.sh --check-map` valida que toda tool registrada con `mcp.AddTool` tenga entrada y que cada ruta exista. Si agregás, renombrás o eliminás una tool, o cambia una dependencia, actualizá el manifiesto en el mismo ciclo. El formato completo está en [tools/impact.md](tools/impact.md).

**Contrato congelado.** `docs/mcp/tools/contracts.json` es el golden de `tools/list` (nombre, descripción, esquema de entrada/salida y anotaciones). `TestToolContracts` vive en `internal/mcp/tools/all/contract_test.go` y cubre el catálogo completo que registra `internal/mcp/tools/all/register.go`; al agregar un dominio, el test falla hasta regenerar el golden. Si el cambio de contrato es intencional:

```bash
go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts
```

Revisá el diff de `contracts.json`, declaralo en `## Impacto en tools MCP` y volvé a correr el test sin el flag. **Prohibido** regenerar el golden sin justificación.

**Contrato backend↔MCP.** `TestMCPBackendContract` (`internal/api/mcp_backend_contract_test.go`) recorre API key → router → handler → servicio → storage → JSON con una DB temporal. Detecta drift de rutas, tags JSON, autenticación o `source=mcp` sin usar datos de producción. No tiene golden: si el cambio de comportamiento es intencional, se ajusta el test y se declara en la spec.

**Hook.** `.githooks/pre-commit` ejecuta dos guardrails cuando hay archivos staged de la superficie MCP:

```bash
./scripts/check-mcp-docs.sh --staged
./scripts/check-mcp-impact.sh --staged --gate
```

El segundo valida toda spec nueva staged (`## Impacto en tools MCP` y `Validación subagente:`) aunque no haya tools afectadas. Si hay impacto, además resuelve la spec en curso, exige que cada tool afectada aparezca nombrada y que `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract` pasen. Si falla, el commit se bloquea (`DELIV-MCP-IMPACT`). **Prohibido** `git commit --no-verify` sin autorización explícita del usuario.

## Probar el flujo OAuth local

El modo HTTP permite `http://` solo para loopback, lo que facilita probar OAuth sin Funnel.

```bash
# 1. Compilar y generar el hash
go build -o bin/p40la-mcp ./cmd/mcp-server
read -rs -p "Contraseña: " P40LA_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_PASSWORD" | ./bin/p40la-mcp -hash-password)
unset P40LA_PASSWORD

# 2. Arrancar con estado temporal
mkdir -p /tmp/p40la-mcp-state && chmod 700 /tmp/p40la-mcp-state
P40LA_BASE_URL="http://127.0.0.1:8088" \
P40LA_API_KEY="demo-api-key" \
P40LA_MCP_TRANSPORT=http \
P40LA_MCP_PUBLIC_URL="http://127.0.0.1:8090" \
P40LA_MCP_OAUTH_PASSWORD_HASH="$HASH" \
P40LA_MCP_STATE_DIR="/tmp/p40la-mcp-state" \
./bin/p40la-mcp
```

```bash
# 3. Discovery y challenge
curl -s http://127.0.0.1:8090/.well-known/oauth-protected-resource
curl -s http://127.0.0.1:8090/.well-known/oauth-authorization-server
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8090/mcp   # 401

# 4. Registro de un cliente de prueba (loopback /callback)
CLIENT_ID=$(curl -s -X POST http://127.0.0.1:8090/register \
  -H 'Content-Type: application/json' \
  -d '{"client_name":"Demo local","redirect_uris":["http://localhost:33418/callback"],"token_endpoint_auth_method":"none"}' \
  | jq -r .client_id)

# 5. PKCE
VERIFIER=$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=')
CHALLENGE=$(printf '%s' "$VERIFIER" | openssl dgst -sha256 -binary | openssl base64 -A | tr '+/' '-_' | tr -d '=')

# 6. Abrir la URL en el navegador, autorizar y copiar el code del redirect
echo "http://127.0.0.1:8090/authorize?response_type=code&client_id=$CLIENT_ID&redirect_uri=http%3A%2F%2Flocalhost%3A33418%2Fcallback&code_challenge=$CHALLENGE&code_challenge_method=S256&state=demo-state"
```

Después de autorizar, el navegador redirige a `http://localhost:33418/callback?code=...&state=demo-state`. Copiá el `code` y canjealo:

```bash
curl -s -X POST http://127.0.0.1:8090/token \
  -d grant_type=authorization_code \
  -d code="$CODE" \
  -d client_id="$CLIENT_ID" \
  -d redirect_uri="http://localhost:33418/callback" \
  -d code_verifier="$VERIFIER" | jq
```

```bash
# 7. Llamar /mcp con el access token
curl -s -X POST http://127.0.0.1:8090/mcp \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"demo","version":"0.1.0"}}}'
```

Para probar el protocolo completo, usá MCP Inspector (Node solo en la máquina de desarrollo):

```bash
npx @modelcontextprotocol/inspector
# Transport: Streamable HTTP
# URL: http://127.0.0.1:8090/mcp
# Header: Authorization: Bearer <access token>
```

Verificaciones útiles:

- Reutilizar el `code` debe fallar con `invalid_grant`.
- Un `code_verifier` distinto debe fallar con `invalid_grant`.
- `POST /admin/revoke` debe invalidar el access y el refresh.
- `P40LA_MCP_REMOTE_WRITES=false` debe devolver `writes_disabled` (por ejemplo, `bills_pay` con `dry_run=false` o un `bank_accounts_create`).
- `audit.jsonl` debe tener una línea por `tools/call`, sin argumentos.

## Documentación obligatoria

Todo cambio en el código del MCP se documenta **en el mismo commit**. La superficie MCP es:

| Área | Archivos |
|---|---|
| Servidor | `cmd/mcp-server/` (código no-test) |
| Dominio MCP | `internal/mcp/` (código no-test) |
| Servidor HTTP y OAuth | `internal/mcp/server/` (incluye `oauth/` y `limiter/`) |
| Auditoría | `internal/mcp/audit/` |
| Imagen y publicación | `Dockerfile.mcp`, `scripts/release.sh`, `.github/workflows/mcp-publish.yml`, `.github/workflows/tailscale-publish.yml`, `deploy/mcp/tailscale/` |
| API keys | `scripts/create-api-key.sh`, `internal/services/api_key.go`, `internal/api/api_key_handlers.go` |
| Generación y pago masivo | `internal/services/bill_bulk.go`, `internal/api/bill_bulk_handlers.go` |

`scripts/check-mcp-docs.sh` cubre `cmd/mcp-server/`, `internal/mcp/` y los archivos de API key/bills; la imagen, `scripts/release.sh` y `deploy/mcp/tailscale/` no disparan el hook por sí solos, pero también exigen actualizar `docs/mcp/` (regla `DELIV-MCP-DOCS`).

El release oficial es `./scripts/release.sh`: publica de forma independiente la app (`vX.Y.Z`), el MCP (`mcp-vX.Y.Z`, workflow `Build & Push MCP Image`) y el wrapper Tailscale (`tailscale-vX.Y.Z`, workflow `Build & Push Tailscale Image`) solo cuando hay cambios en la allowlist de cada artefacto. El detalle de opciones y verificación está en [operations.md](operations.md#release-y-publicación).

**`DELIV-MCP-DOCS`** — agregar o cambiar una tool, un contrato HTTP, una variable de entorno o un flujo operativo exige actualizar `docs/mcp/` en el **mismo commit** que el código. Según el cambio:

- Tool nueva o modificada → `docs/mcp/tools/<dominio>.md`, el manifiesto y el contrato en `docs/mcp/tools/`, y los catálogos de `docs/mcp/README.md` y `docs/mcp/tools/README.md` (banca: `docs/mcp/tools/banking.md`).
- Variable de entorno o configuración → `docs/mcp/configuration.md`.
- Transporte HTTP, OAuth o límites → `docs/mcp/http.md`, `docs/mcp/oauth.md` y `docs/mcp/configuration.md`.
- Publicación en Internet → `docs/mcp/funnel.md`.
- Flujo de API key, build, deploy, backup o verificación → `docs/mcp/operations.md` y `docs/mcp/api-key.md`.
- Modelo de amenazas, PII o permisos → `docs/mcp/security.md`.
- Contrato entre binario y servidor → `docs/mcp/architecture.md`.

**`DELIV-MCP-DOCS-SUBAGENT`** — la actualización de `docs/mcp/` se **delega siempre a un subagente** (`general` o `explore`) lanzado en paralelo. El brief debe incluir:

- archivos cambiados (rutas y resumen del cambio);
- contratos nuevos o modificados (tools, inputs, outputs, códigos de error);
- variables de entorno afectadas, con valores de ejemplo sintéticos;
- ejemplos sintéticos (`F000000000000001`, `demo-account-001`, `Proveedor Demo`, `demo@example.com`);
- reglas `SEC-*` aplicables (`SEC-NO-PII`, `SEC-SYNTHETIC`, `SEC-LOGS`).

El agente principal **revisa e integra** el resultado. Solo escribe la documentación él mismo si el subagente no está disponible, y lo justifica en la spec o en el mensaje de commit.

### Guardrail

`.githooks/pre-commit` ejecuta el guardrail cuando hay archivos staged de la superficie MCP:

```bash
./scripts/check-mcp-docs.sh --staged
```

- Si falla, se tocó código del MCP sin cambios en `docs/mcp/`: se documenta y se vuelve a commitear. **Prohibido** `git commit --no-verify` sin autorización explícita del usuario.
- Fail-safe: si el script no existe o falla internamente, el commit continúa; la revisión manual del diff sigue siendo obligatoria.

## Checklist de una tool nueva

- [ ] Input schema claro y con campos obligatorios/opcionales correctos.
- [ ] Descripción con límites, defaults y requisitos de escritura.
- [ ] Errores con código estable.
- [ ] Si la tool acepta referencias por nombre, resuelve contra la API y devuelve `not_found`/`ambiguous_reference` con opciones.
- [ ] Si la tool crea entidades con nombre, valida duplicados antes del POST (`already_exists` con el `id` existente).
- [ ] Tests de éxito, validación y error de API.
- [ ] Entrada en `docs/mcp/tools/impact.json` y `./scripts/check-mcp-impact.sh --check-map` verde.
- [ ] Contrato regenerado con `-update-contracts` y diff revisado, si el cambio de `tools/list` es intencional.
- [ ] `go test ./internal/mcp/...` y `go test ./internal/api -run TestMCPBackendContract` verdes.
- [ ] Documentación en `docs/mcp/tools/`.
- [ ] Si se tocó cualquier archivo de la superficie MCP, `docs/mcp/` se actualizó en el mismo commit y esa actualización se delegó a un subagente (`DELIV-MCP-DOCS`, `DELIV-MCP-DOCS-SUBAGENT`).
- [ ] Sin secretos, PII ni montos reales en tests o docs.
- [ ] `go build ./...` y `go test ./...` pasan.
