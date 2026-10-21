---
title: "Bot Telegram: listado de servicios con plugin de sincronización y botones de sincronizar"
id: "SPEC-103"
status: "released"
author: "opencode"
created: "2026-10-21"
updated: "2026-10-21"
github_issue: 106
---

# Bot Telegram: listado de servicios con plugin de sincronización y botones de sincronizar

**ID**: SPEC-103  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-10-21  
**Actualizado**: 2026-10-21

---

## 1. Resumen Ejecutivo

La SPEC-092 agregó la sincronización manual de un servicio con los plugins de `p40la-ihost-automation`, pero en Telegram solo existe `/sincronizar_servicio <id>`. Para usarlo hay que recordar o buscar el ID numérico del servicio (badge `ID: N` en la app), lo que hace incómodo sincronizar varios servicios seguidos: se debe escribir un comando por cada uno.

Esta spec agrega un comando nuevo, `/sincronizar`, que lista **solo los servicios que tienen configurada una cuenta de automation** (`services.automation_account_id` no nulo) y envía **un mensaje por servicio** con un botón inline **“🔄 Sincronizar”**. Al tocar el botón, Telegram envía un `callback_query` al bot, que ejecuta exactamente el mismo flujo ya probado de SPEC-092 (`AutomationClient.SyncAccount` → `p40la-ihost-automation` → plugin → webhook) y actualiza ese mismo mensaje con el resultado, dejando el botón disponible para volver a sincronizar. Como cada mensaje tiene su propio botón, el usuario puede sincronizar uno o varios servicios tocando los botones que quiera, sin escribir IDs.

