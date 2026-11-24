---
title: "Tools de facturas"
sidebar_position: 2
description: "Referencia de bills_list, bills_generate, bills_pay, bills_by_month, bills_summary y bills_projection"
---

# Tools de facturas

Las tres tools de escritura y la de listado operan sobre facturas de un **servicio**. El `service_id` es el mismo que aparece en la URL de la UI, por ejemplo `/services/bills/12?tab=facturas` → `service_id: 12`.

Las tres tools de **vistas globales** (`bills_by_month`, `bills_summary` y `bills_projection`) no reciben `service_id`: agregan o proyectan las facturas de **todos los servicios**, con filtro opcional por vivienda. Son de solo lectura.

> El contrato visible de estas tools está congelado en [`contracts.json`](contracts.json). Un cambio intencional de nombre, descripción, schema o anotaciones exige regenerarlo con `go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts` y declararlo en `## Impacto en tools MCP` de la spec. Ver [impact.md](impact.md).

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
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red o del servidor |

## `bills_by_month`

Lista las facturas de **todos los servicios** de un mes: es la vista global multi-servicio de la UI. Solo lectura; no genera facturas.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `period` | string `YYYY-MM` | No | Período a consultar; default el mes actual en `P40LA_TIMEZONE` |
| `home_id` | integer | No | Filtra por vivienda; debe ser mayor que `0` |

### Ejemplo

```json
{
  "period": "2026-11",
  "home_id": 3
}
```

### Salida

La respuesta es un **array** de facturas; no viene envuelta en un objeto.

```json
[
  {
    "id": 1001,
    "service_id": 12,
    "service_name": "Servicio Demo",
    "institution": "Proveedor Demo",
    "home_id": 3,
    "home_name": "Hogar Demo",
    "icon_key": "internet",
    "currency_id": 1,
    "currency_code": "NIO",
    "currency_symbol": "C$",
    "frequency": "monthly",
    "billing_day": 5,
    "active": true,
    "year": 2026,
    "month": 11,
    "period_year": 2026,
    "period_month": 11,
    "period_date": "2026-11-05",
    "date_estimated": false,
    "amount": 1000.00,
    "invoice_number": "F000000000000001",
    "status": "pending",
    "due_date": "2026-11-05"
  }
]
```

### Reglas

- Cada factura incluye el **período efectivo** (`period_year`, `period_month`, `period_date`), calculado por la API a partir del vencimiento. `date_estimated` es `true` cuando la fecha efectiva se infirió.
- `billing_day` puede ser `null` si el servicio no define día de facturación.
- `invoice_number`, `drive_url`, `issue_date`, `due_date`, `paid_at` y `payment_reference` aparecen solo cuando tienen valor.
- `home_id` filtra los servicios de esa vivienda; sin filtro se devuelven todos.
- Una factura **anual** (guardada con mes `0`) se lista en **enero**.
- El orden es por nombre del servicio y `id` de factura.
- No genera facturas ni modifica estados: para eso existen `bills_generate` y `bills_pay`.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_period` | `period` no tiene formato `YYYY-MM`, el año está fuera de `1900..2100` o el mes fuera de `1..12` |
| `invalid_request` | `home_id` es `0` o negativo |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red o del servidor |

## `bills_summary`

Resume las facturas de servicios por período entre dos meses, agrupadas por **período y moneda**: total, pagado, pendiente y cantidad. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `from` | string `YYYY-MM` | Sí | Período inicial (inclusive) |
| `to` | string `YYYY-MM` | Sí | Período final (inclusive) |
| `home_id` | integer | No | Filtra por vivienda; debe ser mayor que `0` |

### Ejemplo

```json
{
  "from": "2026-01",
  "to": "2026-06"
}
```

### Salida

```json
{
  "from": "2026-01",
  "to": "2026-06",
  "periods": [
    {
      "period": "2026-01",
      "year": 2026,
      "month": 1,
      "currency_code": "NIO",
      "currency_symbol": "C$",
      "total": 3000.00,
      "paid": 2000.00,
      "pending": 1000.00,
      "count": 3
    }
  ]
}
```

### Reglas

- Cada fila es un período + moneda; los importes **nunca** se suman entre monedas.
- `total = paid + pending` para cada fila; `count` es la cantidad de facturas del período y moneda.
- Solo aparecen los períodos con facturas: un mes sin facturas no se informa con ceros.
- El rango es inclusivo y se valida antes de llamar a la API: si `from` es posterior a `to`, la tool devuelve `invalid_range` sin consultar.
- No pagina ni capa la cantidad de filas; un rango muy amplio puede acercarse al límite de 1 MiB de respuesta.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_period` | `from` o `to` no tienen formato `YYYY-MM`, o el año/mes están fuera de rango |
| `invalid_range` | `from` es posterior a `to` |
| `invalid_request` | `home_id` es `0` o negativo |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red o del servidor |

## `bills_projection`

Proyecta los próximos meses separando **facturas pendientes confirmadas** de **estimaciones** de servicios recurrentes. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `from` | string `YYYY-MM` | Sí | Mes inicial de la proyección |
| `months` | integer | No | Cantidad de meses, `1..24`; default `6` |
| `home_id` | integer | No | Filtra por vivienda; debe ser mayor que `0` |

### Ejemplo

```json
{
  "from": "2026-11",
  "months": 3
}
```

### Salida

```json
{
  "from": "2026-11",
  "months": 3,
  "periods": [
    {
      "period": "2026-11",
      "year": 2026,
      "month": 11,
      "currency_code": "NIO",
      "currency_symbol": "C$",
      "confirmed": 1000.00,
      "estimated": 500.00,
      "total": 1500.00,
      "services": [
        {
          "service_id": 12,
          "name": "Servicio Demo",
          "amount": 1000.00,
          "source": "bill",
          "currency_code": "NIO"
        },
        {
          "service_id": 13,
          "name": "Internet Demo",
          "amount": 500.00,
          "source": "estimate",
          "currency_code": "NIO"
        }
      ]
    }
  ],
  "estimate_note": true
}
```

### Reglas

- `confirmed` suma las facturas **pendientes** existentes del período; `estimated` suma el `suggested_amount` de los servicios recurrentes activos que todavía no tienen factura en ese período.
- `services[].source` es `bill` (factura existente) o `estimate` (estimación).
- Un servicio anual solo se estima en **enero**; los servicios sin `suggested_amount` mayor que `0` no se estiman.
- Se omiten los períodos y monedas sin montos: `periods` puede ser más corto que `months`.
- `estimate_note` es siempre `true`: los valores de `estimated` no son facturas emitidas.
- `months` fuera de `1..24` devuelve `invalid_request` sin llamar a la API.
- No escribe: no genera las facturas proyectadas.

## Flujo recomendado para un agente

1. `bills_by_month` o `bills_list` para conocer el estado; `bills_summary` para el histórico y `bills_projection` para los próximos meses.
2. `bills_generate` o `bills_pay` con `dry_run` (default).
3. Mostrar el resumen al usuario y pedir confirmación explícita.
4. Repetir con `dry_run=false`.
5. Reportar el resultado y los omitidos con su motivo.

No reintentes una escritura que falló parcialmente sin volver a previsualizar: la operación es idempotente, pero el segundo intento puede incluir períodos que ya se crearon.

Las tres vistas son de solo lectura y no necesitan la compuerta de escritura: sirven para diagnosticar antes de generar o pagar. `bills_projection` **no** crea las facturas que proyecta.
