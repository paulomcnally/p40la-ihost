---
title: "Bot Telegram: comando /budget_transaction para registrar transacciones del presupuesto por conversación guiada"
id: "SPEC-097"
status: "released"
author: "opencode"
created: "2026-09-28"
updated: "2026-09-28"
github_issue: 100
---

# Bot Telegram: comando /budget_transaction para registrar transacciones del presupuesto por conversación guiada

**ID**: SPEC-097  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-28  
**Actualizado**: 2026-09-28

---

## 1. Resumen Ejecutivo

El bot de Telegram embebido (`TelegramBotService`, SPEC-079) hoy es de **solo lectura**: consulta la DB local y responde listados (`/servicios_pendientes`, `/deudas_pendientes`) o dispara acciones puntuales con ID (`/sincronizar_servicio <id>`). No permite registrar datos nuevos. Con el módulo de presupuesto estilo YNAB en producción (SPEC-093), el usuario quiere poder **registrar una transacción del presupuesto directamente desde el chat**: gastar el supermercado sin abrir la app.

Esta spec agrega el comando `/budget_transaction` que inicia una **conversación guiada** por el bot: el usuario elige grupo, categoría, tipo (gasto/ingreso), monto, fecha, cuenta y payee mediante **botones inline** (inline keyboards) y pasos de texto, con un resumen y confirmación final antes de guardar. La transacción se crea reutilizando `BudgetTransactionService.Create` (SPEC-093), que ya valida cuenta, moneda, fecha, montos y categoría. El flujo es una **máquina de estados en memoria por `chat_id`**, sin dependencias nuevas (la librería `go-telegram/bot` ya soporta inline keyboards y callback queries).

**Consideraciones iHost**: estado de conversación en memoria (despreciable en RAM, ~100 bytes por chat activo, con TTL de limpieza). Sin escrituras a disco nuevas, sin dependencias externas, sin tablas SQL nuevas. El bot ya corre dentro del server; solo se inyectan servicios existentes ya instanciados en `main.go`.

**Alcance**: solo el canal Telegram (el usuario aclaró que "WhatsApp" fue un error de su parte). No se toca la UI web ni el i18n.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Nuevo comando `/budget_transaction` en el bot de Telegram que inicia una conversación guiada para registrar una transacción del presupuesto. Se registra en `registerCommands` (visible en el menú del bot) y en la ayuda de `/start`.
2. **REQ-002**: Validación de prerequisitos al iniciar el flujo: si no existe al menos un grupo con categorías activas o al menos una cuenta, el bot responde un aviso indicando qué falta crear (y desde dónde) y no inicia la conversación.
3. **REQ-003**: Paso **Grupo**: el bot muestra botones inline con los grupos de categorías (`CategoryGroupService.List`). Al elegir uno, avanza.
4. **REQ-004**: Paso **Categoría**: el bot muestra botones inline con las categorías activas del grupo elegido. Al elegir una, avanza.
5. **REQ-005**: Paso **Tipo**: botones inline *Gasto* / *Ingreso* (determina `outflow` vs `inflow`).
6. **REQ-006**: Paso **Monto**: el bot pide el monto como texto libre; valida que sea un número positivo mayor a cero.
7. **REQ-007**: Paso **Fecha**: el bot pide la fecha como texto `YYYY-MM-DD`; si se responde vacío o "hoy", usa la fecha actual en la zona horaria configurada (mismo criterio que SPEC-084 ADR-004).
8. **REQ-008**: Paso **Cuenta**: botones inline con las cuentas activas (`AccountService.List`). **La moneda de la transacción se deriva de la moneda de la cuenta elegida** (patrón del default del frontend, SPEC-093).
9. **REQ-009**: Paso **Payee**: texto libre opcional (se puede omitir).
10. **REQ-010**: Paso **Confirmación**: el bot muestra un resumen formateado con los montos usando el formato de moneda configurado (`formatAmount`, patrón SPEC-084) y botones *Confirmar* / *Cancelar*. Confirmar crea la transacción vía `BudgetTransactionService.Create` y responde éxito; Cancelar descarta la sesión.
11. **REQ-011**: En **cada** paso hay un botón *Cancelar* que descarta la sesión. El comando `/budget_transaction` reinicia cualquier sesión previa del mismo chat.
12. **REQ-012**: Sesiones con **timeout por inactividad** (~15 min). Un mensaje de un chat sin sesión activa sigue cayendo en `handleDefault` con la respuesta "comando no reconocido" actual (no rompe el comportamiento existente).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-013**: Manejo de errores amigables: ante un monto o fecha inválidos, el bot explica el formato esperado y **repite la misma pregunta** (no pierde el estado ni los datos ya ingresados).
2. **REQ-014**: Los callback queries responden con `AnswerCallbackQuery` (feedback táctil) y el teclado del paso actual se reemplaza por el del siguiente paso vía `EditMessageReplyMarkup` para que el chat no se llene de mensajes.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-015**: Si la categoría tiene una cuenta asociada (`categories.account_id`, SPEC-093), proponerla como cuenta preseleccionada en el paso Cuenta (primer botón "Sugerida: <cuenta>").

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: overhead despreciable (estado en memoria, una consulta de listado por paso). Sin impacto en la vista mensual ni en los schedulers.
- **Seguridad**: los comandos y callbacks respetan la allowlist de `chat_ids` existente (`isAuthorized`/`checkAuthorized`). No se exponen datos sensibles nuevos. Callback data acotada a IDs (64 bytes límite de Telegram).
- **Almacenamiento**: sin cambios de esquema. Solo memoria.
- **Disponibilidad**: sesiones en memoria se pierden al reiniciar el server (aceptado; el usuario reinicia el comando). El supervisor del bot (`run`) no cambia.
- **iHost**: sin dependencias nuevas en Go; build multi-arch intacto; consumo de RAM despreciable.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- **`go-telegram/bot` v1.27.0** (ya en `go.mod` como dependencia del bot, `internal/services/telegram_bot.go`). Verificado en el módulo cacheado:
  - `models.InlineKeyboardMarkup` / `models.InlineKeyboardButton` (`models/reply_markup.go`) → botones inline.
  - `bot.HandlerTypeCallbackQueryData` con `MatchTypePrefix` → handlers de callbacks (`handlers.go`).
  - `bot.AnswerCallbackQuery`, `bot.EditMessageReplyMarkup` (`methods.go`/`methods_params.go`).
  - `bot.WithDefaultHandler` (ya usado en `telegram_bot.go:122`) recibe **todo** mensaje no capturado → es la puerta de entrada a los pasos de texto del flujo.
