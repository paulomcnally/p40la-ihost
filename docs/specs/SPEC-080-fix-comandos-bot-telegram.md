---
title: "Bot Telegram: fix matcheo de comandos + registro de comandos (SetMyCommands)"
id: "SPEC-080"
status: "in_progress"
author: "p40la-ihost-team"
created: "2026-09-16"
updated: "2026-09-16"
github_issue: 83
---

# Bot Telegram: fix matcheo de comandos + registro de comandos (SetMyCommands)

**ID**: SPEC-080  
**Estado**: in_progress  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-16  
**Actualizado**: 2026-09-16

---

## 1. Resumen Ejecutivo

El bot de Telegram integrado al server (SPEC-079) está en producción, pero se detectaron dos problemas:

1. **El comando `/pendientes` no matchea**: el handler se registró con el patrón `"/pendientes"` (con slash), pero la librería `go-telegram/bot` v1.27 compara el comando **sin** el slash (`data[Offset+1:Offset+Length]` en `handlers.go`). Su ejemplo oficial registra `"foo"` sin slash. Resultado: TODOS los mensajes caen al handler por defecto y el bot responde "Comando no reconocido" incluso para `/pendientes` (evidencia: mensaje del usuario del 2026-09-16 15:59).
2. **El menú de comandos de Telegram sigue mostrando los 9 comandos del bot Python anterior**: el bot Python (`~/.hermes/bots/p40la-tg/bot.py` línea 543) llamaba `set_my_commands()` con `start, casas, servicios, instituciones, facturas, autos, search, stats, sql`. Esa lista persiste server-side en Telegram y el bot Go NUNCA llama `SetMyCommands` para sobrescribirla.

Esta spec corrige ambos y, por requerimiento del usuario durante el desarrollo, **separa el comando genérico `/pendientes` en dos comandos específicos**:

- `/servicios_pendientes` — servicios con facturas pendientes (lo que hacía `/pendientes`).
- `/deudas_pendientes` — deudas con cuotas (`debt_bills`) pendientes: descripción, institución, cantidad de cuotas y monto total.

El comando `/pendientes` se elimina. Los handlers se registran con el patrón correcto (sin slash) y se llama `SetMyCommands` al iniciar el polling para que Telegram muestre solo los comandos actuales (`start`, `servicios_pendientes`, `deudas_pendientes`).

