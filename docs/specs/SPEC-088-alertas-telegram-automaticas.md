---
title: "Alertas automáticas por Telegram (canal push, espejo de los mails)"
id: "SPEC-088"
status: "released"
author: "opencode"
created: "2026-09-19"
updated: "2026-09-19"
github_issue: 91
---

# Alertas automáticas por Telegram (canal push, espejo de los mails)

**ID**: SPEC-088  
**Estado**: released  
**Autor**: opencode  
**Creado**: 2026-09-19  
**Actualizado**: 2026-09-19

---

## 1. Resumen Ejecutivo

El bot de Telegram (SPEC-079 a SPEC-086) hoy es **solo reactivo (pull)**: responde a los comandos `/servicios_pendientes` y `/deudas_pendientes` cuando el usuario pregunta, pero jamás envía mensajes por iniciativa propia. En cambio, las alertas por email y por voz (Alexa) son **push**: los schedulers (`BillSummaryScheduler`, `DebtDueScheduler`, `AlertScheduler`, `BillingScheduler`, `PensionNotificationService`) se ejecutan en horarios programados y despachan alertas por los canales habilitados por alerta (sistema multicanal de SPEC-032/033, tabla `alerts` con flags `mail_enabled` / `voice_enabled`).

Esta spec agrega **Telegram como tercer canal de alerta (push)**: cuando una alerta tiene el canal telegram habilitado y el bot está configurado, los schedulers existentes enviarán mensajes proactivos a los chat_ids autorizados, con el **mismo formato de texto** que ya producen los comandos del bot (reutilizando `formatServiciosPendientes` / `formatDeudasPendientes` y sus variantes). El usuario recibirá en Telegram los resúmenes diarios, las cuotas que vencen hoy, los seguros de autos vencidos y los avisos de facturas/pensión sin tener que preguntar.

No se requiere infraestructura nueva: el bot ya tiene token, allowlist de chat_ids y conexión con Telegram por long polling. El impacto en el iHost es mínimo: un `SendAlerts` que reutiliza formatters existentes y envía mensajes con la misma librería `go-telegram/bot` ya integrada.

## 2. Requerimientos

### 2.1 Requerimientos Funcionales (P0 - Obligatorios)

1. **REQ-001**: Agregar el canal `telegram` al catálogo de alertas: nueva columna `telegram_enabled` en la tabla `alerts` (migración `0031`), constante `AlertChannelTelegram = "telegram"` en `internal/models/alert.go`, y soporte en `AlertService.IsEnabled` / `SetFlags` y `AlertStorage`.
2. **REQ-002**: `TelegramBotService.SendAlerts(ctx, texts []string)`: envía mensajes **proactivos** a TODOS los `chat_ids` autorizados de la config (no depende de un `update` de Telegram). Si no hay chat_ids configurados o el bot no está habilitado/configurado, no envía nada y loguea a nivel debug/warn.
3. **REQ-003**: Helper `dispatchTelegram(ctx, alerts, bot, key, texts)` en `internal/services/alert_dispatch.go` (espejo de `dispatchVoice`): valida canal habilitado para la alerta + bot habilitado (`telegram_bot_enabled`) + configurado (token) + al menos un chat_id. **Si el bot no está habilitado en settings, JAMÁS se intenta enviar**: el gate corta antes de crear cualquier instancia de bot o llamar a la API de Telegram (ni siquiera un intento fallido). No bloqueante: un error se loguea sin interrumpir el scheduler.
4. **REQ-004**: Integrar el canal telegram en los schedulers que hoy envían email/voz, reutilizando el contenido de texto de los comandos del bot:
   - `BillSummaryScheduler` (AlertKeyBillSummary): resumen diario de facturas pendientes → `formatServiciosPendientes`.
   - `DebtDueScheduler` (AlertKeyDebtDue): cuotas que vencen hoy → `formatDeudasPendientes` (o un formatter específico de "vence hoy" según decisión de implementación, ver ADR-002).
   - `AlertScheduler` (AlertKeyInsurance): seguros de autos vencidos/sin seguro → formatter nuevo de texto plano equivalente a `renderAlertsContent`.
   - `BillingScheduler` (AlertKeyBillCreated): aviso de facturas generadas automáticamente.
   - `PensionNotificationService` (AlertKeyPension*): avisos de eventos de pensión (los que hoy usan `send` con title/content).
