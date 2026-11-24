---
title: "MCP de facturas: servidor de tools y generación/pago masivo por servicio"
id: "SPEC-145"
status: "pending_release"
author: "opencode"
created: "2026-11-23"
updated: "2026-11-23"
github_issue: 148
---

# MCP de facturas: servidor de tools y generación/pago masivo por servicio

**ID**: SPEC-145  
**Estado**: pending_release  
**Autor**: opencode  
**Creado**: 2026-11-23  
**Actualizado**: 2026-11-23

---

## 1. Resumen Ejecutivo

Hoy las facturas de un servicio se crean y se pagan de a una desde la UI. No existe una interfaz máquina-a-máquina para que un agente haga operaciones repetitivas: por ejemplo, generar las facturas de un servicio desde enero hasta la fecha (11 facturas en noviembre de 2026) o pagar solo la de enero, un rango de meses o todas las pendientes hasta hoy. Hacerlo a mano es lento y propenso a errores, y no deja un registro uniforme de qué hizo el agente.

Esta spec define un **servidor MCP** (Model Context Protocol) propio del proyecto, en Go, que expone tools para operar el sistema a través de su **API HTTP existente**. No accede a SQLite ni al volumen de datos: se autentica con la sesión normal y llama endpoints del servidor. Las operaciones masivas (`generar` y `pagar`) se implementan como endpoints de negocio en el backend —reutilizando `BillService`, el historial de facturas y las validaciones existentes— para que la lógica viva en la capa de servicios y el MCP sea un adaptador delgado. La primera entrega incluye tres tools: `bills_list`, `bills_generate` y `bills_pay`.

El diseño prioriza la seguridad y la idempotencia: las escrituras están deshabilitadas por defecto (`P40LA_MCP_ALLOW_WRITES`), `dry_run` viene activado por defecto en las tools que escriben, y toda generación verifica el período antes de crear para no duplicar facturas. El servidor MCP es un binario separado que no se carga dentro del proceso del iHost, por lo que no aumenta el consumo de RAM del add-on; las llamadas al backend son secuenciales y acotadas (máximo 36 períodos por operación).

