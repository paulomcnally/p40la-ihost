---
title: "Webhooks por servicio para facturas"
id: "SPEC-069"
status: "released"
author: "paulomcnally"
created: "2026-09-11"
updated: "2026-09-11"
github_issue: 72
---

# Webhooks por servicio para facturas

**ID**: SPEC-069  
**Estado**: released  
**Autor**: paulomcnally  
**Creado**: 2026-09-11  
**Actualizado**: 2026-09-11

---

## 1. Resumen Ejecutivo

El usuario quiere automatizar otro proyecto que consulta las facturas de algunos servicios y empujar esa información hacia p40la-ihost. Actualmente las facturas (tabla `bills`) solo se cargan manualmente (formulario) o mediante el analizador de documentos (subida de PDFs). No existe ningún mecanismo programático para que un sistema externo cree o actualice facturas, por lo que la automatización es imposible sin tocar la UI.

Esta spec agrega un **webhook por servicio**: cada servicio recibe una URL única del estilo `POST /webhooks/{uuid}` donde el `uuid` identifica al servicio. El cliente externo envía el mes y año (junto con los datos de la factura) y el backend hace un **upsert**: si la factura para ese servicio/período no existe la crea, y si existe la actualiza. Las facturas pueden llegar con estado `paid` o `pending`, y el webhook debe reflejar ese estado (incluyendo `paid_at`, `payment_reference` y `drive_url` cuando aplique).

Para validar que solo clientes autorizados puedan escribir, se define una **api_key interna** de uso global que viaja en los headers de la solicitud (ej: `X-Webhook-Key`). Sin la api_key correcta, el webhook responde `401 Unauthorized`. El `uuid` en la URL identifica el servicio destino; la api_key autentica al emisor. Se documenta además un **schema JSON de entrada** al que deben adaptarse los clientes, de forma que cualquier sistema externo pueda integrarse de forma determinística.

