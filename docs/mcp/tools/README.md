---
title: "Catálogo de tools"
sidebar_position: 1
description: "Convenciones y listado de tools MCP disponibles"
---

# Catálogo de tools

## Convención de nombres

- `snake_case`, con el dominio primero: `bills_list`, `entities_search`, `autos_maintenance_register`, `bank_accounts_search`, `budget_transactions_create`.
- El prefijo del dominio separa conceptos homónimos: las cuentas del presupuesto son `budget_accounts_list` (tabla `accounts`) y las cuentas bancarias son `bank_accounts_search` (tabla `bank_accounts`).
- Verbo al final: `list`/`search` (lectura), `create` (alta), `update` (edición), `generate` (alta masiva), `pay` (acción), `register` (registro de un hecho).
- Las tools de escritura respetan `P40LA_MCP_ALLOW_WRITES`. Las de facturas, autos, presupuesto, las siete de reparaciones del hogar y las cuatro de reportes de vivienda declaran `dry_run`; las altas de entidades y las de banca no tienen previsualización y verifican duplicados antes del POST; `homes_update` es una edición idempotente de un hogar, rechaza un `name` que ya pertenece a otro hogar y tampoco tiene `dry_run`. Las de banca resuelven banco y moneda por nombre/código; las de reparaciones incluyen el alta de la reparación vacía, el registro de líneas —que resuelve o crea el mantenimiento del día y deduplica por línea—, la edición de la cabecera y de cada línea, el borrado de una línea —que nunca borra la cabecera— y el movimiento de líneas entre reparaciones del mismo hogar; las de reportes de vivienda dan de alta un reporte en estado `reported`, editan su cabecera con merge parcial, lo convierten en una reparación vacía o lo descartan/reactivan; las de presupuesto resuelven la categoría por ID o por nombre exacto, la cuenta por ID, vínculo de la categoría o primera activa, y deduplican el gasto por fecha, categoría, monto y `payee`. Las **31 tools de lectura** no dependen de la compuerta: búsquedas (`entities_search`, `bank_accounts_search`, `credit_cards_search`, `autos_list`) y las vistas y detalles agregados en SPEC-154, SPEC-159, SPEC-163 y SPEC-165 (pensión, alquileres, deudas, estados de cuenta, reparaciones, reportes de vivienda, facturas por mes, alertas, configuración y presupuesto).

## Tools actuales

El catálogo tiene **59 tools**: 31 de lectura y 28 de escritura.

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
| `home_maintenance_get` | Reparaciones | Lectura | [home-maintenance.md](home-maintenance.md) |
| `home_work_request_list` | Reportes | Lectura | [home-work-requests.md](home-work-requests.md) |
| `home_work_request_get` | Reportes | Lectura | [home-work-requests.md](home-work-requests.md) |
| `automation_accounts_list` | Automatizaciones | Lectura | [automation.md](automation.md) |
| `automation_runs_list` | Automatizaciones | Lectura | [automation.md](automation.md) |
| `alerts_list` | Alertas | Lectura | [alerts.md](alerts.md) |
| `settings_status` | Configuración | Lectura | [settings.md](settings.md) |
| `budget_categories_list` | Presupuesto | Lectura | [budget.md](budget.md) |
| `budget_accounts_list` | Presupuesto | Lectura | [budget.md](budget.md) |
| `budget_transactions_list` | Presupuesto | Lectura | [budget.md](budget.md) |
| `budget_month_view` | Presupuesto | Lectura | [budget.md](budget.md) |
| `bills_generate` | Facturas | Escritura | [bills.md](bills.md) |
| `bills_pay` | Facturas | Escritura | [bills.md](bills.md) |
| `autos_create` | Autos | Escritura | [autos.md](autos.md) |
| `autos_update` | Autos | Escritura | [autos.md](autos.md) |
| `autos_maintenance_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_inspection_register` | Autos | Escritura | [autos.md](autos.md) |
| `autos_insurance_register` | Autos | Escritura | [autos.md](autos.md) |
| `home_maintenance_create` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_maintenance_update` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_maintenance_labor_register` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_maintenance_supplies_register` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_maintenance_item_update` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_maintenance_item_delete` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_maintenance_item_move` | Reparaciones | Escritura | [home-maintenance.md](home-maintenance.md) |
| `home_work_request_create` | Reportes | Escritura | [home-work-requests.md](home-work-requests.md) |
| `home_work_request_update` | Reportes | Escritura | [home-work-requests.md](home-work-requests.md) |
| `home_work_request_convert` | Reportes | Escritura | [home-work-requests.md](home-work-requests.md) |
| `home_work_request_status` | Reportes | Escritura | [home-work-requests.md](home-work-requests.md) |
| `homes_create` | Hogares | Escritura | [entities.md](entities.md) |
| `homes_update` | Hogares | Escritura | [entities.md](entities.md) |
| `institutions_create` | Instituciones | Escritura | [entities.md](entities.md) |
| `banks_create` | Bancos | Escritura | [entities.md](entities.md) |
| `services_create` | Servicios | Escritura | [entities.md](entities.md) |
| `bank_accounts_create` | Banca | Escritura | [banking.md](banking.md) |
| `bank_accounts_update` | Banca | Escritura | [banking.md](banking.md) |
| `credit_cards_create` | Banca | Escritura | [banking.md](banking.md) |
| `credit_cards_update` | Banca | Escritura | [banking.md](banking.md) |
| `budget_transactions_create` | Presupuesto | Escritura | [budget.md](budget.md) |