El cambio es **solo backend/bot**: no requiere migraciones, no toca la UI web y no agrega dependencias (se reutiliza `github.com/go-telegram/bot` v1.27.0, ya usado por SPEC-097 para botones inline). El consumo en el iHost es despreciable: una consulta `SELECT` a `services`, un mapa en memoria acotado por servicio para evitar dobles ejecuciones y los mismos timeouts/HTTP client de automation que ya existen.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Nuevo comando `/sincronizar` (sin argumentos) que responde con la lista de servicios sincronizables. Se considera sincronizable todo servicio con `deleted_at IS NULL` y `automation_account_id IS NOT NULL`. El comando no reemplaza a `/sincronizar_servicio <id>` (se mantiene por compatibilidad).
2. **REQ-002**: Por cada servicio sincronizable, enviar **un mensaje independiente** con: nombre del servicio, ID de cuenta de automation, estado (activo/inactivo) y estado de la última factura (o “sin facturas”). Cada mensaje incluye un teclado inline con un botón **“🔄 Sincronizar”** cuyo `callback_data` es `sync:<service_id>`.
3. **REQ-003**: Si no hay servicios con cuenta de automation, responder un mensaje explicativo indicando cómo vincular una cuenta (editar el servicio en la app → “ID de cuenta en Automation”).
4. **REQ-004**: Si automation no está configurado (`automation_base_url`/`automation_api_key` vacíos), el comando responde con un aviso que indica configurarlo en P40LA → Configuración → Automation y **no envía botones**.
5. **REQ-005**: Handler de `callback_query` con prefijo `sync:`: valida que el chat esté autorizado (misma allowlist `chat_ids` del bot), parsea el ID, verifica que el servicio exista, siga sin eliminarse y conserve `automation_account_id`, y ejecuta `AutomationClient.SyncAccount` con el `automation_account_id`.
6. **REQ-006**: El `callback_query` se responde **de inmediato** (`AnswerCallbackQuery`, texto “Sincronizando…”) para cortar el spinner antes de que Telegram lo marque como expirado; la sincronización puede tardar hasta 30 s (timeout del cliente de automation).
7. **REQ-007**: Al finalizar, editar el mensaje del botón (`EditMessageText`) para mostrar el resultado (`✅ Entregadas: N · Fallidas: M` y hora) o el error (`⚠️ No se pudo sincronizar: …`). El teclado con el botón **se conserva** para permitir reintentos y sincronizaciones sucesivas.
8. **REQ-008**: Si el mensaje original ya no es editable (mensaje inaccesible o demasiado viejo), enviar un mensaje nuevo al chat con el resultado, sin romper el flujo.
9. **REQ-009**: Evitar ejecuciones concurrentes duplicadas del **mismo** servicio mediante un registro en memoria de sincronizaciones en curso (`map[int64]bool` protegido por mutex). Si llega un segundo callback para un servicio ya en curso, se responde “Ya hay una sincronización en curso para este servicio” sin lanzar otra ejecución. Servicios distintos sí pueden sincronizarse en paralelo.
10. **REQ-010**: Registrar `/sincronizar` en `setMyCommands` (SPEC-080), en el texto de `/start` y en el mensaje de comando no reconocido.
11. **REQ-011**: El listado debe enviarse de forma secuencial y tolerar los límites de Telegram: si `sendMessage` devuelve error de rate limit (429), se loguea y se continúa; el comando nunca debe abortar el resto de la lista por un mensaje fallido.
12. **REQ-012**: Los logs del handler no deben incluir `automation_api_key` ni datos sensibles; solo `service_id`, `automation_account_id`, resultado y error.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: Mostrar en el mensaje inicial la fecha/hora del último webhook recibido (`last_webhook_request`, SPEC-074) cuando exista, como referencia de la última sincronización efectiva.
2. **REQ-014**: Aceptar `/sincronizar <id>` como atajo opcional: si se pasa un ID, ejecuta el sync de ese servicio (equivalente a `/sincronizar_servicio <id>`) sin listar. Útil para automatizaciones o comandos guardados.
3. **REQ-015**: Incluir un mensaje final de resumen con el total de servicios listados y la aclaración de que cada botón sincroniza solo su servicio.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-016**: Botón **“🔄 Sincronizar todos”** en el mensaje de resumen, que ejecuta los servicios **secuencialmente** (nunca en paralelo) y edita el resumen con el progreso. Queda fuera del MVP por el riesgo de timeouts y rate limits.
2. **REQ-017**: Filtro `/sincronizar activos` para listar únicamente servicios activos.
3. **REQ-018**: Incluir el nombre del hogar en cada mensaje (requiere inyectar `HomeStorage` en `TelegramBotService`; no se hace en el MVP para no ampliar dependencias del bot).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: el comando hace una sola consulta `List` a SQLite y N envíos de mensajes. El callback reutiliza el timeout de 30 s de `AutomationClient`. Sin CPU/RAM relevante para el iHost.
- **Seguridad**: mismos controles que el resto del bot: allowlist de `chat_ids`; el `callback_data` solo contiene el ID numérico del servicio. No se confía en datos enviados por el cliente más allá del ID; el vínculo con automation se relee de la DB. La `api_key` de automation nunca se loguea ni se envía a Telegram.
- **Almacenamiento**: sin tablas ni columnas nuevas. Solo un mapa en memoria con, como máximo, una entrada por servicio sincronizándose.
- **Disponibilidad**: si automation está caído, cada callback falla de forma aislada y el bot sigue operativo. El listado y los demás comandos no se ven afectados.
- **Límites de Telegram**: `callback_data` ≤ 64 bytes (`sync:<id>` cumple); texto de mensaje ≤ 4096 caracteres; envío secuencial para acercarse al límite de ~1 mensaje/s por chat.
- **iHost**: sin dependencias nuevas; se reutilizan `go-telegram/bot` y `AutomationClient` ya presentes.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **SPEC-092 (implementada)**: `POST /api/services/{id}/sync` + `AutomationClient.SyncAccount(ctx, automationAccountID)` es el flujo oficial de sincronización. El vínculo plugin ↔ servicio es la columna nullable `services.automation_account_id` (migración `0034`). El comando `/sincronizar_servicio <id>` ya usa ese mismo camino y devuelve `delivered`/`failed` (`internal/services/telegram_bot.go`).
- **SPEC-097 (implementada)**: antecedente directo de botones inline en este bot. Usa `tgmodels.InlineKeyboardMarkup` con `CallbackData` y registra `bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "bt:", bot.MatchTypePrefix, s.handleBudgetCallback)`. También usa `AnswerCallbackQuery` y `EditMessageReplyMarkup`, por lo que la librería y el patrón ya están validados en el proyecto.
- **Librería `github.com/go-telegram/bot` v1.27.0**: soporta `InlineKeyboardMarkup`, `AnswerCallbackQuery` y `EditMessageText` (con `ReplyMarkup`). El matcher `MatchTypePrefix` permite registrar un handler por prefijo, por lo que `sync:` no colisiona con `bt:`.
- **`ServiceStorage.List(ctx, nil)`**: ya devuelve todos los servicios no eliminados con `automation_account_id`, `last_webhook_request` y `latest_bill_status`, sin queries nuevas.
- **Patrón del bot (SPEC-082)**: los listados largos se envían como **un mensaje por ítem** para respetar el límite de 4096 caracteres. Este requerimiento sigue ese patrón.
- **Telegram Bot API**: `callback_query` debe responderse con `answerCallbackQuery`; si el bot tarda, el cliente muestra error de timeout. Por eso se responde antes de sincronizar y el resultado se refleja editando el mensaje. `callback_data` tiene un límite de 64 bytes. Límite de envío aproximado: 30 mensajes/s global y ~1 mensaje/s por chat.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Un solo mensaje con un botón por servicio | Menos mensajes; vista compacta | Muchos servicios generan teclados enormes; el mensaje crece y se vuelve ilegible; difícil mostrar el resultado por servicio sin editar todo el teclado | ❌ Rechazada |
| Un mensaje por servicio con botón inline (patrón SPEC-082) | Resultado y estado por servicio en su propio mensaje; botón reutilizable; escala a cualquier cantidad; coincide con la UX actual del bot | Más mensajes (rate limit) | ✅ Seleccionada |
| Comando interactivo con selección múltiple (checkboxes + “confirmar”) | Sincroniza varios de una vez | Requiere estado de sesión en memoria, teclados que se reescriben y una UX más compleja; Telegram no tiene checkboxes nativos (hay que emular estado) | ❌ Rechazada para el MVP (queda como REQ-016/P2) |
| Reutilizar `/sincronizar_servicio <id>` y solo mejorar la ayuda | Cero código nuevo | Sigue obligando a conocer/tipear IDs; no cumple el pedido | ❌ Rechazada |
| Listar desde `p40la-ihost-automation` (consultar cuentas/plugins) | Refleja exactamente qué cuentas tienen plugin | Requiere un endpoint nuevo en automation, credenciales y acoplar el bot a otro servicio; el vínculo ya está en `services.automation_account_id` | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El criterio de “servicio con plugin de sincronización” es `automation_account_id IS NOT NULL`.
- **Contexto**: `p40la-ihost` no conoce los plugins de automation; solo conoce la cuenta vinculada. Consultar automation para saber si la cuenta tiene plugin agregaría un endpoint, latencia y un punto de fallo.
- **Decisión**: listar los servicios con `automation_account_id` configurado. Si la cuenta no tiene plugin, el error se propaga desde automation al presionar el botón, con mensaje legible (comportamiento ya existente de SPEC-092).
- **Consecuencias**: listado rápido y offline; un servicio mal configurado aparece y falla al sincronizar con un error claro.

