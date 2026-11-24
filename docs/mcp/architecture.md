---
title: "Arquitectura del MCP"
sidebar_position: 2
description: "Componentes, flujo y decisiones del servidor MCP de p40la-ihost"
---

# Arquitectura

## Vista general

```
Cliente MCP (OpenCode / Claude Desktop)
        │  stdio: JSON-RPC 2.0
        ▼
┌─────────────────────────────┐
│ p40la-mcp                   │
│ cmd/mcp-server              │
│ internal/mcp/config         │
│ internal/mcp/api            │
│ internal/mcp/registry       │
│ internal/mcp/tools/bills    │
└──────────────┬──────────────┘
               │ HTTP + cookie de sesión
               ▼
┌─────────────────────────────┐
│ p40la-ihost server          │
│ POST /api/login             │
│ GET  /api/services/{id}     │
│ GET  /api/services/{id}/bills
│ POST /api/services/{id}/bills/generate
│ POST /api/services/{id}/bills/pay
└──────────────┬──────────────┘
               │ BillService → BillStorage
               ▼
        SQLite (bills, bill_history)
```

## Componentes

| Componente | Ubicación | Responsabilidad |
|---|---|---|
| Entrypoint | `cmd/mcp-server/main.go` | Configuración, registro de tools y transporte stdio |
| Config | `internal/mcp/config` | Variables de entorno, validación y zona horaria |
| Cliente API | `internal/mcp/api` | Login, cookie en memoria, llamadas HTTP y errores tipados |
| Registry | `internal/mcp/registry` | Punto de montaje de conjuntos de tools |
| Tools de facturas | `internal/mcp/tools/bills` | `bills_list`, `bills_generate`, `bills_pay` |
| Registro global | `internal/mcp/tools/all` | Único lugar que importa los paquetes de tools |
| Backend masivo | `internal/services/bill_bulk.go` | Reglas de generación y pago por rango |
| Handlers HTTP | `internal/api/bill_bulk_handlers.go` | Parseo, códigos de estado y respuesta |

## Por qué el MCP no accede a SQLite

- La DB del iHost usa WAL y es producción: un proceso externo escribiendo el archivo es riesgoso.
- Saltaría validaciones de negocio, el historial de facturas y la transacción de presupuesto al pagar.
- La API es el contrato estable; el MCP queda desacoplado del esquema.

## Lógica masiva en el backend

La expansión de rangos, la resolución de montos, la detección de duplicados y la decisión de qué pagar viven en `BillService`:

- `GenerateForService`: genera períodos inclusivos, omite existentes y reporta slots soft-deleted.
- `PayForService`: aplica un selector único, omite pagadas y períodos futuros.

El MCP solo traduce la tool a una llamada HTTP y presenta el resumen. Así la UI puede reutilizar los mismos endpoints en el futuro.

## Idempotencia

- `bills` tiene `UNIQUE(service_id, year, month)`, incluido el borrado lógico.
- La generación consulta `FindByServicePeriodIncludingDeleted` antes de crear.
- Repetir una generación devuelve `skipped` con motivo `exists_pending`, `exists_paid` o `deleted_slot`.
- Repetir un pago devuelve `skipped` con motivo `already_paid`.
- No hay reintentos automáticos de escrituras; solo se reintenta una vez tras un `401` de autenticación, que ocurre antes de ejecutar el handler.

## Auditoría

Las altas y pagos hechos por el MCP envían el header `X-P40LA-Source: mcp`. El backend lo valida y registra `bill_history.source = mcp`. La UI sigue registrando `dashboard` y los webhooks `webhook`.

La migración `0058_bill_history_source_mcp` habilita el nuevo valor en el `CHECK` de `source`; es reversible y no crea tablas nuevas.

## Límites

- Máximo 36 períodos por generación.
- No se generan meses futuros.
- Las llamadas al backend son secuenciales.
- El binario MCP no se ejecuta dentro del proceso del servidor ni del contenedor por defecto.