La documentación vive en `docs/mcp/` en Markdown plano, con frontmatter y estructura compatibles con Docusaurus, para publicarla como sitio web más adelante sin reescribir el contenido. Esta spec **no implementa** el sitio: prepara los archivos para que un futuro proyecto de documentación los consuma.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`).

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-STACK` | Go + `net/http` para el cliente y `log/slog`; el MCP usa el SDK oficial de Go de MCP. Sin Node.js en runtime. | `CA-ARCH-STACK` |
| `ARCH-FOLDERS` | Agrega `cmd/mcp-server/`, `internal/mcp/` (adaptador), `internal/services/bill_bulk.go`, `internal/api/bill_bulk_handlers.go` y `docs/mcp/`. | `CA-ARCH-FOLDERS` |
| `ARCH-LAYERS` | La lógica masiva vive en `BillService`; el handler solo parsea; el MCP no contiene SQL ni reglas de negocio, solo llama la API. | `CA-ARCH-LAYERS` |
| `ARCH-MODULES` | N/A justificado: el MCP es un adaptador/cliente externo, no un módulo de dominio con storage. Las operaciones masivas extienden el servicio legacy de facturas, que la regla no obliga a migrar. | — |
| `ARCH-GO` | Código en inglés, `gofmt`, errores explícitos, configuración por env y `slog` a stderr (stdout reservado al protocolo MCP). | `CA-ARCH-GO` |
| `ARCH-IHOST` | El binario MCP no se ejecuta dentro del server ni del contenedor por defecto; llamadas secuenciales, máximo 36 períodos, sin migraciones ni procesos extra en el iHost. | `CA-ARCH-IHOST` |
| `ARCH-CONFIG` | N/A justificado: no se modifican `opencode.json` ni las instrucciones del proyecto; solo se documentan ejemplos de configuración de clientes MCP. | — |
| `UI-SELECT`, `UI-SELECT-ADD`, `UI-NUMBER-INPUT`, `UI-PHONE-INPUT`, `UI-INPUT-TOKENS`, `UI-DARKMODE`, `UI-TOGGLE`, `UI-LIST-PATTERN`, `UI-CREATE-MENU`, `UI-EDIT-LABEL`, `UI-I18N`, `UI-STYLE`, `UI-GUARDRAIL` | N/A justificado: la spec no toca frontend, formularios, listados, i18n ni componentes visuales. | — |
| `UX-PREREQ` | El servicio debe existir y estar activo; se valida en backend y el error se propaga a la tool con código accionable. | `CA-UX-PREREQ` |
| `UX-FEEDBACK` | Toda tool devuelve un resumen por ítem (`creada`, `omitida`, `fallida`) con motivo, además de totales. El `dry_run` muestra el resultado antes de escribir. | `CA-UX-FEEDBACK` |
| `UX-BACK`, `UX-MODAL`, `UX-MODAL-CONFIRM`, `UX-DETAIL-ACTIONS`, `UX-DETAIL-READONLY`, `UX-EMPTY`, `UX-LOADING`, `UX-FORMS` | N/A justificado: no hay pantallas, rutas ni modales nuevos. | — |
| `DATA-SQLITE` | N/A justificado: no cambia la configuración de SQLite; las operaciones masivas usan `BillStorage` existente. | — |
| `DATA-MIGRATIONS` | Una migración reversible (`0060_bill_history_source_mcp`) habilita `source=mcp` en `bill_history`; la key global usa `system_settings` y no requiere tabla nueva. | `CA-DATA-MIGRATIONS` |
| `DATA-NO-DROP` | El MCP nunca abre ni reemplaza `data/app.db`; las pruebas usan `/tmp/test-app.db`. | `CA-DATA-NO-DROP` |
| `DATA-PRODCOPY` | La validación con datos reales se hace sobre una copia `.backup` en `/tmp`; nunca contra la DB local ni escribiendo producción durante pruebas. | `CA-DATA-PRODCOPY` |
| `DATA-PROD-WRITE` | Las escrituras reales se hacen por la API, no con SQL directo. Antes del primer backfill en el iHost se exige copia de respaldo y confirmación explícita del usuario. | `CA-DATA-PROD-WRITE` |
| `DATA-PROTECTED-SCHEMA` | N/A justificado: no se tocan `debts` ni `debt_bills`. | — |
| `SEC-NO-PII` | Ejemplos con datos sintéticos; logs solo con conteos e IDs, nunca montos, emails, cookies ni contraseñas. | `CA-SEC-NO-PII` |
| `SEC-SYNTHETIC` | `demo@example.com`, `demo-password`, `demo-ref-001`, montos de ejemplo; el `service_id: 12` es un identificador técnico del ejemplo del usuario, no PII. | `CA-SEC-SYNTHETIC` |
| `SEC-PRODCOPY` | Copias de producción solo en `/tmp` y se borran al terminar. | `CA-SEC-PRODCOPY` |
| `SEC-COMMIT` | Revisión manual de `git diff --cached` antes de cada commit. | `CA-SEC-COMMIT` |
| `SEC-HOOKS` | Hooks instalados; prohibido `--no-verify` sin autorización. | `CA-SEC-HOOKS` |
| `SEC-INCIDENT` | Protocolo documentado en `docs/mcp/security.md`: rotar, purgar historia y avisar. | `CA-SEC-INCIDENT` |
| `SEC-LOGS` | Logs del MCP a stderr, sin payloads sensibles; errores con códigos, no volcados. | `CA-SEC-LOGS` |
| `TEST-LOCAL` | `go build ./...` y `go test ./...` obligatorios; integración con API falsa y server local sobre `/tmp`. | `CA-TEST-LOCAL` |
| `TEST-SERVER` | Antes de avisar al usuario se ejecuta `./scripts/check-server.sh`. | `CA-TEST-SERVER` |
| `TEST-KILLALL` | Uso exclusivo de `killall`, nunca `pkill`. | `CA-TEST-KILLALL` |
| `TEST-TESTUSER` | La integración autenticada usa `./scripts/create-test-user.sh` y `/tmp/cookies.txt`. | `CA-TEST-TESTUSER` |
| `TEST-USER-FLOW` | Se pregunta 🤖 antes de correr el server y no se cierra la spec sin confirmación de pruebas. | `CA-TEST-USER-FLOW` |
| `TEST-ROBOT` | Las preguntas de flujo usan 🤖. | `CA-TEST-ROBOT` |
| `TEST-CHECKLIST` | Se completa la checklist completa antes de `pending_release`. | `CA-TEST-CHECKLIST` |
| `DELIV-WORKTREE` | La implementación se hace en `../p40la-ihost-spec-145` (`feature/SPEC-145`). | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge, push y deploy requieren confirmación explícita; prohibido `--force`. | `CA-DELIV-CONFIRM` |
| `DELIV-COMMIT` | Commits con formato `SPEC-145: descripción corta`. | `CA-DELIV-COMMIT` |
| `DELIV-RELEASE` | Se sigue el flujo completo de release (merge, tracker, label, issue, limpieza). | `CA-DELIV-RELEASE` |
| `DELIV-CHECKRELEASE` | Se ejecuta `/spec-check-release` antes de declarar terminada la spec. | `CA-DELIV-CHECKRELEASE` |
| `DELIV-HOOKS` | Hooks instalados en el worktree; sin `--no-verify`; incluye `check-mcp-docs.sh`. | `CA-DELIV-HOOKS` |
| `DELIV-MCP-DOCS` | Cada cambio de la superficie MCP incluye `docs/mcp/` en el mismo commit; el hook lo bloquea si falta. | `CA-DELIV-MCP-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La actualización de `docs/mcp/` se delega a un subagente en paralelo y el principal la revisa. | `CA-DELIV-MCP-DOCS-SUBAGENT` |
| `DELIV-DOCKER` | N/A justificado: no se crea ni publica una imagen nueva. El binario MCP se compila localmente con `go build`; si en el futuro se publica una imagen, deberá ser multi-arch. | — |
| `DELIV-LABELS` | Issue con `spec/draft`; transiciones con labels válidos y cierre solo en release. | `CA-DELIV-LABELS` |
| `DELIV-DEPLOY` | El deploy al iHost lo aprueba el usuario y se verifica `/health` después. | `CA-DELIV-DEPLOY` |
| `DELIV-ESCALATION` | Si la spec queda bloqueada más de 3 días, se documenta y se escala. | `CA-DELIV-ESCALATION` |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001 — Servidor MCP propio**: binario Go `cmd/mcp-server` que habla MCP por **stdio** (JSON-RPC 2.0) y usa el SDK oficial de Go. No escribe en stdout nada que no sea protocolo; logs a stderr.
2. **REQ-002 — Registro extensible de tools**: contrato único (`Name`, `Description`, `InputSchema`, `Handler`) y un punto de registro (`internal/mcp/tools/all/register.go`). Agregar una tool nueva debe requerir un archivo de tool, su registro y su documentación, sin tocar el transporte.
3. **REQ-003 — Tool `bills_list`**: lista facturas de un servicio con filtros `from`, `to`, `status` (`pending`, `paid`, `all`). Solo lectura.
4. **REQ-004 — Tool `bills_generate`**: genera facturas de un servicio para un rango inclusivo `from`/`to` en formato `YYYY-MM`, con `to` por defecto el mes actual. Para servicios mensuales genera una por mes; para anuales genera la de enero (`month=0`). Omite períodos que ya tienen factura activa y reporta los que tienen una factura soft-deleted. Máximo 36 períodos. `dry_run` por defecto `true` en la capa MCP.
5. **REQ-005 — Tool `bills_pay`**: paga facturas de un servicio con **exactamente un** selector: `bill_ids`, `month` (`YYYY-MM`), `from`+`to` (`YYYY-MM`) o `all_pending_up_to` (`YYYY-MM`). `paid_at` por defecto hoy; no acepta fechas futuras. Omite las ya pagadas. `dry_run` por defecto `true`.
6. **REQ-006 — Autenticación por API key global**: el MCP se autentica con una única API key de sistema (`Authorization: Bearer ...`), nunca con usuario/contraseña ni cookie. La key es global, tiene todos los permisos, se guarda hasheada en `system_settings` y se puede rotar o revocar.
7. **REQ-007 — Compuerta de escritura**: `bills_generate` y `bills_pay` solo escriben si `P40LA_MCP_ALLOW_WRITES=true`. Sin la compuerta, `dry_run=true` funciona y `dry_run=false` devuelve `writes_disabled`.
8. **REQ-008 — Endpoints masivos de backend**: `POST /api/services/{id}/bills/generate` y `POST /api/services/{id}/bills/pay`, con lógica en `BillService`, respuesta de resumen por ítem y código HTTP 200 aun con fallos parciales; las precondiciones inválidas devuelven 4xx.
9. **REQ-009 — Auditoría**: las altas y pagos hechos por las tools registran el origen `mcp` en `bill_history` (nuevo `BillSourceMCP`), sin alterar el origen `dashboard` de la UI. El origen se toma de un header interno `X-P40LA-Source` con allowlist. La migración `0060_bill_history_source_mcp` amplía el `CHECK` de `source` y es reversible.
10. **REQ-010 — Documentación**: `docs/mcp/` con índice, arquitectura, configuración, seguridad, operación, desarrollo de nuevas tools, catálogo y referencia de `bills_*`, más ejemplos de configuración para OpenCode y Claude Desktop. Markdown plano con frontmatter Docusaurus.
11. **REQ-011 — Idempotencia**: repetir `bills_generate` sobre el mismo rango no duplica facturas; repetir `bills_pay` sobre facturas pagadas devuelve `skipped` con motivo `already_paid`.
12. **REQ-012 — Errores accionables**: errores con código estable (`invalid_range`, `service_not_found`, `service_inactive`, `amount_unresolved`, `deleted_slot`, `writes_disabled`, `auth_failed`, `upstream_error`) y mensaje en español claro, sin stack traces ni datos sensibles.
13. **REQ-013 — Gestión de la API key global**: hash SHA-256 y prefijo en `system_settings`, con `created_at` y `last_used_at`. Endpoints autenticados por sesión para consultar estado, regenerar (rotación) y revocar; la key completa se muestra una sola vez. Script `scripts/create-api-key.sh` para provisionar. Una sola key global con todos los permisos; sin scopes ni tabla de keys.

### 3.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-014 — Fechas de emisión y vencimiento**: extender `POST /api/bills` y los endpoints masivos para aceptar `issue_date` y `due_date` (`YYYY-MM-DD`), calculando el vencimiento con `billing_day` del servicio y el último día del mes.
2. **REQ-015 — Montos por período**: permitir un mapa opcional `amounts: {"2026-01": 1000.00, ...}` para backfills con montos históricos distintos.
3. **REQ-016 — `invoice_number` opcional**: aceptar una plantilla simple (ej. `F-{YYYY}{MM}-{service_id}`) con validación de longitud, sin duplicados.
4. **REQ-017 — Script de build**: `scripts/build-mcp.sh` que compile `bin/p40la-mcp` para `linux/amd64`, `linux/arm64` y `darwin/arm64`, con versión embebida.
5. **REQ-018 — Preflight de salud**: la tool `bills_list` o un recurso MCP `health` consulta `GET /health` para diagnosticar conectividad antes de operar.

### 3.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-019 — Transporte HTTP**: variante Streamable HTTP del MCP para clientes remotos, con token dedicado; no reemplaza stdio en la primera entrega.
2. **REQ-020 — Nuevas tools de dominio**: deudas, presupuestos y automatizaciones usando el mismo registro, sin cambiar el transporte.
3. **REQ-021 — Recursos MCP**: exponer recursos de solo lectura (ej. resumen de un servicio) además de tools.

### 3.4 Requerimientos No Funcionales

- **Rendimiento**: una generación de 11 facturas debe completarse en menos de 5 segundos en LAN; máximo 36 períodos por llamada; llamadas secuenciales al backend (sin paralelismo).
- **Memoria**: el binario MCP no se ejecuta dentro del add-on; objetivo de RSS menor a 30 MB. El server del iHost no aumenta su consumo por esta spec.
- **Seguridad**: API key solo por entorno, nunca por argumentos ni logs; hasheada en la DB y mostrada una sola vez; revocable; escrituras deshabilitadas por defecto; sin PII en documentación ni ejemplos.
- **Almacenamiento**: una migración acotada y reversible (`0058` para `source=mcp`); la key global vive hasheada en `system_settings`. Sin archivos de estado ni caché persistente.
- **Disponibilidad**: el MCP no es un servicio residente; se inicia por el cliente MCP y termina al cerrarse la sesión. Timeout HTTP configurable, 15 s por defecto.
- **iHost**: no se agrega Node.js al runtime; no se agregan procesos permanentes; el backend solo suma dos rutas HTTP y lógica ya existente.
- **Compatibilidad**: la API nueva es aditiva; la UI y los webhooks actuales no cambian de comportamiento.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado

- **MCP**: protocolo JSON-RPC 2.0 para exponer tools y recursos a clientes LLM. La revisión estable de referencia es la disponible al implementar (al momento de esta spec, `2025-06-18`). El SDK oficial de Go (`github.com/modelcontextprotocol/go-sdk`) implementa inicialización, `tools/list`, `tools/call`, esquemas y transporte stdio.
- **API existente** (`internal/api/routes.go`):
  - `POST /api/login` con `{email, password, remember}` y cookie `session`.
  - `GET /api/services/{service_id}/bills` (facturas activas del servicio, ordenadas por período descendente).
  - `POST /api/bills` (`service_id`, `year`, `month`, `amount`, `invoice_number`, `status`, `drive_url`).
  - `POST /api/bills/{id}/pay` (`paid_at`, `drive_url`, `payment_reference`).
- **Modelo de datos**: `bills` tiene `UNIQUE(service_id, year, month)` y borrado lógico (`deleted_at`). Una factura soft-deleted **sigue ocupando** la clave única; `FindByServicePeriodIncludingDeleted` existe para detectarlo (precedente SPEC-071). Por eso la generación no puede basarse solo en el listado activo.
- **Historial**: `bill_history` guarda `action` y `source` (`dashboard`, `webhook`). No existe un origen para agentes/API; se agrega `mcp`.
- **iHost**: ARM64, recursos limitados, runtime distroless. El MCP no necesita correr allí: el cliente MCP lo lanza en la máquina del usuario y este habla con `http://ihost.local:8088`.
- **Docusaurus**: consume Markdown con frontmatter (`title`, `sidebar_position`, `description`) y carpetas con `_category_.json`; no requiere un build en este repo.

