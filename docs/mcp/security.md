---
title: "Seguridad"
sidebar_position: 5
description: "API key global, OAuth, compuerta de escritura, datos y protocolo de incidentes"
---

# Seguridad

## Principios

1. **Dos credenciales separadas**: OAuth 2.1 protege la entrada remota; la API key global protege la salida hacia la API del sistema.
2. **Sin contraseña del dashboard en el MCP**: el agente usa una API key, no las credenciales del usuario. La contraseña del MCP remoto es otra credencial, solo para autorizar clientes OAuth.
3. **Menor exposición**: las credenciales se guardan hasheadas, se muestran una vez y se pueden revocar.
4. **Confirmación humana**: toda tool de escritura exige confirmación del usuario antes de ejecutar. Las de facturas, autos, presupuesto, reparaciones y reportes de vivienda previsualizan con `dry_run`; las de banca no tienen `dry_run` y el agente debe mostrar el resumen (banco y moneda resueltos, cuerpo y duplicados) antes de escribir. Las altas de entidades tampoco tienen `dry_run`: se muestra el cuerpo y si ya existe un duplicado. `homes_update` es una edición directa: se muestra el hogar resuelto y los campos que cambian.
5. **Datos mínimos**: los logs no incluyen montos, tokens, cookies ni payloads.
6. **Lectura por defecto en remoto**: publicar el MCP en Internet no habilita escrituras.

## Dos superficies, dos credenciales

| Superficie | Quién la usa | Credencial | Dónde se valida |
|---|---|---|---|
| `/mcp` (HTTPS público) | claude.ai / cliente remoto | Bearer token OAuth | `internal/mcp/server/oauth` |
| API de p40la-ihost (`:8088`) | el binario MCP | API key global `p40la_...` | `SessionOrAPIKeyMiddleware` |

Un token OAuth **no** sirve contra la API del backend y la API key **no** sirve contra `/mcp`. Si se filtra una, la otra sigue siendo válida: hay que rotar la comprometida y revisar si la otra pudo obtenerse por el mismo canal.

## Superficie remota

Con `P40LA_MCP_TRANSPORT=http` y Funnel activo, Internet puede alcanzar:

| Ruta | Autenticación | Notas |
|---|---|---|
| `POST /mcp` | Bearer OAuth obligatorio | Sin token: `401` + `WWW-Authenticate` |
| `/.well-known/*` | Ninguna | Discovery público por diseño |
| `POST /register` | Ninguna, 10/h por IP | DCR limitado a redirect URIs allowlisted |
| `GET\|POST /authorize` | Contraseña bcrypt | 5 intentos/min por IP |
| `POST /token` | Código + PKCE o refresh | 20 req/min por IP |
| `/healthz`, `/admin/revoke` | Solo loopback | **Nunca** publicar por Funnel |

El listener administrativo (`P40LA_MCP_ADMIN_LISTEN`) no está detrás de Funnel. Su única protección es el bind a loopback y la verificación de `RemoteAddr`; no agregues un proxy público delante.

## OAuth 2.1

- **Obligatorio**: sin `Authorization: Bearer` válido, `/mcp` no ejecuta ninguna tool. No existe modo “URL secreta”.
- **Usuario único**: la contraseña se guarda como hash bcrypt (costo 12) en `P40LA_MCP_OAUTH_PASSWORD_HASH`; el `subject` es opaco (`owner`).
- **Clientes públicos**: DCR solo acepta `token_endpoint_auth_method=none`; no hay `client_secret`.
- **PKCE S256 obligatorio**; el código de autorización es de un solo uso y dura 5 minutos.
- **Redirect URIs allowlisted**: solo `https://claude.ai/api/mcp/auth_callback` y loopback `http://localhost|127.0.0.1|::1/callback` con cualquier puerto. Cualquier otro destino se rechaza en el registro y en `/authorize`.
- **Tokens opacos**: 32 bytes aleatorios, guardados como SHA-256 en `oauth.json` (`0600`). No son JWT: no se pueden validar offline ni reutilizar fuera de este servidor.
- **Vigencias**: access token 1 hora; refresh token 30 días con rotación (el anterior se invalida al usarlo).
- **Sin CORS**: el flujo está pensado para redirects del lado del servidor, no para llamadas desde JavaScript de terceros.