**ADR-002**: Un mensaje por servicio con botón inline `sync:<id>`.
- **Contexto**: se necesita sincronizar uno o varios servicios y ver el resultado de cada uno.
- **Decisión**: replicar el patrón de listados de SPEC-082: un mensaje por servicio, con teclado de una fila y un botón. El `callback_data` lleva solo el ID.
- **Consecuencias**: UX simple y escalable; a cambio se envían N mensajes y hay que cuidar el rate limit (REQ-011).

**ADR-003**: Responder el callback antes de sincronizar y editar el mensaje al terminar.
- **Contexto**: Telegram invalida el spinner del botón si `answerCallbackQuery` tarda; el sync puede tardar hasta 30 s.
- **Decisión**: `AnswerCallbackQuery` inmediato con “Sincronizando…”, luego `EditMessageText` con el resultado conservando el teclado. Si el edit falla, se envía un mensaje nuevo.
- **Consecuencias**: feedback inmediato y resultado visible; requiere manejar el caso de mensaje inaccesible (REQ-008).

**ADR-004**: Registro en memoria de sincronizaciones en curso por `service_id`.
- **Contexto**: un usuario puede tocar el botón dos veces o dos chats autorizados pueden sincronizar el mismo servicio a la vez, generando llamadas duplicadas a automation.
- **Decisión**: `map[int64]struct{}` + `sync.Mutex` en `TelegramBotService`. La entrada se elimina con `defer`. No se bloquea el sync de servicios distintos.
- **Consecuencias**: se evitan duplicados sin serializar todo el bot; el estado se pierde al reiniciar el proceso, lo cual es aceptable.

