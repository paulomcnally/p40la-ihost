---
title: "Tools de facturas"
sidebar_position: 2
description: "Referencia de bills_list, bills_generate y bills_pay"
---

# Tools de facturas

Las tres tools operan sobre facturas de un **servicio**. El `service_id` es el mismo que aparece en la URL de la UI, por ejemplo `/services/bills/12?tab=facturas` → `service_id: 12`.

## `bills_list`

Lista facturas activas de un servicio. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `service_id` | integer | Sí | ID del servicio |
| `from` | string `YYYY-MM` | No | Período inicial (inclusive) |
| `to` | string `YYYY-MM` | No | Período final (inclusive) |
| `status` | string | No | `pending`, `paid` o `all` (default `all`) |

### Ejemplo

```json
{
  "service_id": 12,
  "from": "2026-01",
  "to": "2026-11",
  "status": "pending"
}
```

### Salida

```json
{
  "service_id": 12,
  "service_name": "Servicio Demo",
  "from": "2026-01",
  "to": "2026-11",
  "status": "pending",
  "count": 11,
  "pending": 11,
  "paid": 0,
  "total_amount": 11000.00,
  "bills": [
    {
      "id": 1001,
      "year": 2026,
      "month": 1,
      "period": "2026-01",
      "amount": 1000.00,
      "status": "pending"
    }
  ]
}
```

El total es la suma de las facturas listadas, sin convertir moneda. Un servicio tiene una sola moneda; el código se informa en las operaciones masivas.

## `bills_generate`

Genera facturas para un rango inclusivo de meses. Es idempotente: omite los períodos que ya tienen factura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `service_id` | integer | Sí | ID del servicio |
| `from` | string `YYYY-MM` | Sí | Período inicial |
| `to` | string `YYYY-MM` | No | Período final; default mes actual en `P40LA_TIMEZONE` |
| `amount` | number | No | Monto de cada factura; si se omite se resuelve automáticamente |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

### Resolución del monto

1. `amount` explícito, si se envía.
2. Monto de la factura activa más reciente del servicio.
3. `suggested_amount` del servicio.
4. Si el resultado es `0` y no hay `amount`, error `amount_unresolved`.

### Reglas

- Servicio mensual: una factura por mes.
- Servicio anual: solo enero, con `month=0` en la base.
- No se generan meses futuros.
- Máximo 36 meses por llamada.
- Período con factura activa: `skipped` (`exists_pending` o `exists_paid`).
- Período con factura soft-deleted: `skipped` (`deleted_slot`) con el `bill_id`; no se resucita automáticamente.
- Servicio inactivo: error `service_inactive` (la tool no expone `allow_inactive`).

### Ejemplo: enero a noviembre de 2026

```json
{
  "service_id": 12,
  "from": "2026-01",
  "to": "2026-11",
  "dry_run": true
}
```

Salida esperada si no existe ninguna factura:

```json
{
  "service_id": 12,
  "service_name": "Servicio Demo",
  "from": "2026-01",
  "to": "2026-11",
  "dry_run": true,
  "amount": 1000.00,
  "currency_code": "NIO",
  "created": 0,
  "skipped": 0,
  "failed": 0,
  "items": [
    {
      "year": 2026,
      "month": 1,
      "period": "2026-01",
      "amount": 1000.00,
      "status": "planned"
    }
  ]
}
```

Con `dry_run=false` y la compuerta activa, el mismo rango devuelve `created: 11` y cada ítem pasa a `status: "created"` con su `bill_id`. Repetirlo devuelve `created: 0`, `skipped: 11`.

## `bills_pay`

Paga facturas pendientes. Es idempotente: las ya pagadas se omiten.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `service_id` | integer | Sí | ID del servicio |
| `bill_ids` | integer[] | Selector | IDs exactos a pagar |
| `month` | string `YYYY-MM` | Selector | Paga el mes indicado |
| `from` + `to` | string `YYYY-MM` | Selector | Paga el rango inclusivo |
| `all_pending_up_to` | string `YYYY-MM` | Selector | Paga todas las pendientes hasta ese mes |
| `paid_at` | string `YYYY-MM-DD` | No | Fecha de pago; default hoy; no puede ser futura |
| `payment_reference` | string | No | Referencia interna del pago |
| `drive_url` | string | No | URL de Google Drive del comprobante |
| `dry_run` | boolean | No | `true` por defecto: previsualiza sin escribir |

Se debe indicar **exactamente un** selector. `from` y `to` cuentan como uno solo y deben venir juntos.

### Reglas

- Solo facturas `pending` y activas.
- Períodos futuros: `skipped` (`future_period`).
- Facturas ya pagadas: `skipped` (`already_paid`).
- Una factura anual de enero se puede seleccionar con `month: "YYYY-01"`.
- El pago conserva el historial y la transacción de presupuesto del sistema.

### Ejemplo: pagar enero

```json
{
  "service_id": 12,
  "month": "2026-01",
  "paid_at": "2026-11-23",
  "dry_run": true
}
```

### Ejemplo: pagar todo lo pendiente hasta la fecha

```json
{
  "service_id": 12,
  "all_pending_up_to": "2026-11",
  "paid_at": "2026-11-23",
  "dry_run": true
}
```

### Salida

```json
{
  "service_id": 12,
  "service_name": "Servicio Demo",
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
      "period": "2026-01",
      "amount": 1000.00,
      "status": "paid"
    }
  ]
}
```

## Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `service_id` ausente o inválido |
| `invalid_range` | Formato de período, rango invertido, futuro o más de 36 meses |
| `invalid_selector` | Cero o más de un selector de pago, o rango incompleto |
| `invalid_paid_at` | Fecha inválida o futura |
| `service_not_found` | El servicio no existe |
| `service_inactive` | Servicio inactivo al generar |
| `amount_unresolved` | No hay monto explícito, última factura ni monto sugerido |
| `deleted_slot` | El período está ocupado por una factura eliminada |
| `writes_disabled` | `dry_run=false` sin `P40LA_MCP_ALLOW_WRITES=true` |
| `auth_failed` | Credenciales o cookie inválidas |
| `upstream_error` | Error de red o del servidor |

## Flujo recomendado para un agente

1. `bills_list` para conocer el estado.
2. `bills_generate` o `bills_pay` con `dry_run` (default).
3. Mostrar el resumen al usuario y pedir confirmación explícita.
4. Repetir con `dry_run=false`.
5. Reportar el resultado y los omitidos con su motivo.

No reintentes una escritura que falló parcialmente sin volver a previsualizar: la operación es idempotente, pero el segundo intento puede incluir períodos que ya se crearon.