Detalle del flujo y troubleshooting en [oauth.md](oauth.md).

## API key global

- La key tiene **todos los permisos**; equivale a un usuario con acceso completo.
- Se guarda como hash SHA-256 en `system_settings`; el texto plano solo existe al generarla.
- Se crea/rota con `scripts/create-api-key.sh` o `POST /api/api-key/regenerate` (sesión).
- Se revoca con `DELETE /api/api-key` o rotándola.
- `last_used_at` permite detectar uso inesperado.
- Si se filtra: rotar de inmediato, revisar `last_used_at` y avisar al usuario.

La administración de la key es **solo por sesión**: una key no puede crear ni rotar otras keys.

## Compuerta de escritura

`P40LA_MCP_ALLOW_WRITES` debe ser exactamente `true` para que **cualquier** tool de escritura ejecute cambios: `bills_generate` y `bills_pay`, las `autos_*` que escriben, las siete tools de reparaciones del hogar (`home_maintenance_create`, `home_maintenance_update`, `home_maintenance_labor_register`, `home_maintenance_supplies_register`, `home_maintenance_item_update`, `home_maintenance_item_delete` y `home_maintenance_item_move`, con `dry_run=false`), las siete tools de reportes de vivienda (`home_work_request_create`, `home_work_request_update`, `home_work_request_convert`, `home_work_request_status`, `home_work_request_attachment_upload_link`, `home_work_request_attachment_add` y `home_work_request_attachment_delete`, con `dry_run=false`), `budget_transactions_create` (con `dry_run=false`), `homes_update`, `bank_accounts_create`/`bank_accounts_update`, `credit_cards_create`/`credit_cards_update` y las `*_create` de entidades (directas). En modo HTTP se suma `P40LA_MCP_REMOTE_WRITES=true`; **ambas** deben estar encendidas.

| Modo | Escritura con compuertas apagadas | Escritura con compuertas encendidas |
|---|---|---|
| stdio | `writes_disabled`; no llama a la API | Ejecuta con origen `mcp` |
| HTTP | `writes_disabled` si falta cualquiera de las dos | Ejecuta con origen `mcp` y auditoría local |

En facturas, autos, presupuesto, reparaciones del hogar y reportes de vivienda, “escritura” significa `dry_run=false`; `home_maintenance_list`, `home_maintenance_get`, `home_work_request_list`, `home_work_request_get` y `home_work_request_attachment_list` son de lectura y no dependen de la compuerta. En reparaciones, `home_maintenance_create` solo crea la cabecera; `home_maintenance_update` e `home_maintenance_item_update` reemplazan la cabecera o la línea completa (por eso declaran `destructiveHint=true`); `home_maintenance_item_delete` es **destructiva** porque borra una línea de forma permanente y puede dejar la reparación sin líneas, aunque nunca elimina la cabecera; y `home_maintenance_item_move` con `delete_source_if_empty=true` puede borrar la reparación de origen vacía, por eso también declara `destructiveHint=true`. En reportes de vivienda, `home_work_request_create` da de alta un reporte `reported` sin costos y, con `client_request_id`, evita duplicados; `home_work_request_update` edita la cabecera y conserva el estado; `home_work_request_convert` crea una reparación vacía y enlaza `maintenance_id` (destructiva porque crea una entidad); `home_work_request_status` descarta o reactiva el reporte, por eso declara `destructiveHint=true`; `home_work_request_attachment_upload_link` emite un token de corta vida; `home_work_request_attachment_add` carga un archivo pequeño y deduplica por `sha256`; y `home_work_request_attachment_delete` borra un adjunto de forma permanente, por eso declara `destructiveHint=true`. En banca no hay `dry_run`: si la compuerta está activa, la tool escribe directamente y verifica duplicados antes de llamar a la API; si está apagada, devuelve `writes_disabled` sin llamar a la API. Las lecturas (`bills_list`, `bank_accounts_search`, `credit_cards_search`, `home_maintenance_list`, `home_maintenance_get`, `home_work_request_list`, `home_work_request_get` y `home_work_request_attachment_list`) funcionan siempre, con o sin compuerta.