**ADR-005**: Sin cambios en la UI web ni en la API HTTP.
- **Contexto**: el pedido es un comando de Telegram; la UI ya tiene la acción “Sincronizar” por servicio (SPEC-092).
- **Decisión**: implementar todo en `internal/services` del bot, reutilizando `ServiceStorage` y `AutomationClient`.
- **Consecuencias**: cambio acotado y de bajo riesgo; no hay migraciones ni rebuild de frontend.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
Usuario ── /sincronizar ──▶ Bot Telegram (p40la-ihost)
                                │
                                ├─ ServiceStorage.List()  → SQLite (services)
                                │     filtro: automation_account_id != NULL
                                │
                                └─ por servicio: SendMessage(texto, botón "🔄 Sincronizar")
                                       callback_data = "sync:<service_id>"

Usuario ── toca botón ──▶ callback_query "sync:<id>"
                                │
                                ├─ AnswerCallbackQuery("Sincronizando…")   (inmediato)
                                ├─ valida chat autorizado + servicio + account_id
                                ├─ guard anti-duplicado por service_id
                                ├─ AutomationClient.SyncAccount(account_id)
                                │     └─ POST {automation}/api/accounts/{id}/webhook:run
                                │           └─ plugin + POST webhook del servicio
                                └─ EditMessageText(resultado, mismo botón)
```

### 4.2 Componentes

#### 4.2.1 `internal/services/telegram_sync.go` (nuevo)
- **Responsabilidad**: handlers y helpers del listado y de los callbacks de sincronización.
- **Interfaz**:
  - `handleSincronizarServicios(ctx, b, update)` — comando `/sincronizar`.
  - `handleSyncCallback(ctx, b, update)` — callbacks `sync:<id>`.
  - `syncableServices(ctx) ([]models.Service, error)` — `List` + filtro `AutomationAccountID != nil`.
  - `formatSyncServiceMessage(svc models.Service) string`.
  - `syncButton(serviceID int64) *tgmodels.InlineKeyboardMarkup`.
  - `editSyncMessage(...)` — `EditMessageText` con fallback a `SendMessage`.
- **Dependencias**: `ServiceStorage`, `SystemSettingsService`, `AutomationClient`, `go-telegram/bot`.
- **Ubicación**: `internal/services/telegram_sync.go`.

#### 4.2.2 `internal/services/telegram_bot.go` (modificar)
- **Responsabilidad**:
  - Registrar en `runBot`: `b.RegisterHandler(bot.HandlerTypeMessageText, "sincronizar", bot.MatchTypeCommand, s.handleSincronizarServicios)` y `b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "sync:", bot.MatchTypePrefix, s.handleSyncCallback)`.
  - Agregar `/sincronizar` a `registerCommands`, a `handleStart` y al texto de `handleDefault`.
  - Agregar al struct `TelegramBotService`: `syncMu sync.Mutex` y `syncInFlight map[int64]struct{}` (inicializado en `NewTelegramBotService`).
- **Nota**: el patrón se registra **sin slash** (SPEC-080) y `sincronizar` no colisiona con `sincronizar_servicio` porque `MatchTypeCommand` compara el comando completo.

#### 4.2.3 Tests (`internal/services/telegram_sync_test.go` nuevo o `telegram_bot_test.go` extendido)
- **Responsabilidad**: cubrir listado con/sin servicios, configuración de automation ausente, callback exitoso, callback duplicado, servicio inexistente/sin cuenta y fallo de automation.
- **Infra**: extender `fakeTelegramAPI` para aceptar `answerCallbackQuery` y `editMessageText` y capturar el `text` editado y el `reply_markup` enviado.

#### 4.2.4 Sin cambios
- No hay cambios en `internal/api`, `internal/storage`, `internal/models`, migraciones ni frontend.

### 4.3 Modelo de datos

No hay cambios de esquema. Se reutiliza:

```
services
- id: INTEGER PK
- name: TEXT
- active: BOOLEAN
- automation_account_id: INTEGER NULL   -- vínculo con la cuenta/plugin de automation (SPEC-092)
- last_webhook_request: DATETIME NULL   -- última entrega efectiva del webhook (SPEC-074)
- latest_bill_status: TEXT NULL         -- calculado en el SELECT (pendiente/pagada)
```

### 4.4 APIs / Contratos

No se agregan endpoints HTTP. Los contratos son de Telegram:

#### Comando: `/sincronizar`

**Respuesta** (N mensajes, uno por servicio + 1 resumen):

```
🔌 *Internet Casa*
  Cuenta automation: 7
  Estado: activo
  Última factura: pendiente
  Último webhook: 20 oct 2026 18:04

