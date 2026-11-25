---
title: "Arquitectura del MCP"
sidebar_position: 4
description: "Componentes, flujo y decisiones del servidor MCP de p40la-ihost"
---

# Arquitectura

## Vista general

```
Claude Code / Desktop local                claude.ai / Claude móvil / Cowork
        │ stdio                                     │ HTTPS 443 (OAuth 2.1)
        ▼                                           ▼
┌──────────────────────┐                  ┌──────────────────────────┐
│ p40la-mcp (stdio)    │                  │ Tailscale Funnel         │
│ cmd/mcp-server       │                  │ demo-node.example.ts.net │
└──────────┬───────────┘                  └────────────┬─────────────┘
           │                                           │ 127.0.0.1:8090
           │                                           ▼
           │                                  ┌──────────────────────────┐
           │                                  │ p40la-mcp (HTTP)         │
           │                                  │ /mcp + OAuth 2.1         │
           │                                  │ audit.jsonl en /data     │
           │                                  └────────────┬─────────────┘
           │                                               │
           └───────────────────┬───────────────────────────┘
                               │ HTTP + Authorization: Bearer <API key>
                               ▼
                 ┌─────────────────────────────┐
                 │ p40la-ihost server          │
                 │ SessionOrAPIKeyMiddleware   │
                 │ GET  /api/services/{id}     │
                 │ GET  /api/services/{id}/bills
                 │ POST /api/services/{id}/bills/generate
                 │ POST /api/services/{id}/bills/pay
                 └──────────────┬──────────────┘
                                │ BillService → BillStorage
                                ▼
                        SQLite (bills, bill_history)
```

Los dos transportes comparten configuración de API, registry y tools. La única diferencia es cómo llega el JSON-RPC al binario y qué autenticación protege la entrada.

El diagrama usa facturas como ejemplo; las tools de autos recorren el mismo camino contra `/api/autos`, `/api/autos/{id}/maintenances`, `/api/autos/{id}/inspections` y `/api/autos/{id}/services`.

## Despliegue en el iHost: dos nodos Tailscale

El iHost ya publica el dashboard con **Serve** en 443, solo dentro de la tailnet. Como Tailscale no permite Serve y Funnel en el mismo puerto, el MCP usa un **nodo dedicado** con su propio estado y su propio Funnel; el nodo del dashboard queda intacto.

```
claude.ai / Claude móvil
        │ HTTPS 443 + OAuth
        ▼
┌──────────────────────────────────────────────┐
│ tailscale-mcp  (red host, TS_USERSPACE=true) │
│ p40la-mcp.example.ts.net                     │
│ Funnel 443 ──► http://127.0.0.1:8090         │
└───────────────────┬──────────────────────────┘
                    │ loopback
                    ▼
┌──────────────────────────────────────────────┐
│ p40la-mcp  (red host)                        │
│ :8090 /mcp + OAuth · :8091 admin             │
│ volúmenes: p40la-mcp-data, tailscale-mcp-state │
└───────────────────┬──────────────────────────┘
                    │ HTTP + bearer API key
                    ▼
┌──────────────────────────────────────────────┐
│ app p40la-ihost :8088                        │
└──────────────────────────────────────────────┘

Dispositivo en la tailnet
        │ Serve 443 (sin Funnel)
        ▼
ihost.example.ts.net ──► 127.0.0.1:8088   (nodo actual, intacto)
```

| Nodo | Contenedor / imagen | Publicación | Alcance |
|---|---|---|---|
| `ihost.example.ts.net` | Tailscale actual del add-on | Serve 443 → `127.0.0.1:8088` | Solo tailnet |
| `p40la-mcp.example.ts.net` | `tailscale-mcp` · `paulomcnally/p40la-mcp-tailscale:0.1.0` | Funnel 443 → `127.0.0.1:8090` | Internet + OAuth |