### 4.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| A. MCP con acceso directo a SQLite | Menos capas; una sola operación | Rompe ARCH-LAYERS; riesgo de corrupción WAL; requiere acceso al volumen; saltea validaciones e historial | ❌ Rechazada |
| B. MCP que orquesta N llamadas `POST /api/bills` | No requiere cambios de backend | Reglas duplicadas en el cliente; fallos parciales difíciles de reportar; no puede detectar slots soft-deleted desde el listado activo; N round-trips | ❌ Rechazada |
| C. Endpoints masivos en backend + MCP delgado | Lógica única en `BillService`; idempotencia y auditoría centralizadas; una llamada; reutilizable por la UI futura | Aumenta el alcance del backend en dos rutas y métodos de servicio | ✅ Seleccionada |
| D. CLI en vez de MCP | Simple | No es controlable por agentes MCP; el usuario pidió MCP | ❌ Rechazada |

### 4.3 Decisiones arquitectónicas (ADRs)

**ADR-001 — El MCP es un cliente de la API, no un acceso a la base**
- **Contexto**: el iHost usa SQLite en WAL y la DB local del usuario es producción. Un proceso externo escribiendo el archivo es riesgoso y saltea reglas de negocio.
- **Decisión**: todo pasa por la API HTTP autenticada; el MCP no importa `internal/storage` ni abre archivos de datos.
- **Consecuencias**: se necesita login y los endpoints masivos; a cambio, se conservan validaciones, historial, presupuesto automático al pagar y auditoría.

**ADR-002 — Operaciones masivas en `BillService`, no en el MCP**
- **Contexto**: expandir rangos, resolver montos, detectar duplicados y decidir qué se omite son reglas de negocio.
- **Decisión**: `internal/services/bill_bulk.go` implementa `GenerateForService` y `PayForService`; los handlers solo parsean y el MCP solo traduce tools a llamadas HTTP.
- **Consecuencias**: una sola fuente de verdad; la UI puede reutilizar los endpoints; el MCP queda chico y testeable con una API falsa.

**ADR-003 — SDK oficial de MCP en Go**
- **Contexto**: implementar JSON-RPC, inicialización, esquemas y cancelación a mano es propenso a drift de protocolo.
- **Decisión**: usar el SDK oficial, fijado a una versión exacta en `go.mod`, y encapsularlo en `internal/mcp/registry` para que las tools dependan de un contrato propio.
- **Consecuencias**: una dependencia nueva y algo de tamaño de binario; se gana cumplimiento de protocolo y mantenimiento. Si el SDK resultara incompatible, el contrato propio permite reemplazar el transporte sin tocar las tools.

