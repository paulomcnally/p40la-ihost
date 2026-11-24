---
title: "Seguridad"
sidebar_position: 4
description: "Credenciales, compuerta de escritura, datos y protocolo de incidentes"
---

# Seguridad

## Principios

1. **Menor privilegio**: por defecto el MCP solo lee. Las escrituras requieren una variable explícita.
2. **Sin acceso directo a datos**: no abre SQLite, no lee el volumen, no escribe archivos de estado.
3. **Confirmación humana**: las tools de escritura previsualizan con `dry_run` y el agente debe pedir confirmación antes de ejecutar.
4. **Datos mínimos**: los logs no incluyen montos, emails, cookies, contraseñas ni payloads.

## Credenciales

- `P40LA_EMAIL` y `P40LA_PASSWORD`, o `P40LA_SESSION_COOKIE`.
- Solo por variables de entorno o un archivo `0600`; nunca por argumentos.
- La cookie de sesión vive en memoria (`http.CookieJar`) y no se persiste.
- Si se filtra una cookie o contraseña, se rota la contraseña del usuario y se invalida la sesión.
- En redes no confiables, usá HTTPS mediante un proxy inverso (Tailscale, Caddy, Nginx). El valor por defecto es HTTP en LAN.

## Compuerta de escritura

`P40LA_MCP_ALLOW_WRITES` debe ser exactamente `true` para que `bills_generate` y `bills_pay` ejecuten cambios con `dry_run=false`.

| Situación | Resultado |
|---|---|
| `dry_run=true` (default) | Previsualiza; no escribe |
| `dry_run=false` + compuerta apagada | Error `writes_disabled`; no llama a la API |
| `dry_run=false` + compuerta encendida | Ejecuta con origen de auditoría `mcp` |

La compuerta se valida en el MCP **y** el backend valida el origen; no es una defensa criptográfica contra un usuario con la contraseña, sino una protección contra escrituras accidentales del agente.

## Datos y PII

- La documentación, los ejemplos y los tests usan datos sintéticos (`demo@example.com`, montos ficticios).
- Los logs del MCP usan `slog` a **stderr** y no registran requests ni respuestas.
- El contenido de una tool incluye montos porque es el dato que el usuario pidió operar; no se guarda en el repo ni en logs.
- Prohibido pegar respuestas reales, capturas con montos o dumps en issues, specs o PRs.

## Backfill en producción

Antes de generar facturas históricas contra el iHost:

1. Ejecutar `bills_list` para revisar el estado actual.
2. Ejecutar `bills_generate` con `dry_run=true` y mostrar el resumen.
3. Tomar un respaldo de la DB según `docs/rules/40-data.md` (`DATA-PROD-WRITE`).
4. Confirmar con el usuario.
5. Ejecutar con `dry_run=false`.
6. Verificar con `bills_list` y el historial de una factura (`source=mcp`).

Nunca se valida con escrituras contra la DB local del usuario ni se usa `data/app.db` como conejillo de indias.

## Auditoría

- `bill_history.source = mcp` identifica las operaciones del agente.
- El header `X-P40LA-Source` tiene allowlist (`dashboard`, `mcp`); cualquier otro valor devuelve `invalid_source`.
- El historial conserva el diff de campos en altas y pagos.

## Protocolo de incidente

Si se commitea o se filtra una credencial, cookie o dato real:

1. Rotar la credencial o contraseña de inmediato.
2. Eliminar el dato del árbol **y de la historia** de git.
3. Documentar el incidente y avisar al usuario.
4. Revisar logs y accesos por si hubo uso indebido.

Ver `docs/rules/50-security.md` (`SEC-INCIDENT`).

## Errores seguros

El MCP no devuelve stack traces ni SQL. Los errores tienen código estable y mensaje accionable:

| Código | Significado |
|---|---|
| `auth_failed` | Credenciales o cookie inválidas |
| `writes_disabled` | Escritura sin compuerta |
| `invalid_range` | Rango de períodos inválido o futuro |
| `invalid_selector` | Selector de pago ausente o múltiple |
| `invalid_paid_at` | Fecha de pago inválida o futura |
| `service_not_found` | Servicio inexistente |
| `service_inactive` | Servicio inactivo al generar |
| `amount_unresolved` | No se pudo determinar el monto |
| `deleted_slot` | Período ocupado por una factura eliminada |
| `upstream_error` | Falla de red o del servidor |
