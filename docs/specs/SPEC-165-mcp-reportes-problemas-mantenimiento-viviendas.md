---
title: "MCP: reportes de problemas y mantenimiento en viviendas (ampliación: evidencia por MCP)"
id: "SPEC-165"
status: "draft"
author: "opencode"
created: "2026-11-27"
updated: "2026-11-27"
github_issue: 167
github_issue_amendment: 169
release: "2026-11-27; app v0.5.24 (workflow 46101201103); MCP mcp-v0.6.4 multi-arch (workflow 46101203390); merge 66885698; implementación 6c4a1625; bump 336a8324; issue #167 cerrado"
---

# MCP: reportes de problemas y mantenimiento en viviendas (ampliación: evidencia por MCP)

**ID**: SPEC-165
**Estado**: draft (ampliación de una spec released)
**Release original**: 2026-11-27 — app v0.5.24 (workflow `46101201103`); MCP `mcp-v0.6.4` multi-arch (workflow `46101203390`); merge `66885698`
**Ampliación**: 2026-11-27 — carga y administración de evidencia desde MCP
**Autor**: opencode
**Creado**: 2026-11-27
**Actualizado**: 2026-11-27

> Esta spec **conserva la identidad y las decisiones** de SPEC-165, ya liberada. No se crea una spec nueva ni se duplica el trabajo de edición: la ampliación agrega el alcance de evidencia (fotos y videos) al mismo flujo `home_work_request_*`. Los requerimientos y criterios originales se conservan tal como se liberaron; los nuevos se agregan en subsecciones marcadas «Ampliación».

---

## 1. Resumen Ejecutivo

SPEC-164 agregó a la app web los **reportes de problemas y mantenimiento por vivienda** (`home.workrequest`): un reporte nace en estado `reported`, sin insumos ni mano de obra, con evidencia en fotos/video, y puede convertirse en una reparación de `home.maintenance`, descartarse o reactivarse. El módulo HTTP ya expone `GET|POST /api/homes/{id}/workrequests`, `GET|PUT|DELETE .../{work_request_id}`, `.../convert` y `.../status`.

El servidor MCP no tiene tools para ese módulo. Un agente que opera p40la-ihost puede consultar y editar reparaciones (`home_maintenance_*`) pero no puede ver los reportes pendientes, reportar una falla nueva, convertirla en reparación ni descartarla. Esta spec agrega seis tools sobre la API existente, sin tocar el backend ni el esquema.

El resultado esperado es un toolset `home_work_request_*` de dos tools de lectura y cuatro de escritura, con `dry_run` por defecto y la compuerta de escritura del MCP. No hay migraciones, dependencias ni cambios en la app web; el release publica una nueva imagen `p40la-mcp`.

### 1.1 Ampliación 2026-11-27: evidencia por MCP

El caso de uso diario es que el usuario envíe **de 1 a 10 fotos o videos + contexto** en el chat y que el LLM cree el reporte y adjunte todos los archivos en el mismo flujo, sin entrar a la web. Hoy `home_work_request_create` crea el reporte pero la evidencia solo se carga desde la interfaz web: el agente ve las imágenes en el chat, pero la tool no recibe sus bytes ni tiene una vía para subirlos.

La ampliación agrega cuatro tools (`home_work_request_attachment_list`, `_upload_link`, `_add` y `_delete`), un endpoint REST de carga multipart autorizado por un **token firmado de corta vida y alcance único a un reporte**, idempotencia de alta (`client_request_id`) y deduplicación de archivos por SHA-256. El objetivo es que un agente pueda ejecutar el flujo completo —crear reporte → obtener link de carga → subir archivos → verificar— usando la red y los archivos que tenga disponibles, con un fallback humano explícito cuando el sandbox no pueda alcanzar el host.

**Restricción de exposición**: esta spec **no autoriza ni implementa** Tailscale Funnel ni un reverse proxy nuevo para el backend. El endpoint de carga queda disponible donde ya esté disponible la app (LAN/Tailscale). Exponerlo a Internet es una decisión operativa separada, con aprobación explícita del usuario, y debe limitarse al path de carga; nunca se publica el dashboard completo. Si el entorno del agente no alcanza el host, el flujo sigue siendo válido con el fallback web.

**Límites reales del entorno del agente (investigados, no asumidos)**:

- Los argumentos de una tool MCP son **JSON**. No existe un tipo de contenido “archivo” cliente→servidor: `ImageContent`, `AudioContent` y `EmbeddedResource` son resultados servidor→cliente. Un modelo de chat no puede transcribir un binario en base64 de forma fiable; `attachment_add` solo sirve cuando el **host/cliente** inyecta los bytes.
- La extensión oficial de archivos de MCP (`io.modelcontextprotocol/files`) exige que el conector esté listado en el Directory, un host público aprobado por Anthropic y la revisión MCP `2026-07-28`. Un servidor agregado por URL en `*.ts.net` **no recibe archivos** por esa vía; queda fuera de alcance.
- El sandbox de Claude **no tiene Internet abierto**: usa una allowlist de dominios (package managers) y `*.ts.net` no está. `/mnt/user-data/uploads` no es una ruta documentada; las rutas públicas documentadas son otras (`/mnt/session/uploads`, `$OUTPUT_DIR`) y dependen del producto. Por eso la spec **no depende** de esa ruta ni de `curl` desde el sandbox: es una capacidad condicional.
- El conector MCP sí es invocado desde la nube de Anthropic (no desde el sandbox), pero solo puede enviar JSON; no puede adjuntar bytes.
- Claude chat **no acepta video como entrada del modelo** (solo imágenes y documentos). Un video se puede **almacenar** como evidencia si los bytes llegan por el link o por la web, pero el agente no puede verlo ni describirlo.

Conclusión de diseño: la vía principal es **out-of-band** (link de carga + multipart o fallback humano). La vía inline queda acotada a clientes que inyecten bytes. El flujo cumple el criterio “sin pasos manuales” cuando el entorno dispone de red/archivos; cuando no, el agente debe decirlo y entregar `report_url`.

**Impacto iHost**: la carga se procesa de a un archivo (pico de memoria acotado por el límite de 25 MB existente), no se agregan dependencias (solo `crypto/hmac`, `crypto/sha256` y `encoding/base64` de la stdlib) y la migración agrega dos columnas, dos índices parciales y una tabla efímera de tokens.

---

## 2. Cumplimiento de reglas

Reglas del proyecto aplicables a esta spec (índice: `docs/rules/00-index.md`). Las reglas no se copian; se citan por ID.

| Regla | Cómo la cumple la spec | Criterio |
|---|---|---|
| `ARCH-MCP-TOOLS-MAP` | Las seis tools originales y las cuatro de evidencia se declaran en `docs/mcp/tools/impact.json` y en `contracts.json`. | `CA-MCP-WR-EV-MAP` |
| `DELIV-MCP-DOCS` | Se actualizan `docs/mcp/tools/home-work-requests.md`, `security.md`, `configuration.md`, `architecture.md`, `tools/README.md` y `docs/ihost.md`. | `CA-MCP-WR-EV-DOCS` |
| `DELIV-MCP-DOCS-SUBAGENT` | La documentación de `docs/mcp/` la redacta un subagente y se integra en el mismo commit. | `CA-MCP-WR-EV-DOCS-SUBAGENT` |
| `DELIV-MCP-IMPACT` | §6 completa, tools nombradas y `./scripts/check-mcp-impact.sh --staged --gate` verde. | `CA-MCP-WR-EV-IMPACT` |
| `DELIV-MCP-IMPACT-SUBAGENT` | Análisis de impacto delegado a un subagente `explore`; veredicto citado en §6. | `CA-MCP-WR-EV-IMPACT-SUBAGENT` |
| `TEST-MCP-CONTRACT` | `TestToolContracts`, `TestMCPBackendContract` y tests unitarios del cliente/toolset, incluido el flujo de carga. | `CA-MCP-WR-EV-CONTRACT` |
| `ARCH-STACK` | Go + `net/http` + SDK MCP existentes; cero dependencias nuevas. | `CA-MCP-WR-EV-STACK` |
| `ARCH-LAYERS` | Tools y handlers validan forma; `service.go` decide token, estado, deduplicación y límites; `storage.go` es el único con SQL. | `CA-MCP-WR-EV-LAYERS` |
| `ARCH-IHOST` | El MCP sigue siendo cliente HTTP; no abre SQLite ni el volumen de datos. La carga la escribe el backend. | `CA-MCP-WR-EV-IHOST` |
| `ARCH-MODULES` | El token y el endpoint de carga viven en `internal/modules/home/workrequest/`; no se toca el contrato de otros módulos. | `CA-MCP-WR-EV-LAYERS` |
| `DATA-MIGRATIONS` | Migración `0063` con `up`/`down` reversible; se prueba en `/tmp`. | `CA-MCP-WR-EV-MIGRATION` |
| `DATA-PRODCOPY` | La validación con datos reales, si aplica, usa copia en `/tmp`; nunca se escribe en producción. | `CA-MCP-WR-EV-PRODCOPY` |
| `DATA-NO-DROP` | No se borra ni reemplaza `data/app.db`; los tests usan DB temporal. | `CA-MCP-WR-EV-PRODCOPY` |
| `SEC-NO-PII` / `SEC-SYNTHETIC` | Ejemplos y tests con `Casa Demo`, `demo-*`, tokens `demo-token`; sin archivos ni metadatos reales. | `CA-MCP-WR-EV-SYNTHETIC` |
| `SEC-COMMIT` | `git diff --cached` revisado; sin tokens, archivos binarios ni DBs. | `CA-MCP-WR-EV-SECRETS` |
| `SEC-LOGS` | El token viaja en header, nunca en URL ni logs; no se registran nombres de archivo. | `CA-MCP-WR-EV-SECRETS` |
| `SEC-INCIDENT` | Si se filtra el secreto de firma: rotarlo, invalidar links y documentar. | `CA-MCP-WR-EV-SECRETS` |
| `TEST-LOCAL` | `go build ./...`, `go test ./...` y `check-server.sh` antes de entregar. | `CA-MCP-WR-EV-TESTS` |
| `TEST-SERVER` | `./scripts/check-server.sh` verde con el endpoint nuevo montado. | `CA-MCP-WR-EV-TESTS` |
| `TEST-TESTUSER` | La verificación manual usa `create-test-user.sh` y datos sintéticos. | `CA-MCP-WR-EV-TESTS` |
| `UI-*` / `UX-*` | `N/A`: no se agregan pantallas ni componentes. El fallback reutiliza la página existente de reportes. | — |
| `DELIV-WORKTREE` | Trabajo en `../p40la-ihost-spec-165`, rama `feature/SPEC-165`. | `CA-DELIV-WORKTREE` |
| `DELIV-CONFIRM` | Merge, release y publicación de imágenes requieren orden explícita; esta entrega es solo spec draft. | `CA-DELIV-CONFIRM` |

---

## 3. Requerimientos

### 3.1 Requerimientos Funcionales originales (P0 - Obligatorios, liberados)