[ 🔄 Sincronizar ]
```

**Sin servicios**:
```
No hay servicios con una cuenta de Automation vinculada.
Editá un servicio en la app → "ID de cuenta en Automation".
```

**Automation no configurado**:
```
⚠️ Automation no está configurada.
Configurala en P40LA → Configuración → Automation.
```

#### Callback: `sync:<service_id>`

- Éxito → edita el mensaje:
```
✅ *Internet Casa* sincronizado
  Entregadas: 2
  Fallidas: 0
  09:41
[ 🔄 Sincronizar ]
```
- Error → edita el mensaje:
```
⚠️ *Internet Casa*
  No se pudo sincronizar: automation respondió 502: ...
[ 🔄 Sincronizar ]
```
- Servicio ya en curso → `AnswerCallbackQuery("Ya hay una sincronización en curso…")`.

### 4.5 Dependencias

- **Internas**: `ServiceStorage`, `SystemSettingsService`, `AutomationClient`, `TelegramBotService` (existente).
- **Externas**: ninguna nueva. Se usa `github.com/go-telegram/bot` v1.27.0 y `net/http` ya presentes.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado un bot configurado y al menos un servicio con `automation_account_id`, cuando el usuario envía `/sincronizar`, entonces recibe un mensaje por cada servicio sincronizable (y ninguno por los que no tienen cuenta) con un botón “🔄 Sincronizar”.
- [ ] CA-002: Dado el listado anterior, cuando el usuario toca el botón de un servicio, entonces el bot responde el callback de inmediato, ejecuta `SyncAccount` con el `automation_account_id` correcto y edita el mensaje con `Entregadas`/`Fallidas`.
- [ ] CA-003: Dado el listado anterior, cuando el usuario toca los botones de **dos servicios distintos**, entonces ambos se sincronizan de forma independiente y cada mensaje muestra su propio resultado.
- [ ] CA-004: Dado un servicio sin `automation_account_id`, cuando se ejecuta `/sincronizar`, entonces ese servicio no aparece en el listado.
- [ ] CA-005: Dado que no hay servicios con cuenta de automation, cuando se ejecuta `/sincronizar`, entonces el bot responde el mensaje explicativo sin botones.
- [ ] CA-006: Dado que automation no está configurado, cuando se ejecuta `/sincronizar`, entonces el bot avisa cómo configurarlo y no envía botones.
- [ ] CA-007: Dado un `callback_query` con un ID inexistente o de un servicio eliminado, entonces el bot responde un aviso claro y no ejecuta el sync.
- [ ] CA-008: Dado un segundo callback para un servicio que ya se está sincronizando, entonces el bot responde “Ya hay una sincronización en curso” y no lanza una segunda llamada a automation.
- [ ] CA-009: Dado un fallo de automation, entonces el mensaje del servicio muestra el error y conserva el botón para reintentar; el resto de los mensajes y comandos sigue funcionando.
- [ ] CA-010: Dado un chat no autorizado, entonces los comandos y callbacks se ignoran en silencio (misma allowlist de `chat_ids`).
- [ ] CA-011: `/sincronizar` aparece en el menú de comandos de Telegram (`setMyCommands`), en `/start` y en la ayuda del handler default.
- [ ] CA-012: `/sincronizar_servicio <id>` sigue funcionando exactamente igual que antes.
- [ ] CA-DARK: No aplica — la spec no toca formularios ni UI web.
- [ ] CA-BACK: No aplica — la spec no crea páginas de detalle ni formularios con lista padre.

### 5.2 No funcionales

- [ ] CA-NF-001: `callback_data` ≤ 64 bytes en todos los botones.
- [ ] CA-NF-002: El listado no falla completo si un `sendMessage` individual falla; los errores se loguean y se continúa.
- [ ] CA-NF-003: Los logs no contienen `automation_api_key`.
- [ ] CA-NF-004: `go test ./...` pasa y `gofmt` no reporta cambios.
- [ ] CA-NF-005: No se agregan dependencias ni migraciones; el binario y el consumo de memoria del iHost no cambian de forma perceptible.

### 5.3 Testing

- **Unit tests**: filtro de servicios sincronizables, formato de mensaje, parseo de `sync:<id>`, guard anti-duplicado.
- **Integration tests**: con `fakeTelegramAPI` extendido, ejecutar `/sincronizar` y verificar N mensajes + `reply_markup`; simular `callback_query` y verificar `answerCallbackQuery` + `editMessageText`; simular automation con `httptest` (patrón de `TestSincronizarServicioCommand`).
- **E2E/manual**: en el iHost o local con un bot real, ejecutar `/sincronizar`, tocar dos botones y verificar facturas/`last_webhook_request` actualizados en la app.
- **Carga/Performance**: verificar con ~20 servicios que el listado se envía completo y que no se dispara rate limit grave.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `telegram_sync.go`: filtro, formato, teclado y handler `/sincronizar` | 2-3 h | Ninguna |
| 2 | Handler `sync:` + guard anti-duplicado + edición de mensaje con fallback | 2-3 h | Fase 1 |
| 3 | Registro de comando/handlers y ayudas (`setMyCommands`, `/start`, default) | 30 min | Fase 1 |
| 4 | Tests unitarios/integración + extensión de `fakeTelegramAPI` | 2-3 h | Fases 1-3 |
| 5 | Prueba manual local/iHost y ajustes de formato | 1 h | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: `/sincronizar` + un mensaje por servicio con botón + callback de sync + resultado editado + guard anti-duplicado. Incluye REQ-001 a REQ-012.
2. **V1.0**: REQ-013 a REQ-015 (último webhook, atajo con ID, mensaje resumen).
3. **P2**: REQ-016 a REQ-018 (sincronizar todos, filtro de activos, nombre de hogar).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Rate limit de Telegram al listar muchos servicios | Media | Medio | Envío secuencial, log y continuación ante error; si aparece 429 con `retry_after`, evaluar pausa corta en implementación |
| El sync tarda más que el timeout percibido por Telegram | Media | Bajo | Responder el callback antes de sincronizar (ADR-003) y editar el mensaje al finalizar |
| Mensaje original no editable (viejo/inaccesible) | Baja | Bajo | Fallback a `SendMessage` con el resultado (REQ-008) |
| Doble ejecución por doble toque | Media | Medio | Guard en memoria por `service_id` (ADR-004) |
| Nombres de servicio con caracteres Markdown rompen el mensaje | Baja | Bajo | Mantener el patrón actual del bot (MarkdownV1); si se detecta, escapar `*`, `_`, `` ` `` y `[` en el nombre |
| Cuenta de automation sin plugin o mal configurada | Media | Bajo | El error de automation se muestra en el mensaje; el resto de la lista no se ve afectado |
| Cambios en automation rompen el contrato | Baja | Alto | Se reutiliza el contrato de SPEC-092 (`webhook:run` + `X-Webhook-Key`); tests de integración con `httptest` |