**Consideraciones iHost**: solución backend-only con SQLite; sin dependencias externas nuevas; la api_key se guarda en `system_settings` (texto plano, suficiente para el alcance interno de este proyecto). El consumo de memoria/CPU es despreciable (una ruta más + una columna nueva en `services`).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Cada servicio tiene un `webhook_uuid` único (generado automáticamente). El endpoint de webhook es `POST /webhooks/{uuid}` y permite crear o actualizar facturas del servicio identificado por ese `uuid`.
2. **REQ-002**: El payload del webhook usa un schema JSON documentado. Campos mínimos: `year`, `month`, `amount`. Opcionales: `invoice_number`, `status` (`pending`|`paid`), `paid_at`, `payment_reference`, `drive_url`.
3. **REQ-003**: **Upsert**: si ya existe una factura para `(service_id, year, month)` se actualiza (amount, invoice_number, drive_url, estado); si no existe, se crea con esos datos. La unicidad ya está garantizada por `UNIQUE(service_id, year, month)` en `bills`.
4. **REQ-004**: **Estado**: si el payload trae `status: "paid"`, la factura queda pagada (`status='paid'`, `paid_at` = valor enviado o ahora, `payment_reference` y `drive_url` si se envían). Si trae `status: "pending"` (u omitido), la factura queda pendiente (sin `paid_at`).
5. **REQ-005**: **Validación de api_key**: existe una api_key global (interna) almacenada en `system_settings` (`webhook_api_key`). El webhook exige el header `X-Webhook-Key` y solo procesa la solicitud si coincide (comparación constante en tiempo). Si falta o es incorrecta → `401`.
6. **REQ-006**: **Gestión de la api_key**: la api_key global se gestiona desde **Configuración (Settings)**. Endpoints autenticados por sesión para obtener (`GET /api/webhook/key`) y regenerar (`POST /api/webhook/key/regenerate`) la api_key global.
7. **REQ-007**: **Toggle maestro en Settings**: en Configuración hay un toggle que habilita/deshabilita la feature de webhooks (clave `webhook_enabled`). Si está deshabilitado, el endpoint `POST /webhooks/{uuid}` responde `403` y el modal de webhook no muestra la URL. Con el toggle encendido, se muestra la api_key global con opción de regenerar y copiar.
8. **REQ-008**: **UI del webhook por servicio (modal)**: desde el menú de 3 puntos (`CardMenu`) de cada servicio en `ServicesPage` y desde el menú del header de `BillsPage`, una opción **"Webhooks"** abre un modal (`WebhookModal`) que muestra la URL del webhook (`{base_url}/webhooks/{uuid}`) con botón de copiar y opción de regenerar el `uuid`. No se agrega nada al formulario de edición de servicio.
9. **REQ-008a**: **Base URL configurable**: la URL del webhook apunta por defecto a `http://ihost.local:8088` (clave `webhook_base_url` en `system_settings`), pero es **editable en Settings** (sección Webhooks). El modal y el handler de regeneración usan esta base URL configurada para construir la URL completa.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Validación de input en el webhook: `month` entre 1 y 12, `year` razonable (1900-2100), `amount >= 0`, `status` solo `pending`/`paid`. Errores de validación → `400` con mensaje en JSON.
2. **REQ-010**: Respuestas del webhook en JSON: `200` con la factura creada/actualizada (o `{ "bill_id": ..., "created": true|false }`), `401` si api_key inválida, `403` si `webhook_enabled` está apagado, `404` si el `uuid` no corresponde a ningún servicio activo, `400` si el body es inválido.
3. **REQ-011**: El `webhook_uuid` se genera automáticamente al **crear** un servicio (si no se provee) y se expone en el GET de servicio. La migración debe poblar los servicios existentes con un `uuid` (regeneración backfill).
4. **REQ-012**: La api_key global se genera automáticamente al primer arranque (o primera consulta) si no existe en `system_settings`.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-013**: Opción de "desactivar" el webhook de un servicio (toggles en el servicio) para que un `uuid` válido pero desactivado responda `403`. Inicialmente se puede omitir en el MVP y quedar documentado como futura.
2. **REQ-014**: Endpoint `GET /api/webhooks` (autenticado) que liste la URL de webhook de cada servicio, para facilitar la configuración del proyecto externo.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El webhook hace 1-2 queries (SELECT por uuid → SELECT/INSERT/UPDATE por período). Sin impacto medible en el iHost.
- **Seguridad**: La api_key viaja por header (`X-Webhook-Key`), nunca en la URL. Comparación de strings en tiempo constante (`crypto/subtle`). El `uuid` del servicio es un secreto per-service (tipo API token), no un ID público.
- **Almacenamiento**: Una columna nueva `webhook_uuid` en `services` (migración `0027`) + una clave en `system_settings`. Sin tablas nuevas.
- **Disponibilidad**: El endpoint se registra fuera de `AuthMiddleware` (sesión), protegido por la api_key. El resto de rutas no cambia.
- **iHost**: Sin dependencias nuevas (Go stdlib + `crypto/rand`/`crypto/subtle`). Solo backend y una pequeña adición de UI.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Modelo actual**: `bills` (service_id, year, month, amount, invoice_number, status, drive_url, paid_at, payment_reference, file_hash, deleted_at) con `UNIQUE(service_id, year, month)` (migración `0005`). `BillStorage` ya tiene `FindByServicePeriod`, `Create`, `Update` y `Pay` (SPEC-043). El flujo de pago manual (SPEC-043) y el upsert del analizador (SPEC-021) son precedentes de upsert.
- **Rutas**: `internal/api/routes.go` usa `http.ServeMux` con patterns Go 1.22 (`POST /api/bills/{id}/pay`). Hay rutas públicas (`/health`, `/api/login`) y el resto detrás de `AuthMiddleware` (cookie de sesión). El webhook debe registrarse como ruta pública con su propia auth.
- **Settings**: `system_settings` (key/value) con `SystemSettingsStorage.Get/Set` — lugar natural para `webhook_api_key`. Precedentes: SMTP/VoiceMonkey configs.
- **UI**: `ServiceFormPage.tsx` (formulario de crear/editar servicio) es el lugar natural para mostrar/regenerar el webhook en modo edición. No existe página de detalle de servicio (los servicios navegan a `/services/bills/{id}`).
- **Formato de URL pedido por el usuario**: `localhost:8088/webhooks/uuid` → ruta raíz `/webhooks/{uuid}`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Webhook con `uuid` en la URL + api_key global en header | URL única por servicio (fácil de configurar en el proyecto externo), auth simple, `uuid` es secreto adicional | Dos secretos que administrar | ✅ Seleccionada |
| Un solo webhook con `service_id` en el body | Menos rutas | El cliente debe conocer el id interno; menos semántico | ❌ Rechazada |
| HMAC/firma por servicio | Muy seguro | Complejidad innecesaria para uso interno (un solo usuario/proyecto) | ❌ Rechazada |
| Basic auth / sesión en el webhook | Reutiliza la auth existente | Las sesiones expiran y no son aptas para integraciones server-to-server | ❌ Rechazada |
| Guardar la api_key en `services` (per-service) | Aislamiento por servicio | Más columnas y gestión; el usuario pidió "una api_key de uso interno" global | ❌ Rechazada |
| Guardar la api_key en `system_settings` global | Una sola clave, simple, centralizada | Compartida entre todos los webhooks | ✅ Seleccionada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: `uuid` del servicio en la URL + api_key global en header.
- **Contexto**: El usuario quiere una URL tipo `/webhooks/{uuid}` y una api_key interna que viaje en los headers. El `uuid` identifica el recurso (servicio), la api_key autentica al emisor.
- **Decisión**: Ruta pública `POST /webhooks/{uuid}`. Middleware `WebhookAuth` valida `X-Webhook-Key` contra `system_settings.webhook_api_key` con `subtle.ConstantTimeCompare`. Luego se resuelve el servicio por `webhook_uuid`.
- **Consecuencias**: Los clientes externos necesitan conocer la URL completa y la api_key. Positivo: no dependen de sesiones; fácil de automatizar.