Las **19 tools de la Ola 1 de SPEC-154** son de solo lectura, reutilizan endpoints existentes y no agregan la compuerta de escritura. Las que pueden devolver datos personales o secretos aplican una lista de omisión o una allowlist: `alerts_list` omite `speech`, `settings_status` expone solo una allowlist (sin secretos, emails ni chat IDs), `automation_accounts_list` y `automation_runs_list` omiten `identifier`, `webhook_url` y payloads, y las `rental_*` omiten emails y teléfonos de primer nivel.

Las **7 tools de escritura de reparaciones del hogar** respetan `dry_run` y la compuerta: `home_maintenance_create` (SPEC-157) da de alta la reparación vacía; `home_maintenance_update` (SPEC-159) edita la cabecera con merge parcial y no toca las líneas; `home_maintenance_labor_register` y `home_maintenance_supplies_register` (SPEC-155) resuelven o crean el mantenimiento del día y deduplican por línea; `home_maintenance_item_update` (SPEC-159) edita una línea y devuelve los totales recalculados; `home_maintenance_item_delete` (SPEC-159) borra una línea sin borrar la cabecera; y `home_maintenance_item_move` (SPEC-157) mueve líneas a otra reparación del mismo hogar, con destino existente o creado en la misma transacción. `home_maintenance_list` y `home_maintenance_get` (SPEC-159) son de lectura.

El **toolset de reportes de vivienda** (SPEC-165) agrega dos tools de lectura y cuatro de escritura sobre el módulo `home.workrequest` de SPEC-164. `home_work_request_list` y `home_work_request_get` consultan los reportes pendientes, convertidos y descartados, con el conteo de evidencia y, en el detalle, los metadatos de fotos y videos. `home_work_request_create` da de alta un reporte en estado `reported`, sin costos; `home_work_request_update` edita la cabecera con merge parcial sin cambiar el estado; `home_work_request_convert` lo convierte en una reparación vacía de `home.maintenance`; y `home_work_request_status` lo descarta o reactiva. Las cuatro de escritura respetan `dry_run` (`true` por defecto) y la compuerta. La evidencia no se carga por MCP: el alta y el borrado de archivos se hacen desde la web. El detalle está en [home-work-requests.md](home-work-requests.md).

El **toolset de presupuesto** (SPEC-163) agrega cuatro tools de lectura y una de escritura sobre el módulo estilo YNAB de SPEC-093. Las lecturas no dependen de la compuerta y sirven para que el agente mapee un gasto a su categoría y su cuenta: `budget_categories_list` devuelve los grupos con sus categorías activas, `budget_accounts_list` las cuentas del presupuesto con su moneda y saldo calculado, `budget_transactions_list` las transacciones de un mes y `budget_month_view` la vista mensual con `assigned`, `activity` y `available` por moneda. `budget_transactions_create` registra un gasto (`outflow`), deriva el mes de la fecha y la moneda de la cuenta, resuelve la categoría por ID o por nombre exacto y deduplica por fecha, categoría, monto y `payee`; respeta `dry_run` y la compuerta.

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

| Transporte | Lectura (las 31 tools de solo lectura) | Escritura (`dry_run=false` en facturas, autos, presupuesto, reparaciones del hogar y reportes de vivienda; `*_create`/`*_update` de entidades y banca) |
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