5. **REQ-005**: Frontend `SettingsAlertsPage`: toggle "Telegram" por alerta (además de Mail y Alexa). Gating estricto: **si el bot no está habilitado en settings, el toggle Telegram no se puede activar** (aparece deshabilitado con hint, o directamente oculto — decisión de UI en implementación). Si el bot está habilitado pero sin configurar (token/chat_ids), también deshabilitado con hint (espejo del gating actual de mail/alexa, SPEC-037).
6. **REQ-006**: API de alertas: `GET /api/alerts` devuelve `telegram_enabled`; `PUT /api/alerts/{key}` acepta `telegram_enabled`. **Validación estricta en backend**: si el bot no está habilitado (`telegram_bot_enabled=0`), sin token o sin chat_ids, el PUT con `telegram_enabled=true` se rechaza con error (`422`), igual que las validaciones actuales de mail/voz. Nunca se persiste `telegram_enabled=1` con el bot inactivo.

### 2.2 Requerimientos Funcionales (P1 - Importantes)

1. **REQ-007**: Tests unitarios: `dispatchTelegram` (gates), `SendAlerts` (envío a múltiples chat_ids, sin chat_ids, bot deshabilitado), formatters nuevos, y wiring de schedulers con bot mock.
2. **REQ-008**: El mensaje de bienvenida de `/start` menciona que el bot también envía alertas automáticas configuradas desde P40LA.

### 2.3 Requerimientos Funcionales (P2 - Deseables)

1. **REQ-009**: Comando `/stop_alerts` (o equivalente) para pausar alertas push en un chat sin quitar la allowlist.
2. **REQ-010**: Registrar en `lastChatID`-style una key `last_<alerta>_telegram_check` reutilizando el dedupe diario existente por scheduler (los schedulers ya tienen `lastCheckKey`; no duplicar).

### 2.4 Requerimientos No Funcionales

- **Rendimiento**: `SendAlerts` envía secuencialmente (patrón `sendMany` existente); N chat_ids × M mensajes con timeout por envío. No bloquear el scheduler: envío en la misma goroutine con log de errores (patrón dispatchVoice actual).
- **Seguridad**: los mensajes solo se envían a chat_ids de la allowlist (`telegram_bot_chat_ids`). El token nunca se expone en la API (ya es así).
- **Almacenamiento**: una columna nueva en `alerts`; sin tablas nuevas.
- **Disponibilidad**: si Telegram falla, el scheduler continúa (dispatch no bloqueante, mismo patrón que mail/voz).
- **iHost**: cero dependencias nuevas; reutiliza `go-telegram/bot` ya compilado en el binario.

## 3. Investigación y Decisiones Técnicas

### 3.1 Contexto investigado

- `internal/services/telegram_bot.go`: bot con long polling, handlers de comandos, `sendMany` secuencial, formatters `formatServiciosPendientes` / `formatDeudasPendientes` con chunking por 4096 chars, allowlist vía `isAuthorized`.
- `internal/services/alert_dispatch.go`: helpers `alertMailEnabled` / `alertVoiceEnabled` / `dispatchVoice` — patrón a replicar para telegram.
- `internal/services/alert_scheduler.go`, `bill_summary_scheduler.go`, `debt_due_scheduler.go`, `billing_scheduler.go`, `pension_notification.go`: schedulers con `lastCheckKey` de dedupe diario y hora configurable (`alert_check_hour`).
- `internal/models/alert.go`: `AlertChannelMail` / `AlertChannelVoice`; el comentario del modelo ya anticipa "mail / voz / futuro".
- `migrations/0013_create_alerts.up.sql`: tabla `alerts` con `mail_enabled`, `voice_enabled`; la última migración es `0030`.
- `frontend/src/pages/SettingsAlertsPage.tsx`: toggles por alerta con gating; `frontend/src/pages/SettingsTelegramBotPage.tsx`: config del bot.

### 3.2 Opciones evaluadas