**ADR-002**: Upsert sobre `(service_id, year, month)`.
- **Contexto**: La unicidad ya está en el esquema (`UNIQUE(service_id, year, month)`). El analizador (SPEC-021) y el pago (SPEC-043) ya escriben facturas.
- **Decisión**: En el webhook, buscar por `FindByServicePeriod`. Si no existe → `Create`. Si existe → actualizar con los campos provistos; si `status=paid`, usar la lógica de `Pay` (setear `paid_at`, `payment_reference`, `drive_url`).
- **Consecuencias**: Comportamiento idempotente: reenviar el mismo payload no duplica facturas.

**ADR-003**: api_key en `system_settings` con generación automática.
- **Contexto**: No existe gestión de tokens en el proyecto; `system_settings` es el mecanismo KV existente.
- **Decisión**: Clave `webhook_api_key` (32+ bytes aleatorios hex). Se genera en el arranque si falta. Endpoints autenticados para leer/regenerar.
- **Consecuencias**: Simple y centralizado. La clave se guarda en texto plano (igual que SMTP password/voicemonkey key existentes) — aceptable para uso personal interno.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Proyecto externo (automatización)]
        |  POST /webhooks/{uuid}
        |  Header: X-Webhook-Key: <api_key>
        |  Body: { year, month, amount, status, ... }
        v
[Server Go - mux]
   ├── WebhookAuth (valida api_key global)  -> 401 si falla
   ├── Resolver uuid -> service_id          -> 404 si no existe
   └── BillService.UpsertByPeriod()
        ├── FindByServicePeriod(service_id, year, month)
        ├── Create(...) o Update(...) / Pay(...)
        └── [SQLite: services + bills + system_settings]
