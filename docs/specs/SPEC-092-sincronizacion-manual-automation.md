---
title: "Sincronización manual de servicios con los plugins de p40la-ihost-automation (endpoint + UI + bot)"
id: "SPEC-092"
status: "in_progress"
author: "opencode"
created: "2026-09-24"
updated: "2026-09-24"
github_issue: 95
---

# Sincronización manual de servicios con los plugins de p40la-ihost-automation (endpoint + UI + bot)

**ID**: SPEC-092  
**Estado**: in_progress  
**Autor**: opencode  
**Creado**: 2026-09-24  
**Actualizado**: 2026-09-24

---

## 1. Resumen Ejecutivo

Las facturas de algunos servicios (Claro, Tigo, DISNORTE/DISSUR, ENACAL, ASSA) se obtienen automáticamente desde el proyecto hermano **`p40la-ihost-automation`** (puerto `8089`), que ejecuta los **plugins** (llamado externo al proveedor) y luego **empuja** cada factura al webhook de `p40la-ihost` (`POST /webhooks/{uuid}`, SPEC-069). Hoy ese flujo solo corre por el scheduler del proyecto de automation o manualmente desde su propia UI (botón *test* de webhook). No existe forma de dispararlo desde `p40la-ihost` (dashboard ni bot de Telegram).

Esta spec agrega una **sincronización manual**: un **endpoint** `POST /api/services/{id}/sync` que llama a los plugins de automation (ejecuta el llamado externo y el envío al webhook de forma manual), una **acción "Sincronizar"** en el menú de 3 puntos de cada servicio en la UI, y un **comando de bot** `/sincronizar_servicio <id>` donde `<id>` es el ID del registro. Para facilitar el uso del comando, las **listas de servicios y deudas exponen el ID** de cada registro (`ID: 1`).

El trabajo abarca **dos repos**: `p40la-ihost` (endpoint, UI, bot) y **`p40la-ihost-automation`** (nuevo endpoint de ejecución manual del plugin + webhook, autenticado por token server-to-server). El cambio en automation se documenta aquí y se trackea con su propia spec en ese repo (sistema de specs propio de `p40la-ihost-automation`).

**Consideraciones iHost**: solución backend-only + bot + pequeña UI. Sin dependencias externas nuevas (client HTTP de la stdlib). El vínculo servicio ↔ cuenta de automation se guarda en una columna nullable de `services` y el endpoint de automation (base URL + api_key) en `system_settings` (mismo patrón que SMTP/VoiceMonkey). Memoria/CPU despreciables: una ruta, un cliente HTTP y un handler de bot.