- **Bot actual**: comandos registrados con `RegisterHandler(HandlerTypeMessageText, "<cmd>", MatchTypeCommand, ...)` (`telegram_bot.go:141-144`). `handleDefault` hoy responde "Comando no reconocido" a todo (`telegram_bot.go:271`). Patrón a extender: si hay sesión activa para el chat, rutear; si no, respuesta por defecto.
- **Servicios del presupuesto** (SPEC-093, ya instanciados en `main.go:134-141`): `CategoryGroupService.List` (grupos + categorías activas), `AccountService.List`, `CurrencyService.List`, `BudgetTransactionService.Create` (valida cuenta/moneda/fecha/outflow-inflow/categoría en `transaction.go:76`).
- **Formato de montos**: `formatAmount(monto, symbol, CurrencyFormat)` y `DefaultCurrencyFormat()` ya usados por el bot (SPEC-084). Zona horaria: `settings.GetTimezoneLocation(ctx)` (SPEC-084 ADR-004).
- **Prerequisito UI existente**: la app valida dependencias en backend y frontend y redirige al formulario faltante (AGENTS.md). El bot replica ese principio con mensaje de aviso.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| **Botones inline (inline keyboard + callback queries)** | UX superior, selección sin errores de tipeo, el chat no se llena | Un poco más de código (callbacks + edición de teclados) | ✅ Seleccionada (decisión del usuario) |
| Selección por números de texto | Más simple, consistente con `/sincronizar_servicio <id>` | Errores de tipeo, peor UX | ❌ Rechazada (decisión del usuario) |
| Flujo mínimo (sin cuenta/payee) | Menos pasos | Pierde datos que la UI sí pide | ❌ Rechazada (decisión del usuario: flujo completo) |
| Diálogo vía reply keyboard (teclado fijo) | Simple | No soporta listas dinámicas largas ni edición del mensaje | ❌ Rechazada |
| Persistir sesión en SQLite | Sobrevive reinicios | Complejidad y disco innecesarios; la app ya persiste el resultado (transacción) | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Estado de conversación en memoria con TTL.
- **Contexto**: un flujo multi-paso necesita recordar entre mensajes qué respondió el usuario en el paso anterior.
- **Decisión**: struct `budgetTxSession` guardado en un `map[chatID]*budgetTxSession` protegido con `sync.Mutex` dentro de `TelegramBotService`, con `expiresAt` para limpieza por inactividad (~15 min).
- **Consecuencias**: se pierde la conversación si el server se reinicia (aceptado, costo bajo). No requiere tablas ni migraciones. El comando `/budget_transaction` reemplaza la sesión activa del chat.

