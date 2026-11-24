---
title: "Tools de mantenimiento del hogar"
sidebar_position: 10
description: "Referencia de home_maintenance_list"
---

# Tools de mantenimiento del hogar

La tool consulta las **reparaciones y mantenimientos** registrados en una vivienda (`home.maintenance`), con sus líneas de mano de obra e insumos y los totales por moneda. Es de **solo lectura**: no registra mantenimientos, no agrega líneas y no depende de la compuerta de escritura.

## Resumen

| Tool | Tipo | Llamadas HTTP | Qué hace |
|---|---|---|---|
| `home_maintenance_list` | Lectura | 1 | Histórico de reparaciones de una vivienda con totales por moneda |

## `home_maintenance_list`

Lista los mantenimientos de una vivienda, con filtros opcionales de año y tipo, y devuelve cada registro con su desglose de mano de obra e insumos. Solo lectura.

### Entrada

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `home_id` | integer | Sí | ID de la vivienda; debe ser mayor que `0` |
| `year` | integer | No | Filtra por año del mantenimiento; rango `1900..9999` |
| `kind` | string | No | `corrective` (correctivo) o `preventive` (preventivo) |

Los filtros se combinan con AND. Sin `year` ni `kind` se devuelve el histórico completo de la vivienda.

### Ejemplo

```json
{
  "home_id": 3,
  "year": 2026,
  "kind": "preventive"
}
```

### Salida

```json
{
  "home_id": 3,
  "count": 1,
  "maintenances": [
    {
      "id": 1,
      "home_id": 3,
      "performed_at": "2026-03-10",
      "kind": "preventive",
      "area": "plumbing",
      "institution_id": 7,
      "institution_name": "Taller Demo",
      "description": "Revisión demo de fontanería",
      "totals_by_currency": [
        {
          "currency_id": 1,
          "currency_code": "NIO",
          "labor_cost": 500.00,
          "supplies_cost": 1200.00,
          "total": 1700.00
        }
      ],
      "items": [
        {
          "id": 101,
          "maintenance_id": 1,
          "item_type": "labor",
          "description": "Mano de obra demo",
          "quantity": 1,
          "unit": "unit",
          "amount": 500.00,
          "currency_id": 1,
          "currency_code": "NIO",
          "institution_id": 7,
          "institution_name": "Taller Demo"
        },
        {
          "id": 102,
          "maintenance_id": 1,
          "item_type": "supplies",
          "description": "Insumos demo",
          "quantity": 2,
          "unit": "unit",
          "amount": 600.00,
          "currency_id": 1,
          "currency_code": "NIO",
          "institution_id": 7,
          "institution_name": "Taller Demo"
        }
      ],
      "created_at": "2026-03-10T14:00:00Z",
      "updated_at": "2026-03-10T14:00:00Z"
    }
  ]
}
```

### Reglas

- `kind` es `corrective` o `preventive`; cualquier otro valor devuelve `invalid_request` **sin** llamar a la API.
- `year` fuera de `1900..9999` devuelve `invalid_request` sin llamar a la API.
- `area` describe el sector de la vivienda: `plumbing`, `electrical`, `roof`, `paint`, `carpentry`, `appliances`, `garden`, `cleaning` u `other`.
- `items[].item_type` es `labor` (mano de obra) o `supplies` (insumos). `quantity` y `unit` acompañan a los insumos (`unit`, `m`, `m2`, `kg`, `lb`, `l`, `gal`, `sack`, `box`).
- `totals_by_currency` es por mantenimiento y suma `labor_cost` + `supplies_cost` en `total`, separado por moneda. Nunca se suman monedas distintas.
- `count` es la cantidad de mantenimientos devueltos; no se informa un total general del histórico en la respuesta de esta tool.
- Los mantenimientos se devuelven ordenados por `performed_at` descendente y, en empate, por `id` descendente.
- Una vivienda inexistente devuelve `home_not_found`.

### Nota de implementación

La API `GET /api/homes/{id}/maintenances` responde `{"items": [...], "totals_by_currency": [...]}`. La tool decodifica la respuesta como una lista, así que **contra la API actual la llamada devuelve `upstream_error`** hasta que se alinee el contrato (leer `items` o cambiar el endpoint). El esquema de cada mantenimiento descrito arriba es el real; solo cambia el sobre de la respuesta.

### Errores

| Código | Cuándo |
|---|---|
| `invalid_request` | `home_id` es `0` o negativo, `year` fuera de `1900..9999`, o `kind` distinto de `corrective`/`preventive` |
| `home_not_found` | La vivienda no existe |
| `auth_failed` | API key inválida, revocada o rotada |
| `upstream_error` | Error de red, timeout o respuesta con un sobre distinto al esperado |

## Privacidad y datos ocultos

La tool devuelve el registro tal como lo entrega la API; no aplica allowlist. Los campos con datos financieros (`institution_name`, importes) y las descripciones libres deben reportarse solo en lo necesario: no los incluyas en logs ni en capturas.

## Flujo recomendado para un agente

1. Pedir el `home_id` (el mismo de `/homes/{id}` en la UI).
2. Llamar `home_maintenance_list`, opcionalmente con `year` y `kind` para acotar.
3. Reportar la cantidad de mantenimientos y los **totales por moneda** de mano de obra e insumos.
4. Si el usuario quiere registrar un mantenimiento, aclarar que esta tool no escribe: la carga se hace desde la UI.
