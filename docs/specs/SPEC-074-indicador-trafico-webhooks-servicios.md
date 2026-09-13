---
title: "Indicador de tráfico de webhooks en cards de servicios (last_webhook_request)"
id: "SPEC-074"
status: "in_progress"
author: "paulomcnally"
created: "2026-09-12"
updated: "2026-09-12"
github_issue: 77
---

# Indicador de tráfico de webhooks en cards de servicios (last_webhook_request)

**ID**: SPEC-074  
**Estado**: in_progress  
**Autor**: paulomcnally  
**Creado**: 2026-09-12  
**Actualizado**: 2026-09-12

---

## 1. Resumen Ejecutivo

Los servicios pueden tener webhooks configurados (SPEC-069) para recibir facturas desde sistemas externos (bancos, APIs de terceros, etc.), pero actualmente **no hay forma de saber si un servicio está recibiendo tráfico real de webhook**. Un servicio puede tener la URL configurada en un cliente externo que nunca envía datos, o el sistema externo puede no estar integrado todavía. El usuario necesita identificar de un vistazo, en las cards de la página de Servicios, qué servicios están recibiendo tráfico de webhook y cuáles no.

La solución es registrar a nivel de servicio un campo `last_webhook_request` (timestamp) que se actualiza cada vez que el endpoint `POST /webhooks/{uuid}` recibe un request para ese servicio. La card del servicio muestra un indicador simple (badge con la fecha del último request) cuando el campo está presente. La ausencia del campo indica que el servicio aún no tiene un sistema de webhooks operativo (o no está enviando datos).

Impacto en iHost: mínimo. Es una columna TIMESTAMP adicional en `services`, un `UPDATE` de 1 fila por request de webhook (frecuencia baja) y un badge en el frontend estático. Sin dependencias nuevas, sin cambios de esquema en otras tablas.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar columna `last_webhook_request` (TIMESTAMP, nullable) a la tabla `services` vía migración `0029`.
2. **REQ-002**: Registrar la fecha/hora actual en `last_webhook_request` del servicio cada vez que `POST /webhooks/{uuid}` recibe un request cuyo UUID corresponde a un servicio activo (independientemente de la validez del payload, ya que el request es tráfico real hacia la URL).
3. **REQ-003**: Exponer `last_webhook_request` en la respuesta del modelo `Service` (JSON `last_webhook_request`) para que el frontend lo consuma en el listado de servicios.
4. **REQ-004**: Mostrar en **TODAS** las cards de servicios (ServicesPage) un indicador con el ícono de webhook y el texto de la fecha **relativa** (ej: "Hace 10 días", "Hace 2 horas", "Hace 5 minutos"). Si `last_webhook_request` es NULL, el indicador muestra "Nunca" (el servicio aún no recibió tráfico de webhook).
5. **REQ-005**: No hacer backfill: los servicios existentes quedan con `NULL` hasta que reciban su primer request (no hay dato histórico confiable).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-006**: Agregar claves i18n (es/en) para el texto del indicador en `frontend/public/i18n/` y regenerar el build de Vite.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-007**: (Opcional, fuera de alcance inicial) Mostrar el indicador también en la página de detalle de facturas del servicio si el usuario lo pide.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El registro del timestamp es un único `UPDATE` por request de webhook; impacto despreciable en SQLite WAL.
- **Seguridad**: No se expone información sensible; solo un timestamp por servicio en el JSON ya devuelto por `GET /api/services` (endpoint autenticado por sesión).
- **Almacenamiento**: 1 columna TIMESTAMP nullable (8 bytes + overhead); sin rotación necesaria.
- **Disponibilidad**: Sin cambios en health checks ni en rutas existentes.
- **iHost**: Sin dependencias nuevas; cero costo en memoria runtime del backend; el frontend es estático pre-build.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Flujo actual de webhooks: `POST /webhooks/{uuid}` → `WebhookHandlers.UpsertBill` (internal/api/webhook_handlers.go) → busca servicio por UUID → `WebhookService.UpsertBill` (internal/services/webhook.go) → upsert de factura.
- Modelo `Service` (internal/models/service.go) ya tiene `WebhookUUID`; el storage de servicios (`internal/storage/service.go`) usa `serviceColumns` compartido para todos los SELECT.
- Última migración existente: `0028_create_bill_history`. La próxima es `0029`.
- Frontend: `ServicesPage.tsx` renderiza cards con chips de estado (Inactivo/Vencido) — patrón de UI a reutilizar para el badge de webhook.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Columna `last_webhook_request` en `services` | Lectura trivial en listados (mismo SELECT), sin joins, costo mínimo, semántica clara | No hay historial (solo último request) | ✅ Seleccionada |
| Tabla `service_webhook_events` con historial | Permite métricas/gráficos históricos | Tabla nueva, joins en listados, más almacenamiento, overkill para el objetivo | ❌ Rechazada |
| Derivar de `bill_history` (source=webhook) | Sin columna nueva | No cubre requests con payload inválido, join + subquery en cada listado, más complejo | ❌ Rechazada |
| Badge con fecha vs indicador booleano | La fecha agrega contexto útil sin costo extra | — | ✅ Seleccionada (badge con fecha) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001: Registrar el timestamp al matchear el UUID (tráfico), no solo en upserts exitosos**
- **Contexto**: El objetivo del usuario es saber qué servicios "están recibiendo tráfico de webhooks". Un request con payload inválido o status rechazado sigue siendo tráfico real hacia la URL del webhook (indica que el sistema externo está configurado y enviando).
- **Decisión**: En `WebhookHandlers.UpsertBill`, una vez encontrado el servicio por UUID, se registra `last_webhook_request` antes de procesar/validar el payload. El registro vive en `WebhookService.RecordWebhookRequest(ctx, serviceID)` (capa de negocio) que delega en `ServiceStorage.SetLastWebhookRequest` (única capa con SQL).
- **Consecuencias**: Un servicio con cliente mal configurado (payload inválido) también aparece como "con tráfico", lo cual es correcto para el objetivo (el sistema externo está integrado). Costo: 1 UPDATE extra por request.