Consideraciones iHost: cambios en `internal/services/telegram_bot.go`, `internal/storage/debt_bill.go` (nuevo método de lectura) y wiring en `cmd/server/main.go`. Sin cambios de DB (solo lectura de `debt_bills`), API, frontend ni dependencias nuevas. Una llamada extra a la Bot API al arrancar el polling; si falla, solo se loguea (no afecta el polling ni el server).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Corregir el registro de handlers: `RegisterHandler(HandlerTypeMessageText, "servicios_pendientes", MatchTypeCommand, ...)`, `"deudas_pendientes"` y `"start"` (SIN slash), alineado con el matcher de `go-telegram/bot` v1.27. Los comandos enviados desde Telegram DEBEN ejecutar su handler real (no el default).
2. **REQ-002**: Llamar `SetMyCommands` al iniciar el polling con la lista actual de comandos (`start` → "Bienvenida y comandos disponibles", `servicios_pendientes` → "Servicios con facturas pendientes", `deudas_pendientes` → "Deudas con cuotas pendientes"), sobrescribiendo la lista persistida por el bot Python (los 9 comandos viejos dejan de mostrarse en el menú).
3. **REQ-003**: El fallo de `SetMyCommands` (red caída, API down) NO debe detener ni retrasar el polling: solo loguear warning y continuar.
4. **REQ-006**: `/servicios_pendientes` responde los servicios con facturas pendientes (`bills.status='pending'`, `deleted_at IS NULL`): nombre del servicio, casa, cantidad de facturas y monto total (formato actual de `/pendientes`, renombrado). El comando `/pendientes` deja de existir.
5. **REQ-007**: `/deudas_pendientes` responde las deudas con cuotas pendientes (`debt_bills.status='pending'`, `deleted_at IS NULL`, deuda no eliminada): descripción de la deuda, institución, cantidad de cuotas pendientes y monto total. Ordenado por monto descendente, `LIMIT 30`.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Verificar con un test unitario que el matcher de la librería coincide con los handlers registrados (simulando un `Update` con entidad `BotCommand` para `/servicios_pendientes`, `/deudas_pendientes` y `/start`), para evitar regresiones.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-005**: Al detener el bot (Stop) no se borra la lista de comandos (queda la última registrada). No usar `DeleteMyCommands` para evitar menú vacío en caso de reconfiguraciones temporales.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Una llamada HTTP extra a la Bot API por arranque de polling. Sin impacto en memoria.
- **Seguridad**: Sin cambios de tokens ni exposición de secretos.
- **Almacenamiento**: Sin archivos nuevos.
- **Disponibilidad**: El polling arranca aunque `SetMyCommands` falle (timeout corto de 10s).
- **iHost**: Sin dependencias nuevas; solo se modifica `internal/services/telegram_bot.go` y sus tests.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Código de la librería `go-telegram/bot` v1.27.0 (`handlers.go`): el matcher de `MatchTypeCommand` itera `update.Message.Entities` y compara `data[e.Offset+1:e.Offset+e.Length]` contra el patrón. El slice **omite el slash** (offset+1). Por lo tanto el patrón registrado debe ir SIN `/`.
- Ejemplo oficial de la librería (`examples/command_handler/main.go`): `b.RegisterHandler(bot.HandlerTypeMessageText, "foo", bot.MatchTypeCommand, fooHandler)` — sin slash.
- Evidencia en producción: mensaje `P40la` → "Comando no reconocido. Usa /pendientes..." (texto del `handleDefault` del bot Go) al enviar `/pendientes` el 2026-09-16 15:59.
- El bot Python anterior registraba 9 comandos con `set_my_commands` (bot.py:543); la lista persiste en Telegram hasta que se sobrescriba con `SetMyCommands`.
- API `SetMyCommands` disponible en la librería (`methods.go:561`): `SetMyCommands(ctx, *SetMyCommandsParams{Commands: []models.BotCommand{...}})`.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Registrar handlers sin slash (`"pendientes"`) | Alineado con el matcher de la librería y su ejemplo oficial | Ninguna | ✅ Seleccionada |
| Registrar con slash y usar `MatchTypeExact` | El patrón se lee natural con `/` | No maneja `/pendientes@BotName` ni payloads; se desvía del patrón de la librería | ❌ Rechazada |
| Parsear comandos manualmente en el default handler | Cero dependencia del matcher | Duplica lógica de la librería, más código a mantener | ❌ Rechazada |
| No tocar handlers y solo agregar SetMyCommands | Resuelve solo el menú | El comando `/pendientes` seguiría roto | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Patrón de handlers sin slash (convención de la librería)
- **Contexto**: El matcher de `MatchTypeCommand` compara el comando sin `/`; el código actual registra con `/` y nunca matchea.
- **Decisión**: `RegisterHandler(bot.HandlerTypeMessageText, "servicios_pendientes", bot.MatchTypeCommand, s.handleServiciosPendientes)`, `"deudas_pendientes"` y `"start"`.
- **Consecuencias**: Los comandos vuelven a responder con datos reales. El patrón queda alineado con el ejemplo oficial de la librería, evitando confusión futura.

**ADR-002**: `SetMyCommands` en cada arranque de polling, best-effort
- **Contexto**: La lista de comandos es server-side en Telegram; el bot Python la dejó con 9 comandos obsoletos.
- **Decisión**: En `runBot`, tras crear el bot y antes de `Start(ctx)`, llamar `SetMyCommands` con `start`, `servicios_pendientes` y `deudas_pendientes` (contexto con timeout de 10s). Si falla, warning y continuar (REQ-003).
- **Consecuencias**: El menú de Telegram muestra solo los comandos actuales. No se borra la lista al detener (REQ-005).

**ADR-003**: Dos comandos específicos en vez de `/pendientes` genérico
- **Contexto**: El usuario pidió separar el comando genérico en dos: servicios y deudas.
- **Decisión**: `/servicios_pendientes` (renombra `/pendientes`, misma respuesta) y `/deudas_pendientes` (nuevo, agrupa cuotas pendientes de `debt_bills` por deuda). Se elimina `/pendientes`.
- **Consecuencias**: Menú más claro y extensible; la lógica de formato se parametriza para ambas listas.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram UI (menú /)] <-- setMyCommands (start, servicios_pendientes, deudas_pendientes) --+
                                                                                             |