| Opción | Pros | Contras | Decisión |
|--------|------|---------|----------|
| Telegram como canal en el sistema multicanal existente (tabla `alerts` + dispatch + schedulers) | Consistente con mail/voz; un toggle por alerta; los schedulers ya tienen la lógica de hora/dedupe | Requiere migración y tocar todos los schedulers | ✅ Seleccionada |
| Servicio separado tipo "TelegramAlertPusher" que consulta pending y envía | Aislado, sin tocar schedulers | Duplica hora/dedupe/condiciones; dos fuentes de verdad del "qué enviar" | ❌ Rechazada |
| Solo agregar comando nuevo (ej: `/alertas_del_dia`) | Cero cambios en schedulers | No es automático (sigue siendo pull); no cumple el requerimiento | ❌ Rechazada |
| Master toggle `telegram_alerts_enabled` nuevo | Control global explícito | Config extra redundante: `telegram_bot_enabled` + chat_ids ya son el gate natural (espejo de como email_alerts_enabled gatea SMTP) | ❌ Rechazada (ver ADR-001) |

### 3.3 Decisiones arquitectónicas (ADRs)

**ADR-001**: El gate del canal telegram es la config existente del bot, sin master toggle nuevo.
- **Contexto**: El canal mail tiene `email_alerts_enabled` porque SMTP es una config separada de los toggles. El bot de Telegram ya tiene `telegram_bot_enabled` + `telegram_bot_token` + `telegram_bot_chat_ids` como config propia.
- **Decisión**: `dispatchTelegram` activa el envío solo si: alerta con `telegram_enabled=1` ∧ bot enabled ∧ token presente ∧ ≥1 chat_id. No se agrega `telegram_alerts_enabled`.
- **Consecuencias**: Menos configuración; apagar el bot (toggle general) apaga también las alertas push, lo cual es el comportamiento esperado.

**ADR-002**: Reutilizar los formatters de texto de los comandos para las alertas push.
- **Contexto**: `formatServiciosPendientes` / `formatDeudasPendientes` ya producen el formato aprobado por el usuario (SPEC-082/083/084/086): semáforo, fechas legibles, separador, totales por moneda, chunking.
- **Decisión**: `BillSummaryScheduler` envía `formatServiciosPendientes(...)`; `DebtDueScheduler` envía `formatDeudasPendientes(...)` (con el filtro que corresponda al evento: para "vence hoy" se puede filtrar por `dueInRange`/fecha exacta). Para seguros (AlertScheduler) y eventos de facturas/pensión se crean formatters de texto plano nuevos en `telegram_alerts.go`, espejo de los HTML actuales.
- **Consecuencias**: Cero duplicación de formato; el push se ve idéntico al pull. Las config de `separator_length` y `show_months` aplican también al push.

**ADR-003**: `SendAlerts` envía a TODOS los chat_ids de la allowlist, no a `lastChatID`.
- **Contexto**: Existe `lastChatID` (último chat que interactuó). Usarlo haría las alertas dependientes de interacción previa y fallaría si el usuario nunca escribió al bot.
- **Decisión**: Destino = `cfg.ChatIDs` completo. Si la lista está vacía, no se envía nada (log warn). Esto mantiene la regla de autorización actual (allowlist vacía = cualquiera, pero en modo push sin destinatarios explícitos no tiene sentido enviar).
- **Consecuencias**: Las alertas llegan a todos los chats autorizados; consistente con "destinatarios" del mail (`alert_emails`).

## 4. Diseño Técnico

### 4.1 Diagrama de arquitectura

```
                    ┌────────────────────────────────────────────┐
                    │              Schedulers (diarios)          │
                    │  BillSummaryScheduler / DebtDueScheduler   │
                    │  AlertScheduler / BillingScheduler         │
                    │  PensionNotificationService                │
                    └───────────────┬────────────────────────────┘
                                    │ dispatchTelegram(key, texts)
                                    ▼
                    ┌────────────────────────────────────────────┐
                    │          TelegramBotService.SendAlerts     │
                    │  → bot.SendMessage a cada ChatID (allowlist)│
                    └───────────────┬────────────────────────────┘
                                    ▼
                              Telegram API
                    (mismo long-polling / go-telegram/bot)

    Frontend SettingsAlertsPage ──PUT /api/alerts/{key}──▶ alerts table
                    (telegram_enabled)
```

### 4.2 Componentes

#### 4.2.1 `internal/models/alert.go`
- **Responsabilidad**: definir el canal nuevo.
- **Cambios**: `AlertChannelTelegram AlertChannel = "telegram"`; campo `TelegramEnabled bool` en `Alert`.

#### 4.2.2 `internal/storage/alert_storage.go`
- **Responsabilidad**: persistencia de flags.
- **Cambios**: `SetFlags` acepta `telegramEnabled *bool`; `List`/`GetByKey` leen `telegram_enabled`.

