---
title: "Tools de estados de cuenta"
sidebar_position: 9
description: "Referencia de credit_card_statements_list y credit_card_statements_get"
---

# Tools de estados de cuenta

Las dos tools son de **solo lectura** y cubren los **estados de cuenta de tarjetas de crédito** (`bank.creditcard`): listado de cortes por tarjeta y detalle de un corte con su resumen por tipo de movimiento y su conciliación. No importan estados, no crean movimientos, no pagan y no dependen de la compuerta de escritura.

El `card_id` y el `statement_id` son los mismos IDs que usa la UI. Para llegar a la tarjeta se puede usar antes `credit_cards_search` de [banking.md](banking.md).

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `credit_card_statements_list` | Lectura | 1 | Cortes de una tarjeta, del más reciente al más antiguo |
| `credit_card_statements_get` | Lectura | 1 | Detalle de un corte con resumen por tipo y conciliación |

## `credit_card_statements_list`

Lista los estados de cuenta de una tarjeta, ordenados por fecha de corte descendente, con la cantidad de movimientos de cada corte. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `card_id` | integer | Sí | ID de la tarjeta de crédito; debe ser mayor que `0` |

### Ejemplo

```json
{
  "card_id": 7
}
```

### Salida

```json
{
  "card_id": 7,
  "count": 2,
  "statements": [
    {
      "id": 9,
      "card_id": 7,
      "period_start": "2026-10-06",
      "period_end": "2026-11-05",
      "min_payment_due_date": "2026-11-20",
      "full_payment_due_date": "2026-11-25",
      "previous_balance": 500.00,
      "interest_current": 12.50,
      "mora_amount": 0,
      "arrears_count": 0,
      "min_payment": 25.00,
      "total_payment": 300.00,
      "new_balance": 200.00,
      "status": "unpaid",
      "source": "manual",
      "transaction_count": 3
    }
  ]
}
```

### Reglas

- El orden es por `period_end` descendente: el primer elemento es el corte más reciente.
- `transaction_count` lo calcula la API con un `COUNT` de los movimientos del corte; los movimientos **no** se listan.
- `status` del corte es `unpaid`, `partial` o `paid`.
- `source` indica el origen del corte: `manual` o `import`. `source_module_id` aparece solo cuando lo cargó un módulo.
- `notes` se devuelve si el corte tiene notas.
- Los importes vienen en moneda local y, cuando el banco los registró, también en dólares (`previous_balance_usd`, `min_payment_usd`, `new_balance_usd`, etc.).
- `count` es la cantidad de cortes devueltos.
- Sin cortes devuelve `count: 0` y `statements: []` (no es un error).
- Una tarjeta inexistente devuelve `credit_card_not_found` (la API responde `404`).

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `card_id` es `0` o negativo |
| `credit_card_not_found` | La tarjeta no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## `credit_card_statements_get`

Devuelve un estado de cuenta completo con su **resumen por tipo de movimiento** y su **conciliación** contra los totales impresos por el banco. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `statement_id` | integer | Sí | ID del estado de cuenta; debe ser mayor que `0` |

### Ejemplo

```json
{
  "statement_id": 9
}
```

### Salida

```json
{
  "id": 9,
  "card_id": 7,
  "period_start": "2026-10-06",
  "period_end": "2026-11-05",
  "min_payment_due_date": "2026-11-20",
  "full_payment_due_date": "2026-11-25",
  "previous_balance": 500.00,
  "interest_current": 12.50,
  "mora_amount": 0,
  "arrears_count": 0,
  "min_payment": 25.00,
  "total_payment": 300.00,
  "new_balance": 200.00,
  "status": "unpaid",
  "source": "manual",
  "transaction_count": 3,
  "summary": [
    {
      "type": "purchase",
      "amount_local": 300.00,
      "amount_usd": 8.22,
      "count": 2
    },
    {
      "type": "payment",
      "amount_local": 100.00,
      "amount_usd": 2.74,
      "count": 1
    }
  ],
  "reconciliation": {
    "expected_cash": 200.00,
    "printed_cash": 200.00,
    "difference_cash": 0,
    "expected_balance": 200.00,
    "printed_balance": 200.00,
    "difference_balance": 0,
    "cash_applicable": true,
    "balance_applicable": true,
    "balanced": true
  }
}
```

### Reglas

- `summary` agrupa los movimientos por `type` (`purchase`, `payment`, `payment_bonus`, `cash_advance`, `cash_advance_fee`, `interest_current`, `interest_late`, `fee`, `tax`, `bonus`, `refund`, `adjustment` u `other`) y suma en moneda local (`amount_local`) y en dólares (`amount_usd`).
- `reconciliation` compara los totales impresos (`printed_*`) con los calculados desde los movimientos (`expected_*`). `difference_*` es la resta y `balanced` indica si ambas bases cuadran.
- Hay **dos bases**: `cash` (pago de contado: saldo anterior + cargos − abonos) y `balance` (saldo al corte, que agrega el interés bonificable). `cash_applicable` y `balance_applicable` indican si el banco imprimió cada total; una base no aplicable no debe interpretarse como error.
- El detalle incluye los agregados del corte (`interest_bonifiable`, `mora_amount`, `over_limit_amount`, `points_*`, `*_usd`) cuando el banco los registró.
- `status`, `source` y `notes` son los del corte.
- Un `statement_id` inexistente devuelve `statement_not_found` (la API responde `404`).

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `statement_id` es `0` o negativo |
| `statement_not_found` | El estado de cuenta no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## Privacidad y datos ocultos

Las tools no devuelven el número completo de la tarjeta, el titular ni el CVV: trabajan con `card_id` y `statement_id`. Tampoco descargan PDFs ni comprobantes.

`notes` puede contener texto libre cargado por el usuario. No lo reproduzcas completo en respuestas extensas ni en logs.

## Flujo recomendado para un agente

1. Si no conocés la tarjeta, buscarla con `credit_cards_search` y quedarte con su `id`.
2. Llamar `credit_card_statements_list` con `card_id` para ubicar el corte.
3. Llamar `credit_card_statements_get` con el `id` del corte para el resumen y la conciliación.
4. Reportar montos por moneda y aclarar si la conciliación está `balanced`; si una base no es aplicable, decilo explícitamente.
