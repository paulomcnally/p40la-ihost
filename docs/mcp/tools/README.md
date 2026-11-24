---
title: "Catálogo de tools"
sidebar_position: 1
description: "Convenciones y listado de tools MCP disponibles"
---

# Catálogo de tools

## Convención de nombres

- `snake_case`, con el dominio primero: `bills_list`, `entities_search`, `autos_maintenance_register`, `bank_accounts_search`.
- Verbo al final: `list`/`search` (lectura), `create` (alta), `update` (edición), `generate` (alta masiva), `pay` (acción), `register` (registro de un hecho).
- Las tools de escritura respetan `P40LA_MCP_ALLOW_WRITES`. Las de facturas, autos y reparaciones del hogar declaran `dry_run`; las altas de entidades y las de banca no tienen previsualización y verifican duplicados antes del POST. Las de banca resuelven banco y moneda por nombre/código; las de reparaciones resuelven o crean el mantenimiento del día y deduplican por línea. Las **24 tools de lectura** no dependen de la compuerta: búsquedas (`entities_search`, `bank_accounts_search`, `credit_cards_search`, `autos_list`) y las vistas y detalles agregados en SPEC-154 (pensión, alquileres, deudas, estados de cuenta, reparaciones, facturas por mes, alertas y configuración).

## Tools actuales

El catálogo tiene **41 tools**: 24 de lectura y 17 de escritura.