- `p40la-mcp` corre con la imagen `paulomcnally/p40la-mcp:0.5.0`, red host, `restart: unless-stopped` y estado en `/data`.
- `tailscale-mcp` está basado en `tailscale/tailscale`, usa `TS_USERSPACE=true` (no hay `/dev/net/tun` en el iHost), estado en `/var/lib/tailscale` y un `serve.json` incluido en la imagen.
- El nodo del dashboard **no se reinicia, no se reconfigura y no se publica**: su `serve.json` conserva `AllowFunnel: false`.
- El runbook completo está en [ihost.md](ihost.md) y el detalle de Funnel en [funnel.md](funnel.md).

## Componentes

| Componente | Ubicación | Responsabilidad |
|---|---|---|
| Entrypoint | `cmd/mcp-server/main.go` | Configuración, registro de tools, elección de transporte y `-hash-password` |
| Config | `internal/mcp/config` | Variables de entorno, API key, zona horaria, validación del modo HTTP |
| Cliente API | `internal/mcp/api` | `Authorization: Bearer`, llamadas HTTP y errores tipados |
| Registry | `internal/mcp/registry` | Punto de montaje de conjuntos de tools |
| Tools de facturas | `internal/mcp/tools/bills` | `bills_list`, `bills_generate`, `bills_pay` |
| Tools de autos | `internal/mcp/tools/autos` | `autos_list`, `autos_create`, `autos_update`, `autos_maintenance_register`, `autos_inspection_register`, `autos_insurance_register` |
| Tools de banca | `internal/mcp/tools/banking` | `bank_accounts_search`, `bank_accounts_create`, `bank_accounts_update`, `credit_cards_search`, `credit_cards_create`, `credit_cards_update` |
| Registro global | `internal/mcp/tools/all` | Único lugar que importa los paquetes de tools |
| Servidor HTTP | `internal/mcp/server/server.go` | Listeners público y admin, auth, rate limit, body limit, headers, health y revocación |
| Middleware de auditoría | `internal/mcp/server/middleware.go` | Registra solo `tools/call`, sin argumentos ni resultados |
| OAuth 2.1 | `internal/mcp/server/oauth` | Discovery, DCR, authorize con PKCE, token, refresh y revocación |
| Rate limit | `internal/mcp/server/limiter` | Ventana fija en memoria por token o IP |
| Auditoría | `internal/mcp/audit` | JSONL con rotación por tamaño |
| Imagen del MCP | `Dockerfile.mcp`, `scripts/release.sh`, `.github/workflows/mcp-publish.yml` | El release crea el tag `mcp-vX.Y.Z`; CI publica la imagen multi-arch independiente del add-on web |
| Imagen del nodo Tailscale | `deploy/mcp/tailscale/Dockerfile` | Wrapper de `tailscale/tailscale` con `serve.json` para Funnel 443 → MCP |
| Despliegue en el iHost | `deploy/mcp/` | Compose de referencia, env example y runbook del nodo dedicado |
| Funnel | `deploy/mcp/tailscale/serve-funnel.example.json` | Ejemplo declarativo de publicación en 443 |
| Backend masivo | `internal/services/bill_bulk.go` | Reglas de generación y pago por rango |
| Handlers HTTP | `internal/api/bill_bulk_handlers.go` | Parseo, códigos de estado y respuesta |
| API key global | `internal/services/api_key.go` | Generación, hash, validación, rotación y revocación |
| Middleware | `internal/api/middleware.go` | Sesión o API key global |
| Provisión | `scripts/create-api-key.sh` | Crea/rota la key con una sesión |
| Manifiesto de impacto | `docs/mcp/tools/impact.json` | Declara entrypoints, dependencias, endpoints y tests de cada tool |
| Analizador de impacto | `scripts/check-mcp-impact.sh`, `scripts/mcp-impact/` | Cruza archivos cambiados contra el manifiesto y aplica el gate de spec |
| Contrato congelado | `docs/mcp/tools/contracts.json`, `internal/mcp/tools/all/contract_test.go` | Congela `tools/list` y falla ante cambios no declarados |
| Contrato backend↔MCP | `internal/api/mcp_backend_contract_test.go` | Recorre API key → router → handler → servicio → storage → JSON con una DB temporal |

