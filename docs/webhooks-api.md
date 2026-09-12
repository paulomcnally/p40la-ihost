# Webhooks de Facturas por Servicio — Guía de Integración

> **Proyecto**: p40la-ihost (SONOFF iHost)  
> **Spec**: SPEC-069 — Webhooks por servicio para facturas (SPEC-071 — resiliencia al soft-delete)  
> **Versión**: 1.1 (2026-09-12)  
> **Estado**: implementado en rama `feature/SPEC-071`

---

## 1. Propósito

Este documento describe la **API de webhooks** que p40la-ihost expone para que un **proyecto externo automatizado** pueda empujar facturas hacia la app. El proyecto externo se conecta a los sitios de los servicios (Claro, DISNORTE, etc.), lee los datos de las facturas y los envía a este webhook para **crear, actualizar o marcar como pagadas** las facturas en p40la-ihost.

```
[Proyecto externo (automatización)]
   |  POST /webhooks/{uuid}
   |  Header: X-Webhook-Key: <api_key>
   |  Body: { year, month, amount, status, ... }
   v
[p40la-ihost]  →  SQLite (bills)
```

---

## 2. Conceptos clave

| Concepto | Descripción |
|----------|-------------|
| **`{uuid}`** | Identificador único por servicio. Cada servicio tiene un UUID (ej: `fd281ed1-e0b2-4c9c-8dab-d7080f28adf4`). Identifica **a qué servicio** pertenece la factura. |
| **`X-Webhook-Key`** | Header con la **api_key global** (la misma para todos los webhooks). Autentica que el emisor está autorizado. |
| **Feature toggle** | En Configuración → Webhooks hay un toggle maestro. Si está OFF, el endpoint responde `403` y no procesa nada. |

> **Regla**: el `uuid` identifica el recurso (servicio), la `api_key` autentica al emisor. Ambos son necesarios.

---

## 3. Endpoint

```
POST /webhooks/{uuid}
```

**Headers requeridos**:

| Header | Valor | Obligatorio |
|--------|-------|-------------|
| `X-Webhook-Key` | api_key global (hex de 64 chars) | ✅ |
| `Content-Type` | `application/json` | ✅ |

---

## 4. Schema del payload (contrato)

Los clientes deben adaptar su envío a este schema. **`year`, `month` y `amount` son obligatorios.** El resto es opcional.

```json
{
  "year": 2026,
  "month": 9,
  "amount": 1234.56,
  "invoice_number": "INV-2026-09",
  "status": "paid",
  "paid_at": "2026-09-10T18:30:00Z",
  "payment_reference": "TXN-12345",
  "drive_url": "https://drive.google.com/file/d/..."
}
```

### 4.1 Campos

| Campo | Tipo | Obligatorio | Descripción |
|-------|------|-------------|-------------|
| `year` | int | ✅ | Año de la factura (1900–2100) |
| `month` | int | ✅* | Mes de la factura (1–12). **Para servicios anuales se ignora** (la factura se identifica solo por año). |
| `amount` | number | ✅ | Monto de la factura (>= 0) |
| `invoice_number` | string | ❌ | Número/folio de la factura |
| `status` | string | ❌ | `"pending"` o `"paid"`. Si se omite en una **creación** → queda `pending`; en una **actualización** → no toca el estado actual. |
| `paid_at` | string | ❌ | Fecha/hora de pago. Formatos aceptados: `RFC3339` (`2026-09-10T18:30:00Z`), `YYYY-MM-DDTHH:MM:SS`, `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD`. Si se envía `status:"paid"` y no se incluye → usa la fecha/hora actual. |
| `payment_reference` | string | ❌ | Referencia interna del pago (ej: número de transacción) |
| `drive_url` | string | ❌ | Link de Google Drive del comprobante. Solo se aceptan URLs de `drive.google.com` / `docs.google.com`. |

> ⚠️ `paid_at` **no puede ser futura** (más de 24h desde ahora).

---

## 5. Comportamiento (Upsert)

El endpoint hace un **upsert** sobre el par `(service_id, year, month)`:

