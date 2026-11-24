---
title: "Tools de alquileres"
sidebar_position: 7
description: "Referencia de rental_leases_list, rental_lease_get, rental_home_status y rental_portfolio_summary"
---

# Tools de alquileres

Las cuatro tools cubren el módulo inmobiliario (`realestate.rental` y `realestate.portfolio`): contratos de alquiler, estado de una vivienda y resumen mensual del portafolio. Son de **solo lectura**: no crean contratos, no generan cargos, no emiten facturas y no dependen de la compuerta de escritura.

`home_id` es el mismo ID de la vivienda que aparece en la UI (`/homes/7`) y `lease_id` el del contrato (`/rental/leases/5`).

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `rental_leases_list` | Lectura | 1 | Lista contratos con filtros de estado y vivienda, con saldos derivados |
| `rental_lease_get` | Lectura | 2 | Detalle de un contrato y su balance (cargado, pagado, pendiente) |
| `rental_home_status` | Lectura | 1 | Estado de una vivienda: publicación, contrato, apartado, cargos y recordatorios |
| `rental_portfolio_summary` | Lectura | 1 | Ingresos, gastos y pendientes del mes por vivienda y totales |

## `rental_leases_list`

Lista los contratos de alquiler no eliminados, ordenados por fecha de inicio descendente. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `status` | string | No | `draft`, `active` o `terminated`; se envía tal cual a la API |
| `home_id` | integer | No | Filtra por vivienda; debe ser mayor que `0` |

Los filtros se combinan con AND. Sin filtros se devuelven todos los contratos.

### Ejemplo

```json
{
  "status": "active",
  "home_id": 3
}
```

### Salida

```json
{
  "count": 1,
  "leases": [
    {
      "id": 5,
      "home_id": 3,
      "home_name": "Hogar Demo",
      "home_address": "Calle Demo 123",
      "status": "active",
      "start_date": "2026-01-01",
      "end_date": "2026-12-31",
      "term_months": 12,
      "rent_amount": 800.00,
      "maintenance_amount": 50.00,
      "currency": "NIO",
      "payment_day": 5,
      "deposit_amount": 800.00,
      "auto_renew": false,
      "notice_days": 30,
      "commission_percent": 10,
      "commission_type": "percent",
      "commission_amount": 80.00,
      "commission_base": "rent",
      "agency_name": "Agencia Demo",
      "agent_name": "Agente Demo",
      "expiring": false,
      "charged_total": 1600.00,
      "paid_total": 800.00,
      "balance": 800.00
    }
  ]
}
```

### Reglas

- `balance` es derivado: `charged_total - paid_total`, redondeado a dos decimales.
- `expiring` es `true` cuando el contrato está `active` y vence dentro de los próximos **60 días**.
- `parties` **no** aparece en el listado (solo en el detalle). `charge_count` e `invoice_count` tampoco se informan en esta tool.
- `status` no se valida en la tool: un valor fuera de la lista devuelve `count: 0`, no un error.
- `home_name`, `home_address`, `agency_name`, `agent_name` y los campos de comisión aparecen solo cuando tienen valor.
- `count` es la cantidad de contratos devueltos, no el total del sistema.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` es `0` o negativo |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## `rental_lease_get`

Devuelve el detalle de un contrato con sus partes y su balance de cobranza. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `lease_id` | integer | Sí | ID del contrato; debe ser mayor que `0` |

### Ejemplo

```json
{
  "lease_id": 5
}
```

### Salida

```json
{
  "lease": {
    "id": 5,
    "home_id": 3,
    "home_name": "Hogar Demo",
    "status": "active",
    "start_date": "2026-01-01",
    "end_date": "2026-12-31",
    "rent_amount": 800.00,
    "currency": "NIO",
    "payment_day": 5,
    "expiring": false,
    "charged_total": 1600.00,
    "paid_total": 800.00,
    "balance": 800.00,
    "parties": [
      {
        "id": 11,
        "lease_id": 5,
        "person_id": 4,
        "role": "tenant",
        "person": {
          "id": 4,
          "full_name": "Inquilino Demo",
          "tax_id": "DEMO-TAX-001"
        }
      }
    ]
  },
  "balance": {
    "charged": 1600.00,
    "paid": 800.00,
    "balance": 800.00,
    "currency": "NIO"
  }
}
```

### Reglas

- Hace **2 llamadas**: el contrato y su balance. Si el contrato existe pero el balance falla, la tool responde error y no devuelve un objeto parcial.
- `lease.parties` es la lista de partes con su rol (`tenant`, `guarantor`, `owner`, `agent`, entre otros) y la persona asociada.
- `balance.charged` es el total cargado del contrato, `balance.paid` lo cobrado y `balance.balance` la diferencia; `currency` repite la moneda del contrato.
- Un `lease_id` inexistente devuelve `lease_not_found` (la API responde `404`).

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `lease_id` es `0` o negativo |
| `lease_not_found` | El contrato no existe o fue eliminado |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## `rental_home_status`

Devuelve el estado de una vivienda: uso, estado derivado de publicación, contrato activo, apartado activo, próximos cargos pendientes y recordatorios. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |

### Ejemplo

```json
{
  "home_id": 3
}
```

### Salida

```json
{
  "home_id": 3,
  "home_name": "Hogar Demo",
  "home_address": "Calle Demo 123",
  "usage": "rental",
  "listing_status": "rented",
  "active_lease": {
    "id": 5,
    "status": "active",
    "start_date": "2026-01-01",
    "end_date": "2026-12-31",
    "rent_amount": 800.00,
    "currency": "NIO",
    "payment_day": 5,
    "expiring": false,
    "charged_total": 1600.00,
    "paid_total": 800.00,
    "balance": 800.00
  },
  "next_due_date": "2026-12-05",
  "upcoming_charges": [
    {
      "id": 31,
      "lease_id": 5,
      "kind": "rent",
      "period": "2026-12",
      "description": "Alquiler diciembre",
      "amount": 800.00,
      "currency": "NIO",
      "due_date": "2026-12-05",
      "status": "pending"
    }
  ],
  "reminders": [
    {
      "id": 7,
      "kind": "payment",
      "title": "Recordatorio demo",
      "schedule_type": "monthly",
      "next_due_at": "2026-12-01",
      "enabled": true
    }
  ]
}
```

### Reglas

- `usage` es el uso manual de la vivienda: `rental`, `inhabited` o `unused`.
- `listing_status` es el estado **derivado**: `rented` (contrato activo), `reserved` (apartado activo), `listed` (sin contrato ni apartado), `inhabited` o `unused` (según `usage`).
- `active_lease` y `active_reservation` se omiten cuando la vivienda no tiene contrato o apartado activo.
- `upcoming_charges` trae como máximo los **5 cargos pendientes** más próximos; `next_due_date` es el vencimiento del primero.
- `reminders` incluye los recordatorios de la vivienda, tengan o no fecha próxima.
- Un `home_id` inexistente devuelve `home_not_found` (la API responde `404`).

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` es `0` o negativo |
| `home_not_found` | La vivienda no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## `rental_portfolio_summary`