**ADR-004 — stdio primero, HTTP después**
- **Contexto**: los clientes MCP locales (OpenCode, Claude Desktop) lanzan el servidor por stdio; el iHost no debe exponer otro puerto.
- **Decisión**: P0 solo stdio; Streamable HTTP queda en P2 con token dedicado.
- **Consecuencias**: integración local simple y segura; acceso remoto requerirá un proxy o la variante HTTP futura.

**ADR-005 — Escrituras apagadas por defecto + `dry_run` por defecto**
- **Contexto**: generar o pagar facturas afecta datos financieros reales y puede ejecutarse por error desde un agente.
- **Decisión**: `P40LA_MCP_ALLOW_WRITES` debe ser `true` para escribir; `bills_generate` y `bills_pay` traen `dry_run=true` por defecto y las descripciones instruyen mostrar el resumen y pedir confirmación.
- **Consecuencias**: un paso extra para operar; se elimina la escritura accidental y se conserva un preview auditable.

**ADR-006 — Documentación en `docs/mcp/` lista para Docusaurus**
- **Contexto**: se quiere publicar documentación web sin acoplarla a un sitio ahora.
- **Decisión**: Markdown plano, frontmatter estándar, `_category_.json`, enlaces relativos y ejemplos sintéticos. No se agrega build de Docusaurus en esta spec.
- **Consecuencias**: publicar luego es montar la carpeta como `docs` o `docs/mcp`; no hay dependencias de Node en el repo por documentación.

**ADR-007 — API key global en lugar de usuario/contraseña**
- **Contexto**: el MCP no debe manejar la contraseña personal del usuario. La única autenticación existente es la sesión por cookie y la API key de webhooks es global pero está limitada a `/webhooks/{uuid}`.
- **Decisión**: agregar una única API key global del sistema, hasheada y revocable, con todos los permisos. Se guarda en `system_settings` (hash, prefijo, fechas) y se gestiona con endpoints autenticados por sesión. El MCP usa solo `P40LA_API_KEY` y `Authorization: Bearer`.
- **Consecuencias**: la key equivale a un usuario con acceso completo: si se filtra, hay que rotarla de inmediato. Se gana independencia de la contraseña, rotación y trazabilidad (`last_used_at`, auditoría `mcp`). No hay scopes granulares en esta entrega.

---

## 5. Diseño Técnico

### 5.1 Diagrama de arquitectura

```
┌────────────────────┐        stdio (JSON-RPC 2.0)        ┌─────────────────────┐
│ Cliente MCP        │ ◄────────────────────────────────► │ p40la-mcp           │
│ (OpenCode/Claude)  │                                    │ cmd/mcp-server      │
└────────────────────┘                                    │ internal/mcp/*      │
                                                          └──────────┬──────────┘
                                                                     │ HTTP + Bearer token
                                                                     ▼
                                                          ┌─────────────────────┐
                                                          │ p40la-ihost server  │
                                                          │ API key middleware  │
                                                          │ /api/services/...   │
                                                          │ /api/bills/...      │
                                                          └──────────┬──────────┘
                                                                     │
                                                                     ▼
                                                          ┌─────────────────────┐
                                                          │ BillService         │
                                                          │ BillStorage         │
                                                          │ SQLite (bills,      │
                                                          │ bill_history)       │
                                                          └─────────────────────┘
```

### 5.2 Componentes

#### 5.2.1 `cmd/mcp-server/`
- **Responsabilidad**: entrypoint del binario MCP. Lee configuración, crea el cliente HTTP, registra las tools y ejecuta el transporte stdio. Sin lógica de negocio.
- **Interfaz**: proceso que habla MCP por stdin/stdout.
- **Dependencias**: `internal/mcp/config`, `internal/mcp/api`, `internal/mcp/tools/all`.
- **Ubicación**: `cmd/mcp-server/main.go`.

#### 5.2.2 `internal/mcp/config/`
- **Responsabilidad**: cargar y validar variables de entorno; nunca imprime secretos.
- **Variables**: `P40LA_BASE_URL` (default `http://ihost.local:8088`), `P40LA_API_KEY` (obligatoria, formato `p40la_...`), `P40LA_MCP_ALLOW_WRITES` (default `false`), `P40LA_TIMEOUT` (default `15s`), `P40LA_TIMEZONE` (default `America/Managua`).
- **Validación**: URL absoluta `http`/`https`; API key con prefijo y longitud válidos; timeout positivo y acotado (máx. 60 s).

#### 5.2.3 `internal/mcp/api/`
- **Responsabilidad**: cliente HTTP de la API de p40la. Envía `Authorization: Bearer <API key>` en cada llamada; no maneja login ni cookies.
- **Métodos**: `Health`, `GetService`, `ListBills`, `GenerateBills`, `PayBills`.
- **Errores**: `APIError{Status, Code, Message}` con el código estable devuelto por el backend; los errores de red se envuelven sin exponer payloads.
- **Ubicación**: `internal/mcp/api/client.go`, `internal/mcp/api/types.go`.

#### 5.2.4 `internal/mcp/registry/`
- **Responsabilidad**: montar conjuntos de tools sin que `cmd/mcp-server` conozca los paquetes de cada dominio.
- **Contrato**:

```go
type RegisterFunc func(*mcp.Server)

type Registry struct{ /* ... */ }

func (r *Registry) Add(fn RegisterFunc)
func (r *Registry) Mount(server *mcp.Server)
```

El catálogo y los schemas los produce el SDK al registrar cada tool con `mcp.AddTool`.

#### 5.2.5 `internal/mcp/tools/bills/`
- **Responsabilidad**: implementación de las tres tools de facturas y de la expansión de períodos.
- **Archivos**: `tools.go`, `results.go`, `list.go`, `generate.go`, `pay.go`, `period.go`, `tools_test.go`, `protocol_test.go`.
- **Registro**: `internal/mcp/tools/all/register.go` importa el paquete y registra las tools en un solo lugar.

#### 5.2.6 `internal/services/bill_bulk.go`
- **Responsabilidad**: lógica de generación y pago masivo. Reutiliza `BillService`, `BillStorage`, `FindByServicePeriodIncludingDeleted`, `PayBill` y el historial.
- **Métodos**:
  - `GenerateForService(ctx, serviceID int64, opts GenerateBillsOptions) (*GenerateBillsResult, error)`
  - `PayForService(ctx, serviceID int64, opts PayBillsOptions) (*PayBillsResult, error)`
- **Reglas**: rango inclusivo, máximo 36 períodos, no generar meses futuros, servicio activo, resolución de monto, detección de duplicados y slots borrados, pago solo de pendientes y no futuras.

#### 5.2.7 `internal/api/bill_bulk_handlers.go`
- **Responsabilidad**: parsear y validar requests, llamar al servicio y responder el resumen. Sin SQL ni reglas.
- **Rutas** (en `internal/api/routes.go`, con `authMiddleware`):
  - `POST /api/services/{id}/bills/generate`
  - `POST /api/services/{id}/bills/pay`

#### 5.2.8 `docs/mcp/`
- **Responsabilidad**: documentación operativa y de desarrollo, lista para Docusaurus.
- **Estructura**:

