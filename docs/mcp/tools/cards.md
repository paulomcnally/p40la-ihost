---
title: "Tools de tarjetas de crédito"
sidebar_position: 9
description: "Referencia de credit_card_statements_list y credit_card_statements_get"
---

# Tools de tarjetas de crédito

Tools de **solo lectura** sobre estados de cuenta de tarjetas (SPEC-154). No crean estados, no importan PDFs y no registran pagos.

## `credit_card_statements_list`

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `card_id` | integer | Sí | ID de la tarjeta |

```json
{ "card_id": 7 }
```

```json
{
  "card_id": 7,
  "count": 1,
  "statements": [
    { "id": 1, "period_end": "2026-10-31", "status": "unpaid", "new_balance": 500.00, "transaction_count": 3 }
  ]
}
```

## `credit_card_statements_get`

Devuelve el estado con su resumen por tipo de movimiento y la conciliación calculada.

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `statement_id` | integer | Sí | ID del estado de cuenta |

```json
{ "statement_id": 9 }
```

```json
{
  "id": 9,
  "card_id": 7,
  "status": "unpaid",
  "total_payment": 500.00,
  "new_balance": 520.00,
  "reconciliation": {
    "cash_base_balanced": true,
    "balance_base_balanced": false,
    "balanced": false
  },
  "summary": [{ "type": "purchase", "count": 3, "amount": 520.00 }]
}
```

## Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `card_id` o `statement_id` ≤ 0 |
| `not_found` | La tarjeta o el estado no existen |
| `auth_failed` | API key inválida o ausente |

## Notas

- La conciliación es de solo lectura: no existe acción de conciliar.
- El pago de tarjeta **no** está expuesto: la API no tiene un endpoint atómico de pago.