**ADR-002**: Moneda derivada de la cuenta elegida.
- **Contexto**: `transactions.currency_id` es obligatorio; en la UI el default es la primera moneda, pero cada cuenta tiene su moneda (`accounts.currency_id`).
- **Decisión**: la moneda de la transacción se toma de la cuenta seleccionada en el paso Cuenta. No hay paso de moneda separado.
- **Consecuencias**: menos pasos en el flujo; coherencia garantizada entre cuenta y moneda. Si más adelante se quiere elegir moneda distinta a la cuenta, se agrega como P2.

**ADR-003**: Callback data acotada con prefijo `bt:`.
- **Contexto**: Telegram limita el callback data a 64 bytes y el handler se registra por prefijo.
- **Decisión**: datos `bt:grp:<id>`, `bt:cat:<id>`, `bt:type:<outflow|inflow>`, `bt:acct:<id>`, `bt:ok`, `bt:cancel`. Un único handler `RegisterHandler(HandlerTypeCallbackQueryData, "bt:", MatchTypePrefix, ...)`.
- **Consecuencias**: simple y robusto; el paso actual de la sesión valida que el callback sea coherente (p.ej. ignorar un callback de grupo si la sesión está en el paso categoría).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram user]
   │  /budget_transaction + callbacks (bt:*) + texto
   ▼
[TelegramBotService.runBot (go-telegram/bot, polling)]
   │  handlers: handleBudgetTransaction (comando) · handleBudgetCallback (bt:) · handleDefault (texto)
   ▼
[budgetTxSession map[chatID] → máquina de estados, sync.Mutex + TTL]
   │  pasos: grupo → categoría → tipo → monto → fecha → cuenta → payee → confirmar
   ▼
[services: CategoryGroupService.List · AccountService.List · CurrencyService (solo lectura)
           · BudgetTransactionService.Create (validación + persistencia)]
   ▼
[SQLite (transactions)]
```

### 4.2 Componentes

#### 4.2.1 `internal/services/telegram_bot.go`
- **Responsabilidad**: hosting del comando, callbacks y máquina de estados del flujo `budget_transaction`.
- **Nuevos tipos**:
  - `budgetTxStep` (`const`): `stepGroup`, `stepCategory`, `stepType`, `stepAmount`, `stepDate`, `stepAccount`, `stepPayee`, `stepConfirm`.
  - `budgetTxSession`: `chatID int64`, `step budgetTxStep`, `groupID int64`, `categoryID int64`, `categoryName string`, `isInflow bool`, `amount float64`, `date string`, `accountID int64`, `accountName string`, `currencyID int64`, `currencySymbol string`, `payee string`, `expiresAt time.Time`.
  - Campos nuevos en `TelegramBotService`: `budgetTxSessions map[int64]*budgetTxSession`, `txMu sync.Mutex`, `groups *CategoryGroupService`, `accounts *AccountService`, `currencies *CurrencyService`, `transactions *BudgetTransactionService`.
- **Métodos nuevos**:
  - `handleBudgetTransaction(ctx, b, update)`: valida prerequisitos, crea/reemplaza sesión, envía el teclado de Grupos.
  - `handleBudgetCallback(ctx, b, update)`: parsea `bt:...`, valida sesión activa y coherencia de paso, avanza la máquina de estados, responde con `AnswerCallbackQuery` y edita el teclado.
  - `budgetSendKeyboard(ctx, b, chatID, text, buttons)`: helper para enviar/editar mensaje con inline keyboard.
  - `budgetAskAmount` / `budgetAskDate` / `budgetAskPayee`: mensajes de texto que esperan respuesta.
  - `budgetHandleText(ctx, b, update)`: invocado desde `handleDefault` cuando hay sesión activa; valida y avanza pasos de texto (monto/fecha/payee).
  - `budgetSummary(session) string`: resumen formateado para la confirmación (usa `formatAmount`).
  - `budgetCleanup()`: recorre sesiones y elimina las expiradas (se invoca al iniciar un comando o en `handleBudgetCallback`).
  - Helpers puros testables: `parseAmount(string) (float64, bool)`, `parseDate(string, time.Time) (string, bool)`.
- **Dependencias**: `CategoryGroupService`, `AccountService`, `CurrencyService`, `BudgetTransactionService`, `SystemSettingsService` (timezone + currency format, ya presente).
- **Ubicación**: `internal/services/telegram_bot.go`.

#### 4.2.2 `cmd/server/main.go`
- **Responsabilidad**: inyectar los servicios de presupuesto al bot.
- **Cambio**: `NewTelegramBotService(...)` pasa `categoryGroupService, accountService, currencyService, budgetTransactionService` además de los argumentos actuales.
- **Ubicación**: `cmd/server/main.go` (línea ~128).

#### 4.2.3 `internal/services/telegram_bot_test.go`
- **Responsabilidad**: unit tests de la máquina de estados y helpers puros.
- **Cobertura**: `parseAmount`, `parseDate`, `budgetSummary`, transiciones de pasos, cancelar, prerequisitos faltantes. Se actualiza el test de ciclo de vida por la nueva firma del constructor (`telegram_bot_test.go:524`).

### 4.3 Modelo de datos

Sin cambios de esquema. Solo memoria:

```
budgetTxSession (en memoria, por chat_id)
- chatID: int64
- step: budgetTxStep
- groupID / categoryID / categoryName: selección
- isInflow: bool  (true = ingreso, false = gasto)
- amount: float64
- date: string ("YYYY-MM-DD")
- accountID / accountName / currencyID / currencySymbol
- payee: string
- expiresAt: time.Time  (TTL ~15 min)
```

### 4.4 APIs / Contratos

No hay endpoints HTTP nuevos. Interfaz del bot:

```
Comando:
  /budget_transaction      → inicia/renueva la conversación