**ADR-002: Columna nullable sin backfill**
- **Contexto**: No existe dato histórico de tráfico de webhooks; backfill sería inventar datos.
- **Decisión**: Migración `ALTER TABLE services ADD COLUMN last_webhook_request TIMESTAMP;` sin valores por defecto. `NULL` = nunca recibió un request.
- **Consecuencias**: Los servicios ya integrados muestran el indicador recién tras su próximo request real. Comportamiento aceptable y honesto.

**ADR-003: Indicador visible en TODAS las cards con fecha relativa**
- **Contexto**: El usuario quiere identificar de un vistazo el estado de webhooks de TODOS los servicios, no solo los que tienen tráfico. Un badge solo-cuando-hay-dato no permite distinguir "sin tráfico" de "sin webhook" en una lista larga.
- **Decisión**: Todas las cards muestran el chip con ícono `link`. Con `last_webhook_request` presente, muestra la fecha relativa ("Hace 10 días", "Hace 2 horas") calculada con `Intl.RelativeTimeFormat` (locale del i18n activo, sin dependencias). Con `NULL`, muestra "Nunca" (es) / "Never" (en) en estilo gris.
- **Consecuencias**: Consistencia visual en toda la lista; el estado NULL es explícito. Costo: un cálculo de fecha por card en el frontend (despreciable).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Cliente externo] --POST /webhooks/{uuid}--> [WebhookHandlers.UpsertBill]
                                                    |
                                                    v
                                    [WebhookService.RecordWebhookRequest]
                                                    |
                                                    v
                                    [ServiceStorage.SetLastWebhookRequest]
                                                    |
                                                    v
                                        [SQLite: services.last_webhook_request]
                                                    |
                                                    v
[ServicesPage] <--GET /api/services-- [ServiceStorage.List (serviceColumns)]
```

### 4.2 Componentes

#### 4.2.1 Migración 0029
- **Responsabilidad**: Agregar la columna `last_webhook_request` a `services`.
- **Interfaz**: `migrations/0029_add_services_last_webhook_request.up.sql` / `.down.sql`.
- **Dependencias**: Ninguna.
- **Ubicación**: `migrations/`

#### 4.2.2 Modelo `Service`
- **Responsabilidad**: Exponer el nuevo campo en JSON.
- **Interfaz**: `LastWebhookRequest *time.Time \`json:"last_webhook_request,omitempty"\``.
- **Dependencias**: `time`.
- **Ubicación**: `internal/models/service.go`

