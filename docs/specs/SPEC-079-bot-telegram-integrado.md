---
title: "Bot de Telegram integrado al server (migración Python → Go)"
id: "SPEC-079"
status: "released"
author: "p40la-ihost-team"
created: "2026-09-16"
updated: "2026-09-16"
github_issue: 82
---

# Bot de Telegram integrado al server (migración Python → Go)

**ID**: SPEC-079  
**Estado**: released  
**Autor**: p40la-ihost-team  
**Creado**: 2026-09-16  
**Actualizado**: 2026-09-16

---

## 1. Resumen Ejecutivo

Actualmente existe un bot de Telegram en Python (`~/.hermes/bots/p40la-tg/bot.py`) que consulta la DB del iHost **vía SSH** (`sshpass ssh` a `ihost.local:2222` + `sqlite3` remoto). Corre como proceso independiente en la dev machine, lee el token desde un `.env`, y expone 9 comandos read-only (casas, servicios, instituciones, facturas, autos, search, stats, sql).

Esta spec integra el bot **dentro del server Go** (`cmd/server/main.go`) como un servicio embebido con acceso **directo a la DB SQLite local** (sin SSH), siguiendo el stack del proyecto (Go + net/http + mínimas dependencias). El token se configura desde la página de Configuración (nueva sección "Bot de Telegram", patrón Voice Monkey/SPEC-033) y se persiste en `system_settings`.

Por requerimiento del usuario, el bot arranca con **un solo comando**: `/pendientes`, que devuelve todos los servicios con facturas pendientes (nombre, casa, cantidad y monto pendiente). Los demás comandos del bot Python se descartan por ahora (se pueden agregar en specs futuras). El comando `/sql` libre NO se migra (superficie de ataque innecesaria con el bot embebido en el server de producción).

Consideraciones iHost: cero procesos extra, cero dependencias nuevas pesadas (librería `go-telegram/bot` es zero-dependencies), polling liviano, y la DB se consulta por los storages existentes sin abrir conexiones adicionales.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Migrar el bot de Telegram de Python a Go e integrarlo embebido en el server (`cmd/server/main.go`) como goroutine que arranca con el server.
2. **REQ-002**: Nuevo servicio `TelegramBotService` en `internal/services/telegram_bot.go` que consulta la DB local **directamente** (vía storages existentes, sin SSH) y responde comandos de Telegram por long polling.
3. **REQ-003**: Config del bot persistida en `system_settings` con dos claves: `telegram_bot_enabled` (toggle) y `telegram_bot_token` (secreto). El bot solo arranca el polling si `enabled=1` y hay token.
4. **REQ-004**: Comando único `/pendientes`: devuelve los servicios con facturas pendientes (`bills.status='pending'` y `deleted_at IS NULL`), mostrando nombre del servicio, casa, cantidad de facturas y monto total pendiente. Ordenado por monto descendente.
5. **REQ-005**: Sección "Bot de Telegram" en la página de Configuración (patrón SettingsVoiceMonkeyPage): toggle habilitar + campo token (type=password) + botón "Reconfigurar" + estado "configurado". Guardado vía `PUT /api/system-settings` existente.
6. **REQ-006**: Reconfiguración en runtime: al guardar/limpiar el token o toggle, el bot arranca/detiene/reinicia el polling sin reiniciar el server.
7. **REQ-007**: Handler por defecto para mensajes no reconocidos (responde con el comando disponible).

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-008**: Allowlist de chat_ids autorizados (comma-separated) en `system_settings` (`telegram_bot_chat_ids`). Si está vacía, el bot responde a cualquier chat (comportamiento actual del bot Python). Si se configura, solo responde a esos chats.
2. **REQ-009**: `/start` responde con la bienvenida y el comando disponible.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-010**: Botón "Probar" que envía un mensaje de prueba al último chat autorizado que interactuó con el bot.

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: Polling con timeout de 30s; memoria objetivo < 5MB extra. Sin goroutines adicionales por mensaje.
- **Seguridad**: El token JAMÁS se devuelve en respuestas de API (solo flag `telegram_bot_configured`). Comando único sin acceso libre a SQL. Mensajes de chats no autorizados se ignoran silenciosamente.
- **Almacenamiento**: Sin archivos nuevos en disco. Logs vía `slog` existente.
- **Disponibilidad**: Si el polling falla (red caída), se reintenta con backoff (10s/30s/60s). El server no se ve afectado por errores del bot.
- **iHost**: Dependencia externa única: `github.com/go-telegram/bot` (zero-dependencies, solo stdlib). Sin Node ni procesos extra.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- Bot actual: `~/.hermes/bots/p40la-tg/bot.py` (585 líneas, python-telegram-bot, SSH + sqlite3 remoto, token en `.env`).
- El proyecto usa Go 1.24 con solo 2 dependencias directas (`golang.org/x/crypto`, `modernc.org/sqlite`) — filosofía de mínimas dependencias.
- Librerías Go evaluadas para Telegram Bot API (ver 3.2).
- `go-telegram/bot` está listado en la sección oficial de muestras de la documentación de Telegram (core.telegram.org/bots/samples) y declara **cero dependencias externas** (solo stdlib).

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| `github.com/go-telegram/bot` (v1.25) | Zero-dependencies (solo stdlib), activo (Bot API 10.3), en la lista oficial de Telegram, long polling con `Start(ctx)` | API nueva (menos ejemplos que tgbotapi) | ✅ Seleccionada |
| `gopkg.in/telebot.v3` | Madura, muy usada | Varias deps transitivas, estilo con callbacks pesados | ❌ Rechazada |
| `github.com/go-telegram-bot-api/telegram-bot-api/v5` | La más conocida | Deps extra, API antigua, memoria/goroutines extra | ❌ Rechazada |
| Cliente propio con `net/http` + `getUpdates` | Cero deps | Reescribir getUpdates/sendMessage/setMyCommands, más código a mantener | ❌ Rechazada |
| Mantener bot Python (SSH) | Ya funciona | Proceso aparte, sshpass, .env fuera de settings, idioma distinto al stack | ❌ Rechazada |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: Bot embebido en el server (misma goroutine del proceso)
- **Contexto**: El bot Python corre aparte y accede vía SSH. El server Go ya tiene la DB local abierta y storages de negocio.
- **Decisión**: `TelegramBotService` vive en `internal/services/`, se instancia y arranca en `main.go` con `Start()`/`Stop()`, consultando la DB vía los storages existentes.
- **Consecuencias**: Positivo: cero infra extra, acceso directo a la DB, config desde Settings. Negativo: un fallo del bot no debe tumbar el server (se mitiga con recover/backoff).