**Consideraciones de UI**: los únicos controles nuevos son campos de texto en formularios existentes (ServiceFormPage y Settings) y un item de menú; deben usar tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y verificarse en darkmode. No hay páginas de detalle nuevas, por lo que no aplica `BACK_ROUTES` (SPEC-063).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Endpoint `POST /api/services/{id}/sync` (autenticado por sesión) que dispara la sincronización manual del servicio: ejecuta los plugins de `p40la-ihost-automation` (llamado externo) y el envío al webhook del servicio (upsert de facturas, SPEC-069). Devuelve `{ delivered, failed }`.
2. **REQ-002** (cambio en `p40la-ihost-automation`): nuevo endpoint `POST /api/accounts/{accountId}/webhook:run` que ejecuta `WebhookService.RunJob` (fetch del plugin + entrega al webhook). Autenticado server-to-server por token en header `X-Webhook-Key` (la api_key de webhooks del propio automation), sin requerir sesión. Se registra en `routes.go` fuera de `AuthMiddleware`. Este cambio se trackea con una spec propia en el repo de automation.
3. **REQ-003**: `AutomationClient` en `p40la-ihost`: llama a `POST {base}/api/accounts/{automation_account_id}/webhook:run` con header `X-Webhook-Key` (api_key de webhooks de automation). Config en `system_settings`: `automation_base_url` (default `http://ihost.local:8089`) y `automation_api_key`. Sin login ni cookies de sesión. No loguear la api_key.
4. **REQ-004**: Vínculo servicio ↔ cuenta de automation: columna nullable `automation_account_id` en `services` (migración `0034`), editable en `ServiceFormPage`.
5. **REQ-005**: Acción **"Sincronizar"** en el menú de 3 puntos de cada card de servicio (`ServicesPage`): ejecuta el sync, muestra estado de carga y toast con el resultado (entregadas/fallidas) o el error.
6. **REQ-006**: Comando de bot `/sincronizar_servicio <id>`: sincroniza el servicio con ese ID y responde con el resultado (`✅ X entregadas / Y fallidas`) o un error claro. Sin argumento o ID no numérico → mensaje de ayuda.
7. **REQ-007**: Exponer el **ID** del registro en la lista de **servicios** (`ServicesPage`): badge discreto `ID: N` en cada card.
8. **REQ-008**: Exponer el **ID** del registro en la lista de **deudas** (`DeudasPage`): badge discreto `ID: N` en cada card.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-009**: Registro del comando `/sincronizar_servicio` en `setMyCommands` del bot (SPEC-080) con descripción y ayuda en `/start` y en el fallback.
2. **REQ-010**: Manejo de errores del sync con códigos HTTP semánticos: `404` servicio inexistente, `400` servicio sin `automation_account_id` o automation sin configurar, `401` api_key de automation inválida, `502` automation inalcanzable. Errores de automation (`webhook_enabled` off, sin api_key, sin plugin, sin webhook_url) se propagan con mensaje legible.
3. **REQ-011**: Timeout y resiliencia del client: timeout HTTP de 30s (el llamado externo al proveedor puede tardar). Auth stateless por token (sin cookies de sesión ni re-login).

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-012**: Mostrar el estado del último sync en la card de servicio (p. ej. reutilizar `last_webhook_request` ya expuesto, SPEC-074) y, si aplica, una accion de sync también desde `BillsPage` del servicio.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: El sync es síncrono y puede tardar segundos (llamado externo al proveedor). La UI debe mostrar loading; el bot responde al finalizar. Timeout total de 30s.
- **Seguridad**: La api_key de automation (token) se guarda en `system_settings` (texto plano, mismo patrón que SMTP/VoiceMonkey — aceptable para uso personal interno). Jamás loguear la api_key. El endpoint de sync exige sesión válida; el endpoint de run en automation exige el token.
- **Almacenamiento**: Una columna nullable en `services` + 2 claves en `system_settings`. Sin tablas nuevas.
- **Disponibilidad**: Si automation está caído, el sync responde `502` con mensaje claro; el resto de la app no se ve afectada.
- **iHost**: Sin dependencias nuevas (Go stdlib `net/http`). Consumo de memoria mínimo.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **Proyecto automation (`p40la-ihost-automation`, puerto 8089)**: expone `POST /api/accounts/{accountId}/webhook:test` (sesión-autenticado) que ejecuta `WebhookService.RunJob`: consulta las facturas con el plugin de la cuenta (`FetchBills`) y hace el POST a la `webhook_url` configurada con header `X-Webhook-Key` (`internal/services/webhook.go`). `RunJob` ya hace exactamente lo pedido ("llamado externo + webhook manual"), pero su única vía HTTP es la sesión (`AuthMiddleware`), inapropiada para server-to-server sin login. Por eso se agrega un **endpoint de run con auth por token** (REQ-002).
- **Token de automation**: `WebhookService.RunJob` valida `IsWebhookEnabled` + `GetWebhookAPIKey` (api_key de webhooks del propio automation, en `settings`). Esa misma api_key se reutiliza como token de entrada del nuevo endpoint `webhook:run` (header `X-Webhook-Key`), evitando crear una clave adicional.
- **Webhook de p40la-ihost (SPEC-069)**: `POST /webhooks/{uuid}` con api_key global en header, upsert de facturas por `(service_id, year, month)`. El servicio ya expone `webhook_uuid` y `last_webhook_request` (SPEC-074), por lo que tras un sync el tráfico queda visible en la card.
- **Modelo de servicio (p40la-ihost)**: `internal/models/service.go` — `Service` no tiene vínculo con automation. El vínculo implícito existente (la `webhook_url` de la cuenta de automation contiene el `webhook_uuid` del servicio) no es consultable de forma confiable por p40la-ihost sin listar cuentas de automation.
- **Bot (p40la-ihost)**: `internal/services/telegram_bot.go` — handlers con `MatchTypeCommand` (patrones sin slash, SPEC-080). Los comandos con argumentos reciben el texto completo en `update.Message.Text`; se parsea el argumento en el handler.
- **UI**: `ServicesPage.tsx` usa `CardMenu` (menú de 3 puntos) por card — lugar natural para "Sincronizar". `DeudasPage.tsx` usa cards por deuda. Las cards ya muestran badges; se agrega un badge `ID: N`.
- **Settings**: `system_settings` KV con `SystemSettingsService.Get/Set` — patrón para `automation_*` (precedentes: webhooks, SMTP, VoiceMonkey, telegram bot).
- **Migraciones**: en p40la-ihost la última es `0033`; la nueva será `0034`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| p40la-ihost reutiliza `POST /api/accounts/{accountId}/webhook:test` de automation (login con credenciales) | Cero cambios en automation; endpoint ya ejecuta plugin + webhook | Necesita email/password de automation en p40la-ihost; login por sesión con caché/re-login; frágil | ❌ Rechazada |
| Agregar endpoint `webhook:run` en automation con auth por token (reutiliza su api_key de webhooks) | Auth server-to-server stateless sin sesión; sin credenciales de usuario en p40la-ihost; mínimo cambio en automation | Un cambio en el repo de automation | ✅ Seleccionada |
| Vínculo servicio↔cuenta derivado de la webhook_url (contiene el uuid del servicio) | Sin columna nueva | Requiere listar cuentas de automation y matchear URLs; frágil y lento | ❌ Rechazada |
| Columna `automation_account_id` en `services`, configurable en el formulario | Explícito, simple, rápido de resolver | Config manual por servicio | ✅ Seleccionada |
| Sin exponer IDs en listas | Menos cambios de UI | El comando del bot requiere conocer el ID | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El endpoint de sync de p40la-ihost llama a un endpoint de run dedicado en automation, autenticado por token.
- **Contexto**: `RunJob` de automation ya implementa el flujo exacto (plugin → webhook), pero su única vía HTTP (`webhook:test`) está detrás de `AuthMiddleware` (sesión). Para server-to-server sin guardar credenciales de usuario en p40la-ihost, automation expone un endpoint de run con auth por token.
- **Decisión**: En `p40la-ihost-automation` se agrega `POST /api/accounts/{accountId}/webhook:run` (registrado fuera de `AuthMiddleware`) que valida el header `X-Webhook-Key` contra la api_key de webhooks del propio automation y ejecuta `RunJob`. `AutomationClient` en p40la-ihost guarda `automation_base_url` + `automation_api_key` en `system_settings` y llama a ese endpoint.
- **Consecuencias**: Positivo: auth stateless, sin login ni caché de cookies, sin credenciales de usuario en p40la-ihost. Negativo: requiere un cambio pequeño en el repo de automation (trackeado con su propia spec).