#### 4.2.3 Storage de servicios
- **Responsabilidad**: Persistir y leer el campo.
- **Interfaz**: `SetLastWebhookRequest(ctx, id int64, at time.Time) error`; agregar `last_webhook_request` a `serviceColumns` y al scan en `scanService`/`scanServices` (sql.NullTime).
- **Dependencias**: `database/sql`.
- **Ubicación**: `internal/storage/service.go`

#### 4.2.4 Servicio de webhooks
- **Responsabilidad**: Lógica de negocio del registro de tráfico.
- **Interfaz**: `RecordWebhookRequest(ctx context.Context, serviceID int64) error` (delega en storage; error no bloquea el upsert, solo se loguea).
- **Dependencias**: `ServiceStorage`.
- **Ubicación**: `internal/services/webhook.go`

#### 4.2.5 Handler de webhooks
- **Responsabilidad**: Invocar el registro tras matchear el UUID.
- **Interfaz**: En `UpsertBill`, tras `FindByWebhookUUID` exitoso, llamar `RecordWebhookRequest`.
- **Ubicación**: `internal/api/webhook_handlers.go`

#### 4.2.6 Frontend ServicesPage
- **Responsabilidad**: Mostrar el indicador de tráfico de webhook en todas las cards.
- **Interfaz**: Tipo `Service.last_webhook_request?: string | null` en `frontend/src/types/index.ts`; helper `formatRelativeTime(iso, lang)` (Intl.RelativeTimeFormat) en `frontend/src/utils/relativeTime.ts`; chip con ícono `link` + texto relativo en `frontend/src/pages/ServicesPage.tsx`.
- **Dependencias**: i18n.
- **Ubicación**: `frontend/src/pages/ServicesPage.tsx`, `frontend/src/utils/relativeTime.ts`

### 4.3 Modelo de datos

```
Entidad: services
- last_webhook_request: TIMESTAMP nullable (último request de webhook recibido)
- (existente) webhook_uuid: TEXT nullable (UUID único del webhook del servicio)
- Relaciones: sin cambios
```

**Migración up** (`0029_add_services_last_webhook_request.up.sql`):
```sql
ALTER TABLE services ADD COLUMN last_webhook_request TIMESTAMP;
```

**Migración down** (`0029_add_services_last_webhook_request.down.sql`):
```sql
ALTER TABLE services DROP COLUMN last_webhook_request;
```

### 4.4 APIs / Contratos

#### Cambio en `GET /api/services` (y GET por id) — modelo `Service` extendido

**Response (fragmento)**:
```json
{
  "id": 3,
  "name": "Claro Internet",
  "webhook_uuid": "abc-123",
  "last_webhook_request": "2026-09-12T20:15:03Z"
}
```

`last_webhook_request` se omite (omitempty) cuando es `NULL`.

#### Endpoint: `POST /webhooks/{uuid}` (sin cambios de contrato, efecto secundario)

- Al matchear el UUID: se actualiza `last_webhook_request` con la hora actual del servidor.
- Respuestas: sin cambios (200 con `WebhookResult`; 400/403/404 según los casos existentes).

### 4.5 Dependencias

- **Internas**: `ServiceStorage`, `WebhookService`, `WebhookHandlers`, `ServicesPage`, `types/index.ts`, i18n es/en.
- **Externas**: Ninguna.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un servicio con webhook, cuando se envía `POST /webhooks/{uuid}` con payload válido, entonces `last_webhook_request` del servicio queda con la fecha/hora del request y aparece en `GET /api/services`.
- [ ] CA-002: Dado un servicio con webhook, cuando se envía `POST /webhooks/{uuid}` con payload inválido (ej: year fuera de rango), entonces igual se actualiza `last_webhook_request` (el request es tráfico real).
- [ ] CA-003: Dado un request con UUID inexistente o webhooks deshabilitados, entonces NO se actualiza ningún `last_webhook_request` (no hay servicio match).
- [ ] CA-004: Dado el listado de servicios en ServicesPage, entonces TODAS las cards muestran el indicador con ícono `link`: fecha relativa ("Hace 10 días", "Hace 2 horas", "Hace 5 minutos") cuando hay `last_webhook_request`, o "Nunca" (estilo gris) cuando es NULL.
- [ ] CA-005: Dado un servicio sin `last_webhook_request` (NULL), entonces su card muestra "Nunca" en el indicador.
- [ ] CA-006: Dados servicios existentes antes de la migración, entonces todos quedan con `last_webhook_request = NULL` (sin backfill).
- [ ] CA-007: La migración `0029` up/down funciona sobre una DB de prueba en local (`/tmp/test-app.db`), y los tests existentes del módulo webhook siguen pasando.
- [ ] CA-008: Las claves i18n nuevas existen en `frontend/public/i18n/{es,en}.json`, el build de Vite se regenera y el server sirve las claves (`curl /i18n/es.json`).

