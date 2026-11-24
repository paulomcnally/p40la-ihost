---
title: "Tools de reparaciones del hogar"
sidebar_position: 10
description: "Referencia de home_maintenance_list"
---

# Tools de reparaciones del hogar

Tool de **solo lectura** sobre el módulo `home.maintenance` (SPEC-154). No registra, edita ni borra reparaciones.

## `home_maintenance_list`

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda |
| `year` | integer | No | Filtra por año (1900–9999) |
| `kind` | string | No | `corrective` o `preventive` |

```json
{ "home_id": 3, "year": 2026, "kind": "preventive" }
```

```json
{
  "home_id": 3,
  "count": 1,
  "maintenances": [
    {
      "id": 1,
      "performed_at": "2026-03-10",
      "kind": "preventive",
      "area": "plumbing",
      "institution_name": "Proveedor Demo",
      "totals_by_currency": [
        { "currency_code": "NIO", "labor_cost": 500.00, "supplies_cost": 200.00, "total": 700.00 }
      ]
    }
  ]
}
```

## Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` ≤ 0, `kind` inválido o `year` fuera de rango |
| `not_found` | La vivienda no existe o está eliminada |
| `auth_failed` | API key inválida o ausente |

## Notas

- Los totales se agrupan por moneda e incluyen mano de obra e insumos.
- El listado no incluye el detalle de líneas; para eso se requiere el detalle por ID (no expuesto en esta ola).