**ADR-002**: Vínculo servicio ↔ cuenta de automation por columna explícita.
- **Contexto**: No existe vínculo en el esquema; el único lazo implícito (webhook_url con el uuid) no es consultable de forma robusta.
- **Decisión**: Columna `services.automation_account_id` (nullable) + campo en `ServiceFormPage`. Si no está seteada, el sync responde `400` con mensaje que indica configurar el vínculo.
- **Consecuencias**: Config manual por servicio, pero determinística y barata de resolver.

**ADR-003**: Token de automation en `system_settings`.
- **Contexto**: El patrón de configuración sensible del proyecto es KV en `system_settings` (SMTP password, VoiceMonkey key, api_key de webhooks).
- **Decisión**: Claves `automation_base_url` y `automation_api_key` en `system_settings`, editables en Settings → Automation. La api_key coincide con la api_key de webhooks de automation. Nunca se loguea.
- **Consecuencias**: Simple y consistente; texto plano aceptable para uso personal interno (documentado).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[UI ServicesPage / Bot Telegram / curl]
        |  POST /api/services/{id}/sync  (sesión)
        v
[Server Go p40la-ihost]
   ├── ServiceHandlers.SyncService
   ├── AutomationClient (services/automation_client.go)
   │     └── POST {base}/api/accounts/{automation_account_id}/webhook:run
   │           Header: X-Webhook-Key: <automation_api_key>   (token stateless)
   │           → [Server Go p40la-ihost-automation]
   │             └── WebhookRunHandler (nuevo, auth por token)
   │                   └── WebhookService.RunJob
   │                         ├── plugin.FetchBills (llamado externo al proveedor)
   │                         └── POST webhook_url con X-Webhook-Key
   │                               → p40la-ihost: POST /webhooks/{uuid} (upsert, SPEC-069)
   │                               → bills creadas/actualizadas + last_webhook_request actualizado
   └── [SQLite: services + system_settings]