## 8. Notas y Referencias

- **Fix incidental incluido en esta sesión**: `TestWebhookUpsertSoftDeletedNotFoundPropagates` fallaba en `main` desde octubre 2026 porque usaba el período fijo `2026/10`, que ya no es "nuevo": `ServiceService.Create` genera automáticamente la factura del mes en curso para servicios activos. Se ajustó el test para usar el período anterior al mes actual (`previousPeriod`), sin cambios de código productivo. No forma parte de la funcionalidad de esta spec, pero se documenta aquí por trazabilidad.
- SPEC-092: Sincronización manual de servicios con los plugins de `p40la-ihost-automation` (endpoint + UI + bot).
- SPEC-097: Bot Telegram `/budget_transaction` — precedente de botones inline y callbacks.
- SPEC-082: Un mensaje por servicio y totales al final — patrón de listados del bot.
- SPEC-080: Registro de comandos y matcheo sin slash.
- SPEC-074: Indicador `last_webhook_request` en servicios.
- `internal/services/telegram_bot.go`, `internal/services/automation_client.go`, `internal/storage/service.go`.
- Telegram Bot API — Inline keyboards y CallbackQuery: <https://core.telegram.org/bots/api#inlinekeyboardmarkup>, <https://core.telegram.org/bots/api#callbackquery>.
- `github.com/go-telegram/bot` v1.27.0 — `EditMessageText`, `AnswerCallbackQuery`.

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-10-21 | opencode | Creación inicial de la especificación (draft) |
| 2026-10-21 | opencode | Estado a `in_progress`; inicio de implementación (MVP REQ-001..REQ-012 + REQ-013/REQ-015) |
| 2026-10-21 | opencode | Fix incidental del test date-dependent `TestWebhookUpsertSoftDeletedNotFoundPropagates` (período fijo 2026/10 → período anterior al mes actual) |
| 2026-10-21 | opencode | Prueba manual local con bot real: `/sincronizar` listó 2 servicios y ambos botones sincronizaron (delivered=2, failed=0). Release a `main` en commit `9954f5d`. |