1. **REQ-001 — Listado.** `home_work_request_list` lista los reportes de una vivienda con filtros opcionales `status` (`reported`/`converted`/`dismissed`) y `category` (`repair`/`improvement`/`maintenance`/`other`). Devuelve `pending_count`, `truncated` y los reportes con `attachment_count`, ordenados por `reported_at DESC, id DESC`.
2. **REQ-002 — Detalle.** `home_work_request_get` devuelve un reporte por vivienda con sus metadatos de evidencia (no el binario) y el vínculo `maintenance_id`/`converted_at`.
3. **REQ-003 — Alta.** `home_work_request_create` crea un reporte con `title` obligatorio, `description`, `category`, `priority` y `reported_at` (default: hoy, no futura). El backend fija `status=reported` y el reporte nace sin costos.
4. **REQ-004 — Edición.** `home_work_request_update` edita parcialmente `title`, `description`, `category`, `priority` y `reported_at`: los campos omitidos conservan su valor y el estado no se modifica. Devuelve `changed_fields` y `status=unchanged` si no hay cambios efectivos.
5. **REQ-005 — Conversión.** `home_work_request_convert` convierte un reporte `reported` en una reparación vacía, con `kind` (default `corrective`) y `area` (default `other`). Devuelve `maintenance_id` y el reporte `converted`. Si ya fue convertido, devuelve `already_converted`; un reporte descartado no se convierte.
6. **REQ-006 — Descarte y reactivación.** `home_work_request_status` cambia `reported → dismissed` o `dismissed → reported`; no admite `converted` ni otros estados.
7. **REQ-007 — Compuerta de escritura.** Las cuatro tools de escritura respetan `dry_run` (`true` por defecto) y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). Las de lectura no dependen de la compuerta.
8. **REQ-008 — Contrato y documentación.** Las seis tools se registran, se documentan, se declaran en el manifiesto de impacto y se congelan en `contracts.json`; `TestMCPBackendContract` cubre la cadena real contra el módulo `home.workrequest`.

### 3.2 Requerimientos Funcionales originales (P1 - Importantes, liberados)

1. **REQ-009 — Salida accionable.** Las tools de escritura devuelven `status` (`planned`/`created`/`updated`/`unchanged`/`converted`), `dry_run`, `changed_fields` cuando aplica y el registro completo; los `dry_run` no hacen POST/PUT.
2. **REQ-010 — Errores estables.** Se propagan `home_not_found`, `work_request_not_found`, `already_converted`, `invalid_status`, `writes_disabled` y `auth_failed` con mensajes y hints accionables.

### 3.3 Requerimientos No Funcionales originales

- **Dependencias**: cero nuevas en `go.mod`; se reutilizan `internal/mcp/api`, `toolkit` y el SDK MCP.
- **iHost**: el MCP sigue siendo un cliente HTTP; no abre SQLite ni el volumen de datos.
- **Rendimiento**: una llamada HTTP por lectura; las escrituras hacen como máximo un GET previo y una escritura.
- **Alcance**: no se modifica la app web, el backend ni las migraciones; solo el MCP, sus tests y su documentación.

### 3.4 Ampliación: Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-011 — Listado de evidencia.** `home_work_request_attachment_list(home_id, work_request_id)` devuelve `attachment_count`, `max_attachments` (10), `max_bytes` (25 MB), `remaining_slots` y `attachments[]` con `id`, `media_type`, `file_name`, `mime_type`, `size_bytes`, `sha256` y `created_at`. Es de solo lectura y no depende de la compuerta.
2. **REQ-012 — Link de carga.** `home_work_request_attachment_upload_link(home_id, work_request_id, count?, ttl_minutes?, dry_run?)` devuelve `upload_url`, `token`, `expires_at`, `max_bytes`, `max_request_bytes`, `max_files`, `accepted_mime`, `attachments_remaining`, `report_url`, `reachability` (`public`/`private`) y `curl_hint` con placeholder `<TOKEN>`. El token está firmado, es de corta vida (default 10 min, máximo 30), está limitado a **un** `home_id` + `work_request_id` y a **`max_files`** archivos; se persiste su `jti` para que el replay no supere ese cupo.
3. **REQ-013 — Carga REST multipart.** `POST /api/work-request-uploads` acepta `multipart/form-data` con 1..`max_files` partes `file`, autenticado con `Authorization: Bearer <token>`. Procesa los archivos de a uno (no bufferea el lote completo), valida firma real (magic bytes), MIME permitido, tamaño por archivo y cupo del reporte. Devuelve un resultado por archivo (`added`/`duplicate`/`rejected`) y el estado final del reporte.
4. **REQ-014 — Deduplicación de adjuntos.** En las vías MCP (token de carga e inline), cada adjunto guarda `sha256`; subir dos veces el mismo contenido al mismo reporte devuelve `status=duplicate` con el adjunto existente, sin crear fila ni archivo nuevo. La unicidad se garantiza con un índice parcial `(work_request_id, sha256) WHERE sha256 <> ''`. La carga desde la web conserva su comportamiento actual (no calcula hash ni deduplica); los adjuntos previos a la migración también quedan con `sha256=''` (REQ-027).
5. **REQ-015 — Idempotencia de alta.** `home_work_request_create` acepta `client_request_id` opcional (1..64 caracteres, `[A-Za-z0-9._:-]`). Repetir el alta con el mismo `client_request_id` en la misma vivienda devuelve el reporte existente con `status=existing` y `deduplicated=true`, sin crear un segundo registro ni adjuntos. Si se omite, el comportamiento actual no cambia.
6. **REQ-016 — Borrado de evidencia.** `home_work_request_attachment_delete(home_id, work_request_id, attachment_id, dry_run?)` elimina el adjunto y su archivo; `dry_run=true` (default) solo previsualiza. Requiere la compuerta de escritura y respeta el estado del reporte (ver REQ-018).
7. **REQ-017 — Compuerta y `dry_run`.** Las tools de escritura de evidencia (`upload_link`, `add`, `delete`) respetan `dry_run=true` por defecto y la compuerta `P40LA_MCP_ALLOW_WRITES` (`P40LA_MCP_REMOTE_WRITES` en HTTP). `attachment_list` es de lectura. `upload_link` con `dry_run=false` requiere además que el backend tenga configurado `UPLOAD_TOKEN_SECRET`; si no, devuelve `upload_link_disabled`.
8. **REQ-018 — Restricciones de estado.** El link de carga solo se emite y solo acepta archivos cuando el reporte está `reported`. Un reporte `dismissed` devuelve `invalid_status` con hint para reactivarlo; uno `converted` devuelve `invalid_status` (la evidencia se adjunta antes de convertir). El borrado de evidencia se permite en `reported` y `dismissed`, y se rechaza en `converted`. La carga desde la web existente no cambia de comportamiento.
9. **REQ-019 — Sin exposición pública nueva.** El endpoint de carga queda en el mismo origen del backend. Esta spec **no** configura Funnel ni reverse proxy y **no** publica el dashboard. El resultado de `upload_link` clasifica `reachability` y documenta el fallback: abrir `report_url` en la web y usar el botón «Adjuntar» para cargar los archivos de a uno. Exponer el endpoint a Internet requiere una decisión operativa separada con aprobación explícita y un proxy limitado al path de carga.
10. **REQ-020 — Contrato, manifiesto y documentación.** Las cuatro tools nuevas se registran, se documentan, se declaran en `impact.json`, se congelan en `contracts.json` y se cubren en `TestMCPBackendContract` con el flujo completo (crear idempotente → link → upload multipart → listar → borrar).

### 3.5 Ampliación: Requerimientos Funcionales (P1 - Importantes)

1. **REQ-021 — Carga inline para archivos pequeños.** `home_work_request_attachment_add(home_id, work_request_id, filename, mime?, media_type?, data_base64, sha256?, dry_run?)` decodifica base64 en el MCP y reenvía multipart al endpoint autenticado existente. Acepta hasta **256 KiB** de contenido real por llamada (base64 ≈ 341 KiB) para no superar el límite de 1 MiB del body de `/mcp`. Es idempotente por `sha256`. Está pensada para clientes que **inyectan bytes** en los argumentos de la tool; un modelo de chat no puede transcribir un binario, así que no es la vía principal para Claude.
2. **REQ-022 — Errores estables de evidencia.** Se agregan `attachment_limit`, `attachment_not_found`, `invalid_token`, `expired_token`, `upload_token_exhausted`, `invalid_file`, `unsupported_media`, `request_too_large`, `upload_link_disabled`, `invalid_client_request_id` y `invalid_status`, con `message` y `hint` accionables. El endpoint devuelve códigos HTTP coherentes (`400`, `401`, `404`, `409`, `413`, `415`, `429`).
3. **REQ-023 — Rate limit del endpoint.** El endpoint de carga limita a **30 requests por minuto** por hash de token (ventana fija en memoria, suficiente para una instancia) y responde `429` con `Retry-After`. No se usa `X-Forwarded-For` (es falsificable sin un proxy confiable). El contador se reinicia con el proceso.
4. **REQ-024 — Verificación posterior.** `attachment_list` es la vía canónica para confirmar que los N archivos quedaron adjuntos; el agente debe llamarla después de subir y reportar `attachment_count` y `remaining_slots` al usuario.
5. **REQ-025 — Logs y auditoría.** El token no se registra en logs, auditoría MCP ni mensajes de error. El endpoint registra método, path fijo, estado y duración; nunca el header `Authorization`, el token, el nombre de archivo ni el contenido.

### 3.6 Ampliación: Requerimientos Funcionales (P2 - Deseables)

1. **REQ-026 — Carga reanudable por chunks.** Para videos muy grandes, un endpoint de carga por chunks con reanudación. **Fuera de alcance** de esta iteración; se documenta como evolución y no se implementa.
2. **REQ-027 — Backfill de hashes.** Calcular `sha256` de los adjuntos existentes (previos a la migración) para deduplicar también contra ellos. **Fuera de alcance**; se documenta la limitación.
3. **REQ-028 — Token estrictamente de un solo uso.** Además del cupo por `max_files`, marcar el token como consumido tras la primera request. **Fuera de alcance**; se mitiga con TTL corto, cupo por token y rate limit.
4. **REQ-029 — Sanitización de metadatos.** Eliminar EXIF/GPS de las fotos y metadatos equivalentes de video antes de guardar. **Fuera de alcance** de esta iteración: hoy los bytes se guardan tal cual (incluido EXIF). Se documenta el riesgo y se recomienda al usuario revisar las fotos antes de subirlas.

### 3.7 Ampliación: Requerimientos No Funcionales