```

### 4.2 Componentes

#### 4.2.1 `internal/services/automation_client.go` (nuevo, p40la-ihost)
- **Responsabilidad**: Cliente HTTP hacia `p40la-ihost-automation`. Lee `automation_base_url` y `automation_api_key` de `system_settings` y ejecuta el job de cuenta del endpoint de run.
- **Interfaz**: `SyncAccount(ctx, accountID int64) (delivered, failed int, err error)`.
- **Dependencias**: `SystemSettingsService`, `net/http`, `encoding/json`.
- **Ubicación**: `internal/services/automation_client.go`.

#### 4.2.1a `internal/api/webhook_run_handlers.go` (nuevo, p40la-ihost-automation)
- **Responsabilidad**: Handler `POST /api/accounts/{accountId}/webhook:run`. Valida el header `X-Webhook-Key` contra la api_key de webhooks de automation y ejecuta `WebhookService.RunJob`. Reutiliza el mapeo de errores existente (`mapWebhookError`).
- **Interfaz**: `POST {base}/api/accounts/{accountId}/webhook:run` → `200 {"delivered": n, "failed": m}` o error JSON.
- **Dependencias**: `WebhookService`, `AppSettingsService` (api_key), `crypto/subtle`.
- **Ubicación**: `p40la-ihost-automation/internal/api/webhook_handlers.go` (extender) + ruta en su `routes.go`.

#### 4.2.2 `internal/services/system_settings.go` (extender, p40la-ihost)
- **Responsabilidad**: `Get/SetAutomationConfig(ctx)` para `automation_base_url` (default `http://ihost.local:8089`) y `automation_api_key`. La api_key se devuelve enmascarada o con flag `automation_api_key_set` (nunca en claro en el GET).
- **Dependencias**: `system_settings`.

#### 4.2.3 `internal/api/service_handlers.go` (extender)
- **Responsabilidad**: Handler `POST /api/services/{id}/sync`: valida servicio, lee `automation_account_id`, llama a `AutomationClient.SyncAccount`, mapea errores a códigos HTTP.
- **Dependencias**: `AutomationClient`, `ServiceStorage`.

#### 4.2.4 `internal/api/routes.go` (extender)
- **Responsabilidad**: Registrar `POST /api/services/{id}/sync` bajo `authMiddleware`.

#### 4.2.5 `internal/storage/service.go` (extender)
- **Responsabilidad**: Incluir `automation_account_id` en el SELECT/INSERT/UPDATE y exponerlo en `models.Service`.

#### 4.2.6 `internal/models/service.go` (extender)
- **Responsabilidad**: Campo `AutomationAccountID *int64 \`json:"automation_account_id,omitempty"\``.

#### 4.2.7 Migración `0034_add_services_automation_account_id.up.sql` / `.down.sql`
```sql
-- up
ALTER TABLE services ADD COLUMN automation_account_id INTEGER;

-- down
ALTER TABLE services DROP COLUMN automation_account_id;
```

#### 4.2.8 `internal/services/telegram_bot.go` (extender)
- **Responsabilidad**: Handler `handleSincronizarServicio`: parsea `<id>` del texto, valida, llama al sync y responde el resultado. Registro en `setMyCommands`, ayuda en `/start` y fallback.

#### 4.2.9 `frontend/src/pages/ServicesPage.tsx` (modificar)
- **Responsabilidad**: Item **"Sincronizar"** (icono `refresh`/`sync`) en el `CardMenu` de cada card con estado de carga por card y toast con el resultado/error. Badge `ID: N` en la card.
- **Dependencias**: `api.services.sync`, i18n, store de toasts.

