---
title: "Tools de pensión"
sidebar_position: 6
description: "Referencia de pension_month_summary y pension_catalogs_list"
---

# Tools de pensión

Las dos tools cubren el módulo de **Pensión Alimenticia**: registros mensuales de manutención, pagos de salario, cierre del mes y los catálogos que alimentan la generación automática. Son de **solo lectura**, reutilizan endpoints HTTP existentes y no acceden a SQLite.

No dependen de la compuerta de escritura: funcionan igual en stdio y en HTTP, sin `P40LA_MCP_ALLOW_WRITES` ni `P40LA_MCP_REMOTE_WRITES`.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `pension_month_summary` | Lectura | 3 | Registros y pagos de salario de un mes, con totales por moneda y estado de cierre |
| `pension_catalogs_list` | Lectura | 5 | Hijos, categorías, salarios, configuraciones de generación y monedas |

## `pension_month_summary`

Resume un mes de pensión: trae los registros de manutención, los pagos de salario y el estado de cierre, y calcula conteos por estado y **totales agrupados por moneda**. No modifica datos.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `period` | string `YYYY-MM` | No | Período a consultar; default el mes actual en `P40LA_TIMEZONE` |
| `child_id` | integer | No | Filtra **solo los registros de manutención** por hijo; debe ser mayor que `0` |

### Ejemplo

```json
{
  "period": "2026-11",
  "child_id": 1
}
```

### Salida

```json
{
  "year": 2026,
  "month": 11,
  "period": "2026-11",
  "closed": false,
  "records_count": 2,
  "paid_count": 1,
  "pending_count": 1,
  "rejected_count": 0,
  "record_totals": [
    {
      "currency": "NIO",
      "amount": 150.00,
      "count": 2
    }
  ],
  "salary_count": 1,
  "received_count": 1,
  "salary_totals": [
    {
      "currency": "NIO",
      "amount": 900.00,
      "count": 1
    }
  ],
  "records": [
    {
      "id": 1,
      "child_id": 1,
      "child_name": "Hijo Demo",
      "pension_category_id": 1,
      "category_name": "Colegio",
      "year": 2026,
      "month": 11,
      "amount": 100.00,
      "currency": "NIO",
      "status": "paid",
      "paid_at": "2026-11-05T10:00:00Z",
      "payment_method": "transfer"
    },
    {
      "id": 2,
      "child_id": 1,
      "child_name": "Hijo Demo",
      "pension_category_id": 1,
      "category_name": "Colegio",
      "year": 2026,
      "month": 11,
      "amount": 50.00,
      "currency": "NIO",
      "status": "pending"
    }
  ],
  "salary_payments": [
    {
      "id": 9,
      "salary_id": 1,
      "employer": "Proveedor Demo",
      "year": 2026,
      "month": 11,
      "amount": 900.00,
      "currency": "NIO",
      "status": "received",
      "received_amount": 900.00,
      "received_at": "2026-11-15T09:30:00Z"
    }
  ]
}
```

### Reglas

- `records_count` y `salary_count` son la cantidad de filas devueltas; `paid_count`, `pending_count` y `rejected_count` cuentan solo esos tres estados de registro, y `received_count` cuenta pagos de salario con `status: "received"`.
- `record_totals` y `salary_totals` agrupan por `currency` y suman `amount` con dos decimales. Nunca se suman monedas distintas.
- `closed_at` solo aparece cuando `closed` es `true`; el valor es una fecha RFC 3339. Un mes abierto devuelve `closed: false` sin `closed_at`.
- `child_id` filtra únicamente los registros de manutención: la API de pagos de salario no acepta ese filtro, así que `salary_payments` sigue cubriendo todo el mes.
- Los arrays vacíos se devuelven como `[]`, nunca `null`.
- `status` de un registro puede ser `paid`, `pending` o `rejected`; un valor distinto se cuenta en `records_count` pero no en los conteos por estado.
- La tool hace siempre **3 llamadas**: registros, pagos de salario y estado de cierre. Si cualquiera falla, responde error y no devuelve un resumen parcial.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_period` | `period` no tiene formato `YYYY-MM`, el año está fuera de `1900..2100` o el mes fuera de `1..12` |
| `invalid_request` | `child_id` es `0` o negativo |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## `pension_catalogs_list`

Devuelve los catálogos necesarios para operar pensión: hijos, categorías de pensión, salarios, configuraciones de generación automática y monedas. No recibe parámetros.

### Entrada

La tool no recibe campos: se llama con un objeto vacío.

### Ejemplo

```json
{}
```

### Salida

```json
{
  "children": [
    {
      "id": 1,
      "first_name": "Hijo",
      "last_name": "Demo",
      "birth_date": "2015-06-01"
    }
  ],
  "pension_categories": [
    {
      "id": 1,
      "name": "Colegio",
      "share_percent": 50,
      "currency": "NIO",
      "auto_generate": true
    }
  ],
  "salaries": [
    {
      "id": 1,
      "employer": "Proveedor Demo",
      "amount": 900.00,
      "currency_id": 1,
      "payment_day": 15,
      "active": true
    }
  ],
  "configs": [
    {
      "id": 1,
      "child_id": 1,
      "child_name": "Hijo Demo",
      "pension_category_id": 1,
      "category_name": "Colegio",
      "amount": 100.00,
      "currency": "NIO",
      "is_active": true,
      "auto_generate": true
    }
  ],
  "currencies": [
    {
      "id": 1,
      "code": "NIO",
      "name": "Córdoba",
      "symbol": "C$"
    }
  ]
}
```

### Reglas

- Cada catálogo es una lista independiente; si uno viene vacío se devuelve `[]`.
- La tool hace **5 llamadas** (una por catálogo). Si una falla, responde error y no devuelve un resultado parcial.
- Los campos son una **allowlist**: se omiten notas, comprobantes, referencias de pago y marcas de tiempo que la API sí devuelve.
- `share_percent` es el porcentaje de participación de la categoría; `auto_generate` indica si la configuración genera registros automáticamente.
- `currency_id` de un salario se resuelve con el catálogo `currencies` de la misma respuesta.

### Errores

| Código | Cuándo |
|---|---|
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta del servidor no interpretable |

## Privacidad y datos ocultos

`pension_month_summary` y `pension_catalogs_list` proyectan una allowlist de campos (`SEC-NO-PII`). No exponen:

- `notes`, `evidence_notes` ni `payment_reference` de los registros.
- `proof_file_name`: el nombre del archivo de comprobante.
- `exchange_rate`, `original_amount` y `original_currency` de registros en otra moneda.
- Notas y marcas de tiempo internas de hijos, categorías, salarios y configuraciones.

Los hijos devuelven `first_name`, `last_name` y `birth_date`: son datos personales. Usalos solo para desambiguar y no los reproduzcas en respuestas extensas, logs ni capturas.

## Flujo recomendado para un agente

1. Llamar `pension_catalogs_list` para obtener los IDs de hijo, categoría y moneda.
2. Llamar `pension_month_summary` con el `period` que el usuario quiera revisar, y `child_id` si corresponde a un solo hijo.
3. Reportar **totales por moneda** (nunca una suma entre monedas) y los conteos por estado.
4. Si `closed` es `true`, aclarar que el mes está cerrado y que cualquier corrección se hace desde la UI.