```
docs/mcp/
├── README.md              # índice, qué es y cómo empezar
├── _category_.json        # metadatos de carpeta para Docusaurus
├── api-key.md             # creación, rotación y revocación de la key global
├── architecture.md        # componentes, flujo y capas
├── configuration.md       # variables de entorno y clientes MCP
├── security.md            # API key, compuerta de escritura, incidentes
├── operations.md          # build, ejecución, diagnóstico, backup
├── development.md         # cómo agregar una tool nueva
├── tools/
│   ├── README.md          # catálogo de tools
│   └── bills.md           # referencia de bills_list/generate/pay
└── examples/
    ├── opencode.json      # configuración de ejemplo (sintética)
    └── claude-desktop.json
```

#### 5.2.9 API key global del sistema
- **Servicio**: `internal/services/api_key.go` genera `p40la_<32 bytes base64url>`, guarda el hash SHA-256 y el prefijo en `system_settings`, valida en tiempo constante y actualiza `last_used_at`.
- **Handlers**: `internal/api/api_key_handlers.go` con `GET /api/api-key` (estado), `POST /api/api-key/regenerate` (rotación) y `DELETE /api/api-key` (revocación), todos solo por sesión.
- **Middleware**: `SessionOrAPIKeyMiddleware` acepta sesión o `Authorization: Bearer`/`X-API-Key`; la key global tiene todos los permisos y accede a cualquier ruta autenticada.
- **Script**: `scripts/create-api-key.sh` inicia sesión una vez, rota/crea la key y la imprime.

### 5.3 Modelo de datos

Se reutilizan `bills`, `bill_history` y `system_settings`. Solo se agrega una migración:

- `0060_bill_history_source_mcp`: recrea `bill_history` para ampliar el `CHECK` de `source` con `mcp`.

```
Entidad: system_settings (existente, valores nuevos)
- mcp_api_key_hash       (SHA-256 hex; nunca se guarda la key completa)
- mcp_api_key_prefix     (primeros caracteres, para identificar la key)
- mcp_api_key_created_at
- mcp_api_key_last_used_at
```

```
Entidad: bills (existente)
- id, service_id, cycle_id
- year, month (0 = anual)
- amount, invoice_number, status (pending|paid)
- issue_date, due_date, paid_at, payment_reference
- deleted_at (borrado lógico; sigue ocupando UNIQUE(service_id, year, month))
- UNIQUE(service_id, year, month)

Entidad: bill_history (existente)
- bill_id, action, source, changes, created_at
- source: dashboard | webhook | mcp (nuevo valor)
```

Reglas de negocio del rango:

- Rango `from`/`to` inclusivo en `YYYY-MM`.
- Servicio mensual: un período por mes.
- Servicio anual: solo enero, con `month=0`.
- No se generan períodos posteriores al mes actual (zona `P40LA_TIMEZONE`).
- Máximo 36 períodos; `from <= to`.
- Monto: `amount` explícito > monto de la factura activa más reciente > `suggested_amount` del servicio. Si el resultado es `0` y no hay monto explícito, error `amount_unresolved`.
- Período con factura activa: `skipped` (`exists_pending` o `exists_paid`).
- Período con factura soft-deleted: `skipped` (`deleted_slot`) y se informa el `bill_id`; no se resucita automáticamente.

### 5.4 APIs / Contratos

#### Autenticación por API key

Todas las rutas del MCP aceptan:

```http
Authorization: Bearer p40la_<secreto>
```

o, como alternativa:

```http
X-API-Key: p40la_<secreto>
```

La key se hashea (SHA-256) y se compara en tiempo constante contra `mcp_api_key_hash`. Una key inexistente, revocada o incorrecta devuelve `401 auth_failed`. La key global tiene todos los permisos y accede a cualquier ruta autenticada. Las peticiones autenticadas por key se auditan con `source=mcp` y no pueden simular `dashboard`.

#### Endpoints de administración de la key global (solo sesión)

`GET /api/api-key` devuelve el estado sin secreto:

```json
{
  "configured": true,
  "prefix": "p40la_AbCdEfGh",
  "created_at": "2026-11-23T00:00:00Z",
  "last_used_at": "2026-11-23T00:00:00Z"
}
```

`POST /api/api-key/regenerate` genera una nueva key, invalida la anterior y devuelve el secreto **una sola vez**:

```json
{
  "api_key": "p40la_...",
  "prefix": "p40la_AbCdEfGh",
  "created_at": "2026-11-23T00:00:00Z"
}
```

`DELETE /api/api-key` revoca la key vigente; las peticiones con la key anterior pasan a `401`.

#### Endpoint: `POST /api/services/{id}/bills/generate`

**Request**:
```json
{
  "from": "2026-01",
  "to": "2026-11",
  "amount": 1000.00,
  "dry_run": false,
  "allow_inactive": false
}
```

- `from` obligatorio, `to` opcional (default mes actual), ambos `YYYY-MM`.
- `amount` opcional; si se omite se resuelve según 5.3.
- `dry_run` default `false` en la API; el MCP lo fuerza a `true` por defecto.
- Header opcional `X-P40LA-Source: mcp` para auditoría (allowlist `dashboard|mcp`).

**Response 200**:
```json
{
  "service_id": 12,
  "from": "2026-01",
  "to": "2026-11",
  "dry_run": false,
  "amount": 1000.00,
  "currency_code": "NIO",
  "created": 11,
  "skipped": 0,
  "failed": 0,
  "items": [
    {
      "year": 2026,
      "month": 1,
      "amount": 1000.00,
      "status": "created",
      "bill_id": 1001,
      "reason": ""
    }
  ]
}
```

**Response Error**:
```json
{
  "error": "invalid_range",
  "message": "El rango máximo es de 36 meses y no puede incluir meses futuros"
}
```

#### Endpoint: `POST /api/services/{id}/bills/pay`

**Request**:
```json
{
  "bill_ids": [1001],
  "amount": null,
  "paid_at": "2026-11-23",
  "payment_reference": "demo-ref-001",
  "drive_url": "",
  "dry_run": false
}
```

Selectores válidos (exactamente uno): `bill_ids`, `month`, `from`+`to`, `all_pending_up_to`.
```json
{ "month": "2026-01" }
{ "from": "2026-01", "to": "2026-11" }
{ "all_pending_up_to": "2026-11" }
```

**Response 200**:
```json
{
  "service_id": 12,
  "selector": "month",
  "paid_at": "2026-11-23",
  "dry_run": false,
  "paid": 1,
  "skipped": 0,
  "failed": 0,
  "total_amount": 1000.00,
  "currency_code": "NIO",
  "items": [
    {
      "bill_id": 1001,
      "year": 2026,
      "month": 1,
      "amount": 1000.00,
      "status": "paid",
      "reason": ""
    }
  ]
}
```

Reglas de pago:
- Solo facturas `pending` y activas.
- No se pagan períodos futuros; si el selector incluye uno, se omite con `future_period`.
- Factura ya pagada: `skipped` (`already_paid`), no error.
- `paid_at` default hoy; futura → 400 `invalid_paid_at`.
- `bill_ids` debe pertenecer al servicio y existir; IDs inválidos → 400 antes de escribir.
- El pago usa `BillService.PayBill`, por lo que conserva historial y transacción de presupuesto.

