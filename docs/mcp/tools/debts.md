---
title: "Tools de deudas"
sidebar_position: 8
description: "Referencia de debt_bills_list"
---

# Tools de deudas

La tool consulta las **cuotas** de las deudas registradas en el módulo de Deudas. Es de **solo lectura**: no reconcilia deudas, no genera cuotas nuevas, no paga y no depende de la compuerta de escritura.

> Esta ola **no** expone un listado de deudas (`debts_list` quedó diferida). El `debt_id` y el `period` se obtienen de la UI: `/deudas` para el ID de la deuda y `/deudas/calendario` para la vista por mes.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `debt_bills_list` | Lectura | 1 | Lista cuotas por deuda o por mes, con estado, vencimiento y monto |

## `debt_bills_list`

Lista las cuotas de una deuda o las cuotas de un mes, según el modo elegido. Solo lectura. Hay **exactamente dos modos** y se debe indicar uno solo:

| Modo | Campo | Endpoint |
|---|---|---|
| Por deuda | `debt_id` | `GET /api/debts/{id}/bills` |
| Por mes | `period` | `GET /api/debt-bills?year=&month=` |

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `debt_id` | integer | Modo deuda | ID de la deuda; debe ser mayor que `0` |
| `period` | string `YYYY-MM` | Modo mes | Mes a consultar |

Enviar los dos campos o ninguno devuelve `invalid_request`. Un `period` vacío (`""`) se trata como ausente.

### Ejemplo: por deuda

```json
{
  "debt_id": 4
}
```

```json
{
  "mode": "debt",
  "debt_id": 4,
  "count": 2,
  "bills": [
    {
      "id": 10,
      "debt_id": 4,
      "debt_description": "Deuda Demo",
      "institution_name": "Banco Demo",
      "currency_code": "NIO",
      "installment_number": 1,
      "due_date": "2026-11-15",
      "amount": 1000.00,
      "status": "pending",
      "has_invoice": true,
      "has_receipt": false
    },
    {
      "id": 11,
      "debt_id": 4,
      "debt_description": "Deuda Demo",
      "institution_name": "Banco Demo",
      "currency_code": "NIO",
      "installment_number": 2,
      "due_date": "2026-12-15",
      "amount": 1000.00,
      "status": "paid",
      "paid_at": "2026-12-14T18:00:00Z",
      "has_invoice": false,
      "has_receipt": true
    }
  ]
}
```

### Ejemplo: por mes

```json
{
  "period": "2026-11"
}
```

```json
{
  "mode": "month",
  "period": "2026-11",
  "count": 1,
  "bills": [
    {
      "id": 10,
      "debt_id": 4,
      "debt_description": "Deuda Demo",
      "institution_name": "Banco Demo",
      "currency_code": "NIO",
      "installment_number": 1,
      "due_date": "2026-11-15",
      "amount": 1000.00,
      "status": "pending",
      "has_invoice": true,
      "has_receipt": false
    }
  ]
}
```

### Reglas

- `mode` indica qué modo se usó: `debt` o `month`. En modo deuda devuelve `debt_id`; en modo mes devuelve `period` normalizado (`YYYY-MM`).
- La vista por mes filtra por **fecha de vencimiento** de la cuota, no por el período del calendario de la deuda.
- Una deuda inexistente **no es un error**: devuelve `count: 0` y `bills: []`.
- Los estados de cuota son `pending` y `paid`.
- `installment_number` es el número de cuota dentro de la deuda y `due_date` su vencimiento (`YYYY-MM-DD`).
- `has_invoice` y `has_receipt` indican si existen los PDFs de factura y comprobante en el servidor; no se descargan ni se devuelven como archivos.
- `count` es la cantidad de cuotas devueltas.
- La tool **no reconcilia** deudas ni crea cuotas faltantes: para eso existe la UI.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | Se enviaron `debt_id` y `period`, no se envió ninguno, o `debt_id` es `0` o negativo |
| `invalid_period` | `period` no tiene formato `YYYY-MM`, el año está fuera de `1900..2100` o el mes fuera de `1..12` |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## Privacidad y datos ocultos

La tool elimina `invoice_file` y `receipt_file` si aparecen en la respuesta (`SEC-NO-PII`); la API actual informa la existencia de los PDFs con `has_invoice` y `has_receipt`, sin rutas ni nombres de archivo.

`payment_reference` se devuelve tal como está registrada. Puede contener una referencia bancaria: no la reproduzcas en logs ni en capturas.

## Flujo recomendado para un agente

1. Pedir al usuario el `debt_id` (o el mes) que quiere revisar; esta ola no incluye un listado de deudas.
2. Llamar `debt_bills_list` con **uno** de los dos modos.
3. Reportar la cantidad de cuotas, los estados y los vencimientos. No sumar montos de monedas distintas (`currency_code` puede variar entre deudas).
4. Si el usuario quiere pagar una cuota, indicar que la acción se hace desde la UI: esta tool no paga.