#### 4.2.10 `frontend/src/pages/DeudasPage.tsx` (modificar)
- **Responsabilidad**: Badge `ID: N` en cada card de deuda (solo identificación).

#### 4.2.11 `frontend/src/pages/ServiceFormPage.tsx` (modificar)
- **Responsabilidad**: Campo numérico **"ID de cuenta en Automation"** (`automation_account_id`) opcional. Con tokens del tema y verificado en darkmode.

#### 4.2.12 `frontend/src/pages/SettingsPage.tsx` (o subpágina `SettingsAutomationPage.tsx`) (nuevo)
- **Responsabilidad**: Sección **"Automation"**: base URL y api_key (con estado "configurada"), persistencia vía `PUT /api/system-settings`.

#### 4.2.13 `frontend/src/api/index.ts` y `frontend/public/i18n/{es,en}.json` (modificar)
- **Responsabilidad**: `api.services.sync(id)`, endpoints de system-settings de automation y claves i18n (`services.sync*`, `services.id_badge`, `deudas.id_badge`, `settings.automation.*`). **Editar SIEMPRE en `frontend/public/i18n/`** (regla AGENTS.md).

### 4.3 Modelo de datos

```
Entidad: services
- automation_account_id: INTEGER NULL (ID de la cuenta en p40la-ihost-automation)

system_settings (KV, p40la-ihost):
- automation_base_url: TEXT  (default "http://ihost.local:8089")
- automation_api_key:  TEXT  (api_key de webhooks de p40la-ihost-automation)
```

### 4.4 APIs / Contratos

#### Endpoint: `POST /api/services/{id}/sync` (autenticado por sesión)

**Request**: sin body.

**Response 200**:
```json
{ "delivered": 3, "failed": 0 }
```

**Response Error**:
```json
{ "error": "automation_not_configured", "message": "Configurá Automation en Settings antes de sincronizar" }
```

| Código | Caso |
|--------|------|
| 200 | Sync completado (`delivered`/`failed`) |
| 400 | Servicio sin `automation_account_id`; automation sin configurar (base URL/api_key) |
| 401 | api_key de automation inválida (rechazada por el endpoint de run) |
| 404 | Servicio inexistente o eliminado |
| 502 | Automation inalcanzable o error de red |
| 502/400 | Error del job de automation (webhooks deshabilitados, sin api_key, sin plugin, sin webhook_url) — mensaje propagado |

#### Endpoint (p40la-ihost-automation): `POST /api/accounts/{accountId}/webhook:run` (token `X-Webhook-Key`)

**Request headers**:
```
X-Webhook-Key: <api_key de webhooks de automation>
```

**Response 200**:
```json
{ "delivered": 3, "failed": 0 }
```

| Código | Caso |
|--------|------|
| 200 | Job ejecutado (plugin + webhook) |
| 401 | Header `X-Webhook-Key` faltante o incorrecto |
| 400 | Errores de dominio (webhooks deshabilitados, sin api_key, sin plugin, sin webhook_url, cuenta inexistente) |
| 500 | Error interno |

#### Comando de bot: `/sincronizar_servicio <id>`

**Respuesta éxito**:
```
✅ Sincronización de *<nombre del servicio>*
  Entregadas: 3
  Fallidas: 0
```
**Respuestas de error**: mensajes claros para servicio inexistente, sin vínculo de automation, automation no configurada o inalcanzable, o ayuda cuando falta el argumento/ID inválido.

### 4.5 Dependencias