#### 4.2.3 `internal/services/alert_service.go`
- **Responsabilidad**: catálogo + gates por canal.
- **Cambios**: `SetFlags(key, mail, voice, telegram)`; `IsEnabled` case `AlertChannelTelegram`.

#### 4.2.4 `internal/services/alert_dispatch.go`
- **Responsabilidad**: dispatch multicanal compartido.
- **Cambios**: `alertTelegramEnabled(ctx, alerts, key)` + `dispatchTelegram(ctx, alerts, bot *TelegramBotService, key string, texts []string)`. Valida gates del ADR-001 y llama `bot.SendAlerts`.

#### 4.2.5 `internal/services/telegram_bot.go` (+ `telegram_alerts.go` nuevo)
- **Responsabilidad**: envío push.
- **Cambios**: método `SendAlerts(ctx, texts []string)` (crea bot con el token de config, envía a cada chat_id de la allowlist; si el polling ya está activo se puede reutilizar la instancia del bot — ver implementación). Formatters nuevos de texto plano para seguros/facturas/pensión.

#### 4.2.6 Schedulers (5 archivos)
- **Responsabilidad**: decidir cuándo y qué enviar.
- **Cambios**: recibir `telegramBot *TelegramBotService` en el constructor (nil-safe), y tras el envío mail/voz actual agregar `dispatchTelegram` con los textos del evento.

#### 4.2.7 `internal/api/alerts_handlers.go`
- **Responsabilidad**: API de toggles.
- **Cambios**: request/response con `telegram_enabled`; validación espejo de mail/voz.

#### 4.2.8 `frontend/src/pages/SettingsAlertsPage.tsx` + i18n
- **Responsabilidad**: UI de toggles.
- **Cambios**: tercer toggle por alerta; estado deshabilitado con hint si el bot no está activo/configurado/sin chat_ids. Claves i18n en `frontend/public/i18n/{es,en}.json` (`settings.alerts.telegram*`).

### 4.3 Modelo de datos

Migración `migrations/0031_add_alerts_telegram_enabled.up.sql`:

```sql
ALTER TABLE alerts ADD COLUMN telegram_enabled INTEGER NOT NULL DEFAULT 0;
```

`0031...down.sql`:

```sql
ALTER TABLE alerts DROP COLUMN telegram_enabled;
```

(Si SQLite de la versión del server no soporta `DROP COLUMN`, el down puede ser no-op documentado — validar en implementación; SQLite ≥3.35 soporta DROP COLUMN.)

### 4.4 APIs / Contratos

#### Endpoint: `GET /api/alerts`

**Response 200** (item del array):
```json
{
  "id": 1,
  "key": "bill_summary",
  "title": "Resumen diario de facturas",
  "description": "...",
  "mail_enabled": true,
  "voice_enabled": false,
  "telegram_enabled": true,
  "speech": "..."
}
```

#### Endpoint: `PUT /api/alerts/{key}`

**Request**:
```json
{
  "mail_enabled": true,
  "voice_enabled": false,
  "telegram_enabled": true
}
```

**Response 200**: alerta actualizada con los 3 flags.
**Response Error**: `422` con mensaje cuando el canal telegram no se puede activar (bot no habilitado / sin token / sin chat_ids), espejo de las validaciones actuales.

### 4.5 Dependencias

- **Internas**: `AlertService`, `AlertStorage`, `TelegramBotService`, 5 schedulers, `alerts_handlers.go`, `SettingsAlertsPage.tsx`, i18n.
- **Externas**: ninguna nueva (`go-telegram/bot` ya integrado).

## 5. Criterios de Aceptación

### 5.1 Funcionales

- [ ] CA-001: Dado el bot configurado y una alerta con `telegram_enabled=1`, cuando el scheduler correspondiente corre a la hora configurada, entonces se envía el mensaje a TODOS los chat_ids de la allowlist con el formato de los comandos.
- [ ] CA-002: Dado el bot deshabilitado en settings (o sin token o sin chat_ids), cuando un scheduler corre, entonces NO se envía nada por Telegram y NO se intenta siquiera contactar la API de Telegram (el gate corta antes; debug log); el resto de los canales sigue funcionando.
- [ ] CA-003: Dado `GET /api/alerts`, entonces cada alerta expone `telegram_enabled`; dado `PUT /api/alerts/{key}` con `telegram_enabled=true` y el bot deshabilitado/sin configurar, entonces se rechaza con error `422` y no se persiste el flag.
- [ ] CA-004: Dado el gating en `SettingsAlertsPage`, cuando el bot no está habilitado en settings, entonces el toggle Telegram no se puede activar (deshabilitado con hint u oculto); lo mismo si está habilitado pero sin token/chat_ids.
- [ ] CA-005: Dado el resumen diario habilitado por telegram, cuando hay facturas pendientes, entonces el texto del push es idéntico al de `/servicios_pendientes` (mismo formatter, separador y show_months).
- [ ] CA-006: Dado un envío de alerta que falla (red/API Telegram), entonces el scheduler continúa sin crashear y loguea el error.
- [ ] CA-DARK: Los toggles/hints nuevos de `SettingsAlertsPage` usan tokens del tema y se verificó legibilidad en darkmode.