**ADR-002**: Token en `system_settings` (no en `.env`)
- **Contexto**: El bot Python lee `BOT_TOKEN` de `.env`. El proyecto ya persiste secretos (SMTP, Voice Monkey) en `system_settings` con patrón público/privado.
- **Decisión**: Claves `telegram_bot_enabled`, `telegram_bot_token` (y P1 `telegram_bot_chat_ids`). El token nunca se expone por API; solo `telegram_bot_configured`.
- **Consecuencias**: El usuario configura el bot desde la UI sin tocar archivos. El `.env` de `~/.hermes` queda obsoleto para este bot.

**ADR-003**: Comando único `/pendientes` en vez de migrar los 9 comandos
- **Contexto**: El usuario pidió específicamente un solo comando por ahora.
- **Decisión**: Solo `/pendientes` (+ `/start` y handler por defecto). El bot Python permanece operativo en paralelo hasta que el usuario decida retirarlo.
- **Consecuencias**: Superficie mínima, fácil de extender con más comandos en specs futuras.

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
[Telegram] --(long polling getUpdates)--> [TelegramBotService (goroutine en server Go)]
     ^                                          |
     | sendMessage                              v
     +--------------------------------- [storage.BillStorage / ServiceStorage] -- SQLite local
                                        
[Frontend Settings "Bot de Telegram"] --(PUT /api/system-settings)--> [SystemSettingsService]
                                                                              |
                                                                              v
                                                              [system_settings table]