- **Memoria**: la carga multipart se procesa archivo por archivo; el pico es el tamaño máximo de un archivo (25 MB) más el overhead de multipart. No se bufferean los N archivos juntos.
- **Disco**: el tope existente de 10 adjuntos × 25 MB por reporte se mantiene (250 MB por reporte como máximo teórico).
- **Base de datos**: dos columnas nuevas (`client_request_id`, `sha256`), dos índices parciales y una tabla efímera de tokens (`work_request_upload_tokens`); sin cambios destructivos sobre reportes ni adjuntos.
- **Dependencias**: cero nuevas; `crypto/hmac`, `crypto/sha256`, `encoding/base64`, `mime/multipart` y `net/http` son stdlib.
- **Rendimiento**: el endpoint autenticado de carga y el token de corta vida agregan una verificación HMAC y una lectura de reporte por request; el costo es despreciable frente a la escritura del archivo.
- **Disponibilidad**: si el secreto de firma no está configurado, el link de carga se deshabilita con un error claro; el resto de las tools sigue funcionando.
- **iHost**: sin procesos nuevos, sin cron, sin acceso del MCP a SQLite ni al filesystem de evidencia.

---

## 4. Investigación y Decisiones Técnicas

### 4.1 Contexto investigado (original)

- El módulo `home.workrequest` (SPEC-164) expone endpoints autenticados bajo `/api/homes/{id}/workrequests` y usa `status`: `reported`, `converted`, `dismissed`.
- Las tools existentes de `home.maintenance` (`internal/mcp/tools/homemaint/`) son el patrón canónico: `api.HomeMaintenanceAPI`, toolset con `writeMode`, `dry_run` y `toolkit.APIError`.
- El catálogo MCP se monta en `internal/mcp/tools/all/register.go`; los contratos se congelan con `TestToolContracts` y el manifiesto `impact.json` se valida con `check-mcp-impact.sh`.
- `FullAPI` en `internal/mcp/api/autos.go` es la unión de contratos que usa el servidor MCP; agregar un contrato nuevo exige actualizar el stub de `contract_test.go`.

### 4.2 Opciones evaluadas (original)

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| Un toolset nuevo por dominio (`workrequest`) | Aísla el dominio, sigue el patrón de `homemaint` | Un paquete más | ✅ Seleccionada |
| Meter las tools dentro de `homemaint` | Menos paquetes | Mezcla reportes con reparaciones y confunde el catálogo | ❌ Rechazada |
| Tool única `home_work_request_manage` con `action` | Menos tools | Pierde esquemas tipados y validación por acción; no sigue la convención del proyecto | ❌ Rechazada |
| Incluir subida de evidencia por MCP (evaluación original) | Cobertura total | Requiere binarios/base64 y multipart; el MCP no maneja archivos y la web ya lo cubre | ⚠️ Revisada en la ampliación (§4.5) |

### 4.3 Decisiones arquitectónicas originales (ADRs)

**ADR-001 — Seis tools tipadas, no una tool genérica.**
- **Contexto**: el catálogo usa un verbo por tool y esquemas de entrada congelados.
- **Decisión**: `home_work_request_list`, `_get`, `_create`, `_update`, `_convert` y `_status`.
- **Consecuencias**: contratos claros y validación por tool; el agente descubre capacidades por nombre.

**ADR-002 — La evidencia se consulta, no se sube por MCP (original).**
- **Contexto**: las tools MCP hablan JSON; la evidencia es multipart y puede pesar 25 MB.
- **Decisión original**: `home_work_request_get` devuelve metadatos de adjuntos y `home_work_request_list` el conteo; la carga/borrado de archivos queda en la UI.
- **Consecuencias**: sin base64 ni límites de contexto; el agente puede pedir al usuario que adjunte desde la web.
- **Revisión 2026-11-27**: se mantiene para el binario grande, pero se agrega una vía de carga para archivos pequeños y un link HTTP para el resto. Ver ADR-004..007.

**ADR-003 — `status` separado de `update`.**
- **Contexto**: el backend no permite editar `status` por PUT; los cambios de estado son acciones de negocio.
- **Decisión**: `home_work_request_status` con `status` `reported`/`dismissed`; `convert` es una acción aparte.
- **Consecuencias**: no se puede dejar un reporte en un estado inválido por edición.

### 4.4 Ampliación: investigación realizada

**Fuentes internas (código y reglas)**

- `internal/modules/home/workrequest/`: endpoints `/attachments` ya existen para la web (`POST`, `GET`, `DELETE`), con detección de MIME por magic bytes, límite de 10 adjuntos y 25 MB por archivo, y almacenamiento en `DATA_DIR/workrequests/<work_request_id>/<attachment_id>-<name>`.
- `internal/api/middleware.go`: `SessionOrAPIKeyMiddleware` acepta sesión o API key global (`Authorization: Bearer` o `X-API-Key`). No existe un patrón de token temporal para terceros.
- `internal/mcp/server/server.go`: el listener HTTP de `/mcp` limita el body a **1 MiB** (`maxMCPBody = 1 << 20`), con rate limit por token/IP y timeouts. Ese límite acota cualquier `data_base64` que viaje como argumento de tool.
- `internal/mcp/api/client.go`: cliente JSON con `Authorization: Bearer <API key>` y `X-P40LA-Source: mcp`; no tiene helper multipart.
- `docs/mcp/security.md`: dos credenciales separadas (OAuth para `/mcp`, API key para el backend); el dashboard del iHost se publica con Tailscale **Serve** (solo tailnet) y el MCP tiene un nodo dedicado con **Funnel**. El backend de la app **no** está publicado a Internet.
- `docs/mcp/tools/impact.json`: 59 tools registradas; las tools de reportes declaran sus endpoints y tests. `migrations/` está en `shared.affects_all`, por lo que una migración obliga a declarar impacto.
- `migrations/0062_home_work_requests.up.sql`: tablas `home_work_requests` y `home_work_request_attachments` sin `sha256` ni `client_request_id`.

**Fuentes externas (cliente MCP, adjuntos y exposición)**

