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
4. **Confirmación humana**: toda tool de escritura exige confirmación del usuario antes de ejecutar. Las de facturas, autos y reparaciones previsualizan con `dry_run`; las de banca no tienen `dry_run` y el agente debe mostrar el resumen (banco y moneda resueltos, cuerpo y duplicados) antes de escribir. Las altas de entidades tampoco tienen `dry_run`: se muestra el cuerpo y si ya existe un duplicado.
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

`P40LA_MCP_ALLOW_WRITES` debe ser exactamente `true` para que **cualquier** tool de escritura ejecute cambios: `bills_generate` y `bills_pay`, las `autos_*` que escriben, las cuatro tools de reparaciones del hogar (`home_maintenance_create`, `home_maintenance_labor_register`, `home_maintenance_supplies_register` y `home_maintenance_item_move`, con `dry_run=false`), `bank_accounts_create`/`bank_accounts_update`, `credit_cards_create`/`credit_cards_update` y las `*_create` de entidades (directas). En modo HTTP se suma `P40LA_MCP_REMOTE_WRITES=true`; **ambas** deben estar encendidas.

| Modo | Escritura con compuertas apagadas | Escritura con compuertas encendidas |
|---|---|---|
| stdio | `writes_disabled`; no llama a la API | Ejecuta con origen `mcp` |
| HTTP | `writes_disabled` si falta cualquiera de las dos | Ejecuta con origen `mcp` y auditoría local |

En facturas, autos y reparaciones del hogar, “escritura” significa `dry_run=false`; `home_maintenance_list` es de lectura y no depende de la compuerta. En reparaciones, `home_maintenance_create` solo crea la cabecera y `home_maintenance_item_move` con `delete_source_if_empty=true` puede borrar la reparación de origen vacía, por eso declara `destructiveHint=true`. En banca no hay `dry_run`: si la compuerta está activa, la tool escribe directamente y verifica duplicados antes de llamar a la API; si está apagada, devuelve `writes_disabled` sin llamar a la API. Las lecturas (`bills_list`, `bank_accounts_search`, `credit_cards_search`, `home_maintenance_list`) funcionan siempre, con o sin compuerta.

La compuerta se valida en el MCP **y** el backend valida el origen. No protege contra un usuario que ya tiene la API key: es una defensa contra escrituras accidentales del agente y contra un conector remoto mal configurado.

Recomendación: dejar `P40LA_MCP_REMOTE_WRITES=false` de forma permanente si claude.ai solo necesita consultar. Para escribir, usar Claude Code local por stdio o habilitar el remoto de forma temporal y con aprobación explícita.

## Datos que viajan a Anthropic

Cuando se usa un conector remoto, las llamadas a tools y sus resultados se procesan en la infraestructura de Anthropic:

- Viajan el nombre de la tool, sus argumentos (IDs, períodos, referencias de banco y moneda, alias, montos, fechas y `dry_run` cuando aplica; en autos, marca, modelo, placa, VIN y póliza; en reparaciones, área, descripción y montos de mano de obra e insumos y, en un movimiento, los IDs de la reparación de origen, del destino y de las líneas) y el resultado (facturas, cuentas, tarjetas, estados, montos, registros de autos o líneas de reparaciones).
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
| `invalid_token` | Bearer OAuth ausente, vencido o revocado (HTTP 401) |
| `too_many_requests` | Rate limit alcanzado (HTTP 429) |
| `payload_too_large` | Body mayor a 1 MiB (HTTP 413) |