[Telegram] --(getUpdates)---> [TelegramBotService (server Go)]
     ^                             | handlers: "servicios_pendientes", "deudas_pendientes", "start"
     | sendMessage                 v
     +-------------------- [BillStorage / DebtBillStorage] -- SQLite local
```

### 4.2 Componentes

#### 4.2.1 TelegramBotService (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Corregir el registro de handlers y registrar los comandos en Telegram.
- **Cambios**:
  1. En `runBot`: `b.RegisterHandler(bot.HandlerTypeMessageText, "servicios_pendientes", bot.MatchTypeCommand, s.handleServiciosPendientes)`, `b.RegisterHandler(bot.HandlerTypeMessageText, "deudas_pendientes", bot.MatchTypeCommand, s.handleDeudasPendientes)` y `"start"` (sin slash).
  2. En `runBot`, antes de `b.Start(ctx)`: llamada a `SetMyCommands` con las descripciones de `start`, `servicios_pendientes` y `deudas_pendientes`, usando un contexto con timeout (10s). Si falla, `slog.Warn` y continuar.
  3. `handlePendientes`/`formatPendientes` → `handleServiciosPendientes`/`formatServiciosPendientes`. Nuevo `handleDeudasPendientes`/`formatDeudasPendientes` (agrupa cuotas por deuda: descripción, institución, cant, monto, símbolo de moneda). `NewTelegramBotService` recibe también `*storage.DebtBillStorage`.
- **Dependencias**: `github.com/go-telegram/bot` (ya incluida), `models.BotCommand`, `storage.DebtBillStorage`.
- **Ubicación**: `internal/services/telegram_bot.go`

#### 4.2.2 DebtBillStorage (`internal/storage/debt_bill.go`)
- **Responsabilidad**: Nuevo método de lectura `ListPendingWithDetails(ctx)` que devuelve las cuotas pendientes (`status='pending'`, `deleted_at IS NULL`) con descripción de deuda, institución y símbolo de moneda, excluyendo deudas eliminadas.
- **Ubicación**: `internal/storage/debt_bill.go`

#### 4.2.3 main.go (`cmd/server/main.go`)
- **Responsabilidad**: Pasar `debtBillStorage` al constructor del bot.
- **Ubicación**: `cmd/server/main.go`

#### 4.2.4 Tests (`internal/services/telegram_bot_test.go` + `internal/storage/debt_bill_test.go`)
- **Responsabilidad**: Verificar el matcheo de los handlers registrados y el formato de ambos mensajes.
- **Cambios**: Test E2E con `ProcessUpdate` (dispatcher real de la librería + Telegram API fake) para `servicios_pendientes`, `deudas_pendientes` y `start`; test de `formatServiciosPendientes` y `formatDeudasPendientes` (con/sin filas, orden por monto); test del storage para `ListPendingWithDetails`.

### 4.3 Modelo de datos

Sin cambios de esquema. No hay migraciones ni claves nuevas. Nuevo struct de lectura `PendingDebtDetail` en `internal/models/debt_bill.go`:

```
PendingDebtDetail:
- DebtID: int64
- DebtDescription: string
- InstitutionName: string
- DueDate: string
- Amount: float64
- CurrencySymbol: string
```

### 4.4 APIs / Contratos

Sin cambios de API HTTP. Solo Bot API interna:
- `setMyCommands` con `commands: [{command: "start", ...}, {command: "servicios_pendientes", ...}, {command: "deudas_pendientes", ...}]`.

#### Comando Telegram: `/servicios_pendientes` (ex `/pendientes`)

```
⏳ *Servicios con facturas pendientes* (2)

⚡ *Claro Internet*
  Casa: Casa Central
  Facturas: 3
  Pendiente: $1,250.00
```

Si no hay pendientes: `✅ No hay facturas pendientes.`

#### Comando Telegram: `/deudas_pendientes`

```
⏳ *Deudas con cuotas pendientes* (2)

💳 *Préstamo LAFISE*
  Institución: Banco LAFISE
  Cuotas: 5
  Pendiente: $1,250.00