### 5.2 No funcionales

- [ ] CA-NF-001: Cero dependencias nuevas; el binario no crece con librerías nuevas (solo lógica propia).
- [ ] CA-NF-002: Los mensajes push respetan el límite de 4096 chars (chunking existente de `maxBillsPerMessage`/`maxInstallmentsPerMessage`).

### 5.3 Testing

- **Unit tests**: `dispatchTelegram` (gates: canal off, bot off, sin token, sin chat_ids, ok), `SendAlerts` (múltiples chat_ids, lista vacía), formatters nuevos (seguros, facturas, pensión), `SetFlags`/`IsEnabled` con canal telegram.
- **Integration tests**: scheduler con bot mock que verifica textos y destinatarios; API PUT/GET con telegram_enabled.
- **E2E/manual**: configurar bot + activar canal en una alerta → forzar `CheckNow()` de un scheduler → verificar llegada a Telegram. Verificar gating en la UI.
- **iHost**: verificar que el bot sigue respondiendo comandos y además envía push sin aumento de memoria relevante.

## 6. Plan de Implementación

### 6.1 Fases

| Fase | Descripción | Estimación | Dependencias |
|------|-------------|------------|--------------|
| 1 | Migración `0031` + modelos/storage/service (canal telegram en catálogo y API) | 0.5 día | Ninguna |
| 2 | `SendAlerts` + `dispatchTelegram` + formatters de texto plano nuevos | 0.5 día | Fase 1 |
| 3 | Wiring de los 5 schedulers con dispatchTelegram | 0.5 día | Fase 2 |
| 4 | Frontend: toggle Telegram en SettingsAlertsPage + i18n + gating | 0.5 día | Fase 1 |
| 5 | Tests unitarios + integración + validación manual local (server corriendo, prueba real de push) | 0.5 día | Fases 1-4 |

### 6.2 Milestones

1. **MVP**: Canal telegram para `bill_summary` y `debt_due` (los dos comandos que ya tienen formato).
2. **V1.0**: Todos los schedulers (seguros, facturas, pensión) + UI completa + tests.

## 7. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| `SendAlerts` crea una instancia de bot separada del polling y los rate limits de Telegram | Media | Medio | Enviar con el bot del polling activo (mantener referencia) o tolerancia de errores por mensaje (ya es el patrón `sendMany`); backoff del supervisor existente |
| Alertas push duplicadas con el mismo contenido que el mail | Media | Bajo | Es el comportamiento pedido (canales independientes por alerta); el usuario decide qué canales activa |
| `DROP COLUMN` no soportado en la versión de SQLite del iHost | Baja | Bajo | Validar versión en implementación; down no-op documentado |
| Spam al usuario (push + pull idénticos) | Media | Bajo | La alerta queda off por defecto (`DEFAULT 0`); el usuario la activa explícitamente |

## 8. Notas y Referencias

- SPEC-032/033: sistema multicanal de alertas (mail/voz).
- SPEC-037: gating de toggles por canal.
- SPEC-079 a SPEC-086: bot de Telegram (pull), formatters y config.
- `frontend/public/i18n/es.json` / `en.json`: fuente de verdad del i18n (editar ahí, nunca en `public/i18n/`).

## 9. Historial de Cambios

| Fecha | Autor | Descripción |
|-------|-------|-------------|
| 2026-09-19 | opencode | Creación inicial de la especificación |
| 2026-09-19 | opencode | Implementación (canal telegram en alertas, SendAlerts, dispatch, wiring de 5 schedulers, UI, i18n, tests) y validación manual del usuario |
| 2026-09-19 | opencode | Release: merge a main, issue #91 cerrado con label spec/released |