## Guardrail de impacto y contrato

El guardrail vive en desarrollo, no en el runtime. El hook `pre-commit` cruza los archivos staged contra el manifiesto, valida toda spec nueva staged y exige que la spec en curso declare el impacto antes de permitir el commit:

```
git add
   │
   ├──► scripts/check-mcp-docs.sh --staged     (docs/mcp/ obligatorio)
   └──► scripts/check-mcp-impact.sh --staged --gate
            ├── docs/specs/SPEC-* (nueva) ───► sección + Validación subagente
            ├── docs/mcp/tools/impact.json ──► tools afectadas
            ├── docs/specs/SPEC-* (en curso) ► tools nombradas + Validación
            ├── go test ./internal/mcp/... ──► TestToolContracts vs contracts.json
            └── go test ./internal/api ... ──► TestMCPBackendContract (cadena real)
```

`TestToolContracts` compara `tools/list` con el golden: un cambio de nombre, descripción, schema o anotación falla hasta regenerar `contracts.json` con `-update-contracts` y declararlo en la spec. `TestMCPBackendContract` recorre la cadena API key → router → handler → servicio → storage → JSON con una DB temporal y detecta drift de rutas, tags JSON, auth o `source=mcp`. El detalle operativo está en [tools/impact.md](tools/impact.md) y [development.md](development.md).

## Dos transportes

| Aspecto | stdio | HTTP |
|---|---|---|
| Activación | Default | `P40LA_MCP_TRANSPORT=http` |
| Entrada | stdin/stdout | `POST /mcp` (Streamable HTTP) |
| Auth del cliente | Confianza del proceso local | OAuth 2.1 + bearer token |
| Exposición | Ninguna | HTTPS 443 vía Funnel |
| Escrituras | `P40LA_MCP_ALLOW_WRITES=true` | `ALLOW_WRITES=true` **y** `REMOTE_WRITES=true` |
| Auditoría | Historial del backend | `audit.jsonl` + historial del backend |
| Estado en disco | Ninguno | `oauth.json` y `audit.jsonl` en `/data` |

## Listeners

El modo HTTP levanta dos servidores `net/http` en el mismo proceso:

- **Público** (`P40LA_MCP_LISTEN`, default `127.0.0.1:8090`): expone `/mcp` y los endpoints OAuth. Se publica con Funnel y exige bearer token en `/mcp`.
- **Administrativo** (`P40LA_MCP_ADMIN_LISTEN`, default `127.0.0.1:8091`): expone `GET /healthz` y `POST /admin/revoke`. Solo acepta peticiones cuyo `RemoteAddr` sea loopback y nunca se publica.

`config.Load` rechaza direcciones que no sean loopback, puertos repetidos y una URL pública con path, query o fragment. El binario no arranca si la configuración no es segura.

## Flujo OAuth

```
Cliente → discovery (.well-known) → DCR (/register)
        → /authorize (PKCE S256 + contraseña)
        → redirect con code → /token (code + verifier)
        → access token 1 h + refresh 30 días
        → POST /mcp con Authorization: Bearer
```

- Un solo usuario; contraseña bcrypt en `P40LA_MCP_OAUTH_PASSWORD_HASH` y `subject` opaco (`owner`).
- Clientes públicos (`token_endpoint_auth_method=none`); DCR solo acepta el callback de Claude y loopback `/callback`.
- Tokens opacos hasheados en `oauth.json` (`0600`), refresh rotativo y códigos de un solo uso.
- El detalle está en [oauth.md](oauth.md).

## Estado persistente

| Archivo | Contenido | Permisos |
|---|---|---|
| `oauth.json` | Clientes, hashes de códigos y hashes de tokens | `0600` |
| `audit.jsonl` | Llamadas a tools (`ts`, `request_id`, `subject`, `tool`, `result`, `duration_ms`) | `0600` |

