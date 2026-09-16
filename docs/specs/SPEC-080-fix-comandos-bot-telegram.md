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

Esta spec corrige ambos: registra los handlers con el patrón correcto (sin slash) y llama `SetMyCommands` al iniciar el polling para que Telegram muestre solo los comandos actuales (`start` y `pendientes`).

Consideraciones iHost: cambios solo en `internal/services/telegram_bot.go` (backend). Sin cambios de DB, API, frontend ni dependencias nuevas. Una llamada extra a la Bot API al arrancar el polling; si falla, solo se loguea (no afecta el polling ni el server).

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Corregir el registro de handlers: `RegisterHandler(HandlerTypeMessageText, "pendientes", MatchTypeCommand, ...)` y `"start"` (SIN slash), alineado con el matcher de `go-telegram/bot` v1.27. El comando `/pendientes` enviado desde Telegram DEBE ejecutar el handler real (no el default).
2. **REQ-002**: Llamar `SetMyCommands` al iniciar el polling con la lista actual de comandos (`start` → "Bienvenida y comandos disponibles", `pendientes` → "Servicios con facturas pendientes"), sobrescribiendo la lista persistida por el bot Python (los 9 comandos viejos dejan de mostrarse en el menú).
3. **REQ-003**: El fallo de `SetMyCommands` (red caída, API down) NO debe detener ni retrasar el polling: solo loguear warning y continuar.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-004**: Verificar con un test unitario que el matcher de la librería coincide con los handlers registrados (simulando un `Update` con entidad `BotCommand` para `/pendientes` y `/start`), para evitar regresiones.

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
- **Decisión**: `RegisterHandler(bot.HandlerTypeMessageText, "pendientes", bot.MatchTypeCommand, s.handlePendientes)` (y `"start"`).
- **Consecuencias**: `/pendientes` vuelve a responder con datos reales. El patrón queda alineado con el ejemplo oficial de la librería, evitando confusión futura.

**ADR-002**: `SetMyCommands` en cada arranque de polling, best-effort
- **Contexto**: La lista de comandos es server-side en Telegram; el bot Python la dejó con 9 comandos obsoletos.
- **Decisión**: En `runBot`, tras crear el bot y antes de `Start(ctx)`, llamar `SetMyCommands` con `start` y `pendientes` (contexto con timeout de 10s). Si falla, warning y continuar (REQ-003).
- **Consecuencias**: El menú de Telegram muestra solo los comandos actuales. No se borra la lista al detener (REQ-005).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram UI (menú /)] <-- setMyCommands (start, pendientes) --+
                                                                |
[Telegram] --(getUpdates)---> [TelegramBotService (server Go)]
     ^                             | handlers: "pendientes", "start"
     | sendMessage                 v
     +-------------------- [storage.BillStorage] -- SQLite local
```

### 4.2 Componentes

#### 4.2.1 TelegramBotService (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Corregir el registro de handlers y registrar los comandos en Telegram.
- **Cambios**:
  1. En `runBot`: `b.RegisterHandler(bot.HandlerTypeMessageText, "pendientes", bot.MatchTypeCommand, s.handlePendientes)` y `b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, s.handleStart)` (sin slash).
  2. En `runBot`, antes de `b.Start(ctx)`: llamada a `SetMyCommands` con las descripciones de `start` y `pendientes`, usando un contexto con timeout (10s). Si falla, `slog.Warn` y continuar.
- **Dependencias**: `github.com/go-telegram/bot` (ya incluida), `models.BotCommand`.
- **Ubicación**: `internal/services/telegram_bot.go`

#### 4.2.2 Tests (`internal/services/telegram_bot_test.go`)
- **Responsabilidad**: Verificar el matcheo de los handlers registrados y el formato del mensaje.
- **Cambios**: Agregar test que construya un `models.Update` con `Message.Text = "/pendientes"` y `Entities` con un `BotCommand` (Offset=0, Length=11) y verifique que el matcher de la librería devuelve true con el patrón `"pendientes"` (y `"start"`). Test del registro de handlers vía `RegisterHandler` + `match()` si es accesible, o verificación del comportamiento esperado.

### 4.3 Modelo de datos

Sin cambios. No hay migraciones ni claves nuevas.

### 4.4 APIs / Contratos

Sin cambios de API HTTP. Solo Bot API interna:
- `setMyCommands` con `commands: [{command: "start", description: ...}, {command: "pendientes", description: ...}]`.

### 4.5 Dependencias

- **Internas**: `internal/services/telegram_bot.go`, `internal/services/telegram_bot_test.go`.
- **Externas**: `github.com/go-telegram/bot` v1.27.0 (ya en go.mod, sin cambios).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Enviando `/pendientes` al bot (producción o local con token real), responde la lista de servicios con facturas pendientes (NO "Comando no reconocido").
- [ ] CA-002: El menú de comandos de Telegram (al escribir `/`) muestra solo `start` y `pendientes`; los 9 comandos del bot Python desaparecen.
- [ ] CA-003: `/start` responde la bienvenida con el comando disponible.
- [ ] CA-004: Un mensaje no reconocido sigue respondiendo "Comando no reconocido".
- [ ] CA-005: Si `SetMyCommands` falla (simulado), el polling arranca igual y el server responde `/health` normalmente.
- [ ] CA-DARK: No aplica (no toca UI).
- [ ] CA-BACK: No aplica (no toca UI).

### 5.2 No funcionales

- [ ] CA-NF-001: `go build ./...`, `go vet ./...` y `go test ./...` sin errores.
- [ ] CA-NF-002: Sin dependencias nuevas en `go.mod`.

### 5.3 Testing

- **Unit tests**: matcheo del patrón `"pendientes"`/`"start"` contra un `Update` simulado con entidad `BotCommand`; mensajes no reconocidos caen al default.
- **Integration tests**: no aplican (sin cambios de storage/API).
- **E2E**: enviar `/pendientes` desde Telegram contra el server local con token real (prueba manual con el usuario) y verificar el menú de comandos.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Fix de handlers (sin slash) + `SetMyCommands` en `runBot` | 0.25d | Ninguna |
| 2 | Test unitario de matcheo de comandos | 0.25d | Fase 1 |
| 3 | Build, vet, tests locales + prueba manual con el usuario | 0.5d | Fase 2 |
| 4 | Release: merge a main, build Docker multi-arch, deploy iHost | 0.5d | Fase 3 |

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