| Escenario | Comportamiento |
|-----------|----------------|
| **No existe** factura para `(servicio, año, mes)` | La **crea**. `status` default `pending`. |
| **Existe** y payload trae `status: "paid"` | La **marca como pagada**: actualiza `amount`, `invoice_number`, `drive_url`, guarda `paid_at` y `payment_reference`. |
| **Existe** y payload trae `status: "pending"` | Actualiza datos descriptivos y **revierte a pendiente** (limpia `paid_at` y `payment_reference`). |
| **Existe** y payload **omite** `status` | Actualiza solo `amount`, `invoice_number`, `drive_url`. **No toca el estado actual** (si estaba pagada, sigue pagada). |
| **Reenvío idéntico** | No duplica facturas (idempotente). |
| **Período con factura soft-deleted** | La factura fue borrada desde la UI (`DELETE /api/bills/{id}`), pero sigue ocupando la clave única. El webhook **la reactiva** y la actualiza con el payload, respondiendo `200` con `created: false` (SPEC-071). |

**Sobre los servicios anuales**: si el servicio tiene `frequency: "yearly"`, el campo `month` del payload **se ignora** y la factura se indexa como `month: 0`. Envialo igual por consistencia o pon `0`.

> **Soft-delete (SPEC-071)**: borrar una factura desde la UI no la elimina físicamente (setea `deleted_at`). Como la tabla `bills` tiene `UNIQUE(service_id, year, month)` a nivel de tabla, la fila borrada sigue bloqueando el período. Desde SPEC-071 el webhook detecta el conflicto de UNIQUE (código SQLite 2067), localiza la fila soft-deleted del mismo período, la reactiva y aplica el payload. Esto es **idempotente**: reenviar el período una segunda vez vuelve a actualizar la misma fila, sin duplicar.

---

## 6. Respuestas

### 200 OK — Factura creada o actualizada

```json
{
  "bill": {
    "id": 42,
    "service_id": 7,
    "year": 2026,
    "month": 9,
    "amount": 1234.56,
    "invoice_number": "INV-2026-09",
    "status": "paid",
    "paid_at": "2026-09-10T18:30:00Z",
    "payment_reference": "TXN-12345"
  },
  "created": true
}
```

- `created: true` → la factura se creó
- `created: false` → la factura existente se actualizó

### Códigos de error

| Código | Caso | Body |
|--------|------|------|
| `200` | Factura creada/actualizada | `{ bill, created }` |
| `400` | Body inválido (JSON malformado, validación fallida) | `{ "error": "invalid_body", "message": "..." }` |
| `401` | Header `X-Webhook-Key` faltante o incorrecto | `{ "error": "unauthorized", "message": "api_key de webhook inválida" }` |
| `403` | Feature de webhooks deshabilitada en Configuración | `{ "error": "webhook_disabled", "message": "..." }` |
| `404` | `uuid` no corresponde a ningún servicio activo | `{ "error": "not_found", "message": "..." }` |

---

## 7. Ejemplos

### 7.1 Crear una factura pendiente

```bash
curl -X POST http://ihost.local:8088/webhooks/fd281ed1-e0b2-4c9c-8dab-d7080f28adf4 \
  -H "X-Webhook-Key: 7381d6dc90a8a9ad421c2e751521a1553ebe10951c7dab491574376d653076bf" \
  -H "Content-Type: application/json" \
  -d '{"year":2026,"month":9,"amount":520,"invoice_number":"INV-2026-09"}'
```

### 7.2 Marcar una factura como pagada

```bash
curl -X POST http://ihost.local:8088/webhooks/fd281ed1-e0b2-4c9c-8dab-d7080f28adf4 \
  -H "X-Webhook-Key: 7381d6dc90a8a9ad421c2e751521a1553ebe10951c7dab491574376d653076bf" \
  -H "Content-Type: application/json" \
  -d '{
    "year": 2026,
    "month": 9,
    "amount": 540,
    "invoice_number": "INV-2026-09",
    "status": "paid",
    "paid_at": "2026-09-05",
    "payment_reference": "TXN-001"
  }'
```

### 7.3 Python (requests)