Ambos viven en `P40LA_MCP_STATE_DIR` (default `/data`). La escritura es atómica y el directorio se crea con `0700`. No se guardan argumentos, resultados, montos ni PII.

## Autenticación

Hay **dos credenciales distintas** y no son intercambiables:

1. **OAuth 2.1** protege la entrada del MCP remoto. La usa el cliente (claude.ai) y la valida `internal/mcp/server/oauth`.
2. **API key global** protege la salida del MCP hacia la API de p40la-ihost. La usa el binario en ambos transportes.

La API key:

- Es única, global y tiene todos los permisos.
- Se guarda hasheada (SHA-256) en `system_settings`, junto con prefijo y fechas.
- Se envía como `Authorization: Bearer <key>` o `X-API-Key: <key>`.
- `SessionOrAPIKeyMiddleware` acepta la key o una sesión normal. La sesión sigue siendo necesaria para administrar la key.
- Las peticiones con key se auditan como `source=mcp`; no pueden simular `dashboard`.
- No se usan usuario/contraseña ni cookies del dashboard en el MCP.

Un bearer token OAuth **no** sirve contra la API del backend, y la API key **no** sirve contra `/mcp`.


## Por qué el MCP no accede a SQLite

- La DB del iHost usa WAL y es producción: un proceso externo escribiendo el archivo es riesgoso.
- Saltaría validaciones de negocio, el historial de facturas y la transacción de presupuesto al pagar.
- La API es el contrato estable; el MCP queda desacoplado del esquema.

## Lógica masiva en el backend

La expansión de rangos, la resolución de montos, la detección de duplicados y la decisión de qué pagar viven en `BillService`:

- `GenerateForService`: genera períodos inclusivos, omite existentes y reporta slots soft-deleted.
- `PayForService`: aplica un selector único, omite pagadas y períodos futuros.

El MCP solo traduce la tool a una llamada HTTP y presenta el resumen.

## Idempotencia

- `bills` tiene `UNIQUE(service_id, year, month)`, incluido el borrado lógico.
- La generación consulta `FindByServicePeriodIncludingDeleted` antes de crear.
- Repetir una generación devuelve `skipped` con motivo `exists_pending`, `exists_paid` o `deleted_slot`.
- Repetir un pago devuelve `skipped` con motivo `already_paid`.
- En autos, `autos_update` es idempotente porque `PUT` reemplaza el registro. Las altas de auto, mantenimiento e inspección crean un registro nuevo en cada llamada; el alta de seguro corta con `already_exists` si el auto ya tiene ese `service_id`.
- No hay reintentos automáticos de escrituras.

## Auditoría

Las altas y pagos hechos por el MCP registran `bill_history.source = mcp`. La UI sigue registrando `dashboard` y los webhooks `webhook`.

La migración `0060_bill_history_source_mcp` habilita el nuevo valor en el `CHECK` de `source`; es reversible y no crea tablas nuevas. La key global usa `system_settings`, sin migración adicional.

En modo HTTP se agrega una segunda capa local: `audit.jsonl` registra cada `tools/call` con `ts`, `request_id`, `subject`, `tool`, `result` y `duration_ms`. No incluye argumentos, resultados, montos ni tokens. El archivo rota por tamaño (`P40LA_MCP_AUDIT_MAX_MB`, default 5 MB) y conserva `audit.jsonl` más tres rotaciones.

## Escrituras

| Modo | Compuerta |
|---|---|
| stdio | `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

`Config.WritesEnabled()` combina ambas condiciones. Si el gate remoto está apagado, la tool devuelve `writes_disabled` y no llama a la API. El backend valida además el origen y registra `source=mcp`.

## Límites

- Máximo 36 períodos por generación.
- No se generan meses futuros.
- Las llamadas al backend son secuenciales.
- El binario MCP no se ejecuta dentro del proceso del servidor ni del contenedor por defecto.
- En modo HTTP: body de 1 MiB, 60 req/min por token, access token de 1 h, refresh de 30 días, código de 5 minutos y auditoría con rotación.
