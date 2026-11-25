---
title: "MCP de p40la-ihost"
sidebar_position: 1
description: "Servidor MCP y tools para operar p40la-ihost desde agentes"
---

# MCP de p40la-ihost

`p40la-mcp` es un servidor [Model Context Protocol](https://modelcontextprotocol.io/) escrito en Go que expone tools para operar p40la-ihost desde un agente (OpenCode, Claude Desktop, claude.ai u otro cliente MCP). Usa la **API HTTP** del sistema con una **API key global**; no accede a SQLite ni al volumen de datos.

El binario tiene dos transportes con el mismo catálogo de tools:

| Modo | Transporte | Cliente | Exposición |
|---|---|---|---|
| Local (default) | `stdio` | OpenCode, Claude Code y Claude Desktop local | Ninguna |
| Remoto | Streamable HTTP en `/mcp` | claude.ai, Claude móvil, Cowork | HTTPS 443 vía Tailscale Funnel + OAuth 2.1 |

El modo remoto está pensado para conectores que Anthropic ejecuta desde su infraestructura: exige HTTPS público, OAuth 2.1 y una contraseña de administración. El modo local no necesita nada de eso. Ver [http.md](http.md), [oauth.md](oauth.md) y [funnel.md](funnel.md).

El catálogo tiene **44 tools**: 24 de lectura y 20 de escritura. Cubre facturas de servicios, autos, entidades base, banca y, desde SPEC-154, pensión, alquileres, deudas, estados de cuenta, reparaciones del hogar, automatizaciones, alertas y configuración; SPEC-155 suma el registro de mano de obra e insumos de esas reparaciones, SPEC-156 la edición de hogares y SPEC-157 el alta de la reparación vacía y el movimiento de líneas entre reparaciones. Las **20 tools originales** son:

| Tool | Tipo | Qué hace |
|---|---|---|
| `bills_list` | lectura | Lista facturas de un servicio con filtros de período y estado |
| `entities_search` | lectura | Busca hogares, servicios, bancos o instituciones por nombre y devuelve sus IDs |
| `autos_list` | lectura | Lista y filtra autos por marca, modelo, placa o VIN, o pide uno por ID |
| `bills_generate` | escritura | Genera facturas de un rango de meses (por ejemplo, enero a la fecha) |
| `bills_pay` | escritura | Paga una factura, un mes, un rango o todas las pendientes hasta una fecha |
| `autos_create` | escritura | Crea un auto |
| `autos_update` | escritura | Reemplaza los datos de un auto existente |
| `autos_maintenance_register` | escritura | Registra un mantenimiento y sus líneas de mano de obra e insumos |
| `autos_inspection_register` | escritura | Registra una inspección mecánica y devuelve el cronograma |
| `autos_insurance_register` | escritura | Asocia un servicio de seguro a un auto |
| `homes_create` | escritura | Crea un hogar con nombre, dirección y uso |
| `institutions_create` | escritura | Crea una institución o proveedor y resuelve su categoría |
| `banks_create` | escritura | Crea un banco con código corto, país y estado |
| `services_create` | escritura | Crea un servicio en un hogar, con moneda e institución por nombre o ID |
| `bank_accounts_search` | lectura | Busca cuentas bancarias por alias, número, beneficiario o banco |
| `bank_accounts_create` | escritura | Crea una cuenta bancaria y resuelve el banco y la moneda |
| `bank_accounts_update` | escritura | Edita una cuenta existente sin cambiar el banco |
| `credit_cards_search` | lectura | Busca tarjetas por alias, últimos cuatro dígitos, titular o banco |
| `credit_cards_create` | escritura | Crea una tarjeta de crédito con los defaults del sistema |
| `credit_cards_update` | escritura | Edita una tarjeta existente sin cambiar el banco |

### Ola 1 de SPEC-154 (19 tools de lectura)

SPEC-154 agregó 19 tools de **solo lectura** sobre funcionalidades que ya existían en la web. Reutilizan endpoints HTTP, no agregan escrituras ni migraciones y **no dependen de la compuerta** `P40LA_MCP_ALLOW_WRITES`:

| Dominio | Tools | Documentación |
|---|---|---|
| Pensión | `pension_month_summary`, `pension_catalogs_list` | [tools/pension.md](tools/pension.md) |
| Alquileres | `rental_leases_list`, `rental_lease_get`, `rental_home_status`, `rental_portfolio_summary` | [tools/rental.md](tools/rental.md) |
| Deudas | `debt_bills_list` | [tools/debts.md](tools/debts.md) |
| Estados de cuenta | `credit_card_statements_list`, `credit_card_statements_get` | [tools/cards.md](tools/cards.md) |
| Reparaciones | `home_maintenance_list` | [tools/home-maintenance.md](tools/home-maintenance.md) |
| Facturas (vistas) | `bills_by_month`, `bills_summary`, `bills_projection` | [tools/bills.md](tools/bills.md) |
| Autos | `autos_inspection_alerts` | [tools/autos.md](tools/autos.md) |
| Entidades | `services_get` | [tools/entities.md](tools/entities.md) |
| Automatizaciones | `automation_accounts_list`, `automation_runs_list` | [tools/automation.md](tools/automation.md) |
| Alertas | `alerts_list` | [tools/alerts.md](tools/alerts.md) |
| Configuración | `settings_status` | [tools/settings.md](tools/settings.md) |

Las tools que pueden devolver datos personales o secretos aplican omisión o allowlist de campos: `alerts_list` omite el texto de voz, `settings_status` omite secretos, emails y chat IDs, `automation_*` omite identificadores de cuenta, URLs de webhook y payloads, y `rental_*` omite emails y teléfonos de primer nivel. El detalle de cada tool, con parámetros, ejemplos y errores, está en el [catálogo de tools](tools/README.md).

### Registro de reparaciones del hogar (SPEC-155, 2 tools de escritura)

SPEC-155 agrega dos tools de **escritura** sobre el módulo `home.maintenance`. Resuelven el mantenimiento del mismo día (lo reutilizan o lo crean con la línea en una sola transacción), deduplican por línea antes de escribir y respetan `dry_run` y la compuerta:

| Tool | Qué registra | Documentación |
|---|---|---|
| `home_maintenance_labor_register` | Una línea de mano de obra (`item_type=labor`) | [tools/home-maintenance.md](tools/home-maintenance.md) |
| `home_maintenance_supplies_register` | Una línea de insumos (`item_type=supplies`) con cantidad y unidad | [tools/home-maintenance.md](tools/home-maintenance.md) |

`home_maintenance_list` sigue siendo de lectura: gana los filtros `date`, `item_type`, `query` e `include_items` y el campo `truncated` para revisar lo existente antes de registrar.

### Alta y movimiento de reparaciones del hogar (SPEC-157, 2 tools de escritura)

SPEC-157 suma dos tools de **escritura** al módulo `home.maintenance`. `home_maintenance_create` da de alta la reparación **vacía** (sin líneas) y `home_maintenance_item_move` mueve líneas entre reparaciones del mismo hogar, con destino existente o creado en la misma transacción. Ambas respetan `dry_run` y la compuerta:

| Tool | Qué hace | Documentación |
|---|---|---|
| `home_maintenance_create` | Crea la reparación sin líneas; las líneas se agregan con las tools de registro o con un movimiento | [tools/home-maintenance.md](tools/home-maintenance.md) |
| `home_maintenance_item_move` | Mueve 1–100 líneas `labor` y/o `supplies`; puede crear el destino y borrar el origen vacío con `delete_source_if_empty=true` | [tools/home-maintenance.md](tools/home-maintenance.md) |

El backend crea el destino y mueve las líneas en una sola transacción: si el `UPDATE` no afecta todas las líneas devuelve `move_conflict` y revierte todo. `home_maintenance_create` no deduplica: repetir el alta crea otra reparación.

### Edición de hogares (SPEC-156, 1 tool de escritura)

SPEC-156 agrega `homes_update`, que reutiliza `PUT /api/homes/{id}`. Resuelve el hogar por `home_id` o por nombre y hace **edición parcial**: `name`, `address` y `usage` son opcionales y lo omitido conserva su valor actual. No tiene `dry_run` y depende de la compuerta de escritura.

| Tool | Tipo | Qué hace |
|---|---|---|
| `homes_update` | escritura | Edita un hogar existente; `address` vacío limpia la dirección, `usage` acepta `rental`, `inhabited` o `unused` y un contrato o reserva activa bloquea el cambio de uso |

## Requisitos

- Go 1.26 o superior para compilar.
- Acceso HTTP al servidor de p40la-ihost (`http://ihost.local:8088` por defecto).
- Una **API key global** del sistema. Se crea una vez con `scripts/create-api-key.sh` o desde `POST /api/api-key/regenerate` con una sesión autenticada.
- Opcional para el modo local: `P40LA_MCP_ALLOW_WRITES=true` para habilitar las tools que escriben.
- Para el modo remoto: una URL HTTPS pública (`P40LA_MCP_PUBLIC_URL`), un hash bcrypt (`p40la-mcp -hash-password`) y, si se publica desde el iHost, Tailscale Funnel habilitado por el usuario.

La key es global y tiene **todos los permisos**. No uses la contraseña del dashboard en la configuración del MCP: solo se necesita para provisionar la key. El modo remoto usa además una **contraseña de administración propia**, distinta de la del dashboard, que solo se guarda como hash bcrypt.

## Inicio rápido (local)

```bash
# 1. Compilar el binario
go build -o bin/p40la-mcp ./cmd/mcp-server

# 2. Crear/rotar la API key global (una sola vez; la contraseña no queda en el MCP)
EMAIL="demo@example.com" PASSWORD="demo-password" ./scripts/create-api-key.sh

# 3. Exportar la key y arrancar el MCP
export P40LA_API_KEY="p40la_..."
export P40LA_BASE_URL="http://ihost.local:8088"
./bin/p40la-mcp
```

Configuración típica en OpenCode (ver [configuration.md](configuration.md)):

```json
{
  "mcp": {
    "p40la": {
      "type": "local",
      "command": ["/home/demo/bin/p40la-mcp"],
      "environment": {
        "P40LA_BASE_URL": "http://ihost.local:8088",
        "P40LA_API_KEY": "{env:P40LA_API_KEY}",
        "P40LA_MCP_ALLOW_WRITES": "false"
      }
    }
  }
}
```

## Inicio rápido (remoto)

Para claude.ai o Claude móvil no se configura un comando local: se publica `https://demo-node.example.ts.net/mcp` y se agrega como conector. El resumen es:

```bash
# 1. Hash de la contraseña de administración
read -rs -p "Contraseña: " P40LA_MCP_PASSWORD; echo
HASH=$(printf '%s\n' "$P40LA_MCP_PASSWORD" | ./bin/p40la-mcp -hash-password)
unset P40LA_MCP_PASSWORD

# 2. Arrancar en modo HTTP (solo loopback)
P40LA_BASE_URL="http://ihost.local:8088" \
P40LA_API_KEY="demo-api-key" \
P40LA_MCP_TRANSPORT=http \
P40LA_MCP_PUBLIC_URL="https://demo-node.example.ts.net" \
P40LA_MCP_OAUTH_PASSWORD_HASH="$HASH" \
P40LA_MCP_REMOTE_WRITES=false \
./bin/p40la-mcp

# 3. Publicar 443 → 127.0.0.1:8090 (requiere Funnel habilitado en la tailnet)
tailscale funnel --bg 8090
```

El paso 3 es una decisión explícita del usuario y deja el endpoint en Internet. Seguí el runbook de [funnel.md](funnel.md) y el checklist de [security.md](security.md). En el iHost, el paso 3 se hace en el nodo dedicado, **nunca** en el nodo del dashboard: ver [ihost.md](ihost.md). Claude Desktop local sigue usando **stdio**: el modo HTTP es para conectores remotos.

Si corrés el binario fuera del contenedor, definí `P40LA_MCP_STATE_DIR` en un directorio escribible por tu usuario (por ejemplo `/tmp/p40la-mcp-state`); el default `/data` es para la imagen Docker.

## Despliegue 24/7 en el iHost

Para dejarlo corriendo sin depender de una laptop, el MCP se despliega en el iHost con un **nodo Tailscale dedicado**. El nodo que ya sirve el dashboard (`ihost.example.ts.net`) conserva su Serve en 443 y **no se modifica**; el MCP usa otro hostname (`p40la-mcp.example.ts.net`) con su propio Funnel y su propio estado.

El runbook completo —prerrequisitos, API key, hash OAuth, volúmenes, alta por UI, verificación, conector de claude.ai y rollback— está en [ihost.md](ihost.md). Los assets de referencia viven en [`deploy/mcp/`](https://github.com/paulomcnally/p40la-ihost/tree/main/deploy/mcp).

## Flujo seguro de escritura

1. En facturas, autos y reparaciones del hogar, el agente llama la tool de escritura con `dry_run=true` (el valor por defecto) y muestra el resumen: períodos, montos, datos del registro, ítems omitidos y, en reparaciones, las líneas que ya existen ese día y, en un movimiento, las líneas que se mueven, las que quedan y si el origen se borraría. En entidades y banca, que no tienen `dry_run`, muestra antes la entidad, el banco y la moneda resueltos, el cuerpo que se enviará y si ya existe un duplicado. En `homes_update`, muestra el hogar resuelto y los campos que cambian: la tool completa los omitidos con el valor actual.
2. El usuario confirma.
3. El agente ejecuta: en facturas, autos y reparaciones repite con `dry_run=false`; las tools de entidades y banca se llaman directamente. La compuerta `P40LA_MCP_ALLOW_WRITES=true` debe estar activa; si el transporte es HTTP, también `P40LA_MCP_REMOTE_WRITES=true`.
4. El agente reporta el resultado y verifica con una lectura.

En facturas, las operaciones son idempotentes por período: repetir una generación no duplica facturas y pagar una factura ya pagada devuelve `skipped`. En autos, solo `autos_update` es idempotente: repetir un alta de auto, mantenimiento o inspección crea otro registro, y repetir un seguro devuelve `already_exists`. En reparaciones del hogar, las tools de registro buscan líneas equivalentes del mismo día: repetir devuelve `duplicate` con el registro existente y no crea otra fila, salvo `allow_duplicate=true`; dos escrituras concurrentes podrían pasar el chequeo. `home_maintenance_create` no deduplica: repetir crea otra reparación. `home_maintenance_item_move` no es idempotente: repetir el mismo movimiento devuelve `item_not_found` porque las líneas ya están en el destino. En entidades y banca, las ediciones (`homes_update`, `bank_accounts_update`, `credit_cards_update`) son idempotentes y las altas comprueban duplicados: repetir un alta devuelve `already_exists` con el `id` existente, sin crear otro registro.

Antes de un alta o de editar un hogar, `entities_search` permite verificar si la entidad ya existe y obtener su `id`; es de solo lectura y no requiere la compuerta.

Las tools de alta (`homes_create`, `institutions_create`, `banks_create`, `services_create`) **no tienen `dry_run`**: escriben en una sola llamada. Mostrá al usuario los datos que se van a crear y pedí confirmación antes de invocarlas; si el nombre ya existe, devuelven `already_exists` con el `id` de la entidad existente. `homes_update` tampoco tiene `dry_run`: la edición es parcial, se ejecuta en una sola llamada después de la compuerta, rechaza un nombre que ya pertenece a otro hogar y, si el hogar tiene un contrato o reserva activa, el cambio de uso devuelve `home_has_active_rental`; `name` y `address` se pueden editar sin cambiar `usage`. Ver [tools/entities.md](tools/entities.md).

## Documentación

- [API key global](api-key.md)
- [Configuración](configuration.md)
- [Arquitectura](architecture.md)
- [Seguridad](security.md)
- [Operación](operations.md)
- [Desarrollo de nuevas tools](development.md)
- [Servidor HTTP remoto](http.md)
- [OAuth 2.1](oauth.md)
- [Tailscale Funnel](funnel.md)
- [Despliegue 24/7 en el iHost](ihost.md)
- [Catálogo de tools](tools/README.md)
- [Tools de facturas](tools/bills.md)
- [Tools de autos](tools/autos.md)
- [Tools de entidades](tools/entities.md)
- [Tools de banca](tools/banking.md)
- [Tools de pensión](tools/pension.md)
- [Tools de alquileres](tools/rental.md)
- [Tools de deudas](tools/debts.md)
- [Tools de estados de cuenta](tools/cards.md)
- [Tools de mantenimiento del hogar](tools/home-maintenance.md)
- [Tools de automatizaciones](tools/automation.md)
- [Tools de alertas](tools/alerts.md)
- [Tools de configuración](tools/settings.md)
- [Impacto y contrato de tools](tools/impact.md) — manifiesto `tools/impact.json`, golden `tools/contracts.json` y test backend↔MCP

## Publicación web

Esta carpeta es Markdown plano con frontmatter de Docusaurus (`title`, `sidebar_position`, `description`) y `_category_.json`. Para publicarla, un sitio Docusaurus puede usar `docs/mcp` como `path` de un plugin de docs o copiar la carpeta dentro de su `docs/`. No se requiere build de Node en este repositorio.