```

### 4.2 Componentes

#### 4.2.1 TelegramBotService (`internal/services/telegram_bot.go`)
- **Responsabilidad**: Long polling con la Bot API, registro de handlers (`/pendientes`, `/start`, default), formateo de respuestas, allowlist de chats, arranque/parada/reconfiguración.
- **Interfaz**: `NewTelegramBotService(settings *SystemSettingsService, billStorage *storage.BillStorage, serviceStorage *storage.ServiceStorage, homeStorage *storage.HomeStorage)`, `Start()`, `Stop()`, `NotifyConfigChanged()`.
- **Dependencias**: `SystemSettingsService` (config), storages de bill/service/home (datos).
- **Ubicación**: `internal/services/telegram_bot.go`

#### 4.2.2 SystemSettingsService (extensión)
- **Responsabilidad**: Getters/setters para `telegram_bot_enabled`, `telegram_bot_token`, `telegram_bot_chat_ids`, y `GetTelegramBotConfigPublic` (sin token, con flag `Configured`).
- **Ubicación**: `internal/services/system_settings.go`

#### 4.2.3 SystemSettingsHandlers (extensión)
- **Responsabilidad**: Aceptar `telegram_bot_enabled`, `telegram_bot_token`, `telegram_bot_chat_ids` en `settingsRequest`/`UpdateSystemSettings`; devolver `telegram_bot_enabled`, `telegram_bot_configured` en `GetSystemSettings`. Notificar al `TelegramBotService` tras cada update relevante.
- **Ubicación**: `internal/api/system_settings_handlers.go`

#### 4.2.4 Frontend: `SettingsTelegramBotPage.tsx`
- **Responsabilidad**: Página de settings del bot (patrón `SettingsVoiceMonkeyPage.tsx`): toggle, token (password), chat_ids opcional, estado configurado, botón Reconfigurar.
- **Ruta**: `/settings/telegram-bot`. Se agrega fila en `SettingsPage.tsx` (sección advanced) e i18n en `frontend/public/i18n/{es,en}.json`.
- **Ubicación**: `frontend/src/pages/SettingsTelegramBotPage.tsx`

#### 4.2.5 main.go (extensión)
- **Responsabilidad**: Instanciar el servicio, `Start()` tras los schedulers, `defer Stop()`.
- **Ubicación**: `cmd/server/main.go`

### 4.3 Modelo de datos

No hay migración nueva. Se usan claves en la tabla `system_settings` existente:

```
Clave: telegram_bot_enabled    (string "0"|"1")
Clave: telegram_bot_token      (string, secreto)
Clave: telegram_bot_chat_ids   (string, comma-separated, opcional P1)
```

Consulta para `/pendientes` (definida en la spec, implementada vía storage):

```sql
SELECT s.name AS servicio, h.name AS casa,
       COUNT(b.id) AS cant,
       printf('$%.2f', COALESCE(SUM(b.amount), 0)) AS monto
FROM services s
JOIN homes h ON s.home_id = h.id
LEFT JOIN bills b ON b.service_id = s.id
     AND b.status = 'pending' AND b.deleted_at IS NULL
WHERE s.deleted_at IS NULL AND h.deleted_at IS NULL
GROUP BY s.id
HAVING COUNT(b.id) > 0
ORDER BY SUM(b.amount) DESC
LIMIT 30
```

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/system-settings` (extensión del response existente)

Se agregan los campos:

```json
{
  "telegram_bot_enabled": false,
  "telegram_bot_configured": false
}
```

#### Endpoint: `PUT /api/system-settings` (extensión del request existente)

```json
{
  "telegram_bot_enabled": true,
  "telegram_bot_token": "123456:ABC-DEF...",
  "telegram_bot_chat_ids": "123456789,987654321"
}
```

Reglas:
- `telegram_bot_token` solo se persiste si viene no vacío (patrón `setIfNonEmpty`, no pisa el existente).
- El token NUNCA aparece en ningún response.
- Al recibir cambios de estas claves, el handler llama `telegramBot.NotifyConfigChanged()`.

#### Comando Telegram: `/pendientes`

**Response** (Markdown de Telegram):

```
⏳ *Servicios con facturas pendientes* (2)

⚡ *Claro Internet*
  Casa: Casa Central
  Facturas: 3
  Pendiente: $1,250.00

⚡ *ENATREL*
  Casa: Casa Central
  Facturas: 1
  Pendiente: $350.00
```

Si no hay pendientes: `✅ No hay facturas pendientes.`

### 4.5 Dependencias