```

Si no hay pendientes: `✅ No hay deudas pendientes.`

### 4.5 Dependencias

- **Internas**: `internal/services/telegram_bot.go`, `internal/services/telegram_bot_test.go`, `internal/storage/debt_bill.go`, `internal/models/debt_bill.go`, `cmd/server/main.go`.
- **Externas**: `github.com/go-telegram/bot` v1.27.0 (ya en go.mod, sin cambios).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Enviando `/servicios_pendientes` al bot (producción o local con token real), responde la lista de servicios con facturas pendientes (NO "Comando no reconocido").
- [ ] CA-002: El menú de comandos de Telegram (al escribir `/`) muestra solo `start`, `servicios_pendientes` y `deudas_pendientes`; los 9 comandos del bot Python desaparecen.
- [ ] CA-003: `/start` responde la bienvenida con los comandos disponibles.
- [ ] CA-004: Un mensaje no reconocido sigue respondiendo "Comando no reconocido".
- [ ] CA-005: Si `SetMyCommands` falla (simulado), el polling arranca igual y el server responde `/health` normalmente.
- [ ] CA-006: `/deudas_pendientes` responde las deudas con cuotas pendientes (descripción, institución, cant, monto) ordenadas por monto descendente.
- [ ] CA-007: `/pendientes` ya no responde (comando eliminado) — cae al default handler.
- [ ] CA-DARK: No aplica (no toca UI).
- [ ] CA-BACK: No aplica (no toca UI).

### 5.2 No funcionales

- [ ] CA-NF-001: `go build ./...`, `go vet ./...` y `go test ./...` sin errores.
- [ ] CA-NF-002: Sin dependencias nuevas en `go.mod`.

### 5.3 Testing

- **Unit tests**: matcheo de los patrones `"servicios_pendientes"`/`"deudas_pendientes"`/`"start"` contra un `Update` simulado con entidad `BotCommand` (dispatcher real); mensajes no reconocidos caen al default; `formatServiciosPendientes` y `formatDeudasPendientes` (con/sin filas, orden, formato de moneda).
- **Integration tests**: `ListPendingWithDetails` de `DebtBillStorage` con DB de test (`:memory:` + migraciones).
- **E2E**: enviar `/servicios_pendientes` y `/deudas_pendientes` desde Telegram contra el server local con token real (prueba manual con el usuario) y verificar el menú de comandos.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Fix de handlers (sin slash) + `SetMyCommands` en `runBot` | 0.25d | Ninguna |
| 2 | Test unitario de matcheo de comandos | 0.25d | Fase 1 |
| 3 | Iteración usuario: `/servicios_pendientes` + `/deudas_pendientes` (storage + handler + formatter + tests) | 0.5d | Fase 2 |
| 4 | Build, vet, tests locales + prueba manual con el usuario | 0.5d | Fase 3 |
| 5 | Release: merge a main, build Docker multi-arch, deploy iHost | 0.5d | Fase 4 |

### 6.2 Milestones

1. **MVP**: `/pendientes` responde con datos reales y el menú de Telegram muestra solo los comandos actuales.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| El matcher de la librería cambia en futuras versiones | Baja | Medio | Test unitario de matcheo que detecta la regresión; go.mod fija v1.27.0 |
| `SetMyCommands` falla por red/API | Media | Bajo | Best-effort: warning + continuar; el menú viejo quedaría hasta el próximo arranque exitoso |
| Otro consumidor del token (bot Python re-arrancado) compite por getUpdates | Media | Alto | No volver a arrancar el bot Python; documentar en la spec el conflicto 409 |
| Comando con `@BotUsername` en grupos | Baja | Bajo | El matcher compara entidad completa; en chats directos (caso de uso) no aplica |

## 8. Notas y Referencias

- Matcher de la librería: `$(go env GOMODCACHE)/github.com/go-telegram/bot@v1.27.0/handlers.go` (`MatchTypeCommand`)
- Ejemplo oficial: `.../go-telegram/bot@v1.27.0/examples/command_handler/main.go`
- `SetMyCommands`: `.../go-telegram/bot@v1.27.0/methods.go:561` y `methods_params.go:743`
- Bot Python anterior: `~/.hermes/bots/p40la-tg/bot.py` (comandos en línea 532-543)
- Spec previa: SPEC-079 (bot de Telegram integrado)

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-16 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-16 | p40la-ihost-team | Iteración solicitada por el usuario: `/pendientes` se reemplaza por `/servicios_pendientes` y se agrega `/deudas_pendientes` (cuotas pendientes de deudas) |