### 5.2 No funcionales

- [ ] CA-NF-001: El registro del timestamp no agrega dependencias nuevas ni más de 1 UPDATE por request de webhook.
- [ ] CA-NF-002: Sin cambios en el contrato de respuestas de `POST /webhooks/{uuid}` (compatibilidad con clientes externos).

### 5.3 Testing

- **Unit tests**: `ServiceStorage.SetLastWebhookRequest` persiste y lee el valor; `WebhookService.RecordWebhookRequest` delega correctamente.
- **Integration tests**: Test del handler `UpsertBill` verificando que `last_webhook_request` se actualiza con payload válido e inválido, y NO se actualiza con UUID inexistente/feature deshabilitada (extender `webhook_handlers_test.go` / `webhook_test.go`).
- **E2E tests**: Manual — correr server local, enviar curl al webhook de un servicio, verificar el badge en la card de ServicesPage (light + darkmode).
- **Carga/Performance**: No aplica (1 UPDATE por request, frecuencia baja).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0029` (up/down) + campo en modelo `Service` + storage (`serviceColumns`, scan, `SetLastWebhookRequest`) | 0.5 día | Ninguna |
| 2 | `WebhookService.RecordWebhookRequest` + invocación en `WebhookHandlers.UpsertBill` | 0.5 día | Fase 1 |
| 3 | Frontend: tipo `Service`, badge en card de ServicesPage, i18n es/en + build Vite | 0.5 día | Fase 1 |
| 4 | Tests unitarios/integración backend + validación local completa (server + curl + darkmode) | 0.5 día | Fases 2-3 |

### 6.2 Milestones

1. **MVP**: Fases 1-3 — el badge aparece en las cards y el backend registra el tráfico.
2. **V1.0**: Fase 4 — tests y validación manual del usuario en local.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| `DROP COLUMN` no soportado por la versión de SQLite embebida | Baja | Medio | modernc.org/sqlite actual soporta DROP COLUMN (SQLite 3.35+); verificar en pruebas locales; el down solo se usa en dev |
| Escritura extra por request de webhook en WAL | Baja | Bajo | 1 UPDATE de 1 fila indexada por PK; frecuencia de webhooks baja |
| Olvidar editar i18n en `frontend/public/i18n/` y que el build borre las claves | Media | Medio | Regla crítica de AGENTS.md: editar SIEMPRE en `frontend/public/i18n/` y verificar con `curl /i18n/es.json` (precedente SPEC-032/033) |
| Badge ilegible en darkmode | Baja | Bajo | Usar tokens del tema (`bg-primary/10`, `text-primary`) y verificar en darkmode (regla SPEC-060) |

## 8. Notas y Referencias

- SPEC-069: Webhooks por servicio para facturas (origen del endpoint `POST /webhooks/{uuid}`).
- SPEC-070: Historial de cambios de facturas (bill_history, descartado como fuente de datos para este indicador).
- SPEC-071: Reactivación de facturas soft-deleted vía webhook.
- SPEC-072: Modal de webhook en servicios (copiar URL).
- `internal/services/webhook.go`, `internal/api/webhook_handlers.go`, `internal/storage/service.go`, `frontend/src/pages/ServicesPage.tsx`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-12 | paulomcnally | Creación inicial de la especificación |
| 2026-09-12 | paulomcnally | Cambio solicitado por usuario: TODAS las cards muestran el indicador con ícono + fecha relativa ("Hace 10 días"); con NULL muestra "Nunca" (REQ-004, ADR-003, CA-004/005 actualizados) |