Resume el portafolio de alquileres de un mes: ingresos, gastos, neto y pendientes por vivienda, más los totales y el conteo de viviendas por estado. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `period` | string `YYYY-MM` | No | Período a resumir; default el mes actual en `P40LA_TIMEZONE` |

### Ejemplo

```json
{
  "period": "2026-11"
}
```

### Salida

```json
{
  "year": 2026,
  "month": 11,
  "homes": [
    {
      "home_id": 3,
      "name": "Hogar Demo",
      "address": "Calle Demo 123",
      "usage": "rental",
      "state": "rented",
      "income": [
        {
          "currency": "NIO",
          "amount": 800.00
        }
      ],
      "expenses": [
        {
          "currency": "NIO",
          "amount": 150.00
        }
      ],
      "net": [
        {
          "currency": "NIO",
          "amount": 650.00
        }
      ],
      "pending_income": [],
      "pending_expenses": []
    }
  ],
  "totals": {
    "income": [
      {
        "currency": "NIO",
        "amount": 800.00
      }
    ],
    "expenses": [
      {
        "currency": "NIO",
        "amount": 150.00
      }
    ],
    "net": [
      {
        "currency": "NIO",
        "amount": 650.00
      }
    ],
    "pending_income": [],
    "pending_expenses": []
  },
  "state_counts": {
    "listed": 0,
    "reserved": 0,
    "rented": 1,
    "inhabited": 0,
    "unused": 0
  }
}
```

### Reglas

- Cada importe es un par `currency` + `amount`: **nunca** se suman monedas distintas.
- Los importes en cero se omiten, por lo que `income`, `expenses`, `net`, `pending_income` y `pending_expenses` pueden venir como `[]`.
- `net` es ingresos menos gastos por moneda, calculado por la API.
- `state_counts` cuenta las viviendas por estado derivado (`listed`, `reserved`, `rented`, `inhabited`, `unused`) e incluye en cero los estados sin viviendas.
- El período debe estar entre 2000 y 2100; fuera de ese rango la API responde `invalid_period`.
- No genera cargos ni movimientos: solo agrega datos existentes.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_period` | `period` no tiene formato `YYYY-MM`, o la API rechaza el año o el mes |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## Privacidad y datos ocultos

Las tools de alquileres eliminan estas claves de **primer nivel** en cada objeto que devuelven (`SEC-NO-PII`):

`email`, `notify_email`, `phone`, `person_email`, `tenant_email`, `guarantor_email`, `applicant_email`, `agent_email`, `agency_email`.

Tené en cuenta que:

- La eliminación **no recorre objetos anidados**. `parties[].person.email`, `parties[].person.phone`, `active_reservation.applicant.email` o `reminders[].extra_emails` pueden aparecer si la API los devuelve: tratalos como datos personales y no los reproduzcas.
- `owner_bank_account` del contrato es un número de cuenta del propietario: no lo muestres en respuestas, logs ni capturas.
- `tax_id` de una persona es un documento de identidad: usalo solo para desambiguar.

## Flujo recomendado para un agente

1. Empezar por `rental_portfolio_summary` para el panorama del mes, o por `rental_leases_list` si buscás un contrato concreto.
2. Con el `lease_id`, llamar `rental_lease_get` para el detalle y el balance.
3. Con el `home_id`, llamar `rental_home_status` para ver contrato, apartado y próximos cargos.
4. Reportar montos **por moneda** y aclarar que estas tools no crean cargos, facturas ni contratos: cualquier alta se hace desde la UI.