```python
import requests

WEBHOOK_URL = "http://ihost.local:8088/webhooks/fd281ed1-e0b2-4c9c-8dab-d7080f28adf4"
API_KEY = "7381d6dc90a8a9ad421c2e751521a1553ebe10951c7dab491574376d653076bf"

payload = {
    "year": 2026,
    "month": 9,
    "amount": 540,
    "invoice_number": "INV-2026-09",
    "status": "paid",
    "paid_at": "2026-09-05",
    "payment_reference": "TXN-001",
}

resp = requests.post(
    WEBHOOK_URL,
    json=payload,
    headers={"X-Webhook-Key": API_KEY},
)
if resp.status_code == 200:
    data = resp.json()
    print(f"created={data['created']} bill_id={data['bill']['id']}")
else:
    print(f"ERROR {resp.status_code}: {resp.json().get('message')}")
```

---

## 8. Guía de operación

### 8.1 Obtener la URL y la api_key (manual, una vez)

1. **Activar la feature**: Configuración → **Webhooks** → activar el toggle **"Habilitar webhooks"**. Con el toggle OFF el endpoint responde `403`.
2. **Copiar la api_key**: Configuración → Webhooks → botón copiar en el campo **"API key global"**.
3. **Obtener el UUID de cada servicio**: Servicios → menú de 3 puntos del servicio → **Webhook** → copiar la URL (contiene el `uuid`).

> La api_key es **global** (una sola para todos los webhooks). El `uuid` es **por servicio**.

### 8.2 Regeneración (rompe integraciones)

- **Regenerar `uuid`**: menú 3 puntos del servicio → Webhook → ícono de refrescar. La URL vieja deja de funcionar (`404`).
- **Regenerar api_key**: Configuración → Webhooks → "Regenerar API key". La clave vieja deja de funcionar (`401`).

Después de regenerar cualquiera de los dos, el proyecto externo debe actualizarlos.

---

## 9. Recomendaciones para el agente que implementa el cliente

1. **Configuración centralizada**: guardar `WEBHOOK_URL_BASE`, `API_KEY` y la lista de `{servicio → uuid}` en un archivo de config (env vars o JSON), no hardcodeados.
2. **Idempotencia**: al reenviar datos del mismo período, el webhook no duplica. Es seguro re-ejecutar.
3. **Manejo de errores**:
   - `401` → la api_key cambió: regenerar/actualizar credenciales y reintentar.
   - `403` → feature deshabilitada: avisar (requiere acción manual en Settings).
   - `404` → el `uuid` cambió: actualizar el mapeo servicio→uuid.
   - `400` → validar el payload (mes 1-12, monto >= 0, `status` solo `pending`/`paid`).
4. **Estados de pago**: si la automatización confirma pago, enviar `status:"paid"` con `paid_at` y `payment_reference`. Si solo quiere sincronizar datos sin tocar estado, **omitir** `status`.
5. **URL base**: en producción usar `http://ihost.local:8088`; en local `http://localhost:8088`.
6. **`drive_url`**: solo se aceptan enlaces de Google Drive (`https://drive.google.com/...` o `https://docs.google.com/...`).

---

## 10. Referencias de implementación (p40la-ihost)

| Archivo | Rol |
|---------|-----|
| `internal/api/webhook_handlers.go` | Handler `POST /webhooks/{uuid}` + gestión de api_key |
| `internal/api/middleware.go` | `WebhookAuthMiddleware` (valida `X-Webhook-Key`) |
| `internal/services/webhook.go` | `WebhookService`: upsert de facturas, api_key, UUID, recuperación de soft-deleted (SPEC-071) |
| `internal/services/system_settings.go` | Toggle `webhook_enabled` |
| `internal/storage/bill.go` | `FindByServicePeriod`, `FindByServicePeriodIncludingDeleted`, `Create`, `Pay`, `MarkPending`, `Reactivate`, `UpdateWebhookFields` |
| `migrations/0027_add_services_webhook_uuid.up.sql` | Columna `webhook_uuid` en `services` |
| `frontend/src/components/WebhookModal.tsx` | Modal webhook por servicio |
| `frontend/src/pages/SettingsPage.tsx` | Sección Webhooks (toggle + api_key) |

---

*Documento generado para integración externa. Cualquier cambio al contrato debe reflejarse en la spec SPEC-069.*