- **Internas**: `services.SystemSettingsService`, `storage.BillStorage`, `storage.ServiceStorage`, `storage.HomeStorage`, `api.SystemSettingsHandlers`, `cmd/server/main.go`, frontend (SettingsPage, api/index.ts, i18n).
- **Externas**: `github.com/go-telegram/bot` + `github.com/go-telegram/bot/models` (zero-dependencies). Requiere `go get`.

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Con token + enabled guardados en Settings, el bot responde `/pendientes` con los servicios que tienen facturas pendientes (servicio, casa, cant, monto). Con la DB local.
- [ ] CA-002: Sin token o con enabled off, el bot no inicia polling (verificable en logs: sin errores de conexión a Telegram).
- [ ] CA-003: El token guardado en settings NUNCA aparece en `GET /api/system-settings` (solo `telegram_bot_configured`).
- [ ] CA-004: "Reconfigurar" limpia token + toggles y detiene el polling.
- [ ] CA-005: Guardar/limpiar la config reconfigura el polling en runtime sin reiniciar el server.
- [ ] CA-006: Mensaje no reconocido → respuesta con el comando disponible.
- [ ] CA-007 (P1): Si `telegram_bot_chat_ids` tiene valores, los mensajes de chats fuera de la lista se ignoran silenciosamente.
- [ ] CA-DARK: La página `SettingsTelegramBotPage` verifica inputs en darkmode (tokens `bg-card`/`text-text`), siguiendo SPEC-060.
- [ ] CA-BACK: No aplica (no es página de detalle; es página de settings raíz).

### 5.2 No funcionales

- [ ] CA-NF-001: El server sigue respondiendo `/health` cuando el polling del bot falla (simulado con token inválido).
- [ ] CA-NF-002: Build `go build ./...` y `go vet ./...` sin errores; tests unitarios del formateador y del servicio pasan.

### 5.3 Testing

- **Unit tests**: formateo del mensaje `/pendientes` (con/sin filas, filas con $0), allowlist de chat_ids, `NotifyConfigChanged` (arranca/para polling).
- **Integration tests**: `SystemSettingsService` — set/get de `telegram_bot_enabled`/`telegram_bot_token`/`chat_ids` con DB de test (`/tmp/test-app.db`).
- **E2E**: Configurar token real en la UI local, enviar `/pendientes` desde Telegram, verificar respuesta. (Prueba manual con el usuario.)

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | `go get github.com/go-telegram/bot`; extensiones de `SystemSettingsService` (keys + config pública) + tests | 0.5d | Ninguna |
| 2 | `TelegramBotService`: polling, handlers `/pendientes`/`/start`/default, allowlist, reconfiguración + tests | 1d | Fase 1 |
| 3 | Handlers API (request/response + `NotifyConfigChanged`) + integración en `main.go` | 0.5d | Fase 2 |
| 4 | Frontend: `SettingsTelegramBotPage.tsx`, fila en SettingsPage, api/index.ts, i18n es/en, build + verificación darkmode | 0.5d | Fase 3 |
| 5 | Pruebas locales + build Docker multi-arch + validación con el usuario | 0.5d | Fase 4 |

### 6.2 Milestones

1. **MVP**: `/pendientes` funcionando contra la DB local con token configurado desde Settings.
2. **V1.0**: Allowlist de chats + reconfiguración en runtime robusta.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| Fallo del polling (token inválido, red caída) afecta al server | Media | Medio | Bot corre en goroutine con recover; errores solo logueados; backoff de reintentos |
| El bot responde a desconocidos (datos personales) | Media | Alto | Allowlist de chat_ids (P1); documentar que vacío = abierto (comportamiento heredado) |
| Token visible en logs/UI | Baja | Alto | Nunca se loguea; nunca se devuelve por API; input type=password |
| Librería `go-telegram/bot` nueva para el proyecto | Baja | Bajo | Zero-deps, en lista oficial de Telegram; API pequeña para un solo comando |
| DB con muchos pendientes → mensaje gigante | Baja | Medio | `LIMIT 30` en la query |

## 8. Notas y Referencias

- Bot Python fuente: `~/.hermes/bots/p40la-tg/bot.py`
- Librería: `github.com/go-telegram/bot` — https://github.com/go-telegram/bot (zero-dependencies)
- Bot API: https://core.telegram.org/bots/api
- Patrón de config con secretos: SPEC-033 (Voice Monkey), SPEC-034 (Reconfigurar)
- Patrón de inputs/darkmode: SPEC-060

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-16 | p40la-ihost-team | Creación inicial de la especificación |
| 2026-09-16 | p40la-ihost-team | Implementación completa y validación local por el usuario. Release: commit `<commit-hash>` en main |