#### MCP: tools expuestas

| Tool | Tipo | Entrada | Salida |
|---|---|---|---|
| `bills_list` | lectura | `service_id`, `from?`, `to?`, `status?` | Lista de facturas + totales |
| `bills_generate` | escritura | `service_id`, `from`, `to?`, `amount?`, `dry_run?` | Resumen de generación |
| `bills_pay` | escritura | `service_id`, selector, `paid_at?`, `payment_reference?`, `drive_url?`, `dry_run?` | Resumen de pago |

Contrato de `bills_generate` (JSON Schema resumido):
```json
{
  "type": "object",
  "required": ["service_id", "from"],
  "properties": {
    "service_id": { "type": "integer", "minimum": 1 },
    "from": { "type": "string", "pattern": "^[0-9]{4}-(0[1-9]|1[0-2])$" },
    "to": { "type": "string", "pattern": "^[0-9]{4}-(0[1-9]|1[0-2])$" },
    "amount": { "type": "number", "minimum": 0 },
    "dry_run": { "type": "boolean", "default": true }
  }
}
```

Contrato de `bills_pay` (JSON Schema resumido):
```json
{
  "type": "object",
  "required": ["service_id"],
  "properties": {
    "service_id": { "type": "integer", "minimum": 1 },
    "bill_ids": { "type": "array", "items": { "type": "integer" } },
    "month": { "type": "string", "pattern": "^[0-9]{4}-(0[1-9]|1[0-2])$" },
    "from": { "type": "string", "pattern": "^[0-9]{4}-(0[1-9]|1[0-2])$" },
    "to": { "type": "string", "pattern": "^[0-9]{4}-(0[1-9]|1[0-2])$" },
    "all_pending_up_to": { "type": "string", "pattern": "^[0-9]{4}-(0[1-9]|1[0-2])$" },
    "paid_at": { "type": "string", "format": "date" },
    "payment_reference": { "type": "string", "maxLength": 120 },
    "drive_url": { "type": "string", "format": "uri" },
    "dry_run": { "type": "boolean", "default": true }
  }
}
```

Errores MCP (contenido estructurado + `isError`):
```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas. Definí P40LA_MCP_ALLOW_WRITES=true para ejecutar cambios.",
  "hint": "Podés repetir con dry_run=true para ver el resultado sin escribir."
}
```

### 5.5 Registro y ciclo de una tool

1. El cliente MCP inicia el binario y envía `initialize`.
2. El SDK responde con capacidades y luego `tools/list` devuelve las tools registradas.
3. El modelo elige una tool; el cliente envía `tools/call` con `name` y `arguments`.
4. El handler valida la entrada, llama al cliente HTTP y devuelve contenido estructurado.
5. Si la tool escribe y `P40LA_MCP_ALLOW_WRITES` no está activo, devuelve `writes_disabled` sin tocar la API.
6. Si `dry_run` es `true`, la API calcula el resultado y no persiste nada.

Para agregar una tool nueva: crear el archivo en `internal/mcp/tools/<dominio>/`, implementar el contrato, registrarla en `internal/mcp/tools/all/register.go`, agregar tests con API falsa y documentarla en `docs/mcp/tools/<dominio>.md`. No se toca el transporte.

### 5.6 Dependencias

- **Internas**: `internal/services`, `internal/storage`, `internal/models`, `internal/api/routes.go`, `internal/mcp/*`.
- **Externas**:
  - SDK oficial `github.com/modelcontextprotocol/go-sdk` fijado en `v1.8.0` (justificación en ADR-003).
  - Solo stdlib para HTTP, JSON, cookies y logging.
- **Sin dependencias nuevas en el frontend** y sin Node.js en runtime.

### 5.7 Documentación y publicación web

Cada archivo de `docs/mcp/` lleva frontmatter compatible con Docusaurus:

```markdown
---
title: "Tools de facturas"
sidebar_position: 2
description: "Referencia de bills_list, bills_generate y bills_pay"
---
```

`docs/mcp/_category_.json`:
```json
{
  "label": "MCP",
  "position": 8,
  "link": { "type": "generated-index", "description": "Servidor MCP y tools de p40la-ihost" }
}
```

Contenido mínimo por archivo:

| Archivo | Contenido |
|---|---|
| `README.md` | Qué es MCP, alcance, requisitos, inicio rápido, enlaces |
| `api-key.md` | Key global: creación, rotación, revocación, headers y seguridad |
| `architecture.md` | Diagrama, capas, por qué no hay acceso directo a SQLite, idempotencia |
| `configuration.md` | Variables de entorno, ejemplo OpenCode y Claude Desktop, timeouts |
| `security.md` | Credenciales, compuerta de escritura, `dry_run`, logs, incidentes |
| `operations.md` | Build, instalación, verificación `/health`, diagnóstico y backup previo |
| `development.md` | Contrato de tool, registro, tests, documentación y checklist |
| `tools/README.md` | Catálogo y convención de nombres |
| `tools/bills.md` | Schemas, ejemplos sintéticos, errores y flujo de confirmación |

La documentación no se publica en esta spec; queda lista para que un proyecto Docusaurus use `docs/mcp` como carpeta de docs.

---

## 6. Criterios de Aceptación

> IDs canónicos por dominio (`CA-<PREFIJO>-<NOMBRE>`) para auditoría con grep.

### 6.1 Funcionales