- **Internas (p40la-ihost)**: `routes.go`, `service_handlers.go`, `service.go`/`service_storage.go`, `system_settings.go`, `telegram_bot.go`, `ServicesPage.tsx`, `DeudasPage.tsx`, `ServiceFormPage.tsx`, `SettingsPage.tsx`, `api/index.ts`, `frontend/public/i18n/{es,en}.json`, migración `0034`.
- **Internas (p40la-ihost-automation)**: `webhook_handlers.go` (handler `webhook:run` + auth por token), `routes.go` (registro fuera de `AuthMiddleware`), reutiliza `WebhookService.RunJob` y `mapWebhookError`. Trackeado con una spec propia en ese repo.
- **Externas**: Ninguna nueva. Solo stdlib Go (`net/http`, `encoding/json`, `time`, `crypto/subtle`).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un servicio con `automation_account_id` y automation configurado (base URL + api_key válidas), cuando se llama `POST /api/services/{id}/sync`, entonces automation ejecuta el plugin (llamado externo) y el webhook del servicio, y responde `200` con `{ delivered, failed }`; las facturas del servicio se crean/actualizan y `last_webhook_request` se actualiza.
- [ ] CA-001a: Dado el repo de automation, cuando se invoca `POST /api/accounts/{id}/webhook:run` con `X-Webhook-Key` correcto, entonces ejecuta `RunJob` y responde `200` con `{ delivered, failed }`; sin header o con key incorrecta responde `401` y no ejecuta nada.
- [ ] CA-002: Dado un servicio sin `automation_account_id`, cuando se llama el sync, entonces responde `400` con mensaje indicando configurar el vínculo.
- [ ] CA-003: Dado automation sin configurar (base URL/api_key inválidas), cuando se llama el sync, entonces responde `400`/`401` con mensaje claro y NO se crashea.
- [ ] CA-004: Dado automation inalcanzable, cuando se llama el sync, entonces responde `502` con mensaje de error de red.
- [ ] CA-005: Dado un ID de servicio inexistente, cuando se llama el sync, entonces responde `404`.
- [ ] CA-006: Dada la lista de servicios, cuando se elige "Sincronizar" en el menú de 3 puntos de una card, entonces se ejecuta el sync con estado de carga y se muestra un toast con el resultado (entregadas/fallidas) o el error.
- [ ] CA-007: Dado el bot de Telegram, cuando se envía `/sincronizar_servicio 1`, entonces responde con el resultado del sync del servicio 1; con `/sincronizar_servicio` (sin ID) o ID no numérico responde mensaje de ayuda.
- [ ] CA-008: Dada la lista de servicios, cuando se renderiza, entonces cada card muestra un badge discreto `ID: N`.
- [ ] CA-009: Dada la lista de deudas, cuando se renderiza, entonces cada card muestra un badge discreto `ID: N`.
- [ ] CA-010: Dado Settings → Automation, cuando se guardan base URL/api_key, entonces persisten en `system_settings` y el sync las usa; la api_key no se devuelve en claro en el GET.
- [ ] CA-011: Dado `ServiceFormPage`, cuando se edita un servicio, entonces el campo `automation_account_id` se guarda y persiste.
- [ ] CA-DARK: Los controles nuevos (campo `automation_account_id`, sección Automation en Settings) usan tokens del tema (`bg-card`, `text-text`, `text-text-secondary`) y se verificó legibilidad en darkmode (texto y placeholders).
- [ ] CA-BACK: No aplica páginas de detalle nuevas con lista padre (la acción vive en el menú de 3 puntos y en el bot).

### 5.2 No funcionales

- [ ] CA-NF-001: El sync responde en < 30s (timeout del client); sin impacto en el resto de rutas mientras corre.
- [ ] CA-NF-002: Sin dependencias externas nuevas; `go build` y `go test ./...` pasan en ambos repos.
- [ ] CA-NF-003: El build del frontend (`npm run build` en `frontend/`) pasa y las claves i18n nuevas se sirven en `/i18n/es.json` y `/i18n/en.json` (editadas en `frontend/public/i18n/`).
- [ ] CA-NF-004: La api_key de automation jamás se loguea.

### 5.3 Testing

- **Unit tests**:
  - `AutomationClient` (p40la-ihost): build de request con header `X-Webhook-Key`, mapeo de `{ delivered, failed }`, errores de red/HTTP.
  - Handler `webhook:run` (automation): auth por token (401 sin header/key incorrecta), 200 con resultado, propagación de errores de `RunJob`.
  - Parse del argumento del bot (`/sincronizar_servicio 1`, sin ID, ID inválido).
  - `SystemSettingsService.Get/SetAutomationConfig` (defaults, api_key enmascarada).
  - Migración `0034` (columna nullable, up/down).