- **Argumentos = JSON.** En `tools/call`, `params.arguments` es un objeto JSON validado por `inputSchema`; no hay bloques de contenido en la petición. `ImageContent`, `AudioContent`, `EmbeddedResource` y `ResourceLink` son contenido de **resultado** (servidor→cliente). Por tanto, no existe un “archivo como argumento” estándar ([MCP Tools, rev. 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/server/tools)).
- **Base64 solo funciona si lo inyecta el host.** Un modelo tendría que emitir los bytes como tokens; la extensión oficial de archivos prohíbe expresamente poner bytes de archivo en el contexto del modelo y recomienda sustituir un handle antes de enviar la llamada ([File Uploads WG](https://github.com/modelcontextprotocol/experimental-ext-file-uploads/blob/main/specification/draft/file-uploads.md)). Esto confirma que `attachment_add` no es la vía principal para Claude chat.
- **Extensión oficial `io.modelcontextprotocol/files` (beta).** El flujo correcto sería: la tool recibe metadata, responde `upload_required` con un target HTTPS y el host sube los bytes out-of-band; luego la tool se reintenta con `requestState`. Requiere que el conector esté **listado en el Directory**, que la dirección coincida con la del listing, un host público aprobado por Anthropic, propiedad del dominio y revisión MCP `2026-07-28` ([Claude — File transfers](https://claude.com/docs/connectors/building/files)). Un conector agregado por URL en `*.ts.net` **no recibe archivos** por esta vía; queda fuera de alcance.
- **Sandbox con egress limitado.** En la API el sandbox no tiene red; en claude.ai el acceso depende del plan y pasa por una allowlist de dominios (npm, PyPI, GitHub, crates, Ubuntu, Yarn); `*.ts.net` no está. Free/Pro/Max no permiten agregar dominios; Team/Enterprise sí, pero es una configuración del owner ([Code execution — Networking](https://platform.claude.com/docs/en/agents-and-tools/tool-use/code-execution-tool#networking-and-security), [Create and edit files](https://support.claude.com/en/articles/12111783-create-and-edit-files-with-claude)). La ruta `/mnt/user-data/uploads` **no está documentada** como contrato; las rutas documentadas son `/mnt/session/uploads` (Managed Agents) y `$OUTPUT_DIR` (code execution).
- **Video no es entrada del modelo.** Claude chat acepta imágenes (JPEG/PNG/GIF/WebP) y documentos; no hay soporte documentado de video como entrada. El video se puede almacenar como evidencia si los bytes llegan por el link, pero el agente no puede verlo.
- **Límites de archivos del chat.** Chat: 500 MB por archivo, 20 archivos por chat, 10 MB por imagen, 20 imágenes por mensaje. Project files y code execution: 30 MB por archivo ([Upload files to Claude](https://support.claude.com/en/articles/8241126-upload-files-to-claude), [Vision limits](https://platform.claude.com/docs/en/build-with-claude/vision#request-limits)). La spec no asume estos límites: el dominio p40la mantiene 10 adjuntos y 25 MB por archivo.
- **Tailscale Funnel.** Publica un listener en 443/8443/10000 y permite montar un prefijo con `--set-path`; **todo** lo que el backend sirva bajo ese prefijo queda público. No tiene auth, WAF ni rate limit propios ([Tailscale Funnel](https://tailscale.com/kb/1223/funnel), [CLI](https://tailscale.com/docs/reference/tailscale-cli/funnel)). Por eso publicar el backend completo sigue siendo riesgoso; si se expone, debe montarse un prefijo dedicado que solo enrute el endpoint de carga, y eso queda fuera de esta spec.
- **Buenas prácticas de upload.** Allowlist de MIME y firma real, nombre generado/sanitizado, límite de tamaño, autorización antes de aceptar bytes, rate limit, no seguir redirects, hash SHA-256 y staging antes de promover ([OWASP File Upload](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html)). La extensión oficial exige targets de un solo archivo y one-write; para nuestro caso se traduce en un token acotado por reporte y por cantidad de archivos.

> **Incertidumbre documentada**: el soporte exacto de adjuntos del cliente MCP de Claude cambia con el producto y no se puede verificar desde este repo. La spec no depende de él: define una vía HTTP (link + multipart) que funciona con cualquier cliente que pueda ejecutar `curl` o abrir una URL, una vía inline acotada para hosts que inyecten bytes, y un fallback humano. La alcanzabilidad de `*.ts.net` desde el sandbox con “All domains” (Team/Enterprise) no está documentada y no se asume.

**Validación subagente (investigación)**: explore — sesión `ses_ddfc5e210ffeLyqICFJTkdMtZp` — veredicto: la evidencia existente se apoya en un endpoint multipart autenticado por sesión/API key; no hay token temporal ni deduplicación; el MCP no tiene helper multipart y el body de `/mcp` es 1 MiB; el backend no está publicado a Internet. general — sesión `ses_ddfc5e20dffeSGZ8w5l3vC1gHk` — veredicto: los argumentos de tool son JSON; el binario tipado MCP es servidor→cliente; la extensión oficial de archivos exige Directory + host aprobado y no aplica a un conector por URL en `*.ts.net`; el sandbox usa allowlist de egress y no alcanza `*.ts.net` por defecto; Funnel puede montar un prefijo con `--set-path` pero expone todo lo que el backend sirva bajo ese prefijo; video no es entrada del modelo. El detalle de fuentes está en §10.

### 4.5 Ampliación: opciones evaluadas para transportar evidencia

| Opción | Pros | Contras | Decisión |
|---|---|---|---|
| **A. Link de carga + endpoint REST multipart en el backend** | Funciona con cualquier cliente que pueda ejecutar `curl`; reutiliza la validación y el almacenamiento existentes; el token se valida en el dominio; no duplica archivos en el MCP | Requiere endpoint sin sesión (protegido por token) y una migración para deduplicar; si el backend no es alcanzable, no sirve | ✅ **Seleccionada** |
| B. Endpoint de carga alojado en el MCP (segundo salto al backend) | Reutiliza el Funnel ya público del MCP; no expone la app | Duplica la lógica de carga y el secreto en el MCP; el listener de `/mcp` tiene body de 1 MiB y habría que abrir otro path; en stdio no hay listener; más superficie y más código | ❌ Rechazada |
| C. Solo `attachment_add` con base64 en argumentos | Una sola tool, sin endpoint nuevo | El body de `/mcp` es 1 MiB; el modelo no puede transcribir binarios; no cubre videos ni 10 archivos | ❌ Rechazada como vía principal; se conserva acotada como P1 |
| D. Solo fallback web (sin cambios) | Cero riesgo | No cumple el caso de uso “sin entrar a la web” | ❌ Rechazada |
| E. Publicar el backend completo con Funnel y cargar directo | Menos código | Expone el dashboard y todo `/api`; Funnel no tiene auth ni WAF propios; riesgo alto | ❌ Rechazada explícitamente |
| F. Extensión oficial MCP `io.modelcontextprotocol/files` | Es el mecanismo estándar y el host sube out-of-band; no pasa bytes por el modelo | Requiere conector listado en el Directory, host público aprobado por Anthropic, propiedad del dominio y revisión MCP 2026-07-28; un conector por URL en `*.ts.net` no recibe archivos | ❌ Fuera de alcance (futuro si se publica en el Directory con dominio propio) |
| G. MCP App con file picker (iframe) | El usuario elige N archivos en el cliente y suben directo al backend; cubre imágenes y video | Requiere implementar y mantener una app MCP, CSP `connectDomains`, render en web/móvil y pruebas por cliente; más superficie y UX nueva | ⏸️ Diferida (evolución) |

### 4.6 Ampliación: decisiones arquitectónicas (ADRs)

**ADR-004 — El token de carga lo emite y valida el backend, no el MCP.**
- **Contexto**: el secreto de firma y la validación de estado/permisos viven en el dominio; el MCP ya es un cliente HTTP del backend.
- **Decisión**: `POST /api/homes/{id}/workrequests/{wid}/upload-link` (autenticado por sesión/API key) emite el token; `POST /api/work-request-uploads` (sin sesión, token en `Authorization`) lo valida y escribe. El MCP solo orquesta.
- **Consecuencias**: una sola implementación de token, reutilizable por la web o por otros clientes; el MCP no necesita el secreto; el endpoint nuevo queda en el mismo origen del backend.

**ADR-005 — `attachment_add` es una vía acotada, no la principal.**
- **Contexto**: el body de `/mcp` es 1 MiB; los argumentos de tool son JSON y el modelo no puede emitir binarios fiables.
- **Decisión**: aceptar hasta 256 KiB de contenido real por llamada, solo para clientes/hosts que **inyectan bytes** en los argumentos; documentar que Claude chat debe usar `upload_link` y que Claude Code puede usar `curl`/stdio con acceso al filesystem.
- **Consecuencias**: no se promete una capacidad que el cliente no puede cumplir; se evita subir el límite de body de `/mcp` y con ello el riesgo de DoS. La extensión oficial de archivos MCP queda fuera de alcance (ADR-012).

**ADR-006 — Idempotencia explícita: `client_request_id` para reportes y `sha256` para adjuntos.**
- **Contexto**: los reintentos del agente y de la red no deben duplicar registros ni archivos.
- **Decisión**: índice único parcial `(home_id, client_request_id)` y `(work_request_id, sha256)`; la API devuelve el registro existente con `deduplicated=true` en vez de un error.
- **Consecuencias**: reintentar es seguro; el cliente debe enviar `client_request_id` estable (por ejemplo, un UUID derivado del mensaje). Si se omite, se conserva el comportamiento actual (sin deduplicación de reportes).

**ADR-007 — Token HMAC en header, corto, acotado por reporte y por cantidad de archivos.**
- **Contexto**: el endpoint puede quedar alcanzable desde la red; un token puramente stateless sería replayable hasta expirar. El iHost es una sola instancia y ya tiene SQLite.
- **Decisión**: `v1.<payload_b64url>.<sig_b64url>` con HMAC-SHA256 y secreto dedicado `UPLOAD_TOKEN_SECRET`; payload `{hid, wid, exp, jti, max_files}`; TTL default 10 min, máximo 30; se envía en `Authorization: Bearer`, nunca en la URL. Se persiste una fila por `jti` en `work_request_upload_tokens` con `max_files`, `files_used` y `expires_at`.
- **Consecuencias**: un token permite **varias** requests (por ejemplo, un `curl` por archivo) pero no más archivos que `max_files`; el replay no puede llenar el reporte más allá de lo pedido. Rotar el secreto invalida todos los links; borrar la fila revoca el token. La limpieza de filas expiradas es oportunista (al emitir un link nuevo). No se implementa token estrictamente de un solo uso (REQ-028).

**ADR-008 — La evidencia se adjunta antes de convertir.**
- **Contexto**: un reporte `converted` ya generó una reparación y su evidencia es histórica.
- **Decisión**: emitir links y aceptar cargas solo en `reported`; borrar en `reported`/`dismissed`; rechazar `converted` con `invalid_status`. La web conserva su comportamiento actual.
- **Consecuencias**: el flujo diario (reportar → adjuntar → convertir) es consistente; si hace falta evidencia después de convertir, se usa la web o se reabre el flujo con una decisión de producto.

**ADR-009 — Fallback humano explícito.**
- **Contexto**: el sandbox del agente puede no alcanzar el host.
- **Decisión**: `upload_link` devuelve `report_url` (página web existente del reporte) y `reachability`; la documentación indica abrir esa página y cargar los archivos con el botón «Adjuntar» (hoy acepta un archivo por vez) cuando la carga automática no sea posible.
- **Consecuencias**: el usuario nunca queda bloqueado; el agente debe decir la verdad sobre si pudo subir los archivos.

**ADR-010 — Sin exposición pública nueva en esta spec.**
- **Contexto**: el backend no está publicado; Funnel puede montar un prefijo (`--set-path`) pero expone todo lo que el backend sirva bajo ese prefijo y no tiene auth/WAF propios.
- **Decisión**: no se configura Funnel ni reverse proxy; el endpoint queda en el origen actual. Exponerlo es una decisión operativa separada, con aprobación explícita, TLS y un prefijo/proxy dedicado que solo enrute `POST /api/work-request-uploads`.
- **Consecuencias**: el flujo remoto funciona solo si el entorno alcanza el host; el release no cambia la superficie de red.

**ADR-011 — El video se almacena, no se analiza.**
- **Contexto**: Claude chat no acepta video como entrada del modelo; el agente solo ve imágenes y documentos.
- **Decisión**: permitir video como evidencia (MP4/QuickTime/WebM) en el almacenamiento y en el link de carga, pero no prometer que el agente lo describa o resuma. El contexto del reporte lo aporta el usuario o una imagen/frame que él adjunte.
- **Consecuencias**: el flujo “1–10 fotos o videos” se cumple para **adjuntar**, no para **entender** el video; la documentación lo dice explícitamente.

**ADR-012 — La extensión oficial de archivos MCP queda fuera de alcance.**
- **Contexto**: `io.modelcontextprotocol/files` resolvería el flujo out-of-band de forma estándar, pero exige Directory, host aprobado, propiedad del dominio y revisión MCP 2026-07-28; un conector por URL en `*.ts.net` no recibe archivos.
- **Decisión**: no se implementa `x-mcp-file` ni `upload_required` en esta iteración. Se documenta como evolución si el conector se publica en el Directory con un dominio propio.
- **Consecuencias**: el diseño no depende de una capacidad beta cerrada; el link propio cubre el caso hoy.

---

## 5. Diseño Técnico

### 5.1 Diagrama

```
[Agente MCP] -> [tool: home_work_request_*] -> [api.Client] -> HTTP /api/homes/{id}/workrequests...
                                                          |
                                                [módulo home.workrequest]
                                                          |
                                            [home_maintenances] (solo al convertir)

Ampliación de evidencia:

[Agente] --home_work_request_create(client_request_id)--> [MCP] --> POST /api/homes/{id}/workrequests
   |
   +--home_work_request_attachment_upload_link--> [MCP] --> POST .../upload-link
   |                                                        (HMAC token, scope 1 reporte)
   |
   +--curl -F file=@... + Authorization: Bearer <token>--> POST /api/work-request-uploads
   |                                                        (sin sesión; token; multipart)
   |                                                             |
   |                                                             v
   |                                                    [service.AddAttachment]
   |                                                    (magic bytes, 25 MB, cupo 10)
   |                                                             |
   |                                                             v
   |                                                    DATA_DIR/workrequests/<wid>/
   |
   +--home_work_request_attachment_list--> [MCP] --> GET .../{wid} (metadatos)
```

### 5.2 Componentes

#### 5.2.1 Cliente API — `internal/mcp/api/home_work_request.go`
- `HomeWorkRequestAPI` suma `ListHomeWorkRequestAttachments` (puede reutilizar `GetHomeWorkRequest`), `CreateHomeWorkRequestUploadLink`, `AddHomeWorkRequestAttachment` y `DeleteHomeWorkRequestAttachment`.
- DTOs nuevos: `HomeWorkRequestUploadLink`, `HomeWorkRequestUploadLinkRequest`, `HomeWorkRequestAttachmentResult`.
- `HomeWorkRequestRequest` suma `ClientRequestID *string` (solo create).
- `HomeWorkRequest` y `HomeWorkRequestAttachment` suman `deduplicated` y `sha256`.
- Escrituras con `sourceHeaders()` (`X-P40LA-Source: mcp`).

#### 5.2.2 Cliente multipart — `internal/mcp/api/upload.go` (nuevo)
- `postMultipart(ctx, path, fields map[string]string, fileName, mime string, data []byte, out any)`.
- Reutiliza `c.http`, `c.apiKey` y `sourceHeaders()`; límite de respuesta 1 MiB; errores tipados `APIError`.
- No escribe el archivo en disco: decodifica base64 en memoria y lo envía al backend.

#### 5.2.3 Toolset — `internal/mcp/tools/workrequest/`
- `attachment_list.go`, `attachment_upload_link.go`, `attachment_add.go`, `attachment_delete.go`.
- `workrequest.go`: registrar las cuatro tools; constantes `maxAttachments=10`, `maxAttachmentBytes=25<<20`, `maxInlineBytes=256<<10`, `acceptedMIME`.
- `upload_link` valida `count` (1..10), `ttl_minutes` (1..30), lo acota a `remaining_slots` (si es 0, `attachment_limit`) y compone `upload_url`/`report_url` con `cfg.UploadPublicURL` o `cfg.BaseURL`; `reachability=public` solo si `P40LA_UPLOAD_PUBLIC_URL` está definido, y `private` si se usa `P40LA_BASE_URL`.
- `attachment_add` valida base64, tamaño decodificado ≤ 256 KiB, `sha256` opcional y deriva `media_type` de `media_type`, del prefijo de `mime` o, si ambos faltan, de una detección local de magic bytes; si no puede determinarlo, `invalid_request` pidiendo `media_type`. El backend vuelve a validar la firma real.
- `attachment_delete` hace GET previo para validar existencia/estado y devuelve `status=planned|deleted|unchanged`.
- `toolkit.APIError`: agregar hints para `attachment_limit`, `attachment_not_found`, `request_too_large`, `invalid_file`, `upload_link_disabled`, `upload_token_exhausted` e `invalid_status`, para que los errores del backend sean accionables.
- Las tools nuevas devuelven `map[string]any`/`any` con `toolkit.JSONResult`, igual que las existentes, para no introducir un `output_schema` inferido en `contracts.json`.

#### 5.2.4 Backend — `internal/modules/home/workrequest/`
- `token.go` (nuevo): `MintUploadToken`, `ParseUploadToken`, `UploadTokenConfig` (secreto, TTL), errores `invalid_token`/`expired_token`/`upload_token_exhausted`.
- `model.go`: `ClientRequestID`, `Attachment.SHA256`, `Deduplicated`; `CreateInput` (embed `Input` + `ClientRequestID`); `UploadLink`, `UploadLinkRequest`, `UploadResult`, `UploadFileResult`, `UploadToken`.
- `storage.go`: `FindByClientRequestID`, `GetAttachmentBySHA256`, `CreateAttachment` con `sha256`; `CreateUploadToken`, `GetUploadToken`, `ConsumeUploadSlot` (incremento condicional), `DeleteExpiredUploadTokens`; manejo de colisión única.
- `service.go`: `Create` idempotente; `MintUploadLink` (valida home/reporte/estado/cupo, inserta la fila del `jti`); `UploadWithToken` (valida token/fila/estado, consume cupo por archivo nuevo, procesa archivos); `AddEvidence` (infiere `media_type` por magic bytes, calcula `sha256` y deduplica; lo usan el token y `attachment_add`); `DeleteAttachment` con restricción de estado para el flujo por token (la web conserva el suyo). El `AddAttachment` existente de la web no cambia: no calcula hash ni deduplica. El chequeo del cupo global (10), el incremento del cupo del token y la inserción del adjunto van en una **misma transacción** (`BEGIN IMMEDIATE`) para cerrar la carrera actual entre uploads concurrentes. Si falla la escritura atómica del archivo después del commit, se borra la fila y se libera el slot del token (`files_used = files_used - 1`) en una transacción de compensación.
- `handlers.go`: `POST .../upload-link` (auth normal), `POST .../attachments/evidence` (auth normal, una sola parte `file`, usa `AddEvidence` y devuelve `deduplicated`) y `POST /api/work-request-uploads` (**sin** envolver con `auth`; el handler valida el token del header), `MaxBytesReader` por request y rate limit en memoria. El endpoint público se monta directo en `h.register`: `mux.Handle("POST /api/work-request-uploads", http.HandlerFunc(h.uploadByToken))`.
- `module.go`: recibe `UploadConfig`, registra ambos endpoints y agrega `0063` a `Info().Migrations`.
- `handlers_test.go`: `TestModuleInfoAndRoutes` espera hoy exactamente `1` migración (`0062`); actualizar a `2` y a `0062` + `0063`.

#### 5.2.5 Configuración
- Backend (`internal/config/config.go`): `UploadTokenSecret` (`UPLOAD_TOKEN_SECRET`, mínimo 32 caracteres; vacío = función deshabilitada), `UploadLinkTTL` (`UPLOAD_LINK_TTL`, default `10m`, máximo `30m`).
- MCP (`internal/mcp/config/config.go`): `UploadPublicURL` (`P40LA_UPLOAD_PUBLIC_URL`, opcional; si está vacío se usa `P40LA_BASE_URL`). Si se define, debe ser `https` salvo loopback, sin query/fragment; puede incluir un prefijo de proxy.
- `internal/modules/all/register.go` y `cmd/server/main.go`: pasan `workrequest.UploadConfig` al módulo.
- `docs/ihost.md`: documenta `UPLOAD_TOKEN_SECRET` y `UPLOAD_LINK_TTL` (backend).
- `docs/mcp/configuration.md` y `deploy/mcp/mcp.env.example`: documentan `P40LA_UPLOAD_PUBLIC_URL` (MCP).

### 5.3 Modelo de datos

Migración `migrations/0063_work_request_evidence_idempotency.up.sql`:

```sql
ALTER TABLE home_work_requests ADD COLUMN client_request_id TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_home_work_requests_client_request
  ON home_work_requests(home_id, client_request_id)
  WHERE client_request_id <> '';

ALTER TABLE home_work_request_attachments ADD COLUMN sha256 TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_home_work_request_attachments_sha
  ON home_work_request_attachments(work_request_id, sha256)
  WHERE sha256 <> '';

CREATE TABLE IF NOT EXISTS work_request_upload_tokens (
    jti TEXT PRIMARY KEY,
    work_request_id INTEGER NOT NULL,
    max_files INTEGER NOT NULL,
    files_used INTEGER NOT NULL DEFAULT 0,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (work_request_id) REFERENCES home_work_requests(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_work_request_upload_tokens_expiry
    ON work_request_upload_tokens(expires_at);
```

`0063_...down.sql`: elimina la tabla de tokens, ambos índices y luego ambas columnas (`DROP COLUMN`; el proyecto ya usa ese patrón en migraciones previas). No hay pérdida de datos de reportes/adjuntos existentes; los tokens son efímeros.

Entidades:

```
WorkRequest
- client_request_id: string (interno, no se devuelve en el JSON; vacío si no se envió; único por home cuando no está vacío)
- deduplicated: bool (solo respuesta de create)
- attachments: Attachment[]

Attachment
- sha256: string hex (vacío en filas anteriores a 0063)
- deduplicated: bool (solo respuesta)

UploadToken (efímero, no expuesto en la API)
- jti: string (PK)
- work_request_id: int (FK)
- max_files: int
- files_used: int
- expires_at: datetime
```

### 5.4 APIs / Contratos

#### Endpoint: `POST /api/homes/{id}/workrequests/{work_request_id}/upload-link`

Auth: sesión o API key global (el MCP usa su API key y `X-P40LA-Source: mcp`).

**Request**:
```json
{
  "count": 10,
  "ttl_minutes": 10
}
```

**Response 200**:
```json
{
  "token": "demo-token.v1.payload.signature",
  "path": "/api/work-request-uploads",
  "expires_at": "2026-11-27T15:10:00Z",
  "max_bytes": 26214400,
  "max_request_bytes": 263192576,
  "max_files": 10,
  "accepted_mime": ["image/jpeg", "image/png", "image/webp", "image/heic", "video/mp4", "video/quicktime", "video/webm"],
  "attachment_count": 2,
  "remaining_slots": 8,
  "work_request_id": 10,
  "home_id": 3
}
```

Errores: `invalid_request` (count/ttl fuera de rango, 400), `home_not_found` (404), `work_request_not_found` (404), `invalid_status` (409, no `reported`), `attachment_limit` (409, sin cupo), `upload_link_disabled` (503, `UPLOAD_TOKEN_SECRET` vacío). El token del ejemplo es sintético; el real nunca se commitea.

#### Endpoint: `POST /api/work-request-uploads`

Auth: `Authorization: Bearer <token>`; **no** acepta sesión ni API key global. `Content-Type: multipart/form-data`.

**Request** (multipart):
- 1..`max_files` partes con nombre `file`.
- El `media_type` (`image`/`video`) se **infiere por magic bytes**; no se confía en el `Content-Type` de la parte. Un archivo que no coincide con la allowlist se rechaza con `invalid_file`.

Ejemplo (con `UPLOAD_URL` y `TOKEN` tomados de `upload_link`; el path local es ilustrativo y depende del entorno del agente):
```bash
curl -sS -X POST "$UPLOAD_URL" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/demo-foto-1.jpg" \
  -F "file=@/tmp/demo-video-1.mp4"
```

**Response 200**:
```json
{
  "home_id": 3,
  "work_request_id": 10,
  "added": 1,
  "duplicates": 1,
  "rejected": 0,
  "attachment_count": 4,
  "remaining_slots": 6,
  "results": [
    {
      "file_name": "foto-demo-1.jpg",
      "status": "added",
      "attachment": {
        "id": 21,
        "work_request_id": 10,
        "media_type": "image",
        "file_name": "foto-demo-1.jpg",
        "mime_type": "image/jpeg",
        "size_bytes": 1024,
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "created_at": "2026-11-27T15:00:00Z"
      }
    },
    {
      "file_name": "foto-demo-1.jpg",
      "status": "duplicate",
      "attachment": { "id": 21, "file_name": "foto-demo-1.jpg", "deduplicated": true }
    }
  ]
}
```

Errores: `invalid_token`/`expired_token` (401), `work_request_not_found` (404), `invalid_status` (409), `attachment_limit` (409 si no hay cupo al iniciar), `upload_token_exhausted` (409 si el token ya usó `max_files`), `request_too_large` (413), `unsupported_media` (415 si el Content-Type no es multipart), `too_many_requests` (429 con `Retry-After`). Un archivo individual inválido o sin cupo no aborta el lote: se informa en `results[]` con `status=rejected` y `error`/`message`. El handler setea `Cache-Control: no-store` y `X-Content-Type-Options: nosniff`.

#### Endpoint: `POST /api/homes/{id}/workrequests/{work_request_id}/attachments/evidence`

Auth: sesión o API key global. `Content-Type: multipart/form-data` con **una** parte `file`. Es la vía que usa `home_work_request_attachment_add` para archivos pequeños; calcula `sha256`, deduplica y solo acepta reportes `reported`. La web sigue usando `POST .../attachments` (sin hash ni dedup).

**Response 201 (nuevo)**:
```json
{
  "attachment": {
    "id": 21,
    "work_request_id": 10,
    "media_type": "image",
    "file_name": "foto-demo-1.jpg",
    "mime_type": "image/jpeg",
    "size_bytes": 1024,
    "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
    "created_at": "2026-11-27T15:00:00Z"
  },
  "deduplicated": false
}
```

**Response 200 (duplicado)**: mismo cuerpo con `deduplicated=true` y el adjunto existente. Errores: `invalid_file` (400), `attachment_limit` (409), `invalid_status` (409), `request_too_large` (413), `home_not_found`/`work_request_not_found` (404).

#### Herramientas MCP (ampliación)

##### `home_work_request_attachment_list`
Entrada: `home_id` (req), `work_request_id` (req).
Salida: `home_id`, `work_request_id`, `attachment_count`, `max_attachments`, `max_bytes`, `remaining_slots`, `attachments[]`.
Lectura: 1 `GET /api/homes/{id}/workrequests/{wid}`. No depende de la compuerta. Los adjuntos anteriores a la migración `0063` devuelven `sha256: ""` (no se backfillean; REQ-027).

##### `home_work_request_attachment_upload_link`
Entrada: `home_id`, `work_request_id`, `count?` (1..10, default 10), `ttl_minutes?` (1..30, default 10), `dry_run?` (default true).
Salida: `status` (`planned`/`created`), `dry_run`, `upload_url`, `token` (`null` en `planned`), `expires_at`, `max_bytes`, `max_request_bytes`, `max_files`, `accepted_mime[]`, `attachments_remaining`, `report_url`, `reachability`, `curl_hint`.
Llamadas: `planned` = 1 `GET` (para cupo/estado); `created` = 1 `POST .../upload-link`.
Anotaciones: `DestructiveHint=false`, `IdempotentHint=false`, `OpenWorldHint=true`.

##### `home_work_request_attachment_add`
Entrada: `home_id`, `work_request_id`, `filename`, `mime?`, `media_type?` (`image`/`video`; default derivado del prefijo de `mime`, o `image`), `data_base64`, `sha256?`, `dry_run?`.
Salida: `status` (`planned`/`added`/`duplicate`), `dry_run`, `attachment`, `attachment_count`.
Límite: contenido decodificado ≤ **256 KiB**; si supera, `invalid_request` con hint a `upload_link`. El backend vuelve a validar la firma real; un `media_type` que no coincide con los bytes se rechaza con `invalid_file`.
Llamadas: `planned` = 1 `GET` (para estado/cupo, sin POST); `added`/`duplicate` = 1 `GET` + 1 `POST .../attachments/evidence` (multipart, API key). El GET previo permite rechazar un reporte `converted`/`dismissed` antes de escribir.
Anotaciones: `DestructiveHint=false`, `IdempotentHint=true`, `OpenWorldHint=false`.

##### `home_work_request_attachment_delete`
Entrada: `home_id`, `work_request_id`, `attachment_id`, `dry_run?`.
Salida: `status` (`planned`/`deleted`/`unchanged`), `dry_run`, `attachment_id`, `deleted` (bool), `attachment_count`.
Llamadas: `GET` previo + `DELETE .../attachments/{attachment_id}` si `dry_run=false` y el adjunto existe.
Idempotencia: si el adjunto ya no existe, devuelve `status=unchanged`, `deleted=false` (no es error); un `attachment_id` inválido (≤0) devuelve `invalid_request`.
Anotaciones: `DestructiveHint=true`, `IdempotentHint=true`, `OpenWorldHint=false`.

#### Cambio de contrato en `home_work_request_create`
- Entrada nueva opcional: `client_request_id` (string, 1..64, `[A-Za-z0-9._:-]`); un valor inválido devuelve `invalid_client_request_id` sin escribir.
- Salida: `status` puede ser `existing`; agrega `deduplicated` (bool).
- Backend: si el `client_request_id` ya existe en la misma vivienda, devuelve **HTTP 200** con el reporte existente y `deduplicated=true`; si es nuevo, **HTTP 201** como hoy.
- `dry_run=true` no consulta duplicados (no llama a la API); el `status=existing` solo se conoce al escribir.
- Sin `client_request_id`, el comportamiento y la salida son compatibles con la versión liberada.

### 5.5 Especificación del token

```
v1.<payload_b64url>.<sig_b64url>
payload = {"hid":3,"wid":10,"exp":1795737600,"jti":"<16 bytes hex>","max_files":4}
sig     = HMAC-SHA256(UPLOAD_TOKEN_SECRET, "v1." + payload_b64url)
```

- `UPLOAD_TOKEN_SECRET`: mínimo 32 caracteres; se compara en tiempo constante.
- `exp`: epoch UTC; default `now + UPLOAD_LINK_TTL` (10 min), máximo 30 min.
- Al emitir: se valida home/reporte/estado/cupo, se genera `jti` aleatorio (16 bytes) y se inserta la fila `work_request_upload_tokens(jti, work_request_id, max_files, files_used=0, expires_at)`. Se limpian oportunistamente las filas expiradas hace más de 24 h.
- Validación por request: versión, firma, `exp > now`, `hid/wid > 0`, fila existente, `row.work_request_id == wid`, `row.expires_at > now`, y luego `home exists` + `work_request exists` + `home_id` coincide + `status == reported`.
- Cupo del token: por cada archivo **nuevo** se incrementa el contador con
  `UPDATE ... SET files_used = files_used + 1 WHERE jti = ? AND files_used < max_files`.
  El incremento y la inserción del adjunto van en la **misma transacción** (`BEGIN IMMEDIATE`) que el chequeo del cupo global; si el cupo global está lleno o la inserción falla, la transacción revierte el incremento. Si el `UPDATE` no afecta filas, ese archivo se rechaza con `upload_token_exhausted` y el resto del lote sigue procesándose. Los duplicados y los rechazados **no** consumen cupo. Si al iniciar el request `files_used >= max_files`, se responde `409 upload_token_exhausted` sin leer el body.
- El token no autoriza listar, descargar, borrar ni editar: solo agregar archivos a ese reporte.
- El token no se escribe en logs, auditoría ni URL. El `curl_hint` usa `<TOKEN>` como placeholder.
- Rotar el secreto invalida todos los links emitidos; borrar la fila del `jti` revoca un link puntual. No requiere migración.

### 5.6 Validación de archivos y límites

| Regla | Valor |
|---|---|
| Adjuntos por reporte | 10 (existente) |
| Tamaño por archivo | 25 MB (existente) |
| Archivos por link | 1..10 (`count`) |
| Tamaño total por request | `25 MB × max_files + 1 MB` (tope de `MaxBytesReader`) |
| Imágenes | JPEG, PNG, WebP, HEIC (magic bytes) |
| Videos | MP4, QuickTime, WebM (magic bytes) |
| Detección | Firma real; se ignora la extensión y el `Content-Type` del cliente |
| Nombre | Se sanitiza; sin rutas, sin `..`, máximo 180 caracteres |
| Vacío | Rechazado (`invalid_file`) |
| Duplicado | `sha256` idéntico en el mismo reporte, en vías MCP → `duplicate`; la web no deduplica |
| Video | Se almacena como evidencia; **no** se analiza ni se describe (Claude no lo recibe como entrada) |

### 5.7 Flujos

**Flujo feliz (sandbox con red y archivos)**
1. El agente crea el reporte con `client_request_id` estable y `dry_run=false` (con la compuerta activa).
2. Pide `upload_link` con `count=N` y `dry_run=false` (con la compuerta activa); si el usuario solo quiere previsualizar, usa `dry_run=true`.
3. Ejecuta `curl -F` por cada archivo (o un solo request con N partes) contra `upload_url` con el token.
4. Llama `attachment_list` y confirma `attachment_count` y `remaining_slots`.
5. Si el usuario corrige título/prioridad, usa `update` (no crea otro reporte).

**Reintento**
- Crear de nuevo con el mismo `client_request_id` → `existing`, mismo `id`.
- Subir el mismo archivo → `duplicate`, mismo `attachment_id`.
- Repetir `upload_link` → nuevo token, mismo reporte.

**Fallback (sandbox sin red o host inalcanzable)**
- `upload_link` devuelve `reachability=private` y `report_url`.
- El agente explica que no pudo subir los archivos y entrega `report_url` para que el usuario los cargue desde el detalle del reporte (botón «Adjuntar», un archivo por vez).
- El reporte ya creado no se duplica.

**Error de token**
- Expirado/alterado → `401`; no se escribe nada.
- Reporte convertido entre el mint y la carga → `409 invalid_status`; no se escribe nada.

### 5.8 Dependencias

- **Internas**: `internal/modules/home/workrequest` (token, endpoints, migración), `internal/config`, `internal/modules/all`, `internal/api/routes.go` (indirecto por el registro del módulo), `internal/mcp/api`, `internal/mcp/config`, `internal/mcp/tools/workrequest`, `internal/mcp/tools/all`.
- **Backend**: endpoints existentes de `home.workrequest` (SPEC-164), sin cambios de contrato para la web.
- **Externas**: ninguna nueva.

---

## 6. Impacto en tools MCP

### 6.1 Veredicto

**Ninguna tool existente se rompe funcionalmente.** Se agregan cuatro tools nuevas sobre el módulo `home.workrequest` y se amplía el contrato de `home_work_request_create` con un campo opcional (`client_request_id`) y un `status` adicional (`existing`) compatibles hacia atrás. El único cambio compartido es `api.FullAPI` y el stub `contractAPI` de `internal/mcp/tools/all/contract_test.go`, que se actualizan para compilar; además se regenera `contracts.json` y se agregan entradas al manifiesto.

### 6.2 Tools nuevas

`home_work_request_attachment_list`, `home_work_request_attachment_upload_link`, `home_work_request_attachment_add`, `home_work_request_attachment_delete`.

El catálogo pasa de **59** a **63** tools (32 de lectura y 31 de escritura).

### 6.3 Tools existentes afectadas

- `home_work_request_create`: cambio **compatible** de contrato (campo opcional nuevo y `status=existing`). Requiere regenerar `contracts.json` y actualizar su documentación.
- `home_work_request_get`, `_list`, `_update`, `_convert`, `_status`: sin cambios en `tools/list` (sus handlers devuelven `any`, sin `output_schema`). `_get` ahora puede devolver `sha256` y `deduplicated` en los adjuntos; son campos aditivos del JSON de salida. `_list` sigue devolviendo solo `attachment_count`.
- Las **59 tools existentes** aparecen en el análisis de impacto por recompilación compartida (`internal/mcp/api/`, `internal/mcp/tools/all/`, `internal/modules/all/`, `internal/config/`, `migrations/` están en `shared.affects_all`); el impacto funcional real es nulo salvo el create.

Listado de tools existentes: `bills_list`, `bills_generate`, `bills_pay`, `homes_create`, `homes_update`, `institutions_create`, `banks_create`, `services_create`, `entities_search`, `autos_list`, `autos_create`, `autos_update`, `autos_maintenance_register`, `autos_inspection_register`, `autos_insurance_register`, `bank_accounts_search`, `bank_accounts_create`, `bank_accounts_update`, `credit_cards_search`, `credit_cards_create`, `credit_cards_update`, `pension_month_summary`, `pension_catalogs_list`, `rental_leases_list`, `rental_lease_get`, `rental_home_status`, `rental_portfolio_summary`, `debt_bills_list`, `credit_card_statements_list`, `credit_card_statements_get`, `home_maintenance_list`, `home_maintenance_labor_register`, `home_maintenance_supplies_register`, `home_maintenance_create`, `home_maintenance_item_move`, `home_maintenance_get`, `home_maintenance_update`, `home_maintenance_item_update`, `home_maintenance_item_delete`, `autos_inspection_alerts`, `bills_by_month`, `bills_summary`, `bills_projection`, `services_get`, `automation_accounts_list`, `automation_runs_list`, `alerts_list`, `settings_status`, `budget_categories_list`, `budget_accounts_list`, `budget_transactions_create`, `budget_transactions_list`, `budget_month_view`, `home_work_request_list`, `home_work_request_get`, `home_work_request_create`, `home_work_request_update`, `home_work_request_convert`, `home_work_request_status`.

### 6.4 Manifiesto y contrato

- `docs/mcp/tools/impact.json`: cuatro entradas nuevas y actualización de la entrada de `home_work_request_create` (dependencias/endpoints si cambian).
  - `attachment_list`: `GET /api/homes/{id}/workrequests/{work_request_id}`.
  - `attachment_upload_link`: `POST /api/homes/{id}/workrequests/{work_request_id}/upload-link` y `POST /api/work-request-uploads`.
  - `attachment_add`: `GET /api/homes/{id}/workrequests/{work_request_id}` y `POST .../attachments/evidence`.
  - `attachment_delete`: `GET .../{work_request_id}` y `DELETE .../attachments/{attachment_id}`.
- `docs/mcp/tools/contracts.json`: regenerado con `TestToolContracts -update-contracts`.
- `TestMCPBackendContract`: agrega el flujo crear (idempotente) → upload-link → upload multipart con token → attachment_list → delete, y verifica que el MCP envíe `X-P40LA-Source: mcp`, los códigos de error y que el token no se filtre en la respuesta del listado. El módulo `workrequest` hoy **no consume** `X-P40LA-Source`; el test verifica el header en la request, no una auditoría inexistente. Si se quiere auditar el origen de la evidencia, es un requerimiento aparte.
- `docs/mcp/README.md`, `docs/mcp/tools/README.md` y `docs/mcp/tools/impact.md`: actualizar los conteos (59 → 63), las tablas y las anotaciones.
- `deploy/mcp/mcp.env.example`: agregar `P40LA_UPLOAD_PUBLIC_URL` si se documenta como variable del MCP.

**Validación subagente**: explore — sesión `ses_ddfc5e210ffeLyqICFJTkdMtZp` — veredicto: **ninguna tool existente se rompe**; el cambio compartido es `api.FullAPI`, `contractAPI` y el golden; el módulo ya tiene carga multipart y validación por magic bytes, pero no token temporal ni deduplicación; el body de `/mcp` (1 MiB) limita la vía inline; `TestModuleInfoAndRoutes` y `Info().Migrations` deben actualizarse con `0063`. general — sesión `ses_ddfc5e20dffeSGZ8w5l3vC1gHk` — veredicto: los argumentos de tool son JSON; el binario tipado de MCP es servidor→cliente; la extensión oficial exige Directory/host aprobado; el sandbox no alcanza `*.ts.net`; Funnel puede montar un prefijo pero expone todo lo que el backend sirva bajo él. La decisión de no exponer el backend queda confirmada por ambas investigaciones.

---

## 7. Criterios de Aceptación

### 7.1 Originales (liberados)

- [ ] `CA-MCP-WR-001`: `home_work_request_list` filtra por estado/categoría y devuelve `pending_count`, `truncated` y `attachment_count`.
- [ ] `CA-MCP-WR-002`: `home_work_request_get` devuelve el detalle con metadatos de evidencia y el vínculo a la reparación.
- [ ] `CA-MCP-WR-003`: `home_work_request_create` en `dry_run` no escribe y en modo escritura crea `reported` sin costos.
- [ ] `CA-MCP-WR-004`: `home_work_request_update` hace merge parcial y devuelve `unchanged` cuando no hay cambios.
- [ ] `CA-MCP-WR-005`: `home_work_request_convert` crea la reparación vacía, enlaza `maintenance_id` y rechaza una segunda conversión con `already_converted`.
- [ ] `CA-MCP-WR-006`: `home_work_request_status` descarta y reactiva; rechaza `converted`.
- [ ] `CA-MCP-WR-007`: las escrituras sin compuerta devuelven `writes_disabled`; `dry_run` sigue disponible.
- [ ] `CA-MCP-WR-008`: `TestToolContracts`, `TestMCPBackendContract` y `go test ./internal/mcp/...` verdes.
- [ ] `CA-MCP-WR-DOCS`: `docs/mcp/tools/home-work-requests.md` y los índices actualizados.
- [ ] `CA-MCP-WR-IMPACT`: `check-mcp-impact.sh --staged --gate` verde.
- [ ] `CA-MCP-WR-STACK`: sin dependencias nuevas ni cambios en `go.mod`.
- [ ] `CA-MCP-WR-SYNTHETIC`: tests con `Casa Demo`, `demo-*`, sin PII.
- [ ] `CA-MCP-WR-RELEASE`: `mcp-v0.6.4` publicado multi-arch; issue cerrado.

### 7.2 Ampliación: funcionales

- [ ] `CA-MCP-WR-EV-001` (idempotencia de alta): dos `home_work_request_create` con el mismo `client_request_id` y `home_id` devuelven el mismo `work_request.id`; la segunda respuesta es `status=existing`, `deduplicated=true`, y la DB tiene **un** registro.
- [ ] `CA-MCP-WR-EV-002` (link de carga): `upload_link` con `dry_run=true` no emite token (`token=null`) y devuelve `status=planned`; con `dry_run=false` y compuerta activa devuelve `status=created`, `token` no vacío, `expires_at` ≤ 30 min, `max_files` 1..10 y `reachability` correcto.
- [ ] `CA-MCP-WR-EV-003` (carga multipart): un `curl -F` con 3 archivos válidos contra `upload_url` devuelve `added=3`; `attachment_list` muestra `attachment_count` incrementado en 3 y los `sha256` correspondientes.
- [ ] `CA-MCP-WR-EV-004` (deduplicación): subir dos veces el mismo archivo al mismo reporte por la vía MCP (token o inline) devuelve `duplicate` en el segundo intento, sin nueva fila ni archivo; `attachment_count` no cambia. La web conserva su comportamiento (puede duplicar) y no se usa para este criterio.
- [ ] `CA-MCP-WR-EV-005` (token inválido): token alterado, expirado, de otro `work_request_id` o con `jti` inexistente devuelve `401`/`invalid_token` o `expired_token` y no escribe ni crea archivos.
- [ ] `CA-MCP-WR-EV-005b` (cupo del token): un token con `max_files=2` permite 2 archivos; un tercer archivo en un request posterior devuelve `409 upload_token_exhausted`, y dentro del mismo lote sale `rejected`/`upload_token_exhausted`; los duplicados no consumen cupo y los rechazados no lo consumen.
- [ ] `CA-MCP-WR-EV-006` (límites): con 10 adjuntos existentes, un nuevo link devuelve `attachment_limit` (409); dentro de un lote, el archivo que excede el cupo sale `rejected`/`attachment_limit` y los anteriores se conservan; un archivo > 25 MB devuelve `request_too_large` (413); un `.txt` o un archivo con MIME falso devuelve `invalid_file`/`rejected`; no queda fila ni archivo parcial del archivo rechazado.
- [ ] `CA-MCP-WR-EV-007` (estado): `upload_link` y la carga rechazan un reporte `dismissed` o `converted` con `invalid_status`; el borrado rechaza `converted`; la web conserva su comportamiento.
- [ ] `CA-MCP-WR-EV-008` (compuerta): `upload_link`, `attachment_add` y `attachment_delete` con `dry_run=false` y sin `P40LA_MCP_ALLOW_WRITES` devuelven `writes_disabled`; `attachment_list` funciona siempre.
- [ ] `CA-MCP-WR-EV-009` (borrado): `attachment_delete` con `dry_run=true` no borra; con `dry_run=false` elimina la fila y el archivo y devuelve el `attachment_count` actualizado; repetir el borrado devuelve `status=unchanged`, `deleted=false` y no falla.
- [ ] `CA-MCP-WR-EV-010` (inline): `attachment_add` con un JPEG demo de ≤ 256 KiB devuelve `added`; con > 256 KiB devuelve `invalid_request` con hint a `upload_link`; con el mismo `sha256` devuelve `duplicate`.
- [ ] `CA-MCP-WR-EV-011` (flujo completo): crear → upload_link → subir N archivos → attachment_list → corregir título/prioridad con `update` deja **un** reporte y **N** adjuntos; no se crean registros nuevos por la corrección.
- [ ] `CA-MCP-WR-EV-012` (fallback): con `reachability=private`, la salida incluye `report_url` y la documentación indica cargarlos desde el detalle del reporte con el botón «Adjuntar»; el agente no afirma haber subido lo que no subió.
- [ ] `CA-MCP-WR-EV-013` (video): un MP4/QuickTime/WebM demo se almacena y se lista como `media_type=video`; la documentación aclara que el agente no lo analiza ni lo describe porque Claude no recibe video como entrada.

### 7.3 Ampliación: criterios del usuario (trazabilidad)

- [ ] `CA-USER-EV-001` (un mensaje, N adjuntos): con un entorno que pueda ejecutar `curl` contra `upload_url` y leer los archivos, el agente crea **un** reporte y sube **N** archivos (1..10) sin que el usuario entre a la web; si el entorno no puede, el agente entrega `report_url` y lo dice.
- [ ] `CA-USER-EV-002` (corrección sin duplicar): corregir `title` o `priority` con `home_work_request_update` no crea un reporte nuevo ni toca los adjuntos.
- [ ] `CA-USER-EV-003` (reintentos): repetir el alta con el mismo `client_request_id` y el mismo archivo con el mismo hash no duplica reportes ni adjuntos.
- [ ] `CA-USER-EV-004` (compuertas): todas las escrituras siguen con `dry_run=true` por defecto y requieren `P40LA_MCP_ALLOW_WRITES=true` (más `P40LA_MCP_REMOTE_WRITES=true` en HTTP).

### 7.4 Ampliación: no funcionales y seguridad

- [ ] `CA-MCP-WR-EV-MIGRATION`: `0063` aplica y revierte en una DB temporal (`/tmp/*.db`) sin pérdida de reportes/adjuntos; `up` y `down` son idempotentes en el orden correcto.
- [ ] `CA-MCP-WR-EV-PRODCOPY`: si se valida con datos, se usa copia en `/tmp`; no se escribe en `data/app.db` ni en producción.
- [ ] `CA-MCP-WR-EV-SECRETS`: el token no aparece en logs, URL, auditoría MCP, mensajes de error ni en `curl_hint`; `git diff --cached` no contiene secretos ni binarios.
- [ ] `CA-MCP-WR-EV-SYNTHETIC`: todos los ejemplos usan `Casa Demo`, `demo-*`, `demo-token`, `00000000-0000-4000-8000-000000000000`; no hay fotos, nombres ni direcciones reales.
- [ ] `CA-MCP-WR-EV-DOCS`: `home-work-requests.md`, `security.md`, `configuration.md`, `architecture.md`, `tools/README.md`, `docs/mcp/README.md`, `tools/impact.md`, `deploy/mcp/mcp.env.example` y `docs/ihost.md` actualizados; catálogo 63 tools.
- [ ] `CA-MCP-WR-EV-DOCS-SUBAGENT`: la documentación de `docs/mcp/` fue redactada por un subagente y revisada por el agente principal.
- [ ] `CA-MCP-WR-EV-IMPACT`: `./scripts/check-mcp-impact.sh --staged --gate` verde con la spec y el código.
- [ ] `CA-MCP-WR-EV-IMPACT-SUBAGENT`: el veredicto del subagente está citado en §6 con su sesión.
- [ ] `CA-MCP-WR-EV-CONTRACT`: `TestToolContracts` (golden regenerado y revisado) y `TestMCPBackendContract` (flujo de evidencia) verdes.
- [ ] `CA-MCP-WR-EV-STACK`: `go.mod` sin dependencias nuevas; solo stdlib.
- [ ] `CA-MCP-WR-EV-LAYERS`: no hay SQL en `service.go` ni en `handlers.go`; la validación de token/estado y la deduplicación viven en `service.go`, y el SQL en `storage.go`.
- [ ] `CA-MCP-WR-EV-IHOST`: el MCP no abre SQLite ni `DATA_DIR`; el backend procesa la carga de a un archivo y el pico de memoria no supera ~25 MB por archivo.
- [ ] `CA-MCP-WR-EV-TESTS`: `go build ./...`, `go test ./...`, `go test ./internal/mcp/...`, `go test ./internal/api -run TestMCPBackendContract` y `./scripts/check-server.sh` verdes.
- [ ] `CA-MCP-WR-EV-NO-PUBLIC`: no se agrega Funnel, proxy ni regla de firewall; el endpoint queda en el origen actual y la spec documenta que publicarlo requiere aprobación separada.
- [ ] `CA-MCP-WR-EV-MAP`: las cuatro tools y el create actualizado están en `impact.json` y `contracts.json`; `--check-map` verde.

### 7.5 Testing

- **Unit tests**: token (firma, expiración, scope, tamper), `client_request_id` (válido/inválido/duplicado), SHA-256 (dedup, colisión), magic bytes (todos los MIME permitidos y un `.txt`), sanitización de nombres, límites (10/25 MB/256 KiB), rate limit.
- **Integration tests**: handler de upload-link (auth, estado, cupo), handler público (token, multipart, lote mixto, 401/409/413/415/429), persistencia en DB temporal y archivos en `t.TempDir()`.
- **Contract tests**: `TestToolContracts` + `TestMCPBackendContract` con el flujo completo.
- **E2E/manual**: usuario de prueba, `Casa Demo`, crear reporte, `curl -F` con 2 imágenes y 1 video demo, verificar en la web que se ven y que el borrado funciona. Probar el fallback abriendo `report_url`.

---

## 8. Plan de Implementación

### 8.1 Fases originales (liberadas)

| Fase | Descripción |
|---|---|
| 1 | Cliente API y DTOs (`home_work_request.go`, `FullAPI`) |
| 2 | Toolset `workrequest` (6 tools) y registro |
| 3 | Tests unitarios, contrato y backend real |
| 4 | Manifiesto, `contracts.json` y documentación (subagente) |
| 5 | Release `mcp-v0.6.4` y verificación de imagen |

### 8.2 Fases de la ampliación

| Fase | Descripción | Dependencias |
|---|---|---|
| 6 | Migración `0063`, modelo y storage (`client_request_id`, `sha256`, búsquedas y unicidad) | — |
| 7 | Servicio: create idempotente, dedup de adjuntos, `AddAttachmentDetected`, token (mint/parse) y estado | Fase 6 |
| 8 | Handlers: `upload-link`, `POST /api/work-request-uploads`, `.../attachments/evidence`, rate limit, rutas y config | Fase 7 |
| 9 | MCP: config `P40LA_UPLOAD_PUBLIC_URL`, cliente API/multipart, 4 tools y `client_request_id` en create | Fase 8 |
| 10 | Tests: unitarios, handlers, contrato backend, MCP y manual con `curl` | Fases 6-9 |
| 11 | Manifiesto, `contracts.json`, documentación (subagente) y gates | Fase 10 |

**No incluye**: tarea del Developer, merge, push a `main` ni release. La spec queda en `draft` para revisión según el modo «solo spec draft».

### 8.3 Milestones

1. **MVP**: create idempotente + upload-link + upload multipart + list + dedup.
2. **V1.0**: delete, `attachment_add` inline, rate limit, docs y contrato completos.
3. **Futuro**: chunks reanudables (REQ-026), `jti` de un solo uso (REQ-028), backfill de hashes (REQ-027).

---

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Exponer el backend por Funnel “para que funcione” | Media | Alto | `CA-MCP-WR-EV-NO-PUBLIC`: no se configura Funnel/proxy; se documenta la decisión separada y el fallback. Si se aprueba, montar un prefijo dedicado con `--set-path` que solo enrute la carga |
| Token filtrado en logs o en la respuesta al modelo | Media | Alto | Header, no URL; no se loguea; TTL corto; `curl_hint` con placeholder; rotación del secreto |
| Replay del token dentro del TTL | Media | Bajo | Fila por `jti` con `max_files`/`files_used`; TTL 10 min; rate limit; cupo global de 10 |
| El sandbox no puede alcanzar el host (`*.ts.net` fuera de la allowlist) | Alta | Medio | `reachability` + `report_url` + fallback web; la documentación no promete que `curl` funcione desde claude.ai |
| El cliente MCP no puede pasar bytes como argumento | Alta | Medio | La vía principal es el link HTTP; `attachment_add` solo para hosts que inyectan bytes |
| Extensión oficial de archivos no disponible para el conector | Alta | Bajo | Fuera de alcance (ADR-012); se documenta como evolución si se publica en el Directory |
| El usuario espera que el agente “vea” un video | Media | Medio | ADR-011: el video se almacena, no se analiza; se pide una imagen/frame o una descripción |
| Duplicados por reintentos | Alta | Medio | `client_request_id` + índice único; `sha256` + índice único; respuestas `existing`/`duplicate` |
| Carga de archivos maliciosos | Media | Alto | Magic bytes, allowlist de MIME, sanitización de nombre, `X-Content-Type-Options: nosniff`, sin ejecución; tamaño máximo |
| EXIF/GPS en fotos de vivienda | Media | Medio | REQ-029 fuera de alcance; se documenta que los bytes se guardan tal cual y se recomienda revisar las fotos; el volumen es privado y el acceso autenticado |
| Carrera de cupo (11.º adjunto concurrente) | Baja | Medio | Chequeo de cupo + inserción en una transacción `BEGIN IMMEDIATE`; índice único de hash para duplicados concurrentes |
| El dashboard no tiene drag & drop y acepta un archivo por vez | Alta | Bajo | El fallback documenta el botón «Adjuntar» y la carga de a uno; la carga masiva automática es la vía del link |
| Consumo de memoria/almacenamiento en el iHost | Media | Medio | Streaming por archivo, 25 MB por archivo, 10 por reporte; no se bufferean los N archivos |
| Migración fallida en producción | Baja | Alto | `up`/`down` probados en `/tmp`; copia de producción; migración aditiva sin `DROP TABLE` |
| Incompatibilidad del cliente MCP con adjuntos | Alta | Medio | La vía principal es HTTP (`curl`), no argumentos binarios; `attachment_add` es opcional y acotada |
| Sandbox sin acceso al host | Alta | Medio | `reachability` + `report_url` + fallback web documentado |
| Regenerar `contracts.json` sin revisar | Baja | Medio | `TEST-MCP-CONTRACT`: diff revisado y declarado en §6 |

---

## 10. Notas y Referencias

- SPEC-164: módulo `home.workrequest` (endpoints y semántica de estados).
- Issue de la ampliación: #169. Issue original de SPEC-165: #167.
- SPEC-153: manifiesto de impacto y gate MCP.
- SPEC-159: patrón de tools de detalle/edición de reparaciones.
- SPEC-147 / SPEC-148: transporte HTTP, OAuth y nodo Tailscale del MCP.
- `docs/mcp/tools/home-maintenance.md`: referencia de estilo y documentación.
- `docs/mcp/security.md`: dos credenciales, compuerta de escritura, datos y PII.
- `docs/mcp/architecture.md`: Serve vs Funnel, listeners y límites.
- `docs/rules/`: `ARCH-*`, `DATA-*`, `SEC-*`, `TEST-*`, `DELIV-*` citadas en §2.
- Investigación externa (2026-11-27; conclusiones aplicadas en §4.4 y §4.5):
  - MCP Tools (rev. 2026-07-28): https://modelcontextprotocol.io/specification/2026-07-28/server/tools
  - Extensión de archivos MCP (draft): https://github.com/modelcontextprotocol/experimental-ext-file-uploads/blob/main/specification/draft/file-uploads.md
  - Claude — File transfers (beta, requisitos de Directory/host): https://claude.com/docs/connectors/building/files
  - Claude — Code execution, networking/seguridad: https://platform.claude.com/docs/en/agents-and-tools/tool-use/code-execution-tool#networking-and-security
  - Claude — Create and edit files (egress por plan): https://support.claude.com/en/articles/12111783-create-and-edit-files-with-claude
  - Claude — Upload files (límites de chat): https://support.claude.com/en/articles/8241126-upload-files-to-claude
  - Claude — Vision request limits: https://platform.claude.com/docs/en/build-with-claude/vision#request-limits
  - Tailscale Funnel: https://tailscale.com/kb/1223/funnel y https://tailscale.com/docs/reference/tailscale-cli/funnel
  - OWASP File Upload Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html
  - OWASP MCP Security Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/MCP_Security_Cheat_Sheet.html
- Incertidumbres no verificadas: alcanzabilidad real de `*.ts.net` desde el sandbox con “All domains” (Team/Enterprise), soporte de video como entrada, y render de MCP Apps en móvil. La spec no depende de ninguna de ellas.

---

## 11. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-11-27 | opencode | Creación inicial para agregar tools MCP de reportes de vivienda |
| 2026-11-27 | opencode | Implementación: cliente `HomeWorkRequestAPI`, toolset `workrequest` con 6 tools, registro, tests, manifiesto y `contracts.json`; documentación delegada a subagente |
| 2026-11-27 | opencode | Verificación local: `go build ./...`, `go test ./...`, `check-mcp-docs` y `check-mcp-impact --staged --gate` verdes; preparada para release `mcp-v0.6.4` |
| 2026-11-27 | opencode | Released: app `v0.5.24` y MCP `mcp-v0.6.4` multi-arch; workflows `46101201103` y `46101203390`; merge `66885698`; issue #167 cerrado |
| 2026-11-27 | opencode | Ampliación (draft): evidencia por MCP con `attachment_list`, `upload_link`, `attachment_add` y `attachment_delete`; endpoint multipart con token HMAC de corta vida y cupo por token; idempotencia `client_request_id` y dedup `sha256`; migración `0063`; sin exposición pública nueva. Issue de ampliación #169. Investigación con subagentes `explore` y `general`; se corrigieron los supuestos sobre adjuntos del cliente MCP, sandbox y video |