Callback data (prefijo bt:):
  bt:grp:<id>              → seleccionar grupo
  bt:cat:<id>              → seleccionar categoría
  bt:type:outflow|inflow   → tipo de transacción
  bt:acct:<id>             → seleccionar cuenta
  bt:ok                    → confirmar y crear
  bt:cancel                → cancelar y descartar sesión

Texto:
  <monto>                  → paso monto (número > 0)
  <YYYY-MM-DD> | hoy       → paso fecha
  <payee> | /saltar        → paso payee
```

Confirmar llama `BudgetTransactionService.Create` con:
```json
{ "account_id": <acct>, "category_id": <cat>, "currency_id": <cur>,
  "date": "YYYY-MM-DD", "payee": "<payee>", "memo": "",
  "outflow": <amount> | 0, "inflow": 0 | <amount>, "cleared": false }
```

### 4.5 Dependencias

- **Internas**: `cmd/server/main.go`, `internal/services/telegram_bot.go`, `internal/services/telegram_bot_test.go`.
- **Externas**: ninguna nueva. `github.com/go-telegram/bot` v1.27.0 ya está en `go.mod` y soporta inline keyboards + callbacks.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: `/budget_transaction` aparece en el menú de comandos del bot y en la ayuda de `/start`.
- [ ] CA-002: Si no hay grupos con categorías o no hay cuentas, el bot avisa qué falta crear y no inicia la conversación.
- [ ] CA-003: El flujo completo (grupo → categoría → tipo → monto → fecha → cuenta → payee → confirmar) crea una transacción visible en la grilla del presupuesto (`/budget/transacciones`) con los datos elegidos.
- [ ] CA-004: Un monto inválido (0, negativo, texto) o una fecha inválida re-preguntan el mismo paso sin perder el estado.
- [ ] CA-005: La fecha vacía/"hoy" usa la fecha actual en la zona horaria configurada.
- [ ] CA-006: La moneda de la transacción es la de la cuenta elegida.
- [ ] CA-007: *Cancelar* descarta la sesión en cualquier paso; un mensaje posterior del mismo chat vuelve a la respuesta "comando no reconocido".
- [ ] CA-008: Los callback queries tienen feedback (`AnswerCallbackQuery`) y el teclado avanza/editando el mensaje (REQ-014).
- [ ] CA-009: El bot solo responde a chats autorizados (allowlist) tanto para comandos como para callbacks.

### 5.2 No funcionales

- [ ] CA-NF-001: Sin dependencias nuevas en Go; `go build` y `go test ./...` pasan; build multi-arch intacto.
- [ ] CA-NF-002: Sesiones con TTL de 15 min y limpieza perezosa; sin fugas de memoria significativas.

### 5.3 Testing

- **Unit tests**: `parseAmount` (válidos/inválidos, decimales), `parseDate` (formatos, hoy), `budgetSummary` (gasto/ingreso, símbolo de moneda), transiciones de la máquina de estados (incl. callback fuera de paso), cancelar.
- **Integration tests**: flujo completo con storages de prueba en `transaction_test` (grupo→categoría→...→`Create` persiste y afecta `activity`).
- **E2E tests**: prueba manual local con un bot real (token de prueba) o mocking del bot; verificación de que la transacción aparece en la grilla.
- **Carga/Performance**: irrelevante (estado por chat en memoria, consultas de listado ya existentes).

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Inyección de dependencias: `main.go` pasa `CategoryGroupService`, `AccountService`, `CurrencyService`, `BudgetTransactionService` al bot; actualizar firma de `NewTelegramBotService` y su test de ciclo de vida | 0.5 día | Ninguna |
| 2 | Máquina de estados en `telegram_bot.go`: tipos, mapa de sesiones con TTL, handler del comando, handler de callbacks, routing en `handleDefault`, helpers de teclado y de texto | 1.5 días | Fase 1 |
| 3 | Registro del comando en `registerCommands` + ayuda `/start` + hint en `handleDefault` | 0.25 día | Fase 2 |
| 4 | Unit tests (parseo, resumen, transiciones) + integración mínima | 0.5 día | Fases 2-3 |
| 5 | Compilación, tests, corrida local para evaluación del usuario | 0.25 día | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: flujo completo de texto+botones que crea la transacción correctamente (Fases 1-3).
2. **V1.0**: tests + evaluación manual del usuario (Fases 4-5).

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Callback query fuera de paso (usuario toca un botón viejo) | Media | Medio | La máquina valida que el callback corresponda al paso actual; si no, responde aviso y reenvía el teclado correcto |
| Estado inconsistente al reiniciar el server | Media | Bajo | Aceptado (ADR-001); el usuario reinicia `/budget_transaction` |
| Límite de 64 bytes de callback data | Baja | Bajo | IDs numéricos cortos (`bt:cat:123`) |
| Montos con separador decimal distinto (coma vs punto) | Media | Baja | `parseAmount` acepta punto y coma decimal normalizado |
| La firma del constructor rompe otros tests/callers | Baja | Medio | Solo hay un caller (`main.go`) y un test; se actualizan en la misma spec |

## 8. Notas y Referencias

- `github.com/go-telegram/bot` v1.27.0: `models/reply_markup.go` (inline keyboards), `handlers.go` (`HandlerTypeCallbackQueryData`, `MatchTypePrefix`), `methods.go` (`AnswerCallbackQuery`, `EditMessageReplyMarkup`).
- `internal/services/telegram_bot.go` (patrón de handlers existente), `internal/services/transaction.go` (validación de `Create`), `cmd/server/main.go:134-141` (servicios de presupuesto).
- SPEC-093 (módulo presupuesto YNAB), SPEC-084 (formato de montos y zona horaria del bot), SPEC-079 (bot Telegram), SPEC-080 (registro de comandos sin slash).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-28 | opencode | Creación inicial de la especificación (requerimiento relevado con usuario: botónes inline, flujo completo grupo/categoría/tipo/monto/fecha/cuenta/payee/confirmación). Estado: draft → pending_execution |
| 2026-09-28 | opencode | Implementación: inyección de servicios de presupuesto en `TelegramBotService` (main.go), máquina de estados `/budget_transaction` en `telegram_bot.go` (pasos + callbacks `bt:` + routing en `handleDefault`), registro del comando y ayuda `/start`, unit tests en `budget_transaction_test.go`. Estado: in_progress. Build y `go test ./...` en verde; server local levantado para evaluación del usuario. |
| 2026-09-28 | opencode | **Release**: usuario autoriza release asumiendo funcionamiento correcto (prueba en vivo pendiente de despliegue a iHost). Merge de `feature/SPEC-097` a `main` + push a `origin/main`. Issue #100 cerrado con label `spec/released`. Worktree y rama de la spec limpiados. |