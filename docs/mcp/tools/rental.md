---
title: "Tools de alquileres"
sidebar_position: 7
description: "Referencia de rental_leases_list, rental_lease_get, rental_home_status y rental_portfolio_summary"
---

# Tools de alquileres

Tools de **solo lectura** sobre los módulos `realestate.rental` y `realestate.portfolio` (SPEC-154). No activan contratos, no emiten facturas y no registran pagos.

> Todas las salidas eliminan emails y teléfonos de las partes, en cualquier nivel de anidamiento.

## `rental_leases_list`

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `status` | string | No | `draft`, `active` o `terminated` |
| `home_id` | integer | No | Filtra por vivienda |

```json
{ "status": "active", "home_id": 3 }
```

```json
{ "count": 1, "leases": [{ "id": 5, "home_id": 3, "status": "active", "rent_amount": 500.00 }] }
```

## `rental_lease_get`

Detalle de un contrato más su balance.

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `lease_id` | integer | Sí | ID del contrato |

```json
{ "lease_id": 5 }
```

```json
{
  "lease": { "id": 5, "status": "active", "term_months": 12, "rent_amount": 500.00 },
  "balance": { "charged": 1000.00, "paid": 500.00 }
}
```

## `rental_home_status`

Estado de una vivienda: publicación, contrato activo, apartado activo, cargos pendientes y recordatorios.

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda |

```json
{ "home_id": 3 }
```

## `rental_portfolio_summary`

Resumen mensual del portafolio.

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `period` | string `YYYY-MM` | No | Default: mes actual |

```json
{ "period": "2026-11" }
```

Devuelve ingresos, pendientes, gastos y estado por vivienda tal como los calcula `portfolio`.

## Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `lease_id` o `home_id` ≤ 0 |
| `invalid_period` | `period` inválido |
| `not_found` | El contrato o la vivienda no existen |
| `auth_failed` | API key inválida o ausente |

## Privacidad

Se omiten `email`, `notify_email`, `phone` y variantes (`tenant_email`, `guarantor_email`, `agent_email`, `agency_email`, `applicant_email`) en todo el árbol.