```

### 4.2 Componentes

#### 4.2.1 `internal/api/webhook_handlers.go` (nuevo)
- **Responsabilidad**: Handler `POST /webhooks/{uuid}` + handlers autenticados de gestión de api_key (`GET /api/webhook/key`, `POST /api/webhook/key/regenerate`) y de URL por servicio si se aplica REQ-013.
- **Interfaz**: Ruta pública (no `AuthMiddleware`) con middleware propio de api_key.
- **Dependencias**: `BillStorage`, `ServiceStorage`, `SystemSettingsStorage`, `crypto/rand`, `crypto/subtle`.
- **Ubicación**: `internal/api/webhook_handlers.go`.

#### 4.2.2 `internal/api/middleware.go` (extender)
- **Responsabilidad**: Agregar `WebhookAuth` que valida el header `X-Webhook-Key` contra `system_settings.webhook_api_key`.
- **Dependencias**: `SystemSettingsStorage`.

#### 4.2.3 `internal/storage/service.go` (extender)
- **Responsabilidad**: Agregar `FindByWebhookUUID(ctx, uuid) (*models.Service, error)`.
- **Dependencias**: `services` con columna `webhook_uuid`.

#### 4.2.4 `internal/storage/bill.go` (extender)
- **Responsabilidad**: Agregar método de upsert por período que centralice crear/actualizar/pagar según el payload (reutiliza `FindByServicePeriod`, `Create`, `Update`, `Pay`).
- **Dependencias**: `bills`.

#### 4.2.5 `internal/storage/system_settings.go` (extender)
- **Responsabilidad**: Agregar `GetOrCreateWebhookKey(ctx)` que genera y persiste la api_key si no existe.
- **Dependencias**: `system_settings`.

#### 4.2.6 `internal/models/service.go` y `bill.go` (extender)
- **Responsabilidad**: Agregar `WebhookUUID string \`json:"webhook_uuid,omitempty"\`` a `Service`. (Los campos de Bill ya existen.)
- **Dependencias**: ninguno.

#### 4.2.7 `frontend/src/components/WebhookModal.tsx` (nuevo)
- **Responsabilidad**: Modal del webhook de un servicio. Muestra la URL `{base_url}/webhooks/{uuid}` (solo lectura con botón copiar), botón "Regenerar uuid" y la api_key global (con copiar). Si `webhook_enabled` está apagado, muestra un aviso de que la feature está deshabilitada en Settings. La `base_url` se lee de `system_settings.webhook_base_url` (default `http://ihost.local:8088`). Con tokens del tema y verificado en darkmode.
- **Dependencias**: `api` client + i18n.

#### 4.2.8 `frontend/src/pages/ServicesPage.tsx` (modificar)
- **Responsabilidad**: Agregar opción **"Webhook"** al `CardMenu` de cada card de servicio, que abre `WebhookModal`. Se quita cualquier referencia al webhook del formulario de edición.
- **Dependencias**: `WebhookModal`.

#### 4.2.9 `frontend/src/pages/ServiceFormPage.tsx` (modificar)
- **Responsabilidad**: **Quitar** la sección Webhook previamente agregada. El webhook vive en el modal desde `ServicesPage` y en Settings para la api_key.

#### 4.2.10 `frontend/src/pages/SettingsPage.tsx` (modificar)
- **Responsabilidad**: Nueva sección **"Webhooks"** con toggle maestro `webhook_enabled`, campo editable `webhook_base_url` (default `http://ihost.local:8088`) y api_key global (monospace, con copiar y regenerar). Con OFF, hint de que la feature está deshabilitada. Con tokens del tema y darkmode.
- **Dependencias**: `api.webhooks.*` + i18n.

#### 4.2.11 `frontend/src/api/index.ts` y `frontend/public/i18n/{es,en}.json` (modificar)
- **Responsabilidad**: Endpoints del cliente (`webhooks.getApiKey`, `webhooks.regenerateApiKey`, `services.regenerateWebhook`) + claves i18n nuevas (`services.webhook.*`, `settings.webhooks.*`). **Editar SIEMPRE en `frontend/public/i18n/`** (regla AGENTS.md), nunca en `public/i18n/`.

### 4.3 Modelo de datos

Migración `0027_add_services_webhook_uuid.up.sql` / `.down.sql`:

```sql
-- up
ALTER TABLE services ADD COLUMN webhook_uuid TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_webhook_uuid ON services(webhook_uuid) WHERE webhook_uuid IS NOT NULL;

-- down
DROP INDEX IF EXISTS idx_services_webhook_uuid;
ALTER TABLE services DROP COLUMN webhook_uuid;  -- SQLite >= 3.35
```

**Backfill**: al arrancar (o en la migración vía trigger no posible), el código garantiza que todo servicio sin `webhook_uuid` reciba uno (generación lazy en `GetService`/webhook lookup + al crear servicio). Se documenta en REQ-010.

**system_settings**: `webhook_api_key` → string hex aleatorio (64 chars, 32 bytes).

### 4.4 APIs / Contratos

#### Endpoint: `POST /webhooks/{uuid}` (público, requiere `X-Webhook-Key`)

**Request headers**:
```
X-Webhook-Key: <api_key_global>
Content-Type: application/json
```

**Request body** (schema al que deben adaptarse los clientes):
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

**Response 200**:
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
  "created": false
}
```

**Response Error**:
```json
{ "error": "invalid_body", "message": "month must be between 1 and 12" }
```

| Código | Caso |
|--------|------|
| 200 | Factura creada o actualizada (`created: true/false`) |
| 400 | Body malformado o validación fallida |
| 401 | Header `X-Webhook-Key` faltante o incorrecto |
| 403 | `webhook_enabled` está deshabilitado en Settings |
| 404 | `uuid` no corresponde a ningún servicio activo |

#### Endpoint: `GET /api/webhook/key` (autenticado por sesión)
**Response 200**:
```json
{ "api_key": "a1b2..." }
```

#### Endpoint: `POST /api/webhook/key/regenerate` (autenticado por sesión)
**Response 200**:
```json
{ "api_key": "nuevo..." }
```

#### Endpoint: `POST /api/services/{id}/webhook/regenerate` (autenticado por sesión, REQ-008)
**Response 200**:
```json
{ "webhook_uuid": "nuevo-uuid", "webhook_url": "http://host:8088/webhooks/nuevo-uuid" }
```

### 4.5 Dependencias

- **Internas**: `routes.go` (registro de rutas), `service_handlers.go`/`service.go` (uuid), `bill.go` (upsert), `middleware.go` (WebhookAuth), `system_settings.go` (api_key + toggle `webhook_enabled` + `webhook_base_url`), `SettingsPage.tsx` (toggle + base URL + api_key), `WebhookModal.tsx` + `ServicesPage.tsx`/`BillsPage.tsx` (modal webhooks), `api/index.ts` + i18n (UI).
- **Externas**: Ninguna nueva. Solo stdlib Go (`crypto/rand`, `crypto/subtle`) y utilidades existentes.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [x] CA-001: Dado un servicio con `webhook_uuid`, cuando un cliente hace `POST /webhooks/{uuid}` con header `X-Webhook-Key` correcto y body `{year, month, amount}`, entonces se crea la factura pendiente y responde `200` con `created: true`.
- [x] CA-002: Dada una factura existente `(service_id, year, month)`, cuando se reenvía el webhook con `status: "paid"` y `payment_reference`, entonces la factura queda `paid` con `paid_at` y referencia guardadas (`created: false`).
- [x] CA-003: Dado un webhook sin header o con api_key incorrecta, cuando se invoca el endpoint, entonces responde `401` y no modifica nada.
- [x] CA-004: Dado un `uuid` inexistente, cuando se invoca el webhook, entonces responde `404`.
- [x] CA-005: Dado un body con `month: 13` o `amount: -1` o `status: "cancelado"`, cuando se invoca el webhook, entonces responde `400` con mensaje claro.
- [x] CA-006: Dado un servicio recién creado, cuando se consulta su GET, entonces incluye `webhook_uuid` no vacío; los servicios existentes también lo tienen tras el backfill.
- [x] CA-007: Dado un usuario autenticado, cuando consulta `GET /api/webhook/key` y regenera con `POST /api/webhook/key/regenerate`, entonces obtiene/actualiza la api_key y la nueva clave invalida la anterior.
- [x] CA-008: Dado el menú de 3 puntos de un servicio en `ServicesPage`, cuando se elige "Webhook", entonces se abre `WebhookModal` con la URL del webhook (copiar + regenerar uuid); el formulario de edición NO muestra la sección webhook.
- [x] CA-009: Dado Settings con el toggle `webhook_enabled` OFF, cuando un cliente invoca `POST /webhooks/{uuid}`, entonces responde `403`; con el toggle ON responde según el caso (200/401/404/400).
- [x] CA-010: Dado Settings, cuando el toggle `webhook_enabled` está ON, entonces se muestra la api_key global con opciones de copiar y regenerar; con OFF, hint de feature deshabilitada.
- [x] CA-011: Dado `WebhookModal` abierto con `webhook_enabled` OFF, entonces muestra aviso de que la feature está deshabilitada y no muestra la URL.
- [x] CA-012: Dado Settings, cuando se edita `webhook_base_url`, entonces el modal y el handler de regeneración construyen las URLs con esa base; el default es `http://ihost.local:8088`.
- [x] CA-DARK: Los inputs/UI nuevos (modal webhook y sección webhooks en Settings) usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (texto y placeholders).
- [x] CA-BACK: No aplica páginas de detalle nuevas con lista padre (el webhook vive en un modal desde `ServicesPage` y en Settings).