La compuerta se valida en el MCP **y** el backend valida el origen. No protege contra un usuario que ya tiene la API key: es una defensa contra escrituras accidentales del agente y contra un conector remoto mal configurado.

Recomendación: dejar `P40LA_MCP_REMOTE_WRITES=false` de forma permanente si claude.ai solo necesita consultar. Para escribir, usar Claude Code local por stdio o habilitar el remoto de forma temporal y con aprobación explícita.

## Evidencia y links de carga

La carga de fotos y videos por MCP usa un **token de carga** separado de la API key y de OAuth:

| Control | Implementación |
|---|---|
| Secreto | `UPLOAD_TOKEN_SECRET` en el backend, mínimo 32 caracteres; si está vacío, los links se deshabilitan (`upload_link_disabled`) |
| Firma | HMAC-SHA256 sobre `v1.<payload>`; payload con `hid`, `wid`, `exp`, `jti` y `max_files` |
| Alcance | Un `home_id` y un `work_request_id`; solo permite agregar archivos, no listar, descargar, borrar ni editar |
| Vigencia | Default `10m`, máximo `30m` (`ttl_minutes` 1..30). Rotar el secreto invalida todos los links emitidos |
| Cupo | `max_files` 1..10 por link; se persiste el `jti` y solo los archivos nuevos consumen cupo |
| Transporte | `Authorization: Bearer <token>` en `POST /api/work-request-uploads`; **nunca** en la URL, logs, auditoría MCP ni mensajes de error |
| Rate limit | 30 requests/min por hash de token; `429` con `Retry-After` |
| Endpoint | `POST /api/work-request-uploads` no acepta sesión ni API key global; solo el token de carga |
| Estado | Solo reportes `reported`; un `dismissed` devuelve `invalid_status` con hint para reactivar y un `converted` rechaza la carga |

- **El token se devuelve al agente** como parte del resultado de `home_work_request_attachment_upload_link`, porque lo necesita para el `curl`; por eso es de corta vida, está acotado a un reporte y a `max_files`, y no sirve para otro reporte ni para listar, descargar o borrar.
- **No se registran nombres de archivo ni contenido**: el endpoint no escribe el header `Authorization`, el token, el nombre del archivo ni los bytes en logs, auditoría MCP o mensajes de error.
- **La vía inline expone los bytes al proveedor del modelo**: `home_work_request_attachment_add` envía el archivo en base64 como argumento de la tool, así que el contenido viaja a Anthropic. Para archivos grandes o sensibles, preferí `attachment_upload_link` + `curl`, que no pasa los bytes por el modelo.
- **Sin Funnel ni proxy nuevo**: el endpoint queda en el mismo origen del backend. `P40LA_UPLOAD_PUBLIC_URL` solo clasifica `reachability=public` y compone `upload_url`; no publica el endpoint. Exponerlo a Internet requiere una decisión operativa separada con aprobación explícita y un proxy limitado al path de carga.
- **Rotación**: cambiar `UPLOAD_TOKEN_SECRET` y reiniciar el backend invalida todos los links pendientes; no requiere migración. Para revocar un link puntual se puede borrar su fila `jti` de `work_request_upload_tokens`.

## Datos que viajan a Anthropic

Cuando se usa un conector remoto, las llamadas a tools y sus resultados se procesan en la infraestructura de Anthropic:

- Viajan el nombre de la tool, sus argumentos (IDs, períodos, referencias de banco y moneda, alias, montos, fechas y `dry_run` cuando aplica; en autos, marca, modelo, placa, VIN y póliza; en reparaciones, área, descripción y montos de mano de obra e insumos, los campos que cambian en una edición y sus valores propuestos, el `item_id` de una línea que se edita o se borra y, en un movimiento, los IDs de la reparación de origen, del destino y de las líneas; en reportes de vivienda, título, descripción, categoría, prioridad, `client_request_id`, IDs de adjuntos y, en la vía inline, el contenido del archivo en base64) y el resultado (facturas, cuentas, tarjetas, estados, montos, registros de autos, líneas de reparaciones, reportes y metadatos de evidencia, incluidos `file_name` y `sha256`, además del token de carga cuando se emite un link).
- El canal es HTTPS 443; el contenido lo procesa Anthropic para generar la respuesta del modelo.
- El binario no envía datos a ningún otro destino: solo llama a `P40LA_BASE_URL`.
- La auditoría local no guarda argumentos ni resultados; el proveedor del modelo sí los ve para razonar.

