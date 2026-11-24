---
title: "Tools de deudas"
sidebar_position: 8
description: "Referencia de debt_bills_list"
---

# Tools de deudas

Tool de **solo lectura** sobre cuotas de deuda (SPEC-154). No paga cuotas, no crea deudas y **no reconcilia**: a diferencia de `GET /api/debts`, no genera cuotas faltantes.

## `debt_bills_list`

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `debt_id` | integer | Uno de los dos | Lista las cuotas de una deuda |
| `period` | string `YYYY-MM` | Uno de los dos | Lista las cuotas de un mes (calendario) |

Es obligatorio indicar **exactamente uno** de `debt_id` o `period`.

### Ejemplo por deuda

```json
{ "debt_id": 4 }
```

```json
{
  "mode": "debt",
  "debt_id": 4,
  "count": 1,
  "bills": [
    { "id": 10, "installment_number": 1, "amount": 100.00, "status": "pending", "due_date": "2026-11-30" }
  ]
}
```

### Ejemplo por mes

```json
{ "period": "2026-11" }
```

```json
{
  "mode": "month",
  "period": "2026-11",
  "count": 1,
  "bills": [{ "id": 11, "debt_id": 4, "amount": 50.00, "status": "paid" }]
}
```

## Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | Se envían ambos modos, ninguno, o `debt_id` ≤ 0 |
| `invalid_period` | `period` inválido |
| `auth_failed` | API key inválida o ausente |

## Notas

- Los nombres de archivo de factura y recibo no se devuelven.
- El esquema de `debts` / `debt_bills` es protegido: esta tool no lo modifica (`DATA-PROTECTED-SCHEMA`).