- **Integration tests**: `POST /api/services/{id}/sync` contra un `httptest.Server` que simula el endpoint `webhook:run` de automation, cubriendo 200/400/401/404/502. En automation, `webhook:run` con token real contra `httptest`.
- **E2E tests**: Prueba manual local: configurar Automation en Settings → crear/editar un servicio con `automation_account_id` → "Sincronizar" desde la card → verificar toast con entregadas/fallidas y facturas en `BillsPage`; probar `/sincronizar_servicio 1` por Telegram; verificar badges `ID: N` en servicios y deudas; darkmode.
- **Carga/Performance**: Sin impacto significativo; verificar con `./scripts/check-server.sh` tras levantar en local.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 0 | **Automation**: endpoint `POST /api/accounts/{id}/webhook:run` (auth por token) + tests (spec propia en ese repo) | 0.5 día | Ninguna |
| 1 | p40la-ihost: migración `0034` (`automation_account_id`) + modelo + storage | 0.5 día | Ninguna |
| 2 | p40la-ihost: `AutomationClient` + config `automation_base_url`/`automation_api_key` en `system_settings` | 1 día | Fase 0, 1 |
| 3 | p40la-ihost: handler `POST /api/services/{id}/sync` + rutas + mapeo de errores | 0.5 día | Fase 2 |
| 4 | p40la-ihost: comando de bot `/sincronizar_servicio <id>` + `setMyCommands` + ayuda | 0.5 día | Fase 3 |
| 5 | p40la-ihost: frontend (badge `ID: N` en servicios y deudas, item "Sincronizar", campo `automation_account_id`, Settings → Automation, api + i18n + build) | 1 día | Fases 2-3 |
| 6 | Tests (unit + integration) + validación local end-to-end (ambos repos) + darkmode | 1 día | Fases 0-5 |

### 6.2 Milestones

1. **MVP**: Fases 0-3 — endpoint de run en automation + sync por API en p40la-ihost (curl/endpoint) funcional.
2. **V1.0**: Fases 4-6 — bot + UI completa + badges de ID + tests end-to-end.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| api_key de automation en texto plano | Media | Medio | Aceptable para uso personal interno (mismo patrón que SMTP/voicemonkey). Documentar y no loguear. |
| Cambio cruzado entre repos (automation y p40la-ihost) | Media | Medio | El endpoint de run se trackea con spec propia en automation; contrato documentado y simulado con `httptest`. |
| Sync lento por el llamado externo del proveedor | Media | Medio | Timeout 30s; la UI muestra loading; el bot responde al final. |
| Vínculo service↔cuenta mal configurado | Media | Bajo | Error `400` claro que indica cómo configurarlo. |
| Rotura por cambio futuro en la API de automation | Baja | Medio | Documentar el contrato en esta spec; los tests con `httptest` simulan el contrato. |

## 8. Notas y Referencias

- Contrato de automation (nuevo): `POST /api/accounts/{accountId}/webhook:run` → `WebhookService.RunJob` (`p40la-ihost-automation/internal/services/webhook.go`). Base de partida: `webhook:test` (sesión) en `p40la-ihost-automation/internal/api/webhook_handlers.go`.
- Webhook de p40la-ihost: SPEC-069 (`POST /webhooks/{uuid}`, api_key global) y SPEC-074 (`last_webhook_request`).
- Bot: SPEC-079 (bot embebido), SPEC-080 (match de comandos sin slash), SPEC-081/083/084 (formatos).
- Patrón de configuración sensible en `system_settings`: SPEC-069 (webhooks), SPEC-032/033 (alerts), SPEC-089 (telegram bot).
- Migración de p40la-ihost anterior: `0033`; nueva: `0034`.
- El cambio en `p40la-ihost-automation` se trackea con una spec propia en ese repo (sistema de specs del repo de automation).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-24 | opencode | Creación inicial de la especificación |
| 2026-09-24 | opencode | Alcance ampliado a **dos repos** (confirmado por el usuario): automation agrega `POST /api/accounts/{id}/webhook:run` autenticado por token (`X-Webhook-Key` = api_key de webhooks de automation); p40la-ihost usa `AutomationClient` stateless (base_url + api_key en `system_settings`), sin login ni cookies de sesión. Se actualizaron REQs, ADRs, diseño, contratos, criterios y plan. |