| Tool | Dominio | Tipo | Documentación |
|---|---|---|---|
| `bills_list` | Facturas | Lectura | [bills.md](bills.md) |
| `bills_by_month` | Facturas | Lectura | [bills.md](bills.md) |
| `bills_summary` | Facturas | Lectura | [bills.md](bills.md) |
| `bills_projection` | Facturas | Lectura | [bills.md](bills.md) |
| `entities_search` | Entidades | Lectura | [entities.md](entities.md) |
| `services_get` | Servicios | Lectura | [entities.md](entities.md) |
| `autos_list` | Autos | Lectura | [autos.md](autos.md) |
| `autos_inspection_alerts` | Autos | Lectura | [autos.md](autos.md) |
| `bank_accounts_search` | Banca | Lectura | [banking.md](banking.md) |
| `credit_cards_search` | Banca | Lectura | [banking.md](banking.md) |
| `credit_card_statements_list` | Banca | Lectura | [cards.md](cards.md) |
| `credit_card_statements_get` | Banca | Lectura | [cards.md](cards.md) |
| `pension_month_summary` | Pensión | Lectura | [pension.md](pension.md) |
| `pension_catalogs_list` | Pensión | Lectura | [pension.md](pension.md) |
| `rental_leases_list` | Alquileres | Lectura | [rental.md](rental.md) |
| `rental_lease_get` | Alquileres | Lectura | [rental.md](rental.md) |
| `rental_home_status` | Alquileres | Lectura | [rental.md](rental.md) |
| `rental_portfolio_summary` | Alquileres | Lectura | [rental.md](rental.md) |
| `debt_bills_list` | Deudas | Lectura | [debts.md](debts.md) |
| `home_maintenance_list` | Reparaciones | Lectura | [home-maintenance.md](home-maintenance.md) |
| `automation_accounts_list` | Automatizaciones | Lectura | [automation.md](automation.md) |
| `automation_runs_list` | Automatizaciones | Lectura | [automation.md](automation.md) |
| `alerts_list` | Alertas | Lectura | [alerts.md](alerts.md) |
| `settings_status` | Configuración | Lectura | [settings.md](settings.md) |
| `bills_generate` | Facturas | Escritura | [bills.md](bills.md) |
| `bills_pay` | Facturas | Escritura | [bills.md](bills.md) |
| `autos_create` | Autos | Escritura | [autos.md](autos.md) |
| `autos_update` | Autos | Escritura | [autos.md](autos.md) |
| `autos_maintenance_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_inspection_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_insurance_register` | Autos | Escritura | [autos.md](autos.md) |
| `home_maintenance_labor_register` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_maintenance_supplies_register` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `homes_create` | Hogares | Escritura | [entities.md](entities.md) |
| `institutions_create` | Instituciones | Escritura | [entities.md](entities.md) |
| `banks_create` | Bancos | Escritura | [entities.md](entities.md) |
| `services_create` | Servicios | Escritura | [entities.md](entities.md) |
| `bank_accounts_create` | Banca | Escritura | [banking.md](banking.md) |
| `bank_accounts_update` | Banca | Escritura | [banking.md](banking.md) |
| `credit_cards_create` | Banca | Escritura | [banking.md](banking.md) |
| `credit_cards_update` | Banca | Escritura | [banking.md](banking.md) |

Las **19 tools de la Ola 1 de SPEC-154** son de solo lectura, reutilizan endpoints existentes y no agregan la compuerta de escritura. Las que pueden devolver datos personales o secretos aplican una lista de omisión o una allowlist: `alerts_list` omite `speech`, `settings_status` expone solo una allowlist (sin secretos, emails ni chat IDs), `automation_accounts_list` y `automation_runs_list` omiten `identifier`, `webhook_url` y payloads, y las `rental_*` omiten emails y teléfonos de primer nivel.

Las **2 tools de registro de reparaciones del hogar de SPEC-155** (`home_maintenance_labor_register` y `home_maintenance_supplies_register`) son las únicas de escritura de ese dominio: resuelven o crean el mantenimiento del día, deduplican por línea y respetan `dry_run`. `home_maintenance_list` sigue siendo de lectura.

## Contrato congelado y manifiesto de impacto

El catálogo se controla con:

- [`impact.json`](impact.json) — manifiesto de dependencias: entrypoints, archivos de backend, endpoints, tests y documentación. Lo valida `./scripts/check-mcp-impact.sh --check-map` y lo exige el hook `pre-commit`. El formato, los comandos y el flujo de trabajo están en [impact.md](impact.md).
- [`contracts.json`](contracts.json) — golden de `tools/list`: nombres, descripciones, esquemas de entrada/salida y anotaciones. `TestToolContracts` (`internal/mcp/tools/all/contract_test.go`) falla si el contrato cambia sin actualizar el archivo.
- `TestMCPBackendContract` (`internal/api/mcp_backend_contract_test.go`) — sin golden: recorre API key → router → handler → servicio → storage → JSON con una DB temporal y detecta drift de rutas, tags JSON, auth o `source=mcp`.

Un cambio de contrato es intencional solo si la spec lo declara. Para regenerar el golden:

```bash
go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts
```

Después revisá el diff de `contracts.json`, corré el test sin `-update-contracts` y verificá el impacto:

```bash
go test ./internal/mcp/...
go test ./internal/api -run TestMCPBackendContract
./scripts/check-mcp-impact.sh --staged --gate
```

## Mismos tools en stdio y HTTP

El catálogo no cambia según el transporte. `p40la-mcp` monta el mismo `mcp.Server` en stdio (Claude Desktop/Code local, OpenCode) y en Streamable HTTP (`/mcp`, claude.ai). No hay tools “remotas” ni permisos distintos por cliente: la API key global tiene todos los permisos y el `subject` OAuth (`owner`) es solo un identificador de auditoría.

Lo que cambia es la **compuerta de escritura**:

| Transporte | Lectura (las 24 tools de solo lectura) | Escritura (`dry_run=false` en facturas, autos y reparaciones; `*_create`/`*_update` de entidades y banca) |
|---|---|---|
| stdio | Siempre disponible | Requiere `P40LA_MCP_ALLOW_WRITES=true` |
| HTTP | Siempre disponible | Requiere `P40LA_MCP_ALLOW_WRITES=true` **y** `P40LA_MCP_REMOTE_WRITES=true` |

Si falta la compuerta remota, la tool responde:

```json
{
  "error": "writes_disabled",
  "message": "Las escrituras están deshabilitadas en este servidor MCP.",
  "hint": "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true, o repetí con dry_run=true para previsualizar."
}
```

En las tools de banca, que no tienen `dry_run`, el `hint` omite esa alternativa y solo indica las variables de la compuerta.

El modo HTTP no agrega tools nuevas ni cambia los schemas de entrada. Los límites de body (1 MiB), rate limit y auditoría son del servidor, no de cada tool. Ver [http.md](../http.md), [oauth.md](../oauth.md) y [security.md](../security.md).

En las tools `*_create` el `hint` solo pide habilitar las variables, porque no existe `dry_run` para previsualizar. La compuerta se evalúa antes de validar la entrada: sin permisos, la tool no llama al backend.

## Respuestas

Las tools devuelven contenido de texto JSON y `structuredContent` con el mismo objeto. Los errores de negocio usan `isError=true` y un objeto:

```json
{
  "error": "invalid_range",
  "message": "el período final no puede ser anterior al inicial",
  "hint": "Usá from y to en formato YYYY-MM"
}
```

## Anotaciones MCP

- Lectura: las **24 tools de solo lectura** (`bills_list`, `entities_search`, `services_get`, `autos_list`, `autos_inspection_alerts`, `bank_accounts_search`, `credit_cards_search`, las vistas de facturas y las de pensión, alquileres, deudas, estados de cuenta, reparaciones, automatizaciones, alertas y configuración) usan `readOnlyHint=true` y `openWorldHint=false`. Ninguna declara `destructiveHint` ni modifica datos.
- Escritura idempotente de facturas y autos: `bills_generate`, `bills_pay` y `autos_update` usan `destructiveHint=false`, `idempotentHint=true`, `openWorldHint=false`.
- Altas de autos: `autos_create`, `autos_maintenance_register`, `autos_inspection_register` y `autos_insurance_register` usan `destructiveHint=false`, `idempotentHint=false`, `openWorldHint=false`.
- Altas de reparaciones del hogar: `home_maintenance_labor_register` y `home_maintenance_supplies_register` son aditivas y no garantizan idempotencia: deduplican por día antes de escribir, pero dos llamadas concurrentes podrían crear líneas distintas. Sus anotaciones quedan congeladas en `contracts.json`.
- Altas de entidades: `homes_create`, `institutions_create`, `banks_create` y `services_create` usan `destructiveHint=true`, `openWorldHint=false`. No declaran `idempotentHint`.
- Escritura de banca: las altas y ediciones usan `destructiveHint=true` y `openWorldHint=false`. Las ediciones (`*_update`) usan `idempotentHint=true`; las altas (`*_create`) usan `idempotentHint=false`.

`destructiveHint=false` en facturas y autos significa que son aditivas (crean o actualizan estado), no borran datos. En las altas de entidades y en banca el valor es `true` de forma conservadora: las tools de alta no garantizan ser seguras ante reintentos, aunque en la práctica solo agregan registros y nunca borran; las ediciones modifican datos existentes. La confirmación humana sigue siendo obligatoria por el impacto en los datos. `autos_insurance_register` declara `idempotentHint=false`; repetir con el mismo `service_id` devuelve `already_exists` y no crea un duplicado.