- Lectura: las **31 tools de solo lectura** (`bills_list`, `entities_search`, `services_get`, `autos_list`, `autos_inspection_alerts`, `bank_accounts_search`, `credit_cards_search`, las vistas de facturas y las de pensión, alquileres, deudas, estados de cuenta, reparaciones y reportes de vivienda —incluidos `home_maintenance_get`, `home_work_request_list` y `home_work_request_get`—, automatizaciones, alertas, configuración y presupuesto) usan `readOnlyHint=true` y `openWorldHint=false`. Ninguna declara `destructiveHint` ni modifica datos.
- Escritura idempotente de facturas y autos: `bills_generate`, `bills_pay` y `autos_update` usan `destructiveHint=false`, `idempotentHint=true`, `openWorldHint=false`.
- Altas de autos: `autos_create`, `autos_maintenance_register`, `autos_inspection_register` y `autos_insurance_register` usan `destructiveHint=false`, `idempotentHint=false`, `openWorldHint=false`.
- Reparaciones del hogar: `home_maintenance_create`, `home_maintenance_labor_register` y `home_maintenance_supplies_register` son aditivas y declaran `destructiveHint=false`, `idempotentHint=false`, `openWorldHint=false`; el alta no deduplica y las de registro deduplican por día antes de escribir, pero dos llamadas concurrentes podrían crear líneas distintas. `home_maintenance_item_move` declara `destructiveHint=true` (con `delete_source_if_empty=true` puede borrar la reparación de origen), `idempotentHint=false` y `openWorldHint=false`. Las ediciones `home_maintenance_update` y `home_maintenance_item_update` declaran `destructiveHint=true` (reemplazan la cabecera o la línea completa), `idempotentHint=true` y `openWorldHint=false`. `home_maintenance_item_delete` declara `destructiveHint=true`, `idempotentHint=true` (repetir el borrado no cambia el estado, aunque el segundo intento devuelve `item_not_found`) y `openWorldHint=false`. Sus anotaciones quedan congeladas en `contracts.json`.
- Reportes de vivienda: `home_work_request_list` y `home_work_request_get` declaran `readOnlyHint=true` y `openWorldHint=false`. `home_work_request_create` es aditiva y declara `destructiveHint=false`, `idempotentHint=false` y `openWorldHint=false`. `home_work_request_update` y `home_work_request_status` declaran `destructiveHint=true`, `idempotentHint=true` y `openWorldHint=false`; `home_work_request_convert` declara `destructiveHint=true`, `idempotentHint=false` y `openWorldHint=false` porque crea una reparación y enlaza el reporte. Sus anotaciones quedan congeladas en `contracts.json`.
- Altas de entidades: `homes_create`, `institutions_create`, `banks_create` y `services_create` usan `destructiveHint=true`, `openWorldHint=false`. No declaran `idempotentHint`.
- Edición de hogares: `homes_update` usa `destructiveHint=true`, `idempotentHint=true`, `openWorldHint=false`. No tiene `dry_run`.
- Escritura de banca: las altas y ediciones usan `destructiveHint=true` y `openWorldHint=false`. Las ediciones (`*_update`) usan `idempotentHint=true`; las altas (`*_create`) usan `idempotentHint=false`.
- Presupuesto: las cuatro tools de lectura (`budget_categories_list`, `budget_accounts_list`, `budget_transactions_list` y `budget_month_view`) declaran `readOnlyHint=true` y `openWorldHint=false`. `budget_transactions_create` es aditiva y declara `destructiveHint=false`, `idempotentHint=false` y `openWorldHint=false`: deduplica antes de escribir, pero dos llamadas concurrentes podrían crear dos filas. Sus anotaciones quedan congeladas en `contracts.json`.

`destructiveHint=false` en facturas y autos significa que son aditivas (crean o actualizan estado), no borran datos. En las altas de entidades y en banca el valor es `true` de forma conservadora: las tools de alta no garantizan ser seguras ante reintentos, aunque en la práctica solo agregan registros y nunca borran; las ediciones modifican datos existentes. La confirmación humana sigue siendo obligatoria por el impacto en los datos. `autos_insurance_register` declara `idempotentHint=false`; repetir con el mismo `service_id` devuelve `already_exists` y no crea un duplicado.