- [ ] **CA-MCP-001**: `initialize` y `tools/list` exponen exactamente `bills_list`, `bills_generate` y `bills_pay` con descripciones y schemas válidos.
- [ ] **CA-MCP-002**: para un servicio mensual de ejemplo (`service_id: 12`), `bills_generate` con `from=2026-01` y `to=2026-11` genera **11** facturas (enero a noviembre inclusive) y devuelve `created=11`.
- [ ] **CA-MCP-003**: repetir la misma generación devuelve `created=0` y `skipped=11`; no se duplica ninguna factura.
- [ ] **CA-MCP-004**: con `dry_run=true`, el resumen muestra las 11 facturas planificadas y la base no cambia.
- [ ] **CA-MCP-005**: sin `P40LA_MCP_ALLOW_WRITES=true`, `dry_run=false` devuelve `writes_disabled`; con la variable activa, escribe.
- [ ] **CA-MCP-006**: `bills_pay` con `month=2026-01` paga solo la factura de enero y deja el resto pendiente.
- [ ] **CA-MCP-007**: `bills_pay` con `all_pending_up_to=2026-11` paga todas las pendientes con período menor o igual a noviembre y omite futuras; las ya pagadas se reportan `already_paid`.
- [ ] **CA-MCP-008**: un período con factura soft-deleted se reporta `deleted_slot` con su `bill_id`, no se recrea y no falla el resto del lote.
- [ ] **CA-MCP-009**: servicio inexistente, rango invertido, más de 36 meses, mes futuro, selector de pago ausente o doble devuelven 4xx con código estable y sin escrituras parciales.
- [ ] **CA-MCP-010**: las facturas creadas y pagadas por las tools quedan en `bill_history` con `source=mcp`; la UI sigue registrando `dashboard`.
- [ ] **CA-MCP-011**: `docs/mcp/` contiene todos los archivos de 5.7, enlaces relativos válidos, ejemplos sintéticos y frontmatter Docusaurus.
- [ ] **CA-MCP-012**: los ejemplos de configuración de OpenCode y Claude Desktop arrancan el binario y listan las tools usando `P40LA_API_KEY`.
- [ ] **CA-APIKEY-001**: regenerar la key devuelve el secreto completo una sola vez; `system_settings` guarda solo hash, prefijo y fechas; el estado nunca expone el secreto.
- [ ] **CA-APIKEY-002**: la key global accede a todas las rutas autenticadas del sistema; una key inválida, revocada o rotada devuelve 401 `auth_failed`.
- [ ] **CA-APIKEY-003**: el MCP funciona con `P40LA_API_KEY` y `Authorization: Bearer`, sin `P40LA_EMAIL`, `P40LA_PASSWORD` ni cookie.
- [ ] **CA-APIKEY-004**: las operaciones del MCP quedan en `bill_history` con `source=mcp`; una key no puede declarar `source=dashboard`.
- [ ] **CA-APIKEY-005**: rotar o revocar la key bloquea la siguiente petición de inmediato.
- [ ] **CA-APIKEY-006**: `scripts/create-api-key.sh` crea/rota la key global y no la persiste en el repo; si se usa `KEY_FILE`, lo escribe con permisos `0600`.
- [ ] **CA-UX-PREREQ**: el backend valida que el servicio exista y esté activo; el MCP devuelve un mensaje accionable.
- [ ] **CA-UX-FEEDBACK**: cada operación devuelve totales y detalle por ítem con estado y motivo; el `dry_run` permite previsualizar.

### 6.2 No funcionales

- [ ] **CA-ARCH-STACK**: el MCP compila con Go y no agrega Node.js ni un runtime nuevo al iHost.
- [ ] **CA-ARCH-FOLDERS**: existen `cmd/mcp-server/`, `internal/mcp/`, `internal/services/bill_bulk.go`, `internal/api/bill_bulk_handlers.go` y `docs/mcp/` con la estructura acordada.
- [ ] **CA-ARCH-LAYERS**: no hay SQL en handlers ni en `internal/mcp/`; las reglas de negocio están en `BillService`.
- [ ] **CA-ARCH-GO**: `gofmt`, errores explícitos, `slog` a stderr, configuración por env; `go vet ./...` sin hallazgos.
- [ ] **CA-ARCH-IHOST**: el add-on no arranca el MCP; el binario no se incluye en el proceso del server; RSS objetivo < 30 MB.
- [ ] **CA-DATA-MIGRATIONS**: la migración `0060_bill_history_source_mcp` es reversible y se aplicó correctamente en los tests; la key global usa `system_settings` sin migración.
- [ ] **CA-DATA-NO-DROP**: ninguna prueba toca `data/app.db`; se usa `/tmp/test-app.db`.
- [ ] **CA-DATA-PRODCOPY**: la validación con datos reales usa una copia `.backup` en `/tmp` y se borra al terminar.
- [ ] **CA-DATA-PROD-WRITE**: antes del primer backfill real se toma respaldo, se corre `dry_run` y el usuario confirma.
- [ ] **CA-SEC-NO-PII**: no hay PII, montos reales, cookies ni credenciales en código, docs, tests, issues ni logs; las API keys se guardan hasheadas y nunca se muestran en listados.
- [ ] **CA-SEC-SYNTHETIC**: todos los ejemplos usan `demo-*` y montos ficticios.
- [ ] **CA-SEC-PRODCOPY**: copias de producción solo en `/tmp`, eliminadas al finalizar.
- [ ] **CA-SEC-COMMIT**: `git diff --cached` revisado antes de cada commit.
- [ ] **CA-SEC-HOOKS**: hooks instalados y commits sin `--no-verify`.
- [ ] **CA-SEC-INCIDENT**: el protocolo de incidente está documentado en `docs/mcp/security.md`.
- [ ] **CA-SEC-LOGS**: logs sin payloads sensibles; solo conteos, IDs y códigos de error.
- [ ] **CA-TEST-LOCAL**: `go build ./...` y `go test ./...` pasan.
- [ ] **CA-TEST-SERVER**: `./scripts/check-server.sh` responde `{"status":"ok"}` antes de avisar al usuario.
- [ ] **CA-TEST-KILLALL**: se usa `killall`, nunca `pkill`.
- [ ] **CA-TEST-TESTUSER**: la integración autenticada usa `create-test-user.sh` y `/tmp/cookies.txt`.
- [ ] **CA-TEST-USER-FLOW**: se pregunta 🤖 antes de correr el server y se espera confirmación antes de cerrar.
- [ ] **CA-TEST-ROBOT**: las preguntas de flujo llevan 🤖.
- [ ] **CA-TEST-CHECKLIST**: checklist completa antes de `pending_release`.
- [ ] **CA-DELIV-WORKTREE**: todo el trabajo ocurre en `feature/SPEC-145`.
- [ ] **CA-DELIV-CONFIRM**: no hay merge, push ni deploy sin confirmación.
- [ ] **CA-DELIV-COMMIT**: mensajes `SPEC-145: ...`.
- [ ] **CA-DELIV-RELEASE**: release completo en el mismo ciclo.
- [ ] **CA-DELIV-CHECKRELEASE**: `/spec-check-release` ejecutado antes de declarar terminado.
- [ ] **CA-DELIV-HOOKS**: hooks instalados y activos, incluido el guardrail `check-mcp-docs.sh`.
- [ ] **CA-DELIV-MCP-DOCS**: `./scripts/check-mcp-docs.sh --staged` bloquea un cambio de superficie MCP sin `docs/mcp/` y el `pre-commit` lo ejecuta.
- [ ] **CA-DELIV-MCP-DOCS-SUBAGENT**: la actualización de `docs/mcp/` de esta spec fue redactada por un subagente y revisada por el agente principal.
- [ ] **CA-DELIV-LABELS**: issue con `spec/draft` y transiciones con labels válidos.
- [ ] **CA-DELIV-DEPLOY**: deploy aprobado por el usuario y `/health` verificado.
- [ ] **CA-DELIV-ESCALATION**: bloqueo > 3 días documentado y escalado.

### 6.3 Testing

- **Unit tests**:
  - Expansión de rango: enero–noviembre = 11; año bisiesto; servicio anual; clamp al mes actual; rango > 36.
  - Resolución de monto: explícito, última factura, `suggested_amount`, error si queda 0.
  - Duplicados: activo, pagado, soft-deleted.
  - Selectores de pago: IDs, mes, rango, `all_pending_up_to`; futuro; ya pagada.
  - Compuerta de escritura y `dry_run`.
  - Traducción de errores HTTP a errores MCP.