Esto incluye datos financieros. El usuario debe aceptarlo antes de habilitar el conector. Si no lo acepta, usar stdio local (el modelo recibe los mismos datos, pero no hay exposición pública) o mantener el remoto en modo lectura.

## Datos y PII

- La documentación, los ejemplos y los tests usan datos sintéticos (`demo@example.com`, `demo-node.example.ts.net`, montos ficticios).
- Los logs del MCP usan `slog` a **stderr**. En HTTP registran método, path, estado y duración; **no** headers, tokens, cuerpos ni IPs.
- La auditoría JSONL registra `tool`, `result` y `duration_ms`, sin argumentos ni resultados.
- El contenido de una tool incluye montos porque es el dato que el usuario pidió operar; no se guarda en el repo ni en logs.
- Prohibido pegar respuestas reales, capturas con montos, `audit.jsonl`, `oauth.json` o dumps en issues, specs o PRs.

## Higiene del repositorio público

Este repositorio es público. Nunca commitees:

- el FQDN real de la tailnet ni nombres de nodos;
- direcciones del rango CGNAT de Tailscale (`100.64/10`), rangos privados (`192.168/16`, `10/8`) o MACs;
- API keys, bearer tokens, refresh tokens, `client_id` reales, hashes bcrypt o contraseñas;
- `oauth.json`, `audit.jsonl`, `.env`, capturas de la pantalla de autorización o de claude.ai con datos reales;
- logs de Tailscale, `tailscale status`, `funnel status` o configuraciones con el dominio real.

En docs, issues y ejemplos usá `demo-node.example.ts.net`, `192.0.2.10`, `demo@example.com`, `demo-api-key` y UUID cero (`00000000-0000-4000-8000-000000000000`). Ver `SEC-NO-PII`, `SEC-SYNTHETIC` y `SEC-LOGS`.

## Auditoría

- `bill_history.source = mcp` identifica las operaciones del agente en el backend.
- Una petición autenticada por API key fuerza `source=mcp`; el header `X-P40LA-Source` no puede simular `dashboard`.
- El historial conserva el diff de campos en altas y pagos.
- En HTTP, `audit.jsonl` agrega una capa local por cada `tools/call`: `ts`, `request_id`, `subject`, `tool`, `result`, `duration_ms`. Sin argumentos, resultados, montos ni tokens.
- El `request_id` es aleatorio por request y permite correlacionar una llamada con los logs del proceso, que tampoco incluyen payloads.
- Los archivos de auditoría son `0600`, rotan por tamaño y no se envían a ningún servicio externo.

## Rate limit y límites

| Límite | Valor | Respuesta |
|---|---|---|
| `/mcp` por token | 60 req/min (1–600) | `429` + `Retry-After` |
| `POST /authorize` por IP | 5/min | `429` |
| `POST /register` por IP | 10/h | `429` |
| `POST /token` por IP | 20/min | `429` |
| `POST /api/work-request-uploads` por hash de token | 30/min | `429` + `Retry-After` |
| Body de `/mcp` | 1 MiB | `413` |
| Headers | 64 KiB en el listener público | `431`/cierre |
| Lectura | 30 s | timeout |

Los contadores viven en memoria y se reinician con el proceso. No son una defensa contra un atacante con muchos recursos, pero reducen fuerza bruta y abuso accidental. El límite por token usa el hash del bearer; los endpoints OAuth usan la IP (priorizando `X-Forwarded-For`).

## Rotación y revocación