### 5.2 No funcionales

- [x] CA-NF-001: El endpoint de webhook responde en < 200ms local con SQLite.
- [x] CA-NF-002: Sin dependencias externas nuevas; `go build` y `go test ./...` pasan.
- [x] CA-NF-003: El build del frontend (`npm run build` en `frontend/`) pasa y las claves i18n nuevas se sirven en `/i18n/es.json` y `/i18n/en.json` (editadas en `frontend/public/i18n/`).

### 5.3 Testing

- **Unit tests**:
  - `webhook` upsert: creación, actualización, pago, y que no duplica con payload idéntico.
  - Validación de body (mes, año, monto, estado) y auth (api_key faltante/incorrecta).
  - `FindByWebhookUUID` (encontrado, no encontrado, servicio eliminado).
  - `GetOrCreateWebhookKey` (genera una vez, reutiliza después).
  - Migración `0027`: backfill de `webhook_uuid` en servicios existentes.
- **Integration tests**: `POST /webhooks/{uuid}` completo con api_key real contra DB temporal; regeneración de api_key invalida la anterior.
- **E2E tests**: Prueba manual local: crear servicio → copiar URL webhook → `curl -X POST` con la api_key → verificar factura creada y/o pagada en UI; error 401 sin key.
- **Carga/Performance**: Sin impacto significativo (rutas simples sobre SQLite); verificar con `./scripts/check-server.sh` tras levantar en local.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0027` (columna `webhook_uuid` + índice) + modelo `Service.WebhookUUID` + backfill lazy | 0.5 día | Ninguna |
| 2 | Backend: `GetOrCreateWebhookKey`, `WebhookAuth` middleware, `FindByWebhookUUID`, upsert en `BillStorage`, handlers + rutas | 1.5 días | Fase 1 |
| 3 | Tests backend (unit + integration) | 1 día | Fase 2 |
| 4 | Frontend: sección webhook en `ServiceFormPage`, endpoints en `api.ts`, claves i18n (en `frontend/public/i18n/`) + build | 1 día | Fase 2 |
| 5 | Validación local (server corriendo, `check-server.sh`), pruebas manuales con curl y UI, darkmode | 1 día | Fases 2-4 |

### 6.2 Milestones

1. **MVP**: Fases 1-2-3 — webhook funcional por API (el proyecto externo puede empujar facturas).
2. **V1.0**: Fase 4-5 — UI de configuración del webhook + validación completa.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| La api_key en `system_settings` en texto plano | Media | Medio | Aceptable para uso personal interno (mismo patrón que SMTP/voicemonkey). Documentar. |
| Colisión/duplicación de `webhook_uuid` | Baja | Alto | Índice UNIQUE + generación con `crypto/rand` (128 bits). |
| Cliente externo reenvía factura pagada como `pending` y "despaga" | Media | Medio | Documentar en el schema que `status` explícito manda; si se omite, no se toca el estado actual (salvo creación). |
| Backfill de servicios existentes sin uuid | Media | Medio | Generación lazy garantizada al consultar; cubrir con test de migración. |
| Rotura de integraciones al regenerar uuid/api_key | Media | Medio | UI clara de regeneración + confirmación; documentar que rompe el cliente externo hasta actualizar la URL/clave. |

## 8. Notas y Referencias

- Schema de referencia (espejo de `models.Bill`): `internal/models/bill.go`; tabla `bills` en `migrations/0005_create_bills.up.sql`.
- Flujo de pago existente (reutilizado): SPEC-043; upsert del analizador: SPEC-021.
- Settings KV: `internal/storage/system_settings.go` y `internal/storage/settings.go`.
- Registro de rutas: `internal/api/routes.go`.
- Precedente de fix i18n (editar en `frontend/public/i18n/`): SPEC-032/033.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-11 | paulomcnally | Creación inicial de la especificación |
| 2026-09-11 | paulomcnally | Estado a `pending_execution` (api_key global confirmada por el usuario). Issue #72 label `spec/pending-execution`. |
| 2026-09-11 | paulomcnally | Implementación backend completa: migración 0027 (`webhook_uuid` en services), `WebhookService` (api_key global + upsert de facturas), middleware `WebhookAuth` (header `X-Webhook-Key`), handlers `POST /webhooks/{uuid}`, `GET/POST /api/webhook/key`, `POST /api/services/{id}/webhook/regenerate`, backfill de uuid al arranque. Frontend: sección Webhook en `ServiceFormPage` (URL + copiar + regenerar uuid/api_key), iconos `copy`/`link`, i18n es/en. Tests: `webhook_test.go` (services) y `webhook_handlers_test.go` (api). Validación local: 13 escenarios curl OK (401/404/400/200, crear/actualizar/pagar/revertir, regenerar clave/uuid invalida anteriores). `go build`, `go test ./...` y `npm run build` pasan. |
| 2026-09-11 | paulomcnally | Rediseño UI (feedback del usuario): el webhook pasa del formulario de edición a un **modal** accesible desde el menú de 3 puntos de cada servicio (`WebhookModal`), la api_key global se gestiona en **Settings** con un **toggle maestro `webhook_enabled`** (feature OFF → endpoint responde 403, modal avisa que está deshabilitada). Backend: toggle `webhook_enabled` en `system_settings` + `SystemSettingsService.Get/SetWebhookEnabled` + `403` en `UpsertBill`. Frontend: `WebhookModal.tsx`, opción "Webhook" en `CardMenu` de `ServicesPage`, sección Webhooks en `SettingsPage` (toggle + api_key con copiar/regenerar), i18n es/en actualizado. Tests: `TestWebhookDisabledReturns403`. Validado local contra DB de producción: 403 con feature OFF, 200 con ON, toggle persiste, i18n servido. |
| 2026-09-11 | paulomcnally | El modal webhooks también se accede desde el menú del header de `BillsPage` (página de facturas del servicio). Título renombrado a "Webhooks" (es/en). |
| 2026-09-11 | paulomcnally | **Base URL configurable** (REQ-008a): la URL del webhook apunta por defecto a `http://ihost.local:8088` (clave `webhook_base_url` en `system_settings`), editable en Settings → Webhooks. `SystemSettingsService.Get/SetWebhookBaseURL`, `WebhookService.WebhookURL(ctx, uuid)`, campo `webhook_base_url` en `GET/PUT /api/system-settings`, `WebhookModal` y handler de regeneración usan la base configurada. Test `TestWebhookURLDefaultAndCustomBase`. Validado: default ihost.local, edición a localhost:9000 reflejada en URLs, restauración a default con valor vacío. |
| 2026-09-11 | paulomcnally | Release: merge a `main`, issue #72 cerrado con label `spec/released`. Commit: `2d616c7` — `SPEC-069: webhooks por servicio para facturas (upsert + api_key global + base URL configurable)`. Push a `origin/main` completado. |