- **Integration tests**:
  - API falsa con `httptest` para el cliente MCP (login, 401 + relogin, timeouts).
  - Server real con DB en `/tmp/test-app.db`, usuario de `create-test-user.sh`, generación y pago de 11 facturas; verificar historial `source=mcp`.
  - Protocolo MCP: `initialize`, `tools/list`, `tools/call` con transporte en memoria; validar `isError` y contenido estructurado.
- **E2E/manual**:
  - Configurar OpenCode con el binario, ejecutar `bills_list`, `bills_generate` en `dry_run`, confirmar con el usuario y ejecutar la generación real sobre una copia o el iHost aprobado.
  - Repetir para pagar enero y luego todas las pendientes hasta la fecha.
- **Carga/Performance**:
  - 36 períodos en menos de 5 s en LAN; RSS medido con `/usr/bin/time -v` o `ps` menor a 30 MB.

---

## 7. Plan de Implementación

### 7.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|---|---|---|---|
| 1 | Servicio masivo (`bill_bulk.go`) + migración de auditoría `mcp` + tests unitarios | 1.5 días | Ninguna |
| 2 | Endpoints HTTP masivos + validaciones + tests de handlers | 1 día | Fase 1 |
| 2b | API key global: servicio, middleware, endpoints y script (sin tabla nueva) | 1 día | Fase 1 |
| 3 | Esqueleto MCP: config, cliente API, registry, transporte stdio | 1.5 días | Fase 2, 2b |
| 4 | Tools `bills_list`, `bills_generate`, `bills_pay` + tests con API falsa | 1.5 días | Fase 3 |
| 5 | Documentación `docs/mcp/` + ejemplos de configuración | 1 día | Fase 4 |
| 6 | Validación local contra copia de producción, prueba con usuario y ajustes | 1 día | Fases 1–5 |
| 7 | Release: build, merge, deploy aprobado y verificación `/health` | 0.5 día | Fase 6 |

### 7.2 Milestones

1. **MVP**: endpoints masivos + `bills_generate`/`bills_pay` funcionando por HTTP, con tests; sin MCP.
2. **V1.0**: MCP stdio con las tres tools, compuerta de escritura, documentación completa y validación del usuario.
3. **V1.1 (P1)**: fechas de vencimiento, montos por período, plantilla de número de factura y script de build multi-plataforma.

### 7.3 Estado de implementación

Implementado en `feature/SPEC-145`:

- Migración `0060_bill_history_source_mcp` (up/down).
- `BillSourceMCP`, `CreateWithSource`, `PayBillWithSource`, `SetCurrencyStorage`.
- `internal/services/bill_bulk.go` con generación/pago masivo, errores estables e idempotencia.
- Endpoints `POST /api/services/{service_id}/bills/generate` y `POST /api/services/{service_id}/bills/pay`.
- API key global hasheada en `system_settings`, middleware sesión-o-key y endpoints `GET/POST/DELETE /api/api-key`.
- `scripts/create-api-key.sh` para crear/rotar la key con una sesión.
- `internal/mcp/{config,api,registry,tools}` y `cmd/mcp-server`; el MCP usa solo `P40LA_API_KEY` con `Authorization: Bearer`.
- Tools `bills_list`, `bills_generate` y `bills_pay` con compuerta de escritura y `dry_run`.
- Documentación `docs/mcp/` lista para Docusaurus, incluida `api-key.md`.
- Reglas `DELIV-MCP-DOCS` y `DELIV-MCP-DOCS-SUBAGENT` + guardrail `scripts/check-mcp-docs.sh` integrado en `.githooks/pre-commit`.
- Tests de servicio, handlers, middleware, API key, cliente HTTP, tools y protocolo MCP.

Validación local completada contra una copia de la DB del iHost: service 12 quedó con 11 facturas pagadas y la generación repetida no duplicó. `go test ./...` pasa. Pendiente: release (merge a `main`, tag e imagen multi-arch) y deploy al iHost.

---

## 8. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Duplicar facturas por reintentos o doble ejecución | Media | Alto | Idempotencia por período; `dry_run` por defecto; sin reintentos automáticos en escrituras |
| Recrear una factura soft-deleted y violar `UNIQUE` | Media | Alto | `FindByServicePeriodIncludingDeleted`; se reporta `deleted_slot` y no se resucita |
| Monto incorrecto en un backfill | Media | Alto | Preview obligatorio, resolución explícita de monto y confirmación del usuario |
| Credenciales filtradas en logs o argumentos | Baja | Alto | API key solo por entorno; nunca payloads en logs; protocolo de incidente |
| API key global filtrada | Baja | Alto | Hash en DB, mostrada una vez, rotación/revocación inmediata y `last_used_at`; rotar ante cualquier duda. Al ser global, equivale a acceso completo |
| Carga excesiva al iHost | Baja | Medio | Máximo 36 períodos, llamadas secuenciales, timeout de 15 s |
| Drift del protocolo MCP | Media | Medio | SDK oficial fijado; contrato propio aísla las tools |
| Documentación desactualizada | Media | Medio | Checklist de docs en cada cambio de tool; ejemplos probados |
| Escrituras accidentales desde un agente | Media | Alto | `P40LA_MCP_ALLOW_WRITES=false` por defecto; `dry_run=true`; descripciones explícitas |
| Dependencia nueva incompatible con el build del iHost | Baja | Medio | El MCP es un binario separado; `go build ./...` y Docker del server siguen verificándose |
| PII en ejemplos o issues | Baja | Alto | Datos `demo-*`; revisión `SEC-COMMIT`; hooks |

---

## 9. Notas y Referencias

- Reglas aplicadas: `docs/rules/00-index.md`, `10-architecture.md`, `40-data.md`, `50-security.md`, `60-testing.md`, `70-delivery.md`.
- MCP: <https://modelcontextprotocol.io/specification/2025-06-18>
- SDK Go de MCP: <https://github.com/modelcontextprotocol/go-sdk>
- API existente: `internal/api/routes.go`, `internal/api/bill_handlers.go`, `internal/services/bill.go`, `internal/storage/bill.go`.
- Historial y borrado lógico: SPEC-070, SPEC-071.
- Worktrees y entrega: SPEC-066.
- Documentación de webhooks como referencia de estilo: `docs/webhooks-api.md`.
- Documentación del MCP: `docs/mcp/README.md`.
- Provisión de API key: `scripts/create-api-key.sh`.
- Issue de esta spec: [#148](https://github.com/paulomcnally/p40la-ihost/issues/148) (`spec/in_progress`).

---

## 10. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-23 | opencode | Creación inicial de la especificación |
| 2026-11-23 | opencode | Implementación inicial completa en `feature/SPEC-145`; pendiente validación del usuario y release |
| 2026-11-23 | opencode | Se reemplaza usuario/contraseña por una API key global del sistema, hasheada en `system_settings`, con rotación/revocación y provisión por script |
| 2026-11-23 | opencode | Se agregan `DELIV-MCP-DOCS` (docs obligatorias en cada cambio MCP) y `DELIV-MCP-DOCS-SUBAGENT` (docs por subagente) con hook `check-mcp-docs.sh` |
| 2026-11-23 | opencode | Merge con `main` (SPEC-143/144); migración renumerada a `0060`; validación local contra copia del iHost; estado `pending_release` |