| Credencial | Rotar | Revocar | Efecto |
|---|---|---|---|
| API key | Script o `POST /api/api-key/regenerate` | `DELETE /api/api-key` | La key anterior deja de funcionar de inmediato |
| Access/refresh OAuth | Automática (refresh rotativo) | `POST /admin/revoke` | Todos los tokens y códigos se invalidan; los clientes se conservan |
| Contraseña OAuth | Nuevo hash + reinicio | — | Solo afecta autorizaciones nuevas; **revocar** después |
| `UPLOAD_TOKEN_SECRET` | Nuevo secreto + reinicio del backend | Borrar la fila `jti` del link | Invalida todos los links de carga emitidos; no requiere migración |
| Estado OAuth completo | Borrar `oauth.json` con el proceso detenido | — | Se pierden clientes y tokens |

Al rotar la contraseña, los tokens ya emitidos siguen vigentes hasta su vencimiento o hasta `revoke`. Si se sospecha una filtración, rotar **y** revocar.

## Backfill en producción

Antes de generar facturas históricas contra el iHost:

1. Ejecutar `bills_list` para revisar el estado actual.
2. Ejecutar `bills_generate` con `dry_run=true` y mostrar el resumen.
3. Tomar un respaldo de la DB según `docs/rules/40-data.md` (`DATA-PROD-WRITE`).
4. Confirmar con el usuario.
5. Ejecutar con `dry_run=false` (y `P40LA_MCP_REMOTE_WRITES=true` si es por HTTP).
6. Verificar con `bills_list` y el historial de una factura (`source=mcp`).

Nunca se valida con escrituras contra la DB local del usuario ni se usa `data/app.db` como conejillo de indias.

## Protocolo de incidente

Si se filtra una API key, un bearer token, un refresh token, la contraseña de administración, el nombre real de la tailnet o un dato real:

1. **Contener**: si el token OAuth se filtró, `POST /admin/revoke`; si la API key, rotarla; si el endpoint está comprometido, `tailscale funnel reset` y detener el contenedor.
2. **Rotar**: generar una contraseña nueva y su hash; rotar la API key; reiniciar el proceso con las credenciales nuevas.
3. **Purgar**: eliminar el dato del árbol **y de la historia** de git si se commiteó (no alcanza con un commit posterior).
4. **Revisar**: `audit.jsonl`, `last_used_at` de la API key, `tailscale funnel status` y logs del contenedor para detectar uso indebido.
5. **Documentar** el incidente y avisar al usuario.

Ver `docs/rules/50-security.md` (`SEC-INCIDENT`).

## Errores seguros

El MCP no devuelve stack traces ni SQL. Los errores tienen código estable y mensaje accionable:

| Código | Significado |
|---|---|
| `auth_failed` | API key inválida, revocada o rotada |
| `writes_disabled` | Escritura sin compuerta local o remota |
| `invalid_range` | Rango de períodos inválido o futuro |
| `invalid_selector` | Selector de pago ausente o múltiple |
| `invalid_paid_at` | Fecha de pago inválida o futura |
| `service_not_found` | Servicio inexistente |
| `service_inactive` | Servicio inactivo al generar |
| `amount_unresolved` | No se pudo determinar el monto |
| `deleted_slot` | Período ocupado por una factura eliminada |
| `upstream_error` | Falla de red o del servidor |
| `invalid_token` | Bearer OAuth ausente, vencido o revocado (HTTP 401); también token de carga inválido |
| `expired_token` | Token de carga vencido (HTTP 401) |
| `too_many_requests` | Rate limit alcanzado (HTTP 429) |
| `payload_too_large` | Body mayor a 1 MiB (HTTP 413) |
| `attachment_limit` | El reporte ya tiene 10 adjuntos (HTTP 409) |
| `attachment_not_found` | El adjunto no existe (HTTP 404); `attachment_delete` lo trata como `unchanged` |
| `upload_token_exhausted` | El token de carga agotó su cupo de archivos (HTTP 409) |
| `invalid_file` | Archivo vacío, tipo no permitido o multipart inválido (HTTP 400) |
| `unsupported_media` | El request de carga no es `multipart/form-data` (HTTP 415) |
| `request_too_large` | Archivo o lote de carga supera el límite (HTTP 413) |
| `upload_link_disabled` | Falta `UPLOAD_TOKEN_SECRET` en el backend (HTTP 503) |
